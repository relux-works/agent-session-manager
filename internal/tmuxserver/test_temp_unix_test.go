//go:build !windows

package tmuxserver

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

var compactTempSequence atomic.Uint64

// shortTestTempDir keeps production Unix socket fixtures under sun_path while
// retaining testing.T ownership: create with t.TempDir, relocate the owned
// directory under canonical /var/tmp, and remove the relocated path at cleanup.
func shortTestTempDir(t *testing.T) string {
	t.Helper()
	t.Setenv("TMPDIR", "/var/tmp")
	generated := t.TempDir()
	shortParent, err := filepath.EvalSymlinks("/var/tmp")
	if err != nil {
		t.Fatalf("resolve short temporary parent: %v", err)
	}
	for {
		candidate := filepath.Join(shortParent, fmt.Sprintf("ax%d-%d", os.Getpid(), compactTempSequence.Add(1)))
		if err := os.Rename(generated, candidate); err != nil {
			if os.IsExist(err) {
				continue
			}
			t.Fatalf("relocate testing.T temporary directory under socket path bound: %v", err)
		}
		t.Cleanup(func() {
			if err := os.RemoveAll(candidate); err != nil {
				t.Errorf("remove relocated testing.T temporary directory: %v", err)
			}
		})
		return candidate
	}
}
