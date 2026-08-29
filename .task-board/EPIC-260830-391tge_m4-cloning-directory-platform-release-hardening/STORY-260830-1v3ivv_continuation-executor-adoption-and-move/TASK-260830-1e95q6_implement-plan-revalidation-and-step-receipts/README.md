# TASK-260830-1e95q6: implement-plan-revalidation-and-step-receipts

## Description
Revalidate every expected source/lease/runtime/target/auth/workspace/policy/capability fact and execute stable idempotent steps by operation ID. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §13.15, §14.5. Work only inside the continuation-executor-adoption-and-move story boundary.

## Acceptance Criteria
Production behavior demonstrates: Revalidate every expected source/lease/runtime/target/auth/workspace/policy/capability fact and execute stable idempotent steps by operation ID. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
