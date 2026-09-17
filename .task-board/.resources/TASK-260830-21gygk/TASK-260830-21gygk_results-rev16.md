# TASK-260830-21gygk producer outcome — rev16

## Result

The CR15 alias finding is repaired in the uncommitted Story candidate. The
shared `go/types` record-consumption census now recognizes the semantic
identity of sealed checkpoint types through Go aliases, including chained
aliases and aliases nested in containers and function signatures. The
candidate also contains no Python cache artifact.

Authority is `agent-session-manager-spec` v0.6.0,
`0cbdf100dbf84df50c64f792b1f940e3a67859a6`; historical scope remains v0.5.0,
`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`.

Candidate checkpoint: `e119aaeede327580a470238fdb1ccf340a785b39`.
Expected base: `8626fb361c1d6b62d5f95c7be4cb33c85d4aadf3`.
Branch: `task-board/story/STORY-260830-3tq4ns`.

## Implementation and evidence

- `internal/sessquery/rev11_regression_test.go` applies `types.Unalias` before
  sealed-type object identity checks and before every recursive value descent.
  The identity walker covers pointers, arrays, slices, maps, channels, structs,
  signatures, generic arguments, interfaces, type parameters, and unions. It
  does not open unrelated named capability types through their underlying
  representation.
- `internal/sessquery/rev15_regression_test.go` drives direct seal, token,
  chained-alias, container-alias, and function-signature-alias plants through
  `rev12CheckRecordConsumptionSource`; all are rejected by the same production
  census entry.
- `internal/sessquery/rev14_regression_test.go` keeps the unknown callback
  negative in the approved profile flow so the callback object-identity gate is
  independently observable. Its named narrowing mutant is killed.
- `internal/sessquery/testdata/mutate.py` adds
  `N-census-alias-value-normalization`, which changes only the nested sealed
  value normalization helper from `types.Unalias(typ)` to `typ`.

The initial alias baseline is intentionally red (exit 1) in
`rev16-alias-baseline.log`: the pre-fix census admitted the alias plants. The
post-fix focused census and callback logs are green. The final mutation copy
restores its source bytes and is stored in `mutants-rev16b/`.

## Mutation classification

`mutants-rev16b/mutants.json` contains 71 classified rows:

| Class | Count | Treatment |
| --- | ---: | --- |
| Applied N/B `KILLED` | 66 | Numerator: 63 narrowing and 3 deterministic-order mutants; every applied behavioral plant failed its named test. |
| Control rows | 2 | `control-before` and `control-after`, both exit 0. |
| Applied harmless control `SURVIVED` | 1 | Expected behavior-preserving control; not counted as a kill. |
| Not applied control | 1 | `C-not-applied`, classified `NOT_APPLIED`; not evidence of a kill. |
| Compile-failure control | 1 | `C-compile-failure`, classified `COMPILE_OR_HARNESS_FAILURE`; not evidence of a kill. |

The new alias row is:

```text
N-census-alias-value-normalization KILLED 1
TestRev15AliasAwareRecordConsumptionCensus/container_of_alias
```

The callback row is also a real kill through the same census instrument:

```text
N-census-unknown-callback KILLED 1
TestRev14CallbackProvenanceRejectsUnknownFunctions/unknown_callback.go
```

## Conformance ratio and bounds

- Shared implementation: **8 of 8 AC rows driven** by named tests through
  `Reader.Resolve`, `Reader.List`, `Reader.Status`,
  `Reader.AuthoritativeStatus`, `Reader.AuthoritativeList`, `Reader.BuildPlan`,
  `Reader.Revalidate`, and the shared checkpoint-admission/census path.
- Public CLI/lifecycle: **0 of 8 delivered here**, an explicit ownership bound.
  This tree has no `ax` executable or Result-5/transport/lifecycle entry point;
  no unsupported capability is advertised.
- CR15 closed findings are retained: canonical Lease Record identity, full
  winning ancestry checkpoint/session/lease/creator/variant/head relations,
  lagging copies, required parked-source refusal, seven capability names,
  predecessor/reducer behavior, live changed-record refusal, and the
  immutable old-plan revalidation boundary.
- Reads and revalidation do not mutate durable state. Crash/idempotency
  evidence is therefore not applicable to this leaf; existing repository
  recovery tests remain green.
- Removed `internal/sessquery/testdata/__pycache__/mutate.cpython-314.pyc` and
  its empty directory. A final `find internal/sessquery -type d -name
  __pycache__ -o -name '*.pyc'` returned no paths.

## Validation

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/sessquery ./internal/sessrepo -count=1 -v` | 0 | `rev16-sessquery-sessrepo.log` |
| `go test ./... -count=1 -v` | 0 | `rev16-go-test-all-v.log` |
| `go test ./... -count=1 -cover` | 0 | `rev16-go-test-all-cover.log` |
| `go test ./... -count=1 -race` | 0 | `rev16-go-test-all-race.log` |
| `go vet ./...` | 0 | `rev16-vet.log` |
| `go build ./...` | 0 | `rev16-build.log` |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `rev16-tracecheck.log` |
| `gofmt -l internal` | 0, no output | `rev16-gofmt.log` |
| `git diff --check` | 0 | `rev16-diff-check.log` |
| `PYTHONDONTWRITEBYTECODE=1 python3 -B internal/sessquery/testdata/mutate.py .temp/TASK-260830-21gygk/mutants-rev16b` | 0 | `mutants-rev16b-run.log` and `mutants-rev16b/mutants.json` |

All CI evidence is local; no hosted CI result is claimed. The complete
source/status and SHA-256 manifest is
`TASK-260830-21gygk_source-provenance-rev16.txt`. The candidate is left
uncommitted for the normal Story handoff and independent review.
