# TASK-260830-14kb4p: implement-observation-metrics-health-and-retention

## Description
Emit secret-free events/metrics/health, retain immutable audits, rotate debug logs, and support incident reconstruction. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §15-18. Work only inside the full-security-observability-and-compatibility-hardening story boundary.

## Acceptance Criteria
Production behavior demonstrates: Emit secret-free events/metrics/health, retain immutable audits, rotate debug logs, and support incident reconstruction. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
