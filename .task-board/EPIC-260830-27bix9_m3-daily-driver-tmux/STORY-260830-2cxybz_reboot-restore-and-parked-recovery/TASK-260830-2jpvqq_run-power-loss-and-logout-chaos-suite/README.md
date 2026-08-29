# TASK-260830-2jpvqq: run-power-loss-and-logout-chaos-suite

## Description
Cut power or kill processes at every restore phase across macOS/Linux and assert identities, epochs, and outcomes. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §13.11-13.13. Work only inside the reboot-restore-and-parked-recovery story boundary.

## Acceptance Criteria
Production behavior demonstrates: Cut power or kill processes at every restore phase across macOS/Linux and assert identities, epochs, and outcomes. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
