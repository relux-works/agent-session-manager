# Inventory conformance traceability

Authority: `internal/specdoc/SPEC.v0.7.0.md`, §11.3 (RPC rows and bounds,
lines 7555–7561 and 7734–7738), §11.4 (node rules and fixtures, lines
7764–7934), §11.5 (raw blob transfer), and §10 (object classes). The task
record's v0.5.0 reference is stale; the producer pins the v0.7.0 repository
copy. `CONFORMANCE-MATRIX.md` and `IMPORTER-OUTCOMES.md` carry the complete
axis and direct-importer censuses.

The first Story leaf established the RPC 2 inventory and in-process real-byte
exchange. The `TASK-260830-147hsj` supplement below extends its §11.4 union
binding with durable JSON persistence, common/missing-ID exchange, lease-head
derivation, and projection rebuild. The story-final registry binds all three
leaves' acceptance cases to the clause rows their named tests drive.

## Clause-to-test bindings

| Specification clause | Production call site | Named evidence | Narrowing evidence or stated bound |
| --- | --- | --- | --- |
| §11.4 node rules 1–5 and `urn:ax:merkle-node:1` JCS encoding | `Index.Root` / `Index.Child` → `nodeFor` → `ComputeNode` → `buildNode` | `TestNormativeRootAndChildFixtures`, `TestDispatchServesNormativeChildrenBody`, `TestRuleFourRetainsSingleChildAtEverySharedPrefix`, `TestNodeRuleShapesAndDepth64IntegrityFailure` | Isolated `rule1-duplicate-set-member` through `rule5-depth64-duplicate`; depth-64 count>1 refuses with literal `integrity_failure`. |
| §11.4 empty/singleton/branch fixtures and child hashes | `Index.Dispatch` → `childrenBody` → `Index.Child` → `ComputeNode` | `TestDispatchServesNormativeChildrenBody` pins all three complete response bodies; `TestNormativeRootAndChildFixtures` pins the branch children | `rule2-empty-root-count`, `rule3-root-singleton-leaf`, and `children-label-order` (branch subtest) run alone and are killed. |
| §11.4 MIXED-NS-1 six roots | `Index.Root` → `nodeFor` → `ComputeNode` | `TestMixedNS1RootsFromNormativeSyntheticIDs` checks the six literal counts and roots through the production root entry | `mixed-ns1-root-member-narrowing` omits one exact fixture identity and is killed by this test alone. The spec says synthetic IDs stand for schema-valid objects and their contents do not feed the trie. Test-only seeding is limited to this exact hash fixture; ingestion is separately driven through `ClassifyJSON` and `Index.AddJSON`. |
| §11.4 MIXED-NS-EXCHANGE identity-level proof | `Index.Dispatch` → `rootsBody` / `childrenBody`; `MissingNamespaceIDs` → `walkMissing` | `TestMixedNSExchangeIdentityLevelSyntheticIDs` drives both peer identity sets through `inventory.roots`, production child serving along every missing-ID path, the recursive walk, and convergence of all six roots | The test claims inventory identity behavior only. Pinned §11.4 states the synthetic IDs stand for schema-valid objects/bytes whose contents do not feed the trie. `identity-walk-blob-narrowing` and `objects-get-blob-transfer-boundary` run against the test alone. |
| §11.4 RPC 2 namespace membership table and §10 identity classes | `Index.AddJSON` → `ClassifyJSON` → `canonicaljson.VerifyObjectIdentity`; `Index.AddBlob` for raw bytes | `TestRPC2SchemaMembershipTableIsTotalAndDisjoint`, `TestMixedNSN1RejectsDescriptorRelabelingChunksAndLocalMarker`, `TestBlobDescriptorOwnsManifestMembershipAndCompleteRawBlob` | `descriptor-namespace`, `unknown-schema-fail-closed`, and `excluded-local-marker` narrowings. The 19 included rows and 42 excluded IDs are enumerated in `CONFORMANCE-MATRIX.md`. Four additional per-row tests and admitting narrowings keep the incomplete schemas fail-closed; their namespace mapping stays total. |
| §10 / §11.4 mapped schemas without a complete canonicaljson shape | `Index.AddJSON` → `ClassifyJSON` → `canonicaljson.VerifyObjectIdentity` | `TestUnsupportedSessionRecord31RemainsFailClosed`, `TestUnsupportedMaterializationPlan10RemainsFailClosed`, `TestUnsupportedMaterializationPlan20RemainsFailClosed`, `TestUnsupportedTaskBoardBundle10RemainsFailClosed` | Each named test asserts literal `integrity_failure`, all six roots/counts unchanged, and no quarantine. One isolated narrowing per row is killed by that row's test. Owners are `STORY-260830-4n0fo8 — authoritative-record-schema-core`, `STORY-260830-2r137i — workspace-materialization-and-groups` (both plan versions), and `STORY-260830-27pqyi — task-board-bridge-and-bundle`; the decision is quoted in `CONFORMANCE-MATRIX.md`. |
| §11.4 MIXED-NS-N1 | `Index.Root`; `Index.AddJSON` / `ClassifyJSON`; `Index.ObjectsGet` with caller namespace context | `TestMixedNSN1RejectsDescriptorRelabelingChunksAndLocalMarker`, `TestObjectsGetRevalidatesStoredSchemaMembership` | The N1 test first pins all six MIXED-NS-1 roots, then models Descriptor-as-record, independent Chunk enumeration, and local-marker inclusion; the resulting `record`, `manifest`, and `blob` roots differ while event/Tombstone/Acknowledgement roots remain exact. Production classification separately requires Descriptor → manifest and excludes Chunk/local marker; `objects-get-schema-membership` kills wrong-namespace admission. |
| §11.3 `inventory.roots` exact ordered result | `Index.Dispatch` → `rootsBody` → `rpcwire.DecodeRequest` / `EncodeSuccess` | `TestDispatchServesAllSixInventoryRootsInRPCWireOrder` | Namespace order/cardinality/membership is owned by rpcwire; its separate narrowings run alone in `internal/rpcwire/mutations.py`. This task adds no duplicate validator. |
| §11.3 request-body field types, required members, and closed shape | `Index.Dispatch` → `rpcwire.DecodeRequest` / `rootsBody` / `childrenBody` / `objectsGetRequest` → `decodeObjectsGetRequest` | `TestDispatchRequestShapeStrictTypes`, `TestDispatchRejectsEveryNonExactOperationBodyShape` | Every member of `inventory.roots`, `inventory.children`, and `objects.get` is driven through `Dispatch` with wrong-kind, casefolded, missing, extra, and duplicated variants. Every refusal has no success body and the literal local refusal `invalid inventory request`; `children-prefix-type-null` and `server-casefold-prefix` are killed alone. The namespace for `objects.get` is contextual input, not a body member. |
| §11.3 exact outer envelopes and returned WireObject shape (rev3 rework) | `Index.Dispatch` → `rpcwire.DecodeRequest`; `FetchObjects` → `rpcwire.DecodeResponse` / `decodeOneWireObject` | `TestDispatchRejectsEveryNonExactOuterRequestEnvelopeShape`, `TestFetchObjectsRejectsEveryNonExactSuccessWireShape`, `TestFetchObjectsRejectsNonObjectItemsAndWrongArrayCardinality`, `TestFetchObjectsRefusesWellFormedRPCFailureEnvelope` | Every request and success/failure response envelope member plus objects-body/WireObject member has wrong-kind, casefolded, missing, extra, and duplicated cases. Response array item kinds and singleton cardinality are driven through hostile raw `ObjectSource` bodies; malformed or well-formed failure responses yield exact local `invalid inventory object`. `client-casefold-wireobject` and `client-two-wireobjects` kill alone. Outer envelopes and nested Structured Error are delegated to the landed `internal/rpcwire`/`axerror` validators. |
| Single strict decode path and decoder bypass guard (rev3 rework) | All direct JSON reads → `strict_json.go:strictJSON`; direct unmarshalling is allowlisted only inside `strictJSON` | `TestStrictJSONRejectsStructDestinations`, `TestWireDecoderASTGuardRejectsAliasedUnmarshalBindings`, `TestWireDecoderGuardScansProductionAliases` | The package AST guard rejects aliased `encoding/json.Unmarshal`, bound decoder function values, `json.NewDecoder`, dot imports, and `raw*` accessors. Applied alias control plant `guard-aliased-json-unmarshal` is killed alone. See the complete site-by-site census in `CONFORMANCE-MATRIX.md`. |
| §11.3 client encoding negotiation (rev2 note) | `FetchObjects` → `decodeOneWireObject` | `TestFetchObjectsRefusesCBORLabelAfterJSONRequest` | A hostile JSON response labelled `cbor` after a JSON request is refused as `invalid inventory object`; `client-fetch-cbor-label` admits the label and is killed by this test alone. |
| §11.3 `inventory.children` prefix, child and leaf bounds | `Index.Dispatch` → `childrenBody` → `Index.Child` → `validPrefix` | `TestDispatchPrefixAxisCoversEveryLengthAndAlphabet`, `TestDispatchBuildsSixteenSortedNibbleChildren` | Lowercase hex `[0..65]`; materialized and valid-absent paths at every length `[1..64]`; uppercase/non-hex `[1..65]`; root `[0]`; child counts `[0,1,2,16]`; leaf IDs `[0,1]`. Absent prefixes assert literal `not_found`; lengths >64 and invalid alphabet assert the literal local request refusal. `emptychild2` is killed by the two-nibble absent cell; prior `prefix-length-65`, `uppercase-prefix`, `missing-prefix-not-found`, and `rule4-retain-single-child` remain. |
| §11.3 `objects.get` schema, ordering, encoding and bound | `Index.ObjectsGet` → `decodeObjectsGetRequest` / `decodeOneWireObject` → `ClassifyJSON`; `FetchObjects` | `TestObjectsGetRefusesUnsortedDuplicateAndUnavailableEncodingRequests`, `TestObjectsGetRefusesRequestIDCountOutsideBound`, `TestObjectsGetValidatesRequestedNamespaceAndNegotiatedLineLimit`, `TestFetchObjectsUsesBoundedSingletonBatches`, `TestObjectsGetRefusesEveryForeignNamespacePair`, `TestFetchObjectsRefusesEveryHostileForeignNamespacePair` | ID count `[0,1,2,4096,4097]`, order `[ascending,descending]`, duplicate `[absent,present]`, encoding count `[0,1,2]`, JSON/CBOR, and negotiated line `[limit,one byte over]`; server namespace matrix `[6×5]` with all 25 off-diagonal refusals and client JSON namespace matrix `[5×5]` with all 20 off-diagonal refusals. Every cross-namespace refusal asserts literal `integrity_failure`; `crossns` and `fetchns` narrow and kill one manifest/record cell apiece. Blob is bounded to the §11.5 transfer path; RPC 3/4 namespaces are not negotiated here. |
| §11.5 Descriptor/complete raw bytes boundary | `Index.AddBlob` → `verifyDescriptorBytes`; `Index.AddJSON` | `TestBlobDescriptorOwnsManifestMembershipAndCompleteRawBlob`, `TestBlobDescriptorByteReferencesAreVerifiedBeforeAdmission` | Descriptor identity stays in manifest; complete bytes enter blob only after whole digest, size, range and chunk digest validation. Chunk staging/retry is owned by `STORY-260830-14qxuc — resumable-blob-and-manifest-transfer`; workspace finalization by `STORY-260830-2r137i — workspace-materialization-and-groups`. |
| §11.4 MIXED-NS-EXCHANGE validated real-byte proof | `MissingObjectIDs` → `walkMissing`; `FetchObjects` → `ObjectSource.ObjectsGet`; `Index.AddJSON` / `AddBlob` | `TestInProcessExchangeWalksChangedNamespacesAndFetchesExactObjects` checks exact missing Checkpoint/Descriptor IDs, schema/digest validation, changed roots, record-before-blob order, and six literal roots derived from its real bytes | This is a separate proof from synthetic identity-level roots. `exchange-objects-get-schema-membership` kills the named test alone. The `objects.get` conflict refusal is at the production `Index.ObjectsGet` entry; `TestAddJSONQuarantinesSameIDDifferentStoredBytes` verifies quarantine and error propagation. |
| §11.4 union rules 1–4 and 7 | `DurableIndex.AddJSON` → `ClassifyJSON` → atomic semantic-ID install; `DurableIndex.SyncFrom` → common-ID `FetchObjects` audit and `MissingObjectIDs` fetch; deferred closure via `ValidateUnionClosure` | `TestDurableAddRejectsBeforePersistingInvalidObjects`, `TestDurableJSONAddIsIdempotentAcrossCrashRestartByNamespace`, `TestDurableSyncAuditsCommonIDsAndQuarantinesDifferentBytes`, `TestDurableSyncFindsMissingJSONObjectsAndRetainsTombstoneRecords`, plus the existing in-memory identity and Tombstone/Acknowledgement tests | `durable-event-replay`, `durable-same-id-different-bytes`, `sync-common-record-audit`, `sync-missing-record-fetch`, `sync-skip-tombstone-union`, and `union-ack-arrival-order` are killed by the named tests alone. The sync test confirms retained Tombstone/Ack bytes and an unchanged `sessrepo` session row. Blob/chunk staging remains §11.5-owned under rule 7. |
| §11.4 union rules 5–6; §5.3 lease/event semantics | `Index.RebuildProjection` → `sessquery.Reader.LeaseHeadsForSession` → `sessstate.Projector.Project`; durable wrapper `DurableIndex.RebuildProjection` | `TestLeaseHeadsForSessionReturnsEveryTupleAcrossTimestampPerturbation`; `TestProjectionRebuildExhaustsUnionArrivalsWithoutTimestampAuthority` drives every order of six union objects under two opposing timestamp assignments; `TestDurableConflictCrashRestoresQuarantineAndAbortsSync` also exercises persisted refusal | `union-lease-tuple-omission`, `union-lease-created-at-winner`, and `projection-drops-union-head` are isolated narrowings. Losing event bytes remain in the union; the fixed `sessrepo` authoritative chain alone supplies event inputs to the reducer. Projection is deterministic for that authority snapshot plus the union set; this leaf does not rebuild an arbitrary DAG/branch index from union event objects. |
| §11.4 compatibility with later Mesh RPC namespaces | `Index.New` → `rpcwire.Namespaces` | `TestNewRefusesHigherRPCNamespaceSets` | This leaf implements the six RPC 2 namespaces of §11.3. Pinned §11.8 adds RPC 3 `directory_record`; §11.9 adds RPC 4 `terminal_backend_evidence`. `Index.New` refuses those versions and this package makes no RPC 3/4 serving claim. |

