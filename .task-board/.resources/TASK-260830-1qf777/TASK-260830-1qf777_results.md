# TASK-260830-1qf777 implementation evidence

## Outcome

- Added explicit production `config.Migrate` / `config.MigrateOS` entries for Configuration 1.0.0 -> 2.0.0 or 3.0.0 and 2.0.0 -> 3.0.0.
- Migration validates through `Load` before any durable write, requires the v1 generated-summary disclosure choice, preserves exact old bytes in a deterministic owner-only backup, writes and fsyncs a same-directory temporary file, atomically replaces the source, and fsyncs the directory.
- Existing exact backups are reusable, target-version reruns are no-ops, and a post-replace directory-sync failure restores the exact source bytes before returning an error. The recovery test retries the same migration successfully.
- Added structurally read-only `AssessCompatibility`; newer documents yield `read-only-diagnostic` without decoding a Configuration or exposing a writer path. Malformed/read-failure facts remain refusals rather than absence/fallback.
- CORRECTED IN ROUND 3: an earlier round of this task tightened the real OS loader to `os.Lstat`, which silently refused a symlinked configuration root. The pinned SPEC resolves symlinks before checking and never refuses because-symlink, so that was an invented constraint and has been reverted. The read seam follows symlinks and applies the Section 3.2 value kind to the resolved target; only the durable migration seam is no-follow, and for a stated reason (an atomic rename would replace the operator's link rather than the document). See `TASK-260830-1qf777_rework-r3-evidence.md`.
- Deferred `UserHomeDir` failure until a platform default actually needs home. Explicit path sets now work with empty `HOME`, while the original lookup failure remains unwrap-able and cannot be mistaken for legitimate absence.
- Added reviewed traceability owners for Sections 6.4, 6.5, and 17.4 and updated README/tool output documentation without claiming an `ax migrate config` command, doctor result, backend availability, or runtime capability.

## Production call sites and negative evidence

- `config.Migrate` drives the real `Load` gate. `TestMigrateProductionEntryRefusesInvalidSourceBeforeBackupOrWrite` proves unknown closed fields, a narrowed `sync.max_parallel_chunks=33` bound, and duplicate TerminalBackend IDs are refused before backup or replacement.
- `config.Migrate` is exercised for owner-only backup, complete v2 output, v2->v3 backend mapping, invalid choice, downgrade refusal, idempotent replay, post-replace sync rollback, and recovery retry.
- `config.AssessCompatibility` is exercised for v3 read by a v1 reader and for unknown reader, wrong schema, malformed version, unknown older version, and malformed TOML refusals.
- `config.LoadOS` is exercised against real symlinks in both directions by `TestLoadOSResolvesSymlinksAndStillEnforcesKindsAtProductionEntry` (six cases). `config.MigrateOS`, the OS migration entry, is exercised end to end against a real directory tree by `internal/config/migration_os_test.go` (round 3).

## Validation evidence

All commands were run directly as standalone processes. Exit codes are the real process results.

| Command | Exit | Result |
| --- | ---: | --- |
| `go test ./internal/config -run 'Test(Migrate|AssessCompatibility)' -count=1 -v` | 0 | All new migration/downgrade groups passed. |
| `go test ./internal/config -count=1` | 0 | Full config suite and refusal-site inventory passed. |
| `go test ./internal/config -cover -count=1` | 0 | 90.0% statement coverage. |
| `go test ./... -v` (final rerun) | 0 | Repository-wide verbose suite passed. |
| `go test ./... -cover` | 0 | Repository-wide coverage suite passed; config 90.1%, all packages green. |
| `go vet ./...` (final rerun) | 0 | No findings. |
| `go build ./...` (final rerun) | 0 | Build passed. |
| `go run ./internal/traceability/cmd/tracecheck -section 6.1 -section 6.2 -section 6.3 -section 6.4 -section 6.5 -section 17.1 -section 17.2 -section 17.4` | 0 | 60 contracts, 36 normative sections, 34 acceptance cases, 30 fixtures, 55 compatibility contracts, 8 assigned scopes. |
| `git diff --check` | 0 | No whitespace errors. |
| Synthetic unexercised `configError` injected into isolated copy; `go test ./internal/config -count=1` | 1 (expected red) | Refusal inventory named the synthetic `migration.go:394` site; original worktree was untouched. |
| Dynamic refusal/narrowing mutation sweep in isolated copy | 0 | 113 total mutants; 113 killed by inner expected-red exit 1; 0 survivors; 0 invalid; task-scoped `GOCACHE`. |
| `env -i PATH=... HOME="" GOCACHE=<task> GOMODCACHE=<offline-cache> GOPROXY=off GOTOOLCHAIN=local go test ./... -count=1` (final rerun) | 0 | All 9 packages passed with no ambient home and an isolated Go build cache. |

## Expected development failures resolved

- The first post-patch config run exited 1 because the refusal inventory correctly found six new unexercised/refusal sites; task-specific negative tests and two explicit subsumption annotations resolved it.
- The first focused migration run exited 1 only because a test assumed double-quoted TOML while the encoder emitted valid single-quoted TOML; the assertion now reloads through the production reader and checks typed version/backend values.
- The first top-level trace invocation exited 1 because parent headings `6`/`17` have no scoped owner; the exact subsection invocation is the valid gate.
- The first exact trace invocation exited 1 on the stale reviewed ownership projection digest; the reviewed registry digest was repinned after adding the two acceptance owners.
- The first repository-wide verbose run exited 1 because one traceability test still expected 32 acceptance cases; it was updated to the reviewed count of 34, the affected packages passed, and the full verbose suite then passed.
- The first clean-environment suite exited 1 because `OSInputs` eagerly required `HOME` before considering explicit paths. Home lookup failure is now deferred and preserved as error evidence only when a default needs it; the clean-environment rerun passed all packages.

## Round 3 supersession

The validation table above records round 1. Round 3 closed reviewer findings F1
(symlink stance), F2 (`MigrateOS` at 0.0% coverage) and F3
(`(*MigrationError).Error` at 0.0% coverage), moved the selected-file kind check
ahead of every durable write, and registered `MigrateOS` as a traceability
production owner (acceptance cases 34 -> 35). Current figures: `internal/config`
coverage 92.3%, `MigrateOS` and `(*MigrationError).Error` at 100.0%, 29/29
mutants dead with zero survivors. Current gate exit codes are in
`TASK-260830-1qf777_rework-r3-evidence.md`.

## Normative source

The local sibling specification checkout resolved commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c` exactly. Implementation follows Sections 6.4, 6.5, 17.2, and 17.4: explicit major migration, exact legacy terminal mapping, owner-only backup, same-directory fsync/atomic replacement, failure preservation/retry, unknown closed-field refusal, and read-only downgrade behavior.
