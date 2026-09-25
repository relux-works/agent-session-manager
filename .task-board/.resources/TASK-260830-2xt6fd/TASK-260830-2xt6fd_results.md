# TASK-260830-2xt6fd developer handoff results

Status: **ready for review handoff**. The managed Story candidate remains
uncommitted at the recorded checkpoint for the handoff snapshot.

## Delivered behavior

- `gitsnap.AssembleProvisional` now drives real repository-local pack creation,
  isolated pack/index import, exact object inventories, raw/logical index
  agreement, blob descriptors, and workspace root/member/child closure.
- `(*tmuxserver.Lifecycle).CaptureGitWorkspace` is the production coordinator.
  It admits only a current `quiescing` instance with matching input/provider
  boundary receipts or `stopped`; it rechecks state and source identity after
  assembly and never releases, restores, or unlocks the instance.
- Source-change, path/partition, byte/digest, unsupported-input, crash/retry,
  and stop-point refusal cases are exercised through production entries.
- `UnixDialer.Dial` distinguishes a missing socket under an existing directory
  from a missing parent or failed parent read; unknown remains unknown.
- The v0.7.0 ownership registry includes the three Story leaves. README's
  coverage pin matches the final tracecheck report. No CLI/doctor capture
  capability was added.

## Measured coverage

The six-row final-leaf acceptance matrix is driven **6 of 6 rows** through the
call sites listed in `coverage-map.md`. The brief supplied no surface table;
that is recorded as a brief gap, not as waived coverage. The handoff scope map
records the named out-of-contract clauses and each acceptance-criteria clause
that bounds them.

## Reviewer test retention

The tracked baseline has 3,347 distinct `Test*` names across Go files. The
candidate retains all of them; the candidate has 3,350 names and the measured
missing-name set is empty. This includes the hidden task-resource test files.
`TestContentNormativeWorkspaceBytes` moved between test files with its name
unchanged and passes when run alone. The two committed home-probe reviewer tests
remain in their original task-resource file; that exact file was materialized
temporarily into `internal/config` to run both original names, and both passed.
The temporary copy was removed. Audit counts, zero-missing output, and raw test
logs are included in the evidence archive.

Full tracecheck: exit 0; `contracts=64`, `normative_sections=36`,
`acceptance_cases=163`, `bindings=73`, `full=4`, `partial=11`, `sliver=13`,
`clauses_discharged=106/597`. README carries the same coverage pin.

## Mutant evidence

Across the recorded narrowing runs, 19 narrowing plants were killed by named
tests. The measured `N-assembly-closure-admits-one-missing-child` plant
survived: the malformed fixture is still rejected by the required repository
tree binding, so it proves no independent child-visitor gate. This bound is
not counted as independent evidence. The harmless comment-only `C-control`
survived as expected and is not a gate mutant. Full per-plant table, killer
names, survivor bound, and raw-log paths are in `coverage-map.md` and the
evidence archive.

Fresh final plants:

| Mutant | Narrowed admission | Named test / result |
| --- | --- | --- |
| `N-capture-admits-parked` | Admits only `parked` at capture admission | `TestCaptureRejectsParkedEvenWithPriorQuiescenceReceipt` alone — killed; behavior exit 1, harness exit 0 |
| `N-capture-restores-with-boundary-token-preserved` | Uses `restore` instead of the owner's `wait-safe-boundary` while retaining receipt/parser tokens | `TestCaptureGitWorkspaceQuiescedAssemblesAndRechecks` and `TestCaptureNeverTransitionsOutsideStopPointOwnerSet` — killed; behavior exit 1, harness exit 0 |
| `N-unix-dialer-missing-parent-admits-stale` | Maps only ENOENT with an absent containing directory to `stale` | `TestUnixDialerMissingDirectoryIsUnknown` alone — killed; behavior exit 1, harness exit 0 |

## Validation

