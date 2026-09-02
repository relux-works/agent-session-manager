# TASK-260830-1qf777 rework evidence (RUN-260901-335354)

Closes CR-TASK-260830-1qf777-1 rev1 (CHANGES REQUESTED) and the cross-lane
derived-gate finding. Behavior was already correct at the real entry points;
this round is evidence work plus one dead-guard removal.

## 1. The derived refusal inventory now proves its own coverage

The previous gate walked `configError` call sites only, so roughly thirteen
`&MigrationError{}` literal sites and nineteen `&Error{}` literal sites reported
green and the README refusal-inventory guarantee was over-broad.

Production change: every refusal is now constructed through exactly one
instrumented package-level constructor per error type.

| Error type | Constructor | Sites rewritten |
| --- | --- | ---: |
| `DocumentError` | `configError` (pre-existing) | 0 |
| `Error` | `loaderError` (new, `loader.go`) | 19 |
| `MigrationError` | `migrationError` (new, `migration.go`) | 18 |

`deriveRefusalInventory` (in `internal/config/refusal_inventory_test.go`) now
derives, from package sources and never from a hand-list:

- every type declaring `Error() string` -> the complete error-type set;
- every package-level `var name = func(...) error` that builds one of those
  types -> the constructor set, requiring exactly one per type;
- every composite literal of an error type built outside its own constructor or
  outside a constructor argument -> a bypass;
- every constructor call site -> the exercised-site inventory.

`TestMain` fails a full-package run when any of: a type has no constructor, a
type has more than one, a bypass literal exists, or a site was never reached.

## 2. Six unpinned gates from the review, now negative-tested

All in `internal/config/migration_refusal_test.go`, driven through the real
`Migrate` entry (or `migrate` with an injected filesystem for durable faults).

1. `TestMigrateRefusesExistingBackupThatIsNotTheExactOwnerOnlySource` -
   stale bytes, group-readable mode, and an unreadable existing backup, each
   asserting `ErrMigrationBackup`, an unchanged source, an unchanged
   pre-existing backup, and no `.ax-config-*` leak; plus a positive control
   proving an exact owner-only backup from an interrupted run is reused.
2. `TestMigrateRefusesEveryTargetOutsideTheUpgradeVocabulary` - `ErrMigrationTarget`
   for empty, unknown, patch-level, and non-semver targets, and separately for
   `Version1`, which is a *known* version and therefore isolates the
   non-upgrade-target clause; plus positive controls for both accepted targets.
3. `TestMigrateRefusesAbsentSourceWithoutWriting` and
   `TestMigrateRefusesSourceThatStoppedBeingARegularFileAfterLoad` -
   `ErrMigrationSourceAbsent`, and the time-of-check to time-of-use recheck for
   both a non-regular source and a failed source stat.
4. `TestMigrateRefusesDisclosureChoiceOutsideTheClosedVocabulary` - six refused
   values including non-empty near-misses (`public`, `local-only`, `LOCAL_ONLY`,
   ` local_only`, `local`) and all three accepted values, so the vocabulary dies
   to both narrowing and widening.
5. `TestMigrateReportsRecoveryFailureWhenRollbackAlsoFails` - `ErrMigrationRecovery`
   with the post-replace sync failing and the rollback staging write also
   failing; asserts the surviving on-disk document is a complete loadable
   Configuration 3.0.0, never torn.
6. Covered by item 1 above (the inventory now scans `MigrationError`).

Additionally `TestMigrateLeavesTheSourceIntactAtEveryDurableFailurePoint`
injects a fault at each of the six durable steps (write backup, publish backup,
remove staging, sync backup directory, write replacement, replace source),
asserting a typed error, an untorn source, no staging leak, and a successful
clean retry.

## 3. Dead guard removed rather than papered over

`migrate` had two separate refusals for an absent source: `!ConfigPresent()`
and `!Configuration()` ok. The second is unreachable while `Load` holds its
contract. The two were merged into one reachable guard instead of being marked
subsumed.

## 4. Subsumed sites are pinned

Four sites cannot be reached without breaking an invariant they exist to
protect. Each carries a `config-refusal-subsumed:` rationale naming its
subsuming check, and `TestRefusalSubsumptionInventoryIsPinned` ratchets the set
so a new exclusion cannot be added silently.

