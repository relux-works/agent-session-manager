# TASK-260830-21gygk — Clause-to-owner conformance matrix (rev7 rework)

Authority: `internal/specdoc/SPEC.v0.6.0.md` (AX v0.6.0,
`0cbdf100dbf84df50c64f792b1f940e3a67859a6`), Sections 5.3, 5.4, 14.7.2.
Scope: every Session/Lease/Checkpoint relationship this task owns.
For each row: the canonical shape owner, the shared related-record
admission owned by `internal/sessquery/lease.go` (with production call
site), and the previously accepted caller/publication boundary that
this leaf MUST NOT absorb. "Entries" = the four shared production
entries `BuildPlan`, `Revalidate` (fresh + old-plan), `AuthoritativeStatus`,
`AuthoritativeList`, all funnelling through `winningLeaseFor`.

Rev7 correction: C6 no longer delegates event-head closure to a generic
checkpoint consumer. The owned admission is implemented here with the
winning-source chain handle; profile derivation remains outside with
named real evidence (see C6).

## Session Record relationships

| # | Spec clause | Relationship | Canonical shape owner | Shared admission (this leaf) | Accepted caller/publication boundary (out of scope) |
|---|---|-------------|--------------|----------------------|------------------------------|
| S1 | 5.1 + 14.7.2 `session_record_id` | `session_id`/`subject_id` UUIDv7 + equality; `record_id` canonical self digest; immutable record identity | `canonicaljson` closed shape + `sessrepo` attestation at the durable boundary (`sessrepo.AttestSessionRecord` in `store.go:CreateSession`; verified by `specdoc060_test`); `sessstate.DecodeRecord` projects members without re-attesting (accepted division, `sessstate/doc.go:24-32`) | Consumes `SessionID`/`RecordID`/`Kind` from the winning-source projection in `bindPlan` (`plan.go:267`), `Revalidate` (`revalidate.go:91`), `authorize` via `Status`/`List`; cross-source digest disagreement is `integrity_failure` (`checkUnionCopy` `revalidate.go:229`, `selectIdentity`, `checkRecordAgreement`) | Record creation/publication is the `sessrepo` owner; no leaf mints records |
| S2 | 5.1 `kind` | `kind` in {`direct`, `task_board`}; `task_board` object required iff `kind = task_board` | `canonicaljson.validateSessionRecordV1` (`requireEnum` + kind-conditional `task_board` null/object check, `core_records.go`) | Consumes `Kind` from the projection; binds every required checkpoint's persistence variant to it (`checkCheckpointPersistence` `lease.go:555`, CR6) | Manifest publication/storage/transport for either path stays caller-owned |
| S3 | 2.3 + 14.7.1 | Name grammar, exact tiers, ASCII-collision ambiguity, UUID/qualified selection, stable sorting | `sessrepo.Resolve` (local tier, `store.go:113`) + `canonicaljson` name grammar | `Reader.Resolve` single shared path (`resolveBare`/`resolveExplicit` in `query.go:247`); no duplicate resolver | CLI rendering/exit mapping is the CLI leaf (accepted stated bound; no `ax` command in this tree) |

## Lease Record relationships (5.3)

| # | Spec clause | Relationship | Canonical shape owner | Shared admission (this leaf) | Accepted caller/publication boundary |
|---|---|-------------|--------------|----------------------|------------------------------|
| L1 | 5.3 table | Closed shape + canonical self identity (`record_id`); all members required incl. nullables | `canonicaljson` (`sessrepo.AttestLeaseRecord` in `internal/sessrepo/lease.go:22`); `parseLeaseRecord` (`lease.go:72`) maps failures to `invalid_config` | Absence for a session is `observation_unavailable`, never a minted triple (`winningLeaseFor` absence gate `lease.go:333`) | Lease publication (takeover/recovery writers) is caller-owned |
| L2 | 5.3 table | `session_id`/`subject_id`/epoch/holder/lease-ID grammars; `created_by == issued_by` (admitted, not re-gated) | `canonicaljson` + `parseLeaseRecord` member checks (`lease.go:89-127`) | Grammar failures refuse `invalid_config` before any authority use | — |
| L3 | 5.3 succession | Winner = greatest (`epoch`, `lease_id`), bytewise tie-break | — (rule implemented here) | `winningLeaseFor` via `sessstate.Compare` (`lease.go:337`) | Union winner across sources uses the same rule (`checkUnionCopy`); envelope triples never substitute |
| L4 | 5.3 succession | Predecessor ancestry: epoch-1 root has null predecessor; epoch > 1 names a known same-session predecessor at epoch exactly +1 | — | `checkLeaseChain` (`lease.go:371`): missing name/record is `observation_unavailable`; contradictory/duplicate/self/cyclic/skipped links are `integrity_failure` | — |
| L5 | 5.3 handoff base | Epoch > 1 references a validated checkpoint for its session + predecessor lease; epoch-1 `create` MAY omit before the first provider boundary | — | `checkWinnerCheckpoint` + `checkAncestorCheckpoints` over the whole winning ancestry (CR5, `lease.go:435,487`); unresolvable/misbound is `observation_unavailable` | Checkpoint publication (quiescence-gated writers) is caller-owned |

