//go:build windows

package provider

import (
	"os"
	"testing"
	"time"
)

// windowsUnattestedFileInfo carries no UID metadata: native Windows has
// no UID model, so attestation must refuse for every input.
type windowsUnattestedFileInfo struct{}

func (windowsUnattestedFileInfo) Name() string       { return "unattested" }
func (windowsUnattestedFileInfo) Size() int64        { return 1 }
func (windowsUnattestedFileInfo) Mode() os.FileMode  { return 0o700 }
func (windowsUnattestedFileInfo) ModTime() time.Time { return time.Time{} }
func (windowsUnattestedFileInfo) IsDir() bool        { return false }
func (windowsUnattestedFileInfo) Sys() any           { return nil }

// TestFileOwnerUIDRefusesOnWindows proves the native Windows seam fails
// closed: with no UID model, every attestation refuses, so external
// executables stay undiscoverable there. This file compiles only under
// GOOS=windows; the rework evidence includes a GOOS=windows vet run.
func TestFileOwnerUIDRefusesOnWindows(t *testing.T) {
	if _, err := fileOwnerUID(windowsUnattestedFileInfo{}); err == nil {
		t.Fatal("fileOwnerUID attested an owner on native Windows, want refusal")
	}
}
