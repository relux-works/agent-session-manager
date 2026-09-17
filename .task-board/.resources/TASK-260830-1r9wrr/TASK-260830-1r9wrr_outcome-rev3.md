# TASK-260830-1r9wrr — developer evidence for review, rev3

Candidate tree: `2303fc0dd5ecb51f59ab456256ff1ca3b6c2bbf6`. Signed Story checkpoint remains `0d9d0ad5acee26fed7bff78b605a07ff5230242c`.
No manual commit, branch switch, rebase, sibling run, upstream #176/#177 work,
or change to sessrepo/provhost occurred. Managed developer handoff owns the
candidate publication; independent review/checkpoint/integration belong to
the orchestrator. Exactly 16 CR paths, enumerated below. This rework
changes 7 of those paths against rev2; runtime implementation bytes
are preserved (only documentation and tests change in this rework).

## R2-F1: prerequisite established before broader rework

The already-attached `TASK-260830-1r9wrr_missing-gate-prerequisite.md` records
the gate-first ordering. Every direct Type-field use is derived via invcore,
classified by exact owning AST context and matched both directions. Aliases,
helper arguments, address-taking, package bindings, writes, closures and
unclassifiable contexts cannot silently disappear from that denominator.
Existing diagnostic expressions, effect dispatch and lease subtype comparison
remain the four classified uses; there is no list of newly guessed event names.
`TestEventTypeUsesAreOwned` runs with the event-spelling census in every
unfiltered suite. `TestSessstateStaticAudit` separately exposes static audits.

Residual bound: this is direct-field-access ownership, not whole-program taint
analysis. Reflection/unsafe/serialization of whole Event values and downstream
interpretation of permitted diagnostic strings are not proved. Other selectors
named Type are conservatively rejected for classification. This explicitly
withdraws the old exhaustive arbitrary-dispatch claim. No product redesign or
runtime validation was added to conceal the limitation.

The permanent `TestCensusLiveEventOwnershipPlants` compiles and wires the
reviewer plants into `Reduce -> apply -> effect -> reviewExtra -> step`,
executes separate behavior/census/audit selectors, then adds an independent
control-effect probe. All subprocess commands and real exits are retained in
full logs. The temporary copy is restored; the managed package is untouched.

| Mutant | Narrowing / behavior change | Build | B / C / A exits | Named failing test / survival bound |
| --- | --- | --- | --- | --- |
| PA | direct unknown selector handler | 0 | 0 / 1 / 0 | TestEventHandlingIsCensused; TestEventTypeUsesAreOwned |
| PB | copied kind switched in helper | 0 | 0 / 1 / 0 | TestEventTypeUsesAreOwned |
| PC | Type passed to string helper | 0 | 0 / 1 / 0 | TestEventTypeUsesAreOwned |
| PD | pointer to Type read in helper | 0 | 0 / 1 / 0 | TestEventTypeUsesAreOwned |
| live XN | identity-transform tail ID | 0 | 0 / 0 / 0 | SURVIVED: behavior-neutral identity transform only |
| live XK | inert default changed to idle, selector/case tokens preserved | 0 | 1 / 0 / 0 | TestReduceTreatsUnknownV1TypeAsInert |
| G-N | gate permits only the kind := event.Type copy while every other use still requires ownership | 0 | composed wrapper exit 1 | TestCensusLiveEventOwnershipPlants/PB |

4 of 4 dispatch plants killed, all census-only; 0 of 4 behavioral kills.
The post-gate `TestPlantUnknownV1Effect` is generated from each plant's unknown
literal and fails at Reduce with idle (expected exit 1) in all four cases.
Those probes establish live control effects, not additional delivered behavior
coverage. PB/PC/PD and XK retain the old searched tokens. G-N is an applied
narrowing, not a deletion: baseline green, weakened PB census green, permanent
live-control wrapper red. Its `go test` exit 1 is expected-red evidence; the
Python harness exit 0 only verifies that expected failure. Parse-only ownership
controls are separate from compiling live controls.

## R2-F2: explicit empty-chain Local bound

`TestReduceEmptyChainLocalIsIgnored` drives 14 combinations through
Reduce: absent/lesser/equal/greater/malformed/epoch-only/ID-only Local against
empty Union and a valid multi-entry Union with a selected winner. It compares
the entire Projection with the no-Local baseline, checks the named union
conflict, and pins creating plus unknown local role/owner. Local is intentionally
not validated when there is no authoritative event/process lease. README and
doc.go say so. Historical F7 closure now refers only to the Union half.

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


The 10 of 10 mapping is carried from accepted rev2 review findings; the named
committed tests ran again in the full suite. Closed F1/F2/F4/F5/F6, canonical
enums, multiple predecessors and the existing create-window recovery tests
were preserved. Prior independent 64-call reviewer probe was accepted from
attached evidence and not repeated; committed purity coverage did run again.
The read-only projector owns no durable write window. Existing tests exercise
the four sessrepo create boundaries via secconftest and recover through Project.

## Validation run directly in this developer session

