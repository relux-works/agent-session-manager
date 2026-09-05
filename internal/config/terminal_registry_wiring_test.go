package config

import (
	"errors"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// TestEncodeCurrentRefusesReservedBackendNamespace proves the terminal backend
// registry is a live production call site of configuration validation: an
// ax.-namespaced ID outside the two canonical built-ins is refused at the
// exact clause, while the built-ins and vendor namespaces keep working.
func TestEncodeCurrentRefusesReservedBackendNamespace(t *testing.T) {
	t.Parallel()

	t.Run("selected reserved ID", func(t *testing.T) {
		t.Parallel()
		configuration := validCurrentConfiguration()
		configuration.Terminal.BackendID = "ax.evil"
		_, err := EncodeCurrent(configuration, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
		requireConfigClause(t, err, "terminal.backend_id")
	})

	t.Run("external trust reserved ID", func(t *testing.T) {
		t.Parallel()
		configuration := validCurrentConfiguration()
		configuration.Terminal.ExternalTrust = []ExternalExecutableTrust{{
			BackendID:        "ax.evil",
			ExecutablePath:   "/opt/vendor/bin/ax-backend-evil",
			ExecutableDigest: testDigest,
			Enabled:          true,
		}}
		_, err := EncodeCurrent(configuration, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
		requireConfigClause(t, err, "terminal.external_trust[0].backend_id")
	})

	// External trust malformed ID proves the registry-grammar call in the
	// trust loop is load-bearing against deletion: without it this entry
	// sails past the ax. reservation (no such prefix), the path and digest
	// checks, and the registered-set test, and is admitted. Reverting the
	// call to the previous local pattern is instead an equivalent mutant —
	// the old grammar refuses this input too, and the error clause is
	// intentionally non-distinguishing — so that variant is documented as
	// equivalent rather than covered.
	t.Run("external trust malformed ID", func(t *testing.T) {
		t.Parallel()
		configuration := validCurrentConfiguration()
		configuration.Terminal.ExternalTrust = []ExternalExecutableTrust{{
			BackendID:        "INVALID_ID",
			ExecutablePath:   "/opt/vendor/bin/ax-backend-term",
			ExecutableDigest: testDigest,
			Enabled:          true,
		}}
		_, err := EncodeCurrent(configuration, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
		requireConfigClause(t, err, "terminal.external_trust[0].backend_id")
	})

	t.Run("external trust claiming a built-in ID is ambiguous", func(t *testing.T) {
		t.Parallel()
		configuration := validCurrentConfiguration()
		configuration.Terminal.ExternalTrust = []ExternalExecutableTrust{{
			BackendID:        "ax.tmux",
			ExecutablePath:   "/opt/vendor/bin/ax-backend-tmux",
			ExecutableDigest: testDigest,
			Enabled:          true,
		}}
		_, err := EncodeCurrent(configuration, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
		requireConfigClause(t, err, "terminal.external_trust[0].backend_id")
	})

	t.Run("canonical built-ins still admitted", func(t *testing.T) {
		t.Parallel()
		configuration := validCurrentConfiguration()
		configuration.Terminal.BackendID = "ax.tmux"
		if _, err := EncodeCurrent(configuration, DecodeContext{RuntimePlatform: scalar.PlatformMacOS}); err != nil {
			t.Fatalf("EncodeCurrent(ax.tmux) error = %v", err)
		}
	})
}

// TestDecodeRefusesNonConptyBackendOnNativeWindows pins the Section 6.2
// sentence "On native Windows, ... terminal backend MUST be conpty"
// (SPEC.md:2415-2418) through the legacy translation path: a v1 document
// naming tmux on Windows refuses at the production load entry, while conpty
// loads. The v3 lane split is pinned separately by the "built-in backend
// unsupported on platform" Decode case and the "conpty platform" refusal
// case; this test closes the legacy arm without extending the rule to
// third-party registered IDs, which Section 6.5 admits as valid.
func TestDecodeRefusesNonConptyBackendOnNativeWindows(t *testing.T) {
	t.Parallel()

	tmux := append(minimalValidConfigVersion(scalar.PlatformWindows, Version1), []byte("\n[terminal]\nbackend = \"tmux\"\n")...)
	if _, err := Decode(tmux, DecodeContext{RuntimePlatform: scalar.PlatformWindows}); err == nil {
		t.Fatal("Decode(v1 backend=tmux on Windows) error = nil, want refusal")
	} else if !errors.Is(err, ErrConfigValidation) {
		t.Fatalf("Decode(v1 backend=tmux on Windows) error = %v, want ErrConfigValidation", err)
	}

	conpty := append(minimalValidConfigVersion(scalar.PlatformWindows, Version1), []byte("\n[terminal]\nbackend = \"conpty\"\n")...)
	loaded, err := Decode(conpty, DecodeContext{RuntimePlatform: scalar.PlatformWindows})
	if err != nil {
		t.Fatalf("Decode(v1 backend=conpty on Windows) error = %v", err)
	}
	if loaded.Value.Terminal.BackendID != "ax.conpty" {
		t.Fatalf("migrated terminal backend = %q, want ax.conpty", loaded.Value.Terminal.BackendID)
	}
}
