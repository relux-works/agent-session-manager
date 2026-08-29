# STORY-260830-bxaztj: macos-arm64-daily-driver-lane

## Description
Implement pass the production macOS arm64 tmux/Aqua lane according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §4.2, §4.4, §19.2. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Pass the production macos arm64 tmux/aqua lane is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
