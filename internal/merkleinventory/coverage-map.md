# Mesh RPC 2 inventory coverage map

The producer and rework briefs do not include a formal surface table. This map
records the explicit acceptance clause and the rework's finite test surfaces;
the missing brief table is reported as a brief gap rather than treated as
implicit coverage.

| Surface | Production entry | Named test | Narrowing killed by the named test |
| --- | --- | --- | --- |
| §11.4 rule 1: sort and deduplicate identity set | `Index.Root` / `Index.Child` → `ComputeNode` | `TestNodeRuleShapesAndDepth64IntegrityFailure` | `rule1-duplicate-set-member` |
| §11.4 rule 2: empty root body/hash | `Index.Dispatch` → `childrenBody` → `ComputeNode` | `TestDispatchServesNormativeChildrenBody/empty` | `rule2-empty-root-count` |
| §11.4 rule 3: singleton root leaf | `Index.Dispatch` → `childrenBody` → `ComputeNode` | `TestDispatchServesNormativeChildrenBody/singleton` | `rule3-root-singleton-leaf` |
| §11.4 rule 4: retain a sole child below root | `Index.Child` → `ComputeNode` | `TestRuleFourRetainsSingleChildAtEverySharedPrefix` | `rule4-retain-single-child` |
| §11.4 rule 5: depth-64 duplicate integrity refusal | `Index.Child` → `ComputeNode` | `TestNodeRuleShapesAndDepth64IntegrityFailure` | `rule5-depth64-duplicate` |
| Empty, singleton, branch roots, both branch child hashes and exact response bodies | `Index.Child` / `Index.Dispatch` → `childrenBody` → `ComputeNode` | `TestNormativeRootAndChildFixtures`, `TestDispatchServesNormativeChildrenBody` | `rule2-empty-root-count`, `rule3-root-singleton-leaf`, `children-label-order` (branch subtest) |
| All six MIXED-NS-1 roots | `Index.Root` → `nodeFor` → `ComputeNode` | `TestMixedNS1RootsFromNormativeSyntheticIDs` | `mixed-ns1-root-member-narrowing` omits the exact synthetic Tombstone Acknowledgement identity at the shared production root builder. Fixture identities are seeded only for this normative identity-set row. |
| Membership table, unknown schemas, Descriptor/Chunk/local classes | `Index.AddJSON` / `Index.AddBlob` → `ClassifyJSON` | `TestRPC2SchemaMembershipTableIsTotalAndDisjoint`, `TestMixedNSN1RejectsDescriptorRelabelingChunksAndLocalMarker`, `TestBlobDescriptorOwnsManifestMembershipAndCompleteRawBlob` | `unknown-schema-fail-closed`, `descriptor-namespace`, `excluded-local-marker`, `blob-chunk-content-mismatch` |
| Four schema rows without complete validators stay fail-closed | `Index.AddJSON` → `ClassifyJSON` → `canonicaljson.VerifyObjectIdentity` | `TestUnsupportedSessionRecord31RemainsFailClosed`, `TestUnsupportedMaterializationPlan10RemainsFailClosed`, `TestUnsupportedMaterializationPlan20RemainsFailClosed`, `TestUnsupportedTaskBoardBundle10RemainsFailClosed` | One isolated matching `unsupported-schema-*` narrowing per row; owner Stories are listed in `CONFORMANCE-MATRIX.md`. |
| Strict request field types, member types, missing and duplicate fields | `Index.Dispatch` for roots/children; `Index.ObjectsGet` for objects.get | `TestDispatchRequestShapeStrictTypes` | `children-prefix-type-null` (`inventory_children_prefix_null`). Test cells cover all six JSON kinds, relevant array member kinds, missing and duplicate for each field. |
| Prefix grammar at every length, existing and absent nodes | `Index.Dispatch` → `childrenBody` → `validPrefix` | `TestDispatchPrefixAxisCoversEveryLengthAndAlphabet` | `emptychild2` (`lower_hex_len_02_absent`), `prefix-length-65`, `uppercase-prefix`; prior `missing-prefix-not-found` also targets an absent child. |
| Server requested namespace versus each true JSON-object namespace | `Index.ObjectsGet` | `TestObjectsGetRefusesEveryForeignNamespacePair` | `crossns` (`requested_manifest_object_record`). All 25 mismatch cells across the 6×5 axis require literal `integrity_failure`; 5 diagonal controls succeed. |
| Client namespace check against hostile object source | `FetchObjects` → `decodeOneWireObject` | `TestFetchObjectsRefusesEveryHostileForeignNamespacePair` | `fetchns` (`requested_manifest_object_record`). All 20 mismatches among five JSON namespaces require literal `integrity_failure`; 5 diagonal controls succeed. |
| Bounded IDs, encodings, batching and response line | `Index.ObjectsGet`; `FetchObjects` | `TestObjectsGetRefusesRequestIDCountOutsideBound`, `TestObjectsGetRefusesUnsortedDuplicateAndUnavailableEncodingRequests`, `TestObjectsGetValidatesRequestedNamespaceAndNegotiatedLineLimit`, `TestFetchObjectsUsesBoundedSingletonBatches` | `objects-get-request-count`, `objects-get-duplicate-id`, `objects-get-cbor-only`, `objects-get-line-limit`, `objects-get-unbounded-batch` |
| MIXED-NS-EXCHANGE identity-set walk and convergence | `Index.Dispatch` / `MissingNamespaceIDs` | `TestMixedNSExchangeIdentityLevelSyntheticIDs` | `identity-walk-blob-narrowing`; this proof claims no object validation, as pinned §11.4 defines the synthetic IDs by validated identity. |
| MIXED-NS-EXCHANGE validated real-byte retrieval and record-before-blob admission | `MissingObjectIDs` / `FetchObjects` / `Index.AddJSON` / `Index.AddBlob` | `TestInProcessExchangeWalksChangedNamespacesAndFetchesExactObjects` | `exchange-objects-get-schema-membership` narrows the requested namespace check and is killed by this test alone. |
| Same-digest, different-byte quarantine and abort | `Index.AddJSON` → `preflightLocked` / `quarantineLocked` | `TestAddJSONQuarantinesSameIDDifferentStoredBytes` | `same-id-quarantine` admits the exact conflicting byte variant and is killed by this test alone. |
| Tombstone/Acknowledgement union with no timestamp winner or execution | `Index.AddJSON` / `MissingObjectIDs` / `FetchObjects` | `TestInProcessTombstoneUnionRetainsBothTimesAndAcknowledgements` | `tombstone-timestamp-winner` chooses the later `created_at` Tombstone and is killed by this test alone; the test also asserts both Acknowledgements remain. |

