REWORK for TASK-260830-g0pcnt, CR rev2 → rev3. First read `TASK-260830-g0pcnt_review-verdict-rev2.md`; its findings JSON outranks this brief. The base is trunk `360c8bd`, and the Story tip `5443b3b` already descends from it. Do NOT refresh unless trunk moves again. If it does move, you MUST run `task-board worktree refresh-candidate TASK-260830-g0pcnt` before handoff, because CR construction requires the checkpoint to descend from the current trunk.

THIS IS THE SECOND CONSECUTIVE ROUND OF ONE CLASS (`custody-mask-pinned-single-bit-axis`, repeat-of rev1 P2). Rev1 witnessed the modes 0600 and 0644. Rev2 witnessed each non-owner bit singly. Both times an admitting narrowing survived one step away:
- rev2 M2: an ancestor with setgid is treated as sticky, so 02777 is admitted;
- rev2 M5: a root at 0750 is admitted.

Adding more rows cannot close this class. It closes only when the property holds BY CONSTRUCTION and is tested over the WHOLE input space.

THE INVARIANT (pinned SPEC v0.7.0 §3.2, lines 806-813). For each path-component class (socket leaf, runtime `tmux` dir, runtime root, each ancestor), custody admits a mode if and only if a single predicate that you write in the TEST as an independent oracle admits it. That oracle is derived from the spec text, not copied from `bind.go`. Leaf, runtime dir and root: owner-only, meaning no group or other bit and no special bit the spec does not name. Ancestors: not writable by group or other, unless sticky is set. Setuid and setgid never substitute for sticky. Owner and kind requirements are unchanged.

DELIVER:
1. **An exhaustive oracle test.** For every component class, enumerate ALL 4096 values of the 12 low mode bits (perm plus setuid, setgid and sticky) with a fixed valid kind and owner. Assert that the production custody decision equals the oracle for every value. Drive it through each of the three production entries: `ServerProber.Probe`, `ServerSpawner.Spawn` and `Lifecycle.Execute`. Each (entry × class) is a named subtest. Every refusal asserts the literal `tmux_unsafe_socket_path` and its detail, with zero Dial, spawn and dispatch. Every admit reaches the next stage. Use the fixture seams the census already uses: 4096 × classes × 3 entries must run in seconds and must never touch a real socket. If some combination cannot be materialized on the host filesystem (for example setuid on a directory as non-root), drive that class through the production predicate the entries share, and prove by a wiring test that each entry calls that exact predicate. Wire-level plus predicate-level is acceptable ONLY with that wiring proof.
2. **Mutants.** Ship the reviewer's M2 and M5, plus these narrowings:
   - a combination-admit narrowing for the socket leaf and for `ownerOnlyRuntimeDirMode`;
   - a setuid-as-sticky narrowing;
   - a "sticky ignored" narrowing, in the refuse direction.

   Each must be KILLED by the oracle test run ALONE. Include an applied neutral control.
3. **N1.** For the concurrent-winner arm at `termbind/attach.go:249`, either pin it with a hook-driven concurrent test (the `AfterStage` winner) or prove it is unreachable from Execute with the reason (the admission lock serializing same-client attaches), plus a test demonstrating that serialization. Do not leave it as prose.
4. **N2 (note).** Record a stated bound, with an owner, for ancestor ownership (a foreign non-root owner of an ancestor could rename it), or close it if §3.2 requires an ownership check. Cite the line.
5. Update the census table, TRACEABILITY and the conformance matrix. Re-derive the digest and the README pin only if cited test names change. Keep `task-board.config.json` byte-identical to `360c8bd`. Run the importer grid, `GOOS=windows GOARCH=amd64 go vet ./...` and the full configured suite. Attach the evidence tar (under 1 MiB). Keep the checklist current, then run `task-board handoff TASK-260830-g0pcnt --role developer`.

Also follow the developer role's Handoff Preconditions (coverage map: every surface row mapped to tests plus a killed narrowing). That contract is in your role body now that spawns use the full profile. LIVE-INDEX RULE applies. Canonical CLI: /Users/iv/.curator/global/bin/task-board. Model: gpt-6-luna max. Reviewer: claude-opus-5-5 low.
