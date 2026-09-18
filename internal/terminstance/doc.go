// Package terminstance executes the Terminal Instance lifecycle contract of
// SPEC §4.C over the landed internal/terminalbackend semantic core.
//
// Normative scope is relux-works/agent-session-manager-spec §4.C (lifecycle
// state machine, transition/authorization/idempotency table), §4.B (the
// backend_generation canonical bound), §4.D (capability conditionals
// evaluated at the operation call site), §5.2 (status/unknown discipline),
// §7.A (descriptor generation binding, cited landed), §13.1 (durable
// terminal entry before provider launch) and §13.7 (cold force
// takeover: the prior owner is stale when it learns the winner). Worked against
// internal/specdoc/SPEC.v0.7.0.md: §§4.B, 4.C, 4.D, 5.2 and 7.A are
// byte-identical to v0.6.0, while §13.1 carries the v0.7.0 launch-plan
// insertion that precedes step 2 (a pure addition; the durable
// terminal-entry rule text is unchanged).
//
// Composition, not re-implementation: the eight-state enum, the ten-operation
// enum, the ten-effect enum, the transition matrix, the allowed-error sets,
// the idempotency key shapes and the capability registry stay owned by
// internal/terminalbackend (ParseInstanceState, ParseOperation,
// ParseSideEffect, CheckTransition, CheckErrorAllowed, IdempotencyKey,
// CheckOperation, GenerationDigest, CheckProviderDescriptor). Closed-shape
// JSON admission stays owned by internal/environ (DecodeStrictObject,
// CheckStringBounds, CheckUint53Bounds, CheckDigest, CheckUUIDv7,
// CheckTimestamp, CheckSemver). Scalar grammars stay owned by
// internal/scalar. Fencing truth stays owned by internal/fencing
// (Authorize, StaleRelativeToWinner): this package never compares
// lease tuples itself to decide staleness; ObserveFencing reads only
// the landed park vocabulary and the landed staleness verdict. What
// this package owns is the execution layer the landed core deliberately
// leaves out: the AXAuthorization closed object, the RetryDisposition and
// ProviderProofKind closed enums, the mutating-request/response identity
// contract, the per-side-effect recheck loop, the error-to-observation
// mapping, and the durable idempotency receipt store.
//
// Stated bounds (no unsupported claims): no ax command, no tmux/ConPTY
// process control and no provider process exist here; Backend is a modeled
// caller interface. The Terminal Instance Binding 1.0.0 closed object has
// no parser on trunk (canonicaljson explicitly rejects
// urn:ax:schema:terminal-instance-binding) and the CLI Result 4 bodies
// carry no raw generation member, so the generation bound is enforced on
// the surfaces this leaf owns (operation bodies and results) and cited
// landed for the Provider Protocol 3 descriptor; full Binding-object
// parsing belongs with the provisioning surface. Attach, restore,
// terminate-stale, manifest and probe operations are refused by the engine
// scope gate; restore/terminate-stale recovery belongs to the story's
// final leaf. Events (terminal.created/session.resumed emission) belong to
// the final leaf. On reattach the recorded Binding terminal instance ID is
// authoritative; a request-built descriptor names the request's own fresh
// instance, never the recorded child.
package terminstance
