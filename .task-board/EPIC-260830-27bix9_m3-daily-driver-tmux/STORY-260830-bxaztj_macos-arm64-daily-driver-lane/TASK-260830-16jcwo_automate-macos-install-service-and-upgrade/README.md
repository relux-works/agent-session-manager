# TASK-260830-16jcwo: automate-macos-install-service-and-upgrade

## Description
Package ax, launchd control plane, Aqua broker, private runtime permissions, upgrade, uninstall, and rollback. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §4.2, §4.4, §19.2. Work only inside the macos-arm64-daily-driver-lane story boundary.

## Acceptance Criteria
Production behavior demonstrates: Package ax, launchd control plane, Aqua broker, private runtime permissions, upgrade, uninstall, and rollback. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
