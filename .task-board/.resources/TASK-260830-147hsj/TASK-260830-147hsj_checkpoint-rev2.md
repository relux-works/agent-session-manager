CHECKPOINT RUN for the independently accepted Change Request revision 2 of TASK-260830-147hsj, the second, non-final leaf of STORY-260830-ub60id. Reviewer RUN-260924-19d34f (claude-opus-5-5 low) accepted it with candidate tree `af575645ae5704570dd9dbbde87e2fd47c36af8e` on base `b81258e` (Story checkpoint over trunk 0ca3e4c), verdict `TASK-260830-147hsj_review-verdict-rev2.md`. You are the bound producer-role integration run. Follow the runtime's Integration Assignment: confirm the landing preconditions, attach fresh evidence, and end. The runner performs `worktree checkpoint` itself after your turn. Do not run checkpoint or integrate yourself, do not edit or commit anything, and do not call handoff or set status.

Preconditions to confirm and record:
- `task-board worktree status STORY-260830-ub60id --json` shows CR rev2 accepted, and its candidate tree equals `af575645`. Check the live worktree tree through a scratch GIT_INDEX_FILE and never write the real index.
- `git fetch origin` shows `origin/main` = `0ca3e4c`, or any move is recorded.

Attach them as `TASK-260830-147hsj_checkpoint-preconditions-rev2.md` and exit. Canonical CLI: /Users/iv/.curator/global/bin/task-board.
