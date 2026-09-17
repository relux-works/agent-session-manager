package config

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

func validV4Configuration() Configuration {
	configuration := validCurrentConfiguration()
	configuration.SchemaVersion = Version4
	configuration.Mesh.Transport = TransportSSHTLS13
	configuration.Mesh.HostChannel = &HostChannel{Version: HostChannelVersion, CredentialID: testDigest}
	return configuration
}

func decodeContextForTest() DecodeContext {
	return DecodeContext{RuntimePlatform: scalar.PlatformMacOS}
}

func TestDecodeConfiguration4(t *testing.T) {
	document, err := EncodeVersion4(validV4Configuration(), decodeContextForTest())
	if err != nil {
		t.Fatalf("EncodeVersion4 error = %v", err)
	}
	loaded, err := Decode(document, decodeContextForTest())
	if err != nil {
		t.Fatalf("Decode(v4) error = %v", err)
	}
	if loaded.SourceVersion != Version4 {
		t.Fatalf("Decode(v4) source = %q, want 4.0.0", loaded.SourceVersion)
	}
	if loaded.Value.Mesh.Transport != TransportSSHTLS13 {
		t.Fatalf("Decode(v4) transport = %q", loaded.Value.Mesh.Transport)
	}
	channel := loaded.Value.Mesh.HostChannel
	if channel == nil || channel.Version != HostChannelVersion || channel.CredentialID != testDigest {
		t.Fatalf("Decode(v4) host_channel = %+v", channel)
	}
	// Retained Configuration 3.0.0 members survive the migration shape.
	if len(loaded.Value.Mesh.Peers) != 1 || loaded.Value.HostName != "fixture-host" {
		t.Fatalf("Decode(v4) dropped retained members: %+v", loaded.Value.Mesh.Peers)
	}
}

func TestDecodeConfiguration4Refusals(t *testing.T) {
	base, err := EncodeVersion4(validV4Configuration(), decodeContextForTest())
	if err != nil {
		t.Fatalf("EncodeVersion4 error = %v", err)
	}
	valid := string(base)
	tests := []struct {
		name     string
		document string
	}{
		{"missing transport", strings.Replace(valid, "transport = 'ssh_tls13'\n", "", 1)},
		{"legacy transport", strings.Replace(valid, `transport = 'ssh_tls13'`, `transport = 'ssh'`, 1)},
		{"unknown transport", strings.Replace(valid, `transport = 'ssh_tls13'`, `transport = 'quic'`, 1)},
		{"missing host_channel", strings.Replace(valid, "[mesh.host_channel]\nversion = '1.0.0'\ncredential_id = '"+testDigest+"'\n", "", 1)},
		{"missing channel version", strings.Replace(valid, "version = '1.0.0'\n", "", 1)},
		{"missing credential_id", strings.Replace(valid, "credential_id = '"+testDigest+"'\n", "", 1)},
		{"wrong channel version", strings.Replace(valid, `version = '1.0.0'`, `version = '2.0.0'`, 1)},
		{"bad credential_id", strings.Replace(valid, "credential_id = '"+testDigest+"'", `credential_id = 'md5:00'`, 1)},
		{"unknown mesh key", strings.Replace(valid, "[mesh]\n", "[mesh]\npeer_override = true\n", 1)},
		{"per peer override", strings.Replace(valid, "[[mesh.peers]]", "[mesh.peers.host_channel]\nversion = '1.0.0'\n[[mesh.peers]]", 1)},
		{"unknown root key", strings.Replace(valid, "platform = 'macos'\n", "platform = 'macos'\ncertificate = 'bytes'\n", 1)},
		{"credential bytes in toml", valid + "[mesh.tls_identity]\ncertificate_pem = 'x'\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Decode([]byte(test.document), decodeContextForTest()); err == nil {
				t.Fatalf("Decode(v4 %s) succeeded, want refusal", test.name)
			} else {
				var refusal *DocumentError
				if !errors.As(err, &refusal) {
					t.Fatalf("Decode(v4 %s) error type = %T, want *DocumentError", test.name, err)
				}
			}
		})
	}
}

func TestDecodeConfiguration4MemberGates(t *testing.T) {
	base, err := EncodeVersion4(validV4Configuration(), decodeContextForTest())
	if err != nil {
		t.Fatalf("EncodeVersion4 error = %v", err)
	}
	valid := string(base)
	tests := []struct {
		name     string
		document string
	}{
		{"bad host_id", strings.Replace(valid, "host_id = '0198f4c8-4a10-7b22-8b3c-1234567890ab'", "host_id = 'not-a-uuid'", 1)},
		{"bad host_name", strings.Replace(valid, "host_name = 'fixture-host'", "host_name = ''", 1)},
		{"platform mismatch", strings.Replace(valid, "platform = 'macos'", "platform = 'linux'", 1)},
		{"bad service interval", strings.Replace(valid, "health_interval_seconds = 30", "health_interval_seconds = 4", 1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Decode([]byte(test.document), decodeContextForTest()); err == nil {
				t.Fatalf("Decode(v4 %s) succeeded, want refusal", test.name)
			} else if !errors.Is(err, ErrConfigValidation) {
				t.Fatalf("Decode(v4 %s) error = %v", test.name, err)
			}
		})
	}
}

