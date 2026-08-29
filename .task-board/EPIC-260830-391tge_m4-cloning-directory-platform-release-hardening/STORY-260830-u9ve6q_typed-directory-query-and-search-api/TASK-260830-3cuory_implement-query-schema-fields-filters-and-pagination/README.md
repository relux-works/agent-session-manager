# TASK-260830-3cuory: implement-query-schema-fields-filters-and-pagination

## Description
Implement schema discovery, presets, typed filters, projection, batching, deterministic sorting, skip/take bounds, count, and distinct. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §10.8.5, §14.5. Work only inside the typed-directory-query-and-search-api story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement schema discovery, presets, typed filters, projection, batching, deterministic sorting, skip/take bounds, count, and distinct. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
