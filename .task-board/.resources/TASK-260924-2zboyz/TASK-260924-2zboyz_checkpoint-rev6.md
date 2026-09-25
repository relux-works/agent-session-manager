CHECKPOINT RUN for the independently accepted Change Request revision 6 of TASK-260924-2zboyz, the fourth, non-final leaf of STORY-260830-21bxa3. Reviewer RUN-260924-2a3e9f (codex gpt-6-sol medium) accepted it with candidate tree `cf13fcb5aba321d93e6fcd58418886da22fd69ef` on base `5e40826` (Story checkpoint), verdict `TASK-260924-2zboyz_review-verdict-rev6.md`. You are the bound producer-role integration run. Follow the runtime's Integration Assignment: confirm the landing preconditions, attach fresh evidence, and end. The runner performs `worktree checkpoint` itself after your turn. Do not run checkpoint or integrate yourself, do not edit or commit anything, and do not call handoff or set status.

Preconditions to confirm and record:
- `task-board worktree status STORY-260830-21bxa3 --json` shows CR rev6 accepted, and its candidate tree equals `cf13fcb5`. Check the live worktree tree through a scratch GIT_INDEX_FILE and never write the real index.
- `git fetch origin` shows `origin/main` = `6cfe6af` (moved; checkpoint does not require descent), or any move is recorded.

Attach them as `TASK-260924-2zboyz_checkpoint-preconditions-rev6.md` and exit. Canonical CLI: /Users/iv/.curator/global/bin/task-board.
