# TASK-260830-33sfxc — rework results (CR-TASK-260830-33sfxc-1 rev1, CHANGES REQUESTED)

Commit `209e0f2` on `task-board/story/STORY-260830-2jylym`, parent `f34d91d`
(the recorded checkpoint), single parent, signed, worktree clean. The previous
head `b21fede` was amended rather than followed by a second commit, so the leaf
stays exactly one commit past the checkpoint. `git rev-list --parents -n 1 HEAD`
reports one parent; `git verify-commit HEAD` exits 0 with a Good signature for
`oparin@me.com`, ECDSA `SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM`.

**READ THIS BEFORE REVIEWING THE DIFF.** `CR-TASK-260830-33sfxc-1` rev1 still
records `candidate_tree_oid 49da1aa`, which is the PRE-amend tree, and
`task-board handoff` exited 0 without constructing a new revision — Change
Request construction happens at spawn-run completion, not at handoff. The tree
to review is `HEAD^{tree}` = `c4967d95623cc20e4ffae2121065ba34cae1eb58`. Compare `candidate_tree_oid` against it
before trusting any stored patch; if a rev2 was not built for `209e0f2`, the
stored `TASK-260830-33sfxc_change-request_rev1.patch` describes the superseded
tree and must not be reviewed as if it were this work.

Every finding in the verdict is closed. Nothing was waived.

---

## F1 (BLOCKING) — Read admitted a duplicate-keyed Structured Error

### Reproduced first, then fixed

Driven through `cliresult.Read` against the frozen `error-1.0.0-workspace-conflict.json`
fixture at its own exit status 5, on the pre-fix tree:

```
dup code same class    ADMITTED code="destination_not_empty" retryable=false details=[expected_checkpoint remediations]
dup retryable          ADMITTED code="workspace_conflict"    retryable=true  details=[expected_checkpoint remediations]
dup details            ADMITTED code="workspace_conflict"    retryable=false details=[expected_checkpoint observed_checkpoint remediations]
dup exit_code          ADMITTED code="workspace_conflict"    retryable=false details=[expected_checkpoint remediations]
dup schema             REFUSED  ErrOutcomeDisagreement (the wrong fact: an exit-status disagreement for a document whose schema was never settled)
```

The `details` row is the one that is not a last-wins resolution at all:
`encoding/json` MERGES a repeated map-typed member, so the reader saw the union
of both occurrences — three keys where each occurrence declared fewer — a detail
set no writer emitted and neither occurrence declared.

After the fix every one of those rows is refused with `ErrUnreadableDocument`
naming the repeated member.

### Root cause

`cliresult.Decode` ran `canonicaljson.Canonicalize`, whose strict decoder refuses
a duplicate object member at any depth. `axerror.Decode` referenced
`canonicaljson` nowhere. So one branch of the same reading enforced the Section
1.6 common logical data model and the other did not, and the exit-status
corroboration this leaf added cannot see the difference, because both
occurrences of a repeated `code` can share one exit class.

`client.go`'s justification for not settling duplicates in the discriminator —
"the closed decoder for the selected contract then validates the whole object,
including the duplicate members … this discriminator deliberately does not settle
on its own" — was true of `Schema` and false of `axerror.Schema`, and was written
in the same change that shipped the gap.

### Reach

`axerror.Decode` is reached through `DecodeBound` for peer-supplied provider,
task-board bridge, Mesh RPC, session-adapter and terminal-backend envelopes, so
the forged-retryable shape arrived from a **remote peer**, not only from a local
pipe. `TestABoundPeerCannotSendADuplicateKeyedEnvelope` drives all fifteen bound
contracts and requires each to refuse.

### Fix

- `axerror.requireCommonDataModel` runs `canonicaljson.Canonicalize` on the
  caller's bytes at the top of `Decode`, ahead of the envelope identity. One
  place covers every peer surface.
- `cliresult.documentSchema` runs the same gate before it reads `schema`.
  A repeated `schema` selects the branch, and neither closed decoder can settle
  that for the discriminator, because both run after the branch is chosen.
  `TestADuplicateSchemaMemberCannotSelectTheBranch` pins that the answer is
  `ErrUnreadableDocument` and explicitly NOT `ErrOutcomeDisagreement`.
