# TASK-260830-2zvo8m rev2 results — CR1 rework (P1-1, P1-2 trunk restorations)

Rework run after the CR1 changes-requested verdict
(`TASK-260830-2zvo8m_review-verdict-rev1.md`, RUN-260917-3ff440).
Product code was fully verified at rev1; rev2 changes **only**
`LOGBOOK.md` and `README.md`, restoring two trunk hunks the base
refresh had carried stale pre-trunk copies over. No product-code
change; no new behavior; rev1 test/mutant/traceability evidence
stands and every gate below re-ran green on the rev2 tree.

Story tip unchanged: `54ed1b3` on trunk `e4e3e88`. Candidate left
UNCOMMITTED in `.temp/STORY-260830-315721/worktree`.

## What changed since rev1

### P1-1 — LOGBOOK.md: 3lt2xv trunk entry restored

Re-inserted the 5-line block from `git show HEAD:LOGBOOK.md`
(lines 8–12) between the 2zvo8m entry and the 3uzfyn entry.
Newest-first order preserved: 2zvo8m, 3lt2xv, 3uzfyn, kp4zpu, ….

Restored hunk (byte-identical to HEAD, verified by script):

```text
### TASK-260917-3lt2xv — release-agnostic cataloggen check via -adopted
- CONTRACT: validation command 21 was pinned to explicit v0.6.0 inputs, …
- GATE: command 21 is now `go run ./internal/catalog/cmd/cataloggen -adopted -output internal/catalog/catalog_gen.go -check`; …
- TESTS: adopted positives, byte-identity …, narrowing mutants …, harmless control SURVIVED. …
- DOCS: README regenerate block and tool row document the `-adopted` form and the explicit equivalent. …
```

(`…` abbreviates here only; the file carries the full HEAD lines.)
`git diff HEAD -- LOGBOOK.md` is now purely additive: 51 insertions,
0 deletions.

### P1-2 — README.md: both `-adopted` hunks restored

1. Regenerate block — restored the `-adopted` check line, the
   derivation paragraph, and the explicit-equivalent block, keeping
   the Native-resume smoke section and figure updates:

```bash
go generate ./internal/catalog
go run ./internal/catalog/cmd/cataloggen -adopted -output internal/catalog/catalog_gen.go -check
go test ./internal/catalog ./internal/cataloggen ./internal/catalog/cmd/cataloggen -count=1
```

```text
The `-adopted` form derives the metadata and lock inputs from the code-level
adopted release, so the check stays valid when the adopted release changes.
The explicit equivalent pins the same inputs by path:
```

```bash
go run ./internal/catalog/cmd/cataloggen -metadata internal/catalog/catalog.v0.6.0.json -contracts internal/specpin/v0.6.0.lock.json -output internal/catalog/catalog_gen.go -check
```

2. Go-toolchain table row — restored the `-adopted` command with the
   explicit-selector note:

```text
`go run ./internal/catalog/cmd/cataloggen -adopted -output internal/catalog/catalog_gen.go -check` (`-metadata`/`-contracts` select the same inputs explicitly);
```

Both restored segments verified byte-identical to HEAD (regen
section, tool row, and LOGBOOK block compared as slices: all True;
`-adopted` count 3 = 3; `TASK-260917-3lt2xv` count 1 = 1).

`git diff HEAD -- README.md`: 79 insertions, 13 deletions. Every
deleted line is an intentional traceability-figure rewrite for this
leaf (113 cases, 29/463 clauses, full=2 partial=4 sliver=3,
unevidenced 48→44, admitted bindings 1→2 with the Section 2.4
sentences) — none touches cataloggen/`-adopted` content. Full list
in `rev2-readme-deleted-lines.txt` inside the evidence tarball.

Restoration method: targeted insertion of exact HEAD bytes via a
throwaway script (`/tmp/restore_trunk_hunks.py`, kept in the
tarball); neither file was wholesale-checked-out.

## Stale-copy audit (trunk 62d4463 → e4e3e88 vs candidate)

Trunk changed (non-board paths): `LOGBOOK.md`, `README.md`,
`internal/catalog/catalog.go`, `internal/catalog/catalog_test.go`,
`internal/catalog/cmd/cataloggen/main.go`,
`internal/catalog/cmd/cataloggen/main_test.go`,
`task-board.config.json`.

- Candidate tracked modifications: `LOGBOOK.md`, `README.md`
  (both fixed above), `internal/provhost/identity.go`,
  `internal/provhost/probe.go`,
  `internal/traceability/cmd/tracecheck/main_test.go`,
  `internal/traceability/ownership.v0.6.0.json`,
  `internal/traceability/traceability.go`,
  `internal/traceability/traceability_test.go` — the
  provhost/traceability paths are disjoint from the trunk list, so
  they cannot revert trunk content.
