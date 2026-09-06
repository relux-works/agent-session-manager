// Package dirnode implements the ax host side of the companion
// Directory Node protocol of pinned specification Section 7.9: the
// two-major bootstrap, envelope framing, manifest and probe
// validation, scan/inventory request framing with (operation,
// operation_id) idempotency, Session Directory Query framing of
// Section 10.8.5, façade binding checks, and Structured Error
// 1.2.0 refusals.
//
// The actor every Section 7.9 RFC 2119 keyword governs here is the ax
// host (this repository). The directory node is a child surface this
// package never trusts: it runs in the same per-environment
// implementation as Provider 2 and Session Adapter 1, and every byte
// it emits is attacker-influenced, so every frame is validated
// before any member is used. Neither major adds operations to
// Provider 2, executes Continuation Plans, or transports
// transcript/workspace bytes, and this host never asks for any of
// those through this package.
//
// Major negotiation is caller-driven enumeration, not a hello: the
// caller attempts its locally supported majors in strictly
// descending order with one manifest request each, each attempt on a
// fresh process. Only the exact downgrade tuple — one well-framed
// failure echoing the request, response schema_version 1.0.0,
// Structured Error 1.2 incompatible_protocol with exit 6 and
// retryable=false, then process exit 6 — authorizes the next lower
// attempt. Deciding that tuple is DecideBootstrapStep; launching and
// reaping processes belongs to the caller, which records each
// attempt's process token in a ProcessGuard so a returned or failed
// process is never reused for a lower major.
//
// A failed read is never an absence: a partial, malformed,
// over-limit, mislabeled, or escaped result is an
// adapter_protocol_violation or transport_failure, never an empty
// result and never fallback permission. A syntactically valid frame
// naming an unknown operation is operation_unknown, refused
// locally before any node surface is touched. Query framing
// defects are query_invalid.
//
// Stated bounds: transport process management (spawning the node
// executable, deadlines, stdio ownership, allowlist/SSH admission)
// belongs to the caller, which supplies frames this package
// validates as bytes; the content of nested observation and lineage
// types owned by Sections 10.8 and 13.14 (EnvironmentObservation,
// NativeSessionObservation, InventoryBatch, PreviewExcerpt payload
// meaning, EnrichmentJobRequest/Receipt chains, ManagementBinding
// content, RuntimeExpectation) is checked for presence and object
// shape only — the shared-environment leaf of this Story owns that
// content, and inventing its validation here would collide with that
// ownership at integration. The scan journal in this package is an
// in-memory (operation, operation_id) record with byte export and
// import so crash recovery is demonstrable; its durable placement is
// the core's property. Caller authentication (CallerContext is
// authenticated server-side, never accepted from an unverified body
// alone) belongs to the caller: this package validates the context
// shape. Section 10.8.5 prose names a query_cursor_mismatch code the
// pinned error registry does not register; cursor reuse after a
// bound change is refused here as query_invalid with a
// cursor-mismatch detail, and the divergence is recorded on the
// refusing call site, not hidden.
package dirnode
