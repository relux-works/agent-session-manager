# TASK-260830-24z2b3 Conformance Matrix (rev5, re-executed on `a12d1bd`)

Rev5b completion note: the first rev5 construction failed at
validation command 6 on the trunk resumesmoke time-bomb
(BUG-260918-354b03), not on the candidate. The candidate was
refreshed `2fc6d50` → `a12d1bd` (trunk fix, audit clean, candidate
code unchanged) and every row below was re-executed on the new base.
No row changed; the ratio stays 18 of 18.

Authority: `internal/specdoc/SPEC.v0.7.0.md`, Sections 1.6, 7.8,
10.2, 13.14.1 (§1.6 and §13.14.1 verified byte-identical to
v0.6.0). Package: `internal/clonebundle`. Normative spec text is
quoted from the pinned file; the prose here maps, never restates.

Rev5 answers review RUN-260917-ab25c8: P2-δ (shared valid-UTF-8
text gate at every Build admission + encode-time identity
sanitize; sealed bytes never carry a substituted U+FFFD) and the P3
list (clean evidence archive, corrected `-0` sentence, build
reason-duplicate row + plant, producer-only checklist).

## AC coverage: 18 of 18 rows driven

Every row is driven through the named production entry point by the
named committed test. Prose ratio: **18 of 18 AC rows driven**.

| # | AC row | Production call site | Named test |
|---|---|---|---|
| 1 | Raw Object Manifest closed envelope + omit-self ID | `BuildRawObjectManifest`, `DecodeRawObjectManifest` | `TestBuildRawManifestRoundTrip`, `TestRawManifestRefusals/decode` |
| 2 | RawObjectEntry class subset (credential/auth/runtime/lock forbidden) | `buildRawEntries`, `decodeRawEntries`, `validRawEntryClass` | `TestRawManifestRefusals/build` (unknown + 4 forbidden-class rows) |
| 3 | Sorted unique entries, total_bytes = sum | `buildRawEntries`, `decodeRawEntries` | `TestRawManifestRefusals` (unsorted, duplicate at build + decode, total mismatch, not-an-array, 65536 maximum, byte_count overflow) |
| 4 | Sanitized native keys, never an AX Session ID (+ exact bound: C1/invisible/UTF-8/userinfo/case) | `SanitizeNativeKey` | `TestIdentityRefusals/sanitizer` (incl. U+2029, invalid UTF-8) + key rows in raw/capture/session/event tables |
| 5 | Row-21 exclusion allowlist at construction | `refuseExcludedMember` | `TestRawManifestExcludesTrustMaterial`, `TestCaptureManifestExcludesTrustMaterial` |
| 6 | Capture Manifest envelope + omit-self ID + decode item rules | `BuildCaptureManifest`, `DecodeCaptureManifest`, `decodeCaptureItems`, `decodeCaptureItem` | `TestBuildCaptureManifestDerivesRawComplete`, `TestCaptureManifestRefusals/decode` (incl. mystery class, dup/unsorted items, disposition/content rows, version, not-an-array, 65536 maximum) |
| 7 | CaptureItem 9-class vocabulary + included/excluded content rule | `ValidCaptureClass`, `checkItemDisposition` | `TestCaptureClassVocabularyIsPinned`, `TestCaptureItemRefusals`, decode rows in `TestCaptureManifestRefusals/decode` |
| 8 | Always-excluded classes never included | `alwaysExcludedClass` | `TestCaptureItemRefusals` (credential, machine_auth, runtime_state, transient_lock rows) + decode `credential_included` row |
| 9 | excluded_classes exactly the excluded-row classes | `excludedRowClasses`, `excludedClassesExact` | `TestExcludedClassesExactGate`, `TestCaptureManifestRefusals/decode` (extra, missing, unsorted, unknown, duplicate) |
| 10 | Core-derived raw_complete + plan/object reconciliation + one-per-plan-candidate + dup-key refusal | `deriveRawComplete`, `VerifyCaptureReconciliation`, `checkPlanItemMatch` | `TestBuildCaptureManifestDerivesRawComplete`, `TestBuildCaptureManifestUnknownMakesIncomplete`, `TestCaptureManifestRefusals/reconciliation`, `TestCaptureManifestRefusals/build_plan_match` (incl. duplicate plan keys) |
| 11 | Unknown blocks maximal_safe | `RefuseMaximalSafeUnlessComplete` | `TestBuildCaptureManifestUnknownMakesIncomplete`, `TestCaptureManifestRefusals/reconciliation` |
| 12 | Capture Source Basis closed union | `DecodeSourceBasis`, `buildSourceBasis` | `TestSourceBasisRefusals`, `TestExtensionValueModelAtEverySite/source_basis_ax`, `/source_basis_external` |
| 13 | Capture Boundary union + stable-proof coupling | `DecodeCaptureBoundary`, `buildBoundary`, `checkProofCoupling` | `TestBoundaryRefusals` (incl. input_blocked row), `TestBuildCaptureManifestUnstableArchive`, per-site stable_proof/boundary rows, manifest `boundary_proof_unequal_digests` row |
| 14 | Unstable is core-created; G2 rejects it | `buildBoundary` (Core), `RefuseUnstableForTarget` | `TestBoundaryRefusals/build_core-only_unstable`, `TestBuildCaptureManifestUnstableArchive` |
| 15 | NativeIdentity + WorkspaceBinding identities + digest re-decode + encode validation incl. encode-time identity sanitize | `DecodeNativeIdentity`, `EncodeNativeIdentity`, `IdentityDigest`, `DecodeWorkspaceBinding` | `TestNativeIdentityRoundTrip`, `TestIdentityDigestIsStable`, `TestIdentityDigestRefusals`, `TestIdentityRefusals`, `TestEncodeNativeIdentityRefusals` (incl. unsanitized-identity row), `TestBuildTextMustBeValidUTF8` (identity native/opaque/digest rows), per-site identity/workspace rows |
| 16 | Canonical Session + Actor rules (one main, event/head IDs) + decode actor rules + dup heads | `BuildCanonicalSession`, `DecodeCanonicalSession`, `decodeActor` | `TestBuildCanonicalSessionRoundTrip`, `TestCanonicalSessionRefusals` (incl. decode main-with-parent, external-null-parent, null-parent, ghost kind, dup heads, version, not-an-array, 1024-actor maximum), `TestBuildTextMustBeValidUTF8` (title/actor rows), per-site actor/session rows |
| 17 | Canonical Event + SourceEvidence + raw refs + ordinals + generations + decode event rules + block XOR + payload value model | `BuildCanonicalEvent`, `DecodeCanonicalEvent`, `DecodeSourceEvidence`, `checkPayload`, `checkPayloadValues`, `checkContentBlocks`, `CheckOrdinalContiguity`, `ParseGeneration` | `TestEventKindVocabularyIsPinned`, `TestBuildCanonicalEventRoundTrip`, `TestCanonicalEventRefusals` (incl. decode visibility/parents/evidence rows, XOR rows incl. null arms, top-level duplicate, ordinal/offset/length overflow, version, not-an-array, raw_refs/reason-codes maxima, build `reasons duplicate`), `TestCanonicalEventPayloadValueModel`, `TestBuildTextMustBeValidUTF8` (evidence/generation rows), per-site event/evidence/payload/block rows |
| 18 | Blob Descriptor identity + entry identity link + byte-count agreement + no-replace install | `VerifyDescriptorAgreement`, `InstallRawBlob`, `VerifyRawManifestDescriptors` | `TestBlobAgreementRefusals`, `TestInstallRawBlobIsIdempotent`, `TestInstallRawBlobRefusals`, `TestVerifyRawManifestDescriptorsRefusals`, `TestBuildRawManifestRoundTrip` |

