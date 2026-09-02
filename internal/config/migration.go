package config

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

var (
	ErrMigrationTarget         = errors.New("unsupported configuration migration target")
	ErrMigrationDowngrade      = errors.New("configuration migration would downgrade")
	ErrMigrationChoiceRequired = errors.New("generated-summary disclosure choice required")
	ErrMigrationSourceAbsent   = errors.New("configuration migration source is absent")
	ErrMigrationBackup         = errors.New("configuration migration backup failed")
	ErrMigrationWrite          = errors.New("configuration migration write failed")
	ErrMigrationSync           = errors.New("configuration migration sync failed")
	ErrMigrationReplace        = errors.New("configuration migration atomic replace failed")
	ErrMigrationRecovery       = errors.New("configuration migration recovery failed")
	ErrCompatibilityReader     = errors.New("unsupported configuration reader version")
	ErrCompatibilityAssessment = errors.New("configuration compatibility assessment failed")
)

// MigrationOptions are the explicit choices supplied by the ax migrate config
// owner. Major-version migration is never invoked from Load or startup.
type MigrationOptions struct {
	TargetVersion                 string
	GeneratedSummaryUpgradeChoice string
}

// MigrationResult reports durable migration without claiming CLI or doctor
// availability. Changed is false for an already-target-version document.
// BackupPath is the deterministic name the backup takes and is only guaranteed
// to exist on disk when Migrate returned a nil error; a caller must not treat
// it as a readable path after a refusal.
type MigrationResult struct {
	SourceVersion string
	TargetVersion string
	BackupPath    string
	Changed       bool
}

// MigrationError renders no machine-local path or document content.
type MigrationError struct {
	Operation string
	Err       error
}

func (err *MigrationError) Error() string {
	return "configuration migration " + err.Operation + " failed"
}
func (err *MigrationError) Unwrap() error { return err.Err }

// MigrateOS captures real process inputs and performs one explicit durable
// major-version migration. The selected source is validated through Load
// before any backup or replacement is attempted.
func MigrateOS(platform scalar.Platform, overrides Overrides, options MigrationOptions) (MigrationResult, error) {
	inputs, err := OSInputs(platform)
	if err != nil {
		return MigrationResult{}, err
	}
	return Migrate(inputs, overrides, options)
}

// Migrate is the dependency-injected production entry point. It writes only
// the selected configuration file and a deterministic source-version backup
// in the same directory.
func Migrate(inputs Inputs, overrides Overrides, options MigrationOptions) (MigrationResult, error) {
	return migrate(inputs, overrides, options, osMigrationFileSystem{})
}

// CompatibilityMode is deliberately narrower than runtime capability state.
type CompatibilityMode string

const (
	CompatibilityCompatible CompatibilityMode = "compatible"
	CompatibilityReadOnly   CompatibilityMode = "read-only-diagnostic"
)

// CompatibilityAssessment contains only envelope facts needed to prevent a
// downgraded writer. It exposes no decoded Configuration and performs no I/O.
type CompatibilityAssessment struct {
	ReaderVersion string
	SourceVersion string
	Mode          CompatibilityMode
}

// AssessCompatibility determines whether a known reader may open the document
// normally or must remain read-only. Newer documents are not decoded or
// rewritten, so fields unknown to the older reader remain untouched.
func AssessCompatibility(document []byte, readerVersion string) (CompatibilityAssessment, error) {
	reader, ok := knownConfigVersion(readerVersion)
	if !ok {
		return CompatibilityAssessment{}, migrationError(MigrationError{Operation: "assess reader", Err: ErrCompatibilityReader})
	}
	var envelope rawEnvelope
	if err := decodeEnvelope(document, &envelope); err != nil {
		return CompatibilityAssessment{}, err
	}
	if envelope.Schema != SchemaID {
		return CompatibilityAssessment{}, configError("schema", ErrConfigValidation)
	}
	source, err := parseSemverCore(envelope.SchemaVersion)
	if err != nil {
		return CompatibilityAssessment{}, configError("schema_version", errors.Join(ErrCompatibilityAssessment, err))
	}
	mode := CompatibilityCompatible
	if compareSemver(source, reader) > 0 {
		mode = CompatibilityReadOnly
	} else if _, known := knownConfigVersion(envelope.SchemaVersion); !known {
		return CompatibilityAssessment{}, configError("schema_version", ErrUnsupportedConfigVersion)
	}
	return CompatibilityAssessment{ReaderVersion: readerVersion, SourceVersion: envelope.SchemaVersion, Mode: mode}, nil
}

