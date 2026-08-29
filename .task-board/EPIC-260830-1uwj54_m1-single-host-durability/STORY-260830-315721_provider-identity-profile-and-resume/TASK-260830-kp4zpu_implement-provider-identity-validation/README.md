# TASK-260830-kp4zpu: implement-provider-identity-validation

## Description
Create and validate exact native session IDs, store roots, backend realms, build tuples, and discovery evidence. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §2.4, §5.5, §7.5-7.7, §8. Work only inside the provider-identity-profile-and-resume story boundary.

## Acceptance Criteria
Production behavior demonstrates: Create and validate exact native session IDs, store roots, backend realms, build tuples, and discovery evidence. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
