# TASK-260830-1c28dz — independent review of revision 7

Verdict: **CHANGES REQUESTED**. Two P1 findings and one P2; route to `to-dev`. Do not accept revision 7.

Exact candidate: `a7ffc935ff3c44396734ea1b39cdc1c8acc2651a`; base/checkpoint: `d4bd91d0e740285b21b8d65bf6f074c8009b04ae`. Authority: repository-pinned SPEC v0.7.0, §4.C (1095–1096 and the operation table at 1205–1211), §4.2 and §3.2. The task's v0.5.0 reference is stale. The reviewer brief's denial of prior findings contradicts the actual verdict history; revision-6 verdict and revision-7 rework brief were read and treated as authoritative. Review was read-only against the live Story sources. All plants/probes ran in disposable copies; no commit, branch change, index write, checkpoint, integration or acknowledgement occurred.

## P1-A — stop escalation executes after its authorization facts go stale

`Lifecycle.Execute(request-stop) → executeEngineOp → terminstance.Engine → backend.PerformEffect(process_closed) → confirmClosed`, `backend.go:356–379`, checks the live authorization/generation in the engine **before** `pollSession`. After that wait, `confirmClosed` issues `kill-session` at line 369 without revalidating generation, authorization expiry/current lease, or the operation deadline. The next engine recheck refuses only after the kill has happened.

`TestReviewStopRechecksBeforeEscalation/{generation,authorization,deadline}` drives the production lifecycle entry. It starts with a valid request, sends the graceful interrupt, changes exactly one admission fact during the `has-session` poll, and then observes `kill-session` despite the changed fact. Generation rotates to `generation-two`; authorization expires at +3ms; the independent request deadline expires at +3ms. The fake clock advances 10ms during the poll, past the 5ms graceful wait. Each case fails twice under `-race` on unchanged production. The returned result already lists `graceful_stop_requested` and `process_closed`; the eventual stale-generation/unauthorized/stop-timeout error does not undo the forbidden kill. There is no data-race diagnostic.

**Repeat-of:** revision-6 P1/P2-A's admission-to-effect interval, now inside a composite stop effect. The rev7 results explicitly say the sibling paths have no such interval because the outer loop rechecks by construction; this counterexample disproves that claim. B42 covers attach's immediate post-check/file-I/O window, not a stop poll followed by a new destructive command. B29's unmeasured poll boundary is not an authorization deferral.

Required repair: carry a coherent revalidation contract to the actual command/effect boundary after waits, including escalation; distinguish graceful timeout from the overall operation deadline. Commit a negative that asserts **no kill command**, not merely the final error. Audit other multi-command/waiting effects using the same invariant and either close them or state precise owner-bound limitations.

## P1-B — overlapping clients bypass the specified capability gates

The §4.C attach row requires `multi_attach` for overlap and `multiple_input_clients` plus AX policy for concurrent input. `executeAttach` checks operation and transport capability, then commits the receipt at `ops.go:153`; neither it nor `AttachStore.Attach` checks existing clients or the two overlap capabilities. There is no corresponding census row or explicit bound.

`TestReviewAttachOverlapCapability` performs two distinct client attaches to the same instance through `Lifecycle.Execute`, keeping the first receipt and vector:

- `read-only-no-multi`: remove `multi_attach`, attach one read-only client, then attach a second with source `active`. The second call succeeds and commits its receipt.
- `writable-no-multiple-input`: the fixture advertises `multi_attach` but **not** `multiple_input_clients`; both input-authorized attaches succeed, both receipts persist, and the second returns `InputAuthorized:true` and a writable vector.

Both cases fail twice under `-race` on the unmutated candidate. No real tmux is needed to witness this leaf's authorized-input receipt/vector effect. The first vector remains valid for its caller; the implementation has no liveness or active-client census that could prove non-overlap. Correct transport authorization does not confer the missing capability.

Required repair: implement overlap admission using an authoritative active-client/receipt contract, with same-client retry distinguished from a new client and with atomic admission where needed. Preserve the landed authorization owner. If client liveness is outside this leaf, explicitly route that owner/contract gap and fail closed where overlap cannot be established; do not silently claim the attach row complete. Add the missing gates and witnesses to the conformance census.

## P2 — attach deadlines reject after waiting but do not cancel waiting

`ops.go:102` calls `ServerAdmission()` synchronously without a deadline/context contract; `ops.go:136` uses an uncancellable mutex lock. The new deadline check at line 141 runs only after those waits finish. It correctly prevents the late receipt, but a blocked admission or another holder of the barrier can block this request indefinitely after its deadline.

