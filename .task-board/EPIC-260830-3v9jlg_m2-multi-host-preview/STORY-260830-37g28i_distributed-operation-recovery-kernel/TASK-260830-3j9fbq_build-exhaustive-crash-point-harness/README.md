# TASK-260830-3j9fbq: build-exhaustive-crash-point-harness

## Description
Run every inter-phase crash/restart/lost-response boundary and assert only safe_retry, explicit_rollback, or recoverable_parked_state. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §10.6, §13.12-13.13. Work only inside the distributed-operation-recovery-kernel story boundary.

## Acceptance Criteria
Production behavior demonstrates: Run every inter-phase crash/restart/lost-response boundary and assert only safe_retry, explicit_rollback, or recoverable_parked_state. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
