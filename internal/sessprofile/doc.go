// Package sessprofile derives and persists the Section 2.4 effective
// execution profile: the Session Record creation value followed by the
// newest authoritative profile.changed event in lease/sequence order.
//
// Normative scope: AX v0.6.0 Sections 2.4 (execution profiles, the
// seven PROFILE-* fixtures), 5.1/5.2 (the creation member and the
// launch/resume/fork/profile.changed payload pairs), 5.4 (the
// checkpoint event-head closure that fixes the pair), 7.7 (the
// provider mapping the launch projection resolves — owned by
// internal/provhost), and 9.3 (the Task-board Bundle pair). Sections
// 2.4 and 7.7 are textually identical in v0.5.0. Sections 5.5 and 8
// are the tuple authority behind the mapping resolver, not profile
// authority: provider mapping strings are evidence, never the
// persisted profile.
//
// Division of authority inside this story:
//
//   - Durable session state — the immutable record, the ordered event
//     chain, sequence continuity, and attestation — is owned by
//     internal/sessrepo and reached only through its production
//     entries. Decode projects members from attested bytes and never
//     re-verifies; MintChangeEvent computes the omit-self digest of
//     bytes it authors through canonicaljson.CalculateObjectIdentity
//     (the CreateIdentity precedent) and never calls
//     VerifyObjectIdentity, so the provhost
//     no-attestation-outside-the-leaf bound holds: only the sessrepo
//     append path attests the minted event.
//   - Lifecycle derivation stays with internal/sessstate, which
//     treats profile.changed and the profile pairs as inert facts.
//     This package derives the profile pair only and re-checks chain
//     continuity over its input for the same reason sessstate does:
//     a chain-forbidden reordering refuses instead of silently
//     deriving a different pair.
//   - Lease/authority admission stays with sessrepo (chain order) and
//     internal/sessquery (the winning-lease checkpoint admission,
//     whose checkCheckpointProfileAuthority is the admission twin of
//     DeriveForHeads). That twin's closure capability is unexported
//     and bound to the selector Reader, so no external package can
//     consume it; this package therefore defines its closure input
//     explicitly — heads over the sessrepo chain index, transitive
//     predecessors, the Session Record digest as the genesis
//     terminal, index membership as authority — with identical
//     closure semantics.
//   - The Section 7.7 mapping table and the exact-version probe gate
//     stay with internal/provhost (ProfileMapping, CheckResumeTuple,
//     ResolveMapping). This package never maps a profile to a flag.
//   - Takeover, resume, fork, materialization, and bridge
//     transactions stay with their owning leaves. This package owns
//     the reducer-level pair projections those transactions must
//     carry (bundle, plugin resume, resumed/fork, finalize, bridge)
//     and the checks that refuse a divergent pair with
//     integrity_failure.
//
// Confirmation is a publication rule, not derivation input: the
// set-profile transaction refuses an unconfirmed change to yolo, and
// every chained profile.changed event is then authoritative by
// construction — matching the sessquery admission, which reads only
// the target. There is no time-expiring ownership lease in the
// pinned Section 5.3, so "losing or expired" at set-profile means
// the acting lease no longer equals the chain-head lease (a stale
// epoch or a divergent same-epoch token); wall-clock liveness is not
// authority.
//
// Stated bounds (not silent gaps):
//
//   - No named profile registry exists in the pinned specification:
//     the profile vocabulary is exactly the standard|yolo enum named
//     by the CLI and bridge flags. Absent-flag defaulting for ax
//     start --profile belongs to the CLI-surface leaf, which passes
//     this package an explicit value; ResolveCreationProfile refuses
//     anything else, including empty.
//   - No ax command, doctor result, or runtime capability claim is
//     added. The set-profile transaction mints and appends the event
//     the ax session set-profile surface will drive; CLI parsing,
//     rendering, and the cliresult body stay with their owners.
//   - On an eventless chain the acting lease opens sequence 1: the
//     Lease Record binding of that opening lease belongs to the
//     ownership-leases story, and the caller passes the winning
//     lease. On a non-empty chain the acting lease must equal the
//     chain head exactly; a greater-epoch acting lease is refused
//     because set-profile authors under the current lease only —
//     lease succession arrives through takeover flows, never through
//     a profile event.
//   - Set-profile replay recognizes an already-committed change by
//     its request envelope (from, to, lease, author, instant) on the
//     newest authoritative change. A retry under a superseded lease
//     still refuses the lease first: replay never bypasses fencing.
package sessprofile
