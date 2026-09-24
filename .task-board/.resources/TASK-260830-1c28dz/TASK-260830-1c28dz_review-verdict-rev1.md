# TASK-260830-1c28dz — CR revision 1 review

Verdict: **CHANGES REQUESTED**, route to `to-dev`. No external blocker. Do not accept CR revision 1.

Reviewed base `1ba06e99090e3b7bf1aa2f4865914f4080a644ed`, candidate tree `c6b59758f3320546be5bbd855cf4137126e5cc52`. Downloaded patch SHA-256 matches `84cf8c74f0c940bc98b66859a3f734c3c70d39b64a1bc7b59783cadc9fd28b6e`. All executions and attacks use task-local git-archive copies. The live Story source, index, HEAD and branch were not edited, committed, switched, rebased or integrated. Authority is pinned SPEC.v0.7.0, not the stale task v0.5.0 reference. origin/main remains `799c338`; HEAD has zero commits behind local main. The overlap with origin/main consists of predecessor Story changes, not a new trunk refresh. Config and internal/traceability are unchanged in the CR.

## Blocking findings

### P1-A — Safe-boundary success does not observe a safe boundary

`exec.go:199`, `backend.go:observeBoundary`, `Lifecycle.Execute(wait-safe-boundary)`: the vector is `wait-for -L ax-boundary-<generation>`. This acquires a channel lock; on a fresh channel it succeeds immediately, without any wrapper/provider signal. `observeBoundary` turns exit 0 into safe-boundary evidence. The requested provider proof kind is not passed into this observation. SPEC §4.C row 1208 and §4.E 1384 require a proof or timeout. `TestReviewBoundaryNeedsSignal` reaches the production entry and fails on the vector; committed `TestExecuteBoundary` merely scripts success for the wrong command. The authoritative tmux manual distinguishes lock acquisition from waiting for a signal: https://man.openbsd.org/tmux (wait-for). This is a production semantic defect, not a need for more mock assertions. Similarly, `lock-session` locks currently attached clients, whereas SPEC 1247/1383 requires rejecting new operator/provider input. No AX/provider input barrier is implemented by that command alone. Establish and drive the actual barrier and boundary protocol, including new clients and provider inputs.

### P1-B — Read-only attach yields a writable execution vector

`ops.go:executeAttach` / `exec.go:BuildCommand(DirectiveAttachSession)` omit the input authorization when constructing argv. With a valid authorization and `input_authorized=false`, Execute returns `tmux -S ... attach-session -t ...`, exactly the writable vector returned for true. `TestReviewReadOnlyAttach` fails through Execute. Reporting false in AttachOutcome does not enforce it on the caller-executed vector. Preserve the authorization at the execution boundary and prove input cannot reach the pane. The tmux manual documents read-only attach (`-r`); another enforced mechanism is acceptable, an informational boolean is not.

### P1-C — Probe failures manufacture absence and closure

`backend.go:pollForAbsence`, `terminateStale`, and `status.go:probePanes` treat every nonzero exit as absence. OSRunner converts a signal-killed child into `ExitCode=-1, nil`, so a timeout or killed probe can become successful closure. `TestReviewStopUnknownExit` drives Execute(request-stop) and gets nil error with both ProcessClosed and StoreClosed true after a -1 has-session result. SPEC status row 1206 says read failure is unknown, never absent; stop and stale termination require closure evidence (1209-1210). Distinguish positively observed absence from command/protocol/permission/cancellation failure, preserve needed stderr/error classification, and drive all three consumers. Restricting to exit 1 alone is not sufficient unless its possible command failures are distinguished too.

### P1-D — Custody follows an intermediate symlink

`bind.go:checkCustodyAncestors` calls EvalSymlinks before its walk. A symlinked intermediate component disappears from the path being checked. `TestReviewCustodyIntermediateSymlink` constructs an alias above the valid root and CheckSocketCustody returns nil. This contradicts SPEC 809-812 component-by-component no-follow custody and the function's own no-symlink claim. The claimed B19/B20 closure is incomplete. Use the existing custody owner/descriptor-safe traversal; do not paper over arbitrary symlinks by treating them as the macOS /tmp convention. Re-prove before-connect/bind entries and identity stability.

