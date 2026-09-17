//go:build darwin || linux

package sessckpt

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// Real process-termination evidence for the crash boundary: the
// child runs the genuine production Capture entry and SIGKILLs
// itself between the checkpoint-blob install and the operation
// receipt. The parent then proves the interrupted capture left no
// partial record behind and the identical retry converges.
//
// The child rebuilds nothing: it opens the parent's repository
// and store directories from the environment and derives the same
// head from the chain, so its closure inputs are byte-identical
// to the parent's retry by construction.
const (
	crashChildEnv   = "AX_SESSCKPT_CRASH_CHILD"
	crashChildRepo  = "AX_SESSCKPT_CRASH_REPO"
	crashChildStore = "AX_SESSCKPT_CRASH_STORE"
)

func TestCaptureCrashChildSelfTerminates(t *testing.T) {
	if os.Getenv(crashChildEnv) == "1" {
		captureCrashChild()
		return
	}
	repoDir := t.TempDir()
	repo, err := sessrepo.Open(repoDir)
	if err != nil {
		t.Fatal(err)
	}
	record := decodeObject(t, specSessionRecord)
	record["session_id"], record["subject_id"], record["name"] = testSessionID, testSessionID, "alpha"
	identified := identify(t, record, "record_id")
	ref, err := repo.CreateSession(identified)
	if err != nil {
		t.Fatal(err)
	}
	_ = appendChainEvent(t, repo, "session.created", 1, testLease, 1, ref.RecordID, map[string]any{
		"session_record_id": ref.RecordID, "bootstrap_operation_id": testHostA, "first_checkpoint_operation_id": testHostB,
	})
	storeDir := t.TempDir()
	if _, err := Open(storeDir); err != nil {
		t.Fatal(err)
	}
	// A bounded context keeps a hung child a test failure,
	// never a hung suite: the child must die at the boundary or
	// not return at all.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, os.Args[0],
		"-test.run=^TestCaptureCrashChildSelfTerminates$",
		"-test.count=1",
	)
	child.Env = append(os.Environ(),
		crashChildEnv+"=1",
		crashChildRepo+"="+repoDir,
		crashChildStore+"="+storeDir,
	)
	runErr := child.Run()
	var exitErr *exec.ExitError
	if !errors.As(runErr, &exitErr) {
		t.Fatalf("child run = %v, want a signal-kill exit", runErr)
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
		t.Fatalf("child status = %v, want killed by SIGKILL", exitErr)
	}
	// The kill landed between the blob and the receipt: one
	// complete blob that verifies, no receipt, no partial bytes.
	parent, err := Open(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	if blobs, receipts := countFiles(t, parent); blobs != 1 || receipts != 0 {
		t.Fatalf("after SIGKILL: blobs = %d, receipts = %d, want 1, 0", blobs, receipts)
	}
	everyFileVerifies(t, parent)
	// The identical retry converges on one checkpoint identity
	// with its receipt: the interrupted capture is resumed, not
	// duplicated.
	chain, err := sessrepo.Open(repoDir)
	if err != nil {
		t.Fatal(err)
	}
	events, err := chain.ListEvents(testSessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("chain events = %d, want the single bootstrap event", len(events))
	}
	recovered, raw, err := parent.Capture(chain, testInputs([]string{events[0].EventID}, testOpA))
	if err != nil {
		t.Fatalf("retry Capture error = %v", err)
	}
	if recovered.OperationID != testOpA {
		t.Fatalf("retry OperationID = %q", recovered.OperationID)
	}
	stored, err := parent.Get(recovered.CheckpointID)
	if err != nil {
		t.Fatalf("Get after retry error = %v", err)
	}
	if string(stored) != string(raw) {
		t.Fatalf("retry bytes differ from stored bytes")
	}
	if blobs, receipts := countFiles(t, parent); blobs != 1 || receipts != 1 {
		t.Fatalf("after retry: blobs = %d, receipts = %d, want 1, 1", blobs, receipts)
	}
}

// captureCrashChild runs the production Capture entry and kills
// its own process at the AfterBlob boundary. It never returns.
func captureCrashChild() {
	repo, err := sessrepo.Open(os.Getenv(crashChildRepo))
	if err != nil {
		os.Exit(64)
	}
	events, err := repo.ListEvents(testSessionID)
	if err != nil || len(events) != 1 {
		os.Exit(65)
	}
	store, err := Open(os.Getenv(crashChildStore))
	if err != nil {
		os.Exit(66)
	}
	store.AfterBlob = func() error {
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGKILL)
		// The kill is asynchronous under instrumentation: block
		// here so a delivered SIGKILL always wins the race
		// against process exit. If the kill failed, the parent
		// context times out and fails the test instead of
		// passing on a wrong exit.
		select {}
	}
	_, _, _ = store.Capture(repo, testInputs([]string{events[0].EventID}, testOpA))
	// The hook must have fired: reaching here means the boundary
	// was skipped, so fail the parent with a clean exit.
	os.Exit(0)
}
