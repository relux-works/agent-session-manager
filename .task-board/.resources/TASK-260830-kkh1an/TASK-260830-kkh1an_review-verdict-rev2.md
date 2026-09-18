# TASK-260830-kkh1an — independent review verdict, CR-TASK-260830-kkh1an-2 revision 2

**Verdict: CHANGES REQUESTED (routed `to-dev`).** No P1. Two P2 production
findings on the lifecycle contract itself (F1: an interrupted attempt poisons
its idempotency key forever, so the row recovery column "status, then same-key
retry" and create's "idempotent across controller crash" are unimplementable;
F2: `ObserveFencing` never fences a stale incarnation whose winner is on
another host — the force-takeover case §13.3 step 15 exists for), two P2
evidence findings (E1: the create row's authorization/generation/headless
gates have no negative test at the engine; E2: the per-effect recheck is pinned
at one position and the reported session is never compared in any status test)
and seven P3 items. Everything else the producer claims holds on the exact
tree: the shipped harness reproduces 70/70 twice, the ratio re-derives to
72 of 72, the real-kill seam holds, hygiene is clean.

Reviewer: RUN-260918-4f5cc1 (claude-opus-5 max). Reviewed bytes: base
`c61fc06ad8190a73c3005e137282829936dac970` (the predecessor checkpoint, = the
Story branch tip), candidate tree `b605cc51c9dee44bc90f90d950892a58b09571ca`
(recomputed from the live Story worktree with a temporary untracked-aware index:
equal), patch sha256
`bf88c269de2bece432ded49c931c2d763cd82788b2f9b6c1eecc12323fe2ae52` (equal to the
CR record and to `git diff --binary` of the two OIDs). Every probe ran in
isolated `git archive` copies of the exact tree under
`.temp/TASK-260830-kkh1an/review-rev2/` (`cand/` tree-exact checks, `cand-h/`
shipped harness, `cand-m/` reviewer mutants, `cand-p/` probes and witnesses;
`cand-h`/`cand-m` carry their own `.git` and their post-run `write-tree` equals
the candidate OID; `cand-p` is diffed against `cand/` and differs only by the
three probe files). The live Story worktree, index, branch and HEAD were never
touched (`git status` after the review: `M LOGBOOK.md`, `M README.md`,
`M internal/axpane/decide.go`, `?? internal/terminstance/` only).
`PYTHONDONTWRITEBYTECODE=1` everywhere; 0 `__pycache__` in any copy. Every
instrument, raw log and subprocess exit is in
`TASK-260830-kkh1an_review-evidence-rev2.tar.gz` (paths below are relative to
that archive).

## 0. Inherited advisories (predecessor P3-1..P3-7), graded

| Item | Producer disposition | Grade |
| --- | --- | --- |
| P3-1 journal-only restore closure path | Left to TASK-260830-2056mm, stated in TRACEABILITY ("recovery domain") and results | Acceptable: explicit, with a reason |
| P3-2 pure core not fail-closed on a missing closure | Bound stated: no I/O-free `Decide` caller in this leaf | Acceptable as a bound (verified: `grep -r "axpane\." internal/terminstance` = 0 production references) |
| P3-3 remote-owned newest absent locally | Bound stated: no checkpoint loader here | Acceptable as a bound |
| P3-4 `EmitParked` winner binding | Left to 2056mm, stated ("events domain") | Acceptable: explicit |
| P3-5 `LoadCheckpoint` absence-vs-failure | Left to 2056mm, stated ("recovery domain") | Acceptable: explicit |
| P3-6 reattach carries two instance IDs | Closed by a doc sentence on `Decision.Descriptor` (`internal/axpane/decide.go` +4/−1, comment-only) plus the consumer sentence in `terminstance/doc.go` | Closed (the reviewer offered "document or rebuild"; documentation chosen; no code path changed, axpane suite green under `-race`, `logs/70`) |
| P3-7 evidence archive hygiene | `TASK-260830-kkh1an_producer-evidence.tar.gz`: task-scoped README, 188-entry `MANIFEST.sha256`, 0 `__pycache__`/`.pyc`, no foreign task files, raw per-plant logs on BOTH passes (70+70) | Closed |

Nothing was silently dropped.

## 1. What holds (verified myself on the exact tree)

