# TASK-260830-kluuit: implement-route-and-outcome-decision-engine

## Description
Resolve attach, resume, takeover, fork, adoption, same/cross-environment clone/move, unmanaged local open, and archive/context fallback. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §10.8.4, §13.15. Work only inside the pure-continuation-planner story boundary.

## Acceptance Criteria
Production behavior demonstrates: Resolve attach, resume, takeover, fork, adoption, same/cross-environment clone/move, unmanaged local open, and archive/context fallback. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
