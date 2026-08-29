# TASK-260830-355og8: implement-chunked-transfer-and-resume

## Description
Implement chunk descriptors, range requests, verified partial staging, receiver limits, and interruption resume. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §10.2-10.4, §11.5. Work only inside the resumable-blob-and-manifest-transfer story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement chunk descriptors, range requests, verified partial staging, receiver limits, and interruption resume. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
