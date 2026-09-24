# TASK-260830-1c28dz — revision 5 independent review

Verdict: **CHANGES REQUESTED** — two P1 findings, two P2 findings, one P3 finding. Route to `to-dev`; do not accept revision 5.

Reviewed immutable candidate tree `1d4067513dbda9296b3889b055f4b31d7490d08a` against base `d4bd91d0e740285b21b8d65bf6f074c8009b04ae`. Authority: repository-pinned SPEC v0.7.0, §4.2 and §4.C–E, with §3.2 custody. The task's v0.5.0 citation is stale. Disposable archive copies held every test and mutation; the live Story source, index, branch and HEAD were not edited. The brief's statement that earlier revisions had no review conflicts with the actual attached verdicts; the verdicts were used.

## P1-A — a failed closure-report write still reopens input

`lifecycle.go:558–559` returns an integrity failure when the outcome cannot be installed, after the engine has committed input closure and its receipt. `ops.go:174–183` admits writable attach when the report scan finds no closure. It does not distinguish no closure from closure committed but its report failed. Advisory memory being ignored does not solve this missing-proof window.

`TestReviewOutcomeWriteFailureMustNotReopen` first drives `Lifecycle.Execute(attach)` successfully, then drives `Execute(quiesce-input)` with a StateHooks.AfterStage failure restricted to `/outcomes/`. Both `lock-session` and `detach-client` execute successfully; quiesce returns `terminal_backend_integrity_failure` at `quiesce outcome image`. After removing the fault, a new `Execute(attach)` succeeds with `InputAuthorized:true` and a vector without `-r`. No production source was mutated. The regression fails twice, including in the race-enabled run.

This is the same closure-authority invariant as revision 4, at another ordinary storage-failure boundary. Row 79 explicitly claims a lost barrier write fails closed. B34 describes only a re-observed timestamp after a receipt/report crash; it does **not** disclose that input can reopen before retry. The outcome proof cannot be the only source of admission if failure to create that proof is treated as permission. Preserve pending/committed closure authority across every report-write/crash boundary and drive attach before recovery, not only the recovery retry. If this requires the landed receipt owner's contract, name that owner and the scoped change instead of another advisory guard.

## P1-B — an in-flight attach can commit after successful quiescence

`ops.go:94` reads the barrier once; `ops.go:100` obtains server admission; `ops.go:113` then creates the durable writable client receipt without synchronizing with closure or revalidating it. Quiescence may complete between the first check and that effect.

`TestReviewAttachMustRecheckClosureBeforeReceipt` runs `Execute(attach)` in a goroutine and pauses its ServerAdmission adapter using channels. While attach is paused, the main goroutine completes `Execute(quiesce-input)`, including its durable report. Releasing the attestation call lets attach return success with writable authorization and a vector without `-r`. This deterministic interleaving fails twice under `-race` with **no data-race diagnostic**: it is a logical admission race, not an unsynchronized Go-memory access. The earlier nested-callback version reproduces the same ordering; the archived final source uses two goroutines.

Make closure and writable-client admission obey one ordering contract at the effect boundary. Merely moving the same check later leaves another check/use interval. Include the production returned-vector handoff in that contract. The review does not claim a raw provider or real tmux process was launched; it proves the scoped production entry authorizes the forbidden writable attach.

## P2-A — three new read-error clauses survive the committed suite

Four independently authored, single-site admitting narrowings and one applied harmless control were run against the full committed tmuxserver suite with `-count=2`. Three narrowings survived both repetitions. Every targeted reviewer killer passes the original production code and fails twice under its plant. No setup failure, compilation failure, timeout, or unapplied edit counts as a kill.

| Plant | Admission introduced / production call site | Committed suite, twice | Reviewer killer, twice |
| --- | --- | --- | --- |
| R1 | EISDIR reading the incarnation document becomes absence / Execute(attach) → checkAttachQuiesced → LookupIncarnation | SURVIVED, exit 0 | TestReviewKillIncarnationReadFailure: KILLED, exit 1 |
| R2 | ENOTDIR reading the outcomes directory becomes absence / Execute(attach) → QuiesceBarrierProven | SURVIVED, exit 0 | TestReviewKillOutcomeDirectoryReadFailure: KILLED, exit 1 |
| R3 | EISDIR reading one outcome document is skipped / Execute(attach) → QuiesceBarrierProven | SURVIVED, exit 0 | TestReviewKillOutcomeFileReadFailure: KILLED, exit 1 |
| R4 | Unsafe-root refusal bypassed only for attach at the lifecycle entry; other entries/classes remain guarded | KILLED, exit 1 | TestReviewKillAttachCustody: KILLED, exit 1 |
| C | Comment-only edit | SURVIVED, exit 0 | All four reviewer killers pass, exit 0 |

