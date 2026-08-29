# TASK-260830-34elja: implement-structured-error-registry

## Description
Implement Structured Error versions, typed details, stable codes, retryability, and causal redaction. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §14.2, §15, §17.2. Work only inside the structured-errors-cli-envelope-and-exit-codes story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement Structured Error versions, typed details, stable codes, retryability, and causal redaction. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
