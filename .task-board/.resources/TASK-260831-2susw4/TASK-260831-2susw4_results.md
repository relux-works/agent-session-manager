# TASK-260831-2susw4 developer evidence

## Delivered scope

- Extended the immutable `v0.5.0` source pin and reviewed catalog metadata to
  Sections 1-11 and 13-20 plus Appendices A and D (21 source-scope entries).
- Regenerated `internal/catalog/catalog_gen.go`; the typed catalog retains the
  unchanged upstream identity and now carries the expanded source scope.
- Added `traceability.VerifyAssignedSections` and repeatable
  `tracecheck -section` arguments. Granular assignments and same-top-level
  ranges resolve to a pinned `source:<top-level>` ownership binding. Every
  resolved group still has an AST-verified production declaration and a
  registered executable acceptance case.
- Kept the global `VerifyRepository` gate exhaustive. It now reports 60
  contracts, 33 source/catalog normative keys, 16 executable acceptance cases,
  30 fixture identities/anchors, and 55 compatibility contracts.
- Updated README usage and tool documentation for assigned-scope tracecheck.

No runtime capability, support, availability, or conformance claim was added.

## Immutable source identity

| Field | Verified value |
| --- | --- |
| Release/tag | `v0.5.0` |
| Annotated tag object | `d3da6614a6c7bf119a88c9596a86c0853c22cfb9` |
| Commit | `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c` |
| Document | `SPEC.md` |
| Document SHA-256 | `562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a` |

A direct Python identity assertion against the lock and the locally retained
exact `SPEC.md` bytes exited 0. The expanded lock's own reviewed byte digest is
`2bf6f65b244fee705a2f15986beecc120b173bc70c7860b48c9e0b40a50f246c`;
changing that lock digest does not change the upstream source identity.

## Meaningful negative evidence

The pre-implementation focused command exited 1 as expected: the old source
scope lacked Sections 2-11 and 13-19, `VerifyAssignedSections` did not exist,
tracecheck still reported 17/15, and `source:10` could not be removed because it
was not registered. See `expected-red-01.log`.

`TestMainRejectsOneNarrowedAssignedSectionBinding` drives the real production
call chain:

```text
main -> run -> traceability.VerifyAssignedSections
```

In an isolated repository module it removes exactly `source:10`, explicitly
re-pins the narrowed ownership projection (so the pre-existing digest gate
cannot mask the tested bound), and requires `-section 10.1` to fail with no
success output. The same narrowed fixture must still pass `-section 9.2`.
Sections 10.1, 10.2, 10.3, and 10.4 also pass together against the candidate:

```text
traceability ok: contracts=60 normative_sections=33 acceptance_cases=16 fixtures=30 compatibility_contracts=55 assigned_scopes=4
```

Malformed, empty, Section 12 (unpinned), absent-owner, forged-owner, missing
declaration, stale-generation, partial-read, and unknown-field paths remain
negative-tested.

## Validation

Every configured command ran directly as its own foreground process. The table
contains the real exit code observed for each command.

| # | Configured command | Exit | Evidence note |
| ---: | --- | ---: | --- |
| 1 | `test -z "$(gofmt -l .)"` | 0 | No unformatted Go files |
| 2 | `go build ./...` | 0 | Native Darwin/arm64 build |
| 3 | `go vet ./...` | 0 | Vet clean |
| 4 | `go test ./... -count=1 -v` | 0 | Full uncached verbose suite, including narrowed production subprocess |
| 5 | `go test ./... -race -count=1` | 0 | All six packages green under race detector |
| 6 | `go test ./... -cover -count=1` | 0 | traceability 84.3%; tracecheck 87.5% |
| 7 | `go run ./internal/traceability/cmd/tracecheck` | 0 | Exact global inventory `60/33/16/30/55`, assigned scopes 0 |
| 8 | `go generate ./internal/catalog && git diff --exit-code -- internal/catalog/catalog_gen.go` | 0 | Generated catalog current under configured gate |
| 9 | `GOOS=linux GOARCH=amd64 go build ./...` | 0 | Linux cross-build |
| 10 | `GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows cross-build |
| 11 | configured tracked-JSON parse pipeline, with `pipefail` enabled | 0 | Tracked JSON parsed |
| 12 | `task-board validate` | 0 | Validator emitted 264 inherited `MISSING_ACTIVITY` diagnostics; see anomaly below |
| 13 | `git diff --check` | 0 | No whitespace errors |

Additional direct checks also exited 0:

- focused six-package test suite with `-count=1 -v`;
- assigned-scope production tracecheck for Sections 10.1-10.4;
- strict parsing of the three changed JSON files by explicit path; and
- immutable source identity plus exact retained `SPEC.md` byte digest.

## Anomalies and scope preservation

- The Story workspace inherited orchestrator-managed index state where the
  catalog paths are staged-deleted while their candidate files, plus
  traceability and CI paths, are present as untracked files. This task did not
  reset, stage, or absorb that index state. Because configured `git ls-files`
  therefore does not enumerate every changed candidate JSON, an additional
  explicit-path JSON parse was run and exited 0.
- `task-board validate` returned exit 0 while printing 264 inherited
  `MISSING_ACTIVITY` diagnostics for legacy board elements. The configured
  suite therefore passed by its actual exit-code contract, but the diagnostics
  are reported rather than described as absent. The authoritative board and
  this task's activity/resources remain managed through the CLI.
- Initial post-edit `go generate` attempts exited 1 until the reviewed semantic
  metadata digest was updated. The generator diagnostic was improved to report
  exact `got` and `reviewed` digests; the repeated generation and all final
  gates exited 0.
