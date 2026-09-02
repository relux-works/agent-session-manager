//go:build windows

package localstore

import (
	"fmt"
	"os"
)

func verifyOwnerFileInfo(os.FileInfo, os.FileMode) error {
	return fmt.Errorf("%w: native Windows user-only DACL enforcement is not implemented", ErrUnsupportedPlatform)
}