| Command | Real exit | Full log |
| --- | --- | --- |
| `go test ./... -v` | 0 | `full-test-03.log` |
| `go test ./... -cover` | 0 | `coverage-03.log` |
| `go build ./...` | 0 | `build-03.log` |
| `go vet ./...` | 0 | `vet-03.log` |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `tracecheck-03.log` |
| `go test ./internal/sessstate -count=1 -run '^TestReduceEmptyChainLocalIsIgnored$' -v` | 0 | `local-bound-01.log` |
| `go test ./internal/sessstate -count=1 -run '^TestCensusLiveEventOwnershipPlants$' -v` | 0 | `live-ownership-02.log` |
| `python3 battery.py all` | 0 | `battery-03.log` |
| `python3 battery.py gate` | 0 | `gate-narrowing-03.log` |
| gofmt -l over all sessstate Go files | 0, empty output | format-03.log |
| git diff --check | 0, empty output | diff-check-03.log |

Full suite: 25 package rows. Coverage: 25 package
rows, sessstate 92.2%. Platform: Go 1.25.5, darwin/arm64. Native
build/test is the relevant target; Linux/Windows compile and full-repo race,
fuzz and remaining configured validation are not claimed in this pre-handoff
artifact. The observed handoff command returned exit 0, status to-review and checklist
27/27; it did not return a CR tree or execute the configured publication suite
in this live turn. Managed run completion captures the CR and its configured
validation after the headless session exits. That later system evidence must
not be confused with the checks already executed and attached here.
No tests were left running at handoff. The copy-local mutation and gate tests
use direct subprocess exit statuses; no tee/pipeline hides a gate status.

## Re-derived census and mutation measurements

After the last managed artifact: refusal sites 69 / rows
69 ({'decode.go': 28, 'project.go': 2, 'sessstate.go': 39}); states 11 /
rows 11; handled events 24 / rows
24; Type uses 4 / ownership rows
4. `census-measure-03.log` derives these through the same
invcore functions, and baseline census tests require equality in both directions.
Unregistered/orphan/unclassifiable controls, import aliases and var bindings
remain in the committed suites. This is the production-derived inventory
against which selected mutations are reported, not a claim of one mutant per
refusal or exhaustive arbitrary-Go mutation coverage.

Legacy selected roster re-executed: 28 killed / 28
applied, 22 behavioral, 3 census-only,
3 audit-only; mutation categories: 14
narrowing, 8 arm-deletion, 3
census-only, 3 audit-only. All compile (build exit 0).
Neutral XN survives with its bound; legacy XK fails its named stopped-state
test. These controls are excluded from the 28-member mutation denominator.
The four live dispatch plants and one gate-instrument narrowing are separately
reported above to avoid mixing selected-gate and instrument coverage.

B/C/A are disjoint anchored selectors: 53
behavioral tests, 8
census tests and one static-audit test. AST selection (live controls) and
compiled-test discovery (external runner) exclude the recursive live wrapper.
The TestMain runtime refusal census is exercised by the ordinary unfiltered
full-suite and coverage commands, not smuggled into the isolated layers.
A named failure is required for every kill below; full co-firing failures and
the exact source replacements are in `battery-results.json`. Mutant test
commands with exit 1 are expected failures; none is reported as passing.

