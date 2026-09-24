REWORK for TASK-260830-g0pcnt, CR rev3 → rev4. Read `TASK-260830-g0pcnt_review-verdict-rev3.md` first; its findings JSON outranks this brief. The base is trunk `0ca3e4c`: trunk moved from `360c8bd` because PR #61 changed only `task-board.config.json`. Before handoff you MUST run `task-board worktree refresh-candidate TASK-260830-g0pcnt`, because CR construction requires the checkpoint to descend from current trunk. After the refresh, `task-board.config.json` must equal `0ca3e4c`'s version, and every other trunk-only path must be blob-equal to `0ca3e4c`.

THIRD CONSECUTIVE ROUND OF ONE CLASS. The loop contract's response to that is a design change, not another row. Here is how each round went:
- rev1 pinned two modes;
- rev2 pinned each mode bit singly;
- rev3 pinned all 4096 modes, but only at the first ancestor. A walk that stops after that ancestor survived (verdict P1).

The orchestrator's rev3 brief defined "the whole input space" as the mode space. That definition was wrong; the error is mine, not yours. The input of the custody predicate is the WHOLE PATH: an ordered list of components, each with (position/depth, component class, kind, owner, mode), reached through an entry.

DELIVER — the custody property by construction over the whole path:
1. **Axis inventory first.** In the conformance matrix, write one row per axis for every custody gate: component class, position/depth (1..N up to the filesystem root, where N ≥ 3 in the fixture), mode (4096), kind (dir, file, symlink, FIFO/socket where applicable), owner (current user, other user, root), and entry (Probe, Spawn, Execute). For each axis, name the test that varies it, or state a bound with its reason and an owner. An axis without a row is the next finding, so write the inventory before writing code.
2. **Oracle over the product.** Extend the independent oracle test so that, for every position on the walk from the socket leaf up to the filesystem root of the fixture, it sets exactly one component to each of the 4096 modes while all others stay safe. Production must equal the oracle at every (position × mode × class) through every entry. Also cover the admit direction at every depth (01777 sticky). Every position of the actual walk is swept, not just one. A single "grandparent 0777" row is the point-fix shape and is NOT acceptable on its own.
3. **Kind and owner axes.** Pin these at every position by enumeration: kind ∈ {dir, symlink, regular file} and owner ∈ {self, other non-root, root}. Where the host cannot materialize a combination without root (for example a foreign owner), drive the shared production predicate through the fixture seam, and prove by a wiring test that each entry calls that predicate. Ancestor ownership (N2) was a stated bound: keep it as a bound or close it, citing §3.2 806-813.
4. **Mutants.** Ship these, each KILLED by the oracle test run ALONE:
   - P1 (stop after the first ancestor);
   - "skip the first ancestor";
   - "skip the last ancestor below the filesystem root";
   - "check only even depths".

   Keep the neutral control. Record `custodyModeProjection` in TRACEABILITY with the other test seams, as the reviewer noted.
5. Rerun the importer grid, `GOOS=windows GOARCH=amd64 go vet ./...` and the full configured suite. Update the census, matrix and evidence tar (under 1 MiB). Keep the checklist current, run `refresh-candidate` if you have not yet, then `task-board handoff TASK-260830-g0pcnt --role developer`.

If the oracle over the whole path cannot be expressed without a production seam you consider unsafe, stop and say so rather than narrowing the sweep. LIVE-INDEX RULE applies. Canonical CLI: /Users/iv/.curator/global/bin/task-board. Model: gpt-6-luna max. Reviewer: claude-opus-5-5 low.
