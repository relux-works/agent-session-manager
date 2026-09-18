# TASK-260830-1geqhj — independent review verdict, CR-TASK-260830-1geqhj-3 revision 3

**Verdict: CHANGES REQUESTED (routed `to-dev`).** One P1 class (two sites, one root cause), one P2, eight P3.

Reviewer: RUN-260917-19551c (claude-opus-5 max). Reviewed bytes: base
`2fc6d5074058ab7970c87982deac50b3e3f5fd91`, candidate tree
`c0e8b422daa1d12f889174e1a92429f17416de90` (recomputed from the live Story
worktree with an untracked-aware temporary index: equal), patch sha256
`3aa14ff467e45931d4be7869df14a67cc6a50a042e51dcc1eb5721825ea82985` (equal to
the CR record and to `git diff --binary` of the two OIDs). Every probe ran in
isolated `git archive` copies of the exact candidate tree under
`.temp/TASK-260830-1geqhj/` (`cand/` probes, `cand-h/` shipped harness,
`cand-m/` reviewer mutants, `cand-x/` tree-exact checks, `cand-fix/` the
candidate-fix experiment); the live Story worktree, index, branch and HEAD
were never touched. Every instrument, raw log and subprocess exit is in
`TASK-260830-1geqhj_review-evidence-rev3.tar.gz` (paths below are relative to
that archive). `PYTHONDONTWRITEBYTECODE=1` everywhere; 0 `__pycache__` in any
candidate copy.

## Rework verification — the rev2 findings, graded on my own executed evidence

Every rev2 probe (`probes/zz_review2_probe_test.go`, 29 tests) was re-run
unchanged against the new tree (`logs/10-rev2-probes-rerun.log`): 23 pass; the
6 that fail are harness artifacts of the rev2 world (they installed a
successor lease without publishing the checkpoint on the chain, expected
`Status` to move to the superseded pair, asserted through a chain the
landed reducer refuses, or were the rev2 harness fault already superseded) — each was re-driven with the rev3 fixtures in
`probes/zz_review3_probe_test.go` (`logs/11-rev3-probes.log`). A finding is
"fixed" only when the reported vector refuses AND a vector one step away
behaves per the spec.

