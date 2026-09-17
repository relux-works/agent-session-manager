package matjournal

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/secconftest"
)

// reopen loads a fresh store over the same data root: the simulated
// clean restart after a crash. Hooks never survive the restart.
func reopen(t *testing.T, store *Store) *Store {
	t.Helper()
	root := store.root
	dataRoot := filepath.Dir(root)
	fresh, err := Open(dataRoot)
	if err != nil {
		t.Fatal(err)
	}
	fresh.Now = fixedClock
	return fresh
}

// mustCrashFault fails unless err is the injected fault for the armed
// point carrying the expected outcome.
func mustCrashFault(t *testing.T, err error, point string, outcome secconftest.Outcome, what string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s error = nil, want injected fault at %s", what, point)
	}
	var fault *secconftest.Fault
	if !errors.As(err, &fault) {
		t.Fatalf("%s error = %v, want *secconftest.Fault", what, err)
	}
	if fault.Point() != point {
		t.Fatalf("%s point = %q, want %q", what, fault.Point(), point)
	}
	if fault.Outcome() != outcome {
		t.Fatalf("%s outcome = %q, want %q", what, fault.Outcome(), outcome)
	}
}

// journalFilesExist reports whether the journal and receipt exist.
func journalFilesExist(t *testing.T, store *Store) (journal, receipt bool) {
	t.Helper()
	matDir := filepath.Join(store.root, testMatID)
	if _, err := os.Stat(filepath.Join(matDir, "journal.json")); err == nil {
		journal = true
	}
	entries, err := os.ReadDir(filepath.Join(matDir, "operations"))
	if err == nil && len(entries) > 0 {
		receipt = true
	}
	return journal, receipt
}

func TestCrashBeforeCreateIsSafeRetry(t *testing.T) {
	store := openTestStore(t)
	injector := &secconftest.Injector{}
	store.BeforeWrite = func() error { return injector.MaybeFail(secconftest.PointPrepareEnter) }
	injector.Arm(secconftest.PointPrepareEnter)
	// CR-MAT-01: the crash lands before journal creation.
	_, _, err := store.Create(testInputsComposite())
	mustCrashFault(t, err, secconftest.PointPrepareEnter, secconftest.OutcomeSafeRetry, "Create(prepare fault)")
	if journal, receipt := journalFilesExist(t, store); journal || receipt {
		t.Fatalf("after prepare fault: journal = %v, receipt = %v, want no durable state", journal, receipt)
	}
	if remaining := injector.RemainingArmed(); len(remaining) != 0 {
		t.Fatalf("armed points remain: %q", remaining)
	}
	// After the clean restart, recovery classifies safe_retry and the
	// identical retry creates exactly one journal.
	restarted := reopen(t, store)
	input := baseRecoveryInput()
	input.Boundary = BoundaryBeforeCreate
	input.Provider.State = ProviderUnknown
	outcome, _, err := restarted.Recover(testMatID, input)
	if err != nil {
		t.Fatalf("Recover error = %v", err)
	}
	mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(CR-MAT-01)")
	ref, _ := mustCreate(t, restarted, testInputsComposite())
	if ref.Phase != PhaseStaging {
		t.Fatalf("retry phase = %q", ref.Phase)
	}
	if journal, receipt := journalFilesExist(t, restarted); !journal || !receipt {
		t.Fatalf("after retry: journal = %v, receipt = %v, want both durable", journal, receipt)
	}
}

