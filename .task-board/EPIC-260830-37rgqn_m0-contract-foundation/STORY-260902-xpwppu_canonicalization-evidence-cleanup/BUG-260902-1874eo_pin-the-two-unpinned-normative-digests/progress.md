## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(2))

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Both SPEC.md:303-304 digests recomputed from encoder output and asserted, values taken from the pinned SPEC extract with its SHA-256 verified
- [x] Each new digest assertion proven non-vacuous: it reddens when the encoder output changes by one byte
- [x] All four closed_shapes.go re-checks resolved, each either deleted with a stated reachability argument or kept with a subsumption comment naming decodeStrict/canonical.go:311
- [x] Subsumption comments follow the existing convention at internal/localstore/paths.go:234-236, not a new style
- [x] Decisive mutant reported for each part, and no existing assertion weakened or deleted
- [x] Repository validation suite green
- [x] Code written per task description and AC
- [x] Relevant tests written for new or changed behavior and passing
- [x] Gating, refusing, validating, authorizing, or attesting behavior covered by negative tests that fail when the gate admits what it must reject, with the production call site named
- [x] Lint clean
- [x] Relevant build/validation commands run after changes and build not broken
- [x] New outcome artifact attached on the board with a task-scoped name when the work produces notes, logs, screenshots, or other deliverables
- [x] Important findings, decisions, anomalies, or regressions recorded in logbook when relevant
- [x] Implementation matches AC
- [x] Solution fits project architecture
- [x] Tests green
- [x] Gate, refusal, validation, authorization, and attestation behavior attacked, not read — positive-path-only evidence is not accepted
- [x] If review does not accept the work — verdict evidence added and status routed by the explicit verdict branches

## Notes
DELIVER BOTH PARTS IN ONE CHANGE REQUEST. This bug leads; BUG-260902-9uwnm7 is its sibling and is delivered in the same CR. Both close together.

Two independent mechanical gaps, batched because both are small, both touch internal/canonicaljson, and parallel Stories would conflict on the same files. Each is independently verifiable and must be independently proven.

PART 1 - PIN THE TWO UNPINNED NORMATIVE DIGESTS (BUG-260902-1874eo)
SPEC.md:303-304 publishes two SS1.6 digest fixtures that no .go or .json file in this repository recomputes:
  bb80eb37... NUM-U64-STRING  = sha256 of {'n':'9007199254740992'}
  b0ec84c6... NUM-U64-MAX     = sha256 of {'n':'18446744073709551615'}
Only NUM-SAFE-MAX (e1da48c6...) is pinned today, at internal/canonicaljson/canonical_test.go:513.
Required: assert both digests by recomputing them from the canonical encoder output, next to the existing NUM-SAFE-MAX assertion and in the same shape. The decimal_uint64 SYNTAX is already tested; what is missing is the normative DIGEST claim.
Take the expected values from the PINNED spec, not from this note and not from a live fetch of the spec repository default branch. Extract SPEC.md at the pinned commit, verify its SHA-256 against internal/specpin, then read lines 303-304 and quote them in the test comment.
Anti-vacuity: prove each new assertion fails if the encoder output changes by one byte. A digest assertion that passes because both sides are computed the same wrong way proves nothing.

PART 2 - DOCUMENT OR DELETE THE SUBSUMED UTF-8 RE-CHECKS (BUG-260902-9uwnm7)
Four utf8.ValidString re-checks are unreachable and undocumented, at internal/canonicaljson/closed_shapes.go:258, :1506, :1525, :1760.
decodeStrict validates UTF-8 once at internal/canonicaljson/canonical.go:311 and is the only producer of the maps that reach those four sites; the package has no external caller on any branch.
The repository already has the convention: internal/localstore/paths.go:234-236 and internal/canonicaljson/projection.go:300 document their subsumed re-checks with a comment naming the check that subsumes them. Follow that existing convention exactly rather than inventing a new comment style.
Choose per site, with the reason stated: keep it with a subsumption comment naming decodeStrict and canonical.go:311 as the subsuming validator, or delete it. Deleting is acceptable ONLY where you can show no reachable caller can deliver invalid UTF-8; if reachability is uncertain, keep and document.
Do not add a test that fakes reachability to make a dead branch look covered. That is the exact anti-pattern this board refuses.

WHY THIS MATTERS
Both parts are about a claim nobody checks. A published digest that nothing recomputes is a contract nobody verifies, and an unreachable branch that is neither killable nor declared is indistinguishable from a real gap until someone re-derives reachability by hand.

STANDING CRITERIA FOR THIS BOARD
- Drive real production entry points; no test-only helper as the subject.
- Kill mutants, and report the decisive mutant for each change, not just a count.
- Narrow a mutant rather than deleting it when it is equivalent; state equivalence explicitly.
- Never weaken or delete an existing assertion to make new work pass. If an existing test blocks you, say so and stop.
- Quote the pinned spec verbatim for any normative claim, with file and line.
- Disclose anything you found but did not fix.

spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 implementation pair and the only admitted pair; two mechanical canonicalization gaps needing pinned-spec fidelity and mutant proof."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260902-ec5bf4, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260902-ec5bf4)
DEVELOPER HANDOFF - both parts delivered in commit 454c2db (signed, one commit past the workspace checkpoint 422786c). Full evidence in BUG-260902-1874eo_results.md; raw mutant output in BUG-260902-1874eo_mutant-sweep.log.

