Independently review Change Request revision 4 of TASK-260924-3n78rv (implement-projection-planning-and-authority-rules). This is the first leaf of STORY-260830-21bxa3: task_delta on the Story branch (trunk base 6d3bff9; checkpointed predecessors 618d78d 35ee723 ; tip 618d78de05450c40ec39df7c97073a6feecc8f81), published by RUN-260924-fda5ad, 20 changed paths, patch and validation log attached as TASK-260924-3n78rv_change-request_rev4.patch / _rev4-validation.log. Revisions 1..3 are earlier construction attempts of the same producer scope that failed the Change Request validation suite on the pre-PR51 10m race-test timeout under concurrent suites; they carry no review verdict and no prior review findings — review revision 4 on its own merits.. It is task_delta, so a registry edit is out of scope and counts as a finding. Read the producer brief `TASK-260924-3n78rv_producer.md`, the results, the conformance matrix, the evidence tar, the CR patch and validation log, and the immutable CR bytes.

REVISION HISTORY: if this is not revision 1, every earlier verdict is attached as `TASK-260924-3n78rv_review-verdict-rev<N>.md`. Read the one before yours and the brief that answered it, and use `repeat-of:` for the same class at an adjacent site. Attached verdicts outrank this brief.

SCRATCH RULE: put every `git archive` copy, probe and mutant workspace under `/Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/review-scratch/$TASK_BOARD_RUN_ID/` — never under `/tmp` or `/private/tmp` (on 2026-09-23 reviewer copies there filled the disk to 1 GB and broke three Change Request constructions) — and delete that directory after your evidence tar is attached. Exclude `.task-board` from archive copies unless a check needs it (run `specpin` in the live worktree instead).

LIVE-INDEX RULE: never run `git add`, `git add -N`, `git rm --cached`, `git reset` or any other command that writes the real index of the live Story worktree. Compare bytes through a scratch `GIT_INDEX_FILE`, and work on `git archive` copies.

NORMATIVE AUTHORITY: pinned `internal/specdoc/SPEC.v0.7.0.md` §13.14.1 (10385-10393: inert historical tools, low-authority foreign instructions, usage not accounting), §13.14.2 (10548-10570 strategies/profiles; ~10800-10812 VisibleMigrationProjection authority=user_context). The Story was refreshed from base 0ca3e4c onto trunk 6d3bff9: every trunk-only path must be blob-equal to the new base.

NOT OPTIONAL — the rev1 reviewer marked the importer outcome grid `not-attacked: budget`; that is not acceptable for this row. Regardless of how the rest goes. State each result, and RERUN each yourself; do not accept producer evidence:
- the full `go test ./...` on the exact tree;
- `GOOS=windows GOARCH=amd64 go vet ./...`;
- the importer outcome comparison;
- a determinism rerun, `-count=3` on the new tests;
- a byte-exact recomputation of every normative fixture, using your own tiny independent JCS+SHA-256 script (Python is fine), not the candidate's code.

Verify with your own instruments:
(1) **Authority by type, never text.** Build your own adversarial text corpus (instruction-like, reply-like, control-token-like, authorization-like, mixed-script, very long) and drive it through every production entry at every event position; any emitted assistant reply, system instruction, tool call or authorization, or any authority other than user_context, is a P1. Plant a narrowing that lets one text class raise authority; it must die ALONE. Check the structural (AST) guard with your own control plant.
(2) **Planning over the whole domain.** Plant: a profile ignored for one strategy, a wrong disposition for one event kind, continuation_context marked native, an incomplete call not aborted, source usage leaking into target accounting. Each must die ALONE. Oracles must come from spec text, not production code.
(3) **Entries and axes.** Every production entry; a rule pinned at one entry or along one axis is a finding.
(4) **Composition and hygiene.** Importer outcome grid keyed `(package, entry, input)` rerun YOURSELF; harness rerun, each KILLED twice; at least four own plants plus an applied control; exact-tree `git diff --check` including untracked; Windows vet; `task-board.config.json` byte-identical to base; full configured suite green on the exact tree.

Report the ratio YOU measured. Attach `TASK-260924-3n78rv_review-verdict-rev4.md` and `TASK-260924-3n78rv_review-evidence-rev4.tar.gz` and complete the live checklist. Then either run `accept_cr(TASK-260924-3n78rv, revision=4, evidence=TASK-260924-3n78rv_review-verdict-rev4.md)`, or hand off changes requested with P1/P2/P3 findings and a "Rework scope (for the producer)" section. Model: gpt-6-sol medium. Canonical CLI: /Users/iv/.curator/global/bin/task-board.
