# TASK-260830-21gygk — qualified-selector contract decision

Run: RUN-260907-c21363. Active board goal: GOAL-260907-d1d459 revision 1.
Resolved scope: TASK-260830-1r9wrr, TASK-260830-21gygk, TASK-260830-wbpf1v.
The exact authoritative objective is retained in goal-checkpoint.txt. It requires
all assigned acceptance/checklist evidence at the role end status, or an evidenced
Stop-The-Line boundary. This packet records the latter boundary, not implementation
acceptance or a review handoff. The provider goal remains active at this first
blocked observation.

## Constraint and evidence

The task explicitly requires UUID/name/**qualified selectors**, exact contract
fixtures, and implementation from pinned v0.5.0 without redesigning the spec.
The task has no precondition resources; its Story has no additional selector
contract. Task/Story descriptions, scope, AC, and notes were read through the CLI.

The byte-pinned normative document at commit
28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c has SHA-256
562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a.
Section 2.3, lines 577–610, defines local exact live name, allowlisted-peer exact
live name, exact UUID, and not-found precedence, with ASCII-fold collision refusal.
It does not define a qualified selector's syntax, qualification target, or effect
on precedence/ambiguity. Section 14.1 names NAME and explicit action/destination
flags, but no qualified NAME grammar. Sections 5.1–5.2, 5.7 and 14.4 supply record,
event, projection and summary obligations, not that missing rule.

Full-document searches for qualified, selector, disambiguation, name@, @HOST,
host/name, NAME/HOST usages were checked against their contexts. Existing matches
for qualified paths and Git refs are unrelated. This search alone is not claimed
as a semantic completeness gate; the scoped normative paragraphs, CLI grammar,
task and Story contract were read directly. Exact excerpts are retained.

Implementing `name@host`, `host/name`, or a structured host filter would choose a
new product rule. In particular, source peer and current owner are different
possible qualifiers, and qualifier precedence may bypass the mandated local-name
priority or collision refusal. Tests choosing one would enshrine that assumption,
not prove compliance with an existing contract.

## Attempts and preserved work

No product-code changes were made. No grammar workaround, fallback, stub, manual
commit, rebase, branch switch, checkpoint, or integration was attempted. The
managed worktree remains clean at HEAD
7208cc7427e1ebe95106d34f98214e2bbea4ae1c. Its signature verifies (exit 0).

Preceding repository TASK-260830-wbpf1v is integrationCheckpointed=true, status
integrating, with accepted review revision 4. Reducer TASK-260830-1r9wrr is
integrationCheckpointed=true, status integrating, with accepted review revision 3
from RUN-260907-364977. Their acceptance artifacts are retained in the archive.
Those statuses were not regressed to manufacture the goal's literal to-review
predicate. This run does not own their checkpoint/integration.

The accepted reducer's unknown owner/lease behavior for empty/parked sessions is
preserved. Focused existing tests re-confirm it. The closed CLI SessionSummary
requires known owner/lease fields, so future summary work must obtain authoritative
supplemental facts or report a supported failure; inventing owner/lease values is
not permissible. This is recorded as an implementation constraint, not a second
human-only blocker and not a reopened reducer finding.

## Options and recommendation

1. Supply an existing approved qualified-selector contract, if one exists. This
   preserves scope and avoids specification work. Recommended first action.
2. Have the product/spec owner define qualification syntax, whether it identifies
   source peer or winning owner, and its interaction with the four-step resolution
   order and ambiguity rule. This preserves the full deliverable but needs an
   explicit authoritative decision before implementation.
3. Explicitly revise the task to defer qualified selectors and retain only the
   pinned NAME/UUID behavior. This reduces scope and is not an assumption this
   producer may silently make.

Exact input needed: the location of the approved qualified-selector contract, or
an explicit product-owner decision choosing option 2 or option 3. No routine
implementation or commit permission is requested. The user's task assignment's
“Stop-The-Line: No Forced Fits” requires stopping at an unresolved product model
decision and recording this packet before marking the task blocked.

## Acceptance and validation honesty

**0 of 5 AC behavior rows driven for this new leaf.** Denominator: selector
resolution, ambiguity, list summaries, status summaries, deterministic sorting.
All five remain unimplemented/unverified here. There is no new production call
site or committed test to cite; prerequisite tests do not count toward these rows.
No implementation checklist item is checked by this packet.

| Command run directly | Real exit | Evidence |
| --- | --- | --- |
| go test ./internal/specpin ./internal/specdoc -count=1 | 0 | spec-pin.log |
| go test ./internal/sessstate -run '^(TestReduceEmptyChainLocalIsIgnored\|TestProjectDerivesParkedBareDirectory\|TestProjectDerivesParkedRecordWithoutChain\|TestProjectIsIdempotentAcrossReads)$' -count=1 -v | 0 | accepted-projection-bound.log |
| git verify-commit HEAD | 0 | checkpoint-signature.log |
| git status --porcelain=v1 | 0, empty | worktree-status.txt |
| git rev-parse HEAD HEAD^{tree} | 0 | worktree-identity.txt |

No validation was piped through tee. No expected-red command is reported green.
Both test processes were observed to exit 0 before evidence attachment.
No process is left running. Full repository tests/coverage, lint, build, new-leaf
behavioral tests, mutation battery, and CR validation were not run: this is a
pre-implementation product-contract stop, not a candidate handoff. Earlier review
results are accepted only as prerequisite history and are not claimed as reruns.

| Mutant | Gate narrowing | Named failing test | Survival bound |
| --- | --- | --- | --- |
| None applied | No new gate implemented | None | No mutation-coverage claim |

This turn made progress by verifying the prerequisites and normative pin and
identifying/persisting the missing product decision. No previous goal turn is
available to classify. The provider blocked audit has only its first observation;
no provider goal update is justified yet.
