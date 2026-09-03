## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(5))

## Blocked By
- TASK-260830-1bipsa

## Blocks
- TASK-260830-2890sd
- TASK-260830-2xdt8t
- TASK-260830-2z3se0
- TASK-260830-2ciy0s
- TASK-260830-z1yxg9
- TASK-260830-3cuory

## Checklist
- [x] Production entry points implement the scoped deliverable: Prove machine clients never depend on message text, stderr, or exit code alone and historical envelopes remain readable
- [x] Relevant positive, negative, compatibility, and recovery tests pass with logs attached
- [x] README/doctor/capability evidence and specification traceability are updated without unsupported claims
- [x] Code written per task description and AC
- [x] Relevant tests written for new or changed behavior and passing
- [x] Gating, refusing, validating, authorizing, or attesting behavior covered by negative tests that fail when the gate admits what it must reject, with the production call site named
- [x] Lint clean
- [x] Relevant build/validation commands run after changes and build not broken
- [x] New outcome artifact attached on the board with a task-scoped name when the work produces notes, logs, screenshots, or other deliverables
- [x] Important findings, decisions, anomalies, or regressions recorded in logbook when relevant
- [x] Implementation matches AC
- [x] Solution fits project architecture
- [x] Tests green
- [x] Gate, refusal, validation, authorization, and attestation behavior attacked, not read — positive-path-only evidence is not accepted
- [x] If review does not accept the work — verdict evidence added and status routed by the explicit verdict branches

## Notes
LEAF 3 OF 3 — test-error-and-result-compatibility. FINAL LEAF: your publication is the Story's integration unit, so it is typed story_final and the Story lands on your work. Leaves 1 and 2 are accepted and checkpointed.

SCOPE
Prove machine clients never depend on message text, stderr, or exit code alone, and that historical envelopes remain readable.

WHAT YOU INHERIT
- internal/axerror (leaf 1): structured error versions, typed details, stable codes, retryability, causal redaction.
- internal/cliresult (leaf 2): CLI Result versions, JSON/JSONL serialization, human rendering boundaries, exit-code mapping.
- The ownership gate stands at 17/403 discharged clauses. section:14.2 is partial at 8/9; section:15.1 partial 5/7; section:15.3 partial 2/3; section:15.2 deliberately NOT claimed.

A DECISION LEAF 2's REVIEW EXPLICITLY LEFT TO YOU — THIS IS THE CENTRE OF YOUR TASK
`Decode` runs `decodeClosedDocument`, which refuses an unknown top-level member, BEFORE `verifyEnvelopeIdentity`, which refuses an unsupported major. Measured:
  future-major (2.0.0) + a new top-level member -> 'unknown top-level member \'new_in_v2\'', errors.Is(ErrUnsupportedMajor) = false
  future-major alone                            -> ErrUnsupportedMajor (correct)
Both paths refuse, so there is NO bypass and this is not a defect to fix reflexively. It matters here because §1.6's fail-closed unknown-member rule is scoped to 'a major version 1 object' — precisely the case where the major should already have been settled — so a future-major document carrying a new member reports a structural fact where a compatibility caller needs the version fact. §17.2 numbers 'rejects an unsupported major' first but states no explicit precedence, so the current order is defensible.
Decide it, with the pinned text quoted. If you change the order, prove both refusals still hold and that nothing that used to be refused is now admitted. If you keep it, say why and pin the observed behaviour so a later change cannot silently reorder it. Either way the outcome must be a pinned decision, not an undocumented accident.

THIS IS THE FINAL LEAF — TWO DELIVERY FACTS THAT HAVE COST THIS BOARD TIME
1. Your Change Request will be typed `story_final` only if completing this leaf would close the Story. Leaves 1 and 2 are already done, so that holds — but do not leave a sibling open, and do not create new siblings.
2. Leave the branch exactly one commit past the checkpoint with a clean worktree. A committed final leaf is a legal publication shape and the base refresh now supports it (fixed this session), but a second commit past the checkpoint is not.

STANDING CRITERIA — THESE ARE NOT BOILERPLATE, EACH ONE COST A REWORK THIS SESSION
- Validation that does not survive the package's own accessors is not validation. Leaf 1 shipped a shallow copy whose accessor handed out live interior state, so every declared bound could be violated after construction. Probe both inherited packages this way before trusting their guarantees in your compatibility tests.
- Before claiming a clause, name the ACTOR its RFC 2119 keyword governs and confirm this repository implements that actor. One clause was claimed against an obligation binding the provider plugin, which this repository does not build.
- A gate reports coverage as a measured ratio, not as prose. A pin over a documented claim must exercise the claim's SUBJECT, not only its parameter — a test that matched a number and then exercised the dimension it already believed in passed a false noun.
- 'Nothing is there' is not 'nothing this checker sees'. If a class cannot be checked, name it as a stated bound.
- Never invent a constraint the pinned document does not declare. Use internal/specdoc to check your own citations rather than trusting a line number from any report; line numbers in reports decay faster than findings do.
- Negative tests for every gate. Prove a bound by NARROWING, not only by deleting. A bound whose only evidence is deleting the gate is unpinned — minRedactableCause was exactly that.
- Confirm every mutant is PRESENT in the file before believing a green or a red, and make the presence check assert the MUTATED text.

DISCLOSE
Anything found and not fixed, including in the two inherited packages. A compatibility leaf that finds nothing in 7000 lines of new sibling code has probably not looked.

spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 and only admitted pair; final leaf owning a deferred precedence decision and the Story's integration unit."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260903-8527e7, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260903-8527e7)
Commit f80cd8d on task-board/story/STORY-260830-2jylym, exactly one commit past checkpoint f34d91d; worktree clean.

Production: cliresult.Read classifies one completed `ax --json` invocation from stdout + exit status; InvocationOutput has no stderr member (structural); axerror.CodesByExitStatus projects the registry status->codes and the refusals quote its count.

Claims proved: (1) message-independence by replay - every readable envelope re-read with the message replaced by another code name, by JSON, by 4096 chars, by non-ASCII, machine answers identical, vacuous rows fail; (2) stderr-independence structurally (reflection over the member set) plus a mutant that routes the one JSON document to stderr and leaves the client with ErrAbsentDocument; (3) exit-status insufficiency as a measured fan-in (47 codes over 17 statuses at 1.0.0, 109 at 1.3.0) narrowed to the retry decision - exit 12 carries staging_incomplete (may retry) and transaction_unknown (may not); (4) historical readability over two frozen SHA-256-pinned corpora, 9 CLI invocations and 7 protocol-bound envelopes covering all four bound error versions, plus a cross-binding sweep refusing a payload-declared version.

Absence, read failure, and foreign schema stay three refusals; a status outside the Section 15.2 table (1 included) is refused rather than classified.

Mutation: 12 mutants, each verified applied and compiled, 11 killed, 1 declared subsumed (the exit-0 guard in readFailure - the exit_code equality check refuses the same input one guard later because decodeExitStatus never admits exit_code 0), 0 unexplained survivors.

Traceability: 6 new acceptance cases, acceptance_cases 65->71, digest re-pinned, clauses_discharged unchanged at 17/403. Section 17.2 rebound from config EncodeCurrent to Read and deliberately left unevidenced at 0/1 with a gap saying an unknown error code is not an unknown Session Event - truthfulness fix, not a coverage gain. Section 14.2 stays partial at 8/9; Read checks the exit-status equality from the consuming side, which narrows 14.2#6 to the missing ax process without closing it.

Validation exit codes: go build 0, go vet 0, gofmt -l no output, go test ./... 0 (13 packages), go test ./... -cover 0 (cliresult 95.3%, axerror 99.5%), tracecheck 0.

Stated bounds: the two uncovered branches in client.go are unreachable by construction and TestEveryRegisteredCommandMajorBindsAnErrorVersion proves it over all 44 registered tags; no durable state is mutated, so there is no crash/idempotency surface; no ax binary exists, so 14.2#6 process half remains unowned.
Commit b21fede on task-board/story/STORY-260830-2jylym, exactly one commit past checkpoint f34d91d; worktree clean.

DEFERRED DECISION SETTLED — refusal precedence. A document of an unbound major carrying an unknown top-level member satisfied two refusals and the member rule won, so cliresult answered `unknown top-level member "new_in_v2"` with errors.Is(ErrUnsupportedMajor)=false; axerror had the same shape answering `unknown field`. DECIDED: the envelope identity is settled first, in BOTH readers. Section 1.6 scopes the member rule to "a major version 1 object", Section 17.1 to "within any negotiated major version", Section 17.2 lists "rejects an unsupported major" first, and Section 15.1 forbids parsing a different majors payload far enough to trust its fields. Both orders refuse, so nothing was admitted before or now; what changed is which fact the caller reads, and a caller deciding whether its ax is too old cannot decide that from a member name. cliresult.decodeClosedDocument split into parseDocument + verifyClosedMembers; axerror gained parseEnvelopeIdentity. Pinned by TestUnsupportedMajorIsSettledBeforeTheClosedMemberRule (6 rows cliresult, 5 axerror) and TestReorderingTheIdentityCheckAdmitsNothingItUsedToRefuse (13 previously-refused shapes per package); two harness mutants restore the old order and are killed.

FOUND WHILE DOING IT: the reorder made axerrors trailing-content guard in decodeClosedDocument unreachable; it was moved into parseEnvelopeIdentity rather than left as a branch no test can redden. Both inherited packages were probed for the accessor-aliasing class before their guarantees were used - Result.Body, Result.Extension and Error.Detail all deep-copy, and TestReadingAccessorsDoNotHandOutLiveInteriorState mutates each returned container two levels deep. Nothing further found unfixed in either package.

Production: cliresult.Read classifies one completed `ax --json` invocation from stdout + exit status; InvocationOutput has no stderr member (structural); axerror.CodesByExitStatus projects the registry status->codes and the refusals quote its count.

Claims proved: (1) message-independence by replay - every readable envelope re-read with the message replaced by another code name, by JSON, by 4096 chars, by non-ASCII, machine answers identical, vacuous rows fail; (2) stderr-independence structurally (reflection over the member set) plus a mutant routing the one JSON document to stderr, leaving the client with ErrAbsentDocument; (3) exit-status insufficiency as a measured fan-in (47 codes over 17 statuses at 1.0.0, 109 at 1.3.0) narrowed to the retry decision - exit 12 carries staging_incomplete (may retry) and transaction_unknown (may not); (4) historical readability over two frozen SHA-256-pinned corpora, 9 CLI invocations and 7 protocol-bound envelopes covering all four bound error versions, plus a cross-binding sweep refusing a payload-declared version.

Absence, read failure, and foreign schema stay three refusals; a status outside the Section 15.2 table (1 included) is refused rather than classified.

Mutation: 14 mutants, each verified applied and compiled, 13 killed, 1 declared subsumed (the exit-0 guard in readFailure - the exit_code equality check refuses the same input one guard later because decodeExitStatus never admits exit_code 0), 0 unexplained survivors.

Traceability: 8 new acceptance cases, acceptance_cases 65->73, digest re-pinned, clauses_discharged unchanged at 17/403. Section 17.2 rebound from config EncodeCurrent to Read and deliberately left unevidenced at 0/1 with a gap saying an unknown error code is not an unknown Session Event - truthfulness fix, not a coverage gain. Section 14.2 stays partial at 8/9; Read checks the exit-status equality from the consuming side, which narrows 14.2#6 to the missing ax process without closing it.

Validation exit codes: go build 0, go vet 0, gofmt -l no output, go test ./... 0 (13 packages), go test ./... -cover 0 (cliresult 95.5%, axerror 99.5%), tracecheck 0.

Stated bounds: the two uncovered branches in client.go are unreachable by construction and TestEveryRegisteredCommandMajorBindsAnErrorVersion proves it over all 44 registered tags; CodesByExitStatus cannot reach its ExitCodeFor failure path and the registry invariant that makes it unreachable is measured in TestCodesByExitStatusMeasuresTheFanIn; no durable state is mutated, so there is no crash/idempotency surface; no ax binary exists, so the 14.2#6 process half remains unowned.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260903-8527e7, pid=26474, exit=0)
spawn autonomous recovery: run RUN-260903-8527e7 queued successor RUN-260903-bcd7ed (attempt 1/3, model=claude-opus-5): Change Request construction for TASK-260830-33sfxc failed: change_request_base_authority_mismatch: the STORY-260830-2jylym committed candidate b21fede3ce5830020b83aadecf141eb1dfb8ceb1 is not exactly one direct single-parent commit past checkpoint d1d3eceb2c42e7a094206465d128272bc81008f5: <nil>
spawn run started: [implementer] developer (claude) (run=RUN-260903-bcd7ed)
spawn run RUN-260903-bcd7ed cancelled by operator; operator action required; reason: Deterministic refusal: change_request_base_authority_mismatch because checkpoint_oid never advanced past origin/main. Two prior checkpoints were no-ops on an empty CR delta, so the final leaf sits three commits past the recorded checkpoint. A successor re-running the same check against unchanged state cannot succeed.
agent completed: [implementer] developer (claude) (exit=143)
spawn run completed: claude (run=RUN-260903-bcd7ed, pid=56553, exit=143)
PUBLICATION-ONLY RUN. THE WORK IS COMPLETE AND COMMITTED. DO NOT CHANGE ANY FILE.

Your leaf is finished and committed at b21fede on the Story branch. Its parent f34d91d is now the recorded checkpoint, so the branch is exactly one direct single-parent commit past it — the shape story_final publication requires.

WHY THIS RUN EXISTS. Your previous run completed the work but its Change Request construction refused with change_request_base_authority_mismatch: the workspace record still named origin/main as the checkpoint, because checkpointing a zero-delta revision recorded no commit and never advanced the record. That was a tool defect, not a fault in your work. It is fixed and installed (skill-project-management PRs #114, #115, #116), and the record has been reconciled to f34d91d.

YOUR TASK
1. Verify the state rather than trusting this brief:
   git log --oneline -1        # expect b21fede
   git rev-parse HEAD^         # expect f34d91d
   git status --porcelain      # expect empty
2. Change NOTHING. Write no file, edit no file, add no test. If you believe something needs fixing, report it and stop.
3. Check your checklist items and hand off so the story_final Change Request is published from the existing committed state.

WHAT YOUR LEAF DELIVERED, FOR THE HANDOFF RECORD
Compatibility proof for the error and result envelopes: a consuming entry point Read(command, InvocationOutput{Stdout, ExitStatus}) — the first thing in the tree that reads a completed invocation rather than writing one; InvocationOutput with no stderr member at all, so a machine reading structurally cannot depend on a diagnostic stream, asserted through reflection so the pin reddens the moment a member appears; and the exit-status fan-in measured rather than argued — 47 codes over 17 failure statuses in 1.0.0, 14 of them carrying more than one code, 109 codes in 1.3.0.

DO NOT
- Do not amend, rebase, reset, or create a second commit. A second commit past the checkpoint breaks the leaf shape again.
- Do not re-run the mutation campaign; the work was reviewed on its merits.
- Do not touch .task-board directly.

spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 and only admitted pair; publication-only run emitting story_final from already-complete committed state after the checkpoint record was repaired."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260903-b931b2, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260903-b931b2)
PUBLICATION RUN RUN-260903-b931b2 — no repository file created, edited, or deleted; git status --porcelain empty at start and at handoff.

STATE VERIFIED INDEPENDENTLY, NOT TRUSTED FROM THE BRIEF. HEAD b21fede; HEAD^ f34d91d4b55c7521f81fb432af56c7d9e848f230; git rev-list --parents -n 1 HEAD shows one parent, so the commit is direct and single-parent; branch task-board/story/STORY-260830-2jylym.

CHECKPOINT RECONCILIATION CONFIRMED. task-board worktree status --json for WS-4fa8daeca025 reads checkpoint_oid=f34d91d4b55c7521f81fb432af56c7d9e848f230, branch_tip_oid=b21fede3ce5830020b83aadecf141eb1dfb8ceb1, checkpoint_reachable=true, dirty=false. checkpoint_oid now equals HEAD^, so the branch is exactly one direct single-parent commit past the checkpoint — the story_final shape the prior refusal (change_request_base_authority_mismatch, checkpoint stuck at origin/main d1d3ece) could not satisfy. The workspace lists CR-TASK-260830-1bipsa-1 and CR-TASK-260830-34elja-2, both checkpointed, and NO change request for TASK-260830-33sfxc, so handoff constructs a fresh CR against the current tree rather than republishing a stale candidate_tree_oid.

SIGNATURE. git verify-commit HEAD exit 0; Good git signature for oparin@me.com, ECDSA SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM; author Ivan Oparin <oparin@me.com>; %G? = G.

VALIDATION RE-RUN IN THIS SESSION, each as a standalone process reporting its real exit code: gofmt -l . exit 0 no output; go build ./... exit 0; go vet ./... exit 0; go test ./... -count=1 exit 0 across 13 packages; go run ./internal/traceability/cmd/tracecheck exit 0 reporting acceptance_cases=73 and clauses_discharged=17/403, which reproduces the previously attached traceability evidence exactly and confirms the committed tree is the tree that produced it.

STATED BOUNDS OF THIS RUN. The 14-mutant campaign was NOT re-executed (the brief forbade it and the work was reviewed on merits); that claim rests on the previously attached TASK-260830-33sfxc_mutation-report.json. Coverage percentages were NOT re-measured; go test ./... -cover was not re-run, only the suite itself. Nothing else above is inherited.

NEW ARTIFACTS: TASK-260830-33sfxc_publication-state.md, TASK-260830-33sfxc_publication-run-go-test.log, TASK-260830-33sfxc_publication-run-tracecheck.log.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260903-b931b2, pid=81180, exit=0)
REVIEWER CONTEXT — FINAL LEAF OF STORY-260830-2jylym. Your acceptance makes this the Story's integration unit.

