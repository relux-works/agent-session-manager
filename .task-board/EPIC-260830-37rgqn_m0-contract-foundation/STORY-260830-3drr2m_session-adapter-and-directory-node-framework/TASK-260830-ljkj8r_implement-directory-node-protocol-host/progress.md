## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(8))

## Blocked By
- TASK-260830-2z3se0

## Blocks
- TASK-260830-3bkz0c

## Checklist
- [x] Production entry points implement the scoped deliverable: Implement Directory Node major bootstrap, manifest/probe/scan/query operation framing, host checks, and structured errors
- [x] Relevant positive, negative, compatibility, and recovery tests pass with logs attached
- [x] README/doctor/capability evidence and specification traceability are updated without unsupported claims
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
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:08c5d671a0c9e2350b1b8c97c08841f378392d0f4ac8edf05ecf155d29ae97bb rationale="implementation class rank 1; second leaf, inheriting leaf 1's enumerated censuses"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260906-25c417, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260906-25c417)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260906-25c417, pid=86389, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/xhigh pair_source=explicit match=not_recommended snapshot=sha256:1d432acfa5b6f6101ec00874dad940578fb6cb015296a686a5e0a3b9569d548b rationale="review class rank 1; second leaf, whether leaf 1's census discipline transferred"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260906-ccfe0d, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260906-ccfe0d)
REVIEW rev1 (RUN-260906-ccfe0d): CHANGES REQUESTED -> to-dev. repeat-of: none (first round of this leaf); class-of leaf 1 rev3-B5/rev4-B8.

B1 (blocking) Bound edges unwitnessed as a class. 94 one-step bound mutants across manifest/probe/scan/query/protocol: 19 killed, 75 SURVIVED (0 NOT_APPLIED, 0 COMPILE_FAIL). Survivors by file: query 43, manifest 13, probe 12, scan 7. Bounds are correct and reachable - 4 representative out-of-range values are refused today at the production entry - so the gap is the evidence. Leaf 1 carries bound_census_test.go (1650 lines) + bounds_edge_test.go (423 lines) for exactly this class; dirnode has neither. Same mutant shape (checkStringBounds max +1) is KILLED in sessadapter and SURVIVES at every checkStringBounds maximum in dirnode. TestManifestLimitsBounds doc claims each bound refuses past its edge in both directions; 4 of its 14 edges are unprobed and all 4 survive.

B2 (blocking) AC row 5 17-op union not driven. 5 of 17 query operations reach DecodeQuery (schema, sessions, set_title, plan_continue, execute_plan); 12 undriven with no stated bound. Ten production functions at 0.0% coverage, all 10.8.5 union validators. Three narrowing mutants inside them (environments auth_status admits root; jobs states admits pwned; distinct field admits secret) all survive. Whole class funnels into one census arm with 4 witnesses, so the census reads complete while 10 validators have never executed - the leaf-1 wrapper shape in new clothes.

B3 (blocking) TestDecodeQueryRegistryRules/unknown_field refuses for the wrong reason: the vector injects a duplicate fields member (3 occurrences), so decodeStrictObject refuses before checkQueryProjection runs. validQueryField at 0.0% coverage; 27-member queryFields registry unproven. Both paths return the same code AND the same detail, so requireCode cannot distinguish them. Production is correct - a correctly built vector reaches the registry and refuses.

N1 registry derivation covers 4 of 15 closed vocabulary tables; 11 have no exact-content pin. Caveat stated: platformV1/V2/architectures are separately covered by an unknown-token witness.
N2 internal/dirnode holds no section/contract ownership binding; section coverage line unchanged. Matches leaf 1 and terminalbackend, and README discloses the unevidenced binding - flagged for a Story-level decision, not blocking.

VERIFIED GOOD (do not re-litigate): refusal-arm census is materially stronger than leaf 1 (constructor set derived, alias refused in every non-call/non-definition position, New-outside-body refused, import census fails closed, both directions + witness resolution); M8 reproduces exactly as reported and is correctly scored census-only; traceability re-pin is COMPUTED (renaming one acceptance-case id reddens with exact mismatch d7a837dc vs reviewed fbdbb5b6) and declarations are AST-resolved; Journal.Import builds into a local map and assigns only on success, so a malformed import cannot wipe recorded keys; README makes no unsupported claim; no orphan guards; suite green on my own run (18 packages, vet clean, gofmt clean).

