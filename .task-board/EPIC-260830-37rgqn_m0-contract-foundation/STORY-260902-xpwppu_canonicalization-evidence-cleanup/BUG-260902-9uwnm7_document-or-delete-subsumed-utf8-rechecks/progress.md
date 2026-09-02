## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(1))

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Branch state verified: HEAD is 8a0dced, parent is 422786c, worktree clean, diff is 5 files +732 -0
- [x] No file changed during this run; git status --porcelain empty at handoff
- [x] story_final Change Request published from the existing committed state
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
Delivered together with BUG-260902-1874eo in a single Change Request; the full brief for both parts is on that bug and on the parent Story STORY-260902-xpwppu. Part 2 (the four subsumed utf8.ValidString re-checks in internal/canonicaljson/closed_shapes.go at :258, :1506, :1525 and :1760) is this bug scope.
Delivered in commit 454c2db together with BUG-260902-1874eo, as the shared brief requires. Evidence attached as BUG-260902-9uwnm7_results.md; part 2 is the second half of that document.

Resolution: KEEP AND DOCUMENT, all eight sites, not four. The audit line numbers (closed_shapes.go:258/:1506/:1525/:1760) match commit 7b94c9ad, five commits back; current lines are 302/1830/1852/2091. That commit carries eight utf8.ValidString sites with no property separating the cited subset, so resolving only four would leave the clause sweep reporting four identical unexplained survivors. canonical.go:311 is now canonical.go:328, so the comments name decodeStrict by function rather than by line.

Kept rather than deleted because the subsumption is a package-internal invariant, not a property of these validators: CanonicalByteLength(value any) already shows the package can export a non-[]byte entry point, and two of the eight are registered as invalidUTF8Refusal in refusal_guards_test.go, so deleting rewrites pinned refusal statements. One shared note in closed_shapes.go carries the argument; each site names decodeStrict and states which sibling terms stay reachable, following the block-comment convention at internal/localstore/paths.go:234-236.

The AC requires the dependency to be stated explicitly. It is also machine-checked in new internal/canonicaljson/utf8_subsumption_test.go, with derived (not listed) guard and decode sites: no exported entry point reaching a re-check may take a non-[]byte parameter, and json.NewDecoder/json.Unmarshal may appear only inside decodeStrict. Four decisive mutants, all red: D2 exported map-taking entry point; G2 exported []byte entry point decoding with json.Unmarshal; E3 comment deleted; F3 narrowing, comment kept but naming requireExactMembers.

No test fakes reachability and no dead branch was made to look covered.

FOUND, NOT FIXED: core_records.go:278 in validateProviderIdentityRecord carries a ninth re-check of the same class, outside this bug closed_shapes.go scope. It is now the only undocumented survivor of this class in the package and deserves its own board item.
Delivered and reviewed inside the sibling BUG-260902-1874eo Change Request revision 2, as planned when the two were batched into one Story. Scope covered: all 8 utf8.ValidString re-checks in internal/canonicaljson/closed_shapes.go (lines 300, 347, 463, 1827, 1849, 1955, 2089, 2244 - a superset of the four the audit named, whose line numbers were stale and pointed at unrelated code) carry a subsumption comment naming decodeStrict, in the existing convention from internal/localstore/paths.go:234-236. The subsumption is machine-checked rather than asserted in prose: utf8_subsumption_test.go derives the guarded set from the AST and pins that no exported function or method may hand an already-decoded value to a re-check. Coverage is 7 of 7, asserted as a ratio. Round 1 rejected this part when the reviewer proved coverage was only 3 of 7; the rework closed it and the round-2 reviewer independently killed the surviving mutant H and eight others. The published bound is honest: a mutant built specifically to break it, dispatch through a func-typed struct field, survives exactly as the README and closed_shapes.go:57-61 declare it would.
PUBLICATION-ONLY TASK. DO NOT CHANGE ANY FILE.

The work for this bug and its sibling BUG-260902-1874eo is ALREADY COMPLETE, REVIEWED AND ACCEPTED. It is committed on the Story branch as a single commit, 8a0dced, exactly one direct single-parent commit past the checkpoint 422786c (origin/main). The worktree tree is clean.

WHY YOU ARE HERE. The Change Request published earlier was typed task_delta, not story_final, because at publication time this sibling bug was still open, so the board answered 'completing that leaf would not close the Story'. The CR kind is derived from the board and never declared. worktree integrate accepts only an accepted story_final Change Request, so the Story cannot be landed without one. Now that BUG-260902-1874eo is done, completing THIS leaf closes the Story, so your publication will be typed story_final. That is the entire purpose of this run.

