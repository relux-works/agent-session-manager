CHECKPOINT RUN for the independently accepted Change Request revision 3 of TASK-260830-2ya5le, the first, non-final leaf of STORY-260830-21bxa3. Reviewer RUN-260923-61c37e (codex gpt-6-sol medium) accepted it with candidate tree `9870fb42e454a1163b4fc6abd778210260cf84d9` on base `0ca3e4c`, verdict `TASK-260830-2ya5le_review-verdict-rev3.md`. You are the bound producer-role integration run. Follow the runtime's Integration Assignment: confirm the landing preconditions, attach fresh evidence, and end. The runner performs `worktree checkpoint` itself after your turn. Do not run checkpoint or integrate yourself, do not edit or commit anything, and do not call handoff or set status.

Preconditions to confirm and record:
- `task-board worktree status STORY-260830-21bxa3 --json` shows CR rev3 accepted, and its candidate tree equals `9870fb42`. Check the live worktree tree through a scratch GIT_INDEX_FILE and never write the real index.
- `git fetch origin` shows `origin/main` = `0ca3e4c`, or any move is recorded.

Attach them as `TASK-260830-2ya5le_checkpoint-preconditions-rev3.md` and exit. Canonical CLI: /Users/iv/.curator/global/bin/task-board.
