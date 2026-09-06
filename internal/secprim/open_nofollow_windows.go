package secprim

import (
	"os"
)

// openNoFollow is the Windows best effort: Go has no O_NOFOLLOW, so the
// caller (OpenNoFollowFile) stats with Lstat first and refuses symlinks,
// reparse points, and non-regular targets before opening. The Lstat/open
// check-then-open race remains: a swapped path between the two calls is
// not detected. Staging roots are owner-only directories, which narrows
// the swap window to the owner, but the bound is stated, not closed: a
// compromised owner account can still win the race. Unix builds close it
// with O_NOFOLLOW instead.
//
// Reparse coverage (established from the go1.25.5 runtime source,
// src/os/types_windows.go, (*fileStat).mode(): Go sets ModeSymlink
// only for IO_REPARSE_TAG_SYMLINK). The default-arm reparse points —
// junctions and mount points, AppExecLinks, cloud placeholders —
// surface as ModeIrregular, never as ModeSymlink, so a symlink-only
// gate would admit a junction-to-directory as a plain directory. The
// gate below therefore refuses ModeIrregular alongside ModeSymlink
// (and stays refused under GODEBUG=winsymlink=0, where a mount point
// reports ModeSymlink instead). Two tags are counterexamples to any
// universal claim and are stated, not covered:
// IO_REPARSE_TAG_AF_UNIX surfaces as ModeSocket (it passes this Lstat
// gate and is refused later by CheckRegularTarget as "target special
// file"), and IO_REPARSE_TAG_DEDUP carries no type bit at all
// (regular by Go's explicit design decision, admitted). Neither is
// escape-capable. Windows behavior in this package is compile-and-vet
// only, never executed: the refusal shape above is established from
// the cited source, not from a run.
func openNoFollow(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
		return nil, errIsSymlink
	}
	return os.OpenFile(path, os.O_RDONLY, 0)
}

// openNoFollowDir opens a directory for validation walks after the same
// Lstat gate. The same check-then-open bound applies.
func openNoFollowDir(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
		return nil, errIsSymlink
	}
	return os.Open(path)
}
