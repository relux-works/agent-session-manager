# TASK-260830-2it6xy: pin-v0-5-0-source-and-digests

## Description
Pin the exact tag, commit, document digest, contract versions, and fixture identities consumed by the implementation. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §1, §17, §20, Appendix A, Appendix D. Work only inside the pin-normative-source-and-contract-catalog story boundary.

## Acceptance Criteria
Production behavior demonstrates: Pin the exact tag, commit, document digest, contract versions, and fixture identities consumed by the implementation. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
