## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(8))

## Blocked By
- TASK-260906-2okwyf

## Blocks
- TASK-260907-9ny4xl

## Checklist
- [x] The remaining independent environment-rule copies are converged onto internal/environ, or each retained copy carries a bidirectional agreement test that fails when the two diverge in either direction
- [x] The residue stated in TASK-260830-3bkz0c is closed or re-stated as a bound naming what is unproven and why, with the count re-derived rather than carried forward
- [x] environ drives its refusal exits through production entry points; the 27-of-100 exit coverage from the source Story is re-measured and reported
- [x] Every census derives its denominator from production and fails closed on an unregistered site, an orphan row and an unclassifiable site, each control-planted including an import alias and a var binding
- [x] Witnesses resolve to the site, not to a code-and-detail pair, using invcore runtime site recording
- [x] Mutation battery reports killed over applied on a production-derived denominator with narrowing, arm-deletion, census-only and audit-only separate, and NOT_APPLIED and COMPILE_FAIL as distinct rows
- [x] Candidate tree OID equals the record's, verified by detached-index write-tree, and the outcome enumerates exactly the CR changed paths
- [x] No untracked ungitignored scratch file is present in the worktree root or the candidate tree
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
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:08c5d671a0c9e2350b1b8c97c08841f378392d0f4ac8edf05ecf155d29ae97bb rationale="implementation class rank 1; fifth convergence leaf, the remaining environment-rule copies"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260907-cf8395, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260907-cf8395)
Leaf work handed off: outcome + battery log + mutants json + harness + mutant table attached. Candidate tree 8dbf62d7e4904da8078f94b0bb74a57787afc145 over cd8591d, 21 paths, worktree uncommitted. All gates exit 0 (race scoped to 7 pkgs, full-repo race not run). Battery 54/55 killed, 1 predicted survivor (R_digestnonstr, bound stated).
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260907-cf8395, pid=77365, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/xhigh pair_source=explicit match=not_recommended snapshot=sha256:e56c6ea07da9d1ce38c1188799bf45e3861c8eeac30ff4e8fceba3c2a0b95517 rationale="review class rank 1; environment convergence — judge re-measurement, bidirectionality and the scoped race gate"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260907-fb5fc8, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260907-fb5fc8)
Review rev1 (RUN-260907-fb5fc8): CHANGES REQUESTED -> to-dev. repeat-of: F3 repeats the TASK-260830-3bkz0c rev4 denominator-basis finding; F1/F2 new. Evidence TASK-260906-33xcnc_review-verdict-rev1.md.

BLOCKING F1 - the leaf own convergence gate is defeated by a token-preserving decoy. pinsCheckDelegation (shape_census_test.go:1044) requires only that a helper body reference the identifier environ; the alias audit requires only direct-call position. Plant in sessadapter/decode.go checkDigest: _, _ = environ.CheckDigest(raw) as decoy plus a full regrown local copy drifted to ParseDigest(strings.ToLower(value)). go test ./... -count=1 = 23 packages ok, WHOLE REPO GREEN, including TestCheckHelpersDelegateToEnviron, TestDelegatingWrappersCallEnviron, TestCensusScopeHasNoAliasedSharedRules and TestTupleAgreementAcrossFacades. Divergent through production entries: sessadapter.DecodeTuple ADMITS an uppercase-hex store_schema_fingerprint that environ.DecodeTuple REFUSES. No token-preserving mutant exists for either new structural delegation gate.