## Checkpoint Record relationships (5.4)

| # | Spec clause | Relationship | Canonical shape owner | Shared admission (this leaf) | Accepted caller/publication boundary |
|---|---|-------------|--------------|----------------------|------------------------------|
| C1 | 5.4 table | Closed shape + canonical self identity (`checkpoint_id`); **exactly one** of `provider_manifest_id` / `task_board_bundle_id` | `canonicaljson` checkpoint validator (`requireNullableDigestPresence` + xor, `core_records.go:147-165`) via `sessrepo.AttestCheckpointRecord` (`internal/sessrepo/checkpoint.go:22`); `parseCheckpointRecord` maps failures to `invalid_config` | Shape alone never admits: every required reference is re-bound below | Both-null/both-non-null (CP-N2/CP-N3 class) already refuse at the canonical owner before publication |
| C2 | 5.4 `lease_epoch`/`lease_id` | Bound to the referenced winning lease: successor checkpoints name the predecessor lease; epoch-1 root checkpoints name self | — | `checkWinnerCheckpoint` (winner) + `checkCheckpointBinding` (every necessary ancestor, `lease.go:526`) | — |
| C3 | 5.4 `created_by_host_id` | Creator is the current (owning) lease holder | — | Same two sites as C2 (CR5 holder binding); mismatch is `observation_unavailable` | — |
| C4 | 5.4 persistence variant | Referenced Session Record selects the variant: `direct` requires provider non-null + bundle null; `task_board` requires the reverse | Canonical owner proves exactly-one-present (C1) but its no-Session-Record API **cannot** select the variant | `checkCheckpointPersistence` (`lease.go:555`, CR6), called for the winner and every necessary ancestor from all four entries; mismatch is `observation_unavailable` | Which store persists/publishes bytes (provider Transfer Manifest vs task-board bundle) stays caller-owned; this leaf checks the admitted variant, never the bytes |
| C5 | 5.4 safe boundary + quiescence | Closed Safe Boundary Evidence; publication requires all-idle + zero counters; `status = validated`; `workspace_manifest_id` digest | `canonicaljson.validateSafeBoundaryEvidence` (`core_records.go:167`, requires `input_blocked/foreground_idle/background_idle=true`, counters zero) + status/digest requires (publication gate, CP-N1/CP-N4 class, verified by `TestCheckpointPublishRefusals` in `sessrepo`) | NOT re-checked: admitted records already carry the owner's verdict; re-gating diagnostics would fork the owner | Checkpoint publication/storage/transport is caller-owned; no demand is made here to implement manifest storage |
| C6 | 5.4 `event_heads` + 2.4 profile | Sorted unique heads 1..64; transitive closure fixes the effective execution profile; no later local-only fallback | `canonicaljson` shape (sorted-unique 1..64 digests, `core_records.go:147-153`); `parseCheckpointRecord` extracts `EventHeads` (`lease.go:253`, `invalid_config` on grammar failure) | OWNED (rev7, corrected): `checkCheckpointEventHeads` (`lease.go:551`), called for the winner (`checkWinnerCheckpoint`) and every necessary ancestor (`checkCheckpointBinding`) from all four entries via `winningLeaseFor(sessionID, kind, repo)`: `BuildPlan` passes `selected.repo` (`plan.go:274`), `Revalidate` passes the plan-source `repo` (`revalidate.go:101`), `AuthoritativeStatus` passes `selected.repo` (`summary.go:320`), `AuthoritativeList` passes `reader.Local` (`summary.go:340`). Each head must name a chained event for the same session at or before its bound lease (`repo.ListEvents`); missing (incl. cross-session digests, losing-lease preserved blobs outside the authoritative chain, synthetic placeholders), later-epoch, and same-epoch foreign-lease heads refuse `observation_unavailable`. Historical heads admissible, never required to equal the current tail. Tests: `TestRev7CheckpointHeadAuthority` (winner/ancestor × missing/later through all four entries + old-plan), `TestRev7IndependentPersistence` controls; mutant `N-checkpoint-heads` admits exactly the all-zero head and is killed by the missing_head negatives. | Effective-profile derivation (newest authoritative `profile.changed`, bundle/resume/fork carrying P1/E1 per 2.4 fixtures) is NOT performed by any `sessquery` entry: `grep -rn profile internal/sessquery/*.go` shows only fixture payloads and the `session.set-profile` action tag, no derivation; `sessstate/doc.go:69-72` explicitly excludes persisted-profile derivation (profile pairs retained as inert facts); `provhost.ProfileMapping` (`profile.go:57`) maps strings to adapter flags only. Bundle/resume/fork materialization APIs do not exist in this tree. The owned later-lease bound above already prevents consulting a later local-only event through an admitted checkpoint; full P1/E1 bundle checks belong to those future materialization owners, not this selector leaf. No generic consumer prose is invoked. |

