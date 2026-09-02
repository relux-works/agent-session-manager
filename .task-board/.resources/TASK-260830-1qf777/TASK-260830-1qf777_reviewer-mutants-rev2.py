import os, shutil, subprocess, sys, json

ROOT = os.path.abspath(".temp/TASK-260830-1qf777/review-r2/tree")
CFG = os.path.join(ROOT, "internal/config")

MUTANTS = [
 # id, file, old, new, description
 ("R01","migration.go",
  "\tif compareSemver(source, target) > 0 {\n\t\treturn result, migrationError(MigrationError{Operation: \"select target\", Err: ErrMigrationDowngrade})\n\t}\n",
  "\tif false {\n\t\treturn result, migrationError(MigrationError{Operation: \"select target\", Err: ErrMigrationDowngrade})\n\t}\n",
  "delete downgrade refusal"),
 ("R02","migration.go",
  "\tif !targetKnown || options.TargetVersion == Version1 {",
  "\tif !targetKnown {",
  "narrow target vocabulary: admit Version1 as a migration target"),
 ("R03","migration.go",
  "\tif !targetKnown || options.TargetVersion == Version1 {",
  "\tif false {",
  "delete target vocabulary gate entirely"),
 ("R04","migration.go",
  "\t\tif !oneOf(options.GeneratedSummaryUpgradeChoice, \"local_only\", \"mesh_sanitized\", \"reference_only\") {",
  "\t\tif options.GeneratedSummaryUpgradeChoice == \"\" {",
  "narrow disclosure gate to empty-only (widens accepted vocabulary)"),
 ("R05","migration.go",
  "\t\tif !oneOf(options.GeneratedSummaryUpgradeChoice, \"local_only\", \"mesh_sanitized\", \"reference_only\") {",
  "\t\tif !oneOf(options.GeneratedSummaryUpgradeChoice, \"local_only\", \"mesh_sanitized\", \"reference_only\", \"broadcast_everything\") {",
  "widen disclosure vocabulary by one extra member"),
 ("R06","migration.go",
  "\t\tif readErr != nil || statErr != nil || !bytes.Equal(existing, original) || info.Mode().Perm()&0o077 != 0 {",
  "\t\tif readErr != nil || statErr != nil || info.Mode().Perm()&0o077 != 0 {",
  "narrow existing-backup gate: drop exact-content clause"),
 ("R07","migration.go",
  "\t\tif readErr != nil || statErr != nil || !bytes.Equal(existing, original) || info.Mode().Perm()&0o077 != 0 {",
  "\t\tif readErr != nil || statErr != nil || !bytes.Equal(existing, original) {",
  "narrow existing-backup gate: drop owner-only permission clause"),
 ("R08","migration.go",
  "\t\tif readErr != nil || statErr != nil || !bytes.Equal(existing, original) || info.Mode().Perm()&0o077 != 0 {",
  "\t\tif false {",
  "delete existing-backup integrity gate"),
 ("R09","migration.go",
  "\tbackupTemp, err := writeTempFile(filesystem, directory, \".ax-config-backup-*\", original, 0o600)",
  "\tbackupTemp, err := writeTempFile(filesystem, directory, \".ax-config-backup-*\", original, 0o644)",
  "widen backup permissions from owner-only to world-readable"),
 ("R10","migration.go",
  "\tif !info.Mode().IsRegular() {\n\t\treturn migrationError(MigrationError{Operation: \"inspect source mode\", Err: errors.Join(ErrMigrationWrite, ErrConfigNotRegular)})\n\t}",
  "\tif false {\n\t\treturn migrationError(MigrationError{Operation: \"inspect source mode\", Err: errors.Join(ErrMigrationWrite, ErrConfigNotRegular)})\n\t}",
  "delete post-load non-regular-source TOCTOU recheck"),
 ("R11","migration.go",
  "\tif !snapshot.ConfigPresent() || !decoded {",
  "\tif false && (!snapshot.ConfigPresent() || !decoded) {",
  "delete absent-source refusal"),
 ("R12","migration.go",
  "\tif compareSemver(source, reader) > 0 {\n\t\tmode = CompatibilityReadOnly",
  "\tif false {\n\t\tmode = CompatibilityReadOnly",
  "delete read-only downgrade determination in AssessCompatibility"),
 ("R13","migration.go",
  "\t} else if _, known := knownConfigVersion(envelope.SchemaVersion); !known {\n\t\treturn CompatibilityAssessment{}, configError(\"schema_version\", ErrUnsupportedConfigVersion)\n\t}",
  "\t}",
  "delete unsupported-source refusal in AssessCompatibility"),
 ("R14","migration.go",
  "\treader, ok := knownConfigVersion(readerVersion)\n\tif !ok {",
  "\treader, ok := knownConfigVersion(readerVersion)\n\tif false && !ok {",
  "delete unknown-reader refusal in AssessCompatibility"),
 ("R15","migration.go",
  "\tif envelope.Schema != SchemaID {",
  "\tif false {",
  "delete schema-id refusal in AssessCompatibility"),
 ("R16","loader.go",
  "\t\tStat:         os.Lstat,",
  "\t\tStat:         os.Stat,",
  "revert OS stat to symlink-following os.Stat"),
 ("R17","writer.go",
  "\tdefault:\n\t\treturn nil, configError(\"terminal.backend\", errors.Join(ErrConfigEncode, fmt.Errorf(\"cannot represent backend in Configuration %s\", Version2)))",
  "\tdefault:\n\t\tbackend = configuration.Terminal.BackendID",
  "widen v2 encoder to pass any backend id through"),
 ("R18","validation.go",
  "\tif !between(value.MaxParallelChunks, 1, 32) {",
  "\tif !between(value.MaxParallelChunks, 1, 64) {",
  "widen sync.max_parallel_chunks upper bound 32 -> 64"),
 ("R19","validation.go",
  "\tif !between(value.MaxParallelChunks, 1, 32) {",
  "\tif !between(value.MaxParallelChunks, 0, 32) {",
  "widen sync.max_parallel_chunks lower bound 1 -> 0"),
 ("R20","schema.go",
  "\tdecoder := toml.NewDecoder(bytes.NewReader(document)).DisallowUnknownFields()",
  "\tdecoder := toml.NewDecoder(bytes.NewReader(document))",
  "delete unknown-field refusal (DisallowUnknownFields)"),
 ("R21","validation.go",
  "\t\t\treturn configError(\"terminal.external_trust duplicate backend_id\", ErrConfigValidation)",
  "\t\t\t_ = seen",
  "delete duplicate external_trust backend rejection"),
 ("R22","validation.go",
  "\t\t\treturn configError(\"terminal.backend_config duplicate backend_id\", ErrConfigValidation)",
  "\t\t\t_ = seen",
  "delete duplicate backend_config backend rejection"),
 ("R23","migration.go",
  "\tif err := replaceDurably(filesystem, filename, backup, snapshot.Document(), replacement); err != nil {",
  "\tif err := func() error { return nil }(); err != nil {\n\t\t_ = replaceDurably\n\t\t_ = backup",
  "skip durable replace + backup entirely"),
 ("R24","migration.go",
  "\t\tif rollbackErr != nil {\n\t\t\tif rollbackTemp != \"\" {\n\t\t\t\t_ = filesystem.Remove(rollbackTemp)\n\t\t\t}\n\t\t\treturn migrationError(MigrationError{Operation: \"recover source after sync\", Err: errors.Join(ErrMigrationSync, ErrMigrationRecovery, syncErr, rollbackErr)})\n\t\t}",
  "\t\tif false {\n\t\t\tif rollbackTemp != \"\" {\n\t\t\t\t_ = filesystem.Remove(rollbackTemp)\n\t\t\t}\n\t\t\treturn migrationError(MigrationError{Operation: \"recover source after sync\", Err: errors.Join(ErrMigrationSync, ErrMigrationRecovery, syncErr, rollbackErr)})\n\t\t}",
  "delete rollback-failure reporting branch"),
 ("R25","migration.go",
  "\t\trollbackTemp, rollbackErr := writeTempFile(filesystem, directory, \".ax-config-rollback-*\", original, info.Mode().Perm())\n\t\tif rollbackErr == nil {\n\t\t\trollbackErr = filesystem.Rename(rollbackTemp, filename)\n\t\t}",
  "\t\trollbackTemp, rollbackErr := \"\", error(nil)\n\t\tif false {\n\t\t\trollbackErr = filesystem.Rename(rollbackTemp, filename)\n\t\t}",
  "delete post-sync-failure rollback of the original document"),
 ("R26","migration.go",
  "\tif err := syncDirectory(filesystem, directory); err != nil {\n\t\treturn migrationError(MigrationError{Operation: \"sync backup directory\", Err: errors.Join(ErrMigrationSync, err)})\n\t}",
  "\tif false {\n\t\treturn migrationError(MigrationError{Operation: \"sync backup directory\", Err: errors.Join(ErrMigrationSync, err)})\n\t}\n\t_ = syncDirectory",
  "delete backup directory fsync retry"),
]

