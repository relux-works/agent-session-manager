// Package provhost implements the ax host side of the provider plugin
// JSON-over-stdio protocol of pinned specification Section 7.2: one-frame
// JSONL transport, deadlines, size limits, stdout/stderr separation,
// operation dispatch, and status recovery.
//
// The actor every Section 7.2 RFC 2119 keyword governs here is the ax host
// (this repository). The provider plugin is a child process this package
// starts but never trusts: every byte it emits is attacker-influenced, so
// every frame is validated before any member is used, and a different-major
// payload is never parsed far enough to trust its error code, retryable
// bit, details, or authority fields (Section 15.1).
//
// This package owns the envelope layer and the operation-layer
// conformance decoders. Operation dispatch means the host maps an
// operation name to one plugin invocation and refuses unknown names
// locally without spawning; the conformance decoders validate what a
// plugin returned: the closed manifest (7.3), probe and capability
// gate (7.4, 8.3), profile mapping (7.7), quiescence proof (7.6),
// SpawnPlan success bodies for launch and resume, the Provider
// Identity Record and identify-session result (5.5), and the single
// (operation, operation_id) mutation key (7.5). Full Section 7.5
// request-body vocabularies beyond these surfaces, and the
// materialize commit/rollback result bodies, cross this package
// opaquely: a body must still be a JSON object, and the
// materialize-status body this package interprets for recovery must
// satisfy the state/nullability/identity rules Section 7.5 states
// for ProviderTransactionStatus.
//
// A failed read is never an absence: a crashed, timed-out, oversized,
// unparsable, or correlation-mismatched invocation fails with a local
// Structured Error 1.0.0, never with a partial response the caller could
// mistake for a result. Status recovery is fail-closed the same way: an
// unknown or integrity-invalid transaction fails with integrity_failure
// and must be quarantined rather than represented as a successful status.
//
// The package mutates no durable state of its own and keeps no cross-call
// cache: Host starts one plugin process per operation, so successive calls
// are separate processes observing durable state through the authority the
// caller passes, which is the property the cross-process recovery cases
// rest on. Retry decisions belong to the caller; every local failure this
// package emits is non-retryable, because nothing observed at an unusable
// frame licenses a safe-retry claim.
//
// Stated bounds: stderr capture is capped at 1 MiB (an implementation
// resource bound, not a protocol limit); error details carry the failure
// class, member names, and the observed version string only — never
// environment or stderr content. Mutation receipts live in the
// provider transaction document and the plugin, not in the host: the
// host holds no cross-call cache, so cross-crash durability of
// (operation, operation_id) results is the document's property, and
// the host side pins the sole key shape, byte-identical retry bytes,
// and the mismatch surface. The Section 8 provider/platform cell
// values are evidence labels, not host behavior: the host pins the
// fail-closed direction (conditional, unsupported, and unknown are
// never usable) for every capability-by-status tuple rather than one
// cell value. Section 8.2 store-exclusion roots and the remaining
// Section 7.5 request vocabularies have no host production yet.
//
// Provider Protocol 3.0.0, which the v0.5.0 catalog pins alongside
// 2.0.0, has no implementation owner in this package: this host
// speaks major 2 only, classifies any recognizable 3.x envelope as
// incompatible_protocol without trusting its payload, and carries no
// Structured Error 1.3.0 binding or launch/resume descriptor carriage.
// A v3 plugin is therefore refused loudly rather than misread as v2;
// naming the v3 envelope owner is open work, not a silent scope.
package provhost
