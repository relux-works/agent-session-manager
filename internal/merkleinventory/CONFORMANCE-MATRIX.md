# Mesh RPC 2 inventory conformance matrix

The rows below preserve the first-leaf inventory census. The
`TASK-260830-147hsj` section records durable object union, lease heads, and
projection rebuild; the final `TASK-260830-2h5uv9` section adds generated sync
convergence evidence and the Story registry bindings.

Authority: `internal/specdoc/SPEC.v0.7.0.md` §10, §11.3, §11.4, and §11.5.
The task record's v0.5.0 reference is stale; the producer instruction pins the
v0.7.0 text. Base checkpoint: `0ca3e4c26e2b275212796657f785b9b450f6174e`.
Candidate remains uncommitted in the managed Story worktree. No network is
used. The generated property uses independent per-case durable stores under
Go's `t.TempDir`; it perturbs stored timestamps without consulting wall clock.

## Producer property results

| Property | Result | Evidence and remaining bound |
| --- | --- | --- |
| Deterministic trie | Driven | `TestNormativeRootAndChildFixtures`, `TestDispatchServesNormativeChildrenBody`, `TestMixedNS1RootsFromNormativeSyntheticIDs`, and rule 4 depth sweep drive `Index.Root`, `Index.Child`, `ComputeNode`, and `Index.Dispatch`; rule 1–5, child-label-order, and MIXED-NS-1 root-member narrowings run alone and are killed. |
| Total, disjoint membership | Driven with four explicit fail-closed bounds | The 19 included rows and 42 exclusions have a closed mapping test, Descriptor/Chunk/local-marker negatives, and cross-namespace `objects.get` refusal. Each of the four mapped schemas without a complete canonicaljson validator has its own typed-refusal, no-root-change, no-quarantine test and isolated admitting narrowing; validator ownership is named below. No unsupported shape is admitted. |
| Bounded serving | Driven | `Index.Dispatch` composes `rpcwire.DecodeRequest` / `EncodeSuccess`; namespace, prefix, response, and negotiated-line axes have named tests and narrowings. `FetchObjects` sends singleton batches under the line limit. |
| Union exchange | Split identity-level and validated-byte proofs driven | `TestMixedNSExchangeIdentityLevelSyntheticIDs` reproduces the exact MIXED-NS-1 roots through `inventory.roots`, identifies only the normative missing IDs with the recursive identity walk and `inventory.children`, then converges all six roots. It claims no object validation. `TestInProcessExchangeWalksChangedNamespacesAndFetchesExactObjects` separately proves validated real-byte retrieval/admission and pins six literal roots derived from those bytes. Tombstone/Acknowledgement union, identical replay, conflict quarantine, and timestamp non-selection have their own production-entry tests. The two rows are kept separate because §11.4 says the synthetic IDs stand for schema-valid objects/bytes and their contents do not feed the trie. |
| Strict closed wire decoding (rev3 rework) | Driven across the package's wire decode sites | The decoder census below covers every direct decode/read site and delegates outer RPC envelopes to `rpcwire`. Every member of request bodies, outer request envelopes, success and failure response envelopes, objects bodies and WireObject items has casefold/missing/extra/duplicate/six-kind cells through production entry points. Response array item kinds and singleton cardinality are also tested. `TestWireDecoderASTGuardRejectsAliasedUnmarshalBindings` rejects direct or aliased `encoding/json` bypasses; `guard-aliased-json-unmarshal`, `server-casefold-prefix`, `client-casefold-wireobject`, `client-two-wireobjects`, and `client-fetch-cbor-label` are killed alone. |

## Gate × production entry × axis

