// Package sessrepo persists and validates AX Session Records, the
// append-only per-session event chain with source sequence continuity, and
// the Lease Record lifecycle with compare-and-swap succession.
//
// Normative scope: AX v0.5.0 sections 2.3 (session name resolution), 5.1
// (Session Record), 5.2 (Session Event), 5.7 (derived session states), and
// 14.4 (list and status fields), plus the v0.6.0 lease lifecycle in
// sections 2.2 (global invariants), 5.3 (Lease Record and ownership), and
// 13.6-13.10 (takeover, fork, stop, resume). This package owns durable
// session state: the immutable record, the ordered event chain beneath it,
// the immutable lease blobs with their winning head, and the stored
// identity fields that later leaves project into lifecycle state, resolved
// names, and list/status rows.
//
// Division of authority inside this story:
//
//   - Validation of record, event, and lease closed shapes and of every
//     self-identity digest is owned by internal/canonicaljson and reached
//     only through its production entries (VerifyObjectIdentity for
//     attestation, CalculateObjectIdentity for minting identity). Entry
//     decodes refuse a foreign schema before Verify; the load path
//     re-verifies stored blobs through Verify — recomputing whatever
//     digest the bytes claim — and refuses a non-event_id self field or
//     an index-disagreeing digest at its named arms, or a torn lease blob
//     through the lease load funnel. This package never re-decodes a
//     frame, never recomputes a bound, and never selects a self field.
//   - Strict frame decoding and scalar bound checks on the members this
//     package must read (session and lease identifiers, lease_sequence,
//     predecessor digests, names, fencing tokens, timestamps) are owned by
//     internal/environ and internal/scalar and reached only through their
//     production checks.
//   - Crash-point vocabulary and outcomes are owned by
//     internal/secconftest. The repository fires the owner's own points
//     through the owner's Injector: a fault before the durable write carries
//     safe_retry, a fault after it carries recoverable_parked_state, and a
//     crashed commit counts as committed.
//   - The refusal census denominator is derived from production through
//     internal/invcore; see census_test.go.
//   - The winning-tuple rule is owned by internal/sessstate (Compare).
//     The lease store restates it as CompareLeaseTuple because sessstate
//     imports this package, and a cross-package agreement test pins
//     identical order. Query-layer admission (internal/sessquery
//     winningLeaseFor) is unchanged and admits store-minted records.
//
// Per-session parked reporting (SPEC.md:9082, Section 13.13
// recoverable_parked_state; Section 14.4 per-session warnings).
//
// ListSessions reports one entry per session directory in session-ID
// order. A session whose bytes no longer verify is returned as a parked
// entry — Parked=true with the blocking reason and the same-operation
// retry — and never fails the listing for its healthy siblings. Resolve
// routes only healthy sessions; a query naming only a parked session is
// not found. GetRecord, GetEvent, and ListEvents still refuse a parked
// session per session with the load failure.
//
// Why per-session instead of repository-wide closure: a repository-wide
// listing error carries no per-session channel, so the Section 5.7
// reducer and the Section 14.4 rendering leaves could not project the
// parked lifecycle state, the blocking reason, or the warnings from what
// this leaf exposes — and one torn session denied listing and name
// resolution to every healthy sibling permanently when its original
// bytes were lost. Per-session reporting keeps healthy sessions usable
// while the parked session stays visible. The tradeoff is that callers
// must check Parked: a listing that contains parked entries returns nil
// error by design.
//
// Retry table (same-operation retry, Section 13.13):
//
//   - Bare session directory (no record.json, no chain.json): retry
//     CreateSession with the session record for this session ID. Any
//     valid record for the ID resumes; the parked state is the bare
//     directory left by a crash between the directory and record steps.
//   - Record parked without a chain index (record.json verifies, no
//     chain.json): retry CreateSession with the byte-identical session
//     record. A different record for the same session ID is refused
//     with ErrSessionExists and the parked bytes survive; the
//     byte-equality is full-content, not length-only.
//   - Torn store (chain.json present but the session no longer
//     verifies): no CreateSession retry heals it. A byte-identical
//     retry is refused as an existing session and a differing retry is
//     refused the same way; the store stays parked.
//   - Missing lease after a killed install: retry CreateLease or
//     CompareAndSwapLease with the byte-identical input. The install is
//     stage-then-rename, so a kill leaves at most an orphaned staged
//     temp (ignored on load, never a torn final); the retry replays to
//     the same digest, and a duplicate install of identical bytes is
//     reused. A CAS retry still names its pre-commit expectation: the
//     replay check re-mints against that basis and answers the persisted
//     successor.
//   - Torn lease blob (disagreeing bytes at a digest path, or a blob
//     that no longer attests): no retry heals it. The colliding write is
//     refused with ErrChainCorrupt and every lease read funnels the same
//     class; the operator remedy below applies to the leases namespace.
//
// Operator remedy. There is no delete, prune, or quarantine entry by
// design: records and events are append-only and no production path
// rewrites or deletes an existing entry. When no same-operation retry
// heals a parked session — the original record bytes are lost, or the
// store is torn past the create window — the operator removes the
// session directory at <data-root>/sessions/<session-id> from the
// filesystem after confirming no host will retry with the original
// bytes, and the listing heals on the next read. Removing the directory
// discards whatever that session had parked; it never touches siblings.
// Downstream leaves consume the parked channel as: SessionSummary.Parked
// selects the parked/failed lifecycle projection, BlockingReason feeds
// status/doctor blocking reason and warnings, and RetryHint feeds the
// retry or remedy line.
//
// Stated bounds (not silent gaps):
//
//   - Single authoritative branch. The repository tracks one ordered chain
//     per session in arrival order. A same-epoch event under a different
//     lease is preserved as an immutable blob but refused from the chain
//     with ErrDivergentBranch; converging lease histories across
//     partitions (union and divergent-branch reconciliation) belongs to
//     the sibling ownership leaves, not to this record lifecycle.
//   - No checkpoint admission on the write path. CAS requires a
//     well-formed checkpoint digest for every successor, but validating
//     that the digest names an admitted checkpoint for the predecessor
//     lease stays with the query-layer admission this store feeds.
//   - No wall-clock lease expiry. The lease never expires (Section 5.3);
//     only the process-local fencing grant lapses past the refresh
//     interval, and revalidation renews it. Grants are never persisted
//     and never replicated.
//   - No lifecycle derivation. Section 5.7 SessionState reduction and the
//     full Section 2.3 four-step resolution order (peer-learned names,
//     interactive choice) belong to the sibling reducer and name-resolution
//     leaves. This package exposes the stored facts they project: exact
//     local names with ASCII case-fold ambiguity detection, UUID routing,
//     per-session chain heads, and the parked channel above.
//   - No Section 14.4 rendering. ListSessions exposes stored identity plus
//     chain-head facts (or the parked channel); human or JSON rendering
//     of owner host, checkpoint age, capability status, and warnings
//     belongs to the CLI surface leaf.
//
// The package adds no ax command, no doctor result, and no runtime
// capability claim.
package sessrepo
