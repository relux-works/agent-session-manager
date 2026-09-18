# STORY-260830-4qojoz: mesh-rpc-framing-and-negotiation

## Description
Implement Mesh RPC historical/current major negotiation and closed-operation conformance according to the pinned AX specification, using the existing envelope/hello prerequisite TASK-260830-z1yxg9 now delivered in the peer-authentication Story.

## Scope
Normative scope: sections 11.2-11.3 and 17 for the remaining major-negotiation and closed-operation conformance tasks. Envelope/hello implementation ownership remains TASK-260830-z1yxg9 in STORY-260830-1kiyj6. Preserve stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Implement mesh rpc historical/current major negotiation and closed operation dispatch is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
