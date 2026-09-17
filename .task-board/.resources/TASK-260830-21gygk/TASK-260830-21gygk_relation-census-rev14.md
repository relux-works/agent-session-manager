# TASK-260830-21gygk rev14 relation census

This is the relation-level audit for the recurring canonical-identity-versus-semantic-authority defect family. The implementation is evaluated over the complete winning ancestry, not just the canonical digest shape.

| Relation | Required semantic guard | Production call site | Evidence |
| --- | --- | --- | --- |
| Winner → checkpoint | A non-root winner must name an admitted checkpoint for the session and the owning lease tuple; a root may omit the checkpoint only before a provider boundary. | `Reader.winningLeaseFor` → `checkWinnerCheckpoint` → `admitCheckpoint` | `TestValidSuccessorBuildsAndRevalidates`, `TestRev4MissingCheckpointMustRefuse`, `N-checkpoint-placeholder`, `N-checkpoint-holder` |
| Necessary ancestor → checkpoint | Every necessary ancestry checkpoint is admitted, including multi-generation chains; skipping one is unavailable authority, not absence. | `Reader.winningLeaseFor` → `checkAncestorCheckpoints` → `checkCheckpointBinding` → `admitCheckpoint` | `TestRev5AncestorCheckpointAuthority/complete`, `/missing_ancestor_checkpoint`, `TestRev5AncestorCheckpointBinding`, `N-ancestor-checkpoint`, `N-ancestor-holder` |
| Checkpoint → owning lease | Checkpoint session, epoch, lease ID, and creator must match the owning lease and its holder. Canonical checkpoint identity alone is insufficient. | `admitCheckpoint` | `TestRev5CheckpointCreatorMustBeHolder`, `TestRev5AncestorCheckpointBinding`, `TestRev11ReferencedCheckpointAdmissionRefusalClass`, `N-checkpoint-holder`, `N-ancestor-holder`, `N-referenced-creator` |
| Checkpoint → Session.kind | Direct sessions admit provider-manifest persistence only; task-board sessions admit task-board-bundle persistence only. | `admitCheckpoint` → `checkCheckpointPersistence` | `TestRev6CheckpointPersistenceMustMatchSession`, `TestRev6TaskBoardPersistenceControl`, `TestRev6PersistenceMismatchIsObservationUnavailable`, `N-checkpoint-persistence`, `N-referenced-variant` |
| Checkpoint → event heads | Every head must resolve to a chained event in the same session and at or before the owning lease; historical heads are valid, later/lost/unknown heads are unavailable. | `admitCheckpoint` → `checkCheckpointEventHeads` | `TestRev7CheckpointHeadAuthority`, `TestRev7IndependentPersistence`, `N-checkpoint-heads` |
| Checkpoint → profile closure | Profile derivation reads the checkpoint's admitted closure only; first launch, newest change, resume reference, and fork pairs are independently bound. | `checkCheckpointProfileAuthority` → nine profile helpers | `TestRev8ProfileChangedPositiveControl`, `TestRev8ProfileResumePositiveControl`, `TestRev8ProfileTwoGenerationControl`, `TestRev8ProfileSourceRefusals`, `N-profile-change-direction`, `N-profile-first-value` |
| Resume → referenced checkpoint | The referenced checkpoint's owner is the consuming lease or an earlier lease in the same admitted ancestry; every referenced head is an ancestor of the resume event. | `buildCheckpointAdmission` → `checkReferencedCheckpointBinding` → `admitCheckpoint` → `checkReferencedCheckpointTemporal` | `TestRev12ReferencedCheckpointLaterLease`, `TestReview11ReferencedTemporalAuthority`, `N-referenced-temporal`, `N-profile-resume-missing`, `N-profile-resume-newest` |
| Plan → current union | Build and revalidation validate all allowed sources for the pinned UUID; names are not re-resolved, lagging copies remain current, greater lease/tombstone facts become stale, and parked/unreadable required sources fail closed. | `Reader.BuildPlan`, `Reader.Revalidate` → `validateAuthorityUnion` | `TestBuildPlanValidatesUnionAtBuild`, `TestRevalidateUnionAgreeingReplicaStaysCurrent`, `TestBuildWithLaggingLeaseCopy`, `TestRevalidateUnionParkedCopyRefuses`, `N-union-record`, `N-union-lease`, `N-union-tombstone`, `N-parked-union-refusal` |

## Boundary audit

`admittedCheckpoint` is not an exported record-shaped DTO. Its private seal
token is constructed only by the exact free `admitCheckpoint` function, and
every profile entry starts with `authority()`. The rev14 census resolves exact
`*types.Func` identity and rejects raw values, field assembly, embedding,
interface/closure paths, capability maps, methods with the same name, and
unresolved callees. The applied `N-census-raw-owner-object` mutant weakens
that exact identity to a method-name test and is killed by the raw-only method
plant through the same census test.
