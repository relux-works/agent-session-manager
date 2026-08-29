# TASK-260830-29l8x1: run-repeated-move-sync-and-attach-canaries

## Description
Run at least 100 fault-injected transfers plus repeated operator moves/attaches with identity and owner assertions. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §19.1-19.4. Work only inside the multi-host-preview-acceptance-gate story boundary.

## Acceptance Criteria
Production behavior demonstrates: Run at least 100 fault-injected transfers plus repeated operator moves/attaches with identity and owner assertions. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
