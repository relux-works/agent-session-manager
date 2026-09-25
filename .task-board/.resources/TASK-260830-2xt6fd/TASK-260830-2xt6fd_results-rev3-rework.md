# Developer handoff addendum — TASK-260830-2xt6fd rev3 rework

Candidate is ready for developer handoff to review. The Story worktree remains uncommitted at checkpoint `49d1d660c11c4e1f1564d72ab7e28b3f6cd63203`.

## Rework delta and scope

Relative to the pre-rework candidate tree `f7f04778d7f82cccf23015d5c939483ef2d0c4ad`, the scratch-index tree is `13a8375c0e98fc5dec425389532d6d4e22417959`. The rework diff has one path only: `internal/tmuxserver/capture_git_workspace_test.go` (6 insertions, 2 deletions). It strengthens `TestCaptureAdmission_StoppedBoundaryBodyForbidden` with both status observations needed to let a narrowed gate reach successful capture, and changes `TestCaptureAdmission_IncarnationStableAcrossCapture` to use the stopped path so the independent quiescence barrier cannot mask the closing-incarnation check. No production code, test names, registry, README, or task configuration changed in this rework.

The AST census and literal code/detail matrix cover 22/22 reachable `captureUnavailable` sites. The census rejects an unmapped site and a non-literal detail. All acceptance-map tests continue to enter through the production call site `(*Lifecycle).CaptureGitWorkspace`; the full acceptance map remains 6/6 rows, as detailed in `TASK-260830-2xt6fd_coverage-map-rev2.md`. The brief supplied no surface table; this is recorded as a brief gap, not a waiver.

## Narrowing evidence

Each site plant was applied alone; the harness ran the named test alone. Every row below was KILLED twice. The two repaired fixtures now fail their expected literal refusal assertion with the narrowed gate admitted and capture returning `<nil>`.

| Mutant | Narrowing | Named test that fails | Result |
| --- | --- | --- | --- |
| `N-capture-admits-nil-lifecycle` | Admits nil lifecycle for one valid fixture group | `TestCaptureAdmission_LifecycleRequired` | KILLED twice |
| `N-capture-admits-session-scoped-status` | Admits status missing exact instance identity | `TestCaptureAdmission_ExactStatusIdentityRequired` | KILLED twice |
| `N-capture-admits-opening-status-nonmatch` | Admits one opening status identity mismatch | `TestCaptureAdmission_OpeningStatusIdentityMatch` | KILLED twice |
| `N-capture-admits-absent-opening-state` | Admits absent at the opening state gate | `TestCaptureAdmission_InitialAbsent` | KILLED twice |
| `N-capture-admits-parked` | Admits parked at the opening state gate | `TestCaptureAdmission_InitialParked` | KILLED twice |
| `N-capture-admits-active-opening-state` | Admits active at the opening state gate | `TestCaptureAdmission_InitialActive` | KILLED twice |
| `N-capture-admits-stale-fenced-opening-state` | Admits stale_fenced at the opening state gate | `TestCaptureAdmission_InitialStaleFenced` | KILLED twice |
| `N-capture-admits-unavailable-opening-state` | Admits unavailable at the opening state gate | `TestCaptureAdmission_InitialUnavailable` | KILLED twice |
| `N-capture-admits-nil-assembly-runner` | Admits a missing runner for one fixture group | `TestCaptureAdmission_AssemblyRunnerRequired` | KILLED twice |
| `N-capture-admits-stopped-boundary-body` | Admits a stopped request carrying a boundary body | `TestCaptureAdmission_StoppedBoundaryBodyForbidden` | KILLED twice; current raw logs show `<nil>` refusal result |
| `N-capture-admits-stopped-without-incarnation` | Admits a stopped request without a current incarnation | `TestCaptureAdmission_StoppedIncarnationRequired` | KILLED twice |
| `N-capture-admits-quiescing-without-boundary` | Admits quiescing without a boundary body | `TestCaptureAdmission_QuiescingBoundaryRequired` | KILLED twice |
| `N-capture-admits-boundary-identity-drift` | Admits a mismatched boundary identity | `TestCaptureAdmission_BoundaryIdentityMatchesStatus` | KILLED twice |
| `N-capture-admits-checkpoint-proof` | Admits `ax_checkpoint_boundary` as provider proof | `TestCaptureAdmission_CheckpointOnlyProviderProof` | KILLED twice |
| `N-capture-admits-quiescing-without-incarnation` | Admits quiescing without a current incarnation | `TestCaptureAdmission_QuiescingIncarnationRequired` | KILLED twice |
| `N-capture-admits-foreign-quiesce-receipt` | Admits one foreign input-closure receipt | `TestCaptureAdmission_CurrentInputClosureReceiptRequired` | KILLED twice |
| `N-capture-admits-foreign-provider-boundary-receipt` | Admits one foreign provider-boundary receipt | `TestCaptureAdmission_CurrentProviderBoundaryReceiptRequired` | KILLED twice |
| `N-capture-admits-closing-status-nonmatch` | Admits one closing status identity mismatch | `TestCaptureAdmission_ClosingStatusIdentityMatch` | KILLED twice |
| `N-capture-admits-changed-closing-incarnation` | Admits a rotated incarnation at close | `TestCaptureAdmission_IncarnationStableAcrossCapture` | KILLED twice; current raw logs show `<nil>` refusal result |
| `N-capture-admits-final-parked-state` | Admits parked as the closing state | `TestCaptureAdmission_FinalStateRemainsHeld` | KILLED twice |
| `N-capture-admits-stopped-to-quiescing-for-one-group` | Admits a stopped capture that becomes quiescing | `TestCaptureAdmission_StoppedRemainsStopped` | KILLED twice |
| `N-capture-admits-unproven-barrier-for-one-group` | Admits a lost closing barrier for one group | `TestCaptureAdmission_ClosureBarrierStillProven` | KILLED twice |

