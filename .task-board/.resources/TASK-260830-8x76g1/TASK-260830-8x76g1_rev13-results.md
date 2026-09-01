# TASK-260830-8x76g1 rev13 developer results

## Scope

This revision is test-only, as required by the CR revision 12 verdict. It changes
`internal/canonicaljson/boundary_constraints_test.go` and does not change
`closed_shapes.go`, the pinned specification, README capability claims, or any
accepted leaf.

The production composition remains `prepareObjectIdentity` ->
`validateImmutableObjectShape`, reached by both `CalculateObjectIdentity` and
`VerifyObjectIdentity`. The added cases drive those two public identity entries
through `assertIdentityEntriesAcceptShape` and
`assertIdentityEntriesRefuseShape`.

## Review findings closed

| Finding | Executable proof |
| --- | --- |
| `maxBlobChunks` moved with its test | The public accept/refuse boundary now uses independent literals `32768` and `32769`; changing the production constant to `32769` is killed. The accepted object encodes below the 5,242,880-byte public identity cap. |
| NUL symlink target | A safe relative target is accepted; `sub/\u0000escape` is refused by both public entries. |
| absolute-slash symlink target | A safe relative target is accepted; `/etc/passwd` is refused by both public entries. |
| two-character Windows drive target | `C` is accepted; `C:` is refused by both public entries, independently pinning the `len(target) >= 2` branch. |
| first ManifestEntry comparison | Two bytewise-sorted directory entries are accepted; the same two entries reversed are refused by both public entries. |
| first WorkspaceSnapshot member comparison | Two workspace-ID-sorted members are accepted; the same two members reversed are refused by both public entries. |

## Derived mutation sweep

The attached CR rev12 `genmutants.py` was run against the whole current
`internal/canonicaljson/closed_shapes.go`. It mechanically generated 71 mutants,
including the `maxBlobChunks` and `maxChunkSize` constant declarations. The four
attached symlink clause mutants were also run, for the reviewer-equivalent total
of 75.

- 59 raw mutants were killed by uncached `go test ./internal/canonicaljson/ -count=1` runs.
- 16 raw mutants survived.
- 0 survivors are actionable.

The 16 raw survivors are the same reviewer-audited non-actionable set already
present in CR revision 12: mutations of non-behavioral error text; bounds
subsumed by stricter regex, traversal-count, coverage, positional-index, or
cross-field checks; and mechanically broad mutants for which the independent
public-entry boundary mutants were already killed. In particular, the CR rev12
verdict independently re-derived the surviving chunk coverage, zero-size,
extension-key minimum, logical-ID, submodule traversal, and uint32-index cases.
No new survivor appeared.

All six revision-12 findings are killed in the sweep:

- `const maxBlobChunks 32_768 -> 32_769`
- absolute symlink clause removal
- NUL symlink clause removal
- drive guard `len(target) >= 2 -> >= 3`
- ManifestEntry guard `index > 0 -> index > 1`
- WorkspaceSnapshot member guard `index > 0 -> index > 1`

The harness restored `closed_shapes.go` to its pre-sweep SHA-256 after every
mutant; the final `shasum -a 256 -c` exited 0.

## Validation

Every command below was run as a standalone foreground gate and exited 0.

| Gate | Result |
| --- | --- |
| tracked and untracked Go formatting check | exit 0 |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| focused canonicaljson tests | exit 0 |
| `go test ./... -count=1 -v` | exit 0 |
| `go test ./... -race -count=1` | exit 0 |
| `go test ./... -cover -count=1` | exit 0; canonicaljson 83.6%, scalar 90.1%, traceability 85.0% |
| `FuzzScalarProductionEntries`, fixed `100x`, `parallel=1` | exit 0; 37-seed baseline |
| `FuzzCanonicalizeRoundTrip`, fixed `100x`, `parallel=1` | exit 0; 27-seed baseline |
| `FuzzObjectIdentityRepresentationInvariant`, fixed `100x`, `parallel=1` | exit 0; 28-seed baseline |
| `FuzzClosedIdentityShapeRefusal`, fixed `100x`, `parallel=1` | exit 0; 73-seed baseline |
| `tracecheck` | exit 0; `assigned_scopes=0` |
| `tracecheck -section 17.3` | exit 0; `assigned_scopes=1` |
| `cataloggen -check` | exit 0 |
| Linux amd64 build | exit 0 |
| Windows amd64 build | exit 0 |
| tracked JSON parse gate with `pipefail` | exit 0 |
| `git diff --check` | exit 0 |
| accepted-leaf byte comparison | exit 0 |
| `task-board validate` | exit 0; 262 inherited `MISSING_ACTIVITY` diagnostics, none for this task |

This operation is read-only identity validation; durable-state crash recovery is
not applicable. No unsupported capability is advertised.