WHICH COMMIT. HEAD b21fede, parent f34d91d (leaf 2, the recorded checkpoint). Verify reachability first.
NOTE: unlike every other Change Request on this Story, this one carries a REAL delta — 66 changed paths — because the checkpoint record was repaired. You do not need to reconstruct the diff by hand; earlier reviewers on this Story did, and that was a tooling defect (skill-project-management #113, fixed by #114/#115/#116).

WHAT THIS LEAF DELIVERED
A compatibility proof for the error and result envelopes, built on internal/axerror (leaf 1) and internal/cliresult (leaf 2):
- Read(command, InvocationOutput{Stdout, ExitStatus}) — a CONSUMING entry point. Nothing in the tree previously read a completed invocation; Emit writes one. The Story's whole claim is about what a machine client may depend on, and until now there was no client.
- InvocationOutput has exactly two members and NO stderr member, so a machine reading structurally cannot depend on a diagnostic stream. TestMachineReadingCannotSeeStderr asserts the member set through reflection rather than asserting that today's code ignores a field it has.
- The exit-status fan-in is MEASURED, not argued: 47 codes over 17 failure statuses in Structured Error 1.0.0, 14 of them carrying more than one code, the largest 6 codes at exit 6; 1.3.0 reaches 109 codes.
- A precedence decision leaf 2's review explicitly deferred here: Decode ran decodeClosedDocument (refusing an unknown top-level member) before verifyEnvelopeIdentity (refusing an unsupported major), so a future-major document carrying a new member reported a structural fact where a compatibility caller needs the version fact. Both paths refuse, so there was no bypass. Check what was decided, that the pinned text supports it, and that whichever order was chosen is now PINNED rather than incidental.

WHAT TO ATTACK, HARDEST FIRST
1. THE REFLECTION PIN. Asserting a member set is exactly right for this claim, but confirm it cannot be satisfied by a type that carries a diagnostic stream under another name, or by an embedded struct. A structural pin that only checks top-level field names is a presence check wearing better clothes.
2. THE MEASURED FAN-IN. Re-derive 47/17 and 109 independently. A ratio published as evidence must be reproducible, and this board has already shipped one inflated ratio (10/394 where the honest number was 9/394) because a clause was counted against an actor the repository does not implement.
3. LEAF 1 SHIPPED VALIDATION THAT DID NOT SURVIVE ITS OWN ACCESSORS — a shallow copy whose accessor handed out live interior state. Probe THIS leaf's new surface the same way, and probe the two inherited packages through the new consuming entry point, which is a path they had no caller for until now.
4. HISTORICAL ENVELOPE CORPORA. Two frozen corpora were added. Confirm they are frozen in a way that reddens if regenerated, and that 'historical envelopes remain readable' is proven by reading them through the production decoder rather than by a fixture that was written to match it.
5. Confirm nothing accepted in leaves 1 and 2 regressed.

STANDING REVIEW BAR
- Positive-path-only evidence is refused. Prove a bound by NARROWING, not only by deleting.
- Confirm every mutant is PRESENT in the file before believing a green or a red, and make the presence check assert the MUTATED text. A guard is not proven by a mutant that makes the guarded thing unreachable — that error was made and corrected in this session's own tooling work.
- A gate reports coverage as a measured ratio; a pin must exercise its claim's SUBJECT, not only its parameter; 'nothing is there' is not 'nothing this checker sees'.
- Before accepting any clause claim, name the ACTOR its RFC 2119 keyword governs and confirm this repository implements that actor.
- FINAL LEAF: accept with accept_cr if it holds — check your reviewer DoD items first, and do not supply commit_ack.

spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Rank-1 and only admitted pair; final leaf whose acceptance becomes the Story integration unit, with a real 66-path delta to review."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260903-29885b, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260903-29885b)
Reviewer verdict CR-TASK-260830-33sfxc-1 rev1 (RUN-260903-29885b): CHANGES REQUESTED. Evidence: TASK-260830-33sfxc_review-verdict.md, TASK-260830-33sfxc_review-mutation-rerun.json.

Reproduced independently: go build/vet/gofmt clean; go test ./... -cover exit 0 (cliresult 95.5%, axerror 99.5%); tracecheck exit 0 at acceptance_cases=73 clauses_discharged=17/403; the attached 14-mutant harness re-runs to 13 killed / 1 subsumed / 0 unexplained survivors; the two normative fixtures are byte-for-byte verbatim from SPEC.md at pinned commit 28bf96d. Six reviewer narrowing mutants (exit_code equality narrowed to status 5, unregistered-status gate admitting exit 1, wrong absence sentinel, exit-status-only classification bypass, message-derived Retryable, member rule restored before identity) were all killed. The message/stderr/fan-in/historical/precedence work holds up under attack.

BLOCKING 1 - cliresult/client.go:139 + axerror/decode.go:74: Read admits a Structured Error carrying duplicate top-level members, which SPEC.md:221 forbids. Through Read on the frozen workspace-conflict fixture at exit 5: a duplicate code inside the same exit class is admitted and Code() returns the second occurrence; a duplicate retryable flips Retryable() false to true, a forged retry claim; a duplicate details resolves to the union of both occurrences, neither one alone. The exit_code-equality check does not catch it when both codes share the exit class. cliresult.Decode refuses the identical shape via canonicaljson (decode.go:86, pinned by TestDuplicateAndMalformedDocumentsAreRefused); axerror.Decode has zero canonicaljson references. client.go:131-134, new in this CR, justifies not settling duplicates in the discriminator by claiming the selected contracts closed decoder validates them - true for one branch, false for the other. axerror.Decode also reads peer-supplied provider/bridge/rpc/adapter/terminal envelopes, so the forged-retryable shape is remotely reachable.

BLOCKING 2 - README fan-in table and LOGBOOK entry 1003: the Largest class column is wrong for two of four rows. 1.2.0 claims 12 codes at exit 6, measured 14 at exit 16. 1.3.0 claims 15 codes at exit 6, measured 17 at exit 6. No ties at the maximum. TestCodesByExitStatusMeasuresTheFanIn pins statuses/codes/ambiguous for all four versions but largest/largestFor only for 1.0.0 and 1.1.0, which is why these rows drifted.

NIT - client_test.go:145: TestReadDistinguishesAbsenceFromAReadFailure asserts errors.Is membership only; aliasing ErrAbsentDocument to ErrUnreadableDocument survives the whole suite. The realistic drift is killed, so nothing is broken today.

ORCHESTRATOR OBSERVATION - the CR base OID d1d3ece predates the two accepted sibling leaves ebc4e31 and f34d91d, so the delta spans 66 files / 13845 insertions across all three leaves instead of this leafs 31. candidate_tree_oid 49da1aa equals HEAD^{tree}, so the candidate is honest; the story checkpoint is stale. Review was scoped to b21fede.
CORRECTION to the orchestrator observation above: the CR base d1d3ece is NOT a stale checkpoint. Workspace WS-4fa8daeca025 has checkpoint_oid correctly reconciled to f34d91d; the CR was built from current_base_oid = origin/main = d1d3ece, which is the right base for a story_final CR whose acceptance is the Story integration unit. The 66-path delta is the whole Story vs trunk and is correct.

What IS notable: CR-TASK-260830-34elja-2 (base ebc4e31) and CR-TASK-260830-1bipsa-1 (base f34d91d) both carry repository_delta=empty, each based on its own leaf commit, so neither sibling reviewer had a CR diff to read. This story_final CR is the first time internal/axerror and internal/cliresult appear in a reviewable delta at all, which is exactly why BLOCKING 1 lands here: the brief asked to probe the inherited packages through the new consuming entry point, and driving axerror.Decode through cliresult.Read is what surfaced it.

Stated bound on this verdict: review depth was concentrated on b21fede (this leaf 31 files); leaves 1 and 2 were spot-checked through the new entry point, not re-reviewed in full. The verdict artifact TASK-260830-33sfxc_review-verdict.md has been updated to say this.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260903-29885b, pid=89337, exit=0)
REWORK BRIEF — TWO BLOCKING FINDINGS. The design of this leaf is accepted; both findings are about enforcement gaps in what it claims.

F1 (BLOCKING) — Read ADMITS A DUPLICATE-KEYED STRUCTURED ERROR, AND MACHINE ANSWERS CHANGE WITH IT
SPEC.md:221 puts duplicate keys outside the common logical data model. cliresult.Decode enforces it — decode.go:86 runs canonicaljson.Canonicalize, pinned by TestDuplicateAndMalformedDocumentsAreRefused. axerror.Decode never calls canonicaljson at all, so the FAILURE branch of the same production entry point does not enforce it.
Reproduced through cliresult.Read against the frozen fixture at its own exit status 5:
- duplicate 'code' in the same exit-5 class -> ADMITTED, Code() returns the SECOND occurrence
- duplicate 'retryable':true on a document declaring false -> ADMITTED, Retryable() = true. That is a forged retry claim through a duplicate member.
- duplicate 'details' -> DetailKeys() returns the UNION of both occurrences, neither one alone
- the same shape on a CLI Result -> correctly refused
Three reasons this is yours and not a sibling's to wave through:
1. The exit-status corroboration THIS leaf added does not catch it: both occurrences share the exit-5 class, so readFailure's exit_code equality passes.
2. client.go:131-134 states the delegation as its justification — 'the closed decoder for the selected contract then validates the whole object, including the duplicate members and trailing content this discriminator deliberately does not settle on its own.' True for Schema, false for axerror.Schema. That comment is new in this CR and is the reason documentSchema does not settle duplicates itself.
3. The claim under test IS this one. Two conforming readers, first-wins and last-wins, get a different code and a different retryable from the same bytes, and details resolves to neither occurrence because parseEnvelopeIdentity's map decode and decodeClosedDocument's struct decode resolve repeats differently within one document.
REACH: axerror.Decode is also the reader for peer-supplied provider, bridge, rpc, session-adapter and terminal-backend envelopes, so the forged-retryable shape is reachable from a REMOTE PEER, not only a local pipe. Treat it accordingly.
Required: route axerror.Decode through the same common-data-model gate the CLI Result reader uses, or an equivalent duplicate-member refusal; extend the Read refusal table and the axerror precedence rows with duplicate-member negatives for code, retryable, details and schema; correct the documentSchema comment so it describes what each branch actually validates; add the narrowing mutant to the harness.

F2 (BLOCKING) — THE FAN-IN TABLE IS WRONG IN TWO OF FOUR ROWS
Measured from axerror.CodesByExitStatus, no ties at the maximum:
  1.0.0  claimed 6 at exit 6   -> correct
  1.1.0  claimed 9 at exit 6   -> correct
  1.2.0  claimed 12 at exit 6  -> actually 14 at exit 16. Wrong count AND wrong status.
  1.3.0  claimed 15 at exit 6  -> actually 17 at exit 6. Wrong count.
The 1.3.0 error is repeated in LOGBOOK entry 1003.
The cause is the shape this board keeps closing: TestCodesByExitStatusMeasuresTheFanIn pins statuses/codes/ambiguous for all four versions and those are correct, but largest/largestFor are asserted only for 1.0.0 and 1.1.0 — so the other two rows drifted into prose nobody measures. Your own evidence standard, a measured ratio rather than prose, failed on the artifact the DoD asks for.
Required: correct both rows in README and LOGBOOK, and extend the pin to assert largest/largestFor for EVERY version so a published row cannot drift again. Deriving the prose from the measurement is better than asserting them separately.

F3 (NON-BLOCKING, CLOSE IT WHILE THERE)
TestReadDistinguishesAbsenceFromAReadFailure checks errors.Is per row and never the negative, so aliasing the two sentinels survives the whole suite. Nothing is broken today, but the doc comment claims the distinction is deliberate. One mutual-exclusion assertion closes it.

METHOD
- Confirm every mutant is PRESENT in the file before believing a green or a red. A guard is not proven by a mutant that makes the guarded thing unreachable — that error was made and corrected in this session's tooling work.
- Do not weaken any assertion accepted in leaves 1 and 2.
- Report the corrected fan-in figures as measured output, not as prose.

spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 and only admitted pair; closing a duplicate-member bypass reachable from a remote peer plus a published table wrong in half its rows."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260903-d32271, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260903-d32271)
REWORK RUN RUN-260903-d32271 COMPLETE. Commit 931d4de on task-board/story/STORY-260830-2jylym; parent f34d91d = the recorded checkpoint; git rev-list --parents -n 1 HEAD shows one parent; worktree clean. b21fede was AMENDED, not followed by a second commit, so the leaf shape is preserved. git verify-commit HEAD exit 0, Good signature for oparin@me.com, ECDSA SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM. New artifact: TASK-260830-33sfxc_rework-results.md.

F1 CLOSED (blocking). Reproduced first: through cliresult.Read on the frozen workspace-conflict fixture at exit 5, a duplicate code inside the exit class was admitted answering the SECOND occurrence, a duplicate retryable flipped Retryable() false->true (forged retry claim), a duplicate details resolved to the UNION of both occurrences (3 keys, neither occurrence), a duplicate exit_code was admitted, and a duplicate schema routed to readSuccess and reported ErrOutcomeDisagreement - a fact about the exit status for a document whose schema was never settled. Cause: cliresult.Decode canonicalized, axerror.Decode referenced canonicaljson nowhere, and client.go:131-134 justified the discriminator not settling duplicates by delegating to a decoder that did not check. FIX: axerror.requireCommonDataModel runs canonicaljson.Canonicalize at the top of Decode, ahead of the envelope identity, so all fifteen bound peer contracts (provider, bridge, RPC, session-adapter, terminal-backend, cli-result) are gated in one place; documentSchema runs the same gate before it reads schema, because a repeated schema selects the branch and neither closed decoder can settle that for it. documentSchema comment corrected to state what each branch actually validates and to record that the earlier sentence was true of one branch and false of the other.

TWO DECISIONS INSIDE THE FIX, BOTH PINNED. (1) The canonical bytes are DISCARDED, not adopted: RFC 8785 rewrites 1e1 to 10 and 9.0 to 9, and decodeExitStatus reads exit_code from raw bytes precisely so those forms are refused, so adopting the transform would be a widening dressed as validation. TestTheCanonicalGateDoesNotLaunderTheExitStatusToken proves BOTH halves per row - that Canonicalize really rewrites the literal, and that Decode still refuses it - and the adopt-the-canonical-bytes mutant is killed. (2) The gate runs BEFORE the identity: a document whose schema_version repeats has no version, and identity-first would answer ErrUnsupportedMajor or ErrVersionMismatch from one of two members. TestADuplicateIdentityMemberIsNotResolvedIntoAVersionFact asserts the data-model fact and explicitly NOT the version fact, on a major disagreement and a minor disagreement; the reorder mutant is killed.

parseEnvelopeIdentity trailing-content guard REMOVED rather than left as a branch no test can redden; the delete-the-gate mutant reddens 31 rows including both pre-existing trailing-content rows, so the coverage moved with it. STATED BOUND, measured not assumed: removing the gate does NOT redden the unescaped-control-character row, because validateMessage refuses that byte independently - recorded as subsumed in the test and the production comment, not counted as coverage the gate provides.

F2 CLOSED (blocking). Measured independently before any edit (TASK-260830-33sfxc_fan-in-measured.log): 1.0.0 6@6 correct; 1.1.0 9@6 correct; 1.2.0 published 12@6, measured 14 at exit 16 - wrong count AND wrong status; 1.3.0 published 15@6, measured 17@6. No ties at any maximum. Both README rows and the LOGBOOK 1003 sentence corrected. The larger half of the fix: TestREADMEFanInTableIsDerivedFromTheMeasuredProjection PARSES the README table and re-derives every cell from CodesByExitStatus, requires every registered version exactly once, refuses a published version the registry lacks, refuses a registry version the table omits, and refuses a tie at the maximum because the published row names one status. A sibling test does the same for the two figures entry 1003 states in prose. TestExitStatusAloneIdentifiesNoFailure now covers all four versions with largest/largestFor. Three mutants - restore the wrong 1.2.0 figure, restore the wrong 1.3.0 LOGBOOK figure, delete the 1.3.0 row - are all killed.

F3 CLOSED. The FIRST mutual-exclusion fix also survived its own mutant: it skipped the wanted sentinel by comparing VALUES, and under the alias the two values are equal, so it skipped exactly the comparison that would have caught it. Caught by running the mutant, not by reading the fix. The rows now name their sentinel as a string; TestTheObservationSentinelsAreDistinctValues closes it over all five reader sentinels requiring distinct identities and distinct text. The aliasing mutant is killed.

A SURVIVOR THE F1 FIX CREATED, CLOSED NOT DECLARED. The gate in documentSchema made second-document-on-stdout-admitted survive, because the gate refuses the same bytes as trailing data one guard later. It was not declared subsumed: the two guards report different facts, and which fact a caller is told is what this leaf spent its precedence work on. The two-documents row now asserts the Section 14.2 text and the mutant is killed again.

MUTATION: 23 mutants, each verified applied, verified to carry the MUTATED text on disk, and compiled before measurement. 22 killed, 1 declared subsumed (the exit-0 guard in readFailure), 0 unexplained survivors, suite restored green. NINE narrow rather than delete: the gate restricted to one of four versions, moved behind the identity, made to adopt its own canonical bytes; the discriminator copy disabled alone; the sentinel aliased; two published figures restored to their wrong values; one README row deleted.

