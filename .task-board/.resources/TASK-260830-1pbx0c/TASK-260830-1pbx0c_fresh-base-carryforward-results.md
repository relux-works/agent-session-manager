# TASK-260830-1pbx0c implementation evidence

## Outcome

Replayed only the seven reviewed `internal/scalar` files from the carry-forward
archive onto fresh trunk `c9e5290b1506275f5417b26070fad0391a09c50a` and added
the scalar-specific ownership, tracecheck tests, and README documentation from
CR revision 4. No pin, catalog, inventory, or general tracecheck implementation
work was reimplemented.

The 22 paths represented by CR revision 4 are byte-identical to the reconstructed
revision-4 tree after combining current trunk with this task delta. The actual
task delta is 12 paths: seven scalar files, four traceability files, and README.

## Pinned authority and scope

- Source: `relux-works/agent-session-manager-spec@v0.5.0`, commit
  `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, `SPEC.md`.
- The authenticated read-only GitHub API fetch exited 0.
- Fetched document SHA-256:
  `562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`,
  exactly matching the committed source lock.
- Sections 1.6, 10.1-10.4, and 17.3 were inspected. Section 1.6 supplies the
  common scalar grammar and path/integer bounds; Sections 10.1-10.4 exercise
  UUID, digest, bounded integer, and closed manifest-enum shapes. Section 17.3
  does not create durable-state behavior in this read-only scalar package.

## Production behavior

- `ParseUUIDv4` and `ParseUUIDv7` enforce canonical lowercase UUID versions and
  RFC 4122 variant bits.
- `ParseTimestamp`, JSON decode, and text decode accept real UTC RFC3339 values
  with millisecond-or-finer precision, including the published leap second
  `1990-12-31T23:59:60.000Z`; they reject an ordinary-minute or unpublished-date
  `:60`.
- `ParseDigest` and `SHA256Digest` enforce canonical `sha256:` identity.
- `ParsePlatform`, `ParseProviderID`, `ParseRelativePath`, and
  `ParseAbsolutePath` enforce closed vocabularies, size/grammar bounds,
  traversal/encoded-separator refusals, and containing-platform context.
- `NewSafeInteger`, `NewUint53`, `NewBoundedInteger`,
  `DecodeBoundedIntegerJSON`, and `ParseDecimalUint64` enforce exact numeric
  bounds and refuse uninitialized bounded values at publication.
- `ParseClosedEnum` and `DecodeClosedEnumJSON` require the exact negotiated
  nonempty vocabulary and reject duplicates, unknowns, malformed UTF-8, and
  lone surrogates.
- Zero/forged validated values cannot be marshaled through the production JSON
  entry points.

The package mutates no durable state, so crash/retry recovery is not applicable.
README explicitly avoids `doctor`, availability, runtime support, or capability
claims.

## Negative evidence

- `TestTimestampAcceptsPublishedLeapSecondsAndRefusesFabricatedOnes` drives
  `ParseTimestamp`, `Timestamp.UnmarshalJSON`, and `Timestamp.UnmarshalText`.
- `TestJSONDecodersCannotBypassScalarValidation` attacks the real JSON decode
  boundaries with wrong versions, unsafe integers, traversal, unknown enums,
  malformed UTF-8, and lone surrogates.
- `TestZeroValuesCannotBePublishedAsValidatedScalars` attacks forged and absent
  construction context through `json.Marshal`.
- `TestMainRejectsRenamedScalarSectionOwnerDeclarations` drives
  `main -> run -> traceability.VerifyAssignedSections` and independently
  renames the owner declaration for each of Sections 1.6 and 10.1-10.4. All
  five mutants are refused with no success output.

## Validation commands and real exit codes

| Command | Exit | Evidence |
| --- | ---: | --- |
| `gofmt -d internal/scalar/*.go internal/traceability/traceability_test.go internal/traceability/cmd/tracecheck/main_test.go` | 0 | Empty diff |
| `go test ./internal/scalar -v -count=1` | 0 | `scalar-tests.log` |
| `go test ./internal/scalar -cover -count=1` | 0 | `scalar-coverage.log`; 89.0% |
| `go test ./internal/traceability/cmd/tracecheck -run TestMainRejectsRenamedScalarSectionOwnerDeclarations -v -count=1` | 0 | `tracecheck-owner-mutants.log` |
| assigned `tracecheck` for Sections 1.6 and 10.1-10.4 | 0 | `tracecheck-assigned.log`; `assigned_scopes=5` |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `tracecheck-global.log` |
| `go test ./... -v -count=1` | 0 | `go-test-all.log` |
| `go test ./... -cover -count=1` | 0 | `go-coverage-all.log` |
| `go test ./... -race -count=1` | 0 | `go-test-race.log` |
| `go vet ./...` | 0 | `go-vet.log` |
| `go build ./...` | 0 | `go-build.log` |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | `go-build-linux-amd64.log` |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | `go-build-windows-amd64.log` |
| `go generate ./internal/catalog` | 0 | `go-generate.log`; no generated delta |
| `git diff --check` | 0 | `git-diff-check.log` |
| byte comparison against all CR revision-4 paths | 0 | No mismatches |
| `curator install` | 0 | `curator-install.log` |
| `curator status --check` after install | 0 | `curator-status-rerun.log` |
| `task-board validate` | 0 | `task-board-validate.log`; prints 264 inherited `MISSING_ACTIVITY` diagnostics |

## Operational anomalies

- The role-provided `.claude/skills/go-testing-tools/SKILL.md` path was absent
  in the fresh worktree. Initial `curator status --check` exited 1 with
  `go-testing-tools not-installed`; `curator install` exited 0 and the exact
  status check rerun exited 0. No tracked Curator files changed.
- Three ad-hoc revision-4 byte-comparison wrappers exited 1 before producing a
  product signal: the first used zsh read-only `status`; the next two used zsh
  special `path`, which temporarily broke `PATH` and made `cmp`/`diff`
  unavailable. The corrected wrapper used `task_scope_diff`, `file_path`, and
  absolute `/usr/bin/diff`; it exited 0 with no mismatch.
- `task-board validate` preserves its existing compatibility behavior: exit 0
  while printing 264 inherited missing-activity diagnostics. None is introduced
  by this task; the exact output is retained in the log bundle.

