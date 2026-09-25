package gitsnap

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// Arithmetic edges are proven in both directions at the production
// entry: accept at the limit, refuse one past it. Character bounds count
// characters (runes), never bytes: every identity edge ships a multibyte
// witness proving which unit the gate measures.

// TestEdgeRemotesSixteenAccepts proves the upper remotes bound accepts
// exactly 16 through Capture.
func TestEdgeRemotesSixteenAccepts(t *testing.T) {
	t.Parallel()
	var builder strings.Builder
	for index := 0; index < 16; index++ {
		fmt.Fprintf(&builder, "r%02d\tssh://git@github.com/relux/repo%02d.git (fetch)\nr%02d\tssh://git@github.com/relux/repo%02d.git (push)\n", index, index, index, index)
	}
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"remote", "-v"}, okResult(builder.String()))
	snapshot, err := Capture(context.Background(), fake, "/tmp/repo")
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if len(snapshot.Remotes) != 16 {
		t.Errorf("Remotes = %d, want 16", len(snapshot.Remotes))
	}
}

// TestEdgeRemotesSeventeenRefuses proves one past the remotes bound is
// refused through Capture.
func TestEdgeRemotesSeventeenRefuses(t *testing.T) {
	t.Parallel()
	var builder strings.Builder
	for index := 0; index < 17; index++ {
		fmt.Fprintf(&builder, "r%02d\tssh://git@github.com/relux/repo%02d.git (fetch)\nr%02d\tssh://git@github.com/relux/repo%02d.git (push)\n", index, index, index, index)
	}
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"remote", "-v"}, okResult(builder.String()))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateRemotesRange)
}

// edgeIndex builds count sorted stage-0 entries with zero-padded names
// so byte order and path order agree.
func edgeIndex(count int) string {
	var builder strings.Builder
	for index := 0; index < count; index++ {
		fmt.Fprintf(&builder, "100644 b6b0be997c9c8246cdd346dd7ece72140d74dee0 0\tf%06d\x00", index)
		builder.WriteString(stageProbeDebug)
	}
	return builder.String()
}

// TestEdgeIndexMaxAccepts proves 65536 entries capture through Capture.
func TestEdgeIndexMaxAccepts(t *testing.T) {
	t.Parallel()
	index := edgeIndex(65536)
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"ls-files", "--stage", "--debug", "-z"}, okResult(index), okResult(index))
	snapshot, err := Capture(context.Background(), fake, "/tmp/repo")
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if snapshot.Index.EntryCount != 65536 || len(snapshot.Index.Entries) != 65536 {
		t.Errorf("EntryCount = %d", snapshot.Index.EntryCount)
	}
}

// TestEdgeIndexMaxPlusOneRefuses proves 65537 entries are refused
// through Capture with capability_unavailable: larger lists partition
// into child manifests in the manifest leaf, never here.
func TestEdgeIndexMaxPlusOneRefuses(t *testing.T) {
	t.Parallel()
	index := edgeIndex(65537)
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"ls-files", "--stage", "--debug", "-z"}, okResult(index), okResult(index))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateIndexEntriesRange)
}

// TestEdgeIdentity256CharsAccepts proves a 256-character identity
// captures.
func TestEdgeIdentity256CharsAccepts(t *testing.T) {
	t.Parallel()
	identity := "relux/" + strings.Repeat("a", 250)
	if len([]rune(identity)) != 256 {
		t.Fatalf("fixture is %d runes", len([]rune(identity)))
	}
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"remote", "-v"}, okResult("origin\tssh://git@github.com/"+identity+".git (fetch)\norigin\tssh://git@github.com/"+identity+".git (push)\n"))
	snapshot, err := Capture(context.Background(), fake, "/tmp/repo")
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if snapshot.RepositoryIdentity != identity {
		t.Errorf("RepositoryIdentity = %q", snapshot.RepositoryIdentity)
	}
}

// TestEdgeIdentity257CharsRefuses proves one character past the bound is
// refused.
func TestEdgeIdentity257CharsRefuses(t *testing.T) {
	t.Parallel()
	identity := "relux/" + strings.Repeat("a", 251)
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"remote", "-v"}, okResult("origin\tssh://git@github.com/"+identity+".git (fetch)\norigin\tssh://git@github.com/"+identity+".git (push)\n"))
	_, err := Capture(context.Background(), fake, "/tmp/repo")
	requireRefusal(t, err, GateIdentityLength)
}

// TestEdgeIdentityCountsCharactersNotBytes proves the bound measures
// characters: 256 multibyte characters (512 bytes on the wire) capture.
func TestEdgeIdentityCountsCharactersNotBytes(t *testing.T) {
	t.Parallel()
	identity := strings.Repeat("é", 256)
	if len(identity) == 256 {
		t.Fatal("fixture must exceed 256 bytes to prove the unit")
	}
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"remote", "-v"}, okResult("origin\tssh://git@github.com/"+identity+".git (fetch)\norigin\tssh://git@github.com/"+identity+".git (push)\n"))
	snapshot, err := Capture(context.Background(), fake, "/tmp/repo")
	if err != nil {
		t.Fatalf("Capture() error = %v, want character-counted accept", err)
	}
	if snapshot.RepositoryIdentity != identity {
		t.Errorf("RepositoryIdentity mismatch")
	}
}

// TestEdgeStageThreeAccepts proves stage 3 captures: the top of the
// closed 0..3 domain.
func TestEdgeStageThreeAccepts(t *testing.T) {
	t.Parallel()
	index := stageProbeIndex("100644", "19d9cc8584ac2c7dcf57d2680375e80f099dc481", "3", "README.md")
	fake := scriptBaseline(newScriptedRunner())
	fake.Rescript([]string{"ls-files", "--stage", "--debug", "-z"}, okResult(index), okResult(index))
	snapshot, err := Capture(context.Background(), fake, "/tmp/repo")
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if len(snapshot.Index.Entries) != 1 || snapshot.Index.Entries[0].Stage != 3 {
		t.Errorf("entries = %+v", snapshot.Index.Entries)
	}
}

// TestEdgeEntryCountIsConstructionInvariant proves EntryCount always
// equals len(Entries): both are set from the one parsed length, so a
// count mismatch cannot be constructed here. The wire-side mismatch
// refusal (TM-GIT-N2 count arm) belongs to internal/canonicaljson,
// pinned by the "index entry count" case of
// TestTransferManifestNestedValueConstraintsReachBothIdentityEntries;
// this test guards the construction against drift.
func TestEdgeEntryCountIsConstructionInvariant(t *testing.T) {
	t.Parallel()
	fake := scriptBaseline(newScriptedRunner())
	snapshot, err := Capture(context.Background(), fake, "/tmp/repo")
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if snapshot.Index.EntryCount != len(snapshot.Index.Entries) {
		t.Errorf("EntryCount = %d, len = %d", snapshot.Index.EntryCount, len(snapshot.Index.Entries))
	}
}
