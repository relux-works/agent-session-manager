# BUG-260917-2fwf8e conformance matrix — revision 3

Authority: `internal/specdoc/SPEC.v0.7.0.md`, §4.2 after-restore sequence,
step 4. Scope is the arm ordering in `internal/fencing.Authorize`; the lease
chain remains owned by `sessrepo` and the `session.parked` vocabulary remains
owned by `internal/fencing`.

## Acceptance rows

| Row | Production call site | Named test/evidence | Measured result |
| --- | --- | --- | --- |
| Remote interactive owner with a lapsed local grant reaches the offer path | `fencing.Authorize(OperationRestore, ...)`, then `axpane.Decide` | `TestAuthorizeRemoteInteractiveOwnerLapsedGrantUsesRemoteOfferArm`; `TestDecideLapsedRemoteOwnerOffersOrParksByInteractiveContext/{restore_interactive_attach,restore_interactive_takeover,launch_interactive_attach}` | **1 of 1 AC row:** direct `PARK(remote_owner, not_owner)`, never literal `lease_conflict`; wrapper emits `attach_remote` or `takeover_offer`. |
| Arm ordering is pinned and swap is shipped | `fencing.Authorize` | `TestAuthorizeLapsedRemoteOwnerUsesDirectionForEveryEntry`; `N-arm-order-expiry-before-direction` | **1 of 1 AC row:** expiry-first swap is applied and exits 1 in both fencing mutation passes. |
| Previously reported vector plus neighbours | `fencing.Authorize`; `axpane.Decide` | `TestAuthorizeArmOrderingNeighbourVectors/{live_local_grant,remote_owner_live_grant,local_owner_lapsed_grant}`; wrapper context test | **1 of 1 AC row:** direct reported vector plus three required neighbours are driven; attach, takeover, and noninteractive wrapper outcomes are driven. |
| Per-test and moved-outcome accounting | importer-set package entries | `outcomes/per-test-diff.tsv`, five-vector grid, terminstance property logs | **1 of 1 evidence row:** outcome-keyed diff names V01, V07, V08, V09, V14 and the six terminstance non-stale movements. |
| Negative/gate evidence | direct gates and composed fencing | fencing/terminstance mutation passes and raw per-plant logs | **1 of 1 evidence row:** every shipped gate has a narrowing attack; intentional bounds and the harmless survivor are named. |

Task AC coverage is **3 of 3**; the six task-specific behavior evidence rows
are **6 of 6**. The implementation remains one arm-ordering leaf.

## Gate × entry census

Each cell is either a named committed test plus an admitting narrowing,
`unreachable` with a reason, or an explicit `BOUND` with an owner.

| Entry/cell | State and owner |
| --- | --- |
| `Authorize(OperationRestore)` | `TestAuthorizeLapsedRemoteOwnerUsesDirectionForEveryEntry/restore` + `N-arm-order-expiry-before-direction` (KILLED). |
| `Authorize(OperationActivation)` | `TestAuthorizeLapsedRemoteOwnerUsesDirectionForEveryEntry/activation` + the same applied/KILLED swap row. |
| `Authorize(OperationInput)` | `TestAuthorizeLapsedRemoteOwnerUsesDirectionForEveryEntry/input` + the same applied/KILLED swap row. |
| `Authorize(OperationMutation)` | `TestAuthorizeLapsedRemoteOwnerUsesDirectionForEveryEntry/mutation` + the same applied/KILLED swap row. |
| `Authorize(OperationCheckpoint)` | `TestAuthorizeLapsedRemoteOwnerUsesDirectionForEveryEntry/checkpoint` + the same applied/KILLED swap row. |
| `axpane ModeLaunch → AuthorizeActivation` | **BOUND:** `TestDecideLapsedRemoteOwnerOffersOrParksByInteractiveContext/launch_interactive_attach` drives the wrapper, but the shipped swap row's mask is `./internal/fencing`; importer-set review evidence owns the wrapper-cell measurement. |

The Authorize gate is therefore measured at **5 of 5 reachable fencing
entries**, with no false claim that the direct swap row covers the sixth
wrapper cell.

## Reviewer probe and neighbour evidence

The probe input is `TASK-260830-1geqhj_review-evidence-rev1.tar.gz`, SHA-256
`9f473449f33fda2be591f23396da2b9b9b995ba18708b02afdff7a550e840ef4`.
The baseline is checkpoint `799c338e401fca0b24859c0b870cd665927209f2`.

