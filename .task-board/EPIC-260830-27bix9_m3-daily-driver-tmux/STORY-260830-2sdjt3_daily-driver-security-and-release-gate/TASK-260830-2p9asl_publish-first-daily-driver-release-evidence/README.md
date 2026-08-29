# TASK-260830-2p9asl: publish-first-daily-driver-release-evidence

## Description
Publish signed capability matrix and limitations only when all blocking macOS/Linux tmux gates pass. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §16, §18-19. Work only inside the daily-driver-security-and-release-gate story boundary.

## Acceptance Criteria
Production behavior demonstrates: Publish signed capability matrix and limitations only when all blocking macOS/Linux tmux gates pass. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
