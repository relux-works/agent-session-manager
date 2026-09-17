#!/usr/bin/env python3
"""Narrowing-mutant harness for the Section 13.13 crash gate.

Every plant weakens one gate to admit exactly one member of the
class it must reject; the gate stays present. Delete-only mutants
are not accepted as evidence. One plant (N-registry-grace11)
preserves the searched-for boundary token and changes behavior: the
static ID-set derivation still passes while the behavioral suite
fails, so the harness runs the behavioral suite for every plant,
not only the static checker.

Targets: internal/crashgate/registry.go (the applicability
classifier) and internal/matjournal/recover.go (the recovery
evaluator the gate drives). Cross-package mutation is deliberate:
the gate's classification predicates live in the evaluator.

Usage:
    PYTHONDONTWRITEBYTECODE=1 python3 internal/crashgate/mutant_harness.py --log-dir .temp/TASK-260830-17ootk/mutants

Exit status is 0 only when every narrowing mutant is KILLED and the
harmless control SURVIVES.
"""

import argparse
import json
import subprocess
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
GO_TEST = ["go", "test", "./internal/crashgate/", "-count=1"]
STATIC_RUN = GO_TEST + ["-run", "TestRegistryDerivesFromPinnedSpec|TestRegistryStructure"]
BEHAVIOR_RUN = GO_TEST + ["-run", "TestCrashGate"]
TIMEOUT = 300

