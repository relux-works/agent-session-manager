# Projection Fidelity Clause Coverage (TASK-260830-32ypr2)

Owner: `internal/cloneproject`. Authority:
`internal/specdoc/SPEC.v0.7.0.md`, Sections 7.8, 10.2, 13.14.1
(the pinned sections are byte-identical to v0.6.0).
This is the FINAL leaf's clause record for the projection-fidelity
closure; the `internal/traceability` registry bindings and re-pin
ship in this leaf as well (see the Story-close section below).

## Why this package normalizes instead of clonebundle or clonesnap

The first leaf (`internal/clonebundle`) owns the capture contracts:
the closed manifest/boundary/session/event shapes, their builders
and decoders. The second leaf (`internal/clonesnap`) owns the
capture OPERATION over a real provider store. This package owns
NORMALIZATION: deriving Canonical Sessions and Canonical Events
from captured bytes. Every event and the session seal through the
landed `clonebundle` builders (`BuildCanonicalEvent`,
`BuildCanonicalSession`, `CheckOrdinalContiguity`,
`RefuseUnstableForTarget`, `InstallRawBlob`) and re-decode through
the landed decoders before return; overflow Blob Descriptors seal
through `clonesnap.BuildBlobDescriptor`; no second manifest,
session, event, blob, or identity model exists here.

## Landed-owner reuse

| Scope | Owner | Use |
|---|---|---|
| Session/event/evidence shapes | `internal/clonebundle` | all Build/Decode/Check/Refuse entries above |
| Overflow descriptor shape | `internal/clonesnap` | `BuildBlobDescriptor` (errors wrapped with the overflow cause) |
| Blob install (no-replace, fsync, verify) | `internal/clonebundle` + `internal/localstore` | `InstallRawBlob`, `ObjectStore.PutBlob` |
| Digests, UUIDv7, timestamps | `internal/scalar` | `ParseDigest`, `SHA256Digest`, `ParseUUIDv7` |
| Character string measure | `internal/environ` | `StringLength` at every native bound |
| Strict JSON framing | `internal/environ` | `DecodeStrictObject` at `strictMembers`: both frame sites (record envelope, per-kind bodies) refuse lone-surrogate escapes, duplicate members, and trailing data before any member is read; the envelope is then read from the admitted map by exact member name with no second decoder, so case-folded aliases are inert extras (pinned by `TestNormalizeCaseAliasEnvelopeIgnoresUnclaimedMembers`) |
| uint53 value gate | `internal/environ` | `CheckUint53Bounds` at `bodyUint53` and the envelope version check: fractions, exponents, strings, and out-of-range magnitudes refuse as not-a-uint53 |
| Canonical bytes | `internal/canonicaljson` | `Canonicalize`, `CalculateObjectIdentity` (tests resolve overflow descriptors) |

## Fixture-native per-kind payload fact registry

Section 13.14.1 requires payloads to carry only their registered
typed facts. The envelope and value model are landed (`checkPayload`
in `clonebundle`); this package registers the exact facts it emits
for each kind it projects. No other Canonical kind is emitted here.

| Kind | Exact facts |
|---|---|
| `user_message`, `assistant_message`, `reasoning_summary` | `content_blocks` (one `text` block, inline at or under 64 KiB, else `blob_descriptor_id` + `media_type`), `extensions` |
| `opaque_reasoning` | `protection` (`encrypted`\|`signed`), `byte_count`, `blob_descriptor_id`, `extensions` |
| `opaque_event` (unknown record) | `native_type`, `byte_count`, `blob_descriptor_id`, `extensions` |
| `opaque_event` (protected known record) | `native_type`, `protection`, `byte_count`, `blob_descriptor_id`, `extensions` |
| `opaque_event` (unknown member) | `native_item_key`, `byte_count`, `blob_descriptor_id`, `extensions` |
| `instruction_snapshot` | `authority` (`low` forced for foreign), `directives`, `extensions` |
| `tool_definition_snapshot` | `tool_name`, `definition_digest`, `callable` (always false), `extensions` |
| `tool_call` | `call_id`, `tool_name`, `resolution` (`completed`\|`aborted`), `extensions` |
| `tool_result` | `call_id`, `status` (`ok`\|`error`), `resolution`, `extensions` |
| `usage` | `input_tokens`, `output_tokens`, `extensions` |

