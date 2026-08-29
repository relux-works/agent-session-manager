# STORY-260830-1i3qu7: security-primitives-and-conformance-ci

## Description
Implement establish safe process/filesystem primitives and the continuous conformance gate according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §16, §19.2-19.5, §20.2, Appendix D. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Establish safe process/filesystem primitives and the continuous conformance gate is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
