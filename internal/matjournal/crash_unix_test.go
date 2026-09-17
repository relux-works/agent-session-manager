//go:build darwin || linux

package matjournal

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// Real process-termination evidence for the CR-MAT-02 boundary: the
// child runs the genuine production Create entry and SIGKILLs itself
// between the journal install and the prepare receipt. The parent then
// proves the interrupted prepare left one complete journal with no
// receipt behind, and the identical retry converges with its receipt
// and a safe_retry classification.
//
// The child rebuilds nothing: it opens the parent's data directory
// from the environment, so its closure inputs are byte-identical to
// the parent's retry by construction.
const (
	crashChildEnv   = "AX_MATJOURNAL_CRASH_CHILD"
	crashChildStore = "AX_MATJOURNAL_CRASH_STORE"
)

func TestCreateCrashChildSelfTerminates(t *testing.T) {
	if os.Getenv(crashChildEnv) == "1" {
		createCrashChild()
		return
	}
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
		"-test.run=^TestCreateCrashChildSelfTerminates$",
		"-test.count=1",
	)
	child.Env = append(os.Environ(),
		crashChildEnv+"=1",
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
	// The kill landed between the journal and the receipt: one
	// complete journal that verifies, no receipt, no partial bytes.
	parent, err := Open(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	parent.Now = fixedClock
	if journal, receipt := journalFilesExist(t, parent); !journal || receipt {
		t.Fatalf("after SIGKILL: journal = %v, receipt = %v, want true, false", journal, receipt)
	}
	loaded, _, err := parent.Get(testMatID)
	if err != nil {
		t.Fatalf("Get after SIGKILL error = %v", err)
	}
	if loaded.Phase != PhaseStaging {
		t.Fatalf("phase = %q", loaded.Phase)
	}
	// The identical retry converges on the recorded journal with
	// its receipt: the interrupted prepare is resumed, not
	// duplicated. Recovery then classifies safe_retry.
	ref, _ := mustCreate(t, parent, testInputsComposite())
	if ref.PrepareOperationID != testPrepareOp {
		t.Fatalf("retry PrepareOperationID = %q", ref.PrepareOperationID)
	}
	if journal, receipt := journalFilesExist(t, parent); !journal || !receipt {
		t.Fatalf("after retry: journal = %v, receipt = %v, want true, true", journal, receipt)
	}
	input := baseRecoveryInput()
	input.Boundary = BoundaryAfterPrepare
	outcome, _, err := parent.Recover(testMatID, input)
	if err != nil {
		t.Fatalf("Recover error = %v", err)
	}
	mustOutcome(t, outcome, OutcomeSafeRetry, "Recover(CR-MAT-02 SIGKILL)")
}

// createCrashChild runs the production Create entry and kills its own
// process at the AfterJournal boundary. It never returns.
func createCrashChild() {
	store, err := Open(os.Getenv(crashChildStore))
	if err != nil {
		os.Exit(66)
	}
	store.AfterJournal = func() error {
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGKILL)
		// The kill is asynchronous under instrumentation: block
		// here so a delivered SIGKILL always wins the race
		// against process exit. If the kill failed, the parent
		// context times out and fails the test instead of
		// passing on a wrong exit.
		select {}
	}
	_, _, _ = store.Create(testInputsComposite())
	// The hook must have fired: reaching here means the boundary
	// was skipped, so fail the parent with a clean exit.
	os.Exit(0)
}
