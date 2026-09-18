# Clone Bundle Clause Coverage (TASK-260830-24z2b3)

Owner: `internal/clonebundle`. Authority:
`internal/specdoc/SPEC.v0.7.0.md`, Sections 7.8, 10.2, 13.14.1
(the pinned sections are byte-identical to v0.6.0).
The Story's FINAL leaf carries the `internal/traceability` registry
bindings and re-pin; this file is the first-leaf clause record.

## Why this package validates instead of canonicaljson

The `canonicaljson` closed-shape registry keeps
`rejectUnsupportedImmutableObjectShape` rows for
`clone-raw-object-manifest`, `clone-capture-manifest`,
`canonical-session`, and `canonical-event` 1.0.0, and that table's
census pins every row. Replacing them would fork shape ownership, so
this package is the validating owner: it reuses
`canonicaljson.Canonicalize` (byte transform),
`canonicaljson.CalculateObjectIdentity` plus local claim comparisons
(Blob Descriptors; never the attesting entry, per the provhost
no-attestation bound),
`sessadapter.DecodeTuple` (Environment Tuples),
the `environ` scalar gate set (`CheckStringBounds`,
`CheckUint53Bounds`, `CheckDigest`, `CheckUUIDv7`,
`CheckTimestamp`, `CheckSortedUniqueStrings`,
`CheckSortedUniqueDigests`, `CheckExtensions`, `StringLength`,
`DecodeStrictObject`) through one-line delegating wrappers in the
`sessadapter` convention, keeping only package-specific rules
(sanitizer, exclusion, coupling, reconciliation, extension values),
`scalar` (digests, UUIDv7, timestamps, paths),
`hosttrust` (row-21 exclusion matchers), and
`localstore.ObjectStore.PutBlob` (blob install). No rule is
reimplemented where a landed owner exists.

## Clause map

