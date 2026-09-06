# TASK-260830-7a0s2c round 2 — review-verdict-rev1 rework (C1–C4)

Round-2 rework of the `internal/secconftest` instrument leaf. Round 1
is kept as history (`TASK-260830-7a0s2c_secconftest.md`); this document
supersedes its claims where the review proved them wrong. Leaf-1
`internal/secprim` production code is untouched
(`git diff HEAD --stat -- internal/secprim/` prints nothing).

Candidate tree: 15 files under `internal/secconftest/` (13 round-1 +
`skip_fifo_unix.go`, `skip_fifo_windows.go`). Fingerprint is the
sha256 of the sorted per-file sha256 manifest:
`5d64ba3bdd16db7121fef1eb8d12f278a7b402569ba667482087fad3f62502cd`.
Manifest re-taken after the battery is byte-identical
(`diff` of pre/post manifests empty). Base commit `722ed1a`.

Files changed vs round 1: `crash.go` (finalize gate,
`ErrRollbackPastFinalize`, `RemainingArmed`), `crash_test.go`
(finalize arm, commit-requires-prepare, consumption, vocabulary pin),
`skip.go` (`Report()` over live counters, real fifo probe),
`skip_fifo_unix.go` + `skip_fifo_windows.go` (new),
`skip_test.go` (fifo exchange + denial gating tests, counter test,
probe-honesty test), `runner_test.go` (report/name digest tests),
`hostile_test.go` (bidi pin, Gate wiring), `fuzz_test.go` (no-op
poles), `doc.go` (bounds), plus README + LOGBOOK corrections.

## AC coverage: 6 of 6 rows driven through production entry points

| # | AC row | Production call site | Named committed test |
|---|--------|----------------------|----------------------|
| 1 | deterministic fixture runners | `secconftest.Runner.Run` / `Digest` | `TestRunnerDigestIsDeterministic` (digest `5569f34caab734eb` x5 processes), `TestRunnerExecutesInSortedOrder`, `TestRunnerDigestDistinguishesReports`, `TestRunnerDigestDistinguishesNames` (digest witness is 3 of 3 components, was 1 of 3) |
| 2 | fake clocks | `secconftest.Fake.Now` / `Advance` / `Set` | `TestFakeClockIgnoresRealTime`, `TestRecordingNoticesUnwiredClock` |
| 3 | crash points | `secconftest.Injector.MaybeFail`, `Transactor.Prepare/Commit/Rollback` | `TestTransactorCrashOutcomes` (4 points + arm consumption), `TestTransactorRecoversAfterFault`, `TestTransactorRefusesRollbackPastFinalize` (commits, then rolls back: `ErrRollbackPastFinalize`, bytes intact), `TestTransactorCommitRequiresPrepare` (incl. planted staged file), `TestCrashPointVocabularyIsPinned` |
| 4 | protocol fuzzing | 8 entries (table below) | 8 `Fuzz*` targets + `TestFuzzDriversReachProduction` (22 arrivals/entry) + no-op poles in `FuzzIsEnvName` / `FuzzRedactCorpus` |
| 5 | hostile strings | `secprim.CheckMemberPath`, `Guard.Resolve/Open`, `DetectCaseCollision`, `CheckArgv/NewCommand`, `IsEnvName/BuildEnv`, `EscapeForTerminal`, `Redact` | hostile battery + `TestHostileGatesAreWired` (all 18 Gates pinned and driven) + `TestBidiRunesArePinned` (16/16 runes, corpus-covered) |
| 6 | platform capability skips | `secconftest.Require` / probes | `TestSkipDecideTable`, `TestSkipReportReadsRecordedVerdicts`, `TestSkipFifoFixtureExchangesBytes`, `TestSkipModeBitsAndNonRootDenialsHold`, `TestSkipFifoProbeAttemptsItsCapability` (4 of 4 probes wired, was 1 of 4) |

Negative/refusal rows added in round 2: post-commit rollback refused
with bytes intact; commit with staged bytes but no prepare refused;
report-only / name-only digest changes redden; bidi shrink reddens;
any Gate rename reddens; gutted verdict counter reddens; empty env
name admitted / well-formed refused reddens the env fuzz; passthrough
secret / rewritten innocuous line reddens the redact fuzz.

## Fuzz targets → entry points (arrival proven by seed run)

Unchanged from round 1: all 8 targets × 22 arrivals via
`TestFuzzDriversReachProduction`. New: `FuzzIsEnvName` refuses `""`
and admits `"AX_FOO"` on every input (an admit-all / refuse-all gate
fails); `FuzzRedactCorpus` requires `Redact("password=hunter2")` to
rewrite and `Redact("plain hello")` to pass through (an identity gate
fails). Interior grammar and per-class markers stay with the hostile
corpus tests named on each property (stated bound).

