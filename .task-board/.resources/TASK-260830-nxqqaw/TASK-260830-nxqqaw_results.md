# TASK-260830-nxqqaw developer handoff results

**Disposition: ready for review.** The candidate remains uncommitted in the managed Story worktree; task status stays `development` until the developer handoff admits the snapshot.

- **Story:** `STORY-260830-ub60id`
- **Base/checkpoint:** `0ca3e4c26e2b275212796657f785b9b450f6174e`
- **Candidate tree (scratch `GIT_INDEX_FILE`, uncommitted):** `de8741fcc0f1cda0bef3ab870dfad1c795e4d43b`
- **Candidate branch:** `task-board/story/STORY-260830-ub60id`
- **Authority:** pinned `internal/specdoc/SPEC.v0.7.0.md`, §§10, 11.3–11.5; the task record's v0.5.0 reference is stale.

## Rev3 rework

The rev2 finding was case-insensitive `encoding/json` matching for a `WireObject` response member. The package now routes direct JSON reads through `strictJSON`: `canonicaljson.Canonicalize` rejects malformed JSON and duplicate members first, exact map keys and typed readers reject casefolded, missing, extra, and wrong-kind members, and only the allowlisted RawMessage/map/array/string/`scalar.Uint53` destinations reach `json.Unmarshal`. A production AST guard catches aliased `Unmarshal`, bound decoder references, `json.NewDecoder`, dot imports, and `raw*` accessors. RPC request/response envelopes and nested structured errors continue through the landed `internal/rpcwire` / `internal/axerror` validators.

The decoder census in `internal/merkleinventory/CONFORMANCE-MATRIX.md` lists all 10 package production decode/read sites and their strict path or delegated owner. The 8 closed RPC wire shapes (three operation bodies, the outer request envelope, success and failure response envelopes, the objects body, and `WireObject`) have per-member casefold/missing/extra/duplicate and six-JSON-kind cases at `Index.Dispatch` or `FetchObjects` with a hostile raw `ObjectSource`. Response array item kinds and 0/1/2 cardinality are also driven through `FetchObjects`. Internal immutable-object schema shape is validated by `canonicaljson`; the local Tombstone/Acknowledgement link and Descriptor/BlobChunk reads use the same strict helpers after that validation.

Rev2 notes are pinned: a `cbor`-labelled response after a JSON request is refused by `TestFetchObjectsRefusesCBORLabelAfterJSONRequest`, and `client-fetch-cbor-label` kills the narrowing alone. The unreachable `ObjectsGet` quarantine arm was removed because quarantine removes both the stored object and its namespace location before serving.

## Implementation and measured coverage

The production `internal/merkleinventory` entries provide deterministic RFC 8785 namespace roots/children, total schema/byte-class membership, bounded `inventory.roots`, `inventory.children`, and `objects.get`, recursive differing-node retrieval, and singleton-bounded fetches. No unsupported anti-entropy capability is advertised.

**High-level acceptance ratio: 1 of 1 criterion rows driven.** Production call sites are `Index.Root`, `Index.Child`, `Index.Dispatch`, `Index.AddJSON`, `Index.AddBlob`, `Index.ObjectsGet`, `MissingNamespaceIDs`, `MissingObjectIDs`, and `FetchObjects`.

**Original producer-property ratio: 4 of 4 rows driven.** The table in `CONFORMANCE-MATRIX.md` binds each row to the named production entries, tests, isolated narrowing, or explicit owner-bound.

**Wire-shape ratio: 8 of 8 closed RPC wire shapes exhaustively varied** by member key and JSON value kind at the server/client entry points above. **Decoder source census: 10 of 10 read/decode sites use `strictJSON` or the landed `rpcwire`/`canonicaljson` owner.** This ratio covers decoder use and wire member shapes, not the complete semantics of every schema-owned immutable object.

