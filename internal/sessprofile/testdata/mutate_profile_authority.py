#!/usr/bin/env python3
"""Measure profile-source authority with append admission on and off.

Usage: PYTHONDONTWRITEBYTECODE=1 python3 internal/sessprofile/testdata/mutate_profile_authority.py /absolute/evidence/directory

The instrumented copy disables only sessrepo.checkWinningLease and reruns
the five committed derivation-path witnesses. Narrowing M1 then admits the
exact losing-lease event and must fail each entry witness alone. Narrowing M2
matches leases by epoch alone and must fail the same-epoch tuple witness.
Separate full-suite runs repeat M1 and M2 with the append gate left on, proving
ordinary CI catches both. Raw subprocess logs and exact exit codes are kept.
"""
from pathlib import Path
import json
import re
import shutil
import subprocess
import sys


ROOT = Path(__file__).resolve().parents[3]
OUTPUT = Path(sys.argv[1]).resolve()
INSTRUMENTED = OUTPUT / "mutation-source-instrumented"
GATE_ON = OUTPUT / "mutation-source-gate-on"
OUTPUT.mkdir(parents=True, exist_ok=True)


def copy_project(destination: Path) -> None:
    if destination.exists():
        raise SystemExit(f"refusing to overwrite existing mutation source: {destination}")
    shutil.copytree(
        ROOT,
        destination,
        ignore=shutil.ignore_patterns(".git", ".temp", ".task-board", "node_modules", "DerivedData"),
    )
    # The specpin suite derives its normative-scope assertion from Story
    # README files. Copy just those read-only inputs instead of the 1.3 GB
    # task-board tree, keeping the mutation source isolated and reproducible.
    board_source = ROOT / ".task-board"
    if board_source.is_dir():
        for story_readme in board_source.glob("EPIC-*/STORY-*/README.md"):
            relative = story_readme.relative_to(board_source)
            copied = destination / ".task-board" / relative
            copied.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(story_readme, copied)


def run_logged(name: str, command: list[str], cwd: Path, timeout: int = 540):
    path = OUTPUT / f"{name}.log"
    with path.open("w") as log:
        result = subprocess.run(command, cwd=cwd, stdout=log, stderr=subprocess.STDOUT, timeout=timeout)
    return result, path.read_text()


def failures(output: str) -> list[str]:
    return re.findall(r"^--- FAIL: (Test[^/\s]+)", output, re.M)


def assert_killed(name: str, result, output: str, expected: str) -> None:
    failed = failures(output)
    if result.returncode == 0 or expected not in failed:
        raise SystemExit(f"{name} survived or missed its named killer: exit={result.returncode}, failed={failed}")


copy_project(INSTRUMENTED)
copy_project(GATE_ON)

append_source = INSTRUMENTED / "internal/sessrepo/chain.go"
append_base = append_source.read_text()
append_function = '''func checkWinningLease(winner LeaseSummary, event eventView) error {
	if event.leaseEpoch < winner.Epoch {
		return refuse(ErrStaleLease, "event lease epoch %d precedes winning lease epoch %d", event.leaseEpoch, winner.Epoch)
	}
	if event.leaseEpoch == winner.Epoch && event.leaseID != winner.LeaseID {
		return refuse(ErrDivergentBranch, "event lease %s loses to winning lease %s at epoch %d", event.leaseID, winner.LeaseID, winner.Epoch)
	}
	return nil
}'''
if append_base.count(append_function) != 1:
    raise SystemExit("append gate instrument was not unique")
append_source.write_text(append_base.replace(append_function, "func checkWinningLease(winner LeaseSummary, event eventView) error {\n\treturn nil\n}"))

authority_source = INSTRUMENTED / "internal/sessprofile/authority.go"
authority_base = authority_source.read_text()
gate_on_authority = GATE_ON / "internal/sessprofile/authority.go"
gate_on_base = gate_on_authority.read_text()