def restore():
    subprocess.run(["git","checkout","--","internal/config"],cwd=".",check=False)

def apply(mfile, old, new):
    path = os.path.join(CFG, mfile)
    src = open(path).read()
    if src.count(old) < 1:
        return None, "PATTERN-NOT-FOUND"
    if src.count(old) > 1:
        return None, "PATTERN-AMBIGUOUS(%d)" % src.count(old)
    open(path,"w").write(src.replace(old,new,1))
    return src, None

results = []
only = sys.argv[1:] if len(sys.argv)>1 else None
for mid, mfile, old, new, desc in MUTANTS:
    if only and mid not in only: continue
    path = os.path.join(CFG, mfile)
    backup = open(path).read()
    _, err = apply(mfile, old, new)
    if err:
        results.append((mid,mfile,desc,err,""))
        open(path,"w").write(backup)
        continue
    build = subprocess.run(["go","build","./..."],cwd=ROOT,capture_output=True,text=True)
    if build.returncode != 0:
        results.append((mid,mfile,desc,"BUILD-FAIL(invalid mutant)",build.stderr[:300]))
        open(path,"w").write(backup)
        continue
    run = subprocess.run(["go","test","./internal/config/","-count=1"],cwd=ROOT,capture_output=True,text=True)
    out = (run.stdout+run.stderr)
    status = "DIED" if run.returncode != 0 else "SURVIVED"
    first = ""
    for line in out.splitlines():
        if "--- FAIL" in line or "configuration refusal call sites" in line or "FAIL\t" in line:
            first = line.strip(); break
    results.append((mid,mfile,desc,status,first))
    open(path,"w").write(backup)
    print("%s %-12s %-9s %s | %s" % (mid, mfile, status, desc, first), flush=True)

print("\n=== SUMMARY ===")
died = sum(1 for r in results if r[3]=="DIED")
surv = [r for r in results if r[3]=="SURVIVED"]
other = [r for r in results if r[3] not in ("DIED","SURVIVED")]
print("applied=%d died=%d survived=%d other=%d" % (len(results), died, len(surv), len(other)))
for r in surv: print("SURVIVED:", r[0], r[2])
for r in other: print("OTHER:", r[0], r[2], r[3], r[4][:200])
