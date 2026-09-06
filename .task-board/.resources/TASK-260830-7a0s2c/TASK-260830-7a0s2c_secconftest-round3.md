# TASK-260830-7a0s2c round 3 — review-verdict-rev2 rework (C1/C1b/C2/C2b)

Round-3 rework of the `internal/secconftest` instrument leaf. Rounds 1–2
stay as history; this document supersedes their claims where review proved
them wrong, including two round-2 outcome sentences named below. Leaf-1
`internal/secprim` production code is untouched
(`git diff 722ed1a -- internal/secprim` empty, no untracked additions there).

- Base commit `722ed1a`.
- Candidate manifest (sha256 of the sorted per-file sha256 manifest over
  `internal/secconftest/`):  
  `c6018721bd0941b4ce29a7e36fd37108b625711c6024fb541ab9ef7cb22cfb0d`.
  Re-taken after the battery: byte-identical (`diff` of pre/post manifests
  empty, battery prints `MANIFEST IDENTICAL`).

Files changed vs round 2: `crash.go` (crashed-commit finalization,
`ErrPreparePastFinalize` / `ErrCommitPastFinalize`, rollback forgets,
staged-missing naming), `crash_test.go` (crashed-commit rule, terminal
neighbours, empty-ID, rollback-enter drive), `skip.go` (`requireT`
interface), `skip_test.go` (fake-Require wiring tests, strictness pin,
M02 rows), `main_test.go` (new: `TestMain` final report),
`hostile_test.go` (M27 message pin), `doc.go` + `README.md` (bounds),
`LOGBOOK.md` (round-1 entry corrected in place).

## C1 decision — a crashed commit counts as committed

Rule chosen: the commit-apply fault fires **after** the durable write, so
the operation is committed in every observable way (`CommittedBytes`
returns the body, and the package's own parked-bytes assertion requires it
to survive). The fault path therefore stores a `recoverable_parked_state`
receipt and marks `finalized`; `Rollback` afterwards refuses with
`ErrRollbackPastFinalize` and the bytes survive. Recovery from parked is via
`CommittedBytes`/operator, never via `Rollback` deleting evidence.

Why: the alternative — admitting the rollback and deleting the parked bytes
— walks through both halves of AC-CLONE-005 at once (rollback forbidden
after Provider commit; all bundle evidence survives). Either of the other
two rules would be defensible in isolation; picking neither, which is what
the leaf did, is the one unacceptable answer. Driven by
`TestTransactorCrashedCommitIsFinal` (fault → bytes → refused rollback →
bytes intact → second commit and re-prepare refused → bytes intact).

## C1b neighbours — terminal means terminal, every refusal named

| Sequence | Result now | Named test |
| --- | --- | --- |
| rollback twice, pre-commit | 1st `nil` (forgets op), 2nd `rollback without prepare` | `TestTransactorTerminalNeighbours/rollback-twice` |
| commit twice, cleanly | 2nd `ErrCommitPastFinalize`, zero `Receipt`, bytes intact | `.../commit-twice` |
| commit after rollback | `commit without prepare`, no committed file | `.../commit-after-rollback` |
| prepare after clean commit, identical body | `ErrPreparePastFinalize` (idempotent retry covers pre-commit only) | `.../prepare-after-clean-commit` |
| prepare after rollback | admitted, restages (the recovery retry), commits byte-exact | `.../prepare-after-rollback-restages` |
| receipt present, staged bytes missing | `commit without staged bytes` (wraps ENOENT, names state) | `.../commit-without-staged-names-state` |

No terminal path inherits a raw filesystem error. `P7` is driven, not
stated: `TestTransactorCrashOutcomes` now arms all five points including
`PointRollbackEnter` (prepare, then faulting rollback, staged bytes kept).

## C2 — round-1 LOGBOOK entry corrected in place

The 2035 entry's `EVIDENCE` line no longer asserts the disproven numbers;
it keeps the true digest and carries a `CORRECTION (round-3)` note naming
the withdrawn claims and pointing at the superseding documents. Sweep for
superseded numbers-as-claims: `internal/secconftest/`, `README.md`,
`task-board.config.json` clean (remaining mentions describe the false
citation as false or disavow it in the correction note).

## C2b — Require wiring pinned, final report observable

