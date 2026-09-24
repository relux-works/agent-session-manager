//go:build !windows

package termbind

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func tryLockAdmissionFile(file *os.File) (bool, func() error, error) {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err == nil {
		return true, func() error { return unix.Flock(int(file.Fd()), unix.LOCK_UN) }, nil
	}
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return false, nil, nil
	}
	return false, nil, err
}