| Check | Result | Evidence |
| --- | --- | --- |
| gofmt / go vet / go build; GOOS=linux and GOOS=windows builds; GOOS=windows vet of the package | all exit 0 | `logs/02` |
| `go test ./internal/terminstance -count=1 -v` | exit 0, 168 `--- PASS`, 0 FAIL, 0 SKIP | `logs/01` |
| `go test ./... -count=1` (command 4 without `-v`) on the exact tree, rerun by me | exit 0, 40/40 packages `ok` | `logs/04` |
| `-race` on terminstance, axpane, terminalbackend, fencing; `-cover` terminstance 83.8% | all `ok` | `logs/70` |
| tracecheck (registry untouched: contracts=64, clauses 56/569 — trunk figures) / `cataloggen -adopted … -check` / `git diff --check` | exit 0 / 0 / 0 | `logs/70` |
| CR construction suite | runtime-recorded `required=30 green=30 failed=0 missing=0` on the exact tree (accepted from the attached `_rev2-validation.log`; the race gate and fuzz gates were not rerun by me) | resource |
| Shipped harness, isolated pristine copy, two full passes with raw per-plant output | 70/70 both passes (68 narrowing KILLED + 1 tightening KILLED + control SURVIVED), verdict lists identical, 70 raw logs per pass, tree OID equal to the candidate after each pass | `logs/40-harness-pass{1,2}.log`, `logs/mutants-pass{1,2}/` |
| Producer real-kill test `TestExecuteCrashChildSelfTerminates` ×2; my own real SIGKILL at the ENGINE `BeforeEffect` seam of `create` ×2 (receipt durable, no completion, no effect ran, the retry never invents absence, status reads absence) | PASS ×2; my seam holds ×2 for the durability half (the liveness half is F1) | `logs/70`, `logs/60` |
| Ratio | 72 data rows counted in `TRACEABILITY.md`; 70 distinct top-level tests all present in `go test -list` and all `--- PASS` in my verbose run; 7 cited subtests all PASS; rows 53/54 are cross-references to rows 26/36/39/43/49/50 | `logs/82` |
| Spec fidelity (§4.C extracted and diffed): eight-state enum, ten-operation enum, side-effect enum, transition matrix, allowed-error sets, key shapes all landed (`terminalbackend`) and composed, never re-spelled; `AXAuthorization` closed 7-member set with unknown/missing/frame refusals; epoch bound witnessed at 1, 9007199254740991, 0, 9007199254740992 at `ParseAXAuthorization` (committed) and through the nested `MutationContext` surface (my `TestRV2_EpochBoundThroughNestedContext`); `RetryDisposition` closed 4 with a copy-counting AST census plus the behavioural parse pin; `ProviderProofKind` closed 3; `AuthorizationKind` closed 4; exact key material literals asserted (`/quiesce/`, `/boundary/…/<kind>`, `/stop/`, `session/bootstrap`); refusal sets through the landed `CheckErrorAllowed` with in-set and outside-set codes per row; status timeout/read failure → error with no report, never absent; every engine result path carries exactly one of the four literal dispositions (my `TestRV2_EveryResultCarriesExactlyOneLiteralDisposition`) | holds | `logs/12` |
| Generation bound: 0/256/257 (and 256/257 multibyte, chars not bytes) refused/admitted with the landed `terminal_backend_stale_generation` class on the context document, the status body, the Go-level twin and the landed `GenerationDigest`; Binding object and CLI Result 4 stated as bounds (no landed surface — verified: `canonicaljson` rejects the binding URN, `internal/cliresult` carries no raw generation member) | holds | `logs/12` (`TestRV2_DescriptorGenerationBoundLanded`) |
| Composition: `fencing.Authorize`/`ParkDetails`, `terminalbackend.CheckTransition/CheckErrorAllowed/CheckOperation/CheckStatusResult/IdempotencyKey/GenerationDigest/CheckVersionTuple/ParseID`, `environ.DecodeStrictObject/Check*`, `scalar.Parse*` — no second enum, no second fencing comparison, no second JSON discipline. `kindFor` restates the landed (unexported) `Transition.Authorization` column (P3-6 below). | holds | source read |
| Bounds and hygiene: changed paths = exactly the 31 candidate paths; `internal/traceability` untouched; `task-board.config.json` equal to base; README +47/−0 with test and harness commands and the explicit "no `ax` command, no `doctor` result, no runtime capability claim"; LOGBOOK +12/−0 newest-first; 0 `__pycache__`/`.pyc` in the tree; `os/exec`/`syscall`/`tmux`/ConPTY appear in production only inside the two stated-bound comments | holds | `logs/80` |

