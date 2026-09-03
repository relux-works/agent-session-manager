# TASK-260830-33sfxc — round-7 rework results

Reviewed head: `e708c4f`, tree `8f2a9527298fcd0a2b01bcacfb4f30e73a52f542`.
One blocking finding, closed. Three reviewer observations acted on, one left
with its reason. One production behaviour change, found by the traversal the
brief asked for.

## F1 (BLOCKING) — the reader-side unregistered-exit-status gate

`internal/cliresult/client.go:121`. Proven at four sampled values of a 240-value
domain; the twin `axerror.decodeExitStatus` was swept over 0..255 one round
earlier and the pattern was applied to one of the two.

Reproduced on the UNCHANGED reviewed tree before any test was written
(`repro-r7.json`), measured against `go test ./... -count=1`:

| Probe | Result | Suite exit |
| --- | --- | ---: |
| delete-only control (`if false && ...`) | KILLED | 1 |
| restricted to exactly `{1, 42, -1, 255}` | SURVIVED | 0 |
| narrowed to spare 127 and 137 | SURVIVED | 0 |

Closed the same way its twin was closed.
`TestTheReadLevelExitStatusAdmissionIsSweptOverTheWholeDomain` drives `Read`
over 0..255 plus eight values outside the byte range, requires admission past
the gate if and only if the Section 15.2 meaning table registers the status,
asserts the admitted (18) and refused (238) counts, cross-checks the gate's two
production predicates against the meaning-table oracle at every point, and
requires the refusal never to carry the fan-in sentence.
`TestTheReadLevelExitStatusAdmissionAdmitsARealReadingAtEveryRegisteredStatus`
is the positive half: a conforming document at every registered status must be
classified, not merely refused differently.

The oracle is `ExitStatusMeaning`, not `IsFailureExitStatus`, because the latter
is a predicate the gate calls and an oracle built from it would move with a
mutant that narrows it.

## The guard inventory — the mechanism the brief asked for

`TestTheRefusalGuardInventoryIsDerivedFromTheReaderSource` parses every refusal
site out of `internal/cliresult/client.go` with `go/ast` and requires a
bijection with the twelve rows published under README's
"### Refusal guard inventory". Each row names its function, sentinel, refusal
marker, domain-evidence class from a closed vocabulary
(`measured` / `stated bound` / `declared subsumed`), and the test declarations
that measure it; every named test must exist. A second table enumerates the
seven guards this leaf added or reordered outside that file, with
`decodeExitStatus` required in it by name.

Attacked in both directions: a deleted row, a row naming a test that does not
exist, a marker that no longer resolves, an evidence class outside the closed
vocabulary, a dropped twin row, a sibling function renamed out of its file, a
NEW refusal added to the reader with no row, and the derivation itself narrowed
to skip one function. All eight KILLED.

## What the traversal found that no reviewer had named

| Site | Narrowing | Before | After |
| --- | --- | --- | --- |
| `ErrForeignDocument` branch | restricted to the one schema identifier its only row drove | SURVIVED | KILLED |
| `schema` member type guard | restricted to JSON numbers, the one form its only row drove | SURVIVED | KILLED |
| `schema` member presence guard | narrowed by an arbitrary exit-status predicate | SURVIVED | stated bound (see below) |

Both closed by measurement: the foreign-schema branch over every contract
identifier the pinned catalog registers (60 contracts, 58 of them foreign; denominator asserted
against `catalog.Current().Contracts`) plus twelve near-miss neighbours of the
two admitted identifiers; the type guard over every JSON value form with a
string control required to reach the discriminator instead.

The presence guard is left as a STATED BOUND with the contract reason: its
domain is two-valued and both values are driven, and the surviving predicate is
keyed on the exit status, which is not a dimension of Section 14.2's
exactly-one-document rule. This is the arbitrary-predicate class round 4
declined to block on, recorded rather than silently omitted.

## Production behaviour change (one)

`encoding/json` admits JSON `null` into a `string` and yields `""`, so a
document whose `schema` member was `null` was answered as a FOREIGN CONTRACT
carrying the schema `""` — a claim that some other contract owns the document,
when what is true is that this one is not readable. The guard now checks the raw
token alongside the unmarshal. Found only because the type guard's domain was
swept rather than sampled; the harness carries the mutant that restores it.

