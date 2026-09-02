//go:build !darwin && !linux && !windows

package localstore

import "os"

func atomicRenameNoReplace(oldPath, newPath string) error {
	if err := os.Link(oldPath, newPath); err != nil {
		return err
	}
	return os.Remove(oldPath)
}
