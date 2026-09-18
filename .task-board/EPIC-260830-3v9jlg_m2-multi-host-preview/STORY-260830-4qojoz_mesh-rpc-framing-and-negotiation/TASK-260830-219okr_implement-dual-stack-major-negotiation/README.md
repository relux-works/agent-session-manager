# TASK-260830-219okr: implement-dual-stack-major-negotiation

## Description
Negotiate supported majors, preserve core-only peers, expose unsupported directory/backend activation, and refuse no-common-major. This is an implementation deliverable, not a specification rewrite.

## Scope
Adopted current authority: relux-works/agent-session-manager-spec@v0.6.0, commit 0cbdf100dbf84df50c64f792b1f940e3a67859a6. Own subsequent major negotiation under sections 11.2-11.3/17 with adopted RPC 5 and explicit historical RPC 2/3/4 bounds. Config-4 Host Channel endpoints select RPC 5 only; legacy installations keep explicit historical behavior. Negotiation cannot downgrade a required authenticated endpoint or substitute for TASK-260830-z1yxg9 admission. Historical scope retained without weakening: relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); \u00a711.2-11.3, \u00a717. Work only inside the mesh-rpc-framing-and-negotiation story boundary. Activation waits for signed STORY-260908-18woqo landing; this binding does not accept or alter the preserved partial implementation.

## Acceptance Criteria
Production behavior demonstrates: Negotiate supported majors, preserve core-only peers, expose unsupported directory/backend activation, and refuse no-common-major. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
