REWORK for TASK-260830-2h5uv9, CR rev3 → rev4. Read `TASK-260830-2h5uv9_review-verdict-rev3.md` first; its findings JSON outranks this brief. The Story is based on trunk `6d3bff9`. If trunk moves, run `refresh-candidate` exactly once, as the LAST step before the handoff, on the final candidate, with MERGE of the registry.

The finding is `sync-conflict-not-crossed-with-partial-peer`. The same-identity/different-bytes branch is exercised only as a fixed tail scenario, in which the conflict peer holds nothing but the conflicting object. It is not crossed with the perturbation product. A narrowing that audits common IDs only when the peer holds nothing else therefore survives.

INVARIANT: every identity common to local and peer is byte-audited on every sync, regardless of anything else the peer holds or the order, duplication or gaps in delivery. A mismatch always yields the literal `integrity_failure`, quarantine, abort, and no silent divergence.

DELIVER BY CONSTRUCTION:
1. **Conflict is a CLASS inside the generator product,** not a separate tail. Every generated case may carry a same-identity/different-bytes object, at any position, from a peer that also holds identities the local lacks, crossed with reorder × duplicate × gap-filled rounds × partial peer × repeated sync. For every conflicting run, compare both replicas' bytes and assert `integrity_failure` plus quarantine. Add an always-run named test for partial overlap, with the reviewer's reproduction as the minimum.
2. **Structural argument.** Show that the audit loop over common IDs has no dependency on the size of the missing set, the peer set or the namespace set. An AST check over `DurableIndex.SyncFrom` and its callees works: no condition on `len(missing)`, `len(peerIDs)` or a namespace index gates the audit. Control-plant it.
3. **Mutants,** each KILLED by its named test run ALONE under the configured suite:
   - the reviewer's `if len(common) == len(peerIDs) { audit }`;
   - audit only when `len(missing) == 0`;
   - audit only the first namespace;
   - audit only the first round.
4. Rerun tracecheck, the README pin and the digest, since registry test lists may name the new tests. Decode the registry and prove every case of all three leaves is in its clause list. Rerun the importer grid, `GOOS=windows GOARCH=amd64 go vet ./...`, exact-tree hygiene through a scratch index, the harness and the full configured suite. Keep the checklist current, then run `task-board handoff TASK-260830-2h5uv9 --role developer`.

SCRATCH RULE: no /tmp. LIVE-INDEX RULE applies. Canonical CLI: /Users/iv/.curator/global/bin/task-board. Model: gpt-6-luna max. Reviewer: claude-opus-5-5 low.
