//go:build windows

package hosttrust

import (
	"errors"
	"os"
	"sync"

	"golang.org/x/sys/windows"
)

// fileLock is the machine-local cross-process authorization lock shared by
// trust changes and Configuration 4.0.0 peer/credential changes. Readers hold
// it shared; transactions hold it exclusive.
//
// Every acquisition opens its own handle: Windows byte-range locks attach to
// the file object, so goroutines sharing one retained handle would not
// serialize. Per-acquisition handles make both same-process goroutines and
// separate processes block on a held exclusive lock. A caller must never
// request a second acquisition while holding one in the same goroutine;
// internal reads nested inside a holder use handle-direct unlocked reads
// instead of re-acquiring.
type fileLock struct {
	path string
}

// lockInitMu serializes concurrent creation of the lock file itself.
var lockInitMu sync.Mutex

// openLock attaches to the shared lock file, creating it exactly once
// across all processes. Creation is atomic and non-replacing (CREATE_NEW):
// a process that observes absence while another process creates the lock
// loses the race and opens the existing file instead of replacing a live
// lock, so concurrent first-open can never split the authorization lock
// across two files.
func openLock(filesystem FileSystem, path string) (*fileLock, error) {
	lockInitMu.Lock()
	defer lockInitMu.Unlock()
	_, err := filesystem.Lstat(path)
	if err == nil {
		if err := verifyLockFile(filesystem, path); err != nil {
			return nil, err
		}
		return &fileLock{path: path}, nil
	}
	if !os.IsNotExist(err) {
		return nil, trustError(TrustError{Operation: "open lock", Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	if err := filesystem.CreateExclusive(path, 0o600); err != nil {
		// Lost the creation race: another process published the lock
		// first. Open the existing file; never replace it.
		if !isLockExistsError(err) {
			return nil, trustError(TrustError{Operation: "open lock", Err: errors.Join(ErrTrustStoreUnreadable, err)})
		}
		if err := verifyLockFile(filesystem, path); err != nil {
			return nil, err
		}
		return &fileLock{path: path}, nil
	}
	if err := filesystem.Chmod(path, 0o600); err != nil {
		return nil, trustError(TrustError{Operation: "open lock", Err: errors.Join(ErrTrustUnsafeCustody, err)})
	}
	if err := secureStaged(path, false); err != nil {
		return nil, err
	}
	info, err := filesystem.Lstat(path)
	if err != nil {
		return nil, trustError(TrustError{Operation: "open lock", Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	if err := verifyOwnerFile(path, info, 0o600); err != nil {
		return nil, err
	}
	return &fileLock{path: path}, nil
}

// isLockExistsError reports a lost creation race: the lock file already
// exists, so the caller must open it rather than publish over it.
func isLockExistsError(err error) bool {
	return os.IsExist(err) || errors.Is(err, windows.ERROR_FILE_EXISTS) || errors.Is(err, windows.ERROR_ALREADY_EXISTS)
}

func openLockHandle(path string) (windows.Handle, error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, trustError(TrustError{Operation: "open lock", Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return 0, trustError(TrustError{Operation: "open lock", Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	return handle, nil
}

func (lock *fileLock) shared() (func(), error) {
	handle, err := openLockHandle(lock.path)
	if err != nil {
		return nil, err
	}
	overlapped := new(windows.Overlapped)
	if err := windows.LockFileEx(handle, 0, 0, 1, 0, overlapped); err != nil {
		_ = windows.CloseHandle(handle)
		return nil, err
	}
	return func() {
		_ = windows.UnlockFileEx(handle, 0, 1, 0, overlapped)
		_ = windows.CloseHandle(handle)
	}, nil
}

func (lock *fileLock) exclusive() (func(), error) {
	handle, err := openLockHandle(lock.path)
	if err != nil {
		return nil, err
	}
	overlapped := new(windows.Overlapped)
	if err := windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, overlapped); err != nil {
		_ = windows.CloseHandle(handle)
		return nil, err
	}
	return func() {
		_ = windows.UnlockFileEx(handle, 0, 1, 0, overlapped)
		_ = windows.CloseHandle(handle)
	}, nil
}
