# TASK-260830-17suox CR revision 5 validation

## Scope

This rework closes CR revision 4's evidence findings. It adds isolated negative
coverage for every required closed-map member and makes the Section 6 extension
canonical-byte fixture independent of the production constant. Production code
is unchanged.

## Green gates

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/config -count=1` | 0 | `config-package-final-01.log` |
| `go test ./internal/config -cover -count=1` | 0 | `config-cover-final-01.log` (93.2%) |
| `go test ./... -v -count=1` | 0 | `go-test-all-final-01.log` |
| `go test ./... -cover -count=1` | 0 | `go-test-cover-final-01.log` |
| `go vet ./...` | 0 | `go-vet-final-01.log` |
| `go build ./...` | 0 | `go-build-final-01.log` |
| `go mod verify` | 0 | `go-mod-verify-final-01.log` |
| scoped `tracecheck` for 3.2, 6.1-6.5, 17.1-17.2 | 0 | `tracecheck-final-01.log` (`assigned_scopes=8`) |
| `gofmt -l internal/config` plus empty-output assertion | 0 | `gofmt-list-final-01.log` |
| `git diff --check` | 0 | `git-diff-check-final-01.log` |

## Expected-red gate attacks

Each mutant was applied alone, executed with `-count=1`, and restored before the
next. Exit 1 is the expected and truthful result.

| Mutant | Exit | Killing observation |
| --- | ---: | --- |
| N-1: initialize absent maps before source-presence check | 1 | all four isolated omission cases observe unexpected acceptance |
| P-1: drop installation `extensions` presence disjunct | 1 | exact clause changes to downstream `.extensions` |
| P-2: drop profile `extensions` presence disjunct | 1 | exact clause changes to downstream `.extensions` |
| P-3: drop disclosure `extensions` presence disjunct | 1 | exact clause changes to downstream `.extensions` |
| P-4: drop backend-config `settings` presence disjunct | 1 | exact clause changes to downstream `.settings` |
| E-5: widen `65_536` to `655_360` | 1 | 65,537-byte object is unexpectedly accepted |

`schema.go` and `validation.go` matched their pre-mutant task-scoped backups
byte-for-byte after restoration. The final unmutated package and repository
gates then passed.

## Applicability

The scoped loader and current-version encoder are read-only/in-memory surfaces.
No durable state is mutated, so crash-recovery evidence is not applicable here;
durable migration remains owned by `TASK-260830-1qf777`.
