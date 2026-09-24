# Integration preconditions — TASK-260830-2h5uv9 revision 5

Observed at 2026-09-24T22:24:38Z by integration run `RUN-260924-afb950` in the managed Story worktree.

## Accepted candidate

`task-board worktree status STORY-260830-ub60id --json` exited 0. It reports `CR-TASK-260830-2h5uv9-5` at revision 5 with state `accepted`, kind `story_final`, candidate tree `6d76b8d4c6dbdeb6f7716071ad578dda9ff1534d`, base `6d3bff999f75ed8585a7ef32074ab108f364ec51`, and checkpoint `86a6188a0e35fd04a24eba62a01b71a54d6faa5d`.

The live candidate tree was computed without writing the real index. Commands:

```sh
mkdir -p .temp/TASK-260830-2h5uv9
GIT_INDEX_FILE="$PWD/.temp/TASK-260830-2h5uv9/integration-index" git read-tree HEAD
GIT_INDEX_FILE="$PWD/.temp/TASK-260830-2h5uv9/integration-index" git add -A
GIT_INDEX_FILE="$PWD/.temp/TASK-260830-2h5uv9/integration-index" git write-tree
```

The scratch-index `git write-tree` exited 0 and returned `6d76b8d4c6dbdeb6f7716071ad578dda9ff1534d`, equal to the accepted candidate tree. The real index was not written.

## Landing classification and trunk

`task-board worktree integrating` exited 0 and classified `TASK-260830-2h5uv9` revision 5 as `awaiting_landing`.

`git fetch origin` exited 0. Afterwards, `git rev-parse origin/main` exited 0 and returned `6d3bff999f75ed8585a7ef32074ab108f364ec51`, the same as the recorded base; origin/main had not moved past `6d3bff9`.

A post-fetch `task-board worktree integrating` exited 0 and continued to classify revision 5 as `awaiting_landing` with protected `refs/heads/main` at `6d3bff999f75ed8585a7ef32074ab108f364ec51`.

`git config branch.main.merge` exited 0 and returned `refs/heads/main`.

The required status mutation exited 0 and reported old and new status both `integrating`.
