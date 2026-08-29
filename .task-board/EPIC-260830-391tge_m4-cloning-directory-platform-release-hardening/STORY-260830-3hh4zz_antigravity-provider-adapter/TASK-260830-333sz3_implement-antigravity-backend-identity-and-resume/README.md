# TASK-260830-333sz3: implement-antigravity-backend-identity-and-resume

## Description
Discover authenticated realm and conversation UUID, validate backend presence, resume, and reject blank replacement. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §8, §19.3. Work only inside the antigravity-provider-adapter story boundary.

## Acceptance Criteria
Production behavior demonstrates: Discover authenticated realm and conversation UUID, validate backend presence, resume, and reject blank replacement. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
