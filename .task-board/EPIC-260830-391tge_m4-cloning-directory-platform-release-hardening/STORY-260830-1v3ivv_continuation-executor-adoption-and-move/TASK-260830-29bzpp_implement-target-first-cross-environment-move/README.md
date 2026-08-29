# TASK-260830-29bzpp: implement-target-first-cross-environment-move

## Description
Commit/validate target before source stop and return cloned_source_still_active when post-commit release fails without deleting target. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §13.15, §14.5. Work only inside the continuation-executor-adoption-and-move story boundary.

## Acceptance Criteria
Production behavior demonstrates: Commit/validate target before source stop and return cloned_source_still_active when post-commit release fails without deleting target. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
