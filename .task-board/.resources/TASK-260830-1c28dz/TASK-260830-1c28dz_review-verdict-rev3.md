# TASK-260830-1c28dz — revision 3 review

Verdict: **CHANGES REQUESTED**, route to `to-dev`. No external blocker. Do not accept CR revision 3.

Reviewed base `d4bd91d0e740285b21b8d65bf6f074c8009b04ae`, immutable candidate tree `8ce88f1e30a3faebe3717417faac1f6a03026fb1`. Patch SHA-256 verified: `53e66d703d06c227e2c5de374d8baaa503577c00f8528959c5045cb399799567`. Authority is pinned `internal/specdoc/SPEC.v0.7.0.md`, particularly §3.2, §4.C–E and §4.2; the task's v0.5.0 reference is stale. The contradictory revision-history boilerplate is not evidence: rev1 and rev2 have changes-requested verdicts, which were read alongside the rework instructions and predecessor rev7 evidence.

## P1-A — quiescence fails open after its state write fails

`lifecycle.go:410–426` commits the engine operation then calls `recordAfter`; `recordAfter` at line 242 ignores `States.Record` errors. `ops.go:156–167` enforces writable-attach refusal only from that supposedly advisory memory. Its absence admits input.

`TestReviewQuiesceStateWriteFailureCannotReopenInput` drives the unmodified production `Lifecycle.Execute` twice. A state-store `AfterStage` failure for the instance document leaves quiesce returning success. After removing the hook, writable attach succeeds with `tmux -S ... attach-session -t ...`, without `-r`. The runner already executed lock-session and detach-client. This is a durable-state failure bypass of the input barrier, not the already-fixed read-only-attach sequence and not the separately declared B34 report-time window. It contradicts §4.C quiesce and §4.E input closure. Persist/recover the enforcing barrier authoritatively; a failed write must not produce successful closure followed by writable admission.

## P1-B — refreshed lease decision is not bound to the operation it executes

`wrapper.go:114–139` decides over the refreshed winner and then passes the independently supplied `req.Restore` into `Lifecycle.Execute`. Backend authorization consults `lc.CurrentLease()` again, but nothing ensures this is the winner the wrapper just used.

`TestReviewRefreshedWinnerMustBindExecutedRestore` refreshes to local lease B, epoch 9, and presents that tuple to the wrapper decision. The backend request and `CurrentLease` still carry lease A, epoch 7. ExecuteWrapperRestore returns `ActionLaunch`, nil error, and executes `new-session` under the old authorization. Both separate gates pass while their composition authorizes different leases. Bind the selected winner, decision identity/materialization and effect request; reconcile or refuse divergent observations before execution, without weakening backend restore authorization. §4.2 step 3 requires the winning local lease, not independent successes for two leases.

## P1-C — expired refresh is accepted as verified, and the advertised bound is only cooperative

`wrapper.go:168–179` calls the callback synchronously, cancels its context, and checks only the returned error. It never tests whether the refresh deadline elapsed before accepting the answer.

`TestReviewExpiredRefreshMustNotResume` waits for `ctx.Done()` inside the refresh callback, then returns a local winner with nil error. The unmodified entry reports `RefreshSucceeded=true`, selects launch and executes new-session. The documented timed-out fallback is bypassed.

`TestReviewRefreshBoundNotJustCooperativeDeadline` gives the entry a 1ms bound and a callback returning after 100ms; the entry blocks for 102.575ms. This finite probe demonstrates that the entry does not enforce its claimed bound. It does not pretend to have measured an infinite wait. No production LeaseRefresh adapter exists outside this function-field interface; the committed test uses a cooperative callback and allows up to five seconds for a 50ms setting. Reject expired/cancelled results, and provide the bounded adapter/composition or accurately state and prove the required cooperative contract. The §4.2 step-2 nonblocking requirement and the comments saying the bound is “not advisory” are not established by a context deadline alone.

## P2-A — requested importer outcome comparison is still absent

The attached producer archive's `outcome_compare.py` keys `out[ev["Test"]] = ev["Action"]`; it does not even include Package in the key. The result says “keyed by full test name” and claims retained test passes prove no landed behavior changed. This is the same explicitly rejected evidence shape as rev2, not a production input/outcome grid.

Independent `go list -json ./...` derivation yields 34 production packages in the transitive importer closure of the nine assigned composing packages (36 including test-only importers `crashgate` and `environ`). The full reviewer suite passes all 45 packages, including that closure. Neither that green nor the producer comparison measures changes between business admission/refusal outcomes. Supply base/candidate runtime input/outcome observations across the derived closure, with instrumentation and every moved class identified. Existing pass/fail inventories may remain supplementary.

## P2-B — traceability overstates coverage and misses refusal clauses