## Mirrored gate × entry × axis census

This census mirrors the matrix's full table. Every axis cell names its
production test and isolated narrowing above or states the owner/bound.

| Gate and production entry | Axis enumeration | Evidence or bound |
| --- | --- | --- |
| RPC namespace request: `Index.Dispatch` → `rpcwire.DecodeRequest` | RPC `[2.0.0,3.0.0,4.0.0,5.0.0,6.0.0]`; namespace `[blob,event,manifest,record,tombstone,tombstone_ack,unknown]`; count `[0,1,6,7]`; sorted/descending/duplicate | `TestDispatchServesAllSixInventoryRootsInRPCWireOrder`; rpcwire envelope test and seven named mutants. |
| Strict closed wire decoder: `Index.Dispatch` / `FetchObjects` | Request-body, outer request/response, objects-body and WireObject member `[wrong type: string/number/boolean/null/array/object; casefold/missing/extra/duplicate]`; response array item `[same six kinds]`; object count `[0,1,2]` | `TestDispatchRejectsEveryNonExactOperationBodyShape`, `TestDispatchRejectsEveryNonExactOuterRequestEnvelopeShape`, `TestFetchObjectsRejectsEveryNonExactSuccessWireShape`, `TestFetchObjectsRejectsNonObjectItemsAndWrongArrayCardinality`; exact package call sites are in the decoder census in `CONFORMANCE-MATRIX.md`. `guard-aliased-json-unmarshal`, `server-casefold-prefix`, `client-casefold-wireobject`, `client-two-wireobjects`, and `client-fetch-cbor-label` are each killed alone. Outer envelopes compose the landed `rpcwire` validators. |
| Prefix validation: `Index.Dispatch` → `childrenBody` → `validPrefix` | Lowercase-hex length `[0..65]`; uppercase and non-hex `[1..65]`; materialized/absent `[1..64]`; root `[0]` | `TestDispatchPrefixAxisCoversEveryLengthAndAlphabet`; `emptychild2` is killed at absent length 2; `prefix-length-65`, `uppercase-prefix`, and `missing-prefix-not-found` remain isolated. |
| Trie membership/count: `Index.Root` / `Index.Child` → `ComputeNode` | Count `[0,1,2,5,16]`; duplicate `[absent,present]`; ID order `[ascending,unsorted]`; shared prefix `[0..63]`; depth `[64]` | Exact node fixtures and rule1–rule5 narrowings; count>1 at depth64 is `integrity_failure`. |
| Roots/children bodies: `Index.Dispatch` → `rootsBody` / `childrenBody` | Namespace order `[blob,event,manifest,record,tombstone,tombstone_ack]`; root count `[0,1,2,5,16]`; children `[0..16]`; child labels `[0..9,a..f]`; ids `[0,1]` | Exact roots/children body tests; rpcwire owns request member/ordering validation. |
| Schema membership: `Index.AddJSON` → `ClassifyJSON` | `record`: session-record 1/2/3/3.1, lease, checkpoint, workspace-group, provider-identity; `event`: session-event 1/2/3/4; `manifest`: transfer-manifest, blob, materialization-plan 1/2, task-board-bundle; `tombstone`; `tombstone_ack`; raw blob; 42 excluded schema IDs; unknown/missing schema | `TestRPC2SchemaMembershipTableIsTotalAndDisjoint`; Descriptor/class/Chunk/local negatives and N1 are driven. The four mapped but unsupported rows each have a named refusal test and isolated admitting mutant below. |
| Incomplete validator rows: `Index.AddJSON` → `ClassifyJSON` → `canonicaljson.VerifyObjectIdentity` | `session-record@3.1.0`; `materialization-plan@1.0.0`; `materialization-plan@2.0.0`; `task-board-bundle@1.0.0` | Four `TestUnsupported…RemainsFailClosed` tests assert `integrity_failure`, unchanged all-six roots/counts, and empty quarantine. One narrowing per schema row is killed alone. Owner mapping: `STORY-260830-4n0fo8`, `STORY-260830-2r137i` (both plan versions), `STORY-260830-27pqyi`. |
| Identity-level synthetic exchange: `Index.Dispatch` / `MissingNamespaceIDs` | Six namespaces; missing `[record:Z(3), manifest:T(2), blob:sha256:555…555]`; changed roots `[record,manifest,blob]`; converged roots `[all six fixture roots]` | `TestMixedNSExchangeIdentityLevelSyntheticIDs`; `identity-walk-blob-narrowing` proves blob identity collection; `objects-get-blob-transfer-boundary` preserves the no-JSON-object-transfer refusal. No byte validation is claimed. |
| Request IDs and response bytes: `Index.ObjectsGet` / `FetchObjects` | ID count `[0,1,2,4096,4097]`; sorted/descending/duplicate; encoding count `[0,1,2]`; JSON/CBOR; exact/over line limit; batch `[1,2]`; stored/asked namespace match | Production tests and `objects-get-*` narrowings in the clause table. |
| Requested/object namespace pair: `Index.ObjectsGet` / `FetchObjects` | Server requested namespace × true JSON namespace `[6×5]`; client requested namespace × true JSON namespace `[5×5]`; match/mismatch `[diagonal/off-diagonal]`; client source `[hostile foreign object]` | `TestObjectsGetRefusesEveryForeignNamespacePair` and `TestFetchObjectsRefusesEveryHostileForeignNamespacePair`; all 25 server and 20 client mismatches assert literal `integrity_failure`. `crossns` and `fetchns` run the `requested_manifest_object_record` cells alone and are killed. Raw blob requests and RPC 3/4 vocabulary are explicit bounds below. |
| Blob and chunk classes: `Index.AddBlob` / `Index.AddJSON` | complete byte array vs Chunk; digest, size, offset/range, chunk digest; Descriptor namespace | Descriptor positive and malformed-reference negatives; chunk transfer/recovery bound to §11.5. |
| Validated real-byte exchange: `MissingObjectIDs` / `FetchObjects` / `Index.AddJSON` / `Index.AddBlob` | roots changed `[record,manifest,blob]`; fetched IDs `[exact Checkpoint,exact Descriptor]`; namespace `[record/manifest]`; Tombstone/Ack `[both retained]`; timestamp `[no winner]`; blob admission `[after record-root equality]` | `TestInProcessExchangeWalksChangedNamespacesAndFetchesExactObjects` pins six roots derived from real identities; `exchange-objects-get-schema-membership` is killed by that test alone. Rule 2 replay and rule 3 quarantine are separately driven by `TestAddJSONQuarantinesSameIDDifferentStoredBytes`; Tombstone/Ack union by `TestInProcessTombstoneUnionRetainsBothTimesAndAcknowledgements`. `TASK-260830-147hsj` adds durable missing/common-ID sync and branch projection. §11.5 chunk transfer belongs to `STORY-260830-14qxuc`. |

