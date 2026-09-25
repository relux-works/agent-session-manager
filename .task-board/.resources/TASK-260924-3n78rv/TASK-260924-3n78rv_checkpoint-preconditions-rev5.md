# TASK-260924-3n78rv checkpoint preconditions, revision 5

Integration preconditions confirmed for accepted Change Request `CR-TASK-260924-3n78rv-5`.

- `task-board worktree status STORY-260830-21bxa3 --json` reported CR revision 5 as `accepted`, with `candidate_tree_oid` `a5583e589d9ac3b6eaa2ed5504a25959c38970f7`.
- The live Story worktree is on `task-board/story/STORY-260830-21bxa3`; `HEAD` and the recorded workspace checkpoint are both `618d78de05450c40ec39df7c97073a6feecc8f81`.
- The candidate tree was recomputed without writing the real index. A scratch index at `.temp/TASK-260924-3n78rv/checkpoint-evidence/index` was initialized from `HEAD`, updated from the worktree with `git add -A -- .`, and read with `git write-tree`. Result: `a5583e589d9ac3b6eaa2ed5504a25959c38970f7`, exactly matching the accepted CR candidate tree.
- `git fetch origin` succeeded. `git rev-parse origin/main` returned `6d3bff999f75ed8585a7ef32074ab108f364ec51`, matching the board's current integration base and the assignment's expected trunk.
- The board task remains `integrating`. No checkpoint, integrate, handoff, or commit command was run.

Commands and observed values:

```text
task-board worktree status STORY-260830-21bxa3 --json
CR-TASK-260924-3n78rv-5: state=accepted, revision=5
candidate_tree_oid=a5583e589d9ac3b6eaa2ed5504a25959c38970f7
workspace checkpoint_oid=618d78de05450c40ec39df7c97073a6feecc8f81

GIT_INDEX_FILE=.temp/TASK-260924-3n78rv/checkpoint-evidence/index git write-tree
a5583e589d9ac3b6eaa2ed5504a25959c38970f7

git fetch origin
(exit code 0)
git rev-parse origin/main
6d3bff999f75ed8585a7ef32074ab108f364ec51
```