| Gate | Production entry and owned check | Enumerated axes | Named evidence, narrowing, or bound |
| --- | --- | --- | --- |
| RPC version and namespace request envelope | `Index.New`; `Index.Dispatch` → `rpcwire.DecodeRequest` / `EncodeSuccess` | RPC `[2.0.0,3.0.0,4.0.0,5.0.0,6.0.0]`; namespace `[blob,event,manifest,record,tombstone,tombstone_ack,unknown]`; cardinality `[0,1,6,7]`; order `[sorted,descending,duplicate]` | `TestNewRefusesHigherRPCNamespaceSets`; `TestDispatchServesAllSixInventoryRootsInRPCWireOrder`; `rpcwire.TestInventoryNamespacesAndCardinality`. `rpc2-version-admission` and delegated rpcwire narrowings `namespace-empty`, `namespace-member`, `namespace-duplicate`, `namespace-sort`, `roots-missing`, `root-association`, `namespace-admits-six-letter` are isolated and killed. |
| Trie set/count rules and exact JCS nodes | `Index.Root` / `Index.Child` → `nodeFor` → `ComputeNode` → `buildNode` | Member count `[0,1,2,5,16]`; duplicate ID `[absent,present]`; input order `[ascending,unsorted]`; shared prefix `[0..63]`; full depth `[64]`; children `[0..16]` | Empty/singleton/branch literals and MIXED-NS-1 roots: `TestNormativeRootAndChildFixtures`, `TestRuleFourRetainsSingleChildAtEverySharedPrefix`, `TestMixedNS1RootsFromNormativeSyntheticIDs`. Rule narrowings `rule1-duplicate-set-member` through `rule5-depth64-duplicate` each run alone and are killed; `mixed-ns1-root-member-narrowing` is killed by the exact six-root test. Count>1 at depth 64 returns literal `integrity_failure`. |
| Root and children response bodies | `Index.Dispatch` → `rootsBody` / `childrenBody` → `rpcwire.EncodeSuccess` | Namespace order `[blob,event,manifest,record,tombstone,tombstone_ack]`; root count `[0,1,2,5,16]`; child labels `[0..9,a..f]`; leaf IDs `[0,1]` | `TestDispatchServesAllSixInventoryRootsInRPCWireOrder`, `TestDispatchServesNormativeChildrenBody`, and `TestDispatchBuildsSixteenSortedNibbleChildren` pin exact root/child JSON bodies. `rule2-empty-root-count`, `rule3-root-singleton-leaf`, and `children-label-order` target exact served root/singleton/branch bodies. Request validation is owned by rpcwire; no validator is copied here. |
| Strict request-body JSON types | `Index.Dispatch` for `inventory.roots` / `inventory.children`; `Index.ObjectsGet` for `objects.get`; array shape is delegated to `rpcwire.DecodeRequest` where it owns it | Each field `[namespaces,namespace,prefix,object_ids,encodings]` × JSON kind `[string,number,boolean,null,array,object]`; array members by the same six kinds; field state `[present,missing,duplicate]` | `TestDispatchRequestShapeStrictTypes` names every cell, drives each real entry, asserts no response body and the literal local refusal `invalid inventory request`. `children-prefix-type-null` narrows the string-kind gate to accept null and is killed by `inventory_children_prefix_null`. `objects.get` uses `Index.ObjectsGet` because the operation carries its namespace as call context, not a request-body field. |
| Prefix grammar and missing-node refusal | `Index.Dispatch` → `childrenBody` → `nodeFor` → `validPrefix` | Lowercase-hex lengths `[0..65]`; lengths `[1..64]` each materialized and absent; uppercase and non-hex lengths `[1..65]`; root `[0]` | `TestDispatchPrefixAxisCoversEveryLengthAndAlphabet` drives all cells through `Dispatch`; valid absent nodes at every length 1–64 assert literal `not_found`; lengths above 64 and invalid alphabet assert the literal local refusal. `emptychild2` is killed by the length-2 absent cell; prior `prefix-length-65`, `uppercase-prefix`, and `missing-prefix-not-found` plants remain. |
| Schema and byte class membership | `Index.AddJSON` → `ClassifyJSON` → `canonicaljson.VerifyObjectIdentity`; `Index.AddBlob` → Descriptor check | All included schema/version pairs and excluded classes enumerated below; raw bytes `[complete,chunk]`; schema `[known,unknown,missing]`; local marker `[present,absent]` | `TestRPC2SchemaMembershipTableIsTotalAndDisjoint`, `TestMixedNSN1RejectsDescriptorRelabelingChunksAndLocalMarker`, `TestBlobDescriptorOwnsManifestMembershipAndCompleteRawBlob`; the N1 test pins all six valid roots and proves Descriptor-as-record, independent Chunk enumeration, and local-marker inclusion change the `record`/`manifest`/`blob` roots while schema admission rejects each class error. `descriptor-namespace`, `unknown-schema-fail-closed`, `excluded-local-marker`, and `blob-chunk-content-mismatch` are isolated narrowings. Four mapped rows fail closed at canonicaljson and remain partial. |
| Incomplete schema-validator refusal | `Index.AddJSON` → `ClassifyJSON` → `canonicaljson.VerifyObjectIdentity` | `session-record@3.1.0`; `materialization-plan@1.0.0`; `materialization-plan@2.0.0`; `task-board-bundle@1.0.0` | Four named tests assert literal `integrity_failure`, unchanged six roots/counts, and no quarantine. `unsupported-schema-session-record-31`, `unsupported-schema-materialization-plan-10`, `unsupported-schema-materialization-plan-20`, and `unsupported-schema-task-board-bundle-10` each admit exactly its row and are killed by that row's test. The schema-to-namespace mapping remains total while admission waits for the owner validator. |
| Identity and requested namespace membership | `ClassifyJSON`; `Index.ObjectsGet`; `FetchObjects` → `decodeOneWireObject` | Server requested namespace × true JSON-object namespace `[6×5]` (25 mismatch cells, 5 diagonal controls); client namespace × true object namespace `[5×5]` (20 mismatch cells, 5 diagonal controls); hostile source `[honest,foreign]` | `TestObjectsGetRefusesEveryForeignNamespacePair` asserts no body and literal `integrity_failure` for every server mismatch; `TestFetchObjectsRefusesEveryHostileForeignNamespacePair` does the same for every client mismatch. `crossns` and `fetchns` each narrow one manifest/record cell and are killed by that named cell. `FetchObjects` refuses the `blob` namespace at its API boundary; §11.5 owns raw-blob transfer. RPC 3/4 are not negotiated by this constructor, so their namespace rows are bounded by `Index.New` refusing versions above 2.0.0. |
| Blob membership and integrity | `Index.AddBlob` → `verifyDescriptorBytes` | Descriptor namespace `[manifest,other]`; raw digest `[match,mismatch]`; size `[match,mismatch]`; chunk ranges `[inside,outside]`; chunk digest `[match,mismatch]` | `TestBlobDescriptorOwnsManifestMembershipAndCompleteRawBlob`, `TestBlobDescriptorByteReferencesAreVerifiedBeforeAdmission`, `TestMixedNSN1RejectsDescriptorRelabelingChunksAndLocalMarker`. Chunks have no root. Actual `chunks.put`, staging, restart, and recovery are bounded to the §11.5 transfer owner. |
| `objects.get` request shape and pagination | `Index.ObjectsGet` → `decodeObjectsGetRequest`; `FetchObjects` | ID count `[0,1,2,4096,4097]`; ID order `[ascending,descending]`; duplicate `[absent,present]`; encoding count `[0,1,2]`; encoding `[JSON,CBOR,duplicate,unsorted]`; exact field-type/missing/duplicate matrix above | `TestObjectsGetRefusesUnsortedDuplicateAndUnavailableEncodingRequests`, `TestObjectsGetRefusesRequestIDCountOutsideBound`, and `TestDispatchRequestShapeStrictTypes`; `objects-get-request-count`, `objects-get-duplicate-id`, and `objects-get-cbor-only` are killed. 4096 sorted IDs pass shape and return literal `not_found` on an empty index. RPC 2 supports JSON only. |
| Negotiated response bytes and bounded batches | `Index.ObjectsGet`; `FetchObjects` | Response line `[at limit,one byte over]`; batch width `[1,2]`; line limit `[128,measured singleton response]` | `TestObjectsGetValidatesRequestedNamespaceAndNegotiatedLineLimit`, `TestFetchObjectsUsesBoundedSingletonBatches`; `objects-get-line-limit` and `objects-get-unbounded-batch` are killed. |
| Identity-level MIXED-NS-EXCHANGE | `Index.Dispatch` → `inventory.roots` / `inventory.children`; `MissingNamespaceIDs` → `walkMissing` | Namespace `[blob,event,manifest,record,tombstone,tombstone_ack]`; B missing `[Z(3),T(2),sha256:555…555]`; changed roots `[record,manifest,blob]`; post-add root set `[all six MIXED-NS-1 roots]` | `TestMixedNSExchangeIdentityLevelSyntheticIDs` seeds only the normative identity sets, uses the production RPC roots and child serving entries plus the production recursive walk, and proves the exact missing set. `identity-walk-blob-narrowing` and `objects-get-blob-transfer-boundary` are isolated and killed. This row explicitly claims no object validation. |
| Validated real-byte exchange and immutable Tombstone/Acknowledgement union | `MissingObjectIDs` → `walkMissing`; `FetchObjects` → `ObjectSource.ObjectsGet`; `Index.AddJSON` / `Index.AddBlob` | Changed roots `[record,manifest,blob]`; retrieved IDs `[exact Checkpoint,exact Descriptor]`; stored/requested membership `[record/record,manifest/record]`; link sets `[same Tombstone/Ack,two Tombstones/two Acks]`; timestamps `[different,not winner-selected]`; blob admission `[before,after record-root equality]` | `TestInProcessExchangeWalksChangedNamespacesAndFetchesExactObjects` pins six roots from real identities and checks validated object responses; `exchange-objects-get-schema-membership` is killed by that test alone. `TestInProcessTombstoneUnionRetainsBothTimesAndAcknowledgements` asserts both differently timestamped immutable Tombstones and both Acknowledgements remain; `tombstone-timestamp-winner` narrows that union to the later timestamp and is killed by that test alone. `TestAddJSONQuarantinesSameIDDifferentStoredBytes` and `same-id-quarantine` cover rule 3; `TestClassifyRefusesIdentityDigestMismatch` covers rule 1. Blob bytes are admitted only after record-root equality in the in-process caller. `TASK-260830-147hsj` adds durable sync and projection as detailed in the task delta section. |
| Source ownership of identity attestation | `internal/provhost.TestNoProductionPathAttestsProviderIdentityBinding` | Production `VerifyObjectIdentity` sites `[5 sessrepo,1 merkleinventory]`; sibling/foreign path `[absent]`; searched token `[present,bypassed]` | Exact owner path census and synthetic foreign path test; `identity-inventory-owner-prefix` is killed by the source test, and `identity-token-preserving-refusal-bypass` preserves the searched token while the behavioral suite rejects forged membership. This is a source test, not a production capability. |

