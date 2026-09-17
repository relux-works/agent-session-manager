# TASK-260830-21gygk relation census — rev16

This is the implementation-side relation inventory for v0.6.0 Sections 5.3,
5.4, 14.7.1, and 14.7.2. It records the relationships that must be admitted
before a selector result, summary, or immutable plan can consume authority.
The source checkpoint is `e119aaeede327580a470238fdb1ccf340a785b39`; the
expected base is `8626fb361c1d6b62d5f95c7be4cb33c85d4aadf3`.

| Relation | Shared production owner | Positive control | Refusal control and proof |
| --- | --- | --- | --- |
| Winning lease identity | `Reader.winningLeaseFor` and `checkLeaseChain` in `internal/sessquery/lease.go` | `TestBuildPlanBindsCurrentFacts`, `TestValidSuccessorBuildsAndRevalidates`, `TestBuildWithLaggingLeaseCopy` | `TestReviewerPlanRequiresLeaseRecord`, `TestSuccessorCycleMustRefuse`, `TestSkippedEpochMustRefuse`, `TestEpochOneWithPredecessorMustRefuse`, `TestSelfPredecessorWithCheckpointMustRefuse`; `N-lease-missing` and `N-chain-self` are killed. |
| Necessary ancestry completeness | `checkAncestorCheckpoints` / `checkCheckpointBinding` | `TestRev5AncestorCheckpointAuthority` complete multi-generation control | Missing ancestor checkpoint, wrong session, wrong lease, and wrong creator are driven through BuildPlan, old-plan Revalidate, AuthoritativeStatus, and AuthoritativeList. `N-ancestor-checkpoint` and `N-ancestor-holder` are killed. |
| Checkpoint session/lease tuple | `checkCheckpointBinding` and `checkReferencedCheckpointBinding` | `TestValidSuccessorBuildsAndRevalidates`, `TestReview10ReferencedCheckpointAdmission` valid direct/task_board and winner/ancestor controls | Wrong-session, wrong-lease, and unowned fencing tuple refuse `selector_observation_unavailable` through all four shared entries; `N-referenced-creator`, `N-referenced-variant`, and the retained referenced-lease narrowing are killed. |
| Checkpoint creator equals owning holder | `checkCheckpointCreator` / shared referenced-checkpoint binding | `TestRev5CheckpointCreatorMustBeHolder` valid winner and `TestRev5AncestorCheckpointBinding` valid ancestry | Non-holder creator controls fail through BuildPlan, old-plan Revalidate, and both authoritative summaries. `N-checkpoint-holder` and `N-ancestor-holder` are killed by named negative tests. |
| Session.kind selects persistence variant | `checkCheckpointPersistence` | `TestRev6TaskBoardPersistenceControl`, direct and task_board valid variants | Swapped provider/task_board variants refuse on winner and necessary ancestor paths, including old plans and both summaries. `N-checkpoint-persistence` and `N-referenced-variant` are killed. |
| Checkpoint heads belong to the bound session and lease era | `checkCheckpointEventHeads` | `TestRev7CheckpointHeadAuthority` real-head controls; `TestEpochOneWithSelfBoundCheckpointBuilds` | Missing, cross-session, losing-lease, and later-lease heads refuse unavailable authority. `N-checkpoint-heads` is a token-preserving narrowing kill; the class pin prevents a later profile backstop from hiding the weakened head gate. |
| Profile pair follows the same checkpoint closure | `checkCheckpointProfileAuthority`, `profileClosure`, and `checkClosureProfilePairs` | `TestRev8ProfileTwoGenerationControl`, `TestRev8ProfileHistoricalClosureAdmits`, `TestRev10TaskBoardLaunchProfileAuthority` | Wrong first/later/fork/resume pairs, stale sources, losing branches, dangling predecessors, and non-newest referenced changes refuse integrity failure. `N-profile-*` narrowing rows are killed. |
| Resume reference is admitted before consumption | `referencedCheckpointProfile` → `checkpointAdmission` → `checkReferencedCheckpointBinding` | `TestReview9ResumeReferencedCheckpoint`, `TestRev10ResumeReferencedCheckpoint`, `TestRev10ResumeReferencedWithdrawal` | Missing, wrong-session, wrong-head, wrong-creator, wrong-variant, wrong-lease, and future-owned references refuse unavailable authority. The exact callback parameter object is required by the package census; `N-census-unknown-callback` is killed. |
| Current authority union remains complete | `validateAuthorityUnion`, `collectAuthorityHeads`, `Reader.Revalidate` | `TestBuildPlanValidatesUnionAtBuild`, `TestRevalidateUnionAgreeingReplicaStaysCurrent`, `TestBuildWithLaggingLeaseCopy` | Divergent records, greater lease winners, fresh tombstones, unreadable required peers, and parked required copies refuse or stale as specified; lagging agreeing copies do not. `N-union-record`, `N-union-lease`, `N-union-tombstone`, and `N-parked-union-refusal` are killed. |
| Plan remains pinned to identity, not a fresh name resolution | `Reader.Revalidate` and `ParsePlan` | `TestRevalidateUnionNameDriftStaysCurrent`, `TestPlanDigestAndMarshalRoundTrip` | A same-name gain for another UUID does not retarget an old plan; changed bound facts refuse stale. `N-rev-*` rows cover each bound fact, and `N-id-through-names` covers the UUID/name split. |

## Alias census relation

`TestRev15AliasAwareRecordConsumptionCensus` builds a temporary package copy
containing these source shapes:

- an alias of `checkpointSeal` used by `new`;
- an alias of `checkpointSealToken` used by `new`;
- a two-step alias chain;
- aliases nested in a map/struct/slice container; and
- an alias nested in a function signature.

The production census entry is `rev12CheckRecordConsumptionSource` in
`internal/sessquery/rev11_regression_test.go`. Its seal/token identity path
uses `types.Unalias` before each recursive descent. The applied narrowing
mutant changes only `rev14UnaliasSealedValueType` to return its input; the
container plant then fails in the same `go test` subprocess. This is behavioral
evidence for alias normalization, not a source-token scan.

The prior exact-source CR15 evidence remains the authority for the closed
ancestry, creator, canonical identity, lagging-copy, parked-source, capability
vocabulary, and predecessor/reducer findings. rev16 adds only the alias
normalization closure and the corrected callback negative instrument; it does
not widen caller/publication/storage ownership.
