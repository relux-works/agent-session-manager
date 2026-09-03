# TASK-260830-33sfxc — Error and Result compatibility evidence

Commit: `b21fede` on `task-board/story/STORY-260830-2jylym`, exactly one commit
past checkpoint `f34d91d`. Working tree clean.

## What was built (production entry points)

| Entry point | File | What it decides |
| --- | --- | --- |
| `cliresult.Read(command, InvocationOutput)` | `internal/cliresult/client.go` | Classifies one completed `ax --json` invocation from stdout and the exit status |
| `cliresult.InvocationOutput` | `internal/cliresult/client.go` | The observation a machine client may depend on: exactly `Stdout` and `ExitStatus`, no stderr member |
| `cliresult.Reading` | `internal/cliresult/client.go` | The machine answers; `HumanMessage` is the only message-derived value and nothing else is computed from it |
| `axerror.CodesByExitStatus(version)` | `internal/axerror/registry.go` | Projects the code registry from exit status to codes; the measured fan-in the refusals quote |

## The deferred decision: which refusal wins when two apply

Leaf 2's review left this open. A document of a major the reader is not bound to,
carrying an unknown top-level member, satisfied two refusals; the member rule ran
first, so `cliresult` answered `unknown top-level member "new_in_v2"` and
`errors.Is(err, ErrUnsupportedMajor)` was false. `axerror` had the same shape and
answered `unknown field`.

**Decided: the envelope identity is settled first, in both readers.** The pinned
text scopes the member rule to the object it governs three times over — §1.6 "a
reader MUST reject an unknown top-level field in **a major version 1 object**",
§17.1 "**within any negotiated major version** … unknown top-level fields remain
an error", §17.2 rule 1 "rejects an unsupported major" listed first — and §15.1
states the posture outright: "receivers MUST NOT parse a different major's
payload far enough to trust its error code, retryable bit, details, or authority
fields". Reporting which members an unbound major's payload carries answers a
compatibility question with a structural claim about a document whose major was
never established, and a caller deciding "is my ax too old for this output"
cannot decide that from a member name.

Both orders refuse, so nothing was ever admitted either way and nothing is
admitted now:

| Document | Refusal |
| --- | --- |
| Unbound major, unknown member | `ErrUnsupportedMajor` (was: the member fact) |
| Unbound major alone | `ErrUnsupportedMajor` (unchanged) |
| Bound major, unknown member | unknown top-level member (unchanged) |
| Bound major, missing member | missing required member (unchanged) |
| Foreign schema, unknown member | schema refusal (unchanged) |

- `cliresult.decodeClosedDocument` split into `parseDocument` + `verifyClosedMembers`.
- `axerror` gained `parseEnvelopeIdentity`, which reads only `schema` and
  `schema_version` from the raw bytes before the closed decode.
- `cliresult.rawString` now separates "the member is absent" from "the member is
  not a string", because the identity check is where a missing `schema` is first
  observed.
- `TestUnsupportedMajorIsSettledBeforeTheClosedMemberRule` pins every row in both
  packages; `TestReorderingTheIdentityCheckAdmitsNothingItUsedToRefuse` re-drives
  13 previously-refused shapes per package; two harness mutants restore the old
  order and are killed.

### Found while doing it

`axerror.decodeClosedDocument`'s trailing-content guard became unreachable after
the reorder — `parseEnvelopeIdentity` settles it on the same bytes first. The
guard was **moved** rather than left as a branch no test could redden, which is
the call the 0742 logbook entry made about the leading-quote guard in
`decodeExitStatus`.

Both inherited packages were probed for the accessor-aliasing defect class before
their guarantees were used here: `Result.Body`, `Result.Extension` and
`Error.Detail` all deep-copy, and
`TestReadingAccessorsDoNotHandOutLiveInteriorState` mutates what each accessor
returns two containers deep and requires the object to be unchanged. Nothing
further was found unfixed in either package.

## The three claims and how each is proved

### 1. A machine client never depends on message text

`TestMessageTextChangesNoMachineAnswer` and
`TestHistoricalFailuresAreClassifiedWithoutTheirMessages` replay every readable
envelope with the message replaced — by another code's name, by a JSON object
(`{"code":"not_found","exit_code":4,"retryable":true}`), by 4,096 characters, by
non-ASCII — and require every machine answer to be identical while
`HumanMessage` changes. A row whose rewrite produced byte-identical output fails
rather than passing vacuously. `TestHistoricalErrorEnvelopesAreClassifiedWithoutTheirMessages`
does the same over the protocol-bound corpus.

### 2. A machine client never depends on stderr

- Structural: `InvocationOutput` has no stderr member, so `Read` cannot receive
  one. `TestMachineReadingCannotSeeStderr` asserts the member set by reflection
  and reddens the moment a member appears; its mutant adds `Stderr []byte` and
  is killed.
- Behavioural: `TestClassificationSurvivesDiscardedStderr` drives the production
  `Emitter` in JSON mode with logs and progress on stderr, classifies from
  stdout alone, and asserts stderr carries none of the machine facts.
- Mutant: `TestAMisroutedDocumentLeavesTheMachineClientWithNothing` swaps the
  two streams so the one JSON document lands on stderr. The machine client is
  left with `ErrAbsentDocument`, not a classification recovered from the status.

### 3. A machine client never depends on the exit status alone

The fan-in is measured, not argued:

