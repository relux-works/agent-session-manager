REWORK for TASK-260830-1jmmqn, CR rev3 → rev4. Read `TASK-260830-1jmmqn_review-verdict-rev3.md` first. Every rev1 finding is closed. The one new finding, `plan-constant-second-value-uncounted`, belongs to the same family as rev1: a hand-picked sample standing in for a domain. `TestPlanConstants` probes three invalid strings (`prepare`, `clone2`, `Clone`), so widening a gate to admit `copy` survives.

The invalid domain of a closed constant is every other string, which is unbounded. Close it BY CONSTRUCTION:
1. **Structural guard.** Add an AST/source test over the production files. Every closed-constant gate (`materialization_intent`, `target_collision_policy`, `activation`, the ReadBack `modes`, the `require_*`, `opens_existing_identity`, `allow_blank_fallback`, `required`, `retain_through` and `forbidden_after_provider_commit` constants, and any other closed literal in these schemas) must compare the decoded value for equality against exactly ONE spec literal and refuse otherwise. No second admitted value, no switch with several accepting cases, no prefix/fold/contains comparison. Derive the gate list mechanically: find every field typed as a closed constant, never from a hand list. Control-plant the guard with the reviewer's `copy` widening and with a two-case switch; it must redden on both.
2. **Generated invalid values** for every closed constant at both entries (Build and Decode):
   - every string at edit distance 1 from the literal (insertion, deletion, substitution and transposition over a lowercase+digit+`_` alphabet);
   - every case variant class;
   - the empty string, whitespace padding and non-string JSON types;
   - a fixed-seed random sample of other strings.

   Each must be refused with the literal code from the pinned spec. For boolean constants, test the other boolean and every non-boolean JSON type.
3. **Mutants.** Ship an admitting narrowing for EACH independent constant gate on both the build and decode sides (the reviewer found two independent intent gates). Each must be KILLED by a named test run ALONE, with raw logs and subprocess exit codes. Keep the neutral control.
4. Update the conformance matrix, TRACEABILITY and the measured ratio. Run `GOOS=windows GOARCH=amd64 go vet ./...`, exact-tree hygiene through a scratch index (including untracked files), the harness and the full configured suite. Keep the checklist current, then run `task-board handoff TASK-260830-1jmmqn --role developer`.

Base trunk `0ca3e4c`. If it moves, run `refresh-candidate` first. SCRATCH RULE: no /tmp. Canonical CLI: /Users/iv/.curator/global/bin/task-board. Model: muse-spark max. Reviewer: gpt-6-sol medium.
