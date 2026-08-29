# TASK-260830-3esaam: implement-record-envelope-and-session-records

## Description
Implement Record Envelope plus immutable Session Record versions and creation/derivation provenance unions. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §2, §5, §10.1, §18.1. Work only inside the authoritative-record-schema-core story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement Record Envelope plus immutable Session Record versions and creation/derivation provenance unions. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
