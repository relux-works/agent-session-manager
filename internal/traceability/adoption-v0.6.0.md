# v0.6.0 adoption ownership

Authority: `agent-session-manager-spec@v0.6.0`, commit
`0cbdf100dbf84df50c64f792b1f940e3a67859a6`, as verified by
`internal/specpin/v0.6.0.lock.json`. The 13 new headings are the difference
between the pinned v0.6.0 and v0.5.0 section inventories. Catalogue census is
not implemented runtime support. Every new heading is refused by
`traceability.VerifyAssignedSections`; task IDs in registry gaps identify
pending delivery owners and do not discharge clauses.

| Sections | Primary pending owner | Responsibility and boundary |
| --- | --- | --- |
| 6.6 | TASK-260909-2ez769 | Config-4 closed reader, explicit exact-preview migration, complete enrollment, atomic durable activation/rollback; preserve Config 1/2/3 |
| 11.10, 11.10.1 | TASK-260830-z1yxg9 | Host Channel launch, mutual TLS, exact ALPN, authenticated UUID before hello/dispatch, deadlines and current-generation admission; consume credential APIs |
| 11.10.2, 11.10.3 | TASK-260909-2ez769 | Machine-local trust/credential custody, issuance, out-of-band enrollment, rotation, revocation, generation API, exclusion from replication |
| 11.10.4 | TASK-260830-2x16gz | Executed host-channel conformance including certificate/handshake/race/platform/transport parity; source vectors alone are insufficient |
| 11.10.5 | TASK-260830-2x16gz | Consume upstream synthetic admission evidence and prove corresponding runtime behavior; the public source validator belongs to the spec repository |
| 14.7, 14.7.1, 14.7.2 | TASK-260830-21gygk | One literal selector resolver and locally attested SelectionPlan/revalidation API; source reads, failure distinctions, authority and actual-invocation binding |
| 14.7.3 | TASK-260830-1i7bqr | CLI Result 5/Error 1.4 closed reader/emitter mapping, authoritative summary integration, both remote attach/log endpoints and continuation/retry |
| 14.7.4 | TASK-260830-1wb06o | Interrupted record-only creation/recovery from original durable inputs; no invented initial lease/observation facts |
| 14.7.5 | TASK-260830-1i7bqr | Composed product selector conformance across callers; the publication evaluator and source-only case inventory belong to the spec repository |

The shared Section 14.7.2 API is implemented once by the selector owner.
TASK-260830-1i7bqr owns invoking it on CLI/remote route boundaries;
TASK-260830-1wb06o owns lifecycle pre-effect/fencing/commit/recovery boundaries.
This split is caller integration, not duplicate resolution implementations.
Section 14.7.3 summaries also consume TASK-260830-21gygk's authoritative
Section 5.7 projection. Every action and boundary in the source table remains
an acceptance obligation; later command owners must use these shared APIs.

Refined historical clauses remain in scope: Section 2.3 selection belongs to
TASK-260830-21gygk; Sections 11.1–11.3 bilateral admission and hello belong to
TASK-260830-z1yxg9. TASK-260830-219okr retains subsequent major negotiation
under Sections 11.2–11.3/17, now including RPC 5 with no Config-4 downgrade.
Existing TASK-260830-2u34k1 identity and TASK-260830-1tvg8e SSH acceptance
remain historical inputs; their preserved partial implementations are untouched.

Board scopes append the adopted authority and these exact responsibilities,
preserving the complete previous scope and AC. The six existing dependent
owners and the new credentials/migration task are linked to
STORY-260908-18woqo; RPC admission also depends on the credential API task.
No owner is unblocked by this adoption candidate. Resume follows reviewed,
signed Story delivery through the orchestrator.

`catalog.Current` is the 63-contract v0.6.0 census, including RPC 5 vocabulary.
Error 1.4 is census-only: no new error-code runtime or CLI5 reader is advertised.
Config-4 reader claims are refused by AssessCompatibility; historical readers
can report Config-4 source documents read-only without decoding them. Decode
and Migrate continue to refuse Config-4 runtime documents/targets. Historical
V050/V043 catalogue projections, specpin/specdoc historical readers, the old
ownership registry, and localstore's explicit CurrentV050 binding remain valid.
The unchanged 17 discharged clauses remain 17; the new denominator is 463.
