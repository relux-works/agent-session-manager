// Package sessstate derives deterministic logical-session state from
// immutable records according to AX v0.5.0 Sections 2.3, 5.1-5.2, 5.7,
// and 14.4: every SessionState value, the winning lease epoch, lease
// conflicts, the current checkpoint, provider identity, and terminal
// binding.
//
// The reducer is a pure function of its input: Reduce takes decoded
// record and event facts and returns one Projection. It performs no
// I/O, reads no clock, and keeps no cache, so the same event sequence
// always yields the same state. Project binds the pure core to a
// sessrepo.Repository for one session: it loads the stored record and
// the authoritative chain through the repository's production entries
// and folds them through Reduce. Project only reads; it mutates no
// durable state of its own and owns no crash window.
//
// Division of authority inside this story:
//
//   - Durable session state — the immutable record, the ordered event
//     chain, source sequence continuity, and the per-session parked
//     channel — is owned by internal/sessrepo and reached only through
//     its production entries (ListSessions, GetRecord, ListEvents,
//     GetEvent). This package never decodes a chain index, never
//     attests a blob, and never lists a session directory.
//   - Canonical attestation stays single-owned: identity, the closed
//     Session Record and Session Event shapes, the per-type payload
//     union, and the unknown-v1-type inert rule are verified by
//     internal/canonicaljson at the sessrepo durable boundary only.
//     The provhost no-attestation-outside-the-leaf gate pins that
//     exclusivity, so Decode projects members from attested bytes and
//     never re-verifies. The payload members Reduce reads were
//     admitted by that closed shape, and a member that no longer
//     decodes refuses the derivation instead of reading as absent.
//   - Strict frame decoding and scalar bound checks stay with their
//     owners: environ.DecodeStrictObject for the JSON frame,
//     environ.CheckUUIDv7 and environ.CheckUint53Bounds for identifier
//     and counter members, scalar.ParseDigest and scalar.ParseUUIDv4
//     for digest and UUID grammars (UUIDv7 members stay with
//     environ.CheckUUIDv7, which owns that check),
//     scalar.ParseProviderID for the provider alphabet, and
//     terminalbackend.ParseID for the terminal backend alphabet.
//   - The refusal census denominator is derived from production
//     through internal/invcore; see census_test.go.
//
// A parked session from sessrepo reaches SessionState as a state, not
// as an error or an omission: when ListSessions reports Parked for the
// session, Project returns a parked Projection carrying the blocking
// reason and the retry hint, and Reduce folds an explicit Parked input
// the same way. A name that exists but is parked still resolves as not
// found at the sessrepo entry; Project takes a session UUID, and the
// listing distinguishes a parked session (entry present) from an
// unknown one (entry absent), so no sessrepo interface change is
// needed for this leaf.
//
// Stated bounds (not silent gaps):
//
//   - Single authoritative chain. Reduce folds exactly the events it
//     is given in order and re-checks chain continuity itself, so a
//     chain-forbidden reordering refuses instead of silently deriving
//     a different state. Cross-partition lease convergence beyond the
//     tuple rule belongs to the ownership-leases story: competing
//     off-chain leases enter only through Input.Union, and the union
//     never rewrites authoritative state — it selects the reported
//     winner and records conflicts.
//   - No name resolution, no list/status rendering, no peer names, and
//     no interactive choice: the local sessrepo steps are reused
//     through Project's listing lookup, and the full Section 2.3 order
//     with choice plus Section 14.4 rendering belong to the sibling
//     name-resolution leaf that builds on this projection.
//   - No persisted profile derivation. Section 2.4 effective-profile
//     authority is out of this leaf's scoped sections; profile.changed
//     and the profile pairs on launch, resume, and fork events are
//     retained as inert facts.
//   - No execution-realm readiness, workspace materialization status,
//     capability map, or sync-age derivation: those Section 14.4
//     fields belong to the provider, workspace, and mesh leaves. The
//     warnings member carries only what this leaf derives
//     (divergent history, stale process, parked blocking reason).
//   - Local role derives from a caller-supplied local host ID against
//     the owner host of the newest authoritative event. An empty local
//     host ID leaves the role unknown rather than guessing replica.
//   - An empty chain derives creating with no authoritative lease, but
//     the union is still validated and resolves the reported winner
//     through the same resolveWinner path: a malformed union refuses
//     exactly as on a non-empty chain, while a well-formed union moves
//     only the reported winner (with its named conflict) and never
//     rewrites the creating state. Local is ignored on this path, including
//     malformed and partial tuples: without an authoritative event there is
//     no local process lease to fence or owner host to compare. LocalHostID
//     leaves the role unknown, even when Union reports a winner. No Local
//     validation is claimed. TestReduceEmptyChainLocalIsIgnored pins this
//     bound through Reduce with and without a selected union winner.
//
// The package adds no ax command, no doctor result, and no runtime
// capability claim.
package sessstate
