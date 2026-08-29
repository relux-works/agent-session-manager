# TASK-260830-1jx0ut: implement-muse-quiescence-and-portability-refusals

## Description
Fence benign cron/tool work, prove clean exit, preserve encrypted reasoning, and advertise portable_store=false with P disabled. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §8, §19.3. Work only inside the muse-provider-adapter story boundary.

## Acceptance Criteria
Production behavior demonstrates: Fence benign cron/tool work, prove clean exit, preserve encrypted reasoning, and advertise portable_store=false with P disabled. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
