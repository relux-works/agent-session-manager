# STORY-260830-1oqfec: ownership-leases-and-fencing

## Description
Implement make one winning owner and monotonic fencing epoch enforceable locally according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §2.2, §5.3, §13.6-13.10. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Make one winning owner and monotonic fencing epoch enforceable locally is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
