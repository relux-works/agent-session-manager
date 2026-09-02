# TASK-260830-1qf777 — Review Verdict, CR rev2

- Run: `RUN-260901-de6b47`
- Change Request: `CR-TASK-260830-1qf777-2` revision `2`
- Base `48db30b59e5e1bbc5e0cf73ec2e0e0eec3d215d1` -> candidate tree `a9b9e5fad99897cd11906d89cc44c21251f5a4cd`
- Worktree `git write-tree` re-derived `a9b9e5fa...`, so every command below ran against the exact reviewed candidate.

## Verdict: CHANGES REQUESTED -> `to-dev`

Rejected on two narrow evidence gaps, not on migration behavior. Everything the
rev1 review asked for landed and holds under an independent attack.

## What rev2 fixed, verified independently

All six rev1 findings are closed. I did not reuse the producer's mutant list; I
wrote 40 mutants of my own against the production gates and ran each as
`go build ./... && go test ./internal/config/ -count=1` in a separate tree copy.
**38 died, 2 survived, and both survivors are provably subsumed rather than
bypasses.** Full transcript in `TASK-260830-1qf777_reviewer-mutant-evidence.log`.

Gates that died, by rev1 finding:

| rev1 finding | reviewer mutant | killed by |
| --- | --- | --- |
| (1) existing-backup integrity | R06b drop exact-content clause; R07b drop owner-only clause; R08c delete the gate | `TestMigrateRefusesExistingBackupThatIsNotTheExactOwnerOnlySource` |
| (2) `ErrMigrationTarget` free-floating | R02 admit `Version1` as target; R03b delete the gate | `TestMigrateRefusesEveryTargetOutsideTheUpgradeVocabulary` |
| (3) absent source + non-regular TOCTOU | R11, R10 | `TestMigrateRefusesAbsentSourceWithoutWriting`, `TestMigrateRefusesSourceThatStoppedBeingARegularFileAfterLoad` |
| (4) disclosure gate empty-only | R04 narrow gate to empty-only | `TestMigrateRefusesDisclosureChoiceOutsideTheClosedVocabulary` |
| (5) `ErrMigrationRecovery` rollback branch | R24 delete the report; R25 delete the rollback; R26 delete the fsync retry | `TestMigrateReportsRecoveryFailureWhenRollbackAlsoFails` |
| (6) inventory blind to a second refusal form | moved-marker attack below | `deriveRefusalInventory` self-coverage audit |

Also killed: R01 downgrade refusal, R09 backup permissions widened to 0644,
R12b/R12c read-only downgrade determination deleted and narrowed to `>1`,
R13/R14/R15 `AssessCompatibility` envelope refusals, R16 `os.Lstat` reverted to
`os.Stat`, R18/R19 `sync.max_parallel_chunks` widened past 32 and below 1,
R20 `DisallowUnknownFields` deleted, R21b/R22b duplicate `backend_id` rejections
deleted, R23b durable replace skipped entirely, R27 back up the new document
instead of the original, R28 name the backup after the target version, R29 remove
the already-at-target short circuit, R30b replace in place with a truncated
non-atomic write. Bounds die in both directions and vocabularies die to narrowing.

The derived refusal inventory now genuinely proves its own coverage. I tried to
game `TestRefusalSubsumptionInventoryIsPinned`, which compares only the *count* of
`config-refusal-subsumed` markers and never their positions, by MOVING a marker so
the count stayed 7. It reddened: `configuration refusal call sites without an
exercised negative path: writer.go:54`. Any removal reddens, so the count can only
rise and a rise reddens. The ratchet holds; not a finding.

Both traceability gates fail closed under attack: a fabricated `section:99.9`
binding is rejected as a self-minted owner, and an ownership edit without
re-pinning the digest is rejected on the projection digest.

Repository gates all pass on the candidate: `go build`, `go vet`, `gofmt -l`,
`go test ./...`, `go test ./... -cover` (internal/config 91.8%), `go mod verify`,
`git diff --check`, `tracecheck`, and `go test ./internal/config/` under
`env -i` with `HOME=/nonexistent-empty-home`.

