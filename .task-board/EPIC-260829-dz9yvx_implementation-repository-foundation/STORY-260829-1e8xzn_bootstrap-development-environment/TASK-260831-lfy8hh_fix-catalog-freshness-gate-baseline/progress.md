## Status
to-dev

## Review
required

## Task Class
metadata

## Estimate
notEstimated

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
(empty)

## Notes
The freshness gate compared the working tree against the git index, which is only equivalent to the intended property in a clean CI checkout. In a managed Story worktree producer changes are unstaged by construction, so the gate falsely refused a generated file that regeneration provably does not change. Replaced with a snapshot-and-compare form that tests regeneration idempotence directly and restores the tree. Verified: passes on a correct unstaged tree where the old form failed, still fails on genuine drift. This gate was authored by the Orchestrator in TASK-260831-3e99as and is the same false-refusal shape reviews on this board reject in production code.

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260831-lfy8hh_gate-defect-and-fix.md](file://TASK-260831-lfy8hh/TASK-260831-lfy8hh_gate-defect-and-fix.md) — Measured false-refusal defect in the catalog freshness gate, the corrected command, and its positive/negative verification

## Created
2026-08-31T16:16:13Z

## Last Update
2026-08-31T16:17:11Z

## Assigned To
claude-orchestrator