## Wire decoder census (rev3)

This census was taken from every `encoding/json` decode, raw member lookup,
`Decode` call, and `raw*` accessor in `internal/merkleinventory` production
files. `json.Marshal` calls only serialize locally constructed output/request
values. `strict_json.go:47` is the only package `json.Unmarshal`; the AST
guard rejects any other call, decoder alias, `json.NewDecoder`, dot import, or
`raw*` accessor. No production wire path decodes JSON into a struct.

| Production site | Shape read | Decoder / shape enforcement | Production-entry evidence |
| --- | --- | --- | --- |
| `serve.go:21` `Index.Dispatch` | Outer request envelope for `inventory.roots`, `inventory.children`, `objects.get` | `rpcwire.DecodeRequest` owns exact RPC members, value kinds, request correlation fields, and operation-body byte framing; body-specific exact member/type checks follow in this package. | `TestDispatchRejectsEveryNonExactOuterRequestEnvelopeShape` drives `Dispatch`; delegated rpcwire envelope tests are listed in the RPC row above. |
| `serve.go:55-63` `rootsBody` | `inventory.roots` body: exactly `namespaces: string[]` | `strictObject` → `exactMembers` → `requiredStringArray`; `rpcwire.DecodeRequest` also owns supported namespace vocabulary, sorted uniqueness and 1..6 cardinality. | `TestDispatchRejectsEveryNonExactOperationBodyShape` and `TestDispatchRequestShapeStrictTypes`, both through `Index.Dispatch`. |
| `serve.go:80-90` `childrenBody` | `inventory.children` body: exactly `namespace: string`, `prefix: string` | Same strict decoder; exact keys and types before namespace/prefix grammar validation. | `TestDispatchRejectsEveryNonExactOperationBodyShape` and `TestDispatchPrefixAxisCoversEveryLengthAndAlphabet`, through `Index.Dispatch`; `server-casefold-prefix` kills alone. |
| `serve.go:283-287` `FetchObjects` | Locally encoded outbound `objects.get` request correlation and the returned outer RPC success/error envelopes | Outbound request bytes are produced by `json.Marshal` and `rpcwire.EncodeRequest`; the generated request and response are parsed by `rpcwire.DecodeRequest` / `rpcwire.DecodeResponse`, which own their closed RPC shapes. | `TestFetchObjectsRejectsEveryNonExactSuccessWireShape` exercises every member variant for both success and failure envelopes through a hostile raw source; `TestFetchObjectsRefusesWellFormedRPCFailureEnvelope` pins the valid failure variant. `rpcwire` remains unmodified and owns nested Structured Error validation. |
| `serve.go:301-316` `decodeOneWireObject` | Response body `objects` array and one exact WireObject `{object_id,media_type,encoding,data}` | `strictObject`, `strictArray`, `exactMembers`, and `requiredJSONString`; then strict base64url and `ClassifyJSON` validation. Array items must be objects and the result must contain exactly one item because `FetchObjects` sends one ID per request. | `TestFetchObjectsRejectsEveryNonExactSuccessWireShape`, `TestFetchObjectsRejectsNonObjectItemsAndWrongArrayCardinality`, and `TestFetchObjectsRefusesEveryHostileForeignNamespacePair`, through hostile `ObjectSource`; `client-casefold-wireobject` and `client-two-wireobjects` kill alone. |
| `index.go:80-105` `ClassifyJSON` | Immutable JSON top-level members `schema` and `schema_version` before schema identity validation | `strictObject`, `strictJSONString`, `requiredJSONString`; `canonicaljson.VerifyObjectIdentity` owns complete schema shape, canonical digest and exact schema identity validation. | Membership, N1, unknown-class and fail-closed schema tests exercise `Index.AddJSON` / `ClassifyJSON`; identity and membership mutants are listed above. |
| `index.go:137-159` `validateTombstoneAckLinkLocked` | Validated Tombstone Acknowledgement `subject_id`, `tombstone_id`; referenced Tombstone `subject_id` | Complete closed-shape validation is delegated to `canonicaljson`; link fields are re-read with `strictObject` / `requiredJSONString`. | `TestAddTombstoneAcknowledgementRequiresAStoredMatchingTombstone` drives missing/mismatched references through `Index.AddJSON`; tombstone link narrowings are run alone. |
| `index.go:174-186`, `264-298` `AddBlob` / `verifyDescriptorBytes` | Validated Blob Descriptor `blob_id`, `size`, `chunks`; each BlobChunk `{chunk_id,index,offset,size}` | Complete descriptor schema shape is delegated to `canonicaljson`; cross-byte fields use `strictObject`, `strictArray`, `exactMembers`, `requiredJSONString`, and `requiredUint53`. | Descriptor and malformed-reference tests drive `Index.AddBlob`; `descriptor-namespace` and `blob-chunk-content-mismatch` are isolated plants. |
| `serve.go:403-429` `decodeObjectsGetRequest` | `objects.get` body: exactly `object_ids: string[]`, `encodings: string[]` | `strictObject`, `exactMembers`, `requiredStringArray`, then digest/order/cardinality and encoding validation. | `TestDispatchRejectsEveryNonExactOperationBodyShape`, `TestObjectsGetRefusesUnsortedDuplicateAndUnavailableEncodingRequests`, and `TestObjectsGetRefusesRequestIDCountOutsideBound`; all request-body cases are sent through `Index.Dispatch`. |
| `strict_json.go:19-50` `strictJSON` | The typed JSON fragments used by all direct readers above | Calls `canonicaljson.Canonicalize` first (including duplicate-member refusal), restricts destinations to RawMessage maps/arrays, string, or `scalar.Uint53`, then performs its sole allowlisted `json.Unmarshal`. | `TestStrictJSONRejectsStructDestinations`, `TestWireDecoderASTGuardRejectsAliasedUnmarshalBindings`, `TestWireDecoderGuardScansProductionAliases`; the guard's aliased `Unmarshal` control plant is `guard-aliased-json-unmarshal`. |

