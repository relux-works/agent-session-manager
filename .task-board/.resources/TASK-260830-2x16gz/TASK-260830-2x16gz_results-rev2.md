# TASK-260830-2x16gz results rev2 — base refresh + reconcile + full revalidation

Role: developer (republish run, no re-implementation).
Rev1 outcomes (`TASK-260830-2x16gz_results.md`, `..._conformance-matrix.md`,
`..._producer-evidence.tar.gz`) remain the implementation evidence; this rev2
record covers the base refresh, the candidate reconcile, and the rerun of the
full configured suite on the new base. All implementation ratios and bounds
from rev1 stand unchanged.

## 1. Refresh record

Story branch `task-board/story/STORY-260830-1kiyj6`, worktree
`.temp/STORY-260830-1kiyj6/worktree`.

- Pre-refresh tip: `244a7dce2f687c58238b1b11fbba67d618521632` (z1yxg9 checkpoint).
- Old trunk base: `62d446304391af187c417a88a2e14012b467956e`.
- Fresh authority: `origin/main = e4e3e8834675cf3814effd3b2673b4931dfbf311`
  (STORY-260917-158jyi release-agnostic catalog gate + board-state record).
- Command: `task-board worktree refresh-candidate TASK-260830-2x16gz`
  (+ `--replay-resolutions /tmp/republish-2x16gz/resolutions.json`).
- Outcome: `refresh_advanced`, new tip
  `c3de159762f3e3d0a926ece0b0d4231abb669562`.

Replayed checkpoints (original -> replayed, all `git verify-commit` Good for
`oparin@me.com`, parent chain reaches `e4e3e88`):

| # | Original | Replayed | Subject |
|---|---|---|---|
| 1 | `efe119a` | `14d1176bc794e460db6cf48d7436d3ccb601baa1` | 2u34k1 host/peer identity |
| 2 | `5e54874` | `4677a87bc4588c5e4d412a775113d4d51727f8a7` | 1tvg8e SSH transport |
| 3 | `ae5a4cc` | `a14f718288902627340f5f8086e9c984e060191a` | 2ez769 credentials/config4 |
| 4 | `244a7dc` | `c3de159762f3e3d0a926ece0b0d4231abb669562` | z1yxg9 RPC envelope/hello |

Checkpoints 1–2 replayed without conflict. Checkpoints 3–4 each stopped once
on a `LOGBOOK.md` append-append conflict; each was resolved with a
checkpoint-bound replacement (template
`{"version":1,"branch_oid","trunk_oid","candidate_tree_oid","resolutions":[...]}`):

| Resolution | `checkpoint_oid` (REBASE_HEAD) | Replacement sha256 | Bytes |
|---|---|---|---|
| 1 | `ae5a4cc5394ef80fc2fc3ef12965acc0670cb5a4` | `53ae8caf766ab8e929ec36c36e1457fb5789d71a2621a54ea92ba9618bc2208e` | 464104 |
| 2 | `244a7dce2f687c58238b1b11fbba67d618521632` | `ef8d5804d4e7581e4a0a455aeef7f64e5eef42ead97cba3060608c80b6dbb077` | 467766 |

Resolution rule (both): trunk's newer entries first, then the replayed
checkpoint's added block(s), everything else byte-identical. Verified per
resolution: `diff` against the replay side is a pure insertion (zero
deletions), every `### ` entry from both sides present, zero duplicates, zero
conflict markers. Note: the ae5a4cc checkpoint adds LOGBOOK entries in two
places (top rev12..09-10 block plus an end-of-file `## 2026-09-10 (later)`
CR3/rev8/rev9 block); both were carried.

Replay-fidelity verification (all four checkpoints):

- `git diff <original> <replayed> --name-only` equals exactly the 28-file
  trunk delta `62d4463..e4e3e88` (comm both directions empty).
- Every trunk-delta file except two is byte-identical between the replayed
  tree and `e4e3e88`. The two exceptions are correct 3-way unions, verified
  both directions: `README.md` (story Host-Trust section + trunk `-adopted`
  docs) and `task-board.config.json` (story rpcwire fuzz gate + trunk
  `-adopted` command 22).
- `git diff 244a7dc c3de159 -- internal/hostchannel internal/traceability`
  is empty (replay preserved the story implementation byte-for-byte).

## 2. Stale-copy audit (carried candidate vs new HEAD)

Pre-refresh safety: full candidate diff + both new test files copied to
`/tmp/republish-2x16gz/` before the refresh; the worktree-local `.task-board`
checkout artifact (spuriously modified by an earlier session running board
commands inside the worktree) was restored to `HEAD` before the refresh so
the carried delta held only the 13 candidate paths.

Trunk delta (`62d4463..e4e3e88`) is 28 files: 22 `.task-board` records,
4 `internal/catalog` files, `LOGBOOK.md`, `README.md`, `task-board.config.json`.

