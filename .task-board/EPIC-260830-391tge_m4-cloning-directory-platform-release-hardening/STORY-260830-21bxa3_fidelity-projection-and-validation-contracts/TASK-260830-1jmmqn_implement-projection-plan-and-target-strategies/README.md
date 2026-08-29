# TASK-260830-1jmmqn: implement-projection-plan-and-target-strategies

## Description
Plan pair-neutral target projection, checkpoint message authority, inactive tools/instructions, token metadata, and vendor importer strategies. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §13.14.2, §13.14.4. Work only inside the fidelity-projection-and-validation-contracts story boundary.

## Acceptance Criteria
Production behavior demonstrates: Plan pair-neutral target projection, checkpoint message authority, inactive tools/instructions, token metadata, and vendor importer strategies. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
