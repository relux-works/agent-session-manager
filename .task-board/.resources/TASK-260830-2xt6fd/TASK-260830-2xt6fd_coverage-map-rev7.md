# Coverage map — TASK-260830-2xt6fd

## Brief gap

The assignment supplied no surface table. This is a brief gap, not a silent
coverage waiver. The map below covers every row in the six-row acceptance
matrix in `internal/gitsnap/testdata/assembly-acceptance.md`.

| Acceptance row | Production call site | Named test(s) | Narrowing evidence |
| --- | --- | --- | --- |
| Held input/provider quiescence | `(*tmuxserver.Lifecycle).CaptureGitWorkspace` | `TestCaptureStopPointStateDomain`, `TestCaptureAdmissionGateValidControls`, `TestCaptureProviderProofRequirements`, `TestCaptureRejectsCheckpointBoundaryAsProviderProof`, `TestCaptureRejectsQuiesceReceiptFromAnotherIncarnation`, `TestCaptureRejectsBarrierLostAfterAssembly`, `TestCaptureStoppedMustRemainStoppedAtClose`, `TestCaptureFinalStatusStateDomain`, `TestCaptureNeverTransitionsOutsideStopPointOwnerSet`, `TestCaptureCoordinatorAdmissionGateCensus` | The stop-point table admits only valid quiescing-with-boundary and stopped rows. Seven narrowing plants (`N-capture-admits-parked`, the four individually named coordinator gates, `N-capture-restores-with-boundary-token-preserved`, and `N-unix-dialer-missing-parent-admits-stale`) were each run alone twice and killed. The four gate narrowings change only the predicate, preserve the refusal branch/code/detail tokens, and execute the named behavioral test; they do not rely on the source census alone. The census control rejects an added refusal site; the harmless `C-control` survived both runs. |
| Source change during capture and retry | `(*tmuxserver.Lifecycle).CaptureGitWorkspace` → `gitsnap.AssembleProvisional`; `gitsnap.Capture` | `TestCaptureGitWorkspaceSourceChangeRefusalAndRetry`, `TestAssemblySourceChangesAndRetry`, `TestContentDigestRaceAndRetry` | `N-assembly-source-admits-working-file-change` kills `TestAssemblySourceChangesAndRetry`; `N-consistency-admits-five-byte-digest-change` kills `TestContentDigestRaceAndRetry`. |
| Repository-local packs, exact inventories, raw/logical indexes | `gitsnap.AssembleProvisional` → `captureAssembly` / `assemblePack` | `TestAssemblyPackIndexAndBytes`, `TestAssemblyRecursivePacks`, `TestAssemblyIndexCompatibility`, `TestAssemblyRawIndexRefusals`, `TestAssemblyFailedReadsAndCorruptObjects`, `TestAssemblyIsolatedDatabaseIgnoresAmbientObjects` | `N-assembly-inventory-admits-four-object-metadata-mismatch`, `N-assembly-raw-index-admits-two-entry-disagreement`, and `N-assembly-checksum-admits-last-bit-flip` each kill their named test. |
| Workspace root/child, blobs, cwd/config, safe-path closure | `gitsnap.AssembleProvisional` → `ValidateProvisional` | `TestValidateProvisionalClosureAndDescriptors`, `TestAssemblyGroupRecordAgreement`, `TestAssemblyMissingConfigClosure`, `TestAssemblyRecursiveCWDAndConfigClosure`, `TestValidateProvisionalParentChildOverlap` | `N-assembly-descriptor-admits-seven-byte-entry`, `N-assembly-group-admits-repository-identity-mismatch`, `N-assembly-closure-admits-one-unbound-tree`, `N-assembly-path-admits-missing-config-token-preserved`, and `N-assembly-path-admits-one-parent-partition-entry` each kill their named test. `N-assembly-closure-admits-one-missing-child` survives because the same fixture is rejected by another parent/tree binding; it is not counted as independent evidence. |
| Actual bytes, recursive dirty state, truthful unsupported inputs | `gitsnap.AssembleProvisional` → `gitsnap.Capture` with `ContentOptions`; `tmuxserver.UnixDialer.Dial` for status observation | `TestAssemblyNormativeCorpus`, `TestAssemblyRecursivePacks`, `TestAssemblyIndexCompatibility`, `TestAssemblyOptionsAndUnsupportedSources`, `TestContentSubmoduleUninitializedAndRefusals`, `TestContentSymlinkChainsAndExcludedTargets`, `TestContentSymlinkChainBoundaryAtProductionCapture`, `TestUnixDialerMissingSocketIsStale`, `TestUnixDialerMissingDirectoryIsUnknown` | `N-path-admits-escaping-chain`, `N-submodule-admits-stray-uninitialized`, `N-assembly-features-admits-origin-promisor`, `N-assembly-policy-admits-unclassified-extra`, and `N-unix-dialer-missing-parent-admits-stale` each kill their named production-entry test. A missing parent or failed parent read remains unknown under the §4.C successful-status-only absence rule. |
| Truthful capability, crash/idempotency, non-publication | `(*tmuxserver.Lifecycle).CaptureGitWorkspace` → `gitsnap.AssembleProvisional` | `TestCaptureFailureRetryUsesRequestStop`, `TestCaptureProcessCrashRetry`, `TestAssemblyCrashAfterObjectsAndRetry`, `TestAssemblyStoreFailureAndRetry` | `N-assembly-install-admits-shard-failure` kills `TestAssemblyStoreFailureAndRetry`; the stop-point narrowings above cover refusal. Crash exits 74/75 are expected child exits, followed by same-store retry. |