## Mutant battery: 29 applied, 27 killed, 2 survived with bounds

Denominator derived from this tree (battery
`/tmp/mutant-battery-round2.py`, log attached): applied 29,
NOT_APPLIED 0, COMPILE_FAIL 0, VET_FAIL 0 (every mutant vetted
clean), KILLED 27, SURVIVED 2. Split: narrowing 17 (15 killed, 2
survived), arm-deletion 8 (8 killed), census-only 4 (4 killed).
N10 (token-preserving section-drop) ran the full behavioral suite as
its harness. One harness-broken A1 attempt (unused variable,
COMPILE_FAIL) was repaired to a compiling deletion and re-run killed;
the stale line was removed from the log, noted here.

| Mutant | What it narrows the gate to | Named failing test |
|--------|-----------------------------|--------------------|
| N1 narrowing: Digest drops Report | report-class unwitnessed | TestRunnerDigestDistinguishesReports |
| N2 narrowing: Digest drops Name | name-class unwitnessed | TestRunnerDigestDistinguishesNames |
| N3 narrowing: Digest counts-only | name+report unwitnessed | TestRunnerDigestDistinguishesReports + Names |
| N4 narrowing: bidi drops U+FEFF | 15-rune vocabulary admitted | TestBidiRunesArePinned |
| N5 narrowing: bidi cut 16→1 | 1-rune vocabulary admitted | TestBidiRunesArePinned |
| N6 narrowing: path-traversal Gate→Redact | wrong gate decides the class | TestHostileGatesAreWired |
| N7 narrowing: CommitApply→explicit_rollback | one wrong outcome admitted | TestCrashClassifyTable |
| N8 narrowing: MaybeFail keeps the arm | every arm fires forever | TestCrashPointFiresOnceWhenArmed |
| N10 narrowing, token-preserving: section check dropped | wrong-section quote admitted | TestQuoteInSectionRefusals (full suite harness) |
| N11 narrowing: finalize gate + dead conjunct | post-commit rollback admitted | TestTransactorRefusesRollbackPastFinalize |
| N12 narrowing: RemainingArmed constant-empty | unreached arms invisible | TestInjectorArmedPointIsConsumedByTheDrive |
| N13 narrowing: Classify admits CR-MAT-01 | registry name accepted | TestCrashPointVocabularyIsPinned |
| N14 narrowing: fifo probe true-on-failure | broken build reported ok | TestSkipFifoProbeAttemptsItsCapability |
| N15 narrowing: DriveEnvName admits all | empty name admitted | FuzzIsEnvName (seed run) |
| N16 narrowing: DriveRedact stops delegating | — SURVIVOR, bound below | (suite green) |
| N17 narrowing: Fake returns time.Now() | wall clock leaks in | TestFakeClockIgnoresRealTime (+ StartsPinned) |
| A1 arm-deletion: finalize gate deleted | post-commit rollback nils and deletes | TestTransactorRefusesRollbackPastFinalize |
| A2 arm-deletion: Commit stops marking finalized | post-commit rollback admitted | TestTransactorRefusesRollbackPastFinalize |
| A3 arm-deletion: commit-without-prepare refusal deleted | staged-bytes-no-receipt commit admitted | TestTransactorCommitRequiresPrepare |
| A4 arm-deletion: recordVerdict gutted | verdicts never recorded | TestSkipReportReadsRecordedVerdicts |
| A5 arm-deletion: Run sort deleted | registration order leaks in | TestRunnerExecutesInSortedOrder |
| A6 arm-deletion: DriveEnvName stops counting | zero arrivals pass as coverage | TestFuzzDriversReachProduction |
| A7 arm-deletion: unknown-point error dropped | typo arms fire nothing, silently | TestCrashPointFiresOnceWhenArmed |
| A8 arm-deletion: idempotency refusal deleted | changed-body retry stages anew | TestTransactorIdempotentRetry |
| C1 census: roster loses path-device | 17-class roster passes | TestHostileRosterShape |
| C2 census: FuzzTargetEntry loses one | 7-entry map passes | TestFuzzDriversReachProduction |
| C3 census: traversal corpus loses one | 5-member class passes | TestHostileMemberCounts |
| C4 census: DefaultCapabilities loses nonroot | 3-probe list passes | TestSkipDefaultProbesTerminate |
| S1 narrowing: mode-bits probe false-on-denial | — SURVIVOR-AS-SKIP, bound below | (denial test SKIPs, suite green) |