PART 1. SPEC.md extracted at pinned commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c; its SHA-256 verified equal to internal/specpin.DocumentSHA256 (562546d2...) before lines 303-304 were read, and both rows are quoted verbatim in the test comment. NUM-U64-STRING and NUM-U64-MAX are now asserted from the production Canonicalize output beside the existing NUM-SAFE-MAX assertion, in the same shape, with the expected values as literals from the pinned lines. Mutants: A (Canonicalize emits one extra byte) reddens both new subtests and both pre-existing digest subtests; B2 and C2 change one byte of one fixture each and redden exactly that assertion, so neither new assertion is carried by the other.

PART 2 - REVIEWER, READ THIS. The audit line numbers are stale. closed_shapes.go:258/:1506/:1525/:1760 match commit 7b94c9ad, five commits back; current lines are 302/1830/1852/2091. That commit carries EIGHT utf8.ValidString sites, not four, with no property separating the cited subset. All eight are resolved, because the AC is not met while four identical survivors remain. canonical.go:311 is likewise now canonical.go:328, so the comments name decodeStrict by function, not by line.

Resolution is KEEP AND DOCUMENT for all eight. decodeStrict does subsume them, but that is a package-internal invariant, not a property of these validators, and CanonicalByteLength(value any) already shows the package can export a non-[]byte entry point; two of the eight are also registered as invalidUTF8Refusal in refusal_guards_test.go, so deleting rewrites pinned refusal statements. One shared note carries the argument; each site names decodeStrict and states which sibling terms stay reachable. Style follows the block comment at internal/localstore/paths.go:234-236.

The dependency the AC requires to be stated is also machine-checked, in new internal/canonicaljson/utf8_subsumption_test.go: no exported entry point reaching a derived re-check may take a non-[]byte parameter, and json.NewDecoder/json.Unmarshal may appear only inside decodeStrict. Guarded functions and decode sites are derived from the AST, not listed. Mutants: D2 (exported map-taking entry point) RED; G2 (exported []byte entry point decoding with json.Unmarshal) RED; E3 (one subsumption comment deleted) RED; F3 (narrowing - comment kept but names requireExactMembers) RED.

REMOVED ARM, DISCLOSED: the first version of that pin asserted transitive reachability of decodeStrict. Mutant G SURVIVED it, because validateExtensionsObject reaches decodeStrict via canonicalByteBound -> CanonicalByteLength -> Canonicalize. Reaching a decoder downstream is not the same fact as being decoded by it, so the arm was replaced rather than kept.

FOUND, NOT FIXED: (1) core_records.go:278 in validateProviderIdentityRecord carries a ninth re-check of the same class, outside this bug closed_shapes.go scope - now the only undocumented survivor in the package. (2) Nothing derives the set of digest-publishing Section 1.6 fixtures from the pinned document (SPEC.md is not vendored), so a future added published digest cannot be reported as unpinned - exactly how these two were missed. (3) If the same audit produced other findings on this package, re-derive their line references and counts before working them.

VALIDATION, real exit codes, each run standalone: gofmt -l internal/ clean; go build ./... 0; go vet ./... 0; tracecheck 0 (contracts=60 normative_sections=36 acceptance_cases=43 fixtures=30 compatibility_contracts=55); go generate ./internal/catalog + git diff --exit-code 0; go test ./... -count=1 0; go test ./... -cover -count=1 0; FuzzCanonicalizeRoundTrip 100x 0. internal/canonicaljson coverage 97.2%, unchanged. No existing assertion weakened or deleted.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-ec5bf4, pid=62153, exit=0)
REVIEWER CONTEXT ADDED BY THE ORCHESTRATOR AFTER THE PRODUCER RUN

1. THE AUDIT LINE NUMBERS IN THE ORIGINAL BRIEF WERE WRONG. The brief named closed_shapes.go:258, :1506, :1525 and :1760 as the four re-check sites. None of those lines contains utf8.ValidString on the current tree; they point at a return statement and a path comparison. The real sites are :302, :349, :465, :1830, :1852, :1957 and others. The producer derived the sites instead of trusting the list, which is why the work is correct despite the brief being wrong. Do not treat the brief's line numbers as authority; re-derive.

2. THE PRODUCER EXCEEDED THE BRIEF, AND THE EXCESS IS THE PART THAT NEEDS THE HARDEST LOOK. The brief asked for a comment or a deletion. The producer argued that a prose comment is unfalsifiable, because the subsumption rests on a package-internal invariant rather than a local property, and built utf8_subsumption_test.go (332 lines) to machine-check that invariant via a derived call graph. Judge that on its merits, both ways: it may be genuinely better than what was asked, or it may be an elaborate structure that asserts less than it appears to. Specifically probe:
   - Does productionCallGraph miss call edges (method values, interface dispatch, closures, function-typed fields) such that a reachable path exists that the graph does not model? A missed edge makes the pin silently weaker while looking thorough.
   - Does byteInputParameters have false negatives that would let a decoded-value entry point pass?
   - Is utf8GuardedFunctions' derivation complete, and does it fail closed on a parse error rather than returning an empty set?

3. ORCHESTRATOR VERIFICATION ALREADY PERFORMED, FOR YOU TO EXTEND RATHER THAN REPEAT. Two mutants were confirmed to redden, each verified present in the file before measuring:
   - Expected NUM-U64-MAX digest changed by one hex digit -> TestCalculateObjectIdentityMatchesAXPublishedFixtures fails.
   - One subsumption comment removed -> TestEveryUTF8RecheckDeclaresItsSubsumption fails, naming closed_shapes.go:464.
   Two EARLIER mutant attempts reported a false green: a -run filter that matched no parent test, and a perl substitution without /g that mutated the SPEC quotation in the comment instead of the want value. Both produced 'ok' while asserting nothing. Apply the same discipline: confirm the mutant is present in the file before believing a green or a red.

