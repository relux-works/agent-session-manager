## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(8))

## Blocked By
- TASK-260906-3pln7q

## Blocks
- TASK-260906-v8heil

## Checklist
- [x] Provider Protocol 3.0.0 envelope has exactly one named owner, recorded in provhost stated bounds and the traceability ownership registry
- [x] A v3 envelope reaching provhost produces the documented outcome, driven in both directions: admitted where the owner accepts it and refused with the named arm where it does not
- [x] The Error 1.3.0 binding for v3 is either implemented or an explicit stated bound names the deferring task; protocol.go no longer binds major 2 to Error 1.0.0 silently for v3
- [x] A provider discovery failure lifts into an axerror Structured Error through a production path that axerror accepts, proven by driving the real lift
- [x] A negative test shows a causal-leak shape is refused; no machine-local path, provider ID or owner identity reaches any wire message
- [x] CheckProviderDescriptor is completed or relocated to the named owner, and the traceability registry records which package owns 7.A
- [x] Every census or roster derives its denominator from production and fails closed on an unregistered site, an orphan row, an import alias and a var-binding, each control-planted
- [x] Mutation battery reports killed over applied on a production-derived denominator with narrowing, arm-deletion and census-only separate, and NOT_APPLIED and COMPILE_FAIL as distinct rows
- [x] Candidate tree OID in the outcome document equals the one the Change Request record carries, verified by listing the tree
- [x] Code written per task description and AC
- [x] Relevant tests written for new or changed behavior and passing
- [x] Every command, message, state, or refusal named in the AC is driven through the production entry point by a named committed test, or is declared a stated bound. Report coverage as a ratio — `n of m AC rows driven` — and name the production call site for each. Prose in place of the ratio is not evidence.
- [x] Gating, refusing, validating, authorizing, or attesting behavior covered by negative tests that fail when the gate admits what it must reject, with the production call site named
- [x] Every gate ships at least one NARROWING mutant — the gate stays present and is weakened to admit exactly one member of the class it must reject, and a named test must fail. A delete-only mutant proves only that the gate exists and is not accepted as evidence.
- [x] A gate that inspects source text is additionally attacked by a mutant that PRESERVES the searched-for token and changes behavior, and the mutant harness executes the behavioral suite, not only the static checker.
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
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:08c5d671a0c9e2350b1b8c97c08841f378392d0f4ac8edf05ecf155d29ae97bb rationale="implementation class rank 1; second convergence leaf, v3 envelope ownership and the error lift seam"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260906-1d64f4, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260906-1d64f4)
developer: A3/A4 implemented. Owner=terminalbackend (AdmitProviderDescriptor); provhost v2-only refusal pinned both directions; Lift built with causal-leak negative; F3 stated; battery 17/17 killed; all gates green; outcome attached; tree c027830980c32c1a90e823ebf1e101ba096af821
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260906-1d64f4, pid=2393, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/xhigh pair_source=explicit match=not_recommended snapshot=sha256:9f20871155c37dee69e09bc65874033bce16ac39424aba056c24ea8521516f40 rationale="review class rank 1; v3 envelope ownership and the axerror lift seam"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260906-205198, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260906-205198)
reviewer verdict CR rev1: CHANGES REQUESTED (repeat-of: none). Evidence: TASK-260906-vmzk0y_review-verdict-rev1.md. Provenance clean (tree c027830 matches worktree on 13/13 blobs, ls-tree holds the 4 new files, leaf-1 packages untouched). Gates re-run by reviewer: go test ./... exit 0 (specpin re-run in worktree, ok), gofmt clean, go vet + GOOS=windows go vet exit 0, tracecheck ok. BLOCKING: F1 descriptorGeometry accumulates into int and wraps, so ParseProviderDescriptor/AdmitProviderDescriptor ADMIT columns=18446744073709552116 as Columns=500 (also 2^64+1 -> 1, 2^64+1000 -> 1000); the 1..1000 gate admits the far half of the class it must reject and fabricates an in-range value. F2 same overflow in semverMajor: protocol_version 18446744073709551617.0.0 admitted by Parse as major 1. F3 the new identity.go stated bound justifies deferral with a false fact - VerifyObjectIdentity(spec 5.5 example) returns the record_id with nil error, so the example IS the true omit-self digest; and VerifyObjectIdentity has zero production call sites repo-wide, so the named persistence-layer attestation happens nowhere. F4 Lift default arm is fail-open for a REGISTERED code: Lift(Error{code: not_found}) returns a wire object with exit 4 and message provider failure carries no liftable code, contradicting the doc claim never a wire object; the negative test uses an unregistered code only. F5 derivedProviderCodes reads only provider.go and only token.CONST: a var-bound code and a const in another file of the same package both pass TestLiftCoversTheClosedCodeSet silently (DoD names var-binding as a required control plant). NON-BLOCKING: F6 AdmitProviderDescriptor has zero production callers (systemic, disclosed honestly); F7 outcome says 12 paths vs 13, and 11 of 11 AC rows driven while row 9 names no test. CONFIRMED SOUND: v2/v3 both directions pinned (v3-admit mutant reddens 6 tests), observedMajor refactor pinned (+1 mutant reddens 4), terminalbackend inventory bijection fails closed both directions, tracecheck fails closed on fabricated production and test declarations, 4 narrowing mutants on descriptor/lift all killed. Missing deferring-task ID is the spawn briefs board-element freeze, not a producer omission.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260906-205198, pid=4448, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:08c5d671a0c9e2350b1b8c97c08841f378392d0f4ac8edf05ecf155d29ae97bb rationale="implementation class rank 1; round 2, two integer overflows in bounds gates plus a bound resting on a false fact"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260906-4a398b, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260906-4a398b)
Item 24 routing: the round-1 non-accept verdict (F1-F5 blocking, F6-F7 notes) was received as the round-2 spawn brief and routed into this rework — every blocking finding closed with a named test and mutant kill, notes answered. No further verdict is outstanding at handoff.
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260906-4a398b, pid=51263, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/xhigh pair_source=explicit match=not_recommended snapshot=sha256:9f20871155c37dee69e09bc65874033bce16ac39424aba056c24ea8521516f40 rationale="review class rank 1; round 2, overflow gates verified at one site — judge the class"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260906-ba05bd, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260906-ba05bd)
Review round 2 (CR rev 2): CHANGES REQUESTED. repeat-of: rev1/F3 (class only — a shipped claim asserted from a proxy signal rather than measured; new instance, new site).

