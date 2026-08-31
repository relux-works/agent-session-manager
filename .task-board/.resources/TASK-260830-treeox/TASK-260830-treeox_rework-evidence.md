# TASK-260830-treeox developer rework evidence

## Outcome

Reviewer finding F1 is addressed with a test-only change in
`internal/traceability/traceability_test.go`. The production implementation and
ownership registry are unchanged.

The new tests drive the production `VerifyRepository` entry point and protect
both owner-resolution calls that survived the first review:

- `TestVerifyRepositoryRejectsAbsentAcceptanceTestDeclaration` renames the
  registered `ci-entrypoint` test declaration in the repository fixture while
  leaving the registry unchanged. It requires the acceptance-case test-owner
  absent-declaration diagnostic from
  `verifyAcceptanceCases -> checker.verify(test, true)`.
- `TestVerifyRepositoryRejectsAbsentOwnershipGroupProductionDeclaration`
  renames the unique fixture production declaration while leaving the registry
  unchanged. It requires the ownership-group production-owner
  absent-declaration diagnostic from
  `verifyOwnershipGroups -> checker.verify(group.Production, false)`.

`replaceRepositorySourceOnce` fails unless the intended source declaration is
present exactly once, so the fixture mutation cannot silently target a proxy or
an unrelated occurrence.

The exact rework delta from reviewed candidate tree
`6d9a90eb92a2414ac4e72450a34f4ba4fa68ff6c` is stored in
`TASK-260830-treeox_rework.patch`. The patch-generation command exited 1, as
expected for `git diff --no-index` when a real delta is present; it is not
reported as a passing gate.

## Meaningful expected-red evidence

Both reviewer mutants were applied to task-local working bytes, run with the
reviewer's exact two-package scope and `-count=1`, then restored from a saved
copy rather than from Git.

| Mutant | Command | Real exit | Required failure |
| --- | --- | ---: | --- |
| Acceptance-test verification bypassed with `if test { return nil }` | `go test ./internal/traceability ./internal/traceability/cmd/tracecheck -count=1 -v` | 1 | `TestVerifyRepositoryRejectsAbsentAcceptanceTestDeclaration` failed because `VerifyRepository` incorrectly returned nil |
| Ownership-group production verification call removed | `go test ./internal/traceability ./internal/traceability/cmd/tracecheck -count=1 -v` | 1 | `TestVerifyRepositoryRejectsAbsentOwnershipGroupProductionDeclaration` failed because `VerifyRepository` incorrectly returned nil |

The expected-red logs truthfully contain failing suites. The sibling CLI package
remained green under each mutant, isolating the new failure to the owner class
under attack.

After each mutant, `internal/traceability/traceability.go` was restored and
compared byte-for-byte with the saved pre-mutant copy. Both hashes are
`e8bb9312ab55909500f7e4b3f3dc4bcfe36fd804`.

## Green validation

| Command | Real exit | Evidence/result |
| --- | ---: | --- |
| New named tests with `-count=1 -v` | 0 | Both source-declaration refusal tests passed |
| Six relevant specpin/catalog/traceability packages with `-count=1 -v` | 0 | Positive, negative, compatibility, durable recovery, and CLI tests passed |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `60/17/15/30/55` exact inventory |
| `go generate ./internal/catalog` | 0 | Empty output; generated catalog hash still equals Story HEAD |
| `go test ./... -count=1 -v` | 0 | Repository-wide verbose suite passed |
| `go test ./... -count=1 -cover` | 0 | Traceability 82.1%; tracecheck 80.0% |
| `go test ./... -count=1 -race` | 0 | All six packages passed |
| `go vet ./...` | 0 | Empty output |
| `go build ./...` | 0 | Empty output |
| `gofmt -l internal/catalog internal/cataloggen internal/traceability` | 0 | Zero paths |
| `git diff --check` and `git diff --cached --check` | 0 / 0 | Empty output; orchestrator-managed index state preserved |
| Post-mutant two-package rerun with `-count=1 -v` | 0 | Both new tests passed on restored production bytes |
| Post-mutant `go test ./... -count=1` | 0 | Repository-wide suite passed on final bytes |
| Post-mutant `go build ./...` | 0 | Final project build passed |
| Post-mutant tracecheck | 0 | Exact `60/17/15/30/55` inventory reproduced |

## Source and capability boundaries

The local normative lock still identifies
`relux-works/agent-session-manager-spec@v0.5.0`, commit
`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, document SHA-256
`562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`,
and scope `1`, `17`, `20`, `appendix-a`, `appendix-d`.

An independent immutable GitHub/raw fetch returned a cache-miss error. That is
recorded as an unknown network read, not as absence or validation evidence. The
CI gate remains based on repository-local, digest-verified pinned bytes.

This rework adds no runtime availability, support, doctor, or capability claim.
