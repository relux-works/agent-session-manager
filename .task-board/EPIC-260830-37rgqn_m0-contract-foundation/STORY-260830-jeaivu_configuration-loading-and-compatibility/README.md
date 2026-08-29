# STORY-260830-jeaivu: configuration-loading-and-compatibility

## Description
Implement implement closed Configuration 1/2/3 loading, precedence, validation, and downgrade behavior according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §6, §17. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Implement closed configuration 1/2/3 loading, precedence, validation, and downgrade behavior is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
