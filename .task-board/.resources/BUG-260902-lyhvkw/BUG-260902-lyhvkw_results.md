# BUG-260902-lyhvkw: cap canonical decode recursion depth

Worktree: `.temp/STORY-260830-2wdm5e/worktree`, branch `task-board/story/STORY-260830-2wdm5e` at fork point `ad72751`.
An earlier developer run (RUN-260901-ae07f5) wrote the fix and evidence logs 00-13 and was cancelled by the operator (exit 143) before handoff.
This run (RUN-260901-4717b6) verified the source matches that run's snapshot byte-for-byte and re-ran every gate itself (logs 14-21).

## Fix

- `internal/canonicaljson/canonical.go`: declared `MaxNestingDepth = 256` with a stated rationale (fatal, uncatchable stack overflow at depth 1,000,000 with a 2 MB input under the 5,242,880-byte identity cap; deepest pinned v0.5.0 closed shape decodes at depth 38, under half the bound; 256 frames cost tens of KB of stack; bound sits below encoding/json's own 10,000-level limit so callers always see the typed refusal).
- `decodeValue(decoder, depth)` refuses a container at level `depth+1 > MaxNestingDepth` with typed `ErrInvalidJSON` before recursing. `decodeStrict` is the single shared decode path, so `Canonicalize`, `CalculateObjectIdentity` and `VerifyObjectIdentity` (via `prepareObjectIdentity`) inherit the bound.
- `internal/canonicaljson/nesting_depth_test.go`: accept-at-256 / refuse-at-257 through `Canonicalize` for nested arrays, nested objects and alternation; both identity entries proven to admit the at-limit document (the downstream extension depth-4 gate is what refuses it, typed `ErrInvalidIdentity`, not `ErrInvalidJSON`) and to refuse one past the limit with the typed decode error; regression case replays the exact 2,000,000-byte one-million-level nested-array document at all three entries; deepest pinned closed shape headroom check.
- Tests pin the bound from a test-local literal (`documentedMaxNestingDepth`), so widening or narrowing the production constant reddens the suite.
- `README.md` line 146 and `internal/canonicaljson/testdata/constraint-enumeration.md` row "Decoder robustness bound" document the bound.

## Reproduction of the original crash (unpatched main 48db30b)

`01-repro-unpatched.log`: `go run` of `repro/main.go` against the main tree copy in `main-tree/`:
`runtime: goroutine stack exceeds 1000000000-byte limit` / `fatal error: stack overflow` / `exit status 2` for a 2,000,000-byte input.

## Mutant evidence (re-run this session, `14-mutants-rerun.log`)

| Mutant | Command | Exit | Reddened tests |
| --- | --- | ---: | --- |
| widen-512 | `go test ./internal/canonicaljson -count=1` | 1 | pin, accept/refuse, identity inherit, regression |
| widen-257 | same | 1 | pin, accept/refuse, identity inherit, regression |
| narrow-255 | same | 1 | pin, accept/refuse, identity inherit, regression |
| delete-gate | `go test ./internal/canonicaljson -count=1 -run 'NestingDepth|NestedArrayRegression'` | 1 | accept/refuse, identity inherit; regression test aborts the binary with `fatal error: stack overflow` (original crash shape) |

Source restored after each mutant and diffed against the fixed snapshot: identical.

## Gates re-run this session (standalone processes, real exit codes)

| Gate | Exit | Log |
| --- | ---: | --- |
| `gofmt -l` over tracked+untracked Go files (empty) | 0 | inline |
| `go vet ./...` | 0 | inline |
| `go build ./...` | 0 | inline |
| `go test ./internal/canonicaljson -run 'NestingDepth|DeepestPinned|NestedArrayRegression' -v` | 0 | inline |
| `go test ./... -count=1 -v` | 0 | 15-go-test-all-v.log |
| `go test ./... -cover -count=1` | 0 | 16-go-test-cover.log |
| `go test ./... -race -count=1` | 0 | 17-go-test-race.log |
| 4 configured fuzz seeds (`-fuzztime=100x`) | 0 each | 18-fuzz-seeds.log |
| `go run ./internal/traceability/cmd/tracecheck` full and `-section 1.6 10.1 10.2 10.3 10.4 17.3` | 0, 0 | 19-tracecheck.log |
| cataloggen `-check` | 0 | 20-catalog-check.log |
| `GOOS=linux` / `GOOS=windows` `go build ./...` | 0, 0 | 21-cross-build.log |

## Coverage

| Package | Baseline (00) | After (16) |
| --- | ---: | ---: |
| internal/canonicaljson | 87.1% | 87.2% |
| all other packages | unchanged | unchanged |

## Not run

Nothing from the configured gate list was skipped. Prior-run logs 00-13 were kept for provenance only; every verdict above comes from this session's own re-runs.
