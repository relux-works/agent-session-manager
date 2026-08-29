# TASK-260830-2v504o: implement-enrichment-job-receipts-and-races

## Description
Derive job state from immutable receipt chains and publish only after subject/profile/policy/evidence recheck; changed heads supersede. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §10.8.2-10.8.3, §16.7. Work only inside the annotations-manual-metadata-and-enrichment story boundary.

## Acceptance Criteria
Production behavior demonstrates: Derive job state from immutable receipt chains and publish only after subject/profile/policy/evidence recheck; changed heads supersede. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
