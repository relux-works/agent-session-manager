# TASK-260830-21gygk — rev8 rework outcome (CR8 P1 repaired)

Authority: AX v0.6.0 (`0cbdf100dbf84df50c64f792b1f940e3a67859a6`).
Candidate: uncommitted Story worktree
`task-board/story/STORY-260830-3tq4ns` (no producer commit; handoff
snapshots the working tree). Base `8cf4aaaa190e6a11dff2661aa6806ce476128653`.
All CI local (darwin/arm64, Go 1.25.5); no hosted CI.

## What changed

1. `internal/sessquery/lease.go`: new shared admission
   `checkCheckpointProfileAuthority` (+ `sessionCreationProfile`,
   `profileClosure`, `eventProfilePair`, `eventProfileTarget`,
   `eventPayloadMembers`, `checkClosureProfilePairs`,
   `checkProfilePairSource`, `profilePair`), called after the heads
   gate in `checkWinnerCheckpoint` and `checkCheckpointBinding`, so
   all four entries (`BuildPlan`, fresh and old-plan `Revalidate`,
   `AuthoritativeStatus`, `AuthoritativeList`) admit Section 2.4
   profile authority over the winning-source closure. Semantic
   violations refuse `integrity_failure` (SPEC 5.4); failed reads
   refuse `observation_unavailable` with the cause kept.
2. `internal/sessquery/rev8_regression_test.go` (new): ported
   reviewer probe `TestRev8CheckpointProfileAuthority` plus
   real-change positives, eight refusal classes with the integrity
   class pinned, historical-closure admission, and old-plan
   profile-only substitution.
3. `internal/sessquery/rev7_regression_test.go`: missing/later-head
   refusals pin the `observation_unavailable` class
   (`rev7CheckHeadRefusalClass` + old-plan pin), locking heads-before-
   profile gate order against the new backstop.
4. `internal/sessquery/testdata/mutate.py`: four new narrowing
   plants (`N-profile-first-source`, `N-profile-newest`,
   `N-profile-value`, token-preserving `N-profile-change-direction`).
5. `internal/sessquery/TRACEABILITY.md`, `summary.go`/`lease.go`
   comments, `LOGBOOK.md`: evidence and bounds updated. No product
   behavior outside the assigned selector/plan/summary scope.

## AC ratio: 8 of 8 rows driven through shared production entries, 8 of 8 established

Public CLI remains a stated bound (0 of 8 CLI rows delivered by this leaf).

| Row | Production call and named evidence | Result |
| --- | --- | --- |
| UUID/name | Reader.Resolve; TestResolvePinnedPrecedence, TestResolveBareIdentityUnion | Established (unchanged, suite green) |
| Qualified selectors | Reader.Resolve/resolveExplicit; TestResolveQualifiedSourcesSelectOneIndex, TestResolveQualifiedSourceNeverFallsBack, TestResolveExplicitSourceReadFailures | Established (unchanged, suite green) |
| Ambiguity | Reader.Resolve/matchName/checkRecordAgreement; TestResolveExactNamesAndASCIICollisions, TestResolveBareIdentityUnion | Established (unchanged, suite green) |
| List summaries | Reader.AuthoritativeList/authorize/winningLeaseFor; TestSummaryBootstrapRefusals, TestSummaryMixedListRefusesWhole, TestAuthoritativeAdmitsFullCapabilityRegistry; new TestRev8CheckpointProfileAuthority, TestRev8ProfileSourceRefusals, TestRev8ProfileHistoricalClosureAdmits | Established (P1 closed) |
| Status summaries | Reader.AuthoritativeStatus/authorize/winningLeaseFor; TestAuthoritativeStatusHealthy, TestAuthoritativeMissingProcessRefusesStatus, TestAuthoritativeHostNameBound64; new rev8 suite as above | Established (P1 closed) |
| Stable sorting | Reader.List/Resolve; TestListStatusDerivedFactsAndStableOrder, TestResolvePeerOrderAndReplicatedIdentity | Established (unchanged, suite green) |
| SelectionPlan | Reader.BuildPlan/Revalidate/ParsePlan; TestBuildPlanBindsCurrentFacts, TestRevalidateDetectsEachFactChange, TestValidSuccessorBuildsAndRevalidates, TestRevalidateUnionParkedCopyRefuses, TestRev5/6/7 suites; new rev8 suite + old-plan substitution + fresh-plan revalidation | Established (P1 closed) |
| Negative/refusal | Shared entries above; shipped refusal tests and mutation instrument; new rev8 suite + 4 narrowing plants | Established (P1 closed) |

