# TASK-260830-2xdt8t: implement-terminal-backend-registry

## Description
Implement built-in/external backend registration, canonical IDs, version tuples, trust checks, and duplicate or drift refusal. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §4.A-4.1, §6.5, §7.A. Work only inside the terminal-backend-registry-and-contract story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement built-in/external backend registration, canonical IDs, version tuples, trust checks, and duplicate or drift refusal. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
