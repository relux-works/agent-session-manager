# TASK-260830-2z3se0: implement-session-adapter-protocol-host

## Description
Implement Session Adapter discovery, manifest/probe, closed operations, limits, idempotency, and tuple gates. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §7.8-7.9, §13.14, §10.8. Work only inside the session-adapter-and-directory-node-framework story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement Session Adapter discovery, manifest/probe, closed operations, limits, idempotency, and tuple gates. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