## Measured narrowing results

Each `BEHAVIOR_KILL` below was produced by an applied source plant, a successful
compile, and a nonzero behavior-test exit containing the named test failure.
The `closure-admits-one-missing-child` row is the only surviving gate plant.

| Mutant | What the plant admits | Named failing test / result |
| --- | --- | --- |
| `N-path-admits-escaping-chain` | One named symlink chain bypasses the safety walk | `TestContentSymlinkChainsAndExcludedTargets` — killed |
| `N-submodule-admits-stray-uninitialized` | One uninitialized submodule with stray entries bypasses refusal | `TestContentSubmoduleUninitializedAndRefusals` — killed |
| `N-consistency-admits-five-byte-digest-change` | Digest mismatch is ignored for a five-byte source | `TestContentDigestRaceAndRetry` — killed |
| `N-entry-set-admits-65537` | Shared manifest entry cap increases by one | `TestCheckManifestEntriesCountBounds` — killed |
| `N-assembly-raw-index-admits-two-entry-disagreement` | A two-entry raw/logical index mismatch is admitted | `TestAssemblyFailedReadsAndCorruptObjects` — killed |
| `N-assembly-checksum-admits-last-bit-flip` | A raw index checksum differing by one low bit is admitted | `TestAssemblyRawIndexRefusals` — killed |
| `N-assembly-source-admits-working-file-change` | A changed included working file is ignored by the closing observation | `TestAssemblySourceChangesAndRetry` — killed |
| `N-assembly-group-admits-repository-identity-mismatch` | Group/member repository identity disagreement is ignored | `TestAssemblyGroupRecordAgreement` — killed |
| `N-assembly-descriptor-admits-seven-byte-entry` | A seven-byte entry may disagree with its descriptor size | `TestValidateProvisionalClosureAndDescriptors` — killed |
| `N-assembly-closure-admits-one-missing-child` | One missing child may be treated as reached | `TestValidateProvisionalClosureAndDescriptors` — **survived**; another required repository-tree binding refuses the same malformed fixture, so this plant proves no independent missing-child visit gate. |
| `N-assembly-path-admits-missing-config-token-preserved` | The named missing project config bypasses path closure | `TestAssemblyMissingConfigClosure` — killed |
| `N-assembly-features-admits-origin-promisor` | `remote.origin.promisor=true` is treated as supported | `TestAssemblyOptionsAndUnsupportedSources` — killed |
| `N-assembly-policy-admits-unclassified-extra` | One unclassified include-policy value is admitted | `TestAssemblyInvalidIncludePolicy` — killed |
| `N-assembly-closure-admits-one-unbound-tree` | One extra reachable workspace tree may lack a root/member binding | `TestValidateProvisionalClosureAndDescriptors` — killed |
| `N-assembly-inventory-admits-four-object-metadata-mismatch` | Exact reachable-object metadata mismatch is admitted for a four-object pack | `TestAssemblyFailedReadsAndCorruptObjects` — killed |
| `N-assembly-install-admits-shard-failure` | One object-store shard-creation failure is ignored | `TestAssemblyStoreFailureAndRetry` — killed |
| `N-assembly-path-admits-one-parent-partition-entry` | One named child path may overlap the parent partition | `TestValidateProvisionalParentChildOverlap` — killed |
| `N-capture-admits-parked` | A parked Terminal Instance passes the stop-point admission gate | `TestCaptureStopPointStateDomain` alone — killed; the unsupported row asserts its literal code and detail. |
| `N-capture-admits-checkpoint-proof` | Admits exactly `ax_checkpoint_boundary` at the provider-proof gate | `TestCaptureRejectsCheckpointBoundaryAsProviderProof` alone — killed; starts from the helper's valid Runner, Assembly, same-incarnation closure receipt and barrier, then changes only the proof kind; asserts literal code and detail. |
| `N-capture-admits-foreign-quiesce-receipt` | Admits a quiesce receipt from another incarnation while the separate provider-boundary receipt remains current | `TestCaptureRejectsQuiesceReceiptFromAnotherIncarnation` alone — killed; asserts literal code and receipt detail. |
| `N-capture-admits-unproven-barrier-for-one-group` | Admits a lost closing input-closure barrier for one group ID | `TestCaptureRejectsBarrierLostAfterAssembly` alone — killed; invalidates the closure and provider receipts only after pack creation and asserts literal code and detail. |
| `N-capture-admits-stopped-to-quiescing-for-one-group` | Admits a stopped capture that changes to quiescing at its closing read | `TestCaptureStoppedMustRemainStoppedAtClose` alone — killed; mutates state after assembly and asserts literal code and detail. |
| `C-capture-census-unlisted-refusal-site` | Adds an unlisted direct refusal under the proof-kind branch | `TestCaptureCoordinatorAdmissionGateCensus` alone — killed; census reports two direct refusal sites for one matrix row. This is a census control, not a gate narrowing. |
| `N-capture-restores-with-boundary-token-preserved` | The coordinator sends `restore` where it must call `wait-safe-boundary`, while receipt/parser tokens remain | `TestCaptureGitWorkspaceQuiescedAssemblesAndRechecks` and `TestCaptureNeverTransitionsOutsideStopPointOwnerSet` — both killed; raw log records both failures. |
| `N-unix-dialer-missing-parent-admits-stale` | An absent containing directory with an ENOENT socket lookup is misclassified as stale | `TestUnixDialerMissingDirectoryIsUnknown` alone — killed; raw log records test failure and process exit 1. |
| `C-control` (control only) | Comment-only source edit with no behavior change | `TestSocketPathDerivesOnlyFromRuntimeDir` passes — expected survivor; this is not a gate mutant. |

