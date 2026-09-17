# TASK-260830-21gygk rev15 conformance matrix

Authority is v0.6.0 commit `0cbdf100dbf84df50c64f792b1f940e3a67859a6`;
historical scope retains v0.5.0 commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`.
This matrix records the shared selector/plan/summary boundary and does not
claim CLI or lifecycle ownership.

| Normative requirement | Production owner and input | Positive evidence | Negative/refusal evidence |
| --- | --- | --- | --- |
| §2.3 / §14.7 literal grammar and tier precedence | `Reader.Resolve` → `parseSelector`, `resolveBare`, `resolveExplicit`; raw selector and validated source config | `TestParseSelectorLiteralGrammar`, `TestResolveLocalNameAndUUID`, `TestResolveBareIdentityUnion` | `N-first-at-split`, `N-local-prefix`, `N-id-case`, `N-id-through-names`, `N-explicit-fallback` |
| §14.7.1 exact source mapping | `locateSource`; exact local/peer/host mapping, allowlist, one selected source | `TestResolveQualifiedSourcesSelectOneIndex`, `TestResolveQualifiedNameBeforeUUIDInSource` | `TestResolveExplicitSourceRefusals`, `TestResolveExplicitSourceReadFailures`, `N-unknown-alias-local`, `N-disallowed-admit`, `N-unlisted-peer`, `N-read-failure-fallback` |
| §14.7.1 collision/tombstone semantics | `matchName`, `selectIdentity`, `checkRecordAgreement`; authoritative rows only | `TestResolveExactNamesAndASCIICollisions`, `TestResolveQualifiedAmbiguityAndExclusions`, `TestResolveExcludesTombstonedButListsIt`, `TestResolveBareIdentityUnion` | `N-collision-pair`, `N-case-variant`, `N-parked-id`, `N-tombstoned-id`, `N-identity-diverge`, `N-agreement-name` |
| §14.7.2 immutable SelectionPlan | `Reader.BuildPlan` → `bindPlan`; selected source, validated record, winning Lease Record and authority heads | `TestBuildPlanBindsCurrentFacts`, `TestBuildPlanPeerSourceFacts`, `TestPlanDigestAndMarshalRoundTrip`, `TestReviewerPlanRequiresLeaseRecord`, `TestRev3RealLeaseRecordIdentity` | `TestBuildPlanRecordOnlySessionRefusesBootstrap`, `TestBuildPlanLocalSourceRequiresKnownHost`, `N-plan-bootstrap`, `N-plan-source-host`, `N-lease-missing` |
| §14.7.2 current-fact revalidation | `Reader.Revalidate` → fixed-fact comparison and `validateAuthorityUnion`; pinned UUID, no name re-resolution | `TestRevalidateDetectsEachFactChange`, `TestRevalidateUnionAgreeingReplicaStaysCurrent`, `TestBuildWithLaggingLeaseCopy`, `TestRevalidateUnionNameDriftStaysCurrent` | `TestReviewerOtherSourceDivergenceMustRefuse`, `TestRevalidateUnionParkedCopyRefuses`, `TestRevalidateUnionUnreadablePeerRefuses`, `N-rev-*`, `N-union-*`, `N-parked-union-refusal` |
| §5.3 winning succession and full ancestry | `Reader.winningLeaseFor` → `checkLeaseChain`, `checkWinnerCheckpoint`, `checkAncestorCheckpoints`; all winning/necessary Lease and Checkpoint Records | `TestValidSuccessorBuildsAndRevalidates`, `TestRev5AncestorCheckpointAuthority/complete` (three generations), `TestRev12ReferencedCheckpointLaterLease` | `TestSuccessorCycleMustRefuse`, `TestSkippedEpochMustRefuse`, `TestSelfPredecessorWithCheckpointMustRefuse`, `TestRev5AncestorCheckpointAuthority/missing_ancestor_checkpoint`, `TestRev5AncestorCheckpointBinding/wrong_creator`, `N-chain-self`, `N-ancestor-checkpoint`, `N-checkpoint-holder` |
| §5.4 checkpoint semantic admission | `admitCheckpoint`; raw validated record plus owning lease, holder, Session.kind, source chain, optional consuming resume | `TestRev5CheckpointCreatorMustBeHolder/holder`, `TestRev6TaskBoardPersistenceControl`, `TestRev7CheckpointHeadAuthority` real-head controls, `TestReview11ReferencedTemporalAuthority/good_prechange`, `/good_divergent` | `TestRev5CheckpointCreatorMustBeHolder/non_holder`, `TestRev5AncestorCheckpointBinding`, `TestRev6PersistenceMismatchIsObservationUnavailable`, `TestRev7CheckpointHeadAuthority` missing/later heads, `N-checkpoint-holder`, `N-ancestor-holder`, `N-checkpoint-persistence`, `N-checkpoint-heads`, `N-referenced-creator`, `N-referenced-variant`, `N-referenced-temporal` |
| §14.7.2 profile authority and sealed capability | `checkCheckpointProfileAuthority` and nine profile helpers; private `admittedCheckpoint`; exact-object census | `TestRev8ProfileChangedPositiveControl`, `TestRev8ProfileResumePositiveControl`, `TestRev8ProfileTwoGenerationControl`, `TestRev8ProfileHistoricalClosureAdmits`, `TestRev14CapabilitySeal`, `TestRev14RecordConsumptionCensusRejectsAlternatePaths` | `TestRev8ProfileSourceRefusals`, `TestRev10ResumeReferencedCheckpoint`, `TestRev14CallbackProvenanceRejectsUnknownFunctions`, `N-profile-*`, `N-census-raw-owner-object`, `N-census-unknown-callback` |
| §5.7 / §14.7.3 summary facts and closed capabilities | `Reader.List`, `Reader.Status`, `Reader.AuthoritativeStatus`, `Reader.AuthoritativeList`; projection plus `checkSummaryRepresentable`, `authorize`, `checkCapabilities` | `TestListStatusDerivedFactsAndStableOrder`, `TestAuthoritativeStatusHealthy`, `TestAuthoritativeAdmitsFullCapabilityRegistry`, `TestAuthoritativeOptionalObservations` | `TestSummaryBootstrapRefusals`, `TestSummaryMixedListRefusesWhole`, `TestAuthoritativeMissingHostMetadataRefuses`, `TestAuthoritativeMissingProcessRefusesStatus`, `TestRev4UnknownCapabilityMustRefuse`, `N-summary-bootstrap`, `N-observation-*`, `N-host-bound64`, `N-capability-name` |

## Ownership and capability bounds

- Shared API rows: **8 of 8 driven**.
- CLI/lifecycle rows: **0 of 8**, explicitly bounded to owning CLI/lifecycle
  leaves. No `ax` executable, CLI Result renderer, transport, fencing, or
  mutation caller is introduced here.
- Durable mutation: none in `sessquery` read/plan/summary entries; crash and
  idempotency are not applicable to this operation. Existing repository
  recovery tests remain in the full suite.
- Capability claims: no new provider/platform capability. Authoritative
  summaries admit exactly the seven provhost-owned names and refuse invented
  names.
