package resumesmoke

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/provhost"
)

// This file kills a real installer mid-write: the crash child is
// the test binary re-executed, signaled ready after its bytes land
// but before the fsync, and then killed. The retry must recover to
// byte-identical evidence.

// TestStoreRealKillBetweenWriteAndSync kills the installer after
// the write and before the fsync, then proves the identical retry
// installs byte-identical evidence that loads valid.
func TestStoreRealKillBetweenWriteAndSync(t *testing.T) {
	tuple := provhost.BuildTuple{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "macos", Architecture: "arm64"}
	report := runRowFixture(t, tuple)
	directory := t.TempDir()
	input := filepath.Join(directory, "input.json")
	if err := os.WriteFile(input, report.Bytes, 0o600); err != nil {
		t.Fatalf("write helper input: %v", err)
	}
	ready := filepath.Join(directory, "ready")
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	command := exec.Command(executable)
	command.Env = append(os.Environ(),
		"RESUMESMOKE_HELPER=crash-store",
		"RESUMESMOKE_INPUT="+input,
		"RESUMESMOKE_DIR="+directory,
		"RESUMESMOKE_READY="+ready,
	)
	if err := command.Start(); err != nil {
		t.Fatalf("start crash child: %v", err)
	}
	waitReady(t, ready, 30*time.Second)
	if err := command.Process.Kill(); err != nil {
		t.Fatalf("kill crash child: %v", err)
	}
	_ = command.Wait()
	// The kill landed after the write: the record file exists with
	// the complete but unsynced bytes.
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	installed := ""
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "native-resume-smoke-") {
			installed = filepath.Join(directory, entry.Name())
		}
	}
	if installed == "" {
		t.Fatal("killed installer left no record file; the kill missed its window")
	}
	// The identical retry reuses the killed installer's bytes and
	// loads valid: the kill cost durability timing, never evidence.
	path, err := Store(directory, report.Bytes, nil)
	if err != nil {
		t.Fatalf("Store(retry after kill) error = %v", err)
	}
	if path != installed {
		t.Fatalf("retry path = %q, want the killed install %q", path, installed)
	}
	_, loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load error = %v", err)
	}
	if !bytes.Equal(loaded, report.Bytes) {
		t.Fatal("post-kill bytes differ from the record")
	}
}

// waitReady polls for the helper's readiness signal until the
// deadline: the kill must land after the write, never before it.
func waitReady(t *testing.T, ready string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(ready); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("crash child never signaled ready within %v", timeout)
}
