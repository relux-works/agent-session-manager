# STORY-260830-2sdjt3: daily-driver-security-and-release-gate

## Description
Implement prove the tmux-first slice is operationally safe enough for sustained use according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §16, §18-19. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Prove the tmux-first slice is operationally safe enough for sustained use is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
