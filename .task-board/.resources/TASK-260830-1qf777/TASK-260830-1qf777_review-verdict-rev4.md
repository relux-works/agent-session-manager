# TASK-260830-1qf777 review verdict — CR revision 4

**Verdict: changes requested (`to-dev`).** One blocking finding. Everything the
acceptance criteria name is proven; the one gap is on a production refusal this
task itself removed.

Reviewed candidate tree `0ea2e47f55b3ce32227d69763c4bcc0118b5453d` against base
`48db30b59e5e1bbc5e0cf73ec2e0e0eec3d215d1`. The worktree was verified to hash to
exactly that candidate tree (`git write-tree` on a temporary index).

## Verified green

| Check | Result |
| --- | --- |
| `go build ./...`, `go vet ./...` | clean |
| `go test ./... -cover` | all 9 packages ok; `internal/config` 93.1% of statements |
| `gofmt -l internal/` | clean |
| `tracecheck` (global) | `contracts=60 normative_sections=36 acceptance_cases=35 fixtures=30 compatibility_contracts=55 assigned_scopes=0` |
| `tracecheck -section 3.2 6.1..6.5 17.1 17.2 17.4` | same, `assigned_scopes=9` |
| Ownership pin `reviewedOwnershipCanonicalSHA256` | not accepted on trust — an independent source scan confirms all three new acceptance cases name a production declaration and 21 test declarations that exist in the named files |

## The three revision-3 blocking findings are fixed, and I re-killed each

| rev3 finding | rev4 status | Mutant that now dies |
| --- | --- | --- |
| F1 read-only downgrade unproven for the one-major step | fixed | `compareSemver(source, reader) > 0` → `source.major > reader.major+1` dies at `TestAssessCompatibilityPinsEveryPinnedVersionPairAtTheProductionEntry` on both `(2.0.0, reader 1.0.0)` and `(3.0.0, reader 2.0.0)` |
| F2 migration never carried a non-default member | fixed | dropping `SafeBoundaryTimeoutSeconds` from `currentWire`, dropping it from `encodeVersion2`, dropping `GracefulStopTimeoutSeconds`, and zeroing `Mesh`/`Profiles` all die at `TestMigrationRetainsEveryConfiguration1MemberOnEveryTarget` |
| F3 temp-file fsync untested | fixed | deleting `file.Sync()` from `writeTempFile` dies at `TestMigrateFsyncsEveryStagedFileBeforeItBecomesVisible` |

The rev3 non-blocking notes 1, 2 and 4 are also addressed: the `v2 re-read` and
`terminal.backend` refusals are now individually pinned in the subsumption
inventory, `TestRefusalSubsumptionInventoryIsPinned` reads the pin's keys and
justifications instead of only its length, and tightening the replacement mode to
`0o600` now reddens as well as widening it.

The preservation design is the strongest part of this revision. It is derived,
not listed: `wireLeafTOMLPaths(reflect.TypeOf(rawV1{}))` drives the member set,
every non-exempt member must differ from the value it would decay to, every
exemption is falsified through the real `Load` entry, and
`TestVersion1SourcesNeverCarryDirectoryCollections` fails the moment `rawV1`
grows a directory collection. A member added later fails closed without anyone
editing a test.

## Independent mutation sweep

44 mutants, one source edit each, `go test ./internal/config/ -count=1` per run.
**41 killed, 3 equivalent, 2 genuine survivors.** Full table:
`TASK-260830-1qf777_reviewer-mutation-sweep-rev4.log`.

Every acceptance-criterion gate dies under a *narrowing* mutant, not just a
delete-only one:

- **backup** — mode `0o600`→`0o644`, backup holding the replacement instead of the
  original, `.bak.<version>`→`.bak`, `Link`→`Rename`, staging-file leaks on both
  failure paths, and each of the three existing-backup verification clauses
  (bytes, `0o077` mode, read-failure-is-not-an-absence) individually.
- **migration** — target vocabulary admitting `1.0.0`, the downgrade gate widened
  by one major, the disclosure-choice vocabulary widened by `""` and by anything,
  the `!IsRegular` re-inspection, the no-op reporting `Changed`, the backup and
  replacement directory fsyncs, the rollback restoring the wrong bytes, and the
  lost `ErrMigrationRecovery` marker.
- **unknown-field refusal** — dropping `DisallowUnknownFields()` dies on all three
  versioned readers *and* at `TestMigrateProductionEntryRefusesInvalidSourceBeforeBackupOrWrite`.
- **bounds** — four bounds widened or loosened by one, all four die at
  `TestConfigurationNumericBoundsAcceptLimitsAndRefusePastLimitsAtProductionEntry`.
- **duplicate backend rejection** — both the `external_trust` and `backend_config`
  duplicate gates die, the former at the migration entry too.
