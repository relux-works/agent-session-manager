Implement TASK-260830-nxqqaw (implement-namespace-merkle-inventories), the FIRST leaf of STORY-260830-ub60id (immutable-anti-entropy-and-inventory, milestone M2). Trunk is `360c8bd`. The Story has no checkpoints yet. Your Change Request is `task_delta`, not story_final: do NOT edit the traceability registry `internal/traceability/ownership.v0.7.0.json`, because the Story's final leaf carries the registry for every leaf. Do record your clause-to-test bindings in the package TRACEABILITY.md, so that the final leaf can transcribe them.

AUTHORITY. The task record names spec v0.5.0 §11.4/§10. That is STALE. Use the pinned `internal/specdoc/SPEC.v0.7.0.md`:
- §11.4 "Anti-entropy union" (lines 7764-7934): the namespace table, node rules 1-5, JCS node encoding with domain `urn:ax:merkle-node:1`, the empty/singleton/branch fixtures and their exact `inventory.children` bodies, MIXED-NS-1, MIXED-NS-EXCHANGE, MIXED-NS-N1, and union rules 1-7;
- §11.3, the `inventory.roots` / `inventory.children` / `objects.get` rows (7555-7561) and the prefix and batching rules (7734-7738);
- §10 for the object classes.
Quote the pinned file. Where v0.7.0 adds namespaces for Mesh RPC ≥3/≥4 (`directory_record`, `terminal_backend_evidence`, §11.8/§11.9), decide whether they are in scope and state it with a citation. The landed `internal/rpcwire.Namespaces(version)` already models that vocabulary.

THE PROPERTIES, each by construction and each driven through a production entry. For each one, commit an admitting narrowing that its named test kills when that test runs ALONE:
1. **Deterministic trie.** One construction function yields every node hash, and the root_id / node_hash on the wire are exactly that value. Every normative fixture reproduces byte-exact through the repository's RFC 8785 implementation (`internal/canonicaljson`): the three root fixtures, both child hashes of the branch fixture, the three exact `inventory.children` bodies, and all six MIXED-NS-1 roots. A duplicate at 64 nibbles is an integrity failure, not a silent dedupe.
2. **Total, disjoint membership.** Every row of the §11.4 table maps to exactly one namespace or to excluded, and the mapping is decided by the validated schema/byte class and nothing else. MIXED-NS-N1 must fail both the expected roots and schema-to-namespace validation. `objects.get` rejects an ID whose schema maps to another requested namespace. Excluded local objects never move a root or a count.
3. **Bounded serving.** `inventory.roots` and `inventory.children` enforce the §11.3 shapes: sorted unique namespaces 1..6, prefix 0..64 lowercase hex, children 0..16, ids 0..1. A valid prefix that is not materialized returns `not_found` and never an invented empty child. `objects.get` batching stays within the negotiated limit. COMPOSE with the landed `internal/rpcwire` validators; do not write a second validator that can drift from them. If rpcwire's contract is insufficient, stop and name that owner.
4. **Union exchange.** Drive MIXED-NS-EXCHANGE end to end, in process. Only the record, manifest and blob roots differ. The recursive walk plus `objects.get` fetches exactly the missing Checkpoint and Descriptor. The rules hold:
   - unknown objects are added only after validation;
   - an identical object is idempotent;
   - same digest with different bytes quarantines and aborts;
   - tombstones and acks are unioned, not executed;
   - no timestamp selects a winner;
   - referenced blob transfer starts only after record union.

   If blob transfer or materialization is owned by another leaf or Story (§11.5/§11.6), model it at the boundary with a stated bound and an owner. Do not fake it.

CENSUS, FROM THIS FIRST BRIEF. Commit a gate × entry table in the conformance matrix, mirrored in TRACEABILITY, with an AXIS enumeration per gate:
- namespace;
- prefix length 0/1/63/64/65 and non-hex/uppercase;
- count 0/1/2/duplicate;
- id order and duplicates in requests;
- schema class per namespace row.

Each cell holds one of: a named test plus the narrowing it kills; `unreachable` with a reason; or a stated bound with an owner. Past leaves lost 7 to 11 revisions to rules pinned at one entry or along one axis. Do not make the reviewer find the next one. Assert literal spec tokens, never a production constant.

COMPOSITION. Build runtime outcome grids keyed on `(package, entry, input)` over the complete importer set of every package you touch, including `internal/rpcwire` if you change it, on both the base and the candidate. Name every moved class.