| Surface | Before | After |
| --- | --- | --- |
| Direct `Authorize` | `refused / lease_conflict` | `parked / remote_owner`, cause `not_owner` |
| Restore interactive + attach admitted | `refused / lease_conflict` | `attach_remote / remote_owner` |
| Restore interactive + attach not admitted | `refused / lease_conflict` | `takeover_offer / remote_owner` |
| Restore non-interactive | `refused / lease_conflict` | `parked / remote_owner` |

The live-local, remote-live, remote-noninteractive-lapsed, local-lapsed,
no-grant, no-clock, and invalid-policy neighbours are preserved in the before/
after grid logs. The archived source's removed `Realm.AttestedServer` field is
a recorded API-drift bound; the minimal current probe and direct regression
cover the same vector.

## Outcome movement and full-suite diff

The comparison keys on outcome tuples `(entry, vector, outcome)` over the
complete importer set, not only test names in `internal/fencing`. The five
direct classes that move from `lease_conflict` on trunk are exactly V01 probe
10, V07 stale epoch, V08 same-epoch losing lease, V09 future epoch, and V14
empty `LocalHostID`. Each becomes `remote_owner/not_owner` on
activation/restore and `not_owner` on input/mutation/checkpoint. V14 is
intentionally recorded as a pre-existing empty-host bound, not fixed here.

The terminstance non-stale rows (winning/future token × active/parked/quiescing)
move from `REFUSE(lease_conflict)` to `PARK(remote_owner)` with no transition;
the six stale rows remain `stale_fenced` with a transition. The full named
diff has 30 new-or-changed rows: 5 axpane, 18 fencing, and 7 terminstance;
there are no pre-existing named pass/fail changes. The fencing slice is 309
baseline rows versus 327 candidate rows. See `outcomes/per-test-diff.tsv` and
the `logs/*-fencing-grid.log` files.

## Finding closure and bounds

| Finding | Closure/evidence |
| --- | --- |
| P1-1 stale fencing | `TestRV3F3_GrantLessStaleFences` remains a literal `stale_fenced` regression; the grant-independent `StaleRelativeToWinner` fallback restores all six stale rows while direct Authorize still exposes the offer. |
| P2-1 stale board/archive record | Results and matrix are rebuilt from this worktree, packed from absolute paths, read back from the board, and byte-diffed. `board-verify/` and `MANIFEST.txt` are the checks. |
| P2-2 unnamed moved inputs | V01/V07/V08/V09/V14 and the six terminstance movements are named with before/after outcomes; outcome-keyed diff is shipped. |
| P3-1 redundant relative arms | RM-O5 and RM-O6 survived twice over the complete 1,103-row importer set. The relative arm is retained but explicitly documented as redundant; the grant-independent fallback is the measured load-bearing path. |
| P3-2 missing outer row | `N-fencing-remote-lapsed-outer-fallback` includes the outer return plus `observeRemoteWinner` anchor and is KILLED twice. |
| P3-3 incomplete swap mask | `ModeLaunch` is a stated BOUND owned by importer-set review evidence; the swap row claims only `./internal/fencing`. |
| P3-4 prose/attribution | TRACEABILITY names the real `restore_*`/`launch_*` subtests; command 25 is silent and command 29 owns the standing 180 `MISSING_ACTIVITY` diagnostics. |
| P3-5 empty-host direction | RM-W1 survives candidate twice and trunk once; this pre-existing gap belongs to the Story final leaf and is not widened here. |

## Negative and mutation evidence

The fencing mutation battery has 37 rows in each of two passes: 31 narrowing
rows plus the token-preserving `T-census-alias` are KILLED; the harmless
comment control survives; neutral not-applied and compile-failure controls
remain classified. `N-arm-order-expiry-before-direction` is APPLIED/KILLED in
both passes. The terminstance battery has 123 rows in each pass: 121
narrowing plus one tightening row KILLED, and `C-control` SURVIVED. The new
outer-fallback row is APPLIED/KILLED twice. Raw logs include subprocess exits;
the harnesses use `PYTHONDONTWRITEBYTECODE=1` and produce no pycache.

## Validation and scope

Configured commands 1–30, plus `GOOS=windows GOARCH=amd64 go vet ./...`, all
returned exit 0. `tracecheck` passed with contracts=64, acceptance_cases=152,
clauses_discharged=70/574; command 25 was silent; command 29 returned 0 with
the standing 180 issue diagnostics. Coverage includes `internal/fencing`
99.3% and `internal/terminstance` 86.6%. No lease-chain, park-vocabulary, or
`internal/traceability` change is in scope.

If the run had stopped after rewriting the old terminstance expectation, it
would have reported the six-row stale-fencing regression and stopped rather
than blessing no-transition state. That is why the final composition keeps
both the remote offer and literal `stale_fenced` property.

The candidate is intentionally uncommitted; handoff creates the reviewable
commit from the Story checkpoint.
