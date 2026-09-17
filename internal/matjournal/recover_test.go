package matjournal

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testLease returns the winning-lease observation.
func testLease() LeaseIdentity {
	return LeaseIdentity{Epoch: 4, ID: testLeaseID}
}

// baseRecoveryInput returns the consistent probe closure: prepared
// provider status with agreement, no bridge effect, no marker, a
// healthy host, and the unchanged winning lease.
func baseRecoveryInput() RecoveryInput {
	return RecoveryInput{
		Boundary:    BoundaryAfterPrepare,
		Path:        PathOwnerResume,
		Provider:    ProviderProbe{State: ProviderPrepared, IDsMatch: true, PlanMatches: true, TokenMatches: true},
		Bridge:      BridgeProbe{State: BoardNotStarted},
		Marker:      MarkerAbsent,
		Host:        HostOK,
		LeaseKnown:  true,
		LeaseBefore: testLease(),
		LeaseAfter:  testLease(),
	}
}

// mustOutcome asserts the exact outcome and the never-fourth-outcome
// rule.
func mustOutcome(t *testing.T, got, want Outcome, what string) {
	t.Helper()
	switch got {
	case OutcomeSafeRetry, OutcomeExplicitRollback, OutcomeRecoverableParked:
	default:
		t.Fatalf("%s outcome = %q, want one of the three Section 13.13 outcomes", what, got)
	}
	if got != want {
		t.Fatalf("%s outcome = %q, want %q", what, got, want)
	}
}

// mustEvidenceComplete asserts the Section 13.13 evidence record names
// every required fact.
func mustEvidenceComplete(t *testing.T, evidence Evidence, what string) {
	t.Helper()
	if evidence.Boundary == "" || evidence.Path == "" || len(evidence.OperationIDs) == 0 {
		t.Fatalf("%s evidence lacks boundary, path, or operation IDs: %+v", what, evidence)
	}
	if evidence.PhaseBefore == "" || evidence.PhaseAfter == "" {
		t.Fatalf("%s evidence lacks pre/post durable facts: %+v", what, evidence)
	}
	if evidence.Reason == "" || evidence.Remediation == "" || evidence.DecidedAt == "" {
		t.Fatalf("%s evidence lacks reason, remediation, or decided_at: %+v", what, evidence)
	}
	if evidence.LeaseBefore.Epoch == 0 || evidence.LeaseAfter.Epoch == 0 {
		t.Fatalf("%s evidence lacks the winning lease before/after: %+v", what, evidence)
	}
}

func TestRecoverBeforeCreate(t *testing.T) {
	store := openTestStore(t)
	input := baseRecoveryInput()
	input.Boundary = BoundaryBeforeCreate
	input.Provider.State = ProviderUnknown
	outcome, evidence, err := store.Recover(testMatID, input)
	if err != nil {
		t.Fatalf("Recover error = %v", err)
	}
	mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(CR-MAT-01)")
	if evidence.PhaseBefore != "" || evidence.ReceiptPresent {
		t.Fatalf("evidence = %+v, want no durable facts", evidence)
	}
	// Any other boundary with no journal is absence, not a
	// classification.
	input.Boundary = BoundaryAfterPrepare
	_, _, err = store.Recover(testMatID, input)
	if !errors.Is(err, ErrUnknownJournal) {
		t.Fatalf("Recover(absent) = %v, want ErrUnknownJournal", err)
	}
}

func TestRecoverRefusesBadBoundary(t *testing.T) {
	store := openTestStore(t)
	mustCreate(t, store, testInputs())
	cases := []struct {
		name   string
		mutate func(*RecoveryInput)
	}{
		{"bad_boundary", func(i *RecoveryInput) { i.Boundary = "CR-MAT-09" }},
		{"bad_path", func(i *RecoveryInput) { i.Path = "migration" }},
		{"bad_provider", func(i *RecoveryInput) { i.Provider.State = "preparing" }},
		{"bad_bridge", func(i *RecoveryInput) { i.Bridge.State = "launched" }},
		{"bad_marker", func(i *RecoveryInput) { i.Marker = "present" }},
		{"bad_host", func(i *RecoveryInput) { i.Host = "on_fire" }},
		{"bad_lease_epoch", func(i *RecoveryInput) { i.LeaseAfter.Epoch = 0 }},
		{"bad_lease_id", func(i *RecoveryInput) { i.LeaseAfter.ID = "nope" }},
		{"bad_materialization", func(i *RecoveryInput) {}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := baseRecoveryInput()
			tc.mutate(&input)
			id := testMatID
			if tc.name == "bad_materialization" {
				id = "not-a-uuid"
			}
			_, _, err := store.Recover(id, input)
			mustInvalid(t, err, "Recover(boundary)")
		})
	}
}

