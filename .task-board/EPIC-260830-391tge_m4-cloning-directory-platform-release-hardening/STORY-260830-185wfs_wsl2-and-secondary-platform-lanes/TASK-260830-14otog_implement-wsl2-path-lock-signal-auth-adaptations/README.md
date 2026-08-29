# TASK-260830-14otog: implement-wsl2-path-lock-signal-auth-adaptations

## Description
Handle Linux-home versus Windows-mounted filesystems, systemd, tmux restore, paths, locks, signals, and auth boundaries. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §8, §19.2. Work only inside the wsl2-and-secondary-platform-lanes story boundary.

## Acceptance Criteria
Production behavior demonstrates: Handle Linux-home versus Windows-mounted filesystems, systemd, tmux restore, paths, locks, signals, and auth boundaries. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
