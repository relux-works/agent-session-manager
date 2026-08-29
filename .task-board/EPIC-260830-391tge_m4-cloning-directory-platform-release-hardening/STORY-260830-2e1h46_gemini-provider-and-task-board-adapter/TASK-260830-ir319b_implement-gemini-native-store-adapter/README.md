# TASK-260830-ir319b: implement-gemini-native-store-adapter

## Description
Implement exact tuple discovery, identity, capture/materialize, quiescence, resume, and refusal for unsupported stores. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §8, §9, §19.3. Work only inside the gemini-provider-and-task-board-adapter story boundary.

## Acceptance Criteria
Production behavior demonstrates: Implement exact tuple discovery, identity, capture/materialize, quiescence, resume, and refusal for unsupported stores. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
