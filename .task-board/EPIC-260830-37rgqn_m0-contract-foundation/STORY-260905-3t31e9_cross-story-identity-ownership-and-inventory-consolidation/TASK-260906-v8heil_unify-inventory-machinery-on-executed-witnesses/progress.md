## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(13))

## Blocked By
- TASK-260906-vmzk0y

## Blocks
- TASK-260906-2okwyf

## Checklist
- [x] One shared test-support core owns production-file selection, fail-closed parsing, and the both-direction check harness; every existing inventory is expressed through it
- [x] terminalbackend refusal arms are resolved by executed witnesses that refuse at the production entry point, not by source text mentioning a detail string
- [x] Alias-bypass coverage is preserved for every package that had it, proven by a negative test per bypass shape that fails when the core drops that direction
- [x] The core fails closed on an unregistered site, an orphan row and a site it cannot classify, each control-planted including an import alias and a var binding
- [x] Arm counts before and after are reported per package with any change accounted for; no package regresses its floor
- [x] The digit-guard census stated bound is completed or widened: code-point, named-rune and strconv spellings are enumerated or explicitly named as outside the classifier
- [x] Mutation battery reports killed over applied on a production-derived denominator with narrowing, arm-deletion and census-only separate, and NOT_APPLIED and COMPILE_FAIL as distinct rows
- [x] Per-arm behavioural-versus-census kill split is reported, with the behavioural mask verified non-empty
- [x] Candidate tree OID in the outcome document equals the one the Change Request record carries, verified by listing the tree
- [x] Code written per task description and AC
- [x] Relevant tests written for new or changed behavior and passing
- [x] Every command, message, state, or refusal named in the AC is driven through the production entry point by a named committed test, or is declared a stated bound. Report coverage as a ratio — `n of m AC rows driven` — and name the production call site for each. Prose in place of the ratio is not evidence.
- [x] Gating, refusing, validating, authorizing, or attesting behavior covered by negative tests that fail when the gate admits what it must reject, with the production call site named
- [x] Every gate ships at least one NARROWING mutant — the gate stays present and is weakened to admit exactly one member of the class it must reject, and a named test must fail. A delete-only mutant proves only that the gate exists and is not accepted as evidence.
- [x] A gate that inspects source text is additionally attacked by a mutant that PRESERVES the searched-for token and changes behavior, and the mutant harness executes the behavioral suite, not only the static checker.
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
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:08c5d671a0c9e2350b1b8c97c08841f378392d0f4ac8edf05ecf155d29ae97bb rationale="implementation class rank 1; third convergence leaf, the shared inventory core leaf 2 spent six rounds justifying"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260907-396326, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260907-396326)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260907-396326, pid=15341, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:08c5d671a0c9e2350b1b8c97c08841f378392d0f4ac8edf05ecf155d29ae97bb rationale="implementation class; publish the Change Request the previous run handed off without constructing"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260907-2ee023, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260907-2ee023)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260907-2ee023, pid=85648, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/xhigh pair_source=explicit match=not_recommended snapshot=sha256:9f20871155c37dee69e09bc65874033bce16ac39424aba056c24ea8521516f40 rationale="review class rank 1; the shared inventory core — judge whether it keeps the union of three directions"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260907-e3b260, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260907-e3b260)
reviewer verdict rev2 (RUN-260907-e3b260): changes_requested -> to-dev. repeat-of: none.

BLOCKING F1 - terminalbackend executed witnesses assert (code,detail), not the arm. Plant P8 in conformance.go CheckEntrypoint (`if err != nil` -> `if err != nil || sessionID != argv[2]`) makes sites :713 and :716 dead by construction; derived arm set byte-identical, bijection 210/210, all three witnesses PASS through sibling #1, `go test ./internal/terminalbackend/ -count=1` = ok. Contradicts refusal_arm_inventory_test.go:24-27 and outcome section 2. Measured residual: 62 of 184 witnessed arms share a (code,detail) pair with a sibling (20 distinct clauses, largest x11); 19 of those share (file,function). Deletion IS caught (plant P4 reddens); only shadowing is invisible.

