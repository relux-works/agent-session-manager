//go:build darwin || linux

package crashgate

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/matjournal"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// Real process-termination evidence for the two durable-write seams
// the gate depends on: the child runs the genuine production entry
// (sessckpt Capture, matjournal Create) and SIGKILLs itself between
// the bytes install and the operation receipt. The parent then
// proves the interrupted operation left complete verifying bytes
// with no receipt behind, and the identical retry converges with
// its receipt and a safe_retry classification.
//
// The children rebuild nothing: they open the parent's directories
// from the environment, so their inputs are byte-identical to the
// parent's retry by construction.
const (
	sigChildEnv   = "AX_CRASHGATE_SIG_CHILD"
	sigChildRepo  = "AX_CRASHGATE_SIG_REPO"
	sigChildStore = "AX_CRASHGATE_SIG_STORE"
	sigChildJDir  = "AX_CRASHGATE_SIG_JDIR"
)

func TestRealSIGKILLSeams(t *testing.T) {
	switch os.Getenv(sigChildEnv) {
	case "checkpoint":
		sigCheckpointChild()
		return
	case "journal":
		sigJournalChild()
		return
	}
	t.Run("checkpoint_capture", func(t *testing.T) {
		repoDir, _ := chainFixture(t)
		_, storeDir := checkpointDirs(t)
		spawnSIGChild(t, "checkpoint", []string{sigChildRepo + "=" + repoDir, sigChildStore + "=" + storeDir})
		// The kill landed between the blob and the receipt: one
		// complete blob that verifies, no receipt, no partial
		// bytes.
		if blobs, receipts := countCheckpointFiles(t, storeDir); blobs != 1 || receipts != 0 {
			t.Fatalf("after SIGKILL: blobs = %d, receipts = %d, want 1, 0", blobs, receipts)
		}
		everyCheckpointVerifies(t, storeDir)
		chain := mustOpenRepo(t, repoDir)
		events, err := chain.ListEvents(testSessionID)
		if err != nil || len(events) != 1 {
			t.Fatalf("chain events = %d, want the single bootstrap event", len(events))
		}
		restarted := reopenCheckpoint(t, storeDir)
		ref, raw, err := restarted.Capture(chain, captureInputs([]string{events[0].EventID}, testOpA))
		if err != nil {
			t.Fatalf("retry Capture error = %v", err)
		}
		stored, err := restarted.Get(ref.CheckpointID)
		if err != nil {
			t.Fatalf("Get after retry error = %v", err)
		}
		if string(stored) != string(raw) {
			t.Fatalf("retry bytes differ from stored bytes")
		}
		if blobs, receipts := countCheckpointFiles(t, storeDir); blobs != 1 || receipts != 1 {
			t.Fatalf("after retry: blobs = %d, receipts = %d, want 1, 1", blobs, receipts)
		}
		lease := leaseString(matjournal.LeaseIdentity{Epoch: 1, ID: testLeaseID})
		writeRecord(t, conformanceDir(t), "CR-STOP-02_direct_sigkill_after_blob", ConformanceRecord{
			Boundary: "CR-STOP-02", RegistryPath: PathDirect, EvaluatedVia: "capture-blob-receipt-seam",
			Closure: PathDirect, Variant: "sigkill_after_blob", Driver: DriverCheckpointCapture,
			OperationIDs: map[string]string{"capture_operation_id": testOpA},
			PhaseBefore:  "blobs=1 receipts=0", PhaseAfter: "blobs=1 receipts=1 checkpoint=" + ref.CheckpointID,
			ReceiptPresent: true, ExternalEffect: "none (capture performs no provider or task-board I/O)",
			StatusProbe: "real SIGKILL between blob install and receipt; parent verified signal death",
			LeaseBefore: lease, LeaseAfter: lease,
			NativeBefore: testProviderManifest, NativeAfter: testProviderManifest,
			Selected:    string(matjournal.OutcomeSafeRetry),
			Reason:      "real SIGKILL left one verifying blob with no receipt; the identical retry converged with its receipt",
			Remediation: "retry the same capture operation with byte-identical inputs",
		})
	})
	t.Run("journal_create", func(t *testing.T) {
		_, dataDir := journalDirs(t)
		spawnSIGChild(t, "journal", []string{sigChildJDir + "=" + dataDir})
		// The kill landed between the journal and the receipt: one
		// complete journal that verifies, no receipt.
		if journal, receipt := journalFilesExist(t, dataDir); !journal || receipt {
			t.Fatalf("after SIGKILL: journal = %v, receipt = %v, want true, false", journal, receipt)
		}
		restarted := reopenJournal(t, dataDir)
		loaded, _, err := restarted.Get(testMatID)
		if err != nil {
			t.Fatalf("Get after SIGKILL error = %v", err)
		}
		if loaded.Phase != matjournal.PhaseStaging {
			t.Fatalf("phase = %q", loaded.Phase)
		}
		ref, _ := mustCreate(t, restarted, compositeInputs())
		if ref.PrepareOperationID != testPrepareOp {
			t.Fatalf("retry PrepareOperationID = %q", ref.PrepareOperationID)
		}
		if journal, receipt := journalFilesExist(t, dataDir); !journal || !receipt {
			t.Fatalf("after retry: journal = %v, receipt = %v, want true, true", journal, receipt)
		}
		evidence := recoverMust(t, restarted, matjournal.BoundaryAfterPrepare, matjournal.PathOwnerResume, nil, matjournal.OutcomeSafeRetry, "Recover(CR-MAT-02 SIGKILL)")
		record := recordFromEvidence("CR-MAT-02", PathTaskBoard, matjournal.BoundaryAfterPrepare, "composite", "sigkill_after_journal", DriverJournalCreate, evidence)
		record.StatusProbe = "real SIGKILL between journal install and prepare receipt; " + record.StatusProbe
		mustRecordComplete(t, record, "CR-MAT-02 SIGKILL")
		writeRecord(t, conformanceDir(t), "CR-MAT-02_task_board_sigkill_after_journal", record)
	})
}

