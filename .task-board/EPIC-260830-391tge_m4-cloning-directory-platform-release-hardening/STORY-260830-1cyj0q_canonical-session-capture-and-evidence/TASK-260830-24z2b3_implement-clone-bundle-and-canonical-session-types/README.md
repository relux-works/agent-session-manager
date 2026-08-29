# TASK-260830-24z2b3: implement-clone-bundle-and-canonical-session-types

## Description
Implement Clone Bundle Manifest, Canonical Session/Event, raw evidence descriptors, extensions, identities, and immutable generations. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §7.8, §13.14.1, §10.2. Work only inside the canonical-session-capture-and-evidence story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement Clone Bundle Manifest, Canonical Session/Event, raw evidence descriptors, extensions, identities, and immutable generations. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
