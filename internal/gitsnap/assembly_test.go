package gitsnap

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
)

func assemblyOptions(t *testing.T, root string) AssemblyOptions {
	options := AssemblyOptions{ScratchRoot: t.TempDir(), WorkspaceGroupID: "01900000-0000-7000-8000-000000000001", CreatedByHostID: "01900000-0000-7000-8000-000000000002", CreatedAt: "2026-09-08T00:00:00.000Z", Members: []WorkspaceSource{{WorkspaceID: "01900000-0000-7000-8000-000000000003", GroupRelativePath: "repo", Directory: root, MaterializationPolicy: "shared_checkout", Content: contentOptions(t), ProjectConfigPaths: map[string][]string{".": {"AGENTS.md"}}}}}
	setAssemblyRecord(t, &options)
	return options
}
func assemble(t *testing.T, runner AssemblyRunner, options AssemblyOptions) *ProvisionalAssembly {
	t.Helper()
	result, err := AssembleProvisional(context.Background(), runner, options)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func wireRoot(t *testing.T, value *ProvisionalAssembly) map[string]any {
	t.Helper()
	var root map[string]any
	if err := json.Unmarshal(value.Manifests[value.RootID], &root); err != nil {
		t.Fatal(err)
	}
	return root
}
func firstMember(t *testing.T, value *ProvisionalAssembly) map[string]any {
	return wireRoot(t, value)["workspace_snapshot"].(map[string]any)["members"].([]any)[0].(map[string]any)
}

func TestAssemblyPackIndexAndBytes(t *testing.T) {
	t.Parallel()
	root := initLiveRepo(t)
	writeContent(t, root, "README.md", []byte("staged\n"))
	gitRun(t, root, "add", "README.md")
	writeContent(t, root, "README.md", []byte("working\n"))
	writeContent(t, root, "notes.txt", []byte("notes\n"))
	options := assemblyOptions(t, root)
	indexBefore, err := os.ReadFile(filepath.Join(root, ".git/index"))
	if err != nil {
		t.Fatal(err)
	}
	result := assemble(t, ExecGitRunner{}, options)
	member := firstMember(t, result)
	pack := member["object_pack"].(map[string]any)
	inventory := storedBytes(t, options.Members[0].Content, pack["inventory_blob_id"].(string))
	if len(bytes.Split(bytes.TrimSuffix(inventory, []byte("\n")), []byte("\n"))) != int(pack["object_count"].(float64)) {
		t.Fatal("inventory count")
	}
	rawIndex := storedBytes(t, options.Members[0].Content, member["index"].(map[string]any)["blob_id"].(string))
	if !bytes.Equal(indexBefore, rawIndex) {
		t.Fatal("raw index not preserved")
	}
	after, _ := os.ReadFile(filepath.Join(root, ".git/index"))
	if !bytes.Equal(after, indexBefore) {
		t.Fatal("source index changed")
	}
	var tree map[string]any
	if err := json.Unmarshal(result.Manifests[member["working_tree_manifest_id"].(string)], &tree); err != nil {
		t.Fatal(err)
	}
	for _, item := range tree["entries"].([]any) {
		entry := item.(map[string]any)
		if entry["path"] == "README.md" {
			if !bytes.Equal(storedBytes(t, options.Members[0].Content, entry["blob_id"].(string)), []byte("working\n")) {
				t.Fatal("staged bytes replaced working bytes")
			}
		}
	}
	again := assemble(t, ExecGitRunner{}, options)
	if result.RootID != again.RootID {
		t.Fatal("repeat assembly not idempotent")
	}
}

func TestAssemblyRecursivePacks(t *testing.T) {
	t.Parallel()
	root, child := submoduleRepo(t)
	writeContent(t, child, "unique", []byte("child only\n"))
	gitRun(t, child, "add", "unique")
	gitRun(t, child, "commit", "-qm", "child unique")
	options := assemblyOptions(t, root)
	result := assemble(t, ExecGitRunner{}, options)
	member := firstMember(t, result)
	sub := member["submodules"].([]any)[0].(map[string]any)
	parentPack := member["object_pack"].(map[string]any)
	childPack := sub["object_pack"].(map[string]any)
	parentInventory := storedBytes(t, options.Members[0].Content, parentPack["inventory_blob_id"].(string))
	childInventory := storedBytes(t, options.Members[0].Content, childPack["inventory_blob_id"].(string))
	childOID := strings.TrimSpace(gitRun(t, child, "rev-parse", "HEAD"))
	if bytes.Contains(parentInventory, []byte(childOID)) {
		t.Fatal("child commit leaked into parent pack")
	}
	if !bytes.Contains(childInventory, []byte(childOID+" commit ")) {
		t.Fatal("child HEAD missing")
	}
	if len(result.Manifests) != 3 {
		t.Fatal("root/parent/child closure missing")
	}
}

type assemblyIntercept struct {
	ExecGitRunner
	input    func(context.Context, string, []byte, []string) (GitResult, error)
	isolated func(context.Context, string, []byte, []string) (GitResult, error)
	run      func(context.Context, string, []string) (GitResult, error)
}

func (r assemblyIntercept) Run(ctx context.Context, root string, args ...string) (GitResult, error) {
	if r.run != nil {
		return r.run(ctx, root, args)
	}
	return r.ExecGitRunner.Run(ctx, root, args...)
}
func (r assemblyIntercept) RunInput(ctx context.Context, root string, input []byte, args ...string) (GitResult, error) {
	if r.input != nil {
		return r.input(ctx, root, input, args)
	}
	return r.ExecGitRunner.RunInput(ctx, root, input, args...)
}
func (r assemblyIntercept) RunIsolated(ctx context.Context, root string, input []byte, args ...string) (GitResult, error) {
	if r.isolated != nil {
		return r.isolated(ctx, root, input, args)
	}
	return r.ExecGitRunner.RunIsolated(ctx, root, input, args...)
}

func TestAssemblySourceChangesAndRetry(t *testing.T) {
	t.Parallel()
	for _, shape := range []string{"file", "new-file", "mode", "index", "head", "config", "child"} {
		t.Run(shape, func(t *testing.T) {
			root := initLiveRepo(t)
			child := ""
			if shape == "child" {
				root, child = submoduleRepo(t)
			}
			options := assemblyOptions(t, root)
			mutated := false
			runner := assemblyIntercept{input: func(ctx context.Context, dir string, input []byte, args []string) (GitResult, error) {
				result, err := (ExecGitRunner{}).RunInput(ctx, dir, input, args...)
				if !mutated {
					mutated = true
					switch shape {
					case "file":
						writeContent(t, root, "AGENTS.md", []byte("other\n"))
					case "new-file":
						writeContent(t, root, "new-file", []byte("new\n"))
					case "mode":
						if err := os.Chmod(filepath.Join(root, "AGENTS.md"), 0755); err != nil {
							t.Fatal(err)
						}
					case "index":
						gitRun(t, root, "update-index", "--assume-unchanged", "AGENTS.md")
					case "head":
						gitRun(t, root, "checkout", "--detach", "-q")
					case "config":
						gitRun(t, root, "config", "core.filemode", "false")
					case "child":
						writeContent(t, child, "README.md", []byte("child changed\n"))
					}
				}
				return result, err
			}}
			result, err := AssembleProvisional(context.Background(), runner, options)
			if err == nil || result != nil || !mutated {
				t.Fatalf("source change admitted: %v", err)
			}
			assemble(t, ExecGitRunner{}, options)
		})
	}
}

func TestAssemblyFailedReadsAndCorruptObjects(t *testing.T) {
	t.Parallel()
	for _, shape := range []string{"warning", "pack-byte", "pack-count", "inventory-count", "inventory-size", "inventory-byte-size", "inventory-duplicate", "index-mismatch"} {
		t.Run(shape, func(t *testing.T) {
			root := initLiveRepo(t)
			options := assemblyOptions(t, root)
			injected := false
			runner := assemblyIntercept{}
			runner.input = func(ctx context.Context, dir string, input []byte, args []string) (GitResult, error) {
				result, err := (ExecGitRunner{}).RunInput(ctx, dir, input, args...)
				switch shape {
				case "warning":
					result.Stderr = []byte("partial read")
					injected = true
				case "pack-byte":
					result.Stdout[len(result.Stdout)-1] ^= 1
					injected = true
				case "pack-count":
					result.Stdout[11]++
					injected = true
				}
				return result, err
			}
			runner.isolated = func(ctx context.Context, dir string, input []byte, args []string) (GitResult, error) {
				result, err := (ExecGitRunner{}).RunIsolated(ctx, dir, input, args...)
				if args[0] == "cat-file" && len(args) > 1 && args[1] == "--batch-all-objects" {
					lines := strings.Split(string(result.Stdout), "\n")
					switch shape {
					case "inventory-count":
						result.Stdout = []byte(strings.Join(lines[1:], "\n"))
						injected = true
					case "inventory-size":
						fields := strings.Fields(lines[0])
						fields[2] = "01"
						lines[0] = strings.Join(fields, " ")
						result.Stdout = []byte(strings.Join(lines, "\n"))
						injected = true
					case "inventory-byte-size":
						fields := strings.Fields(lines[0])
						size, _ := strconv.Atoi(fields[2])
						fields[2] = strconv.Itoa(size + 1)
						lines[0] = strings.Join(fields, " ")
						result.Stdout = []byte(strings.Join(lines, "\n"))
						injected = true
					case "inventory-duplicate":
						lines[1] = lines[0]
						result.Stdout = []byte(strings.Join(lines, "\n"))
						injected = true
					}
				}
				if args[0] == "ls-files" && shape == "index-mismatch" {
					result.Stdout = bytes.Replace(result.Stdout, []byte("AGENTS.md"), []byte("BGENTS.md"), 1)
					injected = true
				}
				return result, err
			}
			value, err := AssembleProvisional(context.Background(), runner, options)
			if err == nil || value != nil || !injected {
				t.Fatalf("bad assembly admitted/injection missed: %v %t", err, injected)
			}
		})
	}
}

func resealRoot(t *testing.T, value *ProvisionalAssembly, root map[string]any) {
	t.Helper()
	delete(value.Manifests, value.RootID)
	raw, _ := json.Marshal(root)
	id, _, err := canonicaljson.CalculateObjectIdentity(raw)
	if err != nil {
		t.Fatal(err)
	}
	root["manifest_id"] = id.String()
	raw, _ = json.Marshal(root)
	value.RootID = id.String()
	value.Manifests[value.RootID] = raw
}
func TestValidateProvisionalClosureAndDescriptors(t *testing.T) {
	t.Parallel()
	root := initLiveRepo(t)
	options := assemblyOptions(t, root)
	for _, shape := range []string{"child-missing", "descriptor-missing", "descriptor-wrong-size", "entry-size", "unbound-tree", "cwd-missing", "config-missing", "tree-binding"} {
		t.Run(shape, func(t *testing.T) {
			value := assemble(t, ExecGitRunner{}, options)
			member := firstMember(t, value)
			switch shape {
			case "child-missing":
				delete(value.Manifests, member["working_tree_manifest_id"].(string))
			case "descriptor-missing":
				for id := range value.Descriptors {
					delete(value.Descriptors, id)
					break
				}
			case "unbound-tree":
				treeID := member["working_tree_manifest_id"].(string)
				var tree map[string]any
				if err := json.Unmarshal(value.Manifests[treeID], &tree); err != nil {
					t.Fatal(err)
				}
				tree["entries"] = append(tree["entries"].([]any), map[string]any{"path": "zz-extra", "type": "directory", "mode": float64(0755)})
				raw, _ := json.Marshal(tree)
				id, _, err := canonicaljson.CalculateObjectIdentity(raw)
				if err != nil {
					t.Fatal(err)
				}
				tree["manifest_id"] = id.String()
				raw, _ = json.Marshal(tree)
				value.Manifests[id.String()] = raw
				object := wireRoot(t, value)
				children := []string{treeID, id.String()}
				sort.Strings(children)
				object["child_manifest_ids"] = children
				resealRoot(t, value, object)
			case "entry-size":
				treeID := member["working_tree_manifest_id"].(string)
				var tree map[string]any
				if err := json.Unmarshal(value.Manifests[treeID], &tree); err != nil {
					t.Fatal(err)
				}
				for _, item := range tree["entries"].([]any) {
					entry := item.(map[string]any)
					if entry["path"] == "AGENTS.md" {
						entry["size"] = float64(7)
					}
				}
				raw, _ := json.Marshal(tree)
				id, _, err := canonicaljson.CalculateObjectIdentity(raw)
				if err != nil {
					t.Fatal(err)
				}
				tree["manifest_id"] = id.String()
				raw, _ = json.Marshal(tree)
				delete(value.Manifests, treeID)
				value.Manifests[id.String()] = raw
				object := wireRoot(t, value)
				object["child_manifest_ids"] = []string{id.String()}
				object["workspace_snapshot"].(map[string]any)["members"].([]any)[0].(map[string]any)["working_tree_manifest_id"] = id.String()
				resealRoot(t, value, object)
			case "descriptor-wrong-size":
				for id, d := range value.Descriptors {
					d.Size++
					value.Descriptors[id] = d
					break
				}
			default:
				object := wireRoot(t, value)
				m := object["workspace_snapshot"].(map[string]any)["members"].([]any)[0].(map[string]any)
				switch shape {
				case "cwd-missing":
					m["repo_relative_cwd"] = "missing"
				case "config-missing":
					m["agent_project_config_paths"] = []string{"missing"}
				case "tree-binding":
					m["workspace_id"] = "01900000-0000-7000-8000-000000000009"
				}
				resealRoot(t, value, object)
			}
			if err := ValidateProvisional(value); err == nil {
				t.Fatal("invalid closure admitted")
			}
		})
	}
}

func TestAssemblyNormativeCorpus(t *testing.T) {
	t.Parallel()
	root, payloads, _, _, parentOID, childOID := normativeGitWorkspace(t)
	options := assemblyOptions(t, filepath.Join(root, "src"))
	options.Members[0].ProjectConfigPaths["vendor/lib"] = []string{"README.md"}
	value := assemble(t, ExecGitRunner{}, options)
	member := firstMember(t, value)
	sub := member["submodules"].([]any)[0].(map[string]any)
	if member["head"].(map[string]any)["oid"] != parentOID || sub["head"].(map[string]any)["oid"] != childOID {
		t.Fatal("pinned HEAD identities differ")
	}
	if member["repo_relative_cwd"] != "src" {
		t.Fatal("cwd lost")
	}
	for _, repo := range []struct {
		prefix string
		wire   map[string]any
	}{{"parent", member}, {"child", sub}} {
		pack := repo.wire["object_pack"].(map[string]any)
		inventory := storedBytes(t, options.Members[0].Content, pack["inventory_blob_id"].(string))
		if !bytes.Equal(inventory, payloads[repo.prefix+"_inventory"]) {
			t.Fatalf("%s exact inventory differs: %s", repo.prefix, inventory)
		}
		index := repo.wire["index"].(map[string]any)
		raw := storedBytes(t, options.Members[0].Content, index["blob_id"].(string))
		if !bytes.Equal(raw, payloads[repo.prefix+"_index"]) {
			t.Fatalf("%s raw index differs", repo.prefix)
		}
		// Import generated bytes into a second independent database, then resolve
		// the expected HEAD and every inventory object there, with no source access.
		scratch := t.TempDir()
		if _, err := isolatedRun(context.Background(), ExecGitRunner{}, scratch, nil, "init", "--bare", "-q", "--template=", "."); err != nil {
			t.Fatal(err)
		}
		if _, err := isolatedRun(context.Background(), ExecGitRunner{}, scratch, storedBytes(t, options.Members[0].Content, pack["blob_id"].(string)), "index-pack", "--stdin", "--strict"); err != nil {
			t.Fatal(err)
		}
		got, err := isolatedRun(context.Background(), ExecGitRunner{}, scratch, nil, "cat-file", "--batch-all-objects", "--batch-check=%(objectname) %(objecttype) %(objectsize)")
		if err != nil || !bytes.Equal(got, inventory) {
			t.Fatalf("offline inventory differs: %v", err)
		}
	}
}

func TestAssemblyIndexCompatibility(t *testing.T) {
	t.Parallel()
	for _, shape := range []string{"v2", "v3", "v4", "split", "unborn", "sha256", "flags", "sparse", "linked", "promisor-false"} {
		t.Run(shape, func(t *testing.T) {
			root := initLiveRepo(t)
			switch shape {
			case "v2", "v3", "v4":
				gitRun(t, root, "update-index", "--index-version="+shape[1:])
			case "split":
				gitRun(t, root, "update-index", "--split-index")
			case "unborn", "sha256":
				root = t.TempDir()
				args := []string{"init", "-q", "-b", "main"}
				if shape == "sha256" {
					args = append(args, "--object-format=sha256")
				}
				gitRun(t, root, args...)
				gitRun(t, root, "remote", "add", "origin", "https://example.com/repo.git")
				writeContent(t, root, "AGENTS.md", []byte("agent\n"))
				if shape == "sha256" {
					gitRun(t, root, "add", "AGENTS.md")
					gitRun(t, root, "commit", "-qm", "sha256")
				}
			case "promisor-false":
				gitRun(t, root, "config", "remote.origin.promisor", "false")
			case "flags":
				gitRun(t, root, "update-index", "--assume-unchanged", "AGENTS.md")
				gitRun(t, root, "update-index", "--skip-worktree", "README.md")
				writeContent(t, root, "intent", []byte("intent\n"))
				gitRun(t, root, "add", "-N", "intent")
			case "sparse":
				gitRun(t, root, "sparse-checkout", "set", "src")
			case "linked":
				linked := filepath.Join(t.TempDir(), "linked")
				gitRun(t, root, "worktree", "add", "-q", "-b", "linked", linked)
				root = linked
			}
			options := assemblyOptions(t, root)
			before := captureContent(t, root, options.Members[0].Content)
			value := assemble(t, ExecGitRunner{}, options)
			member := firstMember(t, value)
			raw := storedBytes(t, options.Members[0].Content, member["index"].(map[string]any)["blob_id"].(string))
			if string(raw[:4]) != "DIRC" {
				t.Fatal("no portable raw index")
			}
			after := captureContent(t, root, options.Members[0].Content)
			if !sameObservation(before, after) {
				t.Fatal("assembly mutated source")
			}
		})
	}
}

func TestAssemblyOptionsAndUnsupportedSources(t *testing.T) {
	t.Parallel()
	for _, shape := range []string{"nil-runner", "no-members", "group-id", "host-id", "workspace-id", "timestamp", "checkpoint", "policy", "group-path", "config-missing", "config-unknown-repo", "config-duplicate", "config-excluded", "store-missing", "overlap", "shallow", "partial", "replace"} {
		t.Run(shape, func(t *testing.T) {
			root := initLiveRepo(t)
			options := assemblyOptions(t, root)
			var runner AssemblyRunner = ExecGitRunner{}
			switch shape {
			case "nil-runner":
				runner = nil
			case "no-members":
				options.Members = nil
			case "group-id":
				options.WorkspaceGroupID = "bad"
			case "host-id":
				options.CreatedByHostID = "bad"
			case "workspace-id":
				options.Members[0].WorkspaceID = "bad"
			case "timestamp":
				options.CreatedAt = "yesterday"
			case "checkpoint":
				bad := "bad"
				options.BaseCheckpointID = &bad
			case "policy":
				options.Members[0].MaterializationPolicy = "copy"
			case "group-path":
				options.Members[0].GroupRelativePath = "../repo"
			case "config-missing":
				options.Members[0].ProjectConfigPaths["."] = []string{"missing"}
			case "config-unknown-repo":
				options.Members[0].ProjectConfigPaths["missing"] = []string{"AGENTS.md"}
			case "config-duplicate":
				options.Members[0].ProjectConfigPaths["."] = []string{"AGENTS.md", "AGENTS.md"}
			case "config-excluded":
				options.Members[0].Content.Exclude = map[string]string{"AGENTS.md": "credential"}
			case "store-missing":
				options.Members[0].Content.Store = nil
			case "overlap":
				other := options.Members[0]
				other.WorkspaceID = "01900000-0000-7000-8000-000000000004"
				other.GroupRelativePath = "repo/nested"
				options.Members = append(options.Members, other)
			case "shallow":
				head := gitRun(t, root, "rev-parse", "HEAD")
				writeContent(t, root, ".git/shallow", []byte(head))
			case "partial":
				gitRun(t, root, "config", "remote.origin.promisor", "true")
			case "replace":
				head := strings.TrimSpace(gitRun(t, root, "rev-parse", "HEAD"))
				gitRun(t, root, "update-ref", "refs/replace/"+head, head)
			}
			value, err := AssembleProvisional(context.Background(), runner, options)
			if err == nil || value != nil {
				t.Fatal("invalid source/options admitted")
			}
		})
	}
}

func TestAssemblyIsolatedDatabaseIgnoresAmbientObjects(t *testing.T) {
	source := initLiveRepo(t)
	oid := strings.TrimSpace(gitRun(t, source, "rev-parse", "HEAD"))
	t.Setenv("GIT_ALTERNATE_OBJECT_DIRECTORIES", filepath.Join(source, ".git/objects"))
	t.Setenv("GIT_OBJECT_DIRECTORY", filepath.Join(source, ".git/objects"))
	t.Setenv("GIT_INDEX_FILE", filepath.Join(source, ".git/index"))
	scratch := t.TempDir()
	if _, err := isolatedRun(context.Background(), ExecGitRunner{}, scratch, nil, "init", "--bare", "-q", "--template=", "."); err != nil {
		t.Fatal(err)
	}
	result, err := (ExecGitRunner{}).RunIsolated(context.Background(), scratch, nil, "cat-file", "-e", oid)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode == 0 {
		t.Fatal("isolated verifier borrowed ambient object database")
	}
}

func TestAssemblyMultipleMembers(t *testing.T) {
	t.Parallel()
	root := initLiveRepo(t)
	other := initLiveRepo(t)
	options := assemblyOptions(t, root)
	second := options.Members[0]
	second.Directory = other
	second.WorkspaceID = "01900000-0000-7000-8000-000000000004"
	second.GroupRelativePath = "other"
	options.Members = append(options.Members, second)
	setAssemblyRecord(t, &options)
	value := assemble(t, ExecGitRunner{}, options)
	if len(value.Manifests) != 3 {
		t.Fatal("member tree closure lost")
	}
}

func setAssemblyRecord(t *testing.T, options *AssemblyOptions) {
	t.Helper()
	members := []any{}
	for _, source := range options.Members {
		snapshot := captureLive(t, source.Directory)
		urls := map[string]bool{}
		for _, remote := range snapshot.Remotes {
			urls[remote.FetchURL] = true
			if remote.PushURL != nil {
				urls[*remote.PushURL] = true
			}
		}
		members = append(members, map[string]any{"workspace_id": source.WorkspaceID, "kind": "git", "group_relative_path": source.GroupRelativePath, "repository_identity": snapshot.RepositoryIdentity, "sanitized_remote_urls": sortedKeys(urls), "repo_relative_cwd": snapshot.Worktree.CWD, "agent_project_config_paths": append([]string{}, source.ProjectConfigPaths["."]...), "materialization_policy": source.MaterializationPolicy})
	}
	group := map[string]any{"schema": "urn:ax:schema:workspace-group", "schema_version": "1.0.0", "record_id": "sha256:" + strings.Repeat("0", 64), "subject_id": options.WorkspaceGroupID, "workspace_group_id": options.WorkspaceGroupID, "display_name": "assembly fixture", "members": members, "created_by_host_id": options.CreatedByHostID, "created_at": options.CreatedAt, "extensions": map[string]any{}}
	raw, _ := json.Marshal(group)
	id, _, err := canonicaljson.CalculateObjectIdentity(raw)
	if err != nil {
		t.Fatal(err)
	}
	group["record_id"] = id.String()
	options.GroupRecord, _ = json.Marshal(group)
}

func TestAssemblyGroupRecordAgreement(t *testing.T) {
	t.Parallel()
	for _, shape := range []string{"absent", "malformed", "identity", "group", "member-count", "workspace", "path", "repository", "cwd", "config", "policy", "remote"} {
		t.Run(shape, func(t *testing.T) {
			root := initLiveRepo(t)
			options := assemblyOptions(t, root)
			switch shape {
			case "absent":
				options.GroupRecord = nil
			case "malformed":
				options.GroupRecord = []byte("{")
			default:
				var group map[string]any
				if err := json.Unmarshal(options.GroupRecord, &group); err != nil {
					t.Fatal(err)
				}
				member := group["members"].([]any)[0].(map[string]any)
				switch shape {
				case "identity":
					group["display_name"] = "tampered"
				case "group":
					group["workspace_group_id"] = "01900000-0000-7000-8000-000000000009"
					group["subject_id"] = group["workspace_group_id"]
				case "member-count":
					other := map[string]any{}
					for k, v := range member {
						other[k] = v
					}
					other["workspace_id"] = "01900000-0000-7000-8000-000000000009"
					other["group_relative_path"] = "another"
					group["members"] = append(group["members"].([]any), other)
				case "workspace":
					member["workspace_id"] = "01900000-0000-7000-8000-000000000009"
				case "path":
					member["group_relative_path"] = "elsewhere"
				case "repository":
					member["repository_identity"] = "other/repo"
				case "cwd":
					member["repo_relative_cwd"] = "missing"
				case "config":
					member["agent_project_config_paths"] = []string{}
				case "policy":
					member["materialization_policy"] = "separate_worktree"
				case "remote":
					member["sanitized_remote_urls"] = []string{"https://example.com/other.git"}
				}
				raw, _ := json.Marshal(group)
				if shape != "identity" {
					id, _, err := canonicaljson.CalculateObjectIdentity(raw)
					if err != nil {
						t.Fatal(err)
					}
					group["record_id"] = id.String()
					raw, _ = json.Marshal(group)
				}
				options.GroupRecord = raw
			}
			value, err := AssembleProvisional(context.Background(), ExecGitRunner{}, options)
			if err == nil || value != nil {
				t.Fatal("group disagreement admitted")
			}
		})
	}
}

func TestAssemblyRawIndexRefusals(t *testing.T) {
	t.Parallel()
	for _, shape := range []string{"checksum", "required-extension"} {
		t.Run(shape, func(t *testing.T) {
			root := initLiveRepo(t)
			options := assemblyOptions(t, root)
			name := filepath.Join(root, ".git/index")
			raw, err := os.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			if shape == "checksum" {
				raw[len(raw)-1] ^= 1
			} else {
				body := append([]byte{}, raw[:len(raw)-sha1.Size]...)
				body = append(body, []byte{'a', 'x', 'x', 'x', 0, 0, 0, 0}...)
				sum := sha1.Sum(body)
				raw = append(body, sum[:]...)
			}
			if err := os.WriteFile(name, raw, 0600); err != nil {
				t.Fatal(err)
			}
			value, err := AssembleProvisional(context.Background(), ExecGitRunner{}, options)
			if err == nil || value != nil {
				t.Fatal("unsupported raw index admitted")
			}
			t.Logf("refused %s: %v", shape, err)
		})
	}
}

func TestAssemblyMissingConfigClosure(t *testing.T) {
	t.Parallel()
	root := initLiveRepo(t)
	options := assemblyOptions(t, root)
	options.Members[0].ProjectConfigPaths["."] = []string{"missing"}
	setAssemblyRecord(t, &options)
	value, err := AssembleProvisional(context.Background(), ExecGitRunner{}, options)
	if err == nil || value != nil {
		t.Fatal("group-named absent config entered manifest closure")
	}
	if refusal := requireRefusal(t, err, GateContentPath); refusal == nil {
		t.Fatal("missing config closure refusal")
	}
}

func TestAssemblyStoreFailureAndRetry(t *testing.T) {
	t.Parallel()
	root := initLiveRepo(t)
	options := assemblyOptions(t, root)
	objects := filepath.Join(options.Members[0].Content.Store.DataRoot(), "objects")
	runner := assemblyIntercept{input: func(ctx context.Context, dir string, input []byte, args []string) (GitResult, error) {
		result, err := (ExecGitRunner{}).RunInput(ctx, dir, input, args...)
		if err := os.Chmod(objects, 0755); err != nil {
			t.Fatal(err)
		}
		return result, err
	}}
	value, err := AssembleProvisional(context.Background(), runner, options)
	if err == nil || value != nil {
		t.Fatal("failed pack install returned an assembly")
	}
	if refusal := requireRefusal(t, err, GateBlobInstall); refusal == nil {
		t.Fatal("missing blob install refusal")
	}
	if err := os.Chmod(objects, 0700); err != nil {
		t.Fatal(err)
	}
	assemble(t, ExecGitRunner{}, options)
}

// This tests the production graph-construction component at 1024/1025 trees.
// It is not a 1025-live-repository capture or pack/materialization measurement.
func TestAssemblyManifestFanout(t *testing.T) {
	t.Parallel()
	root := initLiveRepo(t)
	options := assemblyOptions(t, root)
	state := &assemblyState{options: options, content: &contentState{options: options.Members[0].Content}, result: &ProvisionalAssembly{Manifests: map[string]json.RawMessage{}, ObjectAddresses: map[string]string{}}}
	for i := 0; i < 1025; i++ {
		subject := "01900000-0000-7000-8000-" + fmt.Sprintf("%012d", i/256+10)
		manifest := state.manifest("workspace_tree", subject, []map[string]any{{"path": fmt.Sprintf("dir%04d", i), "type": "directory", "mode": uint32(0755)}}, []string{}, []string{})
		if _, err := state.seal(context.Background(), manifest); err != nil {
			t.Fatal(err)
		}
		if i == 1023 {
			children, err := state.rootChildren(context.Background())
			if err != nil || len(children) != 1024 {
				t.Fatalf("1024 edge: %d %v", len(children), err)
			}
		}
	}
	children, err := state.rootChildren(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 5 {
		t.Fatalf("partitions=%d, want 5", len(children))
	}
	found := map[string]bool{}
	for _, id := range children {
		var composite map[string]any
		if err := json.Unmarshal(state.result.Manifests[id], &composite); err != nil {
			t.Fatal(err)
		}
		if composite["kind"] != "composite" || len(composite["entries"].([]any)) != 0 {
			t.Fatal("partition duplicates entries")
		}
		for _, child := range composite["child_manifest_ids"].([]any) {
			found[child.(string)] = true
		}
	}
	if len(found) != 1025 {
		t.Fatal("partition lost tree closure")
	}
}

func TestAssemblyInvalidIncludePolicy(t *testing.T) {
	t.Parallel()
	root := initLiveRepo(t)
	options := assemblyOptions(t, root)
	options.Members[0].Content.IncludeIgnored = map[string]bool{"ignored-extra": false}
	value, err := AssembleProvisional(context.Background(), ExecGitRunner{}, options)
	if err == nil || value != nil {
		t.Fatal("unclassified include policy admitted")
	}
	if refusal := requireRefusal(t, err, GateCapturePolicy); refusal.Gate != GateCapturePolicy.Name {
		t.Fatalf("refusal gate = %q", refusal.Gate)
	}
}

func TestAssemblyRecursiveCWDAndConfigClosure(t *testing.T) {
	t.Parallel()
	root, child := submoduleRepo(t)
	writeContent(t, child, "src/agent.cfg", []byte("nested config\n"))
	options := assemblyOptions(t, root)
	options.Members[0].ProjectConfigPaths["."] = []string{"vendor/lib/src/agent.cfg"}
	setAssemblyRecord(t, &options)
	var group map[string]any
	if err := json.Unmarshal(options.GroupRecord, &group); err != nil {
		t.Fatal(err)
	}
	group["members"].([]any)[0].(map[string]any)["repo_relative_cwd"] = "vendor/lib/src"
	raw, _ := json.Marshal(group)
	id, _, err := canonicaljson.CalculateObjectIdentity(raw)
	if err != nil {
		t.Fatal(err)
	}
	group["record_id"] = id.String()
	options.GroupRecord, _ = json.Marshal(group)
	value := assemble(t, ExecGitRunner{}, options)
	if firstMember(t, value)["repo_relative_cwd"] != "vendor/lib/src" {
		t.Fatal("selected group cwd lost")
	}
	options.Members[0].Content.Exclude = map[string]string{"vendor/lib/src/agent.cfg": "credential"}
	rejected, err := AssembleProvisional(context.Background(), ExecGitRunner{}, options)
	if err == nil || rejected != nil {
		t.Fatal("missing nested config admitted")
	}
	if refusal := requireRefusal(t, err, GateContentPath); refusal.Gate != GateContentPath.Name {
		t.Fatalf("refusal gate = %q", refusal.Gate)
	}
}

func TestValidateProvisionalParentChildOverlap(t *testing.T) {
	t.Parallel()
	root, _ := submoduleRepo(t)
	options := assemblyOptions(t, root)
	value := assemble(t, ExecGitRunner{}, options)
	member := firstMember(t, value)
	treeID := member["working_tree_manifest_id"].(string)
	var tree map[string]any
	if err := json.Unmarshal(value.Manifests[treeID], &tree); err != nil {
		t.Fatal(err)
	}
	tree["entries"] = append(tree["entries"].([]any), map[string]any{"path": "vendor/lib/spill", "type": "directory", "mode": float64(0755)})
	raw, _ := json.Marshal(tree)
	id, _, err := canonicaljson.CalculateObjectIdentity(raw)
	if err != nil {
		t.Fatal(err)
	}
	tree["manifest_id"] = id.String()
	raw, _ = json.Marshal(tree)
	delete(value.Manifests, treeID)
	value.Manifests[id.String()] = raw
	object := wireRoot(t, value)
	children := []string{}
	for _, item := range object["child_manifest_ids"].([]any) {
		child := item.(string)
		if child == treeID {
			child = id.String()
		}
		children = append(children, child)
	}
	sort.Strings(children)
	object["child_manifest_ids"] = children
	object["workspace_snapshot"].(map[string]any)["members"].([]any)[0].(map[string]any)["working_tree_manifest_id"] = id.String()
	resealRoot(t, value, object)
	err = ValidateProvisional(value)
	if refusal := requireRefusal(t, err, GateContentPath); refusal.Gate != GateContentPath.Name {
		t.Fatalf("refusal gate = %q", refusal.Gate)
	}
}
