# TASK-260830-33sfxc — round-4 rework results

Signed head `b43a746` (amend of `1d68325`; one commit past checkpoint `f34d91d`;
`git verify-commit` clean; tree `2e893d0d9b476991e481098bafff9b685374aa18`).

**No production code changed.** The one blocking finding and both acted-on
observations were evidence defects, not behaviour defects.

## F1 (blocking) — README published 73 acceptance cases; the gate measures 74

`README.md` stated "73 executable acceptance cases" while
`ownership.v0.5.0.json` holds 74 entries with 74 distinct ids and `tracecheck`
prints `acceptance_cases=74`. Nothing measured it: the round-3 reviewer set the
figure to 999 and both `go test ./...` and `tracecheck` stayed at exit 0.

Third occurrence of the class in this artifact, the first two recorded by this
leaf itself (LOGBOOK 1015, 1157). The tool for the class —
`parsePublishedFanIn` at `internal/axerror/exit_table_pin_test.go:148` — existed
and had never been pointed at the neighbouring paragraph.

### What was done

1. Figure corrected to 74.
2. `TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport`
   (`internal/traceability/readme_figures_pin_test.go`) locates the paragraph by
   its heading, refuses a heading found twice or not at all, unwraps it, and
   re-derives every figure from `traceability.VerifyRepository`.
3. **All seven figures** of the sentence are pinned, not only the one that
   drifted: contracts 60, normative sections 36, acceptance cases 74, section
   bindings 49, unowned sections 2, fixtures 30, compatibility contracts 55.
   The other six were correct today and unmeasured for exactly the same reason.
4. Each figure is located by the **noun phrase naming what it counts**, so
   re-ordering the clause cannot re-point a pin at a different number.
5. The pin polices its own completeness: every digit run in the paragraph that
   no pinned phrase consumed is reported as unmeasured, with only dotted version
   and path tokens exempt under a deliberately narrow rule (`v`-introduced, or
   preceded by a dot that itself follows a digit). **Deleting a row from
   `publishedOwnershipFigures` is therefore a red**, not a quietly smaller
   comparison.

### Mutants (all verified PRESENT on disk and compiled before measurement)

| Mutant | Result |
| --- | --- |
| `readme-ownership-acceptance-cases-restored-to-73` | KILLED |
| `readme-ownership-contract-rows-drifted` (60→61) | KILLED |
| `readme-ownership-normative-section-keys-drifted` (36→35) | KILLED |
| `readme-ownership-section-bindings-drifted` (49→48) | KILLED |
| `readme-ownership-unowned-sections-drifted` (2→3) | KILLED |
| `readme-ownership-fixture-identities-drifted` (30→29) | KILLED |
| `readme-ownership-compatibility-subset-drifted` (55→56) | KILLED |
| `readme-ownership-unmeasured-figure-added` | KILLED |
| `readme-ownership-pin-narrowed-by-one-figure` | KILLED |

The restore-73 mutant was reproduced by hand first, with the mutated text
confirmed present at README line 1643, and reddened with
`README ownership paragraph at line 1639 publishes 73 executable acceptance
cases; VerifyRepository measures 74`.

## Observation A (acted on) — the refusal's measured count was not measured

`exitStatusIsNotEnough` documents its own count as "a measured count rather than
as advice" and `README.md` repeated the claim; every assertion on those refusals
matched the trailing clause `identifies no failure` and nothing read the number.

`TestTheRefusalStatesTheMeasuredFanInRatherThanAdvice` drives **every registered
command at every registered Section 15.2 failure status** through the production
`Read` entry point, parses the refusal, and re-derives the count from
`axerror.CodesByExitStatus` for the version that command binds. The stated
status, the stated Structured Error version, and the count are each checked, and
the number of refusals read is asserted against `len(commands) * len(statuses)`.

| Mutant | Result |
| --- | --- |
| `refusal-fan-in-count-replaced-by-a-constant` | KILLED |
| `refusal-fan-in-count-replaced-by-advice` (`%d registered` → `many`) | KILLED |
| `refusal-fan-in-count-narrowed-to-one-status` (correct at exit 6 only) | KILLED |

The third is the point: it keeps the count measured at exactly the one status
the reviewer proposed asserting, so the minimal single-assertion fix would have
survived it. The sweep is what kills it.

## Observation B (acted on) — retry independence proven at one status

`TestExitStatusAloneDecidesNoRetry` drives exit 12 only, and the one exit-15
envelope any test read already declared `retryable: true`, so a fabricated
status-keyed branch returned the answer the document would have given.

`TestTheRetryDecisionFollowsTheDocumentAtEveryFailureStatus` drives **both
polarities across the whole failure domain**. The false arm covers all seventeen
registered failure statuses; the true arm covers fourteen, because the pinned
document forbids the claim for every code of the other three, and that figure is
asserted (`retryPermittedFailureStatuses`) rather than left implicit. The
constant was set from the measurement, which first reported 14 against a
provisional 17.