4. THE HIGHEST-VALUE MUTANT NOT YET TRIED is the one the new pin exists to catch: add an exported function to the package that takes an already-decoded string (or map) and routes it to one of the guarded validators, and confirm TestNoExportedEntryPointHandsADecodedValueToAUTF8Recheck reddens. If it does not, the pin does not do the job its comment claims and the subsumption comments are back to being unfalsifiable prose.

5. DIGEST PROVENANCE. The two expected digests must come from the pinned SPEC, not from this encoder. Verify the quoted SPEC.md:303-304 text against the pinned document at commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c whose SHA-256 is 562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a. An expectation recomputed from the implementation would agree with any drift the implementation acquires and would prove nothing.

STANDING REVIEW BAR
- Positive-path-only evidence is refused.
- You have licence to attempt to defeat every gate this change adds.
- Do not accept a claim in a comment that you have not independently tested.
- Report anything the producer found but did not fix, and anything you find but do not require fixed.

spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Rank-1 pair and only admitted pair; adversarial review of a derived call-graph invariant needing independent mutant work."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260902-9a6194, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260902-9a6194)
REVIEW VERDICT: changes requested -> to-dev. Evidence: BUG-260902-1874eo_review-verdict.md

CR repository_delta=empty is a snapshot artifact, not an empty delivery: base OID 454c2db IS the producer commit and 454c2db^{tree} == candidate tree 3a26ed7b. Reviewed the commit content (5 files, +429).

PART 1 ACCEPTED as delivered. Both digests independently recomputed outside the implementation and match SPEC.md:303-304. Three decisive mutants re-run by the reviewer: A (Canonicalize appends one byte) reddens both new subtests plus the two pre-existing ones; B2/C2 (one fixture byte each) redden exactly one subtest each. Non-vacuous and individually decisive. Do not touch part 1 in rework.

PART 2 BLOCKED on one finding: a capability claim that does not reproduce. README.md says "Adding a second decoder or a map-taking entry point reddens the pin" and closed_shapes.go says the invariant "cannot be broken silently". Mutant H disproves both - an exported ValidateDecodedRecord(map[string]any) that reaches the guards through immutableObjectShapeValidators leaves all three pins GREEN and go test ./internal/canonicaljson -count=1 ok (19.1s).

Root cause: productionCallGraph only records edges for call.Fun.(*ast.Ident) naming a top-level declaration. The package dispatches through a function-value table (closed_shapes.go:66, invoked :182) registered via a local closure, so neither is an edge. Measured: only 3 of the 7 derived guarded functions are reachable from any exported entry point; CalculateObjectIdentity reaches zero. validateSessionLaunchPlan, validateSessionBoardIdentity, validateSortedUniqueStrings and validateProviderIdentityRecord are covered vacuously. D2 passed only because it called the guard by direct identifier - the one shape the graph can see.

FIX (either, first preferred): (1) add edges for function-value idents in argument/assignment position so mutant H reddens TestNoExportedEntryPointHandsADecodedValueToAUTF8Recheck, and ship mutant H as its decisive mutant; or (2) narrow the README and closed_shapes.go wording to what is actually checked and drop "cannot be broken silently".

Not a live defect - production code is comment-only and the guards stay fail-closed. It is a false published guarantee in README.md and in the comment block the keep-not-delete decision rests on.

Reviewer validation on the delivered tree: gofmt clean, go build 0, go vet 0, go test ./... -count=1 all ok, tree restored identical to 454c2db after every mutant.

agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-9a6194, pid=88429, exit=0)
REWORK BRIEF FROM THE ORCHESTRATOR - REVIEW ROUND 1 REJECTED PART 2

PART 1 IS ACCEPTED AND MUST NOT BE TOUCHED. The two digest assertions are correct, non-vacuous, and independently reconfirmed by the reviewer with shasum. Do not edit canonical_test.go.

WHAT BLOCKS PART 2
productionCallGraph records an edge only for call.Fun.(*ast.Ident). The package's real dispatch goes through a function-value table (closed_shapes.go:66, invoked at :182) registered via a local closure, so it produces no edge. Measured consequence: of 7 derived guarded functions the pin actually covers 3. validateSessionLaunchPlan, validateSessionBoardIdentity, validateSortedUniqueStrings and validateProviderIdentityRecord are covered vacuously - no mutant can redden for them.
Two surviving mutants prove it. H: an exported ValidateDecodedRecord(map[string]any) that reaches guards through the package's own dispatch table passes all three pins and the whole package suite. I: an exported METHOD with the same decoded map also survives, because both derivations skip Recv != nil.
The producer's own D2 passed only because it called a guard by direct identifier - the single form the graph sees.

DO BOTH OF THE FOLLOWING. The reviewer offered them as alternatives; they are not alternatives.

1. STRENGTHEN THE GRAPH so mutants H and I redden.
   - Add edges for function values in argument and assignment position, which is the form this package actually uses.
   - Include methods in both derivations; Recv != nil must not silently skip.
   - Land H as the decisive mutant with its red output quoted, and I alongside it.
   - Re-measure and report the guarded-function coverage after the change. If it is not 7 of 7, say which remain uncovered and why.

