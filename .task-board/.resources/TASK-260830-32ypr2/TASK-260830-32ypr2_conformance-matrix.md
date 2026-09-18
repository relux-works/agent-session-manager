# TASK-260830-32ypr2 Conformance Matrix (rev5) — test-unknown-encrypted-and-incomplete-events

Authority: `internal/specdoc/SPEC.v0.7.0.md`, Sections 7.8, 10.2, 13.14.1.
Production entry for every row: `Normalize` (`internal/cloneproject/normalize.go:97`).
Every proof builds an on-disk fixture provider store, runs the real
`clonesnap.Capture` entry (`internal/clonesnap/capture.go`), then `Normalize`
over the sealed manifests and installed blobs. No row drives hand-written
Canonical Events or calls an unexported helper directly (verified: no test
references `dispatchRecord`, `dispatchProtected`, `resolveToolCalls`,
`deriveLiveSurface`, `foldInstructions`, `foldUsage`, `contentBlockForText`,
`installOverflow`, `parseNativeLine`, `collectMembers`, `splitRecordLines`,
`strictMembers`, `bodyUint53`, `decodeBodyObject`, `bodyString`, or
`checkActorSelector`).

## A. AC coverage: 47 of 47 rows driven and measured

One row per behavior the brief's four invariants name plus the
rework-mandated strict-framing, value-model, alias-axis, and gate
rows. "Narrowing" names the harness row(s) in
`internal/cloneproject/testdata/mutant_harness.py` that redden the
row's killer(s); every narrowing admits exactly one member of the
class its gate must reject. "(killer)" marks the plant whose
committed killer is the row's test; other entries name the shared
gate the plant narrows. Battery: 80 narrowings + 5 labelled rows
KILLED + control SURVIVED, executed twice, exit 0 both runs, one
raw log per plant per run under `mutants/run1/` + `mutants/run2/`
in the producer evidence archive. The sibling `internal/clonebundle`
battery (76 rows incl. the 9 new far-edge narrowings) runs twice
with per-plant logs under `mutants/sibling-clonebundle-run1/` +
`mutants/sibling-clonebundle-run2/`.

