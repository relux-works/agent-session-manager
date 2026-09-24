# TASK-260830-1c28dz — independent review of revision 9

Verdict: **CHANGES REQUESTED**. One P1 and two P2 findings. Route to `to-dev`; do not accept revision 9.

Candidate tree: `ddae5b7781152de3ffa737879f970129694d8fea`; base/checkpoint: `d4bd91d0e740285b21b8d65bf6f074c8009b04ae`. The downloaded CR patch SHA-256 is `b97ab0e74a4289dff536c675d7ccdf0773ec964ac76f56701ad712e1dc56a3e8`, equal to the assignment. Authority: pinned SPEC v0.7.0 §4.C, §4.2 and §3.2. The task's v0.5.0 scope and the review brief's denial of earlier verdicts are stale. Read the actual rev8 verdict and superseding rev9 rework brief. Client liveness remains assigned to TASK-260922-vcx6yo; no finding here widens that handover.

## P1 — peer identity corruption can still become an empty census

Production call site: `Lifecycle.Execute(attach) → executeAttach → checkAttachOverlap → AttachStore.Peers`, `internal/termbind/overlap.go:65-72`.

The new directory check is correct, but a decoded receipt is only checked against the requested session and instance. Its client ID is never checked against the `<client UUID>.json` filename being read. The code then excludes any receipt whose *content* names the requesting client. A valid receipt stored at another client's receipt path therefore disappears from the census instead of refusing corruption. The landed `AttachStore.Lookup` already performs the missing client-pair comparison at its keyed read boundary.

`TestReviewAttachPeerFilenameMismatchFailsClosed` drives attach for B, moves the valid B receipt bytes to A's receipt filename, then drives B again with `multi_attach` absent. B no longer has a recorded receipt at its own key, so this is not a validated durable replay. Actual result: success, a newly persisted B receipt, `InputAuthorized:true`, and a writable attach vector. Expected: refuse the corrupt receipt namespace before creating a receipt/vector. The production-entry probe fails twice under `-race`, with no data-race diagnostic, on restored candidate production bytes (`probes-detailed-{1,2}.log`).

**Repeat-of:** rev8 P1's receipt-census integrity boundary, at the decoded client/filename identity arm instead of the directory arm. B44 defers liveness; it does not authorize treating invalid identity evidence as no peers. The producer fixed the concrete directory case but not the complete pair identity invariant.

Repair: validate the enumerated receipt's client identity against its durable filename before applying the requesting-client exclusion. Retain the session/instance checks and staging/non-receipt rules. Drive this through Execute, assert no receipt/vector/input authorization, and add an admitting narrowing witness.

## P2-A — post-escalation closure probe inherits the expired graceful deadline

Production call site: `Lifecycle.Execute(request-stop) → executeEngineOp → backend.confirmClosed → pollSession`, `internal/tmuxserver/backend.go:419` and `:469`; the graceful bound is assigned at `lifecycle.go:506`.

Revision 9 correctly adds context cancellation to read-only probes. However, the same `runner.deadline` is used for the initial graceful wait and for the fresh confirmation after `kill-session`. The initial wait has already exhausted that graceful deadline when escalation happens. The second `probeContext` is therefore immediately canceled, even though the operation deadline and authorization are still valid. An executor honoring cancellation cannot observe the newly closed session.

`TestReviewStopEscalationWithContextHonoringRunner` reuses the committed `TestExecuteStopEscalatesAfterGracefulTimeoutAlone` scenario: a 5ms graceful allowance, a presence observation advancing the fixture clock by 10ms, valid later operation deadline, successful kill, and an available marked-absence response. Its sole runner change is honoring `ctx.Err()` before delegating, matching `OSRunner`/`exec.CommandContext` cancellation semantics. Actual calls end at `[send-keys has-session kill-session]`; the final has-session is denied its opportunity to run, and Execute returns `terminal_backend_process_failed` / `unavailable`, not stopped with closure. It fails twice under `-race`, without a data-race diagnostic (`probes-detailed-{1,2}.log`). The committed positive control passes because `fakeRunner.Run` discards its context entirely.

**Repeat-of:** rev8 P2-A's runner-wait contract, now a regression in the positive escalation path caused by carrying the old wait bound into a new observation. SPEC §4.C's request-stop row requires observed closure and the implementation explicitly promises graceful-timeout escalation; context cancellation must not make that success path unreachable. B45 concerns commit waits, not this read-only observation.

Repair: distinguish the graceful wait budget from the post-escalation observation/operation budget. Keep the fresh authorization, generation and operation-deadline rechecks before kill; keep timeouts unknown and never manufacture closure. Add a context-honoring success witness for the escalation path alongside the existing hung-probe refusal witnesses.

## P2-B — complete-importer outcome evidence regressed to a name-keyed suite grid

