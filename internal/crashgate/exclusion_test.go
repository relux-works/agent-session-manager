package crashgate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/matjournal"
)

// recoveryJSONExists reports whether the parked/rollback evidence
// file is durable for the test materialization.
func recoveryJSONExists(t *testing.T, dataDir string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(dataDir, "materializations", testMatID, "recovery.json"))
	return err == nil
}

// TestCrashGateMutualExclusion proves the outcomes mutually exclusive
// and collectively exhaustive through the production Recover entry:
// contradictory, missing, torn, and ambiguous evidence all select
// recoverable_parked_state; safe_retry establishes no new durable
// fact beyond the journal and receipt it returns (no recovery.json),
// while every parked assessment persists its blocking evidence; no
// run invents a fourth outcome or reports an unclassified
// successful restart.
func TestCrashGateMutualExclusion(t *testing.T) {
	t.Run("contradictory_early_activation_parks", func(t *testing.T) {
		store, dataDir := journalDirs(t)
		mustCreate(t, store, compositeInputs())
		// The sub-table is phase-agnostic, so the store admits an
		// adopted bridge at staging through the production
		// entries; recovery quarantines the contradiction.
		if _, err := store.UpdateTaskBoard(testMatID, importedBoard()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, openedBoard()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, adoptedBoard()); err != nil {
			t.Fatal(err)
		}
		restarted := reopenJournal(t, dataDir)
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterActivation, matjournal.PathOwnerResume, func(input *matjournal.RecoveryInput) {
			input.Bridge.State = matjournal.BoardAdopted
			input.Bridge.LeaseActive = true
			input.Bridge.BindingMatches = true
			input.Bridge.ManagerMatches = true
			input.NativeKnown = true
			input.NativeBefore, input.NativeAfter = testNative, testNative
		}, matjournal.OutcomeRecoverableParked, "Recover(contradictory)")
		if !strings.Contains(evidence.Reason, "contradicts") {
			t.Fatalf("reason = %q", evidence.Reason)
		}
		parkedMustFailClosed(t, dataDir, matjournal.PhaseStaging, evidence, "Recover(contradictory)")
	})
	t.Run("contradictory_rollback_with_activation_parks", func(t *testing.T) {
		store, dataDir := journalDirs(t)
		mustCreate(t, store, compositeInputs())
		convergePrepared(t, store)
		if _, err := store.Transition(testMatID, matjournal.PhaseCommitting, matjournal.TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, adoptedBoard()); err != nil {
			t.Fatal(err)
		}
		// A rollback phase with an activated sub-state contradicts
		// the progression; no production entry writes it (rollback
		// refuses past activation), so the test plants the torn
		// progression bytes directly and recovery quarantines
		// them instead of trusting them.
		journalPath := filepath.Join(dataDir, "materializations", testMatID, "journal.json")
		raw, err := os.ReadFile(journalPath)
		if err != nil {
			t.Fatal(err)
		}
		var document map[string]any
		if err := json.Unmarshal(raw, &document); err != nil {
			t.Fatal(err)
		}
		document["phase"] = matjournal.PhaseRollingBack
		planted, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(journalPath, planted, 0o600); err != nil {
			t.Fatal(err)
		}
		restarted := reopenJournal(t, dataDir)
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterActivation, matjournal.PathOwnerResume, func(input *matjournal.RecoveryInput) {
			input.Bridge.State = matjournal.BoardAdopted
			input.Bridge.LeaseActive = true
			input.Bridge.BindingMatches = true
			input.Bridge.ManagerMatches = true
			input.NativeKnown = true
			input.NativeBefore, input.NativeAfter = testNative, testNative
		}, matjournal.OutcomeRecoverableParked, "Recover(contradictory rollback)")
		if !strings.Contains(evidence.Reason, "contradicts") {
			t.Fatalf("reason = %q", evidence.Reason)
		}
	})
	t.Run("missing_lease_parks", func(t *testing.T) {
		store, dataDir := journalDirs(t)
		mustCreate(t, store, compositeInputs())
		restarted := reopenJournal(t, dataDir)
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterPrepare, matjournal.PathOwnerResume, func(input *matjournal.RecoveryInput) {
			input.LeaseKnown = false
		}, matjournal.OutcomeRecoverableParked, "Recover(missing lease)")
		parkedMustFailClosed(t, dataDir, matjournal.PhaseStaging, evidence, "Recover(missing lease)")
	})
	t.Run("missing_native_parks", func(t *testing.T) {
		store, dataDir := journalDirs(t)
		mustCreate(t, store, compositeInputs())
		convergePrepared(t, store)
		if _, err := store.Transition(testMatID, matjournal.PhaseCommitting, matjournal.TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, adoptedBoard()); err != nil {
			t.Fatal(err)
		}
		restarted := reopenJournal(t, dataDir)
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterActivation, matjournal.PathOwnerResume, func(input *matjournal.RecoveryInput) {
			input.Bridge.State = matjournal.BoardAdopted
			input.Bridge.LeaseActive = true
			input.Bridge.BindingMatches = true
			input.Bridge.ManagerMatches = true
			input.NativeKnown = false
		}, matjournal.OutcomeRecoverableParked, "Recover(missing native)")
		parkedMustFailClosed(t, dataDir, matjournal.PhaseCommitting, evidence, "Recover(missing native)")
	})
	t.Run("marker_mismatch_parks", func(t *testing.T) {
		store, dataDir := journalDirs(t)
		mustCreate(t, store, compositeInputs())
		restarted := reopenJournal(t, dataDir)
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterPrepare, matjournal.PathOwnerResume, func(input *matjournal.RecoveryInput) {
			input.Marker = matjournal.MarkerMismatch
		}, matjournal.OutcomeRecoverableParked, "Recover(marker mismatch)")
		parkedMustFailClosed(t, dataDir, matjournal.PhaseStaging, evidence, "Recover(marker mismatch)")
	})
	t.Run("marker_invalid_parks", func(t *testing.T) {
		store, dataDir := journalDirs(t)
		mustCreate(t, store, compositeInputs())
		restarted := reopenJournal(t, dataDir)
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterPrepare, matjournal.PathOwnerResume, func(input *matjournal.RecoveryInput) {
			input.Marker = matjournal.MarkerInvalid
		}, matjournal.OutcomeRecoverableParked, "Recover(marker invalid)")
		parkedMustFailClosed(t, dataDir, matjournal.PhaseStaging, evidence, "Recover(marker invalid)")
	})
	t.Run("unknown_unrecorded_prepare_parks", func(t *testing.T) {
		store, dataDir := journalDirs(t)
		mustCreate(t, store, compositeInputs())
		// The prepare may have executed unrecorded from CR-MAT-05
		// on and its status is unknown: recovery parks.
		restarted := reopenJournal(t, dataDir)
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterPrepareOp, matjournal.PathOwnerResume, func(input *matjournal.RecoveryInput) {
			input.Provider.State = matjournal.ProviderUnknown
		}, matjournal.OutcomeRecoverableParked, "Recover(unrecorded prepare)")
		parkedMustFailClosed(t, dataDir, matjournal.PhaseStaging, evidence, "Recover(unrecorded prepare)")
	})
	t.Run("unknown_unrecorded_import_parks", func(t *testing.T) {
		store, dataDir := journalDirs(t)
		mustCreate(t, store, boardOwnershipInputs())
		// The import may have executed unrecorded from CR-MAT-05
		// on and its status is unknown: recovery parks.
		restarted := reopenJournal(t, dataDir)
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterPrepareOp, matjournal.PathOwnerResume, func(input *matjournal.RecoveryInput) {
			input.Provider.State = matjournal.ProviderUnknown
			input.Bridge.State = matjournal.ProviderUnknown
		}, matjournal.OutcomeRecoverableParked, "Recover(unrecorded import)")
		parkedMustFailClosed(t, dataDir, matjournal.PhaseStaging, evidence, "Recover(unrecorded import)")
	})
	t.Run("torn_journal_parks", func(t *testing.T) {
		store, dataDir := journalDirs(t)
		mustCreate(t, store, compositeInputs())
		journalPath := filepath.Join(dataDir, "materializations", testMatID, "journal.json")
		if err := os.WriteFile(journalPath, []byte(`{"torn":true}`), 0o600); err != nil {
			t.Fatal(err)
		}
		restarted := reopenJournal(t, dataDir)
		// Torn bytes are untrusted, so the record carries unknown
		// phases and no operation IDs by design: the assertions
		// below name the torn shape instead of the complete
		// record.
		input := baseRecoveryInput()
		input.Boundary = matjournal.BoundaryAfterPrepare
		outcome, evidence, err := restarted.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, matjournal.OutcomeRecoverableParked, "Recover(torn)")
		if evidence.PhaseBefore != "unknown" || len(evidence.OperationIDs) != 0 {
			t.Fatalf("evidence = %+v, want unknown phases and no operation IDs", evidence)
		}
		if !strings.Contains(evidence.Reason, "validation") && !strings.Contains(evidence.Reason, "quarantined") {
			t.Fatalf("reason = %q", evidence.Reason)
		}
		if !recoveryJSONExists(t, dataDir) {
			t.Fatal("parked torn assessment left no durable evidence")
		}
	})
	t.Run("safe_retry_writes_no_new_evidence", func(t *testing.T) {
		store, dataDir := journalDirs(t)
		mustCreate(t, store, directInputs())
		if _, err := store.RecordProgress(testMatID, map[string][]uint32{testBlobA: {0}}, nil); err != nil {
			t.Fatal(err)
		}
		restarted := reopenJournal(t, dataDir)
		recoverMust(t, restarted, matjournal.BoundaryAfterTransfer, matjournal.PathOwnerResume, func(input *matjournal.RecoveryInput) {
			input.Provider.State = matjournal.ProviderUnknown
		}, matjournal.OutcomeSafeRetry, "Recover(resume)")
		if recoveryJSONExists(t, dataDir) {
			t.Fatal("safe_retry assessment wrote recovery.json; the journal and receipt it returns are the evidence")
		}
	})
	t.Run("terminal_replays_recorded_result", func(t *testing.T) {
		store, dataDir := journalDirs(t)
		mustCreate(t, store, boardDormantInputs())
		if _, err := store.UpdateTaskBoard(testMatID, importedBoardDormant()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, openedBoardDormant()); err != nil {
			t.Fatal(err)
		}
		stagingBackup := "/home/ivan/.local/state/ax/materializations/" + testMatID + "/board_staging"
		if _, err := store.UpdateAuthority(testMatID, "board_staging", matjournal.AuthorityState{
			RootPath: testStagingRoot, CompletedSequences: []uint64{1},
			RollbackRoot: &stagingBackup, State: matjournal.AuthorityPrepared,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateAuthority(testMatID, "board_staging", matjournal.AuthorityState{
			RootPath: testStagingRoot, CompletedSequences: []uint64{1}, State: matjournal.AuthorityCommitted,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, matjournal.PhaseValidating, matjournal.TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, matjournal.PhasePrepared, matjournal.TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, matjournal.PhaseCommitting, matjournal.TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		board := openedBoardDormant()
		dormant := "dormant"
		if _, err := store.UpdateTaskBoard(testMatID, matjournal.TaskBoardTransaction{
			BundleID: board.BundleID, ActivationMode: board.ActivationMode,
			ImportOperationID: board.ImportOperationID, OpenOperationID: board.OpenOperationID,
			AdoptOperationID: board.AdoptOperationID, ResumeOperationID: board.ResumeOperationID,
			State: matjournal.BoardDormantFinalized, DormantManagerRef: board.DormantManagerRef,
			LastBridgeState: &dormant, LastStatusAt: board.LastStatusAt,
			CleanupState: "pending_expiry", CleanupAfter: board.OpenExpiresAt,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, matjournal.PhaseCommitted, matjournal.TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		restarted := reopenJournal(t, dataDir)
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterFinalize, matjournal.PathPassiveSync, func(input *matjournal.RecoveryInput) {
			input.Provider.State = matjournal.ProviderUnknown
			input.Bridge.State = "dormant"
			input.Bridge.ManagerMatches = true
		}, matjournal.OutcomeSafeRetry, "Recover(terminal replay)")
		if evidence.PhaseAfter != matjournal.PhaseCommitted {
			t.Fatalf("phase_after = %q, want the recorded committed result", evidence.PhaseAfter)
		}
		if recoveryJSONExists(t, dataDir) {
			t.Fatal("terminal replay wrote recovery.json; the recorded result is the evidence")
		}
	})
	t.Run("terminal_parks_on_moved_lease", func(t *testing.T) {
		store, dataDir := journalDirs(t)
		mustCreate(t, store, boardDormantInputs())
		if _, err := store.UpdateTaskBoard(testMatID, importedBoardDormant()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, openedBoardDormant()); err != nil {
			t.Fatal(err)
		}
		stagingBackup := "/home/ivan/.local/state/ax/materializations/" + testMatID + "/board_staging"
		if _, err := store.UpdateAuthority(testMatID, "board_staging", matjournal.AuthorityState{
			RootPath: testStagingRoot, CompletedSequences: []uint64{1},
			RollbackRoot: &stagingBackup, State: matjournal.AuthorityPrepared,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateAuthority(testMatID, "board_staging", matjournal.AuthorityState{
			RootPath: testStagingRoot, CompletedSequences: []uint64{1}, State: matjournal.AuthorityCommitted,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, matjournal.PhaseValidating, matjournal.TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, matjournal.PhasePrepared, matjournal.TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, matjournal.PhaseCommitting, matjournal.TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		board := openedBoardDormant()
		dormant := "dormant"
		if _, err := store.UpdateTaskBoard(testMatID, matjournal.TaskBoardTransaction{
			BundleID: board.BundleID, ActivationMode: board.ActivationMode,
			ImportOperationID: board.ImportOperationID, OpenOperationID: board.OpenOperationID,
			AdoptOperationID: board.AdoptOperationID, ResumeOperationID: board.ResumeOperationID,
			State: matjournal.BoardDormantFinalized, DormantManagerRef: board.DormantManagerRef,
			LastBridgeState: &dormant, LastStatusAt: board.LastStatusAt,
			CleanupState: "pending_expiry", CleanupAfter: board.OpenExpiresAt,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, matjournal.PhaseCommitted, matjournal.TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		restarted := reopenJournal(t, dataDir)
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterFinalize, matjournal.PathPassiveSync, func(input *matjournal.RecoveryInput) {
			input.Provider.State = matjournal.ProviderUnknown
			input.Bridge.State = "dormant"
			input.Bridge.ManagerMatches = true
			input.LeaseAfter = matjournal.LeaseIdentity{Epoch: 5, ID: testLeaseID}
		}, matjournal.OutcomeRecoverableParked, "Recover(terminal moved lease)")
		if !recoveryJSONExists(t, dataDir) {
			t.Fatal("parked terminal assessment left no durable evidence")
		}
		loaded, _, err := restarted.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.Phase != matjournal.PhaseCommitted {
			t.Fatalf("phase = %q, want the frozen committed journal", loaded.Phase)
		}
		if len(loaded.LastError) != 0 && string(loaded.LastError) != "null" {
			t.Fatalf("terminal journal recorded last_error %s; terminal journals stay frozen", loaded.LastError)
		}
		_ = evidence
	})
}
