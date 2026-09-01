# TASK-260830-1pbx0c recovery results

## Outcome

- Recovered Change Request revision 3 after configured validation command 8 failed.
- Root cause: the refreshed `c9e5290b` normative-scope authority was present in the working tree but not in the candidate index, so `go generate ./internal/catalog && git diff --exit-code -- internal/catalog/catalog_gen.go` compared the correct generated catalog against the stale Story base.
- Regenerated `internal/catalog/catalog_gen.go`. Its metadata digest (`40023bc194ab63d9f29f90746c2c71fe1ef561bd1a38f766dd7753c150f21bfd`) and 24-root normative scope are byte-for-byte identical to landed authority commit `c9e5290b`.
- Staged only the explicit 22-path candidate comprising refreshed authority plus the scalar implementation, tests, traceability ownership, and README evidence. No board or `.temp` paths were staged.
- The scalar package implements UUIDv4/v7, UTC RFC3339 timestamps, SHA-256 digests, platform/provider IDs, relative and platform-bound absolute paths, safe/bounded integers, decimal uint64 strings, and negotiated closed enums.
- `ParseTimestamp` accepts the published `1990-12-31T23:59:60.000Z` leap second through direct, JSON, and text entry points, while ordinary-minute and unpublished-date `:60` values are refused.
- Assigned Sections 1.6 and 10.1-10.4 resolve to `internal/scalar` production declarations. The production `tracecheck` entry reports `assigned_scopes=5`; a focused mutant test renames each owner independently and requires refusal.
- This package is read-only. No durable-state crash/recovery behavior applies, and no doctor result, provider/platform availability, or runtime capability is advertised.

## Validation

The initial recovery reproduction of configured command 8 exited 1 as expected and is retained in `gate-08-go-generate-expected-red.log`. After the candidate index was restored, the same command exited 0.

| Gate | Command | Exit |
| ---: | --- | ---: |
| focused | `go test ./internal/scalar -count=1 -v` | 0 |
| focused | `go run ./internal/traceability/cmd/tracecheck -section 1.6 -section 10.1 -section 10.2 -section 10.3 -section 10.4` | 0 |
| negative | `go test ./internal/traceability/cmd/tracecheck -run TestMainRejectsRenamedScalarSectionOwnerDeclarations -count=1 -v` | 0 |
| 1 | `test -z "$(gofmt -l .)"` | 0 |
| 2 | `go build ./...` | 0 |
| 3 | `go vet ./...` | 0 |
| 4 | `go test ./... -count=1 -v` | 0 |
| 5 | `go test ./... -race -count=1` | 0 |
| 6 | `go test ./... -cover -count=1` | 0 |
| 7 | `go run ./internal/traceability/cmd/tracecheck` | 0 |
| 8 | `go generate ./internal/catalog && git diff --exit-code -- internal/catalog/catalog_gen.go` | 0 after expected-red recovery reproduction |
| 9 | `GOOS=linux GOARCH=amd64 go build ./...` | 0 |
| 10 | `GOOS=windows GOARCH=amd64 go build ./...` | 0 |
| 11 | JSON parse validation for every tracked `*.json` under `zsh -o pipefail` | 0 |
| 12 | `task-board validate` | 0 |
| 13 | `git diff --check` | 0 |

Repository-wide coverage exited 0; `internal/scalar` measured 89.0%. Both staged and unstaged `git diff --check` are clean. `task-board validate` exited 0 while still printing 264 inherited `MISSING_ACTIVITY` diagnostics; this is recorded as an existing board anomaly, not represented as a clean diagnostic report.

