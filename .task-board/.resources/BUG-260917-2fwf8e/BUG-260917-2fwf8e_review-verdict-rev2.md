# BUG-260917-2fwf8e — review verdict, Change Request revision 2

- Reviewer run: RUN-260921-442ca9 (claude-opus-5 max), role `reviewer`, read-only.
- Change Request: `CR-BUG-260917-2fwf8e-2` rev 2; base `799c338e401fca0b24859c0b870cd665927209f2`
  (= `origin/main` at review time, 0 commits of drift, `logs/00-tree-identity.log`);
  candidate tree `d176176776814911e60f84826b5cdd799dee4244`; the live Story worktree
  (tracked + untracked, `.temp` excluded) writes exactly that tree OID; my `git diff
  base candidate` is byte-identical to the attached `_change-request_rev2.patch`
  (sha256 `9ea381d8…`). `internal/fencing/gate.go` is byte-identical to revision 1;
  revision 2's production delta is the `terminstance.observeRemoteWinner` fallback.
- Normative authority: `internal/specdoc/SPEC.v0.7.0.md` §4.2 after-restore steps 1–5
  (lines 1433–1441), §13.4, §13.7, §4.C — the same pins as the rev1 verdict.
- All instruments ran on `git archive` copies of the two trees under
  `.temp/BUG-260917-2fwf8e/review-rev2/` (probe copies, harness copies and suite copy
  are separate directories); the Story worktree, index, branch and HEAD were not
  mutated. Host load during the review: 5–17 (a second Story's suite ran
  concurrently); wall-clock is recorded per gate.

## Verdict: CHANGES REQUESTED → `to-dev`

The code is right. Every finding of revision 1 is closed in the tree: probe 10 is
fixed at `Authorize`, `Decide` and the real `Run` entry; the §13.7/§13.4 stale-fencing
property is restored (the 60-row `ObserveFencing` grid is identical to trunk, my
property test passes 6/6 where it failed 6/6 on rev1); the landed `stale_fenced`
assertion is back in `TestRV3F3_GrantLessStaleFences`; the order is pinned on all
five gate entries and on `axpane` `ModeLaunch`; the two rev1 survivors (RM-A, RM-H)
and the RM-C gap are KILLED, RM-C now inside `internal/fencing`; the swap row is a
true narrowing, APPLIED and KILLED in both harness passes; both shipped harnesses
are 100% on two passes; scope holds; the full configured suite is green on the exact
tree (table at the end).

It is not acceptable as handed off because the board record of this revision is the
record of the rejected one. Both named outcome resources — `BUG-260917-2fwf8e_results.md`
and `BUG-260917-2fwf8e_conformance-matrix.md`, top-level AND the copies inside the
attached evidence archive — are the revision-1 files (byte-identical to what
RUN-260921-766ad0 reviewed; they still say "Updated old expiry-first expectations to
the correct remote-owner park/no transition" and "No pre-existing per-test outcome
changed or moved across a refusal arm", the two statements the rev1 P1/P2-2 were
about). The revision-2 results and matrix the rework brief required exist only as
unattached scratch files in the control checkout's `.temp/`, and the attached
terminstance battery evidence is a two-row targeted run, not the 122-row pass the
results describe. One P2, five P3.

Measured AC coverage against the six task-specific DoD rows: **6 of 6 driven in the
tree** (call sites below); row 5's named moved-vector table and row 6's stop
statement are not on the board (P2-1).

