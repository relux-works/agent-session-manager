package config

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// TestMigrateRefusesEveryTargetOutsideTheUpgradeVocabulary drives the real
// Migrate entry. The Version1 case isolates the non-upgrade-target clause: it
// is a known configuration version, so no earlier disjunct can refuse it.
func TestMigrateRefusesEveryTargetOutsideTheUpgradeVocabulary(t *testing.T) {
	refused := map[string]string{
		"empty target":                    "",
		"unknown newer target":            "9.9.9",
		"unknown patch target":            "3.0.1",
		"non-semver target":               "v3",
		"known but not an upgrade target": Version1,
	}
	for name, target := range refused {
		name, target := name, target
		t.Run(name, func(t *testing.T) {
			directory, filename, original := seedMigrationSource(t, Version2)
			_, err := Migrate(migrationInputs(directory, filename), nil, MigrationOptions{TargetVersion: target})
			if !errors.Is(err, ErrMigrationTarget) {
				t.Fatalf("Migrate(target=%q) error = %v, want ErrMigrationTarget", target, err)
			}
			assertMigrationSourceUntouched(t, filename, original)
			assertNoMigrationArtifacts(t, directory, filename)
		})
	}

	// Positive control: the two accepted targets are not refused, so the gate
	// is a vocabulary and not a blanket rejection.
	for _, target := range []string{Version2, CurrentVersion} {
		directory, filename, _ := seedMigrationSource(t, Version1)
		result, err := Migrate(migrationInputs(directory, filename), nil, MigrationOptions{
			TargetVersion: target, GeneratedSummaryUpgradeChoice: "local_only",
		})
		if err != nil || !result.Changed {
			t.Fatalf("Migrate(target=%q) = %#v, %v", target, result, err)
		}
	}
}

// TestMigrateRefusesAbsentSourceWithoutWriting covers the load-source guard on
// an admissible but not-yet-created configuration path, which Load itself
// accepts rather than refuses.
func TestMigrateRefusesAbsentSourceWithoutWriting(t *testing.T) {
	directory := t.TempDir()
	filename := filepath.Join(directory, "config.toml")

	snapshot, err := Load(migrationInputs(directory, filename), nil)
	if err != nil || snapshot.ConfigPresent() {
		t.Fatalf("Load(absent source) = present %v, %v; the migration guard must be the only refusal", snapshot.ConfigPresent(), err)
	}
	_, err = Migrate(migrationInputs(directory, filename), nil, MigrationOptions{TargetVersion: CurrentVersion})
	if !errors.Is(err, ErrMigrationSourceAbsent) {
		t.Fatalf("Migrate(absent source) error = %v, want ErrMigrationSourceAbsent", err)
	}
	if _, statErr := os.Stat(filename); !errors.Is(statErr, fs.ErrNotExist) {
		t.Fatalf("Migrate(absent source) created the selected file: %v", statErr)
	}
	assertNoMigrationArtifacts(t, directory, filename)
}

// TestMigrateRefusesDisclosureChoiceOutsideTheClosedVocabulary proves the
// generated-summary gate in both directions at the production entry.
func TestMigrateRefusesDisclosureChoiceOutsideTheClosedVocabulary(t *testing.T) {
	refused := map[string]string{
		"absent choice":       "",
		"unlisted value":      "public",
		"hyphenated variant":  "local-only",
		"uppercase variant":   "LOCAL_ONLY",
		"padded variant":      " local_only",
		"listed value prefix": "local",
	}
	for name, choice := range refused {
		name, choice := name, choice
		t.Run(name, func(t *testing.T) {
			directory, filename, original := seedMigrationSource(t, Version1)
			_, err := Migrate(migrationInputs(directory, filename), nil, MigrationOptions{
				TargetVersion: CurrentVersion, GeneratedSummaryUpgradeChoice: choice,
			})
			if !errors.Is(err, ErrMigrationChoiceRequired) {
				t.Fatalf("Migrate(choice=%q) error = %v, want ErrMigrationChoiceRequired", choice, err)
			}
			assertMigrationSourceUntouched(t, filename, original)
			assertNoMigrationArtifacts(t, directory, filename)
		})
	}
	for _, choice := range []string{"local_only", "mesh_sanitized", "reference_only"} {
		choice := choice
		t.Run("accepted "+choice, func(t *testing.T) {
			directory, filename, _ := seedMigrationSource(t, Version1)
			result, err := Migrate(migrationInputs(directory, filename), nil, MigrationOptions{
				TargetVersion: CurrentVersion, GeneratedSummaryUpgradeChoice: choice,
			})
			if err != nil || !result.Changed {
				t.Fatalf("Migrate(choice=%q) = %#v, %v", choice, result, err)
			}
			loaded := loadMigrated(t, directory, filename)
			if loaded.Value.Directory.GeneratedSummaryUpgradeChoice != choice {
				t.Fatalf("migrated choice = %q, want %q", loaded.Value.Directory.GeneratedSummaryUpgradeChoice, choice)
			}
		})
	}
}

