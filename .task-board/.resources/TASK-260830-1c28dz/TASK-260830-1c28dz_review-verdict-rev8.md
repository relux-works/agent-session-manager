# TASK-260830-1c28dz — independent review of revision 8

Verdict: **CHANGES REQUESTED**. One P1 and four P2 findings; route to `to-dev`. Do not accept revision 8.

Exact candidate: `02441ead3068785b9ad5d52771a073da74a13986`; base/checkpoint: `d4bd91d0e740285b21b8d65bf6f074c8009b04ae`. Patch SHA-256 matches the assignment: `59d54056fc7d6245bc71ea31f9be6a538093d326566b15ef069503d7c35f066b`. Authority: pinned SPEC v0.7.0, especially §4.C 1095–1096 and its attach/stop operation rows, §4.2, and §3.2. The task's v0.5.0 reference and the reviewer brief's denial of previous review findings are stale. Read the actual revision-7 verdict and superseding revision-8 rework brief; the overlap liveness subsystem remains assigned to TASK-260922-vcx6yo. No request here expands this leaf into that subsystem.

## P1 — the new peer census treats a corrupt receipt entry as absence

`Lifecycle.Execute(attach) → executeAttach → checkAttachOverlap → AttachStore.Peers`, `internal/termbind/overlap.go:47`, skips every directory before trying to read a receipt. This includes a directory at the exact `<client UUID>.json` receipt path. Such an entry is unreadable/corrupt receipt evidence, not an unrelated staging file or evidence that the client disappeared. `checkAttachOverlap` then takes `len(peers)==0` at `internal/tmuxserver/ops.go:298` and admits a second client without `multi_attach`.

`TestReviewAttachPeerDirectoryFailsClosed` first drives a valid writable attach through `Lifecycle.Execute`, then replaces that client's receipt file with a directory at the same receipt name (retaining its original bytes inside that directory), and drives a different writable client with `multi_attach` absent. On unchanged production it returns success, persists the second receipt, and returns `InputAuthorized:true`. The test fails twice under `-race`; there is no data-race diagnostic and no real tmux process.

**Repeat-of:** revision-7 P1-B's overlap admission class, now at the newly introduced census read boundary. This is within the narrowed obligation: fail closed where overlap cannot be established. B44 only defers liveness; it explicitly promises unreadable peers fail closed and does not authorize treating corrupt receipt-shaped entries as an empty census.

Required repair: distinguish legitimate staging/non-receipt entries from invalid objects in the receipt namespace; refuse unknown/corrupt receipt entries before committing a new attach. Add this production-entry negative and a narrowing witness. Keep the client-liveness owner handoff intact.

## P2-A — a stop probe can wait indefinitely beyond the operation deadline

`Lifecycle.Execute(request-stop) → executeEngineOp → terminstance.Engine → backend.confirmClosed → pollSession`, `internal/tmuxserver/backend.go:453`, passes the original caller context to the blocking runner. `executeEngineOp` computes a deadline in a field but does not give this read-only probe a deadline-bounded context. The clock checks after `Run` cannot cancel a `Run` that has not returned. `OSRunner` uses `exec.CommandContext`, so it likewise only receives cancellation supplied by this context.

`TestReviewStopPollCommandHonorsDeadline` uses a real clock, a 200ms operation deadline, valid hour-long authorization, and a cooperative runner that waits for either its supplied context cancellation or an explicit external release at about 600ms. After the graceful interrupt, `has-session` remains blocked until that external release, then the operation finally returns `stop_timeout`. The final two runs return at approximately 619ms and 621ms; both fail under `-race`, with no race diagnostic. This is a release-controlled counterexample, not a scheduler-latency inference. The test drains the operation before exit.

**Repeat-of:** revision-7 P2's deadline-cancels-waiting requirement, at the stop polling entry instead of attach admission/barrier. Revision 8 correctly bounds the latter waits and correctly revalidates before escalation, but does not bound this wait. B29 discloses unmeasured poll instants; it does not document an intentional unbounded subprocess or override §4.C.

