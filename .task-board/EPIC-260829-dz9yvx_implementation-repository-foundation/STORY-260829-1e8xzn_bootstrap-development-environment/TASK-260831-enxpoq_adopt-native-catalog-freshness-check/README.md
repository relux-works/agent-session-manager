# TASK-260831-enxpoq: adopt-native-catalog-freshness-check

## Description
Two independent fixes to the same catalog freshness defect collided. Trunk carries the Orchestrator mktemp/diff snapshot form; the in-flight STORY-260830-2wdm5e checkpoint carries a native cataloggen -check flag that validates generated output without temp files or git semantics. The textual collision on task-board.config.json aborted the Story base replay with a merge conflict.

## Scope
One command in spawn.worktree_isolation.validation.commands, aligned byte-for-byte with the in-flight Story checkpoint.

## Acceptance Criteria
Command 8 equals the Story checkpoint text exactly so the base replay applies cleanly; the native -check form is proven to pass on a current candidate and to fail on injected drift.