The 119×21 Table D recount is correct: 167 M, 152 B, 2180 U. Reclassifying reachable-undriven cells to B39 is an improvement and is accepted as an honest limitation, not as measured coverage. However:

- Table C still lists 18 U2c gate names while claiming 19. It remains prose instead of a fully expanded census.
- Tables A/B/C/D contain 264+504+1309+2499 = **4576** cells, not the claimed **5576**. The listed categories also sum to 4576 (227+156+4193).
- Row 83 names five removed tests: `TestExecuteRestoreRemoteOfferOnLapsedGrant`, `TestExecuteRestoreOffersWhenRemoteHoldsLocalTuple`, `TestExecuteRestoreResumesUnderLocalLease`, `TestExecuteRestoreParksOtherwise`, `TestExecuteRestoreRefreshBounded`. The same stale row is in the attached matrix.
- Row 83 and results say successor persistence precedes effects. In production, `executeRestore` calls `executeWithReceipt` at ops.go:430 and only then `restoreBinding` at line 437. The latter persists the successor. Prior validation precedes effects; successor persistence does not. Keep this distinction explicit.
- The traceability introduction still says 91/91, rows 59–91, while the table now contains row 92. Producer results also say no test reads wall time, contradicted by the newly added elapsed-time wrapper test.

Four additional clause-level admitting narrowings, not present in the shipped harness, survive the entire committed tmuxserver suite twice. Each leaves the gate and its other refusal members intact:

| Plant | Admitted member / production entry | Committed suite | Reviewer test |
|---|---|---|---|
| R1 | list-sessions exit 1 with plausible stdout / Execute(status) → probeAttached | SURVIVED ×2 | TestReviewKillStatusAttachedExit KILLED ×2 |
| R2 | exactly `review-foreign-instance` as a matching attached-count row / Execute(status) → probeAttached | SURVIVED ×2 | TestReviewKillStatusForeignRow KILLED ×2 |
| R3 | unknown list-panes exit 2 with plausible stdout / Execute(status) → probePanes | SURVIVED ×2 | TestReviewKillStatusUnknownPanes KILLED ×2 |
| R4 | exactly the NUL-containing `review\x00arg` argument / OSRunner.Run argv admission | SURVIVED ×2 | TestReviewKillOSRunnerArgv KILLED ×2 |

A fifth plant narrows the shared root-custody refusal specifically at `ServerSpawner.Spawn`, while the measured helper/Execute sides stay intact. It admits the root-writability refusal only: SURVIVED ×2, `TestReviewKillSpawnRootCustody` KILLED ×2. This confirms the producer's already-declared SPW/B39 gap; it is not mislabeled as a false M cell. All five added killers pass against the original candidate. Applied harmless comment control survives twice. The first R5 attempt used the wrong error-string prefix and did not admit anything; those raw logs are retained as an invalid probe and are NOT credited. `r5fix-results.json` and `R5-corrected-*` contain the verified admitting plant.

## Improvements retained and limits

The previously fixed read-only argv, probe unknown/signal handling, and no-follow custody remain intact under the committed suites and harness. The normal read-only observation no longer reopens quiescence. Corrupt and empty outcome reads now refuse rather than becoming fresh evidence. Status observes the stored binding and detects tested identity drift. A first generation-changing restore returns a successor and its identical retry converges. The actual lapsed-grant remote branch now reaches the landed axpane decision before backend authorization.

The new wrapper entry still returns offer/launch decisions; the local branch recreates a parked wrapper, and the remote branch returns an offer without an external consumer. The committed tests establish those outcomes, not actual provider resumption or a displayed/accepted remote offer. Do not relabel them as complete provider/UI effects. B27 (Acquire wiring), B33 (wrapper never emits the boundary signal), B34 (receipt/report-time crash window), B38 and B39 remain explicit bounds.

B18 socket byte limits, B19/B20 tested custody entries, B16 ambient-free lifecycle/scrub plus Acquire nested refusal, and the foreground fresh-decoy test are retained. Exec, dial, probe and spawn adapters exist and execute in deterministic tests. Socket-unlink crash/idempotency tests pass. No newly advertised CLI or runtime capability was found. No change to internal/traceability or task-board.config.json exists in this CR.

## Coverage actually measured

**8 of 8 operation entries driven** by existing candidate tests that passed:

| Operation | Named test | Production call site |
|---|---|---|
| create | TestExecuteCreateInteractive | Lifecycle.Execute → executeCreate → terminstance.Engine |
| attach | TestExecuteAttach | Lifecycle.Execute → executeAttach |
| status | TestExecuteStatusPresent | Lifecycle.Execute → executeStatus → backend.ObserveStatus |
| quiesce | TestExecuteQuiesce | Lifecycle.Execute → executeEngineOp → closeInput |
| safe-boundary | TestExecuteBoundary | Lifecycle.Execute → executeEngineOp → observeBoundary |
| stop | TestExecuteStop | Lifecycle.Execute → executeEngineOp → confirmClosed |
| stale termination | TestExecuteTerminate | Lifecycle.Execute → executeTerminate → runLifecycleEffects |
| restore | TestExecuteRestore | Lifecycle.Execute → executeRestore → runLifecycleEffects |