The applied `C-capture-census-unlisted-refusal-site` and `C-capture-census-nonliteral-detail` controls were KILLED by `TestCaptureCoordinatorAdmissionGateCensus`. `N-capture-restores-with-boundary-token-preserved` was KILLED by its AST transition test and successful-capture behavior test. The applied comment-only `C-control` SURVIVED as expected and proves the harness reports survivors. A carried assembly-closure plant remains a declared survivor: removing the local child-visit refusal does not admit the fixture because the required outer tree-binding gate still refuses it; it is not counted as a kill or as independent child-visitor coverage.

Raw logs are packaged one per plant per run. Pass summaries show 22/22 capture-site mutants killed in each pass, plus the control results. A prior compiler failure and the broad-shard timeout are retained as diagnostics, not counted as kills or green checks.

## Validation results

All gate and validation commands were direct standalone processes; log paths are inside the attached evidence archive unless marked as carried evidence.

| Command / evidence | Exit | Result |
| --- | ---: | --- |
| `go test ./internal/tmuxserver -count=1` | 0 | Full changed package passed on the current candidate in 597.398s. |
| Capture admission/state/census matrix with `-count=3 -v` | 0 | Three deterministic repetitions passed. |
| `TestCaptureAdmission_StoppedBoundaryBodyForbidden` alone | 0 | Passed on the repaired fixture. |
| `TestCaptureAdmission_IncarnationStableAcrossCapture` alone | 0 | Passed on the repaired fixture. |
| Two complete capture-site harness passes | 0 each | 22/22 narrowing plants killed alone in each pass. |
| Census controls, transition narrowing, and harmless control | 0 | Census plants and transition narrowing killed; comment control survived as expected. |
| `env -u TMPDIR GOOS=windows GOARCH=amd64 go vet ./...` | 0 | Fresh Windows/amd64 vet on the current candidate. |
| `env -u TMPDIR go vet ./...` | 0 | Fresh host vet. |
| `env -u TMPDIR go build ./...` | 0 | Fresh host build. |
| `env -u TMPDIR go run ./internal/traceability/cmd/tracecheck` | 0 | `contracts=64`, `normative_sections=36`, `acceptance_cases=163`, `bindings=73`, `clauses=106/597`. |
| README measured-coverage pin test | 0 | Current README pin matches tracecheck's `106/597`. |
| `git diff --check` and scratch-index `git diff --cached --check` | 0 | Clean. |
| Scratch-index `read-tree`, `add -A`, `write-tree` | 0 each | Scratch index only; candidate tree `13a8375c0e98fc5dec425389532d6d4e22417959`. |
| Live-index/config hygiene | 0 | Live index hash remained `0d837dfc6f07e8517a48e0bd37cf3857cabda5eb55ae19fc9ee23e9fcbbc8107`; no staged diff; `task-board.config.json` byte-matches `HEAD`. |
| Current remote `main` probe | 0 | `6d3bff999f75ed8585a7ef32074ab108f364ec51`; no refresh required. |

