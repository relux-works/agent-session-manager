#!/usr/bin/env python3
"""Narrowing-mutant harness for internal/sessckpt (TASK-260830-14yo67).

Each plant weakens exactly one production gate so it admits one
member of the class the gate must reject; the named killer test
must then FAIL (mutant KILLED). A delete-only mutant proves only
that the gate exists and is never accepted as evidence, so every
plant here preserves the gate and narrows it. The evidence-enum
plant additionally preserves every searched-for token while
changing behavior.

One harmless SURVIVED control (a comment-only edit) proves the
harness reports survival instead of silently passing.

Usage (from the repository root):

    PYTHONDONTWRITEBYTECODE=1 python3 internal/sessckpt/mutant_harness.py \
        --log-dir .temp/TASK-260830-14yo67/mutants

For every plant the harness writes the raw per-plant subprocess
log to <log-dir>/<name>.log (command, return code, full output)
plus <log-dir>/summary.json. A plant whose anchor occurs zero or
multiple times is reported NOT_APPLIED; a plant that breaks the
build or matches no test is reported COMPILE_OR_HARNESS_FAILURE.
Both stay outside the killed numerator. The process exits nonzero
unless every killed-expectation is KILLED and every
survived-expectation SURVIVED.
"""

import argparse
import json
import os
import subprocess
import sys

PACKAGE = "./internal/sessckpt/"

PLANTS = [
    {
        "name": "N-bg-idle",
        "gate": "quiescence (safe_boundary.background_idle)",
        "file": "internal/sessckpt/capture.go",
        "old": "\tif !boundary.BackgroundIdle {",
        "new": "\tif !boundary.BackgroundIdle && boundary.OpenProcesses != 0 {",
        "run": "TestCaptureRefusesNonQuiescentBoundary/cp_n1_background_idle_false",
        "expect": "killed",
    },
    {
        "name": "N-both-present",
        "gate": "variant presence (exactly one persistence leg)",
        "file": "internal/sessckpt/capture.go",
        "old": "\tif hasProvider == hasBoard {",
        "new": "\tif !hasProvider && !hasBoard {",
        "run": "TestCaptureRefusesBadPersistenceVariant/cp_n3_both_present",
        "expect": "killed",
    },
    {
        "name": "N-swapped-direct",
        "gate": "kind-selected variant (direct takes provider)",
        "file": "internal/sessckpt/capture.go",
        "old": "\t\tif !hasProvider || hasBoard {",
        "new": "\t\tif !hasProvider && !hasBoard {",
        "run": "TestCaptureRefusesBadPersistenceVariant/swapped_direct_takes_bundle",
        "expect": "killed",
    },
    {
        "name": "N-duplicate-heads",
        "gate": "event-head sorted-unique order",
        "file": "internal/sessckpt/capture.go",
        "old": "\t\tif len(out) > 0 && name <= previous {",
        "new": "\t\tif len(out) > 0 && name < previous {",
        "run": "TestCaptureRefusesMalformedHeads/duplicated",
        "expect": "killed",
    },
    {
        "name": "N-later-epoch-head",
        "gate": "event-head at-or-before owning lease (epoch arm)",
        "file": "internal/sessckpt/capture.go",
        "old": "\t\tif summary.LeaseEpoch > inputs.LeaseEpoch {",
        "new": "\t\tif summary.LeaseEpoch > inputs.LeaseEpoch+1 {",
        "run": "TestCaptureRefusesLaterLeaseHead",
        "expect": "killed",
    },
    {
        "name": "N-idempotency-session",
        "gate": "same-operation moved-inputs conflict",
        "file": "internal/sessckpt/capture.go",
        "old": (
            "\t\tif recorded.InputDigest != inputDigest {\n"
            '\t\t\treturn CheckpointRef{}, nil, fmt.Errorf("%w: operation %s captured different inputs (idempotency_mismatch)",\n'
        ),
        "new": (
            "\t\tif recorded.InputDigest != inputDigest && recorded.SessionID != session {\n"
            '\t\t\treturn CheckpointRef{}, nil, fmt.Errorf("%w: operation %s captured different inputs (idempotency_mismatch)",\n'
        ),
        "run": "TestCaptureWithMovedInputsRefuses",
        "expect": "killed",
    },
    {
        "name": "N-evidence-enum",
        "gate": "safe_boundary evidence closed enum (token-preserving)",
        "file": "internal/sessckpt/capture.go",
        "old": "\tcase EvidenceProviderAPI, EvidenceProviderEvent, EvidenceManagedPTY, EvidenceTaskBoardBridge, EvidenceAcceptedTest:",
        "new": '\tcase EvidenceProviderAPI, EvidenceProviderEvent, EvidenceManagedPTY, EvidenceTaskBoardBridge, EvidenceAcceptedTest, "provider_gossip":',
        "run": "TestCaptureRefusesMalformedClosureMembers/bad_evidence",
        "expect": "killed",
    },
    {
        "name": "C-harmless-comment",
        "gate": "control: comment-only edit, behavior unchanged",
        "file": "internal/sessckpt/doc.go",
        "old": "// Authority: relux-works/agent-session-manager-spec@v0.6.0, Sections 5.4",
        "new": "// Authority: relux-works/agent-session-manager-spec@v0.6.0; Sections 5.4",
        "run": "TestCaptureDirectInstallsAttestedRecord",
        "expect": "survived",
    },
]


