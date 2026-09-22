# BUG-260917-2fwf8e results — revision 3

Status: ready for review. The candidate is intentionally uncommitted in the
managed Story worktree; the handoff snapshot owns the commit.

## Outcome and measured ratios

`internal/fencing.Authorize` now validates the refresh policy, evaluates
ownership direction, and only then applies the saved lapsed-grant refusal. A
remote winner with a lapsed local grant therefore reaches the existing
`remote_owner` park with literal `not_owner`, rather than `lease_conflict`.
The existing `axpane.Decide` wrapper can consequently emit `attach_remote` or
`takeover_offer` for an interactive caller and `parked` for a non-interactive
caller. A local lapsed grant remains literal `lease_conflict`; a remote
unusable policy remains literal `invalid_arguments`.

The three task Acceptance Criteria rows are driven: **3 of 3**. The six
task-specific behavior rows called out by the producer/reviewer evidence are
also driven: **6 of 6**. The Authorize gate census is **5 of 5 gate entries**
with a named test plus an admitting narrowing; the `axpane` `ModeLaunch`
wrapper cell is an explicit **BOUND**, because the shipped arm-swap row is
masked to `./internal/fencing` (the full importer-set reviewer probe still
drives the wrapper cell). No lease-chain implementation or park vocabulary
changed.

## Reviewer probe 10 before/after

Source archive: `TASK-260830-1geqhj_review-evidence-rev1.tar.gz`, SHA-256
`9f473449f33fda2be591f23396da2b9b9b995ba18708b02afdff7a550e840ef4`.
The baseline was run from checkpoint `799c338e401fca0b24859c0b870cd665927209f2`.

| Production surface | Before fix | After fix |
| --- | --- | --- |
| Direct `fencing.Authorize` | `refused / lease_conflict` | `parked / remote_owner`, winning lease retained, cause `not_owner` |
| `axpane.Decide`, restore, interactive, attach admitted | `refused / lease_conflict` | `attach_remote / remote_owner` |
| `axpane.Decide`, restore, interactive, attach not admitted | `refused / lease_conflict` | `takeover_offer / remote_owner` |
| `axpane.Decide`, restore, non-interactive | `refused / lease_conflict` | `parked / remote_owner` |

Evidence: `logs/baseline-axpane-probe10-neighbours.log` and
`logs/candidate-axpane-probe10-neighbours.log`; the direct five-entry grid is
in `logs/baseline-fencing-grid.log` and `logs/candidate-fencing-grid.log`.
The archived probe source refers to the removed `Realm.AttestedServer` field
and cannot compile unchanged on this tree. The current minimal probe and the
committed direct regression reproduce the same vector; that API drift is an
explicit bound, not an inferred absence.

## Neighbour vectors

`TestAuthorizeArmOrderingNeighbourVectors` drives the direct gate and
`TestDecideLapsedRemoteOwnerOffersOrParksByInteractiveContext` drives the
wrapper context.

| Vector | Direct `Authorize` result | Wrapper result |
| --- | --- | --- |
| Live local grant | `MINT` | existing launch path |
| Remote owner, live grant | `PARK(remote_owner, not_owner)` | existing remote-owner path |
| Remote owner, lapsed grant (reported vector) | `PARK(remote_owner, not_owner)` | `attach_remote` / `takeover_offer` |
| Remote owner, lapsed grant, non-interactive | `REFUSE(not_owner)` at non-launch entries | `parked(remote_owner)` |
| Local owner, lapsed grant | `REFUSE(lease_conflict)` | `refused / lease_conflict` |

The adjacent negative vectors remain explicit: absent grant, missing clock, and
unusable policy remain `invalid_arguments`; the last is pinned by
`TestAuthorizeRemoteOwnerInvalidPolicyRemainsInvalidArguments`.

## Outcome-keyed moved-vector census

The comparison is keyed on observed outcome tuples, not test names and not
only `internal/fencing`: the complete importer set records the baseline and
candidate outcome for every grid cell, then diffs `(entry, vector, outcome)`.
The named direct movements are:

| Vector class | Before, all five entries | After activation/restore | After input/mutation/checkpoint | Intent |
| --- | --- | --- | --- | --- |
| V01 / probe 10, remote winner, lapsed grant | `REFUSE(lease_conflict)` | `PARK(remote_owner, not_owner)` | `REFUSE(not_owner)` | intended offer reachability |
| V07, stale-epoch token, remote winner, lapsed grant | `REFUSE(lease_conflict)` | `PARK(remote_owner, not_owner)` | `REFUSE(not_owner)` | intended direction precedence; stale fencing is composed separately |
| V08, same-epoch losing lease, remote winner, lapsed grant | `REFUSE(lease_conflict)` | `PARK(remote_owner, not_owner)` | `REFUSE(not_owner)` | intended direction precedence |
| V09, future-epoch token, remote winner, lapsed grant | `REFUSE(lease_conflict)` | `PARK(remote_owner, not_owner)` | `REFUSE(not_owner)` | intended direction precedence |
| V14, empty `LocalHostID`, remote winner, lapsed grant | `REFUSE(lease_conflict)` | `PARK(remote_owner, not_owner)` | `REFUSE(not_owner)` | measured movement; pre-existing empty-host bound below |

