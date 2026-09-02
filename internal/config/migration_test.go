package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

func TestMigrateProductionEntryCreatesOwnerOnlyBackupAndCompleteVersion2(t *testing.T) {
	directory := t.TempDir()
	filename := filepath.Join(directory, "config.toml")
	original := append(minimalValidConfigVersion(scalar.PlatformMacOS, Version1), []byte("\n[terminal]\nbackend = \"tmux\"\n")...)
	if err := os.WriteFile(filename, original, 0o640); err != nil {
		t.Fatal(err)
	}
	inputs := migrationInputs(directory, filename)

	result, err := Migrate(inputs, nil, MigrationOptions{
		TargetVersion: Version2, GeneratedSummaryUpgradeChoice: "reference_only",
	})
	if err != nil {
		t.Fatalf("Migrate(v1 to v2) error = %v", err)
	}
	if !result.Changed || result.SourceVersion != Version1 || result.TargetVersion != Version2 {
		t.Fatalf("Migrate(v1 to v2) result = %#v", result)
	}
	backup, err := os.ReadFile(result.BackupPath)
	if err != nil {
		t.Fatalf("read migration backup: %v", err)
	}
	if !bytes.Equal(backup, original) {
		t.Fatal("migration backup does not preserve exact v1 bytes")
	}
	backupInfo, err := os.Stat(result.BackupPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := backupInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("backup mode = %04o, want 0600", got)
	}
	// The replacement inherits the selected file's exact mode. Asserting
	// equality rather than a ceiling pins the migration in both directions: it
	// may neither widen the operator's mode nor silently tighten it.
	migratedInfo, err := os.Stat(filename)
	if err != nil {
		t.Fatal(err)
	}
	if got := migratedInfo.Mode().Perm(); got != 0o640 {
		t.Fatalf("migrated configuration mode = %04o, want the source mode 0640", got)
	}

	snapshot, err := Load(inputs, nil)
	if err != nil {
		t.Fatalf("Load(migrated v2) error = %v", err)
	}
	loaded, ok := snapshot.Configuration()
	if !ok || loaded.SourceVersion != Version2 {
		t.Fatalf("Load(migrated v2) = %#v, present=%v", loaded, ok)
	}
	if loaded.Value.Directory.GeneratedSummaryUpgradeChoice != "reference_only" {
		t.Fatalf("generated-summary choice = %q", loaded.Value.Directory.GeneratedSummaryUpgradeChoice)
	}
	if loaded.Value.Terminal.BackendID != "ax.tmux" {
		t.Fatalf("migrated terminal backend = %q", loaded.Value.Terminal.BackendID)
	}
}

func TestMigrateProductionEntryMapsLegacyTerminalToVersion3AndIsIdempotent(t *testing.T) {
	directory := t.TempDir()
	filename := filepath.Join(directory, "config.toml")
	original := append(minimalValidConfigVersion(scalar.PlatformMacOS, Version2), []byte("\n[terminal]\nbackend = \"tmux\"\n")...)
	if err := os.WriteFile(filename, original, 0o600); err != nil {
		t.Fatal(err)
	}
	inputs := migrationInputs(directory, filename)

	first, err := Migrate(inputs, nil, MigrationOptions{TargetVersion: CurrentVersion})
	if err != nil {
		t.Fatalf("Migrate(v2 to v3) error = %v", err)
	}
	migrated, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	migratedSnapshot, err := Load(inputs, nil)
	if err != nil {
		t.Fatalf("Load(migrated v3) error = %v", err)
	}
	migratedConfiguration, ok := migratedSnapshot.Configuration()
	if !ok || migratedConfiguration.SourceVersion != CurrentVersion || migratedConfiguration.Value.Terminal.BackendID != "ax.tmux" {
		t.Fatalf("migrated v3 configuration = %#v, present=%v", migratedConfiguration, ok)
	}
	second, err := Migrate(inputs, nil, MigrationOptions{TargetVersion: CurrentVersion})
	if err != nil {
		t.Fatalf("Migrate(already v3) error = %v", err)
	}
	if !first.Changed || second.Changed || second.BackupPath != "" {
		t.Fatalf("migration idempotency results = first %#v, second %#v", first, second)
	}
	afterSecond, _ := os.ReadFile(filename)
	if !bytes.Equal(afterSecond, migrated) {
		t.Fatal("idempotent migration rewrote current configuration")
	}
	backup, _ := os.ReadFile(first.BackupPath)
	if !bytes.Equal(backup, original) {
		t.Fatal("v2 backup changed after idempotent retry")
	}
}

