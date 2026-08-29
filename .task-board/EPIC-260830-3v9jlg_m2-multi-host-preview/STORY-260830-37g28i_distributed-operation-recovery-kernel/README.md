# STORY-260830-37g28i: distributed-operation-recovery-kernel

## Description
Implement extend journals and idempotency across peer operations and lost responses according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §10.6, §13.12-13.13. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Extend journals and idempotency across peer operations and lost responses is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
