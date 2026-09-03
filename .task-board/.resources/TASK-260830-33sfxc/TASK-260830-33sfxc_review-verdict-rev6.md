# TASK-260830-33sfxc — review verdict, Change Request revision 6

- Run: `RUN-260903-250f8d` (reviewer)
- Change Request: `CR-TASK-260830-33sfxc-6`, revision 6, `repository_delta=present`
- Reviewed bytes: `git rev-parse HEAD^{tree}` = `8f2a9527298fcd0a2b01bcacfb4f30e73a52f542` = the candidate tree OID. Commit `e708c4f`.
- Working tree returned byte-identical after every probe (`git status --porcelain` empty; scratch diffed against the worktree).

## Verdict: CHANGES REQUESTED → `to-dev`. One blocking finding.

The round-5 blocking finding is closed, and closed well. The new finding is a
direct answer to the question this round was scoped to ask.

## The question this round asked, answered directly

> Is there a reachable refusal guard in this leaf that is neither measured
> across its domain nor carries a stated bound?

**Yes, one: the `ErrUnregisteredExitStatus` gate at `internal/cliresult/client.go:121`.**
It is new production code in this Change Request, it is exercised at four sample
points of a 240-value domain, restricting it to exactly those four points passes
all thirteen packages, and no bound is stated for it in README, LOGBOOK, or the
rework results. It is the reader-side twin of `decodeExitStatus`, which this
very round swept over 0..255 after round 5 flagged the identical four-value
shape. The pattern was applied to one of the two.

Every other guard I attacked is either measured across its domain or carries a
stated bound. Details in "Held under attack" below.

## Re-run first, not accepted from the logs

Each a standalone process with its real exit status, at the candidate tree:

| Command | Exit | Result |
| --- | ---: | --- |
| `go build ./...` | 0 | — |
| `go vet ./...` | 0 | — |
| `gofmt -l .` | 0 | no output |
| `go test ./... -count=1` | 0 | 13 packages ok |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `acceptance_cases=74 clauses_discharged=17/403 normative_sections=36 bindings=49 contracts=60 fixtures=30 compatibility_contracts=55` |

Every published figure reproduces.

## Round-5 finding: CLOSED, re-measured here

The producer asked for four narrowings to be re-run and required KILLED. All
four are KILLED, and the delete-only control is still KILLED, so the coverage
is real rather than the guard merely having become unreachable.

| Mutant on `internal/axerror/decode.go` | At `1d0d702` (round 5) | Here |
| --- | --- | --- |
| `if false && exitCode != expectedExit` (delete, control) | KILLED | **KILLED** |
| `&& code == "observation_gap"` (restrict to its one covered code) | SURVIVED | **KILLED** |
| `&& code != "policy_refused"` | SURVIVED | **KILLED** |
| `&& code != "authentication_failed"` | SURVIVED | **KILLED** |
| `!IsFailureExitStatus(status) && status != 42` | SURVIVED | **KILLED** |
| `!IsFailureExitStatus(status) && status != 256` (out-of-range arm) | — | **KILLED** |

## BLOCKING F1 — the Read-level unregistered-exit-status gate is proven at four points and bounds nothing

`internal/cliresult/client.go:121`

```go
if output.ExitStatus != SuccessExitStatus && !axerror.IsFailureExitStatus(output.ExitStatus) {
	return nil, fmt.Errorf("%w: %d", ErrUnregisteredExitStatus, output.ExitStatus)
}
```

Its whole coverage is one subtest of
`TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus`
(`client_test.go:511`), which drives `{1, 42, -1, 255}`.

Measured against `go test ./... -count=1`, each mutant verified present on disk
and compiled before measurement:

| Mutant | Verdict |
| --- | --- |
| `if false && ...` (delete, control) | **KILLED** — the four rows are real |
| `&& output.ExitStatus != 127 && output.ExitStatus != 137` | **SURVIVED** |
| `&& (ExitStatus == 1 \|\| == 42 \|\| == -1 \|\| == 255)` — restricted to exactly its covered sample | **SURVIVED**, all 13 packages ok |

That last row is round 5's F1 shape exactly: restrict a guard to precisely the
points its coverage drives, and nothing reddens.

**What it costs, reproduced through `cliresult.Read` with only the `{127, 137}`
narrowing applied.** 127 is command-not-found and 137 is SIGKILL/OOM — the two
statuses a wrapper most plausibly returns when `ax` never ran or was killed
before it could write anything, which makes this narrowing realistic rather
than artificial.

| Invocation | Unmutated | Narrowed |
| --- | --- | --- |
| absent stdout @ exit 127 | `ErrUnregisteredExitStatus: 127` | `ErrAbsentDocument: ... exit status 127 is assigned to 0 registered Structured Error 1.0.0 codes` |
| garbage on stdout @ exit 137 | `ErrUnregisteredExitStatus: 137` | `ErrUnreadableDocument: ... exit status 137 is assigned to 0 registered Structured Error 1.0.0 codes` |
| valid failure doc @ exit 127 | `ErrUnregisteredExitStatus: 127` | `ErrOutcomeDisagreement` |

