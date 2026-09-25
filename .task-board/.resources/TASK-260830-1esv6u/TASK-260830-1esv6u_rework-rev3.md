REWORK for TASK-260830-1esv6u, CR rev2 → rev3. Read `TASK-260830-1esv6u_review-verdict-rev2.md` first; its three findings outrank this brief. The refresh onto `5b7876b` is done. Do not refresh again unless trunk moves; if it does, refresh exactly once, as the last step before the handoff.

1. **P1 source-evidence-chain-mismatch.** INVARIANT: each non-synthesized fidelity row's source evidence is the evidence of ITS OWN candidate/raw chain. Global membership is not enough. `Reconcile` must bind every `SourceEvidenceIDs` digest to `rawByCandidate[row.SourceItemKey]`, and must refuse, each with a literal code AND detail:
   - a swapped digest (another candidate's raw);
   - a foreign digest (not in any chain);
   - a missing digest;
   - a double-claimed digest (one raw claimed by two rows).

   Put this relation class in the oracle so it is enumerated, not hand-picked. Add a public-entry regression test and one admitting narrowing per relation class, each killed ALONE.
2. **P2 reconciliation-product-not-enumerated.** Enumerate the FULL product: every capture-class assignment for N = 1..6 (all class vectors, not uniform or periodic samples) × all nine read-back outcomes, against the independent oracle. The domain is a pure function over roughly 5^6 × 9 cases and fits in the test budget. If it does not, shard it deterministically, but every shard must run in the configured suite. Keep an EXECUTABLE coverage assertion that computes the expected count from the class and outcome vocabularies and compares it to the executed count, so sampling or duplicates can never be reported as exhaustive. Add a narrowing that the N ≤ 3 subset cannot kill, one that only N ≥ 4 reaches, and show it killed.
3. **P2 story-cases-have-no-clause-edges. ORCHESTRATOR DECISION.** The traceability clause model extracts clauses on RFC keywords. §13.14.2's reconciliation sentence ("Every captured candidate reconciles once …") contains no keyword, so the section yields ZERO clauses, and no clause edge can exist. Do not fabricate clause edges, and do not keep a test that merely asserts "zero clauses" as satisfying the contract. Instead:
   - (a) bind all five Story cases at the SECTION level (`section:13.14.2` acceptance_cases, decoded and proven present);
   - (b) add a registry test that FAILS when a section with ≥1 extracted clause has a Story case without a clause edge, so the stricter rule applies wherever it is representable;
   - (c) record the clause-model limitation as a stated bound in TRACEABILITY and the results. Name its owner, the traceability tool (clause extraction does not recognise declarative normative sentences), and add a LOGBOOK note.

   Re-derive the digest, run tracecheck and the section-scoped runs, and check the README pin.

Rerun the importer grid, `GOOS=windows GOARCH=amd64 go vet ./...`, exact-tree hygiene through a scratch index, the harness and the full configured suite. Keep the checklist current, then run `task-board handoff TASK-260830-1esv6u --role developer`. SCRATCH RULE: no /tmp. Canonical CLI: /Users/iv/.curator/global/bin/task-board. Model: muse-spark max. Reviewer: gpt-6-sol medium.
