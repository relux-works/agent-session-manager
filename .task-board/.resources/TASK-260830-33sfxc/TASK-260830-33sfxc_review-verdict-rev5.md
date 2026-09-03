# TASK-260830-33sfxc — review verdict, CR revision 5

**Verdict: changes requested.** One blocking finding. The round-4 blocking
finding is closed, and every other gate I attacked in this leaf's own new code
held.

- Reviewed delta: `d1d3ece..42898ceb` (candidate tree), 69 files.
  `git rev-parse HEAD^{tree}` = `42898ceb9945bf52251e846548b7bec9826d9085`, so
  the reviewed bytes are the candidate bytes.
- Working tree was returned byte-identical to the candidate after every probe
  below (`git status --short` empty, tree hash re-verified).

## What was re-run here, not accepted from the attached logs

| Check | Command | Result |
| --- | --- | --- |
| Build | `go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Format | `gofmt -l .` (excluding `.temp`) | no files |
| Full suite | `go test ./... -count=1` | exit 0, 13 packages |
| Traceability | `go run ./internal/traceability/cmd/tracecheck` | exit 0; `acceptance_cases=74`, `clauses_discharged=17/403`, `normative_sections=36`, `bindings=49` — matches the published figures |

The attached `…_mutation-report-r5.json` (53 mutants) was read but **not** taken
as the coverage measurement. 29 independent mutants were written and run against
this tree; every one was verified present on disk carrying the MUTATED text and
compiled before measurement, and the original was restored and re-verified after
each. Report: `…_review-rev5-independent-mutation.json`.

## Round-4 finding: CLOSED

Required by the previous verdict — both surviving command-agreement narrowings
must now be KILLED. Re-run here against `go test ./... -count=1`:

| Mutant | Round 4 | Round 5 |
| --- | --- | --- |
| `result.Command() != command && command != CommandList` | SURVIVED | **KILLED** |
| `result.Command() != command && result.Command() != CommandTakeover` | SURVIVED | **KILLED** |
| `result.Command() != command && command != CommandDoctor` (control) | KILLED | KILLED |

Both die in `TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus` and
in the new `TestTheCommandAgreementGuardOwnsAMeasuredShareOfTheTagVocabulary`.
The 44x44 grid is a real measured ratio, not prose.

## BLOCKING — F1: the code-to-exit-status agreement in `axerror.decodeBody` has delete-only coverage, and a forged retry claim reaches through it

`internal/axerror/decode.go:285` — `if exitCode != expectedExit {`, the guard
that refuses a Structured Error whose `code` and `exit_code` name different
Section 15.2 classes.

### Measured, against the full 13-package suite

| Mutant | Result |
| --- | --- |
| `if false && exitCode != expectedExit` — delete the gate (control) | **KILLED** |
| `&& code != "policy_refused"` — spare one registered code | **SURVIVED**, `go test ./... -count=1` exit 0 |
| `&& code != "authentication_failed"` — spare a different registered code | **SURVIVED**, exit 0 |
| `&& code == "observation_gap"` — restrict the gate to exactly the one code its coverage drives | **SURVIVED**, exit 0, all 13 packages `ok` |

The third row is the shape. The whole class is covered by one row of
`TestDecodeRefusesClosedShapeViolations`:

```
{name: "exit status that contradicts a registered code",
 document: strings.Replace(conformingDocument, `"exit_code": 9`, `"exit_code": 5`, 1)}
```

One code (`observation_gap`), one wrong status (5). Measured size of the class
this guard owns, derived here from the production enumerations `CodesFor` and
`CodesByExitStatus`:

| Version | Registered codes | Failure statuses | (code, wrong-status) ordered pairs |
| --- | ---: | ---: | ---: |
| `1.0.0` | 47 | 17 | 752 |
| `1.1.0` | 66 | 17 | 1056 |
| `1.2.0` | 94 | 17 | 1504 |
| `1.3.0` | 109 | 17 | 1744 |

1 of 752. The guard is proven reachable and is unmeasured over its class — the
same shape the round-4 verdict blocked on for the command-tag agreement, one
guard along the same reading path.

### It is a bypass path, not only a coverage gap

The guard is the sole owner of this refusal — probed at the unmutated tree:

```
Decode(1.1.0, policy_refused @ exit 9)      err = invalid structured error: "policy_refused" maps to exit 16, document carries 9
Decode(1.1.0, transaction_unknown @ exit 9) err = invalid structured error: "transaction_unknown" maps to exit 12, document carries 9
```

Nothing else refuses the pairing. And the retryability refusal that this package
publishes as a gate keys on the **exit status**
(`retryabilityRefusalsByExitStatus`: 7 authorization, 16 refusal, 130 interrupt),
so relabelling a code out of its own class disarms it. Driven through the
production entry point `cliresult.Read`, with only the `code == "observation_gap"`
narrowing applied and nothing else changed:

```
document: {"schema":"urn:ax:schema:error","schema_version":"1.0.0",
           "code":"authentication_failed","message":"m",
           "exit_code":9,"retryable":true,"details":{}}
Read(CommandList, stdout=document, ExitStatus=9)
  -> ADMITTED: code="authentication_failed" exitStatus=9 retryable=true
```

`authentication_failed` is registered at exit 7, whose Section 15.2 meaning is
"Authentication/authorization/allowlist failure" and whose whole class this
package forbids from claiming `retryable: true` — the refusal
`TestRetryableRefusedForEveryForbiddenClass` exists for. Moving the document's
`exit_code` to 9 walks around it, and the machine client is handed a `Reading`
whose code and exit status name different classes with a safe-retry claim
attached. The entire repository suite stays green.

This is the leaf's own contract, not a neighbouring package's business. `Read`'s
design is "the document decides the outcome and the exit status corroborates
it"; the corroboration `readFailure` performs (`exit_code` equals the process
status) is swept, and this second corroboration — that the code and the
`exit_code` belong to the same class — is the one holding the pair together.
`CodesByExitStatus`, added by this leaf, and the fan-in this leaf quotes inside
`exitStatusIsNotEnough` describe a code-to-status mapping that only exists at
read time because this guard enforces it.

The revision's own results doc states bounds for the harness figures, the
payload-size narrowings, and the dismissed equivalent survivors. It states no
bound for this guard: it is unmeasured rather than knowingly limited.

**What would close it.** The pattern is already in this leaf. Mirror
`TestTheCommandAgreementGuardOwnsAMeasuredShareOfTheTagVocabulary`: for every
registered version, drive every registered code against every registered failure
status other than its own, require `ErrInvalidStructuredError` **and** this
guard's own sentence `maps to exit %d, document carries %d` (so a pair settled by
`decodeExitStatus` or the retryability gate does not count as coverage of this
guard), require the agreeing pair to be admitted as the positive control, and
assert the counts against `CodesFor`/`CodesByExitStatus` so the loop cannot go
vacuous. Test-only; no production code needs to move.

## Non-blocking observations

- **Sampled, not swept — worth closing in the same pass.**
  `axerror.decodeExitStatus`'s `!IsFailureExitStatus(status)` refusal is driven
  at `{0, 1, 18, 99}` only. `&& status != 42` SURVIVES the full suite. The
  registered side is swept
  (`TestTheRetryDecisionFollowsTheDocumentAtEveryFailureStatus`), so this is
  weaker than F1, and the complement is unbounded — but a sweep over `0..255`
  asserting admitted-iff-`IsFailureExitStatus` is cheap and turns it into a
  measured ratio. Reachable without a process: `DecodeBound` reads
  peer-supplied provider, bridge, RPC, session-adapter and terminal-backend
  envelopes, where no exit status corroborates the member.
- **Equivalent mutants — do not chase them in round 6.** Three of my survivors
  are equivalent and I am recording them so the next round does not re-derive
  them: `ExitCodeFor`'s version scoping narrowed to spare
  `terminal_backend_capability_unproven`, and the F1 guard narrowed to spare
  `session_not_found` or `provider_auth_smoke_failed`. None of those three is a
  registered code, so none reaches the mutated branch. The version-scoping gate
  itself is genuinely covered: the same narrowing on the registered
  `terminal_backend_stale_generation` is **KILLED** by
  `TestExitCodeForRefusesVersionAndCodeDrift`.
- **Inherited README sentence, now measurably false.** `README.md` (written by
  leaf 1, `ebc4e31`, untouched by this leaf) states "Every gate above has a
  negative test that narrows it rather than deleting it". F1 is a
  counterexample. Not attributed to this leaf, but it stops being true the
  moment F1 is published, and closing F1 makes it true again.

## Verified holding under attack (killed independent mutants)

This leaf's own new production code in `internal/cliresult/client.go` was
attacked directly and did not yield:

| Independent mutant | Killed |
| --- | --- |
| `Read`'s unregistered-exit-status gate narrowed to spare status 42 | yes |
| `readSuccess`'s success-status gate narrowed to spare exit 7 | yes |
| `readFailure`'s `exit_code`-equality gate narrowed to spare exit 7 | yes |
| `documentSchema`'s absence gate narrowed to `len(stdout) == 0` (whitespace-only stdout becomes a read failure instead of an absence) | yes |
| `documentSchema`'s trailing-content gate removed | yes |
| `documentSchema`'s common-data-model gate narrowed to spare Structured Error documents | yes |
| `exitStatusIsNotEnough` reporting the fan-in count off by one | yes |
| `axerror.requireCommonDataModel` skipped for one bound version (`1.1.0`) | yes |
| `verifyEnvelopeIdentity`'s schema gate widened to admit `urn:ax:schema:cli-result` | yes |
| `RetryabilityRefusal` narrowed to spare `operation_uncertain` | yes |
| `RetryabilityRefusal` narrowed to spare exit class 7 | yes |
| `ExitCodeFor` version scoping narrowed to spare a real 1.3.0-only code | yes |
| `ExitCodeFor` version scoping deleted (control) | yes |

Also checked directly and found sound:

- **Historical corpora are frozen in both directions.** Deleting
  `error-1.1.0-session-adapter-timeout.json` reddens
  `TestHistoricalErrorCorpusIsFrozenAndCoversEveryBoundVersion`; adding an
  unlisted fixture reddens the same test naming the extra file. Neither an
  unread fixture nor a silently dropped one stays green.
- **README figure pins are live.** Perturbing the `1.2.0` fan-in row to the
  wrong values it was first published with reddens
  `TestREADMEFanInTableIsDerivedFromTheMeasuredProjection` with the measured
  cell quoted back.
- **Message-independence replays are not vacuous.** Both corpora assert the
  rewrite changed bytes, assert the message actually moved, and assert the
  replay count against the reviewed corpus (`6` errors, `5` CLI failures), so a
  shrinking corpus cannot pass quietly.
- `InvocationOutput` structurally cannot carry stderr, and the Section 17.2
  rebinding is reported as unevidenced 0/1 with a stated gap rather than
  claimed.

## Routing

`to-dev`. One sweep test closes F1; the `0..255` exit-status sweep closes the
observation in the same pass. No production code needs to move. Re-review must
re-run the three F1 narrowings above and require all three KILLED, with the
delete-only control still KILLED.
