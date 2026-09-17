# TASK-260830-2x16gz: test-peer-auth-and-hostile-network-failures

## Description
Prove unknown peers, key changes, spoofed host IDs, disconnects, replay, oversized frames, and disclosure mismatches fail closed. This is an implementation deliverable, not a specification rewrite.

## Scope
Adopted current authority: relux-works/agent-session-manager-spec@v0.6.0, commit 0cbdf100dbf84df50c64f792b1f940e3a67859a6. Primary implementation-conformance owner for section 11.10.4 / AC-HOST-001, across both OpenSSH and native Tailscale SSH, with real certificate chains, handshake, timeout/race, custody, revocation and generation tests. Use section 11.10.5 and its upstream publication vectors as source evidence only; do not claim their synthetic passes establish executed product support. Historical scope retained without weakening: relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); \u00a76, \u00a711.1, \u00a716.1. Work only inside the peer-configuration-and-authentication story boundary. Activation waits for signed STORY-260908-18woqo landing; this binding does not accept or alter the preserved partial implementation.

## Acceptance Criteria
Production behavior demonstrates: Prove unknown peers, key changes, spoofed host IDs, disconnects, replay, oversized frames, and disclosure mismatches fail closed. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
