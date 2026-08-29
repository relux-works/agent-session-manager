# STORY-260830-1cyj0q: canonical-session-capture-and-evidence

## Description
Implement capture native source bytes and normalize every item into a canonical session graph according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §7.8, §13.14.1, §10.2. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Capture native source bytes and normalize every item into a canonical session graph is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