- **Deterministic trie:** empty, singleton, and branch roots; both branch child hashes; three exact `inventory.children` bodies; and all six MIXED-NS-1 roots reproduce byte-exact through `internal/canonicaljson`. More than one identity at depth 64 refuses with literal `integrity_failure`.
- **Total, disjoint membership:** all 19 included schema/version rows, 42 excluded schema identities, complete raw blobs, Chunks, and local/transient classes are mapped. MIXED-NS-N1 demonstrates Descriptor-as-record, independently enumerated Chunks, and a local marker against roots and schema/byte-class admission. `objects.get` refuses a stored Descriptor requested as `record`.
- **Bounded serving:** tests cover six sorted unique namespaces; every valid lowercase prefix length 0–64, plus invalid lengths/alphabet/case; `not_found` at every valid absent non-root prefix; child counts 0–16; leaf IDs 0–1; ordered unique request IDs; count limits 4096/4097; JSON-only encoding; negotiated line limits; and singleton object batches.
- **Identity-level MIXED-NS-EXCHANGE:** `TestMixedNSExchangeIdentityLevelSyntheticIDs` drives the exact fixture identity sets through production roots/children and recursive walking. Peer B differs only in record, manifest, and blob; the missing set is exactly `{Z(3), T(2), blob-5}`; after adding those identities all six roots match. It makes no object-validation claim, consistent with §11.4's statement that the synthetic IDs' contents are not used beyond validated identities.
- **Validated real-byte exchange:** `TestInProcessExchangeWalksChangedNamespacesAndFetchesExactObjects` fetches exactly the missing Checkpoint and Descriptor, validates before admission, pins six literal roots derived from real identities, and verifies record-root equality before raw blob admission. Separate production-entry tests pin identical replay, same-digest/different-bytes quarantine, Tombstone/Acknowledgement union without execution, and no timestamp winner.

### Mapped schemas that remain fail-closed

The orchestrator decision rev1 states: “KEEP FAIL-CLOSED. Do not add shallow validators.” These rows remain mapped to exactly one namespace; admission waits for the complete schema owner validator. Each test asserts literal `integrity_failure`, unchanged six roots/counts, and no quarantine; its row-specific admitting narrowing is killed alone.

| Schema row | Namespace | Refusal test | Validator owner |
| --- | --- | --- | --- |
| `session-record@3.1.0` | `record` | `TestUnsupportedSessionRecord31RemainsFailClosed` | `STORY-260830-4n0fo8` — authoritative-record-schema-core |
| `materialization-plan@1.0.0` | `manifest` | `TestUnsupportedMaterializationPlan10RemainsFailClosed` | `STORY-260830-2r137i` — workspace-materialization-and-groups |
| `materialization-plan@2.0.0` | `manifest` | `TestUnsupportedMaterializationPlan20RemainsFailClosed` | `STORY-260830-2r137i` — workspace-materialization-and-groups |
| `task-board-bundle@1.0.0` | `manifest` | `TestUnsupportedTaskBoardBundle10RemainsFailClosed` | `STORY-260830-27pqyi` — task-board-bridge-and-bundle |

## Out-of-contract rows and owners

| Out-of-contract row | Acceptance/spec clause | Owner or bound |
| --- | --- | --- |
| Complete validation/admission for the four mapped schema rows above | Orchestrator decision rev1 Item 1: “KEEP FAIL-CLOSED. Do not add shallow validators.” | The four validator Stories listed above; all four rows refuse at `Index.AddJSON`. |
| Chunk staging, retry, transfer recovery, and destination materialization | Task requires bounded inventory exchange; decision rev1 Item 7 directs transfer/materialization to the owning boundary; pinned SPEC §§11.5–11.6. | `STORY-260830-14qxuc` owns chunk transfer; `STORY-260830-2r137i` owns workspace materialization. This package admits complete raw bytes only. |
| Lease-head derivation and losing-lease-event preservation | Pinned SPEC §11.4 rules 5–6 require lease/branch state beyond this inventory package. | Sibling `TASK-260830-147hsj`; no lease winner or event branch is computed here. |
| Crash/restart evidence for durable exchange | Task AC makes crash/idempotency evidence conditional on durable state mutation. | `Index` is process-local and performs no durable writes; the trigger does not apply. |
| Mesh RPC 3/4 inventory serving | Task scope pins §11.3 RPC 2 shapes; pinned SPEC §11.8 adds RPC 3 `directory_record`, and §11.9 adds RPC 4 `terminal_backend_evidence`. | `Index.New` accepts RPC 2.0.0 only and composes `rpcwire.Namespaces(version)`; this leaf makes no RPC 3/4 serving claim. |
| Public `ax` doctor/Host Channel integration | Acceptance requires README/doctor/capability evidence without unsupported claims; this leaf delivers the internal inventory implementation. | No Host Channel handler or doctor surface is added, and no anti-entropy capability is advertised. |

