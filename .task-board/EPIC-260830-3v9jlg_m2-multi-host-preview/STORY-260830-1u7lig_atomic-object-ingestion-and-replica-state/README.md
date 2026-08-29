# STORY-260830-1u7lig: atomic-object-ingestion-and-replica-state

## Description
Implement commit verified received objects atomically and derive non-owning replicas according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §10.5-10.7, §11.5-11.6. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Commit verified received objects atomically and derive non-owning replicas is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
