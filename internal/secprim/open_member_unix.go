//go:build !windows

package secprim

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// openCommitRelative walks one validated member from the staging-root
// directory handle to an open file descriptor without ever re-resolving
// a path string: every component is opened relative to the previous
// component's descriptor with openat. Intermediate components carry
// O_NOFOLLOW|O_DIRECTORY, so a symlinked directory on the path fails
// with ELOOP instead of redirecting the walk outside the root; the
// final component carries O_NOFOLLOW|O_NONBLOCK, so a trailing symlink
// fails with ELOOP and a FIFO returns immediately instead of stalling
// the opener until a writer appears. Descriptors pin what they opened:
// a rename or swap after any step cannot retarget a descriptor the
// walk already holds, which is what closes the validate-then-open
// TOCTOU rather than repeating it with more steps.
//
// The walk reports no refusal of its own: ELOOP maps to the shared
// errCommitSymlink signal (refused as "member symlink escape" by the
// caller) and every other failure returns raw (refused as "open
// failed" with its cause). Shape checks on the final descriptor stay
// with the caller, which stats the descriptor rather than the path.
//
// rootPath is the guard root the caller already bound to root with
// guardRootMatches before descending: the walk descends from the
// handle alone because descriptors pin what they opened, while the
// Windows walk descends from the verified path string. Both name the
// same directory past the shared binding check, so the containment
// root is one value on both platforms, not one per walk.
//
// Error classification runs before any descriptor is closed: from the
// second intermediate onward previous == current, so closing first
// would fstat a recycled descriptor number and misreport the shape.
func openCommitRelative(root *os.File, rootPath string, member string, segments []string) (*os.File, error) {
	// rootPath names the guard root already bound to root by the
	// caller; it is documentation here, enforcement in Guard.Open.
	_ = rootPath
	previous := -1
	current := int(root.Fd())
	for _, segment := range segments[:len(segments)-1] {
		next, err := unix.Openat(current, segment,
			unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
		if err != nil {
			classified := classifyCommitErr(current, segment, err)
			if previous >= 0 {
				_ = unix.Close(previous)
			}
			return nil, classified
		}
		if previous >= 0 {
			_ = unix.Close(previous)
		}
		previous, current = next, next
	}
	last := segments[len(segments)-1]
	final, err := unix.Openat(current, last,
		unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if err != nil {
		classified := classifyCommitErr(current, last, err)
		if previous >= 0 {
			_ = unix.Close(previous)
		}
		return nil, classified
	}
	if previous >= 0 {
		_ = unix.Close(previous)
	}
	return os.NewFile(uintptr(final), member), nil
}

// classifyCommitErr sorts the walk failures: a symlink anywhere on the
// path becomes the escape signal, while a missing component, a file
// where a directory stands, or a permission failure stays raw for the
// caller's open-failed refusal.
//
// The errno alone does not name the shape on every kernel: Linux
// reports ELOOP for an O_NOFOLLOW openat meeting a symlink, while
// Darwin reports ENOTDIR for the O_DIRECTORY half of the same event.
// Either errno therefore re-examines the component with fstatat at
// AT_SYMLINK_NOFOLLOW — still relative to the parent descriptor, so
// no path string is re-resolved — and only a component that is a
// symlink right now becomes the escape signal. A swap between the
// failed open and this check fails closed in every direction: each
// outcome (symlink, directory, missing) refuses, and only the rule
// name differs.
func classifyCommitErr(parent int, segment string, err error) error {
	if errors.Is(err, unix.ELOOP) {
		return errCommitSymlink
	}
	if errors.Is(err, unix.ENOTDIR) {
		var status unix.Stat_t
		if fstatErr := unix.Fstatat(parent, segment, &status, unix.AT_SYMLINK_NOFOLLOW); fstatErr == nil {
			if status.Mode&unix.S_IFMT == unix.S_IFLNK {
				return errCommitSymlink
			}
		}
	}
	return err
}

// clearNonblock drops O_NONBLOCK from an admitted regular file so a
// borrowed descriptor never leaks the flag to its reader. Regular-file
// reads ignore the flag either way.
func clearNonblock(file *os.File) error {
	return unix.SetNonblock(int(file.Fd()), false)
}
