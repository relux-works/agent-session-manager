# TASK-260830-1geqhj — independent review verdict, CR-TASK-260830-1geqhj-2 revision 2

**Verdict: CHANGES REQUESTED (routed `to-dev`).** One P1 class, two P2, eight P3.

Reviewer: RUN-260917-368f1f (claude-opus-5 max). Reviewed bytes: base
`888ae3dff6a55c29bfaf311957748d405ed1ab06`, candidate tree
`0a5c5e7818aa94404420a4f2904924fc90e76083` (recomputed from the live Story
worktree with an untracked-aware temporary index: equal), patch sha256
`3a6ab390af68440df8791d3949378ffb2dfd3627010aa24b693d8a34faec31ff` (equal to
the CR record). Every probe ran in an isolated `git archive` copy of the exact
candidate tree under `.temp/TASK-260830-1geqhj/review-rev2/cand/`; the live
Story worktree, index, branch and HEAD were never touched. Every instrument,
raw log and subprocess exit is in
`TASK-260830-1geqhj_review-evidence-rev2.tar.gz` (paths below are relative to
that archive). `PYTHONDONTWRITEBYTECODE=1` everywhere; 0 `__pycache__` left.

## Rework verification — the twelve rev1 findings, graded on my own executed evidence

A finding is "fixed" only when the reported vector refuses AND a vector one
step away refuses. Probe names are the tests in
`probes/zz_review2_probe_test.go` (`logs/10-review-probes.log`,
`logs/11-review-probes-b.log`); mutant rows are in `review2_mutants.py`
(`logs/50-*`, `logs/51-*`, raw per-plant logs in `logs/mutants-pass{1,2}/`).

