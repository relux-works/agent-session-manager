# Provisional Git transfer assembly — TASK-260830-2xt6fd

Authority: AX v0.7.0, commit `32b3f2ba7c377248a53cd42389abbd2f1c321834`,
Sections 10.4 and 12.1–12.3. The v0.5.0 clauses originally named by this Story
were byte-identical after spec adoption; the active traceability registry now
cites the v0.7.0 line inventory. `TestAssemblyNormativeCorpus` consumes the
exact pinned payload corpus through the same fixture loader as the content tests.

`(*Lifecycle).CaptureGitWorkspace` is the production composition entry. It is a
stop-point operation: it admits only same-incarnation `quiescing` with input
closure and provider-boundary receipts, or `stopped`. It does not reopen input.
A crash left in `quiescing` uses the landed owner's `request-stop` or stale
termination path before retry. All six original final-leaf rows have production
entry drivers: **6 of 6 AC rows are driven**. This ratio covers capture and
transfer-object assembly; it does not claim materialization, checkpoint
publication, doctor capability, or Story delivery.

| Original final-leaf AC row | Production call site and named tests | Measured result and bound |
| --- | --- | --- |
| Held input/provider quiescence through capture | `(*Lifecycle).CaptureGitWorkspace`; `TestCaptureGitWorkspaceQuiescedAssemblesAndRechecks`, `TestCaptureGitWorkspaceStoppedAssemblesWithoutBoundary`, `TestCaptureStopPointStateDomain`, `TestCaptureAdmissionGateValidControls`, `TestCaptureProviderProofRequirements`, `TestCaptureRejectsCheckpointBoundaryAsProviderProof`, `TestCaptureRejectsQuiesceReceiptFromAnotherIncarnation`, `TestCaptureRejectsBarrierLostAfterAssembly`, `TestCaptureStoppedMustRemainStoppedAtClose`, `TestCaptureFinalStatusStateDomain`, `TestCaptureNeverTransitionsOutsideStopPointOwnerSet`, `TestCaptureCoordinatorAdmissionGateCensus` | Same-incarnation input closure and provider boundary are required in `quiescing`; `stopped` needs no boundary. The stop-point state table includes exactly those two admitted rows and literal refusal rows for unsupported observed states. The status owner does not persist transient `creating` (it projects it to `unavailable`), and capture refuses that observed `unavailable` state without reclassifying it. No release operation is added. |
| Source-change refusal through the capture interval | `CaptureGitWorkspace -> AssembleProvisional -> Capture`; `TestCaptureGitWorkspaceSourceChangeRefusalAndRetry`, `TestAssemblySourceChangesAndRetry` | A file changed after pack assembly refuses; source-restored retry succeeds. Reads cannot detect a fully reverted change between observations; held quiescence protects the interval. |
| Repository-local packs, exact inventories, raw/logical indexes | `AssembleProvisional -> captureAssembly/assemblePack -> RunInput/RunIsolated`; `TestAssemblyNormativeCorpus`, `TestAssemblyPackIndexAndBytes`, `TestAssemblyRecursivePacks`, `TestAssemblyFailedReadsAndCorruptObjects`, `TestAssemblyIndexCompatibility`, `TestAssemblyRawIndexRefusals` | Generated packs are independently imported and compared to exact inventories; raw/logical indexes match the pinned corpus; isolated verification cannot borrow ambient object databases. |
| Workspace root/child, blob, cwd/config, safe path closure | `AssembleProvisional -> repository/seal -> ValidateProvisional/checkGroupRecord/checkCapturedPaths`; `TestValidateProvisionalClosureAndDescriptors`, `TestAssemblyGroupRecordAgreement`, `TestAssemblyMissingConfigClosure`, `TestAssemblyMultipleMembers`, `TestAssemblyRecursiveCWDAndConfigClosure`, `TestValidateProvisionalParentChildOverlap` | Canonical identities/descriptors delegate to `canonicaljson`; absent, forged and mismatched closure refuses. Group-history authority/conflict resolution belongs to the record owner. |
| Actual bytes, recursive dirty state and truthful unsupported cases | `CaptureGitWorkspace -> AssembleProvisional -> Capture` with `ContentOptions`; `TestAssemblyNormativeCorpus`, `TestAssemblyRecursivePacks`, `TestAssemblyIndexCompatibility`, `TestAssemblyOptionsAndUnsupportedSources`, `TestContentSubmodulePointerStates`, `TestContentSymlinkChainBoundaryAtProductionCapture` | Existing byte, exclusion, symlink and submodule owners are composed. Shallow, partial-clone, replacement-history and other unsupported repository sources refuse; no materialization is claimed. |
| Truthful capability, mutation/recovery, publication evidence | `CaptureGitWorkspace -> AssembleProvisional`; `TestCaptureFailureRetryUsesRequestStop`, `TestCaptureProcessCrashRetry`, `TestAssemblyCrashAfterObjectsAndRetry`, `TestAssemblyStoreFailureAndRetry`; scoped mutant logs | Crash exits 74/75 are expected child-process failures, followed by successful retry against the same durable instance/store. Immutable objects may remain unreferenced. No checkpoint publication, doctor capability, or external transfer is claimed. |

