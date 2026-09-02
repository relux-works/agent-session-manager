# TASK-260830-17suox CR revision 4 validation

Specification authority: local `agent-session-manager-spec` tag `v0.5.0`, commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`; reviewed §4.D, §4.E, §6, and §17.

All accepted Go gates below used task-scoped `GOCACHE=.temp/TASK-260830-17suox/gocache` after the shared cache twice lost build artifacts.

## Red-to-green evidence

| Command / probe | Exit | Evidence |
| --- | ---: | --- |
| Baseline `go test ./internal/config -count=1` | 0 | CR rev3 baseline green before new tests. |
| New compatibility/refusal test selection, first red | 1 | Empty multiline maps failed; initial Windows/WSL2 fixtures also exposed missing explicit test environment. |
| Same selection after platform-aware explicit fixtures, before production fix | 1 | Only the four schema-derived empty map round trips and multiline empty-table reader case failed. |
| Same selection after schema-driven presence restoration | 0 | All empty-map, platform, legacy, nested-root, and bound cases green. |
| Tracecheck immediately after expanding ownership evidence | 1 | Expected digest refusal: computed `dde31415b72aa47c121af9698be593ec82e4787bc07393b3b17e8cce6c0b12b1` differed from prior reviewed digest. |
| Tracecheck after explicit digest review/update | 0 | Registry declarations and canonical semantic digest accepted. |

## Mutation attack

- Runner attempt 1: outer exit 1, 93 killed, 0 survived, 16 invalid compile-only mutations. Not accepted.
- Runner attempt 2: outer exit 1, 84 killed, 0 survived, 25 invalid due disappearing shared Go-cache artifacts. Not accepted.
- Corrected final runner: outer exit 0; `total=109 killed=109 survived=0 invalid=0`.
- Every accepted inner mutant invocation exited 1 from behavioral test failure, not build failure.
- Inventory: 99 AST-derived production `configError` disablements plus 10 narrowing mutants covering peer platform/count-only validation, scan-authority minimum, legacy conpty translation, legacy/current Windows defaults, logical-root grammar, extension namespace grammar, int64 safe bound, and empty-map restoration narrowed away from settings.

Full per-mutant results: `TASK-260830-17suox_clause-mutation-sweep-rev4.log`.

## Accepted green gates

| Command | Exit | Result |
| --- | ---: | --- |
| `go test ./internal/config -cover -count=1` | 0 | 93.2% statements |
| `go test ./... -count=1` | 0 | 9/9 packages pass |
| `go test ./... -cover -count=1` | 0 | 9/9 packages pass; config 93.2% |
| `go vet ./...` | 0 | no output |
| `go build ./...` | 0 | no output |
| `go mod verify` | 0 | `all modules verified` |
| catalog generator `-check` | 0 | no output |
| global `tracecheck` | 0 | contracts=60, normative_sections=36, acceptance_cases=32, fixtures=30, compatibility_contracts=55, assigned_scopes=0 |
| scoped `tracecheck` for 3.2, 6.1-6.5, 17.1-17.2 | 0 | assigned_scopes=8 |
| `gofmt -l internal` | 0 | no paths reported |
| `git diff --check` | 0 | no output |

One non-accepted normal gate attempt, `go test ./internal/config -cover -count=1`, exited 1 because the shared Go cache was missing the compiled `fmt` artifact. The task-scoped-cache rerun above is the accepted result; the failure was not presented as green.

No durable configuration state is mutated by this task, so crash-recovery evidence for durable writes is not applicable. Existing read-only/idempotency coverage remains green; migration atomicity is separate scope.
