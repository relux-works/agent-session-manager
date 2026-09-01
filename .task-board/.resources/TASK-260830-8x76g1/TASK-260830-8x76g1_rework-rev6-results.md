# TASK-260830-8x76g1 — developer rework revision 6 results

## Candidate and normative binding

- Normative source: `relux-works/agent-session-manager-spec` v0.5.0 at
  `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, Sections 1.6, 10.1–10.4,
  and 17.3.
- Accepted-leaves archive SHA-256:
  `9ae5e624addc7d954c391d96cab7c9f7aac3e6bcee58c76dd0cc95533d0eac9c`.
- Audit-rework carry-forward archive SHA-256:
  `df8d8fa712e0ce85bde4776dc1f83cef9d373593e76fd1145ce9d85b593ce4ee`.
- Both archives were extracted in accepted-then-carry-forward order into a
  task-local overlay. Before this revision's edits, all 65 overlay files
  compared byte-for-byte with the provisioned workspace (exit 0); no archive
  content was re-derived.

## Production behavior

The composed public path is:

`CalculateObjectIdentity|VerifyObjectIdentity -> prepareObjectIdentity -> validateImmutableObjectShape -> immutableObjectShapeValidators[schema,version]`

This revision removes the extension-only default path. Every schema/version in
`catalog.Current().SelfIdentities` now resolves to an explicit validator. A
registry-derived completeness check fails on a missing, extra, or nil entry, so
catalog growth cannot silently create an attestation bypass.

Supported identity shapes at this boundary are now explicit:

- Session Record `1.0.0`: exact top-level Section 10.1 envelope and complete
  nested Launch Plan, Task-board Reference, Board Identity, Board Goal, and
  Fork Provenance closed/tagged shapes.
- Blob Descriptor `1.0.0`: complete Section 10.2 closed and recursive
  `BlobChunk` validation.
- Transfer Manifest `1.0.0`: complete reachable Section 10.4 closed, recursive,
  ordering, uniqueness, bounds, tagged-union, and cross-field validation.

Remaining registered schemas are not inferred to be safe from catalog
membership. Other Section 10.1 records validate their common envelope and then
return an explicit unsupported-shape refusal; all other registered identities
also have an explicit refusal row. `README.md` states this support boundary and
does not advertise publication, atomic-reference migration, or unimplemented
shape support.

## Negative and compatibility evidence

- `TestImmutableObjectShapeValidatorCompletenessRejectsDeletedRegisteredSchema`
  derives the expected key set from the catalog and deletes one row. The gate
  fails, proving totality without circularly enumerating the production table.
- `TestSessionRecordV1ClosedShapeReachesBothIdentityEntries` attacks missing
  `created_by_host_id`, malformed `subject_id`, an impossible `created_at`, an
  unknown top-level member, and an unknown nested Launch Plan member through
  both calculation and verification.
- `TestSessionRecordV1NestedTaggedShapesReachBothIdentityEntries` exercises
  valid and invalid direct/task-board/primary-goal/fork variants through both
  public entries.
- `TestUnsupportedSection10RecordSchemasValidateCommonEnvelopeBeforeRefusal`
  proves all remaining Section 10.1 record keys reject malformed common UUIDs
  before their explicit unsupported-shape refusal.
- `FuzzClosedIdentityShapeRefusal` has committed deterministic seeds for the
  five Session Record attacks above in addition to unknown Blob/Manifest
  members, chunk order/bounds/coverage, malformed scalar identifiers,
  TM-GIT-N2 stage 4, Unicode bounds, nested Transfer Manifest unions, and
  submodule state/depth/cycle cases.
- The cross-platform golden fixture contains full valid Session Record
  representations with LF/CRLF, insignificant whitespace, and reordered
  top-level/nested keys. The stable expected identity is
  `sha256:b3cfa25ae57de833d64361e86ede48691b0470610525ddfd1edfa1d03b34504b`.
- Linux and Windows production builds and `canonicaljson` test-binary
  cross-compiles passed.

Expected-red evidence was recorded with its real failure status:

- Before the total registry existed, the focused canonicaljson test build
  failed with exit 1 on undefined completeness/registry symbols.
- After trace declarations changed, tracecheck failed with exit 1 because the
  ownership projection digest changed from the reviewed digest to
  `30403d58757c36e9d0e5c5849c17105684b3fbd5d1f80cb5b57fd3d227a57a65`.
  The reviewed constant was updated only after this refusal was observed.

## Direct validation results

Every command below was run as a standalone foreground process. No gate was
backgrounded or piped through `tee`.

| Command | Exit | Evidence |
| --- | ---: | --- |
| tracked/non-ignored `gofmt` check | 0 | `gofmt-check-final.log` |
| `go build ./...` | 0 | `go-build-final.log` |
| `go vet ./...` | 0 | `go-vet-final.log` |
| `go test ./... -count=1 -v` | 0 | `go-test-all-final.log` |
| `go test ./... -race -count=1` | 0 | `go-test-race-final.log` |
| `go test ./... -cover -count=1` | 0 | `go-test-cover-final.log` |
| `FuzzScalarProductionEntries`, fixed `100x`, `parallel=1` | 0 | `fuzz-scalar-final.log` |
| `FuzzCanonicalizeRoundTrip`, fixed `100x`, `parallel=1` | 0 | `fuzz-canonical-roundtrip-final.log` |
| `FuzzObjectIdentityRepresentationInvariant`, fixed `100x`, `parallel=1` | 0 | `fuzz-identity-invariant-final.log` |
| `FuzzClosedIdentityShapeRefusal`, fixed `100x`, `parallel=1` | 0 | `fuzz-closed-shape-final.log` |
| global tracecheck | 0 | `tracecheck-final.log` |
| tracecheck Sections 1.6, 10.1–10.4, 17.3 | 0 | `tracecheck-scoped-final.log` |
| catalog generator `-check` | 0 | `cataloggen-check-final.log` |
| Linux amd64 build | 0 | `go-build-linux-final.log` |
| Windows amd64 build | 0 | `go-build-windows-final.log` |
| Linux amd64 canonicaljson test compile | 0 | `canonicaljson-linux-test-build-final.log` |
| Windows amd64 canonicaljson test compile | 0 | `canonicaljson-windows-test-build-final.log` |
| tracked JSON validation with `pipefail` | 0 | `json-validation-final.log` |
| `curator status --check` | 0 | `curator-status-final.log` |
| `task-board validate` | 0 | `task-board-validate-final.log` |
| `git diff --check` | 0 | `git-diff-check-final.log` |

Coverage for the changed identity package is 80.6%; scalar production
validators are 90.1%. Full package coverage is preserved in the attached log.
The scoped trace report is:
`contracts=60 normative_sections=36 acceptance_cases=29 fixtures=30 compatibility_contracts=55 assigned_scopes=6`.

The first configured format attempt returned exit 1 because `gofmt -l .`
included gitignored reviewer probes under `.temp/`. Those evidence files were
not rewritten. The validation command now enumerates tracked plus non-ignored
Go sources, including new untracked production packages while excluding the
repository's scratch contract; its direct rerun returned exit 0.

`task-board validate` returned exit 0 while reporting 262 inherited
`MISSING_ACTIVITY` diagnostics elsewhere on the board. This task does not claim
those board-wide diagnostics were repaired. The pre-existing untracked
`.DS_Store` was not modified or absorbed.

This identity operation is read-only and does not mutate durable state, so
crash/idempotent recovery evidence for a durable mutation is not applicable.
