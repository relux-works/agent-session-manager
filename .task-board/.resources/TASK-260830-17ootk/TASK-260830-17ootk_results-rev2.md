# TASK-260830-17ootk results rev2 — republish onto trunk 9e9fe51

## Outcome

Ready for review (republish). The rev1 implementation (accepted by
RUN-260917-be21ad, then demoted to stale by `integration_base_moved` on
LOGBOOK.md) is preserved and reconciled onto trunk `9e9fe51`
(STORY-260830-315721). No product-code change beyond the reconciliation:
`internal/crashgate` is byte-identical to the accepted revision; the six
tracked candidate paths carry the rev1 candidate delta re-applied onto the
replayed base with re-pinned figures.

## Refresh record (exact commands, OIDs, bindings)

Trunk: `origin/main = 9e9fe5154f5e8d97151670c3f689821cc3431d79` (confirmed
after `git fetch origin`). Story branch `task-board/story/STORY-260830-2rqigd`
tip before refresh: `bde093e242b688d8348818706e4720fed3eec96d` on base
`e4e3e8834675cf3814effd3b2673b4931dfbf311`.

1. `task-board worktree refresh-candidate TASK-260830-17ootk` → stopped on
   `UU LOGBOOK.md` at checkpoint `8a757e0f602e3a42acf157533785bf5f3ea36811`;
   retained replay `replay-2130542576`.
2. Built `/tmp/republish_resolutions.json` (header copied from the retained
   `resolution-template.json`: branch `bde093e…`, trunk `9e9fe51…`,
   candidate tree `25b762c…`) with resolution #1:
   `{checkpoint_oid: 8a757e0…, path: LOGBOOK.md,
   sha256: c30af45c4e5999920404b30f0bd484c8e3659fbdeb63113fb12f72887341c508}`
   (trunk entries `3uzfyn`, `kp4zpu` first, then the checkpoint's `14yo67`
   entry, newest-first; everything else byte-identical).
3. Re-ran with `--replay-resolutions` → checkpoint 1 replayed; stopped at
   checkpoint `bde093e242b688d8348818706e4720fed3eec96d` with six
   conflicted files (retained replay `replay-1380464330`). The trunk side
   and the checkpoint side both extended the traceability registry and
   figures, so a LOGBOOK-only resolution was insufficient; each file got a
   union resolution:
   - `LOGBOOK.md` (sha256 `64ccbe9dcb4a…`): trunk entries + `3k3e6m` entry.
   - `README.md` (`4682b10c08d5…`): replay-merged figures (117 cases,
     59 bindings) + trunk prose.
   - `internal/traceability/ownership.v0.6.0.json` (`c6c7df42277a…`):
     case union 101 shared + 12 trunk + 4 checkpoint = 117; ownership
     arrays were already identical on both sides.
   - `internal/traceability/traceability.go` (`afb6e8e00725…`): digest
     re-pinned to the merged projection
     `89e9e8fd1a42363673be341e0600a6dff0085cfd143705d946f901272977eef0`
     (read from the `tracecheck` refusal line, then verified green).
   - `internal/traceability/traceability_test.go` (`619d217ec855…`) and
     `cmd/tracecheck/main_test.go` (`5b535c3630b0…`): pins set to the
     merged report (`bindings=59 full=2 partial=4 sliver=3 unevidenced=46
     unmeasured=4 unowned=12 clauses_discharged=29/489`).
   Merged-report truth was established by running the real `tracecheck`
   and `go test ./internal/traceability/...` inside the replay worktree
   (both green) before sealing the resolutions.
4. Re-ran with the 7-resolution file → `refresh_advanced`, new branch tip
   `813b05d3c43605c8e54a8d54f410455b6463b086`.
5. Verified: parent chain `813b05d → 8299fdc → 9e9fe51`; both replayed
   commits signed (`git verify-commit` OK); `git diff 9e9fe51 8299fdc`
   shows only the 14yo67 delta (10 files), `git diff 8299fdc 813b05d`
   only the 3k3e6m delta (19 files). Note: the republish brief expected
   `git diff <original> <replayed>` to be empty except LOGBOOK.md; that
   holds only for LOGBOOK-single-conflict refreshes. Here trunk changed
   the same traceability files, so the replayed trees necessarily carry
   the merged figures — the per-delta check above is the meaningful
   invariant and it holds.

## Stale-copy audit (path-by-path)

After the refresh the working tree overlaid pre-refresh bytes on the new
HEAD. 59 non-candidate paths restored verbatim with
`git checkout HEAD -- <path>` (list in `/tmp/republish_restore_list.txt`
during the run; classes below):

