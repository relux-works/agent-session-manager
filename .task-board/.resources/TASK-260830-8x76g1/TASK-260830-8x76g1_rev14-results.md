# TASK-260830-8x76g1 rev14 developer results

## Source binding and scope

- Base commit: `ad7275181ca82fc3fa29544e3893923a92d7b9d5`.
- Prior reviewed candidate tree (CR revision 13): `7769a2c43ec8c6f224e3fbf6eeb1ca402747302d`.
- Rev14 candidate tree from a private index: `0ca65322a7bf171b8e51bef353250a35d4f094f5`.
- Rev13 -> rev14 delta: two files, +49/-4:
  - `internal/canonicaljson/boundary_constraints_test.go`
  - `internal/canonicaljson/testdata/constraint-enumeration.md`
- No production file, pinned specification, README capability claim, validation configuration, or accepted leaf changed in rev14.

The accepted-leaves attachment matched its declared SHA-256
`9ae5e624addc7d954c391d96cab7c9f7aac3e6bcee58c76dd0cc95533d0eac9c`.
`internal/catalog`, `internal/cataloggen`, `internal/specpin`, and the seven
accepted scalar files were compared byte-for-byte with the extracted archive;
all four comparison gates exited 0.

## Finding closed

`TestTrailingBlobChunkSizeMinimumReachesBothIdentityEntries` now pairs an
accepted final chunk of one byte with a trailing zero-size chunk in a non-empty
Blob Descriptor. The malformed descriptor has a valid preceding 4 MiB chunk,
contiguous offsets, and exact total coverage if the lower-bound guard is
disabled. It therefore reaches `validateBlobDescriptor` at
`closed_shapes.go:614` instead of being refused by the empty-descriptor rule.

Both public entries are driven:

- `CalculateObjectIdentity` must refuse with the exact `BlobChunk[1] size must
  lie in [1, 4194304]` reason.
- `VerifyObjectIdentity` receives a correct omit-self claim and must refuse for
  the same reason.

The old misleading `blob chunk size non-zero` table case was replaced by the
constraint it really exercised: an empty Blob Descriptor must contain no
chunks. The `BlobChunk.size` enumeration row now names the dedicated public-entry
test and no longer leaves room for a positional/coverage subsumption claim.

## Negative and mutation evidence

With only `if size == 0` changed to `if false && size == 0`, the dedicated test
failed with exit 1 because `CalculateObjectIdentity` returned no error. The
production file was restored from a task-scoped byte copy and verified before
any later gate.

The rev12 reviewer `genmutants.py` was re-run over the complete current
`closed_shapes.go`, including constant declarations. It derived 71 mutants;
the four reviewer symlink-clause mutants brought the executed total to 75.
The final sweep was sequential against the final test tree and used uncached
`go test ./internal/canonicaljson/ -count=1` for every mutant.

| Mutation result | Count | Interpretation |
| --- | ---: | --- |
| Killed | 60 | Includes the trailing zero-size guard and all four symlink clauses. |
| Raw survivors | 15 | Same mechanically redundant or non-behavioral set independently classified in CR revision 13. |
| Actionable survivors | 0 | Every raw survivor is subsumed, jointly pinned, narrowing-only, or error-text-only. |

The production SHA-256 after the sweep is
`3d34cb52cb483e513a8ba398ed351993a02c46c01d5f4086af083f038304b307`,
identical to the pre-sweep hash. Detailed mechanisms are recorded in
`TASK-260830-8x76g1_rev14-mutation-summary.md`.

## Validation

Every command below ran as a standalone foreground process. Passing gates have
exit code 0; the one expected-red mutant gate is reported as the real failure
that it is.

| Gate | Exit | Result |
| --- | ---: | --- |
| Final focused boundary tests | 0 | Both declared-bound table and dedicated trailing-chunk test passed. |
| Zero-size guard disabled (expected red) | 1 | Dedicated public-entry test failed because malformed input was attested. |
| Derived 71-mutant sweep | 0 | 71 results, no setup/compile failure; zero-size guard killed. |
| Four symlink-clause mutants | 0 | 4/4 killed. |
| Sweep restoration SHA check | 0 | Production source restored byte-for-byte. |
| Configured Go formatting check | 0 | No formatted-file drift. |
| `go build ./...` | 0 | Build passed. |
| `go vet ./...` | 0 | Vet/lint passed. |
| `go test ./... -count=1 -v` | 0 | All eight packages passed. |
| `go test ./... -race -count=1` | 0 | Race suite passed. |
| `go test ./... -cover -count=1` | 0 | canonicaljson 83.6%; scalar 90.1%; traceability 85.0%. |
| `FuzzScalarProductionEntries -fuzztime=100x -parallel=1` | 0 | 37 seed corpus entries; 100 executions. |
| `FuzzCanonicalizeRoundTrip -fuzztime=100x -parallel=1` | 0 | 28 seed entries; 114 executions, one new interesting input. |
| `FuzzObjectIdentityRepresentationInvariant -fuzztime=100x -parallel=1` | 0 | 30 seed entries; 100 executions, one new interesting input. |
| `FuzzClosedIdentityShapeRefusal -fuzztime=100x -parallel=1` | 0 | 74 seed entries; 100 executions. |
| `tracecheck` | 0 | `assigned_scopes=0`. |
| `tracecheck -section 17.3` | 0 | `assigned_scopes=1`. |
| `cataloggen -check` | 0 | Generated catalog is current. |
| Linux amd64 build | 0 | Cross-platform build passed. |
| Windows amd64 build | 0 | Cross-platform build passed. |
| Tracked JSON parse with `pipefail` | 0 | Every tracked JSON document parsed. |
| `task-board validate` | 0 | 262 inherited `MISSING_ACTIVITY` diagnostics; this task is not listed. |
| `git diff --check` | 0 | No whitespace errors. |

This task validates read-only identity calculation and verification. Durable
state mutation and crash recovery are not applicable, and no unsupported
runtime capability is advertised.