| # | AC behavior | Driving test(s) → call site | Narrowing(s) |
|---|---|---|---|
| 1 | Wholly unknown native type → `opaque_event`, `opaque` visibility, `unknown_native_event` reason, identity kept | `TestNormalizeUnknownRecordBecomesOpaqueEvent` → `Normalize` → `dispatchRecord` default arm | `N-unknown-kind-wholly` (killer: admits exactly `frobnicate` into `user_message`) |
| 2 | Almost-known record (right envelope, unknown inner type) never coerced into a neighbouring kind | `TestNormalizeAlmostKnownRecordBecomesOpaqueEvent` → same arm | `N-unknown-kind` (killer: admits exactly `message/smoke` into `user_message`) |
| 3 | Unknown-class member → whole-member `opaque_event` (offset 0, full length, null native IDs) | `TestNormalizeUnknownMemberBecomesWholeMemberOpaque` → `Normalize` → `collectMembers` | `N-unknown-member` (killer: admits exactly `store/mystery` into JSONL parsing) |
| 4 | Evidence ranges resolve to the exact captured bytes at every offset (manifest + descriptor linkage, in range, offsets asserted past 0) | `TestNormalizeMultiRecordRefsResolveAtAssertedOffsets` (every event's `RawRefs[0]` resolves to its own line; offsets `0`, `len(l1)+1`, `len(l1)+len(l2)+2` asserted; multibyte first line) + `TestNormalizeUnterminatedTailResolvesWithOffset` (a multi-line member without a trailing newline keeps its last record with offsets `0`, `len(l1)+1` asserted) + `TestNormalizeCRLFResolvesWithCarriageReturn` (each CRLF range covers its line plus the CR and resolves exactly) + `mustResolveRef` assertions in rows 1–3 tests + `mustResolveOverflow` in row 20 test; single-ref evidences asserted `len==1` (trivially sorted/unique) | `N-offset-newline` (labelled behaviour swap, killer: drops the newline stride); `N-unterminated-tail-drop` (labelled behaviour drop, killer: drops the unterminated last line of a multi-line member); `N-digest-verify`, `N-missing-blob` (fetch-integrity gates, killers on the refusal rows); the reviewer `RV-offset-line-number` and `RV-evidence-offset-second-line` shapes KILLED firsthand by the multi-record row |
| 5 | Never dropped, guessed, or re-typed on a second projection | `TestNormalizeOpaqueIsStableAcrossProjections` (`Normalize` ×2 over fresh sinks, byte-equal, still `opaque_event`) | gate-level: rows 1–3 narrowings pin the kinds; rows 25–28 pin the no-guess axes (strict framing); row 39 pins the no-coercion alias axis; structural: `Normalize` holds no cross-call state |
| 6 | Foreign encrypted reasoning → `opaque_reasoning`, `foreign_encrypted_payload`, no content members, byte-exact round trip | `TestNormalizeForeignEncryptedReasoningStaysOpaque` → `Normalize` → `dispatchProtected` | `N-protection-encrypted` (killer; token-preserving: the `encrypted` label still matches, the decision changes) |
| 7 | Foreign signed reasoning → `opaque_reasoning`, `foreign_signature_unverifiable`, byte-exact | `TestNormalizeForeignSignedReasoningStaysOpaque` → same gate | `N-protection-signed` (killer: admits exactly `signed` past the gate) |
| 8 | Never decrypted, re-encoded, summarized, truncated, or promoted; native-protected carries no foreign code; protected known content never semantic; unknown+protected carries both reasons | `TestNormalizeNativeProtectedReasoningStaysOpaque` + `TestNormalizeProtectedMessageBecomesOpaqueEvent` + `TestNormalizeUnknownProtectedRecordCarriesBothReasons` + forbidden-member sweeps (`content_blocks`, `content`, `text`, `ciphertext`, `summary` absent) in rows 6–7 tests | `N-native-reason-foreign` (killer), `N-reasons-both-single` (killer), `N-protection-encrypted` cross-kill on the protected-message test (verified by hand probe at rev1, `crosskill-encrypted-vs-protected-message.log`: exit 1 under the plant, `ok` restored); row 39 pins the no-promotion alias axis |
| 9 | Foreign instructions are low-authority history: cannot change the effective snapshot or authority | `TestNormalizeForeignInstructionCannotChangeEffectiveSnapshot` + `TestNormalizeForeignOnlyInstructionLeavesSnapshotEmpty` → `foldInstructions` / `dispatchInstruction` | `N-foreign-instruction`, `N-foreign-payload-authority` (killers); row 39 pins the no-promotion alias axis |
| 10 | Source usage never becomes target accounting | `TestNormalizeSourceUsageIsNotTargetAccounting` → `foldUsage` | `N-usage-target` (killer); `N-usage-number`, `N-usage-string`, `N-usage-magnitude`, `N-usage-overflow` (input arm), `N-usage-overflow-output` (output arm, killer on `usage_ledger_overflow_output`) (value model) |
| 11 | `tool_definition_snapshot` is history, `callable` false, never registers; live claims refuse | `TestNormalizeToolDefinitionNeverRegistersCallable` + refusal row `live_attestation_claim` → `dispatchDefinition` / `deriveLiveSurface` | `N-callable-false`, `N-callable` (killers) |
| 12 | Call with result → `completed`, `internal` visibility, empty live surface | `TestNormalizeToolMatrix/call_with_result` → `resolveToolCalls` / `dispatchCall` | `N-pending-completed` (killer) |
| 13 | Call without result → `aborted`, `unsafe_pending_action`, blocks pending action | `TestNormalizeToolMatrix/call_without_result` → same | `N-tool-incomplete` (killer narrowing); `N-pending-aborted` (labelled add-arm killer: `Normalize` passes nil work orders, so the plant appends the call explicitly) |
| 14 | Result without call → `aborted` orphan | `TestNormalizeToolMatrix/result_without_call` → `resolveToolCalls` / `dispatchResult` | `N-tool-orphan` (killer) |
| 15 | Result arriving after the capture boundary completes the same call | `TestNormalizeResultAfterBoundaryCompletes` (first capture `aborted`, second `completed`) → same pairing decision | gate-level: `N-tool-incomplete` (same `resolveToolCalls` decision; no dedicated plant — disclosed) |
| 16 | Nested/subagent call attributes to a parented subagent actor, never main | `TestNormalizeToolMatrix/nested_subagent_call` → `checkActors` / `buildResult` | `N-subagent-actor` (killer); row 39 pins the no-reattribution alias axis |
| 17 | `internal` visibility and no pending action for every matrix row; live surface empty over mixed history | visibility assertions in every matrix subtest + `TestNormalizeLiveSurfaceStaysEmpty` → `deriveLiveSurface` | `N-pending-completed`, `N-live-followup` (killer narrowings on the named rows); `N-pending-aborted` (labelled add-arm: nothing captured can pend — matrix §D bound) |
| 18 | Canonical Event exact shape: contiguous ordinals from zero, predecessor parents, UUIDv7 actors, literal kinds/visibilities, landed re-decode of every sealed byte | `TestNormalizeSealsSessionShape` → `buildEvents` → `clonebundle.BuildCanonicalEvent` + `DecodeCanonicalEvent` | `N-parents-chain`, `N-native-type-token` (killers; token-preserving) |
| 19 | Source Evidence `exact` status with stable reason codes | reason assertions in rows 1, 6, 7, 13 tests + row 41 (status asserted for every emitted kind) → `evidenceFor` | `N-reasons-both-single`, `N-native-reason-foreign`, `N-capture-status-partial` (killers) |
| 20 | 64 KiB edge: 65536-byte text inline, 65537-byte text a byte-exact blob reference, never a truncation | `TestNormalizeInlineContentBoundary` → `contentBlockForText` / `installOverflow` | `N-overflow-bound` (killer); `N-overflow-truncate` (labelled behaviour swap, killer) |
| 21 | Canonical Session shape: main + referenced extras, ordinal-ordered unique IDs, last event as head, parent null only for main | `TestNormalizeSealsSessionShape` (incl. `ParentActorID == nil` for main) + row 16 subagent-parent assertion → `buildResult` → `clonebundle.BuildCanonicalSession` + decode | `N-main-parent` (killer: admits exactly main to the parented class; the landed builder refuses), `N-parents-chain` |
| 22 | Determinism: same captured bytes → byte-identical session + events twice, along both axes | `TestNormalizeIsDeterministic` (base proof) + `TestNormalizeActorOrderIsDeterministic` (3 extra actors × 12 projections, one distinct session) | `N-actors-sort` (killer on the actor-order axis: leaves exactly the 3-actor shape unsorted) |
| 23 | Every refusal through `Normalize` naming the member, line, or blob | `TestNormalizeRefusalTable` (54 subtests) + `TestNormalizeRefusesUnstableCapture` + `TestNormalizeRefusesEnvironmentDrift` | one narrowing per family, each killing its row: `N-envelope-version`, `N-envelope-version-string`, `N-origin`, `N-protection-value`, `N-actor-selector`, `N-actor-empty-subagent-name` (message pin: admits `subagent:` past the shape arm, still refuses at the actor mapping), `N-malformed-json`, `N-blank-line`, `N-body-member`, `N-allowed-call-live-followup`, `N-allowed-result-live-attestation`, `N-body-text-number`, `N-body-bool-string`, `N-body-required-skip-text`, `N-instruction-authority`, `N-directive-count`, `N-directive-empty`, `N-definition-digest`, `N-usage-number`, `N-usage-string`, `N-usage-magnitude`, `N-usage-overflow`, `N-usage-overflow-output`, `N-duplicate-call`, `N-duplicate-result`, `N-live-followup`, `N-callable`, `N-actor-unmapped-fallback`, `N-actor-unreferenced`, `N-main-in-map`, `N-subagent-name`, `N-manifest-link`, `N-missing-blob`, `N-digest-verify`, `N-size-verify`, `N-empty-projection` (labelled single-member class), `N-unstable`, `N-env-drift`, `N-string-bound`, `N-reasoning-contradiction`, `N-reasoning-encrypted-signed`, `N-reasoning-signed-encrypted`, `N-result-status-pending`, `N-unknown-protected-body`, `N-required-member-body`, `N-strict-text-surrogate`, `N-strict-envelope-surrogate`, `N-strict-envelope-duplicate`, `N-strict-body-duplicate` (the five `N-strict-*` rows patch the shared `strictMembers` body: one gate, two call sites), `N-trailing-admitted` |
| 24 | No partial bundle on failure; idempotent replay into a shared sink (byte-identical output, stable blob count) | `TestNormalizeFailureReturnsNoPartialBundle` + `TestNormalizeIsIdempotentIntoSharedSink` | none: failure-injection proofs, not gates; crash semantics of the write itself owned by `internal/localstore` (landed fault suites) |
| 25 | Lone-surrogate text member refuses, never rewritten to U+FFFD | `TestNormalizeRefusalTable/lone_surrogate_text` → `Normalize` → `strictMembers` + `environ.DecodeStrictObject` (line and body frame sites) | `N-strict-text-surrogate` (killer: admits exactly the killer line past the strict gate at both frame sites into a lenient decode) |
| 26 | Lone-surrogate envelope member refuses, evidence identity never rewritten | `TestNormalizeRefusalTable/lone_surrogate_envelope` → same gate | `N-strict-envelope-surrogate` (killer) |
| 27 | Duplicate envelope member refuses, never coerced into a neighbouring kind | `TestNormalizeRefusalTable/duplicate_envelope_member` → same gate | `N-strict-envelope-duplicate` (killer: last-wins decode coerces into `user_message` under the plant) |
| 28 | Duplicate body member refuses, never guessed last-wins | `TestNormalizeRefusalTable/duplicate_body_member` → same gate | `N-strict-body-duplicate` (killer) |
| 29 | String-typed numbers refuse: a uint53 member is a JSON integer | `TestNormalizeRefusalTable/string_usage_number` + `/string_envelope_version` → `bodyUint53` / envelope version check + `environ.CheckUint53Bounds` | `N-usage-string`, `N-envelope-version-string` (killers) |
| 30 | Actor-order determinism over ≥2 extra actors | `TestNormalizeActorOrderIsDeterministic` → `buildResult` actor sort | `N-actors-sort` (killer) |
| 31 | Reasoning type/protection agreement refuses per arm | `TestNormalizeRefusalTable/encrypted_type_signed_protection` + `/signed_type_encrypted_protection` → `dispatchProtected` | `N-reasoning-encrypted-signed`, `N-reasoning-signed-encrypted` (killers) |
| 32 | Protected non-reasoning tool records reach opaque only via the fold-skip | `TestNormalizeProtectedToolCallStaysOpaqueHistory` → `foldTools` skip + `dispatchProtected` | `N-protected-tools-skip` (killer: the admitted call refuses on its ciphertext body under the plant) |
| 33 | Result status vocabulary is exactly `ok|error` | `TestNormalizeRefusalTable/bad_result_status` → `decodeToolResultBody` | `N-result-status-pending` (killer: the call resolves `completed` under the plant) |
| 34 | Size arm refuses with its own message before the digest arm | `TestNormalizeRefusalTable/blob_size_drift` (+ `tampered_blob` asserting `blob digest disagrees`) → `collectMembers` | `N-size-verify` (killer: the drift reaches the digest arm with its different message under the plant) |
| 35 | Trailing bytes refuse | `TestNormalizeRefusalTable/trailing_bytes` → `strictMembers` + landed gate | `N-trailing-admitted` (killer: the lenient decode refuses with a different message under the plant) |
| 36 | Unknown+protected non-ciphertext body refuses (decided: only exactly-`ciphertext` protected bodies project opaque) | `TestNormalizeRefusalTable/unknown_protected_plaintext_body` → `dispatchProtected` | `N-unknown-protected-body` (killer: projects `opaque_event` under the plant) |
| 37 | Non-JSONL included members refuse the whole projection (stated bound, pinned) | `TestNormalizeNonJSONLIncludedMemberRefuses` (binary index + text cache subtests) → `collectMembers` JSONL parsing | gate-level: the strict-decode narrowings pin the enforcing gate; no dedicated plant (bound — disclosed) |
| 38 | Null directives decode as empty (stated bound, pinned) | `TestNormalizeNullDirectivesDecodeAsEmpty` → `decodeInstructionBody` | none: admitted shape, not a gate (disclosed) |
| 39 | Case-folded envelope aliases never override claimed members (kind in both document orders, origin/authority, protection/promotion, body/content, evidence identity, actor/attribution; unrelated-extra control stays admitted) | `TestNormalizeCaseAliasEnvelopeIgnoresUnclaimedMembers` (8 subtests) → `Normalize` → `parseNativeLine` strict-map envelope | `N-alias-native-type`, `N-alias-origin`, `N-alias-protection`, `N-alias-body`, `N-alias-native-event-id`, `N-alias-actor` (killers: each restores the case-folded read for exactly one member) |
| 40 | Unprotected reasoning summaries project `reasoning_summary`, `internal`, with content and `exact` | `TestNormalizeReasoningSummaryProjectsInternal` → `Normalize` → `dispatchMessage` default arm | `N-reasoning-summary-routed` (killer: routes exactly these records to `user_message`/`public`) |
| 41 | Every emitted kind stamps capture status `exact` | `TestNormalizeCaptureStatusExactForEveryKind` (one record per kind) → `evidenceFor` | `N-capture-status-partial` (killer: stamps `partial` for exactly `message/user`) |
| 42 | The native instruction fold is last-wins | `TestNormalizeLastNativeInstructionWins` (two native records) → `foldInstructions` | `N-instruction-fold-first` (killer: folds exactly the first record instead) |
| 43 | The external selector seals as an external actor parented on main | `TestNormalizeExternalActorSealsAsExternal` → `buildResult` | `N-external-actor-kind` (killer: admits exactly `external` into the subagent arm) |
| 44 | Unknown types without a body refuse (the body member is required for every record) | `TestNormalizeRefusalTable/unknown_type_without_body` → `parseNativeLine` required-member check | `N-required-member-body` (killer: a known type still refuses at the body decode, an unknown type without a body is admitted opaque) |
| 45 | Null text seals as empty inline content (stated bound, pinned) | `TestNormalizeNullTextDecodesAsEmpty` → `decodeMessageBody` | none: admitted shape, not a gate (disclosed) |
| 46 | Null optional booleans decode as false (stated bound, pinned) | `TestNormalizeNullOptionalBoolsDecodeAsFalse` (null attestation stays uncallable history; null followup completes) → `bodyOptionalBool` | none: admitted shape, not a gate (disclosed) |
| 47 | Present-but-null body on an unknown type projects opaque (stated bound, pinned) | `TestNormalizeUnknownTypeNullBodyProjectsOpaque` (body never read; missing body still refuses) → `parseNativeLine` required-member check + `dispatchRecord` default arm | none: admitted shape, not a gate (disclosed) |

Ratio: **47 of 47 AC rows driven** through the production entry by named
committed tests. No row is prose-only.

## B. Registry bindings claimed by this Story (tracecheck-measured)

| Clause | Acceptance case → executing test(s) | Measured |
|---|---|---|
| `7.8#1` large data by reference, never embedded in the 8 MiB frame | `session-adapter-frame-bound` → `TestDecodeRequestFrameRefusesOversizeFrame`, `TestFrameBoundEdges` (production `internal/sessadapter/protocol.go` `DecodeRequestFrame`, sessadapter owner) | full 2/2, `--section 7.8` admitted |
| `7.8#2` execution bindings equal fresh facts | `session-adapter-call-binding` → `TestCheckCallBinding`, `TestCheckCallBindingZeroFactsRefuse` (sessadapter owner) | full 2/2 |
| `10.2#1` metadata in manifest | `clone-capture-contracts` → `TestBuildRawManifestRoundTrip`, `TestBuildCaptureManifestDerivesRawComplete`, `TestBuildCanonicalEventRoundTrip`, `TestRawManifestRefusals`, `TestCaptureManifestRefusals` | full 5/5, `--section 10.2` admitted |
| `10.2#2` fsync + verify + atomic install | `clone-native-capture` → `TestCaptureSealsBothManifestsFromStoreBytes`, `TestCaptureInstallsEveryPayloadBlob`, `TestCaptureMultiChunkMember`, `TestCaptureLogRecordsDigestsOnly`, `TestCaptureRefusesOversizedMember`; plus `clone-projection-fidelity` → `TestNormalizeIsIdempotentIntoSharedSink`; plus `localstore-immutable-blob-install` (the landed fsync/verify/atomic-install owner) | full 5/5 |
| `10.2#3` digest-only metrics | `clone-native-capture` (same suite) | full 5/5 |
| `10.2#4` chunk agreement | `clone-native-capture` (same suite) | full 5/5 |
| `10.2#5` oversize refuses `capability_unavailable` | `clone-native-capture` (same suite) | full 5/5 |

The `section:10.2` binding owns the union of the eight landed trunk
cases (`scalar-digest-validation`,
`scalar-bounded-integer-validation`, `canonical-jcs-rfc8785`,
`canonical-object-identity`, `canonical-identity-refusal`,
`localstore-digest-path-v1`, `localstore-immutable-blob-install`,
`localstore-sqlite-projection`) and the three clone cases. The
projection overflow reference test (`TestNormalizeInlineContentBoundary`)
constructs no adapter frame and discharges no 7.8 clause.

`section:13.14.1` is deliberately NOT BOUND in the registry: the section
carries no RFC 2119 keyword line the `tracecheck` scanner can see (its
obligations are closed-shape "contains exactly" sentences), so any binding
would measure `unmeasured`, which assigned-scope admission deliberately
refuses. The Story's 13.14.1 record is the three package `TRACEABILITY.md`
files plus the three leaf matrices, executed by the three real suites bound
above as `clone-capture-contracts`, `clone-native-capture`, and
`clone-projection-fidelity`. This leaf's 13.14.1 rows are section C below.

## C. Section 13.14.1 clause record (this leaf)

Each row names the committed test that executes it through `Normalize`;
full entry + note detail lives in `internal/cloneproject/TRACEABILITY.md`
(Clause map), which this matrix incorporates.

| Pinned clause | Test |
|---|---|
| Unknown native records become raw-addressable opaque events | `TestNormalizeUnknownRecordBecomesOpaqueEvent`, `TestNormalizeMultiRecordRefsResolveAtAssertedOffsets` (every event's ref resolves to its own line with offsets asserted past 0), `TestNormalizeUnterminatedTailResolvesWithOffset` (a multi-line member without a trailing newline keeps its last record with its offset asserted), `TestNormalizeCRLFResolvesWithCarriageReturn` (each CRLF range covers its line plus the CR and resolves exactly) |
| Almost-known records never coerced into a neighbouring kind | `TestNormalizeAlmostKnownRecordBecomesOpaqueEvent` |
| Unknown-class members become whole-member opaque events | `TestNormalizeUnknownMemberBecomesWholeMemberOpaque` |
| Opaque events never dropped, guessed, or re-typed | `TestNormalizeOpaqueIsStableAcrossProjections`, `TestNormalizeCaseAliasEnvelopeIgnoresUnclaimedMembers/kind_alias_after`, `/kind_alias_before` |
| Foreign encrypted reasoning is opaque-preserved | `TestNormalizeForeignEncryptedReasoningStaysOpaque` |
| Foreign signed reasoning is opaque-preserved | `TestNormalizeForeignSignedReasoningStaysOpaque` |
| Native protected reasoning stays opaque with no foreign code | `TestNormalizeNativeProtectedReasoningStaysOpaque` |
| Protected known content never reaches a semantic kind | `TestNormalizeProtectedMessageBecomesOpaqueEvent` |
| Unknown+protected records carry both stable reasons | `TestNormalizeUnknownProtectedRecordCarriesBothReasons` |
| Foreign instructions are low-authority history | `TestNormalizeForeignInstructionCannotChangeEffectiveSnapshot`, `TestNormalizeForeignOnlyInstructionLeavesSnapshotEmpty`, `TestNormalizeCaseAliasEnvelopeIgnoresUnclaimedMembers/origin_alias` |
| Null directives decode as empty (stated bound) | `TestNormalizeNullDirectivesDecodeAsEmpty` |
| Source usage is not target accounting | `TestNormalizeSourceUsageIsNotTargetAccounting` |
| Historical tools inert; call+result resolves completed | `TestNormalizeToolMatrix/call_with_result` |
| Incomplete calls become aborted history and block pending action | `TestNormalizeToolMatrix/call_without_result`, `TestNormalizeToolMatrix/result_without_call` |
| Protected tool calls stay opaque history via the fold-skip | `TestNormalizeProtectedToolCallStaysOpaqueHistory` |
| Results arriving after the boundary complete the same call | `TestNormalizeResultAfterBoundaryCompletes` |
| Nested calls attribute to a parented subagent actor | `TestNormalizeToolMatrix/nested_subagent_call` |
| Definitions never register a callable tool | `TestNormalizeToolDefinitionNeverRegistersCallable` |
| Live surface stays empty over mixed history | `TestNormalizeLiveSurfaceStaysEmpty` |
| Canonical Session bounds (actors, ordered event IDs, head) | `TestNormalizeSealsSessionShape` |
| Inline content at most 64 KiB; oversized becomes a blob reference | `TestNormalizeInlineContentBoundary` |
| Same captured bytes project byte-identical output | `TestNormalizeIsDeterministic`, `TestNormalizeActorOrderIsDeterministic` |
| `unstable_archive` cannot enter target projection | `TestNormalizeRefusesUnstableCapture` |
| Manifest link, blob fetch, digest/size agreement (each arm its own message) | `TestNormalizeRefusalTable/manifest_closure_drift`, `/tampered_blob`, `/missing_blob`, `/blob_size_drift` |
| Framing-malformed records refuse with member+line named | `TestNormalizeRefusalTable/malformed_json_line`, `/missing_envelope_member`, `/bad_envelope_version`, `/string_envelope_version`, `/blank_line`, `/unknown_body_member`, `/call_live_followup_member`, `/result_live_attestation_member`, `/body_text_number`, `/body_live_attestation_string`, `/body_missing_text`, `/empty_directive`, `/contradictory_protection`, `/protected_plaintext_body`, `/unknown_protected_plaintext_body`, `/unknown_type_without_body`, `/lone_surrogate_text`, `/lone_surrogate_envelope`, `/duplicate_envelope_member`, `/duplicate_body_member`, `/trailing_bytes`, `/invalid_utf8_line`, `/empty_subagent_name` |
| Reasoning type/protection agreement refuses per arm | `TestNormalizeRefusalTable/encrypted_type_signed_protection`, `/signed_type_encrypted_protection` |
| Live claims by captured history refuse | `TestNormalizeRefusalTable/live_attestation_claim`, `/live_followup_claim` |
| Ambiguous tool pairings refuse | `TestNormalizeRefusalTable/duplicate_call_id`, `/duplicate_result_id` |
| Exact actor mapping (referenced iff mapped) | `TestNormalizeRefusalTable/unmapped_actor`, `/unmapped_extra_actor`, `/main_actor_in_map` |
| Instruction authority, usage value model, and result status vocabulary | `TestNormalizeRefusalTable/bad_instruction_authority`, `/fractional_usage`, `/unsafe_usage_magnitude`, `/usage_ledger_overflow`, `/usage_ledger_overflow_output`, `/string_usage_number`, `/bad_result_status` |
| Non-JSONL included members refuse (stated bound) | `TestNormalizeNonJSONLIncludedMemberRefuses` |
| Blob Descriptor reference resolves to installed bytes | `TestNormalizeInlineContentBoundary` (`mustResolveOverflow`) |
| Fsync + verify + atomic install before referencing | `TestNormalizeIsIdempotentIntoSharedSink` |
| Large data by reference at the content level (discharges no 7.8 clause) | `TestNormalizeInlineContentBoundary` |
| String measure in characters | `TestNormalizeMultibyteStringMeasure`, `TestNormalizeRefusalTable/long_subagent_name` (per-edge directive/call/tool-name rows are stated bounds) |
| Text is valid UTF-8, never rewritten | `TestNormalizeRefusalTable/lone_surrogate_text`, `/lone_surrogate_envelope`, `/duplicate_envelope_member`, `/duplicate_body_member`, `/trailing_bytes`, `/invalid_utf8_line` (gate: `environ.DecodeStrictObject` at `strictMembers`); `TestNormalizeCaseAliasEnvelopeIgnoresUnclaimedMembers/body_alias`, `/evidence_alias` (no second decoder; values from the strict map by exact name) |
| Case-folded aliases never override claimed members | `TestNormalizeCaseAliasEnvelopeIgnoresUnclaimedMembers` (8 subtests: 6 axes + unrelated-extra control) |
| Unprotected reasoning summaries project internal | `TestNormalizeReasoningSummaryProjectsInternal` |
| Every emitted kind stamps capture status exact | `TestNormalizeCaptureStatusExactForEveryKind` |
| The native instruction fold is last-wins | `TestNormalizeLastNativeInstructionWins` |
| The external selector seals as an external actor | `TestNormalizeExternalActorSealsAsExternal` |
| Null text seals as empty inline content (stated bound) | `TestNormalizeNullTextDecodesAsEmpty` |
| Null optional booleans decode as false (stated bound) | `TestNormalizeNullOptionalBoolsDecodeAsFalse` |
| Present-but-null body on an unknown type projects opaque (stated bound) | `TestNormalizeUnknownTypeNullBodyProjectsOpaque` |

## D. Stated bounds and NOT APPLICABLE (with owners)

| Area | Status | Owner / evidence |
|---|---|---|
| Fixture-native record envelope (`v`, `native_event_id`, `native_type`, `origin`, `protection`, `actor`, `body`) | Stated bound: test-defined framing, not a specification shape; body shapes exact; envelope extras unclaimed and unable to override a claimed member (strict-map read by exact name, no second decoder — pinned by `TestNormalizeCaseAliasEnvelopeIgnoresUnclaimedMembers`) | this leaf (`doc.go`, `native.go` + refusal rows) |
| Canonical kinds never emitted here (`session_started`, `approval_request`, `plan_update`, `progress`, `rate_limit`, `file_change`, `compaction`, `turn_*`, `subagent_*`, `error`, `migration_checkpoint`, `session_finished`) | NOT APPLICABLE: fact registries cover exactly the ten emitted kinds | normalization-sibling scope |
| Non-text content blocks (`json`, `image`, `audio`, `document`, `resource_link`, `redacted`, `opaque`) | NOT APPLICABLE: this leaf seals `text` blocks only (inline or blob-referenced) | normalization-sibling scope; closed vocabulary enforced by the landed builder |
| Capture statuses `partial`, `synthesized`, `unavailable` | NOT APPLICABLE: this leaf emits `exact` only | shape rules owned by the accepted `internal/clonebundle` suite |
| Closed-shape internals (26-kind registry, 0..64 parents sorted-unique, UUIDv7, turn/time nullability, extensions) | Shape rules owned by the `internal/clonebundle` suite; this leaf proves its output satisfies them by sealing + re-decoding every event and session (builder refusal = projection error) | `internal/clonebundle` (leaf 1, accepted) |
| Session/event scale far edges (parents 65, heads 1025, actors 1025, event IDs 1000001 — each asserted with code + message at build AND decode) | Driven, not delegated: `TestCanonicalSessionRefusals/build_actors/too_many_actors` + `N-actors-max-build`, `/build_envelope/too_many_events` + `N-eventids-max-build`, `/build_envelope/too_many_heads` + `N-heads-max-build`, `/decode/too_many_actors` + `N-actors-max-decode`, `/decode/too_many_events` + `N-eventids-max-decode`, `/decode/too_many_heads` + `N-heads-max-decode`, `TestCanonicalEventRefusals/build_envelope/too_many_parents` + `N-parents-max-build`, `/decode/too_many_parents` + `N-parents-max-decode`; build-side plants kill by admission (`error = <nil>`), decode-side by the self-digest message | this leaf (rows committed in `internal/clonebundle/refusal_session_event_test.go`) + `internal/clonebundle/TRACEABILITY.md` clause map |
| Decode-side distinct unsorted `raw_refs` | Driven: `TestCanonicalEventRefusals/decode/evidence_raw_refs_unsorted` + `N-rawrefs-unsorted-decode` (complements `N-rawrefs-dup-decode`, which admits only the equal duplicate) | this leaf (row committed in `internal/clonebundle/refusal_session_event_test.go`) |
| Actor UUIDv7 parsing | Gate owned by `internal/scalar`; entry refusal pinned by `TestNormalizeRefusalTable/bad_actor_uuid` | `internal/scalar` (landed) |
| `parent_actor_id` shape rule (null iff main) | Enforced by the landed builder; main-null asserted in `TestNormalizeSealsSessionShape`, subagent-parent in `nested_subagent_call`, wiring narrowed by `N-main-parent` | `internal/clonebundle` builder + this leaf |
| Closed core reason vocabulary (`unknown_native_event`, `foreign_encrypted_payload`, `foreign_signature_unverifiable`, `unsafe_pending_action`, …) | Referenced, not owned: defined by §13.14.2; this leaf emits four codes as evidence strings without implementing dispositions | §13.14.2 owner (sibling) |
| §13.14.2 fidelity dispositions, projection planning, lineage | NOT APPLICABLE: sibling scope | §13.14.2 owner (sibling) |
| §13.14.3 Migration Checkpoint / bundle chain | NOT APPLICABLE: sibling scope | §13.14.3 owner (sibling) |
| §13.14.4 transaction / staging | NOT APPLICABLE: this Story plans no projection, stages no target | §13.14.4 owner (sibling) |
| §13.14.5 tuple admission | NOT APPLICABLE: this Story admits no tuple | §13.14.5 owner (sibling) |
| §7.8 operation frames (capture-plan/capture/normalize) and the 8 MiB frame bound | NOT APPLICABLE here: owned and discharged by `sessadapter` (`session-adapter-frame-bound` case); this leaf owns the fidelity rules those operations must satisfy | `internal/sessadapter` (landed) |
| Per-edge character-width rows past the envelope ID edge (native_type 512/513, directive 4096/4097, call ID 512/513, tool names) | Stated bound: every edge checks through the landed `environ.StringLength` gate, but only the envelope ID edge carries a narrowing (`N-string-bound`); the empty-directive low edge is pinned, not deferred (`empty_directive` + `N-directive-empty`) | this leaf (deferred rows + one pinned edge) |
| Non-JSONL included members | Stated bound, pinned: every included non-unknown member parses as the JSONL log; routing by member class is sibling scope | this leaf (`TestNormalizeNonJSONLIncludedMemberRefuses`) |
| `directives:null` | Stated bound, pinned: decodes as zero directives | this leaf (`TestNormalizeNullDirectivesDecodeAsEmpty`) |
| `text:null` / `ciphertext:null` | Stated bound, pinned: null text seals as empty inline content (`TestNormalizeNullTextDecodesAsEmpty`); null ciphertext admitted but harmless (value never read) | this leaf |
| `live_followup:null` / `live_attestation:null` | Stated bound, pinned: null decodes as false exactly like an absent member (`TestNormalizeNullOptionalBoolsDecodeAsFalse`); neither null claim refuses nor promotes | this leaf |
| Present-but-null body on unknown types | Stated bound, pinned: null satisfies the required body member and the record projects opaque (`TestNormalizeUnknownTypeNullBodyProjectsOpaque`); the body is never read; a missing body refuses | this leaf |
| CRLF members | Stated bound, pinned: admitted with each trailing CR inside its record range; the range covers the line plus the CR and resolves exactly (`TestNormalizeCRLFResolvesWithCarriageReturn`; `splitRecordLines` doc comment) | this leaf |
| Empty `subagent:` selector | Pinned message pin: refuses at the shape gate with the shape message (`TestNormalizeRefusalTable/empty_subagent_name`); a plant admitting it past the shape arm still refuses at the actor mapping (`N-actor-empty-subagent-name`) | this leaf |
| Per-type body registries past the shared loop | The shared exact-shape loop is narrowed by `N-body-member`; the call and result registries carry their own narrowings (`call_live_followup_member` + `N-allowed-call-live-followup`, `result_live_attestation_member` + `N-allowed-result-live-attestation`); the message, protected, instruction, definition, and usage registries are covered by the shared-loop narrowing (seven-site census in the results) | this leaf |
| Refusal message cosmetics (actor wrap shape; clonebundle `session.go:527` helper wording for event parents) | Declined cosmetic: the uniform `record <member> line <n>` + package-prefix shape is AC-pinned (member+line naming); the clonebundle helper is accepted sibling production with verified-correct behavior | this leaf (declined) / `internal/clonebundle` (owner) |
| Durable-write crash semantics (stage/fsync/verify/atomic install) | Owned by `internal/localstore` (landed fault suites); this leaf adds bundle atomicity (row 24) | `internal/localstore` (landed) |
| `pending` live-work-order map parameter of `deriveLiveSurface` | Live-sibling API: `Normalize` always passes nil, so nothing captured can pend | live sibling (stated bound) |
| Inherited advisories P3-a, P3-b, P3-c/d, P3-f, P3-g (capture-operation rows) | DEFERRED to a follow-up on `internal/clonesnap` (orchestrator-routed); this leaf changes no clonesnap behavior | clonesnap follow-up |
| Inherited P3-e AfterWalk-seam bound | CLOSED here as a stated bound (`internal/clonesnap/TRACEABILITY.md` + results) | this leaf |
| Inherited P3-η `EncodeNativeIdentity` standalone contract | DEFERRED to a follow-up on the `internal/clonebundle` owner | clonebundle owner follow-up |
| Inherited P3-ζ′ fidelity vocabulary spelled twice | BOUND: both spellings named with pinning tests; unification deferred to a follow-up on `internal/sessadapter` + `internal/clonesnap` | sessadapter/clonesnap follow-up |
| Inherited P3-ζ lone surrogate | CLOSED here at the projection entry (rows 25–26 + narrowings) | this leaf |
| `ax` command, `doctor` result, runtime capability | NOT CLAIMED anywhere in code, tests, README, or LOGBOOK | — |

## E. Negative-evidence index

Every refusal below is driven through `Normalize` over real captured bytes
by the named committed test and reddened by the named narrowing (single
member admitted). Production call sites: `Normalize` (`normalize.go`),
`collectMembers` + `sealedEnvironment` (closure), `strictMembers` +
`parseNativeLine` + `splitRecordLines` + body decoders (`native.go`),
`dispatchRecord` / `dispatchProtected` (`dispatch.go`), `resolveToolCalls` +
`deriveLiveSurface` (`tools.go`), `checkActors` + `buildEvents` +
`buildResult` (`dispatch.go`).

Refusal rows (54 in `TestNormalizeRefusalTable` + 3 standalone:
`TestNormalizeRefusesUnstableCapture`,
`TestNormalizeRefusesEnvironmentDrift`,
`TestNormalizeNonJSONLIncludedMemberRefuses`): see section
A rows 11, 13, 17, 23, 25–29, 31, 33–37, 39, 44 and section C. Token-preserving attacks:
`N-native-type-token` (the `message/user` label still matches, routes to
`assistant_message`; killer `TestNormalizeSealsSessionShape` executes the
behavioral suite) and `N-protection-encrypted` (the `encrypted` label still
matches, skips opaque dispatch; killer `TestNormalizeForeignEncryptedReasoningStaysOpaque`).
The six `N-alias-*` rows restore the case-folded read for exactly one
envelope member each (killers: the six axis subtests of
`TestNormalizeCaseAliasEnvelopeIgnoresUnclaimedMembers`).
Labelled non-narrowings: `N-empty-projection` (single-member class),
`N-overflow-truncate` (behaviour swap), `N-pending-aborted`
(add-arm: `Normalize` passes nil live work orders),
`N-offset-newline` (behaviour swap: drops the newline stride), and
`N-unterminated-tail-drop` (behaviour drop: drops the unterminated
last line of a multi-line member).
Harmless control: `C-doc-comment` (comment-only edit, SURVIVED exit 0 both
runs), proving the harness observes outcomes instead of hard-coding kills.
