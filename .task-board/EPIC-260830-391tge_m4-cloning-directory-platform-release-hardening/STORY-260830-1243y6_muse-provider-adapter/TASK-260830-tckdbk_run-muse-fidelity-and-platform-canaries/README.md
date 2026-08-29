# TASK-260830-tckdbk: run-muse-fidelity-and-platform-canaries

## Description
Test subagents/tool output/reasoning, shards/path changes, forks without duplicate schedules, incompatible versions, and TP lanes. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §8, §19.3. Work only inside the muse-provider-adapter story boundary.

## Acceptance Criteria
Production behavior demonstrates: Test subagents/tool output/reasoning, shards/path changes, forks without duplicate schedules, incompatible versions, and TP lanes. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
