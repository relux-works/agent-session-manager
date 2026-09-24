# TASK-260830-21gygk: implementation evidence

Authority: relux-works/agent-session-manager-spec v0.6.0, commit
`0cbdf100dbf84df50c64f792b1f940e3a67859a6`, adopted through
STORY-260908-18woqo. Primary scope is Sections 14.7 and 14.7.1 with
shared SelectionPlan construction and revalidation in 14.7.2, refining
Section 2.3 and the authoritative summaries in 5.7/14.7.3. Historical
scope is retained without weakening: v0.5.0 (commit
`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`), Sections 2.3, 5.1-5.2,
5.7, 14.4. This file records implementation evidence; it changes no
normative ownership and claims no public CLI delivery.

The v0.6.0 adoption decides the two product questions the partial leaf
left open. The qualified-selector syntax, source-peer versus
winning-owner target, and precedence are now normative text in 14.7.1:
split at the first literal `@`, exact configured alias or host identity
as the source, Section 2.3 tiers for bare input, name-first then UUID
inside one qualified source, and no explicit-source fallback. The
prior stop-line packet on those questions is superseded, not
re-opened. The creating/record-only routing clarification is applied:
creating is a Session Record AND an authoritative initial lease
(5.7/13.1 step 2); a record-only chain is an interrupted persistence
prefix for the bootstrap-recovery leaf (14.7.4), and list/status
entries refuse it with selector_bootstrap_incomplete instead of
returning an ownerless row.

## Acceptance rows

8 of 8 AC rows are driven at the shared-library production entries
named below; 0 of 8 are delivered as a public CLI surface, which is an
explicit caller-integration bound (no `ax` session command exists in
this tree), not a gap in the assigned shared behavior. CLI/Result-5
delivery, wire exit binding, lifecycle effects, and the 14.7.4
recovery flow belong to their owning leaves, which invoke this API at
every required boundary instead of resolving again.

