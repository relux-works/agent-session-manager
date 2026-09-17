# TASK-260830-21gygk — Clause-to-owner conformance matrix (rev11 rework)

Rev11 change (CR10 P1, fifth C6 member): the R5 resume row now admits its
REFERENCED Checkpoint Record through the shared semantic owner
(`checkReferencedCheckpointBinding`) before deriving. The standalone
census file `TASK-260830-21gygk_relation-census-rev11.md` carries the same
six pair rows plus the new K1-K7 record-consumption table; this matrix is
that census in clause form. Battery: 63 unique N/B plants (60 N, 3 B):
46 semantic-admission, 17 precision.

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

Rev10 correction: C6b is now the full relation census (six rows, one
per event kind x relationship) instead of two witnessed arms. The
standalone census file
`TASK-260830-21gygk_relation-census-rev10.md` carries the same rows
with per-row negatives and plants; this section is that census in
matrix form. P1-A (fork new-session pair) and P1-B (resume
referenced-closure pair) are admitted in the shared owner;
later-launch derivation is unchanged by normative text (launches
carry no `checkpoint_id`) and is pinned by retained positives plus
new task_board.launched rows.

Mutation accounting (rev11, extending the corrected CR9 P2 base): the battery
holds 63 unique N/B plants (60 N, 3 B): 46 semantic-admission plants
whose kill is a behavioral admission failure (43 retained + 3 new
`N-referenced-creator/variant/lease`), and 17
label/class/output-precision plants (the 3 B output-order plants
plus 14 N plants). `N-checkpoint-heads` is a class/order precision
kill (weakened heads surface the wrong class through the profile
backstop), not semantic admission. The three classifier controls
(harmless SURVIVED applied via the same harness, NOT_APPLIED,
COMPILE_OR_HARNESS_FAILURE) report separately and never join the kill
numerator. New-plant kills are full-suite behavioral admission failures
with per-plant raw logs and subprocess exits; the static census gate
passes for all three (token-preserving), so only the behavioral suites
fail.

## Session Record relationships

| # | Spec clause | Relationship | Canonical shape owner | Shared admission (this leaf) | Accepted caller/publication boundary (out of scope) |
|---|---|-------------|--------------|----------------------|------------------------------|
| S1 | 5.1 + 14.7.2 `session_record_id` | `session_id`/`subject_id` UUIDv7 + equality; `record_id` canonical self digest; immutable record identity | `canonicaljson` closed shape + `sessrepo` attestation at the durable boundary (`sessrepo.AttestSessionRecord` in `store.go:CreateSession`; verified by `specdoc060_test`); `sessstate.DecodeRecord` projects members without re-attesting (accepted division, `sessstate/doc.go:24-32`) | Consumes `SessionID`/`RecordID`/`Kind` from the winning-source projection in `bindPlan` (`plan.go:267`), `Revalidate` (`revalidate.go:91`), `authorize` via `Status`/`List`; cross-source digest disagreement is `integrity_failure` (`checkUnionCopy` `revalidate.go:229`, `selectIdentity`, `checkRecordAgreement`) | Record creation/publication is the `sessrepo` owner; no leaf mints records |
| S2 | 5.1 `kind` | `kind` in {`direct`, `task_board`}; `task_board` object required iff `kind = task_board` | `canonicaljson.validateSessionRecordV1` (`requireEnum` + kind-conditional `task_board` null/object check, `core_records.go`) | Consumes `Kind` from the projection; binds every required checkpoint's persistence variant to it (`checkCheckpointPersistence` `lease.go`, CR6) | Manifest publication/storage/transport for either path stays caller-owned |
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
| C4 | 5.4 persistence variant | Referenced Session Record selects the variant: `direct` requires provider non-null + bundle null; `task_board` requires the reverse | Canonical owner proves exactly-one-present (C1) but its no-Session-Record API **cannot** select the variant | `checkCheckpointPersistence` (`lease.go`, CR6), called for the winner and every necessary ancestor from all four entries; mismatch is `observation_unavailable` | Which store persists/publishes bytes (provider Transfer Manifest vs task-board bundle) stays caller-owned; this leaf checks the admitted variant, never the bytes |
| C5 | 5.4 safe boundary + quiescence | Closed Safe Boundary Evidence; publication requires all-idle + zero counters; `status = validated`; `workspace_manifest_id` digest | `canonicaljson.validateSafeBoundaryEvidence` (`core_records.go:167`, requires `input_blocked/foreground_idle/background_idle=true`, counters zero) + status/digest requires (publication gate, CP-N1/CP-N4 class, verified by `TestCheckpointPublishRefusals` in `sessrepo`) | NOT re-checked: admitted records already carry the owner's verdict; re-gating diagnostics would fork the owner | Checkpoint publication/storage/transport is caller-owned; no demand is made here to implement manifest storage |
| C6a | 5.4 `event_heads` shape + resolution | Sorted unique heads 1..64; each head resolves to a chained event for the same session at or before the owning lease | `canonicaljson` shape (sorted-unique 1..64 digests, `core_records.go:147-153`); `parseCheckpointRecord` extracts `EventHeads` (`lease.go:253`, `invalid_config` on grammar failure) | OWNED (rev7, retained): `checkCheckpointEventHeads` (`lease.go`), called for the winner (`checkWinnerCheckpoint`) and every necessary ancestor (`checkCheckpointBinding`) from all four entries via `winningLeaseFor(sessionID, kind, repo)`: `BuildPlan` passes `selected.repo` (`plan.go:274`), `Revalidate` passes the plan-source `repo` (`revalidate.go:101`), `AuthoritativeStatus` passes `selected.repo` (`summary.go:320`), `AuthoritativeList` passes `reader.Local` (`summary.go:340`). Each head must name a chained event for the same session at or before its bound lease (`repo.ListEvents`); missing (incl. cross-session digests, losing-lease preserved blobs outside the authoritative chain, synthetic placeholders), later-epoch, and same-epoch foreign-lease heads refuse `observation_unavailable`. Historical heads admissible, never required to equal the current tail. Tests: `TestRev7CheckpointHeadAuthority` (winner/ancestor x missing/later through all four entries + old-plan) with the `observation_unavailable` class pin (`rev7CheckHeadRefusalClass`); mutant `N-checkpoint-heads` admits exactly the all-zero head and is killed by the missing_head negatives surfacing the wrong class — a label/class precision kill (refusal order), not semantic admission. | Effective-profile derivation over the closure is the C6b census below, not a consumer concern. Bundle/resume/fork materialization APIs do not exist in this tree; no generic consumer prose is invoked. |

