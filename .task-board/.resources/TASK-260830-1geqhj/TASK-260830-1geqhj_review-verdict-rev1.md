# TASK-260830-1geqhj — independent review verdict, CR-TASK-260830-1geqhj-1 revision 1

**Verdict: CHANGES REQUESTED (routed `to-dev`).** Four P1 classes, five P2, three P3.

Reviewer: RUN-260917-3934e0 (claude-opus-5 max). Reviewed bytes: base
`7efe3854a5a14545f9fa2b7697accb1883249f1f`, candidate tree
`bb8dbdebcd5102a772960d8399661a2b2dcae0c1` (recomputed from the live worktree
with an untracked-aware temporary index: equal), patch sha256
`472870c8b6c2f69e54a8de0901e0ccf127f3f4aee4d85bbe070d45339043c3b8` (equal to the CR record). All probes ran in an isolated
`git archive` copy under `.temp/TASK-260830-1geqhj/review/`; the live Story
worktree, index, branch and HEAD were never touched. Every instrument, raw
log and subprocess exit is in `TASK-260830-1geqhj_review-evidence-rev1.tar.gz`
(paths below are relative to that archive).

## What holds (verified myself)

| Check | Result | Evidence |
| --- | --- | --- |
| gofmt / go vet / go build on the exact tree | exit 0 / 0 / 0 | `logs/01-03`, `logs/70-tree-exact-checks.log` |
| GOOS=linux, GOOS=windows builds | exit 0 / 0 | `logs/70` |
| tracecheck (registry untouched) | exit 0, contracts=63 sections=36 | `logs/70` |
| `cataloggen -adopted … -check` | exit 0 | `logs/70` |
| `go test ./internal/axpane -count=1 -v` | exit 0, 119 PASS lines | `logs/05-*` |
| axpane+fencing+sessprofile+matjournal+sessckpt+provhost | all ok, exit 0 | `logs/71-six-packages.log` |
| Shipped mutant harness, isolated copy, two full passes | 25/25 match both passes (24 KILLED ×2, control SURVIVED ×2); blobs restored (OIDs equal to candidate tree) | `logs/40`, `logs/41`, `logs/42-43` |
| Real-kill after commit (`TestBindCrashChildSelfTerminates`) ×2 | PASS ×2 | `logs/60` |
| My real SIGKILL before commit (AfterStage) | restart proves absence (found=false, err=nil); retry installs exactly one `binding.json`; never a second child | `logs/10` probe C1 |
| Identical retries with different child facts ×3 | same inode, same mtime, same bytes, session dir has 1 entry | `logs/10` probe C2 |
| Receipt shape / PID | 7 closed members, no PID/handle/socket; `pid` appears only in comments | `logs/80-hygiene.log` |
| Changed paths | exactly the 21 candidate paths; 0 `__pycache__`/`.pyc` in tree; `internal/traceability` untouched | `logs/80` |
| Capability outside the closed §4.D set (`teleport`) | refused `terminal_backend_manifest_probe_mismatch` before Decide | probe 12 |
| Descriptor generation ≠ host binding | refused `terminal_backend_stale_generation` | probe 13 |
| Arm precedence (invalid config + changed op) | `invalid_config` wins | probe 11 |
| Race gate (`internal/sessquery -race`) | **host-capacity artifact, not a regression**: my rerun exit 0 in 656 s (`-timeout 25m`) with load 13→27 and 11 other `go test` processes — *not* an idle host; import census 0/0 both directions; the candidate touches no sessquery byte; the only README/LOGBOOK reference in sessquery is `testdata/mutate.py`. The configured `go test ./... -race` without `-timeout` is a latent trunk flake for whoever owns CI, not this leaf. | `logs/30`, `logs/20-import-census.log` |

## P1 — must fix before this leaf can be accepted

### P1-1 §5.2 event authority: `session.parked` is authored under a lease the local host does not hold
`run.go:245-282` (`emitParked`) always authors under `observation.Winner`
(`LeaseEpoch: observation.Winner.Epoch`, `LeaseID: observation.Winner.LeaseID`)
with `created_by_host_id = LocalHostID`. For every `remote_owner` park (both
offers) and for unverified/ambiguous/failed-handoff parks whose winner is
remote, that lease belongs to another host. §5.2: "The winning owner MUST
serialize state-changing events … two different events at one sequence under
the winning lease … is `invalid_state_transition`; the session MUST park".