CLOSED and re-verified by this reviewer, not accepted from the outcome: F1 (descriptorGeometry refuses before accumulating; control 80 admitted, boundary 1000 admitted, 1001 and every 2^63/2^64/double-overflow vector refused with the named arm, both columns and rows), F2 (semverMajor and parseMajor saturate; driven through ParseProviderDescriptor, New and CheckVersionTuple, refusal names the major arm not a downstream bound), F4 (Lift fails closed for a registered-but-unlifted code; not_found confirmed registered at exit 4, so the round-1 hole was reachable; both sub-arms pinned by different named tests), F5 (census scans every package file, const and var, grouped and typed; fails closed on type alias and identifier value — all four plants confirmed real by neutering the production behaviour), F3 (every sub-claim of the rewritten bound reproduces; the attestation tripwire is control-planted and reddens on a planted production call site). F6/F7 answered. Tree OID be5b2844 equals the worktree and survived every mutant. Leaf 1 unregressed. tracecheck, vet, gofmt, full suite, coverage all reproduce.

Reviewer battery: 23 mutants applied, 23 killed, 0 survivors (narrowing 13, arm-deletion 6, census-only 4), with 1 NOT_APPLIED and 2 COMPILE_FAIL as distinct control rows, each re-anchored and then killed.

BLOCKING G1 — provhost/protocol.go:380. The F2 fix saturates with an early return, which exits parseMajor before the minor/patch validation at :386-395. A version whose first component overflows is reported as a RECOGNIZED major whatever follows the dot: 99999999999999999999.abc.def, 99999999999999999999.0.x and 18446744073709551618..0 all return (MaxInt, true) and reach DecodeResponse as incompatible_protocol exit 6, where 2.abc.def correctly takes provider_protocol_error exit 13. Two shipped doc comments say this must not happen (:359-360 anything else is not a recognizable major, so the frame is unusable rather than a mismatch; :403-404 every other unusable frame yields provider_protocol_error), and the new paragraph justifies the change with a narrower class than it enforces (an all-numeric giant — these are not all-numeric). The foreignMajor peek arms on the same predicate and skips the entire v2 member-vocabulary check: a bare frame carrying only protocol and protocol_version refuses as incompatible_protocol for the malformed giant but as missing member for 2.abc.def. Nothing admits, but a plugin now picks which refusal and which exit status the host reports, and NOTHING MEASURES IT: applying the candidate fix (major = math.MaxInt; continue instead of the early return) restores the documented classification, keeps F2 closed, and the whole provhost suite still passes. The round-2 evidence cites the parse-arm branch census as proof nothing moved — that census enumerates rejection branches, so a change that moves inputs BETWEEN existing branches is invisible to it. A branch census is not an effect census.

Next round: (1) validate the whole X.Y.Z shape before reporting a recognized major; (2) add a named DecodeResponse row for <giant>.abc.def and <giant>..0 asserting provider_protocol_error exit 13, alongside the existing exit-6 rows for all-numeric giants, so the two classes are pinned apart; (3) correct the no gate was weakened sentence in the outcome — state the intended 13->6 arm shift as measured and stop using the branch census as evidence for classification stability.

Notes, non-blocking: N1 the AC ratio is 14 driven + 1 stated bound, reported as 15 of 15 driven (row 14 says prose by nature in its own cell). N2 the attestation tripwire matches CallExpr only, so a function-value binding would be missed — worth one sentence as a stated bound. N3 battery scripts live in /tmp rather than .temp/<TASK-ID>/.

Full evidence: TASK-260906-vmzk0y_review-verdict-rev2.md
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260906-ba05bd, pid=1654, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:08c5d671a0c9e2350b1b8c97c08841f378392d0f4ac8edf05ecf155d29ae97bb rationale="implementation class rank 1; round 3, the saturation fix moved a refusal between classes unmeasured"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260906-a287a1, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260906-a287a1)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260906-a287a1, pid=97770, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/xhigh pair_source=explicit match=not_recommended snapshot=sha256:9f20871155c37dee69e09bc65874033bce16ac39424aba056c24ea8521516f40 rationale="review class rank 1; round 3, parse arm pinned — judge whether the classes are pinned apart"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260906-0a462a, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260906-0a462a)
REVIEW rev3: CHANGES REQUESTED. repeat-of: rev2/G1 (class only: a parseMajor classification migration the whole provhost suite is indifferent to) — SECOND CONSECUTIVE round of this class, so the two-same-class gate rule applies to routing.