// spawnSIGChild runs the test binary as the named crash child and
// requires death by SIGKILL. A bounded context keeps a hung child
// a test failure, never a hung suite.
func spawnSIGChild(t *testing.T, mode string, env []string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, os.Args[0],
		"-test.run=^TestRealSIGKILLSeams$",
		"-test.count=1",
	)
	child.Env = append(os.Environ(), append([]string{sigChildEnv + "=" + mode}, env...)...)
	runErr := child.Run()
	var exitErr *exec.ExitError
	if !errors.As(runErr, &exitErr) {
		t.Fatalf("child run = %v, want a signal-kill exit", runErr)
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
		t.Fatalf("child status = %v, want killed by SIGKILL", exitErr)
	}
}

// sigCheckpointChild runs the production Capture entry and kills
// its own process at the AfterBlob boundary. It never returns.
func sigCheckpointChild() {
	repo, err := sessrepo.Open(os.Getenv(sigChildRepo))
	if err != nil {
		os.Exit(64)
	}
	events, err := repo.ListEvents(testSessionID)
	if err != nil || len(events) != 1 {
		os.Exit(65)
	}
	store, err := openCheckpointEnv(sigChildStore)
	if err != nil {
		os.Exit(66)
	}
	store.AfterBlob = func() error {
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGKILL)
		select {}
	}
	_, _, _ = store.Capture(repo, captureInputs([]string{events[0].EventID}, testOpA))
	os.Exit(0)
}

// sigJournalChild runs the production Create entry and kills its own
// process at the AfterJournal boundary. It never returns.
func sigJournalChild() {
	store, err := openJournalEnv(sigChildJDir)
	if err != nil {
		os.Exit(66)
	}
	store.AfterJournal = func() error {
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGKILL)
		select {}
	}
	_, _, _ = store.Create(compositeInputs())
	os.Exit(0)
}
