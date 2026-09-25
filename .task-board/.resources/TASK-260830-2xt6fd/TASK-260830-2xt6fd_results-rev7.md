# TASK-260830-2xt6fd — developer rework results rev7

## Candidate and scope

- Rework class: `repeat-of: capture-gate-disjunct-unmeasured`; the rev5 prior verdict and rev6 rework are preserved in the task resources.
- Story branch: `task-board/story/STORY-260830-35dbcs`, checkpoint `1121b4ab54709a9966fbbcc359230d15b0e7592b`; current `origin/main` base observed as `eb12183056eb86dba663540518c6ac8f79f56a87`. The final scratch-index tree is `531a9e60aac785325e3de407b950e5ead241804e`. The candidate remains uncommitted; no live Git index was written. The only tracked edit made after the prior complete suite was this rev7 LOGBOOK entry. Source, test, README, registry, and config identities are unchanged from the prior complete suite.
- The refreshed Story invariant had 0 blob mismatches across 2,547 trunk-only paths. Final trunk fetch/check is reserved for immediately before handoff.
- Scope stays within the Story capture implementation, its owner integration and traceability: `internal/gitsnap`, `internal/provhost`, `internal/tmuxserver`, `internal/traceability`, `README.md`, and `LOGBOOK.md`. No `internal/merkleinventory` production or test file changed.

## Rev6 repeat-of and held quiescence

The capture AST census recursively splits every `||` in the guards of reachable `captureUnavailable(...)` sites and counts 27 disjunct clauses. The matrix in `TASK-260830-2xt6fd_capture-admission-matrix-rev6.md` has one production-entry row for every clause, with literal refusal code and detail. Each row derives from a valid control request and varies only that clause. The two allowed controls (quiescing with a current provider boundary, and stopped without a boundary) pass. All 27 row narrowings were killed alone twice. The unlisted-site, non-literal-detail and added-disjunct census controls were each killed alone twice.

The rev5 repeat-of concerning the empty provider-boundary timestamp is covered by `TestCaptureOwnerBoundaryEmptyTimestampRepeatOf`; `N-capture-admits-empty-provider-boundary-timestamp` was killed alone twice. Prior reviewer test names remain present. The owner census control and behavior suite are both executed.

Capture remains the pinned §4.C stop-point operation: `CaptureGitWorkspace` admits only stopped or quiescing with a current safe boundary; it composes landed owner calls, retains the held state, and does not add a release/restore transition. Crashed quiescing captures recover via the existing owner stop path and retry from stopped. The v0.5.0→v0.7.0 clause crosswalk records byte-identical clause bodies and current citations: §4.C operation table lines 1198–1215, §12.2 starting at 9047, §12.3 starting at 9074.

## Measured acceptance coverage

The brief supplied no surface table. This is reported as a brief gap; the coverage map still covers all six acceptance rows. Measured ratio: **6 of 6 acceptance rows driven** through these production call sites.

| Acceptance row | Production call site | Evidence map |
| --- | --- | --- |
| Held input/provider quiescence | `(*tmuxserver.Lifecycle).CaptureGitWorkspace` | Stop-point state domain, receipt/proof refusals, closure barrier and close-state checks; per-disjunct capture matrix |
| Source change during capture and retry | `CaptureGitWorkspace` → `gitsnap.AssembleProvisional`; `gitsnap.Capture` | `TestCaptureGitWorkspaceSourceChangeRefusalAndRetry`, `TestAssemblySourceChangesAndRetry`, `TestContentDigestRaceAndRetry` |
| Repository-local packs, exact inventories, raw/logical indexes | `gitsnap.AssembleProvisional` → `captureAssembly` / `assemblePack` | Pack/index/byte and inventory tests in coverage map; named narrowing mutants |
| Workspace root/child, blobs, cwd/config, safe-path closure | `gitsnap.AssembleProvisional` → `ValidateProvisional` | Closure/descriptors, group identity, config and parent/child-overlap tests |
| Actual bytes, recursive dirty state and truthful unsupported inputs | `gitsnap.AssembleProvisional` → `gitsnap.Capture` with `ContentOptions`; `tmuxserver.UnixDialer.Dial` for status observation | Normative corpus, symlink/submodule, unsupported-source and absent-vs-failed-read tests |
| Truthful capability, crash/idempotency and non-publication | `CaptureGitWorkspace` → `gitsnap.AssembleProvisional` | Owner recovery, capture/assembly crash retry and store-failure tests |

