# TASK-260830-2atgj4 results rev2: republish on fresh trunk (refresh/reconcile record)

Role: developer. Run: republish-rev1 (rebase onto trunk, no
re-implementation). Worktree:
`.temp/STORY-260830-1oqfec/worktree` on
`task-board/story/STORY-260830-1oqfec`, candidate left UNCOMMITTED
for the handoff snapshot. Status at handoff: ready for review.

The rev1 implementation record
(`TASK-260830-2atgj4_results.md`: 11 property tests, RED-first
`resolveWinner` fix, 8 narrowing mutants killed, traceability
story-close) stands unchanged: this run changed no product code
beyond the mechanical base reconciliation below, and re-ran the
full validation suite plus the property and mutation batteries on
the fresh base. All green.

## Refresh record (exact commands and OIDs)

Old Story base: trunk `62d4463`; checkpoints `0ebb7fa` (2f5393)
and `1b8e75a` (3g12yp). Fresh trunk authority:
`e4e3e8834675cf3814effd3b2673b4931dfbf311` (STORY-260917-158jyi:
release-agnostic cataloggen `-adopted` check).

1. `git fetch origin`
2. `task-board worktree refresh-candidate TASK-260830-2atgj4` ->
   stops at the 2f5393 replay with `UU LOGBOOK.md` in the
   retained replay worktree
   `.temp/base-refresh/replay-2355938507/worktree`
   (`REBASE_HEAD=0ebb7faf656ebc9059d7f51f652fee08c680f8bf`;
   HEAD side = trunk's 3lt2xv block, theirs = the 2f5393 block,
   both under `## 2026-09-17`, exactly as predicted).
3. Built the resolved LOGBOOK (markers removed, both blocks kept,
   trunk 3lt2xv first then 2f5393, one blank line between,
   everything else byte-identical;
   sha256 `6524d022b343605e2a860556350b7e0bcf028754bfdf5062c2d880ba8ef1f17b`,
   432317 bytes) and wrote the retained
   `resolution-template.json` shape with one resolution bound to
   checkpoint `0ebb7faf656ebc9059d7f51f652fee08c680f8bf`.
4. `task-board worktree refresh-candidate TASK-260830-2atgj4
   --replay-resolutions /tmp/resolutions-2atgj4.json` -> 2f5393
   replay accepted; stops at the 3g12yp replay with
   `UU LOGBOOK.md` in
   `.temp/base-refresh/replay-3296328019/worktree`
   (`REBASE_HEAD=1b8e75ac703eaa5b6a3f9bb009709d74a759c015`).
5. Added the second resolution bound to
   `1b8e75ac703eaa5b6a3f9bb009709d74a759c015` (content keeps
   3lt2xv, then the 3g12yp block, then 2f5393;
   sha256 `bd7944f0aa6f0b52eeaddedb9b5643c9123103450cb817629ea14c39fa046b83`,
   434919 bytes) and reran -> `refresh_advanced`,
   `BranchOID=ef71cef5e094d085a3dde8bdc805709f27a85c4a`.

## Replayed-checkpoint verification

- Chain: `ef71cef` (replayed 3g12yp) ->
  `67445fa2da460b7c6610e78294b72627538c00cf` (replayed 2f5393)
  -> `e4e3e88` (trunk). `git verify-commit` good on both
  replayed commits (oparin@me.com).
- Whole-tree `git diff <original> <replayed> --stat` shows only
  the trunk delta (3lt2xv board files, LOGBOOK +6, README,
  `internal/catalog/*`, `task-board.config.json`); none of the
  checkpoints' own files (sessrepo, fencing, traceability,
  sessquery) differ. Each replayed checkpoint's own file set is
  intact except the LOGBOOK resolution.

## Candidate reconciliation (by hand, no wholesale checkout)

`git checkout HEAD -- task-board.config.json
internal/catalog/catalog.go internal/catalog/catalog_test.go
internal/catalog/cmd/cataloggen/main.go
internal/catalog/cmd/cataloggen/main_test.go .task-board`
restored every trunk-changed path outside the 13 candidate paths
(`task-board.config.json` now equals HEAD, zero diff).

