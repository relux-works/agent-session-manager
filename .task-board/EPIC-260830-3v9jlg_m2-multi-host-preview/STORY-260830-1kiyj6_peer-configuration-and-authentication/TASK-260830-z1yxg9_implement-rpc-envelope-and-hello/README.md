# TASK-260830-z1yxg9: implement-rpc-envelope-and-hello

## Description
Implement request/response correlation, hello maps, namespace/cardinality contracts, limits, deadlines, and structured error framing. This is an implementation deliverable, not a specification rewrite.

## Scope
Adopted current authority: relux-works/agent-session-manager-spec@v0.6.0, commit 0cbdf100dbf84df50c64f792b1f940e3a67859a6. Primary runtime owner of sections 11.10 and 11.10.1, refining 11.1/11.2/11.3/17: explicit host-channel launch, mutual TLS 1.3, ALPN ax-host/1, no resumption/early data/fallback, identity from verified enrolled certificates before hello and dispatch, both hello IDs checked, bounded streams/timeouts and current authorization generation at dispatch/mutation. Consume credential/config APIs owned by TASK-260909-2ez769 and accepted historical SSH/identity code. Section 11.10.5 is an upstream synthetic source validator, not an AX runtime JSON-policy reader. Historical scope retained without weakening: relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); sections 11.2-11.3 and 17, with the section 11.1 bilateral host-admission prerequisite. Work inside the peer-configuration-and-authentication Story boundary for the full existing RPC envelope/hello deliverable. Subsequent major negotiation remains TASK-260830-219okr; preserve the original acceptance criteria. Activation waits for signed STORY-260908-18woqo landing; this binding does not accept or alter the preserved partial implementation.

## Acceptance Criteria
Production behavior demonstrates: Implement request/response correlation, hello maps, namespace/cardinality contracts, limits, deadlines, and structured error framing. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
