# TASK-260830-treeox revision-2 rework evidence

## Outcome

The reviewer-requested rework is test-only and closes both surviving narrowed
mutants from `CR-TASK-260830-treeox-2`:

1. `TestVerifyRepositoryRejectsAbsentAcceptanceProductionDeclaration` renames
   the unique `catalog-generation-idempotency-recovery -> writeIfChanged`
   declaration inside an isolated `fstest.MapFS` snapshot, leaves the reviewed
   ownership registry unchanged, drives production `VerifyRepository`, and
   requires `ErrTraceability` plus the exact acceptance-production diagnostic.
2. `TestRunRejectsRegisteredContractWithoutImplementationOwner` copies the
   repository's `internal/` tree into `t.TempDir`, removes only the registered
   Session Directory Query contract owner, drives the production `run` entry
   point, requires `ErrTraceability` plus the owner-specific diagnostic, and
   requires zero success output.

No production source, ownership registry, catalog, README claim, durable state,
or advertised capability changed. The new command fixture mutates only a
task-local temporary directory. Crash recovery is therefore not applicable;
the existing read-only/idempotency coverage remains green.

## Exact delta identity

- Reviewed candidate tree: `9ce4e16f8831238a416504bccb76f9348ad7d6ac`
- Rework tree: `892b4dc4c8453138a790ce85c97ebc9e0ec44f3e`
- Delta: 2 test files, 59 insertions
  - `internal/traceability/traceability_test.go`
  - `internal/traceability/cmd/tracecheck/main_test.go`
- Exact-tree `git diff --check`: exit 0
- Production blob identity against the reviewed candidate:
  - `traceability.go`: `e8bb9312ab55909500f7e4b3f3dc4bcfe36fd804` (match)
  - `tracecheck/main.go`: `23924b44c40309a5741958f34c24ef950d4844b4` (match)
  - `ownership.v0.5.0.json`: `16781235c8ce0714871a5d8464105f2cef3caa36` (match)

## Meaningful negative evidence

| Gate | Mutation | Real result | Intended catcher |
| --- | --- | ---: | --- |
| Acceptance production owner | Removed only `checker.verify(acceptance.Production, false)` and its diagnostic | exit 1 (expected failure) | `TestVerifyRepositoryRejectsAbsentAcceptanceProductionDeclaration` failed because production admitted the renamed declaration |
| Headless CLI propagation | Suppressed only errors containing `has no implementation owner`, preserving all other error propagation | exit 1 (expected failure) | `TestRunRejectsRegisteredContractWithoutImplementationOwner` failed because `run` returned nil for the owner loss |

Both mutants ran the exact two-package command with cache disabled:

```text
go test ./internal/traceability ./internal/traceability/cmd/tracecheck -count=1 -v
```

Each production file was restored from a pre-mutation copy and compared
byte-for-byte with `cmp -s`; both restore comparisons exited 0.

## Green validation

| Command | Exit | Evidence |
| --- | ---: | --- |
| Focused acceptance-production test | 0 | `rework2-acceptance-production-test.log` |
| Focused tracecheck propagation test | 0 | `rework2-tracecheck-propagation-test.log` |
| Six-package targeted suite, `-count=1 -v` | 0 | `rework2-targeted-green.log` |
| `go test ./... -count=1 -v` | 0 | `rework2-go-test-all.log` |
| `go test ./... -count=1 -cover` | 0 | `rework2-go-test-cover.log` |
| `go test ./... -count=1 -race` | 0 | `rework2-go-test-race.log` |
| `go vet ./...` | 0 | empty diagnostic log |
| Darwin `go build ./...` | 0 | empty diagnostic log |
| Linux amd64 `go build ./...` | 0 | empty diagnostic log |
| Windows amd64 `go build ./...` | 0 | empty diagnostic log |
| `go generate ./internal/catalog` | 0 | generated file remained byte-identical (`cmp -s` exit 0) |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | exact inventory `60/17/15/30/55` |
| `gofmt -l internal` plus empty-output assertion | 0 / 0 | no paths emitted |
| Working, cached, and exact-tree diff checks | 0 / 0 / 0 | no whitespace errors |

Coverage remained 82.1% for `internal/traceability` and 80.0% for
`internal/traceability/cmd/tracecheck`; all repository packages passed.

## Normative-source boundary

The local upstream clone resolves exact commit
`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`. GitHub and raw GitHub fetches
returned cache misses, so public immutable fetch status is recorded as unknown,
not as absence. The production tracecheck and full suite re-verified the local
digest-pinned lock and catalogs. No capability availability/support claim was
added.
