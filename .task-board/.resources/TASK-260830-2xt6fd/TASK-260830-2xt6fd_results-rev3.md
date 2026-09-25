# Developer handoff evidence — TASK-260830-2xt6fd rev3

State at artifact creation: development; developer handoff to to-review is pending. The managed Story candidate remains uncommitted at checkpoint 49d1d660c11c4e1f1564d72ab7e28b3f6cd63203.

## Rev3 rework scope

The rev3 code/test/harness delta is limited to internal/tmuxserver/capture_git_workspace.go, internal/tmuxserver/capture_git_workspace_test.go, and internal/tmuxserver/mutant_harness.py. Earlier Story files and evidence remain carried from CR2. No repository config or registry file changed during this rework. The scratch-index tree for the candidate is f7f04778d7f82cccf23015d5c939483ef2d0c4ad; the live worktree index was not written.

The coordinator now has one literal refusal detail for each reachable captureUnavailable site. Its AST census follows package-local direct calls from (*Lifecycle).CaptureGitWorkspace, rejects non-literal detail expressions, and requires a bijection with production-entry matrix tests. Each row starts from a valid Runner/Assembly/boundary/current-receipt/barrier request and breaks only its named gate input; its assertion pins literal capability_unavailable plus the literal detail. A control admits both allowed entry states. See TASK-260830-2xt6fd_capture-gate-coverage-rev3.md for the full 22-row mapping.

The stop-point owner remains the landed Terminal Instance lifecycle. Capture accepts current quiescing-with-boundary evidence or stopped, rechecks closing identity/state/barrier, and never releases or restores the instance. The owner-returned typed wait-safe-boundary mutation shape is a structural owner invariant: Lifecycle.Execute supplies it, while capture separately validates its safe-boundary digest and persisted same-incarnation receipt. No synthetic seam was added to forge an owner result. A live-session return to active remains outside contract under the closed v0.7.0 §4.C vocabulary.

## Measured coverage

- Final-leaf acceptance map: 6 of 6 rows driven through the production call sites named in TASK-260830-2xt6fd_coverage-map-rev2.md.
- Coordinator refusal census: 22 of 22 reachable literal sites have exactly one production-entry row.
- Coordinator narrowing evidence: 22 of 22 site plants were killed by their named row when run alone; each was killed twice. One first-pass close-status plant hit a disk-full compiler error, was retried alone and killed, then killed again in the repeat batch. The initial infrastructure error is not counted as a kill.
- Census controls: an added site and a computed/non-literal detail each made the census fail. The harmless comment-only C-control survived as expected.
- Brief gap: no surface table was supplied. The six-row acceptance map and per-gate supplement are attached; the missing table is not treated as waived coverage.

## Mutant evidence

For KILLED rows, the harness exit is 0 for a valid verdict and the individual behavior go test exits 1 with the named test failure. Each raw process log is in the task-scoped evidence archive.

