# TASK-260830-20z11h: test-upgrade-downgrade-and-mixed-version-fleet

## Description
Test every current/historical contract reader/writer, dual-stack peer, config migration, unknown backend/provider, and no-redigest rule. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §15-18. Work only inside the full-security-observability-and-compatibility-hardening story boundary.

## Acceptance Criteria
Production behavior demonstrates: Test every current/historical contract reader/writer, dual-stack peer, config migration, unknown backend/provider, and no-redigest rule. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
