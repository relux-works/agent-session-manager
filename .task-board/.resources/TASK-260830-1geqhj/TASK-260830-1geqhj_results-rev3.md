# TASK-260830-1geqhj rev3 — producer results

Task: implement `ax pane` SESSION_ID enforcement wrapper
(rev3 rework of review-verdict-rev2: 1 P1 class, 2 P2, 8 P3).
Trunk: `2fc6d50` (`internal/specdoc/SPEC.v0.7.0.md`, headings
byte-identical to v0.6.0 in scope). Candidate left UNCOMMITTED in
the Story worktree for handoff snapshot.

## Pre-existing worktree state (partial-attempt recovery)

On entry the worktree already held the rev2 candidate UNCOMMITTED:
untracked `internal/axpane/` plus additive hunks in `LOGBOOK.md`
(+17) and `README.md` (+56). No commit existed past the
checkpoint. I read `TASK-260830-1geqhj_review-verdict-rev2.md`
(CHANGES REQUESTED) and the review-evidence archive before
touching code, then ran
`task-board worktree refresh-candidate TASK-260830-1geqhj` →
`refresh_advanced`, trunk `2fc6d50`. The refresh left old-base
content in trunk-moved paths, so I restored `HEAD` content for
every trunk-moved path and re-applied only my additive hunks
(verified: `git diff` shows 0 deletions across `LOGBOOK.md` /
`README.md`, `task-board.config.json` == `HEAD`). Final
`git status`: `M LOGBOOK.md`, `M README.md`,
`?? internal/axpane/` — candidate paths only.

## Root-cause fixes (production call sites)

- P1-1 (window/admission/journal keying): the window predicate,
  checkpoint admission, and journal source binding read the chain
  fold's newest checkpoint (`Input.HasNewestCheckpoint` /
  `NewestCheckpointID`, loaded by `Run` via
  `LoadNewestCheckpoint` → `loadFoldInput` + `sessstate.Reduce`),
  never `Winner.HasCheckpoint`/`Winner.Checkpoint`. A successor
  lease's handoff base must agree with the fold
  (`leaseCheckpointAgrees`, `decide.go`); disagreement or a null
  fold with a requirement parks `restore_policy`. A null fold
  parks; the only launch-class path left in that state is the
  bootstrap retry without a checkpoint.
- P2-1 (receipt retention): the store keeps one receipt per pair
  (`Store.Lookup`, `receipts/<op>.json`, no-replace link commit)
  with `binding.json` as the first-binding anchor;
  `Store.Supersede` is no-replace per pair plus anchor claim, and
  `Store.SessionBound` (anchor-or-receipts) drives the
  changed-operation arm with the fold window. All four §4.1
  vectors hold: any recorded pair replays, concurrent
  different-op supersedes each record, identical retry replays.
- P2-2 (unknown checkpoint store): `Run` fails when the winner
  carries a checkpoint and `stores.Ckpt == nil` (`run.go`,
  mirroring the `CheckpointRequired` guard) — an absent store is
  unknown, never "no closure".
- P3-1: losing-lease emission test uses a foldable (failed) chain
  and asserts the `AuthorizeMutation` refusal itself (`not_owner`
  remote, `stale_owner` local successor), plus the `N-emit-losing-epoch`
  (RV7) narrowing row.
- P3-2: launch-mode create-from-stopped post-window restore
  through `Run` (`TestRunCreateFromStoppedPostWindow`).
- P3-3: ratio re-derived by counting: 43/43 (see below).
- P3-4: the §4.C credential conditional applies to every caller
  with `Smoke.Required`, not only background (`checkRealm`).
- P3-5: the realm cross-bind accepts when ANY realm object binds
  (`checkRealmBinding` scans; pinned in both evidence orders).
- P3-6: dead (expired/pre-reboot) realm evidence refuses the
  typed `capability_unavailable` with the reconcile failure as
  cause (`deadRealmRefusal`); a usable row plus an unrelated
  backend fault keeps the backend class (pinned).
- P3-7: misleading test renamed to
  `TestRunRemoteNonInteractiveParksWithoutEmission`;
  `TestRunPostWindowSupersedes` rewritten (sourced journal,
  chain-published checkpoint, `Lookup` assertions).
- P3-8: the rev2 LOGBOOK entry explicitly reverses rev1 F1
  (remote offers author nothing).

## AC coverage: 43 of 43 rows driven