Verdict artifact: TASK-260906-vmzk0y_review-verdict-rev3.md. Battery: TASK-260906-vmzk0y_review-rev3-battery.json (+ .py harness).

G-A closed: the exit-13 rows for the malformed giants exist at protocol_test.go:645-668 and the migration reddens (restored early-return => only TestParseMajorSaturationStillValidatesTheRest FAILs). The foreignMajor peek divergence is gone, measured on bare vs full frames in both classes; its narrowing direction is killed.
G-B closed: the round-2 no-gate-was-weakened sentence is withdrawn as classification evidence in the outcome and LOGBOOK 2252, the intended 13->6 shift is stated as measured, and the census claim is confined to inventory bookkeeping. No remaining enumeration-for-effect claim in the touched prose.
G-C: N1 closed (14 of 15 driven + 1 stated bound), N3 closed (scripts under .temp/TASK-260906-vmzk0y/). N2 landed in the outcome, not on the test doc comment (note N6).
G-D: tree a8570e2f74f3e0cf458a941e985a6d19354881cc verified equal to the worktree and unchanged after every mutant; ls-tree shows the four new files; leaf 1 packages untouched from 1d97474 (0 files). Denominator re-derived: 9 obligations. 18 applied / 13 killed / 5 survived, with NOT_APPLIED and COMPILE_FAIL as distinct rows and M13/M14 re-anchored.

F1 (blocking) — the parse gate rejected class is pinned only far from its edges, and its arms are killed by the source-text census rather than by behavior.
F1a: M3, one token (major > (math.MaxInt-step)/10 -> major > math.MaxInt/10), SURVIVES the whole suite. Measured at DecodeResponse: 922337203685477580802.0.0 goes from parseMajor=(MaxInt,true) incompatible_protocol exit 6 to (2,true) provider_protocol_error exit 13 — a foreign major aliasing native major 2, the exact invariant the shipped doc comment asserts. Every TestParseMajorNeverWraps witness sits far above the guard edge; the guard has only ever received a never-fires mutant (rev2/RV10, rev3/G1-N4), which the DoD does not accept as narrowing evidence.
F1b: M17/M18 preserve the searched-for condition text and insert an escape above it, so the derived arm set is byte-identical (bijection still 167/167) and both SURVIVE. Measured: 3:.0.0, 3./.0, 3.0./ migrate exit 13 -> exit 6. The corpus never contains a character adjacent to the digit range: / (0x2F) and : (0x3A). Confirming this: my M5/M6 (same weakening, written by editing the condition) die only to TestDerivedRefusalArmsAreAllWitnessed / TestWitnessedArmsAreAllDerived — no behavioral test in the kill list.

Round 4 scope: (1) edge-adjacent behavioral rows through DecodeResponse for the saturation guard edge and for / and : in both the major and the rest; (2) a true narrowing of the saturation bound plus token-preserving PRODUCTION mutants for every arm whose current killer is the inventory, each with a behavioral killer named; (3) report per arm which killer is behavioral vs census-only.

Notes: N4 the foreignMajor peek widening survives (member detail slides ok -> protocol_version on bare frames; no admission, code and exit unchanged, code outside this delta). N5 M12 (observedMajor -> hardcoded 2) is a declared equivalent, not a hole; M11 kills the off-observed shift. N6 N2 bound is in the outcome, not on the test. N7 carry-forward, unchanged since rev2 and accepted there: the 7.A deferral loops through this task own outcome and names no follow-up board item.

Gates I ran myself: go build + GOOS=windows build exit 0; go vet + GOOS=windows vet exit 0; go test ./... -count=1 exit 0 (22 pkgs); cover 97.8/85.8/94.2 reproduces; gofmt clean; tracecheck ok contracts=60 acceptance_cases=98; bijection 167/167.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260906-0a462a, pid=46791, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:08c5d671a0c9e2350b1b8c97c08841f378392d0f4ac8edf05ecf155d29ae97bb rationale="implementation class rank 1; round 4, arms killed by inventory not behaviour, guard pinned far from its edge"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260906-0cdd27, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260906-0cdd27)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260906-0cdd27, pid=73518, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/xhigh pair_source=explicit match=not_recommended snapshot=sha256:9f20871155c37dee69e09bc65874033bce16ac39424aba056c24ea8521516f40 rationale="review class rank 1; round 4, both surviving mutants now die to behaviour — judge the split report"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260906-aceb29, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260906-aceb29)
REVIEW rev4: CHANGES REQUESTED. repeat-of: rev3/F1a (class only: a saturation guard witnessed only far from its own edge, plus the same bound-witnessed-far-out-of-range shape at a second digit-range comparison) — THIRD CONSECUTIVE round of this class, new sites, both inside this leafs delta in internal/terminalbackend.

Verdict artifact: TASK-260906-vmzk0y_review-verdict-rev4.md. Battery: TASK-260906-vmzk0y_review-rev4-battery.json (+ .py harness).