PLANTS = [
    {
        "id": "N-registry-grace11",
        "kind": "narrowing",
        "file": "internal/crashgate/registry.go",
        "weakening": "GRACE-11 task-board path flipped reachable:true to NOT APPLICABLE with an owner; the CR-GRACE-11 token is preserved",
        "killer": "TestCrashGateConformance coverage + TestCrashGateNotApplicable (behavioral); static derivation must still pass",
        "find": '{Name: PathTaskBoard, Reachable: true, Driver: DriverJournalActivation, Note: "task-board activation through the journal-activation seam, evaluated with the graceful_takeover path"},',
        "replace": '{Name: PathTaskBoard, Reachable: false, Owner: "task-board bridge runtime (Section 9/13.2; launch/export/adopt execution unlanded)", Note: "task-board activation through the journal-activation seam, evaluated with the graceful_takeover path"},',
        "expect_static": "pass",
        "expect_behavior": "fail",
    },
    {
        "id": "N-recover-lease-backward",
        "kind": "narrowing",
        "file": "internal/matjournal/recover.go",
        "weakening": "lease-moved gate admits a backward epoch move (epoch != becomes epoch >)",
        "killer": "TestCrashGateRejections/lease_moved_backward",
        "find": "if input.LeaseKnown && (input.LeaseAfter.Epoch != input.LeaseBefore.Epoch || input.LeaseAfter.ID != input.LeaseBefore.ID) {",
        "replace": "if input.LeaseKnown && (input.LeaseAfter.Epoch > input.LeaseBefore.Epoch || input.LeaseAfter.ID != input.LeaseBefore.ID) {",
        "expect_static": "any",
        "expect_behavior": "fail",
    },
    {
        "id": "N-recover-native-blank",
        "kind": "narrowing",
        "file": "internal/matjournal/recover.go",
        "weakening": "substitution gate admits the relabelled blank state (empty native-after)",
        "killer": "TestCrashGateRejections/blank_relabel",
        "find": "if input.NativeAfter != input.NativeBefore {",
        "replace": 'if input.NativeAfter != input.NativeBefore && input.NativeAfter != "" {',
        "expect_static": "any",
        "expect_behavior": "fail",
    },
    {
        "id": "N-recover-two-auth-manager",
        "kind": "narrowing",
        "file": "internal/matjournal/recover.go",
        "weakening": "two-live-authorities gate admits the manager-mismatched member (additionally requires ManagerMatches)",
        "killer": "TestCrashGateRejections/two_live_authorities",
        "find": "assessment.bridgeLive() && !input.Bridge.BindingMatches {",
        "replace": "assessment.bridgeLive() && !input.Bridge.BindingMatches && input.Bridge.ManagerMatches {",
        "expect_static": "any",
        "expect_behavior": "fail",
    },
    {
        "id": "N-recover-unfenced-live",
        "kind": "narrowing",
        "file": "internal/matjournal/recover.go",
        "weakening": "unfenced-continuation gate admits state-derived liveness (explicit Live flag only)",
        "killer": "TestCrashGateRejections/unfenced_state_liveness",
        "find": "if assessment.bridgeLive() && !input.Bridge.LeaseActive && assessment.bridgeAdopted() {",
        "replace": "if input.Bridge.Live && !input.Bridge.LeaseActive && assessment.bridgeAdopted() {",
        "expect_static": "any",
        "expect_behavior": "fail",
    },
    {
        "id": "N-recover-host-rename",
        "kind": "narrowing",
        "file": "internal/matjournal/recover.go",
        "weakening": "host-condition gate keeps disk_full and admits exactly rename_blocked",
        "killer": "TestCrashGateSection1312/rename_blocked_probe_parks",
        "find": '\tcase HostRenameBlocked:\n\t\treturn assessment.park("staging_incomplete", "atomic rename blocked: staging retained, never overwritten in place",\n\t\t\t"close the blocking handles, then retry the same operation")\n',
        "replace": "",
        "expect_static": "any",
        "expect_behavior": "fail",
    },
    {
        "id": "N-recover-marker-mismatch",
        "kind": "narrowing",
        "file": "internal/matjournal/recover.go",
        "weakening": "marker gate keeps invalid and admits exactly mismatch",
        "killer": "TestCrashGateMutualExclusion/marker_mismatch_parks",
        "find": '\tcase MarkerMismatch:\n\t\treturn assessment.park("integrity_failure", "destination marker disagrees with the journal (MJ-CRASH-MARKER-MISMATCH): quarantined, nothing further mutates",\n\t\t\t"quarantine the transaction, reconcile the marker against the plan and checkpoint, then re-run recovery")\n',
        "replace": "",
        "expect_static": "any",
        "expect_behavior": "fail",
    },
    {
        "id": "N-recover-provider-floor",
        "kind": "narrowing",
        "file": "internal/matjournal/recover.go",
        "weakening": "unrecorded-prepare uncertainty floor raised from CR-MAT-05 to CR-MAT-06 (admits unknown provider status at CR-MAT-05)",
        "killer": "TestCrashGateMutualExclusion/unknown_unrecorded_prepare_parks",
        "find": "\tif journal.Provider != nil {\n\t\treturn journal.Phase == PhasePrepared || journal.Phase == PhaseCommitting ||\n\t\t\tjournal.Phase == PhaseRollingBack\n\t}\n\treturn boundaryAtOrAfter(assessment.input.Boundary, BoundaryAfterPrepareOp)",
        "replace": "\tif journal.Provider != nil {\n\t\treturn journal.Phase == PhasePrepared || journal.Phase == PhaseCommitting ||\n\t\t\tjournal.Phase == PhaseRollingBack\n\t}\n\treturn boundaryAtOrAfter(assessment.input.Boundary, BoundaryAfterOpen)",
        "expect_static": "any",
        "expect_behavior": "fail",
    },
    {
        "id": "N-recover-bridge-floor",
        "kind": "narrowing",
        "file": "internal/matjournal/recover.go",
        "weakening": "unrecorded-import uncertainty floor raised from CR-MAT-05 to CR-MAT-06 (admits unknown bridge status at CR-MAT-05)",
        "killer": "TestCrashGateMutualExclusion/unknown_unrecorded_import_parks",
        "find": "\tcase BoardNotStarted:\n\t\treturn boundaryAtOrAfter(assessment.input.Boundary, BoundaryAfterPrepareOp)",
        "replace": "\tcase BoardNotStarted:\n\t\treturn boundaryAtOrAfter(assessment.input.Boundary, BoundaryAfterOpen)",
        "expect_static": "any",
        "expect_behavior": "fail",
    },
    {
        "id": "C-harmless-comment",
        "kind": "control",
        "file": "internal/crashgate/registry.go",
        "weakening": "comment-only edit (harmless control)",
        "killer": "none (behavioral suite must pass)",
        "find": "// Registry returns an isolated copy of the classified boundary\n// table in specification order.",
        "replace": "// Registry returns an isolated copy of the classified boundary\n// table in specification order. Harmless control edit.",
        "expect_static": "pass",
        "expect_behavior": "pass",
    },
]


