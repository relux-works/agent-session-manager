# TASK-260830-1aea3m: run-macos-amd64-linux-arm64-windows-arm64-lanes

## Description
Execute core/provider subsets only where artifacts exist and capture explicit conditional or unsupported evidence. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §8, §19.2. Work only inside the wsl2-and-secondary-platform-lanes story boundary.

## Acceptance Criteria
Production behavior demonstrates: Execute core/provider subsets only where artifacts exist and capture explicit conditional or unsupported evidence. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
