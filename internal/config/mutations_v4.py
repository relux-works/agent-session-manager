#!/usr/bin/env python3
"""Configuration 4.0.0 behavioral overlays; no source-only or compile-error kills."""
import argparse
import json
import os
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
VALIDATION = "internal/config/validation.go"
APPLY = "internal/config/migration_v4_apply.go"
MIGRATION = "internal/config/migration.go"
PREVIEW = "internal/config/migration_v4.go"
CENSUS = "internal/config/writepath_census_types_test.go"
HOLD = "internal/hosttrust/hold.go"
TESTS = "Test"
PROBES = [
 ("neutral", PREVIEW, "SourceGeneration: trust.Generation,", "SourceGeneration: trust.Generation + 0,", "Equivalent generation pin", None),
 ("transport-ssh", VALIDATION, "if mesh.Transport != TransportSSHTLS13 {", "if mesh.Transport != TransportSSHTLS13 && mesh.Transport != TransportSSH {", "Admits exactly the legacy ssh transport as v4", "TestDecodeConfiguration4Refusals/legacy_transport"),
 ("preview-length", APPLY, 'if !bytes.Equal(rendered.Replacement, preview.Replacement) {\n\t\treturn migrationError(MigrationError{Operation: "validate preview bytes", Err: ErrMigrationV4PreviewMismatch})', 'if len(rendered.Replacement) != len(preview.Replacement) {\n\t\treturn migrationError(MigrationError{Operation: "validate preview bytes", Err: ErrMigrationV4PreviewMismatch})', "Admits same-length tampered previews at the in-lock revalidation", "TestRevalidateApplyV4PreviewMismatch"),
 ("confirm-drops", APPLY, "if !confirm {", "if !confirm && len(preview.DroppedPeers) == 0 {", "Admits exactly unconfirmed previews that drop peers", "TestApplyV4ConfirmRequiredWithDrops"),
 ("apply-gen-direction", APPLY, 'if current.Generation != preview.SourceGeneration {\n\t\treturn migrationError(MigrationError{Operation: "validate preview generation", Err: ErrMigrationV4StaleGeneration})', 'if current.Generation < preview.SourceGeneration {\n\t\treturn migrationError(MigrationError{Operation: "validate preview generation", Err: ErrMigrationV4StaleGeneration})', "Admits newer generations into the in-lock apply revalidation", "TestRevalidateApplyV4StaleGeneration"),
 ("v4-explicit-label", MIGRATION, 'migrationError(MigrationError{Operation: "select target", Err: ErrMigrationV4Explicit})', 'migrationError(MigrationError{Operation: "select target", Err: ErrMigrationTarget})', "LABEL-PRECISION, not a semantic admission: mislabels the explicit-preview refusal; named test pins the exact error", "TestMigrateRefusesV4Target"),
 ("compensate-source-swap", APPLY, "writeTempReplace(hold, filesystem, localPaths, preview.SourceDocument)", "writeTempReplace(hold, filesystem, localPaths, preview.Replacement)", "Restores the replacement instead of the source: admits exactly the no-op apply restore", "TestApplyV4CompensatesBumpFailure"),
 ("compensate-rollback-source-swap", APPLY, "writeTempReplace(hold, filesystem, localPaths, snapshot.Document())", "writeTempReplace(hold, filesystem, localPaths, backupBytes)", "Restores the backup instead of the source: admits exactly the no-op rollback restore", "TestRollbackV4CompensatesBumpFailure"),
 ("replace-require-skip", MIGRATION, "func requireHoldForConfig(hold hosttrust.HeldExclusive, paths localstore.ResolvedPaths) error {\n\tif err := requireHold(hold); err != nil {\n\t\treturn err\n\t}\n\tif err := hold.ValidateConfigPaths(paths); err != nil {", "func requireHoldForConfig(hold hosttrust.HeldExclusive, paths localstore.ResolvedPaths) error {\n\tif hold.IsZero() {\n\t\treturn nil\n\t}\n\tif err := requireHold(hold); err != nil {\n\t\treturn err\n\t}\n\tif err := hold.ValidateConfigPaths(paths); err != nil {", "Admits exactly the zero token into the config pair writers", "TestReplaceDurablyRefusesZeroHold"),
 ("legacy-revalidate-skip", MIGRATION, "\tif !bytes.Equal(current.Document(), snapshot.Document()) {\n\t\treturn migrationError(MigrationError{Operation: \"revalidate source\", Err: ErrMigrationStaleSource})\n\t}\n", "", "Drops the byte comparison while keeping the version check: admits exactly same-version source changes", "TestRevalidateLegacySourceRefusesChangedBytes"),
 ("replace-restore-skip", MIGRATION, "\t\t\trollbackErr = filesystem.Rename(rollbackTemp, filename)", "\t\t\trollbackErr = filesystem.Remove(rollbackTemp)", "Cleans the restore staging instead of reinstalling it: admits exactly unrestored post-rename sync failures", "TestMigrateRecoversOriginalAfterPostReplaceDirectorySyncFailureAndRetries"),
 ("writepath-method-value-skip", CENSUS, "\tif censusIsOSMutation(function) || censusIsRawMutation(function) {\n\t\tposition := scanner.unit.fset.Position(selector.Pos())", "\tif censusIsOSMutation(function) || (censusIsRawMutation(function) && function.Name() == \"not-a-real-mutation\") {\n\t\tposition := scanner.unit.fset.Position(selector.Pos())", "Preserves the raw-mutation symbol check while admitting method values such as backend Rename into package variables", "TestWritePathGateFlagsExecutableRogues/backend_method_value"),
 ("writepath-interface-callsite-skip", CENSUS, "\t\tif !scanner.allowedInterfaceCall(current, call, object) {", "\t\tif !scanner.allowedInterfaceMethods[current][object] {", "Preserves the allowlisted interface-method identity while admitting the same method at every call site and before hold verification", "TestWritePathCensusRejectsUnlistedInterfaceCallSite"),
 ("config-association-root-skip", HOLD, "\tassociated := targetPresent && validateBindingForStore(target, binding.ConfigPath, store.root) == nil", "\tassociated := targetPresent && target.ConfigPath == binding.ConfigPath", "Preserves target presence and exact target-path checks while admitting a foreign store whose state-root association is different", "TestWriteTempReplaceRefusesForeignStoreWithTargetBinding"),
]

