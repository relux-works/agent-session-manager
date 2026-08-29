# TASK-260830-3hc11u: run-gemini-platform-acceptance-suites

## Description
Run D/P/M/TP matrices on supported macOS/Linux/WSL2/Windows tuples and publish only passing cells. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §8, §9, §19.3. Work only inside the gemini-provider-and-task-board-adapter story boundary.

## Acceptance Criteria
Production behavior demonstrates: Run D/P/M/TP matrices on supported macOS/Linux/WSL2/Windows tuples and publish only passing cells. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