The detailed test-to-mutant mapping is attached as `TASK-260830-2xt6fd_coverage-map-rev7.md`; generated ranges and structural bounds are in `TASK-260830-2xt6fd_axis-inventory-rev7.md`.

## Whole-Story synchronization-property audit

The inherited `DurableIndex.SyncFrom` product test was inspected and exercised. `TestDurableSyncGeneratedPerturbationProduct` generates N=1..3 in the configured suite and N=1..6 under nested selectors; it enumerates every permutation through N=5 and 34 deterministic permutations at N=6. Dimensions include both clock-skew directions, duplicate replay absent/present, no gap or a cyclically delayed object, every proper peer subset, repeated valid sync passes 1..3, and feasible conflict target × conflict round 1..3. The object fixture mixes a repository record, two competing lease records, event, Tombstone and Tombstone Acknowledgement. The oracle is separate reference code for tuple winner selection, Merkle roots, and byte snapshots. N and rounds are generated ranges; six objects and three rounds are the stated fixture bounds, not claims over arbitrary set size or rounds. Production `SyncFrom` iterates complete identity slices and fetches in 4096-object chunks; larger-scale behavior remains unmeasured.

| Mutant | What it narrows | Named failing test | Result |
| --- | --- | --- | --- |
| `union-lease-created-at-winner` | Uses time as a tie-break for equal-epoch lease heads | `TestDurableSyncClockSkewDoesNotSelectLeaseWinner` | KILLED twice alone |
| `sync-arrival-order-truncates-fetch` | Drops an identity under one arrival order | `TestDurableSyncArrivalOrderConverges` | KILLED twice alone |
| `sync-partial-peer-regresses-local-root` | Admits regression from a proper peer subset | `TestDurableSyncPartialPeerNeverRegressesLocalRoots` | KILLED twice alone |
| `sync-repeat-record-quarantine-write` | Rewrites durable quarantine bytes on repeated conflict sync | `TestDurableSyncGeneratedPerturbationProduct` conflict pass | KILLED twice alone |
| `sync-common-audit-partial-overlap-peer` | Skips common-object validation for a partial-overlap peer | `TestDurableSyncConflictWithPartialOverlapPeer` | KILLED twice alone |
| `sync-duplicate-delivery-drops-active-object` | Removes a persisted active object during identical AddJSON replay | `TestDurableJSONAddIsIdempotentAcrossCrashRestartByNamespace` | KILLED in three isolated runs |

The duplicate-drop narrowing first survived `TestDurableSyncDuplicateDeliveryIsByteIdentical`: its snapshot begins after initial delivery and therefore missed an object already removed. The same one-condition plant was rerun against the existing crash/reopen idempotency test and was killed in three runs; both the initial survivor and all killed raw logs are retained. No merkleinventory source or test was edited.

## Survivor table

| Mutant | What it narrows / named test | Bound that explains survival |
| --- | --- | --- |
| `D-attach-entry-deadline-shadowed` | Admits the exact on-deadline instant at `TestExecuteAttachRefusesDeadline/boundary` | Downstream wait-bound rejects the same instant before admission. It is supplementary; the independent `N-attach-postlock-deadline` and `N-attach-wait-deadline` rows are killed. |
| `N-assembly-closure-admits-one-missing-child` | Allows one missing child at `TestValidateProvisionalClosureAndDescriptors` | The malformed fixture still fails a required repository-tree binding; this plant is not independent child-visit evidence. |
| `C-control` | Comment-only edit; `TestSocketPathDerivesOnlyFromRuntimeDir` | Expected behavior-neutral harness control; no behavior changes. |