- Probe 1 (`TestReviewProbeRemoteOwnerParkedEventLease`): after a graceful
  takeover to `fixtureRemoteHost` (lease B, epoch 2) `Run` → `takeover_offer`
  and writes `session.parked` with `lease_id=B epoch=2 seq=1
  created_by=<local host>`. The owner's own first event at (B, seq 1) is then
  refused locally: `event lease sequence 1 repeats chained sequence through 1`
  — the collision the spec forbids, manufactured by the parked host.
- Probe 1b: `Verified=false` with a remote winner → `parked(restore_policy)`
  authored under the remote-held lease B as well.
- Probe 2 (`TestReviewProbeParkedEventFoldsInStateEngine`): the chain the
  candidate's own fixtures produce (`session.created` → wrapper's
  `session.parked`) is refused by the landed `sessstate.Reduce`:
  `invalid_state_transition: event session.parked … moves "creating" to
  "parked" without a Section 5.7 edge`. The wrapper never consults the
  lifecycle state before authoring a lifecycle-changing event; only
  materializing/stopped/failed may move to parked.
- Probe 9 (`TestReviewProbeEmitUnderLosingLease`): `EmitParked` under the
  superseded lease A (epoch 1) with the tail still at A is **accepted**
  (`err=nil`); the landed repo only refuses once the tail has moved
  (`ErrStaleLease`). `Emit` (`events.go:161-212`) never composes
  `fencing.AuthorizeMutation` — grep: zero callers of `AuthorizeMutation`/
  `AuthorizeInput` anywhere; `doc.go:29` nevertheless lists `AuthorizeInput`
  as composed.
- Probe 15 (`TestReviewProbeLosingLeaseProfileEventDrivesLaunch`) shows the
  blast radius of the same hole: a `profile.changed` appended under losing
  lease A after local successor A2 won is accepted by the chain and then
  **drives the launch**: `profile=yolo source=<losing event>
  mapping="--dangerously-bypass-approvals-and-sandbox"`. §2.4: "Losing-lease
  or ambiguous events MUST NOT change [the effective profile]".

Required: author parked evidence only under a lease the local host holds
(or through the landed parked channel / a non-chain host-local record —
the orchestrator/spec decides), gate every `Emit` through
`fencing.AuthorizeMutation` (the landed "event append" entry), and refuse to
author a lifecycle event the §5.7 table cannot fold from the derived state.
The pre-existing tail-only acceptance in `sessrepo.checkAppend` is a landed
property worth a separate board item; it is not this leaf's to fix, but this
leaf must not build a new writer on it.

### P1-2 §4.2 realm rule: an unbound boolean authorizes a background resume
`decide.go:582-603` (`checkRealm`): `if input.Realm.AttestedServer { return
nil }`. The only "structural" defence (`TestRealmCarriesNoCachedEvidence`) is
a reflection census of field *names*; it proves that no field is *called*
cached-sentinel, not that a cached observation cannot authorize.

- Probe 3 (`TestReviewProbeAttestedServerBooleanAuthorizes`): background
  caller, `AttestedServer=true`, `Smoke.Required=true`, passing smoke record,
  `Target.TmuxServerGeneration="generation-STALE-pre-reboot"` (≠ the backend's
  raw generation) → **launch**; admitted set
  `[durable_disconnect headless_creation local_attach reboot_restoration]` —
  no `credential_capable_execution_realm` row exists anywhere in the
  universe, and with `Smoke.Required=false` the same boolean launches with no
  realm evidence at all.
- `checkSmoke` (`decide.go:537-556`) binds provider/version/platform/arch
  only; the resumesmoke record carries no tmux-server generation, macOS
  version or expiry. `Target.TmuxServerGeneration`/`MacOSVersion` are refusal
  decoration, never compared to anything. §4.2: "Evidence MUST bind the exact
  tmux server generation, provider build, and macOS version … Logout or
  reboot invalidates unrenewed realm evidence. A cached sentinel or prior
  managername observation MUST NOT authorize resume." §13.11: background
  restore "MUST NOT … infer provider authentication from launchctl
  managername or a cached sentinel."
