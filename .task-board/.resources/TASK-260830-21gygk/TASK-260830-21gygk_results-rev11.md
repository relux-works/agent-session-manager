# TASK-260830-21gygk — rev11 rework outcome (CR10 P1 referenced admission closed)

Authority: AX v0.6.0 (`0cbdf100dbf84df50c64f792b1f940e3a67859a6`).
Candidate: uncommitted Story worktree `task-board/story/STORY-260830-3tq4ns`
(no producer commit; handoff snapshots the working tree). Base
`9ff7d2c1d4d391dbc58d40b812757052d324a775` with checkpoints `7189c41`
and `1f34c0c`; continued from the uncommitted CR10 candidate (tree
`106f5e99000827f20cfae6780c6f93eb4321c505`); preserved predecessors and
foreign changes; no manual commits, no `.task-board` edits, no
self-acceptance. All CI local (darwin/arm64, Go 1.25.5); no hosted CI.
Curator `go-testing-tools` skill read first.

## What changed (5 files, shared owner only)

1. `internal/sessquery/lease.go`: new `checkReferencedCheckpointBinding`
   admits every `session.resumed` referenced Checkpoint Record through the
   same semantic owner the winner and necessary-ancestor checkpoints use:
   owning lease tuple resolved in the winning ancestry chain, creator-holder,
   `checkCheckpointPersistence` (Session.kind variant), and
   `checkCheckpointEventHeads` (at-or-before owning lease) via the shared
   helpers. The winning chain and Session.kind thread from `winningLeaseFor`
   through `checkWinnerCheckpoint`/`checkCheckpointBinding` →
   `checkCheckpointProfileAuthority` → `checkClosureProfilePairs` →
   `referencedCheckpointProfile` (signatures extended; winner/ancestor gate
   bodies untouched). The old session-only + head-membership check is
   replaced; missing/wrong-session/wrong-lease/wrong-creator/wrong-variant/
   unresolvable-head references refuse `observation_unavailable`;
   contradictory pairs still refuse `integrity_failure`. Historical,
   branching, and lagging references stay admitted. No new store,
   transport, publication, or materialization.
2. `internal/sessquery/rev11_regression_test.go` (new): ported
   `TestReview10ReferencedCheckpointAdmission` kept verbatim (12 negatives +
   4 controls, direct/task_board × winner/ancestor), plus
   `TestRev11ReferencedCheckpointAdmissionRefusalClass` (12 unavailable
   pins), `TestRev11ReferencedCheckpointAdmissionOldPlan` (12 old-plan
   unavailable), and `TestRev11RecordConsumptionCensus` (call-graph gate:
   resume calls the shared referenced owner before `profileClosure`, owner
   calls the shared persistence/head helpers, source-order check).
3. `internal/sessquery/testdata/mutate.py`: three new narrowing plants
   (`N-referenced-creator/variant/lease`, each admits exactly one invalid
   referenced member while preserving every census call token) + `LEASECYCLEB`
   constant. Battery now 63 N/B (60 N, 3 B).
4. `internal/sessquery/TRACEABILITY.md`: SelectionPlan row extended with
   rev11 tests and referenced admission semantics; four new refusal rows
   (creator/variant/lease + census gate); battery 63 (46 semantic, 17
   precision); token-preserving census note; bounds paragraph updated.
5. `LOGBOOK.md`: rev11 entry (defect/fix/instrument, battery, RED log).

No product behavior outside the assigned selector/plan/summary scope.
README unchanged (high-level claims still accurate).

## AC ratio: 8 of 8 rows driven through shared production entries, 8 of 8 established

Public CLI remains a stated bound (0 of 8 CLI rows delivered by this leaf).

