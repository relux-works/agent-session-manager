# TASK-260830-3g2wu8: test-directory-convergence-and-privacy

## Description
Test reorder/duplicate/gap/conflict/offline convergence and reject raw IDs, paths, transcripts, credentials, runtime handles, and model payloads. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §11.8-11.9, §16.7, §17.5. Work only inside the directory-metadata-replication story boundary.

## Acceptance Criteria
Production behavior demonstrates: Test reorder/duplicate/gap/conflict/offline convergence and reject raw IDs, paths, transcripts, credentials, runtime handles, and model payloads. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
