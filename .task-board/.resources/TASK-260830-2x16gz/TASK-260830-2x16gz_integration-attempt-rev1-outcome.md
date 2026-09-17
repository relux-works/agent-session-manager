# Integration attempt record — TASK-260830-2x16gz rev1 (RUN-260917-1542c6)

- Task: TASK-260830-2x16gz (test-peer-auth-and-hostile-network-failures, FINAL leaf of STORY-260830-1kiyj6, story_final)
- Change Request: CR-TASK-260830-2x16gz-1 revision 1 (accepted by RUN-260917-f36c51)
- CR base: `e4e3e8834675cf3814effd3b2673b4931dfbf311`
- Run: RUN-260917-1542c6 (developer / implementer archetype, bound producer-role integration run)
- Date (UTC): 2026-09-17T12:57:33Z
- Control root: /Users/iv/Developer/ReluxWorks/agent-session-manager

Naming note: the brief's step 5 asks for an outcome named
`TASK-260830-2x16gz_integration-attempt-rev1.md`, but that resource name is
already taken by the input brief itself. This record is attached as
`TASK-260830-2x16gz_integration-attempt-rev1-outcome.md` so the brief is not
overwritten.

## 1. Trunk tip confirmation

```
$ git fetch origin
$ git rev-parse origin/main
fc67abdfc3888d6683b1eb89bc248e089ca6aca3
```

Expected `fc67abdfc3888d6683b1eb89bc248e089ca6aca3` → TIP_CONFIRMED (exit 0).

Trunk advanced past the CR base with 119 non-board paths, including paths this
CR also changes (`README.md`, `LOGBOOK.md`,
`internal/traceability/ownership.v0.6.0.json`,
`internal/traceability/traceability.go`,
`internal/traceability/traceability_test.go`,
`internal/traceability/cmd/tracecheck/main_test.go`, `task-board.config.json`).

## 2. Integration command (run exactly once)

```
$ task-board worktree integrate STORY-260830-1kiyj6 --cr TASK-260830-2x16gz --revision 1 --commit-time "2026-09-17T12:57:33Z"
```

Verbatim output:

```
integration_base_moved: the accepted Change Request is stale: trunk advanced with a change to LOGBOOK.md, which this Change Request also changes; no one has looked at the combination
  cr_id: CR-TASK-260830-2x16gz-1
  story_id: STORY-260830-1kiyj6
```

Exit code: 1 (typed refusal `integration_base_moved`, the expected result).

The success branch (step 3 of the brief: signature verification, validation
suite, delivery-branch push, PR creation) was therefore NOT executed. No
commits were created, nothing was pushed, no PR was opened.

## 3. Transaction state

```
$ task-board worktree transaction show STORY-260830-1kiyj6
No integration transaction is recorded for STORY-260830-1kiyj6
```

Exit code: 0. The refusal left no recorded transaction.

## 4. Board state left untouched

- Task TASK-260830-2x16gz status: `integrating` (unchanged; set/confirmed at run start).
- CR-TASK-260830-2x16gz-1 rev 1: accepted / stale-by-refusal (demotion to stale
  is the operator's `worktree invalidate-acceptance` step, explicitly out of
  scope for this run).
- No file was edited, no refresh was run, no status was changed by this run
  beyond the run-start `integrating` confirmation.

## 5. Conclusion

The §6.2 stale-base precondition is now evidenced: the accepted CR cannot land
on current trunk without a reviewed refresh/republish cycle. Next operator
steps (not taken here): `worktree invalidate-acceptance`, then the
refresh/republish cycle per the republish brief.