RPC request/response envelopes use the already-landed `internal/rpcwire`
validators and are not reimplemented here. RPC errors are returned as refusal
errors by `FetchObjects`; the success-envelope and hostile-body paths are the
only response shapes this API admits as an object result. The package's local
typed response error is `invalid inventory object`; server body-shape errors
are `invalid inventory request`, with no success body.

### Included schema/version rows

These are the 19 rows of the RPC 2 inventory mapping table, grouped by one
namespace. The byte classes `raw blob` → `blob`, `Chunk` → excluded, and local
or transient envelopes → excluded are separate from the JSON rows.

| Namespace | Schema/version rows | Admission state |
| --- | --- | --- |
| `record` | `session-record@1.0.0`, `@2.0.0`, `@3.0.0`, `@3.1.0`; `lease@1.0.0`; `checkpoint@1.0.0`; `workspace-group@1.0.0`; `provider-identity@1.0.0` | `session-record@3.1.0` is mapped but fail-closed pending its owner validator; other rows have complete validators. |
| `event` | `session-event@1.0.0`, `@2.0.0`, `@3.0.0`, `@4.0.0` | Complete validators. |
| `manifest` | `transfer-manifest@1.0.0`; `blob@1.0.0`; `materialization-plan@1.0.0`, `@2.0.0`; `task-board-bundle@1.0.0` | Both Materialization Plan rows and Task-board Bundle are mapped but fail-closed pending their owner validators. |
| `tombstone` | `tombstone@1.0.0` | Complete Section 10.7 shape; object admission also requires matching content digest. |
| `tombstone_ack` | `tombstone-ack@1.0.0` | Complete Section 10.7 shape; `Index.AddJSON` also requires an already admitted Tombstone with the same `subject_id`. |

### Fail-closed schema rows and validator owners

The orchestrator decision rev1 says: “KEEP FAIL-CLOSED. Do not add shallow
validators.” Each row remains mapped to exactly one Section 11.4 namespace;
admission refuses until its owning Story supplies a complete shape validator.