## F1 — `os.Stat` -> `os.Lstat` silently refuses symlinked directories, untested in either direction

`loader.go:187` swaps `Stat: os.Stat` for `Stat: os.Lstat`. `Load` uses that one
seam for two different questions: the regular-file contract on `ConfigFile`, and
the *directory* kind check in `validateRootKinds` and `loadAbsentConfig`. Only the
first needs `Lstat`. Driving the real `LoadOS` entry on the candidate:

```
PROBE symlinked DataRoot                     -> ErrRootNotDirectory
PROBE absent config under symlinked parent   -> ErrConfigParentNotDirectory
```

A symlink to a directory is a usable directory, and symlinked XDG / Application
Support roots are a mainstream dotfiles layout, so this refuses a working setup
with an error class that says the path is not a directory when it is one.

`TestLoadOSRefusesRealSymlinkAtProductionEntry` covers only the config *file*.
Nothing asserts the directory outcome in either direction, no README sentence
mentions it, and `loader.go:187` carries no comment. My R16 mutant (revert to
`os.Stat`) dies only on the config-file test, which confirms the directory
consequence is entirely unpinned. Per this task's own "prove, or report nothing",
an untested, undocumented production behavior change at the real entry point is
not acceptable regardless of which way it is resolved.

I could not establish the spec's position: `SPEC.md` is not checked into this
repository and `internal/specpin/v0.5.0.lock.json` carries no prose. Reporting
that as unknown rather than inferring it.

Resolve it either way, but resolve it explicitly:

- If refusing symlinked roots is intended, cite the §6 clause, add a negative test
  for a symlinked root and a symlinked config parent, and say so in the README
  paragraph that already describes the loader's kind checks.
- If it is not intended, split the seam so `Lstat` guards only the `ConfigFile`
  regular-file contract while directory-kind checks keep following symlinks, and
  add the positive test that a symlinked root loads.

## F2 — `MigrateOS`, the OS production entry, has 0.0% coverage

`go tool cover -func` on the candidate reports `MigrateOS 0.0%`. The README this CR
writes states "`Migrate` and `MigrateOS` are the explicit Section 6.4/6.5 durable
migration entries", and the ownership registry binds §6.4/§6.5 to `Migrate` only.
`LoadOS` — the sibling OS entry — carries four real production-entry tests
(`TestLoadOSCapturesExplicitEnvironmentAtProductionEntry`,
`...RefusesRealSymlinkAtProductionEntry`, `...AppliesExplicitOverridesAtProductionEntry`,
`...RefusesInvalidRuntimePlatformBeforeReadingOS`). The durable-migration OS entry
gets none, so the composition of `OSInputs` with `Migrate` — including that
`OSInputs` now hands it `os.Lstat` — is never driven.

Add at least one `MigrateOS` test that migrates a real on-disk file through
explicit `Overrides` and asserts the backup and replacement, plus one refusal
(an invalid runtime platform is the cheapest, mirroring the `LoadOS` lane).

## F3 — `(*MigrationError).Error()` has 0.0% coverage, so its rendering claim is unproven

`migration.go:50` documents "MigrationError renders no machine-local path or
document content", and `replaceDurably` joins raw `os` errors carrying absolute
paths into the wrapped chain. `errors.Is` never calls `Error()`, and the `%v`
formats in the migration tests only run on failure, so the method is never
executed. The equivalent loader claim *is* proven, by
`TestLoadErrorsDoNotEchoSelectedPathValues` and
`TestErrorFormattingNeverEchoesWrappedDetails`. Extend one of those to
`*MigrationError` — a rendered migration refusal must not contain the temp
directory, the config filename, or the backup path.

## Not stop-the-line

No external blocker and no human-only decision. F1 needs a spec reading the
producer can perform inside the pinned scope; F2 and F3 are additive tests.
