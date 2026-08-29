# TASK-260830-2u34k1: implement-host-and-peer-identity

## Description
Implement host IDs, peer aliases, SSH targets, key provenance, allowlists, disclosure policy, and duplicate identity refusal. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §6, §11.1, §16.1. Work only inside the peer-configuration-and-authentication story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement host IDs, peer aliases, SSH targets, key provenance, allowlists, disclosure policy, and duplicate identity refusal. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