| Schema/version | Namespace mapping | Refusal test | Isolated admitting mutant | Validator owner from task board |
| --- | --- | --- | --- | --- |
| `session-record@3.1.0` | `record` | `TestUnsupportedSessionRecord31RemainsFailClosed` | `unsupported-schema-session-record-31` | `STORY-260830-4n0fo8 — authoritative-record-schema-core` |
| `materialization-plan@1.0.0` | `manifest` | `TestUnsupportedMaterializationPlan10RemainsFailClosed` | `unsupported-schema-materialization-plan-10` | `STORY-260830-2r137i — workspace-materialization-and-groups` |
| `materialization-plan@2.0.0` | `manifest` | `TestUnsupportedMaterializationPlan20RemainsFailClosed` | `unsupported-schema-materialization-plan-20` | `STORY-260830-2r137i — workspace-materialization-and-groups` |
| `task-board-bundle@1.0.0` | `manifest` | `TestUnsupportedTaskBoardBundle10RemainsFailClosed` | `unsupported-schema-task-board-bundle-10` | `STORY-260830-27pqyi — task-board-bridge-and-bundle` |

### Excluded schema classes

The 42 known excluded schema identities are checked individually by
`TestRPC2SchemaMembershipTableIsTotalAndDisjoint`; none overlaps an included
row:

```text
config, provider-manifest, provider-probe, terminal-backend-manifest,
terminal-backend-probe, chunk, canonical-event, materialization-journal,
terminal-instance-binding, session-adapter-manifest, session-adapter-probe,
session-directory-node-manifest, session-directory-node-request,
session-directory-node-response, host-trust-store, launch-plan-request,
error, observation, cli-result, session-clone-bundle,
clone-raw-object-manifest, clone-capture-manifest, canonical-session,
migration-checkpoint, fidelity-report, projection-plan,
clone-projected-object-manifest, clone-read-back-evidence-manifest,
clone-validation-report, clone-lineage-receipt, supported-environment-tuples,
environment-observation, native-session-observation, session-inventory-batch,
conversation-lineage-link, session-annotation, session-enrichment-profile,
session-enrichment-job-request, session-enrichment-job-receipt,
session-continuation-plan, session-directory-operation-receipt,
session-directory-query
```

Each name is prefixed `urn:ax:schema:` in the code and test table.

## Mutation and importer evidence

The package harness executes each narrowing alone under a source overlay,
records the subprocess exit and raw JSON test log beside that overlay, and
includes a harmless applied comment control that must survive. The current
candidate run count and every individual verdict are recorded in
`TASK-260830-nxqqaw_results.md` and the evidence archive. The token-preserving
identity-validation bypass keeps `canonicaljson.VerifyObjectIdentity` in the
source while the behavioral suite runs; source-text visibility alone is not a
kill. Each of the four validator-owner rows and the identity-level/validated
exchange entries has its own named test and isolated plant.

Direct importer sets for the changed `canonicaljson`/`provhost` packages, the
inventory package, and its `rpcwire` dependency are listed with each imported
entry, input class, and base/candidate outcome in
[`IMPORTER-OUTCOMES.md`](IMPORTER-OUTCOMES.md). The Story-final registry
bindings are described below and are checked by the traceability regression
test.

## Explicit scope bounds and owners

| Row outside this leaf's production entry points | Acceptance clause / pinned clause | Owner or bound |
| --- | --- | --- |
| Complete validator and admission for `session-record@3.1.0` | Orchestrator decision rev1, Item 1: “KEEP FAIL-CLOSED. Do not add shallow validators.” | `STORY-260830-4n0fo8 — authoritative-record-schema-core`; mapped to `record`, refused and tested here until that owner supplies the complete shape. |
| Complete validators and admission for both Materialization Plan versions | Same decision clause: keep the canonicaljson refusal; do not add a shallow validator. | `STORY-260830-2r137i — workspace-materialization-and-groups`; mapped to `manifest`, each version refused and tested here. |
| Complete validator and admission for Task-board Bundle | Same decision clause: keep the canonicaljson refusal; do not add a shallow validator. | `STORY-260830-27pqyi — task-board-bridge-and-bundle`; mapped to `manifest`, refused and tested here. |
| Chunk staging/retry, transfer recovery and destination materialization | Task AC requires bounded inventory exchange; decision rev1 Item 7 says transfer/materialization owned by another leaf must be modeled at its boundary. Pinned SPEC §§11.5–11.6. | `STORY-260830-14qxuc — resumable-blob-and-manifest-transfer` owns chunk transfer; `STORY-260830-2r137i — workspace-materialization-and-groups` owns workspace materialization. This package only validates/adopts complete raw bytes with `Index.AddBlob`. |
| Lease-head derivation and losing-lease-event preservation (union rules 5–6) | Pinned SPEC §11.4 rules 5–6 and §5.3; implemented by `Index.RebuildProjection` → `Reader.LeaseHeadsForSession` → `sessstate.Projector.Project` | `TestLeaseHeadsForSessionReturnsEveryTupleAcrossTimestampPerturbation` and `TestProjectionRebuildExhaustsUnionArrivalsWithoutTimestampAuthority`; see `TASK-260830-147hsj` delta section for all axes and killed mutants. |
| Crash/restart evidence for durable exchange | Task AC: “crash/idempotency evidence is included when the operation mutates durable state.” | `DurableIndex.AddJSON` persists validated objects and quarantine records. `TestDurableJSONAddIsIdempotentAcrossCrashRestartByNamespace` covers all five JSON namespaces; `TestDurableConflictCrashRestoresQuarantineAndAbortsSync` covers conflict recovery. Process-crash checks do not claim physical power-loss simulation. |
| Mesh RPC 3/4 namespace serving | Pinned SPEC §11.8 adds `directory_record`; §11.9 adds `terminal_backend_evidence`; §11.3 RPC 2 shapes remain six namespaces. | This leaf's `Index.New` accepts RPC 2.0.0 only and composes `rpcwire.Namespaces(version)`; RPC 3/4 inventory serving is not claimed here. |
| Public `ax` doctor/capability or Host Channel integration | Task AC is the internal inventory implementation deliverable. | This package has no Host Channel handler or doctor surface and advertises no anti-entropy capability. |