func TestCrashBetweenJournalAndReceiptReplays(t *testing.T) {
	store := openTestStore(t)
	interrupted := errors.New("crash after journal before receipt")
	store.AfterJournal = func() error { return interrupted }
	// CR-MAT-02: the journal is durable but the prepare receipt never
	// installed.
	_, _, err := store.Create(testInputsComposite())
	if !errors.Is(err, interrupted) {
		t.Fatalf("Create(journal fault) = %v, want the interior fault", err)
	}
	if journal, receipt := journalFilesExist(t, store); !journal || receipt {
		t.Fatalf("after journal fault: journal = %v, receipt = %v, want journal only", journal, receipt)
	}
	// MJ-RPC-PREPARE-LOST: the retry carries the caller-retained IDs
	// and the identical body, completes the receipt, and creates no
	// second journal.
	restarted := reopen(t, store)
	ref, journal := mustCreate(t, restarted, testInputsComposite())
	if ref.Phase != PhaseStaging || journal.PrepareOperationID != testPrepareOp {
		t.Fatalf("replay = %+v", ref)
	}
	input := baseRecoveryInput()
	input.Boundary = BoundaryAfterPrepare
	outcome, evidence, err := restarted.Recover(testMatID, input)
	if err != nil {
		t.Fatalf("Recover error = %v", err)
	}
	mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(CR-MAT-02)")
	if !evidence.ReceiptPresent {
		t.Fatalf("receipt absent after replay")
	}
}

func TestCrashAfterTransferResumes(t *testing.T) {
	store := openTestStore(t)
	mustCreate(t, store, testInputs())
	if _, err := store.RecordProgress(testMatID, map[string][]uint32{testBlobA: {0}}, nil); err != nil {
		t.Fatal(err)
	}
	// CR-MAT-03: the crash lands after transfer progress with no
	// verified whole blob (MJ-CRASH-STAGE).
	restarted := reopen(t, store)
	loaded, _, err := restarted.Get(testMatID)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.CompletedBlobChunks[testBlobA]) != 1 || len(loaded.VerifiedBlobIDs) != 0 {
		t.Fatalf("staged facts = %+v", loaded.CompletedBlobChunks)
	}
	input := baseRecoveryInput()
	input.Boundary = BoundaryAfterTransfer
	input.Provider.State = ProviderUnknown
	outcome, _, err := restarted.Recover(testMatID, input)
	if err != nil {
		t.Fatalf("Recover error = %v", err)
	}
	mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(CR-MAT-03)")
	// The resume requests only absent indexes on the same blob.
	updated, err := restarted.RecordProgress(testMatID, map[string][]uint32{testBlobA: {1}}, nil)
	if err != nil {
		t.Fatalf("resume RecordProgress error = %v", err)
	}
	if len(updated.CompletedBlobChunks[testBlobA]) != 2 {
		t.Fatalf("chunks = %+v", updated.CompletedBlobChunks)
	}
}

