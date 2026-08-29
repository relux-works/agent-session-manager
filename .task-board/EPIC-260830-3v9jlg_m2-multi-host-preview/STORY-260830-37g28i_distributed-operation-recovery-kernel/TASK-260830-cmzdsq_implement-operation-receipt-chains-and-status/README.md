# TASK-260830-cmzdsq: implement-operation-receipt-chains-and-status

## Description
Persist caller operation IDs before mutation, bind input digests, expose evolving status reads, and detect uncertain results. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §10.6, §13.12-13.13. Work only inside the distributed-operation-recovery-kernel story boundary.

## Acceptance Criteria
Production behavior demonstrates: Persist caller operation IDs before mutation, bind input digests, expose evolving status reads, and detect uncertain results. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
