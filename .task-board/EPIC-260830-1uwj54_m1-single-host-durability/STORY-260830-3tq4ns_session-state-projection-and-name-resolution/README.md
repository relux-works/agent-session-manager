# STORY-260830-3tq4ns: session-state-projection-and-name-resolution

## Description
Implement derive deterministic logical-session state and operator resolution from immutable records according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §2.3, §5.1-5.2, §5.7, §14.4. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Derive deterministic logical-session state and operator resolution from immutable records is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