| Row | Production call and named evidence | Result |
| --- | --- | --- |
| UUID/name | `Reader.Resolve`; `TestResolvePinnedPrecedence`, `TestResolveBareIdentityUnion` | Established (unchanged, suite green) |
| Qualified selectors | `Reader.Resolve`/`resolveExplicit`; `TestResolveQualifiedSourcesSelectOneIndex`, `TestResolveQualifiedSourceNeverFallsBack`, `TestResolveExplicitSourceReadFailures` | Established (unchanged, suite green) |
| Ambiguity | `Reader.Resolve`/`matchName`/`checkRecordAgreement`; `TestResolveExactNamesAndASCIICollisions`, `TestResolveBareIdentityUnion` | Established (unchanged, suite green) |
| List summaries | `Reader.AuthoritativeList`/`authorize`/`winningLeaseFor`; `TestSummaryBootstrapRefusals`, `TestAuthoritativeAdmitsFullCapabilityRegistry`, `TestReview9*`/`TestRev10*`, new `TestReview10*` + `TestRev11*` | Established (P1 closed; 12 negatives refuse unavailable, 4 controls admit) |
| Status summaries | `Reader.AuthoritativeStatus`/`authorize`/`winningLeaseFor`; `TestAuthoritativeStatusHealthy`, `TestAuthoritativeMissingProcessRefusesStatus`, `TestReview9*`/`TestRev10*`, new `TestReview10*` + `TestRev11*` | Established (P1 closed) |
| Stable sorting | `Reader.List`/`Resolve`; `TestListStatusDerivedFactsAndStableOrder`, `TestResolvePeerOrderAndReplicatedIdentity` | Established (unchanged, suite green) |
| SelectionPlan | `Reader.BuildPlan`/`Revalidate`/`ParsePlan` (`plan.go:274`, `revalidate.go:101` via `winningLeaseFor`); `TestBuildPlanBindsCurrentFacts`, `TestRevalidateDetectsEachFactChange`, `TestRev10ResumeOldPlanSubstitution/ReferencedWithdrawal`, new `TestRev11*OldPlan` (12 old-plan unavailable) | Established (P1 closed; fresh + old-plan refuse unavailable) |
| Negative/refusal | Shared entries above; shipped refusal tests + 63-plant battery + census gate; new `TestReview10*` (12+4), `TestRev11*` (12+12+gate), 3 N-referenced plants | Established (P1 closed; P2 accounting extended honestly) |

## Relation census: 6 of 6 pair rows + 7 of 7 consumption rows driven, 0 undriven

`TASK-260830-21gygk_relation-census-rev11.md` (and C6b in
`TASK-260830-21gygk_conformance-matrix-rev11.md`): R1-R6 retained with R5
extended (referenced semantic admission, 12 new negatives, 5 plants); new
K1-K7 record-consumption table (Session Record, winner checkpoint,
necessary-ancestor checkpoint, REFERENCED resume checkpoint, lease records,
profile.changed and other events) each with admission owner, call site,
positive control, negatives (wrong creator/variant/lease, missing,
read-failed), and narrowing plant. Caller bounds named with owning clauses
(fork provenance 2.4, storage existence, launch-mode/reducer,
confirmation/publication, CLI).

## Failing baseline (pre-fix, recorded)

`.temp/TASK-260830-21gygk/rev11-baseline-red.log` (exit 1): ported
`TestReview10ReferencedCheckpointAdmission` on the unmodified candidate — 4
controls passed, all 12 negatives admitted (BuildPlan succeeded, fresh
Revalidate nil, both summaries succeeded); old-plan Revalidate refused
`selector_plan_stale` in all 12 (valid, preserved as refusal; no new
error-order requirement).

## Final validation (real exits, final source)

- `go build ./...` → 0; `go vet ./...` → 0; `gofmt -l internal/` →
  empty; `git diff --check` → 0.
- `go test ./... -count=1` → 0 (all packages ok; log
  `full-suite-final-rev11.log`).
- `go test ./internal/sessquery ./internal/sessrepo -count=1 -cover`
  → 0; sessquery 86.8%, sessrepo 87.2% (`cover-final-rev11.log`).
- Focused: `TestReview10*` + `TestRev11*` + `TestReview9*` + `TestRev10*`
  resume/fork green (`rev11-final-green.log`, exit 0); retained
  `TestReview9|TestRev10|TestRev8ProfileHistorical|TestRev8ProfileTwoGeneration`
  green (exit 0, 72s).
- Census gate `TestRev11RecordConsumptionCensus` → 0 (0.00s).

## Mutation evidence (same instrument, final Go source for new plants)

Shipped harness `internal/sessquery/testdata/mutate.py` (with 3 new plants)
run as 2 two-to-four-plant slices + 1 harmless-only slice on exact final
bytes (slice harnesses are mechanical name-filter copies, retained in the
bundle); per-slice `mutants.json` + raw logs in the evidence bundle.
Result for new plants + controls:

- `N-referenced-creator` KILLED exit 1 (15 failed: 4 TestReview10 wrong_creator
  + 4 RefusalClass wrong_creator + 4 OldPlan wrong_creator + 3 parents;
  census PASS). Log `slice1_out/N-referenced-creator.log` (63K).
- `N-referenced-variant` KILLED exit 1 (15 failed, all wrong_variant; census
  PASS). Log `slice1_out/N-referenced-variant.log` (63K).
- `N-referenced-lease` KILLED exit 1 (15 failed, all wrong_lease; census
  PASS). Log `slice2_out/N-referenced-lease.log` (63K).
