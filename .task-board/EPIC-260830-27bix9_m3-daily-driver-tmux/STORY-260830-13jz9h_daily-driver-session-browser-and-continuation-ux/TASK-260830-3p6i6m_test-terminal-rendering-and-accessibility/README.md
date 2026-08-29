# TASK-260830-3p6i6m: test-terminal-rendering-and-accessibility

## Description
Test narrow/wide terminals, ANSI/OSC/bidi/control/width injection, noninteractive output, and keyboard-safe attach handoff. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §14.1-14.4, §18. Work only inside the daily-driver-session-browser-and-continuation-ux story boundary.

## Acceptance Criteria
Production behavior demonstrates: Test narrow/wide terminals, ANSI/OSC/bidi/control/width injection, noninteractive output, and keyboard-safe attach handoff. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
