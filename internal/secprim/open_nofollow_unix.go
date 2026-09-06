//go:build !windows

package secprim

import (
	"os"

	"golang.org/x/sys/unix"
)

// openNoFollow opens path read-only without following a trailing symlink:
// O_NOFOLLOW makes the open itself fail on a symlink target instead of
// traversing it. O_CLOEXEC keeps the descriptor out of any spawned child.
// O_NONBLOCK keeps a FIFO from stalling the opener: a read-only open of
// a FIFO with no writer blocks in the kernel, and the post-open shape
// check that refuses it runs only after the open returns, so without
// this flag the refusal is unreachable and the caller hangs instead.
// The caller clears the flag on admitted regular files.
func openNoFollow(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
}

// openNoFollowDir opens a directory read-only without following a trailing
// symlink, returning a handle that pins the directory for validation walks
// (Fstat, ReadDirnames) instead of re-resolving its name.
func openNoFollowDir(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_DIRECTORY, 0)
}
