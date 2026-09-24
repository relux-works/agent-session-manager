REWORK for TASK-260830-2h5uv9, CR rev4 → rev5. Read `TASK-260830-2h5uv9_review-verdict-rev4.md` first. The generated product exists, but the configured suite never RUNS it: `TestDurableSyncGeneratedPerturbationProduct` calls `t.Skip()` without a nested `-run` selector. The always-run test also puts at most one common identity per namespace, at the first position, so a narrowing that audits only the first common ID survives the suite.

INVARIANTS:
- Evidence a finding relies on runs in the configured suite. A test that skips by default proves nothing in CI.
- Every common identity is audited at EVERY position in the common set, in every namespace.

DELIVER:
1. **An always-on bounded shard of the generated product.** It runs with no selector under `go test ./internal/merkleinventory` and under the configured suite. It covers N ≤ 3 objects, every target, every round count, the conflict class, and ≥2 common identities in ONE namespace with the conflicting identity at EVERY position (first, middle, last), crossed with peer-only identities. The large product may stay behind a selector, but the shard must independently kill every mutant below.
2. **A skip census.** A committed test scans the package's `_test.go` files (AST) for `t.Skip`/`t.Skipf`/`t.SkipNow`. It fails unless each one carries an allowlisted, justified reason (platform or an environment capability) and a sibling always-on test covers the same property. Control-plant it with a planted unjustified skip.
3. **Mutants,** each KILLED ALONE under the configured suite (not only under a selector):
   - the reviewer's plant E (`common = common[:min(1,len(common))]`);
   - "audit only the last common ID";
   - "audit only even positions";
   - "audit only the first namespace".

   Keep the applied neutral control.
4. Remove the `.pyc` artifact and add a hygiene check that no `.pyc`/`__pycache__` lands in the tree. Rerun tracecheck, the README pin and the digest if registry test lists change, and decode the registry edges. Rerun the importer grid, `GOOS=windows GOARCH=amd64 go vet ./...`, exact-tree hygiene through a scratch index, the harness and the full configured suite. Keep the checklist current, then run `task-board handoff TASK-260830-2h5uv9 --role developer`. If trunk moves, run `refresh-candidate` exactly once, as the last step before the handoff.

SCRATCH RULE: no /tmp. Canonical CLI: /Users/iv/.curator/global/bin/task-board. Model: gpt-6-luna max. Reviewer: claude-opus-5-5 low.
