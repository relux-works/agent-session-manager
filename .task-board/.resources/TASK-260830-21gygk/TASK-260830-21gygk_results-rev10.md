# TASK-260830-21gygk — rev10 rework outcome (CR9 P1-A/P1-B repaired)

Authority: AX v0.6.0 (`0cbdf100dbf84df50c64f792b1f940e3a67859a6`).
Candidate: uncommitted Story worktree
`task-board/story/STORY-260830-3tq4ns` (no producer commit; handoff
snapshots the working tree). Base `8cf4aaaa190e6a11dff2661aa6806ce476128653`.
All CI local (darwin/arm64, Go 1.25.5); no hosted CI.

## What changed

1. `internal/sessquery/lease.go`: the shared profile owner now
   admits the full 5.2 pair sentence. New `fork.created` arm in
   `checkClosureProfilePairs` compares the event pair to the newly
   persisted Session Record creation profile with a null source
   (P1-A; `source_profile_event_id` provenance never read). New
   `referencedCheckpointProfile` derives every `session.resumed`
   expectation from its referenced checkpoint's own event-head
   closure through the admitted Checkpoint Records threaded from
   the reader — no new store (`checkWinnerCheckpoint` and
   `checkCheckpointBinding` already held the validated map) — and
   resolves the source against that referenced closure (P1-B).
   Missing/wrong-session/unresolvable referenced authority refuses
   `observation_unavailable`; contradictory pairs refuse
   `integrity_failure`. Later-launch derivation is unchanged by
   normative text (launches carry no `checkpoint_id`; the admitting
   closure in 2.4 lease/sequence order is their only referenced
   authority). All four entries (`BuildPlan`, fresh and old-plan
   `Revalidate`, `AuthoritativeStatus`, `AuthoritativeList`) funnel
   through the same owner.
2. `internal/sessquery/rev10_regression_test.go` (new): ported
   reviewer probes `TestReview9ForkLocalProfile` and
   `TestReview9ResumeReferencedCheckpoint` kept verbatim, plus
   fork/resume matrices (direct/task_board x winner/ancestor with
   refusal-class pins), fork and resume old-plan substitution,
   referenced withdrawal, and task_board.launched first/later
   suites.
3. `internal/sessquery/rev8_regression_test.go`: two retained
   resume fixtures repaired for the new required input (the
   positive control supplies the referenced record; the
   dangling-source negative isolates the source with a valid
   reference). No retained expectation weakened.
4. `internal/sessquery/testdata/mutate.py`: four new narrowing
   plants (`N-profile-fork-value`, `N-profile-resume-missing`,
   `N-profile-resume-newest`, `N-profile-first-value`).
5. `internal/sessquery/TRACEABILITY.md`, `README.md`, `LOGBOOK.md`:
   derivation text, refusal rows, and mutation accounting updated;
   the P2 accounting correction is recorded (56-plant rev8 base =
   39 semantic + 17 precision, not 59 narrowing kills;
   `N-checkpoint-heads` is class/order precision). No product
   behavior outside the assigned selector/plan/summary scope.

## AC ratio: 8 of 8 rows driven through shared production entries, 8 of 8 established

Public CLI remains a stated bound (0 of 8 CLI rows delivered by this leaf).

| Row | Production call and named evidence | Result |
| --- | --- | --- |
| UUID/name | Reader.Resolve; TestResolvePinnedPrecedence, TestResolveBareIdentityUnion | Established (unchanged, suite green) |
| Qualified selectors | Reader.Resolve/resolveExplicit; TestResolveQualifiedSourcesSelectOneIndex, TestResolveQualifiedSourceNeverFallsBack, TestResolveExplicitSourceReadFailures | Established (unchanged, suite green) |
| Ambiguity | Reader.Resolve/matchName/checkRecordAgreement; TestResolveExactNamesAndASCIICollisions, TestResolveBareIdentityUnion | Established (unchanged, suite green) |
| List summaries | Reader.AuthoritativeList/authorize/winningLeaseFor; TestSummaryBootstrapRefusals, TestAuthoritativeAdmitsFullCapabilityRegistry; new TestReview9* + TestRev10* suites | Established (P1-A/B closed) |
| Status summaries | Reader.AuthoritativeStatus/authorize/winningLeaseFor; TestAuthoritativeStatusHealthy, TestAuthoritativeMissingProcessRefusesStatus; new TestReview9* + TestRev10* suites | Established (P1-A/B closed) |
| Stable sorting | Reader.List/Resolve; TestListStatusDerivedFactsAndStableOrder, TestResolvePeerOrderAndReplicatedIdentity | Established (unchanged, suite green) |
| SelectionPlan | Reader.BuildPlan/Revalidate/ParsePlan; TestBuildPlanBindsCurrentFacts, TestRevalidateDetectsEachFactChange, TestValidSuccessorBuildsAndRevalidates, TestRevalidateUnionParkedCopyRefuses, TestRev5/6/7/8 suites; new rev10 suite + substitution + withdrawal + fresh-plan revalidation | Established (P1-A/B closed) |
| Negative/refusal | Shared entries above; shipped refusal tests and mutation instrument; new rev10 suite + 4 narrowing plants | Established (P1-A/B closed; P2 accounting corrected) |

