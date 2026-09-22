# BUG-260917-2fwf8e — review verdict, Change Request revision 1

- Reviewer run: RUN-260921-766ad0 (claude-opus-5 max), role `reviewer`, read-only.
- Change Request: `CR-BUG-260917-2fwf8e-1` rev 1; base `799c338e401fca0b24859c0b870cd665927209f2`
  (= `origin/main` at review time, 0 commits of drift); candidate tree
  `e1cf09a76f7e713171c366406b337064454d2907`; live Story worktree blobs equal the
  candidate tree for all 9 changed paths (log `00-tree-identity.log`).
- Normative authority: `internal/specdoc/SPEC.v0.7.0.md` §4.2 after-restore steps 1–5
  (lines 1433–1441), §13.4 local attach (lines 9462–9473), §13.7 force takeover
  (lines 9613–9781), §4.C (line 1115: only AX fencing observation enters `stale_fenced`).
- All instruments ran on immutable `git archive` copies of the two trees under
  `.temp/BUG-260917-2fwf8e/review-rev1/`; the Story worktree, index, branch and HEAD
  were not mutated. Host load during the review: 9–17 (a second Story's suite ran
  concurrently); wall-clock is recorded per gate below.

## Verdict: CHANGES REQUESTED → `to-dev`

The fencing reorder itself is spec-grounded and correct at the reported vector
(probe 10 reproduced: trunk `refused/lease_conflict`, candidate
`attach_remote`/`takeover_offer`/`parked(remote_owner)`, through `Authorize`,
`Decide` and the real `Run` entry with a repository-recorded remote takeover).
It is not acceptable as shipped because its composition into the landed
Terminal-Instance lifecycle regresses a normative fencing property, and the
landed regression test that protected that property was rewritten to expect the
regression. Two coverage findings and three hygiene findings follow.

Measured AC coverage against the six task-specific DoD rows: **4 of 6 driven**
(rows 1–4); row 5 (per-test diff names every moved input) and row 6 (scope held
without silently absorbing a landed-behaviour change) are not satisfied. Row 1 is
driven on 1 of 5 gate entries (see P2-1).

## P1-1 — a stale incarnation under a remote winner no longer fences when its grant has lapsed (terminstance.ObserveFencing), and the landed regression test was rewritten to bless it

**What changed.** `terminstance.ObserveFencing` asks `fencing.Authorize` twice for a
remote winner: the direct question, and the "relative" question from the winner's
own host (`observeRemoteWinner`, `internal/terminstance/fencing.go:100-112`). With
the candidate order the relative question now dies on the deferred lapsed-grant
refusal before the tuple arms (`lease_conflict`, log `10-fencing-grid-candidate.log`
rows `token=stale|loser ... relative(from winner host)=REFUSE(lease_conflict)`), so
`observeRemoteWinner` returns "leave state" and the `StaleRelativeToWinner` fallback
that the local-winner path uses is never consulted.

**Measured effect** (`11-terminstance-grid-{trunk,candidate}.log`, 60 rows each:
{remote, local winner} × {stale-epoch, losing-lease token} × {live, lapsed, none,
no-clock, unusable-policy grant} × {active, parked, quiescing}):

| ObserveFencing row (remote winner, stale-epoch or losing-lease token) | trunk 799c338 | candidate e1cf09a7 |
| --- | --- | --- |
| live grant × 3 states | `stale_fenced`, transitioned | `stale_fenced`, transitioned |
| **lapsed grant × 3 states × 2 tokens (6 rows)** | `stale_fenced`, transitioned | **stays `active`/`parked`/`quiescing`, transitioned=false, err = `parked: park (remote_owner) … not_owner`** |
| no grant / no clock / unusable policy | `stale_fenced`, transitioned | `stale_fenced`, transitioned |

