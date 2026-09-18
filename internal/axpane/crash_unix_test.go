//go:build darwin || linux

package axpane

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// Real process-termination evidence for the bootstrap binding seam:
// the child runs the genuine production Bind entry and SIGKILLs
// itself in the AfterInstall hook, after the no-replace commit and
// before the directory sync. The parent then proves the interrupted
// bind left the complete binding behind (never a torn prefix and
// never a second child), and the identical retry reattaches to it.
const (
	crashChildEnv   = "AX_AXPANE_CRASH_CHILD"
	crashChildStore = "AX_AXPANE_CRASH_STORE"
)

func TestBindCrashChildSelfTerminates(t *testing.T) {
	if os.Getenv(crashChildEnv) == "1" {
		bindCrashChild(t)
		return
	}
	storeDir := t.TempDir()
	if _, err := Open(storeDir); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, os.Args[0],
		"-test.run=^TestBindCrashChildSelfTerminates$",
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
	store, err := Open(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	proven, found, err := store.Status(fixtureSession)
	if err != nil {
		t.Fatalf("Status() after kill error = %v", err)
	}
	if !found {
		t.Fatal("Status() after kill reports absence, want the committed binding")
	}
	if proven.SessionID != fixtureSession || proven.OperationID != fixtureBootstrap || proven.TerminalInstanceID != fixtureInstance {
		t.Fatalf("Status() after kill = %+v, want the complete receipt", proven)
	}
	second, reattached, err := store.Bind(fixtureSession, fixtureBootstrap, bindCandidate())
	if err != nil {
		t.Fatalf("Bind() retry after kill error = %v", err)
	}
	if !reattached {
		t.Fatal("Bind() retry after kill installs fresh, want reattach to the one child")
	}
	if second != proven {
		t.Fatalf("Bind() retry = %+v, want %+v", second, proven)
	}
}

func bindCrashChild(t *testing.T) {
	t.Helper()
	store, err := Open(os.Getenv(crashChildStore))
	if err != nil {
		t.Fatalf("child Open() error = %v", err)
	}
	candidate := bindCandidate()
	store.WithHooks(&Hooks{
		AfterInstall: func(path string) error {
			proc, err := os.FindProcess(os.Getpid())
			if err != nil {
				t.Fatalf("child find self: %v", err)
			}
			_ = proc.Signal(syscall.SIGKILL)
			select {}
		},
	})
	_, _, _ = store.Bind(fixtureSession, fixtureBootstrap, candidate)
	t.Fatal("child Bind() returned past SIGKILL")
}
