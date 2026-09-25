# TASK-260924-3n78rv retry note (CR rev2 validation failure → rev3)

Run: RUN-260924-c14411 (autonomous recovery successor of RUN-260924-d25897).
No code changed in this run: the tree is byte-identical to the rev2 CR
patch (verified with `diff -r` against the applied patch: identical),
HEAD `618d78d` still descends from trunk `6d3bff9` (verified after a
fresh `git fetch`), and the candidate scope is still exactly the 17 new
paths under `internal/cloneplanning/`. This run diagnoses the rev2
validation failure and re-runs the tree-bound validation for the retry.

## The failure (from `TASK-260924-3n78rv_change-request_rev2-validation.log`)

The configured suite stopped at command 4 of 30, `go test ./... -count=1
-v`, exit 1:

- Package `internal/traceability/cmd/tracecheck` hit the 10-minute
  `go test` timeout (`panic: test timed out after 10m0s`) while
  `TestMainRejectsOneNarrowedAssignedSectionBinding` was mid-`CopyFS`
  and three parallel tests waited at the parallel barrier.
- No test failed: the package log shows subtests passing (27–41s each)
  until the timeout killed the run. `coverage_unit=exact_command_shard
  required=30 green=3 failed=1 missing=26`.

## Diagnosis: load-contention timeout, not a candidate defect

1. **Zero dependency edge.** `go list -deps
   ./internal/traceability/cmd/tracecheck/ | grep -c cloneplanning`
   returns 0, and no `.go` file outside `internal/cloneplanning/`
   references the package. The timed-out package cannot observe our
   behavior; our 17 small files add only milliseconds to its `CopyFS`.
2. **The exact timed-out test passes alone on this tree.**
   `TestMainRejectsOneNarrowedAssignedSectionBinding` run alone
   (`-count=1 -timeout 9m`): PASS in 33.89s, rerun PASS in 37.29s,
   both exit 0 (`heavy-isolation.log`).
3. **The package passes in short mode**: `go test -short -count=1
   ./internal/traceability/cmd/tracecheck/` exit 0 (36s).
4. **The full suite is green on this tree at bounded parallelism**:
   48/48 packages across 4 shards (`-p 2 -count=1`, exits 0/0/0/0),
   including `tracecheck` itself (249.9s) and `tmuxserver` (281.0s).
   At default parallelism these heavy binary-building packages compete
   with 47 others and blow past the 600s package timeout under load.
5. **Load documented**: `uptime` showed load 23.2/19.2/18.4 falling to
   14.4/15.1/16.8 on 16 CPUs during validation. The rev1 reviewer and
   the rev2 producer both recorded this same default-parallelism
   load-flake class on this host.

## Reran myself (real exit codes, this tree)

| Command | Exit | Evidence |
|---|---|---|
| `diff -r` applied rev2 patch vs `internal/cloneplanning/` | identical | (worktree state) |
| `git fetch origin main` + trunk-ancestor check | 0, descends | (worktree state) |
| `gofmt` clean check over tracked+untracked `*.go` | 0, clean | (rerun post-cleanup) |
| `go build ./...` | 0 | (run output) |
| `go vet ./internal/cloneplanning/` | 0 | (run output) |
| `go test -count=1 ./internal/cloneplanning/` | 0 | `package.log` |
| `go test -count=1 -cover ./internal/cloneplanning/` | 0, 97.8% statements | (run output) |
| `go test -count=3 ./internal/cloneplanning/` | 0 | (run output) |
| 9 owner suites (clonefidelity/cloneplan/clonebundle/cloneproject/clonesnap/environ/scalar/sessadapter/canonicaljson) | 0, 9/9 ok | (run output) |
| `go test -short -count=1 ./internal/traceability/cmd/tracecheck/` | 0 | (run output) |
| heavy isolation probe (exact timed-out test, alone, ×2) | 0, 0 | `heavy-isolation.log` |
| full suite shard 1–4 (`-p 2 -count=1`, 12 pkgs each) | 0/0/0/0, 48/48 ok | `shard1..4.log` + `.args` |
| `go vet ./...` | 0 | `vet.log` (empty) |
| `GOOS=windows GOARCH=amd64 go vet ./...` | 0 | `winvet.log` (empty) |
| scratch-index `git diff --cached --check` incl. untracked | 0 | (rerun post-cleanup) |

## Reused from rev2 (source/test/config identity unchanged)

Mutant battery (50 narrowings KILLED + neutral control SURVIVED),
AST guard control probes, old-test-survives / new-test-kills pair,
importer byte-identity grid (232 files), and the 97.8% coverage
figure (re-measured equal this run) are bound to the identical
bytes verified above, so they are reused, not rerun.

## What the retry changes

Nothing in the candidate. The CR rev2 validation failure is an
environmental timeout in an unrelated package; the evidence above
shows the tree is green. This handoff re-enters the same candidate
so the configured suite reruns automatically.
