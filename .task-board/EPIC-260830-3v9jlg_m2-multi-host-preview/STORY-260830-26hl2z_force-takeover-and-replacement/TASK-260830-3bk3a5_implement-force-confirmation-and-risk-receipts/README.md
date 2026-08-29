# TASK-260830-3bk3a5: implement-force-confirmation-and-risk-receipts

## Description
Require exact session/epoch/host confirmation, emit force and replacement events, and preserve old-owner external-effect warnings. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §13.7, §15, §16.5. Work only inside the force-takeover-and-replacement story boundary.

## Acceptance Criteria
Production behavior demonstrates: Require exact session/epoch/host confirmation, emit force and replacement events, and preserve old-owner external-effect warnings. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
