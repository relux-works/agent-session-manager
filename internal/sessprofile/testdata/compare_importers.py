#!/usr/bin/env python3
"""Compare profile-source outcomes over sessprofile's complete importer set.

The comparator builds the exact HEAD tree and an isolated copy of the current
candidate, disables only sessrepo.checkWinningLease in both copies, adds the
same runtime outcome probe, and records outcomes keyed by (package, entry,
input). The generated test drives production entries; it is preserved with the
raw logs and JSON report under the requested evidence directory.

Usage:
  PYTHONDONTWRITEBYTECODE=1 python3 internal/sessprofile/testdata/compare_importers.py /absolute/evidence/directory
"""
from __future__ import annotations

import hashlib
import json
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tarfile


ROOT = Path(__file__).resolve().parents[3]
OUTPUT = Path(sys.argv[1]).resolve()
MODULE = "github.com/relux-works/agent-session-manager/internal/sessprofile"
PROBE_PACKAGE = "internal/axpane"
APPEND_FUNCTION = re.compile(r"(?ms)^func checkWinningLease\(.*?^}")


def run(command: list[str], cwd: Path, log: Path, timeout: int = 600) -> int:
    with log.open("w") as stream:
        result = subprocess.run(command, cwd=cwd, stdout=stream, stderr=subprocess.STDOUT, timeout=timeout)
    return result.returncode


def git_archive(destination: Path) -> str:
    result = subprocess.run(["git", "rev-parse", "HEAD"], cwd=ROOT, check=True, capture_output=True, text=True)
    base_commit = result.stdout.strip()
    destination.mkdir(parents=True)
    archive = subprocess.Popen(["git", "archive", "--format=tar", "HEAD"], cwd=ROOT, stdout=subprocess.PIPE)
    if archive.stdout is None:
        raise RuntimeError("git archive did not provide a stream")
    with tarfile.open(fileobj=archive.stdout, mode="r|") as tar:
        # The checked-in board export can contain large historical evidence
        # blobs; it is not part of any Go package or production source.
        for member in tar:
            if member.name == ".task-board" or member.name.startswith(".task-board/"):
                continue
            tar.extract(member, destination, filter="data")
    if archive.wait() != 0:
        raise RuntimeError("git archive failed while streaming baseline")
    return base_commit


def copy_candidate(destination: Path) -> None:
    ignored = shutil.ignore_patterns(".git", ".temp", ".task-board", ".planning", "DerivedData", "node_modules", ".DS_Store")
    shutil.copytree(ROOT, destination, ignore=ignored)


def tree_digest(root: Path) -> str:
    digest = hashlib.sha256()
    for path in sorted(p for p in root.rglob("*") if p.is_file() and "outcome_probe_test.go" not in p.name):
        relative = path.relative_to(root).as_posix().encode()
        digest.update(len(relative).to_bytes(8, "big"))
        digest.update(relative)
        raw = path.read_bytes()
        digest.update(len(raw).to_bytes(8, "big"))
        digest.update(raw)
    return digest.hexdigest()


def importer_rows(root: Path, output: Path) -> tuple[list[str], list[dict[str, str]], int]:
    command = ["go", "list", "-test", "-f", '{{.ImportPath}}|{{join .Imports ","}}', "./..."]
    log = output / (root.name + "-importers.log")
    with log.open("w") as stream:
        result = subprocess.run(command, cwd=root, stdout=stream, stderr=subprocess.STDOUT, timeout=600)
    if result.returncode != 0:
        raise RuntimeError(f"go list importer census failed in {root.name}: exit={result.returncode}; see {log}")
    rows: list[dict[str, str]] = []
    packages: set[str] = set()
    for line in log.read_text().splitlines():
        package, separator, imports = line.partition("|")
        if separator and MODULE in imports.split(","):
            rows.append({"go_list_package": package, "imports_sessprofile": "true"})
            packages.add(package.split(" [", 1)[0])
    return sorted(packages), rows, result.returncode


