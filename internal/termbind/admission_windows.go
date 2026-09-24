//go:build windows

package termbind

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

func tryLockAdmissionFile(file *os.File) (bool, func() error, error) {
	overlapped := new(windows.Overlapped)
	handle := windows.Handle(file.Fd())
	err := windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, overlapped)
	if err == nil {
		return true, func() error { return windows.UnlockFileEx(handle, 0, 1, 0, overlapped) }, nil
	}
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return false, nil, nil
	}
	return false, nil, err
}
