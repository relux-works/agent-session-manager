package secprim

import (
	"errors"
	"os"
	"strings"
)

// errIsSymlink reports the Windows Lstat gate refusal. It stays internal:
// callers see the failPath refusal, and errors.Is against ErrUnsafePath
// is the supported match.
var errIsSymlink = errors.New("secprim symlink refused before open")

// errCommitSymlink is the internal signal from the platform commit walk
// (openCommitRelative) that a symlink or reparse point stands on the
// member path at commit time. It stays internal: Open maps it to the
// failPath "member symlink escape" refusal, and errors.Is against
// ErrUnsafePath is the supported match. The walk returns no failPath
// sites of its own so the refusal inventory stays in shared code,
// which the suite executes on every platform it runs on.
var errCommitSymlink = errors.New("secprim symlink on the commit path")

// guardRootMatches reports whether the open directory handle refers to
// the same directory as rootPath: the handle's fstat identity against a
// stat of the path (device and inode on unix via os.SameFile, volume
// and file index on Windows). A stat failure on either side is not a
// match: an unreadable root fails closed into the mismatch refusal
// rather than admitting an unverifiable handle.
func guardRootMatches(root *os.File, rootPath string) bool {
	handleInfo, err := root.Stat()
	if err != nil || handleInfo == nil {
		return false
	}
	pathInfo, err := os.Stat(rootPath)
	if err != nil || pathInfo == nil {
		return false
	}
	return os.SameFile(handleInfo, pathInfo)
}

// OpenNoFollowFile opens an existing regular file read-only, refusing a
// trailing symlink instead of traversing it (Section 16.3
// "symlink/reparse escape during both validation and commit"). The
// descriptor refers to the opened inode, and the post-open Fstat below
// reads that descriptor rather than the path, so a swap between open and
// stat cannot retarget the check: only the final-component open itself
// races, and on unix O_NOFOLLOW closes even that.
//
// The file must be regular: directories, devices, pipes, and sockets are
// refused after opening (and the descriptor closed) rather than handed
// back. Parent directories are the caller's responsibility: open the
// staging root with OpenNoFollowDir and commit members through
// Guard.Open, which binds the walk to the root handle. Composing
// Guard.Resolve with this function instead is not contained: Resolve
// answers the lexical question only, and a symlinked intermediate
// directory redirects a path-string open outside the root.
func OpenNoFollowFile(path string) (*os.File, error) {
	file, err := openNoFollow(path)
	if err != nil {
		return nil, failPath("open failed", memberErrorTarget(path)).WithCause(err)
	}
	info, err := statOpenedFile(file)
	if err != nil {
		_ = file.Close()
		return nil, failPath("open stat failed", memberErrorTarget(path)).WithCause(err)
	}
	if err := CheckRegularTarget(info); err != nil {
		_ = file.Close()
		return nil, err
	}
	// The unix open sets O_NONBLOCK so a FIFO can never stall the opener
	// (its refusal above runs only after the open returns). A regular
	// file ignores O_NONBLOCK on reads, and clearing it here keeps a
	// borrowed descriptor from leaking the flag to its reader. fcntl on
	// a just-opened valid descriptor cannot fail; the best-effort clear
	// carries no refusal site for that reason.
	_ = clearNonblock(file)
	return file, nil
}

