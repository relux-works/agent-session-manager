# TASK-260830-2e8d5i: implement-replica-marker-and-derived-state

## Description
Create managed replica markers and derive browsable Replica state without lease or provider activation. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §10.5-10.7, §11.5-11.6. Work only inside the atomic-object-ingestion-and-replica-state story boundary.

## Acceptance Criteria
Production behavior demonstrates: Create managed replica markers and derive browsable Replica state without lease or provider activation. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
