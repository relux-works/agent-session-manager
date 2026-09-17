#!/usr/bin/env python3
"""Narrowing-mutant harness for internal/matjournal.

Every gate ships a narrowing mutant: the gate stays present and is
weakened to admit exactly one member of the class it must reject, and
the named killer test must fail. A delete-only mutant proves only that
the gate exists and is not accepted as evidence.

Each row runs M (mutate one production file in place), R (run the
scoped Go killer through the production entry point), V (verdict from
the Go exit code plus a FAIL kill-line: KILLED iff R fails with a test
failure line, SURVIVED iff R passes, ERROR for anything else —
unapplied patches, build breaks, timeouts). M failures and R
infrastructure errors are ERROR, never kills or survivals. The file is
restored from its backup after every row, pass or fail.

Row C-doc-comment is the harmless control: a comment-only edit that
must SURVIVE, proving the harness observes test outcomes instead of
hard-coding kills.

Usage:
  python3 internal/matjournal/mutant_harness.py            # all rows
  python3 internal/matjournal/mutant_harness.py N-phase-table
"""

import shutil
import subprocess
import sys
import tempfile
from dataclasses import dataclass
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
PACKAGE = "internal/matjournal"


@dataclass(frozen=True)
class Mutant:
    name: str
    path: str
    find: str
    replace: str
    count: int
    run: str
    expect: str
    note: str


MUTANTS = [
    Mutant(
        name="N-phase-table",
        path=f"{PACKAGE}/journal.go",
        find="\tPhaseStaging:     {PhaseValidating, PhaseRollingBack, PhaseFailed},\n",
        replace="\tPhaseStaging:     {PhaseValidating, PhaseRollingBack, PhaseFailed, PhaseCommitted},\n",
        count=1,
        run="TestTransitionRefusesIllegalEdges",
        expect="KILLED",
        note="Admits exactly staging-to-committed; every phase literal is preserved, only the edge set changes.",
    ),
    Mutant(
        name="N-board-subtable",
        path=f"{PACKAGE}/journal.go",
        find="\tBoardNotStarted:       {BoardImported, BoardRolledBack, BoardFailed},\n",
        replace="\tBoardNotStarted:       {BoardImported, BoardOpened, BoardRolledBack, BoardFailed},\n",
        count=1,
        run="TestBoardNullabilityRefusals/subtable_refuses_skip",
        expect="KILLED",
        note="Admits exactly not_started-to-opened; literals preserved, edge added.",
    ),
    Mutant(
        name="N-provider-token",
        path=f"{PACKAGE}/journal.go",
        find='''	if provider.State == ProviderPrepared {
		if provider.RollbackToken == nil {
			return invalid("provider prepared state requires a non-null rollback token")
		}
''',
        replace='''	if provider.State == ProviderPrepared {
		if provider.RollbackToken == nil && provider.State == ProviderUnknown {
			return invalid("provider prepared state requires a non-null rollback token")
		}
''',
        count=1,
        run="TestProviderTokenAndIDRules/prepared_requires_token",
        expect="KILLED",
        note="Admits exactly prepared-without-token; the null check stays, gated on an impossible state.",
    ),
    Mutant(
        name="N-board-import",
        path=f"{PACKAGE}/journal.go",
        find="\tif board.ImportToken == nil || board.StagedManagerRef == nil || board.ImportExpiresAt == nil {\n\t\treturn invalid(\"task-board imported requires the import token, staged reference, and expiry\")\n",
        replace="\tif board.ImportToken == nil || board.StagedManagerRef == nil {\n\t\treturn invalid(\"task-board imported requires the import token, staged reference, and expiry\")\n",
        count=1,
        run="TestBoardNullabilityRefusals/imported_triple_incomplete",
        expect="KILLED",
        note="Admits exactly expiry-less imports; the token and reference checks stay.",
    ),
    Mutant(
        name="N-id-drift-adopt",
        path=f"{PACKAGE}/journal.go",
        find="""	if board.ImportOperationID != bound.Import || board.OpenOperationID != bound.Open ||
		board.AdoptOperationID != bound.Adopt || board.ResumeOperationID != bound.Resume {
""",
        replace="""	if board.ImportOperationID != bound.Import || board.OpenOperationID != bound.Open ||
		board.ResumeOperationID != bound.Resume {
""",
        count=1,
        run="TestBoardNullabilityRefusals/operation_drift_refuses",
        expect="KILLED",
        note="Admits exactly adopt-ID drift; import, open, and resume still bind.",
    ),
    Mutant(
        name="N-create-first-prepare",
        path=f"{PACKAGE}/store.go",
        find="\tif journal.PrepareOperationID != prepare {\n",
        replace='\tif journal.PrepareOperationID != prepare && journal.PrepareOperationID == "" {\n',
        count=1,
        run="TestCreateReplayAndConflict",
        expect="KILLED",
        note="Admits exactly the different-operation same-body replay past the first-prepare check.",
    ),
    Mutant(
        name="N-rollback-adopt",
        path=f"{PACKAGE}/journal.go",
        find="""		case BoardAdopted, BoardResumed:
			return invalid("byte rollback is forbidden after adopt: the manager is active authority")
""",
        replace="""		case BoardResumed:
			return invalid("byte rollback is forbidden after adopt: the manager is active authority")
""",
        count=1,
        run="TestTransitionGuards/rollback_refused_after_adopt",
        expect="KILLED",
        note="Admits exactly adopted rollback; resumed still refuses.",
    ),
    Mutant(
        name="N-failed-null",
        path=f"{PACKAGE}/journal.go",
        find="\t\tif len(opts.LastError) == 0 || isNull(opts.LastError) {\n",
        replace="\t\tif len(opts.LastError) == 0 {\n",
        count=1,
        run="TestTransitionGuards/failed_requires_error",
        expect="KILLED",
        note="Admits exactly the explicit-null explaining error; missing and malformed still refuse.",
    ),
    Mutant(
        name="N-commit-marker",
        path=f"{PACKAGE}/journal.go",
        find="""		if journal.ManagedReplicaID != nil {
			if len(opts.Marker) == 0 {
				return invalid("committed workspace transaction requires the destination marker")
			}
""",
        replace="""		if journal.ManagedReplicaID != nil && len(journal.AuthorityStates) == 0 {
			if len(opts.Marker) == 0 {
				return invalid("committed workspace transaction requires the destination marker")
			}
""",
        count=1,
        run="TestTransitionGuards/committed_workspace_requires_marker",
        expect="KILLED",
        note="Admits exactly marker-less commits on journals with authorities; the killer asserts the guard message.",
    ),
    Mutant(
        name="N-recover-uncertain-floor",
        path=f"{PACKAGE}/recover.go",
        find="""	if journal.Provider != nil {
		return journal.Phase == PhasePrepared || journal.Phase == PhaseCommitting ||
			journal.Phase == PhaseRollingBack
	}
	return boundaryAtOrAfter(assessment.input.Boundary, BoundaryAfterPrepareOp)
""",
        replace="""	if journal.Provider != nil {
		return journal.Phase == PhasePrepared || journal.Phase == PhaseCommitting ||
			journal.Phase == PhaseRollingBack
	}
	return boundaryAtOrAfter(assessment.input.Boundary, BoundaryAfterFinalize)
""",
        count=1,
        run="TestParkedEvidenceIsDurable",
        expect="KILLED",
        note="Admits exactly unrecorded-prepare unknown probes at CR-MAT-05..07; the parked assessment must still refuse.",
    ),
    Mutant(
        name="N-token-length",
        path=f"{PACKAGE}/journal.go",
        find="\tif len(raw) < minTokenBytes || len(raw) > maxTokenBytes {\n",
        replace="\tif len(raw) < minTokenBytes && len(raw) != 3 || len(raw) > maxTokenBytes {\n",
        count=1,
        run="TestProviderTokenAndIDRules/short_token_refuses",
        expect="KILLED",
        note="Admits exactly 3-byte tokens; every other short or long token still refuses.",
    ),
    Mutant(
        name="C-doc-comment",
        path=f"{PACKAGE}/doc.go",
        find="// Authority: relux-works/agent-session-manager-spec@v0.6.0, Section 10.6\n",
        replace="// Authority: relux-works/agent-session-manager-spec@v0.6.0, Section 10.6 (control).\n",
        count=1,
        run="TestTokenFixturesAreWellFormed",
        expect="SURVIVED",
        note="Harmless control: a comment-only edit must survive.",
    ),
]