Required repair: apply the operation/graceful wait contract to the actual polling command context and distinguish timeout from evidence of absence. Audit other runner waits through the same shared contract; preserve the no-kill-after-stale-facts repair. A read-only poll timeout must not manufacture process closure.

## P2-B — the claimed same-client replay exemption disappears when another peer exists

`checkAttachOverlap` excludes the requesting client from the peer list, but never checks whether that requesting client already has a valid receipt. Therefore a replay with another recorded peer still requires overlap capabilities, before the durable store can replay the existing receipt. The new `TestAttachSameClientRetryIgnoresOverlap` only uses a lone client, so its name and the B44/row-79/results claims exceed the scenario it measures.

`TestReviewAttachRetryWithOtherPeer` successfully attaches clients A and B with the full admitted capability set, then replays A's identical body with only `local_attach`, exactly the reduced-capability shape used by the committed retry test. It returns `terminal_backend_capability_unproven` instead of replaying. It fails twice under `-race`. The claimed exemption works for 1 of the 2 exercised populations (alone / another peer present), not both.

Required repair: distinguish a validated identical replay from a new client at the overlap admission boundary, preserving authorization, generation and integrity checks. Test replay with another peer present, including read-only and input-authorized shapes as appropriate. Do not infer replay solely from the caller's client ID.

## P2-C — the attached producer archive is revision 3, not revision 8 evidence

The board resource `TASK-260830-1c28dz_producer-evidence.tar.gz` downloads successfully and is a real gzip archive, SHA-256 `32b8972a5d15f26d6213d24d6d4814954ed133fc0c31cb8fc58328d1ebe60b52`. Its embedded results start with “Producer results, rev3”, describe 41 changed paths, and its four harness chunks contain 235 verdict rows. The current standalone results describe revision 8, 48 paths, 293 tmuxserver rows and 41 termbind rows. The scoped board resource inventory has no revision-8 rework archive. These are different candidates; the old archive cannot substantiate the current per-plant/raw-log and composition claims.

The exact CR8 validation attachment separately reports 30/30 configured commands green; that is not missing and is not relabeled as revision 3. The stale archive finding concerns the claimed detailed producer evidence. Independent reviewer reruns below do not repair the producer's provenance.

Required repair: attach the actual revision-8/current-candidate evidence using absolute paths, read the resource back, verify its digest and contents, and make the standalone results point to those exact bytes. Preserve earlier evidence under its actual revision rather than silently presenting it as current.

## P2-D — per-entry kill attribution overstates the new symmetry and wait coverage

The new TRACEABILITY symmetry line claims `stop 4 / quiesce 4` for effectRecheck (three shared arms plus one per-site skip each), and the waitbound row claims four killer paths (attach admission, attach barrier, quiesce span, boundary commit). The shared mutant's process failing does not establish all those paths killed it.

Both full harness passes show generation failing through stop and quiesce, but `N-stop-escalation-expiry` and `N-stop-escalation-deadline` fail only through stop. Quiesce advances from 12:00 to 14:00 against a 13:00 authorization expiry, so the minus-one-hour authorization mutation still rejects at exact expiry; its deadline case is strictly past the deadline and cannot kill an equal-instant narrowing. The boundary-commit test advances seven hours against a six-hour remaining deadline; adding one hour in `N-attach-wait-deadline` leaves its wait immediately expired. That test therefore does not kill the declared wait widening.

The three plants were additionally rerun with only the claimed quiesce or boundary killer selected. All **3 of 3 isolated attributions SURVIVE twice** (six exit-0 runs), with raw logs and exact plant definitions attached. Measured new recheck symmetry is **stop 4 / quiesce 2**, and wait-bound kill paths are **3 of 4**, not four. The full 293-row harness still has its expected verdicts because another selected entry kills each shared plant.

**Repeat-of:** the Story's recurring one-side-kills/shared-row-counted-on-both-sides evidence defect, explicitly identified in the reviewer brief's census requirement. This is not a demand for more abstract census machinery: adjust the affected entry fixtures to drive the admitted class, prove their effect assertions fail under the narrowing, and correct the counts until that evidence exists. Keep precision controls that exercise the still-refused members.