## Relation census: 6 of 6 rows driven, 0 undriven

`TASK-260830-21gygk_relation-census-rev10.md` (also the C6 section of
`TASK-260830-21gygk_conformance-matrix-rev10.md`): provider.launched
first/later, task_board.launched first/later, session.resumed,
fork.created — each with authority input, owner arm, positive
control, negatives, and narrowing plant. Caller bounds named with
owning clauses (fork provenance 2.4, storage existence,
launch-mode/reducer, confirmation/publication, CLI).

## Failing baseline (pre-fix, recorded)

`.temp/TASK-260830-21gygk/rev10-baseline-red.log` (exit 1): ported
probes on the unmodified candidate — 3 positive controls passed, all
3 negatives admitted (BuildPlan succeeded, fresh Revalidate nil,
both summaries succeeded).

## Final validation (real exits, final source)

- `go build ./...` → 0; `go vet ./...` → 0; `gofmt -l internal/` →
  empty; `git diff --check` → 0.
- `go test ./... -count=1` → 0 (all packages ok; log
  `full-suite-final.log`).
- `go test ./internal/sessquery ./internal/sessrepo -count=1 -cover`
  → 0; sessquery 86.7%, sessrepo 87.2% (`cover-final.log`).
- Focused: `TestReview9*` + `TestRev10*` verbose green
  (`rev10-final-green.log`); retained `TestRev8*`/`TestRev7*` green
  in the full suite.

## Mutation evidence (same instrument, final Go source)

Shipped harness `internal/sessquery/testdata/mutate.py` (with the 4
new plants) run as 30 two-plant slices + 2 classifier slices on
exact final bytes; per-slice `mutants.json` + raw logs in the
evidence bundle. Result: all 60 unique N/B plants (57 N, 3 B)
KILLED with named failing tests and real exit 1 — 43
semantic-admission kills (behavioral admission failures) and 17
label/class/output-precision kills (14 N + 3 B output-order, incl.
`N-checkpoint-heads` by wrong-class pin); controls
`control-before`/`control-after` exit 0 in every slice;
`C-harmless-comment` SURVIVED applied (exit 0, same instrument);
`C-not-applied` NOT_APPLIED and `C-compile-failure`
COMPILE_OR_HARNESS_FAILURE report separately and are not counted
as kills.

Slice harnesses were mechanical name-filter copies of the shipped
instrument (same application, subprocess, classifier, and restore
code; deleted after the runs); the Go source under battery was
byte-identical to final (see `source-manifest-rev10.sha256`).

New plants: `N-profile-fork-value` killed by the ported fork probe,
the rev10 fork matrix, and fork old-plan substitution (admission);
`N-profile-resume-missing` killed by the missing negatives
(admission; other missing pairs still refuse);
`N-profile-resume-newest` killed by the non-newest negative
(admission); `N-profile-first-value` killed by the first-profile
negatives (admission). Per-plant admission lines are quoted in the
evidence bundle (`mutation-classification-rev10.md`).

Environment note: two slice runs were discarded for environmental
pollution (disk-full TempDir failures; a corrupt shared Go build
cache after the pressure — repaired with `go clean -cache` and
verified with a clean build). No kill is claimed from a polluted
run; every reported slice ran green controls on the exact final
source.

## Stated bounds (no unsupported claims)

No public CLI surface; no publication/storage/transport/CLI
integration. Fork `source_profile_event_id` provenance is
caller-owned cross-session input (non-participation proven);
referenced-checkpoint storage existence is caller-owned (supplied
closure access is owned); task-board launch lease/mode binding,
change-source continuity, and confirmation gating keep their
retained boundaries. A non-null fork source has no chained fixture
(the canonical owner pins it at append — observed refusal); a
well-formed post-head launch citation has none by causal necessity
(historical controls prove non-consultation). Mid-derivation
read-failed branches are defensive (same retained helpers, cause
kept in unavailable). `lease_record_id` remains absent-nowhere:
bound from the same winning record as before (unchanged).

## Deliverables for review