YOUR TASK, IN FULL
1. Verify the state rather than trusting this brief:
   git log --oneline -1              # expect 8a0dced
   git rev-parse HEAD^               # expect 422786c
   git status --porcelain            # expect empty
   git diff --stat 422786c HEAD      # expect 5 files, +732, 0 deletions
2. Change NOTHING. Write no file, edit no file, add no test, fix no comment. If you believe something needs fixing, do not fix it - report it in your handoff and stop.
3. Check this bug's checklist items and hand off as developer so the story_final Change Request is published from the existing committed state.

WHAT THIS BUG'S SCOPE ALREADY IS, FOR YOUR HANDOFF EVIDENCE
All 8 utf8.ValidString re-checks in internal/canonicaljson/closed_shapes.go (lines 300, 347, 463, 1827, 1849, 1955, 2089, 2244) carry a subsumption comment naming decodeStrict, following the existing convention at internal/localstore/paths.go:234-236. The subsumption is machine-checked rather than asserted in prose: utf8_subsumption_test.go derives the guarded set from the AST and pins that no exported function or method may hand an already-decoded value to a re-check. Coverage is 7 of 7 and is asserted as a ratio.
Review history: round 1 rejected this part after proving coverage was only 3 of 7 and that README published a guarantee that did not reproduce. The rework strengthened the call graph and narrowed the published claim to a stated bound. The round-2 reviewer killed the surviving mutant H plus eight others, and confirmed the published bound is honest by building a mutant specifically to break it - dispatch through a func-typed struct field - which survives exactly as closed_shapes.go:57-61 and README declare it would.

DO NOT
- Do not amend, rebase, reset, or create a commit. The branch shape is already correct and a second commit past the checkpoint would break the leaf shape.
- Do not run mutants or re-verify the accepted work; it has been reviewed twice.
- Do not touch .task-board directly.

spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 and only admitted pair; publication-only run to emit the story_final Change Request from already-accepted committed state."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260902-018b16, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260902-018b16)
PUBLICATION RUN RUN-260902-018b16 (developer, publication-only).

No repository file was created, edited or deleted. Verified rather than trusted: HEAD 8a0dced, HEAD^ 422786c, git status --porcelain empty, git diff --shortstat 422786c HEAD = 5 files changed, 732 insertions(+), 0 deletions. Leaf shape intact - exactly one direct single-parent commit past the checkpoint. Porcelain re-checked after the full validation suite including all five fuzz targets and still empty, so the tree the validation observed is the tree being published.

All 18 configured spawn.worktree_isolation.validation.commands were run directly as standalone processes in this worktree, each exit 0: gofmt assert, go build, go vet, go test -count=1, -race, -cover, five fuzz targets at 100x, tracecheck, cataloggen -check, GOOS=linux and GOOS=windows builds, JSON validity over git ls-files, task-board validate, git diff --check. internal/canonicaljson coverage 97.2%, unchanged from the accepted revision. Targeted pin run go test ./internal/canonicaljson -run UTF8|Utf8|Subsum -v exit 0, with TestNoExportedEntryPointHandsADecodedValueToAUTF8Recheck, TestEveryUTF8RecheckIsCoveredByTheEntryPointPin and TestEveryUTF8RecheckDeclaresItsSubsumption all PASS.

NOT RE-RUN, STATED PLAINLY: the mutant campaign. The brief forbids re-verifying accepted work and running mutants requires editing files, which this run may not do. The negative-evidence claim rests on the round-1 and round-2 review records already on the board (D2, G2, E3, F3, H, I, I2, K, K2 plus the reviewer func-typed-struct-field mutant that survives exactly as the published bound declares), not on anything this run reproduced. This run establishes independently only that the pins are present, reachable and green at this exact tree.

CHECKLIST ITEM 3 ORDERING: handoff refuses while item 3 is unchecked, but the story_final Change Request is emitted by the run-completion publication path AFTER handoff, not by any command this run can issue. Item 3 is therefore checked as an ordering pre-declaration - the candidate is in the exact state publication requires - and not as an observation of a CR that already exists. If publication refuses, item 3 is false and must be reopened.

No logbook entry added: findings for this scope are already in LOGBOOK.md inside commit 8a0dced, and writing one now would move the candidate tree.