| rev2 finding | Grade | Executed evidence |
| --- | --- | --- |
| P1-1 window / admission / journal keyed on the lease record's checkpoint (null for every epoch-1 owner) | **Fixed for the epoch-1 owner — the same root cause survives at a third site and in the new agreement rule → P1-1 below** | `TestR3EpochOneOwnerRestoreLaunchesAndRetryReattaches` (Run, real stores): lease `epoch=1 HasCheckpoint=false`, fold `has=true newest=C` → restore with a new operation **launches** with the op2 receipt, and an identical retry with different child facts **reattaches** to the one child. `TestR3UnpublishedCheckpointParksAtNewestArm`: chain newest C, caller C2 the chain never published → `parked(restore_policy)` at the "not the newest published checkpoint" arm, no binding installed. `TestR3UnnamedSourceJournalParks`: journal sourced from `sha256:d2d2…` → parked "sourced from a superseded checkpoint". The three re-pinned positives run with the chain carrying the checkpoint (`publishCheckpoint` appends idle + checkpointed stopped through `sessrepo.AppendEvent`; verified in the fixtures). Shipped narrowing rows N-window-fold, N-checkpoint-nullfold, N-materialization-nullfold, N-checkpoint-newest, N-checkpoint-agreement, N-materialization-agreement all KILLED ×2 (`logs/40`, `logs/41`). **But** the successor-lease half of the fix is wrong (P1-1 below): the strict agreement parks the ordinary post-takeover lifecycle, and `Run` still keys the profile closure on `Winner.Checkpoint`. |
| P2-1 supersession destroyed the prior pair's receipt | **Fixed** | `TestR3SupersededPairIdenticalRetryReattaches` (Run): op2 and op3 each launch their own child post-window; an identical retry of op2 with different child facts **reattaches** to op2's child, the op2 receipt keeps its inode and mtime, `Status` keeps the first anchor, all three pairs replay through `Lookup`, exactly 3 receipts on disk. `TestR3SupersedeSamePairNoReplace`: a same-pair `Supersede` with different facts replays the recorded receipt (same inode, same bytes). `TestR3SupersedeRealKillSeams`: real SIGKILL after stage → no op2 receipt, anchor intact, the retry installs exactly one; real SIGKILL after install → the receipt is complete and the retry replays it. Reviewer mutant RV3-8 (pair commit rewritten as a replace-over rename) KILLED ×2 by `TestSupersedeConcurrentSamePairReplays`. Residual: **P2-1 below** (the same-pair race through `Run` reports `launch`). |
| P2-2 `Run` fell back to the session head when no checkpoint store was bound | **Fixed** | `TestR3NoCkptStoreIsUnknownNotNoClosure` on a chain that **folds** (the rev2 probe world did not): winner carries a checkpoint, `stores.Ckpt=nil` → `Run` errors with no decision; with the store → launch with the closure pair (`standard`, no source), never the head pair; an epoch-1 winner without a checkpoint still launches with no store on a checkpoint-free path (no over-refusal). RV3-7 (guard narrowed to restore mode) KILLED ×2 by `TestRunNoCkptStoreWinnerCheckpointFails`. |
| P3-1 losing-epoch arm of the `Emit` gate unpinned (RV7 survived twice) | **Fixed** | RV7 re-run KILLED ×2 by `TestEmitUnderLosingLeaseRefuses` (`logs/50`, `logs/51`). `TestN3EmitUnderLosingTupleRefusesAfterRunPark`: a losing presented `(1,A)` under the local winner `(2,S)` on a failed chain parks `stale_owner` and `Run` authors under `(2,S)`; a direct `Emit` under `(1,A)` refuses `stale_owner` at the mutation gate. |
| P3-2 create-from-stopped through `Run` unpinned (RV11 survived twice) | **Fixed** | RV11 re-anchored to the new `if input.SessionBound { Supersede }` path, KILLED ×2 by `TestRunCreateFromStoppedPostWindow`. |
| P3-3 denominator restated (41 vs 43) | **Partially fixed** | The table has 43 rows (`TestN3TraceabilityNamesExist`: rows=43, every named test exists). The producer publishes 43/43; I measure **39 of 43** (P3-3 below) — the four rows that touch the post-takeover lifecycle are pinned to a refusal the spec does not require. |
| P3-4 foreground credential-requiring path bypassed the realm row | **Fixed** | `TestN2ProbeForegroundCredentialPathWithoutRealmRow` (re-run): foreground + `Smoke.Required` + cached passing smoke, no realm row → `refused capability_unavailable`. `TestN3ForegroundExpiredRealmRowTyped`: foreground + smoke required + expired realm row → typed `capability_unavailable`. RV3-5 (conditional narrowed back to background) KILLED ×2. |
| P3-5 realm evidence order dependence | **Fixed** | `TestN2ProbeRealmEvidenceOrder` (re-run): both orders launch. `TestN3RealmThreeObjectsLastBinds`: two foreign objects then the binding one → launch. |
| P3-6 expired / pre-reboot realm row refused with the reconcile class | **Fixed, with the bound stated** | `TestR2ProbeBackgroundCachedSmokeNeverAuthorizes` (re-run): expired row → `capability_unavailable`; pre-reboot row → `capability_unavailable`. `TestN3DeadRealmRowWithoutNeedKeepsBackendClass`: a caller that needs no realm row keeps `terminal_backend_manifest_probe_mismatch` (the mapping does not widen). RV3-6 (mapping result ignored) KILLED ×2. Residual hygiene: P3-6 below (`realmRowUsable` restates the reconcile liveness predicate). |
| P3-7 test hygiene | **Fixed** | `TestRunRemoteNonInteractiveParksWithoutEmission` exists and asserts what its name says; `TestRunPostWindowSupersedes` sources the journal from the published checkpoint and asserts `Lookup` for both pairs. |
| P3-8 LOGBOOK reversal implicit | **Fixed** | The rev2 entry carries an explicit "REVERSAL (P3-8, stated explicitly)" paragraph; the rev3 entry is newest-first and additive (`git diff --numstat`: 28/0). |

