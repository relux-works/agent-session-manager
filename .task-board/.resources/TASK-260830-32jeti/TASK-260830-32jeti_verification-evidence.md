# TASK-260830-32jeti — Windows-vet conformance fix: verification evidence

## Finding
`GOOS=windows go vet ./...` failed on the merged tree with exactly one error:

```
vet: internal/terminalbackend/terminalbackend_test.go:972:20: undefined: syscall.Mkfifo
```

Two green parents, one red combination: the story branch added the gate before
the `terminalbackend` package existed; trunk added the `syscall.Mkfifo` FIFO
case without the gate.

## Fix (only Go files touched)
- Modified: `internal/terminalbackend/terminalbackend_test.go`
  (removed `syscall` import + FIFO block from `TestDigestFile`, left pointer comment)
- Added: `internal/terminalbackend/terminalbackend_unix_test.go`
  (new `TestDigestFileRefusesFIFO`, same assertions, `//go:build unix`)
- Appended: `LOGBOOK.md` entry (new section only, sibling blocks untouched)

No other Go file modified. No commit, no push (per brief).

## Correction discovered during the fix
The `_unix` filename suffix alone does NOT exclude Windows: `unix` is a build
tag, not a GOOS value, so it matches no `_GOOS` filename pattern. Suffix-only
first attempt still failed windows vet with
`terminalbackend_unix_test.go:23:20: undefined: syscall.Mkfifo`.
The explicit `//go:build unix` line is what excludes Windows. (The repo's
existing `projection_unix_test.go` carries both suffix and explicit
`//go:build darwin || linux`.)

## Gate results (all run directly in the story worktree, exit codes real)
| Gate | Result |
| --- | --- |
| `GOOS=windows go vet ./...` | exit 0 |
| `go vet ./...` (linux) | exit 0 |
| `gofmt -l internal/terminalbackend/` | clean (no output) |
| `go test ./internal/terminalbackend/ -run TestDigestFile -v -count=1` | PASS incl. `TestDigestFileRefusesFIFO` (observed RUN/PASS, not skipped) |
| `go test ./... -count=1` | 16/16 packages ok |
| `go run ./internal/traceability/cmd/tracecheck` | exit 0 (`acceptance_cases=81`, `clauses_discharged=17/428`) |

## Worktree revision (no commit per brief — nothing published)
- Worktree HEAD at fix time: `2512f20` (branch `task-board/story/STORY-260830-3jqsx1`)
- `repository_delta` (my change vs HEAD):
  - `M internal/terminalbackend/terminalbackend_test.go` (-13/+4 lines, syscall removal + pointer comment)
  - `?? internal/terminalbackend/terminalbackend_unix_test.go` (new, 27 lines)
  - `M LOGBOOK.md` (appended entry only)
- All other worktree modifications (`.github`, `README.md`, traceability,
  `internal/provider/`, `internal/provhost/`) are pre-existing story state,
  untouched by this fix.
