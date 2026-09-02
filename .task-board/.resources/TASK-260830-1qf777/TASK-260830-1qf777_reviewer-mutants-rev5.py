#!/usr/bin/env python3
"""Independent reviewer mutation sweep for CR-TASK-260830-1qf777 rev5.

Each mutant is applied to the restored tree, the WHOLE internal/config package
is run with -count=1 (never a -run mask), and the tree is restored afterwards.
RED  = the suite failed  -> the gate is pinned.
GREEN= the suite passed  -> the gate is unpinned (a finding).
"""
import os, shutil, subprocess, sys, tempfile

ROOT = os.path.abspath(os.path.dirname(__file__) + "/../..")
LOADER = "internal/config/loader.go"
MIGRATION = "internal/config/migration.go"
VALIDATION = "internal/config/validation.go"
SCHEMA = "internal/config/schema.go"
WRITER = "internal/config/writer.go"

# (id, file, anchor, replacement, note)
MUTANTS = [
 # --- Group A: the B1 home-capture evidence under review -------------------
 ("A01", LOADER, "\thome, homeErr := os.UserHomeDir()\n",
  "\thome, homeErr := os.UserHomeDir()\n\t_ = homeErr\n\thomeErr = nil\n",
  "rev4 survivor H01: drop the captured cause (compiles cleanly)"),
 ("A02", LOADER, "\thome, homeErr := os.UserHomeDir()\n",
  "\thome, homeErr := os.UserHomeDir()\n\thome = \"\"\n",
  "rev4 survivor H02: capture returns an empty home"),
 ("A03", LOADER, "\thome, homeErr := os.UserHomeDir()\n",
  "\t_, homeErr := os.UserHomeDir()\n\thome := \"/ax-constant-home\"\n",
  "constant home ignores the real capture"),
 ("A04", LOADER, "\t\thome, homeErr := os.UserHomeDir()\n", None, "unused"),
 ("A05", LOADER,
  "\t\treturn join(inputs.Platform, inputs.HomeDir, \"Library\", \"Application Support\", \"ax\"), nil",
  "\t\treturn join(inputs.Platform, inputs.HomeDir, \"Library\", \"Application Support\", \"ax\"), nil",
  "unused"),
 ("A06", LOADER, "\thome, homeErr := os.UserHomeDir()\n",
  "\thome, _ := os.UserHomeDir()\n\thomeErr := errors.New(\"user home unavailable\")\n",
  "self-minted substitute cause replaces the real one"),
 # --- Group B: acceptance-criteria gates, narrowing not only deletion ------
 ("B01", VALIDATION, "if !between(value.MaxParallelChunks, 1, 32) {",
  "if !between(value.MaxParallelChunks, 1, 33) {", "widen sync.max_parallel_chunks upper bound"),
 ("B02", VALIDATION, "if !between(value.MaxParallelChunks, 1, 32) {",
  "if !between(value.MaxParallelChunks, 2, 32) {", "narrow sync.max_parallel_chunks lower bound"),
 ("B03", VALIDATION, "if !between(terminal.SafeBoundaryTimeoutSeconds, 1, 3_600) {",
  "if !between(terminal.SafeBoundaryTimeoutSeconds, 1, 3_601) {", "widen terminal safe-boundary bound"),
 ("B04", SCHEMA, "decoder := toml.NewDecoder(bytes.NewReader(document)).DisallowUnknownFields()",
  "decoder := toml.NewDecoder(bytes.NewReader(document))", "accept unknown fields"),
 ("B05", VALIDATION,
  "\t\tif _, exists := trustIDs[entry.BackendID]; exists {\n\t\t\treturn configError(\"terminal.external_trust duplicate backend_id\", ErrConfigValidation)\n\t\t}\n",
  "\t\tif _, exists := trustIDs[entry.BackendID]; exists && index > 1000 {\n\t\t\treturn configError(\"terminal.external_trust duplicate backend_id\", ErrConfigValidation)\n\t\t}\n",
  "narrow external_trust duplicate backend_id rejection"),
 ("B06", VALIDATION,
  "\t\tif _, exists := configIDs[entry.BackendID]; exists {\n\t\t\treturn configError(\"terminal.backend_config duplicate backend_id\", ErrConfigValidation)\n\t\t}\n",
  "\t\tif _, exists := configIDs[entry.BackendID]; exists && index > 1000 {\n\t\t\treturn configError(\"terminal.backend_config duplicate backend_id\", ErrConfigValidation)\n\t\t}\n",
  "narrow backend_config duplicate backend_id rejection"),
 ("B07", MIGRATION,
  "if readErr != nil || statErr != nil || !bytes.Equal(existing, original) || info.Mode().Perm()&0o077 != 0 {",
  "_ = info\n\t\tif readErr != nil || statErr != nil || !bytes.Equal(existing, original) {",
  "drop the owner-only permission clause of the existing-backup integrity gate"),
 ("B08", MIGRATION,
  "if readErr != nil || statErr != nil || !bytes.Equal(existing, original) || info.Mode().Perm()&0o077 != 0 {",
  "_ = bytes.Equal(existing, original)\n\t\tif readErr != nil || statErr != nil || info.Mode().Perm()&0o077 != 0 {",
  "drop the content clause of the existing-backup integrity gate (import preserved)"),
 ("B09", MIGRATION, "\tif compareSemver(source, reader) > 0 {\n\t\tmode = CompatibilityReadOnly",
  "\tif source.major > reader.major+1 {\n\t\tmode = CompatibilityReadOnly",
  "rev3 survivor: narrow read-only downgrade to a two-major gap"),
 ("B10", MIGRATION, "\tif compareSemver(source, target) > 0 {\n\t\treturn result, migrationError(MigrationError{Operation: \"select target\", Err: ErrMigrationDowngrade})",
  "\tif source.major > target.major+1 {\n\t\treturn result, migrationError(MigrationError{Operation: \"select target\", Err: ErrMigrationDowngrade})",
  "narrow the migration downgrade refusal"),
 ("B11", WRITER, "BackendID: pointer(configuration.Terminal.BackendID), SafeBoundaryTimeoutSeconds: pointer(configuration.Terminal.SafeBoundaryTimeoutSeconds),",
  "BackendID: pointer(configuration.Terminal.BackendID),",
  "drop terminal.safe_boundary_timeout_seconds from the current wire encoder"),
 ("B12", WRITER, "ChunkBytes: pointer(configuration.Sync.ChunkBytes), MaxParallelChunks: pointer(configuration.Sync.MaxParallelChunks),",
  "ChunkBytes: pointer(configuration.Sync.ChunkBytes),",
  "drop sync.max_parallel_chunks from the current wire encoder"),
 ("B13", MIGRATION, "\tif err := file.Sync(); err != nil {\n\t\treturn clean(err)\n\t}\n", "",
  "delete the per-file fsync in writeTempFile"),
 ("B14", MIGRATION, "func (osMigrationFileSystem) Stat(name string) (fs.FileInfo, error) { return os.Lstat(name) }",
  "func (osMigrationFileSystem) Stat(name string) (fs.FileInfo, error) { return os.Stat(name) }",
  "widen the mutating seam to follow symlinks"),
 ("B15", LOADER, "\t\tStat:         os.Stat,", "\t\tStat:         os.Lstat,",
  "narrow the read seam so a symlinked root is refused"),
 ("B16", MIGRATION, "\tif !targetKnown || options.TargetVersion == Version1 {",
  "\tif !targetKnown {", "drop the Version1-target clause of ErrMigrationTarget"),
 ("B17", MIGRATION, "if !oneOf(options.GeneratedSummaryUpgradeChoice, \"local_only\", \"mesh_sanitized\", \"reference_only\") {",
  "if !oneOf(options.GeneratedSummaryUpgradeChoice, \"local_only\", \"mesh_sanitized\", \"reference_only\", \"anything\") {",
  "widen the disclosure-choice vocabulary"),
 ("B18", MIGRATION, "\tif !info.Mode().IsRegular() {\n\t\treturn migrationError(MigrationError{Operation: \"inspect source mode\", Err: errors.Join(ErrMigrationWrite, ErrConfigNotRegular)})\n\t}\n",
  "", "delete the pre-durable source-kind refusal"),
 ("B19", MIGRATION, "\tif !snapshot.ConfigPresent() || !decoded {",
  "\tif !snapshot.ConfigPresent() && !decoded {", "narrow the absent-source refusal"),
 ("B20", MIGRATION, "\treader, ok := knownConfigVersion(readerVersion)\n\tif !ok {",
  "\treader, ok := knownConfigVersion(readerVersion)\n\tif !ok && readerVersion != \"9.9.9\" {",
  "widen the known-reader-version gate"),
 ("B21", MIGRATION, "if !oneOf(options.GeneratedSummaryUpgradeChoice, \"local_only\", \"mesh_sanitized\", \"reference_only\") {",
  "if !oneOf(options.GeneratedSummaryUpgradeChoice, \"local_only\", \"mesh_sanitized\", \"reference_only\", \"public\") {",
  "widen the disclosure vocabulary to admit a value the suite already supplies"),
 ("B22", VALIDATION, "if !oneOf(directory.Mode, \"on_demand\", \"service\") {",
  "if !oneOf(directory.Mode, \"on_demand\", \"service\", \"anything\") {",
  "control: arbitrary widening of a sibling closed vocabulary"),
 ("B23", VALIDATION, "if !oneOf(directory.GeneratedSummaryUpgradeChoice, \"unset\", \"local_only\", \"mesh_sanitized\", \"reference_only\") {",
  "if !oneOf(directory.GeneratedSummaryUpgradeChoice, \"unset\", \"local_only\", \"mesh_sanitized\", \"reference_only\", \"everything\") {",
  "control: arbitrary widening of the config-side disclosure vocabulary"),
 ("C01", MIGRATION, "migrationError(MigrationError{Operation: \"select target\", Err: ErrMigrationTarget})",
  "error(&MigrationError{Operation: \"select target\", Err: ErrMigrationTarget})",
  "bypass literal: build a refusal outside the single constructor"),
 ("C02", MIGRATION, [(MIGRATION, "var migrationError = func(value MigrationError) error {\n\treturn &value\n}",
   "var migrationError = func(value MigrationError) error {\n\treturn &value\n}\n\nvar migrationErrorTwo = func(value MigrationError) error {\n\treturn &value\n}"),
   (MIGRATION, "migrationError(MigrationError{Operation: \"select target\", Err: ErrMigrationDowngrade})",
    "migrationErrorTwo(MigrationError{Operation: \"select target\", Err: ErrMigrationDowngrade})")], "x",
  "duplicate constructor: a second equivalent construction form"),
 ("C03", MIGRATION, "\tsnapshot, err := Load(inputs, overrides)\n",
  "\tif options.TargetVersion == \"\\x00never\" {\n\t\treturn MigrationResult{}, migrationError(MigrationError{Operation: \"synthetic never exercised\", Err: ErrMigrationTarget})\n\t}\n\tsnapshot, err := Load(inputs, overrides)\n",
  "synthetic never-exercised refusal site"),
 ("C04", MIGRATION, "\treader, ok := knownConfigVersion(readerVersion)\n\tif !ok {",
  "\treader, ok := knownConfigVersion(readerVersion)\n\tif !ok && readerVersion != \"0.0.0\" {",
  "widen the known-reader gate to admit the value the suite supplies"),
 ("D01", LOADER, " // config-refusal-subsumed: os.Getwd has no injectable seam and does not fail on any supported host, including a cwd unlinked underneath the process, so this guard stays fail-closed and is pinned by the refusal-subsumption inventory", "",
  "delete a declared subsumption marker"),
 ("D02", MIGRATION, "\tsnapshot, err := Load(inputs, overrides)\n",
  "\tif options.TargetVersion == \"\\x00never\" {\n\t\treturn MigrationResult{}, migrationError(MigrationError{Operation: \"synthetic never exercised\", Err: ErrMigrationTarget}) // config-refusal-subsumed: synthetic claim added by the reviewer to test whether a marker can silence a brand-new unexercised site\n\t}\n\tsnapshot, err := Load(inputs, overrides)\n",
  "silence a synthetic never-exercised site with a self-declared subsumption marker"),
 ("E01", MIGRATION, "\t\tif rollbackErr == nil {\n\t\t\trollbackErr = filesystem.Rename(rollbackTemp, filename)\n\t\t}\n",
  "", "skip the rollback rename after a failed replacement sync"),
 ("E02", MIGRATION, "writeTempFile(filesystem, directory, \".ax-config-backup-*\", original, 0o600)",
  "writeTempFile(filesystem, directory, \".ax-config-backup-*\", replacement, 0o600)",
  "back up the new document instead of the original"),
 ("E03", MIGRATION, "\tif err := filesystem.Rename(replacementTemp, filename); err != nil {",
  "\tif err := writeNonAtomically(filesystem, replacementTemp, filename, replacement); err != nil {",
  "replace the atomic rename with a torn in-place write"),
 # --- positive control -----------------------------------------------------
 ("PC1", MIGRATION, "// MigrationError renders no machine-local path or document content.",
  "// MigrationError renders no machine-local path or document content (control).",
  "comment-only change; must stay GREEN"),
]
MUTANTS = [m for m in MUTANTS if m[3] is not None and m[4] != "unused"]