**Coverage-map brief gap:** the task and producer brief contain no surface table. The gate × entry × axis census is in the conformance matrix; no surface-row coverage map can be derived from the missing table.

## Importer outcome comparison

The base and rev3 candidate outcome grids are keyed by `(imported package, importer package, entry, input class)` in `internal/merkleinventory/IMPORTER-OUTCOMES.md`. Both source snapshots use Go 1.25.5 on darwin/arm64 and are anchored to base `0ca3e4c26e2b275212796657f785b9b450f6174e` and this candidate.

| Imported package | Base → candidate importer paths | Added importer | Moved class |
| --- | ---: | --- | --- |
| `internal/canonicaljson` | 22 → 23 | `internal/merkleinventory` | Tombstone and Tombstone Acknowledgement v1.0.0 move from fail-closed to admitted after Section 10.7 shape and self-digest validation. |
| `internal/rpcwire` | 3 → 4 | `internal/merkleinventory` | No validator moved; `rpcwire` production source is unchanged. |
| `internal/provhost` | 5 → 5 | none | Test-only exact source-ownership allowlist remains anchored to `internal/merkleinventory/index.go`. |

Base importer tests: 26/26 packages, 21,792 passed test/subtest events, 14 skipped, exit 0. Candidate importer tests: 27/27 packages, 22,854 passed test/subtest events, 14 skipped, exit 0. The final candidate graph is `.temp/TASK-260830-nxqqaw/importer/go-list-candidate-final.json`; the complete raw test JSON streams remain in the ignored worktree `.temp`. The under-1-MiB evidence archive includes a compact terminal-event projection preserving each `Action`, `Package`, and `Test`, plus command metadata. No additional runtime class moved in rev3. Four unsupported schemas remain fail-closed. The Story final leaf owns `internal/traceability/ownership.v0.7.0.json`; this task_delta did not edit it.

The rework diff outside `internal/merkleinventory` is declared: `internal/canonicaljson` implements Section 10.7 Tombstone/Acknowledgement validators used by the inventory membership path; `internal/provhost` test/traceability files bind the identity-verification ownership gate; root/package READMEs and `LOGBOOK.md` document the behavior and evidence. No `internal/rpcwire` production code or `task-board.config.json` changed.

## Validation results

Every gate below was run as a standalone process. All listed commands exited 0. Logs are under `.temp/TASK-260830-nxqqaw/validation/rev3/` unless noted.