BLOCKING F2 - the retained provhost frame decoder justification is measurably false. Surrogate gate: claim holds, both directions reddened by my plants. Frame decoder duplicate-member arm: the frame battery has exactly one duplicate row, key v. Narrowing sweep duplicate && key != K on protocol.go:295 over 8 real member names: 6 SURVIVED (body, ok, protocol_version, error, capabilities, provider_id), 2 KILLED (v, request_id). With key != body and the full suite green, provhost.DecodeResponse admits a frame carrying two body members and returns the SECOND (last-wins). The producer own M15n_framedupnarrow admits exactly a duplicated v, i.e. aimed at the single witnessed member: it measures the witness, not the class. Same class in the shared owner environ.DecodeStrictObject:84 - 7 of 10 keys survived. Also: frozen by accepted leaves 2/4 is not a board mechanic - leaves 1-4 are at cd8591d, a leaf-5 edit cannot disturb an accepted revision.

BLOCKING F3 - G-A denominator uses the basis a prior review already corrected. 31 refuse sites + 59 bool-false = 90 reproduces exactly under my own count. The 11 &Fault{} returns in DecodeStrictObject (decode.go 60,63,68,72,78,82,85,89,94,96,99) are still excluded. 89+11=100 is the 27-of-100 figure; the outcome calls 100 two stories stale when it is the CORRECTED value of the same inventory. Correct now = 101 (105 counting bool return <expr> arms). Tell holds: N1/M15/M15n/M16/M17/M17n all weaken exits inside the excluded 11.

NON-BLOCKING N1 (G-C closed by reviewer): I ran full-repo -race myself in two bounded calls - 20 pkgs 45s, then canonicaljson+localstore+tracecheck 2m05. ALL 23 PACKAGES GREEN. The stated bound does not support itself: canonicaljson was already inside the 7 run, and the remaining 16 cost 45s. N2 (G-D): R_digestnonstr bound rests on ParseDigest("") refusing, which no test pins (scalar_test.go:158-164 has 5 negative vectors, none empty) - rationale, not proven property. N3: the stale tree 8dbf62d7/21 paths is in the HANDOFF NOTE, not the outcome document; the document names 5063b78a/22 paths and git diff between the trees is LOGBOOK.md only (+9 lines), so no measurement was computed against a differing code path - correction confirmed. N4: refuse became a mutable package-level var to serve an instrument; package-private, swapped in TestMain before m.Run, environ green under -race - noted, not a defect.

HELD UNDER MY OWN PLANTS: scalar retained copy both directions RED; provhost surrogate both directions RED; canonicaljson both directions RED; refusal-site audit RED on an unexercised production refuse site (names tuple.go:135, closed rule set 39->40) and RED on deny := refuse alias; census RED on var-binding alias, RED on fresh-name grammar copy (unregistered grammar copy under a fresh name), RED on orphan ledger row.

PROVENANCE: candidate tree 5063b78a32fd32c25d4bd3f1bcc911951c0c8900 verified by my own detached-index write-tree = CR record; 22 changed paths match the CR in both directions; no untracked ungitignored file beyond the 6 new test files; worktree root clean; tree restored byte-identical after every plant. Untouched-candidate gates: build 0, vet 0, gofmt clean, go test ./... 23 ok, go test ./... -race 23 ok.

