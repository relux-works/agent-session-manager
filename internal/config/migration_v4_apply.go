package config

import (
	"bytes"
	"errors"
	"time"

	"github.com/relux-works/agent-session-manager/internal/hosttrust"
	"github.com/relux-works/agent-session-manager/internal/localstore"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// ApplyV4OS captures real process inputs and performs one explicit durable
// Configuration 4.0.0 migration. The caller supplies the exact confirmed
// preview from PreviewV4; confirm must be true.
func ApplyV4OS(platform scalar.Platform, overrides Overrides, preview V4Preview, confirm bool, now time.Time) (V4ApplyResult, error) {
	inputs, err := OSInputs(platform)
	if err != nil {
		return V4ApplyResult{}, err
	}
	return applyV4(inputs, overrides, preview, confirm, now, osMigrationFileSystem{}, openStoreForConfig)
}

// ApplyV4 is the dependency-injected production entry point. It writes only
// the selected configuration file, a deterministic source-version backup in
// the same directory, and one trust generation bump under the shared
// authorization lock.
func ApplyV4(inputs Inputs, overrides Overrides, preview V4Preview, confirm bool, now time.Time) (V4ApplyResult, error) {
	return applyV4(inputs, overrides, preview, confirm, now, osMigrationFileSystem{}, openStoreForConfig)
}

type storeOpener func(localstore.ResolvedPaths) (*hosttrust.Store, error)

func openStoreForConfig(paths localstore.ResolvedPaths) (*hosttrust.Store, error) {
	return hosttrust.Open(paths)
}

func applyV4(inputs Inputs, overrides Overrides, preview V4Preview, confirm bool, now time.Time,
	filesystem migrationFileSystem, openStore storeOpener) (V4ApplyResult, error) {
	if !confirm {
		return V4ApplyResult{}, migrationError(MigrationError{Operation: "confirm preview", Err: ErrMigrationV4ConfirmRequired})
	}
	if preview.SourceVersion != CurrentVersion || len(preview.SourceDocument) == 0 || len(preview.Replacement) == 0 {
		return V4ApplyResult{}, migrationError(MigrationError{Operation: "validate preview", Err: ErrMigrationV4PreviewMismatch})
	}
	// Unlocked fast-fail checks first; the same pins are revalidated inside
	// the joint transaction below, closing the concurrent-change window
	// between this read and the commit.
	snapshot, err := Load(inputs, overrides)
	if err != nil {
		return V4ApplyResult{}, err
	}
	loaded, decoded := snapshot.Configuration()
	if !snapshot.ConfigPresent() || !decoded {
		return V4ApplyResult{}, migrationError(MigrationError{Operation: "load source", Err: ErrMigrationSourceAbsent})
	}
	if loaded.SourceVersion != CurrentVersion {
		return V4ApplyResult{}, migrationError(MigrationError{Operation: "select source", Err: ErrMigrationV4Source})
	}
	if !bytes.Equal(snapshot.Document(), preview.SourceDocument) {
		return V4ApplyResult{}, migrationError(MigrationError{Operation: "validate preview source", Err: ErrMigrationV4StaleSource})
	}
	localPaths := snapshot.LocalPaths()
	filename, err := configPathFromLocalPaths(localPaths)
	if err != nil {
		return V4ApplyResult{}, err
	}
	store, err := openStore(localPaths)
	if err != nil {
		return V4ApplyResult{}, err
	}
	if err := store.EnsureConfigBinding(localPaths); err != nil {
		return V4ApplyResult{}, err
	}
	current, err := store.ReadSnapshot()
	if err != nil {
		return V4ApplyResult{}, err
	}
	if current.Generation != preview.SourceGeneration {
		return V4ApplyResult{}, migrationError(MigrationError{Operation: "validate preview generation", Err: ErrMigrationV4StaleGeneration})
	}
	rendered, err := renderV4Preview(snapshot.Document(), loaded.Value, current, preview.CredentialID, preview.DroppedPeers, now, inputs)
	if err != nil {
		return V4ApplyResult{}, err
	}
	if !bytes.Equal(rendered.Replacement, preview.Replacement) {
		return V4ApplyResult{}, migrationError(MigrationError{Operation: "validate preview bytes", Err: ErrMigrationV4PreviewMismatch})
	}
	// The custody files behind the selected credential must validate before
	// use, exactly as Section 11.10.2 requires for local identity.
	credentialHex := previewCredentialHex(preview.CredentialID)
	if err := store.ValidateCustody(credentialHex, loaded.Value.HostID, now); err != nil {
		return V4ApplyResult{}, err
	}
	backup := filename + ".bak." + loaded.SourceVersion
	// filename is the selected snapshot path, a scalar.AbsolutePath by
	// loader construction; JointCommit still revalidates the absolute
	// marker path before staging anything.
	marker := hosttrust.PendingCommit{
		Schema:            "urn:ax:schema:host-pending-commit",
		SchemaVersion:     "1.0.0",
		Operation:         hosttrust.JointApplyV4,
		ConfigPath:        filename,
		SourceSHA256:      hosttrust.SHA256HexOf(preview.SourceDocument),
		ReplacementSHA256: hosttrust.SHA256HexOf(preview.Replacement),
		SourceGeneration:  preview.SourceGeneration,
		BackupPath:        backup,
	}
	generation, err := store.JointCommit(localPaths, marker,
		func(current hosttrust.TrustStore) error {
			return revalidateApplyV4(inputs, overrides, store, preview, current, now)
		},
		func(hold hosttrust.HeldExclusive) error {
			return replaceDurably(hold, filesystem, localPaths, backup, preview.SourceDocument, preview.Replacement)
		},
		func(hold hosttrust.HeldExclusive) error {
			_, err := writeTempReplace(hold, filesystem, localPaths, preview.SourceDocument)
			return err
		})
	if err != nil {
		if errors.Is(err, hosttrust.ErrJointCompensationFailed) {
			return V4ApplyResult{}, migrationError(MigrationError{Operation: "compensate joint commit", Err: errors.Join(ErrMigrationSync, ErrMigrationRecovery, err)})
		}
		return V4ApplyResult{}, err
	}
	return V4ApplyResult{
		Migration:  MigrationResult{SourceVersion: loaded.SourceVersion, TargetVersion: Version4, BackupPath: backup, Changed: true},
		Generation: generation,
	}, nil
}

// revalidateApplyV4 repeats the confirmed pins inside the joint transaction:
// current source bytes, source version, preview rendering, generation and
// credential custody. It runs under the exclusive authorization lock, so a
// concurrent trust or config commit cannot slip between validation and the
// replacement.
func revalidateApplyV4(inputs Inputs, overrides Overrides, store *hosttrust.Store, preview V4Preview, current hosttrust.TrustStore, now time.Time) error {
	snapshot, err := Load(inputs, overrides)
	if err != nil {
		return err
	}
	loaded, decoded := snapshot.Configuration()
	if !snapshot.ConfigPresent() || !decoded {
		return migrationError(MigrationError{Operation: "load source", Err: ErrMigrationSourceAbsent})
	}
	if loaded.SourceVersion != CurrentVersion {
		return migrationError(MigrationError{Operation: "select source", Err: ErrMigrationV4Source})
	}
	if !bytes.Equal(snapshot.Document(), preview.SourceDocument) {
		return migrationError(MigrationError{Operation: "validate preview source", Err: ErrMigrationV4StaleSource})
	}
	if current.Generation != preview.SourceGeneration {
		return migrationError(MigrationError{Operation: "validate preview generation", Err: ErrMigrationV4StaleGeneration})
	}
	rendered, err := renderV4Preview(snapshot.Document(), loaded.Value, hosttrust.Snapshot{Generation: current.Generation, Trust: current}, preview.CredentialID, preview.DroppedPeers, now, inputs)
	if err != nil {
		return err
	}
	if !bytes.Equal(rendered.Replacement, preview.Replacement) {
		return migrationError(MigrationError{Operation: "validate preview bytes", Err: ErrMigrationV4PreviewMismatch})
	}
	// The store is already open: custody revalidation reads files only and
	// never re-acquires the held authorization lock.
	return store.ValidateCustody(previewCredentialHex(preview.CredentialID), loaded.Value.HostID, now)
}

// RollbackV4OS captures real process inputs and performs one explicit
// operator rollback: stopping every host channel is the operator's act, and
// this call replaces configuration from the preserved backup and bumps the
// trust generation so stale-generation streams cannot survive.
func RollbackV4OS(platform scalar.Platform, overrides Overrides, options RollbackV4Options) (V4ApplyResult, error) {
	inputs, err := OSInputs(platform)
	if err != nil {
		return V4ApplyResult{}, err
	}
	return rollbackV4(inputs, overrides, options, osMigrationFileSystem{}, openStoreForConfig)
}

// RollbackV4 is the dependency-injected production entry point.
func RollbackV4(inputs Inputs, overrides Overrides, options RollbackV4Options) (V4ApplyResult, error) {
	return rollbackV4(inputs, overrides, options, osMigrationFileSystem{}, openStoreForConfig)
}

// errRollbackConverged signals that the rollback target already holds: the
// configuration converged to the backup between the fast-fail check and the
// joint transaction. JointCommit stages no marker for it and passes the
// sentinel through; rollbackV4 reports the no-op success instead.
var errRollbackConverged = errors.New("configuration rollback already holds")

func rollbackV4(inputs Inputs, overrides Overrides, options RollbackV4Options,
	filesystem migrationFileSystem, openStore storeOpener) (V4ApplyResult, error) {
	if !options.AcknowledgeNoHostChannelAssurance {
		return V4ApplyResult{}, migrationError(MigrationError{Operation: "acknowledge rollback", Err: ErrRollbackV4AckRequired})
	}
	snapshot, err := Load(inputs, overrides)
	if err != nil {
		return V4ApplyResult{}, err
	}
	loaded, decoded := snapshot.Configuration()
	if !snapshot.ConfigPresent() || !decoded {
		return V4ApplyResult{}, migrationError(MigrationError{Operation: "load source", Err: ErrMigrationSourceAbsent})
	}
	localPaths := snapshot.LocalPaths()
	filename, err := configPathFromLocalPaths(localPaths)
	if err != nil {
		return V4ApplyResult{}, err
	}
	backup := options.BackupPath
	if backup == "" {
		backup = filename + ".bak." + CurrentVersion
	}
	backupBytes, err := filesystem.ReadFile(backup)
	if err != nil {
		return V4ApplyResult{}, migrationError(MigrationError{Operation: "read backup", Err: errors.Join(ErrRollbackV4Backup, err)})
	}
	context := DecodeContext{RuntimePlatform: inputs.Platform, BackendSettings: inputs.BackendSettings}
	backupLoaded, err := Decode(backupBytes, context)
	if err != nil {
		return V4ApplyResult{}, migrationError(MigrationError{Operation: "validate backup", Err: errors.Join(ErrRollbackV4Backup, err)})
	}
	if backupLoaded.SourceVersion == Version4 {
		return V4ApplyResult{}, migrationError(MigrationError{Operation: "validate backup", Err: ErrRollbackV4Backup})
	}
	if bytes.Equal(snapshot.Document(), backupBytes) {
		return V4ApplyResult{Migration: MigrationResult{SourceVersion: loaded.SourceVersion, TargetVersion: backupLoaded.SourceVersion, BackupPath: backup}}, nil
	}
	store, err := openStore(localPaths)
	if err != nil {
		return V4ApplyResult{}, err
	}
	if err := store.EnsureConfigBinding(localPaths); err != nil {
		return V4ApplyResult{}, err
	}
	preRollback := filename + ".pre-rollback." + loaded.SourceVersion
	marker := hosttrust.PendingCommit{
		Schema:            "urn:ax:schema:host-pending-commit",
		SchemaVersion:     "1.0.0",
		Operation:         hosttrust.JointRollbackV4,
		ConfigPath:        filename,
		SourceSHA256:      hosttrust.SHA256HexOf(snapshot.Document()),
		ReplacementSHA256: hosttrust.SHA256HexOf(backupBytes),
		BackupPath:        preRollback,
	}
	// The source generation pins inside the transaction: reread the trust
	// there so a concurrent commit cannot slip between this read and the
	// replacement.
	current, err := store.ReadSnapshot()
	if err != nil {
		return V4ApplyResult{}, err
	}
	marker.SourceGeneration = current.Generation
	generation, err := store.JointCommit(localPaths, marker,
		func(current hosttrust.TrustStore) error {
			return revalidateRollbackV4(inputs, overrides, snapshot.Document(), backupBytes, backupLoaded.SourceVersion, marker.SourceGeneration, current)
		},
		func(hold hosttrust.HeldExclusive) error {
			return replaceDurably(hold, filesystem, localPaths, preRollback, snapshot.Document(), backupBytes)
		},
		func(hold hosttrust.HeldExclusive) error {
			_, err := writeTempReplace(hold, filesystem, localPaths, snapshot.Document())
			return err
		})
	if err != nil {
		if errors.Is(err, errRollbackConverged) {
			return V4ApplyResult{Migration: MigrationResult{SourceVersion: loaded.SourceVersion, TargetVersion: backupLoaded.SourceVersion, BackupPath: backup}}, nil
		}
		if errors.Is(err, hosttrust.ErrJointCompensationFailed) {
			return V4ApplyResult{}, migrationError(MigrationError{Operation: "compensate joint commit", Err: errors.Join(ErrMigrationSync, ErrMigrationRecovery, err)})
		}
		return V4ApplyResult{}, err
	}
	return V4ApplyResult{
		Migration:  MigrationResult{SourceVersion: loaded.SourceVersion, TargetVersion: backupLoaded.SourceVersion, BackupPath: preRollback, Changed: true},
		Generation: generation,
	}, nil
}

// revalidateRollbackV4 repeats the rollback pins inside the joint
// transaction: the current document is still the pinned source, still
// differs from the backup, the backup still decodes below v4, and the trust
// generation has not moved. A converged target (current equals backup) is
// the no-op success, not a commit: it returns errRollbackConverged before
// any marker is staged.
func revalidateRollbackV4(inputs Inputs, overrides Overrides, sourceDocument, backupBytes []byte, backupVersion string, expectedGeneration uint64, current hosttrust.TrustStore) error {
	snapshot, err := Load(inputs, overrides)
	if err != nil {
		return err
	}
	loaded, decoded := snapshot.Configuration()
	if !snapshot.ConfigPresent() || !decoded {
		return migrationError(MigrationError{Operation: "load source", Err: ErrMigrationSourceAbsent})
	}
	_ = loaded
	if bytes.Equal(snapshot.Document(), backupBytes) {
		return errRollbackConverged
	}
	if !bytes.Equal(snapshot.Document(), sourceDocument) {
		return migrationError(MigrationError{Operation: "validate rollback source", Err: ErrRollbackV4StaleSource})
	}
	if current.Generation != expectedGeneration {
		return migrationError(MigrationError{Operation: "validate rollback generation", Err: ErrRollbackV4StaleGeneration})
	}
	context := DecodeContext{RuntimePlatform: inputs.Platform, BackendSettings: inputs.BackendSettings}
	backupLoaded, err := Decode(backupBytes, context)
	if err != nil {
		return migrationError(MigrationError{Operation: "validate backup", Err: errors.Join(ErrRollbackV4Backup, err)})
	}
	if backupLoaded.SourceVersion == Version4 || backupLoaded.SourceVersion != backupVersion {
		return migrationError(MigrationError{Operation: "validate backup", Err: ErrRollbackV4Backup})
	}
	return nil
}
