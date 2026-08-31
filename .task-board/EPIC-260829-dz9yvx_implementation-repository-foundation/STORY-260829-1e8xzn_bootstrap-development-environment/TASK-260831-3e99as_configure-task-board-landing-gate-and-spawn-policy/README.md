# TASK-260831-3e99as: configure-task-board-landing-gate-and-spawn-policy

## Description
Add the missing repository task-board.config.json. AX currently has no sidecar config at all, which makes worktree integrate refuse validation_not_configured and leaves every spawn on the CLI default reasoning effort medium instead of the project ceiling.

## Scope
Repository-root task-board.config.json only: local board dir, version_control policy, spawn ceilings and workload classes pinned to codex/gpt-5.6-sol/high, spawn.worktree_isolation.validation.commands, and feature toggles. No production code delta.

## Acceptance Criteria
worktree integrate no longer refuses validation_not_configured; every configured validation command is proven to pass on the accepted candidate tree and the gating ones are proven to fail on an injected mutant; curator status --check is excluded with recorded evidence that Curator-managed paths are gitignored and therefore absent from a managed Story worktree; spawn ceilings and all workload classes resolve to codex/gpt-5.6-sol/high; the change lands as a signed reviewed fast-forward on main.
