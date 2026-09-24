//go:build !windows

package tmuxserver

import (
	"io/fs"
	"os"
	"syscall"
)

// effectiveUID reports the process effective UID. It is a variable
// only so seam-failure tests can force the foreign-ownership path
// through the production entries: staging a foreign-owned directory
// needs privilege no test may assume, and without the seam the
// ownership refusal is untestable. Production code never reassigns
// it.
var effectiveUID = os.Geteuid

// currentUID reports the process effective UID the broker same-user
// arm decides over. It reads the effectiveUID seam so tests stage a
// foreign broker without privilege; production code never reassigns
// the seam.
func currentUID() uint32 { return uint32(effectiveUID()) }

// verifyOwnership enforces the ownership gate on unix: the directory
// must belong to the effective UID. Mode bits are checked by the
// caller against the exact 0700 requirement, so any group or world
// access already refused there. A stat without ownership metadata
// fails closed: unreadable ownership is never legitimate custody.
func verifyOwnership(info fs.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return &Error{Code: CodeUnsafeRuntimeDir, Detail: "runtime ownership"}
	}
	if stat.Uid != uint32(effectiveUID()) {
		return &Error{Code: CodeUnsafeRuntimeDir, Detail: "runtime ownership"}
	}
	return nil
}

// verifyOwnershipAt applies the same effective-UID custody check while
// retaining the path-aware fixture seam used to enumerate component owners.
// With no test projection, this is byte-for-byte the same decision as
// verifyOwnership.
func verifyOwnershipAt(path string, info fs.FileInfo) error {
	if !custodyOwnershipMatches(path, info) {
		return &Error{Code: CodeUnsafeRuntimeDir, Detail: "runtime ownership"}
	}
	return nil
}

// ownershipMatches reports whether info belongs to the effective UID.
// A stat without ownership metadata fails closed. The bind step uses
// the boolean form because the custody detail it reports names the
// socket root, not the runtime leaf.
func ownershipMatches(info fs.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	return stat.Uid == uint32(effectiveUID())
}