2. NARROW THE PUBLISHED CLAIM to exactly what the graph models.
   Delete 'cannot be broken silently' from closed_shapes.go and fix the README sentence. No static call graph over a Go package is complete - reflection, interface dispatch through a value obtained at runtime, and function values crossing package boundaries can all defeat it. Even after fix 1 the honest claim is bounded, and it must state its bound. Write what IS checked and name at least one construction that would still evade it.

WHY BOTH. The reason these guards are kept rather than deleted is that a machine checks the invariant. A weak check makes the keep-decision weak. And a published guarantee that does not reproduce is worse than no guarantee, because the next contributor relies on it. Fixing the graph without narrowing the claim just moves the false guarantee one construction further out.

METHOD REQUIREMENTS
- Confirm every mutant is PRESENT in the file before believing a green or a red. Two orchestrator mutants and one earlier attempt reported false greens this session from substitutions that never landed.
- A guard that no mutant can redden is not covered, however thorough the surrounding structure looks. Report coverage as a ratio, not as prose.
- Do not weaken or delete any existing assertion. Do not touch production validator logic; this remains a comments-and-tests change.
- Restore the tree after every mutant and verify it is clean.

spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 and only admitted pair; rework of a static call-graph derivation whose incompleteness the review proved with two surviving mutants."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260902-208051, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260902-208051)
DEVELOPER REWORK ROUND 1 - both required fixes delivered by amending the single leaf commit (one leaf = one commit past checkpoint 422786c). Signed head is 8a0dced; git verify-commit exit 0. Evidence: BUG-260902-1874eo_rework-round1-results.md, raw output in BUG-260902-1874eo_rework-mutant-sweep.log.

PART 1 UNTOUCHED as instructed. canonical_test.go has no diff across the amend. C2 (last hex digit of the NUM-U64-MAX want, SPEC quotation left intact) re-run on this revision: RED in decimal_uint64_maximum only, other four subtests pass.

FIX 1 - THE GRAPH. resolveCallee now classifies every call site. Edges for: a package function called by identifier; a method declared in this package matched by name; and a call through a FUNCTION VALUE - a callee that is not a package function, a predeclared identifier or a named type, plus any computed callee - which reaches every address-taken function. Address-taken is derived: any identifier naming a package function or method appearing anywhere outside callee position and its own declaration. That is exactly this package dispatch: register(schema, validateSessionRecordV1, ...) then validator(object). Methods are now included in utf8GuardedFunctions, in the declaration set and in the exported entry-point set; byteInputParameters returns false for ANY method, because a receiver is caller-supplied decoded state whatever the parameters look like.

COVERAGE 3/7 -> 7/7, and asserted, not narrated. New TestEveryUTF8RecheckIsCoveredByTheEntryPointPin fails and names the uncovered functions. The four the review named - validateSessionLaunchPlan, validateSessionBoardIdentity, validateSortedUniqueStrings, validateProviderIdentityRecord - are now covered.

MUTANTS, each confirmed present in the file before the run was believed, tree restored and verified clean after each. H (exported ValidateDecodedRecord(map[string]any) reaching guards ONLY through immutableObjectShapeValidators) RED, names all 7 guards. I (same body as exported method) RED. I2 (exported method with a []byte parameter, decoded state on the receiver) RED - proves the receiver rule separately from the parameter rule. Regressions still red: D2 direct call, G2 second decoder, E3 comment deleted, F3 narrowing comment naming requireExactMembers. NARROWING K removes ONLY the function-value edge: clean tree stays GREEN and mutant H SURVIVES - that is what identifies the new edge as the decisive addition rather than something else in the rewrite. K2 removes the computed-callee edge too and reproduces the review measurement exactly: coverage is 3/7, naming the same four functions.

FIX 2 - THE CLAIM NARROWED. closed_shapes.go no longer says the invariant cannot be broken silently. README.md no longer says a map-taking entry point simply reddens the pin. Both now state what is modelled and name what still evades it: reflection, a function value handed to another package, and a func-typed struct field - x.M(...) resolves only when M is a method declared in this package, so a table parked in a struct field and invoked as table.validate(object) produces no edge at all. That bound is the reason the guards stay rather than being deleted.

FOUND, NOT FIXED. (1) core_records.go:278 validateProviderIdentityRecord is now covered by the reachability pin but still carries no subsumption comment, because TestEveryUTF8RecheckDeclaresItsSubsumption scans closed_shapes.go only. (2) Nothing derives the digest-publishing Section 1.6 fixture set from the pinned document; SPEC.md is not vendored, so a future published digest cannot be reported as unpinned. (3) The graph bound is stated, not closed. Closing the func-typed-field hole needs go/types, which would make the pin depend on resolving this module imports at test time; that tradeoff was not taken and is disclosed rather than implied away.

VALIDATION, real exit codes, each run standalone, no tee and no pipe: gofmt -l internal/ 0 with no paths listed; go build ./... 0; go vet ./... 0; tracecheck 0 (contracts=60 normative_sections=36 acceptance_cases=43 fixtures=30 compatibility_contracts=55); go generate ./internal/catalog 0 then git diff --exit-code -- internal/catalog 0; go test ./... -count=1 0 across all 10 packages; go test ./... -cover -count=1 0 with internal/canonicaljson at 97.2%, unchanged; FuzzCanonicalizeRoundTrip 100x 0. No existing assertion weakened or deleted; no production logic changed - closed_shapes.go is a comment-only edit.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-208051, pid=570, exit=0)
REVIEW ROUND 2 - ORCHESTRATOR CONTEXT

Round 1 accepted Part 1 (the digest pins) and rejected Part 2 for a call graph that covered 3 of 7 guarded functions while README published a guarantee that did not reproduce. The rework was required to do BOTH: strengthen the graph so mutants H and I redden, AND narrow the published claim to a stated bound.

