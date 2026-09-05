# TASK-260830-32jeti — restore accepted Story delta onto trunk 2512f20

## Base (step 1 of brief)

`git merge-base --is-ancestor 2512f20 HEAD` exits 0. Workspace HEAD is 2512f20
(`Record STORY-260830-3m2mw8 board state`), branch
`task-board/story/STORY-260830-3jqsx1`. Base confirmed before any change.

## Restore (steps 2–3 of brief)

- Applied `story-final-rev7.patch` (60 paths) excluding the 6 files that
  conflict with the landed sibling trunk: `LOGBOOK.md`, `README.md`, and the 4
  traceability files the sibling also touched
  (`ownership.v0.5.0.json`, `traceability.go`, `traceability_test.go`,
  `cmd/tracecheck/main_test.go`).
- The 54 applied files (`.github/workflows/ci.yml`, all of
  `internal/provider/`, all of `internal/provhost/`) are byte-identical to the
  pre-abort worktree (`worktree-pre-refresh.tgz`), which is the accepted tree.
- `LOGBOOK.md` / `README.md` taken from `LOGBOOK.merged.md` / `README.merged.md`.
- The 4 traceability files were three-way merged (base `57afcc6`, trunk HEAD,
  accepted tree) instead of taken byte-for-byte, because the brief's
  "every Go file matches the patch byte for byte" expectation is unsatisfiable
  there: both stories changed the same counters. Sibling: acceptance 74→77.
  Ours: acceptance 74→78, bindings 49→53, unevidenced 41→45, clauses 403→428.
  Overwriting with patch bytes would have dropped the sibling's landed
  ownership claims. Merge result:
  - `ownership.v0.5.0.json`: sibling's 3 blocks kept semantically identical to
    trunk (verified by JSON comparison), our 4 blocks appended after them;
    acceptance_cases array now holds 81 entries (74+3+4).
  - Counters recomputed from ground truth, not added by hand: merged JSON
    installed, `tracecheck` run, actual digest
    `c54d3475...f110674dd` pinned in `traceability.go`, measured report
    `acceptance_cases=81 bindings=53 unevidenced=45 clauses=17/428` pinned in
    `traceability_test.go` and `main_test.go` (both `-root` and `-section` arms).
- `README.md` ownership paragraph: merged file carried our story's figure of 78
  acceptance cases; the pin test
  `TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport` failed (published
  78, measured 81), and the single figure was corrected to 81. No other README
  figure changed (bindings 53, clauses 428, discharged 17 already matched).
- Sibling preservation: all 762 trunk `LOGBOOK.md` lines present in order, all
  86 sibling-added lines verbatim. README: 22 of 23 sibling-added lines
  verbatim; the 1 absent line is the sibling's `77 executable acceptance cases`
  fragment of the ownership paragraph both stories rewrote — the documented
  second README resolution, now carrying the measured 81 and guarded by the
  passing pin test.
- `git status -uall` shows exactly the patch's 60 paths, no extras.

## Gates (step 4 of brief; real exit codes, standalone processes)

| Gate | Exit | Note |
|---|---|---|
| `go test ./... -count=1` | 0 | all packages ok |
| `go test -race -count=1 ./internal/provider/... ./internal/provhost/... ./internal/traceability/...` | 0 | race scoped to touched trees; full native suite above covers the rest |
| `go vet ./...` | 0 | |
| `GOOS=windows go vet ./...` | 1 | PRE-EXISTING trunk failure, out of scope: `internal/terminalbackend/terminalbackend_test.go:972:20: undefined: syscall.Mkfifo`. File is byte-identical to trunk (sibling's, not in the 60 paths). Verdict: expected-red, not introduced here. |
| `gofmt -l .` | 0 | clean, no files listed |
| `go run ./internal/traceability/cmd/tracecheck -root .` | 0 | `traceability ok: contracts=60 normative_sections=36 acceptance_cases=81 fixtures=30 compatibility_contracts=55 assigned_scopes=0` + `section coverage: bindings=53 full=1 partial=3 sliver=1 unevidenced=45 unmeasured=3 unowned=2 clauses_discharged=17/428` |

## Publish (step 5 of brief)

Published via role handoff (`task-board handoff TASK-260830-32jeti --role
developer`), which snapshots the Change Request revision. Changed paths: 60.
No commit by hand, no touch of `main`, no push to any remote.
Stash `stash@{0}` ("pre-base-refresh…", from the earlier re-provisioning run)
was left untouched.
