# TASK-260924-2zboyz — CR rev5 re-verification note (RUN-260924-24be9f, for the rev6 handoff)

## Why this run exists

CR rev5 construction failed validation at command 5/30
(`go test ./... -race -count=1 -timeout 25m`, exit -1: the
harness killed the run during the race suite, after
`tracecheck` passed at 540.9s and 10 race packages passed).
CR rev4 failed at command 4/30 (`go test ./... -count=1 -v`,
tracecheck 10m timeout under 49-way contention). Both
failures are environmental load shapes on heavyweight
pre-existing suites, not leaf defects: the rev4 and rev5
patches are byte-identical (sha256
`e0f422e67753c29fc0562f388a180dd9bb14674c473501efee00292621ce4fae`),
and no old package imports the leaf (verified per-package
below).

## Candidate identity (verified, not assumed)

- `git apply` of the rev5 patch onto a scratch tree followed by
  `diff -r` against the live worktree: no differences.
- Live worktree: `git status` shows only
  `?? internal/clonereadback/`; `git diff` and `git diff --cached`
  empty; real index untouched (all tree reads through
  `GIT_INDEX_FILE`).
- Scratch-index tree OID `cf13fcb5aba321d93e6fcd58418886da22fd69ef`,
  stable before and after both mutant batteries (proves harness
  restoration).
- Trunk unmoved at `6d3bff9` (`origin/main` identical) — no
  `refresh-candidate` run. No registry edit; `task_delta` scope
  unchanged. No `.pyc`/`__pycache__` anywhere in the worktree
  (plus committed `TestNoGeneratedBytecode` green).
- This run changed NO source bytes: the rev6 handoff carries the
  rev5-identical candidate answering the rev3 verdict's single
  finding `report-sibling-provenance-unbound` (sealed
  `ValidatedReadBack` siblings; report entries take ONLY sealed
  siblings and refuse zero first at both entries).

## What this run reran itself (exit codes observed)

| Gate | Command | Exit | Detail |
|---|---|---|---|
| leaf suite | `go test -count=1 ./internal/clonereadback/` | 0 | ok 4.940s; verbose 74 PASS / 0 FAIL |
| leaf cover | `go test -count=1 -cover ./internal/clonereadback/` | 0 | 95.6% |
| determinism | `go test -count=3 ./internal/clonereadback/` | 0 | ok 14.083s |
| leaf race | `go test -race -count=1 ./internal/clonereadback/` | 0 | ok 57.922s, race-clean |
| importer set | owner suites `environ scalar canonicaljson clonebundle sessadapter` | 0 | 5/5 ok |
| importer edge | per-old-package `go list -deps` grep | 0 | 0/48 old pkgs reach clonereadback |
| vet | `go vet ./...` | 0 | clean |
| windows vet | `GOOS=windows GOARCH=amd64 go vet ./...` | 0 | clean |
| gofmt | `gofmt -l internal/` | 0 | clean |
| battery run1 | harness 53 rows in 3 shards | 0 | 43 KILLED, 1 SURVIVED (control), 9 RED, failures=0 |
| battery run2 | harness 53 rows in 3 shards | 0 | 43 KILLED, 1 SURVIVED (control), 9 RED, failures=0 |
| shard A0/A1/A2 | 15 fast pkgs each | 0 | 15 ok × 3 |
| resumesmoke | alone | 0 | ok 200.069s |
| tmuxserver | alone | 0 | ok 151.245s |
| traceability | `traceability` + `tracecheck` | 0 | 2 ok, tracecheck 200.203s |
| independent JCS | own Python JCS+SHA-256 over 3 sealed docs | 0 | 3/3 match, sealed bytes canonical |

Full-suite total rerun by this run: 49/49 packages green,
every command exit 0, no contention timeout in any shard.

## Mutant audits (this run)

- Kill mechanisms audited from the raw logs (not the verdict
  lines): `N-seal-build` → `--- FAIL:
  TestBuildReportRefusesUnsealedSiblings`, subprocess exit 1,
  fall-through to the downstream distinctness literal (the
  documented N-closed precedent); `N-seal-decode` symmetric at
  `sealed_test.go:73`; `N-control-neutral` exit 0 SURVIVED.
- One raw log per row per run rides the tar (`run1/`, `run2/`,
  53 files each) plus the six shard summaries. No
  compile-error kills (named `--- FAIL` in every audited row).
- The full 43-narrowing × named-killer table is unchanged from
  the rev4 results doc (identical code) and is not retyped
  here; both batteries above re-prove every cell.

## Reuse statement

Rerun by this run: every gate in the table above (leaf,
importer set + edge, vet × 2, gofmt, both full mutant
batteries, full 49-package suite, independent JCS).
Reused from the rev4 evidence: results/conformance/TRACEABILITY
prose, the mutant-to-killer table, and the two withdrawn-row
logs (the candidate is byte-identical to the tree that
evidence was recorded on, verified above, so the binding
holds per the unchanged-identity rule).

## Implementation spot-check vs the rework brief (this run)

Read `sealed.go`, `authority.go`, `readback.go`, `report.go`,
`decode.go`, `sealed_test.go` in full: report entries take ONLY
`ValidatedReadBack` siblings, `checkSiblingsSealed` runs first
at both entries, read-back entries mint the seal, AST guards
(`TestReportEntriesTakeOnlySealedSiblings`,
`TestValidatedReadBackSealedConstruction`) plus zero-valued,
forged-sibling, and cross-plan-sibling regressions present
with the names the results doc claims, and the per-entry seal
narrowings kill alone (audited above). Coverage ratio restated
from the unchanged rev4 measurement: 6 of 6 AC rows driven,
18 of 18 counted gates carry a narrowing killed alone.