- Candidate: uncommitted worktree diff (product: `lease.go`;
  tests: `rev10_regression_test.go`, `rev8_regression_test.go`
  fixture repairs; harness: `testdata/mutate.py`; docs:
  `TRACEABILITY.md`, `README.md`, `LOGBOOK.md`).
- Attached: `TASK-260830-21gygk_results-rev10.md` (this file),
  `TASK-260830-21gygk_conformance-matrix-rev10.md`,
  `TASK-260830-21gygk_relation-census-rev10.md`, and
  `TASK-260830-21gygk_producer-evidence-rev10.tar.gz` (baseline RED
  log, final validation logs, 32-slice mutants.json + raw logs,
  mutation classification, conformance matrix, census, source
  manifest).
- Handoff via normal task-board CR lifecycle; no self-acceptance,
  no integration, no manual commits.

## Republish (RUN-260916-b35886)

Republish-only run; no product edits. Previous runs: RUN-260916-dea59a
completed the rev10 rework and ran handoff at 16:29:04Z but was killed at
the 4h launcher budget (17:00:12Z) during CR construction, so no CR10
exists; RUN-260916-3a31de was cancelled for the missing outcome update
(this section satisfies the completion guard).

Manifest verification (worktree checkpoint 83640d19):
- `sha256sum -c .temp/TASK-260830-21gygk/source-manifest-rev10.sha256`:
  all 7 files OK (exit 0).
- Tarball manifest
  (`TASK-260830-21gygk_producer-evidence-rev10.tar.gz:source-manifest-rev10.sha256`)
  byte-identical to worktree manifest (diff exit 0).
- No `internal/sessquery/testdata/rev10_s*_mutate.py`, `rev10_slice*`,
  or probe/slice files under internal/sessquery, internal/sessrepo,
  internal/sessstate.
- `git status` excluding the pre-existing Sep 7-8 `.task-board`
  checkout-artifact dirt matches `candidate-base-rev10.txt` exactly
  (8 candidate paths; branch still at 83640d19, candidate uncommitted).

Quick gates (this run):
- `gofmt -l internal/`: exit 0, no output (clean).
- `go build ./...`: exit 0.
- `go vet ./...`: exit 0.
- `go test ./internal/sessquery/ ./internal/sessrepo/ -count=1`:
  exit 0 (`ok sessquery 60.333s`, `ok sessrepo 4.827s`).
- Full suites / race / cover / mutation batteries intentionally not
  rerun per the republish brief; the runtime's 26-command validation
  suite runs during CR construction.

## Republish (RUN-260916-2ef543)

Republish-only run; no product edits. Previous runs: RUN-260916-dea59a
completed the rev10 rework and ran handoff at 16:29:04Z but was killed at
the 4h launcher budget (17:00:12Z) during CR construction, so no CR10
existed; RUN-260916-3a31de was cancelled for the missing outcome update;
RUN-260916-b35886 verified the candidate but CR construction refused
change_request_base_authority_mismatch (checkpoint 83640d19 predates trunk
9ff7d2c1/PR41). This run performed the managed refresh first:
`task-board worktree refresh-candidate` -> refresh_advanced, new signed
checkpoint 1f34c0c1 descending from trunk 9ff7d2c1 (ANCESTOR_OK); old-vs-new
checkpoint tree diff touches task-board.config.json only (the PR41 trunk
delta), all other committed content byte-identical. The refresh exposed a
spurious working-tree revert of task-board.config.json (stale pre-refresh
bytes vs new HEAD); restored to HEAD via `git checkout --` so the routing
policy is not part of the candidate. No other working-tree bytes touched.

Manifest verification (new checkpoint 1f34c0c1):
- `sha256sum -c .temp/TASK-260830-21gygk/source-manifest-rev10.sha256`:
  all 7 files OK (exit 0).
- Tarball manifest byte-identical to worktree manifest (diff exit 0).
- No `internal/sessquery/testdata/rev10_s*_mutate.py`, `rev10_slice*`,
  or probe/slice files under internal/sessquery, internal/sessrepo,
  internal/sessstate (testdata/ holds mutate.py only).
- `git status` excluding the pre-existing `.task-board` checkout-artifact
  dirt matches `candidate-base-rev10.txt` exactly (5 modified + 3 untracked
  candidate paths; candidate uncommitted).

Quick gates (this run):
- `gofmt -l internal/`: exit 0, no output (clean).
- `go build ./...`: exit 0.
- `go vet ./...`: exit 0.
- `go test ./internal/sessquery/ ./internal/sessrepo/ -count=1`:
  exit 0 (`ok sessquery 55.329s`, `ok sessrepo 3.148s`).
- Full suites / race / cover / mutation batteries intentionally not
  rerun per the republish brief; the runtime's 26-command validation
  suite runs during CR construction.
