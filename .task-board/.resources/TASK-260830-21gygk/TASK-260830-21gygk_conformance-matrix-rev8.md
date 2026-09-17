# TASK-260830-21gygk — Clause-to-owner conformance matrix (rev8 rework)

Authority: `internal/specdoc/SPEC.v0.6.0.md` (AX v0.6.0,
`0cbdf100dbf84df50c64f792b1f940e3a67859a6`), Sections 5.3, 5.4 with
referenced 2.4 and 5.2, plus 14.7.2 for the plan rows.
Scope: every Session/Lease/Checkpoint relationship this task owns.
For each row: the canonical shape owner, the shared related-record
admission owned by `internal/sessquery/lease.go` (with production call
site), and the previously accepted caller/publication boundary that
this leaf MUST NOT absorb. "Entries" = the four shared production
entries `BuildPlan`, `Revalidate` (fresh + old-plan), `AuthoritativeStatus`,
`AuthoritativeList`, all funnelling through `winningLeaseFor`.

Rev8 correction: C6 no longer checks only head existence/tuples.
The owned admission is implemented here with the winning-source
chain handle AND the actual canonical event payloads, Session
Record, and admitted checkpoint closure. Profile derivation is
consumed authority, not a delegation claim: every input below is a
real API on the threaded repo handle.

## Session Record relationships

| # | Spec clause | Relationship | Canonical shape owner | Shared admission (this leaf) | Accepted caller/publication boundary (out of scope) |
|---|---|-------------|--------------|----------------------|------------------------------|
| S1 | 5.1 + 14.7.2 `session_record_id` | `session_id`/`subject_id` UUIDv7 + equality; `record_id` canonical self digest; immutable record identity | `canonicaljson` closed shape + `sessrepo` attestation at the durable boundary (`sessrepo.AttestSessionRecord` in `store.go:CreateSession`; verified by `specdoc060_test`); `sessstate.DecodeRecord` projects members without re-attesting (accepted division, `sessstate/doc.go:24-32`) | Consumes `SessionID`/`RecordID`/`Kind` from the winning-source projection in `bindPlan` (`plan.go:267`), `Revalidate` (`revalidate.go:91`), `authorize` via `Status`/`List`; cross-source digest disagreement is `integrity_failure` (`checkUnionCopy` `revalidate.go:229`, `selectIdentity`, `checkRecordAgreement`) | Record creation/publication is the `sessrepo` owner; no leaf mints records |
| S2 | 5.1 `kind` | `kind` in {`direct`, `task_board`}; `task_board` object required iff `kind = task_board` | `canonicaljson.validateSessionRecordV1` (`requireEnum` + kind-conditional `task_board` null/object check, `core_records.go`) | Consumes `Kind` from the projection; binds every required checkpoint's persistence variant to it (`checkCheckpointPersistence` `lease.go:555`, CR6) | Manifest publication/storage/transport for either path stays caller-owned |
| S3 | 2.3 + 14.7.1 | Name grammar, exact tiers, ASCII-collision ambiguity, UUID/qualified selection, stable sorting | `sessrepo.Resolve` (local tier, `store.go:113`) + `canonicaljson` name grammar | `Reader.Resolve` single shared path (`resolveBare`/`resolveExplicit` in `query.go:247`); no duplicate resolver | CLI rendering/exit mapping is the CLI leaf (accepted stated bound; no `ax` command in this tree) |

Unchanged since rev7; reverified by the green full suite on final source.

## Lease Record relationships (5.3)

