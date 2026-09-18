package clonesnap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
)

// Workspace checkpoint evidence: CheckpointWorkspace builds the
// closed WorkspaceBinding from observed repository state through
// the production entry, and every escape or shape violation refuses
// with the member named.

func TestCheckpointWorkspaceSealsObservedState(t *testing.T) {
	request := fixtureWorkspace(t)
	sealed, err := CheckpointWorkspace(request)
	if err != nil {
		t.Fatalf("CheckpointWorkspace() error = %v", err)
	}
	binding, err := clonebundle.DecodeWorkspaceBinding(sealed)
	if err != nil {
		t.Fatalf("DecodeWorkspaceBinding() error = %v", err)
	}
	if binding.LogicalWorkspaceID.String() != fixtureWorkspaceID {
		t.Fatalf("LogicalWorkspaceID = %q", binding.LogicalWorkspaceID)
	}
	if binding.CwdRelative != "work/trees/alpha" {
		t.Fatalf("CwdRelative = %q", binding.CwdRelative)
	}
	if len(binding.RemoteFingerprints) != 1 || binding.RemoteFingerprints[0].String() != fixtureDigest("snap-remote") {
		t.Fatalf("RemoteFingerprints = %v", binding.RemoteFingerprints)
	}
	if binding.Branch == nil || *binding.Branch != "main" {
		t.Fatalf("Branch = %v", binding.Branch)
	}
	if binding.HeadDigest == nil || binding.HeadDigest.String() != fixtureDigest("snap-head") {
		t.Fatalf("HeadDigest = %v", binding.HeadDigest)
	}
	if binding.IndexDigest != nil || binding.WorkingTreeDigest != nil {
		t.Fatal("null digests did not seal as null")
	}
}

func TestCheckpointWorkspaceRelativeCwd(t *testing.T) {
	request := fixtureWorkspace(t)
	request.Cwd = "work/trees/alpha"
	sealed, err := CheckpointWorkspace(request)
	if err != nil {
		t.Fatalf("CheckpointWorkspace() error = %v", err)
	}
	binding, err := clonebundle.DecodeWorkspaceBinding(sealed)
	if err != nil {
		t.Fatalf("DecodeWorkspaceBinding() error = %v", err)
	}
	if binding.CwdRelative != "work/trees/alpha" {
		t.Fatalf("CwdRelative = %q", binding.CwdRelative)
	}
}

func TestCheckpointWorkspaceRootCwd(t *testing.T) {
	request := fixtureWorkspace(t)
	request.Cwd = request.WorkspaceRoot
	sealed, err := CheckpointWorkspace(request)
	if err != nil {
		t.Fatalf("CheckpointWorkspace() error = %v", err)
	}
	binding, err := clonebundle.DecodeWorkspaceBinding(sealed)
	if err != nil {
		t.Fatalf("DecodeWorkspaceBinding() error = %v", err)
	}
	if binding.CwdRelative != "." {
		t.Fatalf("CwdRelative = %q, want .", binding.CwdRelative)
	}
}

func TestCheckpointWorkspaceRefusesEscape(t *testing.T) {
	request := fixtureWorkspace(t)
	request.Cwd = "/outside-workspace"
	if _, err := CheckpointWorkspace(request); err == nil {
		t.Fatal("CheckpointWorkspace() admitted an absolute escape")
	} else {
		requireRefusalDetail(t, err, "escapes the workspace root")
	}
	// A different escape still refuses after the
	// admit-exactly-/outside-workspace narrowing.
	request.Cwd = "/outside-workspace-2"
	if _, err := CheckpointWorkspace(request); err == nil {
		t.Fatal("CheckpointWorkspace() admitted an absolute escape")
	} else {
		requireRefusalDetail(t, err, "escapes the workspace root")
	}
}

func TestCheckpointWorkspaceRefusesDotSegments(t *testing.T) {
	request := fixtureWorkspace(t)
	request.Cwd = "work/../escape"
	if _, err := CheckpointWorkspace(request); err == nil {
		t.Fatal("CheckpointWorkspace() admitted parent segments")
	} else {
		requireRefusalDetail(t, err, `"work/../escape"`)
	}
}

func TestCheckpointWorkspaceRefusesMissingCwd(t *testing.T) {
	request := fixtureWorkspace(t)
	request.Cwd = "work/trees/missing"
	if _, err := CheckpointWorkspace(request); err == nil {
		t.Fatal("CheckpointWorkspace() admitted an unobserved cwd")
	} else {
		requireRefusalDetail(t, err, `"work/trees/missing"`)
	}
}

