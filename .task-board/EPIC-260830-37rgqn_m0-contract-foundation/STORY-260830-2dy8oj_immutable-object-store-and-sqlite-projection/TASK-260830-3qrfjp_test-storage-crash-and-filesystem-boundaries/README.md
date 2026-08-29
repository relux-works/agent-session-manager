# TASK-260830-3qrfjp: test-storage-crash-and-filesystem-boundaries

## Description
Fault-inject object writes, SQLite rebuilds, permissions, symlinks, special files, short writes, and full-disk boundaries. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §3, §10.1-10.2, §18.4. Work only inside the immutable-object-store-and-sqlite-projection story boundary.

## Acceptance Criteria
Production behavior demonstrates: Fault-inject object writes, SQLite rebuilds, permissions, symlinks, special files, short writes, and full-disk boundaries. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
