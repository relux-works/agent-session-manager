# TASK-260909-2ez769: implement-host-credentials-and-config4-migration

## Description
Implement machine-local Host Trust Store 1.0.0, issuance and explicit enrollment, rotation/revocation and authorization generations, and explicit Configuration 4.0.0 migration. Preserve accepted historical Config 1/2/3 and peer identity/SSH work; consume it through reviewed integration. This is pending runtime work, not delivered by catalogue adoption.

## Scope
relux-works/agent-session-manager-spec@v0.6.0 commit 0cbdf100dbf84df50c64f792b1f940e3a67859a6; sections 6.6, 11.10.2, 11.10.3. Own credential custody and trust-generation APIs, durable preview/apply/rollback and credential exclusion; RPC transport/admission stays TASK-260830-z1yxg9, conformance TASK-260830-2x16gz.

## Acceptance Criteria
Exact closed Config-4 and Host Trust Store readers; explicit out-of-band enrollment, unique credential-to-host mapping, fresh-key bounded rotation and immediate revocation; current-generation mutation authorization; owner-only atomic crash-durable writes; exact-preview confirmed Config-4 migration with complete peer enrollment; refuse partial/unreadable/missing state, downgrade and implicit legacy fallback. Positive and narrowing-negative production tests, platform custody and crash evidence; no secret replication. Resume only after signed STORY-260908-18woqo lands.