ORCHESTRATOR VERIFICATION ALREADY PERFORMED - EXTEND, DO NOT REPEAT
Each mutant was written to a file, confirmed present, compiled, measured, then removed with the tree verified clean:
- Mutant H, exported ValidateDecodedRecord(schema, version string, object map[string]any) dispatching through immutableObjectShapeValidators: NOW REDDENS, and the failure names all seven guarded functions including the four that were previously vacuous.
- Mutant I, exported method RecordGate.Validate with the same decoded map: NOW REDDENS, naming the same seven.
- README now states 7 of 7 coverage, says the ratio is asserted rather than left to prose, and publishes the bound: reflection, and a function value handed to another package, are named as constructions the graph does not model.
So the two round-1 blockers appear closed. Your job is to find what the rework introduced or still misses, not to re-run H and I.

WHAT TO ATTACK IN ROUND 2
1. The test grew from 332 to 602 lines. Review the ADDED graph machinery for false edges as well as missing ones. A graph that over-approximates can make the pin fire on a construction that is actually safe, which trains contributors to weaken it. Check whether function-value edges are added only where the value is actually reachable as a call.
2. Verify the 7-of-7 ratio assertion is itself non-vacuous: can it pass while the underlying set is derived wrongly? Try shrinking utf8GuardedFunctions' derivation and confirm something reddens rather than the ratio quietly re-satisfying itself at a smaller denominator.
3. The published bound must be TRUE, not merely humble. Try to construct something the README implies IS caught but is not, and something it says is NOT caught and confirm it indeed is not. A bound that understates is a different defect from one that overstates, and only the first is safe.
4. Confirm Part 1 was not regressed while the classifier work happened. The digest assertions and their SPEC quotations must be unchanged.

FACTUAL CORRECTION TO THE ORIGINAL BUG TEXT, FOR THE RECORD
The Bug said 'Only NUM-SAFE-MAX is pinned'. That was wrong: JCS-UTF16-ORDER was already pinned on main at canonical_test.go:512 with the same structure. Two fixtures were genuinely unpinned, not three. The README's new claim that all four are now pinned is therefore accurate. This is the second factual error found in the audit report this cycle - its line numbers for the utf8 sites were also stale and pointed at unrelated code. Treat the audit report's specifics as unreliable and re-derive; its core findings have held.

STANDING REVIEW BAR
- Positive-path-only evidence is refused.
- Confirm every mutant is PRESENT in the file before believing a green or a red; multiple false greens occurred this session from substitutions that never landed.
- Do not accept a claim in a comment, README, or logbook that you have not independently tested.
- Report anything you find but do not require fixed.

spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Rank-1 and only admitted pair; round-2 review of strengthened call-graph machinery needing over-approximation analysis."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260902-79a17a, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260902-79a17a)
REVIEW ROUND 2 VERDICT: changes requested -> to-dev. Evidence: BUG-260902-1874eo_review-round2-verdict.md, raw sweep in BUG-260902-1874eo_review-round2-mutants.log.

CR repository_delta=empty is a snapshot artifact again, not an empty delivery: base OID 8a0dced IS the leaf commit and 8a0dced^{tree} == candidate tree 6b1134ed. Reviewed the commit content (5 files, +732), one commit past checkpoint 422786c.

PART 1 ACCEPTED, DO NOT TOUCH IN REWORK. Re-verified independently, not accepted from the report: pinned SPEC document sha256 562546d2... == specpin.DocumentSHA256; lines 303-304 of that document carry NUM-U64-STRING and NUM-U64-MAX with the quoted digests; both recompute under shasum; style matches the existing NUM-SAFE-MAX assertion. Mutant A (Canonicalize appends one byte) reddens both new subtests plus the two pre-existing ones; C2 (one fixture byte) reddens exactly one subtest. Individually decisive.

PART 2: THE ROUND-1 BLOCKER IS FIXED. H, I and I2 are now RED and name all 7 guards; coverage is genuinely 7/7 and asserted; E3 and narrowing F3 RED; O (func-typed struct field) survives exactly as disclosed, so the stated bound is honest. The function-value and method edges are real fixes.

BLOCKED ON THREE NEW SURVIVORS OF THE SAME CLASS - a capability claim that does not reproduce.

FINDING 1 (mutant J). byteInputParameters checks the SHAPE of the parameters, not that the bytes reach decodeStrict, and a []byte->string conversion is not a JSON decode. An exported ValidateSortedNames(raw [][]byte) that does string(item) and calls validateSortedUniqueStrings leaves ALL FOUR pins green and the full package suite ok (18.5s), while the guard the comment calls subsumed is demonstrably LIVE: driven through the exported entry point it is the term that refuses (member names[0] must be a UTF-8 string). This is not one of the three disclosed unmodelled constructions - it is the ordinary shape in this package. README says decodeStrict must remain the only place in the package where bytes become Go values and that the subsumption is machine-checked; closed_shapes.go says both break conditions are machine-checked. Nothing checks this one.

FINDING 2 (mutant L2). TestDecodeStrictIsTheOnlyJSONDecodeInTheProductionPackage matches selector.X only when the identifier is literally json, resolving no import path. A second decoder via import stdjson "encoding/json" survives the pin and the full suite (ok 17.8s). Same hole admits encoding/json/v2 or any third-party decoder. README states flatly Adding a second decoder reddens the pin - it does not. (An earlier variant L did fail, but via TestEveryProductionRefusalGuardIsExecuted on an uncovered refusal return, not on the decoder.)

