# TASK-260830-nxqqaw conformance matrix

Authority: `internal/specdoc/SPEC.v0.7.0.md` §10, §11.3, §11.4, and §11.5.
The task record's v0.5.0 reference is stale; the producer instruction pins the
v0.7.0 text. Base checkpoint: `0ca3e4c26e2b275212796657f785b9b450f6174e`.
Candidate remains uncommitted in the managed Story worktree. Fixtures use no
network, wall clock, or durable storage.

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
| Validated real-byte exchange and immutable Tombstone/Acknowledgement union | `MissingObjectIDs` → `walkMissing`; `FetchObjects` → `ObjectSource.ObjectsGet`; `Index.AddJSON` / `Index.AddBlob` | Changed roots `[record,manifest,blob]`; retrieved IDs `[exact Checkpoint,exact Descriptor]`; stored/requested membership `[record/record,manifest/record]`; link sets `[same Tombstone/Ack,two Tombstones/two Acks]`; timestamps `[different,not winner-selected]`; blob admission `[before,after record-root equality]` | `TestInProcessExchangeWalksChangedNamespacesAndFetchesExactObjects` pins six roots from real identities and checks validated object responses; `exchange-objects-get-schema-membership` is killed by that test alone. `TestInProcessTombstoneUnionRetainsBothTimesAndAcknowledgements` asserts both differently timestamped immutable Tombstones and both Acknowledgements remain; `tombstone-timestamp-winner` narrows that union to the later timestamp and is killed by that test alone. `TestAddJSONQuarantinesSameIDDifferentStoredBytes` and `same-id-quarantine` cover rule 3; `TestClassifyRefusesIdentityDigestMismatch` covers rule 1. Blob bytes are admitted only after record-root equality in the in-process caller. No aggregate sync coordinator exists here; §11.5 chunk transfer/materialization and §11.4 rules 5–6 remain owner-bounded below. |
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
new inventory package, and its `rpcwire` dependency are listed with each
imported entry, input class, and base/candidate outcome in
[`IMPORTER-OUTCOMES.md`](IMPORTER-OUTCOMES.md). The conformance registry
`internal/traceability/ownership.v0.7.0.json` is unchanged; the Story final
leaf owns its update.

## Explicit scope bounds and owners

| Row outside this leaf's production entry points | Acceptance clause / pinned clause | Owner or bound |
| --- | --- | --- |
| Complete validator and admission for `session-record@3.1.0` | Orchestrator decision rev1, Item 1: “KEEP FAIL-CLOSED. Do not add shallow validators.” | `STORY-260830-4n0fo8 — authoritative-record-schema-core`; mapped to `record`, refused and tested here until that owner supplies the complete shape. |
| Complete validators and admission for both Materialization Plan versions | Same decision clause: keep the canonicaljson refusal; do not add a shallow validator. | `STORY-260830-2r137i — workspace-materialization-and-groups`; mapped to `manifest`, each version refused and tested here. |
| Complete validator and admission for Task-board Bundle | Same decision clause: keep the canonicaljson refusal; do not add a shallow validator. | `STORY-260830-27pqyi — task-board-bridge-and-bundle`; mapped to `manifest`, refused and tested here. |
| Chunk staging/retry, transfer recovery and destination materialization | Task AC requires bounded inventory exchange; decision rev1 Item 7 says transfer/materialization owned by another leaf must be modeled at its boundary. Pinned SPEC §§11.5–11.6. | `STORY-260830-14qxuc — resumable-blob-and-manifest-transfer` owns chunk transfer; `STORY-260830-2r137i — workspace-materialization-and-groups` owns workspace materialization. This package only validates/adopts complete raw bytes with `Index.AddBlob`. |
| Lease-head derivation and losing-lease-event preservation (union rules 5–6) | Pinned SPEC §11.4 rules 5–6; they require lease/branch state beyond namespace identity exchange. | Sibling `TASK-260830-147hsj — implement-object-discovery-and-union`, found under this Story. No lease winner or event branch is calculated by this package. |
| Crash/restart evidence for durable exchange | Task AC: “crash/idempotency evidence is included when the operation mutates durable state.” | This `Index` is process-local and does not write durable state; the conditional trigger does not apply. No crash-recovery capability is claimed. |
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
