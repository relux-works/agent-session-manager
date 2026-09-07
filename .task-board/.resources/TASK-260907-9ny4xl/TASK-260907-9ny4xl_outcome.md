# TASK-260907-9ny4xl outcome — correct-cigate-mechanism-claims-and-measure-marker-effect (story final leaf)

Status: ready for review (handed off to review; review/integration are orchestrator steps).
No board element was created under STORY-260905-3t31e9.

## Changed paths (exact CR set, 7 files)

- `.github/workflows/ci.yml`
- `LOGBOOK.md`
- `README.md`
- `internal/catalog/catalog_test.go`
- `internal/cigate/claims.go` (comments only — no behavior change)
- `internal/cigate/claims_test.go`
- `internal/cigate/doc.go`

Leaves 1–5 production untouched: `git diff --name-only` shows only the
7 paths above; no provider/provhost/terminalbackend/invcore/environ/
sessadapter/dirnode/scalar/secprim/secconftest production file differs
from the checkpoint.

Candidate tree OID (detached-index `write-tree`, after the last
in-worktree artifact): `1ed479a0c1228b651f06fa2f9c8a80350a304fe6`
(2724 blobs; all 7 changed files byte-match the worktree).
Base checkpoint: `613cbd7`. Work left UNCOMMITTED for the board CR
(`story_final` derivation is the orchestrator step).
Worktree root: no untracked, ungitignored scratch file (`git status`
shows only the 7 `M` entries; no `.bak`/`.log`/`.clean` in root).

## AC coverage: 5 of 5 rows driven through production call sites

| AC row | Production call site | Named committed test |
| --- | --- | --- |
| 1. mechanism claims in ci.yml, doc.go, README.md, LOGBOOK describe the code, proven by divergence-failing tests | `cigate.CheckAdvertisements` (claims.go:226); CI derive/freshness steps | `TestAvailabilityFieldDrivesVerdict` (same claim refused unavailable / admitted available — hardcoding either direction reddens it); `TestEveryNonClaimMarkerClassifiesCorpusSentence` (README cell "negated or marker-classified" machine-checked per mention); R2 shell semantics re-measured by executing the derive step + empty-leg replica (transcript below); directive form pinned by `TestGenerateDirectiveStaysRecognizedForm` + the freshness-step guard |
| 2. marker census measures effect, fails closed on a non-classifying marker | `cigate.CheckAdvertisements` + production `containsAnyFold`/`wordMatches` | `TestEveryNonClaimMarkerClassifiesCorpusSentence` (per-mention arm from production predicates, cross-checked against the production verdict; per-marker sole-classifier necessity) |
| 3. sentence-splitter narrowing mutant dies | `cigate.CheckAdvertisements` → `splitSentences` | `TestSameBlockSentencesSplitOnTerminal` (period/bang/query same-block rows) |
| 4. catalog-freshness step rejects the token-preserving `//go:generate` → `// go:generate` rewrite | CI step runs `go generate ./internal/catalog` + the new anchored guard; pin reads `internal/catalog/catalog.go` | `TestGenerateDirectiveStaysRecognizedForm` + the step guard (both kill; catalog/cataloggen otherwise green under the mutant) |
| 5. every stated bound names its escape and survives direct attack | bounds in README/ci.yml/LOGBOOK (see below) | R5 re-measured this session (13 smokes); R6 answered by observed PR #36 checks; traceability Contains-blindness re-attacked this session (Vet-step-to-comment keeps `TestCIWorkflowInvokesTraceabilityGate` green; ci.yml restored byte-identical) |

Negative tests (gate admits what it must reject → red): `TestPositiveClaimsAreRefused`
(5 verbs + capital), `TestUnclassifiedMentionIsRefused`, `TestEmptyInputsAreRefused`,
`TestSubstringIsNotAMarker`, `TestUnbacktickedProseIsOutsideTheScanner`,
`TestMixedAvailabilityFindsOnlyUnavailable`, all via `CheckAdvertisements`.

## What changed (R1–R6 + carried defect)

- R1: ci.yml capability-claims comment + LOGBOOK 2249 no longer state
  `ProbeState.Available` is dead. Both now state what `doc.go` states
  (sharpened): the gate reads `Available` from its input states while the
  live probe outcome reaches no availability reader (`ProbeStates` feeds
  only `CapabilityIDs`, which drops availability).
