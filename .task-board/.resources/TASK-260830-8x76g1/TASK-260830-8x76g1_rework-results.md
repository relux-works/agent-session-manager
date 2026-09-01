# TASK-260830-8x76g1 rework results

## Outcome

The reviewer-found identity attestation bypass is closed. Both production
entries, `CalculateObjectIdentity` and `VerifyObjectIdentity`, now pass through
`prepareObjectIdentity`, whose composed call to
`validateImmutableObjectShape` occurs after trusted schema/version selection
and before digest calculation or verification.

The scoped validator implements:

- the exact Blob Descriptor top-level shape and exact BlobChunk nested shape;
- BlobChunk zero-based index order, contiguous offsets, 1..4 MiB size bounds,
  fixed non-final chunk size, uint53 overflow protection, and exact whole-blob
  coverage;
- the exact Transfer Manifest top-level shape, every ManifestEntry variant,
  WorkspaceSnapshot and both member variants, and every recursively embedded
  closed Git type listed in Section 10.4;
- Transfer Manifest kind/tag invariants and sorted-unique manifest IDs and
  exclusion classes; and
- the closed `extensions["works.relux.ax.migrated-from"]` identity
  contribution from Section 17.3.

Section 10.1 deliberately does not receive a fabricated universal top-level
whitelist: the normative record envelope explicitly admits schema-specific
fields. Its existing schema/version identity resolution and common scalar
validation remain the scoped implementation evidence.

## Negative evidence

Attack tests drive both production identity entries. Verification inputs carry
the correct omit-self digest, so a refusal cannot be explained by a mismatched
claim. The tests require refusal for unknown Blob/Manifest top-level members,
unknown BlobChunk, ManifestEntry, WorkspaceSnapshot, WorkspaceSnapshotMember,
GitRemote, GitHead, GitObjectPack, GitIndex, GitIndexEntry, GitSubmodule, and
GitFeatures members; malformed migration provenance; BlobChunk index, offset,
size, fixed-partition, bounds, and coverage violations; invalid tagged-kind
combinations; and unsorted/duplicate arrays.

`FuzzClosedIdentityShapeRefusal` carries a committed seed corpus for the prior
review shapes and deterministically injects a still-invalid shape for every
mutation. The older scalar and JCS/identity fuzz targets remain bounded at
`100x` and were rerun.

## Validation

Every final gate below was run as a standalone process with direct log
redirection; every listed exit code is 0.

| Gate | Exit | Evidence |
| --- | ---: | --- |
| Focused canonical tests | 0 | `rework-01-canonical-test.log` |
| Canonical race tests | 0 | `rework-02-canonical-race.log` |
| Canonical coverage (80.7%) | 0 | `rework-03-canonical-cover.log` |
| Canonicalization fuzz, fixed 100x | 0 | `rework-04-fuzz-canonical.log` |
| Identity invariance fuzz, fixed 100x | 0 | `rework-05-fuzz-identity.log` |
| Closed-shape refusal fuzz, fixed 100x | 0 | `rework-06-fuzz-closed-shape.log` |
| Scalar production fuzz, fixed 100x | 0 | `rework-07-fuzz-scalar.log` |
| Tracecheck Sections 1.6, 10.1-10.4, 17.3 | 0 | `rework-08-trace-scoped.log` |
| Traceability tests and narrowed-owner mutants | 0 | `rework-09-trace-tests.log` |
| Generated catalog check | 0 | `rework-10-catalog-check.log` |
| Full repository verbose tests | 0 | `rework-11-full-test.log` |
| Full repository coverage | 0 | `rework-12-full-coverage.log` |
| `go vet ./...` | 0 | `rework-13-vet.log` |
| `go build ./...` | 0 | `rework-14-build.log` |
| Linux amd64 canonical test cross-build | 0 | `rework-15-linux-cross-test-build.log` |
| Windows amd64 canonical test cross-build | 0 | `rework-16-windows-cross-test-build.log` |
| `git diff --check` | 0 | `rework-17-diff-check.log` |
| `task-board validate` | 0 | `rework-18-board-validate.log` |

`task-board validate` still reports 262 inherited `MISSING_ACTIVITY`
diagnostics and exits 0; the current task is not among those diagnostics.

During development, three expected-red classes were observed and reported with
their real non-zero statuses: initial `gofmt` exposed a syntax error (exit 2),
the old synthetic closed-schema fixture failed after the gate was introduced
(focused test exit 1), and tracecheck rejected each reviewed-registry edit until
its explicit projection digest was updated (exit 1). All were corrected before
the final standalone reruns above.

## Capability boundary

This implementation is read-only and deterministic. Durable-state crash or
idempotency recovery is not applicable. It does not claim migration
publication, atomic reference advancement, rollback retention, `ax migrate`,
`ax doctor`, or any runtime capability.
