# TASK-260830-2ya5le Results — implement-fidelity-report-and-reason-registry (rev3 rework)

Status: ready for review. Candidate left UNCOMMITTED in the Story
worktree (`task-board/story/STORY-260830-21bxa3`, tip `0ca3e4c`);
no commit on the Story branch. This is the FIRST leaf of
STORY-260830-21bxa3; the Change Request is `task_delta` (no
registry edit — the Story's final leaf carries it).

Rev3 answers review verdict rev2 (`unmeasured-build-utf8-sites`,
CHANGES REQUESTED) per `TASK-260830-2ya5le_rework-rev3.md`: close
the CLASS, not the three sites. The invariant is that every
string-valued member of every shape this leaf builds or decodes
refuses invalid UTF-8 at every entry with `ErrInvalid` and a
pinned literal detail. The `byte_counts` stated bound is kept
exactly as the orchestrator accepted it; the reviewer held every
other row, and nothing unrelated was rewritten.

Rev3 delta vs the rev2 candidate tree `7bade702`, measured by
`cmp` per file (`rev2-tree-compare.log` in the evidence tar): all
10 production files byte-identical, all 9 existing test files
byte-identical; only `testdata/mutant_harness.py` differs (8 new
rows + the trailing-newline fix) and
`internal/clonefidelity/utf8_reflection_test.go` is new, plus
TRACEABILITY wording. Zero production bytes changed, as the
rework brief requires.

Authority: the task record's v0.5.0 §13.14.2/§13.14.4 is STALE per
the brief. All behavior is pinned to `internal/specdoc/SPEC.v0.7.0.md`
§13.14.2 "Fidelity, projection, and lineage" (line 10548+), quoted
in code comments and tests.

## What was built (rev3)

- `TestUTF8RefusalReflectionGrid`
  (`internal/clonefidelity/utf8_reflection_test.go`): derives the
  33 string-carrying members from `FidelityReportInput` /
  `DispositionRecordInput` by reflection (asserted non-empty; a new
  string field fails the test until probed; a new unhandled kind
  fails loudly in the walker) and drives member × 5
  invalid-UTF-8 classes (lone continuation `80`, `ff fe`,
  overlong `c0 af`, encoded surrogate `ed a0 80`, truncated
  sequence `e2 82`) × both entries = 330 cells. Build cells pin
  member-specific literals; decode cells pin the frame literal
  `fidelity report not valid UTF-8 (frame)`, because the landed
  `environ.DecodeStrictObject` gate trips before any member
  dispatch — member-level decode attribution would re-implement
  the landed frame parser, which the composition rule forbids.
  The per-member plants prove no member's bytes bypass the frame
  gate. Stated plainly in the matrix (§A row 33, §D).
- `TestRegistryPredicatesRefuseInvalidUTF8`: the 5 remaining
  string-taking public entries × 5 classes return false.
- `TestBuildUTF8ExtensionsNestedDepth`: the delegated extension
  UTF-8 gate fires below the top level (nested map, slice) at
  both layers. `TestBuildUTF8` is kept green as committed.
- 8 new narrowing rows `N-build-utf8-*`: the three reviewer sites
  (`source_class`, `target_locator`, `forbid_reasons`) plus one
  per shape and the remaining explicit pre-marshal sites
  (`source_item_key`, `explanation`, the reason-codes arm, both
  extension call sites). Each admits exactly the `ff fe` token
  and is KILLED by the grid run alone. The reason-codes kill
  arrives via the downstream vocabulary literal (documented in
  the row note). The decode side carries no plant: invalid bytes
  trip the shared strict frame first, which `N-frame-report` /
  `N-frame-row` already pin.
- `mutant_harness.py` trailing blank line removed
  (`git diff --check` clean; untracked files additionally
  scanned).

Explicit bounds (named, not waived): tuple-sub-member UTF-8
enumeration belongs to `sessadapter` (this leaf pins per-tuple
refusal × 5 classes at both tuple members and both entries);
decode-side attribution is frame-level by landed-owner design
(driven per member, not waived). Both are in the matrix §D.

## AC coverage: 39 of 40 rows driven + 1 stated bound (measured)

`TASK-260830-2ya5le_conformance-matrix.md` (attached, §E now
complete — the rev2 attachment was truncated mid-table): 40 AC
rows, 39 driven through a production entry by a named committed
test; every gate row carries a directly-reddening narrowing (§A).
The rev2 reviewer measured 38 fully driven + row 33 partial (4 of
7 sites) + 1 bound; rev3 drives row 33 over the full 33-member
derived class, so the measured ratio is back to 39 of 40 + 1
stated bound (row 24, byte-aggregate reconciliation; owner: spec
clarification (agent-session-manager-spec) + the §13.14.1
capture-manifest leaf). Rows 34–36 are admitted properties /
cross-checks with no gate to narrow (disclosed, not waived).
Nothing here claims `byte_counts` is row-reconciled.

## Mutants

`internal/clonefidelity/testdata/mutant_harness.py`: 70 narrowing
rows + harmless `C-doc-comment` SURVIVED control. Battery executed
TWICE on the final tree, exit 0, 71/71 `ok:` lines each run (70
KILLED + control SURVIVED), one raw log per plant per run
(`mutants/run1/`, `mutants/run2/` in the archive, 71 logs each).
`PYTHONDONTWRITEBYTECODE=1`, zero `__pycache__`/`.pyc` (verified:
no such files under the package after the runs).

## Validation rerun (rev3 run, final tree, all exit 0)

All 30 configured commands from `task-board.config.json`
(`spawn.worktree_isolation.validation.commands`), exits observed
firsthand (logs `cmds/cmd*.log`; race as `race/race-G1…G4.log` +
`race/race-sessquery.log`; every log carries its command line +
output + exit):

- 1 gofmt check: 0. 2 build: 0. 3 vet: 0.
- 4 `go test ./... -count=1 -v`: 0 — 45 packages `ok`, 2865
  top-level PASS (rev1 2862 + 3 new tests), zero FAIL.
- 5 race gate: 0 in 5 bounded groups with `-timeout 25m`
  (44/44 `ok` in G1–G4 + sessquery `ok` 532.4s; zero `DATA
  RACE`). Grouping is process-identical to `./...` (one process
  per package either way). One rerun was needed: the first G1–G4
  attempt passed bare `internal/...` paths (setup failure, my
  invocation error, no product signal); the corrected `./…`
  rerun is the logged evidence.
- 6 cover gate: 0 — 45 `ok`; `clonefidelity` 97.5%,
  `clonebundle` 86.7%, `environ` 92.4% of statements.
- 7–23 fuzz gates (rpcwire ×4, scalar ×1, canonicaljson ×4,
  secconftest ×8, each `-fuzztime=100x -parallel=1`): all 0.
- 24 tracecheck: 0 (`clauses_discharged=70/574` — registry
  untouched). 25 cataloggen `-check`: 0.
- 26 linux build: 0. 27 windows build: 0. 28 JSON sweep: 0.
- 29 `task-board validate`: 0. 30 `git diff --check`: 0.
- Brief extras: `GOOS=windows GOARCH=amd64 go vet ./...`: 0;
  importer set (clonefidelity/clonebundle/environ/sessadapter/
  scalar/canonicaljson): all `ok`; new UTF-8 tests `-count=3`:
  0; clonefidelity `-v` 699 pass lines / 0 FAIL; whitespace scan
  of the new/edited untracked files: 0 issues.

## Handoff state

Candidate UNCOMMITTED in the Story worktree for the handoff
snapshot (4 modified landed files + new `internal/clonefidelity/`;
scratch-index tree `9870fb42e454a1163b4fc6abd778210260cf84d9`
over base `0ca3e4c`). Checklist: all items verified true this
run — checked. Attached (rev3): `TASK-260830-2ya5le_results.md`
(this file), `TASK-260830-2ya5le_conformance-matrix.md`,
`TASK-260830-2ya5le_producer-evidence.tar.gz` (real gzip, under
1 MiB: rev3 gate logs + 142 per-plant logs + `MANIFEST.sha256`).