TRACEABILITY: new acceptance case structured-error-common-data-model (production internal/axerror/decode.go#requireCommonDataModel, ten tests across both packages) attached to the section:15.1 and section:17.2 bindings; the fan-in case gained the two derivation tests. acceptance_cases 73->74. clauses_discharged UNCHANGED at 17/403 on purpose - the gate discharges no obligation the clause scanner measures, and claiming one would be the inflated-ratio defect this board has already shipped once. Ownership digest re-pinned; both pinned counts updated.

VALIDATION, each a standalone process with its real exit code: gofmt -l . exit 0 no output; go build ./... exit 0; go vet ./... exit 0; go test ./... -count=1 exit 0 across 13 packages; go test ./... -cover -count=1 exit 0 (cliresult 95.5%, axerror 99.5%); go run ./internal/traceability/cmd/tracecheck exit 0 at acceptance_cases=74 clauses_discharged=17/403; python3 mutants.py exit 0.

DISCLOSED, NOT FIXED. (1) cliresult.parseDocument trailing-content guard is UNREACHABLE - Decode canonicalizes first and parses the canonical bytes, which never carry trailing content. Leaf 2 code, predates this rework, mirror of the branch removed here from axerror; behaviour correct either way, so a dead branch not a hole. Left alone because changing an accepted sibling decoder was outside this brief. (2) cliresult.Decode parses the CANONICAL bytes rather than the caller bytes, unlike the choice made here for axerror. It matters only where a number token FORM is load-bearing and the CLI Result body carries no such member (exit_code lives in the Structured Error, read by axerror). No defect found and no number-form assertion weakened; recorded so the asymmetry is a stated decision. (3) 14.2#6 stays partial at 8/9 - no ax binary exists. (4) No durable state is mutated anywhere in this leaf, so there is no crash/idempotency surface.
FINAL HEAD CORRECTION AND A CR WARNING. The rework head is 209e0f2, not 931d4de: after the first amend a false citation was caught in this rework s OWN first draft and fixed, so the commit was amended once more. 209e0f2 parent f34d91d (the recorded checkpoint), git rev-list --parents -n 1 HEAD reports one parent, worktree clean, git verify-commit HEAD exit 0 Good signature oparin@me.com ECDSA SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM. HEAD^{tree} = c4967d95623cc20e4ffae2121065ba34cae1eb58.

CR WARNING FOR THE REVIEWER AND THE ORCHESTRATOR. CR-TASK-260830-33sfxc-1 rev1 still records candidate_tree_oid 49da1aa, which is the PRE-amend tree from RUN-260903-b931b2. task-board handoff exited 0 and constructed NO new revision - Change Request construction happens at spawn-run completion, not at handoff - so at the time this note was written the stored TASK-260830-33sfxc_change-request_rev1.patch describes the SUPERSEDED tree. Verify candidate_tree_oid against c4967d95623cc20e4ffae2121065ba34cae1eb58 before reviewing any stored patch. Review the commit 209e0f2 itself if a rev2 was not built for it.

THE FALSE CITATION, disclosed because it is the same class this board tracks. requireCommonDataModel s first doc comment cited Section 1.6 as fixing the wire format as "UTF-8 JSON restricted to the common logical data model". That sentence is NOWHERE in the pinned document - a paraphrase wearing quotation marks. Caught by checking the citation through internal/specdoc instead of trusting the wording. The two real fragments are SPEC.md:218 "map keys MUST be UTF-8 strings and MUST be unique" and SPEC.md:221 "floating-point numbers, NaN, Infinity, non-string map keys, and duplicate keys are forbidden", both confirmed in Section 1.6 through specdoc.Load/QuoteLines/SectionID. TestTheCommonDataModelCitationsAreVerbatimInThePinnedDocument locates both BY CONTENT rather than by line number, re-derives the section, and asserts the ABSENCE of the paraphrase so the correction cannot be undone by someone who remembers the wording.

A CLAUSE DELIBERATELY NOT CLAIMED. SPEC.md:218 is a MUST and section:1.6 is bound to internal/scalar/scalar.go#ErrInvalidScalar at unevidenced. This rework does NOT claim it: the gate is a reader refusing a document, the binding names a different production owner, and rebinding a section was outside this brief. clauses_discharged therefore stays at 17/403. Stated so the choice is visible rather than silent.

REVALIDATED AT 209e0f2, each command a standalone process with its real exit code: gofmt -l . exit 0 no output; go build ./... exit 0; go vet ./... exit 0; go test ./... -count=1 exit 0 across 13 packages; go test ./... -cover -count=1 exit 0 (cliresult 95.5%, axerror 99.5%); go run ./internal/traceability/cmd/tracecheck exit 0 at acceptance_cases=74 clauses_discharged=17/403; python3 mutants.py exit 0 at 23 mutants, 22 killed, 1 declared subsumed, 0 unexplained survivors, suite restored green. The ownership digest was re-pinned twice, once for the new acceptance case and once when the citation test joined it. All five evidence artifacts were re-uploaded from this head.
REVIEWABLE DIFFS ATTACHED, because the stored CR patch is stale. TASK-260830-33sfxc_rework-delta.patch is git diff --binary b21fede..209e0f2 - the rework delta only, 13 files / 1221 insertions / 38 deletions, which is what changed since the reviewed rev1 head and is the shortest honest read for this round. TASK-260830-33sfxc_leaf-full.patch is git diff --binary f34d91d..209e0f2 - the whole story_final leaf against its checkpoint. Use these rather than TASK-260830-33sfxc_change-request_rev1.patch until a CR revision is built whose candidate_tree_oid equals c4967d95623cc20e4ffae2121065ba34cae1eb58.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260903-d32271, pid=99287, exit=0)
REVIEW ROUND 2 — FINAL LEAF. Your acceptance makes this the Story's integration unit.

WHICH COMMIT. HEAD 209e0f2. Verify reachability first. The CR carries a real 68-path delta; its base is trunk, which is correct for a story_final — the Story's integration unit is everything the Story adds to trunk, not the last leaf's delta.

ROUND 1 BLOCKED ON TWO FINDINGS, BOTH NOW ADDRESSED
F1: axerror.Decode never called canonicaljson, so the failure branch of the same production entry point admitted duplicate members. A duplicate 'retryable':true forged a retry claim on a document declaring false; duplicate 'details' resolved to the union of both occurrences. Reachable from a remote peer, since axerror.Decode also reads provider, bridge, rpc, session-adapter and terminal-backend envelopes.
F2: the published fan-in table was wrong in two of four rows — 1.2.0 claimed 12 codes at exit 6 where the measurement is 14 at exit 16, and 1.3.0 claimed 15 where it is 17.

ORCHESTRATOR VERIFICATION ALREADY PERFORMED — EXTEND, DO NOT REPEAT
- F1: probed through the real API. Both `'retryable':true` and `'code':'native_store_conflict'` appended as duplicates are now refused, naming Section 1.6 and the duplicate member by name.
- F2: the rows are corrected AND the pin now PARSES the README table with Sscanf and compares it against the measured projection. Restoring the original wrong row reddens TestREADMEFanInTableIsDerivedFromTheMeasuredProjection, which prints both the published and the measured tuple. The prose is the test's input rather than a second assertion beside it.

WHAT TO ATTACK
1. THE DUPLICATE GATE'S REACH. axerror.Decode is fixed — check DecodeBound and every other exported path into the same envelopes. A gate installed on one of two entry points is exactly the shape F1 was. Name each exported decode path and say which are covered.
2. WHAT ELSE DOES canonicaljson NOW REJECT HERE? Routing a decoder through the common-data-model gate can refuse documents that used to be accepted for reasons unrelated to duplicates — NaN, non-string keys, floating-point. Confirm the frozen historical corpora still read, and that nothing that MUST be readable became unreadable. A compatibility leaf that breaks compatibility while proving it would be the session's best joke and its worst outcome.
3. THE README PIN. It parses one table. Confirm it fails closed when the table is absent, renamed, or reordered, rather than passing vacuously on a document it could not find.
4. F3, the absence/read-failure sentinels: aliasing them survived the suite in round 1. Confirm the mutual-exclusion assertion was added and kills that aliasing.
5. Confirm nothing accepted in leaves 1 and 2 regressed, and that the exit-status corroboration this leaf added still holds.

STANDING REVIEW BAR
- Positive-path-only evidence is refused. Prove a bound by NARROWING, not only by deleting.
- Confirm every mutant is PRESENT in the file before believing a green or a red. A guard is not proven by a mutant that makes the guarded thing unreachable.
- A gate reports coverage as a measured ratio; a pin must exercise its claim's SUBJECT, not only its parameter.
- FINAL LEAF: accept with accept_cr if it holds — check your reviewer DoD items first, and do not supply commit_ack.

spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Rank-1 and only admitted pair; round-2 review checking the new duplicate gate's reach and that a compatibility leaf did not break compatibility."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260903-00a6b8, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260903-00a6b8)
Review of CR-TASK-260830-33sfxc-2 (rev 2): CHANGES REQUESTED. Independently reproduced gofmt/build/vet clean, go test ./... -cover exit 0 (cliresult 95.5%, axerror 99.5%), tracecheck exit 0 at acceptance_cases=74 clauses_discharged=17/403 - every quoted figure reproduces. Ran 12 independent mutants in a scratch copy; 10 killed, including moving requireCommonDataModel out of axerror.Decode into DecodeBound (killed by 5 tests, so the rework claim holds). Two narrowing mutants on internal/cliresult/client.go survive the whole suite: (1) readFailure exit-status equality narrowed to `failure.CodeRegistered() && ...` - admits the frozen later-minor fixture (exit_code 10) at observed exit 5 as code=terminal_backend_stale_generation ExitStatus()=5, and this is the class where the guard is sole enforcement because axerror.decodeBody only cross-checks exit_code against the registry for registered codes; (2) readSuccess success-at-a-failure-status guard narrowed past exit 130 - admits a CLI Result success object at exit 130 (Section 15.2 interrupted-before-a-clean-response) as Succeeded()=true. Both disagreement rows are single-sample in TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus. Production code is correct in both cases; the proof is not, and this leaf published exactly that standard. Two non-blocking observations (details-bearing narrowing near-equivalent; CodesByExitStatus empty-group mutant genuinely equivalent) are recorded. Full evidence: TASK-260830-33sfxc_review-verdict-rev2.md.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260903-00a6b8, pid=37925, exit=0)
REWORK BRIEF — TWO BLOCKING FINDINGS, BOTH THE SAME SHAPE, BOTH MEASURED AGAINST YOUR OWN PUBLISHED STANDARD

The implementation is correct and is not being asked to change. The evidence is incomplete on the two gates this leaf exists to prove, and the shortfall is exactly what your own README states: 'A gate whose only evidence is that deleting it reddens something has been shown to exist, not to hold.' Both findings are surviving NARROWING mutants — the gate is present, reachable and green, and a narrowed version of it is equally green.