// TestMigrateRefusesExistingBackupThatIsNotTheExactOwnerOnlySource is the
// headline prove-the-backup gate: a pre-existing backup is only reused when it
// is byte-identical to the source and readable by nobody else.
func TestMigrateRefusesExistingBackupThatIsNotTheExactOwnerOnlySource(t *testing.T) {
	t.Run("stale bytes", func(t *testing.T) {
		directory, filename, original := seedMigrationSource(t, Version1)
		backup := filename + ".bak." + Version1
		stale := append(append([]byte(nil), original...), []byte("\n# not the source\n")...)
		writeBackupFixture(t, backup, stale, 0o600)

		_, err := Migrate(migrationInputs(directory, filename), nil, MigrationOptions{
			TargetVersion: CurrentVersion, GeneratedSummaryUpgradeChoice: "local_only",
		})
		if !errors.Is(err, ErrMigrationBackup) {
			t.Fatalf("Migrate(stale backup) error = %v, want ErrMigrationBackup", err)
		}
		assertMigrationSourceUntouched(t, filename, original)
		assertBackupUntouched(t, backup, stale, 0o600)
		assertNoStagingLeak(t, directory)
	})

	t.Run("group readable backup", func(t *testing.T) {
		directory, filename, original := seedMigrationSource(t, Version1)
		backup := filename + ".bak." + Version1
		writeBackupFixture(t, backup, original, 0o644)

		_, err := Migrate(migrationInputs(directory, filename), nil, MigrationOptions{
			TargetVersion: CurrentVersion, GeneratedSummaryUpgradeChoice: "local_only",
		})
		if !errors.Is(err, ErrMigrationBackup) {
			t.Fatalf("Migrate(group-readable backup) error = %v, want ErrMigrationBackup", err)
		}
		assertMigrationSourceUntouched(t, filename, original)
		assertBackupUntouched(t, backup, original, 0o644)
		assertNoStagingLeak(t, directory)
	})

	t.Run("unreadable backup is not an absence", func(t *testing.T) {
		directory, filename, original := seedMigrationSource(t, Version1)
		backup := filename + ".bak." + Version1
		writeBackupFixture(t, backup, original, 0o600)
		filesystem := &faultMigrationFileSystem{failReadFile: backup}

		_, err := migrate(migrationInputs(directory, filename), nil, MigrationOptions{
			TargetVersion: CurrentVersion, GeneratedSummaryUpgradeChoice: "local_only",
		}, filesystem)
		if !errors.Is(err, ErrMigrationBackup) {
			t.Fatalf("Migrate(unreadable existing backup) error = %v, want ErrMigrationBackup", err)
		}
		assertMigrationSourceUntouched(t, filename, original)
		assertNoStagingLeak(t, directory)
	})

	// Positive control: an exact owner-only backup left by an interrupted run
	// is reused, so the gate admits the case it must admit.
	t.Run("exact owner-only backup is reused", func(t *testing.T) {
		directory, filename, original := seedMigrationSource(t, Version1)
		backup := filename + ".bak." + Version1
		writeBackupFixture(t, backup, original, 0o600)

		result, err := Migrate(migrationInputs(directory, filename), nil, MigrationOptions{
			TargetVersion: CurrentVersion, GeneratedSummaryUpgradeChoice: "local_only",
		})
		if err != nil || !result.Changed {
			t.Fatalf("Migrate(exact backup) = %#v, %v", result, err)
		}
		assertBackupUntouched(t, backup, original, 0o600)
		if loaded := loadMigrated(t, directory, filename); loaded.SourceVersion != CurrentVersion {
			t.Fatalf("migrated source version = %q", loaded.SourceVersion)
		}
		assertNoStagingLeak(t, directory)
	})
}

