# BUG-260917-3lddu0 results

Status: ready for review handoff; the Story worktree is deliberately
UNCOMMITTED.

## Outcome

`internal/sessrepo.Repository.AppendEvent` now checks the durable winning lease
before normal chain admission. Lower-epoch events return literal
`ErrStaleLease`; same-epoch events from a losing lease return literal
`ErrDivergentBranch`. Both refusal arms preserve the event bytes as immutable
blobs without changing the authoritative chain. Historical replay deliberately
does not apply this new-event gate.

`sessprofile.SetProfile`, `axpane.Emit`, and `axpane.EmitParked` compose this
entry unchanged. `TestLosingLeaseProfileEventIgnored` drives the refusal,
derives the effective profile, and runs `Run`: a losing `profile.changed`
cannot become the effective source or produce the old `yolo` /
`--dangerously-bypass-approvals-and-sandbox` result.

This is route (b) from the rework brief. The profile-source property is closed
only through the durable `AppendEvent` gate. `sessprofile.Derive` is unchanged
and cannot observe the lease store; its independent derivation-side ownership
is a stated bound. The follow-up owner for that independent authority is
`STORY-260922-cpkajd` / `TASK-260922-31qyyi`
(`lease-aware-profile-source-authority` /
`derivation-side-profile-source-gate`). This candidate does not claim that
`Derive` alone can reject a losing event already present in an otherwise
continuous chain. The Story registry case is owned by
`internal/sessrepo/sessrepo.go:AppendEvent`.

Acceptance ratio: **2 of 2 AC rows driven** through named production call
sites. The gate-entry census measures **8 of 10 cells**. The direct
`AppendEvent`, `SetProfile`, `Emit`, and `EmitParked` sides each have lower
epoch and same-epoch losing tests; the two `Run → EmitParked` cells are stated
bounds. Row symmetry is lower epoch **4 of 4** and same-epoch loser **4 of 4**
on reachable writer entries; `Run → EmitParked` is bounded on both arms. The
complete table is in `BUG-260917-3lddu0_conformance-matrix.md` and mirrored in
`internal/sessrepo/TRACEABILITY.md`.

## Before/after evidence

The exact trunk baseline direct production reproduction is
`.temp/BUG-260917-3lddu0/baseline-probe-9-direct.log`: successor B was already
winning while the tail remained on A, yet the old implementation returned
`err=<nil>` instead of the expected stale refusal. The archived probe-15 log
records the old blast radius as `profile={Profile:yolo ... HasSource:true}`
with mapping `--dangerously-bypass-approvals-and-sandbox`.

The API-adapted baseline reproduction `baseline-probe-15-direct.log` logs
`profile=yolo`, `has_source=true`, and that exact mapping after the losing
append was accepted. The archived wrapper is not claimed as a literal
current-tree rerun because its API predates this candidate. Candidate
production-entry tests are:

- `TestAppendEventRefusesSupersededLeaseWhileTailStillMatches`
- `TestAppendEventRefusesSameEpochLosingLeaseWhileTailStillMatches`
- `TestSetProfileRefusesSupersededLeaseWhileTailStillMatches`
- `TestEmitReachesAppendAdmissionGateAfterStaleObservation`
- `TestEmitParkedReachesAppendAdmissionGateAfterStaleObservation`
- `TestEmitReachesSameEpochAppendAdmissionGate`
- `TestEmitParkedReachesSameEpochAppendAdmissionGate`
- `TestLosingLeaseProfileEventIgnored`

The direct same-epoch test and SetProfile, Emit, and EmitParked compositions
exercise both loser-ID directions (`cccccccc-dddd-4eee-8fff-111111111111` and
`11111111-2222-4333-8444-555555555555`). Direct and non-direct paths read back
preserved same-epoch bytes; axpane reads the event directory after refusal.

## Landed-test meaning changes

The importer test-status grid cannot detect semantic changes under an unchanged
test name. Its `2,018 before / 2,030 after` keys include the nine package
summary rows, so the actual test-row counts are `2,009 / 2,021`; it is not an
admission-outcome grid. The four landed tests below were therefore reviewed
explicitly. The first three swapped fixture ordering: under the old ordering,
`publishCheckpoint` attempted its `session.idle` and `session.stopped` events
under A after successor B had won. With the new gate those lower-epoch inputs
are `ErrStaleLease`. The candidate publishes the checkpoint closure before
the successor CAS, which is the `5.2`/`5.3` graceful-takeover order.

