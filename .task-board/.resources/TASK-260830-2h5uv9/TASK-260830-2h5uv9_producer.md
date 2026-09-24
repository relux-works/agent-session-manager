Implement TASK-260830-2h5uv9 (property-test-order-duplicate-gap-convergence), the FINAL leaf of STORY-260830-ub60id (M2 anti-entropy). The Story branch carries two signed checkpoints: nxqqaw (namespace Merkle inventories, trie, serving, walk, fetch) and 147hsj (durable union, arrival-order independence, projection rebuild). Your Change Request is **story_final** and carries the WHOLE Story, including the traceability registry.

AUTHORITY. Use pinned `internal/specdoc/SPEC.v0.7.0.md`:
- §11.4 (7764-7934): union rules 1-7, "no last-writer-wins", MIXED-NS-EXCHANGE;
- §5.3 (1987-2061);
- §10 for object classes.

The task record's v0.5.0 is stale.

THE PROPERTY, generated over the WHOLE perturbation product and driven through the production sync entry. Do not hand-pick scenarios. Build a generator over small object sets (N ≤ 6, including competing leases, a losing-lease branch, tombstones and acks, and a same-digest-different-bytes pair). Take the product of these axes:
- arrival reorder: every permutation for N ≤ 5, sampled beyond that;
- duplicate delivery;
- a gap: an object missing and filled in a later round;
- clock skew: timestamps perturbed in both directions;
- a partial peer: a subset of namespaces or objects;
- repeated sync: 1 to 3 rounds.

For every generated run, assert that sync EITHER reaches the reference roots and projection OR exposes the explicit, typed conflict (quarantine and abort). It must never diverge silently. Compute the reference with an INDEPENDENT oracle in the test. Also assert:
- a partial peer never makes a local root regress;
- repeated sync after convergence is a no-op with a byte-identical store;
- conflicts carry literal codes at every entry.

Write the AXIS INVENTORY before writing code. Unbounded axes (rounds, set size) need a generated range plus a structural argument for why behaviour cannot depend on values beyond it. Past leaves lost 4 to 5 revisions to fixture constants standing in for domains.

For every rule, ship an admitting narrowing that its named test kills when run ALONE, plus an applied neutral control. Assert literal spec tokens, never a production constant.

REGISTRY: story-final, and the step most likely to go wrong. Edit `internal/traceability/ownership.v0.7.0.json` for ALL THREE leaves. Take each leaf's clause bindings from its package TRACEABILITY.md. Every binding names a clause that an executed test drives. **Decode the registry and prove that every acceptance case appears in its clause's acceptance-case list.** A case defined but referenced by no clause still passes tracecheck; this defect has reached review before. Re-derive `reviewedOwnershipCanonicalSHA256`, run tracecheck plus the section-scoped runs on the exact tree, and quote their output. Keep the README "Measured coverage" subsection equal to the tracecheck output; grep all five coverage-figure sites. Add LOGBOOK entries newest-first.

Another story_final Story (STORY-260830-2t4g7i, tmux) is also in flight and edits the same registry. If trunk moves before your handoff, run `task-board worktree refresh-candidate TASK-260830-2h5uv9`, MERGE the registry so that every trunk row is present, and re-derive the digest after the merge. CR construction requires the checkpoint to descend from current trunk. The trunk base is what the Story forked from (`0ca3e4c`), NOT a Story checkpoint.

COMPOSITION. Build importer outcome grids keyed `(package, entry, input)` over the complete importer set, base vs candidate, and name every moved class.

WORKSPACE. Use `.temp/STORY-260830-ub60id/worktree`. The candidate is the UNCOMMITTED tree on top of 147hsj's checkpoint. Never commit, never edit `.task-board`, never self-accept, never write the real index except through the handoff. SCRATCH RULE: use the worktree's `.temp/TASK-260830-2h5uv9/` or `t.TempDir()`, never `/tmp`.

PROCESS. Read the `go-testing-tools` skill first. All CI is local. Run:
- the package tests;
- the importer set;
- `GOOS=windows GOARCH=amd64 go vet ./...`;
- the harness;
- the full configured suite once.

Evidence tar under 1 MiB. Attach `_results.md`, `_conformance-matrix.md` and `_producer-evidence.tar.gz`, and read each back. Keep the checklist current, then run `task-board handoff TASK-260830-2h5uv9 --role developer`. Hand off within 3.5 hours. Canonical CLI: /Users/iv/.curator/global/bin/task-board. Model: gpt-6-luna max. Reviewer: claude-opus-5-5 low.
REJECTION PATTERNS THE INDEPENDENT REVIEWER HAS ALREADY CAUGHT ON THIS EPIC (2026-09-17: TASK-260830-1geqhj rev1, TASK-260830-24z2b3 rev2, TASK-260830-219okr rev3 were all routed CHANGES REQUESTED, and between them these six shapes account for every finding). Read this as a checklist you run against your own work BEFORE handoff; each item names what a reviewer will do to your candidate.
(a) Circular expectations. An assertion that compares a production value against a production constant (`got.Code == pkg.SomeUnsupportedCode`) pins nothing: swapping the constant to another registered value with the same exit keeps the suite green. Assert the LITERAL token from the pinned specification text, at the production entry, and make sure the literal actually occurs in a committed test — not only in a comment. Derive every expectation from the spec, never from the implementation.
(b) A rule pinned at the helper instead of the entry. A test that calls the inner helper directly proves the helper; it says nothing about whether the composed production entry still reaches it. Drive every rule through the entry point the conformance matrix names as its call site, and if a rule is implemented TWICE (a Build path and a Decode path, an exported entry and its internal sibling), each side needs its own narrowing mutant — a rule pinned only on one side leaves the other side unmeasured, and reviewers plant exactly there.
(c) Forking a landed gate. Never re-implement what internal/environ, internal/canonicaljson, internal/scalar or another landed owner already gates (string bounds in CHARACTERS not bytes, uint53, digests, UUIDv7, timestamps, sorted-unique arrays, closed-shape decoding, the AX number model that forbids floating point, values at or beyond 2^53 and rounding-and-continuing). Delegate to it and keep only your package-specific rules. Every fork this epic produced became a defect: a byte-measured bound, an extension value rounded at 2^60, nested duplicate members collapsed by a lenient decode.
(d) A census or an inventory standing in for behaviour. A reflection census of field NAMES does not prove that a cached observation cannot authorize; an arm inventory keyed by message text does not prove a condition edit is caught. Drive the input, assert the effect.
(e) Mutant rows that do not measure what they claim. An arm-delete (`&& false`) is not a narrowing; a row patching two sites whose kill comes from one is one measured site reported as two; a plant with no raw log and no subprocess exit is a verdict line, not evidence. Ship one raw log per plant per run, state narrowing vs arm-delete per row, and include an applied harmless SURVIVED control so the harness is proven able to report a survivor.
(f) Prose that outruns the tests. "Composes X", "never admits what decoding refuses", "nothing re-implemented", "no cached sentinel can authorize" are censusable claims: a reviewer greps for the callers and plants the counter-example. Before you write a sentence like that in a doc comment, README, LOGBOOK or TRACEABILITY row, make a test fail without the property. If you cannot, state it as an explicit BOUND instead, naming what is not covered.
Finally: report the coverage ratio you MEASURED, not the one you planned. Two of the three rejected revisions claimed a complete ratio and the reviewer measured a smaller one; an honest smaller number with named gaps is accepted, an inflated one is a finding.
