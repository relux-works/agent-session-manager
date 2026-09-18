# TASK-260830-kkh1an — independent review verdict, CR-TASK-260830-kkh1an-4 revision 4

**Verdict: ACCEPTED (`accept_cr`, routed `integrating`).** No P1, no P2.
Both rework classes are closed as classes on my own instruments: F3
(grant-less stale members) fences on every member of the grant axis × both
directions × both stale tuples × every live source, decided by the landed
`fencing.StaleRelativeToWinner` verdict composed in `ObserveFencing` (my
independent census: 144 fencing cells and 336 keep cells all as the oracle
predicts; a 31,104-cell exhaustive oracle census and a 6,912-cell
gate-vs-verdict differential census both hold; my three composition/edge
narrowings on the grant axis, the direction axis and the older-epoch edge
are KILLED ×2 by the committed suite). E3 (expiry axis of the per-effect
recheck) is pinned: the entry-instant mutant that survived rev3 is now a
shipped row (`N-engine-recheck-auth-expiry`, KILLED ×2) with the committed
witness, and both E1 `expired` members kill on the receipt. P3-1..P3-5 of
rev3 are closed (witnesses committed with mirror rows; P3-5 stated). Four
P3 evidence items remain (below), each with a committable witness in the
evidence archive; none is a production defect and none is a member of the
two rework classes, so they go to the Story's final leaf.

Reviewer: RUN-260918-889432 (claude-opus-5 max). Reviewed bytes: base
`c61fc06ad8190a73c3005e137282829936dac970` (predecessor checkpoint = Story
branch tip), candidate tree `7d4619129f1e8583b5c0dc46525edcaed43f471c`
(recomputed from the live Story worktree with a temporary untracked-aware
index: equal), patch sha256
`20a5cbd1e5216641f2eec36c38e7225737c5558698edff43e3af5d862460a211` (equal to
the CR record). `origin/main` is still `c3aae73` (fetched). Every probe ran in
isolated `git archive` copies of the exact tree under
`.temp/TASK-260830-kkh1an/review-rev4/` (`cand/` tree-exact checks, `cand-h/`
shipped harness, `cand-m/` reviewer mutants, `cand-p/` probes); `cand-h` is
byte-identical to the candidate after both harness passes and `cand-m`
differs from it only by the reviewer probe file after both mutant passes.
The live Story worktree, index, branch and HEAD were never touched
(`git status` is unchanged: `M LOGBOOK.md`, `M README.md`,
`M internal/axpane/decide.go`, `M internal/fencing/doc.go`,
`M internal/terminalbackend/…` ×4, `?? internal/fencing/staleness.go`,
`?? internal/fencing/staleness_test.go`, `?? internal/terminstance/`).
`PYTHONDONTWRITEBYTECODE=1` everywhere, 0 `__pycache__` in any copy. Every
instrument, raw log and subprocess exit is in
`TASK-260830-kkh1an_review-evidence-rev4.tar.gz` (paths below are relative to
that archive).

## 0. Rework verification (rev3 findings), graded as CLASSES on my own executed evidence

### F3 — grant-less stale incarnations (P2 production, `repeat-of: rev2 F2`)

Fix landed: `internal/fencing/staleness.go` exports `StaleRelativeToWinner`
(direction/tuple arms over a verified, well-formed winner, no grant
precondition, `LocalHostID` never read) and `internal/terminstance/fencing.go`
consults it on every non-park outcome of the direct question, fencing from a
live source on decided-stale. The `internal/fencing` delta is confined to the
new file, its tests and one `doc.go` sentence (`gate.go` untouched:
`git diff --stat` shows exactly `doc.go +5/-1`, `staleness.go +71`,
`staleness_test.go +229`).

Class enumeration (mine, `probes/zz_review4_probe_test.go`
`TestRV4_F3ClassCensus`, `logs/12`): grant axis {fresh, absent, lapsed,
no-clock, unusable-policy, garbage-token-with-fresh-expiry} × direction
{local, remote} × stale tuple {older epoch/losing lease, same-epoch loser,
older epoch/winning lease} × source {creating, parked, active, quiescing} =
144 cells: **144 fence** to the literal `stale_fenced` with `transitioned`
true and nil error. The same 6 × 2 × 3 over the four non-live sources (144
cells) and the decided-not-stale tokens {winning tuple, future epoch} over
all eight sources under every grant member and direction (192 cells): **336
keep** the state. So every member the CR4 brief names (grant present /
lapsed / absent × local / remote × active / parked / quiescing) is driven,
plus the two members the producer added (no clock reading, unusable policy)
and creating.

