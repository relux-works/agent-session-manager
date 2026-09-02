# TASK-260902-2xbu6m — reapply-record-schema-onto-current-trunk

Three accepted leaves reapplied onto current trunk as ONE signed commit.

| Field | Value |
| --- | --- |
| Base of accepted series | `48db30b` |
| Accepted leaves | `a53dfbb`, `ae284d5`, `fc94e4a` (patch series) |
| Trunk at reapply | `10aaa16` (== `origin/main`) |
| Result commit | `af1ad4e`, signed, one commit past trunk, zero behind |
| Branch | `task-board/story/STORY-260902-2zsqwk` |

## Method

The attached patch series was replayed at its own base `48db30b` in a throwaway
worktree (`git am`, exit 0, no conflicts), producing the exact accepted tree.
The combined diff `48db30b..accepted` was then applied to trunk with
`git apply --3way`, so every non-conflicting hunk landed as a real three-way
merge rather than a reimplementation. The scratch worktree was removed after
the baselines were measured.

## Conflict resolution — nine blocks, not four

The task recorded four structural hunks. Five more appeared because trunk
advanced further after that probe. All nine:

| File | Blocks | Resolution |
| --- | ---: | --- |
| `README.md` tools row | 1 | Union merge: union of both purpose clauses, both scoped-tracecheck section sets merged into one 27-section invocation, `go test ./internal/config -cover` kept, `FuzzObservationEventRefusal` added |
| `internal/traceability/ownership.v0.5.0.json` | 2 | Union merge of both arrays (6 test owners + 2 nesting-depth owners; 4 section bindings + 9 section bindings) |
| `internal/traceability/traceability.go` | 1 | Digest RECOMPUTED, see below |
| `README.md` count sentence | 1 | Derived count recomputed: 38 |
| `internal/traceability/traceability_test.go` | 1 | Derived count recomputed: 38 |
| `internal/traceability/cmd/tracecheck/main_test.go` | 2 | Derived count recomputed: 38 |
| `LOGBOOK.md` | 1 | Add/add (file absent at base); entries interleaved by their own timestamps |

### The recomputed digest

`reviewedOwnershipCanonicalSHA256` was NOT copied from either side. The
registry was union-merged first, then `go run ./internal/traceability/cmd/tracecheck`
refused at `internal/traceability/traceability.go:286` and reported the
computed projection digest, which was then pinned:

```
ownership registry projection digest f126a4a00c64744586cc9d2bfc07e03a3923a93ee910970fe9183c9869759716
  differs from reviewed 9f7737cb07f853012fcc9e2359981e20e9b65df622f9da7fa4935a2180cd04b0
```

Pinned value: `f126a4a00c64744586cc9d2bfc07e03a3923a93ee910970fe9183c9869759716`.

### The derived counts

`acceptance_cases` length: base 29, trunk 35 (+6), accepted 32 (+3).
Union-merged registry = 38, independently confirmed by tracecheck output
(`acceptance_cases=38`). Merged registry has zero duplicate section keys,
zero duplicate acceptance-case ids, and zero duplicate test declarations
within any case.

## Deviation report

Byte-identity to the accepted tree was asserted mechanically for all 33
accepted-changed files outside the conflict set. Two deviated, both because
trunk carried its own delta in the same file:

1. `internal/canonicaljson/testdata/constraint-enumeration.md` — trunk added
   the `maxNestingDepth` row. Verified: merged file is byte-identical to
   `trunk version + accepted delta` (checked by replaying the accepted delta
   onto the trunk blob with `patch` and diffing; zero difference).
2. `task-board.config.json` — trunk rewrote the spawn runtime block
   (`exclusive` -> `mixed`, added codex entries). Verified: the accepted
   one-line delta (`FuzzObservationEventRefusal` in the gate command list)
   is present on top of trunk's version; nothing else from the accepted side
   was needed or dropped.

Neither is an absorbed change; both are trunk content preserved.

### Pre-existing trunk anomaly, reported not absorbed

