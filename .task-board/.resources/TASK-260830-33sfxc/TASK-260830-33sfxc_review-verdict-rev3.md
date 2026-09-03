# TASK-260830-33sfxc — review verdict (CR revision 3)

**Verdict: changes requested.** Route: `to-dev`.

The two round-2 findings are closed and were verified closed independently, not
taken from the rework report. The compatibility claim this leaf exists to make —
that a machine client never depends on message text, stderr, or the exit code
alone, and that historical envelopes stay readable — survived an independent
attack of 40 further mutants written without reading the producer's harness
first. One blocking defect remains, and it is in the artifact the acceptance
criterion names: **a published figure in `README.md` is false, contradicted by
the repository's own gate, and nothing measures it.**

## What was reproduced first, on this exact tree

Candidate tree `bec56317e09b2ed9127fb2cc60900d47081549c7`; `git rev-parse
HEAD^{tree}` equals it, so the reviewed bytes are the shipped bytes.

| Check | Command | Result |
| --- | --- | --- |
| Format | `gofmt -l ./internal` | clean, exit 0 |
| Build | `go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Suite | `go test ./... -count=1` | exit 0, 13 packages (`.temp/TASK-260830-33sfxc/review-gotest-01.log`) |
| Coverage | `go test ./internal/cliresult ./internal/axerror -cover -count=1` | `cliresult` 95.5%, `axerror` 99.5% — the claimed figures |
| Traceability | `go run ./internal/traceability/cmd/tracecheck` | exit 0, `acceptance_cases=74`, `clauses_discharged=17/403` |
| Producer harness, re-run in a scratch copy | `python3 mutants-r3.py` | 28 mutants, 27 killed, 1 subsumed, **0 unexplained survivors** — reproduces |

Fan-in table re-derived independently from `CodesByExitStatus` through a
throwaway test rather than by trusting `TestExitStatusAloneIdentifiesNoFailure`:

| Version | statuses | codes | ambiguous | largest |
| --- | ---: | ---: | ---: | --- |
| 1.0.0 | 17 | 47 | 14 | 6 at exit 6 |
| 1.1.0 | 17 | 66 | 14 | 9 at exit 6 |
| 1.2.0 | 17 | 94 | 15 | 14 at exit 16 |
| 1.3.0 | 17 | 109 | 15 | 17 at exit 6 |

Every cell matches the published README table. Also verified by measurement, not
by reading: 15 bound contracts in `staticBindings`, 9 `cliresult` historical
fixtures, 7 `axerror` historical fixtures over all four bound versions, 17
registered failure statuses.

The round-2 findings are genuinely closed: both narrowing mutants
(`readFailure` equality restricted to registered codes; `readSuccess` guard
narrowed past exit 130) are KILLED in the re-run above.

## Independent mutation attack — 40 mutants, all in a scratch copy at `/tmp/axrev3`

Written before reading `mutants-r3.py`, so the overlap is coincidental rather
than derived. Everything below was verified applied on disk and compiled before
it was measured, and the suite was restored to exit 0 after every round.

KILLED (34, selection): `CodeRegistered()` forced true; `Retryable()` ignoring
the document bit; `Code()` reporting present for a success; command-tag
agreement narrowed to one tag; absence gate narrowed to a nil slice; the
discriminator's data-model gate fed a constant instead of the caller's bytes;
the unregistered-exit-status guard narrowed to `> 255`; `CodesByExitStatus`
dropping one status' group; **both** data-model gates narrowed by document size;
the multi-document guard narrowed to admit a second document;
`RegisteredVersionForCommand` swapped for `VersionForCommand`; success and
failure decode errors swallowed; retry derived from a message containing
`retry`; retry derived from message length; `CodeRegistered` derived from
message length; retry derived from detail-key presence; `CodeRegistered`
derived from the exit status; **retry decided entirely by an exit-status class
with the document bit discarded** — the realistic whole-cloth form of the defect
this leaf exists to refuse — killed by `TestExitStatusAloneDecidesNoRetry`.

SURVIVED and checked as genuinely EQUIVALENT (2): `Succeeded()` recomputed from
`exitStatus`, and `ExitStatus()` returned from `failure.ExitCode()`. Both
survive *because* the `ErrOutcomeDisagreement` guards make the two signals
provably equal on any admitted reading. These are evidence that the guards hold,
not gaps.

SURVIVED and reported below as observations (3), plus 1 that changed no
behaviour and 2 that failed to compile and were discarded.

## Finding 1 (blocking) — `README.md:1643` publishes 73 acceptance cases; the gate measures 74

`README.md`, lines 1642–1646:

> registry independently enumerates implementation owners for all 60 current
> contract rows, 36 pinned or catalog-referenced normative section keys, **73**
> executable acceptance cases, 49 exact section bindings …

Every other number in that sentence reproduces exactly against
`go run ./internal/traceability/cmd/tracecheck` (`contracts=60`,
`normative_sections=36`, `bindings=49`, `unowned=2`, `fixtures=30`,
`compatibility_contracts=55`). One does not:

```
tracecheck:                                     acceptance_cases=74
ownership.v0.5.0.json: len(acceptance_cases) =  74   (74 unique ids)
README.md:1643:                                 73
```

This is not ambiguity about what is being counted. The registry array holds 74
entries with 74 distinct `id`s, and the file the sentence links to is that
registry.

**How it happened, from the leaf's own record.** At the Story checkpoint
`f34d91d` the registry held 65 and the README said 65 — consistent. This leaf's
first pass took it to 73 and updated the sentence. The round-2 rework added the
74th case; LOGBOOK entry 1157 records it correctly (“`acceptance_cases` 73 → 74”)
and entry 1231 restates 74. The README sentence was never carried forward. The
LOGBOOK and the README now contradict each other inside one commit.

**Nothing measures it.** Mutant, applied to the scratch copy:

```
- contract rows, 36 pinned or catalog-referenced normative section keys, 73
+ contract rows, 36 pinned or catalog-referenced normative section keys, 999
```

`go test ./... -count=1` → **exit 0**. `go run ./internal/traceability/cmd/tracecheck`
→ **exit 0**, still printing `acceptance_cases=74`. A figure that can be set to
999 without reddening anything is prose, and this one is prose that is currently
wrong.

**Why this is blocking rather than a typo.** The acceptance criterion is
explicit that "no unsupported capability is advertised", and the DoD item is
"README/doctor/capability evidence and specification traceability are updated
without unsupported claims". More pointedly, this is the **third** occurrence of
the identical class inside this artifact, and the leaf documents the first two
itself: LOGBOOK 1015 records the "nineteen §15.2 rows" figure that nobody
measured, and LOGBOOK 1157 records two of four published fan-in rows being
wrong, with the root cause named as "prose nobody measured". The response to
that finding was `TestREADMEFanInTableIsDerivedFromTheMeasuredProjection`, which
parses the README and re-derives every cell — and it sits in the same
repository, 200 lines from a figure that is restated instead of derived and is
off by one.

**Wanted.** Correct the figure to 74, and make it measured the way the fan-in
table already is: a test that parses this sentence out of `README.md` and
compares the count against `len(registry.acceptance_cases)`. `internal/axerror/
exit_table_pin_test.go:148` (`parsePublishedFanIn`) is the working pattern —
it locates its table by content, refuses a table it cannot parse, and refuses a
duplicated header. Add the mutant that restores 73 to the harness so the bound
is measured rather than asserted. The other five figures in the same sentence
are correct today and are unmeasured for the same reason; pinning the sentence
as a whole costs no more than pinning one number of it.

## Observation A (non-blocking) — the refusal's "measured count" is not itself measured

`internal/cliresult/client.go`, `exitStatusIsNotEnough`. Both the doc comment
("states, as a measured count rather than as advice") and `README.md` ("the
refusals say so with a measured count rather than as advice") make the count in
the refusal text load-bearing. Two surviving mutants:

```
- exitStatus, len(fanIn[exitStatus]), errorVersion)
+ exitStatus, 1, errorVersion)
```

```
- "exit status %d is assigned to %d registered … codes, so the status alone identifies no failure"
+ "exit status %d is assigned to many registered … codes, so the status alone identifies no failure"
```

`go test ./internal/cliresult ./internal/axerror -count=1` → **ok** for both.
Every assertion on these refusals matches the trailing clause
(`"identifies no failure"`); no test reads the number.

Not blocking: the count is diagnostic text a human reads from a log, not a
machine answer, and the projection it draws on (`CodesByExitStatus`) and the
published table are both measured. But the sentence "measured count rather than
advice" is currently true of the implementation and unproven by the suite — one
assertion that the refusal for exit 6 under 1.0.0 names 6 would close it.

## Observation B (non-blocking) — retry independence from the exit status is proven at one status

`TestExitStatusAloneDecidesNoRetry` proves the claim at exit 12, which is the
right choice: it is a class carrying both a code that may claim `retryable` and
one the pinned document forbids it for, so the row exercises the actual
decision. Forcing `Retryable()` true at exit 12 is killed, and the whole-cloth
"retry decided by the status class" mutant is killed.

Surviving: `if reading.exitStatus == 15 { return true }` and the same shape over
`{15, 20, 21, 22}`. They survive because the only exit-15 envelope any test
drives (`error-1.0.0-partial-sync-unknown-details.json`) already declares
`retryable: true`, so the fabricated branch returns the answer the document
would have given.

Reported as an observation, not a finding, for two reasons. `Reading.Retryable()`
is a field accessor with no branch to narrow — these mutants add a special case
rather than restricting a gate — and the gate that actually governs forged retry
claims, `axerror.RetryabilityRefusal`, is exercised on both arms. Worth one
row driving a retryable-false envelope at a second status if the sweep pattern
the round-3 rework introduced is being applied generally.

## Observation C — two equivalent survivors worth recording

`Succeeded()` recomputed from `exitStatus`, and `ExitStatus()` returned from
`failure.ExitCode()`, both survive the suite and both are genuinely equivalent:
`readSuccess` and `readFailure` between them make document and status agree on
every admitted reading, so the two derivations cannot differ. Recorded so a
future round does not spend a cycle rediscovering them as apparent gaps.

## What is not wrong

Stated explicitly, because the blocking finding is small and should not be read
as a judgement on the leaf:

- The stderr claim is structural and holds. `InvocationOutput` has exactly two
  members, `TestMachineReadingCannotSeeStderr` reddens the moment a third
  appears, and the behavioural half drives the real emitter with logs and
  progress on stderr and then classifies from stdout alone, with a stream-swap
  mutant that leaves the client holding `ErrAbsentDocument`.
- The three observation sentinels are genuinely distinct and each row asserts
  which facts the answer is *not*, which is what makes the alias mutant die.
- The historical corpora are frozen bytes, digest-pinned, complete in both
  directions (every file reviewed, every reviewed row present), with the count
  asserted, and each row records the machine answers rather than only that the
  bytes parse.
- The common-data-model gate is closed at `axerror.Decode`, so every
  peer-supplied envelope surface is covered in one place, and the discriminator
  keeps its own copy because it runs before the branch is chosen. Both survive
  size-narrowing and version-narrowing mutants.
- The duplicate-member fix is real: the gate refuses the forged-retry shape and
  the canonical bytes are discarded rather than adopted, with the
  laundering mutant killed.

## Definition of Done status

| Item | State |
| --- | --- |
| Production entry points implement the scoped deliverable | met — `cliresult.Read` is the consuming entry point; every test drives it rather than a helper |
| Positive, negative, compatibility and recovery tests pass with logs attached | met |
| README/doctor/capability evidence and traceability updated without unsupported claims | **not met** — Finding 1 |
| Gating behaviour covered by negative tests that fail when the gate admits what it must reject | met — 27/28 producer mutants killed with the survivor declared subsumed, plus 34 independent kills; no gate bypass found |
| Lint / build / validation clean | met |
| Outcome artifact attached | met |

Nothing here is a stop-the-line boundary. Finding 1 is one digit plus one test
in an existing pattern.