The contextual namespace is intentionally not an `objects.get` JSON member.
`Index.Dispatch` does not dispatch that operation because it has no namespace
context; the production serving entry is `Index.ObjectsGet`. Raw blob bytes
remain on the §11.5 transfer path, and RPC 3/4 namespace axes are not admitted
by this RPC 2 constructor. These are explicit bounds in the conformance
matrix, not untested claims of this map.

## TASK-260830-147hsj derived coverage map

The producer brief did not include a formal surface table. These six rows are
the measured decomposition of its one-sentence acceptance criterion; that
brief gap is also recorded in the handoff note and conformance matrix.

| Surface / AC row | Production call site | Named test | Narrowing killed by that test alone |
| --- | --- | --- | --- |
| Discover missing JSON objects and fetch by digest | `DurableIndex.SyncFrom` → `MissingObjectIDs` / `walkMissing` → `FetchObjects` → `Index.ObjectsGet` | `TestDurableSyncFindsMissingJSONObjectsAndRetainsTombstoneRecords` | `sync-missing-record-fetch` |
| Persist only validated objects; identical add stays idempotent across restart | `DurableIndex.AddJSON` → `ClassifyJSON` → `durableInstallNoReplace`; reopen via `OpenDurable` → `load` | `TestDurableJSONAddIsIdempotentAcrossCrashRestartByNamespace`, `TestDurableAddRejectsBeforePersistingInvalidObjects`, `TestDurableStoreRejectsCorruptActiveBytesOnOpen` | `durable-unknown-schema-admit`, `durable-event-replay`, `durable-load-skips-forged-object` |
| Quarantine same-digest/different-byte data and abort sync | `DurableIndex.SyncFrom` common-ID audit → `addJSONLocked` → quarantine | `TestDurableSyncAuditsCommonIDsAndQuarantinesDifferentBytes`, `TestDurableConflictCrashRestoresQuarantineAndAbortsSync` | `durable-same-id-different-bytes`, `sync-common-record-audit`, `quarantine-crash-recovery` |
| Union Tombstones/Acks as records and do not execute them | `DurableIndex.SyncFrom` → `addJSONLocked` / `validateTombstoneAckClosureLocked` | `TestDurableSyncFindsMissingJSONObjectsAndRetainsTombstoneRecords`, `TestDurableSyncRefusesUnclosedAcknowledgementThenRecovers`, `TestDurableSyncRefusesAcknowledgementWithMismatchedTombstoneSubject` | `union-ack-arrival-order`, `sync-skip-tombstone-union`, `sync-unclosed-tombstone-ack`, `durable-ack-subject-link` |
| Preserve concurrent lease branches, derive all heads after union | `Index.RebuildProjection` → `Reader.LeaseHeadsForSession` → `sessstate.Projector.Project` | `TestLeaseHeadsForSessionReturnsEveryTupleAcrossTimestampPerturbation`, `TestLeaseHeadsForSessionCoversGeneratedCardinalityRange`, `TestLeaseHeadsForSessionRefusesConflictingBytesForOneLeaseID`, `TestProjectionRebuildExhaustsUnionArrivalsWithoutTimestampAuthority` | `union-lease-tuple-omission`, `union-lease-generated-cardinality-omission`, `union-lease-created-at-winner`, `union-lease-conflicting-same-id`, `projection-drops-union-head` |
| Projection consumes only a closed union containing exact authority bytes | `Index.RebuildProjection` / `DurableIndex.RebuildProjection` → `validateTombstoneAckClosure` → `sessrepo` record/event reads | `TestProjectionRebuildRefusesAuthorityBytesMissingFromUnion`, `TestProjectionRebuildRefusesUnclosedAcknowledgement` | `projection-missing-union-record`, `projection-missing-union-event`, `projection-skip-unclosed-ack` |
| Projection independent of arrival order and timestamp | Same projection call sites as above | `TestProjectionRebuildExhaustsUnionArrivalsWithoutTimestampAuthority` (720 permutations × 2 timestamp assignments) | `union-lease-created-at-winner`, `projection-drops-union-head` |
| Carry-over: FetchObjects duplicate-ID refusal | `FetchObjects` | `TestFetchObjectsRefusesDuplicateIDs` | `fetch-dup-ids` |
| Carry-over: base64url exact round-trip rejects embedded newlines | `FetchObjects` → `decodeOneWireObject` | `TestFetchObjectsRefusesNewlineBearingBase64urlData` | `b64-roundtrip` |
| Carry-over: recursive walk refuses local quarantined identity | `MissingObjectIDs` → `walkMissing` | `TestMissingObjectIDsRefusesLocalQuarantinedIdentity` | `walk-quarantine` |
| Carry-over: recursive walk refuses stored cross-namespace identity | `MissingObjectIDs` → `walkMissing` | `TestMissingObjectIDsRefusesCrossNamespaceIdentity` | `walk-crossns` |