- R2: derive-targets comment now names the total-count pin as the
  fail-closed guard (grep refuses nothing — pipeline status discarded;
  `test -s` refuses only the every-package-empty case). Neighbouring steps
  audited: PASS-guard loops (`test $code` + `grep -q`, fail under
  `set -e`), gofmt/coverage/fixture count guards, target-match diff via
  test exit, smoke `elapsed` grep as loop-body tail under `set -e` — no
  same mistake.
- R3: occurrence census replaced by effect census; 20 probe notes added
  (one sole-classifier per marker); measured arm distribution over the 22
  corpus mentions: negation-admitted 2 / marker-admitted 20 / refused 0.
  README claim cell corrected.
- R4: splitter behavioural rows added; X1 dies (was a survivor).
- R5: fuzz-budget bound stated in the README row + smoke-step comment
  with its escape (budget lengthening / corpus shrink moves it silently).
- R6: Linux-execution unknown restated as ANSWERED — PR #36 merged
  2026-09-06 with `Conformance fixtures (ubuntu-latest,
  internal/secprim)` and `(ubuntu-latest, internal/secconftest)` passing
  in both runs, macOS legs and all other jobs green (observed via
  `gh pr checks 36` this session).
- Carried: catalog-freshness token-preserving rewrite now refused by the
  step guard (exit 1, measured) and the committed pin; defeat reproduced
  first (old step exits 0 on the rewrite).
- Recorded (pre-existing, traceability scope, not fixed):
  `TestCIWorkflowInvokesTraceabilityGate` blindness re-attacked (PASS on
  the blinded tree), named in LOGBOOK beside the `go:generate` one.

## Mutation battery: 14 applied / 14 killed / 0 survivors

Denominator (production-derived from this CR's gates): 2 availability
postures + 20 marker arms + 3 splitter terminals + 5/13 verb-negation
sample rows + 3 refusal arms + 1 directive form. Full package suites per
mutant (never a `-run` mask); cp-aside/cp-back with byte-identical
restore verified every row. Harness: `/tmp/mutbattery_9ny4xl.py`,
log `TASK-260907-9ny4xl_mutants.log`.

| Mutant | Narrows the gate to | Named failing test | Status | Killed by |
| --- | --- | --- | --- | --- |
| N-avail-false (narrowing) | available posture admits; all-available README proof refuses | `TestAvailabilityFieldDrivesVerdict` (+2 co-kills) | KILLED | behavioral |
| N-avail-true (narrowing) | unavailable posture admits; positive claims pass unchecked | `TestAvailabilityFieldDrivesVerdict` (+6 co-kills) | KILLED | behavioral |
| N-marker-drop-uid (narrowing) | its note loses its sole classifier, refused unclassified | `TestEveryNonClaimMarkerClassifiesCorpusSentence` + `TestRealREADMECarriesNoPositiveClaim` | KILLED | behavioral |
| N-marker-drop-scan (narrowing) | same, second sample | `TestEveryNonClaimMarkerClassifiesCorpusSentence` + `TestRealREADMECarriesNoPositiveClaim` | KILLED | behavioral |
| N-split-X1-paragraph (narrowing) | one sentence per block; negated first covers positive second | `TestSameBlockSentencesSplitOnTerminal` (3/3 rows) | KILLED | behavioral |
| N-split-dot-only (narrowing) | bang/query terminals stop splitting | `TestSameBlockSentencesSplitOnTerminal` (bang+query rows) | KILLED | behavioral |
| N-verb-drop-works (narrowing) | works-claims admitted; other verbs still refused | `TestPositiveClaimsAreRefused` + splitter reason check | KILLED | behavioral |
| N-negation-drop-only (narrowing) | only-conditional refused as positive | `TestNegatedClaimsAreAdmitted` | KILLED | behavioral |
| N-negation-add-tomorrow (narrowing) | unclassified arm admits exactly the tomorrow-mention | `TestUnclassifiedMentionIsRefused` (+covers-gate row) | KILLED | behavioral |
| G-directive-space (narrowing, token-preserving) | directive disarmed, generator skipped, freshness diff clean | `TestGenerateDirectiveStaysRecognizedForm` (catalog/cataloggen otherwise green) | KILLED | behavioral |
| X3-marker-list-dropped-from-arm (arm-deletion) | marker admission arm removed; all 20 notes refused | `TestEveryNonClaimMarkerClassifiesCorpusSentence` + `TestRealREADMECarriesNoPositiveClaim` | KILLED | behavioral |
| A-positive-arm-deleted (arm-deletion) | positive claims slide to the unclassified reason | `TestPositiveClaimsAreRefused` (+3 co-kills) | KILLED | behavioral |
| A-unclassified-arm-deleted (arm-deletion) | unclassified mentions admitted | `TestUnclassifiedMentionIsRefused` + `TestSubstringIsNotAMarker` | KILLED | behavioral |
| N-marker-add-operational (census-only) | gate unchanged, behavioral green; added marker classifies nothing | `TestEveryNonClaimMarkerClassifiesCorpusSentence` | KILLED | census |
| Harness | Purpose | Status |
| --- | --- | --- |
| H-notapplied | zero-match anchor never written | NOT_APPLIED |
| H-compilefail | arity break must fail the build, never pass as survivor | COMPILE_FAIL |
| C-old-corpus (control) | pre-fix README against the new census: 20 markers classify nothing | KILLED (census) |

Survivors: none. No bound needed for a survivor.
Audit-only: 0 — no AST audit in this CR's scope (inherited invcore
audits green via the full suite). Every gate in scope ships a narrowing
mutant: availability (N-avail ×2), marker arm (N-marker-drop ×2,
N-marker-add census-only), positive arm (N-verb-drop), negation arm
(N-negation-drop), unclassified arm (N-negation-add), splitter
(N-split ×2), freshness directive (G-directive-space).