BLOCKING F2 - battery row C-tb-dead-arm does not measure the dead-arm class. Its mutant adds a NEW function with a NEW detail (deadArmProbe/"dead arm probe"), killed by the declared<->derived bijection - same shape as an added-undeclared-arm mutant (reproduced as plant P3). provhost/provider kill their equivalents via the derived-site-without-exercised-path direction over runtime file:line (invcore.SiteRecorder); terminalbackend has no analogue - it derives (file,function,code,detail,occurrence) with no line and no runtime recording. boundUnreachableNumbers claims the residual is measured by the arm-deletion mutants; it is not.

F3 - digit_guard_census_test.go:63-67 states no strconv-delegated site exists in either package. internal/provhost/opdecode.go:67-86 (rawUint53) is exactly that shape. Not an open hole - plant P10 (parsed > maxUint53+1) reddens TestDecodeQuiesceRefusals/count_overflow - but the absence claim is false.

F4 - TestShadowedLookupsHaveNoInput hand-lists 19 pairs with no bijection to the 9 boundShadowedLookup rows.
F5 - invcore.go:369-379 dead loop (_ = hasLocal); ParseSource has zero callers.
F6 - outcome claims zero non-test files; invcore.go and must.go are non-test.
Scope - measured 12 of 50 walker files / 3 of 13 packages ported; narrowing accepted but must be reported as a ratio, not as the review count being stale.

PASSED, measured: G-A union kept - alias plants P1 (var aliasedMismatch = mismatchf) and P2 (import errs "errors" + var mintPlain = errs.New) both reddened against REAL production, plus unregistered-site P3 and orphan-row P4. G-C Errorf* pin committed and its narrowing mutant killed; ran-counts present on every mask. G-D counts 210/210 tb, 167/167 ph, 18/18 pv, no floor regression; base verified 210 textual rows at 5da63ad. G-F tree recomputed dd555ca4 = record, new files in tree, HEAD at base, no production file touched, tree re-verified intact after every plant.

Gates rerun this turn: go vet clean, gofmt empty, go build clean, go test ./... -count=1 exit 0 (23 ok, 0 FAIL). Verdict artifact: TASK-260906-v8heil_review-verdict-rev2.md
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260907-e3b260, pid=94151, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:08c5d671a0c9e2350b1b8c97c08841f378392d0f4ac8edf05ecf155d29ae97bb rationale="implementation class rank 1; round 2, witnesses assert the pair not the arm and terminalbackend lacks runtime site recording"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260907-4a37af, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260907-4a37af)
Round-2 rework addresses review rev2 (RUN-260907-e3b260) branch-by-branch: F1/F2 via the refuse/site-audit seam (S-tb-shadow audit-only kill naming :713/:716), F3 via corrected N-R1 bound + B- row, F4 via derived cover analysis, F5 via dead-loop deletion + ParseSource callers, F6 via withdrawn zero-production-edit claim + disclosed seam, scope as 12/42 files and 3 refusal-arm packages with the 9 remaining named. Evidence: outcome-r2 + 8 battery/mutant resources. Re-review is the orchestrator step.
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260907-4a37af, pid=4226, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/xhigh pair_source=explicit match=not_recommended snapshot=sha256:9f20871155c37dee69e09bc65874033bce16ac39424aba056c24ea8521516f40 rationale="review class rank 1; round 2, the site audit catches P8 — judge whether the shadowing residual moved"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260907-39f8d1, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260907-39f8d1)
reviewer verdict rev3 (RUN-260907-39f8d1): ACCEPTED -> accept_cr. repeat-of: none.