| # | Spec clause | Relationship | Canonical shape owner | Shared admission (this leaf) | Accepted caller/publication boundary |
|---|---|-------------|--------------|----------------------|------------------------------|
| L1 | 5.3 table | Closed shape + canonical self identity (`record_id`); all members required incl. nullables | `canonicaljson` (`sessrepo.AttestLeaseRecord` in `internal/sessrepo/lease.go:22`); `parseLeaseRecord` (`lease.go:72`) maps failures to `invalid_config` | Absence for a session is `observation_unavailable`, never a minted triple (`winningLeaseFor` absence gate `lease.go:333`) | Lease publication (takeover/recovery writers) is caller-owned |
| L2 | 5.3 table | `session_id`/`subject_id`/epoch/holder/lease-ID grammars; `created_by == issued_by` (admitted, not re-gated) | `canonicaljson` + `parseLeaseRecord` member checks (`lease.go:89-127`) | Grammar failures refuse `invalid_config` before any authority use | — |
| L3 | 5.3 succession | Winner = greatest (`epoch`, `lease_id`), bytewise tie-break | — (rule implemented here) | `winningLeaseFor` via `sessstate.Compare` (`lease.go:337`) | Union winner across sources uses the same rule (`checkUnionCopy`); envelope triples never substitute |
| L4 | 5.3 succession | Predecessor ancestry: epoch-1 root has null predecessor; epoch > 1 names a known same-session predecessor at epoch exactly +1 | — | `checkLeaseChain` (`lease.go:371`): missing name/record is `observation_unavailable`; contradictory/duplicate/self/cyclic/skipped links are `integrity_failure` | — |
| L5 | 5.3 handoff base | Epoch > 1 references a validated checkpoint for its session + predecessor lease; epoch-1 `create` MAY omit before the first provider boundary | — | `checkWinnerCheckpoint` + `checkAncestorCheckpoints` over the whole winning ancestry (CR5, `lease.go:435,487`); unresolvable/misbound is `observation_unavailable` | Checkpoint publication (quiescence-gated writers) is caller-owned |

Unchanged since rev7; reverified by the green full suite on final source.

## Checkpoint Record relationships (5.4)