PROBE_TEMPLATE = r'''package axpane

import (
    "encoding/json"
    "testing"

    "github.com/relux-works/agent-session-manager/internal/sessprofile"
    "github.com/relux-works/agent-session-manager/internal/sessckpt"
    "github.com/relux-works/agent-session-manager/internal/sessrepo"
)

func emitProfileSourceOutcome(t *testing.T, entry, input string, pair sessprofile.Pair, err error) {
    t.Helper()
    message := ""
    if err != nil { message = err.Error() }
    raw, marshalErr := json.Marshal(map[string]any{
        "package": "internal/axpane", "entry": entry, "input": input,
        "outcome": map[string]any{"profile": pair.Profile, "source": pair.Source, "has_source": pair.HasSource, "error": message},
    })
    if marshalErr != nil { t.Fatal(marshalErr) }
    t.Log("profile-source-outcome " + string(raw))
}

func emitSetProfileOutcome(t *testing.T, input string, result sessprofile.SetProfileResult, err error) {
    t.Helper()
    message := ""
    if err != nil { message = err.Error() }
    raw, marshalErr := json.Marshal(map[string]any{
        "package": "internal/axpane", "entry": "Transactor.SetProfile.from_end", "input": input,
        "outcome": map[string]any{"previous_profile": result.PreviousProfile, "new_profile": result.NewProfile, "event_id": result.EventID, "error": message},
    })
    if marshalErr != nil { t.Fatal(marshalErr) }
    t.Log("profile-source-outcome " + string(raw))
}

func sourceProbeLoadPair(world *runWorld) (sessprofile.Pair, error) {
    __LOAD_PAIR__
}

func sourceProbeProjectorPair(world *runWorld) (sessprofile.Pair, error) {
    __PROJECTOR_PAIR__
}

func sourceProbeProjectorForHeadsPair(world *runWorld) (sessprofile.Pair, error) {
    summaries, err := world.repo.ListEvents(fixtureSession)
    if err != nil { return sessprofile.Pair{}, err }
    head := ""
    for _, summary := range summaries {
        raw, err := world.repo.GetEvent(fixtureSession, summary.EventID)
        if err != nil { return sessprofile.Pair{}, err }
        event, err := sessprofile.DecodeEvent(raw)
        if err != nil { return sessprofile.Pair{}, err }
        if event.Type == "profile.changed" { head = event.ID }
    }
    if head == "" { return sessprofile.Pair{}, sessprofile.ErrInvalidEvent }
    __PROJECTOR_HEADS_PAIR__
}

func sourceProbeDeriveProfilePair(world *runWorld) (sessprofile.Pair, error) {
    __DERIVE_PROFILE_PAIR__
}

func measureProfileSourcePair(t *testing.T, entry, input string, makeWorld func(*testing.T) *runWorld, derive func(*runWorld) (sessprofile.Pair, error)) {
    t.Helper()
    world := makeWorld(t)
    pair, err := derive(world)
    emitProfileSourceOutcome(t, entry, input, pair, err)
}

func sourceProbeLosingWorld(t *testing.T) *runWorld {
    t.Helper()
    world := buildRunWorld(t)
    leases, err := world.repo.ListLeases(fixtureSession)
    if err != nil || len(leases) != 1 { t.Fatalf("ListLeases() = %v, %v; want epoch-1 lease", leases, err) }
    if _, err := world.repo.CompareAndSwapLease(fixtureSession, sessrepo.LeaseExpectation{RecordID: leases[0].RecordID}, sessrepo.SuccessorLeaseInput{
        CreateLeaseInput: sessrepo.CreateLeaseInput{LeaseID: "cccccccc-dddd-4eee-8fff-000000000002", HolderHostID: fixtureLocalHost, IssuedByHostID: fixtureLocalHost, CreatedAt: fixtureCreatedAt},
        Reason: "graceful_takeover", CheckpointID: world.ckptID,
    }); err != nil { t.Fatalf("CompareAndSwapLease() error = %v", err) }
    appendChainEvent(t, world.repo, "profile.changed", 1, fixtureLeaseA, 2, world.headID, map[string]any{"from": "standard", "to": "yolo", "confirmed": true})
    return world
}

func sourceProbeHigherEpochWorld(t *testing.T) *runWorld {
    t.Helper()
    world := buildRunWorld(t)
    appendChainEvent(t, world.repo, "profile.changed", 2, fixtureLeaseB, 1, world.headID, map[string]any{"from": "standard", "to": "yolo", "confirmed": true})
    return world
}

func sourceProbeEmptyLeaseWorld(t *testing.T) *runWorld {
    t.Helper()
    repository, err := sessrepo.Open(t.TempDir())
    if err != nil { t.Fatal(err) }
    var record map[string]any
    if err := json.Unmarshal([]byte(sessionRecordFixture), &record); err != nil { t.Fatal(err) }
    reference, err := repository.CreateSession(identifyObject(t, record, "record_id"))
    if err != nil { t.Fatalf("CreateSession() error = %v", err) }
    appendChainEvent(t, repository, "profile.changed", 1, fixtureLeaseA, 1, reference.RecordID, map[string]any{"from": "standard", "to": "yolo", "confirmed": true})
    return &runWorld{repo: repository}
}

func sourceProbeAmbiguousTupleWorld(t *testing.T) *runWorld {
    t.Helper()
    repository, err := sessrepo.Open(t.TempDir())
    if err != nil { t.Fatal(err) }
    var record map[string]any
    if err := json.Unmarshal([]byte(sessionRecordFixture), &record); err != nil { t.Fatal(err) }
    reference, err := repository.CreateSession(identifyObject(t, record, "record_id"))
    if err != nil { t.Fatalf("CreateSession() error = %v", err) }
    if _, err := repository.CreateLease(fixtureSession, sessrepo.CreateLeaseInput{LeaseID: fixtureLeaseA, HolderHostID: fixtureLocalHost, IssuedByHostID: fixtureLocalHost, CreatedAt: fixtureCreatedAt}); err != nil { t.Fatalf("CreateLease() error = %v", err) }
    leases, err := repository.ListLeases(fixtureSession)
    if err != nil || len(leases) != 1 { t.Fatalf("ListLeases() = %v, %v; want epoch-1 lease", leases, err) }
    if _, err := repository.CompareAndSwapLease(fixtureSession, sessrepo.LeaseExpectation{RecordID: leases[0].RecordID}, sessrepo.SuccessorLeaseInput{
        CreateLeaseInput: sessrepo.CreateLeaseInput{LeaseID: fixtureLeaseB, HolderHostID: fixtureLocalHost, IssuedByHostID: fixtureLocalHost, CreatedAt: fixtureCreatedAt},
        Reason: "graceful_takeover", CheckpointID: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
    }); err != nil { t.Fatalf("CompareAndSwapLease() error = %v", err) }
    appendChainEvent(t, repository, "profile.changed", 2, fixtureLeaseA, 1, reference.RecordID, map[string]any{"from": "standard", "to": "yolo", "confirmed": true})
    return &runWorld{repo: repository}
}

func sourceProbeWinningHandoffWorld(t *testing.T) *runWorld {
    t.Helper()
    world := buildRunWorld(t)
    changed := appendChainEvent(t, world.repo, "profile.changed", 1, fixtureLeaseA, 2, world.headID, map[string]any{"from": "standard", "to": "yolo", "confirmed": true})
    checkpoint, _, err := world.ckpt.Capture(world.repo, sessckpt.Inputs{
        OperationID: fixtureOtherOp, SessionID: fixtureSession, SessionKind: sessckpt.SessionKindDirect,
        LeaseEpoch: 1, LeaseID: fixtureLeaseA, CreatorHostID: fixtureLocalHost,
        WorkspaceManifestID: seedDigest(0xE1), ProviderManifestID: seedDigest(0xE2),
        Boundary: sessckpt.SafeBoundary{ProviderID: "codex", ProviderVersion: "0.147.0", Evidence: sessckpt.EvidenceAcceptedTest,
            InputBlocked: true, ForegroundIdle: true, BackgroundIdle: true, OpenProcesses: 0, OpenDatabaseHandles: 0},
        EventHeads: []string{changed}, CreatedAt: fixtureCreatedAt, Extensions: map[string]string{},
    })
    if err != nil { t.Fatalf("Capture(winning handoff) error = %v", err) }
    checkpointID := checkpoint.CheckpointID
    leases, err := world.repo.ListLeases(fixtureSession)
    if err != nil || len(leases) != 1 { t.Fatalf("ListLeases() = %v, %v; want epoch-1 lease", leases, err) }
    if _, err := world.repo.CompareAndSwapLease(fixtureSession, sessrepo.LeaseExpectation{RecordID: leases[0].RecordID}, sessrepo.SuccessorLeaseInput{
        CreateLeaseInput: sessrepo.CreateLeaseInput{LeaseID: "cccccccc-dddd-4eee-8fff-000000000002", HolderHostID: fixtureLocalHost, IssuedByHostID: fixtureLocalHost, CreatedAt: fixtureCreatedAt},
        Reason: "graceful_takeover", CheckpointID: checkpointID,
    }); err != nil { t.Fatalf("CompareAndSwapLease() error = %v", err) }
    return world
}

func measureSetProfileOutcome(t *testing.T, input string, world *runWorld, request sessprofile.SetProfileRequest) {
    t.Helper()
    result, err := (&sessprofile.Transactor{__TRANSACTOR_FIELDS__}).SetProfile(request)
    emitSetProfileOutcome(t, input, result, err)
}

func TestProfileSourceOutcomeGrid(t *testing.T) {
    measureProfileSourcePair(t, "LoadProfile", "post_takeover_lower_epoch", sourceProbeLosingWorld, sourceProbeLoadPair)
    measureProfileSourcePair(t, "Projector.Project", "post_takeover_lower_epoch", sourceProbeLosingWorld, sourceProbeProjectorPair)
    measureProfileSourcePair(t, "Projector.ProjectForHeads", "post_takeover_lower_epoch", sourceProbeLosingWorld, sourceProbeProjectorForHeadsPair)
    measureProfileSourcePair(t, "axpane.deriveProfile", "post_takeover_lower_epoch", sourceProbeLosingWorld, sourceProbeDeriveProfilePair)
    measureProfileSourcePair(t, "LoadProfile", "unminted_higher_epoch", sourceProbeHigherEpochWorld, sourceProbeLoadPair)
    measureProfileSourcePair(t, "Projector.Project", "unminted_higher_epoch", sourceProbeHigherEpochWorld, sourceProbeProjectorPair)
    measureProfileSourcePair(t, "Projector.ProjectForHeads", "unminted_higher_epoch", sourceProbeHigherEpochWorld, sourceProbeProjectorForHeadsPair)
    measureProfileSourcePair(t, "axpane.deriveProfile", "unminted_higher_epoch", sourceProbeHigherEpochWorld, sourceProbeDeriveProfilePair)
    measureProfileSourcePair(t, "LoadProfile", "empty_lease_store", sourceProbeEmptyLeaseWorld, sourceProbeLoadPair)
    measureProfileSourcePair(t, "Projector.Project", "empty_lease_store", sourceProbeEmptyLeaseWorld, sourceProbeProjectorPair)
    measureProfileSourcePair(t, "Projector.ProjectForHeads", "empty_lease_store", sourceProbeEmptyLeaseWorld, sourceProbeProjectorForHeadsPair)
    measureProfileSourcePair(t, "axpane.deriveProfile", "empty_lease_store", sourceProbeEmptyLeaseWorld, sourceProbeDeriveProfilePair)
    measureProfileSourcePair(t, "LoadProfile", "unminted_same_epoch_loser", sourceProbeAmbiguousTupleWorld, sourceProbeLoadPair)
    measureProfileSourcePair(t, "Projector.Project", "unminted_same_epoch_loser", sourceProbeAmbiguousTupleWorld, sourceProbeProjectorPair)
    measureProfileSourcePair(t, "Projector.ProjectForHeads", "unminted_same_epoch_loser", sourceProbeAmbiguousTupleWorld, sourceProbeProjectorForHeadsPair)
    measureProfileSourcePair(t, "axpane.deriveProfile", "unminted_same_epoch_loser", sourceProbeAmbiguousTupleWorld, sourceProbeDeriveProfilePair)
    measureProfileSourcePair(t, "Projector.Project", "winning_handoff_contains_prior_source", sourceProbeWinningHandoffWorld, sourceProbeProjectorPair)
    measureProfileSourcePair(t, "Projector.ProjectForHeads", "winning_handoff_contains_prior_source", sourceProbeWinningHandoffWorld, sourceProbeProjectorForHeadsPair)

    losing := sourceProbeLosingWorld(t)
    measureSetProfileOutcome(t, "post_takeover_lower_epoch", losing, sessprofile.SetProfileRequest{
        SessionID: fixtureSession, To: sessprofile.ProfileYOLO, Confirmed: true, LeaseEpoch: 1, LeaseID: fixtureLeaseA,
        CreatedByHostID: fixtureLocalHost, CreatedAt: fixtureCreatedAt,
    })
    ambiguous := sourceProbeAmbiguousTupleWorld(t)
    measureSetProfileOutcome(t, "unminted_same_epoch_loser", ambiguous, sessprofile.SetProfileRequest{
        SessionID: fixtureSession, To: sessprofile.ProfileYOLO, Confirmed: true, LeaseEpoch: 2, LeaseID: fixtureLeaseA,
        CreatedByHostID: fixtureLocalHost, CreatedAt: fixtureCreatedAt,
    })
    current := buildRunWorld(t)
    measureSetProfileOutcome(t, "winning_lease_standard_to_yolo", current, sessprofile.SetProfileRequest{
        SessionID: fixtureSession, To: sessprofile.ProfileYOLO, Confirmed: true, LeaseEpoch: 1, LeaseID: fixtureLeaseA,
        CreatedByHostID: fixtureLocalHost, CreatedAt: fixtureCreatedAt,
    })
}
'''


