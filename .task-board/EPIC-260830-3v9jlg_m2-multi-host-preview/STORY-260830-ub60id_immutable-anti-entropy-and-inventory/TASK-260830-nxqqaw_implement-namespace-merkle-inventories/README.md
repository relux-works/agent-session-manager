# TASK-260830-nxqqaw: implement-namespace-merkle-inventories

## Description
Build deterministic namespace roots/children, object membership validation, pagination, and bounded inventory exchange. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §11.4, §10. Work only inside the immutable-anti-entropy-and-inventory story boundary.

## Acceptance Criteria
Production behavior demonstrates: Build deterministic namespace roots/children, object membership validation, pagination, and bounded inventory exchange. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
