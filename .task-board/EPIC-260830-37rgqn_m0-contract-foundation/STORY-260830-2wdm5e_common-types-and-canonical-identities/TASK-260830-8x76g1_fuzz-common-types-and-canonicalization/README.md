# TASK-260830-8x76g1: fuzz-common-types-and-canonicalization

## Description
Prove recursive closed shapes, Unicode ordering, invalid dates, malformed identifiers, and cross-platform golden identities. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); Sections 1.6, 10.1-10.4, 17.3. This task grew beyond fuzz coverage by review: because identity must never attest a structurally invalid object, it now also owns the scoped closed-shape conformance gate for Sections 10.1-10.4 and the fuzz targets that keep that gate honest. Work only inside the common-types-and-canonical-identities story boundary.

## Acceptance Criteria
Production behavior demonstrates: Prove recursive closed shapes, Unicode ordering, invalid dates, malformed identifiers, and cross-platform golden identities. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