AC COVERAGE MEASURED: 4 of 6 rows driven. NOT SATISFIED: provhost frame decoder (F2), sessadapter/dirnode per-helper copies (F1).
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260907-fb5fc8, pid=39092, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:08c5d671a0c9e2350b1b8c97c08841f378392d0f4ac8edf05ecf155d29ae97bb rationale="implementation class rank 1; round 2, the delegation gate accepts a decoy and the duplicate-member class is live in the owner"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260907-552c28, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260907-552c28)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260907-552c28, pid=51849, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/xhigh pair_source=explicit match=not_recommended snapshot=sha256:e56c6ea07da9d1ce38c1188799bf45e3861c8eeac30ff4e8fceba3c2a0b95517 rationale="review class rank 1; round 2, decoy caught and owner duplicates closed on a sample — judge the class"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260907-911b99, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260907-911b99)
Review rev2 ACCEPTED (RUN-260907-911b99). Evidence: TASK-260906-33xcnc_review-verdict-rev2.md. G-A: five decoys planted in sessadapter/decode.go — A1 (discard+agreeing local) and A4 (var binding) caught by the STRUCTURAL gate alone, A2 by both, A3 (import alias to a decoy package) and A5 (argument mutation) caught behaviourally/by the alias audit but NOT structurally. Structural half is load-bearing. G-B: 30/30 owner keys and 10/10 provhost keys killed under duplicate&&key!=K, each on its own row; row set derived over the member set; rev1 double-body exploit refused; retained copy additionally pinned per key via DecodeManifest for keys absent from its own sweep. G-C: denominator re-derived in source 31+59+11=101 (105 with 4 bool-expr arms); the +1 is the new parseUint53Literal empty guard, killed by delete AND narrow. G-D: full-repo go test -race 23/23 ok, run by reviewer; race bound withdrawn. G-E: detached-index write-tree = ca0bf6b6 = record, 25 paths both directions, no scratch; rev2 production delta byte-identical to rev1 (24208 bytes each side). Bounds recorded, non-blocking: B1 delegation gate does not constrain the delegated call arguments; B2 shape census rune-measure detector is keyed to utf8.RuneCountInString only, a for-range counting loop planted in sessadapter/manifest.go walks the whole suite (no live site of that spelling exists in any of the 7 census packages); B3 TestEveryRefusalSiteIsExercised only fires on a full package run, a -run mask hides an unexercised derived site.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260907-911b99, pid=30103, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:08c5d671a0c9e2350b1b8c97c08841f378392d0f4ac8edf05ecf155d29ae97bb rationale="implementation class; checkpoint-only run for the accepted non-final leaf"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260907-f88aa8, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260907-f88aa8)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260907-f88aa8, pid=47669, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260906-33xcnc_spawn-log_-implementer--developer--muse-_RUN-260907-cf8395.log](file://TASK-260906-33xcnc/TASK-260906-33xcnc_spawn-log_-implementer--developer--muse-_RUN-260907-cf8395.log) — System spawn log captured by task-board
- [TASK-260906-33xcnc_outcome.md](file://TASK-260906-33xcnc/TASK-260906-33xcnc_outcome.md)
- [TASK-260906-33xcnc_battery.log](file://TASK-260906-33xcnc/TASK-260906-33xcnc_battery.log) — Consolidated mutant battery log: 54 killed / 55 applied, denominator 90
- [TASK-260906-33xcnc_mutants.json](file://TASK-260906-33xcnc/TASK-260906-33xcnc_mutants.json) — Mutant battery machine-readable results
- [TASK-260906-33xcnc_mutate.py](file://TASK-260906-33xcnc/TASK-260906-33xcnc_mutate.py) — Mutant battery harness (exact-match appliers, restores verified)
- [TASK-260906-33xcnc_mutant-table.md](file://TASK-260906-33xcnc/TASK-260906-33xcnc_mutant-table.md) — Mutant evidence table: mutant, narrowing, named test, status, killed-by
- [TASK-260906-33xcnc_change-request_rev1.patch](file://TASK-260906-33xcnc/TASK-260906-33xcnc_change-request_rev1.patch) — Change Request CR-TASK-260906-33xcnc-1 revision 1 candidate patch (repository_delta=present, 22 changed paths)
- [TASK-260906-33xcnc_change-request_rev1-validation.log](file://TASK-260906-33xcnc/TASK-260906-33xcnc_change-request_rev1-validation.log) — Change Request CR-TASK-260906-33xcnc-1 revision 1 bounded validation log
- [TASK-260906-33xcnc_spawn-log_-reviewer--reviewer--claude-_RUN-260907-fb5fc8.log](file://TASK-260906-33xcnc/TASK-260906-33xcnc_spawn-log_-reviewer--reviewer--claude-_RUN-260907-fb5fc8.log) — System spawn log captured by task-board
- [TASK-260906-33xcnc_review-verdict-rev1.md](file://TASK-260906-33xcnc/TASK-260906-33xcnc_review-verdict-rev1.md) — Reviewer verdict for CR rev1: CHANGES REQUESTED. 3 blocking findings (decoy-defeated delegation gate with permissive facade divergence; provhost duplicate-member arm pinned at 2 of 8 keys with reachable last-wins exploit; denominator basis repeat), 4 non-blocking. Full-repo -race run by the reviewer: 23 packages green.
- [TASK-260906-33xcnc_spawn-log_-implementer--developer--muse-_RUN-260907-552c28.log](file://TASK-260906-33xcnc/TASK-260906-33xcnc_spawn-log_-implementer--developer--muse-_RUN-260907-552c28.log) — System spawn log captured by task-board
- [TASK-260906-33xcnc_outcome-round2.md](file://TASK-260906-33xcnc/TASK-260906-33xcnc_outcome-round2.md) — Round-2 outcome: F1/F2/F3 closed test-only, candidate ca0bf6b6 over 25 paths, 70/71 battery on 101-exit denominator
- [TASK-260906-33xcnc_battery-round2.log](file://TASK-260906-33xcnc/TASK-260906-33xcnc_battery-round2.log) — Round-2 full mutant battery log: 70/71 killed, denominator 101
- [TASK-260906-33xcnc_mutants-round2.json](file://TASK-260906-33xcnc/TASK-260906-33xcnc_mutants-round2.json) — Round-2 battery results JSON with per-mutant status, suites, and fails
- [TASK-260906-33xcnc_mutate-round2.py](file://TASK-260906-33xcnc/TASK-260906-33xcnc_mutate-round2.py) — Round-2 mutant battery harness with D1/D2/E/P mutants and corrected 101 denominator
- [TASK-260906-33xcnc_mutant-table-round2.md](file://TASK-260906-33xcnc/TASK-260906-33xcnc_mutant-table-round2.md) — Round-2 mutant table: mutant, narrowing, named failing test, bound for the survivor
- [TASK-260906-33xcnc_change-request_rev2.patch](file://TASK-260906-33xcnc/TASK-260906-33xcnc_change-request_rev2.patch) — Change Request CR-TASK-260906-33xcnc-2 revision 2 candidate patch (repository_delta=present, 25 changed paths)
- [TASK-260906-33xcnc_change-request_rev2-validation.log](file://TASK-260906-33xcnc/TASK-260906-33xcnc_change-request_rev2-validation.log) — Change Request CR-TASK-260906-33xcnc-2 revision 2 bounded validation log
- [TASK-260906-33xcnc_spawn-log_-reviewer--reviewer--claude-_RUN-260907-911b99.log](file://TASK-260906-33xcnc/TASK-260906-33xcnc_spawn-log_-reviewer--reviewer--claude-_RUN-260907-911b99.log) — System spawn log captured by task-board
- [TASK-260906-33xcnc_review-verdict-rev2.md](file://TASK-260906-33xcnc/TASK-260906-33xcnc_review-verdict-rev2.md) — Reviewer verdict for CR rev2: accepted; G-A five-decoy delegation probe, G-B 30+10 key duplicate sweep, G-C denominator re-derived at 101, G-D full-repo race, G-E tree match, three recorded bounds
- [TASK-260906-33xcnc_spawn-log_-implementer--developer--muse-_RUN-260907-f88aa8.log](file://TASK-260906-33xcnc/TASK-260906-33xcnc_spawn-log_-implementer--developer--muse-_RUN-260907-f88aa8.log) — System spawn log captured by task-board
- [TASK-260906-33xcnc_checkpoint-report.md](file://TASK-260906-33xcnc/TASK-260906-33xcnc_checkpoint-report.md) — Checkpoint-only run report for CR rev 2

## Created
2026-09-06T11:42:21Z

## Last Update
2026-09-07T11:10:35Z

## Assigned To
[implementer] developer (muse)