| Pinned clause | Production entry | Test |
|---|---|---|
| 13.14.1 Raw Manifest closed envelope + self ID | `BuildRawObjectManifest`, `DecodeRawObjectManifest` | `TestBuildRawManifestRoundTrip`, `TestRawManifestRefusals/decode` |
| 13.14.1 RawObjectEntry class subset (no credential/auth/runtime/lock) | `buildRawEntries`, `decodeRawEntries` | `TestRawManifestRefusals/build` (5 class rows) |
| 13.14.1 sorted unique entries, total_bytes sum | `buildRawEntries`, `decodeRawEntries` | `TestRawManifestRefusals` (unsorted, duplicate at build + decode, total mismatch, not-an-array, 65536 maximum, byte_count overflow) |
| 13.14.1 sanitized native key, never an AX Session ID | `SanitizeNativeKey` | `TestIdentityRefusals/sanitizer`, raw/capture/session/event refusal rows |
| Sanitizer exact bound (pinned text leaves scope open) | `SanitizeNativeKey` | `TestIdentityRefusals/sanitizer` corpus: refuses empty, Cc+C1 controls, Cf/Zl/Zp invisible format marks (U+200B/U+2028/U+2029), invalid UTF-8, absolute paths, UUIDv7, password-colon-before-@ (schemed or schemeless), case-folded `urn:ax:`; admits spaces, bare `@`, bare `:` |
| Row 21 exclusion allowlist at construction | `refuseExcludedMember` | `TestRawManifestExcludesTrustMaterial`, `TestCaptureManifestExcludesTrustMaterial` |
| 13.14.1 CaptureItem 9-class vocabulary | `ValidCaptureClass`, `buildCaptureItem`, `decodeCaptureItem` | `TestCaptureClassVocabularyIsPinned`, `TestCaptureItemRefusals` |
| 13.14.1 included/excluded content rule | `checkItemDisposition` | `TestCaptureItemRefusals` (6 content rows) |
| 13.14.1 always-excluded classes never included | `alwaysExcludedClass` | `TestCaptureItemRefusals` (4 class rows) |
| 13.14.1 exact excluded_classes | `excludedRowClasses`, `excludedClassesExact` | `TestExcludedClassesExactGate`, `TestCaptureManifestRefusals/decode` |
| 13.14.1 core-derived raw_complete + reconciliation | `deriveRawComplete`, `VerifyCaptureReconciliation` | `TestBuildCaptureManifestDerivesRawComplete`, `TestBuildCaptureManifestUnknownMakesIncomplete`, `TestCaptureManifestRefusals/reconciliation` |
| 13.14.1 unknown blocks maximal_safe | `RefuseMaximalSafeUnlessComplete` | `TestBuildCaptureManifestUnknownMakesIncomplete`, `TestCaptureManifestRefusals/reconciliation` |
| 13.14.1 Capture Source Basis union | `DecodeSourceBasis`, `buildSourceBasis` | `TestSourceBasisRefusals` |
| 13.14.1 Capture Boundary union, stable proof coupling | `DecodeCaptureBoundary`, `buildBoundary`, `checkProofCoupling` | `TestBoundaryRefusals` (incl. input_blocked row), `TestBuildCaptureManifestUnstableArchive` |
| 13.14.1 unstable is core-created, G2 rejects | `buildBoundary` (Core), `RefuseUnstableForTarget` | `TestBoundaryRefusals/build_core-only_unstable`, `TestBuildCaptureManifestUnstableArchive` |
| 13.14.1 NativeIdentity closed shape | `DecodeNativeIdentity`, `EncodeNativeIdentity` | `TestNativeIdentityRoundTrip`, `TestIdentityRefusals/native_identity`, `TestEncodeNativeIdentityRefusals` (encode validates extensions, native keys, kind text) |
| 13.14.1 source_identity_digest terminal digest | `IdentityDigest` | `TestIdentityDigestIsStable`, `TestBuildRawManifestRoundTrip` |
| 13.14.1 WorkspaceBinding closed shape, normalized cwd | `DecodeWorkspaceBinding`, `checkCwdRelative` | `TestIdentityRefusals/workspace_binding` |
| 13.14.1 Canonical Session + Actor rules | `BuildCanonicalSession`, `DecodeCanonicalSession` | `TestBuildCanonicalSessionRoundTrip`, `TestCanonicalSessionRefusals` (incl. decode external-null-parent, version, not-an-array, the 1025-actor / 1000001-event / 1025-head far-edge refusals at build and decode: `build_actors/too_many_actors` + `N-actors-max-build`, `build_envelope/too_many_events` + `N-eventids-max-build`, `build_envelope/too_many_heads` + `N-heads-max-build`, `decode/too_many_actors` + `N-actors-max-decode`, `decode/too_many_events` + `N-eventids-max-decode`, `decode/too_many_heads` + `N-heads-max-decode`) |
| 13.14.1 Canonical Event envelope + 26 kinds + visibility | `BuildCanonicalEvent`, `DecodeCanonicalEvent`, `ValidEventKind` | `TestEventKindVocabularyIsPinned`, `TestBuildCanonicalEventRoundTrip`, `TestCanonicalEventRefusals` (incl. ordinal/offset/length overflow, version, not-an-array, raw_refs/reason-codes maxima, the 65-parent far-edge refusals at build and decode: `build_envelope/too_many_parents` + `N-parents-max-build`, `decode/too_many_parents` + `N-parents-max-decode`), `TestCanonicalEventPayloadValueModel` |
| 13.14.1 SourceEvidence + RawReference + synthesized coupling | `DecodeSourceEvidence`, `buildSourceEvidence`, `checkEvidenceCoupling` | `TestBuildCanonicalEventRoundTrip`, `TestCanonicalEventRefusals/evidence`, `TestCanonicalEventRefusals/decode/evidence_raw_refs_unsorted` (distinct unsorted refs refuse at decode) + `N-rawrefs-unsorted-decode` |
| 13.14.1 message-like content blocks, 64 KiB inline bound | `checkPayload`, `checkContentBlocks` | `TestCanonicalEventRefusals/build_envelope` (block rows incl. null content/descriptor, top-level duplicate, 64 KiB boundary) |
| 13.14.1 contiguous ordinals from zero | `CheckOrdinalContiguity` | `TestOrdinalContiguity` |
| 13.14.1 immutable generations | `ParseGeneration`, `Generation.Equal`, `CheckDigestsEqual` | `TestGenerationRefusals`, `TestBoundaryRefusals` |
| 13.14.1 reverse-DNS extensions, no normative reference | `encodeExtensions`, `checkExtensionsClosed` | contract round trips + `bad extensions` rows in every refusal table |
| 1.6 AX number model + nested duplicate refusal on extension values, every site | `checkExtensionsClosed`, `checkExtensionValues`, `checkAXNumberLiteral` | `TestRawManifestRefusals/extension_values` (1.5, 2^53, -(2^53), 2^60, exponent, nested float/unsafe, nested duplicate incl. resealed) + `TestExtensionValueModelAtEverySite` (one value-fault row per admission site, 17 rows) |
| 1.6 string measure in characters | `stringLength` via `environ` | `TestMultibyteStringMeasure` (300-char/600-byte admit, 512 admit, 513 refuse) |
| 1.6 text MUST be valid UTF-8 at Build (no rewrite-and-continue to U+FFFD) | `validText` at every Build string admission, `EncodeNativeIdentity` identity sanitize, `encodeExtensions` text walk | `TestBuildTextMustBeValidUTF8` (22 rows + collision pair, sealed-bytes U+FFFD assertions) |
| 13.14.1 one item per plan candidate at construction | `checkPlanItemMatch` | `TestCaptureManifestRefusals/build_plan_match` (extra, missing, renamed candidate, duplicate plan keys) |
| 13.14.1 content block inline-XOR-descriptor | `checkContentBlocks` | `TestCanonicalEventRefusals/build_envelope` (neither, both) |
| 13.14.1 duplicate raw keys refused | `VerifyCaptureReconciliation`, `deriveRawComplete` | `TestCaptureManifestRefusals/reconciliation` (duplicate raw/plan keys, derive direct) |
| 13.14.1 source_identity_digest over sanitized bytes | `IdentityDigest` (sanitizes before encoding, re-decodes before hashing) | `TestIdentityDigestIsStable`, `TestIdentityDigestRefusals` (bad kind, bad extension key, unsanitized opaque/session) |
| 10.2 Blob Descriptor identity + byte-count agreement | `VerifyDescriptorAgreement` (with entry identity link) | `TestBlobAgreementRefusals`, `TestBuildRawManifestRoundTrip`, `TestVerifyRawManifestDescriptorsRefusals` |
| 10.2 entry names the verified descriptor | `buildRawEntries`, `VerifyRawManifestDescriptors` | build `descriptor id mismatch` row, `TestVerifyRawManifestDescriptorsRefusals` (fetch/absent/wrong/link) |
| 10.2 no-replace install (fsync + verify + atomic) | `InstallRawBlob` via `localstore.PutBlob` | `TestInstallRawBlobIsIdempotent`, `TestInstallRawBlobRefusals` (forged claim, short/long bytes) |
| 7.8 tuple/probe/manifest/discover/inspect reuse | `sessadapter.DecodeTuple` at every `source_environment` | `bad tuple` rows in raw/capture/session/event tables |
| Byte-identical construction | all Build entries | `RoundTrip` tests rebuild and compare bytes |

