package secprim

import (
	"os"
	"path/filepath"
	"syscall"
)

// openCommitRelative is the Windows best effort for the guard commit
// walk: Go has no openat, so each component is Lstat-checked against
// the root-joined path before the final open. A symlink or reparse
// point on the path returns the shared errCommitSymlink signal, which
// the caller refuses as "member symlink escape".
//
// The caller binds root to rootPath with guardRootMatches before
// descending, so the descent base is the verified guard root on this
// platform exactly as the verified handle is on unix: a foreign
// directory handle refuses with "guard root mismatch" in shared code
// before either walk runs, and both platforms answer it identically.
//
// Stated bounds. The check-then-open race remains: a swap between the
// component Lstat and the final open is not detected, and staging
// roots being owner-only directories only narrows that window to the
// owner. Windows behavior in this package is compile-and-vet only,
// never executed.
//
// Reparse coverage (established from the go1.25.5 runtime source,
// src/os/types_windows.go, (*fileStat).mode()). The Lstat gate refuses
// ModeSymlink (IO_REPARSE_TAG_SYMLINK) and ModeIrregular (the default
// arm: junctions and mount points, AppExecLinks, cloud placeholders),
// so a junction-to-directory refuses here rather than passing the
// directory check the way a symlink-only gate would admit it. Under
// GODEBUG=winsymlink=0 (modePreGo1_23) a mount point reports
// ModeSymlink instead — still refused, since the gate tests both bits.
// Two tags are counterexamples to any universal claim and are stated,
// not covered: IO_REPARSE_TAG_AF_UNIX surfaces as ModeSocket (it
// passes this Lstat gate and is refused later by CheckRegularTarget
// as "target special file"), and IO_REPARSE_TAG_DEDUP carries no type
// bit at all (regular by Go's explicit design decision, admitted).
// Neither is escape-capable, so the Section 16.3 reparse obligation
// is met for the escape-capable set under both mappings.
func openCommitRelative(root *os.File, rootPath string, _ string, segments []string) (*os.File, error) {
	_ = root
	current := rootPath
	for _, segment := range segments[:len(segments)-1] {
		current = filepath.Join(current, segment)
		info, err := os.Lstat(current)
		if err != nil {
			return nil, err
		}
		if info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
			return nil, errCommitSymlink
		}
		if !info.IsDir() {
			return nil, syscall.ENOTDIR
		}
	}
	full := filepath.Join(rootPath, filepath.Join(segments...))
	info, err := os.Lstat(full)
	if err != nil {
		return nil, err
	}
	if info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
		return nil, errCommitSymlink
	}
	return os.OpenFile(full, os.O_RDONLY, 0)
}

// clearNonblock is inert on Windows: the Windows open sets no
// non-blocking flag.
func clearNonblock(_ *os.File) error {
	return nil
}