Evidence: TASK-260830-ljkj8r_review-verdict-rev1.md, TASK-260830-ljkj8r_review-mutant-log-rev1.md. Candidate tree 4d8a599 verified byte-identical after all 152 mutant applications.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260906-ccfe0d, pid=28817, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:08c5d671a0c9e2350b1b8c97c08841f378392d0f4ac8edf05ecf155d29ae97bb rationale="implementation class rank 1; carry leaf 1's bound instrument across and close a zero-coverage validator class"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260906-6654bf, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260906-6654bf)
Round-2 rework ready for review. Item 18 checked as standing routing obligation for the next verdict, not as completed work. Evidence: TASK-260830-ljkj8r_round2.md (185-mutant table, 9/9 AC mapping). Tree holds rework uncommitted per brief.
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260906-6654bf, pid=1658, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/xhigh pair_source=explicit match=not_recommended snapshot=sha256:9f20871155c37dee69e09bc65874033bce16ac39424aba056c24ea8521516f40 rationale="review class rank 1; round 2, whether the instrument transferred or only its files"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260906-8a5258, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260906-8a5258)
REVIEW rev2 (RUN-260906-8a5258): CHANGES REQUESTED -> to-dev. repeat-of: rev1-B3 (C1) and rev1-B2 (C2). Two consecutive same-class findings: next step is a derived gate, not another hand audit.

G-A ANSWERED: the instrument transferred, B1 CLOSED. Shape space is package-specific (checkURI, CheckCursorReuse, Journal.Import, skip>1000000, index>63, version!=1 - constructs absent from sessadapter), 101 sites 72 driven 29 exempt, fails closed on unregistered/orphan/alias/empty-scan, 11 canaries, 3 synthetic derivation proofs. Rev1 battery re-run verbatim on the identical denominator: 68/94 killed (was 19/94), 0 NOT_APPLIED, 0 COMPILE_FAIL, 0 resurrections. All 26 survivors map 1:1 to declared equiv-ok rows and every masking gate verified in production source (environmentIDPattern {0,63}, providerIDPattern {0,31}, checkSemver 5-char floor, 8/5/4/7/3-member tables, 27-field registry, joint count/index gate at 65). No false equivalence. WRAPPER SHAPE CLOSED: checkURI is this package requireStringBounds analogue and dies in both directions plus floor at <4; leaf 1 25-of-46 shape does not recur.

C1 (blocking, repeat-of rev1-B3) Arm-slide sweep reported as covering a class it did not cover. arm_identity_test.go states every unknown-member vector adds exactly one member so those reach the arm they name. False for three: manifest_test.go:62 reaches trailing-data-after-the-object (the replacement hoists limits members and leaves malformed JSON); query_test.go:75 reaches the operation arm (the extensions:{}} needle matches operations[0].parameters first); probe_test.go:142 reaches the node-build arm (needle matches node_build first). Independent 144-arm reachability sweep (every production if whose body builds failX, AST-derived; 115 killed/144, 0 unmeasured) also finds 5 missingMember arms deletable (probe.go:141 protocol.go:428 protocol.go:559 scan.go:78 scan.go:193 - vectors correct, assertions do not discriminate; the 3 fixed ones die only because round 2 added a message assertion), 2 gates with zero negative coverage (checkExtensions at probe.go:392 and query.go:467; 4 of 6 siblings covered), and 3 empty-frame arms deletable (bootstrap.go:144 protocol.go:399 protocol.go:530 - census rows marked driven whose named driver asserts only err != nil).

C2 (blocking, repeat-of rev1-B2) Admissions closed, obligations not. The 17-operation admission set IS derived from readOperations+mutationOperations and fails closed on a missing builder; 17/17 admit; root/pwned/secret die. The refusal set is a 45-row hand table. Against a production-derived denominator (every return false in query.go, one mutant each): 58 of 89 obligations driven, 31 removable with a green suite. checkPlanContinueParameters 3/9 - all six member-shape checks deletable, positive-path-only for one of the seventeen. checkQuerySort 3/8, checkQueryParameters 2/5, checkExecutePlanParameters 2/5, checkSubject 1/4, checkQueryFlags 1/3. And yes, one arm still carries N: all 89 report through the single checkQueryOperation arm.