`README.md:450` on trunk `10aaa16` claimed **30** executable acceptance cases
while `ownership.v0.5.0.json` on the same commit carried **35**. That prose
count is not gated by anything — `tracecheck` pins `AcceptanceCases` in
`traceability_test.go` and the two CLI strings, and nothing reads the README
sentence — so it drifted five behind across four Story landings. Base
`48db30b` and both leaf trees agree the sentence should equal
`len(acceptance_cases)`. The merged tree restores that invariant at 38, which
is a +8 correction relative to trunk rather than the +3 the accepted leaves
alone would imply. Flagged here rather than silently carried.

## Gates — all standalone, real exit codes

| Gate | Exit |
| --- | ---: |
| `go build ./...` | 0 |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l .` (empty listing) | 0 |
| `git diff --check` | 0 |
| `go test ./... -count=1` | 0 |
| `go test ./... -cover -count=1` | 0 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 |
| scoped tracecheck, 27 sections (README row) | 0, `assigned_scopes=27` |
| `cataloggen -check` | 0 |
| `FuzzCanonicalizeRoundTrip` 100x | 0 |
| `FuzzObjectIdentityRepresentationInvariant` 100x | 0 |
| `FuzzClosedIdentityShapeRefusal` 100x | 0 |
| `FuzzObservationEventRefusal` 100x | 0 |
| `FuzzScalarProductionEntries` 100x | 0 |

## Coverage — measured against both baselines, not asserted

Both baselines were re-measured in a scratch worktree at the exact trees, not
quoted from earlier runs.

| Package | Accepted tree | Trunk `10aaa16` | Merged | Delta |
| --- | ---: | ---: | ---: | --- |
| `internal/canonicaljson` | 97.2% | 87.2% | 97.2% | no regression |
| `internal/config` | n/a (absent) | 94.4% | 94.4% | no regression |
| `internal/catalog` | 97.6% | 97.6% | 97.6% | equal |
| `internal/catalog/cmd/cataloggen` | 79.3% | 79.3% | 79.3% | equal |
| `internal/cataloggen` | 83.9% | 83.9% | 83.9% | equal |
| `internal/scalar` | 90.1% | 90.1% | 90.1% | equal |
| `internal/specpin` | 85.1% | 85.1% | 85.1% | equal |
| `internal/traceability` | 85.0% | 85.0% | 85.0% | equal |
| `internal/traceability/cmd/tracecheck` | 87.5% | 87.5% | 87.5% | equal |

## Negative evidence for the merge resolution itself

The leaves' own gates were reviewed to acceptance (20/20 mutants killed) and
are reapplied unchanged. What is new here is the resolution, so the resolution
was mutated directly. Production call site: `internal/traceability.Verify`,
refusing at `traceability.go:286`, invoked by
`internal/traceability/cmd/tracecheck/main.go` and by CI.

| Mutant | Expected | Observed |
| --- | --- | --- |
| M1 — flip one hex digit of the recomputed reviewed digest | refuse | `tracecheck` exit **1** with the drift message; `go test ./internal/traceability/...` exit **1**; restored tree exit 0 |
| M2 — rename one union-merged `session_record_versions_test.go` owner to a nonexistent declaration | refuse | `tracecheck` exit **1**: `declaration "...EntriesXX" is absent from "internal/canonicaljson/session_record_versions_test.go"` — refused by name before the digest is even compared; restored tree exit 0 |

M2 is a narrowing mutant on the merged registry content, not a delete-the-gate
mutant: it proves the union-merged entries are individually resolved against
real declarations, not merely counted.

Raw: `TASK-260902-2xbu6m_merge-resolution-mutants.log` inside the gate-log
archive.

## Not done / not claimed

- Nothing was reimplemented. Every code and test delta outside the seven
  conflicted files is byte-identical to the accepted trees.
- The stale trunk README count is corrected as part of the merged derived
  value; no separate cleanup of unrelated trunk drift was attempted.
- Integration into `main` is the orchestrator's step and was not performed.
