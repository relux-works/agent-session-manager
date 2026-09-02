//go:build darwin || linux

package localstore

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

type projectionLock struct {
	file *os.File
}

func acquireProjectionLock(path string) (*projectionLock, error) {
	flags := unix.O_CREAT | unix.O_EXCL | unix.O_RDWR | unix.O_CLOEXEC | unix.O_NOFOLLOW
	fd, err := unix.Open(path, flags, 0o600)
	created := err == nil
	if errors.Is(err, unix.EEXIST) {
		fd, err = unix.Open(path, unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	}
	if err != nil {
		if errors.Is(err, unix.ELOOP) {
			return nil, projectionRefusal(ErrUnsafeOwnership, "projection lock is a symlink")
		}
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	fail := func(err error) (*projectionLock, error) {
		_ = file.Close()
		return nil, err
	}
	if created {
		if err := file.Chmod(0o600); err != nil {
			return fail(err)
		}
	}
	info, err := file.Stat()
	if err != nil {
		return fail(err)
	}
	if !info.Mode().IsRegular() {
		return fail(projectionRefusal(ErrUnsafeOwnership, "projection lock is not a regular file"))
	}
	if err := verifyOwnerFileInfo(info, 0o600); err != nil {
		return fail(projectionOwnershipRefusal(ErrUnsafeOwnership, err, "projection lock %q", path))
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX); err != nil {
		return fail(err)
	}
	return &projectionLock{file: file}, nil
}

func (lock *projectionLock) release() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	file := lock.file
	lock.file = nil
	unlockErr := unix.Flock(int(file.Fd()), unix.LOCK_UN)
	closeErr := file.Close()
	return errors.Join(unlockErr, closeErr)
}
