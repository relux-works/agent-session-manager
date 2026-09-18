package matjournal

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestRPCBlindRetryAfterCommitNeverDoubles is the lost-response recovery
// proof for the Mesh RPC materialize.status entry (Section 11.3): the
// status read locates the one journal by caller-known materialization ID
// and returns the current reconciled durable phase, while a blind retry of
// the prepare after a committed effect replays the recorded receipt instead
// of doubling the mutation. Driven through the production Create, Get, and
// Transition entries.
func TestRPCBlindRetryAfterCommitNeverDoubles(t *testing.T) {
	store := openTestStore(t)
	setupPhase(t, store, PhaseCommitted)

	current, stored, err := store.Get(testMatID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Phase != PhaseCommitted {
		t.Fatalf("Get phase = %q, want committed (the true durable phase, not a stale observation)", current.Phase)
	}
	entries, err := os.ReadDir(store.root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("journal directories = %d, want exactly one", len(entries))
	}

	// The client lost the prepare result and retries blindly with the
	// identical canonical body: the recorded receipt replays.
	again, raw, err := store.Create(testInputs())
	if err != nil {
		t.Fatalf("blind retry Create error = %v", err)
	}
	if again.MaterializationID != testMatID || again.Phase != PhaseCommitted {
		t.Fatalf("blind retry ref = %+v, want the committed receipt", again)
	}
	if !bytes.Equal(raw, stored) {
		t.Fatal("blind retry replayed bytes differ from the recorded journal")
	}
	entries, err = os.ReadDir(store.root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("blind retry left %d journal directories, want exactly one (no second journal)", len(entries))
	}
	after, _, err := store.Get(testMatID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Phase != PhaseCommitted {
		t.Fatalf("Get phase after retry = %q, want committed", after.Phase)
	}
}

// TestRPCChangedBodyAfterCommitRefusesMismatch is the committed-phase half
// of the changed-body rule (Section 11.3): the spec's "a changed canonical
// body is idempotency_mismatch" carries no phase qualifier, so a retry that
// moves the body after a committed effect refuses with the literal
// idempotency_mismatch and writes nothing, through the production Create
// entry. TestCreateReplayAndConflict pins the staging member of the class;
// this test pins the committed member.
func TestRPCChangedBodyAfterCommitRefusesMismatch(t *testing.T) {
	store := openTestStore(t)
	setupPhase(t, store, PhaseCommitted)

	moved := testInputs()
	moved.RequestBody = []byte(`{"materialization_id":"` + testMatID + `","operation_id":"` + testPrepareOp + `","plan_id":"` + testPriorID + `"}`)
	_, _, err := store.Create(moved)
	mustConflict(t, err, "Create(moved body after commit)")
	if !strings.Contains(err.Error(), "idempotency_mismatch") {
		t.Fatalf("error = %v, want idempotency_mismatch named", err)
	}

	entries, err := os.ReadDir(store.root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("journal directories = %d, want exactly one retained", len(entries))
	}
	after, _, err := store.Get(testMatID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Phase != PhaseCommitted {
		t.Fatalf("Get phase after refused retry = %q, want committed", after.Phase)
	}
}
