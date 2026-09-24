//go:build !windows

package tmuxserver

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

func TestEnsureRuntimeDirCreatesOwnerOnly(t *testing.T) {
	root := runtimeRoot(t)
	dir, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
	if err != nil {
		t.Fatal(err)
	}
	if dir != filepath.Join(root, RuntimeDirName) {
		t.Fatalf("dir = %q", dir)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() || info.Mode().Perm() != 0o700 {
		t.Fatalf("mode = %04o dir=%v, want 0700 dir", info.Mode().Perm(), info.IsDir())
	}
}

func TestEnsureRuntimeDirIsIdempotent(t *testing.T) {
	root := runtimeRoot(t)
	first, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("second = %q, want %q", second, first)
	}
	info, err := os.Lstat(second)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("mode after re-ensure = %04o, want 0700", info.Mode().Perm())
	}
}

func TestVerifyRuntimeDirAdmitsCompliant(t *testing.T) {
	root := runtimeRoot(t)
	runtimeDir(t, root)
	if err := VerifyRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureRuntimeDirRefusesWidenedMode(t *testing.T) {
	root := runtimeRoot(t)
	dir, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
	if err != nil {
		t.Fatal(err)
	}
	// The suite widens the mode behind the entry's back; re-entry
	// must refuse, never silently repair.
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err = EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
	requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime mode")
	if err := VerifyRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS); err != nil {
		requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime mode")
	} else {
		t.Fatal("VerifyRuntimeDir admitted a 0755 directory")
	}
}

// "Exactly 0700" is pinned on the narrower side too: a pre-existing
// owner-narrower leaf (0600/0500) refuses the mode gate on both
// entries, never converging by silent repair. The mode is set by an
// explicit chmod so the fixture is exact under any test umask.
func TestEnsureRuntimeDirRefusesNarrowerMode(t *testing.T) {
	for _, tc := range []struct {
		name string
		perm os.FileMode
	}{
		{"0600", 0o600},
		{"0500", 0o500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := runtimeRoot(t)
			leaf := filepath.Join(root, RuntimeDirName)
			if err := os.Mkdir(leaf, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(leaf, tc.perm); err != nil {
				t.Fatal(err)
			}
			_, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
			requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime mode")
			if verr := VerifyRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS); verr != nil {
				requireLocalError(t, verr, "tmux_unsafe_runtime_dir", "runtime mode")
			} else {
				t.Fatal("VerifyRuntimeDir admitted a narrower-than-0700 directory")
			}
		})
	}
}

// The commit side refuses a non-directory leaf through the O_DIRECTORY
// flag on its Openat — the same refusal the verify side pins, on the
// entry the foreground path uses. The fixtures mirror the verify
// side's: a 0600 regular file pins that the kind refusal precedes the
// mode check, a 0700 regular file pins the admit direction (a weakened
// open admits it outright instead of rerouting to the mode gate), and
// a FIFO under a deadline pins the no-hang property — FIFO refusal
// relies on O_DIRECTORY failing fast with ENOTDIR, symmetric with the
// verify side (TRACEABILITY row 5).
func TestEnsureRuntimeDirRefusesNonDirectoryLeaf(t *testing.T) {
	t.Run("regular file 0600", func(t *testing.T) {
		root := runtimeRoot(t)
		leaf := filepath.Join(root, RuntimeDirName)
		if err := os.WriteFile(leaf, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
		requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime containment")
	})
	t.Run("regular file 0700", func(t *testing.T) {
		root := runtimeRoot(t)
		leaf := filepath.Join(root, RuntimeDirName)
		if err := os.WriteFile(leaf, []byte("x"), 0o700); err != nil {
			t.Fatal(err)
		}
		_, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
		requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime containment")
	})
	t.Run("fifo without blocking", func(t *testing.T) {
		root := runtimeRoot(t)
		if err := syscall.Mkfifo(filepath.Join(root, RuntimeDirName), 0o600); err != nil {
			t.Fatal(err)
		}
		err := verifyWithDeadline(t, "EnsureRuntimeDir", func() error {
			_, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
			return err
		})
		requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime containment")
	})
}

// The verify side refuses a non-directory leaf through the O_DIRECTORY
// flag on its Openat — the same refusal the commit side pins, on the
// entry the background path uses. The regular file carries 0700 so a
// weakened open admits it outright instead of rerouting to the mode
// gate; the FIFO runs under a deadline so the no-hang property is
// pinned too. The verify open carries no O_NONBLOCK by decision: FIFO
// refusal relies on O_DIRECTORY failing fast with ENOTDIR, symmetric
// with the commit side (TRACEABILITY row 5).
func TestVerifyRuntimeDirRefusesNonDirectoryLeaf(t *testing.T) {
	t.Run("regular file", func(t *testing.T) {
		root := runtimeRoot(t)
		leaf := filepath.Join(root, RuntimeDirName)
		if err := os.WriteFile(leaf, []byte("x"), 0o700); err != nil {
			t.Fatal(err)
		}
		err := VerifyRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS)
		requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime containment")
	})
	t.Run("fifo without blocking", func(t *testing.T) {
		root := runtimeRoot(t)
		if err := syscall.Mkfifo(filepath.Join(root, RuntimeDirName), 0o600); err != nil {
			t.Fatal(err)
		}
		err := verifyWithDeadline(t, "VerifyRuntimeDir", func() error {
			return VerifyRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS)
		})
		requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime containment")
	})
}

