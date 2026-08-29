# TASK-260830-17suox: implement-versioned-config-schemas

## Description
Implement exact Configuration 1.0.0, 2.0.0, and 3.0.0 readers plus current-version writers and legacy translation. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §6, §17. Work only inside the configuration-loading-and-compatibility story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement exact Configuration 1.0.0, 2.0.0, and 3.0.0 readers plus current-version writers and legacy translation. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
