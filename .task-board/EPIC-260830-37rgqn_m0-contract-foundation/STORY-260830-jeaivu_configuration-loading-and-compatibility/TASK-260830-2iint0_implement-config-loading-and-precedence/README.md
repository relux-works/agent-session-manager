# TASK-260830-2iint0: implement-config-loading-and-precedence

## Description
Implement platform config discovery, explicit overrides, environment handling, defaults, and precedence without secret passthrough. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §6, §17. Work only inside the configuration-loading-and-compatibility story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement platform config discovery, explicit overrides, environment handling, defaults, and precedence without secret passthrough. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
