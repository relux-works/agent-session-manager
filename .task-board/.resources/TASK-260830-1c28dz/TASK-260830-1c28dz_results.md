# TASK-260830-1c28dz rev10 results — implement-tmux-lifecycle-operations

Revision 10 closes all three verdict-rev9 findings: one P1
fail-open census arm (peer receipt misfiled under another
client's key), and two P2 gaps (post-escalation observation
budget, complete-importer outcome evidence). The candidate is
UNCOMMITTED in the Story worktree against checkpoint
`d4bd91d0`.

## 1. Finding dispositions

| Finding | Severity | Disposition | Evidence |
|---|---|---|---|
| P1 peer census turns filename-mismatched identity evidence into "no peers" | P1 | FIXED: `Peers` admits a receipt-namespace entry only as a valid peer receipt for its own durable key — the filename stem must parse as a client identity and the keyed `Lookup` of that stem must return the receipt — before the requesting-client exclusion applies; session/instance checks, staging/non-receipt rules, and the directory refusal are retained. The attach refuses with no receipt, no argv, no authorized input | `TestAttachPeerFilenameMismatchFailsClosed` (entry, keyed-binding refusal + no receipt/vector/authorization) + `TestAttachStorePeersRejectsFilenameMismatch` (census, mismatch + malformed-stem legs); narrowings `N-attach-peers-filename-skip` (tmuxserver, cross-package plant) and `N-attach-peers-filename` (termbind), both KILLED |
| P2-A post-escalation closure probe inherits the expired graceful deadline | P2 | FIXED: the graceful wait budget and the post-escalation observation budget are distinct — the pre-kill poll keeps the graceful lesser-bound, the post-kill re-confirmation observes under the operation deadline. Fresh authorization, generation, and operation-deadline rechecks before the kill are kept; a timeout still concludes unknown, never closure | `TestExecuteStopEscalationSucceedsWithContextHonoringRunner` (context-honoring success witness beside the hung-probe refusal witnesses); narrowing `N-stop-escalation-budget` KILLED |
| P2-B complete-importer outcome evidence regressed to a name-keyed suite grid | P2 | FIXED: the revision-5 outcome-keyed corpus is restored byte-identically (`main.go`/`extra_off.go` unchanged; base output byte-identical to the rev5 base run) and extended with the leaf-new `DirectivesFor` table and `Peers` census as declared candidate-only keys, over the complete agreed 36-package closure | `closuregrid` driver in the tarball; `closure-base.jsonl` (193 keys) vs `closure-candidate.jsonl` (215 keys): 193 shared, 0 moved, 0 base-only, 22 declared candidate-only (§7). Suite-verdict grid kept as supplementary only |

## 2. Runner-wait audit (P2-A)

Every `Runner.Run` call site in the leaf, dispositioned through the
same shared contract (a read-only wait/probe carries a
deadline-derived context; expiry concludes unknown, never a
verdict). Revision 10 splits the stop poll row in two:

| Site | Class | Disposition |
|---|---|---|
| `backend.go` `pollSession` has-session, pre-kill (stop) | read-only poll | BOUNDED by the graceful wait (lesser of request deadline and graceful timeout) via `probeContext`; answered probes follow the clock exactly, only never-answered probes conclude by the context. Test + shared row (rev9, kept) |
| `backend.go` `pollSession` has-session, post-escalation (stop) | read-only observation | BOUNDED by the operation deadline via `probeContext`, selected by `escalationDeadline`; a cancellation-honouring executor observes the closure. New success test + new row (§1); the unknown-on-timeout arms are unchanged |
| `backend.go` `pollSession` has-session (terminate tail) | read-only poll | BOUNDED by the op deadline via `probeContext` (unchanged; terminate sets deadline to the operation deadline) |
| `backend.go` `terminateStale` confirm has-session | read-only probe | BOUNDED by the op deadline via `probeContext` (unchanged) |
| `status.go` `ObserveStatus` list-panes / list-sessions | read-only probes | BOUNDED by the query deadline via `probeContext` (unchanged) |
| `backend.go` `observeBoundary` wait-for | blocking wait | Already bounded (wait context); unchanged |
| `boundaryProviderRows` list-panes | read-only probe | Already bounded (runs under the wait context); unchanged |
| `ServerAdmission` / barrier acquire | blocking waits | Already bounded (uniform wait contract, rev8); unchanged |
| `UnixDialer` dial | blocking dial | Already bounded (1s timeout); unchanged |
| `closeInput` lock/detach, `execEvidence` send-keys/new-session, escalation kill, terminate kill | single-shot commits | Caller-cancellation-bounded; stated bound B45 (unchanged) |
| `ServerSpawner.Spawn` (bind step, `context.Background`) | commit on the acquire path | Caller-cancellation does not reach it; stated bound B45 with the interface change owned by TASK-260830-35urbp (unchanged) |