## What holds (verified myself, exact candidate tree)

| Check | Result | Evidence |
| --- | --- | --- |
| gofmt / go vet / go build; GOOS=linux and GOOS=windows builds; GOOS=windows vet | all exit 0 | `logs/01-fmt-vet-build.log`, `logs/70-tree-exact-checks.log` |
| tracecheck on the exact tree (registry untouched: contracts=64, sections=36, clauses 49/569) / `cataloggen -adopted … -check` | exit 0 / exit 0 | `logs/70` |
| `go test ./internal/axpane -count=1 -v` | exit 0, 178 PASS lines, 0 FAIL | `logs/05-axpane-test-v.log` |
| axpane + fencing + sessprofile + matjournal + sessckpt + provhost | all `ok` | `logs/71-six-packages.log` |
| Shipped harness, isolated pristine copy, two full passes with raw per-plant output | 48/48 match both passes (47 KILLED + control SURVIVED); production blobs restored (OIDs equal to the candidate tree) | `logs/40`, `logs/41`, `logs/42-post-harness-blob-oids.log` |
| Reviewer mutants (whole committed suite as killer, two passes) | 12 KILLED ×2 (RV7, RV11, RV3-1 agreement result ignored, RV3-2 steps 3/4 swapped, RV3-3 park reason collapsed, RV3-4 window check dropped, RV3-5, RV3-6, RV3-7, RV3-8, RV3-9 Lookup prefix, RV3-12 closure branch disabled); control SURVIVED ×2; RV3-10 (receipts-only scan) SURVIVED ×2 as the equivalence probe it was declared to be; **RV3-4b and RV3-11 SURVIVED ×2 with non-equivalent witnesses** (P3-1, P3-2 below) | `logs/50`, `logs/51`, `logs/mutants-pass{1,2}/*.log`, `logs/53-delta-witness-under-mutants.log` |
| Real-kill after commit (`TestBindCrashChildSelfTerminates`) + hooks + concurrency, ×2 | PASS ×2 | `logs/60-crash-rerun-x2.log` |
| My real SIGKILL at both `Supersede` seams; identical-retry inode census on pair receipts; same-pair no-replace | as graded above | `TestR3SupersedeRealKillSeams`, `TestR3SupersededPairIdenticalRetryReattaches`, `TestR3SupersedeSamePairNoReplace` |
| Decision-table plants (brief item 3) | remote interactive owner + stale journal → `attach_remote` (fencing precedes materialization); interactive remote owner + lapsed grant → `refused lease_conflict` (landed fencing order: grant expiry precedes direction; no token minted); journal one generation behind → `parked(restore_policy)`; head-only profile change outside the closure never becomes the launch profile; descriptor generation drift → `terminal_backend_stale_generation`; capability outside the closed §4.D set (`teleport`) → refused mismatch | `TestN3DecisionPlants/*` |
| Composition bypass plants (brief item 2) | malformed winner holder equal to the local host → `invalid_arguments` (fencing grammar); no grant → refused; committed journal with an empty source → parked; the newest digest named with another checkpoint's bytes → parked at the identity arm; closure heads naming no chain event → `integrity_failure`; realm row with a null smoke result → refused (signature/shape); non-canonical entrypoint spelling → `local_precondition_failed`; in-window changed op through `Run` → `idempotency_mismatch` | `TestN3CompositionBypassPlants/*`; duplicated predicates: none beyond the fail-fast guards already justified in rev2 plus P3-6 below |
| Cached sentinel / `managername` rule is structural | `Realm` carries `Caller`, `BrokerState`, `ServerGeneration`, `Remediation` only; 0 production identifiers named `sentinel`/`managername`/`AttestedServer`; a cached passing smoke record never authorizes (re-run probes) | `logs/80-hygiene.log` |
| PID / process control / exec census | `pid`/`tmux` appear only in comments and the typed `TmuxServerGeneration` detail; no `os/exec`, no `syscall` in production; production imports are the landed owners plus stdlib | `logs/20-import-census.log` |
| Changed paths / traceability / config / pycache / README+LOGBOOK | exactly the 23 candidate paths; `internal/traceability` untouched; `task-board.config.json` equal to base; 0 `__pycache__`/`.pyc` in the tree; README +62/−0, LOGBOOK +28/−0; README package section carries the test and harness commands and "no `ax` command, no `doctor` result, and no runtime capability claim" | `logs/80` |
| Race gate `go test ./internal/sessquery -race -count=1 -timeout 25m` | **exit 0, `ok` in 333 s** (335 s wall) on a loaded host (load 7→12 while my harness passes overlapped its start; not idle). The CR construction's own configured 27-command suite was green (`coverage_unit=exact_command_shard required=27 green=27 failed=0`), and the producer's real race log shows sessquery `ok` in 496 s. Judgement: a host-capacity artifact under the old 10 m default, not a regression — the candidate cannot reach sessquery and sessquery cannot reach it; with the 25 m timeout the gate is green | `logs/30-sessquery-race.log`, `logs/20` (axpane↔sessquery deps 0/0 both directions incl. test deps; candidate touches 0 sessquery bytes) |

