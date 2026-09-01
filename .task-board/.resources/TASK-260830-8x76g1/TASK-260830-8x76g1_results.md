# TASK-260830-8x76g1 developer results

## Scope delivered

- Added `FuzzScalarProductionEntries` at the production scalar constructors and JSON/text boundaries.
- Added `FuzzCanonicalizeRoundTrip` and `FuzzObjectIdentityRepresentationInvariant` at `Canonicalize`, `CalculateObjectIdentity`, and `VerifyObjectIdentity`.
- Added checked-in native Go fuzz corpus entries for leap seconds, lowercase RFC3339, an impossible date, a malformed UUID, unsafe integers, Windows reserved device names and wildcards, unpaired surrogates, duplicate map keys, non-string map-key syntax, Unicode UTF-16 ordering, and nested reordered objects.
- Added recursive malformed-object negative tests and a language-neutral cross-platform identity golden. The golden proves identical SHA-256 identity for LF and CRLF representations with different key order, nested Unicode, and UTF-16 ordering.
- Added fixed `-fuzztime=100x -parallel=1` validation commands to the Story landing suite.
- Updated README tool commands and claims without advertising doctor, migration publication, or runtime capabilities.
- Extended the production traceability source checker from executable `Test*` references to executable `Test*|Fuzz*` references. `Fuzzhelper` and `BenchmarkBoundary` remain refused, and the ownership projection digest was reviewed and repinned.

## Production entry points exercised

- `internal/scalar`: `ParseTimestamp`, `ParseUUIDv7`, `ParseUUIDv4`, `ParsePlatform`, `ParseProviderID`, `ParseRelativePath`, `ParseAbsolutePath`, `ParseDigest`, integer JSON decoders, `ParseDecimalUint64`, and `ParseClosedEnum`.
- `internal/canonicaljson`: `Canonicalize`, `CalculateObjectIdentity`, and `VerifyObjectIdentity`.
- `internal/traceability`: `sourceChecker.verify`, reached by `VerifyRepository`, `VerifyAssignedSections`, and the `tracecheck` CLI.

## Validation evidence

Every command below was run directly as a standalone process. Exit codes are the real process exit codes.

| Command | Exit | Evidence |
| --- | ---: | --- |
| `test -z "$(gofmt -l .)"` | 0 | direct run |
| `go build ./...` | 0 | `evidence/go-build.log` |
| `go vet ./...` | 0 | `evidence/go-vet.log` |
| `go test ./internal/scalar ./internal/canonicaljson ./internal/traceability -count=1 -json` | 0 | `evidence/focused-tests.jsonl` |
| `go test ./... -count=1` | 0 | `evidence/full-tests.log` |
| `go test ./... -race -count=1` | 0 | `evidence/race-tests.log` |
| `go test ./... -cover -count=1` | 0 | `evidence/coverage.log` |
| scalar fuzz, `-fuzztime=100x -parallel=1` | 0 | `evidence/fuzz-scalar.log` |
| canonicalization fuzz, `-fuzztime=100x -parallel=1` | 0 | `evidence/fuzz-canonicalize.log` |
| identity fuzz, `-fuzztime=100x -parallel=1` | 0 | `evidence/fuzz-identity.log` |
| assigned-scope `tracecheck` for 1.6 and 10.1-10.4 | 0 | `evidence/tracecheck-scoped.log` |
| global `tracecheck` | 0 | direct run; 60 contracts, 36 sections, 29 acceptance cases, 30 fixtures, 55 compatibility contracts |
| catalog generator check mode | 0 | direct run |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | `evidence/build-linux-amd64.log` |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | `evidence/build-windows-amd64.log` |
| Linux and Windows cross-compilation of scalar/canonicaljson test binaries via `go test -exec=/usr/bin/true` | 0 / 0 | direct runs |
| tracked JSON validation with `pipefail` | 0 | direct run; the new golden was also parsed explicitly before staging |
| `git diff --check` | 0 | `evidence/git-diff-check.log` |
| `task-board validate` | 0 | `evidence/task-board-validate.log`; anomaly below |

Coverage for the principal packages: `internal/scalar` 90.5%, `internal/canonicaljson` 84.0%, and `internal/traceability` 85.0%.

## Negative evidence and recovery applicability

- Recursive duplicate keys, non-string member syntax, and unpaired surrogates are supplied to `Canonicalize` and required to return `ErrInvalidJSON` at the actual production entry point.
- Invalid dates, malformed identifiers, unsafe integers, and Windows device/wildcard paths are committed seeds. Accepted scalar values must publish and read back identically; an accepted-then-refused value fails the fuzz target.
- Identity fuzz reorders top-level keys, canonicalizes nested keys, changes whitespace/line endings, verifies the claimed digest, and recalculates it. Any ordering change, panic, or accepted-then-refused identity fails.
- The first scoped tracecheck after registering fuzz evidence exited 1 with `FuzzCanonicalizeRoundTrip is not an executable Go test reference`. The production checker was then widened only to Go-native `Test*|Fuzz*`; its regression test refuses lowercase helper and benchmark names. The same scoped gate then exited 0.
- This implementation is read-only and mutates no durable AX state. Crash/idempotent durable-state recovery evidence is therefore not applicable; no unsupported mutation or recovery capability is claimed.

## Source and anomaly

The normative source was inspected at `relux-works/agent-session-manager-spec` commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, Sections 1.6, 10.1-10.4, 17.3, and Appendix D fixture rules.

`task-board validate` exited 0 but emitted 262 inherited `MISSING_ACTIVITY` diagnostics for legacy elements. `TASK-260830-8x76g1` was not among them. The anomaly is recorded separately in `TASK-260830-8x76g1_logbook.md` and is not represented as a clean diagnostic-free board.
