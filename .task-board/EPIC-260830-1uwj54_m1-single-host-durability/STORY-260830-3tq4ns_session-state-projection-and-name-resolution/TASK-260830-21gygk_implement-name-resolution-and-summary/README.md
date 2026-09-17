# TASK-260830-21gygk: implement-name-resolution-and-summary

## Description
Resolve UUID/name/qualified selectors, ambiguity, list/status summaries, and stable deterministic sorting. This is an implementation deliverable, not a specification rewrite.

## Scope
Adopted current authority: relux-works/agent-session-manager-spec@v0.6.0, commit 0cbdf100dbf84df50c64f792b1f940e3a67859a6. Primary owner of sections 14.7 and 14.7.1 and shared SelectionPlan construction/revalidation in 14.7.2, refining section 2.3 and authoritative summaries in 5.7/14.7.3. Implement first-at literal grammar, exact source mappings, complete source reads with distinct failure classes, no explicit-source fallback, tier/collision/tombstone semantics, immutable locally attested plan and current-fact validation. Expose one shared API; CLI/lifecycle callers own invoking it at every required boundary, never a duplicate resolver. Historical scope retained without weakening: relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); \u00a72.3, \u00a75.1-5.2, \u00a75.7, \u00a714.4. Work only inside the session-state-projection-and-name-resolution story boundary. Activation waits for signed STORY-260908-18woqo landing; this binding does not accept or alter the preserved partial implementation.

## Acceptance Criteria
Production behavior demonstrates: Resolve UUID/name/qualified selectors, ambiguity, list/status summaries, and stable deterministic sorting. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