func TestCheckpointWorkspaceRefusesFileCwd(t *testing.T) {
	request := fixtureWorkspace(t)
	plain := filepath.Join(request.WorkspaceRoot, "plain-file")
	if err := os.WriteFile(plain, []byte("not-a-directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	request.Cwd = "plain-file"
	if _, err := CheckpointWorkspace(request); err == nil {
		t.Fatal("CheckpointWorkspace() admitted a file cwd")
	} else {
		requireRefusalDetail(t, err, `"plain-file"`)
	}
}

func TestCheckpointWorkspaceRefusesUnsortedFingerprints(t *testing.T) {
	request := fixtureWorkspace(t)
	first := fixtureDigest("snap-remote-a")
	second := fixtureDigest("snap-remote-b")
	fingerprints := []string{first, second}
	if fingerprints[0] > fingerprints[1] {
		fingerprints[0], fingerprints[1] = fingerprints[1], fingerprints[0]
	}
	// Descending order refuses.
	request.RemoteFingerprints = []string{fingerprints[1], fingerprints[0]}
	if _, err := CheckpointWorkspace(request); err == nil {
		t.Fatal("CheckpointWorkspace() admitted unsorted fingerprints")
	} else {
		requireRefusalDetail(t, err, "unsorted or duplicate repository remote fingerprints")
	}
	// Ascending order seals.
	request.RemoteFingerprints = []string{fingerprints[0], fingerprints[1]}
	if _, err := CheckpointWorkspace(request); err != nil {
		t.Fatalf("CheckpointWorkspace() error = %v", err)
	}
}

func TestCheckpointWorkspaceRefusesDuplicateFingerprint(t *testing.T) {
	request := fixtureWorkspace(t)
	duplicate := fixtureDigest("snap-remote")
	request.RemoteFingerprints = []string{duplicate, duplicate}
	if _, err := CheckpointWorkspace(request); err == nil {
		t.Fatal("CheckpointWorkspace() admitted duplicate fingerprints")
	} else {
		requireRefusalDetail(t, err, "unsorted or duplicate repository remote fingerprints")
	}
}

func TestCheckpointWorkspaceRefusesTooManyFingerprints(t *testing.T) {
	request := fixtureWorkspace(t)
	fingerprints := make([]string, 0, 129)
	for index := 0; index < 129; index++ {
		fingerprints = append(fingerprints, fixtureDigest("snap-remote-"+string(rune('a'+index/26))+string(rune('a'+index%26))))
	}
	request.RemoteFingerprints = fingerprints
	if _, err := CheckpointWorkspace(request); err == nil {
		t.Fatal("CheckpointWorkspace() admitted 129 fingerprints")
	} else {
		requireRefusalDetail(t, err, "maximum is 128")
	}
}

func TestCheckpointWorkspaceRefusesBadDigest(t *testing.T) {
	request := fixtureWorkspace(t)
	bad := "not-a-digest"
	request.HeadDigest = &bad
	if _, err := CheckpointWorkspace(request); err == nil {
		t.Fatal("CheckpointWorkspace() admitted a bad head digest")
	} else {
		requireRefusalDetail(t, err, "head_digest")
	}
}

func TestCheckpointWorkspaceRefusesBadWorkspaceID(t *testing.T) {
	request := fixtureWorkspace(t)
	request.LogicalWorkspaceID = "not-a-uuid"
	if _, err := CheckpointWorkspace(request); err == nil {
		t.Fatal("CheckpointWorkspace() admitted a bad workspace ID")
	} else {
		requireRefusalDetail(t, err, "UUIDv7")
	}
}

func TestCheckpointWorkspaceRefusesSymlinkRoot(t *testing.T) {
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "link-root")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	request := fixtureWorkspace(t)
	request.WorkspaceRoot = link
	if _, err := CheckpointWorkspace(request); err == nil {
		t.Fatal("CheckpointWorkspace() admitted a symlinked root")
	}
}

func TestCheckpointWorkspaceMultibyteBranch(t *testing.T) {
	// The branch bound counts characters, not bytes: 1024
	// two-byte characters admit.
	request := fixtureWorkspace(t)
	branch := strings.Repeat("é", 1024)
	request.Branch = &branch
	if _, err := CheckpointWorkspace(request); err != nil {
		t.Fatalf("CheckpointWorkspace() error = %v", err)
	}
	tooLong := strings.Repeat("é", 1025)
	request.Branch = &tooLong
	if _, err := CheckpointWorkspace(request); err == nil {
		t.Fatal("CheckpointWorkspace() admitted a 1025-character branch")
	} else {
		requireRefusalDetail(t, err, "string[1..1024]")
	}
}