- `Require` takes a minimal `requireT` interface (`Helper`/`Skipf`/`Fatalf`);
  `*testing.T` satisfies it, and wiring tests pass a recording fake, so both
  arms are driven through the real function without failing or skipping the
  suite. `TestRequireSkipArmRecordsVerdict` (M05) and
  `TestRequireFailArmRecordsVerdict` (M04) assert the fake saw the control
  call **and** the counters carry the driven ID.
- `TestMain` prints `Report()` after every test including the parallel ones,
  so a real parallel verdict is observable in the output. The mid-suite
  `LogReport` line still prints before the first parallel `CONT` (Go runs
  parallel tests after sequential ones) — that ordering is structural, and
  the document no longer asks that line to witness parallel verdicts.
- `TestSkipReportReadsRecordedVerdicts` is now documented as pinning
  `Report`, not the wiring.

Two round-2 sentences superseded (both untrue as written):

1. S1 bound *"the denial test emits a `--- SKIP` line and `recordVerdict`
   logs it"* — `recordVerdict` logs nothing. Now: the denial test emits
   `--- SKIP` where the probe is unavailable; `Require` records the verdict
   in the process counters and `TestMain` prints them after the suite.
2. Windows/root paragraph *"the SKIP line plus the recorded verdict is the
   evidence in both lanes"* — the recorded verdict was unobservable. Now:
   the SKIP line is the evidence in the skip lane; the counters are
   observable in the `TestMain` line, which honestly reads zero on a host
   where every probe passes.

## AC coverage: 6 of 6 rows driven through production entry points

| # | AC row | Production call site | Named committed test |
|---|--------|----------------------|----------------------|
| 1 | deterministic fixture runners | `secconftest.Runner.Run` / `Digest` | `TestRunnerDigestIsDeterministic` (digest `5569f34caab734eb`), `TestRunnerExecutesInSortedOrder`, report/name/calls distinguishing tests (3 of 3 components) |
| 2 | fake clocks | `secconftest.Fake.Now` / `Advance` / `Set` | `TestFakeClockIgnoresRealTime`, `TestRecordingNoticesUnwiredClock` |
| 3 | crash points | `Injector.MaybeFail`, `Transactor.Prepare/Commit/Rollback` | `TestTransactorCrashOutcomes` (5 points + arm consumption), `TestTransactorRecoversAfterFault`, `TestTransactorRefusesRollbackPastFinalize`, `TestTransactorCrashedCommitIsFinal`, `TestTransactorTerminalNeighbours` (6 subtests), `TestTransactorCommitRequiresPrepare`, `TestCrashPointVocabularyIsPinned` |
| 4 | protocol fuzzing | 8 `Driver` entries | 8 `Fuzz*` targets + `TestFuzzDriversReachProduction` + no-op poles in `FuzzIsEnvName` / `FuzzRedactCorpus` |
| 5 | hostile strings | `secprim.CheckMemberPath`, `Guard.Resolve/Open`, `DetectCaseCollision`, `CheckArgv/NewCommand`, `IsEnvName/BuildEnv`, `EscapeForTerminal`, `Redact` | hostile battery + `TestHostileGatesAreWired` (18 Gates pinned and driven) + `TestBidiRunesArePinned` (16/16 runes, corpus-covered) |
| 6 | platform capability skips | `secconftest.Require` / probes / `TestMain` | `TestSkipDecideTable`, `TestSkipReportReadsRecordedVerdicts`, fake-arm wiring tests, `TestSkipStrictPlatformsArePinned`, fifo/delay gating tests, probe-honesty test (4 of 4 probes wired) |

## Mutant battery: 38 applied, 36 killed, 2 survived with bounds

Denominator derived from this tree's production files (battery
`/tmp/mutant-battery-round3.py`, log attached): applied 38,
NOT_APPLIED 0, COMPILE_FAIL 0, VET_FAIL 0 (every mutant vetted clean),
KILLED 36, SURVIVED 2. Split: narrowing 28 (26 killed, 2 survived),
arm-deletion 6 (6 killed), census-only 4 (4 killed). The token-preserving
mutant (`N-TokenPreserving`: section check dropped, `section` still read)
ran the full behavioral suite as its harness and reddened at
`TestQuoteInSectionRefusals`. `P7`, `M04`, `M05` each killed by name (rows
below). Tree manifest identical after the battery.