## Direct importers and moved classes

`IMPORTER-OUTCOMES.md` records the exact base/candidate importer names from
fresh `go list -json ./...` snapshots at checkpoint
`0ca3e4c26e2b275212796657f785b9b450f6174e` and the rev3 candidate, each entry
and input class, importer-package tests, and moved classes. The base graph has
44 package paths and the candidate has 45. The complete direct importer sets
are 22 → 23 for `canonicaljson`, 3 → 4 for `rpcwire`, and 5 → 5 for
`provhost`; the 27-package candidate importer test set adds only
`internal/merkleinventory` to the 26-package base set. `provhost` remains
test-only source ownership evidence. No `rpcwire` production source or
validator was modified. The rev3 importer rerun artifacts and exact package
lists are recorded in `IMPORTER-OUTCOMES.md` and the task evidence archive.

Moved validator classes: complete Tombstone and Tombstone Acknowledgement
shapes move from explicit fail-closed refusal to canonicaljson identity
admission; their schema namespace remains unchanged. The four named schema
rows remain mapped but fail-closed under orchestrator decision rev1. No
traceability registry row was edited; the Story final leaf owns that registry.

## Measured acceptance ratio

**4 of 4 producer-defined property rows have named production-entry evidence**:
deterministic trie, total/disjoint membership with owner-bounded fail-closed
rows, bounded serving, and the split identity-level plus real-byte union
exchange. The four incomplete schema rows are admitted 0 of 4 by decision and
each has a refusal test, a typed literal code, unchanged roots/counts, no
quarantine, and a targeted narrowing. `CONFORMANCE-MATRIX.md` lists every
out-of-contract row with its acceptance clause and owner/bound.

