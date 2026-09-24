REFRESH-AND-HANDOFF CONTINUATION for TASK-260830-g0pcnt revision 2. This brief SUPERSEDES the "Do NOT refresh for it" sentence in `TASK-260830-g0pcnt_rework-rev2.md`. That sentence was the orchestrator's error. `worktree integrate` does reparent a path-disjoint trunk move, but Change Request CONSTRUCTION requires the Story checkpoint to descend from the CURRENT trunk. So three runs finished the rework and still failed at construction:
- RUN-260923-7aa19c
- RUN-260923-47cac8
- RUN-260923-184b38 (cancelled by the orchestrator)

Each refusal was:

`change_request_base_authority_mismatch: checkpoint 8d944c7 does not descend from selected authority 360c8bd`

THE REWORK IS DONE. Do not redo it. The uncommitted candidate in `.temp/STORY-260830-2t4g7i/worktree` is the finished rev2: the gate × entry × component × bit census, the mask-narrowing mutants, 192 importer rows unchanged, 0 moved, and the digest `3664ab2f…` rederived. A backup is kept at `refs/backup/g0pcnt-rw2-candidate-*` (tree `d8c9cde9`). Read it; never move it.

DO EXACTLY THIS:
1. Run `task-board worktree refresh-candidate TASK-260830-g0pcnt`. The trunk delta `14d636e..360c8bd` touches only `task-board.config.json`, which the Story never changes, so the replay of the three checkpoints should not conflict. If it refuses, record the typed refusal verbatim in notes and in an outcome, then stop. No manual rebase, no git reset.
2. Check the invariant after the refresh. The new Story checkpoint must descend from `360c8bd`, and the three replayed checkpoints must be signed. Every path in `git diff --name-only c9233ce 360c8bd` that the Story does not change must be blob-equal to `360c8bd` in the candidate tree; that includes `task-board.config.json`, which now carries `commit_time_policy` and `board_publication` (see task-board #346/#321: refresh can leave stale trunk copies). Compare through a scratch `GIT_INDEX_FILE`. The candidate's own 24 changed paths must be byte-equal to the backup tree `d8c9cde9`.
3. Rerun only the checks the base move can affect:
   - tracecheck, verifying the digest pin is unchanged;
   - the README coverage pin test;
   - `go build ./...` and `go vet ./...`;
   - `GOOS=windows GOARCH=amd64 go vet ./...`;
   - the tmuxserver and termbind package tests.
   Attach a short `TASK-260830-g0pcnt_refresh-rev2-evidence.md` outcome with these results and the new checkpoint OIDs.
4. Keep the checklist current, then run `task-board handoff TASK-260830-g0pcnt --role developer` and exit.

LIVE-INDEX RULE: write the real index only through the tool and the handoff. Canonical CLI: /Users/iv/.curator/global/bin/task-board. Model: gpt-6-luna max.
