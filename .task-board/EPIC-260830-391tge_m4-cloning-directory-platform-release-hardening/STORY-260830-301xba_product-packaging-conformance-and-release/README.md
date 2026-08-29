# STORY-260830-301xba: product-packaging-conformance-and-release

## Description
Implement package and publish ax only after the complete product acceptance matrix is true according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §19-20, Appendices A-D. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Package and publish ax only after the complete product acceptance matrix is true is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