| AC row | Production call site | Named test(s) | Evidence and bound |
| --- | --- | --- | --- |
| Bare UUID/name selectors | `Reader.Resolve` → `resolveBare` | `TestResolvePinnedPrecedence`, `TestResolveBareIdentityUnion`, `TestResolveAllowlistAndReadFailures` | Driven: Section 2.3 tiers preserved byte for byte, local-name-before-UUID, `id:` bypasses names for the durable union identity, replicated copies deduplicate on agreeing record digests with a bytewise host tie break. The repository-layer exact-name rule underneath the local tier stays pinned by `TestResolveLocalNameAndUUID` at the retained `Repository.Resolve` entry in `sessrepo`, not at this call site. |
| Qualified selectors | `Reader.Resolve` → `resolveExplicit` → `locateSource` | `TestResolveQualifiedSourcesSelectOneIndex`, `TestResolveQualifiedSourceNeverFallsBack`, `TestResolveQualifiedNameBeforeUUIDInSource`, `TestResolveExplicitSourceRefusals`, `TestResolveExplicitSourceReadFailures` | Driven: first-`@` literal grammar, `local`/`peer:` entire-suffix/`id:` host forms, single-source tiers with name-first then UUID, no fallback in either direction, unknown/disallowed/refusal classes. |
| Ambiguity | `matchName`; `selectIdentity`/`checkRecordAgreement` | `TestResolveExactNamesAndASCIICollisions`, `TestResolvePeerOrderAndReplicatedIdentity`, `TestResolveQualifiedAmbiguityAndExclusions`, `TestResolveBareIdentityUnion` | Driven: ASCII-fold collisions refuse within the reached tier and within one qualified source; replicated identical copies stay one identity; disagreeing record digests are integrity failure, never silent first match. Cross-copy agreement carries one narrowing mutant per call site (`N-identity-diverge`, `N-agreement-name`). |
| List summaries | `Reader.List` → `Repository.ListSessions` → `Projector.Project` + `checkSummaryRepresentable` | `TestListStatusDerivedFactsAndStableOrder`, `TestListStatusCheckpointAndProjectionFailure`, `TestParkedReadRecoveryAndMissingVsMalformed`, `TestCreatingSummaryCannotClaimClosedCLIResult`, `TestSummaryBootstrapRefusals`, `TestSummaryMixedListRefusesWhole`, `TestReviewerBootstrapSummaryMustRefuse` | Driven: real identity, provider, owner/lease/local role where known, state, newest checkpoint ID, parked reason/retry, bytewise session-ID order. A complete read proving a record without any valid lease refuses the whole list with selector_bootstrap_incomplete, including a mixed healthy/incomplete listing; no partial rows escape and none is silently omitted. |
| Status summaries | `Reader.Status` → `Resolve` → `Projector.Project` + gate; `Reader.AuthoritativeStatus/List` → `authorize` | Same list tests; `TestAuthoritativeStatusHealthy`, `TestAuthoritativeMissingHostMetadataRefuses`, `TestAuthoritativeUnknownLocalHostRefuses`, `TestAuthoritativeCheckpointRequiresTimestamp`, `TestAuthoritativeInputValidation` (including `unknown_capability_name`), `TestAuthoritativeOptionalObservations`, `TestAuthoritativeAdmitsFullCapabilityRegistry`, `TestAuthoritativeUnknownObservationsRefuse`, `TestAuthoritativeMissingCapabilitiesRefuses`, `TestAuthoritativeMissingProcessRefusesStatus`, `TestAuthoritativeMissingWorkspaceRefuses`, `TestAuthoritativeHostNameBound64`, `TestRev4UnknownCapabilityMustRefuse` | Driven: named/UUID/qualified/id healthy status; record-only status refuses bootstrap_incomplete. The authoritative layer binds owner, lease, and role from the validated winning Lease Record, owner display names only from validated host metadata (1..64), local roles only with a known local host, checkpoint timestamps only from validated observations, workspace_status from the closed five-value enum, capabilities from validated CapabilitySummary maps (0..7, only available may enable) keyed by exactly the provhost-owned Section 7.3 seven-name provider registry (any other name refuses invalid_config through both entries), process liveness as a required boolean for status, and warnings from the projection; unknown required observations refuse selector_observation_unavailable while established absence (workspace absent, empty non-nil capabilities) stays representable. Explicit `InspectLocal` keeps raw access to parked/tombstoned/recovery diagnostics and is not a public summary entry. |
| Stable deterministic sorting | `Reader.read`; `peerCandidates`; `selectIdentity` | `TestListStatusDerivedFactsAndStableOrder`, `TestResolvePeerOrderAndReplicatedIdentity`, `TestResolveBareIdentityUnion` | Driven: bytewise session-ID list order, deterministic bytewise peer-source order with duplicate-allowlist stability, bytewise holder-host tie break for identical copies. Each ordering carries a behavioral mutant (`B-peer-order`, `B-session-order`, `B-tie-break`). |
| SelectionPlan build and revalidate | `Reader.BuildPlan` → `bindPlan`; `Reader.Revalidate` (+ `validateAuthorityUnion` + `collectAuthorityHeads`); `ParsePlan`; `Reader.winningLeaseFor` → `checkLeaseChain` + shared checkpoint admission | `TestBuildPlanBindsCurrentFacts`, `TestBuildPlanPeerSourceFacts`, `TestBuildPlanRecordOnlySessionRefusesBootstrap`, `TestBuildPlanLocalSourceRequiresKnownHost`, `TestBuildPlanValidatesUnionAtBuild`, `TestPlanDigestAndMarshalRoundTrip`, `TestRevalidateDetectsEachFactChange` (including `changed_nonempty_record_refuses`), `TestRevalidateUnionLeaseDivergenceRefusesStale`, `TestRevalidateUnionTombstoneRefusesStale`, `TestRevalidateUnionAgreeingReplicaStaysCurrent`, `TestRevalidateUnionParkedCopyRefuses` (divergent and agreeing record arms, replacing the earlier parked-ignored expectation), `TestRevalidateUnionUnreadablePeerRefuses`, `TestRevalidateUnionNameDriftStaysCurrent`, `TestBuildWithLaggingLeaseCopy`, `TestRevalidateReadFailures`, `TestRevalidateMalformedPlans`, `TestActionBoundaryTable`, `TestReviewerOtherSourceDivergenceMustRefuse`, `TestReviewerPlanRequiresLeaseRecord`, `TestRev3RealLeaseRecordIdentity`, `TestRev3DifferentAttestationMustNotAuthorize`, `TestValidSuccessorBuildsAndRevalidates`, `TestSuccessorCycleMustRefuse`, `TestSkippedEpochMustRefuse`, `TestSelfPredecessorWithCheckpointMustRefuse`, `TestEpochOneWithPredecessorMustRefuse`, `TestEpochOneWithSelfBoundCheckpointBuilds`, `TestDanglingPredecessorMustRefuse`, `TestWrongSessionCheckpointMustRefuse`, `TestWrongPredecessorCheckpointMustRefuse`, `TestMalformedCheckpointBytesRefuseConfig`, `TestRev4SelfPredecessorMustRefuse`, `TestRev4MissingCheckpointMustRefuse`, `TestRev4ParkedAuthorityMustRefuse`, `TestRev4ChangedLiveRecordRefuses`, `TestRev4DifferentLeaseDigestRefuses`, `TestRev5AncestorCheckpointAuthority` (complete multi-generation control plus missing-ancestor refusal through build, old-plan revalidation, and both summaries), `TestRev5CheckpointCreatorMustBeHolder`, `TestRev5AncestorCheckpointBinding` (wrong-session, wrong-lease, and wrong-creator ancestor checkpoints through all four entries), `TestRev6CheckpointPersistenceMustMatchSession` (swapped persistence variant on winner and necessary-ancestor paths through build, old-plan revalidation, and both summaries, with direct controls), `TestRev6TaskBoardPersistenceControl` (task_board positive control plus provider-variant refusal through all four entries including old-plan), `TestRev6PersistenceMismatchIsObservationUnavailable` (variant refusal class), `TestRev7CheckpointHeadAuthority` (missing and later-lease heads through all four entries and old-plan with real-head controls and the observation_unavailable class pin), `TestRev7IndependentPersistence`, `TestRev8CheckpointProfileAuthority`, `TestRev8ProfileChangedPositiveControl`, `TestRev8ProfileResumePositiveControl`, `TestRev8ProfileTwoGenerationControl`, `TestRev8ProfileSourceRefusals`, `TestRev8ProfileHistoricalClosureAdmits`, `TestRev8ProfileOldPlanSubstitution`, `TestReview9ForkLocalProfile`, `TestReview9ResumeReferencedCheckpoint`, `TestRev10ForkLocalProfile`, `TestRev10ForkOldPlanSubstitution`, `TestRev10ResumeReferencedCheckpoint`, `TestRev10ResumeOldPlanSubstitution`, `TestRev10ResumeReferencedWithdrawal`, `TestRev10TaskBoardLaunchProfileAuthority`, `TestReview10ReferencedCheckpointAdmission`, `TestRev11ReferencedCheckpointAdmissionRefusalClass`, `TestRev11ReferencedCheckpointAdmissionOldPlan`, `TestReview11ReferencedTemporalAuthority`, `TestRev12ReferencedCheckpointLaterLease`, `TestRev11RecordConsumptionCensus`, `TestRev12RecordConsumptionCensusRejectsAlternatePaths`, `TestRev15AliasAwareRecordConsumptionCensus`, `TestRev12AdmissionPrecisionCensus` | Driven: all sixteen members bind once from validated Lease Records (canonical digest plus epoch, lease ID, and holder from the same winning record, selected by greatest (epoch, lease_id) with a fully validated succession: the predecessor ancestry chains by epoch plus one to an epoch-1 null-predecessor root, and every lease in that winning ancestry above epoch 1 references an admitted Checkpoint Record for its session and predecessor lease, itself bound at epoch 1 for the root, each created by the owning lease holder and each carrying the persistence variant the referenced Session Record kind selects, each resolving every event head to a chained event for its session at or before its bound lease in the winning source chain (historical heads admissible, never required to equal the current tail; missing, cross-session, losing-lease, and later-lease heads refuse observation_unavailable with the class pinned against the profile backstop), and each admitting its Section 2.4 profile authority over the same closure (first launch carries the creation pair; later launches carry the newest authoritative change at or before them; resumes carry their referenced checkpoint's closure pair with the referenced record itself admitted through the shared owner (owning lease tuple resolved in the winning ancestry, creator-holder, Session.kind variant, and head closure via the shared helpers; a consuming resume may use the same or a later lease, but never a later-owned checkpoint); forks carry the new record pair; contradictory pairs refuse integrity_failure while missing, wrong-session, wrong-lease, wrong-creator, wrong-variant, future-owned, and unresolvable-head referenced authority refuses observation_unavailable and later local-only changes are never consulted); profile derivation receives only `admittedCheckpoint`, and the go/types census rejects raw checkpoint values in any alternate package path. Narrowing mutants listed in the prior revisions remain covered by their attached evidence; rev12 adds the temporal guard and type-level boundary controls. The union winner is the greatest tuple across all allowed sources (lagging copies stay current, greater copies are stale); a parked copy of the pinned session in any required source fails the union closed with selector_observation_unavailable (never evidence, never absence); authority heads bind the winning lease digest plus every non-parked source tail, sorted unique. A same-name gain for another UUID never retargets the pinned plan; an agreeing replica never invalidates it. |
| Negative and refusal cases | All gates below | Every `Test*` refusal arm plus the battery | Driven: distinct classes for invalid arguments, invalid configuration, unknown source, disallowed peer, remote read failure, ambiguity, not found, integrity failure, bootstrap incompleteness, unavailable observations, and stale plans, each killed by narrowing. No unsupported capability is advertised (see bounds). |

## Refusal and recovery coverage

Every `N-` row is a genuine narrowing mutant: the gate stays present
and is weakened to admit exactly one member of the class it must
reject, and the named behavioral test fails through the delivered
harness. Whole-clause disables are not accepted as narrowing. `B-`
rows reverse deterministic orderings. Controls are reported
separately and never counted as kills.

| Gate | Real entry test | Narrowing attack (all killed) |
| --- | --- | --- |
| Folded-name ambiguity | `TestResolveExactNamesAndASCIICollisions` | N-collision-pair admits exactly two colliding identities. |
| Exact-name admission | Shared tiers: `TestResolveExactNamesAndASCIICollisions`; retained repository entry: `TestResolveLocalNameAndUUID` at `Repository.Resolve` in `sessrepo` | N-case-variant admits one non-exact selector at the shared tier and is killed by the sessquery collision test; N-repository-case-variant admits one non-exact query at the retained repository entry and is killed by the sessrepo test at that entry. |
| First-`@` literal split | `TestParseSelectorLiteralGrammar` | N-first-at-split keeps the `@` token and splits at the last one; an alias containing `@` misroutes. |
| `local` source literal | Same | N-local-prefix keeps the token and admits `local`-prefixed sources. |
| `id:` key literal | Same | N-id-case keeps the token and folds the prefix; `ID:` bypasses names. |
| Alias exactness | `TestSelectorConfigurationValidation/alias_casing_is_exact` | N-peer-alias-fold keeps the mapping and folds comparison; `WORK` selects `Work`. |
| `id:` name bypass | `TestResolveQualifiedNameBeforeUUIDInSource` | N-id-through-names routes one explicit `id:` key through the name tier; a UUID-shaped name shadows the identity. |
| No explicit-source fallback | `TestResolveQualifiedSourceNeverFallsBack` | N-explicit-fallback falls back to the bare union on exactly an explicit miss. |
| Unknown source refusal | `TestResolveExplicitSourceRefusals` | N-unknown-alias-local treats exactly an unknown alias as local. |
| Peer allowlist | `TestResolveExplicitSourceRefusals`, `TestResolveAllowlistAndReadFailures` | N-disallowed-admit reads exactly one disallowed peer; N-unlisted-peer reads one unlisted identity. |
| Duplicate learned index | `TestResolveAllowlistAndReadFailures` | N-duplicate-peer admits duplicates for one host ID. |
| Duplicate alias mapping | `TestSelectorConfigurationValidation/duplicate_alias_mapping` | N-dup-alias admits duplicate mappings with last-write-wins. |
| Live eligibility | `TestParkedReadRecoveryAndMissingVsMalformed`, `TestResolveExcludesTombstonedButListsIt`, `TestResolveQualifiedAmbiguityAndExclusions` | N-parked-id and N-tombstoned-id admit one excluded identity each. |
| Cross-copy agreement, identity tier | `TestResolveBareIdentityUnion` (disagreeing-copies arm) | N-identity-diverge admits exactly the idA record disagreement in `selectIdentity`; other identities still refuse. |
| Cross-copy agreement, name tier | `TestResolveBareIdentityUnion` (disagreeing-name-win arm) | N-agreement-name admits exactly the idA record disagreement in `checkRecordAgreement`; other identities still refuse. |
| Read failure propagation | `TestResolveAllowlistAndReadFailures`, `TestResolveExplicitSourceReadFailures`, `TestRevalidateReadFailures`, `TestRevalidateUnionUnreadablePeerRefuses` | N-read-failure-fallback treats repository-path failure as empty. Missing learned index, nil known index, missing root, and non-directory root are separately driven. |
| Projection failure propagation | `TestListStatusCheckpointAndPropagationFailure`, `TestRevalidateDetectsEachFactChange/invalid_projection_propagates` | N-projection-failure-fallback omits one identity on failed projection instead of propagating its failure. |
| Plan revocation | `.../revocation_refuses_allowlist` | N-rev-revocation admits exactly the revoked hostA peer; refusal degrades to stale configuration. |
| Plan configuration binding | `.../unrelated_mapping_change_changes_configuration` | N-rev-config admits exactly the ws-alias configuration for hostC; other changes still refuse. |
| Plan source binding | `.../alias_remap_changes_binding` | N-rev-binding admits exactly the ws remap onto hostB; the refusal degrades to stale configuration. |
| Plan index binding | `.../unrelated_session_change_changes_index` | N-rev-index admits exactly an index containing session idB; other changes still refuse. |
| Plan record binding | `.../record_replacement_changes_digest` | N-rev-record admits exactly the parked chain-mismatch record the replacement fixture presents (a replaced record.json against its chain index parks the session with an empty digest); the refusal degrades to the lease reason. |
| Plan lease binding | `.../successor_lease_changes_winner` | N-rev-lease admits exactly the leaseB successor envelope at the plan source; the refusal degrades to the heads reason. |
| Plan heads binding | `.../same_lease_event_changes_heads` | N-rev-heads admits exactly a two-event chain; the refusal degrades to the index reason. |
| Plan absence | `.../deletion_empties_selection` | N-rev-absent-substitute rebinds exactly a deleted selection to its first listed sibling. |
| Union record agreement | `TestReviewerOtherSourceDivergenceMustRefuse` (also `TestBuildPlanValidatesUnionAtBuild`) | N-union-record admits exactly the idB cross-source record disagreement; other contradictions still refuse. |
| Union lease agreement (greatest winner) | `TestRevalidateUnionLeaseDivergenceRefusesStale`, `TestBuildWithLaggingLeaseCopy` | N-union-lease admits exactly the leaseB greater union successor; lagging smaller tuples stay current and other greater tuples still refuse. |
| Union tombstone evidence | `TestRevalidateUnionTombstoneRefusesStale` | N-union-tombstone admits exactly the idB cross-source tombstone; other tombstone evidence still refuses. |
| Plan bootstrap gate | `TestBuildPlanRecordOnlySessionRefusesBootstrap` | N-plan-bootstrap admits exactly record-only idA past the gate; the shape backstop still refuses it with invalid_arguments. |
| Plan source-host gate | `TestBuildPlanLocalSourceRequiresKnownHost` | N-plan-source-host admits exactly idA past the local-host gate; the shape backstop still refuses the empty host with invalid_arguments. |
| Summary bootstrap gate | `TestReviewerBootstrapSummaryMustRefuse`, `TestCreatingSummaryCannotClaimClosedCLIResult`, `TestSummaryBootstrapRefusals`, `TestSummaryMixedListRefusesWhole` | N-summary-bootstrap admits exactly record-only idA into list/status; other interrupted prefixes still refuse. |
| Summary host metadata gate | `TestAuthoritativeMissingHostMetadataRefuses` | N-observation-host admits exactly owner hostA without metadata; other missing owners still refuse. |
| Summary workspace gate | `TestAuthoritativeMissingWorkspaceRefuses`, `TestAuthoritativeUnknownObservationsRefuse` | N-observation-workspace admits exactly idA without workspace status; other missing workspaces still refuse. |
| Summary capabilities gate | `TestAuthoritativeMissingCapabilitiesRefuses`, `TestAuthoritativeUnknownObservationsRefuse` | N-observation-capabilities admits exactly idA without capabilities; other missing capabilities still refuse. An empty non-nil map is established absence and stays representable. |
| Summary process gate | `TestAuthoritativeMissingProcessRefusesStatus` | N-observation-process admits exactly idA without process liveness for status; other missing liveness still refuses. Lists carry process only when supplied. |
| Summary host-name bound | `TestAuthoritativeHostNameBound64`, `TestAuthoritativeInputValidation/long_display_name` | N-host-bound64 admits exactly the 65-character host name; other overlong names still refuse invalid_config. |
| Lease admission gate | `TestReviewerPlanRequiresLeaseRecord`, `TestRev3DifferentAttestationMustNotAuthorize`, `TestRev3RealLeaseRecordIdentity` | N-lease-missing admits exactly idA with no admitted record past the absence gate with a placeholder; the production tests still refuse observation_unavailable. |
| Winner succession chain | `TestSelfPredecessorWithCheckpointMustRefuse`, `TestSuccessorCycleMustRefuse`, `TestSkippedEpochMustRefuse`, `TestEpochOneWithPredecessorMustRefuse`, `TestRev4SelfPredecessorMustRefuse` | N-chain-self admits exactly the self-predecessor link while terminating the walk; the isolated checkpoint-valid test fails by building successfully while skipped-epoch and cyclic links still refuse integrity_failure. The divergent reviewer probe still refuses through the checkpoint gate (documented subsumption, not a second kill claim). |
| Winner checkpoint reference | `TestRev4MissingCheckpointMustRefuse`, `TestWrongSessionCheckpointMustRefuse`, `TestWrongPredecessorCheckpointMustRefuse`, `TestValidSuccessorBuildsAndRevalidates`, `TestEpochOneWithSelfBoundCheckpointBuilds` | N-checkpoint-placeholder admits exactly the all-zero placeholder reference with an early return (a mere condition weakening would only shift the refusal reason onto the zero value); wrong-session and wrong-lease references still refuse observation_unavailable. |
| Checkpoint event-head closure | `TestRev7CheckpointHeadAuthority` (winner/missing_head, winner/later_lease_head, ancestor/missing_head, ancestor/later_lease_head, each through BuildPlan, fresh and old-plan Revalidate, AuthoritativeStatus, AuthoritativeList, with real-head controls), `TestValidSuccessorBuildsAndRevalidates`, `TestEpochOneWithSelfBoundCheckpointBuilds`, `TestRev5AncestorCheckpointAuthority`, `TestRev6CheckpointPersistenceMustMatchSession` (all with real heads) | N-checkpoint-heads admits exactly the all-zero missing head past the closure gate while later-lease and other unknown heads still refuse observation_unavailable; the later-lease branch is exercised by the later_lease_head negatives (a deleted branch would admit them). Missing and later-lease heads pin the observation_unavailable class in `rev7CheckHeadRefusalClass`, so the profile closure backstop surfaces a weakened heads gate as the wrong class rather than silent admission. |
| Profile closure resolution | `TestRev8ProfileSourceRefusals/dangling_predecessor` (merge naming a dangling digest, through all four entries with integrity_failure pinned) | Total predecessor resolution carries no narrowable predicate: any unknown non-record digest refuses, and the dedicated negative drives it. |
| Profile first-launch pair | `TestRev8CheckpointProfileAuthority` (missing_source on direct/task_board winner and ancestor paths), `TestRev8ProfileSourceRefusals/first_wrong_profile`, `TestRev10TaskBoardLaunchProfileAuthority/bad_first_source` | N-profile-first-source admits exactly the all-zero source where the creation pair wants null; every other unexpected or missing source still refuses integrity_failure. |
| Profile newest source | `TestRev8ProfileSourceRefusals/wrong_type` (non-change event), `/non_newest` (superseded change), `/cross_session` (foreign change), `/losing_branch` (preserved divergent-blob change outside the index), `/dangling_resume_source` | N-profile-newest keeps the first change as authority instead of the newest; citations of a superseded source then admit while other source classes still refuse. |
| Profile effective value | `TestRev8ProfileSourceRefusals/stale_value` (creation fallback past a change), `TestRev8ProfileChangedPositiveControl`, `TestRev8ProfileResumePositiveControl`, `TestRev8ProfileTwoGenerationControl` | N-profile-value admits exactly a yolo-valued pair past the effective-profile compare; every other stale value still refuses integrity_failure. |
| Profile change-target derivation | Same positive controls plus `TestRev8ProfileHistoricalClosureAdmits` | N-profile-change-direction still matches profile.changed and still reads `to` but derives from `from`; the P1/E1 positives fail. The discriminator token is preserved while the authority behavior changes. |
| First-launch value | `TestRev8ProfileSourceRefusals/first_wrong_profile`, `TestRev10TaskBoardLaunchProfileAuthority/bad_first_profile` | N-profile-first-value admits exactly a standard-valued sourceless pair where the creation pair wants yolo/null; the first-profile negatives fail by admission. |
| Fork new-session pair | `TestReview9ForkLocalProfile`, `TestRev10ForkLocalProfile` (bad_profile on direct/task_board winner and ancestor paths), `TestRev10ForkOldPlanSubstitution` | N-profile-fork-value admits exactly a standard-valued fork past the new-record compare; the fork negatives fail by admission. A non-null fork source cannot be chained (the canonical owner pins it at append), so presence stays covered through launches. |
| Resume referenced pair | `TestReview9ResumeReferencedCheckpoint`, `TestRev10ResumeReferencedCheckpoint` (bad_missing, bad_missing_stale, bad_wrong_session, bad_unknown_head refuse observation_unavailable), `TestRev10ResumeReferencedWithdrawal` | N-profile-resume-missing admits exactly a missing-reference resume carrying the creation pair; the missing negative fails by admission while other missing pairs still refuse. |
| Resume referenced newest | `TestRev10ResumeReferencedCheckpoint/bad_non_newest`, `/bad_losing`, `/bad_stale`, `/bad_creation_fallback`, `TestRev10ResumeOldPlanSubstitution` | N-profile-resume-newest keeps the first referenced change as authority instead of the newest; the non-newest negative fails by admission. |
| Referenced checkpoint creator | `TestReview10ReferencedCheckpointAdmission` (wrong_creator on direct/task_board winner and ancestor paths through all four entries), `TestRev11ReferencedCheckpointAdmissionRefusalClass/wrong_creator`, `TestRev11ReferencedCheckpointAdmissionOldPlan/wrong_creator` | N-referenced-creator admits exactly a hostB-created referenced checkpoint past the shared creator gate; the wrong_creator negatives fail by admission while other wrong-holder references still refuse observation_unavailable. |
| Referenced checkpoint variant | `TestReview10ReferencedCheckpointAdmission` (wrong_variant on direct/task_board winner and ancestor paths through all four entries), `TestRev11ReferencedCheckpointAdmissionRefusalClass/wrong_variant`, `TestRev11ReferencedCheckpointAdmissionOldPlan/wrong_variant` | N-referenced-variant admits exactly the idA wrong persistence variant on the referenced path; the wrong_variant negatives fail by admission while other variant mismatches still refuse observation_unavailable. |
| Referenced checkpoint lease | `TestReview10ReferencedCheckpointAdmission` (wrong_lease on direct/task_board winner and ancestor paths through all four entries), `TestRev11ReferencedCheckpointAdmissionRefusalClass/wrong_lease`, `TestRev11ReferencedCheckpointAdmissionOldPlan/wrong_lease` | The shared owner-tuple check keeps a referenced leaseCycleB fencing token unowned; the wrong_lease negatives fail by admission while other unowned tuples still refuse observation_unavailable. |
| Referenced checkpoint temporal authority | `TestReview11ReferencedTemporalAuthority` (same-lease, historical, divergent, future-owner and epoch-mismatch cases through BuildPlan, fresh and old-plan Revalidate, AuthoritativeStatus, AuthoritativeList), `TestRev12ReferencedCheckpointLaterLease` | N-referenced-future-owner removes the owner-at-or-before-consumer guard while preserving the searched identifiers; the four future-owner cases fail behaviorally. Same-lease and later-consumer controls remain admitted, and checkpoint heads must be in the consuming resume's predecessor closure. |
| Record-consumption type boundary | `TestRev11RecordConsumptionCensus`, `TestRev12RecordConsumptionCensusRejectsAlternatePaths`, `TestRev12AdmissionPrecisionCensus` | The package-wide go/types census admits raw checkpoint values only inside the loader and shared admission owners, requires profile derivation to receive `admittedCheckpoint`, and rejects raw map/parameter/new-file/method/closure/inferred-loader plants. The AST call/order census is precision evidence only; the behavioral temporal suites attack semantic admission. |
| Alias-normalized seal/token identity | `TestRev15AliasAwareRecordConsumptionCensus` | The package-wide go/types census normalizes `types.Alias` with `types.Unalias` before identity checks and recursive descent. Chained aliases and aliases inside containers and function signatures are rejected at the same production census entry; removing normalization on the nested value path is a killed narrowing mutant. |
| Historical closure fixation | `TestRev8ProfileHistoricalClosureAdmits` (post-head change never consulted), `TestRev8ProfileOldPlanSubstitution` (old plan refuses a substituted checkpoint whose valid heads cover a stale pair) | Admission behavior with paired refusal coverage above; no separate refusal predicate to narrow. |
| Union parked authority | `TestRevalidateUnionParkedCopyRefuses` (divergent and agreeing arms), `TestRev4ParkedAuthorityMustRefuse` | N-parked-union-refusal admits parked union copies with a nonempty record: the agreeing-record member validates over unknown authority (Revalidate and BuildPlan return nil, the strong kill) while the divergent-record member shifts to the record-agreement refusal. The divergent reviewer probe still refuses through record agreement (documented subsumption). |
| Summary capability vocabulary | `TestRev4UnknownCapabilityMustRefuse`, `TestAuthoritativeAdmitsFullCapabilityRegistry`, `TestAuthoritativeInputValidation/unknown_capability_name` | N-capability-name admits exactly the invented_capability name past the closed registry; every registry name stays admitted and every other invented name still refuses invalid_config. |
| Deterministic order | Order tests above | B-peer-order, B-session-order reverse list orders; B-tie-break reverses the copy tie break. |

The shipped mutation runner executes the behavioral suites in an
isolated source copy, checks exact replacement count, preserves raw
process exits and named failed tests, then restores and verifies the
copied bytes. It has before/after green controls, a harmless
comment control that must survive through the same instrument, a
not-applied control, and a compile-failure control. The latter two
prove no behavioral kill. The rev16 battery holds 66 unique N/B plants
(63 N, 3 B), all killed; the alias-normalization plant is a narrowing
mutant and the three B plants reverse deterministic output order. The
three classifier controls report separately and never join the kill
numerator. Measured runs and per-mutant tables are attached on the task,
never hard-coded as a green claim here. No
new source-text gate lacks its token-preserving mutant: the `@`,
`local`, `id:`, and alias-equality tokens each carry one, the
capability-vocabulary gate carries the invented-name admission plus
the full seven-name positive that preserves every searched-for
token through the behavioral suite, the profile discriminator
carries the from-for-to derivation mutant that preserves the
`profile.changed` match and the `to` read while changing which
member supplies authority. The record-consumption type boundary
carries executable alternate-path plants that preserve the loader
and checkpoint tokens while attempting to move raw values into a
new file, method, closure, or inferred map. The alias-normalization
plant preserves sealed-type identity while removing `types.Unalias`
only from the nested sealed-value recursion path; chained, container,
and function-signature aliases then fail the census. The temporal authority
gate carries the token-preserving N-referenced-future-owner
narrowing mutant; its harness runs the full temporal behavioral
suite and records a real kill, plus an applied harmless-control
survivor through the same instrument.

The read and revalidation entries perform no durable writes. The crash
test interrupts the real repository create after record installation,
inspects the parked projection, retries the byte-identical create
through its owner, and observes the healed read. Repeated lists and
revalidations return identical values. This establishes read
idempotency and composition with existing recovery, not a new crash
protocol.

## Stated bounds (not implementations)

- `lease_record_id` binds the canonical digest of the validated
  winning `urn:ax:schema:lease` Lease Record for the selected
  session, admitted from `Reader.LeaseRecords` through the
  session-persistence owner (`sessrepo.AttestLeaseRecord` →
  canonicaljson closed shape plus canonical self identity; greatest
  (epoch, lease_id) winner with a fully validated succession
  selected in `sessquery`: the predecessor ancestry chains by epoch
  plus one to an epoch-1 null-predecessor root, and every lease in
  that winning ancestry above epoch 1 references an admitted
  `urn:ax:schema:checkpoint` Checkpoint Record from
  `Reader.CheckpointRecords` through
  `sessrepo.AttestCheckpointRecord` for its session and predecessor
  lease (itself bound at epoch 1 for the root), each created by the
  owning lease holder, each carrying the persistence variant the
  referenced Session Record kind selects: direct sessions admit only
  provider-manifest checkpoints and task_board sessions only
  task-board-bundle checkpoints, and each admitting its Section
  2.4 profile authority over the same closure through
  `checkCheckpointProfileAuthority` (transitive predecessor
  closure from the winning source chain with the Session Record
  creation profile: the first launch carries the creation pair,
  later launches carry the newest authoritative profile.changed
  event at or before them with its target profile, resumes carry
  their referenced checkpoint's closure pair with the referenced
  record itself admitted through the shared owner (owning lease
  tuple in the winning ancestry, creator-holder, Session.kind
  variant, and head closure via the shared helpers), and forks
  carry the new record pair; contradictory pairs refuse
  integrity_failure while missing, wrong-session, wrong-lease,
  wrong-creator, wrong-variant, and unresolvable-head referenced
  authority refuses observation_unavailable; historical heads stay
  admissible because derivation never consults the current tail).
  The winning triple binds from
  that same record; envelope triples never substitute for it. A
  session with no admitted winning record refuses
  selector_observation_unavailable; a complete read proving no lease
  refuses selector_bootstrap_incomplete instead of binding an empty
  triple. The provhost no-attestation-outside-the-leaf bound now
  allowlists five session-leaf sites (the two entry decodes, the
  load re-verification, and both record attestations).
- `authority_heads` binds the sorted unique validated lease, event,
  and tombstone heads used for the decision: the winning Lease
  Record digest plus the tail event digest of every allowed source
  holding a non-parked copy of the pinned session. The complete
  authority union (same record, current tombstones, greatest winning
  lease across the local and every allowlisted source) is validated
  at build and at every revalidation; a parked copy of the pinned
  session in any required source fails the union closed with
  selector_observation_unavailable instead of contributing a head;
  unrelated index changes stay conservative stale.
- `source_host_id` is always a bound UUIDv7: a local selection
  without a known local host refuses invalid_config at build time.
- Wire exit binding (CLI Result 5 with Structured Error 1.4.0) belongs
  to its owning leaf; this library keeps refusal classes distinct and
  stable under their normative spellings without registering codes.
- Record-only chains keep the accepted internal projection with empty
  owner/lease facts inside the projector. The list/status summary
  entries refuse them, while explicit local inspection retains raw
  access to the recovery diagnostics. The 14.7.4 recovery flow itself
  belongs to the bootstrap leaf with the accepted projector behavior
  reconciled in its scoped review; this leaf invents neither a state
  nor an error for it.
- Effect authorization (fencing, commit, retry, recovery, remote
  dispatch/admission, transport-resume) and the composed
  invocation-to-plan bindings for attach and logs belong to the CLI
  and lifecycle owners, which invoke `Revalidate` at every boundary in
  `BoundariesFor`. Plans built here authorize read projection only.
- Reads across multiple `Projector.Project` calls are not an atomic
  repository/mesh snapshot; concurrent writers, fresh peer
  acquisition, cross-peer lease union, and differing replicated views
  are not measured. Transports, peer authentication, freshness, mesh
  union, host metadata, and observation contracts remain upstream
  boundaries: configuration, learned indexes, host metadata, and
  observations arrive as trusted caller inputs, and the allowlist proof
  covers filtering, never attestation. The repository/projector own
  record/event admission and validation. No new external provider,
  filesystem durability, terminal, RPC, or CLI capability is attested.

## Rev16 sealed-capability and alias-census update

The profile derivation boundary now receives a private `admittedCheckpoint`
sealed capability. Only the exact free `admitCheckpoint` function can create
the seal; each of the nine profile derivation entries verifies it before
reading authority fields. `TestRev14CapabilitySeal` drives zero and
field-assembled forged values through all nine entries and requires
`selector_observation_unavailable`.

The package-wide record-consumption census now resolves exact `*types.Func`
objects and fails closed on type-check errors. Its alternate-path plants cover
interface assertions, package closures, unresolved callees, methods merely
named `admitCheckpoint`, field assembly, embedding, raw copies, capability
maps, non-composite seal/token construction, generic construction, interface
copies, `reflect.TypeFor`, and `unsafe.Pointer`. The census also requires
exact callback-parameter object provenance, rejecting package variables and
struct fields even when their function type matches the approved callback.
The `N-census-raw-owner-object` narrowing plant changes the raw-owner gate to
admit a method by name; the same census test then fails on
`method_raw_only.go`, so this is behavioral evidence for the object-identity
gate rather than a token-only source scan. `N-census-unknown-callback`
narrowing admits one locally-bound callback by name; the exact callback
subtest then fails through the same census instrument. The new
`TestRev15AliasAwareRecordConsumptionCensus` plant covers direct, chained,
container, and function-signature aliases. `types.Unalias` is applied before
each identity or structural descent, and the nested value-normalization
narrowing fails the container plant when removed.

The rev16 mutation battery contains 66 applied N/B plants (63 narrowing and
3 ordering), all killed; the applied harmless control survives separately,
while the not-applied and compile-failure controls remain classified as
`NOT_APPLIED` and `COMPILE_OR_HARNESS_FAILURE`. The copied sources are
restored after the run and the candidate contains no Python cache artifact.
The raw run, source manifest, and per-mutant logs are attached to the owning
task resource.

## TASK-260830-147hsj lease-union derivation supplement

Authority: pinned `internal/specdoc/SPEC.v0.7.0.md`, §11.4 rules 5–6 and §5.3
(lines 1987–2061, including line 2047). `Reader.LeaseHeadsForSession` is the
post-union projection entry consumed by
`internal/merkleinventory.Index.RebuildProjection`.

| Rule | Production call site | Evidence | Narrowing |
| --- | --- | --- | --- |
| Derive every validated lease tuple; do not choose before union | `Reader.LeaseHeadsForSession` → `validatedLeasesBySession` / `parseLeaseRecord` | `TestLeaseHeadsForSessionReturnsEveryTupleAcrossTimestampPerturbation` asserts both literal sorted tuples for both opposing `created_at` assignments and reversed `LeaseRecords` input order. | `union-lease-tuple-omission` drops one competing UUIDv4 tuple; the named test alone fails. |
| Timestamps have no winner authority | Same entry; returned `sessstate.LeaseHead` contains only epoch and lease UUID | The same test moves each Lease Record's `created_at` to opposite extremes; the tuple result is byte-for-byte equal. Projection permutation coverage is in `internal/merkleinventory/durable_test.go`. | `union-lease-created-at-winner` promotes the later diagnostic timestamp into an epoch; the named test alone fails. |
| Generated tuple cardinality and order | `Reader.LeaseHeadsForSession` → `validatedLeasesBySession` / `parseLeaseRecord` | `TestLeaseHeadsForSessionCoversGeneratedCardinalityRange` generates 0–32 distinct UUIDv4 tuples, independently sorts the expected list, and compares both input order/timestamp assignments. | `union-lease-generated-cardinality-omission` omits a generated tuple at size 17; that subtest alone fails. The loop/map/sort structure has no size-specific branch beyond this generated range. |
| One lease UUID cannot carry two conflicting object identities | Same entry, keyed by `validatedLease.LeaseID` | `TestLeaseHeadsForSessionRefusesConflictingBytesForOneLeaseID` uses two individually validated records with one UUID and distinct canonical bytes and asserts literal `integrity_failure`. | `union-lease-conflicting-same-id` admits the exact test UUID's conflicting bytes; that test alone fails. |

The helper parses every record through the existing `parseLeaseRecord` owner,
rejects conflicting bytes for one lease UUID with literal integrity refusal,
deduplicates identical lease records, and sorts output by the existing
`sessstate.Compare` tuple ordering. It does not select a winner; the existing
pure reducer resolves the full supplied tuple set after union. Its existing
`sessckpt` and `termbind` test importers are unchanged; the new production
importer is `merkleinventory`, listed in
`internal/merkleinventory/IMPORTER-OUTCOMES.md`.
