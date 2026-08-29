# TASK-260830-1ivbfw: run-win11-amd64-core-provider-matrix

## Description
Run core, Codex/Claude conditional, dirty workspace, reboot, transfer, security, and unsupported durability tests on exact Windows tuples. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §4.3, §19.2. Work only inside the native-windows-conpty-backend story boundary.

## Acceptance Criteria
Production behavior demonstrates: Run core, Codex/Claude conditional, dirty workspace, reboot, transfer, security, and unsupported durability tests on exact Windows tuples. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