## R2 replica transcript (empty fuzz leg)

`secconftest` (8 targets) + fuzzless `axerror` (0 targets) through the CI
derive shape: grep refused nothing, `test -s` PASSED with 8 lines despite
the empty leg, the `-eq 13` count pin REFUSED with exit 1. Real derive
step verbatim: 13 targets, green.

## Gates (each run directly, exit codes observed this session)

| Command | Exit | Evidence |
| --- | --- | --- |
| `go build ./...` | 0 | clean |
| `go test ./... -count=1` | 0 | 23 `^ok`, 0 `^FAIL` |
| `go test -race ./... -count=1` | 0 | 23 `^ok`, 0 `DATA RACE` (full repo) |
| `go test ./... -cover -count=1` | 0 | 23 coverage lines; cigate 90.5%, secprim 94.4%, secconftest 92.6% |
| `go vet ./...` | 0 | clean |
| `GOOS=windows go vet ./...` | 0 | clean |
| `gofmt -l internal/` | 0 | empty |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | 60 contracts / 98 cases, unchanged |
| catalog freshness (`go generate` + `git diff --exit-code`) + directive pin | 0 | clean, pin count 1 |
| capability-claims step verbatim | 0 | 6/6 `--- PASS` guards |
| contract-preservation step verbatim | 0 | 6/6 `--- PASS` guards |
| fuzz-smoke loop verbatim (13 targets) | 0 | 13/13 `fuzz: elapsed`; 5 baseline-only, 8 `now fuzzing` |
| `gh pr checks 36` (R6 evidence) | 0 | ubuntu fixture legs pass ×2 runs, all jobs green |

Not run: nothing CI runs was skipped. (CI's macOS legs execute on the
hosted runner at PR time, as before.)

## Follow-up needs (said here, not on the board)

1. The `//go:generate` pin counts exactly 1 directive in
   `internal/catalog/catalog.go`: adding a second directive must update
   the CI guard and `TestGenerateDirectiveStaysRecognizedForm` together
   (both messages say so). Fail-closed by design.
2. The marker census requires one sole-classifying corpus sentence per
   marker: future probe-vocabulary words need a README note each, or the
   census names the dead marker. Removing a marker instead is the
   stricter direction and stays green only if its note goes too (else the
   real-README gate refuses it).
3. `TestCIWorkflowInvokesTraceabilityGate` stays a substring gate until
   the traceability package owns a stricter check (their scope).
4. The R5 bound is scoped to `-fuzztime=100x` and current corpora;
   re-measure if either moves.