Carried forward, out of scope: core_records.go:278 in validateProviderIdentityRecord carries a ninth re-check of this class and is now the only undocumented survivor of the class in the package. It needs its own board item; fixing it here would violate the no-file-change contract.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-018b16, pid=52443, exit=0)
REVIEWER CONTEXT - NARROW SCOPE. READ THIS BEFORE PLANNING WORK.

This is a story_final Change Request whose CONTENT HAS ALREADY BEEN REVIEWED AND ACCEPTED TWICE. Do not re-review the substance. Your job is to confirm the republished candidate is byte-identical to what was accepted, and to accept it so the Story can be integrated.

WHY A SECOND PUBLICATION EXISTED AT ALL
The first Change Request was typed task_delta because this sibling bug was still open when the producer published, and the CR kind is derived from the board, never declared. worktree integrate accepts only an accepted story_final. A publication-only run republished the same candidate, changing no file, so the board would now derive story_final. No code changed between the accepted revision and this one.

WHAT TO VERIFY - THIS IS THE WHOLE JOB
1. Reachability first: git rev-parse HEAD should be 8a0dced; HEAD^ should be 422786c (origin/main); git status --porcelain empty. If the reviewed OID is not reachable from the branch head, refuse as void rather than proceeding - a reviewer on this Story's sibling reviewed an orphaned commit and produced a well-evidenced verdict about a tree that no longer existed.
2. The candidate is exactly one direct, single-parent commit past the checkpoint. Confirm git diff --shortstat 422786c HEAD reports 5 files, 732 insertions, 0 deletions.
3. Confirm the publication run changed nothing: the tree at 8a0dced must be the tree that was accepted. If any file differs from the accepted candidate, that is a blocking finding and you should say so plainly.
4. Run the repository validation suite and confirm it is green on this exact tree.
5. Accept the Change Request with accept_cr. Its checklist gate refuses until your reviewer DoD items are checked, so check them first. Do not supply commit_ack; the orchestrator makes the done transition.

WHAT NOT TO DO
- Do not re-run the mutant campaign. Both parts were attacked across two rounds: round 1 rejected the UTF-8 part for covering 3 of 7 guarded functions and publishing a guarantee that did not reproduce; the rework raised it to 7 of 7 and narrowed the claim; round 2 killed the surviving mutant H plus eight others and confirmed the published bound is honest by building a mutant specifically to break it (dispatch through a func-typed struct field), which survives exactly as documented.
- Do not request changes on the substance already accepted. If you believe an accepted decision was wrong, say so as a recorded non-blocking finding and accept, unless the candidate differs from what was accepted.
- Do not edit any file.

ACCEPTED SCOPE, FOR YOUR RECORD
Part 1: the two normative digest fixtures from SPEC.md:303-304 are recomputed from the production encoder and compared against values quoted from the pinned document, whose SHA-256 was verified against internal/specpin.DocumentSHA256. Each assertion is individually decisive.
Part 2: all 8 utf8.ValidString re-checks in closed_shapes.go carry a subsumption comment naming decodeStrict, in the existing convention from internal/localstore/paths.go:234-236, and the subsumption is machine-checked at 7 of 7 coverage asserted as a ratio.

spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Rank-1 and only admitted pair; narrow acceptance review confirming the republished story_final candidate equals the twice-accepted tree."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260902-bf5b7e, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260902-bf5b7e)
REVIEW ROUND 2 — ACCEPTED. Evidence: BUG-260902-9uwnm7_review-verdict.md.

Verified the reviewed bytes are the committed bytes (git diff --quiet 6b1134ed against HEAD 8a0dced succeeds); worktree porcelain empty before, between every mutant, and at handoff.

AC satisfied. All 8 utf8.ValidString sites in closed_shapes.go carry a comment naming decodeStrict, following the localstore block-comment convention, plus a shared note at :42-62 that publishes the bound of the machine check. The scope expansion from the audit-cited four to all eight is correct: the cited line numbers do not re-check UTF-8 on this tree and no property separates the subset.

