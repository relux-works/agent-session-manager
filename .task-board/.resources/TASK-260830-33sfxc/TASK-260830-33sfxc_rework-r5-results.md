# TASK-260830-33sfxc — round-5 rework results

Reviewed head: `b43a746` (CR revision 4). This round amends that commit; the
branch stays exactly one commit past checkpoint `f34d91d` with a clean worktree.

**No production code changed.** The only non-test edits are `README.md`,
`LOGBOOK.md`, the ownership registry, and its projection digest constant.

## F1 (BLOCKING) — the command-tag agreement was proven at one ordered pair

`internal/cliresult/client.go:readSuccess` — `if result.Command() != command`.

### Reproduced before anything was written

`.temp/TASK-260830-33sfxc/r5/repro-r5.py`, measured against `go test ./... -count=1`
at `b43a746`, each mutant verified present on disk carrying the mutated text and
compiled before measurement:

| Mutant | Pre-fix | Post-fix |
| --- | --- | --- |
| `command != CommandDoctor` (spare the one covered invocation — control) | KILLED | KILLED |
| `command != CommandList` (spare a different invoked command) | **SURVIVED** | KILLED |
| `result.Command() != CommandTakeover` (admit any document claiming takeover) | **SURVIVED** | KILLED |

Reports: `repro-r5.json` (pre-fix), `postfix-r5.json` (post-fix).

### Fix

`TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus/document reports
another command` now drives the agreement over the whole implemented vocabulary
in **both** directions: every implemented tag as the invocation against every
other implemented tag's own emitted document. 306 ordered pairs, each required
to refuse with `ErrOutcomeDisagreement` **and** to carry this guard's own
sentence `stdout reports command %q and the invocation was %q`, so a pair
settled by the version binding or a closed decoder does not count as coverage of
this guard. The 18 agreeing pairs are required to be admitted in the same loop,
which is the positive control against a guard that refuses everything. Counts
are asserted against `ImplementedCommands()`, and `takeover`, `list` and
`doctor` are asserted present in the swept vocabulary so a shrunken enumeration
cannot satisfy the counts quietly.

Every document is emitted through the production emitter for its own tag, so the
only thing a pair moves is the command tag.

### The guard's share of the vocabulary, measured

`TestTheCommandAgreementGuardOwnsAMeasuredShareOfTheTagVocabulary` drives all
44x44 = 1,936 ordered pairs over the registered vocabulary and classifies every
answer:

| Class | Pairs | Settled by |
| --- | ---: | --- |
| Admitted (the agreeing diagonal) | 18 | — |
| Refused by the command-tag agreement | 306 | `readSuccess` |
| Refused before stdout is read | 1,144 | `VersionForCommand` — the invoked tag selects a version this repository does not build |
| Refused by the version binding | 468 | the document claims a tag bound to another CLI Result major |

Nothing outside the diagonal is admitted. Every figure is derived from
`Commands()` and `ImplementedCommands()`, and each non-guard class asserts the
refusal does **not** quote the guard's sentence, so a bypass cannot be counted
as an earlier refusal.

**Stated bound.** For a pair involving one of the 26 registered-but-unimplemented
tags the guard is never the enforcement. That is not a blind spot in the sweep:
it is a measured class with a named earlier refusal behind it. Such a pair can
only be driven with a forged document, because this repository builds no body
for those tags at all.

## Observation A — extended to the sibling reader and closed, not left as a bound

The reviewer measured that narrowing `axerror.requireCommonDataModel` to skip
documents over 4096 bytes SURVIVES, because every duplicate-member fixture is
hand-built and short, and declined to block because byte length is not a Section
1.6 dimension.

Measured here: **the identical narrowing on `documentSchema`'s copy of the same
gate in `cliresult` also survived.** The finding was reported for one reader and
holds in both.

Both are closed with a conforming document several times that size:

- `axerror`: a Structured Error padded through its 64-key `details` bound
  (`largeConformingDocument`), duplicated on `retryable`, `code` and `exit_code`.
- `cliresult`: a CLI Result carrying twelve Session Summaries, emitted through
  the production emitter, with `schema` repeated as the Structured Error schema
  (`TestADuplicateSchemaMemberCannotSelectTheBranchInALargeDocument`).

Each row asserts the undoubled document is admitted first, so it cannot pass for
an unrelated reason, and asserts the byte length so it cannot silently shrink.

