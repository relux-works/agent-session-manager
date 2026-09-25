INTEGRATION RUN for the independently accepted Change Request revision 7 of TASK-260830-2xt6fd, the FINAL (story_final) leaf of STORY-260830-35dbcs (complete-git-workspace-capture). Reviewer RUN-260925-dc1991 (claude-opus-5-5 low) accepted it with candidate tree `3a65113e8e3e4b5700cdce2fc4248db505c99f01` on base `eb12183`; the verdict is `TASK-260830-2xt6fd_review-verdict-rev7.md`. The Story branch carries the signed checkpoints of 2bnr39 and 3m7m7w (replayed onto current trunk). The squash carries all three leaves.

You are the bound producer-role integration run. Follow the runtime's Integration Assignment exactly. Confirm the landing preconditions, attach fresh evidence, and end your turn. The runner then performs `worktree integrate` itself. This project now configures `version_control.commit_time_policy: {kind: now}`, so the runner's integrate needs no `--commit-time`. Do NOT run integrate, checkpoint, git commit or push yourself. Do not call handoff or set status.

Confirm and record in `TASK-260830-2xt6fd_integration-preconditions-rev7.md`:
- `task-board worktree status STORY-260830-35dbcs --json` shows CR rev7 accepted, and the live worktree tree, taken through a scratch GIT_INDEX_FILE, equals `3a65113e8e3e4b5700cdce2fc4248db505c99f01`. Never write the real index.
- `task-board worktree integrating` classifies it as awaiting_landing.
- After `git fetch origin`, record origin/main. If it moved past `eb12183`, note it; the integrate reparents path-disjoint moves.
- `git config branch.main.merge` is `refs/heads/main`.

Canonical CLI: /Users/iv/.curator/global/bin/task-board.
