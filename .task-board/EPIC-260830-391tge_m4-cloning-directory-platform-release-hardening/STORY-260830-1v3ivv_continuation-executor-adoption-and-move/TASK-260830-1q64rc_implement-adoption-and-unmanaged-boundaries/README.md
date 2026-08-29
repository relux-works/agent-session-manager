# TASK-260830-1q64rc: implement-adoption-and-unmanaged-boundaries

## Description
Adopt only source-locally behind exact tuple gates and forbid remote unmanaged open; offer clone when adoption cannot be safe. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §13.15, §14.5. Work only inside the continuation-executor-adoption-and-move story boundary.

## Acceptance Criteria
Production behavior demonstrates: Adopt only source-locally behind exact tuple gates and forbid remote unmanaged open; offer clone when adoption cannot be safe. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
