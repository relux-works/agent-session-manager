// Package cigate implements the M0 continuous-conformance CI gates for the
// security-primitives-and-conformance-ci story: historical and current
// contract preservation, fixture-target derivation, and the
// unsupported-capability advertisement check.
//
// It is a gate, not a second authority. Every behavior it constrains is
// decided by an existing owner; this package only re-derives those decisions
// from their owners and refuses drift:
//
//   - pinned contract versions stay in internal/specpin; the typed release
//     projections stay in internal/catalog. VerifyContractPreservation
//     requires the two authorities to agree exactly and fails closed on a
//     version either side cannot account for;
//   - platform capability availability stays in internal/secconftest probes.
//     CheckAdvertisements takes probe outcomes as input and never defines a
//     capability of its own: it reads ProbeState.Available from the states
//     it is given. CI scans with every probe forced unavailable, which is
//     stronger than live scanning and host-independent; the live probe
//     outcomes reach no availability reader — ProbeStates feeds only
//     CapabilityIDs, which keeps the IDs and drops availability;
//   - the secconftest fuzz targets stay mapped in FuzzTargetEntry.
//     FuzzTargets derives the committed target list from that map rather
//     than retyping it, and TestListedTargetsMatchDerivation holds CI's
//     `go test -list` leg equal to it. Fuzz targets also live in
//     internal/canonicaljson and internal/scalar, which own no entry map;
//     CI smokes those packages from their own `-list` derivations.
//
// Normative scope: AX v0.5.0 Sections 16, 19.2-19.5, 20.2, Appendix D. The
// Section 19.3 provider suites and Section 19.4 lifecycle cases outside
// Section 16 are not driven here; CI runs them only where their owning
// packages already implement them, and this gate claims nothing about the
// rest.
//
// Stated bounds. The gates are read-only: they parse the pinned authorities
// and repository text and write only their verdict to the caller. They mutate
// no durable state, keep no cache, and carry no crash or idempotency surface
// of their own; crash/idempotency evidence for the story is the secconftest
// Transactor battery, which CI executes but which lives outside this package.
// Availability is evaluated on the host that runs the check: a positive claim
// about a capability available here is admitted even when the same sentence
// would be refused on a host where the probe fails. Cross-host claim truth is
// outside this gate. The advertisement scanner matches only backticked
// capability IDs, never bare prose: an advertisement that names a
// capability without backticks — however positive its verbs — is not
// checked and is admitted. The scanner reads backticked prose only; it
// classifies sentences, not intent, and any backticked sentence it cannot
// classify is refused rather than ignored.
//
// Deliberate shape decision: this package ships no main package. Its library
// entry points import internal/secconftest for the probe vocabulary, and
// secconftest reads internal/specdoc, which the specdoc guard forbids any new
// main from reaching (TestEmbeddedDocumentNeverReachesAProductBinary and
// TestModuleHasNoProductCommandYet). A cigate command would need either a
// foreign allowlist widening or a second probe vocabulary, both of which this
// story exists to refuse. CI therefore drives these gates through focused,
// name-guarded go test selections; the guards prove the selection matched.
package cigate