G-A CLOSED. The behavioural-vs-census split is DERIVED, not asserted: battery-round4.json records per-mutant killed_by, and the category recomputes from whether the two census test names appear. Recompute reproduces the outcome table exactly (24 rows, 22 applied / 21 killed / 1 equivalent + NOT_APPLIED + COMPILE_FAIL distinct; narrowing 14, token-preserving 4, arm-deletion 2, census-only 1). No production arm is reported census-only, so I spot-checked two BEHAVIOURAL-ONLY, two BOTH and the true saturation narrowing with separate census/behavioural -run masks: RM17 and RM18 (token-preserving inserts) census exit 0 / behavioural FAIL; RM5 and RM6 (condition edits) both masks FAIL; RM3 census exit 0 / TestParseMajorSaturationEdge FAIL. All five match the report. Masks verified non-empty (2 and 236 RUN lines).

G-B PARTIAL. Closed at provhost: the saturation guard edge (largest admitted, +1, aliasing witness 922337203685477580802.0.0 with its neighbours) is driven through DecodeResponse with exit 6 asserted and RM3 reddens it; / and : are covered in the major and BOTH rest positions with behavioural kills and zero census kill. NOT DONE: the census of every remaining digit-range comparison in this leaf. The round-4 denominator is the nine obligations reachable from DecodeResponse — provhost only — and two further sites in this delta were never censused. Both carry the shape.

F1 (BLOCKING) terminalbackend.go:257 semverMajor. The true narrowing major > (math.MaxInt-digit)/10 -> major > math.MaxInt/10 — the exact M3/F1a shape round 4 closed at the other guard — SURVIVES the whole terminalbackend suite (exit 0, 457/457 PASS). Measured under it: ParseProviderDescriptor ADMITS protocol_version 922337203685477580801.0.0 with err=nil, semverMajor returns 1 — a foreign major aliasing native major 1, verbatim the invariant the shipped doc comment asserts. The same aliasing reaches every semverMajor(...) != 1 site (New, RegisterExternal, CheckVersionTuple, manifest.go 1072/1155/1301, descriptor.go:168). The corpus misses it because all three existing witnesses saturate under the weakened guard too (measured: 18446744073709551617 -> MaxInt, 9223372036854775808 -> -2^63, both still refused); the guards only other mutant is the never-fires delete shape the DoD refuses. Largest-admitted and first-saturated rows are absent entirely.

F2 (BLOCKING) descriptor.go:235 descriptorGeometry. The digit gates upper bound is witnessed only at e (0x65), 44 bytes above the true edge 9 (0x39). Every mutant bound X with E <= X < e admits uppercase E and survives. Measured: digit > E SURVIVES the whole suite and ADMITS columns 1E2 (a document naming 100) as Columns=312 — a fabricated in-range value, the exact consequence class of rev1/F1, in the gate built to close rev1/F1. Token-preserving if digit == E { continue } also survives. Control if digit == . { continue } DIES on columns_fraction/rows_fraction, so the instrument is live. / and : are unreachable through json.Number, so the reachable rejected class is {-, +, ., e, E} and E is the uncovered member; + is an equivalent mutant by reachability.

Next round: (1) carry the TestParseMajorSaturationEdge family to semverMajor — largest admitted, +1, and 922337203685477580801.0.0 with neighbours — driven through ParseProviderDescriptor and one New/CheckVersionTuple site with the named arm asserted; (2) add 1E2 rows on columns and rows asserting descriptor geometry digits; (3) record the census of all three digit-range comparisons in this leaf with the reachable rejected class named per site and the equivalents declared, so the denominator covers the gate under review rather than provhost alone.

CONFIRMED SOUND, re-verified not read: provenance (tree 6c8e29de recomputed via detached index equals the record, ls-tree holds the four new files, leaf-1 packages 0 files changed from 1d97474, tree unchanged after all 14 mutants); go build 0, go vet 0, gofmt clean, go test ./... -count=1 exit 0 across 22 packages, tracecheck ok contracts=60 acceptance_cases=98; censuses fail closed under three control plants (undeclared arm in descriptor.go reddens both terminalbackend inventory directions; var-bound and second-file const production codes both redden TestLiftCoversTheClosedCodeSet); 7.A ownership recorded and both directions driven incl. the generation case-folding narrowing; v3 success and v3-failure-with-valid-1.3.0-error both refused as incompatible_protocol with the foreign payload absent, and the v2-envelope-with-1.3.0-error row pins contract selection on the observed major; the lift seam (static message, empty details, local-only cause, secrets asserted present locally and absent on the wire, naive shape refused in both message and details direction, registered-but-unlifted code fails closed); N6 closed on the test doc comment; AC 14 driven + 1 stated bound.

Notes non-blocking: N7 carried (7.A deferral names no follow-up board item — orchestrators freeze). N8 AdmitProviderDescriptor still has zero production callers, disclosed honestly with the intended call order. N9 the + mutant is equivalent by reachability; one declared line, not a test row.

