// Package matjournal persists the Materialization Journal 2.0.0 recovery
// contract over prepare, staging, validation, commit, and rollback
// (STORY-260830-2rqigd, TASK-260830-3k3e6m).
//
// Authority: relux-works/agent-session-manager-spec@v0.6.0, Section 10.6
// (Materialization Journal), Section 10.5 (Materialization Plan, referenced
// immutable input), Section 13.12 (failure matrix), and Section 13.13
// (crash/restart outcome gate). The task scope cites the same section
// numbers of v0.5.0; the headings are retained in v0.6.0.
//
// The package owns the mutable journal document, its phase machine, the
// prepare receipt, and the status-first recovery evaluator. It reuses the
// landed shared owners and never duplicates them:
//
//   - internal/canonicaljson owns canonical JSON (Canonicalize) for the
//     prepare-request digest and the marker omit-self digest. The closed
//     journal/marker shapes are validated in this package: the shape owner
//     still registers materialization-journal 2.0.0 (and
//     materialization-plan 1.0.0) with rejectUnsupportedImmutableObjectShape,
//     and its census and inventory suites pin that registration, so replacing
//     it is outside this leaf's scope. The plan itself stays an opaque
//     digest input; only its authority routing view is cross-checked.
//   - internal/scalar owns digest, UUID, timestamp, path, and provider-id
//     grammar.
//   - internal/environ owns strict object decode and extension keys.
//   - internal/axerror owns the Structured Error recorded as last_error
//     (cause kept off the wire by construction) and the stable codes that
//     classify recovery outcomes.
//   - internal/sessckpt owns the operation-receipt discipline this journal
//     adopts: an operation-keyed, input-digest-compared, no-replace receipt
//     installed after the bytes it names. The prepare receipt is a distinct
//     record from a checkpoint-capture receipt (different operation
//     namespace, different idempotency key and digest), documented in
//     TRACEABILITY.md; the install order and replay rules are shared.
//   - internal/secconftest owns the crash-injection vocabulary the tests
//     wire to the store hooks. Product code never imports it.
//
// Refusal taxonomy:
//
//   - ErrInvalidJournal reports a journal shape or transition the contract
//     refuses: malformed grammar, closed-shape violations, token-state
//     violations, ID drift, illegal phase or sub-state transitions, and
//     torn durable bytes.
//   - ErrJournalConflict reports the idempotency boundary: the same
//     operation retried with a byte-unequal body (idempotency_mismatch),
//     or a digest path holding disagreeing bytes (a torn store).
//   - ErrUnknownJournal reports a materialization with no journal.
//     Callers distinguish absence (this error) from torn state (the
//     invalid class) and from a store failure (the wrapped operational
//     error).
//
// Durable-write model (TASK-260830-3qrfjp, as in sessckpt/sessrepo):
// immutable sidecars (plan view, prepare receipt) install no-replace
// (O_EXCL) with fsync before close; the mutable journal installs O_EXCL
// at creation and moves through atomic replace (same-directory temp
// file, fsync, rename, directory sync) afterwards, following the Section
// 10.6 current-marker discipline. Directory fsync follows every install.
// A crash leaves either nothing, or complete bytes that re-validate;
// an interrupted operation resumes by retrying the same IDs with a
// byte-identical body (safe_retry), never by inventing state.
//
// Stated bounds: no CLI, no Mesh RPC, no provider plugin or bridge
// process — provider/bridge/marker/lease/native states arrive as modeled
// status inputs to Recover. Filesystem staging removal and predecessor
// restoration are coordinator work; the journal records the converged
// states. Marker history/current files belong to the replica owner;
// this package validates marker bytes and classifies destinations.
package matjournal
