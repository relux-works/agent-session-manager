//go:build darwin || linux

package tmuxserver

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Real process-termination evidence for the runtime-directory seam.
// The child runs the genuine production EnsureRuntimeDir entry and
// SIGKILLs itself in the AfterMkdir hook, after the mkdirat commit
// and before verification and the parent fsync. The parent proves
// the interrupted run left no half-custody behind: the retry
// converges to the one verified owner-only directory.
const (
	crashChildMode = "AX_TMUXSERVER_CRASH_MODE"
	crashChildRoot = "AX_TMUXSERVER_CRASH_ROOT"
)

func TestEnsureCrashChildSelfTerminates(t *testing.T) {
	if os.Getenv(crashChildMode) == "ensure" {
		ensureCrashChild(t)
		return
	}
	root := runtimeRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, os.Args[0],
		"-test.run=^TestEnsureCrashChildSelfTerminates$",
		"-test.count=1",
	)
	child.Env = append(os.Environ(),
		crashChildMode+"=ensure",
		crashChildRoot+"="+root,
	)
	// The child's combined output is captured into the failure
	// message so the next anomalous exit is attributable instead of
	// undiagnosable.
	childOut, runErr := child.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(runErr, &exitErr) {
		t.Fatalf("child run = %v (output: %s), want a signal-kill exit", runErr, childOut)
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
		t.Fatalf("child status = %v (output: %s), want killed by SIGKILL", exitErr, childOut)
	}
	dir, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
	if err != nil {
		t.Fatalf("EnsureRuntimeDir() retry after kill error = %v", err)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() || info.Mode().Perm() != 0o700 {
		t.Fatalf("dir after kill = %04o dir=%v, want 0700 dir", info.Mode().Perm(), info.IsDir())
	}
	if err := VerifyRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS); err != nil {
		t.Fatalf("VerifyRuntimeDir() after kill error = %v", err)
	}
	again, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
	if err != nil {
		t.Fatal(err)
	}
	if again != dir {
		t.Fatalf("second retry = %q, want %q", again, dir)
	}
}

func ensureCrashChild(t *testing.T) {
	t.Helper()
	root := os.Getenv(crashChildRoot)
	hooks := &Hooks{AfterMkdir: func(dir string) {
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGKILL)
	}}
	_, _ = EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, hooks)
	// If the hook failed to kill, fail loudly so the parent never
	// mistakes a clean exit for a kill.
	t.Fatal("crash child survived its kill hook")
}

func TestAcquireForegroundSpawnIsIdempotentAcrossRetry(t *testing.T) {
	root := runtimeRoot(t)
	spawned := false
	spawns := 0
	make := func() Request {
		req := validRequest(t, root)
		req.Deps.ProbeServer = func(socket string) (ServerReport, error) {
			if spawned {
				return ServerReport{Running: true, Admission: boundAdmission()}, nil
			}
			return ServerReport{}, nil
		}
		req.Deps.Spawn = func(socket string, argv []string) error {
			spawns++
			spawned = true
			return nil
		}
		return req
	}
	first, err := Acquire(make())
	if err != nil {
		t.Fatal(err)
	}
	if first.Via != "spawned" {
		t.Fatalf("first via = %q", first.Via)
	}
	second, err := Acquire(make())
	if err != nil {
		t.Fatal(err)
	}
	if second.Via != "attached-running" {
		t.Fatalf("second via = %q, want attached-running", second.Via)
	}
	if second.Socket != first.Socket {
		t.Fatalf("sockets differ: %q vs %q", second.Socket, first.Socket)
	}
	if spawns != 1 {
		t.Fatalf("spawns = %d, want exactly 1", spawns)
	}
}