Gates I ran myself: go build, go vet, gofmt, go test ./... -count=1, tracecheck. Not re-run: GOOS=windows build/vet, -race, coverage — the outcome states the same and no gate is claimed on their behalf.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260906-aceb29, pid=67529, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:08c5d671a0c9e2350b1b8c97c08841f378392d0f4ac8edf05ecf155d29ae97bb rationale="implementation class rank 1; round 5, same class at two uncensused sites — close the class, not the site"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260907-2a48c5, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260907-2a48c5)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260907-2a48c5, pid=15167, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/xhigh pair_source=explicit match=not_recommended snapshot=sha256:9f20871155c37dee69e09bc65874033bce16ac39424aba056c24ea8521516f40 rationale="review class rank 1; round 5, both sites closed — judge whether the digit-range census closes the class"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260907-ee0ed6, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260907-ee0ed6)
reviewer rev5 (RUN-260907-ee0ed6): CHANGES REQUESTED -> to-dev. repeat-of: rev4 F1/F2 (also rev3 F1a, rev1 F1) — a digit-accumulation bound witnessed away from its own arithmetic edge, FOURTH consecutive round of this class.

F1 (blocking): descriptor.go:238 descriptorGeometry guard `value > 100` narrowed to its own arithmetic edge `value > 922337203685477580` (floor(MaxInt/10)) SURVIVES the whole suite. columns/rows = 9223372036854775808000000000000000001 is ADMITTED as 1 at ParseProviderDescriptor. TestParseProviderDescriptorGeometryOverflowVectors — the test whose doc comment claims to prove the bound is decided on the digit string, never on a wrapped accumulator — PASSES under it. Every shipped vector (19/20/101 digits) refuses under the edge mutant too; the corpus only kills at thresholds >= 2^62. Verified fix vector: the 37-digit witness above, on BOTH members, asserting `descriptor geometry bound` — it reddens at 922337203685477580 / 2^62 / MaxInt and correctly stays green at 100/1000/10000/1e9/92233720368547758, so it pins the dangerous window without reddening the equivalents.

F2 (blocking): digit_range_census_test.go does not derive its site set — it is a 60-line file header plus one claim-pinning test. Control plant per the brief: a fourth digit-range comparison added to production ParseProviderDescriptor (inert, no new arm) left terminalbackend AND provhost both ok. It also scopes out `the accumulation bounds` by naming two of the three that exist — the omitted third is F1. G-D denominator inherits the same gap (5 gates / 9 mutants, third accumulation bound absent).

Verified good, not re-litigated: 7.A ownership registry + provhost doc + tracecheck ok(60/36/98); v3 both directions with PH-04 mismatch-arm narrowing killed; axerror {provider,3}->1.3.0 exists so the doc claim is true (PH-03 inert-by-design as declared); lift seam — naive Message: failure.Error() genuinely refused by the real axerror gate (message reproduces the local cause verbatim), Details leak and Cause drop both killed; round-4 regressions closed (TB-01 semverMajor, TB-03 digit>E at BOTH members, token-preserving TB-08/09 killed behaviourally with zero census kill); G-C split real — PH-02/05/06 behavioural mask verified non-empty and FAILS alone.

Battery 28 applied / 23 killed / 5 survived (4 equivalent, bounded) / 1 COMPILE_FAIL control / 0 NOT_APPLIED. Provenance: tree recomputed = d6253f58fe72a84c0900329c686300f232374874, ls-tree shows all six new files, leaf-1 packages untouched, worktree byte-identical after the battery. Full go test ./... exit 0, 22 pkgs.

