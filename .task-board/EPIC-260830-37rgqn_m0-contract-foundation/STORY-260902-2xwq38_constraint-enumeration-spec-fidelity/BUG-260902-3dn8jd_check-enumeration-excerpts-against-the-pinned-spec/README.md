# BUG-260902-3dn8jd: check-enumeration-excerpts-against-the-pinned-spec

## Description
The constraint-enumeration gate compares the artifact against the CODE and never against the SPEC, so artifact and implementation can be wrong about the contract together while the test stays green.

internal/canonicaljson/constraint_inventory_test.go:52-121 derives rows from requireExactMembers literals and compares call sites. The specExcerpt column is only required to be NON-EMPTY at :98-100. It is never compared to SPEC.md.

That is how the quoted word digest reached the Pinned SPEC declaration column for store_schema_fingerprint although no such text exists in the pinned document, and how overlapping was narrowed to the child-partition case while the row still read as enforced.

Both audits reached this independently: the derivation is a self-consistency proof wearing a fidelity proof clothes. It resists drift between artifact and code, which is real value, and proves nothing about the spec.

The pinned SPEC.md is fetchable and its SHA-256 is already pinned in internal/specpin, so the comparison is available - it simply is not made.

## Scope
Normative scope: §17.2 traceability, and the enumeration artifact contract.

## Acceptance Criteria
Every specExcerpt is compared verbatim against the pinned SPEC.md at the declared line, and a row whose quote is absent from the pinned document reddens the suite. The comparison uses the digest already pinned in internal/specpin so a swapped spec cannot pass. Planting a row with an invented quote reddens, proven by doing it. Rows that deliberately paraphrase are marked as such and are held to naming the line they paraphrase.
