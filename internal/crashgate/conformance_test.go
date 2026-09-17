package crashgate

import (
	"errors"
	"fmt"
	"sort"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/matjournal"
	"github.com/relux-works/agent-session-manager/internal/secconftest"
	"github.com/relux-works/agent-session-manager/internal/sessckpt"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// conformanceRow is one executed crash boundary row: the registry
// boundary and path it proves, the MAT seam the evaluator runs
// with (checkpoint rows have no evaluator), the closure and
// variant, and the driver entry.
type conformanceRow struct {
	name         string
	boundary     string
	evaluatedVia string
	path         string
	recoverPath  string
	closure      string
	variant      string
	driver       string
	run          func(t *testing.T) ConformanceRecord
}

// assertConformanceCoverage proves the row table covers the registry
// both ways: every reachable registry path has at least one row,
// every row names a reachable registry path with the registry's
// driver, and no row names a NOT APPLICABLE path. The
// token-preserving mutant (a reachable path flipped to NOT
// APPLICABLE with its ID intact) fails here and in the execution
// below while the ID-set derivation still passes.
func assertConformanceCoverage(t *testing.T, rows []conformanceRow) {
	t.Helper()
	reachable := map[string]string{}
	for _, boundary := range Registry() {
		for _, path := range boundary.Paths {
			if path.Reachable {
				reachable[boundary.ID+"\x00"+path.Name] = path.Driver
			}
		}
	}
	covered := map[string]int{}
	for _, row := range rows {
		key := row.boundary + "\x00" + row.path
		driver, ok := reachable[key]
		if !ok {
			t.Fatalf("row %s names %s/%s, which the registry does not mark reachable", row.name, row.boundary, row.path)
		}
		if row.driver != driver {
			t.Fatalf("row %s drives %s with %q, want registry driver %q", row.name, key, row.driver, driver)
		}
		covered[key]++
	}
	var uncovered []string
	for key := range reachable {
		if covered[key] == 0 {
			uncovered = append(uncovered, key)
		}
	}
	sort.Strings(uncovered)
	if len(uncovered) != 0 {
		t.Fatalf("registry reachable paths without a conformance row: %q", uncovered)
	}
	t.Logf("conformance table drives %d rows over %d reachable paths", len(rows), len(reachable))
}

// recoverMust runs the recovery evaluator after the clean restart
// and asserts exactly the wanted outcome with a complete evidence
// record.
func recoverMust(t *testing.T, store *matjournal.Store, boundary, recoverPath string, mutate func(*matjournal.RecoveryInput), want matjournal.Outcome, what string) matjournal.Evidence {
	t.Helper()
	input := baseRecoveryInput()
	input.Boundary = boundary
	input.Path = recoverPath
	if mutate != nil {
		mutate(&input)
	}
	outcome, evidence, err := store.Recover(testMatID, input)
	if err != nil {
		t.Fatalf("%s Recover error = %v", what, err)
	}
	mustOutcome(t, outcome, want, what)
	mustJournalEvidenceComplete(t, evidence, what)
	if evidence.Boundary != boundary || evidence.Path != recoverPath {
		t.Fatalf("%s evidence names %s/%s, want %s/%s", what, evidence.Boundary, evidence.Path, boundary, recoverPath)
	}
	return evidence
}

// closureInputs selects the prepare closure by label.
func closureInputs(t *testing.T, closure string) matjournal.CreateInputs {
	t.Helper()
	switch closure {
	case "direct":
		return directInputs()
	case "composite":
		return compositeInputs()
	case "direct_provider":
		return directProviderInputs()
	case "board_dormant":
		return boardDormantInputs()
	case "board_ownership":
		return boardOwnershipInputs()
	default:
		t.Fatalf("unknown closure %q", closure)
		return matjournal.CreateInputs{}
	}
}

// driveCheckpointCapture injects the crash at one capture seam,
// restarts clean, and proves the identical retry converges on the
// recorded checkpoint with its receipt: safe_retry by construction.
// Capture is append-only and has no evaluator, rollback, or parked
// semantics; the retry convergence is the outcome proof.
func driveCheckpointCapture(t *testing.T, boundary, path, seam string) ConformanceRecord {
	t.Helper()
	repoDir, created := chainFixture(t)
	store, dataDir := checkpointDirs(t)
	var inputs sessckpt.Inputs
	if path == PathTaskBoard {
		inputs = captureInputsBoard([]string{created}, testOpA)
	} else {
		inputs = captureInputs([]string{created}, testOpA)
	}
	if seam == "before_write" {
		injector := &secconftest.Injector{}
		store.BeforeWrite = func() error { return injector.MaybeFail(secconftest.PointPrepareEnter) }
		injector.Arm(secconftest.PointPrepareEnter)
		_, _, err := store.Capture(mustOpenRepo(t, repoDir), inputs)
		var fault *secconftest.Fault
		if !errors.As(err, &fault) || fault.Point() != secconftest.PointPrepareEnter {
			t.Fatalf("Capture(prepare fault) = %v, want the injected fault", err)
		}
		if blobs, receipts := countCheckpointFiles(t, dataDir); blobs != 0 || receipts != 0 {
			t.Fatalf("after prepare fault: blobs = %d, receipts = %d, want 0, 0", blobs, receipts)
		}
	} else {
		interrupted := errors.New("crash at the " + seam + " seam")
		if seam == "after_blob" {
			store.AfterBlob = func() error { return interrupted }
		} else {
			store.AfterCommit = func() error { return interrupted }
		}
		_, _, err := store.Capture(mustOpenRepo(t, repoDir), inputs)
		if !errors.Is(err, interrupted) {
			t.Fatalf("Capture(%s fault) = %v, want the interior fault", seam, err)
		}
		wantBlobs, wantReceipts := 1, 0
		if seam == "after_commit" {
			wantReceipts = 1
		}
		if blobs, receipts := countCheckpointFiles(t, dataDir); blobs != wantBlobs || receipts != wantReceipts {
			t.Fatalf("after %s fault: blobs = %d, receipts = %d, want %d, %d", seam, blobs, receipts, wantBlobs, wantReceipts)
		}
		everyCheckpointVerifies(t, dataDir)
	}
	preBlobs, preReceipts := countCheckpointFiles(t, dataDir)
	// Clean restart: fresh handles over the same directories; hooks
	// never survive.
	restarted := reopenCheckpoint(t, dataDir)
	chain := mustOpenRepo(t, repoDir)
	events, err := chain.ListEvents(testSessionID)
	if err != nil || len(events) != 1 {
		t.Fatalf("chain after restart lists %d events, want the single bootstrap event", len(events))
	}
	ref, raw, err := restarted.Capture(chain, inputs)
	if err != nil {
		t.Fatalf("retry Capture error = %v", err)
	}
	if _, _, err := sessrepo.AttestCheckpointRecord(raw); err != nil {
		t.Fatalf("retry bytes do not attest: %v", err)
	}
	stored, err := restarted.Get(ref.CheckpointID)
	if err != nil {
		t.Fatalf("Get after retry error = %v", err)
	}
	if string(stored) != string(raw) {
		t.Fatalf("retry bytes differ from stored bytes")
	}
	if blobs, receipts := countCheckpointFiles(t, dataDir); blobs != 1 || receipts != 1 {
		t.Fatalf("after retry: blobs = %d, receipts = %d, want 1, 1", blobs, receipts)
	}
	if seam == "after_commit" {
		again, againRaw, err := restarted.Capture(chain, inputs)
		if err != nil {
			t.Fatalf("replay Capture error = %v", err)
		}
		if again != ref || string(againRaw) != string(raw) {
			t.Fatalf("receipt replay diverged from the recorded result")
		}
	}
	native := testProviderManifest
	if path == PathTaskBoard {
		native = testBoardBundle
	}
	lease := leaseString(matjournal.LeaseIdentity{Epoch: 1, ID: testLeaseID})
	return ConformanceRecord{
		Boundary:       boundary,
		RegistryPath:   path,
		EvaluatedVia:   "capture-blob-receipt-seam",
		Closure:        path,
		Variant:        seam,
		Driver:         DriverCheckpointCapture,
		OperationIDs:   map[string]string{"capture_operation_id": testOpA},
		PhaseBefore:    fmt.Sprintf("blobs=%d receipts=%d", preBlobs, preReceipts),
		PhaseAfter:     fmt.Sprintf("blobs=1 receipts=1 checkpoint=%s", ref.CheckpointID),
		ReceiptPresent: true,
		ExternalEffect: "none (capture performs no provider or task-board I/O)",
		StatusProbe:    "n/a (no external effect to reconcile)",
		LeaseBefore:    lease,
		LeaseAfter:     lease,
		NativeBefore:   native,
		NativeAfter:    native,
		Selected:       string(matjournal.OutcomeSafeRetry),
		Reason:         fmt.Sprintf("identical retry converged on checkpoint %s with its receipt; no second identity", ref.CheckpointID),
		Remediation:    "retry the same capture operation with byte-identical inputs",
	}
}

func mustOpenRepo(t *testing.T, repoDir string) *sessrepo.Repository {
	t.Helper()
	repo, err := sessrepo.Open(repoDir)
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

// driveJournalCreate injects the crash at one prepare seam,
// restarts clean, replays the identical prepare, and evaluates.
func driveJournalCreate(t *testing.T, boundary, evaluatedVia, path, recoverPath, closure, seam string) ConformanceRecord {
	t.Helper()
	store, dataDir := journalDirs(t)
	inputs := closureInputs(t, closure)
	agree := closure == "composite" || closure == "direct_provider"
	switch seam {
	case "before_write":
		interrupted := errors.New("crash before the first durable byte")
		store.BeforeWrite = func() error { return interrupted }
		_, _, err := store.Create(inputs)
		if !errors.Is(err, interrupted) {
			t.Fatalf("Create(prepare fault) = %v, want the interior fault", err)
		}
		if journal, receipt := journalFilesExist(t, dataDir); journal || receipt {
			t.Fatalf("after prepare fault: journal = %v, receipt = %v, want no durable state", journal, receipt)
		}
		restarted := reopenJournal(t, dataDir)
		evidence := recoverMust(t, restarted, evaluatedVia, recoverPath, func(input *matjournal.RecoveryInput) {
			input.Provider.State = matjournal.ProviderUnknown
		}, matjournal.OutcomeSafeRetry, "Recover("+boundary+")")
		ref, _ := mustCreate(t, restarted, inputs)
		if ref.PrepareOperationID != testPrepareOp {
			t.Fatalf("retry PrepareOperationID = %q", ref.PrepareOperationID)
		}
		if journal, receipt := journalFilesExist(t, dataDir); !journal || !receipt {
			t.Fatalf("after retry: journal = %v, receipt = %v, want both durable", journal, receipt)
		}
		return recordFromEvidence(boundary, path, evaluatedVia, closure, seam, DriverJournalCreate, evidence)
	case "after_journal", "after_commit":
		interrupted := errors.New("crash at the " + seam + " seam")
		if seam == "after_journal" {
			store.AfterJournal = func() error { return interrupted }
		} else {
			store.AfterCommit = func() error { return interrupted }
		}
		_, _, err := store.Create(inputs)
		if !errors.Is(err, interrupted) {
			t.Fatalf("Create(%s fault) = %v, want the interior fault", seam, err)
		}
		journal, receipt := journalFilesExist(t, dataDir)
		if !journal || (receipt != (seam == "after_commit")) {
			t.Fatalf("after %s fault: journal = %v, receipt = %v", seam, journal, receipt)
		}
		restarted := reopenJournal(t, dataDir)
		ref, _ := mustCreate(t, restarted, inputs)
		if ref.PrepareOperationID != testPrepareOp {
			t.Fatalf("replay PrepareOperationID = %q", ref.PrepareOperationID)
		}
		if journal, receipt := journalFilesExist(t, dataDir); !journal || !receipt {
			t.Fatalf("after replay: journal = %v, receipt = %v, want both durable", journal, receipt)
		}
		evidence := recoverMust(t, restarted, evaluatedVia, recoverPath, func(input *matjournal.RecoveryInput) {
			if !agree {
				input.Provider.State = matjournal.ProviderUnknown
			}
		}, matjournal.OutcomeSafeRetry, "Recover("+boundary+")")
		return recordFromEvidence(boundary, path, evaluatedVia, closure, seam, DriverJournalCreate, evidence)
	default:
		t.Fatalf("unknown create seam %q", seam)
		return ConformanceRecord{}
	}
}

// driveJournalTransfer crashes after transfer progress and proves
// the resume requests only absent indexes under the same IDs.
func driveJournalTransfer(t *testing.T, boundary, path, recoverPath, closure string) ConformanceRecord {
	t.Helper()
	store, dataDir := journalDirs(t)
	mustCreate(t, store, closureInputs(t, closure))
	if _, err := store.RecordProgress(testMatID, map[string][]uint32{testBlobA: {0}}, nil); err != nil {
		t.Fatal(err)
	}
	restarted := reopenJournal(t, dataDir)
	loaded, _, err := restarted.Get(testMatID)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.CompletedBlobChunks[testBlobA]) != 1 || len(loaded.VerifiedBlobIDs) != 0 {
		t.Fatalf("staged facts = %+v", loaded.CompletedBlobChunks)
	}
	evidence := recoverMust(t, restarted, matjournal.BoundaryAfterTransfer, recoverPath, func(input *matjournal.RecoveryInput) {
		if closure != "composite" {
			input.Provider.State = matjournal.ProviderUnknown
		}
	}, matjournal.OutcomeSafeRetry, "Recover("+boundary+")")
	updated, err := restarted.RecordProgress(testMatID, map[string][]uint32{testBlobA: {1}}, nil)
	if err != nil {
		t.Fatalf("resume RecordProgress error = %v", err)
	}
	if len(updated.CompletedBlobChunks[testBlobA]) != 2 {
		t.Fatalf("chunks = %+v", updated.CompletedBlobChunks)
	}
	return recordFromEvidence(boundary, path, matjournal.BoundaryAfterTransfer, closure, "after_transfer", DriverJournalTransfer, evidence)
}

// driveJournalValidate crashes after validation-phase facts and
// proves they survive the restart.
func driveJournalValidate(t *testing.T, boundary, path, recoverPath, closure string) ConformanceRecord {
	t.Helper()
	store, dataDir := journalDirs(t)
	mustCreate(t, store, closureInputs(t, closure))
	if _, err := store.RecordProgress(testMatID, map[string][]uint32{testBlobA: {0}}, []string{testBlobA}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Transition(testMatID, matjournal.PhaseValidating, matjournal.TransitionOpts{}); err != nil {
		t.Fatal(err)
	}
	restarted := reopenJournal(t, dataDir)
	evidence := recoverMust(t, restarted, matjournal.BoundaryAfterValidation, recoverPath, func(input *matjournal.RecoveryInput) {
		if closure != "composite" {
			input.Provider.State = matjournal.ProviderUnknown
		}
	}, matjournal.OutcomeSafeRetry, "Recover("+boundary+")")
	loaded, _, err := restarted.Get(testMatID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Phase != matjournal.PhaseValidating || len(loaded.VerifiedBlobIDs) != 1 {
		t.Fatalf("phase = %q verified = %+v", loaded.Phase, loaded.VerifiedBlobIDs)
	}
	return recordFromEvidence(boundary, path, matjournal.BoundaryAfterValidation, closure, "after_validation", DriverJournalValidate, evidence)
}

// driveJournalPrepareOp crashes after the provider prepare or the
// bridge import, or executes the allowed rollback when the bridge
// failed before the new lease.
func driveJournalPrepareOp(t *testing.T, boundary, path, recoverPath, variant string) ConformanceRecord {
	t.Helper()
	const closure = "composite"
	store, dataDir := journalDirs(t)
	mustCreate(t, store, compositeInputs())
	interrupted := errors.New("crash after the prepare operation")
	switch variant {
	case "provider_half":
		store.AfterJournal = func() error { return interrupted }
		if _, err := store.UpdateProvider(testMatID, preparedProvider()); !errors.Is(err, interrupted) {
			t.Fatalf("UpdateProvider fault = %v, want the interior fault", err)
		}
		restarted := reopenJournal(t, dataDir)
		loaded, _, err := restarted.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.Provider == nil || loaded.Provider.State != matjournal.ProviderPrepared || loaded.Provider.RollbackToken == nil {
			t.Fatalf("provider = %+v, want the durable prepared token", loaded.Provider)
		}
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterPrepareOp, recoverPath, nil, matjournal.OutcomeSafeRetry, "Recover("+boundary+")")
		return recordFromEvidence(boundary, path, matjournal.BoundaryAfterPrepareOp, closure, variant, DriverJournalPrepareOp, evidence)
	case "bridge_half":
		store.AfterJournal = func() error { return interrupted }
		if _, err := store.UpdateTaskBoard(testMatID, importedBoard()); !errors.Is(err, interrupted) {
			t.Fatalf("UpdateTaskBoard fault = %v, want the interior fault", err)
		}
		restarted := reopenJournal(t, dataDir)
		loaded, _, err := restarted.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.TaskBoard.State != matjournal.BoardImported || loaded.TaskBoard.ImportToken == nil {
			t.Fatalf("task-board = %+v, want the durable imported state", loaded.TaskBoard)
		}
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterPrepareOp, recoverPath, func(input *matjournal.RecoveryInput) {
			input.Bridge.State = matjournal.BoardImported
		}, matjournal.OutcomeSafeRetry, "Recover("+boundary+")")
		return recordFromEvidence(boundary, path, matjournal.BoundaryAfterPrepareOp, closure, variant, DriverJournalPrepareOp, evidence)
	case "rollback_failed":
		if _, err := store.UpdateProvider(testMatID, preparedProvider()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateTaskBoard(testMatID, importedBoard()); err != nil {
			t.Fatal(err)
		}
		restarted := reopenJournal(t, dataDir)
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterPrepareOp, recoverPath, func(input *matjournal.RecoveryInput) {
			input.Bridge.State = matjournal.BoardFailed
			input.Bridge.Failed = true
			input.Bridge.LeaseActive = false
		}, matjournal.OutcomeExplicitRollback, "Recover("+boundary+")")
		loaded, _, err := restarted.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.Phase != matjournal.PhaseRolledBack || len(loaded.LastError) == 0 {
			t.Fatalf("phase = %q last_error = %s, want the terminal rollback", loaded.Phase, loaded.LastError)
		}
		return recordFromEvidence(boundary, path, matjournal.BoundaryAfterPrepareOp, closure, variant, DriverJournalPrepareOp, evidence)
	default:
		t.Fatalf("unknown prepare-op variant %q", variant)
		return ConformanceRecord{}
	}
}

// walkCompositeToValidating drives a composite journal through
// provider prepared, both authorities prepared, bridge imported and
// opened, and the staging-to-validating transition.
func walkCompositeToValidating(t *testing.T, store *matjournal.Store) {
	t.Helper()
	if _, err := store.UpdateProvider(testMatID, preparedProvider()); err != nil {
		t.Fatalf("UpdateProvider error = %v", err)
	}
	providerRoot := testTxRoot + "/backups/codex_sessions"
	if _, err := store.UpdateAuthority(testMatID, "codex_sessions", matjournal.AuthorityState{
		RootPath: testRootProvider, CompletedSequences: []uint64{2}, RollbackRoot: &providerRoot, State: matjournal.AuthorityPrepared,
	}); err != nil {
		t.Fatalf("UpdateAuthority error = %v", err)
	}
	workspaceRoot := "/home/ivan/.local/state/ax/materializations/" + testMatID + "/workspace_relux"
	if _, err := store.UpdateAuthority(testMatID, "workspace_relux", matjournal.AuthorityState{
		RootPath: testRootWorkspace, CompletedSequences: []uint64{1}, RollbackRoot: &workspaceRoot, State: matjournal.AuthorityPrepared,
	}); err != nil {
		t.Fatalf("UpdateAuthority error = %v", err)
	}
	if _, err := store.UpdateTaskBoard(testMatID, importedBoard()); err != nil {
		t.Fatalf("UpdateTaskBoard error = %v", err)
	}
	if _, err := store.UpdateTaskBoard(testMatID, openedBoard()); err != nil {
		t.Fatalf("UpdateTaskBoard error = %v", err)
	}
	if _, err := store.Transition(testMatID, matjournal.PhaseValidating, matjournal.TransitionOpts{}); err != nil {
		t.Fatalf("Transition error = %v", err)
	}
}

// walkDirectProviderToValidating drives a direct closure with a
// provider branch through provider prepared, both authorities
// prepared, and the staging-to-validating transition.
func walkDirectProviderToValidating(t *testing.T, store *matjournal.Store) {
	t.Helper()
	if _, err := store.UpdateProvider(testMatID, preparedProvider()); err != nil {
		t.Fatalf("UpdateProvider error = %v", err)
	}
	providerRoot := testTxRoot + "/backups/codex_sessions"
	if _, err := store.UpdateAuthority(testMatID, "codex_sessions", matjournal.AuthorityState{
		RootPath: testRootProvider, CompletedSequences: []uint64{2}, RollbackRoot: &providerRoot, State: matjournal.AuthorityPrepared,
	}); err != nil {
		t.Fatalf("UpdateAuthority error = %v", err)
	}
	workspaceRoot := "/home/ivan/.local/state/ax/materializations/" + testMatID + "/workspace_relux"
	if _, err := store.UpdateAuthority(testMatID, "workspace_relux", matjournal.AuthorityState{
		RootPath: testRootWorkspace, CompletedSequences: []uint64{1}, RollbackRoot: &workspaceRoot, State: matjournal.AuthorityPrepared,
	}); err != nil {
		t.Fatalf("UpdateAuthority error = %v", err)
	}
	if _, err := store.Transition(testMatID, matjournal.PhaseValidating, matjournal.TransitionOpts{}); err != nil {
		t.Fatalf("Transition error = %v", err)
	}
}

// driveJournalToPrepared crashes after the bridge open or after the
// host commit enters prepared.
func driveJournalToPrepared(t *testing.T, boundary, path, recoverPath, closure, variant string) ConformanceRecord {
	t.Helper()
	store, dataDir := journalDirs(t)
	mustCreate(t, store, closureInputs(t, closure))
	interrupted := errors.New("crash entering prepared")
	switch variant {
	case "bridge_open", "bridge_open_uncertain":
		if _, err := store.UpdateTaskBoard(testMatID, importedBoard()); err != nil {
			t.Fatal(err)
		}
		store.AfterJournal = func() error { return interrupted }
		if _, err := store.UpdateTaskBoard(testMatID, openedBoard()); !errors.Is(err, interrupted) {
			t.Fatalf("UpdateTaskBoard fault = %v, want the interior fault", err)
		}
		restarted := reopenJournal(t, dataDir)
		loaded, _, err := restarted.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.TaskBoard.State != matjournal.BoardOpened {
			t.Fatalf("task-board = %q", loaded.TaskBoard.State)
		}
		if variant == "bridge_open_uncertain" {
			// The provider prepare may have executed unrecorded
			// and its status is unknown: recovery parks
			// instead of guessing.
			evidence := recoverMust(t, restarted, matjournal.BoundaryAfterOpen, recoverPath, func(input *matjournal.RecoveryInput) {
				input.Provider.State = matjournal.ProviderUnknown
				input.Bridge.State = matjournal.BoardOpened
			}, matjournal.OutcomeRecoverableParked, "Recover("+boundary+")")
			return recordFromEvidence(boundary, path, matjournal.BoundaryAfterOpen, closure, variant, DriverJournalToPrepared, evidence)
		}
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterOpen, recoverPath, func(input *matjournal.RecoveryInput) {
			input.Bridge.State = matjournal.BoardOpened
		}, matjournal.OutcomeSafeRetry, "Recover("+boundary+")")
		return recordFromEvidence(boundary, path, matjournal.BoundaryAfterOpen, closure, variant, DriverJournalToPrepared, evidence)
	case "host_prepared", "direct_host", "direct_provider_host":
		switch closure {
		case "composite":
			walkCompositeToValidating(t, store)
		case "direct_provider":
			walkDirectProviderToValidating(t, store)
		default:
			if _, err := store.Transition(testMatID, matjournal.PhaseValidating, matjournal.TransitionOpts{}); err != nil {
				t.Fatal(err)
			}
		}
		store.AfterJournal = func() error { return interrupted }
		if _, err := store.Transition(testMatID, matjournal.PhasePrepared, matjournal.TransitionOpts{}); !errors.Is(err, interrupted) {
			t.Fatalf("Transition fault = %v, want the interior fault", err)
		}
		restarted := reopenJournal(t, dataDir)
		loaded, _, err := restarted.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.Phase != matjournal.PhasePrepared {
			t.Fatalf("phase = %q, want prepared", loaded.Phase)
		}
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterOpen, recoverPath, func(input *matjournal.RecoveryInput) {
			switch closure {
			case "composite":
				input.Bridge.State = matjournal.BoardOpened
			case "direct":
				input.Provider.State = matjournal.ProviderUnknown
			}
		}, matjournal.OutcomeSafeRetry, "Recover("+boundary+")")
		return recordFromEvidence(boundary, path, matjournal.BoundaryAfterOpen, closure, variant, DriverJournalToPrepared, evidence)
	default:
		t.Fatalf("unknown to-prepared variant %q", variant)
		return ConformanceRecord{}
	}
}

