# TASK-260830-1szonv: implement-target-first-graceful-takeover

## Description
Transfer/materialize, validate destination readiness, commit new ownership epoch, then stop and release the source. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §13.6, §13.10, §13.12. Work only inside the graceful-takeover-and-move story boundary.

## Acceptance Criteria
Production behavior demonstrates: Transfer/materialize, validate destination readiness, commit new ownership epoch, then stop and release the source. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
