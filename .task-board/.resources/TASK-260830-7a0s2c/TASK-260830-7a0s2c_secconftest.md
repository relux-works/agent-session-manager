# TASK-260830-7a0s2c — fixture, fuzz, and fault test infrastructure

Leaf 2 of STORY-260830-1i3qu7. New package `internal/secconftest`
(doc, runner, clock, crash, fuzz, hostile, skip + runner, clock,
crash, fuzz, hostile, skip tests) plus 8 fuzz-validation commands in
`task-board.config.json`, a README section, and a LOGBOOK entry.
Leaf-1 `internal/secprim` production code is untouched.

Worktree tree OID (computed, uncommitted): `9ec5199c07a6fcf6f6dfa1ee8b682ec13037b3e4`
Base commit: `722ed1a8c3ac0e4169990f55318897f8e39247f0`
(HEAD tree `5eb6d8a547b6efa392cdfe0c66dc2132fed96ce5`)

## AC coverage: 6 of 6 rows driven through production entry points

| # | AC row | Production call site | Named committed test |
|---|--------|----------------------|----------------------|
| 1 | deterministic fixture runners | `secconftest.Runner.Run` / `Digest` | `TestRunnerDigestIsDeterministic` (digest `5569f34caab734eb` x3 runs), `TestRunnerExecutesInSortedOrder` |
| 2 | fake clocks | `secconftest.Fake.Now` / `Advance` / `Set` | `TestFakeClockIgnoresRealTime` (20 ms sleep moves wall, not fake), `TestRecordingNoticesUnwiredClock` |
| 3 | crash points | `secconftest.Injector.MaybeFail`, `Transactor.Prepare/Commit/Rollback` | `TestTransactorCrashOutcomes` (4 points, exact outcomes + byte state), `TestTransactorRecoversAfterFault` |
| 4 | protocol fuzzing | 8 entries below | 8 `Fuzz*` targets + `TestFuzzDriversReachProduction` (22 arrivals/entry) |
| 5 | hostile strings | `secprim.CheckMemberPath`, `Guard.Resolve/Open`, `DetectCaseCollision`, `CheckArgv/NewCommand`, `IsEnvName/BuildEnv`, `EscapeForTerminal`, `Redact` | `TestHostilePathMembersRefused`, `TestHostileSymlinkEscapeRefusedAtCommit`, `TestHostileArgvMembersStaySingleElements`, `TestHostileRenderMembersNeutralized`, `TestHostileRedactMembersScrubbed`, … |
| 6 | platform capability skips | `secconftest.Require` / probes | `TestSkipDecideTable`, `TestSkipReportLogsHostCounts` |

Negative/refusal rows (gate admits what it must reject → named test
fails): unreached fixture refused (`TestRunnerRefusesUnreachedFixture`);
unwired clock reads zero (`TestRecordingNoticesUnwiredClock`); wrong-section
quote refused (`TestQuoteInSectionRefusals`); strict-platform absence fails
(`TestSkipDecideTable`); changed-body retry → `idempotency_mismatch` with no
second root (`TestTransactorIdempotentRetry`); unmanaged member refused with
`ErrUnsafePath` (`TestHostileUnmanagedMembersRefused`); NUL/empty argv refused
with `ErrUnsafeArgv`; rollback without prepare refused.

## Fuzz targets → entry points (arrival proven by seed run)

| Target | Entry | Arrivals (seed run) |
|--------|-------|---------------------|
| FuzzCheckArgv | secprim.CheckArgv | 22 |
| FuzzCheckMemberPath | secprim.CheckMemberPath | 22 |
| FuzzIsEnvName | secprim.IsEnvName | 22 |
| FuzzRedactCorpus | secprim.Redact | 22 |
| FuzzEscapeForTerminal | secprim.EscapeForTerminal | 22 |
| FuzzRenderForTerminal | secprim.RenderForTerminal | 22 |
| FuzzGuardResolve | secprim.Guard.Resolve | 22 |
| FuzzDetectCaseCollision | secprim.DetectCaseCollision | 22 |

Arrival is established by `TestFuzzDriversReachProduction`, which drives 22
seeds through the same `Driver` methods the fuzz targets call and asserts a
non-zero counter per entry. 8/8 targets pass 5 s smoke and `-fuzztime=100x`
with no corpus written (tree clean after). All 8 are registered in the
configured validation matrix; the repo gate
`TestConfiguredValidationRunsEveryFuzzTargetWithFixedBudget` failed until
they were registered, so unfuzzed-in-CI targets cannot be added silently.