`TestReviewAttachWallDeadlineCancelsWait/{lock,admission}` uses real time and a 200ms operation deadline with an authorization valid for an hour. Each call remains blocked until the reviewer releases the wait at roughly 401–405ms; only then does it return timeout. Both scenarios fail twice under `-race`. `TestReviewAttachDeadlineCancelsLockWait` additionally proves an expired **and context-cancelled** attach stays blocked until external unlock. The probes release every waiter before returning; they leave no goroutine behind. Host load is recorded; this is a release-controlled wait, not an assertion that scheduler latency alone proves an unbounded operation.

**Repeat-of:** revision-6 P2-A, residual waiting half of §4.C's “A deadline cancels waiting, not a committed effect.” Preserve the new no-receipt checks, but bound the waits themselves and assert no late receipt/background commit after the caller has timed out. Apply the waiting contract consistently to barrier users.

## Revision-6 disposition

- P1 post-lock generation: the specific old-generation receipt window is closed. `TestAttachRefusesGenerationRotatedDuringLockWait` passes; its shipped narrowing is rerun.
- P2-A late receipt: operation-expired/authorization-valid and exactly-at-deadline tests pass. P2 above is the remaining cancellation requirement, not a claim that rev7 still commits that receipt.
- P2-B input escalation: the composed test now asserts receipt absence. The shared-owner narrowing kills it; no authorization gate was forked.
- P3 mutation labels: loop wiring rows are now supplementary `D-` rows; the new admitting loop-authorization row and no-bind entry witnesses supply negative evidence. The old two wiring kills are not counted as gate narrowings.
- Preserve the accepted scoped B43 closure-authority owner handoff, incarnation/barrier fixes, and B42's disclosed scope. This verdict does not silently turn those bounds into implemented guarantees.

## Coverage measured

**8 of 8 named operation entries driven**, through passing candidate tests:

| Operation | Named test | Production call site |
| --- | --- | --- |
| create | TestExecuteCreateInteractive | Lifecycle.Execute → executeCreate → terminstance.Engine |
| attach | TestExecuteAttach | Lifecycle.Execute → executeAttach |
| status | TestExecuteStatusPresent | Lifecycle.Execute → executeStatus → backend.ObserveStatus |
| quiesce | TestExecuteQuiesce | Lifecycle.Execute → executeEngineOp → closeInput |
| safe-boundary | TestExecuteBoundary | Lifecycle.Execute → executeEngineOp → observeBoundary |
| stop | TestExecuteStop | Lifecycle.Execute → executeEngineOp → confirmClosed |
| stale termination | TestExecuteTerminate | Lifecycle.Execute → executeTerminate → runLifecycleEffects |
| restore | TestExecuteRestore | Lifecycle.Execute → executeRestore → runLifecycleEffects |

**34 of 34 AC rows (59–92) have their named witness sets executed: 239 distinct test names pass.** This is witness-execution coverage, not acceptance of every claim. The broad generation/deadline claims in rows 85–86 are contradicted at stop escalation; attach's overlap requirements are omitted from the enumerated table entirely. The denominator does not expand itself when a normative requirement was missed, so 34/34 must not be presented as complete §4.C conformance. Bounds and helper-level rows retain their stated scope.

Independent census parsing reproduces Table A 24×11 = 50 M / 4 B / 210 U; B 24×21 = 10 M / 0 B / 494 U; D 140×21 = 196 M / 182 B / 2562 U; C is 140×11 unreachable. Total **256 measured / 5248**, 186 bound, 4806 unreachable. Matrix and TRACEABILITY B/D tables are identical and all 34 AC rows mirror. Counts are reproduced classifications, not a certification that an incomplete normative gate inventory is complete.

The eight operation paths, exec/probe adapters, socket crash/idempotency witnesses, B18 length, B16 ambient decision, B19/B20 custody, foreground fresh-decoy admission and ordered after-restore branches remain driven. The wrapper tests include lapsed-local-grant remote offers, local/parked outcomes and non-cooperative refresh. No `internal/traceability` registry delta; no new CLI/doctor/runtime capability advertisement. README's strict-deadline/recheck prose needs narrowing or repair with P1-A/P2.

## Composition and validation