## Component boundaries

`AssembleProvisional` requires a schema-valid, identity-verified immutable Group
Record and all selected Git members in one immutable store outside the source
checkouts. Record resolution, ambiguous histories and session authority belong
to their assigned owners. The result returns exact manifests/descriptors and
separate byte-digest addresses; it is not an attestation or durable publication.

Each Git pack contains closure rooted in its recorded HEAD/upstream and all
non-gitlink index OIDs. Unrecorded branches and arbitrary unreachable objects are
not copied. Each initialized child is packed independently; three submodule
pointer states remain independent. Raw indexes are copied and decoded by Git in
a disposable repository. Split indexes are flattened there, preserving logical
entries, flags and version; unborn empty checkouts receive an equivalent valid
empty raw index. Unknown required extensions fail through Git. Working bytes
remain distinct from staged blobs. File contents reuse the existing chunked
immutable-install owner; packs and raw indexes currently use memory. No 128 GiB
pack throughput claim is made.

Up to 1024 repository trees are direct children of the workspace root. Larger
groups use empty per-workspace composites; `TestAssemblyManifestFanout` measures
1024/1025 tree construction, not 1025 live repositories. Their local paths are
interpreted by the Git member/submodule mappings; a child README is not a path
partition rooted at the parent's README. Cwd/config paths resolve through the
repository and initialized submodule closure. Parent entries may name submodule
directories but cannot overlap child-owned partitions; required configs must be
files, and ignored configs still need an explicit non-secret include policy.

`ValidateProvisional` verifies graph/schema/descriptor relationships. It does
not reread blob-store content, import externally supplied packs or prove runtime
quiescence. Assembly verifies/imports freshly generated bytes before passing
them through PutBlob. Arbitrary external transfer validation/materialization is
not represented by the graph-only validator.

The coordinator composes the landed lifecycle, Terminal Instance, and axpane
owners; it calls only status and wait-safe-boundary. It proves the current
incarnation's input closure and provider boundary before assembly, then reads
status again and refuses if the source left `quiescing` or `stopped`. A crash
never reopens input; recovery uses the lifecycle owner's existing stop or stale
termination path. Group-record history selection and conflict resolution remain
with the record owner. No user-facing CLI or doctor capture capability is
advertised because this internal stop-point entry is not a resumable capture
workflow.

The runner is trusted to execute the selected Git binary faithfully. Isolated
verification strips ambient Git routing/config variables; source reads preserve
the existing trusted configuration policy, disable lazy fetch/replacement
objects, and do not sandbox arbitrary filters. Native tests are not
cross-platform runtime proof. Repeated observations cannot detect a
change-and-revert between reads. A killed process can leave disposable verifier
directories and verified unreferenced objects. Retry performs fresh observation
and reuses immutable bytes.
