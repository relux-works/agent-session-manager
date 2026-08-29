# TASK-260830-3l8ny3: implement-official-task-board-bridge-client

## Description
Implement exact bridge commands, capability negotiation, goal-bound primary and prompt-spawn modes, and structured errors. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §9, §10.4, §13.2. Work only inside the task-board-bridge-and-bundle story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement exact bridge commands, capability negotiation, goal-bound primary and prompt-spawn modes, and structured errors. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
