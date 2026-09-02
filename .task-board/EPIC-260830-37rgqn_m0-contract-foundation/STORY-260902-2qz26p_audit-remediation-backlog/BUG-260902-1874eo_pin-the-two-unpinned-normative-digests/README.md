# BUG-260902-1874eo: pin-the-two-unpinned-normative-digests

## Description
Two normative §1.6 digest fixtures from the pinned spec are asserted nowhere in the repository, so two published contract claims are unverified.

bb80eb37... (NUM-U64-STRING) and b0ec84c6... (NUM-U64-MAX) appear in no .go or .json file on any ref. Both reproduce exactly: sha256 of {"n":"9007199254740992"} and sha256 of {"n":"18446744073709551615"} equal the values at SPEC.md:303-304.

Only NUM-SAFE-MAX (e1da48c6...) is pinned, at internal/canonicaljson/canonical_test.go:513. The decimal_uint64 SYNTAX is tested; the normative DIGEST claims are not.

This is cheap and it is exactly the kind of gap that matters in a content-addressed system: a published digest that nothing recomputes is a contract nobody checks.

## Scope
Normative scope: §1.6 canonicalization fixtures, SPEC.md:303-304.

## Acceptance Criteria
Both digests are asserted by hashing the JCS bytes of their fixtures, alongside the existing NUM-SAFE-MAX assertion and in the same style. The expected values are taken from the pinned SPEC lines rather than recomputed from the implementation, so the test can fail. Mutating either fixture or the canonicalization reddens.
