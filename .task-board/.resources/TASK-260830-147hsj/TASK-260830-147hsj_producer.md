Implement TASK-260830-147hsj (implement-object-discovery-and-union), the second leaf of STORY-260830-ub60id (M2 anti-entropy). The first leaf, TASK-260830-nxqqaw (namespace Merkle inventories, trie, serving, walk and fetch), is checkpointed on the Story branch. Build on it; do not re-implement it. Your Change Request is `task_delta`. Do NOT edit the traceability registry, because the Story's final leaf carries it. Record your bindings in the package TRACEABILITY.md.

AUTHORITY. The task record's v0.5.0 is stale. Use pinned `internal/specdoc/SPEC.v0.7.0.md`:
- §11.4 union rules 1-7 and "No last-writer-wins rule exists. Timestamps MUST NOT select a winner" (7925-7933);
- §5.3 Lease Record and ownership (1987-2061), including 2047: losing-lease events "preserved in a divergent branch and MUST NOT affect authoritative state";
- §11.5/§11.6 only where union hands off to staging and commit. Model that boundary with a stated bound; do not implement it.

Quote the pinned file.

HOW TO PROVE IT: by construction, over whole domains. Every earlier leaf in this project lost revisions to rules pinned at one point, one entry or one axis. Before writing code, write the AXIS INVENTORY for each gate. For each axis, name the test that varies it, or state a bound with its owner. For an unbounded axis, give a generated range plus a structural argument, never a fixture constant.

1. **Carry-overs from nxqqaw rev3.** The review verdict `TASK-260830-nxqqaw_review-verdict-rev3.md` lists four surviving narrowings on reachable or near-reachable arms of the code you now build on:
   - `fetch-dup-ids`;
   - `b64-roundtrip` (Go base64 skips `\r\n` even in Strict mode);
   - `walk-quarantine` (reachable per the reviewer's probe);
   - `walk-crossns`.

   Pin each one with a committed test and a narrowing that the test kills when run ALONE, or prove the arm unreachable with its reason.
2. **Durable union.** Unknown objects are added only after validation. Identical objects are idempotent, including across crash and restart. Same digest with different bytes quarantines and aborts. Tombstones and acks are unioned, not executed. Every durable mutation carries crash/idempotency evidence.
3. **Arrival-order independence, exhaustively.** For small event sets (N ≤ 6, with competing leases, a losing-lease branch, tombstones, and timestamps perturbed in both directions), enumerate EVERY permutation of arrival order. Assert that the union, the derived lease heads and the rebuilt projection are identical to one reference computed by an independent oracle in the test. Losing-lease events must be preserved and must never affect authoritative state. No timestamp may change the outcome.
4. **Projection rebuild** is a pure function of the unioned set. Prove it with the permutation oracle plus a structural argument: the rebuild never reads wall-clock time or insertion order.

For every gate, ship an admitting narrowing that the named test kills when run ALONE, plus an applied neutral control. Assert literal spec tokens, never a production constant.

COMPOSITION. Build the importer outcome grid keyed `(package, entry, input)` over every package you touch, base vs candidate, and name every moved class.

WORKSPACE. Use the managed Story worktree `.temp/STORY-260830-ub60id/worktree`. The candidate is the UNCOMMITTED tree on top of nxqqaw's checkpoint. Never commit, never edit `.task-board`, never self-accept, never write the real index except through the handoff. If trunk moves before handoff, run `task-board worktree refresh-candidate TASK-260830-147hsj` first. SCRATCH RULE: scratch goes under the worktree's `.temp/TASK-260830-147hsj/` or `t.TempDir()`, never `/tmp`.

PROCESS. Read the `go-testing-tools` skill first. All CI is local. Run:
- the package tests;
- the importer set;
- `GOOS=windows GOARCH=amd64 go vet ./...`;
- the mutation harness;
- the full configured suite once.

Evidence tar under 1 MiB. Attach `_results.md`, `_conformance-matrix.md` and `_producer-evidence.tar.gz`, and read each back. Keep the checklist current, then run `task-board handoff TASK-260830-147hsj --role developer`. Hand off within 3.5 hours. Canonical CLI: /Users/iv/.curator/global/bin/task-board. Model: gpt-6-luna max. Reviewer: claude-opus-5-5 low.
REJECTION PATTERNS THE INDEPENDENT REVIEWER HAS ALREADY CAUGHT ON THIS EPIC (2026-09-17: TASK-260830-1geqhj rev1, TASK-260830-24z2b3 rev2, TASK-260830-219okr rev3 were all routed CHANGES REQUESTED, and between them these six shapes account for every finding). Read this as a checklist you run against your own work BEFORE handoff; each item names what a reviewer will do to your candidate.
(a) Circular expectations. An assertion that compares a production value against a production constant (`got.Code == pkg.SomeUnsupportedCode`) pins nothing: swapping the constant to another registered value with the same exit keeps the suite green. Assert the LITERAL token from the pinned specification text, at the production entry, and make sure the literal actually occurs in a committed test — not only in a comment. Derive every expectation from the spec, never from the implementation.
(b) A rule pinned at the helper instead of the entry. A test that calls the inner helper directly proves the helper; it says nothing about whether the composed production entry still reaches it. Drive every rule through the entry point the conformance matrix names as its call site, and if a rule is implemented TWICE (a Build path and a Decode path, an exported entry and its internal sibling), each side needs its own narrowing mutant — a rule pinned only on one side leaves the other side unmeasured, and reviewers plant exactly there.
(c) Forking a landed gate. Never re-implement what internal/environ, internal/canonicaljson, internal/scalar or another landed owner already gates (string bounds in CHARACTERS not bytes, uint53, digests, UUIDv7, timestamps, sorted-unique arrays, closed-shape decoding, the AX number model that forbids floating point, values at or beyond 2^53 and rounding-and-continuing). Delegate to it and keep only your package-specific rules. Every fork this epic produced became a defect: a byte-measured bound, an extension value rounded at 2^60, nested duplicate members collapsed by a lenient decode.
(d) A census or an inventory standing in for behaviour. A reflection census of field NAMES does not prove that a cached observation cannot authorize; an arm inventory keyed by message text does not prove a condition edit is caught. Drive the input, assert the effect.
(e) Mutant rows that do not measure what they claim. An arm-delete (`&& false`) is not a narrowing; a row patching two sites whose kill comes from one is one measured site reported as two; a plant with no raw log and no subprocess exit is a verdict line, not evidence. Ship one raw log per plant per run, state narrowing vs arm-delete per row, and include an applied harmless SURVIVED control so the harness is proven able to report a survivor.
(f) Prose that outruns the tests. "Composes X", "never admits what decoding refuses", "nothing re-implemented", "no cached sentinel can authorize" are censusable claims: a reviewer greps for the callers and plants the counter-example. Before you write a sentence like that in a doc comment, README, LOGBOOK or TRACEABILITY row, make a test fail without the property. If you cannot, state it as an explicit BOUND instead, naming what is not covered.
Finally: report the coverage ratio you MEASURED, not the one you planned. Two of the three rejected revisions claimed a complete ratio and the reviewer measured a smaller one; an honest smaller number with named gaps is accepted, an inflated one is a finding.
