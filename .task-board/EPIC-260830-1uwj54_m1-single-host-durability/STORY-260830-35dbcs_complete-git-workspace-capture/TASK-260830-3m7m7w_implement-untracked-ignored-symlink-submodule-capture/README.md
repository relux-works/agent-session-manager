# TASK-260830-3m7m7w: implement-untracked-ignored-symlink-submodule-capture

## Description
Capture policy-allowed untracked/ignored content, symlinks, submodule state, large blobs, sparse/linked worktrees, and exclusions. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §10.2-10.4, §12.1-12.3. Work only inside the complete-git-workspace-capture story boundary.

## Acceptance Criteria
Production behavior demonstrates: Capture policy-allowed untracked/ignored content, symlinks, submodule state, large blobs, sparse/linked worktrees, and exclusions. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
