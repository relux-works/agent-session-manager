//go:build darwin

package localstore

import "golang.org/x/sys/unix"

func atomicRenameNoReplace(oldPath, newPath string) error {
	return unix.RenamexNp(oldPath, newPath, unix.RENAME_EXCL)
}