def probe_source(candidate: bool) -> str:
    if candidate:
        replacements = {
            "__LOAD_PAIR__": "derivation, err := LoadProfile(world.repo, world.ckpt, fixtureSession)\n    if err != nil { return sessprofile.Pair{}, err }\n    return derivation.Derive()",
            "__PROJECTOR_PAIR__": "return (&sessprofile.Projector{Repo: world.repo, Ckpt: world.ckpt}).Project(fixtureSession)",
            "__PROJECTOR_HEADS_PAIR__": "return (&sessprofile.Projector{Repo: world.repo, Ckpt: world.ckpt}).ProjectForHeads(fixtureSession, []string{head})",
            "__DERIVE_PROFILE_PAIR__": "derivation, err := LoadProfile(world.repo, world.ckpt, fixtureSession)\n    if err != nil { return sessprofile.Pair{}, err }\n    return deriveProfile(Input{ProfileData: derivation})",
            "__TRANSACTOR_FIELDS__": "Repo: world.repo, Ckpt: world.ckpt",
        }
    else:
        replacements = {
            "__LOAD_PAIR__": "record, events, err := LoadProfile(world.repo, fixtureSession)\n    if err != nil { return sessprofile.Pair{}, err }\n    return sessprofile.Derive(record, events)",
            "__PROJECTOR_PAIR__": "return (&sessprofile.Projector{Repo: world.repo}).Project(fixtureSession)",
            "__PROJECTOR_HEADS_PAIR__": "return (&sessprofile.Projector{Repo: world.repo}).ProjectForHeads(fixtureSession, []string{head})",
            "__DERIVE_PROFILE_PAIR__": "record, events, err := LoadProfile(world.repo, fixtureSession)\n    if err != nil { return sessprofile.Pair{}, err }\n    return deriveProfile(Input{ProfileRecord: record, ProfileEvents: events})",
            "__TRANSACTOR_FIELDS__": "Repo: world.repo",
        }
    source = PROBE_TEMPLATE
    for placeholder, replacement in replacements.items():
        if source.count(placeholder) != 1:
            raise RuntimeError(f"probe template placeholder {placeholder} is not unique")
        source = source.replace(placeholder, replacement)
    if "__" in source:
        raise RuntimeError("unexpanded probe template placeholder")
    return source