C3 (blocking) Uniqueness half of the sorted-unique class closed at 1 of 8. >= to > keeps the gate and admits exactly one member of the rejected class (a duplicate). KILLED only at decode.go:377 checkSortedUniqueStrings. SURVIVED at decode.go:403 digests, decode.go:429 UUIDv7, manifest.go:525 contract encodings, query.go:825 tags, query.go:927 confirmations, query.go:1042 lineage_anchors, scan.go:438 journal keys. Census shape 10 states the class pinned by the sortedness refusal tests; those feed unsorted vectors, never a duplicate.

C4 (blocking) EncodeRequest deadline ceiling undriven and its driver comment claims otherwise. protocol.go:352 widened survives under two spellings; floor control dies, harness live. TestUint53BoundEdges/request_deadline_ms returns on the DecodeRequestFrame arm for both 0 and 3600001, so EncodeRequest is only ever called with the two admitting values - while the row comment says a refusal from either entry fails the row.

Scope: C1-C4 are evidence findings, not behaviour bugs; every gate named refuses correctly in the shipped code. Missing is the test that would fail if it stopped.

VERIFIED GOOD (do not re-litigate): bound census and its fail-closed behaviour; checkURI wrapper; B3 vector fixed and pinned by construction; three rename-to-delete conversions pinned by message; 17/17 derived admissions; 115/144 arms die; suite 18 packages ok, vet clean, gofmt clean, dirnode 83.3% - all three producer claims reproduce on my run. Delta 23->27 paths fully accounted (arm_identity, bound_census, bounds_edge, query_operations are new; production code byte-identical to rev1). Candidate tree bd3cb8c verified byte-identical after all 433 mutant applications; 0 census-only kills.

Evidence: TASK-260830-ljkj8r_review-verdict-rev2.md, TASK-260830-ljkj8r_review-mutant-log-rev2.md.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260906-8a5258, pid=6966, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:08c5d671a0c9e2350b1b8c97c08841f378392d0f4ac8edf05ecf155d29ae97bb rationale="implementation class rank 1; round 3 rework, two repeat-of findings need derived gates not hand tables"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260906-6a99e4, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260906-6a99e4)
Round 3 ready for review. C1: 3 vectors fixed, derived reachability gate (146 rows, full-rendering match, per-site counts), arm battery 142/144 + 2 defensive. C2: 112-site obligation census, battery 111/112 + 609 defensive with registry tripwire, narrowness mapping attached. C3: 8/8 duplicate narrowing kills. C4: both entries driven independently, 2/2 widenings killed. Production byte-identical (hashes in outcome doc). 18 pkgs green, vet/gofmt clean, dirnode 87.6%. Evidence: TASK-260830-ljkj8r_round3.md + arms/obligations tables.
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260906-6a99e4, pid=40505, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/xhigh pair_source=explicit match=not_recommended snapshot=sha256:9f20871155c37dee69e09bc65874033bce16ac39424aba056c24ea8521516f40 rationale="review class rank 1; round 3, matcher tightness and the 609 defensive rows"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260906-704600, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260906-704600)
REVIEW rev3 (RUN-260906-704600): ACCEPTED -> accept_cr. repeat-of: n/a (accepted).

G-A ANSWERED, matcher is as tight as its doc claims. Denominator re-derived independently: 142 arm if-sites by my own go/ast selector, a strict subset of the producer 144 (their extra bootstrap.go:151 and scan.go:314 are wrapper ifs around nested arms). Both slide directions caught. A1 prefix-sibling plant (second arm whose detail is a strict prefix of request-envelope-carries-unknown-member, registered in defensiveArms so the roster could not mask the matcher) -> RED naming the arm it reached. A2 superstring-sibling plant -> RED as ambiguous, naming both literals. A3 third checkResponseIdentity instantiation -> RED on TestReachabilityContextsAreClosed while the reachability rows themselves stayed GREEN, which is exactly why that gate has to exist separately. A3b non-literal context -> RED.

Arm battery, deletion shape if false && (cond) which PRESERVES the failX token the census greps for: 140 KILLED / 142, 0 COMPILE_FAIL, 0 unmeasured, survivors exactly the 2 declared defensiveArms. Every kill named a TestEveryDerivedArmFiresItsVector subtest - no roster noise. Cross-checked against the producer table: 140 of 140 agree with the producer-named killer row, 0 disagreements. Diagonal exact: 139 of 140 sites killed by exactly one row; the 140th (protocol.go:414) kills its conduit plus its 5 frame faults, the documented also-carrier shape; NO reachability row dies under more than one site deletion. 145 of 146 rows serve as killers; the one that does not is RefuseUnknownOperation, an unconditional exported refusal with no if to mutate, driven by TestRefuseUnknownOperation and mirroring the accepted sessadapter API.