Every row of the acceptance table in
`internal/axpane/TRACEABILITY.md` (43 data rows, counted) is
driven through the production entry named in its call-site
column: `Decide` (pure core), `Run` (orchestration), `EmitParked`
/ `Emit`, `Store.Bind` / `Supersede` / `Lookup` / `SessionBound` /
`Status`, `LoadNewestCheckpoint`, `LoadCheckpoint`,
`LoadMaterialization`, `LoadProfile`, `ObserveOwnership`,
`AdmitProviderDescriptor`, `BuildDescriptor`. The four rows the
reviewer measured unfaithful in rev2 (materialization validity,
checkpoint admission, after-restore sequence, bootstrap
idempotency) are re-pinned: positives carry the chain-published
checkpoint; negatives drive the epoch-1 owner.
`TestConformanceMatrixCompleteness` pins the clause→test map
(extended with 5.7 newest, per-pair retention, credential
conditional, dead-realm, order-independence, unknown-store rows).

## Negative and mutant evidence

- 21 new committed tests in `internal/axpane/rev3_test.go` plus
  re-pinned positives; every gate ships narrowing mutants.
- `mutant_harness.py`: 48/48 rows match — 47 KILLED + the
  SURVIVED control, with raw per-plant logs and subprocess exits
  in `mutants.log` (evidence tar). New rows: `N-window-fold`,
  `N-checkpoint-nullfold`, `N-materialization-nullfold`,
  `N-checkpoint-newest`, `N-checkpoint-agreement`,
  `N-materialization-agreement`, `N-idempotency-bound`,
  `N-realm-foreground`, `N-realm-deadmap`,
  `N-supersede-no-replace`, `N-run-ckpt-store`,
  `N-emit-losing-epoch`, `N-bind-first`.
- Crash/idempotency: no-replace link commits on both install
  paths, hook-armed seams, real-SIGKILL test, anchor→pair crash
  convergence; parked events append through the `sessrepo`
  owner (linkage/round-trip proven here).
- No token-preserving source-text mutant applies: no gate
  inspects source text.

## Validation suite (task-board.config.json, in order)

| # | Command | Exit |
| --- | --- | --- |
| 1 | `go test ./... -count=1` | 0 |
| 2 | `go test ./... -cover -count=1` | 0 (axpane 80.6%) |
| 3 | `go vet ./...` | 0 |
| 4 | `gofmt -l $(git ls-files '*.go')` | 0 (empty; plus `gofmt -l internal/axpane` clean for untracked) |
| 5 | `go test -race ./... -count=1` | 0 |
| 6–8 | catalog consistency/docs/capability-docs | 0/0/0 |
| 9 | doctor bundle | 1 — PRE-EXISTING: `internal/doctor` absent at HEAD `2fc6d50` (setup failure, no package to load); unrelated to this candidate, which adds no doctor surface |
| 10–15 | traceability projection/docs/ownership, specdoc, v06, v07 | 0 ×6 |
| 16–18 | bundle install scripts | skipped-dirty, exit 0 (cmd16 additionally never arms: `DIRTY` unset in the suite) |
| 19–27 | catalog locking/concurrency/tx, provhost ×3, terminalbackend ×3 | 0 ×9 |

Full logs in
`.temp/TASK-260830-1geqhj/rev3-evidence.tar.gz` (`cmd01`–`cmd27`,
`mutants.log`, manifest with SHAs, candidate diff).

## Files changed

- `internal/axpane/` (new package): `decide.go`, `run.go`,
  `binding.go`, `events.go`, `gates.go`, `errors.go`, `doc.go`,
  `descriptor.go`, `sync.go`, `*_test.go` (incl. new
  `rev3_test.go`), `TRACEABILITY.md`, `mutant_harness.py`.
- `README.md`: package section (additive, updated to rev3).
- `LOGBOOK.md`: rev3 entry + explicit rev1-F1 reversal (additive).

## Stated bounds (no unsupported claims)

No `ax` command, doctor result, or runtime capability is added;
mesh refresh is the caller-reported `Verified` fact; the
dead-realm mapping fires only when the caller needs the realm
row, submitted a realm row, and none is usable; tuple-gate
defense-in-depth is stated in TRACEABILITY.md. `internal/traceability`
untouched per the Story contract (final leaf re-pins).

## Handoff

Ready for review. Candidate UNCOMMITTED in the Story worktree;
checklist updated on the board; outcome resources attached:
`TASK-260830-1geqhj_results-rev3.md`,
`TASK-260830-1geqhj_conformance-matrix-rev3.md`,
`rev3-evidence.tar.gz`.