| rev1 finding | Grade | Executed evidence |
| --- | --- | --- |
| P1-1 `session.parked` authored under a remote-held lease; no `AuthorizeMutation` caller; unfoldable lifecycle event | **Fixed** | `TestR2ProbeRemoteOwnerParkNoRemoteLeaseEvent`: remote winner → `parked(remote_owner)` / `attach_remote`, `Emitted=false`, event count 1→1, and the owner's own `(B, seq 1)` event then appends cleanly (the rev1 collision is gone). `TestR2ProbeUnverifiedRemoteHolderNoEvent`: unverified+remote → no event. `TestR2ProbeParkedFoldsInStateEngine`: creating chain parks with no emission and `sessstate.Reduce` folds `creating`. `TestR2ProbeLosingLeaseReachesMutationGate` (foldable `failed` chain): losing `(1,A)` under remote `(2,B)` → `not_owner`; under local successor `(2,A2)` → `stale_owner` — the mutation gate itself refuses, not the fold pre-check. `TestN2ProbeStaleLocalPresentedSameEpoch`: a losing same-epoch presented lease under the held local winner parks `stale_owner` and authors under lease A (the held winner) at seq 3; the chain folds to `parked` after one park and after a second one. `TestR2ProbeLosingLeaseProfileEventIgnored` (probe 15): losing-lease `profile.changed` never becomes the launch pair (`standard`, no source). Residual: P3-1 below (the losing-epoch arm of the gate is unpinned by the committed suite). |
| P1-2 bare `AttestedServer` boolean; smoke unbound to generation/OS | **Fixed** (background path) | `TestR2ProbeBackgroundCachedSmokeNeverAuthorizes`: background + cached passing smoke, no realm row → `refused capability_unavailable`; smoke not required, no row → `capability_unavailable`; admitted row + stale typed generation → `capability_unavailable`; admitted row + smoke target bound to a stale generation → `target_auth_missing`; admitted row expired at the admission instant → refused (`terminal_backend_manifest_probe_mismatch at evidence liveness`); pre-reboot row after the raw generation moved → refused (`terminal_backend_stale_generation`). Mutant RV1 (cross-bind verdict ignored) KILLED ×2 by `TestDecideRealmBindingMismatchRefuses`. The boolean no longer exists as an input (`Realm` has no such member; `grep AttestedServer` = 0). Residuals: P3-4 (foreground credential path), P3-5 (evidence order), P3-6 (expired-row refusal class). |
| P1-3 bootstrap window never closes; RM4 survived | **Fixed only for successor leases → P1-1** | `TestR2ProbePostWindowRestoreWithSourcedJournal`: after a local successor lease carrying the captured checkpoint, restore with a NEW operation and a journal sourced from that checkpoint → `launch`, binding = op2 (the producer's own `TestRunPostWindowSupersedes` drops the materialization requirement; this probe keeps it). In a fresh world (winner without checkpoint) the changed op → `refused idempotency_mismatch`. `TestN2ProbeWindowVsRemoteOwner`: a remote winner that closed the window + changed local op → `parked(remote_owner)`, not a mismatch. Mutants RV4 (window closes on any winner) KILLED ×2 by `TestDecideRestoreChangedOperationRefuses`/`TestDecideTable`/`TestRunIdempotencyMismatch`; RV6 (= reviewer RM4, idempotency arm skipped in restore mode) KILLED ×2 by `TestDecideRestoreChangedOperationRefuses`. **But** the window predicate reads `Winner.HasCheckpoint`, a lease-record fact that is null for every epoch-1 owner forever (see P1-1): `TestN2ProbeEpochOneOwnerAfterCheckpoint` — same-host session stopped with a published checkpoint (landed engine: `state=stopped HasCheckpoint=true newest=C`; lease: `epoch=1 HasCheckpoint=false`), restore with a new operation → **`refused idempotency_mismatch` "inside the bootstrap window"** — the rev1 P1-3 symptom on the creating host, unless a takeover happened. Residuals: **P1-1**, **P2-1** (supersession destroys the prior pair's receipt), P3-2 (create-from-stopped through `Run` unpinned). |
| P1-4 checkpoint admission unbound to lease and session | **Partially fixed → P1-1 below** | Reported vectors refuse: `TestR2ProbeCheckpointAdmissionBound`: caller checkpoint ≠ lease checkpoint → `parked(restore_policy)` "required checkpoint is not the winning lease checkpoint"; a foreign-session checkpoint → "checkpoint names another session", even when the lease itself names that foreign digest. Mutant RV9 (lease binding narrowed to launch mode) KILLED ×2 by `TestDecideCheckpointNotLeaseParks`. One step away it admits: with a null lease checkpoint (every epoch-1 owner) any attested checkpoint of the session is admitted, including one the chain never published while the chain's newest is another — see P1-1. |
| P2-1 profile from the session head on resume | **Fixed**, with a residual P2 | `TestR2ProbeResumeProfileUsesClosure`: resume from C1 with a head-only `profile.changed` → `standard`, no source (Decide); through `Run` with the winner carrying the checkpoint and an authoritative head-only `profile.changed` under the winner → `standard`, no source, in restore mode and in create-from-stopped. Shipped N-profile-closure KILLED. Residual: **P2-2** (`Run` silently falls back to the session head when no checkpoint store is bound). |
| P2-2 materialization = `Phase == committed` alone | **Fixed** for a checkpoint-bearing winner | `TestR2ProbeMaterializationOneGenerationBehind`: committed journal sourced one generation behind → `parked(restore_policy)` "sourced from a superseded checkpoint"; prepared journal sourced from the lease checkpoint → parked; committed + sourced from the lease checkpoint → launch. Shipped N-materialization-source KILLED ×2. With a null lease checkpoint (every epoch-1 owner) the source binding is vacuous — P1-1 below, second vector. |
| P2-3 `headless_creation` never evaluated | **Fixed** | `TestR2ProbeNonInteractiveWithoutHeadless` through `Run`: probed `headless_creation=false`, non-interactive create → `refused terminal_backend_capability_unproven`; the interactive create on the same backend launches. Shipped N-headless KILLED ×2. |
| P2-4 step-5 label | **Fixed** | `TestR2ProbeRemoteOwnerLabels`: non-interactive remote owner → `parked(remote_owner)` in launch and restore mode; interactive + attach admitted → `attach_remote`; interactive + attach unproven → `takeover_offer`. Table row and TRACEABILITY corrected. |
| P2-5 survived reviewer mutants RM1/RM2/RM4 | **Fixed** | RV5 (= RM1, `ObserveOwnership` error swallowed) KILLED ×2 by `TestRunTornLeasePropagates`; RV2 (= RM2, steps 3/4 swapped) KILLED ×2 by `TestDecideRemoteOwnerPrecedesMaterialization`; RV6 (= RM4) KILLED ×2. Deterministic across both passes (`logs/50`, `logs/51`). |
| P3-1 `binding-*.tmp` never swept | **Fixed** | `TestBindSweepsStaleStaging` (age-gated 1 min); my `TestR2ProbeCrashBeforeCommit` shows a fresh pre-commit staging file is retained for the minute by design and the retry still installs exactly one `binding.json`. |
| P3-2 hygiene (duplicate `N-discovery`, `doc.go:29` `AuthorizeInput`, "nothing re-implemented") | **Fixed** | TRACEABILITY mutant table = 36 rows, no duplicate name, equal to the 36-row harness; `doc.go` names `AuthorizeMutation` and no `AuthorizeInput` (`grep AuthorizeInput internal/axpane` = 0); `ParkedPayload`/`checkEvidenceIDs` are justified as fail-fast guards. |
| P3-3 landed `fencing.Authorize` ordering / `sessrepo.checkAppend` | Not this leaf's (orchestrator items) | `TestR2ProbeConflictingArmsAndClosedSets`: remote interactive owner + lapsed local grant still refuses `lease_conflict` (landed order); nothing in the candidate touches `fencing`/`sessrepo`. |