- **read-only downgrade** — the narrowing above, plus the schema-id check, an
  unknown *older* document version, and an unknown reader version.
- **the refusal inventory itself** — I added an unreachable production refusal
  site (audit fails the run) and rebuilt an `*Error` as a raw literal bypassing
  `loaderError` (`TestRefusalInventoryDerivesEveryRefusalFormThePackageDeclares`
  fails). The gate is real, not decorative.

Three survivors are equivalent mutants, not gaps: the downgrade comparison
reduced to major-only (`knownConfigVersion` admits only `1.0.0`/`2.0.0`/`3.0.0`,
so minor and patch are always zero), `encodeVersion2` dropping
`DirectoryInstallations` (unreachable from a v1 source, and pinned structurally),
and `!snapshot.ConfigPresent() || !decoded` reduced to `!decoded`
(`loadAbsentConfig` leaves `Snapshot.configuration` nil, so the first clause is
defensive redundancy).

## Blocking finding

### B1 — this change removed a production refusal and replaced it with self-minted evidence

`loader.go` used to fail closed at the process entry:

```go
home, err := os.UserHomeDir()
if err != nil {
    return Inputs{}, &Error{Operation: "capture user home", Err: errors.Join(ErrInvalidContext, err)}
}
```

This revision removes that refusal and defers it: `OSInputs` now captures
`home, homeErr := os.UserHomeDir()` and stores `homeDirError: homeErr`, and the
nine `HomeDir == ""` branches in `macOSDefault`/`linuxDefault` join it into
`ErrPlatformDefaultUnavailable`. The design is right — with `XDG_CONFIG_HOME` set
there is no reason to refuse — but the only test for it is
`TestResolvePathsDefersButPreservesHomeLookupFailure`, which hand-sets the
unexported field on a fixture:

```go
inputs.HomeDir = ""
inputs.homeDirError = homeFailure
```

That proves `ResolvePaths` propagates whatever is in the field. It does not prove
anything ever puts the real capture there, and no test in the package drives the
home-derived platform defaults at a real entry — every `LoadOS` and `MigrateOS`
test supplies explicit overrides for all five Section 3.2 classes. Two mutants
therefore survive the whole suite:

| Mutant in `OSInputs` | Effect | Suite |
| --- | --- | --- |
| `homeErr = nil` after the capture | the operator gets a bare `ErrPlatformDefaultUnavailable` with the cause of the unavailability stripped | SURVIVED |
| `home = ""` after the capture | every home-derived platform default on every platform refuses; `ax` cannot find its own config without explicit overrides | SURVIVED |

This is the *self-minted evidence* shape, on the one production refusal this
change altered, in a task whose deliverable is refusal evidence. It is also the
cheapest finding in three review cycles to close: the behavior is deterministic
and needs no new seam, because `os.UserHomeDir` reads `$HOME` and `t.Setenv` is
already used at the `LoadOS` entry elsewhere in the file.

I wrote and verified the test. It is attached verbatim as
`TASK-260830-1qf777_reviewer-home-probe-rev4_test.go`; drop it into
`internal/config/loader_test.go` (renaming `TestProbe…` to match the file's
conventions).

- Clean tree: both cases PASS.
- Under `homeErr = nil`: `TestProbeOSInputsPreservesRealHomeFailure` FAILS —
  `LoadOS(no home) dropped the captured os.UserHomeDir cause`.
- Under `home = ""`: `TestProbeOSInputsCapturesRealHomeForPlatformDefaults` FAILS —
  `LoadOS(real home) error = configuration resolve platform default for config-file from platform-default failed`.

The negative case also documents the new behavior the change introduced, which is
currently undocumented anywhere: `LoadOS` with no resolvable home refuses at the
first home-derived class rather than at capture, and carries the OS cause.

## Non-blocking observations

1. `MigrationResult.BackupPath` is still populated before `replaceDurably` runs,
   so a refused migration returns a path that may not exist. The doc comment now
   says so explicitly, which is an acceptable resolution; clearing it on the
   error paths would be strictly better.
2. `writeAll`'s `written == 0 → io.ErrShortWrite` guard is load-bearing in an odd
   way: removing it does not redden, it *hangs* the suite under the fake writer.
   Detected, but a bounded fake would fail faster.
3. `auditRefusalInventory` only runs when `test.run` is empty. That is correct and
   documented, but it means a `-run`-filtered CI lane would silently skip the
   whole inventory gate. Worth a one-line note wherever the CI command is pinned.

## Routing

`to-dev` for B1 only. No production-code defect was found on any probe — the
shipped behavior is correct everywhere I attacked it, including the deferred home
path. What is missing is ~30 lines of evidence at the real entry, already written
and verified above.