## SelectionPlan relationships (14.7.2)

| # | Spec clause | Relationship | Canonical shape owner | Shared admission (this leaf) | Accepted caller/publication boundary |
|---|---|-------------|--------------|----------------------|------------------------------|
| P1 | 14.7.2 member table (16) | Resolve once; bind all members; `lease_record_id` + triple from the same winning record; heads after union | Shapes: `checkPlanShapes` (`revalidate.go:385`) | `bindPlan` (build) + `Revalidate` (fixed compare order, union, no re-resolve, no retarget) | Effect authorization (fencing/commit/retry/recovery/remote dispatch/admission/transport-resume) is the CLI/lifecycle owner invoking `Revalidate` via `BoundariesFor` (`plan.go:116`, verified by `TestActionBoundaryTable`); wire Result 5/exit rendering likewise |
| P2 | 14.7.2 revalidation | Changed mapping/alias/allowlist/transport/index/record/heads/lease/action/destination/expectation invalidates (`selector_plan_stale`); revocation is `peer_not_allowlisted`; failed reads keep their class | — | `Revalidate` + `validateAuthorityUnion` + `collectAuthorityHeads` | Durable bootstrap recovery (14.7.4) and crash/idempotency protocols stay with their existing owners |

## Unguarded-relationship audit (owned, rev7)

Every owned relationship above ships a production-entry regression.
CR6 C4 (persistence variant) was repaired with direct + task_board
controls and wrong-variant negatives through all four entries plus
old-plan revalidation and mutant `N-checkpoint-persistence`.
CR7 C6 (event-head closure) was unguarded: a swapped-head checkpoint
kept canonical identity, session, lease, epoch, creator, and variant,
and was admitted by all four entries including old-plan revalidation.
It is repaired by this rework with real-head positive controls and
missing/later-lease negatives on winner and necessary-ancestor paths
through all four entries plus old-plan revalidation, and narrowing
mutant `N-checkpoint-heads`. No owned relationship is declared
delegated: C5 names the canonical publication gate that already
carries its inputs (checkpoint bytes incl. safe boundary,
`TestCheckpointPublishRefusals`); P1-caller names `BoundariesFor`
with its action-tag input (`TestActionBoundaryTable`); C6 now names
`checkCheckpointEventHeads` with its winning-source chain input
(`repo.ListEvents`, `TestRev7CheckpointHeadAuthority`). C4's variant
input (Session Record kind) arrives via the winning-source projection
already bound by S1, so no new cross-owner input was required; C6's
chain input arrives via the winning-source repo already threaded for
projection, so no new store was introduced.
