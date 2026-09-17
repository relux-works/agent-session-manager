# TASK-260830-17ootk integration attempt rev1 — outcome

- Run: RUN-260917-e106c7 (developer, implementer archetype; bound integration run)
- Date (UTC): 2026-09-17T09:43:37Z (commit-time passed to integrate)
- CR: CR-TASK-260830-17ootk-1 revision 1, accepted on trunk base e4e3e8834675cf3814effd3b2673b4931dfbf311
- Control root: /Users/iv/Developer/ReluxWorks/agent-session-manager
- Canonical CLI: /Users/iv/.curator/global/bin/task-board

## Step 1 — fetch and confirm trunk

`git fetch origin` then `git rev-parse origin/main`:

    9e9fe5154f5e8d97151670c3f689821cc3431d79

Confirmed: origin/main equals the expected 9e9fe5154f5e8d97151670c3f689821cc3431d79
(STORY-260830-315721 landed since CR acceptance). Local main HEAD is identical.
Story branch tip unchanged: task-board/story/STORY-260830-2rqigd =
bde093e242b688d8348818706e4720fed3eec96d (3k3e6m checkpoint).

## Step 2 — integrate (exactly once)

Command:

    task-board worktree integrate STORY-260830-2rqigd --cr TASK-260830-17ootk \
      --revision 1 --commit-time "2026-09-17T09:43:37Z"

Exit code: 1

Verbatim output:

    integration_base_moved: the accepted Change Request is stale: trunk advanced with a change to LOGBOOK.md, which this Change Request also changes; no one has looked at the combination
      cr_id: CR-TASK-260830-17ootk-1
      story_id: STORY-260830-2rqigd

This is the expected typed refusal. No landing occurred, no commits were
created, and (per the brief) the success-path steps 3 (verify signatures,
validation suite, push delivery branch, open PR) were correctly NOT executed.
No push was made; main was not touched.

## Step 3 — success path

Not applicable: integrate refused, so no further landing steps were taken.

## Step 4 — transaction state

`task-board worktree transaction show STORY-260830-2rqigd` (exit 0):

    No integration transaction is recorded for STORY-260830-2rqigd

Additional read-only observations:

- `task-board q get(TASK-260830-17ootk)` reports status `integrating`.
- `task-board worktree obligations` does not list TASK-260830-17ootk (the CR is
  owned by this live run, so no row is expected there). Whether the §6.2
  stale demotion was recorded is therefore reported as unknown, not inferred.

## Step 5 — hygiene

- No file was edited, no commit created, no status changed by this run
  (the opening set_status to integrating was a no-op: old_value == new_value).
- `invalidate-acceptance`, refresh, and any rework were NOT run, per the brief.
- The `.task-board/.activity/*.ndjson` working-tree modifications visible in
  `git status` are board runtime artifacts, not edits made by this run.

## Conclusion

The integration attempt produced exactly the expected result:
`integration_base_moved` on LOGBOOK.md, exit 1, no transaction recorded.
The demotion-to-stale precondition for the operator refresh/republish cycle is
the integrate command own refusal record; the operator step
(`worktree invalidate-acceptance` + refresh/republish) is out of scope for
this run and was left untouched.
