# STORY-260830-2jylym: structured-errors-cli-envelope-and-exit-codes

## Description
Implement make all machine-visible failures and CLI results closed, typed, and versioned according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §14.2, §15, §17.2. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Make all machine-visible failures and cli results closed, typed, and versioned is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
