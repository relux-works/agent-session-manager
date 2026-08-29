# STORY-260830-ptxkqe: ax-pane-and-terminal-instance-binding

## Description
Implement make ax pane the only stable provider entry point and bind host-local terminal instances according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §4.B-4.D, §5.2 Terminal Events, §7.A, §13.1. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Make ax pane the only stable provider entry point and bind host-local terminal instances is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