| Mutant | Category / narrowing or change | B / C / A exits | Named failing test; survival bound |
| --- | --- | --- | --- |
| N1 | narrowing: admits the lesser lease ID as winner | 1 / 0 / 0 | `TestWinnerResolutionTie`; `TestWinnerResolutionTieOrdersBytewise` |
| N2 | narrowing: admits a 1→3 jump silently | 1 / 0 / 0 | `TestWinnerResolutionGap` |
| N3 | narrowing: admits checkpointed stop with null checkpoint | 1 / 0 / 0 | `TestReduceRefusesStoppedWithNullCheckpoint` |
| N4 | narrowing: admits retry past an abort | 1 / 0 / 0 | `TestReduceGatesFailedToCreatingRetry`; `TestReduceGatesFailedToCreatingRetry/retry_past_abort_refused` |
| N5 | narrowing: admits exactly one foreign provider | 1 / 0 / 0 | `TestReduceRefusesTaskBoardRepeatMismatch`; `TestReduceRefusesTaskBoardRepeatMismatch/foreign_provider` |
| N6 | narrowing: admits stopped as a first event | 1 / 0 / 0 | `TestReduceRefusesContinuityVectors`; `TestReduceRefusesContinuityVectors/stopped_opens_chain` |
| N7 | narrowing: admits unknown types as lifecycle | 1 / 0 / 0 | `TestReduceTreatsUnknownV1TypeAsInert` |
| N8 | narrowing: admits exactly one envelope-outside lease | 1 / 0 / 0 | `TestReduceRefusesLeaseLinkageMismatch`; `TestReduceRefusesLeaseLinkageMismatch/transfer_outside_envelope` |
| N9 | narrowing:  | 1 / 0 / 0 | `TestReduceRefusesDerivationVectors`; `TestReduceRefusesDerivationVectors/abort_closure_with_checkpoint` |
| N10 | narrowing: admits parked sessions as loadable | 1 / 0 / 0 | `TestProjectDerivesParkedBareDirectory`; `TestProjectDerivesParkedRecordWithoutChain` |
| N11 | narrowing: admits exactly one foreign provider, records its version | 1 / 0 / 0 | `TestProviderMismatchKeepsRecordPinned` |
| N12 | narrowing: admits any epoch with the right lease | 1 / 0 / 0 | `TestReduceRefusesTaskBoardRepeatMismatch`; `TestReduceRefusesTaskBoardRepeatMismatch/later_creation_epoch` |
| D1 | arm-deletion: admits every unlisted move | 1 / 1 / 0 | `TestReduceRefusesUnlistedTransitions`; `TestReduceRefusesUnlistedTransitions/running-to-stopped` |
| D2 | arm-deletion: admits same-epoch second lease | 1 / 0 / 0 | `TestReduceRefusesDivergentEnvelope` |
| D3 | arm-deletion: admits lower-epoch envelopes | 1 / 1 / 0 | `TestReduceRefusesStaleEnvelope` |
| D4 | arm-deletion: parks nothing | 1 / 0 / 0 | `TestProjectDerivesParkedBareDirectory`; `TestProjectDerivesParkedRecordWithoutChain` |
| D5 | arm-deletion: admits foreign providers | 1 / 1 / 0 | `TestReduceRefusesTaskBoardRepeatMismatch`; `TestReduceRefusesTaskBoardRepeatMismatch/foreign_provider` |
| D6 | arm-deletion: admits retry under a new lease | 1 / 1 / 0 | `TestReduceGatesFailedToCreatingRetry`; `TestReduceGatesFailedToCreatingRetry/new_lease_refused` |
| D7 | arm-deletion: admits chains that skip the head | 1 / 0 / 0 | `TestReduceRefusesContinuityVectors`; `TestReduceRefusesContinuityVectors/omits_prior_head` |
| D8 | arm-deletion: refuses confirming events | 1 / 0 / 0 | `TestReduceDerivesFailedFromBootstrapAbortStop`; `TestReduceAdmitsSameStateRestatement` |
| C1 | census-only: unregistered site | 0 / 1 / 0 | `TestRefusalSitesAreCensused` |
| A1 | audit-only: alias use | 0 / 0 / 1 | `TestSessstateStaticAudit` |
| A2 | audit-only: alias use | 0 / 0 / 1 | `TestSessstateStaticAudit` |
| A3 | audit-only: unrouted refusal | 0 / 0 / 1 | `TestSessstateStaticAudit` |
| T1 | census-only: forbidden spelling | 0 / 1 / 0 | `TestStateSpaceIsCensused` |
| T2 | census-only: unhandled handling | 0 / 1 / 0 | `TestEventHandlingIsCensused` |
| N13 | narrowing: admits a duplicated off-chain union entry silently (F1) | 1 / 0 / 0 | `TestWinnerResolutionDuplicateOffChainUnionKeepsConflict`; `TestWinnerResolutionRepeatedOffChainUnionStaysResolved` |
| N14 | narrowing: stales a local lease that won the tuple (F2) | 1 / 0 / 0 | `TestReduceLocalLeaseTupleRule`; `TestReduceLocalLeaseTupleRule/greater_epoch_stands` |
| XN | control: harness control, behaviour-neutral | 0 / 0 / 0 | SURVIVED: identity transform only (`"" + tailID`); no general safety claim |
| XK | control: harness control, known-bad | 1 / 0 / 0 | `TestProjectLeavesStoppedPastTakeoverUnstaled` |

## Candidate identity and exact CR paths

Detached-index commands: `GIT_INDEX_FILE=<task scratch>/delivery.index git
read-tree HEAD`, scoped `git add -- LOGBOOK.md README.md internal/sessstate`,
then `git write-tree`. The real index is read for identity only. The candidate
was re-derived after README, LOGBOOK and every code/test artifact; outcome
and archives live outside the managed worktree so they cannot change this tree.
The re-derived tree equals the attached `candidate-identity.json` record.
The managed CR must publish that same OID. Its record is not yet observable
before this headless run exits; the orchestrator must compare the published
CR tree to this attachment when routing independent review. The task-scoped candidate patch is exactly
HEAD-to-tree; review-delta.patch is the narrower rev2-to-tree delta.

```
LOGBOOK.md
README.md
internal/sessstate/arms_test.go
internal/sessstate/census_plants_test.go
internal/sessstate/census_state_test.go
internal/sessstate/census_test.go
internal/sessstate/decode.go
internal/sessstate/doc.go
internal/sessstate/fixtures_test.go
internal/sessstate/negative_test.go
internal/sessstate/project.go
internal/sessstate/project_test.go
internal/sessstate/purity_test.go
internal/sessstate/reduce_test.go
internal/sessstate/sessstate.go
internal/sessstate/winner_test.go
```

Checkpoint gap to local main: 2 (local-ref evidence only, no remote-main freshness claim).
