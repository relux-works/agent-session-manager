//go:build unix

package terminalbackend_test

import (
	"path/filepath"
	"syscall"
	"testing"

	terminalbackend "github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// TestDigestFileRefusesFIFO pins the regular-file guard itself: a directory is
// refused by os.ReadFile (EISDIR) even without the guard, while a FIFO is the
// documented hazard (README: os.Open blocks indefinitely on a FIFO) and only
// the Mode().IsRegular check refuses it without blocking.
//
// This case lives in a _unix-suffixed file (matching
// internal/localstore/projection_unix_test.go) because syscall.Mkfifo does not
// exist on Windows and would break GOOS=windows go vet on the shared test file.
func TestDigestFileRefusesFIFO(t *testing.T) {
	t.Parallel()

	fifo := filepath.Join(t.TempDir(), "backend-fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatalf("Mkfifo() error = %v", err)
	}
	if _, err := terminalbackend.DigestFile(fifo); err == nil {
		t.Error("DigestFile(fifo) error = nil, want refusal for a non-regular file")
	}
}