- The `documentSchema` comment now states what each branch actually validates,
  names both gates, and records that the earlier sentence was true of one branch
  and false of the other.

### Two design decisions inside the fix, both pinned

**The canonical bytes are discarded, not adopted.** RFC 8785 §3.2.2.3 serializes
every number through `Number.prototype.toString`, so the transform rewrites `1e1`
to `10` and `9.0` to `9`. `decodeExitStatus` reads `exit_code` from its raw bytes
precisely so that the exponent and the point are refused rather than normalized,
and a gate that adopted its own output would be a **widening** dressed as
validation — it would hand the reader a document those literals had been
laundered out of. `TestTheCanonicalGateDoesNotLaunderTheExitStatusToken` proves
both halves per row: that `Canonicalize` really does rewrite the literal (so the
row exercises the laundering it claims to prevent rather than a token nothing
would have changed), and that `Decode` still refuses it. The harness carries the
adopt-the-canonical-bytes mutant.

**The gate runs before the envelope identity.** A document whose `schema_version`
repeats has no version. An identity-first order resolves the repeat to the last
occurrence and then answers `ErrUnsupportedMajor` or `ErrVersionMismatch` — a
version fact derived from one of two members it had no basis for choosing
between. Both orders refuse, so this is not a bypass either way; what changes is
whether a caller asking "is my `ax` too old for this output" is told a version it
can act on or told that the bytes carry none.
`TestADuplicateIdentityMemberIsNotResolvedIntoAVersionFact` asserts the
data-model fact and explicitly NOT the version fact, on both a major
disagreement and a minor disagreement, and the reorder mutant is killed.

### A branch that moved rather than being left unreachable

With the gate in front of it, `parseEnvelopeIdentity`'s trailing-content guard
became a branch no test could redden, so it was removed and the gate owns
trailing content. The coverage did not leave with it:
`TestTheCommonDataModelGateCoversMoreThanDuplicates` drives trailing content,
a trailing token, a lone surrogate escape, invalid UTF-8 and an unescaped control
character through `Decode`, and the delete-the-gate mutant reddens the first four
plus both existing trailing-content rows in `decode_test.go` and
`precedence_test.go`.

**Stated bound, measured rather than assumed:** the control-character row does
NOT redden without the gate, because `validateMessage` refuses that byte
independently for the `message` member. It is recorded as a subsumed case in the
test and in the production comment, not counted as coverage the gate provides.

### Tests added for F1

| Package | Test | What it would catch |
| --- | --- | --- |
| `axerror` | `TestDecodeRefusesADuplicateMemberOnEveryDeclaredMember` | all nine closed members plus one nested inside `details`; each row also asserts the undoubled document is admitted, and that the refusal names the repeated member |
| `axerror` | `TestADuplicateMemberCannotForgeARetryClaim` | the machine answer, not the parse: measures that `encoding/json` still resolves the repeat to `true` before requiring `Decode` to refuse |
| `axerror` | `TestADuplicateMemberCannotSplitTheDetailSet` | measures the 2-key union before requiring the refusal |
| `axerror` | `TestABoundPeerCannotSendADuplicateKeyedEnvelope` | all fifteen bound contracts through `DecodeBound` |
| `axerror` | `TestADuplicateIdentityMemberIsNotResolvedIntoAVersionFact` | gate-before-identity ordering |
| `axerror` | `TestTheCanonicalGateDoesNotLaunderTheExitStatusToken` | the adopt-the-canonical-bytes widening |
| `axerror` | `TestTheCommonDataModelGateCoversMoreThanDuplicates` | what else leaves with the gate |
| `axerror` | 5 new rows in `TestReorderingTheIdentityCheckAdmitsNothingItUsedToRefuse` | duplicate `schema`/`schema_version`/`code`/`retryable`/`details` |
| `cliresult` | `TestReadRefusesADuplicateKeyedDocumentOnEitherBranch` | 12 rows, six per branch, all through `Read` |
| `cliresult` | `TestADuplicateMemberCannotForgeARetryClaimThroughRead` | the forged claim at the production entry point |
| `cliresult` | `TestADuplicateSchemaMemberCannotSelectTheBranch` | branch selection from an ambiguous member |
| `cliresult` | `TestBothClosedDecodersEnforceTheSameDataModel` | the delegation claim, checked against both delegates |