### P1-E — Restore does not implement its required result contract

`ops.go:executeRestore` returns only PriorBindingID and RestoredParked, leaving OpOutcome.Binding nil. SPEC row 1176 requires `binding:TerminalInstanceBinding`. `TestReviewRestoreRequiredBinding` fails after a successful production Execute. The comment that no minter exists is not an external constraint permitting a different wire contract. Compose the binding owner and return the required binding. Also provide the assigned after-restore ordered integration evidence: this candidate only launches `ax pane`, unconditionally reports parked, and neither its restore witness nor its reported evidence drives local lease read -> bounded refresh -> local resume/remote offer/park. Existing axpane decision tests do not demonstrate this new entry is wired to those effects. Preserve the ownership split, but supply the actual composition and the lapsed-local-grant remote-offer witness and measured refresh bound requested by the review brief.

### P2-A — Replayed outcomes change their timestamps

`lifecycle.go:executeEngineOp` regenerates InputClosedAt and BoundaryObservedAt from Now after a stored mutation result is replayed. `TestReviewQuiesceReplayTimestamp` advances the clock by one second between identical requests and observes different closure timestamps without a second effect. SPEC 1207 binds evidence to closure time/generation; 1208 requires the same proof on retry. Persist/replay the complete operation-specific outcome, including original timestamps, and attack crash/retry paths rather than only exec-count equality.

### P2-B — Census and composition evidence do not satisfy the explicit handoff

The census defines U2a as 'entry never calls this code', then uses it for rootwrite/ancestorwrite at status and restore, even though Execute calls CheckSocketCustody before dispatch. Four independent entry-and-refusal-class narrowings (root/status, root/restore, ancestor/status, ancestor/restore) all SURVIVE the complete committed tmuxserver suite twice. Each leaves the custody gate present, admitting exactly one refusal class at one entry. New TestReviewCustodyEntryRoots passes on baseline and KILLS each mutant twice. Thus these are reachable missing refusal witnesses, not unreachable cells. The symmetry paragraph lists names but does not give the required side-by-side row counts. The advertised 168/3648 census is not an accepted measured coverage ratio.

The producer's `outcome_compare.py` explicitly keys by test name; both JSON streams contain only internal/tmuxserver. This is exactly the single-package name-keyed comparison forbidden by the task. Although tmuxserver itself currently has no production importers, the task explicitly requests composition evidence for the listed owners. The transitive importer closure of the nine named composition packages contains 34 packages (recorded in importer-closure.txt); no complete input/outcome comparison is attached. Do not call zero name-level regressions an outcome comparison.

Four referenced AC witnesses do not exist: TestExecuteRefusesNilDependencies (actual name is MissingDependencies), TestExecuteRestoreIdempotent, TestExecuteStatusPresentIncludesIdentity, TestObserveStatusAbsentProbe. The producer results resource is absent as a standalone board resource; its bytes were readable only inside the attached tarball. Correct names, bounds, row counts, comparison semantics and attachment manifest.

## Coverage actually measured

**8 of 8 operation entries exercised** by the unchanged candidate tests, all through Lifecycle.Execute:

| Operation | Named test | Production call site |
|---|---|---|
| create | TestExecuteCreateInteractive | Execute -> executeCreate -> terminstance.Engine |
| attach | TestExecuteAttach | Execute -> executeAttach |
| status | TestExecuteStatusPresent | Execute -> executeStatus -> Engine.ExecuteStatus |
| quiesce | TestExecuteQuiesce | Execute -> executeEngineOp -> backend.PerformEffect |
| safe-boundary | TestExecuteBoundary | Execute -> executeEngineOp -> observeBoundary |
| stop | TestExecuteStop | Execute -> executeEngineOp -> confirmClosed |
| stale termination | TestExecuteTerminate | Execute -> executeTerminate -> runLifecycleEffects |
| restore | TestExecuteRestore | Execute -> executeRestore -> runLifecycleEffects |

