# Mesh RPC 2 namespace inventory

`internal/merkleinventory` builds deterministic Merkle roots and child bodies
for the six Mesh RPC 2 namespaces, validates immutable schema membership and
serves bounded `inventory.roots`, `inventory.children` and `objects.get`
requests. The node encoding uses the RFC 8785 implementation in
`internal/canonicaljson` and Section 11.4 domain
`urn:ax:merkle-node:1`.

Request and response-body decoding share one strict JSON path: duplicate JSON
members are rejected before decoding, keys must match the closed shape exactly,
and each value is read with an explicit JSON type. The AST guard prevents
production code from adding a second `encoding/json` decoder or a lenient
`raw*` accessor. RPC outer envelopes remain owned by `internal/rpcwire`.
In particular,
`inventory.children.prefix` must be a JSON string, so `null` cannot become the
empty root prefix through Go's zero-value decoding behavior. Invalid request
shapes return `ErrInvalidRequest` without a success body. `objects.get` is
served through `Index.ObjectsGet`, which receives the requested namespace as
context; both the server entry and `FetchObjects` re-check schema-to-namespace
membership and refuse foreign objects with `integrity_failure`.

`FetchObjects` requested JSON is required back as `encoding: "json"`; a hostile
peer response labelled `cbor` is rejected as `ErrInvalidObject`. Exact member
names, duplicate/missing/extra members, and all six JSON value kinds are tested
through the production `Dispatch` and `FetchObjects` entries. The complete
decoder-site census and the narrowing/control-plant evidence are in
`CONFORMANCE-MATRIX.md` and the task-scoped evidence archive.

The schema table contains 19 included schema/version pairs and 42 known
excluded schema identities. A Blob Descriptor belongs to `manifest`, complete
raw content belongs to `blob`, and a Chunk is transfer material with no
inventory root. Local markers and transient request/response bodies are
excluded. An unknown immutable schema fails closed. Descriptor-as-record,
independent Chunk admission and a local marker are tested against both
membership and roots.

Four included rows stay fail-closed at `canonicaljson.VerifyObjectIdentity`
because their complete shapes are not implemented there:
`session-record@3.1.0`, `materialization-plan@1.0.0`,
`materialization-plan@2.0.0`, and `task-board-bundle@1.0.0`. Their destination
namespace is explicit in the mapping table, but `ClassifyJSON` does not admit
them. Each has a named admission-refusal test and isolated narrowing. The
orchestrator decision rev1 says “KEEP FAIL-CLOSED. Do not add shallow
validators.” Validator owners are `STORY-260830-4n0fo8` (Session Record),
`STORY-260830-2r137i` (both Materialization Plan versions), and
`STORY-260830-27pqyi` (Task-board Bundle).

`TestMixedNSExchangeIdentityLevelSyntheticIDs` drives the exact MIXED-NS-1
identity sets through production `inventory.roots`, `inventory.children`, and
recursive identity walking. It proves the normative missing set and six
converged roots, while making no object-validation claim. This split follows
§11.4: “The synthetic IDs stand for schema-valid objects/bytes in the row
shown; their contents are not used to compute the inventory trie beyond their
validated identities.” `TestInProcessExchangeWalksChangedNamespacesAndFetchesExactObjects`
is the separate real-byte proof: it validates fetched Checkpoint and Descriptor
objects, exercises the production membership/refusal paths, and pins six roots
computed from the real fixture identities. The two proofs do not claim a
single synthetic-ID/real-byte execution.

`Index` is the in-memory trie and validated-object engine. `DurableIndex`
(`OpenDurable`) persists schema-valid immutable JSON bytes by namespace and
digest, rebuilds the trie after restart, and retains conflicting byte variants
in quarantine. `DurableIndex.SyncFrom` compares common IDs as well as Merkle
differences, uses `MissingObjectIDs` to discover absent objects, and retrieves
them through `FetchObjects`/`objects.get`. It closes Tombstone/Acknowledgement
references after the complete received union exists. Both record classes stay
immutable data; sync does not execute Tombstones or call a session deletion
path.