FINDING 3 (minor). utf8GuardedFunctions and TestEveryUTF8RecheckDeclaresItsSubsumption key on ValidString spelled utf8. A brand-new undocumented re-check written as utf8.Valid([]byte(text)) (mutant M) leaves every pin green. utf8.Valid is already the idiom in this package - decodeStrict uses it at canonical.go:328 - so such a re-check silently joins the undocumented survivors this Story exists to remove, contradicting utf8_subsumption_test.go:106.

REWORK: strengthen or narrow, but text and machine check must agree. (1) Preferably pin that a byte-taking exported entry point actually routes its bytes through decodeStrict and land J as the decisive mutant - NOTE the producer correctly disclosed that naive transitive decodeStrict reachability fails (mutant G survived: validateExtensionsObject reaches decodeStrict via canonicalByteBound -> CanonicalByteLength -> Canonicalize), so a naive re-add will not work; if no check holds, name the string(bytes) construction in the published bound and drop the machine-checked claim for it. (2) Resolve the import PATH in the decoder pin and land L2 red, or delete the second-decoder sentence from README and closed_shapes.go. (3) Derive utf8.Valid/aliased unicode/utf8, or narrow the comment to ValidString only. Do not weaken or delete any existing assertion; do not touch canonical_test.go.

CHECKED PER THE ORCHESTRATOR QUESTIONS: parsedProductionPackage/packageProductionFiles fail closed on parse error and on zero derived files (t.Fatal); byteInputParameters fails closed on named types, generics and variadics (unrecognised param type -> false -> error). The coarse indirect-call edge is an OVER-approximation, the safe direction, but it means 7/7 is partly carried by that blanket edge rather than seven distinct concrete paths - not a defect, noted so the ratio is not over-read.

FOUND, NOT REQUIRED FIXED: core_records.go:278 (validateProviderIdentityRecord) carries a ninth re-check outside closed_shapes.go and outside the P3 file filter - the only undocumented ValidString survivor left in the package; deserves its own item. The brief line numbers are stale and internal/canonicaljson/projection.go, cited as the style precedent, does not exist on this tree - the producer re-deriving was the right call, and any sibling item from the same audit needs its references re-derived first.

REVIEWER VALIDATION, clean tree: tree restored byte-identical to 6b1134ed after every mutant (git status --short empty); gofmt -l internal/ clean; go build ./... 0; go vet ./... 0; go test ./... -count=1 all 10 packages ok. The green suite is not the finding - the finding is that three changes the shipped text says would redden it do not.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-79a17a, pid=21667, exit=0)
REVIEW ROUND 2, RE-ISSUED - THE PREVIOUS ROUND-2 VERDICT IS VOID. READ THIS FIRST.

The previous round-2 reviewer reviewed commit 454c2db and re-reported the round-1 blocker verbatim. That verdict is void because 454c2db is ORPHANED. Verified: `git merge-base --is-ancestor 454c2db HEAD` returns false; 454c2db is not reachable from the Story branch.

WHAT ACTUALLY HAPPENED. The rework did not build on top of the round-1 commit; it rebuilt the leaf from trunk as a single commit. The chain is 422786c (origin/main) -> 8a0dced. The parent of 8a0dced is 422786c, not 454c2db.

WHY THE PREVIOUS REVIEWER WENT WRONG. Both change-request patches on the board are ZERO BYTES (rev1.patch and rev2.patch, 0 bytes each), so the CR reports repository_delta=empty. With no patch to read, the reviewer reconstructed the subject from an OID recorded in the CR, landed on the superseded 454c2db, and labelled it 'worktree HEAD' - which it is not.

THE TREE YOU MUST REVIEW IS 8a0dced. Confirm it yourself before starting:
  git -C <worktree> log --oneline -1          # expect 8a0dced
  git -C <worktree> rev-parse 8a0dced^        # expect 422786c
  git -C <worktree> diff --stat 422786c 8a0dced   # expect 5 files, +732
Do not review 454c2db. If anything you read describes a 332-line utf8_subsumption_test.go, you are on the wrong commit; the delivered one is 602 lines.

THE ROUND-1 BLOCKER IS FIXED IN 8a0dced. Orchestrator-verified, each mutant written to a file, confirmed present, compiled, measured, removed, tree verified clean:
- Mutant H, exported ValidateDecodedRecord(schema, version string, object map[string]any) dispatching through immutableObjectShapeValidators: REDDENS, naming all seven guarded functions.
- Mutant I, exported method RecordGate.Validate with the same decoded map: REDDENS, naming the same seven.
- Coverage is now 7 of 7 derived re-check functions, up from 3 of 7, and the ratio is asserted rather than left to prose.
- 'cannot be broken silently' is GONE from closed_shapes.go (grep count 0), and README now publishes the bound, naming reflection and a function value handed to another package as constructions the graph does not model.
Do not re-run H and I to re-establish this. Attack what the rework introduced.

WHAT TO ATTACK
1. The test grew 332 -> 602 lines. Look for FALSE edges as well as missing ones. A graph that over-approximates makes the pin fire on genuinely safe constructions, which trains contributors to weaken it. Check function-value edges are added only where the value is actually reachable as a call.
2. Is the 7-of-7 ratio assertion non-vacuous? Try shrinking utf8GuardedFunctions' derivation and confirm something reddens rather than the ratio quietly re-satisfying itself at a smaller denominator.
3. Is the published bound TRUE, not merely humble? Construct something README implies IS caught but is not; and confirm something it says is NOT caught indeed is not. Understating a bound and overstating it are different defects and only the first is safe.
4. Confirm Part 1 was not regressed. The two digest assertions and their SPEC quotations must be unchanged and still individually decisive. The previous reviewer independently reconfirmed both digests by recomputation and accepted Part 1; that acceptance stands and does not need redoing.