func TestRecoverTornJournalParks(t *testing.T) {
	store := openTestStore(t)
	mustCreate(t, store, testInputs())
	matDir := filepath.Join(store.root, testMatID)
	if err := os.WriteFile(filepath.Join(matDir, "journal.json"), []byte(`{"torn":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	input := baseRecoveryInput()
	input.Boundary = BoundaryAfterTransfer
	outcome, evidence, err := store.Recover(testMatID, input)
	if err != nil {
		t.Fatalf("Recover error = %v", err)
	}
	mustOutcome(t, outcome, OutcomeRecoverableParked, "Recover(torn)")
	if !strings.Contains(evidence.Reason, "quarantined") {
		t.Fatalf("reason = %q", evidence.Reason)
	}
	if _, err := os.Stat(filepath.Join(matDir, "recovery.json")); err != nil {
		t.Fatalf("recovery.json: %v", err)
	}
}

func TestRecoverGatesPark(t *testing.T) {
	setup := func(t *testing.T, mode string) *Store {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		if _, err := store.UpdateProvider(testMatID, testProvider()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardOpened()); err != nil {
			t.Fatal(err)
		}
		if mode == "staging" {
			return store
		}
		// Prepared and adopted journals walk the legal phase
		// path: adoption happens during finalize.
		providerRoot := testTxRoot + "/backups/codex_sessions"
		if _, err := store.UpdateAuthority(testMatID, "codex_sessions", AuthorityState{
			RootPath: testRootProvider, CompletedSequences: []uint64{2},
			RollbackRoot: &providerRoot, State: AuthorityPrepared,
		}); err != nil {
			t.Fatal(err)
		}
		workspaceRoot := "/home/ivan/.local/state/ax/materializations/" + testMatID + "/workspace_relux"
		if _, err := store.UpdateAuthority(testMatID, "workspace_relux", AuthorityState{
			RootPath: testRootWorkspace, CompletedSequences: []uint64{1},
			RollbackRoot: &workspaceRoot, State: AuthorityPrepared,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, PhaseValidating, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, PhasePrepared, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		if mode == "prepared" {
			return store
		}
		if _, err := store.Transition(testMatID, PhaseCommitting, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardAdopted()); err != nil {
			t.Fatal(err)
		}
		return store
	}
	native := "provider-identity:" + testCheckpointID
	cases := []struct {
		name   string
		mode   string
		mutate func(*RecoveryInput)
		want   string
	}{
		{"lease_moved", "staging", func(i *RecoveryInput) {
			i.LeaseAfter = LeaseIdentity{Epoch: 5, ID: testLeaseB}
		}, "lease"},
		{"native_missing", "adopted", func(i *RecoveryInput) {
			i.NativeKnown = false
		}, "native"},
		{"native_changed", "adopted", func(i *RecoveryInput) {
			i.NativeKnown = true
			i.NativeBefore = native
			i.NativeAfter = "provider-identity:" + testPriorID
		}, "substitution"},
		{"two_live_authorities", "staging", func(i *RecoveryInput) {
			i.Provider.State = ProviderCommitted
			i.Bridge.Live = true
			i.Bridge.BindingMatches = false
		}, "two live authorities"},
		{"unfenced_continuation", "adopted", func(i *RecoveryInput) {
			i.NativeKnown = true
			i.NativeBefore, i.NativeAfter = native, native
			i.Bridge.Live = true
			i.Bridge.LeaseActive = false
		}, "unfenced"},
		{"disk_full", "staging", func(i *RecoveryInput) { i.Host = HostDiskFull }, "disk full"},
		{"rename_blocked", "staging", func(i *RecoveryInput) { i.Host = HostRenameBlocked }, "rename blocked"},
		{"marker_mismatch", "staging", func(i *RecoveryInput) { i.Marker = MarkerMismatch }, "MJ-CRASH-MARKER-MISMATCH"},
		{"marker_invalid", "staging", func(i *RecoveryInput) { i.Marker = MarkerInvalid }, "invalid"},
		{"provider_unknown", "prepared", func(i *RecoveryInput) {
			i.Provider.State = ProviderUnknown
			i.Boundary = BoundaryAfterActivation
		}, "unknown"},
		{"bridge_unknown", "adopted", func(i *RecoveryInput) {
			i.Bridge.State = ProviderUnknown
		}, "unknown"},
		{"token_mismatch", "staging", func(i *RecoveryInput) {
			i.Provider.TokenMatches = false
		}, "token"},
		{"binding_mismatch", "adopted", func(i *RecoveryInput) {
			i.NativeKnown = true
			i.NativeBefore, i.NativeAfter = native, native
			i.Bridge.State = BoardAdopted
			i.Bridge.Live = true
			i.Bridge.LeaseActive = true
			i.Bridge.BindingMatches = false
			i.Bridge.ManagerMatches = true
		}, "TB-TXN-BINDING-MISMATCH"},
		{"lease_unknown", "staging", func(i *RecoveryInput) { i.LeaseKnown = false }, "lease"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := setup(t, tc.mode)
			input := baseRecoveryInput()
			input.Boundary = BoundaryAfterActivation
			if tc.name != "native_missing" && tc.name != "native_changed" {
				input.NativeKnown = true
				input.NativeBefore, input.NativeAfter = native, native
			}
			tc.mutate(&input)
			outcome, evidence, err := store.Recover(testMatID, input)
			if err != nil {
				t.Fatalf("Recover error = %v", err)
			}
			mustOutcome(t, outcome, OutcomeRecoverableParked, "Recover(gate)")
			mustEvidenceComplete(t, evidence, "Recover(gate)")
			if !strings.Contains(evidence.Reason, tc.want) {
				t.Fatalf("reason = %q, want %q", evidence.Reason, tc.want)
			}
			// The parked assessment records the blocking reason
			// as last_error and persists the evidence.
			loaded, _, err := store.Get(testMatID)
			if err != nil {
				t.Fatal(err)
			}
			if len(loaded.LastError) == 0 {
				t.Fatalf("parked journal carries no last_error")
			}
			if _, err := os.Stat(filepath.Join(store.root, testMatID, "recovery.json")); err != nil {
				t.Fatalf("recovery.json: %v", err)
			}
		})
	}
}

func TestRecoverVacuousUnknownProbesResume(t *testing.T) {
	store := openTestStore(t)
	mustCreate(t, store, testInputsComposite())
	if _, err := store.UpdateProvider(testMatID, testProvider()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateTaskBoard(testMatID, testBoardOpened()); err != nil {
		t.Fatal(err)
	}
	// Recorded capabilities at staging stand on their own: the
	// commit is impossible there, so unknown probes are vacuous and
	// the staging transaction resumes.
	input := baseRecoveryInput()
	input.Boundary = BoundaryAfterOpen
	input.Provider.State = ProviderUnknown
	input.Bridge.State = ProviderUnknown
	outcome, _, err := store.Recover(testMatID, input)
	if err != nil {
		t.Fatalf("Recover error = %v", err)
	}
	mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(vacuous unknown)")
}

func TestRecoverContradictoryProgressionParks(t *testing.T) {
	store := openTestStore(t)
	mustCreate(t, store, testInputsComposite())
	// An adopted bridge recorded at staging contradicts the
	// progression: activation happens during finalize. The store
	// admits the sub-state write (the sub-table is phase-agnostic),
	// and recovery quarantines the contradiction.
	if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateTaskBoard(testMatID, testBoardOpened()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateTaskBoard(testMatID, testBoardAdopted()); err != nil {
		t.Fatal(err)
	}
	input := baseRecoveryInput()
	input.Boundary = BoundaryAfterActivation
	input.Bridge.State = BoardAdopted
	input.Bridge.LeaseActive = true
	input.Bridge.BindingMatches = true
	input.Bridge.ManagerMatches = true
	input.NativeKnown = true
	input.NativeBefore = "tbm:manager:x:" + testLeaseID
	input.NativeAfter = input.NativeBefore
	outcome, evidence, err := store.Recover(testMatID, input)
	if err != nil {
		t.Fatalf("Recover error = %v", err)
	}
	mustOutcome(t, outcome, OutcomeRecoverableParked, "Recover(contradictory)")
	if !strings.Contains(evidence.Reason, "contradicts") {
		t.Fatalf("reason = %q", evidence.Reason)
	}
}

func TestRecoverTerminalReplays(t *testing.T) {
	for _, phase := range []string{PhaseCommitted, PhaseRolledBack, PhaseFailed} {
		t.Run(phase, func(t *testing.T) {
			store := openTestStore(t)
			setupPhase(t, store, phase)
			input := baseRecoveryInput()
			input.Boundary = BoundaryAfterFinalize
			input.Provider.State = ProviderUnknown
			outcome, evidence, err := store.Recover(testMatID, input)
			if err != nil {
				t.Fatalf("Recover error = %v", err)
			}
			mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(terminal)")
			mustEvidenceComplete(t, evidence, "Recover(terminal)")
			if evidence.PhaseBefore != phase || evidence.PhaseAfter != phase {
				t.Fatalf("evidence phases = %q to %q", evidence.PhaseBefore, evidence.PhaseAfter)
			}
		})
	}
	t.Run("terminal_parks_frozen", func(t *testing.T) {
		store := openTestStore(t)
		setupPhase(t, store, PhaseCommitted)
		before, _, err := store.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		// Two live authorities park even a committed journal,
		// but the terminal document stays frozen.
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterFinalize
		input.Provider.State = ProviderCommitted
		input.Bridge.Live = true
		input.Bridge.BindingMatches = false
		outcome, _, err := store.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeRecoverableParked, "Recover(terminal)")
		after, _, err := store.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		if string(before.LastError) != string(after.LastError) || before.UpdatedAt != after.UpdatedAt {
			t.Fatalf("terminal journal mutated by park")
		}
		if _, err := os.Stat(filepath.Join(store.root, testMatID, "recovery.json")); err != nil {
			t.Fatalf("recovery.json: %v", err)
		}
	})
}

func TestRecoverRollbackRequired(t *testing.T) {
	t.Run("bridge_failed_before_lease", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		if _, err := store.UpdateProvider(testMatID, testProvider()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterPrepareOp
		input.Bridge.State = BoardFailed
		input.Bridge.Failed = true
		outcome, evidence, err := store.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeExplicitRollback, "Recover(bridge failed)")
		mustEvidenceComplete(t, evidence, "Recover(bridge failed)")
		loaded, _, err := store.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.Phase != PhaseRolledBack || loaded.TaskBoard.State != BoardRolledBack {
			t.Fatalf("phase = %q bridge = %q", loaded.Phase, loaded.TaskBoard.State)
		}
	})
	t.Run("token_expired_before_lease", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		if _, err := store.UpdateProvider(testMatID, testProvider()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterOpen
		input.Bridge.State = BoardImported
		input.Bridge.TokenExpired = true
		outcome, _, err := store.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeExplicitRollback, "Recover(token expired)")
	})
	t.Run("provider_rolled_back_under_prepared", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		mustConvergePrepared(t, store)
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterOpen
		input.Provider.State = ProviderRolledBack
		input.Bridge.State = BoardOpened
		outcome, _, err := store.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeExplicitRollback, "Recover(provider rolled back)")
	})
	t.Run("rolling_back_retries", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		if _, err := store.UpdateProvider(testMatID, testProvider()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, PhaseRollingBack, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterFinalize
		input.Provider.State = ProviderPrepared
		input.Bridge.State = BoardNotStarted
		outcome, _, err := store.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeExplicitRollback, "Recover(rolling_back retry)")
	})
	t.Run("rolling_back_closes", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		if _, err := store.UpdateProvider(testMatID, testProvider()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, PhaseRollingBack, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterFinalize
		input.Provider.State = ProviderRolledBack
		input.Bridge.State = BoardNotStarted
		input.PredecessorsRestored = true
		outcome, _, err := store.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeExplicitRollback, "Recover(rolling_back close)")
		loaded, _, err := store.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.Phase != PhaseRolledBack {
			t.Fatalf("phase = %q", loaded.Phase)
		}
	})
	t.Run("rolling_back_unproven_parks", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		if _, err := store.UpdateProvider(testMatID, testProvider()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, PhaseRollingBack, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterFinalize
		input.Provider.State = ProviderRolledBack
		input.Bridge.State = BoardNotStarted
		outcome, _, err := store.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeRecoverableParked, "Recover(unproven restoration)")
	})
}

func TestRecoverCompletesCommit(t *testing.T) {
	t.Run("provider_workspace", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputs())
		backup := "/home/ivan/.local/state/ax/materializations/" + testMatID + "/workspace_relux"
		if _, err := store.UpdateAuthority(testMatID, "workspace_relux", AuthorityState{
			RootPath: testRootWorkspace, CompletedSequences: []uint64{1},
			RollbackRoot: &backup, State: AuthorityPrepared,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateAuthority(testMatID, "workspace_relux", AuthorityState{
			RootPath: testRootWorkspace, CompletedSequences: []uint64{1}, State: AuthorityCommitted,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, PhaseValidating, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, PhasePrepared, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, PhaseCommitting, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		// MJ-CRASH-COMMIT-LOST: status committed plus the exact
		// destination marker commits the journal.
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterFinalize
		input.Provider.State = ProviderUnknown
		input.Marker = MarkerValidMatch
		input.MarkerBytes = testMarker(t)
		outcome, evidence, err := store.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(commit)")
		mustEvidenceComplete(t, evidence, "Recover(commit)")
		loaded, _, err := store.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.Phase != PhaseCommitted || loaded.DestinationMarkerID == nil {
			t.Fatalf("phase = %q marker = %+v", loaded.Phase, loaded.DestinationMarkerID)
		}
	})
	t.Run("marker_contradicts_parks", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputs())
		backup := "/home/ivan/.local/state/ax/materializations/" + testMatID + "/workspace_relux"
		if _, err := store.UpdateAuthority(testMatID, "workspace_relux", AuthorityState{
			RootPath: testRootWorkspace, CompletedSequences: []uint64{1},
			RollbackRoot: &backup, State: AuthorityPrepared,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateAuthority(testMatID, "workspace_relux", AuthorityState{
			RootPath: testRootWorkspace, CompletedSequences: []uint64{1}, State: AuthorityCommitted,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, PhaseValidating, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, PhasePrepared, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, PhaseCommitting, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		// The probe claims a valid match but the bytes fail: the
		// contradiction parks.
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterFinalize
		input.Provider.State = ProviderUnknown
		input.Marker = MarkerValidMatch
		input.MarkerBytes = []byte(`{"torn":true}`)
		outcome, _, err := store.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeRecoverableParked, "Recover(marker contradiction)")
	})
	t.Run("marker_absent_converged_parks", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputs())
		backup := "/home/ivan/.local/state/ax/materializations/" + testMatID + "/workspace_relux"
		if _, err := store.UpdateAuthority(testMatID, "workspace_relux", AuthorityState{
			RootPath: testRootWorkspace, CompletedSequences: []uint64{1},
			RollbackRoot: &backup, State: AuthorityPrepared,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateAuthority(testMatID, "workspace_relux", AuthorityState{
			RootPath: testRootWorkspace, CompletedSequences: []uint64{1}, State: AuthorityCommitted,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, PhaseValidating, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, PhasePrepared, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, PhaseCommitting, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		// Everything converged but the marker is absent: recovery
		// cannot revalidate the install, so it parks.
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterFinalize
		input.Provider.State = ProviderUnknown
		outcome, _, err := store.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeRecoverableParked, "Recover(marker absent)")
	})
	t.Run("dormant_finalize_closes", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsBoardOnly())
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImportedDormant()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardOpenedDormant()); err != nil {
			t.Fatal(err)
		}
		stagingBackup := "/home/ivan/.local/state/ax/materializations/" + testMatID + "/board_staging"
		if _, err := store.UpdateAuthority(testMatID, "board_staging", AuthorityState{
			RootPath: testStagingRoot, CompletedSequences: []uint64{1},
			RollbackRoot: &stagingBackup, State: AuthorityPrepared,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateAuthority(testMatID, "board_staging", AuthorityState{
			RootPath: testStagingRoot, CompletedSequences: []uint64{1}, State: AuthorityCommitted,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, PhaseValidating, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, PhasePrepared, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, PhaseCommitting, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		// CR-MAT-08 on the passive path: the dormant probe closes
		// the finalize and the commit together.
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterFinalize
		input.Provider.State = ProviderUnknown
		input.Bridge.State = "dormant"
		input.Bridge.ManagerMatches = true
		outcome, _, err := store.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(dormant)")
		loaded, _, err := store.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.Phase != PhaseCommitted || loaded.TaskBoard.State != BoardDormantFinalized {
			t.Fatalf("phase = %q bridge = %q", loaded.Phase, loaded.TaskBoard.State)
		}
		if loaded.TaskBoard.OpenToken != nil {
			t.Fatalf("dormant finalize keeps the open token")
		}
	})
}

func TestRecoverResumes(t *testing.T) {
	t.Run("staging_resumes", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputs())
		if _, err := store.RecordProgress(testMatID, map[string][]uint32{testBlobA: {0}}, nil); err != nil {
			t.Fatal(err)
		}
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterTransfer
		input.Provider.State = ProviderUnknown
		outcome, evidence, err := store.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(staging)")
		if !strings.Contains(evidence.Reason, "absent chunk") {
			t.Fatalf("reason = %q", evidence.Reason)
		}
	})
	t.Run("committing_retries_commit", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		mustConvergePrepared(t, store)
		if _, err := store.Transition(testMatID, PhaseCommitting, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		// MJ-CRASH-COMMIT-LOST: prepared status retries the commit.
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterFinalize
		input.Bridge.State = BoardOpened
		outcome, evidence, err := store.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(retry commit)")
		if !strings.Contains(evidence.Reason, "retries the commit") {
			t.Fatalf("reason = %q", evidence.Reason)
		}
	})
	t.Run("bridge_import_retries", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		// TB-TXN-IMPORT-LOST: import may have succeeded while the
		// journal is not_started; the same-ID retry is safe.
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterPrepareOp
		input.Bridge.State = BoardNotStarted
		outcome, evidence, err := store.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(import lost)")
		if !strings.Contains(evidence.Reason, "import") {
			t.Fatalf("reason = %q", evidence.Reason)
		}
	})
	t.Run("bridge_open_retries", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		// TB-TXN-OPEN-LOST: open may have succeeded; the stable-ID
		// retry persists before commit.
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterOpen
		input.Bridge.State = BoardImported
		outcome, _, err := store.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(open lost)")
	})
	t.Run("bridge_adopt_retries", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardOpened()); err != nil {
			t.Fatal(err)
		}
		// TB-TXN-ADOPT-LOST: adopt may have succeeded after the
		// lease became authoritative; status plus the same-ID
		// retry, never a second manager.
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterActivation
		input.Bridge.State = BoardOpened
		input.Bridge.LeaseActive = true
		input.Bridge.BindingMatches = true
		input.Bridge.ManagerMatches = true
		outcome, evidence, err := store.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(adopt lost)")
		if !strings.Contains(evidence.Reason, "adopt") {
			t.Fatalf("reason = %q", evidence.Reason)
		}
	})
	t.Run("bridge_resume_retries", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		mustConvergePrepared(t, store)
		if _, err := store.Transition(testMatID, PhaseCommitting, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardAdopted()); err != nil {
			t.Fatal(err)
		}
		// TB-TXN-RESUME-LOST: resume may have started provider
		// work; status plus the same-ID retry.
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterFinalize
		input.Bridge.State = BoardAdopted
		input.Bridge.LeaseActive = true
		input.Bridge.BindingMatches = true
		input.Bridge.ManagerMatches = true
		input.NativeKnown = true
		input.NativeBefore = "tbm:manager:0198f4c8-7a10-7b22-8b3c-2234567890ab:" + testLeaseID
		input.NativeAfter = input.NativeBefore
		outcome, evidence, err := store.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(resume lost)")
		if !strings.Contains(evidence.Reason, "resume") {
			t.Fatalf("reason = %q", evidence.Reason)
		}
	})
}

// TestRecoverFailureMatrixRows pins the journal-observable Section 13.12
// rows through the production recovery entry.
func TestRecoverFailureMatrixRows(t *testing.T) {
	t.Run("owner_resume_lost_finalizes", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		mustConvergePrepared(t, store)
		// The owner-resume response is lost: the prepared journal
		// is preserved, no second manager starts, and the
		// reconciled statuses finalize under the same IDs.
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterActivation
		input.Bridge.State = BoardOpened
		input.Bridge.LeaseActive = true
		input.Bridge.BindingMatches = true
		input.Bridge.ManagerMatches = true
		outcome, evidence, err := store.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(owner resume lost)")
		mustEvidenceComplete(t, evidence, "Recover(owner resume lost)")
		loaded, _, err := store.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.Phase != PhasePrepared {
			t.Fatalf("phase = %q, want the preserved prepared journal", loaded.Phase)
		}
	})
	t.Run("operator_interrupt_retries_same_operation", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputs())
		if _, err := store.RecordProgress(testMatID, map[string][]uint32{testBlobA: {0, 1}}, nil); err != nil {
			t.Fatal(err)
		}
		// An operator interrupt preserves the journal and reports
		// whether authority changed: here it did not, so the same
		// operation retries.
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterTransfer
		input.Provider.State = ProviderUnknown
		outcome, evidence, err := store.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(interrupt)")
		if evidence.LeaseBefore != evidence.LeaseAfter {
			t.Fatalf("authority changed across the interrupt: %+v", evidence)
		}
		loaded, _, err := store.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		if len(loaded.CompletedBlobChunks[testBlobA]) != 2 {
			t.Fatalf("chunks = %+v, want the preserved progress", loaded.CompletedBlobChunks)
		}
	})
	t.Run("adopt_fails_after_lease_stays_stopped_owner", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardOpened()); err != nil {
			t.Fatal(err)
		}
		// Adopt fails after the new lease: the destination remains
		// the stopped owner and retries adopt or resume — the
		// journal never rolls back an activated lease.
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterActivation
		input.Bridge.State = "stopped"
		input.Bridge.LeaseActive = true
		input.Bridge.BindingMatches = true
		input.Bridge.ManagerMatches = true
		outcome, evidence, err := store.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(adopt failed)")
		if !strings.Contains(evidence.Reason, "adopt") {
			t.Fatalf("reason = %q", evidence.Reason)
		}
		loaded, _, err := store.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.Phase == PhaseRolledBack || loaded.Phase == PhaseRollingBack {
			t.Fatalf("phase = %q, want no rollback past activation", loaded.Phase)
		}
	})
}

func TestParkedEvidenceIsDurable(t *testing.T) {
	store := openTestStore(t)
	mustCreate(t, store, testInputsComposite())
	if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateTaskBoard(testMatID, testBoardOpened()); err != nil {
		t.Fatal(err)
	}
	input := baseRecoveryInput()
	input.Boundary = BoundaryAfterActivation
	input.Provider.State = ProviderUnknown
	input.Bridge.State = ProviderUnknown
	outcome, _, err := store.Recover(testMatID, input)
	if err != nil {
		t.Fatalf("Recover error = %v", err)
	}
	mustOutcome(t, outcome, OutcomeRecoverableParked, "Recover(parked)")
	// The parked record persists with the retained lease, the exact
	// IDs, the checkpoint/native identity, and the last error.
	framed, err := os.ReadFile(filepath.Join(store.root, testMatID, "recovery.json"))
	if err != nil {
		t.Fatal(err)
	}
	var evidence Evidence
	if err := json.Unmarshal(framed, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.Selected != OutcomeRecoverableParked {
		t.Fatalf("selected = %q", evidence.Selected)
	}
	if evidence.LeaseAfter != testLease() {
		t.Fatalf("retained lease = %+v", evidence.LeaseAfter)
	}
	if evidence.OperationIDs["prepare_operation_id"] != testPrepareOp ||
		evidence.OperationIDs["adopt_operation_id"] != testAdoptOp {
		t.Fatalf("operation IDs = %+v", evidence.OperationIDs)
	}
	if evidence.Reason == "" || evidence.Remediation == "" {
		t.Fatalf("evidence = %+v", evidence)
	}
	loaded, _, err := store.Get(testMatID)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.LastError) == 0 {
		t.Fatalf("parked journal carries no last_error")
	}
	// The outcome is one of exactly three: never a fourth.
	switch evidence.Selected {
	case OutcomeSafeRetry, OutcomeExplicitRollback, OutcomeRecoverableParked:
	default:
		t.Fatalf("selected = %q", evidence.Selected)
	}
}
