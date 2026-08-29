# TASK-260830-3uzfyn: implement-profile-resolution-and-persistence

## Description
Resolve named profiles, CLI overrides, task-board profiles, provider argv/env, and persist effective non-secret profile state. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §2.4, §5.5, §7.5-7.7, §8. Work only inside the provider-identity-profile-and-resume story boundary.

## Acceptance Criteria
Production behavior demonstrates: Resolve named profiles, CLI overrides, task-board profiles, provider argv/env, and persist effective non-secret profile state. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