All commands below ran as standalone processes. Logs are in
`.temp/TASK-260830-2xt6fd/` and included in the evidence archive.

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/tmuxserver -count=1 -v` | 0 | `tmuxserver-full-03.log`; package reports 337.331s |
| `go test ./internal/gitsnap -count=1 -v` | 0 | `gitsnap-final-04.log` |
| `go test ./internal/tmuxserver -run 'TestCapture' -count=1 -v` | 0 | `tmuxserver-capture-final-01.log` |
| `go test ./internal/config -run '^TestProbeOSInputs(PreservesRealHomeFailure|CapturesRealHomeForPlatformDefaults)$' -count=1 -v` | 0 | Both original reviewer names pass from the exact attached test source; `reviewer-tests-materialized-01.log` |
| `go test ./internal/gitsnap -run '^TestContentNormativeWorkspaceBytes$' -count=1 -v` | 0 | Original name preserved after file move; `reviewer-test-preserved-01.log` |
| Nine bounded `go test -cover <package group>` shards | 0 each | `go-cover-shard-01.log` … `go-cover-shard-09.log`; audit verifies all 46/46 packages (`go-cover-audit-01.log`) |
| `go vet ./...` | 0 | `go-vet-host-03.log` |
| `GOOS=windows GOARCH=amd64 go vet ./...` | 0 | `go-vet-windows-02.log` |
| `go build ./...` | 0 | `go-build-host-03.log` |
| `gofmt -l internal`; `test ! -s gofmt-final-02.log` | 0 / 0 | `gofmt-final-02.log` |
| `git diff --check` | 0 | `git-diff-check-final-02.log` |
| `golangci-lint run --new-from-rev=HEAD` | 0 | `golangci-lint-new-05.log`; no new delta findings |
| `golangci-lint run ./...` | 1 | `golangci-lint-01.log`; 136 repository-wide findings, including unchanged paths. Delta-specific lint is green; the full repository command remains red. |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `tracecheck-full-10.log` |
| `go run .../tracecheck -section 10.4` | 1, expected scope refusal | 21/25; clauses #3, #5, #15, #25 are out of contract |
| `go run .../tracecheck -section 12.1` | 1, expected scope refusal | 0/1; checkpoint publisher/linkage is out of contract |
| `go run .../tracecheck -section 12.2` | 1, expected scope refusal | 2/6; clauses #3–#6 are destination-materialization gates |
| `go run .../tracecheck -section 12.3` | 1, expected scope refusal | 2/5; materialization steps/matrix/network rule are out of contract |
| The three final mutant harness commands above | 0 each | Harness verdicts and raw per-plant logs are archived; each mutated behavior command exits 1 on its named test |

Earlier monolithic `go test ./... -v` attempts were not green: two pre-fix
runs exited 1 on the dialer classification, and a third attempt was interrupted
at the call limit (exit 143). A pre-fix full `tmuxserver` attempt also exited 1
because long test temp paths exhausted `sun_path` before custody checks. The
fixture now gets its directory from `t.TempDir()` and relocates that owned
directory under canonical `/var/tmp` with `t.Cleanup`; the final full
`tmuxserver` run and all coverage shards pass. Four broad mutation controls
exceeded their 240-second harness limit and are diagnostics only.

## Specification boundary and evidence map

The clause crosswalk and line ranges are in `spec-crosswalk.md`. Section bodies
for v0.5.0 → pinned v0.7.0 are byte-identical where compared. The final leaf
does not implement checkpoint publication (§12.1#1), Git destination
materialization (§12.2#3–#6; §12.3 steps 2–8), external provider transfer
response admission (§10.4#15), cross-platform runtime proof (§10.4#25), or
capture returning to `active`; the §4.C lifecycle vocabulary is closed. The
acceptance clauses bounding these rows are listed in `coverage-map.md`.

Supporting artifacts: `coverage-map.md`, `axis-inventory.md`,
`importer-outcome-grid.md`, and `spec-crosswalk.md`. The archive includes these
documents, the final logs, selected historical failure logs, mutation result
JSON, and one raw behavior log per plant. Archive size is kept below 1 MiB.
