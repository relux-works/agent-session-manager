# TASK-260830-2g5be6 Conformance Matrix

Authority: `internal/specdoc/SPEC.v0.7.0.md`, Sections 1.6, 7.8,
10.2, 13.14.1 (§7.8, §10.2 and §13.14.1 verified byte-identical to
v0.6.0). Package: `internal/clonesnap`. Normative spec text is
quoted from the pinned file; the prose here maps, never restates.
Every artifact below is produced through the landed `clonebundle`
builders named in the call-site column; the
`internal/clonesnap/TRACEABILITY.md` clause map carries the same
rows with the landed-owner reuse table.

## AC coverage: 17 of 17 rows driven and measured

Every row is driven through the named production entry point by the
named committed test, and every row's gate carries at least one
narrowing the shipped battery reddens 2/2 (41 KILLED narrowings +
2 KILLED add-writes + 1 SURVIVED control, battery run twice).
Ratio: **17 of 17 AC rows driven and measured at the member
level**. The per-row killer mapping is in
`TASK-260830-2g5be6_results.md` (rev3 finding table + coverage
table); rows 8 and 10 are the two rev2 gaps this round closed,
rows 2, 3, 9, 11, 15 each gained a sub-class narrowing.

| # | AC row | Production call site | Named test |
|---|---|---|---|
| 1 | Capture walk yields sorted unique raw entries from real bytes: sanitized key, class, byte count, blob ID, descriptor ID | `Capture` → `walkAndMeasure`, `BuildRawObjectManifest` | `TestCaptureSealsBothManifestsFromStoreBytes`, `TestCaptureIsByteIdenticalAcrossRuns` |
| 2 | Containment: symlinked intermediate, trailing symlink, FIFO, directory, symlinked store root refuse before any payload open with the member named; capture never escapes the store root | `openMember` via `secprim.Guard`, `blockedAncestor`, `intermediatePrefix`, store-root handle via `secprim.OpenNoFollowDir` | `TestCaptureRefusesSymlinkedIntermediate/OptionalSymlinkedIntermediate/TrailingSymlink/FIFO/OptionalFIFO/DirectoryMember/ParentPlanKey/SymlinkedStoreRoot` |
| 3 | Nine-class classification; included carries descriptor + count and no reason, excluded carries null content + stable reason; one item per plan candidate; unplanned regular and special members refuse | `checkPlan` via `ValidCaptureClass`, `sealAndPublish` | `TestCaptureSealsBothManifestsFromStoreBytes`, `TestCaptureOptionalAbsentSealsExcluded`, `TestCaptureRefusesMysteryPlanClass/UnplannedMember/UnplannedSpecialMember/PrefixSibling/RequiredMissing/DuplicatePlanKey` |
| 4 | credential/machine_auth/runtime_state/transient_lock always excluded; bytes never reach a blob, a manifest, a log line, or an error string | `excludedClass` (never opened) | `TestCaptureExcludesSecrets` |
| 5 | unknown drives raw_complete=false and blocks maximal_safe through the projection entry | `sealAndPublish`, `AdmitForTarget` via `RefuseMaximalSafeUnlessComplete` | `TestCaptureUnknownMakesRawIncomplete` (included), `TestCaptureOptionalAbsentUnknownMakesRawIncomplete` (excluded, P3-ε), `TestProjectMaximalSafeRequiresComplete/MaximalSafeAdmitsComplete/StrictExactAdmitsUnknown` |
| 6 | Stable boundary from real evidence: closed proof kind, generation, equal pre/post digests, idle facts | `sealAndPublish` via `buildBoundary` (landed) | `TestCaptureSealsBothManifestsFromStoreBytes`, `TestCaptureImmutableSnapshotProof`, `TestCaptureRefusesUnknownProofKind/IdleCouplingViolation/SnapshotIdentityCoupling/EmptyGeneration` |
| 7 | Size equality is never proof: equal-size differing bytes refuse stable | `checkSourceRace` via `CheckDigestsEqual` | `TestCaptureRaceChangedByte` |
| 8 | unstable_archive carries generation, pre/post digests, source_not_quiescent, explicit ack, target_projection_forbidden; core-created, archive-only, never enters a target branch (projection entry) | `raceOutcome`, `sealAndPublish` (`Core: true`), `AdmitForTarget` via `RefuseUnstableForTarget` | `TestCaptureArchiveSealsUnstable` (sealed pre/post == measured pre/post, sealed generation == request generation), `TestCaptureArchiveRequiresExplicit`, `TestProjectRefusesUnstable` (refusal + target sink empty), `TestProjectRefusesUnstableDirect` |
| 9 | Race check: changed byte, changed size, excluded size change, appended record, replaced file, unreadable at post each refuse or seal unstable_archive, never silently stable | `walkAndMeasure`, `measurePost`, `gateSourceRace` | `TestCaptureRaceChangedByte/ChangedSize/ExcludedSizeChange/AppendedRecord/ReplacedFile/RemovedFile/UnreadableAtPost`, `TestCaptureArchiveSealsUnstable` |
| 10 | Mutation check runs before any seal or projection output | `Capture`, `CaptureAndProject` (ORDER-PIN) | `TestCaptureRaceSealsNoManifest`, `TestCaptureRaceEmitsNoReceipt` (race refusal + target sink empty + capture store holds no sealed manifest) |
| 11 | WorkspaceBinding from real state: normalized cwd, sorted unique fingerprints[0..128], branch, head/index/tree digests | `CheckpointWorkspace` via `DecodeWorkspaceBinding` | `TestCheckpointWorkspaceSealsObservedState/RelativeCwd/RootCwd/UnsortedFingerprints/DuplicateFingerprint/TooManyFingerprints/BadDigest/BadWorkspaceID`, `TestCaptureSealsBothManifestsFromStoreBytes` |
| 12 | No member grants filesystem authority: escapes, absolute paths, dot segments, unobserved cwds refuse | `workspaceCwdRelative` | `TestCheckpointWorkspaceRefusesEscape/DotSegments/MissingCwd/FileCwd/SymlinkRoot` |
| 13 | Source basis bound: ax_session IDs or sanitized external_native reference; SourceIdentity re-validated at the entry | `Capture` via `buildSourceBasis` + `IdentityDigest` (landed) | `TestCaptureExternalNativeBasis`, `TestCaptureRefusesMixedSourceBasis/UnknownBasisKind/UnsanitizedExternalRef/InvalidSourceIdentity` |
| 14 | Blob install fsyncs, verifies, atomically installs; every referenced blob present after Capture and after the crash replay; replayed capture publishes no half-written manifest | `InstallRawBlob`, `PutBlob` (landed) | `TestCaptureInstallsEveryPayloadBlob`, crash replay assertion in `TestCaptureCrashChildSelfTerminates`, `TestProjectIsIdempotent`, `TestCaptureHookFires` |
| 15 | Descriptor chunks sorted, contiguous, non-overlapping, from zero, covering size; empty blob has no chunks; past-128 GiB fails with capability_unavailable before a partial manifest | `BuildBlobDescriptor` via `CalculateObjectIdentity` + `VerifyDescriptorAgreement` | `TestCaptureEmptyMemberSealsEmptyBlob/MultiChunkMember/DescriptorAgreementEndToEnd/RefusesOversizedMember/RefusesSparseOversizedMember`, `TestDefaultMaxSingleBytesIs128GiB` |
| 16 | Logs and errors record digest, size, media type only | `CaptureLog` (no byte parameter) | `TestCaptureLogRecordsDigestsOnly`, `TestCaptureExcludesSecrets` (log + error scan) |
| 17 | Fidelity profiles strict_exact\|maximal_safe\|compact\|messages_only; stable captures admit with a sealed receipt, unknown profiles refuse | `AdmitForTarget`, `BuildAdmissionReceipt` | `TestProjectAdmitsStableEveryProfile`, `TestProjectRefusesUnknownProfile` |

