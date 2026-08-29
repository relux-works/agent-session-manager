# TASK-260830-2gapi7: test-attach-neutrality-and-disconnects

## Description
Prove multiple clients, disconnect/reconnect, stale owner, unsupported backend, and hostile terminal strings never change AX state. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §4.E, §13.5, §16.7. Work only inside the remote-attach-and-terminal-forwarding story boundary.

## Acceptance Criteria
Production behavior demonstrates: Prove multiple clients, disconnect/reconnect, stale owner, unsupported backend, and hostile terminal strings never change AX state. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