The duplicate documents are built by textual rewrite, because no builder in this
repository can emit the shape — a Go map has one value per key — which is also
why it only ever arrives from a writer that composed the bytes itself. The helper
appends the second occurrence as the LAST member, because a prepended copy would
leave the original value winning and the row would pass for the wrong reason; the
first draft of the helper prepended and the forged-retryable row failed, which is
what caught it.

---

## F2 (BLOCKING) — the fan-in table was wrong in two of four rows

Measured independently from `axerror.CodesByExitStatus` before anything was
changed (`fan-in-measured.log`):

| Version | Statuses | Codes | Ambiguous | Published largest | Measured largest | Verdict |
| --- | ---: | ---: | ---: | --- | --- | --- |
| `1.0.0` | 17 | 47 | 14 | 6 codes at exit 6 | 6 codes at exit 6 | correct |
| `1.1.0` | 17 | 66 | 14 | 9 codes at exit 6 | 9 codes at exit 6 | correct |
| `1.2.0` | 17 | 94 | 15 | 12 codes at exit 6 | **14 codes at exit 16** | wrong count AND wrong status |
| `1.3.0` | 17 | 109 | 15 | 15 codes at exit 6 | **17 codes at exit 6** | wrong count |

No ties at the maximum in any version.

The `1.3.0` error had also been copied into Logbook entry 1003.

Both rows are corrected in `README.md` and the sentence in `LOGBOOK.md`. That is
the smaller half of the fix. The larger half is that the rows are no longer
prose:

- `TestREADMEFanInTableIsDerivedFromTheMeasuredProjection` locates the table by
  its exact header, parses every body row, and re-derives every cell from
  `CodesByExitStatus`. It requires every registered version to appear exactly
  once, refuses a published version the registry does not carry, refuses a
  registry version the table omits, and refuses a version whose maximum ties
  across two statuses — because "the largest class" names one status and would
  otherwise be ambiguous, which is reported as unknown rather than resolved by
  picking the lower status.
- `TestLogbookFanInFiguresAreDerivedFromTheMeasuredProjection` does the same for
  the two figures entry 1003 states in prose.
- `TestExitStatusAloneIdentifiesNoFailure` now covers all four registered
  versions with `largest`/`largestFor`, not two, and asserts the no-tie property
  per version.

Three mutants prove the derivation is not decorative: restoring the wrong `1.2.0`
figure, restoring the wrong `1.3.0` Logbook figure, and deleting the `1.3.0`
README row outright. All three are killed.

---

## F3 (NON-BLOCKING) — closed, and the first fix for it was itself defective

`TestReadDistinguishesAbsenceFromAReadFailure` asserted `errors.Is` membership
only, so aliasing `ErrAbsentDocument` onto `ErrUnreadableDocument` survived the
whole suite.

The first mutual-exclusion fix **also survived its own mutant**. It skipped "the
sentinel this row wanted" by comparing error VALUES, and under the alias the two
values are equal, so it skipped precisely the comparison that would have caught
the alias. Caught by running the mutant rather than by reading the fix.

The rows now name their sentinel as a string and the exclusion loop skips by
name. `TestTheObservationSentinelsAreDistinctValues` closes it one level up over
all five reader sentinels, requiring both distinct identities and distinct text.
The `absence-aliased-onto-the-read-failure-sentinel` mutant is killed.

---

## A survivor the F1 fix created, and how it was closed rather than declared

Adding the data-model gate to `documentSchema` made the previously-killed
`second-document-on-stdout-admitted` mutant survive: the gate refuses the same
two-document stdout as trailing data one guard later.