The full configured test set is green by package identity: the prior full run passed 46/46 package entries on the rev3 candidate; this rework changes tests only in `internal/tmuxserver`, and the entire package was rerun on the current candidate with exit 0. The other 45 package source, test, configuration, and environment identities are unchanged, so their exact green evidence is reused. A literal monolithic `go test ./...` was not rerun on the current tree: the `tmuxserver` package alone took 597.398s, while the shell gate is bounded at about ten minutes. This is an explicit command-level bound; the current package-wise result covers 46/46 package identities.

A separate broad `Test[A-D]` shard used `-timeout=8m` and exited 1 at 480.249s while the existing `TestCustodyPathComponentByteLengthsAtProductionEntries` was still executing. This was a shard-sizing timeout, not a product assertion. The current full package rerun above passed that package's tests. An earlier deliberate wrong-README-pin plant exited 1 as expected; its log is carried from the unchanged rev3 validation.

Registry and importer evidence remains unchanged by this test-only rework: the story-final registry was decoded and mapped for all three leaves; canonical SHA-256 was rederived as `bc46c7705670bc331811a32eb25a5712331fd11248c207a61c7544023f84107`; tracecheck and README pin pass above. The prior base/candidate importer grid records 193 shared keys, zero moved, zero base-only, and seven declared candidate-only keys over its measured importer set; the current rework adds no import or production outcome. See `TASK-260830-2xt6fd_importer-outcome-grid-rev3.md` and the prior results artifact. The wrong-pin README plant and full importer comparison were not rerun after this test-only delta.

## Handoff preconditions

1. No surface table was attached to the brief; this is stated as a brief gap. The six acceptance rows are mapped 6/6 through production call sites in the coverage map, and all 22 coordinator refusal sites have one literal code/detail matrix row and one killed narrowing.
2. The landing property and current changed-package suite are green as recorded above.
3. Previously committed reviewer tests remain present under their original names; no test name was changed or removed, and the full `internal/tmuxserver` package suite passes.
4. Out-of-contract rows and their acceptance-criteria clauses remain enumerated in `TASK-260830-2xt6fd_coverage-map-rev2.md`: (a) §10.4#3 manifest selection/history conflicts — the acceptance clause asks this leaf to assemble supplied members and root/child closure; (b) §10.4#5 hardlink semantics — the acceptance clause asks Git byte identity, not inode preservation/materialization; (c) §10.4#15 provider transfer-response admission — the clause is truthful unsupported reporting and this local capture leaf has no provider transfer-response entry; (d) §10.4#25 other platform/provider cells — no unsupported capability is claimed beyond tested evidence; (e) §12.1#1 checkpoint linkage — the clause requires workspace closure, not checkpoint publication; (f) §12.2#3–#6 destination materialization — the clause asks transfer-object assembly, not destination checkout writes; (g) §12.3 steps 2–8 destination replay — the clause asks capture and assembly, not materialization; (h) same-incarnation return to `active` — pinned §4.C has a closed state vocabulary with no release transition.
5. The rework diff is limited to `internal/tmuxserver/capture_git_workspace_test.go`, the capture module. No production files were added outside that scope.

The stop-point decision remains the orchestrator decision: capture requires stopped or current-boundary quiescing and does not release/reopen; recovery uses the landed owner stop/stale-termination path. See the v0.7.0 crosswalk and the prior task results for citations and the runtime owner call chain.
