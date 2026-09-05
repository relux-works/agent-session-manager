#!/usr/bin/env python3
"""Narrowing-mutant sweep over the provhost operation layer.

For each mutant: copy the file aside, apply exactly one source
replacement (asserted count==1), assert the mutant marker is present
on disk, run the full provhost package (expect red), restore the
aside copy, and verify byte-identical restoration. A mutant that does
not compile is not a red; a mutant that leaves the suite green is a
survivor. Hash the package tree before and after: any drift fails.
"""
import hashlib
import pathlib
import subprocess
import sys

ROOT = pathlib.Path("/Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/STORY-260830-3jqsx1/worktree")
PKG = ROOT / "internal" / "provhost"
ASIDE = pathlib.Path("/tmp/sweep-aside")
ASIDE.mkdir(exist_ok=True)

MUTANTS = [
    # (id, file, old, new) — old must occur exactly once.
    ("M01-ops-order", "manifest.go",
     "\tfor index, operation := range operationOrder {\n\t\tif operations[index] != string(operation) {",
     "\tfor index, operation := range operationOrder {\n\t\tif index > 0 && operations[index] != string(operation) {"),
    ("M02-caps-order", "manifest.go",
     "\tfor index, name := range capabilityOrder {\n\t\tif names[index] != name {",
     "\tfor index, name := range capabilityOrder {\n\t\tif index > 0 && names[index] != name {"),
    ("M03-platforms-sorted", "manifest.go",
     "\tif !sortedUniqueStrings(platforms) {",
     "\tif hasDuplicateString(platforms) {"),
    ("M04-display-bound", "manifest.go",
     "runeLength(name) < 1 || runeLength(name) > 128",
     "runeLength(name) < 1 || runeLength(name) > 129"),
    ("M05-semver-grammar", "manifest.go",
     "!semverPattern.MatchString(version)",
     "len(version) == 0"),
    ("M06-enabled-gate", "probe.go",
     "if enabled && status != CapabilityAvailable {",
     "if enabled && status == CapabilityUnknown {"),
    ("M07-usable-or", "probe.go",
     "return status == CapabilityAvailable && enabled",
     "return status == CapabilityAvailable || enabled"),
    ("M08-warnings-bound", "probe.go",
     "if !ok || len(warnings) > 1024 {",
     "if !ok || len(warnings) > 1025 {"),
    ("M09-status-vocab", "probe.go",
     "if !ok || !isProbeStatus(status) {",
     "if !ok {"),
    ("M10-standard-omission", "profile.go",
     "\tcase ProfileStandard:\n\t\treturn \"\", nil",
     "\tcase ProfileStandard:\n\t\treturn mapping, nil"),
    ("M11-pi-mapping", "profile.go",
     '"pi":          "default_unrestricted_tool_set",',
     '"pi":          "--yolo",'),
    ("M12-safe-background", "quiesce.go",
     "backgroundNull || !background || boundaryNull",
     "backgroundNull || boundaryNull"),
    ("M13-counts-zeroed", "quiesce.go",
     "\treturn counts[0], counts[1], nil",
     "\treturn 0, 0, nil"),
    ("M14-blockers-sorted", "quiesce.go",
     "\tif !sortedUniqueStrings(blockers) {",
     "\tif hasDuplicateString(blockers) {"),
    ("M15-literal-disjoint", "spawn.go",
     "\t\tif !validEnvName(key) || inherited[key] {",
     "\t\tif !validEnvName(key) {"),
    ("M16-argv-total", "spawn.go",
     "\tif total > maxSpawnArgvBytes {",
     "\tif total > 1<<20 {"),
    ("M17-mapping-equality", "spawn.go",
     "\tif mapping != want {",
     "\tif mapping != want && mapping != \"\" {"),
    ("M18-subject-session", "identity.go",
     "\tif subject != session {",
     "\tif false {"),
    ("M19-realm-kind", "identity.go",
     '\tif provider == "antigravity" && kind == "backend_conversation_uuid" && isNull {',
     '\tif provider == "antigravity" && kind == "session_uuid" && isNull {'),
    ("M20-opaque-prefix", "identity.go",
     'if strings.HasPrefix(value, "/") || strings.HasPrefix(value, `\\\\`) || windowsDrivePattern.MatchString(value) {',
     'if strings.HasPrefix(value, `\\\\`) || windowsDrivePattern.MatchString(value) {'),
    ("M21-keyed-widen", "idempotency.go",
     "\tfor _, keyed := range keyedOperations {\n\t\tif operation == keyed {",
     "\tfor _, keyed := range keyedOperations {\n\t\tif operation == keyed || true {"),
    ("M22-drop-fork", "idempotency.go",
     "\tOpMaterializeRollback,\n\tOpFork,",
     "\tOpMaterializeRollback,"),
    ("M23-key-unknown-op", "idempotency.go",
     '\tif !validOperation(string(operation)) {',
     '\tif false {'),
    ("M24-uint53-bound", "opdecode.go",
     "\tif err != nil || parsed > maxUint53 {",
     "\tif err != nil || parsed > 1<<53 {"),
    ("M25-sorted-unique", "opdecode.go",
     "\treturn sort.StringsAreSorted(values) && !hasDuplicateString(values)",
     "\treturn sort.StringsAreSorted(values)"),
    ("M26-gate-routing", "probe.go",
     """	known := false
	for _, registry := range capabilityOrder {
		if name == registry {
			known = true
		}
	}
	if !known {""",
     """	known := false
	for _, registry := range capabilityOrder {
		if name == registry {
			known = true
		}
	}
	_ = known
	if false {"""),
]


