# STORY-260830-26hl2z: force-takeover-and-replacement

## Description
Implement support explicit risk-acknowledged recovery when graceful fencing cannot be proven according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §13.7, §15, §16.5. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Support explicit risk-acknowledged recovery when graceful fencing cannot be proven is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
