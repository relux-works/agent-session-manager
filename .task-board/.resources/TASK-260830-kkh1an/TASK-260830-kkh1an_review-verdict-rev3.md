# TASK-260830-kkh1an — independent review verdict, CR-TASK-260830-kkh1an-3 revision 3

**Verdict: CHANGES REQUESTED (routed `to-dev`).** No P1. One P2 production
finding (F3: `ObserveFencing` never fences a stale incarnation whose
observation carries no live fencing grant — the lapsed-grant and the
no-grant/controller-restart members of the cold force takeover, under a local
and a remote winner alike; `repeat-of: rev2 F2` class), one P2 evidence finding
(E3: the per-effect authorization recheck is pinned on the lease-tuple axis
only; an authorization that EXPIRES between effects commits the remaining
effects under a one-line mutant with the whole suite green; `repeat-of: rev2 E2`
class), and five P3 items. Everything else holds on the exact tree and is
confirmed by my own instruments: F1 (same-key retry after `status`) meets the
behavioural bar at the producer's seam and at my own engine seam with real
SIGKILLs, F2's remote-winner arms fence and compose the landed gate, E1's
create-row gates and E2's third-position rechecks and session axis are pinned
and their mutants die, the shipped harness reproduces 100/100 twice, the ratio
re-derives to 84 of 84, the full configured suite is green, hygiene is clean.

Reviewer: RUN-260918-f6b44c (claude-opus-5 max). Reviewed bytes: base
`c61fc06ad8190a73c3005e137282829936dac970` (the predecessor checkpoint = the
Story branch tip), candidate tree `94dc630b5e2eb6c49e13e4b0beff7ee4b434493f`
(recomputed from the live Story worktree with a temporary untracked-aware
index: equal), patch sha256
`ccac72639080b50614afc55463b6d4b68dcdab2b47055e38cf92364163557ba1` (equal to
the CR record). Every probe ran in isolated `git archive` copies of the exact
tree under `.temp/TASK-260830-kkh1an/review-rev3/` (`cand/` tree-exact checks,
`cand-h/` shipped harness, `cand-m/` reviewer mutants, `cand-p/` probes; the
copies' index trees equal the candidate OID before and after every run, and
`cand-m`/`cand-p` differ from the candidate only by the two reviewer probe
files). The live Story worktree, index, branch and HEAD were never touched
(`git status` after the review is unchanged: `M LOGBOOK.md`, `M README.md`,
`M internal/axpane/decide.go`, `M internal/terminalbackend/…` ×4,
`?? internal/terminstance/`). `PYTHONDONTWRITEBYTECODE=1` everywhere, 0
`__pycache__` in any copy. Every instrument, raw log and subprocess exit is in
`TASK-260830-kkh1an_review-evidence-rev3.tar.gz` (paths below are relative to
that archive).

## 0. Rework verification (rev2 findings), graded on my own executed evidence

