# BUG-260902-lyhvkw revision 3: accepted rev-2 work reapplied onto advanced trunk

Workspace: `.temp/STORY-260902-3os1kh/worktree`, branch `task-board/story/STORY-260902-3os1kh`, HEAD `a351afd` (equal to `origin/main` at run time, `git rev-list --count HEAD..origin/main` = 0).

## What changed versus the accepted rev-2 patch

The attached `BUG-260902-lyhvkw_accepted-rev2-work.patch` was reapplied with `git apply --3way`. Every hunk applied cleanly except the digest pin in `internal/traceability/traceability.go`: trunk had independently repinned `reviewedOwnershipCanonicalSHA256` to `7badbfe8...`, so the rev-2 value `e37f8208...` no longer described the current registry. Resolution: take the trunk value, run tracecheck, and repin to the projection digest it reported for the registry that now also lists the nesting-depth evidence tests: `9f7737cb07f853012fcc9e2359981e20e9b65df622f9da7fa4935a2180cd04b0`. No product code, test, or documentation content differs from rev-2 other than an added LOGBOOK entry describing this reapply.

## Gate evidence (each run as a standalone process, real exit codes)

| Command | Exit | Log |
| --- | ---: | --- |
| `go run ./internal/traceability/cmd/tracecheck` (before repin) | 1 | `tracecheck-01.log` (expected red: digest differs) |
| `go run ./internal/traceability/cmd/tracecheck` (after repin) | 0 | `tracecheck-02.log` |
| `go test ./internal/canonicaljson -cover -count=1` | 0 | `canonicaljson-test-01.log` (87.2%) |
| baseline `go test ./internal/canonicaljson -cover -count=1` at trunk `a351afd` | 0 | `baseline-coverage-01.log` (87.1%) |
| `go vet ./...` | 0 | `vet-01.log` |
| `go build ./...` | 0 | `build-01.log` |
| `cataloggen ... -check` | 0 | `catalog-check-01.log` |
| `go test ./... -cover -count=1` | 0 | `test-cover-01.log` |
| `go test ./... -v -count=1` | 0 | `test-v-01.log` |
| `gofmt -l internal/` | 0, empty | inline |

Coverage: canonicaljson 87.1% -> 87.2%, no regression in any other package.

## Mutant evidence (isolated worktree copy, `-run NestingDepth`)

| Mutant | Exit | Failing tests |
| --- | ---: | --- |
| widen `maxNestingDepth` 256 -> 512 | 1 | Pinned, CanonicalizeRefuses, IdentityEntriesRefuse, Regression |
| narrow 256 -> 128 | 1 | Pinned, CanonicalizeAccepts, CanonicalizeRefuses, IdentityEntriesAccept, IdentityEntriesRefuse, Regression |
| delete the `depth >= maxNestingDepth` gate | 1 | `runtime: goroutine stack exceeds 1000000000-byte limit` / `fatal error: stack overflow` via `TestNestingDepthRegressionTwoMegabyteArrayReturnsTypedError` |

Production call site: `decodeValue` inside `decodeStrict` (`internal/canonicaljson/canonical.go`), shared by `Canonicalize`, `CalculateObjectIdentity`, and `VerifyObjectIdentity`.

Logs live under `.temp/BUG-260902-lyhvkw/` in the story worktree. The staged scope is the eight files listed by `git status`; nothing is committed in this run because integration into trunk is the orchestrator's step.
