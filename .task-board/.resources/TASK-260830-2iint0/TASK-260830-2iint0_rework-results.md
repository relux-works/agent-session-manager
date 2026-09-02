# TASK-260830-2iint0 rework outcome

## Scope and decisions

- Reworked the reviewed configuration path-loading slice against
  `relux-works/agent-session-manager-spec` commit
  `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`; the local pinned source checkout
  resolved to that exact commit.
- `Load` now admits the Section 3.2 not-yet-created configuration-file value
  kind only when its parent is an existing directory. `Snapshot.ConfigPresent`
  distinguishes that state from an existing empty file.
- Missing parent, non-directory parent, parent inspection failure, existing
  non-regular config kind, config read failure, and root inspection failure are
  distinct outcomes. No durable state is mutated.
- Root validation evidence is derived from `OverrideRegistry`: every
  non-config registry member is tested for wrong-kind and inspection-failure
  refusal. Config-file kind evidence covers directory, named pipe, socket,
  device, and symlink modes.
- `LoadOS` is driven with task-owned temporary paths and explicit values for
  all five documented `AX_*` variables. `OSInputs` invalid-platform refusal is
  also exercised directly. Tests do not rely on global Git configuration or
  developer-machine path values.
- The premature sole `section:6.1` ownership claim was removed. This task owns
  `section:3.2` plus `AC-PATH-001`; the sibling versioned-schema loader must own
  the whole Section 6.1 entry because unknown-key and secret-field refusals are
  outside this path-selection slice. README now states that boundary without
  advertising CLI, doctor, migration, or runtime capability support.
- No numeric or Unicode-character bound is declared by this slice. No durable
  mutation exists, so crash rollback evidence is not applicable; repeated
  read-only loading and isolated snapshot behavior remain covered.

## Gate evidence

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/config -count=1 -v` | 0 | `go-test-config-rework-04.log` |
| `go test ./internal/config -coverprofile=... -count=1` | 0 | `go-cover-config-rework-03.log`; 86.5% statements |
| `go tool cover -func=...` | 0 | `LoadOS` 100%, `Load` 90.9%, absent/root validators 90.9%/92.9% |
| `zsh .temp/TASK-260830-2iint0/mutate-rework.sh` | 0 | 15/15 independently narrowed mutants killed; 0 survivors |
| `go test ./... -v -count=1` | 0 | `go-test-all-rework-01.log` |
| `go test ./... -cover -count=1` | 0 | `go-cover-all-rework-01.log`; config 86.5% |
| `go test ./... -race -count=1` | 0 | `go-race-all-rework-02.log` |
| `go vet ./...` | 0 | `go-vet-rework-01.log` |
| `go build ./...` | 0 | `go-build-rework-01.log` |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | `go-build-linux-amd64-rework-01.log` |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | `go-build-windows-amd64-rework-01.log` |
| `gofmt -l internal/config internal/traceability` plus empty-output assertion | 0 | `gofmt-rework-01.log` is empty |
| full `tracecheck` | 0 | 60 contracts, 36 normative sections, 30 acceptance cases, 30 fixtures |
| `tracecheck -section 3.2` | 0 | one scoped implementation owner |
| catalog generation `-check` | 0 | `catalog-check-rework-01.log` |
| `git diff --check` | 0 | `git-diff-check-rework-01.log` |

The final mutation campaign started only after a green uncached config suite.
It killed per-class root skips for data/state/cache/runtime, the `IsDir`-only
config gate, the `OSInputs` guard removal, absent-config rejection, empty-flag
misclassification, both nil-filesystem-function removals, missing-parent
collapse, parent-kind removal, both `ConfigPresent` forgeries, and Windows
above-root traversal admission. Each mutant log contains the named failing
production-entry test.

## Non-green and anomalous runs

- Pre-repin `tracecheck -section 3.2` exited 1 as expected and reported the new
  semantic ownership digest; after explicitly updating the reviewed pin, both
  scoped and full tracecheck gates exit 0.
- `go-test-config-rework-02.log` exited 1 because the new Windows fixture used a
  POSIX captured working directory; the fixture was corrected to native Windows
  inputs and rerun green.
- `go-cover-config-rework-02.log` exited 1 on a test compile error (`:=` after
  an existing `err`). A mutation run made before discovering that compile error
  was invalidated and not counted. After fixing it, the baseline ran green and
  all 15 mutants were rerun from scratch with specific failure output.
- Two early mutation-harness attempts exited 1/2 on zsh harness bugs (read-only
  `status` and a multiline `rg` marker). Both were fixed before the counted
  campaign.
- `task-board validate` exited 0 but reported 256 pre-existing
  `MISSING_ACTIVITY` issues elsewhere on the authoritative board; this is not
  represented as a clean semantic validation result and no board files were
  edited directly.
