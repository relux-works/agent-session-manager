# TASK-260830-35htyv: publish-pilot-incidents-and-patterns

## Description
Publish sanitized retention, recurrence, failure, and recovery findings with implementation issues triaged separately. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); Advisory evidence for §13 and §19. Work only inside the multi-host-daily-driver-pilots story boundary.

## Acceptance Criteria
Production behavior demonstrates: Publish sanitized retention, recurrence, failure, and recovery findings with implementation issues triaged separately. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
