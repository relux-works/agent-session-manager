# TASK-260830-19bjfj: fuzz-rpc-operations-and-version-skew

## Description
Fuzz every closed operation body, namespace mismatch, unknown field, replay, lost response, and mixed-version pairing. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §11.2-11.3, §17. Work only inside the mesh-rpc-framing-and-negotiation story boundary.

## Acceptance Criteria
Production behavior demonstrates: Fuzz every closed operation body, namespace mismatch, unknown field, replay, lost response, and mixed-version pairing. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
