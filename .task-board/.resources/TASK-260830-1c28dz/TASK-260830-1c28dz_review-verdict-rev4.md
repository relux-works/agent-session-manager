# TASK-260830-1c28dz — revision 4 independent review

Verdict: **CHANGES REQUESTED**. Route to `to-dev`; do not accept CR revision 4.

Reviewed immutable candidate tree `0f3be651cf65aa9ed9b8458541dbd3c516e426a0` against base `d4bd91d0e740285b21b8d65bf6f074c8009b04ae`. Normative authority is the repository-pinned SPEC v0.7.0, §4.2 and §4.C–E, with §3.2 custody. The task's v0.5.0 reference is stale. All probes and mutation edits ran in disposable git-archive copies. The live Story source, index, branch and HEAD were not modified.

## P1-A — quiescence remains bypassable through stale state and failed operations

`ops.go:169–175` trusts any existing state other than `quiescing` and never consults the durable closure proof in that case. The new recovery only fixes the no-record case. `lifecycle.go:433–437` also overwrites state from failure results, including a request's stale source state.

Two regressions fail against unchanged candidate production code:

- `TestReviewExistingActiveMemoryCannotReopenQuiescedInput`: first call production `Execute(attach)` to record active; run quiesce with an instance-write AfterStage failure, after lock-session and detach-client execute and the closure report persists; remove the hook; a new writable attach **succeeds**, returning `attach-session -t <instance>` without `-r`. An existing active document is the normal case, not an exotic corrupt-store input. The committed revision-4 test instead starts with no document.
- `TestReviewFailedStopCannotEraseQuiescence`: successful quiesce, then a request-stop with expired authorization and stale `Source=active`; stop correctly fails, but its failure result overwrites the barrier with active. A subsequent writable attach **succeeds**. This case requires no storage fault at all.

These violate the input-closure effect and contradict row 79's lost-write fail-closed claim. Preserve the authoritative closure barrier across failed writes and failed/stale operations. Do not let advisory memory, caller-carried source state, or failure reporting reopen it. Add the complete multi-call sequences through `Execute`, not just direct store tests.

## P1-B — wrapper decision and executed restore still name different operations/instances

`wrapper.go:169–178` joins only lease ID and epoch. The independent `req.Decide` identity/descriptor and `req.Restore` mutation context remain unjoined before `lc.Execute` at line 145. Matching a lease does not establish that the effect is the restore the decision authorized.

Two unchanged-production regressions fail:

- `TestReviewWrapperBootstrapMustBindExecutedRestore` supplies a valid launch decision for bootstrap `...333333333331`, while the restore names bootstrap `...444444444441`. The entry returns launch with nil error and executes new-session for the latter.
- `TestReviewWrapperInstanceMustBindExecutedRestore` builds a valid decision descriptor for instance `...555555555552`, while the backend restore targets `...222222222221`. Again launch, nil error and new-session follow.

The previous lease divergence tests now pass; that narrow fix is retained. Bind the complete relevant session/bootstrap/instance/backend-generation identity and materialization decision to the effect request, using the landed owners. Refuse disagreement before effects. §4.2 step 3 and §4.C's identity-bearing mutations require one composed authorization, not two independently valid descriptions.

## P2-A — the complete importer outcome grid is still missing

The new runtime entry grid is real progress, but covers only tmuxserver and termbind. Independent `go list -json ./...` derivation again yields **34 production packages**, or **36 including test-only importers**, over the nine assigned composing packages. The producer still uses `(package,test) -> pass/fail/skip` for that closure. That is supplementary test inventory, not `(package,entry,input class) -> outcome/refusal` evidence requested in the brief and rev3 verdict.

The new entries' candidate-only ADDED rows cannot show the stated admission/refusal changes between prior implementations. The result's claim that 27 retained entry keys plus 23,054 passing test keys prove no pre-existing behavior changed exceeds the measurement. Supply the fixed-corpus production-entry outcome streams over the requested closure, identify each moved/missing key, and retain test-verdict inventories under their accurate label. The package-local pre-existing entry-grid rerun is recorded separately below; it does not substitute for the missing closure.

