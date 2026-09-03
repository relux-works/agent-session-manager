# BUG-260903-3nt3ah: canonicalize-doc-promises-refusal-it-does-not-perform

## Description
Finding N5 (LOW) of the second adversarial audit, at internal/canonicaljson/canonical.go:139-159. The exported Canonicalize doc comment says it rejects non-I-JSON input. It does not: 9007199254740993 becomes 9007199254740992, 18446744073709551615 becomes 18446744073709552000, and 1.0 becomes 1. Probe: TestAuditProbeCanonicalizeNumbers.

The identity path is protected by validateAXNumbers at :182, and tests pin the rounding as RFC 8785 Appendix B behaviour at canonical_test.go:47-70, so no live defect exists today. That is the same shape as the subsumed UTF-8 re-checks: an argument that holds only while no caller exists. Canonicalize is EXPORTED, so the first external caller inherits a documented guarantee the function does not provide.

This is the last of the thirteen second-audit findings still open. The other twelve are landed.

## Scope
(define bug scope / affected area)

## Acceptance Criteria
Canonicalize is documented to do exactly what it does: either it refuses the numbers its doc comment claims to refuse, or the doc states the RFC 8785 Appendix B rounding as intended behaviour with the reason. The documented contract is pinned by a test that fails when the behaviour changes, and the division of guarantees between Canonicalize and validateAXNumbers is stated where a caller will see it.
