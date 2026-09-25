Implement TASK-260830-2ya5le (implement-fidelity-report-and-reason-registry), the FIRST leaf of STORY-260830-21bxa3 (fidelity-projection-and-validation-contracts, milestone M4). Your Change Request is `task_delta`. Do NOT edit `internal/traceability/ownership.v0.7.0.json`; the Story's final leaf carries the registry. Record clause-to-test bindings in the package TRACEABILITY.md.

AUTHORITY. The task record's v0.5.0 §13.14.2/§13.14.4 is STALE. Use pinned `internal/specdoc/SPEC.v0.7.0.md` §13.14.2 "Fidelity, projection, and lineage", starting at line 10548: the disposition, reason, profile and strategy vocabularies; `FidelityCounts`; `FidelityDispositionRecord`; the Fidelity Report 1.0.0 table; and the reconciliation rules that follow it. Consult §13.14.4 (10904+) only where it constrains the report. Projection Plan, Read-Back, Validation Report, Migration Checkpoint and Lineage Receipt belong to sibling leaves. Do not implement them. Where the report references them, carry the identity or digest field only. Quote the pinned file.

HOW TO PROVE IT: by construction, and over the whole domain. The last two leaves of this project each lost a revision because a rule was tested at one point, and the reviewer then found the adjacent member of the class:
- 0600/0644 tested, then the single bits, then the bit combinations;
- one prefix length and one namespace tested, then the reviewer tried the others.

This leaf's domains are small and finite, so enumerate them. Every gate gets a test that sweeps its whole domain and compares production to an INDEPENDENT ORACLE. Write the oracle in the test from the spec text; do not copy it from your implementation.
1. **Closed vocabularies.** Dispositions (7), profiles (5), strategies and the complete closed core reason list:
   - every member is admitted;
   - case variants, whitespace variants, near-misses, the empty string and non-string JSON values are refused with a literal code;
   - a reverse-DNS extension reason is admitted but can never equal, shadow or redefine a core code;
   - a malformed reverse-DNS string is refused.
2. **Record rules over the full grid:** disposition × reason-set size (0, 1, 2, 128, 129) × `canonical_object_id` present/null × source evidence present/absent. The oracle is the four rules in the spec:
   - exact requires an empty reason set;
   - every non-exact disposition requires at least one reason;
   - synthesized requires no source canonical object;
   - every non-synthesized row traces to captured source evidence.

   Every length and cardinality bound is tested at the edge and at edge+1. Sorted-unique arrays are tested with an unsorted array and with a duplicate.
3. **Report.** Check:
   - the exact member set;
   - `fidelity_report_id` equals the JCS digest with only that member omitted, recomputed independently in the test;
   - archive/target nullability is branch-exact;
   - rows are ordered with synthesized rows after source rows;
   - every aggregate map (`counts`, `event_kind_counts`, `content_block_counts`, `byte_counts`, `reason_counts`) is derived from the rows;
   - a report whose aggregate disagrees with its rows by one unit, in any single cell, is refused.

   Enumerate which cell you perturb; do not pick one.

For every gate, ship an admitting narrowing and show it KILLED by the named oracle test run ALONE. Include an applied, line-count-preserving neutral control. NOT_APPLIED counts as unmeasured.

CENSUS. Commit a gate × entry table in the conformance matrix, mirrored in TRACEABILITY, with the domain axes above. Every production entry that reaches a gate is its own cell. Follow the developer role's Handoff Preconditions: a coverage map with each surface row pointing to its tests and a killed narrowing.

COMPOSITION. Reuse `internal/canonicaljson` for JCS and digests, and the landed scalar/digest types. If a type you need is missing, stop and name its owner. Build the importer outcome grid keyed `(package, entry, input)` over every package you touch, base vs candidate, and name every moved class.

