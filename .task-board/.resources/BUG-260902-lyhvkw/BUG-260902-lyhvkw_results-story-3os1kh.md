# BUG-260902-lyhvkw — cap canonical decode recursion depth (STORY-260902-3os1kh workspace)

Run: RUN-260902-605145, developer, worktree `.temp/STORY-260902-3os1kh/worktree`,
branch `task-board/story/STORY-260902-3os1kh` forked from main `48db30b`.

An earlier developer run wrote an equivalent fix in the spent STORY-260830-2wdm5e
workspace (fork point `ad72751`, not a trunk descendant), so Change Request
construction failed. This run re-implemented the fix from scratch in the
trunk-descended workspace and re-ran every gate itself; nothing below is accepted
from the earlier run's evidence.

## Change

- `internal/canonicaljson/canonical.go:37` declares `maxNestingDepth = 256` with
  the rationale in its doc comment (deepest pinned closed shape ≈ depth 40,
  extensions ≤ 4, >6x headroom, far below encoding/json's 10,000 cap and jcs's
  unbounded re-parse).
- `decodeValue(decoder, depth)` refuses at `canonical.go:362` before opening a
  container past the bound with typed `ErrInvalidJSON`
  (`nesting depth 257 exceeds maximum 256`). `decodeStrict` calls it at depth 0,
  so `Canonicalize`, `CalculateObjectIdentity`, and `VerifyObjectIdentity`
  inherit the gate.
- `internal/canonicaljson/nesting_depth_test.go`: pin test, accept-at-256 and
  refuse-at-257 at every public entry in array/object/mixed shapes (identity
  entries prove decode passage by the refusal coming from the later Section 1.6
  extension depth-4 gate with `ErrInvalidIdentity`, never `ErrInvalidJSON`), and
  the 2,000,000-byte one-million-level regression input through all three
  entries. Tests use a test-local literal 256 so a widened or narrowed constant
  reddens the suite.
- Fuzz seed `testdata/fuzz/FuzzCanonicalizeRoundTrip/nesting-depth-over-limit`
  (257 nested arrays) replays on every `go test`.
- `testdata/constraint-enumeration.md`: decoder robustness bound row.
- `internal/traceability/ownership.v0.5.0.json`: new tests registered under
  `canonical-jcs-rfc8785`, `canonical-object-identity`, and
  `canonical-identity-refusal`; `reviewedOwnershipCanonicalSHA256` repinned in
  `internal/traceability/traceability.go:42` from `30403d58…` to `e37f8208…`.
- `README.md`: documents the bound.

## Gates (all run by this run as standalone processes; exit codes real)

| Gate | Command | Exit | Log |
| --- | --- | ---: | --- |
| package coverage baseline (clean tree) | `go test ./internal/canonicaljson -cover -count=1` | 0 (87.1%) | baseline-canonicaljson-cover-01.log |
| package coverage after | same | 0 (87.2%) | canonicaljson-cover-03.log |
| repo tests | `go test ./... -v -count=1` | 0 | test-all-v-02.log |
| repo coverage | `go test ./... -cover -count=1` | 0, no package regressed | test-all-cover-02.log |
| vet | `go vet ./...` | 0 | vet-02.log |
| build | `go build ./...` | 0 | build-02.log |
| gofmt | `gofmt -l internal` | empty | — |
| tracecheck full | `go run ./internal/traceability/cmd/tracecheck` | 0 | tracecheck-full-03.log |
| tracecheck sections 1.6/10.1–10.4/17.3 | same with `-section` flags | 0 | tracecheck-sections-03.log |
| catalog check | `cataloggen … -check` | 0 | catalog-check-01.log |
| fuzz round-trip 100x | `-fuzz=^FuzzCanonicalizeRoundTrip$` | 0 | fuzz-roundtrip-01.log |
| fuzz identity invariant 100x | `-fuzz=^FuzzObjectIdentityRepresentationInvariant$` | 0 | fuzz-identity-01.log |
| fuzz closed shape refusal 100x | `-fuzz=^FuzzClosedIdentityShapeRefusal$` | 0 | fuzz-refusal-01.log |

Tracecheck first failed (exit 1, tracecheck-full-01.log) on the ownership
projection digest after registering the new tests; the repin above is the
intended review binding and the rerun exits 0.

## Mutants (expected red; exit codes real)

| Mutant | Result | Log |
| --- | --- | --- |
| widen `maxNestingDepth` 256 → 512 | exit 1; pin, both refuse-past-limit tests, and regression test fail | mutant-widen-512-01.log |
| narrow 256 → 128 | exit 1; pin, both accept-at-limit and both refuse tests fail | mutant-narrow-128-01.log |
| gate deleted (`if false && …`) | exit 1 with `runtime: goroutine stack exceeds 1000000000-byte limit` / `fatal error: stack overflow` — the original crash shape reproduced through the regression test | mutant-gate-deleted-01.log |
| HEAD decoder + new test file | build failed (test references `maxNestingDepth`, undefined at HEAD); not usable as evidence, the deleted-gate mutant is the crash reproduction | head-decoder-regression-01.log |

Production canonical.go was restored from a saved copy after each mutant and
`git diff` confirmed only the intended delta remained.

## Not run

- `-race` and cross-OS builds: not part of the declared gates for this task.