## Negative/refusal evidence

Every refusal above fails when the gate admits what it must reject:
the 41 narrowing + 2 add-write rows in
`internal/clonesnap/testdata/mutant_harness.py` weaken each gate to
admit exactly one forbidden member (narrowings) or add exactly one
forbidden write (add-writes: `N-archive-emits-receipt`,
`N-excluded-orphan-blob`) and the named killer fails (43 KILLED, 1
harmless SURVIVED control, battery run twice). The token-preserving
row `N-plan-prefix` keeps comparing against the planned keys while
changing exact match to prefix match, and its killer drives the full
`Capture` entry. The cross-package row
`N-clonebundle-unknown-excluded` plants in clonebundle while its
killer lives here. Per-plant raw logs with subprocess exits are in
the producer evidence tar (`mutants/run1/`, `mutants/run2/`).
`N-race-order-project` is killed by the seal error message, not by
a sink assertion; the seal half at the projection entry is pinned
by `N-race-order-project-seal` (capture-store scan). The branch
`string[1..1024]` bound is measured at the landed
`DecodeWorkspaceBinding` (first-hand `maximum+1` survival probe in
`attribution/branch-bound-landed-decoder.log`); the local check is
defence in depth.

## Landed-owner reuse (not new behavior, cited not counted)

| Scope | Owner | Evidence |
|---|---|---|
| Capture contracts (manifests, items, basis, boundary, identities, generations) | `internal/clonebundle` (`Build*`, `Decode*`, `Refuse*`, `SanitizeNativeKey`, `CheckDigestsEqual`, `ValidCaptureClass`) | predecessor suite green in this run; every artifact here produced through those builders |
| Section 7.8 operation frames | `internal/sessadapter` | landed suite green in this run; this package consumes plan keys/classes, not adapter frames |
| Section 10.2 crash/idempotency discipline (fsync, atomic no-replace, quarantine) | `internal/localstore` | landed fault suites green in this run; delegation proven by the crash test + `TestProjectIsIdempotent` |
| Containment walk (openat-relative, symlink/FIFO refusal) | `internal/secprim` (`Guard.Open`, `CheckMemberPath`, `OpenNoFollowDir`) | landed suites green in this run; every payload open here goes through `Guard.Open` |
| Scalar gates + character string measure | `internal/environ`, `internal/scalar` | `TestCheckpointWorkspaceMultibyteBranch`, owner batteries green in this run |

