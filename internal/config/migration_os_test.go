package config

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// migrateOSFixture seeds a real directory tree and returns explicit overrides
// for all five Section 3.2 path classes, so MigrateOS resolves only injected
// values and never the operator's real home, XDG, or Application Support
// locations. The seeded document carries the legacy terminal table so the
// fixture also exercises legacy translation through the durable entry.
func migrateOSFixture(t *testing.T, platform scalar.Platform, version string) (configDirectory, filename string, original []byte, overrides Overrides) {
	t.Helper()
	root := t.TempDir()
	overrides = make(Overrides, len(OverrideRegistry()))
	for _, specification := range OverrideRegistry() {
		if specification.Class == ConfigFile {
			continue
		}
		directory := filepath.Join(root, string(specification.Class))
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		overrides[specification.Class] = directory
	}
	configDirectory = filepath.Join(root, "config")
	if err := os.Mkdir(configDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	filename = filepath.Join(configDirectory, "config.toml")
	original = append(minimalValidConfigVersion(platform, version), []byte("\n[terminal]\nbackend = \"tmux\"\n")...)
	if err := os.WriteFile(filename, original, 0o600); err != nil {
		t.Fatal(err)
	}
	overrides[ConfigFile] = filename
	return configDirectory, filename, original, overrides
}

// TestMigrateOSPerformsOneDurableMigrationAtTheRealProcessEntry drives the OS
// production entry, not the injected Migrate seam: MigrateOS captures real
// process inputs through OSInputs and mutates a real directory. It proves the
// backup, the atomic replacement, the preserved source mode, the legacy
// terminal translation, and that a second run of the same entry is a no-op.
func TestMigrateOSPerformsOneDurableMigrationAtTheRealProcessEntry(t *testing.T) {
	platform := hostPlatform(t)
	directory, filename, original, overrides := migrateOSFixture(t, platform, Version1)

	result, err := MigrateOS(platform, overrides, MigrationOptions{
		TargetVersion: CurrentVersion, GeneratedSummaryUpgradeChoice: "local_only",
	})
	if err != nil {
		t.Fatalf("MigrateOS(v1 to v3) error = %v", err)
	}
	if !result.Changed || result.SourceVersion != Version1 || result.TargetVersion != CurrentVersion {
		t.Fatalf("MigrateOS(v1 to v3) result = %#v", result)
	}
	if want := filename + ".bak." + Version1; result.BackupPath != want {
		t.Fatalf("MigrateOS().BackupPath = %q, want %q", result.BackupPath, want)
	}

	backup, err := os.ReadFile(result.BackupPath)
	if err != nil {
		t.Fatalf("read migration backup: %v", err)
	}
	if !bytes.Equal(backup, original) {
		t.Fatal("MigrateOS backup does not preserve the exact source bytes")
	}
	backupInfo, err := os.Stat(result.BackupPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := backupInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("backup mode = %04o, want 0600", got)
	}
	migratedInfo, err := os.Stat(filename)
	if err != nil {
		t.Fatal(err)
	}
	if got := migratedInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("migrated source mode = %04o, want the preserved 0600", got)
	}

	snapshot, err := LoadOS(platform, overrides)
	if err != nil {
		t.Fatalf("LoadOS(migrated document) error = %v", err)
	}
	loaded, ok := snapshot.Configuration()
	if !ok || loaded.SourceVersion != CurrentVersion {
		t.Fatalf("LoadOS(migrated document) = %#v, present=%v", loaded, ok)
	}
	if loaded.Value.Terminal.BackendID != "ax.tmux" {
		t.Fatalf("migrated terminal backend = %q, want ax.tmux", loaded.Value.Terminal.BackendID)
	}
	if loaded.Value.Directory.GeneratedSummaryUpgradeChoice != "local_only" {
		t.Fatalf("migrated generated-summary choice = %q", loaded.Value.Directory.GeneratedSummaryUpgradeChoice)
	}
	migrated, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}

	second, err := MigrateOS(platform, overrides, MigrationOptions{
		TargetVersion: CurrentVersion, GeneratedSummaryUpgradeChoice: "local_only",
	})
	if err != nil {
		t.Fatalf("MigrateOS(already current) error = %v", err)
	}
	if second.Changed || second.BackupPath != "" {
		t.Fatalf("MigrateOS(already current) result = %#v, want an unchanged no-op", second)
	}
	afterSecond, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(afterSecond, migrated) {
		t.Fatal("second MigrateOS run rewrote an already-current document")
	}
	if afterBackup, _ := os.ReadFile(result.BackupPath); !bytes.Equal(afterBackup, original) {
		t.Fatal("second MigrateOS run rewrote the published backup")
	}
	assertNoStagingLeak(t, directory)
}

