# TASK-260830-kkh1an: implement-terminal-instance-state-machine

## Description
Implement absent/creating/parked/active/quiescing/stopped/stale/unavailable transitions and generation-safe evidence. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §4.B-4.D, §5.2 Terminal Events, §7.A, §13.1. Work only inside the ax-pane-and-terminal-instance-binding story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement absent/creating/parked/active/quiescing/stopped/stale/unavailable transitions and generation-safe evidence. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
