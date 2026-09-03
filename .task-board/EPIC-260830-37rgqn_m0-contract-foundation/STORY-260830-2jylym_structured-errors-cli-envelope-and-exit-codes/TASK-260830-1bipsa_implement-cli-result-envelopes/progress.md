## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(5))

## Blocked By
- TASK-260830-34elja

## Blocks
- TASK-260830-33sfxc

## Checklist
- [x] Production entry points implement the scoped deliverable: Implement CLI Result versions, JSON/JSONL serialization, human rendering boundaries, and exact exit-code mapping
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
LEAF 2 OF 3 — implement-cli-result-envelopes. Leaf 1 (structured-error-registry) is accepted and checkpointed; leaf 3 (compatibility) depends on you. Four other m0 Stories depend on this Story.

SCOPE
CLI Result versions, JSON/JSONL serialization, human rendering boundaries, and exact exit-code mapping, per §15 of the pinned AX v0.5.0 SPEC.

WHAT LEAF 1 LEFT YOU
internal/axerror exists: 1449 production lines, structured error versions, typed details, stable codes, retryability, causal redaction. Its ownership claims took the gate from 2/394 to 9/394 discharged clauses, with section:15.1 at 5/7 and section:15.3 at 2/3, both `partial` with honest gaps. Build on it; do not duplicate it.

TWO FINDINGS FROM LEAF 1'S REVIEW THAT ARE YOURS TO CLOSE

R2 — A MISCOUNT THAT IS NOW REPEATED BOARD-WIDE, INCLUDING IN LANDED main.
The Section 15.2 exit-code table at SPEC.md:11077-11094 has EIGHTEEN body rows, codes [0, 2-17, 130]. I counted them from the vendored digest-verified document. The repository calls it 'nineteen' in at least three places: internal/axerror/registry.go:64, README.md:1248, and README.md:1426 — the last of which is already landed in main and was written by the ownership-gate Story. The implementation and its reviewed literal in registry_test.go are both correct at 18 and are compared by length and content, so nothing is broken; only the prose is wrong.
Fix every occurrence you can find, including the landed one, and pin the count so prose cannot drift from the table again: derive the row count from internal/specdoc rather than writing a word. A number stated in prose beside a table that measures it is exactly the shape this board has spent the session removing.

R1 — A SENSITIVITY BOUND WITH NO TEST BEHIND IT.
minRedactableCause = 8 (internal/axerror/details.go:39) decides which causes get scanned for redaction at all. Raising it to 16 leaves the suite green; raising it to 64 leaves the suite green; only 4096 reddens. Every cause in the test corpus is at least 64 characters, so the constant's real value is unpinned, and an edit widening it to 63 would silently stop scanning a whole class of short causes.
Pin it with a cause SHORTER than the bound that must be scanned, and one at the boundary. This is the 'a bound proved only by deleting the gate' shape, and redaction is the wrong place to leave it.

THE ACCEPTANCE SIGNAL FOR THIS LEAF
tracecheck -section 15.2 exits 1 today: the section is bound to internal/catalog/catalog.go:ForRelease, an unrelated symbol, and nothing implements the exit-code registry — the only os.Exit calls in the tree are the exit(1) failure paths of cataloggen and tracecheck. Exit-code mapping is this leaf's scope, so §15.2 is yours to make honest.
Make it honest by ENUMERATION, not by re-binding: each clause you claim must index the measured inventory, declare its exact SPEC.md line, quote it verbatim beginning on that line, and name an acceptance case the binding owns. Re-binding a section to a friendlier symbol has appeared three times on this board and is reviewed for specifically.
Note the shape leaf 1's review caught: the gate counts a clause by the single RFC 2119 keyword on its line, and on one line that keyword bound the PROVIDER PLUGIN — an actor this repository does not build — while the host obligation carried no keyword at all. Before claiming a clause, name the actor its keyword governs and confirm this repository implements that actor.

