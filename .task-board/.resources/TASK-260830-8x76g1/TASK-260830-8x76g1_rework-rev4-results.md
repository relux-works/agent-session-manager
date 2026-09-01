# TASK-260830-8x76g1 — developer rework revision 4

## Outcome

CR revision 3's narrowed recursive Transfer Manifest gate is repaired by a
systematic derivation from pinned SPEC.md Sections 1.6 and 10.1–10.4, rather
than by patching only the seven reviewer examples. The full derivation and the
boundary between candidate-local and referenced-data constraints is recorded
in `TASK-260830-8x76g1_constraint-enumeration.md`.

Both public entries use the same production call site before calculation or
attestation:

`CalculateObjectIdentity|VerifyObjectIdentity -> prepareObjectIdentity -> validateImmutableObjectShape`

## Implemented scope

- Added production scalar validators for `git-oid`, object-format-bound Git
  OIDs, fully qualified `git-ref`, and sanitized Git URLs; their constructors
  and JSON read/write paths are covered by unit tests and the existing native
  scalar fuzz target.
- Completed candidate-local Transfer Manifest value validation: workspace
  member paths/identities/policies, remote bounds/order/URLs, GitHead union,
  GitObjectPack types, GitIndex version/count/path-stage order/TM-GIT-N2,
  GitIndexEntry scalar/boolean rules, GitFeatures sparse-pair state, Git object
  format consistency, and recursive GitSubmodule nullability, parent gitlink,
  depth 16, total 256, cycle and sibling-path rules.
- Completed reachable ManifestEntry cross-field checks: destination case
  collision, lexical symlink containment, and earlier-file/same-mode hardlink
  targets.
- Enforced the 5,242,880-byte encoded immutable-object maximum before decode.
- Extended deterministic closed-shape fuzz mutations with nested bounds,
  malformed digest/ref/URL, stage 4, impossible submodule state, sparse-pair,
  symlink/hardlink and index-count attacks. The configured suite still invokes
  every repository fuzz target exactly once at fixed `100x`, `parallel=1`.
- Narrowed README claims to self-contained rules reproducible through both
  public entries. Referenced Blob Descriptor/child-manifest/Git pack/index and
  filesystem checks are explicitly not claimed by this package.

## Negative and boundary evidence

Before implementation, the focused production-entry regression command exited
`1`: all 21 initial malformed nested values received an identity. The saved
`expected-red-nested-values.log` records that failure.

After implementation, the focused entry test exits `0` and covers the reviewer
examples plus derived ordering/cross-field rules, exact valid multibyte limits,
recursive depth/count/state, and encoded-size refusal. Exact 256-character
repository/tree/submodule identities, 128-character remote names, 4096-character
symlink targets, and 255-character media types are accepted by both entries;
their over-bound forms refuse. Invalid calendar dates and malformed UUID/digest,
Git OID/ref/URL values refuse before identity calculation and verification.

## Validation

Every command below ran as a standalone foreground process. Exit codes are the
actual process results.

| Gate | Exit | Result |
| --- | ---: | --- |
| Expected-red focused production-entry regression before fix | 1 | Expected failure: 21 nested invalid shapes were attested. |
| Focused production-entry regression after fix | 0 | Pass |
| Cross-platform golden identity + RFC 8785 UTF-16 order | 0 | Pass |
| `go test ./... -count=1` | 0 | Pass |
| `go test ./... -v -count=1` | 0 | Pass |
| `go test ./... -cover -count=1` | 0 | Pass |
| Focused `-race` scalar/canonical tests | 0 | Pass |
| Focused package coverage | 0 | scalar 90.1%; canonicaljson 81.5% |
| `FuzzScalarProductionEntries`, fixed `100x`, `parallel=1` | 0 | Pass |
| `FuzzCanonicalizeRoundTrip`, fixed `100x`, `parallel=1` | 0 | Pass |
| `FuzzObjectIdentityRepresentationInvariant`, fixed `100x`, `parallel=1` | 0 | Pass |
| `FuzzClosedIdentityShapeRefusal`, fixed `100x`, `parallel=1` | 0 | Pass; 48 seed corpus entries loaded |
| Scoped tracecheck 1.6, 10.1–10.4, 17.3 | 0 | `assigned_scopes=6` |
| Catalog generated-output check | 0 | Pass |
| `go build ./...` | 0 | Pass |
| `go vet ./...` | 0 | Pass |
| Linux amd64 canonicaljson test cross-compile | 0 | Pass |
| Windows amd64 canonicaljson test cross-compile | 0 | Pass |
| `gofmt -l internal` | 0 | Empty output |
| `git diff --check` | 0 | Pass |
| `task-board validate` | 0 | 262 inherited `MISSING_ACTIVITY` diagnostics; current task absent |

## Scope boundary

This package is read-only and deterministic, so crash/idempotency evidence for
durable mutation is not applicable. Section 17.3 evidence covers only the
immutable migrated-from identity contribution. It does not claim migration
publication, atomic reference advancement, rollback retention, `ax migrate`,
`ax doctor`, or a runtime capability.

