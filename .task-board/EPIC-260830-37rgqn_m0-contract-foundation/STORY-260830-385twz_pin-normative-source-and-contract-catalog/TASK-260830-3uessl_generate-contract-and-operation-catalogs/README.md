# TASK-260830-3uessl: generate-contract-and-operation-catalogs

## Description
Generate typed contract, operation, capability, event, and error catalogs from reviewed implementation metadata. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §1, §17, §20, Appendix A, Appendix D. Work only inside the pin-normative-source-and-contract-catalog story boundary.

## Acceptance Criteria
Production behavior demonstrates: Generate typed contract, operation, capability, event, and error catalogs from reviewed implementation metadata. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
