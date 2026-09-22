# BUG-260917-2fwf8e — review verdict, Change Request revision 3

- Reviewer run: RUN-260921-a9f83c (claude-opus-5 max), role `reviewer`, read-only.
- Change Request: `CR-BUG-260917-2fwf8e-3` rev 3; base `799c338e401fca0b24859c0b870cd665927209f2`
  (= `origin/main` at review time after `git fetch origin main`; `git diff --name-only base origin/main`
  is empty, so no refresh-carried stale copies are possible); candidate tree
  `6b7603cfaa1ada6a49e05ceeef6b4cf97f1c3cca`. The live Story worktree (tracked + untracked, computed
  with a temporary index, `.temp` ignored) writes exactly that tree OID at the start
  (`00-live-worktree-tree-oid.txt`) and at the end (`00-live-worktree-tree-oid-end.txt`) of the review;
  my `git diff base candidate` is byte-identical to the attached `_change-request_rev3.patch`
  (sha256 `8cede041…`). Revision 3 differs from revision 2 in exactly five paths: a doc-comment
  paragraph in `internal/terminstance/fencing.go` (the P3-1 bound), one new harness row in
  `internal/terminstance/mutant_harness.py` (P3-2), and prose in the two TRACEABILITY files, LOGBOOK
  and README (`02-rev2-to-rev3-*.diff`). `internal/fencing/gate.go` is byte-identical to revisions 1 and 2.
- Normative authority: `internal/specdoc/SPEC.v0.7.0.md` §4.2 after-restore steps 1–5 (lines 1433–1441,
  `05-spec-4.2-after-restore.txt`), §13.4, §13.7, §4.C.
