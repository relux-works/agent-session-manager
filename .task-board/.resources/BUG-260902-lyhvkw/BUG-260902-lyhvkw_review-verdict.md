# BUG-260902-lyhvkw review verdict: ACCEPTED

- Change Request: CR-BUG-260902-lyhvkw-1 revision 1
- Reviewed head: ef0c1a5 on task-board/story/STORY-260902-3os1kh (tree 62a18b4, verified equal to the candidate tree OID); base 48db30b
- Reviewer run: RUN-260902-3da478
- Repository delta: present (8 paths)

## What was attacked, not read

Gate under review: `maxNestingDepth = 256` enforced in `decodeValue` (`internal/canonicaljson/canonical.go:362`), reached through `decodeStrict`, which is the first call in `Canonicalize` and in `prepareObjectIdentity` (shared by `CalculateObjectIdentity` and `VerifyObjectIdentity`). I confirmed no other JSON decoder in the package can be reached with peer bytes: `json.Marshal` + `jcs.Transform` only ever see the already bounded logical value. Decoders elsewhere (`specpin`, `cataloggen`, `traceability`, `scalar.decodeJSONString`) parse pinned repo data or single scalars with `Decoder.Decode`, not peer documents.

Mutants I ran myself (each restored, tree clean afterwards; log `.temp/BUG-260902-lyhvkw-review/mutants-01.log`):

| Mutant | Result |
| --- | --- |
| widen 256 -> 512 | 4 tests fail (pin, refuse-past at Canonicalize and both identity entries, regression) |
| widen 256 -> 257 (off-by-one) | 4 tests fail |
| narrow 256 -> 128 | 6 tests fail (accept-at-limit tests also fail) |
| delete gate (`&& false`) | refuse tests fail, then `runtime: goroutine stack exceeds 1000000000-byte limit` / `fatal error: stack overflow` from the 2 MB regression input, exact original crash shape |

The suite pins the literal 256 in the test file rather than reading the constant, so a narrowing mutant reddens the suite as well as a widening one. Accept-at-limit and refuse-one-past are proven in array, object, and mixed container shapes at all three public entries. The identity at-limit proof is indirect but sound: the refusal comes from the later Section 1.6 extension depth-4 gate with `ErrInvalidIdentity`, which is only reachable after the decoder accepted the 256-deep document. The fuzz seed at 257 brackets is over the limit by exactly one (the `[]byte(` wrapper accounts for the extra bracket pair in the corpus file).

## Gates rerun by me on the candidate tree

- `go test ./... -count=1 -cover`: exit 0, log `repo-test-cover-01.log`
- `go build ./...`, `go vet ./...`, `gofmt -l`: clean
- `go run ./internal/traceability/cmd/tracecheck` (full, and `-section 1.6 -section 10.1`): ok, the repinned ownership digest verifies
- `cataloggen ... -check`: exit 0
- Coverage vs base 48db30b (measured in a `git archive` tree): canonicaljson 87.1% -> 87.2%, every other package unchanged. No regression.
- `git verify-commit HEAD`: good signature for oparin@me.com

## Rationale check

256 vs the deepest closed shape: the suite's 16-level submodule fixture (`canonical_test.go:1205`) still passes, so the headroom claim is not contradicted by any pinned closed shape. The bound sits below encoding/json's 10,000 scanner cap and precedes the JCS re-parse, so the typed refusal is always the first gate.

## Residual, non-blocking

- The depth check runs before the delimiter switch, so a hypothetical stray closing delimiter at depth 256 would be reported as a depth error rather than an unexpected delimiter. Both are `ErrInvalidJSON`, and `Decoder.Token` refuses mismatched delimiters first, so this is unreachable.

Verdict: accepted. Hand-off to the commit-owning mover for checkpoint/integration; no `commit_ack` supplied by this run.
