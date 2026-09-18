# Native Capture Clause Coverage (TASK-260830-2g5be6)

Owner: `internal/clonesnap`. Authority:
`internal/specdoc/SPEC.v0.7.0.md`, Sections 7.8, 10.2, 13.14.1
(the pinned sections are byte-identical to v0.6.0).
The Story's FINAL leaf carries the `internal/traceability` registry
bindings and re-pin; this file is the second-leaf clause record.

## Why this package captures instead of clonebundle

The predecessor leaf (`internal/clonebundle`) owns the capture
contracts: the closed manifest/boundary/blob/identity shapes, their
builders and decoders. This package owns the capture OPERATION over
a real on-disk provider store: the contained walk, item
classification, boundary construction from measured evidence, the
source-race check, the workspace checkpoint, and target admission.
Every artifact emitted here is produced through the landed
clonebundle builders (`BuildRawObjectManifest`,
`BuildCaptureManifest`, `SanitizeNativeKey`, `CheckDigestsEqual`,
`RefuseUnstableForTarget`, `RefuseMaximalSafeUnlessComplete`,
`InstallRawBlob`, `VerifyDescriptorAgreement`,
`DecodeWorkspaceBinding`, `IdentityDigest`) and re-decoded by the
landed decoders before it is published or returned; no second
manifest, boundary, blob, or identity model exists here.

## Landed-owner reuse

| Scope | Owner | Use |
|---|---|---|
| Manifest/boundary/item/identity shapes | `internal/clonebundle` | all Build/Decode/Refuse entries above |
| Store containment (openat-relative Guard walk) | `internal/secprim` | every payload open via `Guard.Open` from a verified root handle; member grammar via `CheckMemberPath` |
| Durable writes (no-replace, fsync, verify) | `internal/localstore` | payload blobs via `InstallRawBlob`, manifests and receipts via `ObjectStore.PutBlob` |
| Blob Descriptor shape and identity | `internal/canonicaljson` | `CalculateObjectIdentity` plus local claim comparisons; never the attesting entry |
| Capture class vocabulary | `internal/clonebundle` | `ValidCaptureClass` (mirrors the `sessadapter` plan vocabulary) |
| String measure, digests, UUIDv7, paths | `internal/environ`, `internal/scalar` | `StringLength`, `ParseDigest`, `ParseUUIDv7`, `ParseRelativePath`, `ParseAbsolutePath` |
| Canonical bytes | `internal/canonicaljson` | `Canonicalize` for the workspace binding and the admission receipt |

## Clause map