## 2. Coverage statement

Producer claim: 72 of 72 rows driven. Re-derived: 72 data rows, every named
test and subtest exists and passed (`logs/82`). **I measure 72 of 72 driven**
at the named production entries. Two rows' prose outruns what the tests show:
row 71 ("… status recovery") and the results' "a lost result recovers through
`status`" — status recovers the *observation*; the *operation* never recovers
(F1). Row 46 ("stale_fenced only via fencing observation") is driven on the
local-winner arms only; the remote-winner member of the class is not fenced
(F2). Rows 43/48/49/69 are driven on one operation or one effect position; the
class is not closed (E1, E2).

## 3. Findings

No P1: the eight-state enum, the operation/effect vocabularies and the
transition matrix are the landed ones; no second state machine exists.

### P2 — production

- **F1 (P2). An interrupted attempt poisons its idempotency key forever; the
  row recovery column cannot be executed.** `ExecuteMutating` (`execute.go:275-284`)
  treats a bound receipt without a completion as `terminal_backend_unavailable`
  / `unavailable` / `status_first` on *every* subsequent same-key request, and
  no production entry (`Engine`, `ReceiptStore`) can move a key out of that
  state. §4.C request-stop row: "lost result requires `status`, **then same-key
  retry only if not closed**"; create row: "identical retry replays it;
  uncertainty requires `status`"; closing paragraph: "Create is idempotent on
  (session_id, bootstrap_operation_id) **across controller crash** and lost
  result … Identical retry returns the one result/instance"; §13.13
  `safe_retry`: "the same logical operation can continue … using every
  caller-stable operation ID … A retry MUST reconcile an uncertain external
  effect before issuing it again and MUST NOT allocate another process". Driven
  witnesses (`probes/zz_review2_probe_test.go`, `probes/zz_review2_crash_unix_test.go`,
  `logs/12`, `logs/60`): (a) quiesce-input receipt without completion →
  retry refuses → `ExecuteStatus` proves `active` (nothing happened) →
  same-key retry **still refuses** `terminal_backend_unavailable at idempotency
  result uncertain`; (b) request-stop receipt without completion → status proves
  `quiescing` (not closed) → same-key retry still refuses; (c) **real
  SIGKILL** of the engine at the `BeforeEffect` seam of `create` (child killed
  by SIGKILL, receipt durable, no completion, no effect ran) → status proves
  `absent` → the identical create retry still refuses ×2 — the
  `(session_id, bootstrap_operation_id)` key can never produce "the one
  result/instance"; (d) the `StoreHooks.AfterCommit` error path: `store.go`
  documents "a hook error propagates to the caller but the receipt stands, so
  the identical retry replays it" — through the engine the identical retry
  refuses uncertain (`TestRV2_HookErrorThenIdenticalRetryDoesNotReplayThroughEngine`).
  The only exit a caller has is a *new* key (new `quiescence_generation` /
  new `bootstrap_operation_id`), which is exactly the second-allocation §13.13
  forbids when the interrupted attempt may have started a wrapper. The
  durability half (receipt before first effect, never inventing absence) is
  correct and pinned; the liveness half is absent and undeclared (no stated
  bound names it; TRACEABILITY rows 70/71 and LOGBOOK F2 describe the refusal
  as "the honest interrupted state" without the resumption).