**This is not an admission bypass, and I am not claiming one.** Nothing
forbidden becomes admitted: a failure document is still refused by
`readFailure`'s equality, a success document by `readSuccess`'s status guard,
and `decodeExitStatus` still refuses an unregistered `exit_code`. What changes
is which fact the machine client is told — and this leaf is the one that spent
a whole round establishing that which fact a caller reads is the deliverable,
not merely that something refused.

Two things make it this leaf's finding rather than a nit:

1. **The guard is the sole enforcement of the thing that goes wrong.**
   `documentSchema` has exactly one caller, `Read`, and this gate is the only
   thing standing before it. Downstream, `exitStatusIsNotEnough` computes
   `len(fanIn[exitStatus])` with no membership check, so under the narrowing it
   publishes *"exit status 127 is assigned to 0 registered Structured Error
   1.0.0 codes"* — a fabricated fan-in for a status the §15.2 table does not
   carry at all. README line 1548 publishes that function as stating "a measured
   count rather than advice", and the gate's own doc comment says "inventing a
   class for a status outside it would be a guess". `0` here is not a
   measurement of an empty class; it is a map miss reported as one. That is the
   "nothing is there is not nothing this checker sees" rule, in this leaf's own
   words, on this leaf's own published sentence.

2. **The sibling was swept this round and this one was not.** Round 5 flagged
   `decodeExitStatus` for having a four-value sample; the producer closed it
   with `TestTheExitStatusAdmissionIsSweptOverEveryByteValue`, 0..255 plus six
   out-of-range values. The reader-side gate has the same four-value shape, the
   same domain, and the helper to sweep it (`registeredFailureExitStatuses`)
   already exists in `client_test.go`. The rework results state bounds for the
   harness figures, the payload-size narrowings and the dismissed equivalents;
   they state none for this gate. It is unmeasured rather than knowingly
   limited, and that distinction is the finding — the same sentence round 5 used.

README line 1519 publishes the row *"A status Section 15.2 assigns no meaning,
`1` included → `ErrUnregisteredExitStatus`"*, and line 1453 publishes *"Every
gate has a negative test that narrows it rather than deleting it"* over the
`cliresult` section. Both are currently true at four points.

**What closes it (test-only, no production change).** Mirror
`TestTheExitStatusAdmissionIsSweptOverEveryByteValue` on the reader: drive
`Read` over every status in 0..255 plus out-of-range values, require admission
past this gate iff `axerror.IsFailureExitStatus`, assert the admitted count
against the enumeration so the loop cannot go vacuous, and add the
restrict-to-the-sample narrowing to the harness. While there, decide whether
`exitStatusIsNotEnough` should distinguish "this status carries no registered
code" from "this status is not in the table"; today only the gate above keeps
the second case from being reported as the first.

## Non-blocking observations

**O1 — a stated reason in the record is wrong, though its conclusion survives.**
Round 5 dismissed three mutants as equivalent because "none is a registered
code", and the round-6 brief asked me to verify that. It is false for one of the
three: `terminal_backend_capability_unproven` **is** registered, at Structured
Error 1.3.0, exit 6 (`internal/catalog/catalog_gen.go:541`; `ExitCodeFor(1.3.0,
…) = 6`). It is unregistered at 1.0.0/1.1.0/1.2.0, which is precisely why the
mutant was about `ExitCodeFor`'s *version scoping*. Re-run on a clean tree,
sparing it in the version-scoping check **SURVIVES**. The conclusion still holds
for a different reason than the one recorded: I measured the same narrowing on
`observation_gap`, also registered at 1.2.0/1.3.0 only, and it is **KILLED** by
`TestExitCodeForRefusesVersionAndCodeDrift`. So the partially-registered-code
class is covered by a representative; only that particular code is not. Worth
correcting in the record because this board tracks restated-reason drift.

**O2 — the common-data-model gate owns less of §1.6 than its first sentence
says, and `Read` admits a §1.6 violation today.** Narrowing `documentSchema`'s
canonical gate to duplicate-key errors only **SURVIVES**; the same narrowing on
`axerror.requireCommonDataModel` is **KILLED**. Probing further,
`canonicaljson.Canonicalize` does not enforce the float or safe-integer half of
§1.6 at all: `Canonicalize([]byte("{\"n\":1.5}"))` returns no error, and
`Read(CommandMaterialize, …)` **admits** a Structured Error carrying
`"details":{"n":1.5}` at exit 5, which SPEC.md:221 forbids. The gate's doc
comment is honest where it counts — it enumerates exactly what it owns
(duplicates, malformed UTF-8, lone surrogate escapes, trailing content) and
names the subsumed case — so this is an overclaim only in its opening sentence
("refuses bytes that are outside the Section 1.6 common logical data model") and
in README's matching phrasing. Nothing false is claimed in traceability:
`section:1.6` is bound to `internal/scalar` and left unevidenced, and this leaf
deliberately claimed no clause for the gate. Not blocking, and the fix is not
obviously this leaf's, but the leaf's brief says disclose what is found and not
fixed, and this is not disclosed anywhere.

