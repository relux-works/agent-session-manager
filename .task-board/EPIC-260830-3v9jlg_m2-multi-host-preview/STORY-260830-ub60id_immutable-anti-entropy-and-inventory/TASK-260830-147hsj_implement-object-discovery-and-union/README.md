# TASK-260830-147hsj: implement-object-discovery-and-union

## Description
Find missing objects, fetch by digest, preserve concurrent branches, and rebuild projections without timestamp authority. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §11.4, §10. Work only inside the immutable-anti-entropy-and-inventory story boundary.

## Acceptance Criteria
Production behavior demonstrates: Find missing objects, fetch by digest, preserve concurrent branches, and rebuild projections without timestamp authority. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