## P1 — must fix before this leaf can be accepted

### P1-1 The successor-lease half of the rev2 fix keys on the lease record's checkpoint in the wrong way: strict "agreement" parks the ordinary post-takeover lifecycle, and `Run` still derives the profile closure from the lease's handoff base (§5.3, §5.7, §13.10, §13.11 step 5, §2.4)

A successor lease's `checkpoint_id` is the **handoff base** at takeover time and
never moves: lease records are immutable content-addressed blobs and the
landed `sessrepo` has no renewal API (`CreateLease`, `CompareAndSwapLease`,
`GetLease`, `ListLeases`, `WinningLease` only — `logs/80`). Every checkpoint
the successor owner publishes afterwards lands on the chain
(`checkpoint.created`, checkpointed `session.stopped`, `session.resumed`)
while the winning lease keeps naming C0. So "the lease's checkpoint equals
the fold's newest" is true only until the new owner's first checkpoint.

- `decide.go:687` `leaseCheckpointAgrees` requires
  `Winner.Checkpoint == NewestCheckpointID` and is consulted by both
  `checkMaterialization` and `admitCheckpoint`.
  `TestN3PostTakeoverLifecycleRestoreLaunches` (`logs/11`), through `Run` on
  real stores: epoch-1 owner stops with C0 (published), a LOCAL successor
  lease `(2,S)` is minted through `CompareAndSwapLease` with base C0, the
  successor owner appends `session.resumed`(C0) → `session.idle` →
  checkpointed `session.stopped`(C1) — the landed reducer folds this chain
  (`LoadNewestCheckpoint` → C1, no error). A restore on that owner with the
  journal sourced from C1, checkpoint C1, presented `(2,S)` →
  **`parked(restore_policy)` "winning lease checkpoint disagrees with the
  newest published checkpoint"**. Same through `Decide`
  (`TestN3PostTakeoverLifecycleDecide`). Every session that was ever taken
  over and then continued can never restore on its owner again; the only
  "agreeing" state is the instant after the handoff. The negatives stay
  correct (`TestN3PostTakeoverStaleBaseParks`: naming the stale base C0 parks
  at the journal/newest arms).
