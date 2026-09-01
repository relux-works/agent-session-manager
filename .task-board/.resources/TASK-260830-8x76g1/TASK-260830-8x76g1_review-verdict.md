# TASK-260830-8x76g1 — CR rev 12 review verdict

- Reviewer run: `RUN-260901-34271e` (claude / opus, independent of the codex producer)
- Change Request: `CR-TASK-260830-8x76g1-12`, revision 12, `repository_delta=present`
- Base OID: `ad7275181ca82fc3fa29544e3893923a92d7b9d5`
- Candidate tree OID: `af4f5e32efde7240c55e2355cd75ac9911ecef16`
- **Verdict: CHANGES REQUESTED -> `to-dev`.** Rework is TEST-ONLY. Production
  code in `internal/canonicaljson/closed_shapes.go` is correct on every finding
  below and must not be edited. The pinned specification must not be edited.

## Binding and integrity

The worktree was hashed into a private index against the base OID before review
and again after every mutant and probe was reverted. Both times it produced
`af4f5e32efde7240c55e2355cd75ac9911ecef16`, byte-identical to the candidate
tree. The real index and `HEAD` (`ad72751`) were never touched. The already
accepted leaves are unchanged: `internal/catalog`, `internal/cataloggen` and
`internal/specpin` are `diff -rq`-identical to the attached checkpoint tarball,
and every one of the seven `internal/scalar` files carried in that tarball is
`cmp`-identical (this leaf only adds its own `git.go`, `git_test.go`,
`fuzz_test.go` and `testdata/`).

The rev11 -> rev12 delta is exactly `internal/canonicaljson/boundary_constraints_test.go`,
+207/-2. No production file and no artefact moved.

## Green gates re-run by the reviewer at the candidate tree

Every configured `spawn.worktree_isolation.validation` command was re-run in the
foreground by this reviewer, not accepted from attached evidence.

| Gate | Result |
| --- | --- |
| `gofmt -l` over tracked+untracked Go files | empty, exit 0 |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./... -count=1` | all 8 packages ok |
| `go test ./... -cover -count=1` | canonicaljson 83.4%, scalar 90.1%, traceability 85.0% |
| `FuzzScalarProductionEntries -fuzztime=100x` | PASS, 37 seeds |
| `FuzzCanonicalizeRoundTrip -fuzztime=100x` | PASS, 26 seeds |
| `FuzzObjectIdentityRepresentationInvariant -fuzztime=100x` | PASS, 28 seeds |
| `FuzzClosedIdentityShapeRefusal -fuzztime=100x` | PASS, 73 seeds |
| `tracecheck` | ok, `assigned_scopes=0` |
| `tracecheck -section 17.3` | ok, `assigned_scopes=1` |
| `cataloggen -check` | exit 0 |
| `GOOS=linux` / `GOOS=windows` builds | exit 0 |
| `git diff --check` | exit 0 |

## Correctly delivered this cycle

The 15 bounds named by the rev11 review are genuinely closed. Every one of them
is now killed by an independent widening mutant driven through
`CalculateObjectIdentity` and `VerifyObjectIdentity`: `BlobChunk.size` 4194304,
`GitIndex.version` lower edge, `GitIndexEntry.mode` uint32, Board Identity
`logical_id` 128 (combined length+pattern), `managed_tree`
`agent_project_config_paths` 256, `GitSubmodule` `repository_identity` 256 and
its `agent_project_config_paths` 256, and the eight lower bounds
(`task_element_id`, `goal_id`, argv element, `tree_identity`, git
`repository_identity`, `GitRemote.name`, submodule `repository_identity`,
`WorkspaceSnapshot.members`). The four "redundantly enforced" survivors the
producer classified as SUBSUMED were re-derived independently and the
classification holds in each case:

- `chunks == 0` when `totalSize > 0` -> `> 1`: the final `covered != totalSize`
  invariant refuses it anyway.
- dropping the `size == 0` chunk guard: non-final chunks must be exactly
  `maxChunkSize` and coverage must equal `totalSize`, so a zero-size chunk can
  never reach acceptance.
- extension key `len(key) < 3` -> `< 2`: `reverseDNSPattern` already requires
  at least three characters.
- `logical_id` length bound and both `submodules` array caps: subsumed by
  `boardLogicalIDPattern` and by the `*total > 256` traversal cap respectively.

## Finding 1 — the declared chunk-count bound is not pinned by any test

`internal/canonicaljson/closed_shapes.go:23` declares `maxBlobChunks = 32_768`.
`testdata/constraint-enumeration.md:27` asserts that `Blob Descriptor.chunks` is
"Enforced exactly as declared" against the pinned SPEC text
"BlobChunk[0..32768]".

Mutating the **constant declaration** from `32_768` to `32_769` leaves the whole
repository suite green:

```
SURVIVED     | const maxBlobChunks 32_768 -> +1                     | suite still ok
```

The reason is that the boundary case at
`boundary_constraints_test.go:137-138` is written as
`blobDescriptorWithChunks(maxBlobChunks)` / `blobDescriptorWithChunks(maxBlobChunks + 1)`.
It references the production symbol, so both the gate and its "boundary" move
together and the test can never fail for a wrong declared value. It pins the
relationship between the call site and the constant, not the value 32768. The
rev11 harness only ever mutated the call site
(`"chunks", maxBlobChunks` -> `maxBlobChunks+1`, batch1.json line 12), which is
why the replay reported KILLED and the constant itself was never exercised —
a real gap in the sweep as re-run, not a misreport.

The sibling constant shows the shape that is required: `maxChunkSize` is
independently pinned by the literal `4194304` in `canonical_test.go:693-694`,
and its constant mutant is correctly KILLED.

The bound is publicly reachable, so it does **not** qualify for the
`GitIndex.entries` unreachability disclosure. Measured through
`CalculateObjectIdentity` against the 5,242,880-byte cap:

| chunks | encoded input bytes | under cap | production result |
| ---: | ---: | --- | --- |
| 32768 | 4,484,676 | yes | accepted |
| 32769 | 4,484,814 | yes | refused: `member chunks exceeds maximum length 32768` |
| 32770 | 4,484,952 | yes | refused |

Production is correct. Required rework: add an accept-at-32768 /
refuse-at-32769 case that uses an independent literal rather than
`maxBlobChunks`, and attach the constant-declaration mutant as expected-red
evidence.

## Finding 2 — three of the four symlink-escape refusal clauses are unpinned

`symlinkTargetEscapes` (`closed_shapes.go:844-849`) refuses a materialization-root
escape on four independent grounds: an embedded NUL, a `/`-absolute target, a
target containing a backslash, and a Windows drive-relative `X:` prefix. The
only symlink-escape negative case anywhere in the repository is `"../escape"`
(`canonical_test.go:1119`, `fuzz_test.go:256`). Only the backslash clause is
additionally pinned.

Individually mutated, with the whole repository suite still printing ok:

```
SURVIVED     | drop absolute-target escape clause                   | suite still ok
SURVIVED     | drop NUL-target escape clause                        | suite still ok
SURVIVED     | line846 >= 2 -> >=3                                  | suite still ok
KILLED       | drop backslash-target escape clause                  | (correctly covered)
```

This is not a hypothetical. Applying all three vet-clean mutants at once and
driving the real public entry point, `CalculateObjectIdentity` **attests** a
Transfer Manifest whose symlink target escapes the root, and `go test ./...`
exits 0:

```
### MUTANT: NUL clause -> rune(1); absolute '/' -> U+0001 prefix; drive guard >=2 -> >=3
    target="/etc/passwd"  calculate_err=<nil>
    target="no\x00ol"     calculate_err=<nil>
    target="C:"           calculate_err=<nil>