def run_mutant(mutant: Mutant) -> str:
    target = REPO / mutant.path
    original = target.read_text()
    found = original.count(mutant.find)
    if found != mutant.count:
        return f"ERROR: {mutant.name}: patch anchors {found}, want {mutant.count}"
    with tempfile.TemporaryDirectory() as staging:
        backup = Path(staging) / "backup"
        shutil.copy2(target, backup)
        try:
            target.write_text(original.replace(mutant.find, mutant.replace))
            run = subprocess.run(
                ["go", "test", f"./{PACKAGE}/", "-run", mutant.run, "-count=1"],
                cwd=REPO,
                capture_output=True,
                text=True,
                timeout=300,
            )
        finally:
            shutil.copy2(backup, target)
    output = (run.stdout + run.stderr).strip()
    if 'build failed' in output or '\n# ' in output or output.startswith('# '):
        return f"ERROR: {mutant.name}: build broke under the patch"
    if run.returncode == 0:
        verdict = "SURVIVED"
    elif '--- FAIL' in output or 'FAIL:' in output:
        verdict = "KILLED"
    else:
        return f"ERROR: {mutant.name}: exit {run.returncode} with no kill line"
    mark = "ok" if verdict == mutant.expect else "MISMATCH"
    return f"{mark}: {mutant.name}: {verdict} (want {mutant.expect}) :: {mutant.note}"


def main(argv: list[str]) -> int:
    selected = [m for m in MUTANTS if not argv or m.name in argv]
    if argv and len(selected) != len(argv):
        unknown = sorted(set(argv) - {m.name for m in selected})
        print(f"unknown mutants: {', '.join(unknown)}")
        return 2
    failures = 0
    for mutant in selected:
        try:
            line = run_mutant(mutant)
        except subprocess.TimeoutExpired:
            line = f"ERROR: {mutant.name}: killer timed out"
        print(line, flush=True)
        if not line.startswith("ok:"):
            failures += 1
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