## TASK-260830-147hsj union and projection bindings

Registry acceptance case: `story-260830-147hsj-durable-union-projection`.

Authority: pinned `internal/specdoc/SPEC.v0.7.0.md`, §11.4 rules 1–7
(lines 7925–7933) and §5.3 (lines 1987–2061, including line 2047). The
normative sentence is: “No last-writer-wins rule exists. Timestamps MUST NOT
select a winner.” The task record's v0.5.0 reference is stale. The selector
uses the greatest `(epoch, lease_id)` tuple in bytewise UUID order; `created_at`
is diagnostic only.

| Acceptance row | Production call site | Named evidence | Narrowing |
| --- | --- | --- | --- |
| Missing-object discovery and digest fetch | `DurableIndex.SyncFrom` → `MissingObjectIDs` / `walkMissing` → `FetchObjects` → `Index.ObjectsGet` | `TestDurableSyncFindsMissingJSONObjectsAndRetainsTombstoneRecords` verifies exact bytes and roots for each of `[event,manifest,record,tombstone,tombstone_ack]` | `sync-missing-record-fetch` |
| Durable validated add and restart idempotency | `DurableIndex.AddJSON` → `ClassifyJSON` → `durableInstallNoReplace`; `OpenDurable` → `load` | `TestDurableJSONAddIsIdempotentAcrossCrashRestartByNamespace` covers all five namespaces; `TestDurableAddRejectsBeforePersistingInvalidObjects`; `TestDurableStoreRejectsCorruptActiveBytesOnOpen` | `durable-event-replay`, `durable-unknown-schema-admit`, `durable-load-skips-forged-object` |
| Same-digest/different-byte quarantine and sync abort | `SyncFrom` common-ID audit → `addJSONLocked` → persistent quarantine/removal | `TestDurableSyncAuditsCommonIDsAndQuarantinesDifferentBytes`; `TestDurableSyncConflictWithPartialOverlapPeer` reproduces a conflicting common identity alongside peer-only records before and after a successful sync; `TestDurableConflictCrashRestoresQuarantineAndAbortsSync` covers crashes after first candidate, both candidates, and active removal | `durable-same-id-different-bytes`, `sync-common-record-audit`, `sync-common-audit-partial-overlap-peer`, `sync-common-audit-only-without-missing`, `sync-common-audit-first-namespace-only`, `sync-common-audit-first-sync-only`, `quarantine-crash-recovery` |
| Tombstone/Acknowledgement union and closure | `SyncFrom` → `addJSONLocked` → `validateTombstoneAckClosureLocked`; `Index.RebuildProjection` → `validateTombstoneAckClosure` | `TestDurableSyncFindsMissingJSONObjectsAndRetainsTombstoneRecords`; `TestDurableSyncRefusesUnclosedAcknowledgementThenRecovers`; `TestDurableSyncRefusesAcknowledgementWithMismatchedTombstoneSubject`; `TestProjectionRebuildRefusesUnclosedAcknowledgement` | `sync-skip-tombstone-union`, `union-ack-arrival-order`, `sync-unclosed-tombstone-ack`, `durable-ack-subject-link`, `projection-skip-unclosed-ack` |
| Complete lease tuple union; preserve a losing branch | `Index.RebuildProjection` → `Reader.LeaseHeadsForSession` → `sessstate.Projector.Project` | `TestLeaseHeadsForSessionReturnsEveryTupleAcrossTimestampPerturbation`; `TestLeaseHeadsForSessionRefusesConflictingBytesForOneLeaseID`; `TestProjectionRebuildExhaustsUnionArrivalsWithoutTimestampAuthority` | `union-lease-tuple-omission`, `union-lease-created-at-winner`, `union-lease-conflicting-same-id`, `projection-drops-union-head` |
| Rebuild only from a closed union with exact repository authority bytes | `Index.RebuildProjection` / `DurableIndex.RebuildProjection` → `sessrepo.GetRecord`, `ListEvents`, `GetEvent` → `sessstate.Projector.Project` | `TestProjectionRebuildRefusesAuthorityBytesMissingFromUnion` and `TestProjectionRebuildExhaustsUnionArrivalsWithoutTimestampAuthority` | `projection-missing-union-record`, `projection-missing-union-event` |
| Arrival-order and timestamp invariance | Same rebuild entries | Every permutation of six union additions (`6! = 720`) under two opposing timestamp assignments yields 1,440 production rebuilds; the oracle asserts literal projection state, lease tuple and conflict kinds plus expected union IDs/bytes/counts/roots. The durable wrapper is called once per timestamp assignment. | `union-lease-created-at-winner`, `projection-drops-union-head` |