## 3. New tests and narrowings

Production call sites: `Peers` (`internal/termbind/overlap.go`),
`checkAttachOverlap` (`internal/tmuxserver/ops.go`),
`confirmClosed` + `pollSession` (`internal/tmuxserver/backend.go`),
`executeEngineOp` (`internal/tmuxserver/lifecycle.go`).

| Test (all through `Lifecycle.Execute` unless noted) | Finding | Narrowing | Verdict |
|---|---|---|---|
| `TestAttachPeerFilenameMismatchFailsClosed` (keyed-binding refusal; no receipt, no argv, no input) | P1 | `N-attach-peers-filename-skip` | KILLED |
| `TestAttachStorePeersRejectsFilenameMismatch` (termbind census direct; mismatch + malformed-stem legs) | P1 | `N-attach-peers-filename` | KILLED |
| `TestExecuteStopEscalationSucceedsWithContextHonoringRunner` (escalation success with a ctx-honoring runner) | P2-A | `N-stop-escalation-budget` | KILLED |

No `tmux` process anywhere in any test (fake runners + a
context-honoring decorator + the release-controlled blocking
runner); zero real-tmux witnesses by the determinism design
(bound B1, carried). The restricted-PATH changed-package run
(§8) re-proves it.

## 4. Isolated kill attribution (P2-D carried, rerun on rev10)

Each shared plant applied once, each claimed killer run ALONE
(anchored `-run`), two rounds on the rev10 tree. Raw logs:
`isolated/round{1,2}/<plant>.<test>.log`.

| Plant | Killer alone | Round 1 | Round 2 | Precision in the same run |
|---|---|---|---|---|
| `N-stop-escalation-generation` | stop stale-facts | KILLED | KILLED | other-fact subtests green |
| `N-stop-escalation-generation` | quiesce stale-facts | KILLED | KILLED | other-fact subtests green |
| `N-stop-escalation-expiry` | stop stale-facts | KILLED | KILLED | other-fact subtests green |
| `N-stop-escalation-expiry` | quiesce stale-facts | KILLED | KILLED | authorization subtest reddens, generation/deadline green |
| `N-stop-escalation-deadline` | stop stale-facts | KILLED | KILLED | deadline-instant reddens, strictly-past green |
| `N-stop-escalation-deadline` | quiesce stale-facts | KILLED | KILLED | deadline-instant reddens, strictly-past green |
| `N-attach-wait-deadline` | attach wait | KILLED | KILLED | admission + barrier subtests both redden |
| `N-attach-wait-deadline` | quiesce-span wait | KILLED | KILLED | single test |
| `N-attach-wait-deadline` | boundary-commit wait | KILLED | KILLED | admitted wait commits, timeout assertion reddens |
| `N-attach-wait-deadline` | attach cancel (companion) | GREEN | GREEN | cancellation unaffected by the widening |
| `N-probe-deadline` | stop poll deadline | KILLED | KILLED | verdict lands at the release, not the bound |
| `N-probe-deadline` | terminate confirm deadline | KILLED | KILLED | verdict lands at the release, not the bound |
| `N-probe-deadline` | status probe deadline | KILLED | KILLED | verdict lands at the release, not the bound |


## 5. Census and battery

- Table D: 149 gates × 21 entries = 3129 cells; 207 M / 187 B /
  2735 U, verified by script over the table (`count_census.py` in
  the tarball). New row: `escalationbound` (post-escalation
  observation budget; M at EXE, B39 at PFX); the `peershape` EXA
  cell gains the filename arm beside the directory arm, each
  narrowing killing through its own entry test.
- Table C recounted from the Table D gate set: 149 × 11 = 1639
  cells (1621 U2a + 18 U2c).
- Totals: 267 measured of 5536 (50+10+207 M across Tables
  A/B/D); 191 bound; 5078 unreachable with reasons. Bound-cell
  split verified by ID: 27 driven-but-rowless
  (B22/B23/B32/B36) + 160 named undriven gaps
  (B24/B25/B26/B28/B31 + 153 B39).
