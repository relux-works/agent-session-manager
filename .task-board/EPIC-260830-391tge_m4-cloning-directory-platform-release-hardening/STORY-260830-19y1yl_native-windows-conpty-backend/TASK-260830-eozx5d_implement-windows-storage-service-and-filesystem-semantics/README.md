# TASK-260830-eozx5d: implement-windows-storage-service-and-filesystem-semantics

## Description
Implement paths, ACLs, named pipes, atomicity, locking, Scheduled Task/service, NTFS, and OpenSSH integration. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §4.3, §19.2. Work only inside the native-windows-conpty-backend story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement paths, ACLs, named pipes, atomicity, locking, Scheduled Task/service, NTFS, and OpenSSH integration. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