- LOGBOOK.md: inserted trunk's 3lt2xv block verbatim between the
  candidate 2atgj4 block and the 3g12yp block. Order is now
  2atgj4, 3lt2xv, 3g12yp, 2f5393 (newest-first by wall time);
  `git diff` vs HEAD is purely additive (48 insertions, 0
  deletions).
- README.md: restored trunk's `-adopted` regenerate block and
  Go-toolchain row cell verbatim; kept the candidate
  property-suite section, the `ownership properties` harness
  table row, and the 110->113 count. Diff vs HEAD: 64
  insertions, 1 deletion (the count line only).

`git status` shows exactly the 13 candidate paths (9 modified +
`internal/fencing/ownership_properties_test.go`,
`internal/sessrepo/lease_properties_test.go`,
`internal/sessstate/ownership_properties_test.go`,
`internal/sessstate/testdata/mutate_properties.py`); no
`__pycache__`, no `.pyc`.

## Re-run validation on the fresh base (all local, real exits)

Full 26-command suite from `task-board.config.json` (command 21
is the `-adopted` form), all exit 0:

| # | Command | Result |
| --- | --- | --- |
| 1 | gofmt clean | OK (empty) |
| 2 | `go build ./...` | exit 0 |
| 3 | `go vet ./...` | exit 0 |
| 4 | `go test ./... -count=1 -v` | exit 0; 27 ok, 0 FAIL; 19910 `--- PASS` |
| 5 | `go test ./... -race -count=1` | exit 0; 27 ok, 0 FAIL |
| 6 | `go test ./... -cover -count=1` | exit 0; 27 ok (sessstate 92.4%, sessrepo 86.6%, fencing 97.5%) |
| 7-11 | scalar + canonicaljson fuzz | 5/5 pass |
| 12-19 | secconftest fuzz | 8/8 pass |
| 20 | tracecheck | exit 0 (`acceptance_cases=113`, `clauses_discharged=28/485`) |
| 21 | cataloggen `-adopted -check` | exit 0 |
| 22-23 | linux/windows cross builds | exit 0 |
| 24 | JSON validation | exit 0 |
| 25 | `task-board validate` | exit 0 (191 pre-existing board notices) |
| 26 | `git diff --check` | exit 0 |

Property suites on the fresh base:

- Focused `go test ./internal/sessstate ./internal/sessrepo
  ./internal/fencing -run '^TestOwnership' -count=1`: exit 0,
  all 11 properties pass
  (`props-focused.log`).
- Mutation battery `mutate_properties.py`: harness exit 0; all
  8 `N-` plants KILLED by their named properties
  (`N-compare-tiebreak` and `N-union-detail-first` by
  `TestOwnershipUnionOrderIndependent`, both loser-drop plants
  by `TestOwnershipLoserHistoryPreserved`, `N-create-clock` by
  `TestOwnershipStoreClockNonAuthority`, `N-expiry-absolute` by
  `TestOwnershipGateClockNonAuthority`, both gate plants by
  `TestOwnershipGateSingleAuthorizedOwner`);
  `C-harmless-comment` SURVIVED (applied),
  `C-not-applied` NOT_APPLIED, `C-compile-failure`
  COMPILE_OR_HARNESS_FAILURE; both controls exit 0
  (`mutants-run.log`, `mutants/mutants-full.json` + per-plant
  raw logs).

The indented `--- FAIL` lines inside the verbose suite log are
the expected inner-harness plant failures captured by passing
tests; top-level `^FAIL` count is 0 in every batch.

## Evidence attached (rev2)

- `TASK-260830-2atgj4_results-rev2.md` (this file)
- `TASK-260830-2atgj4_conformance-matrix-rev2.md`
- `TASK-260830-2atgj4_producer-evidence-rev2.tar.gz` (real
  gzip: 26-command logs, focused property log, mutation
  battery logs + `mutants-full.json` + per-plant raw logs)