F1 (BLOCKING) — THE §14.2 EXIT-STATUS EQUALITY IS PROVEN ONLY FOR REGISTERED CODES
Surviving mutant in readFailure:
  - if failure.ExitCode() != output.ExitStatus {
  + if failure.CodeRegistered() && failure.ExitCode() != output.ExitStatus {
`go test ./internal/cliresult ./internal/axerror -count=1` passes. No test in either package drives an unregistered code at a mismatched status.
Why that row is the one that matters: axerror.decodeBody cross-checks exit_code against the registry ONLY when ExitCodeFor resolves. For a code added by a later compatible minor it takes the ErrUnregisteredCode branch and sets registered=false with no exit check at all. So for a registered code the equality is doubly bound, and for an unregistered code readFailure is the SOLE thing binding the document's exit_code to the status the process actually exited with. The uncovered row is precisely where the guard is the only enforcement.
Reproduced against the mutant with your own frozen fixture: a machine client is handed exit class 5 (workspace conflict, 'no silent overwrite') for a document declaring exit class 10 (ownership/lease/fencing). On unmutated HEAD the same call is correctly refused with ErrOutcomeDisagreement.
Note you already had this insight once: you added error-1.0.0-later-minor-code-forged-retry.json because an unregistered code must not be a way past the RETRY rule. The same reasoning applies to the EXIT-CLASS rule and was not carried over.
Wanted: a row driving Read with an unregistered later-minor code whose exit_code disagrees with the observed status, requiring ErrOutcomeDisagreement, plus the narrowing mutant in the harness so the bound is measured rather than assumed.

F2 (BLOCKING) — THE SUCCESS-AT-A-FAILURE-STATUS GUARD IS PROVEN AT ONE STATUS
Surviving mutant in readSuccess:
  - if output.ExitStatus != SuccessExitStatus {
  + if output.ExitStatus != SuccessExitStatus && output.ExitStatus != 130 {
Passes the same suites. Reproduced: Read(CommandList, {Stdout: <emitted list success>, ExitStatus: 130}) returns Succeeded()=true at status 130.
Exit 130 is the §15.2 row 'Interrupted by operator signal before a clean response; inspect authority before retry'. It is the ONE status where a success document plausibly does reach stdout before the process dies, so a narrowing at exactly this status is a realistic bypass rather than an artificial one.
Wanted: drive the guard at more than one failure status — 130 specifically, and at least one ordinary failure status — and add the narrowing mutant.

TWO SURVIVORS THE REVIEWER CHECKED AND DISMISSED — DO NOT 'FIX' THEM
1. Narrowing requireCommonDataModel to documents whose bytes contain 'details' survives. Every conforming Structured Error requires details, so it is near-equivalent on realistic input; it records only that the gate's evidence rests on documents carrying that member. No action.
2. CodesByExitStatus pre-seeding an empty group for every §15.2 failure status survives and is genuinely EQUIVALENT: all four versions register at least one code at all 17 failure statuses, so 'absent rather than present and empty' is unobservable against today's registry. The documented rule is not wrong, it is vacuous against current data. No action.
Both are recorded because a survivor that is dismissed with a reason is evidence; a survivor that is quietly fixed hides why it survived.

METHOD
- Confirm every mutant is PRESENT in the file before believing a green or a red. A guard is not proven by a mutant that makes the guarded thing unreachable.
- Prove each bound by NARROWING, not by deleting. Deleting shows the gate exists; narrowing shows what it covers.
- Do not weaken anything accepted in leaves 1 and 2 or in revision 1 of this leaf.

spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 and only admitted pair; closing two surviving narrowing mutants on the gates this leaf exists to prove."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260903-dd2e44, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260903-dd2e44)
ROUND-3 REWORK COMPLETE — commit 1d68325 (amend of 209e0f2; one commit past checkpoint f34d91d; signed, git verify-commit clean; tree bec5631).

BOTH BLOCKING FINDINGS CLOSED, NO PRODUCTION CODE CHANGED. The implementation was correct in both; the evidence was not. Both mutants were reproduced as SURVIVED on the unchanged 209e0f2 tree BEFORE any test was written (repro-r3.py, verified present on disk and compiled), and are KILLED after.

F1 — the Section 14.2 exit-status equality was proven only for registered codes. axerror.decodeBody cross-checks exit_code against the registry only when ExitCodeFor resolves; a later-minor code takes the unregistered branch with no exit check at all, so readFailure is the SOLE enforcement on exactly that class. TestTheExitStatusEqualityBindsACodeTheRegistryDoesNotCarry drives the frozen error-1.0.0-later-minor-code.json through cliresult.Read at all sixteen other registered failure statuses, requires ErrOutcomeDisagreement and the refusal naming the equality, and first asserts CodeRegistered()==false so the row cannot stop covering its class in silence.

F2 — the success-at-a-failure-status guard was proven at one status. Both ErrOutcomeDisagreement rows now sweep every registered Section 15.2 failure status, enumerated through axerror.IsFailureExitStatus by one helper that asserts the count is 17 and that exit 130 resolves to the signal-interruption row.

THE ENUMERATOR IS MEASURED TOO. Narrowing its range to status<=20 drops exit 130 out of every loop that draws on it; the asserted count turns that into a red rather than a quietly smaller sweep. That is mutant 5 and it is killed.

MUTATION: harness grown 23 -> 28. 27 killed, 1 declared subsumed (unchanged), 0 unexplained survivors, 14 narrowing rather than deleting, suite restored green. The five new mutants are all narrowing: F1s exact mutant, the same guard narrowed by status instead of by registration, F2s exact mutant, F2 at an ordinary status, and the shared enumerator.

THE TWO DISMISSED SURVIVORS WERE LEFT ALONE as instructed.

TRACEABILITY: new test registered in the cli-result-machine-reading acceptance case, ownership digest re-pinned to f2bc887e. That registration was PROBED rather than assumed — renaming the declaration makes tracecheck exit 1 with declaration ... is absent from internal/cliresult/client_test.go, ahead of the digest check. acceptance_cases stays 74 and clauses_discharged stays 17/403: a test added to an existing case discharges no new clause.

VALIDATION, each run standalone with its real exit status: gofmt -l ./internal 0 (no output), go build 0, go vet 0, go test ./... -count=1 0 across 13 packages, go test ./... -cover -count=1 0 (cliresult 95.5%, axerror 99.5%, both unchanged — the new rows drive statements already executed; what moved is the class each guard is proven over), tracecheck 0, harness 0. gofmt -l ./internal ./cmd was NOT run: this module has no root cmd/ directory and that form exits 2 on the missing path.

REVIEWABLE DIFFS ATTACHED, because the stored CR patch predates this amend. TASK-260830-33sfxc_rework-r3-delta.patch is git diff --binary 209e0f2..1d68325 — 5 files, 188 insertions, 7 deletions, the round-3 delta only. TASK-260830-33sfxc_leaf-full-r3.patch is git diff --binary f34d91d..1d68325, the whole story_final leaf. Use these until a CR revision exists whose candidate_tree_oid equals bec56317e09b2ed9127fb2cc60900d47081549c7.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260903-dd2e44, pid=44473, exit=0)
REVIEW ROUND 3 — FINAL LEAF, NARROW SCOPE. Your acceptance makes this the Story's integration unit.

WHICH COMMIT. HEAD 1d68325, three commits past origin/main, CR rev 3 with a 68-path delta. Verify reachability first.

ROUNDS 1 AND 2 ARE CLOSED. Round 1: a duplicate-member bypass in axerror.Decode that forged a retry claim and was reachable from a remote peer, plus a published fan-in table wrong in two of four rows. Round 2: two surviving NARROWING mutants on the gates this leaf exists to prove. Do not re-review any of it except to confirm it did not regress.

ORCHESTRATOR VERIFICATION ALREADY PERFORMED — EXTEND, DO NOT REPEAT
Both round-2 mutants are now killed, each confirmed present in the file before measuring and the tree restored clean after:
- narrowing readFailure's exit equality to `failure.CodeRegistered() && ...` reddens TestTheExitStatusEqualityBindsACodeTheRegistryDoesNotCarry;
- exempting exit 130 from readSuccess's guard reddens TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus.

WHAT TO ATTACK
1. THE SHAPE THAT PRODUCED BOTH ROUND-2 FINDINGS. Each was a guard proven at ONE point of its domain, and each hole sat exactly where the guard was the SOLE enforcement — for a registered code the exit equality is doubly bound, for an unregistered one readFailure is alone. Sweep for other guards in this leaf that are singly enforced, and check each is exercised at more than one point of its domain. That is a more productive search than re-attacking what is already pinned.
2. THE NEW TESTS THEMSELVES. A test written to kill a named mutant can be written to kill only that mutant. Confirm each new case exercises the production entry point and would still redden under a neighbouring narrowing, not just the exact one quoted in the brief.
3. Confirm the two survivors the round-2 reviewer dismissed with reasons — the near-equivalent 'details' narrowing of requireCommonDataModel, and the genuinely equivalent CodesByExitStatus empty-group pre-seed — were NOT 'fixed'. A survivor dismissed with a reason is evidence; one quietly fixed hides why it survived.
4. Confirm the frozen historical corpora still read through the production decoder now that axerror.Decode routes through the common-data-model gate, and that nothing which must remain readable became unreadable.
5. Confirm leaves 1 and 2 did not regress.

STANDING REVIEW BAR
- Positive-path-only evidence is refused. Prove a bound by NARROWING, not by deleting: deleting shows a gate exists, narrowing shows what it covers.
- Confirm every mutant is PRESENT before believing a green or a red. A guard is not proven by a mutant that makes the guarded thing unreachable.
- FINAL LEAF: accept with accept_cr if it holds — check your reviewer DoD items first, and do not supply commit_ack. If it does not hold, say so plainly; four rounds is not a reason to accept.

spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Rank-1 and only admitted pair; round-3 review sweeping for singly-enforced guards rather than re-attacking what is pinned."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260903-45ad1a, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260903-45ad1a)
Reviewer CR rev3: CHANGES REQUESTED -> to-dev. Round-2 findings verified closed (both narrowing mutants now KILLED; producer harness reproduced independently at 28/27/1/0-unexplained). One blocking finding: README.md:1643 publishes 73 executable acceptance cases while ownership.v0.5.0.json holds 74 unique ids and tracecheck prints acceptance_cases=74; the leafs own LOGBOOK 1157 records the 73->74 transition, so README and LOGBOOK contradict each other in one commit. Nothing measures the figure - setting it to 999 leaves go test ./... and tracecheck both at exit 0. Third occurrence of the unmeasured-published-figure class in this artifact (LOGBOOK 1015, 1157). Fix: correct to 74 and pin the sentence the way exit_table_pin_test.go:148 already pins the fan-in table, plus a restore-73 harness mutant. Two non-blocking observations: exitStatusIsNotEnough measured count is unasserted (mutants replacing len(fanIn[status]) with 1, or the whole clause with the word many, both survive); Retryable() status-independence is proven at exit 12 only (forced-true-at-15 survives, but the realistic whole-cloth status-derived mutant is killed). 40 independent mutants run in /tmp scratch: 34 killed, 2 equivalent (Succeeded from exitStatus, ExitStatus from failure.ExitCode - both survive because the disagreement guards make the signals provably equal), 3 observations, 1 no-op. No gate bypass found; stderr exclusion, the three-way refusal split, the frozen digest-pinned corpora and the common-data-model gate all hold under attack. Evidence: TASK-260830-33sfxc_review-verdict-rev3.md.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260903-45ad1a, pid=67106, exit=0)
REWORK BRIEF — ONE BLOCKING FINDING. Mechanical to fix; the point is why it happened.

WHAT IS CLOSED. Both round-2 findings are verified closed independently, not taken from your report. The compatibility claim survived an attack of 40 mutants written without reading your harness first. The implementation is not in question.

F1 (BLOCKING) — README.md:1643 PUBLISHES 73 ACCEPTANCE CASES; THE GATE MEASURES 74
Every other figure in that sentence reproduces exactly against tracecheck — contracts=60, normative_sections=36, bindings=49, unowned=2, fixtures=30, compatibility_contracts=55. One does not: tracecheck reports acceptance_cases=74, ownership.v0.5.0.json holds 74 entries with 74 distinct ids, and the sentence says 73.
Not ambiguity about what is counted: the registry array the sentence links to has 74 unique ids.
How it happened, from your own record: at the Story checkpoint the registry held 65 and the README said 65. Your first pass took it to 73 and updated the sentence. The round-2 rework added more cases and did not.

WHY THIS IS BLOCKING RATHER THAN A TYPO
This is the THIRD occurrence of the identical class inside this one artifact, and you documented the first two yourself: LOGBOOK 1015 records the 'nineteen §15.2 rows' figure nobody measured, and LOGBOOK 1157 records two of four published fan-in rows being wrong, with the root cause named in your own words as 'prose nobody measured'.
Your response to the second was TestREADMEFanInTableIsDerivedFromTheMeasuredProjection, which parses the README and re-derives every cell. That test sits 200 lines from a figure that is restated instead of derived and is off by one. The tool for this class exists, in this file, written by you, and was not pointed at the neighbouring sentence.

REQUIRED
1. Correct the figure to 74.
2. Pin the SENTENCE, not the number. parsePublishedFanIn at internal/axerror/exit_table_pin_test.go:148 is the working pattern: it locates its target by content, refuses a target it cannot parse, and refuses a duplicated header. Follow it.
3. Pin all six figures in that sentence, not only the one that drifted. The other five are correct today and unmeasured for exactly the same reason; pinning the sentence whole costs no more than pinning one number of it.
4. Add the mutant that restores 73 to the harness, so the bound is measured rather than asserted.

THE GENERAL RULE, WORTH A LOGBOOK LINE
A figure published beside a measurement must be DERIVED from it, not restated next to it. Three instances in one artifact is not carelessness; it is the natural failure mode of prose that sits near data, and the only fix that holds is making the prose an input to the test rather than a second copy of the answer.

METHOD
- Confirm the mutant is PRESENT in the file before believing a green or a red.
- Do not weaken anything accepted in rounds 1 and 2.
- The reviewer recorded three non-blocking observations and two equivalent survivors; read them and say which you acted on and which you are leaving, with the reason. A survivor dismissed with a reason is evidence; one quietly fixed hides why it survived.

spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 and only admitted pair; pinning a published sentence to its measurement, the third instance of that class in one artifact."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260903-e342f8, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260903-e342f8)
ROUND-4 REWORK COMPLETE — commit b43a746 (amend of 1d68325; one commit past checkpoint f34d91d; signed, git verify-commit clean; tree 2e893d0d9b476991e481098bafff9b685374aa18).

THE ONE BLOCKING FINDING IS CLOSED, NO PRODUCTION CODE CHANGED. F1 and both acted-on observations were evidence defects.

F1 — README published 73 executable acceptance cases against a registry of 74 unique ids and a tracecheck that prints 74. Corrected to 74, and the SENTENCE is pinned rather than the number. TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport locates the paragraph by its heading, refuses a heading found twice or not at all, and re-derives ALL SEVEN figures from traceability.VerifyRepository — contracts 60, normative sections 36, acceptance cases 74, section bindings 49, unowned 2, fixtures 30, compatibility contracts 55. The other six were correct today and unmeasured for exactly the same reason. Each figure is located by the noun phrase naming what it counts, so re-ordering the clause cannot re-point a pin at a different number.

THE PIN POLICES ITS OWN COMPLETENESS. Every digit run in the paragraph that no pinned phrase consumed is reported as unmeasured, with only dotted version and path tokens exempt under a narrow rule. That makes deleting a row from publishedOwnershipFigures a red rather than a quietly smaller comparison. Nine mutants cover it: the restore-73, one per remaining figure, an added unmeasured figure, and the list narrowed by one row. The restore-73 was reproduced by hand first, confirmed present at README line 1643, and reddened with the exact measured message.

OBSERVATION A ACTED ON. exitStatusIsNotEnough called its own count measured and nothing read the number. TestTheRefusalStatesTheMeasuredFanInRatherThanAdvice now drives every registered command at every registered Section 15.2 failure status through Read and re-derives each count from CodesByExitStatus, asserting the number of refusals read. Three mutants killed: the constant, the many rewrite, and — the point — the narrowing that keeps the count correct at exit 6 ONLY, which is exactly what the proposed single assertion would have survived.

OBSERVATION B ACTED ON. Retry independence was proven at exit 12 alone and the one exit-15 envelope any test drove already declared retryable true. TestTheRetryDecisionFollowsTheDocumentAtEveryFailureStatus sweeps both polarities across the whole domain: false at all 17 statuses, true at 14, because the pinned document forbids the claim for every code of the other three. That 14 is asserted, and was set FROM the measurement after a provisional 17 reddened. Four mutants killed including the narrowing where the document decides at exit 12 only.

OBSERVATION C AND THE ROUND-2 DISMISSALS LEFT ALONE, deliberately: Succeeded from exitStatus, ExitStatus from failure.ExitCode, the near-equivalent details narrowing of requireCommonDataModel, and the CodesByExitStatus empty-group pre-seed are all unchanged.

I ALSO FOUND THE SAME CLASS IN MY OWN EVIDENCE PROSE. README published a 28-mutant harness killing 27 and fourteen narrowing; both stale and both unmeasured. Corrected AND bounded: the harness mutates the tree, lives under .temp/, is attached here, and is NOT run by go test ./..., so README now states those figures as a bound instead of letting them read as gated. The narrows-rather-than-deletes class is now DEFINED in the harness and counted from that definition, failing closed if the set names a mutant that does not exist; applied to the 28-mutant harness it selects exactly the fourteen that revision published, and 27 here.

MUTATION: 28 -> 44 mutants. 43 killed, 1 declared subsumed (unchanged), 0 unexplained survivors, 27 narrowing, suite restored green. Each mutant verified applied, compiled, and carrying the MUTATED text on disk before measurement.

ANOMALY: my first hand-probe of the many mutant returned non-zero and I read it as a kill. It was a BUILD failure — dropping the %d argument left fanIn declared and not used. The harness NOT-COMPILED arm caught it and the mutant was rewritten to compile. A non-zero exit is not by itself a kill. LOGBOOK 1338.

TRACEABILITY: three tests registered, each registration PROBED rather than assumed — renaming the declaration makes tracecheck exit 1 naming the absent declaration, ahead of the digest check. The digest gate fired on the registry change as designed and was re-pinned to d9a860aa. acceptance_cases stays 74 (tests added to existing cases, no new id) and clauses_discharged stays 17/403.

VALIDATION, each standalone with its real exit status: gofmt -l ./internal 0 (no output), go build ./... 0, go vet ./... 0, go test ./... -count=1 0 across 13 packages, go test ./... -cover -count=1 0 (cliresult 95.5%, axerror 99.5%, traceability 86.6%), tracecheck 0, harness 0. gofmt -l ./internal ./cmd was NOT run: this module has no root cmd/ directory and that form exits 2 on the missing path. cliresult and axerror coverage is unchanged — the new rows drive statements already executed; what moved is the class each guard is proven over.

REVIEWABLE DIFFS ATTACHED, because the stored CR patch predates this amend. TASK-260830-33sfxc_rework-r4-delta.patch is git diff --binary 1d68325..b43a746 — 6 files, 420 insertions, 4 deletions, the round-4 delta only. TASK-260830-33sfxc_leaf-full-r4.patch is git diff --binary f34d91d..b43a746, the whole story_final leaf. Use these until a CR revision exists whose candidate_tree_oid equals 2e893d0d9b476991e481098bafff9b685374aa18.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260903-e342f8, pid=86276, exit=0)
REVIEW ROUND 4 — FINAL LEAF, VERY NARROW. Your acceptance makes this the Story's integration unit.

WHICH COMMIT. HEAD is CR rev 4, 69-path delta, three commits past origin/main. Verify reachability first.

ROUNDS 1-3 ARE CLOSED and were each verified closed independently by the reviewer that followed. Round 1: a duplicate-member bypass forging a retry claim, reachable from a remote peer, plus a fan-in table wrong in two of four rows. Round 2: two surviving narrowing mutants on the gates this leaf exists to prove. Round 3: a published figure off by one — 73 against a measured 74 — which was the THIRD instance of that class in this artifact.

ORCHESTRATOR VERIFICATION ALREADY PERFORMED — EXTEND, DO NOT REPEAT
The figure is 74, and a new internal/traceability/readme_figures_pin_test.go pins SEVEN figures from that paragraph rather than the one that drifted. Restoring 73 reddens TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport with both numbers named. Mutant confirmed present before measuring; tree restored clean.
Method note for your own work: my first attempt at that mutant reported 'ok ... [no tests to run]' because the -run pattern matched nothing. A green from a filter that matched nothing is not a green. Confirm the test you think you ran actually ran.

WHAT TO ATTACK
1. THE NEW PIN'S OWN FAILURE MODES. It locates a paragraph in README by content. Confirm it fails CLOSED when the paragraph is absent, renamed, reordered, or duplicated — the fan-in pin's parsePublishedFanIn already refuses a duplicated header, so the pattern exists. A pin that silently passes on a document it could not find is the same defect it was built to prevent.
2. DOES IT PIN SEVEN OR MERELY PARSE SEVEN? Each figure must be compared against a measured value, not merely extracted. A parser that reads seven numbers and asserts one is worse than a parser that reads one, because it looks thorough.
3. THE THREE NON-BLOCKING OBSERVATIONS AND TWO EQUIVALENT SURVIVORS from round 3. The producer was asked to say which it acted on and which it left, with reasons. Confirm it did, and that no survivor the previous reviewer dismissed WITH A REASON was quietly 'fixed' — that hides why it survived.
4. Confirm nothing from rounds 1-3 regressed, in particular the duplicate-member gate and the two narrowing bounds.

STANDING REVIEW BAR
- Positive-path-only evidence is refused. Prove a bound by NARROWING, not by deleting.
- Confirm every mutant is PRESENT before believing a green or a red, and confirm the test you filtered for actually ran.
- FINAL LEAF: accept with accept_cr if it holds — check your reviewer DoD items first, and do not supply commit_ack. Five rounds is not a reason to accept, and it is not a reason to keep looking either; judge this revision on what it is.

spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Rank-1 and only admitted pair; narrow round-4 review of a new README pin that must fail closed and must compare rather than merely parse."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260903-b96555, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260903-b96555)
Review of CR revision 4 (candidate tree 2e893d0d, re-verified against HEAD^{tree}): CHANGES REQUESTED, one blocking finding. F1 - the command-tag agreement in cliresult.readSuccess is covered by a single ordered pair (invoked doctor, document says list). Narrowing it to spare invoked CommandList survives; narrowing it to admit any document that claims command=takeover survives the entire repository suite (go test ./... -count=1 exit 0, 13 packages), so Read(CommandList, takeover-document) would hand a caller a takeover Result body for an invocation that never ran it. Probe confirms this guard is the sole owner of that refusal - neither Decode nor the identity check catches the pairing. The two sibling subtests in the same test function sweep their whole domain and assert the sweep count; this one takes one sample, and no bound is declared for it anywhere in the rework results. Control mutant sparing the covered pair is KILLED, so the gap is measured, not assumed. Fix is test-only: drive the agreement over the registered command vocabulary in both directions with the refusal count asserted against the enumeration. Independently re-run here: go build, go vet, gofmt, go test ./... -count=1 (exit 0), tracecheck (exit 0, acceptance_cases=74, clauses_discharged=17/403). Eleven reviewer-written mutants; 8 killed, 1 equivalent (Reading.exitStatus from failure.ExitCode - subsumed by the readFailure equality, reproduced not trusted), 2 command-agreement survivors above. Secondary stated bound, not blocking: requireCommonDataModel narrowed to skip documents over 4096 bytes survives, because every duplicate-member test uses a small hand-built document - a byte-length predicate is not a Section 1.6 contract dimension, but payload size is attacker-controlled on the peer-supplied envelopes Decode reads. Verified holding under attack: cliresult.Decode own data-model gate, discriminator gate, decodeExitStatus zero-status refusal, CodeRegistered forgery, major-2 widening, historical corpus completeness in both directions with SHA-256 pins and the message-blinded replay. Working tree restored byte-identical to the candidate. Evidence: TASK-260830-33sfxc_review-verdict-rev4.md.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260903-b96555, pid=30501, exit=0)
REWORK BRIEF — ONE BLOCKING FINDING. Everything else in revision 4 held under attack, including both round-3 findings. The implementation is not in question; the evidence for one guard is.

F1 (BLOCKING) — THE COMMAND-TAG AGREEMENT IS PROVEN FOR ONE ORDERED PAIR
internal/cliresult/client.go, readSuccess: `if result.Command() != command`.
This guard is what stops a caller being handed a Result whose body belongs to a command the invocation never ran. It is covered by exactly one row — the pair (invoked doctor, document says list). Three narrowing mutants, each verified applied and compiled:
  - sparing CommandDoctor            -> KILLED (control: the one row is real)
  - sparing CommandList              -> SURVIVED
  - admitting any document that CLAIMS to be a takeover result -> SURVIVED THE ENTIRE REPOSITORY SUITE, all 13 packages ok
The third is the finding. With it in place, Read(CommandList, <document carrying 'command':'takeover'>) is admitted and the caller receives a Reading whose Result() is a takeover body — the one command in this contract carrying adoption and authority semantics, Result.VerifyTakeoverAdoption — for an invocation that never ran it. Nothing in the repository reddens.
The reviewer confirmed this guard is the SOLE owner of that refusal: both directions of the probe are refused by it, and neither cliresult.Decode nor the envelope decoders catch the class. That is the same shape as round 2's finding — the hole sits exactly where the guard is the only enforcement.
Wanted: cover the guard across the command set rather than at one pair. Commands() and ImplementedCommands() are already exported for exactly this. Drive the disagreement in both directions for every implemented command, and add the takeover-claiming narrowing to the harness so the bound is measured rather than assumed.

TWO NON-BLOCKING ITEMS — RECORD, DO NOT CHASE
1. STATED BOUND. Narrowing requireCommonDataModel to skip documents over 4096 bytes SURVIVES, because every duplicate-member test builds a small document by hand and no frozen historical envelope exercises the gate. A byte-length predicate is not a dimension of the §1.6 contract, so the reviewer did not block on it — an arbitrary predicate can be found for any gate. But record it as a stated bound: axerror.Decode reads peer-supplied provider, bridge, RPC, session-adapter and terminal-backend envelopes, and payload size is attacker-controlled on those paths. If a large-document case is cheap, take it; if not, name the bound.
2. EQUIVALENT, DO NOT 'FIX'. Setting Reading.exitStatus from failure.ExitCode() instead of output.ExitStatus survives and is provably equivalent — readFailure has already refused unless the two are equal. The reviewer reproduced your own conclusion rather than taking it on trust. Leave it.

METHOD
- Confirm every mutant is PRESENT in the file before believing a green or a red, and confirm the test you filtered for actually RAN. A green from a -run pattern that matched nothing is not a green; that error was made twice in this session's orchestration.
- Prove the bound by NARROWING across the domain, not by deleting the guard.
- Do not weaken anything accepted in rounds 1-3.

spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 and only admitted pair; covering a command-tag guard whose narrowing admits forged authority documents across the whole suite."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260903-98a051, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260903-98a051)
ROUND-5 REWORK COMPLETE \u2014 commit 1d0d702 (amend of b43a746), exactly one commit past checkpoint f34d91d, worktree clean, signature verified Good. No production code changed.

F1 (BLOCKING) SETTLED \u2014 the command-tag agreement, readSuccess `if result.Command() != command`.
Reproduced FIRST at the reviewed head, before any test was written, measured against `go test ./... -count=1`: sparing CommandDoctor -> KILLED (control, the one existing row was real), sparing CommandList -> SURVIVED, admitting any document CLAIMING to be a takeover result -> SURVIVED all 13 packages. All three KILLED after the fix. Reports: repro-r5.json / postfix-r5.json.
FIX: the agreement is driven over the whole implemented vocabulary in BOTH directions - every implemented tag as the invocation against every other implemented tag's own emitted document. 306 ordered pairs, each required to refuse with this guard's OWN sentence naming both tags, so a pair settled by the version binding does not count as coverage of it; the 18 agreeing pairs are required to be ADMITTED in the same loop as the positive control against a guard that refuses everything. Counts asserted against ImplementedCommands(); takeover, list and doctor asserted present in the swept vocabulary.
RATIO, not prose: TestTheCommandAgreementGuardOwnsAMeasuredShareOfTheTagVocabulary drives all 1,936 ordered pairs over the 44 registered tags and classifies every answer - 18 admitted, 306 by this guard, 1,144 refused before stdout is read (the invoked tag selects an unbuilt version), 468 by the version binding (the document claims a tag of another major). Each non-guard class asserts the refusal does NOT quote the guard's sentence. STATED BOUND: for a pair involving one of the 26 unimplemented tags the guard is never the enforcement - a measured class with a named earlier refusal behind it, not an assumption that nothing was bypassed.

