//go:build darwin || linux

package hosttrust

import (
	"errors"
	"os"
	"sync"

	"golang.org/x/sys/unix"
)

// fileLock is the machine-local cross-process authorization lock shared by
// trust changes and Configuration 4.0.0 peer/credential changes. Readers hold
// it shared; transactions hold it exclusive, so a revocation or config commit
// can never land between a dispatch check and its side-effect boundary.
//
// Every acquisition opens its own file description: flock locks attach to the
// open file description, so goroutines sharing one retained descriptor would
// not serialize. Per-acquisition descriptors make both same-process
// goroutines and separate processes block on a held exclusive lock. A caller
// must never request a second acquisition while holding one in the same
// goroutine; internal reads nested inside a holder use descriptor-direct
// unlocked reads instead of re-acquiring.
type fileLock struct {
	path string
}

// lockInitMu serializes concurrent creation of the lock file itself.
var lockInitMu sync.Mutex

// openLock attaches to the shared lock file, creating it exactly once
// across all processes. Creation is atomic and non-replacing
// (O_CREAT|O_EXCL): a process that observes absence while another process
// creates the lock loses the race and opens the existing inode instead of
// renaming over a live lock, so concurrent first-open can never split the
// authorization lock across two inodes.
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
		// first. Open the existing inode; never replace it.
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
	return os.IsExist(err) || errors.Is(err, unix.EEXIST)
}

// openLockFD opens one private file description for a single acquisition.
// O_NOFOLLOW refuses symlink escapes at open time; the descriptor never
// follows a link swapped in after Lstat.
func openLockFD(path string) (int, error) {
	fd, err := unix.Open(path, unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		joined := errors.Join(ErrTrustStoreUnreadable, err)
		if errors.Is(err, unix.ELOOP) {
			joined = errors.Join(ErrTrustUnsafeCustody, err)
		}
		return -1, trustError(TrustError{Operation: "open lock", Err: joined})
	}
	return fd, nil
}

func (lock *fileLock) shared() (func(), error) {
	fd, err := openLockFD(lock.path)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(fd, unix.LOCK_SH); err != nil {
		_ = unix.Close(fd)
		return nil, err
	}
	return func() {
		_ = unix.Flock(fd, unix.LOCK_UN)
		_ = unix.Close(fd)
	}, nil
}

func (lock *fileLock) exclusive() (func(), error) {
	fd, err := openLockFD(lock.path)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(fd, unix.LOCK_EX); err != nil {
		_ = unix.Close(fd)
		return nil, err
	}
	return func() {
		_ = unix.Flock(fd, unix.LOCK_UN)
		_ = unix.Close(fd)
	}, nil
}
