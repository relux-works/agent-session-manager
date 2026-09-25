# TASK-260924-2zboyz retry note (run RUN-260924-d4b700)

## Why this retry exists
CR rev1 validation failed at command 4/30: `go test ./... -count=1 -v` exited 1
because package `internal/traceability/cmd/tracecheck` hit the 10-minute
per-package `go test` timeout (`panic: test timed out after 10m0s`,
`FAIL ... 600.409s`). Log:
`TASK-260924-2zboyz_change-request_rev1-validation.log`.
The leaf package `internal/clonereadback` was green; the failure is in an
unrelated trunk test's runtime cost under shared-machine contention.

## Root cause (measured, not guessed)
Each `TestMain*` case in `tracecheck/main_test.go` copies `internal/` (20 MB,
979 files) into a temp module containing only `go.mod`, then runs the
production binary via `go run` (24 invocations across 16 fixtures total).
Measured on this machine:
- `go run` in a fixture WITHOUT `go.sum`: 12.1s (user+sys 1.2s; ~11s blocked
  in module-graph re-resolution / network+lock wait, repeated per invocation).
- `go run` in the same fixture WITH `go.sum` copied in: 2.1s first run,
  0.9s warm. Same module, same pins, same binary semantics.
- `os.CopyFS` of `internal/`: 1.8s-13.5s depending on contention.
- Package alone before fix: 270.8s (user+sys ~40s, i.e. blocked, not CPU).
Under full-suite contention this exceeded 600s and the package timed out.

## Fix (candidate delta vs rev1)
`internal/traceability/cmd/tracecheck/main_test.go`, two sites only:
`isolatedTracecheckFixture` and the inline fixture in
`TestMainRejectsOneNarrowedAssignedSectionBinding` now also copy `go.sum`
next to `go.mod`. No assertion, no fixture content, no production call, and
no checked-in file changed: the fixture resolves the identical module versions
with the redundant re-resolution skipped. `git diff` is 2x identical 4-line
hunks (1 code line + 3 comment lines each).

Out-of-module declaration (rework-diff-bounded precondition): this edit is
outside `internal/clonereadback/` because the CR validation failure it repairs
lives there; the leaf itself is byte-identical to rev1. No registry edit,
no `.task-board` edit, candidate left uncommitted.

## Verification run by this retry (exit codes observed)
- `go test ./internal/traceability/cmd/tracecheck/ -count=1`: exit 0,
  48.7s alone (was 270.8s; 5.6x). All assertions unchanged and passing.
- `go test ./... -count=1` (harness command 4, non-verbose): exit 0,
  49 packages `ok`, 0 FAIL. Slowest: sessquery 174.0s, tmuxserver 121.2s,
  resumesmoke 118.8s, sessstate 101.5s, config 101.3s,
  tracecheck 100.6s (was >600s timeout), cloneproject 90.7s, sessrepo 86.3s.
  Every package is under the 600s timeout with >3x margin.
- `go test ./internal/clonereadback/ -count=1`: exit 0 (5.3s);
  `-cover`: 95.6% statements.
- `go build ./...`: exit 0. `go vet ./...`: exit 0.
  `GOOS=windows GOARCH=amd64 go vet ./...`: exit 0.
- Harness gofmt gate
  `test -z "$(gofmt -l $(git ls-files --cached --others --exclude-standard -- '*.go'))"`:
  exit 0.

## Reused vs rerun evidence (standing order 10)
- Reran by this retry: tracecheck package, full suite, leaf package + cover,
  build, vet, windows vet, gofmt gate (all exit 0, above).
- Reused from rev1 (source identity unchanged): mutant battery
  (30 rows x2, 24 narrowings killed alone, 1 neutral control survived,
  5 structural plants red, 1 withdrawn survivor with log) and the importer
  set — `internal/clonereadback/` is byte-identical to the rev1 candidate,
  so the rev1 bindings still hold. The only candidate delta is the
  tracecheck test-only timing fix, which no leaf mutant or importer row
  depends on (no import edge either direction).
- Trunk unmoved (`origin/main` == `6d3bff9`); no `refresh-candidate` run.

## Coverage-map precondition
Unchanged from rev1: the brief carries no surface table (brief gap, reported
in `TASK-260924-2zboyz_results.md`); the gate x entry table in
`TASK-260924-2zboyz_conformance-matrix.md` covers every gate. Out-of-contract
rows: the 5 stated bounds in results.md. AC coverage remains 6 of 6 rows
driven through the production entries.
