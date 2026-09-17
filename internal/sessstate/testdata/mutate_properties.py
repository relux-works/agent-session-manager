#!/usr/bin/env python3
"""Run isolated behavioral mutants for the ownership reducer properties; never modify the managed Story checkout.
Usage: PYTHONDONTWRITEBYTECODE=1 python3 internal/sessstate/testdata/mutate_properties.py /absolute/evidence/directory [name-filter]
Every subprocess runs directly, logs its real exit, and has a bounded timeout.
The battery proves each of the four story-pinned ownership invariants by
falsification: every N-mutant keeps its reducer present and weakens it to
admit exactly one member of the class the invariant must reject (or to
consult exactly one forbidden clock signal), and a named ownership property
test must fail. Each plant runs only the smallest property pattern that can
observe it; the control rows run every TestOwnership pattern across the
three packages. Full-package health is proven separately by the committed
suite runs, not by this harness. An optional name-filter runs a subset
(substring match); results merge by concatenating mutants JSON rows.
"""
import hashlib
import json
from pathlib import Path
import re
import shutil
import subprocess
import sys

root = Path(__file__).resolve().parents[3]
output = Path(sys.argv[1]).resolve()
output.mkdir(parents=True, exist_ok=True)
name_filter = sys.argv[2] if len(sys.argv) > 2 else ""
tag = re.sub(r"[^a-z0-9]+", "-", name_filter.lower()).strip("-") or "full"
copy = output / ("mutation-source-" + tag)
if copy.exists():
    raise SystemExit("refusing to overwrite an existing mutation source")
copy.mkdir()
shutil.copytree(root / "internal", copy / "internal")
for name in ("go.mod", "go.sum", "README.md", "LOGBOOK.md"):
    shutil.copy2(root / name, copy / name)
results = []

LEASEC = "cccccccc-dddd-4eee-8fff-111111111111"
HOSTB = "0198f4c8-4a10-7b22-8b3c-1234567890ac"


def ownership_command(package, pattern):
    return ["go", "test", "./internal/" + package, "-run", pattern, "-count=1", "-v"]