## Revision-7 disposition and preserved boundaries

- Stop's specific stale-fact escalation and quiesce's command-to-command recheck are repaired. The new committed probes pass and assert absence of the forbidden `kill-session`/`detach-client`, not merely a later error. Preserve these repairs.
- Attach admission/barrier waits, quiesce barrier acquisition and boundary report acquisition are now bounded. The original release-controlled attach regression is covered by passing candidate tests. P2-A above concerns a different blocking wait that still lacks a context bound.
- The narrowed overlap gate now refuses the original two capability scenarios, but P1 and P2-B expose missing census-integrity and replay cases. Liveness remains B44/TASK-260922-vcx6yo.
- The refresh timing test is now release-controlled; the changed-package race run passed. Preserve B42's disclosed in-process ordering scope and B43's receipt-enumeration owner handoff. This review does not convert those bounds into implemented guarantees.

## Coverage measured

**8 of 8 named operation entries driven** by passing tests in the exact candidate:

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

**34 of 34 AC rows (59–92) have their named witness sets executed: 256 distinct test names pass.** The matrix repeats those rows, so the 68 textual occurrences were deduplicated by AC number; they are not 68 independent criteria. This is witness-execution coverage, not acceptance of every statement in those rows. Row 79's unreadable-peer and replay assertions are contradicted by the probes above; strict deadline checking is not the same property as cancellation of the outstanding stop probe. The table's denominator does not establish complete normative conformance.

Independent census parsing reproduces A 24×11 = 50 M / 4 B / 210 U; B 24×21 = 10 M / 0 B / 494 U; D 144×21 = 201 M / 183 B / 2640 U; C is 144×11 unreachable. Total **261 M-labelled / 5376 cells**, 187 bounds, 4928 unreachable classifications. The AC rows mirror TRACEABILITY. Counts reproduce the declared classifications, not a certification that every normative requirement has been enumerated. The claimed stop/quiesce 4/4 symmetry and four waiting killer paths are contradicted by isolated attribution (P2-D). The cell totals above reproduce table labels, not valid per-side kill counts. Waiting coverage also does not prove a stop subprocess wait is bounded.

The exec/probe adapters, socket crash/idempotency witnesses, B18 path length, B16 deliberate ambient behavior, B19/B20 custody and fresh-decoy attach attestation remain driven. Ordered wrapper restore branches pass, including local launch, park and remote offer with a lapsed local grant. An additional ordering plant that executes backend restore before refresh/offer selection is killed twice by `TestWrapperRestoreLapsedGrantRemoteOffer`; both remote offer branches fail. This is supplementary ordering evidence, not a gate narrowing.

No `internal/traceability` delta, no new CLI/doctor/runtime capability advertisement. The implementation extends the landed AttachStore owner with Peers and successor support and composes the existing authorization/transition/receipt/fencing owners. The review does not demand a second liveness owner.

## Composition and validation

Independent production-import closure: **34 packages; 36 with test importers**. A fixed production-entry outcome corpus, retained from the prior review and rerun here on both exact base and candidate, covers all 36 packages: **198 shared (package, entry, input) keys, zero moved, zero base-only**. Seven candidate-only keys were driven (`CheckSocketLength` short/long; `ClassifyStatus` absent/live-tri/negative/parked/quiescing-memory). The stale producer archive cannot substantiate the four additional Peers keys claimed by the current results; this review's corpus deliberately reports its actual seven additions. None of its shared inputs moved across admission/refusal arms. This finite corpus does not subsume the new adversarial probes.

Firsthand checks:

- Full `go test ./... -count=1 -json`: exit 0, 45 packages, all 256 named AC tests passed. Fourteen skips outside the changed packages are listed in `ac-audit.json`.
- Restricted-PATH `go test ./internal/tmuxserver ./internal/termbind -count=1 -json`: exit 0, 1044 passing test events, zero skips/failures. `tmux` lookup is absent; before/after `pgrep -x tmux` both return 1 with empty output and stderr. No AC row claims a real-tmux witness.
- Native vet and `GOOS=windows GOARCH=amd64 go vet ./...`: exit 0. Gofmt output empty, diff-check clean.
- Changed-package `-race -count=1`: exit 0. The three new reviewer regression tests separately fail twice each under `-race`, exit 1, with no data-race diagnostic.
- Repository coverage initially exits 1 because the disk fills (`no space left on device`), not a test assertion. After deleting only reviewer-created duplicate archived board snapshots from four disposable copies, the explicit retry exits 0. Both logs and metadata remain attached.
- Candidate config has **30 validation commands**. The exact CR8 validation attachment reports `required=30 green=30 failed=0 missing=0`. Accepted as handoff evidence; the reviewer did not rerun every configured fuzz, cross-build or full-repository race command. Its command-level summary is not substituted for per-test or per-mutant proof.
- Four reviewer custody-entry narrowings (quiesce, boundary, stop, stale termination) are each applied and killed twice by committed refusal tests and valid-body effect probes. The harmless comment control survives twice. These are prior-review attack shapes independently rerun on revision 8, not reused verdicts. Raw logs and per-plant patches are attached.


Shipped harnesses were rerun twice: **293/293 expected tmuxserver verdicts and 41/41 expected termbind verdicts per pass; 334/334 per pass, 668 total**. Per pass: 331 KILLED and three expected SURVIVED (the two harmless controls and the shadowed attach-entry supplementary row). Tmuxserver classifies 286 narrowing, six supplementary and one control; termbind classifies 39 narrowing, one supplementary and one control. Raw subprocess exits are retained per tmuxserver plant; termbind's verbose logs retain every raw block. This confirms process-level battery verdicts, not the false sibling attribution in P2-D.

The initial ENOSPC shard runs produced **36 explicit build ERRORs** (4/32 by pass), plus **36 missing verdicts** (25/11): the latter were detected by reconciling names against the actual 293-member harness list, not inferred green from the shard summary. All 72 affected/missing rows were rerun with separate logs; each retry has exit 0 at the harness level and the expected test-level verdict. The original failed and truncated logs remain attached. No ERROR or missing row is counted as a kill.

Audit: live changed files equal the immutable candidate; HEAD remains `d4bd91d0`. Local main is zero commits ahead; remote main advertises `40bb8c9f89a5430c556e8746e1dca9769f4b3c47`. The 23 overlapping base/main paths are predecessor Story content: 16 checkpoint-retained, seven intentional leaf edits. Config equals base; no registry delta or Python cache artifacts. All four reviewer source copies compare equal after restoration. Only their archived board duplicates were removed for disk recovery; the complete candidate copy retains its board snapshot. No live product edit, index write, commit, branch switch, rebase, checkpoint, integration or acknowledgement occurred.

This verdict and its board note serve as the read-only review logbook record. Evidence: `TASK-260830-1c28dz_review-evidence-rev8.tar.gz`, with probes, plants, raw logs, real exits, timing/load, source/restoration audits, census/AC/importer audits and downloaded resource digests. Large existing producer artifacts are referenced by task-scoped name and digest rather than repackaged as new evidence.

## Rework scope (for the producer)

1. Make the new Peers census refuse invalid receipt-shaped entries; drive the second attach through Lifecycle.Execute and assert no new receipt/vector. Preserve the explicit liveness handoff.
2. Bound stop's actual polling command wait, retain honest unknown closure on timeout, and apply the shared waiting contract consistently to analogous runner waits.
3. Preserve identical same-client replay with another peer present without bypassing identity, authorization, generation or receipt-integrity checks.
4. Replace the stale producer evidence attachment with current-candidate bytes, verify it after downloading, and correct the affected coverage/replay/fail-closed claims and census witnesses.
5. Repair the three false per-entry kill attributions and report actual symmetry/wait-path counts; a shared process exit cannot stand in for its passing sibling entry.
6. Retain the accepted revision-7 repairs and existing owner bounds. Leave the Story candidate uncommitted; registry and integration remain with their assigned owners.