func TestMigrateProductionEntryRefusesInvalidSourceBeforeBackupOrWrite(t *testing.T) {
	digest := "sha256:" + strings.Repeat("1", 64)
	duplicateBackend := string(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion)) + fmt.Sprintf(`
[[terminal.external_trust]]
backend_id = "com.example.term"
executable_path = "/usr/local/bin/term"
executable_digest = %q
enabled = false

[[terminal.external_trust]]
backend_id = "com.example.term"
executable_path = "/opt/bin/term"
executable_digest = %q
enabled = false
`, digest, digest)
	tests := map[string][]byte{
		"unknown closed root member":  append(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), []byte("unknown = true\n")...),
		"narrowed max parallel bound": configWithScalar("sync", "max_parallel_chunks", "33"),
		"duplicate backend rejection": []byte(duplicateBackend),
	}
	for name, document := range tests {
		name, document := name, document
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			filename := filepath.Join(directory, "config.toml")
			if err := os.WriteFile(filename, document, 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := Migrate(migrationInputs(directory, filename), nil, MigrationOptions{TargetVersion: CurrentVersion})
			if err == nil {
				t.Fatal("Migrate(invalid source) error = nil, want production Load refusal")
			}
			after, readErr := os.ReadFile(filename)
			if readErr != nil || !bytes.Equal(after, document) {
				t.Fatalf("invalid source changed: read=%v equal=%v", readErr, bytes.Equal(after, document))
			}
			matches, globErr := filepath.Glob(filename + ".bak.*")
			if globErr != nil || len(matches) != 0 {
				t.Fatalf("invalid source backups = %v, glob error=%v", matches, globErr)
			}
		})
	}
}

func TestMigrateProductionEntryRequiresExplicitChoiceAndRefusesDowngrade(t *testing.T) {
	t.Run("choice", func(t *testing.T) {
		directory := t.TempDir()
		filename := filepath.Join(directory, "config.toml")
		document := minimalValidConfigVersion(scalar.PlatformMacOS, Version1)
		if err := os.WriteFile(filename, document, 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := Migrate(migrationInputs(directory, filename), nil, MigrationOptions{TargetVersion: Version2})
		if !errors.Is(err, ErrMigrationChoiceRequired) {
			t.Fatalf("Migrate(v1 without choice) error = %v", err)
		}
	})
	t.Run("downgrade", func(t *testing.T) {
		directory := t.TempDir()
		filename := filepath.Join(directory, "config.toml")
		document := minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion)
		if err := os.WriteFile(filename, document, 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := Migrate(migrationInputs(directory, filename), nil, MigrationOptions{TargetVersion: Version2})
		if !errors.Is(err, ErrMigrationDowngrade) {
			t.Fatalf("Migrate(v3 to v2) error = %v", err)
		}
	})
}

func TestMigrateRecoversOriginalAfterPostReplaceDirectorySyncFailureAndRetries(t *testing.T) {
	directory := t.TempDir()
	filename := filepath.Join(directory, "config.toml")
	original := minimalValidConfigVersion(scalar.PlatformMacOS, Version1)
	if err := os.WriteFile(filename, original, 0o600); err != nil {
		t.Fatal(err)
	}
	filesystem := &faultMigrationFileSystem{failDirectorySyncCall: 2}
	options := MigrationOptions{TargetVersion: CurrentVersion, GeneratedSummaryUpgradeChoice: "local_only"}

	_, err := migrate(migrationInputs(directory, filename), nil, options, filesystem)
	if !errors.Is(err, ErrMigrationSync) || errors.Is(err, ErrMigrationRecovery) {
		t.Fatalf("Migrate(post-replace sync failure) error = %v", err)
	}
	afterFailure, _ := os.ReadFile(filename)
	if !bytes.Equal(afterFailure, original) {
		t.Fatal("post-replace sync failure did not restore exact source bytes")
	}
	backup, backupErr := os.ReadFile(filename + ".bak." + Version1)
	if backupErr != nil || !bytes.Equal(backup, original) {
		t.Fatalf("recovery backup: error=%v equal=%v", backupErr, bytes.Equal(backup, original))
	}

	filesystem.failDirectorySyncCall = 0
	result, err := migrate(migrationInputs(directory, filename), nil, options, filesystem)
	if err != nil || !result.Changed {
		t.Fatalf("Migrate(retry after recovered failure) = %#v, %v", result, err)
	}
	snapshot, err := Load(migrationInputs(directory, filename), nil)
	if err != nil {
		t.Fatalf("Load(retry result) error = %v", err)
	}
	loaded, _ := snapshot.Configuration()
	if loaded.SourceVersion != CurrentVersion {
		t.Fatalf("retry source version = %q", loaded.SourceVersion)
	}
}

func TestAssessCompatibilityEnforcesReadOnlyDowngradeWithoutDecodingOrMutation(t *testing.T) {
	document := append(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), []byte("\n[directory]\ngrep_result_max = 10000\n")...)
	original := append([]byte(nil), document...)
	assessment, err := AssessCompatibility(document, Version1)
	if err != nil {
		t.Fatalf("AssessCompatibility(v3 for v1 reader) error = %v", err)
	}
	if assessment.Mode != CompatibilityReadOnly || assessment.SourceVersion != CurrentVersion || assessment.ReaderVersion != Version1 {
		t.Fatalf("downgrade assessment = %#v", assessment)
	}
	if !bytes.Equal(document, original) {
		t.Fatal("read-only compatibility assessment mutated source bytes")
	}

	readWrite, err := AssessCompatibility(minimalValidConfigVersion(scalar.PlatformMacOS, Version1), CurrentVersion)
	if err != nil || readWrite.Mode != CompatibilityCompatible {
		t.Fatalf("older supported assessment = %#v, %v", readWrite, err)
	}
}

