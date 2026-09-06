# CR-shape repair report — TASK-260830-3bkz0c (RUN-260906-737631)

## Result

No new Change Request revision was constructed. The leaf remains `integrating`
with the accepted revision 4 unchanged. Both board moves that would return the
leaf to a producer-handoff state were refused with the integration-path guard.
Per the spawn brief, this refusal is reported verbatim and no workaround was
attempted. The tree was not touched.

## Latest (and only current) revision — revision 4

- revision: 4
- state: accepted
- kind: task_delta (wrong shape; `story_final` required for `worktree integrate`)
- repository_delta: empty
- base_oid: 82c38378fe79b3e2a3e9fa337c23643b856ba7ee (= the delivered commit; zero-byte self-diff)
- changed paths: 0

Corroborated live via `task-board worktree status STORY-260830-3drr2m`:

```
change-req: TASK-260830-3bkz0c rev 4 accepted (repository_delta=empty, 0 changed path(s))
```

Sibling leaves for context: TASK-260830-2z3se0 rev 7 checkpointed (present,
34 paths); TASK-260830-ljkj8r rev 3 checkpointed (present, 29 paths).

No revision 5 exists.

## Refusals (verbatim, in order attempted)

1. `task-board m 'set_status(TASK-260830-3bkz0c, status=development)'` — exit 1:

```
cannot move TASK-260830-3bkz0c from integrating to development outside the integration production path: accepted work remains integrating until integration evidence is applied
```

2. `task-board handoff TASK-260830-3bkz0c --role developer` — exit 1:

```
cannot move TASK-260830-3bkz0c from integrating to to-review outside the integration production path: accepted work remains integrating until integration evidence is applied
```

No other board mutation was attempted. In particular, `worktree
invalidate-acceptance` was not invoked: the CR is `accepted`, not stale under
§6.2 (trunk has not moved), so its precondition is not met.

## Tree state (verbatim)

`git rev-parse HEAD`:

```
82c38378fe79b3e2a3e9fa337c23643b856ba7ee
```

`git status --porcelain=v1` (empty — clean tree, exit 0):

```
(empty)
```

HEAD is unchanged from the brief's required value. No commit, amend, rebase,
cherry-pick, index touch, file edit, or `.task-board` write (other than this
outcome resource) was performed. `task-board q` confirms leaf status
`integrating`.

## What the orchestrator needs to decide

The kind resolver derives `story_final` only when closing the leaf would also
close the Story, and a new `story_final` revision can only be built from a
producer handoff — which is unreachable while the accepted `task_delta`
revision keeps the leaf in `integrating`. The reparenting of TASK-260906-33xcnc
fixed the *next* resolution, but there is no operator path from this state back
to a handoff without either (a) an integration-transaction disposition, or (b)
a privileged correction of the accepted revision's kind/base. Both are outside
this run's authority. No follow-up work was attached to STORY-260830-3drr2m.