The terminstance composition has six corresponding non-stale rows (winning
and future token × active, parked, quiescing): before, `REFUSE(lease_conflict)`
and no transition; after, `PARK(remote_owner)` and no transition. The six
stale rows (stale epoch and same-epoch losing lease × the same three states)
remain `stale_fenced` with transition after the grant-independent stale
fallback. Raw property output is in
`logs/cross-candidate-prod-baseline-tests.log` and
`logs/candidate-stale-remote-lapsed-property.log`.

The full importer-set named diff is in `outcomes/per-test-diff.tsv`: 30
new-or-changed rows (5 `axpane`, 18 `fencing`, 7 `terminstance`), with no
pre-existing named row changing from pass to fail. The fencing slice is 309
baseline rows versus 327 candidate rows; the added rows are the named
regression/neighbor witnesses. This outcome-keyed method is why the six
unchanged-name terminstance movements are named above instead of being hidden
by an unchanged pass total.

## Finding-by-finding closure

| Finding | Change | Test that fails without it | Evidence |
| --- | --- | --- | --- |
| P1-1: stale remote incarnation lost fencing after the reorder | Retained the direct remote offer while restoring the grant-independent `StaleRelativeToWinner` composition and the literal `stale_fenced` assertions | `TestRV3F3_GrantLessStaleFences` fails all six lapsed/remote stale rows; `TestRV3F3_GrantLessDecidedNotStaleLeavesState` fails the winning/future no-transition contract | `logs/cross-candidate-prod-baseline-tests.log`, `logs/candidate-stale-remote-lapsed-property.log`, `outcomes/per-test-diff.tsv` |
| P2-1: board/archive record was stale | Rewrote both outcome sources in this Story worktree, repacked from those absolute paths, read both resources back, and diffed byte-for-byte | The readback `diff -u` fails if the board copy or archive member is stale or differs from the source | `board-verify/`, `MANIFEST.txt`, archive member hashes |
| P2-2: moved inputs were unnamed and name-keyed diff hid them | Added the outcome-keyed five-vector table and the complete importer-set per-test diff | The before/after grid and the six terminstance stale/non-stale assertions fail on the moved classes; a name-only diff cannot represent those outcome movements | `logs/*-fencing-grid.log`, `logs/cross-candidate-prod-baseline-tests.log`, `outcomes/per-test-diff.tsv` |
| P3-1: relative arms measured redundant but prose claimed they were load-bearing | Kept the explicit relative arm and documented the measured bound in `internal/terminstance/fencing.go` and TRACEABILITY; the grant-independent fallback is identified as load-bearing | Reviewer plants RM-O5 (relative question through mutation) and RM-O6 (relative `stale_owner` arm removed) survive twice over the complete importer set; the result is reported as a bound, not hidden | `notes/reviewer-bounds.md`, `internal/terminstance/TRACEABILITY.md`, `internal/terminstance/fencing.go` |
| P3-2: outer fallback had no harness row | Added `N-fencing-remote-lapsed-outer-fallback` with the unique outer-return/`observeRemoteWinner` anchor | `TestRV3F3_GrantLessDecidedNotStaleLeavesState` fails when the outer fallback admits decided-but-not-stale observations | `mutants/terminstance/pass1.raw.log`, `mutants/terminstance/pass2.raw.log`, `preflight-outer-fallback.log` |
| P3-3: arm-swap mask did not execute `ModeLaunch` | Kept the honest bound in the census: the shipped swap row is `./internal/fencing`; the full importer set separately drives the wrapper cell | `N-arm-order-expiry-before-direction` kills the direct gate; no claim is made that this row alone kills `axpane` `ModeLaunch` | `mutants/fencing/pass1/mutants-full.json`, `mutants/fencing/pass2/mutants-full.json`, conformance matrix |
| P3-4: stale names and validation attribution | Corrected fencing TRACEABILITY subtest names and LOGBOOK attribution; command 25 is silent, command 29 emits the standing 180 `MISSING_ACTIVITY` diagnostics | TRACEABILITY/LOGBOOK review checks fail against the old names/attribution | `internal/fencing/TRACEABILITY.md`, `LOGBOOK.md`, `validation/cmd-25-cataloggen.log`, `validation/cmd-29-task-board-validate.log` |
| P3-5: empty `LocalHostID` direction gap | Recorded the surviving RM-W1 bound and assigned it to the Story final leaf; did not widen this bug | RM-W1 survives candidate twice and trunk once over the importer set; a new test would be required to close it | `notes/reviewer-bounds.md`, moved V14 row, Story ownership statement |