// Open commits one guard member to an open regular file through the
// staging-root directory handle, refusing symlink and reparse escape at
// commit time (Section 16.3 "symlink/reparse escape during both
// validation and commit").
//
// Validation and commit are two halves of one entry point, not two calls
// the caller composes: the member grammar and the managed set refuse
// lexically first, then the platform walk (openCommitRelative) opens
// every path component relative to a directory descriptor — never by
// re-resolving a path string — so a symlinked intermediate directory
// cannot redirect the commit outside the root the way composing
// Guard.Resolve with OpenNoFollowFile does (O_NOFOLLOW constrains only
// the final component; every component before it is still followed by
// the kernel when a path string is opened).
//
// The caller opens the staging root with OpenNoFollowDir and passes the
// handle: on unix the walk never leaves that handle (openat from the
// root descriptor down, O_NOFOLLOW per component, TOCTOU-closed); on
// Windows the walk is a best-effort Lstat component check with the
// stated check-then-open bound. The returned file is cleared of
// O_NONBLOCK like OpenNoFollowFile. The caller owns closing both the
// root handle and the returned file.
//
// The handle is bound to the guard root before descending:
// guardRootMatches fstats the handle and compares device and inode
// against a stat of guard.root, refusing a foreign directory handle
// with "guard root mismatch". The check lives here in shared code so
// both platforms answer the mismatch identically; past it, the unix
// walk descends from the verified handle and the Windows walk from the
// verified root path, which name the same directory.
func (guard Guard) Open(root *os.File, member string) (*os.File, error) {
	if err := CheckMemberPath(guard.platform, member); err != nil {
		return nil, err
	}
	if guard.checkManaged && !guard.managed[member] {
		return nil, failPath("member unmanaged", memberErrorTarget(member))
	}
	if root == nil {
		return nil, failPath("guard root handle", memberErrorTarget(member))
	}
	if info, err := root.Stat(); err != nil || info == nil || !info.IsDir() {
		return nil, failPath("guard root handle", memberErrorTarget(member))
	}
	if !guardRootMatches(root, guard.root) {
		return nil, failPath("guard root mismatch", memberErrorTarget(member))
	}
	segments := strings.Split(member, "/")
	file, err := openCommitRelative(root, guard.root, member, segments)
	if err != nil {
		if errors.Is(err, errCommitSymlink) {
			return nil, failPath("member symlink escape", memberErrorTarget(member))
		}
		return nil, failPath("open failed", memberErrorTarget(member)).WithCause(err)
	}
	info, err := statOpenedFile(file)
	if err != nil {
		_ = file.Close()
		return nil, failPath("open stat failed", memberErrorTarget(member)).WithCause(err)
	}
	if err := CheckRegularTarget(info); err != nil {
		_ = file.Close()
		return nil, err
	}
	_ = clearNonblock(file)
	return file, nil
}

// statOpenedFile stats an already-opened descriptor. It is a variable
// only so seam-failure tests can force the stat-failure path through the
// production openers below: fstat on a live descriptor does not fail on
// any supported platform, and without the seam the error branch is
// untestable. Production code never reassigns it.
var statOpenedFile = func(file *os.File) (os.FileInfo, error) {
	return file.Stat()
}

// OpenNoFollowDir opens an existing directory read-only without following
// a trailing symlink, returning a handle for validation walks. The handle
// pins the opened directory: Fstat and ReadDirnames on it observe the
// directory that was opened even if its name is later retargeted.
//
// The Lstat fast path refuses non-directories with the same rules on
// every platform; the post-open Stat below stays authoritative against a
// swap between the check and the open.
func OpenNoFollowDir(path string) (*os.File, error) {
	if info, err := os.Lstat(path); err != nil {
		return nil, failPath("open failed", memberErrorTarget(path)).WithCause(err)
	} else if info.Mode()&os.ModeSymlink != 0 {
		return nil, failPath("target symlink", info.Name())
	} else if !info.IsDir() {
		return nil, failPath("target directory", info.Name())
	}
	dir, err := openNoFollowDir(path)
	if err != nil {
		return nil, failPath("open failed", memberErrorTarget(path)).WithCause(err)
	}
	info, err := statOpenedFile(dir)
	if err != nil {
		_ = dir.Close()
		return nil, failPath("open stat failed", memberErrorTarget(path)).WithCause(err)
	}
	if info == nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		_ = dir.Close()
		name := path
		if info != nil {
			name = info.Name()
		}
		return nil, failPath("target directory", name)
	}
	return dir, nil
}