// verifyWithDeadline turns a blocked custody refusal into a failure
// instead of a stalled suite: with O_DIRECTORY the refusal arrives
// immediately, and without it the FIFO open stalls past the deadline.
// The entry name identifies the blocked call in the failure message;
// both the commit and the verify side run their FIFO fixture through
// this helper.
func verifyWithDeadline(t *testing.T, entry string, fn func() error) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- fn() }()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatalf("%s BLOCKED for 5s on a FIFO leaf (O_DIRECTORY must fail fast)", entry)
		return nil
	}
}

func TestEnsureRuntimeDirRefusesSymlinkLeaf(t *testing.T) {
	root := runtimeRoot(t)
	escape := t.TempDir()
	leaf := filepath.Join(root, RuntimeDirName)
	if err := os.Symlink(escape, leaf); err != nil {
		t.Fatal(err)
	}
	_, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
	requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime containment")
}

func TestEnsureRuntimeDirRefusesForeignOwnership(t *testing.T) {
	root := runtimeRoot(t)
	runtimeDir(t, root)
	previous := effectiveUID
	effectiveUID = func() int { return previous() + 100000 }
	defer func() { effectiveUID = previous }()
	_, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
	requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime ownership")
	if err := VerifyRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS); err != nil {
		requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime ownership")
	} else {
		t.Fatal("VerifyRuntimeDir admitted foreign ownership")
	}
}

func TestEnsureRuntimeDirRefusesHandleMismatch(t *testing.T) {
	root := runtimeRoot(t)
	previous := sameDirectory
	sameDirectory = func(_, _ os.FileInfo) bool { return false }
	defer func() { sameDirectory = previous }()
	_, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
	requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime containment")
	if err := VerifyRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS); err != nil {
		requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime containment")
	} else {
		t.Fatal("VerifyRuntimeDir admitted a handle mismatch")
	}
}

func TestEnsureRuntimeDirRefusesSymlinkRoot(t *testing.T) {
	real := runtimeRoot(t)
	link := filepath.Join(t.TempDir(), "linkroot")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	_, err := EnsureRuntimeDir(link, RuntimeDirName, scalar.PlatformMacOS, nil)
	requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime containment")
}

func TestEnsureRuntimeDirRefusesBadRoots(t *testing.T) {
	root := runtimeRoot(t)
	for _, tc := range []struct {
		name string
		root string
	}{
		{"relative", "relative/root"},
		{"empty", ""},
		{"parent", "/tmp/../etc"},
		{"missing", filepath.Join(root, "no-such-root")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := EnsureRuntimeDir(tc.root, RuntimeDirName, scalar.PlatformMacOS, nil)
			requireLocalError(t, err, "tmux_invalid_arguments", "runtime root")
		})
	}
}

func TestEnsureRuntimeDirRefusesBadNames(t *testing.T) {
	root := runtimeRoot(t)
	for _, tc := range []struct {
		name  string
		value string
	}{
		{"empty", ""},
		{"dot", "."},
		{"parent", ".."},
		{"nested", "a/b"},
		{"absolute", "/abs"},
		{"nul", "a\x00b"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := EnsureRuntimeDir(root, tc.value, scalar.PlatformMacOS, nil)
			requireLocalError(t, err, "tmux_invalid_arguments", "runtime name")
		})
	}
}

func TestEnsureRuntimeDirRefusesBadPlatform(t *testing.T) {
	root := runtimeRoot(t)
	_, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.Platform("plan9"), nil)
	requireLocalError(t, err, "tmux_invalid_arguments", "runtime platform")
}

