# TASK-260830-1geqhj rev4 — producer results

Task: implement `ax pane` SESSION_ID enforcement wrapper
(rev4 rework of review-verdict-rev3: 1 P1 class at two sites with
one root cause, 1 P2, 8 P3). Trunk: `2fc6d50`
(`internal/specdoc/SPEC.v0.7.0.md`, headings byte-identical to
v0.6.0 in scope); `origin/main` verified equal to the candidate
base before handoff, so no refresh was needed. Candidate left
UNCOMMITTED in the Story worktree for handoff snapshot. No
partial-attempt recovery was needed: the worktree held the rev3
candidate UNCOMMITTED (`M LOGBOOK.md`, `M README.md`,
`?? internal/axpane/`) with no commit past the checkpoint.

## Finding-by-finding table

| Finding | Change (production call site) | Test that fails without it | Evidence path |
| --- | --- | --- | --- |
| P1-1 agreement parks the post-takeover lifecycle | Deleted `leaseCheckpointAgrees` from `checkMaterialization` and `admitCheckpoint` (`decide.go`); added the one implication arm directly after the fencing arm in `Decide`: checkpoint-carrying winner + null fold parks `restore_policy` (`errSuccessorNullFold`) | `TestDecidePostTakeoverDivergedBaseLaunches`, `TestRunPostTakeoverRestoreLaunchesWithNewestClosure` (launch under the fix; parked before it), `TestDecideSuccessorNullFoldParks`, `TestRunSuccessorNullFoldParks`, `TestRunPostTakeoverStaleBaseParks` | `axpane-test-v.log`; `mutants/pass1/N-successor-nullfold.log` |
| P1-1 profile closure from the handoff base | `Run` loads the closure from the checkpoint actually resumed — required checkpoint, else the fold's newest via `LoadCheckpoint`, else the session head — never `Winner.Checkpoint` (`run.go`); the unknown-store guard now keys on the published newest | `TestRunPostTakeoverLaunchCarriesNewestClosureStandardToYolo`, `TestRunPostTakeoverLaunchCarriesNewestClosureYoloToStandard` (both directions through `Run`), `TestRunNoCkptStorePublishedNewestFails`, `TestRunNewestAbsentFromStoreFails` | `axpane-test-v.log`; `mutants/pass1/N-closure-source.log` (mutant re-reads `Winner.Checkpoint`, launches stale yolo, dies) |
| P2-1 post-window same-pair race reports launch | `Store.Supersede`/`installPair` report the replay (`binding.go`, `(Binding, bool, error)`); `Run` flips the replayed post-window outcome to `ActionReattach` with the recorded receipt (`run.go`) | `TestRunPostWindowSamePairRaceReattaches` (hook-played race through `Run`); store-level `TestSupersedeConcurrentSamePairReplays` asserts the signal | `axpane-test-v.log`; `mutants/pass1/N-supersede-replay.log` |
| P3-1 window closes on the lease checkpoint | No production change (the predicate already reads the fold only); added the negative and the harness row | `TestRunWindowLeaseCheckpointNullFoldRefuses` | `mutants/pass1/N-window-lease-checkpoint.log` |
| P3-2 `Run` swallows the fold error | No production change (the error already propagates); added the negative and the harness row | `TestRunUnfoldableChainFails` | `mutants/pass1/N-run-fold-error.log` |
| P3-3 ratio 39/43 | Re-derived on the fixed tree (see below) | — (measurement, not a gate) | `ratio-check.log` |
| P3-4 fabricated 27-command list | This document reports the configured `spawn.worktree_isolation.validation.commands` verbatim with real exits (see below) | — | `cmd/cmd01-*.log` … `cmd/cmd27-*.log` |
| P3-5 phantom matrix names | rev4 matrix names only tests verified present via `go test -list` | — | `ratio-check.log` |
| P3-6 `realmRowUsable` restates reconcile | Stated as a bound (code comment + TRACEABILITY): the aggregate `Reconcile` error carries no per-object identity and liveness shares the generic mismatch code, so no per-row verdict can be derived; classification-only, fail-closed both directions | `TestDecideExpiredRealmRowRefusesUnavailable`, `TestDecidePreRebootRealmRowRefusesUnavailable`, `TestDecideLiveRealmRowBackendFaultStands` (unchanged) | `axpane-test-v.log` |
| P3-7 identity not bound to SESSION_ID | `checkProviderIdentity` binds the validated record's own `session_id` (`checkIdentitySession`, `decide.go`); foreign record refuses `invalid_config` | `TestDecideIdentityForeignSessionRefuses` | `axpane-test-v.log`; `mutants/pass1/N-identity-session.log` |
| P3-8 false prose | README profile sentence, TRACEABILITY rows, and the rev3 LOGBOOK entry (explicit REVERSAL paragraph, following the rev2 precedent) corrected | — | candidate diff |

Stated bound added with P1-1: binding the handoff base to a
checkpoint the chain published is the takeover leaf's obligation —
this leaf compares the base against nothing (TRACEABILITY.md,
`Decide` comment).

## AC coverage: 43 of 43 rows driven