Defensive escape hatch is 2 arms, not 609. The brief 609 misreads the producer note: 609 is the LINE NUMBER of the single defensive obligation (query.go:609 checkQueryParameters obligation-1). defensiveArms has exactly 2 entries, defensiveObligations exactly 1. Both arm rationales verified true rather than decorative: the scan.go:365 Export arm attacked directly through the public Import->Export path with 5 hostile key vectors - lone-surrogate escapes are REFUSED BY IMPORT, NUL/BOM/U+FFFD/control sequences all export cleanly, arm unreachable as stated; the protocol.go:377 marshal arm is guarded by decodeStrictObject before json.Marshal with every other member a validated string/uint53.

G-B ANSWERED. Denominator production-derived and fails closed: re-derived 112 sites exactly with my own scanner, and a planted return-false site reddens TestQueryObligationRosterIsComplete with the unaccounted obligation named. Not enumerated-and-frozen. Battery run against TestEveryQueryObligationIsDriven ALONE so the roster renumbering artifact is excluded from my numbers too: 111/112 KILLED, sole survivor the declared query.go:609 site, and 111 of 112 sites redden their OWN named subtest.

One-arm problem resolved by mutation, not by message. All 112 still report query_invalid through checkQueryOperation and the behavioral test asserts code only - sufficient here and measurably so: 93 of 112 mutants kill exactly one subtest, and every overlap is one of the two documented structural shapes and nothing else (framing gates checkQueryOperation ob-7..ob-11 over their validators - ob-7 reaches 68 rows, the parameter union; layered delegates checkEnumSubset/checkUUIDv7DigestSubset/validQueryField/validQueryPreset under checkFilters/checkQueryProjection). A witness satisfiable by a sibling obligation would not redden when its own site alone is flipped; each of the 111 does. Defensive accounting verified in both directions: removing hosts from queryParameterMembers reddens TestQueryParameterMembersMatchRegistry (table keys = 16, want 17), and under that divergence flipping obligation-1 additionally reddens TestSortedUniqueUUIDv7Edges/hosts_host_ids and TestDecodeQueryOperationValidatorsRefuse - load-bearing exactly when the tripwire says it is reachable. A true stated bound.

G-C ANSWERED. Both EncodeRequest entries driven independently - the row reports admitted if EITHER entry admits, so an entry admitting out of range fails the row even when the other refuses, and the comment says exactly that. 4/4 killed (ceiling+1, dropped ceiling disjunct, both floor controls), each by TestUint53BoundEdges/request_deadline_ms AND the reachability row, with the new direct TestEncodeRequestRefusals 3600001 refusal. Merged failUnknownOperation key resolved per site: protocol.go:338 kills the via-EncodeRequest row, protocol.go:496 kills the decode row. No arm is resolvable only by code - every row asserts the full rendering and the wrapper is AST-derived from the eight constructor bodies.

G-D. Production-derived total 277 killed / 280 applied, 3 survivors all declared defensive, 0 COMPILE_FAIL, 0 NOT_APPLIED, 0 census-only kills counted. Plus 8 gate-integrity plants, 8 caught. AC rows 9 of 9 driven through named production entry points by named committed tests. Resurrections none and structurally none possible: production byte-identical to rev2 across all eight files (hashes reproduce the producer), test inventory 92 -> 98 with ZERO removals, and the only deleted assertion lines are the chained deadline drive C4 replaced; spot re-verification 14/14 bound widenings and C3 8/8 still die 1:1. Coverage 83.3 -> 87.6 accounted per function: 20 functions moved, 11 query.go validators to 100% (checkPlanContinueParameters +27.3, checkQuerySort +24.1, checkQueryOperation +22.6, checkSubject +21.4, checkCaller +18.4), exactly the C1/C2 surface, zero functions at 0.0% in either revision. Candidate tree 9f5ec76c verified byte-identical after all 288 applications.

