# TASK-260830-17suox CR revision 3 validation

Pinned authority: `relux-works/agent-session-manager-spec@28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`; downloaded `SPEC.md` SHA-256 `562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`, matching `internal/specpin`.

## Red-to-green evidence

- Baseline `go test ./internal/config -count=1`: exit 0.
- New behavioral tests before the production fix: `go test ./internal/config -count=1`: exit 1. Failures were exactly the `StrictHostKeyChecking=off` split/combined cases, both known-hosts `none` cases, and disabled-only selected external trust.
- Focused final `go test ./internal/config -cover -count=1`: exit 0, coverage 93.2%.

## Gate attacks

- `.temp/TASK-260830-17suox/clause-mutation-sweep.sh`: exit 0.
- Seventeen isolated narrowing/delete mutants each ran `go test ./internal/config -run <target> -count=1` and each returned the expected failing exit 1; survivors: 0.
- Mutants cover OpenSSH `off`/`none`, disabled trust registration, printable UTF-8/control, SSH UTF-8/NUL, extension namespace/key length/forbidden names/float/array depth, both sorted-unique helpers, and both backend-ID bounds.
- The first sweep correctly exited 1 with one survivor because the root forbidden-name fixture was also invalid reverse-DNS. The grammar-valid replacement isolated the intended clause before the accepted rerun.

## Final validation

| Command | Exit | Result |
| --- | ---: | --- |
| `go test ./internal/config -cover -count=1` | 0 | 93.2% statements |
| assigned-scope `go run ./internal/traceability/cmd/tracecheck` for 3.2, 6.1-6.5, 17.1, 17.2 | 0 | 8 scoped bindings |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | global registry accepted |
| `go test ./... -count=1` | 0 | all 9 packages pass |
| `go test ./... -cover -count=1` | 0 | all 9 packages pass with coverage |
| `go vet ./...` | 0 | clean |
| `go build ./...` | 0 | clean |
| `go mod verify` | 0 | all modules verified |
| generated catalog `-check` command from README | 0 | current |
| `gofmt -l .` plus empty-output assertion | 0 | clean |
| `git diff --check` | 0 | clean |

The implementation performs no durable configuration mutation, so crash/idempotency evidence for atomic file replacement belongs to sibling `TASK-260830-1qf777`; this task proves read-only `Load` idempotency and isolated snapshots in its existing production-entry tests.
