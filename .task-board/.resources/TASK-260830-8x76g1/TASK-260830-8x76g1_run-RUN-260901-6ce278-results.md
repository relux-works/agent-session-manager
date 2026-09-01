# TASK-260830-8x76g1 — developer rework results (RUN-260901-6ce278)

## Outcome

The CR revision 10 test-only rework is implemented for every reachable boundary.
`internal/canonicaljson/boundary_constraints_test.go` adds paired positive and
negative cases for the 23 reviewer-listed constraints. Constraints 1–22 drive both
`CalculateObjectIdentity` and `VerifyObjectIdentity` through
`assertIdentityEntriesAcceptShape` and `assertIdentityEntriesRefuseShape`.

Constraint 23 (`GitIndex.entries[0..65536]`) has a normative reachability conflict:

- `validateGitIndex` accepts 65,536 entries and refuses 65,537 entries.
- Widening its cap to 65,537 makes the new boundary test fail (expected-red exit 1).
- The smallest valid closed parent used by the test encodes to 12,323,001 bytes.
- Both public identity entries must refuse it at the earlier 5,242,880-byte object
  cap in `prepareObjectIdentity`; public accept-at-65,536 is therefore impossible
  without changing the production size contract.

No production validator was changed. The constraint enumeration now states the
GitIndex exception instead of claiming public acceptance that cannot reproduce.

## Source binding and restoration

- Accepted-leaves archive SHA-256:
  `9ae5e624addc7d954c391d96cab7c9f7aac3e6bcee58c76dd0cc95533d0eac9c`.
- Audit carryforward archive SHA-256:
  `df8d8fa712e0ce85bde4776dc1f83cef9d373593e76fd1145ce9d85b593ce4ee`.
- Before this run's edits, a temporary-index `git write-tree` produced
  `16e1a20bc556f27c581299160e8f4a12ae555bbb`, exactly CR revision 10's reviewed
  candidate tree. The archive overlay matched 56 files byte-for-byte; the nine
  differences were the later owned CR revision 10 scope named by the reviewer.
- After all 23 mutants, `closed_shapes.go` compared byte-for-byte equal to the
  pre-mutation copy (`cmp` exit 0).

## Negative evidence

Every reviewer-listed widening mutant was applied individually and tested with
`-count=1`. Every command exited 1 because its named boundary test failed. This is
expected-red evidence, not a passing gate. Logs are `mutant-01-*.log` through
`mutant-23-*.log` in the attached validation archive.

The mutants cover: dot-less reverse-DNS, non-canonical semver, argv count/encoded
size, env name/literal counts and lengths, task element length/control characters,
goal ID characters, media-type lowercase, Blob chunk count, Manifest entry count,
mode, Windows-only symlink separators, child IDs, exclusions, remotes, workspace
members, filter names, project paths, and GitIndex entry count.

## Validation run directly by this developer

| Command | Exit | Evidence |
| --- | ---: | --- |
| baseline `go test ./internal/canonicaljson -count=1` | 0 | `baseline-canonicaljson-01.log` |
| public boundary test (constraints 1–22) | 0 | `boundary-public-01.log` |
| GitIndex boundary/outer-cap test | 0 | `git-index-boundary-02.log` |
| configured format gate | 0 | `validation-format-01.log` |
| `go build ./...` | 0 | `validation-build-01.log` |
| `go vet ./...` | 0 | `validation-vet-01.log` |
| first `go test ./... -count=1 -v` | 1 | `validation-full-01.log`; expected rework failure from an artifact call-site column drift, then corrected |
| rerun `go test ./... -count=1 -v` | 0 | `validation-full-02.log` |
| `go test ./... -race -count=1` | 0 | `validation-race-01.log` |
| `go test ./... -cover -count=1` | 0 | `validation-coverage-01.log`; canonicaljson 82.5%, scalar 90.1% |
| four configured fuzz targets, fixed `-fuzztime=100x -parallel=1` | 0 each | `validation-fuzz-*-01.log` |
| `tracecheck` | 0 | `validation-trace-01.log`; 60/36/29/30/55, assigned scopes 0 |
| `tracecheck -section 17.3` | 0 | `validation-trace-17.3-01.log`; assigned scopes 1 |
| catalog freshness | 0 | `validation-catalog-01.log` |
| Linux and Windows cross-builds | 0 each | `validation-linux-build-01.log`, `validation-windows-build-01.log` |
| tracked JSON parse, board validation, diff check | 0 each | corresponding `validation-*-01.log` |

## Stop-the-line decision required

The literal requirement that all 23 constraints have public accept-at-N and
refuse-at-N+1 evidence conflicts with the existing public encoded-object cap for
GitIndex. Clean options:

1. Recommended: accept the honest exception. Keep the 5 MiB public cap, retain the
   direct GitIndex 65,536/65,537 boundary test plus public outer-cap refusal, and
   amend the checklist wording to distinguish validator reachability from public
   representability.
2. Increase the public identity size cap above 12,323,001 bytes. This makes the
   boundary publicly reachable but is a production/spec contract change, violates
   the review's test-only direction, and changes resource-exhaustion behavior.
3. Lower the declared GitIndex entry maximum to a representable number. This is a
   normative specification change outside the task scope.

Exact decision needed: authorize option 1, or explicitly authorize a production and
specification change under option 2 or 3.
