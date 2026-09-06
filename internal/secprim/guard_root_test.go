package secprim

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// TestGuardOpenRefusesForeignRootHandle is the C1 vector: the guard root
// is one directory and the passed handle is another. The member exists
// under the foreign handle, so without the handle-to-root binding the
// commit walk would admit it and hand back foreign bytes. The refusal
// carries the mismatch rule, not a generic handle shape, and the check
// lives in shared Guard.Open code, so unix and Windows answer
// identically: this file carries no build tag and runs on both.
func TestGuardOpenRefusesForeignRootHandle(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	stage := filepath.Join(base, "stage")
	elsewhere := filepath.Join(base, "elsewhere")
	if err := os.MkdirAll(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(elsewhere, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(elsewhere, "secret"), []byte("FOREIGN"), 0o600); err != nil {
		t.Fatal(err)
	}
	guard, err := NewGuard(stage, scalar.PlatformLinux, nil)
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := OpenNoFollowDir(elsewhere)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = foreign.Close() }()
	file, err := guard.Open(foreign, "secret")
	if err == nil {
		contents, _ := io.ReadAll(file)
		_ = file.Close()
		t.Fatalf("foreign handle admitted; read %q outside the guard root", contents)
	}
	requireRefusal(t, err, ErrUnsafePath, "secprim unsafe path: guard root mismatch: secret")
}

// TestGuardOpenAdmitsMatchedRootHandle is the positive control for the
// binding: the handle opened on the guard root itself still commits.
// It runs on every platform beside the refusal above, so a binding
// that is too tight reddens here rather than passing silently.
func TestGuardOpenAdmitsMatchedRootHandle(t *testing.T) {
	t.Parallel()
	stage := t.TempDir()
	if err := os.WriteFile(filepath.Join(stage, "secret"), []byte("inside"), 0o600); err != nil {
		t.Fatal(err)
	}
	guard, err := NewGuard(stage, scalar.PlatformLinux, nil)
	if err != nil {
		t.Fatal(err)
	}
	root, err := OpenNoFollowDir(stage)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	file, err := guard.Open(root, "secret")
	if err != nil {
		t.Fatalf("matched handle refused: %v", err)
	}
	defer func() { _ = file.Close() }()
	contents, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "inside" {
		t.Fatalf("commit read %q, want the staged bytes", contents)
	}
}