func TestAssessCompatibilityRefusesMalformedEnvelopeFacts(t *testing.T) {
	tests := []struct {
		name, document, reader string
		want                   error
	}{
		{name: "unknown reader", document: string(minimalValidConfigVersion(scalar.PlatformMacOS, Version1)), reader: "0.0.0", want: ErrCompatibilityReader},
		{name: "wrong schema", document: strings.Replace(string(minimalValidConfigVersion(scalar.PlatformMacOS, Version1)), SchemaID, "urn:example:other", 1), reader: Version1, want: ErrConfigValidation},
		{name: "malformed source version", document: strings.Replace(string(minimalValidConfigVersion(scalar.PlatformMacOS, Version1)), Version1, "v1", 1), reader: Version1, want: ErrCompatibilityAssessment},
		{name: "unknown older version", document: strings.Replace(string(minimalValidConfigVersion(scalar.PlatformMacOS, Version2)), Version2, "2.1.0", 1), reader: CurrentVersion, want: ErrUnsupportedConfigVersion},
		{name: "malformed TOML", document: "schema = [", reader: Version1, want: ErrConfigDecode},
		// A newer document takes the read-only branch, which never consults the
		// known-version set, so the canonical-core check is the only thing
		// standing between "01.0.0" and a successful assessment.
		{name: "non-canonical newer version", document: strings.Replace(string(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion)), CurrentVersion, "04.0.0", 1), reader: CurrentVersion, want: ErrCompatibilityAssessment},
		{name: "non-canonical older version", document: strings.Replace(string(minimalValidConfigVersion(scalar.PlatformMacOS, Version1)), Version1, "01.0.0", 1), reader: CurrentVersion, want: ErrCompatibilityAssessment},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := AssessCompatibility([]byte(test.document), test.reader)
			if !errors.Is(err, test.want) {
				t.Fatalf("AssessCompatibility() error = %v, want %v", err, test.want)
			}
		})
	}
}

