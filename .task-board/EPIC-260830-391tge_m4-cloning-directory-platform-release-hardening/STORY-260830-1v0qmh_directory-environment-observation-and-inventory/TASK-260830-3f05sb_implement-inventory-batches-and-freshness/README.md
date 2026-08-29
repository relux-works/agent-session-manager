# TASK-260830-3f05sb: implement-inventory-batches-and-freshness

## Description
Emit contiguous authoritative batches and derive current/aging/stale/offline/partial/conflicted/unknown plus present/missing/unknown semantics. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §7.9, §10.8.1, §13.15. Work only inside the directory-environment-observation-and-inventory story boundary.

## Acceptance Criteria
Production behavior demonstrates: Emit contiguous authoritative batches and derive current/aging/stale/offline/partial/conflicted/unknown plus present/missing/unknown semantics. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
