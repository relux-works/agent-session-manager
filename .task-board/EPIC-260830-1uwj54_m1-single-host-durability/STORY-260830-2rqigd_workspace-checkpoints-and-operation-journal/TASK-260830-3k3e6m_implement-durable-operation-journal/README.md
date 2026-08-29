# TASK-260830-3k3e6m: implement-durable-operation-journal

## Description
Persist operation receipts, phase transitions, idempotency input digests, status-first recovery, and parked/rollback outcomes. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §5.4, §10.5-10.6, §13.12-13.13. Work only inside the workspace-checkpoints-and-operation-journal story boundary.

## Acceptance Criteria
Production behavior demonstrates: Persist operation receipts, phase transitions, idempotency input digests, status-first recovery, and parked/rollback outcomes. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