It was not declared subsumed. The two guards report DIFFERENT facts — Section
14.2's "stdout carries more than one document, and Section 14.2 allows exactly
one" versus a data-model refusal — and which fact a caller is told is exactly
what this leaf spent its precedence work on. The `two documents` row now asserts
the refusal text, and the mutant is killed again.

---

## A false citation this rework caught in its own first draft

`requireCommonDataModel`'s first doc comment cited Section 1.6 as fixing the wire
format as "UTF-8 JSON restricted to the common logical data model". **That
sentence is nowhere in the pinned document.** It was a paraphrase wearing
quotation marks, caught by checking the citation through `internal/specdoc`
rather than by trusting the wording that felt right.

The two real fragments are `SPEC.md:218` "map keys MUST be UTF-8 strings and
MUST be unique" and `SPEC.md:221` "floating-point numbers, NaN, Infinity,
non-string map keys, and duplicate keys are forbidden", both in Section 1.6,
confirmed through `specdoc.Load` / `QuoteLines` / `SectionID`.
`TestTheCommonDataModelCitationsAreVerbatimInThePinnedDocument` locates both by
CONTENT rather than by line number, re-derives the section, and additionally
asserts the ABSENCE of the paraphrase so the correction cannot be undone by
someone who remembers the wording.

The clause at `SPEC.md:218` is a MUST, and `section:1.6` is bound to
`internal/scalar/scalar.go#ErrInvalidScalar` at `unevidenced`. This rework does
**not** claim it. The gate is a reader refusing a document, the binding names a
different production owner, and rebinding a section was outside this brief;
`clauses_discharged` therefore stays at 17/403. Recorded here so the choice is
visible rather than silent.

---

## Validation, each command run as a standalone process with its real exit code

| Command | Exit | Result |
| --- | ---: | --- |
| `gofmt -l .` | 0 | no output |
| `go build ./...` | 0 | |
| `go vet ./...` | 0 | |
| `go test ./... -count=1` | 0 | 13 packages, `go-test-full.log` |
| `go test ./... -cover -count=1` | 0 | `cliresult` 95.5%, `axerror` 99.5%, `go-test-cover.log` |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `acceptance_cases=74`, `clauses_discharged=17/403`, `tracecheck.log` |
| `python3 mutants.py` | 0 | 23 mutants, 22 killed, 1 subsumed, 0 unexplained survivors |

`tracecheck` re-pinned the ownership digest twice, once for the new acceptance
case and once when the citation test joined it.

`git verify-commit HEAD`: Good signature for `oparin@me.com`, ECDSA
`SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM`.

---

## Mutation campaign — 23 mutants

Each mutant is verified applied, verified to carry the MUTATED text on disk (not
merely to differ from the original), and compiled before it is measured. Nine
narrow rather than delete.