Survivor bounds. N16: `DriveRedact` without delegation keeps
counting, and the redact fuzz properties call `secprim.Redact`
directly, so the suite stays green. Bound: driver delegation to
production is read-verified with arrival counts, not mutant-pinned;
stated in README and doc.go. S1: a non-strict probe reporting
unavailable skips by construction — the denial test emits a
`--- SKIP` line and `recordVerdict` logs it. Bound: the SKIP line is
the signal, not silence; CI must watch skip counts rather than
treating green as unskipped. S1 proves the mechanism works: the skip
was recorded and visible, exactly what C2 demanded.

## Review findings closed

- C1 (rollback after commit permitted): fixed. `Transactor` marks
  finalization on clean `Commit`; `Rollback` refuses with
  `ErrRollbackPastFinalize` and the committed bytes survive.
  `TestTransactorRefusesRollbackPastFinalize` drives
  rollback-without-prepare, pre-commit rollback, and the
  commit-then-rollback arm it is named for.
- C2 (constant skip report): fixed. `Report()` reads the live
  counters; `TestSkipReportReadsRecordedVerdicts` proves it moves
  with recorded verdicts. Round-1 `skipped=0` citations replaced
  everywhere (README, LOGBOOK, doc.go, here) with the genuine datum:
  0 `--- SKIP` / 0 `--- FAIL` lines under
  `go test ./internal/secconftest/ -v -count=1` on darwin non-root.
- C3 (three probes gate nothing): fixed. Fifo probe builds a real
  FIFO; all 4 probes gate named tests (symlink→
  `TestHostileSymlinkEscapeRefusedAtCommit`, fifo→
  `TestSkipFifoFixtureExchangesBytes` with byte exchange, mode-bits +
  nonroot→`TestSkipModeBitsAndNonRootDenialsHold`). Probe-honesty
  seam kills success-on-failure (N14).
- C4 (residue): all 8 listed survivors now killed (N1–N6, A3, A4
  rows above). New denominator 29 applied with residue stated, not
  zero claimed.

Verified, not re-litigated: roster spec pinning, fuzz arrival (6/8
secprim defeat plants redden; IsEnvName/Redact now carry no-op poles
at the driver level), determinism, fuzz-registration gate,
provenance. AC row 6 reads 4 of 4 wired; row 1 digest witness 3 of 3.

## Skips on this host (darwin/arm64, uid 502)

`symlink ok` (create+identify), `fifo ok` (mkfifo+Lstat),
`mode-bits ok` (denial enforced), `nonroot ok` (uid 502). Verbose
suite: 0 `--- SKIP`, 0 `--- FAIL`. Self-test verdict line from
`TestSkipReportReadsRecordedVerdicts` shows real nonzero counters,
e.g. `skipped=1 "self-test-skip-probe" failed=1
"self-test-fail-probe"`. On root hosts the denial test skips via the
nonroot probe (recorded); on Windows the fifo/mode-bits tests skip
(non-strict) — the SKIP line plus the recorded verdict is the
evidence in both lanes.

## Determinism

Runner digest `5569f34caab734eb` identical across 5 independent
processes (same value as round 1; `Runner`/`Digest` untouched).
Capability probes deterministic across repeat runs.

## Gates (real exit codes, standalone processes)

- `go test ./... -count=1` → exit 0 (21 pkgs ok, 0 FAIL)
- `go test ./internal/secconftest/ -race -count=1` → exit 0
- `go test ./... -cover -count=1` → exit 0 (secconftest 89.3%, secprim 94.4%)
- `go vet ./...` → exit 0; `GOOS=windows go vet ./...` → exit 0
- `GOOS=windows go build ./...` → exit 0; `go build ./...` → exit 0
- `gofmt -l internal/` → clean; `git diff --check` → clean
- `go run ./internal/traceability/cmd/tracecheck` → exit 0
  (contracts=60 sections=36 cases=94 fixtures=30 compat=55; unchanged)
- 8/8 `secconftest -fuzz -fuzztime=100x` → exit 0 each, no corpus written
- 29/29 mutants `go vet` clean under the plant

## Stated bounds (round-2 deltas; round-1 bounds kept)

- Crash-point names are instrument-local, not the SPEC CR-MAT-01..08
  registry; a site added without an arm is outside the model. Every
  suite arm is consumption-checked via `RemainingArmed`.
- Env-name/redact fuzz pins cover the no-op poles; interior grammar
  and per-class markers stay with the hostile corpus.
- Skip-gated false on non-strict platforms skips by construction
  (S1); the SKIP line plus recorded verdict is the evidence.
- Driver delegation is read-verified with arrival counts (N16).
- No capability is advertised: the instrument enables no platform
  lane or provider suite.