The six-object permutation set contains two competing same-epoch Lease
Records, an authoritative `session.created` and divergent `session.idle` event,
a Tombstone, and its Acknowledgement. The Session Record is the fixed authority
precondition; it is present in every union. In the two timestamp profiles each
`created_at` field is moved in opposite directions. The reducer receives the
fixed repository-owned chain only after each record/event byte sequence is
found exactly in the union. `LeaseHeadsForSession` consumes sorted identity
order and drops `created_at`; the rebuild path has no wall-clock input. Thus the
measured pure-function bound is a fixed `sessrepo` authority snapshot plus an
immutable union set. This leaf does not reconstruct a second event DAG from
arbitrary divergent event objects.

**Measured acceptance ratio: 6 of 6 derived acceptance rows** are driven
through named production call sites. The producer brief has no formal surface
table; `coverage-map.md` calls this out and separately includes the four rev3
carry-over arms and the additional closure/loader gates.

Out-of-contract rows and their acceptance clauses are listed in
`CONFORMANCE-MATRIX.md`: §§11.5–11.6 staging/materialization/commit are modeled
at the boundary only; public network/Host Channel/doctor integration is not
defined by this library task; arbitrary DAG reconstruction is excluded by the
existing `sessrepo` ownership instruction; physical power loss and native
Windows crash simulation are beyond process-crash tests; and full digest/UUID
spaces remain owned by `canonicaljson`/`scalar`.