OBSERVATION A \u2014 EXTENDED AND CLOSED, not recorded as a bound. The 4096-byte narrowing the reviewer measured on axerror.requireCommonDataModel ALSO SURVIVED on documentSchema's copy of the same gate in cliresult: the finding was reported for one reader and held in both. Both closed with a conforming document several times that size - a Structured Error padded through its 64-key details bound, and a CLI Result carrying twelve Session Summaries emitted through the production emitter. Each row asserts the undoubled document is admitted first and asserts the byte length. A size fixture must have ONE knob: the first padding helper padded through both the message bound and details, so narrowing either alone left the document over 4096 and the mutant survived as equivalent.

OBSERVATION B \u2014 LEFT ALONE, as instructed. Setting Reading.exitStatus from failure.ExitCode() is provably equivalent because readFailure already refused unless they agree. The round-2 and round-3 dismissals are likewise untouched.

MUTATION: harness 44 -> 53 mutants, 52 killed, 1 declared subsumed (unchanged), 0 unexplained survivors, 36 narrowing by the harness's own definition, suite restored green, harness exit 0. Nine new rows: three on the guard (spare an invoked command, admit forged takeover claims, restrict to exactly the old single-pair coverage), two on the tests' own enumerators (drop takeover from the sweep, measure the ratio over the implemented tags instead of the registered ones), two payload-size narrowings, two fixture-padding narrowings. Every mutant verified present on disk carrying the MUTATED text and compiled before measurement.

TRACEABILITY: both new tests register against EXISTING acceptance cases, so no coverage figure moved. tracecheck still reports acceptance_cases=74, clauses_discharged=17/403, normative_sections=36, bindings=49. reviewedOwnershipCanonicalSHA256 re-pinned d9a860aa..307d -> 0c8a3014..6e31. Each registration probed by renaming its declaration: tracecheck exits 1 NAMING the missing declaration, ahead of the projection-digest check.

PUBLISHED FIGURES: README and LOGBOOK harness counts moved 44/43/27 -> 53/52/36, both restating the existing bound that the harness mutates the tree, lives under .temp/, and no committed artifact derives its counts. The two derived-figure pins (fan-in table, ownership paragraph) pass unchanged.

VALIDATION EXIT CODES, measured: go build 0, go vet 0, gofmt -l no output, go test ./... -count=1 exit 0 (13 packages), go test ./... -cover -count=1 exit 0 (cliresult 95.5%, axerror 99.5%, both unchanged), tracecheck 0, mutation harness 0.

DISCLOSURES: the cliresult payload-size hole was found by extending the reviewer's axerror observation to the sibling gate, not reported by the review; it was closed rather than recorded. No durable state is mutated anywhere in this leaf, so there is no crash/idempotency surface to evidence. clauses_discharged stays 17/403 and section 14.2 stays partial at 8/9 - this round adds evidence for an existing claim and claims nothing new. Nothing accepted in rounds 1-4 was weakened; every round-4 harness row still runs and still kills.

CR WARNING FOR THE REVIEWER AND THE ORCHESTRATOR \u2014 round 5. `task-board handoff` exited 0 and constructed NO new Change Request revision; CR construction happens at spawn-run completion, not at handoff. The newest stored revision is CR-TASK-260830-33sfxc-4 rev4, whose candidate_tree_oid is 2e893d0d9b476991e481098bafff9b685374aa18 \u2014 the PRE-amend tree of b43a746. This round amended that commit.

The reviewable tree is now 42898ceb9945bf52251e846548b7bec9826d9085 at commit 1d0d702a7eee7a9cc3781d2bf55ae5cdae1bb4f4. Verify candidate_tree_oid against it before reviewing any stored CR patch; review the commit itself if no rev5 was built for it.

REVIEWABLE DIFFS ATTACHED, because the stored CR patch predates this amend. TASK-260830-33sfxc_rework-r5-delta.patch is git diff --binary b43a746..1d0d702 \u2014 7 files, 385 insertions, 12 deletions, the round-5 delta only, and the shortest honest read for this round. TASK-260830-33sfxc_leaf-full-r5.patch is git diff --binary f34d91d..1d0d702, the whole story_final leaf against its checkpoint. Use these until a CR revision exists whose candidate_tree_oid equals 42898ceb9945bf52251e846548b7bec9826d9085.

RE-RUN THESE TO CHECK THE ROUND-5 CLAIM RATHER THAN READING IT: the two narrowings the rev4 review measured as SURVIVED are permanent harness rows now \u2014 `command != CommandList` and `result.Command() != CommandTakeover` on internal/cliresult/client.go readSuccess \u2014 and both must come back KILLED. TASK-260830-33sfxc_repro-r5.py runs exactly those two plus the doctor-sparing control against the full repository suite.

agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260903-98a051, pid=41315, exit=0)
REVIEW ROUND 5 — FINAL LEAF, NARROW. Your acceptance makes this the Story's integration unit.

WHICH COMMIT. CR rev 5, 69-path delta, three commits past origin/main. Verify reachability first.

ROUNDS 1-4 ARE CLOSED, each verified closed independently by the reviewer that followed:
1. a duplicate-member bypass forging a retry claim, reachable from a remote peer, plus a fan-in table wrong in two of four rows;
2. two surviving narrowing mutants on the gates this leaf exists to prove;
3. a published figure off by one, the third instance of that class in one artifact;
4. the command-tag agreement proven for ONE ordered pair, where a narrowing admitting any document CLAIMING to be a takeover result survived all 13 packages.

ORCHESTRATOR VERIFICATION ALREADY PERFORMED — EXTEND, DO NOT REPEAT
The round-4 mutant is dead. Re-planting `result.Command() != command && result.Command() != CommandTakeover` now reddens TWO tests, including a new TestTheCommandAgreementGuardOwnsAMeasuredShareOfTheTagVocabulary. Mutant confirmed present before measuring; tree restored clean afterwards.

WHAT TO ATTACK
1. THE NEW TEST'S OWN CLAIM. Its name asserts a MEASURED SHARE of the tag vocabulary. Confirm the share is derived from Commands()/ImplementedCommands() at run time rather than from a literal list that can drift, and that adding a command to the registry without covering it reddens. A test that names a ratio and pins a constant is the shape this board has removed three times in this artifact alone.
2. THE UNIMPLEMENTED-TAG REASONING. The producer's comment says the guard is never reached for a pair involving an unimplemented tag, and that the forgery is refused elsewhere for those. Verify that claim by probe rather than reading it — if the alternative refusal does not exist, the share is honest but the residual is understated.
3. THE 4096-BYTE STATED BOUND from round 4. The reviewer declined to block on it because a byte-length predicate is not a dimension of the §1.6 contract. Confirm the producer either covered it with a large-document case or recorded it as a stated bound with the peer-supplied-envelope reasoning intact. Either is acceptable; silently dropping it is not.
4. THE EQUIVALENT SURVIVOR round 4 identified — Reading.exitStatus from failure.ExitCode() — must NOT have been 'fixed'. A survivor dismissed with a reason is evidence; one quietly changed hides why it survived.
5. Confirm nothing from rounds 1-4 regressed.

STANDING REVIEW BAR
- Positive-path-only evidence is refused. Prove a bound by NARROWING across the domain, not by deleting the guard.
- Confirm every mutant is PRESENT before believing a green or a red, and confirm the test you filtered for actually RAN.
- FINAL LEAF: accept with accept_cr if it holds — check your reviewer DoD items first, do not supply commit_ack. Six rounds is neither a reason to accept nor a reason to keep hunting; judge this revision on what it is, and if it holds, say so.

spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Rank-1 and only admitted pair; round-5 review checking whether a newly claimed measured share is derived or merely named."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260903-1a92ce, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260903-1a92ce)
REVIEW ROUND 5 — CHANGES REQUESTED. Full verdict: TASK-260830-33sfxc_review-verdict-rev5.md. Evidence: TASK-260830-33sfxc_review-rev5-independent-mutation.json (29 independent mutants), _review-rev5-mutants.py, _review-rev5-go-test-full.log, _review-rev5-tracecheck.log.

REVIEWED BYTES: git rev-parse HEAD^{tree} = 42898ceb9945bf52251e846548b7bec9826d9085 = the candidate tree. Working tree returned byte-identical after every probe.

RE-RUN HERE, NOT ACCEPTED FROM THE LOGS: go build ./... 0; go vet ./... 0; gofmt -l . none; go test ./... -count=1 exit 0 across 13 packages; tracecheck exit 0 with acceptance_cases=74 clauses_discharged=17/403 normative_sections=36 bindings=49, matching every published figure.

ROUND-4 FINDING CLOSED. Both surviving command-agreement narrowings re-run against the full suite and both now KILLED (command != CommandList; result.Command() != CommandTakeover), plus the doctor-sparing control. The 44x44 grid in TestTheCommandAgreementGuardOwnsAMeasuredShareOfTheTagVocabulary is a real measured ratio.

BLOCKING F1 — internal/axerror/decode.go:285, `if exitCode != expectedExit`, the guard refusing a Structured Error whose code and exit_code name different Section 15.2 classes. Coverage is DELETE-ONLY. Measured against go test ./... -count=1, each mutant verified present on disk and compiled:
  `if false && exitCode != expectedExit` (delete, control) -> KILLED
  `&& code != "policy_refused"`                          -> SURVIVED
  `&& code != "authentication_failed"`                   -> SURVIVED
  `&& code == "observation_gap"` (restrict the guard to exactly its one covered code) -> SURVIVED, all 13 packages ok
The whole class is one row of TestDecodeRefusesClosedShapeViolations: one code, one wrong status. Class size derived here from CodesFor/CodesByExitStatus: 752 / 1056 / 1504 / 1744 ordered (code, wrong-status) pairs for 1.0.0 / 1.1.0 / 1.2.0 / 1.3.0. Coverage is 1 of 752.

It is a bypass path, not only a gap. The guard is the sole owner of the refusal (probed at the unmutated tree: policy_refused @ exit 9 and transaction_unknown @ exit 9 are refused only by its own sentence). retryabilityRefusalsByExitStatus keys on the EXIT STATUS, so relabelling a code out of its class disarms it. Through the production entry point cliresult.Read, with only the observation_gap narrowing applied: a document carrying code authentication_failed (registered at exit 7, the authorization class forbidden from claiming retryable) with exit_code 9 and retryable true is ADMITTED — Reading.Code()=authentication_failed, ExitStatus()=9, Retryable()=true — and the whole repository suite stays green.

This is the leaf own contract: Read is designed as document-decides / exit-status-corroborates, readFailure exit_code equality is swept, and this second corroboration is what holds the code and the exit_code in the same class. CodesByExitStatus and the fan-in quoted in exitStatusIsNotEnough — both added by this leaf — describe a mapping that only exists at read time because this guard enforces it. The rework doc states bounds for the harness figures, the payload-size narrowings and the dismissed equivalents, and states no bound for this guard.

WHAT CLOSES IT (test-only, no production change): mirror TestTheCommandAgreementGuardOwnsAMeasuredShareOfTheTagVocabulary — per registered version, drive every registered code against every registered failure status other than its own, require ErrInvalidStructuredError AND this guard own sentence `maps to exit %d, document carries %d` so a pair settled by decodeExitStatus or the retryability gate does not count, admit the agreeing pair as the positive control, and assert counts against CodesFor/CodesByExitStatus.

NON-BLOCKING: (a) decodeExitStatus !IsFailureExitStatus is sampled at {0,1,18,99}; `&& status != 42` SURVIVES. Registered side is swept, so weaker than F1; a 0..255 sweep asserting admitted-iff-IsFailureExitStatus closes it cheaply and DecodeBound reaches it without any process exit status. (b) EQUIVALENT MUTANTS, do not chase in round 6: sparing terminal_backend_capability_unproven in ExitCodeFor version scoping, and sparing session_not_found or provider_auth_smoke_failed in F1 — none is a registered code. Version scoping itself is covered: the same narrowing on the registered terminal_backend_stale_generation is KILLED. (c) README sentence Every gate above has a negative test that narrows it rather than deleting it (leaf 1, ebc4e31) is falsified by F1; closing F1 makes it true again.

HELD UNDER ATTACK: Read unregistered-exit gate spared at 42; readSuccess success-status gate spared at exit 7; readFailure exit_code equality spared at exit 7; documentSchema absence gate narrowed to len(stdout)==0; documentSchema trailing-content gate removed; documentSchema common-data-model gate narrowed to spare Structured Errors; exitStatusIsNotEnough fan-in off by one; requireCommonDataModel skipped for 1.1.0; verifyEnvelopeIdentity schema gate widened to the CLI Result schema; RetryabilityRefusal spared operation_uncertain and spared exit class 7; ExitCodeFor version scoping narrowed and deleted — all KILLED. Historical corpora frozen in both directions (deleted fixture and unlisted extra fixture both redden). README fan-in pin reddens on a perturbed cell. Message-independence replays assert byte change, message movement and replay counts. InvocationOutput structurally cannot carry stderr. Section 17.2 reported unevidenced 0/1 with a stated gap.

ROUTING: to-dev. One sweep test closes F1; the 0..255 sweep closes the observation in the same pass. Re-review must re-run the three F1 narrowings and require all three KILLED with the delete-only control still KILLED.
LOGBOOK: not written by this review. LOGBOOK.md is a tracked file inside the candidate tree and a reviewer writing it would dirty the reviewed bytes (HEAD^{tree} must stay 42898ceb). The F1 finding belongs in the round-6 rework logbook entry, which already touches LOGBOOK.md — record it there as: the code-to-exit-status agreement guard at internal/axerror/decode.go:285 had delete-only coverage (1 of 752 ordered pairs at 1.0.0), and because retryabilityRefusalsByExitStatus keys on the exit status, the gap was a forged-retry bypass reachable through cliresult.Read.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260903-1a92ce, pid=88135, exit=0)
REWORK BRIEF — ONE BLOCKING FINDING, AND A SWEEP I WANT DONE NOW RATHER THAN IN ANOTHER ROUND

The round-4 finding is closed and every gate in this leaf's own new code held. This finding is in scope and I have checked the scope argument rather than taking it: it is not 'a neighbouring package is under-tested'.

F1 (BLOCKING) — THE CODE-TO-EXIT-STATUS AGREEMENT HAS DELETE-ONLY COVERAGE, AND A FORGED RETRY CLAIM REACHES THROUGH IT
internal/axerror/decode.go:285, `if exitCode != expectedExit`. Measured against the full 13-package suite:
  delete the gate (control)                          -> KILLED
  spare one registered code (policy_refused)         -> SURVIVED
  spare a different code (authentication_failed)     -> SURVIVED
  restrict the gate to the ONE code its coverage drives (observation_gap) -> SURVIVED, all 13 packages ok
The whole class is covered by one row of TestDecodeRefusesClosedShapeViolations: one code, one wrong status. The class is 752 ordered (code, wrong-status) pairs at 1.0.0 alone.
The reach, reproduced: a document with code authentication_failed and exit_code 9, read at process status 9, is ADMITTED with retryable=true. authentication_failed is registered at exit 7, a class this package forbids from claiming retryable — TestRetryableRefusedForEveryForbiddenClass exists for exactly that. Moving exit_code to 9 walks around it, and the machine client is handed a Reading whose code and exit status name different classes with a safe-retry claim attached.
WHY IT IS YOURS: Read's design is 'the document decides the outcome and the exit status corroborates it'. readFailure's corroboration — exit_code equals the process status — is swept by setting both to 9. This second corroboration, that the code and exit_code belong to the same class, is the one holding the pair together, and CodesByExitStatus and the fan-in you quote in exitStatusIsNotEnough describe a mapping that only exists at read time because this guard enforces it.
Your results doc states bounds for the harness figures, the payload-size narrowings and the dismissed equivalent survivors. It states NO bound for this guard — it is unmeasured rather than knowingly limited. That distinction is the finding.
WHAT CLOSES IT — the pattern is already yours. Mirror TestTheCommandAgreementGuardOwnsAMeasuredShareOfTheTagVocabulary: for every registered version, drive every registered code against every registered failure status other than its own; require ErrInvalidStructuredError AND this guard's own sentence 'maps to exit %d, document carries %d', so a pair settled by decodeExitStatus or the retryability gate does not count as coverage of THIS guard; require the agreeing pair admitted as the positive control; assert the counts against CodesFor/CodesByExitStatus so the loop cannot go vacuous. Test-only; no production code moves.

THE SWEEP — DO THIS BEFORE HANDING OFF
Rounds 4 and 5 found the same shape twice: a guard whose coverage drives one point of a domain the guard owns across many. You closed the first with a measured-share test and did not look for siblings. Do that now.
Enumerate every refusal guard reachable from Read and from the two decoders, and for each state one of three things: it is exercised across its domain with the share measured; it is exercised at one point and that is a STATED bound with a reason; or it is unmeasured and you are fixing it. A guard with no statement in that list is the next round's finding, and I would rather have your judgement on all of them now than discover them one per round.
Report the list as a table. Where you state a bound rather than closing it, the reason must be about the contract — 'a byte-length predicate is not a dimension of §1.6' is a real reason; 'the test would be long' is not.

METHOD
- Confirm every mutant is PRESENT before believing a green or a red, and confirm the test you filtered for actually RAN.
- Prove a bound by NARROWING across the domain, not by deleting the guard. Delete-only coverage shows a gate exists; it says nothing about what it covers.
- Do not weaken anything accepted in rounds 1-4. Do not 'fix' the survivors previous reviewers dismissed with reasons.

spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 and only admitted pair; closing a forged-retry reach plus a sweep of every reachable guard so the class is settled in one pass."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260903-6054b5, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260903-6054b5)
Revision 6 at amended head e708c4f (one commit past checkpoint f34d91d). Round-5 F1 closed: the code-to-exit-status agreement in axerror.decodeBody is now swept over all 316 registered codes x 4 versions x 17 registered failure statuses (5,372 rows; 316 admitted at the agreeing status, 5,056 refused with this guard own sentence naming both numbers), with per-version totals 752/1,056/1,504/1,744 re-derived from CodesFor and CodesByExitStatus. The bypass is closed at the production entry point too: TestRelabellingACodeOutOfItsExitClassCannotForgeARetryClaimThroughRead relabels the 7 codes of Structured Error 1.0.0 whose class carries an exit-keyed retryability refusal into all 14 statuses that carry none and requires all 98 to be refused, with the retryability-gate control in the same loop. All three reviewer narrowings reproduced SURVIVED at 1d0d702 first and are KILLED here, delete-only control still KILLED. The non-blocking observation is closed in the same pass: decodeExitStatus swept over 0..255 (17 admitted iff registered, 239 refused) plus six out-of-range values, with ExitStatusMeaning as the oracle rather than the predicate under test. No production code changed. Gates, each run standalone: gofmt -l ./internal exit 0, go build exit 0, go vet exit 0, go test ./... -count=1 exit 0 across 13 packages, go test ./... -cover -count=1 exit 0 (cliresult 95.5%, axerror 99.5%, unchanged), tracecheck exit 0 at acceptance_cases=74 clauses_discharged=17/403 (unchanged). Harness 53 -> 63 mutants: 62 killed, 1 declared subsumed, 0 unexplained survivors, 44 narrowing. Harness figures remain a stated bound, not a gated measurement.
CR WARNING FOR THE REVIEWER AND THE ORCHESTRATOR - round 6. task-board handoff exited 0 and constructed NO new Change Request revision; CR construction happens at spawn-run completion, not at handoff. The newest stored revision is CR-TASK-260830-33sfxc-5 rev5, whose candidate_tree_oid is 42898ceb9945bf52251e846548b7bec9826d9085 - the PRE-amend tree of 1d0d702. This round amended that commit.

