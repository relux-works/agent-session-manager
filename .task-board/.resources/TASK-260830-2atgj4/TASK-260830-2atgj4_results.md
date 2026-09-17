# TASK-260830-2atgj4 results: property-test the ownership reducer (final leaf)

Role: developer. Worktree: `.temp/STORY-260830-1oqfec/worktree` on
`task-board/story/STORY-260830-1oqfec`, candidate left UNCOMMITTED
for the handoff snapshot. Status at handoff: ready for review.

## Outcome

The four story-pinned ownership invariants hold as 11 executable
property tests over the landed production reducers
(`sessstate.Reduce`/`Compare`, `sessrepo.WinningLease`/`ListLeases`,
`fencing.Authorize*`), with hand-written closed-alphabet generators
(no new module dependency), exhaustive enumeration where the space is
small, and bounded seeded-random exploration otherwise. The
union-order property exposed one genuine defect on the landed
reducer and fixed it minimally with a RED-first regression (below).
The falsification battery kills 8 narrowing plants through the named
properties with an applied harmless SURVIVED control. Story-close
items are carried: three traceability acceptance cases bound to the
real reducers with honest coverage text and a re-pinned registry
digest, the section:17.2 gap reword, the 3g12yp P3 vector, README and
LOGBOOK updates, and the full configured validation suite green.

## AC coverage: 18 of 18 rows driven, 0 stated bounds on behavior

Every row is driven through the production entry point by a named
committed test. Crash/idempotency: the changed derivation
(`resolveWinner`) is pure with no durable mutation, so no new crash
window exists; the landed lease crash/idempotency tests remain green
(batch A). No unsupported capability is advertised (claims gate
green; README adds no `ax`, `doctor`, or runtime claim).

| # | AC row | Production call site | Named test(s) |
| --- | --- | --- | --- |
| 1 | Union permutations project byte-identically | `sessstate.Reduce` | `TestOwnershipUnionOrderIndependent`, `TestOwnershipUnionOrderRegression` |
| 2 | Successive-union partitions project byte-identically | `sessstate.Reduce` | `TestOwnershipUnionPartitionStable` |
| 3 | Stale (lower-epoch) loser never dropped/rewritten/promoted | `sessstate.Reduce`; `sessrepo.GetLease`, `ListLeases` | `TestOwnershipLoserHistoryPreserved`, `TestOwnershipStoreLoserBytesPreserved` |
| 4 | Same-epoch loser never dropped/rewritten/promoted, tie named | `sessstate.Reduce` | `TestOwnershipLoserHistoryPreserved` |
| 5 | Winner is the greatest tuple; `Compare` is the total order | `sessstate.Compare` | `TestOwnershipWinnerIsTupleMaximum` |
| 6 | `created_at` never influences store authority or fencing verdicts | `sessrepo.CreateLease`, `CompareAndSwapLease`, `WinningLease`, `VerifyFencingToken` | `TestOwnershipStoreClockNonAuthority` |
| 7 | Absolute wall-clock position never influences gate verdicts | `fencing.Authorize*` (all five entries) | `TestOwnershipGateClockNonAuthority` |
| 8 | At most one authoritative lease per session and epoch | `sessrepo.WinningLease`, `ListLeases`, `CompareAndSwapLease`, `GetLease` | `TestOwnershipStoreSingleAuthorityPerEpoch` |
| 9 | Gates admit exactly the winner; pairwise exclusion | `fencing.Authorize*` (all five entries) | `TestOwnershipGateSingleAuthorizedOwner` |
| 10 | Two handles/processes share one head; lagging CAS refuses stale | `sessrepo.CompareAndSwapLease`, `WinningLease` | `TestOwnershipStoreTwoHandlesOneHead` |
| 11 | Exact contract fixtures pass | All entries above | Full suite green (batches A/B/C, 27 packages) |
| 12 | Negative/refusal cases pass | All gates above | Full suite green; landed negatives untouched and green |
| 13 | Crash/idempotency evidence where durable state mutates | (no new durable mutation) | Landed `TestCrashBeforeLeaseWriteIsSafeRetry`, `TestCrashAfterLeaseCommitCountsAsCommitted`, idempotent-retry tests green |
| 14 | No unsupported capability advertised | README scan posture | `TestRealREADMECarriesNoPositiveClaim` et al. green |
| 15 | P3 vector: epoch-zero third lease refuses `invalid_arguments` | `fencing.Authorize*` (all five entries) | `TestAuthorizeRefusesMalformedPresented/epoch_zero_other_lease` |
| 16 | section:17.2 gap reworded to the narrower reader-enum meaning | `internal/traceability/ownership.v0.6.0.json` | `go run ./internal/traceability/cmd/tracecheck` exit 0 |
| 17 | Traceability cases bound with honest coverage, digest re-pinned | Registry + `reviewedOwnershipCanonicalSHA256` | `TestVerifyRepositoryAcceptsExactOwnership` (113 cases), tracecheck pins |
| 18 | Every property falsified by a narrowing mutant with controls | `internal/sessstate/testdata/mutate_properties.py` | 8 KILLED by named properties; harmless SURVIVED; harness exit 0 |

## Defect found by the property (RED-first)