// TestMigrateOSAppliesTheNoFollowMutatingSeamInBothDirections pins the
// declared asymmetry between the two seams at the real OS entry. The read seam
// resolves symlinks, so Load admits a configuration symlinked onto a regular
// file; the mutating seam does not follow symlinks, because an atomic rename
// would replace the operator's link instead of the document it points at. Each
// case differs from the other in exactly one respect - whether the selected
// path is the regular file itself or a symlink onto it.
func TestMigrateOSAppliesTheNoFollowMutatingSeamInBothDirections(t *testing.T) {
	platform := hostPlatform(t)

	t.Run("regular selected source migrates", func(t *testing.T) {
		directory, filename, original, overrides := migrateOSFixture(t, platform, Version1)

		result, err := MigrateOS(platform, overrides, MigrationOptions{
			TargetVersion: CurrentVersion, GeneratedSummaryUpgradeChoice: "local_only",
		})
		if err != nil || !result.Changed {
			t.Fatalf("MigrateOS(regular selected source) = %#v, %v", result, err)
		}
		after, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(after, original) {
			t.Fatal("MigrateOS(regular selected source) did not replace the document")
		}
		assertNoStagingLeak(t, directory)
	})

	t.Run("symlinked selected source is refused and nothing is mutated", func(t *testing.T) {
		directory, target, original, overrides := migrateOSFixture(t, platform, Version1)
		link := symlinkFixture(t, target, filepath.Join(directory, "linked-config.toml"))
		overrides[ConfigFile] = link

		// The read seam accepts the same selected path, so the refusal below
		// isolates the mutating seam rather than a rejected load.
		if _, err := LoadOS(platform, overrides); err != nil {
			t.Fatalf("LoadOS(symlinked configuration) error = %v, want the resolved document", err)
		}

		_, err := MigrateOS(platform, overrides, MigrationOptions{
			TargetVersion: CurrentVersion, GeneratedSummaryUpgradeChoice: "local_only",
		})
		if !errors.Is(err, ErrConfigNotRegular) || !errors.Is(err, ErrMigrationWrite) {
			t.Fatalf("MigrateOS(symlinked configuration) error = %v, want ErrMigrationWrite and ErrConfigNotRegular", err)
		}
		// The rendered production MigrationError must not echo the selected
		// machine-local path even though the wrapped chain carries it.
		if strings.Contains(err.Error(), directory) || strings.Contains(err.Error(), link) {
			t.Fatalf("MigrateOS() error echoed a machine-local path: %v", err)
		}

		linkInfo, statErr := os.Lstat(link)
		if statErr != nil {
			t.Fatal(statErr)
		}
		if linkInfo.Mode()&fs.ModeSymlink == 0 {
			t.Fatal("refused migration replaced the symlink with a regular file")
		}
		if resolved, readErr := os.Readlink(link); readErr != nil || resolved != target {
			t.Fatalf("refused migration retargeted the symlink to %q (error %v)", resolved, readErr)
		}
		assertMigrationSourceUntouched(t, target, original)
		assertNoMigrationArtifacts(t, directory, link)
		assertNoMigrationArtifacts(t, directory, target)
	})
}