R3 stages a `.json` symlink to a directory so the entry is not skipped by DirEntry.IsDir, and the production ReadFile returns EISDIR. R4 is the required different-entry attack on a shared gate: producer rows explicitly mutate the create/status/restore wiring; this plant targets attach. The existing `TestExecuteEachOperationRefusesWritableRoot/attach` kills by rerouting to malformed-body refusal. The reviewer additionally supplies a valid body and proves actual admission if that gate is weakened.

The three survivors are missing clause witnesses, not evidence that unmodified production ignores those particular errors. Ship their production-entry tests and narrowing rows; do not infer read-error coverage from the existing identity/empty-key tests.

## P2-B — importer outcome coverage improved, but the exclusions are false

Independent `go list -json ./...` derivation reproduces 34 production importers, 36 with test-only importers, over the nine assigned owner packages. The supplied fixed-corpus driver was rerun unchanged on base and candidate: **193 shared keys, zero moved, zero base-only, seven declared candidate-only**. The keys are genuinely `(package,entry,input)`, with outcomes/refusals as values. The seven additions are CheckSocketLength short/long and ClassifyStatus absent/live-tri/negative/parked/quiescing-memory. This closes the earlier test-inventory-only defect for the probed corpus.

It probes 33 of 36 packages (32 of 34 production packages). The stated reason for excluding cloneproject, sshtransport and crashgate is “no exported pure callable surface.” That is false:

- `cloneproject.Normalize` has deterministic pre-effect no-fetcher and no-sink refusals.
- `sshtransport.New(config.Snapshot{})` refuses an invalid snapshot without starting SSH or touching the network.
- `crashgate.Lookup` is a pure known/unknown boundary lookup.

The reviewer's five-row supplemental driver calls these entries on both trees, exits 0 twice, and records identical outcomes. Thus no external/platform constraint prevents the requested comparison. Put representative input/outcome rows for these packages into the delivered corpus, and retain any genuine effectful-success bounds with accurate reasons. The reviewer's supplemental five rows are evidence against the exclusion claim, not a claim to have completed the producer's whole behavioral corpus.

## P3 — census denominator and remaining prose are inaccurate

Mechanical recount of the expanded tables reproduces:

| Table | Shape | Measured | Bound | Unreachable |
| --- | ---: | ---: | ---: | ---: |
| A | 24 × 11 = 264 | 50 | 4 | 210 |
| B | 24 × 21 = 504 | 10 | 0 | 494 |
| D | 137 × 21 = 2877 | 190 | 183 | 2504 |

But Table C in both TRACEABILITY and the conformance matrix still says **129 × 11 = 1419**, while Table D now enumerates 137 new gates. Eight new gate rows are omitted from the old-entry cross-product: 88 cells. For the stated complete census, C is 137 × 11 = 1507 and the total is **5152**, not 5064. Under its existing all-unreachable classification that would be 250 measured + 187 bound + 4715 unreachable, subject to actual classification. Recount the gate set once and derive both products from it. No claim is made that every one of the 250 cells was independently re-attributed.

Row 72 also describes attach as carrying “lease members,” although §4.C's AttachAuthorization is explicitly ownership-neutral and contains no lease. Correct it. README's unqualified refusal of new writable attaches and row 79's lost-write claim are contradicted by the P1 evidence; narrow the claims until the invariant is implemented.

## Retained fixes and scope

Revision 4's stale-active/lost-state-write and failed-stop regressions now pass. The previous restore bootstrap and instance divergence tests pass, as do the new session/backend/generation joins. Prior lease-ID/epoch and expired/non-cooperative refresh checks remain green. The review's new P1-B is the attach race; it is not a claim that the previous restore-identity finding remains open.

The eight operation entries, exec/probe/spawn adapters, socket crash/idempotency tests, B18 derived path bound, B16 ambient decision, B19/B20 custody rows and P3-A fresh-decoy attestation witnesses are present and execute. The candidate extends the landed termbind owner for successor minting and composes terminalbackend, terminstance, axpane and fencing. It does not touch the global traceability registry. B27 acquisition composition, B33 absent wrapper boundary signal, B37 absent mesh transport, B38 rival-union modeling, B39 declared undriven cells, and B40 recovery limitations remain stated bounds; this review does not turn them into implemented end-to-end provider or remote-offer behavior. Zero real-tmux witnesses are claimed.

## Coverage actually measured

**8 of 8 named operation entries driven** by passing candidate tests:

| Operation | Named candidate test | Production call site |
| --- | --- | --- |
| create | TestExecuteCreateInteractive | Lifecycle.Execute → executeCreate → terminstance.Engine |
| attach | TestExecuteAttach | Lifecycle.Execute → executeAttach |
| status | TestExecuteStatusPresent | Lifecycle.Execute → executeStatus → backend.ObserveStatus |
| quiesce | TestExecuteQuiesce | Lifecycle.Execute → executeEngineOp → closeInput |
| safe-boundary | TestExecuteBoundary | Lifecycle.Execute → executeEngineOp → observeBoundary |
| stop | TestExecuteStop | Lifecycle.Execute → executeEngineOp → confirmClosed |
| stale termination | TestExecuteTerminate | Lifecycle.Execute → executeTerminate → runLifecycleEffects |
| restore | TestExecuteRestore | Lifecycle.Execute → executeRestore → runLifecycleEffects |

