# TASK-260830-4ayhz8: implement-derivation-provenance-and-clone-events

## Description
Implement new/fork/clone/adopt creation provenance, clone/move lifecycle events, and immutable provider-bound Session Records. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §5.1-5.2, §13.14.2, §13.14.5. Work only inside the clone-lineage-checkpoints-and-provenance story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement new/fork/clone/adopt creation provenance, clone/move lifecycle events, and immutable provider-bound Session Records. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
