# TASK-260830-21gygk — refresh-candidate refusal (RUN-260909-fe37d7)

## Product work state

Complete. 8/8 AC rows driven at the shared library boundary; full suite,
vet, build, gofmt, coverage (91.0%), race, and 29/29 narrowing kills all
re-derived on the final candidate this run. Evidence:
`TASK-260830-21gygk_results.md`,
`TASK-260830-21gygk_evidence-fe37d7.tar.gz`. Candidate left UNCOMMITTED in
the Story worktree; no commit, branch, or registry mutation performed.

## Tool refusal (exact)

Command: `task-board worktree refresh-candidate TASK-260830-21gygk`
(run as the live tracked producer RUN-260909-fe37d7 holding the exclusive
STORY-260830-3tq4ns lease; working candidate unstaged; no CR exists).

Output (identical on two attempts):

```
isolated checkpoint replay retained at /Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/base-refresh/replay-<id>/worktree: git rebase -S --reapply-cherry-picks --empty=keep --onto 8cf4aaaa190e6a11dff2661aa6806ce476128653 7654d7cadb2c226bfa3db5f23ac355a730285eea: exit status 1: Rebasing (1/2)
Rebasing (2/2)
error: could not apply 7208cc7... TASK-260830-1r9wrr: TASK-260830-1r9wrr: implement-session-state-reducer
...
Could not apply 7208cc7... # TASK-260830-1r9wrr: TASK-260830-1r9wrr: implement-session-state-reducer
```

## Conflict characterization (read-only inspection of the retained replay)

- Exactly one conflicting path: `LOGBOOK.md` (`git diff --name-only
  --diff-filter=U` lists only it; the `internal/sessstate/` additions
  applied cleanly).
- Shape: append-append at the newest-first anchor. HEAD side contributes
  the 2026-09-09 v0.6.0 adoption blocks (TASK-260908-3kvnm2,
  TASK-260908-2tkufa); 7208cc7 side contributes the 2026-09-08 1r9wrr
  blocks. Both sides are purely additive log prose; no production code,
  test, pin, catalog, or traceability file conflicts.
- The managed worktree is untouched by the failed replay (HEAD still
  `7208cc7427e1ebe95106d34f98214e2bbea4ae1c`, same dirty set).

## Why not resolved here

Resolving inside the tool-owned replay workspace and continuing its
`signed` checkpoint rebase would hand-fabricate checkpoint records outside
the source-owned recovery path, against the explicit resume directive
("Never fabricate CRs, hand-commit Story checkpoints, edit registries or
reset/clean the worktree"; "A genuine refusal must be recorded and returned
with exact evidence rather than bypassed").

## Needed input (orchestrator or tool fix)

Either (a) source-owned replay with a LOGBOOK merge driver (both sides
additive: keep both blocks, newest-first), after which this producer can
re-run refresh-candidate and hand off normally; or (b) an explicit
orchestrator-owned conflict resolution and re-checkpoint. No product
rework is implied: the candidate content is final and fully evidenced.