// activationSetup drives a composite journal to the committing
// phase: the pre-activation durable state.
func activationSetup(t *testing.T) (*matjournal.Store, string) {
	t.Helper()
	store, dataDir := journalDirs(t)
	mustCreate(t, store, compositeInputs())
	convergePrepared(t, store)
	if _, err := store.Transition(testMatID, matjournal.PhaseCommitting, matjournal.TransitionOpts{}); err != nil {
		t.Fatal(err)
	}
	return store, dataDir
}

// driveJournalActivation crashes after the adopt is recorded with
// agreeing status, or after activation may have occurred with the
// result uncertain.
func driveJournalActivation(t *testing.T, boundary, path, recoverPath, variant string) ConformanceRecord {
	t.Helper()
	const closure = "composite"
	store, dataDir := activationSetup(t)
	switch variant {
	case "adopted_reconciled":
		interrupted := errors.New("crash after adopt recorded")
		store.AfterJournal = func() error { return interrupted }
		if _, err := store.UpdateTaskBoard(testMatID, adoptedBoard()); !errors.Is(err, interrupted) {
			t.Fatalf("UpdateTaskBoard fault = %v, want the interior fault", err)
		}
		restarted := reopenJournal(t, dataDir)
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterActivation, recoverPath, func(input *matjournal.RecoveryInput) {
			input.Bridge.State = matjournal.BoardAdopted
			input.Bridge.LeaseActive = true
			input.Bridge.BindingMatches = true
			input.Bridge.ManagerMatches = true
			input.NativeKnown = true
			input.NativeBefore, input.NativeAfter = testNative, testNative
		}, matjournal.OutcomeSafeRetry, "Recover("+boundary+")")
		if evidence.OperationIDs["adopt_operation_id"] != testAdoptOp {
			t.Fatalf("operation IDs = %+v", evidence.OperationIDs)
		}
		return recordFromEvidence(boundary, path, matjournal.BoundaryAfterActivation, closure, variant, DriverJournalActivation, evidence)
	case "activation_uncertain":
		restarted := reopenJournal(t, dataDir)
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterActivation, recoverPath, func(input *matjournal.RecoveryInput) {
			input.Bridge.State = matjournal.ProviderUnknown
		}, matjournal.OutcomeRecoverableParked, "Recover("+boundary+")")
		return recordFromEvidence(boundary, path, matjournal.BoundaryAfterActivation, closure, variant, DriverJournalActivation, evidence)
	default:
		t.Fatalf("unknown activation variant %q", variant)
		return ConformanceRecord{}
	}
}