LOWER_EPOCH_TESTS = {
    "LoadProfile": "TestLoadProfileDoesNotExposeLosingLeaseAsEffectiveSource",
    "Projector.Project": "TestProbe15DisabledAppendGateProjector",
    "Projector.ProjectForHeads": "TestProjectorForHeadsDoesNotExposeLosingLeaseAsEffectiveSource",
    "SetProfile.from_end": "TestSetProfileFromEndDoesNotReplayLosingLeaseChange",
    "axpane.deriveProfile": "TestAxpaneDeriveProfileDoesNotUseLosingLeaseSource",
}
SAME_EPOCH_TEST = "TestAxpaneDeriveProfileRejectsUnmintedSameEpochLeaseTuple"
ENTRY_TESTS = tuple(LOWER_EPOCH_TESTS.values()) + (SAME_EPOCH_TEST,)
ENTRY_RUN = "^(" + "|".join(ENTRY_TESTS) + ")$"
baseline_command = ["go", "test", "./internal/axpane", "-run", ENTRY_RUN, "-count=1", "-v"]
baseline, baseline_text = run_logged("instrumented-derivation-baseline", baseline_command, INSTRUMENTED)
ran = re.findall(r"^=== RUN   (Test[^/\s]+)", baseline_text, re.M)
skipped = re.findall(r"^--- SKIP: (Test[^/\s]+)", baseline_text, re.M)
failed = failures(baseline_text)
if baseline.returncode != 0 or set(ran) != set(ENTRY_TESTS) or skipped or failed:
    raise SystemExit(f"instrumented entry suite failed: exit={baseline.returncode}, ran={ran}, skipped={skipped}, failed={failed}")

event_ids = set(re.findall(r"gate-on losing profile event id=(sha256:[0-9a-f]{64})", baseline_text))
if len(event_ids) != 1:
    raise SystemExit(f"gate-on losing-tail fixture did not emit one stable event id: {sorted(event_ids)}")
losing_event_id = next(iter(event_ids))

authority_marker = "func authorizedSource(authority SourceAuthority, event Event, requestedClosure map[string]bool, handoffClosure map[string]bool, explicitClosure bool) bool {\n"
if authority_base.count(authority_marker) != 1:
    raise SystemExit("M1 source-authority narrowing plant was not unique")
m1_source = authority_base.replace(
    authority_marker,
    authority_marker + f'\tif event.ID == "{losing_event_id}" {{\n\t\treturn true\n\t}}\n',
)
rows = [
    {"mutant": "instrumented-derivation-baseline", "classification": "CONTROL", "exit": baseline.returncode, "command": baseline_command, "tests": ran, "append_gate": "disabled as an isolated instrument"},
]

authority_source.write_text(m1_source)
for entry, test_name in LOWER_EPOCH_TESTS.items():
    command = ["go", "test", "./internal/axpane", "-run", f"^{test_name}$", "-count=1", "-v"]
    name = f"M1-instrumented-{entry.replace('.', '-')}"
    result, output = run_logged(name, command, INSTRUMENTED)
    assert_killed(name, result, output, test_name)
    rows.append({"mutant": "M1-exact-losing-event", "classification": "KILLED", "append_gate": "disabled as an isolated instrument", "entry": entry, "test": test_name, "exit": result.returncode, "command": command, "failed_tests": failures(output), "event_id": losing_event_id})

same_epoch_marker = "if event.LeaseEpoch == lease.Epoch && event.LeaseID == lease.LeaseID {"
if authority_base.count(same_epoch_marker) != 1:
    raise SystemExit("M2 exact-lease-tuple narrowing plant was not unique")
m2_source = authority_base.replace(same_epoch_marker, "if event.LeaseEpoch == lease.Epoch {")
authority_source.write_text(m2_source)
m2_command = ["go", "test", "./internal/axpane", "-run", f"^{SAME_EPOCH_TEST}$", "-count=1", "-v"]
m2_result, m2_output = run_logged("M2-instrumented-same-epoch-tuple", m2_command, INSTRUMENTED)
assert_killed("M2-instrumented-same-epoch-tuple", m2_result, m2_output, SAME_EPOCH_TEST)
rows.append({"mutant": "M2-epoch-only-known-lease", "classification": "KILLED", "append_gate": "disabled as an isolated instrument", "entry": "axpane.deriveProfile", "test": SAME_EPOCH_TEST, "exit": m2_result.returncode, "command": m2_command, "failed_tests": failures(m2_output)})

authority_source.write_text(authority_base)
control_old = "// SourceAuthority carries the durable lease facts used when selecting a"
control_new = "// SourceAuthority carries the durable lease facts used when selecting a\n// Applied harmless control: no behavior changes."
if authority_base.count(control_old) != 1:
    raise SystemExit("harmless control plant was not unique")
authority_source.write_text(authority_base.replace(control_old, control_new))
control_command = ["go", "test", "./internal/axpane", "-run", ENTRY_RUN, "-count=1", "-v"]
control, control_text = run_logged("C-harmless-comment", control_command, INSTRUMENTED)
if control.returncode != 0 or failures(control_text):
    raise SystemExit(f"harmless control failed instead of surviving: exit={control.returncode}, failed={failures(control_text)}")
