# TASK-260830-1rocmd: publish-platform-capability-matrix

## Description
Generate exact platform/backend/provider cells from signed evidence and keep missing runners or artifacts disabled. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §8, §19.2. Work only inside the wsl2-and-secondary-platform-lanes story boundary.

## Acceptance Criteria
Production behavior demonstrates: Generate exact platform/backend/provider cells from signed evidence and keep missing runners or artifacts disabled. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
