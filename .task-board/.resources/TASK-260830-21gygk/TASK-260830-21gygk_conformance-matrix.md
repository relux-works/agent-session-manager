# TASK-260830-21gygk — Clause-to-owner conformance matrix (rev6 rework)

Authority: `internal/specdoc/SPEC.v0.6.0.md` (AX v0.6.0,
`0cbdf100dbf84df50c64f792b1f940e3a67859a6`), Sections 5.3, 5.4, 14.7.2.
Scope: every Session/Lease/Checkpoint relationship this task owns.
For each row: the canonical shape owner, the shared related-record
admission owned by `internal/sessquery/lease.go` (with production call
site), and the previously accepted caller/publication boundary that
this leaf MUST NOT absorb. "Entries" = the four shared production
entries `BuildPlan`, `Revalidate` (fresh + old-plan), `AuthoritativeStatus`,
`AuthoritativeList`, all funnelling through `winningLeaseFor`.

## Session Record relationships

| # | Spec clause | Relationship | Canonical shape owner | Shared admission (this leaf) | Accepted caller/publication boundary (out of scope) |
|---|-------------|--------------|----------------------|------------------------------|-----------------------------------------------------|
| S1 | 5.1 + 14.7.2 `session_record_id` | `session_id`/`subject_id` UUIDv7 + equality; `record_id` canonical self digest; immutable record identity | `canonicaljson` closed shape + `sessrepo` attestation at the durable boundary; `sessstate.DecodeRecord` projects members without re-attesting (accepted division) | Consumes `SessionID`/`RecordID`/`Kind` from the winning-source projection in `bindPlan`, `Revalidate`, `authorize`; cross-source digest disagreement is `integrity_failure` (`checkUnionCopy`, `selectIdentity`, `checkRecordAgreement`) | Record creation/publication is the `sessrepo` owner; no leaf mints records |
| S2 | 5.1 `kind` | `kind` in {`direct`, `task_board`}; `task_board` object required iff `kind = task_board` | `canonicaljson.validateSessionRecordV1` (`requireEnum` + kind-conditional `task_board` null/object check) | Consumes `Kind` from the projection; **binds every required checkpoint's persistence variant to it** (`checkCheckpointPersistence`, this rework) | Manifest publication/storage/transport for either path stays caller-owned |
| S3 | 2.3 + 14.7.1 | Name grammar, exact tiers, ASCII-collision ambiguity, UUID/qualified selection, stable sorting | `sessrepo.Resolve` (local tier) + `canonicaljson` name grammar | `Reader.Resolve` single shared path (`resolveBare`/`resolveExplicit`); no duplicate resolver | CLI rendering/exit mapping is the CLI leaf (accepted stated bound) |

## Lease Record relationships (5.3)

| # | Spec clause | Relationship | Canonical shape owner | Shared admission (this leaf) | Accepted caller/publication boundary |
|---|-------------|--------------|----------------------|------------------------------|--------------------------------------|
| L1 | 5.3 table | Closed shape + canonical self identity (`record_id`); all members required incl. nullables | `canonicaljson` (`sessrepo.AttestLeaseRecord`); `parseLeaseRecord` maps failures to `invalid_config` | Absence for a session is `observation_unavailable`, never a minted triple (`winningLeaseFor` absence gate) | Lease publication (takeover/recovery writers) is caller-owned |
| L2 | 5.3 table | `session_id`/`subject_id`/epoch/holder/lease-ID grammars; `created_by == issued_by` (admitted, not re-gated) | `canonicaljson` + `parseLeaseRecord` member checks | Grammar failures refuse `invalid_config` before any authority use | — |
| L3 | 5.3 succession | Winner = greatest (`epoch`, `lease_id`), bytewise tie-break | — (rule implemented here) | `winningLeaseFor` via `sessstate.Compare` | Union winner across sources uses the same rule (`checkUnionCopy`); envelope triples never substitute |
| L4 | 5.3 succession | Predecessor ancestry: epoch-1 root has null predecessor; epoch > 1 names a known same-session predecessor at epoch exactly +1 | — | `checkLeaseChain`: missing name/record is `observation_unavailable`; contradictory/duplicate/self/cyclic/skipped links are `integrity_failure` | — |
| L5 | 5.3 handoff base | Epoch > 1 references a validated checkpoint for its session + predecessor lease; epoch-1 `create` MAY omit before the first provider boundary | — | `checkWinnerCheckpoint` + `checkAncestorCheckpoints` over the whole winning ancestry (CR5); unresolvable/misbound is `observation_unavailable` | Checkpoint publication (quiescence-gated writers) is caller-owned |

## Checkpoint Record relationships (5.4)

