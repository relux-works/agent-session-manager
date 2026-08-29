# TASK-260830-rdzru1: run-bidirectional-fidelity-and-resume-matrix

## Description
Test empty/one-turn/long/compacted/branched/active/stopped/corrupt fixtures, dirty workspaces, and native resume smoke both ways. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §8, §13.14, §19.3. Work only inside the codex-claude-bidirectional-clone-adapters story boundary.

## Acceptance Criteria
Production behavior demonstrates: Test empty/one-turn/long/compacted/branched/active/stopped/corrupt fixtures, dirty workspaces, and native resume smoke both ways. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
