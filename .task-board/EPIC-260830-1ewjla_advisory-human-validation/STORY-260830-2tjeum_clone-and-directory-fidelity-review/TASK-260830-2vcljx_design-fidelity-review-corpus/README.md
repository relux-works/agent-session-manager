# TASK-260830-2vcljx: design-fidelity-review-corpus

## Description
Prepare synthetic and consented sanitized sessions covering exact, lossy, opaque, omitted, archive, and continuation-context outcomes. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); Advisory evidence for §13.14-13.15 and §14.5. Work only inside the clone-and-directory-fidelity-review story boundary.

## Acceptance Criteria
Production behavior demonstrates: Prepare synthetic and consented sanitized sessions covering exact, lossy, opaque, omitted, archive, and continuation-context outcomes. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