## Durability bound

This package performs no durable write: `InstallRawBlob` delegates
every byte to `localstore.ObjectStore.PutBlob` (no-replace install,
fsync, size/digest verification, quarantine on collision). The
landed crash/idempotency evidence for that discipline lives with
`internal/localstore` (`object_store_test.go`,
`object_store_fault_unix_test.go`, projection fault suites);
`TestInstallRawBlobIsIdempotent` proves the delegation end to end
(double install agrees, tampered bytes refuse). No second write
path exists in this package.

## Stated bounds (sibling scope)

- Per-kind Canonical Event payload fact registries past the envelope
  (which facts a kind may carry) are the normalization sibling's
  registry. The Section 1.6 value model on every payload member
  (integer literals, |n| <= 2^53-1, no fraction/exponent,
  duplicate-free nested objects, depth bound) is enforced here for
  every kind by `checkPayloadValues`, and payload numbers decode to
  exact int64, so sealed bytes never round or collapse:
  `TestCanonicalEventPayloadValueModel` (build + decode negatives
  per non-message kind, sealed-bytes exactness, resealed decode,
  message regression, depth bound).
- `-0` admits in extension values and payloads (sealed as `0`, the
  JCS form); leading-`+` (`+1`) and leading-zero (`01`) literals
  are refused at the frame gate (`decodeStrictObject` reports "not
  a JSON object", since the tokenizer cannot read them). The model
  refuses fractions, exponents, and magnitudes at or beyond 2^53.
- `BoundaryInput.Core` is caller-asserted; the real unstable-archive
  protection is `RefuseUnstableForTarget` at target admission.
- `InstallRawBlob`'s and `VerifyDescriptorAgreement`'s descriptor
  member-parse arms (`blob_id`/`size`/self-field/object shape)
  repeat the owner grammar after `CalculateObjectIdentity` already
  enforced it (`validateBlobDescriptor` requires a digest
  `blob_id`, a uint53 `size`, and the `descriptor_id` self field):
  defense in depth, unreachable on failure, pinned by the owner
  suite; the agreement/install negatives pin the claim comparison
  and the store size/digest enforcement end to end.
- The raw-manifest `total_bytes` uint53 overflow arm needs entry
  byte counts summing past 2^53-1 with agreeing descriptors (real
  payloads cannot reach it): defense in depth behind the
  per-entry uint53 arm, which carries a `byte_count overflow` row.
- C1 beyond U+0085 (e.g. U+009F), the fraction arm of
  `checkAXNumberLiteral` (subsumed by `ParseInt`), and the exact
  256/257 depth edge are owner-informational: the class is pinned
  (U+0085, `1.5` refusals, the 300-deep payload refusal) without
  pinning every member.
- The extension text walk trusts the marshaled form of values with
  a computed representation (custom `json.Marshaler`
  implementations): it validates strings, map keys, and elements
  reachable by shape, not bytes a `MarshalJSON` method computes.
  Every realistic extension value (JSON-shaped maps, slices, and
  strings, plus struct values) is walked, pinned by the
  `raw_extensions*` rows of `TestBuildTextMustBeValidUTF8`.
- Per-kind Canonical Event payload fact registries beyond the
  Section 13.14.1 envelope enforced in `checkPayload` (normalization
  sibling).
- `maximal_safe` projection planning past
  `RefuseMaximalSafeUnlessComplete` (projection sibling).
- Target-branch (G2) admission past `RefuseUnstableForTarget`.
- Section 7.8 capture-plan/capture/normalize operation wiring
  (stays owned by `sessadapter`); this package owns the contracts
  those operations produce, not the operation frames.
- `internal/traceability` registry bindings and digest re-pin
  (Story FINAL leaf).
