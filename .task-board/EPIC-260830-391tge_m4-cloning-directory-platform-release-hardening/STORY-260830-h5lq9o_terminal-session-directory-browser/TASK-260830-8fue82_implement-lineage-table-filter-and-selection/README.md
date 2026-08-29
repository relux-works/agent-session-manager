# TASK-260830-8fue82: implement-lineage-table-filter-and-selection

## Description
Implement filter/status bar, lineage rows, expansion, stable-ID selection across refresh, deterministic sort, and narrow layouts. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §14.5, §16.7. Work only inside the terminal-session-directory-browser story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement filter/status bar, lineage rows, expansion, stable-ID selection across refresh, deterministic sort, and narrow layouts. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
