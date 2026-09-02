#!/usr/bin/env python3
"""Apply one production mutant at a time, run a scoped test command, require red."""
import subprocess, sys, os, shutil, tempfile

ROOT = os.environ.get("REPO_ROOT", os.getcwd())
GOCACHE = os.path.join(ROOT, ".temp/TASK-260830-1qf777/gocache")

# (id, file, old, new, test-run mask, rationale)
MUTANTS = [
 ("M01-backup-content-check", "internal/config/migration.go",
  "if readErr != nil || statErr != nil || !bytes.Equal(existing, original) || info.Mode().Perm()&0o077 != 0 {",
  "if readErr != nil || statErr != nil || (false && !bytes.Equal(existing, original)) || info.Mode().Perm()&0o077 != 0 {",
  "TestMigrateRefusesExistingBackupThatIsNotTheExactOwnerOnlySource",
  "narrow: stop comparing existing backup bytes to the source"),
 ("M02-backup-perm-check", "internal/config/migration.go",
  "if readErr != nil || statErr != nil || !bytes.Equal(existing, original) || info.Mode().Perm()&0o077 != 0 {",
  "if readErr != nil || statErr != nil || !bytes.Equal(existing, original) || (false && info.Mode().Perm()&0o077 != 0) {",
  "TestMigrateRefusesExistingBackupThatIsNotTheExactOwnerOnlySource",
  "narrow: accept a group/world readable existing backup"),
 ("M03-backup-read-failure-as-absence", "internal/config/migration.go",
  "if readErr != nil || statErr != nil || !bytes.Equal(existing, original) || info.Mode().Perm()&0o077 != 0 {",
  "if statErr != nil || (readErr == nil && !bytes.Equal(existing, original)) || info.Mode().Perm()&0o077 != 0 {",
  "TestMigrateRefusesExistingBackupThatIsNotTheExactOwnerOnlySource",
  "narrow: treat an unreadable backup as a satisfied backup"),
 ("M04-backup-verify-deleted", "internal/config/migration.go",
  """		existing, readErr := filesystem.ReadFile(backup)
		info, statErr := filesystem.Stat(backup)
		if readErr != nil || statErr != nil || !bytes.Equal(existing, original) || info.Mode().Perm()&0o077 != 0 {
			_ = filesystem.Remove(backupTemp)
			return migrationError(MigrationError{Operation: "verify backup", Err: ErrMigrationBackup})
		}""",
  """		existing, readErr := filesystem.ReadFile(backup)
		info, statErr := filesystem.Stat(backup)
		_ = bytes.Equal(existing, original)
		_, _ = readErr, statErr
		_ = info""",
  "TestMigrateRefusesExistingBackupThatIsNotTheExactOwnerOnlySource",
  "delete: publish an existing backup without verifying it"),
 ("M05-target-vocabulary-narrowed", "internal/config/migration.go",
  "if !targetKnown || options.TargetVersion == Version1 {",
  "if !targetKnown {",
  "TestMigrateRefusesEveryTargetOutsideTheUpgradeVocabulary",
  "narrow: admit Configuration 1.0.0 as a migration target"),
 ("M06-target-gate-deleted", "internal/config/migration.go",
  "if !targetKnown || options.TargetVersion == Version1 {",
  "if false && (!targetKnown || options.TargetVersion == Version1) {",
  "TestMigrateRefusesEveryTargetOutsideTheUpgradeVocabulary",
  "delete: accept any target string"),
 ("M07-absent-source-gate-deleted", "internal/config/migration.go",
  "if !snapshot.ConfigPresent() || !decoded {",
  "if false && (!snapshot.ConfigPresent() || !decoded) {",
  "TestMigrateRefusesAbsentSourceWithoutWriting",
  "delete: migrate a configuration that was never created"),
 ("M08-choice-gate-narrowed-to-empty", "internal/config/migration.go",
  'if !oneOf(options.GeneratedSummaryUpgradeChoice, "local_only", "mesh_sanitized", "reference_only") {',
  'if options.GeneratedSummaryUpgradeChoice == "" {',
  "TestMigrateRefusesDisclosureChoiceOutsideTheClosedVocabulary",
  "narrow: only an empty disclosure choice is refused"),
 ("M09-choice-vocabulary-widened", "internal/config/migration.go",
  'if !oneOf(options.GeneratedSummaryUpgradeChoice, "local_only", "mesh_sanitized", "reference_only") {',
  'if !oneOf(options.GeneratedSummaryUpgradeChoice, "local_only", "mesh_sanitized", "reference_only", "public") {',
  "TestMigrateRefusesDisclosureChoiceOutsideTheClosedVocabulary",
  "widen: admit a disclosure value the SPEC does not list"),
 ("M10-toctou-regular-file-recheck-deleted", "internal/config/migration.go",
  "if !info.Mode().IsRegular() {",
  "if false && !info.Mode().IsRegular() {",
  "TestMigrateRefusesSourceThatStoppedBeingARegularFileAfterLoad",
  "delete: overwrite a source that stopped being a regular file"),
 ("M11-downgrade-gate-narrowed", "internal/config/migration.go",
  "if compareSemver(source, target) > 0 {",
  "if compareSemver(source, target) > 1 {",
  "TestMigrateProductionEntryRequiresExplicitChoiceAndRefusesDowngrade",
  "narrow: compareSemver never exceeds 1, so downgrade is admitted"),
 ("M12-recovery-failure-silenced", "internal/config/migration.go",
  """		if rollbackErr != nil {
			if rollbackTemp != "" {
				_ = filesystem.Remove(rollbackTemp)
			}
			return migrationError(MigrationError{Operation: "recover source after sync", Err: errors.Join(ErrMigrationSync, ErrMigrationRecovery, syncErr, rollbackErr)})
		}""",
  """		if rollbackErr != nil {
			if rollbackTemp != "" {
				_ = filesystem.Remove(rollbackTemp)
			}
		}""",
  "TestMigrateReportsRecoveryFailureWhenRollbackAlsoFails",
  "delete: report a failed rollback as an ordinary sync failure"),
 ("M13-durable-replace-error-swallowed", "internal/config/migration.go",
  """	if err := filesystem.Rename(replacementTemp, filename); err != nil {
		_ = filesystem.Remove(replacementTemp)
		return migrationError(MigrationError{Operation: "replace source", Err: errors.Join(ErrMigrationReplace, err)})
	}""",
  """	if err := filesystem.Rename(replacementTemp, filename); err != nil {
		_ = filesystem.Remove(replacementTemp)
		return nil
	}""",
  "TestMigrateLeavesTheSourceIntactAtEveryDurableFailurePoint",
  "delete: report a failed atomic replace as success"),
 ("M14-readonly-downgrade-never-triggers", "internal/config/migration.go",
  "if compareSemver(source, reader) > 0 {",
  "if compareSemver(source, reader) > 1 {",
  "TestAssessCompatibilityEnforcesReadOnlyDowngradeWithoutDecodingOrMutation",
  "narrow: a newer document is assessed as fully compatible"),
 ("M15-unknown-reader-admitted", "internal/config/migration.go",
  "reader, ok := knownConfigVersion(readerVersion)\n\tif !ok {",
  "reader, ok := knownConfigVersion(readerVersion)\n\tif false && !ok {",
  "TestAssessCompatibilityRefusesMalformedEnvelopeFacts",
  "delete: assess compatibility for an unknown reader version"),
 ("M16-max-parallel-bound-widened", "internal/config/validation.go",
  "if !between(value.MaxParallelChunks, 1, 32) {",
  "if !between(value.MaxParallelChunks, 1, 64) {",
  "TestMigrateProductionEntryRefusesInvalidSourceBeforeBackupOrWrite",
  "widen: admit sync.max_parallel_chunks above the SPEC bound"),
 ("M17-duplicate-backend-admitted", "internal/config/validation.go",
  'return configError("terminal.external_trust duplicate backend_id", ErrConfigValidation)',
  '_ = index',
  "TestMigrateProductionEntryRefusesInvalidSourceBeforeBackupOrWrite",
  "delete: admit two external-trust entries for one backend_id"),
 ("M18-unknown-fields-admitted", "internal/config/schema.go",
  "decoder := toml.NewDecoder(bytes.NewReader(document)).DisallowUnknownFields()",
  "decoder := toml.NewDecoder(bytes.NewReader(document))",
  "TestMigrateProductionEntryRefusesInvalidSourceBeforeBackupOrWrite",
  "delete: admit unknown members in the closed document shape"),
 ("M19-refusal-inventory-duplicate-constructor", "internal/config/migration.go",
  "var migrationError = func(value MigrationError) error {\n\treturn &value\n}",
  "var migrationError = func(value MigrationError) error {\n\treturn &value\n}\n\nvar migrationErrorCopy = func(value MigrationError) error {\n\treturn &value\n}",
  "",
  "self-coverage: a byte-identical duplicate refusal constructor must redden the derived gate"),
 ("M21-never-exercised-refusal-site", "internal/config/migration.go",
  "func migrate(inputs Inputs, overrides Overrides, options MigrationOptions, filesystem migrationFileSystem) (MigrationResult, error) {",
  "func migrate(inputs Inputs, overrides Overrides, options MigrationOptions, filesystem migrationFileSystem) (MigrationResult, error) {\n\tif options.TargetVersion == \"never-selected-by-any-test\" {\n\t\treturn MigrationResult{}, migrationError(MigrationError{Operation: \"synthetic\", Err: ErrMigrationTarget})\n\t}",
  "",
  "self-coverage: a synthetic refusal site that no test reaches must redden the derived inventory"),
 ("M22-never-exercised-loader-site", "internal/config/loader.go",
  "func Load(inputs Inputs, overrides Overrides) (Snapshot, error) {",
  "func Load(inputs Inputs, overrides Overrides) (Snapshot, error) {\n\tif inputs.HomeDir == \"never-selected-by-any-test\" {\n\t\treturn Snapshot{}, loaderError(Error{Operation: \"synthetic\", Err: ErrInvalidContext})\n\t}",
  "",
  "self-coverage: the same property holds for the loader refusal form"),
 ("M23-mutating-seam-follows-symlinks", "internal/config/migration.go",
  "func (osMigrationFileSystem) Stat(name string) (fs.FileInfo, error) { return os.Lstat(name) }",
  "func (osMigrationFileSystem) Stat(name string) (fs.FileInfo, error) { return os.Stat(name) }",
  "TestMigrateOSAppliesTheNoFollowMutatingSeamInBothDirections",
  "widen: let the durable migration seam follow a symlink and rename over the operator's link"),
 ("M24-read-seam-refuses-symlinks", "internal/config/loader.go",
  "\t\tStat:         os.Stat,",
  "\t\tStat:         os.Lstat,",
  "TestLoadOSResolvesSymlinksAndStillEnforcesKindsAtProductionEntry",
  "narrow: refuse a symlinked root or configuration file the pinned SPEC does not refuse (the F1 defect)"),
 ("M25-source-stat-failure-admitted", "internal/config/migration.go",
  """	info, err := filesystem.Stat(filename)
	if err != nil {
		return migrationError(MigrationError{Operation: "inspect source mode", Err: errors.Join(ErrMigrationWrite, err)})
	}""",
  """	info, err := filesystem.Stat(filename)
	if false && err != nil {
		return migrationError(MigrationError{Operation: "inspect source mode", Err: errors.Join(ErrMigrationWrite, err)})
	}""",
  "TestMigrateRefusesSourceThatStoppedBeingARegularFileAfterLoad",
  "delete: treat a failed source inspection as a satisfied one"),
 ("M26-kind-check-moved-after-backup", "internal/config/migration.go",
  """	info, err := filesystem.Stat(filename)
	if err != nil {
		return migrationError(MigrationError{Operation: "inspect source mode", Err: errors.Join(ErrMigrationWrite, err)})
	}
	if !info.Mode().IsRegular() {
		return migrationError(MigrationError{Operation: "inspect source mode", Err: errors.Join(ErrMigrationWrite, ErrConfigNotRegular)})
	}
	backupTemp, err := writeTempFile(filesystem, directory, ".ax-config-backup-*", original, 0o600)""",
  """	backupTemp, err := writeTempFile(filesystem, directory, ".ax-config-backup-*", original, 0o600)
	info, statErr := filesystem.Stat(filename)
	if err == nil && statErr != nil {
		return migrationError(MigrationError{Operation: "inspect source mode", Err: errors.Join(ErrMigrationWrite, statErr)})
	}
	if err == nil && !info.Mode().IsRegular() {
		return migrationError(MigrationError{Operation: "inspect source mode", Err: errors.Join(ErrMigrationWrite, ErrConfigNotRegular)})
	}""",
  "TestMigrateRefusesSourceThatStoppedBeingARegularFileAfterLoad|TestMigrateOSAppliesTheNoFollowMutatingSeamInBothDirections",
  "reorder: keep the refusal but run it after a durable staging write, so a refused migration still mutates the directory"),
 ("M27-migrate-os-ignores-input-refusal", "internal/config/migration.go",
  """	inputs, err := OSInputs(platform)
	if err != nil {
		return MigrationResult{}, err
	}
	return Migrate(inputs, overrides, options)""",
  """	inputs, _ := OSInputs(platform)
	return Migrate(inputs, overrides, options)""",
  "TestMigrateOSRefusesAnUnknownPlatformBeforeTouchingTheFilesystem",
  "delete: let MigrateOS continue with empty inputs after OSInputs refused the platform"),
 ("M28-migration-error-echoes-wrapped-details", "internal/config/migration.go",
  """func (err *MigrationError) Error() string {
	return "configuration migration " + err.Operation + " failed"
}""",
  """func (err *MigrationError) Error() string {
	if err.Err != nil {
		return "configuration migration " + err.Operation + " failed: " + err.Err.Error()
	}
	return "configuration migration " + err.Operation + " failed"
}""",
  "TestErrorFormattingNeverEchoesWrappedDetails|TestMigrateOSAppliesTheNoFollowMutatingSeamInBothDirections",
  "widen: render the wrapped OS chain, which carries machine-local paths and document text"),
 ("M29-migrate-os-is-a-stub", "internal/config/migration.go",
  """	return Migrate(inputs, overrides, options)
}""",
  """	_ = inputs
	return MigrationResult{}, nil
}""",
  "TestMigrateOSPerformsOneDurableMigrationAtTheRealProcessEntry",
  "delete: make the advertised OS entry report success without performing the migration"),
 ("M20-refusal-inventory-bypass-literal", "internal/config/migration.go",
  'func (err *MigrationError) Unwrap() error { return err.Err }',
  'func (err *MigrationError) Unwrap() error { return err.Err }\n\nfunc unscannedRefusal() error {\n\tvalue := MigrationError{Operation: "unscanned"}\n\treturn &value\n}',
  "",
  "self-coverage: a raw literal bypassing the constructor must redden the derived gate"),
]

