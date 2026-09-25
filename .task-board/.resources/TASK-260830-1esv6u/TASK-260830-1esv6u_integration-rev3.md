INTEGRATION RUN for the independently accepted Change Request revision 3 of TASK-260830-1esv6u, the FINAL (story_final) leaf of STORY-260830-21bxa3 (fidelity-projection-and-validation-contracts). Reviewer RUN-260925-8eba10 (codex gpt-6-sol medium) accepted it with candidate tree `0409e8ff8001e7848112e23aed6075c22bcbae5e` on base `5b7876b`; the verdict is `TASK-260830-1esv6u_review-verdict-rev3.md`. The Story branch carries the signed checkpoints of 2ya5le, 1jmmqn, 3n78rv and 2zboyz. The squash carries all five leaves.

You are the bound producer-role integration run. Follow the runtime's Integration Assignment exactly. Confirm the landing preconditions, attach fresh evidence, and end your turn. The runner then performs `worktree integrate` itself. This project now configures `version_control.commit_time_policy: {kind: now}`, so the runner's integrate needs no `--commit-time`. Do NOT run integrate, checkpoint, git commit or push yourself. Do not call handoff or set status.

Confirm and record in `TASK-260830-1esv6u_integration-preconditions-rev3.md`:
- `task-board worktree status STORY-260830-21bxa3 --json` shows CR rev3 accepted, and the live worktree tree, taken through a scratch GIT_INDEX_FILE, equals `0409e8ff8001e7848112e23aed6075c22bcbae5e`. Never write the real index.
- `task-board worktree integrating` classifies it as awaiting_landing.
- After `git fetch origin`, record origin/main. If it moved past `5b7876b`, note it; the integrate reparents path-disjoint moves.
- `git config branch.main.merge` is `refs/heads/main`.

Canonical CLI: /Users/iv/.curator/global/bin/task-board.