The cross-namespace walker test deliberately seeds malformed internal state;
`ClassifyJSON` and `AddJSON` cannot create that state without a SHA-256
collision. This is a defensive reachable helper arm, not a valid public-ingest
case. All listed mutant runs use their named test alone and have raw logs in
the attached evidence archive. The mutation harness also has an applied
neutral comment control that survives.

## TASK-260830-2h5uv9 generated sync convergence

The producer brief supplied no surface table. The following task-scoped rows
are the measured decomposition of its single written acceptance criterion;
the missing brief table is recorded as a brief gap.

| Surface / AC row | Production call site | Named test | Narrowing killed by that test alone |
| --- | --- | --- | --- |
| Generated order, duplicate, gap, skew, peer-subset and repeated-pass product | `DurableIndex.SyncFrom` → `MissingObjectIDs` / `FetchObjects`; `RebuildProjection` | `TestDurableSyncGeneratedPerturbationProduct` | `sync-repeat-record-quarantine-write`, `union-lease-created-at-winner`, `sync-partial-peer-regresses-local-root` |
| Arrival reorder | `DurableIndex.SyncFrom` → `fetchAndAdd` | `TestDurableSyncArrivalOrderConverges` | `sync-arrival-order-truncates-fetch` |
| Delayed gap fill | `DurableIndex.SyncFrom` → missing-object fetch | `TestDurableSyncGapFillsOnLaterPass` | `sync-missing-record-fetch` |
| Both timestamp-skew directions | `DurableIndex.SyncFrom` → `RebuildProjection` | `TestDurableSyncClockSkewDoesNotSelectLeaseWinner` | `union-lease-created-at-winner`, `projection-drops-union-head` |
| Partial peer root monotonicity | `DurableIndex.SyncFrom` | `TestDurableSyncPartialPeerNeverRegressesLocalRoots` | `sync-partial-peer-regresses-local-root` |
| Duplicate delivery and byte-identical repeated sync | `DurableIndex.SyncFrom` | `TestDurableSyncDuplicateDeliveryIsByteIdentical`, `TestDurableSyncGeneratedPerturbationProduct` | `sync-repeat-record-quarantine-write` |
| Same-digest/different-byte quarantine and abort | `DurableIndex.SyncFrom` → common-ID audit | `TestDurableSyncSameDigestDifferentBytesIsTypedConflict` | `sync-common-record-audit` |
| Schema/namespace mismatch abort at the sync entry | `DurableIndex.SyncFrom` → `Index.ObjectsGet` | `TestDurableSyncRefusesPeerNamespaceMismatch` | `sync-peer-namespace-mismatch` |
| Unclosed Tombstone/Acknowledgement link refusal and recovery | `DurableIndex.SyncFrom` → union closure | `TestDurableSyncUnclosedAcknowledgementAborts` | `sync-unclosed-tombstone-ack` |

