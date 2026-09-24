Independently review Change Request revision 1 of TASK-260830-nxqqaw (implement-namespace-merkle-inventories). This is the first leaf of STORY-260830-ub60id: task_delta on the Story branch (trunk base 360c8bd; checkpointed predecessors none; tip 360c8bd79cc2325ab661a9ba591b6bfb82ebe0ea), published by RUN-260923-88314c, 31 changed paths, patch and validation log attached as TASK-260830-nxqqaw_change-request_rev1.patch / _rev1-validation.log.. It is task_delta, so a registry edit is out of scope and counts as a finding. Read the producer brief `TASK-260830-nxqqaw_producer.md`, the results, the conformance matrix, the evidence tar, the CR patch and validation log, and the immutable CR bytes.

REVISION HISTORY: if this is not revision 1, every earlier verdict is attached as `TASK-260830-nxqqaw_review-verdict-rev<N>.md`. Read the one before yours and the brief that answered it, and use `repeat-of:` for the same class at an adjacent site. Attached verdicts outrank this brief.

LIVE-INDEX RULE: never run `git add`, `git add -N`, `git rm --cached`, `git reset` or any other command that writes the real index of the live Story worktree. Compare bytes through a scratch `GIT_INDEX_FILE`, and work on `git archive` copies.

NORMATIVE AUTHORITY: the pinned `internal/specdoc/SPEC.v0.7.0.md`: §11.4 (7764-7934), §11.3 inventory/objects rows and bounds (7555-7561, 7734-7738), and §10. The task record's v0.5.0 is stale.

NOT OPTIONAL, regardless of how the rest goes. State each result, and RERUN each yourself; do not accept producer evidence:
- the full `go test ./...` on the exact tree;
- `GOOS=windows GOARCH=amd64 go vet ./...`;
- the importer outcome comparison;
- a determinism rerun, `-count=3` on the new tests;
- a byte-exact recomputation of every normative fixture, using your own tiny independent JCS+SHA-256 script (Python is fine), not the candidate's code.

Verify with your own instruments:
(1) **Fixtures, independently.** Recompute the empty, singleton and branch roots, both branch child hashes, and all six MIXED-NS-1 roots OUTSIDE the candidate. Diff them against what the production entry returns.
(2) **Every §11.4 rule at every entry.** Enumerate each production entry that reaches trie construction, membership classification, `inventory.*` serving and `objects.get`. For each, plant an admitting narrowing at an entry the producer did NOT cite, and name the killer run ALONE. Cover at least these: dropping rule 5 (the 64-nibble duplicate), reordering children labels, accepting uppercase hex or a 65-char prefix, classifying a Blob Descriptor as a record, letting an excluded local object move a count, and serving an invented empty child.
(3) **Axes.** Vary namespace, prefix length, count, id order, duplicates and schema class. A rule pinned along one axis is a finding.
(4) **Union exchange.** Drive MIXED-NS-EXCHANGE and MIXED-NS-N1 yourself. Plant a timestamp tie-break and a same-digest-different-bytes accept, and verify both die.
(5) **Composition.** Confirm the `internal/rpcwire` validators are reused, not duplicated. Diff the runtime outcome grids keyed on `(package, entry, input)` over the importer set; every moved class must be named. A landed test rewritten to accept new behaviour without a spec citation is a P1.
(6) **Census, mutation, hygiene.** Check that the gate × entry census exists and reproduces. Rerun the shipped harness and rerun each KILLED twice. Plant at least four narrowings of your own, plus an applied harmless control; NOT_APPLIED is not a pass. gofmt and vet must be clean. `task-board.config.json` must be byte-identical to the base. The full configured suite must be green on the exact tree.

Report the ratio YOU measured. Attach `TASK-260830-nxqqaw_review-verdict-rev1.md` and `TASK-260830-nxqqaw_review-evidence-rev1.tar.gz` and complete the live checklist. Then either run `accept_cr(TASK-260830-nxqqaw, revision=1, evidence=TASK-260830-nxqqaw_review-verdict-rev1.md)`, or hand off changes requested with P1/P2/P3 findings and a "Rework scope (for the producer)" section. Model: claude-opus-5-5 low. Canonical CLI: /Users/iv/.curator/global/bin/task-board.
