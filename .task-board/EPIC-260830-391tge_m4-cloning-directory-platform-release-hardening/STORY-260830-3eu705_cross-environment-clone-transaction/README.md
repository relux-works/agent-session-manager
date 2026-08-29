# STORY-260830-3eu705: cross-environment-clone-transaction

## Description
Implement materialize a new native target and new AX logical session through a recoverable transaction according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §13.14.3-13.14.5. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Materialize a new native target and new ax logical session through a recoverable transaction is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
