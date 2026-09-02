# BUG-260902-lyhvkw recovery run RUN-260902-e4f8b2 (root RUN-260902-4add4d)

## Why this run existed

RUN-260902-4add4d handed off after rebasing the Story branch onto the local
`main` ref (a351afd) and amending the fix commit into 3e48785. Change Request
construction then refused with `change_request_base_authority_mismatch`: the
candidate must be exactly one direct single-parent commit past the workspace
checkpoint 48db30b, and 3e48785 was three commits past it.

`task-board worktree status` shows why the rebase was wrong under the workspace
contract: the protected authority is `refs/remotes/origin/main`, advertised and
fetched at 48db30b (observed 2026-09-02T00:30:42Z); `selected_base_oid`,
`current_base_oid` and `checkpoint_oid` are all 48db30b. The local `main` ref at
a351afd is ahead of upstream and is not the authority. The tracked-background-
spawn reference states the `story_final` base "is never recomputed through the
local trunk ref", and that the base refresh is performed by task-board itself
before the final leaf's producer starts, never by the producer.

## What this run changed

- Restored the Story branch tip to the pre-rebase signed commit
  ef0c1a5cde66de8773844d8b0f4d71c8d8b11897 (`git reset --hard` on a verified
  clean tree). ef0c1a5 has parent 48db30b and tree
  62a18b40642cb022d882e37ab92dc74fe50b66fa, byte-identical to the
  `candidate_tree_oid` of the accepted Change Request revision 1.
- No source change. LOGBOOK.md was intentionally not amended so the candidate
  tree stays identical to the tree the reviewer already accepted; the finding is
  recorded on the board instead.
- The rebased head 3e48785 remains in the reflog and is saved as
  `rebased-head-3e48785-vs-a351afd.patch` under this evidence directory. Its
  only content beyond ef0c1a5 was the merge with STORY-260830-jeaivu, a
  repinned traceability digest for the merged registry, and a LOGBOOK entry
  describing the rebase.

## Fix under test (unchanged since revision 1)

- `internal/canonicaljson/canonical.go:37` declares `maxNestingDepth = 256` with rationale (deepest normative closed shape ~40 containers, extensions capped at 4, sixfold headroom, far below encoding/json's 10,000 cap and jcs's unbounded re-parse).
- `internal/canonicaljson/canonical.go:362` enforces it inside `decodeValue` before a container opens, returning typed `ErrInvalidJSON`; `Canonicalize`, `CalculateObjectIdentity`, `VerifyObjectIdentity` inherit it through `decodeStrict`.
- `internal/canonicaljson/nesting_depth_test.go` pins the literal 256, proves accept-at-256 and refuse-at-257 at all three entries in array, object and mixed shapes, and replays the 2,000,000-byte one-million-level crash input as a typed refusal.

## Gates rerun in this run on head ef0c1a5 (real exit codes, each a standalone process)

| Gate | Command | Exit | Log |
| --- | --- | ---: | --- |
| Verbose tests | `go test ./... -v -count=1` | 0 | 21-go-test-v-ef0c1a5.log (136 PASS, 0 FAIL) |
| Coverage | `go test ./... -cover -count=1` | 0 | 22-go-test-cover-ef0c1a5.log |
| Vet | `go vet ./...` | 0 | 23-go-vet-ef0c1a5.log |
| Build | `go build ./...` | 0 | 24-go-build-ef0c1a5.log |
| Format | `gofmt -l .` | 0 (no files) | 25-gofmt-ef0c1a5.log |
| Traceability | `go run ./internal/traceability/cmd/tracecheck` | 0 | 26-tracecheck-ef0c1a5.log |
| Catalog | `cataloggen ... -check` | 0 | 27-catalog-check-ef0c1a5.log |
| Authority coverage baseline | `go test ./... -cover -count=1` on `git archive 48db30b` | 0 | 12-trunk-48db30b-cover.log |
| Fuzz seeds (3) | `go test ./internal/canonicaljson -run='^$' -fuzz=^F$ -fuzztime=100x -parallel=1` | 0, 0, 0 | 30-fuzz-*-ef0c1a5.log |
| Race | `go test ./internal/canonicaljson -race -count=1` | 0 | 31-race-canonicaljson-ef0c1a5.log |

The same gates were also run earlier in this session on the rebased head
3e48785 before the restoration (logs 01-11, all exit 0, canonicaljson 87.2% vs
a351afd 87.1%). Those logs are kept for the record but the candidate is ef0c1a5.

## Coverage: authority 48db30b vs head ef0c1a5

| Package | 48db30b | ef0c1a5 |
| --- | ---: | ---: |
| internal/canonicaljson | 87.1% | 87.2% |
| all other packages | unchanged | unchanged |

No regression.

## Mutants of the bound on ef0c1a5 (expected red; `-run NestingDepth`, source restored after each, tree clean)

| Mutant | Exit | Failing tests |
| --- | ---: | --- |
| widen 256 -> 512 | 1 | Pinned, CanonicalizeRefusesPast, IdentityEntriesRefusePast, Regression2MB |
| widen 256 -> 257 | 1 | Pinned, CanonicalizeRefusesPast, IdentityEntriesRefusePast, Regression2MB |
| narrow 256 -> 255 | 1 | Pinned, CanonicalizeAcceptsAt, IdentityEntriesAcceptAt, CanonicalizeRefusesPast, IdentityEntriesRefusePast, Regression2MB |
| delete the `depth >= maxNestingDepth` gate | 1 | CanonicalizeRefusesPast, IdentityEntriesRefusePast, then the 2 MB regression test reproduces `fatal error: stack overflow` |

Widening reddens the suite through the refusal tests themselves, not only through the literal pin.

## Finding for the orchestrator

A producer must never rebase the managed Story branch onto the local trunk ref.
The base refresh belongs to task-board provisioning and is measured against the
protected upstream authority. When origin/main actually advances to a351afd,
the tool's refresh will replay ef0c1a5 onto it; the traceability digest pin at
`internal/traceability/traceability.go:42` will conflict (both Stories repinned
it), which the refresh reports as a same-branch rework rather than failing the
spawn.

## Notes

- One earlier shell call ran mutants from the wrong directory and produced no usable logs; it was rerun with absolute paths. Scratch copies were deleted.
- Working tree verified clean after every mutant restore; HEAD = ef0c1a5.