## TASK-260830-2h5uv9 generated sync convergence

Registry acceptance case: `story-260830-2h5uv9-order-duplicate-gap-convergence`.
Authority: pinned `internal/specdoc/SPEC.v0.7.0.md`, §11.4 rules 1–6 and 7,
§5.3 clauses 5.3#6–#7 for lease tuple choice and losing-branch preservation,
and §10.7 clause 10.7#13 for immutable Tombstone/Acknowledgement exchange.
The central production call site is `DurableIndex.SyncFrom`; successful
projection is checked through `DurableIndex.RebuildProjection`.

| Acceptance behavior | Production entry | Named test | Isolated narrowing |
| --- | --- | --- | --- |
| Reorder × duplicate × gap × two-way skew × partial peer × 1–3 sync-pass generated product; independent valid-union oracle | `DurableIndex.SyncFrom` → `MissingObjectIDs` / `FetchObjects`; `RebuildProjection` | `TestDurableSyncGeneratedPerturbationProduct` | `sync-missing-record-fetch`, `sync-repeat-record-quarantine-write`, `union-lease-created-at-winner`, `sync-partial-peer-regresses-local-root` |
| Every arrival permutation for N≤5; deterministic sampling at N=6 | `DurableIndex.SyncFrom` | `TestDurableSyncArrivalOrderConverges` | `sync-missing-record-fetch` |
| A missing object is fetched after the peer fills the gap | `DurableIndex.SyncFrom` | `TestDurableSyncGapFillsOnLaterPass` | `sync-missing-record-fetch` |
| Opposing `created_at` values do not select the lease winner | `DurableIndex.SyncFrom` → `RebuildProjection` | `TestDurableSyncClockSkewDoesNotSelectLeaseWinner` | `union-lease-created-at-winner`, `projection-drops-union-head` |
| A peer with a proper object subset cannot remove local IDs or lower roots | `DurableIndex.SyncFrom` | `TestDurableSyncPartialPeerNeverRegressesLocalRoots` | `sync-partial-peer-regresses-local-root` |
| Duplicate delivery and post-convergence sync preserve exact durable bytes | `DurableIndex.SyncFrom` | `TestDurableSyncDuplicateDeliveryIsByteIdentical`, `TestDurableSyncGeneratedPerturbationProduct` | `sync-repeat-record-quarantine-write` |
| Same digest with different bytes quarantines and aborts even when the peer has missing identities, after a successful prior sync, or on any namespace | `DurableIndex.SyncFrom` → common-ID audit and quarantine | `TestDurableSyncConflictWithPartialOverlapPeer`, `TestDurableSyncGeneratedPerturbationProduct`, `TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord`, `TestDurableSyncSameDigestDifferentBytesIsTypedConflict` | `sync-common-audit-partial-overlap-peer`, `sync-common-audit-only-without-missing`, `sync-common-audit-first-namespace-only`, `sync-common-audit-first-sync-only`, `sync-common-audit-first-id-only`, `sync-common-audit-last-id-only`, `sync-common-audit-even-ids-only` |
| Common-ID audit has no conditional gate on missing IDs, peer size, namespace position, or common-ID position | `DurableIndex.SyncFrom` AST call site; runtime refusal stays covered by the named regression | `TestDurableSyncCommonIDAuditIsUnconditionalInAST`, `TestDurableSyncConflictWithPartialOverlapPeer`, `TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord` | `sync-common-audit-ast-control`, `sync-common-audit-early-namespace-return`, `sync-common-audit-first-id-only`, `sync-common-audit-last-id-only`, `sync-common-audit-even-ids-only` |
| Peer schema/namespace mismatch refuses before installing the bad identity | `DurableIndex.SyncFrom` → `Index.ObjectsGet` | `TestDurableSyncRefusesPeerNamespaceMismatch` | `sync-peer-namespace-mismatch` |
| Unclosed acknowledgement is a typed conflict and recovers after its Tombstone arrives | `DurableIndex.SyncFrom` → union closure | `TestDurableSyncUnclosedAcknowledgementAborts` | `sync-unclosed-tombstone-ack` |

