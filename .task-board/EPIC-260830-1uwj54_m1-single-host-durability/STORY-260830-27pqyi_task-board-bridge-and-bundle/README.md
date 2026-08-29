# STORY-260830-27pqyi: task-board-bridge-and-bundle

## Description
Implement integrate direct and task-board-managed sessions without transferring board ownership into AX according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §9, §10.4, §13.2. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Integrate direct and task-board-managed sessions without transferring board ownership into ax is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
