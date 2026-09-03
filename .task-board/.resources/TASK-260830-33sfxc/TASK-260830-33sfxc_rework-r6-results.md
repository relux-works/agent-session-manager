# TASK-260830-33sfxc — round-6 rework results

Revision 6 of the leaf, at amended head `e708c4f`, one commit past its
checkpoint `f34d91d`. **No production code changed.** The round-5 blocking
finding and its non-blocking observation were both evidence defects on guards
that were already correct.

## Round-5 verdict: both items closed

### F1 (BLOCKING) — code-to-exit-status agreement had delete-only coverage

`internal/axerror/decode.go:285`, `if exitCode != expectedExit {`.

Reproduced at the reviewed head `1d0d702` before any test was written:

| Mutant | At reviewed head | At this head |
| --- | --- | --- |
| `if false && exitCode != expectedExit` (delete-only control) | KILLED | KILLED |
| `&& code != "policy_refused"` | **SURVIVED**, `go test ./... -count=1` exit 0 | **KILLED** |
| `&& code != "authentication_failed"` | **SURVIVED**, exit 0 | **KILLED** |
| `&& code == "observation_gap"` | **SURVIVED**, exit 0, 13 packages `ok` | **KILLED** |
| `&& version == Version100` (added here) | not measured | KILLED |

Closed by two tests.

`TestTheCodeToExitStatusAgreementIsMeasuredOverTheWholeCodeRegistry`
(`internal/axerror/agreement_test.go`) drives every registered code of every
registered version at every registered Section 15.2 failure status through the
production entry point `axerror.Decode`:

| Version | Registered codes | Failure statuses | Admitted (own status) | Refused by this guard |
| --- | ---: | ---: | ---: | ---: |
| `1.0.0` | 47 | 17 | 47 | 752 |
| `1.1.0` | 66 | 17 | 66 | 1,056 |
| `1.2.0` | 94 | 17 | 94 | 1,504 |
| `1.3.0` | 109 | 17 | 109 | 1,744 |
| total | 316 | 17 | 316 | 5,056 |

