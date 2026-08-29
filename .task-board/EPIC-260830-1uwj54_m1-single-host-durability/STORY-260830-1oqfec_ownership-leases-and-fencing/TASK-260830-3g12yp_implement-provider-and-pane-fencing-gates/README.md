# TASK-260830-3g12yp: implement-provider-and-pane-fencing-gates

## Description
Require winning lease and exact epoch before provider activation, input, mutation, checkpoint, or terminal restore. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §2.2, §5.3, §13.6-13.10. Work only inside the ownership-leases-and-fencing story boundary.

## Acceptance Criteria
Production behavior demonstrates: Require winning lease and exact epoch before provider activation, input, mutation, checkpoint, or terminal restore. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