Registry cases decoded and clause-bound: `story-260830-nxqqaw-inventory-exchange`,
`story-260830-147hsj-durable-union-projection`, and
`story-260830-2h5uv9-order-duplicate-gap-convergence`. The registry regression
test `TestStoryFinalAntiEntropyCasesAppearInDecodedClauseLists` fails if any
of these defined acceptance cases loses all of its clause references.

Measured task AC ratio: **1 of 1 written AC rows driven** through
`DurableIndex.SyncFrom`; ten behavior rows above expose the sentence's
individual production surfaces. The task brief had no formal surface table.

## TASK-260830-2h5uv9 generated-sync coverage map

The producer brief has no formal surface table. This measured decomposition
records the gap and maps the written acceptance criterion to named production
entries and isolated narrowing mutants.

| Surface | Production entry | Named test | Narrowing killed by that test |
| --- | --- | --- | --- |
| Full generated order × duplicate × delayed-gap × skew × partial-peer × repeated-sync product; independent roots and projection | `DurableIndex.SyncFrom` → `MissingObjectIDs` / `FetchObjects`; `DurableIndex.RebuildProjection` | `TestDurableSyncGeneratedPerturbationProduct` | `sync-repeat-record-quarantine-write`, `sync-partial-peer-regresses-local-root` |
| Arrival reorder and every permutation through N=5 | `DurableIndex.SyncFrom` → `fetchAndAdd` | `TestDurableSyncArrivalOrderConverges` | `sync-arrival-order-truncates-fetch` |
| Duplicate delivery and byte-identical repeated sync | `DurableIndex.SyncFrom` | `TestDurableSyncDuplicateDeliveryIsByteIdentical` | `sync-repeat-record-quarantine-write` |
| Gap, explicit refusal while Tombstone/Acknowledgement closure is incomplete, then recovery after fill | `DurableIndex.SyncFrom` → missing-object fetch and union closure | `TestDurableSyncGapFillsOnLaterPass`, `TestDurableSyncRefusesUnclosedAcknowledgementThenRecovers` | `sync-missing-record-fetch`, `sync-unclosed-tombstone-ack` |
| Both clock-skew directions leave the lease tuple winner unchanged | `DurableIndex.SyncFrom` → `RebuildProjection` → `LeaseHeadsForSession` | `TestDurableSyncClockSkewDoesNotSelectLeaseWinner` | `union-lease-created-at-winner`, `projection-drops-union-head` |
| Every compatible partial peer subset preserves local root monotonicity | `DurableIndex.SyncFrom` | `TestDurableSyncPartialPeerNeverRegressesLocalRoots` | `sync-partial-peer-regresses-local-root` |
| Same semantic identity with different bytes is quarantined and aborted even when the peer also has missing identities or a successful prior sync | `DurableIndex.SyncFrom` common-ID audit → quarantine | `TestDurableSyncConflictWithPartialOverlapPeer`, `TestDurableSyncGeneratedPerturbationProduct`, `TestDurableSyncSameDigestDifferentBytesIsTypedConflict` | `sync-common-audit-partial-overlap-peer`, `sync-common-audit-only-without-missing`, `sync-common-audit-first-namespace-only`, `sync-common-audit-first-sync-only` |
| Every common identity is audited at each sorted common-ID position while peer-only identities are present, across reordered, duplicated, skewed and repeated conflict delivery | `DurableIndex.SyncFrom` common-ID audit → quarantine and abort | `TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord` | `sync-common-audit-first-id-only`, `sync-common-audit-last-id-only`, `sync-common-audit-even-ids-only` |
| Common-ID audit is not gated by missing IDs, peer size, or namespace position | `DurableIndex.SyncFrom` AST call site and runtime refusal | `TestDurableSyncCommonIDAuditIsUnconditionalInAST`, `TestDurableSyncConflictWithPartialOverlapPeer`, `TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord` | `sync-common-audit-ast-control`, `sync-common-audit-early-namespace-return`, `sync-common-audit-first-namespace-only` |
| Unjustified `t.Skip`, `t.Skipf`, or `t.SkipNow` cannot hide package tests; the checked tree has no Python bytecode | `TestNoUnjustifiedTestSkips` AST census; `TestPackageTreeHasNoPythonBytecode` filesystem walk | `TestNoUnjustifiedTestSkips`, `TestSkipCensusRejectsAnUnjustifiedPlantedSkip`, `TestSkipCensusRejectsAPlantedSkipWhenItsTokenRemains`, `TestPackageTreeHasNoPythonBytecode`, `TestPythonBytecodeHygieneRejectsPlantedArtifacts` | `skip-census-unjustified-planted-skip`, `skip-census-token-preserving-behavior`; planted `.pyc`/`__pycache__` rejected by `TestPythonBytecodeHygieneRejectsPlantedArtifacts` |
| Namespace/schema mismatch stops the production sync entry with the literal code | `DurableIndex.SyncFrom` → `Index.ObjectsGet` | `TestDurableSyncRefusesPeerNamespaceMismatch` | `sync-peer-namespace-mismatch` |
| Reference lease projection preserves competing lease and losing-event branch | `DurableIndex.RebuildProjection` → `LeaseHeadsForSession` → `sessstate.Projector.Project` | `TestDurableSyncGeneratedPerturbationProduct`, `TestProjectionRebuildExhaustsUnionArrivalsWithoutTimestampAuthority` | `union-lease-tuple-omission`, `union-lease-created-at-winner` |

**Measured task acceptance ratio: 1 of 2 criteria fully driven by named tests.**
The second criterion's exact fixtures, negative/refusal paths, crash recovery,
and idempotency have named passing tests. Its no-unsupported-capability clause
is a stated scope bound: this library adds no public command, doctor, provider,
or network advertisement entry point. The brief has no formal surface table;
this missing table is a brief gap, not a coverage waiver. The generated
production-sync property has 38,000 valid-union and 37,765 conflict cases, and
the always-on common-ID position test adds 432 conflict cases with peer-only
identities.
