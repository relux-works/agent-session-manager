# TASK-260830-1zzmgd: test-credential-realm-regressions

## Description
Exercise locked keychain, changed ACL/build/OS, missing SSH_AUTH_SOCK, GUI logout, prompt requirement, and restored tmux server. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §4.D-4.E, §16.2. Work only inside the terminal-credential-readiness-and-doctor story boundary.

## Acceptance Criteria
Production behavior demonstrates: Exercise locked keychain, changed ACL/build/OS, missing SSH_AUTH_SOCK, GUI logout, prompt requirement, and restored tmux server. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
