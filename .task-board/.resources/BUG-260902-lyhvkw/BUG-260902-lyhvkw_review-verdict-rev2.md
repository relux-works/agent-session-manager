# BUG-260902-lyhvkw review verdict: ACCEPTED

- Change Request: CR-BUG-260902-lyhvkw-2 revision 2 (state ready)
- Reviewed head: ef0c1a5 on task-board/story/STORY-260902-3os1kh; HEAD tree 62a18b4 verified equal to the candidate tree OID; base 48db30b
- Repository delta: present (8 paths); identical tree to revision 1, which RUN-260902-3da478 accepted earlier today. This run re-verified independently rather than inheriting that verdict.
- Signature: `git verify-commit ef0c1a5` good, oparin@me.com

## Gate under review and how it is reached

`maxNestingDepth = 256` (`internal/canonicaljson/canonical.go:37`), enforced in `decodeValue` before a container opens (`canonical.go:362`), typed `ErrInvalidJSON`. `decodeValue` is only reached through `decodeStrict`, which is the first call in `Canonicalize` (`canonical.go:160`) and in `prepareObjectIdentity` (`canonical.go:195`) right after the 5,242,880-byte size gate; both `CalculateObjectIdentity` and `VerifyObjectIdentity` go through `prepareObjectIdentity`. `validateSurrogateEscapes` runs earlier on raw bytes and is a linear loop, not recursive. `json.Marshal` and `jcs.Transform` only see the already depth-bounded logical value. Other decoders in the repo (`specpin`, `cataloggen`, `traceability`, `scalar`) parse pinned repository data or single scalars, not peer documents.

## Attacks run by this reviewer (detached copy of the candidate tree, restored after each; log `mutants-02-rev2.log`)

| Mutant | Result |
| --- | --- |
| widen 256 -> 512 | pin + refuse-past at Canonicalize and both identity entries + regression fail |
| narrow 256 -> 255 | pin + accept-at-limit + refuse-past + regression fail |
| off-by-one `>=` -> `>` | refuse-past at all three entries + regression fail |
| `{` branch passes `depth` instead of `depth+1` | refuse-past at all three entries fail (objects and mixed shapes) |
| `[` branch passes `depth` instead of `depth+1` | refuse-past fail; regression input reproduces `fatal error: stack overflow` |
| delete the gate | refuse-past fail; regression input reproduces the exact original crash shape |

Builder probe: a throwaway test measured every builder (arrays, objects, mixed) at exactly the requested depth for 1, 2, 3, 255, 256, 257, and the identity wrapper at total depth 256 for all three shapes, so accept-at-256 / refuse-at-257 test what they claim. Tests drive the public production entries directly, not the helper.

Identity accept-at-limit is proven indirectly and soundly: the refusal comes from the Section 1.6 extension depth-4 gate (`closed_shapes.go:1512`) with `ErrInvalidIdentity` and never `ErrInvalidJSON`, which is reachable only after the decoder accepted the 256-deep document. The fuzz seed `nesting-depth-over-limit` is 257 brackets deep and runs as part of the normal suite.

## Gates rerun on the candidate tree (logs under `.temp/BUG-260902-lyhvkw-review/`)

- `go test ./internal/canonicaljson -count=1`: ok (`pkg-test-01.log`)
- `go test ./... -count=1 -cover`: exit 0 (`repo-test-cover-candidate-01.log`)
- `gofmt -l`, `go vet ./...`, `go build ./...`: clean (`gates-01.log`)
- tracecheck full and `-section 1.6 10.1 10.2 10.3 10.4 17.3`: ok, repinned ownership digest verifies (`gates-01.log`)
- `cataloggen ... -check`: exit 0 (`gates-01.log`)
- Coverage vs base 48db30b in a detached worktree (`repo-test-cover-base-01.log`): canonicaljson 87.1% -> 87.2%, every other package unchanged. No regression.

## Rationale and docs

Rationale for 256 is stated at the constant and restated in README, LOGBOOK, and the constraint enumeration row; the 16-level submodule fixture still passes so the headroom claim is not contradicted by any pinned closed shape. The bound precedes the JCS re-parse and sits far below encoding/json's 10,000-level scanner cap (which does not apply on the `Decoder.Token` path, as the delete-gate mutant crash confirms).

## Residual, non-blocking

- `LOGBOOK.md` is a new root file introduced by this change; acceptable as the repository had no logbook before.
- The depth check runs before the delimiter switch, so a stray closing delimiter at depth 256 would surface as a depth error; `Decoder.Token` refuses mismatched delimiters first, so this is unreachable and both paths are `ErrInvalidJSON` anyway.

- Integration note for the orchestrator: this revision is reviewed against base 48db30b, which is still `origin/main`. Local `main` is already at a351afd (STORY-260830-jeaivu), and trunk changed README.md, `ownership.v0.5.0.json`, and `traceability.go`, all paths this CR also touches. The rebased head 3e48785 from RUN-260902-4add4d repinned `reviewedOwnershipCanonicalSHA256` to the merged-registry digest 9f7737cb after tracecheck failed closed; expect the same conflict and repin at integration, and re-run tracecheck on the merged tree before landing.

Verdict: accepted via `accept_cr(BUG-260902-lyhvkw, revision=2, ...)`. Hand-off to the commit-owning mover for checkpoint/integration; no `commit_ack` supplied by this run.