- Battery: 298/298 tmuxserver rows as expected (291 narrowing
  KILLED + 5 supplementary KILLED + 1 shadowed SURVIVED + 1
  control SURVIVED) on two full passes with one raw log per
  plant per pass; 43/43 termbind rows as expected (41 narrowing
  KILLED + 1 supplementary + 1 control) on two full verbose
  passes. Combined 341/341.
- Bounds: no new bound numbers; B44 clarified by construction
  (the census fails closed on identity corruption; liveness
  stays with TASK-260922-vcx6yo); B45 unchanged. The
  conformance matrix mirrors Table B/D byte-identically, AC rows
  59-92 in both copies, and bounds B22-B45 (mirror check in the
  tarball).

## 6. Operation coverage (8 of 8)

Each operation driven through `Lifecycle.Execute` by a named
committed test (call site `Execute`, `lifecycle.go`):

| Operation | Witness | Second witness (rev10) |
|---|---|---|
| create | `TestExecuteCreateInteractive` | — |
| attach | `TestExecuteAttach` | `TestAttachPeerFilenameMismatchFailsClosed`, `TestAttachSameClientRetryWithPeerPresent`, `TestAttachPeerDirectoryFailsClosed` |
| status | `TestExecuteStatusPresent` | `TestExecuteStatusProbeHonorsOperationDeadline` |
| quiesce | `TestExecuteQuiesce` | — (stale-facts legs) |
| safe-boundary | `TestExecuteBoundary` | — (commit-wait fixture) |
| stop | `TestExecuteStop` | `TestExecuteStopEscalationSucceedsWithContextHonoringRunner`, `TestExecuteStopPollHonorsOperationDeadline` |
| stale termination | `TestExecuteTerminate` | `TestExecuteTerminateConfirmHonorsOperationDeadline` |
| restore | `TestExecuteRestore` | — |

## 7. Composition (base vs candidate)

Base is the pristine checkpoint tree (`git archive d4bd91d0`);
candidate is the frozen rev10 worktree (base + the leaf patch +
the 41 untracked leaf files, verified byte-identical to the
worktree modulo ignored artifacts). The primary comparison is
outcome-keyed over the complete agreed importer set; the suite
comparison is supplementary regression evidence only, and is not
represented as an outcome grid.

- Entry-outcome corpus (fixed 193-key driver, compiled and run
  on both trees from a single source; `main.go` and
  `extra_off.go` byte-identical to revision 5, and the base run
  output byte-identical to the rev5 base run): **193 shared
  keys, 0 moved, 0 base-only, 22 declared candidate-only, 0
  fails on either side**. The 22 candidate-only keys are the
  leaf-new surface behind the `closuregrid_candidate` tag: 7
  revision-5 keys (`tmuxserver/CheckSocketLength` ×
  short/long, `tmuxserver/ClassifyStatus` × 5 input classes) +
  11 `tmuxserver/DirectivesFor` keys (8 admitted operations plus
  manifest, probe, and bogus refusals) + 4 `termbind/Peers`
  keys (empty-store admission plus bad-session, bad-instance,
  and bad-client refusals). No input moved across an admission
  or refusal arm — there is nothing to name.
- Closure: the agreed 36-package set from the rev9 independent
  import-graph traversal (`importer-audit.json`: 34 production
  + crashgate + environ), re-verified present in the current
  tree. The corpus drives 33 with production entries; 3 are
  stated bounds with owners (no exported pure callable
  surface; fenced by the supplementary suite grid below):
  cloneproject, crashgate, sshtransport. Zero corpus keys sit
  outside the agreed set. Finite-corpus limits, stated
  honestly: the corpus pins a fixed input class per entry, not
  the entry's whole domain; full-domain behavior stays with
  each package's own suite (supplementary grid) and, for the
  leaf's own entries, with the 263 committed AC witnesses.
  Out-of-closure packages (scalar and environ utility roots,
  catalog/cataloggen/specdoc/specpin tooling and pins,
  traceability registry, invcore test-only helpers) are
  untouched by the leaf delta — no leaf file imports a new
  package — and are fenced by the supplementary grid.
