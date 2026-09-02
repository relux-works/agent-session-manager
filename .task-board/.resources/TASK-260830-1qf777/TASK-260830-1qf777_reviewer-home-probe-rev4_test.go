package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Probe A: the deferred home-lookup failure must reach the caller from the
// real OSInputs capture, not only from a hand-set fixture field.
func TestProbeOSInputsPreservesRealHomeFailure(t *testing.T) {
	platform := hostPlatform(t)
	if platform != scalar.PlatformMacOS && platform != scalar.PlatformLinux {
		t.Skip("probe assumes $HOME semantics")
	}
	for _, specification := range OverrideRegistry() {
		t.Setenv(specification.Environment, "")
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", "")

	_, err := LoadOS(platform, nil)
	if !errors.Is(err, ErrPlatformDefaultUnavailable) {
		t.Fatalf("LoadOS(no home) error = %v, want ErrPlatformDefaultUnavailable", err)
	}
	var refusal *Error
	if !errors.As(err, &refusal) {
		t.Fatalf("LoadOS(no home) refusal = %#v", err)
	}
	joined, ok := refusal.Err.(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("LoadOS(no home) refusal cause is not a join: %#v", refusal.Err)
	}
	var carried bool
	for _, cause := range joined.Unwrap() {
		if !errors.Is(ErrPlatformDefaultUnavailable, cause) {
			carried = true
		}
	}
	if !carried {
		t.Fatalf("LoadOS(no home) dropped the captured os.UserHomeDir cause: %#v", refusal.Err)
	}
}

// Probe B: the real user home captured by OSInputs must reach the platform
// defaults at the production entry.
func TestProbeOSInputsCapturesRealHomeForPlatformDefaults(t *testing.T) {
	platform := hostPlatform(t)
	if platform != scalar.PlatformMacOS {
		t.Skip("probe pins the macOS Application Support default")
	}
	root := t.TempDir()
	for _, specification := range OverrideRegistry() {
		t.Setenv(specification.Environment, "")
	}
	t.Setenv("HOME", root)
	configDirectory := filepath.Join(root, ".config", "ax")
	if err := os.MkdirAll(configDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", "")

	snapshot, err := LoadOS(platform, nil)
	if err != nil {
		t.Fatalf("LoadOS(real home) error = %v", err)
	}
	got, ok := snapshot.Paths().Path(DataRoot)
	if !ok {
		t.Fatal("LoadOS(real home) resolved no data root")
	}
	want := filepath.Join(root, "Library", "Application Support", "ax")
	if got.Value.String() != want {
		t.Fatalf("LoadOS(real home) data root = %q, want %q", got.Value.String(), want)
	}
}
