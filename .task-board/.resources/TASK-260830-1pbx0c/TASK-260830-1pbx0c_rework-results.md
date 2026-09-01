# TASK-260830-1pbx0c rework evidence

## Reviewer findings addressed

- `ParseTimestamp` now accepts the pinned RFC 3339 grammar with at least three
  fractional digits, including precision beyond nanoseconds and lowercase
  `t`/`z`. It preserves the original wire spelling while validating a normalized
  spelling as a real calendar instant. Non-UTC offsets, RFC 3339's unknown-local-
  offset `-00:00`, sub-millisecond precision, and impossible dates remain refused.
- `BoundedInteger` now records whether its schema interval came from
  `NewBoundedInteger`/`DecodeBoundedIntegerJSON`. `MarshalJSON` refuses the
  uninitialized zero value, while an explicitly constructed value in `[0,0]`
  still publishes as JSON `0`.
- Compatibility tests drive `ParseTimestamp`, JSON unmarshal, text unmarshal,
  `Timestamp.Time`, and `BoundedInteger.MarshalJSON`, including a 100-digit
  fractional second and the absent-construction-context bypass.

## Normative traceability

- AX v0.5.0 commit: `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`.
- Downloaded `SPEC.md` SHA-256:
  `562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`,
  matching the repository pin.
- AX Section 1.6 requires real UTC RFC 3339 instants with at least millisecond
  precision. RFC 3339 Section 5.6 defines an unbounded `1*DIGIT` fractional
  grammar and permits lowercase `t`/`z`.
- Sections 10.1-10.4 and 17.3 remain represented through the common scalar
  owners without claiming JCS identity or durable migration behavior owned by
  sibling work.

## Validation

Every gate ran as its own foreground process. Output was redirected directly
to the named task-local log without `tee` or status-obscuring pipelines.

| Command | Exit | Evidence |
| --- | ---: | --- |
| Focused rework tests before production fix | 1 | `01-rework-expected-red.log`; expected failure at the three reviewer-reported boundaries |
| Focused rework tests after production fix | 0 | `02-rework-focused.log` |
| `go test ./internal/scalar -count=1 -v` | 0 | `15-scalar-tests-final.log` |
| `go test ./... -v -count=1` | 0 | `16-go-test-all-final.log` |
| `go test ./... -cover -count=1` | 0 | `17-go-coverage-all-final.log`; scalar coverage 88.4% |
| `go vet ./...` | 0 | `18-go-vet-final.log` |
| `go build ./...` | 0 | `19-go-build-final.log` |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `20-tracecheck-final.log`; 60 contracts, 17 sections, 25 cases, 30 fixtures, 55 compatibility contracts |
| `go generate ./internal/catalog` | 0 | `21-go-generate-final.log` |
| Generated catalog diff gate | 0 | `22-generated-diff-final.log` |
| `git diff --check` | 0 | `23-git-diff-check-final.log` |
| `curator status --check` | 0 | `14-curator-status.log`; pinned project skill up-to-date after installation |

## Durable-state applicability and capability claims

The scalar package only validates immutable in-memory values and hashes
caller-supplied bytes. It performs no durable mutation, so crash recovery and
mutation idempotency are not applicable. README explicitly states that this
work adds no doctor result, availability claim, or runtime capability.
