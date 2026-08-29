# TASK-260830-14yo67: implement-workspace-checkpoint-records

## Description
Capture typed checkpoint closure over provider identity, workspace manifests, task-board bundle, terminal evidence, and source head. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §5.4, §10.5-10.6, §13.12-13.13. Work only inside the workspace-checkpoints-and-operation-journal story boundary.

## Acceptance Criteria
Production behavior demonstrates: Capture typed checkpoint closure over provider identity, workspace manifests, task-board bundle, terminal evidence, and source head. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