| Pinned clause | Production entry | Test |
|---|---|---|
| 13.14.1 Raw Manifest from real bytes: sorted unique entries, sanitized key, class, byte count, blob ID, descriptor ID | `Capture`, `BuildRawObjectManifest` (landed) | `TestCaptureSealsBothManifestsFromStoreBytes`, `TestCaptureIsByteIdenticalAcrossRuns` |
| 13.14.1 Capture Manifest from real evidence: basis, boundary, items, excluded classes, derived raw_complete | `Capture`, `BuildCaptureManifest` (landed) | `TestCaptureSealsBothManifestsFromStoreBytes`, `TestCaptureOptionalAbsentSealsExcluded` |
| 13.14.1 CaptureItem 9-class vocabulary + included/excluded content rule | `checkPlan` via `ValidCaptureClass`, `sealAndPublish` | `TestCaptureSealsBothManifestsFromStoreBytes`, `TestCaptureRefusesMysteryPlanClass` |
| 13.14.1 always-excluded classes never included; bytes never reach blob/manifest/log/error | `excludedClass` (never opened) | `TestCaptureExcludesSecrets` (one fixture member per class, full-artifact scan) |
| 13.14.1 unknown makes raw_complete false and blocks maximal_safe | `sealAndPublish`, `AdmitForTarget` via `RefuseMaximalSafeUnlessComplete` | `TestCaptureUnknownMakesRawIncomplete` (included), `TestCaptureOptionalAbsentUnknownMakesRawIncomplete` (excluded, P3-ε), `TestProjectMaximalSafeRequiresComplete`, `TestProjectMaximalSafeAdmitsComplete`, `TestProjectStrictExactAdmitsUnknown` |
| 13.14.1 stable boundary: closed proof kind, generation, equal pre/post digests, idle facts | `sealAndPublish` via `buildBoundary` (landed) | `TestCaptureSealsBothManifestsFromStoreBytes`, `TestCaptureImmutableSnapshotProof` |
| 13.14.1 size equality is never proof | `checkSourceRace` via `CheckDigestsEqual` | `TestCaptureRaceChangedByte` (equal-size differing bytes refuse) |
| 13.14.1 unstable_archive: generation, pre/post, source_not_quiescent, explicit ack, target forbidden; core-created, archive-only | `raceOutcome`, `sealAndPublish` (`Core: true`) | `TestCaptureArchiveSealsUnstable` (literal kind/reason/ack/forbidden assertions + sealed pre/post digests equal the measured digests + sealed generation equals the request generation; `N-unstable-seals-pre-as-post` KILLED) |
| 13.14.1 G2 rejects unstable | `AdmitForTarget` via `RefuseUnstableForTarget` | `TestProjectRefusesUnstable` (refusal + target sink empty), `TestProjectRefusesUnstableDirect` |
| Source race detected before projection: changed byte/size, appended record, replaced file, unreadable at post, excluded size change | `walkAndMeasure`, `measurePost`, `gateSourceRace` | `TestCaptureRaceChangedByte/ChangedSize/AppendedRecord/ReplacedFile/RemovedFile/UnreadableAtPost/ExcludedSizeChange` (excluded row pins the "size changes still race" claim; `N-excluded-size` KILLED) |
| Race gate precedes any seal or projection output | `Capture`, `CaptureAndProject` (ORDER-PIN) | `TestCaptureRaceSealsNoManifest` (no manifest published), `TestCaptureRaceEmitsNoReceipt` (race refusal, not a seal error, + target sink empty + capture store holds no sealed manifest; `N-race-order-project` killed by the seal error message, `N-race-order-project-seal` killed by the capture-store scan) |
| 13.14.1 WorkspaceBinding from real state: normalized cwd, sorted unique fingerprints[0..128], branch, head/index/tree | `CheckpointWorkspace` via `DecodeWorkspaceBinding` (landed) | `TestCheckpointWorkspaceSealsObservedState`, `...RelativeCwd`, `...RootCwd`, refusal rows |
| 13.14.1 no member grants filesystem authority | `workspaceCwdRelative` | `TestCheckpointWorkspaceRefusesEscape/DotSegments/MissingCwd/FileCwd/SymlinkRoot` |
| 13.14.1 source basis ax_session / external_native | `Capture` via `buildSourceBasis` (landed) | `TestCaptureExternalNativeBasis`, `TestCaptureRefusesMixedSourceBasis/UnknownBasisKind/UnsanitizedExternalRef` |
| SourceIdentity re-validated at the entry (P3-η, this leaf) | `Capture` via `IdentityDigest` (landed, re-decodes) | `TestCaptureRefusesInvalidSourceIdentity` (overlong ID, bogus kind, zero workspace, bad UTF-8, absolute ref; no manifest published) |
| 10.2 fsync + verify + atomic install before publishing a referencing record | `InstallRawBlob`, `PutBlob` (landed) | `TestCaptureInstallsEveryPayloadBlob` (every entry's blob present), crash replay assertion, `TestCaptureSealsBothManifestsFromStoreBytes`, `TestProjectIsIdempotent` |
| 10.2 logs record digest, size, media type only | `CaptureLog` (no byte parameter) | `TestCaptureLogRecordsDigestsOnly`, `TestCaptureExcludesSecrets` (log scan) |
| 10.2 Blob Descriptor: sorted contiguous non-overlapping chunks from zero covering size; index from zero by one; empty = size zero, no chunks | `BuildBlobDescriptor` via `CalculateObjectIdentity` + `VerifyDescriptorAgreement` | `TestCaptureEmptyMemberSealsEmptyBlob`, `TestCaptureMultiChunkMember`, `TestCaptureDescriptorAgreementEndToEnd` |
| 10.2 oversized blob fails with capability_unavailable before a partial manifest | `openMember` limit gate | `TestCaptureRefusesOversizedMember`, `TestCaptureRefusesSparseOversizedMember`, `TestDefaultMaxSingleBytesIs128GiB` |
| 7.8 fidelity profiles strict_exact\|maximal_safe\|compact\|messages_only | `AdmitForTarget`, `BuildAdmissionReceipt` | `TestProjectAdmitsStableEveryProfile`, `TestProjectRefusesUnknownProfile` |
| 1.6 string measure in characters | measured gate: landed `DecodeWorkspaceBinding`; local `workspaceNullableText` via `environ.StringLength` is defence in depth | `TestCheckpointWorkspaceMultibyteBranch` (1024 two-byte characters seal, 1025 refuse with `string[1..1024]`; the `maximum+1` narrowing survives because the landed decoder refuses first — reviewer plant `R2-branch-1025` SURVIVED 2/2) |
| Containment: symlinked intermediate and FIFO refuse before any payload open, member named; no escape past the root | `openMember` via `secprim.Guard`, store-root handle via `secprim.OpenNoFollowDir` | `TestCaptureRefusesSymlinkedIntermediate/OptionalSymlinkedIntermediate/TrailingSymlink/FIFO/OptionalFIFO/DirectoryMember/ParentPlanKey/SymlinkedStoreRoot` (`N-store-root-symlink`, `N-required-blocked-ancestor` KILLED) |
| Plan member-name grammar: no escape past the root | `checkPlan` via `secprim.CheckMemberPath` (the only gate: `SanitizeNativeKey` admits parent segments) | `TestCaptureRefusesParentPlanKey` (`N-plan-grammar-dotdot` KILLED) |
| Workspace fingerprints at most 128 | `workspaceFingerprints` | `TestCheckpointWorkspaceRefusesTooManyFingerprints` (killed by the local `maximum is 128` literal; `N-fingerprints-129` KILLED) |
| First chunk of a multi-chunk blob within uint53[1..4194304] | `BuildBlobDescriptor` via `VerifyDescriptorAgreement` (landed) | `TestCaptureMultiChunkMember` (`N-first-chunk-oversize` KILLED) |
| Unplanned members refuse; one item per plan candidate | `planMemberClass` | `TestCaptureRefusesUnplannedMember/PrefixSibling/RequiredMissing/DuplicatePlanKey/UnplannedSpecialMember` (unplanned symlink refuses `not a plan candidate` with no outside byte installed; `N-unplanned-special` KILLED) |

## Durability

This package performs durable writes only through
`localstore.ObjectStore.PutBlob`: payload blobs (via the landed
`InstallRawBlob`), manifest blobs, and admission receipt blobs. The
landed discipline (stage, fsync, size/digest verification, atomic
no-replace install, directory fsync, quarantine on mismatch) carries
the crash semantics; this package adds the publish ORDER (payloads,
then raw, then capture, then receipt) and the hooks that pin it.
`TestCaptureCrashChildSelfTerminates` kills a live capture between
the raw and capture publishes and proves the replay converges with
byte-identical manifests; `TestProjectIsIdempotent` proves a
replayed capture verifies and reuses every blob and publishes no
half-written manifest (every manifest blob re-decodes). No second
write path exists in this package.

## Stated bounds (sibling scope, not waived)

- No `ax` CLI, no provider process, no network: the store is a
  fixture directory. Section 7.8 capture-plan/capture/normalize
  operation wiring stays owned by `sessadapter`; this package
  consumes plan keys and classes, it does not speak the adapter
  protocol.
- The sealed stable proof carries the two measured source digests
  (pre and post): the race gate proved them equal before the seal
  ran, so on the stable path the sealed bytes equal either
  measurement, and the landed builder refuses unequal digests.
  The ordering mutants move the gate after the seal:
  `N-race-order-capture` is killed by the published manifests,
  `N-race-order-project` by the seal error message (not by a sink
  assertion), and `N-race-order-project-seal` — which preserves
  the race error while sealing first — by the sealed raw manifest
  in the capture store. Together they prove the gate precedes the
  seal at both entries.
- Excluded-class bytes are never opened, so a changed byte at
  equal size inside an excluded member is below the detection
  floor: presence, shape, and size changes still race, pinned by
  `TestCaptureRaceExcludedSizeChange` (`N-excluded-size` KILLED).
  Excluded bytes cannot affect capture output, and the never-open
  rule is what the leak suite proves.
- Enumeration observes names through path strings; every OPEN
  descends from the verified root handle through the Guard. A
  sub-walk transient swap restored before the post measurement is
  below the detection floor (no filesystem snapshot); any
  persistent change races, and blob integrity of the read bytes is
  guaranteed by the install-time verification regardless.
- Optional included-class candidates absent from the store seal as
  excluded rows with reason `plan_optional_absent`; required ones
  refuse. Required-ness for excluded classes is inert (presence is
  irrelevant to bytes that are never opened).
- `MaxSingleBytes` mirrors the Section 7.8 `ResourceLimits`
  per-object bound; zero selects the Section 10.2 default of
  137438953472 (32768 chunks of 4194304 bytes). The sparse
  oversized row pins the default end to end without reading 128
  GiB.
- The admission receipt (`urn:ax:internal:clonesnap-admission-receipt`
  1.0.0) proves target admission and race-before-projection
  ordering only. Projection planning past admission, per-kind
  event payload registries, and `strict_exact`/`compact`/
  `messages_only` planning semantics are the final leaf's and the
  normalization sibling's.
- `internal/traceability` registry bindings and digest re-pin
  (Story FINAL leaf).
- The four-profile fidelity vocabulary is spelled twice (P3-ζ′)
  and the Final leaf does not unify the spellings:
  `sessadapter` owns an unexported table (`operations.go`
  `fidelityProfiles`, pinned by `TestValueVocabulariesMatchSpec`)
  and this package owns a closed switch (pinned by
  `TestProjectAdmitsStableEveryProfile` and
  `TestProjectRefusesUnknownProfile`); unification is deferred to
  a follow-up on `internal/sessadapter` and `internal/clonesnap`.
  The four-class exclusion table is spelled twice the same way
  (`clonebundle.alwaysExcludedClass`, `clonesnap.excludedClass`).
- `EncodeNativeIdentity` standalone trusts the caller for bounds,
  vocabulary, and the workspace ID (P3-η remainder): this leaf's
  seal path re-decodes through the landed callers
  (`sealAndPublish` calls `IdentityDigest` directly and
  `BuildCaptureManifest` re-decodes the source identity
  internally), pinned by `TestCaptureRefusesInvalidSourceIdentity`.
  The standalone-entry contract (re-decode inside the entry or an
  explicit pre-validated-identity contract) is deferred to a
  follow-up on the `internal/clonebundle` owner: the Final leaf
  records the bound and changes no `EncodeNativeIdentity` behavior.
- The lone-surrogate class at JSON text (P3-ζ) is closed at the
  projection entry, not here: fixture-native lines and bodies
  decode through `environ.DecodeStrictObject` at `cloneproject`
  `strictMembers`, which refuses lone surrogate escapes before any
  member is read (rows `lone_surrogate_text` and
  `lone_surrogate_envelope`; narrowings `N-strict-text-surrogate`
  and `N-strict-envelope-surrogate`). This package still owns no
  text admission site.
- The race window is measurable only through the `AfterWalk` seam
  (P3-e bound): the mutation window is exercised only through
  `Hooks.AfterWalk`, and `measurePost` runs unconditionally by
  inspection — no deterministic seam-free injection exists.
