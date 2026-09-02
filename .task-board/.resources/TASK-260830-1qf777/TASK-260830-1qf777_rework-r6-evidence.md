# TASK-260830-1qf777 — Round 6 rework evidence (rev5 candidate)

Producer rounds already on this task use r1/r3/r5; this is **round 6**, producing
the **rev5** candidate. Its sweep artifacts are named `-rev5` after the candidate
revision, not the round.

Closes CR-TASK-260830-1qf777 rev4, finding **B1 (self-minted evidence for a
removed refusal)**. That was the only open finding; rev4 confirmed F1/F2/F3 from
rev3 as fixed and found no production defect.

## The finding

The change removed the `OSInputs` capture-time refusal for a failed
`os.UserHomeDir` and replaced it with a deferred `Inputs.homeDirError` field.
The only case exercising it (`TestResolvePathsDefersButPreservesHomeLookupFailure`)
hand-set that unexported field on a fixture, so it constructed the very state it
claimed to observe. Every other `LoadOS`/`MigrateOS` case overrode all five
Section 3.2 path classes, so nothing drove a home-derived platform default at a
real entry. Two mutants survived the whole rev4 suite:

- `homeDirError: homeErr` → `homeDirError: nil` (the operator loses the cause)
- `home = ""` after the capture (every home-derived default refuses)

## What was changed — tests only, no production edit

`internal/config/loader.go` and `internal/config/migration.go` are byte-identical
to the rev4 candidate. Only test files, the ownership registry, its reviewed
digest, and README changed.

### New file `internal/config/loader_home_test.go`

The state is reached the way production reaches it: the case sets or clears the
exact environment variable `os.UserHomeDir` reads (`HOME` on unix, `USERPROFILE`
on Windows, `home` on plan9) and lets the real `OSInputs` capture run. Nothing
assigns `Inputs.HomeDir` or `Inputs.homeDirError`.

- `homeDrivenPlatforms` — the lanes whose Section 3.2 defaults are home-derived
  **and** whose rendered separator matches the host. On a unix host that is
  macOS, Linux and WSL2, so all three lanes are driven through the real
  `LoadOS` entry from one host rather than being host-gated. Windows derives no
  default from the user home and cannot render another lane's separators, so the
  case skips there with that stated reason instead of passing vacuously.
- `homeDerivedPlatformDefaults` — the expected layout, spelled out from the
  specification rather than read back from `resolvePlatformDefault`, so the
  assertion is not self-comparing.
- `overridesExcept` — supplies a real admissible value for every registry class
  except the one under test, so exactly one class falls through to the
  platform-default layer. Without it the first home-derived class to resolve
  masks every later one and a cause dropped at any single site is invisible.
- `carriesCauseMessage` — `os.UserHomeDir` constructs a fresh error per call, so
  identity comparison is impossible; the expected value is the message a real
  `os.UserHomeDir` call returns in the same process environment, not a literal.

`TestLoadOSCarriesTheRealUserHomeFailureAtEveryHomeDerivedClass` — for every
drivable lane × every home-derived class: clears the home variable, asserts the
real `os.UserHomeDir` now fails, drives `LoadOS`, and requires
`ErrPlatformDefaultUnavailable`, the refusing class and source to be exactly the
isolated one, the captured cause to be carried, and the rendered message to not
echo the raw OS cause. 12 subtests on this host.

`TestLoadOSDerivesPlatformDefaultsFromTheRealCapturedUserHome` — for every
drivable lane, resolves **two distinct real homes** in turn. One home cannot
distinguish "the captured home reached the defaults" from "the defaults were
derived from something else that happened to match"; the second run pins that
every home-derived class moved with the capture.

### `internal/config/migration_os_test.go`

`TestMigrateOSMigratesTheHomeDerivedConfigurationAtTheRealProcessEntry` drives
the second production consumer of the same capture. Every other `MigrateOS` case
overrides all five classes; here the captured home is the only thing selecting
the file that the durable mutation rewrites. Backup path, exact backup bytes,
result fields, the migrated document re-loading as current, and staging-leak
absence are all asserted at the home-derived location, per lane.

### `internal/config/loader_test.go`