// TestMigrateFsyncsEveryStagedFileBeforeItBecomesVisible pins the file half of
// the durability contract in the widening direction. SPEC 6.4 requires that
// migration "writes a complete v2 file to a same-directory temporary file,
// fsyncs it and the directory, and atomically replaces the original", so every
// file this migration stages must be fsynced, not just the last one. Narrowing
// the production entry to fsync one of the two staged files leaves the counts
// unequal; deleting the fsync outright is caught by the paired injected
// failures in TestMigrateLeavesTheSourceIntactAtEveryDurableFailurePoint.
func TestMigrateFsyncsEveryStagedFileBeforeItBecomesVisible(t *testing.T) {
	directory, filename, _ := seedMigrationSource(t, Version1)
	filesystem := &faultMigrationFileSystem{}

	result, err := migrate(migrationInputs(directory, filename), nil, MigrationOptions{
		TargetVersion: CurrentVersion, GeneratedSummaryUpgradeChoice: "local_only",
	}, filesystem)
	if err != nil || !result.Changed {
		t.Fatalf("Migrate(v1 to %s) = %#v, %v", CurrentVersion, result, err)
	}
	if filesystem.createTempCalls != 2 {
		t.Fatalf("staged files = %d, want 2 (backup, replacement)", filesystem.createTempCalls)
	}
	if filesystem.fileSyncCalls != filesystem.createTempCalls {
		t.Fatalf("file fsyncs = %d for %d staged files; every staged file must be fsynced before it becomes visible",
			filesystem.fileSyncCalls, filesystem.createTempCalls)
	}
}

// TestAssessCompatibilityPinsEveryPinnedVersionPairAtTheProductionEntry drives
// the full cross-product of pinned Configuration versions through the real
// AssessCompatibility entry. The expected mode is derived from the pinned
// catalog's declared major versions rather than from the production comparison,
// so the case SPEC 6.5 names literally - "A v1/v2 binary opening v3 is
// read-only diagnostic" - is asserted at both the one-major and two-major step,
// and every same-or-older pair is asserted compatible in the other direction.
func TestAssessCompatibilityPinsEveryPinnedVersionPairAtTheProductionEntry(t *testing.T) {
	versions := pinnedConfigurationVersions(t)
	if len(versions) < 2 {
		t.Fatalf("pinned catalog declares %d Configuration versions, need at least two to bound a downgrade", len(versions))
	}
	readOnlyPairs, compatiblePairs := 0, 0
	for _, source := range versions {
		for _, reader := range versions {
			source, reader := source, reader
			t.Run("document_"+source+"_reader_"+reader, func(t *testing.T) {
				want := CompatibilityCompatible
				if majorVersionForTest(t, source) > majorVersionForTest(t, reader) {
					want = CompatibilityReadOnly
				}
				document := minimalValidConfigVersion(scalar.PlatformMacOS, source)
				original := append([]byte(nil), document...)
				assessment, err := AssessCompatibility(document, reader)
				if err != nil {
					t.Fatalf("AssessCompatibility(%s document, %s reader) error = %v", source, reader, err)
				}
				if assessment.Mode != want {
					t.Fatalf("AssessCompatibility(%s document, %s reader).Mode = %q, want %q", source, reader, assessment.Mode, want)
				}
				if assessment.SourceVersion != source || assessment.ReaderVersion != reader {
					t.Fatalf("assessment envelope facts = %#v", assessment)
				}
				if !bytes.Equal(document, original) {
					t.Fatal("compatibility assessment mutated the source bytes")
				}
			})
			if majorVersionForTest(t, source) > majorVersionForTest(t, reader) {
				readOnlyPairs++
			} else {
				compatiblePairs++
			}
		}
	}
	// Both branches must be exercised, so a gate that answered a single mode
	// for every pair cannot pass by covering only one side.
	if readOnlyPairs == 0 || compatiblePairs == 0 {
		t.Fatalf("cross-product exercised %d read-only and %d compatible pairs; both directions are required", readOnlyPairs, compatiblePairs)
	}
}

// majorVersionForTest parses a major version without reusing the production
// comparison the cross-product is meant to bound.
func majorVersionForTest(t *testing.T, version string) uint64 {
	t.Helper()
	major, _, found := strings.Cut(version, ".")
	if !found {
		t.Fatalf("pinned version %q is not dotted", version)
	}
	parsed, err := strconv.ParseUint(major, 10, 64)
	if err != nil {
		t.Fatalf("pinned version %q has no numeric major: %v", version, err)
	}
	return parsed
}