Every refusal must carry this guard's own sentence `maps to exit %d, document
carries %d`, so a row settled by `decodeExitStatus`, by the closed member set,
or by the retryability gate is not counted as coverage of it. The agreeing row
is required to be admitted as the positive control. Every figure is re-derived
from `CodesFor` and `CodesByExitStatus` and the per-version totals are pinned,
so the loop cannot go vacuous; the harness narrows the code loop to one entry
to prove that.

`TestRelabellingACodeOutOfItsExitClassCannotForgeARetryClaimThroughRead`
(`internal/cliresult/relabel_test.go`) closes the bypass at `cliresult.Read`,
the entry point a machine client calls. The exit-keyed retryability refusal
covers 3 classes (7, 16, 130); Structured Error 1.0.0 registers 7 codes there,
0 of which are also disqualified by name; each is relabelled into all 14
statuses that carry no exit-keyed refusal, with `retryable: true`. 98 refusals
required, each carrying the agreement guard's sentence. The control in the same
loop requires the same code at its own status with `retryable: true` to be
refused by the retryability gate, which is what makes the relabelling a bypass
of something rather than a rewrite of nothing.

### Observation — `decodeExitStatus` registered-status admission was sampled

`internal/axerror/decode.go:356`, driven at `{0, 1, 18, 99}` only.

| Mutant | At reviewed head | At this head |
| --- | --- | --- |
| `&& status != 42` | **SURVIVED**, exit 0 | **KILLED** |
| `if false && !IsFailureExitStatus(status)` (delete-only control) | KILLED | KILLED |

`TestTheExitStatusAdmissionIsSweptOverEveryByteValue` sweeps 0..255 and requires
admission if and only if the Section 15.2 table registers the status as a
failure — 17 admitted, 239 refused with `ErrUnregisteredExit` naming the status
— plus six values outside the byte range (`-1, -130, 256, 1000, 65536,
2147483647`).

Two design points, both stated as bounds rather than assumed:

- The oracle is `ExitStatusMeaning`, not `IsFailureExitStatus`. The latter *is*
  the decision under test; an oracle built from it would move with the mutant.
  The asserted row count of 17 is what catches a mutation that moves both.
- The document carries a code the registry does not register, which takes the
  Section 15.3 unknown-code branch and skips the agreement guard entirely, so
  exactly one decision is left in the row. The test first asserts the code is
  still unregistered, so it cannot stop covering its class in silence.

## Reviewer items deliberately not acted on

- The three equivalent survivors the round-5 verdict recorded (`ExitCodeFor`
  version scoping spared for `terminal_backend_capability_unproven`; the F1
  guard spared for `session_not_found` or `provider_auth_smoke_failed`) are
  left alone. None of the three is a registered code, so none reaches the
  mutated branch. Same for the two round-2 and two round-3 dismissals.
- The inherited README sentence "Every gate above has a negative test that
  narrows it rather than deleting it" is true again now that F1 is closed. The
  paragraph now also says explicitly that it was published before it was true
  of the reader, and points at where the two sweeps live.

## Validation — every command run standalone on this tree, real exit codes

| Gate | Command | Exit | Result |
| --- | --- | ---: | --- |
| Format | `gofmt -l ./internal` | 0 | no files |
| Build | `go build ./...` | 0 | clean |
| Vet | `go vet ./...` | 0 | clean |
| Full suite | `go test ./... -count=1` | 0 | 13 packages `ok` |
| Coverage | `go test ./... -cover -count=1` | 0 | cliresult 95.5%, axerror 99.5% — both unchanged |
| Traceability | `go run ./internal/traceability/cmd/tracecheck` | 0 | `acceptance_cases=74`, `clauses_discharged=17/403` — both unchanged |

Registration probe: renaming
`TestRelabellingACodeOutOfItsExitClassCannotForgeARetryClaimThroughRead` makes
tracecheck fail with `declaration ... is absent from
"internal/cliresult/relabel_test.go"`, ahead of the projection digest check. The
digest gate also fired on the registry edit before it was re-pinned
(`0c8a3014… -> 5563e4b2…`).

## Mutation harness

`.temp/TASK-260830-33sfxc/r6/mutants-r6.py`, extended from the round-5 harness:
**63 mutants, 62 killed, 1 declared subsumed, 0 unexplained survivors, 44
narrowing** by the harness's own definition, suite restored green
(`restored_suite_exit_code: 0`). Each mutant is verified present on disk
carrying the MUTATED text and compiled before it is measured, and the original
is restored afterwards.

Ten mutants added this round: five on the F1 guard (one delete-only control,
four narrowings), two on `decodeExitStatus` (one delete-only control, one
narrowing), and three test-side narrowings that prove the new sweeps fail closed
rather than shrinking — the failure-status enumerator dropped past exit 130, the
code loop narrowed to one entry, and the relabel destination loop narrowed to
one status.

**STATED BOUND, unchanged from previous rounds.** These harness figures are not
measured by any committed artifact. The harness mutates the working tree, so it
is task-scoped evidence under `.temp/` and attached here rather than run by
`go test ./...`. README states them as a bound where it publishes them.

## Scope

| Path | Change |
| --- | --- |
| `internal/axerror/agreement_test.go` | new — F1 registry sweep + 0..255 exit-status sweep |
| `internal/cliresult/relabel_test.go` | new — the bypass closed at `cliresult.Read` |
| `internal/traceability/ownership.v0.5.0.json` | 3 test registrations on existing acceptance cases |
| `internal/traceability/traceability.go` | projection digest re-pinned |
| `README.md` | F1 measurement, the bypass, both sweeps; harness figures 53→63 and 36→44 narrowing |
| `LOGBOOK.md` | entry 1447 |

No production source file changed. `git diff --stat 1d0d702..e708c4f` is 6
files, 596 insertions, 7 deletions.
