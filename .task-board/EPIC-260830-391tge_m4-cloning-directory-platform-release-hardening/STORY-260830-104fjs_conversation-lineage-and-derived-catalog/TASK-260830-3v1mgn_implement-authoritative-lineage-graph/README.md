# TASK-260830-3v1mgn: implement-authoritative-lineage-graph

## Description
Build links from fork/clone/move/adopt/binding/operator evidence, detect cycles/ambiguity, and separate suggestions from authority. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §10.8.2, §10.8.5. Work only inside the conversation-lineage-and-derived-catalog story boundary.

## Acceptance Criteria
Production behavior demonstrates: Build links from fork/clone/move/adopt/binding/operator evidence, detect cycles/ambiguity, and separate suggestions from authority. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