Independent production-import closure: **34 packages; 36 with test importers**. The fixed outcome corpus was run on base and candidate: **198 shared (package, entry, input) keys, zero moved, zero base-only**, seven declared candidate-only keys (`CheckSocketLength` short/long; `ClassifyStatus` absent/live-tri/negative/parked/quiescing-memory). Every importer is represented. There are no unnamed inputs moving across admission/refusal arms in this corpus. Termbind's successor is an extension of its landed owner; the candidate composes terminalbackend, terminstance, axpane and fencing.

Firsthand checks on exact-candidate copies:

- Full `go test ./... -count=1 -json`: exit 0, 45 packages; all referenced AC tests passed. Fourteen skips outside the changed packages are listed in `ac-audit.json`.
- No-tmux PATH run: exit 0, 1012 passing test events in tmuxserver/termbind, zero skip/fail; tmux lookup fails, process snapshots show no tmux process. No row claims a real-tmux witness.
- Native vet, Windows amd64 vet, repository coverage: exit 0. Gofmt output empty; diff-check clean.
- Changed-package race run: **exit 1**, `TestWrapperRestoreRefreshBoundEnforced` measured 52.924166ms against its 50ms ceiling for a configured 1ms refresh. No race diagnostic. This failure remains in the evidence; a targeted five-repeat rerun passed (exit 0) and is recorded separately, not substituted for the failed run.
- Configured validation: candidate config has **30 commands**. Exact CR validation attachment reports `required=30 green=30 failed=0 missing=0`; 353 producer archive entries pass SHA-256 verification. These are accepted as producer/handoff evidence; I did not independently rerun every configured fuzz/cross-build/full-repository-race command. The independently executed checks and failing behavior probes above support the rejecting verdict.
- Four reviewer-only custody-entry narrowings (quiesce, boundary, stop, stale termination), each applied at the shared dispatch guard, are killed twice by committed tests. Those committed kills are refusal reroutes on malformed bodies. Additional valid-body probes pass baseline twice and kill each plant twice by proving that weakened custody admits actual effects under a 0777 root. An applied harmless control survives twice. Per-plant diffs, raw output and exits are attached. A separate restore-ordering plant executes backend admission before refresh/remote-offer selection; both lapsed-grant remote-offer branches kill it twice. This is supplementary ordering evidence, not counted as a narrowing.

Shipped harnesses rerun twice: **285/285 expected tmuxserver verdicts per pass** (284 KILLED, one harmless control SURVIVED) and **40/40 termbind verdicts per pass** (39 KILLED, one control SURVIVED). **325/325 per pass; 650 total**, zero ERROR/MISMATCH. Tmuxserver labels classify 279 narrowing, five supplementary and one control; the corrected rev6 wiring rows remain supplementary. Every tmuxserver plant has a raw log with subprocess exit in each pass; termbind verbose logs retain all 40 raw subprocess blocks per pass. These battery results do not cover the absent overlap gates or the original-source failures above.

Audit: live changed files match the immutable candidate; HEAD remains d4bd91d0; local main is zero commits ahead. Remote main remains `40bb8c9f89a5430c556e8746e1dca9769f4b3c47`. The 23 overlapping base/main paths are predecessor Story changes: 16 checkpoint-retained, seven intentional leaf edits. Config equals base; no registry changes or Python cache artifacts. Early reviewer setup attempts with an incorrect scratch cwd/no matched test and an accidental base-only termbind harness run are excluded from all candidate-test and mutation counts.

The verdict is the read-only review logbook record. Evidence: `TASK-260830-1c28dz_review-evidence-rev7.tar.gz`, containing probes, applied patches, raw test and mutation logs, exact exits, census/importer/tree audits, timing/load and resource verification.

## Rework scope (for the producer)

1. Close P1-A at stop's post-wait escalation boundary and audit the same invariant in composite effects. Assert absence of the forbidden command, retaining all rev7 attach fixes.
2. Implement or explicitly owner-route and fail-close P1-B overlap admission; add multi-attach and multiple-input capability/policy negative witnesses and census cells. Preserve idempotent same-client replay.
3. Make attach admission and barrier waits deadline/cancellation-aware; ensure timed-out calls cannot later create authorized receipts. Add bounded release-controlled tests.
4. Correct the strict deadline/generation/completeness prose and retain honest ratios/bounds. Investigate the recorded race timing failure and rerun relevant checks without relabeling the failed run.
5. Retain all accepted prior closures and scoped owner handoffs. Leave the Story candidate uncommitted; registry and integration stay with their assigned owners.