// ownershipSetup drives a task-board-only ownership journal to the
// committing phase with its authority converged.
func ownershipSetup(t *testing.T) (*matjournal.Store, string) {
	t.Helper()
	store, dataDir := journalDirs(t)
	mustCreate(t, store, boardOwnershipInputs())
	if _, err := store.UpdateTaskBoard(testMatID, importedBoard()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateTaskBoard(testMatID, openedBoard()); err != nil {
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
	return store, dataDir
}

// driveJournalFinalize crashes after the resumed state is recorded
// and completes the commit, parks when finalize is unproven, or
// resumes a direct journal whose plan carries no finalize effect.
func driveJournalFinalize(t *testing.T, boundary, path, recoverPath, closure, variant string) ConformanceRecord {
	t.Helper()
	switch variant {
	case "resumed_completes", "finalize_uncertain":
		store, dataDir := ownershipSetup(t)
		if variant == "resumed_completes" {
			if _, err := store.UpdateTaskBoard(testMatID, adoptedBoard()); err != nil {
				t.Fatal(err)
			}
			interrupted := errors.New("crash after resume recorded")
			store.AfterJournal = func() error { return interrupted }
			if _, err := store.UpdateTaskBoard(testMatID, resumedBoard()); !errors.Is(err, interrupted) {
				t.Fatalf("UpdateTaskBoard fault = %v, want the interior fault", err)
			}
			restarted := reopenJournal(t, dataDir)
			evidence := recoverMust(t, restarted, matjournal.BoundaryAfterFinalize, recoverPath, func(input *matjournal.RecoveryInput) {
				input.Provider.State = matjournal.ProviderUnknown
				input.Bridge.State = "running"
				input.Bridge.LeaseActive = true
				input.Bridge.BindingMatches = true
				input.Bridge.ManagerMatches = true
				input.NativeKnown = true
				input.NativeBefore, input.NativeAfter = testNative, testNative
			}, matjournal.OutcomeSafeRetry, "Recover("+boundary+")")
			loaded, _, err := restarted.Get(testMatID)
			if err != nil {
				t.Fatal(err)
			}
			if loaded.Phase != matjournal.PhaseCommitted {
				t.Fatalf("phase = %q, want the completed commit", loaded.Phase)
			}
			return recordFromEvidence(boundary, path, matjournal.BoundaryAfterFinalize, closure, variant, DriverJournalFinalize, evidence)
		}
		restarted := reopenJournal(t, dataDir)
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterFinalize, recoverPath, func(input *matjournal.RecoveryInput) {
			input.Provider.State = matjournal.ProviderUnknown
			input.Bridge.State = matjournal.ProviderUnknown
		}, matjournal.OutcomeRecoverableParked, "Recover("+boundary+")")
		return recordFromEvidence(boundary, path, matjournal.BoundaryAfterFinalize, closure, variant, DriverJournalFinalize, evidence)
	case "direct_resume":
		store, dataDir := journalDirs(t)
		mustCreate(t, store, directInputs())
		if _, err := store.Transition(testMatID, matjournal.PhaseValidating, matjournal.TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(testMatID, matjournal.PhasePrepared, matjournal.TransitionOpts{}); err != nil {
			t.Fatal(err)
		}
		restarted := reopenJournal(t, dataDir)
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterFinalize, recoverPath, func(input *matjournal.RecoveryInput) {
			input.Provider.State = matjournal.ProviderUnknown
		}, matjournal.OutcomeSafeRetry, "Recover("+boundary+")")
		loaded, _, err := restarted.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.Phase != matjournal.PhasePrepared || loaded.PrepareOperationID != testPrepareOp {
			t.Fatalf("phase = %q prepare = %q, want the unchanged prepared journal", loaded.Phase, loaded.PrepareOperationID)
		}
		return recordFromEvidence(boundary, path, matjournal.BoundaryAfterFinalize, closure, variant, DriverJournalFinalize, evidence)
	default:
		t.Fatalf("unknown finalize variant %q", variant)
		return ConformanceRecord{}
	}
}

// driveJournalDormant crashes on the passive path: after the dormant
// prepare, or after the dormant finalize may have occurred.
func driveJournalDormant(t *testing.T, boundary, path, variant string) ConformanceRecord {
	t.Helper()
	const closure = "board_dormant"
	const recoverPath = matjournal.PathPassiveSync
	store, dataDir := journalDirs(t)
	mustCreate(t, store, boardDormantInputs())
	switch variant {
	case "dormant_prepare":
		_ = store
		faulty, replayDir := journalDirs(t)
		interrupted := errors.New("crash after the dormant journal before its receipt")
		faulty.AfterJournal = func() error { return interrupted }
		if _, _, err := faulty.Create(boardDormantInputs()); !errors.Is(err, interrupted) {
			t.Fatalf("Create fault = %v, want the interior fault", err)
		}
		if journal, receipt := journalFilesExist(t, replayDir); !journal || receipt {
			t.Fatalf("after journal fault: journal = %v, receipt = %v, want journal only", journal, receipt)
		}
		restarted := reopenJournal(t, replayDir)
		ref, _ := mustCreate(t, restarted, boardDormantInputs())
		if ref.PrepareOperationID != testPrepareOp {
			t.Fatalf("replay PrepareOperationID = %q", ref.PrepareOperationID)
		}
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterPrepare, recoverPath, func(input *matjournal.RecoveryInput) {
			input.Provider.State = matjournal.ProviderUnknown
		}, matjournal.OutcomeSafeRetry, "Recover("+boundary+")")
		return recordFromEvidence(boundary, path, matjournal.BoundaryAfterPrepare, closure, variant, DriverJournalDormant, evidence)
	case "dormant_finalize":
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
		restarted := reopenJournal(t, dataDir)
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterFinalize, recoverPath, func(input *matjournal.RecoveryInput) {
			input.Provider.State = matjournal.ProviderUnknown
			input.Bridge.State = "dormant"
			input.Bridge.ManagerMatches = true
		}, matjournal.OutcomeSafeRetry, "Recover("+boundary+")")
		loaded, _, err := restarted.Get(testMatID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.Phase != matjournal.PhaseCommitted || loaded.TaskBoard.State != matjournal.BoardDormantFinalized {
			t.Fatalf("phase = %q bridge = %q, want the closed dormant finalize", loaded.Phase, loaded.TaskBoard.State)
		}
		if loaded.TaskBoard.OpenToken != nil {
			t.Fatalf("dormant finalize keeps the open token")
		}
		return recordFromEvidence(boundary, path, matjournal.BoundaryAfterFinalize, closure, variant, DriverJournalDormant, evidence)
	default:
		t.Fatalf("unknown dormant variant %q", variant)
		return ConformanceRecord{}
	}
}

