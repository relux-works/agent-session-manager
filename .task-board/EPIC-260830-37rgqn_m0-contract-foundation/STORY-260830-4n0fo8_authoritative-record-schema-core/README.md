# STORY-260830-4n0fo8: authoritative-record-schema-core

## Description
Implement provide immutable authoritative record types and explicit versioned unions according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §2, §5, §10.1, §18.1. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Provide immutable authoritative record types and explicit versioned unions is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