| Landed test | Before | Candidate now | Moved input class | Spec |
| --- | --- | --- | --- | --- |
| `internal/axpane/rework_test.go:TestRunPostWindowSupersedes` | Successor B preceded the checkpoint closure; old A then appended the stop fixture. | Closure is published first; B names it as handoff base; op2 launches and op1 remains. | `session.idle` and `session.stopped` under `(epoch 1, lease A)` after B wins → `ErrStaleLease`; now pre-takeover admitted history. | `5.2`/`5.3` |
| `internal/axpane/rev3_test.go:TestRunSupersededPairIdenticalRetry` | Same inverted ordering; old stop events were admitted. | Closure precedes B; op2/op3 launch and identical op2 retry reattaches. | The same `session.idle`/`session.stopped` lower-epoch-A-after-B class moves to `ErrStaleLease`; now pre-takeover history. | `5.2`/`5.3` |
| `internal/axpane/rev3_test.go:TestRunCreateFromStoppedPostWindow` | Same inverted ordering for stopped create-from-post-window. | Stopped/checkpointed state is made while A owns it; B follows; op2 launches. | The same lower-epoch-A-after-B `session.idle`/`session.stopped` class moves to `ErrStaleLease`; now pre-takeover history. | `5.2`/`5.3` |
| `internal/axpane/rework_test.go:TestLosingLeaseProfileEventIgnored` | `appendChainEvent` admitted lower-epoch `profile.changed` under A after B; test asserted the admitted event did not drive `Run`. Baseline reached `yolo` with the bypass mapping. | Raw production `AppendEvent` returns literal `ErrStaleLease`; `Derive` stays standard/no source and `Run` cannot launch yolo from it. | Losing `profile.changed` under `(epoch 1, lease A)` after B: admitted/yolo before → refused and preserved now. | `2.4` + `5.3` |

These four are not four added test-status keys: their names stay stable. This
table is the evidence for every moved input class that a name-keyed status grid
cannot show.

## Mutants

The shipped harness was run twice:

`PYTHONDONTWRITEBYTECODE=1 python3 internal/sessrepo/testdata/mutate_append_admission.py .temp/BUG-260917-3lddu0/mutants-append-rev3-pass1`

and the same command with `mutants-append-rev3-pass2`. Each run exited 0.
`N-stale-winning-admission` retains the gate but admits only reachable
`(winner epoch - 1, lease A)`; `N-same-epoch-winning-admission` retains the
gate but admits only reachable loser
`cccccccc-dddd-4eee-8fff-111111111111`. Both were **KILLED** by direct,
SetProfile, Emit, and EmitParked tests on both runs. The harmless comment
control was `SURVIVED` (exit 0) on both; `C-not-applied` was `NOT_APPLIED`;
`C-compile-failure` was `COMPILE_OR_HARNESS_FAILURE` and is not a kill. Both
control-before and control-after suites exited 0. Raw per-plant logs, exits,
and `mutants.json` are in both pass directories. No `__pycache__` artifacts.
No token-preserving source-text mutant applies because this gate does not
inspect source text.

## Composing-writer comparison

Mask used for both runs:

`go test -json -count=1 ./internal/axpane ./internal/crashgate ./internal/fencing ./internal/sessckpt ./internal/sessprofile ./internal/sessquery ./internal/sessstate ./internal/termbind ./internal/terminstance`

The exact baseline is parent checkpoint
`964fa472c97569fb209e5bdf831d42e2484ef078`; the candidate is that tree plus
this uncommitted delta. All 9 package summaries pass in both runs. The
test-status comparison is **2,018 before / 2,030 after** (including package
rows), with **12 added status keys, 0 removed, and 0 changed statuses**; the
actual test-row counts are **2,009 / 2,021**. The 12 additions are named in
the matrix and include the same-epoch loser-ID subtests. The status grid proves
the composing writers still pass, but not semantic identity for unchanged
names; the four moved input classes are the table above. Direct sessrepo tests
are outside this status mask.

Raw JSONL, exact exits, and normalized comparison are under
`.temp/BUG-260917-3lddu0/importer-outcomes-rev2/`.