Gate attacked, not read. 7 mutants applied and reverted by this review. KILLED: H (exported map-taking entry point reaching guards through the immutableObjectShapeValidators dispatch table, names all 7 guarded functions), I (same body as an exported method), F3 (narrowing — comment kept but names requireExactMembers), G2 (second json.Unmarshal decoder), M6 (new undocumented re-check in closed_shapes.go — killed by two tests, doc gate plus the ratio dropping to 7/8), M7 (new undocumented re-check as its own return in core_records.go — killed by the pre-existing refusal-guard sweep). SURVIVES BY PUBLISHED DESIGN: J2 (func-typed struct field), named as unmodelled in both closed_shapes.go and README.md. H and I are the two round-1 survivors; both now redden, so the function-value and method edges are load-bearing. F3 plus M6 show the doc gate is not delete-only.

Coverage ratio asserted rather than narrated (7/7, failing by name). Absence handled honestly: all four new tests t.Fatal on an empty derivation, and parsedProductionPackage fatals on a parse failure rather than reading it as an absence.

Sibling scope in the same commit: both published digests verified independently with shasum -a 256 over the literal canonical bytes, not recomputed from the encoder. bb80eb37... and b0ec84c6... both match SPEC.md:303-304 and the assertions.

Rerun by this review: gofmt clean, go build ./... exit 0, go vet ./... exit 0, go test ./... -count=1 all 10 packages ok.

FINDINGS, NOT BLOCKING — both are follow-up because the re-checks were KEPT fail-closed rather than deleted, so an entry point the graph misses still hits a real refusal.
(1) core_records.go:278 is a ninth re-check of the same class, still undocumented. The producer disclosed it in LOGBOOK as DISCLOSED/NOT FIXED and scoped it out as Section 2 rather than 1.6 — defensible. Reproduced: mutant M7b adds || !utf8.ValidString(text) as an extra CLAUSE (not a new return) in validateProviderIdentityRecord and the entire suite stays green, because TestEveryUTF8RecheckDeclaresItsSubsumption reads only closed_shapes.go while utf8GuardedFunctions derives package-wide. Follow-up: widen the doc test to the package and add the one comment.
(2) An exported package-level func-typed var is never enumerated as an entry point. Mutant J (var ValidateDecoded = func(map[string]any) error) hands a caller-supplied decoded map to a guard and survives green; exportedProductionEntryPoints walks only *ast.FuncDecl. Arguably already inside the published bound (a function value handed to another package), but worth naming beside the other three.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-bf5b7e, pid=65641, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [BUG-260902-9uwnm7_results.md](file://BUG-260902-9uwnm7/BUG-260902-9uwnm7_results.md) — Same delivery evidence; part 2 (the subsumed UTF-8 re-checks) is this bug's scope. Delivered in one change with BUG-260902-1874eo as the board note requires
- [BUG-260902-9uwnm7_spawn-log_-implementer--developer--claude-_RUN-260902-018b16.log](file://BUG-260902-9uwnm7/BUG-260902-9uwnm7_spawn-log_-implementer--developer--claude-_RUN-260902-018b16.log) — System spawn log captured by task-board
- [BUG-260902-9uwnm7_story-final-publication.md](file://BUG-260902-9uwnm7/BUG-260902-9uwnm7_story-final-publication.md) — Publication-only run RUN-260902-018b16: branch-state verification, all 18 configured validation commands with real exit codes, and the explicit statement of what was not re-run
- [BUG-260902-9uwnm7_change-request_rev1.patch](file://BUG-260902-9uwnm7/BUG-260902-9uwnm7_change-request_rev1.patch) — Change Request CR-BUG-260902-9uwnm7-1 revision 1 candidate patch (repository_delta=present, 5 changed paths)
- [BUG-260902-9uwnm7_change-request_rev1-validation.log](file://BUG-260902-9uwnm7/BUG-260902-9uwnm7_change-request_rev1-validation.log) — Change Request CR-BUG-260902-9uwnm7-1 revision 1 bounded validation log
- [BUG-260902-9uwnm7_spawn-log_-reviewer--reviewer--claude-_RUN-260902-bf5b7e.log](file://BUG-260902-9uwnm7/BUG-260902-9uwnm7_spawn-log_-reviewer--reviewer--claude-_RUN-260902-bf5b7e.log) — System spawn log captured by task-board
- [BUG-260902-9uwnm7_review-verdict.md](file://BUG-260902-9uwnm7/BUG-260902-9uwnm7_review-verdict.md) — Round-2 review verdict: accepted, with 7 mutants rerun locally and two disclosed follow-up findings

## Created
2026-09-02T12:00:23Z

## Last Update
2026-09-02T21:41:35Z

## Assigned To
[reviewer] reviewer (claude)
