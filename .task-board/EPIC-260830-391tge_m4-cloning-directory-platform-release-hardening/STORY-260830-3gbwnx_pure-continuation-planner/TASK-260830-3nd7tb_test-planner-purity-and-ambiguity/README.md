# TASK-260830-3nd7tb: test-planner-purity-and-ambiguity

## Description
Prove planning may probe only, requires exact source selection, exposes unsupported routes, and never quiesces/transfers/materializes/launches. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §10.8.4, §13.15. Work only inside the pure-continuation-planner story boundary.

## Acceptance Criteria
Production behavior demonstrates: Prove planning may probe only, requires exact source selection, exposes unsupported routes, and never quiesces/transfers/materializes/launches. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
