# BUG-260917-3lddu0 conformance matrix

Candidate is the uncommitted Story worktree. Candidate TREE OID (detached-index
`write-tree`, after final tracked documentation edits) is
`3306cab83d2eece38ffc3ac43c87445e57993211`. The baseline checkpoint for the
importer comparison is `964fa472c97569fb209e5bdf831d42e2484ef078`.

## Acceptance rows

| Row | Production call site | Named test | Result |
| --- | --- | --- | --- |
| Superseded lease append is refused while the tail still names the old lease | `internal/sessrepo/sessrepo.go:Repository.AppendEvent` → `checkWinningLease` | `internal/sessrepo/sessrepo_test.go:TestAppendEventRefusesSupersededLeaseWhileTailStillMatches`; composing `TestSetProfileRefusesSupersededLeaseWhileTailStillMatches`, `TestEmitReachesAppendAdmissionGateAfterStaleObservation`, `TestEmitParkedReachesAppendAdmissionGateAfterStaleObservation` | Lower epoch returns literal `ErrStaleLease`; same-epoch loser returns literal `ErrDivergentBranch`; bytes are preserved and the authoritative chain is unchanged. **1 of 1 driven.** |
| Losing-lease `profile.changed` cannot become the effective profile source | `internal/sessrepo/sessrepo.go:Repository.AppendEvent` → `internal/axpane/run.go:Run` → `sessprofile.LoadProfile`/`Derive` | `internal/axpane/rework_test.go:TestLosingLeaseProfileEventIgnored` | Production append refuses the losing event; `Derive` remains standard/no source; `Run` cannot use the event for yolo or the bypass mapping. **1 of 1 driven.** This is route (b): the independent derivation-side property is a bound owned by `STORY-260922-cpkajd` / `TASK-260922-31qyyi` (`lease-aware-profile-source-authority` / `derivation-side-profile-source-gate`). |

Task AC ratio: **2 of 2 AC rows driven**. The archived probe-15 before-state
is retained as historical evidence, with `profile=yolo`,
`has_source=true`, and
`--dangerously-bypass-approvals-and-sandbox` named explicitly.

## Gate × entry census

The new gate is `checkWinningLease`, called by
`internal/sessrepo/sessrepo.go:Repository.AppendEvent`. Existing
`checkAppend` continuity logic is unchanged.

| Gate arm | Direct `AppendEvent` | `Transactor.SetProfile` | `axpane.Emit` | `axpane.EmitParked` | `Run → EmitParked` |
| --- | --- | --- | --- | --- | --- |
| Lower epoch → `ErrStaleLease` | `TestAppendEventRefusesSupersededLeaseWhileTailStillMatches` | `TestSetProfileRefusesSupersededLeaseWhileTailStillMatches/lower epoch` | `TestEmitReachesAppendAdmissionGateAfterStaleObservation` | `TestEmitParkedReachesAppendAdmissionGateAfterStaleObservation` | Bound: unreachable except by an interleaving between `Observe` and `AppendEvent`, closed by the durable gate. Owner `internal/axpane/run.go:Run` + `internal/sessrepo/sessrepo.go:AppendEvent`. |
| Same epoch, losing lease → `ErrDivergentBranch` | `TestAppendEventRefusesSameEpochLosingLeaseWhileTailStillMatches` (both loser IDs) | `TestSetProfileRefusesSupersededLeaseWhileTailStillMatches` (both loser-ID subtests) | `TestEmitReachesSameEpochAppendAdmissionGate` (both loser IDs) | `TestEmitParkedReachesSameEpochAppendAdmissionGate` (both loser IDs) | Bound: unreachable except by an interleaving between `Observe` and `AppendEvent`, closed by the durable gate. Owner `internal/axpane/run.go:Run` + `internal/sessrepo/sessrepo.go:AppendEvent`. |

Gate-entry ratio: **8 of 10 cells driven**; **2 of 10 cells are stated
bounds**. Measured row symmetry is lower epoch **4 of 4** and same-epoch loser
**4 of 4** across the four reachable writer entries. The `Run → EmitParked`
entry is bounded in both arms with the same owner and reason.

The direct same-epoch and composing tests cover both loser-ID directions:
`cccccccc-dddd-4eee-8fff-111111111111` and
`11111111-2222-4333-8444-555555555555`. Direct and axpane composing paths
read back the preserved same-epoch blob; the chain remains unchanged.