| Rev2 finding | What I executed | Grade |
| --- | --- | --- |
| **F1** key poisoned forever; "status, then same-key retry" unimplementable | Producer real-kill tests `TestExecuteCrashChildSelfTerminates` / `TestExecuteCrashCreateResumesAfterStatus` PASS ×2 (`logs/60-realkill-run{1,2}.log`). My own real SIGKILL at the ENGINE `BeforeEffect` seam of `create` (`probes/zz_review3_crash_unix_test.go`, `TestRV3_RealKillBeforeFirstCreateEffectThenResume`, PASS ×2): child killed by SIGKILL, receipt durable, no completion, no effect ran → same-key retry with the backend read failing refuses `terminal_backend_unavailable` / `unavailable` / `status_first` with no effect → status proves absence (canonical non-match form) → same-key retry completes `active`/`replay_same` with exactly one instance's `[binding_persisted wrapper_started]`, one completion, exactly one exported receipt → a further identical retry replays the one result with no new effect. `TestRV3_IdempotencyReplayAndMismatchThroughEngine`: identical retry replays byte-equal; changed operation ID in-window is `idempotency_mismatch` / `idempotency key conflict` / source restored / `status_first`. | **Closed.** Behavioural bar met at two seams. |
| **F2** stale incarnation under a remote winner never fenced | Committed `TestObserveFencingRemoteWinner{StaleEpoch,LosingLease}Fences` drive both remote members from `active`, `parked`, `quiescing`; `…WinningTokenLeavesState` pins the no-transition member. `TestRV3_RemoteWinnerLeaseSummaryHostIsTheOnlySwitch`: the direct question parks `remote_owner`, the relative question (only `LocalHostID` switched to `Winner.HolderHostID`, nothing fabricated) parks `stale_owner` with the winning lease `f47ac10b-…`. `TestRV3_FencingNoLocalTupleComparison`: `fencing.go` code lines reference no `presented.Epoch/LeaseID`, `Winner.Epoch/LeaseID`, `Grant.Token`; exactly two `fencing.Authorize(` calls. Shipped rows `N-fencing-remote`, `N-fencing-operation-remote`, `N-fencing-remote-stale-only` KILLED ×2; my `RV3-M6` (remote path skips the live-source gate) KILLED ×4 by the committed suite. | **Closed for the fresh-grant members; the grant-less members of the same class are open — F3 below.** |
| **E1** create-row gates positive-path only at the engine | Shipped rows `N-engine-auth-entry` / `N-engine-generation-entry` are now the create-exempting mutants (`&& operation != terminalbackend.OperationCreate`) keyed to `TestExecuteCreateEntryAuthorizationRefuses` / `…GenerationRefuses`: KILLED ×2 with the receipt witness on the control-kind and foreign-lease members (`logs/mutants-pass1/N-engine-auth-entry.log`: "Lookup(key) found a receipt for the unauthorized create, want none bound") and the literal-token witness on the expired member; `N-engine-headless-absent` and `N-engine-committed-timeout-disposition` KILLED ×2. | **Closed.** (One note: the `expired` member kills on the code literal, not on the receipt — see E3.) |
| **E2** recheck pinned at one position; session never compared | `N-engine-recheck-{auth,generation}-third` KILLED ×2 (rotation at the fourth `CurrentLease`/`CurrentGeneration` read, two effects committed); `N-status-session-{scoped,exact}` KILLED ×2; `N-status-match-*` ×5 KILLED ×2. | **Closed** for the tuple and generation axes; the expiry axis is E3. |
| P3-1 pre-link store failure | `TestExecutePreLinkStoreFailureRestoresSource` (read-only pending dir → `after=active`, `required_operator_action`, no receipt); `N-engine-bind-{prelink,postlink}` KILLED ×2 | Closed |
| P3-2 fencing sources | `fenceLive`: creating/parked/active/quiescing fence; absent/stopped/stale_fenced/unavailable surface the park; `N-fencing-source-{absent,stopped,creating}` KILLED ×2 | Closed |
| P3-3 deadline instant | `TestExecuteEntryDeadlineInstantRefuses` (4 rows, no receipt) / `TestExecuteStatusDeadlineInstantRefuses` (no observation); rows KILLED ×2 | Closed |
| P3-4 `attach`/`none` kind negatives | `N-auth-kind-{attach,none}` KILLED ×2 | Closed (the class recurs on another enum — P3-4 below) |
| P3-5 match conjunction per axis | `TestExecuteStatusIdentityMatchEachAxis` 10 subtests; 5 rows KILLED ×2 | Closed |
| P3-6 `kindFor` twin | landed `terminalbackend.TransitionAuthorization` accessor + column test + census/inventory/audit registration; `TestKindForAgreesWithLandedTable`; `N-engine-kind-map` re-keyed KILLED ×2; the three terminalbackend pins PASS (`logs/82`) | Closed |
| P3-7 `unavailable`+`new_authorization` | kept as the leaf's stated mapping contract, recorded as a literal tension in TRACEABILITY | Acceptable (stated) |
| `holder_host_id` unbound | stated in `CheckAuthorization` doc + TRACEABILITY | Acceptable (stated) |
| TRACEABILITY 70/71 + "recovers through status" prose | reworded to the resumed behaviour; LOGBOOK carries an explicit REVERSAL entry | Closed |

## 1. What holds (verified myself on the exact tree)

