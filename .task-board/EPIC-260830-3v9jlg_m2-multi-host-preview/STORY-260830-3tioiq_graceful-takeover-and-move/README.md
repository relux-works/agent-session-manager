# STORY-260830-3tioiq: graceful-takeover-and-move

## Description
Implement move an exact managed session target-first with fenced source shutdown and one owner according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §13.6, §13.10, §13.12. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Move an exact managed session target-first with fenced source shutdown and one owner is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
