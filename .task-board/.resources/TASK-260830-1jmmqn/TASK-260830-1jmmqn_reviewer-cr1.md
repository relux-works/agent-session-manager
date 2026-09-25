Independently review Change Request revision 1 of TASK-260830-1jmmqn (implement-projection-plan-and-target-strategies). This is the first leaf of STORY-260830-21bxa3: task_delta on the Story branch (trunk base 0ca3e4c; checkpointed predecessors 547ea28 ; tip 547ea289e411bea803f4cd47b6314e4bb558ceb0), published by RUN-260923-71a3e2, 24 changed paths, patch and validation log attached as TASK-260830-1jmmqn_change-request_rev1.patch / _rev1-validation.log.. It is task_delta, so a registry edit is out of scope and counts as a finding. Read the producer brief `TASK-260830-1jmmqn_producer.md`, the results, the conformance matrix, the evidence tar, the CR patch and validation log, and the immutable CR bytes.

REVISION HISTORY: if this is not revision 1, every earlier verdict is attached as `TASK-260830-1jmmqn_review-verdict-rev<N>.md`. Read the one before yours and the brief that answered it, and use `repeat-of:` for the same class at an adjacent site. Attached verdicts outrank this brief.

SCRATCH RULE: put every `git archive` copy, probe and mutant workspace under `/Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/review-scratch/$TASK_BOARD_RUN_ID/` — never under `/tmp` or `/private/tmp` (on 2026-09-23 reviewer copies there filled the disk to 1 GB and broke three Change Request constructions) — and delete that directory after your evidence tar is attached. Exclude `.task-board` from archive copies unless a check needs it (run `specpin` in the live worktree instead).

LIVE-INDEX RULE: never run `git add`, `git add -N`, `git rm --cached`, `git reset` or any other command that writes the real index of the live Story worktree. Compare bytes through a scratch `GIT_INDEX_FILE`, and work on `git archive` copies.

NORMATIVE AUTHORITY: pinned `internal/specdoc/SPEC.v0.7.0.md` §13.14.2 (10548+): Projection Plan 1.0.0, its component schemas and Clone Projected Object Manifest 1.0.0. The leaf was split: planning semantics (strategy/profile choice, message authority, inactive tools/instructions, token metadata) are OUT of scope (TASK-260924-3n78rv) — implementing them here is a scope finding. The task record's v0.5.0 is stale.

NOT OPTIONAL — the rev1 reviewer marked the importer outcome grid `not-attacked: budget`; that is not acceptable for this row. Regardless of how the rest goes. State each result, and RERUN each yourself; do not accept producer evidence:
- the full `go test ./...` on the exact tree;
- `GOOS=windows GOARCH=amd64 go vet ./...`;
- the importer outcome comparison;
- a determinism rerun, `-count=3` on the new tests;
- a byte-exact recomputation of every normative fixture, using your own tiny independent JCS+SHA-256 script (Python is fine), not the candidate's code.

Verify with your own instruments:
(1) **Oracle independence and derived census.** The member census must be DERIVED from the Go types (reflection), not typed by hand; oracles must come from the spec text. Recompute at least two ids with your own tiny independent JCS+SHA-256 script.
(2) **Whole-domain sweeps.** Plant narrowings the producer did NOT ship: a case-insensitive member match, a DAG admitting an equal-sequence dependency, a nullability branch swap for one kind, a bound widened by one, a plan constant accepting a second value. Name each killer run ALONE.
(3) **Entries and axes.** Every production entry reaching each gate; a gate witnessed at one entry is a finding at the next. Check the axis inventory.
(4) **Composition and hygiene.** Importer outcome grid keyed `(package, entry, input)` rerun YOURSELF; shipped harness rerun, each KILLED twice; at least four own plants plus an applied control. gofmt/vet clean, Windows vet, `task-board.config.json` byte-identical to base, full configured suite green on the exact tree.

Report the ratio YOU measured. Attach `TASK-260830-1jmmqn_review-verdict-rev1.md` and `TASK-260830-1jmmqn_review-evidence-rev1.tar.gz` and complete the live checklist. Then either run `accept_cr(TASK-260830-1jmmqn, revision=1, evidence=TASK-260830-1jmmqn_review-verdict-rev1.md)`, or hand off changes requested with P1/P2/P3 findings and a "Rework scope (for the producer)" section. Model: gpt-6-sol medium. Canonical CLI: /Users/iv/.curator/global/bin/task-board.