def run_suite():
    env = dict(os.environ)
    env["GOCACHE"] = os.path.join(ROOT, ".temp/TASK-260830-1qf777/gocache")
    p = subprocess.run(["go", "test", "./internal/config", "-count=1"],
                       cwd=ROOT, env=env, capture_output=True, text=True)
    return p.returncode, (p.stdout + p.stderr)

def main():
    only = sys.argv[1:] if len(sys.argv) > 1 else None
    code, out = run_suite()
    if code != 0:
        print("RESTORED TREE IS NOT GREEN - aborting\n" + out[-4000:])
        return 2
    print("restored tree green")
    results = []
    for mid, path, anchor, replacement, note in MUTANTS:
        if only and mid not in only:
            continue
        edits = anchor if isinstance(anchor, list) else [(path, anchor, replacement)]
        originals = {}
        bad = False
        for epath, eanchor, erepl in edits:
            efull = os.path.join(ROOT, epath)
            if efull not in originals:
                originals[efull] = open(efull, encoding="utf-8").read()
            if originals[efull].count(eanchor) != 1 and open(efull, encoding="utf-8").read().count(eanchor) != 1:
                print(f"{mid} INVALID anchor occurs {open(efull, encoding='utf-8').read().count(eanchor)} times in {epath}")
                bad = True
        if bad:
            results.append((mid, "INVALID", note))
            continue
        try:
            for epath, eanchor, erepl in edits:
                efull = os.path.join(ROOT, epath)
                cur = open(efull, encoding="utf-8").read()
                open(efull, "w", encoding="utf-8").write(cur.replace(eanchor, erepl, 1))
            code, out = run_suite()
        finally:
            for efull, content in originals.items():
                open(efull, "w", encoding="utf-8").write(content)
        if code == 0:
            verdict = "GREEN(SURVIVOR)"
            tail = ""
        else:
            verdict = "RED"
            fails = [l for l in out.splitlines() if l.startswith("--- FAIL")]
            tail = "  killed by: " + (fails[0][11:] if fails else "build/vet failure")
        print(f"{mid} {verdict} {path}: {note}\n{tail}")
        results.append((mid, verdict, note))
    code, out = run_suite()
    print("post-sweep restored tree:", "green" if code == 0 else "NOT GREEN")
    red = sum(1 for _, v, _ in results if v == "RED")
    green = [m for m, v, _ in results if v == "GREEN(SURVIVOR)"]
    invalid = [m for m, v, _ in results if v == "INVALID"]
    print(f"\nSUMMARY: {len(results)} mutants, {red} RED, {len(green)} SURVIVOR {green}, {len(invalid)} INVALID {invalid}")
    return 0

sys.exit(main())
