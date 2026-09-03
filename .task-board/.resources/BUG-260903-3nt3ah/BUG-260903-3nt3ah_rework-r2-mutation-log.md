# BUG-260903-3nt3ah rev2 — mutation log (raw)

Harness: `.temp/BUG-260903-3nt3ah/mutate.py`. For every mutant it restores the
pristine file, applies the substitution, **re-reads the file and asserts the
mutant is present**, and only then measures. It exits non-zero if any mutant
survives.

## Summary run

```
N1 fixture re-pointed to the neighbouring row
    mutant present: yes
    RED (exit 1)
    --- FAIL: TestCanonicalizeSpecificationCitationsQuoteThePinnedSpecification (0.01s)

N2 line re-pointed to the neighbouring row
    mutant present: yes
    RED (exit 1)
    --- FAIL: TestCanonicalizeSpecificationCitationsQuoteThePinnedSpecification (0.01s)

N3 quotation weakened to a non-unique fragment
    mutant present: yes
    RED (exit 1)
    --- FAIL: TestCanonicalizeSpecificationCitationsQuoteThePinnedSpecification (0.01s)

N4 fixture citation deleted rather than corrected
    mutant present: yes
    RED (exit 1)
    --- FAIL: TestCanonicalizeSpecificationCitationsQuoteThePinnedSpecification (0.01s)

N5 fixture row cited without naming the fixture
    mutant present: yes
    RED (exit 1)
    --- FAIL: TestCanonicalizeSpecificationCitationsQuoteThePinnedSpecification (0.01s)

N6 documented container limit drifted from maxNestingDepth
    mutant present: yes
    RED (exit 1)
    --- FAIL: TestCanonicalizeDocumentedContainerLimitIsTheEnforcedOne (0.00s)

N7 named Appendix B pin renamed in prose
    mutant present: yes
    RED (exit 1)
    --- FAIL: TestCanonicalizeDocCommentNamesOnlyTestsThatExist (0.01s)

N8 false clause claim added beside the true one
    mutant present: yes
    RED (exit 1)
    --- FAIL: TestCanonicalizeSpecificationCitationsQuoteThePinnedSpecification (0.01s)

N9 clause claim re-pointed to a neighbouring clause
    mutant present: yes
    RED (exit 1)
    --- FAIL: TestCanonicalizeSpecificationCitationsQuoteThePinnedSpecification (0.01s)

restored: 880ca9ad3dc8145886f8ea9b3aa90cf813242cd836e233b808149ddcedb10f6a  internal/canonicaljson/canonical.go
SURVIVORS: none
```

## Representative assertion text

```
--- N1 (exit 1) ---
    canonical_number_contract_test.go:303: citation SPEC.md:302 NUM-UNSAFE-NUMBER "Reject from the JSON number token before conversion to a host double; an implementation that first rounds it to 9007199254740992 is nonconforming" attributes the quote to fixture NUM-UNSAFE-NUMBER, but SPEC.md:302 is the row that declares "NUM-UNSAFE-ROUND"
    canonical_number_contract_test.go:321: Canonicalize doc comment no longer attributes a quotation to normative fixture NUM-UNSAFE-ROUND; the required set is [NUM-UNSAFE-NUMBER NUM-UNSAFE-ROUND]

--- N3 (exit 1) ---
    canonical_number_contract_test.go:289: citation SPEC.md:302 NUM-UNSAFE-ROUND "Reject" quotes text that begins at [301 302 305 306 1211 2264 5191 5201 5202 9067], so it does not identify one place in the pinned SPEC.md; quote enough of the line to be unique rather than a fragment that matches everywhere

--- N5 (exit 1) ---
    canonical_number_contract_test.go:299: citation SPEC.md:302 "Reject from the JSON number token before conversion to a host double; an implementation that first rounds it to 9007199254740992 is nonconforming" quotes the "Fixture" table row that declares "NUM-UNSAFE-ROUND" without naming it; a normative fixture row must be cited by its fixture identifier
    canonical_number_contract_test.go:321: Canonicalize doc comment no longer attributes a quotation to normative fixture NUM-UNSAFE-ROUND; the required set is [NUM-UNSAFE-NUMBER NUM-UNSAFE-ROUND]

--- N8 (exit 1) ---
    canonical_number_contract_test.go:339: doc comment names SPEC.md clauses [1.5 1.6], but every citation lands in [1.6]

restored 880ca9ad3dc8145886f8ea9b3aa90cf813242cd836e233b808149ddcedb10f6a  internal/canonicaljson/canonical.go
```
