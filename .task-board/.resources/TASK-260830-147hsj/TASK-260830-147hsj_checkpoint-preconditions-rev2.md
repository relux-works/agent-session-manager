# Checkpoint preconditions — CR revision 2

Task: `TASK-260830-147hsj`  
Story workspace: `STORY-260830-ub60id`  
Integration run: `RUN-260924-e43ebf`  
Observed: `2026-09-24 04:24:57 UTC`

## Confirmed state

- `task-board q 'get(TASK-260830-147hsj) { id status }'` exited 0 and reported `integrating`.
- `task-board worktree status STORY-260830-ub60id --json` exited 0. It reports `CR-TASK-260830-147hsj-2`, revision 2, state `accepted`, kind `task_delta`, and candidate tree `af575645ae5704570dd9dbbde87e2fd47c36af8e`. The workspace checkpoint and branch tip are both `b81258e3c5bc0f6321ae8f6bea81ec24cc298d70`.
- The live worktree tree was recomputed using a scratch index at `.temp/TASK-260830-147hsj/integration.index`; `git write-tree` exited 0 with `af575645ae5704570dd9dbbde87e2fd47c36af8e`, equal to the accepted candidate tree. The real index was not used for writes.
- `git status --short && git rev-parse HEAD` exited 0. The worktree is dirty with the accepted candidate paths, and `HEAD` remains the checkpoint `b81258e3c5bc0f6321ae8f6bea81ec24cc298d70`.
- `git fetch origin && git rev-parse origin/main` exited 0 and reported `0ca3e4c26e2b275212796657f785b9b450f6174e`, matching the Story workspace's recorded current base.

## Integration boundary

This run only verified and recorded the preconditions. It did not change task status, run `worktree checkpoint`, run `worktree integrate`, or execute tests/builds. The runner owns the synchronous checkpoint/integration transaction after this handoff.