## Clause map

| Pinned clause | Production entry | Test |
|---|---|---|
| 13.14.1 unknown native records become raw-addressable opaque events | `Normalize`, `dispatchRecord` | `TestNormalizeUnknownRecordBecomesOpaqueEvent` (kind, `opaque` visibility, `unknown_native_event` reason, raw range resolves to the exact line); `TestNormalizeMultiRecordRefsResolveAtAssertedOffsets` (every event's `RawRefs[0]` resolves to its own line with offsets `0`, `len(l1)+1`, `len(l1)+len(l2)+2` asserted, multibyte first line); `TestNormalizeUnterminatedTailResolvesWithOffset` (a multi-line member without a trailing newline keeps its last record with its offset asserted); `TestNormalizeCRLFResolvesWithCarriageReturn` (each CRLF range covers its line plus the CR and resolves exactly) |
| 13.14.1 almost-known records are never coerced into a neighbouring kind | `dispatchRecord` (exact type switch) | `TestNormalizeAlmostKnownRecordBecomesOpaqueEvent` (`message/smoke` stays `opaque_event`, never `user_message`) |
| 13.14.1 unknown-class members become whole-member opaque events | `collectMembers` | `TestNormalizeUnknownMemberBecomesWholeMemberOpaque` (offset 0, full length, null native IDs, resolves to the whole member) |
| 13.14.1 opaque events are never dropped, guessed, or re-typed | `Normalize` (deterministic order), `parseNativeLine` (strict-map envelope) | `TestNormalizeOpaqueIsStableAcrossProjections` (project twice, byte-identical, still `opaque_event`); `TestNormalizeCaseAliasEnvelopeIgnoresUnclaimedMembers/kind_alias_after` + `/kind_alias_before` (a `NATIVE_TYPE` alias in either document order still projects `opaque_event`, never coerced into a neighbouring kind) |
| 13.14.1 foreign encrypted reasoning is opaque-preserved | `dispatchProtected` | `TestNormalizeForeignEncryptedReasoningStaysOpaque` (`opaque_reasoning`, `foreign_encrypted_payload`, no content members, byte-exact round trip) |
| 13.14.1 foreign signed reasoning is opaque-preserved | `dispatchProtected` | `TestNormalizeForeignSignedReasoningStaysOpaque` (`foreign_signature_unverifiable`, byte-exact) |
| 13.14.1 native protected reasoning stays opaque with no foreign code | `dispatchProtected`, `cryptoReasons` | `TestNormalizeNativeProtectedReasoningStaysOpaque` (empty reasons) |
| 13.14.1 protected known content never reaches a semantic kind | `dispatchProtected` | `TestNormalizeProtectedMessageBecomesOpaqueEvent` (encrypted assistant message is `opaque_event`) |
| 13.14.1 unknown+protected records carry both stable reasons | `dispatchProtected` | `TestNormalizeUnknownProtectedRecordCarriesBothReasons` |
| 13.14.1 foreign instructions are low-authority history | `dispatchInstruction`, `foldInstructions` | `TestNormalizeForeignInstructionCannotChangeEffectiveSnapshot` (effective snapshot native-only, foreign payload pins `low`), `TestNormalizeForeignOnlyInstructionLeavesSnapshotEmpty`, `TestNormalizeCaseAliasEnvelopeIgnoresUnclaimedMembers/origin_alias` (an `Origin` alias cannot promote foreign history into the snapshot) |
| 13.14.1 foreign encrypted reasoning is never promoted by an alias | `parseNativeLine` (strict-map envelope), `dispatchProtected` | `TestNormalizeCaseAliasEnvelopeIgnoresUnclaimedMembers/protection_alias` (a `Protection` alias cannot downgrade the record: the plaintext body refuses) |
| 13.14.1 attribution follows the claimed actor member, never an alias | `parseNativeLine` (strict-map envelope), `checkActors` | `TestNormalizeCaseAliasEnvelopeIgnoresUnclaimedMembers/actor_alias` (an `Actor` alias cannot re-attribute an unmapped subagent record to `main`) |
| 13.14.1 unprotected reasoning summaries project internal | `dispatchMessage` (default arm) | `TestNormalizeReasoningSummaryProjectsInternal` (kind `reasoning_summary`, `internal`, content block, `exact`) |
| 13.14.1 every emitted kind stamps capture status exact | `evidenceFor` | `TestNormalizeCaptureStatusExactForEveryKind` (one record per kind, every event asserts `exact`) |
| 13.14.1 the native instruction fold is last-wins | `foldInstructions` | `TestNormalizeLastNativeInstructionWins` (two native records, the snapshot carries the second) |
| 13.14.1 the external selector seals as an external actor | `buildResult` | `TestNormalizeExternalActorSealsAsExternal` (kind `external`, parented on `main`) |
| 13.14.1 source usage is not target accounting | `foldUsage` | `TestNormalizeSourceUsageIsNotTargetAccounting` (source sums exact, target zero) |
| 13.14.1 historical tools are inert; call+result resolves completed | `resolveToolCalls`, `deriveLiveSurface` | `TestNormalizeToolMatrix/call_with_result` (kinds, `internal` visibility, `completed`, empty live surface) |
| 13.14.1 incomplete calls become aborted history and block pending action | `resolveToolCalls`, `deriveLiveSurface` | `TestNormalizeToolMatrix/call_without_result` (`aborted`, `unsafe_pending_action`, empty live surface), `TestNormalizeToolMatrix/result_without_call` (orphan `aborted`) |
| 13.14.1 results arriving after the boundary complete the same call | `resolveToolCalls` over two captures | `TestNormalizeResultAfterBoundaryCompletes` (first capture `aborted`, second `completed`) |
| 13.14.1 nested calls attribute to a parented subagent actor | `checkActors`, `buildResult` | `TestNormalizeToolMatrix/nested_subagent_call` (subagent actor, parent is main) |
| 13.14.1 definitions never register a callable tool | `deriveLiveSurface`, `dispatchDefinition` | `TestNormalizeToolDefinitionNeverRegistersCallable` (`callable` false, `CallableTools` empty) |
| 13.14.1 live surface stays empty over mixed history | `deriveLiveSurface` | `TestNormalizeLiveSurfaceStaysEmpty` |
| 13.14.1 Canonical Session bounds (actors, ordered event IDs, head) | `buildResult` | `TestNormalizeSealsSessionShape` (ordinals, parent chain, head is last) |
| 13.14.1 message-like inline content at most 64 KiB; oversized becomes a blob reference | `contentBlockForText`, `installOverflow` | `TestNormalizeInlineContentBoundary` (65536 inline, 65537 blob-referenced and byte-exact, never truncated) |
| 13.14.1 same captured bytes project byte-identical output | `Normalize` (sorted inputs, no clock), `buildResult` actor sort | `TestNormalizeIsDeterministic`, `TestNormalizeActorOrderIsDeterministic` (three extra actors, twelve projections, one distinct session) |
| 13.14.1 unstable_archive cannot enter target projection | `Normalize` via `RefuseUnstableForTarget` | `TestNormalizeRefusesUnstableCapture` |
| 13.14.1 capture closure: manifest link, blob fetch, digest/size agreement | `Normalize`, `collectMembers` | `TestNormalizeRefusalTable/manifest_closure_drift/tampered_blob/missing_blob/blob_size_drift` (each arm asserts its own message) |
| 13.14.1 framing-malformed records refuse with member+line named | `strictMembers`, `splitRecordLines`, `parseNativeLine`, body decoders | `TestNormalizeRefusalTable/malformed_json_line/missing_envelope_member/bad_envelope_version/string_envelope_version/blank_line/unknown_body_member/call_live_followup_member/result_live_attestation_member/body_text_number/body_live_attestation_string/body_missing_text/empty_directive/contradictory_protection/protected_plaintext_body/unknown_protected_plaintext_body/unknown_type_without_body/lone_surrogate_text/lone_surrogate_envelope/duplicate_envelope_member/duplicate_body_member/trailing_bytes/invalid_utf8_line/empty_subagent_name` |
| 13.14.1 live claims by captured history refuse | `deriveLiveSurface`, `resolveToolCalls` | `TestNormalizeRefusalTable/live_attestation_claim/live_followup_claim` |
| 13.14.1 ambiguous tool pairings refuse | `resolveToolCalls` | `TestNormalizeRefusalTable/duplicate_call_id/duplicate_result_id` |
| 13.14.1 exact actor mapping (referenced iff mapped) | `checkActors` | `TestNormalizeRefusalTable/unmapped_actor/unmapped_extra_actor/main_actor_in_map` |
| 13.14.1 instruction authority, usage value model, and result status vocabulary | `decodeInstructionBody`, `bodyUint53`, `decodeToolResultBody` | `TestNormalizeRefusalTable/bad_instruction_authority/fractional_usage/unsafe_usage_magnitude/string_usage_number/bad_result_status` |
| 13.14.1 reasoning type/protection agreement refuses per arm | `dispatchProtected` | `TestNormalizeRefusalTable/encrypted_type_signed_protection/signed_type_encrypted_protection` |
| 13.14.1 protected non-reasoning tool records reach opaque only via the fold-skip | `foldTools` skip, `dispatchProtected` | `TestNormalizeProtectedToolCallStaysOpaqueHistory` (protected call is `opaque_event`, the unprotected result an aborted orphan, live surface empty) |
| 13.14.1 non-JSONL included members refuse the whole projection | `collectMembers` JSONL parsing | `TestNormalizeNonJSONLIncludedMemberRefuses` (stated bound: routing by member class is sibling scope) |
| 10.2 Blob Descriptor reference resolves to installed bytes | `installOverflow` | `TestNormalizeInlineContentBoundary` (`mustResolveOverflow` through the sink) |
| 10.2 fsync + verify + atomic install before referencing | `InstallRawBlob`, `PutBlob` (landed) | `TestNormalizeIsIdempotentIntoSharedSink` (replay verifies and reuses, blob count stable) |
| 7.8 large data by reference, never embedded in the 8 MiB frame | `sessadapter.DecodeRequestFrame` (`MaxFrameBytes`) | `TestDecodeRequestFrameRefusesOversizeFrame`, `TestFrameBoundEdges` (sessadapter owner; `session-adapter-frame-bound` case). `TestNormalizeInlineContentBoundary` proves reference-not-embedded at the content level but constructs no frame and discharges no 7.8 clause |
| 1.6 string measure in characters | `environ.StringLength` at every native bound | `TestNormalizeMultibyteStringMeasure` (512-char ID admits, 513 refuses) and `TestNormalizeRefusalTable/long_subagent_name`; per-edge directive/call/tool-name rows are stated bounds |
| 1.6 text is valid UTF-8, never rewritten | `environ.DecodeStrictObject` at `strictMembers` (both frame sites) after line-level `utf8.Valid`; envelope values come from the strict map by exact name | `TestNormalizeRefusalTable/lone_surrogate_text/lone_surrogate_envelope/duplicate_envelope_member/duplicate_body_member/trailing_bytes/invalid_utf8_line`; `TestNormalizeCaseAliasEnvelopeIgnoresUnclaimedMembers/body_alias` (sealed content ignores the `Body` alias) + `/evidence_alias` (the evidence identity ignores the `Native_Event_Id` alias) |

## Durability

This package holds no local state. Its only durable write is
overflow-blob installation: payload bytes through the landed
`InstallRawBlob` (descriptor identity plus blob-ID/size agreement)
and descriptor bytes through `ObjectStore.PutBlob`, both
content-addressed under the landed no-replace discipline (stage,
fsync, verify, atomic install). Crash semantics are the landed
`localstore` discipline's (proven by that package's fault suites);
this package adds bundle atomicity at the return:
`TestNormalizeFailureReturnsNoPartialBundle` proves a mid-projection
fetch failure returns no partial bundle and a healthy re-run
converges, and `TestNormalizeIsIdempotentIntoSharedSink` proves a
replay into a shared sink seals byte-identical output while the sink
blob count stays stable (verify-and-reuse, no reinstall). A refused
projection may leave orphan overflow blobs from earlier units in the
sink; they are content-addressed and inert, and no bundle ever
references them.