## Mutant battery: 10 applied, 10 killed, 0 survivors

| Mutant | What it narrows the gate to | Named failing test |
|--------|-----------------------------|--------------------|
| M1 narrowing: arrival gate exempts one unreached member | zero-call class minus `"unreached"` | TestRunnerRefusesUnreachedFixture |
| M2 narrowing: prepare-commit outcome → safe_retry | one wrong outcome admitted | TestCrashClassifyTable (+ TestTransactorCrashOutcomes) |
| M3 narrowing: strict-unavailable → skip | unexpected skip passes quietly | TestSkipDecideTable |
| M4 token-preserving: QuoteInSection drops section check, keeps token match | wrong-section quote admitted | TestQuoteInSectionRefusals |
| M5 arm-deletion: MaybeFail never fires | no crash point exists | TestCrashPointFiresOnceWhenArmed |
| M6 arm-deletion: Fake.Now returns time.Now() | wall clock leaks in | TestFakeClockStartsPinned (+ IgnoresRealTime) |
| M7 arm-deletion: runner sort removed | registration order leaks in | TestRunnerExecutesInSortedOrder |
| M8 arm-deletion: DriveEscape returns input | entry unreachable | TestFuzzDriversReachProduction |
| M9 census: roster class renamed out of coverage | 18-class census breaks | TestHostileRosterShape |
| M10 census: digest drops last fixture | execution log unwitnessed | TestRunnerDigestCoversAllFixtures |

Split: narrowing 4, arm-deletion 4, census-only 2. M4 additionally ran the
full behavioral suite under the mutant: 222 pass, exactly 1 fail
(TestQuoteInSectionRefusals) — the kill is unconfounded and no behavior
test depends on the checker. Denominator is production-derived (all mutants
in `internal/secconftest` non-test files); every mutant was reverted and the
package is green after each (`RESTORED-GREEN` x4 runs).

Survivors: none. No bound is claimed from a survivor.

## Skips on this host (darwin/arm64)

`symlink ok, fifo ok, mode-bits ok, nonroot ok` — suite report
`skipped=0 failed=0`. secprim + secconftest verbose run: 0 `--- SKIP`
lines. An unexpected skip on a strict platform fails via `decide`.

## Determinism

Runner digest `5569f34caab734eb` identical across 3 processes;
registration-order reversal identical; capability probes deterministic
across repeat runs.

## Gates (real exit codes, standalone processes)

- `go test ./... -count=1` → exit 0 (21 pkgs ok)
- `go test ./... -race -count=1` → exit 0
- `go test ./... -cover -count=1` → exit 0 (secconftest 86.9%)
- `go vet ./...` → exit 0; `GOOS=windows go vet ./...` → exit 0
- `GOOS=windows go build ./...` → exit 0; `go build ./...` → exit 0
- `gofmt -l internal/` → clean; `git diff --check` → clean
- `go run ./internal/traceability/cmd/tracecheck` → exit 0
  (contracts=60 sections=36 cases=94 fixtures=30 compat=55;
  coverage line unchanged from baseline)

## Reuse (no new copies)

No decoder, bound check, string measure, path handling, or escaping was
written: string measure stays in `environ.StringLength`, member grammar in
`scalar.ParseRelativePath`, device table in `scalar.IsReservedWindowsDeviceName`,
refusals/escaping/redaction in `secprim`. Hostile members derive from the
pinned spec text (`specdoc`) + Unicode tables + `scalar` predicates, never
from `secprim` tables.

## Stated bounds

- The Transactor proves the injector/outcome/idempotency vocabulary, not any
  product recovery path.
- Fuzz covers Section 16 string/vector gates, not containment races or TOCTOU.
- Tombstone-deletion and TOCTOU shapes have no product surface in this story
  and are not driven. `path-symlink-escape` is an FS witness, not a string
  corpus member (pinned by roster-shape test).
- `FuzzTargetEntry` map↔target correspondence is pinned by count + arrival
  tests; Go has no runtime fuzz-target registry (stated, not claimed).
- No capability is advertised: the instrument cannot enable a platform lane
  or provider suite.

## For the next leaf (CI gates) — not board elements, prose only

- The 8 `secconftest -fuzz -fuzztime=100x` commands added to
  `task-board.config.json` are ready for the M0 CI leaf to execute.
- Suggested follow-up (no board element created per story rule): crash-point
  coverage for product journals when a product journal lands; Windows-lane
  skip-count evidence from a Windows runner.