| Carried path in `git status` | In trunk delta? | Disposition |
|---|---|---|
| 8 `.task-board/...` paths (new-trunk records missing from stale copy) | yes | `git checkout HEAD -- .task-board` (artifact, not candidate) |
| `internal/catalog/catalog.go`, `catalog_test.go`, `cmd/cataloggen/main.go`, `cmd/cataloggen/main_test.go` | yes | `git checkout HEAD -- <each>` (stale pre-refresh copies; this task never touched catalog) |
| `task-board.config.json` | yes | `git checkout HEAD -- task-board.config.json`; verified `cmp` equals `HEAD` exactly (`HEAD` = trunk `-adopted` command 22 + story rpcwire fuzz gate, both intentional) |
| `LOGBOOK.md` | yes | Hand merge: candidate 2x16gz entry kept newest-first on top, then `HEAD` content byte-identical. `git diff HEAD` = 11 insertions, 0 deletions |
| `README.md` | yes | Hand merge: restored trunk's two `-adopted` regions (regenerate block, Go-toolchain row) verbatim; kept candidate figure rewrites (101->102 / 56->61 / 12->8) and hostile-suite docs. `git diff HEAD` = 22 insertions + 3 figure-line rewrites only |
| `internal/hostchannel/*` (5 tracked + 2 new), `internal/traceability/*` (4) | no | Untouched carries; tracked diff byte-identical to pre-refresh backup (modulo diff index lines); new files sha256-identical to backup |

Digest pin: `reviewedOwnershipCanonicalSHA256` stays
`e327bb08ae514279a8ab2b1f0ec58d02a5d368ca593cc14f3c3dd5f5dc6b14ba`.
No re-pin was required: trunk did not touch the traceability registry or its
pins, the carried registry is byte-identical to the reviewed rev1 bytes, and
`tracecheck` exits 0 with the pinned digest (report: 102 cases / 61 bindings /
8 unowned, matching the README figures).

Final `git status`: exactly the 13 candidate paths (11 modified + 2 new),
no `__pycache__`, no `.pyc` (`PYTHONDONTWRITEBYTECODE=1` throughout).
Candidate left UNCOMMITTED on the story branch; no commit, no push, no board
edit from this run beyond status/evidence.

## 3. Full validation rerun on `c3de159` (all observed this run)

Configured suite = 27 commands from `task-board.config.json`
(`spawn.worktree_isolation.validation.commands`):

| # | Command | Result |
|---|---|---|
| 1 | `gofmt` clean check | pass |
| 2 | `go build ./...` | pass |
| 3 | `go vet ./...` | pass |
| 4 | `go test ./... -count=1` | exit 0, 31 packages ok, 0 FAIL |
| 5 | `go test ./... -race -count=1` | exit 0, 31 ok, 0 FAIL, 0 data races |
| 6 | `go test ./... -cover -count=1` | exit 0 (config 92.9, hostchannel 84.1, hosttrust 76.3, localstore 83.7, peeridentity 97.5, rpcwire 97.8, secprim 94.4 — identical to rev1) |
| 7–20 | 14 fuzz smokes (rpcwire 1, scalar 1, canonicaljson 4, secconftest 8) | 14/14 pass |
| 21 | `tracecheck` | exit 0; 102 cases / 61 bindings / 8 unowned / 17/487 clauses |
| 22 | `cataloggen -adopted ... -check` (new trunk form) | exit 0, no generated drift |
| 23–24 | `GOOS=linux/windows go build ./...` | exit 0 both |
| 25 | JSON parse over tracked `*.json` | pass |
| 26 | `task-board validate` | exit 0; 191 pre-existing board-hygiene issues, 0 referencing this task |
| 27 | `git diff --check` | clean |

Own suites (this run):

- `go test ./internal/hostchannel -run 'TestHostile' -count=1 -v`: exit 0,
  54 `--- PASS`, 0 FAIL. `TestHostileRealCarrierOpenSSH` PASS (0.25 s —
  OpenSSH loopback lane executed, not skipped). The single SKIP is
  `TestHostileSSHHelperResponder`, the helper-process guard (skips unless
  invoked as the forced-command child — by design).
- Mutation batteries were NOT rerun in this republish run: reconciliation
  changed zero bytes of production or test source (hostchannel/rpcwire diff
  `244a7dc..c3de159` is empty; carried candidate bytes identical to backup),
  so the rev1 battery evidence (9/9 new probes killed, 26 prior probes
  re-run) still binds to this exact source. Full behavioral suites above
  were rerun instead.

## 4. Coverage restatement (unchanged from rev1)

- Hostile rows: 7 of 7 driven (`Dial`/`Serve`/`Call`).
- AC-HOST-001 rows: 14 of 15 fully driven; B8 partial (measured bound).
- HC-* families: 10 of 10. P3 notes: 2 of 2 closed.
- Stated bounds unchanged (native Tailscale SSH, non-host lanes, idle
  self-close measured, 11.10.5 policy reader by design, no `ax` CLI).
- No unsupported capability advertised; README/hostchannel docs and the
  traceability bindings already updated in the carried candidate.

## 5. Handoff evidence (rev2)

- `TASK-260830-2x16gz_results-rev2.md` (this file)
- `TASK-260830-2x16gz_conformance-matrix-rev2.md` (rev1 matrix + refresh addendum)
- `TASK-260830-2x16gz_producer-evidence-rev2.tar.gz` (real gzip: rev2 suite/
  race/cover/fuzz/tracecheck/cataloggen/cross-build/validate logs,
  hostile-focused log, both LOGBOOK resolutions with sha256, stale-copy
  audit diffs, rev2 matrix)
