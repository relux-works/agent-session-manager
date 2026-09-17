# TASK-260830-1r9wrr outcome — implement-session-state-reducer (rev2)

**Status: ready for review.** Rework round after CR rev1 `changes_requested`
(F1–F7). Same story worktree, one handoff past the rev1 state, still
uncommitted. `internal/sessrepo` production is untouched;
`internal/provhost` is untouched. Every number below was re-derived after
the last artifact was written.

Candidate tree (detached-index `read-tree HEAD` + add of the 16 CR paths,
`write-tree`): `6bdd953fb044b5b5fad568f94fcff86477e9d7fb`, holding all 14
`internal/sessstate` files. CR paths, exactly 16:

- `LOGBOOK.md`, `README.md`
- `internal/sessstate/doc.go`, `sessstate.go`, `decode.go`, `project.go`
- `internal/sessstate/arms_test.go`, `census_plants_test.go`,
  `census_state_test.go`, `census_test.go`, `fixtures_test.go`,
  `negative_test.go`, `project_test.go`, `purity_test.go`,
  `reduce_test.go`, `winner_test.go`

## What changed since rev1 (finding → fix → pin)

| Finding | Fix (production) | Pinning test (fails on the old code) |
| ------- | ---------------- | ------------------------------------ |
| F1: `resolveWinner` `case 0:` cleared `offChainWinner` exactly when the winner was already off-chain; duplicated union entry dropped the `union_supersedes_chain` conflict + warning | arm is a documented no-op | `TestWinnerResolutionDuplicateOffChainUnionKeepsConflict` (reported vector) + `TestWinnerResolutionRepeatedOffChainUnionStaysResolved` (triple copy + losing branch, both orders — one step past) |
| F2: `applyLocalLease` staled on not-equal against three sources stating lost-the-tuple | `>= 0` stands, only `< 0` stales | `TestReduceLocalLeaseTupleRule`: equal / greater-epoch / greater-lease-id stand, lesser stales (lease-id row hits the second `Compare` arm — one step past) |
| F3: event census denominator was one function name (`effect`) | `deriveHandledEventTypes` collects every switch over a `.Type` selector plus every ==/!= comparison against one; a second switch dispatch site outside `effect` fails closed | reviewer PA replayed live: KILLED at the behavioral gate (`event-type dispatch outside effect` + `session.reaped` unregistered); permanent controls `second switch dispatch site is unregistered` (PA shape) and `if-compared spelling is unregistered` (different shape) |
| F4: published 67-site (39/26/2) inventory | re-derived in source: 69 sites (39/28/2); corrected in README and here | `TestRefusalSitesAreCensused` (both directions) + `grep refuse` == roster == derived |
| F5: X1 NOT_APPLIED / X2 COMPILE_FAIL never reached the gates | replaced by live controls XN/XK; either misbehavior prints HARNESS-FAIL and exits nonzero | XN neutral (`"" + fold.tailID`) CONTROL-OK/SURVIVED; XK known-bad (staleSources admits stopped) CONTROL-OK/KILLED by `TestProjectLeavesStoppedPastTakeoverUnstaled` |
| F6: `scalar.ParseUUIDv7` watched but never called | dropped from `doc.go`, the census spec, and README; UUIDv7 stays with `environ.CheckUUIDv7` | alias audit has no unfireable row |
| F7: empty chain skipped union validation | early return removed: empty chains derive creating through the same `resolveWinner`/`applyLocalLease`/`emitWarnings` path (same refusal sites, no roster churn); bound stated in `doc.go` + README | `TestReduceEmptyChainValidatesUnion` (malformed refuses ×2, well-formed keeps creating with reported winner, bare stays zero) |

The refusal roster moved with the F7 edit (every `sessstate.go` row past
line 323 shifted +4, remapped mechanically and verified byte-equal against
`grep -n "refuse(Err"` output — 39/39). Three shifted rows (426, 444, 448)
collided numerically with stale rows, which is why the remap was verified
against source rather than trusted from the diff.

## AC coverage: 10 of 10 rows driven (unchanged, not restructured)

