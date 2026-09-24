# TASK-260830-1c28dz — CR revision 2 review

Verdict: **CHANGES REQUESTED**, route to `to-dev`. No external blocker. Do not accept revision 2.

Reviewed base `1ba06e99090e3b7bf1aa2f4865914f4080a644ed`, immutable candidate tree `d11c7f0c35c7566274275ec4adc5d1fbd70c5385`. Authority: the candidate's pinned `internal/specdoc/SPEC.v0.7.0.md`, especially §3.2, §4.C–E and §4.2. The task's v0.5.0 scope is stale. The reviewer-cr2 template incorrectly says revision 1 had no findings; the actual rev1 verdict and rework brief were read and checked. All tests and mutations ran in archive copies under `.temp/TASK-260830-1c28dz-review/`. No Story source, index, branch, HEAD, or product documentation was modified. No commit, checkpoint, integration, or acknowledgement was attempted.

## Blocking production findings

### P1-A — Read-only observation reopens quiesced input

`ops.go:130` unconditionally records `StateActive` after attach. `checkAttachQuiesced` consults that same memory. A successful quiesce followed by an admitted read-only attach changes the memory to active; a subsequent input-authorized attach then succeeds with writable `attach-session -t ...` argv. `TestReviewReadonlyMustNotReopenQuiescedInput` FAILS through three `Lifecycle.Execute` calls, on the unmodified candidate. This is a sequential bypass, not a speculative race. It contradicts §4.E line 1383 and the input-quiescence semantics in §4.D: observation must not reopen input. The new committed test checks writable refusal BEFORE read-only observation and stops before the bypass. Preserve the barrier across observation, retry, and recovery; make the full sequence a committed regression.

### P1-B — Exact-instance status manufactures matching identity from the query

`status.go:60–78`, `backend.ObserveStatus`, sets IdentityMatch true and fills SessionID/ImplVersion/ProtoVersion/Generation from the request. `list-panes` only supplies command-name rows; no independently observed binding identity establishes that the requested session/version owns the instance. `TestReviewStatusMustObserveIdentityNotEchoQuery` records a valid binding, then queries the same present instance under another session ID or implementation version `9.9.9`: both return `identity_match:true`, parked, attachable. The unrelated protocol-major case correctly refuses and is retained as a control, not counted as a bug. §4.C lines 1178–1187 require all identity members to match and prescribe the absent/non-match result. Compose the durable binding/observed identity owner rather than echoing requested identity into the engine's evidence.

### P1-C — Reboot restore cannot produce a binding for the new server generation

