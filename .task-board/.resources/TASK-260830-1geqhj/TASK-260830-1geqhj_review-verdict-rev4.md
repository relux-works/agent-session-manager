# TASK-260830-1geqhj — independent review verdict, CR-TASK-260830-1geqhj-4 revision 4

**Verdict: ACCEPTED (`accept_cr` revision 4 → `integrating`).** No P1, no P2.
Seven P3 items are recorded below for the Story's follow-up leaves; none of
them is a production defect on a path `Run` reaches, and none blocks the
repository DoD.

Reviewer: RUN-260917-e644e0 (claude-opus-5 max). Reviewed bytes: base
`2fc6d5074058ab7970c87982deac50b3e3f5fd91` (= `origin/main` at review time,
re-fetched), candidate tree `c61b754c5c049cb22c54ea8b10be8a322ca3de17`
(recomputed from the live Story worktree with a temporary untracked-aware
index: equal), patch sha256
`116f644906b2bdcb9933a6e528ca1e336ebdc5101094eb83938cd76cdda4a59c` (equal to
the CR record and to `git diff --binary` of the two OIDs). Every probe ran in
isolated `git archive` copies of the exact tree under
`.temp/TASK-260830-1geqhj/rev4-review/` (`cand/` tree-exact checks, `cand-p/`
probes, `cand-h/` shipped harness, `cand-m/` reviewer mutants; each copy has
its own `.git`, and the post-run `write-tree` of `cand-h`/`cand-m` equals the
candidate OID). The live Story worktree, index, branch and HEAD were never
touched (`git status` after the review: `M LOGBOOK.md`, `M README.md`,
`?? internal/axpane/` only). `PYTHONDONTWRITEBYTECODE=1` everywhere; 0
`__pycache__` in any copy. Every instrument, raw log and subprocess exit is
in `TASK-260830-1geqhj_review-evidence-rev4.tar.gz` (paths below are relative
to that archive).

## 0. Rework verification — the rev3 findings, graded on my own executed evidence

The rev3 reviewer's instruments (`probes/zz_review3_probe_test.go`, 21 tests,
and `probes/zz_review3_delta_test.go`, 2 tests) were re-run against the NEW
tree with only the `Supersede` call sites adapted to the new
`(Binding, bool, error)` signature (five `_,` insertions; no assertion
changed): **23 of 23 pass** (`logs/10-rev3-probes-rerun.log`). The rev2
instruments (`probes/zz_review2_probe_test.go`, 29 tests) were re-run the same
way: 23 pass and the same 6 fail that failed for the rev3 reviewer, for the
same rev2-harness reasons that reviewer already graded (a successor lease
installed without publishing its checkpoint, a chain the landed reducer
refuses, `Status` expected to move to a superseded pair) — each is re-driven
by a rev3/rev4 instrument (`logs/11-rev2-probes-rerun.log`). A finding is
"fixed" only when the reported vector behaves per the spec AND a vector one
step away does too.

