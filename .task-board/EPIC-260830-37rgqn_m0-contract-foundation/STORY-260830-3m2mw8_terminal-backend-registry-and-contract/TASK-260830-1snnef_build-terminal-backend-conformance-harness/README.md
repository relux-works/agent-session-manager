# TASK-260830-1snnef: build-terminal-backend-conformance-harness

## Description
Prove lifecycle semantics, attach ownership neutrality, ax pane enforcement, replication exclusions, and historical translation. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §4.A-4.1, §6.5, §7.A. Work only inside the terminal-backend-registry-and-contract story boundary.

## Acceptance Criteria
Production behavior demonstrates: Prove lifecycle semantics, attach ownership neutrality, ax pane enforcement, replication exclusions, and historical translation. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
