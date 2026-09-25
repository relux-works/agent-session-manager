package gitsnap

import (
	"context"
	"strings"
	"testing"
)

// The WS-GIT-ROUNDTRIP-1 language-neutral corpus of Section 10.4 is the
// exact contract fixture for this leaf's index scope. The parent and
// child ls-files lines below are quoted verbatim from the specification;
// the expected logical entries are the normative wire entries of the
// workspace-group root fixture. The test drives them through Capture so
// the production parser, not a copy of it, proves the cross-check. Pack,
// inventory, and manifest assembly belong to sibling leaves and are
// asserted here only as out-of-scope bounds.

// Normative parent superproject index lines (Section 10.4,
// parent_index_entries).
var fixtureParentIndexLines = []string{
	"100644 5461fe036f6cf55f98d75ee651c2b4cc13a80c66 0\t.gitmodules",
	"100644 b6b0be997c9c8246cdd346dd7ece72140d74dee0 0\tAGENTS.md",
	"100644 19d9cc8584ac2c7dcf57d2680375e80f099dc481 0\tREADME.md",
	"160000 25eec72bdd91287a7d68f206907a859b5a7b5524 0\tvendor/lib",
}

// Normative child submodule index lines (Section 10.4,
// child_index_entries).
var fixtureChildIndexLines = []string{
	"100644 a69c0feac9815fe47cecb849931d858109a5a0c9 0\tREADME.md",
}

// fixtureDebugBytes renders ls-files lines as --debug -z output with
// clear flags.
func fixtureDebugBytes(lines []string) string {
	var builder strings.Builder
	for _, line := range lines {
		builder.WriteString(line + "\x00" + stageProbeDebug)
	}
	return builder.String()
}

// TestFixtureParentIndexLinesMatchWireEntries proves the parser turns
// the exact parent corpus lines into the normative wire entries: paths,
// stages, modes (33188 file, 57344 gitlink), and OIDs with the sha1:
// prefix the wire requires.
func TestFixtureParentIndexLinesMatchWireEntries(t *testing.T) {
	t.Parallel()
	index := fixtureDebugBytes(fixtureParentIndexLines)
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"ls-files", "--stage", "--debug", "-z"}, okResult(index), okResult(index))
	snapshot, err := Capture(context.Background(), fake, "/tmp/repo")
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	want := []IndexEntry{
		{Path: ".gitmodules", Stage: 0, Mode: 33188, OID: "sha1:5461fe036f6cf55f98d75ee651c2b4cc13a80c66"},
		{Path: "AGENTS.md", Stage: 0, Mode: 33188, OID: "sha1:b6b0be997c9c8246cdd346dd7ece72140d74dee0"},
		{Path: "README.md", Stage: 0, Mode: 33188, OID: "sha1:19d9cc8584ac2c7dcf57d2680375e80f099dc481"},
		{Path: "vendor/lib", Stage: 0, Mode: 57344, OID: "sha1:25eec72bdd91287a7d68f206907a859b5a7b5524"},
	}
	if len(snapshot.Index.Entries) != len(want) {
		t.Fatalf("entries = %+v, want %+v", snapshot.Index.Entries, want)
	}
	for i, entry := range want {
		got := snapshot.Index.Entries[i]
		if got != entry {
			t.Errorf("entry %d = %+v, want %+v", i, got, entry)
		}
	}
	// The staged README blob named in fixture prose is exactly the
	// stage-0 README index OID: staged and unstaged stay distinct.
	if snapshot.Index.Entries[2].OID != "sha1:19d9cc8584ac2c7dcf57d2680375e80f099dc481" {
		t.Errorf("staged README OID = %q", snapshot.Index.Entries[2].OID)
	}
}

// TestFixtureChildIndexLineMatchesWireEntry proves the child corpus
// line parses to the normative single child entry.
func TestFixtureChildIndexLineMatchesWireEntry(t *testing.T) {
	t.Parallel()
	index := fixtureDebugBytes(fixtureChildIndexLines)
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"ls-files", "--stage", "--debug", "-z"}, okResult(index), okResult(index))
	snapshot, err := Capture(context.Background(), fake, "/tmp/repo")
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	want := IndexEntry{Path: "README.md", Stage: 0, Mode: 33188, OID: "sha1:a69c0feac9815fe47cecb849931d858109a5a0c9"}
	if len(snapshot.Index.Entries) != 1 || snapshot.Index.Entries[0] != want {
		t.Errorf("entries = %+v, want [%+v]", snapshot.Index.Entries, want)
	}
}

// TestFixturePackSeparationIsOutOfScope states the pack-boundary bound
// the corpus pins: the parent inventory intentionally omits the child
// commit. Pack construction and isolated-database verification belong to
// the object-pack sibling leaf; this leaf records index OIDs only and
// asserts no pack claim here.
func TestFixturePackSeparationIsOutOfScope(t *testing.T) {
	t.Parallel()
	fake := scriptBaseline(newScriptedRunner())
	snapshot, err := Capture(context.Background(), fake, "/tmp/repo")
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	for _, entry := range snapshot.Index.Entries {
		if !strings.HasPrefix(entry.OID, "sha1:") {
			t.Errorf("entry OID = %q, want sha1: prefix matching the corpus object format", entry.OID)
		}
	}
}
