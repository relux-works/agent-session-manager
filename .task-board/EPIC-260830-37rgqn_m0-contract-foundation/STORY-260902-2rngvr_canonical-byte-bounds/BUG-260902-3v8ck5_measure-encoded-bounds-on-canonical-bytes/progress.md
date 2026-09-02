## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(3))

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Every declared byte bound is measured on canonical bytes, by reusing Canonicalize or an encoder with SetEscapeHTML(false)
- [x] The argv site and the config extension site share one helper rather than each measuring its own way
- [x] A property test asserts a document at the declared limit is accepted regardless of how many characters JSON escaping would expand, using the SPEC literal and not the implementation constant
- [x] A mutant restoring escaped measurement reddens
- [x] Accept-at-limit and refuse-one-past are both proven at the production entries
- [x] Repository tests, vet, build, tracecheck and catalog check exit 0 with no coverage regression
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
FRONT-LOADED DoD from the Stories landed today.

THE DEFECT, reproduced by the audit. With the full Launch Plan shape, an argv of 24 x 1000 less-than characters is 24,073 canonical bytes but 144,073 through json.Marshal, and is refused with the message argv encodes to 144073 bytes at internal/canonicaljson/closed_shapes.go:262-267. Plain control characters at the same canonical size are accepted. U+2028 doubles as well: 65,569 canonical becomes 131,089 escaped. The same defect exists in internal/config at validation.go:752 on the extension bound.

IT FAILS CLOSED, so nothing oversized is admitted. The harm is the other direction: a CONFORMING document is refused, the declared 65,536 bound is not the bound in force, and two call sites in one repository disagree about what a byte is.

THE REPOSITORY ALREADY CONTAINS THE FIX. closed_shapes.go:1487-1495 measures canonical bytes for extensions. Reuse that, do not invent a third way, and make the disagreeing sites share one helper so they cannot drift apart again.

1. ASSERT AGAINST THE SPEC LITERAL, never the implementation constant. A bound compared to the constant it derives from cannot fail - that exact mistake shipped on the configuration Story.

2. BOUNDS PROVEN IN BOTH DIRECTIONS AND AT ADJACENCY. Accept at the limit, refuse one past it, never two past. A case two steps beyond pins a range rather than the limit; that shipped today on the endpoint work and had to be corrected.

3. THE PROPERTY IS THE POINT. A document whose canonical size is at the limit must be accepted no matter how far escaping would expand it - build the case from characters that expand (less-than, U+2028), or the test proves nothing about the defect.

4. A mutant restoring escaped measurement must redden. If it does not, the property is not pinned.

5. DRIVE THE REAL PRODUCTION ENTRY POINTS in both packages.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 recommendation: make the declared byte bound the bound actually in force, and make two disagreeing call sites share one measurement."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260902-a96491, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260902-a96491)
DEVELOPER HANDOFF - ready for review.

FIX. New internal/canonicaljson/declared_byte_bounds.go carries the one measurement, CanonicalByteLength, which encodes with SetEscapeHTML(false) and then runs the RFC 8785 transform, plus canonicalByteBound(name, value, maximum), the one refusal gate. SetEscapeHTML(false) ALONE IS NOT THE FIX: encoding/json escapes U+2028 and U+2029 regardless of that setting, so only the canonical transform makes the measurement independent of the encoder. Call sites: closed_shapes.go:284 (Launch Plan argv) and closed_shapes.go:1794 (extensions object) go through the gate; internal/config/validation.go:763 calls the measurement. internal/config now imports internal/canonicaljson - no cycle, canonicaljson imports only catalog and scalar.

SHARED, NOT DUPLICATED. The two sites that disagreed now reach the same function. canonicalByteBound also has the derived bound-helper shape (name string + maximum int), so both byte bounds became first-class derived obligations in declared_bounds_test.go; two boundaryConstraintCase entries claim them, and changing either literal reddens the derivation gate as well as the new tests.

PROPERTY TESTS. canonical_byte_bound_measurement_test.go and extension_canonical_byte_bound_test.go each run six rows: less-than, greater-than, ampersand, U+2028, U+2029, and one encoding-neutral ASCII control. Every row asserts the accepted at-limit document is OVER the bound when measured the escaped way - that assertion, not the acceptance, is what kills the mutant. Fixture sizes are verified through Canonicalize rather than through the helper under test, and the bound is a local SPEC literal (specificationDeclaredByteMaximum / specificationExtensionCanonicalMaxBytes), never maxConfigExtensionBytes.

PRODUCTION ENTRIES. CalculateObjectIdentity and VerifyObjectIdentity for canonicaljson; EncodeCurrent (writer) and Load (reader) for configuration. Accept at exactly 65536 canonical bytes, refuse at exactly 65537, never two past. The reader-side one-past case edits the encoded TOML document because the writer refuses the over-limit configuration before it can produce one. The canonicaljson refusals are asserted to name the canonical size the bound measured, so an incidental refusal from another gate cannot pass for this one.

MUTANTS: 8 applied, 8 killed, tree restored each time. M1 CanonicalByteLength returns the marshalled length (reddens both packages). M2-M5 narrow and widen each canonicaljson bound by one. M6-M7 do the same to maxConfigExtensionBytes. M8 restores the escaped measurement at the config site alone. Narrow/widen rather than delete-only, because a deleted gate proves existence and says nothing about where the bound sits. Transcript attached as BUG-260902-3v8ck5_mutation-sweep.log.

VALIDATION, real exit codes, each command a standalone process: go build ./... 0; go vet ./... 0; gofmt -l internal/ 0 with no output; go test ./... -cover -count=1 0; tracecheck 0 (contracts=60 normative_sections=36 acceptance_cases=43 fixtures=30 compatibility_contracts=55); go generate ./internal/catalog 0; cataloggen -check 0; four canonicaljson seeded fuzz targets and the scalar one at 100x each, all 0.

