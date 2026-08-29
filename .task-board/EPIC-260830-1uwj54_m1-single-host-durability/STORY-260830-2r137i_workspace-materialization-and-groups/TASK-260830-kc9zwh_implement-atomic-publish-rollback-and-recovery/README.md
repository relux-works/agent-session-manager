# TASK-260830-kc9zwh: implement-atomic-publish-rollback-and-recovery

## Description
Publish atomically with rollback retained through live validation and recover every journal phase. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §10.5-10.7, §12.3-12.6. Work only inside the workspace-materialization-and-groups story boundary.

## Acceptance Criteria
Production behavior demonstrates: Publish atomically with rollback retained through live validation and recover every journal phase. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
