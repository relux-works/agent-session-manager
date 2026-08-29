# TASK-260830-1kj7ae: implement-clone-prepare-stage-and-publish

## Description
Prepare target with caller operation ID, staged read-back, source race check, atomic publish, rollback retention, and live discovery. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §13.14.3-13.14.5. Work only inside the cross-environment-clone-transaction story boundary.

## Acceptance Criteria
Production behavior demonstrates: Prepare target with caller operation ID, staged read-back, source race check, atomic publish, rollback retention, and live discovery. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