- The landed composition target exists and is unused: §4.D
  `credential_capable_execution_realm` Capability Evidence (terminal_binding_id,
  provider_id/build, sentinel_result, provider_auth_smoke_result,
  backend_generation_digest, os_version, expires_at, signature) admitted by
  `terminalbackend.Reconcile`; §4.C names it as the create/restore dependency
  "when provider credentials are required". `terminalbackend.CheckOperation`'s
  own doc assigns the conditional dependencies to "the lifecycle owner" —
  this wrapper — and the wrapper never evaluates them.
- README claim "no cached sentinel or `managername` observation can authorize
  resume" is therefore unsupported as worded.

Required: for a background caller / credential-requiring path require the
admitted `credential_capable_execution_realm` row bound to the exact host
binding, probed provider build and current generation (the landed evidence
object carries all of it); drop the bare boolean as an authorizing input.

### P1-3 Bootstrap window not modeled: one binding per session, forever
`binding.go:92-98,125-134` keys the receipt by `session_id` only and refuses
*any* different operation (`existing.OperationID != operationID` →
`idempotency_mismatch`), and `decide.go:321` mirrors it. §4.1: "changing the
operation for a session still in the bootstrap window is
idempotency_mismatch"; §13.1 closes the window at the first checkpoint; the
§4.C `restore` row carries its own `bootstrap_operation_id` with key
`session_id + "/" + bootstrap_operation_id`; `create` is allowed from
`stopped`.

- Probe 5 (`TestReviewProbeSecondBootstrapOperationAfterWindow`): create
  binds op1; a successor lease with the captured checkpoint closes the window;
  `Run(ModeRestore, op2, committed materialization, admitted checkpoint)` →
  **refused `idempotency_mismatch`** ("changed inside the bootstrap window").
  On any host that once created the session, the entire after-restore
  sequence is unreachable and no API can clear the receipt.
- The winner's `HasCheckpoint`/epoch (already in `Observation`) is the
  landed fact that defines the window; the store needs a per-operation key
  (or supersession) plus the window predicate.
- Reviewer mutant RM4 (idempotency arm dropped for restore mode) **survived
  the whole suite twice**; witness under the mutant: restore + changed
  operation → `reattach` to a child bound under *another* operation. No
  restore-mode idempotency test exists.

### P1-4 Checkpoint admission is unbound to session and lease
`decide.go:496-508` (`admitCheckpoint`) = digest self-consistency +
equality with a *caller-supplied* id; `gates.go:120` loads any digest from
`sessckpt` with no session scoping.

- Probe 6 (`TestReviewProbeCheckpointNotLeaseCheckpoint`): winning lease
  checkpoint `sha256:c9c9…`, caller names the older `sha256:83ec…` → **launch**.
  §13.11 step 5: "validate the newest checkpoint".
- Probe 6b (`TestReviewProbeForeignCheckpointAdmitted`): a valid checkpoint
  captured for session `…90ff` is admitted for resume of session `…90ab` →
  **launch**. The checkpoint's own `session_id`/lease members are never read.

Required: admit exactly the winning lease's checkpoint (`Winner.Checkpoint`)
and bind its `session_id` (the landed `sessckpt.Admit`/head-binding path or
the attested members) — "resume authorization through the fencing gates plus
the checkpoint admission" currently has no admission.

## P2 — fix in the same rework

- **P2-1 Profile authority on resume uses the session head, not the
  checkpoint closure.** `decide.go:378` calls `sessprofile.Derive` in every
  mode; §13.10: "For a validated checkpoint, derive the exact effective
  profile/source from its event-head closure"; the landed entry is
  `DeriveForHeads`. Probe 8: resume from C1 (closure `[created]`) with a
  head-only `profile.changed` E2 → launch carries `yolo/E2`. TRACEABILITY
  claims the PROFILE-DIRECT-RESUME direction; the test that "pins" it
  (`TestDecideProfileSourcePinsEvent`) runs in launch mode without a
  checkpoint.
- **P2-2 Materialization "valid" = `Phase == committed` only**
  (`decide.go:344`). Probe 7: a committed journal whose
  `source_checkpoint_id` is one generation behind the lease checkpoint →
  launch. Bind the journal to the lease checkpoint (the journal carries
  `SourceCheckpointID`; the winner carries `Checkpoint`).
