# TASK-260830-i5uni3: run-local-and-remote-session-walkthroughs

## Description
Observe operators using discovery, attach, resume, planning, move, recovery, and TUI workflows on representative machines. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); Advisory evidence for §14 and §19. Work only inside the operator-ux-field-evaluation story boundary.

## Acceptance Criteria
Production behavior demonstrates: Observe operators using discovery, attach, resume, planning, move, recovery, and TUI workflows on representative machines. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