def tree_hash():
    h = hashlib.sha256()
    for path in sorted(PKG.glob("*.go")):
        h.update(path.read_bytes())
    return h.hexdigest()


def run_package():
    proc = subprocess.run(
        ["go", "test", "./internal/provhost/", "-count=1"],
        cwd=ROOT, capture_output=True, text=True, timeout=300)
    return proc.returncode


def main():
    before = tree_hash()
    print(f"tree before: {before}")
    kills, survivors, errors = [], [], []
    for mid, fname, old, new in MUTANTS:
        path = PKG / fname
        aside = ASIDE / (mid + "__" + fname)
        aside.write_bytes(path.read_bytes())
        text = path.read_text()
        count = text.count(old)
        if count != 1:
            errors.append(f"{mid}: anchor occurs {count}x, want 1")
            aside.unlink()
            continue
        path.write_text(text.replace(old, new))
        # Presence: mutant marker on disk before believing any color.
        on_disk = path.read_text()
        if new not in on_disk:
            errors.append(f"{mid}: mutant absent after apply")
            aside.write_bytes(path.read_bytes())  # keep aside intact
            path.write_bytes(aside.read_bytes())
            continue
        try:
            code = run_package()
        except subprocess.TimeoutExpired:
            code = "TIMEOUT"
        # Restore and verify byte-for-byte.
        want = aside.read_bytes()
        path.write_bytes(want)
        aside.unlink()
        if path.read_bytes() != want:
            errors.append(f"{mid}: restore mismatch")
            print(f"{mid}: RESTORE MISMATCH")
            continue
        if code == 0:
            survivors.append(mid)
            print(f"{mid}: SURVIVED (exit 0)")
        elif code == "TIMEOUT":
            errors.append(f"{mid}: verification timed out")
            print(f"{mid}: TIMEOUT")
        else:
            kills.append(mid)
            print(f"{mid}: killed (exit {code})")
    after = tree_hash()
    print(f"tree after:  {after}")
    print(f"kills={len(kills)} survivors={survivors} errors={errors}")
    if before != after:
        print("TREE DRIFT: restore failed")
        sys.exit(2)
    if survivors or errors:
        sys.exit(1)
    print("SWEEP CLEAN")


if __name__ == "__main__":
    main()