## P2-B — unmeasured refusal clauses survive the committed suite

Each following plant is an admitting narrowing, leaves other refusal members intact, applies at exactly one site, and ran against the full committed tmuxserver suite twice. Each **SURVIVED twice, exit 0**. Each targeted reviewer killer passes against the original production code and **KILLS twice, exit 1 with a named test failure**, proving actual admission rather than a build failure. An applied comment-only control survives both full-suite and reviewer-test passes.

| Plant | Admitted member / production entry | Reviewer killer |
| --- | --- | --- |
| R1 | malformed attached count `0junk` past the Atoi-error clause; negative-count refusal remains / Execute(status) → probeAttached | TestReviewKillAttachedParseError |
| R2 | exactly `{}` past missing idempotency key in barrier scan / Execute(attach) → QuiesceBarrierProven | TestReviewKillEmptyBarrierKey |
| R3 | writable-root custody refusal only at ServerProber.Probe; helper, lifecycle and spawner sides remain enforced | TestReviewKillProberRootCustody |
| R4 | EISDIR state-read failure treated as absent; other read failures remain errors / Execute(attach) → InstanceStates.Lookup | TestReviewKillStateReadFailure |

R3 confirms the **already-declared PRB/rootwrite B39 bound**; it is not misrepresented as a false measured cell. The other three expose clause gaps: the census's statusnegative row measures the negative count, not its parse-error sibling, and the new barrier scan's empty-key and the state-read-error branches have no corresponding measured narrowing. Ship tests and rows for the gaps or accurately disposition permissible bounds; do not infer coverage of both clauses from one killed compound-condition mutant.

## P3 — residual inaccurate prose

The table recount is fixed: A=264, B=504, C=1419, D=2709; total **4896 cells = 238 measured + 181 bound + 4477 unreachable**. Table C's 18 U2c names now agree with its main prose. However `internal/tmuxserver/TRACEABILITY.md:771`, bound B27, still says “19 Table C U2c cells.” Correct that remaining reference. Table C remains represented by prose rather than individually expanded cells; its stated dimensions were checked separately from the parsed A/B/D tables.

The producer results also say no committed test dials a real socket. `TestUnixDialerLiveStaleUnknown` in `probe_unix_test.go:79` creates a real local Unix listener and calls `UnixDialer.Dial`. This is a synthetic local socket, not a real tmux server; correct the sentence without confusing those two properties.

## Retained improvements and explicit limits

The expired and non-cooperative refresh cases now pass through the enforced timeout, including the both-ready select race; the previous lease-ID/epoch divergence refuses. Revision-3's four status/argv clause killers and the spawner custody killer are now in the committed suite. Read-only attach, no-follow custody and OSRunner signal/unknown handling remain covered. Socket crash/idempotency, B18 derived socket length, B19/B20 tested custody entries, B16 deliberate ambient behavior, and P3-A fresh-decoy attestation witnesses pass. Exec/probe/spawn adapters exist.

The second leaf extends the landed termbind owner with successor minting and composes terminstance, terminalbackend, axpane and fencing; it does not modify `internal/traceability`. B27 acquisition composition, B33 missing wrapper boundary signal, B34 receipt/report crash window, B37 absent production mesh transport, B38 rival-union modeling and B39 reachable-undriven cells remain explicit limits. The launch result recreates a parked wrapper; tests do not establish a real provider resuming or a remote offer being displayed/accepted. Zero real-tmux witnesses are claimed.

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

**34 of 34 second-leaf AC rows (59–92) have their named witness sets resolved and executed; 205 of 205 distinct referenced names pass.** This is execution/reference coverage, not acceptance of every asserted clause: rows 79/81/88/92 are affected by the regressions above. The claimed 238 measured census cells were recounted, not all independently re-attributed; no fully verified 238/4896 conformance claim is made.