G-A PASS. Residual re-derived independently: 210 sites, 24 shared (code,detail) clauses over 88 sites, 26 bound-exempt -> 62/184 witnessed. Audit-only kill declared as its own class (S-tb-shadow: census SURVIVED 2, behav SURVIVED 185, audit KILLED 657, killers=[audit]). Class generalizes: planted shadowing in the 12-way CodeMismatch/document-member-type clause (manifest.go parseClaim value/generation_variable), NOT CheckEntrypoint -> behav mask ok, census masks PASS, full run FAIL naming manifest.go:766.

G-B PASS + finding F-B1. strconv site exists at provhost/opdecode.go rawUint53; narrowing maxUint53 -> maxUint53+1 reddens exactly TestDecodeQuiesceRefusals/count_overflow with census green (reproduces battery row B-provhost-strconv-bound). Code-point spellings enumerated with a pin. F-B1: the named-rune bullet claims a named-constant gate would fail as unclassifiable, not pass silently - false and self-contradictory (same bullet says the chain prunes). Control plant: an additive pure named-const digit gate in provhost/protocol.go leaves both packages ok. The absence it justifies is independently TRUE (AST scan: 0 digit-valued named consts, 0 comparisons). Rewriting an existing rowed gate IS caught (orphan row). One sentence to delete.

G-C PASS. F4 cover analysis is derived (11 misses, both-direction join to derived arms, recursive caller cover, covered<->bound and uncovered<->witnessed, covered==boundRows); neutering checkExactMembers in ParseAttachAuthorization reddens with covers 8 want 9. F5 dead loop gone, ParseSource has 3 callers. F6 disclosure accurate. Scope 42 base -> 30 now = 12 ported reproduced; 3 of 12 packages, 9 named; 50->42 is a metric change (go/ast union go/parser = 50) stated as method but not reconciled to the prior number.

G-D PASS. 19 KILLED / 19 applied / 0 SURVIVED, NOT_APPLIED + COMPILE_FAIL distinct, classes separate (narrowing 11, arm-deletion 3, census-only 3, token-preserving 1, shadow 1), every mask ran >= 1. Denominator re-derived by me: 210 tb + 167 ph + 18 pv = 395.

G-E PASS + finding F-E1. Tree 316d093b recomputed before first plant and after last revert, equal both times; ls-tree carries invcore and both new tb test files; no provhost/provider production file modified; floors unchanged (ph 166=166, tb 209 rows / 210 derived, pv 18=18). F-E1: outcome doc and logbook name tree 9d7eaa4d, record carries 316d093b; delta is exactly LOGBOOK.md +9 lines (the append written after the doc). G-E s no-production-file-outside-invcore clause is stale vs the accepted seam design; verified mechanically instead - 89/89 single-line hunks byte-identical to base modulo the refuse() wrapper, 0 unexplained, recordRefusal has no non-test assignment.

Beyond the brief. Smuggling into tb production: 5 shapes, 5 caught (raw &Error literal, package-level var literal, refuse alias called, refuse(non-literal), unused var alias of refuse -> proves refuse is in the REAL production watch list). Core narrowing mutants: 7 directions, 7 killed by a named test (import-path resolution, var binding of mismatchf, dot-import fail-closed, DiffSets orphan direction, zero-file floor, unparseable-skip, empty-derivation floor).

F-C1 (non-blocking, PRE-EXISTING, measured): an init-time constructor alias in provider production walks through. var aliasedFailInvalid = failInvalid + calling through it at provider.go:404 -> provider suite ok. The alias captures the value before TestMain swaps it AND the derivation drops the site (keys on ident name in direct-call position); provider has no site-count floor. Identical plant on a clean 5da63ad extract is also green, so nothing regressed - coverage is preserved, which is what the AC asks. provhost catches the same shape (ctor obligation orphans the witness); tb catches it (only tb runs invcore.AuditConstructorReferences over production). The defect is the evidence: outcome section 5 s provider row names two CORE tests over synthetic source as if they covered provider production. Closure = run AuditConstructorReferences over provider/provhost production; story-seam work, not this leaf s AC.

