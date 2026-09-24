# TASK-260830-1c28dz — independent review of revision 6

Verdict: **CHANGES REQUESTED**. One P1, two P2 and one P3 finding; route to `to-dev`. Do not accept CR revision 6.

Reviewed immutable candidate tree `0bb8d6c9c04d0cdb97483eed8f11803d91a97967` against `d4bd91d0e740285b21b8d65bf6f074c8009b04ae`. Authority is the repository-pinned SPEC v0.7.0, especially §4.C (1095–1096, 1205) and §4.2 (1433–1441), with §3.2 custody. The task's v0.5.0 scope is stale. The reviewer brief contradicts itself about prior verdicts: the actual revision-5 verdict and revision-6 rework brief were read and used. All source mutations and new probes ran in disposable archive copies; live Story source, index, branch and HEAD were preserved.

## P1 — generation admission goes stale while attach waits for its new lock

`ops.go:102–107` obtains and checks server attestation before `barrierMu.Lock()` at line 116. After acquiring that lock it rechecks only closure, then creates the durable writable-client receipt at line 127. There is no generation recheck at that effect boundary. SPEC §4.C requires generation and authorization to be rechecked immediately before each side effect.

`TestReviewAttachGenerationAfterLockWait` holds the lifecycle barrier lock, starts `Lifecycle.Execute(attach)`, and waits for a valid old-generation server admission. It then changes the live generation source (also used by subsequent ServerAdmission calls) to `generation-two` before releasing the lock. Attach succeeds with `InputAuthorized:true` under the old request generation. The regression fails twice under `-race`, with no data-race diagnostic and no production mutation. The test uses an atomic generation source and channels; this is a logical stale-snapshot defect, not a data race.

Repeat-of: revision-5 P1-B's admission-to-effect interval, now at generation rather than closure. The closure recheck and its mutex do close the originally reported quiescence interleaving. Preserve that fix, but make the effect-boundary admission include live generation. B42's cross-process and already-returned-vector bounds do not cover a new writable receipt admitted under a generation that changed before this same Lifecycle acquired its lock.

## P2-A — attach commits a new receipt after its operation deadline

`ops.go:81` checks deadline only before potentially blocking ServerAdmission and lock acquisition. Neither the post-lock block nor `backend.createAttachClient` checks the request deadline. The latter rechecks authorization expiry, which is a different timestamp.

`TestReviewAttachDeadlineAfterAdmission` starts with the fixture clock at 12:00, a request deadline at 18:00, and authorization valid until the next day. Its ServerAdmission advances the clock to 19:00 and returns a valid admission. `Lifecycle.Execute(attach)` returns success, creates a previously absent durable receipt, and emits writable argv without `-r`. It fails twice under `-race` on unchanged production. This is a newly committed effect after the deadline, not a timeout racing an already committed receipt. Row 85's assertion that deadlines are checked before every effect is false for attach.

Keep the request deadline distinct from authorization expiry and revalidate it after waits before the receipt effect. Add a regression whose operation expires while its authorization remains valid, and include lock-wait admission in the deadline contract.

## P2-B — false-to-true input escalation lacks an effect-level negative witness

Reviewer narrowing A3 changes only the landed CheckAttachRequest input-binding clause: it continues rejecting a false request against true authorization, but admits a true request against false authorization. This single-site weakening is used by both the lifecycle entry and the durable attach store. The committed tmuxserver suite and the terminalbackend/termbind owner suites each survive twice. A broader `go test ./... -count=2 -timeout=5m` attempt exits 1: environ refusal-site audit failures and a sessquery timeout prevent a whole-repository survival verdict. The environ count=2 failure independently reproduces on the unchanged candidate. Those failures are not credited as behavioral kills of A3. The three relevant packages pass both repetitions within that broader attempt as well.

The new `TestReviewAttachAuthorizationMembers/input` passes twice on the original candidate and fails twice under A3. Under the mutant, Execute eventually returns `terminal_backend_unauthorized` from CheckAttachResult, but the durable writable receipt has already been committed. An assertion about the returned error alone is insufficient: the forbidden effect must not exist. Commit the composed negative test and its narrowing row, asserting absence of the receipt as well as refusal. Do not fork the landed authorization gate.