| Command | Exit | Log |
| --- | ---: | --- |
| `test -z "$(gofmt -l $(git ls-files --cached --others --exclude-standard -- '*.go'))"` | 0 | `format.log` |
| `go build ./...` | 0 | `build.log` |
| `go vet ./...` | 0 | `vet-native.log` |
| `GOOS=windows GOARCH=amd64 go vet ./...` | 0 | `vet-windows.log` |
| `go test ./... -count=1 -v` | 0 | `full-test-verbose.log` |
| `go test -race -count=1 -timeout 25m` — group 1, 8 package args from `race-subsets-01.txt` | 0 | `race-group-01.log` |
| Same race command — group 2, 8 package args from `race-subsets-01.txt` | 0 | `race-group-02.log` |
| Same race command — group 3, 8 package args including `merkleinventory` from `race-subsets-01.txt` | 0 | `race-group-03.log` |
| Same race command — group 4, 8 package args from `race-subsets-01.txt` | 0 | `race-group-04.log` |
| Same race command — group 5, 8 package args from `race-subsets-01.txt` | 0 | `race-group-05.log` |
| Same race command — group 6, 5 package args from `race-subsets-01.txt` | 0 | `race-group-06.log` |
| `go test ./... -cover -count=1` | 0 | `full-cover.log` |
| `go test ./internal/merkleinventory -count=3` | 0 | `merkleinventory-count3-final.log` |
| `python3 .temp/TASK-260830-nxqqaw/validation/rev3/run_importer_grid.py .temp/TASK-260830-nxqqaw/importer/base-0ca3e4c .temp/TASK-260830-nxqqaw/importer/importer-packages-base-rev3.txt .temp/TASK-260830-nxqqaw/validation/rev3/importer-base.json` (nested `go test -json -p=1 -parallel=1 -count=1`, 26 packages) | 0 | `importer-base.json` + `.meta.json` |
| `python3 .temp/TASK-260830-nxqqaw/validation/rev3/run_importer_grid.py . .temp/TASK-260830-nxqqaw/importer/importer-packages-candidate-rev3.txt .temp/TASK-260830-nxqqaw/validation/rev3/importer-candidate-final.json` (nested `go test -json -p=1 -parallel=1 -count=1`, 27 packages) | 0 | `importer-candidate-final.json` + `.meta.json` |
| `python3 internal/merkleinventory/mutations.py --output .temp/TASK-260830-nxqqaw/validation/rev3/mutations-full-final` | 0 | `mutations-full-final/results.json` and per-plant `test.log` files |
| `go test ./internal/rpcwire -run=^$ -fuzz=^FuzzUntrustedEnvelopes$ -fuzztime=100x -parallel=1` | 0 | `fuzz-rpcwire-01.log` |
| `go test ./internal/rpcwire -run=^$ -fuzz=^FuzzClosedOperationBodies$ -fuzztime=100x -parallel=1` | 0 | `fuzz-rpcwire-02.log` |
| `go test ./internal/rpcwire -run=^$ -fuzz=^FuzzNamespaceVocabulary$ -fuzztime=100x -parallel=1` | 0 | `fuzz-rpcwire-03.log` |
| `go test ./internal/rpcwire -run=^$ -fuzz=^FuzzUnknownFields$ -fuzztime=100x -parallel=1` | 0 | `fuzz-rpcwire-04.log` |
| `go test ./internal/scalar -run=^$ -fuzz=^FuzzScalarProductionEntries$ -fuzztime=100x -parallel=1` | 0 | `fuzz-scalar-01.log` |
| `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzCanonicalizeRoundTrip$ -fuzztime=100x -parallel=1` | 0 | `fuzz-canonicaljson-02.log` |
| `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzObjectIdentityRepresentationInvariant$ -fuzztime=100x -parallel=1` | 0 | `fuzz-canonicaljson-03.log` |
| `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzClosedIdentityShapeRefusal$ -fuzztime=100x -parallel=1` | 0 | `fuzz-canonicaljson-04.log` |
| `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzObservationEventRefusal$ -fuzztime=100x -parallel=1` | 0 | `fuzz-canonicaljson-05.log` |
| `go test ./internal/secconftest -run=^$ -fuzz=^FuzzCheckArgv$ -fuzztime=100x -parallel=1` | 0 | `fuzz-secconftest-01.log` |
| `go test ./internal/secconftest -run=^$ -fuzz=^FuzzCheckMemberPath$ -fuzztime=100x -parallel=1` | 0 | `fuzz-secconftest-02.log` |
| `go test ./internal/secconftest -run=^$ -fuzz=^FuzzIsEnvName$ -fuzztime=100x -parallel=1` | 0 | `fuzz-secconftest-03.log` |
| `go test ./internal/secconftest -run=^$ -fuzz=^FuzzRedactCorpus$ -fuzztime=100x -parallel=1` | 0 | `fuzz-secconftest-04.log` |
| `go test ./internal/secconftest -run=^$ -fuzz=^FuzzEscapeForTerminal$ -fuzztime=100x -parallel=1` | 0 | `fuzz-secconftest-05.log` |
| `go test ./internal/secconftest -run=^$ -fuzz=^FuzzRenderForTerminal$ -fuzztime=100x -parallel=1` | 0 | `fuzz-secconftest-06.log` |
| `go test ./internal/secconftest -run=^$ -fuzz=^FuzzGuardResolve$ -fuzztime=100x -parallel=1` | 0 | `fuzz-secconftest-07.log` |
| `go test ./internal/secconftest -run=^$ -fuzz=^FuzzDetectCaseCollision$ -fuzztime=100x -parallel=1` | 0 | `fuzz-secconftest-08.log` |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `tracecheck-final.log` |
| `go run ./internal/catalog/cmd/cataloggen -adopted -output internal/catalog/catalog_gen.go -check` | 0 | `cataloggen-check.log` |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | `build-linux.log` |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | `build-windows.log` |
| `set -o pipefail; git ls-files -z '*.json' | xargs -0 -n1 python3 -c 'import json,sys;json.load(open(sys.argv[1]))'` | 0 | `json-syntax.log` |
| `/Users/iv/.curator/global/bin/task-board validate` | 0 | `task-board-validate.log` |
| `go list -json ./...` | 0 | `importer/go-list-candidate-final.json` |
| `git diff --check` | 0 | `diff-check-final.log` |
| `git diff --exit-code HEAD -- task-board.config.json` | 0 | `task-board-config-unchanged.log` |
| `git diff --exit-code HEAD -- internal/traceability/ownership.v0.7.0.json` | 0 | `registry-unchanged.log` |

