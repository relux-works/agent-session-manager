//go:build windows

package tmuxserver

import "testing"

func shortTestTempDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}
