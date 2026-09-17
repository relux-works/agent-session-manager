# STORY-260830-1kiyj6: peer-configuration-and-authentication

## Description
Implement peer configuration, SSH transport and the existing RPC envelope/hello prerequisite before complete hostile-peer conformance, according to the pinned AX specification. TASK-260830-z1yxg9 retains its full implementation ownership here so bilateral runtime host admission precedes TASK-260830-2x16gz.

## Scope
Normative scope: sections 6, 11.1, 16.1, plus the existing TASK-260830-z1yxg9 envelope/hello deliverable under sections 11.2-11.3 and 17. Preserve stronger global AX invariants and historical contract versions. Subsequent major negotiation and closed-operation conformance remain in STORY-260830-4qojoz.

## Acceptance Criteria
Establish explicit allowlisted host identity and authenticated ssh/private-mesh transport is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
