# Review verdict — CR-TASK-260830-33sfxc-7 revision 7

**Verdict: ACCEPTED.**

Reviewed candidate tree `67961efb6a03edfecd442df798d20368ca185cd2` at leaf commit
`adca16a`. The CR base `d1d3ece` is the Story checkpoint, so the reviewable delta
spans the two already-accepted sibling leaves (`ebc4e31` TASK-260830-34elja,
`f34d91d` TASK-260830-1bipsa) plus this leaf's own commit `adca16a`. This leaf's
commit is the scope of the verdict; the siblings were spot-checked.

## What I re-ran myself, standalone, on the candidate tree

| Check | Result |
| --- | --- |
| `gofmt -l .` | no output |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./... -count=1` | exit 0, 13 packages |
| `go test ./... -cover -count=1` | exit 0; `cliresult` 95.5%, `axerror` 99.5% — matches the claim |
| `go run ./internal/traceability/cmd/tracecheck` | exit 0; `acceptance_cases=74`, `clauses_discharged=17/403` — matches the claim |
| `git status --short` after all mutation work | empty; `HEAD^{tree}` == `67961efb…` |

Logs under `.temp/TASK-260830-33sfxc/`.

## Attack, not reading: 32 independent mutants

Seven batches. Each mutant applied by anchored replacement with the anchor
asserted present, compiled by the measuring `go test` run, and the tree restored
from a byte backup after every measurement (never `git checkout` — this is a
shared story worktree).

**29 killed. 3 survivors, all classified below; none admits anything.**

Killed by *narrowing* rather than deletion (the class this leaf exists to close):

| Mutant | Killed by |
| --- | --- |
| M3 `readFailure` `exit_code` equality restricted to registered codes | `TestTheExitStatusEqualityBindsACodeTheRegistryDoesNotCarry` |
| M4 `Read` registered-status gate narrowed to negatives | `TestTheReadLevelExitStatusAdmissionIsSweptOverTheWholeDomain` |
| M9 retryability refusal restricted to registered codes | `TestHistoricalEnvelopesRemainReadable`, `TestDecodeRetainsUnknownCodeExitClassWithoutSuccess` |
| M21 `axerror` data-model gate moved behind the identity check | `TestADuplicateIdentityMemberIsNotResolvedIntoAVersionFact` |
| M25 absence narrowed to zero-length stdout (whitespace collapses into a read failure) | `TestReadDistinguishesAbsenceFromAReadFailure` |
| M27 bound-version equality narrowed to zero minors | 4 tests across both packages |
| M28 one command tag spared from the agreement guard | `TestTheCommandAgreementGuardOwnsAMeasuredShareOfTheTagVocabulary` |
| M29 unknown `urn:ax:` schemas routed to the success branch | `TestTheForeignSchemaRefusalIsMeasuredOverTheRegisteredContractVocabulary` |
| M30 exit 130 spared from the success-at-failure-status guard | `TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus` |
| M31 exit 9 spared from the `exit_code` equality | `TestTheExitStatusEqualityBindsACodeTheRegistryDoesNotCarry` |
| M32 exit 1 admitted past the registered-status gate | `TestTheReadLevelExitStatusAdmissionIsSweptOverTheWholeDomain` |
| M22 `CodesByExitStatus` reports empty groups instead of absence | the fan-in measurement |

Killed by deletion: M1/M2 (both data-model gates), M5/M6 (command agreement,
success-at-failure-status), M7 (adding a `Stderr` member to `InvocationOutput` —
`TestMachineReadingCannotSeeStderr`), M8 (the JSON-`null` schema case), M10
(measured fan-in replaced with prose), M12 (quoted `exit_code` token).

### Published-figure pins, attacked separately

- README fan-in table, four independent falsifications — largest-class *status*
  for 1.2.0, registered-code *count* for 1.3.0, a whole row deleted, the
  multi-code-status count for 1.0.0 — **all four killed** by
  `TestREADMEFanInTableIsDerivedFromTheMeasuredProjection`. The table is derived
  cell by cell, not restated.
- Derived guard inventory: an unlisted `fmt.Errorf` refusal added to `client.go`
  → killed, naming the function, sentinel and format literal; a row pointing at
  a renamed test → killed; a `measured` cell replaced with prose → killed against
  the closed evidence-class vocabulary.
- Traceability registration: renaming a registered acceptance-case declaration
  makes `tracecheck` exit 1 naming the absent declaration. The registration is
  real, not decorative.
- Corpus completeness is derived from production `BoundContracts()`/`BindingFor()`,
  so a fifth bound version reddens rather than silently going uncovered.

### The three survivors

1. **M11 — `documentSchema`'s data-model gate moved to just before the return.**
   Equivalent. The gate still runs before the branch is selected, so no admission
   changes; only the detail sentence of a refusal on a document with two defects
   at once differs. The load-bearing form of this mutant (gate removed entirely,
   relying on the closed decoders) is M1, and M1 is killed by five tests.
2. **M24 — the exactly-one-document guard keyed on a failure exit status.**
   Subsumed, verified rather than assumed: I probed two concatenated conforming
   success documents at exit 0 under the mutant and `Read` still refuses with
   `ErrUnreadableDocument`, from the canonical gate's trailing-token refusal.
   Same shape as declared row 11, and inventory row 7 already discloses that "a
   predicate keyed on the exit status survives" for exactly this rule.
3. **M26 — `major != 1` narrowed to `major > 1` in `axerror.verifyEnvelopeIdentity`.**
   Relabel-only, verified: a `schema_version` of `0.9.0` is still refused and no
   object is produced, but the reported fact becomes `ErrUnsupportedVersion`
   rather than `ErrUnsupportedMajor` because `isRegisteredVersion` catches it one
   line later. No body is read, so there is no Section 15.1 trust leak.
   Recorded as a **non-blocking observation** below.

## Non-blocking observation for a future leaf

`axerror.verifyEnvelopeIdentity`'s major comparison is the untested twin of
`cliresult.verifyEnvelopeIdentity`'s, which this leaf swept over majors 0..64
through `Read`. The `axerror` side is asserted at major 2 only — one sample
*above* the bound — so narrowing it to `major > 1` misreports a major-0 envelope
as an unsupported *version*. That is the "twin one file away" class the leaf's
own inventory was built for, and the leaf states the bound honestly rather than
concealing it: the inventory's STATED BOUND says the refusals carried by
`internal/axerror/decode.go` are the earlier leaf's inventory, not this one's.
The guard is correct as written and admits nothing; this is a coverage gap on
TASK-260830-34elja's guard, not a defect in this delta. Not grounds to reopen.

## What I did not re-verify

The producer's 86-mutant harness is a STATED BOUND under `.temp/`, and the README
labels it as such rather than deriving it from a committed measurement. I did not
re-run it. My 32 mutants are independent and were written against the candidate
tree without reading the harness first.

## Against the acceptance criteria

- **Machine clients never depend on message text** — `HumanMessage` is the sole
  message-derived accessor; `TestMessageTextChangesNoMachineAnswer` and the
  historical replay with every message rewritten hold every machine answer fixed,
  and both require the rewritten bytes to actually differ so the row cannot pass
  vacuously.
- **…nor on stderr** — structural, not promised: `InvocationOutput` has exactly
  `Stdout` and `ExitStatus`, asserted by reflection; the behavioural half drives
  the real emitter with logs and progress on stderr, and the misrouting mutant
  leaves the client with `ErrAbsentDocument` rather than a status-recovered guess.
- **…nor on the exit status alone** — the document decides and the status only
  corroborates; all five disagreement shapes refuse, the refusals quote the
  *measured* fan-in rather than advice, and an unregistered status is refused
  outright.
- **Historical envelopes remain readable** — 9 CLI invocations and 7 protocol
  envelopes, SHA-256-pinned, each row recording the machine answers a client must
  still get, driven through the production entry points; corpus completeness
  derived from the production binding table in both directions.
- **No unsupported capability advertised** — the two disclosures (the Section 1.6
  number rule not enforced on this path; the mutant count as a stated bound) are
  asserted by tests that redden when the gap closes, and Section 17.2 stays
  `unevidenced` at 0/1 with a gap that names its own blind spot.