type semverCore struct{ major, minor, patch uint64 }

func knownConfigVersion(value string) (semverCore, bool) {
	switch value {
	case Version1, Version2, CurrentVersion:
		parsed, _ := parseSemverCore(value)
		return parsed, true
	default:
		return semverCore{}, false
	}
}

func parseSemverCore(value string) (semverCore, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return semverCore{}, ErrUnsupportedConfigVersion
	}
	values := make([]uint64, 3)
	for index, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return semverCore{}, ErrUnsupportedConfigVersion
		}
		parsed, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return semverCore{}, ErrUnsupportedConfigVersion
		}
		values[index] = parsed
	}
	return semverCore{major: values[0], minor: values[1], patch: values[2]}, nil
}

func compareSemver(left, right semverCore) int {
	if left.major != right.major {
		if left.major < right.major {
			return -1
		}
		return 1
	}
	if left.minor != right.minor {
		if left.minor < right.minor {
			return -1
		}
		return 1
	}
	if left.patch != right.patch {
		if left.patch < right.patch {
			return -1
		}
		return 1
	}
	return 0
}

func decodeEnvelope(document []byte, envelope *rawEnvelope) error {
	if err := tomlDecode(document, envelope); err != nil {
		return configError("TOML", errors.Join(ErrConfigDecode, err))
	}
	return nil
}

func migrate(inputs Inputs, overrides Overrides, options MigrationOptions, filesystem migrationFileSystem) (MigrationResult, error) {
	target, targetKnown := knownConfigVersion(options.TargetVersion)
	if !targetKnown || options.TargetVersion == Version1 {
		return MigrationResult{}, migrationError(MigrationError{Operation: "select target", Err: ErrMigrationTarget})
	}
	snapshot, err := Load(inputs, overrides)
	if err != nil {
		return MigrationResult{}, err
	}
	loaded, decoded := snapshot.Configuration()
	if !snapshot.ConfigPresent() || !decoded {
		return MigrationResult{}, migrationError(MigrationError{Operation: "load source", Err: ErrMigrationSourceAbsent})
	}
	source, _ := knownConfigVersion(loaded.SourceVersion)
	result := MigrationResult{SourceVersion: loaded.SourceVersion, TargetVersion: options.TargetVersion}
	if compareSemver(source, target) > 0 {
		return result, migrationError(MigrationError{Operation: "select target", Err: ErrMigrationDowngrade})
	}
	if compareSemver(source, target) == 0 {
		return result, nil
	}
	if loaded.SourceVersion == Version1 {
		if !oneOf(options.GeneratedSummaryUpgradeChoice, "local_only", "mesh_sanitized", "reference_only") {
			return result, migrationError(MigrationError{Operation: "validate disclosure choice", Err: ErrMigrationChoiceRequired})
		}
		loaded.Value.Directory.GeneratedSummaryUpgradeChoice = options.GeneratedSummaryUpgradeChoice
	}
	context := DecodeContext{RuntimePlatform: inputs.Platform, BackendSettings: inputs.BackendSettings}
	var replacement []byte
	switch options.TargetVersion {
	case Version2:
		replacement, err = encodeVersion2(loaded.Value, context)
	case CurrentVersion:
		replacement, err = EncodeCurrent(loaded.Value, context)
	}
	if err != nil {
		return result, migrationError(MigrationError{Operation: "encode target", Err: err}) // config-refusal-subsumed: every encoder refusal below this propagation is pinned on its own clause, at writer.go terminal.backend and writer.go v2 wire TOML for the v2 encoder, at writer.go v2 re-read for its defence-in-depth round trip, and by the EncodeCurrent validation suite for v3
	}
	selected, _ := snapshot.Paths().Path(ConfigFile)
	filename := selected.Value.String()
	backup := filename + ".bak." + loaded.SourceVersion
	result.BackupPath = backup
	if err := replaceDurably(filesystem, filename, backup, snapshot.Document(), replacement); err != nil {
		return result, err
	}
	result.Changed = true
	return result, nil
}

