# TASK-260830-2g5be6: implement-native-capture-and-source-race-check

## Description
Capture stable provider store bytes, item identities, workspace checkpoint, source head, and detect mutation before projection. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §7.8, §13.14.1, §10.2. Work only inside the canonical-session-capture-and-evidence story boundary.

## Acceptance Criteria
Production behavior demonstrates: Capture stable provider store bytes, item identities, workspace checkpoint, source head, and detect mutation before projection. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