| Mutant | Narrowed admission | Named failing test | Result |
| --- | --- | --- | --- |
| N-capture-admits-nil-lifecycle | Lets a nil lifecycle pass for one fixture group | TestCaptureAdmission_LifecycleRequired | KILLED twice |
| N-capture-admits-session-scoped-status | Lets status without exact instance identity pass | TestCaptureAdmission_ExactStatusIdentityRequired | KILLED twice |
| N-capture-admits-opening-status-nonmatch | Admits one mismatched opening status | TestCaptureAdmission_OpeningStatusIdentityMatch | KILLED twice |
| N-capture-admits-absent-opening-state | Admits only absent | TestCaptureAdmission_InitialAbsent | KILLED twice |
| N-capture-admits-parked | Admits only parked | TestCaptureAdmission_InitialParked | KILLED twice |
| N-capture-admits-active-opening-state | Admits only active | TestCaptureAdmission_InitialActive | KILLED twice |
| N-capture-admits-stale-fenced-opening-state | Admits only stale_fenced | TestCaptureAdmission_InitialStaleFenced | KILLED twice |
| N-capture-admits-unavailable-opening-state | Admits only unavailable | TestCaptureAdmission_InitialUnavailable | KILLED twice |
| N-capture-admits-nil-assembly-runner | Admits a nil Runner for one fixture group | TestCaptureAdmission_AssemblyRunnerRequired | KILLED twice |
| N-capture-admits-stopped-boundary-body | Admits one stopped capture with a boundary body | TestCaptureAdmission_StoppedBoundaryBodyForbidden | KILLED twice |
| N-capture-admits-stopped-without-incarnation | Admits the fixture's missing stopped incarnation | TestCaptureAdmission_StoppedIncarnationRequired | KILLED twice |
| N-capture-admits-quiescing-without-boundary | Admits one quiescing request without a boundary body | TestCaptureAdmission_QuiescingBoundaryRequired | KILLED twice |
| N-capture-admits-boundary-identity-drift | Admits one mismatched boundary identity | TestCaptureAdmission_BoundaryIdentityMatchesStatus | KILLED twice |
| N-capture-admits-checkpoint-proof | Admits ax_checkpoint_boundary as provider proof | TestCaptureAdmission_CheckpointOnlyProviderProof | KILLED twice |
| N-capture-admits-quiescing-without-incarnation | Admits the fixture's missing quiescing incarnation | TestCaptureAdmission_QuiescingIncarnationRequired | KILLED twice |
| N-capture-admits-foreign-quiesce-receipt | Admits the one foreign input-closure receipt | TestCaptureAdmission_CurrentInputClosureReceiptRequired | KILLED twice |
| N-capture-admits-foreign-provider-boundary-receipt | Admits the one foreign provider-boundary receipt | TestCaptureAdmission_CurrentProviderBoundaryReceiptRequired | KILLED twice |
| N-capture-admits-closing-status-nonmatch | Admits one closing status identity mismatch | TestCaptureAdmission_ClosingStatusIdentityMatch | KILLED twice; one initial compiler ERROR retried |
| N-capture-admits-changed-closing-incarnation | Admits one changed closing incarnation | TestCaptureAdmission_IncarnationStableAcrossCapture | KILLED twice |
| N-capture-admits-final-parked-state | Admits only parked as the closing state | TestCaptureAdmission_FinalStateRemainsHeld | KILLED twice |
| N-capture-admits-stopped-to-quiescing-for-one-group | Allows one stopped capture to become quiescing | TestCaptureAdmission_StoppedRemainsStopped | KILLED twice |
| N-capture-admits-unproven-barrier-for-one-group | Admits one lost closing quiescence barrier | TestCaptureAdmission_ClosureBarrierStillProven | KILLED twice |
| C-capture-census-unlisted-refusal-site | Control adds a refusal site without a matrix row | TestCaptureCoordinatorAdmissionGateCensus | KILLED control |
| C-capture-census-nonliteral-detail | Control computes a detail while preserving its runtime value | TestCaptureCoordinatorAdmissionGateCensus | KILLED control |
| N-capture-restores-with-boundary-token-preserved | Substitutes restore for the owner's wait-safe-boundary transition | TestCaptureGitWorkspaceQuiescedAssemblesAndRechecks; TestCaptureNeverTransitionsOutsideStopPointOwnerSet | KILLED |
| C-control | Comment-only edit with no behavior effect | TestSocketPathDerivesOnlyFromRuntimeDir | SURVIVED as expected |
| N-assembly-closure-admits-one-missing-child (carried from earlier Story evidence) | Allows the one missing-child fixture past the local visitor check | TestValidateProvisionalClosureAndDescriptors | SURVIVED; the same fixture is refused by the required tree-binding gate, so this plant proves no independent child-visitor coverage and is not counted as a kill |

Earlier broad harness controls that exceeded the harness's 240-second limit are diagnostics (ERROR), not kills or survivors; their evidence is in the previously attached rev2 archive. The rev2 coverage map names their bounds.

## Validation and evidence

All commands below ran as standalone processes. Logs are under .temp/TASK-260830-2xt6fd/ and the rev3 logs are included in the evidence archive.

