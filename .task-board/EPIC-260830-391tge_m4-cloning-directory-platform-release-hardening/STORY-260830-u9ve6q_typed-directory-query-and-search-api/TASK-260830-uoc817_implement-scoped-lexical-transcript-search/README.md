# TASK-260830-uoc817: implement-scoped-lexical-transcript-search

## Description
Implement source-local explicit grep with authorization, redaction, bounded excerpts, cost metadata, and no default mesh fan-out. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §10.8.5, §14.5. Work only inside the typed-directory-query-and-search-api story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement source-local explicit grep with authorization, redaction, bounded excerpts, cost metadata, and no default mesh fan-out. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