Two additional plants (expiry exactly at its instant, and local transport against a mesh authorization) survive the tmuxserver suite but are killed by owner suites. These are composed-witness limitations, not undiscovered production vulnerabilities, and are not separate blocking findings. Their new composed killers pass baseline and fail twice under each plant. A stopped-source attach narrowing is already killed by the committed tmuxserver suite.

## P3 — two reported narrowing rows are positive-path wiring mutations

`N-lifecycle-restore-authkind` and `N-lifecycle-terminate-authkind` replace the required effect-loop authorization kind with `control`, while the entry still requires `restore` or `force_stale`. Their raw kills are successful-operation tests now receiving `terminal_backend_unauthorized`; they do not demonstrate admission of a previously forbidden grant through the effect-loop gate. The harness notes accurately describe the refusal, but the aggregate labels both as narrowing rows and the census symmetry line counts them as the second authkind side.

Keep these useful wiring checks as supplementary positive-path mutations. Do not count them as admitting narrowings. Provide a genuinely admitting effect-loop authorization mutant with a negative witness, or label that side as an explicit bound. Recount classifications from actual plant behavior; this review does not certify the advertised 277 narrowing count merely because 277 row names begin with N-.

## Revision-5 disposition and retained behavior

- P1-A closure-report loss: **disposition accepted as the expressly authorized scoped bound**, not fixed. B43 names `internal/terminstance.ReceiptStore` as owner of completed-closure enumeration and this leaf as consumer. It explicitly discloses writable reopening until identical retry after outcome-report loss. The rework brief permits that owner/contract handoff; this review does not demand a fourth advisory guard or call it implemented.
- P1-B attach/quiesce ordering: the late-admission and both overlap regressions pass. The lock is per Lifecycle; B42 accurately limits cross-process exclusion and already-issued vectors. The new P1 above is a distinct stale-generation arm of the effect admission.
- P2-A read failures: all three committed production-entry witnesses pass, and their shipped narrowings are replayed in the battery. Attach custody now has a valid-body witness.
- P2-B importer corpus: independently derived 34 production importers / 36 including test importers. The fixed-corpus driver was rerun unchanged on base and candidate: **198 shared input keys, zero moved, zero base-only, seven declared candidate-only**. All 36 packages are represented, including cloneproject, sshtransport and crashgate. The seven additions are CheckSocketLength short/long and ClassifyStatus absent/live-tri/negative/parked/quiescing-memory.
- P3 census: corrected. A = 24×11, 50 measured/4 bound/210 unreachable; B = 24×21, 10/0/494; D = 140×21, 195/182/2563. C derives 140×11 = 1540 unreachable cells. Total **255 measured / 5248**, 186 bound, 4807 unreachable. B and D mirror the matrix byte-for-byte; the matrix references frozen predecessor Table A. These are mechanically reproduced classifications, not a claim that I independently proved every attribution.

All eight named operation entries execute. Exec/probe/spawn adapters and socket crash/idempotency witnesses are present and pass; B18, B16, B19/B20 and the foreground fresh-decoy witnesses remain dispositioned. The after-restore local/remote/parked, lapsed-grant offer, expired-answer and non-cooperative-refresh tests execute through ExecuteWrapperRestore and pass. Existing bounds (including transport/provider integration, missing wrapper signal, outcome authority and cross-process exclusion) remain bounds. The candidate extends termbind for successor minting and composes landed terminalbackend, terminstance, axpane and fencing rather than replacing their owners. Global traceability registry is untouched.

## Coverage actually measured

**8 of 8 operation entries driven** by passing candidate tests:

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

**34 of 34 AC rows (59–92) have all named witness sets resolved and executed: 232 distinct referenced test names pass.** This is witness execution coverage, not acceptance of every claim: attach's generation/deadline and authorization-effect completeness fail the attacks above. In particular row 85 is not behaviorally satisfied. No helper-only witness or stated bound is promoted to full end-to-end provider behavior.

## Validation and adversarial evidence

Firsthand on isolated exact-candidate copies:

