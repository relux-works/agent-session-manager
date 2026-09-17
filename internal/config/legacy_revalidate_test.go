package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/hosttrust"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// revalidateLegacySource pins the legacy source inside the authorization
// transaction: the current document must still carry the selected version
// and byte-exact bytes. The legacy-revalidate-skip narrowing plant drops
// the byte comparison while keeping the version check; it is killed by
// the same-version-different-bytes case below.

func writeRevalidateDoc(t *testing.T, filename string, document []byte) {
	t.Helper()
	if err := os.WriteFile(filename, document, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRevalidateLegacySourceAcceptsUnchanged(t *testing.T) {
	directory := t.TempDir()
	filename := filepath.Join(directory, "config.toml")
	writeRevalidateDoc(t, filename, minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion))
	inputs := migrationInputs(directory, filename)
	snapshot, err := Load(inputs, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := revalidateLegacySource(inputs, nil, snapshot); err != nil {
		t.Fatalf("revalidateLegacySource(unchanged) err = %v", err)
	}
}

func TestRevalidateLegacySourceRefusesChangedBytes(t *testing.T) {
	directory := t.TempDir()
	filename := filepath.Join(directory, "config.toml")
	selected := append(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), []byte("# selected by the operator\n")...)
	moved := append(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), []byte("# moved by a concurrent writer\n")...)
	writeRevalidateDoc(t, filename, selected)
	inputs := migrationInputs(directory, filename)
	snapshot, err := Load(inputs, nil)
	if err != nil {
		t.Fatal(err)
	}
	writeRevalidateDoc(t, filename, moved)
	if err := revalidateLegacySource(inputs, nil, snapshot); !errors.Is(err, ErrMigrationStaleSource) {
		t.Fatalf("revalidateLegacySource(changed bytes) err = %v, want %v", err, ErrMigrationStaleSource)
	}
}

func TestRevalidateLegacySourceRefusesUpgradedVersion(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	snapshot, err := Load(fixture.inputs, nil)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyV4(fixture.inputs, nil, preview, true, fixture.now); err != nil {
		t.Fatal(err)
	}
	// A joint Configuration 4.0.0 commit landed after the legacy source
	// was selected: the delayed legacy write must refuse stale rather
	// than reinstall older bytes over the committed state.
	if err := revalidateLegacySource(fixture.inputs, nil, snapshot); !errors.Is(err, ErrMigrationStaleSource) {
		t.Fatalf("revalidateLegacySource(upgraded to v4) err = %v, want %v", err, ErrMigrationStaleSource)
	}
}

func TestMigrateRefusesWhenTrustStoreCannotOpen(t *testing.T) {
	directory := t.TempDir()
	filename := filepath.Join(directory, "config.toml")
	original := append(minimalValidConfigVersion(scalar.PlatformMacOS, Version1), []byte("\n[terminal]\nbackend = \"tmux\"\n")...)
	writeRevalidateDoc(t, filename, original)
	stateDir := filepath.Join(directory, "state")
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		t.Fatal(err)
	}
	// A regular file where the host-channel directory belongs makes the
	// coordinating store unopenable: the migration must refuse closed
	// instead of writing beside unknown authority.
	if err := os.WriteFile(filepath.Join(stateDir, "host-channel"), []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	inputs := Inputs{
		Platform: scalar.PlatformMacOS, HomeDir: directory, TempDir: directory, WorkingDir: directory,
		LookupEnv: func(name string) (string, bool) {
			switch name {
			case "AX_CONFIG":
				return filename, true
			case "AX_STATE_DIR":
				return stateDir, true
			}
			return "", false
		},
		Stat: os.Stat, ReadFile: os.ReadFile,
	}
	_, err := Migrate(inputs, nil, MigrationOptions{TargetVersion: Version2, GeneratedSummaryUpgradeChoice: "reference_only"})
	if err == nil {
		t.Fatal("Migrate(unopenable store) succeeded, want refusal")
	}
	if !errors.Is(err, hosttrust.ErrTrustUnsafeCustody) {
		t.Fatalf("Migrate(unopenable store) err = %v, want custody refusal", err)
	}
	after, readErr := os.ReadFile(filename)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != string(original) {
		t.Fatal("refused migration changed the configuration file")
	}
}

func TestRevalidateLegacySourceRefusesMissingDocument(t *testing.T) {
	directory := t.TempDir()
	filename := filepath.Join(directory, "config.toml")
	writeRevalidateDoc(t, filename, minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion))
	inputs := migrationInputs(directory, filename)
	snapshot, err := Load(inputs, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filename); err != nil {
		t.Fatal(err)
	}
	if err := revalidateLegacySource(inputs, nil, snapshot); err == nil {
		t.Fatal("revalidateLegacySource(missing document) succeeded, want refusal")
	}
}
