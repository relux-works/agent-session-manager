# TASK-260830-2atgj4 results rev3: republish on fresh trunk (refresh/reconcile record)

Role: developer. Run: republish-rev3 (rebase onto trunk, no
re-implementation). Worktree:
`.temp/STORY-260830-1oqfec/worktree` on
`task-board/story/STORY-260830-1oqfec`, candidate left UNCOMMITTED
for the handoff snapshot. Status at handoff: ready for review.

The rev1 implementation record
(`TASK-260830-2atgj4_results.md`: 11 property tests, RED-first
`resolveWinner` fix, 8 narrowing mutants killed, traceability
story-close) stands unchanged: this run changed no product code
beyond the mechanical base reconciliation below (one doc-accuracy
line in the leaf's own LOGBOOK entry, noted), and re-ran the
full validation suite plus the property and mutation batteries on
the fresh base. All green.

## Refresh record (exact commands and OIDs)

Old Story base: trunk `e4e3e8834675cf3814effd3b2673b4931dfbf311`;
checkpoints `67445fa` (2f5393) and `ef71cef` (3g12yp). Fresh trunk
authority: `7bf90affef6c880e243f6176e6e5f762a293e1a3`
(STORY-260830-315721 provider-identity-profile-and-resume and
STORY-260830-2rqigd workspace-checkpoints-and-operation-journal,
both touching README/LOGBOOK/registry paths this Story owns).

1. `git fetch origin` (origin/main confirmed at `7bf90af`).
2. `task-board worktree refresh-candidate TASK-260830-2atgj4` ->
   the 2f5393 replay applied with NO conflict; stops at the
   3g12yp replay in the retained replay worktree
   `.temp/base-refresh/replay-1265929232/worktree`
   (`REBASE_HEAD=ef71cef5e094d085a3dde8bdc805709f27a85c4a`).
3. Deviation from the brief's expectation (LOGBOOK-only
   conflicts): LOGBOOK.md merged cleanly at both replays. The
   3g12yp replay conflicts are 5 files, all registry/README
   figure collisions between trunk's two stories and the fencing
   checkpoint: `README.md` (6 hunks), `internal/
   traceability/ownership.v0.6.0.json` (9 hunks),
   `internal/traceability/traceability.go` (digest pin),
   `internal/traceability/traceability_test.go` (Report
   figures), `internal/traceability/cmd/tracecheck/
   main_test.go` (2 figure pins).
4. Built the 5 resolutions (header copied from the retained
   `resolution-template.json`: version 1,
   branch `ef71cef5e094`, trunk `7bf90affef6c`, candidate tree
   `92d7da93a719`; every entry bound to checkpoint
   `ef71cef5e094d085a3dde8bdc805709f27a85c4a`):
   - `README.md`, 213793 bytes,
     sha256 `2f24d8f21e7b8ee5859ce63c9a0c6e3242cd812f1ad6506080c27eb15fe7d402`
   - `internal/traceability/cmd/tracecheck/main_test.go`,
     25865 bytes,
     sha256 `7f4c8d92c39f91766dd967855965763f865e0cb28f3eece93e20a20874023ce5`
   - `internal/traceability/ownership.v0.6.0.json`, 187236
     bytes,
     sha256 `df398705608cb683b84aced023ac2246a4429511bdfee74b52cf5b4b7e15aaad`
   - `internal/traceability/traceability.go`, 48524 bytes,
     sha256 `b0f374c5daa7675c92c06b70e24d3c2faaad2e80797024b2e0f2a75eaa0f2cf9`
   - `internal/traceability/traceability_test.go`, 52024
     bytes,
     sha256 `99b60fc1445a2a8770af6b04a923bab8d811231ea2002830ed1114256909a32c`
5. `task-board worktree refresh-candidate TASK-260830-2atgj4
   --replay-resolutions /tmp/resolutions-2atgj4-rev3.json` ->
   `refresh_advanced`,
   `BranchOID=995fe54531528bbed167653d44887d61a793f8ba`.

## Merge rules used for the 5 resolutions

- Registry JSON: semantic union, verified key by key against
  base/head/theirs. Acceptance cases: trunk's 119 plus the
  checkpoint's 9 `lease-*` cases appended in checkpoint order
  (128, no dupes). Ownership rows: trunk's 64 with the 5.3 row
  taken from the checkpoint (trunk never touched it) and the
  checkpoint's 2.2 row appended (65). Unowned: the checkpoint's
  11 (2.2 owned). No overlapping row modification on either
  side, so the union is unambiguous. JSON formatting
  round-trips byte-identically (`indent=2`, `ensure_ascii=False`,
  trailing newline).