# Each mutant: (name, kind, file, edits|add-path+content, command, bound).
# kind edit: [(old, new)] applied to file, each anchored exactly once.
# kind add: writes path with content for the run, then removes it.
MUTANTS = [
    ("N-compare-tiebreak", "edit", "internal/sessstate/sessstate.go",
     [("\tcase a.LeaseID < b.LeaseID:\n\t\treturn -1",
        "\tcase a.LeaseID < b.LeaseID:\n\t\treturn 1"),
      ("\tcase a.LeaseID > b.LeaseID:\n\t\treturn 1",
        "\tcase a.LeaseID > b.LeaseID:\n\t\treturn -1")],
     ownership_command("sessstate", r"^TestOwnershipUnionOrderIndependent$"),
     "Invert the lease_id tie-break; every permutation still agrees but the winner disagrees with the independent maximum."),
    ("N-union-detail-first", "edit", "internal/sessstate/sessstate.go",
     [('"union lease (%d, %s) loses to (%d, %s); its history stays preserved and never applies",\n\t\t\tlease.Epoch, lease.LeaseID, winner.Epoch, winner.LeaseID)',
        '"union lease (%d, %s) loses to (%d, %s); its history stays preserved and never applies",\n\t\t\tlease.Epoch, lease.LeaseID, union[0].Epoch, union[0].LeaseID)')],
     ownership_command("sessstate", r"^TestOwnershipUnionOrderIndependent$"),
     "Name the first-arriving union entry instead of the final winner; arrivals permute the evidence while the winner stands."),
    ("N-loser-drop-same-epoch", "edit", "internal/sessstate/sessstate.go",
     [("\t\tif Compare(lease, winner) < 0 {",
        "\t\tif Compare(lease, winner) < 0 && lease.Epoch != winner.Epoch {")],
     ownership_command("sessstate", r"^TestOwnershipLoserHistoryPreserved$"),
     "Drop same-epoch losers from the preserved evidence; lower-epoch losers still report."),
    ("N-loser-drop-lower-epoch", "edit", "internal/sessstate/sessstate.go",
     [("\t\tif Compare(lease, winner) < 0 {",
        "\t\tif Compare(lease, winner) < 0 && lease.Epoch == winner.Epoch {")],
     ownership_command("sessstate", r"^TestOwnershipLoserHistoryPreserved$"),
     "Drop lower-epoch losers from the preserved evidence; same-epoch losers still report."),
    ("N-create-clock", "edit", "internal/sessrepo/lease_store.go",
     [('\tcandidate, digest, err := mintLeaseRecord(view.record.sessionID, 1,',
        '\tif input.CreatedAt < "2026-01-01T00:00:00.000Z" {\n\t\treturn LeaseRef{}, refuse(ErrInvalidLease, "clock plant: creation time precedes the policy epoch")\n\t}\n\tcandidate, digest, err := mintLeaseRecord(view.record.sessionID, 1,')],
     ownership_command("sessrepo", r"^TestOwnershipStoreClockNonAuthority$"),
     "Refuse creation timestamps before 2026; the backwards shift errors while baseline shifts still mint."),
    ("N-expiry-absolute", "edit", "internal/sessrepo/lease_store.go",
     [("\tif now.Sub(grant.ValidatedAt) > policy.RefreshInterval {",
        "\tif now.Sub(grant.ValidatedAt) > policy.RefreshInterval && grant.ValidatedAt.Year() == 2026 {")],
     ownership_command("fencing", r"^TestOwnershipGateClockNonAuthority$"),
     "Expire only grants validated in 2026; a lapsed grant translated to 2027 authorizes while the baseline refuses."),
    ("N-gate-same-epoch-loser", "edit", "internal/fencing/gate.go",
     [("\tif presented.LeaseID != observation.Winner.LeaseID {",
        '\tif presented.LeaseID != observation.Winner.LeaseID && presented.LeaseID != "' + LEASEC + '" {')],
     ownership_command("fencing", r"^TestOwnershipGateSingleAuthorizedOwner$"),
     "Admit exactly the above-winner loser past the tie-break; the loser authorizes and exactly-one fails."),
    ("N-gate-remote-holder", "edit", "internal/fencing/gate.go",
     [("\tif observation.Winner.HolderHostID != observation.LocalHostID {",
        '\tif observation.Winner.HolderHostID != observation.LocalHostID && observation.LocalHostID != "' + HOSTB + '" {')],
     ownership_command("fencing", r"^TestOwnershipGateSingleAuthorizedOwner$"),
     "Admit exactly the remote host past the locality arm; the remote exact token authorizes and the two-host exclusion fails."),
    ("C-harmless-comment", "edit", "internal/sessstate/sessstate.go",
     [("\t// First pass: validate every union entry and select the greatest",
        "\t// First pass: validate every union entry and select the greatest\n\t// classifier control: comment only, no behavior change.")],
     ownership_command("sessstate", r"^TestOwnershipUnionOrderIndependent$"),
     "Applied control: a behavior-preserving plant survives through the same instrument."),
    ("C-not-applied", "edit", "internal/sessstate/sessstate.go",
     [("TOKEN_THAT_IS_NOT_PRESENT", "replacement")],
     ownership_command("sessstate", r"^TestOwnershipUnionOrderIndependent$"),
     "Application control: missing patch is not a killed mutant."),
    ("C-compile-failure", "edit", "internal/sessstate/sessstate.go",
     [("package sessstate", "package sessstate\nthis is invalid Go")],
     ownership_command("sessstate", r"^TestOwnershipUnionOrderIndependent$"),
     "Compilation control: no named failed behavioral test is not a kill."),
]