| Command | Exit | Result / evidence |
| --- | ---: | --- |
| go test ./internal/tmuxserver -run '^(TestCaptureAdmission_.*|TestCaptureAdmissionGateValidControls|TestCaptureStopPointStateDomain|TestCaptureCoordinatorAdmissionGateCensus|TestCaptureNeverTransitionsOutsideStopPointOwnerSet)$' -count=1 -v | 0 | capture-matrix-01.log |
| Same matrix command with -count=3 -v | 0 | All three repetitions pass; capture-admission-determinism-rev3-01.log, 188.238s |
| go test ./internal/tmuxserver -count=1 -v | 0 | Complete changed package passes; tmuxserver-full-rev3-01.log, 454.500s |
| go test ./internal/tmuxserver -cover -run '^TestCapture' -count=1 -v | 0 | Capture behavior plus crash/retry coverage passes; 35.0% of statements in this filtered run; tmuxserver-capture-cover-rev3-01.log, 313.040s |
| Capture-site harness first/repeat runs | 0 per harness command | All 22 site mutants killed twice in separate row runs. First-pass logs and compiler-error retry are under mutants-sites-*; repeats are under mutants-sites-repeat-*. |
| Census controls, harmless control, and restore-transition plant | 0 | Both census controls and the restore narrowing killed; comment-only C-control survived; mutants-controls-repeat-summary-01.log plus raw files. |
| env -u TMPDIR GOOS=windows GOARCH=amd64 go vet ./... | 0 | Full Windows vet on rev3; go-vet-windows-rev3-01.log |
| env -u TMPDIR go vet ./internal/tmuxserver | 0 | Host vet of changed package; go-vet-tmuxserver-rev3-01.log |
| env -u TMPDIR go build ./... | 0 | Full host build on rev3; go-build-rev3-01.log |
| env -u TMPDIR golangci-lint run --new-from-rev=HEAD | 0 | Rev3 delta lint clean; golangci-lint-rev3-01.log |
| gofmt -l internal | 0 | No output; gofmt-rev3-01.log |
| git diff --check and scratch-index git diff --cached --check | 0 each | git-diff-check-rev3-01.log, scratch-index-hygiene-rev3-01.log |
| Scratch-index git read-tree HEAD, git add -A, git write-tree | 0 each | Only .temp/TASK-260830-2xt6fd/hygiene-rev3.index was written; candidate tree ID in hygiene-rev3-tree.txt. task-board.config.json byte comparison against HEAD passed (config-identity-rev3-01.log). |
| go test ./internal/traceability -run '^TestTaskTemporaryOwnershipDigest$' -count=1 -v | 0 | Temporary test source removed afterward. Re-derived canonical SHA-256 bc46c7705670bc331811a32eb25a5712331fd11248c207a61c7544023f84107b; registry-digest-rederive-rev3-01.log. |
| env -u TMPDIR go run ./internal/traceability/cmd/tracecheck | 0 | contracts=64, normative_sections=36, acceptance_cases=163, bindings=73; clauses 106/597; tracecheck-rev3-01.log. README pin equals this line. |
| README pin: go test ./internal/traceability/cmd/tracecheck -run '^TestREADMEMeasuredCoverageMatchesTracecheckReport$' -count=1 -v | 0 | readme-coverage-pin-rev3-01.log |
| Same README test in a task-scoped copy with 106/597 changed to 107/597 | 1, expected red | Test rejects the wrong figure against tracecheck; readme-coverage-plant-rev3-01.log. Scratch copy removed after the run. |
| Current go list importer census vs rev2 candidate census | 0 | Identical graph; importers-candidate-rev3.log; empty importer-census-rev3-delta.log. |

Full-suite evidence is composed according to unchanged-identity reuse: the prior env -u TMPDIR go test ./... -count=1 run passed all 46 package entries (go-test-all-rework-final-02.log). Rev3 changed only the internal/tmuxserver source/tests and harness; that entire package was rerun on rev3 with exit 0. All other package source, test, configuration and environment identities are unchanged, and the current importer graph is byte-identical to rev2. Thus current package outcomes are green for 46 of 46 packages without replaying unchanged packages. The previous full go test ./... -cover attempt exited 1 on the repository's 10-minute package timeout; unchanged packages also have the nine green coverage shards and 46/46 shard audit. Rev3 reran filtered capture coverage (35.0%) on the changed package. No full one-shot go test ./... or go test ./... -cover command was rerun on rev3; the exact reuse boundary and logs are stated here rather than represented as such a command.

The independent base/candidate importer grid and moved classes are in TASK-260830-2xt6fd_importer-outcome-grid-rev3.md, together with the rev2 full grid. The current rev3 importer graph diff is empty.

## Out-of-contract rows and acceptance-criteria bounds

| Out-of-contract row | Acceptance-criteria clause that bounds it |
| --- | --- |
| §10.4#3 checkpoint/provider/task-board manifest selection and conflict resolution | “Drive complete Git transfer-object assembly through the real capture path … workspace root/child manifest closure.” This leaf assembles the explicitly supplied members and their root/child closure; it does not select history. |
| §10.4#5 hardlink target semantics | “Detect source changes during capture and prove manifest closure, path safety, byte identity …” The output carries Git tree/blob identity; this leaf does not preserve inode/hardlink relationships or materialize a destination tree. |
| §10.4#15 provider transfer-response admission | “Truthful unsupported cases” and “no unsupported capability is advertised.” There is no external provider transfer-response entry in this local Git capture leaf. |
| §10.4#25 cross-platform conditional support | “No unsupported capability is advertised.” Only the tested host/runtime is evidenced; no other provider/platform cell is claimed. |
| §12.1#1 checkpoint linkage | “Workspace root/child manifest closure” requires the capture closure, not publishing a checkpoint reference. This leaf has no checkpoint publisher. |
| §12.2#3–#6 destination materialization gates | “Transfer-object assembly through the real capture path” asks for pack/index/manifest assembly; the leaf does not write or replay a destination checkout. |
| §12.3 steps 2–8 materialization | “Detect source changes during capture” and assemble transfer objects; destination materialization/replay is not requested. |
| Capture returning to active | “Detect source changes during capture” does not request a live-session return; §4.C is closed and has no release transition. |

Crash/idempotency remains in scope because the acceptance criterion requires it when durable state mutates; the rev3 filtered coverage run passed the named pack and close-status crash phases and same-store retries.


## Candidate refresh check

Immediately before handoff, origin/main was read as 6d3bff999f75ed8585a7ef32074ab108f364ec51, the existing refreshed trunk base. The Story worktree HEAD remains the recorded checkpoint, and the candidate remains uncommitted; no additional refresh was needed. Evidence: origin-main-pre-handoff-rev3.log.
