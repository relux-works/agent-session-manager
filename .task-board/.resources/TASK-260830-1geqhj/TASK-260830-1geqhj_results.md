# TASK-260830-1geqhj results — ax pane enforcement wrapper

Status: ready for review (producer handoff; candidate UNCOMMITTED in the
Story worktree on `task-board/story/STORY-260830-ptxkqe`).

## Deliverable

`internal/axpane`: the validation core of the `ax pane SESSION_ID`
enforcement wrapper. The pure core `Decide` returns exactly one of
`launch|reattach|attach_remote|takeover_offer|parked(reason)|refused(class)`;
`Run` orchestrates the adapters over the real durable stores and applies
exactly the authorized effect; the bootstrap `(session_id,
bootstrap_operation_id)` store binds the one recorded wrapper/child under
the no-replace + fsync discipline with crash hooks; §5.2 events author
through `sessrepo.AppendEvent` under the current lease; the §7.A descriptor
builder supplies the provider binding context.

Every check composes a landed gate (`fencing`, `matjournal`,
`sessckpt`/`sessrepo`, `sessprofile`, `provhost`, `resumesmoke`,
`terminalbackend`, `config`, `axerror`); nothing is re-implemented.
`internal/traceability` is untouched per the Story contract (the final leaf
re-pins); clause coverage lives in `internal/axpane/TRACEABILITY.md` and the
conformance matrix.

## AC coverage: 41 of 41 rows driven

Every command, message, state, and refusal in the AC is driven through a
production entry by a named committed test. Ratio: **41 of 41**.
Production call sites per row are in the conformance matrix
(`TASK-260830-1geqhj_conformance-matrix.md`) and
`internal/axpane/TRACEABILITY.md`. Headline rows:

- Owner epoch / fencing: lower, losing, future, foreign, expired, malformed,
  absent, ambiguous, unverified, failed-handoff, remote (attach/takeover),
  mode-selected entry — via `Decide` → `fencing.AuthorizeActivation` /
  `AuthorizeRestore`.
- Materialization (required + committed) and checkpoint (required + attested
  + identity) — via `Decide` + `LoadMaterialization` / `LoadCheckpoint`.
- Effective profile + source, `profile_mapping_unavailable` — via `Decide` →
  `sessprofile.Derive` / `provhost.ResolveMapping`.
- Resume tuple, identity bind, discovery bind, smoke precondition with typed
  `target_auth_missing` — via `Decide` → `provhost.*` / `resumesmoke.*`.
- Backend identity (`Reconcile`), closed capabilities (`CheckOperation`),
  entrypoint, descriptor (`AdmitProviderDescriptor`) — via `Decide`.
- Background caller `capability_unavailable` with typed realm details;
  cached sentinel/`managername` structurally unauthorizing — via `Decide`.
- Bootstrap idempotency (same reattach, changed mismatch, commit race),
  status-proves-absence, real-kill crash seam, no-PID receipt — via
  `Store.Bind` / `Store.Status` / `Run`.
- `session.parked` (all four reasons, incl. offers), v4 `terminal.created` /
  `session.resumed` — via `EmitParked` / `Emit` / payload builders.
- After-restore steps 1-5, arm precedence, orchestrated effects — via
  `Decide` / `Run`.

## Validation (this worktree, this session)

- `go test ./internal/axpane -count=1` — exit 0 (full package suite).
- `go test ./internal/axpane -cover -count=1` — exit 0 (see coverage log).
- `go test ./... -count=1` — exit 0, 38 packages ok, no FAIL.
- `go test ./... -race -count=1` — exit 1 SOLELY on
  `internal/sessquery` hitting the default 10m `go test` timeout
  (37/38 packages ok, including `internal/axpane` in ~8s; no DATA
  RACE, no individual test failure anywhere). Reran in isolation:
  `go test ./internal/sessquery/ -race -count=1 -timeout 20m` —
  exit 0 in 552s. The suite is pre-existing slow under race (it
  needs ~9.2 minutes); my change cannot reach it (new package,
  zero importers; `sessquery` reads neither README nor LOGBOOK).
  Recorded honestly as a failing-as-configured gate with the
  isolation evidence attached (`sessquery-rerun.log`,
  `sessquery-long.log`).
- `go test ./internal/axpane/ -race -count=1` — exit 0.
- `go test ./internal/axpane/ -cover -count=1` — exit 0, 81.7%.
- `go test ./... -cover -count=1` — exit 0 (see cover log).
- `go build`, `go vet`, `gofmt` check, `GOOS=linux`/`windows` builds,
  JSON census, `git diff --check` — all exit 0.
- `tracecheck` — exit 0 (no registry change, as required for a first leaf).
- `cataloggen -adopted -output internal/catalog/catalog_gen.go -check` —
  exit 0.
- 14/14 fuzz gates (`-fuzztime=100x`) — exit 0.
- `task-board validate` — exit 0 (187 pre-existing MISSING_ACTIVITY
  warnings on other items, none on this task or story).
- Mutant harness: 25/25 rows match — 24 narrowing mutants KILLED plus the
  applied harmless SURVIVED control (`mutants.log` with per-plant raw logs
  and subprocess exits).

## Crash / idempotency evidence

- Real-kill test `TestBindCrashChildSelfTerminates` (SIGKILL after the
  no-replace commit, before directory sync): the complete binding survives
  and the identical retry reattaches — no torn prefix, no second child.
- Hook tests pin both boundaries (aborted stage → absence + converging
  retry; post-commit hook error → commit stands + reattach).
- Commit-race tests pin linearizability (same-operation loser reattaches,
  changed-operation loser refuses `idempotency_mismatch`).
- Parked events append through the `sessrepo` owner (linkage + round-trip
  proven here; the owner's own crash evidence stays with that package).

## Files (all UNCOMMITTED, candidate snapshot)

- `internal/axpane/doc.go`, `decide.go`, `errors.go`, `binding.go`,
  `sync.go`, `events.go`, `descriptor.go`, `gates.go`, `run.go`
- `internal/axpane/fixtures_test.go`, `decide_test.go`, `binding_test.go`,
  `crash_unix_test.go`, `run_test.go`, `events_test.go`,
  `descriptor_test.go`, `conformance_test.go`, `mutant_harness.py`,
  `TRACEABILITY.md`
- `README.md` (new package section, no capability/CLI claim),
  `LOGBOOK.md` (newest-first entry)

## Stated bounds (no unsupported claims)

No `ax` CLI, no tmux/ConPTY process control, no provider supervision, no
mesh transport, no broker. The step-2 refresh is the caller-reported
`Verified` fact; signatures are caller-supplied; backend liveness and
backend operations belong to the later leaf; the host-local binding arrives
from the modeled backend caller; committed-phase and step-5 park readings
are stated; unknown SESSION_ID refuses `invalid_arguments` (no minted
code). Full list in TRACEABILITY.md.

## For the reviewer

Suggested entry: `internal/axpane/TRACEABILITY.md` (41-row matrix + 25-row
mutant table + bounds), then `decide.go`, then the evidence tarball
(`mutants.log`, full-suite logs, race/cover logs, fuzz logs).
