# TASK-260830-219okr: implement-dual-stack-major-negotiation

## Description
Negotiate supported majors, preserve core-only peers, expose unsupported directory/backend activation, and refuse no-common-major. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §11.2-11.3, §17. Work only inside the mesh-rpc-framing-and-negotiation story boundary.

## Acceptance Criteria
Production behavior demonstrates: Negotiate supported majors, preserve core-only peers, expose unsupported directory/backend activation, and refuse no-common-major. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