## C6b — profile-pair relation census (5.4 closure + 2.4 profile, 5.2 pairs)

Every row is DRIVEN through the four shared entries. The standalone
census file carries the same rows with full per-row negative tables;
status: 6 of 6 rows driven, 0 undriven, 0 scope exemptions.

| # | Spec clause | Relationship | Canonical shape owner | Shared admission (this leaf) | Accepted caller/publication boundary |
|---|---|-------------|--------------|----------------------|------------------------------|
| C6b-R1 | 5.2 pair sentence ("for the first launch, the Session Record and null") + 2.4 + 13.1 step 5 | provider.launched first launch = Session Record creation profile + null source | Session Record creation profile: `canonicaljson` closed shape requires `execution_profile` standard\|yolo via `sessrepo` record admission; event pair members: `canonicaljson` closed tagged union enforced at append/attest time | OWNED: `checkClosureProfilePairs` launch case, `firstLaunch` branch + `checkProfilePairSource` (`lease.go`). Inputs: `repo.GetRecord` (creation), `repo.ListEvents` (order), `repo.GetEvent` (pair). Tests: `TestRev8CheckpointProfileAuthority` (missing_source=false controls), `TestRev8ProfileSourceRefusals/first_wrong_profile`, rev10 fork/resume good-mode first launches. Plants: `N-profile-first-source` (admits exactly the all-zero source; killed by the missing-source negatives by admission), `N-profile-first-value` (admits exactly standard/null where yolo/null is wanted; killed by first_wrong_profile + task_board bad_first_profile by admission). | Non-newest/losing/wrong-session/later-local-only are N/A (a first launch cites no change by construction). Mid-derivation read failures are a defensive bound (helpers wrap the cause in unavailable; unreachable without a storage race). |
| C6b-R2 | 5.2 pair sentence + 2.4 lease/sequence derivation + 5.4 no-later-local-only | provider.launched later launch = newest authoritative `profile.changed` at or before the event in the admitting closure (+ target), else the creation pair. Launches carry no `checkpoint_id` (5.2 payload table), so the admitting closure in lease/sequence order is the only referenced authority. | Same pair/change payload owners as R1 (`core_records.go`, `from` != `to`) | OWNED: same launch case, `!firstLaunch && hasNewest` branch; source resolved against the admitting closure. Tests: `TestRev8ProfileChangedPositiveControl`, `TestRev8ProfileTwoGenerationControl`, `TestRev8ProfileSourceRefusals` (wrong-type, non-newest, stale-value, cross-session, losing-branch, dangling-predecessor; integrity pinned), `TestRev8ProfileHistoricalClosureAdmits`, `TestRev8ProfileOldPlanSubstitution`, rev10 ancestor later-launch positives. Plants: `N-profile-newest`, `N-profile-value`, token-preserving `N-profile-change-direction` (all admission kills). | A later launch citing a well-formed post-head digest has no fixture by causal necessity (stated bound, retained); non-consultation is proven by the historical control. |
| C6b-R3 | 5.2 pair sentence + 2.4 + 13.2 step 3 | task_board.launched first launch = Session Record creation profile + null source | Same owners as R1 | OWNED: shared launch case, `firstLaunch` branch (same arm as R1). Tests: `TestRev10TaskBoardLaunchProfileAuthority` good_first/good_historical (positives), bad_first_profile/bad_first_source (integrity, winner/ancestor). Plants: shared-arm `N-profile-first-source` (presence) and `N-profile-first-value` (value; kills through the task-board fixture). | Same N/A classes and read-failed bound as R1. The 5.2 non-pair relationship (creation-lease repeat, provider/launch-mode binding) stays outside: the reducer checks provider and creation lease; no launch-mode consumer is added here (retained boundary). |
| C6b-R4 | 5.2 pair sentence + 2.4 + 5.4 (same launch reasoning as R2) | task_board.launched later launch = newest authoritative change at or before the event (+ target), else the creation pair | Same owners as R1 | OWNED: shared launch case, later branch. Tests: `TestRev10TaskBoardLaunchProfileAuthority` second/third-launch positives, bad_later_stale, bad_later_non_newest (integrity, winner/ancestor), good_historical later-local-only control. Plants: shared-arm `N-profile-newest` (kills through bad_later_non_newest by admission), `N-profile-value`, `N-profile-change-direction`. | Wrong-type/cross-session/losing/dangling share the R2 helpers (driven there). Post-head citation: same causal-necessity bound as R2. |
| C6b-R5 | 5.2 pair sentence ("at the referenced checkpoint") + 2.4 PROFILE-*-RESUME + 13.10 resume derivation + 5.4 | session.resumed = effective pair of its OWN referenced checkpoint's event-head closure (newest change + target, else creation pair). Never the outer-walk prefix. | Pair members + `checkpoint_id:digest` (non-null, v1-v4 tables) via the closed union at append time | OWNED (rev10 P1-B + rev11 P1 referenced admission): `checkClosureProfilePairs` session.resumed case + `referencedCheckpointProfile` + `checkReferencedCheckpointBinding` (`lease.go`): `checkpoint_id` -> semantically admitted Checkpoint Record (`Reader.CheckpointRecords` through the shared owner: owning lease tuple in winning ancestry, creator-holder, Session.kind variant, head closure via the shared helpers; chain + kind threaded from `winningLeaseFor`) -> transitive closure over the winning-source index -> newest change. Source resolved against the REFERENCED closure. Missing/wrong-session/wrong-lease/wrong-creator/wrong-variant/unresolvable-head references refuse `observation_unavailable`; contradictory pairs refuse `integrity_failure`. Tests: ported `TestReview9ResumeReferencedCheckpoint`, `TestRev10ResumeReferencedCheckpoint` (10 modes x direct/task_board x winner/ancestor with class pins), `TestRev10ResumeOldPlanSubstitution` (integrity), `TestRev10ResumeReferencedWithdrawal` (unavailable), ported `TestReview10ReferencedCheckpointAdmission` (12 negatives + 4 controls, direct/task_board x winner/ancestor), `TestRev11ReferencedCheckpointAdmissionRefusalClass` (12 unavailable pins), `TestRev11ReferencedCheckpointAdmissionOldPlan` (12 old-plan unavailable), `TestRev11RecordConsumptionCensus` (call-graph gate), updated `TestRev8ProfileResumePositiveControl` + `dangling_resume_source`. Plants: `N-profile-resume-missing` (admission kill), `N-profile-resume-newest` (admission kill), `N-referenced-creator/variant/lease` (each admits exactly one invalid referenced member; census tokens preserved, behavioral admission kills). | Referenced-checkpoint existence as a generic storage service stays outside; access to the already-supplied checkpoint closure is owned. Mid-derivation read failures: defensive bound as in R1 (missing vs read-failed stay distinct errors). |
| C6b-R6 | 5.2 fork sentence + 2.4 fork boundary + 13.8 step 8 + 2.4 PROFILE-*-FORK | fork.created = newly persisted Session Record profile + null source. The fork lives in the new session chain, so the admitting record IS the new record. | Pair members via the closed union; the canonical owner pins fork `profile_source_event_id` to null at append time (observed refusal) | OWNED (rev10 P1-A): `checkClosureProfilePairs` fork.created case + `checkProfilePairSource` (creation, null). `source_profile_event_id` is never read. Tests: ported `TestReview9ForkLocalProfile`, `TestRev10ForkLocalProfile` (good/good_provenance/bad_profile x direct/task_board x winner/ancestor, integrity pinned), `TestRev10ForkOldPlanSubstitution` (integrity pinned). Plant: `N-profile-fork-value` (admits exactly a standard-valued fork; admission kill). | Non-null fork source has no chained fixture (canonical pins at append; stated bound with the owning package named). `source_profile_event_id` provenance is caller-owned cross-session input (2.4: never participates); good_provenance proves non-participation. Non-newest/losing/wrong-session/later-local-only are N/A (null source cites no change). |

