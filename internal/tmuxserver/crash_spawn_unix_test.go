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

// Real process-termination evidence for the spawn seam. The child runs
// the genuine production unlinkStaleSocket over a staged stale socket
// and SIGKILLs itself where the spawn commit would run — after the
// unlink, before the server spawn. The parent proves the interrupted
// run left a clean bind behind: the retry spawns exactly once through
// the production spawner with no tmux process anywhere (fake runner).
const (
	spawnCrashChildMode   = "AX_TMUXSERVER_SPAWN_CRASH_MODE"
	spawnCrashChildSocket = "AX_TMUXSERVER_SPAWN_CRASH_SOCKET"
)

func TestSpawnCrashBetweenUnlinkAndSpawnConverges(t *testing.T) {
	if os.Getenv(spawnCrashChildMode) == "unlink" {
		spawnCrashChild(t)
		return
	}
	root, socket := lxProbeRoot(t)
	stageStaleSocket(t, socket)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, os.Args[0],
		"-test.run=^TestSpawnCrashBetweenUnlinkAndSpawnConverges$",
		"-test.count=1",
	)
	child.Env = append(os.Environ(),
		spawnCrashChildMode+"=unlink",
		spawnCrashChildSocket+"="+socket,
	)
	childOut, runErr := child.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(runErr, &exitErr) {
		t.Fatalf("child run = %v (output: %s), want a signal-kill exit", runErr, childOut)
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
		t.Fatalf("child status = %v (output: %s), want killed by SIGKILL", exitErr, childOut)
	}
	if _, err := os.Lstat(socket); !os.IsNotExist(err) {
		t.Fatalf("stale socket after kill: %v", err)
	}
	runner := newFakeRunner().queue("new-session", "", 0)
	spawner := ServerSpawner{Root: root, Platform: scalar.PlatformMacOS, Runner: runner}
	good, err := BuildArgv(root+"/tmux", socket, lxSession)
	if err != nil {
		t.Fatal(err)
	}
	if err := spawner.Spawn(socket, good); err != nil {
		t.Fatalf("Spawn() retry after kill error = %v", err)
	}
	if runner.callCount() != 1 {
		t.Fatalf("spawn calls = %d, want 1", runner.callCount())
	}
}

func spawnCrashChild(t *testing.T) {
	t.Helper()
	socket := os.Getenv(spawnCrashChildSocket)
	if _, err := unlinkStaleSocket(socket); err != nil {
		t.Fatalf("child unlink: %v", err)
	}
	_ = syscall.Kill(syscall.Getpid(), syscall.SIGKILL)
	t.Fatal("crash child survived its kill")
}