`ops.go:409–449` first commits wrapper restoration, then reads the CREATE-time binding and requires its generation to equal the new mutation context. `TestReviewRestoreAcrossServerGeneration` drives a successful production create, reopens the reboot-scoped receipt store (the producer's own reboot convention), changes CurrentGeneration and the request to `generation-after-reboot`, and drives Execute(restore). The production runner receives `new-session`, then restore fails with `terminal_backend_integrity_failure at restore binding image`. §4.C line 1211 explicitly requires prior binding, Checkpoint, and **new binding** evidence; §4.E line 1387 and §4.2 require restoration after reboot. Re-reading the old binding only solves the nil result within the old generation. Compose the binding owner to persist/return the correct successor, validate before irreversible effects where possible, and prove retry/crash recovery across a real generation change.

### P1-D — After-restore composition is still a report with no production consumer

`ops.go:466–513` implements a second branch calculation in tmuxserver, returns `resumed_locally` / `remote_offered`, and nothing consumes these results outside this package. The existing wrapper decision owner is not composed into those effects. The candidate itself says the wire result is always parked and no provider is launched; its tests assert strings, not a resume or offer effect. This does not close rev1 P1-E or the explicit ordered-composition assignment for §4.2 lines 1433–1441.

The committed `TestExecuteRestoreRemoteOfferOnLapsedGrant` does not lapse its grant: it uses the standard future expiry. The reviewer probe with an actually expired grant never reaches refresh or offer: `refreshed=false`, branch nil, unauthorized expiry. That backend refusal is correct under the separate restore-authorization contract and MUST NOT be removed to make this probe pass. It proves the claimed lapsed-grant wrapper composition is absent at the selected entry. Put the ordered decision at the correct wrapper/composition entry and drive the effects, including the lapsed-local-grant remote offer; do not weaken backend authorization. The callback receives a timeout context, but the committed “blocking-refresh” test neither measures elapsed duration nor drives a production mesh adapter. Report the actual bound and adapter assumptions accurately.

## Blocking evidence/recovery findings

### P2-A — Corrupt outcome reads become fresh successful evidence

`lifecycle.go:477–500` treats ANY `LookupOutcome` error like absence and ignores `RecordOutcome` errors. `TestReviewCorruptOutcomeMustNotBecomeFreshEvidence` succeeds at quiesce, corrupts only the persisted report, advances the clock one second, then retries the identical request. Execute returns success and changes input_closed_at from `12:00:00.000Z` to `12:00:01.000Z` without closing input again. This is not just the declared B34 crash window: an existing corrupt/read-failed report is silently replaced. §4.C 1207–1208 requires closure-time/generation evidence and stable retry proof. Keep failed reads distinct from missing reports and persist authoritative operation evidence with recovery semantics; do not repair corruption by inventing a new event time.

### P2-B — Reachable census cells and missing refusal witnesses; comparison is still test-verdict keyed

Five independent admitting narrowings each SURVIVE the entire committed tmuxserver suite twice:

| Plant | Site/member admitted | Full committed suite | Added reviewer killer |
|---|---|---|---|
| R1 | restore binding instance equality: one substituted UUID | SURVIVED ×2 | TestReviewKillRestoreInstanceNarrowing KILLED ×2 |
| R2 | outcome LOOKUP identity: `review-forged-key` | SURVIVED ×2 | TestReviewKillOutcomeLookupNarrowing KILLED ×2 |
| R3 | status attached-count check: exactly -1 | SURVIVED ×2 | TestReviewKillNegativeCountNarrowing KILLED ×2 |
| R4 | materialization error check: valid=true plus one named error | SURVIVED ×2 | TestReviewKillMaterializationErrorNarrowing KILLED ×2 |
| R5 | root custody refusal at CREATE, other operations/classes intact | SURVIVED ×2 | TestReviewKillCustodyCreateNarrowing KILLED ×2 |

R1–R4 attack clause sites/members not planted by the shipped harness; R5 moves the measured status/restore custody gate to another reachable production entry, as requested. The gate remains present in every plant. All added killers PASS on the original candidate. The outcome-key killer is intentionally at the public store lookup entry; the other four drive Lifecycle.Execute. Applied harmless control SURVIVED twice. Raw logs, commands, replacements, subprocess exits and scripts are attached; no NOT_APPLIED or build failure is credited as a kill.

The 115×21 Table D arithmetic reproduces: 148 M, 30 B, 2237 U. However **131 U2b cells mean only that current tests supply post-gate values**, not that production cannot reach the refusal. Root/create R5 is a concrete counterexample. Those cells must be measured or explicit owned bounds, not included in “4206 unreachable.” Table C is summarized in prose instead of expanded cells, claims 19 U2c cells, and lists only 18 gate names. The 208/4448 is a declared census count, not independently established entry coverage. Symmetry counts now exist, but do not repair these missing sides/members. The matrix duplicates the 33 AC rows once; TRACEABILITY has 33 unique rows.

The new importer artifacts cover the correct **34/34 package closure**, an improvement over rev1. Their rows are still `go test -json` events: keys Time/Action/Package/Test/Elapsed, values pass/fail/skip. `importer-added.txt` is literally test-name plus pass. This proves test stability, not application inputs/results or movement between admission/refusal arms. No complete business-outcome comparison is provided. Replace that assertion with actual production input/outcome observations and explicitly identify moved classes.

### P2-C — Candidate is no longer current outside the Story boundary

At review, remote main independently resolves to `40bb8c9f89a5430c556e8746e1dca9769f4b3c47`; HEAD is two commits behind local main. Candidate remains byte-identical to its checkpoint outside this leaf, but differs from current main in sibling-owned fencing, sessrepo, terminstance, axpane and traceability files. `stale-outside-story.json` enumerates them; new main-only tests are also absent. This is not a claim that the leaf's 34-path patch itself edits those owners, and the reviewer did not refresh/rebase the managed branch. The brief's “trunk has NOT moved” premise is now false. A managed refresh/integration reconciliation and validation against current dependencies is needed before asserting current-trunk composition. Preserve Story ownership and use the orchestrator's managed path; do not copy old sibling files over main.

## Confirmed improvements and retained bounds

- Read-only attach now emits `-r`; the original writable-vector defect is fixed. The quiesce sequencing defect above remains.
- OSRunner carries stderr and signal death as error; unmarked/negative/permission probe failures no longer directly prove closure in the tested consumers.
- Custody uses no-follow opens and a lexical ancestor walk. Intermediate-symlink refusal, identity seam, root/status and root/restore regressions pass. B19/B20 implementation improved; the missing entry witnesses are separately identified.
- B18 byte limits remain driven; B16 remains a documented split between ambient-free Lifecycle and Acquire's nested refusal. Foreground fresh-decoy P3-A tests pass. Exec/probe/spawn adapters and socket-unlink crash tests exist and pass. Production→Acquire composition remains bound B27, not silently proven.
- B33 explicitly admits the wrapper never emits the boundary signal today. Timeout is fail-closed; do not call this a complete effect-level boundary protocol. B34 describes a separate outcome-record crash window, not permission to swallow corrupt outcome reads. B35 documents tmux-version-specific stderr markers.
- No candidate delta in `internal/traceability` or task-board.config.json; the registry remains the final leaf's scope. README's unconditional quiesce/replay claims must be corrected with the fixes above. No runtime capability or CLI command is newly advertised.

## Coverage measured by this reviewer

**8 of 8 operation entries exercised**, through Lifecycle.Execute:

| Operation | Committed passing witness | Production call site |
|---|---|---|
| create | TestExecuteCreateInteractive | Execute → executeCreate → Engine |
| attach | TestExecuteAttach | Execute → executeAttach |
| status | TestExecuteStatusPresent | Execute → executeStatus → ObserveStatus |
| quiesce | TestExecuteQuiesce | Execute → executeEngineOp → closeInput |
| safe-boundary | TestExecuteBoundary | Execute → executeEngineOp → observeBoundary |
| stop | TestExecuteStop | Execute → executeEngineOp → confirmClosed |
| stale termination | TestExecuteTerminate | Execute → executeTerminate → runLifecycleEffects |
| restore | TestExecuteRestore | Execute → executeRestore → runLifecycleEffects |

**33 of 33 distinct AC rows (59–91) have a named reference that executed and passed; 161 of 161 distinct referenced test names resolve and pass.** This is test-execution coverage, not 33/33 production conformance. Rows 79, 80, 81, 83, 88 and 89 are materially falsified or incomplete above. No accepted complete gate×entry ratio can be stated while reachable cells are mislabeled. Zero real-tmux witnesses; no acceptance row relies on one. Baseline package run with tmux unresolvable and no tmux process: 604 PASS including subtests, zero FAIL, zero SKIP.

## Validation and evidence boundaries

- Source identity audit: 891 non-board candidate files in the baseline copy match immutable tree blobs after tests; no mismatches. Story source untouched.
- Reviewer firsthand: gofmt clean; native `go vet ./...` exit 0; Windows amd64 `go vet ./...` exit 0; `go test ./... -count=1` exit 0, 45 packages, 224.741 seconds. Per-command wall time and host load are logged. The full suite does not include the separately injected reviewer regressions.
- Reviewer regressions: five top-level probes fail on original production, including the explicitly labeled missing-composition probe; the protocol-major refusal control passes. Four initial killer probes plus the later negative-count probe pass baseline and kill their respective plants twice.
- Configured suite contains 30 commands. The CR validation resource is TRUNCATED (5,902,256 bytes omitted), exposing only 21 command/exit records. The producer archive has separate full/race/cover/fuzz/gate logs and reports success; these are producer evidence, not reviewer execution. No claim is made that this reviewer replayed all 30 commands or independently verified the omitted runtime exits. The independently rerun full ordinary suite, vet, adversarial tests and mutation batteries are specified exactly here. Race/fuzz/cross-build replay beyond those checks is not claimed.
- No Python cache artifacts in the immutable CR or review copies at audit. Board mutation failures during discovery were unknown operation names, not evidence about task state; corrected commands used documented contracts.
- Full mutation pass completion and archive hashes are recorded in the final evidence addendum below before routing the verdict.

## Rework scope (for the producer)

1. Preserve quiescence through read-only observation and bind status to independently observed identity; commit the supplied failing sequences.
2. Restore a new-generation binding with correct recovery ordering; compose the actual wrapper after-restore decision/effects and drive the real lapsed-grant branch without weakening backend authorization.
3. Stop manufacturing fresh replay evidence after corrupt/read-failed outcome records; fix durable outcome recovery.
4. Add the five missing refusal witnesses, replace U2b pseudo-unreachability with measured/bounded cells, repair Table C/counts, and supply production input/outcome composition evidence over the 34-package closure.
5. Coordinate the managed current-main refresh, rerun appropriate checks on the revised immutable candidate, correct README/results closure claims, and hand off uncommitted through the existing CR workflow.

## Final evidence addendum

Both full shipped-harness passes completed **210/210 expected verdicts, exit 0 each**: 209 KILLED (207 labeled narrowing rows and 2 supplementary additive rows), one harmless control SURVIVED. There are 210 raw subprocess logs per pass. Second pass elapsed 214.011 seconds; load and timestamps are in its log. These expected kills do not override the five additional surviving narrowings or the unmodified-production failures.

Reviewer battery: 5 admitting plants ×2 full-package executions = **10/10 SURVIVED**; applied harmless control ×2 = **2/2 SURVIVED**. Added killers: **10/10 KILLED**, with each of the five killers passing the original candidate. No NOT_APPLIED, ERROR or timeout was credited as evidence.

Final live-worktree audit: all 891 non-board candidate files match the immutable candidate, HEAD remains the checkpoint, staged diff empty. No tmux process at final check (pgrep exit 1, empty stdout/stderr). Patch SHA-256 matches the supplied `2c6eaf427828737206ff4fa1eb3ea59ac5a1ef41a24b96b2b36649d98c6aac93`. Unsupported checklist claims were unchecked; rejection is not represented as a completed acceptance checklist.

Evidence archive: `TASK-260830-1c28dz_review-evidence-rev2.tar.gz`, containing the raw logs, audit JSON, reviewer probe sources, mutation scripts, verdict and SHA-256 manifest. Producer/previous-review archives remain separate board resources; their contents are not relabeled as reviewer runs. This verdict and the archive are read back through the board and verified byte-identical before routing.
