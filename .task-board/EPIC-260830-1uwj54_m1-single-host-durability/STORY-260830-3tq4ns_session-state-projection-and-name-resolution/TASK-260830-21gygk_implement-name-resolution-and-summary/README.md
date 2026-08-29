# TASK-260830-21gygk: implement-name-resolution-and-summary

## Description
Resolve UUID/name/qualified selectors, ambiguity, list/status summaries, and stable deterministic sorting. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §2.3, §5.1-5.2, §5.7, §14.4. Work only inside the session-state-projection-and-name-resolution story boundary.

## Acceptance Criteria
Production behavior demonstrates: Resolve UUID/name/qualified selectors, ambiguity, list/status summaries, and stable deterministic sorting. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
