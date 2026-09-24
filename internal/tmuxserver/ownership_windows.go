//go:build windows

package tmuxserver

import "io/fs"

// currentUID reports the process identity the broker same-user arm
// decides over. On Windows it is a compile-only stub returning a
// value no authenticated broker reports: every production entry
// refuses before consulting process identity, because native Windows
// MUST NOT claim tmux or tmux resurrection (SPEC 1458) and the
// Windows commit/verify arms refuse every platform. The stub exists
// so the shared acquisition file compiles under GOOS=windows.
func currentUID() uint32 { return 0 }

// ownershipMatches is the Windows stub of the unix ownership
// comparison: no custody can be proven, so it fails closed. Every
// production entry refuses before reaching it (B2).
func ownershipMatches(_ fs.FileInfo) bool { return false }
