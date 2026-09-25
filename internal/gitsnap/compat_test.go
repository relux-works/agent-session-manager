package gitsnap

import (
	"context"
	"strings"
	"testing"
)

// TestCompatIndexVersions admits every member of the closed index
// version domain through the production entry: 2, 3, and 4 all capture.
func TestCompatIndexVersions(t *testing.T) {
	t.Parallel()
	for _, version := range []string{"2", "3", "4"} {
		fake := scriptBaseline(newScriptedRunner())
		fake.Rescript([]string{"update-index", "--show-index-version"}, okResult(version+"\n"), okResult(version+"\n"))
		snapshot, err := Capture(context.Background(), fake, "/tmp/repo")
		if err != nil {
			t.Errorf("Capture() with index version %s error = %v", version, err)
			continue
		}
		if snapshot.Index.Version != int(version[0]-'0') || snapshot.Index.Format != "git_index" {
			t.Errorf("Index = %+v, want git_index version %s", snapshot.Index, version)
		}
	}
}

// TestCompatAbsentIndexDefaultsToTwo covers the documented version-0
// rule: a repository with no index file yet captures version 2, the
// version git itself would write.
func TestCompatAbsentIndexDefaultsToTwo(t *testing.T) {
	t.Parallel()
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"update-index", "--show-index-version"}, okResult("0\n"), okResult("0\n"))
	fake.Script([]string{"config", "--get", "index.version"}, exitResult(1, ""), exitResult(1, ""))
	snapshot, err := Capture(context.Background(), fake, "/tmp/repo")
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if snapshot.Index.Version != 2 {
		t.Errorf("Index.Version = %d, want 2", snapshot.Index.Version)
	}
}

// TestCompatAbsentIndexHonorsConfiguredVersion covers the second arm of
// the version-0 rule: an explicit index.version=3 is honored.
func TestCompatAbsentIndexHonorsConfiguredVersion(t *testing.T) {
	t.Parallel()
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"update-index", "--show-index-version"}, okResult("0\n"), okResult("0\n"))
	fake.Script([]string{"config", "--get", "index.version"}, okResult("3\n"), okResult("3\n"))
	snapshot, err := Capture(context.Background(), fake, "/tmp/repo")
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if snapshot.Index.Version != 3 {
		t.Errorf("Index.Version = %d, want 3", snapshot.Index.Version)
	}
}

// TestCompatConflictStages Admits every conflict stage: 1, 2, and 3
// capture alongside stage 0 in one entry stream.
func TestCompatConflictStages(t *testing.T) {
	t.Parallel()
	var builder strings.Builder
	for _, stage := range []string{"0", "1", "2", "3"} {
		builder.WriteString(stageProbeIndex("100644", "19d9cc8584ac2c7dcf57d2680375e80f099dc481", stage, "README.md"))
	}
	// Same-path stages sort after the stage-0 AGENTS.md entry.
	index := "100644 b6b0be997c9c8246cdd346dd7ece72140d74dee0 0\tAGENTS.md\x00" + stageProbeDebug + builder.String()
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"ls-files", "--stage", "--debug", "-z"}, okResult(index), okResult(index))
	snapshot, err := Capture(context.Background(), fake, "/tmp/repo")
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	stages := map[int]bool{}
	for _, entry := range snapshot.Index.Entries {
		if entry.Path == "README.md" {
			stages[entry.Stage] = true
		}
	}
	for _, want := range []int{0, 1, 2, 3} {
		if !stages[want] {
			t.Errorf("stages present = %v, want 0..3", stages)
		}
	}
}

// TestCompatFSMonitorBit proves the fsmonitor_valid flag bit parses
// through the production entry. Live git on this host never sets the
// bit, so the scripted conversation stands in; the bound is stated in
// the artifact, not hidden.
func TestCompatFSMonitorBit(t *testing.T) {
	t.Parallel()
	index := "100644 b6b0be997c9c8246cdd346dd7ece72140d74dee0 0\tAGENTS.md\x00" +
		"  ctime: 1:2\n  mtime: 3:4\n  dev: 5\tino: 6\n  uid: 7\tgid: 8\n  size: 6\tflags: 10000000\n"
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"ls-files", "--stage", "--debug", "-z"}, okResult(index), okResult(index))
	snapshot, err := Capture(context.Background(), fake, "/tmp/repo")
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if len(snapshot.Index.Entries) != 1 || !snapshot.Index.Entries[0].FSMonitorValid {
		t.Errorf("entries = %+v, want fsmonitor_valid", snapshot.Index.Entries)
	}
}

// TestCompatSHA256ObjectFormat proves cross-format OIDs are admitted
// when they match the repository format and refused otherwise.
func TestCompatSHA256ObjectFormat(t *testing.T) {
	t.Parallel()
	oid256 := strings.Repeat("2b", 32)
	head256 := strings.Repeat("f2", 32)
	index := "100644 " + oid256 + " 0\tREADME.md\x00" + stageProbeDebug
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"rev-parse", "--is-inside-work-tree", "--is-bare-repository", "--show-object-format", "--absolute-git-dir", "--git-common-dir", "--is-shallow-repository"},
		okResult("true\nfalse\nsha256\n/tmp/repo/.git\n/tmp/repo/.git\nfalse\n"))
	fake.Rescript([]string{"rev-parse", "--verify", "--quiet", "HEAD"}, okResult(head256+"\n"), okResult(head256+"\n"))
	fake.Rescript([]string{"ls-files", "--stage", "--debug", "-z"}, okResult(index), okResult(index))
	snapshot, err := Capture(context.Background(), fake, "/tmp/repo")
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if snapshot.ObjectFormat != "sha256" || *snapshot.Head.OID != "sha256:"+head256 {
		t.Errorf("Head = %+v, want sha256 OID", snapshot.Head)
	}
	if snapshot.Index.Entries[0].OID != "sha256:"+oid256 {
		t.Errorf("entry OID = %q", snapshot.Index.Entries[0].OID)
	}
}