`task-board validate` exited 0 and reported 174 `[MISSING_ACTIVITY]` warnings, 256 existing board issues, and 3279/3279 ledger rows mirrored. None of its warnings named this Story or task. The importer base source identity did not change during rev3; its earlier green base run is reused, while the candidate grid was rerun after the final response-envelope test additions.

The six race commands partition the full 45-package set to stay inside the shell call limit; their exact package arguments are in `race-subsets-01.txt`. After the suite, only tracked Markdown documentation and ignored task evidence were edited; Go source, Go tests, and project configuration remained byte-identical for the measured validation identity. Final `tracecheck` and `git diff --check` were rerun after those documentation edits.

## Mutation evidence

The final mutation harness recorded 49 applied plants: 47 ordinary narrowing mutants killed, one token-preserving behavioral bypass killed with its source census passing, and one harmless applied neutral-comment control surviving. The harness itself exited 0. Each narrowing's standalone test process exited 1 and the named test failed; the control's test process exited 0. Raw plant logs and overlays are stored under `mutations-full-final/`.

| Mutant | Narrowing | Named test / raw log | Test exit and result | Survivor bound |
| --- | --- | --- | ---: | --- |
| `identity-token-preserving-refusal-bypass` | Leaves canonicaljson.VerifyObjectIdentity in the owned inventory call site but converts its validation error into an accepted membership. | `TestClassifyRefusesIdentityDigestMismatch` / `identity-token-preserving-refusal-bypass/behavioral-suite.log`; source gate `passed with searched token preserved` (`identity-token-preserving-refusal-bypass/source-gate.log`) | 1 — killed-by-behavior; source census passed | — |
| `identity-inventory-owner-prefix` | Broadens the exact inventory index allowlist to every Go file under internal/merkleinventory, admitting the synthetic other.go caller. | `TestAttestationAllowlistAnchorsOwningPath` / `identity-inventory-owner-prefix/test.log` | 1 — killed | — |
| `control-neutral-comment` | Equivalent comment-only edit; this applied control must survive. | `TestDispatchServesNormativeChildrenBody` / `control-neutral-comment/test.log` | 0 — survived-control | Comment-only calibration control; no behavior bound claimed. |
| `rule1-duplicate-set-member` | Retains a second copy of one duplicate ID in the node set. | `TestNodeRuleShapesAndDepth64IntegrityFailure` / `rule1-duplicate-set-member/test.log` | 1 — killed | — |
| `rule2-empty-root-count` | Admits one phantom member into the empty-root count. | `TestDispatchServesNormativeChildrenBody` / `rule2-empty-root-count/test.log` | 1 — killed | — |
| `rule3-root-singleton-leaf` | Stops encoding a singleton as a leaf at the root prefix. | `TestDispatchServesNormativeChildrenBody` / `rule3-root-singleton-leaf/test.log` | 1 — killed | — |
| `rule4-retain-single-child` | Drops a required one-child node below the root. | `TestRuleFourRetainsSingleChildAtEverySharedPrefix` / `rule4-retain-single-child/test.log` | 1 — killed | — |
| `rule5-depth64-duplicate` | Admits two entries at a full 64-nibble prefix. | `TestNodeRuleShapesAndDepth64IntegrityFailure` / `rule5-depth64-duplicate/test.log` | 1 — killed | — |
| `children-label-order` | Reverses the specified ascending nibble-label order in the served branch body. | `TestDispatchServesNormativeChildrenBody` / `children-label-order/test.log` | 1 — killed | — |
| `prefix-length-65` | Admits a 65-nibble prefix. | `TestDispatchPrefixAxesAndNotFound` / `prefix-length-65/test.log` | 1 — killed | — |
| `uppercase-prefix` | Admits uppercase A–F in a prefix. | `TestDispatchPrefixAxesAndNotFound` / `uppercase-prefix/test.log` | 1 — killed | — |
| `missing-prefix-not-found` | Returns an invented empty body for the valid missing prefix f. | `TestDispatchPrefixAxesAndNotFound` / `missing-prefix-not-found/test.log` | 1 — killed | — |
| `children-prefix-type-null` | Admits JSON null as the empty root prefix through encoding/json's string zero value. | `TestDispatchRequestShapeStrictTypes` / `children-prefix-type-null/test.log` | 1 — killed | — |
| `server-casefold-prefix` | Admits the miscased Prefix member at the server inventory.children entry. | `TestDispatchRejectsEveryNonExactOperationBodyShape` / `server-casefold-prefix/test.log` | 1 — killed | — |
| `client-casefold-wireobject` | Admits one case-folded WireObject member at the hostile client response entry. | `TestFetchObjectsRejectsEveryNonExactSuccessWireShape` / `client-casefold-wireobject/test.log` | 1 — killed | — |
| `client-two-wireobjects` | Accepts two WireObjects for the one-ID singleton FetchObjects request and silently returns only the first. | `TestFetchObjectsRejectsNonObjectItemsAndWrongArrayCardinality` / `client-two-wireobjects/test.log` | 1 — killed | — |
| `client-fetch-cbor-label` | Admits a cbor-labelled response object after the client requested json. | `TestFetchObjectsRefusesCBORLabelAfterJSONRequest` / `client-fetch-cbor-label/test.log` | 1 — killed | — |
| `guard-aliased-json-unmarshal` | Control plant: adds an aliased json.Unmarshal function value outside strictJSON; the AST guard must fail. | `TestWireDecoderASTGuardRejectsAliasedUnmarshalBindings` / `guard-aliased-json-unmarshal/test.log` | 1 — killed | — |
| `emptychild2` | Returns an invented empty child only for a valid absent two-nibble prefix. | `TestDispatchPrefixAxisCoversEveryLengthAndAlphabet` / `emptychild2/test.log` | 1 — killed | — |
| `crossns` | Lets a stored record requested through manifest fall through to not_found instead of the cross-namespace refusal. | `TestObjectsGetRefusesEveryForeignNamespacePair` / `crossns/test.log` | 1 — killed | — |
| `fetchns` | Admits a hostile peer's valid record as a manifest object on the client fetch path. | `TestFetchObjectsRefusesEveryHostileForeignNamespacePair` / `fetchns/test.log` | 1 — killed | — |
| `descriptor-namespace` | Relabels Blob Descriptor schema membership from manifest to record. | `TestBlobDescriptorOwnsManifestMembershipAndCompleteRawBlob` / `descriptor-namespace/test.log` | 1 — killed | — |
| `excluded-local-marker` | Admits one machine-local terminal marker into the record root. | `TestMixedNSN1RejectsDescriptorRelabelingChunksAndLocalMarker` / `excluded-local-marker/test.log` | 1 — killed | — |
| `mixed-ns1-root-member-narrowing` | Omits the exact synthetic Tombstone Acknowledgement identity from its production namespace root. | `TestMixedNS1RootsFromNormativeSyntheticIDs` / `mixed-ns1-root-member-narrowing/test.log` | 1 — killed | — |
| `objects-get-schema-membership` | Admits one manifest object from the stored-object check when queried as record. | `TestObjectsGetRevalidatesStoredSchemaMembership` / `objects-get-schema-membership/test.log` | 1 — killed | — |
| `exchange-objects-get-schema-membership` | Admits a real manifest Descriptor from a misrouted record store during the real-byte exchange. | `TestInProcessExchangeWalksChangedNamespacesAndFetchesExactObjects` / `exchange-objects-get-schema-membership/test.log` | 1 — killed | — |
| `objects-get-line-limit` | Emits exactly one byte over the negotiated response line limit. | `TestObjectsGetValidatesRequestedNamespaceAndNegotiatedLineLimit` / `objects-get-line-limit/test.log` | 1 — killed | — |
| `objects-get-request-count` | Admits a request with 4097 object IDs. | `TestObjectsGetRefusesRequestIDCountOutsideBound` / `objects-get-request-count/test.log` | 1 — killed | — |
| `same-id-quarantine` | Treats one different-byte stored variant as an identical object instead of quarantining it. | `TestAddJSONQuarantinesSameIDDifferentStoredBytes` / `same-id-quarantine/test.log` | 1 — killed | — |
| `tombstone-timestamp-winner` | Selects the later-created Tombstone as a winner and discards the same-subject earlier immutable Tombstone. | `TestInProcessTombstoneUnionRetainsBothTimesAndAcknowledgements` / `tombstone-timestamp-winner/test.log` | 1 — killed | — |
| `rpc2-version-admission` | Admits RPC 3.0.0 through this RPC 2-only inventory constructor. | `TestNewRefusesHigherRPCNamespaceSets` / `rpc2-version-admission/test.log` | 1 — killed | — |
| `blob-chunk-content-mismatch` | Admits the one wrong chunk digest used by the negative fixture while retaining all other digest checks. | `TestBlobDescriptorByteReferencesAreVerifiedBeforeAdmission` / `blob-chunk-content-mismatch/test.log` | 1 — killed | — |
| `unknown-schema-fail-closed` | Classifies one unknown schema as excluded instead of refusing it. | `TestRPC2SchemaMembershipTableIsTotalAndDisjoint` / `unknown-schema-fail-closed/test.log` | 1 — killed | — |
| `unsupported-schema-session-record-31` | Admits only urn:ax:schema:session-record@3.1.0 after canonicaljson refuses its identity contract; the injected byte digest makes the bad admission measurable at inventory roots/counts. | `TestUnsupportedSessionRecord31RemainsFailClosed` / `unsupported-schema-session-record-31/test.log` | 1 — killed | — |
| `unsupported-schema-materialization-plan-10` | Admits only urn:ax:schema:materialization-plan@1.0.0 by narrowing the fail-closed shape validator for that row; the fixture has a valid omit-self digest so validation refusal is isolated to the missing schema shape. | `TestUnsupportedMaterializationPlan10RemainsFailClosed` / `unsupported-schema-materialization-plan-10/test.log` | 1 — killed | — |
| `unsupported-schema-materialization-plan-20` | Admits only urn:ax:schema:materialization-plan@2.0.0 by narrowing the fail-closed shape validator for that row; the fixture has a valid omit-self digest so validation refusal is isolated to the missing schema shape. | `TestUnsupportedMaterializationPlan20RemainsFailClosed` / `unsupported-schema-materialization-plan-20/test.log` | 1 — killed | — |
| `unsupported-schema-task-board-bundle-10` | Admits only urn:ax:schema:task-board-bundle@1.0.0 by narrowing the fail-closed shape validator for that row; the fixture has a valid omit-self digest so validation refusal is isolated to the missing schema shape. | `TestUnsupportedTaskBoardBundle10RemainsFailClosed` / `unsupported-schema-task-board-bundle-10/test.log` | 1 — killed | — |
| `identity-walk-blob-narrowing` | Narrows identity-only recursive inventory walking to omit the fixture's valid raw-blob identity. | `TestMixedNSExchangeIdentityLevelSyntheticIDs` / `identity-walk-blob-narrowing/test.log` | 1 — killed | — |
| `objects-get-blob-transfer-boundary` | Admits a raw-blob identity through the object-only MissingObjectIDs walk in the supported RPC 2 version. | `TestMixedNSExchangeIdentityLevelSyntheticIDs` / `objects-get-blob-transfer-boundary/test.log` | 1 — killed | — |
| `objects-get-duplicate-id` | Admits a repeated object ID by weakening strict ascending order to non-decreasing order. | `TestObjectsGetRefusesUnsortedDuplicateAndUnavailableEncodingRequests` / `objects-get-duplicate-id/test.log` | 1 — killed | — |
| `objects-get-cbor-only` | Admits a CBOR-only request even though this RPC 2 implementation can return JSON only. | `TestObjectsGetRefusesUnsortedDuplicateAndUnavailableEncodingRequests` / `objects-get-cbor-only/test.log` | 1 — killed | — |
| `objects-get-unbounded-batch` | Sends the complete missing-ID list in each request instead of one bounded object per request. | `TestFetchObjectsUsesBoundedSingletonBatches` / `objects-get-unbounded-batch/test.log` | 1 — killed | — |
| `tombstone-target-scope-kind` | Admits a target-kind mismatch for the session scope. | `TestTombstoneIdentityEntryRefusesNarrowedTargetAndAuthorityShapes` / `tombstone-target-scope-kind/test.log` | 1 — killed | — |
| `tombstone-path-question-wildcard` | Admits a workspace path containing the question-mark wildcard. | `TestTombstoneIdentityEntryRefusesNarrowedTargetAndAuthorityShapes` / `tombstone-path-question-wildcard/test.log` | 1 — killed | — |
| `tombstone-logical-root-maximum` | Admits a 65-character logical_root beyond the pinned 64-character maximum. | `TestEveryCoreRecordDeclaredBoundAcceptsAtItsLimitAndRefusesPastIt` / `tombstone-logical-root-maximum/test.log` | 1 — killed | — |
| `tombstone-ack-conflict-checkpoint` | Admits a retained_conflict acknowledgement with a null checkpoint. | `TestTombstoneAckIdentityEntryEnforcesDispositionAndIssuerCouplings` / `tombstone-ack-conflict-checkpoint/test.log` | 1 — killed | — |
| `tombstone-ack-created-by` | Admits an applied acknowledgement created by a different host. | `TestTombstoneAckIdentityEntryEnforcesDispositionAndIssuerCouplings` / `tombstone-ack-created-by/test.log` | 1 — killed | — |
| `tombstone-ack-requires-reference` | Admits an unreferenced acknowledgement for one pinned fixture subject. | `TestAddTombstoneAcknowledgementRequiresAStoredMatchingTombstone` / `tombstone-ack-requires-reference/test.log` | 1 — killed | — |
| `tombstone-ack-subject-link` | Admits a mismatched acknowledgement subject for one pinned negative fixture. | `TestAddTombstoneAcknowledgementRequiresAStoredMatchingTombstone` / `tombstone-ack-subject-link/test.log` | 1 — killed | — |