The Story-final `TestDurableSyncGeneratedPerturbationProduct` drives 75,765
generated scenarios through `DurableIndex.SyncFrom`: 38,000 valid-union cases
and 37,765 generated same-identity/different-byte conflict cases. It crosses
N=1..6 fixtures (the N=6 fixture includes competing leases, a losing branch, a
Tombstone and its Acknowledgement), every arrival permutation for N≤5 and 34
deterministic N=6 orders, duplicate delivery, a later-filled gap, both
created-at orderings, every proper peer-object subset, and one-to-three sync
passes. Every feasible conflict-target × conflict-round pair (rounds 1, 2, 3)
is included at N=2..6; the conflicting peer also carries an identity missing
locally. An independent test oracle constructs all six namespace roots and
the expected lease projection for successful unions. Each completed union is
checked for exact object bytes; partial peers may not remove identities or
lower roots; an extra post-convergence sync must leave the entire durable
store byte-identical. Every generated conflict returns literal
`integrity_failure`, quarantines both exact byte variants, and aborts without
mutating the peer. `TestDurableSyncConflictWithPartialOverlapPeer` reproduces
the reviewer case both before and after a successful prior sync. The N=6 gap
that separates a Tombstone from its Acknowledgement refuses on the first pass
and converges after the peer fills the gap. N=6 order sampling, object size
above six, and peers that continue growing after pass 3 remain stated bounds.

The broad configured `go test ./...` run skips this parent test because the
75,765-case product is executed by bounded nested-subtest selectors. Selecting
the parent with an `-run` expression that includes the nested size/order path
executes the requested shard; the exact current-tree ranges and exits are in
the task results and shard manifest.

`Index.RebuildProjection` verifies exact presence of the `sessrepo` record and
authoritative event bytes in the union, derives every validated lease tuple
from the complete union through `sessquery.Reader.LeaseHeadsForSession`, and
passes them to the existing pure `sessstate.Projector`. Divergent event bytes
remain stored in the union; only the repository-owned authoritative event
chain reaches the reducer. The projection is invariant across all 720 orders
of six union objects under each of two opposing timestamp assignments (1,440
production `Index.RebuildProjection` cases). Its reference uses literal expected state, lease
tuple, and conflict classes. Structurally, the rebuild reads sorted identity
snapshots and validated records; neither `created_at`, wall-clock time, nor
map insertion order selects a lease. More precisely, the projection is a
function of the immutable union plus the fixed `sessrepo` authority snapshot;
this leaf does not reconstruct a new event-chain index from arbitrary union
objects.

The pinned specification states: “No last-writer-wins rule exists. Timestamps
MUST NOT select a winner.” Section 5.3 selects the greatest
`(epoch, lease_id)` tuple using bytewise UUID order and defines `created_at` as
diagnostic only. Crash/restart tests use child-process exit at object install
and quarantine persistence boundaries, then reopen and retry. These are
process-crash checks, not power-loss or physical-storage fault simulation.

This is a package library, not a public `ax sync` command or advertised doctor,
Host Channel, or network anti-entropy capability. Raw blob/chunk staging and
destination materialization stay at the §11.5–§11.6 boundary. RPC 3/4 inventory
serving remains outside `Index.New`'s RPC 2 contract. Board owners for those
boundaries are `STORY-260830-14qxuc` (chunk transfer) and
`STORY-260830-2r137i` (workspace materialization).

`New` accepts Mesh RPC 2.0.0 only. The pinned spec's §11.8 adds the RPC 3
`directory_record` namespace and §11.9 adds RPC 4
`terminal_backend_evidence`; these are outside this leaf's six-namespace RPC 2
serving contract and are refused by `Index.New`, not mislabeled as RPC 2
exclusions. RPC 5 authenticated host dispatch is also outside scope. The
package is not wired into the authenticated Host Channel handler.

## Commands and outputs

From the repository root:

| Purpose | Command | Output |
| --- | --- | --- |
| Package tests with the generated product selected separately | `go test ./internal/merkleinventory ./internal/sessquery -count=1` | Package logs under `.temp/TASK-260830-2h5uv9/`; crash fixtures use `t.TempDir()` |
| Generated reorder/duplicate/gap/skew/partial-peer and partial-overlap-conflict product | Example bounded shard: `go test ./internal/merkleinventory -run '^TestDurableSyncGeneratedPerturbationProduct$/^N5$/^skew[01]$/^order(000|001|002|003|004)$' -parallel=4 -count=1` | Task-scoped N/order/shard logs under `.temp/TASK-260830-2h5uv9/`; exact ranges and exits are in the task result |
| Windows cross-platform vet | `GOOS=windows GOARCH=amd64 go vet ./...` | Vet diagnostics under `.temp/TASK-260830-2h5uv9/` |
| Narrowing evidence | `python3 internal/merkleinventory/mutations.py --output .temp/TASK-260830-2h5uv9/mutations-final` | One raw test log per mutant, plus `results.json` and `table.md` |

See [CONFORMANCE-MATRIX.md](CONFORMANCE-MATRIX.md) for the gate/entry axis
census and [TRACEABILITY.md](TRACEABILITY.md) for clause-to-test bindings and
measured bounds.