## Mutants

`testdata/mutant_harness.py` ships 80 narrowing rows, 5 labelled
non-narrowing rows, and the harmless `C-doc-comment` SURVIVED
control. Every narrowing weakens one gate to admit exactly one
member of the class it must reject, and every killer drives the
production `Normalize` entry over real captured bytes.
`N-native-type-token` and `N-protection-encrypted` are the
token-preserving attacks: the searched-for text still matches
while the decision changes, and the killers execute the
behavioral suite. The six `N-alias-*` rows restore the
case-folded read for exactly one envelope member each; every
other alias stays an unclaimed extra under the plant.
`N-empty-projection` is labelled single-member class,
`N-overflow-truncate` is labelled behaviour swap,
`N-pending-aborted` is labelled add-arm (`Normalize` passes nil
live work orders, so the narrowed condition alone cannot pend
from the entry and the plant appends the call explicitly), and
`N-offset-newline` is labelled behaviour swap (drops the newline
stride from the offset accumulation; the asserted offsets
redden), and `N-unterminated-tail-drop` is labelled behaviour
drop (drops the unterminated last line of a multi-line member;
the missing tail event reddens); no other row is an arm-delete. The five `N-strict-*`
rows patch the
shared `strictMembers` body used by both frame sites (one gate,
two call sites); every other plant patches one site. The battery
runs twice with one raw log per plant per run carrying the
subprocess exit.