## Candidate and handoff state

The Story leaf remains an uncommitted task_delta. No traceability ownership registry change was made. Results, conformance matrix, mutation evidence, and validation logs are task-scoped outcome resources; the candidate is left ready for review at the Story checkpoint.

## Continuation verification (2026-09-24)

The scratch-index tree recomputed from the live uncommitted candidate is `de8741fcc0f1cda0bef3ab870dfad1c795e4d43b`, identical to the tree recorded above. The base remains `0ca3e4c26e2b275212796657f785b9b450f6174e`. No tracked source, test, configuration, or environment input changed after that candidate's full validation; this continuation added only ignored `.temp` evidence. Go is `go1.25.5 darwin/arm64`.

Fresh focused checks on this continuation:

| Command | Exit / result | Evidence |
| --- | ---: | --- |
| `go test ./internal/merkleinventory -count=3` | 0 | `.temp/TASK-260830-nxqqaw/validation/rev3/current-package-count3.log` |
| `python3 internal/merkleinventory/mutations.py --only server-casefold-prefix,client-casefold-wireobject` (run 1) | harness 0; both mutants killed with test-process exit 1 | `mutations-rework-client-server-01/results.json` and each plant's `test.log` |
| Same targeted mutation command (run 2) | harness 0; both mutants killed with test-process exit 1 | `mutations-rework-client-server-02/results.json` and each plant's `test.log` |