| DoD row | Production call site | Committed test / evidence | Measured |
| --- | --- | --- | --- |
| 1 remote interactive owner + lapsed grant → offer path, via `Authorize` itself | `fencing.Authorize` (restore + activation + input + mutation + checkpoint via `fenceEntries()`); `axpane.Decide` → `AuthorizeRestore`/`AuthorizeActivation` | `TestAuthorizeRemoteInteractiveOwnerLapsedGrantUsesRemoteOfferArm`, `TestAuthorizeLapsedRemoteOwnerUsesDirectionForEveryEntry` (5 subtests), `TestDecideLapsedRemoteOwnerOffersOrParksByInteractiveContext/{restore_interactive_attach,restore_interactive_takeover,restore_noninteractive_park,launch_interactive_attach}` | driven, 5 of 5 gate entries + 2 of 2 `axpane` modes; literal `remote_owner`/`not_owner` asserted, `lease_conflict` asserted absent |
| 2 ordering pinned by a test that fails on swap; swap shipped as a labelled narrowing row | `Authorize` | `N-arm-order-expiry-before-direction` (`internal/fencing/testdata/mutate.py`) | APPLIED, both gates present, KILLED ×2 (`harness/run{1,2}/N-arm-order-expiry-before-direction.log`: named killer fails on the literal `lease_conflict` at `authorize_order_test.go:24`; every-entry witness fails on all 5) |
| 3 probe 10 reproduced on trunk, passes after, before/after recorded | `axpane.Decide` (verbatim probe body), `axpane.Run` | my `logs/14-axpane-probe10-{trunk,cand}.log`, `logs/15-axpane-run-probe10-{trunk,cand}.log`; producer `logs/{baseline,candidate}-axpane-probe10-neighbours.log` | trunk `refused/lease_conflict` → candidate `attach_remote/remote_owner`; through `Run`: restore/launch × interactive → `attach_remote`, × non-interactive → `parked(remote_owner)`, events 1→1 (no event authored under the remote lease, as on every remote park) |
| 4 ≥3 neighbours a step away driven and stated | `Authorize` ×5 entries; `Decide` | `TestAuthorizeArmOrderingNeighbourVectors/{live_local_grant,remote_owner_live_grant,local_owner_lapsed_grant}`, `TestAuthorizeLapsedLocalOwnerStaleTokenRemainsLeaseConflict` (5), `TestAuthorizeRemoteOwnerInvalidPolicyRemainsInvalidArguments`, `…/restore_noninteractive_park` | live local → MINT (both trees); remote non-interactive + lapsed → `parked(remote_owner)` (trunk: refused); local lapsed → `lease_conflict` ×5 (both); remote + no grant → `invalid_arguments` (both, stated bound); unusable policy → `invalid_arguments` (both) |
| 5 per-test outcomes diffed before/after, moved inputs named | importer set | committed `internal/fencing/TRACEABILITY.md` (V01/V07/V08/V09/V14 before/after), `internal/terminstance/TRACEABILITY.md` row 86; my `logs/20-pertest-*` | 1,073 → 1,103 rows, 0 changed, 30 added; my outcome-keyed grids below name every moved class; the board `results.md` says the opposite (P2-1) |
| 6 scope held; stop instead of widen | diff | `logs/40-hygiene.log` | no `sessrepo`/`sessstate`/lease-chain path, park vocabulary identical (4 constants), `internal/traceability` untouched; the composition repair is the one the rework brief authorized |

## P2-1 — the attached outcome resources describe revision 1, and the terminstance battery evidence is a two-row targeted run

Measured (`logs/50-evidence-identity.log`):

| Artifact | On the board | Content |
| --- | --- | --- |
| `BUG-260917-2fwf8e_results.md` (top-level outcome, updated 22:28) | md5 `215e4f60…`, 5,819 B | revision-1 results: "Updated old expiry-first expectations to the correct remote-owner park/no transition"; "No pre-existing per-test outcome changed or moved across a refusal arm"; per-test diff `internal/fencing` only; no moved-vector table, no gate × entry census, no finding-by-finding closure, no stop statement |
| `BUG-260917-2fwf8e_conformance-matrix.md` (top-level outcome, 22:28) | md5 `e5b38b9d…`, 6,019 B | revision-1 matrix ("the only outside package changes are the required `terminstance` test/comment updates that remove the old expiry-first expectation") |
| the same two files inside `BUG-260917-2fwf8e_producer-evidence.tar.gz` | same md5s; the archive `MANIFEST.txt` hashes them | revision 1 |
| `mutants/terminstance/` inside the archive | `harness.stdout` 461 B, 2 rows (`N-fencing-remote-stale-only`, `N-fencing-remote-lapsed-not-stale`), `harness.rc` 0 | not the "120 narrowing rows plus one labelled tightening row" the results claim; the new `N-fencing-remote-lapsed-fallback` row is absent |
| `/Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/BUG-260917-2fwf8e/BUG-260917-2fwf8e_results.md` (control checkout, 22:27, 7,664 B, md5 `63a9f6f5…`) | **not attached anywhere** | the revision-2 results (moved-vector table, importer-set diff, battery claims, "Scope: … and its `terminstance.ObserveFencing` composition") |
| `…/.temp/BUG-260917-2fwf8e/BUG-260917-2fwf8e_conformance-matrix.md` (22:27, 2,798 B) | **not attached anywhere** | the revision-2 matrix |
| `…/.temp/BUG-260917-2fwf8e/mutants-terminstance-full-v4/harness.stdout` (72 KB, 122 rows, 122 raw blocks) | **not attached anywhere** | the full pass the results describe |