Measured, not restated: 43 data rows counted in
`internal/axpane/TRACEABILITY.md`, every named test verified
present via `go test -list`, zero rows without a named test, and
the full package suite green (`ratio-check.log`,
`axpane-test-v.log`: 192 `--- PASS`, 0 `--- FAIL`). The four rows
the rev3 reviewer measured unfaithful now pin the post-takeover
launch positives: materialization and checkpoint rows drive the
C1-sourced launch and the stale-base park, the after-restore row
drives both owners, and the profile row drives both directions
plus the restore closure through `Run`.

## Negative and mutant evidence

- 11 new committed tests in `internal/axpane/rev4_test.go`, 3
  rewritten in `rev3_test.go` (agreement negatives became the
  diverged-base positive and the null-fold negative; the no-store
  guard test follows the newest), plus replay-signal assertions
  on the `Supersede` tests.
- `mutant_harness.py`: 53/53 rows match on two full passes — 52
  KILLED + the SURVIVED control. New rows: `N-successor-nullfold`
  (null-fold conditional), `N-closure-source` (re-reads
  `Winner.Checkpoint`), `N-supersede-replay` (replay keeps
  launch), `N-window-lease-checkpoint`, `N-run-fold-error`,
  `N-identity-session`, `N-run-newest-absent`; re-anchored:
  `N-run-ckpt-store`, `N-supersede-no-replace`; deleted with the
  equality: `N-materialization-agreement`, `N-checkpoint-agreement`.
  Per-plant raw logs with subprocess exits in `mutants/pass1/`
  (`PYTHONDONTWRITEBYTECODE=1`, no `__pycache__`); production
  blobs verified byte-identical before/after (`blob-oids-*.txt`).
- Crash/idempotency: unchanged no-replace link commits on both
  install paths, hook-armed seams, real-SIGKILL test; the replay
  signal adds no new durable write.
- No token-preserving source-text mutant applies: no gate
  inspects source text.

## Validation suite (configured commands, in order)

Every command below is quoted verbatim from
`task-board.config.json`
`spawn.worktree_isolation.validation.commands` (27 commands) and
was executed once against the exact candidate tree; logs in `cmd/`.

| # | Command | Exit |
| --- | --- | --- |
| 1 | `test -z "$(gofmt -l $(git ls-files --cached --others --exclude-standard -- '*.go'))"` | 0 |
| 2 | `go build ./...` | 0 |
| 3 | `go vet ./...` | 0 |
| 4 | `go test ./... -count=1 -v` | 0 (38 packages ok, 21766 `--- PASS`, 0 package `FAIL`) |
| 5 | `go test ./... -race -count=1 -timeout 25m` | 0 (38 ok, 0 data races; axpane 17.5s) |
| 6 | `go test ./... -cover -count=1` | 0 (axpane 80.9%) |
| 7–12 | rpcwire/scalar/canonicaljson fuzz ×6 (`-fuzztime=100x`) | 0 ×6 |
| 13–20 | secconftest fuzz ×8 (`-fuzztime=100x`) | 0 ×8 |
| 21 | `go run ./internal/traceability/cmd/tracecheck` | 0 (registry untouched: contracts=64, sections=36, clauses 49/569) |
| 22 | `go run ./internal/catalog/cmd/cataloggen -adopted -output internal/catalog/catalog_gen.go -check` | 0 |
| 23 | `GOOS=linux GOARCH=amd64 go build ./...` | 0 |
| 24 | `GOOS=windows GOARCH=amd64 go build ./...` | 0 |
| 25 | JSON validation over tracked `*.json` | 0 |
| 26 | `task-board validate` | 0 (185 pre-existing board-hygiene notes on other elements, unrelated to this candidate) |
| 27 | `git diff --check` | 0 |

Note on cmd4: the log contains indented `--- FAIL` lines that are
nested expected-failure output inside passing tests (e.g. the
resumesmoke negative demonstrations); every package reports `ok`
and the command exits 0.

## Files changed

- `internal/axpane/` (new package, still untracked):
  `decide.go` (null-fold arm, agreement deleted, identity session
  bind), `run.go` (closure source, newest-based guard,
  replay flip), `binding.go` (replay signal), `errors.go`
  (null-fold + identity causes), `doc.go`, `rev4_test.go` (new),
  `rev3_test.go`, `conformance_test.go`, `mutant_harness.py`,
  `TRACEABILITY.md` (rewrites above).
- `README.md`: package section (additive: implication, closure
  source, identity bind, 52-mutant count).
- `LOGBOOK.md`: rev4 entry + explicit rev3 REVERSAL (additive).
- `internal/traceability` untouched per the Story contract.

## Stated bounds (no unsupported claims)

No `ax` command, doctor result, or runtime capability is added;
mesh refresh is the caller-reported `Verified` fact; the
dead-realm mapping fires only when the caller needs the realm
row, submitted a realm row, and none is usable; the handoff-base
binding is the takeover leaf's obligation; the realm usability
predicate copy is classification-only. `internal/traceability`
untouched per the Story contract (final leaf re-pins).

## Handoff

Ready for review. Candidate UNCOMMITTED in the Story worktree;
checklist updated on the board; outcome resources attached:
`TASK-260830-1geqhj_results-rev4.md`,
`TASK-260830-1geqhj_conformance-matrix-rev4.md`,
`TASK-260830-1geqhj_rev4-evidence.tar.gz`.