- Suite-verdict grid, supplementary (`go test ./... -count=1
  -json` on both trees, keyed by `(package, test)` top-level
  verdict; driver-free trees — the specdoc tripwire fails any
  suite run with the outcome driver present):
  2894 shared keys, 2894 retained, **0 moved**, 0 base-only,
  319 candidate-only, 0 fails on either side. All 319
  candidate-only tests sit in the leaf's own packages (308
  `internal/tmuxserver`, 11 `internal/termbind`); no other
  package gained, lost, or moved a test.
- Raw artifacts: `closure-base.jsonl`,
  `closure-candidate.jsonl`, `closure-diff.txt`,
  `closuregrid/main.go`, `closuregrid/extra_on.go`,
  `closuregrid/extra_off.go`, `base-suite.json`,
  `candidate-suite.json`, `suite_grid.py` (all in the tarball).

## 8. Validation

- Full 30-command configured suite: **30/30 passed, 0 failed**,
  firsthand on the final tree (`validation.log`, 56594 lines):
  gofmt clean, `go build ./...`, `go vet ./...`, full suite
  `-count=1 -v` (248s), race gate `-race -timeout 25m` (443s),
  cover gate (tmuxserver 89.0%, termbind 83.5%), 17 fuzz gates
  (100x each: rpcwire 4, scalar 1, canonicaljson 4,
  secconftest 8), tracecheck, cataloggen `-check`, linux +
  windows builds, JSON check, `task-board validate`,
  `git diff --check`. (The `FAIL` strings inside the log are
  nested fixture output of the resumesmoke negative tests; every
  package reports `ok` in the suite, race, and cover runs.)
- `GOOS=windows GOARCH=amd64 go vet ./...`: exit 0, test files
  included.
- Changed-package race: `go test ./internal/tmuxserver/
  ./internal/termbind/ -count=1 -race` green (20.5s + 3.3s).
- Restricted-PATH no-tmux suite: exit 0 with `tmux`
  unresolvable in `PATH`; `pgrep -x tmux` clean before and
  after; 461 top-level PASS, 0 fails (`no-tmux-run.log`). No
  acceptance row claims a real-tmux witness. Harness note: a
  TMPDIR-less environment breaks the landed dial test through a
  sun_path-length coupling (see the LOGBOOK anomaly); the proof
  runs with TMPDIR set.
- The composition snapshot was refreshed to the final tree
  (byte-identical modulo git-ignored artifacts) before the
  candidate composition runs; `git diff --check` (command 30/30)
  and the final `git status` (50 intended paths, HEAD still
  `d4bd91d0`, no commits) confirm the candidate tree.

## 9. Evidence manifest

- `TASK-260830-1c28dz_producer-evidence.tar.gz` (sha256
  `2ed1f346a338c6e02e3738631865c9b3643f326def16d8bacc6461ef435ca2ff`,
  3756830 bytes, 657 entries): isolated attribution logs (2
  rounds), tmuxserver harness pass 1 + pass 2 (per-plant raw
  logs), termbind harness pass 1 + pass 2 (verbose), full
  30-command validation log, outcome corpus (driver sources,
  base + candidate JSONL, diff), suite grids (base + candidate
  JSON), no-tmux run log, `count_census.py` + its outputs,
  `mirror_rev10.py` + its output, `isolate.py`, the validation
  runner + command list, and a per-file sha256 `MANIFEST.txt`
  (648/648 entries verify). Built from the final rev10 tree,
  attached by absolute path, read back, and digest-verified.
- `TASK-260830-1c28dz_conformance-matrix.md`: updated matrix
  (this revision).
- `TASK-260830-1c28dz_results.md`: this file.

## 10. Carried bounds and handover notes

- B18 (sun_path 104-byte limit), B16 (nested invocation ambient
  collision — deliberate ambient refusal), B19/B20 (SPEC 810-812
  before-bind unsafe-root rejection): unchanged from rev9, all
  measured in rows 59-77.
- Rev7 P3-A (fresh-decoy foreground attach wiring): unchanged;
  attach semantics here are the structured-command path, and
  the rev7 wiring question stays with the g0pcnt-side handover.
- B44 handover to TASK-260922-vcx6yo stays intact: this leaf
  distinguishes validated replay from new client by durable
  receipt and now fails the census closed on identity
  corruption; live-client overlap admission is theirs.
- The exec adapter (`BuildCommand` + `ExecRunner`) and the probe
  adapters (`UnixDialer`, `ServerSpawner`, `ServerProbe`) exist
  and are driven as in rev9; the server socket crash/idempotency
  evidence is carried (crash-converges rows, spawn rows).
