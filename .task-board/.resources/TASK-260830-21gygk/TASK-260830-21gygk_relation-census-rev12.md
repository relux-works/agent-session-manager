# TASK-260830-21gygk: checkpoint/profile relation census rev12

Candidate source: `e119aaeede327580a470238fdb1ccf340a785b39` plus the
uncommitted Story candidate. This is a bounded implementation census for
the v0.6.0 Sections 5.3, 5.4, 2.4, and 14.7.2 relationships. It names the
production gate and the tests that drive it; it does not claim CLI or
lifecycle ownership.

| Relation | Production gate / source of truth | Positive control | Negative / refusal control |
| --- | --- | --- | --- |
| K4-1: checkpoint owner lease is at or before consuming resume | `checkReferencedCheckpointTemporal` compares the referenced owner lease position with the consuming resume lease position in the winning chain | `TestReview11ReferencedTemporalAuthority` same-lease and historical cases; `TestRev12ReferencedCheckpointLaterLease` earlier checkpoint consumed by later lease | `TestReview11ReferencedTemporalAuthority` `future_owner` cases through BuildPlan, fresh/old Revalidate, AuthoritativeStatus, AuthoritativeList; refuses `selector_observation_unavailable` |
| K4-2: checkpoint event heads are authoritative at/before owner and consuming closure | `checkCheckpointEventHeads` validates each head against the owning lease chain; `checkReferencedCheckpointTemporal` additionally checks resume predecessor closure | `TestRev7CheckpointHeadAuthority` real historical heads and valid head controls; temporal same-lease/historical controls | missing/later-lease/unknown head arms in `TestRev7CheckpointHeadAuthority`, plus divergent and epoch-mismatch arms in `TestReview11ReferencedTemporalAuthority` |
| K4-3: consuming resume can only use its own predecessor closure | `checkReferencedCheckpointTemporal` reads the selected repository events, follows resume predecessors, and requires each referenced head in that closure; the repository-ID genesis boundary is explicit | `TestReview11ReferencedTemporalAuthority` valid and `good_prechange` controls | `good_divergent` and `epoch_mismatch` paths refuse rather than accepting a head outside the consuming closure |
| K4-4: necessary checkpoint binds session and owning lease | `checkReferencedCheckpointBinding` locates the owning lease in the complete winning ancestry and calls `admitCheckpoint` | `TestRev5AncestorCheckpointAuthority`, `TestRev5AncestorCheckpointBinding`, `TestRev6TaskBoardPersistenceControl` | wrong-session and wrong-lease references refuse through all four shared entries and old-plan Revalidate |
| K4-5: checkpoint creator is the owning lease holder | `admitCheckpoint` checks `CreatedByHostID == owner.HolderHostID` | direct/task_board valid controls in `TestReview10ReferencedCheckpointAdmission` | `TestRev5CheckpointCreatorMustBeHolder`, `wrong_creator` arms in `TestReview10ReferencedCheckpointAdmission`, `TestRev11ReferencedCheckpointAdmissionRefusalClass`, and old-plan variant |
| K4-6: persistence variant matches Session.kind | `admitCheckpoint` calls `checkCheckpointPersistence` for the referenced session kind | direct and task_board valid controls; `TestRev6TaskBoardPersistenceControl` | wrong-variant arms in `TestReview10ReferencedCheckpointAdmission`, `TestRev11ReferencedCheckpointAdmissionRefusalClass`, and old-plan variant |
| K4-7: profile derivation consumes admitted capability only | `checkCheckpointProfileAuthority` and profile helpers accept `admittedCheckpoint`, not `validatedCheckpoint` | `TestRev8ProfileHistoricalClosureAdmits`, `TestRev8ProfileTwoGenerationControl`, `TestReview10ReferencedCheckpointAdmission` | `TestRev12RecordConsumptionCensusRejectsAlternatePaths` rejects new-file, method, closure, and inferred-loader raw paths; `TestRev12AdmissionPrecisionCensus` checks call/order edges |
| K4-8: creator/variant/temporal guards are shared by winner and referenced paths | winner and binding paths both route through `admitCheckpoint`; referenced path is `buildCheckpointAdmission` -> `checkReferencedCheckpointBinding` -> `admitCheckpoint` | `TestRev5AncestorCheckpointAuthority`, `TestRev6CheckpointPersistenceMustMatchSession`, `TestRev7CheckpointHeadAuthority`, temporal valid controls | `TestRev12AdmissionPrecisionCensus` verifies the shared call edges; rev12 narrowing mutants behaviorally kill removal of temporal and raw-boundary gates |
| Lease ancestry itself is complete before checkpoint admission | `checkLeaseChain` validates epoch succession, predecessor links, root shape, and cycle absence before checkpoint checks | `TestValidSuccessorBuildsAndRevalidates`, `TestRev5AncestorCheckpointAuthority`, `TestRev6TaskBoardPersistenceControl` | `TestSuccessorCycleMustRefuse`, `TestSkippedEpochMustRefuse`, `TestDanglingPredecessorMustRefuse`, `TestRev4SelfPredecessorMustRefuse` |
| Current-plan union is revalidated without name retargeting | `Reader.Revalidate` -> `validateAuthorityUnion` and `collectAuthorityHeads`, keyed by pinned UUID | `TestRevalidateUnionAgreeingReplicaStaysCurrent`, `TestBuildWithLaggingLeaseCopy`, `TestRevalidateUnionNameDriftStaysCurrent` | `TestReviewerOtherSourceDivergenceMustRefuse`, `TestRevalidateUnionLeaseDivergenceRefusesStale`, `TestRevalidateUnionParkedCopyRefuses`, `TestReview11ReferencedTemporalAuthority` old-plan cases |

## Package boundary evidence

The production parsed type `validatedCheckpoint` is not an authority
capability. The only successful `admittedCheckpoint{...}` construction is in
`admitCheckpoint`; profile derivation signatures take `admittedCheckpoint`.
`TestRev11RecordConsumptionCensus` uses package-wide `go/types` analysis and
`TestRev12RecordConsumptionCensusRejectsAlternatePaths` writes five executable
alternate-path plants against the same census. The tests reject raw values in
a new file, method receiver, closure factory, inferred loader, and bypass
helper. The AST checks in `TestRev12AdmissionPrecisionCensus` supplement this
with the required call/order relationships, but are not used as a substitute
for the type-level boundary.

## Temporal interpretation

The owner-at-or-before rule is about the checkpoint's owning lease, not about
the consumer's lease. Therefore a later lease may consume a checkpoint owned
by an earlier lease when the referenced event heads are in the consuming
resume's predecessor closure. A checkpoint owned by a future lease is refused
even when its canonical identity and session fields otherwise match. This
distinguishes forbidden retargeting from the required semantic refusal.

## AC and caller boundary

The shared API drives 8 of 8 acceptance rows. CLI/lifecycle publication,
wire exit binding, durable recovery/effects, transport, and external
observation ownership are caller-owned and are not claimed here. No read or
revalidation path writes durable state.
