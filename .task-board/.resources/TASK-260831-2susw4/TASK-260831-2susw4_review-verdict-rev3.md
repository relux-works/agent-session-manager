# TASK-260831-2susw4 review verdict — CR revision 3

Verdict: **changes requested** (`to-dev`).

Candidate reviewed: tree `5e0189423f4cf12252c23b8f40241260faa5397a`, patch SHA-256 `1023652af9a32c2879c0126cc879d2b547574d0cc12825096ae1ef52a9dc9222`.

## Finding 1 — assigned scope is not required to retain its scope-specific executable case

The production path `main -> run -> traceability.VerifyAssignedSections -> verifyOwnership` checks that the assigned `source:<top-level>` key exists, but it does not require that key's ownership group to retain the `assigned-section-binding` acceptance case. `verifyOwnershipGroups` only requires a non-empty list of any registered acceptance cases, and the later assigned-binding loop checks only key presence.

In an isolated exact-candidate mutant I removed only `assigned-section-binding` from the `source:*` ownership group, retained every source key and the globally registered acceptance-case row, and updated `reviewedOwnershipCanonicalSHA256` to the mutant registry digest. The narrowed production command `tracecheck -section 10.1` still exited 0, and `go test ./internal/traceability/... -count=1` remained green. The assigned scope was therefore admitted using generic source-pin tests that do not exercise assigned-scope resolution.

This is the **bypass path around the check** / **narrowed gate survives** shape. It violates the acceptance criterion that an assigned scope be bound to an executable acceptance case rather than merely being internally consistent with registered rows.

## Finding 2 — the reported implementation owner is the universal pin validator, not the scoped implementation

All 24 `source:*` keys, including `source:10`, resolve to `internal/specpin/pin.go:Verify`. The reviewed candidate contains no `internal/scalar` implementation from the blocked `TASK-260830-1pbx0c` candidate, yet the production command accepts Sections 10.1-10.4 and reports four assigned scopes. Thus the scoped gate can report those sections as owned while the implementation whose review it is intended to unblock is absent.

This is **absent evidence treated as satisfied**: the generic normative-lock validator is accepted as the implementation owner for every product section. Collapsing every subsection to one top-level pin owner also prevents the ownership result from distinguishing actual implementations of 10.1, 10.2, 10.3, and 10.4.

## Required rework

1. Make the assigned-scope production path require a scope-relevant implementation owner and scope-specific executable acceptance case. A generic `specpin.Verify` owner plus unrelated source-pin cases must not satisfy product-section ownership.
2. Add a production-entry narrowed test that removes only the assigned scope's acceptance-case binding, re-pins the isolated mutant, requires the assigned section to fail, and keeps an unrelated section green.
3. Add a production-entry negative test proving the assigned section refuses when its actual implementation declaration is absent or renamed; name the real `main -> run -> VerifyAssignedSections` call site.
4. Preserve the already-correct revision-3 work: the 24-entry board union, exact v0.5.0 identity, 157-entry reviewed inventory, nonexistent-section refusal, catalog generation, and existing `source:10` key-removal mutant.

## Evidence and validation

- Exact upstream tag object `d3da6614a6c7bf119a88c9596a86c0853c22cfb9`, peeled commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, and `SPEC.md` SHA-256 `562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a` independently reproduced.
- Candidate inventory matched the pinned upstream numeric headings plus top-level Appendices A-D exactly.
- All 13 configured validation commands independently exited 0 on an archived exact candidate tree. `task-board validate` still printed 264 inherited `MISSING_ACTIVITY` diagnostics while exiting 0; this is pre-existing board evidence noise, not a candidate regression.
- Candidate `base..tree` diff passed `git diff --check`; repository source files were not modified during review.

Attached logs:

- `TASK-260831-2susw4_reviewer-binding-attack-rev3.log`
- `TASK-260831-2susw4_reviewer-source-identity-rev3.log`
- `TASK-260831-2susw4_reviewer-validation-rev3.log`
