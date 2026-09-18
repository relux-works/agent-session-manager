package axpane

import (
	"fmt"
	"os"
	"runtime"
)

// syncDirectory fsyncs a directory so an install inside it survives
// a crash. Opening a directory fails on Windows, where the install
// itself is the durability boundary; there the sync is skipped,
// never faked. This mirrors the landed matjournal discipline.
func syncDirectory(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open directory for fsync: %w", err)
	}
	defer func() { _ = directory.Close() }()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("fsync directory: %w", err)
	}
	return nil
}
