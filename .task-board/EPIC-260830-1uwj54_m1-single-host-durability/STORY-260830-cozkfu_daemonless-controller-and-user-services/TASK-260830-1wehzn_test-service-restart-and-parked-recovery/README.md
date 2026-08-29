# TASK-260830-1wehzn: test-service-restart-and-parked-recovery

## Description
Prove service crashes/restarts never self-elect ownership or activate without verified terminal/provider readiness. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §3.1, §4.4, §13.11. Work only inside the daemonless-controller-and-user-services story boundary.

## Acceptance Criteria
Production behavior demonstrates: Prove service crashes/restarts never self-elect ownership or activate without verified terminal/provider readiness. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