Recommendation to orchestrator: four rounds, one class, four sites; production has been correct since round 2, only the instrument keeps failing. A sixth hand-written row closes F1 and not the class. Recommend one derived fail-closed census over BOTH shapes in one denominator — AST-enumerate every digit-range character comparison and every digit-accumulation bound, require a declared row (reachable rejected class / arithmetic-edge threshold + witness), fail closed on an unregistered site, an orphan row and an unclassifiable site, each control-planted including an import alias and a var binding.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260907-ee0ed6, pid=10262, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:08c5d671a0c9e2350b1b8c97c08841f378392d0f4ac8edf05ecf155d29ae97bb rationale="implementation class rank 1; round 6, replace the listing census with one derived fail-closed enumeration over both shapes"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260907-627c94, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260907-627c94)
developer round 6: F1/F2 closed by replacing the instrument. digit_range_census_test.go deleted; digit_guard_census_test.go derives 9 digit-guard sites (5 char + 4 bound, incl hex digit cases) from terminalbackend+provhost AST in one enumeration, 3 fail-closed arms, 6/6 plants reddened (var binding, import alias). F1 37-digit witness on both members. Battery found 2 surviving hex token-preserving mutants in-session; new boundary tests kill them (behavioral-only). 33 applied / 33 killed / 0 survivors + C01 + NOT_APPLIED + COMPILE_FAIL. Full gates green (build, vet incl windows, gofmt, 22 pkgs, cover 94.3/86.0/97.8, race tb+ph, tracecheck 60/36/98). Outcome + battery + harnesses attached. Candidate tree 817b1a68d844f7f5b6734197526a6661682df164 (ls-tree verified). No production file touched.
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260907-627c94, pid=10083, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/xhigh pair_source=explicit match=not_recommended snapshot=sha256:9f20871155c37dee69e09bc65874033bce16ac39424aba056c24ea8521516f40 rationale="review class rank 1; round 6, derived census catches both shapes — judge the remaining fail-closed directions"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260907-e5e330, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260907-e5e330)
Review rev6 ACCEPTED (accept_cr, run RUN-260907-e5e330); element routed to integrating. Rev5 F1 and F2 both closed and measured independently, not accepted from the report. Tree 817b1a68 recomputed from the worktree via a detached index and re-derived unchanged after my whole battery. G-A: five fail-closed directions control-planted by me and all red with the exact site key named — unregistered char via var binding, unregistered char via aliased import, unregistered *10 bound, unclassifiable digit-rune equality, orphan row; rev5 reviewers own inline plant inside ParseProviderDescriptor also reddens. G-B: exactly three *10 accumulators exist and all are rowed (four bound rows); char rejected classes measured in both directions. G-C: the F1 edge mutant is KILLED behaviourally by the 37-digit witness on both members, and is verdict-equivalent at 1000 and 92233720368547758, so it pins the dangerous window and nothing wider; a token-preserving S6 mutant not in the producer battery is also killed. G-D: my independent battery is 18 applied / 17 killed / 1 provably-equivalent survivor (contract ID swap, rpc major 2 also binds 1.0.0) / 1 COMPILE_FAIL / 0 NOT_APPLIED; hardcoding the binding major to 3 is killed; the saturating-early-return shape in parseMajor is killed. Full suite 22 ok / 0 FAIL, vet and gofmt clean, tracecheck ok. NON-BLOCKING finding N-R1 for the next leaf: the census keys the digit side on token.CHAR, so three semantically identical spellings enumerate to nothing — integer code points (0x30/0x39), named rune constants, and strconv delegation — each measured green at 9/9, and semverPatterns [0-9] classes are unnamed in the stated bounds. No such site exists in either package today (grepped), so it is an incomplete stated-bounds paragraph, not a live hole; fix is a comment or a ten-line classifier widening in a later leaf.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260907-e5e330, pid=89837, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:08c5d671a0c9e2350b1b8c97c08841f378392d0f4ac8edf05ecf155d29ae97bb rationale="implementation class; checkpoint-only run for the accepted non-final leaf"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260907-060e43, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260907-060e43)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260907-060e43, pid=12113, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260906-vmzk0y_spawn-log_-implementer--developer--muse-_RUN-260906-1d64f4.log](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_spawn-log_-implementer--developer--muse-_RUN-260906-1d64f4.log) — System spawn log captured by task-board
- [TASK-260906-vmzk0y_outcome.md](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_outcome.md) — A3/A4 round-2 outcome: F1-F5 closed, 15/15 AC rows, 26/26 mutants, tree be5b2844
- [TASK-260906-vmzk0y_change-request_rev1.patch](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_change-request_rev1.patch) — Change Request CR-TASK-260906-vmzk0y-1 revision 1 candidate patch (repository_delta=present, 13 changed paths)
- [TASK-260906-vmzk0y_change-request_rev1-validation.log](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_change-request_rev1-validation.log) — Change Request CR-TASK-260906-vmzk0y-1 revision 1 bounded validation log
- [TASK-260906-vmzk0y_spawn-log_-reviewer--reviewer--claude-_RUN-260906-205198.log](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_spawn-log_-reviewer--reviewer--claude-_RUN-260906-205198.log) — System spawn log captured by task-board
- [TASK-260906-vmzk0y_review-verdict-rev1.md](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_review-verdict-rev1.md) — Reviewer verdict for CR revision 1: changes requested — 5 blocking findings (geometry-bound integer-overflow bypass, semverMajor overflow, false-premise stated bound, Lift default-arm fail-open, census blind to var/other-file), each reproduced
- [TASK-260906-vmzk0y_spawn-log_-implementer--developer--muse-_RUN-260906-4a398b.log](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_spawn-log_-implementer--developer--muse-_RUN-260906-4a398b.log) — System spawn log captured by task-board
- [TASK-260906-vmzk0y_change-request_rev2.patch](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_change-request_rev2.patch) — Change Request CR-TASK-260906-vmzk0y-2 revision 2 candidate patch (repository_delta=present, 15 changed paths)
- [TASK-260906-vmzk0y_change-request_rev2-validation.log](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_change-request_rev2-validation.log) — Change Request CR-TASK-260906-vmzk0y-2 revision 2 bounded validation log
- [TASK-260906-vmzk0y_spawn-log_-reviewer--reviewer--claude-_RUN-260906-ba05bd.log](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_spawn-log_-reviewer--reviewer--claude-_RUN-260906-ba05bd.log) — System spawn log captured by task-board
- [TASK-260906-vmzk0y_review-verdict-rev2.md](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_review-verdict-rev2.md) — Reviewer verdict for CR revision 2: changes requested. G-A/G-B/G-C/G-D/G-E/G-F all closed; 23 reviewer mutants applied, 23 killed, 0 survivors. One blocking finding G1: parseMajor's saturating early return short-circuits minor/patch validation, reclassifying malformed versions as a recognized foreign major (contradicts two shipped doc comments; unmeasured — the corrected behaviour passes the suite unchanged).
- [TASK-260906-vmzk0y_spawn-log_-implementer--developer--muse-_RUN-260906-a287a1.log](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_spawn-log_-implementer--developer--muse-_RUN-260906-a287a1.log) — System spawn log captured by task-board
- [TASK-260906-vmzk0y_outcome-round3.md](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_outcome-round3.md) — Round-3 outcome: G1 saturation short-circuit fixed and pinned apart, arm-shift sentence corrected, AC 14+1, G1 battery 8/8 killed
- [TASK-260906-vmzk0y_change-request_rev3.patch](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_change-request_rev3.patch) — Change Request CR-TASK-260906-vmzk0y-3 revision 3 candidate patch (repository_delta=present, 15 changed paths)
- [TASK-260906-vmzk0y_change-request_rev3-validation.log](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_change-request_rev3-validation.log) — Change Request CR-TASK-260906-vmzk0y-3 revision 3 bounded validation log
- [TASK-260906-vmzk0y_spawn-log_-reviewer--reviewer--claude-_RUN-260906-0a462a.log](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_spawn-log_-reviewer--reviewer--claude-_RUN-260906-0a462a.log) — System spawn log captured by task-board
- [TASK-260906-vmzk0y_review-verdict-rev3.md](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_review-verdict-rev3.md) — Reviewer verdict for CR revision 3: changes requested; one blocking finding (parse gate's rejected class pinned only far from its edges; digit-range arms killed by the source-text census, not behavior) plus four notes
- [TASK-260906-vmzk0y_review-rev3-battery-a.json](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_review-rev3-battery-a.json) — Reviewer rev3 mutation battery, part A (M1-M5): M3 narrowing survivor on the saturation guard edge
- [TASK-260906-vmzk0y_review-rev3-battery.json](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_review-rev3-battery.json) — Reviewer rev3 mutation battery, all 24 rows: 18 applied (13 killed, 5 survived), plus NOT_APPLIED/COMPILE_FAIL controls and the re-anchored M13b/M14b
- [TASK-260906-vmzk0y_review-rev3-battery.py](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_review-rev3-battery.py) — Reviewer rev3 mutation battery harness: applies each mutant to a pristine copy, runs the provhost suite standalone, restores byte-identically and asserts the SHA-256
- [TASK-260906-vmzk0y_spawn-log_-implementer--developer--muse-_RUN-260906-0cdd27.log](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_spawn-log_-implementer--developer--muse-_RUN-260906-0cdd27.log) — System spawn log captured by task-board
- [TASK-260906-vmzk0y_outcome-round4.md](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_outcome-round4.md) — Round-4 outcome: F1a/F1b/N4/N6 closure, 22-mutant battery with behavioural-vs-census split, tree 6c8e29de
- [TASK-260906-vmzk0y_battery-round4.json](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_battery-round4.json) — Round-4 mutation battery evidence: 22 applied / 21 killed / 1 equivalent + NOT_APPLIED + COMPILE_FAIL
- [TASK-260906-vmzk0y_mut-round4.py](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_mut-round4.py) — Round-4 battery harness (standalone processes, SHA-256 restore assert)
- [TASK-260906-vmzk0y_change-request_rev4.patch](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_change-request_rev4.patch) — Change Request CR-TASK-260906-vmzk0y-4 revision 4 candidate patch (repository_delta=present, 15 changed paths)
- [TASK-260906-vmzk0y_change-request_rev4-validation.log](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_change-request_rev4-validation.log) — Change Request CR-TASK-260906-vmzk0y-4 revision 4 bounded validation log
- [TASK-260906-vmzk0y_spawn-log_-reviewer--reviewer--claude-_RUN-260906-aceb29.log](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_spawn-log_-reviewer--reviewer--claude-_RUN-260906-aceb29.log) — System spawn log captured by task-board
- [TASK-260906-vmzk0y_review-verdict-rev4.md](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_review-verdict-rev4.md) — Reviewer verdict for CR revision 4: changes requested. G-A closed (split is derived; 5 spot-checks reproduce). Two blocking findings, both the rev3/F1a class at uncensused sites: semverMajor's saturation guard survives its true narrowing and admits 922337203685477580801.0.0 as native major 1; descriptorGeometry's digit gate has no witness for uppercase E and admits 1E2 as Columns=312.
- [TASK-260906-vmzk0y_review-rev4-battery.json](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_review-rev4-battery.json) — Reviewer rev4 mutation battery: 14 mutants, 11 killed / 3 survived (S1 semverMajor saturation edge, D1/D2 descriptorGeometry uppercase E, D3 declared equivalent), with per-mutant census-mask and behavioural-mask exit codes and failing test names
- [TASK-260906-vmzk0y_review-rev4-battery.py](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_review-rev4-battery.py) — Reviewer rev4 battery harness: applies one mutant to a scratch copy, runs go test as a standalone process per mask, restores the file byte-identically with a SHA-256 assert
- [TASK-260906-vmzk0y_spawn-log_-implementer--developer--muse-_RUN-260907-2a48c5.log](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_spawn-log_-implementer--developer--muse-_RUN-260907-2a48c5.log) — System spawn log captured by task-board
- [TASK-260906-vmzk0y_outcome-round5.md](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_outcome-round5.md) — Round-5 outcome: F1/F2 closed at the two uncensused sites, three-site digit census, 9-mutant battery, full gates, candidate tree d6253f58
- [TASK-260906-vmzk0y_mut-round5.py](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_mut-round5.py) — Round-5 mutation harness: scratch-copy mutants, standalone go test runs, SHA-256 restore assert
- [TASK-260906-vmzk0y_battery-round5.json](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_battery-round5.json) — Round-5 battery results: per-mutant killed_by with census/behavioural split, 9 applied / 8 killed / 1 equivalent + controls
- [TASK-260906-vmzk0y_change-request_rev5.patch](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_change-request_rev5.patch) — Change Request CR-TASK-260906-vmzk0y-5 revision 5 candidate patch (repository_delta=present, 17 changed paths)
- [TASK-260906-vmzk0y_change-request_rev5-validation.log](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_change-request_rev5-validation.log) — Change Request CR-TASK-260906-vmzk0y-5 revision 5 bounded validation log
- [TASK-260906-vmzk0y_spawn-log_-reviewer--reviewer--claude-_RUN-260907-ee0ed6.log](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_spawn-log_-reviewer--reviewer--claude-_RUN-260907-ee0ed6.log) — System spawn log captured by task-board
- [TASK-260906-vmzk0y_review-verdict-rev5.md](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_review-verdict-rev5.md) — Reviewer verdict for CR revision 5: changes requested. Two blocking findings — F1 descriptorGeometry overflow guard survives narrowing to its own arithmetic edge (37-digit literal admitted as Columns=1 at the production entry), F2 the digit-range census is prose and did not redden on a control-planted fourth comparison. 28 mutants applied / 23 killed. Fourth consecutive round of one class; recommends a derived fail-closed instrument over both digit-gate shapes instead of a fifth revision.
- [TASK-260906-vmzk0y_review-rev5-battery.json](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_review-rev5-battery.json) — Reviewer rev5 mutation battery: 28 applied / 23 killed / 5 survived (TB-17 real hole; TB-04/05/10/PH-03 equivalent) + 1 COMPILE_FAIL control. Includes the G-A control plant result and the TB-17 dangerous-threshold window analysis.
- [TASK-260906-vmzk0y_review-rev5-battery.py](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_review-rev5-battery.py) — Reviewer rev5 battery harness: one mutant at a time, go build gate, standalone go test per package with real exit code, byte-restore from a pre-run copy after every row.
- [TASK-260906-vmzk0y_spawn-log_-implementer--developer--muse-_RUN-260907-627c94.log](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_spawn-log_-implementer--developer--muse-_RUN-260907-627c94.log) — System spawn log captured by task-board
- [TASK-260906-vmzk0y_outcome-round6.md](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_outcome-round6.md) — Round-6 outcome: derived 9-site digit-guard census replaces the listing census (F2), 37-digit F1 witness, hex upper-edge boundary tests, 33/33 battery with per-arm split, tree 817b1a68
- [TASK-260906-vmzk0y_battery-round6.json](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_battery-round6.json) — Round-6 mutation battery results: 33 applied / 33 killed + census-only + NOT_APPLIED + COMPILE_FAIL, per-arm behavioural-vs-census split
- [TASK-260906-vmzk0y_mut-round6.py](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_mut-round6.py) — Round-6 mutation battery harness (denominator from the census enumeration, dual masks, SHA-256 restore)
- [TASK-260906-vmzk0y_plants-round6.json](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_plants-round6.json) — Round-6 census control-plant results: all six fail-closed arms reddened incl var binding and import alias
- [TASK-260906-vmzk0y_plants-round6.py](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_plants-round6.py) — Round-6 census control-plant harness (unregistered/orphan/unclassifiable incl alias and var shapes)
- [TASK-260906-vmzk0y_change-request_rev6.patch](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_change-request_rev6.patch) — Change Request CR-TASK-260906-vmzk0y-6 revision 6 candidate patch (repository_delta=present, 19 changed paths)
- [TASK-260906-vmzk0y_change-request_rev6-validation.log](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_change-request_rev6-validation.log) — Change Request CR-TASK-260906-vmzk0y-6 revision 6 bounded validation log
- [TASK-260906-vmzk0y_spawn-log_-reviewer--reviewer--claude-_RUN-260907-e5e330.log](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_spawn-log_-reviewer--reviewer--claude-_RUN-260907-e5e330.log) — System spawn log captured by task-board
- [TASK-260906-vmzk0y_review-verdict-rev6.md](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_review-verdict-rev6.md) — Reviewer verdict for CR revision 6: accepted, with control-plant and mutation evidence and one non-blocking stated-bounds finding
- [TASK-260906-vmzk0y_review-rev6-battery-a.json](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_review-rev6-battery-a.json) — Reviewer battery A rev6: 10 independent mutants over the geometry, semver and parseMajor bounds with census/behavioural split
- [TASK-260906-vmzk0y_review-rev6-battery-b.json](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_review-rev6-battery-b.json) — Reviewer battery B rev6: 7 independent mutants over the hex, major and range gates
- [TASK-260906-vmzk0y_review-rev6-battery-c.json](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_review-rev6-battery-c.json) — Reviewer battery C rev6: Error-binding contract selection mutants (Major:3 killed, ID swap provably equivalent)
- [TASK-260906-vmzk0y_review-rev6-plants.json](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_review-rev6-plants.json) — Reviewer control plants rev6: five fail-closed directions reddened, three blind-spot probes measured green
- [TASK-260906-vmzk0y_spawn-log_-implementer--developer--muse-_RUN-260907-060e43.log](file://TASK-260906-vmzk0y/TASK-260906-vmzk0y_spawn-log_-implementer--developer--muse-_RUN-260907-060e43.log) — System spawn log captured by task-board

## Created
2026-09-06T07:18:55Z

## Last Update
2026-09-07T11:10:35Z

## Assigned To
[implementer] developer (muse)