- Digest pin: measured, not hand-computed — the merged registry
  was installed in the retained replay worktree and
  `tracecheck` reported projection digest
  `cc021c38c9c80cda7ebf12740380e1bfca9269ed4515003ffbb69abf69430fdb`,
  which was pinned. Merged figures: `acceptance_cases=128`,
  `bindings=60 full=2 partial=6 sliver=4 unevidenced=44
  unmeasured=4 unowned=11 clauses_discharged=49/511`
  (exactly trunk's figures plus the 2.2 binding and the 5.3
  clauses: 38+7+4 discharged, 489+22 total).
- README: trunk's two new sections kept contiguous in place; the
  fencing section appended after the resumesmoke section
  (order: Execution profiles, Native-resume smoke, Provider and
  pane fencing gates, Session selector). Coverage prose is
  trunk's text with the checkpoint's 5.3-partial and 2.2-sliver
  sentences inserted verbatim and every count recomputed
  (Sixty bindings, 49 of 511, six partial, four sliver,
  forty-four unevidenced, eleven unowned, sixty in the
  admission sentence).
- Both test-pin files: new-figure pins only (128/60/49/511).
- Pre-submission proof on the resolved replay tree (before the
  resolutions were submitted): `go build ./...` exit 0;
  `go test ./internal/sessstate ./internal/sessrepo
  ./internal/fencing ./internal/sessquery ./internal/
  traceability/... -count=1` all ok; `tracecheck` exit 0 with
  the merged figures above. The retained worktree was scratch
  only; the refresh re-applied the resolutions itself.

## Replayed-checkpoint verification

- Chain: `995fe54` (replayed 3g12yp) -> `bf21147`
  (replayed 2f5393) -> `7bf90af` (trunk).
  `git verify-commit` good on both replayed commits
  (oparin@me.com).
- 2f5393 replay: delta stat identical to the original (12
  files, 2356+/39-); every own-set blob byte-identical to the
  original except LOGBOOK.md (trunk entries above) and
  README.md (clean content union: trunk's two sections plus
  the checkpoint's Session Repository section, both verified
  present). No registry files in this checkpoint's set.
- 3g12yp replay: same 22-file set; all 15 `internal/fencing/*`
  blobs and `internal/sessquery/lease_store_adopt_test.go`
  byte-identical to the original. Differing own-set blobs are
  exactly the 5 resolved files plus LOGBOOK.md (trunk entries
  above; both checkpoint entries preserved verbatim: 2f5393
  and 3g12yp headings confirmed in the replayed LOGBOOK, trunk
  entries 17ootk/2zvo8m/3lt2xv/3uzfyn/kp4zpu/3k3e6m/14yo67
  intact ahead of them).
- Note: the brief's "`git diff <original> <replayed> --stat`
  empty except LOGBOOK" check cannot hold literally here
  because trunk advanced on README/registry paths the 3g12yp
  checkpoint also owns; the per-file blob record above (plus
  the merge rules) is the substitute evidence.

## Candidate reconciliation (by hand, no wholesale checkout)

After the refresh, 108 tracked paths were dirty: the 9
candidate modifications plus 99 stale carries (old-base board
state and pre-trunk file copies). All 99 were restored with
`git checkout HEAD -- <paths>` (classes: 18 `.task-board`
board-state files; trunk-new packages `internal/crashgate`,
`internal/matjournal`, `internal/resumesmoke`,
`internal/sessckpt`, `internal/sessprofile`, new `provhost`
files; trunk-modified `internal/environ/census_test.go`,
`internal/secprim/census_test.go`, `internal/provhost`
census/inventory/protocol files). `task-board.config.json` was
never dirty and equals HEAD (zero diff).

The 9 candidate paths were reconciled by three-way merge
(old=`ef71cef`, new=`995fe54`, stale=carried candidate) with
per-file verification:

