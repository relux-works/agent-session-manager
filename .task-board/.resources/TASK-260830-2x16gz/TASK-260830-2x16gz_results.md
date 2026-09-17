# TASK-260830-2x16gz results — hostile-network conformance + AC-HOST-001

Role: developer. Worktree: `.temp/STORY-260830-1kiyj6/worktree` at `244a7dc`
(z1yxg9 checkpoint). Test-only leaf: no production file changed.

## Deliverable

Executable conformance suite over the landed production entries
(`hostchannel.Dial`/`Serve`/`Call`, hosttrust enrollment/rotation/revocation/
generation, Config-4 selection, SSH transport boundary):

- `internal/hostchannel/hostile_test.go` (new, ~2000 lines): seven hostile
  cases re-derived through the real authenticated channel + AC-HOST-001 rows
  (rotation lifecycle, copied-key halves, revocation fencing + watch,
  stale-read integrity, recovery bypasses, concurrency/stall races, role swap,
  v4 loader gate), every vector with Section 15 class + handler-effect + state
  census assertions.
- `internal/hostchannel/hostile_carrier_test.go` (new): real OpenSSH loopback
  carrier lane (unprivileged `sshd`, forced command, exact `ServeArgv` on the
  wire observed server-side, TLS-first bytes, no PTY, bounded stderr) with an
  asserted skip when `sshd` cannot bind.
- `internal/hostchannel/mutations.py`: 9 new narrowing probes (multi-file
  plant support added for the dual-enforced line cap) + shipped harness.
- `internal/hostchannel/verify_test.go`: `ax-host/2` same-family ALPN row (P3a).
- `internal/hostchannel/refusal_test.go`: bad-destination deadline hook (P3b).
- `internal/hostchannel/gates_test.go`: registry extended with the new vectors.
- `internal/traceability/ownership.v0.6.0.json`: `AC-HOST-001` registered over
  20 suite tests; new `unevidenced` bindings for 11.10.1–11.10.4 + 16.1;
  6.6/11.10 gaps refreshed; 11.10.5 stays unowned by design; digest re-pinned.
- `internal/traceability/traceability.go`: `reviewedOwnershipCanonicalSHA256`
  re-pinned to `e327bb08…`; `main_test.go` + `traceability_test.go` report pins
  updated (102 cases / 61 bindings / 8 unowned / 17/487 clauses).
- `README.md` + `internal/hostchannel/README.md`: suite docs, test commands,
  refreshed stated bounds; no capability claims (no `ax` command exists).
- `LOGBOOK.md`: entry prepended.

## Coverage ratios (production call site per row in the matrix)

- Hostile rows: **7 of 7** driven (`Dial`/`Serve`/`Call`).
- AC-HOST-001 rows: **14 of 15** fully driven; B8 (revocation ≤ 1 s) partial —
  next-dispatch fence + `WatchGeneration` measured < 1 s, idle-without-traffic
  self-close measured NOT satisfied (stated bound with log evidence).
- HC-* families: **10 of 10** with positive + negative vectors.
- P3 notes: **2 of 2** closed.
- Narrowing mutants: **9 of 9** new probes killed by named tests; 26 prior
  probes re-run (neutral passed, 24 killed, 1 documented harmless SURVIVED).

## Validation (this worktree, exit codes observed)

- `gofmt` check: clean. `go build ./...`: ok. `go vet ./...`: exit 0.
- `go test ./... -count=1`: exit 0, 31 packages ok, 0 failures.
- `go test ./... -race -count=1`: exit 0, 31 packages ok, no warnings.
- `go test ./... -cover -count=1`: exit 0 (hostchannel 84.1%, rpcwire 97.8%,
  hosttrust 76.3%).
- Fuzz smoke 14/14: rpcwire 1 + scalar 1 + canonicaljson 4 + secconftest 8.
- `tracecheck`: ok (report above). `cataloggen -check` (worktree-configured
  `-metadata` form): exit 0.
- `GOOS=linux/windows go build ./...`: ok. JSON check: ok. `git diff --check`:
  clean. `task-board validate`: exit 0 (191 pre-existing board-hygiene issues
  on other items, none mine).
- `task-board.config.json`: untouched by this task. Vs `origin/main` it carries
  the z1yxg9 rpcwire fuzz gate (expected) plus trunk drift from `c5582bc`
  (cataloggen `-adopted` form); the worktree keeps its consistent older form.

## Bounds and notes

- Stated bounds (matrix §D): native Tailscale SSH, non-host lanes, idle
  self-close (measured), 11.10.5 policy reader (not implemented by design),
  no `ax` CLI, unreachable `LaunchForConfig` nil-binding branch (loader owns
  the gate), TLS alert text / `Dial`-side rendezvous messages (class pinned).
- No new source-text-inspecting gate was added; the DoD static-token clause is
  satisfied by prior work (census-evasion mutant re-run green).
- `task-board worktree refresh-candidate` replay against `origin/main e4e3e88`
  fails on a LOGBOOK.md rebase conflict (missing replay resolution) —
  orchestrator-side integration issue; worktree untouched at `244a7dc`.
- Candidate left UNCOMMITTED on `task-board/story/STORY-260830-1kiyj6`.

## Handoff evidence

- `TASK-260830-2x16gz_results.md` (this file)
- `TASK-260830-2x16gz_conformance-matrix.md` (row-by-row matrix, §A–F)
- `TASK-260830-2x16gz_producer-evidence.tar.gz` (real gzip: hostile logs, full
  suite/race/cover logs, vet/fuzz/tracecheck/cataloggen logs, mutant chunk
  outputs A–D with per-plant `test.log`/`results.json`/`table.md`, matrix)