## Measured acceptance ratio

**4 of 4 producer-defined property rows have named production-entry evidence**:
deterministic trie construction; total, disjoint membership with four explicit
owner-bounded fail-closed schemas; bounded serving; and the split identity-level
plus validated-real-byte exchange. The four incomplete schema rows are
admitted 0 of 4 by the explicit decision and each refuses through
`Index.AddJSON` with a targeted narrowing test. This ratio does not include
chunk transport/materialization, lease/event branch union, durable restart, or
RPC 3/4 serving; each is listed above with its clause and owner/bound.

## TASK-260830-147hsj durable union and projection delta

Authority is the pinned `internal/specdoc/SPEC.v0.7.0.md`, not the task
record's stale v0.5.0. The relevant §11.4 text (rules 1–7, lines 7925–7933)
states:

> No last-writer-wins rule exists. Timestamps MUST NOT select a winner.

Section 5.3 (lines 1987–2061, including line 2047) selects the greatest
`(epoch, lease_id)` tuple using bytewise UUID order; `created_at` is diagnostic
only. A losing lease's events remain in a divergent branch and do not affect
authoritative state.

| Acceptance row | Production call site | Varied axes and evidence | Narrowing killed by named test alone |
| --- | --- | --- | --- |
| Discover absent JSON objects and fetch by digest | `DurableIndex.SyncFrom` → `MissingObjectIDs` / `walkMissing` → `FetchObjects` → `Index.ObjectsGet` | Namespace `[event,manifest,record,tombstone,tombstone_ack]`; one missing valid object per namespace; after sync every exact byte sequence and namespace root matches peer. | `sync-missing-record-fetch` → `TestDurableSyncFindsMissingJSONObjectsAndRetainsTombstoneRecords` |
| Admit validated durable objects, idempotently across crash/restart | `DurableIndex.AddJSON` → `ClassifyJSON` → `durableInstallNoReplace`; reopen via `OpenDurable` → `load` | Validated JSON classes `[5 namespaces]`; malformed/unknown inputs `[invalid JSON, unsupported schema]`; install crash `[after atomic object install]`; identical retry `[before and after reopen]`; count remains 1; corrupt active bytes refuse reopen. | `durable-event-replay` → `TestDurableJSONAddIsIdempotentAcrossCrashRestartByNamespace`; `durable-unknown-schema-admit` → `TestDurableAddRejectsBeforePersistingInvalidObjects`; `durable-load-skips-forged-object` → `TestDurableStoreRejectsCorruptActiveBytesOnOpen` |
| Quarantine same-digest/different-byte conflict and abort sync | `DurableIndex.SyncFrom` common-ID audit → `addJSONLocked` → persistent quarantine/removal | Byte variants `[canonical same identity, exact whitespace-distinct bytes]`; precondition `[equal roots]`; conflict path `[common IDs]`; crash points `[after first candidate, after both candidates, after active removal]`; reopen has no active ID and retry still refuses literal `integrity_failure`. | `durable-same-id-different-bytes`, `sync-common-record-audit` → `TestDurableSyncAuditsCommonIDsAndQuarantinesDifferentBytes`; `quarantine-crash-recovery` → `TestDurableConflictCrashRestoresQuarantineAndAbortsSync` |
| Union Tombstones and Acknowledgements as records, not actions | `DurableIndex.SyncFrom` → `addJSONLocked`; closure via `validateTombstoneAckClosureLocked`; projection via `Index.RebuildProjection` | Arrival `[Ack before Tombstone at the peer]`; classes `[Tombstone,Acknowledgement]`; sync retains both exact bytes and roots; a preexisting `sessrepo` session remains after sync. Ack-only union remains stored, refuses closure, then recovers after Tombstone arrival. A wrong `subject_id` refuses and both objects remain. | `union-ack-arrival-order`, `sync-skip-tombstone-union` → `TestDurableSyncFindsMissingJSONObjectsAndRetainsTombstoneRecords`; `sync-unclosed-tombstone-ack` → `TestDurableSyncRefusesUnclosedAcknowledgementThenRecovers`; `durable-ack-subject-link` → `TestDurableSyncRefusesAcknowledgementWithMismatchedTombstoneSubject`; `projection-skip-unclosed-ack` → `TestProjectionRebuildRefusesUnclosedAcknowledgement` |
| Derive all lease heads after union and preserve a losing branch | `Index.RebuildProjection` → `Reader.LeaseHeadsForSession` → `sessstate.Projector.Project` | Lease tuple set `[two same-epoch competing UUIDv4 IDs]`; generated cardinalities `[0..32]`; event objects `[authoritative session.created, divergent session.idle]`; repository chain `[only session.created]`; stored union `[both exact event bytes]`; timestamps reversed in both directions; duplicate lease ID with different bytes refuses. | `union-lease-tuple-omission` → `TestLeaseHeadsForSessionReturnsEveryTupleAcrossTimestampPerturbation`; `union-lease-generated-cardinality-omission` → `TestLeaseHeadsForSessionCoversGeneratedCardinalityRange/size_17`; `union-lease-conflicting-same-id` → `TestLeaseHeadsForSessionRefusesConflictingBytesForOneLeaseID`; `projection-drops-union-head` → `TestProjectionRebuildExhaustsUnionArrivalsWithoutTimestampAuthority` |
| Rebuild projection only from closed union and exact authority bytes | `Index.RebuildProjection` / `DurableIndex.RebuildProjection` → `validateTombstoneAckClosure` and `sessrepo` record/event reads | Inputs `[record missing, event missing, Ack missing Tombstone]`; event/record bytes must match union byte-for-byte before reduce. | `projection-missing-union-record`, `projection-missing-union-event` → `TestProjectionRebuildRefusesAuthorityBytesMissingFromUnion`; `projection-skip-unclosed-ack` → `TestProjectionRebuildRefusesUnclosedAcknowledgement` |
| Rebuild projection independent of union arrival and timestamp | `Index.RebuildProjection` and durable wrapper `DurableIndex.RebuildProjection` | Six permuted union objects (`6! = 720` orders) × two opposing timestamp assignments = 1,440 in-memory production rebuilds; fixed repository authority snapshot; oracle pins literal state, `(epoch, lease_id)` winner, conflicts, all namespace IDs/bytes/counts and roots. Durable wrapper is also driven once for each timestamp assignment. | `union-lease-created-at-winner`, `projection-drops-union-head` → `TestProjectionRebuildExhaustsUnionArrivalsWithoutTimestampAuthority` |

