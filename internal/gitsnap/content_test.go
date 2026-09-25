package gitsnap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/localstore"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

func contentOptions(t *testing.T) ContentOptions {
	t.Helper()
	platform := scalar.PlatformLinux
	if runtime.GOOS == "darwin" {
		platform = scalar.PlatformMacOS
	}
	if runtime.GOOS == "windows" {
		platform = scalar.PlatformWindows
	}
	paths, err := localstore.ResolvePaths(localstore.ResolveRequest{Platform: platform, HomeDir: t.TempDir(), TemporaryDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	store, err := localstore.OpenObjectStore(paths)
	if err != nil {
		t.Fatal(err)
	}
	return ContentOptions{Store: store}
}
func writeContent(t *testing.T, root, name string, data []byte) {
	t.Helper()
	native := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(native), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(native, data, 0644); err != nil {
		t.Fatal(err)
	}
}
func captureContent(t *testing.T, root string, options ContentOptions) *Snapshot {
	t.Helper()
	value, err := Capture(context.Background(), ExecGitRunner{}, root, options)
	if err != nil {
		t.Fatal(err)
	}
	if value.Content == nil {
		t.Fatal("content capture was bypassed")
	}
	return value
}
func entriesByPath(content *Content) map[string]map[string]any {
	result := map[string]map[string]any{}
	for _, entry := range content.Entries {
		result[entry["path"].(string)] = entry
	}
	return result
}
func storedBytes(t *testing.T, options ContentOptions, id string) []byte {
	t.Helper()
	parsed, err := scalar.ParseDigest(id)
	if err != nil {
		t.Fatal(err)
	}
	relative, err := localstore.DigestPathV1(parsed, localstore.PathStylePOSIX)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(options.Store.DataRoot(), "objects", filepath.FromSlash(relative)))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestContentCaptureSelectionAndBytes(t *testing.T) {
	root := initLiveRepo(t)
	options := contentOptions(t)
	writeContent(t, root, ".gitignore", []byte("build/\nAGENTS.local.md\n"))
	writeContent(t, root, "README.md", []byte("staged\n"))
	gitRun(t, root, "add", "README.md")
	writeContent(t, root, "README.md", []byte("working\n"))
	writeContent(t, root, "notes.txt", []byte("notes\n"))
	writeContent(t, root, "AGENTS.local.md", []byte("project rules\n"))
	writeContent(t, root, "build/output", []byte("build bytes"))
	writeContent(t, root, "auth.json", []byte("excluded auth"))
	writeContent(t, root, "private/runtime", []byte("local bytes"))
	writeContent(t, root, "nested/tool.lock", []byte("live lock"))
	options.IncludeIgnored = map[string]bool{"AGENTS.local.md": true, "auth.json": true}
	options.Exclude = map[string]string{"private": "terminal_runtime", "nested/tool.lock": "transient_lock"}
	writeContent(t, root, "Cargo.lock", []byte("dependency versions"))
	value := captureContent(t, root, options)
	entries := entriesByPath(value.Content)
	for _, name := range []string{"README.md", "AGENTS.md", "notes.txt", "AGENTS.local.md", ".gitignore", "nested", "Cargo.lock"} {
		if entries[name] == nil {
			t.Errorf("missing %s", name)
		}
	}
	for _, name := range []string{"build", "build/output", "auth.json", "private", "nested/tool.lock", ".git"} {
		if entries[name] != nil {
			t.Errorf("excluded %s captured", name)
		}
	}
	for name, want := range map[string]string{"README.md": "working\n", "notes.txt": "notes\n", "AGENTS.local.md": "project rules\n"} {
		if got := string(storedBytes(t, options, entries[name]["blob_id"].(string))); got != want {
			t.Errorf("%s bytes=%q", name, got)
		}
	}
	if len(value.Staged) != 1 || len(value.Unstaged) != 1 {
		t.Fatalf("lost staged/unstaged distinction: %+v %+v", value.Staged, value.Unstaged)
	}
	classes := strings.Join(value.Content.ExcludedClasses, ",")
	for _, class := range []string{"credential", "ignored", "terminal_runtime", "transient_lock", "git_metadata"} {
		if !strings.Contains(classes, class) {
			t.Errorf("missing class %s", class)
		}
	}
	for _, descriptor := range value.Content.Blobs {
		raw, _ := json.Marshal(descriptor)
		if _, _, err := canonicaljson.VerifyObjectIdentity(raw); err != nil {
			t.Fatal(err)
		}
	}
}

func TestContentIgnoredDefaultAndPolicyRefusal(t *testing.T) {
	root := initLiveRepo(t)
	options := contentOptions(t)
	writeContent(t, root, ".gitignore", []byte("ignored/\n"))
	writeContent(t, root, "ignored/allow", []byte("yes"))
	writeContent(t, root, "ignored/deny", []byte("no"))
	if entriesByPath(captureContent(t, root, options).Content)["ignored"] != nil {
		t.Fatal("ignored directory included by default")
	}
	options.IncludeIgnored = map[string]bool{"ignored/allow": true}
	entries := entriesByPath(captureContent(t, root, options).Content)
	if entries["ignored/allow"] == nil || entries["ignored/deny"] != nil {
		t.Fatal("ignored include scope widened")
	}
	options.IncludeIgnored["ignored/allow"] = false
	value, err := Capture(context.Background(), ExecGitRunner{}, root, options)
	requireRefusal(t, err, GateCapturePolicy)
	if value != nil {
		t.Fatal("partial capture")
	}
}

func TestContentSymlinks(t *testing.T) {
	root := initLiveRepo(t)
	options := contentOptions(t)
	for name, target := range map[string]string{"current": "README.md", "dangling": "absent", "nested/up": "../README.md"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	entries := entriesByPath(captureContent(t, root, options).Content)
	if entries["current"]["target"] != "README.md" || entries["dangling"]["target"] != "absent" || entries["nested/up"]["target"] != "../README.md" {
		t.Fatal("symlink targets not preserved")
	}
	for _, target := range []string{"../outside", "/etc/passwd"} {
		t.Run(target, func(t *testing.T) {
			if err := os.Symlink(target, filepath.Join(root, "escape")); err != nil {
				t.Fatal(err)
			}
			defer os.Remove(filepath.Join(root, "escape"))
			_, err := Capture(context.Background(), ExecGitRunner{}, root, options)
			requireRefusal(t, err, GateContentPath)
		})
	}
}

func TestContentLargeBlobChunksAndLimit(t *testing.T) {
	root := initLiveRepo(t)
	options := contentOptions(t)
	data := bytes.Repeat([]byte{0, 1, 2, 3}, chunkSize/2+1)
	writeContent(t, root, "large.bin", data)
	writeContent(t, root, "empty", nil)
	value := captureContent(t, root, options)
	entries := entriesByPath(value.Content)
	if !bytes.Equal(storedBytes(t, options, entries["large.bin"]["blob_id"].(string)), data) {
		t.Fatal("large bytes differ")
	}
	for _, descriptor := range value.Content.Blobs {
		if descriptor.BlobID == entries["large.bin"]["blob_id"] {
			if len(descriptor.Chunks) != 3 || descriptor.Chunks[2].Size != 4 || descriptor.Chunks[2].Offset != 2*chunkSize {
				t.Fatalf("chunks=%+v", descriptor.Chunks)
			}
		}
		if descriptor.BlobID == entries["empty"]["blob_id"] && len(descriptor.Chunks) != 0 {
			t.Fatal("empty blob has chunks")
		}
	}
	if err := os.Truncate(filepath.Join(root, "large.bin"), maxBlobSize+1); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	value, err := Capture(ctx, ExecGitRunner{}, root, options)
	requireRefusal(t, err, GateBlobLimit)
	if value != nil {
		t.Fatal("partial oversize capture")
	}
}

type contentFaultRunner struct {
	Runner
	command string
	result  GitResult
	action  func()
}

func (r contentFaultRunner) Run(ctx context.Context, root string, args ...string) (GitResult, error) {
	if strings.Join(args, " ") == r.command {
		if r.action != nil {
			r.action()
		}
		return r.result, nil
	}
	return r.Runner.Run(ctx, root, args...)
}
func TestContentReadFailureIsNotAbsence(t *testing.T) {
	root := initLiveRepo(t)
	options := contentOptions(t)
	for _, result := range []GitResult{{ExitCode: 128}, {ExitCode: 0, Stdout: []byte("partial")}} {
		runner := contentFaultRunner{Runner: ExecGitRunner{}, command: "ls-files --others --ignored --exclude-standard --directory -z", result: result}
		value, err := Capture(context.Background(), runner, root, options)
		requireRefusal(t, err, GateContentRead)
		if value != nil {
			t.Fatal("partial snapshot")
		}
	}
	captureContent(t, root, options)
}
func TestContentBlobInstallFailureAndRetry(t *testing.T) {
	root := initLiveRepo(t)
	options := contentOptions(t)
	objects := filepath.Join(options.Store.DataRoot(), "objects")
	if err := os.Chmod(objects, 0755); err != nil {
		t.Fatal(err)
	}
	value, err := Capture(context.Background(), ExecGitRunner{}, root, options)
	requireRefusal(t, err, GateBlobInstall)
	if value != nil {
		t.Fatal("failed install returned snapshot")
	}
	if err := os.Chmod(objects, 0700); err != nil {
		t.Fatal(err)
	}
	first := captureContent(t, root, options)
	second := captureContent(t, root, options)
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if !bytes.Equal(a, b) {
		t.Fatal("retry not idempotent")
	}
}
func TestContentSparseLinkedWorktree(t *testing.T) {
	root := initLiveRepo(t)
	options := contentOptions(t)
	writeContent(t, root, "present/a", []byte("present"))
	writeContent(t, root, "absent/b", []byte("absent"))
	gitRun(t, root, "add", ".")
	gitRun(t, root, "commit", "-qm", "directories")
	linked := filepath.Join(t.TempDir(), "linked")
	gitRun(t, root, "worktree", "add", "-b", "sparse", linked)
	gitRun(t, linked, "sparse-checkout", "set", "present")
	value := captureContent(t, linked, options)
	entries := entriesByPath(value.Content)
	if entries["present/a"] == nil || entries["absent/b"] != nil {
		t.Fatal("sparse working bytes incorrect")
	}
	if value.Content.SparsePatterns == nil || !value.Features.SparseCheckout {
		t.Fatal("missing sparse patterns")
	}
	patterns := storedBytes(t, options, value.Content.SparsePatterns.BlobID)
	if !bytes.Contains(patterns, []byte("present")) {
		t.Fatal("wrong worktree patterns")
	}
	if value.Worktree.GitDir == value.Worktree.CommonDir {
		t.Fatal("linked worktree metadata collapsed")
	}
}

func submoduleRepo(t *testing.T) (string, string) {
	t.Helper()
	root := initLiveRepo(t)
	source := initLiveRepo(t)
	gitRun(t, root, "-c", "protocol.file.allow=always", "submodule", "add", source, "vendor/lib")
	child := filepath.Join(root, "vendor/lib")
	gitRun(t, root, "config", "--file", ".gitmodules", "submodule.vendor/lib.url", "ssh://git@github.com/relux/lib.git")
	gitRun(t, child, "remote", "set-url", "origin", "ssh://git@github.com/relux/lib.git")
	gitRun(t, root, "add", ".gitmodules", "vendor/lib")
	gitRun(t, root, "commit", "-qm", "submodule")
	return root, child
}
func TestContentSubmodulePointerStates(t *testing.T) {
	for _, shape := range []string{"clean", "staged", "unstaged", "combined"} {
		t.Run(shape, func(t *testing.T) {
			root, child := submoduleRepo(t)
			options := contentOptions(t)
			base := strings.TrimSpace(gitRun(t, child, "rev-parse", "HEAD"))
			indexOID := base
			headOID := base
			advance := func(label string) string {
				writeContent(t, child, "README.md", []byte(label))
				gitRun(t, child, "add", "README.md")
				gitRun(t, child, "commit", "-qm", label)
				return strings.TrimSpace(gitRun(t, child, "rev-parse", "HEAD"))
			}
			if shape == "staged" || shape == "combined" {
				indexOID = advance("staged child")
				headOID = indexOID
				gitRun(t, root, "add", "vendor/lib")
			}
			if shape == "unstaged" || shape == "combined" {
				headOID = advance("unstaged child")
			}
			writeContent(t, child, "notes", []byte("child untracked"))
			writeContent(t, child, "README.md", []byte("child dirty"))
			value := captureContent(t, root, options)
			if len(value.Content.Submodules) != 1 {
				t.Fatal("missing child")
			}
			sub := value.Content.Submodules[0]
			if !sub.Initialized || sub.GitlinkOID != "sha1:"+indexOID || sub.State.Head.OID == nil || *sub.State.Head.OID != "sha1:"+headOID {
				t.Fatalf("pointer state %+v", sub)
			}
			if entriesByPath(sub.State.Content)["notes"] == nil || entriesByPath(value.Content)["vendor/lib/notes"] != nil {
				t.Fatal("child boundary lost")
			}
			if got := strings.TrimSpace(gitRun(t, root, "ls-tree", "HEAD", "vendor/lib")); !strings.Contains(got, base) {
				t.Fatal("parent HEAD pointer changed")
			}
		})
	}
}
func TestContentSubmoduleUninitializedAndRefusals(t *testing.T) {
	root, child := submoduleRepo(t)
	options := contentOptions(t)
	gitRun(t, root, "submodule", "deinit", "-f", "vendor/lib")
	value := captureContent(t, root, options)
	sub := value.Content.Submodules[0]
	if sub.Initialized || sub.State != nil {
		t.Fatal("uninitialized state invented")
	}
	writeContent(t, child, "stray", []byte("unrepresented"))
	_, err := Capture(context.Background(), ExecGitRunner{}, root, options)
	requireRefusal(t, err, GateSubmoduleState)
	os.Remove(filepath.Join(child, "stray"))
	gitRun(t, root, "config", "--file", ".gitmodules", "submodule.vendor/lib.url", "https://user:secret@example.com/lib.git")
	_, err = Capture(context.Background(), ExecGitRunner{}, root, options)
	requireRefusal(t, err, GateSubmoduleState)
}

func TestContentSymlinkChainsAndExcludedTargets(t *testing.T) {
	root := initLiveRepo(t)
	options := contentOptions(t)
	if err := os.Symlink("missing", filepath.Join(root, "chain-end")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("chain-end", filepath.Join(root, "chain-start")); err != nil {
		t.Fatal(err)
	}
	captureContent(t, root, options)
	outside := t.TempDir()
	writeContent(t, outside, "secret", []byte("outside bytes"))
	if err := os.Symlink(outside, filepath.Join(root, "outside")); err != nil {
		t.Fatal(err)
	}
	options.Exclude = map[string]string{"outside": "machine_local"}
	if err := os.Symlink("outside/secret", filepath.Join(root, "escape-chain")); err != nil {
		t.Fatal(err)
	}
	value, err := Capture(context.Background(), ExecGitRunner{}, root, options)
	requireRefusal(t, err, GateContentPath)
	if value != nil {
		t.Fatal("outside symlink admitted")
	}
}

func TestContentSymlinkChainBoundaryAtProductionCapture(t *testing.T) {
	root := initLiveRepo(t)
	options := contentOptions(t)
	link := func(i int) string { return filepath.Join(root, fmt.Sprintf("link-%02d", i)) }
	for i := 0; i < 41; i++ {
		target := "missing"
		if i < 40 {
			target = fmt.Sprintf("link-%02d", i+1)
		}
		if err := os.Symlink(target, link(i)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Capture(context.Background(), ExecGitRunner{}, root, options); err != nil {
		t.Fatalf("40-hop symlink chain refused: %v", err)
	}
	if err := os.Remove(link(40)); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("link-41", link(40)); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing", link(41)); err != nil {
		t.Fatal(err)
	}
	_, err := Capture(context.Background(), ExecGitRunner{}, root, options)
	if refusal := requireRefusal(t, err, GateContentPath); refusal.Gate != GateContentPath.Name {
		t.Fatalf("refusal gate = %q", refusal.Gate)
	}
}

func TestContentLocalPathOwnerExclusions(t *testing.T) {
	root := initLiveRepo(t)
	options := contentOptions(t)
	platform := scalar.PlatformLinux
	if runtime.GOOS == "darwin" {
		platform = scalar.PlatformMacOS
	}
	if runtime.GOOS == "windows" {
		platform = scalar.PlatformWindows
	}
	paths, err := localstore.ResolvePaths(localstore.ResolveRequest{Platform: platform, HomeDir: root, TemporaryDir: root})
	if err != nil {
		t.Fatal(err)
	}
	options.LocalPaths = &paths
	for _, p := range paths.Paths() {
		name := p.Value.String()
		if p.Class != localstore.PathConfig {
			name = filepath.Join(name, "sensitive")
		}
		rel, err := filepath.Rel(root, name)
		if err != nil {
			t.Fatal(err)
		}
		writeContent(t, root, filepath.ToSlash(rel), []byte("machine local"))
	}
	value := captureContent(t, root, options)
	for _, entry := range value.Content.Entries {
		for _, p := range paths.Paths() {
			if pathWithin(p.Value.String(), filepath.Join(root, entry["path"].(string))) {
				t.Fatal("AX local path copied")
			}
		}
	}
	if !strings.Contains(strings.Join(value.Content.ExcludedClasses, ","), "machine_local") {
		t.Fatal("missing exclusion evidence")
	}
}

func TestContentDigestRaceAndRetry(t *testing.T) {
	root := initLiveRepo(t)
	options := contentOptions(t)
	gitRun(t, root, "update-index", "--assume-unchanged", "README.md")
	gitRun(t, root, "sparse-checkout", "set", "src")
	sparse := gitRun(t, root, "rev-parse", "--path-format=absolute", "--git-path", "info/sparse-checkout")
	fired := false
	runner := contentFaultRunner{Runner: ExecGitRunner{}, command: "rev-parse --path-format=absolute --git-path info/sparse-checkout", result: okResult(sparse), action: func() { fired = true; writeContent(t, root, "README.md", []byte("race\n")) }}
	value, err := Capture(context.Background(), runner, root, options)
	requireRefusal(t, err, GateConsistency)
	if !fired || value != nil {
		t.Fatal("race did not fire or partial capture")
	}
	captureContent(t, root, options)
}

func TestContentSubmoduleCountBounds(t *testing.T) {
	root := initLiveRepo(t)
	options := contentOptions(t)
	oid := strings.TrimSpace(gitRun(t, root, "rev-parse", "HEAD"))
	var mappings, index strings.Builder
	for i := 0; i < 256; i++ {
		name := fmt.Sprintf("child-%03d", i)
		fmt.Fprintf(&mappings, "[submodule \"%s\"]\n path = %s\n url = ssh://git@github.com/relux/lib.git\n", name, name)
		fmt.Fprintf(&index, "160000 %s\t%s\n", oid, name)
	}
	writeContent(t, root, ".gitmodules", []byte(mappings.String()))
	gitRun(t, root, "add", ".gitmodules")
	updateIndex := func(raw string) {
		command := exec.Command("git", "update-index", "--index-info")
		command.Dir = root
		command.Env = isolatedGitEnv()
		command.Stdin = strings.NewReader(raw)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("update-index: %v %s", err, output)
		}
	}
	updateIndex(index.String())
	value := captureContent(t, root, options)
	if len(value.Content.Submodules) != 256 {
		t.Fatal("lost uninitialized submodules")
	}
	updateIndex(fmt.Sprintf("160000 %s\tchild-256\n", oid))
	_, err := Capture(context.Background(), ExecGitRunner{}, root, options)
	refusal := requireRefusal(t, err, GateSubmoduleState)
	if !strings.Contains(refusal.Detail, "total") {
		t.Fatal("count boundary masked by other refusal")
	}
}

func TestContentRelativeSubmoduleURL(t *testing.T) {
	root, _ := submoduleRepo(t)
	options := contentOptions(t)
	gitRun(t, root, "config", "--file", ".gitmodules", "submodule.vendor/lib.url", "../lib.git")
	value := captureContent(t, root, options)
	if value.Content.Submodules[0].SanitizedURL != "ssh://git@github.com/relux/lib.git" {
		t.Fatal("relative URL not resolved against default remote")
	}
	gitRun(t, root, "remote", "add", "upstream", "https://example.com/team/parent.git")
	gitRun(t, root, "config", "branch.main.remote", "upstream")
	value = captureContent(t, root, options)
	if value.Content.Submodules[0].SanitizedURL != "https://example.com/team/lib.git" {
		t.Fatal("tracking remote ignored")
	}
	gitRun(t, root, "config", "branch.main.remote", ".")
	_, err := Capture(context.Background(), ExecGitRunner{}, root, options)
	requireRefusal(t, err, GateSubmoduleState)
}

func TestContentNestedDetachedSubmodule(t *testing.T) {
	root, child := submoduleRepo(t)
	source := initLiveRepo(t)
	options := contentOptions(t)
	gitRun(t, child, "-c", "protocol.file.allow=always", "submodule", "add", source, "nested")
	grand := filepath.Join(child, "nested")
	gitRun(t, child, "config", "--file", ".gitmodules", "submodule.nested.url", "ssh://git@github.com/relux/nested.git")
	gitRun(t, grand, "remote", "set-url", "origin", "ssh://git@github.com/relux/nested.git")
	gitRun(t, grand, "checkout", "--detach", "-q")
	writeContent(t, grand, ".gitignore", []byte("AGENTS.local.md\n"))
	writeContent(t, grand, "AGENTS.local.md", []byte("nested rules"))
	writeContent(t, grand, "notes", []byte("nested untracked"))
	options.IncludeIgnored = map[string]bool{"vendor/lib/nested/AGENTS.local.md": true}
	value := captureContent(t, root, options)
	sub := value.Content.Submodules[0].State.Content.Submodules[0]
	if sub.State == nil || sub.State.Head.Mode != "detached" || sub.State.Head.Ref != nil {
		t.Fatal("nested detached state lost")
	}
	entries := entriesByPath(sub.State.Content)
	if entries["AGENTS.local.md"] == nil || entries["notes"] == nil {
		t.Fatal("nested content policy lost")
	}
}

func TestContentExclusionClassCountBounds(t *testing.T) {
	root := initLiveRepo(t)
	options := contentOptions(t)
	options.Exclude = map[string]string{}
	for i := 0; i < 127; i++ {
		name := fmt.Sprintf("excluded-%03d", i)
		writeContent(t, root, name, []byte("excluded"))
		options.Exclude[name] = fmt.Sprintf("class-%03d", i)
	}
	value := captureContent(t, root, options)
	if len(value.Content.ExcludedClasses) != 128 {
		t.Fatal("wrong at-limit class count")
	}
	writeContent(t, root, "excluded-127", []byte("excluded"))
	options.Exclude["excluded-127"] = "class-127"
	_, err := Capture(context.Background(), ExecGitRunner{}, root, options)
	requireRefusal(t, err, GateCapturePolicy)
}