The lapsed grant is now the ONLY grant shape under which a stale incarnation is
not fenced — exactly the shape a partitioned prior owner has after a force
takeover (it could not refresh, its grant lapsed, a remote host force-took-over,
and on union it observes the remote winner). SPEC §13.7: "its fencing token
rejects the prior owner … the old process MAY continue until it observes that
losing lease and parks or stops", "The prior owner becomes stale", "The loser
MUST stop accepting input when it learns the winner"; §13.4: "If a higher or
winning same-epoch lease is observed, the wrapper MUST block input, park or
terminate its stale provider"; §4.C: only AX fencing observation enters
`stale_fenced`. The candidate's `fencing.StaleRelativeToWinner` doc comment states
the same property ("a losing tuple under a remote winner is stale relative to the
winner on any host").

**Named reddening evidence.**

- `13-property-stale-remote-lapsed-{trunk,candidate}.log`:
  `TestReviewStaleUnderRemoteWinnerLapsedGrantMustFence` (the landed
  `TestRV3F3_GrantLessStaleFences` ObserveFencing assertion with the old-order
  precondition removed, so it measures the property and not the arm order) —
  PASS 6/6 on trunk, **FAIL 6/6 on the candidate** (`ObserveFencing() error = parked:
  park (remote_owner) … not_owner, want the fencing transition`).
- `12-cross-candidate-prod-trunk-tests.log`: trunk's
  `internal/terminstance/fencing_grantless_test.go` against the candidate
  production — 8 subtests red: `TestRV3F3_GrantLessStaleFences/lapsed_grant/remote_winner/{stale_epoch,losing_lease}/{active,parked,quiescing}`
  (the regression) and `TestRV3F3_GrantLessDecidedNotStaleLeavesState/{winning_token,future_epoch}_remote`
  (intended moves: the winning/future token under a remote winner now surfaces
  the `remote_owner` park with no transition; those two rewrites are fine).

**The producer's argument, graded.** Results: "Updated old expiry-first
expectations to the correct remote-owner park/no transition"; matrix: "the
required terminstance test/comment updates that remove the old expiry-first
expectation". The trunk test's first assertion (direct question "never a park")
did encode the old order and legitimately needed updating for the lapsed/remote
rows. Its second assertion — ObserveFencing reaches literal `stale_fenced` — is
the rev2-F2/rev3-F3 property the test exists for (the comment says so), and it
was replaced by `transitioned=false`/`state==current` with no spec citation. The
old expectation was protecting a real property; the rewrite converts a landed
regression test into a witness of the regression.

**Scope note (DoD row 6).** The diff touches no lease-chain code and no park
vocabulary, and `internal/traceability` is untouched. But the composition into
`terminstance` changed a landed lifecycle outcome, and the DoD row's instruction
for that case is to stop and say so, not to absorb it by rewriting the test.

## P2-1 — direction-before-expiry is pinned on 1 of 5 gate entries; two reviewer narrowings survive every importer package (2/2 runs)

Every shipped witness drives `OperationRestore` only
(`authorize_order_test.go` all three tests; `decide_test.go` new test uses
`ModeRestore`; the `terminstance` rows use `OperationRestore`). The gate.go doc
comment, README paragraph and TRACEABILITY matrix state the order for `Authorize`
as a whole, and `axpane` reaches the offer through `AuthorizeActivation` on
`ModeLaunch` as well (my Run probe: `mode=launch interactive=true → attach_remote`,
log `15-axpane-run-probe10-candidate.log`).

Reviewer mutants (`review-mutants/run1` over fencing+terminstance+axpane,
`run2-importers` over `./internal/fencing/... ./internal/terminstance
./internal/axpane ./internal/termbind` — the complete importer set per
`30-importer-census.log`; one raw log per plant, real exits, mutation source
restored and hash-checked, `PYTHONDONTWRITEBYTECODE=1`, zero `__pycache__`):

| Row | Kind | Result (run1 / run2) | Killer(s) |
| --- | --- | --- | --- |
| RM-A restore-only-direction-first (activation/input/mutation/checkpoint revert to expiry-first) | narrowing | **SURVIVED / SURVIVED**, exit 0 | none; `31-rma-applied-grid.log` proves it applied: activation/input/mutation/checkpoint back to `lease_conflict`, restore still parks |
| RM-H non-launch remote+lapsed refuses `lease_conflict` instead of `not_owner` | narrowing | **SURVIVED / SURVIVED**, exit 0 | none |
| RM-B direction arm fires only under a live grant | narrowing | KILLED / KILLED, 15 fails | `TestAuthorizeRemoteInteractiveOwnerLapsedGrantUsesRemoteOfferArm`, `TestDecideLapsedRemoteOwner…`, `TestRV3F3_*` |
| RM-D policy-validity arm moved after direction | order | KILLED / KILLED, 8 fails | `TestAuthorizeRemoteOwnerInvalidPolicyRemainsInvalidArguments`, `TestRV3F3_GrantLessStaleFences` |
| RM-G lapsed remote parks `restore_policy` | narrowing | KILLED / KILLED, 15 fails | as RM-B |
| RM-C deferred lapsed refusal only for the exact winning tuple | narrowing | KILLED / KILLED, 15 fails | `TestRV3F3_*` only — no `internal/fencing` test pins "local lapsed + stale token → hard `lease_conflict`, not a `stale_owner` park" |
| RM-T remote second question dropped | arm-delete (composition control) | KILLED / KILLED, 2 fails | `TestObserveFencingRemoteWinner{StaleEpoch,LosingLease}Fences` |
| C-harmless-comment | control | SURVIVED / SURVIVED | — |
| C-not-applied | control | NOT_APPLIED / NOT_APPLIED | — |

## P2-2 — moved inputs are not named; the per-test diff cannot see them

Results: "No pre-existing per-test outcome changed or moved across a refusal
arm." The producer's diff is `internal/fencing` only and keyed by test name.
Measured at `Authorize` over all five entries (`10-fencing-grid-{trunk,candidate}.log`,
17 vectors × 5 entries), these classes moved:

| Vector (remote winner, lapsed grant, +…) | trunk | candidate |
| --- | --- | --- |
| V01 winning tuple (probe 10) | `lease_conflict` ×5 | activation/restore `park(remote_owner, not_owner)`; input/mutation/checkpoint `not_owner` |
| V07 stale-epoch token | `lease_conflict` ×5 | same as V01 |
| V08 same-epoch losing lease | `lease_conflict` ×5 | same as V01 |
| V09 future-epoch token | `lease_conflict` ×5 | same as V01 |
| V14 empty `LocalHostID` | `lease_conflict` ×5 | same as V01 |

Unchanged (both trees): V02 remote+live → `remote_owner`/`not_owner`; V03 local
lapsed → `lease_conflict`; V04 remote + no grant → `invalid_arguments` (step 4
still unreachable with no grant at all — a stated bound of this fix, not a
finding, since `Observe` gets the grant from the caller and the refusal names the
remedy); V05/V06 no clock / unusable policy → `invalid_arguments`; V10 local
lapsed + stale token → `lease_conflict`; V11–V13 unverified/ambiguous/handoff
parks → unchanged; V15/V16 expiry-edge → unchanged.

Name-keyed per-test diff of the three touched packages
(`20-pertest-{trunk,candidate}.tsv`, `20-pertest-diff.txt`): fencing 310→316,
axpane 195→199, terminstance 303→303, only additions — while 8 terminstance rows
changed meaning under unchanged names (P1-1). The results name V01 and the
terminstance rewrite; V07/V08/V09/V14 and the non-launch `lease_conflict → not_owner`
move (the `axpane.Emit`/`AuthorizeMutation` path; still a refusal, still covered by
`isFencingRefusal`, no behaviour change at `Run`) are unnamed.

## P3 — hygiene

- P3-1 `BUG-260917-2fwf8e_producer-evidence.tar.gz` embeds the entire foreign
  TASK-260830-1geqhj rev1 review archive (`probe10/evidence/…`: its mutants logs,
  probes, REVIEW-MANIFEST, verdict). Ship only the probe body/log this task used.
- P3-2 `TestAuthorizeArmOrderingNeighbourVectors/remote_noninteractive_owner` drives
  a LIVE grant and interactivity is not a fencing fact; the TRACEABILITY
  "Remote non-interactive owner" row cites it as the fencing witness. Rename
  (`remote_owner_live_grant`) and let the `axpane` `noninteractive_park` row carry
  the wrapper-context fact.
- P3-3 LOGBOOK (2026-09-21 entry, SCOPE bullet), TRACEABILITY prose and the
  terminstance doc comment present the no-transition outcome as correct; rewrite
  after P1-1.
- Measured `internal/fencing` coverage on the exact tree is 99.3% (cmd 06), not the
  97.9% stated; harmless, but report the measured figure.

## What holds

- Probe 10 reproduced by me on both trees (`14-axpane-probe10-*.log`,
  `15-axpane-run-probe10-*.log`, `10-fencing-grid-*.log` V01). The archived probe
  file as a whole does not compile on this tree (removed `Realm.AttestedServer` in
  other probes), but probe 10's body compiles and runs verbatim with the landed
  fixtures; the producer's "cannot compile unchanged" bound is over-stated but
  not wrong.
- Step-away vectors driven at `Authorize` (all five entries) and `Decide`
  (`14-axpane-probe10-*.log` rows A–J): live local grant + remote interactive →
  `attach_remote` on both trees; remote non-interactive + lapsed → `parked/remote_owner`
  (candidate) vs `refused/lease_conflict` (trunk); local owner + lapsed →
  `lease_conflict` both; remote interactive + no grant → `invalid_arguments` both;
  attach-not-admitted → `takeover_offer`; unverified → `restore_policy` park both.
- Through the real `Run` entry with a repository-recorded remote takeover: no
  parked event is authored under the remote lease (events 1→1) for every mode ×
  interactivity combination (`15-axpane-run-probe10-candidate.log`).
- Swap row `N-arm-order-expiry-before-direction`: APPLIED, a true arm swap (both
  gates present), KILLED in both harness passes with the named killer failing on
  the literal `lease_conflict` assertion (`harness/run{1,2}/N-arm-order-expiry-before-direction.log`).
  Shipped harness: 32/32 N+T rows KILLED, `C-harmless-comment` SURVIVED,
  `C-not-applied` NOT_APPLIED, `C-compile-failure` classified, both passes (34–36 s
  each); mutation source restored to the candidate blobs; zero `__pycache__`/`.pyc`.
- Scope at the diff level: no lease-chain (`sessrepo`, `sessstate` untouched), no
  park-vocabulary constant changed, `internal/traceability` untouched,
  `terminstance/fencing.go` comment-only; `origin/main == base`, no refresh
  residue. `gofmt -l` empty on the live worktree's tracked+untracked Go files;
  `git diff --check` clean; JSON parse clean; `task-board validate` exit 0 with its
  standing 180-issue ledger notice (`41-task-board-validate.log`).
- Full configured suite on the probe-free exact tree: see the table appended
  below (`suite/logs/suite-summary.log`, per-command logs and `.rc` files).

## Rework scope (for the producer)

1. Restore the §13.7/§13.4 property: a stale incarnation under a remote winner
   must reach `stale_fenced` for the lapsed-grant shape exactly as it does for
   live/no/no-clock/unusable-policy grants. Minimal repair inside the already-touched
   composition: in `terminstance.observeRemoteWinner`, when the relative question is
   not a `stale_owner` park, consult the landed `fencing.StaleRelativeToWinner`
   fallback (the same one the local-winner path uses) before returning "leave
   state" — it decides tuple staleness without the grant precondition and yields
   false for the winning and future-epoch tokens, so the candidate's two
   `GrantLessDecidedNotStaleLeavesState` rewrites stay valid. No lease-chain or
   park-vocabulary change is needed. If the Story owner rules the terminstance
   composition outside this leaf, stop the line per DoD row 6 instead of
   rewriting the landed test.
2. Restore the `stale_fenced` assertion for the six `lapsed grant/remote winner`
   rows of `TestRV3F3_GrantLessStaleFences`, keeping only the direct-question
   precondition update (`remote_owner` park). Add the reviewer property test
   (`probes/zz_review_terminstance_property_test.go`) or an equivalent named test
   that fails without the fallback, and a narrowing row for it.
3. Pin the order on all five entries: roster `fenceEntries()` in the
   remote+lapsed witness (activation/restore park `remote_owner`/`not_owner`;
   input/mutation/checkpoint literal `not_owner`), add a `ModeLaunch` row to the
   `Decide` test, and make the swap row's killer mask cover that test so RM-A and
   RM-H are KILLED; add an `internal/fencing` witness for "local lapsed + stale
   token → hard `lease_conflict`" (RM-C is currently killed only by terminstance).
4. Results: name every moved class (V01/V07/V08/V09/V14 × launch/non-launch) with
   before/after codes; extend the per-test diff to terminstance and axpane and
   add a cross-run (trunk test files against the candidate production) since a
   same-name rewrite is invisible to a name-keyed diff.
5. Hygiene: drop the foreign archive from the evidence tarball; rename the
   `remote_noninteractive_owner` fencing row; correct LOGBOOK/TRACEABILITY/doc
   comment wording and the coverage figure.

## Full configured suite on the exact tree (30 commands read from `task-board.config.json`)

Commands 2–27 ran from `suite/run-suite.sh` on a probe-free `git archive` of
`e1cf09a7` (`suite/logs/cmd-NN.log` + `.rc`); commands 1, 28, 29, 30 need git and
ran read-only against the live worktree (`logs/40-hygiene.log`,
`logs/41-task-board-validate.log`). Wall-clock is with the concurrent Story's suite
on the host (load 9–15).

| # | Command | rc | Elapsed | Note |
| --- | --- | ---: | ---: | --- |
| 1 | gofmt -l over tracked+untracked Go files | 0 | — | empty |
| 2 | go build ./... | 0 | 2s | |
| 3 | go vet ./... | 0 | 1s | |
| 4 | go test ./... -count=1 -v | 0 | 190s | 44 ok, 0 FAIL |
| 5 | go test ./... -race -count=1 -timeout 25m | 0 | 498s | 44 ok, 0 FAIL |
| 6 | go test ./... -cover -count=1 | 0 | 168s | fencing 99.3%, terminstance 86.6%, axpane 81.2%, termbind 83.7% |
| 7–23 | 17 fuzz lanes, 100x, parallel=1 | 0 each | 0–3s each | |
| 24 | tracecheck | 0 | 0s | `traceability ok: contracts=64 … clauses_discharged=70/574` |
| 25 | cataloggen -adopted -check | 0 | 1s | |
| 26 | GOOS=linux go build | 0 | 1s | |
| 27 | GOOS=windows go build | 0 | 1s | |
| 28 | JSON parse of tracked *.json | 0 | — | |
| 29 | task-board validate (authoritative board) | 0 | — | standing "180 issue(s)" ledger notice, 2714/2714 mirrored |
| 30 | git diff --check | 0 | — | |
| extra | GOOS=windows GOARCH=amd64 go vet ./... | 0 | 1s | |

Suite green on the exact tree; the P1 is not a suite failure — it is a landed
property the candidate's own suite no longer asserts. The suite going green
after the test rewrite is the incident, not the evidence.

## Evidence index (`BUG-260917-2fwf8e_review-evidence-rev1.tar.gz`)

- `logs/00-tree-identity.log` — live worktree vs candidate blobs, origin/main == base.
- `logs/10-fencing-grid-{trunk,candidate}.log` — 17 vectors × 5 entries at `Authorize`; relative-question rows.
- `logs/11-terminstance-grid-{trunk,candidate}.log` — 60 ObserveFencing rows each; 6 rows differ.
- `logs/12-cross-candidate-prod-trunk-tests.log` — trunk test files against candidate production (8 red).
- `logs/13-property-stale-remote-lapsed-{trunk,candidate}.log` — §13.7 property test PASS/FAIL.
- `logs/14-axpane-probe10-{trunk,candidate}.log` — probe 10 verbatim + Decide neighbour grid A–J.
- `logs/15-axpane-run-probe10-{trunk,candidate}.log` — probe 10 through `Run` with a recorded remote takeover.
- `logs/20-pertest-*.{jsonl,tsv}`, `20-pertest-diff.txt` — name-keyed per-test diff, three packages.
- `logs/30-importer-census.log`, `logs/31-rma-applied-grid.log`, `logs/40-hygiene.log`, `logs/41-task-board-validate.log`.
- `probes/` — the five reviewer probe/test sources (report-only grids + the property test).
- `harness/run{1,2}.stdout`, `harness/run{1,2}/*.log`, `mutants-full.json` — shipped harness, two passes.
- `review-mutants/review_mutants*.py`, `run1/`, `run2-importers/` — reviewer plants, raw logs, JSON.
- `suite/run-suite.sh`, `suite/logs/` — full configured suite logs and `.rc` files.