## SelectionPlan relationships (14.7.2)

| # | Spec clause | Relationship | Canonical shape owner | Shared admission (this leaf) | Accepted caller/publication boundary |
|---|---|-------------|--------------|----------------------|------------------------------|
| P1 | 14.7.2 member table (16) | Resolve once; bind all members; `lease_record_id` + triple from the same winning record; heads after union | Shapes: `checkPlanShapes` (`revalidate.go:385`) | `bindPlan` (build) + `Revalidate` (fixed compare order, union, no re-resolve, no retarget) | Effect authorization (fencing/commit/retry/recovery/remote dispatch/admission/transport-resume) is the CLI/lifecycle owner invoking `Revalidate` via `BoundariesFor` (`plan.go:116`, verified by `TestActionBoundaryTable`); wire Result 5/exit rendering likewise |
| P2 | 14.7.2 revalidation | Changed mapping/alias/allowlist/transport/index/record/heads/lease/action/destination/expectation invalidates (`selector_plan_stale`); revocation is `peer_not_allowlisted`; failed reads keep their class | — | `Revalidate` + `validateAuthorityUnion` + `collectAuthorityHeads` | Durable bootstrap recovery (14.7.4) and crash/idempotency protocols stay with their existing owners |

Unchanged since rev7; reverified by the green full suite on final source. Profile revalidation needs no new compare member: the profile gate re-runs inside `winningLeaseFor` on the plan-source repo at every `Revalidate` (fresh and old-plan alike), including referenced-checkpoint resolution from the current record set (proven by the rev10 substitution and withdrawal tests), so substituted or withdrawn profile authority refuses before any digest compare.

