# TASK-260830-3kle7h: implement-controller-locking-and-local-ipc

## Description
Implement per-session controller exclusion, local authenticated IPC, cancellation, and status observation. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §3.1, §4.4, §13.11. Work only inside the daemonless-controller-and-user-services story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement per-session controller exclusion, local authenticated IPC, cancellation, and status observation. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