DETERMINISM. No network and no clocks. Keep fixtures table-driven.

WORKSPACE. The managed Story worktree for STORY-260830-ub60id is the one the spawn provisions. The candidate is the UNCOMMITTED tree. Never commit on the Story branch, never edit `.task-board`, never self-accept. Never write the worktree's real index except through the handoff; compare through a scratch `GIT_INDEX_FILE`.

PROCESS. Read the `go-testing-tools` skill first. All CI is local. Run:
- package tests;
- the importer set;
- `GOOS=windows GOARCH=amd64 go vet ./...`;
- the mutation harness;
- the full suite once, from `spawn.worktree_isolation.validation.commands` (command 5 is the 25-minute `-race` run).

Resolve artifact paths ABSOLUTELY. Keep each evidence tar under 1 MiB. Attach `TASK-260830-nxqqaw_results.md`, `_conformance-matrix.md` and `_producer-evidence.tar.gz`, and verify each by reading it back off the board. Keep the checklist current, then run `task-board handoff TASK-260830-nxqqaw --role developer` and exit. If trunk moves while you work, use `task-board worktree refresh-candidate TASK-260830-nxqqaw` yourself: it is the producer's command. Hand off no later than 3.5 hours in. Canonical CLI: /Users/iv/.curator/global/bin/task-board. Model: gpt-6-luna max. Reviewer: claude-opus-5-5 low.
REJECTION PATTERNS THE INDEPENDENT REVIEWER HAS ALREADY CAUGHT ON THIS EPIC (2026-09-17: TASK-260830-1geqhj rev1, TASK-260830-24z2b3 rev2, TASK-260830-219okr rev3 were all routed CHANGES REQUESTED, and between them these six shapes account for every finding). Read this as a checklist you run against your own work BEFORE handoff; each item names what a reviewer will do to your candidate.
(a) Circular expectations. An assertion that compares a production value against a production constant (`got.Code == pkg.SomeUnsupportedCode`) pins nothing: swapping the constant to another registered value with the same exit keeps the suite green. Assert the LITERAL token from the pinned specification text, at the production entry, and make sure the literal actually occurs in a committed test — not only in a comment. Derive every expectation from the spec, never from the implementation.
(b) A rule pinned at the helper instead of the entry. A test that calls the inner helper directly proves the helper; it says nothing about whether the composed production entry still reaches it. Drive every rule through the entry point the conformance matrix names as its call site, and if a rule is implemented TWICE (a Build path and a Decode path, an exported entry and its internal sibling), each side needs its own narrowing mutant — a rule pinned only on one side leaves the other side unmeasured, and reviewers plant exactly there.
(c) Forking a landed gate. Never re-implement what internal/environ, internal/canonicaljson, internal/scalar or another landed owner already gates (string bounds in CHARACTERS not bytes, uint53, digests, UUIDv7, timestamps, sorted-unique arrays, closed-shape decoding, the AX number model that forbids floating point, values at or beyond 2^53 and rounding-and-continuing). Delegate to it and keep only your package-specific rules. Every fork this epic produced became a defect: a byte-measured bound, an extension value rounded at 2^60, nested duplicate members collapsed by a lenient decode.
(d) A census or an inventory standing in for behaviour. A reflection census of field NAMES does not prove that a cached observation cannot authorize; an arm inventory keyed by message text does not prove a condition edit is caught. Drive the input, assert the effect.
(e) Mutant rows that do not measure what they claim. An arm-delete (`&& false`) is not a narrowing; a row patching two sites whose kill comes from one is one measured site reported as two; a plant with no raw log and no subprocess exit is a verdict line, not evidence. Ship one raw log per plant per run, state narrowing vs arm-delete per row, and include an applied harmless SURVIVED control so the harness is proven able to report a survivor.
(f) Prose that outruns the tests. "Composes X", "never admits what decoding refuses", "nothing re-implemented", "no cached sentinel can authorize" are censusable claims: a reviewer greps for the callers and plants the counter-example. Before you write a sentence like that in a doc comment, README, LOGBOOK or TRACEABILITY row, make a test fail without the property. If you cannot, state it as an explicit BOUND instead, naming what is not covered.
Finally: report the coverage ratio you MEASURED, not the one you planned. Two of the three rejected revisions claimed a complete ratio and the reviewer measured a smaller one; an honest smaller number with named gaps is accepted, an inflated one is a finding.