The reviewable tree is now 8f2a9527298fcd0a2b01bcacfb4f30e73a52f542 at commit e708c4fbc623782305243c0ff88f655eabbdf3ae, still exactly one commit past the checkpoint f34d91d. Verify candidate_tree_oid against it before reviewing any stored CR patch; review the commit itself if no rev6 was built for it.

REVIEWABLE DIFFS ATTACHED, because the stored CR patch predates this amend. TASK-260830-33sfxc_rework-r6-delta.patch is git diff --binary 1d0d702..e708c4f - 6 files, 596 insertions, 7 deletions, the round-6 delta only and the shortest honest read for this round; no production source file is in it. TASK-260830-33sfxc_leaf-full-r6.patch is git diff --binary f34d91d..e708c4f, the whole story_final leaf against its checkpoint. Use these until a CR revision exists whose candidate_tree_oid equals 8f2a9527298fcd0a2b01bcacfb4f30e73a52f542.

RE-REVIEW ASK, from the round-5 verdict routing: re-run the three F1 narrowings on internal/axerror/decode.go line 285 (code != policy_refused, code != authentication_failed, code == observation_gap) and require all three KILLED with the delete-only control still KILLED, plus the decodeExitStatus narrowing status != 42. All four reproduce as SURVIVED at 1d0d702 and as KILLED here; the harness at TASK-260830-33sfxc_mutants-r6.py carries all of them plus a version-scoping narrowing and three test-side narrowings that prove the new sweeps fail closed.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260903-6054b5, pid=13295, exit=0)
REVIEW ROUND 6 — FINAL LEAF. Your acceptance makes this the Story's integration unit.

WHICH COMMIT. CR rev 6, 71-path delta. Verify reachability first.

ROUND 5's BLOCKING FINDING IS CLOSED, and the producer went past it.
- The code-to-exit-status agreement is now measured over the WHOLE registry: 752 / 1,056 / 1,504 / 1,744 refused (code, wrong-status) pairs for 1.0.0 / 1.1.0 / 1.2.0 / 1.3.0, with the agreeing pair admitted as positive control. All three round-5 survivors are killed, plus one the producer added itself.
- decodeExitStatus is swept over every byte value 0..255 plus six out-of-range values: 17 admitted, 239 refused.
- A further bypass at cliresult.Read was found by the producer's own sweep and closed (relabel_test.go).

I ASKED FOR ONE THING THAT WAS NOT DELIVERED AS ASKED, AND I WANT YOUR JUDGEMENT ON WHETHER IT MATTERS
I asked for an enumerated table of every refusal guard reachable from Read and the two decoders, each marked as: measured across its domain, a stated bound with a contract reason, or unmeasured and being fixed. The producer did the WORK — closed F1 across the registry, swept the byte range, found another bypass, and annotated three stated bounds in README — but produced no consolidated list.
The table was never the point. The question it was meant to answer is: IS THERE A REACHABLE REFUSAL GUARD IN THIS LEAF THAT IS NEITHER MEASURED ACROSS ITS DOMAIN NOR CARRIES A STATED BOUND? Answer that directly. If the answer is no, the missing table costs nothing and I do not want it manufactured. If the answer is yes, name the guard — that is a finding, and it is the seventh round of the same class, which would say the sweep needs to be a table after all.

ALSO VERIFY
1. The measured-share tests must DERIVE their domains from the production enumerations at run time — CodesFor, CodesByExitStatus, Commands, ImplementedCommands — not from literal lists. A test that names a ratio and pins a constant is the shape this artifact has shed three times.
2. The 0..255 sweep's oracle is ExitStatusMeaning rather than IsFailureExitStatus, on the stated ground that the latter IS the decision under test and an oracle built from it would move with the mutant. Verify that reasoning holds and that the asserted row count of 17 catches a mutation moving both.
3. The producer disclosed that the inherited README sentence 'Every gate above has a negative test that narrows it rather than deleting it' was published BEFORE it was true of the reader, and says so in the paragraph now. Confirm the sentence is true today and that the disclosure is accurate rather than softened.
4. The equivalent survivors from rounds 2, 3 and 5 must NOT have been 'fixed'. The producer states none of the three round-5 codes is registered, so none reaches the mutated branch. Verify that claim.
5. Nothing from rounds 1-5 regressed.

STANDING REVIEW BAR
- Positive-path-only evidence is refused. Prove a bound by NARROWING across the domain, not by deleting.
- Confirm every mutant is PRESENT before believing a green or a red, and confirm the test you filtered for actually RAN.
- FINAL LEAF: accept with accept_cr if it holds — check your reviewer DoD items first, do not supply commit_ack. Seven rounds is not a reason to accept and not a reason to keep hunting. If it holds, say so plainly.

spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Rank-1 and only admitted pair; round-6 review answering whether any reachable guard remains without a measured domain or a stated bound."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260903-250f8d, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260903-250f8d)
REVIEW ROUND 6 (RUN-260903-250f8d) — CHANGES REQUESTED, one blocking finding. Full verdict: TASK-260830-33sfxc_review-verdict-rev6.md. Evidence: TASK-260830-33sfxc_review-rev6-mutation.json (31 independent mutants), _review-rev6-mutants.py, _review-rev6-go-test.log.

REVIEWED BYTES: git rev-parse HEAD^{tree} = 8f2a9527298fcd0a2b01bcacfb4f30e73a52f542 = the candidate tree. Commit e708c4f. Working tree byte-identical after every probe.

RE-RUN HERE, NOT ACCEPTED FROM THE LOGS: go build 0; go vet 0; gofmt -l . none; go test ./... -count=1 exit 0 across 13 packages; tracecheck exit 0 at acceptance_cases=74 clauses_discharged=17/403 normative_sections=36 bindings=49 contracts=60 fixtures=30 compatibility_contracts=55. Every published figure reproduces.

ROUND-5 FINDING CLOSED. All four narrowings the producer asked to be re-run come back KILLED (code==observation_gap, code!=policy_refused, code!=authentication_failed, decodeExitStatus status!=42), delete-only control still KILLED, plus an out-of-range arm status!=256 KILLED.

THE QUESTION THIS ROUND ASKED, ANSWERED DIRECTLY: yes, there is one reachable refusal guard in this leaf that is neither measured across its domain nor carries a stated bound.

BLOCKING F1 — internal/cliresult/client.go:121, the ErrUnregisteredExitStatus gate. New production code in this CR. Its whole coverage is one subtest driving {1, 42, -1, 255} (client_test.go:511). Measured against go test ./... -count=1, each mutant verified present on disk and compiled: `if false && ...` (delete, control) -> KILLED, so the four rows are real; `&& ExitStatus != 127 && ExitStatus != 137` -> SURVIVED; the guard restricted to exactly its four covered values -> SURVIVED, all 13 packages ok. That last is round-5 F1s shape exactly.

NOT an admission bypass and I do not claim one: a failure doc is still refused by readFailures equality, a success doc by readSuccesss status guard, and decodeExitStatus still refuses an unregistered exit_code. What changes is WHICH FACT the client is told, which is the deliverable this leaf spent a round establishing. Reproduced through Read with only the {127,137} narrowing (command-not-found and SIGKILL/OOM — the two statuses a wrapper most plausibly returns when ax never ran): absent stdout at 127 answers ErrAbsentDocument instead of ErrUnregisteredExitStatus, and the refusal states `exit status 127 is assigned to 0 registered Structured Error 1.0.0 codes` — a fabricated fan-in for a status the 15.2 table does not carry. exitStatusIsNotEnough computes len(fanIn[status]) with no membership check, documentSchema has exactly one caller, so this gate is the SOLE enforcement of the thing that goes wrong. README:1548 publishes that function as a measured count rather than advice; 0 there is a map miss reported as a measurement.

WHY IT IS THIS LEAFS: the sibling gate decodeExitStatus had the identical four-value shape, was flagged in round 5, and was swept 0..255 THIS ROUND. The reader-side twin has the same domain and the sweep helper already exists in client_test.go. The rework results state bounds for the harness figures, the payload-size narrowings and the dismissed equivalents; none for this gate. Unmeasured rather than knowingly limited.

CLOSES IT (test-only): mirror TestTheExitStatusAdmissionIsSweptOverEveryByteValue on the reader — drive Read over 0..255 plus out-of-range, admit past this gate iff axerror.IsFailureExitStatus, assert the admitted count against the enumeration, add the restrict-to-the-sample narrowing to the harness. While there, decide whether exitStatusIsNotEnough should distinguish `this status carries no registered code` from `this status is not in the table`.

NON-BLOCKING, CLASSIFIED:
O1 — a restated reason in the record is WRONG though its conclusion survives. Round 5 dismissed three mutants because `none is a registered code`; terminal_backend_capability_unproven IS registered at 1.3.0 exit 6 (catalog_gen.go:541). Sparing it in ExitCodeFor version scoping SURVIVES on a clean tree. The conclusion holds for a different reason: the same narrowing on observation_gap (also partially registered) is KILLED by TestExitCodeForRefusesVersionAndCodeDrift, so the class has a covered representative.
O2 — documentSchemas canonical gate narrowed to duplicate-key errors only SURVIVES while the axerror twin is KILLED. Probing further, canonicaljson.Canonicalize does not enforce the float/safe-integer half of 1.6 at all: Read ADMITS a Structured Error carrying `details:{n:1.5}` at exit 5, which SPEC.md:221 forbids. The gates doc comment enumerates what it owns honestly, so this is an overclaim only in its opening sentence and READMEs matching phrasing; section:1.6 is bound to internal/scalar and left unevidenced, so nothing false is claimed in traceability. Not disclosed anywhere — the leafs brief asks for exactly this.
O3 — verifyEnvelopeIdentitys major gate is proven at major 2 only; (major != 1 && major != 3) SURVIVES. Measured, not inferred: a 3.0.0 document then reports ErrUnsupportedVersion, the same sentinel as the same-major unknown minor 1.9.0. Leaf-1 code and an unbounded domain, so a sample is defensible — state it as a bound rather than leaving it silent.
O4 — narrowing documentSchemas trailing-content and missing-schema gates by exit status both SURVIVE. Exit status is not a dimension of 14.2s exactly-one-document rule, so this is the arbitrary-predicate class round 4 declined to block on. No action wanted.

ORCHESTRATOR CHECKS, ANSWERED: (1) domains ARE derived at run time — the agreement sweep iterates Versions/CodesFor/CodesByExitStatus and cross-checks its literal totals against len(codes)*(len(statuses)-1) in the same assertion. (2) the ExitStatusMeaning oracle reasoning HOLDS by measurement — narrowing IsFailureExitStatus is KILLED by the sweep itself, and adding an exitMeanings row, which moves both, is KILLED by the asserted 17-count. (3) the README disclosure paragraph is accurate and unsoftened; it names both axerror gates. My F1 falsifies the sibling sentence at README:1453 over the cliresult section. (4) the round-2/3/5 equivalent survivors were NOT fixed — see O1 for the one restated reason that is wrong. (5) nothing from rounds 1-5 regressed: the round-1 duplicate gate, round-2 status/equality guards at a different status, the round-4 command agreement on a different tag, the stderr structural pin and the frozen corpus SHA-256 pins are all KILLED under attack.

STATED BOUNDS OF THIS REVIEW: depth concentrated on guards reachable from Read and the two decoders; leaves 1 and 2 probed through the entry point, not re-reviewed in full. The producers 63-mutant harness was NOT re-executed — my 31 mutants were written independently and measured against the full suite. Coverage percentages NOT re-measured. One batch of 8 was DISCARDED and re-run: a probe package main directory I left in the scratch module reddened TestModuleHasNoProductCommandYet and would have recorded a false KILL for the capability_unproven row; only the clean re-run is in the report.

ROUTING: to-dev. On re-review, require the restrict-to-the-sample narrowing of client.go:121 KILLED with the delete-only control still KILLED.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260903-250f8d, pid=48994, exit=0)
REWORK BRIEF — ONE BLOCKING FINDING, AND IT IS THE TWIN OF THE ONE YOU JUST FIXED

F1 (BLOCKING) — THE Read-LEVEL UNREGISTERED-EXIT-STATUS GATE IS PROVEN AT FOUR POINTS AND BOUNDS NOTHING
internal/cliresult/client.go:121, the ErrUnregisteredExitStatus gate. It is NEW production code in this Change Request. It is exercised at four sample points of a 240-value domain. Restricting it to exactly those four points passes all thirteen packages. No bound is stated for it in README, LOGBOOK, or the rework results.
It is the reader-side twin of decodeExitStatus, which you swept over 0..255 in this very revision after round 5 flagged the identical four-value shape. The pattern was applied to one of the two.
Close it the same way you closed its twin: sweep the domain, derive it from the production enumeration rather than a literal list, and assert the admitted/refused counts so the loop cannot go vacuous.

WHAT THE REVIEWER CONFIRMED, SO YOU KNOW WHERE THIS ENDS
Every other reachable guard is either measured across its domain or carries a stated bound. This is the last instance of the class, not the next in an unbounded series.

AN ORCHESTRATOR ERROR THAT COST THIS ROUND, STATED SO IT IS NOT REPEATED
I asked for an enumerated inventory of every reachable guard, each marked measured / stated-bound / being-fixed. You did the sweep's work and not its list, and I decided not to insist — reasoning that the table was a means, not an end, and that demanding a format for its own sake is ceremony.
That was wrong, and this finding is why. The list is not reporting; it is a forced traversal. Working from memory closes the guard you were told about and misses its twin one file away. A table would have put both rows next to each other and made the omission visible before a reviewer had to find it.
So: produce the list this time, with this gate and its twin as two rows of it. Not as a deliverable for its own sake — as the mechanism that stops the seventh instance.

METHOD
- Confirm every mutant is PRESENT before believing a green or a red, and confirm the test you filtered for actually RAN.
- Prove a bound by NARROWING across the domain, not by deleting the guard.
- Do not weaken anything accepted in rounds 1-5. Do not 'fix' survivors previous reviewers dismissed with reasons.
- Where you state a bound instead of measuring, the reason must be about the contract, not about effort.

spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 and only admitted pair; sweeping the reader-side twin of a gate already swept, plus the guard inventory that would have caught it."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260903-541bc0, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260903-541bc0)
ROUND-7 REWORK COMPLETE (developer). Commit amended to adca16a, tree 67961efb6a03edfecd442df798d20368ca185cd2. Working tree clean; one commit per leaf, three past main.

F1 (BLOCKING) CLOSED. internal/cliresult/client.go:121, the ErrUnregisteredExitStatus gate. REPRODUCED ON THE UNCHANGED REVIEWED TREE FIRST (TASK-260830-33sfxc_repro-r7.json): restricted to exactly its four covered values {1,42,-1,255} SURVIVED go test ./... -count=1 at exit 0; narrowed to spare 127 and 137 SURVIVED; the delete-only control KILLED in the same run; tree restored green. Closed the way its twin was: TestTheReadLevelExitStatusAdmissionIsSweptOverTheWholeDomain drives Read over 0..255 plus eight out-of-range values, admits past the gate iff the Section 15.2 MEANING TABLE registers the status (oracle is ExitStatusMeaning, not the IsFailureExitStatus predicate the gate calls), asserts admitted=18 and refused=238, cross-checks both production predicates against the oracle at every point, and requires the refusal never to carry the fan-in sentence. A second test drives a conforming document at every registered status so admission means classified, not merely refused differently.

THE LIST, PRODUCED AND DERIVED. TestTheRefusalGuardInventoryIsDerivedFromTheReaderSource parses every refusal site out of client.go with go/ast and requires a BIJECTION with twelve published README rows under "### Refusal guard inventory". Each row: function, sentinel, refusal marker checked against the format literal, domain-evidence class from a closed vocabulary (measured / stated bound / declared subsumed), and test declarations that must exist. A second table enumerates the seven guards this leaf added or reordered outside that file, with decodeExitStatus required by name — the twin and its counterpart as adjacent rows, which is what the brief asked for. Attacked in both directions, eight mutants, all KILLED: deleted row, row naming a nonexistent test, unresolvable marker, evidence class outside the vocabulary, dropped twin row, renamed sibling function, a NEW refusal added to the reader with no row, and the derivation narrowed to skip one function.

THE TRAVERSAL FOUND THREE MORE INSTANCES NO REVIEWER HAD NAMED (TASK-260830-33sfxc_traverse-r7.json, all SURVIVED before this round). (a) The ErrForeignDocument branch was covered by one schema identifier; restricting it to that identifier and routing every other foreign schema into readSuccess passed all 13 packages. Now measured over all 60 catalog contract identifiers plus twelve near-miss neighbours, denominator asserted against catalog.Current().Contracts. (b) The schema-member type guard was covered by the number 5; restricting it to JSON numbers passed. Now measured over every JSON value form with a string control. (c) The schema-member PRESENCE guard survives an arbitrary exit-status predicate — left as a STATED BOUND with the contract reason (exit status is not a dimension of Section 14.2 exactly-one-document), which is the class round 4 declined to block on.

PRODUCTION BEHAVIOUR CHANGE (one, found only by sweeping the domain): encoding/json admits JSON null into a string as "", so a document whose schema member was null was answered as a FOREIGN CONTRACT carrying schema "" — a claim that another contract owns the document when the truth is that this one is unreadable. The guard now checks the raw token alongside the unmarshal; the harness carries the mutant that restores the old behaviour.

exitStatusIsNotEnough SETTLED WITHOUT A DEFENSIVE BRANCH. A branch no production path can take promises nothing and is never measured. TestTheFanInSentenceCannotReportAMapMissAsAMeasurement proves the fabrication unreachable: documentSchema has exactly one caller and it is Read, exitStatusIsNotEnough is called only from documentSchema, the gate precedes that call by AST position, and every registered (version, failure status) pair measures at least one code, so a 0 could only have been a map miss. Moving the gate behind the discriminator KILLS it.