The reducer receives only validated lease tuples after a sorted identity
snapshot. `LeaseHeadsForSession` drops `created_at`; `Index.RebuildProjection`
has no clock input or `time.Now` call. Event input comes from the validated
`sessrepo` authoritative chain after exact byte-presence checks against the
union. Thus the projection is deterministic for the same immutable union and
fixed `sessrepo` authority snapshot. This leaf preserves divergent event bytes
but does not build a new arbitrary event-DAG index or mutate the authoritative
chain.

## TASK-260830-147hsj explicit bounds

| Out-of-contract row | Acceptance clause / pinned clause | Owner or measured bound |
| --- | --- | --- |
| Raw blob/chunk transfer, retained staging, destination materialization and atomic commit | Pinned §11.4 rule 7: referenced blob transfer and destination materialization begin only after record union; task instructions explicitly say model §§11.5–11.6 at the boundary, do not implement them. | `STORY-260830-14qxuc` owns transfer/staging; `STORY-260830-2r137i` owns destination materialization. Durable storage only persists the five JSON namespaces. |
| Network transport, public `ax sync`, Host Channel wiring and doctor/capability advertisement | Acceptance asks for the scoped production behavior and says no unsupported capability is advertised; it does not define a transport or public command. | This is an in-process package API. No `ax sync`/Host Channel/doctor surface is wired or claimed. |
| Reconstructing a separate authoritative chain from arbitrary union events | Task implementation instructions bind event authority to the existing `sessrepo` chain and send only that chain to the existing pure reducer. | This leaf retains union event bytes and derives lease conflicts; it does not alter `sessrepo` or build a second DAG interpreter. |
| Power loss, physical disk faults, and native Windows crash execution | Acceptance requires crash/idempotency evidence for durable mutations. | Tests terminate a child process at install/quarantine boundaries and reopen. Windows cross-vet is required; native Windows runtime and physical power-loss fault injection are not run. |
| Exhaustive enumeration of SHA-256 and UUID domains | Acceptance requires every arrival permutation for a small event/object set; schema/digest grammar already has landed owners. | The event/object permutation domain is fully enumerated at N=6; full digest/UUID domains delegate to `canonicaljson` and `scalar`. |

The brief has no formal surface table. The task handoff therefore reports this
as a brief gap and includes a six-row derived coverage map in
`coverage-map.md`; it is not treated as a waiver.

**Measured TASK-260830-147hsj coverage: 6 of 6 derived acceptance rows driven
through named production call sites.** The exact call sites, test names, axes,
and killed mutants are in the table above and `coverage-map.md`.

## TASK-260830-2h5uv9 generated convergence

The production entry is `DurableIndex.SyncFrom`; projection checks run through
`DurableIndex.RebuildProjection`. The candidate source tests 38,000 valid-union
cases and 37,765 generated same-identity/different-byte conflict cases
(75,765 total): N=1..5 full arrival permutations and 34 deterministic unique
N=6 orders; duplicate off/on; gap absent/present (the missing object arrives
in pass 2); timestamp skew in both directions; every proper object subset of
the peer; and pass prefixes 1, 2, and 3. The set includes competing lease
tuples, a losing-lease event branch, and a Tombstone/Acknowledgement pair.
Every feasible conflict-target × conflict-round pair (rounds 1, 2, 3) is
crossed with those perturbation axes; each conflicting peer also holds an
identity absent locally. Reference roots and projection are computed
independently of `Index.Root` and the production lease comparator. Each
converged case repeats sync and compares all stored file bytes.

