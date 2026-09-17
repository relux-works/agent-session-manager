//go:build !windows

package hosttrust

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
)

// platformModeOK reports whether Unix permission bits satisfy the exact
// owner-only requirement. Any group/world access already refuses here.
func platformModeOK(info fs.FileInfo, mode fs.FileMode) bool {
	return info.Mode().Perm() == mode
}

// secureStaged is a no-op on Unix: staging files are created with exact
// owner-only modes, which the caller verifies after install.
func secureStaged(_ string, _ bool) error { return nil }

// verifyOwner enforces owner-only custody on Unix: the path must belong to
// the effective UID. Mode bits are checked by the caller against the exact
// 0700/0600 requirement, so any group/world access already refused there.
func verifyOwner(_ string, info fs.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return trustError(TrustError{Operation: "inspect ownership", Err: errors.Join(ErrTrustUnsafeCustody, errors.New("ownership metadata unavailable"))})
	}
	if stat.Uid != uint32(os.Geteuid()) {
		return trustError(TrustError{Operation: "inspect ownership", Err: ErrTrustUnsafeCustody})
	}
	return nil
}
