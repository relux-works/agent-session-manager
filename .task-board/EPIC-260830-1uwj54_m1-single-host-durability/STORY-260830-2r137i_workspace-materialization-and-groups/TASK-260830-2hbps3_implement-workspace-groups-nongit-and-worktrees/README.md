# TASK-260830-2hbps3: implement-workspace-groups-nongit-and-worktrees

## Description
Support non-Git managed trees, group/cohort identity, path relocation, shared repositories, and worktree conflict rules. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §10.5-10.7, §12.3-12.6. Work only inside the workspace-materialization-and-groups story boundary.

## Acceptance Criteria
Production behavior demonstrates: Support non-Git managed trees, group/cohort identity, path relocation, shared repositories, and worktree conflict rules. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
