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
Added the repository task-board.config.json: local board dir, version_control confirm with AX real-time commit policy (this repository does not backdate), spawn ceilings and all eleven workload classes pinned to codex/gpt-5.6-sol/high, a 13-command worktree_isolation validation suite, and feature toggles. Every command was probed on the accepted candidate tree and each gating command was proven to kill a mutant. curator status --check is deliberately excluded: Curator-managed paths are gitignored, so a managed Story worktree never contains them and the command exits 1 there. The JSON gate was narrowed from a find-based scan to git ls-files after the find form was measured scanning 287 working-tree files including gitignored scratch artifacts and failing with an xargs argument-length error; the narrowed form was proven to pass on tracked JSON, fail on a broken tracked file, and ignore untracked junk. Delivered as a tracked inline change rather than through a Story worktree because integrating it would require the landing gate it introduces.

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260831-3e99as_gate-design-evidence.md](file://TASK-260831-3e99as/TASK-260831-3e99as_gate-design-evidence.md) — Empirical probe of every configured landing-gate command on the accepted candidate tree, with a killed mutant for each gating command and the recorded reason curator status --check is excluded
- [TASK-260831-3e99as_suite-on-landing-head.log](file://TASK-260831-3e99as/TASK-260831-3e99as_suite-on-landing-head.log) — All 13 configured validation commands executed on the landing head; 13/13 exit 0

## Created
2026-08-31T10:22:57Z

## Last Update
2026-08-31T10:33:42Z

## Assigned To
claude-orchestrator
