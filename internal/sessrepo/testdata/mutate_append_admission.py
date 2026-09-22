#!/usr/bin/env python3
"""Run narrowing mutants for the sessrepo append winner gate.

Usage: PYTHONDONTWRITEBYTECODE=1 python3 \
  internal/sessrepo/testdata/mutate_append_admission.py /absolute/evidence-dir

Each N-mutant keeps the gate present and admits exactly one rejected lease
class. The selected Go tests drive Repository.AppendEvent directly and through
the composing SetProfile, Emit, and EmitParked entries, so a surviving mutant
is a real production-entry failure rather than a source-text check.
"""

import hashlib
import json
from pathlib import Path
import re
import shutil
import subprocess
import sys


ROOT = Path(__file__).resolve().parents[3]
OUTPUT = Path(sys.argv[1]).resolve()
OUTPUT.mkdir(parents=True, exist_ok=True)
COPY = OUTPUT / "mutation-source"
if COPY.exists():
    raise SystemExit("refusing to overwrite an existing mutation source")
COPY.mkdir()
shutil.copytree(ROOT / "internal", COPY / "internal")
for name in ("go.mod", "go.sum", "README.md", "LOGBOOK.md"):
    shutil.copy2(ROOT / name, COPY / name)


TESTS = {
    "N-stale-winning-admission": r"^(TestAppendEventRefusesSupersededLeaseWhileTailStillMatches|TestSetProfileRefusesSupersededLeaseWhileTailStillMatches|TestEmitReachesAppendAdmissionGateAfterStaleObservation|TestEmitParkedReachesAppendAdmissionGateAfterStaleObservation)$",
    "N-same-epoch-winning-admission": r"^(TestAppendEventRefusesSameEpochLosingLeaseWhileTailStillMatches|TestSetProfileRefusesSupersededLeaseWhileTailStillMatches|TestEmitReachesSameEpochAppendAdmissionGate|TestEmitParkedReachesSameEpochAppendAdmissionGate)$",
    "C-harmless-comment": r"^TestAppendEventChainInOrder$",
    "C-not-applied": r"^TestAppendEventChainInOrder$",
    "C-compile-failure": r"^TestAppendEventChainInOrder$",
}


def command_for(name):
    pattern = TESTS.get(name)
    if pattern is None:
        raise SystemExit(f"no behavioral test mapping for {name}")
    return ["go", "test", "./internal/sessrepo", "./internal/sessprofile", "./internal/axpane", "-run", pattern, "-count=1", "-v"]


sources = {
    "chain": COPY / "internal/sessrepo/chain.go",
    "sessrepo": COPY / "internal/sessrepo/sessrepo.go",
}
originals = {name: path.read_bytes() for name, path in sources.items()}
results = []
env = dict(__import__("os").environ)
env["PYTHONDONTWRITEBYTECODE"] = "1"


def run(name, source_name, old, new, note):
    source = sources[source_name]
    original = source.read_text()
    count = original.count(old)
    if count != 1:
        results.append({"mutant": name, "classification": "NOT_APPLIED", "count": count, "bound": note})
        print(name, "NOT_APPLIED", count, flush=True)
        return
    source.write_text(original.replace(old, new))
    command = command_for(name)
    log_path = OUTPUT / f"{name}.log"
    try:
        with log_path.open("w") as log:
            process = subprocess.run(command, cwd=COPY, env=env, stdout=log, stderr=subprocess.STDOUT, timeout=180)
        text = log_path.read_text()
        failures = re.findall(r"^\s*--- FAIL: ([^ ]+)", text, re.M)
        ran_tests = re.findall(r"^=== RUN ", text, re.M)
        if not ran_tests or process.returncode not in (0, 1):
            classification = "COMPILE_OR_HARNESS_FAILURE"
        elif process.returncode == 0:
            classification = "SURVIVED"
        elif failures:
            classification = "KILLED"
        else:
            classification = "COMPILE_OR_HARNESS_FAILURE"
        results.append({
            "mutant": name,
            "classification": classification,
            "exit": process.returncode,
            "failed_tests": failures,
            "bound": note,
            "command": command,
        })
        print(name, classification, process.returncode, ", ".join(failures), flush=True)
    finally:
        source.write_text(original)
        if source.read_bytes() != originals[source_name]:
            raise SystemExit(f"failed to restore {source_name} after {name}")


PACKAGE_COMMAND = ["go", "test", "./internal/sessrepo", "./internal/sessprofile", "./internal/axpane", "-count=1"]

with (OUTPUT / "control-before.log").open("w") as log:
    before = subprocess.run(PACKAGE_COMMAND, cwd=COPY, env=env, stdout=log, stderr=subprocess.STDOUT, timeout=300)
results.append({"mutant": "control-before", "classification": "CONTROL", "exit": before.returncode})
if before.returncode:
    raise SystemExit("baseline failed")

run(
    "N-stale-winning-admission",
    "chain",
    "if event.leaseEpoch < winner.Epoch {",
    'if event.leaseEpoch < winner.Epoch && (event.leaseEpoch != winner.Epoch-1 || event.leaseID != "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee") {',
    "Admit exactly the reachable (winner epoch minus one, lease A) stale member while retaining the lower-epoch gate for every other stale class and the same-epoch arm.",
)
run(
    "N-same-epoch-winning-admission",
    "chain",
    "if event.leaseEpoch == winner.Epoch && event.leaseID != winner.LeaseID {",
    'if event.leaseEpoch == winner.Epoch && event.leaseID != winner.LeaseID && event.leaseID != "cccccccc-dddd-4eee-8fff-111111111111" {',
    "Admit exactly the reachable lease-C same-epoch loser while retaining the same-epoch gate for every other losing lease and the lower-epoch arm.",
)
run(
    "C-harmless-comment",
    "chain",
    "// checkWinningLease enforces the owner-side admission rule for a new event",
    "// checkWinningLease is a behavior-preserving owner-side admission rule for a new event",
    "Comment-only control must survive.",
)
run(
    "C-not-applied",
    "chain",
    "TOKEN_NOT_PRESENT_IN_CHAIN_SOURCE",
    "replacement",
    "Missing patch control must be reported, never counted as a kill.",
)
run(
    "C-compile-failure",
    "chain",
    "package sessrepo",
    "package sessrepo\nthis is not Go",
    "Compilation control is not a behavioral kill.",
)

with (OUTPUT / "control-after.log").open("w") as log:
    after = subprocess.run(PACKAGE_COMMAND, cwd=COPY, env=env, stdout=log, stderr=subprocess.STDOUT, timeout=300)
results.append({"mutant": "control-after", "classification": "CONTROL", "exit": after.returncode})

for name, path in sources.items():
    if path.read_bytes() != originals[name]:
        raise SystemExit(f"source changed after battery: {name}")
(OUTPUT / "mutants.json").write_text(json.dumps(results, indent=2) + "\n")
bad = [row for row in results if row["mutant"].startswith("N-") and row["classification"] != "KILLED"]
if after.returncode:
    bad.append({"mutant": "control-after", "classification": "CONTROL_FAILED"})
raise SystemExit(1 if bad else 0)