**O3 — `verifyEnvelopeIdentity`'s major gate is proven at major 2 only.**
`(major != 1 && major != 3)` **SURVIVES**. Measured consequence, not inferred: a
`3.0.0` document then reports `ErrUnsupportedVersion` instead of
`ErrUnsupportedMajor` — indistinguishable from the same-major unknown minor
`1.9.0`, which reports the identical sentinel. That is the fact this leaf's own
precedence decision exists to protect ("a caller deciding whether its ax is too
old cannot decide that from a member name"). It is leaf-1 production code and
the pin this leaf added drives 2.0.0 only. Recorded, not blocked: the domain
"every major that is not 1" is unbounded and one representative is a defensible
sample. Say so as a bound rather than leaving it silent.

**O4 — arbitrary-predicate narrowings, recorded and explicitly not chased.**
Narrowing `documentSchema`'s trailing-content gate and its missing-`schema`
gate by exit status (`&& exitStatus != 7`) both **SURVIVE**. Exit status is not
a dimension of §14.2's "exactly one JSON document" rule, so this is the same
class the round-4 reviewer declined to block on for the 4096-byte predicate: an
arbitrary predicate can be found for any gate. No action wanted. Recorded
because a survivor dismissed with a reason is evidence.

## Held under attack — 31 mutants, 21 killed, 10 survivors all classified

The oracle question the brief raised is settled by measurement.
`registeredFailureStatusesFromTheMeaningTable` reads `ExitStatusMeaning` while
the decision under test calls `IsFailureExitStatus`, and the two are separate
functions over one map. Narrowing the predicate (`IsFailureExitStatus` returning
false for 130) is **KILLED**, including by the sweep itself — so the oracle does
not move with the mutant. Adding a row to `exitMeanings`, which moves both, is
**KILLED** by the asserted 17-count and by
`TestExitStatusRegistryMatchesThePinnedTableRowForRow`. The reasoning holds.

Domains are derived at run time, not pinned as literals:
`TestTheCodeToExitStatusAgreementIsMeasuredOverTheWholeCodeRegistry` iterates
`Versions()` / `CodesFor` / `CodesByExitStatus`, and its literal per-version
totals (752/1056/1504/1744) are cross-checked against
`len(codes)*(len(statuses)-1)` in the same assertion, so a literal alone cannot
carry it.

Killed here: the retryability refusal narrowed at exit class 16 and at 130;
`readSuccess`'s success-at-a-failure-status guard spared at 16 (round-2 F2 class
re-checked at a different status); `readFailure`'s exit equality spared at 16
(round-2 F1 class); the command agreement spared for `CommandMaterialize`
(round-4 class re-checked on a different tag); the absence gate narrowed by exit
status; `axerror`'s common-data-model gate deleted and narrowed to duplicates
(round-1 class); the `InvocationOutput.Stderr` member added, which reddens
`TestMachineReadingCannotSeeStderr`; one byte perturbed in a frozen historical
fixture, which reddens the SHA-256 corpus pin.

The two README pins compare rather than parse, and fail closed. Perturbing the
1.3.0 fan-in cell and deleting the 1.3.0 row both redden
`TestREADMEFanInTableIsDerivedFromTheMeasuredProjection`. Restoring 73
acceptance cases and changing 49 bindings to 48 both redden
`TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport`, so at least two of
the seven figures are genuinely compared. Renaming the heading and renaming a
figure's noun phrase both redden it too, so it fails closed on a target it
cannot locate and on a figure that fell out of measurement.

Nothing from rounds 1-5 regressed, and no survivor a previous reviewer dismissed
with a reason was quietly fixed.

## Stated bounds of this review

- Review depth was concentrated on the guards reachable from `cliresult.Read`
  and the two decoders. Leaves 1 and 2 were probed through the new consuming
  entry point, not re-reviewed in full.
- The producer's 63-mutant harness was **not** re-executed. The 31 mutants above
  were written independently of it and measured against the full 13-package
  suite.
- Coverage percentages were not re-measured; `go test ./... -cover` was not run
  in this review, only the suite itself.
- One batch of 8 mutants was discarded and re-run: a probe `package main`
  directory I left inside the scratch module reddened
  `TestModuleHasNoProductCommandYet`, which would have recorded a false KILL for
  the `capability_unproven` row. Only the clean re-run appears in the report.

## Routing

`to-dev`. One sweep test closes F1. On re-review, require the
restrict-to-the-sample narrowing of `client.go:121` to come back KILLED with the
delete-only control still KILLED.