type migrationFile interface {
	io.Writer
	Name() string
	Chmod(fs.FileMode) error
	Sync() error
	Close() error
}

type migrationDirectory interface {
	Sync() error
	Close() error
}

type migrationFileSystem interface {
	CreateTemp(string, string) (migrationFile, error)
	Link(string, string) error
	ReadFile(string) ([]byte, error)
	Stat(string) (fs.FileInfo, error)
	Remove(string) error
	Rename(string, string) error
	OpenDirectory(string) (migrationDirectory, error)
}

type osMigrationFileSystem struct{}

func (osMigrationFileSystem) CreateTemp(dir, pattern string) (migrationFile, error) {
	return os.CreateTemp(dir, pattern)
}
func (osMigrationFileSystem) Link(oldname, newname string) error   { return os.Link(oldname, newname) }
func (osMigrationFileSystem) ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }

// Stat does not follow a symlink, unlike the symlink-following read-side seam
// in OSInputs. Durable migration replaces the selected file with an atomic
// rename, which replaces a symlink itself rather than its target: migrating a
// symlinked configuration would silently detach the operator's link and leave
// the real document holding stale bytes next to a backup of the resolved
// target. The specification is silent here, so the mutating path fails closed
// with ErrConfigNotRegular instead of guessing which file the operator meant.
func (osMigrationFileSystem) Stat(name string) (fs.FileInfo, error) { return os.Lstat(name) }
func (osMigrationFileSystem) Remove(name string) error              { return os.Remove(name) }
func (osMigrationFileSystem) Rename(oldname, newname string) error {
	return os.Rename(oldname, newname)
}
func (osMigrationFileSystem) OpenDirectory(name string) (migrationDirectory, error) {
	return os.Open(name)
}

