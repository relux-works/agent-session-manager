# TASK-260830-2uowwk — M0 CI and release gates (rev2, answers review-verdict-rev1)

Candidate tree OID: `8d8231ac0ae2ba557f4c02d0c88b9f905a19821a`

Derived without touching the real index (`GIT_INDEX_FILE` temp index seeded
from `HEAD`, `git add -A`, `git write-tree`) and checked before naming:
`git ls-tree -r <oid> -- internal/cigate` returns 7 rows, and
`git diff-tree --no-commit-id --name-only -r HEAD <oid>` returns exactly the
10 changed files (`.github/workflows/ci.yml`, `LOGBOOK.md`, `README.md`,
plus the 7 new `internal/cigate` files). The rev1 defect (naming
`HEAD^{tree}`, which omits untracked `cigate`) is closed by construction:
an OID is named only after its tree is shown to contain the work.

Story: STORY-260830-1i3qu7 (security-primitives-and-conformance-ci), final leaf.
Scope: AX v0.5.0 §16, §19.2–19.5, §20.2, Appendix D. Work stays inside the
story boundary: `internal/secprim` and `internal/secconftest` production code
are untouched (`git status` clean on both after two probe mutants, each
restored byte-identical with `cmp`), and no foreign guard was weakened.

## What changed since rev1 (review-verdict-rev1 F1–F5 + battery observation)

- F1: deleted the `Record live probe verdicts` CI step (synthetic verdicts,
  constant grepped line, non-gating pipe). The `capability-claims` job now has
  one step, renamed `Check README with all probes forced unavailable`, with a
  workflow comment stating live probe outcomes are read by no call site.
  README gate table and `cigate/doc.go` describe the forced-unavailable
  posture instead of live scanning.
- F3: wired instead of deleted. New `TestListedTargetsMatchDerivation` holds
  CI's `go test -list ^Fuzz` leg equal to `cigate.FuzzTargets()`; the
  fuzz-smoke job runs it (plus `TestFuzzTargetsAreDerivedSorted`) name-guarded
  before smoking. `targets.go` and `doc.go` state this exact mechanism.
- F5: fuzz-smoke derives per package over `internal/secconftest`,
  `internal/canonicaljson`, `internal/scalar` (8+4+1=13, count pinned in CI),
  each target smoked with the `^fuzz: elapsed` guard. `doc.go` no longer
  claims entry points stay in one package.
- F4: the backtick bound is now stated in `doc.go` Stated bounds and in the
  README gate row: only backticked capability IDs are scanned; bare-prose
  advertisements are admitted unchecked. Pinned by new
  `TestUnbacktickedProseIsOutsideTheScanner`.
- Battery: both size `Fatalf`s removed. Each verb/negation row kills by
  behaviour in its own subtest; `battery-covers-gate` census subtests fail on
  gate additions without rows. Three negation rows rewritten so each carries
  exactly one negation (`n't`, `pending`, `unless` previously hid behind a
  second negation).
- `LOGBOOK.md`: entry 2249 (this round). Not charged, noted only: the
  catalog-freshness step is still defeated by `//go:generate` →
  `// go:generate` (token preserved) — pre-existing at HEAD, and the
  contract-version gate still catches version drift with the directive
  disarmed.

## AC coverage: 11 of 11 rows driven

| # | AC row | Production entry point | Named committed test |
| --- | --- | --- | --- |
| 1 | historical (v0.4.3) contract preservation runs in CI | `cigate.VerifyContractPreservation` → `specpin.Current` + `catalog.ForRelease(v0.4.3)` | `TestVerifyContractPreservationLive` |
| 2 | current (v0.5.0) contract preservation runs in CI | `cigate.VerifyContractPreservation` → `specpin.Current` + `catalog.ForRelease(v0.5.0)` | `TestVerifyContractPreservationLive`, `TestDerivedSetsAreComplete` |
| 3 | linters run in CI | `go vet`, `GOOS=windows go vet`, `gofmt` (workflow) | `TestCIWorkflowInvokesTraceabilityGate` pins vet/build literals; gofmt guard proven red (§Red) |
| 4 | race tests run in CI | `go test -race ./...` (workflow race job) | full suite green under `-race`; `-race` proven rejecting (§Red) |
| 5 | coverage runs in CI | `go test ./... -cover` (workflow coverage job, numbers only, no floor) | 22 `coverage:` lines; missing-package fails (§Red) |
| 6 | fixture matrices run in CI | os×package matrix (`secprim`, `secconftest`); per-package `-list`-derived fuzz loop over `secconftest`, `canonicaljson`, `scalar` with `FuzzTargets` cross-check | `TestFuzzTargetsAreDerivedSorted`, `TestListedTargetsMatchDerivation`, matrix `--- PASS` guards, per-target `fuzz: elapsed` guard, 13-target count pin |
| 7 | unsupported-capability claim check runs in CI | `cigate.CheckAdvertisements` over README with every probe forced unavailable (`allUnavailable` posture; `CapabilityIDs` supplies the vocabulary via `ProbeStates`) | `TestRealREADMECarriesNoPositiveClaim`, `TestCapabilityIDsAreDerivedSorted`, `TestUnbacktickedProseIsOutsideTheScanner` (stated bound) |
| 8 | exact contract fixtures pass | `secprim.CheckMemberPath/CheckArgv/IsEnvName/EscapeForTerminal/Redact` via hostile corpus | `TestHostilePathMembersRefused`, `TestHostileMemberCounts` (17 classes + counts), `TestFuzzDriversReachProduction` |
| 9 | negative/refusal cases pass | same gates, refusal arms | `TestHostileGatesAreWired`, `TestQuoteInSectionRefusals`, `TestRunnerRefusesUnreachedFixture` |
| 10 | crash/idempotency evidence (durable-state operations) | `secconftest.Transactor` (Prepare/Commit/Rollback) in matrix | `TestTransactorIdempotentRetry`, `TestTransactorCrashedCommitIsFinal`, `TestTransactorRefusesRollbackPastFinalize`; `cigate` itself is read-only (stated bound) |
| 11 | no unsupported capability advertised | `cigate.CheckAdvertisements` over README with every probe forced unavailable — host-independent, stronger than live scanning; live `secconftest` probe outcomes are read by no call site | `TestRealREADMECarriesNoPositiveClaim` (all-unavailable posture), `TestUnbacktickedProseIsOutsideTheScanner` (bound: backticked IDs only) |

