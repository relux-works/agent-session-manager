# STORY-260830-3m3s4a: conflict-tombstone-and-gc-union

## Description
Implement preserve conflicts and distribute scoped deletions without erasing evidence according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §10.7, §11.7, §17.3. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Preserve conflicts and distribute scoped deletions without erasing evidence is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
