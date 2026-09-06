// Package sessadapter implements the ax host side of the companion
// Session Adapter protocol of pinned specification Section 7.8: the
// closed operation registry, envelope framing, manifest and probe
// validation, call-context and authority gates, resource limits,
// request-digest binding, discovery binding, and Environment Tuple
// admission.
//
// The actor every Section 7.8 RFC 2119 keyword governs here is the ax
// host (this repository). The session adapter is a child surface this
// package never trusts: it runs in the same trusted
// ax-provider-<id> executable as Provider Protocol 2.0.0 and is not a
// separately discovered plugin, so discovery here means binding the
// adapter to the independently observed provider candidate — the same
// executable path, the same host-observed executable digest, the same
// provider identifier, and the same manifest digest — before every
// call and every target mutation. Every byte the adapter emits is
// attacker-influenced, so every frame is validated before any member
// is used.
//
// This package owns the closed-object decoders and the fail-closed
// gates. Operation dispatch means the host maps an operation name to
// one adapter invocation and refuses unknown names locally with
// operation_unknown; an unavailable operation is refused with
// capability_unavailable. A failed read is never an absence: a
// partial, malformed, over-limit, or escaped result is a
// session_adapter_protocol_error, never an empty result and never
// fallback permission.
//
// The package mutates no durable state of its own and keeps no
// cross-call cache: adapter mutations address fresh isolated sinks
// the core creates per call, and candidate objects are addressed only
// by their exact *_candidate_id in the request's sink. The core
// retrieves, validates, rehashes, and seals them. Retry receipts live
// in the core and the sink, not in the host: Section 7.8 defines no
// (operation, operation_id) retry key — the pinned catalog carries an
// empty mutation-group list for the session_adapter family — so the
// idempotency this package enforces is the request-digest binding
// (the adapter answers for exactly the canonical body the host sent)
// and the byte-for-byte context echo, plus the fresh-sink authority
// rule that keeps one call's writes out of every other call's sink.
// Cross-crash durability of adapter results is the core document's
// property, the same division the provider host records for its
// transaction document.
//
// Stated bounds: transport process management (spawning the provider
// executable, deadlines, stdio ownership) belongs to the caller, which
// supplies frames this package validates as bytes; nested objects
// owned by Sections 13.14 and 10.8 (CaptureBoundary, NativeIdentity,
// StableSnapshotProof, CloneSourceSummary, FidelityCounts,
// WorkspaceBinding, and the remaining clone-plan types) are checked
// for presence and object shape only, never for content — their
// content owners are the directory-node and boundary leaves of this
// Story, and inventing their validation here would collide with that
// ownership at integration. The tuple-registry signature, sequence
// monotonicity, and publication rules of Section 13.14 fail closed at
// the registry owner, which does not exist yet; this package admits
// single entries against caller-held registry facts. Resume-smoke
// family and secret checks on resume-plan argv rest on
// ResumeSmokeEvidence (native_cli_family), which lives in the same
// unowned registry scope: this package enforces the exact-once
// identity occurrence and the argv shape, not the family name or the
// absence of secrets.
package sessadapter