The generator executes 38,000 valid-union cases plus 37,765 same-identity /
different-byte conflict cases. It enumerates all object arrival permutations
for N=1..5 and 34 deterministic unique orders for N=6. It crosses two
timestamp assignments, duplicate off/on, gap absent/present, and every proper
subset of non-gap peer identities. A gap fills on pass 2; passes 1, 2, and 3
are asserted as prefixes. Each six-object set contains two competing lease
tuples, a losing-lease event branch, a Tombstone and matching Acknowledgement.
Every feasible conflict-target × conflict-round pair (rounds 1, 2, 3) is
generated at N=2..6 and crossed with those same perturbation axes; the peer
also contains an identity absent locally. The reference computes namespace
roots and projected winner/loser state in test code without calling the
production Merkle builder or lease comparator. Completed cases repeat
`SyncFrom` and compare the durable directory byte for byte. The named
partial-overlap regression exercises the reviewer's case before and after a
successful prior sync. The configured suite always runs a bounded N=1..3
valid-union shard, plus 432 generated conflict cases in
`TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord`: all
three sorted common-ID positions, rounds 1..3, both skew directions, all six
arrival orders, both duplicate modes, and each of two peer-only records. The
large N=1..6 product remains available through explicit nested selectors.

The task brief supplied no surface table; the table above is its measured
decomposition and that brief gap is recorded in `coverage-map.md`. The written
acceptance criterion is **1 of 1 rows driven** through the production
`DurableIndex.SyncFrom` entry; the ten behavior rows above make that sentence
auditable. Out-of-contract clauses are listed in `CONFORMANCE-MATRIX.md`.