func replaceDurably(filesystem migrationFileSystem, filename, backup string, original, replacement []byte) error {
	directory := filepath.Dir(filename)
	// Re-inspect the selected file before anything durable is written. Load
	// resolved and validated it through the symlink-following read seam; this
	// mutating seam does not follow symlinks, so a selected file that is not
	// itself a regular file is refused here rather than after a backup has
	// already been published. A refused migration therefore leaves the
	// directory exactly as it found it, and the captured permission bits are
	// the ones the replacement inherits.
	info, err := filesystem.Stat(filename)
	if err != nil {
		return migrationError(MigrationError{Operation: "inspect source mode", Err: errors.Join(ErrMigrationWrite, err)})
	}
	if !info.Mode().IsRegular() {
		return migrationError(MigrationError{Operation: "inspect source mode", Err: errors.Join(ErrMigrationWrite, ErrConfigNotRegular)})
	}
	backupTemp, err := writeTempFile(filesystem, directory, ".ax-config-backup-*", original, 0o600)
	if err != nil {
		return migrationError(MigrationError{Operation: "write backup", Err: errors.Join(ErrMigrationBackup, err)})
	}
	if err := filesystem.Link(backupTemp, backup); err != nil {
		if !errors.Is(err, fs.ErrExist) {
			_ = filesystem.Remove(backupTemp)
			return migrationError(MigrationError{Operation: "publish backup", Err: errors.Join(ErrMigrationBackup, err)})
		}
		existing, readErr := filesystem.ReadFile(backup)
		info, statErr := filesystem.Stat(backup)
		if readErr != nil || statErr != nil || !bytes.Equal(existing, original) || info.Mode().Perm()&0o077 != 0 {
			_ = filesystem.Remove(backupTemp)
			return migrationError(MigrationError{Operation: "verify backup", Err: ErrMigrationBackup})
		}
	}
	if err := filesystem.Remove(backupTemp); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return migrationError(MigrationError{Operation: "remove backup staging file", Err: errors.Join(ErrMigrationBackup, err)})
	}
	// Retry the directory sync even when an exact backup already exists: it may
	// be the visible result of an interrupted run whose directory fsync failed.
	if err := syncDirectory(filesystem, directory); err != nil {
		return migrationError(MigrationError{Operation: "sync backup directory", Err: errors.Join(ErrMigrationSync, err)})
	}

	replacementTemp, err := writeTempFile(filesystem, directory, ".ax-config-migrate-*", replacement, info.Mode().Perm())
	if err != nil {
		return migrationError(MigrationError{Operation: "write replacement", Err: errors.Join(ErrMigrationWrite, err)})
	}
	if err := filesystem.Rename(replacementTemp, filename); err != nil {
		_ = filesystem.Remove(replacementTemp)
		return migrationError(MigrationError{Operation: "replace source", Err: errors.Join(ErrMigrationReplace, err)})
	}
	if err := syncDirectory(filesystem, directory); err == nil {
		return nil
	} else {
		syncErr := err
		rollbackTemp, rollbackErr := writeTempFile(filesystem, directory, ".ax-config-rollback-*", original, info.Mode().Perm())
		if rollbackErr == nil {
			rollbackErr = filesystem.Rename(rollbackTemp, filename)
		}
		if rollbackErr == nil {
			rollbackErr = syncDirectory(filesystem, directory)
		}
		if rollbackErr != nil {
			if rollbackTemp != "" {
				_ = filesystem.Remove(rollbackTemp)
			}
			return migrationError(MigrationError{Operation: "recover source after sync", Err: errors.Join(ErrMigrationSync, ErrMigrationRecovery, syncErr, rollbackErr)})
		}
		return migrationError(MigrationError{Operation: "sync replacement directory", Err: errors.Join(ErrMigrationSync, syncErr)})
	}
}

func writeTempFile(filesystem migrationFileSystem, directory, pattern string, document []byte, mode fs.FileMode) (name string, err error) {
	file, err := filesystem.CreateTemp(directory, pattern)
	if err != nil {
		return "", err
	}
	name = file.Name()
	clean := func(cause error) (string, error) {
		_ = file.Close()
		_ = filesystem.Remove(name)
		return "", cause
	}
	if err := file.Chmod(mode); err != nil {
		return clean(err)
	}
	if err := writeAll(file, document); err != nil {
		return clean(err)
	}
	if err := file.Sync(); err != nil {
		return clean(err)
	}
	if err := file.Close(); err != nil {
		_ = filesystem.Remove(name)
		return "", err
	}
	return name, nil
}

func writeAll(writer io.Writer, document []byte) error {
	for len(document) > 0 {
		written, err := writer.Write(document)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		document = document[written:]
	}
	return nil
}

func syncDirectory(filesystem migrationFileSystem, directory string) error {
	handle, err := filesystem.OpenDirectory(directory)
	if err != nil {
		return err
	}
	if err := handle.Sync(); err != nil {
		_ = handle.Close()
		return err
	}
	return handle.Close()
}

// Indirections keep compatibility-envelope parsing in this file while using
// the same TOML implementation as Decode.
var tomlDecode = func(document []byte, destination any) error {
	return toml.Unmarshal(document, destination)
}

// migrationError is the only construction site for *MigrationError, for the
// same reason loaderError is the only construction site for *Error.
var migrationError = func(value MigrationError) error {
	return &value
}