| # | AC row | Production call site | Driving test |
| - | ------ | -------------------- | ------------ |
| 1 | All 11 SessionState values derived | `sessstate.Reduce` + `Projector.Project` | `TestReduceDerivesEverySessionState` (11 subtests, each through both entries) |
| 2 | Winning epochs: clear win, tie, gap | `Reduce` → `resolveWinner`, `Compare` | `TestWinnerResolutionClearWin/Tie/TieOrdersBytewise/Gap` + `TestWinnerResolutionDuplicateOffChainUnionKeepsConflict` + `TestWinnerResolutionRepeatedOffChainUnionStaysResolved` |
| 3 | Conflicts refused or resolved by a named rule | `chainFold.conflict` (5 kinds, 5 rules) | `TestWinnerResolutionTie/LosingBranchPreserved/UnionSupersedesChain/PastEpochTie`, `TestProviderMismatchKeepsRecordPinned` |
| 4 | Current checkpoints | `noteCheckpoint` (checkpoint.created, stopped, resumed) | `TestProjectDerivesFullLifecycle` (newest follows created→stopped→resumed), state table stopped |
| 5 | Provider identity | `effectProviderLaunched/Identified` | `TestProjectDerivesFullLifecycle` (codex 0.147.0, identity record, exact) |
| 6 | Terminal binding, v1 and v4 | `effectTerminalCreated/Resumed` | `TestProjectDerivesFullLifecycle` (tmux/native-1), `TestProjectDerivesV4TerminalBinding` (digest on ax.tmux) |
| 7 | Parked session reaches state, not error/omission | `Projector.Project` parked branch; `Reduce` parked input | `TestProjectDerivesParkedBareDirectory/RecordWithoutChain/TornStore` (secconftest points inside the create window) |
| 8 | Negative/refusal cases | `step`, `trackLease`, integrity arms, decode arms | `TestReduceRefusesUnlistedTransitions` (6 rows) + control, `TestReduceRefusesStoppedWithNullCheckpoint`, `...BootstrapAbortStopWithoutAbort`, `...TaskBoardRepeatMismatch` (3 rows), `...LeaseLinkageMismatch` (3 rows), member/continuity/union vectors, `TestReduceEmptyChainValidatesUnion`, `TestReduceLocalLeaseTupleRule` |
| 9 | Crash/idempotency evidence | secconftest points at the 4 owned create-step boundaries driving `Project` + purity | `TestProjectDerivesParkedBareDirectory/RecordWithoutChain/EventsDirBoundary` (fault→parked→retry heals), `TestProjectRecoversPastChainBoundary`, `TestProjectIsIdempotentAcrossReads`, `TestReduceIsPure` |
| 10 | No unsupported capability advertised | no command/doctor/claim added | `tracecheck` exit 0; `TestNoProductionPathAttestsProviderIdentityBinding` (provhost) green |

## Gate table (each run directly, exit codes observed)

| Gate | Command | Exit |
| ---- | ------- | ---- |
| build | `go build ./...` | 0 |
| vet | `go vet ./internal/sessstate/` | 0 |
| windows | `GOOS=windows go build ./...`, `GOOS=windows go vet ./internal/sessstate/` | 0, 0 |
| format | `gofmt -l internal/sessstate/` | clean |
| tracecheck | `go run ./internal/traceability/cmd/tracecheck` | 0 |
| full suite | `go test ./... -count=1` (25 pkgs ok, 0 FAIL) | 0 |
| race | `go test -race ./internal/sessstate/ ./internal/sessrepo/ -count=1` | 0 |
| cover | `go test ./internal/sessstate/ -cover -count=1` | 0 — 92.2% of statements |
| battery | `.temp/TASK-260830-1r9wrr/mutate.sh` (full run, rev2 log attached) | 0 — 28 killed, controls 2/2 OK |

## Census numbers (all derived through invcore, fail-closed both directions)

- Refusals: **69** derived sites = `sessstate.go` 39 + `decode.go` 28 +
  `project.go` 2; roster 69 rows each naming its driver; exercised set
  equals derived; observed sentinel set equals the 8-sentinel roster.
- States: 11 spellings derived from the `State`-typed const block; each
  lives exactly once as a production literal; each driven to through
  `Reduce` in the census; `States()` order pinned to Section 5.7.
- Events: 24 handled literals derived from **every production dispatch on
  event type** (each `.Type` switch plus each ==/!= comparison); roster
  equals the pinned v1 registry exactly (24/24), each row with
  lifecycle-or-fact class and driver; a second switch dispatch site
  outside `effect` fails closed.
- Alias audit over the funnel plus environ/scalar/terminalbackend
  delegations (no unfireable rows); unrouted `%w`-with-sentinel audit.
  Plants: import alias and var binding in both directions, dot import,
  shadowed name, multi-line/non-ident/unknown-sentinel/shared-line
  refuse, `State()` over a variable, new spelling, orphan row,
  non-literal event-type case, second dispatch site (PA shape),
  if-compared spelling (different shape), unparseable-harness control.