COVERAGE. Baseline measured in a detached worktree at HEAD f20aeec with the same commands. Every package unchanged; canonicaljson 97.2 percent before and after. Exact statement counts: before 2233/2170/63 uncovered = 97.179, after 2238/2175/63 uncovered = 97.185. An intermediate shape that left the refusal inline at both call sites measured 64 uncovered (97.143); folding both sites onto the shared gate removed that extra unreachable guard, which is why the final shape matches the baseline uncovered count.

REVIEW ATTENTION. (1) internal/config gaining a dependency on internal/canonicaljson is deliberate and is what the AC asks for. (2) Three unreachable-refusal declarations in refusal_guards_test.go moved from the old call sites into the shared helper, under the existing decodedValueRefusal reason; the values reaching them were produced by decodeStrict or already validated by validateExtensionValue, so neither encoding/json nor the RFC 8785 transform can fail on them.

Candidate is uncommitted in the Story worktree at checkpoint f20aeec. Full write-up: BUG-260902-3v8ck5_results.md.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-a96491, pid=40542, exit=0)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Verify the at-limit cases are built from expanding characters so they actually exercise the defect, that bounds compare to the SPEC literal rather than the implementation constant, and that a mutant restoring escaped measurement reddens."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260902-974a06, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260902-974a06)
Review ACCEPTED (CR rev 1, tree d92be21). Attacked rather than read: reviewer-run mutants beyond the producer sweep — delete the argv gate (4 tests red), delete the extensions-object gate (5 tests red), neuter the config bound to 99x (2 tests red), narrow argv to 65535 (red); dropping SetEscapeHTML(false) is green as designed, since Canonicalize is the actual fix. Proved the property claim complete instead of trusting the table: a temporary probe over every valid rune U+0000..U+10FFFF found exactly 5 divergent runes and the fixture table names all 5. Confirmed out-of-tree that SetEscapeHTML(false) alone would NOT have satisfied the AC (U+2028 still 10 escaped bytes vs 7 canonical), so the Canonicalize path was the required alternative. Bypass audit: zero remaining len(json.Marshal) size measurements in production; all 14 extensions call sites funnel through the two shared entries; production entries driven are CalculateObjectIdentity, VerifyObjectIdentity, EncodeCurrent and loadConfigDocument. Re-ran at the candidate tree: build, vet, gofmt, go test ./... -cover, tracecheck and cataloggen -check all exit 0, coverage identical to baseline in every package (canonicaljson 97.2%, config 94.7%). Reported as unknown, not a finding: validation.go:465 mesh SSH argv 65536 bound sums raw bytes and never used json.Marshal, so it is not this defect class; whether the spec declares it on a canonical encoding could not be established from the local specpin lock, which carries no such literal. Out of scope here; possible separate item. No commit_ack supplied — orchestrator owns the commit and the done transition. Evidence: BUG-260902-3v8ck5_review-verdict.md.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-974a06, pid=61374, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [BUG-260902-3v8ck5_spawn-log_-implementer--developer--claude-_RUN-260902-a96491.log](file://BUG-260902-3v8ck5/BUG-260902-3v8ck5_spawn-log_-implementer--developer--claude-_RUN-260902-a96491.log) — System spawn log captured by task-board
- [BUG-260902-3v8ck5_results.md](file://BUG-260902-3v8ck5/BUG-260902-3v8ck5_results.md) — Implementation, mutation sweep and validation evidence for canonical-byte measurement of the declared byte bounds
- [BUG-260902-3v8ck5_mutation-sweep.log](file://BUG-260902-3v8ck5/BUG-260902-3v8ck5_mutation-sweep.log) — Transcript of the eight-mutant sweep with real exit codes
- [BUG-260902-3v8ck5_change-request_rev1.patch](file://BUG-260902-3v8ck5/BUG-260902-3v8ck5_change-request_rev1.patch) — Change Request CR-BUG-260902-3v8ck5-1 revision 1 candidate patch (repository_delta=present, 8 changed paths)
- [BUG-260902-3v8ck5_change-request_rev1-validation.log](file://BUG-260902-3v8ck5/BUG-260902-3v8ck5_change-request_rev1-validation.log) — Change Request CR-BUG-260902-3v8ck5-1 revision 1 bounded validation log
- [BUG-260902-3v8ck5_spawn-log_-reviewer--reviewer--claude-_RUN-260902-974a06.log](file://BUG-260902-3v8ck5/BUG-260902-3v8ck5_spawn-log_-reviewer--reviewer--claude-_RUN-260902-974a06.log) — System spawn log captured by task-board
- [BUG-260902-3v8ck5_review-verdict.md](file://BUG-260902-3v8ck5/BUG-260902-3v8ck5_review-verdict.md) — Reviewer verdict: accepted. Independent mutant sweep (delete/neuter/narrow/neutrality), exhaustive Unicode divergence probe proving the fixture table complete, bypass-path audit, and full gate re-run at the candidate tree.
- [BUG-260902-3v8ck5_reviewer-divergence-probe_test.go.txt](file://BUG-260902-3v8ck5/BUG-260902-3v8ck5_reviewer-divergence-probe_test.go.txt) — Reviewer probe (run then removed from the tree): enumerates every valid rune U+0000..U+10FFFF and proves exactly 5 runes diverge between encoding/json and RFC 8785 canonical length, matching the fixture table exactly.

## Created
2026-09-02T11:59:24Z

## Last Update
2026-09-02T17:36:21Z

## Assigned To
[reviewer] reviewer (claude)