- Controls `control-before`/`control-after` exit 0 in every slice;
  `C-harmless-comment` SURVIVED applied exit 0 via dedicated slice
  (`slice_harmless_out`, 62K log, no failed tests, same instrument);
  `C-not-applied` NOT_APPLIED (count 0, no log by design);
  `C-compile-failure` COMPILE_OR_HARNESS_FAILURE exit 1 with no named
  failed tests (intended syntax error; sessrepo still PASS).
- All three kills are behavioral admission failures (e.g.,
  `BuildPlan admitted invalid lease/checkpoint authority; Revalidate=<nil>`);
  the static census gate passes for all three (token-preserving), so only
  the behavioral suites fail. Harness always executes the behavioral suites.

Retained 60-plant accounting (CR10 P2, reviewed correct: 43 semantic + 17
precision) is preserved and extended honestly to 63 (46 semantic + 17
precision):

- Reran myself (rev11 final source): 3 new plants (full suites, same
  instrument, per-plant logs + exits above) + harmless SURVIVED (same
  instrument) + 3 most affected old plants via focused behavioral reruns
  (exit 1, named failures, logs in `mutants-rev11/focused-old/`):
  `N-profile-resume-missing` (fails bad_missing),
  `N-profile-resume-newest` (fails bad_non_newest), `N-checkpoint-heads`
  (fails missing_head with wrong-class precision).
- Accepted from already-attached rev10 exact-candidate evidence (tree
  106f5..., 60 KILLED with raw logs, CR10 verdict P2): remaining 57 old
  plants' full-suite kills. Bounds: all 63 N/B anchors verified count 1 on
  rev11 final source (plus C-harmless 1, C-compile 1, C-not-applied 0);
  non-lease source files byte-identical between rev10 and rev11 candidates
  (only lease.go, rev11 test, mutate.py, TRACEABILITY, LOGBOOK changed);
  lease-body anchors unchanged (only signatures threaded with chain/kind;
  winner/ancestor gate bodies untouched); focused reruns confirm no
  regression in the touched file.
- Environment notes (no kill claimed from polluted runs): one parallel slice
  attempt timed out under contention and was discarded with no logs retained;
  one harmless run hit transient Go cache corruption (stdlib import failures)
  and was discarded and rerun cleanly after `go clean -cache` (clean build +
  vet verified). N-lease, C-compile, C-not-applied, and controls from the
  rerun slice retained (logs show no cache errors). Slice harnesses retained
  in the bundle (unlike rev10, whose filtering was producer-provenance only).

Per-plant admission lines, exits, and inventory are in
`mutation-classification-rev11.md` in the bundle.

## Stated bounds (no unsupported claims)

No public CLI surface (0 of 8 CLI rows); no publication/storage/transport/CLI
integration. Fork `source_profile_event_id` provenance is caller-owned
cross-session input (non-participation proven); referenced-checkpoint storage
existence is caller-owned (supplied closure + ancestry access is owned);
task-board launch lease/mode binding, change-source continuity, and
confirmation gating keep their retained boundaries. A non-null fork source has
no chained fixture (canonical pins at append); a well-formed post-head launch
citation has none by causal necessity (historical controls prove
non-consultation). Mid-derivation read-failed branches are defensive (same
retained helpers, cause kept in unavailable; missing vs read-failed stay
distinct). `lease_record_id` remains absent-nowhere: bound from the same
winning record as before (unchanged). Old-plan class change disclosed: the 12
ported old-plans previously refused `selector_plan_stale` (winning lease
changed) because invalid referenced authority was admitted; they now refuse
`selector_observation_unavailable` (referenced admission) because authority is
validated before digest comparison. Both are refusals; no new error-order
requirement (per CR10 verdict bound). No unsupported end-to-end claim.

## Deliverables for review

- Candidate: uncommitted worktree diff (product: `lease.go` referenced
  admission + threading; tests: `rev11_regression_test.go` (ported + 3 new
  suites); harness: `testdata/mutate.py` (3 new plants); docs:
  `TRACEABILITY.md`, `LOGBOOK.md`).
- Attached: `TASK-260830-21gygk_results-rev11.md` (this file),
  `TASK-260830-21gygk_conformance-matrix-rev11.md`,
  `TASK-260830-21gygk_relation-census-rev11.md` (with K1-K7 table), and
  `TASK-260830-21gygk_producer-evidence-rev11.tar.gz` (baseline RED log,
  final validation logs, 3-slice mutants.json + raw logs + slice harnesses,
  focused-old logs, mutation classification, census, matrix, source manifest).
- Handoff via normal task-board CR lifecycle; no self-acceptance, no
  integration, no manual commits.
