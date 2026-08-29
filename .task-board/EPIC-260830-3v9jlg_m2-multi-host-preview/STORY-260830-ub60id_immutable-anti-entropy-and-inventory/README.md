# STORY-260830-ub60id: immutable-anti-entropy-and-inventory

## Description
Implement converge immutable authoritative objects by set union without last-write-wins according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §11.4, §10. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Converge immutable authoritative objects by set union without last-write-wins is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