## Stated bounds (sibling scope, not waived)

- The fixture-native record envelope is test-defined framing, not a
  specification shape: `v`, `native_event_id`, `native_type`,
  `origin`, `protection`, `actor`, `body`. Real providers define
  their own native logs; the adapter layer is `sessadapter` scope.
- Per-kind payload facts cover exactly the registry above. Other
  Canonical kinds (`session_started`, `approval_request`,
  `plan_update`, `progress`, `rate_limit`, `file_change`,
  `compaction`, `turn_*`, `subagent_*`, `error`,
  `migration_checkpoint`, `session_finished`) are never emitted
  here; their fact registries are normalization-sibling scope.
- Framing-malformed lines refuse (unparseable JSON, lone
  surrogate escapes, duplicate members, trailing data, missing
  envelope members, blank lines, invalid UTF-8, contradictory
  protection claims, unknown body members inside a known type)
  instead of projecting: without attributable identity a line
  cannot become evidence. Type-level unknowns become
  `opaque_event`.
- Every included non-unknown member is parsed as the JSONL record
  log: a non-JSONL included member refuses the whole projection
  with member+line named (pinned by
  `TestNormalizeNonJSONLIncludedMemberRefuses`); routing by member
  class is sibling scope.
- Envelope extras are unclaimed and can never override a claimed
  member: the envelope is read from the strict map by exact name
  with no second decoder, and case-folded aliases are inert
  extras in any document order (pinned by
  `TestNormalizeCaseAliasEnvelopeIgnoresUnclaimedMembers`, whose
  control subtest keeps an unrelated extra admitted).