The current results §7 labels both comparisons outcome-keyed, but its first comparison explicitly keys `(package, test)` top-level verdicts. `suite_grid.py` does exactly that. A same-named test remaining green does not prove that its admission/refusal outcome stayed unchanged.

The actual entry-outcome driver imports four packages: scalar, terminalbackend, terminstance and termbind. Independent Go import-graph traversal reproduces **34 production packages / 36 including test importers** in the required closure. Only three of those 36 are directly driven by this corpus; scalar is outside that reverse-importer closure. **33 of 36 closure packages have no entry in this outcome corpus.** The 28-key corpus was rerun on exact base and candidate and is identical, but cannot support the claim that no input changed across the complete importer set. `importer-audit.json` names every missing package.

**Repeat-of:** the composition evidence gap explicitly called out in the producer brief and earlier reviews. Rev8's independent reviewer reported a broader finite corpus, but that earlier measurement does not substantiate rev9's complete-set claim or replace the producer's required current-candidate artifact. This review does not claim to have rerun that older 198-key corpus: its source was not present in the downloaded rev8 review archive.

Repair: restore an outcome-keyed corpus over the complete computed importer set, with stable `(package, production entry, input)` keys and actual returned decisions/effects. Run the same inputs on base and candidate; name every moved admission/refusal class. Keep suite verdicts as supplementary regression evidence and state the finite corpus's limits honestly.

## Disposition of revision 8 and scope boundaries

- P1 directory-shaped receipt: the specific directory refusal now passes and its narrowing kills. The adjacent filename/content identity hole remains P1 above.
- P2-A in-flight stop polling: the new deadline tests for stop, terminate confirm and status pass; the shared widening is killed by each selected test alone twice. Preserve this repair; fix the new escalation regression without removing the bound.
- P2-B replay with another peer: input-authorized, read-only and conflicting retry legs pass. Existing authorization/generation checks remain in force. Preserve them.
- P2-C stale archive: closed. Current downloaded archive SHA-256 `fa7b00c29ce887ce3d2d566cbebd2f5f8328eab0ddb0c750b23b74021734fa56` matches the results; **640/640 manifest entries verify**. Current isolated logs and battery records are present. The results' isolated `producer-evidence-rev9.tar.gz` spelling in §1 differs from the actual generic attachment name correctly given in §9; this is not another stale-archive finding.
- P2-D false attribution: closed. Independently reran all 12 claimed killer selections plus the cancellation companion, twice, with one selected test per process. All 24 killer executions fail as claimed; both companions pass. Stop/quiesce effect-recheck symmetry and the four wait paths now have the stated evidence. Logs: `isolated/round{1,2}/`.
- B45 explicitly records the remaining single-shot commit-wait bound and its owners. It does not excuse P2-A, a read-only probe with an expired context.
- B18 socket length, B16 ambient behavior, B19/B20 custody, socket crash/idempotency and exec/probe adapter witnesses all execute in the passing named AC set. Structured attach checks the fresh admission/generation; no registry/CLI wiring is claimed here. The first-leaf foreground integration handover stays with g0pcnt.
- The code composes the landed authorization, receipt, binding, transition, scalar and fencing owners. AttachStore is extended with Peers and successor support; this review does not request a new liveness owner or parallel receipt model.

## Coverage and census actually measured

**8 of 8 operation entries driven by passing candidate tests:**

| Operation | Named witness | Production call site |
| --- | --- | --- |
| create | TestExecuteCreateInteractive | Lifecycle.Execute → executeCreate → terminstance.Engine |
| attach | TestExecuteAttach | Lifecycle.Execute → executeAttach |
| status | TestExecuteStatusPresent | Lifecycle.Execute → executeStatus → ObserveStatus |
| quiesce | TestExecuteQuiesce | Lifecycle.Execute → executeEngineOp → closeInput |
| safe-boundary | TestExecuteBoundary | Lifecycle.Execute → executeEngineOp → observeBoundary |
| stop | TestExecuteStop | Lifecycle.Execute → executeEngineOp → confirmClosed |
| stale termination | TestExecuteTerminate | Lifecycle.Execute → executeTerminate → runLifecycleEffects |
| restore | TestExecuteRestore | Lifecycle.Execute → executeRestore → runLifecycleEffects |

**34 of 34 AC rows (59–92) have their witness sets executed; all 261 distinct named tests pass.** Duplicate textual table copies were deduplicated by row number. This is witness-execution coverage, not acceptance of every row's behavior. Rows 79 and 81 have production counterexamples above. The n-of-8 operation ratio likewise does not establish full branch coverage.

Independent census count reproduces A: 24×11, 50 M / 4 B / 210 U; B: 24×21, 10 M / 0 B / 494 U; D: 148×21, 206 M / 186 B / 2716 U; C: 148×11 declared unreachable. Total **266 M-labelled / 5504 cells**, 190 bound and 5048 unreachable. Tables B and D mirror the conformance matrix byte-for-byte. These are reproduced classifications, not a guarantee that no omitted integrity gate exists.

