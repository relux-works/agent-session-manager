# TASK-260830-2bnr39: implement-git-repository-and-index-snapshot

## Description
Capture repository identity, HEAD/ref, worktree metadata, index stages/flags, staged and unstaged deltas, and file modes. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §10.2-10.4, §12.1-12.3. Work only inside the complete-git-workspace-capture story boundary.

## Acceptance Criteria
Production behavior demonstrates: Capture repository identity, HEAD/ref, worktree metadata, index stages/flags, staged and unstaged deltas, and file modes. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
