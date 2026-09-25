# Checkpoint preconditions — TASK-260924-2zboyz rev6

Observed at 2026-09-24 22:39:30 UTC in run RUN-260924-f127cf.

## Accepted Change Request

`task-board worktree status STORY-260830-21bxa3 --json` returned CR `CR-TASK-260924-2zboyz-6` at revision 6 with state `accepted`.

- Recorded base OID: `5e40826725a0880897d6bd8321c8fa8e062620d3`
- Accepted candidate tree OID: `cf13fcb5aba321d93e6fcd58418886da22fd69ef`
- Recorded index tree OID: `a5583e589d9ac3b6eaa2ed5504a25959c38970f7`
- CR diff SHA-256: `e0f422e67753c29fc0562f388a180dd9bb14674c473501efee00292621ce4fae`

## Live worktree tree

The live worktree is branch `task-board/story/STORY-260830-21bxa3`, at HEAD `5e40826725a0880897d6bd8321c8fa8e062620d3` (HEAD tree `a5583e589d9ac3b6eaa2ed5504a25959c38970f7`). Its candidate tree was recomputed without writing the real index:

1. `GIT_INDEX_FILE=.temp/TASK-260924-2zboyz/integration/scratch-index-RUN-260924-f127cf git read-tree HEAD`
2. `GIT_INDEX_FILE=.temp/TASK-260924-2zboyz/integration/scratch-index-RUN-260924-f127cf git add -A`
3. `GIT_INDEX_FILE=.temp/TASK-260924-2zboyz/integration/scratch-index-RUN-260924-f127cf git write-tree`

The resulting tree was `cf13fcb5aba321d93e6fcd58418886da22fd69ef`, matching the accepted candidate exactly. `git status --short` reports only `?? internal/clonereadback/`; no commit was created. All index writes above targeted the ignored scratch index under `.temp/`.

## Fresh upstream observation

`git fetch origin` exited 0. After the fetch, `git rev-parse origin/main` returned `5b7876be29cf578ff02583660d77fc10944e855c`.

The workspace status still records its earlier upstream/protected-authority observation as `6cfe6af46efc579003f6f4524dc5d75b5e5379bc`. Thus `origin/main` has moved since that observation; the fresh OID is recorded here. The integration assignment says checkpointing does not require this checkpoint to descend from the new trunk tip.
