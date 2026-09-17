# TASK-260830-21gygk rev15 relation census

This audit closes the recurring canonical-identity-versus-semantic-authority
defect family over the complete winning ancestry. A canonical digest is never
treated as sufficient admission. All listed positives and negatives drive the
shared production API rather than calling validators directly.

| Relation | Required semantic guard | Production call site | Evidence |
| --- | --- | --- | --- |
| Winner → checkpoint | A non-root winner names an admitted checkpoint for the session and owning lease tuple; a root may omit it only before a provider boundary. | `Reader.winningLeaseFor` → `checkWinnerCheckpoint` → `admitCheckpoint` | `TestValidSuccessorBuildsAndRevalidates`, `TestRev4MissingCheckpointMustRefuse`, `N-checkpoint-placeholder`, `N-checkpoint-holder` |
| Every necessary ancestor → checkpoint | Every necessary ancestry checkpoint is admitted, including the positive three-generation chain; omission is unavailable authority, not an ignorable missing copy. | `Reader.winningLeaseFor` → `checkAncestorCheckpoints` → `checkCheckpointBinding` → `admitCheckpoint` | `TestRev5AncestorCheckpointAuthority/complete` (epoch 1/2/3), `/missing_ancestor_checkpoint`, `TestRev5AncestorCheckpointBinding`, `N-ancestor-checkpoint`, `N-ancestor-holder` |
| Checkpoint → owning lease and holder | Session, epoch, lease ID, and `created_by_host_id` must match the owning Lease Record and its `holder_host_id`; canonical checkpoint identity alone is insufficient. | `admitCheckpoint` | `TestRev5CheckpointCreatorMustBeHolder`, `TestRev5AncestorCheckpointBinding/wrong_creator`, `TestRev11ReferencedCheckpointAdmissionRefusalClass/wrong_creator`, `N-checkpoint-holder`, `N-ancestor-holder`, `N-referenced-creator` |
| Checkpoint → Session.kind | Direct sessions admit provider-manifest persistence; task-board sessions admit task-board-bundle persistence only. | `admitCheckpoint` → `checkCheckpointPersistence` | `TestRev6CheckpointPersistenceMustMatchSession`, `TestRev6TaskBoardPersistenceControl`, `TestRev6PersistenceMismatchIsObservationUnavailable`, `N-checkpoint-persistence`, `N-referenced-variant` |
| Checkpoint → event-head closure | Every head resolves to a chained event in the same session at or before its bound lease in the winning source chain. Historical heads are admissible; unknown, losing, later-lease, and cross-session heads refuse. | `admitCheckpoint` → `checkCheckpointEventHeads` | `TestRev7CheckpointHeadAuthority`, `TestRev7IndependentPersistence`, `N-checkpoint-heads` |
| Checkpoint → profile closure | Profile derivation reads only the admitted closure: first launch, newest change, resume reference, fork pair, and historical pair are bound independently. | `checkCheckpointProfileAuthority` → nine profile helpers | `TestRev8ProfileChangedPositiveControl`, `TestRev8ProfileResumePositiveControl`, `TestRev8ProfileTwoGenerationControl`, `TestRev8ProfileSourceRefusals`, `N-profile-*`, `TestRev14CapabilitySeal` |
| Resume → referenced checkpoint | The referenced checkpoint belongs to the consuming lease or an earlier lease in the same validated succession, and every referenced head is in the resume predecessor closure. | `buildCheckpointAdmission` → `checkReferencedCheckpointBinding` → `admitCheckpoint` → `checkReferencedCheckpointTemporal` | `TestRev12ReferencedCheckpointLaterLease`, `TestReview11ReferencedTemporalAuthority` including `good_prechange` and `good_divergent`, `N-referenced-temporal`, `N-profile-resume-missing`, `N-profile-resume-newest` |
| Plan → current authority union | Build and revalidation inspect all allowed sources for the pinned UUID. Names are never re-resolved; lagging copies remain current; greater lease/tombstone facts, divergent records, parked required copies, and unreadable required sources fail closed. | `Reader.BuildPlan`, `Reader.Revalidate` → `validateAuthorityUnion` | `TestBuildPlanValidatesUnionAtBuild`, `TestRevalidateUnionAgreeingReplicaStaysCurrent`, `TestBuildWithLaggingLeaseCopy`, `TestRevalidateUnionParkedCopyRefuses`, `TestRevalidateUnionUnreadablePeerRefuses`, `N-union-*`, `N-parked-union-refusal` |

## Boundary audit

`admittedCheckpoint` is a private sealed capability, not an exported
record-shaped DTO. Only the exact free `admitCheckpoint` function constructs
the seal. Each profile entry starts from `authority()` and receives the
capability. The rev14 census resolves exact `go/types` objects and rejects
raw values, field assembly, embedding, interface/closure paths, capability
maps, methods with the same name, non-composite/generic construction,
interface copies, reflection/unsafe construction, and unapproved callback
objects. The applied `N-census-raw-owner-object` and
`N-census-unknown-callback` narrowing mutants preserve the searched tokens and
are killed by the same census instrument.

The temporal controls are not aliases: `good_prechange` validates a historical
checkpoint/profile pair, while `good_divergent` proves a preserved losing
branch does not displace the authoritative winning pair. Negative future-owner
and epoch-mismatch cases remain `selector_observation_unavailable` through
BuildPlan, old-plan Revalidate, AuthoritativeStatus, and AuthoritativeList.