Independence of the oracle: `TestRV4_ObserveFencingMatchesIndependentOracle`
restates the rule in the test (well-formed presented token naming the
observation's session; established well-formed winner; verified, unambiguous,
not a failed handoff; older epoch or same-epoch lease mismatch; live source)
and compares `ObserveFencing` against it over sessions {A, B, garbage} ×
epochs {0..3} × leases {A, B, garbage} × 9 ownership shapes × 6 grant shapes ×
2 directions × 8 sources = 31,104 cells, 288 of them fencing: **0
disagreements**. `TestRV4_VerdictIsTheGatesTupleFact` checks "no second
staleness rule": over 6,912 (token, observation) cells, wherever the landed
`Authorize(restore)` asked from the winner's host with a fresh grant reaches
its tuple arms (nil, `stale_owner` park, or the future-epoch `restore_policy`
park — 144 cells), the verdict over the SAME observation under ANY grant
member agrees (stale ⇔ `stale_owner`), and wherever the gate refuses or
parks before the grant arms the verdict is undecided: **0 disagreements**.

Reviewer narrowings on the composition (`probes/review4_mutants.py`,
`logs/50-rv4-mutants-pass{1,2}.log`, raw logs in `logs/rv4-mutants-pass{1,2}/`):
`RV4-M2` (fence grant-less only when `!HasGrant`: lapsed/no-clock/unusable
no longer fence) **KILLED ×2**; `RV4-M3` (fence grant-less only under a
local winner) **KILLED ×2**; `RV4-M4` (verdict older-epoch arm widened by
one: exactly one epoch older decides not-stale) **KILLED ×2** — all by the
committed suite (`TestRV3F3_GrantLessStaleFences` and the `internal/fencing`
tuple tests). Shipped verdict rows `N-fencing-verdict-*` (11) and
`N-fencing-verdict-composition` KILLED ×2 in my harness reruns.

Why the producer's completeness argument holds: the direct question's only
non-park refusals for a stale tuple over a verified well-formed winner are
the four grant-precondition arms (`gate.go`: no grant, no clock reading,
lapsed grant, unusable policy — in that order, after the ownership arms and
before direction); the verdict never reads `Grant`/`Now`/`Policy`, so no
fifth grant member can hide behind it; the relative question carries the
direct question's grant facts unchanged and the grant arms never read
`LocalHostID`, so a direct `remote_owner` park implies the relative question
passes the grant arms. My differential census is the executed form of that
argument. **Closed as a class.**