- **P2-3 `headless_creation` conditional never evaluated.** `Interactive` is
  not a `Decide` input; probe 4: non-interactive create with probed
  `headless_creation=false` (durable_disconnect still confers create) →
  launch. §4.C create row: "`headless_creation` when non-interactive".
- **P2-4 Step-5 label.** Probe 14 / producer row `takeover_offer_noninteractive`:
  a remote owner outside an interactive terminal yields `takeover_offer`;
  §4.2 step 5: "enter parked without launching the provider in all other
  cases". Durable effect is the same parked event, so this is a label, but
  it is pinned wrong by a committed test and copied into TRACEABILITY.
- **P2-5 Survived reviewer mutants (full suite as killer, 2/2 runs, deltas
  witnessed under each mutant, `logs/50-53`):**
  - RM1 — `ObserveOwnership` error swallowed into an empty observation →
    SURVIVED; witness: a torn lease blob turns from `err=… chain is corrupt`
    into `action=parked err=nil`. "Unknown is never absence" is pinned only
    for the pane store, not for the lease path.
  - RM2 — materialization arm moved before `authorize` (steps 3/4 swapped) →
    SURVIVED; witness: remote interactive owner + staging journal turns from
    `attach_remote` into `parked(restore_policy)`. The after-restore
    precedence the scenario table claims to pin is unpinned.
  - RM4 — see P1-3.
  - Killed as expected: RM3 park-reason collapse, RM5 `AttestedServer` arm
    narrowing (the producer never mutated the authorizing arm; RM5 is
    killed only because the fixture's generation string matches), RM6 offer
    emission dropped, RM7 successor sequence; control RC SURVIVED.

## P3

- **P3-1** Pre-commit crash leaves `binding-*.tmp` staging garbage that no
  retry sweeps (probe C1: `[binding-2863015058.tmp binding.json]`). Documented
  as ignored; a sweep is cheap.
- **P3-2** Hygiene: TRACEABILITY mutant table lists `N-discovery` twice
  (26 rows shown, "25 of 25" claimed; the harness has 25); `doc.go:29` names
  `AuthorizeInput` as composed (it is not); `ParkedPayload`/`checkEvidenceIDs`
  duplicate closed-shape predicates `canonicaljson` already enforces at
  `AppendEvent` (harmless pre-checks, but "nothing re-implemented" is
  overstated).
- **P3-3** Landed-gate observation (not this leaf's defect, for the
  orchestrator): `fencing.Authorize` checks grant expiry before ownership
  direction, so a remote interactive owner with a lapsed local grant refuses
  `lease_conflict` instead of offering attach/takeover (probe 10); and
  `sessrepo.checkAppend` accepts a stale-lease append while the tail sits on
  that lease (probe 9). Both deserve their own board items.

## Coverage statement

Producer claim: 41 of 41 AC rows driven. Names and call sites resolve, but
**six rows are not faithfully driven per the pinned text**: "Cached evidence
never authorizes" (field-name census; P1-2), "Checkpoint admission gates
resume" (P1-4), "Bootstrap idempotency" (no window; P1-3), "`session.parked`
authored under lease" (wrong authority; P1-1), "After-restore sequence 1-5"
(step 4/5 label, 3/4 order unpinned; P2-4/P2-5), "Effective profile derived
with source" (head, not closure; P2-1). Honest ratio: 35 of 41.

## Instruments (all in the evidence archive)

`probes/zz_review_probe_test.go` (16 probes), `probes/zz_review_delta_test.go`
(mutant delta witnesses), `review_mutants.py` (8 rows, whole-suite killer,
raw logs in `logs/mutants/`), `delta_under_mutant.py`, `logs/10-11`
(probe runs), `logs/40-41` (shipped harness ×2 with raw per-plant logs),
`logs/30` (race rerun with uptime before/after), `logs/70-71`, `logs/80`.
`PYTHONDONTWRITEBYTECODE=1` everywhere; 0 `__pycache__` left.

Verdict recorded by RUN-260917-3934e0; no product edits, commits,
checkpoints or integration were performed; the isolated copies are deleted
after attachment.
