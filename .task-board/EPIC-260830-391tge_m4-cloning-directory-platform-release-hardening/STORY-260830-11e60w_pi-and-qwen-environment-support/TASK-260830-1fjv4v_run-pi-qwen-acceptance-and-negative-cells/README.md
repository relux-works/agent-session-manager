# TASK-260830-1fjv4v: run-pi-qwen-acceptance-and-negative-cells

## Description
Run Pi D/P/M where supported, Qwen TP lanes, and deterministic disabled/unknown future-plugin cases. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §8, §19.3. Work only inside the pi-and-qwen-environment-support story boundary.

## Acceptance Criteria
Production behavior demonstrates: Run Pi D/P/M where supported, Qwen TP lanes, and deterministic disabled/unknown future-plugin cases. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
