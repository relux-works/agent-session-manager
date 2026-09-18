package cloneproject

// This package implements the Section 13.14.1 projection-fidelity
// closure: Normalize derives a Canonical Session and Canonical Events
// from captured bytes (a sealed Clone Capture Manifest, its sealed
// Clone Raw Object Manifest, and the installed payload blobs). It is
// the final leaf of the canonical-session-capture-and-evidence
// story: contracts come from internal/clonebundle, capture from
// internal/clonesnap, and this package owns normalization only.
//
// Authority: relux-works/agent-session-manager-spec@v0.7.0, Sections
// 7.8, 10.2, and 13.14.1 (internal/specdoc/SPEC.v0.7.0.md).
//
// The projection input is captured bytes, never hand-written
// Canonical Events: every proof builds a fixture provider store,
// runs clonesnap.Capture over it, and normalizes the sealed output.
// Identical captured bytes produce byte-identical canonical output.
//
// Native records: the provider-native session log this package reads
// is newline-delimited JSON (one record per line) carried by included
// durable capture members. The envelope is fixture-native and NOT a
// specification shape:
//
//	{"v":1,"native_event_id":"...","native_type":"...",
//	 "origin":"native|foreign","protection":"none|encrypted|signed",
//	 "actor":"main|subagent:<name>|external","body":{...}}
//
// Projection rules (each driven and mutated, see TRACEABILITY.md):
//
//   - Unknown native types become raw-addressable opaque_event
//     records with opaque visibility, a whole-record byte-range
//     reference, and reason unknown_native_event. Unknown-class
//     members become one whole-member opaque_event. Nothing
//     attributable is dropped, guessed, or re-typed.
//   - Foreign encrypted/signed reasoning becomes opaque_reasoning
//     with opaque visibility, preserved byte-exact through the blob
//     store, never decrypted, re-encoded, summarized, truncated, or
//     promoted. Foreign instructions are low-authority history: they
//     never change the effective instruction snapshot.
//   - Historical tools are inert. A tool_definition_snapshot never
//     registers a callable tool (a captured live claim refuses).
//     A tool_call resolves completed only with its tool_result;
//     otherwise it becomes aborted history with reason
//     unsafe_pending_action. Pending actions require a live work
//     order, which capture never carries, so the live surface is
//     always empty.
//   - Source usage is counted into the source ledger only; target
//     accounting stays zero.
//   - Message-like text over 64 KiB becomes a Blob Descriptor
//     reference to an installed overflow blob, never a truncation.
//
// Reuse, not fork: manifests, sessions, events, evidence, and the
// unstable/maximal-safe gates come from internal/clonebundle;
// overflow Blob Descriptors seal through clonesnap.BuildBlobDescriptor
// and install through clonebundle.InstallRawBlob onto the landed
// localstore no-replace discipline; digests and UUIDv7 come from
// internal/scalar; strict JSON framing, character measures, and the
// uint53 value gate come from internal/environ (DecodeStrictObject,
// StringLength, CheckUint53Bounds). No rule is reimplemented where
// a landed owner exists.
//
// Stated bounds (not waived): the fixture-native record envelope is
// test-defined, not a specification shape; envelope extras are
// unclaimed and can never override a claimed member (the envelope
// is read from the strict map by exact name with no second
// decoder); per-kind payload fact registries cover exactly the
// facts this package emits for the ten fixture-native known types
// (other Canonical kinds are never emitted here);
// framing-malformed lines (unparseable JSON, lone surrogate
// escapes, duplicate members, trailing data, missing envelope
// members, contradictory protection claims, unknown body members)
// refuse instead of projecting; every included non-unknown member
// is parsed as the JSONL record log, so a non-JSONL member refuses
// the whole projection (routing by member class is sibling scope);
// per-edge character-width rows past the envelope ID edge
// (native_type, directives, call IDs, tool names) are deferred
// (the shared StringLength gate is landed), except the empty
// directive, which refuses with its index named; "directives":null
// decodes as zero directives and "text":null seals as empty inline
// content; "live_followup":null and "live_attestation":null decode
// as false, exactly like an absent member; present-but-null
// satisfies the required body member for unknown types (the body
// is never read) while a missing body refuses; turn IDs and source
// timestamps are always null; tool call arguments are not carried;
// unknown-class excluded members project nothing (their bytes were
// never captured); 13.14.2-13.14.5 (fidelity dispositions,
// projection planning, transaction, lineage) are sibling scope.
