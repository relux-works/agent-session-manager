# EPIC-260830-1uwj54: m1-single-host-durability

## Description
Deliver durable local AX sessions with exact ownership, workspace, provider, task-board, and tmux behavior.

## Scope
AX v0.5.0 M1; SPEC sections 3-5, 9-10, 12-15, 18, and M1 acceptance cases. macOS/Linux/WSL2 tmux target; no multi-host ownership transfer.

## Acceptance Criteria
A Codex or Claude managed session survives detach and process restart on one Unix host, preserves exact provider/workspace identity, and resumes only through fenced ax pane activation.