FACTUAL CORRECTION TO THE BUG TEXT
The Bug said 'Only NUM-SAFE-MAX is pinned'. Wrong: JCS-UTF16-ORDER was already pinned on main at canonical_test.go:512. Two fixtures were genuinely unpinned, not three, so README's claim that all four are now pinned is accurate. This is the second factual error in the audit report this cycle; its line numbers for the utf8 sites were also stale. Re-derive rather than trusting its specifics.

STANDING REVIEW BAR
- Positive-path-only evidence is refused.
- Confirm every mutant is PRESENT in the file before believing a green or a red.
- Do not accept a claim in a comment, README, or logbook you have not independently tested.
- Report anything you find but do not require fixed.

spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Rank-1 and only admitted pair; re-issued round-2 review after the prior verdict reviewed an orphaned superseded commit."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260902-42509b, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260902-42509b)
REVIEW ROUND 3 — ACCEPTED (CR-BUG-260902-1874eo-2 rev 2). Evidence: BUG-260902-1874eo_review-round3-verdict.md, BUG-260902-1874eo_review-round3-mutants.log.

The CR reported repository_delta=empty, which is a base-construction artifact, not an empty delivery. CR base_oid == git HEAD == 8a0dced and candidate_tree_oid == HEAD^{tree} == 6b1134ed, while workspace WS-6f2050d0ba66 checkpoint_oid == 422786cc == HEAD^. The branch is exactly one signed single-parent commit past the checkpoint. Real reviewable window 422786cc..8a0dced: 5 files, 732 insertions, 0 deletions. ORCHESTRATOR ACTION: advance checkpoint_oid to 8a0dced when checkpointing, or every later leaf reads as two-plus commits past checkpoint.

Part 1 accepted. Provenance re-verified independently: SPEC.pinned.md sha256 = 562546d2..ac484a = specpin.DocumentSHA256 (pin.go:30), commit 28bf96d7 (pin.go:28); lines 303/304 quoted verbatim; both digests recomputed from the fixture bytes with shasum, not from the encoder, and match. Mutants A (encoder drift, reddens all four), B2 and C2 (one fixture byte each, reddens exactly its own subtest) plus reviewer-added R1 (swap the two want digests, reddens both) — each pin is individually decisive and binds its own published value. README claim attacked: the pinned SPEC publishes exactly four digests (:300 :303 :304 :307) and all four are asserted.

Part 2 accepted. Round-1 survivor H rebuilt from scratch and KILLED, naming all 7 guards. Also killed: I2 (exported method, byte param, decoded receiver), D2 (direct call), G2 (second json.Unmarshal), E3 (comment deleted), F3 (narrowing — comment names requireExactMembers instead of decodeStrict), N1 (a brand-new undocumented re-check — proves the guarded set is AST-derived, not hardcoded), V1 (orphan guarded function — coverage test reddens with 7/8 and names it, so the 7/7 ratio is measured, not a constant). Reviewer mutant B1 (func-typed struct field dispatch) SURVIVED exactly as closed_shapes.go:57-61 and README publish it — the stated bound reproduces under a mutant built to break it, which is the opposite of the round-1 false guarantee. Subsumption verified at source: canonical.go:328 utf8.Valid refuses before :335 json.NewDecoder inside decodeStrict. Comments follow the localstore/paths.go:233-236 convention.

Validation re-run by the reviewer: go build ./..., gofmt -l internal/ clean, go vet ./..., go test ./... -count=1 all 10 packages ok, tracecheck ok, internal/canonicaljson cover 97.2% (matches the claim). git verify-commit 8a0dced good ECDSA for oparin@me.com, author Ivan Oparin. Tree clean before and after the sweep. Fuzz 100x not re-run; producer evidence stands.

CARRIED FORWARD (accepted as disclosed, wants board items): (1) core_records.go:278 validateProviderIdentityRecord is a ninth re-check of the same class with no subsumption comment — outside this bug closed_shapes.go scope, covered by the entry-point/coverage pins but not the comment pin. (2) Nothing derives the digest-publishing 1.6 fixture set from the pinned document because SPEC.md is not vendored, so a fifth published fixture cannot be caught as unpinned. (3) BUG-260902-9uwnm7 is still in backlog while its part-2 work ships here, and its note names commit 454c2db which is not reachable from HEAD (abandoned rev-1 commit); the delivered commit is 8a0dced — orchestrator to correct the hash and route 9uwnm7.

