# TASK-260830-z1yxg9: implement-rpc-envelope-and-hello

## Description
Implement request/response correlation, hello maps, namespace/cardinality contracts, limits, deadlines, and structured error framing. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §11.2-11.3, §17. Work only inside the mesh-rpc-framing-and-negotiation story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement request/response correlation, hello maps, namespace/cardinality contracts, limits, deadlines, and structured error framing. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