BOUNDS = {
 "neutral": "Equivalent arithmetic; no independent kill claimed.",
}

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True)
    parser.add_argument("--only", help="Comma-separated probe names for a bounded follow-up")
    opts = parser.parse_args()
    if opts.only and not set(opts.only.split(",")).issubset({p[0] for p in PROBES}):
        parser.error("--only contains an unknown probe")
    out = Path(opts.output).resolve()
    out.mkdir(parents=True, exist_ok=True)
    results = []
    unexpected = False
    for name, path, old, new, narrows, expected_test in PROBES:
        if opts.only and name not in opts.only.split(","):
            continue
        source = ROOT / path
        original = source.read_bytes()
        text = original.decode()
        if text.count(old) != 1:
            raise RuntimeError(f"{name}: plant must match exactly once, got {text.count(old)}")
        folder = out / name
        folder.mkdir(exist_ok=True)
        replacement = folder / source.name
        replacement.write_text(text.replace(old, new, 1))
        overlay = folder / "overlay.json"
        overlay.write_text(json.dumps({"Replace": {str(source): str(replacement)}}))
        command = ["go", "test", "-overlay", str(overlay), "./internal/config", "-count=1", "-json", "-timeout=120s", "-run", TESTS]
        environment = os.environ.copy()
        environment["CENSUS_MUTATION_OVERLAY"] = str(overlay)
        proc = subprocess.run(command, cwd=ROOT, env=environment, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=300)
        (folder / "test.log").write_bytes(proc.stdout)
        events = []
        for line in proc.stdout.decode(errors="replace").splitlines():
            try:
                events.append(json.loads(line))
            except json.JSONDecodeError:
                pass
        failed = [e["Test"] for e in events if e.get("Action") == "fail" and "Test" in e]
        passed = [e["Test"] for e in events if e.get("Action") == "pass" and "Test" in e]
        killed = proc.returncode != 0 and bool(failed) and (expected_test is None or expected_test in failed)
        if name == "neutral":
            valid = proc.returncode == 0 and bool(passed)
            state = "neutral passed" if valid else "invalid control"
        elif name in BOUNDS:
            valid = proc.returncode == 0 and bool(passed)
            state = "survived" if valid else "unexpected result"
        else:
            valid = killed
            state = "killed" if killed else "survived" if proc.returncode == 0 and passed else "invalid instrument/compile failure"
        if not valid:
            unexpected = True
        if source.read_bytes() != original:
            raise RuntimeError(f"{name}: source changed during overlay run")
        row = {"mutant": name, "narrowing": narrows, "exit_code": proc.returncode, "result": state,
               "named_failing_tests": failed, "expected_test": expected_test,
               "bound": BOUNDS.get(name, "" if killed or name == "neutral" else "No named failing test established; this gate member is unproven."),
               "command": command}
        results.append(row)
        print(f"{name}: {state}, exit={proc.returncode}, tests={','.join(failed)}", flush=True)
        (out / "results.json").write_text(json.dumps(results, indent=2) + "\n")
    rows = ["| Mutant | What the gate is narrowed to admit | Named failing test | Exit | Result / surviving bound |",
            "| --- | --- | --- | ---: | --- |"]
    for r in results:
        tests = r["expected_test"] if r["expected_test"] in r["named_failing_tests"] else ", ".join(r["named_failing_tests"])
        rows.append(f'| {r["mutant"]} | {r["narrowing"]} | {tests or "none"} | {r["exit_code"]} | {r["result"]}: {r["bound"]} |')
    (out / "table.md").write_text("\n".join(rows)+"\n")
    return int(unexpected)


if __name__ == "__main__":
    sys.exit(main())
