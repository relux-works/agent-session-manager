Implement TASK-260924-2zboyz (read-back manifest and validation report schemas), the FOURTH leaf of STORY-260830-21bxa3 (M4). It was split from TASK-260830-1esv6u on 2026-09-24 so that each leaf has one attack surface. The Story branch carries three signed checkpoints:
- 2ya5le: `internal/clonefidelity`;
- 1jmmqn: `internal/cloneplan` schemas;
- 3n78rv: `internal/cloneplanning`.

Build on them. Reconciliation completeness and the Story registry belong to the final leaf 1esv6u; do not implement them here. Your Change Request is `task_delta`. Do NOT edit the traceability registry.

AUTHORITY. Use pinned `internal/specdoc/SPEC.v0.7.0.md` §13.14.2: Clone Read-Back Evidence Manifest 1.0.0 (line 10737+) and Clone Validation Report 1.0.0 (10755+), plus the prose at ~10580-10590 ("Modes cannot be relabeled", "every applicable check must pass"). Quote the pinned file.

SINGLE OWNER PER RULE, FROM THE FIRST REVISION. The previous leaf of this Story needed five revisions because it re-implemented subsets of a landed owner's rules at each new entry. `internal/clonefidelity` owns the Fidelity Report and the disposition/reason rules. `internal/cloneplan` owns the Projection Plan and manifest rules. Every exported entry here that accepts data governed by those owners must call THEIR validators. Enforce that with an AST/call-graph census that fails on an entry that does not reach the owner (control-plant it), and with a structural guard against any second local implementation.

HOW TO PROVE IT: by construction, over whole domains.
1. **Member census by reflection** over every schema type, nested types included. For every member, at every entry: missing, extra, duplicated, miscased and wrong JSON type (6) are refused with a literal code. Derive the member set from the types; never type it by hand. Add a guard that fails if a literal member list appears in the tests.
2. **Identities.** Recompute each id independently: the JCS digest with only the id member omitted.
3. **Semantic gates by whole-domain oracle.**
   - staged/live mode cannot be relabeled;
   - expected equals observed native ID;
   - sorted-unique evidence blobs;
   - `valid=true` if and only if every APPLICABLE check passes: enumerate every combination of check outcome × applicability;
   - bounds at edge and edge+1.
4. Write the AXIS INVENTORY first. For every gate, ship an admitting narrowing that its named test kills when run ALONE, plus an applied neutral control.

TRUNK. The Story sits on trunk `6d3bff9`. If trunk moves, run `task-board worktree refresh-candidate TASK-260924-2zboyz` exactly ONCE, as the last step before the handoff, on the final uncommitted candidate. Never run it on a clean worktree (task-board #316).

CENSUS. Commit a gate × entry table in the conformance matrix, mirrored in TRACEABILITY, with the domain axes above. Every production entry that reaches a gate is its own cell. Follow the developer role's Handoff Preconditions: a coverage map with each surface row pointing to its tests and a killed narrowing.

COMPOSITION. Reuse `internal/canonicaljson` for JCS and digests, and the landed scalar/digest types. If a type you need is missing, stop and name its owner. Build the importer outcome grid keyed `(package, entry, input)` over every package you touch, base vs candidate, and name every moved class.

WORKSPACE. Use the managed Story worktree for STORY-260830-21bxa3 that the spawn provisions. The candidate is the UNCOMMITTED tree. Never commit on the Story branch, never edit `.task-board`, never self-accept, and never write the real index except through the handoff. If trunk moves before your handoff, run `task-board worktree refresh-candidate TASK-260924-2zboyz` first: CR construction requires the checkpoint to descend from current trunk.

PROCESS. Read the `go-testing-tools` skill first. All CI is local. Run:
- the package tests;
- the importer set;
- `GOOS=windows GOARCH=amd64 go vet ./...`;
- the mutation harness;
- the full configured suite once.

Evidence tar under 1 MiB. Attach `TASK-260924-2zboyz_results.md`, `_conformance-matrix.md` and `_producer-evidence.tar.gz`, and read each back off the board. Keep the checklist current, run `task-board handoff TASK-260924-2zboyz --role developer`, and exit. Hand off within 3 hours. Canonical CLI: /Users/iv/.curator/global/bin/task-board. Model: muse-spark max. Reviewer: gpt-6-sol medium.
REJECTION PATTERNS THE INDEPENDENT REVIEWER HAS ALREADY CAUGHT ON THIS EPIC (2026-09-17: TASK-260830-1geqhj rev1, TASK-260830-24z2b3 rev2, TASK-260830-219okr rev3 were all routed CHANGES REQUESTED, and between them these six shapes account for every finding). Read this as a checklist you run against your own work BEFORE handoff; each item names what a reviewer will do to your candidate.
(a) Circular expectations. An assertion that compares a production value against a production constant (`got.Code == pkg.SomeUnsupportedCode`) pins nothing: swapping the constant to another registered value with the same exit keeps the suite green. Assert the LITERAL token from the pinned specification text, at the production entry, and make sure the literal actually occurs in a committed test — not only in a comment. Derive every expectation from the spec, never from the implementation.
(b) A rule pinned at the helper instead of the entry. A test that calls the inner helper directly proves the helper; it says nothing about whether the composed production entry still reaches it. Drive every rule through the entry point the conformance matrix names as its call site, and if a rule is implemented TWICE (a Build path and a Decode path, an exported entry and its internal sibling), each side needs its own narrowing mutant — a rule pinned only on one side leaves the other side unmeasured, and reviewers plant exactly there.
(c) Forking a landed gate. Never re-implement what internal/environ, internal/canonicaljson, internal/scalar or another landed owner already gates (string bounds in CHARACTERS not bytes, uint53, digests, UUIDv7, timestamps, sorted-unique arrays, closed-shape decoding, the AX number model that forbids floating point, values at or beyond 2^53 and rounding-and-continuing). Delegate to it and keep only your package-specific rules. Every fork this epic produced became a defect: a byte-measured bound, an extension value rounded at 2^60, nested duplicate members collapsed by a lenient decode.
(d) A census or an inventory standing in for behaviour. A reflection census of field NAMES does not prove that a cached observation cannot authorize; an arm inventory keyed by message text does not prove a condition edit is caught. Drive the input, assert the effect.
(e) Mutant rows that do not measure what they claim. An arm-delete (`&& false`) is not a narrowing; a row patching two sites whose kill comes from one is one measured site reported as two; a plant with no raw log and no subprocess exit is a verdict line, not evidence. Ship one raw log per plant per run, state narrowing vs arm-delete per row, and include an applied harmless SURVIVED control so the harness is proven able to report a survivor.
(f) Prose that outruns the tests. "Composes X", "never admits what decoding refuses", "nothing re-implemented", "no cached sentinel can authorize" are censusable claims: a reviewer greps for the callers and plants the counter-example. Before you write a sentence like that in a doc comment, README, LOGBOOK or TRACEABILITY row, make a test fail without the property. If you cannot, state it as an explicit BOUND instead, naming what is not covered.
Finally: report the coverage ratio you MEASURED, not the one you planned. Two of the three rejected revisions claimed a complete ratio and the reviewer measured a smaller one; an honest smaller number with named gaps is accepted, an inflated one is a finding.