| # | Spec clause | Relationship | Canonical shape owner | Shared admission (this leaf) | Accepted caller/publication boundary |
|---|---|-------------|--------------|----------------------|------------------------------|
| C1 | 5.4 table | Closed shape + canonical self identity (`checkpoint_id`); **exactly one** of `provider_manifest_id` / `task_board_bundle_id` | `canonicaljson` checkpoint validator (`requireNullableDigestPresence` + xor, `core_records.go:147-165`) via `sessrepo.AttestCheckpointRecord` (`internal/sessrepo/checkpoint.go:22`); `parseCheckpointRecord` maps failures to `invalid_config` | Shape alone never admits: every required reference is re-bound below | Both-null/both-non-null (CP-N2/CP-N3 class) already refuse at the canonical owner before publication |
| C2 | 5.4 `lease_epoch`/`lease_id` | Bound to the referenced winning lease: successor checkpoints name the predecessor lease; epoch-1 root checkpoints name self | — | `checkWinnerCheckpoint` (winner) + `checkCheckpointBinding` (every necessary ancestor, `lease.go:526`) | — |
| C3 | 5.4 `created_by_host_id` | Creator is the current (owning) lease holder | — | Same two sites as C2 (CR5 holder binding); mismatch is `observation_unavailable` | — |
| C4 | 5.4 persistence variant | Referenced Session Record selects the variant: `direct` requires provider non-null + bundle null; `task_board` requires the reverse | Canonical owner proves exactly-one-present (C1) but its no-Session-Record API **cannot** select the variant | `checkCheckpointPersistence` (`lease.go:555`, CR6), called for the winner and every necessary ancestor from all four entries; mismatch is `observation_unavailable` | Which store persists/publishes bytes (provider Transfer Manifest vs task-board bundle) stays caller-owned; this leaf checks the admitted variant, never the bytes |
| C5 | 5.4 safe boundary + quiescence | Closed Safe Boundary Evidence; publication requires all-idle + zero counters; `status = validated`; `workspace_manifest_id` digest | `canonicaljson.validateSafeBoundaryEvidence` (`core_records.go:167`, requires `input_blocked/foreground_idle/background_idle=true`, counters zero) + status/digest requires (publication gate, CP-N1/CP-N4 class, verified by `TestCheckpointPublishRefusals` in `sessrepo`) | NOT re-checked: admitted records already carry the owner's verdict; re-gating diagnostics would fork the owner | Checkpoint publication/storage/transport is caller-owned; no demand is made here to implement manifest storage |
| C6a | 5.4 `event_heads` shape + resolution | Sorted unique heads 1..64; each head resolves to a chained event for the same session at or before the owning lease | `canonicaljson` shape (sorted-unique 1..64 digests, `core_records.go:147-153`); `parseCheckpointRecord` extracts `EventHeads` (`lease.go:253`, `invalid_config` on grammar failure) | OWNED (rev7, retained): `checkCheckpointEventHeads` (`lease.go:551`), called for the winner (`checkWinnerCheckpoint`) and every necessary ancestor (`checkCheckpointBinding`) from all four entries via `winningLeaseFor(sessionID, kind, repo)`: `BuildPlan` passes `selected.repo` (`plan.go:274`), `Revalidate` passes the plan-source `repo` (`revalidate.go:101`), `AuthoritativeStatus` passes `selected.repo` (`summary.go:320`), `AuthoritativeList` passes `reader.Local` (`summary.go:340`). Each head must name a chained event for the same session at or before its bound lease (`repo.ListEvents`); missing (incl. cross-session digests, losing-lease preserved blobs outside the authoritative chain, synthetic placeholders), later-epoch, and same-epoch foreign-lease heads refuse `observation_unavailable`. Historical heads admissible, never required to equal the current tail. Tests: `TestRev7CheckpointHeadAuthority` (winner/ancestor × missing/later through all four entries + old-plan) with the `observation_unavailable` class pin (`rev7CheckHeadRefusalClass`); mutant `N-checkpoint-heads` admits exactly the all-zero head and is killed by the missing_head negatives surfacing the wrong class. | Effective-profile derivation over the closure is the C6b admission below, not a consumer concern. Bundle/resume/fork materialization APIs do not exist in this tree; no generic consumer prose is invoked. |
| C6b | 5.4 closure + 2.4 profile, 5.2 pairs | Transitive closure fixes the effective execution profile and its nullable source; first launch carries the Session Record creation profile with null; later launch/resume pairs carry the newest authoritative `profile.changed` at or before them; missing/losing/non-newest references are `integrity_failure`; no later local-only fallback, no creation fallback | Session Record creation profile: `canonicaljson` closed shape requires `execution_profile` standard\|yolo (`closed_shapes.go:207,229,285`) via `sessrepo` record admission; event pair/change payloads: `canonicaljson` closed tagged union (`core_records.go:526,533,539-540,544` — pair members, `fork.created` null-source pin, `profile.changed` from≠to) enforced at append/attest time | OWNED (rev8): `checkCheckpointProfileAuthority` (`lease.go`), called after the heads gate for the winner and every necessary ancestor from all four entries through the same `winningLeaseFor(sessionID, kind, repo)` threading (same four repo handles as C6a). Inputs are all already-supplied authority on that handle: `repo.GetRecord` (creation profile + genesis record digest), `repo.ListEvents` (index order, types, predecessors for the transitive closure walk; genesis link must name the record, any other unknown predecessor is `integrity_failure`; divergent blobs outside the index never enter), `repo.GetEvent` (pair/change payloads, fetched only for in-closure launch/resume/change events). Derivation in chain order: first launch pinned to (creation, null); every later `provider.launched`/`task_board.launched`/`session.resumed` pinned to the newest in-closure `profile.changed` at or before it with that change's `to` profile (creation pair while none exists). Only `to` feeds derivation; `from`/confirmation belong to the publication flow. Violations refuse `integrity_failure` with the violated expectation named. `fork.created` is not re-derived: its pair is new-session authority plus cross-session provenance (2.4), unresolvable from this closure, and the canonical owner already pins its null source. Failed blob/record reads keep their cause inside `observation_unavailable`. Tests: ported `TestRev8CheckpointProfileAuthority` (direct/task_board × winner/ancestor × missing-source through all four entries); `TestRev8ProfileChangedPositiveControl`, `TestRev8ProfileResumePositiveControl`, `TestRev8ProfileTwoGenerationControl` (real changes, fresh-plan revalidation); `TestRev8ProfileSourceRefusals` (wrong-type, non-newest, stale-value, first-wrong-profile, dangling-resume-source, cross-session, losing-branch via a preserved divergent blob, dangling-predecessor — integrity class pinned); `TestRev8ProfileHistoricalClosureAdmits` (post-head change never consulted); `TestRev8ProfileOldPlanSubstitution` (old plan refuses a substituted checkpoint whose valid heads cover a stale pair). Mutants: `N-profile-first-source` (admits exactly the all-zero source where null is wanted), `N-profile-newest` (first change wins), `N-profile-value` (admits exactly a yolo-valued mismatch), `N-profile-change-direction` (still matches `profile.changed` and reads `to` but derives from `from`; token-preserving behavioral mutant). | Out of this admission by normative text, not by delegation: `fork.created.source_profile_event_id` provenance (2.4: never participates in new-session derivation, unresolvable from the winning-source closure); `session.resumed.checkpoint_id` existence (not a profile relationship); `task_board.launched` creation-lease repeat and provider/launch-mode binding (5.2 non-pair relationship, no fixture demand); `profile.changed.from` continuity and `confirmed` gating (publication-flow concerns). A citation of a well-formed post-head digest from inside the closure has no fixture by causal necessity (chain continuity forces every prior event into the closure, so a writer cannot cite its future); non-consultation is proven by the historical control instead. |