| rev3 finding | Grade | Executed evidence (exact tree) |
| --- | --- | --- |
| **P1-1 (site 1)** strict `Winner.Checkpoint == NewestCheckpointID` parked the ordinary post-takeover lifecycle | **Fixed** | `leaseCheckpointAgrees` and both `*LeaseDisagree` causes are deleted (rev3→rev4 production delta, `logs/80`); the only production read of the lease record's checkpoint is `decide.go:455` — the implication arm `HasWinner && Winner.HasCheckpoint && !HasNewestCheckpoint → parked(restore_policy)` placed directly after `authorize`, i.e. on every launch-class path, launch and restore alike, before the journal and checkpoint arms. Rev3 instruments: `TestN3PostTakeoverLifecycleRestoreLaunches` (Run, real stores: successor `(2,S)` base C0, fold newest C1, journal from C1, checkpoint C1 → **launch**) and `TestN3PostTakeoverLifecycleDecide` pass. One step away (`TestR4TwoCheckpointsAfterTakeover`, `logs/12`): the successor owner publishes C1 **and** C2 — restore from C2 launches with C2's closure (`yolo`, sourced), restore from the owner's own superseded C1 parks `restore_policy`, restore from the base C0 parks, no binding installed on either park. `TestR4LeaseBaseNeverPublishedStillLaunchesFromNewest`: a successor base the chain never published, fold newest C1 → launch — nothing else about the base is asserted (the stated takeover-leaf bound). `TestR4SuccessorNullFoldRestoreModeNoRequirements`: restore mode, no journal, no checkpoint, successor over a null fold → parked with the implication cause. Reviewer mutants RV4-2 (arm moved BEFORE fencing: a remote checkpoint-carrying winner over a null local fold parks instead of offering) and RV4-5 (arm admits exactly the epoch-2 successor) KILLED ×2. |
| **P1-1 (site 2)** `Run` derived the profile closure from `Winner.Checkpoint` | **Fixed** | `run.go:219-239`: closure from the required checkpoint when one is required, else the fold's newest loaded through `sessckpt` by ID (absent → error, never the head), else the session head; `Winner.Checkpoint` is read nowhere in `run.go`. Rev3 instruments `TestN3PostTakeoverLaunchModeProfileFromHandoffBase` (standard→yolo) and `TestN3PostTakeoverLaunchModeYoloFromStaleBase` (yolo→standard, the dangerous direction) pass; the committed `TestRunPostTakeoverLaunchCarriesNewestClosure{StandardToYolo,YoloToStandard}` and `TestRunPostTakeoverRestoreLaunchesWithNewestClosure` drive both directions and the restore positive through `Run` with real stores. One step away (`TestR4HeadOnlyChangeAfterNewestNeverLaunches`): a `profile.changed → yolo` appended AFTER the checkpointed stop (inert for the fold) never becomes the launch profile through `Run` in launch mode or restore mode (`standard`, mapping empty). Shipped `N-closure-source` (reads `Winner.Checkpoint` again) KILLED ×2 by `TestRunPostTakeoverLaunchCarriesNewestClosureYoloToStandard` (`logs/40`, `41`). Residual: **P3-1** (an unpinned member of the class: the journal-only restore path) and **P3-2** (the pure core with no heads supplied). |
| **P2-1** post-window same-pair race through `Run` reported `launch` with the winner's child | **Fixed** | `Supersede`/`installPair` return the replay signal (found before install, or link `EEXIST` re-read), `run.go:292-301` flips the outcome to `reattach`. Rev3 instrument `TestN3PostWindowSamePairRaceThroughRun` passes (loser holds the winner's 3333 receipt AND reports `reattach`). One step away: `TestR4PostWindowDifferentPairRaceStillLaunches` (a concurrent DIFFERENT pair between stage and link: our op2 still launches its own child, all three receipts on disk) and `TestR4SupersedeFoundBeforeInstallReportsReplay` (the pre-install `Lookup`-found path reports replay). Mutants RV4-7 (flip narrowed to an equal instance — never true in a real race), RV4-8 (found path reports a fresh install) and RV4-11 (in-window `Bind` flip narrowed to restore) KILLED ×2. |
| P3-1 window closes on the lease checkpoint over a null fold (RV3-4b survived) | **Fixed** | `TestDX_WindowLeaseCheckpointNullFold` (rev3 delta witness) → `refused idempotency_mismatch`; committed `TestRunWindowLeaseCheckpointNullFoldRefuses`; shipped row `N-window-lease-checkpoint` KILLED ×2. |
| P3-2 `Run` swallowing the fold error (RV3-11 survived) | **Fixed** | `TestDX_UnfolderableChainThroughRun` → error, no decision; committed `TestRunUnfoldableChainFails`; shipped row `N-run-fold-error` KILLED ×2. |
| P3-3 ratio 43/43 vs measured 39/43 | **Fixed** | Re-derived myself: 43 data rows counted in `TRACEABILITY.md`, every named test and subtest present in my own verbose run (`logs/82-ratio-check.log`, `logs/05`). The four rows the rev3 reviewer measured unfaithful now pin the post-takeover positives on the fixed tree (§2 below). **I measure 43 of 43.** |
| P3-4 fabricated 27-command list | **Fixed** | `results-rev4.md` quotes the 27 configured `spawn.worktree_isolation.validation.commands` verbatim (compared against `task-board.config.json`, equal) with real exits; `cmd/cmd01..27` logs are in the producer archive; the CR construction's own suite is `required=27 green=27 failed=0`. |
| P3-5 phantom matrix test names | **Fixed** | Every test named in `conformance-matrix-rev4.md` (66) and `results-rev4.md` (17) ran in my verbose run (`logs/82`). |
| P3-6 `realmRowUsable` restates reconcile | **Bound stated** | Stated as a classification-only, fail-closed-both-directions bound in the code comment and TRACEABILITY with the reason (the aggregate `Reconcile` error carries no per-object identity). Accepted as a bound; the three realm-row tests are unchanged and green. |
| P3-7 identity record not bound to SESSION_ID | **Fixed** | `checkIdentitySession` binds the record's own `session_id` after `VerifyIdentityBuild`; rev3 plant `identity_record_for_another_session` now `refused invalid_config`; committed `TestDecideIdentityForeignSessionRefuses`; shipped `N-identity-session` and reviewer RV4-1 (result ignored) KILLED ×2; neighbour `TestR4IdentitySessionBindingNeighbours` (same session, version drift → landed refusal). |
| P3-8 prose the fix invalidated | **Fixed** | README sentence rewritten ("from the checkpoint actually resumed — the required checkpoint when one is required, else the fold's newest, never the winning lease's handoff base"); TRACEABILITY rows carry no "agreeing" clause (`grep`: 0 hits); the rev3 LOGBOOK entry keeps its original sentence (append-only log) and carries an explicit `REVERSAL (P1-1, stated explicitly)` paragraph; the rev4 entry is newest-first and additive (`numstat` +37/−0). |

