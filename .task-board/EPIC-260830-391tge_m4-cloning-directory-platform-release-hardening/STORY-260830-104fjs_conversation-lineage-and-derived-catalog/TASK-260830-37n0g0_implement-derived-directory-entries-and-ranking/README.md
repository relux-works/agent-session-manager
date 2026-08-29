# TASK-260830-37n0g0: implement-derived-directory-entries-and-ranking

## Description
Select deterministic representatives or mixed aggregates, title subjects, freshness, conflicts, reachability, and open loops. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §10.8.2, §10.8.5. Work only inside the conversation-lineage-and-derived-catalog story boundary.

## Acceptance Criteria
Production behavior demonstrates: Select deterministic representatives or mixed aggregates, title subjects, freshness, conflicts, reachability, and open loops. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