rows.append({"mutant": "C-harmless-comment", "classification": "SURVIVED", "exit": control.returncode, "command": control_command, "tests": re.findall(r"^=== RUN   (Test[^/\s]+)", control_text, re.M)})
authority_source.write_text(authority_base)
if append_source.read_text() != append_base:
    append_source.write_text(append_base)
if append_source.read_text() != append_base:
    raise SystemExit("instrumented append source was not restored")

# Repeat each gate in a plain copy with sessrepo.checkWinningLease intact.
m1_plain_source = gate_on_base.replace(
    authority_marker,
    authority_marker + f'\tif event.ID == "{losing_event_id}" {{\n\t\treturn true\n\t}}\n',
)
if m1_plain_source == gate_on_base:
    raise SystemExit("M1 plain-suite source mutation did not apply")
gate_on_authority.write_text(m1_plain_source)
for entry, test_name in LOWER_EPOCH_TESTS.items():
    command = ["go", "test", "./internal/axpane", "-run", f"^{test_name}$", "-count=1", "-v"]
    name = f"M1-gate-on-{entry.replace('.', '-')}"
    result, output = run_logged(name, command, GATE_ON)
    assert_killed(name, result, output, test_name)
    rows.append({"mutant": "M1-exact-losing-event", "classification": "KILLED", "append_gate": "enabled", "entry": entry, "test": test_name, "exit": result.returncode, "command": command, "failed_tests": failures(output), "event_id": losing_event_id})

plain_all_command = ["go", "test", "./...", "-count=1"]
plain_m1, plain_m1_text = run_logged("M1-gate-on-go-test-all", plain_all_command, GATE_ON)
plain_m1_failures = failures(plain_m1_text)
expected_m1_failures = set(LOWER_EPOCH_TESTS.values())
if plain_m1.returncode == 0 or set(plain_m1_failures) != expected_m1_failures:
    raise SystemExit(f"plain go test ./... did not fail only the five named M1 witnesses: exit={plain_m1.returncode}, failed={plain_m1_failures}")
rows.append({"mutant": "M1-exact-losing-event", "classification": "KILLED", "append_gate": "enabled", "entry": "all five derivation paths", "exit": plain_m1.returncode, "command": plain_all_command, "failed_tests": plain_m1_failures})

gate_on_authority.write_text(gate_on_base)
if gate_on_base.count(same_epoch_marker) != 1:
    raise SystemExit("M2 plain-suite source anchor was not unique")
gate_on_authority.write_text(gate_on_base.replace(same_epoch_marker, "if event.LeaseEpoch == lease.Epoch {"))
plain_m2_killer_command = ["go", "test", "./internal/axpane", "-run", f"^{SAME_EPOCH_TEST}$", "-count=1", "-v"]
plain_m2_killer, plain_m2_killer_text = run_logged("M2-gate-on-named-killer", plain_m2_killer_command, GATE_ON)
assert_killed("M2-gate-on-named-killer", plain_m2_killer, plain_m2_killer_text, SAME_EPOCH_TEST)
rows.append({"mutant": "M2-epoch-only-known-lease", "classification": "KILLED", "append_gate": "enabled", "entry": "axpane.deriveProfile", "test": SAME_EPOCH_TEST, "exit": plain_m2_killer.returncode, "command": plain_m2_killer_command, "failed_tests": failures(plain_m2_killer_text)})

plain_m2, plain_m2_text = run_logged("M2-gate-on-go-test-all", plain_all_command, GATE_ON)
plain_m2_failures = failures(plain_m2_text)
if plain_m2.returncode == 0 or set(plain_m2_failures) != {SAME_EPOCH_TEST}:
    raise SystemExit(f"plain go test ./... did not fail only the named M2 witness: exit={plain_m2.returncode}, failed={plain_m2_failures}")
rows.append({"mutant": "M2-epoch-only-known-lease", "classification": "KILLED", "append_gate": "enabled", "entry": "axpane.deriveProfile", "exit": plain_m2.returncode, "command": plain_all_command, "failed_tests": plain_m2_failures})
gate_on_authority.write_text(gate_on_base)

(OUTPUT / "mutants.json").write_text(json.dumps(rows, indent=2) + "\n")
print("instrumented derivation baseline CONTROL", baseline.returncode)
print("M1 exact losing-event narrowing KILLED", sum(row["mutant"] == "M1-exact-losing-event" for row in rows), "including every required derivation path and the gate-on full suite")
print("M2 epoch-only known-lease narrowing KILLED", sum(row["mutant"] == "M2-epoch-only-known-lease" for row in rows), "including the gate-on full suite")
print("C-harmless-comment SURVIVED", control.returncode)
