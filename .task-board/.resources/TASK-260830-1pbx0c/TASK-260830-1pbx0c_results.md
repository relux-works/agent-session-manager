# TASK-260830-1pbx0c implementation evidence

## Scope delivered

- Added `internal/scalar` production value types for UUIDv4, UUIDv7, UTC
  RFC3339 timestamps with 3–9 fractional digits, SHA-256 digest identifiers,
  AX platform, provider ID, platform-neutral relative path, platform-bound
  absolute path, safe integer, uint53, schema-bounded safe integer,
  decimal_uint64, and negotiated closed enum values.
- Added validated JSON and text boundaries. Context-dependent absolute paths
  require the containing platform; closed enums require the exact negotiated
  vocabulary. JSON map-key round trips exercise the identifier text boundary.
- Added positive, boundary, wrong-version, wrong-variant, impossible-date,
  wrong-platform, traversal, recursively encoded-separator, unsafe-number,
  non-canonical decimal, unknown-enum, null, malformed UTF-8, and lone-surrogate
  refusal tests against the production parse/decode entry points.
- Updated README and the reviewed ownership registry to 25 executable
  acceptance cases. No doctor result or provider/platform/runtime capability is
  advertised.

## Normative traceability

- Source: `relux-works/agent-session-manager-spec@v0.5.0`, commit
  `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`.
- Downloaded source SHA-256 matched the pinned document digest
  `562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`.
- Implemented Section 1.6 scalar/common-data rules used by Sections 10.1–10.4
  and the immutable migration reference in Section 17.3.
- JCS and omit-self-field identities remain intentionally outside this Task and
  are owned by sibling TASK-260830-236x9n.

## Durable-state and recovery applicability

This change only validates immutable in-memory wire values and computes a
SHA-256 digest from caller-supplied bytes. It performs no file, database,
reference, or other durable-state mutation. Crash recovery and mutation
idempotency fixtures are therefore not applicable to this Task; the package
does not claim them.

## Validation results

Every command below ran as a standalone foreground process. Exit codes are the
real command exit codes; gate output was redirected directly to the named log,
without `tee` or a pipeline.

| Command | Exit | Evidence / result |
| --- | ---: | --- |
| `go test ./internal/scalar -count=1` (negative-first) | 1 | Expected red: production entry points did not exist yet |
| `go test ./internal/traceability -run TestVerifyRepositoryAcceptsExactOwnership -count=1` (registry update) | 1 | Expected red: reported the new reviewed projection digest before pin update |
| `go test ./internal/scalar -count=1` | 0 | `01-scalar-test.log` |
| `go test ./internal/scalar -coverprofile=... -count=1` | 0 | `02-scalar-coverage.log`; 88.2% statements |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `09-tracecheck-final.log`; 60 contracts, 17 sections, 25 cases, 30 fixtures, 55 compatibility contracts |
| `go test ./... -v -count=1` (first full run) | 1 | `04-go-test-all.log`; stale expected acceptance count 15 exposed and corrected |
| `go test ./... -v -count=1` (rerun) | 0 | `05-go-test-all-rerun.log` |
| `go test ./... -cover -count=1` | 0 | `06-go-coverage-all.log`; scalar 88.2%, all packages listed green |
| `go vet ./...` | 0 | `07-go-vet.log` (empty success output) |
| `go build ./...` | 0 | `08-go-build.log` (empty success output) |
| `go generate ./internal/catalog` | 0 | `11-go-generate.log` (empty success output) |
| `git diff --exit-code -- internal/catalog/catalog_gen.go` | 0 | `12-generated-diff.log`; generated catalog unchanged |
| `git diff --check` | 0 | `10-git-diff-check.log` |

The final source-of-truth run is the successful rerun, not the earlier expected
or rework reds.
