//go:build darwin || linux

package gitsnap

import "syscall"

// makeFifo creates a FIFO for the special-file refusal test.
func makeFifo(path string) error {
	return syscall.Mkfifo(path, 0o644)
}