`resolveWinner` reported each losing union lease against the transient
arrival-order winner: with union `{R@1, A@1}` on an empty chain, one
arrival order records R's loss and the other drops it entirely, and
multi-entry unions name different losers-to targets per order. The
winner was always right; the divergent-history evidence was not a
function of the multiset. Fix (`internal/sessstate/sessstate.go`
only): select the greatest tuple in a first pass, then report every
entry below it in ascending tuple order naming the final winner, with
the off-chain flag recomputed from the final winner (proven
equivalent to the tracked flag in all cases). RED: 4 of 5 property
tests fail on the landed code (`red-ownership-properties.log`).
GREEN: 5 of 5 pass after the fix (`green-ownership-properties.log`).
All landed union assertions were order-free, so the only collateral
was the refusal-census line pins (sessstate.go:924/927 to :929/:932).

## Files changed (candidate, uncommitted)

- `internal/sessstate/sessstate.go`: two-pass `resolveWinner` (the fix).
- `internal/sessstate/ownership_properties_test.go` (new): P1/P2
  properties (5 tests).
- `internal/sessrepo/lease_properties_test.go` (new): P3/P4 store
  properties (4 tests).
- `internal/fencing/ownership_properties_test.go` (new): P3/P4 gate
  properties (2 tests).
- `internal/fencing/gate_test.go`: P3 vector
  `epoch_zero_other_lease`.
- `internal/sessstate/testdata/mutate_properties.py` (new): the
  falsification battery.
- `internal/sessstate/census_state_test.go`: census line re-pin.
- `internal/traceability/ownership.v0.6.0.json`,
  `internal/traceability/traceability.go` (digest re-pin),
  `internal/traceability/traceability_test.go` (113),
  `internal/traceability/cmd/tracecheck/main_test.go` (113 x2):
  registry story-close.
- `README.md`: property-suite section, harness table row, 113 figure.
- `LOGBOOK.md`: task entry.

## Validation (all local, real exits, this tree)

- `go vet ./...`: exit 0. `GOOS=windows go vet ./...`: exit 0.
  `gofmt -l internal`: empty. `go build ./...`: exit 0.
  `GOOS=windows go build ./...`: exit 0.
- `go run ./internal/traceability/cmd/tracecheck`: exit 0
  (`acceptance_cases=113`, `clauses_discharged=28/485`).
- Catalog directive pinned (1 exact `//go:generate`) +
  `go generate ./internal/catalog` + clean diff: exit 0.
- cigate contract gates (6 tests) and claim gates (6 tests): all PASS.
- Test batches A (sessstate/sessrepo/fencing/sessquery), B (12 pkgs),
  C (11 pkgs): 27/27 `ok`, 0 FAIL.
- Race batches (same splits): 27/27 `ok`, no `DATA RACE`.
- Coverage batches: every package reports a number; sessstate 92.4%,
  sessrepo 86.6%, fencing 97.5% (reported, no floor).
- Fixture matrix: secprim 51 PASS, secconftest 59 PASS, skips loud.
- Fuzz smoke: 13/13 derived targets print `fuzz: elapsed`.
- Mutant battery: 8 KILLED by named properties, harmless SURVIVED,
  harness exit 0.
- `git status` at handoff: only the 13 candidate paths above (plus
  ignored `.temp/` evidence); no `__pycache__`, no `.pyc`.

## Trunk drift (refresh-candidate outcome)

`origin/main` moved during the run from `62d4463` to `e4e3e88`
(STORY-260917-158jyi: release-agnostic cataloggen `-adopted` check,
plus its board-state record). `task-board worktree refresh-candidate
TASK-260830-2atgj4` replays the story checkpoints in isolation and
stops at a mechanical LOGBOOK append-append conflict at the 2f5393
replay (trunk's 3lt2xv entry vs the 2f5393 entry, both under
`## 2026-09-17`); trunk's entry is newer (11:05 vs 09:17 UTC+4), so
the correct resolution keeps trunk's entry on top. The
`--replay-resolutions` file schema is not documented on any surface
available here: `path`/`sha256(base64 content)` parse, but every
checkpoint-binding field name tried is refused as unknown while the
binding itself is required (`replay resolution checkpoint is outside
the original sequence`). The candidate worktree is verified intact
(HEAD `1b8e75a`, all 13 candidate paths, `task-board.config.json`
unmodified and equal to HEAD). Integration is the orchestrator's
step; notes for it: (a) trunk touches none of my Go paths (only
`internal/catalog*`, board files, README catalog rows, one LOGBOOK
entry); (b) the only same-file adjacencies are LOGBOOK top (keep all four
entries, newest-first by wall time: this 2atgj4 entry, trunk 3lt2xv
11:05, 3g12yp 11:01, 2f5393 09:17, all UTC+4 2026-09-17) and the
README tools table (trunk rewrote the Go-toolchain
row's cataloggen command; I add a new row after the fencing row —
keep both); (c) my evidence quotes the `-adopted`-neutral commands
only (`go generate`, tracecheck), never command 21's explicit form.

## Evidence attached

- `TASK-260830-2atgj4_results.md` (this file)
- `TASK-260830-2atgj4_conformance-matrix.md`
- `TASK-260830-2atgj4_producer-evidence.tar.gz` (real gzip: RED/GREEN
  logs, affected-package logs, validation batch logs, coverage logs,
  mutants/ raw logs + mutants.json)
