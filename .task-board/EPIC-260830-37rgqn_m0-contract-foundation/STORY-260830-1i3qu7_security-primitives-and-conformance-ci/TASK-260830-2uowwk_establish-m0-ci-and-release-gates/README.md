# TASK-260830-2uowwk: establish-m0-ci-and-release-gates

## Description
Run historical/current contract preservation, linters, race tests, coverage, fixture matrices, and unsupported-capability claim checks in CI. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §16, §19.2-19.5, §20.2, Appendix D. Work only inside the security-primitives-and-conformance-ci story boundary.

## Acceptance Criteria
Production behavior demonstrates: Run historical/current contract preservation, linters, race tests, coverage, fixture matrices, and unsupported-capability claim checks in CI. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