The measured acceptance-row ratio is **6 of 6 rows driven** through their
listed production call sites. This is capture/assembly scope only. It does not
claim materialization, checkpoint publication, doctor advertising, or Story
integration. No handoff claim is made that every sub-gate has a separate
narrowing beyond the named mutants above.

## Out-of-contract rows and the acceptance-criteria boundary

| Row | Acceptance-criteria clause that bounds this leaf |
| --- | --- |
| §10.4#3 checkpoint/provider/task-board manifest selection and conflict resolution | “Drive complete Git transfer-object assembly through the real capture path … workspace root/child manifest closure.” This leaf assembles the explicitly supplied Git workspace members and their root/child closure; it does not select checkpoint/provider/task-board history. |
| §10.4#5 hardlink target semantics | “Detect source changes during capture and prove manifest closure, path safety, byte identity …” The deliverable emits Git tree file modes and blob identity; it does not preserve filesystem inode/hardlink relationships or materialize a destination tree. The path-safety tests cover symlink resolution and partition overlap. |
| §10.4#15 `PROVIDER-CAPTURE-N1` transfer-response admission | “Truthful unsupported cases” and “no unsupported capability is advertised.” This leaf accepts local Git capture inputs and has no external provider transfer-response entry. |
| §10.4#25 cross-platform conditional support | “No unsupported capability is advertised.” The evidence is for the tested host/runtime; no other provider/platform cell is advertised as supported. |
| §12.1#1 checkpoint linkage | “Workspace root/child manifest closure” is required; publishing a checkpoint that references the captured root is not. No checkpoint publisher is part of this capture leaf. |
| §12.2#3–#6 parser-before-destination materialization gates | “Transfer-object assembly through the real capture path” requires the Git pack/index/manifest output. This leaf does not write or replay a destination checkout and does not claim those materialization parser gates. |
| §12.3 steps 2–8 materialization | The acceptance criterion asks to “Detect source changes during capture” and assemble Git transfer objects; it does not ask this leaf to materialize or replay a destination tree. |
| Capture returning to `active` | “Detect source changes during capture” does not ask for a live-session return transition. The §4.C operation table is closed and has no release transition; the capture is a stop-point operation. |

Crash/idempotency is in contract because the AC says “crash/idempotency evidence
is included when the operation mutates durable state”; object-store pack and
blob installation do, so child-process crash/retry evidence is included.

Raw per-plant logs and result JSON are under `.temp/TASK-260830-2xt6fd/` in
`mutants-narrowing-01/`, `mutants-closure-02/`, `mutants-extra-01/`,
`mutants-capture-02/`, `mutants-token-preserved-01/`, and
`mutants-control-01/`, plus final reruns in `mutants-capture-final-01/`,
`mutants-capture-final-02/`, `mutants-unix-dialer-final/`, and the two rev2
task-scoped runs `mutants-capture-rework-final-01/` and
`mutants-capture-rework-final-02/`. The rev2 runs each applied eight narrowing
rows and two controls independently: seven gate narrowings and the census
control were killed; the harmless `C-control` survived, as expected. Each new
four-gate test was executed alone under its matching narrowing, and both
allowed-state controls passed. The raw harness summaries are
`mutants-capture-rework-final-01-harness.log` and
`mutants-capture-rework-final-02-harness.log`; each directory contains one raw
behavior log per mutant. `capture-rework-determinism-02.log` records the nine
focused production-entry tests at `-count=3`.

