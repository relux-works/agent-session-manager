package gitsnap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The real repositories are created from the exact pinned Section 10.4 pack,
// raw index, and working-byte corpus. Packs are imported into separate object
// databases. This is a capture test, not an implementation of materialization.
func normativeGitWorkspace(t *testing.T) (string, map[string][]byte, map[string]string, map[string]string, string, string) {
	t.Helper()
	source, err := os.ReadFile("../specdoc/SPEC.md")
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(source)
	if digestHex(hash[:]) != "sha256:562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a" {
		t.Fatal("source is not the pinned specification")
	}
	var corpus struct {
		Fixture  string `json:"fixture"`
		Parent   string `json:"parent_commit"`
		Child    string `json:"child_commit"`
		Payloads []struct {
			Label  string `json:"label"`
			Base64 string `json:"base64url"`
			Blob   string `json:"blob_id"`
		} `json:"payloads"`
	}
	found := false
	for _, match := range regexp.MustCompile("(?s)~~~json\\n(.*?)\\n~~~").FindAllSubmatch(source, -1) {
		var tag struct {
			Fixture string `json:"fixture"`
		}
		if json.Unmarshal(match[1], &tag) == nil && tag.Fixture == "ax-git-workspace-v1" {
			if err := json.Unmarshal(match[1], &corpus); err != nil {
				t.Fatal(err)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatal("normative corpus not found")
	}
	payloads := map[string][]byte{}
	ids := map[string]string{}
	for _, p := range corpus.Payloads {
		raw, err := base64.RawURLEncoding.DecodeString(p.Base64)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		if digestHex(sum[:]) != p.Blob {
			t.Fatal("payload digest mismatch")
		}
		payloads[p.Label] = raw
		ids[p.Label] = p.Blob
	}
	root := initLiveRepo(t)
	child := filepath.Join(root, "vendor/lib")
	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatal(err)
	}
	gitRun(t, child, "init", "-q", "-b", "main")
	gitRun(t, child, "remote", "add", "origin", "ssh://git@github.com/relux/lib.git")
	for _, repo := range []struct{ dir, prefix, head string }{{root, "parent", corpus.Parent}, {child, "child", corpus.Child}} {
		command := exec.Command("git", "index-pack", "--stdin")
		command.Dir = repo.dir
		command.Env = isolatedGitEnv()
		command.Stdin = bytes.NewReader(payloads[repo.prefix+"_pack"])
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("index-pack: %v %s", err, output)
		}
		gitRun(t, repo.dir, "update-ref", "refs/heads/main", strings.TrimPrefix(repo.head, "sha1:"))
		writeContent(t, repo.dir, ".git/index", payloads[repo.prefix+"_index"])
	}
	files := map[string]string{"AGENTS.md": "agent_file", "README.md": "working_file", "notes.txt": "notes_file", ".gitmodules": "gitmodules_file", "vendor/lib/README.md": "child_file"}
	for name, label := range files {
		writeContent(t, root, name, payloads[label])
	}
	if err := os.MkdirAll(filepath.Join(root, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	return root, payloads, ids, files, corpus.Parent, corpus.Child
}

func TestContentNormativeWorkspaceBytes(t *testing.T) {
	root, payloads, ids, files, parentOID, childOID := normativeGitWorkspace(t)
	options := contentOptions(t)
	value := captureContent(t, root, options)
	parentEntries := entriesByPath(value.Content)
	childEntries := entriesByPath(value.Content.Submodules[0].State.Content)
	for name, label := range files {
		entry := parentEntries[name]
		if name == "vendor/lib/README.md" {
			entry = childEntries["README.md"]
		}
		if entry == nil || entry["blob_id"] != ids[label] {
			t.Fatalf("%s differs from pinned working-byte corpus", name)
		}
		if !bytes.Equal(storedBytes(t, options, entry["blob_id"].(string)), payloads[label]) {
			t.Fatal("stored fixture differs")
		}
	}
	if value.Content.Submodules[0].GitlinkOID != childOID || *value.Head.OID != parentOID {
		t.Fatal("fixture pointers differ")
	}
}

func TestContentExcludedSubmoduleCannotBypassPolicy(t *testing.T) {
	root, _ := submoduleRepo(t)
	options := contentOptions(t)
	options.Exclude = map[string]string{"vendor": "credential"}
	value, err := Capture(context.Background(), ExecGitRunner{}, root, options)
	requireRefusal(t, err, GateCapturePolicy)
	if value != nil {
		t.Fatal("excluded child captured")
	}
}

func TestContentSparseReadFailure(t *testing.T) {
	root := initLiveRepo(t)
	options := contentOptions(t)
	gitRun(t, root, "sparse-checkout", "set", "src")
	sparse := strings.TrimSpace(gitRun(t, root, "rev-parse", "--path-format=absolute", "--git-path", "info/sparse-checkout"))
	if err := os.Remove(sparse); err != nil {
		t.Fatal(err)
	}
	_, err := Capture(context.Background(), ExecGitRunner{}, root, options)
	requireRefusal(t, err, GateContentRead)
}
