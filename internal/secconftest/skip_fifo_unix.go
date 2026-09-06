//go:build !windows

package secconftest

import (
	"fmt"
	"path/filepath"
	"syscall"
)

// makeFifo creates a named pipe at dir/name with mode 0600. Unix-only:
// Windows has no Mkfifo and its file carries the unavailable stub, so
// the fifo probe fails (and skips, since Windows is not strict)
// instead of pretending.
func makeFifo(dir, name string) (string, error) {
	return fifoBuild(dir, name)
}

// fifoBuild is the seam the probe-honesty test faults: it swaps in a
// failing build, drives FifoCapability().Probe(), and requires
// failure. A probe closure hardened to constant true ignores the
// fault and fails the test.
var fifoBuild = func(dir, name string) (string, error) {
	path := filepath.Join(dir, name)
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		return "", fmt.Errorf("secconftest: mkfifo %q: %w", path, err)
	}
	return path, nil
}
