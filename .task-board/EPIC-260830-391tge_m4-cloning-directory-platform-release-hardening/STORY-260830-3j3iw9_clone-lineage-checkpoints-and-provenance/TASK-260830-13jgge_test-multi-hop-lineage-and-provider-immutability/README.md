# TASK-260830-13jgge: test-multi-hop-lineage-and-provider-immutability

## Description
Prove clone chains, moves, forks, adoption, archive/context outcomes, and provider changes always preserve identity rules. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §5.1-5.2, §13.14.2, §13.14.5. Work only inside the clone-lineage-checkpoints-and-provenance story boundary.

## Acceptance Criteria
Production behavior demonstrates: Prove clone chains, moves, forks, adoption, archive/context outcomes, and provider changes always preserve identity rules. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