## Mutant battery: 30 applied / 0 NOT_APPLIED / 0 COMPILE_FAIL; 28 killed, 2 survived with bounds

Harness: `/tmp/run-mutants1.py` + `/tmp/run-mutants2.py` (kept outside the
worktree for re-runs); full log `TASK-260830-2uowwk_mutants-rev2.log`.
Each mutant is applied by an exact single-occurrence anchor (else NOT_APPLIED),
run against the full `internal/cigate` suite, then restored and verified
byte-identical by sha256 before the next mutant. Production denominator:
every gate word (5 verbs + 13 negations), both matcher narrowings, case
folding, version ordering, every refusal arm, both order censuses.

### Narrowing mutants (gate stays, admits exactly one member of the class)

| Mutant | What it narrows the gate to | Named test that fails |
| --- | --- | --- |
| V-available (drop `available`) | admits the available-claim | `TestPositiveClaimsAreRefused/available` (+ `TestFindingCarriesLine`, shared sentence) |
| V-supported (drop `supported`) | admits the supported-claim | `TestPositiveClaimsAreRefused/supported` |
| V-enabled (drop `enabled`) | admits the enabled-claim | `TestPositiveClaimsAreRefused/enabled` |
| V-works (drop `works`) | admits the works-claim | `TestPositiveClaimsAreRefused/works` |
| V-passes (drop `passes`) | admits the passes-claim | `TestPositiveClaimsAreRefused/passes` |
| G-not … G-only (13 negation drops, one per word) | refuses the honestly conditional sentence | `TestNegatedClaimsAreAdmitted/<word>` — all 13 fire in their own subtest |
| N3 boundary-drop (substring match) | admits `protest`-style token-preserving mention | `TestSubstringIsNotAMarker` |
| B1 backtick-widen (match bare IDs) | refuses bare-prose mentions | `TestUnbacktickedProseIsOutsideTheScanner` (+ `TestRealREADMECarriesNoPositiveClaim`: bare `fifo` in README prose gets refused) |
| N5 case-narrow (no folding) | admits capitalised `Available` / `No` | `TestPositiveClaimsAreRefused/capital`, `TestNegatedClaimsAreAdmitted/no` |
| N4 version-order-blind | admits reordered versions | `TestCheckAgreementRefusals/version order changed` |
| E1 entry-drop (`FuzzGuardResolve` removed from `FuzzTargetEntry`) | CI list covers a target the derivation does not | `TestListedTargetsMatchDerivation` (+ `TestFuzzDriversReachProduction` in `secconftest`); restored byte-identical (`cmp`), frozen leaves clean |

### Arm-deletion mutants (one refusal arm removed)

| Mutant | What it deletes | Named test that fails |
| --- | --- | --- |
| A1 positive arm off | admits marked positive claims | `TestPositiveClaimsAreRefused` verb subtests (+ `TestFindingCarriesLine`) |
| A2 unclassified arm off | admits unmarked neutral mentions | `TestUnclassifiedMentionIsRefused` (+ `TestSubstringIsNotAMarker`) |
| A3 empty guards off | passes on empty vocabulary/document | `TestEmptyInputsAreRefused` |
| A4 catalog-extra direction off | admits catalog-only contracts | `TestCheckAgreementRefusals` |
| A5 target guards off | admits malformed/empty derivation | `TestSortedTargetsRefusals` |
| A6 root comparison off | admits one-sided release rename | `TestCheckReleaseRootsRefusesDrift` |

### Survivors (census-only, with bounds)