- `git diff HEAD --name-only -- internal/catalog
  task-board.config.json` → empty (0 paths); `task-board.config.json`
  shows no diff.
- Untracked candidate paths (`internal/provhost/accessors_test.go`,
  `internal/resumesmoke/`) are new files; they revert nothing.
- `.task-board/` paths in the trunk range are board artifacts, not
  worktree scope; the candidate touches none.

## Validation (all observed this session on the rev2 tree)

26-command suite = the 18 README Go-toolchain commands plus 8 CI
extras; every command exit 0 (fuzz legs additionally require the
`^fuzz: elapsed` execution proof):

| # | Command | Result |
| - | ------- | ------ |
| 1 | `go run ./internal/traceability/cmd/tracecheck` | exit 0 — 113 cases, 29/463 |
| 2 | `tracecheck -section 6.2` | exit 0 (admitted) |
| 3–5 | `go test ./internal/config ./internal/localstore ./internal/scalar -cover -count=1` | exit 0 (via ./... legs) |
| 6 | scalar fuzz `FuzzScalarProductionEntries` 100x | elapsed ✓ |
| 7–9 | `go test ./internal/canonicaljson ./internal/axerror ./internal/cliresult -cover -count=1` | exit 0 (via ./... legs) |
| 10–13 | canonicaljson 4 fuzz targets 100x | elapsed ✓ each |
| 14 | `go generate ./internal/catalog` + `git diff --exit-code` on `catalog_gen.go` | clean |
| 15 | `cataloggen -adopted -output internal/catalog/catalog_gen.go -check` | exit 0 |
| 16 | `go test ./... -v -count=1` | exit 0 — 28 ok, 0 FAIL, 1 top-level SKIP (`TestDumpSweepSites`, skips by design when `AX_SWEEP_DUMP` is unset; canonicaljson harness hook outside candidate scope) |
| 17 | `go test ./... -cover -count=1` | exit 0 — 28 coverage lines, resumesmoke 77.0% |
| 18 | `go build ./...` | exit 0 |
| 19 | `gofmt -l internal` | clean (empty) |
| 20 | `go vet ./...` | exit 0 |
| 21 | `GOOS=windows go vet ./...` | exit 0 |
| 22 | `GOOS=linux go build ./...` | exit 0 |
| 23 | `GOOS=windows go build ./...` | exit 0 |
| 24 | JSON validity over `git ls-files '*.json'` | pass |
| 25 | `git diff --check` | clean |
| 26 | `go test -race ./internal/resumesmoke ./internal/provhost ./internal/sessprofile` | exit 0 |

Beyond the 26, also green this session:

- Focused `go test ./internal/resumesmoke ./internal/provhost
  ./internal/sessprofile ./internal/traceability/... -count=1` —
  exit 0 (41.4 s / 8.8 s / 5.8 s / 2.9 s / 48.5 s).
- `go test ./... -count=1` (non-verbose) — exit 0, 28 ok, 0 FAIL.
- cigate contract gates (6 tests) — exit 0.
- cigate README-claim gates incl.
  `TestRealREADMECarriesNoPositiveClaim` (all probes unavailable)
  — exit 0, so the restored README carries no positive capability
  claim.
- `tracecheck -section 2.4` — exit 0 (admitted).
- secconftest fuzz targets (5 of the 13 total; 13/13 with elapsed
  lines, target list derived from `-list`, count pin 13 holds).
- Native-Windows bind rows skip loudly with the exact reason
  (`windows native-store bind requires a Windows host; cell … is
  pinned by TestResumeMatrixCoversEverySpecRow`).

## Coverage and hygiene

- AC coverage: 16 of 16 rows driven through production entries —
  reviewer-confirmed at rev1; rev2 touches no production code, and
  the focused + full suites re-ran green on the rev2 tree.
- `git status` shows only candidate paths (8 modified + 2 untracked
  entries); no `__pycache__`, no `.pyc`; candidate UNCOMMITTED; no
  commit made on the Story branch.
- Conformance matrix unchanged from rev1; re-attached under the
  rev2 name per the rework brief.

## Attached

- `TASK-260830-2zvo8m_results-rev2.md` (this file)
- `TASK-260830-2zvo8m_conformance-matrix-rev2.md`
- `TASK-260830-2zvo8m_producer-evidence-rev2.tar.gz` (real gzip:
  rev2 validation logs, diffs, restoration script, byte-identity
  proof)
