# TASK-260830-g0pcnt Integration Preconditions — CR rev6

Observed: 2026-09-24T06:41:19Z

## Preconditions

- `task-board worktree status STORY-260830-2t4g7i --json` exited 0. CR `CR-TASK-260830-g0pcnt-6` is revision 6, state `accepted`, base `0ca3e4c26e2b275212796657f785b9b450f6174e`, candidate tree `bc9bb130f49e74e14a27a440ad949aa71b87c37b`. The active workspace checkpoint is `5a64077facb9f93357b9bbc8ebdba93069105715` and the lease is held by this integration run.
- `task-board worktree integrating` exited 0 and classifies `TASK-260830-g0pcnt` revision 6 as `awaiting_landing`.
- The worktree candidate tree was independently materialized without touching the real index. A scratch index at `.temp/TASK-260830-g0pcnt/scratch.index` was initialized from `HEAD`, staged the working tree with `GIT_INDEX_FILE` set to that scratch path, and `git write-tree` returned `bc9bb130f49e74e14a27a440ad949aa71b87c37b`, exactly matching the accepted CR candidate tree.
- `git fetch origin` exited 0. Afterwards `git rev-parse refs/remotes/origin/main` returned `0ca3e4c26e2b275212796657f785b9b450f6174e`; `origin/main` has not moved past the accepted CR base.
- `git config branch.main.merge` exited 0 and returned `refs/heads/main`.

## Command results

| Check | Result |
| --- | --- |
| Board CR state | accepted, revision 6 |
| Worktree integrating classification | awaiting_landing |
| Accepted candidate tree vs scratch-index tree | exact match: `bc9bb130f49e74e14a27a440ad949aa71b87c37b` |
| Fetched `origin/main` | `0ca3e4c26e2b275212796657f785b9b450f6174e` (same as CR base) |
| `branch.main.merge` | `refs/heads/main` |

No checkpoint, integration, commit, push, or producer handoff command was run. The runner owns the landing transaction.
