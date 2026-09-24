INTEGRATION RUN for the independently accepted Change Request revision 6 of TASK-260830-g0pcnt, the FINAL (story_final) leaf of STORY-260830-2t4g7i (production-tmux-backend). Reviewer RUN-260924-d2d2e0 (claude-opus-5-5 low) accepted it with candidate tree `bc9bb130f49e74e14a27a440ad949aa71b87c37b` on base `0ca3e4c`; the verdict is `TASK-260830-g0pcnt_review-verdict-rev6.md`. The Story branch carries the signed checkpoints of 35urbp, 1c28dz and vcx6yo. The squash carries all four leaves.

You are the bound producer-role integration run. Follow the runtime's Integration Assignment exactly. Confirm the landing preconditions, attach fresh evidence, and end your turn. The runner then performs `worktree integrate` itself. This project now configures `version_control.commit_time_policy: {kind: now}`, so the runner's integrate needs no `--commit-time`. Do NOT run integrate, checkpoint, git commit or push yourself. Do not call handoff or set status.

Confirm and record in `TASK-260830-g0pcnt_integration-preconditions-rev6.md`:
- `task-board worktree status STORY-260830-2t4g7i --json` shows CR rev6 accepted, and the live worktree tree, taken through a scratch GIT_INDEX_FILE, equals `bc9bb130f49e74e14a27a440ad949aa71b87c37b`. Never write the real index.
- `task-board worktree integrating` classifies it as awaiting_landing.
- After `git fetch origin`, record origin/main. If it moved past `0ca3e4c`, note it; the integrate reparents path-disjoint moves.
- `git config branch.main.merge` is `refs/heads/main`.

Canonical CLI: /Users/iv/.curator/global/bin/task-board.
