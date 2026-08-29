# STORY-260830-35dbcs: complete-git-workspace-capture

## Description
Implement capture the complete dirty Git workspace closure into immutable transfer objects according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §10.2-10.4, §12.1-12.3. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Capture the complete dirty git workspace closure into immutable transfer objects is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
