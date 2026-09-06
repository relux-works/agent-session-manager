//go:build !windows

package secprim

import (
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

// makeFifo creates a named pipe for the special-file gate. It lives in a
// unix-tagged file because syscall.Mkfifo does not compile on Windows.
func makeFifo(t *testing.T, path string) {
	t.Helper()
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}
}

// privilegedUser reports whether permission refusals are untestable
// because the process runs as root, which bypasses mode bits.
func privilegedUser() bool {
	return unix.Geteuid() == 0
}