STANDING CRITERIA
- Negative tests for every gate, refusal, validation and authorization path. A positive test proves reachability, not correctness. Prove a bound by NARROWING, not only by deleting.
- Validation that does not survive the package's own accessors is not validation — leaf 1 shipped exactly that and had to fix it. Whatever your envelopes guarantee at construction must still hold through every exported path.
- A gate reports coverage as a measured ratio, not as prose.
- A pin over a documented claim must exercise the claim's SUBJECT, not only its parameter.
- Never invent a constraint the pinned document does not declare. Use internal/specdoc to check your own citations rather than trusting a line number from any report.
- Confirm every mutant is PRESENT in the file before believing a green or a red, and make the presence check assert the MUTATED text.

DELIVERY
Non-final leaf: it will be checkpointed, not integrated. Leave the branch exactly one commit past the checkpoint and the worktree clean. Do not land anything on trunk while this Story is in flight.

spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 and only admitted pair; second leaf carrying two named residuals plus the exit-code registry the ownership gate measured as unimplemented."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260903-9058a8, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260903-9058a8)
READY FOR REVIEW — leaf 2 of 3, one signed commit f34d91d past the checkpoint ebc4e31 on task-board/story/STORY-260830-2jylym. Worktree clean. Not integrated; that is the orchestrator step.

DELIVERED
internal/cliresult (3,438 production lines, 3,098 test lines): the Section 14.2 CLI Result contract. Closed eight-member envelope (New, Decode, MarshalJSON); static per-tag version selection (VersionForCommand); eighteen closed command bodies and the six embedded types (validateBody); identifier nullability and session-scope equality (validateIdentifiers, validateNestedSessionScope); Section 17.2 reader rules (Decode rejects an unsupported major before reading anything else; acceptsVersion); the ten Section 14.2 common flags with its two stated refusals (ParseCommonFlags, AcceptsYes, AcceptsJSON); the destructive-operation confirmation gate (RequireConfirmation, ExpectationFlags); the stdout/stderr rendering boundary (Emitter); and the exact Section 15.2 process status (ExitStatus, Emit).

The writer and the reader share ONE validator over ONE value model: New marshals the caller body, runs it through the strict canonicaljson model, and re-parses it, so the writer structurally cannot emit an object its own Decode refuses. That is the CR-TASK-260830-34elja-1 defect removed by construction rather than by a test. TestEveryImplementedCommandRoundTripsThroughItsOwnReader drives all eighteen tags through it.

BOTH NAMED RESIDUALS CLOSED
R2 (nineteen vs eighteen): Section 15.2 has EIGHTEEN body rows [0, 2-17, 130]. Corrected in internal/axerror/registry.go, README.md x2, internal/traceability/traceability.go and traceability_test.go — including the occurrence already landed in main. The count is no longer written in prose: TestExitStatusRegistryMatchesThePinnedTableRowForRow reads the rows out of internal/specdoc through TableRowAt/SectionID and compares the count and every status against exitMeanings and a reviewed literal set.
R1 (minRedactableCause): pinned one character on either side of 8 — a cause of exactly 8 characters that must be scanned and one of 7 that must be skipped, on the message and on a diagnostic value, plus a multi-byte fixture a byte-counted bound would fail. Verified by hand: 7, 9 and 16 each redden.

SECTION 15.2 AND CLAUSE 14.2#6 — WHAT IS NOT CLAIMED
15.2 stays unmeasured. Its table carries no RFC 2119 keyword, so the clause scanner measures zero obligations and NO clause can be enumerated; enumeration was the instruction and it is not available for this section. The binding stays on internal/axerror/registry.go:ExitCodeFor (leaf 1 already moved it off catalog.ForRelease) and its gap now records that the mapping is implemented and measured against the pinned table, that internal/cliresult maps a failure to that exact status, and that what is missing is the PROCESS: no ax binary exists and the only os.Exit calls in the tree are the exit(1) paths of cataloggen and tracecheck.
For the same reason clause 14.2#6 is NOT claimed. Section 14.2 is bound at 8 of its 9 clauses.