The ordered wrapper restore tests all pass, including lapsed-grant remote attach/takeover, local resume, parking and bounded refresh. An independent ordering attack invoking backend restore before refresh/offer selection is killed twice by `TestWrapperRestoreLapsedGrantRemoteOffer`. This is supplementary ordering evidence, not counted as a gate narrowing.

Four independent per-entry custody narrowings admit only quiesce, boundary, stop or stale-termination past the still-present shared custody guard. Each is killed twice by the committed per-operation refusal test and the review's valid-body effect probe. A harmless applied comment control survives twice. These are the prior review's attack shapes independently executed on rev9, not reused verdicts.

## Validation and evidence

Firsthand on the exact candidate tree:

- Full `go test ./... -count=1 -json`: exit 0, 254.56s; all 261 named AC witnesses pass, 14 pre-existing skips outside the changed packages. No failing package.
- Restricted-PATH changed-package suite: exit 0, 35.36s, 1058 pass events, zero skips/failures. `tmux` lookup is absent; before/after `pgrep -x tmux` both return exit 1 with empty stdout/stderr. No acceptance row claims a real-tmux witness.
- Changed-package race suite: exit 0, 48.77s. The two adversarial production probes separately fail twice under race with assertion failures, no race diagnostic.
- Native vet: exit 0, 6.92s; `GOOS=windows GOARCH=amd64 go vet ./...`: exit 0, 27.00s. Gofmt produces no paths.
- Repository coverage: exit 0, 227.66s; tmuxserver 89.0%, termbind 83.3%. Host load is captured alongside every timed suite in `checks.jsonl` and `harness-runs.jsonl`; concurrent host work drove observed load as high as 164.23, and no timing failure was dismissed as load.
- The config contains 30 validation commands. The exact CR9 handoff attachment reports `required=30 green=30 failed=0 missing=0`, with actual command exits. Accepted as existing handoff evidence; this reviewer did not rerun every fuzz target, cross-build or the full-repository race command. The separate Windows vet above includes test files.
- Shipped harnesses: **296/296 tmuxserver and 42/42 termbind expected verdicts per pass, twice; 676 total verdicts**. Each pass has 335 KILLED and three expected SURVIVED (both harmless controls and the shadowed supplementary attach row). Names reconcile against the exact harness lists, with no missing, ERROR or MISMATCH row. Per-plant logs include subprocess exits; termbind verbose logs retain each raw block. Each Go killer is bounded at 300s and the harness is partitioned into at most 40-row subprocess shards with 540s bounds. Source restoration and zero Python cache artifacts were independently checked after both passes.
- The first two integrity-probe launch attempts ran before the disposable copy finished extracting and returned setup failure (`directory not found`). They are preserved as `integrity-{1,2}.log` and are not counted as behavioral evidence. All findings use the subsequent compiled, completed, restored-source race runs.

The current producer archive's 640 manifest entries all match. The exact CR patch and downloaded resources have digests recorded in `resource-digests.json`. Independent full-importer outcome coverage remains unverified as explained in P2-B; neither a green suite nor the 28 identical inputs is represented as a substitute.

All production review work used isolated archived copies. Live changed files equal the candidate, HEAD remains the checkpoint, task-board config equals base, and no `internal/traceability` delta exists. Remote main advertises `40bb8c9f89a5430c556e8746e1dca9769f4b3c47`; local main is zero commits ahead of this Story worktree. The 23 overlapping base/main paths are predecessor Story content: 16 retained checkpoint paths and seven intended leaf edits. No live product/index edit, commit, branch switch, rebase, checkpoint, integration or acknowledgement was performed.

This verdict and its board note are the read-only review logbook record. Raw logs, subprocess exits, timing/load, probes, plants, artifact digests, census/AC/importer audits and restoration checks are attached as `TASK-260830-1c28dz_review-evidence-rev9.tar.gz`. Existing large producer resources are referenced by digest, not repackaged as new evidence.

## Rework scope (for the producer)

1. Bind every enumerated peer receipt to its filename client identity before exclusion; fail closed on mismatch and add a production-entry negative plus narrowing.
2. Give post-escalation observation the correct remaining budget. Prove successful escalation with a context-honoring runner while retaining unknown-on-timeout and stale-fact refusal.
3. Restore complete-importer outcome-keyed comparison and correct the §7 evidence claim. Attach current raw corpus, inputs, outputs and package closure.
4. Preserve the accepted rev9 directory, peer replay, deadline and isolated-attribution repairs, and the client-liveness/registry owner boundaries. Update the affected matrix/TRACEABILITY/results claims and leave the candidate uncommitted for managed handoff.