- Every instrument ran on `git archive` copies of the two trees under
  `.temp/BUG-260917-2fwf8e/review-rev3/` (immutable `cand/` and `trunk/`; separate probe, cross-run,
  harness, plant and suite copies). The Story worktree, index, branch and HEAD were not mutated (one
  `git add -N` I ran at the start was reverted with `git reset -- <path>` before any measurement; the
  index is back to its original state and the tree OID is unchanged). Host load 8–24 throughout (a
  second Story's suite ran concurrently); wall-clock recorded per gate.

## Verdict: ACCEPTED → `accept_cr(BUG-260917-2fwf8e, revision=3)`

The code was already right at revision 2 and is unchanged in substance here. Revision 3 closes the one
P2 of revision 2 — the board record now describes this tree, not the rejected revision 1 — and every
P3. I reproduced the defect and the fix myself on both trees at three production entries, drove the
neighbour vectors at every gate entry, diffed per-test outcomes and outcome grids across the complete
importer set, re-ran both shipped harnesses twice, planted nineteen rows of my own twice, and ran the
full 30-command configured suite on the exact tree. No P1, no P2. Four P3 notes are recorded below
for the Story's final leaf; none contradicts the tree or the record, so none justifies a fourth
revision.

Measured AC coverage against the six task-specific DoD rows: **6 of 6 driven in the tree**; task AC
(a)/(b)/(c): **3 of 3**.

| DoD row | Production call site | Committed test / evidence | Measured |
| --- | --- | --- | --- |
| 1 remote interactive owner + lapsed local grant → offer path via `Authorize` itself, not `lease_conflict` | `fencing.Authorize` through all five entries (`AuthorizeRestore/Activation/Input/Mutation/Checkpoint`); `axpane.Decide` → `authorize()` → `AuthorizeRestore`/`AuthorizeActivation` → `decideParked` | `TestAuthorizeRemoteInteractiveOwnerLapsedGrantUsesRemoteOfferArm`, `TestAuthorizeLapsedRemoteOwnerUsesDirectionForEveryEntry` (5 subtests), `TestDecideLapsedRemoteOwnerOffersOrParksByInteractiveContext/{restore_interactive_attach,restore_interactive_takeover,restore_noninteractive_park,launch_interactive_attach}` | driven; literals `remote_owner`/`not_owner`/`attach_remote`/`takeover_offer` asserted, `lease_conflict` asserted absent; my grid: restore/activation `PARK(remote_owner,cause=not_owner)`, input/mutation/checkpoint `REFUSE(not_owner)` (`logs/11-*`) |
| 2 ordering pinned by a test that fails on swap; swap shipped as a labelled narrowing row | `Authorize` | `N-arm-order-expiry-before-direction` in `internal/fencing/testdata/mutate.py` | APPLIED (the harness classifies NOT_APPLIED separately and did so for its control), both arms present and only reordered (`DIRECTION_THEN_EXPIRY`→`EXPIRY_THEN_DIRECTION`), KILLED ×2 by the named killers on the literal `lease_conflict` (`harness/fencing-pass{1,2}/N-arm-order-expiry-before-direction.log`); my re-plant under the importer-set mask KILLED ×2 with 22 failing rows across fencing/axpane/terminstance |
| 3 probe 10 reproduced on trunk before, passes after, before/after recorded | `axpane.Decide` (verbatim probe body), `axpane.Run` | `logs/10-axpane-probe-{trunk,cand}.log`; producer `logs/{baseline,candidate}-axpane-probe10-neighbours.log` | trunk `action=refused class=lease_conflict` → candidate `action=attach_remote reason=remote_owner cause=… not_owner`; through the real `Run` with a durable remote takeover: restore/launch × interactive → `attach_remote`, × non-interactive → `parked(remote_owner)` under lease B (the remote host's), `emitted=false`, events 1→1 on both trees |
| 4 ≥3 neighbours a step away driven and stated | `Authorize` ×5 entries; `Decide` | `TestAuthorizeArmOrderingNeighbourVectors/{live_local_grant,remote_owner_live_grant,local_owner_lapsed_grant}`, `TestAuthorizeLapsedLocalOwnerStaleTokenRemainsLeaseConflict` (5), `TestAuthorizeRemoteOwnerInvalidPolicyRemainsInvalidArguments`, `…/restore_noninteractive_park` | live local → MINT ×5 / `launch`; remote + live → same park/refusal as remote + lapsed; local + lapsed → `lease_conflict` ×5 for all four token classes (both trees); remote non-interactive + lapsed → `parked(remote_owner)`; remote + no grant / no clock / unusable policy (also unusable policy AND lapsed) → `invalid_arguments` on both trees, stated as a bound by the results |
| 5 per-test outcomes diffed before/after, moved inputs named | importer set (`fencing`, `terminstance`, `axpane`, `termbind`) | producer `outcomes/per-test-diff.tsv` + the moved-vector table in the results; my `logs/20-pertest-diff.txt`, `logs/11-fencing-grid-diff.txt`, `logs/12-terminstance-grid-diff.txt`, `logs/10-axpane-grid-diff.txt`, `logs/21-cross-runs.txt` | 1,073 → 1,103 rows, 0 changed, 0 removed, 30 added; outcome grids: exactly the classes the results name moved (below) |
| 6 scope held; stop instead of widen | diff | `logs/40-scope.log`; results "Explicit stop statement" | `sessrepo`, `sessstate`, `fencing/staleness.go`, `fencing/token.go`, `internal/traceability` byte-identical to trunk; the four park constants and six error sentinels identical; the stop statement is on the board |

## What I measured

### Probe 10 and the composed entries (`logs/10-*`, `logs/14`-equivalent)

The verbatim probe body (`TASK-260830-1geqhj_review-evidence-rev1.tar.gz` sha256 `9f473449…`,
`evidence/probes/zz_review_probe_test.go:451-460`) compiles and runs on both trees when extracted on
its own; the producer's note that the whole archived file no longer compiles (`Realm.AttestedServer`,
5 references, absent from the candidate) is correct and does not affect the probe. Recorded outcome
in the archive: `action=refused class=lease_conflict`; trunk today: identical; candidate:
`action=attach_remote class= reason=remote_owner cause=parked: park (remote_owner) under winning
lease aaaaaaaa-… : not_owner`.

`Decide` neighbour grid, 120 rows (mode × owner{local,remote,empty LocalHostID} × grant{live,lapsed,
absent,noclock,badpolicy} × remoteInteractive × attachAdmitted): 16 rows moved, all
`owner∈{remote,emptyhost} × grant=lapsed`, from `refused/lease_conflict` to `attach_remote` /
`takeover_offer` / `parked(remote_owner)` by interactivity and attach admission — the same rows the
live grant already produced on trunk. `Run` grid, 8 rows over a real chain after a remote CAS
takeover: the 4 lapsed rows moved the same way; the winning lease reported is lease B held by the
remote host, and no event is authored under it (`emitted=false`, events 1→1) on either tree — "under
the winning lease" still means the other host's lease.

### `Authorize` grid, 1,008 rows (`logs/11-fencing-grid-*`)

6 ownership shapes × 7 grant shapes × 4 token classes × 5 entries, plus the relative question from
the winner's host with the `StaleRelativeToWinner` verdict beside it. Exactly 40 rows moved:
`{remote, empty LocalHostID} × lapsed_by_1s × {winning, stale_epoch, losing_lease, future_epoch}
× 5 entries`, every one `REFUSE(lease_conflict)` → activation/restore `PARK(remote_owner,cause=not_owner)`,
input/mutation/checkpoint `REFUSE(not_owner)`. These are precisely the V01/V07/V08/V09/V14 classes the
results and `internal/fencing/TRACEABILITY.md` name. Unchanged: every live and boundary-instant row,
every absent/noclock/badpolicy row (including `badpolicy_and_lapsed`, which proves policy validity is
still checked before direction), every unverified/ambiguous/failed-handoff row, and every local row.
The relative question under a lapsed grant answers `REFUSE(lease_conflict)` for all four token
classes on both trees while `StaleRelativeToWinner` decides them — which is why the composition needs
the grant-independent fallback.

### `ObserveFencing` grid, 320 rows, and the §13.7 property (`logs/12-*`)

winner{local,remote} × grant{live,lapsed,absent,noclock,badpolicy} × 4 tokens × 8 states. 24 rows
moved, all `remote × lapsed × {winning, future_epoch} × every state` plus `remote × lapsed ×
{stale_epoch, losing_lease} × {absent, stopped, stale_fenced, unavailable}` — every moved row changes
ONLY the surfaced error (`REFUSE(lease_conflict)` → `PARK(remote_owner)`) with `transitioned=false`
on both trees. The stale-fencing transitions are identical: 80 `stale_fenced` transitions on each
tree; every stale token under a remote winner fences from creating/parked/active/quiescing under all
five grant shapes. My property test (`TestRev3StaleUnderRemoteWinnerLapsedGrantFences`, 8 rows)
passes on both trees. The revision-1 regression is closed.

### Per-test and cross-run diffs (`logs/20-*`, `logs/21-*`, `pertest/`)

Name-keyed, importer set: fencing 310→328, terminstance 303→310, axpane 195→200, termbind 265→265;
0 changed, 0 removed, 30 added, 0 candidate failures. Trunk test files against candidate production:
10 rows red, all at the landed precondition assertions that encoded expiry-before-direction
(`fencing_grantless_test.go:57` "want the grant-gated refusal (never a park)", `:101` "parked, want
the grant refusal") — the `stale_fenced` assertion is never what fails, so the candidate's rewrite of
those two tests is the legitimate update and the rewritten `TestRV3F3_GrantLessStaleFences` keeps the
literal `stale_fenced` assertion on the lapsed/remote rows (read at
`fencing_grantless_test.go:58-76`). Candidate test files against trunk production: 22 rows red — the
fix witnesses (`TestAuthorizeRemoteInteractiveOwnerLapsedGrantUsesRemoteOfferArm`,
`…UsesDirectionForEveryEntry` ×5, `TestDecide…` ×4, the rewritten `TestRV3F3_*` rows) — while
`TestObserveFencingRemoteWinnerLapsedGrantFencesStaleIncarnation`, the neighbour tests and the
local-lapsed test pass on trunk (guards, as the results say).

### Shipped harnesses, two passes each, isolated copy (`harness/`)

- `internal/fencing/testdata/mutate.py`: 37 rows, 33 s / 32 s. 31 `N-` rows + `T-census-alias` KILLED
  (32/32), `C-harmless-comment` SURVIVED, `C-not-applied` NOT_APPLIED, `C-compile-failure`
  COMPILE_OR_HARNESS_FAILURE, controls before/after exit 0, verdict lists identical across passes,
  zero `__pycache__`/`.pyc`.
- `internal/terminstance/mutant_harness.py` (`AX_MUTANT_VERBOSE=1`): 123 rows, 211 s / 81 s. 122 KILLED
  (121 narrowing + `N-auth-epoch-high` tightening), `C-control` SURVIVED, 123 raw blocks with exits per
  pass, verdict lists identical, package blobs byte-identical before/after each pass
  (`harness/terminstance-blobs-*.sha256`). The new `N-fencing-remote-lapsed-outer-fallback` row is
  anchored uniquely (outer return + `observeRemoteWinner` declaration), APPLIED and KILLED ×2 by
  `TestRV3F3_GrantLessDecidedNotStaleLeavesState` (4 rows); `N-fencing-remote-lapsed-fallback` and
  `N-fencing-remote-lapsed-not-stale` KILLED ×2.

### Reviewer plants, complete importer-set mask, two passes (`review-mutants/`, `probes/review_mutants_rev3.py`)

One raw `-json` log and one unified diff per plant per pass; sources restored from pristine bytes and
sha256-checked; `rm-cand/internal` diff-identical to the candidate after both passes.

| Plant | File | Kind | pass 1 / pass 2 | Failing rows (top-level) |
| --- | --- | --- | --- | --- |
| C3-harmless-comment | `fencing/gate.go` | control-survive | SURVIVED / SURVIVED | — |
| C3-not-applied | `fencing/gate.go` | control-not-applied | NOT_APPLIED / NOT_APPLIED | — |
| C3-swap-reproduced | `fencing/gate.go` | order (producer swap row re-planted) | KILLED / KILLED, 22 | `TestAuthorizeLapsedRemoteOwnerUsesDirectionForEveryEntry`, `TestAuthorizeRemoteInteractiveOwnerLapsedGrantUsesRemoteOfferArm`, `TestDecideLapsedRemoteOwnerOffersOrParksByInteractiveContext`, `TestRV3F3_GrantLessDecidedNotStaleLeavesState`, `TestRV3F3_GrantLessStaleFences` |
| RM3-B policy-validity arm for launch entries only | `fencing/gate.go` | narrowing | KILLED / KILLED, 5 | `TestAuthorizeRefusesExpiredGrant/zero_policy` |
| RM3-C direction arm only under a live grant | `fencing/gate.go` | narrowing | KILLED / KILLED, 22 | same set as the swap |
| RM3-D non-launch remote+lapsed falls to `lease_conflict` | `fencing/gate.go` | narrowing | KILLED / KILLED, 4 | `TestAuthorizeLapsedRemoteOwnerUsesDirectionForEveryEntry/{input,mutation,checkpoint}` |
| RM3-E lapsed refusal only for the winning tuple | `fencing/gate.go` | narrowing | KILLED / KILLED, 15 | `TestAuthorizeLapsedLocalOwnerStaleTokenRemainsLeaseConflict`, `TestRV3F3_*` |
| RM3-F lapsed launch parks `restore_policy` instead of refusing | `fencing/gate.go` | narrowing (park-vocabulary change) | KILLED / KILLED, 25 | `TestAuthorizeRefusesExpiredGrant`, `TestAuthorizeArmOrderingNeighbourVectors`, `TestDecideTable`, `TestOwnershipGateClockNonAuthority`, … |
| RM3-G policy arm moved after direction | `fencing/gate.go` | order | KILLED / KILLED, 8 | `TestAuthorizeRemoteOwnerInvalidPolicyRemainsInvalidArguments`, `TestRV3F3_GrantLessStaleFences` |
| RM3-H lapsed refusal moved after the tuple arms | `fencing/gate.go` | order | KILLED / KILLED, 15 | `TestAuthorizeLapsedLocalOwnerStaleTokenRemainsLeaseConflict`, `TestRV3F3_*` |
| RM3-I empty `LocalHostID` treated as the winner's host | `fencing/gate.go` | narrowing | **SURVIVED / SURVIVED; trunk SURVIVED** | — (pre-existing; P3-5 of rev2, recorded as a bound by the results) |
| RM3-P lapsed remote parks `restore_policy` (offer removed) | `fencing/gate.go` | narrowing | KILLED / KILLED, 19 | literal `remote_owner` assertions in fencing, axpane, terminstance |
| RM3-Q lapsed refusal code swap `lease_conflict`→`not_owner` (same exit) | `fencing/gate.go` | code swap | KILLED / KILLED, 22 | `TestAuthorizeRefusesExpiredGrant`, `TestAuthorizeArmOrderingNeighbourVectors`, `…StaleTokenRemainsLeaseConflict`, `TestOwnershipGateClockNonAuthority` |
| RM3-J inner fallback deleted | `terminstance/fencing.go` | arm-delete (control) | KILLED / KILLED, 14 | `TestObserveFencingRemoteWinnerLapsedGrantFencesStaleIncarnation`, `TestRV3F3_GrantLessStaleFences` |
| RM3-K inner fallback fences from `active` only | `terminstance/fencing.go` | narrowing | KILLED / KILLED, 10 | same |
| RM3-L relative `stale_owner` arm deleted | `terminstance/fencing.go` | arm-delete | **SURVIVED / SURVIVED** | — (the documented P3-1 bound, measured again) |
| RM3-M outer fallback fences on `decided` alone (producer's new row re-planted) | `terminstance/fencing.go` | narrowing | KILLED / KILLED, 5 | `TestRV3F3_GrantLessDecidedNotStaleLeavesState` |
| RM3-N the whole remote-park dispatch to `observeRemoteWinner` deleted | `terminstance/fencing.go` | arm-delete | **SURVIVED / SURVIVED** | — (P3 note 2 below) |
| RM3-O the wrapper re-checks expiry and parks instead of offering | `axpane/decide.go` | narrowing at the composed entry | KILLED / KILLED, 4 | `TestDecideLapsedRemoteOwnerOffersOrParksByInteractiveContext` |

Thirteen kills plus the swap; the three survivors are the two bounds the record already states and
one structural observation. Every kill includes at least one test that asserts the literal token, so
none of the code-identity pins is circular.

### Scope, forks, hygiene (`logs/40-scope.log`, `logs/41-doc-test-names.txt`)

- 11 changed paths, none under `sessrepo`, `sessstate` or `internal/traceability`; those owners,
  `fencing/staleness.go` and `fencing/token.go` are byte-identical to trunk; the four `ParkReason`
  constants and six sentinels are identical. `CheckFencingExpiry` remains the expiry owner;
  `terminstance` performs no local tuple comparison (the composition delegates to `Authorize` and
  `StaleRelativeToWinner`); the `Run` probe shows no event authored under the remote host's lease.
- All ten test references this leaf added to `internal/fencing/TRACEABILITY.md` resolve to real tests
  (the rev2 P3-4 names are fixed); the two it added to `internal/terminstance/TRACEABILITY.md` resolve
  (one is the harness `-run` prefix `…/stale_epoch`, matching that file's existing convention).
- `gofmt -l` empty over tracked + untracked Go files; `go vet` and `GOOS=windows GOARCH=amd64 go vet ./...`
  exit 0; `git diff --check` clean; zero `__pycache__`/`.pyc` in the candidate tree, the live worktree,
  and every harness copy.

### Board record (the revision-2 P2)

`BUG-260917-2fwf8e_results.md` (13,811 B, sha256 `b16e0e3e…`) and `_conformance-matrix.md` (8,759 B,
`38b7b84a…`) on the board are the revision-3 files and are byte-identical to the copies inside the
evidence archive; the archive (`_rev3-evidence.tar.gz` == `_producer-evidence.tar.gz`, sha256
`e26cab6d…`) is a real gzip whose `MANIFEST.txt` hashes 1,821 of 1,821 members correctly, has no
self-entry, and lists no unlisted file; no foreign archive; the terminstance battery inside it is
the 123-row pass with 123 raw blocks per pass and the `N-fencing-remote-lapsed-fallback` row present.
The results carry the finding-by-finding table, the gate × entry census with three-valued cells
(5 of 5 entries + the `ModeLaunch` cell stated as a bound — my swap re-plant under the importer-set
mask shows that cell IS killed, so the bound is conservative), the moved-vector table, and the stop
statement. The record now describes the tree.

## P3 notes (recorded for the Story's final leaf; no rework requested)

1. **`internal/axpane/decide.go` doc comments still say "lapsed grants refuse"** (`authorize`, line 585;
   `fencingClass`, line 1063). The rev3 brief made fixing them optional but asked the producer to say
   so if left; the results do not. Line 585 is now true only for a local owner. Untouched file; fold the
   two-word qualifier into whichever leaf next touches `axpane`.
2. **The remote-winner dispatch is structurally redundant, not just its relative arm.** RM3-N deletes
   the entire `if parked && reason == ParkRemoteOwner { return observeRemoteWinner(…) }` dispatch and
   the complete importer set stays green twice: the OUTER `StaleRelativeToWinner` fallback in
   `ObserveFencing` produces the same fence/no-fence outcome and surfaces the same `authErr` for every
   remote-winner row under every grant shape. The candidate documents the relative arm as a measured
   bound (RM-O5/RM-O6), which the rev2 verdict accepted as a resolution; this note only records that
   the inner fallback rows (`N-fencing-remote-lapsed-fallback`, `-not-stale`) are KILLED because the
   dispatch keeps them on-path, and that `observeRemoteWinner` as a whole could be removed without any
   behavioural change. A simplification for a later leaf, not a defect.
3. **Remote interactive owner with NO grant at all is a stated bound on both trees**: `!HasGrant` refuses
   `invalid_arguments` before direction, so `Decide` refuses rather than offering. This leaf's scope is
   the lapsed-grant ordering and the results name the no-grant/no-clock/unusable-policy neighbours as
   preserved, so it is not a finding here. It is worth a product decision in the leaf that mints grants
   (no production code constructs a `FencingGrant` or calls `VerifyFencingToken` yet): after a
   controller restart the prior owner has no grant, and a local token cannot be revalidated against a
   remote holder, so whether §4.2 step 4 must also be reachable with `HasGrant=false` is unresolved.
4. Evidence hygiene: `outcomes/tree-identity.txt` in the producer archive records `candidate_worktree=799c338`
   (the base commit), not the candidate tree OID; the results' "309 vs 327" fencing row counts are the
   `./internal/fencing` mask, mine (`./internal/fencing/...`) are 310/328 — same delta of 18.

## Full configured suite on the exact tree (30 commands read from the candidate `task-board.config.json`)

Commands 2–27 and the extra Windows vet ran from a probe-free copy of tree `6b7603cf` that includes
the committed `.task-board` (my first attempt of command 4 omitted it and `internal/specpin`
failed on `lstat ../../.task-board`; that attempt is kept as `cmd-04-attempt1-incomplete-copy.log`
and was rerun on the complete copy). Commands 1, 28, 30 ran read-only against the live worktree;
command 29 against the authoritative board.

| # | Command | rc | Elapsed | Note |
| --- | --- | ---: | ---: | --- |
| 1 | gofmt -l over tracked+untracked Go files | 0 | — | empty |
| 2 | go build ./... | 0 | 1s | |
| 3 | go vet ./... | 0 | 2s | |
| 4 | go test ./... -count=1 -v | 0 | 153s | 44 ok, 0 FAIL; load 10–11 |
| 5 | go test ./... -race -count=1 -timeout 25m | 0 | 412s | 44 ok, 0 FAIL, no DATA RACE; load 11→17 |
| 6 | go test ./... -cover -count=1 | 0 | 191s | fencing 99.3%, terminstance 86.6%, axpane 81.2%, termbind 83.7% — equal to the producer's figures; load up to 24 |
| 7–23 | 17 fuzz lanes, 100x, parallel=1 | 0 each | 0–4s each | |
| 24 | tracecheck | 0 | 0s | `contracts=64 acceptance_cases=152 clauses_discharged=70/574` |
| 25 | cataloggen -adopted -check | 0 | 0s | silent (0 bytes) |
| 26 | GOOS=linux go build | 0 | 1s | |
| 27 | GOOS=windows go build | 0 | 1s | |
| 28 | JSON parse of tracked *.json | 0 | — | |
| 29 | task-board validate (authoritative board) | 0 | 6s | standing "180 issue(s)" `MISSING_ACTIVITY` ledger notice, 2766/2766 mirrored |
| 30 | git diff --check | 0 | — | |
| extra | GOOS=windows GOARCH=amd64 go vet ./... | 0 | 2s | |

30 of 30 green on the exact tree.

## Evidence index (`BUG-260917-2fwf8e_review-evidence-rev3.tar.gz`)

- `00-*` — start/end UTC, load, live-worktree tree OID before and after, git status, resource sha256s.
- `01-cr-delta.diff` / `.stat` — the reviewed delta (byte-identical to the CR patch resource);
  `02-rev2-to-rev3-{code,docs}.diff` — what revision 3 changed relative to revision 2.
- `03-evidence-files.txt`, `03-manifest-check.txt` — producer archive census and manifest verification.
- `04-validation-commands.txt`, `05-spec-4.2-after-restore.txt`.
- `logs/06-tool-readiness.log`; `logs/10-axpane-*` (probe 10 verbatim, `Decide` grid, `Run` grid, diff);
  `logs/11-fencing-grid-*` (1,008-row `Authorize` grid + relative question, diff);
  `logs/12-terminstance-grid-*` (320-row `ObserveFencing` grid, §13.7 property, diff);
  `logs/13-importer-census.txt`; `logs/20-pertest-*`, `pertest/*.jsonl|tsv` (name-keyed diff);
  `logs/21-cross-runs.txt`, `pertest/{candprod-trunktests,trunkprod-candtests}.jsonl`;
  `logs/30-*`, `harness/fencing-pass{1,2}/` (shipped fencing harness ×2, per-plant logs, JSON);
  `logs/31-*`, `harness/terminstance-pass{1,2}.raw.log`, `harness/terminstance-blobs-*.sha256`;
  `logs/32-review-mutants-runs.log`, `review-mutants/{pass1-gate,pass1-rm3,pass2,trunk-RM3-I}/`
  (one `.log` + `.diff` per plant, `review-mutants.json`);
  `logs/40-scope.log`, `logs/41-doc-test-names.txt`; `suite/logs/cmd-NN.log` + `.rc`, `summary.txt`.
- `probes/` — `zz_rev3_axpane_probe_test.go`, `zz_rev3_fencing_grid_test.go`,
  `zz_rev3_terminstance_grid_test.go`, `cross-run.py`, `review_mutants_rev3.py`.