- **F2 (P2). `ObserveFencing` does not fence a stale incarnation whose winner
  is on another host.** `fencing.go:41-48` keys the transition on
  `ParkStaleOwner` only, but the landed `fencing.Authorize` (gate.go, restore
  is launch-class) evaluates the remote-owner arm *before* the epoch/lease
  arms ("the operative fact beats the credential: a stale token under a remote
  winner parks remote, it does not report stale"). So a local epoch-1
  incarnation observed under an epoch-2 winner held by another host — the
  force takeover §13.3 step 15 describes ("its winning lease fences the prior
  owner logically") and §13.4 ("If a higher or winning same-epoch lease is
  observed, the wrapper MUST block input, park or terminate its stale
  provider … and return `stale_owner`") — returns `(current, false,
  park remote_owner)` from `active`, `parked` and `quiescing`, and the
  same-epoch losing lease under a remote holder does the same
  (`TestRV2_StaleTokenUnderRemoteWinnerIsNotFenced`,
  `TestRV2_SameEpochLosingLeaseUnderRemoteWinnerIsNotFenced`, `logs/12`). The
  local-winner arms fence (control `TestRV2_StaleTokenUnderLocalWinnerFences`
  passes; the committed `TestObserveFencingStaleEpochFences` /
  `…LeaseMismatchFences` use a local holder). Consequences: the prior owner's
  Terminal Instance never reaches `stale_fenced` after a cross-host takeover,
  so the `terminate-stale` row (`stale_fenced|unavailable → stopped`, final
  leaf) cannot target it from that state, and the committed
  `TestObserveFencingNonStaleLeavesState/remote` pins the *winning* token under
  a foreign local host — a different member (not stale) — so the suite reads
  "remote leaves state" as if it covered this case. Input stays blocked by the
  landed input gate (`not_owner`), so this is a wrong projection, not an
  admitted mutation. The fix must not compare tuples locally (a second fencing
  model): either derive staleness from the landed gate with the observation's
  own host as the local host (authorize against `LocalHostID = Winner.HolderHostID`
  to ask "is this token stale relative to the winner", then read the park), or
  land an exported "is this token stale" verdict in `internal/fencing` first.

### P2 — evidence

- **E1 (P2). The create row's gates have positive-path-only evidence at the
  engine.** Whole-suite reviewer mutants (`probes/review2_mutants.py`,
  `logs/50-rv2-mutants-pass{1,2}.log`, raw per-plant logs in
  `logs/rv2-mutants-pass{1,2}/`, two passes with identical verdicts; instrument
  proven by the known-killed control `RV2-K` KILLED ×2, the build-break control
  reporting `build broke` ×2 and the comment control SURVIVED ×2):
  `RV2-M8` (entry authorization gate exempts exactly `create`) **SURVIVED ×2**,
  `RV2-M7` (entry generation gate exempts `create`) **SURVIVED ×2**,
  `RV2-M10` (headless_creation conditional skipped for a headless create from
  `absent`) **SURVIVED ×2**, `RV2-M5` (disposition mapping collapsed at the
  engine for a committed `terminal_backend_timeout`, reachable only through
  create's second effect) **SURVIVED ×2**. Witnesses (`logs/rv2-witness-under-mutant/`,
  each PASS pristine / FAIL under its mutant): under M8 a create presented
  with a control-kind, an expired or a foreign-lease authorization **binds a
  durable receipt** before the pre-first-effect recheck refuses it — which, by
  F1, poisons the `(session_id, bootstrap_operation_id)` key; under M7 the same
  for a stale generation; under M10 a headless create from `absent` proceeds
  without `headless_creation`; under M5 create's second-effect timeout reports
  `replay_same` with `after=unavailable`. The committed pins
  (`TestExecutePreEffectErrorsRestoreSource`, harness rows
  `N-engine-auth-entry`/`N-engine-generation-entry`, both keyed on the quiesce
  fixture's operation ID) cover the quiesce row only; `TestExecuteCreateReceiptRule`
  drives the headless conditional from `stopped` only. DoD: "Gating …
  authorizing … behavior covered by negative tests that fail when the gate
  admits what it must reject" is not met for the create row.
- **E2 (P2). The per-effect recheck is pinned at one position, and the
  reported session is never compared.** `RV2-M1` (auth recheck skipped exactly
  before the THIRD effect) and `RV2-M2` (generation recheck skipped before the
  third effect) **SURVIVED ×2**: under a lease rotated at the fourth
  `CurrentLease` read, `request-stop` commits `backend_store_closed`
  unauthorized and reports `stopped`/`replay_same` (witness
  `TestRV2W_M1_LeaseRotatesBeforeThirdEffect`); the committed
  `TestExecuteRechecksBeforeEachEffect` rotates at the third read only (the
  recheck before effect 2). "Rechecked immediately before **each** side
  effect" is a headline AC item (3). `RV2-M15` (session-scoped status lookup
  skips the reported-session comparison) **SURVIVED ×2**: a backend reporting
  ANOTHER session's instance with `identity_match:true` is adopted as this
  session's status (`TestRV2W_M15_SessionScopedForeignSessionRefuses`, pristine
  refuses `status identity binding`); no committed test drifts the reported
  session on either lookup path (`TestExecuteStatusReportMemberRefusals/session`
  uses a malformed value, refused at the grammar).

### P3

- **P3-1 Pre-link receipt-store failure moves the observation to
  `unavailable`.** `execute.go:266-274`: any uncoded `Bind` error → `after =
  unavailable`, `terminal_backend_process_failed`, `status_first`. With the
  pending directory read-only (stage fails before any link, nothing durable,
  nothing attempted) the result is `unavailable` (`TestRV2_PreLinkStoreFailureMovesToUnavailable`,
  `logs/12`). §4.C: "An error before a side effect restores the source state."
  Over-strict direction; the receipt is not a side effect. Distinguish the
  pre-link failures (restore source) from the post-link ones (hook error,
  directory sync — genuinely uncertain).
- **P3-2 `ObserveFencing` fences from `absent`, `stopped`, `creating` and
  `unavailable`** (`TestRV2_ObserveFencingFencesFromAbsentAndStopped`): an
  incarnation that does not exist becomes `stale_fenced`, and a
  `terminate-stale` (final leaf) would then target nothing. Bound the sources
  (the live states) or state the bound.
- **P3-3 Deadline instant admitted by no test.** `RV2-M3` (entry deadline
  `!Before` → `After`, admits `now == deadline_at`) and `RV2-M4` (same at
  `ExecuteStatus`) SURVIVED ×2; under M3 a receipt is bound for a request whose
  deadline has arrived, under M4 status proceeds at the instant. The committed
  deadline tests use an hour past the deadline. Add the instant to
  `TestExecuteEntryDeadlineCodes` and a status-deadline test.
- **P3-4 `ParseAuthorizationKind` admits the landed sibling name `attach`
  under a narrowing (`RV2-M6` SURVIVED ×2).** The committed non-members are
  `admin`, `CREATE`, `""`, `force-stale`; the AST census inspects case clauses
  only, so an `if value == "attach"` in the default arm is invisible to it.
  Add `attach` and `none` (the two landed transition-table kinds that are not
  AXAuthorization kinds) to the negative corpus.
- **P3-5 identity_match six-way conjunction pinned on the generation axis
  only.** `RV2-M9` (drop the implementation conjunct) SURVIVED ×2; a lying
  match with only the implementation version drifted is adopted. My
  `TestRV2_StatusIdentityMatchEachAxis` drives all five axes both ways on the
  pristine tree (all pass): commit it or its equivalent.
- **P3-6 `kindFor` restates the landed `Transition.Authorization` column.**
  The landed table is unexported, so the sibling package re-spells
  create→create, quiesce/wait/stop→control. Pinned by `N-engine-kind-map`, but a
  drift between the two tables is invisible. An exported accessor in
  `terminalbackend` (or extending that package) removes the twin.
- **P3-7 `unavailable` paired with `new_authorization`.** A lease rotation
  after a committed effect returns `after=unavailable` with
  `new_authorization` (`TestExecuteRechecksBeforeEachEffect/lease_rotates_mid-operation`,
  pinned; harness `N-disp-unauthorized`). §4.C's only sentence about
  `unavailable` pairs it with `status_first`; the leaf's per-code mapping is
  its own contract (stated), so this is recorded as a literal tension, not a
  defect: the caller that lost the lease cannot act on `new_authorization`
  without a status read either.

### Characterizations (not findings)

- The replay path is reachable only from the original source state: an
  identical retry presented from the post-transition state refuses
  `local_precondition_failed` at the landed `CheckTransition` before the
  receipt is read (`TestRV2_ReplayRequiresOriginalSource`). Consistent with the
  lost-result model (the caller never observed the transition).
- `holder_host_id` is parsed (UUIDv7) and never bound: `LeaseView` carries
  `(lease_id, epoch)` only, which is the §5.3 winning tuple; a foreign holder
  under the winning tuple is admitted (`TestRV2_HolderHostIDIsUnboundAtCheckAuthorization`).
  Not a spec violation; worth a stated bound.
- `N-auth-epoch-high` is a tightening row, documented as such; the min edge,
  the delegate's own uint53 ceiling (`environ`) and the four-value witness
  triangulate the bound. Accepted.

## 4. DoD walk (live merged checklist)

Production entry points implement the scoped deliverable — partially: the
rows, gates, receipts and rechecks exist; the recovery column (F1) and the
remote-winner fencing member (F2) do not. Positive/negative/compatibility/
recovery tests pass with logs — yes for what is committed. README/traceability
without unsupported claims — README yes; TRACEABILITY/results "recovers
through status" outruns the behaviour (F1). Uncommitted in the Story worktree
— yes (tree OID equal). Ratio — 72 of 72 with call sites. Negative tests that
fail when the gate admits — not for the create row (E1). Narrowing mutant per
gate — shipped 69 KILLED ×2; eleven reviewer narrowings SURVIVED ×2 with
non-equivalent witnesses (E1, E2, P3-3/4/5). No source-text gate in production
(the AST census is a test instrument, not a gate; its blind spot is P3-4).
Lint/build — clean. Outcome artifacts — attached and hygienic. Logbook —
newest-first, additive. Implementation matches AC — no (F1, F2). Fits the
architecture — yes (composition over the landed owners; P3-6). Tests green —
yes. Gates attacked, not read — yes, and the attack found the gaps above.

## 5. Instruments (all in the evidence archive)

`probes/zz_review2_probe_test.go` (15 probes: 6 CHARACTERIZATION failures =
F1 ×3, F2 ×2, P3-1 ×1; the other 9 PASS) plus `TestRV2_RealKillAtFirstEffectSeam`
in `probes/zz_review2_crash_unix_test.go` (F1 ×1 with a real SIGKILL; 26
top-level RV2 tests in all: 19 PASS, 7 CHARACTERIZATION FAIL, `logs/12`), `probes/zz_review2_witness_test.go` (10 delta witnesses, all PASS
pristine, each FAIL under its mutant), `probes/review2_mutants.py` (11
narrowing rows + 3 controls, whole-suite killer, raw per-plant logs with
exits), `probes/review2_witness_runner.py`, `logs/01`, `02`, `04`, `10`–`12`
(probes), `40` + `mutants-pass{1,2}/` (shipped harness ×2), `50` +
`rv2-mutants-pass{1,2}/` (reviewer mutants ×2), `51` + `rv2-witness-under-mutant/`
(witness under each mutant), `60` (real kill ×2), `70` (tree-exact checks),
`80` (hygiene), `82` (ratio), `REVIEW-MANIFEST.txt` (sha256 of every file).

## Rework scope (for the producer)

1. **F1 — make the same-key retry after `status` executable.** When
   `Bind` replays a receipt that has no completion, do not refuse
   unconditionally: reconcile through the production status entry (or accept a
   fresh `StatusReport` the caller obtained) and, exactly as the row columns
   say, replay the recorded result when the target state is proven, continue
   the same operation from the proven source state when nothing happened
   (create: absence proven by a successful status read → perform the effects
   under the same receipt and complete it), and keep refusing `status_first`
   only while the observation is genuinely unknown. Keep the SIGKILL evidence
   and add the resumption to it: crash at the receipt seam → status → same-key
   retry completes with one result/instance and no second receipt; a same-key
   retry WITHOUT a status read still refuses. Fix the `StoreHooks` doc sentence
   or make it true at the engine. Add a narrowing row per new arm.
2. **F2 — fence the stale incarnation under a remote winner** without a local
   tuple comparison (compose the landed gate; do not fork it). Drive both
   remote members (older epoch, same-epoch losing lease) from `active`,
   `parked` and `quiescing`; keep the winning-token-under-foreign-host case as
   "no transition". Add narrowing rows.
3. **E1 — create-row negatives at the engine:** control/expired/foreign-lease
   authorization, stale generation, headless-from-`absent` without
   `headless_creation`, and create's second-effect committed timeout —
   asserting no receipt is bound in the pre-effect cases. Re-key the harness
   rows `N-engine-auth-entry` / `N-engine-generation-entry` so that a mutant
   exempting `create` is killed.
4. **E2 — recheck at every position and the session axis:** rotate the lease
   and the generation before the third stop effect; drift the reported session
   on both status lookup paths (and each of the other five axes, P3-5).
5. P3-1 (pre-link store failure restores the source), P3-2 (fencing sources or
   a stated bound), P3-3 (deadline instant on both entries), P3-4 (`attach`,
   `none` in the kind negatives), P3-6/P3-7 as you see fit; document
   `holder_host_id` as unbound.
6. Reword TRACEABILITY rows 70/71 and the results/LOGBOOK "recovers through
   status" once (1) is in; keep everything else as is — the rest of the leaf
   holds.

Verdict recorded by RUN-260918-4f5cc1; no product edits, commits, checkpoints
or integration were performed; the isolated copies are deleted after
attachment. `commit_ack` is not supplied.