--- PASS
### SAME MUTANT, WHOLE REPOSITORY SUITE
rc=0   (all 8 packages ok)
```

Unmutated production refuses all three with
`symlink ManifestEntry[0] target escapes the materialization root`, so the
product is correct and this is squarely the "gate admits what it must reject
while the suite stays green" shape. Required rework: negative cases through both
public identity entries for an absolute target, a NUL-containing target, and a
bare two-character drive target `C:` (the last one specifically pins the
`len(target) >= 2` guard).

## Finding 3 — two strict-sortedness guards have no two-element case

```
SURVIVED     | line762 > 0 -> >1   (Transfer Manifest entry path sortedness)
SURVIVED     | line882 > 0 -> >1   (WorkspaceSnapshot member workspace-ID sortedness)
```

Widening the loop guard from `index > 0` to `index > 1` skips the first
comparison, so a two-element out-of-order array is accepted and nothing in the
suite notices. The equivalent guards at lines 1034, 1172, 1599, 1851 and 1866
are all KILLED, so this is an inconsistency in the negative corpus rather than a
systemic hole. Required rework: a two-entry unsorted Transfer Manifest and a
two-member unsorted WorkspaceSnapshot, refused through both public entries.

## Not findings

- `closed_shapes.go:195` matches a `{0,63}` sweep pattern but is the text of an
  error message, not a bound.
- `requireUint(chunk, "index", 1<<32-1)` narrowing survives because
  `maxBlobChunks` is strictly stricter; rev11 already recorded this correctly.
- `README.md` needs no change. It claims the member inventory is checked against
  `requireExactMembers`, which reproduces, and makes no per-bound coverage claim.

## Scope instruction for the next producer

Test-only. Do not touch `closed_shapes.go`, any other production file, or the
pinned specification. Close findings 1-3, then re-run the **whole** bound sweep —
including constant declarations, not only call sites — and require zero
actionable survivors before handing back. The reviewer harness that produced the
table above is attached as
`TASK-260830-8x76g1_rev12-reviewer-sweep.tar.gz` (`genmutants.py` derives the
mutant set mechanically from the production file, so it re-derives rather than
replays a fixed list).