## SelectionPlan relationships (14.7.2)

| # | Spec clause | Relationship | Canonical shape owner | Shared admission (this leaf) | Accepted caller/publication boundary |
|---|---|-------------|--------------|----------------------|------------------------------|
| P1 | 14.7.2 member table (16) | Resolve once; bind all members; `lease_record_id` + triple from the same winning record; heads after union | Shapes: `checkPlanShapes` (`revalidate.go:385`) | `bindPlan` (build) + `Revalidate` (fixed compare order, union, no re-resolve, no retarget) | Effect authorization (fencing/commit/retry/recovery/remote dispatch/admission/transport-resume) is the CLI/lifecycle owner invoking `Revalidate` via `BoundariesFor` (`plan.go:116`, verified by `TestActionBoundaryTable`); wire Result 5/exit rendering likewise |
| P2 | 14.7.2 revalidation | Changed mapping/alias/allowlist/transport/index/record/heads/lease/action/destination/expectation invalidates (`selector_plan_stale`); revocation is `peer_not_allowlisted`; failed reads keep their class | — | `Revalidate` + `validateAuthorityUnion` + `collectAuthorityHeads` | Durable bootstrap recovery (14.7.4) and crash/idempotency protocols stay with their existing owners |

Unchanged since rev7; reverified by the green full suite on final source. Profile revalidation needs no new compare member: the profile gate re-runs inside `winningLeaseFor` on the plan-source repo at every `Revalidate` (fresh and old-plan alike), so substituted profile authority refuses before any digest compare.

## Unguarded-relationship audit (owned, rev8)

Every owned relationship above ships a production-entry regression.
CR8 C6b (profile authority) was unguarded: a first launch citing a
dangling source kept canonical identity, session, lease, epoch,
creator, variant, and head resolution, and was admitted by all four
entries including old-plan revalidation. It is repaired by this
rework with real-change positive controls and
missing/wrong-type/wrong-session/wrong-lease(non-newest, losing,
cross-session)/non-newest/stale-value/first-launch/dangling-closure
negatives on winner and necessary-ancestor paths through all four
entries plus old-plan substitution, and four narrowing mutants.
No owned relationship is declared delegated: C5 names the canonical
publication gate that already carries its inputs (checkpoint bytes
incl. safe boundary, `TestCheckpointPublishRefusals`); P1-caller
names `BoundariesFor` with its action-tag input
(`TestActionBoundaryTable`); C6a names `checkCheckpointEventHeads`
with its winning-source chain input (`repo.ListEvents`,
`TestRev7CheckpointHeadAuthority` + class pin); C6b names
`checkCheckpointProfileAuthority` with its winning-source
record/event/closure inputs (`repo.GetRecord`, `repo.GetEvent`,
`repo.ListEvents`, rev8 suite). C4's variant input (Session Record
kind) arrives via the winning-source projection already bound by
S1, so no new cross-owner input was required; C6b's chain inputs
arrive via the winning-source repo already threaded for
projection, so no new store was introduced. The C6b out-of-scope
notes above cite the normative sentences that place each item
outside closure admission; none of them names a future API as an
authority owner.