Gates re-run: build 0, vet 0, GOOS=windows vet 0, gofmt clean, go test ./... -count=1 all ok, -race on the 4 packages ok, cover 70.2/86.0/97.8/95.0 exact match, tracecheck ok, go generate no diff. CI ci.yml:115 runs go test ./... with no -run mask, so the TestMain audit does fire in CI.

Evidence: TASK-260906-v8heil_review-verdict-rev3.md
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260907-39f8d1, pid=46027, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:08c5d671a0c9e2350b1b8c97c08841f378392d0f4ac8edf05ecf155d29ae97bb rationale="implementation class; checkpoint-only run for the accepted non-final leaf"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260907-b78bbb, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260907-b78bbb)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260907-b78bbb, pid=66561, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260906-v8heil_spawn-log_-implementer--developer--muse-_RUN-260907-396326.log](file://TASK-260906-v8heil/TASK-260906-v8heil_spawn-log_-implementer--developer--muse-_RUN-260907-396326.log) — System spawn log captured by task-board
- [TASK-260906-v8heil_outcome.md](file://TASK-260906-v8heil/TASK-260906-v8heil_outcome.md) — Unify inventory machinery: census, battery, gates, tree OID
- [TASK-260906-v8heil_battery.json](file://TASK-260906-v8heil/TASK-260906-v8heil_battery.json) — 15/15 killed mutation battery with behavioural-vs-census split
- [TASK-260906-v8heil_change-request_rev1.patch](file://TASK-260906-v8heil/TASK-260906-v8heil_change-request_rev1.patch) — Change Request CR-TASK-260906-v8heil-1 revision 1 candidate patch (repository_delta=present, 17 changed paths)
- [TASK-260906-v8heil_change-request_rev1-validation.log](file://TASK-260906-v8heil/TASK-260906-v8heil_change-request_rev1-validation.log) — Change Request CR-TASK-260906-v8heil-1 revision 1 bounded validation log
- [TASK-260906-v8heil_spawn-log_-implementer--developer--muse-_RUN-260907-2ee023.log](file://TASK-260906-v8heil/TASK-260906-v8heil_spawn-log_-implementer--developer--muse-_RUN-260907-2ee023.log) — System spawn log captured by task-board
- [TASK-260906-v8heil_publication.md](file://TASK-260906-v8heil/TASK-260906-v8heil_publication.md) — Change Request publication report: rev 1 record fields and tree-OID verification
- [TASK-260906-v8heil_change-request_rev2.patch](file://TASK-260906-v8heil/TASK-260906-v8heil_change-request_rev2.patch) — Change Request CR-TASK-260906-v8heil-2 revision 2 candidate patch (repository_delta=present, 17 changed paths)
- [TASK-260906-v8heil_change-request_rev2-validation.log](file://TASK-260906-v8heil/TASK-260906-v8heil_change-request_rev2-validation.log) — Change Request CR-TASK-260906-v8heil-2 revision 2 bounded validation log
- [TASK-260906-v8heil_spawn-log_-reviewer--reviewer--claude-_RUN-260907-e3b260.log](file://TASK-260906-v8heil/TASK-260906-v8heil_spawn-log_-reviewer--reviewer--claude-_RUN-260907-e3b260.log) — System spawn log captured by task-board
- [TASK-260906-v8heil_review-verdict-rev2.md](file://TASK-260906-v8heil/TASK-260906-v8heil_review-verdict-rev2.md) — Reviewer verdict rev2: changes_requested — executed witnesses attribute the clause, not the arm (sibling-swallow survives green); C-tb-dead-arm measures a different class; digit-guard bound rests on a false absence claim
- [TASK-260906-v8heil_spawn-log_-implementer--developer--muse-_RUN-260907-4a37af.log](file://TASK-260906-v8heil/TASK-260906-v8heil_spawn-log_-implementer--developer--muse-_RUN-260907-4a37af.log) — System spawn log captured by task-board
- [TASK-260906-v8heil_outcome-r2.md](file://TASK-260906-v8heil/TASK-260906-v8heil_outcome-r2.md) — Round-2 outcome: F1-F6 fixes, site audit, 19/19 battery, tree 9d7eaa4d
- [TASK-260906-v8heil_battery-tb-r2.json](file://TASK-260906-v8heil/TASK-260906-v8heil_battery-tb-r2.json) — Round-2 tb battery raw records (8/8 incl audit-only S-tb-shadow)
- [TASK-260906-v8heil_muts-tb-r2.json](file://TASK-260906-v8heil/TASK-260906-v8heil_muts-tb-r2.json) — Round-2 tb mutant specs (replay with mutate2.py)
- [TASK-260906-v8heil_battery-ph-strconv.json](file://TASK-260906-v8heil/TASK-260906-v8heil_battery-ph-strconv.json) — F3 pin battery row B-provhost-strconv-bound
- [TASK-260906-v8heil_muts-ph-r2.json](file://TASK-260906-v8heil/TASK-260906-v8heil_muts-ph-r2.json) — F3 pin mutant spec
- [TASK-260906-v8heil_battery-core-r2.json](file://TASK-260906-v8heil/TASK-260906-v8heil_battery-core-r2.json) — Round-2 core battery re-run (4/4 on new tree)
- [TASK-260906-v8heil_battery-pv-r2.json](file://TASK-260906-v8heil/TASK-260906-v8heil_battery-pv-r2.json) — Round-2 provider battery re-run (3/3)
- [TASK-260906-v8heil_battery-ph-r2.json](file://TASK-260906-v8heil/TASK-260906-v8heil_battery-ph-r2.json) — Round-2 provhost battery re-run (3/3)
- [TASK-260906-v8heil_battery-x-r2.json](file://TASK-260906-v8heil/TASK-260906-v8heil_battery-x-r2.json) — Round-2 control rows (NOT_APPLIED + COMPILE_FAIL distinct)
- [TASK-260906-v8heil_change-request_rev3.patch](file://TASK-260906-v8heil/TASK-260906-v8heil_change-request_rev3.patch) — Change Request CR-TASK-260906-v8heil-3 revision 3 candidate patch (repository_delta=present, 22 changed paths)
- [TASK-260906-v8heil_change-request_rev3-validation.log](file://TASK-260906-v8heil/TASK-260906-v8heil_change-request_rev3-validation.log) — Change Request CR-TASK-260906-v8heil-3 revision 3 bounded validation log
- [TASK-260906-v8heil_spawn-log_-reviewer--reviewer--claude-_RUN-260907-39f8d1.log](file://TASK-260906-v8heil/TASK-260906-v8heil_spawn-log_-reviewer--reviewer--claude-_RUN-260907-39f8d1.log) — System spawn log captured by task-board
- [TASK-260906-v8heil_review-verdict-rev3.md](file://TASK-260906-v8heil/TASK-260906-v8heil_review-verdict-rev3.md) — Reviewer verdict rev3: ACCEPTED — shadow class killed in a different clause, audit-only kill declared, F3 bound reproduces, 7/7 core narrowing mutants killed, tree verified before/after plants; 3 non-blocking findings
- [TASK-260906-v8heil_spawn-log_-implementer--developer--muse-_RUN-260907-b78bbb.log](file://TASK-260906-v8heil/TASK-260906-v8heil_spawn-log_-implementer--developer--muse-_RUN-260907-b78bbb.log) — System spawn log captured by task-board
- [TASK-260906-v8heil_checkpoint-r3.md](file://TASK-260906-v8heil/TASK-260906-v8heil_checkpoint-r3.md) — Checkpoint-only run report: commit/tree/parent OIDs, verify status, CR state, leaf status, tree provenance

## Created
2026-09-06T07:19:18Z

## Last Update
2026-09-07T11:10:35Z

## Assigned To
[implementer] developer (muse)
