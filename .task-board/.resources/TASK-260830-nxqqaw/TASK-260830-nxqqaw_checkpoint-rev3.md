CHECKPOINT RUN for the independently accepted Change Request revision 3 of TASK-260830-nxqqaw, the first, non-final leaf of STORY-260830-ub60id. Reviewer RUN-260923-78f0db (claude-opus-5-5 low) accepted it with candidate tree `de8741fcc0f1cda0bef3ab870dfad1c795e4d43b` on base `0ca3e4c`, verdict `TASK-260830-nxqqaw_review-verdict-rev3.md`. You are the bound producer-role integration run. Follow the runtime's Integration Assignment: confirm the landing preconditions, attach fresh evidence, and end. The runner performs `worktree checkpoint` itself after your turn. Do not run checkpoint or integrate yourself, do not edit or commit anything, and do not call handoff or set status.

Preconditions to confirm and record:
- `task-board worktree status STORY-260830-ub60id --json` shows CR rev3 accepted, and its candidate tree equals `de8741fc`. Check the live worktree tree through a scratch GIT_INDEX_FILE and never write the real index.
- `git fetch origin` shows `origin/main` = `0ca3e4c`, or any move is recorded.

Attach them as `TASK-260830-nxqqaw_checkpoint-preconditions-rev3.md` and exit. Canonical CLI: /Users/iv/.curator/global/bin/task-board.
