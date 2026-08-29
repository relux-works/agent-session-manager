# TASK-260830-1rmysu: implement-mesh-health-events-and-metrics

## Description
Emit sanitized RPC/transfer/conflict/lag observations and health checks without transcript, credential, or terminal leakage. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §13.3, §14, §18. Work only inside the sync-status-doctor-and-peer-operations story boundary.

## Acceptance Criteria
Production behavior demonstrates: Emit sanitized RPC/transfer/conflict/lag observations and health checks without transcript, credential, or terminal leakage. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