func TestEnsureRuntimeDirRefusesWindowsOnUnixHost(t *testing.T) {
	// Native Windows MUST NOT claim tmux: the Windows platform
	// refuses at the lexical gate on every host, before any
	// host-specific commit runs — on the commit and the verify
	// entries alike, since both share the lexical gate.
	for _, root := range []string{`C:\ax-runtime`, `D:\other-runtime`} {
		_, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformWindows, nil)
		requireLocalError(t, err, "tmux_invalid_arguments", "runtime platform")
		if verr := VerifyRuntimeDir(root, RuntimeDirName, scalar.PlatformWindows); verr != nil {
			requireLocalError(t, verr, "tmux_invalid_arguments", "runtime platform")
		} else {
			t.Fatal("VerifyRuntimeDir admitted the Windows platform")
		}
	}
}

// A symlinked runtime root refuses on the verify side exactly as on
// the commit side: the pinned root handle never follows the last
// component, and the shared classifier maps the no-follow failure to
// the containment refusal.
func TestVerifyRuntimeDirRefusesSymlinkRoot(t *testing.T) {
	real := runtimeRoot(t)
	link := filepath.Join(t.TempDir(), "linkroot")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	err := VerifyRuntimeDir(link, RuntimeDirName, scalar.PlatformMacOS)
	requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime containment")
}

func TestVerifyRuntimeDirRefusesSymlinkLeaf(t *testing.T) {
	root := runtimeRoot(t)
	escape := t.TempDir()
	leaf := filepath.Join(root, RuntimeDirName)
	if err := os.Symlink(escape, leaf); err != nil {
		t.Fatal(err)
	}
	err := VerifyRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS)
	requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime containment")
}

// The explicit post-mkdirat mode enforcement is load-bearing: under a
// restrictive umask the mkdir mode alone would leave a narrower leaf,
// and only the explicit chmod converges to the exact owner-only mode.
func TestEnsureRuntimeDirEnforcesModeUnderRestrictiveUmask(t *testing.T) {
	root := runtimeRoot(t)
	previous := syscall.Umask(0o177)
	defer syscall.Umask(previous)
	dir, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() || info.Mode().Perm() != 0o700 {
		t.Fatalf("mode under umask 0177 = %04o dir=%v, want 0700 dir", info.Mode().Perm(), info.IsDir())
	}
}

func TestVerifyRuntimeDirRefusesAbsence(t *testing.T) {
	root := runtimeRoot(t)
	err := VerifyRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS)
	requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime absent")
}

func TestVerifyRuntimeDirRefusesBadLexical(t *testing.T) {
	root := runtimeRoot(t)
	if err := VerifyRuntimeDir("relative/root", RuntimeDirName, scalar.PlatformMacOS); err != nil {
		requireLocalError(t, err, "tmux_invalid_arguments", "runtime root")
	} else {
		t.Fatal("VerifyRuntimeDir admitted a relative root")
	}
	if err := VerifyRuntimeDir(root, "a/b", scalar.PlatformMacOS); err != nil {
		requireLocalError(t, err, "tmux_invalid_arguments", "runtime name")
	} else {
		t.Fatal("VerifyRuntimeDir admitted a nested name")
	}
	if err := VerifyRuntimeDir(root, RuntimeDirName, scalar.Platform("plan9")); err != nil {
		requireLocalError(t, err, "tmux_invalid_arguments", "runtime platform")
	} else {
		t.Fatal("VerifyRuntimeDir admitted an unknown platform")
	}
}

func TestVerifyRuntimeDirRefusesMissingRoot(t *testing.T) {
	root := filepath.Join(runtimeRoot(t), "no-such-root")
	err := VerifyRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS)
	requireLocalError(t, err, "tmux_invalid_arguments", "runtime root")
}

// A commit-phase filesystem failure refuses with the commit detail
// instead of being mistaken for an existing leaf: the suite revokes
// root write permission, so Mkdirat fails EACCES, and restores it so
// the temporary directory cleans up. Non-root only (bound B14).
func TestEnsureRuntimeDirRefusesReadOnlyRoot(t *testing.T) {
	root := runtimeRoot(t)
	if err := os.Chmod(root, 0o500); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chmod(root, 0o700); err != nil {
			t.Fatal(err)
		}
	}()
	_, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
	requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime commit")
}

func TestLinuxAndWSL2ShareUnixCustody(t *testing.T) {
	for _, platform := range []scalar.Platform{scalar.PlatformLinux, scalar.PlatformWSL2} {
		root := runtimeRoot(t)
		dir, err := EnsureRuntimeDir(root, RuntimeDirName, platform, nil)
		if err != nil {
			t.Fatalf("%s: %v", platform, err)
		}
		info, err := os.Lstat(dir)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o700 {
			t.Fatalf("%s: mode = %04o", platform, info.Mode().Perm())
		}
	}
}
