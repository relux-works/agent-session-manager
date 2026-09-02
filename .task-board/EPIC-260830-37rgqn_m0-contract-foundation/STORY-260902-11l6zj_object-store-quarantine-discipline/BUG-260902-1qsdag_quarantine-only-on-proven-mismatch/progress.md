## Status
blocked

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(3))

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] A read failure during inspection is reported as a durability error with the object left in place, never quarantined
- [x] Quarantine happens only on a proven digest mismatch or representation disagreement
- [x] storeOperations gains an injectable reader so the transient-failure path is driven by a test rather than argued
- [x] A negative case proves a transient read error leaves the object in place, and reddens when the classification is collapsed back
- [x] The projection and store paths agree on this classification, and the agreement is asserted rather than assumed
- [x] Repository tests, vet, build, tracecheck and catalog check exit 0 with no coverage regression
- [x] Code written per task description and AC
- [x] Relevant tests written for new or changed behavior and passing
- [x] Gating, refusing, validating, authorizing, or attesting behavior covered by negative tests that fail when the gate admits what it must reject, with the production call site named
- [x] Lint clean
- [x] Relevant build/validation commands run after changes and build not broken
- [x] New outcome artifact attached on the board with a task-scoped name when the work produces notes, logs, screenshots, or other deliverables
- [x] Important findings, decisions, anomalies, or regressions recorded in logbook when relevant

## Notes
FRONT-LOADED DoD from the Stories that landed today.

THE DEFECT, established by code reading rather than reproduced - the auditor did not build the injection because storeOperations exposes no injectable reader, which is itself part of the fix. internal/localstore/object_store.go:216: any non-nil existingErr that is not ErrUnsafeOwnership reaches quarantineExisting at :229, and inspectExisting returns raw os.Open, io.Copy and Close errors at :282-293. So EMFILE or a transient I/O error is indistinguishable from a hash mismatch at the decision point. SPEC.md:819-820 mandates quarantine only for a hash mismatch or representation disagreement; a read that did not complete proves neither.

THE REPOSITORY ALREADY CONTAINS THE CORRECT SHAPE. projection.go:704-774 classifies a read failure as sourceFailure and refuses rather than quarantining. Do not invent a third classification - make the store agree with the projection, and assert the agreement so the two cannot drift apart again.

1. NEVER INVENT A CONSTRAINT. Quote SPEC.md:819-820 verbatim; a quote not literally present at 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c is itself a defect.

2. A NEGATIVE CASE MUST BE REFUSED BY THE CLAUSE IT NAMES. Your transient-read case must reach the new classification, not an ownership check or an earlier guard. Prove the case fails against the current tree, the way the manifest-overlap work proved its subtests fail on base - for a behaviour that does not exist yet, that is the only evidence that the test detects it.

3. EVERY CLAUSE DIES UNDER A NARROWING MUTANT, not deletion only, or is documented as subsumed with the subsuming check named.

4. DRIVE THE REAL PRODUCTION ENTRY POINT, PutBlob, not the helper.

5. Quarantine is destructive and irreversible in effect. Prove the object is still in place after a transient failure by reading it back, not by asserting an error type.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 recommendation: stop a transient read failure from permanently quarantining a valid immutable object, and make the store agree with the projection path that already gets it right."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260902-164375, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260902-164375)
BLOCKED on a workspace-base decision. The fix is implemented and fully verified, but it is NOT committed to any branch.

WHY: internal/localstore does not exist on this Story workspace base. STORY-260902-11l6zj was forked from main (010d114); the package lives only on task-board/story/STORY-260830-2dy8oj, whose leaves are checkpointed and not integrated. Verified: git ls-tree -r --name-only HEAD | grep -i localstore returns nothing; git merge-base --is-ancestor task-board/story/STORY-260830-2dy8oj main -> NO. Committing the fix here would import ~4k lines of another Story unaccepted package into this branch and guarantee a divergent duplicate at integration; this worker is also forbidden from switching, rebasing or merging this branch.

WHAT IS DONE: existingVerdict classification in inspectExisting (absent/matches/disagrees/unsafe/unreadable); resolveExistingConflict as the single move-decision owner shared by the pre-install and raced create-new sites; storeOperations.openForRead injectable reader plus the PutBlob init guard. A read failure now returns ErrDurability and moves nothing; only a completed differing read quarantines.

TESTS: internal/localstore/object_store_quarantine_proof_test.go adds three tests driving PutBlob and openProjection. Mutant proof: fixed tree exit 0; delete-mutant (drop the existingUnreadable guard) exit 1 reproducing the original defect with ExistingQuarantinePath populated; narrowing mutant (existingDisagrees on read failure) exit 1.

GATES, all run in .temp/BUG-260902-1qsdag/scratch at base 81305c8, all exit 0: gofmt -l ., go vet ./..., go build ./..., go test ./internal/localstore -count=1 -cover (85.0%, baseline 83.8%), go test ./... -count=1 -cover, tracecheck global and -section 3.2/3.3/10.1/10.2/18.4, cataloggen -check. No coverage regression. Checklist items are checked against these runs, not against this branch.

ARTIFACTS: BUG-260902-1qsdag_results.md, BUG-260902-1qsdag_quarantine-classification.patch (git apply --check exit 0 and tests exit 0 against a fresh export of 81305c8), BUG-260902-1qsdag_mutant-evidence.log, BUG-260902-1qsdag_logbook-entry.md (fold into LOGBOOK.md when the fix lands).

DECISION NEEDED (orchestrator, pick one): (1) re-provision STORY-260902-11l6zj worktree forked from task-board/story/STORY-260830-2dy8oj instead of main and re-spawn -- RECOMMENDED, smallest move, patch applies as-is; (2) integrate STORY-260830-2dy8oj into main first, then re-provision and re-spawn; (3) reassign BUG-260902-1qsdag under STORY-260830-2dy8oj so it lands in the Story that owns the file.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-164375, pid=70834, exit=0)
No Change Request revision was published for BUG-260902-1qsdag (handoff_unsatisfied): the board is not at to-review

## Precondition Resources
(none)

## Outcome Resources
- [BUG-260902-1qsdag_spawn-log_-implementer--developer--claude-_RUN-260902-164375.log](file://BUG-260902-1qsdag/BUG-260902-1qsdag_spawn-log_-implementer--developer--claude-_RUN-260902-164375.log) — System spawn log captured by task-board
- [BUG-260902-1qsdag_results.md](file://BUG-260902-1qsdag/BUG-260902-1qsdag_results.md) — Verified fix, negative+mutant evidence, gate results, and the workspace-base blocker with the exact decision needed
- [BUG-260902-1qsdag_quarantine-classification.patch](file://BUG-260902-1qsdag/BUG-260902-1qsdag_quarantine-classification.patch) — Verified patch against task-board/story/STORY-260830-2dy8oj @ 81305c8; git apply --check exit 0, tests exit 0 after apply
- [BUG-260902-1qsdag_mutant-evidence.log](file://BUG-260902-1qsdag/BUG-260902-1qsdag_mutant-evidence.log) — Fixed tree exit 0; delete-mutant and narrowing-mutant both exit 1 on the three proof tests
- [BUG-260902-1qsdag_logbook-entry.md](file://BUG-260902-1qsdag/BUG-260902-1qsdag_logbook-entry.md) — LOGBOOK.md entry to fold in when the fix lands at a base containing internal/localstore

## Created
2026-09-02T11:59:24Z

## Last Update
2026-09-02T15:35:14Z

## Assigned To
[implementer] developer (claude)