OBSERVATIONS. O1 CORRECTED: terminal_backend_capability_unproven IS registered at 1.3.0 exit 6 (catalog_gen.go:541, verified here), so the round-5 dismissal reason was wrong; the conclusion holds because the same narrowing on observation_gap is killed by TestExitCodeForRefusesVersionAndCodeDrift. Recorded in README and LOGBOOK. O2 REPRODUCED INDEPENDENTLY AND DISCLOSED: Read admits a Structured Error whose details carry 1.5, which SPEC.md forbids; Canonicalize does not enforce the number half of Section 1.6; that rule is bound to internal/scalar and left unevidenced, so nothing false is claimed in traceability. The admission is now ASSERTED, so closing the gap reddens the pin. O3 CLOSED BY MEASUREMENT rather than the offered stated bound: majors 0..64 swept through Read. O4 no action, recorded as row 7 stated bound. Equivalent survivors from rounds 2/3/5 left alone.

HONESTY NOTES. One round-7 mutant was DISCARDED as equivalent rather than recorded as a kill: disabling the assertion in the Section 1.6 disclosure test changes no outcome while the gap is open. It was replaced by a mutant that closes the gap in production and requires the pin to redden. One of my own new tests initially had a vacuous denominator (narrowing the catalog loop to two entries SURVIVED); the denominator is now asserted against the catalog and that mutant is KILLED.

VALIDATION, run on this tree as standalone processes with real exit codes: gofmt -l . exit 0 no output; go build ./... exit 0; go vet ./... exit 0; go test ./... -count=1 exit 0 across 13 packages; go test ./... -cover -count=1 exit 0 (cliresult 95.5%, axerror 99.5%, both unchanged); tracecheck exit 0 at acceptance_cases=74, clauses_discharged=17/403, normative_sections=36, bindings=49, contracts=60, fixtures=30, compatibility_contracts=55 — all unchanged. Eight new tests registered against the existing cli-result-machine-reading case; registration probed by renaming a declaration and confirming tracecheck names the missing declaration AHEAD of the digest check, then the digest re-pinned.

HARNESS 63 -> 86 mutants: 85 KILLED, 1 declared subsumed (unchanged), 0 unexplained survivors, 64 narrowing by the harness own definition, every narrowing name asserted to be a declared mutant. Run in three batches to stay inside the shell time bound; each batch restored the tree and re-ran the suite, all three restored_suite_exit_code values 0. Rounds 1-6 re-measured in full, nothing regressed.

STATED BOUNDS. The inventory derivation is exhaustive over internal/cliresult/client.go, which this leaf added in full; the sibling table is enumerated, not derived, and refusals carried from leaves 1 and 2 in axerror/decode.go, axerror/registry.go and cliresult/decode.go are those leaves inventories. The harness figures remain a stated bound: the harness mutates the tree, lives under .temp/, and no committed artifact derives its counts.
CR WARNING FOR THE REVIEWER AND THE ORCHESTRATOR — round 7. Verified after handoff, not assumed: task-board handoff exited 0 and constructed NO new Change Request revision. The newest stored revision is CR-TASK-260830-33sfxc-6 rev6, state ready, whose candidate_tree_oid is 8f2a9527298fcd0a2b01bcacfb4f30e73a52f542 — the PRE-amend tree of e708c4f. This round amended that commit.

THE REVIEWABLE TREE IS NOW 67961efb6a03edfecd442df798d20368ca185cd2 at commit adca16af03663682ec57993122c22c667813afc9, signature verified good for oparin@me.com. Working tree clean; the branch is three commits past main, one per leaf. Compare candidate_tree_oid against 67961efb before reading any stored CR patch, and review the commit itself if no rev7 was built for it.

REVIEWABLE DIFFS ATTACHED, because the stored CR patch predates this amend. TASK-260830-33sfxc_rework-r7-delta.patch is git diff --binary e708c4f..adca16a — 7 files, 1141 insertions, 7 deletions, the round-7 delta only and the shortest honest read for this round. TASK-260830-33sfxc_leaf-full-r7.patch is git diff --binary f34d91d..adca16a, the whole story_final leaf against its checkpoint.

WHAT TO RE-RUN RATHER THAN READ. TASK-260830-33sfxc_repro-r7.json is the F1 reproduction on the UNCHANGED reviewed tree — re-run it after reverting the round-7 tests if you want the SURVIVED result yourself. TASK-260830-33sfxc_mutants-r7.py takes an index slice (argv[1] argv[2]) so the 86 mutants can be run in batches inside a bounded shell; indices 63..86 are this round. Required to come back KILLED on re-review: read-exit-status-gate-restricted-to-the-four-sampled-values, with read-exit-status-gate-deleted still KILLED as the delete-only control; new-refusal-site-added-to-the-reader-without-an-inventory-row and inventory-row-deleted-from-the-published-table, which are the two directions of the inventory bijection; and schema-member-null-admitted-as-the-empty-schema, which restores the production defect this round fixed.

ONE THING I DID NOT DO, STATED RATHER THAN LEFT TO BE FOUND. The inventory derivation is exhaustive over internal/cliresult/client.go only. The sibling table is enumerated by hand and pinned only by resolution — the named function must be declared in its named file and the named tests must exist — so a guard added to internal/axerror/decode.go with no row is NOT a red. That is a real hole in the mechanism and the reason is scope, not contract: those files are leaves 1 and 2. If the reviewer judges the mechanism should cover them, that is a defensible ask and I did not pre-empt it.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260903-541bc0, pid=80873, exit=0)
REVIEW ROUND 7 — FINAL LEAF. Your acceptance makes this the Story's integration unit.

WHICH COMMIT. CR rev 7, 73-path delta. Verify reachability first.

ROUND 6's FINDING IS CLOSED, and the mechanism that would have caught it now exists.
- The Read-level ErrUnregisteredExitStatus gate — the reader-side twin swept at four points of a 240-value domain — is closed.
- The guard inventory I asked for twice now exists and is DERIVED, not written: TestTheRefusalGuardInventoryIsDerivedFromTheReaderSource parses every refusal site out of client.go and requires a bijection with a README table. Deleting a row reddens it with 'a guard without a row is a guard nobody traversed'. Verified by mutant, confirmed present before measuring.
- The inventory immediately found TWO more survivors the memory-based sweep had missed: the `schema` member type guard (restricted to JSON numbers, the one form its only row drove) now killed, and the `schema` member presence guard now carrying a stated bound.

ORCHESTRATOR NOTE ON METHOD, BECAUSE IT KEEPS HAPPENING TO ME
My first attempt at the inventory mutant deleted a row from the coverage-level LEGEND rather than from the guard table, and reported a false green. That is the fourth time this session I have believed a measurement whose target I had not verified — twice a -run filter that matched no test, once a substitution that never landed, once this. Confirm not only that your mutant is present, but that it is present IN THE THING YOU MEANT TO MUTATE.

WHAT TO ATTACK
1. THE BIJECTION'S OWN FAILURE MODES. It parses refusal sites out of one file. Confirm it fails closed when the README section is absent, renamed, or duplicated; when a guard is added in a file it does not parse; and when a row exists for a guard that does not. A one-directional check dressed as a bijection is the exact shape this inventory exists to prevent.
2. THE TWO NEWLY CLOSED SURVIVORS. Verify the `schema` type guard is now measured across its domain rather than at a second sampled point, and that the `schema` presence guard's stated bound has a CONTRACT reason rather than an effort reason.
3. IS THE INVENTORY COMPLETE? It claims a bijection with client.go. Guards in axerror that Read depends on are outside that file. Say whether the boundary is right and whether it is stated.
4. Confirm nothing from rounds 1-6 regressed, and that no survivor dismissed with a reason was quietly fixed.

STANDING REVIEW BAR
- Positive-path-only evidence is refused. Prove a bound by NARROWING across the domain, not by deleting.
- Confirm every mutant is PRESENT, in the intended target, before believing a green or a red.
- FINAL LEAF: accept with accept_cr if it holds — check your reviewer DoD items first, do not supply commit_ack. Eight rounds is not a reason to accept and not a reason to keep hunting. Round 6 established that every other reachable guard is measured or bounded; if that still holds and this round's items check out, say so plainly and accept.

spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Rank-1 and only admitted pair; round-7 review testing whether a derived guard bijection fails closed in every direction."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260903-8436b6, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260903-8436b6)
Round-7 review: ACCEPTED. 32 independent reviewer mutants across 7 batches against candidate tree 67961efb; 29 killed, 3 survivors all classified and none admitting anything: one equivalent ordering mutant, one subsumed by the canonical gate (probed empirically at exit 0), one relabel-only narrowing of the axerror major comparison that reports ErrUnsupportedVersion instead of ErrUnsupportedMajor for a major-0 envelope. That last one is a non-blocking observation on TASK-260830-34elja s guard, explicitly outside this leaf s stated inventory bound. README fan-in table and the derived guard inventory falsified seven ways and killed every time; tracecheck exit 1 on a renamed acceptance-case declaration. Validation re-run standalone: gofmt clean, build/vet exit 0, go test ./... -count=1 exit 0 across 13 packages, cover cliresult 95.5% / axerror 99.5%, tracecheck exit 0 at acceptance_cases=74 clauses_discharged=17/403. Tree restored bit-for-bit after all mutation work.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260903-8436b6, pid=21076, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260830-33sfxc_spawn-log_-implementer--developer--claude-_RUN-260903-8527e7.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_spawn-log_-implementer--developer--claude-_RUN-260903-8527e7.log) — System spawn log captured by task-board
- [TASK-260830-33sfxc_results.md](file://TASK-260830-33sfxc/TASK-260830-33sfxc_results.md) — Compatibility evidence: the deferred refusal-precedence decision, message/stderr/exit-status independence, frozen historical corpora, mutation and traceability results
- [TASK-260830-33sfxc_mutation-report.json](file://TASK-260830-33sfxc/TASK-260830-33sfxc_mutation-report.json) — 14-mutant harness report: 13 killed, 1 declared subsumed, 0 unexplained survivors, suite restored green
- [TASK-260830-33sfxc_mutants.py](file://TASK-260830-33sfxc/TASK-260830-33sfxc_mutants.py) — Mutation harness, 23 mutants: each verified applied, verified to carry the MUTATED text on disk, and compiled before measurement
- [TASK-260830-33sfxc_go-test-full.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_go-test-full.log) — go test ./... -count=1 exit 0 across 13 packages at commit 209e0f2
- [TASK-260830-33sfxc_go-test-cover.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_go-test-cover.log) — go test ./... -cover -count=1 exit 0 at commit 209e0f2; cliresult 95.5%, axerror 99.5%
- [TASK-260830-33sfxc_tracecheck.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_tracecheck.log) — tracecheck exit 0 at commit 209e0f2: acceptance_cases=74, clauses_discharged=17/403 unchanged
- [TASK-260830-33sfxc_spawn-log_-implementer--developer--claude-_RUN-260903-bcd7ed.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_spawn-log_-implementer--developer--claude-_RUN-260903-bcd7ed.log) — System spawn log captured by task-board
- [TASK-260830-33sfxc_spawn-log_-implementer--developer--claude-_RUN-260903-b931b2.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_spawn-log_-implementer--developer--claude-_RUN-260903-b931b2.log) — System spawn log captured by task-board
- [TASK-260830-33sfxc_publication-state.md](file://TASK-260830-33sfxc/TASK-260830-33sfxc_publication-state.md) — Publication-only run: verified commit shape, reconciled checkpoint_oid, signature, and re-run validation exit codes
- [TASK-260830-33sfxc_publication-run-go-test.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_publication-run-go-test.log) — go test ./... -count=1 exit 0, 13 packages, re-run in the publication session
- [TASK-260830-33sfxc_publication-run-tracecheck.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_publication-run-tracecheck.log) — tracecheck exit 0 re-run in the publication session: acceptance_cases=73, clauses_discharged=17/403
- [TASK-260830-33sfxc_change-request_rev1.patch](file://TASK-260830-33sfxc/TASK-260830-33sfxc_change-request_rev1.patch) — Change Request CR-TASK-260830-33sfxc-1 revision 1 candidate patch (repository_delta=present, 66 changed paths)
- [TASK-260830-33sfxc_change-request_rev1-validation.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_change-request_rev1-validation.log) — Change Request CR-TASK-260830-33sfxc-1 revision 1 bounded validation log
- [TASK-260830-33sfxc_spawn-log_-reviewer--reviewer--claude-_RUN-260903-29885b.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_spawn-log_-reviewer--reviewer--claude-_RUN-260903-29885b.log) — System spawn log captured by task-board
- [TASK-260830-33sfxc_review-verdict.md](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-verdict.md) — Reviewer verdict CR-TASK-260830-33sfxc-1 rev1: changes requested. 2 blocking findings (duplicate-keyed Structured Error admitted through Read; README/LOGBOOK fan-in largest-class wrong for 1.2.0 and 1.3.0), 1 nit, plus the story_final CR base explanation.
- [TASK-260830-33sfxc_review-mutation-rerun.json](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-mutation-rerun.json) — Reviewer's independent re-run of the producer mutation harness: 14 mutants, 13 killed, 1 subsumed, 0 unexplained survivors, suite restored green
- [TASK-260830-33sfxc_spawn-log_-implementer--developer--claude-_RUN-260903-d32271.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_spawn-log_-implementer--developer--claude-_RUN-260903-d32271.log) — System spawn log captured by task-board
- [TASK-260830-33sfxc_rework-mutation-report.json](file://TASK-260830-33sfxc/TASK-260830-33sfxc_rework-mutation-report.json) — 23-mutant harness at commit 209e0f2: 22 killed, 1 declared subsumed, 0 unexplained survivors, suite restored green
- [TASK-260830-33sfxc_fan-in-measured.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_fan-in-measured.log) — Independent per-status fan-in measurement for all four Structured Error versions, taken before any file was changed; the source of the two corrected README rows
- [TASK-260830-33sfxc_rework-results.md](file://TASK-260830-33sfxc/TASK-260830-33sfxc_rework-results.md) — Rework results at commit 209e0f2: F1 duplicate-member bypass, F2 fan-in derivation, F3 sentinel exclusion, a false citation caught in this rework's own draft, 23-mutant campaign, disclosures, and the CR candidate_tree_oid warning
- [TASK-260830-33sfxc_rework-delta.patch](file://TASK-260830-33sfxc/TASK-260830-33sfxc_rework-delta.patch) — git diff --binary b21fede..209e0f2 - the rework delta only: what changed since the reviewed rev1 head. 13 files, 1221 insertions, 38 deletions
- [TASK-260830-33sfxc_leaf-full.patch](file://TASK-260830-33sfxc/TASK-260830-33sfxc_leaf-full.patch) — git diff --binary f34d91d..209e0f2 - the whole leaf against its checkpoint, for a reviewer who wants the full story_final delta rather than only the rework
- [TASK-260830-33sfxc_change-request_rev2.patch](file://TASK-260830-33sfxc/TASK-260830-33sfxc_change-request_rev2.patch) — Change Request CR-TASK-260830-33sfxc-2 revision 2 candidate patch (repository_delta=present, 68 changed paths)
- [TASK-260830-33sfxc_change-request_rev2-validation.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_change-request_rev2-validation.log) — Change Request CR-TASK-260830-33sfxc-2 revision 2 bounded validation log
- [TASK-260830-33sfxc_spawn-log_-reviewer--reviewer--claude-_RUN-260903-00a6b8.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_spawn-log_-reviewer--reviewer--claude-_RUN-260903-00a6b8.log) — System spawn log captured by task-board
- [TASK-260830-33sfxc_review-verdict-rev2.md](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-verdict-rev2.md) — Reviewer verdict for CR revision 2: changes requested, two surviving narrowing mutants on the Section 14.2 outcome gates
- [TASK-260830-33sfxc_spawn-log_-implementer--developer--claude-_RUN-260903-dd2e44.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_spawn-log_-implementer--developer--claude-_RUN-260903-dd2e44.log) — System spawn log captured by task-board
- [TASK-260830-33sfxc_rework-r3-results.md](file://TASK-260830-33sfxc/TASK-260830-33sfxc_rework-r3-results.md) — Round-3 rework results at commit 1d68325: both surviving narrowing mutants reproduced then killed, 28-mutant harness, ownership-registration probe, validation exit codes
- [TASK-260830-33sfxc_mutation-report-r3.json](file://TASK-260830-33sfxc/TASK-260830-33sfxc_mutation-report-r3.json) — 28-mutant harness at commit 1d68325: 27 killed, 1 declared subsumed, 0 unexplained survivors, suite restored green
- [TASK-260830-33sfxc_mutants-r3.py](file://TASK-260830-33sfxc/TASK-260830-33sfxc_mutants-r3.py) — Round-3 mutation harness, 28 mutants; five new ones narrow the two Section 14.2 outcome guards and the shared failure-status enumerator
- [TASK-260830-33sfxc_repro-r3.py](file://TASK-260830-33sfxc/TASK-260830-33sfxc_repro-r3.py) — Pre-fix reproduction harness: both round-2 narrowing mutants applied to 209e0f2, verified present, and measured SURVIVED before any test was written
- [TASK-260830-33sfxc_ownership-pin-probe.json](file://TASK-260830-33sfxc/TASK-260830-33sfxc_ownership-pin-probe.json) — Probe result: the new acceptance-case registration resolves against the real Go declaration; renaming it fails tracecheck ahead of the digest check
- [TASK-260830-33sfxc_go-test-full-r3.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_go-test-full-r3.log) — go test ./... -count=1 exit 0 across 13 packages at commit 1d68325
- [TASK-260830-33sfxc_go-test-cover-r3.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_go-test-cover-r3.log) — go test ./... -cover -count=1 exit 0 at commit 1d68325; cliresult 95.5%, axerror 99.5%, both unchanged
- [TASK-260830-33sfxc_tracecheck-r3.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_tracecheck-r3.log) — tracecheck exit 0 at commit 1d68325: acceptance_cases=74, clauses_discharged=17/403, both unchanged
- [TASK-260830-33sfxc_rework-r3-delta.patch](file://TASK-260830-33sfxc/TASK-260830-33sfxc_rework-r3-delta.patch) — git diff --binary 209e0f2..1d68325 - the round-3 delta only: 5 files, 188 insertions, 7 deletions
- [TASK-260830-33sfxc_leaf-full-r3.patch](file://TASK-260830-33sfxc/TASK-260830-33sfxc_leaf-full-r3.patch) — git diff --binary f34d91d..1d68325 - the whole story_final leaf against its checkpoint at the post-amend head
- [TASK-260830-33sfxc_change-request_rev3.patch](file://TASK-260830-33sfxc/TASK-260830-33sfxc_change-request_rev3.patch) — Change Request CR-TASK-260830-33sfxc-3 revision 3 candidate patch (repository_delta=present, 68 changed paths)
- [TASK-260830-33sfxc_change-request_rev3-validation.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_change-request_rev3-validation.log) — Change Request CR-TASK-260830-33sfxc-3 revision 3 bounded validation log
- [TASK-260830-33sfxc_spawn-log_-reviewer--reviewer--claude-_RUN-260903-45ad1a.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_spawn-log_-reviewer--reviewer--claude-_RUN-260903-45ad1a.log) — System spawn log captured by task-board
- [TASK-260830-33sfxc_review-verdict-rev3.md](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-verdict-rev3.md) — Reviewer verdict for CR revision 3: changes requested; README acceptance-case figure 73 vs measured 74 and unpinned; 40 independent mutants, two observations
- [TASK-260830-33sfxc_review-mutation-independent.json](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-mutation-independent.json) — Independent reviewer mutation runs for CR rev3 (round 1: 15 mutants over client.go/registry.go/decode.go)
- [TASK-260830-33sfxc_spawn-log_-implementer--developer--claude-_RUN-260903-e342f8.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_spawn-log_-implementer--developer--claude-_RUN-260903-e342f8.log) — System spawn log captured by task-board
- [TASK-260830-33sfxc_mutation-report-r4.json](file://TASK-260830-33sfxc/TASK-260830-33sfxc_mutation-report-r4.json) — Round-4 mutation harness report: 44 mutants, 43 killed, 1 declared subsumed, 0 unexplained survivors, 27 narrowing, suite restored green
- [TASK-260830-33sfxc_mutants-r4.py](file://TASK-260830-33sfxc/TASK-260830-33sfxc_mutants-r4.py) — Round-4 mutation harness source with the explicit NARROWING class definition
- [TASK-260830-33sfxc_rework-r4-delta.patch](file://TASK-260830-33sfxc/TASK-260830-33sfxc_rework-r4-delta.patch) — git diff --binary 1d68325..b43a746 - the round-4 delta only, 6 files, 420 insertions, 4 deletions
- [TASK-260830-33sfxc_leaf-full-r4.patch](file://TASK-260830-33sfxc/TASK-260830-33sfxc_leaf-full-r4.patch) — git diff --binary f34d91d..b43a746 - the whole story_final leaf at signed head b43a746
- [TASK-260830-33sfxc_rework-r4-results.md](file://TASK-260830-33sfxc/TASK-260830-33sfxc_rework-r4-results.md) — Round-4 rework results: the blocking published-figure finding, two acted-on observations, the dismissed survivors left alone, mutants, traceability probes and validation exit codes
- [TASK-260830-33sfxc_go-test-full-r4.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_go-test-full-r4.log) — Round-4 validation log, exit 0
- [TASK-260830-33sfxc_go-test-cover-r4.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_go-test-cover-r4.log) — Round-4 validation log, exit 0
- [TASK-260830-33sfxc_tracecheck-r4.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_tracecheck-r4.log) — Round-4 validation log, exit 0
- [TASK-260830-33sfxc_change-request_rev4.patch](file://TASK-260830-33sfxc/TASK-260830-33sfxc_change-request_rev4.patch) — Change Request CR-TASK-260830-33sfxc-4 revision 4 candidate patch (repository_delta=present, 69 changed paths)
- [TASK-260830-33sfxc_change-request_rev4-validation.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_change-request_rev4-validation.log) — Change Request CR-TASK-260830-33sfxc-4 revision 4 bounded validation log
- [TASK-260830-33sfxc_spawn-log_-reviewer--reviewer--claude-_RUN-260903-b96555.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_spawn-log_-reviewer--reviewer--claude-_RUN-260903-b96555.log) — System spawn log captured by task-board
- [TASK-260830-33sfxc_review-verdict-rev4.md](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-verdict-rev4.md) — Review verdict for CR revision 4: changes requested; command-tag agreement in Read proven for one ordered pair, forged-takeover-document narrowing survives the full suite
- [TASK-260830-33sfxc_review-rev4-independent-mutation.json](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-rev4-independent-mutation.json) — Reviewer-written mutation batch 1 (8 mutants, 5 killed, 3 survivors) run against the candidate tree
- [TASK-260830-33sfxc_review-rev4-independent-mutation-2.json](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-rev4-independent-mutation-2.json) — Reviewer-written mutation batch 2: command-agreement narrowings, 1 killed control and 2 survivors
- [TASK-260830-33sfxc_review-rev4-mutants2.py](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-rev4-mutants2.py) — Reviewer mutation harness for the command-agreement narrowings; each mutant verified applied and compiled before measurement
- [TASK-260830-33sfxc_spawn-log_-implementer--developer--claude-_RUN-260903-98a051.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_spawn-log_-implementer--developer--claude-_RUN-260903-98a051.log) — System spawn log captured by task-board
- [TASK-260830-33sfxc_rework-r5-results.md](file://TASK-260830-33sfxc/TASK-260830-33sfxc_rework-r5-results.md) — Round-5 rework results at commit 1d0d702: the blocking command-agreement finding reproduced then killed, the guard's measured 306-of-1936 share, the payload-size hole extended to the sibling reader, 53-mutant campaign, traceability probes and validation exit codes
- [TASK-260830-33sfxc_repro-r5.json](file://TASK-260830-33sfxc/TASK-260830-33sfxc_repro-r5.json) — Pre-fix reproduction at reviewed head b43a746: both round-4 narrowings measured SURVIVED against go test ./... and the doctor-sparing control KILLED, before any test was written
- [TASK-260830-33sfxc_postfix-r5.json](file://TASK-260830-33sfxc/TASK-260830-33sfxc_postfix-r5.json) — The same three mutants re-measured after the sweep: all three KILLED against go test ./... -count=1
- [TASK-260830-33sfxc_repro-r5.py](file://TASK-260830-33sfxc/TASK-260830-33sfxc_repro-r5.py) — Reproduction harness for the two round-4 command-agreement narrowings; each mutant verified present on disk carrying the mutated text and compiled before measurement
- [TASK-260830-33sfxc_mutation-report-r5.json](file://TASK-260830-33sfxc/TASK-260830-33sfxc_mutation-report-r5.json) — 53-mutant harness at commit 1d0d702: 52 killed, 1 declared subsumed, 0 unexplained survivors, 36 narrowing, suite restored green
- [TASK-260830-33sfxc_mutants-r5.py](file://TASK-260830-33sfxc/TASK-260830-33sfxc_mutants-r5.py) — Round-5 mutation harness, 53 mutants; nine new rows cover the command-tag agreement in both directions, its two test enumerators, and the payload-size narrowing on both common-data-model gates
- [TASK-260830-33sfxc_ownership-pin-probe-r5.json](file://TASK-260830-33sfxc/TASK-260830-33sfxc_ownership-pin-probe-r5.json) — Probe: both new acceptance-case registrations resolve against real Go declarations; renaming each makes tracecheck name the missing declaration ahead of the projection digest
- [TASK-260830-33sfxc_go-test-full-r5.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_go-test-full-r5.log) — go test ./... -count=1 exit 0 across 13 packages at commit 1d0d702
- [TASK-260830-33sfxc_go-test-cover-r5.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_go-test-cover-r5.log) — go test ./... -cover -count=1 exit 0 at commit 1d0d702; cliresult 95.5%, axerror 99.5%, both unchanged
- [TASK-260830-33sfxc_tracecheck-r5.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_tracecheck-r5.log) — tracecheck exit 0 at commit 1d0d702: acceptance_cases=74, clauses_discharged=17/403, both unchanged
- [TASK-260830-33sfxc_rework-r5-delta.patch](file://TASK-260830-33sfxc/TASK-260830-33sfxc_rework-r5-delta.patch) — git diff --binary b43a746..1d0d702 - the round-5 delta only: 7 files, 385 insertions, 12 deletions
- [TASK-260830-33sfxc_leaf-full-r5.patch](file://TASK-260830-33sfxc/TASK-260830-33sfxc_leaf-full-r5.patch) — git diff --binary f34d91d..1d0d702 - the whole story_final leaf against its checkpoint at the post-amend head
- [TASK-260830-33sfxc_change-request_rev5.patch](file://TASK-260830-33sfxc/TASK-260830-33sfxc_change-request_rev5.patch) — Change Request CR-TASK-260830-33sfxc-5 revision 5 candidate patch (repository_delta=present, 69 changed paths)
- [TASK-260830-33sfxc_change-request_rev5-validation.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_change-request_rev5-validation.log) — Change Request CR-TASK-260830-33sfxc-5 revision 5 bounded validation log
- [TASK-260830-33sfxc_spawn-log_-reviewer--reviewer--claude-_RUN-260903-1a92ce.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_spawn-log_-reviewer--reviewer--claude-_RUN-260903-1a92ce.log) — System spawn log captured by task-board
- [TASK-260830-33sfxc_review-verdict-rev5.md](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-verdict-rev5.md) — Review verdict for CR revision 5: changes requested; the code-to-exit-status agreement guard in axerror.decodeBody has delete-only coverage (1 of 752 pairs) and a forged retry claim on the exit-7 authorization class reaches through it via cliresult.Read. Round-4 command-agreement finding confirmed closed.
- [TASK-260830-33sfxc_review-rev5-independent-mutation.json](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-rev5-independent-mutation.json) — Reviewer-independent mutation evidence for CR rev5: 29 mutants across 5 batches, each verified present on disk with the MUTATED text and compiled before measurement; includes the round-4 survivor re-run (now KILLED) and the F1 narrowings
- [TASK-260830-33sfxc_review-rev5-mutants.py](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-rev5-mutants.py) — Reviewer mutation harness used for CR rev5; anchors are uniqueness-checked, mutation presence asserted on disk, compile gate before measurement, original restored and re-verified after each mutant
- [TASK-260830-33sfxc_review-rev5-go-test-full.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-rev5-go-test-full.log) — Reviewer re-run of go test ./... -count=1 at the candidate tree 42898ceb, exit 0, 13 packages
- [TASK-260830-33sfxc_review-rev5-tracecheck.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-rev5-tracecheck.log) — Reviewer re-run of tracecheck at the candidate tree, exit 0: acceptance_cases=74, clauses_discharged=17/403
- [TASK-260830-33sfxc_spawn-log_-implementer--developer--claude-_RUN-260903-6054b5.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_spawn-log_-implementer--developer--claude-_RUN-260903-6054b5.log) — System spawn log captured by task-board
- [TASK-260830-33sfxc_rework-r6-results.md](file://TASK-260830-33sfxc/TASK-260830-33sfxc_rework-r6-results.md) — Round-6 rework results: the F1 code-to-exit-status agreement closed as a measured 5,372-row sweep, the retry-claim bypass closed at cliresult.Read, and the 0..255 exit-status sweep; every gate exit code stated
- [TASK-260830-33sfxc_mutation-report-r6.json](file://TASK-260830-33sfxc/TASK-260830-33sfxc_mutation-report-r6.json) — 63-mutant harness at commit e708c4f: 62 killed, 1 declared subsumed, 0 unexplained survivors, 44 narrowing, suite restored green
- [TASK-260830-33sfxc_mutants-r6.py](file://TASK-260830-33sfxc/TASK-260830-33sfxc_mutants-r6.py) — Round-6 mutation harness: anchors uniqueness-checked, MUTATED text asserted on disk, compile gate before measurement, original restored after each
- [TASK-260830-33sfxc_go-test-full-r6.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_go-test-full-r6.log) — go test ./... -count=1 exit 0 across 13 packages at commit e708c4f
- [TASK-260830-33sfxc_go-test-cover-r6.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_go-test-cover-r6.log) — go test ./... -cover -count=1 exit 0 at commit e708c4f; cliresult 95.5%, axerror 99.5%, both unchanged
- [TASK-260830-33sfxc_tracecheck-r6.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_tracecheck-r6.log) — tracecheck exit 0 at commit e708c4f: acceptance_cases=74, clauses_discharged=17/403, both unchanged
- [TASK-260830-33sfxc_rework-r6-delta.patch](file://TASK-260830-33sfxc/TASK-260830-33sfxc_rework-r6-delta.patch) — git diff --binary 1d0d702..e708c4f - the round-6 delta only: 6 files, 596 insertions, 7 deletions, no production source changed
- [TASK-260830-33sfxc_leaf-full-r6.patch](file://TASK-260830-33sfxc/TASK-260830-33sfxc_leaf-full-r6.patch) — git diff --binary f34d91d..e708c4f - the whole story_final leaf against its checkpoint at the post-amend head
- [TASK-260830-33sfxc_change-request_rev6.patch](file://TASK-260830-33sfxc/TASK-260830-33sfxc_change-request_rev6.patch) — Change Request CR-TASK-260830-33sfxc-6 revision 6 candidate patch (repository_delta=present, 71 changed paths)
- [TASK-260830-33sfxc_change-request_rev6-validation.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_change-request_rev6-validation.log) — Change Request CR-TASK-260830-33sfxc-6 revision 6 bounded validation log
- [TASK-260830-33sfxc_spawn-log_-reviewer--reviewer--claude-_RUN-260903-250f8d.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_spawn-log_-reviewer--reviewer--claude-_RUN-260903-250f8d.log) — System spawn log captured by task-board
- [TASK-260830-33sfxc_review-verdict-rev6.md](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-verdict-rev6.md) — Round-6 reviewer verdict: CHANGES REQUESTED, one blocking finding on the Read-level unregistered-exit-status gate, plus four classified observations
- [TASK-260830-33sfxc_review-rev6-mutation.json](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-rev6-mutation.json) — 31 independent reviewer mutants measured against go test ./... -count=1; 21 killed, 10 survivors all classified
- [TASK-260830-33sfxc_review-rev6-mutants.py](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-rev6-mutants.py) — Reviewer mutation harness: applies, verifies present on disk, compiles, measures, restores
- [TASK-260830-33sfxc_review-rev6-go-test.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-rev6-go-test.log) — go test ./... -count=1 at the candidate tree, exit 0 across 13 packages
- [TASK-260830-33sfxc_spawn-log_-implementer--developer--claude-_RUN-260903-541bc0.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_spawn-log_-implementer--developer--claude-_RUN-260903-541bc0.log) — System spawn log captured by task-board
- [TASK-260830-33sfxc_rework-r7-results.md](file://TASK-260830-33sfxc/TASK-260830-33sfxc_rework-r7-results.md) — Round-7 rework results: blocking F1 closed, derived guard inventory, three new class instances found by the traversal, one production fix, validation table
- [TASK-260830-33sfxc_mutation-report-r7.json](file://TASK-260830-33sfxc/TASK-260830-33sfxc_mutation-report-r7.json) — 86-mutant harness: 85 killed, 1 declared subsumed, 0 unexplained survivors, 64 narrowing; three batches each restoring a green suite
- [TASK-260830-33sfxc_mutants-r7.py](file://TASK-260830-33sfxc/TASK-260830-33sfxc_mutants-r7.py) — Round-7 mutation harness source, 86 declared mutants with the narrowing class defined in the harness
- [TASK-260830-33sfxc_repro-r7.json](file://TASK-260830-33sfxc/TASK-260830-33sfxc_repro-r7.json) — F1 reproduction on the UNCHANGED reviewed tree: two narrowings SURVIVED, delete-only control KILLED, tree restored green
- [TASK-260830-33sfxc_traverse-r7.json](file://TASK-260830-33sfxc/TASK-260830-33sfxc_traverse-r7.json) — Inventory traversal probes: three further client.go guards SURVIVED narrowing before this round's tests existed
- [TASK-260830-33sfxc_go-test-r7.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_go-test-r7.log) — go test ./... -count=1 on the amended tree, exit 0 across 13 packages
- [TASK-260830-33sfxc_rework-r7-delta.patch](file://TASK-260830-33sfxc/TASK-260830-33sfxc_rework-r7-delta.patch) — git diff --binary e708c4f..adca16a — the round-7 delta only, 7 files, 1141 insertions, 7 deletions; the shortest honest read for this round
- [TASK-260830-33sfxc_leaf-full-r7.patch](file://TASK-260830-33sfxc/TASK-260830-33sfxc_leaf-full-r7.patch) — git diff --binary f34d91d..adca16a — the whole story_final leaf against its checkpoint
- [TASK-260830-33sfxc_change-request_rev7.patch](file://TASK-260830-33sfxc/TASK-260830-33sfxc_change-request_rev7.patch) — Change Request CR-TASK-260830-33sfxc-7 revision 7 candidate patch (repository_delta=present, 73 changed paths)
- [TASK-260830-33sfxc_change-request_rev7-validation.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_change-request_rev7-validation.log) — Change Request CR-TASK-260830-33sfxc-7 revision 7 bounded validation log
- [TASK-260830-33sfxc_spawn-log_-reviewer--reviewer--claude-_RUN-260903-8436b6.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_spawn-log_-reviewer--reviewer--claude-_RUN-260903-8436b6.log) — System spawn log captured by task-board
- [TASK-260830-33sfxc_review-verdict-rev7.md](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-verdict-rev7.md) — Review verdict for CR revision 7: ACCEPTED. 32 independent reviewer mutants, 29 killed, 3 survivors all classified (1 equivalent, 1 subsumed and probed, 1 relabel-only observation on a sibling leaf's guard); README fan-in and guard-inventory pins falsified 7 ways and all killed; validation re-run standalone on the candidate tree
- [TASK-260830-33sfxc_review-rev7-mutants-round1.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-rev7-mutants-round1.log) — Reviewer mutation batch 1: data-model gates and outcome corroboration, 6 mutants, all killed
- [TASK-260830-33sfxc_review-rev7-mutants-round2.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-rev7-mutants-round2.log) — Reviewer mutation batch 2 against candidate tree 67961efb
- [TASK-260830-33sfxc_review-rev7-mutants-round3.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-rev7-mutants-round3.log) — Reviewer mutation batch 3 against candidate tree 67961efb
- [TASK-260830-33sfxc_review-rev7-mutants-round5.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-rev7-mutants-round5.log) — Reviewer mutation batch 5 against candidate tree 67961efb
- [TASK-260830-33sfxc_review-rev7-mutants-round6.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-rev7-mutants-round6.log) — Reviewer mutation batch 6 against candidate tree 67961efb
- [TASK-260830-33sfxc_review-rev7-mutants-round7.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-rev7-mutants-round7.log) — Reviewer mutation batch 7 against candidate tree 67961efb
- [TASK-260830-33sfxc_review-rev7-tracecheck-probe.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-rev7-tracecheck-probe.log) — Reviewer probe: renaming a registered acceptance-case declaration makes tracecheck exit 1 naming the absent declaration
- [TASK-260830-33sfxc_review-rev7-go-test-cover.log](file://TASK-260830-33sfxc/TASK-260830-33sfxc_review-rev7-go-test-cover.log) — Reviewer re-run on the restored candidate tree: go test ./... -cover -count=1 exit 0, cliresult 95.5%, axerror 99.5%

## Created
2026-08-29T21:59:58Z

## Last Update
2026-09-03T12:12:33Z

## Assigned To
[reviewer] reviewer (claude)
