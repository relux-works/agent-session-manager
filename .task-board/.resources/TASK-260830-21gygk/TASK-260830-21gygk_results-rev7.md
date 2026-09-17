# TASK-260830-21gygk — rev7 rework outcome (rev7 verdict changes-requested)

Producer rework for `TASK-260830-21gygk_review-verdict-rev7.md` (CR7 tree
`0715659e5ac515b387a9a3e5926cba4138d9cbea`, base `8cf4aaa...`, story
checkpoint `83640d191f78f0cd685e5f709027819799efe1cf`). P1 checkpoint
event-head authority fixed on the shared semantic admission path.
Candidate left UNCOMMITTED in the Story worktree for CR snapshotting;
no branch mutation, no integration, no hosted CI.

## Failing baseline (pre-fix, ported reviewer regression)

Copied `/tmp/rev7ev/reviewer_rev7_test.go` to
`internal/sessquery/rev7_regression_test.go` and ran before any
production change (`go test ./internal/sessquery -run '^TestRev7'`):

- `TestRev7IndependentPersistence` — all 8 direct/task_board
  winner/ancestor control/wrong_variant subtests PASSED (CR6 closed).
- `TestRev7CheckpointHeadAuthority/winner/control` PASSED,
  `ancestor/control` PASSED (real idle heads).
- `winner/missing_head`, `winner/later_lease_head`,
  `ancestor/missing_head`, `ancestor/later_lease_head` FAILED: BuildPlan
  admitted, fresh Revalidate nil, AuthoritativeStatus/List admitted;
  ancestor cases additionally admitted old-plan Revalidate after
  swapping only the ancestor checkpoint heads.
- Raw log: worktree `.temp/TASK-260830-21gygk/rev7-failing-baseline.log`
  (ignored scratch, exit 1).

## Production fix (`internal/sessquery/lease.go`, `plan.go`, `revalidate.go`, `summary.go`)

- `validatedCheckpoint` gains `EventHeads []string`; `parseCheckpointRecord`
  extracts `event_heads` as digests (`invalid_config` on grammar failure;
  shape 1..64 sorted-unique still owned by `canonicaljson` via
  `sessrepo.AttestCheckpointRecord`).
- New `checkCheckpointEventHeads(checkpoint, owner, repo)`: each head must
  name a chained event for the same session at or before its bound lease
  in the winning source chain (`repo.ListEvents`). Missing (incl.
  cross-session digests, losing-lease preserved blobs outside the
  authoritative chain, synthetic placeholders), later-epoch, and
  same-epoch foreign-lease heads refuse `selector_observation_unavailable`;
  historical heads admissible, never required to equal the current tail
  (branching/lagging preserved). Nil repo or failed chain read fails
  closed in the same class with its cause.
- `winningLeaseFor(sessionID, kind, repo)` threads the winning-source chain
  handle; `checkWinnerCheckpoint` and `checkAncestorCheckpoints` /
  `checkCheckpointBinding` call persistence then heads for the winner and
  every necessary ancestor.
- Call sites: `bindPlan` passes `selected.repo` (`plan.go`), `Revalidate`
  passes the plan-source `repo` (`revalidate.go`), `AuthoritativeStatus`
  resolves then passes `selected.repo`, `AuthoritativeList` passes
  `reader.Local` (`summary.go:authorize`). Direct test callers pass
  `reader.Local`.
- Post-fix: `go test ./internal/sessquery -run '^TestRev7'` — all pass
  (log `.temp/TASK-260830-21gygk/rev7-passing.log`, exit 0).

## Committed tests

- Ported reviewer test kept as `internal/sessquery/rev7_regression_test.go`
  (header documents the port; `TestRev7CheckpointHeadAuthority` +
  `TestRev7IndependentPersistence` through all four entries + old-plan).
- New helpers `checkpointWithHeads` / `headAtOrBefore` in
  `fixtures_test.go`; older successor fixtures rebound to real heads so
  closed gates stay isolated: `TestValidSuccessorBuildsAndRevalidates`,
  `TestEpochOneWithSelfBoundCheckpointBuilds` (now names the real
  bootstrap event), `TestSelfPredecessorWithCheckpointMustRefuse` (now
  carries a fully valid checkpoint as its comment claims),
  `TestMalformedCheckpointBytesRefuseConfig`, rev5 `complete`/`holder`
  controls, `rev5AncestorFixture` (reheads caller-supplied cp1 + cp2),
  rev6 winner/ancestor/task_board controls (variant swaps preserve heads),
  `withSuccessor` (union lagging copy).
- `TestRev5AncestorCheckpointBinding` wrong-session/lease/creator cases
  still refuse at their earlier gates (session/lease/creator precede
  heads), now with valid heads so the refusal proves the intended gate.

## Narrowing mutants (focused battery, same instrument as committed `testdata/mutate.py`)

Driver: worktree `.temp/TASK-260830-21gygk/focused_mutants_rev7.py`
(verbatim `run()`/command/timeout/classification, filtered selection);
raw logs + manifest: `/tmp/mutants-rev7-focused/` (scratch;
`mutants.json` copied to
`.temp/TASK-260830-21gygk/mutants-rev7-focused.json`). Each plant keeps
the gate and admits exactly one member; bytes restored with digest
verification; controls green.

