# TASK-260830-27abiw: implement-owner-resolution-and-attach-routing

## Description
Resolve the winning owner, verify reachability/backend capability, and route attach without lease mutation. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §4.E, §13.5, §16.7. Work only inside the remote-attach-and-terminal-forwarding story boundary.

## Acceptance Criteria
Production behavior demonstrates: Resolve the winning owner, verify reachability/backend capability, and route attach without lease mutation. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