## Stated bounds (sibling scope, not waived)

- No `ax` CLI, no provider process, no network: the store is a
  fixture directory. Section 7.8 operation wiring stays
  `sessadapter`-owned.
- The sealed stable proof carries the two measured source digests
  (pre and post); the race gate proved them equal before the seal
  ran. The ordering mutants prove the gate precedes the seal:
  `N-race-order-capture` (published-manifest scan),
  `N-race-order-project` (seal error message),
  `N-race-order-project-seal` (capture-store scan at the
  projection entry).
- Excluded-byte equal-size changes are below the detection floor
  (excluded bytes are never opened by design); presence, shape,
  and size changes still race (size pinned by
  `TestCaptureRaceExcludedSizeChange` + `N-excluded-size`).
- Sub-walk transient swaps restored before the post measurement
  are below the floor (no filesystem snapshot); persistent changes
  race and install-time verification still guards the read bytes.
- Optional included-class candidates absent from the store seal as
  excluded rows (`plan_optional_absent`); required ones refuse.
- `MaxSingleBytes` mirrors the Section 7.8 `ResourceLimits`
  per-object bound; zero selects 137438953472.
- The admission receipt proves admission and ordering only;
  projection planning past admission is the final leaf's.
- The four-profile fidelity vocabulary is spelled twice
  (`sessadapter` table + this package's switch, P3-ζ′); the Final
  leaf unifies or exports one owner.
- `EncodeNativeIdentity` standalone trusts the caller for bounds,
  vocabulary, and the workspace ID (P3-η remainder); this leaf's
  seal path re-decodes at the entry. The standalone-entry contract
  belongs to the clonebundle owner or the Final leaf.
- The lone-surrogate member at the shared text gate (P3-ζ) is
  deferred to the Final leaf (no text admission site here).
- `internal/traceability` bindings and re-pin (Story FINAL leaf).
