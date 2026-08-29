# STORY-260830-3j3iw9: clone-lineage-checkpoints-and-provenance

## Description
Implement integrate clone/adopt provenance and hash-chained lineage without mutating provider identity according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §5.1-5.2, §13.14.2, §13.14.5. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Integrate clone/adopt provenance and hash-chained lineage without mutating provider identity is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
