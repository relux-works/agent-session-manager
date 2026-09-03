# TASK-260830-33sfxc — review verdict (CR revision 2)

**Verdict: changes requested.** Route: `to-dev`.

The implementation is correct. The *evidence* is not complete on the one gate
this leaf exists to prove, and the shortfall is the exact shape the leaf itself
publishes as its standard: "A gate whose only evidence is that deleting it
reddens something has been shown to exist, not to hold" (README) and "nine of
those mutants narrow rather than delete".

Two narrowing mutants on `readFailure`/`readSuccess` in
`internal/cliresult/client.go` survive the entire suite. Both were verified
applied, compiled, and confirmed to change observable behaviour with a probe
driving the production `Read` entry point.

## What was verified before the findings

Reproduced independently in this worktree at candidate tree
`c4967d95623cc20e4ffae2121065ba34cae1eb58`:

| Check | Command | Result |
| --- | --- | --- |
| Format | `gofmt -l .` | clean |
| Build | `go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Suite + coverage | `go test ./... -cover` | exit 0, 13 packages; `cliresult` 95.5%, `axerror` 99.5% — the claimed figures |
| Traceability | `go run ./internal/traceability/cmd/tracecheck` | exit 0, `acceptance_cases=74`, `clauses_discharged=17/403` — the claimed figures |

Twelve independent mutants were run in a scratch copy (`/tmp/axreview`, never
the reviewed worktree). Ten were killed, including every mutant aimed at the
claims the leaf makes loudest: the measured fan-in string in the refusals
(`TestAMisroutedDocumentLeavesTheMachineClientWithNothing`), the
schema-member type check, the unregistered-exit-status guard narrowed to
negative statuses, the command-tag agreement narrowed to one tag, the
discriminator's data-model gate narrowed by member content, absence narrowed to
zero length, the identity check made permissive on an absent `schema`, and
moving `requireCommonDataModel` out of `axerror.Decode` into `DecodeBound`
(killed by five tests, which confirms the rework's central claim that the gate
is closed at `Decode` for every peer-supplied envelope).

## Finding 1 (blocking) — the Section 14.2 exit-status equality is proven only for registered codes

`internal/cliresult/client.go`, `readFailure`.

Surviving mutant:

```go
- if failure.ExitCode() != output.ExitStatus {
+ if failure.CodeRegistered() && failure.ExitCode() != output.ExitStatus {
```

`go test ./internal/cliresult ./internal/axerror -count=1` → **ok**. No test in
either package drives an unregistered code at a mismatched status.

This is the class where the guard is the *sole* enforcement.
`axerror.decodeBody` (`internal/axerror/decode.go:281-297`) cross-checks
`exit_code` against the registry **only** when `ExitCodeFor` resolves; for a
code added by a later compatible minor it takes the `ErrUnregisteredCode`
branch and sets `registered = false` without any exit check. So for a
registered code the equality is doubly bound, and for an unregistered code
`readFailure` is the only thing binding the document's `exit_code` to the status
the process actually exited with — and that is precisely the row the tests do
not cover.

Failure scenario, reproduced against the mutant with the leaf's own frozen
fixture:

```
Read(CommandStatus, InvocationOutput{
    Stdout: testdata/historical/error-1.0.0-later-minor-code.json,  // exit_code 10
    ExitStatus: 5,
})
ADMITTED: code="terminal_backend_stale_generation" registered=false
          ExitStatus()=5  Failure().ExitCode()=10
```

The machine client is handed exit class 5 (workspace conflict, "no silent
overwrite") for a document declaring exit class 10 (ownership/lease/fencing).
On unmutated `HEAD` the same call is correctly refused with
`ErrOutcomeDisagreement`. `TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus`
subtest "failure exit_code differs from the process status"
(`client_test.go:268`) exercises `workspace_conflict`, a registered code, only.

The leaf added `error-1.0.0-later-minor-code-forged-retry.json` because it
recognised that an unregistered code must not be a way past the *retry* rule.
The same reasoning applies to the exit-class rule and was not carried over.

**Wanted:** a row that drives `Read` with an unregistered (later-minor) code
whose `exit_code` disagrees with the observed status, and requires
`ErrOutcomeDisagreement`. Add the narrowing mutant to the harness so the bound
is measured rather than assumed.

## Finding 2 (blocking) — the success-at-a-failure-status guard is proven at one status

`internal/cliresult/client.go`, `readSuccess`.

Surviving mutant:

```go
- if output.ExitStatus != SuccessExitStatus {
+ if output.ExitStatus != SuccessExitStatus && output.ExitStatus != 130 {
```

`go test ./internal/cliresult ./internal/axerror -count=1` → **ok**.

Failure scenario, reproduced against the mutant:

```
Read(CommandList, InvocationOutput{Stdout: <emitted list success>, ExitStatus: 130})
ADMITTED: Succeeded()=true  ExitStatus()=130
```

Exit 130 is the Section 15.2 row "Interrupted by operator signal before a clean
response; inspect authority before retry". It is the one status where a
success document plausibly *does* reach stdout before the process dies, so a
narrowing at exactly this status is a realistic bypass rather than an artificial
one — and it hands a machine client `Succeeded() == true` for an invocation the
pinned table says was interrupted before a clean response.

`TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus` subtest "success
object at a failure status" (`client_test.go:262`) uses exit 5 and nothing else;
the same single-sample shape applies to the "failure object at exit 0" row.

**Wanted:** drive the success-at-a-failure-status row across the registered
failure statuses (or at minimum across the classes the guard must hold for,
130 included), stating the coverage as a measured ratio rather than as one
sample.

## Non-blocking observation

A third mutant survived and is reported as an observation, not a finding:
narrowing `requireCommonDataModel` in `axerror.Decode` to documents whose bytes
contain `"details"` survives. Every conforming Structured Error requires
`details`, so the mutant is close to equivalent on any realistic input; it only
records that the gate's evidence happens to rest entirely on documents carrying
that member. No action required.

A fourth mutant — `CodesByExitStatus` pre-seeding an empty group for every
Section 15.2 failure status — also survived and was checked and dismissed as a
genuinely **equivalent** mutant: all four registered versions register at least
one code at all 17 failure statuses, so "absent rather than present and empty"
is currently unobservable. The documented rule is not wrong, it is vacuous
against today's registry. No action required.

## Definition of Done status

| Item | State |
| --- | --- |
| Production entry points implement the scoped deliverable | met — `cliresult.Read` classifies a completed invocation from stdout + status; `InvocationOutput` structurally excludes stderr |
| Positive, negative, compatibility and recovery tests pass with logs | met for the suite; **not met** for the two gates above |
| README/doctor/capability evidence and traceability updated without unsupported claims | met — README fan-in table and Logbook figures are re-derived from `CodesByExitStatus`; tracecheck reproduces the quoted counts |
| Gating behaviour covered by negative tests that fail when the gate admits what it must reject | **not met** — two narrowing survivors |
| Lint / build / validation clean | met |
| Outcome artifact attached | met |

Nothing here is a stop-the-line boundary: both fixes are ordinary test additions
inside the existing harness.