// TestMigrateRefusesSourceThatStoppedBeingARegularFileAfterLoad covers the
// time-of-check to time-of-use re-inspection performed after Load and before
// anything durable is written, so the refusal publishes neither a backup nor a
// staging file.
func TestMigrateRefusesSourceThatStoppedBeingARegularFileAfterLoad(t *testing.T) {
	t.Run("no longer regular", func(t *testing.T) {
		directory, filename, original := seedMigrationSource(t, Version1)
		filesystem := &faultMigrationFileSystem{statOverride: func(name string) (fs.FileInfo, error) {
			if name == filename {
				return migrationFixtureInfo{name: name, mode: fs.ModeSymlink | 0o600}, nil
			}
			return osMigrationFileSystem{}.Stat(name)
		}}
		_, err := migrate(migrationInputs(directory, filename), nil, MigrationOptions{
			TargetVersion: CurrentVersion, GeneratedSummaryUpgradeChoice: "local_only",
		}, filesystem)
		if !errors.Is(err, ErrConfigNotRegular) || !errors.Is(err, ErrMigrationWrite) {
			t.Fatalf("Migrate(source became a symlink) error = %v", err)
		}
		assertMigrationSourceUntouched(t, filename, original)
		assertNoMigrationArtifacts(t, directory, filename)
	})

	t.Run("source stat failed", func(t *testing.T) {
		directory, filename, original := seedMigrationSource(t, Version1)
		filesystem := &faultMigrationFileSystem{statOverride: func(name string) (fs.FileInfo, error) {
			if name == filename {
				return nil, fs.ErrPermission
			}
			return osMigrationFileSystem{}.Stat(name)
		}}
		_, err := migrate(migrationInputs(directory, filename), nil, MigrationOptions{
			TargetVersion: CurrentVersion, GeneratedSummaryUpgradeChoice: "local_only",
		}, filesystem)
		if !errors.Is(err, ErrMigrationWrite) || !errors.Is(err, fs.ErrPermission) {
			t.Fatalf("Migrate(source stat failure) error = %v", err)
		}
		assertMigrationSourceUntouched(t, filename, original)
		assertNoMigrationArtifacts(t, directory, filename)
	})
}

