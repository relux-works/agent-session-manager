# STORY-260830-4qojoz: mesh-rpc-framing-and-negotiation

## Description
Implement implement Mesh RPC historical/current major negotiation and closed operation dispatch according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §11.2-11.3, §17. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Implement mesh rpc historical/current major negotiation and closed operation dispatch is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