func TestValidateConfigurationV4SchemaGate(t *testing.T) {
	// Both producers set Version4 before validating, so a mislabeled value
	// can only arrive through direct misuse. The gate stays and is pinned.
	configuration := validV4Configuration()
	configuration.SchemaVersion = CurrentVersion
	if err := validateConfigurationV4(&configuration, decodeContextForTest()); err == nil {
		t.Fatal("validateConfigurationV4(mislabeled value) succeeded, want refusal")
	} else if !errors.Is(err, ErrConfigValidation) {
		t.Fatalf("validateConfigurationV4 error = %v", err)
	}
}

func TestValidateMeshV4NilChannel(t *testing.T) {
	// Decode and EncodeVersion4 both guarantee a non-nil binding before this
	// check, so only direct misuse reaches it. The guard stays and is pinned.
	configuration := validV4Configuration()
	configuration.Mesh.HostChannel = nil
	if err := validateMeshV4(&configuration); err == nil {
		t.Fatal("validateMeshV4(nil channel) succeeded, want refusal")
	} else if !errors.Is(err, ErrConfigValidation) {
		t.Fatalf("validateMeshV4 error = %v", err)
	}
}

func TestConfiguration4NeverReadsAsLegacy(t *testing.T) {
	document, err := EncodeVersion4(validV4Configuration(), decodeContextForTest())
	if err != nil {
		t.Fatalf("EncodeVersion4 error = %v", err)
	}
	loaded, err := Decode(document, decodeContextForTest())
	if err != nil {
		t.Fatalf("Decode(v4) error = %v", err)
	}
	if loaded.SourceVersion == CurrentVersion {
		t.Fatal("Configuration 4.0.0 decoded with a 3.0.0 source version: implicit downgrade")
	}
}

func TestEncodeCurrentRefusesHostChannel(t *testing.T) {
	configuration := validV4Configuration()
	if _, err := EncodeCurrent(configuration, decodeContextForTest()); err == nil {
		t.Fatal("EncodeCurrent(v4 value) succeeded, want refusal: emitting v4 as v3 would drop the credential binding")
	} else if !errors.Is(err, ErrConfigEncode) {
		t.Fatalf("EncodeCurrent(v4 value) error = %v", err)
	}
}

func TestEncodeVersion4Refusals(t *testing.T) {
	plain := validCurrentConfiguration()
	if _, err := EncodeVersion4(plain, decodeContextForTest()); err == nil {
		t.Fatal("EncodeVersion4 without host_channel succeeded, want refusal")
	}
	downgraded := validV4Configuration()
	downgraded.Mesh.Transport = TransportSSH
	if _, err := EncodeVersion4(downgraded, decodeContextForTest()); err == nil {
		t.Fatal("EncodeVersion4(ssh transport) succeeded, want refusal")
	}
}

func TestCompatibilityV4(t *testing.T) {
	document, err := EncodeVersion4(validV4Configuration(), decodeContextForTest())
	if err != nil {
		t.Fatalf("EncodeVersion4 error = %v", err)
	}
	current, err := AssessCompatibility(document, Version4)
	if err != nil {
		t.Fatalf("AssessCompatibility(v4 doc, v4 reader) error = %v", err)
	}
	if current.Mode != CompatibilityCompatible || current.SourceVersion != Version4 {
		t.Fatalf("v4 reader assessment = %+v, want compatible/4.0.0", current)
	}
	legacy, err := AssessCompatibility(document, CurrentVersion)
	if err != nil {
		t.Fatalf("AssessCompatibility(v4 doc, v3 reader) error = %v", err)
	}
	if legacy.Mode != CompatibilityReadOnly {
		t.Fatalf("v3 reader assessment = %+v, want read-only-diagnostic", legacy)
	}
}

func TestMigrateRefusesV4Target(t *testing.T) {
	inputs := migrationInputs(t.TempDir(), t.TempDir()+"/config.toml")
	if _, err := Migrate(inputs, nil, MigrationOptions{TargetVersion: Version4}); err == nil {
		t.Fatal("Migrate(target 4.0.0) proceeded, want explicit-preview refusal")
	} else if !errors.Is(err, ErrMigrationV4Explicit) {
		t.Fatalf("Migrate(target 4.0.0) error = %v", err)
	}
}

func TestMigrateRefusesV4Downgrade(t *testing.T) {
	document, err := EncodeVersion4(validV4Configuration(), decodeContextForTest())
	if err != nil {
		t.Fatalf("EncodeVersion4 error = %v", err)
	}
	directory := t.TempDir()
	filename := directory + "/config.toml"
	if err := os.WriteFile(filename, document, 0o600); err != nil {
		t.Fatal(err)
	}
	inputs := migrationInputs(directory, filename)
	if _, err := Migrate(inputs, nil, MigrationOptions{TargetVersion: CurrentVersion}); err == nil {
		t.Fatal("Migrate(v4 to v3) succeeded, want downgrade refusal")
	} else if !errors.Is(err, ErrMigrationDowngrade) {
		t.Fatalf("Migrate(v4 to v3) error = %v", err)
	}
}
