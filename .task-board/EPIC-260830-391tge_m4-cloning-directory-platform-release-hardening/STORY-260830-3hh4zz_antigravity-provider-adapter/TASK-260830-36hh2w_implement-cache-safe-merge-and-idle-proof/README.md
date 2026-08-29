# TASK-260830-36hh2w: implement-cache-safe-merge-and-idle-proof

## Description
Merge caches without losing unrelated entries, close SQLite handles, and implement Stop.fullyIdle false/true evidence. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §8, §19.3. Work only inside the antigravity-provider-adapter story boundary.

## Acceptance Criteria
Production behavior demonstrates: Merge caches without losing unrelated entries, close SQLite handles, and implement Stop.fullyIdle false/true evidence. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
