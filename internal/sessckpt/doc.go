// Package sessckpt captures typed Checkpoint Record 1.0.0 closures over
// provider identity, workspace manifests, the task-board bundle, terminal
// evidence, and the source head (STORY-260830-2rqigd, TASK-260830-14yo67).
//
// Authority: relux-works/agent-session-manager-spec@v0.6.0, Sections 5.4
// (Checkpoint Record), 10.5-10.6 (Materialization Plan/Journal closure
// inputs), and 13.12-13.13 (failure matrix and crash/restart outcome
// gate). The task scope cites the same section numbers of v0.5.0; the
// headings are retained in v0.6.0.
//
// The package owns capture and durable install only. It reuses the
// landed shared owners and never duplicates them:
//
//   - internal/canonicaljson owns the closed Checkpoint Record shape
//     and the canonical omit-self digest (checkpoint_id).
//   - internal/sessrepo owns record attestation
//     (AttestCheckpointRecord) and the event chain the source head
//     binds against (ListEvents).
//   - internal/sessquery owns semantic admission (admitCheckpoint:
//     owning lease tuple, creator-holder, persistence variant, event
//     heads at or before the owning lease). Capture pre-checks the
//     variant and head binding it can decide from its inputs so an
//     unpublishable closure is refused before publication, but the
//     admission verdict stays with the consumer; tests drive the
//     consumer on captured records instead of re-implementing it.
//   - internal/scalar owns digest, UUID, and timestamp grammar.
//   - internal/secconftest owns the crash-injection vocabulary the
//     tests wire to the store hooks.
//
// Refusal taxonomy (SPEC Section 5.4: CP-N1..CP-N4 MUST be rejected
// with incompatible_schema before publication):
//
//   - ErrInvalidCheckpoint wraps canonicaljson.ErrInvalidIdentity, so
//     errors.Is proves the incompatible_schema class for every
//     unpublishable closure: malformed grammar, closed-shape
//     violations, non-quiescent terminal evidence, missing or
//     doubled persistence members, the wrong persistence variant
//     for the declared session kind, and heads that postdate the
//     owning lease.
//   - Unknown sessions and unknown heads propagate the sessrepo
//     chain-read errors (ErrUnknownSession, ErrUnknownEvent), so
//     errors.Is preserves the owner's vocabulary.
//   - ErrCheckpointConflict reports the idempotency boundary: the
//     same operation retried with moved inputs, and a digest path
//     holding disagreeing bytes.
//
// Durable-write model (TASK-260830-3qrfjp): content-addressed blobs
// installed no-replace (O_EXCL) with fsync before close, operation
// receipts installed no-replace after the blob they name, directory
// fsync after install. A crash leaves either nothing, or complete
// bytes that verify through the canonical owner; an interrupted
// capture is resumed by retrying the same operation with
// byte-identical inputs (safe_retry), never by inventing state.
//
// Stated bounds: the raw AdmitRecord path attests shape and the
// variant/head gates it can decide, but chain binding needs the
// caller's repository handle; a nil chain refuses closed rather
// than skipping the gate. Capture does not consult lease records:
// creator-holder binding is decided by the consumer at admission
// (pinned by tests here). Capture performs no provider or
// task-board I/O and offers no CLI surface.
package sessckpt
