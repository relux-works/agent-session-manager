# TASK-260830-bowadc: test-safe-gc-and-resurrection-prevention

## Description
Prove offline peers, missing acknowledgements, concurrent tombstones, retained evidence, and reintroduced objects cannot silently resurrect state. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §10.7, §11.7, §17.3. Work only inside the conflict-tombstone-and-gc-union story boundary.

## Acceptance Criteria
Production behavior demonstrates: Prove offline peers, missing acknowledgements, concurrent tombstones, retained evidence, and reintroduced objects cannot silently resurrect state. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
