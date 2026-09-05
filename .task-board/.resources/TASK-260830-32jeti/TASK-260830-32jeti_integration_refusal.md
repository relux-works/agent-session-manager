# Integration refusal — TASK-260830-32jeti rev 7

Command: task-board worktree integrate STORY-260830-3jqsx1 --cr TASK-260830-32jeti --revision 7 --commit-time 2026-09-05T18:42:46Z
Exit code: 1

Verbatim output:
```
integration_base_moved: the accepted Change Request is stale: trunk advanced with a change to LOGBOOK.md, which this Change Request also changes; no one has looked at the combination
  cr_id: CR-TASK-260830-32jeti-7
  story_id: STORY-260830-3jqsx1
```

Post-check (read-only):
- TASK-260830-32jeti status: integrating
- STORY-260830-3jqsx1 status: integrating
- No commit created by this run; no files changed; worktree retained with uncommitted accepted work intact.
- This is NOT the known wrong-integration-owner failure mode; that string did not appear.
- Per guard rails: stopped, changed nothing, committed nothing, pushed nothing.
