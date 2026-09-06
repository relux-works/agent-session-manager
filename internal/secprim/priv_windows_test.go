//go:build windows

package secprim

import (
	"testing"
)

// privilegedUser on Windows: mode-bit permission tests are skipped there
// unconditionally (Windows ACLs do not express them), so this is inert.
func privilegedUser() bool {
	return false
}

// makeFifo on Windows: named-pipe fixtures are not built there (the
// special-file subtest is skipped on Windows — see TestCheckRegularTarget),
// so this stub skips instead of building.
func makeFifo(t *testing.T, path string) {
	t.Helper()
	t.Skip("named-pipe fixtures are not built on Windows")
}