`TestResolvePathsDefersButPreservesHomeLookupFailure` is kept but relabelled as
the injected half only, naming the two cases that pin the real capture. It was
not widened — the reviewer explicitly ruled that out.

## Mutation sweep

Harness: `TASK-260830-1qf777_home-mutants-rev5.py`; transcript:
`TASK-260830-1qf777_home-mutation-sweep-rev5.log`. Every mutant is applied to
production source only. The gate is the **whole** `internal/config` package with
`-count=1`, never a `-run` mask, so a mutant cannot look dead because an
unrelated case was excluded. Each mutant asserts its anchor occurs exactly once
(0 INVALID) and the tree is restored and re-verified afterwards.

| ID | Mutant | Verdict |
| --- | --- | --- |
| H01 | `homeDirError: homeErr` → `nil` (**rev4 survivor 1**) | RED |
| H02 | `home = ""` after the capture (**rev4 survivor 2**) | RED |
| H03 | capture replaced by a non-empty constant (`os.TempDir()`) | RED |
| H04 | captured cause replaced by a self-minted `errors.New` | RED |
| H05 | captured home corrupted rather than emptied | RED |
| H06–H09 | cause drop at each macOS home-derived site (config-file, data, state, cache) | RED |
| H10–H13 | cause drop at each Linux/WSL2 home-derived site | RED |
| H14 | gate deletion at macOS data-root: unavailable home yields a rooted path instead of a refusal | RED |
| H15 | layout drift at macOS cache-root | RED |
| H16 | layout drift at Linux data-root | RED |

**16 mutants, 16 RED, 0 SURVIVOR, 0 INVALID.** Positive control (unmutated tree)
green before the sweep; restored tree green after it. H06–H13 are the narrowing
direction the delete-only survivors could not reach: the gate is proven per site,
not only at the capture. H10–H13 were host-gated survivors in an earlier draft
that pinned only `hostPlatform`; driving the Linux and WSL2 lanes through the
real `LoadOS` entry from the darwin host turned all four RED.

## Gates — standalone processes, no `tee`, no pipes, real exit codes

| Command | Exit |
| --- | ---: |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l .` (empty output) | 0 |
| `go test ./... -cover -count=1` | 0 |
| `go test ./internal/config -race -count=1` | 0 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 |
| `tracecheck -section 3.2 6.1..6.5 17.1 17.2 17.4` (`assigned_scopes=9`) | 0 |
| `cataloggen -metadata … -contracts … -output … -check` | 0 |
| `go generate ./internal/catalog` + `git diff --exit-code internal/catalog` | 0 |
| `go mod verify` | 0 |
| `git diff --check` | 0 |
| `go test ./internal/config -count=1` under `env -i` with empty `HOME` | 0 |

`internal/config` coverage 93.1% → **93.7%**. All 9 packages ok.

The empty-`HOME` hermetic run matters here specifically: the new cases must not
depend on the developer's real home, and they do not.

## Traceability

- `AC-PATH-001` gains `TestLoadOSCarriesTheRealUserHomeFailureAtEveryHomeDerivedClass`
  and `TestLoadOSDerivesPlatformDefaultsFromTheRealCapturedUserHome`.
- `config-durable-migration-os-entry` gains
  `TestMigrateOSMigratesTheHomeDerivedConfigurationAtTheRealProcessEntry`.
- `reviewedOwnershipCanonicalSHA256` `8885305e…` → `7badbfe8…`. That is the
  review gate working as designed, not formatting drift; the ownership JSON was
  edited surgically as text (12 added lines) with no reformatting.
- No new section binding, no new acceptance-case ID, no CLI, doctor or backend
  capability advertised.

## README

One paragraph added to the Section 3.2 description stating why the home lookup
failure is deferred rather than refused at capture time (Windows derives no
default from the user home; several classes are satisfied by an XDG override),
that the captured cause travels into every home-derived class's refusal, and
that both halves are proven at the real entries. It claims no capability.

## Not done / limits

- The Windows lane is skipped with a stated reason on any host: production
  derives no path default from the user home there, so a home-capture case has
  nothing to pin. This is a real absence in the specification's Windows layout,
  not missing coverage.
- No production code was changed this round. If the reviewer wants the
  capture-time refusal restored instead of deferred, that is a product decision
  about Windows and XDG-satisfied classes, not a test gap.