## Mutation battery (production-derived denominator; read, decode, and
project paths — there is no write path)

| Mutant | What it narrows the gate to | Named test that fails |
| ------ | --------------------------- | --------------------- |
| N1 tie-break inverted | admits the lesser lease ID as winner | `TestWinnerResolutionTie` (+bytewise, +losing) |
| N2 gap `+1`→`+2` | admits a 1→3 jump silently | `TestWinnerResolutionGap` |
| N3 stopped drops null check | admits checkpointed stop with null checkpoint | `TestReduceRefusesStoppedWithNullCheckpoint` |
| N4 retry drops abort arm when no checkpoint | admits retry past an abort | `.../retry_past_abort_refused` |
| N5 tb-launched admits provider `qwen` | admits exactly one foreign provider | `.../foreign_provider` |
| N6 initial targets admit stopped | admits stopped as a first event | `.../stopped_opens_chain` |
| N7 unknown type derives idle | admits unknown types as lifecycle | `TestReduceTreatsUnknownV1TypeAsInert` |
| N8 transfer admits new-id `cccc…` | admits exactly one envelope-outside lease | `.../transfer_outside_envelope` |
| N9 abort closure `||`→`&&` | admits abort with checkpoint but no resumable | `.../abort_closure_with_checkpoint` |
| N10 parked-with-retry treated healthy | admits parked sessions as loadable | parked tests (bare/record/torn) |
| N11 launched admits provider `qwen` | admits exactly one foreign provider, records its version | `TestProviderMismatchKeepsRecordPinned` |
| N12 tb-launched drops epoch arm | admits any epoch with the right lease | `.../later_creation_epoch` |
| N13 case-0 off-chain reset reintroduced | admits a duplicated off-chain union entry silently (F1) | `TestWinnerResolutionDuplicateOffChainUnionKeepsConflict` + `...StaysResolved` (both orders) |
| N14 local comparison `>=`→`==` | stales a local lease that won the tuple (F2) | `TestReduceLocalLeaseTupleRule/greater_epoch_stands` + `/greater_lease_id_stands` |
| D1 transition gate deleted | admits every unlisted move | `TestReduceRefusesUnlistedTransitions` (+roster orphan) |
| D2 divergent arm neutered | admits same-epoch second lease | `TestReduceRefusesDivergentEnvelope` (+orphan) |
| D3 stale arm neutered | admits lower-epoch envelopes | `TestReduceRefusesStaleEnvelope` (+orphan) |
| D4 parked branch neutered | parks nothing | parked tests (+orphan) |
| D5 tb provider arm deleted | admits foreign providers | `.../foreign_provider` (+orphan) |
| D6 retry new-lease arm neutered | admits retry under a new lease | `.../new_lease_refused` (+orphan) |
| D7 predecessor-omission arm neutered | admits chains that skip the head | `.../omits_prior_head` (+orphan) |
| D8 restatement rule deleted | refuses confirming events | `TestReduceAdmitsSameStateRestatement` |
| C1 planted live refuse site | unregistered site | `TestRefusalSitesAreCensused` (census-test kill) |
| A1 var-bound funnel in live line | alias use | TestMain alias audit only (BEH green, FULL red) |
| A2 shadowed funnel in live line | alias use | TestMain alias audit only |
| A3 `%w`-wrapped sentinel in live line | unrouted refusal | TestMain unrouted audit only |
| T1 `creating`→`created` spelling | forbidden spelling | `TestStateSpaceIsCensused` |
| T2 extra `session.future` case | unhandled handling | `TestEventHandlingIsCensused` |
| XN neutral `"" + fold.tailID` | harness control, behaviour-neutral | CONTROL-OK — SURVIVED as intended |
| XK staleSources admits stopped | harness control, known-bad | CONTROL-OK — KILLED as intended by `TestProjectLeavesStoppedPastTakeoverUnstaled` |

28 applied, 28 killed (14 narrowing, 8 arm-deletion, 3 census-test,
3 audit-only). Survival bound, stated honestly: the single surviving
mutant is the neutral control XN, which survives by design — it is a
behaviour-neutral identity transform (`tailEventID` returning
`"" + fold.tailID`) proving the harness can report SURVIVED, the outcome
every production kill above is measured against. No production mutant
survives, so no further survival bound is owed.
