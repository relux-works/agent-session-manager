# TASK-260830-19kf8t: implement-tombstones-and-acknowledgements

## Description
Implement scoped tombstone issuance/resolution, peer acknowledgements, authorization events, retention, and union behavior. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §10.7, §11.7, §17.3. Work only inside the conflict-tombstone-and-gc-union story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement scoped tombstone issuance/resolution, peer acknowledgements, authorization events, retention, and union behavior. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
