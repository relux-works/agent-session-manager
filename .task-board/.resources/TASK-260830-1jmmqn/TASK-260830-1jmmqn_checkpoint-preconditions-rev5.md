# TASK-260830-1jmmqn checkpoint preconditions — CR revision 5

Run: RUN-260924-4d64b1  
Date: 2026-09-24

## Preconditions

- `task-board worktree status STORY-260830-21bxa3 --json` exited 0. It reports `CR-TASK-260830-1jmmqn-5`, revision 5, state `accepted`, producer role `developer`, candidate tree `33fc5a45bfc45c7da2da4a2297a3d66fa4166887`, checkpoint `547ea289e411bea803f4cd47b6314e4bb558ceb0`, and the active Story worktree.
- The live worktree branch tip is `547ea289e411bea803f4cd47b6314e4bb558ceb0`; its candidate changes are uncommitted. The board reports `dirty: true`.
- Recomputed the live candidate tree with a scratch `GIT_INDEX_FILE` (`git read-tree HEAD`, `git add -A`, `git write-tree`). The resulting tree was `33fc5a45bfc45c7da2da4a2297a3d66fa4166887`, matching the accepted CR candidate tree exactly. The real index was not used for this computation.
- `git fetch origin` exited 0. The subsequent `git rev-parse origin/main` reported `6d3bff999f75ed8585a7ef32074ab108f364ec51`. Thus `origin/main` has moved from the checkpoint brief's `0ca3e4c26e2b275212796657f785b9b450f6174e`; the move is recorded here. The board status also reports the protected authority and selected base at `6d3bff9`.

No source files or commits were created in this integration run. No checkpoint, integration, or generic handoff command was run. Spawn directives were checked; none were recorded.
