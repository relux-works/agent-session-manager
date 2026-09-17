# TASK-260830-17ootk results — fault-test-local-crash-boundaries (story-final)

## Outcome

Ready for review. The Section 13.13 crash/restart outcome gate ships as an
executable conformance harness (`internal/crashgate`) over the landed
`sessckpt` + `matjournal` owners. No product behavior changed; no owner
source touched (mutant targets restored; `git status` shows only the
candidate paths below).

## Coverage ratios (production call sites in parentheses)

- AC rows driven: **4 of 4** (`sessckpt.Store.Capture`,
  `matjournal.Store.Create`/updates/`matjournal.Store.Recover`).
- Reachable boundary paths driven: **48 of 48** (31 of 94 IDs), each with a
  crash, a clean restart, exactly one outcome, and a complete conformance
  record. 72 table rows + 2 real-SIGKILL rows.
- NOT APPLICABLE paths recorded with exact owners, none driven: **121 of
  121** (63 of 94 IDs).
- Section 13.13 clauses discharged in the registry: **9 of 11** (`partial`);
  `13.13#4/#5` journal-halves only, disclosed in the binding gap.
- Section 13.12 rows driven: **7 of 19** (operator interrupt, owner-resume
  lost, rename blocked x2 proofs, disk full x2 proofs, import/open/adopt
  fail before lease x2 probes, adopt fails after lease); the other 12 name
  their unlanded owners in the matrix.
- Rejection cases refused as any successful outcome: **10 of 10** (two live
  authorities, unfenced continuation x2, new-session launch, fresh handle,
  blank relabel, different realm, moved lease x3) — all parked with durable
  blocking evidence, frozen phase, and no second allocation.
- Gating negative tests with narrowing mutants: **9 of 9 KILLED**, each by
  its named behavioral killer; 1 harmless control SURVIVED. The
  token-preserving registry mutant passes static derivation and fails the
  behavioral suite.

## What was built

- `internal/crashgate/registry.go` — classified boundary table (94 IDs,
  per-path drivers/owners/notes) + lookup accessors. Pure data; no I/O.
- `internal/crashgate/*_test.go` — derivation test (ID set parsed from the
  pinned v0.6.0 text, zero hand-typed IDs), 72-row conformance table with
  per-row JSON records, rejections, mutual-exclusion/exhaustiveness,
  Section 13.12 rows, real-SIGKILL seams (unix).
- `internal/crashgate/mutant_harness.py` — 9 narrowing + 1 control; runs
  the behavioral suite per plant.
- `internal/crashgate/TRACEABILITY.md` — clause/boundary/seam matrix.
- Registry: `section:13.13` unevidenced→partial (9/11),
  `section:13.12` unmeasured with harness case, 2 new acceptance cases
  (`crash-gate-journal-conformance`, `crash-gate-checkpoint-conformance`),
  `reviewedOwnershipCanonicalSHA256` re-pinned to
  `a1b48479025ce73cefdde8771879d6b3cdb93c0e4f1ec4051f4b35b79f357fac`;
  pinned report/CLI-output tests and README figures/coverage refreshed.
- README package section; LOGBOOK entry (newest-first).

## Validation (all exit 0; logs in the evidence tarball)

| Gate | Command | Result |
| --- | --- | --- |
| tracecheck | `go run ./internal/traceability/cmd/tracecheck` | ok, 107 cases, 26/489 clauses |
| catalog | `go generate` + `cataloggen -adopted -output ... -check` (post-refresh trunk form) | clean |
| contracts | cigate 6 gates | 6 PASS |
| claims | cigate README 6 gates | 6 PASS |
| vet | `go vet ./...`, `GOOS=windows go vet ./...` | clean |
| format | `gofmt -l internal` (437 files) | clean |
| build | `go build ./...`, `GOOS=windows go build ./...` | ok |
| test | `go test ./... -count=1` | 29 ok, 0 fail, 0 skip |
| cover | `go test ./... -cover -count=1` | 29 ok (crashgate 36.8%, matjournal 77.0%, sessckpt 82.1%) |
| race | `go test -race` all 29 packages (4 chunks) | 29 ok, 0 fail |
| fixtures | secprim + secconftest `-v` | 51 + 59 PASS, 0 skip |
| fuzz | 13 derived targets at 100x | 13 elapsed, 0 fail |
| mutants | crashgate harness | 9 KILLED, 1 SURVIVED, exit 0 |

## Candidate paths (`git status`)

- `M README.md`, `M LOGBOOK.md`
- `M internal/traceability/ownership.v0.6.0.json`
- `M internal/traceability/traceability.go` (re-pin)
- `M internal/traceability/traceability_test.go` (report counts)
- `M internal/traceability/cmd/tracecheck/main_test.go` (CLI output)
- `?? internal/crashgate/` (registry, tests, harness, TRACEABILITY.md)

Left UNCOMMITTED in the Story worktree for handoff snapshot, per the
managed-worktree contract. `.temp/` evidence is git-ignored and ships via
board resources + the tarball below.

## Evidence attached

- `TASK-260830-17ootk_results.md` (this file)
- `TASK-260830-17ootk_conformance-matrix.md` (clause + 94-ID matrix)
- `TASK-260830-17ootk_producer-evidence.tar.gz` (74 records, mutant logs +
  summary, suite/race/cover/cigate/fuzz logs)

## Base refresh during the run

Trunk moved while the candidate was open (release-agnostic catalog gate,
`e4e3e88`). `task-board worktree refresh-candidate` replayed the Story
branch with two checkpoint-bound LOGBOOK resolutions (newest-first order
kept); stale trunk paths (`internal/catalog/*`, `task-board.config.json`,
`.task-board/`) restored via `checkout HEAD`, LOGBOOK entry re-prepended,
trunk README hunks merged. Post-refresh re-validation (new `-adopted`
catalog form, tracecheck, full suite) is green; pre-refresh evidence logs
remain valid (tested code byte-identical).

## Stated bounds (see TRACEABILITY.md for all six)

Probes stay modeled; marker completion bytes stay `matjournal`-owned;
clone finalization prefixes unlanded; rollback/parked cross-surface
visibility unlanded; full hook matrix at STOP-02/MAT-01/02 with
representative seams elsewhere; epoch-1 abort classification with the
lifecycle story.