## The `exitStatusIsNotEnough` question, settled

The round-6 review asked whether the function should distinguish "this status
carries no registered code" from "this status is not in the table", since it
computes `len(fanIn[status])` with no membership check.

Not with a defensive branch. A branch no production path can take promises
nothing and is never measured.
`TestTheFanInSentenceCannotReportAMapMissAsAMeasurement` proves the fabrication
unreachable instead: `documentSchema` has exactly one caller and it is `Read`,
`exitStatusIsNotEnough` is called only from `documentSchema`, the gate precedes
the `documentSchema` call in `Read`'s body by AST position, and every
(registered version, registered failure status) pair measures at least one code,
so a 0 in that sentence could only ever have been a map miss. Moving the gate
behind the discriminator KILLS it.

## Reviewer observations

- **O1 — corrected.** Round 5 dismissed three mutants because "none is a
  registered code". `terminal_backend_capability_unproven` IS registered at
  1.3.0 exit 6 (`catalog_gen.go:541`), verified here. The conclusion holds for a
  different reason — the same narrowing on `observation_gap` is killed by
  `TestExitCodeForRefusesVersionAndCodeDrift` — so the class has a covered
  representative. The dismissal stands; the sentence behind it did not. Recorded
  in README and LOGBOOK.
- **O2 — reproduced and disclosed.** `Read` admits a Structured Error whose
  `details` carry `1.5`, which SPEC.md forbids. Verified independently before
  publishing. `canonicaljson.Canonicalize` does not enforce the number half of
  Section 1.6; Section 1.6's number rule is bound to `internal/scalar` and left
  unevidenced there, so nothing false is claimed in the traceability projection.
  The admission is now ASSERTED by
  `TestTheCommonDataModelGateDoesNotCoverTheSection16NumberRule`, so closing the
  gap reddens the pin. The obvious mutant on that test is EQUIVALENT on this
  tree and was DISCARDED rather than recorded as a kill; the replacement closes
  the gap in production and requires the pin to redden.
- **O3 — closed by measurement rather than by a stated bound.**
  `verifyEnvelopeIdentity`'s major comparison was proven at major 2 alone.
  `TestTheUnsupportedMajorGuardIsSweptOverAMeasuredMajorRange` sweeps majors
  0..64 through `Read` with the bound stated for the tail. Where a domain can be
  swept cheaply, a stated bound would be a bound about effort.
- **O4 — no action, as the reviewer asked.** Recorded as row 7's stated bound.
- The equivalent survivors dismissed in rounds 2, 3 and 5 were left alone.

## Validation, run on this tree as standalone processes

| Command | Exit |
| --- | ---: |
| `gofmt -l .` | 0 (no output) |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `go test ./... -count=1` | 0, 13 packages |
| `go test ./... -cover -count=1` | 0 (cliresult 95.5%, axerror 99.5%, both unchanged) |
| `go run ./internal/traceability/cmd/tracecheck` | 0 |

`tracecheck`: `acceptance_cases=74`, `clauses_discharged=17/403`,
`normative_sections=36`, `bindings=49`, `contracts=60`, `fixtures=30`,
`compatibility_contracts=55` — all unchanged. The eight new tests register
against the existing `cli-result-machine-reading` case; the registration was
probed by renaming a declaration and confirming `tracecheck` names the missing
declaration ahead of the projection digest, which was then re-pinned.

## Harness

63 -> 86 mutants: 85 KILLED, 1 declared subsumed (unchanged), 0 unexplained
survivors, 64 narrowing by the harness's own definition. Every narrowing name is
asserted to be a declared mutant. Run in three batches to stay inside the shell
time bound; each batch restores the tree and re-runs the suite, and all three
`restored_suite_exit_code` values are 0.

STATED BOUND of the inventory: the derivation is exhaustive over
`internal/cliresult/client.go`, which this leaf added in full. The sibling table
is enumerated rather than derived; refusals that `internal/axerror/decode.go`,
`internal/axerror/registry.go` and `internal/cliresult/decode.go` carry from
earlier leaves are those leaves' inventories, not this one's.
