#!/usr/bin/env python3
"""Run isolated behavioral mutants for the fencing gates; never modify the managed Story checkout.
Usage: python3 internal/fencing/testdata/mutate.py /absolute/evidence/directory [name-filter]
Every subprocess runs directly, logs its real exit, and has a bounded timeout.
The battery proves each fencing gate by narrowing, never by deletion:
every N-mutant keeps its gate present and weakens it to admit exactly one
member of the class the gate must reject, and a named behavioral test must
fail. The T-mutant preserves the searched-for leaseSeal token while
constructing through an alias, and the harness executes the full behavioral
suite for it, not only the static census. An optional name-filter runs a
subset (substring match); results merge by concatenating mutants JSON rows.
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

LEASEA = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
LEASEC = "cccccccc-dddd-4eee-8fff-111111111111"
SESSIONB = "0198f4c8-3e70-7a11-8a2b-1234567890ac"

GATE_IMPORT = '\t"errors"\n\t"fmt"\n\t"time"'
GATE_IMPORT_STRINGS = '\t"errors"\n\t"fmt"\n\t"strings"\n\t"time"'

# Each mutant: (name, kind, file, edits|add-path+content, command, bound).
# kind edit: [(old, new)] applied to file, each anchored exactly once.
# kind add: writes path with content for the run, then removes it.
MUTANTS = [
    ("N-op", "edit", "internal/fencing/gate.go",
     [("case OperationActivation, OperationInput, OperationMutation, OperationCheckpoint, OperationRestore:",
        'case OperationActivation, OperationInput, OperationMutation, OperationCheckpoint, OperationRestore, "bogus":')],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeRefusesUnknownOperation$", "-count=1", "-v"],
     "Admit exactly the bogus operation past the enum; the token authorizes and the negative fails by success."),
    ("N-presented-session", "edit", "internal/fencing/gate.go",
     [("if _, err := scalar.ParseUUIDv7(presented.SessionID); err != nil {",
        'if _, err := scalar.ParseUUIDv7(presented.SessionID); err != nil && presented.SessionID != "not-a-uuid" {')],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeRefusesMalformedPresented$", "-count=1", "-v"],
     "Admit exactly the not-a-uuid session; the refusal degrades to the foreign-session class."),
    ("N-presented-epoch", "edit", "internal/fencing/gate.go",
     [("if presented.Epoch == 0 {",
        'if presented.Epoch == 0 && presented.LeaseID != "' + LEASEA + '" {')],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeRefusesMalformedPresented$", "-count=1", "-v"],
     "Admit exactly the epoch-zero winner-lease vector; the refusal degrades to the stale class."),
    ("N-presented-lease", "edit", "internal/fencing/gate.go",
     [("if _, err := scalar.ParseUUIDv4(presented.LeaseID); err != nil {",
        'if _, err := scalar.ParseUUIDv4(presented.LeaseID); err != nil && presented.LeaseID != "zzz" {')],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeRefusesMalformedPresented$", "-count=1", "-v"],
     "Admit exactly the zzz token lease; the refusal degrades to the loser class."),
    ("N-winner", "edit", "internal/fencing/gate.go",
     [("if observation.Winner.Epoch == 0 || !validLeaseID(observation.Winner.LeaseID) || !validHostID(observation.Winner.HolderHostID) {",
        'if (observation.Winner.Epoch == 0 || !validLeaseID(observation.Winner.LeaseID) || !validHostID(observation.Winner.HolderHostID)) && !(observation.Winner.Epoch == 0 && observation.Winner.LeaseID == "zzz") {')],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeRefusesGarbageWinner$", "-count=1", "-v"],
     "Admit exactly the epoch-zero zzz winner; the refusal degrades to the ahead class."),
    ("N-session-prefix", "edit", "internal/fencing/gate.go",
     [(GATE_IMPORT, GATE_IMPORT_STRINGS),
      ("if presented.SessionID != observation.SessionID {",
        'if strings.SplitN(presented.SessionID, "-", 2)[0] != strings.SplitN(observation.SessionID, "-", 2)[0] {')],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeRefusesForeignSession$", "-count=1", "-v"],
     "Compare first UUID segments only; the prefix-sharing session authorizes while the fully foreign session still refuses."),
    ("N-absent", "edit", "internal/fencing/gate.go",
     [("if !observation.HasWinner {",
        "if !observation.HasWinner && !observation.Verified {")],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeParksAndRefusesAbsentWinner$", "-count=1", "-v"],
     "Admit exactly the verified absence; the refusal degrades to the malformed-observation class."),
    ("N-verified", "edit", "internal/fencing/gate.go",
     [("if !observation.Verified {",
        "if !observation.Verified && observation.Ambiguous {")],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeParksAndRefusesUnverified$", "-count=1", "-v"],
     "Admit exactly the unambiguous unverified observation; the token authorizes and the negative fails by success."),
    ("N-ambiguous", "edit", "internal/fencing/gate.go",
     [("if observation.Ambiguous {",
        "if observation.Ambiguous && !observation.Verified {")],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeParksAndRefusesAmbiguous$", "-count=1", "-v"],
     "Admit exactly the verified ambiguity; the token authorizes and the negative fails by success."),
    ("N-handoff", "edit", "internal/fencing/gate.go",
     [("if observation.HandoffFailed {",
        "if observation.HandoffFailed && !observation.Verified {")],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeParksAndRefusesFailedHandoff$", "-count=1", "-v"],
     "Admit exactly the verified failed handoff; the token authorizes and the negative fails by success."),
    ("N-grant", "edit", "internal/fencing/gate.go",
     [("if !observation.HasGrant {",
        "if !observation.HasGrant && !observation.Verified {")],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeRefusesMissingGrant$", "-count=1", "-v"],
     "Admit exactly the verified grantless call; the refusal degrades to the lapsed-grant class."),
    ("N-clock", "edit", "internal/fencing/gate.go",
     [("if observation.Now.IsZero() {",
        "if observation.Now.IsZero() && !observation.Grant.ValidatedAt.IsZero() {")],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeRefusesMissingClock$", "-count=1", "-v"],
     "Admit exactly the clockless never-validated grant; the token authorizes and the negative fails by success."),
    ("N-expiry", "edit", "internal/fencing/gate.go",
     [("sessrepo.CheckFencingExpiry(observation.Grant, observation.Policy, observation.Now)",
        "sessrepo.CheckFencingExpiry(observation.Grant, sessrepo.FencingPolicy{RefreshInterval: observation.Policy.RefreshInterval + time.Hour}, observation.Now)")],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeRefusesExpiredGrant$", "-count=1", "-v"],
     "Extend the refresh interval by an hour; the lapsed-by-seconds grant authorizes while the never-validated grant still refuses."),
    ("N-expiry-map", "edit", "internal/fencing/gate.go",
     [("if errors.Is(err, sessrepo.ErrFencingExpired) {",
        "if errors.Is(err, sessrepo.ErrInvalidLease) {")],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeRefusesExpiredGrant$", "-count=1", "-v"],
     "Swap the lapsed/unusable classes; both vectors change class and the test fails."),
    ("N-remote-prefix", "edit", "internal/fencing/gate.go",
     [(GATE_IMPORT, GATE_IMPORT_STRINGS),
      ("if observation.Winner.HolderHostID != observation.LocalHostID {",
        'if strings.SplitN(observation.Winner.HolderHostID, "-", 2)[0] != strings.SplitN(observation.LocalHostID, "-", 2)[0] {')],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeParksAndRefusesRemote$", "-count=1", "-v"],
     "Compare first UUID segments only; the prefix-sharing remote holder authorizes while the fully remote holder still refuses."),
    ("N-epoch-gte", "edit", "internal/fencing/gate.go",
     [("epochsEqual := presented.Epoch == observation.Winner.Epoch",
        "epochsEqual := presented.Epoch >= observation.Winner.Epoch")],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeRefusesEpochBeyondWinner$|^TestAuthorizeRefusesLowerEpoch$", "-count=1", "-v"],
     "Admit exactly the ahead epoch past the epoch arm; the lower-epoch vectors still refuse stale in the same run."),
    ("N-epoch-lte", "edit", "internal/fencing/gate.go",
     [("epochsEqual := presented.Epoch == observation.Winner.Epoch",
        "epochsEqual := presented.Epoch <= observation.Winner.Epoch")],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeRefusesEpochBeyondWinner$|^TestAuthorizeRefusesLowerEpoch$", "-count=1", "-v"],
     "Admit exactly the lower epoch past the epoch arm; the ahead-epoch vectors still refuse in the same run."),
    ("N-lease-vector", "edit", "internal/fencing/gate.go",
     [("if presented.LeaseID != observation.Winner.LeaseID {",
        'if presented.LeaseID != observation.Winner.LeaseID && presented.LeaseID != "' + LEASEC + '" {')],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeRefusesSameEpochLoser$", "-count=1", "-v"],
     "Admit exactly the standard loser; the prefix-sharing loser still refuses."),
    ("N-lease-prefix", "edit", "internal/fencing/gate.go",
     [(GATE_IMPORT, GATE_IMPORT_STRINGS),
      ("if presented.LeaseID != observation.Winner.LeaseID {",
        'if strings.SplitN(presented.LeaseID, "-", 2)[0] != strings.SplitN(observation.Winner.LeaseID, "-", 2)[0] {')],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeRefusesSameEpochLoser$", "-count=1", "-v"],
     "Compare first UUID segments only; the prefix-sharing loser authorizes while the standard loser still refuses."),
    ("N-launch-add-input", "edit", "internal/fencing/gate.go",
     [("return op == OperationActivation || op == OperationRestore",
        "return op == OperationActivation || op == OperationRestore || op == OperationInput")],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeParksAndRefusesRemote$", "-count=1", "-v"],
     "Park input instead of refusing: the remote input vector parks and the hard-refusal assertion fails."),
    ("N-launch-drop-restore", "edit", "internal/fencing/gate.go",
     [("return op == OperationActivation || op == OperationRestore",
        "return op == OperationActivation")],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeParksAndRefusesRemote$", "-count=1", "-v"],
     "Refuse restore instead of parking: the remote restore vector refuses hard and the park assertion fails."),
    ("N-bind-seal", "edit", "internal/fencing/token.go",
     [("if token.seal == nil || token.seal.token == nil {",
        "if token.seal == nil {")],
     ["go", "test", "./internal/fencing", "-run", r"^TestBindRefusesForgedToken$", "-count=1", "-v"],
     "Admit exactly the field-assembled seal; the zero token still refuses."),
    ("N-bind-op", "edit", "internal/fencing/token.go",
     [("case ProviderQuiesce, ProviderCapture, ProviderMaterialize:",
        'case ProviderQuiesce, ProviderCapture, ProviderMaterialize, "bogus":')],
     ["go", "test", "./internal/fencing", "-run", r"^TestBindRefusesUnknownOperation$", "-count=1", "-v"],
     "Admit exactly the bogus provider operation; the projection mints and the negative fails by success."),
    ("N-term-presented", "edit", "internal/fencing/terminate.go",
     [("if _, err := scalar.ParseUUIDv4(presented.LeaseID); err != nil {",
        'if _, err := scalar.ParseUUIDv4(presented.LeaseID); err != nil && presented.LeaseID != "zzz" {')],
     ["go", "test", "./internal/fencing", "-run", r"^TestTerminateStaleRefusesMalformedTarget$", "-count=1", "-v"],
     "Admit exactly the zzz target lease; the termination authorizes and the negative fails by success."),
    ("N-term-winner", "edit", "internal/fencing/terminate.go",
     [("if winner.Epoch == 0 || !validLeaseID(winner.LeaseID) {",
        'if (winner.Epoch == 0 || !validLeaseID(winner.LeaseID)) && !(winner.Epoch == 0 && winner.LeaseID == "zzz") {')],
     ["go", "test", "./internal/fencing", "-run", r"^TestTerminateStaleRefusesGarbageWinner$", "-count=1", "-v"],
     "Admit exactly the epoch-zero zzz winner; the termination authorizes and the negative fails by success."),
    ("N-term-absent", "edit", "internal/fencing/terminate.go",
     [("if !hasWinner {",
        "if !hasWinner && !forceRecovery {")],
     ["go", "test", "./internal/fencing", "-run", r"^TestTerminateStaleRefusesWithoutWinner$", "-count=1", "-v"],
     "Admit exactly the forced winnerless call; the refusal degrades to the malformed-observation class."),
    ("N-term-force", "edit", "internal/fencing/terminate.go",
     [("if !forceRecovery {",
        "if !forceRecovery && !diagnosticsPreserved {")],
     ["go", "test", "./internal/fencing", "-run", r"^TestTerminateStaleRefusesWithoutForce$", "-count=1", "-v"],
     "Admit exactly the unforced diagnosed call; the termination authorizes and the negative fails by success."),
    ("N-term-diag", "edit", "internal/fencing/terminate.go",
     [("if !diagnosticsPreserved {",
        "if !diagnosticsPreserved && !forceRecovery {")],
     ["go", "test", "./internal/fencing", "-run", r"^TestTerminateStaleRefusesWithoutDiagnostics$", "-count=1", "-v"],
     "Admit exactly the undiagnosed forced call; the termination authorizes and the negative fails by success."),
    ("N-term-session", "edit", "internal/fencing/terminate.go",
     [("if presented.SessionID != winner.SessionID {",
        'if presented.SessionID != winner.SessionID && presented.SessionID != "' + SESSIONB + '" {')],
     ["go", "test", "./internal/fencing", "-run", r"^TestTerminateStaleRefusesForeignSession$", "-count=1", "-v"],
     "Admit exactly the prefix-sharing foreign target; the refusal degrades to the live-owner class."),
    ("N-term-live", "edit", "internal/fencing/terminate.go",
     [("if presented.Epoch == winner.Epoch && presented.LeaseID == winner.LeaseID {",
        "if presented.Epoch == winner.Epoch && presented.LeaseID == winner.LeaseID && winner.Epoch != 1 {")],
     ["go", "test", "./internal/fencing", "-run", r"^TestTerminateStaleRefusesLiveOwner$", "-count=1", "-v"],
     "Admit exactly the epoch-1 live owner; the epoch-2 owner still refuses."),
    ("T-census-alias", "add", "internal/fencing/zzplant.go",
     "package fencing\n\ntype plantBackdoor = leaseSeal\n\nfunc plantAlias() LeaseToken { return LeaseToken{seal: &plantBackdoor{}} }\n",
     ["go", "test", "./internal/fencing", "-count=1", "-v"],
     "Preserve the leaseSeal token while constructing through an alias; the census rejects the plant and the full behavioral suite runs."),
    ("C-harmless-comment", "edit", "internal/fencing/gate.go",
     [("// Arm order is precedence: malformed calls die first, then foreign",
        "// Arm order is precedence: malformed calls die first, then foreign\n// classifier control: comment only, no behavior change.")],
     ["go", "test", "./internal/fencing", "-count=1"],
     "Applied control: a behavior-preserving plant survives through the same instrument."),
    ("C-not-applied", "edit", "internal/fencing/gate.go",
     [("TOKEN_THAT_IS_NOT_PRESENT", "replacement")],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeAdmitsWinnerExactEpochAllOperations$", "-count=1", "-v"],
     "Application control: missing patch is not a killed mutant."),
    ("C-compile-failure", "edit", "internal/fencing/gate.go",
     [("package fencing", "package fencing\nthis is invalid Go")],
     ["go", "test", "./internal/fencing", "-run", r"^TestAuthorizeAdmitsWinnerExactEpochAllOperations$", "-count=1", "-v"],
     "Compilation control: no named failed behavioral test is not a kill."),
]

selected = [row for row in MUTANTS if name_filter in row[0]] if name_filter else MUTANTS
if name_filter and not selected:
    raise SystemExit(f"no mutants match filter {name_filter!r}")


def run_command(label, command, log_name):
    with (output / log_name).open("w") as log:
        process = subprocess.run(command, cwd=copy, stdout=log, stderr=subprocess.STDOUT, timeout=300)
    return process.returncode


with (output / f"control-before-{tag}.log").open("w") as log:
    before = subprocess.run(["go", "test", "./internal/fencing", "-count=1"], cwd=copy, stdout=log, stderr=subprocess.STDOUT, timeout=300)
results.append(dict(mutant=f"control-before-{tag}", classification="CONTROL", exit=before.returncode))
if before.returncode:
    raise SystemExit("baseline failed")


def classify(name, command, bound):
    try:
        with (output / (name + ".log")).open("w") as log:
            process = subprocess.run(command, cwd=copy, stdout=log, stderr=subprocess.STDOUT, timeout=300)
        text = (output / (name + ".log")).read_text()
        failures = re.findall(r"^\s*--- FAIL: ([^ ]+)", text, re.M)
        ran_tests = re.findall(r"^=== RUN ", text, re.M)
        # Non-verbose full-suite runs report failures as "^--- FAIL:" and
        # "FAIL\tpackage" lines without === RUN markers; accept either.
        if not ran_tests:
            ran_tests = re.findall(r"^(ok|FAIL)\s", text, re.M)
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


for name, kind, path, payload, command, bound in selected:
    if kind == "edit":
        run_edit(name, path, payload, command, bound)
    else:
        run_add(name, path, payload, command, bound)

with (output / f"control-after-{tag}.log").open("w") as log:
    after = subprocess.run(["go", "test", "./internal/fencing", "-count=1"], cwd=copy, stdout=log, stderr=subprocess.STDOUT, timeout=300)
results.append(dict(mutant=f"control-after-{tag}", classification="CONTROL", exit=after.returncode))
(output / f"mutants-{tag}.json").write_text(json.dumps(results, indent=2) + "\n")
bad = [row for row in results if (row["mutant"].startswith("N-") or row["mutant"].startswith("T-")) and row["classification"] != "KILLED"]
raise SystemExit(1 if bad or after.returncode else 0)