## Landed-test meaning changes

The importer test-status grid cannot see a semantic change under a stable test
name. Its counts include nine package summary rows: `2,018 / 2,030` status
keys, or `2,009 / 2,021` actual test rows. These four tests were changed and
their moved input classes are recorded here and in the results/LOGBOOK:

| Test | Previous fixture/meaning | Candidate fixture/meaning | Moved class | Spec |
| --- | --- | --- | --- | --- |
| `TestRunPostWindowSupersedes` | Successor B was installed before `publishCheckpoint`. | `publishCheckpoint` precedes successor B; op2 launches and op1 remains. | `session.idle` and `session.stopped` under A after B → `ErrStaleLease`; now pre-takeover admitted. | `5.2`/`5.3` |
| `TestRunSupersededPairIdenticalRetry` | Same inverted ordering; old stop events were admitted. | Closure precedes B; op2/op3 launch; op2 retry reattaches. | Same lower-epoch-A-after-B `session.idle`/`session.stopped` class → `ErrStaleLease`; now pre-takeover history. | `5.2`/`5.3` |
| `TestRunCreateFromStoppedPostWindow` | Same inverted ordering for stopped create-from-post-window. | Stopped/checkpointed state is made under A, then B follows; op2 launches. | Same lower-epoch-A-after-B `session.idle`/`session.stopped` class → `ErrStaleLease`; now pre-takeover history. | `5.2`/`5.3` |
| `TestLosingLeaseProfileEventIgnored` | `appendChainEvent` admitted lower-epoch `profile.changed` after B; only the downstream `Run` source was checked. Baseline reached yolo/bypass. | Raw `AppendEvent` must return literal `ErrStaleLease`; profile stays standard/no source; `Run` cannot launch yolo from it. | Losing `profile.changed` under A after B: admitted/yolo → refused and preserved. | `2.4` + `5.3` |

The four stable names therefore do not appear as status changes in the importer
grid; the table is the semantic evidence.

## Narrowing evidence

Harness command:
`PYTHONDONTWRITEBYTECODE=1 python3 internal/sessrepo/testdata/mutate_append_admission.py <evidence-dir>`.

| Plant | Reachable narrowing | Behavioral mask | Pass 1 / Pass 2 |
| --- | --- | --- | --- |
| `N-stale-winning-admission` | Retains the gate but admits exactly `(winner epoch - 1, lease A)`; all other lower-epoch classes remain gated. | Direct stale append, SetProfile lower arm, Emit stale observation, EmitParked stale observation. | **KILLED / KILLED**, subprocess exit 1 on each plant run; control-before/after exit 0. |
| `N-same-epoch-winning-admission` | Retains the gate but admits exactly reachable loser `cccccccc-dddd-4eee-8fff-111111111111`; other losing IDs remain gated. | Direct same-epoch append, SetProfile same-epoch arm, Emit same-epoch, EmitParked same-epoch. | **KILLED / KILLED**, subprocess exit 1 on each plant run; control-before/after exit 0. |
| `C-harmless-comment` | Comment-only change. | `TestAppendEventChainInOrder`. | **SURVIVED / SURVIVED**, exit 0. |
| `C-not-applied` | Missing replacement token. | No plant run. | **NOT_APPLIED / NOT_APPLIED**, not a kill. |
| `C-compile-failure` | Deliberate syntax failure. | `TestAppendEventChainInOrder`. | **COMPILE_OR_HARNESS_FAILURE / COMPILE_OR_HARNESS_FAILURE**, not a kill. |

Evidence directories are
`.temp/BUG-260917-3lddu0/mutants-append-rev3-pass1/` and
`mutants-append-rev3-pass2/`. Both true mutants are narrowing, not delete-only.
No token-preserving source-text mutant applies because the gate does not inspect
source text.

## Complete importer comparison

Mask used for both JSONL runs:

`go test -json -count=1 ./internal/axpane ./internal/crashgate ./internal/fencing ./internal/sessckpt ./internal/sessprofile ./internal/sessquery ./internal/sessstate ./internal/termbind ./internal/terminstance`

| Measurement | Before: parent `964fa472` | Candidate after |
| --- | ---: | ---: |
| Package summaries | 9 PASS | 9 PASS |
| Test-status keys (package rows included) | 2,018 | 2,030 |
| Actual test rows | 2,009 | 2,021 |
| Added status keys | — | 12 named additions |
| Removed status keys | — | 0 |
| Changed statuses | — | 0 |