def run(cmd):
    proc = subprocess.run(
        cmd, cwd=REPO, capture_output=True, text=True, timeout=TIMEOUT
    )
    return proc.returncode, proc.stdout + proc.stderr


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--log-dir", required=True)
    args = parser.parse_args()
    log_dir = Path(args.log_dir)
    if not log_dir.is_absolute():
        log_dir = REPO / log_dir
    log_dir.mkdir(parents=True, exist_ok=True)

    originals = {}
    for plant in PLANTS:
        path = REPO / plant["file"]
        if plant["file"] not in originals:
            originals[plant["file"]] = path.read_text()
    for path_text in originals.values():
        if not path_text.endswith("\n"):
            print("target file lacks trailing newline; aborting")
            return 2

    def restore():
        for name, text in originals.items():
            (REPO / name).write_text(text)

    log_lines = []
    log_lines.append("pre-flight: behavioral suite on the clean tree")
    code, out = run(BEHAVIOR_RUN)
    log_lines.append(f"$ {' '.join(BEHAVIOR_RUN)}\nreturncode={code}\n{out}")
    if code != 0:
        (log_dir / "preflight.log").write_text("\n".join(log_lines) + "\n")
        print("pre-flight behavioral suite fails on the clean tree; aborting")
        return 2

    results = []
    try:
        for plant in PLANTS:
            pid = plant["id"]
            path = REPO / plant["file"]
            current = path.read_text()
            occurrences = current.count(plant["find"])
            entry = {
                "id": pid,
                "kind": plant["kind"],
                "file": plant["file"],
                "weakening": plant["weakening"],
                "killer": plant["killer"],
            }
            plant_log = [f"plant {pid} ({plant['kind']}): {plant['weakening']}", f"file: {plant['file']}"]
            if occurrences != 1:
                entry["verdict"] = "NOT_APPLIED"
                entry["detail"] = f"pattern occurs {occurrences} times, want exactly 1"
                plant_log.append(entry["detail"])
                (log_dir / f"{pid}.log").write_text("\n".join(plant_log) + "\n")
                results.append(entry)
                continue
            path.write_text(current.replace(plant["find"], plant["replace"]))
            static_code, static_out = run(STATIC_RUN)
            behavior_code, behavior_out = run(BEHAVIOR_RUN)
            plant_log.append(f"$ {' '.join(STATIC_RUN)}\nreturncode={static_code}\n{static_out}")
            plant_log.append(f"$ {' '.join(BEHAVIOR_RUN)}\nreturncode={behavior_code}\n{behavior_out}")
            entry["static"] = "pass" if static_code == 0 else "fail"
            entry["behavior"] = "pass" if behavior_code == 0 else "fail"
            if plant["kind"] == "control":
                entry["verdict"] = "SURVIVED" if behavior_code == 0 else "UNEXPECTED_KILL"
            else:
                entry["verdict"] = "KILLED" if behavior_code != 0 else "SURVIVED"
                if plant.get("expect_static") == "pass" and static_code != 0:
                    entry["verdict"] = "STATIC_KILL"
                    entry["detail"] = "static checker failed: the token-preserving claim does not hold"
            (log_dir / f"{pid}.log").write_text("\n".join(plant_log) + "\n")
            results.append(entry)
            restore()
    finally:
        restore()

    for name, text in originals.items():
        if (REPO / name).read_text() != text:
            print(f"restore failed for {name}")
            return 2
    diff = subprocess.run(
        ["git", "diff", "--quiet", "--"] + list(originals.keys()),
        cwd=REPO,
    )
    summary = {
        "plants": results,
        "killed": sum(1 for r in results if r["verdict"] == "KILLED"),
        "survived": sum(1 for r in results if r["verdict"] == "SURVIVED"),
        "not_applied": sum(1 for r in results if r["verdict"] == "NOT_APPLIED"),
        "anomalies": [r for r in results if r["verdict"] not in ("KILLED", "SURVIVED")],
        "tree_restored": diff.returncode == 0,
    }
    (log_dir / "summary.json").write_text(json.dumps(summary, indent=2) + "\n")
    narrowing = [r for r in results if r["kind"] == "narrowing"]
    controls = [r for r in results if r["kind"] == "control"]
    ok = (
        all(r["verdict"] == "KILLED" for r in narrowing)
        and all(r["verdict"] == "SURVIVED" for r in controls)
        and diff.returncode == 0
    )
    print(
        f"mutants: {summary['killed']} killed, {summary['survived']} survived, "
        f"{summary['not_applied']} not applied, anomalies={len(summary['anomalies'])}"
    )
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