One documented member is kept undecided by design: `HandoffFailed` (the
caller's own losing force lease). The producer states it in `doc.go`,
`fencing.go`, TRACEABILITY row 87 and the results ("ownership history, not a
grant precondition"), drives it (`TestRV3F3_GrantLessUndecidedLeavesState/failed_handoff`,
row `N-fencing-verdict-handoff` KILLED ×2), and no production caller sets
`HandoffFailed` today (`grep`: only tests). Recorded as a note for the final
leaf (below), not a finding: it is an ownership arm, not a member of the
grant class.

### E3 — expiry axis of the per-effect recheck (P2 evidence, `repeat-of: rev2 E2`)

`TestRV3W_M1_AuthExpiryRecheckedBeforeEachEffect` is committed verbatim
(`review_witness_test.go`): request-stop, deadline `2026-09-03` > expiry
`2026-09-02`, `Now` advanced past the expiry after effect 1 → literal
`terminal_backend_unauthorized` / `ax authorization expiry`,
`after=unavailable`, `new_authorization`, exactly `[graceful_stop_requested]`
committed. Shipped row `N-engine-recheck-auth-expiry` keeps the per-effect
tuple recheck and freezes only the expiry instant (`entryNow`): **KILLED ×2**
(`logs/mutants-pass{1,2}/N-engine-recheck-auth-expiry.log`). The fixture gained
`fixtureLateDeadline`/`fixturePastExpiry`, and both E1 `expired` members
(`TestExecuteCreateEntryAuthorizationRefuses/expired`,
`TestExecutePreEffectErrorsRestoreSource/entry_auth_expired`) now run with the
deadline past the expiry so a create-exempting mutant binds the receipt and
kills on the receipt assertion (rows `N-engine-auth-entry`,
`N-engine-auth-entry-quiesce` KILLED ×2).

Class enumeration (mine): axes {kind, expiry, tuple} of `CheckAuthorization`
plus the generation recheck, × positions {1, 2, 3} of the three-effect stop
row. Kind cannot rotate mid-operation (one kind per context). Tuple: positions
1/2/3 pinned (`N-engine-recheck-auth`, `…-third`). Generation: positions 1/2/3
pinned. Expiry: the realistic one-line mutant (entry instant for every
position) is killed at position 2; **position 3 alone is unmeasured** — my
`RV4-M1` (expiry evaluated at the entry instant only when two effects have
already committed) **SURVIVED ×2** against the committed suite while my
witness `TestRV4W_ExpiryRecheckedBeforeThirdEffect` fails under it ×2 ("error
= nil": the third effect commits under an expired authorization). Graded P3
(below), not a repeat of E3: the production gate is one shared line per
effect, the mutant the class was graded on is dead, and the witness is
committable as is. **Closed for the member the finding named; one residual
position member recorded as P3-1.**

### P3-1..P3-5 of rev3

| Rev3 item | What I executed | Grade |
| --- | --- | --- |
| P3-1 evidence bound edge | `TestRV3W_M2_EvidenceBoundEdge` committed (256/257 on `CheckResult` and `ExecuteStatus`); row `N-result-evidence-bound` KILLED ×2 | Closed |
| P3-2 conditional disposition at the engine | `TestRV3W_M3_ConditionalCapabilityDisposition` (3 subtests) committed; row `N-engine-conditional-disposition` KILLED ×2 | Closed |
| P3-3 `last_operation_id` grammar | `TestRV3W_M8_StatusLastOperationGrammar` committed; row `N-status-last-operation-empty` KILLED ×2 | Closed |
| P3-4 proof-kind sibling names | `TestRV3W_M4_ProofKindSiblingNamesRefused` committed; sibling corpora added to all three enum tests; row `N-proof-sibling` KILLED ×2 | Closed (the mixed-case spelling member of the same class is P3-4 below) |
| P3-5 interim-proven create member | TRACEABILITY stated bound: interim `creating` with identity match refuses uncertain, key parked, exit `unavailable → terminate-stale` (final leaf); absent-proven member resumes | Closed (stated) |

## 1. What holds (verified myself on the exact tree)

| Check | Result | Evidence |
| --- | --- | --- |
| gofmt (empty list) / `go build ./...` / `go vet` on the four touched packages; `GOOS=windows go vet` terminstance+fencing; `GOOS=linux` build | all exit 0 | `logs/02` |
| `go test ./internal/{terminstance,fencing,terminalbackend,axpane} -count=1 -v` | exit 0; 397 top-level `--- PASS`, 0 FAIL, 0 SKIP | `logs/01` |
| `go test ./... -count=1` on the exact tree | exit 0, 40/40 packages `ok` | `logs/04` |
| `-race` on the four packages; `-cover` terminstance 86.0% / fencing 97.8%; `git diff --check` base..candidate | all exit 0 | `logs/70` |
| tracecheck (`contracts=64`, `clauses_discharged=56/569` — trunk figures, registry untouched) / `cataloggen -adopted … -check`; tree unmodified by both | exit 0 / exit 0 | `logs/71` |
| CR construction suite | runtime-recorded `required=30 green=30 failed=0 missing=0` on the exact tree (accepted from the attached bounded `_rev4-validation.log`, which shows 19 of the 30 `[exit 0]` lines before its 64 KiB bound; the whole-repo race gate, the 17 fuzz gates, JSON validity and `task-board validate` were not rerun by me — the producer's archive carries the race gate in 4 chunks, 40/40, 0 `DATA RACE`, `.go` sha listing identical before/after) | resource, `logs/81` |
| Shipped harness, isolated pristine copy, two full passes with raw per-plant output | 116/116 both passes (114 narrowing KILLED + 1 tightening KILLED + `C-control` SURVIVED), verdict lists identical incl. exits, 116 raw logs per pass with subprocess exits, `cand-h` byte-identical to the candidate after both passes, 0 `__pycache__` | `logs/40-harness-pass{1,2}-verdicts.log`, `logs/mutants-pass{1,2}/` |
| Real-kill evidence | producer `TestExecuteCrashChildSelfTerminates` / `TestExecuteCrashCreateResumesAfterStatus` PASS ×2 (real SIGKILL in the store `AfterCommit` hook); rev3 reviewer's engine seams `TestRV3_RealKill{BeforeFirstCreateEffectThenResume,AfterFirstCreateEffect}` PASS ×2 on the rev4 tree; my own new seam `TestRV4_RealKillAfterLastStopEffectBeforeCompletion` PASS ×2 (below) | `logs/60-*`, `logs/61-*` |
| Idempotency | `TestRV4_IdempotencyReplayAndMismatchThroughEngine`: identical retry replays byte-equal with `replay_same` and one effect; changed operation ID in-window → `idempotency_mismatch` / `idempotency key conflict`, source restored, `status_first` | `logs/12` |
| Spec fidelity (§4.C extracted from `SPEC.v0.7.0.md` lines 1082–1218): eight states through the landed parser, ten operations / ten side effects / transition matrix / allowed-error sets composed from `terminalbackend` (no second enum: no state or effect literal in production, `grep`); `AXAuthorization` closed 7-member set; `RetryDisposition`, `AuthorizationKind`, `ProviderProofKind` literals; key material as whole strings (`…bbb1/quiesce/…ddd1`, `…/boundary/…/provider_process_exit`, `…/stop/sha256:bbb…`, `session/bootstrap`); epoch bound 1 / 9007199254740991 admitted, 0 / 9007199254740992 / `1.0` / `-1` refused at `ParseAXAuthorization`, through the nested `MutationContext` and through the quiesce-input body; generation 256 chars (ASCII and `é`) admitted, 0 / 257 refused with the landed `terminal_backend_stale_generation` on the context and the status body | holds | `TestRV4_SpecLiteralsAtEntries`, `logs/12` |
| Row sets: an outside-set backend code on quiesce-input (`terminal_backend_unavailable`), wait (`stop_timeout`) and stop (`quiesce_timeout`) → `terminal_backend_protocol_error` / `operation error vocabulary`, `after=unavailable`, `status_first`; status timeout and uncoded read failure → error with NO report (unknown, never absent); status outside-set code → protocol error | holds | `TestRV4_OutsideSetCodeRefusedPerRow`, `logs/12` |
| State-entry rules: `creating` only via create after the receipt (`InterimState` + `AfterReceipt`, real kills); pre-effect errors restore the source; post-effect unprovable → `unavailable`/`status_first`; only quiesce-input → `quiescing`; `stale_fenced` only via `ObserveFencing` (no `StateStaleFenced` in the engine); per-effect tuple + generation + deadline rechecks at positions 1–3 and the expiry recheck at position 2; a deadline never cancels a committed effect | holds (production unchanged since rev3 except `fencing.go`; rev3 instruments re-run) | source read, `logs/40`, `logs/50` |
| Composition: `fencing.Authorize` ×2 + `fencing.StaleRelativeToWinner` + `ParkDetails`; no `Winner.Epoch`/`Winner.LeaseID`/`presented.*` reference in any terminstance production file outside the landed calls (`grep` over the package, not only `fencing.go`) | holds | `logs/80`, `TestRV3_FencingNoLocalTupleComparison` |
| Bounds and hygiene: changed paths = exactly the 44 candidate paths; trunk `c3aae73`→candidate touches only the Story's paths; `internal/traceability` untouched; `task-board.config.json` equal to base and to `origin/main`; README +55/−0 (test + harness commands, "adds no `ax` command, no `doctor` result, and no runtime capability claim"); LOGBOOK +29/−0 newest-first under `## 2026-09-18`; 0 `__pycache__`/`.pyc` in the tree; `tmux`/`ConPTY`/`os/exec` absent from production | holds | `logs/80` |
| Producer evidence archive: real gzip, task-scoped README, 280-entry `MANIFEST.sha256` verifying 0 mismatches, 116 raw per-plant logs on both passes, `cmd01..cmd30` (race in 4 chunks, 40 distinct packages, 0 `DATA RACE`), `.go` sha identical before/after the race gate, production blobs identical before/after/across passes, 0 pycache, no foreign-task artefacts (the only foreign ID is inside the captured `task-board validate` output) | hygienic | `logs/81` |

## 2. Coverage statement

Producer claim: 93 of 93 rows driven. Re-derived (`logs/82-ratio.log`): 93
numbered data rows (1..93, no gaps, unique); 109 distinct cited top-level
tests, every one present in `go test -list` (terminstance + fencing +
terminalbackend) and every one `--- PASS` in my verbose run; 52 cited
subtests all `--- PASS`; rows 53 and 54 are cross-references to driven rows,
as in rev3. **I measure 93 of 93 driven** at the named production entries.
One row's prose outruns its test by one position: row 89 ("auth expiry
rechecked before each effect") is driven at the second stop effect only
(P3-1). Row 87's "failed handoff (park preserved)" is exactly what the test
drives.

## 3. Findings

No P1: the eight-state enum, the operation/effect vocabularies and the
transition matrix are the landed ones; no second state machine, no second
fencing model (the verdict is landed in `internal/fencing` and agrees with
`Authorize` on every cell of the differential census).

No P2.

### P3 (evidence; each with a committable witness in `probes/zz_review4_probe_test.go`, all PASS pristine and FAIL ×2 under their plant)

- **P3-1 Expiry recheck unmeasured at the third position.** `RV4-M1`
  (`execute.go`: expiry evaluated at the entry instant only once two effects
  have committed; the tuple recheck and positions 1–2 unchanged) SURVIVED ×2.
  Witness `TestRV4W_ExpiryRecheckedBeforeThirdEffect`: request-stop, late
  deadline, `Now` past the expiry after effect 2 → pristine refuses
  `ax authorization expiry` / `after=unavailable` / `new_authorization` with
  exactly `[graceful_stop_requested process_closed]` committed and no
  completion; mutant commits the third effect and completes. Same position
  class the producer closed for the tuple and generation axes (`-third`
  rows); add the witness and a `-third` row for the expiry axis.
- **P3-2 The replay seam validates the stored image at the helper only.**
  `RV4-M6` (`execute.go` replay site: `CheckResult` refusals tolerated for the
  stale-generation member) and `RV4-M6b` (the replay-site refusal reports
  `replay_same` instead of the class-mapped `status_first`) both SURVIVED ×2:
  no committed test replays a stored completion against a request whose
  validated binding generation moved, so the engine's composition of
  `CheckResult` on the replay path (and the disposition of that arm) is
  unpinned — the `N-result-*` rows are all killed at the `CheckResult` helper.
  Witness `TestRV4W_ReplayedImageGenerationBindingAtEngine`: a quiesce
  completed under `generation-one`, then the identical key/operation ID
  replayed with `backend_generation=generation-two` and
  `CurrentGeneration=generation-two` → pristine refuses
  `terminal_backend_stale_generation` / `result generation binding`,
  `after=unavailable`, `status_first`, no second effect; under `RV4-M6` the
  stale image is handed back as a successful replay. The unmarshal arm of the
  same seam is characterized (`TestRV4_CorruptCompletionImageIsIntegrityFailure`:
  a `{not json` completion → `terminal_backend_integrity_failure` /
  `idempotency result image`, `unavailable`, `status_first`, no effect) and
  is likewise unpinned by any committed test. Rule-pinned-at-the-helper class
  (rev3 P3-2's class); commit both witnesses with rows on the engine site.
- **P3-3 In-loop deadline instant unwitnessed.** `RV4-M7` (`execute.go` loop:
  `!Now().Before(deadline)` → `Now().After(deadline)`, admitting exactly the
  deadline instant between effects; the entry check unchanged) SURVIVED ×2.
  Witness `TestRV4W_LoopDeadlineInstantCancelsWaiting`: `Now` equal to
  `deadline_at` after the first stop effect → pristine `stop_timeout` /
  `operation deadline`, `after=unavailable`, `status_first`, one committed
  effect; mutant commits the remaining effects. Adjacent-edge class; the entry
  instant is pinned (`N-engine-deadline-instant`), the loop instant is not.
- **P3-4 Closed-enum corpora carry no mixed-case sibling spellings.** `RV4-M5`
  (`auth.go` default arm admits exactly `Control` as `control`) SURVIVED ×2:
  the corpora hold all-caps (`CREATE`, `REPLAY_SAME`, `PROVIDER_QUIESCENCE`)
  and landed sibling names but no title-case member, and the copy-counting
  AST census cannot see a default-arm admit. Witness
  `TestRV4W_AuthorizationKindCaseSiblingRefused` (`Control`, `Create`,
  `CONTROL`, ` control`, `control `, `Force_Stale`, `Restore` at the direct
  entry and through the AXAuthorization document; the same class on
  `ParseRetryDisposition` and `ParseProviderProofKind`). Same class as rev3
  P3-4; extend the three corpora.

### Notes for the final leaf (not findings)

- The `HandoffFailed` member of the ownership preconditions is kept
  undecided by an explicit, documented product decision; from a live state
  it therefore cannot reach `stale_fenced`, and the final leaf's
  `terminate-stale` row (`stale_fenced|unavailable → stopped`) cannot target
  it. No production caller reports `HandoffFailed` today; if the final leaf
  wires the events-and-recovery flow that does, decide then whether a stale
  tuple under a failed handoff should fence and drive it.
- The rev3 characterizations stand unchanged (target-proven same-key retry
  regresses the local observation to `unavailable`; wait resumption
  re-performs `safe_boundary_observed`).

## 4. New crash evidence (mine)

`probes/zz_review4_crash_unix_test.go` `TestRV4_RealKillAfterLastStopEffectBeforeCompletion`
(PASS ×2, `logs/61-rv4-realkill-run{1,2}.log`): the child runs the production
`ExecuteMutating` request-stop and SIGKILLs itself in the engine `AfterEffect`
hook of `backend_store_closed` — every effect committed, the completion image
not yet durable (the row's "lost result"). Parent: the receipt is durable and
no completion exists; the same-key retry with status proving `stopped`
(closed) refuses `terminal_backend_unavailable` / `idempotency result
uncertain`, `after=unavailable`, `status_first`, performs no effect and makes
exactly one reconciliation read; the same-key retry with status proving the
source (`quiescing`) resumes under the same receipt, performs the three
effects once in transition order, records one completion and exports exactly
one receipt; a further identical retry replays with no new effect. That is
the §4.C stop row's "lost result requires status, then same-key retry only if
not closed" executed at a seam neither the producer nor the rev3 review had
cut.

## 5. DoD walk (live merged checklist)

Production entry points implement the scoped deliverable — yes, including
the grant-less fencing members. Positive/negative/recovery tests pass with
logs — yes. README/traceability without unsupported claims — yes (row 89's
"before each effect" is one position short; P3-1). Uncommitted in the Story
worktree — yes (tree OID equal, HEAD = checkpoint). Ratio — 93 of 93 with
call sites. Negative tests that fail when the gate admits — yes for every
AC-named gate; the four P3 residuals are unmeasured members or seams, not
admitting gates. Narrowing mutant per gate — shipped 115 KILLED ×2 with a
working control; my 7 narrowings: 3 KILLED ×2, 4 SURVIVED ×2 with
non-equivalent witnesses (P3-1..P3-4). No source-text gate in production
(the composition pin is a test instrument). Lint/build — clean.
Outcome artefacts — attached and hygienic. Logbook — newest-first, additive.
Implementation matches AC — yes. Fits the architecture — yes: the verdict is
exported by the landed fencing owner and composed, never re-implemented; no
second enum, no second state machine. Tests green — yes. Gates attacked, not
read — yes.

## 6. Instruments (all in the evidence archive)

`probes/zz_review4_probe_test.go` (11 top-level probes: 3 census/differential
probes, 4 witnesses `TestRV4W_*`, 4 re-verifications/characterizations; all
PASS pristine, `logs/12`), `probes/zz_review4_crash_unix_test.go` (1 real-kill
probe, PASS ×2, `logs/61`), `probes/zz_review3_crash_unix_test.go` (the rev3
reviewer's engine-seam real kills, re-run ×2 on this tree, `logs/60`),
`probes/review4_mutants.py` (7 narrowing rows + 3 controls: `RV4-K` known
kill KILLED, `RV4-C` comment SURVIVED, `RV4-B` build break `ERROR(build
broke)`; two passes with identical verdicts; raw per-plant logs with exits and
diffs in `logs/rv4-mutants-pass{1,2}/`), `probes/run_harness.sh` +
`probes/split_raw.py` (shipped harness driver; `logs/40-*`,
`logs/mutants-pass{1,2}/` 116 raw logs each), `logs/01`, `02`, `04`, `70`,
`71`, `80`, `81`, `82`, `cand*.sha` (tree listings proving the copies were
restored), `REVIEW-MANIFEST.txt` (sha256 of every file).

Verdict recorded by RUN-260918-889432 via `accept_cr`; no product edits,
commits, checkpoints or integration were performed; the isolated copies are
deleted after attachment. `commit_ack` is not supplied.