## Gate × entry census

The three allowed cell states are named test plus admitting narrowing,
`unreachable` with a reason, or an explicit `BOUND` with an owner.

| Gate/entry cell | Census state |
| --- | --- |
| `Authorize(OperationRestore)` | `TestAuthorizeLapsedRemoteOwnerUsesDirectionForEveryEntry/restore` + `N-arm-order-expiry-before-direction` (KILLED) |
| `Authorize(OperationActivation)` | `TestAuthorizeLapsedRemoteOwnerUsesDirectionForEveryEntry/activation` + `N-arm-order-expiry-before-direction` (KILLED) |
| `Authorize(OperationInput)` | `TestAuthorizeLapsedRemoteOwnerUsesDirectionForEveryEntry/input` + `N-arm-order-expiry-before-direction` (KILLED) |
| `Authorize(OperationMutation)` | `TestAuthorizeLapsedRemoteOwnerUsesDirectionForEveryEntry/mutation` + `N-arm-order-expiry-before-direction` (KILLED) |
| `Authorize(OperationCheckpoint)` | `TestAuthorizeLapsedRemoteOwnerUsesDirectionForEveryEntry/checkpoint` + `N-arm-order-expiry-before-direction` (KILLED) |
| `axpane` `ModeLaunch` → `AuthorizeActivation` | **BOUND:** `TestDecideLapsedRemoteOwnerOffersOrParksByInteractiveContext/launch_interactive_attach` drives the production wrapper, but the shipped swap row is masked to `./internal/fencing`; the importer-set review probe is the owner of this wrapper-cell measurement. |

Thus the Authorize gate ordering is measured at **5 of 5** reachable fencing
entries. The sixth wrapper cell is not silently counted as a sixth direct
gate entry.

## Mutation and negative evidence

`internal/fencing/testdata/mutate.py` ran twice with
`PYTHONDONTWRITEBYTECODE=1`: 37 rows each pass, comprising 31 narrowing `N-`
rows plus `T-census-alias` (token-preserving behavioral narrowing), all 32
KILLED; `C-harmless-comment` SURVIVED; `C-not-applied` was NOT_APPLIED; and
`C-compile-failure` was classified `COMPILE_OR_HARNESS_FAILURE` with exit 1.
`N-arm-order-expiry-before-direction` was APPLIED and KILLED in both passes.

The terminstance harness ran 123 rows twice: 121 narrowing rows plus one
tightening row were KILLED, and the harmless `C-control` survived as the
required control. The new `N-fencing-remote-lapsed-outer-fallback` row was
APPLIED and KILLED in both passes. The relative-arm reviewer plants RM-O5 and
RM-O6 deliberately survived twice; they are the measured P3-1 bound, not
unreported failures. No `__pycache__` or `.pyc` files were present.

## Validation

Configured validation commands 1–30 all returned exit 0 on the final tree;
the per-command logs and `.rc` files are in `validation/`. Command 5 used the
configured `go test ./... -race -count=1 -timeout 25m`; command 6 produced
package coverage, including `internal/fencing` 99.3% and
`internal/terminstance` 86.6%. All 17 fuzz commands ran with `-fuzztime=100x`
and `-parallel=1`. The producer-required `GOOS=windows GOARCH=amd64 go vet
./...` also returned 0.

`go run ./internal/traceability/cmd/tracecheck` passed with
`contracts=64`, `acceptance_cases=152`, and `clauses_discharged=70/574`.
The catalog check (command 25) was silent. `task-board validate` (command 29)
returned 0 while printing the standing 180 `MISSING_ACTIVITY` diagnostics and
the ledger mirror summary; those diagnostics are not attributed to
cataloggen. `internal/traceability` was not edited.

## Explicit stop statement

If this run had stopped after merely rewriting the landed terminstance tests
from expiry-first refusal to remote-owner park/no-transition, it would have
reported a stale-fencing regression: six lapsed-grant/remote-winner stale
rows would remain active, parked, or quiescing instead of reaching literal
`stale_fenced`. It would have stopped and requested a composition fix rather
than blessing an unchanged-state assertion. The final candidate restores that
property while retaining the step-4 remote offer.

## Scope and bounds

- `sessrepo.CheckFencingExpiry` remains the expiry owner.
- No lease append, tuple, chain, or park-vocabulary implementation changed.
- `internal/traceability` was not edited.
- `RM-W1-direction-skipped-on-empty-localhost` remains a pre-existing bound
  owned by the Story's final leaf; this bug does not widen to fix it.
- No public `ax` CLI surface is claimed; the offer path is measured through
  the existing `axpane.Decide` production entry.
- The candidate remains uncommitted on
  `task-board/story/STORY-260917-kf9g1h` for handoff.
