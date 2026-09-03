# TASK-260830-33sfxc — review verdict, CR revision 4

**Verdict: changes requested.** One blocking finding. Everything else in the
revision holds up under attack, including both findings from the previous round.

- Reviewed delta: `d1d3ece..2e893d0d` (candidate tree), 69 files.
  `git rev-parse HEAD^{tree}` = `2e893d0d9b476991e481098bafff9b685374aa18`,
  so the reviewed bytes are the candidate bytes.
- Working tree was returned byte-identical to the candidate after every mutation
  probe below (`git status --short` empty, tree hash re-verified).

## What was re-run here, not accepted from the attached logs

| Check | Command | Result |
| --- | --- | --- |
| Build | `go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Format | `gofmt -l .` (excluding `.temp`) | no files |
| Full suite | `go test ./... -count=1` | exit 0, 13 packages |
| Traceability | `go run ./internal/traceability/cmd/tracecheck` | exit 0; `acceptance_cases=74`, `clauses_discharged=17/403` — matches the figures the attached log and README publish |

The attached mutation reports (`…_mutation-report-r4.json`, 44 mutants, 43
killed, 1 declared subsumed) were read but **not** taken as the coverage
measurement. Eleven independent mutants were written and run against this tree
instead; the results below are mine.

## BLOCKING — F1: the command-tag agreement in `Read` is proven for one ordered pair, and the forged-document direction has no coverage at all

`internal/cliresult/client.go:readSuccess` — `if result.Command() != command`.

This guard is part of the deliverable's own corroboration set: it is what stops
a caller from being handed a `Result` whose body belongs to a command the
invocation never ran. It is currently covered by exactly one row,
`TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus/document reports
another command`, which drives the single pair *(invoked `doctor`, document says
`list`)*. Three narrowing mutants, each verified applied and compiled:

| Mutant | Result |
| --- | --- |
| `result.Command() != command && command != CommandDoctor` (spare the one covered invocation) | **KILLED** — control; the row is real |
| `result.Command() != command && command != CommandList` (spare a different invoked command) | **SURVIVED**, `go test ./internal/{cliresult,axerror,traceability}/... -count=1` exit 0 |
| `result.Command() != command && result.Command() != CommandTakeover` (admit any document that *claims* to be a takeover result) | **SURVIVED** the entire repository suite: `go test ./... -count=1` exit 0, all 13 packages `ok` |

The third one is the finding. With that narrowing in place, `Read(CommandList,
<a document carrying "command": "takeover">)` is admitted and the caller gets a
`Reading` whose `Result()` is a takeover body — the one command in this contract
that carries adoption/authority semantics (`Result.VerifyTakeoverAdoption`) — for
an invocation that never ran it. Nothing in the repository reddens.

The probe confirming the guard is the sole owner of this refusal (i.e. this is a
real class, not one `Decode` would catch anyway):

```
Read(CommandList, takeover-document)  err = invocation stdout and exit status disagree: stdout reports command "takeover" and the invocation was "list"
Read(CommandTakeover, list-document)  err = invocation stdout and exit status disagree: stdout reports command "list" and the invocation was "takeover"
```

Both refusals come from this guard. Neither `cliresult.Decode` nor the envelope
identity check refuses the pairing.

This is delete-only coverage of a gate, in the one function where the revision
otherwise holds itself to the stronger standard. The two sibling subtests in the
*same* test function sweep their whole domain and assert the sweep count — the
success-at-a-failure-status row drives every registered Section 15.2 status and
asserts `130` is among them, and the exit_code-equality row moves the status
across all 16 other registered statuses and asserts `moved`. The command-tag row
takes one sample. `…_rework-r4-results.md` states bounds for the harness figures,
the `details` narrowing, and the `CodesByExitStatus` pre-seed, but states no
bound for this guard: it is unmeasured rather than knowingly limited.

**What would close it:** drive the agreement over the registered command
vocabulary in both directions — for every invoked command, a document claiming a
different command; and for every command a document may claim, an invocation
that is not it — with the count of refusals asserted against the enumeration so
the loop cannot go vacuous, the way the two neighbouring subtests already do.
`Commands()` and `ImplementedCommands()` are already exported for exactly this.

## Non-blocking observations

- **Stated bound, not a defect.** `requireCommonDataModel` narrowed to skip
  documents over 4096 bytes SURVIVES. Every duplicate-member test builds a small
  document by hand, and no frozen historical envelope exercises the gate, so the
  gate's coverage is bounded to short documents. A byte-length predicate is not a
  dimension of the Section 1.6 contract, so I am not blocking on it — an
  arbitrary predicate can be found for any gate. It is worth recording as a
  stated bound because `axerror.Decode` reads peer-supplied provider, bridge,
  RPC, session-adapter and terminal-backend envelopes, and payload size is
  attacker-controlled on those paths.
- **Equivalent mutant, do not chase.** Setting `Reading.exitStatus` from
  `failure.ExitCode()` instead of `output.ExitStatus` SURVIVES and is provably
  equivalent: `readFailure` has already refused unless the two are equal. The
  revision's own results doc reaches the same conclusion; I reproduced it rather
  than taking it on trust.

## Verified holding under attack (killed independent mutants)

| Independent mutant | Killed |
| --- | --- |
| `cliresult.Decode`'s own common-data-model gate removed (pass `data` through instead of the canonical form) | yes — the success-branch gate is independently covered, so the client.go comment claiming *both* closed decoders run it is accurate |
| Discriminator gate in `documentSchema` narrowed to failure statuses only | yes |
| `axerror.decodeExitStatus` narrowed to admit exit status 0 | yes |
| `Reading.CodeRegistered` forged to `true` for every failure | yes |
| `verifyEnvelopeIdentity` major gate widened to admit major 2 | yes |

Also checked directly and found sound: the historical corpora are complete in
both directions (`TestHistoricalCorpusIsFrozenAndComplete` compares the directory
listing against the reviewed table and asserts the count, so neither an unread
fixture nor a silently dropped one stays green), each row is SHA-256 pinned, the
whole failure corpus is replayed with every message overwritten and the machine
answers asserted unchanged, `InvocationOutput` structurally cannot carry stderr,
and the Section 17.2 rebinding is reported as unevidenced 0/1 with a stated gap
rather than claimed.

## Routing

`to-dev`. One targeted test change; no production code needs to move. Re-review
should re-run the two surviving command-agreement narrowings above and require
them KILLED.

## Evidence files

- `.temp/TASK-260830-33sfxc/review-rev4/independent-mutation.json` — batch 1, 8 mutants
- `.temp/TASK-260830-33sfxc/review-rev4/independent-mutation-2.json` — batch 2, 3 command-agreement mutants
- `.temp/TASK-260830-33sfxc/review-rev4/mutants.py`, `mutants2.py` — the harnesses; each mutant verified present on disk and compiled before measurement, original restored after
