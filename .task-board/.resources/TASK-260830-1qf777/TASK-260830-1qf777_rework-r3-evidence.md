# TASK-260830-1qf777 — rework round 3 evidence

Scope of this round: close reviewer findings F1, F2, F3 carried in the task
notes, and the orchestrator's spec-backed F1 resolution. Everything else in the
task was landed in earlier rounds and is re-verified here on the restored tree.

## F1 — symlink stance (orchestrator-resolved against the pinned SPEC)

The orchestrator resolved F1 against SPEC.md at
`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`: the normative model is
resolve-then-check, never refuse-because-symlink, so refusing a symlinked
directory root is an invented constraint.

Decision taken and now declared, not implied:

| Seam | Stance | Production site | Basis |
| --- | --- | --- | --- |
| Read (`Load`/`LoadOS`) | follows symlinks, applies the Section 3.2 value kind to the resolved target | `internal/config/loader.go:199` (`Stat: os.Stat`) | SPEC states no no-follow requirement for these five classes and resolves symlinks before comparison where it speaks about them; the SPEC.md:2648 idiom (resolve, then require a regular file) is followed rather than the stricter reading of SPEC.md:738 |
| Mutating (`Migrate`/`MigrateOS`) | does not follow symlinks | `internal/config/migration.go` `osMigrationFileSystem.Stat` (`os.Lstat`) | SPEC is silent about this mutation; an atomic rename replaces the operator's link instead of the document it points at, so the mutating path fails closed with `ErrConfigNotRegular` |

Both stances are declared in `README.md` (new paragraph in the configuration
section) and in code comments at both seams, and both are pinned in **both**
directions at the real OS entries:

- read seam: `TestLoadOSResolvesSymlinksAndStillEnforcesKindsAtProductionEntry`
  (six cases: symlinked config onto a regular file loads; symlinked config onto
  a directory refused; symlinked data root onto a directory loads; symlinked
  data root onto a regular file refused; not-yet-created config under a
  symlinked parent admitted; dangling config symlink admitted as
  not-yet-created)
- mutating seam:
  `TestMigrateOSAppliesTheNoFollowMutatingSeamInBothDirections`
  (regular selected source migrates; symlinked selected source refused). The
  refusal subtest first asserts `LoadOS` *accepts* the same selected path, so
  the case isolates the mutating seam rather than a rejected load.

Production change made while closing F1: `replaceDurably` now re-inspects the
selected file's kind and permission bits **before** any durable write, instead
of after the backup was already published. A refused migration therefore
publishes neither a backup nor a staging file. The prior placement would have
left a `config.toml.bak.<version>` file next to a symlink it then refused to
migrate.

## F2 — `MigrateOS` production entry had 0.0% coverage

`internal/config/migration_os_test.go` drives the OS entry end to end against a
real temporary directory tree with explicit overrides for all five Section 3.2
path classes (so the operator's real home / XDG / Application Support locations
are never read or written):

- `TestMigrateOSPerformsOneDurableMigrationAtTheRealProcessEntry` — v1 with a
  legacy `[terminal] backend = "tmux"` table migrates to 3.0.0; exact backup
  bytes at `<config>.bak.1.0.0` with mode 0600; migrated file keeps its 0600
  source mode; `LoadOS` reads back `SourceVersion=3.0.0`,
  `Terminal.BackendID=ax.tmux`, `GeneratedSummaryUpgradeChoice=local_only`; a
  second `MigrateOS` run is a no-op that rewrites neither the document nor the
  backup; no staging leak.
- `TestMigrateOSAppliesTheNoFollowMutatingSeamInBothDirections` — see F1.
- `TestMigrateOSRefusesAnUnknownPlatformBeforeTouchingTheFilesystem` — asserts
  the refusing `Operation` is `"capture OS inputs"`, not merely
  `ErrInvalidContext`. This matters: `ResolvePaths` refuses the same platform
  later under `"validate inputs"` with the same sentinel, so the first version
  of this test admitted a mutant that discarded the `OSInputs` error (see M27
  below). The sentinel-only assertion was subsumed; the operation assertion is
  not.

Coverage after: `MigrateOS` 100.0%.

## F3 — `(*MigrationError).Error()` had 0.0% coverage

`TestErrorFormattingNeverEchoesWrappedDetails` in `loader_test.go` was extended
(as instructed, rather than adding a bespoke assertion) to hold
`*MigrationError` to the same contract as `*Error`: a wrapped
`*fs.PathError` carrying an absolute path and a joined error carrying document
text must not survive formatting, the rendered string must equal
`configuration migration replace source failed`, and `errors.Is` identity must
survive. The MigrateOS symlink-refusal subtest additionally asserts the **real
production** `MigrationError` does not echo the temporary root or the selected
link path.