| Mutant | Bound that survival states |
| --- | --- |
| C1 release order swap | iteration order is presentation; agreement is keyed, tests assert sets — order carries no verdict |
| C2 finding order reverse | emission order is presentation; the mixed-availability test asserts refusal sets — order carries no verdict |

Behavioural-kill spot checks (beyond the harness parent lines): under the
`supported` drop, `TestPositiveClaimsAreRefused/supported` fires while
`battery-covers-gate` stays green; under the `pending` drop,
`TestNegatedClaimsAreAdmitted/pending` fires. The kills are behavioural,
not bookkeeping.

## Deliberate red per job (all observed, real exit codes)

| Job | Failing input | Observed |
| --- | --- | --- |
| contract-preservation | `-run TestDoesNotExist` exits 0; `--- PASS` guard exits 1 | selector green on empty, guard red — the class this job exists to catch |
| contract-preservation | `tracecheck -section bogus` | exit 1 (`invalid assigned section`) |
| contract-preservation | version-drop/reorder/extra mutants | killed (battery N4/A4/A6) |
| lint | scratch `gofmt`-dirty file | listed (exit 0) → guard exit 1 |
| test/race | scratch data-race module under `-race` | `WARNING: DATA RACE`, exit 1 |
| coverage | `go test ./internal/does-not-exist -cover` | exit 1 (setup failed) |
| fixture-matrix | empty target list to `test -s` | exit 1 |
| fuzz-smoke | `-list '^ZZZNoSuchFuzz'` → zero `^Fuzz` lines | `grep`/`test -s` exit 1; real lists yield 8+4+1 |
| fuzz-smoke | count pin against 99 | `test … -eq 99` exit 1; real total is 13 |
| fuzz-smoke | `-fuzz='^FuzzNoSuchTarget$'` exits 0 with "no fuzz tests to fuzz" | `^fuzz: elapsed` guard exit 1; all 13 real targets show the elapsed line |
| fuzz-smoke | secconftest leg vs derivation | entry-drop mutant fails `TestListedTargetsMatchDerivation` and `TestFuzzDriversReachProduction` |
| capability-claims | claim appended to real README | `TestRealREADMECarriesNoPositiveClaim` FAIL (README restored byte-clean, sha256 verified) |
| capability-claims | bare-prose claim (`The fifo capability is available…`, no backticks) | admitted by design — the stated bound, pinned by `TestUnbacktickedProseIsOutsideTheScanner` |
| windows-compile-only | scratch `syscall.Mkfifo` under `GOOS=windows` | exit 1 (`undefined: syscall.Mkfifo`) |
| gates-verdict | simulated `{"lint": {"result": "skipped"}}` | `jq -e` exit 1; all-success exit 0 |

## Gates (candidate tree worktree, exit codes)

- `go test ./... -count=1` → exit 0, 22 packages ok, 0 FAIL
- `go test -race ./... -count=1` → exit 0, 22 ok, 0 DATA RACE
- `go test ./... -cover -count=1` → exit 0, 22 `coverage:` lines, 0 `no test files` (cigate 90.5%)
- `go vet ./...` → exit 0; `GOOS=windows go vet ./...` → exit 0; `GOOS=windows go build ./...` → exit 0
- `gofmt -l internal` → empty; `go generate ./internal/catalog` + diff → clean
- `go run ./internal/traceability/cmd/tracecheck` → exit 0 (contracts=60, compatibility=55, cases=94, unchanged)
- `TestCIWorkflowInvokesTraceabilityGate` → PASS (all four literals present after the F1/F5 edits)
- CI YAML parses; 10 jobs, 9 non-verdict all in `needs`, `timeout-minutes` on all 10, exactly one `if:` (`always()` on `gates-verdict`)

## Bounds and follow-ups for the orchestrator (prose, no board elements created)

1. Availability is host-local: a positive claim about a here-available
   capability is admitted. Cross-host claim truth is outside the gate.
2. Only backticked capability IDs are scanned. A prose advertisement without
   backticks is admitted unchecked — stated in `doc.go`, the README row, and
   pinned by test.
3. Live probe outcomes are read by no call site; the check posture is
   all-probes-unavailable, which is stronger and host-independent.
4. Coverage has no floor by requirement; adding one needs its own observed
   rejection first.
5. Windows green means compiled+vetted, never executed. macOS/Windows hosted
   execution beyond the fixture matrix (provider suites, §19.3–19.4
   lifecycle) belongs to later stories with real lanes.
6. `task-board.config.json` validation commands were not extended: no new
   Fuzz targets were added, so `TestConfiguredValidationRunsEveryFuzzTargetWithFixedBudget`
   still covers its 8 secconftest targets; the 5 canonicaljson/scalar targets
   are smoked by the fuzz-smoke job, not by that validation command.
7. Catalog freshness vs `//go:generate` → `// go:generate` (token preserved)
   is pre-existing at HEAD, not introduced here; the contract-version gate
   still catches version drift with the directive disarmed. Do not lose this.
8. Suggested (not created): a follow-up leaf owning hosted macOS/Windows
   lane evidence and the §19.5 release-acceptance checklist when product
   lanes exist.