// conformanceRows enumerates every executed crash boundary row. The
// MAT rows use the owner_resume evaluation path (the canonical path
// of the owner suites); mapped rows use their flow's path, so all
// five evaluation paths appear across the table.
func conformanceRows() []conformanceRow {
	checkpoint := func(boundary, path, seam string) conformanceRow {
		return conformanceRow{
			name: boundary + "_" + path + "_" + seam, boundary: boundary,
			evaluatedVia: "capture-blob-receipt-seam", path: path, closure: path,
			variant: seam, driver: DriverCheckpointCapture,
			run: func(t *testing.T) ConformanceRecord {
				return driveCheckpointCapture(t, boundary, path, seam)
			},
		}
	}
	create := func(boundary, evaluatedVia, path, recoverPath, closure, seam string) conformanceRow {
		return conformanceRow{
			name: boundary + "_" + path + "_" + seam, boundary: boundary,
			evaluatedVia: evaluatedVia, path: path, recoverPath: recoverPath,
			closure: closure, variant: seam, driver: DriverJournalCreate,
			run: func(t *testing.T) ConformanceRecord {
				return driveJournalCreate(t, boundary, evaluatedVia, path, recoverPath, closure, seam)
			},
		}
	}
	transfer := func(boundary, path, recoverPath, closure string) conformanceRow {
		return conformanceRow{
			name: boundary + "_" + path + "_transfer", boundary: boundary,
			evaluatedVia: matjournal.BoundaryAfterTransfer, path: path, recoverPath: recoverPath,
			closure: closure, variant: "after_transfer", driver: DriverJournalTransfer,
			run: func(t *testing.T) ConformanceRecord {
				return driveJournalTransfer(t, boundary, path, recoverPath, closure)
			},
		}
	}
	validate := func(boundary, path, recoverPath, closure string) conformanceRow {
		return conformanceRow{
			name: boundary + "_" + path + "_validation", boundary: boundary,
			evaluatedVia: matjournal.BoundaryAfterValidation, path: path, recoverPath: recoverPath,
			closure: closure, variant: "after_validation", driver: DriverJournalValidate,
			run: func(t *testing.T) ConformanceRecord {
				return driveJournalValidate(t, boundary, path, recoverPath, closure)
			},
		}
	}
	prepareOp := func(boundary, path, recoverPath, variant string) conformanceRow {
		return conformanceRow{
			name: boundary + "_" + path + "_" + variant, boundary: boundary,
			evaluatedVia: matjournal.BoundaryAfterPrepareOp, path: path, recoverPath: recoverPath,
			closure: "composite", variant: variant, driver: DriverJournalPrepareOp,
			run: func(t *testing.T) ConformanceRecord {
				return driveJournalPrepareOp(t, boundary, path, recoverPath, variant)
			},
		}
	}
	toPrepared := func(boundary, path, recoverPath, closure, variant string) conformanceRow {
		return conformanceRow{
			name: boundary + "_" + path + "_" + variant, boundary: boundary,
			evaluatedVia: matjournal.BoundaryAfterOpen, path: path, recoverPath: recoverPath,
			closure: closure, variant: variant, driver: DriverJournalToPrepared,
			run: func(t *testing.T) ConformanceRecord {
				return driveJournalToPrepared(t, boundary, path, recoverPath, closure, variant)
			},
		}
	}
	activation := func(boundary, path, recoverPath, variant string) conformanceRow {
		return conformanceRow{
			name: boundary + "_" + path + "_" + variant, boundary: boundary,
			evaluatedVia: matjournal.BoundaryAfterActivation, path: path, recoverPath: recoverPath,
			closure: "composite", variant: variant, driver: DriverJournalActivation,
			run: func(t *testing.T) ConformanceRecord {
				return driveJournalActivation(t, boundary, path, recoverPath, variant)
			},
		}
	}
	finalize := func(boundary, path, recoverPath, closure, variant string) conformanceRow {
		return conformanceRow{
			name: boundary + "_" + path + "_" + variant, boundary: boundary,
			evaluatedVia: matjournal.BoundaryAfterFinalize, path: path, recoverPath: recoverPath,
			closure: closure, variant: variant, driver: DriverJournalFinalize,
			run: func(t *testing.T) ConformanceRecord {
				return driveJournalFinalize(t, boundary, path, recoverPath, closure, variant)
			},
		}
	}
	dormant := func(boundary, path, variant string) conformanceRow {
		return conformanceRow{
			name: boundary + "_" + path + "_" + variant, boundary: boundary,
			evaluatedVia: map[string]string{"dormant_prepare": matjournal.BoundaryAfterPrepare, "dormant_finalize": matjournal.BoundaryAfterFinalize}[variant],
			path:         path, recoverPath: matjournal.PathPassiveSync,
			closure: "board_dormant", variant: variant, driver: DriverJournalDormant,
			run: func(t *testing.T) ConformanceRecord {
				return driveJournalDormant(t, boundary, path, variant)
			},
		}
	}
	resume := matjournal.PathOwnerResume
	return []conformanceRow{
		// Checkpoint-capture boundaries: the full seam matrix on
		// CR-STOP-02, the post-write seams on the GRACE rows.
		checkpoint("CR-STOP-02", PathDirect, "before_write"),
		checkpoint("CR-STOP-02", PathDirect, "after_blob"),
		checkpoint("CR-STOP-02", PathDirect, "after_commit"),
		checkpoint("CR-STOP-02", PathTaskBoard, "before_write"),
		checkpoint("CR-STOP-02", PathTaskBoard, "after_blob"),
		checkpoint("CR-STOP-02", PathTaskBoard, "after_commit"),
		checkpoint("CR-GRACE-04", PathDirect, "after_blob"),
		checkpoint("CR-GRACE-04", PathDirect, "after_commit"),
		checkpoint("CR-GRACE-04", PathTaskBoard, "after_blob"),
		checkpoint("CR-GRACE-04", PathTaskBoard, "after_commit"),
		checkpoint("CR-GRACE-08", PathDirect, "after_blob"),
		checkpoint("CR-GRACE-08", PathDirect, "after_commit"),
		checkpoint("CR-GRACE-08", PathTaskBoard, "after_blob"),
		checkpoint("CR-GRACE-08", PathTaskBoard, "after_commit"),
		// CR-MAT-01..02 and the mapped prepare rows.
		create("CR-MAT-01", matjournal.BoundaryBeforeCreate, PathDirect, resume, "direct", "before_write"),
		create("CR-MAT-01", matjournal.BoundaryBeforeCreate, PathTaskBoard, resume, "composite", "before_write"),
		create("CR-MAT-02", matjournal.BoundaryAfterPrepare, PathDirect, resume, "direct", "after_journal"),
		create("CR-MAT-02", matjournal.BoundaryAfterPrepare, PathDirect, resume, "direct", "after_commit"),
		create("CR-MAT-02", matjournal.BoundaryAfterPrepare, PathTaskBoard, resume, "composite", "after_journal"),
		create("CR-MAT-02", matjournal.BoundaryAfterPrepare, PathTaskBoard, resume, "composite", "after_commit"),
		create("CR-GRACE-01", matjournal.BoundaryAfterPrepare, PathDirect, matjournal.PathGracefulTakeover, "direct", "after_journal"),
		create("CR-GRACE-01", matjournal.BoundaryAfterPrepare, PathTaskBoard, matjournal.PathGracefulTakeover, "composite", "after_journal"),
		create("CR-RESUME-02", matjournal.BoundaryAfterPrepare, PathDirect, resume, "direct", "after_journal"),
		create("CR-RESUME-02", matjournal.BoundaryAfterPrepare, PathTaskBoard, resume, "composite", "after_journal"),
		create("CR-FORCE-D-01", matjournal.BoundaryAfterPrepare, PathDirect, matjournal.PathForceTakeover, "direct", "after_journal"),
		create("CR-FORCE-TB-01", matjournal.BoundaryAfterPrepare, PathTaskBoard, matjournal.PathForceTakeover, "composite", "after_journal"),
		// CR-MAT-03..04 transfer and validation rows.
		transfer("CR-MAT-03", PathDirect, resume, "direct"),
		transfer("CR-MAT-03", PathTaskBoard, resume, "composite"),
		validate("CR-MAT-04", PathDirect, resume, "direct"),
		validate("CR-MAT-04", PathTaskBoard, resume, "composite"),
		// CR-MAT-05 prepare-op rows with the failed-import
		// rollback variant.
		prepareOp("CR-MAT-05", PathTaskBoard, resume, "provider_half"),
		prepareOp("CR-MAT-05", PathTaskBoard, resume, "bridge_half"),
		prepareOp("CR-MAT-05", PathTaskBoard, resume, "rollback_failed"),
		// To-prepared rows: CR-MAT-06 in full, one host variant
		// per mapped row.
		toPrepared("CR-MAT-06", PathTaskBoard, resume, "composite", "bridge_open"),
		toPrepared("CR-MAT-06", PathTaskBoard, resume, "composite", "bridge_open_uncertain"),
		toPrepared("CR-MAT-06", PathTaskBoard, resume, "composite", "host_prepared"),
		toPrepared("CR-MAT-06", PathDirect, resume, "direct", "direct_host"),
		toPrepared("CR-RESUME-03", PathDirect, resume, "direct", "direct_host"),
		toPrepared("CR-RESUME-03", PathTaskBoard, resume, "composite", "host_prepared"),
		toPrepared("CR-GRACE-09", PathDirect, matjournal.PathGracefulTakeover, "direct", "direct_host"),
		toPrepared("CR-GRACE-09", PathTaskBoard, matjournal.PathGracefulTakeover, "composite", "host_prepared"),
		toPrepared("CR-FORK-05", PathDirect, matjournal.PathFork, "direct", "direct_host"),
		toPrepared("CR-FORK-05", PathTaskBoard, matjournal.PathFork, "composite", "host_prepared"),
		toPrepared("CR-FORCE-D-02", PathDirect, matjournal.PathForceTakeover, "direct_provider", "direct_provider_host"),
		toPrepared("CR-FORCE-TB-02", PathTaskBoard, matjournal.PathForceTakeover, "composite", "host_prepared"),
		// Activation rows: reconciled and uncertain halves.
		activation("CR-MAT-07", PathTaskBoard, resume, "adopted_reconciled"),
		activation("CR-MAT-07", PathTaskBoard, resume, "activation_uncertain"),
		activation("CR-GRACE-11", PathTaskBoard, matjournal.PathGracefulTakeover, "adopted_reconciled"),
		activation("CR-GRACE-11", PathTaskBoard, matjournal.PathGracefulTakeover, "activation_uncertain"),
		activation("CR-RESUME-05", PathTaskBoard, resume, "adopted_reconciled"),
		activation("CR-RESUME-05", PathTaskBoard, resume, "activation_uncertain"),
		activation("CR-FORK-07", PathTaskBoard, matjournal.PathFork, "adopted_reconciled"),
		activation("CR-FORK-07", PathTaskBoard, matjournal.PathFork, "activation_uncertain"),
		activation("CR-FORCE-TB-03", PathTaskBoard, matjournal.PathForceTakeover, "adopted_reconciled"),
		activation("CR-FORCE-TB-03", PathTaskBoard, matjournal.PathForceTakeover, "activation_uncertain"),
		// Finalize rows: completion, uncertain, and direct
		// resume halves.
		finalize("CR-MAT-08", PathTaskBoard, resume, "board_ownership", "resumed_completes"),
		finalize("CR-MAT-08", PathTaskBoard, resume, "board_ownership", "finalize_uncertain"),
		finalize("CR-MAT-08", PathDirect, resume, "direct", "direct_resume"),
		finalize("CR-GRACE-12", PathTaskBoard, matjournal.PathGracefulTakeover, "board_ownership", "resumed_completes"),
		finalize("CR-GRACE-12", PathTaskBoard, matjournal.PathGracefulTakeover, "board_ownership", "finalize_uncertain"),
		finalize("CR-GRACE-12", PathDirect, matjournal.PathGracefulTakeover, "direct", "direct_resume"),
		finalize("CR-RESUME-06", PathTaskBoard, resume, "board_ownership", "resumed_completes"),
		finalize("CR-RESUME-06", PathTaskBoard, resume, "board_ownership", "finalize_uncertain"),
		finalize("CR-RESUME-06", PathDirect, resume, "direct", "direct_resume"),
		finalize("CR-FORK-08", PathTaskBoard, matjournal.PathFork, "board_ownership", "resumed_completes"),
		finalize("CR-FORK-08", PathTaskBoard, matjournal.PathFork, "board_ownership", "finalize_uncertain"),
		finalize("CR-FORK-08", PathDirect, matjournal.PathFork, "direct", "direct_resume"),
		finalize("CR-FORCE-TB-04", PathTaskBoard, matjournal.PathForceTakeover, "board_ownership", "resumed_completes"),
		finalize("CR-FORCE-TB-04", PathTaskBoard, matjournal.PathForceTakeover, "board_ownership", "finalize_uncertain"),
		finalize("CR-FORCE-D-05", PathDirect, matjournal.PathForceTakeover, "direct", "direct_resume"),
		// Passive sync dormant rows.
		dormant("CR-SYNC-06", PathTaskBoard, "dormant_prepare"),
		dormant("CR-SYNC-07", PathTaskBoard, "dormant_finalize"),
	}
}