The current shipped tmuxserver manifest has 393 rows. Its broad split attempt recorded 391 outcomes before one group was interrupted; the two missing rows were run separately and killed twice. The resulting 393-row account is 391 KILLED and the two survivors above that belong to that harness (`D-attach-entry-deadline-shadowed`, `C-control`). The rev6 broad harness group containing the interruption is not represented as a green invocation. The six capture/product tables above carry their own isolated raw evidence.

## Out-of-contract rows

| Row | Acceptance-criteria clause that bounds this leaf |
| --- | --- |
| §10.4#3 checkpoint/provider/task-board manifest selection and conflict resolution | The AC requires explicit workspace root/child closure; it does not select checkpoint/provider/task-board history. |
| §10.4#5 hardlink target semantics | The AC requires Git tree modes and blob identity, not inode preservation or destination materialization. |
| §10.4#15 `PROVIDER-CAPTURE-N1` transfer-response admission | The AC requires truthful unsupported cases; this local capture entry has no provider transfer-response entry. |
| §10.4#25 cross-platform conditional support | The AC requires no unsupported capability be advertised; evidence is for tested host/runtime only. |
| §12.1#1 checkpoint linkage | Root/child closure is required; publishing a checkpoint referencing the root is not. |
| §12.2#3–#6 parser-before-destination materialization gates | This leaf assembles Git pack/index/manifest output and does not write/replay a destination checkout. |
| §12.3 steps 2–8 materialization | This AC asks for capture-time race detection and transfer-object assembly, not destination materialization/replay. |
| Capture returning to `active` | §4.C has a closed operation vocabulary and no release transition; a live return is a spec-change bound. |

## Registry, importer and validation

- Registry JSON was decoded for all three Story leaves; acceptance cases and clause edges point to their implementing production owners. Re-derived `reviewedOwnershipCanonicalSHA256=b3c19a372b72d48f01757bc209162e8da7d3fd4a56908169fc18c7d9009314d2`. Full tracecheck exited 0: 64 contracts, 36 normative sections, 171 acceptance cases, 76 bindings, 113/622 clauses discharged. README pin test exited 0. A scratch README plant changed the displayed bindings 76→77; `TestREADMEMeasuredCoverageMatchesTracecheckReport` failed as expected (exit 1).
- Section-scoped diagnostics exited 1 because this leaf reports partial aggregate clause coverage: §10.4 21/25, §12.1 0/1, §12.2 2/6, §12.3 2/5, §4.C 5/7, §4.2 5/11. These are recorded as partial results, not full section passes.
- Importer outcome grid was rerun on base `origin/main` `eb12183056eb86dba663540518c6ac8f79f56a87`, with 51 base vs 52 candidate packages and eight touched-package comparisons. No shared test key moved status. The gitsnap package is absent at base; the detailed importer report names every package/key membership change and raw JSONL path.
- `go build ./...`, `go vet ./...`, and `GOOS=windows GOARCH=amd64 go vet ./...` exited 0. `golangci-lint run --new-from-rev=HEAD` exited 0 with 0 issues; it emitted a nonfatal missing-file warning for an unrelated sibling worktree. `gofmt` and `git diff --check` were clean. Eight bounded coverage shards all exited 0 across 52 packages. The focused capture admission determinism suite at `-count=3` exited 0; a broader `-run '^TestCapture' -count=3` attempt was interrupted (exit 1, `signal: interrupt`) and is not pass evidence.
- Full `go test ./...` and `go test ./... -v` passed with exit 0 on the candidate's unchanged source/test/config/environment identity before the latest LOGBOOK-only append; the attached verbose log ends in `PASS` and covers all 52 packages. After the append, one exact command attempt received SIGTERM at `internal/tmuxserver` while two other full Go suites were observed running; an isolated `go test ./internal/tmuxserver -count=1` then passed (exit 0, 220.577s). A second all-package retry was stopped at 9m18s (exit 143) to remain within the single-call time bound while concurrent suites were still active; its partial log is not counted as a pass. This follows the task instruction to reuse complete green evidence when source, tests, config and environment identity are unchanged.

The task-scoped raw logs and focused evidence are in `TASK-260830-2xt6fd_evidence-rev7.tar.gz` (under the 1 MiB limit). Prior attached rev6 evidence remains relevant for unchanged validations.
