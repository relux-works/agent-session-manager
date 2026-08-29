# TASK-260830-1ccn30: implement-pi-provider-adapter

## Description
Implement Pi discovery, identity, store capture/materialization, quiescence, resume, and exact platform tuple gating. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §8, §19.3. Work only inside the pi-and-qwen-environment-support story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement Pi discovery, identity, store capture/materialization, quiescence, resume, and exact platform tuple gating. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