| Structured Error version | Failure statuses | Registered codes | Statuses with >1 code | Largest class |
| --- | ---: | ---: | ---: | ---: |
| `1.0.0` | 17 | 47 | 14 | 6 codes at exit 6 |
| `1.1.0` | 17 | 66 | 14 | 9 codes at exit 6 |
| `1.2.0` | 17 | 94 | 15 | 12 codes at exit 6 |
| `1.3.0` | 17 | 109 | 15 | 15 codes at exit 6 |

`TestExitStatusAloneDecidesNoRetry` narrows the ratio to the decision a client
makes with it: under the `session.clone.*` binding, exit 12 carries both
`staging_incomplete`, which may claim `retryable`, and `transaction_unknown`,
which §15.1 calls "a parked ambiguous effect" and which may not. A client keying
on exit 12 would retry the parked one. All three arms are driven through `Read`.

The refusals quote the count rather than giving advice: `exitStatusIsNotEnough`
reports how many registered codes the bound version assigns to the observed
status.

### Refusal table (`Read`)

| Observation | Answer |
| --- | --- |
| No bytes on stdout | `ErrAbsentDocument` — never resolved from the exit status |
| Bytes that are not one readable JSON document | `ErrUnreadableDocument` — a read failure is not an absence |
| One document of another schema | `ErrForeignDocument` |
| Success at a failure status, failure at exit 0, `exit_code` ≠ observed status, tag disagreement | `ErrOutcomeDisagreement` |
| A status §15.2 assigns no meaning, `1` included | `ErrUnregisteredExitStatus` |

## 4. Historical envelopes remain readable

Two frozen corpora, checked-in bytes rather than objects a test builds, each
file pinned by SHA-256 with its expected machine answers recorded per row:

- `internal/cliresult/testdata/historical` — 9 CLI invocations: the §14.2 and
  §15.1 normative examples verbatim, a namespaced extension this reader knows
  nothing about, unknown detail keys, a code a later compatible minor added,
  that code with `retryable` forged to true (must be refused), and a
  `session.clone.*` failure.
- `internal/axerror/testdata/historical` — 7 protocol-bound envelopes covering
  all four bound Structured Error versions through provider 2, bridge 1,
  session-adapter 1, RPC 3, directory-query 1, and terminal-backend 1.

`TestABoundReaderNeverAdoptsThePayloadDeclaredVersion` offers every frozen
envelope to every containing contract binding a different version and requires
each cross pair to be refused, so a peer cannot select its own error version by
writing one into the payload.

The `session.clone.*` row is the compatibility fact the two CLI bindings
produce: this repository builds no clone success body, so `Read` returns
`ErrUnimplementedVersion` for its success, while its failure classifies
completely. "This build cannot construct that success" and "this failure is
unreadable" are different facts and stay different.

## Mutation evidence

14 mutants, each verified applied and compiled before measurement:
**13 killed, 1 declared subsumed, 0 unexplained survivors.** Harness and raw
report: `TASK-260830-33sfxc_mutation-report.json`, `TASK-260830-33sfxc_mutants.py`.

The subsumed mutant removes the exit-0 guard in `readFailure`; the `exit_code`
equality check refuses the same input one guard later, because
`decodeExitStatus` never admits a Structured Error carrying `exit_code` 0.

## Traceability

- Eight new acceptance cases: `cli-result-machine-reading`,
  `cli-result-stderr-independence`, `cli-result-message-independence`,
  `cli-result-historical-envelopes`, `cli-result-identity-precedence`,
  `structured-error-historical-envelopes`, `structured-error-exit-status-fan-in`,
  `structured-error-identity-precedence`. `acceptance_cases` 65 → 73.
- §17.2 rebound from `internal/config`'s `EncodeCurrent` to `Read` and left
  **unevidenced at 0/1** with a gap saying why: its single scanner-visible
  clause is an unknown-*event* obligation, and an unknown error code or inert
  unknown extension is not an unknown Session Event. The reader rules 1–4 carry
  no RFC 2119 keyword and are invisible to the clause scanner. This is a
  truthfulness fix, not a coverage gain — the ratio did not move.
- §14.2's gap now records that `Read` checks the exit-status equality from the
  consuming side, which narrows clause `14.2#6` to the missing `ax` process and
  does not close it. Coverage stays `partial` at 8/9.
- Ownership digest re-pinned; `clauses_discharged` unchanged at 17/403.

## Validation actually run (exit codes)

| Command | Exit | Result |
| --- | ---: | --- |
| `go build ./...` | 0 | clean |
| `go vet ./...` | 0 | clean |
| `gofmt -l internal` | 0 | no output |
| `go test ./... -count=1` | 0 | 13 packages ok |
| `go test ./... -cover -count=1` | 0 | `cliresult` 95.5%, `axerror` 99.5% |
| `go run ./internal/traceability/cmd/tracecheck -root .` | 0 | `acceptance_cases=73`, `clauses_discharged=17/403` |
| `python3 .temp/TASK-260830-33sfxc/mutants.py` | 0 | 13 killed, 1 subsumed, suite restored green |

## Stated bounds

- Package coverage is 95.5%; the two uncovered branches in `Read` and
  `exitStatusIsNotEnough` are unreachable by construction, and
  `TestEveryRegisteredCommandMajorBindsAnErrorVersion` proves that over all 44
  registered command tags rather than leaving them untested.
- Nothing here mutates durable state, opens a file outside `testdata`, or starts
  a process, so there is no crash or idempotency surface to evidence.
- No `ax` binary exists, so clause `14.2#6`'s process-level half remains
  unowned. `Read` checks the same equality from the reading side only.