selected = [row for row in MUTANTS if name_filter in row[0]]
if name_filter and not selected:
    raise SystemExit("name-filter matched no mutant")


def classify(name, command, bound):
    try:
        with (output / (name + "-" + tag + ".log")).open("w") as log:
            process = subprocess.run(command, cwd=copy, stdout=log, stderr=subprocess.STDOUT, timeout=300)
        text = (output / (name + "-" + tag + ".log")).read_text()
        failures = re.findall(r"^\s*--- FAIL: ([^ ]+)", text, re.M)
        ran_tests = re.findall(r"^=== RUN ", text, re.M)
        classification = "COMPILE_OR_HARNESS_FAILURE" if not ran_tests else ("SURVIVED" if process.returncode == 0 else ("KILLED" if failures else "COMPILE_OR_HARNESS_FAILURE"))
        results.append(dict(mutant=name, classification=classification, exit=process.returncode, failed_tests=failures, bound=bound, command=command))
        print(name, classification, process.returncode, ", ".join(failures), flush=True)
    except subprocess.TimeoutExpired:
        results.append(dict(mutant=name, classification="TIMEOUT", bound=bound, command=command))
        print(name, "TIMEOUT", flush=True)


def run_edit(name, path, edits, command, bound):
    source = copy / path
    original = source.read_text()
    for old, _ in edits:
        if original.count(old) != 1:
            results.append(dict(mutant=name, classification="NOT_APPLIED", count=original.count(old), bound=bound))
            print(name, "NOT_APPLIED", original.count(old), flush=True)
            return
    changed = original
    for old, new in edits:
        changed = changed.replace(old, new)
    source.write_text(changed)
    try:
        classify(name, command, bound)
    finally:
        source.write_text(original)
        assert hashlib.sha256(source.read_bytes()).digest() == hashlib.sha256(original.encode()).digest()


def run_add(name, path, content, command, bound):
    target = copy / path
    if target.exists():
        results.append(dict(mutant=name, classification="NOT_APPLIED", count=-1, bound=bound))
        print(name, "NOT_APPLIED exists", flush=True)
        return
    target.write_text(content)
    try:
        classify(name, command, bound)
    finally:
        target.unlink(missing_ok=True)
        assert not target.exists()


with (output / ("control-before-" + tag + ".log")).open("w") as log:
    before = subprocess.run(
        ["go", "test", "./internal/sessstate", "./internal/sessrepo", "./internal/fencing",
         "-run", "^TestOwnership", "-count=1"],
        cwd=copy, stdout=log, stderr=subprocess.STDOUT, timeout=600)
results.append(dict(mutant="control-before-" + tag, classification="CONTROL", exit=before.returncode))
print("control-before-" + tag, "CONTROL", before.returncode, flush=True)
if before.returncode:
    raise SystemExit("baseline failed")

for name, kind, path, payload, command, bound in selected:
    if kind == "edit":
        run_edit(name, path, payload, command, bound)
    else:
        run_add(name, path, payload, command, bound)

with (output / ("control-after-" + tag + ".log")).open("w") as log:
    after = subprocess.run(
        ["go", "test", "./internal/sessstate", "./internal/sessrepo", "./internal/fencing",
         "-run", "^TestOwnership", "-count=1"],
        cwd=copy, stdout=log, stderr=subprocess.STDOUT, timeout=600)
results.append(dict(mutant="control-after-" + tag, classification="CONTROL", exit=after.returncode))
print("control-after-" + tag, "CONTROL", after.returncode, flush=True)
(output / ("mutants-" + tag + ".json")).write_text(json.dumps(results, indent=2) + "\n")
bad = [row for row in results if row["mutant"].startswith("N-") and row["classification"] != "KILLED"]
survived_control = [row for row in results
                    if row["mutant"] == "C-harmless-comment" and row["classification"] != "SURVIVED"]
raise SystemExit(1 if bad or survived_control or after.returncode else 0)
