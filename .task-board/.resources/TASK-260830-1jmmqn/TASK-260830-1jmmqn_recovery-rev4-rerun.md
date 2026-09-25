# TASK-260830-1jmmqn recovery rerun (RUN-260924-2f8e60)

Rev4 CR validation failed in `tracecheck` (`FAIL ... 600.657s`, suite
stopped at command 6 of 30) — the same concurrent-race-suite 10m
timeout shape that sank rev1/rev2 and was resolved by re-handoff as
rev3. No code finding: the candidate is byte-identical to the rev4
patch (all 25 `internal/cloneplan/` blobs hash-match; `plans.go`
still `sha256:c2f7eb93506291635218beb49a589d8920bf32459d2906d73c9c3c74e311f02f`,
the reviewer's pinned blob). Trunk is still `0ca3e4c`; no refresh
needed. This run re-verified the rev4 candidate with its own
instruments and re-hands it off.

## Reran myself (real exit codes, raw logs in the tar)

| Command | Exit | Evidence |
|---|---|---|
| `go test -count=1 ./internal/cloneplan/` | 0 (51s) | package run observed inline |
| determinism `-count=3` on the 6 new constant tests + `TestPlanConstants` | 0 (4.5s) | observed inline |
| `go vet ./internal/cloneplan/` | 0 | observed inline |
| `gofmt -l internal/cloneplan/` | 0, no files | observed inline |
| `GOOS=windows GOARCH=amd64 go vet ./...` | 0 | `raw/winvet-recover.log` |
| scratch-index `git diff --cached --check` incl. untracked | 0 | observed inline |
| `task-board.config.json` vs HEAD | byte-identical (`cmp`) | observed inline |
| 6 owner suites on candidate | 0 (all `ok`) | observed inline |
| full suite shard 1 (23 pkgs) | 0, 23 `ok` | `raw/shard1.log` |
| full suite shard 2 (23 pkgs, incl. tracecheck 84s) | 0, 23 `ok` | `raw/shard2.log` |
| 20 new mutant rows (`N-copy-*`, `N-null-*`) | harness exit 0, 20/20 KILLED | `raw/mut-recover/` (one log per plant) |
| build-side `copy` narrowing, killer alone | go test exit 1 | `raw/copy-build-kill.log` |
| decode-side `copy` narrowing, killer alone | go test exit 1 | `raw/copy-decode-kill.log` |
| guard on build-side `copy` widening | go test exit 1, gate named | `raw/guard-copy-redden.log` |

Kill attribution (isolated, one plant each):
- Build side: `TestPlanConstantCopyRegression` fails on the literal
  `materialization_intent is not clone` refusal.
- Decode side: the narrowed doc passes the intent gate
  (`decodeTransactionPlan`, plan.go:727) and is refused downstream
  by `verifySelfDigest` (plan.go:784) with the identity-mismatch
  token; the test pins the intent literal, so it reddens. Honest
  statement: the decode kill is observed as a refusal-token change,
  not as the intent literal itself.
- Guard: `TestClosedConstantGateShape` reddens naming
  `buildTransactionPlan build gate for MaterializationIntent`.
- Every plant was restored byte-identical (`plans.go` sha
  re-verified after each); one earlier control-plant attempt used a
  shell-mangled patch (build failure, discarded and redone — the
  committed log is the exact-bytes plant).

## Accepted from rev4 evidence (unchanged identity)

- Pre-existing 125-row battery x2 runs, coverage 96.8%, and
  base-vs-candidate owner byte-identity (171 files): source, tests,
  config, and environment identity unchanged since rev4 (blob
  comparison above), so these are reused, not replayed. The 20 new
  rows were rerun above.

## Finding status

`plan-constant-second-value-uncounted` stays answered: both
independent `copy` narrowings are killed by the named regression
alone, the guard reddens on the widening, and the generated sweep
is green and deterministic. Measured coverage remains 18 of 18
leaf-schema rows per the rev4 results; no production line changed.
Candidate left uncommitted (25 paths under `internal/cloneplan/`
only); real index untouched.
