# TASK-260830-33sfxc — round-3 rework results

Commit: `1d68325` (amend of `209e0f2`; one commit past the Story checkpoint
`f34d91d`, signed and `git verify-commit` clean).

Two blocking findings, both surviving **narrowing** mutants on the two
Section 14.2 outcome guards this leaf exists to prove. **No production code
changed** — the implementation was correct in both cases and the reviewer said
so. What was wrong was the evidence, measured against the standard this leaf
itself published: "a gate whose only evidence is that deleting it reddens
something has been shown to exist, not to hold."

## Reproduced first, on the unchanged tree

Before writing a line of test, both mutants were applied to `209e0f2`, verified
present on disk, compiled, and measured against
`go test ./internal/cliresult ./internal/axerror -count=1`
(`.temp/TASK-260830-33sfxc/repro-r3.py`):

| Mutant | On `209e0f2` | After this rework |
| --- | --- | --- |
| `readFailure` equality narrowed to `failure.CodeRegistered() && …` | SURVIVED (exit 0) | KILLED (exit 1) |
| `readSuccess` guard narrowed past exit 130 | SURVIVED (exit 0) | KILLED (exit 1) |

Suite restored green after each, exit 0.

## F1 — the §14.2 equality was proven only for registered codes

`axerror.decodeBody` cross-checks a document's `exit_code` against the pinned
registry **only when `ExitCodeFor` resolves**. For a code a later compatible
minor added it takes the `ErrUnregisteredCode` branch and runs no exit check at
all — Section 15.3 keeps the *envelope's exit class*, and the class is the
status, not the code. So for a registered code the equality is bound twice, and
for an unregistered one `readFailure` is the **sole** enforcement. That is
precisely the row the narrowing left uncovered.

Concretely, under the mutant: the frozen `error-1.0.0-later-minor-code.json`,
which declares `exit_code` 10 (ownership/lease/fencing), is admitted at observed
exit 5 (workspace/native-store conflict, "no silent overwrite") and reported to
a machine client as exit class 5. Two different remediations; the wrong one
chosen.

The same insight had already been had once on this leaf and not carried across:
`error-1.0.0-later-minor-code-forged-retry.json` exists because an unregistered
code must not be a way past the §15.3 RETRY rule. The identical reasoning
applies to the EXIT-CLASS rule.

**Fix.** `TestTheExitStatusEqualityBindsACodeTheRegistryDoesNotCarry`
(`internal/cliresult/client_test.go`) drives the frozen envelope through the
production entry point `cliresult.Read` at **all sixteen** other registered
§15.2 failure statuses, requires `ErrOutcomeDisagreement` and requires the
refusal to name the equality it enforces. It first asserts `CodeRegistered() ==
false` on the conforming read, so if that code is ever registered the row fails
loudly instead of quietly ceasing to cover its class.

## F2 — the success-at-a-failure-status guard was proven at one status

Exit 130 is the §15.2 row "Interrupted by operator signal before a clean
response; inspect authority before retry" — the one failure status at which a
success document plausibly *does* reach stdout before the process dies, which
makes a narrowing there a realistic bypass rather than an artificial one.

**Fix.** The `success object at a failure status` row and the `failure exit_code
differs from the process status` row both now sweep **every** registered failure
status, enumerated through the production predicate `axerror.IsFailureExitStatus`
by one helper, `registeredFailureExitStatuses`, that asserts the count is 17 and
that exit 130's §15.2 meaning resolves. A predicate answering false for
everything would redden rather than turning each loop into a green test that
drove nothing.

## The enumerator is measured too

A shared helper that decides how much a sweep covers is itself a gate.
Narrowing its range to `status <= 20` drops exit 130 out of every loop that
draws on it; the asserted count is what makes that a red rather than a silently
smaller sweep. That narrowing is mutant 5 and is killed.

## Mutation harness

`.temp/TASK-260830-33sfxc/mutants-r3.py`, grown from 23 to **28** mutants. Each
is verified applied, verified to carry the MUTATED text on disk (not merely to
differ from the original), and compiled before it is measured.