def instrument_append_gate(root: Path) -> None:
    source_path = root / "internal/sessrepo/chain.go"
    source = source_path.read_text()
    matches = list(APPEND_FUNCTION.finditer(source))
    if len(matches) != 1:
        raise RuntimeError(f"expected one checkWinningLease in {source_path}, found {len(matches)}")
    original = matches[0].group(0)
    signature = original.split("\n", 1)[0]
    source_path.write_text(source[:matches[0].start()] + signature + "\n\treturn nil\n}" + source[matches[0].end():])


def parse_outcomes(log: Path) -> list[dict[str, object]]:
    rows: list[dict[str, object]] = []
    for line in log.read_text().splitlines():
        match = re.search(r"profile-source-outcome (\{.*\})$", line)
        if match:
            rows.append(json.loads(match.group(1)))
    keys = [(row["package"], row["entry"], row["input"]) for row in rows]
    if len(keys) != len(set(keys)):
        raise RuntimeError(f"duplicate runtime outcome key in {log}")
    return sorted(rows, key=lambda row: (str(row["package"]), str(row["entry"]), str(row["input"])))


def main() -> int:
    if len(sys.argv) != 2:
        raise SystemExit("usage: compare_importers.py /absolute/evidence/directory")
    if OUTPUT.exists():
        raise SystemExit(f"refusing to overwrite existing outcome evidence: {OUTPUT}")
    OUTPUT.mkdir(parents=True)
    copies = OUTPUT / "copies"
    baseline = copies / "baseline"
    candidate = copies / "candidate"
    copies.mkdir()
    base_commit = git_archive(baseline)
    copy_candidate(candidate)

    before_importers, before_rows, before_list_exit = importer_rows(baseline, OUTPUT)
    after_importers, after_rows, after_list_exit = importer_rows(candidate, OUTPUT)
    if before_importers != after_importers:
        raise RuntimeError(f"sessprofile importer set changed: before={before_importers}, after={after_importers}")
    if not before_importers:
        raise RuntimeError("no direct Go importer of internal/sessprofile was found")

    probe = probe_source(candidate=False)
    (OUTPUT / "outcome_probe_test.go").write_text(probe)
    (baseline / "internal/axpane/outcome_probe_test.go").write_text(probe)
    (candidate / "internal/axpane/outcome_probe_test.go").write_text(probe_source(candidate=True))
    for root in (baseline, candidate):
        instrument_append_gate(root)

    command = ["go", "test", "./internal/axpane", "-run", "^TestProfileSourceOutcomeGrid$", "-count=1", "-v"]
    before_exit = run(command, baseline, OUTPUT / "outcomes-before.log")
    after_exit = run(command, candidate, OUTPUT / "outcomes-after.log")
    if before_exit != 0 or after_exit != 0:
        raise RuntimeError(f"outcome probe failed: before={before_exit}, after={after_exit}; see raw logs")
    before = parse_outcomes(OUTPUT / "outcomes-before.log")
    after = parse_outcomes(OUTPUT / "outcomes-after.log")
    before_map = {(row["package"], row["entry"], row["input"]): row["outcome"] for row in before}
    after_map = {(row["package"], row["entry"], row["input"]): row["outcome"] for row in after}
    if before_map.keys() != after_map.keys():
        raise RuntimeError(f"outcome keys changed: before={sorted(before_map)}, after={sorted(after_map)}")
    moved = [
        {"package": key[0], "entry": key[1], "input": key[2], "before": before_map[key], "after": after_map[key]}
        for key in sorted(before_map)
        if before_map[key] != after_map[key]
    ]
    expected_classes = {"post_takeover_lower_epoch", "unminted_higher_epoch", "empty_lease_store", "unminted_same_epoch_loser"}
    moved_classes = {str(row["input"]) for row in moved}
    if not expected_classes.issubset(moved_classes):
        raise RuntimeError(f"expected profile-source moves are absent: expected={sorted(expected_classes)}, moved={sorted(moved_classes)}")
    expected_keys = {
        (PROBE_PACKAGE, entry, input_name)
        for input_name in ("post_takeover_lower_epoch", "unminted_higher_epoch", "empty_lease_store", "unminted_same_epoch_loser")
        for entry in ("LoadProfile", "Projector.Project", "Projector.ProjectForHeads", "axpane.deriveProfile")
    }
    expected_keys.update({
        (PROBE_PACKAGE, "Projector.Project", "winning_handoff_contains_prior_source"),
        (PROBE_PACKAGE, "Projector.ProjectForHeads", "winning_handoff_contains_prior_source"),
        (PROBE_PACKAGE, "Transactor.SetProfile.from_end", "post_takeover_lower_epoch"),
        (PROBE_PACKAGE, "Transactor.SetProfile.from_end", "unminted_same_epoch_loser"),
        (PROBE_PACKAGE, "Transactor.SetProfile.from_end", "winning_lease_standard_to_yolo"),
    })
    if set(before_map) != expected_keys:
        raise RuntimeError(f"runtime probe does not cover the exact entry/input census: got={sorted(before_map)}, want={sorted(expected_keys)}")
    expected_moved = {
        key for key in expected_keys
        if key[2] in expected_classes or key == (PROBE_PACKAGE, "Transactor.SetProfile.from_end", "post_takeover_lower_epoch")
    }
    if {tuple(row[field] for field in ("package", "entry", "input")) for row in moved} != expected_moved:
        raise RuntimeError("runtime outcomes moved outside or short of the exact expected derivation and SetProfile keys")
    after_by_key = {(row["package"], row["entry"], row["input"]): row["outcome"] for row in after}
    for key in (
        (PROBE_PACKAGE, "Transactor.SetProfile.from_end", "post_takeover_lower_epoch"),
        (PROBE_PACKAGE, "Transactor.SetProfile.from_end", "unminted_same_epoch_loser"),
        (PROBE_PACKAGE, "Transactor.SetProfile.from_end", "winning_lease_standard_to_yolo"),
    ):
        outcome = after_by_key[key]
        if outcome.get("error") or outcome.get("previous_profile") != "standard" or outcome.get("new_profile") != "yolo":
            raise RuntimeError(f"composing SetProfile writer did not complete from the expected effective source at {key}: {outcome}")

    report = {
        "base_commit": base_commit,
        "base_tree_sha256": tree_digest(baseline),
        "candidate_tree_sha256": tree_digest(candidate),
        "importer_set_command": ["go", "list", "-test", "-f", '{{.ImportPath}}|{{join .Imports ","}}', "./..."],
        "importer_package_set": before_importers,
        "importer_go_list_rows_before": before_rows,
        "importer_go_list_rows_after": after_rows,
        "go_list_exit_codes": {"before": before_list_exit, "after": after_list_exit},
        "append_gate_instrument": "internal/sessrepo/chain.go: checkWinningLease returns nil in both isolated copies",
        "runtime_command": command,
        "runtime_exit_codes": {"before": before_exit, "after": after_exit},
        "outcome_key_count_before": len(before),
        "outcome_key_count_after": len(after),
        "moved_outcome_key_count": len(moved),
        "moved_input_classes": sorted(moved_classes),
        "outcomes_before": before,
        "outcomes_after": after,
        "moved_outcomes": moved,
    }
    (OUTPUT / "outcome-comparison.json").write_text(json.dumps(report, indent=2) + "\n")
    shutil.rmtree(copies)
    print(f"importer packages: {', '.join(before_importers)}")
    print(f"outcome keys: {len(before)} before / {len(after)} after; moved={len(moved)}")
    print(f"moved classes: {', '.join(sorted(moved_classes))}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