The adopted runtime outcome comparison is the reviewer grid
`BUG-260917-3lddu0_review-runtime-outcomes-rev2.md`, over the same complete
nine-package importer set and a mechanically derived direct/internal/external
importer census. It records actual `AppendEvent` outcomes keyed by executing
package, event type, epoch, sequence, winner relation, and outcome: **130
outcome keys / 2,926 calls before** versus **137 / 2,955 after**, with **10
new outcome keys and 3 removed**. It records the moved arms: losing
`profile.changed` lower epoch `ADMIT → STALE`, same-epoch losers
`DIVERGENT`, the added stale `session.parked`, and the three old-owner
`session.idle`/`session.stopped` classes moving from lower-epoch-after-takeover
to same-winner pre-takeover history. This runtime grid, not unchanged PASS
status, is the outcome evidence.

## Registry and traceability

The v0.7.0 registry contains the two Story cases, and the losing-profile case
is referenced by the Section 2.4 binding and clause `2.4#2`. It is owned by
`internal/sessrepo/sessrepo.go:AppendEvent`, not
`internal/sessprofile/profile.go:Derive`. The existing Section 2.4
`sessprofile.Derive` binding remains the owner of independent derivation cases;
this leaf records the inability of `Derive` to see lease-store authority as a
bound and claims no independent Derive-side guarantee. That bound is owned by
`STORY-260922-cpkajd` / `TASK-260922-31qyyi`, including the disclosed
never-minted higher-epoch profile event and empty-lease-store classes; this leaf
does not decide or close either ambiguity.

Canonical registry projection digest, re-derived from the candidate registry:

`174728a8a01e8429004e4f9c4687f8fa1f17406b641012236c3427a758baf31e`

Exact-tree tracecheck output:

`traceability ok: contracts=64 normative_sections=36 acceptance_cases=154 fixtures=33 compatibility_contracts=55 assigned_scopes=0`
`section coverage: bindings=69 full=4 partial=9 sliver=9 unevidenced=43 unmeasured=4 unowned=7 clauses_discharged=70/574`

The README measured-coverage fenced line is byte-equal to the second line.
Section 2.4 scoped tracecheck admits; Section 5.3 scoped tracecheck refuses
as the expected 7/8 partial binding. Logs are under `traceability/`.

## Bounds and P3 dispositions

- **Independent profile-source authority (route (b) bound):** a never-minted higher-epoch profile event (epoch 3 while the store winner is epoch 2) remains admitted by `checkWinningLease` and may become a profile source; an empty lease store also skips the winner gate and can admit valid `(epoch 7, lease B)` sequence 1. These are not independent Section 2.4 ambiguity closure. The named owner is `STORY-260922-cpkajd` / `TASK-260922-31qyyi`; this leaf does not implement or decide those classes.
- **Run → EmitParked census cell:** unreachable except by an interleaving between `Observe` and `AppendEvent`, closed by the durable gate. Owner: `internal/axpane/run.go:Run` plus `internal/sessrepo/sessrepo.go:AppendEvent`.
- The axpane `decide.go` “lapsed grants refuse” qualifier remains untouched and scoped to local ownership.
- `observeRemoteWinner` structural redundancy is record-only; it was not simplified.
- The no-grant remote interactive-owner step-4 product question remains unresolved; this leaf makes no decision.
- Evidence hygiene records the candidate tree OID in the matrix and keeps the exact test mask beside every census and importer count.

## Validation

A fresh rev3 validation run is recorded under
`.temp/BUG-260917-3lddu0/validation-rev3/`. All 30 configured commands exited 0:
gofmt, build, vet, verbose full test, race, coverage, all 17 fuzz commands,
tracecheck, cataloggen, JSON parsing, task-board validation, and diff check. The
task-board validator reported its existing 179 `MISSING_ACTIVITY` diagnostics
but exited 0. The scoped tracecheck exited 0 for §2.4 and exited 1 for §5.3,
the expected refusal of its honest 7/8 partial binding. The final candidate
TREE OID is `3306cab83d2eece38ffc3ac43c87445e57993211` (detached-index
`write-tree`, after final tracked documentation edits). No review acceptance or
main integration is claimed.

The implementation/test/documentation delta is intentionally uncommitted for
the managed Story worktree snapshot handoff.
