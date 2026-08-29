# TASK-260830-7x33gi: implement-safe-shorthand-route-resolution

## Description
Allow ax NAME auto-execution only for one unique safe attach/resume route and return interactive_choice_required otherwise. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §2.3, §13, §14. Work only inside the operator-command-routing-and-confirmation story boundary.

## Acceptance Criteria
Production behavior demonstrates: Allow ax NAME auto-execution only for one unique safe attach/resume route and return interactive_choice_required otherwise. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
