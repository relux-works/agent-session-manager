# TASK-260830-8x76g1 — recovery validation rerun

## Outcome

The revision-6 candidate is ready for review. The composed production path is:

`CalculateObjectIdentity|VerifyObjectIdentity -> prepareObjectIdentity -> validateImmutableObjectShape -> immutableObjectShapeValidators[schema,version]`

The schema/version validator registry is total over
`catalog.Current().SelfIdentities`. Session Record 1.0, Blob Descriptor 1.0,
and Transfer Manifest 1.0 use concrete closed-shape validators. Every other
registered identity resolves to an explicit fail-closed validator rather than
falling through from catalog recognition to attestation.

The candidate retains the native scalar, canonicalization, representation
invariance, and closed-shape fuzz targets with fixed `100x`, `parallel=1`
validation commands and committed review-derived seed corpora. Negative tests
drive both public identity entries for registry incompleteness, malformed common
record envelopes, unknown top-level and nested members, invalid chunk coverage,
Unicode character bounds, invalid dates and identifiers, TM-GIT-N2 stage 4,
and recursive Transfer Manifest Git/submodule rules.

## Recovery and archive binding

- Accepted-leaves archive SHA-256:
  `9ae5e624addc7d954c391d96cab7c9f7aac3e6bcee58c76dd0cc95533d0eac9c`.
- Audit carry-forward archive SHA-256:
  `df8d8fa712e0ce85bde4776dc1f83cef9d373593e76fd1145ce9d85b593ce4ee`.
- The archives were extracted accepted-first into a task-local comparison
  overlay. Every unchanged archive file matches the workspace byte-for-byte.
  The only nine differences are the later task-owned README, validator,
  test/golden, traceability, and validation-config deltas from revisions 4–6.
  No accepted content was re-derived or overwritten.

The failed CR revision-5 validation log invoked
`test -z "$(gofmt -l .)"` and stopped at command 1/13 with exit 1. A direct
expected-red rerun also returned exit 1 and identified only four ignored Go
review probes under `.temp/`; no production Go file was listed. The probes were
mechanically formatted, after which both the old base command and the candidate
command that excludes ignored scratch returned exit 0. This recovery successor
made no new repository source delta.

## Direct foreground validation

Every command was run directly as a standalone foreground process. No gate was
backgrounded or piped through `tee`. The JSON pipeline was run with `pipefail`.

| Command | Exit | Evidence |
| --- | ---: | --- |
| focused canonicaljson + scalar tests | 0 | `focused-tests-01.log` |
| old base `gofmt -l .` gate after scratch formatting | 0 | `format-base-gate-01.log` |
| candidate tracked/non-ignored gofmt gate | 0 | `format-candidate-gate-01.log` |
| `go build ./...` | 0 | `go-build-01.log` |
| `go vet ./...` | 0 | `go-vet-01.log` |
| `go test ./... -count=1 -v` | 0 | `go-test-all-01.log` |
| `go test ./... -race -count=1` | 0 | `go-test-race-01.log` |
| `go test ./... -cover -count=1` | 0 | `go-test-cover-01.log` |
| `FuzzScalarProductionEntries`, fixed 100x | 0 | `fuzz-scalar-01.log` |
| `FuzzCanonicalizeRoundTrip`, fixed 100x | 0 | `fuzz-canonical-roundtrip-01.log` |
| `FuzzObjectIdentityRepresentationInvariant`, fixed 100x | 0 | `fuzz-identity-invariant-01.log` |
| `FuzzClosedIdentityShapeRefusal`, fixed 100x | 0 | `fuzz-closed-shape-01.log` |
| global tracecheck | 0 | `tracecheck-global-01.log` |
| scoped tracecheck for 1.6, 10.1–10.4, 17.3 | 0 | `tracecheck-scoped-01.log` |
| catalog generator check | 0 | `cataloggen-check-01.log` |
| Linux amd64 production build | 0 | `go-build-linux-01.log` |
| Windows amd64 production build | 0 | `go-build-windows-01.log` |
| Linux canonicaljson test compile | 0 | `canonicaljson-linux-test-compile-01.log` |
| Windows canonicaljson test compile | 0 | `canonicaljson-windows-test-compile-01.log` |
| tracked JSON validation with `pipefail` | 0 | `json-validation-01.log` |
| `curator status --check` | 0 | `curator-status-01.log` |
| `task-board validate` | 0 | `task-board-validate-01.log` |
| `git diff --check` | 0 | `git-diff-check-01.log` |

Coverage is 80.6% for `internal/canonicaljson` and 90.1% for
`internal/scalar`. Scoped tracecheck reports
`contracts=60 normative_sections=36 acceptance_cases=29 fixtures=30 compatibility_contracts=55 assigned_scopes=6`.

`task-board validate` returned exit 0 with 262 inherited
`MISSING_ACTIVITY` diagnostics and no diagnostic for this task. The operation is
read-only and does not mutate durable state, so durable crash/idempotency
recovery evidence is not applicable. README makes no migration-publication or
atomic-reference capability claim.
