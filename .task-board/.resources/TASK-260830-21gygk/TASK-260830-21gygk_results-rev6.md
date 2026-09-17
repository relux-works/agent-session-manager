# TASK-260830-21gygk — rev6 rework outcome (rev5 verdict changes-requested)

Producer rework for `TASK-260830-21gygk_review-verdict-rev5.md` (CR5 tree
`ab3a8894386128681aa4e9dce451cb43c261db3e`, base `8cf4aaa...`, story
checkpoint `83640d191f78f0cd685e5f709027819799efe1cf`). Both P1 findings
are fixed on the shared semantic admission path. Candidate left
UNCOMMITTED in the Story worktree for CR snapshotting; no branch
mutation, no integration, no hosted CI.

## Failing baseline (pre-fix, ported reviewer regressions)

Copied `/tmp/rev5ev/reviewer_rev5_test.go` to
`internal/sessquery/rev5_regression_test.go` and ran before any
production change (`go test ./internal/sessquery -run '^TestRev5'`):

- `TestRev5AncestorCheckpointAuthority/missing_ancestor_checkpoint`
  FAIL: BuildPlan admitted, Revalidate(old plan and new) nil,
  AuthoritativeStatus/List admitted.
- `TestRev5CheckpointCreatorMustBeHolder/non_holder` FAIL: all four
  entries admitted.
- Both `complete`/`holder` positive controls PASSED.
- Raw log: worktree `.temp/rev5-failing-baseline.log` (ignored scratch).

## Production fix (`internal/sessquery/lease.go` only)

- `validatedCheckpoint` gains `CreatorHostID`; `parseCheckpointRecord`
  parses `created_by_host_id` as UUIDv7 (`invalid_config` when malformed).
- `checkLeaseChain` now returns the winner-to-root ancestry chain in
  addition to the direct predecessor (walk semantics unchanged).
- `checkWinnerCheckpoint` additionally enforces checkpoint creator ==
  owning lease holder (predecessor holder at epoch > 1, self at epoch 1).
- New `checkAncestorCheckpoints` + `checkCheckpointBinding` validate
  every non-winner lease of the winning ancestry: epoch > 1 must
  reference an admitted checkpoint for its session and predecessor
  lease with creator == predecessor holder; the epoch-1 root may omit
  the reference, otherwise it must bind to itself with creator == its
  holder. Unresolvable/misbound references refuse
  `selector_observation_unavailable`. Only the winner's ancestry is
  consulted, so legal branching, lagging copies, canonical identity,
  and parked-source refusal are preserved.
- `winningLeaseFor` (shared by BuildPlan/bindPlan, Revalidate,
  authorize/AuthoritativeStatus/AuthoritativeList) runs all three
  checks, so old-plan invalidation after loss of ancestor checkpoint
  falls out of the same gate.

## Committed tests

- Ported reviewer tests kept in `internal/sessquery/rev5_regression_test.go`.
- Added `TestRev5AncestorCheckpointBinding` (wrong-session, wrong-lease,
  wrong-creator ancestor checkpoints, each through BuildPlan,
  Revalidate, AuthoritativeStatus, AuthoritativeList) with shared
  `rev5AncestorFixture` 3-generation control.
- Post-fix: `go test ./internal/sessquery -run '^TestRev5'` — all pass.

## Narrowing mutants (focused battery, same instrument as committed `testdata/mutate.py`)

Driver + raw logs + manifest: worktree
`.temp/TASK-260830-21gygk/mutants-rev5/` (ignored scratch;
`mutants-rev5.json` manifest, per-plant `.log` files with real exits).
Each plant keeps the gate and admits exactly one member; files restored
with digest verification; controls green.

