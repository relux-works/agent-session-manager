# TASK-260830-2lr4yz: implement-keychain-sentinel-and-provider-auth-smoke

## Description
Run no-secret AX sentinel and provider-auth probes inside the exact backend instance and bind evidence to all required tuples. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §4.2, §4.4, §16.2. Work only inside the macos-aqua-terminal-broker story boundary.

## Acceptance Criteria
Production behavior demonstrates: Run no-secret AX sentinel and provider-auth probes inside the exact backend instance and bind evidence to all required tuples. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
