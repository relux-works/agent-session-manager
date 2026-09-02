# STORY-260902-2xwq38: constraint-enumeration-spec-fidelity

## Description
The constraint-enumeration gate compares the artifact against the code and never against the pinned SPEC, so artifact and implementation can be wrong about the contract together while the suite stays green. This is the mechanism behind the invented constraints found repeatedly during audit remediation, not another instance of them.

## Scope
Normative scope: the pinned AX v0.5.0 SPEC.md as the fidelity source; internal/canonicaljson/constraint_inventory_test.go and internal/specpin.

## Acceptance Criteria
Every specExcerpt row is compared against the pinned SPEC.md text, not merely required to be non-empty, and a row quoting text absent from the pinned document reddens.
