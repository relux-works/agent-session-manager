//go:build !windows

package localstore

import (
	"fmt"
	"os"
	"syscall"
)

func verifyOwnerFileInfo(info os.FileInfo, want os.FileMode) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("%w: ownership metadata unavailable", ErrUnsafeOwnership)
	}
	if stat.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("%w: path belongs to another user", ErrUnsafeOwnership)
	}
	if info.Mode().Perm() != want {
		return fmt.Errorf("%w: mode is %04o, want %04o", ErrUnsafeOwnership, info.Mode().Perm(), want)
	}
	return nil
}
