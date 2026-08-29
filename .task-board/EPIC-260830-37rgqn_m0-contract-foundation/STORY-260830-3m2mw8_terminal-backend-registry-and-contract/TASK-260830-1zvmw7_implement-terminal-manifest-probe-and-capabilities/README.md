# TASK-260830-1zvmw7: implement-terminal-manifest-probe-and-capabilities

## Description
Implement closed manifest/probe schemas, generation-bound capability evidence, dependencies, expiry, and fail-closed admission. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §4.A-4.1, §6.5, §7.A. Work only inside the terminal-backend-registry-and-contract story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement closed manifest/probe schemas, generation-bound capability evidence, dependencies, expiry, and fail-closed admission. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
