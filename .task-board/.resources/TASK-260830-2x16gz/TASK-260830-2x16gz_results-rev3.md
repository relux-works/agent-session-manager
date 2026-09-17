# TASK-260830-2x16gz results rev3 — base refresh + reconcile + full revalidation

Role: developer (republish run, no re-implementation).
Rev1 outcomes (`TASK-260830-2x16gz_results.md`, `..._conformance-matrix.md`,
`..._producer-evidence.tar.gz`) remain the implementation evidence; rev2
(`..._results-rev2.md`, `..._conformance-matrix-rev2.md`) remains the first
refresh record. This rev3 record covers the second base refresh (onto
`fc67abd` after CR1 went stale via `integration_base_moved`), the candidate
reconcile, and the rerun of the full configured suite plus the shipped
mutation battery on the new base. All implementation ratios and stated bounds
from rev1 stand unchanged; registry/report figures moved additively with
trunk (see §2–§4).

## 1. Refresh record

Story branch `task-board/story/STORY-260830-1kiyj6`, worktree
`.temp/STORY-260830-1kiyj6/worktree`.

- Pre-refresh tip: `c3de159762f3e3d0a926ece0b0d4231abb669562` (z1yxg9 replay).
- Old trunk base: `e4e3e8834675cf3814effd3b2673b4931dfbf311`.
- Fresh authority: `origin/main = fc67abdfc3888d6683b1eb89bc248e089ca6aca3`
  (STORY-260830-315721, STORY-260830-2rqigd, STORY-260830-1oqfec, the PR48
  routing change, plus board-state records).
- Commands: `task-board worktree refresh-candidate TASK-260830-2x16gz`
  (+ `--replay-resolutions /tmp/rev3/resolutions-rev3.json`, two resolutions,
  accumulated across the two stops).
- Outcome: `refresh_advanced`, new tip
  `9636508e5a1d1fd3bd2fbd21e4cadcddd88aeecd`.

Replayed checkpoints (original -> replayed, all `git verify-commit` Good,
parent chain reaches `fc67abd`):

| # | Original | Replayed | Subject |
|---|---|---|---|
| 1 | `14d1176` | `3337a0b04c474f4bd1d300b8c4e0aa32cbfe0e99` | 2u34k1 host/peer identity |
| 2 | `4677a87` | `6ebe7748c81377939fec88cc0509337957389488` | 1tvg8e SSH transport |
| 3 | `a14f718` | `c3fec77d3a73be5bdecbb090178cf1103893d03c` | 2ez769 credentials/config4 |
| 4 | `c3de159` | `9636508e5a1d1fd3bd2fbd21e4cadcddd88aeecd` | z1yxg9 RPC envelope/hello |

Checkpoints 1–2 replayed without conflict. Unlike rev2, this cycle stopped
twice on non-LOGBOOK content conflicts (template
`{"version":1,"branch_oid","trunk_oid","candidate_tree_oid","resolutions":[...]}`,
header copied from each retained
`.temp/base-refresh/replay-*/resolution-template.json`, both headers
identical):

| Resolution | `checkpoint_oid` (REBASE_HEAD) | Path | Replacement sha256 | Bytes |
|---|---|---|---|---|
| 1 | `a14f718288902627340f5f8086e9c984e060191a` | `internal/secprim/census_test.go` | `7a99d2da0aad39657c98d42f109c7672a01b1b84ce1f7988c6129781dc1bace5` | 28287 |
| 2 | `c3de159762f3e3d0a926ece0b0d4231abb669562` | `internal/traceability/traceability.go` | `4497782a1f1814a7befbed6eca6bcbe15e589d1f8904a94376fd2692c0584a8f` | 48524 |

Resolution 1 rule: same-anchor row insertion — trunk's 1 `provhost`
census row plus the checkpoint's 6 credential/custody rows are all kept
(trunk row first, then the 6 checkpoint rows as new rows). The census
checks are order-independent set checks (no row-count assertion), so the
union is exact; the merged file is `gofmt`-clean and parses.

Resolution 2 rule: the pinned `reviewedOwnershipCanonicalSHA256` line
conflicted (trunk `39087ef8…` vs checkpoint `5102d3bc…`) while the
registry itself auto-merged (trunk's 65 bindings + the checkpoint's 7
gap-line enrichments). Neither pin fits the merged registry, so the
replacement re-pins to the recomputed canonical digest `eefba9e5…` of
the merged bytes. The recompute method (standalone Go over verbatim
projection structs, no custom marshalers in the path) was verified by
reproducing BOTH known digests first: trunk registry -> `39087ef8…`,
`c3de159` registry -> `5102d3bc…`.

Replay-fidelity verification (all four checkpoints):

- `git verify-commit` Good on all four replayed commits.
- Per-commit patch paths identical: `git diff <o>^ <o> --name-only` equals
  `git diff <n>^ <n> --name-only` for all four pairs (4/4 identical).