- Full repository `go test ./... -count=1 -json`: exit 0, 45 packages, 24,365 passing test events, 14 skips outside the changed packages.
- No-tmux PATH: 1000 passing events across tmuxserver and termbind, zero failures/skips. The PATH has a go-only symlink directory plus system bins; tmux cannot resolve. Process snapshots found no process whose executable name was tmux; no user process was stopped. Zero rows claim a real-tmux witness.
- Native vet, Windows amd64 vet, changed-package race and repository-wide coverage: exit 0 each. gofmt output empty; live diff-check clean. Exact commands, elapsed times and host load are recorded. Go 1.25.5 on darwin/arm64; another Story runs on this host, so no timing comparison is inferred.
- Broader A3 run: exit 1 after 322s under concurrent load; environ count=2 refusal-audit failures and sessquery timeout at 300s. This is not a green whole-repository mutation run and is not used to claim one.
- Original-source generation and deadline regressions: two intended failures each under `-race`, no race diagnostic. Original-source authorization-member and valid-body custody probes pass twice.
- Four new custody entry narrowings (quiesce, boundary, stop, stale termination) are killed twice by committed tests, mostly through rerouted malformed-body refusals. The reviewer additionally supplies valid bodies and proves each weakened entry actually executes forbidden effects under an unsafe root; its targeted killer fails twice. This is the required different-entry attack on the shared custody gate.
- Four independent admission narrowings plus a harmless applied control: stopped-source transition killed; expiry/input/transport survive the committed tmuxserver suite twice; owner tests additionally kill expiry and transport. The input-escalation gap is P2-B. Each corresponding composed killer passes original and fails its mutant twice. Both independent harmless controls survive twice. Applied patches and raw logs contain subprocess exits; no unapplied or compilation failure is counted as a kill.
- Shipped harnesses rerun twice: **281/281 expected tmuxserver verdicts per pass** (280 KILLED, one control SURVIVED), and **40/40 termbind verdicts per pass** (39 KILLED, one control SURVIVED). **321/321 expected per pass; 642 total verdicts**, zero ERROR/MISMATCH. Every tmuxserver plant has a raw log for each pass; termbind logs contain 40 raw subprocess blocks and exits per pass. The N/D classification caveat is P3, not a harness-execution mismatch.

Configured suite count is **30**, read from candidate config. The exact CR-validation attachment reports `required=30 green=30 failed=0 missing=0`, but truncates 5,929,443 bytes. The producer's separate suite logs report all 30 exits; 343 archived files pass their SHA-256 manifest. I retain those as producer/handoff evidence and do not claim independently rerunning every configured fuzz/cross-build/full-repository-race command. The independently executed checks and attacks above are the basis of this rejecting verdict. Early reviewer setup attempts (incorrect scratch cwd / no tests matched) are not counted as test evidence.

Audit: live non-board files match the exact candidate; HEAD remains d4bd91d0, local main is zero commits ahead, task-board.config.json equals base, and there is no registry delta or Python cache artifact in the candidate. Remote main is 40bb8c9f89a5430c556e8746e1dca9769f4b3c47. The 23 overlapping base/main paths are predecessor Story changes; 16 retain checkpoint bytes and seven are intentional leaf changes. No out-of-scope revert was found. No commit, branch switch, checkpoint, integration or commit acknowledgement occurred.

Evidence archive: `TASK-260830-1c28dz_review-evidence-rev6.tar.gz`. It includes test/mutation logs, new probe sources, exact patches, independent outcome streams and census/importer/byte audits. The task verdict serves as the read-only review logbook record; live LOGBOOK.md is unchanged.

## Rework scope (for the producer)

1. Revalidate live generation at attach's effect admission after waits; keep closure synchronization and add the concurrent regression.
2. Enforce the operation deadline after admission/lock waits before committing a new attach receipt; preserve valid-authorization versus expired-operation distinction.
3. Commit the false-authorization/true-input production-entry negative test, asserting no durable receipt, plus its narrowing mutant. Keep the shared authorization owner and account for delegated gates in the evidence.
4. Correct the authkind mutation classifications and effect-loop evidence described in P3.
5. Retain the authorized B43 owner/contract handoff, successful rev6 fixes, complete importer corpus and corrected census. Narrow any claims still exceeding the measured behavior. Leave the candidate uncommitted and the final registry/integration work to its assigned owners.
