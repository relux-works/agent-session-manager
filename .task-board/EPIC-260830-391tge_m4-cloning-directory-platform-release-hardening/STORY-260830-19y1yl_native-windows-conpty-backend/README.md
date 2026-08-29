# STORY-260830-19y1yl: native-windows-conpty-backend

## Description
Implement implement native Windows PowerShell, process supervision, storage, SSH, and ConPTY without claiming tmux durability according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §4.3, §19.2. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Implement native windows powershell, process supervision, storage, ssh, and conpty without claiming tmux durability is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
