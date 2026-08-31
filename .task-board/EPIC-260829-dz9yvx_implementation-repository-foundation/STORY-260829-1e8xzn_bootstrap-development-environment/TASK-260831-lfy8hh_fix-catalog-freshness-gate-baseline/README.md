# TASK-260831-lfy8hh: fix-catalog-freshness-gate-baseline

## Description
The configured catalog freshness command uses git diff --exit-code, which compares the working tree against the git index. In CI that is correct because the tree is clean, but a managed Story worktree carries legitimate unstaged changes, so the gate falsely refuses a generated file that already matches its sources. It blocked two consecutive producer runs on TASK-260830-236x9n whose generated output was provably current.

## Scope
One command in spawn.worktree_isolation.validation.commands in task-board.config.json.

## Acceptance Criteria
The freshness gate passes on a tree whose generated file matches its sources regardless of git staging state, fails when the generated file genuinely drifts from its sources, and leaves the working tree unchanged.