| Mutant | Classification | Killed by |
| --- | --- | --- |
| N-lease-missing | KILLED (exit 1) | TestReviewerPlanRequiresLeaseRecord, TestRev3DifferentAttestationMustNotAuthorize |
| N-chain-self | KILLED (exit 1) | TestSelfPredecessorWithCheckpointMustRefuse (now with fully valid checkpoint) |
| N-checkpoint-placeholder | KILLED (exit 1) | TestRev4MissingCheckpointMustRefuse |
| N-ancestor-checkpoint | KILLED (exit 1) | TestRev5AncestorCheckpointAuthority, TestRev5AncestorCheckpointBinding, TestRev6..., TestRev7.../ancestor/... |
| N-checkpoint-holder | KILLED (exit 1) | TestRev5CheckpointCreatorMustBeHolder/non_holder |
| N-ancestor-holder | KILLED (exit 1) | TestRev5AncestorCheckpointBinding/wrong_creator |
| N-checkpoint-persistence | KILLED (exit 1) | TestRev6..., TestRev7IndependentPersistence wrong_variant arms |
| N-checkpoint-heads (new: admit exactly the all-zero missing head) | KILLED (exit 1) | TestRev7CheckpointHeadAuthority/winner/missing_head, ancestor/missing_head (later_lease arms still refuse: narrowing, not deletion) |
| C-harmless-comment | SURVIVED (exit 0, applied log) | control — behavior-preserving |
| control-before / control-after | CONTROL exit 0 | — |

The new heads plant added to committed `testdata/mutate.py`; anchor
`for _, head := range checkpoint.EventHeads {` counts 1 in final source.
All other lease.go anchors still count 1 (no signature drift left stale).

## Validation (all observed this run, final source)

- `go build ./...` — exit 0 (`.temp/.../build-rev7.log`).
- `go vet ./...` — exit 0 (`.temp/.../vet-rev7.log`).
- `gofmt -l internal/` — clean, no files listed (existing path; no `cmd/`
  in repo). NOTE retained: prior `cmd/` gofmt exit-2 description stands
  corrected.
- `git diff --check -- . ':!.task-board'` — exit 0.
- `go test ./... -count=1` — all packages ok.
- Coverage (`go test ./... -cover`): sessquery 87.4%, sessrepo 87.2%,
  plus full per-package table observed (no new uncovered gate).
- No durable writes on these paths (read-only selector leaf); no
  crash/idempotency protocol change — creation-crash composition tests
  still green.

## Docs / traceability

- `README.md` succession paragraph now states the head closure.
- `internal/sessquery/TRACEABILITY.md`: SelectionPlan row extended with
  rev7 tests + `N-checkpoint-heads`; new `Checkpoint event-head closure`
  gate row; driven text states the at-or-before binding.
- `LOGBOOK.md` carries the rev7 rework entry.
- Corrected clause-to-owner matrix:
  `.temp/TASK-260830-21gygk/conformance-matrix-rev7.md` (attached as
  task-scoped outcome): C6 false delegation replaced with owned
  `checkCheckpointEventHeads` call sites/inputs/tests; C5/P1 delegations
  verified against real inputs (`validateSafeBoundaryEvidence`,
  `BoundariesFor` + `TestActionBoundaryTable`); profile derivation shown
  absent by real evidence (no sessquery derivation, sessstate doc
  exclusion, provhost mapping only).

## AC coverage: 8 of 8 rows driven through shared production entries

`Reader.Resolve` (UUID/name: TestResolvePinnedPrecedence,
TestResolveBareIdentityUnion; qualified: TestResolveQualifiedSourcesSelectOneIndex,
TestResolveQualifiedSourceNeverFallsBack, TestResolveExplicitSourceReadFailures;
ambiguity: TestResolveExactNamesAndASCIICollisions; sorting:
TestListStatusDerivedFactsAndStableOrder, TestResolvePeerOrderAndReplicatedIdentity),
`Reader.BuildPlan`/`Revalidate`/`ParsePlan` (TestBuildPlanBindsCurrentFacts,
TestRevalidateDetectsEachFactChange, TestValidSuccessorBuildsAndRevalidates,
TestRevalidateUnionParkedCopyRefuses + rev5 ancestry/holder + rev6
persistence + rev7 head suites),
`Reader.AuthoritativeStatus`/`List` (TestAuthoritativeStatusHealthy,
TestAuthoritativeMissingProcessRefusesStatus,
TestAuthoritativeHostNameBound64, TestAuthoritativeAdmitsFullCapabilityRegistry
+ rev5/rev6/rev7 suites through both entries). Public CLI execution remains the
stated caller-integration bound (0 CLI rows in this leaf).

## Candidate paths (uncommitted, Story worktree)

- `internal/sessquery/lease.go` (heads parsing + shared admission + repo threading)
- `internal/sessquery/plan.go`, `revalidate.go`, `summary.go` (repo threading)
- `internal/sessquery/rev7_regression_test.go` (ported reviewer test, retained)
- `internal/sessquery/fixtures_test.go`, `succession_test.go`,
  `rev5_regression_test.go`, `rev6_regression_test.go` (real-head fixtures)
- `internal/sessquery/testdata/mutate.py` (new narrowing plant)
- `internal/sessquery/TRACEABILITY.md`, `README.md`, `LOGBOOK.md` (docs)
- Retained prior CR scope untouched (sessrepo checkpoint/lease owners,
  provhost/sessrepo test updates already in candidate).
- No stray `--help`/mutation-source tree in the candidate diff;
  mutant scratch lives under ignored `.temp/` and `/tmp/` only.