func TestCrashAfterValidationResumes(t *testing.T) {
	store := openTestStore(t)
	mustCreate(t, store, testInputs())
	if _, err := store.RecordProgress(testMatID, map[string][]uint32{testBlobA: {0}}, []string{testBlobA}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Transition(testMatID, PhaseValidating, TransitionOpts{}); err != nil {
		t.Fatal(err)
	}
	// CR-MAT-04: validation-phase facts survive the restart.
	restarted := reopen(t, store)
	input := baseRecoveryInput()
	input.Boundary = BoundaryAfterValidation
	input.Provider.State = ProviderUnknown
	outcome, _, err := restarted.Recover(testMatID, input)
	if err != nil {
		t.Fatalf("Recover error = %v", err)
	}
	mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(CR-MAT-04)")
	loaded, _, err := restarted.Get(testMatID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Phase != PhaseValidating || len(loaded.VerifiedBlobIDs) != 1 {
		t.Fatalf("phase = %q verified = %+v", loaded.Phase, loaded.VerifiedBlobIDs)
	}
}

func TestCrashAfterProviderPreparePersistsToken(t *testing.T) {
	store := openTestStore(t)
	mustCreate(t, store, testInputsComposite())
	interrupted := errors.New("crash after provider prepare")
	store.AfterJournal = func() error { return interrupted }
	// CR-MAT-05, provider half: the prepared token is durable even
	// though the caller never saw the result (MJ-CRASH-PREPARE-LOST).
	_, err := store.UpdateProvider(testMatID, testProvider())
	if !errors.Is(err, interrupted) {
		t.Fatalf("UpdateProvider fault = %v, want the interior fault", err)
	}
	restarted := reopen(t, store)
	loaded, _, err := restarted.Get(testMatID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Provider == nil || loaded.Provider.State != ProviderPrepared || loaded.Provider.RollbackToken == nil {
		t.Fatalf("provider = %+v, want the durable prepared token", loaded.Provider)
	}
	// Status agrees with the durable token: the operation continues
	// to commit under the same IDs.
	input := baseRecoveryInput()
	input.Boundary = BoundaryAfterPrepareOp
	input.Bridge.State = BoardNotStarted
	outcome, _, err := restarted.Recover(testMatID, input)
	if err != nil {
		t.Fatalf("Recover error = %v", err)
	}
	mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(CR-MAT-05 provider)")
}

func TestCrashAfterBridgeImportPersists(t *testing.T) {
	store := openTestStore(t)
	mustCreate(t, store, testInputsComposite())
	interrupted := errors.New("crash after bridge import")
	store.AfterJournal = func() error { return interrupted }
	// CR-MAT-05, bridge half: the imported state is durable.
	_, err := store.UpdateTaskBoard(testMatID, testBoardImported())
	if !errors.Is(err, interrupted) {
		t.Fatalf("UpdateTaskBoard fault = %v, want the interior fault", err)
	}
	restarted := reopen(t, store)
	loaded, _, err := restarted.Get(testMatID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TaskBoard.State != BoardImported || loaded.TaskBoard.ImportToken == nil {
		t.Fatalf("task-board = %+v, want the durable imported state", loaded.TaskBoard)
	}
	input := baseRecoveryInput()
	input.Boundary = BoundaryAfterPrepareOp
	input.Provider.State = ProviderPrepared
	input.Bridge.State = BoardImported
	outcome, _, err := restarted.Recover(testMatID, input)
	if err != nil {
		t.Fatalf("Recover error = %v", err)
	}
	mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(CR-MAT-05 bridge)")
}

func TestCrashAfterOpenOrPreparedContinues(t *testing.T) {
	t.Run("open", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		interrupted := errors.New("crash after bridge open")
		store.AfterJournal = func() error { return interrupted }
		// CR-MAT-06, bridge half: the opened state is durable.
		_, err := store.UpdateTaskBoard(testMatID, testBoardOpened())
		if !errors.Is(err, interrupted) {
			t.Fatalf("UpdateTaskBoard fault = %v, want the interior fault", err)
		}
		restarted := reopen(t, store)
		loaded, _, err := restarted.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.TaskBoard.State != BoardOpened {
			t.Fatalf("task-board = %q", loaded.TaskBoard.State)
		}
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterOpen
		input.Provider.State = ProviderPrepared
		input.Bridge.State = BoardOpened
		outcome, _, err := restarted.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(CR-MAT-06 open)")
	})
	t.Run("prepared", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		mustConvergePrepared(t, store)
		// CR-MAT-06, host half: the prepared journal survives the
		// restart and continues under the same IDs.
		restarted := reopen(t, store)
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterOpen
		input.Bridge.State = BoardOpened
		outcome, _, err := restarted.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(CR-MAT-06 prepared)")
	})
}

func TestCrashAfterActivationReconciles(t *testing.T) {
	setup := func(t *testing.T) *Store {
		store := openTestStore(t)
		mustCreate(t, store, testInputsComposite())
		mustConvergePrepared(t, store)
		if _, err := store.Transition(testMatID, PhaseCommitting, TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		return store
	}
	native := "tbm:manager:0198f4c8-7a10-7b22-8b3c-2234567890ab:" + testLeaseID
	t.Run("adopted_finalizes", func(t *testing.T) {
		store := setup(t)
		interrupted := errors.New("crash after adopt recorded")
		store.AfterJournal = func() error { return interrupted }
		// CR-MAT-07: owner activation may have occurred; the
		// adopted binding is durable and status agrees.
		_, err := store.UpdateTaskBoard(testMatID, testBoardAdopted())
		if !errors.Is(err, interrupted) {
			t.Fatalf("UpdateTaskBoard fault = %v, want the interior fault", err)
		}
		restarted := reopen(t, store)
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterActivation
		input.Bridge.State = BoardAdopted
		input.Bridge.LeaseActive = true
		input.Bridge.BindingMatches = true
		input.Bridge.ManagerMatches = true
		input.NativeKnown = true
		input.NativeBefore, input.NativeAfter = native, native
		outcome, evidence, err := restarted.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(CR-MAT-07 adopted)")
		if evidence.OperationIDs["adopt_operation_id"] != testAdoptOp {
			t.Fatalf("operation IDs = %+v", evidence.OperationIDs)
		}
	})
	t.Run("ambiguous_parks", func(t *testing.T) {
		store := setup(t)
		restarted := reopen(t, store)
		// CR-MAT-07 with an unreachable bridge: adoption is
		// uncertain, so recovery parks instead of guessing.
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterActivation
		input.Bridge.State = ProviderUnknown
		outcome, _, err := restarted.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeRecoverableParked, "Recover(CR-MAT-07 ambiguous)")
	})
}

func TestCrashAfterFinalizeCloses(t *testing.T) {
	t.Run("resumed_commits", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsBoardOnlyOwnership())
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardOpened()); err != nil {
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
		if _, err := store.UpdateTaskBoard(testMatID, testBoardAdopted()); err != nil {
			t.Fatal(err)
		}
		interrupted := errors.New("crash after resume recorded")
		store.AfterJournal = func() error { return interrupted }
		// CR-MAT-08: finalize may have occurred; the resumed state
		// is durable.
		_, err := store.UpdateTaskBoard(testMatID, testBoardResumed())
		if !errors.Is(err, interrupted) {
			t.Fatalf("UpdateTaskBoard fault = %v, want the interior fault", err)
		}
		restarted := reopen(t, store)
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterFinalize
		input.Provider.State = ProviderUnknown
		input.Bridge.State = "running"
		input.Bridge.LeaseActive = true
		input.Bridge.BindingMatches = true
		input.Bridge.ManagerMatches = true
		input.NativeKnown = true
		input.NativeBefore = "tbm:manager:0198f4c8-7a10-7b22-8b3c-2234567890ab:" + testLeaseID
		input.NativeAfter = input.NativeBefore
		outcome, _, err := restarted.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(CR-MAT-08 resumed)")
		loaded, _, err := restarted.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.Phase != PhaseCommitted {
			t.Fatalf("phase = %q, want the completed commit", loaded.Phase)
		}
	})
	t.Run("ambiguous_parks", func(t *testing.T) {
		store := openTestStore(t)
		mustCreate(t, store, testInputsBoardOnlyOwnership())
		if _, err := store.UpdateTaskBoard(testMatID, testBoardImported()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, testBoardOpened()); err != nil {
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
		restarted := reopen(t, store)
		// CR-MAT-08 with finalize unproven: recovery parks with
		// the blocking reason durable.
		input := baseRecoveryInput()
		input.Boundary = BoundaryAfterFinalize
		input.Provider.State = ProviderUnknown
		input.Bridge.State = ProviderUnknown
		outcome, _, err := restarted.Recover(testMatID, input)
		if err != nil {
			t.Fatalf("Recover error = %v", err)
		}
		mustOutcome(t, outcome, OutcomeRecoverableParked, "Recover(CR-MAT-08 ambiguous)")
	})
}

// testInputsBoardOnlyOwnership returns the task-board-only closure in
// ownership_transfer mode: no provider branch, no workspace bytes.
func testInputsBoardOnlyOwnership() CreateInputs {
	inputs := testInputsBoardOnly()
	inputs.ActivationMode = ModeOwnershipTransfer
	return inputs
}