**A fixture whose size has two independent sources cannot be shown to fail
closed.** The first padding helper padded through both the 4096-character
`message` bound and the `details` map, so narrowing either loop alone left the
document over 4096 and the mutant survived as an equivalent one. Padding through
`details` alone gives the size one knob, and the narrowed loop now reddens the
row's own length assertion.

## Observation B — left alone, as instructed

Setting `Reading.exitStatus` from `failure.ExitCode()` instead of
`output.ExitStatus` is provably equivalent: `readFailure` has already refused
unless the two are equal. Not touched. The same holds for the survivors the
round-2 and round-3 reviewers dismissed.

## Mutation campaign

`.temp/TASK-260830-33sfxc/r5/mutants-r5.py` — the round-4 harness plus nine new
rows. Each mutant is verified present on disk carrying the MUTATED text, and
compiled, before it is measured.

| Figure | Round 4 | Round 5 |
| --- | ---: | ---: |
| Mutants | 44 | 53 |
| Killed | 43 | 52 |
| Declared subsumed | 1 | 1 |
| Unexplained survivors | 0 | 0 |
| Narrowing (by the harness's own definition) | 27 | 36 |

Harness exit code 0; restored suite exit code 0. Report:
`mutation-report-r5.json`.

New rows:

1. `command-agreement-narrowed-to-spare-one-invoked-command`
2. `command-agreement-narrowed-to-admit-forged-takeover-documents` — the finding
3. `command-agreement-narrowed-to-the-single-pair-previously-covered` — the guard restricted to exactly its old coverage
4. `command-agreement-sweep-enumerator-narrowed-in-the-test` — drops `takeover` from the swept vocabulary
5. `command-agreement-grid-denominator-narrowed-in-the-test` — measures the ratio over the implemented tags instead of the registered ones
6. `axerror-common-data-model-gate-narrowed-by-payload-size`
7. `cliresult-discriminator-gate-narrowed-by-payload-size`
8. `large-document-padding-narrowed-in-the-test`
9. `large-result-padding-narrowed-in-the-test`

The declared-subsumed row is unchanged: removing the exit-0 guard in
`readFailure` is refused one guard later by the `exit_code` equality check,
because `decodeExitStatus` never admits a Structured Error carrying `exit_code` 0.

## Traceability

Both new tests register against **existing** acceptance cases, so no coverage
figure moved:

- `cli-result-machine-reading` gains `TestTheCommandAgreementGuardOwnsAMeasuredShareOfTheTagVocabulary`
- `structured-error-common-data-model` gains `TestADuplicateSchemaMemberCannotSelectTheBranchInALargeDocument`

`reviewedOwnershipCanonicalSHA256` re-pinned
`d9a860aa…307d` -> `0c8a3014…6e31`.

`tracecheck` still reports `acceptance_cases=74`, `clauses_discharged=17/403`,
`normative_sections=36`, `bindings=49` — all unchanged, so no new capability is
advertised and every README ownership figure stays correct.

`ownership-pin-probe-r5.json`: each new registration was probed by renaming its
Go declaration. Both make `tracecheck` exit 1 **naming the missing declaration**
(`declaration "…" is absent from "…"`), ahead of the projection-digest check, so
neither registration is self-minted.

## Published-figure updates

`README.md` and `LOGBOOK.md` publish the harness counts. Both moved 44/43/27 ->
53/52/36, and both restate the existing bound: the harness mutates the working
tree, lives under `.temp/`, and no committed artifact derives its counts. Every
README figure that CAN be derived from a committed measurement still is —
`TestREADMEFanInTableIsDerivedFromTheMeasuredProjection` and
`TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport` both pass unchanged.

## Validation, exit codes reported as measured

| Command | Exit |
| --- | ---: |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l .` (excluding `.temp`) | no files |
| `go test ./... -count=1` | 0, 13 packages |
| `go test ./... -cover -count=1` | 0 — `cliresult` 95.5%, `axerror` 99.5%, both unchanged |
| `go run ./internal/traceability/cmd/tracecheck` | 0 |
| `python3 .temp/TASK-260830-33sfxc/r5/mutants-r5.py` | 0 |

## Disclosures

- The `cliresult` payload-size hole was found by extending the reviewer's
  `axerror` observation to the sibling gate rather than reported by the review.
  It was closed rather than recorded as a bound.
- No durable state is mutated by anything in this leaf, so there is no
  crash/idempotency surface to evidence.
- `clauses_discharged` stays at 17/403 and Section 14.2 stays partial at 8/9.
  This round adds evidence for an existing claim and claims nothing new.
- Nothing accepted in rounds 1-4 was weakened; the round-4 harness rows all
  still run and all still kill.
