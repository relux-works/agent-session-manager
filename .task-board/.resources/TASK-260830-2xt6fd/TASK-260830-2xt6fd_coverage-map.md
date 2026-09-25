# Coverage map — TASK-260830-2xt6fd

## Brief gap

The assignment supplied no surface table. This is a brief gap, not a silent
coverage waiver. The map below covers every row in the six-row acceptance
matrix in `internal/gitsnap/testdata/assembly-acceptance.md`.

| Acceptance row | Production call site | Named test(s) | Narrowing evidence |
| --- | --- | --- | --- |
| Held input/provider quiescence | `(*tmuxserver.Lifecycle).CaptureGitWorkspace` | `TestCaptureStopPointStateDomain`, `TestCaptureProviderProofRequirements`, `TestCaptureRejectsCheckpointBoundaryAsProviderProof`, `TestCaptureFinalStatusStateDomain`, `TestCaptureNeverTransitionsOutsideStopPointOwnerSet` | `N-capture-admits-parked` kills `TestCaptureRejectsParkedEvenWithPriorQuiescenceReceipt` alone; `N-capture-restores-with-boundary-token-preserved` kills the successful-entry and AST transition tests. |
| Source change during capture and retry | `(*tmuxserver.Lifecycle).CaptureGitWorkspace` → `gitsnap.AssembleProvisional`; `gitsnap.Capture` | `TestCaptureGitWorkspaceSourceChangeRefusalAndRetry`, `TestAssemblySourceChangesAndRetry`, `TestContentDigestRaceAndRetry` | `N-assembly-source-admits-working-file-change` kills `TestAssemblySourceChangesAndRetry`; `N-consistency-admits-five-byte-digest-change` kills `TestContentDigestRaceAndRetry`. |
| Missing socket versus missing parent | `tmuxserver.UnixDialer.Dial` | `TestUnixDialerMissingSocketIsStale`, `TestUnixDialerMissingDirectoryIsUnknown` | `N-unix-dialer-missing-parent-admits-stale` admits only ENOENT with an absent containing directory as stale; `TestUnixDialerMissingDirectoryIsUnknown` kills it alone. |
| Repository-local packs, exact inventories, raw/logical indexes | `gitsnap.AssembleProvisional` → `captureAssembly` / `assemblePack` | `TestAssemblyPackIndexAndBytes`, `TestAssemblyRecursivePacks`, `TestAssemblyIndexCompatibility`, `TestAssemblyRawIndexRefusals`, `TestAssemblyFailedReadsAndCorruptObjects`, `TestAssemblyIsolatedDatabaseIgnoresAmbientObjects` | `N-assembly-inventory-admits-four-object-metadata-mismatch`, `N-assembly-raw-index-admits-two-entry-disagreement`, and `N-assembly-checksum-admits-last-bit-flip` each kill their named test. |
| Workspace root/child, blobs, cwd/config, safe-path closure | `gitsnap.AssembleProvisional` → `ValidateProvisional` | `TestValidateProvisionalClosureAndDescriptors`, `TestAssemblyGroupRecordAgreement`, `TestAssemblyMissingConfigClosure`, `TestAssemblyRecursiveCWDAndConfigClosure`, `TestValidateProvisionalParentChildOverlap` | `N-assembly-descriptor-admits-seven-byte-entry`, `N-assembly-group-admits-repository-identity-mismatch`, `N-assembly-closure-admits-one-unbound-tree`, `N-assembly-path-admits-missing-config-token-preserved`, and `N-assembly-path-admits-one-parent-partition-entry` each kill their named test. `N-assembly-closure-admits-one-missing-child` survives because the same fixture is rejected by another parent/tree binding; it is not counted as independent evidence. |
| Actual bytes, recursive dirty state, truthful unsupported inputs | `gitsnap.AssembleProvisional` → `gitsnap.Capture` with `ContentOptions` | `TestAssemblyNormativeCorpus`, `TestAssemblyRecursivePacks`, `TestAssemblyIndexCompatibility`, `TestAssemblyOptionsAndUnsupportedSources`, `TestContentSubmoduleUninitializedAndRefusals`, `TestContentSymlinkChainsAndExcludedTargets`, `TestContentSymlinkChainBoundaryAtProductionCapture` | `N-path-admits-escaping-chain`, `N-submodule-admits-stray-uninitialized`, `N-assembly-features-admits-origin-promisor`, and `N-assembly-policy-admits-unclassified-extra` each kill their named test. |
| Crash, idempotency, non-publication | `(*tmuxserver.Lifecycle).CaptureGitWorkspace` → `gitsnap.AssembleProvisional` | `TestCaptureFailureRetryUsesRequestStop`, `TestCaptureProcessCrashRetry`, `TestAssemblyCrashAfterObjectsAndRetry`, `TestAssemblyStoreFailureAndRetry` | `N-assembly-install-admits-shard-failure` kills `TestAssemblyStoreFailureAndRetry`; the stop-point narrowing above covers refusal. Crash exits 74/75 are expected child exits, followed by same-store retry. |

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
| `N-capture-admits-parked` | A parked Terminal Instance passes the stop-point admission gate | `TestCaptureRejectsParkedEvenWithPriorQuiescenceReceipt` alone — killed; the test asserts literal `capability_unavailable`. |
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
`mutants-control-01/`, plus the final reruns in `mutants-capture-final-01/`,
`mutants-capture-final-02/`, and `mutants-unix-dialer-final/`. Four earlier broad harness controls that attempted the
whole package suite exceeded the harness's 240-second per-test limit; they are
diagnostics, not pass evidence. Their recorded paths are the four
`mutants-*-run-01.log` files for content-a/content-b/assembly-a/assembly-b.