MEASURED EVIDENCE (ratios, not prose)
bindings 48 -> 49; clauses_discharged 9/394 -> 17/403; acceptance_cases 53 -> 65; partial bindings 2 -> 3. Section 14.2 at 8/9 is partial, therefore still refused by assigned-scope admission, and is added to both refusal disclosure tables with its exact ratio. internal/cliresult coverage 95.3%; internal/axerror unchanged at 99.7%. Registered command tags built 18/44; registered versions built 2/4; user command surfaces 29/31.

STATED BOUNDS (cliresult.ContractBound, asserted by TestContractBoundStatesEveryLimitThisPackageActuallyHas)
CLI Result 3.0.0 and 4.0.0 registered and unbuilt, refused with ErrUnimplementedVersion rather than emitted with an unchecked body. The eight session.clone.* tags select 2.0.0 — that selection IS the Section 14.2 rule implemented here — but their Section 14.1 bodies over the Section 13.14 clone types are not built. The takeover adoption rule needs a session kind the body does not carry: New requires it, Decode cannot have it and exposes VerifyTakeoverAdoption rather than skipping a MUST silently. An absolute-path member is admitted when it is absolute on any supported platform because a CLI Result names none; VerifyDestinationPlatform narrows it. The Section 18.1 total order of a logs event array is not checked. Section 17.2 same-or-lower-minor is unit-tested as acceptsVersion but unreachable through Decode while no two registered CLI Result versions share a major.

VALIDATION — REAL EXIT CODES
go build ./... 0; go vet ./... 0; gofmt -l . 0 (no output); go test ./... -count=1 0 across 13 packages; go test ./internal/cliresult -count=1 -v 0 with 321 passing cases; go test ./... -cover -count=1 0; tracecheck 0 at 17/403; tracecheck -section 14.2 EXIT 1 — expected red, 8/9 is partial and admission requires full; mutants.py 0.

MUTATION: 50 mutants, each verified applied AND compiled before measurement, 48 of 48 non-subsumed killed. Two are declared SUBSUMED with the guard that refuses the same input earlier, each with a test pinning the invariant that makes it subsumed (capability count bound vs the seven-member vocabulary; the raw-identifier presence guard vs decodeClosedDocument). One mutant looked killed and was not — a clone tag inside a 1.0.0 envelope errors either way because the unimplemented-body message also contains the selected version — and the decisive case is a 1.0.0 takeover body inside a 2.0.0 envelope.

ARTIFACTS: TASK-260830-1bipsa_results.md, _mutation-01.log, _mutants.py, _tracecheck-01.log, _cliresult-test-01.log. Logbook entry 2026-09-03 1015.

NOT APPLICABLE: crash/idempotency evidence. This package holds no state, opens no file, starts no process, and writes only to streams a caller hands it, so it mutates no durable state and has no recovery surface to test.

FOR LEAF 3 (compatibility): the version registry, the static per-tag selection, acceptsVersion, and the Decode major/version refusals are the surfaces to attack. ContractBound names every limit this package actually has; the ratios above are the denominators.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260903-9058a8, pid=36357, exit=0)
REVIEWER CONTEXT — LEAF 2 OF 3, cli-result-envelopes

WHICH COMMIT. HEAD f34d91d. Verify reachability first. repository_delta=empty is a snapshot artifact — filed as #113; the real delta is the commit, 31 files +7241.

WHAT WAS BUILT. internal/cliresult, a new package. CLI Result versions, JSON/JSONL serialization, human rendering boundaries, exit-code mapping.

THE MEASURED RESULT
The ownership gate went from 9/394 to 17/403. The denominator grew because section:14.2, which had NO scoped implementation owner after leaf 1, is now bound and its nine clauses entered the inventory; eight of them are discharged, declared `partial` with one honest gap. bindings 48 -> 49, partial 2 -> 3.