- Per-edge character-width rows past the envelope ID edge
  (native_type 512/513, directive 4096/4097, call ID 512/513,
  tool names) are stated bounds: every edge checks through the
  landed `environ.StringLength` gate, but only the envelope ID
  edge carries a narrowing (`N-string-bound`). The
  empty-directive low edge is pinned, not deferred: the
  `empty_directive` row refuses `directive[0]` with its index
  named, narrowed by `N-directive-empty`.
- `"directives":null` decodes as zero directives: null is not
  distinguished from an empty array at the instruction body.
- `"text":null` seals as empty inline content (pinned by
  `TestNormalizeNullTextDecodesAsEmpty`); `"ciphertext":null`
  is admitted but harmless (the value is never read, only the
  protected shape is checked).
- `"live_followup":null` and `"live_attestation":null` decode
  as false, exactly like an absent member (pinned by
  `TestNormalizeNullOptionalBoolsDecodeAsFalse`): neither null
  claim refuses nor promotes.
- Present-but-null satisfies the required body member for
  unknown types (pinned by
  `TestNormalizeUnknownTypeNullBodyProjectsOpaque`): the body
  is never read, so the record projects opaque; a missing body
  refuses (`unknown_type_without_body`).
- CRLF members are admitted with each trailing CR inside its
  record range (pinned by
  `TestNormalizeCRLFResolvesWithCarriageReturn`): the range
  covers the line plus the CR and resolves exactly.
