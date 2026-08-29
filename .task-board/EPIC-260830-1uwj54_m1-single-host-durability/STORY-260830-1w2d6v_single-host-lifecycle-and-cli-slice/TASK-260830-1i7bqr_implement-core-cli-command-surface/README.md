# TASK-260830-1i7bqr: implement-core-cli-command-surface

## Description
Implement start, list, status, attach, stop, resume, fork, checkpoint, doctor, and machine/human results with yolo profile handling. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §13.1-13.4, §13.8-13.11, §14. Work only inside the single-host-lifecycle-and-cli-slice story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement start, list, status, attach, stop, resume, fork, checkpoint, doctor, and machine/human results with yolo profile handling. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