| Site | Rationale | Subsuming check |
| --- | --- | --- |
| `loader.go` capture working directory | `os.Getwd` has no injectable seam and does not fail on a supported host, including a cwd unlinked underneath the process (verified empirically on darwin) | none; fail-closed guard, pinned by the subsumption inventory |
| `loader.go` read selected file (class missing) | `ResolvePaths` populates every registry class | `TestResolvePathsPopulatesEveryOverrideRegistryClass` |
| `loader.go` inspect selected root (class missing) | same | same |
| `migration.go` encode target | pure propagation; each encoder refusal is pinned on its own clause | `writer.go` terminal.backend site and the `EncodeCurrent` validation suite |

Two pre-existing `writer.go` markers and one `schema.go` marker are unchanged
and now pinned by the same test.

New coverage that made two of these subsumptions honest:
`TestLoadRefusesAbsentConfigWhoseParentIsNotResolvable` drives the real `Load`
with an absent Windows UNC share root, which is the only refusal in
`loadAbsentConfig` that is not a stat outcome, and isolates it against a
one-segment-deeper path that refuses with `ErrConfigParentNotFound` instead.

## 5. Mutant sweep: 22 applied, 22 died, none by compile error

Harness: `.temp/TASK-260830-1qf777/mutants.py`. Each mutant is applied to
production source, a scoped `go test` runs, the exit code is recorded, and the
file is restored; the harness re-runs the full package afterwards and asserts
the tree is green again. Full transcript in
`TASK-260830-1qf777_mutant-sweep.log`.

M01-M04 were rewritten mid-round after their first form died on a Go compile
error rather than on behavior; the recorded run uses forms that compile.

| Mutant | Kind | Result |
| --- | --- | --- |
| M01 backup content check | narrow | RED |
| M02 backup permission check | narrow | RED |
| M03 unreadable backup treated as satisfied | narrow | RED |
| M04 whole existing-backup verify deleted | delete | RED |
| M05 admit Version1 as a migration target | narrow | RED |
| M06 target gate deleted | delete | RED |
| M07 absent-source gate deleted | delete | RED |
| M08 disclosure gate narrowed to empty-only | narrow | RED |
| M09 disclosure vocabulary widened with `public` | widen | RED |
| M10 non-regular-source recheck deleted | delete | RED |
| M11 downgrade gate narrowed | narrow | RED |
| M12 recovery failure reported as ordinary sync failure | delete | RED |
| M13 failed atomic replace reported as success | delete | RED |
| M14 read-only downgrade never triggers | narrow | RED |
| M15 unknown reader admitted | delete | RED |
| M16 `sync.max_parallel_chunks` bound widened to 64 | widen | RED |
| M17 duplicate `external_trust` backend_id admitted | delete | RED |
| M18 `DisallowUnknownFields` removed | delete | RED |
| M19 duplicated `migrationError` constructor | self-coverage | RED |
| M20 raw `MigrationError` literal bypassing its constructor | self-coverage | RED |
| M21 synthetic never-exercised migration refusal site | self-coverage | RED |
| M22 synthetic never-exercised loader refusal site | self-coverage | RED |

M19-M22 are the property the cross-lane finding demanded: the gate fails when
someone adds a second copy of a check, bypasses the instrumented form, or adds
a refusal no test reaches - for both refusal forms, not just the named one.

## 6. Gates run in this round

| Command | Exit | Note |
| --- | ---: | --- |
| `gofmt -l .` | 0 | no output |
| `go build ./...` | 0 | |
| `go vet ./...` | 0 | |
| `go test ./...` | 0 | `go-test-all.log` |
| `go test ./... -cover` | 0 | `internal/config` 91.8% (was 83.9%) |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | contracts=60 sections=36 acceptance_cases=34 |
| `go mod verify` | 0 | all modules verified |
| `git diff --check` | 0 | |
| `env -i` with an empty HOME, `go test ./internal/config/` | 0 | determinism |
| mutant sweep (22) | 0 | all mutants died, tree restored green |

Every command was run as a standalone process with its real exit code reported;
none was piped through `tee`.

## 7. Traceability and README

`ownership.v0.5.0.json` extends the `config-durable-migration` acceptance case
with the seven new migration refusal tests. The reviewed ownership projection
digest was re-pinned to
`22d0ced757dbf729bc312ba45f9b4fe81e2b67d4497609270af6e699ae382fe2`.

README now states the narrower, true guarantee: the refusal inventory proves
its own coverage, subsumed sites are pinned and each names its covering check,
and migration refuses a pre-existing backup that differs in bytes, is readable
beyond the owner, or cannot be read at all. No CLI, doctor, or backend
capability is claimed anywhere in this change.

## Scope note

`loader.go` and `writer.go` belong to the earlier story leaves. This round
touches them only to route refusals through the instrumented constructor and to
add subsumption rationales; no path-resolution or writer behavior changed.
