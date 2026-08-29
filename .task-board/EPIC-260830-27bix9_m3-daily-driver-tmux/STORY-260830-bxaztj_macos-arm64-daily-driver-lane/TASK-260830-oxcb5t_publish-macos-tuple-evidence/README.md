# TASK-260830-oxcb5t: publish-macos-tuple-evidence

## Description
Publish exact OS, architecture, tmux, provider, backend generation, fixture, and resume-smoke evidence. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §4.2, §4.4, §19.2. Work only inside the macos-arm64-daily-driver-lane story boundary.

## Acceptance Criteria
Production behavior demonstrates: Publish exact OS, architecture, tmux, provider, backend generation, fixture, and resume-smoke evidence. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
