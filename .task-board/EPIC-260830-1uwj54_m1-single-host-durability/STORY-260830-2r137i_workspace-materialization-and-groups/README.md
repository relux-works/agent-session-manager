# STORY-260830-2r137i: workspace-materialization-and-groups

## Description
Implement materialize Git and non-Git workspaces transactionally while preserving groups and cohorts according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §10.5-10.7, §12.3-12.6. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Materialize git and non-git workspaces transactionally while preserving groups and cohorts is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
