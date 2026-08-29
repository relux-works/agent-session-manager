# STORY-260830-2rqigd: workspace-checkpoints-and-operation-journal

## Description
Implement create typed checkpoints and durable local operation recovery state according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §5.4, §10.5-10.6, §13.12-13.13. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Create typed checkpoints and durable local operation recovery state is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
