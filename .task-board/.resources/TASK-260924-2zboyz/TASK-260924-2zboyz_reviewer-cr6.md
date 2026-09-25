Independently review Change Request revision 6 of TASK-260924-2zboyz (implement-readback-manifest-and-validation-report-schemas). This is the first leaf of STORY-260830-21bxa3: task_delta on the Story branch (trunk base 6d3bff9; checkpointed predecessors 5e40826 618d78d 35ee723 ; tip 5e40826725a0880897d6bd8321c8fa8e062620d3), published by RUN-260924-24be9f, 21 changed paths, patch and validation log attached as TASK-260924-2zboyz_change-request_rev6.patch / _rev6-validation.log. Revisions 1..5 are earlier construction attempts of the same producer scope that failed the Change Request validation suite on the pre-PR51 10m race-test timeout under concurrent suites; they carry no review verdict and no prior review findings — review revision 6 on its own merits.. It is task_delta, so a registry edit is out of scope and counts as a finding. Read the producer brief `TASK-260924-2zboyz_producer.md`, the results, the conformance matrix, the evidence tar, the CR patch and validation log, and the immutable CR bytes.

REVISION HISTORY: if this is not revision 1, every earlier verdict is attached as `TASK-260924-2zboyz_review-verdict-rev<N>.md`. Read the one before yours and the brief that answered it, and use `repeat-of:` for the same class at an adjacent site. Attached verdicts outrank this brief.

SCRATCH RULE: put every `git archive` copy, probe and mutant workspace under `/Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/review-scratch/$TASK_BOARD_RUN_ID/` — never under `/tmp` or `/private/tmp` (on 2026-09-23 reviewer copies there filled the disk to 1 GB and broke three Change Request constructions) — and delete that directory after your evidence tar is attached. Exclude `.task-board` from archive copies unless a check needs it (run `specpin` in the live worktree instead).

LIVE-INDEX RULE: never run `git add`, `git add -N`, `git rm --cached`, `git reset` or any other command that writes the real index of the live Story worktree. Compare bytes through a scratch `GIT_INDEX_FILE`, and work on `git archive` copies.

NORMATIVE AUTHORITY: pinned `internal/specdoc/SPEC.v0.7.0.md` §13.14.2 Clone Read-Back Evidence Manifest 1.0.0 (10737+) and Clone Validation Report 1.0.0 (10755+). Split leaf: reconciliation completeness and the registry are OUT of scope (1esv6u). Rules owned by internal/clonefidelity / internal/cloneplan must be enforced by calling those owners at every entry; a local partial re-implementation is a finding.

NOT OPTIONAL — the rev1 reviewer marked the importer outcome grid `not-attacked: budget`; that is not acceptable for this row. Regardless of how the rest goes. State each result, and RERUN each yourself; do not accept producer evidence:
- the full `go test ./...` on the exact tree;
- `GOOS=windows GOARCH=amd64 go vet ./...`;
- the importer outcome comparison;
- a determinism rerun, `-count=3` on the new tests;
- a byte-exact recomputation of every normative fixture, using your own tiny independent JCS+SHA-256 script (Python is fine), not the candidate's code.

Verify with your own instruments:
(1) **Oracle independence and derived census.** The member census must be DERIVED from the Go types (reflection), not typed by hand; oracles must come from the spec text. Recompute at least two ids with your own tiny independent JCS+SHA-256 script.
(2) **Whole-domain sweeps.** Plant narrowings the producer did NOT ship: a case-insensitive member match, a staged/live relabel admitted, valid=true with one failing applicable check, an owner-validator call bypassed at one entry, a bound widened by one. Name each killer run ALONE.
(3) **Entries and axes.** Every production entry reaching each gate; a gate witnessed at one entry is a finding at the next. Check the axis inventory.
(4) **Composition and hygiene.** Importer outcome grid keyed `(package, entry, input)` rerun YOURSELF; shipped harness rerun, each KILLED twice; at least four own plants plus an applied control. gofmt/vet clean, Windows vet, `task-board.config.json` byte-identical to base, full configured suite green on the exact tree.

Report the ratio YOU measured. Attach `TASK-260924-2zboyz_review-verdict-rev6.md` and `TASK-260924-2zboyz_review-evidence-rev6.tar.gz` and complete the live checklist. Then either run `accept_cr(TASK-260924-2zboyz, revision=6, evidence=TASK-260924-2zboyz_review-verdict-rev6.md)`, or hand off changes requested with P1/P2/P3 findings and a "Rework scope (for the producer)" section. Model: gpt-6-sol medium. Canonical CLI: /Users/iv/.curator/global/bin/task-board.