| | Count |
| --- | ---: |
| Mutants | 28 |
| Killed | 27 |
| Declared subsumed | 1 |
| Unexplained survivors | 0 |
| Narrowing rather than deleting | 14 |
| Restored suite exit code | 0 |

The subsumed one is unchanged from the previous round: removing the exit-0 guard
in `readFailure` leaves the suite green because `decodeExitStatus` never admits a
Structured Error carrying `exit_code` 0, so the equality check refuses the same
input one guard later.

Five mutants are new, all narrowing:

1. `exit-status-equality-narrowed-to-registered-codes` — F1's exact mutant.
2. `exit-status-equality-narrowed-to-spare-one-status` — the same guard narrowed
   by status instead of by registration, which a single-sample row would have
   survived.
3. `success-at-a-failure-status-narrowed-past-the-interrupted-row` — F2's exact
   mutant.
4. `success-at-a-failure-status-narrowed-past-an-ordinary-status` — so the row is
   not proven only at the one status the finding named.
5. `failure-status-enumeration-narrowed-below-the-interrupted-row` — the shared
   enumerator.

## The two dismissed survivors were left alone

As the brief instructed: the `details`-bearing narrowing of
`requireCommonDataModel` is near-equivalent on conforming input, and pre-seeding
an empty group per failure status in `CodesByExitStatus` is genuinely equivalent
against today's registry. A survivor dismissed with a reason is evidence; a
survivor quietly fixed hides why it survived.

## Traceability

The new test is registered in the `cli-result-machine-reading` acceptance case
and `reviewedOwnershipCanonicalSHA256` re-pinned to
`f2bc887e0d61066993e01b9198b8361f1dae05e50bd3a2923e435dbb630f18f7`.

That registration was not assumed to be checked. Probe
(`.temp/TASK-260830-33sfxc/ownership-pin-probe.py`, output
`ownership-pin-probe.json`): renaming the declaration to one that does not exist
makes `tracecheck` exit 1 with

    acceptance case "cli-result-machine-reading" test owner: declaration
    "TestTheExitStatusEqualityBindsACodeTheRegistryDoesNotCarryX" is absent from
    "internal/cliresult/client_test.go"

— the declaration-resolution check answers **before** the digest check does, so
the registration resolves against the real Go declaration rather than being
prose the digest freezes. Restored: exit 0.

`acceptance_cases` stays 74 and `clauses_discharged` stays 17/403: a test added
to an existing acceptance case discharges no new clause, and claiming otherwise
would be the unsupported-capability shape this Story exists to refuse.

## Validation, re-run on this exact tree

Every command below was run as a standalone process with its real exit status
reported.

| Command | Exit | Note |
| --- | ---: | --- |
| `gofmt -l ./internal` | 0 | no output |
| `go build ./...` | 0 | |
| `go vet ./...` | 0 | |
| `go test ./... -count=1` | 0 | 13 packages |
| `go test ./... -cover -count=1` | 0 | `cliresult` 95.5%, `axerror` 99.5%, both unchanged |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `acceptance_cases=74`, `clauses_discharged=17/403` |
| `python3 mutants-r3.py` | 0 | 28 mutants, 27 killed, 1 subsumed, 0 unexplained survivors |

Coverage did not move, and that is expected rather than disappointing: the new
rows drive statements the suite already executed. What moved is the *class* each
guard is proven over, which a statement-coverage number cannot express.

`gofmt -l ./internal ./cmd` was not run: this module has no `cmd/` directory at
the repository root, so that form exits 2 on a missing path. `./internal` is the
whole Go tree here.

## Review patches

The stored CR patch predates this amend. `TASK-260830-33sfxc_rework-r3-delta.patch`
is `git diff --binary 209e0f2..1d68325` — 5 files, 188 insertions, 7 deletions,
the round-3 delta only and the shortest honest read for this round.
`TASK-260830-33sfxc_leaf-full-r3.patch` is `git diff --binary f34d91d..1d68325`,
the whole `story_final` leaf against its checkpoint. Use these until a CR
revision exists whose `candidate_tree_oid` equals this HEAD's tree.