| Surface | Production entry | Named evidence | Narrowing evidence |
| --- | --- | --- | --- |
| Complete generated perturbation product and independent reference outcome | `DurableIndex.SyncFrom` → `MissingObjectIDs` / `FetchObjects`; `RebuildProjection` | `TestDurableSyncGeneratedPerturbationProduct` | `sync-repeat-record-quarantine-write`, `union-lease-created-at-winner`, `sync-partial-peer-regresses-local-root` |
| Arrival reorder; all permutations through N=5 and sampled N=6 | `DurableIndex.SyncFrom` → `fetchAndAdd` | `TestDurableSyncArrivalOrderConverges`, generated property | `sync-arrival-order-truncates-fetch`, `sync-missing-record-fetch` |
| Duplicate delivery and repeated sync idempotency | `DurableIndex.SyncFrom` / post-convergence `SyncFrom` | `TestDurableSyncDuplicateDeliveryIsByteIdentical`, generated property with whole-store byte snapshots | `sync-repeat-record-quarantine-write` |
| Delayed gap fill | `DurableIndex.SyncFrom` → missing-object fetch | `TestDurableSyncGapFillsOnLaterPass`, generated property | `sync-missing-record-fetch` |
| Clock skew cannot select competing lease winner | `DurableIndex.SyncFrom` → `RebuildProjection` → `LeaseHeadsForSession` | `TestDurableSyncClockSkewDoesNotSelectLeaseWinner`, generated oracle | `union-lease-created-at-winner`, `projection-drops-union-head` |
| Partial peer cannot regress local roots | `DurableIndex.SyncFrom` | `TestDurableSyncPartialPeerNeverRegressesLocalRoots`, generated ID monotonicity checks | `sync-partial-peer-regresses-local-root` |
| Same digest/different bytes quarantines, returns literal code and aborts even alongside peer-only identities or after a successful sync | `DurableIndex.SyncFrom` common-ID audit → quarantine | `TestDurableSyncConflictWithPartialOverlapPeer`, `TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord`, generated conflict class, `TestDurableSyncSameDigestDifferentBytesIsTypedConflict`, conflict crash recovery | `sync-common-audit-partial-overlap-peer`, `sync-common-audit-only-without-missing`, `sync-common-audit-first-namespace-only`, `sync-common-audit-first-sync-only`, `sync-common-audit-first-id-only`, `sync-common-audit-last-id-only`, `sync-common-audit-even-ids-only` |
| Common-ID audit stays unconditional across missing-set size, peer size, namespace position, and every common-ID position | `DurableIndex.SyncFrom` AST call site and runtime refusal | `TestDurableSyncCommonIDAuditIsUnconditionalInAST`, `TestDurableSyncConflictWithPartialOverlapPeer`, `TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord` | `sync-common-audit-ast-control`, `sync-common-audit-early-namespace-return`, `sync-common-audit-first-namespace-only`, `sync-common-audit-first-id-only`, `sync-common-audit-last-id-only`, `sync-common-audit-even-ids-only` |
| Schema/namespace mismatch returns a typed refusal through sync | `DurableIndex.SyncFrom` → `Index.ObjectsGet` | `TestDurableSyncRefusesPeerNamespaceMismatch`; direct server/client hostile namespace sweeps remain in the prior matrix | `sync-peer-namespace-mismatch`, `objects-get-schema-membership`, `fetchns` |
| Unclosed acknowledgement refuses and recovers after missing Tombstone is added | `DurableIndex.SyncFrom` → union closure | `TestDurableSyncUnclosedAcknowledgementAborts`, `TestDurableSyncRefusesUnclosedAcknowledgementThenRecovers` | `sync-unclosed-tombstone-ack` |

**Measured TASK-260830-2h5uv9 ratio: 1 of 1 written acceptance-criterion rows
driven through `DurableIndex.SyncFrom`.** The producer brief supplied no formal
surface table; the ten rows above are its task-scoped measured decomposition
and the handoff records the brief gap.

## TASK-260830-2h5uv9 out-of-contract rows

| Row outside this leaf's production entry | Acceptance clause / pinned clause | Bound |
| --- | --- | --- |
| Raw blob/chunk transfer and destination materialization | Acceptance names convergence under reorder, duplicate, gap, skew, partial peer and repeat sync; pinned §11.4 rule 7 says referenced transfer/materialization starts after record union. | `DurableIndex.SyncFrom` persists five validated JSON namespaces and does not perform §11.5 transfer. The no-transfer boundary is explicitly described in the registry's Section 11.4#6 gap; no transfer capability is claimed. |
| Tombstone issuance, deletion convergence, retention scheduling, and acknowledgement authorization | Acceptance is limited to anti-entropy convergence and says no unsupported capability is advertised; pinned §10.7 covers these lifecycle operations. | The tested clause is §10.7#13 (union without session mutation). All other §10.7 clauses stay unclaimed; no lifecycle or retention capability is advertised. |
| Physical power-loss injection, storage-controller faults, and native Windows process-crash simulation | Acceptance clause requires crash/idempotency evidence when durable state mutates. | Existing named tests crash/reopen child processes at object-install and quarantine boundaries; this does not model physical power loss. Windows `vet` is separate from runtime crash evidence. |
| Object sets larger than N=6 and exhaustive order enumeration at N=6 | Producer requirement explicitly bounds object sets to N≤6 and requires sampling beyond N=5. | N=1..5 permutations are exhaustive; N=6 uses 34 deterministic unique orders. `SyncFrom` processes identity sets without cardinality-specific branches; each extra unrelated validated identity adds one ordinary membership iteration. Cross-object graphs beyond the tested lease branch and Tombstone/Acknowledgement pair are not measured. |
| More than three sync passes, or a peer that continues changing after pass 3 | Producer requires one to three repeated rounds. | Against a fixed peer snapshot, one successful `SyncFrom` exhausts its missing IDs; the delayed gap arrives before pass 2, pass 3 and the extra post-convergence call add no identities, and the common-ID audit repeats on every call. A growing peer beyond this schedule is not measured. |
| Full SHA-256 and UUID spaces | Acceptance focuses on generated small object sets; identity validation is owned by the existing `canonicaljson` and `scalar` gates. | Generated object IDs are valid schema identities, and every same-digest/different-byte branch is exercised; exhaustive digest and UUID domains are not claimed. |

The story-final registry acceptance cases are
`story-260830-nxqqaw-inventory-exchange`,
`story-260830-147hsj-durable-union-projection`, and
`story-260830-2h5uv9-order-duplicate-gap-convergence`. The decoded-clause
regression is `TestStoryFinalAntiEntropyCasesAppearInDecodedClauseLists`.