| Mutant | Result |
| --- | --- |
| `retry-forced-true-at-one-status` (exit 15) | KILLED |
| `retry-forced-true-over-a-status-class` (15, 20, 21, 22) | KILLED |
| `retry-narrowed-to-the-one-status-already-proven` (document decides at 12 only) | KILLED |
| `round-4-failure-status-enumeration-narrowed` (enumerator to ≤20) | KILLED |

## Observation C and the round-2 dismissals (left alone, with reason)

Not touched, deliberately. A survivor dismissed with a reason is evidence; one
quietly fixed hides why it survived.

- `Succeeded()` recomputed from `exitStatus`, and `ExitStatus()` returned from
  `failure.ExitCode()`: genuinely equivalent, because `readSuccess` and
  `readFailure` between them make document and status provably agree on every
  admitted reading.
- The near-equivalent `details` narrowing of `requireCommonDataModel` and the
  `CodesByExitStatus` empty-group pre-seed, dismissed in round 2, are unchanged.

## Published harness figures, and the bound on them

`README.md` published "A 28-mutant harness … kills 27" and "Fourteen … narrow
rather than delete". Both were stale and both were unmeasured prose of exactly
the class F1 is about, so both were corrected AND their bound stated:

- The harness mutates the working tree, so it is task-scoped evidence under
  `.temp/`, attached to this board item, and **not** run by `go test ./...`. No
  committed artifact derives its counts. README now says so where it publishes
  them, rather than letting a harness figure sit beside gated ones and read as
  though it were gated too.
- The "narrows rather than deletes" class is now **defined in the harness** —
  enforcement left in place for part of the domain, whether a restricted class,
  one of several redundant copies, a reordering, or a published claim kept and
  given a specific wrong value — and the count computed from that definition.
  The harness fails closed if the set names a mutant that does not exist.
  Applied to the 28-mutant harness the definition selects **exactly the fourteen
  that revision published**, so the round-4 figure of 27 extends the same class
  rather than redefining it.

## Traceability

Three tests registered, and the registration was **probed rather than assumed**.
Renaming each declaration makes `tracecheck` exit 1 naming the absent
declaration, ahead of the digest check:

| Registered test | Acceptance case | Rename → tracecheck |
| --- | --- | --- |
| `TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport` | `ownership-forgery-refusal` | exit 1, `declaration … is absent from "internal/traceability/readme_figures_pin_test.go"` |
| `TestTheRefusalStatesTheMeasuredFanInRatherThanAdvice` | `structured-error-exit-status-fan-in` | exit 1, absent from `internal/cliresult/client_test.go` |
| `TestTheRetryDecisionFollowsTheDocumentAtEveryFailureStatus` | `structured-error-exit-status-fan-in` | exit 1, absent from `internal/cliresult/client_test.go` |

The projection digest gate fired on the registry change as designed
(`f2bc887e… differs`) and was re-pinned to
`d9a860aa460c1bcc776fb1ca8b705af1713d358869fdb24ea796694f56ca307d`.
`acceptance_cases` stays **74** — tests were added to existing cases, so no new
case id exists and the README figure is unchanged by the registration.
`clauses_discharged` stays 17/403.

## Validation — each command run standalone, real exit status

| Command | Exit | Note |
| --- | ---: | --- |
| `gofmt -l ./internal` | 0 | no output |
| `go build ./...` | 0 | |
| `go vet ./...` | 0 | |
| `go test ./... -count=1` | 0 | 13 packages |
| `go test ./... -cover -count=1` | 0 | cliresult 95.5%, axerror 99.5%, traceability 86.6% |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `acceptance_cases=74` |
| `python3 .temp/TASK-260830-33sfxc/mutants-r4.py` | 0 | 44 mutants, 43 killed, 1 subsumed, 0 unexplained, 27 narrowing |

`gofmt -l ./internal ./cmd` was NOT run: this module has no root `cmd/`
directory and that form exits 2 on the missing path.

Coverage is unchanged for `cliresult` and `axerror` — the new rows drive
statements already executed. What moved is the class each guard is proven over.

## Anomaly worth recording

My first hand-probe of the `many` mutant reported a non-zero exit and I read it
as a kill. It was a BUILD failure: dropping the `%d` argument left `fanIn`
declared and not used. The harness's `NOT-COMPILED` arm caught it on the next
run, and the mutant was rewritten to compile. A mutant that does not compile
proves nothing about the test, and a non-zero exit status is not by itself a
kill. Recorded in LOGBOOK 1338.

## Reviewable diffs

The stored CR patch predates this amend.

- `TASK-260830-33sfxc_rework-r4-delta.patch` — `git diff --binary
  1d68325..b43a746`, the round-4 delta only: 6 files, 420 insertions, 4
  deletions.
- `TASK-260830-33sfxc_leaf-full-r4.patch` — `git diff --binary f34d91d..b43a746`,
  the whole story_final leaf.

Use these until a CR revision exists whose `candidate_tree_oid` equals
`2e893d0d9b476991e481098bafff9b685374aa18`.
