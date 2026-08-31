# TASK-260830-3uessl developer rework evidence

## Scope and reviewed source

This rework addresses reviewer findings F1 and F2 against the catalog candidate.
The source remains `relux-works/agent-session-manager-spec@v0.5.0`, commit
`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`; the verified `SPEC.md` SHA-256 is
`562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`.

The production entry point under test is `cataloggen.Generate`. The CLI at
`internal/catalog/cmd/cataloggen` calls that entry point before atomically
publishing generated output.

## Rework

- Bound the complete decoded metadata projection to reviewed canonical SHA-256
  `e1fa0e1c7c775e87e128709b7cffa597cb070ab3bb7998ddcd4b93c61e4b45ed`.
  Semantic substitutions are refused even when all changed fields remain
  non-empty. JSON-only formatting changes remain accepted.
- Replaced the nonexistent Terminal Backend section/fixture references with
  Section `4.B-4.C` and the exact Appendix D `Terminal Backend protocol` row.
- Split all seven durable Terminal Backend operations by their distinct Section
  4.C canonical key and recovery evidence. Create/restore use
  `session_id + "/" + bootstrap_operation_id`; attach and the four control
  operations retain their own distinct formulas.
- Audited every other durable operation family. Removed unsupported Directory
  Node `status-before-retry` evidence, corrected post-prepare Mesh
  materialization scopes that do not carry `operation_id`, split Directory
  Query annotation/enrichment/execution recovery authorities, and replaced
  descriptive or nonexistent fixture labels with named Appendix D rows/fixture
  IDs.
- Corrected traceability drift found during the audit: Provider operations now
  point to Sections `7.2, 7.5, 7.A`, Mesh RPC includes Section `11.2`, Terminal
  capabilities point to Section `4.D`, and Observation Events point to Section
  `18.1`.
- Added exact positive tests for every durable-operation evidence row and every
  operation/capability/event/error traceability family. Added non-empty
  narrowing and forged-traceability negatives through the production generator.
- Capability records remain vocabulary-only and expose no availability,
  enabled, supported, or status claim. No `ax`, `doctor`, or runtime capability
  surface was added.

## Meaningful red evidence

The new tests were run before the production fix:

`go test ./internal/catalog ./internal/cataloggen -run 'Test(TerminalBackendEvidenceMatchesPinnedContract|GenerateRejectsNonEmptyNarrowedDurableMutationEvidence|GenerateRejectsForgedNonEmptyTraceability)$' -count=1 -v`

Real exit: `1` (expected failure). The log proves that both non-empty narrowing
mutants and both forged non-empty traceability mutants were accepted by the old
generator, while every Terminal Backend exact-value assertion failed.

Log: `meaningful-red-rework-01.log`.

## Validation

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go generate ./internal/catalog` | 0 | `go-generate-rework-01.log` |
| `go test ./internal/catalog ./internal/cataloggen ./internal/catalog/cmd/cataloggen -count=1 -v` | 0 | `targeted-all-rework-01.log` |
| `go test ./... -count=1 -v` | 0 | `go-test-all-rework-01.log` |
| `go test ./... -count=1 -cover` | 0 | `go-test-cover-rework-01.log` |
| `go test ./... -count=1 -race` | 0 | `go-test-race-rework-01.log` |
| `go vet ./...` | 0 | `go-vet-rework-01.log` |
| `go build ./...` | 0 | `go-build-rework-01.log` |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | `go-build-linux-rework-01.log` |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | `go-build-windows-rework-01.log` |
| `gofmt -l internal/catalog internal/cataloggen` | 0, empty output | `gofmt-rework-01.log` |
| `jq empty internal/catalog/catalog.v0.5.0.json` | 0 | `json-validation-rework-01.log` |
| `git diff --check` | 0 | `git-diff-check-rework-01.log` |
| `curator status --check` | 0 | `curator-status-rework-01.log` |
| `task-board validate` | 0 | `task-board-validate-rework-01.log` |

Coverage for changed implementation packages: `internal/catalog` 97.2% and
`internal/cataloggen` 83.9%. The unchanged generator CLI package is 76.6%; the
existing `internal/specpin` package is 83.0%.

Successful build, vet, generation, formatting, and JSON-validation logs are
empty because those commands emitted no diagnostics.
