# TASK-260830-2zs8oz: implement-finalize-and-lineage-publication

## Description
Finalize only after native resume/readiness and publish Migration Checkpoint, operator message, Lineage Receipt, Session Records, and events without hash cycles. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §13.14.3-13.14.5. Work only inside the cross-environment-clone-transaction story boundary.

## Acceptance Criteria
Production behavior demonstrates: Finalize only after native resume/readiness and publish Migration Checkpoint, operator message, Lineage Receipt, Session Records, and events without hash cycles. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