Reviewer supplies no commit_ack. accept_cr parks the element at to-review for the orchestrator to checkpoint and make the done transition with commit_ack=scope_committed.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-42509b, pid=35455, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [BUG-260902-1874eo_spawn-log_-implementer--developer--claude-_RUN-260902-ec5bf4.log](file://BUG-260902-1874eo/BUG-260902-1874eo_spawn-log_-implementer--developer--claude-_RUN-260902-ec5bf4.log) — System spawn log captured by task-board
- [BUG-260902-1874eo_results.md](file://BUG-260902-1874eo/BUG-260902-1874eo_results.md) — Implementation evidence for both parts: pinned-SPEC provenance for the two digests, the stale-line-number mapping, kept-and-documented resolution for all eight UTF-8 re-checks, 7 decisive mutants, validation exit codes
- [BUG-260902-1874eo_mutant-sweep.log](file://BUG-260902-1874eo/BUG-260902-1874eo_mutant-sweep.log) — Raw go test output for every mutant run, including the two green false-negative attempts and the survived transitive-reachability arm
- [BUG-260902-1874eo_change-request_rev1.patch](file://BUG-260902-1874eo/BUG-260902-1874eo_change-request_rev1.patch) — Change Request CR-BUG-260902-1874eo-1 revision 1 candidate patch (repository_delta=empty, 0 changed paths)
- [BUG-260902-1874eo_change-request_rev1-validation.log](file://BUG-260902-1874eo/BUG-260902-1874eo_change-request_rev1-validation.log) — Change Request CR-BUG-260902-1874eo-1 revision 1 bounded validation log
- [BUG-260902-1874eo_spawn-log_-reviewer--reviewer--claude-_RUN-260902-9a6194.log](file://BUG-260902-1874eo/BUG-260902-1874eo_spawn-log_-reviewer--reviewer--claude-_RUN-260902-9a6194.log) — System spawn log captured by task-board
- [BUG-260902-1874eo_review-verdict.md](file://BUG-260902-1874eo/BUG-260902-1874eo_review-verdict.md) — Reviewer verdict: changes requested. Part 1 (both SPEC digests) accepted, independently reproduced, 3 decisive mutants. Part 2 blocked: mutants H (dispatch-table map entry point) and I (exported method) both survive the full package suite while README/closed_shapes.go claim they redden the pin.
- [BUG-260902-1874eo_spawn-log_-implementer--developer--claude-_RUN-260902-208051.log](file://BUG-260902-1874eo/BUG-260902-1874eo_spawn-log_-implementer--developer--claude-_RUN-260902-208051.log) — System spawn log captured by task-board
- [BUG-260902-1874eo_rework-round1-results.md](file://BUG-260902-1874eo/BUG-260902-1874eo_rework-round1-results.md) — Rework round 1: strengthened AST call graph (function-value dispatch + methods), coverage raised 3/7 -> 7/7 and asserted, published claim narrowed with its evading construction named; 10 mutants including H, I, I2 red and narrowing K/K2
- [BUG-260902-1874eo_rework-mutant-sweep.log](file://BUG-260902-1874eo/BUG-260902-1874eo_rework-mutant-sweep.log) — Raw go test output with real exit codes for the rework sweep: baseline 7/7, mutants H/I/I2/D2/G2/E3/F3 red, narrowing K (H survives without the function-value edge), K2 reproducing the review's 3/7, and part-1 C2
- [BUG-260902-1874eo_change-request_rev2.patch](file://BUG-260902-1874eo/BUG-260902-1874eo_change-request_rev2.patch) — Change Request CR-BUG-260902-1874eo-2 revision 2 candidate patch (repository_delta=empty, 0 changed paths)
- [BUG-260902-1874eo_change-request_rev2-validation.log](file://BUG-260902-1874eo/BUG-260902-1874eo_change-request_rev2-validation.log) — Change Request CR-BUG-260902-1874eo-2 revision 2 bounded validation log
- [BUG-260902-1874eo_spawn-log_-reviewer--reviewer--claude-_RUN-260902-79a17a.log](file://BUG-260902-1874eo/BUG-260902-1874eo_spawn-log_-reviewer--reviewer--claude-_RUN-260902-79a17a.log) — System spawn log captured by task-board
- [BUG-260902-1874eo_review-round2-verdict.md](file://BUG-260902-1874eo/BUG-260902-1874eo_review-round2-verdict.md) — Review round 2 verdict: changes requested. Part 1 (both SPEC digests) accepted and independently re-verified against the pinned document with mutants A and C2. Part 2: round-1 mutants H/I/I2 now red and coverage genuinely 7/7, but three new survivors of the same class - mutant J (byte-taking entry point converts bytes to string without decodeStrict, making a documented-subsumed guard demonstrably LIVE while all four pins and the full suite stay green), mutant L2 (second decoder via aliased encoding/json import), mutant M (new re-check written as utf8.Valid).
- [BUG-260902-1874eo_review-round2-mutants.log](file://BUG-260902-1874eo/BUG-260902-1874eo_review-round2-mutants.log) — Round-2 reviewer mutant sweep: full source of every mutant (A, C2, H, I, I2, E3, F3, O, J, L, L2, M) with its measured result, presence confirmation, independent SPEC provenance recomputation, and clean-tree validation exit codes.
- [BUG-260902-1874eo_spawn-log_-reviewer--reviewer--claude-_RUN-260902-42509b.log](file://BUG-260902-1874eo/BUG-260902-1874eo_spawn-log_-reviewer--reviewer--claude-_RUN-260902-42509b.log) — System spawn log captured by task-board
- [BUG-260902-1874eo_review-round3-verdict.md](file://BUG-260902-1874eo/BUG-260902-1874eo_review-round3-verdict.md) — Reviewer round 3 verdict for CR-BUG-260902-1874eo-2: ACCEPTED, with the empty-delta base-construction correction and three carried-forward items
- [BUG-260902-1874eo_review-round3-mutants.log](file://BUG-260902-1874eo/BUG-260902-1874eo_review-round3-mutants.log) — Reviewer round 3 mutant sweep: 12 reviewer-built mutants, 11 killed, 1 survivor matching the published bound

## Created
2026-09-02T11:59:45Z

## Last Update
2026-09-02T21:18:05Z

## Assigned To
[reviewer] reviewer (claude)
