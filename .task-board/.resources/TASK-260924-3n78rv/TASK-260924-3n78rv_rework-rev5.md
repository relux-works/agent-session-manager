REWORK for TASK-260924-3n78rv, CR rev4 → rev5. Read `TASK-260924-3n78rv_review-verdict-rev4.md` first. The rev3 UTF-8 bypass is FIXED. The new finding, `effects-reason-set-coupling-missing`, is the same shape one entry further along. `PlanTargetEffects` checks the disposition and reason vocabulary but not the §13.14.2 record rule "exact requires an empty reason set; every other disposition requires at least one reason". That rule already exists and is proven in the landed leaf 2ya5le (`internal/clonefidelity`). This package re-validates pieces of it by hand and skips the rest.

INVARIANT: the §13.14.2 record rules have ONE owner, `internal/clonefidelity`. Every exported entry in `internal/cloneplanning` that accepts a disposition, reason set, mapping or record calls that owner's validator. It never re-implements a subset. Deliver this BY CONSTRUCTION:
1. **Entry census by AST.** List every exported entry whose parameters carry a disposition, reason set, ProjectionItemMapping or FidelityDispositionRecord, followed transitively. A committed test fails if any listed entry does not reach the `clonefidelity` validator: check the static call graph, or a runtime spy on the owner seam. Control-plant it with an entry that bypasses the owner; the test must redden.
2. **Rule × entry table.** Take EVERY record rule from the owner's validator, not only the reviewer's pair:
   - exact ⇔ empty reason set;
   - non-exact ⇒ at least one reason;
   - synthesized ⇒ no canonical object;
   - non-synthesized ⇒ source evidence;
   - vocabulary membership;
   - sorted-unique arrays;
   - bounds.

   For every rule × every censused entry, drive a violating input and assert the owner's literal code. Drive the reviewer's `semantic` with no reason and `exact` with an `operator_policy` reason through `PlanTargetEffects` explicitly.
3. **Remove the local partial checks** that duplicate the owner, so that exactly one implementation remains. Add a structural guard that the package contains no second disposition/reason rule implementation, for example no switch on disposition that decides admissibility.
4. **Mutants.** Ship one admitting narrowing per rule class at `PlanTargetEffects`, plus a narrowing that bypasses the owner call at one other entry. Each must be KILLED by its named test run ALONE.
5. Keep every held row, including the rev3 UTF-8 census and guard. Include complete command evidence. Run `GOOS=windows GOARCH=amd64 go vet ./...`, exact-tree hygiene through a scratch index, the harness and the full configured suite. Keep the checklist current, then run `task-board handoff TASK-260924-3n78rv --role developer`.

If trunk moves, run `refresh-candidate` once, as the last step before the handoff. SCRATCH RULE: no /tmp. Canonical CLI: /Users/iv/.curator/global/bin/task-board. Model: muse-spark max. Reviewer: gpt-6-sol medium.