// TestMigrateOSRefusesAnUnknownPlatformBeforeTouchingTheFilesystem proves
// MigrateOS propagates the OSInputs refusal instead of continuing with an empty
// Inputs value. The assertion names the refusing operation rather than only its
// sentinel: ResolvePaths refuses the same platform later with ErrInvalidContext
// under "validate inputs", so a sentinel-only assertion would pass even when
// MigrateOS discards the OSInputs error and falls through.
func TestMigrateOSRefusesAnUnknownPlatformBeforeTouchingTheFilesystem(t *testing.T) {
	directory, filename, original, overrides := migrateOSFixture(t, hostPlatform(t), Version1)

	_, err := MigrateOS(scalar.Platform("solaris"), overrides, MigrationOptions{
		TargetVersion: CurrentVersion, GeneratedSummaryUpgradeChoice: "local_only",
	})
	if !errors.Is(err, ErrInvalidContext) {
		t.Fatalf("MigrateOS(unknown platform) error = %v, want ErrInvalidContext", err)
	}
	var refusal *Error
	if !errors.As(err, &refusal) || refusal.Operation != "capture OS inputs" {
		t.Fatalf("MigrateOS(unknown platform) refused at %#v, want the OSInputs capture guard", err)
	}
	assertMigrationSourceUntouched(t, filename, original)
	assertNoMigrationArtifacts(t, directory, filename)
}

// TestMigrateOSMigratesTheHomeDerivedConfigurationAtTheRealProcessEntry drives
// the second production consumer of the OSInputs home capture. Every other
// MigrateOS case supplies explicit overrides for all five path classes, so
// none of them would notice a capture that returned an empty or constant home:
// the durable mutation would land somewhere other than the operator's real
// configuration, or refuse outright, and the suite would stay green. Here the
// only thing that selects the mutated file is the captured home, and the file
// is placed at the Section 3.2 home-derived default of each drivable lane.
func TestMigrateOSMigratesTheHomeDerivedConfigurationAtTheRealProcessEntry(t *testing.T) {
	for _, platform := range homeDrivenPlatforms(t) {
		t.Run(platform.String(), func(t *testing.T) {
			home := t.TempDir()
			defaults := homeDerivedPlatformDefaults(t, platform, home)
			clearAmbientPathEnvironment(t, platform)
			t.Setenv(homeEnvironmentName(), home)

			filename := defaults[ConfigFile]
			if err := os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
				t.Fatal(err)
			}
			original := minimalValidConfigVersion(platform, Version1)
			if err := os.WriteFile(filename, original, 0o600); err != nil {
				t.Fatal(err)
			}

			result, err := MigrateOS(platform, nil, MigrationOptions{
				TargetVersion: CurrentVersion, GeneratedSummaryUpgradeChoice: "local_only",
			})
			if err != nil {
				t.Fatalf("MigrateOS(home-derived source) error = %v", err)
			}
			if !result.Changed || result.SourceVersion != Version1 || result.TargetVersion != CurrentVersion {
				t.Fatalf("MigrateOS(home-derived source) result = %#v", result)
			}
			if want := filename + ".bak." + Version1; result.BackupPath != want {
				t.Fatalf("MigrateOS(home-derived source).BackupPath = %q, want the backup beside the captured-home source %q", result.BackupPath, want)
			}
			backup, err := os.ReadFile(result.BackupPath)
			if err != nil {
				t.Fatalf("read the home-derived backup: %v", err)
			}
			if !bytes.Equal(backup, original) {
				t.Fatal("MigrateOS backup beside the home-derived source does not preserve the exact source bytes")
			}

			snapshot, err := LoadOS(platform, nil)
			if err != nil {
				t.Fatalf("LoadOS(home-derived migrated document) error = %v", err)
			}
			loaded, ok := snapshot.Configuration()
			if !ok || loaded.SourceVersion != CurrentVersion {
				t.Fatalf("LoadOS(home-derived migrated document) = %#v, present=%v", loaded, ok)
			}
			assertNoStagingLeak(t, filepath.Dir(filename))
		})
	}
}
