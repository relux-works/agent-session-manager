# STORY-260830-3bfr34: directory-metadata-replication

## Description
Implement converge permitted sanitized directory records while retaining source-host authority according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §11.8-11.9, §16.7, §17.5. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Converge permitted sanitized directory records while retaining source-host authority is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