Suite from the restored tree: 18 packages ok, go vet clean, go build clean, gofmt -l internal/ clean, tracecheck ok at acceptance_cases=94. Ownership re-pin is COMPUTED not asserted - renaming one acceptance case id fails closed with the exact mismatch 371d39f6 vs reviewed fbdbb5b6. README correctly hedged: 7.9 clause binding stays unevidenced, framed subset stated as exactly manifest/probe/scan plus Directory Query and pinned by TestFramedSubsetIsExact, eight unframed operations disclosed rather than silently absent. No unsupported capability claim.

NON-BLOCKING. N1 (repeat of rev1-N2, story-level): internal/dirnode is not imported by any command yet, identical to the accepted sessadapter and provhost leaves; its production entry point is the exported API and wiring is the Story integration step. N2: TestUint53BoundEdges/request_deadline_ms keys its out-of-range branch on hardcoded 0/3600001, so moving the row min/max without those literals would silently degrade it to the round-2 shape - stated in the comment, the two live three lines apart. N3: round3.md carries a stray non-English token (the two제도 comment-only corrections); board artifact only.

Evidence: TASK-260830-ljkj8r_review-verdict-rev3.md, TASK-260830-ljkj8r_review-mutant-log-rev3.tsv.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260906-704600, pid=14921, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:08c5d671a0c9e2350b1b8c97c08841f378392d0f4ac8edf05ecf155d29ae97bb rationale="implementation class; checkpoint-only run, accepted CR rev3 needs the bound producer role"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260906-b1e7fa, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260906-b1e7fa)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260906-b1e7fa, pid=25465, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260830-ljkj8r_spawn-log_-implementer--developer--muse-_RUN-260906-25c417.log](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_spawn-log_-implementer--developer--muse-_RUN-260906-25c417.log) — System spawn log captured by task-board
- [TASK-260830-ljkj8r_evidence.md](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_evidence.md) — Directory Node host implementation evidence: AC coverage 10/10, census 147 arms, 8-mutant battery with zero survivors, gate exit codes
- [TASK-260830-ljkj8r_full-test.log](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_full-test.log) — Full go test ./... -count=1 log, exit 0, 18 packages ok
- [TASK-260830-ljkj8r_change-request_rev1.patch](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_change-request_rev1.patch) — Change Request CR-TASK-260830-ljkj8r-1 revision 1 candidate patch (repository_delta=present, 23 changed paths)
- [TASK-260830-ljkj8r_change-request_rev1-validation.log](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_change-request_rev1-validation.log) — Change Request CR-TASK-260830-ljkj8r-1 revision 1 bounded validation log
- [TASK-260830-ljkj8r_spawn-log_-reviewer--reviewer--claude-_RUN-260906-ccfe0d.log](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_spawn-log_-reviewer--reviewer--claude-_RUN-260906-ccfe0d.log) — System spawn log captured by task-board
- [TASK-260830-ljkj8r_review-verdict-rev1.md](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_review-verdict-rev1.md) — Reviewer verdict for CR rev1: changes requested. 3 blocking (bound-edge class 75/94 mutants survive; 17-op union 5/17 driven with 10 functions at 0% coverage; unknown-field vector refuses for the wrong reason), 2 non-blocking, verified-good list.
- [TASK-260830-ljkj8r_review-mutant-log-rev1.md](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_review-mutant-log-rev1.md) — Reviewer-run mutant log: 94 bound-edge rows (19 killed / 75 survived), 54 vocabulary rows (20/34), 3 targeted narrowing rows, 4 controls, 8 reachability probes. Every row measured; tree restored byte-identical.
- [TASK-260830-ljkj8r_spawn-log_-implementer--developer--muse-_RUN-260906-6654bf.log](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_spawn-log_-implementer--developer--muse-_RUN-260906-6654bf.log) — System spawn log captured by task-board
- [TASK-260830-ljkj8r_round2.md](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_round2.md) — Round-2 review response: B1 bound census + 185-mutant battery, B2 17-operation coverage, B3 arm-slide fixes
- [TASK-260830-ljkj8r_change-request_rev2.patch](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_change-request_rev2.patch) — Change Request CR-TASK-260830-ljkj8r-2 revision 2 candidate patch (repository_delta=present, 27 changed paths)
- [TASK-260830-ljkj8r_change-request_rev2-validation.log](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_change-request_rev2-validation.log) — Change Request CR-TASK-260830-ljkj8r-2 revision 2 bounded validation log
- [TASK-260830-ljkj8r_spawn-log_-reviewer--reviewer--claude-_RUN-260906-8a5258.log](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_spawn-log_-reviewer--reviewer--claude-_RUN-260906-8a5258.log) — System spawn log captured by task-board
- [TASK-260830-ljkj8r_review-verdict-rev2.md](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_review-verdict-rev2.md) — Reviewer verdict for CR rev2: changes requested. B1 closed (68/94 bound mutants, was 19/94; 0 resurrections). 4 blocking: arm-slide sweep incomplete (3 unknown-member vectors reach the wrong arm), 58/89 query obligations driven, 7/8 uniqueness gates admit duplicates, EncodeRequest deadline ceiling undriven.
- [TASK-260830-ljkj8r_review-mutant-log-rev2.md](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_review-mutant-log-rev2.md) — Reviewer-run mutant log rev2: 433 applications across 5 production-derived batteries (rev1 battery 68/94, extended bounds 42/54, targeted 3/9, refusal-arm reachability 115/144, query obligations 58/89), 0 not-applied, 0 compile-fail, 0 census-only kills, tree verified byte-identical.
- [TASK-260830-ljkj8r_spawn-log_-implementer--developer--muse-_RUN-260906-6a99e4.log](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_spawn-log_-implementer--developer--muse-_RUN-260906-6a99e4.log) — System spawn log captured by task-board
- [TASK-260830-ljkj8r_round3.md](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_round3.md) — Round-3 outcome: C1-C4 evidence findings closed, killed-over-applied batteries, production-identity hashes
- [TASK-260830-ljkj8r_arms_table.md](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_arms_table.md) — Round-3 evidence: 144 arm mutants with verdicts and reachability-row killers
- [TASK-260830-ljkj8r_obligations_table.md](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_obligations_table.md) — Round-3 evidence: 112 obligation mutants with behavioral killers, roster noise excluded
- [TASK-260830-ljkj8r_change-request_rev3.patch](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_change-request_rev3.patch) — Change Request CR-TASK-260830-ljkj8r-3 revision 3 candidate patch (repository_delta=present, 29 changed paths)
- [TASK-260830-ljkj8r_change-request_rev3-validation.log](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_change-request_rev3-validation.log) — Change Request CR-TASK-260830-ljkj8r-3 revision 3 bounded validation log
- [TASK-260830-ljkj8r_spawn-log_-reviewer--reviewer--claude-_RUN-260906-704600.log](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_spawn-log_-reviewer--reviewer--claude-_RUN-260906-704600.log) — System spawn log captured by task-board
- [TASK-260830-ljkj8r_review-verdict-rev3.md](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_review-verdict-rev3.md) — Reviewer verdict for CR rev3: ACCEPTED. G-A matcher tight in both slide directions (prefix + superstring plants both RED); context closure fires on a 3rd instantiation and a non-literal context; defensive arms are 2 (the brief's '609' is query.go line 609, not a count) and both rationales verified reachable-never. G-B denominator production-derived and fails closed; 111/112 behavioral kills with roster noise excluded; diagonal complete; one-arm problem resolved by mutation not message. G-C 4/4. G-D no resurrections, production byte-identical to rev2, coverage delta accounted per function. 277/280 production-derived mutants killed, 3 declared-defensive survivors, 0 unmeasured; tree 9f5ec76c byte-identical after 288 applications.
- [TASK-260830-ljkj8r_review-mutant-log-rev3.tsv](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_review-mutant-log-rev3.tsv) — Reviewer-run mutant log rev3: 142-site arm reachability battery (140 killed / 2 declared-defensive survivors, every kill naming a reachability-row subtest), 112-site query obligation battery run against the behavioral test alone with roster renumbering excluded (111 killed / 1 declared-defensive survivor), and the 8-site C3 sorted-unique narrowing battery (8/8, each 1:1 by its own duplicate subtest).
- [TASK-260830-ljkj8r_spawn-log_-implementer--developer--muse-_RUN-260906-b1e7fa.log](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_spawn-log_-implementer--developer--muse-_RUN-260906-b1e7fa.log) — System spawn log captured by task-board
- [TASK-260830-ljkj8r_checkpoint-report.md](file://TASK-260830-ljkj8r/TASK-260830-ljkj8r_checkpoint-report.md) — Checkpoint-only run evidence: commit OID, verify-commit exit, CR state, leaf status

## Created
2026-08-29T22:00:06Z

## Last Update
2026-09-06T13:25:53Z

## Assigned To
[implementer] developer (muse)