| Mutant | What it narrows the gate to | Named failing test |
|--------|-----------------------------|--------------------|
| C1a narrowing: rollback gate + dead conjunct | post-commit rollback admitted | TestTransactorRefusesRollbackPastFinalize |
| C1b narrowing: crashed commit finalizes only empty bodies | parked rollback admitted, bytes deleted | TestTransactorCrashedCommitIsFinal |
| C1c narrowing: commit past-finalize + dead conjunct | second commit falls to staged error | TestTransactorTerminalNeighbours/commit-twice |
| C1d narrowing: prepare past-finalize + dead conjunct | re-prepare after commit admitted | TestTransactorTerminalNeighbours/prepare-after-clean-commit |
| C1f narrowing: staged-missing naming deleted | raw ENOENT, unnamed | TestTransactorTerminalNeighbours/commit-without-staged-names-state |
| P7 narrowing: Arm drops PointRollbackEnter | 5th point never fires, suite green | TestTransactorCrashOutcomes |
| M02 narrowing: decide drops empty-ID disjunct | empty-ID capability admitted when probe holds | TestSkipDecideTable |
| M04 narrowing: Require fail arm stops recording | failures unrecorded, suite green | TestRequireFailArmRecordsVerdict |
| M05 narrowing: Require skip arm stops recording | skips unrecorded, suite green | TestRequireSkipArmRecordsVerdict |
| M08 narrowing: symlink StrictGOOS drops darwin | unexpected absence skips instead of failing | TestSkipStrictPlatformsArePinned |
| M16 narrowing: Prepare empty-ID guard never matches | empty operation staged | TestTransactorRefusesEmptyOperationID |
| M27 narrowing: empty-quote guard drops TrimSpace | whitespace quote refused downstream by accident | TestQuoteInSectionRefusals |
| N-DigestReport narrowing: Digest drops Report | report-class unwitnessed | TestRunnerDigestDistinguishesReports |
| N-DigestName narrowing: Digest drops Name | name-class unwitnessed | TestRunnerDigestDistinguishesNames |
| N-DigestCalls narrowing: Digest drops Calls | count-class unwitnessed | TestRunnerDigestCoversAllFixtures |
| N-BidiFEFF narrowing: bidi drops U+FEFF | 15-rune vocabulary admitted | TestBidiRunesArePinned |
| N-BidiCut narrowing: bidi cut 16→1 | 1-rune vocabulary admitted | TestBidiRunesArePinned |
| N-GateRename narrowing: path-traversal Gate→Redact | wrong gate decides the class | TestHostileGatesAreWired |
| N-ClassifyRow narrowing: CommitApply→explicit_rollback | one wrong outcome admitted | TestCrashClassifyTable |
| N-MaybeFailKeepsArm narrowing: arm never consumed | every arm fires forever | TestCrashPointFiresOnceWhenArmed |
| N-RemainingEmpty narrowing: RemainingArmed constant-empty | unreached arms invisible | TestInjectorArmedPointIsConsumedByTheDrive |
| N-ClassifyRegistry narrowing: Classify admits CR-MAT-01 | registry name accepted | TestCrashPointVocabularyIsPinned |
| N-FifoTrueOnFailure narrowing: fifo probe true-on-failure | broken build reported ok | TestSkipFifoProbeAttemptsItsCapability |
| N-DriveEnvAdmitsAll narrowing: DriveEnvName admits all | empty name admitted | FuzzIsEnvName (seed run) |
| N-FakeWallClock narrowing: Fake returns time.Now() | wall clock leaks in | TestFakeClockIgnoresRealTime |
| N-TokenPreserving narrowing: section check dropped, token kept | wrong-section quote admitted (full-suite harness) | TestQuoteInSectionRefusals |
| N-DriveRedactNoDelegate narrowing: DriveRedact stops delegating | — SURVIVOR, bound below | (suite green) |
| S1-ModeBitsFalse narrowing: mode-bits probe false-on-denial | — SURVIVOR-AS-SKIP, bound below | (denial test SKIPs, suite green) |
| A-FinalizeDeleted arm-deletion: rollback finalize gate deleted | post-commit rollback nils and deletes | TestTransactorRefusesRollbackPastFinalize |
| A-CommitWithoutPrepareDeleted arm-deletion: without-prepare check deleted | staged-bytes-no-receipt commit admitted | TestTransactorCommitRequiresPrepare |
| A-RecordVerdictGutted arm-deletion: recordVerdict body gutted | verdicts never recorded | TestSkipReportReadsRecordedVerdicts |
| A-RunSortDeleted arm-deletion: Run sort deleted | registration order leaks in | TestRunnerExecutesInSortedOrder |
| A-IdempotencyDeleted arm-deletion: mismatch refusal deleted | changed-body retry stages anew | TestTransactorIdempotentRetry |
| A-RollbackForgetDeleted arm-deletion: rollback forget deleted | second rollback nils silently | TestTransactorTerminalNeighbours/rollback-twice |
| C-RosterDrops census: roster loses path-device | 17-class roster passes | TestHostileRosterShape |
| C-FuzzEntryDrops census: FuzzTargetEntry loses one | 7-entry map passes | TestFuzzDriversReachProduction |
| C-TraversalDrops census: traversal corpus loses one | 5-member class passes | TestHostileMemberCounts |
| C-CapsDrops census: DefaultCapabilities loses nonroot | 3-probe list passes | TestSkipDefaultProbesTerminate |