No-tmux PATH run: **701 tmuxserver + 270 termbind = 971 PASS, zero FAIL, zero SKIP**, including subtests. tmux cannot resolve; process inspection finds no tmux process (Go's `tmuxserver.test` process is correctly distinguished). No operator process was stopped. All eight operation witnesses run without real tmux.

## Validation and provenance

Firsthand checks, on isolated copies of the exact candidate production tree:

- `go test ./... -count=1 -json`: exit 0; 45/45 packages pass, 24,336 passing test events and 14 skips outside the changed packages.
- Native and Windows amd64 `go vet ./...`: both exit 0. Changed-package gofmt output is empty; live `git diff --check` exits 0.
- `go test ./internal/tmuxserver ./internal/termbind -race -count=1 -timeout 5m`: exit 0. This was a separate targeted reviewer race check, not a substitution for or rerun of the configured full-repository 25-minute race command.
- `go test ./... -cover -count=1 -timeout 5m`: exit 0. tmuxserver coverage 87.7%, termbind 83.5%.
- Four unchanged-production regression tests fail, exit 1, and reproduce with `-count=2` (eight failures). These failures are the evidence for P1-A/B, not green gates.
- Independent mutation battery: four admitting plants × two committed-suite runs = eight survivals; four targeted killers × two runs = eight kills; all four killers pass baseline. The harmless control survives twice against the committed suite and twice against the reviewer killers. No build failure, timeout or unapplied plant counts as a kill.
- Independently reran the producer's pre-existing entry-grid drivers on base and candidate: tmuxserver 18/18 retained keys, termbind 9/9 retained keys, zero outcome/code/digest moves. Four driver commands exit 0. This verifies only those two packages.

- Shipped tmuxserver harness twice: **251/251 expected verdicts each, exit 0**; 250 KILLED (248 N rows plus two supplementary D rows), one harmless SURVIVED control per pass, per-plant raw logs. Shipped termbind harness twice: **40/40 expected verdicts each, exit 0**; 39 KILLED and one harmless SURVIVED control, verbose raw subprocess blocks. Combined **291/291 expected per pass**, zero mismatches.

Existing producer evidence is never relabeled as a reviewer run. The configured validation contract contains 30 commands; its CR log reports 30 green but omits 5,925,690 bytes and exposes only 21 individual exit records. Full configured race/fuzz/cross-build replay is not independently claimed in this rejecting review.

Final baseline audit: 899 non-board live files match the candidate tree; HEAD remains d4bd91d0; config equals the base byte-for-byte; no registry changes or Python cache artifacts exist in the candidate. Remote main resolves to 40bb8c9f89a5430c556e8746e1dca9769f4b3c47. Base/main differences outside the leaf are the predecessor's tmuxserver implementation and are preserved byte-for-byte from the checkpoint, not reverted sibling work. Local main is zero commits ahead of this worktree HEAD. No commit, checkpoint, integration, or commit acknowledgement occurred.

## Rework scope (for the producer)

1. Make closure authoritative across stale active memory, state-write failure, and failed operations carrying stale source state; commit both reviewer sequences and narrowing witnesses.
2. Bind the wrapper decision's complete relevant identity/materialization to the restore effect, preserving lease and timeout fixes. Commit the bootstrap and instance mismatch regressions.
3. Complete the requested production outcome grid over the mechanically derived importer closure and remove unsupported equivalence claims.
4. Add the missing parse-error, barrier-key and state-read-error witnesses/rows; optionally close the already-declared prober custody bound with the supplied killer. Keep per-entry attribution honest.
5. Fix B27's stale count, update the matrix/results/logbook, and rerun relevant and configured gates before the next managed handoff. Keep the candidate uncommitted and leave registry work to g0pcnt.

Evidence archive: `TASK-260830-1c28dz_review-evidence-rev4.tar.gz`. It contains raw logs, standalone regression/killers, exact independent mutation patches, grid streams, census/closure/byte audits and a SHA-256 manifest. Unsupported acceptance checklist items are unchecked; changes requested is not represented as completed acceptance.