func migrationInputs(directory, filename string) Inputs {
	return Inputs{
		Platform: scalar.PlatformMacOS, HomeDir: directory, TempDir: directory, WorkingDir: directory,
		LookupEnv: func(name string) (string, bool) {
			if name == "AX_CONFIG" {
				return filename, true
			}
			return "", false
		},
		Stat: os.Stat, ReadFile: os.ReadFile,
	}
}

type faultMigrationFileSystem struct {
	osMigrationFileSystem
	directorySyncCalls    int
	failDirectorySyncCall int
	createTempCalls       int
	failCreateTempCall    int
	removeCalls           int
	failRemoveCall        int
	renameCalls           int
	failRenameCall        int
	fileSyncCalls         int
	failFileSyncCall      int
	shortWrite            bool
	failLink              error
	failReadFile          string
	statOverride          func(string) (fs.FileInfo, error)
}

func (filesystem *faultMigrationFileSystem) ReadFile(name string) ([]byte, error) {
	if filesystem.failReadFile != "" && filesystem.failReadFile == name {
		return nil, fs.ErrPermission
	}
	return filesystem.osMigrationFileSystem.ReadFile(name)
}

func (filesystem *faultMigrationFileSystem) CreateTemp(dir, pattern string) (migrationFile, error) {
	filesystem.createTempCalls++
	if filesystem.createTempCalls == filesystem.failCreateTempCall {
		return nil, fs.ErrPermission
	}
	file, err := filesystem.osMigrationFileSystem.CreateTemp(dir, pattern)
	if err != nil {
		return nil, err
	}
	return &faultMigrationFile{migrationFile: file, filesystem: filesystem}, nil
}

// faultMigrationFile counts and optionally fails the per-file fsync, the half
// of the durability contract that the directory-sync seam cannot observe.
type faultMigrationFile struct {
	migrationFile
	filesystem *faultMigrationFileSystem
}

func (file *faultMigrationFile) Write(document []byte) (int, error) {
	if file.filesystem.shortWrite {
		return 0, nil
	}
	return file.migrationFile.Write(document)
}

func (file *faultMigrationFile) Sync() error {
	file.filesystem.fileSyncCalls++
	if file.filesystem.fileSyncCalls == file.filesystem.failFileSyncCall {
		return fs.ErrInvalid
	}
	return file.migrationFile.Sync()
}

func (filesystem *faultMigrationFileSystem) Link(oldname, newname string) error {
	if filesystem.failLink != nil {
		return filesystem.failLink
	}
	return filesystem.osMigrationFileSystem.Link(oldname, newname)
}

func (filesystem *faultMigrationFileSystem) Remove(name string) error {
	filesystem.removeCalls++
	if filesystem.removeCalls == filesystem.failRemoveCall {
		return fs.ErrPermission
	}
	return filesystem.osMigrationFileSystem.Remove(name)
}

func (filesystem *faultMigrationFileSystem) Rename(oldname, newname string) error {
	filesystem.renameCalls++
	if filesystem.renameCalls == filesystem.failRenameCall {
		return fs.ErrPermission
	}
	return filesystem.osMigrationFileSystem.Rename(oldname, newname)
}

func (filesystem *faultMigrationFileSystem) Stat(name string) (fs.FileInfo, error) {
	if filesystem.statOverride != nil {
		return filesystem.statOverride(name)
	}
	return filesystem.osMigrationFileSystem.Stat(name)
}

func (filesystem *faultMigrationFileSystem) OpenDirectory(name string) (migrationDirectory, error) {
	directory, err := filesystem.osMigrationFileSystem.OpenDirectory(name)
	if err != nil {
		return nil, err
	}
	return &faultMigrationDirectory{migrationDirectory: directory, filesystem: filesystem}, nil
}

type faultMigrationDirectory struct {
	migrationDirectory
	filesystem *faultMigrationFileSystem
}

func (directory *faultMigrationDirectory) Sync() error {
	directory.filesystem.directorySyncCalls++
	if directory.filesystem.directorySyncCalls == directory.filesystem.failDirectorySyncCall {
		return fs.ErrInvalid
	}
	return directory.migrationDirectory.Sync()
}
