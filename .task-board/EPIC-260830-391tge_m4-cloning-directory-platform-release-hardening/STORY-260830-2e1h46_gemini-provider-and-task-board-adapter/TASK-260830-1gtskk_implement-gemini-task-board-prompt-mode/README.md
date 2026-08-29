# TASK-260830-1gtskk: implement-gemini-task-board-prompt-mode

## Description
Integrate tracked prompt bundles, null-goal state, profiles, export/import/open/adopt, and recovery. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §8, §9, §19.3. Work only inside the gemini-provider-and-task-board-adapter story boundary.

## Acceptance Criteria
Production behavior demonstrates: Integrate tracked prompt bundles, null-goal state, profiles, export/import/open/adopt, and recovery. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
