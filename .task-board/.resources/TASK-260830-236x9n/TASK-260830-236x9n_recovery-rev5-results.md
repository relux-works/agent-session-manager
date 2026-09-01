# TASK-260830-236x9n recovery revision 5 results

## Candidate outcome

- RFC 8785 canonical JSON remains implemented at `canonicaljson.Canonicalize`, with strict duplicate-name, UTF-8, surrogate, number-formatting, string-escaping, and UTF-16 key-order validation.
- `canonicaljson.CalculateObjectIdentity` and `VerifyObjectIdentity` resolve the omit-self field from the generated v0.5.0 schema/version contract catalog. They do not count globally registered ID names, so referenced IDs remain valid object members.
- The generated catalog covers 40 self-identity definitions and 46 schema-version rows, including all 14 terminal/clone/registry contracts identified in review revision 2.
- The root `.DS_Store` Finder artifact was removed and `.gitignore` excludes recurrence from Change Request candidates.
- The production canonicalization/identity package is read-only and deterministic. It adds no durable-state mutation, doctor result, migration transaction, or runtime capability claim; crash recovery is therefore not applicable to this production entry.

## Recovery result

The former Change Request command 8 compared generated output against the Git index and falsely rejected correct unstaged generated changes. The configured command now uses the read-only production generator check mode:

```text
go run ./internal/catalog/cmd/cataloggen -metadata internal/catalog/catalog.v0.5.0.json -contracts internal/specpin/v0.5.0.lock.json -output internal/catalog/catalog_gen.go -check
```

This command exited 0 without rewriting the output. Its stale-output negative test verifies refusal without replacement.

## Validation evidence

Every command was run directly as a standalone foreground process. No gate was piped through `tee`.

| Command | Exit | Result |
| --- | ---: | --- |
| `go test ./internal/canonicaljson ./internal/catalog ./internal/cataloggen ./internal/catalog/cmd/cataloggen -count=1 -v` | 0 | Focused production, catalog, generator, positive, negative, and recovery tests passed. |
| `test -z "$(gofmt -l .)"` | 0 | Formatting clean. |
| `go build ./...` | 0 | Native build passed. |
| `go vet ./...` | 0 | Vet passed. |
| `go test ./... -count=1 -v` | 0 | Repository tests passed. |
| `go test ./... -race -count=1` | 0 | Race suite passed. |
| `go test ./... -cover -count=1` | 0 | Repository coverage passed; `internal/canonicaljson` measured 82.5%. |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | Default traceability passed: 60 contracts, 36 normative sections, 29 acceptance cases, 30 fixtures. |
| `cataloggen ... -check` | 0 | Generated catalog is current; the former CR command 8 failure is resolved. |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | Linux cross-build passed. |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows cross-build passed. |
| tracked JSON parse gate with `pipefail` | 0 | Every tracked JSON file parsed. |
| `task-board validate` | 0 | Gate exit was 0; it also printed 263 legacy `MISSING_ACTIVITY` issues, recorded below rather than represented as clean diagnostic output. |
| `git diff --check` | 0 | Diff whitespace check passed. |
| scoped `tracecheck` for 1.6 and 10.1-10.4 | 0 | Five assigned scopes passed. |
| `tracecheck -section 10.999` | 1 | Expected refusal: identifier is not a real v0.5.0 section. |

## Mutation evidence

- Wrong-reference narrowing: temporarily mapped Session Annotation to referenced `profile_id`; the production `CalculateObjectIdentity` test failed with exit 1 because it returned `profile_id` instead of `annotation_id`. The generated file was restored byte-for-byte (matching SHA-256), and the same production test then exited 0.
- Missing implementation row: temporarily deleted the Canonical Session row from the runtime identity table after catalog validation; the production `CalculateObjectIdentity` test failed with exit 1 and the exact unsupported-contract refusal. The source file was restored byte-for-byte (matching SHA-256), and the same production test then exited 0.
- `tracecheck -section 10.999` failed with exit 1 as expected and emitted no success report.

## Artifact map

Detailed command output is packaged in `TASK-260830-236x9n_recovery-rev5-validation-logs.tar.gz`. The final post-mutation checks were:

- `go test ./internal/canonicaljson -count=1`: exit 0
- `cataloggen ... -check`: exit 0
- `git diff --check`: exit 0