| # | Spec clause | Relationship | Canonical shape owner | Shared admission (this leaf) | Accepted caller/publication boundary |
|---|-------------|--------------|----------------------|------------------------------|--------------------------------------|
| C1 | 5.4 table | Closed shape + canonical self identity (`checkpoint_id`); **exactly one** of `provider_manifest_id` / `task_board_bundle_id` | `canonicaljson` checkpoint validator (`requireNullableDigestPresence` + xor) via `sessrepo.AttestCheckpointRecord`; `parseCheckpointRecord` maps failures to `invalid_config` | Shape alone never admits: every required reference is re-bound below | Both-null/both-non-null (CP-N2/CP-N3 class) already refuse at the canonical owner before publication |
| C2 | 5.4 `lease_epoch`/`lease_id` | Bound to the referenced winning lease: successor checkpoints name the predecessor lease; epoch-1 root checkpoints name self | — | `checkWinnerCheckpoint` (winner) + `checkCheckpointBinding` (every necessary ancestor) | — |
| C3 | 5.4 `created_by_host_id` | Creator is the current (owning) lease holder | — | Same two sites as C2 (CR5 holder binding); mismatch is `observation_unavailable` | — |
| C4 | 5.4 persistence variant | Referenced Session Record selects the variant: `direct` requires provider non-null + bundle null; `task_board` requires the reverse | Canonical owner proves exactly-one-present (C1) but its no-Session-Record API **cannot** select the variant | **`checkCheckpointPersistence` (this rework)**, called for the winner and every necessary ancestor from all four entries; mismatch is `observation_unavailable` | Which store persists/publishes bytes (provider Transfer Manifest vs task-board bundle) stays caller-owned; this leaf checks the admitted variant, never the bytes |
| C5 | 5.4 safe boundary + quiescence | Closed Safe Boundary Evidence; publication requires all-idle + zero counters; `status = validated`; `workspace_manifest_id` digest | `canonicaljson.validateSafeBoundaryEvidence` + status/digest requires (publication gate, CP-N1/CP-N4 class) | NOT re-checked: admitted records already carry the owner's verdict; re-gating diagnostics would fork the owner | Checkpoint publication/storage/transport is caller-owned; no demand is made here to implement manifest storage |
| C6 | 5.4 `event_heads` + 2.4 profile | Sorted unique heads 1..64; transitive closure fixes the effective execution profile; no later local-only fallback | `canonicaljson` shape (sorted-unique digests); profile interpretation is the checkpoint consumer/publication owner | `authority_heads` binds the winning lease digest + union tails only (`collectAuthorityHeads`); profile semantics never consulted | Effective-profile resolution stays with checkpoint consumers |

## SelectionPlan relationships (14.7.2)

| # | Spec clause | Relationship | Canonical shape owner | Shared admission (this leaf) | Accepted caller/publication boundary |
|---|-------------|--------------|----------------------|------------------------------|--------------------------------------|
| P1 | 14.7.2 member table (16) | Resolve once; bind all members; `lease_record_id` + triple from the same winning record; heads after union | Shapes: `checkPlanShapes` | `bindPlan` (build) + `Revalidate` (fixed compare order, union, no re-resolve, no retarget) | Effect authorization (fencing/commit/retry/recovery/remote dispatch/admission/transport-resume) is the CLI/lifecycle owner invoking `Revalidate` via `BoundariesFor`; wire Result 5/exit rendering likewise |
| P2 | 14.7.2 revalidation | Changed mapping/alias/allowlist/transport/index/record/heads/lease/action/destination/expectation invalidates (`selector_plan_stale`); revocation is `peer_not_allowlisted`; failed reads keep their class | — | `Revalidate` + `validateAuthorityUnion` + `collectAuthorityHeads` | Durable bootstrap recovery (14.7.4) and crash/idempotency protocols stay with their existing owners |

## Unguarded-relationship audit (owned, before this rework)

Every owned relationship above ships a production-entry regression EXCEPT C4,
which canonical digests alone could not establish (CR6 P1: a swapped-variant
checkpoint keeps canonical identity, session, lease, epoch, and creator, and
was admitted by all four entries including old-plan revalidation). C4 is
repaired by this rework with positive direct + task_board controls and
wrong-variant negatives through all four entries plus old-plan revalidation,
and a qualifying narrowing mutant. No owned relationship is declared delegated:
C5/C6 and P1-caller rows name a delegated API that already carries its
necessary input (canonical checkpoint bytes incl. safe boundary; CLI/lifecycle
`Revalidate` invocation at `BoundariesFor` boundaries); C4's variant selection
input (Session Record kind) arrives via the winning-source projection already
bound by S1, so no new cross-owner input is required.