| Check | Result | Evidence |
| --- | --- | --- |
| gofmt / go vet / go build; GOOS=linux and GOOS=windows builds; GOOS=windows vet of the package | all exit 0 | `logs/02` |
| `go test ./internal/terminstance -count=1 -v` | exit 0, 101 top-level `--- PASS` (224 incl. subtests), 0 FAIL, 0 SKIP | `logs/01` |
| `go test ./... -count=1` on the exact tree, rerun by me | exit 0, 40/40 packages `ok` (4m34s) | `logs/04` |
| `-race` on terminstance, terminalbackend, axpane, fencing; `-cover` terminstance 85.6%; `git diff --check` | all ok / exit 0 | `logs/70` |
| tracecheck (registry untouched: contracts=64, clauses 56/569 — trunk figures) / `cataloggen -adopted … -check` | exit 0 / exit 0, tree unmodified | `logs/71` |
| CR construction suite | runtime-recorded `required=30 green=30 failed=0 missing=0` on the exact tree (accepted from the attached `_rev3-validation.log`; the whole-repo race gate, the 17 fuzz gates, JSON validity and `task-board validate` were not rerun by me) | resource |
| Shipped harness, isolated pristine copy, two full passes with raw per-plant output | 100/100 both passes (98 narrowing KILLED + 1 tightening KILLED + `C-control` SURVIVED), verdict lists identical, 100 raw logs per pass with subprocess exits, `cand-h` index tree equal to the candidate OID and working tree clean after each pass | `logs/40-harness-pass{1,2}-verdicts.log`, `logs/mutants-pass{1,2}/`, `logs/40-harness-driver.log` |
| Real-kill evidence | producer's two seams PASS ×2; my engine `BeforeEffect` seam PASS ×2; my `AfterEffect`-of-`binding_persisted` seam characterized ×2 (below) | `logs/60` |
| Ratio | 84 data rows counted in `TRACEABILITY.md` (numbered 1..84, unique); 95 distinct cited top-level tests, every one present in `go test -list` (94 in-package + the terminalbackend column test) and every one `--- PASS` in my verbose run; every explicitly cited subtest PASS; rows 53/54 are cross-references to driven rows. | `logs/82` |
| Spec fidelity (§4.C extracted from `SPEC.v0.7.0.md` lines 1082–1218 and diffed): eight-state enum, ten operations, ten side effects, transition matrix, allowed-error sets, key shapes all landed (`terminalbackend`) and composed, never re-spelled; `AXAuthorization` closed 7-member set; epoch bound witnessed at 1, 9007199254740991, 0, 9007199254740992 at `ParseAXAuthorization` (committed) and through the nested `MutationContext` and the quiesce-input BODY (`TestRV3_EpochBoundThroughNestedSurfaces`); `RetryDisposition` closed 4 (copy-counting census + parse pin, literal tokens at `conformance_test.go:235`); `ProviderProofKind` closed 3; `AuthorizationKind` closed 4; key material asserted as whole literal strings (`TestRV3_KeyMaterialLiteral`: `…bbb1/quiesce/…ddd1`, `…/boundary/…/provider_process_exit`, `…/stop/sha256:bbb…`, `session/bootstrap`); an outside-set code on the first effect of each of the four rows is `terminal_backend_protocol_error` / `operation error vocabulary` and status's outside-set report the same, status timeout and uncoded read failure return an error with NO report (`TestRV3_OutsideSetCodeRefusedPerRow`) | holds | `logs/12` |
| State-entry rules: `creating` only via create after the receipt (`InterimState` + `AfterReceipt`, real kill); pre-effect errors restore the source; post-effect unprovable → `unavailable`/`status_first`; only quiesce-input → `quiescing`; `stale_fenced` only via `ObserveFencing` (no `StateStaleFenced` in the engine); per-effect tuple+generation rechecks at positions 1, 2, 3; deadline never cancels a committed effect (`RV3-M7` KILLED ×4) | holds (with F3/E3 caveats) | source read, harness, `logs/50` |
| Generation bound 0/256/257 (chars not bytes) on the context, the status body, the result and the Go twin with the landed `terminal_backend_stale_generation`; Binding object and CLI Result 4 stated as bounds (verified unchanged from rev2) | holds | committed tests, `logs/01` |
| Composition: `fencing.Authorize`/`ParkDetails` (two composed questions), `terminalbackend.CheckTransition/CheckErrorAllowed/CheckOperation/CheckStatusResult/IdempotencyKey/GenerationDigest/CheckVersionTuple/ParseID/TransitionAuthorization`, `environ.DecodeStrictObject/Check*`, `scalar.Parse*` — no second enum, no local tuple comparison, no second JSON discipline | holds | source read, `TestRV3_FencingNoLocalTupleComparison` |
| Bounds and hygiene: changed paths = exactly the 39 candidate paths; `internal/traceability` untouched; `task-board.config.json` equal to base and to trunk `c3aae73`; 0 `__pycache__`/`.pyc` in the tree; README +52/−0 with test/harness commands and the explicit no-`ax`/no-`doctor`/no-capability sentence; LOGBOOK +21/−0 newest-first with the rev2-F2 REVERSAL; `tmux`/`ConPTY`/`syscall`/`os/exec` in production only inside the two stated-bound comments; trunk→candidate touches only the Story's paths | holds | `logs/80` |
| Producer evidence archive: task-scoped README, 253-entry `MANIFEST.sha256`, 100 raw per-plant logs on both passes, cmd01..cmd29 logs, `.go` sha listing equal before/after the race gate, 0 pycache, 0 foreign-task files | hygienic | `logs/81` |