| Mutant | Classification | Killed by |
| --- | --- | --- |
| N-ancestor-checkpoint (skip leaseB ancestor) | KILLED (exit 1) | TestRev5AncestorCheckpointAuthority(/missing...), TestRev5AncestorCheckpointBinding(/wrong_session/wrong_lease/wrong_creator) |
| N-checkpoint-holder (admit hostB creator, winner path) | KILLED (exit 1) | TestRev5CheckpointCreatorMustBeHolder(/non_holder) |
| N-ancestor-holder (admit hostB creator, ancestor path) | KILLED (exit 1) | TestRev5AncestorCheckpointBinding(/wrong_creator) |
| C-harmless-comment | SURVIVED (exit 0, applied log) | control — behavior-preserving |
| control-before / control-after | CONTROL exit 0 | — |

The two holder plants kill disjoint test sets, proving the two guard
sites are distinct and both required. The same three plants were added
to the committed `internal/sessquery/testdata/mutate.py` battery; every
lease.go anchor in the full battery counts exactly 1 occurrence in the
final source (N-chain-self, N-checkpoint-placeholder, N-lease-missing,
C-harmless-comment, plus the 3 new).

## Validation (all observed this run)

- `go build ./...` — exit 0.
- `go vet ./internal/sessquery/ ./internal/sessrepo/ ./internal/provhost/` — exit 0.
- `gofmt -l internal/` — clean, no files listed. NOTE correcting the
  prior evidence description: the old `validation-final.log` gofmt
  failure was exit 2 because it passed a nonexistent `cmd/` directory;
  the valid command is `gofmt -l internal/` (no `cmd/` exists in repo).
- `git diff --check -- internal/ README.md` — exit 0.
- `go test ./... -count=1` — all packages ok (sessquery, sessrepo,
  sessstate, provhost, canonicaljson, specdoc, traceability, etc.).
- Coverage: sessquery 87.9%, sessrepo 87.2% of statements.
- No durable writes on these paths (read-only selector leaf); no
  crash/idempotency protocol change — retained creation-crash
  composition tests still green.

## Docs / traceability

- `README.md` and `internal/sessquery/TRACEABILITY.md`
  ("fully validated succession") now state whole-ancestry plus holder
  binding; TRACEABILITY row 45 names the new tests and mutants.
- `LOGBOOK.md` carries the rev6 rework entry.

## AC coverage: 8 of 8 rows driven through shared production entries

Reader.Resolve (UUID/name: TestResolvePinnedPrecedence,
TestResolveBareIdentityUnion; qualified: TestResolveQualifiedSourcesSelectOneIndex,
TestResolveQualifiedSourceNeverFallsBack, TestResolveExplicitSourceReadFailures;
ambiguity: TestResolveExactNamesAndASCIICollisions; sorting:
TestListStatusDerivedFactsAndStableOrder, TestResolvePeerOrderAndReplicatedIdentity),
Reader.BuildPlan/Revalidate/ParsePlan (TestBuildPlanBindsCurrentFacts,
TestRevalidateDetectsEachFactChange, TestValidSuccessorBuildsAndRevalidates,
TestRevalidateUnionParkedCopyRefuses + rev5 ancestry/holder suites),
Reader.AuthoritativeStatus/List (TestAuthoritativeStatusHealthy,
TestAuthoritativeMissingProcessRefusesStatus,
TestAuthoritativeHostNameBound64, TestAuthoritativeAdmitsFullCapabilityRegistry
+ rev5 suites through both entries). Public CLI execution remains the
stated caller-integration bound (0 CLI rows in this leaf).

## Candidate paths (uncommitted, Story worktree)

- `internal/sessquery/lease.go` (fix)
- `internal/sessquery/rev5_regression_test.go` (ported + new tests)
- `internal/sessquery/testdata/mutate.py` (3 narrowing plants)
- `internal/sessquery/TRACEABILITY.md`, `README.md`, `LOGBOOK.md` (docs)
- Retained prior CR5 scope untouched (sessrepo checkpoint/lease owners,
  provhost/sessrepo test updates).
- No stray `--help`/mutation-source tree in the candidate diff;
  mutant scratch lives under ignored `.temp/` only.