- The empty `subagent:` selector refuses at the shape gate with
  the shape message (pinned by
  `TestNormalizeRefusalTable/empty_subagent_name`): the arm is a
  message pin, since a plant admitting it past the shape gate
  still refuses at the actor mapping (`N-actor-empty-subagent-name`).
- Native-protected records carry no reason code: the closed core
  reasons name foreign causes only, and extension reasons are
  operator scope. The payload `protection` fact carries the cause.
- Turn IDs and source timestamps are always null: turns are not
  reconstructed and source clocks are not trusted.
- Tool call arguments are not carried: calls are inert history
  identified by call ID and tool name.
- Unknown-class excluded members project nothing: their bytes were
  never captured.
- Sections 13.14.2-13.14.5 (fidelity dispositions, projection
  planning, Migration Checkpoint, lineage, transaction, tuple
  admission) are sibling scope: this story plans no projection,
  stages no target, and admits no tuple.
- Section 7.8 operation wiring (capture-plan/capture/normalize
  frames) stays owned by `sessadapter`; this package owns the
  fidelity rules those operations must satisfy, not the frames.
- The `pending` live-work-order map parameter of
  `deriveLiveSurface` is live-sibling API: `Normalize` always
  passes nil, so nothing captured can pend.

## Story-close (FINAL leaf)

This leaf binds the story's normative sections in
`internal/traceability/ownership.v0.7.0.json`:

- `section:7.8` is bound FULL (2/2) with a `sessadapter`
  production owner: clause `7.8#1` (large data by reference,
  never embedded in the 8 MiB frame) discharges through the new
  `session-adapter-frame-bound` case
  (`TestDecodeRequestFrameRefusesOversizeFrame` and
  `TestFrameBoundEdges`, both committed by the sessadapter
  owner); clause `7.8#2` (execution bindings equal freshly read
  facts) discharges through the new
  `session-adapter-call-binding` case (`TestCheckCallBinding`
  plus the zero-facts refusal test, both committed by the
  sessadapter owner). The projection overflow reference test
  constructs no adapter frame, so it discharges no 7.8 clause.
- `section:10.2` is bound FULL (5/5): `10.2#1` through
  `clone-capture-contracts`, `10.2#3`–`10.2#5` through
  `clone-native-capture`, and `10.2#2` through
  `clone-native-capture` (capture install),
  `clone-projection-fidelity` (overflow install and idempotent
  replay), and the landed `localstore-immutable-blob-install`
  (the fsync/verify/atomic-install owner). The binding owns the
  union of the eight landed trunk cases and the three clone
  cases.
- `section:13.14.1` carries no RFC 2119 keyword line the
  `tracecheck` scanner can see (its obligations are closed-shape
  "contains exactly" sentences), so no registry binding can
  enumerate it: it would measure `unmeasured`, which assigned-scope
  admission deliberately refuses. The story's 13.14.1 clause record
  is therefore the three package `TRACEABILITY.md` files plus the
  three leaf conformance matrices, executed by the three real
  suites bound as `clone-capture-contracts`, `clone-native-capture`,
  and `clone-projection-fidelity`.
- `reviewedOwnershipCanonicalSHA256` in
  `internal/traceability/traceability.go` is re-derived from the
  edited registry (derivation command and output in
  `TASK-260830-32ypr2_results.md`).