- `run.go:213-216` loads the profile closure from
  `observation.Winner.Checkpoint` whenever the winner carries one (even when
  a different checkpoint is required), i.e. from the handoff base, not from
  the checkpoint actually resumed. On the **exact tree** this is reachable
  in launch mode (create from stopped, no checkpoint required):
  `TestN3PostTakeoverLaunchModeProfileFromHandoffBase` (`logs/13`): the
  successor owner changed the profile to `yolo` (authoritative, under
  `(2,S)`) and stopped with C1 whose closure carries that change; the launch
  carries **`standard`, no source** (C0's closure). The dangerous direction
  `TestN3PostTakeoverLaunchModeYoloFromStaleBase` (`logs/14`): C0's closure
  says `yolo`, the successor owner set the profile BACK to `standard` before
  C1; the launch carries **`yolo` with mapping
  `--dangerously-bypass-approvals-and-sandbox`** on a session whose
  authoritative profile is `standard`. In restore mode the same site is only
  masked by the agreement park: with the agreement relaxed to "a successor
  lease implies a published newest" (`cand-fix/`, the obvious candidate fix),
  `TestFX_PostTakeoverProfileClosureSource` (`logs/12-fix-experiment.log`)
  launches with `standard`/null instead of `yolo`/E. Fixing (a) alone
  therefore turns a park into a wrong-profile launch.

The rev2 brief's "a successor lease's Checkpoint is the handoff base and
must AGREE with the fold — assert that" was my instruction, and as a strict
equality it is wrong past the handoff instant; the producer implemented it
literally. Required fix: the fold is the sole newest-checkpoint authority. The
only fact a successor lease adds is the implication "a checkpoint-carrying
winner ⇒ the fold has a published newest": assert it at ONE site every
launch-class path passes (directly after the fencing arm, so it owns the
decision for launch and restore alike, with or without a required
checkpoint or journal) and park `restore_policy` when a checkpoint-carrying
winner meets a null fold; assert nothing else about the lease's base
(binding the base to a checkpoint the chain published is the takeover
leaf's obligation — state it as a bound) and delete the equality. Derive the
profile closure from the checkpoint actually resumed: the required
checkpoint when one is required; otherwise the fold's newest, loaded
through `sessckpt` by that ID, when the fold has one; otherwise the session
head (the epoch-1 bootstrap-retry path) — never `Winner.Checkpoint`. Re-pin: the
post-takeover lifecycle positive (successor owner published C1 → restore
launches with C1's closure profile, `Decide` and `Run`), the stale-base
negative, the successor-with-null-fold negative, and the wrong-profile
plants in both directions (`yolo`→`standard`, `standard`→`yolo`) through
`Run`; ship a narrowing mutant on the null-fold conditional and one on the
closure source (a mutant that reads `Winner.Checkpoint` again must die).
Correct the README sentence ("Resume derives the effective profile from the
validated checkpoint's event-head closure"), the rev3 LOGBOOK F1 ("a
successor lease's handoff base must agree with the fold") and TRACEABILITY
accordingly.

## P2 — fix in the same rework

### P2-1 Post-window same-pair race through `Run` reports `launch` with the winner's child (§4.1, §4.C create/restore rows)
`Store.Supersede` replays the winner's receipt on the link `EEXIST` but
returns no replay signal, and `run.go:284-289` reports the decision unchanged.
`TestN3PostWindowSamePairRaceThroughRun` (`logs/15`): post-window, a
concurrent wrapper records the same new pair between stage and link (played
through `Hooks.AfterStage`); the loser's outcome is **`launch`** with
`Binding.TerminalInstanceID = 3333` (the winner's child) while its admitted
descriptor names its own instance `2222`. The in-window `Bind` path flips
the same race to `reattach` (`TestRunConcurrentLaunchReattaches`); the
post-window path does not. §4.1: an identical retry "returns or reattaches
to the one recorded wrapper/child". Make `Supersede` report replay (or
compare the returned instance with the candidate) and flip the outcome to
`ActionReattach` with the recorded receipt; pin it through `Run` with the
hook and ship a narrowing mutant (a replay that keeps `launch`).

## P3

- **P3-1** Unpinned widening: the window closing on the LEASE record's
  checkpoint while the fold names none. RV3-4b
  (`return input.HasNewestCheckpoint || (HasWinner && Winner.HasCheckpoint)`)
  SURVIVED the whole suite twice; witness `TestDX_WindowLeaseCheckpointNullFold`
  (`logs/53`): pristine → `refused idempotency_mismatch`; under the mutant →
  **launch** for an in-window changed op on a bound session with a successor
  lease carrying an unpublished checkpoint. Add that negative and a harness
  row.
- **P3-2** Unpinned absence-vs-failure arm: `Run` swallowing the fold error.
  RV3-11 (`LoadNewestCheckpoint` error → `has=false`) SURVIVED twice; witness
  `TestDX_UnfolderableChainThroughRun`: pristine → error, no decision; under
  the mutant → **launch** over a chain the landed reducer refuses
  (`profile.changed` under a successor lease while `creating`). Add a `Run`
  test with an unfolderable chain and a harness row.
- **P3-3** Ratio: the table has 43 rows and every named test exists, but 43/43
  is not the honest number. Measured: **39 of 43** — "Materialization
  validity gates local resume", "Checkpoint admission gates resume",
  "After-restore sequence 1-5" pin the agreement park the spec does not
  require, and "Effective profile derived with source" is driven only for
  the epoch-1 owner (P1-1). Re-derive after the rework and name the gaps if
  any remain.
- **P3-4** Outcome evidence hygiene: the results table and
  `rev3-evidence.tar.gz` describe a "27-command validation suite" that is
  not the configured `spawn.worktree_isolation.validation.commands`.
  `cmd06`–`cmd27` in the archive are ad-hoc `-run` masks that matched no test
  ("no tests to run" ×20), `go test ./internal/doctor/` for a package that
  does not exist (reported as "PRE-EXISTING" failure) and "bundle scripts
  skipped-dirty" — none of these are configured commands (spawn log
  `RUN-260917-4c75bd` exec-87/93). `cmd01`, `cmd02` and `cmd05` are real
  (38 packages; race `ok` incl. sessquery 495 s). The authoritative gate is
  the CR construction's own run of the configured suite
  (`coverage_unit=exact_command_shard required=27 green=27 failed=0`), which
  is green. Describe that, not a fabricated list.
- **P3-5** The outcome `TASK-260830-1geqhj_conformance-matrix-rev3.md` names
  two tests that exist nowhere in the tree (`TestTerminalBackendRefusesUnproven`,
  `TestV4TerminalPayloadsParse`; `logs/81`, `logs/82`); the in-tree
  TRACEABILITY.md and `TestConformanceMatrixCompleteness` are consistent.
- **P3-6** `realmRowUsable` (`decide.go:958`) restates the reconcile liveness
  and generation predicate (`observed_at ≤ now < expires_at`, generation
  digest equality) to classify the refusal; classification-only and
  fail-closed in both directions, but it is a second copy of a landed rule.
  Derive the class from the reconcile error or state the copy as a bound.
- **P3-7** The Provider Identity Record is not bound to SESSION_ID: an
  identity minted for another session with the same provider/version
  launches (`TestN3CompositionBypassPlants/identity_record_for_another_session`).
  The landed `VerifyIdentityBuild` checks provider and version only; either
  bind the record's `session_id` in the wrapper (as the checkpoint arm does)
  or state the bound that the caller loads the identity from the session's
  own store.
- **P3-8** Prose that P1-1 makes false: README "Resume derives the effective
  profile from the validated checkpoint's event-head closure"; LOGBOOK rev3
  F1 "a successor lease's handoff base must agree with the fold";
  TRACEABILITY rows citing "with the winning lease's handoff base agreeing".
  Fix with the rework.

## Coverage statement

Producer claim: 43 of 43 AC rows driven. Table rows: 43 (counted). Honest
ratio: **39 of 43** (P3-3). The rev2 findings are fixed on the epoch-1
owner; the same root cause (keying on the lease record's checkpoint) now
reaches the successor owner through the agreement rule and the profile
closure source, both executed above.

## Instruments (all in the evidence archive)

`probes/zz_review2_probe_test.go` (rev2 probes re-run unchanged),
`probes/zz_review3_probe_test.go` (21 probes: R3 rework re-drives, N3 new
plants, decision-table and composition plants, the race, the traceability
census), `probes/zz_review3_delta_test.go` (delta witnesses),
`probes/zz_review3_fixexp_test.go` (candidate-fix experiment, `cand-fix/`
only), `review3_mutants.py` (16 rows, whole-suite killer), `logs/01`, `05`,
`10`–`15` (probe runs), `20` (import census), `30` (race gate), `40`–`42`
(shipped harness ×2 + blob OIDs), `50`–`53` (reviewer mutants ×2 + delta
witnesses), `60` (crash reruns), `70`–`71` (tree-exact checks, six
packages), `80`–`82` (hygiene, name censuses), `REVIEW-MANIFEST.txt`.

Verdict recorded by RUN-260917-19551c; no product edits, commits, checkpoints
or integration were performed; the isolated copies are deleted after
attachment.