- Resolution blobs byte-identical in the replayed commits (`git show
  c3fec77:internal/secprim/census_test.go` sha256 = `7a99d2da…`;
  `git show 9636508:internal/traceability/traceability.go` sha256 =
  `4497782a…`).
- LOGBOOK.md auto-merged at every step with zero line losses in both
  directions (`comm` of sorted lines vs trunk and vs `c3de159`: both
  empty); no LOGBOOK resolution was needed this cycle. Note on the
  brief's literal check: `git diff <original> <replayed> --stat` compares
  full trees and therefore always shows the 321-file trunk delta after a
  base move; the per-commit patch equivalence above is the operative
  fidelity check, and it holds exactly.

## 2. Stale-copy audit (carried candidate vs new HEAD)

Pre-refresh safety: the 11 tracked candidate files, their leaf diffs vs
`c3de159`, and both new test files were copied to `/tmp/rev3/` (kept in
the evidence tar) before reconciling.

After `refresh_advanced` the worktree held 152 entries: the 13 candidate
paths plus 139 stale carried copies. Every stale path is in the trunk
delta `e4e3e88..fc67abd` (321 files; machine-checked, zero unexplained),
so all 139 were restored with `git checkout HEAD -- <paths>`:

| Stale group | Paths | Disposition |
|---|---|---|
| `internal/provhost` (18), `fencing` (16), `sessprofile` (15), `resumesmoke` (14), `matjournal` (12), `crashgate` (11), `sessrepo` (10), `sessckpt` (9), `sessstate` (4), `sessquery` (1), `secprim/census_test.go` (1), `environ/census_test.go` (1) | 112 | trunk-landed story files reverted/deleted by the carry; restored to `HEAD` |
| `.task-board/.activity` (13) + `.task-board/.../progress.md` (13) | 26 | board-checkout artifacts inside the worktree; restored to `HEAD` |
| `task-board.config.json` | 1 | restored to `HEAD`; verified byte-identical (`git diff HEAD` empty). `HEAD` = trunk `-adopted` command + the story's rpcwire fuzz gate (both intentional, as in rev2); this leaf adds nothing |

Six candidate paths are also in the trunk delta and were hand-merged
(base `c3de159` blob, ours = stale leaf copy, theirs = new `HEAD`):

| Candidate path | Merge |
|---|---|
| `internal/traceability/ownership.v0.6.0.json` | 3-way semantic union, machine-verified: key sets = exact union (70 ownership / 132 acceptance / 7 unowned); leaf adds 5 section bindings + `AC-HOST-001` + 2 gap mods + `11.10.5` unowned text; trunk adds 30 cases + 4 bindings + 7 gap mods + `2.2` unowned removal; modified-key sets disjoint. Single textual conflict (acceptance-array same-anchor insert) spliced trunk-first (30 trunk entries, then `AC-HOST-001`), JSON-validated |
| `internal/traceability/traceability.go` | both sides changed only the pin line; final = `HEAD` + recomputed pin `d3eca906…` over the merged registry (same verified method as resolution 2) |
| `internal/traceability/traceability_test.go`, `internal/traceability/cmd/tracecheck/main_test.go` | figure conflicts resolved to trunk + leaf-delta pins (132 cases / 65 bindings / full 2 / partial 6 / sliver 4 / unevidenced 49 / unmeasured 4 / unowned 7 / clauses 49/535); trunk's `2.2`/`2.4`/`18.4` expectation updates kept via auto-merge |
| `README.md` | single figure conflict resolved to 132/65/7; leaf hostile-suite paragraph + 2 tool-table rows auto-merged in place with correct context |
| `LOGBOOK.md` | leaf entry block (9 lines) inserted verbatim after the first `## 2026-09-17` heading; `git diff HEAD` = 9 insertions, 0 deletions |

Seven candidate paths are clean carries (not in the trunk delta):
5 tracked `internal/hostchannel` files (base blobs identical across the
refresh, diffs untouched) plus the 2 new `hostile_*_test.go` files.

Digest re-pin: `reviewedOwnershipCanonicalSHA256` is now
`d3eca9063c0f7b7a6abc38a7f53c5d6c01bdbc076886e19a2c66e0a87693e58a`
(canonical sha256 of the merged registry; `tracecheck` exits 0 and
`TestVerifyRepositoryAcceptsExactOwnership` passes on it).

Final `git status`: exactly the 13 candidate paths (11 modified + 2 new),
no `__pycache__`, no `.pyc` (`PYTHONDONTWRITEBYTECODE=1` throughout),
no staged changes. Candidate left UNCOMMITTED on the story branch; no
commit, no push, no board edit from this run beyond status/evidence.

## 3. Full validation rerun on `9636508` (all observed this run)

Configured suite = 27 commands from `task-board.config.json`
(`spawn.worktree_isolation.validation.commands`):

