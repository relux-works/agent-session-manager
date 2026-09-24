//go:build !windows

package tmuxserver

import (
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"

	"github.com/relux-works/agent-session-manager/internal/secprim"
)

// commitRuntimeDir creates the leaf relative to the pinned root
// handle and verifies exact custody on the opened descriptor. The
// root itself must already exist: this entry creates exactly one
// leaf and never builds parents, so a missing root refuses instead
// of materializing an arbitrary ancestor chain with owner-only
// modes. Mkdirat from the handle closes the validate-then-create
// TOCTOU a path-string MkdirAll leaves open: descriptors pin what
// they opened, and the leaf name is one grammar-checked segment.
// classifyRootOpenError maps a pinned-root open failure to its
// refusal: a missing root is malformed input — the entry creates
// exactly one leaf and never builds parents — while every other
// failure (unreadable, non-directory, symlink) is a containment
// refusal. Commit and verify share this mapping so the missing-root
// code cannot drift between the two entries.
func classifyRootOpenError(err error) error {
	if os.IsNotExist(err) || errors.Is(err, os.ErrNotExist) {
		return &Error{Code: CodeInvalidArguments, Detail: "runtime root"}
	}
	return &Error{Code: CodeUnsafeRuntimeDir, Detail: "runtime containment"}
}

func commitRuntimeDir(root, name, joined string, hooks *Hooks) error {
	// No platform arm here: the lexical path already refused the
	// Windows platform on every host, so only unix platforms reach
	// this commit.
	rootHandle, err := secprim.OpenNoFollowDir(root)
	if err != nil {
		return classifyRootOpenError(err)
	}
	defer func() { _ = rootHandle.Close() }()
	if !handleBoundToPath(rootHandle, root) {
		return &Error{Code: CodeUnsafeRuntimeDir, Detail: "runtime containment"}
	}
	rootFd := int(rootHandle.Fd())
	created := true
	if err := unix.Mkdirat(rootFd, name, 0o700); err != nil {
		if !errors.Is(err, unix.EEXIST) {
			return &Error{Code: CodeUnsafeRuntimeDir, Detail: "runtime commit"}
		}
		created = false
	}
	if hooks != nil && hooks.AfterMkdir != nil {
		hooks.AfterMkdir(joined)
	}
	leafFd, err := unix.Openat(rootFd, name,
		unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return &Error{Code: CodeUnsafeRuntimeDir, Detail: "runtime containment"}
	}
	leaf := os.NewFile(uintptr(leafFd), joined)
	if created {
		// The mkdir mode honors umask, so the exact owner-only
		// mode is enforced explicitly on a leaf this call
		// created. An existing leaf is verified, never
		// repaired: silently narrowing a widened directory
		// would mask the custody violation.
		if err := unix.Fchmod(leafFd, 0o700); err != nil {
			_ = leaf.Close()
			return &Error{Code: CodeUnsafeRuntimeDir, Detail: "runtime mode"}
		}
	}
	info, err := leaf.Stat()
	if err != nil {
		_ = leaf.Close()
		return &Error{Code: CodeUnsafeRuntimeDir, Detail: "runtime mode"}
	}
	mode := custodyModeForPath(joined, info.Mode())
	// The openat O_DIRECTORY flag already refused a non-directory
	// leaf above; this exact owner-only predicate is shared by the
	// create and verify paths so both reject every group/other bit.
	if !mode.IsDir() || !ownerOnlyRuntimeDirMode(mode) {
		_ = leaf.Close()
		return &Error{Code: CodeUnsafeRuntimeDir, Detail: "runtime mode"}
	}
	if err := verifyOwnershipAt(joined, info); err != nil {
		_ = leaf.Close()
		return err
	}
	// The parent fsync persists the directory entry; without it a
	// crash after mkdir can lose the leaf on journaling filesystems.
	if err := unix.Fsync(rootFd); err != nil {
		_ = leaf.Close()
		return &Error{Code: CodeUnsafeRuntimeDir, Detail: "runtime commit"}
	}
	_ = leaf.Close()
	return nil
}

// verifyRuntimeDir enforces the three custody gates on an existing
// leaf without creating it. A missing leaf refuses with its own
// detail: absence on the authorizing path is a readiness fact the
// background entry maps to capability_unavailable, never a silent
// repair and never the custody code an unsafe leaf earns.
func verifyRuntimeDir(root, name string) error {
	// No platform arm here, for the same reason as the commit arm
	// above: only unix platforms reach this verify.
	rootHandle, err := secprim.OpenNoFollowDir(root)
	if err != nil {
		return classifyRootOpenError(err)
	}
	defer func() { _ = rootHandle.Close() }()
	if !handleBoundToPath(rootHandle, root) {
		return &Error{Code: CodeUnsafeRuntimeDir, Detail: "runtime containment"}
	}
	leafFd, err := unix.Openat(int(rootHandle.Fd()), name,
		unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		if errors.Is(err, unix.ENOENT) {
			return &Error{Code: CodeUnsafeRuntimeDir, Detail: "runtime absent"}
		}
		return &Error{Code: CodeUnsafeRuntimeDir, Detail: "runtime containment"}
	}
	leaf := os.NewFile(uintptr(leafFd), name)
	defer func() { _ = leaf.Close() }()
	info, err := leaf.Stat()
	if err != nil {
		return &Error{Code: CodeUnsafeRuntimeDir, Detail: "runtime mode"}
	}
	path := filepath.Join(root, name)
	mode := custodyModeForPath(path, info.Mode())
	if !mode.IsDir() || !ownerOnlyRuntimeDirMode(mode) {
		return &Error{Code: CodeUnsafeRuntimeDir, Detail: "runtime mode"}
	}
	return verifyOwnershipAt(path, info)
}

// ownerOnlyRuntimeDirMode pins mode 0700 exactly. SPEC §3.2 requires
// AX-owned runtime directories to use that mode; group/other bits,
// special bits, or a partial owner mode all refuse.
func ownerOnlyRuntimeDirMode(mode os.FileMode) bool {
	permissions := mode.Perm()
	return permissions&0o077 == 0 && permissions&0o700 == 0o700 &&
		mode&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) == 0
}