ORCHESTRATOR VERIFICATION ALREADY PERFORMED — EXTEND, DO NOT REPEAT
- section:15.2 was NOT claimed. Its refusal names exactly what exists and what does not: the eighteen-row table is implemented and checked row-for-row against the pinned text, cliresult maps a failure to that exact process status through Emit and ExitStatus, and what is missing is the PROCESS — this repository builds no ax binary and the only os.Exit calls are the exit(1) paths of cataloggen and tracecheck. Claiming it would have been easy; it was refused.
- The R2 miscount is fixed at source. The two surviving occurrences of 'nineteen' are in comments that RECORD the error, and TestExitStatusRegistryMatchesThePinnedTableRowForRow now derives the row count and every status from internal/specdoc rather than restating them.

WHAT TO ATTACK, HARDEST FIRST
1. LEAF 1 SHIPPED VALIDATION THAT DID NOT SURVIVE ITS OWN ACCESSORS — cloneDetails was shallow and Detail handed out the live nested container, so every declared bound could be violated after construction. Leaf 2 is a NEW package with its own envelopes and its own accessors. Probe it the same way, in both directions: mutate what a constructor was given, and mutate what an accessor returns, then check what serialization emits. That defect class is the most likely thing to repeat here.
2. THE EIGHT NEWLY CLAIMED §14.2 CLAUSES. The gate cannot decide whether a named acceptance case exercises a clause's MEANING — that is its own stated residual, and it is why a reviewer reads them. For each, check the acceptance case drives a production entry point with a negative arm.
3. THE WRONG-ACTOR SHAPE. Leaf 1's review found a clause claimed against an obligation binding the provider plugin, an actor this repository does not build, because the line's only RFC 2119 keyword was the plugin's while the host obligation carried none. For each of the eight, name the actor its keyword governs and confirm this repository implements that actor.
4. R1 — minRedactableCause = 8 was unpinned: raising it to 16 or 64 left the suite green because every corpus cause was at least 64 characters. Confirm it is now pinned by a cause SHORTER than the bound and one at the boundary, not merely by a longer corpus.
5. THE DERIVED ROW COUNT. Confirm the count genuinely comes from specdoc and reddens if the pinned table changes — a derivation that caches a literal is the same defect wearing better clothes.

STANDING REVIEW BAR
- Positive-path-only evidence is refused. Prove a bound by NARROWING, not only by deleting.
- Confirm every mutant is PRESENT in the file before believing a green or a red, and make the presence check assert the MUTATED text.
- A gate reports coverage as a measured ratio; a pin must exercise its claim's SUBJECT, not only its parameter; 'nothing is there' is not 'nothing this checker sees'.
- Do not accept a claim in a comment, README or LOGBOOK you have not independently tested.
- NON-FINAL leaf: it will be checkpointed, not integrated.

spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Rank-1 and only admitted pair; new package likely to repeat leaf 1's aliasing defect, plus eight freshly claimed clauses the gate cannot semantically check."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260903-ac1cb0, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260903-ac1cb0)
REVIEW VERDICT: ACCEPTED (CR-TASK-260830-1bipsa-1 rev 1, reviewer RUN-260903-ac1cb0). Evidence: TASK-260830-1bipsa_review-verdict.md; independent attack suite TASK-260830-1bipsa_review-probe.md.

