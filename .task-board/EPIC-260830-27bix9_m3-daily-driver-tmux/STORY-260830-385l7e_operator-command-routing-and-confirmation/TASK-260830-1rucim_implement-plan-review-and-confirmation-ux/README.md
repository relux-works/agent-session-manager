# TASK-260830-1rucim: implement-plan-review-and-confirmation-ux

## Description
Render exact source/target/ownership/workspace/auth/cost/risk facts and require confirmation for every mutating ambiguous route. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §2.3, §13, §14. Work only inside the operator-command-routing-and-confirmation story boundary.

## Acceptance Criteria
Production behavior demonstrates: Render exact source/target/ownership/workspace/auth/cost/risk facts and require confirmation for every mutating ambiguous route. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
