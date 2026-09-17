#!/usr/bin/env python3
"""Host trust behavioral overlays; no source-only or compile-error kills."""
import argparse
import json
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
TRUST = "internal/hosttrust/trust.go"
DECODE = "internal/hosttrust/trust_decode.go"
ENROLL = "internal/hosttrust/enroll.go"
ROTATE = "internal/hosttrust/rotate.go"
AUTHORIZE = "internal/hosttrust/authorize.go"
JOINT = "internal/hosttrust/joint.go"
CUSTODY = "internal/hosttrust/custody.go"
CUSTODY_UNIX = "internal/hosttrust/custody_unix.go"
PROFILE = "internal/hosttrust/profile_verify.go"
TRANSACT = "internal/hosttrust/transact.go"
LOCK_UNIX = "internal/hosttrust/lock_unix.go"
HOLD = "internal/hosttrust/hold.go"
TESTS = "Test"
PROBES = [
 ("neutral", TRANSACT, "next.Generation = committed.Generation + 1", "next.Generation = committed.Generation + 1 + 0", "Equivalent generation increment", None),
 ("gen-ceiling", DECODE, "if parsed == 0 || parsed > MaxGeneration {", "if parsed == 0 || parsed > MaxGeneration+1 {", "Admits exactly generation 2^53", "TestDecodeTrustRefusals/generation_uint53_ceiling"),
 ("dup-credid", DECODE, "if previous >= current {", "if previous == current {", "Admits unsorted-but-distinct entries while keeping duplicate refusal; the mapping gate now subsumes the identical-entry case", "TestDecodeTrustRefusals/unsorted_entries"),
 ("unknown-field", TRUST, 'case "schema", "schema_version", "generation", "entries":', 'case "schema", "schema_version", "generation", "entries", "comment":', "Admits exactly one unknown field", "TestDecodeTrustRefusals/unknown_root_field"),
 ("per-host-bound", ENROLL, "if live >= 2 {", "if live >= 3 {", "Admits exactly the third non-revoked credential", "TestEnrollBoundsPerHost"),
 ("rotation-expiry", ROTATE, "deadline = oldLeaf.NotAfter", "deadline = oldLeaf.NotAfter.Add(time.Hour)", "Extends overlap one hour past old leaf expiry", "TestRotateCapsAtLeafExpiry"),
 ("revoke-state", ROTATE, "trust.Entries[index].State = EntryRevoked", "trust.Entries[index].State = EntryRetiring", "Weakens the terminal state to retiring, admitting re-revocation", "TestRevokeTombstone"),
 ("retire-bound", ROTATE, "fixed.After(now.Add(RotationBound))", "fixed.After(now.Add(48 * time.Hour))", "Admits exactly the 24-48h overlap window", "TestMarkRetiringBounds/past_24h_bound"),
 ("stale-direction", AUTHORIZE, "if snapshot.Generation != current.Generation {", "if snapshot.Generation > current.Generation {", "Admits stale-older snapshots while refusing newer", "TestAuthorizeDispatchRefusals/stale_generation"),
 ("stale-mutation-refresh", AUTHORIZE, "if request.SnapshotGeneration != current.Generation {", "if request.SnapshotGeneration > current.Generation {", "Admits older mutation bindings while refusing newer", "TestWithMutationAuthorizationRefusesStaleGeneration"),
 ("joint-marker-stale", JOINT, "if marker.SourceGeneration != committed.Generation {", "if marker.SourceGeneration > committed.Generation {", "Admits older pinned generations into the joint commit", "TestJointCommitRefusesStaleMarkerGeneration"),
 ("recover-diverged", JOINT, "case marker.SourceGeneration + 1:", "case marker.SourceGeneration + 1, marker.SourceGeneration + 2:", "Completes forward under a diverged generation", "TestRecoverRefusesDivergedGeneration"),
 ("custody-0644", CUSTODY_UNIX, "return info.Mode().Perm() == mode", "return info.Mode().Perm() == mode || info.Mode().Perm() == 0o644", "Admits exactly world-readable 0644 files", "TestCustodyRefusals/world_readable_trust_file"),
 ("expiry-grace", PROFILE, "instant.After(certificate.NotAfter)", "instant.After(certificate.NotAfter.Add(time.Second))", "Admits exactly one second past expiry", "TestVerifyProfileRefusesJustPastExpiry"),
 ("exhaustion", TRANSACT, "if committed.Generation >= MaxGeneration {", "if committed.Generation > MaxGeneration {", "Admits the ceiling generation into the commit path", "TestGenerationExhaustionRefusesMutation"),
 ("digest-root-swap", DECODE, "if scalar.SHA256Digest(entry.LeafDER).String() != entry.CredentialID.String() {", "if scalar.SHA256Digest(entry.LeafDER).String() != entry.CredentialID.String() && scalar.SHA256Digest(entry.RootDER).String() != entry.CredentialID.String() {", "Admits exactly stores naming the root digest as credential_id", "TestDecodeTrustRefusals/credential_id_not_leaf_digest"),
 ("unique-root-pair", TRUST, "if _, duplicate := roots[entry.RootID.String()]; duplicate {", "if _, duplicate := roots[entry.RootID.String()]; duplicate && len(entries) > 2 {", "Admits exactly the minimal two-entry root duplicate", "TestDecodeTrustRefusals/duplicate_root_across_entries"),
 ("retire-expiry", ROTATE, "if fixed.After(oldLeaf.NotAfter) {", "if fixed.After(oldLeaf.NotAfter.Add(time.Hour)) {", "Admits retire_at up to one hour past old leaf expiry", "TestMarkRetiringCapsAtLeafExpiry"),
 ("custody-binding", ENROLL, "if scalar.SHA256Digest(leaf.Raw).Hex() != credentialHex {", "if scalar.SHA256Digest(leaf.Raw).Hex() != credentialHex && scalar.SHA256Digest(rootBlock.Bytes).Hex() != credentialHex {", "Admits exactly directories named by the root digest", "TestValidateCustodyBindsDirectory"),
 ("dance-converge-ignore", TRANSACT, "\tif err := convergeLocked(hold, filesystem, root, paths); err != nil {\n\t\tunlock()\n\t\treturn nil, err\n\t}", "\t_ = convergeLocked(hold, filesystem, root, paths)", "Admits exactly the refused-recovery class into converged reads", "TestReadSnapshotRefusesIntervention"),
 ("transact-converge-ignore", TRANSACT, "\tif err := convergeLocked(hold, filesystem, root, paths); err != nil {\n\t\treturn err\n\t}", "\t_ = convergeLocked(hold, filesystem, root, paths)", "Admits exactly the refused-recovery class into transactions", "TestTransactionRefusesIntervention"),
 ("initialize-converge-ignore", TRANSACT, "\tif err := store.convergeLocked(hold); err != nil {\n\t\treturn Snapshot{}, err\n\t}", "\t_ = store.convergeLocked(hold)", "Admits exactly the refused-recovery class into setup", "TestInitializeRefusesIntervention"),
 ("joint-converge-skip", JOINT, "\tif err := store.convergeLocked(hold); err != nil {\n\t\treturn 0, err\n\t}\n", "", "Skips leftover convergence so the staged-marker refusal fires instead of the abort path; call-site presence paired with the recover-forward-no-bump narrowing mutant", "TestJointCommitConvergesAbortedLeftoverThenCommits"),
 ("recover-forward-no-bump", JOINT, "\t\tswitch committed.Generation {\n\t\tcase marker.SourceGeneration:\n\t\t\tif err := transactLocked(hold, filesystem, root, func(*TrustStore) error { return nil }); err != nil {\n\t\t\t\treturn RecoverResult{Found: true}, err\n\t\t\t}\n\t\tcase marker.SourceGeneration + 1:", "\t\tswitch committed.Generation {\n\t\tcase marker.SourceGeneration, marker.SourceGeneration + 1:", "Completes forward without the generation bump, admitting replaced configuration with the old generation", "TestReadSnapshotConvergesInterruptedApply"),
 ("authorize-converge-ignore", AUTHORIZE, "\tif err := store.convergeLocked(hold); err != nil {\n\t\treturn err\n\t}", "\t_ = store.convergeLocked(hold)", "Admits exactly the refused-recovery class into the mutation boundary", "TestMutationAuthorizationRefusesIntervention"),
 ("marker-unreadable", JOINT, "\tif os.IsNotExist(err) {\n\t\treturn false, nil\n\t}\n\treturn false, trustError(TrustError{Operation: \"inspect joint commit\", Err: errors.Join(ErrTrustStoreUnreadable, err)})", "\tif os.IsNotExist(err) {\n\t\treturn false, nil\n\t}\n\treturn false, nil", "Treats an unreadable marker as absent", "TestReadSnapshotRefusesUnreadableMarker"),
 ("lock-replacing-init", LOCK_UNIX, "\tif err := filesystem.CreateExclusive(path, 0o600); err != nil {\n\t\t// Lost the creation race: another process published the lock\n\t\t// first. Open the existing inode; never replace it.\n\t\tif !isLockExistsError(err) {\n\t\t\treturn nil, trustError(TrustError{Operation: \"open lock\", Err: errors.Join(ErrTrustStoreUnreadable, err)})\n\t\t}\n\t\tif err := verifyLockFile(filesystem, path); err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t\treturn &fileLock{path: path}, nil\n\t}", "\t{\n\t\tstaged, err := filesystem.CreateTemp(\"\", \"lock-stage-*\", 0o600)\n\t\tif err != nil {\n\t\t\treturn nil, trustError(TrustError{Operation: \"open lock\", Err: errors.Join(ErrTrustStoreUnreadable, err)})\n\t\t}\n\t\tname := staged.Name()\n\t\t_ = staged.Close()\n\t\tif err := filesystem.Rename(name, path); err != nil {\n\t\t\t_ = filesystem.Remove(name)\n\t\t\treturn nil, trustError(TrustError{Operation: \"open lock\", Err: errors.Join(ErrTrustStoreUnreadable, err)})\n\t\t}\n\t}", "Publishes the lock with a replacing rename, admitting the concurrent first-open race", "TestFirstOpenPreservesHeldLock"),
 ("compensate-marker-skip", JOINT, "if present && reread != intent {", "if present && reread.SourceGeneration != intent.SourceGeneration {", "Admits markers differing only in non-generation pins", "TestJointCommitCompensationRefusesTamperedMarker"),
 ("compensate-restore-verify-skip", JOINT, "if SHA256HexOf(restored) != intent.SourceSHA256 {", "if SHA256HexOf(restored) != intent.SourceSHA256 && SHA256HexOf(restored) != intent.ReplacementSHA256 {", "Admits exactly restores that leave the replacement in place", "TestJointCommitCompensationVerifiesRestore/noop_restore"),
 ("compensate-restore-ignore", JOINT, "\t\tif err := restore(configHold); err != nil {\n\t\t\treturn failed(err)\n\t\t}\n\t\trestored, err := filesystem.ReadFile(configPath)", "\t\tif err := restore(configHold); err != nil {\n\t\t\t_ = err\n\t\t}\n\t\trestored, err := filesystem.ReadFile(configPath)", "Ignores a failed restore and trusts the post-verify alone: admits exactly fail-after-write restores into the clean abort", "TestJointCommitCompensationFailsAfterDurableRestore"),
 ("hold-require-skip", HOLD, "if !hold.Valid() {", "if !hold.Valid() && !hold.IsZero() {", "Admits exactly the zero token into every token-gated helper", "TestCommitDocumentRefusesZeroHold"),
 ("hold-state-root-skip", HOLD, "\tif err != nil || stateErr != nil || !held.configAssociation || held.configPath == \"\" || held.stateRoot == \"\" || held.configPath != canonical || held.stateRoot != stateRoot {", "\tif err != nil || stateErr != nil || !held.configAssociation || held.configPath == \"\" || held.stateRoot == \"\" || held.configPath != canonical || stateRoot != stateRoot {", "Admits exactly a bound token used with a foreign StateRoot while preserving the configuration-path binding", "TestHeldExclusiveRejectsForeignConfigStateRoot"),
 ("hold-no-lock", HOLD, "\thold, unlock, err := store.lock.exclusiveHold()", "\thold := HeldExclusive{hold: &exclusiveHold{}}\n\tunlock := func() {}\n\tvar err error", "Runs the hold boundary with a genuine token but no lock, admitting exactly unserialized writes", "TestWithExclusiveHoldSerializesWithTransaction"),
 ("resolve-source-skip", JOINT, "if err == nil && SHA256HexOf(configuration) == intent.SourceSHA256 {", "if err == nil && (SHA256HexOf(configuration) == intent.SourceSHA256 || SHA256HexOf(configuration) == intent.ReplacementSHA256) {", "Treats a durable replacement as intact after a replace failure, discarding the intent", "TestResolveFailedReplaceKeepsMarker"),
 ("compensate-verify-before-restore", JOINT, "\t\tif err := restore(configHold); err != nil {\n\t\t\treturn failed(err)\n\t\t}\n\t\trestored, err := filesystem.ReadFile(configPath)\n\t\tif err != nil {\n\t\t\treturn failed(trustError(TrustError{Operation: \"read joint configuration\", Err: errors.Join(ErrTrustStoreUnreadable, err)}))\n\t\t}\n\t\tif SHA256HexOf(restored) != intent.SourceSHA256 {\n\t\t\treturn failed(trustError(TrustError{Operation: \"validate joint restore\", Err: ErrJointIntervened}))\n\t\t}", "\t\trestored, err := filesystem.ReadFile(configPath)\n\t\tif err != nil {\n\t\t\treturn failed(trustError(TrustError{Operation: \"read joint configuration\", Err: errors.Join(ErrTrustStoreUnreadable, err)}))\n\t\t}\n\t\tif SHA256HexOf(restored) != intent.SourceSHA256 {\n\t\t\treturn failed(trustError(TrustError{Operation: \"validate joint restore\", Err: ErrJointIntervened}))\n\t\t}\n\t\tif err := restore(configHold); err != nil {\n\t\t\treturn failed(err)\n\t\t}", "PRESERVE-TOKEN order swap: verifies before restoring while keeping every call, so the static call-site gate still passes and only the behavioral suite can fail", "TestJointCommitCompensatesBumpFailure"),
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
        command = ["go", "test", "-overlay", str(overlay), "./internal/hosttrust", "-count=1", "-json", "-timeout=120s", "-run", TESTS]
        proc = subprocess.run(command, cwd=ROOT, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=300)
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