// TestMigrateLeavesTheSourceIntactAtEveryDurableFailurePoint injects a failure
// at each durable step and proves the selected file is never torn.
func TestMigrateLeavesTheSourceIntactAtEveryDurableFailurePoint(t *testing.T) {
	tests := []struct {
		name string
		// stagingSurvives marks the one injected fault whose own definition is
		// "the staging file could not be removed"; production cannot clean up
		// what Remove refuses to delete, so the leak is asserted to be
		// owner-only instead of absent.
		stagingSurvives bool
		filesystem      func() *faultMigrationFileSystem
		want            []error
	}{
		{
			name:       "write backup",
			filesystem: func() *faultMigrationFileSystem { return &faultMigrationFileSystem{failCreateTempCall: 1} },
			want:       []error{ErrMigrationBackup, fs.ErrPermission},
		},
		{
			name:       "publish backup",
			filesystem: func() *faultMigrationFileSystem { return &faultMigrationFileSystem{failLink: fs.ErrPermission} },
			want:       []error{ErrMigrationBackup, fs.ErrPermission},
		},
		{
			name:            "remove backup staging file",
			stagingSurvives: true,
			filesystem:      func() *faultMigrationFileSystem { return &faultMigrationFileSystem{failRemoveCall: 1} },
			want:            []error{ErrMigrationBackup, fs.ErrPermission},
		},
		{
			// A writer that reports zero progress without an error must not be
			// spun on: the staged file would never hold the complete document.
			name:       "short write",
			filesystem: func() *faultMigrationFileSystem { return &faultMigrationFileSystem{shortWrite: true} },
			want:       []error{ErrMigrationBackup, io.ErrShortWrite},
		},
		{
			// The temp-file half of "fsyncs it and the directory": the backup
			// staging file is fsynced before it is linked into place.
			name:       "sync backup file",
			filesystem: func() *faultMigrationFileSystem { return &faultMigrationFileSystem{failFileSyncCall: 1} },
			want:       []error{ErrMigrationBackup, fs.ErrInvalid},
		},
		{
			name:       "sync backup directory",
			filesystem: func() *faultMigrationFileSystem { return &faultMigrationFileSystem{failDirectorySyncCall: 1} },
			want:       []error{ErrMigrationSync},
		},
		{
			name:       "write replacement",
			filesystem: func() *faultMigrationFileSystem { return &faultMigrationFileSystem{failCreateTempCall: 2} },
			want:       []error{ErrMigrationWrite, fs.ErrPermission},
		},
		{
			// The same half on the replacement: file sync 1 is the backup
			// staging file, sync 2 is the replacement, and the replacement is
			// fsynced before it is renamed over the selected file.
			name:       "sync replacement file",
			filesystem: func() *faultMigrationFileSystem { return &faultMigrationFileSystem{failFileSyncCall: 2} },
			want:       []error{ErrMigrationWrite, fs.ErrInvalid},
		},
		{
			name:       "replace source",
			filesystem: func() *faultMigrationFileSystem { return &faultMigrationFileSystem{failRenameCall: 1} },
			want:       []error{ErrMigrationReplace, fs.ErrPermission},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			directory, filename, original := seedMigrationSource(t, Version1)
			_, err := migrate(migrationInputs(directory, filename), nil, MigrationOptions{
				TargetVersion: CurrentVersion, GeneratedSummaryUpgradeChoice: "local_only",
			}, test.filesystem())
			for _, want := range test.want {
				if !errors.Is(err, want) {
					t.Fatalf("Migrate(%s failure) error = %v, want %v", test.name, err, want)
				}
			}
			if errors.Is(err, ErrMigrationRecovery) {
				t.Fatalf("Migrate(%s failure) reported recovery failure: %v", test.name, err)
			}
			assertMigrationSourceUntouched(t, filename, original)
			if test.stagingSurvives {
				assertStagingFilesAreOwnerOnly(t, directory)
			} else {
				assertNoStagingLeak(t, directory)
			}

			// The failure is recoverable: a clean retry completes.
			result, retryErr := Migrate(migrationInputs(directory, filename), nil, MigrationOptions{
				TargetVersion: CurrentVersion, GeneratedSummaryUpgradeChoice: "local_only",
			})
			if retryErr != nil || !result.Changed {
				t.Fatalf("Migrate(retry after %s failure) = %#v, %v", test.name, result, retryErr)
			}
			if loaded := loadMigrated(t, directory, filename); loaded.SourceVersion != CurrentVersion {
				t.Fatalf("retry source version = %q", loaded.SourceVersion)
			}
		})
	}
}

// TestMigrateReportsRecoveryFailureWhenRollbackAlsoFails covers the branch
// where the post-replace sync failed and the rollback could not restore the
// source. The on-disk document must still be the complete replacement rather
// than a torn one, and the caller must be told recovery failed.
func TestMigrateReportsRecoveryFailureWhenRollbackAlsoFails(t *testing.T) {
	directory, filename, original := seedMigrationSource(t, Version1)
	// Directory sync 1 publishes the backup, sync 2 is the post-replace sync
	// that fails; create-temp 1 is the backup, 2 the replacement, 3 the
	// rollback that must also fail.
	filesystem := &faultMigrationFileSystem{failDirectorySyncCall: 2, failCreateTempCall: 3}

	_, err := migrate(migrationInputs(directory, filename), nil, MigrationOptions{
		TargetVersion: CurrentVersion, GeneratedSummaryUpgradeChoice: "local_only",
	}, filesystem)
	if !errors.Is(err, ErrMigrationRecovery) || !errors.Is(err, ErrMigrationSync) {
		t.Fatalf("Migrate(rollback failure) error = %v, want ErrMigrationRecovery and ErrMigrationSync", err)
	}
	onDisk, readErr := os.ReadFile(filename)
	if readErr != nil {
		t.Fatalf("read source after failed recovery: %v", readErr)
	}
	if bytes.Equal(onDisk, original) {
		t.Fatal("rollback reported failure but the source was restored")
	}
	// Never torn: what survived is a complete, loadable Configuration 3.0.0.
	if loaded := loadMigrated(t, directory, filename); loaded.SourceVersion != CurrentVersion {
		t.Fatalf("surviving document source version = %q", loaded.SourceVersion)
	}
	backup, backupErr := os.ReadFile(filename + ".bak." + Version1)
	if backupErr != nil || !bytes.Equal(backup, original) {
		t.Fatalf("backup after failed recovery: error=%v equal=%v", backupErr, bytes.Equal(backup, original))
	}
	assertNoStagingLeak(t, directory)
}