| Class | Paths | Action |
| --- | --- | --- |
| `.task-board/` activity/progress (checkout artifact) | 10 M/D entries | restored to HEAD |
| Trunk-landed packages (`internal/provhost`, `internal/resumesmoke`, `internal/sessprofile`, `internal/environ`, `internal/secprim`) | 49 M/D entries | restored to HEAD |
| `task-board.config.json` | unmodified | equals HEAD byte-for-byte (`cmp` clean); no Story-carried delta |

The six candidate paths were hand-merged (HEAD content + rev1 candidate
delta re-applied; never wholesale checkout):

| Candidate path | Merge |
| --- | --- |
| `LOGBOOK.md` | rev1 17ootk entry re-prepended onto HEAD (figures refreshed: 119 cases, digest `ba242a9d…0937afd`) + 3-line REPUBLISH bullet; diff vs HEAD purely additive |
| `README.md` | crashgate package section re-inserted at the matjournal anchor (byte-identical block); figures re-pinned (119 cases, 38/489, five partial incl. 13.13 at 9/11, four unmeasured incl. 13.12, forty-five unevidenced, fifty-nine bindings); removed lines are figure-only |
| `internal/traceability/ownership.v0.6.0.json` | HEAD union + 2 crash-gate cases appended (render byte-verified) + 13.12/13.13 entries transplanted from rev1 (HEAD==oldbase precondition verified first) → 119 cases |
| `internal/traceability/traceability.go` | HEAD + digest re-pin `ba242a9d55e11da9299d3732197311f1e874be3f43e0bf11d0ec00de30937afd` |
| `internal/traceability/traceability_test.go` | HEAD + pins (119 / partial 5 / unevidenced 45 / discharged 38) |
| `internal/traceability/cmd/tracecheck/main_test.go` | HEAD + CLI-output pins |
| `internal/crashgate/` (untracked) | carried as-is; verified pin-free vs repo-wide figures and byte-identical to rev1 |

## Validation (rev2, all exit 0)

Full 26-command suite from the worktree's `task-board.config.json`, plus
the leaf's own suites. Logs: `TASK-260830-17ootk_producer-evidence-rev2.tar.gz`.

| # | Command | Result |
| --- | --- | --- |
| 1 | gofmt clean check | exit 0 |
| 2 | `go build ./...` | exit 0 |
| 3 | `go vet ./...` | exit 0 |
| 4 | `go test ./... -count=1 -v` | exit 0 — 31 pkgs ok, 1907 `--- PASS`, 0 fail |
| 5 | `go test ./... -race -count=1` | exit 0 — 31 pkgs ok, 0 fail, no data races |
| 6 | `go test ./... -cover -count=1` | exit 0 — 31 pkgs ok (crashgate 36.8%, matjournal 77.0%, sessckpt 82.1%) |
| 7–19 | fuzz smoke (13 targets at 100x) | 13 elapsed, 0 fail |
| 20 | `tracecheck` | ok, 119 cases, 38/489 clauses |
| 21 | `cataloggen -adopted ... -check` | exit 0 |
| 22–23 | `GOOS=linux/windows go build ./...` | exit 0 |
| 24 | JSON well-formedness census | exit 0 |
| 25 | `task-board validate` | exit 0 |
| 26 | `git diff --check` | exit 0 |
| own | crashgate mutant harness | 9 KILLED, 1 SURVIVED control, exit 0 |
| own | conformance records regeneration | 74 records, byte-identical to rev1 (`diff -rq` clean) |

Rev1 coverage ratios are unchanged by the republish (4 of 4 AC rows; 48 of
48 reachable paths; 121 of 121 N/A paths; 9 of 11 clauses; 7 of 19 §13.12
rows; 10 of 10 rejections; 9 of 9 narrowing mutants killed) — see
`TASK-260830-17ootk_results.md` rev1, whose implementation sections remain
accurate; only base-dependent figures move (107→119 cases, 26/489→38/489,
digest `a1b48479…`→`ba242a9d…0937afd`).

## Candidate paths (`git status` at handoff)

- `M README.md`, `M LOGBOOK.md`
- `M internal/traceability/ownership.v0.6.0.json`
- `M internal/traceability/traceability.go` (re-pin)
- `M internal/traceability/traceability_test.go` (report counts)
- `M internal/traceability/cmd/tracecheck/main_test.go` (CLI output)
- `?? internal/crashgate/` (registry, tests, harness, TRACEABILITY.md)

Left UNCOMMITTED for the handoff snapshot. No `__pycache__`, no `.pyc`.

## Evidence attached (rev2)

- `TASK-260830-17ootk_results-rev2.md` (this file)
- `TASK-260830-17ootk_conformance-matrix-rev2.md` (rev1 matrix + refresh note)
- `TASK-260830-17ootk_producer-evidence-rev2.tar.gz` (records, mutant logs,
  full 26-command logs)
