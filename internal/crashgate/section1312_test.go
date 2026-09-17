package crashgate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/matjournal"
)

// TestCrashGateSection1312 proves the Section 13.12 failure rows the
// harness can reach, each through the production entries with a
// crash and a clean restart: operator interrupt preserves the
// journal and reports whether authority changed; a lost
// owner-resume response preserves the prepared journal with no
// second process and finalizes under the same IDs; atomic rename
// blocked and disk full park with remediation (the ENOSPC errno
// delivered at the write seam plus genuine filesystem faults);
// task-board import/open/adopt failure before the new lease rolls
// back while the old owner stays authoritative, and adopt failure
// after the new lease stays the stopped owner without rolling
// back.
func TestCrashGateSection1312(t *testing.T) {
	t.Run("operator_interrupt_preserves_journal", func(t *testing.T) {
		store, dataDir := journalDirs(t)
		mustCreate(t, store, directInputs())
		if _, err := store.RecordProgress(testMatID, map[string][]uint32{testBlobA: {0, 1}}, nil); err != nil {
			t.Fatal(err)
		}
		interrupted := errors.New("operator interrupt")
		store.BeforeWrite = func() error { return interrupted }
		if _, err := store.Transition(testMatID, matjournal.PhaseValidating, matjournal.TransitionOpts{}); !errors.Is(err, interrupted) {
			t.Fatalf("Transition(interrupt) = %v, want the interrupt fault", err)
		}
		restarted := reopenJournal(t, dataDir)
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterTransfer, matjournal.PathOwnerResume, func(input *matjournal.RecoveryInput) {
			input.Provider.State = matjournal.ProviderUnknown
			input.ExternalEffect = "operator interrupt during transfer"
		}, matjournal.OutcomeSafeRetry, "Recover(interrupt)")
		if evidence.LeaseBefore != evidence.LeaseAfter {
			t.Fatalf("authority changed across the interrupt: %+v", evidence)
		}
		loaded, _, err := restarted.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		if len(loaded.CompletedBlobChunks[testBlobA]) != 2 || loaded.Phase != matjournal.PhaseStaging {
			t.Fatalf("chunks = %+v phase = %q, want the preserved progress", loaded.CompletedBlobChunks, loaded.Phase)
		}
		// The same operation retries: validation proceeds under
		// the same IDs.
		if _, err := restarted.Transition(testMatID, matjournal.PhaseValidating, matjournal.TransitionOpts{}); err != nil {
			t.Fatalf("retry Transition error = %v", err)
		}
	})
	t.Run("owner_resume_lost_finalizes_same_ids", func(t *testing.T) {
		store, dataDir := journalDirs(t)
		mustCreate(t, store, compositeInputs())
		convergePrepared(t, store)
		// The owner-resume response is lost after the prepared
		// journal went durable: the crash is the restart before
		// any second process could start.
		restarted := reopenJournal(t, dataDir)
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterActivation, matjournal.PathOwnerResume, func(input *matjournal.RecoveryInput) {
			input.Bridge.State = matjournal.BoardOpened
			input.Bridge.LeaseActive = true
			input.Bridge.BindingMatches = true
			input.Bridge.ManagerMatches = true
			input.Bridge.Live = false
			input.ExternalEffect = "owner-resume response lost after the prepared journal"
		}, matjournal.OutcomeSafeRetry, "Recover(owner resume lost)")
		loaded, _, err := restarted.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.Phase != matjournal.PhasePrepared {
			t.Fatalf("phase = %q, want the preserved prepared journal", loaded.Phase)
		}
		if loaded.PrepareOperationID != testPrepareOp || loaded.TaskBoard.ManagerSessionRef != nil {
			t.Fatalf("operation IDs moved or a manager started: %+v", loaded.TaskBoard)
		}
		if evidence.OperationIDs["prepare_operation_id"] != testPrepareOp {
			t.Fatalf("operation IDs = %+v, want the same IDs", evidence.OperationIDs)
		}
	})
	t.Run("rename_blocked_probe_parks", func(t *testing.T) {
		store, dataDir := journalDirs(t)
		mustCreate(t, store, compositeInputs())
		restarted := reopenJournal(t, dataDir)
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterPrepare, matjournal.PathOwnerResume, func(input *matjournal.RecoveryInput) {
			input.Host = matjournal.HostRenameBlocked
		}, matjournal.OutcomeRecoverableParked, "Recover(rename blocked)")
		if !strings.Contains(evidence.Remediation, "handles") {
			t.Fatalf("remediation = %q, want the close-handles remediation", evidence.Remediation)
		}
		parkedMustFailClosed(t, dataDir, matjournal.PhaseStaging, evidence, "Recover(rename blocked)")
	})
	t.Run("rename_blocked_real_fault", func(t *testing.T) {
		store, dataDir := journalDirs(t)
		mustCreate(t, store, compositeInputs())
		matDir := filepath.Join(dataDir, "materializations", testMatID)
		before, err := os.ReadFile(filepath.Join(matDir, "journal.json"))
		if err != nil {
			t.Fatal(err)
		}
		// A genuine filesystem fault: the materialization
		// directory refuses writes, so the journal replace
		// fails and no partial bytes land.
		if err := os.Chmod(matDir, 0o555); err != nil {
			t.Fatal(err)
		}
		_, updateErr := store.UpdateProvider(testMatID, preparedProvider())
		if updateErr == nil {
			_ = os.Chmod(matDir, 0o755)
			t.Fatal("UpdateProvider under a read-only directory error = nil, want the filesystem fault")
		}
		after, err := os.ReadFile(filepath.Join(matDir, "journal.json"))
		if err != nil {
			_ = os.Chmod(matDir, 0o755)
			t.Fatal(err)
		}
		if string(after) != string(before) {
			_ = os.Chmod(matDir, 0o755)
			t.Fatal("journal bytes changed across the refused write")
		}
		if err := os.Chmod(matDir, 0o755); err != nil {
			t.Fatal(err)
		}
		restarted := reopenJournal(t, dataDir)
		if _, err := restarted.UpdateProvider(testMatID, preparedProvider()); err != nil {
			t.Fatalf("retry UpdateProvider error = %v", err)
		}
		recoverMust(t, restarted, matjournal.BoundaryAfterPrepareOp, matjournal.PathOwnerResume, nil, matjournal.OutcomeSafeRetry, "Recover(rename fault retry)")
	})
	t.Run("disk_full_enospc_at_seam", func(t *testing.T) {
		store, dataDir := journalDirs(t)
		noSpace := &os.PathError{Op: "write", Path: "journal.json", Err: syscall.ENOSPC}
		store.BeforeWrite = func() error { return noSpace }
		_, _, err := store.Create(compositeInputs())
		if !errors.Is(err, syscall.ENOSPC) {
			t.Fatalf("Create(disk full) = %v, want the ENOSPC identity", err)
		}
		if journal, receipt := journalFilesExist(t, dataDir); journal || receipt {
			t.Fatalf("after ENOSPC: journal = %v, receipt = %v, want no durable state", journal, receipt)
		}
		restarted := reopenJournal(t, dataDir)
		mustCreate(t, restarted, compositeInputs())
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterPrepare, matjournal.PathOwnerResume, func(input *matjournal.RecoveryInput) {
			input.Host = matjournal.HostDiskFull
		}, matjournal.OutcomeRecoverableParked, "Recover(disk full)")
		if !strings.Contains(evidence.Remediation, "space") {
			t.Fatalf("remediation = %q, want the free-space remediation", evidence.Remediation)
		}
		parkedMustFailClosed(t, dataDir, matjournal.PhaseStaging, evidence, "Recover(disk full)")
	})
	t.Run("disk_full_readonly_data", func(t *testing.T) {
		dataDir := t.TempDir()
		store, err := matjournal.Open(dataDir)
		if err != nil {
			t.Fatal(err)
		}
		store.Now = fixedClock
		mats := filepath.Join(dataDir, "materializations")
		if err := os.Chmod(mats, 0o555); err != nil {
			t.Fatal(err)
		}
		_, _, createErr := store.Create(compositeInputs())
		if createErr == nil {
			_ = os.Chmod(mats, 0o755)
			t.Fatal("Create under a read-only root error = nil, want the filesystem fault")
		}
		if err := os.Chmod(mats, 0o755); err != nil {
			t.Fatal(err)
		}
		restarted := reopenJournal(t, dataDir)
		mustCreate(t, restarted, compositeInputs())
		recoverMust(t, restarted, matjournal.BoundaryAfterPrepare, matjournal.PathOwnerResume, nil, matjournal.OutcomeSafeRetry, "Recover(disk fault retry)")
	})
	t.Run("import_fails_before_lease_rolls_back", func(t *testing.T) {
		for _, probe := range []struct {
			name   string
			failed bool
			expire bool
		}{
			{"failed", true, false},
			{"token_expired", false, true},
		} {
			store, dataDir := journalDirs(t)
			mustCreate(t, store, compositeInputs())
			if _, err := store.UpdateProvider(testMatID, preparedProvider()); err != nil {
				t.Fatal(err)
			}
			if _, err := store.UpdateTaskBoard(testMatID, importedBoard()); err != nil {
				t.Fatal(err)
			}
			restarted := reopenJournal(t, dataDir)
			evidence := recoverMust(t, restarted, matjournal.BoundaryAfterPrepareOp, matjournal.PathGracefulTakeover, func(input *matjournal.RecoveryInput) {
				input.Bridge.State = matjournal.BoardFailed
				input.Bridge.Failed = probe.failed
				input.Bridge.TokenExpired = probe.expire
				input.Bridge.LeaseActive = false
				input.ExternalEffect = "bridge import/open/adopt failed before the new lease"
			}, matjournal.OutcomeExplicitRollback, "Recover("+probe.name+")")
			if evidence.LeaseBefore != evidence.LeaseAfter {
				t.Fatalf("%s: authority changed across the rollback: %+v", probe.name, evidence)
			}
			loaded, _, err := restarted.Get(testMatID)
			if err != nil {
				t.Fatal(err)
			}
			if loaded.Phase != matjournal.PhaseRolledBack {
				t.Fatalf("%s: phase = %q, want the terminal rollback", probe.name, loaded.Phase)
			}
		}
	})
	t.Run("adopt_fails_after_lease_stays_stopped_owner", func(t *testing.T) {
		store, dataDir := journalDirs(t)
		mustCreate(t, store, compositeInputs())
		if _, err := store.UpdateTaskBoard(testMatID, importedBoard()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, openedBoard()); err != nil {
			t.Fatal(err)
		}
		restarted := reopenJournal(t, dataDir)
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterActivation, matjournal.PathOwnerResume, func(input *matjournal.RecoveryInput) {
			input.Bridge.State = "stopped"
			input.Bridge.LeaseActive = true
			input.Bridge.BindingMatches = true
			input.Bridge.ManagerMatches = true
			input.ExternalEffect = "bridge adopt failed after the new lease"
		}, matjournal.OutcomeSafeRetry, "Recover(adopt failed)")
		if !strings.Contains(evidence.Reason, "adopt") {
			t.Fatalf("reason = %q, want the adopt retry", evidence.Reason)
		}
		loaded, _, err := restarted.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.Phase == matjournal.PhaseRolledBack || loaded.Phase == matjournal.PhaseRollingBack {
			t.Fatalf("phase = %q, want no rollback past activation", loaded.Phase)
		}
		if evidence.LeaseBefore != evidence.LeaseAfter {
			t.Fatalf("authority changed: %+v", evidence)
		}
	})
}
