# TASK-260830-1sbk2u: implement-migration-checkpoint-and-lineage-receipt

## Description
Create typed checkpoints, low-authority messages, chained receipts, previous-manifest links, and acyclic content identities. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §5.1-5.2, §13.14.2, §13.14.5. Work only inside the clone-lineage-checkpoints-and-provenance story boundary.

## Acceptance Criteria
Production behavior demonstrates: Create typed checkpoints, low-authority messages, chained receipts, previous-manifest links, and acyclic content identities. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