The `N-assembly-closure-admits-one-missing-child` survivor remains a declared
bound: its malformed fixture is rejected by the tree binding, so this mutation
does not independently measure missing-child traversal. The observed
`creating` owner-input is also bounded: transient creation is not persisted by
`InstanceStates.Record`, and the landed status owner projects a raw creating
observation to `unavailable`; the capture test asserts that owner's literal
unavailable refusal rather than inventing a separate status path.

The README coverage pin plant changed only a scratch copy's measured `106/597`
to `107/597`; `TestREADMEMeasuredCoverageMatchesTracecheckReport` refused it
(expected-red test exit 1, `readme-coverage-plant-01.log`), and the original
README was restored byte-identically. Registry re-derivation and importer
outcome evidence are summarized in `TASK-260830-2xt6fd_results.md` and
`TASK-260830-2xt6fd_importer-outcome-grid.md`.

Four earlier broad harness controls that attempted the whole package suite
exceeded the harness's 240-second per-test limit; they are diagnostics, not
pass evidence. Their recorded paths are the four `mutants-*-run-01.log` files
for content-a/content-b/assembly-a/assembly-b.

### Rev6 repeat-of regression and run boundary

`repeat-of: capture-gate-disjunct-unmeasured` is covered by the 27 matrix rows and their row-specific narrowing mutants, each killed alone twice. The mechanical census controls plant an unlisted refusal, a computed detail, and an additional `||` disjunct; each is killed alone twice. The applied harmless `C-control` survives. The separate repeat-of class `capture-receipt-clause-unmeasured` is covered at the provider owner boundary by `TestCaptureOwnerBoundaryEmptyTimestampRepeatOf` and the narrowing `N-capture-admits-empty-provider-boundary-timestamp`, killed alone twice; the original reviewer test name remains available.

On the refreshed candidate, `go vet ./...`, Windows `go vet`, `go build ./...`, and `git diff --check` exited 0. The full `go test ./... -v` was terminated at the ten-minute bounded-command limit (exit 143), so it is not green evidence. The importer outcome grid was measured on base `5b7876b` versus pre-refresh candidate tree `087ac14`; it does not attest the later refreshed tree. Current tracecheck and the README measured-coverage pin test exited 0, and the wrong README pin plant exited 1 as expected. The brief supplied no surface table; that is a brief gap, not a coverage waiver. Final-leaf AC ratio: **6 of 6**.


## Rev7 whole-Story property audit

The review's anti-entropy product check is not an acceptance row for this six-row capture leaf; it is inherited whole-Story evidence. `TestDurableSyncGeneratedPerturbationProduct` drives `DurableIndex.SyncFrom` over the generated finite product and has an independent reference oracle. Axes/ranges and unbounded bounds are recorded in `TASK-260830-2xt6fd_axis-inventory-rev7.md`. The same-digest/different-bytes refusal is separately driven by `TestDurableSyncSameDigestDifferentBytesIsTypedConflict`.

| Mutant | What it narrows | Named test failure | Evidence |
| --- | --- | --- | --- |
| `union-lease-created-at-winner` | Lets timestamp decide an equal-epoch lease winner | `TestDurableSyncClockSkewDoesNotSelectLeaseWinner` | KILLED twice alone |
| `sync-arrival-order-truncates-fetch` | Drops a valid identity under one arrival order | `TestDurableSyncArrivalOrderConverges` | KILLED twice alone |
| `sync-partial-peer-regresses-local-root` | Admits root regression from a proper peer subset | `TestDurableSyncPartialPeerNeverRegressesLocalRoots` | KILLED twice alone |
| `sync-repeat-record-quarantine-write` | Rewrites a persisted conflict record on repeated sync | `TestDurableSyncGeneratedPerturbationProduct` conflict subtest | KILLED twice alone |
| `sync-common-audit-partial-overlap-peer` | Skips common-object audit for a partial-overlap peer | `TestDurableSyncConflictWithPartialOverlapPeer` | KILLED twice alone |
| `sync-duplicate-delivery-drops-active-object` | Removes an active object on an identical byte retry | `TestDurableJSONAddIsIdempotentAcrossCrashRestartByNamespace` | KILLED three times alone. The initial `TestDurableSyncDuplicateDeliveryIsByteIdentical` selector survived because its snapshot begins after initial delivery; that result is retained as an explicit test bound. |

Raw JSON event logs and applied source overlays are in `sync-property-mutants-rev7-pass-01/`, `sync-property-mutants-rev7-pass-02/`, and `rev7-duplicate-drop-mutant/`.
