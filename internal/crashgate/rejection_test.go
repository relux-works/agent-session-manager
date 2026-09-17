package crashgate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/matjournal"
)

// parkedMustFailClosed asserts the parked refusal: the blocking
// reason is durable as recovery.json and last_error, the phase is
// frozen, no second process or manager was allocated (every
// operation ID and manager reference is unchanged), and input stays
// blocked on the unchanged phase.
func parkedMustFailClosed(t *testing.T, dataDir, beforePhase string, evidence matjournal.Evidence, what string) {
	t.Helper()
	if evidence.Selected != matjournal.OutcomeRecoverableParked {
		t.Fatalf("%s selected = %q, want recoverable_parked_state", what, evidence.Selected)
	}
	raw, err := os.ReadFile(filepath.Join(dataDir, "materializations", testMatID, "recovery.json"))
	if err != nil {
		t.Fatalf("%s recovery.json unreadable: %v", what, err)
	}
	var durable matjournal.Evidence
	if err := json.Unmarshal(raw, &durable); err != nil {
		t.Fatalf("%s recovery.json undecodable: %v", what, err)
	}
	if durable.Selected != matjournal.OutcomeRecoverableParked || durable.Reason == "" || durable.Remediation == "" {
		t.Fatalf("%s durable evidence = %+v, want the parked refusal with reason and remediation", what, durable)
	}
	if durable.Boundary != evidence.Boundary || durable.Reason != evidence.Reason {
		t.Fatalf("%s durable evidence diverges from the returned evidence", what)
	}
	loaded, _, err := mustOpenJournal(t, dataDir).Get(testMatID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Phase != beforePhase {
		t.Fatalf("%s phase = %q, want the frozen %q", what, loaded.Phase, beforePhase)
	}
	if len(loaded.LastError) == 0 || string(loaded.LastError) == "null" {
		t.Fatalf("%s last_error is empty, want the blocking reason recorded", what)
	}
	if loaded.PrepareOperationID != testPrepareOp {
		t.Fatalf("%s prepare operation drifted to %q", what, loaded.PrepareOperationID)
	}
}

// mustOpenJournal opens a read handle over a data directory.
func mustOpenJournal(t *testing.T, dataDir string) *matjournal.Store {
	t.Helper()
	store, err := matjournal.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store.Now = fixedClock
	return store
}

// TestCrashGateRejections proves the gate refuses every forbidden
// recovery as any successful outcome: two live authorities, an
// unfenced external continuation, and every substitution shape
// (new-session launch, fresh native handle, relabelled blank
// state, different realm, moved lease) fail closed to
// recoverable_parked_state through the production Recover entry,
// never to safe_retry or explicit_rollback.
func TestCrashGateRejections(t *testing.T) {
	stagingComposite := func(t *testing.T) (string, string) {
		store, dataDir := journalDirs(t)
		mustCreate(t, store, compositeInputs())
		if _, err := store.UpdateProvider(testMatID, preparedProvider()); err != nil {
			t.Fatal(err)
		}
		return dataDir, matjournal.PhaseStaging
	}
	committingAdopted := func(t *testing.T) (string, string) {
		store, dataDir := activationSetup(t)
		if _, err := store.UpdateTaskBoard(testMatID, adoptedBoard()); err != nil {
			t.Fatal(err)
		}
		return dataDir, matjournal.PhaseCommitting
	}
	freshSession := "tbm:manager:0198f4c8-9f60-7a11-8b3c-9999999999ab:" + testLeaseID
	freshHandle := "tbm:manager:0198f4c8-7a10-7b22-8b3c-2234567890ac:" + testLeaseID
	otherRealm := "tbm:manager:0198f4c8-7a10-7b22-8b3c-2234567890ab:" + testLeaseB
	cases := []struct {
		name     string
		setup    func(t *testing.T) (string, string)
		boundary string
		mutate   func(*matjournal.RecoveryInput)
		reason   string
	}{
		{"two_live_authorities", stagingComposite, matjournal.BoundaryAfterPrepareOp, func(input *matjournal.RecoveryInput) {
			input.Provider.State = matjournal.ProviderCommitted
			input.Bridge.Live = true
			input.Bridge.BindingMatches = false
		}, "two live authorities"},
		{"unfenced_continuation", committingAdopted, matjournal.BoundaryAfterActivation, func(input *matjournal.RecoveryInput) {
			input.Bridge.State = matjournal.BoardAdopted
			input.Bridge.Live = true
			input.Bridge.LeaseActive = false
			input.Bridge.BindingMatches = true
			input.Bridge.ManagerMatches = true
			input.NativeKnown = true
			input.NativeBefore, input.NativeAfter = testNative, testNative
		}, "unfenced continuation"},
		{"unfenced_state_liveness", committingAdopted, matjournal.BoundaryAfterActivation, func(input *matjournal.RecoveryInput) {
			input.Bridge.State = "running"
			input.Bridge.Live = false
			input.Bridge.LeaseActive = false
			input.Bridge.BindingMatches = true
			input.Bridge.ManagerMatches = true
			input.NativeKnown = true
			input.NativeBefore, input.NativeAfter = testNative, testNative
		}, "unfenced continuation"},
		{"new_session_launch", committingAdopted, matjournal.BoundaryAfterActivation, func(input *matjournal.RecoveryInput) {
			input.Bridge.State = matjournal.BoardAdopted
			input.Bridge.LeaseActive = true
			input.Bridge.BindingMatches = true
			input.Bridge.ManagerMatches = true
			input.NativeKnown = true
			input.NativeBefore, input.NativeAfter = testNative, freshSession
		}, "substitution"},
		{"fresh_native_handle", committingAdopted, matjournal.BoundaryAfterActivation, func(input *matjournal.RecoveryInput) {
			input.Bridge.State = matjournal.BoardAdopted
			input.Bridge.LeaseActive = true
			input.Bridge.BindingMatches = true
			input.Bridge.ManagerMatches = true
			input.NativeKnown = true
			input.NativeBefore, input.NativeAfter = testNative, freshHandle
		}, "substitution"},
		{"blank_relabel", committingAdopted, matjournal.BoundaryAfterActivation, func(input *matjournal.RecoveryInput) {
			input.Bridge.State = matjournal.BoardAdopted
			input.Bridge.LeaseActive = true
			input.Bridge.BindingMatches = true
			input.Bridge.ManagerMatches = true
			input.NativeKnown = true
			input.NativeBefore, input.NativeAfter = testNative, ""
		}, "substitution"},
		{"different_realm", committingAdopted, matjournal.BoundaryAfterActivation, func(input *matjournal.RecoveryInput) {
			input.Bridge.State = matjournal.BoardAdopted
			input.Bridge.LeaseActive = true
			input.Bridge.BindingMatches = true
			input.Bridge.ManagerMatches = true
			input.NativeKnown = true
			input.NativeBefore, input.NativeAfter = testNative, otherRealm
		}, "substitution"},
		{"lease_moved_forward", stagingComposite, matjournal.BoundaryAfterPrepareOp, func(input *matjournal.RecoveryInput) {
			input.LeaseAfter = matjournal.LeaseIdentity{Epoch: 5, ID: testLeaseID}
		}, "winning lease moved"},
		{"lease_moved_backward", stagingComposite, matjournal.BoundaryAfterPrepareOp, func(input *matjournal.RecoveryInput) {
			input.LeaseAfter = matjournal.LeaseIdentity{Epoch: 3, ID: testLeaseID}
		}, "winning lease moved"},
		{"lease_moved_id", stagingComposite, matjournal.BoundaryAfterPrepareOp, func(input *matjournal.RecoveryInput) {
			input.LeaseAfter = matjournal.LeaseIdentity{Epoch: 4, ID: testLeaseB}
		}, "winning lease moved"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dataDir, phase := tc.setup(t)
			restarted := reopenJournal(t, dataDir)
			input := baseRecoveryInput()
			input.Boundary = tc.boundary
			input.Path = matjournal.PathOwnerResume
			tc.mutate(&input)
			outcome, evidence, err := restarted.Recover(testMatID, input)
			if err != nil {
				t.Fatalf("Recover error = %v", err)
			}
			mustOutcome(t, outcome, matjournal.OutcomeRecoverableParked, "Recover("+tc.name+")")
			mustJournalEvidenceComplete(t, evidence, "Recover("+tc.name+")")
			if !strings.Contains(evidence.Reason, tc.reason) {
				t.Fatalf("reason = %q, want %q", evidence.Reason, tc.reason)
			}
			if strings.Contains(tc.reason, "substitution") && !strings.Contains(evidence.Remediation, "fresh handle") {
				t.Fatalf("remediation = %q, want the fresh-handle prohibition", evidence.Remediation)
			}
			parkedMustFailClosed(t, dataDir, phase, evidence, "Recover("+tc.name+")")
			loaded, _, err := restarted.Get(testMatID)
			if err != nil {
				t.Fatal(err)
			}
			if tc.setup != nil && loaded.TaskBoard != nil && loaded.TaskBoard.State == matjournal.BoardAdopted {
				if loaded.TaskBoard.ManagerSessionRef == nil || *loaded.TaskBoard.ManagerSessionRef != "tbm:manager:0198f4c8-7a10-7b22-8b3c-2234567890ab" {
					t.Fatalf("manager reference moved: %+v", loaded.TaskBoard.ManagerSessionRef)
				}
			}
		})
	}
}
