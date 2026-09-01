# TASK-260830-8x76g1 — CR rev 13 review verdict

- Reviewer run: `RUN-260901-5ad8f6` (claude / opus, independent of the codex producer)
- Change Request: `CR-TASK-260830-8x76g1-13`, revision 13, `repository_delta=present`
- Base OID: `ad7275181ca82fc3fa29544e3893923a92d7b9d5`
- Candidate tree OID: `7769a2c43ec8c6f224e3fbf6eeb1ca402747302d`
- **Verdict: CHANGES REQUESTED -> `to-dev`.** Rework is TEST-ONLY (plus one
  artifact row). Production code in `internal/canonicaljson/closed_shapes.go`
  is correct on the finding below and must not be edited. The pinned
  specification must not be edited.

## Binding and integrity

The worktree was hashed into a private index before review and again after every
mutant and probe was reverted; both times it produced
`7769a2c43ec8c6f224e3fbf6eeb1ca402747302d`, byte-identical to the candidate
tree. The real index and `HEAD` (`ad72751`) were never touched, and
`closed_shapes.go` matches its pre-sweep SHA-256
`3d34cb52cb483e513a8ba398ed351993a02c46c01d5f4086af083f038304b307`.

The accepted leaves are unchanged: `internal/catalog`, `internal/cataloggen`
and `internal/specpin` are `diff -rq`-identical to the attached checkpoint
tarball, and all seven `internal/scalar` files carried in that tarball are
`cmp`-identical.

The rev12 -> rev13 delta is exactly
`internal/canonicaljson/boundary_constraints_test.go`, +51/-2. No production
file, no artifact, and no configuration moved. That matches the rev12 scope
instruction.

## Green gates re-run by the reviewer at the candidate tree

Every configured `spawn.worktree_isolation.validation` command was re-run in the
foreground by this reviewer, not accepted from attached evidence.