| Mutant | Status | Killed by |
| --- | --- | --- |
| `exit-code-equality-dropped` | KILLED | `TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus` |
| `absence-collapsed-into-read-failure` | KILLED | `TestReadDistinguishesAbsenceFromAReadFailure` |
| `success-admitted-at-a-failure-status` | KILLED | `TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus` |
| `failure-admitted-at-exit-zero` | **SUBSUMED** | see below |
| `unregistered-exit-status-admitted` | KILLED | `TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus` |
| `command-tag-agreement-dropped` | KILLED | `TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus` |
| `second-document-on-stdout-admitted` | KILLED | `TestReadDistinguishesAbsenceFromAReadFailure` |
| `error-version-taken-from-a-fixed-default` | KILLED | `TestReadSelectsTheErrorVersionFromTheCommandNotTheDocument` |
| `fan-in-narrowed-to-one-code-per-status` | KILLED | `TestCodesByExitStatusMeasuresTheFanIn` |
| `unregistered-version-answered-with-an-empty-group` | KILLED | `TestCodesByExitStatusRefusesAnUnregisteredVersion` |
| `historical-fixture-regenerated` | KILLED | `TestHistoricalEnvelopesRemainReadable` |
| `cliresult-member-rule-restored-before-the-identity-check` | KILLED | `TestUnsupportedMajorIsSettledBeforeTheClosedMemberRule` |
| `axerror-closed-decode-restored-before-the-identity-check` | KILLED | `TestUnsupportedMajorIsSettledBeforeTheClosedMemberRule` |
| `axerror-common-data-model-gate-removed` | KILLED | `TestDecodeRefusesADuplicateMemberOnEveryDeclaredMember` |
| `axerror-common-data-model-gate-narrowed-to-one-version` | KILLED | `TestDecodeRefusesADuplicateMemberOnEveryDeclaredMember` |
| `axerror-gate-adopts-its-own-canonical-bytes` | KILLED | `TestTheCanonicalGateDoesNotLaunderTheExitStatusToken` |
| `axerror-gate-narrowed-past-the-identity-check` | KILLED | `TestADuplicateIdentityMemberIsNotResolvedIntoAVersionFact` |
| `cliresult-discriminator-data-model-gate-removed` | KILLED | `TestADuplicateSchemaMemberCannotSelectTheBranch` |
| `absence-aliased-onto-the-read-failure-sentinel` | KILLED | `TestReadDistinguishesAbsenceFromAReadFailure` |
| `readme-fan-in-row-restored-to-its-published-error` | KILLED | `TestREADMEFanInTableIsDerivedFromTheMeasuredProjection` |
| `readme-fan-in-row-deleted` | KILLED | `TestREADMEFanInTableIsDerivedFromTheMeasuredProjection` |
| `logbook-fan-in-figure-restored-to-its-published-error` | KILLED | `TestLogbookFanInFiguresAreDerivedFromTheMeasuredProjection` |
| `stderr-member-added-to-the-observation` | KILLED | `TestMachineReadingCannotSeeStderr` |

The one subsumed mutant removes the exit-0 guard in `readFailure`. The
`exit_code` equality check refuses the same input one guard later, because
`decodeExitStatus` never admits a Structured Error carrying `exit_code` 0. The
invariant that makes it subsumed is itself asserted, so the day
`decodeExitStatus` admits 0 the subsumption claim reddens.

---

## Traceability

- New acceptance case `structured-error-common-data-model`, production
  `internal/axerror/decode.go#requireCommonDataModel`, ten tests across both
  packages. Attached to the `section:15.1` and `section:17.2` bindings as
  evidence.
- `structured-error-exit-status-fan-in` gained the two README/Logbook derivation
  tests.
- `acceptance_cases` 73 → 74. **`clauses_discharged` is unchanged at 17/403 on
  purpose**: the gate discharges no obligation the clause scanner measures, so
  claiming one would be the inflated-ratio defect this board has already shipped
  once. Ownership digest re-pinned; the two pinned counts in
  `traceability_test.go` and `tracecheck/main_test.go` updated.

---

## Disclosed, not fixed

- **`cliresult.parseDocument`'s trailing-content guard is unreachable.**
  `Decode` canonicalizes first and parses the canonical bytes, and canonical
  bytes never carry trailing content, so that branch cannot redden. This is
  leaf 2's code and predates this rework; it is the mirror of the branch removed
  from `axerror.parseEnvelopeIdentity` here. Not touched, because changing an
  accepted sibling's decoder was outside this rework's brief. The behaviour is
  correct either way — trailing content is refused by the canonical gate — so
  this is a dead branch, not a hole.
- **`cliresult.Decode` parses the CANONICAL bytes rather than the caller's.**
  That is the difference from the choice made here for `axerror`. It matters
  only where a number token's FORM is load-bearing, and the CLI Result body
  carries no such member (`exit_code` lives in the Structured Error, which is
  read by `axerror`). No defect was found, and no number-form assertion in
  `cliresult` was weakened; recorded so the asymmetry between the two readers is
  a stated decision rather than a discrepancy a later reader has to rediscover.
- **`14.2#6` remains partial at 8/9.** `Read` checks the exit-status equality
  from the consuming side, which narrows the clause to the missing `ax` process
  without closing it. No `ax` binary exists in this repository.
- **No durable state is mutated by anything in this leaf**, so there is no
  crash or idempotency surface to evidence.
