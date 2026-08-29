# TASK-260830-uriz8b: implement-distributed-rollback-and-parking

## Description
Coordinate explicit rollback before commit and recoverable parking when ownership, remote state, or cleanup is ambiguous. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §10.6, §13.12-13.13. Work only inside the distributed-operation-recovery-kernel story boundary.

## Acceptance Criteria
Production behavior demonstrates: Coordinate explicit rollback before commit and recoverable parking when ownership, remote state, or cleanup is ambiguous. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