| Path | Method | Record |
| --- | --- | --- |
| `internal/sessstate/sessstate.go` | clean (trunk untouched) | merged == stale bytes; RED-first fix intact |
| `internal/fencing/gate_test.go` | clean (trunk untouched) | merged == stale bytes; P3 vector intact |
| `internal/sessstate/census_state_test.go` | clean (trunk untouched) | merged == stale bytes; :929/:932 pins intact |
| `internal/traceability/ownership.v0.6.0.json` | clean line-merge, semantically verified | 131 cases (128 + 3 leaf), 65 rows; the 3 leaf rows (2.2, 5.3, 17.2 incl. the 17.2 gap reword) == stale, other 62 == new base; unowned/scalars identical |
| `internal/traceability/traceability.go` | pin recomputed | digest `39087ef8a9696ea46d9408d29c159c732c3c9ab229949f3f6c7c38f867aab8c3` measured via tracecheck over the final registry; file otherwise == new base |
| `internal/traceability/traceability_test.go` | Report figures -> 131 | leaf's only delta was the count; coverage counters already match (60/2/6/4/44/4/11, 49/511) |
| `internal/traceability/cmd/tracecheck/main_test.go` | pins -> 131 | both `want` lines; coverage line unchanged |
| `README.md` | section union + figure 128->131 | leaf property section + harness row carried; order ...resumesmoke, fencing, ownership-properties, session-selector; diff vs HEAD: 64 insertions, 1 deletion (the count line only); trunk's `-adopted` text intact |
| `LOGBOOK.md` | leaf entry first | 2atgj4, 17ootk, 2zvo8m, 3lt2xv, ... newest-first; diff vs HEAD purely additive (47 insertions, 0 deletions); the entry's count clause updated 110->113 to 128->131 (sole doc-accuracy edit; pins :929/:932 re-verified valid since trunk never touched sessstate.go) |

`git status` shows exactly the 13 candidate paths (9 modified
+ `internal/fencing/ownership_properties_test.go`,
`internal/sessrepo/lease_properties_test.go`,
`internal/sessstate/ownership_properties_test.go`,
`internal/sessstate/testdata/`); no `__pycache__`, no `.pyc`;
`.temp` scratch is gitignored.

## Re-run validation on the fresh base (all local, real exits)

Full 26-command suite from `task-board.config.json` (command 21
is the `-adopted` form), all exit 0:

| # | Command | Result |
| --- | --- | --- |
| 1 | gofmt clean | OK (empty) |
| 2 | `go build ./...` | exit 0 |
| 3 | `go vet ./...` | exit 0 |
| 4 | `go test ./... -count=1 -v` | exit 0; 32 ok, 0 FAIL; 1988 top-level `--- PASS` |
| 5 | `go test ./... -race -count=1` | exit 0; 32 ok, 0 FAIL, no data races |
| 6 | `go test ./... -cover -count=1` | exit 0; 32 ok (sessstate 92.4%, sessrepo 86.6%, fencing 97.5% — identical to rev2) |
| 7-11 | scalar + canonicaljson fuzz | 5/5 pass |
| 12-19 | secconftest fuzz | 8/8 pass |
| 20 | tracecheck | exit 0 (`acceptance_cases=131`, `clauses_discharged=49/511`) |
| 21 | cataloggen `-adopted -check` | exit 0 |
| 22-23 | linux/windows cross builds | exit 0 |
| 24 | JSON validation | exit 0 |
| 25 | `task-board validate` | exit 0 (board notices only) |
| 26 | `git diff --check` | exit 0 |

Property suites on the fresh base:

- Focused `go test ./internal/sessstate ./internal/sessrepo
  ./internal/fencing -run '^TestOwnership' -count=1 -v`: exit
  0, all 11 properties pass (`props-focused.log`).
- Mutation battery `mutate_properties.py`: harness exit 0 on
  the rebased tree; all 8 `N-` plants KILLED by their named
  properties (`N-compare-tiebreak` and `N-union-detail-first`
  by `TestOwnershipUnionOrderIndependent`, both loser-drop
  plants by `TestOwnershipLoserHistoryPreserved`,
  `N-create-clock` by `TestOwnershipStoreClockNonAuthority`,
  `N-expiry-absolute` by `TestOwnershipGateClockNonAuthority`,
  both gate plants by `TestOwnershipGateSingleAuthorizedOwner`);
  `C-harmless-comment` SURVIVED (applied),
  `C-not-applied` NOT_APPLIED, `C-compile-failure`
  COMPILE_OR_HARNESS_FAILURE; both controls exit 0
  (`mutant-battery.log`, `mutants/mutants-full.json` +
  per-plant raw logs).

The indented `--- FAIL` lines inside the verbose suite log are
the expected inner-harness plant failures captured by passing
tests; top-level `^FAIL` count is 0 in every batch.

## Evidence attached (rev3)

- `TASK-260830-2atgj4_results-rev3.md` (this file)
- `TASK-260830-2atgj4_conformance-matrix-rev3.md`
- `TASK-260830-2atgj4_producer-evidence-rev3.tar.gz` (real
  gzip: 26-command logs, focused property log, mutation
  battery logs + `mutants-full.json` + per-plant raw logs,
  replay resolutions record)