## Failing baseline (pre-fix, recorded)

`.temp/TASK-260830-21gygk/rev8-baseline-red.log` (exit 1): ported
probe on the unmodified candidate — 4 positive controls passed, all
4 missing-source negatives admitted (BuildPlan succeeded, fresh
Revalidate nil, both summaries succeeded).

## Final validation (real exits, final source)

- `go build ./...` → 0; `go vet ./...` → 0; `gofmt -l internal/` →
  empty; `git diff --check` → 0.
- `go test ./... -count=1` → 0 (all packages ok; log
  `full-suite-final.log`).
- `go test ./internal/sessquery ./internal/sessrepo -count=1 -cover`
  → 0; sessquery 86.7%, sessrepo 87.2% (`cover-final.log`).
- Focused: `TestRev8*` verbose green (`rev8-final-green.log`);
  `TestRev7*` verbose green incl. class pins (`rev7-final-green.log`).

## Mutation evidence (same instrument, final Go source)

Shipped harness `internal/sessquery/testdata/mutate.py` (with the 4
new plants) run as 4 slices on exact final bytes; per-slice
`mutants.json` + raw logs in the evidence bundle. Result: all 59
narrowing/behavioral plants KILLED with named failing tests and
real exit 1; controls `control-before`/`control-after` exit 0;
`C-harmless-comment` SURVIVED applied (exit 0, same instrument);
`C-not-applied` NOT_APPLIED and `C-compile-failure`
COMPILE_OR_HARNESS_FAILURE report separately and are not counted
as kills.

New plants: `N-profile-first-source` killed by the rev8
missing-source negatives + dangling-resume refusal;
`N-profile-newest` killed by the non-newest refusal +
two-generation control; `N-profile-value` killed by the stale-value
refusal; `N-profile-change-direction` killed by all P1/E1 positive
controls. Affected lease plants re-verified: all 8 KILLED,
including `N-checkpoint-heads` by the missing_head negatives
surfacing the wrong class through the new class pin (the profile
backstop refuses the weakened-heads fixture with integrity_failure;
unmutated heads still fire first with observation_unavailable —
gate order locked, no silent admission in either direction).

## Stated bounds (no unsupported claims)

No public CLI surface; no publication/storage/transport/CLI
integration; `fork.created` provenance, resume checkpoint-ID
existence, task-board launch lease-repeat/mode binding,
change-source continuity, and confirmation gating are not
re-derived here per the cited normative sentences (matrix C6b).
A well-formed post-head citation from inside the closure has no
fixture by causal necessity; non-consultation is proven by the
historical control. `lease_record_id` remains absent-nowhere:
bound from the same winning record as before (unchanged).

## Deliverables for review

- Candidate: uncommitted worktree diff (product: `lease.go`,
  `summary.go` comment; tests: `rev8_regression_test.go`,
  `rev7_regression_test.go` class pins; harness:
  `testdata/mutate.py`; docs: `TRACEABILITY.md`, `LOGBOOK.md`).
- Attached: `TASK-260830-21gygk_results-rev8.md` (this file),
  `TASK-260830-21gygk_conformance-matrix-rev8.md`,
  `TASK-260830-21gygk_producer-evidence-rev8.tar.gz` (baseline RED
  log, final validation logs, 4-slice mutants.json + raw logs,
  lease-focused mutants.json + heads log, conformance matrix,
  source manifest).
- Handoff via normal task-board CR lifecycle; no self-acceptance,
  no integration, no manual commits.
