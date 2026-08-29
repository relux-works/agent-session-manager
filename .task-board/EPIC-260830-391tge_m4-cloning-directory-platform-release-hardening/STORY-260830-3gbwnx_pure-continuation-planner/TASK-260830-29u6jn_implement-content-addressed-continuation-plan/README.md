# TASK-260830-29u6jn: implement-content-addressed-continuation-plan

## Description
Bind exact source/target heads, lease/checkpoint/runtime, tuples/auth, workspace cohort, transfer/fidelity, cost, confirmations, expiry, and step DAG. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §10.8.4, §13.15. Work only inside the pure-continuation-planner story boundary.

## Acceptance Criteria
Production behavior demonstrates: Bind exact source/target heads, lease/checkpoint/runtime, tuples/auth, workspace cohort, transfer/fidelity, cost, confirmations, expiry, and step DAG. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
