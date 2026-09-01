# TASK-260830-8x76g1 developer results — RUN-260901-0e2e10

## Outcome

CR revision 15's sole remaining finding is closed by a test-only delta. The new
`TestSymlinkTargetLowerBoundReachesBothIdentityEntries` proves the lower edge of
`ManifestEntry.symlink.target`'s `string[1..4096]` bound through both public
identity entries:

- `target: "a"` is accepted by `CalculateObjectIdentity` and
  `VerifyObjectIdentity`.
- `target: ""` is refused by both entries with
  `member target must be a non-empty UTF-8 string`.

The composed production path under test is
`CalculateObjectIdentity` / `VerifyObjectIdentity` ->
`prepareObjectIdentity` -> `validateImmutableObjectShape` ->
`validateTransferManifest` -> `validateManifestEntries` -> `requireString`.
No production source, validation configuration, README capability claim,
accepted leaf, or pinned specification changed.

The per-member constraint artifact now names this executable accept/refuse
boundary proof. Relative to reviewed CR rev15 tree
`1617e6afc56110589f6f0f524ceb94de192b43b5`, the candidate changes exactly:

- `internal/canonicaljson/clause_refusal_test.go`: +14 lines.
- `internal/canonicaljson/testdata/constraint-enumeration.md`: +1/-1 row text.

Candidate tree: `b4ae5fb03cab8481b83d6f8dd33d07db26322618`.
`internal/canonicaljson/closed_shapes.go` remained byte-identical at SHA-256
`3d34cb52cb483e513a8ba398ed351993a02c46c01d5f4086af083f038304b307`
before and after every mutation run.

## Mutation evidence

- Reviewer rev15 110-site refusal corpus rerun in two foreground halves:
  harness exits 0/0, 110 rows, 86 KILLED / 24 SURVIVED.
- Site 98 / line 1748 changed from SURVIVED to KILLED. A direct diff against
  reviewer rev15 results shows this as the only status change.
- Reviewer rev12 `genmutants.py` regenerated all 71 numeric/bound mutants,
  including constant declarations. The JSON is byte-identical to reviewer
  rev15 at SHA-256
  `f55bb35f5374357e9ee23e1b5a6b06df72c1293aaac73e336d8563684c7cf7bd`.
- The numeric sweep ran in two foreground halves with exits 0/0: 56 KILLED /
  15 previously-triaged raw survivors. Its result file is byte-identical to
  the prior independently reviewed result at SHA-256
  `6ea2a753d056a81cc50d53d28cc15fb0ca848db32c19dbee7de7a21fb2eb0840`.
  No actionable mutant survives.

## Validation run directly by this developer

| Command | Exit | Evidence |
| --- | ---: | --- |
| Focused symlink lower-bound test | 0 | `01-symlink-lower-bound-green.log` |
| Configured gofmt check | 0 | `02-gofmt.log` |
| `go build ./...` | 0 | `03-go-build.log` |
| `go vet ./...` | 0 | `04-go-vet.log` |
| `go test ./... -count=1 -v` | 0 | `05-go-test-full.log` |
| `go test ./... -race -count=1` | 0 | `06-go-test-race.log` |
| `go test ./... -cover -count=1` | 0 | `07-go-test-cover.log` |
| Four configured fuzz targets, each fixed at `100x`, `parallel=1` | 0 / 0 / 0 / 0 | `08` through `11` fuzz logs |
| General tracecheck | 0 | `12-tracecheck.log` |
| `tracecheck -section 17.3` | 0 | `13-tracecheck-section-17.3.log` |
| Catalog freshness check | 0 | `14-catalog-check.log` |
| Linux amd64 build | 0 | `15-build-linux-amd64.log` |
| Windows amd64 build | 0 | `16-build-windows-amd64.log` |
| Tracked JSON parse with `pipefail` | 0 | `17-json-parse.log` |
| `task-board validate` | 0 | `18-task-board-validate.log` |
| `git diff --check` | 0 | `19-git-diff-check.log` |

Coverage remained: canonicaljson 87.1%, scalar 90.1%, traceability 85.0%.
The four fuzz baselines were 38, 30, 31, and 75 seeds and each executed exactly
100 cases. `task-board validate` retained 262 inherited `MISSING_ACTIVITY`
diagnostics; this task ID was absent from that successfully-read diagnostic
set. This is a read-only identity calculation/verification scope, so durable
mutation crash recovery is not applicable and no runtime capability is claimed.