func seedMigrationSource(t *testing.T, version string) (directory, filename string, original []byte) {
	t.Helper()
	directory = t.TempDir()
	filename = filepath.Join(directory, "config.toml")
	original = minimalValidConfigVersion(scalar.PlatformMacOS, version)
	if err := os.WriteFile(filename, original, 0o600); err != nil {
		t.Fatal(err)
	}
	return directory, filename, original
}

func loadMigrated(t *testing.T, directory, filename string) LoadedConfiguration {
	t.Helper()
	snapshot, err := Load(migrationInputs(directory, filename), nil)
	if err != nil {
		t.Fatalf("Load(migrated document) error = %v", err)
	}
	loaded, ok := snapshot.Configuration()
	if !ok {
		t.Fatal("Load(migrated document) reported no configuration")
	}
	return loaded
}

func writeBackupFixture(t *testing.T, path string, content []byte, mode fs.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, content, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func assertMigrationSourceUntouched(t *testing.T, filename string, original []byte) {
	t.Helper()
	after, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("read source after refusal: %v", err)
	}
	if !bytes.Equal(after, original) {
		t.Fatalf("refused migration changed the source document")
	}
}

func assertBackupUntouched(t *testing.T, path string, content []byte, mode fs.FileMode) {
	t.Helper()
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read pre-existing backup: %v", err)
	}
	if !bytes.Equal(after, content) {
		t.Fatal("refused migration rewrote the pre-existing backup")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != mode.Perm() {
		t.Fatalf("pre-existing backup mode = %04o, want %04o", info.Mode().Perm(), mode.Perm())
	}
}

func assertNoMigrationArtifacts(t *testing.T, directory, filename string) {
	t.Helper()
	matches, err := filepath.Glob(filename + ".bak.*")
	if err != nil || len(matches) != 0 {
		t.Fatalf("refused migration produced backups %v (glob error %v)", matches, err)
	}
	assertNoStagingLeak(t, directory)
}

func assertNoStagingLeak(t *testing.T, directory string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(directory, ".ax-config-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("migration left staging files %v (glob error %v)", matches, err)
	}
}

func assertStagingFilesAreOwnerOnly(t *testing.T, directory string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(directory, ".ax-config-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatal("expected the unremovable staging file to survive its own injected fault")
	}
	for _, match := range matches {
		info, statErr := os.Stat(match)
		if statErr != nil {
			t.Fatal(statErr)
		}
		if info.Mode().Perm()&0o077 != 0 {
			t.Fatalf("staging file %s mode = %04o, want owner-only", match, info.Mode().Perm())
		}
	}
}

type migrationFixtureInfo struct {
	name string
	mode fs.FileMode
}

func (info migrationFixtureInfo) Name() string       { return info.name }
func (info migrationFixtureInfo) Size() int64        { return 0 }
func (info migrationFixtureInfo) Mode() fs.FileMode  { return info.mode }
func (info migrationFixtureInfo) ModTime() time.Time { return time.Time{} }
func (info migrationFixtureInfo) IsDir() bool        { return info.mode.IsDir() }
func (info migrationFixtureInfo) Sys() any           { return nil }