## Section 1.6 common-model rows (measured, not counted as AC rows)

| Rule | Production call site | Named test |
|---|---|---|
| AX number model + nested-duplicate refusal on extension values, every admission site | `checkExtensionsClosed`, `checkExtensionValues`, `checkAXNumberLiteral` | `TestRawManifestRefusals/extension_values` + `TestExtensionValueModelAtEverySite` (17 rows) |
| AX number model + nested-duplicate refusal on every event payload member + exact-int64 sealing | `checkPayloadValues`, `convertPayloadNumbers` | `TestCanonicalEventPayloadValueModel` (build/decode/resealed negatives, sealed-bytes exactness, message regression, depth bound) |
| String measure in characters | `stringLength` via `environ` | `TestMultibyteStringMeasure` |
| Text MUST be valid UTF-8 at Build: no rewrite-and-continue to U+FFFD at any admission (incl. lone surrogates, extension keys/values at any depth, identity fields before encode) | `validText`, `validExtensionText`, `EncodeNativeIdentity` identity sanitize | `TestBuildTextMustBeValidUTF8` (22 rows + collision pair; every row asserts the sealed bytes carry no U+FFFD) |

## Negative/refusal evidence

Every refusal above fails when the gate admits what it must reject:
the 66 narrowing mutants in `internal/clonebundle/testdata/mutant_harness.py`
weaken each gate to admit exactly one forbidden member and the named
killer fails (66 KILLED, 1 harmless SURVIVED control, full battery
run twice). Per-plant raw logs with subprocess exits are in the
producer evidence tar (`mutants/run1/`, `mutants/run2/`).

## Landed-owner reuse (not new behavior, cited not counted)

| Scope | Owner | Evidence |
|---|---|---|
| Section 7.8 operation frames (capture-plan/capture/normalize fresh isolated sinks; probe/manifest/discover/inspect read-only) | `internal/sessadapter` (`CheckRequestBody`, `CheckSuccessBody`, `CheckFreshSink`) | landed sessadapter suite, green in this run |
| Section 10.2 crash/idempotency discipline (fsync, atomic no-replace, quarantine) | `internal/localstore` | landed localstore fault suites, green in this run; delegation proven by `TestInstallRawBlobIsIdempotent` + `TestInstallRawBlobRefusals` |
| Scalar gates + string measure | `internal/environ` (delegating wrappers) | `TestMultibyteStringMeasure`, owner batteries green in this run |
| Closed-shape registry rows for these schemas | `internal/canonicaljson` (`rejectUnsupportedImmutableObjectShape`, census-pinned) | intentionally not replaced; this package is the validating owner (see `doc.go`) |

## Stated bounds (sibling scope, not waived)

- Per-kind Canonical Event payload fact registries: which facts a
  kind may carry (normalization sibling). The §1.6 value model on
  every payload member is enforced here.
- `maximal_safe` projection planning past
  `RefuseMaximalSafeUnlessComplete` (projection sibling).
- Target-branch (G2) admission past `RefuseUnstableForTarget`.
- Section 7.8 capture-plan/capture/normalize operation wiring
  (stays `sessadapter`-owned).
- `-0` admits (sealed as `0`, the JCS form); leading-`+` (`+1`)
  and leading-zero (`01`) literals refuse at the frame gate.
- The extension text walk trusts the marshaled form of values with
  a computed representation (custom `json.Marshaler`).
- Descriptor member-parse arms, total_bytes overflow arm, and the
  informational trio (C1-beyond-U+0085, fraction subsumption, exact
  depth edge) as defense-in-depth per TRACEABILITY.md.
- `internal/traceability` registry bindings and digest re-pin
  (Story FINAL leaf).
