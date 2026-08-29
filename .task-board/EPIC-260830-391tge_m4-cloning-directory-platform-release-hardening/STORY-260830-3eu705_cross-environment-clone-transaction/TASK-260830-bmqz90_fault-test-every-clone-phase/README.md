# TASK-260830-bmqz90: fault-test-every-clone-phase

## Description
Crash/loss-inject capture through lineage publication and prove rollback before finalize, idempotent recovery, and no duplicate target session. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §13.14.3-13.14.5. Work only inside the cross-environment-clone-transaction story boundary.

## Acceptance Criteria
Production behavior demonstrates: Crash/loss-inject capture through lineage publication and prove rollback before finalize, idempotent recovery, and no duplicate target session. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
