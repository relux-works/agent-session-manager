# TASK-260830-2wflxd: implement-environment-and-native-session-observers

## Description
Probe installations/realms and emit exact/weak native identities, heads, workspace facts, runtime facts, capabilities, and sanitized metadata. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §7.9, §10.8.1, §13.15. Work only inside the directory-environment-observation-and-inventory story boundary.

## Acceptance Criteria
Production behavior demonstrates: Probe installations/realms and emit exact/weak native identities, heads, workspace facts, runtime facts, capabilities, and sanitized metadata. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