Coverage after: `(*MigrationError).Error` 100.0%.

## Mutation evidence — 29/29 dead, zero survivors

`python3 .temp/TASK-260830-1qf777/mutants.py` (transcript:
`TASK-260830-1qf777_mutant-sweep-r3.log`). The 22 mutants from earlier rounds
were re-run unchanged and all still die. Seven mutants were added for this
round's delta:

| ID | Class | Change | Killed by |
| --- | --- | --- | --- |
| M23 | widen | migration seam follows symlinks (`os.Lstat` → `os.Stat`) | `TestMigrateOSAppliesTheNoFollowMutatingSeamInBothDirections/symlinked...` |
| M24 | narrow | read seam refuses symlinks (`os.Stat` → `os.Lstat`) — the exact F1 defect | `TestLoadOSResolvesSymlinksAndStillEnforcesKindsAtProductionEntry` (3 positive cases fail) |
| M25 | delete | a failed source inspection treated as satisfied | `TestMigrateRefusesSourceThatStoppedBeingARegularFileAfterLoad/source_stat_failed` |
| M26 | reorder | keep the refusal but run it after a durable staging write | both refusal tests fail on the leaked `.ax-config-backup-*` staging file |
| M27 | delete | `MigrateOS` ignores the `OSInputs` refusal | `TestMigrateOSRefusesAnUnknownPlatformBeforeTouchingTheFilesystem` |
| M28 | widen | `MigrationError.Error()` renders the wrapped OS chain | `TestErrorFormattingNeverEchoesWrappedDetails` |
| M29 | delete | `MigrateOS` reports success without migrating | `TestMigrateOSPerformsOneDurableMigrationAtTheRealProcessEntry` |

Honest note on the kill mechanism for **M25**: deleting that guard leaves `info`
nil, so the test dies with a nil-pointer panic rather than an assertion message.
That is still a true kill — it demonstrates the guard is load-bearing and that
no downstream check covers a failed inspection — but the transcript shows a
panic, not a clean assertion, and is reported as such rather than dressed up.

**M27 survived its first run** and was a real gap, not a scripting artifact: the
original assertion checked only `errors.Is(err, ErrInvalidContext)`, which
`ResolvePaths` also satisfies, so `MigrateOS` could discard its own guard and
still look correct. The test was tightened to assert the refusing operation, and
the mutant then died. This is recorded rather than quietly fixed because it is
exactly the "gate present but subsumed downstream" shape.

## Traceability

`MigrateOS` is now a registered production owner rather than an untraced entry:
new acceptance case `config-durable-migration-os-entry`
(`internal/config/migration.go:MigrateOS` + the three new tests), referenced
from the `section:6.4` and `section:6.5` bindings. The ownership projection
digest gate failed closed on the change (`1abb92b1... differs from reviewed
22d0ced7...`) and the reviewed constant plus both count assertions (34 → 35)
were updated deliberately.

## Gate exit codes (restored tree, this round)

| Gate | Command | Exit |
| --- | --- | ---: |
| Full tests | `go test ./... -count=1` | 0 |
| Full coverage | `go test ./... -cover -count=1` | 0 |
| Config race | `go test ./internal/config -race -count=1` | 0 |
| Vet | `go vet ./...` | 0 |
| Build | `go build ./...` | 0 |
| Format | `gofmt -l .` (empty output) | 0 |
| Traceability | `go run ./internal/traceability/cmd/tracecheck` | 0 |
| Assigned traceability | `tracecheck -section 3.2 -section 6.1..6.5 -section 17.1 -section 17.2 -section 17.4` | 0 |
| Catalog freshness | `go generate ./internal/catalog` + `git diff --exit-code -- internal/catalog/catalog_gen.go` | 0 |
| Module verify | `go mod verify` | 0 |
| Whitespace | `git diff --check` | 0 |
| Mutation sweep | `python3 mutants.py` (29 mutants) | 0 |
| Clean environment | `env -i PATH=... HOME="" GOCACHE=<task> GOPROXY=off GOTOOLCHAIN=local go test ./... -count=1` | 0 |

`internal/config` coverage: 92.3% of statements. Three consecutive
`go test ./internal/config -count=1` runs were identical (no ordering or
timing dependence). The new `MigrateOS` cases pass with no ambient home because
they supply explicit overrides for all five path classes rather than relying on
a platform default.

## Not claimed

No `ax migrate config` CLI command, `doctor` result, or runtime capability is
advertised by this slice. `Migrate`/`MigrateOS`/`AssessCompatibility` are Go
package entry points only.
