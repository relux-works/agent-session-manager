# TASK-260830-2890sd: implement-provider-discovery-and-trust

## Description
Implement deterministic executable discovery, build identity, trust policy, substitution detection, and duplicate refusal. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §7.1-7.7, §8. Work only inside the provider-plugin-host-and-protocol story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement deterministic executable discovery, build identity, trust policy, substitution detection, and duplicate refusal. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
