# TASK-260830-8x76g1 rework revision 8 results

## Outcome

The candidate is ready for developer handoff after addressing the revision 7
review findings and the revision 8 Change Request formatting-gate stall.

- `validateSessionRecordV1`, reached through
  `prepareObjectIdentity -> validateImmutableObjectShape`, now applies the
  Section 2.1 Session Record name grammar before both
  `CalculateObjectIdentity` and `VerifyObjectIdentity` attest the object.
  A 64-character ASCII name is accepted; 65 characters, a leading hyphen,
  whitespace, a control character, and non-ASCII text are refused through both
  public entries.
- `validateManifestEntries` replaces the quadratic prior-entry scan with a
  Unicode simple-fold set. The 65,536-entry `workspace_tree` production
  calculation measured 0.31 seconds in this run, versus the review evidence's
  6.61-second quadratic baseline, and remains guarded by a less-than-two-second
  regression test outside race instrumentation.
- `TestConstraintEnumerationMatchesRequireExactMembers` derives every declared
  member from production `requireExactMembers` calls and requires a one-to-one
  row with a production call site and pinned declaration in
  `internal/canonicaljson/testdata/constraint-enumeration.md`.
- The configured format gate now limits `gofmt -l` to tracked and non-ignored
  repository Go files. This prevents task-scoped `.temp` evidence and restored
  archive copies from creating a false formatting failure while still checking
  every candidate Go file.
- README claims are limited to candidate-local identity validation. Rules that
  require referenced descriptors/manifests, raw Git bytes, filesystem
  resolution, or global creation authority remain explicitly external.

## Source binding

- Accepted-leaves archive SHA-256:
  `9ae5e624addc7d954c391d96cab7c9f7aac3e6bcee58c76dd0cc95533d0eac9c`.
- Audit carry-forward archive SHA-256:
  `df8d8fa712e0ce85bde4776dc1f83cef9d373593e76fd1145ce9d85b593ce4ee`.
- Pinned specification: tag `v0.5.0`, commit
  `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, `SPEC.md` SHA-256
  `562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`.
- The combined archive tree contains 65 files: 56 remain byte-identical,
  9 carry the deliberate revision 8 delta, and none are missing.

## Negative evidence

- Narrowing `sessionNamePattern` from a 64-character maximum to 63 caused
  `TestSessionRecordNameGrammarReachesBothIdentityEntries` to fail with exit 1
  because the valid 64-character production object was refused. Restoring the
  production bound made the same test pass with exit 0, including all five
  invalid-name refusals.
- Removing only the `Session Record 1.0.0.name` artifact row caused
  `TestConstraintEnumerationMatchesRequireExactMembers` to fail with exit 1
  and the exact message `missing artifact row Session Record 1.0.0.name`.
  Restoring the row made the same test pass with exit 0.
- The closed-shape fuzz corpus drives unknown top-level/nested members,
  invalid chunk order/bounds/coverage, Session Record envelope/name mutations,
  Transfer Manifest nested shapes, malformed Git values, and TM-GIT-N2 stage 4
  through both public identity entries.

## Direct validation

Every command below ran as a foreground standalone process. No gate was piped
through `tee`; the JSON pipeline ran with `pipefail` enabled.

| Command | Exit | Evidence |
| --- | ---: | --- |
| configured repository-only `gofmt -l` gate | 0 | `gofmt-check-01.log` |
| `go build ./...` | 0 | `go-build-01.log` |
| `go vet ./...` | 0 | `go-vet-01.log` |
| `go test ./... -count=1 -v` | 0 | `go-test-all-01.log` |
| `go test ./... -race -count=1` | 0 | `go-test-race-01.log` |
| `go test ./... -cover -count=1` | 0 | `go-test-cover-01.log` |
| `FuzzScalarProductionEntries`, fixed `100x` | 0 | `fuzz-scalar-01.log` |
| `FuzzCanonicalizeRoundTrip`, fixed `100x` | 0 | `fuzz-canonical-roundtrip-01.log` |
| `FuzzObjectIdentityRepresentationInvariant`, fixed `100x` | 0 | `fuzz-identity-invariant-01.log` |
| `FuzzClosedIdentityShapeRefusal`, fixed `100x` | 0 | `fuzz-closed-shape-01.log` |
| scoped `tracecheck -section 17.3` | 0 | `tracecheck-17.3-01.log` |
| assigned-scope tracecheck for 1.6, 10.1-10.4, 17.3 | 0 | `tracecheck-assigned-scope-01.log` |
| full tracecheck | 0 | `tracecheck-all-01.log` |
| catalog generator `-check` | 0 | `cataloggen-check-01.log` |
| `go generate ./internal/catalog` | 0 | `go-generate-catalog-01.log` |
| Linux amd64 build | 0 | `go-build-linux-amd64-01.log` |
| Windows amd64 build | 0 | `go-build-windows-amd64-01.log` |
| tracked JSON parse gate with `pipefail` | 0 | `json-validation-01.log` |
| `task-board validate` | 0 | `task-board-validate-01.log` |
| `git diff --check` | 0 | `git-diff-check-01.log` |
| post-mutant canonicaljson package | 0 | `canonicaljson-post-mutants-01.log` |
| post-mutant `go build ./...` | 0 | `go-build-post-mutants-01.log` |
| narrowed name-bound mutant | 1, expected red | `name-bound-narrowing-mutant-expected-red-01.log` |
| missing inventory-row mutant | 1, expected red | `member-inventory-missing-row-mutant-expected-red-01.log` |

Coverage on the current tree is 80.8% for `internal/canonicaljson`, 90.1% for
`internal/scalar`, and at least 79.3% for every other tested package. The
65,536-entry production regression passed in 0.31 seconds.

`task-board validate` truthfully exited 0 while reporting 262 inherited
`MISSING_ACTIVITY` diagnostics for legacy board elements. This candidate did
not attempt to rewrite or backfill those infrastructure records.

## Scope boundary

Identity calculation is read-only and deterministic, so there is no durable
mutation or crash-recovery operation to exercise. This task does not claim
migration publication, atomic-reference advancement, rollback retention,
runtime migration, `ax migrate`, `ax doctor`, or any new runtime capability.
