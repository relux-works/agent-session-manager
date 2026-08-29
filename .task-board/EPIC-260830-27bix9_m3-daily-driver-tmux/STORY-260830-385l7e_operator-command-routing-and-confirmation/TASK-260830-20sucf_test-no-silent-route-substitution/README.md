# TASK-260830-20sucf: test-no-silent-route-substitution

## Description
Prove stale facts, unavailable hosts, capability loss, and conflicts never silently replan, force, fork, or fall back. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §2.3, §13, §14. Work only inside the operator-command-routing-and-confirmation story boundary.

## Acceptance Criteria
Production behavior demonstrates: Prove stale facts, unavailable hosts, capability loss, and conflicts never silently replan, force, fork, or fall back. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