| Gate | Result |
| --- | --- |
| `gofmt -l` over tracked+untracked Go files | empty, exit 0 |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./... -count=1` | all 8 packages ok |
| `go test ./... -race -count=1` | all 8 packages ok |
| `go test ./... -cover -count=1` | canonicaljson 83.6%, scalar 90.1%, traceability 85.0% |
| `FuzzScalarProductionEntries -fuzztime=100x -parallel=1` | PASS, 37 seeds |
| `FuzzCanonicalizeRoundTrip -fuzztime=100x -parallel=1` | PASS, 27 seeds |
| `FuzzObjectIdentityRepresentationInvariant -fuzztime=100x -parallel=1` | PASS, 29 seeds |
| `FuzzClosedIdentityShapeRefusal -fuzztime=100x -parallel=1` | PASS, 73 seeds |
| `tracecheck` | ok, `assigned_scopes=0` |
| `tracecheck -section 17.3` | ok, `assigned_scopes=1` |
| `cataloggen -check` | exit 0 |
| `GOOS=linux` / `GOOS=windows` builds | exit 0 |
| tracked JSON parse gate | exit 0 |
| `git diff --check` | exit 0 |

The candidate defines exactly four `Fuzz*` targets (`FuzzScalarProductionEntries`,
`FuzzCanonicalizeRoundTrip`, `FuzzObjectIdentityRepresentationInvariant`,
`FuzzClosedIdentityShapeRefusal`); all four are wired into the configured
validation list with a fixed `-fuzztime=100x`. The two `Fuzz`-prefixed
declarations in `internal/traceability/traceability_test.go:225-226` are string
literals inside an `fstest.MapFS` fixture, not real targets. No gate gap here.

## Correctly delivered this cycle

The three rev12 findings are genuinely closed. Re-deriving the mutant set from
the current production file with the attached rev12 `genmutants.py` reproduces
71 mutants byte-identical to the producer's `rev13-mutants-all.json`; adding the
four attached symlink clause mutants gives the same 75, and my independent
sweep reproduces the same 59 KILLED / 16 SURVIVED split. All six mutants the
rev12 verdict named now die:

```
KILLED | const maxBlobChunks 32_768 -> +1        | blob_chunk_count_32768
KILLED | drop absolute-target escape clause      | symlink_target_rejects_absolute_slash
KILLED | drop NUL-target escape clause           | symlink_target_rejects_NUL
KILLED | line846 >= 2 -> >=3                     | symlink_target_rejects_Windows_drive_roots
KILLED | line762 > 0 -> >1                       | manifest_entries_require_strict_path_order...
KILLED | line882 > 0 -> >1                       | workspace_snapshot_members_require_strict...
KILLED | drop NUL+absolute+backslash+drive       | (combined clause mutant)
KILLED | drop backslash-target escape clause     | (already covered in rev12)
```

I also independently re-derived, rather than accepted, the non-actionable
classification of 15 of the 16 survivors, and proved each one at the public
entries:

- The five `requireBoundedString(..., 1, N)` `min 1 -> 0` survivors (`goal_id`,
  managed-tree `tree_identity`, git `repository_identity`, `GitRemote.name`,
  submodule `repository_identity`) cannot widen behavior:
  `requireString` (`closed_shapes.go:1748`) refuses `""` with "must be a
  non-empty UTF-8 string" before the character-count bound is consulted. Probed
  through both entries. (Note for the record: the rev12 verdict's claim that
  these lower bounds were "killed by an independent widening mutant" does not
  reproduce — they are subsumed, not pinned. Same conclusion, different reason.)
- `logical_id`: the `{0,127}` pattern mutant and the `1..128` length mutant
  survive individually only by mutual redundancy. Applying **both** at once
  admits a 129-character `logical_id` through `CalculateObjectIdentity` and
  `VerifyObjectIdentity`, and
  `TestDeclaredBoundaryConstraintsReachBothIdentityEntries/board_logical_identifier_128_characters`
  correctly fails. The bound is jointly pinned.
- `line582 totalSize > 0 -> > 1`: `size=1, chunks=[]` is still refused, by the
  `covered != totalSize` invariant at `:631`.
- `line1479 len(key) < 3 -> < 2`: `"a."` is still refused by `reverseDNSPattern`.
- `line996`/`line1346` `submodules 256 -> 257`: subsumed by the `*total > 256`
  traversal cap at `:1245`, whose own `> 256 -> > 257` mutant is KILLED by
  `TestTransferManifestSubmoduleTotalCountBoundary`.
- `line195 {0,63} -> {0,64}` is error-message text, and `line596 <32 -> <31` is
  a narrowing artifact of the `1<<32-1` literal. `line596 uint32 max -> 1<<32`
  is subsumed by `maxBlobChunks` plus the `chunkIndex == index` rule.

## Finding 1 — the BlobChunk lower size bound is not pinned, and its "boundary" case passes for the wrong reason

`closed_shapes.go:614-616` refuses `BlobChunk.size == 0`. The
`internal/canonicaljson` suite stays fully green with that clause disabled:

```
SURVIVED     | line614 drop chunk zero-size guard: if size == 0 {   | suite still ok
```

That survivor is **actionable**, not subsumed. Driving the real public entries
under the single vet-clean mutant
`if size == 0 {` -> `if false && size == 0 {`:

```
### MUTANT: line 614 zero-size chunk guard disabled
    Blob Descriptor size=4194304
    chunks=[{index:0, offset:0,       size:4194304},
            {index:1, offset:4194304, size:0}]
    calculate_err=<nil>  field="descriptor_id"
    verify_err=<nil>
### SAME MUTANT, WHOLE internal/canonicaljson SUITE
rc=0   (ok)
```

`CalculateObjectIdentity` and `VerifyObjectIdentity` **attest** a Blob
Descriptor whose final `BlobChunk.size` is `0`, which the pinned declaration
`uint53[1..4194304]` forbids, and nothing in the repository notices. Unmutated
production refuses it correctly with
`BlobChunk[1] size must lie in [1, 4194304]`, so the product is right and this
is exactly the "gate admits what it must reject while the suite stays green"
shape.

The rev12 verdict's subsumption argument for this clause — "non-final chunks
must be exactly `maxChunkSize` and coverage must equal `totalSize`, so a
zero-size chunk can never reach acceptance" — holds only for a **non-final**
chunk. A trailing zero-size chunk adds nothing to `covered`, so both `:617` and
`:631` pass. `TASK-260830-8x76g1_rev13-results.md` and
`rev13-mutation-summary.md` carry that argument forward as
"BlobChunk positional/coverage subsumption" and conclude "0 survivors are
actionable"; that conclusion does not reproduce.

The existing case that appears to cover this is
`boundary_constraints_test.go:236-239`, `"blob chunk size non-zero"`. It is
built by `blobDescriptorWithChunkSize(0)` (`:468-473`), which sets **both**
`Blob Descriptor.size` and `BlobChunk[0].size` to `0`. Production therefore
refuses it at `:579-580`:

```
blobDescriptorWithChunkSize(1) -> accepted, field "descriptor_id"
blobDescriptorWithChunkSize(0) -> "empty Blob Descriptor must contain no chunks"
```

The refusal never reaches `:614`. The case is green for a clause it does not
exercise.

`testdata/constraint-enumeration.md:36` states that `BlobChunk.size` is
"Enforced exactly as declared before identity calculation or verification"
against the pinned text "uint53[1..4194304]; every non-final chunk is exactly
4194304". The `[1..` half of that row is enforced in production but pinned by
nothing.

### Required rework (test-only, plus one artifact row)

1. Add a refuse case through both public identity entries for a **trailing**
   zero-size `BlobChunk` in a Blob Descriptor with `size > 0` — e.g. the
   two-chunk shape above — paired with an accept case whose final chunk size is
   `1`. It must fail when `closed_shapes.go:614` is disabled.
2. Either keep `"blob chunk size non-zero"` and rename it to what it actually
   pins (the empty-descriptor/chunks cross-field rule), or fix
   `blobDescriptorWithChunkSize` so a `0` case reaches the per-chunk bound.
   Do not leave a case whose name claims a clause it never reaches.
3. Re-check `constraint-enumeration.md:36` against the standing rule that a row
   claims only what the suite proves. Once step 1 lands the existing wording is
   accurate; if step 1 is not taken, the row must say plainly that production
   enforces the lower bound while no test pins it.
4. Re-run the whole derived sweep (`genmutants.py` over the current
   `closed_shapes.go`, plus the four symlink clause mutants) and require the
   `line614` mutant to be KILLED, with the sweep output attached.

## Not findings

- `README.md` needs no change. It makes no per-bound or completed-audit claim,
  correctly reports migration publication / atomic-reference advancement /
  `ax migrate` / `ax doctor` as unavailable, and its "linear Unicode simple-fold
  set" claim reproduces through `TestTransferManifestMaximumEntryGateIsLinear`
  (65,536 entries under the 5,242,880-byte cap, `< 2s`, `!race` build tag).
- `TestConstraintEnumerationMatchesRequireExactMembers` genuinely derives the
  member set from the production `requireExactMembers` argument lists via
  `go/ast` and fails on an undocumented call, so artifact and code cannot drift.
- The `GitIndex.entries` unreachability disclosure
  (`constraint-enumeration.md:11-14`) is intact and its validator-level boundary
  is pinned by `TestGitIndexEntryCountBoundaryIsBoundBelowThePublicObjectSizeGate`,
  which kills the `line1158 Array entries 65_536->65_537` mutant.

## Scope instruction for the next producer

Test-only, plus the one artifact row in item 3. Do not touch
`closed_shapes.go`, any other production file, any accepted leaf, or the pinned
specification. Close Finding 1, then re-run the full derived sweep and require
zero actionable survivors — and for each survivor you classify as
non-actionable, state the *mechanism* that subsumes it and prove it at the
public entries, rather than restating a prior cycle's classification.
