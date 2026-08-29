# TASK-260830-1wb06o: implement-local-lifecycle-controller

## Description
Compose direct/task-board launch, local attach, stop, resume, fork, checkpoint, and reboot restore from existing domain operations. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §13.1-13.4, §13.8-13.11, §14. Work only inside the single-host-lifecycle-and-cli-slice story boundary.

## Acceptance Criteria
Production behavior demonstrates: Compose direct/task-board launch, local attach, stop, resume, fork, checkpoint, and reboot restore from existing domain operations. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