Survivor bounds. N-DriveRedactNoDelegate (= round-2 N16): without
delegation the driver keeps counting, and the redact fuzz properties call
`secprim.Redact` directly, so the suite stays green. Bound: driver
delegation is read-verified with arrival counts, not mutant-pinned; stated
in README and doc.go. S1-ModeBitsFalse: a non-strict probe reporting
unavailable skips by construction — the denial test emits a `--- SKIP` line
and the suite stays green. Bound: the SKIP line is the signal; CI watches
skip counts rather than treating green as unskipped.

## Skips on this host (darwin/arm64, non-root)

`symlink ok`, `fifo ok`, `mode-bits ok` (denial enforced), `nonroot ok`.
Verbose suite: 246 `--- PASS`, 0 `--- SKIP`, 0 `--- FAIL`. Mid-suite
self-test line shows its own nonzero counters
(`skipped=1 "self-test-skip-probe" failed=1 "self-test-fail-probe"`,
restored afterwards); the `TestMain` final line honestly reads
`skipped=0 "" failed=0 ""` because every probe passes here.

## Determinism

Runner digest `5569f34caab734eb` (unchanged across rounds;
`Runner`/`Digest` untouched). Capability probes deterministic across repeat
runs. Manifest `c6018721…` identical before/after battery, probes, and
finish.

## Gates (real exit codes, standalone processes)

- `go test ./... -count=1` → exit 0 (21 pkgs ok, 0 FAIL)
- `go test ./internal/secconftest/ -race -count=1` → exit 0
- `go test ./... -cover -count=1` → exit 0 (secconftest 92.6%, secprim 94.4%)
- `go vet ./...` → exit 0; `GOOS=windows go vet ./internal/secconftest/` → exit 0
- `GOOS=windows go build ./...` → exit 0
- `gofmt -l internal/` → clean; 38/38 mutants `go vet` clean under the plant
- `go run ./internal/traceability/cmd/tracecheck` → exit 0
  (contracts=60 sections=36 cases=94 fixtures=30 compat=55; unchanged)
- 8/8 `secconftest -fuzz -fuzztime=100x` → exit 0 each, no corpus written

## Stated bounds (round-3 deltas; round-1/2 bounds kept)

- Finalization is terminal for clean and crashed commits; pre-commit
  rollback forgets so a fresh prepare restages the retry. The commit model
  still proves the injector and vocabulary, not any product recovery path.
- Crash-point names are instrument-local; a site added without an arm is
  outside the model. All five suite arms are driven and consumption-checked.
- Env-name/redact fuzz pins cover the no-op poles; interior grammar and
  per-class markers stay with the hostile corpus.
- Skip-gated false on non-strict platforms skips by construction (S1); the
  SKIP line is the signal. `TestMain` witnesses the counters after the
  whole suite; zero is an honest reading where every probe passes.
- Driver delegation is read-verified with arrival counts (N16).
- No capability is advertised: the instrument enables no platform lane or
  provider suite.

