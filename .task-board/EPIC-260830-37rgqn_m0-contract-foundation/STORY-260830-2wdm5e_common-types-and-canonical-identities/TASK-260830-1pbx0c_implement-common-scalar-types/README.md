# TASK-260830-1pbx0c: implement-common-scalar-types

## Description
Implement UUID, RFC3339, digest, platform, provider, path, bounded integer, and closed-enum value types. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §1.6, §10.1-10.4, §17.3. Work only inside the common-types-and-canonical-identities story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement UUID, RFC3339, digest, platform, provider, path, bounded integer, and closed-enum value types. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
