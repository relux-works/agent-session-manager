# Integration Preconditions — CR revision 7

Task: `TASK-260830-2xt6fd`  
Story: `STORY-260830-35dbcs`  
Change Request: `CR-TASK-260830-2xt6fd-7`  
Checked: 2026-09-25

## Preconditions

- `task-board q 'get(TASK-260830-2xt6fd) { id name status }'` exited 0 and reports task status `integrating`.
- `task-board worktree status STORY-260830-35dbcs --json` exited 0 and reports CR revision 7 as `accepted`, with candidate tree `3a65113e8e3e4b5700cdce2fc4248db505c99f01` and base `eb12183056eb86dba663540518c6ac8f79f56a87`.
- `task-board worktree integrating` exited 0 and classifies `TASK-260830-2xt6fd` revision 7 as `awaiting_landing` (`tree=no`, `delta=present`).
- After `git fetch origin` (exit 0), `origin/main` is `eb12183056eb86dba663540518c6ac8f79f56a87`, equal to the CR base; it has not moved past that base.
- `git config --get branch.main.merge` exited 0 and reports `refs/heads/main`.
- Current `HEAD` is `1121b4ab54709a9966fbbcc359230d15b0e7592b` on `task-board/story/STORY-260830-35dbcs`.

## Worktree tree proof

The live worktree tree was calculated without writing the real Git index:

1. `GIT_INDEX_FILE=<worktree>/.temp/TASK-260830-2xt6fd/integration.index git read-tree HEAD` — exit 0.
2. `GIT_INDEX_FILE=<worktree>/.temp/TASK-260830-2xt6fd/integration.index git add -A` — exit 0; `GIT_INDEX_FILE` pointed to the task-scoped scratch index.
3. `GIT_INDEX_FILE=<worktree>/.temp/TASK-260830-2xt6fd/integration.index git write-tree` — exit 0; tree `3a65113e8e3e4b5700cdce2fc4248db505c99f01`.

This equals the accepted CR candidate tree. No source changes were made during this integration-preconditions check. The runner owns checkpoint/integration and the status transition to `done`.

## Supporting command captures

Raw command captures are bundled in `TASK-260830-2xt6fd_integration-support-rev7.tar.gz`: task/worktree status, landing classification, fetch, HEAD, origin/main, branch merge configuration, and scratch-index tree output. `task-board --version` and `git --version` readiness checks both exited 0; their captured task-board output is also included.
