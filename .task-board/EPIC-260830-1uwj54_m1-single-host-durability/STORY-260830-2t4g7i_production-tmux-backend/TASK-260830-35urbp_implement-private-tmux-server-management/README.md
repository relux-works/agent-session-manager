# TASK-260830-35urbp: implement-private-tmux-server-management

## Description
Create owner-only runtime directories and dedicated tmux -S servers without ambient/default discovery or reuse. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §4.2, §4.C-4.E. Work only inside the production-tmux-backend story boundary.

## Acceptance Criteria
Production behavior demonstrates: Create owner-only runtime directories and dedicated tmux -S servers without ambient/default discovery or reuse. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
