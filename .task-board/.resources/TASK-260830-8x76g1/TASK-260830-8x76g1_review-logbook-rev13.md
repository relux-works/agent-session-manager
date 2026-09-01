# TASK-260830-8x76g1 — CR rev13 reviewer logbook

Reviewer run `RUN-260901-5ad8f6` (claude / opus). Candidate tree
`7769a2c43ec8c6f224e3fbf6eeb1ca402747302d`, base `ad72751`. Verdict:
changes requested -> `to-dev`.

## Finding

`closed_shapes.go:614` (`if size == 0` on `BlobChunk`) survives mutation and the
survivor is actionable. A **trailing** zero-size chunk contributes nothing to
`covered`, so neither the non-final `size == maxChunkSize` rule at `:617` nor
the `covered != totalSize` invariant at `:631` fires. With that one clause
disabled, both `CalculateObjectIdentity` and `VerifyObjectIdentity` attest a
Blob Descriptor with `BlobChunk[1].size == 0` while the whole
`internal/canonicaljson` suite stays green.

The rev12 verdict classified this survivor as subsumed with an argument that is
only valid for non-final chunks, and rev13 carried the classification forward
verbatim as "BlobChunk positional/coverage subsumption". Two consecutive cycles
inherited the same wrong reasoning because neither drove the trailing-chunk
shape at the public entries.

## Anomaly: a green boundary case that never reaches its clause

`boundary_constraints_test.go:236` `"blob chunk size non-zero"` looks like the
lower-bound case for `BlobChunk.size`. Its helper
`blobDescriptorWithChunkSize(0)` sets `Blob Descriptor.size` **and**
`BlobChunk[0].size` to `0`, so production refuses at `:579-580`
("empty Blob Descriptor must contain no chunks") and `:614` is never reached.
The case is green for a clause it does not exercise. This is the reason the
mutant survived despite an apparently matching test name — a reminder that a
named boundary case proves nothing until the refusal *reason* is checked.

## Correction to the rev12 verdict, recorded for the next cycle

The rev12 verdict states the eight declared lower bounds are "killed by an
independent widening mutant". Re-derived here: five of them
(`goal_id`, `tree_identity`, both `repository_identity`, `GitRemote.name`) are
**subsumed**, not pinned — `requireString` (`:1748`) refuses `""` before
`requireBoundedString` consults its character-count minimum, so `min 1 -> 0`
cannot widen behavior at all. Same non-actionable conclusion, different and
verifiable mechanism. Only `task_element_id` (which goes through
`requirePrintableByteBoundedString`) is genuinely pinned by a test.

## Confirmed non-actionable by mechanism, not by inheritance

- `logical_id` 128: the `{0,127}` pattern mutant and the `1..128` length mutant
  are mutually redundant; the **combined** mutant admits 129 characters and is
  killed by `board_logical_identifier_128_characters`.
- `submodules 256` (both sites): subsumed by the `*total > 256` traversal cap,
  which is itself pinned by `TestTransferManifestSubmoduleTotalCountBoundary`.
- `totalSize > 0` chunk-presence guard: subsumed by the exact-coverage check.
- extension key `len < 3`: subsumed by `reverseDNSPattern`.

## Reproduction

The rev12 `genmutants.py` re-run over the current `closed_shapes.go` regenerates
71 mutants byte-identical to the producer's set; with the four symlink clause
mutants that is 75, split 59 KILLED / 16 SURVIVED, matching the producer's
counts exactly. Evidence:
`TASK-260830-8x76g1_review-evidence-rev13.tar.gz`.