## 2. Coverage statement

Producer claim: 84 of 84 rows driven. Re-derived: 84 data rows, every named
test and subtest exists and passed (`logs/82`). **I measure 84 of 84 driven**
at the named production entries. Three rows' prose outruns what the tests
show: rows 46/47/77/78 ("stale_fenced only via fencing observation … non-stale
observations leave state") are driven on fresh-grant observations only — the
lapsed-grant and no-grant members of the stale class are not fenced (F3); row
48 ("auth rechecked before each effect") is driven on the lease-tuple axis
only (E3); row 40 ("shape arms … evidence") is driven without the `[0..256]`
edge (P3-1).

## 3. Findings

No P1: the eight-state enum, the operation/effect vocabularies and the
transition matrix are the landed ones; no second state machine, no second
fencing model.

### P2 — production

- **F3 (P2, `repeat-of: rev2 F2` class — a member of the stale-incarnation
  class that `ObserveFencing` does not fence). A stale incarnation whose
  observation carries no LIVE fencing grant is never fenced.** `fencing.go:59`
  asks the landed `Authorize` first as the local host and, on `remote_owner`,
  again from the winner's host; both questions run inside the landed gate,
  whose arm order (`gate.go:295–375`) evaluates `HasGrant` ("carries no grant;
  validate the winning token first", `ErrInvalidArguments`) and grant expiry
  ("lapsed; revalidate", `ErrLeaseConflict`) BEFORE the direction and tuple
  arms. Neither is a park, so `ObserveFencing` returns `(current, false, err)`
  and the incarnation stays `active`/`parked`/`quiescing`. Witnesses
  (`probes/zz_review3_probe_test.go`, `logs/13`):
  `TestRV3_NoGrantHidesStaleness` — `HasGrant=false`, `Verified=true`,
  well-formed epoch-2 winner, presented epoch-1 token, from `active`, `parked`
  and `quiescing`, under a local AND a remote winner: six times "no
  transition"; `TestRV3_LapsedGrantHidesStaleness` — the same with a grant
  validated 2h ago under a 1h refresh interval: "lapsed; revalidate the winning
  token", no transition, local and remote. This is not "refresh first": a stale
  owner can never obtain a fresh grant, because `sessrepo.VerifyFencingToken`
  (`lease_store.go:427–465`) refuses a token below the winning epoch
  (`ErrStaleLeaseEpoch`) and a token from another holder
  (`ErrLeaseHolderMismatch`), and grants are "machine-local transient state:
  never persisted" (`lease_store.go:163–171`), so after a controller restart
  the prior owner has none. These are exactly the cold force-takeover members
  §13.7 exists for (B force-takes because A is silent longer than the refresh
  interval; "the prior owner becomes stale", "the loser MUST stop accepting
  input when it learns the winner"): input IS blocked by the landed input gate
  (fail-closed, same arm), but the Terminal Instance projection never reaches
  `stale_fenced`, so the `terminate-stale` row (`stale_fenced|unavailable →
  stopped`, final leaf) cannot target the incarnation from that state — the
  same consequence rev2 F2 was graded on. The only fenced members are the
  "hot" ones (A still holding a grant younger than the refresh interval when
  it observes the takeover). The doc comment ("any non-stale refusal or park
  (unverified, ambiguous, no winner) surfaces its own error with no
  transition, because only proven staleness fences"), README ("`stale_fenced`
  is entered only by fencing observation … including under a remote winner")
  and TRACEABILITY rows 47/77 name neither member; the observation itself
  proves staleness (verified winner, well-formed, older epoch) and the
  projection says "not stale". Note for fairness: rev2's rework scope offered
  this composition as its first option and it carries this blind spot; the
  finding is new, not a rework miss. Fix without forking: land an exported
  staleness verdict in `internal/fencing` over a verified, well-formed winner
  (the direction/tuple arms only, no grant precondition — the grant governs
  AUTHORIZATION, not the fact of staleness) and compose it for the projection;
  or, if the product decision is that fencing observation is grant-gated,
  state that bound explicitly in `doc.go`/TRACEABILITY/README and name the
  route the final leaf must supply for the lapsed/no-grant members. Either way
  drive both members from `active`/`parked`/`quiescing` under local and remote
  winners with a narrowing row per arm.

### P2 — evidence

- **E3 (P2, `repeat-of: rev2 E2` class — the per-effect recheck not closed).
  The per-effect authorization recheck is pinned on the lease-tuple axis only;
  the EXPIRY axis is unmeasured at every position.** Reviewer mutant `RV3-M1`
  (`probes/review3_mutants.py`, `logs/50-rv3-mutants-pass{1..4}.log`,
  raw logs and diffs in `logs/rv3-mutants-pass{1..4}/`): `runEffects` reads
  `entryNow := engine.Now()` once before the loop and passes it to every
  in-loop `CheckAuthorization` (the lease tuple is still re-read per effect)
  — **SURVIVED ×4** against the committed suite, while the witness
  `TestRV3W_M1_AuthExpiryRecheckedBeforeEachEffect` passes pristine and fails
  under the mutant ×4: a request-stop whose authorization expires after the
  first effect (deadline 2026-09-03 > expiry 2026-09-02, `Now` advanced past
  the expiry after effect 1) commits `process_closed` and `backend_store_closed`
  under an expired authorization and reports `stopped`/`replay_same`;
  pristine refuses `terminal_backend_unauthorized` / `ax authorization expiry`
  before the second effect with `after=unavailable`, `new_authorization`,
  exactly one committed effect. Root cause in the fixture: `fixtureExpires`
  (2026-09-02T00:00Z) lies AFTER `fixtureDeadline` (2026-09-01T18:00Z), so no
  committed test can observe the expiry arm independently of the deadline arm
  at any position (the same confound makes the E1 `expired` subtest kill on
  the code literal rather than on the receipt: under a create-exempting mutant
  the deadline gate refuses `terminal_backend_timeout` and binds nothing). The
  shipped rows `N-engine-recheck-auth{,-third}` weaken the whole
  `CheckAuthorization` call and are killed by the tuple rotation; a mutant
  that keeps the tuple and drops the expiry walks through. DoD: "negative
  tests that fail when the gate admits what it must reject" is not met for
  the expiry member of the per-effect gate ("an expiring AX authorization";
  "rechecked immediately before each side effect"). The witness above is
  committable as is; add a harness row for it.

### P3

- **P3-1 Evidence bound edge unwitnessed.** `RV3-M2` (`checkSortedUniqueEvidenceWhere`
  `> 256` → `> 257`) SURVIVED ×4; witness `TestRV3W_M2_EvidenceBoundEdge`
  (256 admitted / 257 refused on `CheckResult` — `result evidence bound` — and
  on `ExecuteStatus` — `status evidence bound`) fails under it ×4. No committed
  test carries more than a handful of evidence IDs. Adjacent-edge class.
- **P3-2 The capability-conditional refusal arm's disposition is unpinned at
  the engine.** `RV3-M3` (`execute.go:257–260`: `DispositionFor(code,false)` →
  `DispositionStatusFirst`) SURVIVED ×4; `TestRV3W_M3_ConditionalCapabilityDisposition`
  (headless create from `stopped`, wait without `safe_boundary_observation`,
  provider kind without `provider_process_observation` → literal
  `required_operator_action`) fails under it ×4. `TestDispositionForMapping`
  pins the helper; the committed engine tests for this arm assert code, detail
  and `after` but never the disposition (`create_gates_test.go:92–93`,
  `lifecycle_test.go:45,73`). Rule pinned at the helper, not at this entry.
- **P3-3 Status report `last_operation_id` grammar unwitnessed.** `RV3-M8`
  (`execute.go:540`: admit exactly the empty string) SURVIVED ×4;
  `TestRV3W_M8_StatusLastOperationGrammar` (`""`, `nope`, upper-case UUID →
  `status report operation`) fails under it ×4. `TestExecuteStatusReportMemberRefusals`
  drifts session/backend/versions/instance/generation but never the last
  operation ID; the arm exists (`execute.go:540–543`) with no negative.
- **P3-4 `ParseProviderProofKind` admits a landed sibling name under a
  narrowing (same class as rev2 P3-4).** `RV3-M4` (default arm admits the
  capability name `provider_process_observation` as `provider_quiescence`)
  SURVIVED ×4; `TestRV3W_M4_ProofKindSiblingNamesRefused` (the two capability
  names, the side-effect name `safe_boundary_observed`, `provider_exit`,
  upper-case and trailing-space spellings, at the direct entry and through
  the wait body) fails under it ×4. The AST census counts case clauses only;
  the committed corpus (`provider_restart`, …) has no landed sibling names.
  Extend the corpus (and the `RetryDisposition`/`AuthorizationKind` corpora
  with their sibling vocabularies the same way).
- **P3-5 The partial-progress member of the create recovery is unstated.**
  `TestRV3_PartialCreateProgressIsNotResumable` and my real kill at the
  `AfterEffect` seam of `binding_persisted` (`TestRV3_RealKillAfterFirstCreateEffect`,
  ×2): receipt durable, one effect committed, no completion; a backend that
  reports the interim `creating` with identity match → the same-key retry
  refuses `terminal_backend_unavailable` / `unavailable` / `status_first`
  and the key stays parked (the only exit is `unavailable → terminate-stale`,
  final leaf); a backend that reports the instance `absent` (binding without
  wrapper) → the retry resumes and re-performs BOTH effects under the same
  receipt, one completion, one receipt. Both are defensible under "Only a
  successful status read proves absence" and the backend's key binding, but
  TRACEABILITY's bound names only "the proven TARGET (or closed)"; name the
  interim-proven member and its exit route.

### Characterizations (not findings)

- After a successful status read proves the TARGET (`quiescing`), the same-key
  quiesce retry refuses `terminal_backend_unavailable` with
  `after=unavailable` / `status_first` (`TestRV3_TargetProvenRetryProjectsUnavailable`):
  the local observation regresses from the just-proven target to
  `unavailable`. §4.C's "an error after an effect whose result cannot be
  proven moves the local observation to unavailable with status_first"
  supports the mapping (the lost result's evidence is unprovable); recorded
  because the caller has just been told `quiescing` by the same engine.
- Wait-safe-boundary resumption re-performs `safe_boundary_observed` and
  records the proof the backend returns now (`TestRV3_WaitResumptionReissuesBoundary`);
  "identical retry returns the same proof" is delegated to the backend's key
  binding — stated as a bound in TRACEABILITY, accepted.
- `RV3-M6` (remote path fences any source) and `RV3-M7` (in-loop deadline
  skipped once committed) are KILLED ×4 by the committed suite: the live-source
  gate and "a deadline cancels waiting, not a committed effect" are pinned.

## 4. DoD walk (live merged checklist)

Production entry points implement the scoped deliverable — the rows, gates,
receipts, rechecks, resumption and remote-winner fencing exist; the grant-less
members of the stale class are not fenced (F3). Positive/negative/recovery
tests pass with logs — yes for what is committed. README/traceability without
unsupported claims — README's "including under a remote winner" and
TRACEABILITY rows 47/77 outrun the code for the grant-less members (F3).
Uncommitted in the Story worktree — yes (tree OID equal). Ratio — 84 of 84
with call sites. Negative tests that fail when the gate admits — not for the
expiry member of the per-effect recheck (E3). Narrowing mutant per gate —
shipped 99 KILLED ×2 with a working control; five reviewer narrowings SURVIVED
×4 with non-equivalent witnesses (E3, P3-1..P3-4). No source-text gate in
production (the AST census is a test instrument; its blind spot is P3-4).
Lint/build — clean. Outcome artifacts — attached and hygienic. Logbook —
newest-first, additive, with the explicit reversal. Implementation matches AC —
not for the cold-takeover fencing members (F3). Fits the architecture — yes
(composition over the landed owners; the fix for F3 belongs in
`internal/fencing` as an exported verdict, not in a local comparison). Tests
green — yes. Gates attacked, not read — yes, and the attack found the gaps
above.

## 5. Instruments (all in the evidence archive)

`probes/zz_review3_probe_test.go` (16 top-level probes: 5 witnesses `TestRV3W_M{1,2,3,4,8}_*`,
5 CHARACTERIZATION probes, 6 direct re-verifications; all 16 PASS pristine,
`logs/12`, `logs/13`), `probes/zz_review3_crash_unix_test.go` (2 real-kill
probes, PASS ×2 each, `logs/60`), `probes/review3_mutants.py` (7 narrowing
rows + 3 controls; committed-suite and witness runs; 4 passes with identical
verdicts: M1/M2/M3/M4/M8 SURVIVED with witness KILLED, M6/M7 KILLED, `RV3-K`
known-kill KILLED, `RV3-C` comment control SURVIVED, `RV3-B` build-break
control reported `ERROR(build broke)`; raw per-plant logs with exits and
diffs in `logs/rv3-mutants-pass{1..4}/`; the working tree of `cand-m` differs
from the candidate index by nothing after every pass), `probes/run_harness.sh`
+ `probes/split_raw.py` (shipped harness driver: `logs/40-*`,
`logs/mutants-pass{1,2}/` 100 raw logs each), `logs/01`, `02`, `04`, `70`,
`71`, `80`, `81`, `82`, `REVIEW-MANIFEST.txt` (sha256 of every file).

## Rework scope (for the producer)

1. **F3 — fence the grant-less members of the stale class** (P2,
   `repeat-of: rev2 F2` class). Land an exported staleness verdict in
   `internal/fencing` that decides the direction/tuple arms over a verified,
   well-formed winner WITHOUT the grant precondition (the grant authorizes; it
   does not make an incarnation less stale), compose it for `ObserveFencing`'s
   second question (or for both questions), and drive the no-grant and the
   lapsed-grant members from `active`/`parked`/`quiescing` under a local and a
   remote winner, keeping unverified/ambiguous/no-winner as no-transition.
   Ship a narrowing row per new arm and the named regression test the
   `repeat-of` rule asks for. If instead the product decision is that fencing
   observation is grant-gated, say so explicitly in `doc.go`, TRACEABILITY and
   the README sentence, and name the recovery route the final leaf owes for
   these members — silence is the finding.
2. **E3 — pin the expiry axis of the per-effect recheck** (P2 evidence,
   `repeat-of: rev2 E2` class): commit `TestRV3W_M1_AuthExpiryRecheckedBeforeEachEffect`
   or its equivalent (a context whose deadline lies after the authorization
   expiry; `Now` advanced past the expiry after effect 1; literal
   `ax authorization expiry`, `after=unavailable`, `new_authorization`, one
   committed effect) and a harness row that keeps the tuple recheck and drops
   the expiry recheck. Consider a second fixture instant between the deadline
   and the expiry so the E1 `expired` member also kills on the receipt.
3. P3-1..P3-4: commit the four witnesses (`TestRV3W_M2/M3/M4/M8_*` are
   committable as is) with harness rows; P3-5: state the interim-proven
   member of the create recovery and its exit route. Keep everything else as
   is — the rest of the revision holds and my instruments confirm it.

Verdict recorded by RUN-260918-f6b44c; no product edits, commits, checkpoints
or integration were performed; the isolated copies are deleted after
attachment. `commit_ack` is not supplied.
