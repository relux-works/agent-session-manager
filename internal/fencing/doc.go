// Package fencing gates every activation-class action on the winning
// lease and the exact fencing epoch.
//
// Normative scope: AX v0.6.0 Section 2.2 global invariants (the winning
// committed lease is the only authority that may create a destination
// runtime; a fencing token is the winning lease_id + epoch pair carried
// by every owner-authored event and mutation; force takeover fences the
// prior owner logically), Section 5.3 Lease Record (epoch, predecessor,
// holder, fencing token, revalidation before input, turn, checkpoint,
// push, and resume), Sections 13.6-13.10 (graceful takeover, force
// takeover, fork, stop, resume: which steps require the winning lease
// and the exact epoch), the Section 4 terminal wrapper rule (compare the
// local fencing token before launching a provider; park safely when
// ownership is remote, ambiguous, or unverified), and the Section 7.5
// Provider Protocol LeaseToken carried by quiesce, capture, and
// materialize operations.
//
// Division of authority (no second lease model):
//
//   - The Lease Record shape, identity, and lifecycle (CreateLease,
//     CompareAndSwapLease, WinningLease, VerifyFencingToken,
//     CheckFencingExpiry, CompareLeaseTuple) are owned by
//     internal/sessrepo. This package never mints, persists, orders, or
//     revalidates a lease itself: Observe loads the winning summary
//     through ListLeases, and the expiry arm calls CheckFencingExpiry.
//   - The winning-tuple rule is owned by internal/sessstate (Compare).
//     This package never compares two leases against each other; it
//     compares one presented token for exact equality with the one
//     winning summary the store reports.
//   - Takeover, fork, stop, and resume transactions, terminal backends,
//     and provider plugin processes are callers, not members: the gates
//     model their entry points (AuthorizeActivation, AuthorizeInput,
//     AuthorizeMutation, AuthorizeCheckpoint, AuthorizeRestore,
//     AuthorizeTerminateStale) and the callers report their own sync,
//     handoff, and grant facts through Observation. StaleRelativeToWinner
//     is the same gate's direction/tuple fact without the grant
//     precondition: a grant authorizes, it never decides staleness, so
//     projections that must fence grant-less incarnations read the
//     verdict instead of re-comparing tuples.
//
// The gates are pure over durable inputs: they perform no writes, hold
// no cache, and read no clock except the Now the caller supplies, so
// there is no crash seam and no idempotency hazard to evidence beyond
// the no-write test. Every refusal carries a Section 15 registered code
// and nothing else: lease_conflict, not_owner, and stale_owner at exit
// 10, invalid_arguments at exit 2, and local_precondition_failed at exit
// 3. Launch-class entries (activation, restore) park instead of refusing
// when ownership is remote, ambiguous, unverified, or absent, carrying
// the Section 5.2 session.parked vocabulary
// (remote_owner|stale_owner|restore_policy|failed_handoff) with the
// winning lease ID; every other entry refuses outright.
package fencing
