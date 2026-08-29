# TASK-260830-g0pcnt: test-tmux-generation-and-attach-semantics

## Description
Prove lost-response idempotency, reconnect/multi-attach policy, socket substitution refusal, and ownership-neutral attach. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §4.2, §4.C-4.E. Work only inside the production-tmux-backend story boundary.

## Acceptance Criteria
Production behavior demonstrates: Prove lost-response idempotency, reconnect/multi-attach policy, socket substitution refusal, and ownership-neutral attach. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