// TestCrashGateConformance executes the Section 13.13 outcome gate:
// a crash after every reachable boundary's durable write (and after
// every external effect that may have happened), a clean restart,
// and exactly one outcome with the full conformance record. Each
// row writes its machine-readable record for the evidence tarball.
func TestCrashGateConformance(t *testing.T) {
	rows := conformanceRows()
	assertConformanceCoverage(t, rows)
	dir := conformanceDir(t)
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			record := row.run(t)
			if record.Boundary != row.boundary || record.RegistryPath != row.path ||
				record.EvaluatedVia != row.evaluatedVia || record.Closure != row.closure ||
				record.Variant != row.variant || record.Driver != row.driver {
				t.Fatalf("row %s record metadata = %+v, want the row identity", row.name, record)
			}
			if row.recoverPath != "" && record.RecoverPath != row.recoverPath {
				t.Fatalf("row %s recover path = %q, want %q", row.name, record.RecoverPath, row.recoverPath)
			}
			mustRecordComplete(t, record, row.name)
			writeRecord(t, dir, row.name, record)
		})
	}
}

// TestCrashGateNotApplicable audits the NOT APPLICABLE classification:
// every unreachable path names its exact owner, carries its note,
// and has no conformance row. The audit output enumerates the
// classification for the log evidence.
func TestCrashGateNotApplicable(t *testing.T) {
	rows := conformanceRows()
	driven := map[string]struct{}{}
	for _, row := range rows {
		driven[row.boundary+"\x00"+row.path] = struct{}{}
	}
	count := 0
	for _, boundary := range Registry() {
		for _, path := range boundary.Paths {
			if path.Reachable {
				continue
			}
			count++
			if path.Owner == "" || path.Note == "" || path.Driver != "" {
				t.Fatalf("boundary %s path %s misclassifies NOT APPLICABLE: %+v", boundary.ID, path.Name, path)
			}
			if _, ok := driven[boundary.ID+"\x00"+path.Name]; ok {
				t.Fatalf("boundary %s path %s is NOT APPLICABLE and still has a conformance row", boundary.ID, path.Name)
			}
		}
	}
	if count == 0 {
		t.Fatal("no NOT APPLICABLE paths classified")
	}
	t.Logf("%d NOT APPLICABLE paths recorded with owners; none driven", count)
}
