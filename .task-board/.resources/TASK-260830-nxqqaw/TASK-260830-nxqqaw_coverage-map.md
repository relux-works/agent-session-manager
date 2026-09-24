# TASK-260830-nxqqaw coverage map

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