## Unguarded-relationship audit (owned, rev10)

Every owned relationship above ships a production-entry regression.
CR9 P1-A (fork new-session pair) and P1-B (resume referenced-closure
pair) were unguarded: a corrupt fork pair kept canonical identity,
session, lease, epoch, creator, variant, and head resolution green
and was admitted by all four entries, and a resume citing a change
outside its referenced closure was admitted from the outer-walk
prefix. Both were repaired in rev10 with the six-row census as the
class-closing instrument. CR10 P1 (fifth C6 member, referenced
Checkpoint Record consumed as authority after only session equality
and head membership) was unguarded in rev10: 12 of 12
wrong-creator/variant/lease negatives admitted through all four
entries. It is repaired in rev11 by routing the referenced record
through the shared semantic owner with the winning ancestry and
Session.kind it needs, with the K1-K7 record-consumption table as the
class-closing instrument: every record the shared owner consumes
(Session Record, winner checkpoint, necessary-ancestor checkpoint,
REFERENCED resume checkpoint, lease records, profile.changed and
other events) names its admission owner, call site, positive control,
negatives, and narrowing plant; 7 of 7 rows driven, 0 undriven. No
owned relationship is declared delegated: C5 names the canonical
publication gate that already carries its inputs; P1-caller names
`BoundariesFor`; C6a names `checkCheckpointEventHeads`;
C6b-R1..R6 name `checkClosureProfilePairs` with its winning-source
record/event/closure inputs plus, for R5, the already-supplied
admitted Checkpoint Records and winning ancestry threaded from the
reader (no new store, transport, publication, or materialization was
introduced). The call-graph gate `TestRev11RecordConsumptionCensus`
fails if any K4 derivation reaches `profileClosure` without calling
`checkReferencedCheckpointBinding` first, and the three N-referenced
token-preserving plants prove the gate alone never suffices.
