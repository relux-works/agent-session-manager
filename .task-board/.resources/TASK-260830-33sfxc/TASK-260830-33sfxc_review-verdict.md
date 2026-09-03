# TASK-260830-33sfxc — reviewer verdict: CHANGES REQUESTED

- Change Request: `CR-TASK-260830-33sfxc-1` revision 1
- Reviewer run: `RUN-260903-29885b`
- Reviewed delta: `d1d3ece..49da1aa` (this leaf's own commit is `b21fede`)
- Verdict: **changes requested** → `to-dev`. Two confirmed findings, one non-blocking nit.

## What was verified and reproduced

All checks were re-run by the reviewer in the story worktree, not accepted from
attached logs.

| Check | Command | Result |
| --- | --- | --- |
| Build / vet / format | `go build ./...`, `go vet ./...`, `gofmt -l .` | clean |
| Full suite with coverage | `go test ./... -cover` | 13 packages ok; `cliresult` 95.5%, `axerror` 99.5% |
| Traceability | `go run ./internal/traceability/cmd/tracecheck` | exit 0, `acceptance_cases=73`, `clauses_discharged=17/403` — matches the claim exactly |
| Producer mutation harness | `TASK-260830-33sfxc_mutants.py`, re-run independently | 14 mutants, 13 killed, 1 declared subsumed, 0 unexplained survivors, suite restored green — reproduces exactly |
| Spec fixture provenance | `cli-result-1.0.0-takeover.json` vs SPEC.md §14.2, `error-1.0.0-workspace-conflict.json` vs SPEC.md §15.1 at pinned commit `28bf96d` | byte-for-byte verbatim; the "normative example, verbatim" claim is honest |

### Reviewer's own narrowing mutants (the producer harness is mostly `if false`)

The attached harness proves the gates exist; these narrow them instead, to test
the class each gate covers rather than its presence.

| Mutant | Result |
| --- | --- |
| `readFailure` exit_code equality narrowed to `&& output.ExitStatus == 5` | KILLED by `TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus` |
| Unregistered-status gate narrowed to admit exit 1 | KILLED by the same test |
| Absence branch returns `ErrUnreadableDocument` | KILLED by `TestReadDistinguishesAbsenceFromAReadFailure` |
| `Read` classifies from the exit status when stdout is empty (the bypass) | KILLED by `TestAMisroutedDocumentLeavesTheMachineClientWithNothing` and the absence rows |
| `Retryable()` computed from message text (`strings.Contains(msg, "not_found")`) | KILLED by `TestMessageTextChangesNoMachineAnswer` |
| `verifyClosedMembers` restored ahead of `verifyEnvelopeIdentity` | KILLED by `TestUnsupportedMajorIsSettledBeforeTheClosedMemberRule` |

The core claims of the leaf hold up under attack. The message-independence,
stderr-independence, exit-status-fan-in, historical-corpus, and
identity-precedence work is good, and the traceability gap text for §17.2 and
§14.2 is honest — it narrows the gap and explicitly does not claim to close it.

## Finding 1 (blocking) — `Read` admits a duplicate-keyed Structured Error, and machine answers change with it

`internal/cliresult/client.go:139` / `internal/axerror/decode.go:74`

SPEC.md:221 puts duplicate keys outside the common logical data model:
"floating-point numbers, NaN, Infinity, non-string map keys, and duplicate keys
are forbidden". `cliresult.Decode` enforces that — `decode.go:86` runs
`canonicaljson.Canonicalize`, pinned by `TestDuplicateAndMalformedDocumentsAreRefused`.
`axerror.Decode` never calls `canonicaljson` (0 references in the file), so the
failure branch of the *same* production entry point does not enforce it.

Reproduced through `cliresult.Read`, driving the frozen fixture
`error-1.0.0-workspace-conflict.json` at its own exit status 5:

| Duplicate member appended | `Read(CommandMaterialize, …, ExitStatus: 5)` |
| --- | --- |
| `"code":"native_store_conflict"` (same exit-5 class) | **ADMITTED**, `Code()` = `native_store_conflict`; the document's first occurrence was `workspace_conflict` |
| `"retryable":true` (document declares `false`) | **ADMITTED**, `Retryable()` = `true` — a forged retry claim through a duplicate member |
| `"details":{"remediation":"…"}` | **ADMITTED**, `DetailKeys()` = `[expected_checkpoint remediation remediations]` — the union of both occurrences, neither one alone |
| the same shape on a CLI Result (`"ok":true` duplicated) | refused: `invalid cli result: document is outside the common data model: invalid canonical JSON input: duplicate object member "ok"` |

Three things make this in scope for this leaf rather than a pre-existing sibling
defect to be waved through:

1. **The exit-status corroboration this leaf added does not catch it.** The
   duplicate `code` row is admitted precisely because both occurrences share the
   exit-5 class, so `readFailure`'s `exit_code`-equality check passes.
2. **`client.go` states the delegation as its justification, and the delegation
   does not happen on one of its two branches.** `client.go:131-134`: "the
   closed decoder for the selected contract then validates the whole object,
   including the duplicate members and trailing content this discriminator
   deliberately does not settle on its own." That is true for `Schema` and false
   for `axerror.Schema`. The comment is new in this CR and is the reason
   `documentSchema` does not settle duplicates itself.
3. **The claim under test is exactly this one.** The leaf's deliverable is that
   a machine client's answers do not depend on anything but the document. Here
   two conforming readers — first-wins and last-wins — get a different `code`
   and a different `retryable` from the same bytes, and `details` resolves to
   neither occurrence because `parseEnvelopeIdentity`'s map decode and
   `decodeClosedDocument`'s struct decode resolve repeats differently within one
   document. The rule is enforced in one context profile and absent in the
   other, which is a bypass path regardless of which leaf first opened it.

`axerror.Decode` is also the reader for peer-supplied provider, bridge, rpc,
session-adapter and terminal-backend envelopes, so the forged-`retryable`
shape is reachable from a remote peer, not only from a local CLI pipe.

Requested: route `axerror.Decode` through the same common-data-model gate the
CLI Result reader uses (or an equivalent duplicate-member refusal), extend the
`Read` refusal table and the axerror precedence rows with duplicate-member
negatives for `code`, `retryable`, `details` and `schema`, and correct the
`documentSchema` comment so it describes what each branch actually validates.
Add the corresponding narrowing mutant to the harness.

## Finding 2 (blocking) — the README fan-in table's "Largest class" column is wrong for two of four rows

`README.md` (fan-in table) and `LOGBOOK.md` entry 1003

Measured from `axerror.CodesByExitStatus`, no ties at the maximum:

| Version | README / LOGBOOK claim | Measured | Verdict |
| --- | --- | --- | --- |
| `1.0.0` | 6 codes at exit 6 | 6 at exit 6 | correct |
| `1.1.0` | 9 codes at exit 6 | 9 at exit 6 | correct |
| `1.2.0` | 12 codes at exit 6 | **14 at exit 16** | wrong count and wrong status |
| `1.3.0` | 15 codes at exit 6 | **17 at exit 6** | wrong count |

The 1.3.0 error is repeated in LOGBOOK entry 1003 ("1.3.0 reaches 109 codes with
15 codes at exit 6").

`TestCodesByExitStatusMeasuresTheFanIn` asserts statuses/codes/ambiguous for all
four versions — those columns are correct and pinned. But `largest`/`largestFor`
are asserted only for 1.0.0 and 1.1.0, in
`TestExitStatusAloneIdentifiesNoFailure`, which is why the 1.2.0 and 1.3.0 rows
drifted into prose nobody measures. This is the leaf's own evidence standard —
a measured ratio, not prose — failing on the artifact the DoD asks to be
"updated without unsupported claims".

Requested: correct both rows in README and LOGBOOK, and extend the fan-in test
to assert `largest`/`largestFor` for all four registered versions so the table
cannot drift again.

## Finding 3 (non-blocking nit) — the absence/read-failure distinction is asserted only positively

`internal/cliresult/client_test.go:145-160`

`TestReadDistinguishesAbsenceFromAReadFailure` checks `errors.Is(err, want)` per
row and never checks the negative. Aliasing the two sentinels
(`ErrAbsentDocument = ErrUnreadableDocument`) survives the entire suite. The
realistic drift — the production branch returning the wrong sentinel — is
killed, so nothing is broken today; a single mutual-exclusion assertion would
close the gap the doc comment ("It is deliberately distinct from
ErrUnreadableDocument") claims is closed.

## Observation for the orchestrator — not a producer finding

I initially read the Change Request base OID `d1d3ece` as a stale checkpoint.
It is not. `task-board worktree status` for `WS-4fa8daeca025` shows
`checkpoint_oid` correctly reconciled to `f34d91d`, while the CR was built from
`current_base_oid` = `selected_base_oid` = `origin/main` = `d1d3ece`. For a
`story_final` Change Request that is the right base: the Story's integration
unit is everything the Story adds to trunk, so the 66-path / 13,845-insertion
delta is correct, not inflated.

What is worth the orchestrator's attention is the other two rows in the same
workspace record:

| Change Request | base_oid | repository_delta |
| --- | --- | --- |
| `CR-TASK-260830-34elja-2` (leaf 1) | `ebc4e31` | **empty** |
| `CR-TASK-260830-1bipsa-1` (leaf 2) | `f34d91d` | **empty** |
| `CR-TASK-260830-33sfxc-1` (leaf 3) | `d1d3ece` | present, 66 paths |

Each sibling CR was based on its *own* leaf commit, so its delta was empty and
its reviewer had no CR diff to read — the `#113` defect the brief describes.
This `story_final` CR is therefore the first time `internal/axerror` and
`internal/cliresult` appear inside a reviewable delta at all.

That is directly why Finding 1 lands here rather than against leaf 1. The
reviewer brief asked for exactly this: "probe the two inherited packages through
the new consuming entry point, which is a path they had no caller for until
now." `cliresult.Read` is that caller, and driving `axerror.Decode` through it is
what surfaced the duplicate-member hole. It is in scope for this leaf both
because the CR delta contains it and because the leaf's own new production code
documents a validation it does not receive.

Review depth was concentrated on `b21fede` (this leaf's 31 files); leaves 1 and
2 were spot-checked through the new entry point rather than re-reviewed in full,
since both are `done` and were reviewed by hand at the time. That is a stated
bound on this verdict, not a claim of full coverage over all 66 paths.