The 12 additions are: `TestEmitReachesAppendAdmissionGateAfterStaleObservation`,
`TestEmitParkedReachesAppendAdmissionGateAfterStaleObservation`,
`TestEmitReachesSameEpochAppendAdmissionGate` plus its two loser-ID subtests,
`TestEmitParkedReachesSameEpochAppendAdmissionGate` plus its two loser-ID
subtests, and `TestSetProfileRefusesSupersededLeaseWhileTailStillMatches` plus
its lower-epoch and two same-epoch loser-ID subtests. All inputs in the mask
remain PASS; direct sessrepo tests are outside this status set. The four
semantic moved classes are the table above, not four additional status keys.
Raw JSONL, exits, and normalized summary are under
`.temp/BUG-260917-3lddu0/importer-outcomes-rev2/`.

The adopted runtime outcome grid is the reviewer evidence
`BUG-260917-3lddu0_review-runtime-outcomes-rev2.md`. It uses the same complete
nine-package importer set and records actual `AppendEvent` outcomes rather than
test names: **130 keys / 2,926 calls before** and **137 / 2,955 after**; the
outcome set delta is **10 added and 3 removed**. The moved outcome classes are
the lower-epoch losing `profile.changed` (`ADMIT → STALE`), same-epoch losing
events (`DIVERGENT`), lower-epoch `session.parked` (`STALE`), and the three
old-owner `session.idle`/`session.stopped` classes moved to pre-takeover
same-winner history.

## Registry and traceability

Registry case `story-260917-losing-lease-profile-source` is referenced by the
Section 2.4 binding and clause `2.4#2`, and names
`internal/sessrepo/sessrepo.go:AppendEvent`. The independent Section 2.4
`sessprofile.Derive` owner is retained only for the existing derivation cases;
this leaf records its inability to observe lease-store authority as a bound
owned by `STORY-260922-cpkajd` / `TASK-260922-31qyyi`. That bound includes the
never-minted higher-epoch event and empty-lease-store classes; neither is
implemented or decided here.

Canonical registry projection digest:

`174728a8a01e8429004e4f9c4687f8fa1f17406b641012236c3427a758baf31e`

Exact-tree tracecheck:

`traceability ok: contracts=64 normative_sections=36 acceptance_cases=154 fixtures=33 compatibility_contracts=55 assigned_scopes=0`
`section coverage: bindings=69 full=4 partial=9 sliver=9 unevidenced=43 unmeasured=4 unowned=7 clauses_discharged=70/574`

README's measured-coverage fenced line is byte-equal to the printed second
line. `-section 2.4` admits; `-section 5.3` refuses as the expected 7/8
partial binding. Exact logs are under `traceability/`.

## Bounds and carried P3 dispositions

- Unknown higher epoch: `checkWinningLease` admits epoch 3 when the winner is epoch 2; owner bound `internal/sessrepo/chain.go:checkWinningLease` composed by `internal/sessrepo/sessrepo.go:AppendEvent`. This is not independent `2.4 ambiguity closure; legitimate successor creation is a CAS lifecycle outside this leaf.
- Empty lease store: `AppendEvent` skips the winner gate, so valid `(epoch 7, lease B)` sequence 1 can be admitted; owner bound is `AppendEvent`/`checkWinningLease` plus the lease lifecycle caller/store.
- `Run → EmitParked`: unreachable except by an interleaving between `Observe` and `AppendEvent`, closed by the durable gate; owner `internal/axpane/run.go:Run` + `internal/sessrepo/sessrepo.go:AppendEvent`.
- The axpane `decide.go` doc-comment qualifier remains untouched and scoped to local ownership.
- `observeRemoteWinner` structural redundancy is record-only.
- The no-grant remote interactive-owner step-4 product question remains unresolved and undecided.
- Evidence hygiene records the candidate tree OID here and keeps the exact test
  mask next to every census/importer count.

## Validation and handoff

Fresh rev3 validation is stored under
`.temp/BUG-260917-3lddu0/validation-rev3/`. All 30 configured commands exited
0; `task-board validate` reported 179 existing `MISSING_ACTIVITY` diagnostics
but exited 0. Scoped tracecheck §2.4 exited 0; §5.3 exited 1 as the expected
7/8 partial-binding refusal. Candidate remains
UNCOMMITTED for Story snapshot handoff; no review acceptance or main
integration is claimed.
