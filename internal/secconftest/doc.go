// Package secconftest is the conformance-test instrument for the
// security-primitives-and-conformance-ci story: deterministic fixture
// runners, fake clocks, crash points, protocol fuzz drivers, hostile
// strings, and platform capability skips over the Section 16 production
// gates (internal/secprim).
//
// It is an instrument, not a second authority. Every behavior it covers
// is decided by an existing owner; this package only drives those
// decisions deterministically and proves the drive reaches them:
//
//   - path grammars and bounds stay in internal/scalar;
//   - wire decoding, string measure, and bound checks stay in
//     internal/environ;
//   - argv, environment, redaction, escaping, and containment refusals
//     stay in internal/secprim, whose production code this package must
//     not change;
//   - the pinned specification text stays in internal/specdoc, which
//     this package reads only to derive hostile classes, never to
//     advertise product capabilities.
//
// Normative scope: AX v0.5.0 Sections 16, 19.2-19.5, 20.2, Appendix D.
// Section 19.3 provider suites and Section 19.4 lifecycle cases outside
// Section 16 are not driven here; the per-mechanism tests name that
// bound where it bites.
//
// Stated bounds. The crash Transactor is a test model of a three-phase
// durable commit, not the product journal: it proves the injector and
// the outcome vocabulary work, not that any product recovery path
// works. Finalization is terminal for clean and crashed Commits alike
// (a commit-apply fault already performed the durable write, so
// rollback is forbidden and the bytes survive); a pre-commit Rollback
// forgets the operation so a fresh Prepare restages the recovery
// retry, while Prepare/Commit past finalization name the terminal
// state instead of inheriting ENOENT. The five crash-point names are
// instrument-local phase boundaries, not the SPEC CR-MAT-01..08
// registry: registry-style names are refused as unknown, and a new
// crash site added without an arm is outside this model. Every arm
// the suite sets is consumption-checked through RemainingArmed, so an
// armed point the drive never reaches fails its test instead of
// passing silently; all five points are armed through the Transactor,
// including rollback-enter. The fuzz drivers cover the Section 16
// string and vector gates only; containment races and TOCTOU behavior
// are out of scope for byte-mutating fuzz. The IsEnvName and Redact
// fuzz properties pin the two poles a no-op gate fails (refused
// empty/admitted well-formed name; rewritten secret/passing innocuous
// line); the interior grammar and per-class markers stay covered by
// the hostile corpus tests named on each property. Hostile members
// derive from the specification text plus package-unicode tables and
// internal/scalar predicates, so deleting a secprim gate arm keeps
// the corpus and reddens the gate test instead of shrinking the
// witness with it. Require takes a minimal testing handle so the
// skip and fail arms are driven through the real function with a
// recording fake (both recordVerdict call sites pinned); TestMain
// prints the process-wide Report after every test, including the
// parallel ones, so a real parallel verdict is observable there.
// The per-test skip datum is still the `--- SKIP` lines under
// `go test -v`: on a host where every probe passes no verdict is
// recorded and the final counters honestly read zero.
package secconftest