| # | Command | Result |
|---|---|---|
| 1 | `gofmt` clean check | pass |
| 2 | `go build ./...` | pass |
| 3 | `go vet ./...` | pass |
| 4 | `go test ./... -count=1 -v` | exit 0, 37 packages ok, 0 FAIL, 21523 `--- PASS` (indented `--- FAIL` lines in the log are mutant-harness log echoes, e.g. resumesmoke `TestSmokeMutantsAreKilled` quoting killed-mutant subprocess output; zero top-level `^--- FAIL`, zero `^FAIL`) |
| 5 | `go test ./... -race -count=1` | exit 0, 37 ok, 0 FAIL, 0 data races |
| 6 | `go test ./... -cover -count=1` | exit 0 (hostchannel 84.1, hosttrust 76.3, rpcwire 97.8, traceability 86.2, tracecheck 88.5) |
| 7–20 | 14 fuzz smokes (rpcwire 1, scalar 1, canonicaljson 4, secconftest 8) | 14/14 pass |
| 21 | `tracecheck` | exit 0; `acceptance_cases=132 ... bindings=65 ... unowned=7 clauses_discharged=49/535` |
| 22 | `cataloggen -adopted ... -check` | exit 0, no generated drift |
| 23–24 | `GOOS=linux/windows go build ./...` | exit 0 both |
| 25 | JSON parse over tracked `*.json` | pass |
| 26 | `task-board validate` | exit 0; 191 pre-existing board-hygiene issues, 0 referencing this task/story |
| 27 | `git diff --check` | clean |

Own suites (this run):

- `go test ./internal/hostchannel -run 'TestHostile' -count=1 -v`: exit 0,
  54 `--- PASS`, 0 FAIL. `TestHostileRealCarrierOpenSSH` PASS (0.30 s —
  OpenSSH loopback lane executed, not skipped). The single SKIP is
  `TestHostileSSHHelperResponder`, the helper-process guard (skips unless
  invoked as the forced-command child — by design).
- `python3 internal/hostchannel/mutations.py --output
  .temp/TASK-260830-2x16gz/mutations-hostchannel-rev3`: exit 0, 33/33
  registered probes executed, full behavioral suite per plant, sources
  untouched (overlay). Verdicts: 31 killed, 1 neutral passed, 1
  survived (`client-cache`, exit 0, the documented harmless control —
  caching client sessions while the server issues no tickets changes
  no behavior). All 9 new narrowing probes killed by their named
  vectors: `alpn-same-family` by
  `TestVerifyPeerRefusals/alpn-same-family`, `oversize-dispatch` by
  `TestHostileOversizedFrames/dispatch_line_8mib_plus_one`,
  `stale-client-rebind` by
  `TestHostileRecoveryBypass/stale_binding_never_rebinds`,
  `rpc-contracts` by
  `TestHostileDisclosureMismatch/responder_contracts_drift`,
  `error-version-drift` by
  `TestHostileDisclosureMismatch/error_version_drift`,
  `truncated-dispatch-ignored` by
  `TestHostileDisconnectPhases/mid_request_close`, `snapshot-integrity`
  by `TestHostileStaleReadsAreIntegrityFailures/corrupt_store`,
  `correlation-health` by
  `TestHostileRecoveryBypass/response_id_mismatch_refused`,
  `nonce-static` by `TestHostileReplayNonceFreshness`. Prior 24 probes:
  22 killed, neutral passed, `client-cache` survived as designed.
  (Denominator note: rev1 text says "26 prior"; the shipped registry —
  byte-identical then and now — holds 24 prior + 9 new = 33, and this
  rerun executed all 33. Observed counts above are exact.)

## 4. Coverage restatement (unchanged from rev1/rev2)

- Hostile rows: 7 of 7 driven (`Dial`/`Serve`/`Call`).
- AC-HOST-001 rows: 14 of 15 fully driven; B8 partial (measured bound).
- HC-* families: 10 of 10. P3 notes: 2 of 2 closed.
- Stated bounds unchanged (native Tailscale SSH, non-host lanes, idle
  self-close measured, 11.10.5 policy reader by design, no `ax` CLI).
- Registry/report figures moved with trunk: 102->132 cases, 61->65
  bindings, 8->7 unowned, 17/487->49/535 clauses (trunk +30 cases /
  +4 bindings / -1 unowned / +32 discharged clauses composed with the
  leaf's +1 case / +5 bindings / -4 unowned; verified by the passing
  report test, not by arithmetic alone).
- No unsupported capability advertised; README/hostchannel docs and the
  traceability bindings already updated in the carried candidate.

## 5. Handoff evidence (rev3)

- `TASK-260830-2x16gz_results-rev3.md` (this file)
- `TASK-260830-2x16gz_conformance-matrix-rev3.md` (rev2 matrix + rev3 addendum)
- `TASK-260830-2x16gz_producer-evidence-rev3.tar.gz` (real gzip: rev3 suite/
  race/cover/fuzz/tracecheck/cataloggen/cross-build/validate logs,
  hostile-focused log, mutation battery logs, both replay resolutions
  with sha256, stale-copy audit lists, digest-recompute sources, rev3 matrix)