The acceptance table has **34 distinct rows, 59–92**. All 34 have at least one passing named witness; **33 of 34 rows have their entire named test set resolved and executed**, with **189 of 194 distinct referenced names passing and five absent** (row 83). This is execution/reference coverage, not acceptance of all clauses: rows 79/81/83/92 are affected by the production findings and composition limits above. The correct whole-census denominator is 4576; the claimed 227 measured cells were not all independently attributed, so no fully verified gate×entry conformance ratio is asserted. New termbind row 67 successor tests also pass.

No-tmux environment: PATH excludes Homebrew, `tmux` cannot resolve, pgrep exit 1 with no process; **683 tmuxserver + 270 termbind = 953 PASS, zero FAIL, zero SKIP**, counting subtests. Zero real-tmux witnesses, and no row relies on one. The review never stopped another user's tmux process.

## Validation and provenance

Firsthand, on git-archive copies of the immutable tree under `.temp/TASK-260830-1c28dz/review-rev3/`:

- `go test ./... -count=1 -json`: exit 0, 45/45 packages, 24,318 passing test events and 14 pre-existing skips outside the touched packages. Largest package elapsed 163.748s; timestamps and host load are retained.
- Native `go vet ./...`, Windows amd64 `go vet ./...`, gofmt check, and live diff whitespace check: exit 0, formatting output empty.
- Changed-package coverage: exit 0, tmuxserver 87.9%, termbind 83.5%.
- Shipped tmuxserver harness twice: **235/235 expected verdicts each, exit 0**; 234 KILLED (232 labeled N rows plus two supplementary D rows), one harmless SURVIVED control, per-plant subprocess logs.
- Shipped termbind harness twice: **40/40 expected verdicts each, exit 0**; 39 KILLED and one harmless SURVIVED control, verbose per-plant raw blocks.
- Independent battery: four new clauses ×2 = 8 SURVIVED; corrected cross-entry custody ×2 = 2 SURVIVED; harmless control ×2 = 2 SURVIVED. Added killers: 10/10 KILLED, all five pass baseline. No build failure, timeout or unapplied patch is credited as a kill.
- Four unmodified-production reviewer regressions fail, exit 1, as described in P1-A/B/C. Their failure is the evidence; they are not claimed green.

The configured suite has 30 commands. The CR validation resource omits 5,923,850 bytes and exposes only 21 exit records. Producer archives were inspected as producer evidence, not relabeled as reviewer runs. Full race/fuzz/cross-build replay and all 30 configured exits are NOT independently established here; the explicit checks above and the adversarial failures are this review's evidence. A rejected candidate is not awarded missing validation gates.

Final byte audit: all 898 non-board candidate files in the live Story worktree and restored validation copy match the CR blobs; HEAD remains d4bd91d, staged diff empty, no Python cache artifacts. Current remote main independently resolves to 40bb8c9; HEAD is zero commits behind local main. Main's remaining differences are the Story predecessor plus this leaf's README/LOGBOOK scope, not the stale sibling implementations reported in rev2. No live source/index/branch mutation, commit, checkpoint, integration, or commit acknowledgement occurred. Reviewer probes and mutations exist only in disposable copies. An initial wrong-directory probe invocation ran no reviewer tests; its log is retained and excluded from all counts.

## Rework scope (for the producer)

1. Make the quiescence barrier durable and fail closed across state-write failure/recovery; commit the failing sequence.
2. Bind the wrapper's refreshed decision tuple to the backend restore request and the authorization used for effects; reject divergence.
3. Reject expired refresh results and prove the bounded refresh composition with accurate adapter assumptions; preserve the genuine lapsed-grant remote-offer direction.
4. Add the four missing clause witnesses and corresponding admitting mutants. The attached fifth killer can close the already-owned spawner custody bound.
5. Replace the test-verdict comparison with production input/outcome grids over the 34-package production importer closure; identify changed arms.
6. Repair stale test references, persistence-order claims and census arithmetic. Keep explicit limitations honest, especially actual provider/offer effects and the boundary signal. Rerun relevant checks on the revised candidate and attach complete bounded validation evidence before handoff.

Evidence archive: `TASK-260830-1c28dz_review-evidence-rev3.tar.gz` contains reviewer logs, test sources, mutation scripts, audits and digest manifest. Producer archives remain separate referenced resources. Unsupported acceptance checklist claims are unchecked; changes requested is not represented as a completed acceptance checklist.