def run_plant(root, log_dir, plant):
    path = os.path.join(root, plant["file"])
    with open(path, "r", encoding="utf-8") as handle:
        original = handle.read()
    occurrences = original.count(plant["old"])
    record = {
        "name": plant["name"],
        "gate": plant["gate"],
        "file": plant["file"],
        "run": plant["run"],
        "expect": plant["expect"],
    }
    log_path = os.path.join(log_dir, plant["name"] + ".log")
    if occurrences != 1:
        record["verdict"] = "NOT_APPLIED"
        record["detail"] = "anchor occurs %d times, want exactly 1" % occurrences
        with open(log_path, "w", encoding="utf-8") as handle:
            handle.write("NOT_APPLIED: %s\n" % record["detail"])
        return record
    mutated = original.replace(plant["old"], plant["new"], 1)
    with open(path, "w", encoding="utf-8") as handle:
        handle.write(mutated)
    try:
        command = ["go", "test", PACKAGE, "-run", plant["run"], "-count=1"]
        completed = subprocess.run(
            command,
            cwd=root,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            timeout=300,
        )
        output = completed.stdout
        with open(log_path, "w", encoding="utf-8") as handle:
            handle.write("$ %s\n" % " ".join(command))
            handle.write("returncode: %d\n" % completed.returncode)
            handle.write(output)
        record["returncode"] = completed.returncode
        record["log"] = log_path
        if "no tests to run" in output or "no test files" in output:
            record["verdict"] = "COMPILE_OR_HARNESS_FAILURE"
            record["detail"] = "killer matched no test"
        elif "build failed" in output:
            record["verdict"] = "COMPILE_OR_HARNESS_FAILURE"
            record["detail"] = "plant broke compilation"
        elif completed.returncode != 0:
            record["verdict"] = "KILLED"
            record["detail"] = "killer test failed under the plant"
        else:
            record["verdict"] = "SURVIVED"
            record["detail"] = "killer test passed under the plant"
    except subprocess.TimeoutExpired:
        record["verdict"] = "COMPILE_OR_HARNESS_FAILURE"
        record["detail"] = "subprocess timed out"
        with open(log_path, "w", encoding="utf-8") as handle:
            handle.write("TIMEOUT\n")
    finally:
        with open(path, "w", encoding="utf-8") as handle:
            handle.write(original)
    return record


def main():
    parser = argparse.ArgumentParser(description="sessckpt narrowing-mutant harness")
    parser.add_argument("--log-dir", required=True, help="directory for per-plant raw logs")
    parser.add_argument("--root", default=".", help="repository root")
    args = parser.parse_args()
    root = os.path.abspath(args.root)
    log_dir = args.log_dir if os.path.isabs(args.log_dir) else os.path.join(root, args.log_dir)
    os.makedirs(log_dir, exist_ok=True)
    results = [run_plant(root, log_dir, plant) for plant in PLANTS]
    summary = {"plants": results}
    killed = sum(1 for r in results if r["verdict"] == "KILLED" and r["expect"] == "killed")
    narrowed = sum(1 for r in results if r["expect"] == "killed")
    survived_ok = sum(1 for r in results if r["verdict"] == "SURVIVED" and r["expect"] == "survived")
    survived_total = sum(1 for r in results if r["expect"] == "survived")
    not_applied = sum(1 for r in results if r["verdict"] == "NOT_APPLIED")
    harness_fail = sum(1 for r in results if r["verdict"] == "COMPILE_OR_HARNESS_FAILURE")
    summary["numerator"] = "%d of %d narrowing mutants killed" % (killed, narrowed)
    summary["control"] = "%d of %d survived controls applied" % (survived_ok, survived_total)
    summary["not_applied"] = not_applied
    summary["compile_or_harness_failure"] = harness_fail
    expectations_met = all(
        (r["verdict"] == "KILLED") if r["expect"] == "killed" else (r["verdict"] == "SURVIVED")
        for r in results
    )
    summary["expectations_met"] = expectations_met
    with open(os.path.join(log_dir, "summary.json"), "w", encoding="utf-8") as handle:
        json.dump(summary, handle, indent=2, sort_keys=True)
        handle.write("\n")
    print("narrowing: %s | control: %s | not_applied: %d | harness_failure: %d" % (
        summary["numerator"], summary["control"], not_applied, harness_fail))
    for record in results:
        print("%-22s %-28s expect=%-8s detail=%s" % (
            record["name"], record["verdict"], record["expect"], record.get("detail", "")))
    return 0 if expectations_met else 1


if __name__ == "__main__":
    sys.exit(main())