Cause (read from the producer spawn log, RUN-260921-fcea9e, lines 84973–84975): the
rework wrote the new results/matrix to the control checkout's
`.temp/BUG-260917-2fwf8e/`, then ran `task-board resource update` with
`$project_root/.temp/BUG-260917-2fwf8e/…` where `$project_root` was the Story
worktree — whose `.temp/BUG-260917-2fwf8e/` still held the 20:52 revision-1
copies. The archive was packed from the same stale copies.

Why it is a P2 and not hygiene: the rework brief made the results content a
deliverable ("finding-by-finding table, the moved-vector table, the gate × entry
census and its measured ratio", "say in the results, explicitly, what you would
have reported had you stopped"), and the DoD requires the outcome artifact and the
`n of m` ratio on the board. What is on the board contradicts the candidate on
exactly the two properties revision 1 was rejected for, so an integration that
carried it would carry a record of a regression that is not in the tree. Even the
unattached revision-2 results have no finding-by-finding table, no gate × entry
table with the three-valued cells, and no stop statement; the committed
`internal/fencing/TRACEABILITY.md` carries the entry ratio (5 of 5 + `ModeLaunch`)
and the moved-vector list, so the rework is a re-attachment plus the missing
sections, not new measurement.

## P3 — architecture, harness census, prose, bounds

- **P3-1 the relative-question arms of `observeRemoteWinner` are dead under the new
  composition, and the prose still presents them as the mechanism.** Two reviewer
  plants on the complete importer set (1,103 rows), both passes:
  `RM-O5-relative-question-through-mutation` (the retired `N-fencing-operation-remote`
  row re-planted) **SURVIVED 2/2** — so the producer's equivalence argument for
  retiring it holds; and `RM-O6-relative-question-dropped` (the relative
  `stale_owner` park arm deleted) **SURVIVED 2/2** — on revision 1 the same plant
  (rev1 RM-T) was KILLED by two tests. `fencing.StaleRelativeToWinner` now decides
  every outcome the second `Authorize` call used to decide (a relative `stale_owner`
  park implies a decided-stale verdict; a clean relative authorization implies
  not-stale). The candidate keeps both. Not a correctness defect — the second call is
  pure — but `internal/terminstance/fencing.go` ("stale_owner fences, anything else …
  leaves the state with the original remote park") and
  `internal/terminstance/TRACEABILITY.md` ("fences only on the relative stale verdict
  from a live source") now state a mechanism the measurement says is not load-bearing.
  Either drop the redundant relative arm (and its `N-fencing-remote-stale-only` row,
  which measures a widening of a dead arm) or keep it and state the redundancy as a
  measured bound in the doc comment and TRACEABILITY.
- **P3-2 the OUTER fallback lost its harness row.** `N-fencing-verdict-composition`
  (trunk: the `decided && stale` → `decided` narrowing of the `ObserveFencing` fallback)
  was renamed `N-fencing-remote-lapsed-not-stale` and re-anchored to the new INNER
  fallback; the outer site — the one every hard grant refusal and every local-winner
  grant-less shape goes through — now has no row. It is still pinned by a committed
  test: my `RM-O1-outer-fallback-decided-only` is KILLED 2/2 by
  `TestRV3F3_GrantLessDecidedNotStaleLeavesState` (5 fails), and
  `RM-O7-outer-fallback-stale-epoch-only` is KILLED 2/2 (23 fails). Re-add a row for
  the outer site with a unique anchor (e.g. include the following `return current,
  false, authErr\n}\n\n// observeRemoteWinner` lines) so the harness census matches the
  test census.
- **P3-3 the swap row's killer mask does not execute the `axpane` `ModeLaunch` cell.**
  `N-arm-order-expiry-before-direction` runs `./internal/fencing` only; the
  `launch_interactive_attach` cell is measured only when the importer set is the mask
  (my RM-A run lists `TestDecideLapsedRemoteOwnerOffersOrParksByInteractiveContext`
  among its 7 failures). State it as the cell's bound or add an `axpane` harness row.
- **P3-4 `internal/fencing/TRACEABILITY.md` names subtests that do not exist**:
  `…OffersOrParksByInteractiveContext/interactive_attach` and `/interactive_takeover`
  (actual: `restore_interactive_attach`, `restore_interactive_takeover`,
  `launch_interactive_attach`). `internal/axpane/decide.go` doc comments
  (`authorize`, `fencingClass`) still say "lapsed grants refuse" without the
  local-owner qualifier; untouched file, optional. The committed LOGBOOK entry
  attributes the 180 `MISSING_ACTIVITY` diagnostics to `cataloggen -check`; they come
  from `task-board validate` (cmd 29) — cmd 25 is silent on this tree
  (`suite/logs/cmd-25.log` is empty, `logs/41-task-board-validate.log` has 180).
- **P3-5 stated bound, pre-existing, for the Story's final leaf, not this one**:
  `RM-W1-direction-skipped-on-empty-localhost` (an empty `LocalHostID` treated as the
  winner's own host, so a remote winner's token MINTS under a live grant) **SURVIVED
  on the candidate 2/2 and on trunk 1/1** across the importer set. The
  `Observation.LocalHostID` doc comment promises "fails closed to the remote arm"; no
  committed test pins it. V14 (the empty-`LocalHostID` class the results name) moved
  with the fix in the documented fail-closed direction; the property itself is landed
  and unpinned.

## What holds (measured)

- Probe 10: the archived body (`TASK-260830-1geqhj_review-evidence-rev1.tar.gz`
  sha256 `9f473449…`, `evidence/probes/zz_review_probe_test.go:451-460`, recorded
  outcome `action=refused class=lease_conflict`) run verbatim — trunk
  `refused/lease_conflict`, candidate `attach_remote/remote_owner/not_owner`
  (`logs/14-*`). The producer's before/after logs in the archive agree.
- `Authorize` grid, 17 vectors × 5 entries (`logs/10-fencing-grid-{trunk,cand}.log`):
  moved classes are exactly V01 (probe 10), V07 stale-epoch, V08 same-epoch loser,
  V09 future epoch, V14 empty `LocalHostID`: `lease_conflict` ×5 → activation/restore
  `PARK(remote_owner, not_owner)`, input/mutation/checkpoint `REFUSE(not_owner)`.
  Unchanged: V02 remote+live, V03 local lapsed, V04 remote+no grant
  (`invalid_arguments`), V05/V06, V10 local lapsed + stale token (`lease_conflict`
  ×5), V11–V13 parks, V15/V16 expiry edge. The relative question from the winner's
  host answers `lease_conflict` for all four token classes under a lapsed grant.
- `ObserveFencing` outcome grid, 120 rows on each tree (`logs/11-*` 60 stale rows:
  **0 differences**, every stale row `stale_fenced`/transitioned under every grant
  shape; `logs/11b-*` 60 non-stale rows: 6 differences — winning/future token ×
  lapsed grant × remote winner × 3 states surface `PARK(remote_owner)` instead of
  `REFUSE(lease_conflict)`, transitioned=false on both trees). §13.7 property test:
  PASS 6/6 on trunk and candidate (`logs/13-*`).
- `Decide` neighbour grid A–J and `Run` probe on both trees (`logs/14-*`, `logs/15-*`).
- Name-keyed per-test diff over the importer set (`logs/20-*`): fencing 310→328,
  terminstance 303→310, axpane 195→200, termbind 265→265; 30 additions, 0
  removals/changes. Cross-runs (`logs/12-*`): trunk tests against candidate
  production — 8 leaf rows red, all at the rewritten precondition
  (`fencing_grantless_test.go:57` "want the grant-gated refusal (never a park)" for the
  6 lapsed/remote stale rows; `:101` "parked, want the grant refusal" for
  winning/future remote) — the intended moves, and the `stale_fenced` assertion is not
  what fails; candidate tests against trunk production — the fix witnesses fail on
  trunk (`TestAuthorizeRemoteInteractiveOwnerLapsedGrantUsesRemoteOfferArm`,
  `…UsesDirectionForEveryEntry` ×5, `TestDecide…` ×4, the rewritten `TestRV3F3_*` rows),
  while `TestObserveFencingRemoteWinnerLapsedGrantFencesStaleIncarnation`, the
  neighbour tests and the local-lapsed test pass on trunk (guards, as the results say).
- Shipped fencing harness (`harness/run{1,2}`, 34 s / 32 s): 37 rows — 32 KILLED
  (31 N + `T-census-alias`), `C-harmless-comment` SURVIVED, `C-not-applied`
  NOT_APPLIED, `C-compile-failure` classified, controls before/after exit 0, verdict
  lists identical, `gate.go` restored (sha256 equal to the candidate blob), zero
  `__pycache__`/`.pyc`.
- Shipped terminstance harness (`tharness/pass{1,2}.raw.log`, 118 s / 194 s,
  `AX_MUTANT_VERBOSE=1`): 122/122 `ok` both passes (120 N + `N-auth-epoch-high`
  tightening KILLED, `C-control` SURVIVED), 122 raw blocks with exits, verdict lists
  identical, package blobs byte-identical before/after (`tharness/*-blobs.sha256`).
  `N-fencing-remote-lapsed-fallback` KILLED (`…FencesStaleIncarnation/stale_epoch/active`),
  `N-fencing-remote-lapsed-not-stale` KILLED (`…DecidedNotStaleLeavesState/winning_token_remote`).
- Reviewer plants over the complete importer set (`review-mutants/run{1,2}`, 245 s
  each; one raw log + unified diff per plant; sources restored and hash-checked):

| Row | Kind | run1 / run2 | Killers (top-level) |
| --- | --- | --- | --- |
| RM-A restore-only direction-first (rev1 SURVIVED) | narrowing | KILLED / KILLED, 7 fails | `TestAuthorizeLapsedRemoteOwnerUsesDirectionForEveryEntry`, `TestDecideLapsedRemoteOwner…` |
| RM-H non-launch remote+lapsed → `lease_conflict` (rev1 SURVIVED) | narrowing | KILLED / KILLED, 4 | `…UsesDirectionForEveryEntry` |
| RM-C lapsed refusal only for the winning tuple | narrowing | KILLED / KILLED, 15 | `TestAuthorizeLapsedLocalOwnerStaleTokenRemainsLeaseConflict`, `TestRV3F3_*` |
| RM-C, killer mask `internal/fencing` only (rev1 gap) | narrowing | KILLED / KILLED, 6 | `TestAuthorizeLapsedLocalOwnerStaleTokenRemainsLeaseConflict` |
| RM-B direction arm only under a live grant | narrowing | KILLED / KILLED, 22 | fencing + axpane + terminstance |
| RM-D policy arm after direction | order | KILLED / KILLED, 8 | `TestAuthorizeRemoteOwnerInvalidPolicyRemainsInvalidArguments`, `TestRV3F3_GrantLessStaleFences` |
| RM-G lapsed remote parks `restore_policy` | narrowing | KILLED / KILLED, 19 | fencing + axpane + terminstance |
| RM-E1 deferred lapsed refusal for launch entries only (non-launch MINT) | narrowing | KILLED / KILLED, 14 | `TestAuthorizeRefusesExpiredGrant`, `…StaleTokenRemainsLeaseConflict`, `TestOwnershipGateClockNonAuthority` |
| RM-E2 lapsed refusal moved after the tuple arms | order | KILLED / KILLED, 15 | `…StaleTokenRemainsLeaseConflict`, `TestRV3F3_*` |
| RM-O1 OUTER fallback fences on decided alone | narrowing | (compile-fail on my first spelling) / KILLED, 5 | `TestRV3F3_GrantLessDecidedNotStaleLeavesState` |
| RM-O7 OUTER fallback fences stale-epoch only | narrowing | — / KILLED, 23 | `TestRV3F3_GrantLessStaleFences`, `TestRV3_FencingNoLocalTupleComparison` |
| RM-O2 INNER fallback fences from `active` only | narrowing | KILLED / KILLED, 10 | `…FencesStaleIncarnation`, `TestRV3F3_GrantLessStaleFences` |
| RM-O3 INNER fallback fences stale-epoch only | narrowing | KILLED / KILLED, 9 | same + `TestRV3_FencingNoLocalTupleComparison` |
| RM-O4 INNER fallback deleted | arm-delete (control) | KILLED / KILLED, 14 | same |
| RM-O5 relative question through `OperationMutation` (retired row) | equivalence probe | **SURVIVED / SURVIVED** | — (P3-1) |
| RM-O6 relative `stale_owner` arm deleted | arm-delete | **SURVIVED / SURVIVED** | — (P3-1) |
| RM-W1 empty `LocalHostID` treated as local | narrowing | **SURVIVED / SURVIVED**; trunk SURVIVED | — (P3-5, pre-existing) |
| C-harmless-comment | control | SURVIVED / SURVIVED | — |
| C-not-applied | control | NOT_APPLIED / NOT_APPLIED | — |

- Scope and forks: `sessrepo`, `sessstate`, the lease chain and the four park
  constants untouched; `internal/traceability` untouched; the composition delegates
  to the landed `StaleRelativeToWinner` and `Authorize` (no local tuple comparison);
  "under the winning lease" still means the remote host's lease when the winner is
  remote (`Run` authors no event under it, events 1→1).
- Hygiene (`logs/40-hygiene.log`, `logs/42-git-commands.log`): `gofmt -l` empty over
  the changed files and the whole candidate copy; `go vet` and
  `GOOS=windows GOARCH=amd64 go vet ./...` exit 0; `git diff --check` clean; JSON
  parse clean; zero `__pycache__`/`.pyc` in the candidate tree and the live
  worktree; `task-board validate` exit 0 with the standing "180 issue(s)" ledger
  notice, 2738/2738 mirrored (`logs/41-task-board-validate.log`); the evidence
  archive is a real gzip with a MANIFEST without a self-entry and no foreign archive
  (the P3-1 of rev1 is closed).

## Rework scope (for the producer)

1. Attach the revision-2 `BUG-260917-2fwf8e_results.md` and
   `BUG-260917-2fwf8e_conformance-matrix.md` (`task-board resource update … --type
   outcome`, from the path you actually wrote to) and re-pack the evidence archive
   with them, with the full 122-row terminstance battery stdout (`mutants-terminstance-full-v4`
   or a fresh pass, raw blocks and exits included), and with a MANIFEST that hashes
   the new bytes. Before handoff, `task-board resource get` both files and diff them
   against what you meant to attach.
2. Add to the results what the rework brief asked for and the unattached draft still
   lacks: the finding-by-finding closure table (P1-1, P2-1, P2-2, P3-1..P3-3 of rev1);
   the gate × entry census as its own table (restore, activation, input, mutation,
   checkpoint, `axpane` `ModeLaunch`/`AuthorizeActivation` — each cell: named test +
   the narrowing row that kills it, or `unreachable`, or a stated bound; for the
   `axpane` cell the bound is P3-3 above unless you add a row); the composed-entry
   moves (`ObserveFencing`: winning/future token × lapsed × remote now surface the
   `remote_owner` park instead of `lease_conflict`, no transition either way); and the
   explicit "what I would have reported had I stopped" statement.
3. P3-1: decide the redundant relative arm — remove it with its `N-fencing-remote-stale-only`
   row, or keep it and rewrite the `fencing.go` doc comment and the terminstance
   TRACEABILITY sentence so they no longer claim it decides the remote stale case;
   either way state the measured redundancy (RM-O5/RM-O6) as a bound.
4. P3-2: restore a harness row for the OUTER fallback with a unique anchor;
   P3-4: fix the subtest names in `internal/fencing/TRACEABILITY.md`.
5. No production change is required by this verdict beyond the optional P3-1
   simplification; keep `gate.go` and the terminstance fallback as they are.

## Full configured suite on the exact tree (30 commands read from the candidate `task-board.config.json`)

Commands 2–27 and the extra windows vet ran from `suite/run-suite.sh` on a probe-free
`git archive` copy of `d1761767` (`suite/logs/cmd-NN.log` + `.rc`); commands 1, 28,
30 ran read-only against the live worktree (`logs/42-git-commands.log`) and command
29 against the authoritative board (`logs/41-task-board-validate.log`). Wall-clock is
with the concurrent Story's suite on the host.

| # | Command | rc | Elapsed | Note |
| --- | --- | ---: | ---: | --- |
| 1 | gofmt -l over tracked+untracked Go files | 0 | — | empty (live worktree) |
| 2 | go build ./... | 0 | 4s | |
| 3 | go vet ./... | 0 | 4s | |
| 4 | go test ./... -count=1 -v | 0 | 362s | 44 ok, 0 FAIL (the 21 indented `--- FAIL` lines are child-`go test` output captured by smoke tests, all inside passing tests) |
| 5 | go test ./... -race -count=1 -timeout 25m | 0 | 432s | 44 ok, 0 FAIL; load 15.5 |
| 6 | go test ./... -cover -count=1 | 0 | 243s | fencing 99.3%, terminstance 86.6%, axpane 81.2%, termbind 83.7% — equal to the producer's figures |
| 7–23 | 17 fuzz lanes, 100x, parallel=1 | 0 each | 0–7s each | |
| 24 | tracecheck | 0 | 0s | `clauses_discharged=70/574` |
| 25 | cataloggen -adopted -check | 0 | 1s | no output |
| 26 | GOOS=linux go build | 0 | 2s | |
| 27 | GOOS=windows go build | 0 | 2s | |
| 28 | JSON parse of tracked *.json | 0 | — | live worktree |
| 29 | task-board validate (authoritative board) | 0 | — | standing "180 issue(s)" ledger notice, 2738/2738 mirrored |
| 30 | git diff --check | 0 | — | live worktree |
| extra | GOOS=windows GOARCH=amd64 go vet ./... | 0 | 1s | |

Suite green on the exact tree: the verdict is about the board record, not the code.

## Evidence index (`BUG-260917-2fwf8e_review-evidence-rev2.tar.gz`)

- `logs/00-tree-identity.log` — live worktree tree OID == candidate, origin/main == base, patch sha256.
- `logs/01-candidate.patch` — the reviewed delta (sha256 equal to the CR patch resource).
- `logs/10-fencing-grid-{trunk,cand}.log` — 17 vectors × 5 entries at `Authorize`; relative-question rows.
- `logs/11-terminstance-grid-{trunk,cand}.log` — 60 stale-token `ObserveFencing` rows (0 diff); `logs/11b-terminstance-nonstale-grid-{trunk,cand}.log` — 60 winning/future rows (6 moved).
- `logs/12-cross-candprod-trunktests.jsonl`, `logs/12-cross-trunkprod-candtests.jsonl` — both cross-runs.
- `logs/13-property-stale-remote-lapsed-{trunk,cand}.log` — §13.7 property test, 6/6 both trees.
- `logs/14-axpane-probe10-{trunk,cand}.log`, `logs/15-axpane-run-probe10-{trunk,cand}.log` — probe 10 verbatim, neighbour grid A–J, the `Run` entry.
- `logs/20-pertest-*.{jsonl,tsv}`, `logs/20-pertest-diff.txt` — name-keyed per-test diff over the importer set.
- `logs/30-importer-census.log`, `logs/40-hygiene.log`, `logs/41-task-board-validate.log`, `logs/42-git-commands.log`, `logs/50-evidence-identity.log` (P2-1).
- `probes/` — my rev2 probe; `probes/reused-from-rev1-review/` — the rev1 reviewer probes rerun unchanged.
- `harness/run{1,2}.stdout`, `harness/run{1,2}/*.log`, `mutants-full.json` — shipped fencing harness, two passes.
- `tharness/pass{1,2}.raw.log` (122 raw blocks each), `pass{1,2}.verdicts.log`, `*.meta`, `*-blobs.sha256` — shipped terminstance harness, two passes.
- `review-mutants/review_mutants_rev2.py`, `run{1,2}/` (one `.log` + `.diff` per plant, `review-mutants.json`), `trunk-W1/` — reviewer plants.
- `suite/run-suite.sh`, `suite/logs/cmd-NN.log` + `.rc`, `suite-summary.log` — the configured suite.