def run(mask):
    env = dict(os.environ, GOCACHE=GOCACHE)
    cmd = ["go", "test", "./internal/config/"]
    if mask:
        cmd += ["-run", mask]
    return subprocess.run(cmd, cwd=ROOT, env=env, capture_output=True, text=True)

def main():
    selected = sys.argv[1:] 
    failures = []
    for mid, path, old, new, mask, why in MUTANTS:
        if selected and mid not in selected:
            continue
        full = os.path.join(ROOT, path)
        original = open(full).read()
        if old not in original:
            print("SKIP %s: anchor not found in %s" % (mid, path)); failures.append(mid); continue
        open(full, "w").write(original.replace(old, new, 1))
        try:
            result = run(mask)
            status = "RED" if result.returncode != 0 else "GREEN"
            print("%-44s exit=%d %-5s  %s" % (mid, result.returncode, status, why))
            if result.returncode == 0:
                failures.append(mid)
            else:
                tail = [l for l in (result.stdout + result.stderr).splitlines() if l.strip()][:6]
                for line in tail:
                    print("      | " + line)
        finally:
            open(full, "w").write(original)
    verify = run("")
    print("\nrestored tree: go test ./internal/config/ exit=%d" % verify.returncode)
    if verify.returncode != 0:
        print(verify.stdout[-2000:], verify.stderr[-2000:])
    if failures:
        print("SURVIVING MUTANTS: %s" % ", ".join(failures)); sys.exit(1)
    print("all mutants died")

main()