The real second-leaf acceptance table has **33 rows (59-91)**. **33 of 33 have at least one referenced named test that executed and passed**, 120 of 124 distinct named references resolve to passing tests. This is test execution coverage, NOT 33/33 production conformance: rows include helper-only witnesses and findings above falsify key claims (67/69, 79-83, 89). The complete gate-by-entry acceptance ratio is unverified because the census misclassifies reachable cells. Do not use 8/8 dispatch coverage as 8/8 correctly implemented effects.

B18 length checks exist and their tests pass. B16 is explicitly retained at Acquire and omitted from Lifecycle; this is a documented library-boundary choice, not proof of a composed nested CLI invocation. B19/B20 have working basic root refusal but P1-D and P2-B remain. Foreground fresh-decoy P3-A now has a named table and passes. Exec/probe/spawn adapters exist, and the socket unlink crash test passes; Production-to-Acquire composition remains producer bound B27. Zero real-tmux witnesses, as declared. Tests ran with tmux unresolvable and pgrep empty (environment evidence attached).

## Verification and limits

- Immutable candidate package test: exit 0; no-tmux run: 537 PASS including subtests, zero skip; coverage 87.3%.
- Native package vet, gofmt, and Windows amd64 `go vet ./...`: exit 0 (gofmt output empty).
- Six reviewer regression tests FAIL on original production, as expected: boundary command, writable read-only attach, false closure, symlink custody, replay timestamp, missing restore binding. Root/ancestor entry probes PASS baseline and fail under the four narrowings.
- Four extra broad entry-wiring probes (custody/status, custody/restore, length/status, length/restore) KILLED twice; supplemental only, not counted as the four class-narrowing survivors.
- Full configured suite has 30 commands. The attached CR validation log records 30/30 exit 0; this reviewer does not relabel those as firsthand. Board validation emits pre-existing issues while returning exit 0. No hosted CI was triggered. Independent full package run and two shipped-harness passes are recorded in the final verification addendum below. No claim of full race/fuzz replay is made.
- The evidence records platform and host load; no performance comparison is claimed. No product source edits or fixes were made by this read-only reviewer.

## Rework scope (for the producer)

1. Fix the real boundary/input protocol, read-only execution authorization, and unknown-vs-absent observations; add effect-level deterministic regressions.
2. Repair component custody through every production entry and replay the four supplied narrowing survivors with committed killers.
3. Return the specified restore binding, prove ordered after-restore composition and bounded refresh, and persist complete replay outcomes.
4. Replace false unreachable cells and prose symmetry with measured per-entry rows/counts; repair nonexistent references and attach results explicitly. Produce the required complete-importer input/outcome comparison.
5. Rerun relevant gates on the revised candidate and hand off a new uncommitted CR. Preserve the Story and registry ownership boundaries.

## Final verification addendum

Independent `go test ./... -count=1`: **exit 0**, 45 package results green (`all-tests.log`). Full configured 30-command replay was not performed; the attached 30/30 validation record was read and retained as producer/runtime evidence, not claimed as reviewer execution.

Shipped mutation harness: **176/176 expected outcomes on EACH of two complete passes**, exit 0 both. Per pass: 175 KILLED (173 N-* narrowings, 2 D-* additive supplements), 1 C-control SURVIVED; zero ERROR/NOT_APPLIED. Raw subprocess exits and kill lines are in `mutants-1/` and `mutants-2/`. The four independent class-and-entry narrowings additionally SURVIVED twice each; their reviewer-added killers KILLED twice each. These four target entry cells missing from producer coverage, not four entirely unrelated helper gates. A full independent attack on four additional unrelated gates and a complete importer outcome rerun are not claimed; the demonstrated defects already require rework.

Review evidence is deliberately separate from product changes. Unsupported checklist completion claims were unchecked; successful existing test runs are not acceptance of the six failing reviewer regressions. Task notes record the findings because the reviewer is read-only with respect to repository LOGBOOK.md.