## What holds (verified myself, exact candidate tree)

| Check | Result | Evidence |
| --- | --- | --- |
| gofmt / go vet / go build; GOOS=linux, GOOS=windows builds; `go vet ./...` | all exit 0 | `logs/01-fmt-vet-build.log`, `logs/70-tree-exact-checks.log` |
| tracecheck (registry untouched: contracts=63 sections=36) / `cataloggen -adopted … -check` | exit 0 / 0 | `logs/70` |
| `go test ./internal/axpane -count=1 -v` | exit 0, 148 PASS lines, 0 FAIL | `logs/05-axpane-test-v.log` |
| axpane+fencing+sessprofile+matjournal+sessckpt+provhost | all ok | `logs/71-six-packages.log` |
| Shipped harness, isolated copy, two full passes with raw per-plant output | 36/36 match both passes (35 KILLED, control SURVIVED); production blobs restored (OIDs equal to the live candidate) | `logs/40`, `logs/41`, `logs/42-post-harness-blob-oids.log` |
| My mutants (whole suite as killer, two passes) | 8 KILLED ×2 (RV1, RV2, RV3 park-reason collapse, RV4, RV5, RV6, RV9, RV10), control SURVIVED ×2; 3 SURVIVED ×2 (RV7, RV8, RV11 — graded below) | `logs/50`, `logs/51`, `logs/mutants-pass{1,2}/*.log`, `logs/53-delta-witness-under-mutants.log` |
| RV8 (Run's remote-holder pre-check dropped) SURVIVED harmlessly | under the mutant the remote park still emits nothing: the composed `AuthorizeMutation` gate refuses `not_owner` — the pre-check is defense in depth and the composition catches the bypass | `logs/53` |
| Real-kill after commit (`TestBindCrashChildSelfTerminates`) + hooks + concurrency, ×2 | PASS ×2 | `logs/60-crash-rerun-x2.log` |
| My real SIGKILL before commit / after commit (Bind) | absence proven then exactly one `binding.json` on retry; the committed binding survives and a retry with different child facts reattaches to the one child | `TestR2ProbeCrashBeforeCommit`, `TestR2ProbeCrashAfterCommit` |
| My real SIGKILL at both Supersede seams | kill after stage → old binding (op1) intact; kill after rename → new binding (op2) complete; never torn | `TestR2ProbeSupersedeCrashSeams` |
| Identical retries with different child facts ×3 | same inode, same mtime, same bytes, 1 directory entry | `TestR2ProbeIdenticalRetryInodeCensus` |
| Receipt shape / PID | seven closed members; `pid`/`handle` appear only in comments | `logs/80-hygiene.log` |
| Changed paths | exactly the 22 candidate paths; `internal/traceability` untouched; `task-board.config.json` equal to base; 0 `__pycache__`/`.pyc` in the tree | `logs/80` |
| README package section | test + harness commands present; "no `ax` command, no `doctor` result, no runtime capability claim"; the sentinel/`managername` sentence is true as worded for the background path | `logs/80` |
| Capability outside the closed §4.D set (`teleport`); descriptor generation ≠ host binding; invalid config + changed op; unverified + remote interactive | refused mismatch; refused `stale_generation`; `invalid_config` wins; `parked(restore_policy)` | `TestR2ProbeConflictingArmsAndClosedSets` |
| Race gate `go test ./internal/sessquery -race -count=1 -timeout 25m` | **exit 0, `ok` in 494 s** on a loaded host (load 28→48, ~45 foreign `go test`/`.test` processes; not idle). The CR construction's own 27-command suite was green (`coverage_unit=exact_command_shard required=27 green=27`). Judgement: the rev1 red was a host-capacity artifact under the old 10m default, not a regression; with the 25m timeout the gate is green. | `logs/30-sessquery-race.log`, `logs/20-import-census.log` (axpane↔sessquery deps 0/0 both directions incl. test deps; candidate touches 0 sessquery bytes) |

## P1 — must fix before this leaf can be accepted

### P1-1 The bootstrap window, the checkpoint admission and the journal binding are keyed on the lease record's checkpoint, which is null for every epoch-1 owner forever (§13.1, §13.10, §13.11 step 5, §5.3, §5.7)
`decide.go:507` (`bootstrapWindowClosed`), `decide.go:599` (`admitCheckpoint`)
and `decide.go:571` (`checkMaterialization`) all read
`Observation.Winner.HasCheckpoint`/`Winner.Checkpoint`. Lease records are
immutable content-addressed blobs: `sessrepo.CreateLease` mints the epoch-1
lease with a null checkpoint and only `CompareAndSwapLease` (the next epoch,
a takeover) mints a lease that carries one (§5.3: "Null only for epoch-1
`create`; otherwise the validated materialized handoff base"). Nothing ever
writes a checkpoint onto the epoch-1 lease, however many checkpoints the
session publishes. The landed fact for "the newest checkpoint" is the chain
fold — `sessstate.Projection.HasCheckpoint`/`Newest` from
`checkpoint.created`, checkpointed `session.stopped` and `session.resumed`
(§5.7 `newest_checkpoint_id`, §14.4) — which this package already composes in
`canAuthorParked` and never consults here. Consequences, all executed:

- `TestN2ProbeEpochOneOwnerAfterCheckpoint` (`logs/11`): the ordinary
  same-host story (created → idle → stopped with checkpoint C; landed engine
  `state=stopped HasCheckpoint=true newest=C`; winning lease `epoch=1
  HasCheckpoint=false`), then a restore with a new bootstrap operation and a
  journal sourced from C → **`refused idempotency_mismatch` "bootstrap
  operation changed inside the bootstrap window"**. The window never closes
  on the creating host; the after-restore sequence is reachable only after a
  takeover minted a successor lease. This is the rev1 P1-3 symptom for every
  session that never changed hands.
- Same probe, same state: a second attested checkpoint C2 of this session that
  the chain never published (newest is C) → **`launch`**. "Admit exactly the
  winning lease's checkpoint" admits anything when the lease carries none.
- `TestN2ProbeNullLeaseCheckpointAdmitsCallerCheckpoint` (`logs/10`): restore,
  epoch-1 winner, caller checkpoint the lease never published → `launch`;
  committed journal sourced from `sha256:d2d2…` (a digest no lease and no
  event names) → `launch`; through `Run` on the real stores → `launch` with a
  bootstrap binding installed. §13.1: "Captured objects may exist but no
  Checkpoint Record is authoritative … objects are never relabeled as a
  checkpoint"; §13.11 step 5: "validate the newest checkpoint".
- The committed positives pin this state as a launch:
  `TestDecideTable/launch_restore_committed_checkpoint`,
  `TestDecideAfterRestoreSequence/step3_resume_winning_lease_valid_materialization`
  (journal `{Phase: committed}` with no source, epoch-1 winner) and
  `TestRunRestoreCommitted` (epoch-1 lease, fixture journal sourced from
  0xD2). The producer's window/admission tests close the window only through
  `CompareAndSwapLease` successors. The matrix rows "Checkpoint is the winning
  lease's and names this session", "Materialization bound to the lease
  checkpoint", "Idempotent … window closes on winner checkpoint" and
  "After-restore sequence 1-5" are therefore not faithfully driven.

The rework brief's hint ("using the winner's own `HasCheckpoint`/epoch already
present in `Observation`") is only sufficient for epoch ≥ 2; the fixtures'
own epoch-1 lease could never have closed the window, which the producer did
not notice. Required: derive the newest checkpoint through the landed chain
fold (`sessstate.Reduce` → `Projection.HasCheckpoint`/`Newest.ID`; a
successor lease's `Checkpoint` is the handoff base and must agree with it),
close the window on that fact (epoch ≥ 2 implies it), admit exactly that checkpoint for a required
checkpoint, bind a required journal's `SourceCheckpointID` to it, and park
`restore_policy` when it is null and a checkpoint/materialization is required
(the only launch-class path left in that state is the bootstrap retry without a
checkpoint). Re-pin the three positives with the chain carrying the checkpoint,
add the epoch-1-owner negatives (window, admission, journal — Decide and Run),
and ship narrowing mutants on each of the three conditionals.

## P2 — fix in the same rework

### P2-1 Post-window supersession destroys the prior pair's receipt (§4.1, §4.C create/restore rows)
`Supersede` (`binding.go:221`) renames the new receipt over the one
`binding.json` per session; the superseded `(session_id,
bootstrap_operation_id)` receipt is gone.

- `TestN2ProbeSupersededPairIdenticalRetry` (`logs/10`): post-window, op2
  launches (instance `…2222`), op3 launches (instance `…3333`), then an
  **identical retry of op2 launches a third child** (instance `…4444`) instead
  of replaying op2's receipt; `Status` afterwards identifies only the last
  child. §4.1: "An identical retry returns or reattaches to the one recorded
  wrapper/child"; §4.C: "persist receipt before `wrapper_started`; identical
  retry replays it" — the key is `session_id + "/" + bootstrap_operation_id`,
  one receipt per pair.
- `TestN2ProbePostWindowConcurrentDifferentOps`: two `Supersede` calls with
  different operations both succeed (rename-over carries no no-replace
  guard); nothing in the store or in `Run` orders two post-window wrappers.

The rework brief allowed "key the receipt per operation … or supersede it",
so this is P2, not P1; but supersession as implemented breaks the per-pair
replay the same §4.C row requires. Keep the superseded receipts (per-operation
files under the session directory, or an append-only receipt set) so an
identical retry of any recorded pair reattaches to its own child, and make the
post-window install no-replace per pair.

### P2-2 `Run` silently derives the launch profile from the session head when no checkpoint store is bound (§13.10, §2.4)
`run.go:190`: the winner's closure is loaded only `if … && stores.Ckpt != nil`;
otherwise `deriveProfile` falls back to `sessprofile.Derive` over the head.

- `TestN2ProbeNoCkptStoreProfileFallsBackToHead` (`logs/11`): winner carries
  the checkpoint, authoritative head-only `profile.changed` under the winner,
  `stores.Ckpt = nil` → `launch` with `yolo/E2` (the head pair). The same
  request with the store bound carries `standard/null` (the closure). An
  absent store is unknown, not "no closure": mirror the `CheckpointRequired`
  guard at `run.go:169` and fail the run when the winner carries a checkpoint
  and no checkpoint store is bound.

## P3

- **P3-1** The losing-epoch arm of the `Emit` mutation gate is unpinned.
  `TestEmitUnderLosingLeaseRefuses` uses the `creating` chain, so it refuses
  at the fold pre-check ("does not fold from the derived lifecycle state"),
  never at `AuthorizeMutation`. My mutant RV7 (skip the gate when
  `Presented.Epoch < Winner.Epoch`) SURVIVED the whole committed suite twice;
  under it a losing-lease append is accepted under both a remote and a local
  successor winner (`logs/53`). The gate itself is sound on the pristine tree
  (`TestR2ProbeLosingLeaseReachesMutationGate`). Fix the test (foldable chain)
  or add the RV7 row to the shipped harness.
- **P3-2** Create-from-stopped through `Run` is unpinned: RV11 (Run's
  supersede path narrowed to restore mode) SURVIVED twice; on the pristine
  tree `TestN2ProbeCreateFromStoppedPostWindow` launches with a superseded
  binding, under the mutant `Run` errors `idempotency_mismatch` from the store.
  Add a launch-mode post-window `Run` test.
- **P3-3** The coverage denominator was restated, not re-derived: the
  TRACEABILITY acceptance table has **43 rows** (rev1's had 43 as well) while
  results, TRACEABILITY and the matrix say "41 of 41". The rework brief
  required re-deriving the ratio. My ratio: **39 of 43** driven faithfully —
  rows "Materialization validity gates local resume", "Checkpoint admission
  gates resume", "After-restore sequence 1-5" share P1-1; "Bootstrap
  idempotency" carries P2-1.
- **P3-4** Foreground credential-requiring path: `checkRealm` composes the
  §4.C "`credential_capable_execution_realm` when provider credentials are
  required" conditional only for background callers. A foreground caller with
  `Smoke.Required`, a cached passing `resumesmoke` record and caller-supplied
  `Target` claims (generation/OS are `Input` members, not attested evidence)
  launches without the admitted row
  (`TestN2ProbeForegroundCredentialPathWithoutRealmRow`). The README sentence
  ("no cached sentinel, `managername` observation, or bare boolean") stays
  true as worded; state this reading as a bound in TRACEABILITY or compose the
  row when `Smoke.Required` regardless of caller.
- **P3-5** `checkRealmBinding` returns on the first
  `credential_capable_execution_realm` object: with two admitted realm objects
  the outcome depends on input order (first foreign binding → refused; first
  this-host → launch; `TestN2ProbeRealmEvidenceOrder`). Safe direction only;
  iterate and accept when any admitted object binds this host+build.
- **P3-6** An expired or pre-reboot realm row refuses as
  `terminal_backend_manifest_probe_mismatch` / `terminal_backend_stale_generation`
  (Reconcile's first failure), not the §4.2 `capability_unavailable` with typed
  realm/readiness details. Fail-closed, but the typed refusal the spec names is
  not produced; document as a bound or filter dead evidence before reconcile.
- **P3-7** Test hygiene: `TestRunTakeoverOfferEmitsParked` asserts a
  non-interactive remote park with no emission (name is stale);
  `TestRunPostWindowSupersedes` carries a stream-of-consciousness comment and
  drops the materialization requirement instead of sourcing the journal from
  the winner's checkpoint (my `TestR2ProbePostWindowRestoreWithSourcedJournal`
  shows the full path launches).
- **P3-8** LOGBOOK: the rev1 entry's F1 ("both offers still author
  `session.parked` with `remote_owner`") is now false; the rev2 entry reverses
  it only implicitly. Say so explicitly in the rev2 entry.

## Coverage statement

Producer claim: 41 of 41 AC rows driven. The table has 43 rows. Honest ratio:
**39 of 43** (see P3-3). RM1/RM2/RM4 die; the composition holds under every
bypass I planted except the null-checkpoint conditional (P1-1).

## Instruments (all in the evidence archive)

`probes/zz_review2_probe_test.go` (29 probes: 18 `TestR2Probe*` rev1 re-runs
adapted to the new surface, 11 `TestN2Probe*` new plants; `TestN2ProbeStaleLocalPresentedAuthorsUnderWinner` is a harness fault — a chain the landed reducer refuses — superseded by `TestN2ProbeStaleLocalPresentedSameEpoch`), `review2_mutants.py` (12 rows, whole-suite
killer), `delta_under_mutant.py`, `logs/10-11` (probe runs), `logs/40-42`
(shipped harness ×2 + blob OIDs), `logs/50-53` (my mutants ×2 + delta
witnesses), `logs/60` (crash reruns), `logs/70-71` (tree-exact checks, six
packages), `logs/80-81` (hygiene, TRACEABILITY name/row census), `logs/20`,
`logs/30` (import census, race gate), `REVIEW-MANIFEST.txt`.

Verdict recorded by RUN-260917-368f1f; no product edits, commits, checkpoints
or integration were performed; the isolated copies are deleted after
attachment.