The server narrowing admits the case-folded `Prefix` member at `Index.Dispatch`; `TestDispatchRejectsEveryNonExactOperationBodyShape/inventory_children_prefix_casefold` failed alone. The client narrowing admits a case-folded `object_id` in a hostile `ObjectSource` response; `TestFetchObjectsRejectsEveryNonExactSuccessWireShape/wireobject_object_id_casefold` failed alone. Both were killed in both isolated runs.

The exact-tree rev3 evidence is reused for the slower gates under the exact-evidence rule: `go test ./... -count=1 -v` exit 0, all six configured race-package groups exit 0, `GOOS=windows GOARCH=amd64 go vet ./...` exit 0, and the base/candidate importer grid exit 0. Their logs and command metadata are already attached in the rev3 validation/evidence archives. The importer comparison names the sole moved runtime classes: validated Tombstone and Tombstone Acknowledgement schemas changed from fail-closed to admitted under §10.7; no `rpcwire` validator or other runtime class moved. The candidate importer grid is 27 packages; the base grid is 26 packages.

The rev3 conformance matrix was read back from the board and byte-compared with `internal/merkleinventory/CONFORMANCE-MATRIX.md`; the task-scoped copy now matches it. Decoder census: 10 of 10 production read/decode sites use `strictJSON` or the landed `rpcwire`/`canonicaljson` owner. Closed wire shape coverage: 8 of 8 shapes. Producer property coverage remains 4 of 4 rows. The brief has no surface table; this brief gap remains stated above.
