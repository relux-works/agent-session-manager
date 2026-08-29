# TASK-260830-1dmyt1: test-graceful-takeover-failure-matrix

## Description
Prove stale plan, source race, auth failure, post-commit source-stop failure, lost response, and retry never produce two authorized owners. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §13.6, §13.10, §13.12. Work only inside the graceful-takeover-and-move story boundary.

## Acceptance Criteria
Production behavior demonstrates: Prove stale plan, source race, auth failure, post-commit source-stop failure, lost response, and retry never produce two authorized owners. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
