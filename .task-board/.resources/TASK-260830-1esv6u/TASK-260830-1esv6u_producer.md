Implement TASK-260830-1esv6u, the FINAL (story_final) leaf of STORY-260830-21bxa3 (fidelity projection and validation contracts, M4). The orchestrator split it on 2026-09-24: this leaf proves RECONCILIATION COMPLETENESS and carries the Story registry. The schemas belong to the checkpointed sibling 2zboyz. The Story branch carries four signed checkpoints:
- 2ya5le: `internal/clonefidelity`;
- 1jmmqn: `internal/cloneplan`;
- 3n78rv: `internal/cloneplanning`;
- 2zboyz: `internal/clonereadback`, with the sealed `ValidatedReadBack`.

Your Change Request is story_final and carries the WHOLE Story.

AUTHORITY. Use pinned `internal/specdoc/SPEC.v0.7.0.md` §13.14.2, lines 10554-10556: "Every captured candidate reconciles once to raw evidence or exclusion; every raw item to a canonical item or normalization disposition; every canonical item to staged/live target evidence or a target disposition". Also use the Fidelity Report and Validation Report rules (10548-10760).

THE PROPERTY, BY CONSTRUCTION. Build a generated property over capture sets: N ≤ 6 items, with included, excluded, opaque, synthesized, and lost/unrecoverable items. Cross each set with the read-back outcomes: staged and live present, absent or mismatched. Use an INDEPENDENT oracle written from the spec text. The property:
- every item reconciles EXACTLY ONCE at every tier;
- a missing link, a duplicate link or a double-counted item at ANY tier refuses with a literal code;
- a synthesized row never claims source evidence.

Enumerate the product. Do not hand-pick cases.

OWNERS. This Story already paid five revisions for re-implementing owner rules at new entries. Read-back comes ONLY through `clonereadback` (`ValidatedReadBack`). Report derivation comes ONLY through the `clonefidelity` and `cloneplan` validators. Prove it with an AST/call-graph census that fails on an entry not reaching those owners (control-plant it) and a guard against local re-implementation.

MEASUREMENT. Write the AXIS INVENTORY first. Build a GATE REACHABILITY MATRIX derived mechanically from the refusal sites: every site gets a row that breaks only that gate and asserts its code AND detail, and an unlisted site fails the census. No evidence test may skip by default: add a skip census with a justified allowlist, and make sure an always-on shard kills every mutant under the configured suite. For every gate, ship an admitting narrowing that its named test kills when run ALONE, plus an applied neutral control.

REGISTRY (story-final). Edit `internal/traceability/ownership.v0.7.0.json` for ALL FIVE leaves, taking the bindings from each package's TRACEABILITY.md. DECODE the registry and prove that every acceptance case is in its clause list and that each production owner is the implementing code. Re-derive `reviewedOwnershipCanonicalSHA256`, run tracecheck and the section-scoped runs, and keep the README "Measured coverage" subsection equal to the tracecheck output (grep all five sites). Add LOGBOOK entries newest-first.

TRUNK MOVED. The Story is based on `6d3bff9`. Trunk is now `5b7876b`, after STORY-260830-ub60id landed WITH REGISTRY ROWS and a one-line tmux fixture fix landed. As the LAST step before the handoff, on the final uncommitted candidate, run `task-board worktree refresh-candidate TASK-260830-1esv6u` exactly ONCE. Never run it on a clean worktree (task-board #316). The registry must be MERGED: every trunk row present, with the digest re-derived after the merge. Every trunk-only path must be blob-equal to the new trunk. If the refresh refuses, record the typed refusal and stop.

COMPOSITION. Build the importer outcome grid keyed `(package, entry, input)`, base vs candidate, and name the moved classes.

WORKSPACE. Use `.temp/STORY-260830-21bxa3/worktree`. The candidate is the UNCOMMITTED tree. Never commit, never edit `.task-board`, never self-accept, never write the real index except through the handoff. SCRATCH RULE: no /tmp.

PROCESS. Read the `go-testing-tools` skill first. Run:
- the package tests;
- the importer set;
- `GOOS=windows GOARCH=amd64 go vet ./...`;
- exact-tree hygiene through a scratch index, including untracked files;
- the harness;
- the full configured suite.

Evidence tar under 1 MiB. Keep the checklist current, then run `task-board handoff TASK-260830-1esv6u --role developer`. Hand off within 4 hours. Canonical CLI: /Users/iv/.curator/global/bin/task-board. Model: muse-spark max. Reviewer: gpt-6-sol medium.
REJECTION PATTERNS THE INDEPENDENT REVIEWER HAS ALREADY CAUGHT ON THIS EPIC (2026-09-17: TASK-260830-1geqhj rev1, TASK-260830-24z2b3 rev2, TASK-260830-219okr rev3 were all routed CHANGES REQUESTED, and between them these six shapes account for every finding). Read this as a checklist you run against your own work BEFORE handoff; each item names what a reviewer will do to your candidate.
(a) Circular expectations. An assertion that compares a production value against a production constant (`got.Code == pkg.SomeUnsupportedCode`) pins nothing: swapping the constant to another registered value with the same exit keeps the suite green. Assert the LITERAL token from the pinned specification text, at the production entry, and make sure the literal actually occurs in a committed test — not only in a comment. Derive every expectation from the spec, never from the implementation.
(b) A rule pinned at the helper instead of the entry. A test that calls the inner helper directly proves the helper; it says nothing about whether the composed production entry still reaches it. Drive every rule through the entry point the conformance matrix names as its call site, and if a rule is implemented TWICE (a Build path and a Decode path, an exported entry and its internal sibling), each side needs its own narrowing mutant — a rule pinned only on one side leaves the other side unmeasured, and reviewers plant exactly there.
(c) Forking a landed gate. Never re-implement what internal/environ, internal/canonicaljson, internal/scalar or another landed owner already gates (string bounds in CHARACTERS not bytes, uint53, digests, UUIDv7, timestamps, sorted-unique arrays, closed-shape decoding, the AX number model that forbids floating point, values at or beyond 2^53 and rounding-and-continuing). Delegate to it and keep only your package-specific rules. Every fork this epic produced became a defect: a byte-measured bound, an extension value rounded at 2^60, nested duplicate members collapsed by a lenient decode.
(d) A census or an inventory standing in for behaviour. A reflection census of field NAMES does not prove that a cached observation cannot authorize; an arm inventory keyed by message text does not prove a condition edit is caught. Drive the input, assert the effect.
(e) Mutant rows that do not measure what they claim. An arm-delete (`&& false`) is not a narrowing; a row patching two sites whose kill comes from one is one measured site reported as two; a plant with no raw log and no subprocess exit is a verdict line, not evidence. Ship one raw log per plant per run, state narrowing vs arm-delete per row, and include an applied harmless SURVIVED control so the harness is proven able to report a survivor.
(f) Prose that outruns the tests. "Composes X", "never admits what decoding refuses", "nothing re-implemented", "no cached sentinel can authorize" are censusable claims: a reviewer greps for the callers and plants the counter-example. Before you write a sentence like that in a doc comment, README, LOGBOOK or TRACEABILITY row, make a test fail without the property. If you cannot, state it as an explicit BOUND instead, naming what is not covered.
Finally: report the coverage ratio you MEASURED, not the one you planned. Two of the three rejected revisions claimed a complete ratio and the reviewer measured a smaller one; an honest smaller number with named gaps is accepted, an inflated one is a finding.
