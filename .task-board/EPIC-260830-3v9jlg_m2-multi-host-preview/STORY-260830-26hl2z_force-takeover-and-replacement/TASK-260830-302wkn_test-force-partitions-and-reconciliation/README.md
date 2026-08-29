# TASK-260830-302wkn: test-force-partitions-and-reconciliation

## Description
Exercise offline owner, simultaneous force attempts, later union, losing epochs, and operator-visible conflicts. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §13.7, §15, §16.5. Work only inside the force-takeover-and-replacement story boundary.

## Acceptance Criteria
Production behavior demonstrates: Exercise offline owner, simultaneous force attempts, later union, losing epochs, and operator-visible conflicts. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