WORKSPACE. Use the managed Story worktree for STORY-260830-21bxa3 that the spawn provisions. The candidate is the UNCOMMITTED tree. Never commit on the Story branch, never edit `.task-board`, never self-accept, and never write the real index except through the handoff. If trunk moves before your handoff, run `task-board worktree refresh-candidate TASK-260830-2ya5le` first: CR construction requires the checkpoint to descend from current trunk.

PROCESS. Read the `go-testing-tools` skill first. All CI is local. Run:
- the package tests;
- the importer set;
- `GOOS=windows GOARCH=amd64 go vet ./...`;
- the mutation harness;
- the full configured suite once.

Evidence tar under 1 MiB. Attach `TASK-260830-2ya5le_results.md`, `_conformance-matrix.md` and `_producer-evidence.tar.gz`, and read each back off the board. Keep the checklist current, run `task-board handoff TASK-260830-2ya5le --role developer`, and exit. Hand off within 3 hours. Canonical CLI: /Users/iv/.curator/global/bin/task-board. Model: muse-spark max. Reviewer: gpt-6-sol medium.
REJECTION PATTERNS THE INDEPENDENT REVIEWER HAS ALREADY CAUGHT ON THIS EPIC (2026-09-17: TASK-260830-1geqhj rev1, TASK-260830-24z2b3 rev2, TASK-260830-219okr rev3 were all routed CHANGES REQUESTED, and between them these six shapes account for every finding). Read this as a checklist you run against your own work BEFORE handoff; each item names what a reviewer will do to your candidate.
(a) Circular expectations. An assertion that compares a production value against a production constant (`got.Code == pkg.SomeUnsupportedCode`) pins nothing: swapping the constant to another registered value with the same exit keeps the suite green. Assert the LITERAL token from the pinned specification text, at the production entry, and make sure the literal actually occurs in a committed test — not only in a comment. Derive every expectation from the spec, never from the implementation.
(b) A rule pinned at the helper instead of the entry. A test that calls the inner helper directly proves the helper; it says nothing about whether the composed production entry still reaches it. Drive every rule through the entry point the conformance matrix names as its call site, and if a rule is implemented TWICE (a Build path and a Decode path, an exported entry and its internal sibling), each side needs its own narrowing mutant — a rule pinned only on one side leaves the other side unmeasured, and reviewers plant exactly there.
(c) Forking a landed gate. Never re-implement what internal/environ, internal/canonicaljson, internal/scalar or another landed owner already gates (string bounds in CHARACTERS not bytes, uint53, digests, UUIDv7, timestamps, sorted-unique arrays, closed-shape decoding, the AX number model that forbids floating point, values at or beyond 2^53 and rounding-and-continuing). Delegate to it and keep only your package-specific rules. Every fork this epic produced became a defect: a byte-measured bound, an extension value rounded at 2^60, nested duplicate members collapsed by a lenient decode.
(d) A census or an inventory standing in for behaviour. A reflection census of field NAMES does not prove that a cached observation cannot authorize; an arm inventory keyed by message text does not prove a condition edit is caught. Drive the input, assert the effect.
(e) Mutant rows that do not measure what they claim. An arm-delete (`&& false`) is not a narrowing; a row patching two sites whose kill comes from one is one measured site reported as two; a plant with no raw log and no subprocess exit is a verdict line, not evidence. Ship one raw log per plant per run, state narrowing vs arm-delete per row, and include an applied harmless SURVIVED control so the harness is proven able to report a survivor.
(f) Prose that outruns the tests. "Composes X", "never admits what decoding refuses", "nothing re-implemented", "no cached sentinel can authorize" are censusable claims: a reviewer greps for the callers and plants the counter-example. Before you write a sentence like that in a doc comment, README, LOGBOOK or TRACEABILITY row, make a test fail without the property. If you cannot, state it as an explicit BOUND instead, naming what is not covered.
Finally: report the coverage ratio you MEASURED, not the one you planned. Two of the three rejected revisions claimed a complete ratio and the reviewer measured a smaller one; an honest smaller number with named gaps is accepted, an inflated one is a finding.
