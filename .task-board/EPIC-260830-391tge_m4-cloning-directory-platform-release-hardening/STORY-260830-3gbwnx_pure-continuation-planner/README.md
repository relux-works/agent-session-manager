# STORY-260830-3gbwnx: pure-continuation-planner

## Description
Implement produce immutable expiring plans for every continuation intent without mutation according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §10.8.4, §13.15. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Produce immutable expiring plans for every continuation intent without mutation is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
