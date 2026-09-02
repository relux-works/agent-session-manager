# BUG-260902-9uwnm7 — story_final publication run (RUN-260902-018b16)

Publication-only run. **No repository file was created, edited, or deleted.**
Its purpose is to republish the already-accepted candidate so the board derives
the Change Request kind as `story_final` — the earlier revision was typed
`task_delta` because the sibling BUG-260902-1874eo was still open at publication
time, and `worktree integrate` accepts only an accepted `story_final`.

## 1. Branch state verified, not assumed

| Check | Expected | Observed |
| --- | --- | --- |
| `git rev-parse --short HEAD` | 8a0dced | 8a0dced |
| `git rev-parse --short HEAD^` | 422786c | 422786c |
| `git status --porcelain` | empty | empty (0 lines) |
| `git diff --shortstat 422786c HEAD` | 5 files, +732, -0 | 5 files changed, 732 insertions(+) |

Leaf shape is intact: exactly one direct, single-parent commit past the
checkpoint. No commit, amend, rebase, or reset was performed.

`git status --porcelain` was re-checked after the full validation suite,
including all five fuzz targets, and was still empty. The candidate tree the
validation observed is the candidate tree being published.

## 2. Delivered scope (already reviewed, restated for the handoff record)

All 8 `utf8.ValidString` re-checks in `internal/canonicaljson/closed_shapes.go`
(lines 300, 347, 463, 1827, 1849, 1955, 2089, 2244) carry a subsumption comment
naming `decodeStrict`, following the existing convention at
`internal/localstore/paths.go:234-236`. The audit named four sites at
:258/:1506/:1525/:1760; those line numbers were stale (they match commit
7b94c9ad, five commits back) and no property separated that subset from the
other four, so resolving only four would have left four identical unexplained
survivors in the clause sweep.

Resolution is KEEP AND DOCUMENT, not delete: the subsumption is a
package-internal invariant rather than a property of these validators.
`CanonicalByteLength(value any)` already shows the package can export a
non-`[]byte` entry point, and two of the eight are pinned as
`invalidUTF8Refusal` in `refusal_guards_test.go`.

The AC's explicit-dependency requirement is machine-checked, not left as prose.
`internal/canonicaljson/utf8_subsumption_test.go` derives the guarded set and
decode sites from the AST and pins that no exported function or method may hand
an already-decoded value to a re-check, and that `json.NewDecoder`/
`json.Unmarshal` may appear only inside `decodeStrict`. Coverage is asserted as
a ratio, 7 of 7.

Production call sites guarded by the pin: `validateSessionLaunchPlan`,
`validateSessionBoardIdentity`, `validateExtensionValue`, `nullableString`,
`requireUTF8String`, `validateSortedUniqueStrings` — reached from the exported
identity entry points through `immutableObjectShapeValidators`, the
function-value dispatch table built in
`mustBuildImmutableObjectShapeValidators` and invoked as `validator(object)` in
`validateImmutableObjectShape`.

The published bound is stated rather than overclaimed: `closed_shapes.go:57-61`
and README name what the AST graph does not model — reflection, a function
value handed to another package, and a func-typed struct field.

## 3. Validation run in this session

Every one of the 18 configured `spawn.worktree_isolation.validation.commands`
was run directly as a standalone process in this worktree. Real exit codes:

| Command | Exit |
| --- | ---: |
| `gofmt -l` over `git ls-files -co --exclude-standard -- '*.go'` (empty assert) | 0 |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `go test ./... -count=1` | 0 |
| `go test ./... -race -count=1` | 0 |
| `go test ./... -cover -count=1` | 0 |
| `go test ./internal/scalar -fuzz=FuzzScalarProductionEntries -fuzztime=100x` | 0 |
| `go test ./internal/canonicaljson -fuzz=FuzzCanonicalizeRoundTrip -fuzztime=100x` | 0 |
| `go test ./internal/canonicaljson -fuzz=FuzzObjectIdentityRepresentationInvariant -fuzztime=100x` | 0 |
| `go test ./internal/canonicaljson -fuzz=FuzzClosedIdentityShapeRefusal -fuzztime=100x` | 0 |
| `go test ./internal/canonicaljson -fuzz=FuzzObservationEventRefusal -fuzztime=100x` | 0 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 |
| `cataloggen ... -check` | 0 |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 |
| `git ls-files -z '*.json' \| xargs -0 -n1 python3 -c json.load` | 0 |
| `task-board validate` | 0 |
| `git diff --check` | 0 |

`internal/canonicaljson` coverage 97.2% of statements — unchanged from the
accepted revision.

Targeted pin run, `go test ./internal/canonicaljson -run 'UTF8|Utf8|Subsum' -v`,
exit 0, all PASS, including
`TestNoExportedEntryPointHandsADecodedValueToAUTF8Recheck`,
`TestEveryUTF8RecheckIsCoveredByTheEntryPointPin`, and
`TestEveryUTF8RecheckDeclaresItsSubsumption`.

## 4. What I did NOT run, stated plainly

I did not re-run the mutant campaign. The task brief forbids re-verifying the
accepted work, and running mutants requires editing files, which this run is
forbidden to do. The negative-evidence claim for this scope therefore rests on
the round-1 and round-2 review records already on the board — mutants D2, G2,
E3, F3, H, I, I2, K, K2, plus the reviewer's own func-typed-struct-field mutant
that survives exactly as the published bound declares — and not on anything
this run reproduced. What this run establishes independently is only that the
pins are present, reachable, and green at this exact tree.

No new logbook entry was written: the findings for this scope are already in
`LOGBOOK.md` in commit 8a0dced (+27 lines), and writing one now would move the
candidate tree.

## 5. Carried forward, out of scope

`core_records.go:278` in `validateProviderIdentityRecord` carries a ninth
re-check of this class, outside this bug's `closed_shapes.go` scope. It is the
only remaining undocumented survivor of the class in the package and deserves
its own board item. Not fixed here — fixing it would violate this run's
no-file-change contract.
