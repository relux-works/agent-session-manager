# TASK-260830-3uessl catalog implementation evidence

## Normative source

- Repository: `relux-works/agent-session-manager-spec`
- Release: `v0.5.0`
- Commit: `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`
- Document: `SPEC.md`
- Document SHA-256: `562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`
- Scoped registry anchors: Sections 1, 17, and 20; Appendices A and D

The exact commit was available in the local specification checkout and matched
the previously reviewed `internal/specpin/v0.5.0.lock.json`. The public web
fetch was unavailable, so no unpinned web content was substituted.

## Implemented entry points

- `cataloggen.Generate(metadata, lock)` strictly verifies the immutable source
  lock, decodes reviewed metadata with unknown-field refusal, validates every
  contract/version/release binding and durable-operation recovery declaration,
  and emits deterministic formatted Go.
- `go generate ./internal/catalog` invokes the production generator and
  publishes `catalog_gen.go` through an idempotent same-directory temporary
  write and rename.
- `catalog.Current()` exposes the typed v0.5.0 projection.
- `catalog.ForRelease()` exposes only the exact v0.5.0 and v0.4.3 projections
  and refuses every unknown release.

| Release | Contracts | Operations | Capability names | Events | Error codes |
| --- | ---: | ---: | ---: | ---: | ---: |
| v0.5.0 | 60 | 99 | 46 | 112 | 109 |
| v0.4.3 | 55 | 89 | 30 | 112 | 94 |

The operation catalog identifies each authoritative durable mutation and
retains its idempotency scope plus crash/lost-response recovery evidence.
Session Adapter fresh-sink output remains distinct from authoritative durable
mutation.

## Capability-claim boundary

The generated capability catalog is vocabulary only. `catalog.Capability`
deliberately has no `Available`, `Enabled`, `Supported`, or `Status` field.
No `ax` command, doctor result, manifest, probe result, conformance claim, or
runtime capability is introduced by this task.

## Negative and compatibility evidence

Tests drive the production `cataloggen.Generate`, `catalog.Current`,
`catalog.ForRelease`, and generator file-write entry points. They reject:

- absent, partial, trailing, unknown-member, and source-substituted metadata;
- any byte-changed normative lock;
- duplicate operations, unknown contracts, unsupported contract versions, and
  release/contract substitution;
- narrowed durable-operation recovery evidence;
- capability availability claims embedded in metadata;
- unknown release lookup; and
- unreadable output replacement.

Exact tests cover every operation family, capability family, Session and
Observation event name, Structured Error code-to-exit mapping, durable effect
set, compatibility count, and generated-byte freshness. The file-write test
proves identical retries do not replace the destination.

## Validation record

All commands below ran directly as standalone processes. The recorded exit
codes are their actual process results.

| Gate | Exit | Evidence |
| --- | ---: | --- |
| `go generate ./...` | 0 | `go-generate-final-03.log` |
| Targeted catalog/generator/CLI tests, `-count=1 -v` | 0 | `targeted-tests-final-03.log` |
| `go test ./... -count=1 -v` | 0 | `go-test-all-final-03.log` |
| `go test ./... -cover -count=1` | 0 | `go-test-cover-final-03.log` |
| `go test ./... -race -count=1` | 0 | `go-test-race-final-03.log` |
| `go vet ./...` | 0 | `go-vet-final-03.log` |
| Native `go build ./...` on darwin/arm64 | 0 | `go-build-darwin-arm64-final-03.log` |
| Linux builds on amd64 and arm64 | 0 / 0 | `go-build-linux-amd64-final-03.log`; `go-build-linux-arm64-final-03.log` |
| Windows builds on amd64 and arm64 | 0 / 0 | `go-build-windows-amd64-final-03.log`; `go-build-windows-arm64-final-03.log` |
| `gofmt -l .` plus empty-output assertion | 0 / 0 | `gofmt-final-03.log` |
| Strict JSON parse | 0 | `json-validation-final.log` |
| `git diff --check` | 0 | `git-diff-check-final-03.log` |
| `curator status --check` after installation repair | 0 | `curator-status-02.log` |
| `task-board validate` | 0 | `task-board-validate-pre-handoff.log` |

Coverage was 97.2% for the typed catalog API, 84.2% for the generator, 76.6%
for the thin CLI wrapper (its testable `run` entry point is 100%), and 83.0%
for the pre-existing source-pin package.

Earlier red/rework evidence is intentionally preserved in the task directory:

- `red-tests-01.log`: expected red, exit 1, before production packages existed.
- `go-generate-01.log`: exit 1, malformed JSON detected before output.
- `targeted-tests-02.log`: exit 1, exact effect test caught four misclassified
  Directory Node reads; metadata was corrected from the normative operation
  table.
- `curator-status-final.log`: exit 1 because the isolated story worktree had no
  installed skill adapter; the supported `curator install` command exited 0,
  and the repeated status check exited 0 without tracked repository changes.
