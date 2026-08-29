# TASK-260830-1bipsa: implement-cli-result-envelopes

## Description
Implement CLI Result versions, JSON/JSONL serialization, human rendering boundaries, and exact exit-code mapping. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §14.2, §15, §17.2. Work only inside the structured-errors-cli-envelope-and-exit-codes story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement CLI Result versions, JSON/JSONL serialization, human rendering boundaries, and exact exit-code mapping. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