**34 of 34 second-leaf AC rows (59–92) have their named witness sets resolved and executed; all 225 distinct referenced test names pass.** This is witness/reference execution coverage, not acceptance of every claimed property. Rows 79/81 and the barrier construction/census claims fail the additional attacks above. Helper-only clauses were not relabeled as full end-to-end behavior.

No-tmux PATH run: **723 tmuxserver + 270 termbind = 993 passing test events, zero failures, zero skips**. `command -v tmux` under that PATH exits 1. Process inspections found no tmux process; tmuxserver.test is not counted as tmux. No operator process was stopped. Environment and load samples are attached; timing is not compared against a concurrent Story.

## Validation and evidence provenance

Firsthand checks on isolated copies of the exact candidate:

- `go test ./... -count=1 -json`: exit 0; 45/45 packages pass, 24,358 passing test events and 14 skips outside the changed packages.
- Native and Windows amd64 `go vet ./...`: exit 0 each. Changed-package gofmt output empty; live `git diff --check` exits 0.
- `go test ./internal/tmuxserver ./internal/termbind -race -count=1 -timeout=5m`: exit 0, no races.
- `go test ./... -cover -count=1 -timeout=5m`: exit 0; tmuxserver 88.0%, termbind 83.5%.
- Two unchanged-production regression tests, twice under `-race`: exit 1 with four intended assertion failures, no data-race diagnostic. Initial reviewer setup/path and compile errors are retained as failed setup attempts and not counted as behavioral evidence.
- Independent plants: R1–R3 survive the committed suite twice; R4 is killed twice; all four targeted killers pass baseline and fail twice under their corresponding plants; applied control survives both suites twice. Raw logs include subprocess exits and elapsed times.
- Shipped harnesses: **273/273 tmuxserver expected verdicts per pass** (269 narrowing and three supplementary plants killed; one harmless control survived), and **40/40 termbind expected verdicts per pass** (39 killed, one control survived). All calls exit 0; zero mismatches/errors. Combined **313/313 expected per pass, 626 verdicts total**. Each tmuxserver pass has 273 raw per-plant logs; termbind logs contain per-plant raw subprocess blocks and exits. The first tmuxserver pass finished in one bounded call; the second was split into 140/133-row sequential calls. No mutation touched the live worktree.

The configured suite has 30 commands. The existing CR-validation log reports `required=30 green=30 failed=0 missing=0`, but omits 5,928,743 bytes and exposes only 21 individual exit records. That is producer/handoff evidence, not a reviewer rerun. This rejecting review does not claim independently replaying the configured full-repository 25-minute race, fuzz, and cross-build suite; the firsthand subset is listed above.

Audit: all 900 non-board live files match the immutable candidate. HEAD remains d4bd91d0; config equals the base byte-for-byte; registry delta is empty; no Python cache artifacts exist in the reviewed copies. Remote main resolves to 40bb8c9f89a5430c556e8746e1dca9769f4b3c47. The 23 base/main overlapping paths are predecessor Story work: 16 remain byte-identical to the checkpoint; seven are intentionally changed in this leaf (README, LOGBOOK, tmuxserver TRACEABILITY, errors, harness, and both ownership files). No out-of-scope sibling delta was reverted. Local main is zero commits ahead of worktree HEAD. No commit, checkpoint, integration, or commit acknowledgement occurred.

Evidence archive: `TASK-260830-1c28dz_review-evidence-rev5.tar.gz`, containing raw logs, regression and killer source, exact independent patches, both shipped harness passes, outcome streams/drivers, census/closure/byte audits and checksums. Acceptance items contradicted by these findings are unchecked. Notes and verdict are board resources; LOGBOOK.md remains untouched under the read-only reviewer contract.

## Rework scope (for the producer)

1. Preserve closure authority when outcome persistence fails after closure effects/receipt commit; include writable attach before retry and all crash boundaries. State the receipt-owner change if the invariant cannot fit this leaf.
2. Synchronize writable attach authorization/effect with quiescence, including the returned-vector handoff. Commit the deterministic concurrent regression.
3. Add the three read-error killers and admitting narrowing rows; preserve the successful cross-entry custody behavior.
4. Complete the importer outcome corpus for the three omitted packages; replace inaccurate exclusion reasons with actual bounds where needed.
5. Derive both census dimensions from one gate set, correct the denominator and attach-body prose, and update claims and evidence. Keep the candidate uncommitted; leave registry work to g0pcnt and integration to the authorized producer lifecycle.