## 1. What holds (verified myself on the exact tree)

| Check | Result | Evidence |
| --- | --- | --- |
| gofmt / go vet / go build; GOOS=linux and GOOS=windows builds; GOOS=windows vet | all exit 0 | `logs/01`, `logs/70` |
| tracecheck (registry untouched: contracts=64, sections=36, clauses 49/569) / `cataloggen -adopted … -check` | exit 0 / exit 0 | `logs/70` |
| `go test ./internal/axpane -count=1 -v` | exit 0, 101 top-level tests, 192 RUN lines, 0 FAIL | `logs/05` |
| axpane `-race`, axpane `-cover` (80.9%), six packages (axpane, fencing, sessprofile, matjournal, sessckpt, provhost) | all `ok` | `logs/71`, `72`, `73` |
| Shipped harness, isolated pristine copy, two full passes with raw per-plant output | 53/53 both passes (52 KILLED + control SURVIVED), verdict lists identical; production blob OIDs and the tree OID equal the candidate afterwards | `logs/40`, `41`, `42` |
| Reviewer mutants (whole committed suite as killer, two passes, raw per-plant logs) | 10 KILLED ×2 (RV4-1 gate result ignored, RV4-2 implication arm before fencing, RV4-3 park reason collapsed for `failed_handoff`, RV4-4 window check dropped, RV4-5 arm admits epoch 2, RV4-7, RV4-8, RV4-9 Decide closure fallback disabled, RV4-10 step-5 park narrowed, RV4-11); control SURVIVED ×2; **RV4-6 SURVIVED ×2 with a non-equivalent witness (P3-1)**; RV4-12 and RV4-13 SURVIVED ×2 as the declared equivalence probes behind P3-5 and P3-4 | `logs/50`, `51`, `logs/mutants-pass{1,2}/*.log`, `logs/53-*`, `logs/54-*` |
| Real-kill crash evidence: `TestBindCrashChildSelfTerminates` + hook and concurrency seams ×2; my own real SIGKILL at the Bind anchor seam (anchor committed, pair receipt absent, retry converges to the one child, inode census: no rewrite, exactly one receipt) and at the Supersede anchor-claim seam (pair committed, anchor absent, retry replays and converges the anchor), ×2; rev3's `TestR3SupersedeRealKillSeams` ×2 | all PASS | `logs/60`, `TestR4BindAnchorSeamRealKill`, `TestR4SupersedeAnchorClaimSeamRealKill` in `logs/12` |
| Spec fidelity (§4.1/§4.2/§4.C/§4.D/§5.2/§7.A/§13.1 steps 3-4; v0.7.0 headings byte-identical to v0.6.0 in every pinned section except the §13.1 launch-plan insertion that precedes step 2) | after-restore order latest lease → `Verified` (caller-reported refresh, stated bound) → winning lease AND valid required materialization → interactive remote owner offers attach/takeover → parked otherwise: `TestDecideAfterRestoreSequence`, `TestDecideRemoteOwnerPrecedesMaterialization`, my `TestR4AfterRestoreStepThreeBothHalves` (stale epoch-1 token under a local successor with a valid journal parks `stale_owner`; winning lease with a prepared-not-committed journal parks `restore_policy`, no binding). Park vocabulary closed (`ParkedPayload`, `TestParkedPayloadVocabulary`, RV4-3). Background-caller rule: `TestR4BackgroundCallerCachedSmokeNeverAuthorizes` (cached passing smoke, no realm row → `capability_unavailable` with typed `RealmFailure`; realm row for another build → the same). Cached sentinel/managername structurally absent: `Realm` carries `Caller`, `BrokerState`, `ServerGeneration`, `Remediation` only; 0 production identifiers named sentinel/managername/attested. Idempotency: in-window changed op → `idempotency_mismatch` in both modes, identical retry replays, lost result → `Status` + `Lookup`. PID: `pid`/`tmux`/`os/exec`/`syscall` appear in production only as the typed `TmuxServerGeneration` detail; the receipt is the closed seven-member shape (`TestBindingPersistsNoPID`). | `logs/12`, `logs/80` |
| Composition, not re-implementation | every arm calls the landed gate (`fencing.Authorize*`, `matjournal.Store.Get`, `sessckpt.Get` + `sessrepo.AttestCheckpointRecord`, `sessprofile.Derive/DeriveForHeads`, `provhost.CheckResumeTuple/VerifyIdentityBuild/VerifyIdentityDiscovery/ResolveMapping`, `resumesmoke.VerifyRecord`, `terminalbackend.Reconcile/CheckOperation/CheckEntrypoint/AdmitProviderDescriptor`, `sessstate.Reduce`, `config.Load`, `axerror.New*`); the rev3→rev4 production delta adds only the implication arm, the closure-source rewrite, the replay signal and the identity session read (`logs/80`). Bypass plants: rev3's nine composition plants re-run (`TestN3CompositionBypassPlants`), plus mine: probe carrying an unregistered capability → refused at reconcile; descriptor with a foreign binding digest → refused; remote interactive owner unverified/ambiguous → parked `restore_policy`, never an offer; remote successor over a null local fold → `attach_remote` (fencing precedes the implication arm); required non-newest checkpoint in launch mode → parked; inconsistent fold fact (`has=true`, empty id) → parked; same-epoch loser → `stale_owner` with no token. Duplicated predicates: only the fail-fast guards already justified and the `realmRowUsable` bound. | `logs/12` |
| Decision table | 41 rows + rev3's six plants + my eight (`TestR4DecisionPlants/*`, listed above) — exactly one outcome per row | `logs/12` |
| Events | `session.parked` authored only through `Emit → fencing.AuthorizeMutation → sessrepo.AppendEvent`; a losing tuple refuses (`TestEmitUnderLosingLeaseRefuses`, rev3 `TestN3EmitUnderLosingTupleRefusesAfterRunPark`); my `TestR4EmitUnderRemoteWinnerRefuses` (presented equals the winner but the local host is not the holder → `not_owner`, event count unchanged); v4 payload shapes pinned (`TestEmitTerminalV4RoundTrip`, `TestR4ResumedPayloadRefusesNonPair`). Residual: P3-4. | `logs/12`, `13` |
| Race gate `go test ./internal/sessquery -race -count=1 -timeout 25m` | **exit 0, `ok` in 406 s** (408 s wall) on a loaded host (load ≈10 throughout; my probes and both harness passes overlapped it). Import census: axpane→sessquery and sessquery→axpane 0/0 including test deps; the candidate touches 0 sessquery bytes. Judgement: the earlier 10 m failures were a host-capacity artifact of the pre-PR51 default, not a regression; with the configured 25 m timeout the gate is green (the producer's own cmd05 log shows sessquery `ok` 481 s, axpane 17.5 s, 38/38 packages). | `logs/30`, `logs/80` |
| Bounds and hygiene | changed paths exactly the 24 candidate paths (23 axpane + README + LOGBOOK); `internal/traceability` untouched; `task-board.config.json` equal to base; 0 `__pycache__`/`.pyc` in the tree; README +65/−0 with test and harness commands and "no `ax` command, no `doctor` result, and no runtime capability claim"; LOGBOOK +37/−0 newest-first; no `os/exec`, no process control, no provider process, no mesh transport in production | `logs/80` |

## 2. Coverage statement

Producer claim: 43 of 43 AC rows driven. Denominator re-derived by counting
the table: 43. Every named test and subtest exists and ran (`logs/82`).
Measured: **43 of 43**. The four rows the rev3 review measured unfaithful are
now driven at the production entry on the fixed tree: "Materialization
validity gates local resume" and "Checkpoint admission gates resume" drive the
post-takeover launch from the fold's newest and the stale-base park through
`Run`; "After-restore sequence 1-5" drives both owners; "Effective profile
derived with source" drives both directions in launch mode plus the restore
positive through `Run`. One member of that last row's class is unpinned
(P3-1) — the row is driven, the class is not closed.

## 3. Findings

No P1. No P2.

### P3 (record for the Story's follow-up leaves; none blocks acceptance)

- **P3-1 Unpinned member of the closure-source class: the journal-only restore path.**
  RV4-6 (`run.go`: read `observation.Winner.Checkpoint` instead of the fold's
  newest on exactly `Mode == ModeRestore && Winner.HasCheckpoint`, i.e. a
  restore with the journal required and NO required checkpoint) SURVIVED the
  whole committed suite twice. Witness `TestDX4_JournalOnlyRestoreClosureSource`
  (`logs/53-*`): the yolo→standard post-takeover world, restore with
  `MaterializationRequired` (journal from C1) and `CheckpointRequired=false`
  — pristine launches `standard`/`toStandard`, mapping empty; under the
  mutant it launches **`yolo` with `--dangerously-bypass-approvals-and-sandbox`**
  while `TestRunRestoreCommitted` and
  `TestRunPostTakeoverRestoreLaunchesWithNewestClosure` stay green. The
  production code is correct; the suite cannot see a path-scoped regression of
  the exact rev3 P1 vector. Add that negative through `Run` and a harness row.
- **P3-2 The pure core is not fail-closed on a missing closure.** `Decide`
  with `HasNewestCheckpoint=true`, no `CheckpointRequired` and empty
  `ProfileHeads` derives the SESSION HEAD: `TestR4HeadOnlyChangeAfterNewestNeverLaunches`
  (`logs/12`) — post-takeover world, C1 closure `standard`, a head-only
  `profile.changed → yolo` after the stop; `Run` launches `standard` in both
  modes (correct), `Decide` alone launches **`yolo` with the bypass mapping**.
  `Run` always supplies heads when the fold has a newest (an attested checkpoint
  carries 1..64 heads), so no `Run` path reaches this; the `Input` doc only
  says "I/O-free callers supply them by hand". Either refuse in `Decide` when
  the fold names a newest and no closure was supplied, or state the bound in
  TRACEABILITY with the same words (the later leaves are the I/O-free callers).
- **P3-3 A remote-owned session whose newest checkpoint is absent from the
  LOCAL checkpoint store (or with no store bound) fails the run instead of
  reaching attach/takeover/park.** `TestR4RemoteOwnerNewestAbsentLocally` and
  `TestR4RemoteOwnerNoCkptStore` (`logs/12`): successor `(2,S)` on the remote
  host, chain folded locally, newest absent from the local `sessckpt` store →
  `Run` errors "newest absent from the store" / "no checkpoint store" with no
  decision, where `Decide` would return `attach_remote`. Fail-closed (no
  launch), but the after-restore step 4/5 outcomes become an error on a replica
  host with checkpoint lag; the closure is only needed on the launch path
  (local winner). Load the closure only for a local winner / defer it to the
  launch-class decision, or state the bound.
- **P3-4 `EmitParked` does not bind the payload's `winning_lease_id` to the
  authoring lease.** `TestR4EmitParkedWinningLeaseMismatch` (`logs/13`): a
  `session.parked` event was appended under lease A with
  `payload.winning_lease_id = B`; RV4-13 (payload winner taken from
  `params.LeaseID`) SURVIVED ×2, so nothing distinguishes the decision's winner
  from the authoring lease. Through `Run` both come from `observation.Winner`
  (consistent); a direct caller can author a lying payload. Since
  `AuthorizeMutation` makes the authoring lease the winner, require
  `decision.WinningLeaseID == params.LeaseID` in `EmitParked` and pin it.
- **P3-5 Unpinned absence-vs-failure arm at `LoadCheckpoint`.** RV4-12 (any
  `sessckpt.Get` error laundered as absence) SURVIVED ×2. Witness
  `TestDX4_CorruptCheckpointBlobIsNotAbsence` (`logs/54-*`): a stored
  checkpoint blob corrupted in place (attestation failure, not ENOENT) —
  pristine: `Run` fails with the load error; under the mutant: "newest absent
  from the store". Both fail closed (no decision), the delta is the fact
  reported — the same class as rev3's P3-2. Add the corrupt-blob negative and a
  harness row.
- **P3-6 Reattach outcomes carry two terminal instance IDs.**
  `TestR4ReattachDescriptorNamesWhichInstance` (`logs/12`): on `reattach`,
  `Outcome.Binding.TerminalInstanceID` is the recorded child (1111) while
  `Decision.Descriptor.TerminalInstanceID` is the request's own/freshly minted
  instance (2222), built before the decision. `Decision.Descriptor` is
  documented as "set on launch and reattach only". Document that `Binding` is
  authoritative on reattach (or clear/rebuild the descriptor from the recorded
  child) so no caller targets a child that does not exist. Present since rev1;
  hygiene.
- **P3-7 Evidence archive hygiene.** `TASK-260830-1geqhj_rev4-evidence.tar.gz`
  is a tar of a shared scratch directory: its `README.md` is
  "TASK-260830-24z2b3 producer evidence (rev4)"; it carries
  `TASK-260830-24z2b3_results.md`/`_conformance-matrix.md`, a
  `TASK-260916-fsw7re` `logs-summary.txt`, `__pycache__/*.pyc` (3 files,
  listed in `MANIFEST.sha256`), `review-evidence-rev4.zip`,
  `mutants/run1|run2` (66 rows of a foreign task) and `mutants-run{1,2}.log`.
  The axpane evidence inside is genuine and matches my reruns
  (`mutants/pass1/` 54 raw logs with exits, `pass2-summary.log`,
  `axpane-test-v.log`, `ratio-check.log`, `cmd/cmd01..27`), but the archive
  breaks the task-scoped / no-`__pycache__` contract and pass 2 has only a
  summary, no raw per-plant logs. Re-pack task-scoped next time. (The
  candidate tree itself carries 0 `__pycache__`/`.pyc`.)

### Characterizations (not findings)

- A remote takeover whose base the local chain never published (null local
  fold) with a changed bootstrap operation refuses `idempotency_mismatch`
  before the fencing arm can park `remote_owner` (`TestR4RemoteTakeoverNullFoldChangedOp`).
  The world is inconsistent by construction (a takeover needs a validated
  checkpoint this host would have published) and the outcome is a refusal, not
  a launch.
- A descriptor whose binding digest drifts refuses with the landed
  `terminal_backend_not_found` class (pass-through of `AdmitProviderDescriptor`).

## 4. DoD walk (live merged checklist)

Production entry points implement the scoped deliverable — yes (`Decide`,
`Run`, `Store.*`, `Emit*`, `BuildDescriptor`). Positive/negative/compatibility/
recovery tests pass with logs — yes (`logs/05`, `60`, `71`, `72`). README/
traceability without unsupported claims — yes (no capability/CLI claim; the
"52 narrowing mutants plus one control" count matches the 53 harness rows).
Uncommitted in the Story worktree — yes (`git status`, tree OID equal).
Ratio — 43 of 43 with call sites (TRACEABILITY table, re-counted). Negative
tests that fail when the gate admits — yes (shipped harness 52 KILLED ×2,
reviewer mutants 10 KILLED ×2). Narrowing mutant per gate — yes; no
source-text gate exists (no token-preserving mutant applies). Lint/build —
clean. Outcome artifacts attached — yes (P3-7 hygiene noted). Logbook —
newest-first, additive, explicit reversal. Implementation matches AC, fits the
architecture (composition over the landed owners; delta is exactly the two
P1-1 sites, P2-1, P3-7 and evidence), tests green, gates attacked not read.

## 5. Instruments (all in the evidence archive)

`probes/zz_review4_probe_test.go` (23 probes, 25 top-level with the two delta
witnesses: 19 PASS; the 5 CHARACTERIZATION failures are P3-2, P3-3 ×2, P3-4,
P3-6; `TestR4LosingLeaseProfileChangeAfterNewestIgnored` fails in its fixture
because the landed `sessrepo` refuses the divergent-branch append before the
probe runs — fail-closed by the owner, not a finding; `logs/12`),
`probes/zz_review4_delta_test.go` (2 delta witnesses), the adapted rev3/rev2
probe files, `review4_mutants.py` (14 rows, whole-suite killer), `logs/01`,
`05`, `10`–`13` (probe runs), `30` (race gate), `40`–`42` (shipped harness ×2 +
blob OIDs), `50`–`54` (reviewer mutants ×2, raw per-plant logs, delta
witnesses pristine/under mutant), `60` (crash reruns), `70`–`73` (tree-exact
checks, six packages, race, cover), `80`–`82` (hygiene, name census, ratio),
`REVIEW-MANIFEST.txt` (sha256 of every file).

Verdict recorded by RUN-260917-e644e0; no product edits, commits, checkpoints
or integration were performed; the isolated copies are deleted after
attachment. `commit_ack` is not supplied; the `done` transition belongs to the
producer-side integration run.