EMPTY DELTA IS A SNAPSHOT ARTIFACT (#113), NOT AN EMPTY PRODUCER. Base OID f34d91d IS the leaf commit, so base tree == candidate tree 4eed7417. Real delta is ebc4e31..f34d91d, 31 files +7241/-36. HEAD^{tree} still equals the candidate tree, worktree clean.

ATTACKED, NOT READ. Wrote an out-of-tree probe package importing internal/cliresult, internal/axerror and internal/specdoc and drove the real exported entry points.
- LEAF 1 ALIASING DEFECT DOES NOT REPEAT. Wrote through every caller container at every depth after New returned, through Body(), through Extension(), on both the New and Decode construction paths, and through IDs. Encoded output unchanged in all cases, no poison on the wire, fresh accessors clean.
- GATES HELD: --yes cannot bypass a dropped expectation flag (exit 2) in any position, full set without --yes non-interactively is exit 16, unrelated flags are not accepted as expectation flags, nil invocation refused; exactly 3 surfaces accept --yes and every other refuses it; rpc serve refuses --json in 4 spellings; second stdout document, human text in JSON mode, missing text rendering, both/neither outcome, and a prompt under --non-interactive all refused with nothing written; progress dropped on non-TTY and reports it; ~50 Decode wire mutations all refused; takeover adoption correct in all 8 kind x adopted x resumed combinations and unskippable by omitted or forged kind; clone/3.0.0/4.0.0 tags refused with ErrUnimplementedVersion, never emitted unchecked; exit status matches the registry for every registered code through ExitStatus AND real Emit, with the emitted exit_code member equal to the process status.

R2 CLOSED, DERIVATION IS REAL. Measured 18 body rows in section 15.2 from the pinned document myself. Mutated the production registry 4 ways, each verified present: delete row 17 RED, forged row 99 RED, RENUMBER 16->18 WITH COUNT UNCHANGED RED, successExit 0->2 RED. Not a count-only pin, not a cached literal.

R1 CLOSED, WITH ONE ACCURACY FINDING. minRedactableCause mutated to 7/9/16/63, all RED (previously 16 and 64 were green). FINDING: the pin works via the literal `!= 8` guard, not via the boundary fixtures - both fixtures are computed from the constant and slide with it. Proved it: removed the guard, widened to 63, suite went GREEN. The test comment claiming the two rows redden on a move is false about its own mechanism. Not rework - the residual purpose (a widening must not be silent) is met loudly. Fix the comment, and optionally hard-code the fixtures at 8 and 7, in leaf 3 or a follow-up.

CLAUSE CLAIMS VERIFIED AGAINST THE PINNED DOCUMENT, NOT THE OWNERSHIP FILE. All 8 excerpts appear verbatim at their declared SPEC.md lines, SectionID places each inside 14.2, each carries an RFC 2119 keyword, every named acceptance case is declared. WRONG-ACTOR CHECK: all 8 govern the ax CLI host; none binds the provider plugin or any actor this repo does not build. Leaf 1 shape does not repeat. Section 15.2 was NOT re-bound - still internal/axerror/registry.go:ExitCodeFor, byte-identical to ebc4e31, still unmeasured with an honest gap. 14.2#6 correctly refused: I confirmed internal/cliresult has ZERO importers outside its own package and the only os.Exit calls are cataloggen/tracecheck exit(1). The #3-vs-#6 line is coherent: #3 is a writer property over a caller-supplied stream, #6 needs process termination.

EVIDENCE REPRODUCED, NOT ACCEPTED ON REPORT: build/vet/gofmt clean; go test ./... 13/13 ok; tracecheck exit 0 at bindings=49 clauses_discharged=17/403 exact; -section 14.2 exit 1 at 8/9; mutation harness 48 killed / 2 subsumed / 0 BROKEN in 62s; coverage 95.3% and 99.7%; 44/18 tags, 4/2 versions, 31/29 surfaces all measured true. Harness is sound (unique anchor, asserts mutated text present, requires compile, restores in finally) and includes narrowing mutants, not delete-only. Both subsumption claims are pinned by the invariant that makes them subsumed.

FOR LEAF 3 (COMPATIBILITY), NOT A DEFECT: Decode runs the closed-member check BEFORE the major check, so a future-major document carrying a new top-level member reports "unknown top-level member" and errors.Is(ErrUnsupportedMajor)=false; a future-major document alone reports ErrUnsupportedMajor correctly. Both refuse, so no bypass. It matters because section 1.6 scopes its fail-closed unknown-member rule to "a major version 1 object", exactly the case where the major should already be settled. Section 17.2 numbers the major rejection first but states no explicit precedence, so the current order is defensible; leaf 3 owns the decision.

ENVIRONMENT DISCLOSURE: the Go build cache had reached 134GB against 150MiB free disk and the suite could not link (no space left on device). go clean -cache freed 135GB. Unrelated to this change, but worth host hygiene attention. All production mutations used file backups (never git checkout in a story worktree) and every one was restored; git status clean, HEAD^{tree}=4eed7417.

NEXT: non-final leaf, to be checkpointed not integrated. Orchestrator makes the done transition with commit_ack=scope_committed.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260903-ac1cb0, pid=6709, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260830-1bipsa_spawn-log_-implementer--developer--claude-_RUN-260903-9058a8.log](file://TASK-260830-1bipsa/TASK-260830-1bipsa_spawn-log_-implementer--developer--claude-_RUN-260903-9058a8.log) — System spawn log captured by task-board
- [TASK-260830-1bipsa_results.md](file://TASK-260830-1bipsa/TASK-260830-1bipsa_results.md) — CLI Result envelope implementation: scope, closed residuals R1/R2, measured evidence, stated bounds, and validation exit codes
- [TASK-260830-1bipsa_mutation-01.log](file://TASK-260830-1bipsa/TASK-260830-1bipsa_mutation-01.log) — 50-mutant harness over internal/cliresult gates: 48 of 48 non-subsumed killed, 2 declared subsumed
- [TASK-260830-1bipsa_mutants.py](file://TASK-260830-1bipsa/TASK-260830-1bipsa_mutants.py) — Reproducible mutation harness for internal/cliresult: verifies each mutant applied and compiled before measuring
- [TASK-260830-1bipsa_tracecheck-01.log](file://TASK-260830-1bipsa/TASK-260830-1bipsa_tracecheck-01.log) — tracecheck: repository gate exit 0 at 17/403 clauses; -section 14.2 exit 1 with its measured 8/9 partial ratio
- [TASK-260830-1bipsa_cliresult-test-01.log](file://TASK-260830-1bipsa/TASK-260830-1bipsa_cliresult-test-01.log) — go test ./internal/cliresult -count=1 -v, exit 0
- [TASK-260830-1bipsa_change-request_rev1.patch](file://TASK-260830-1bipsa/TASK-260830-1bipsa_change-request_rev1.patch) — Change Request CR-TASK-260830-1bipsa-1 revision 1 candidate patch (repository_delta=empty, 0 changed paths)
- [TASK-260830-1bipsa_change-request_rev1-validation.log](file://TASK-260830-1bipsa/TASK-260830-1bipsa_change-request_rev1-validation.log) — Change Request CR-TASK-260830-1bipsa-1 revision 1 bounded validation log
- [TASK-260830-1bipsa_spawn-log_-reviewer--reviewer--claude-_RUN-260903-ac1cb0.log](file://TASK-260830-1bipsa/TASK-260830-1bipsa_spawn-log_-reviewer--reviewer--claude-_RUN-260903-ac1cb0.log) — System spawn log captured by task-board
- [TASK-260830-1bipsa_review-verdict.md](file://TASK-260830-1bipsa/TASK-260830-1bipsa_review-verdict.md) — Reviewer verdict: accepted. Empty-CR artifact explained, aliasing/gate/decode/adoption attacks, R1+R2 mutation evidence, clause citations verified against the pinned document, producer evidence reproduced.
- [TASK-260830-1bipsa_review-probe.md](file://TASK-260830-1bipsa/TASK-260830-1bipsa_review-probe.md) — Reviewer-authored independent attack suite (out-of-tree, never committed): aliasing probes, gate/refusal negative arms, ~50 Decode wire mutations, takeover adoption matrix, exit-code mapping over every registered code, clause-citation verification against the pinned document, stated-ratio measurement.

## Created
2026-08-29T21:59:58Z

## Last Update
2026-09-03T05:43:24Z

## Assigned To
[reviewer] reviewer (claude)
