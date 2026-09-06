## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(8))

## Blocked By
- TASK-260830-7a0s2c

## Blocks
- TASK-260830-35urbp
- TASK-260830-2bnr39
- TASK-260830-2u34k1
- TASK-260830-355gon

## Checklist
- [x] Production entry points implement the scoped deliverable: Run historical/current contract preservation, linters, race tests, coverage, fixture matrices, and unsupported-capability claim checks in CI
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
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:08c5d671a0c9e2350b1b8c97c08841f378392d0f4ac8edf05ecf155d29ae97bb rationale="implementation class rank 1; final Story leaf, the CI gate that decides what green means"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260906-fad351, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260906-fad351)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260906-fad351, pid=6352, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/xhigh pair_source=explicit match=not_recommended snapshot=sha256:9f20871155c37dee69e09bc65874033bce16ac39424aba056c24ea8521516f40 rationale="review class rank 1; story_final CR — the CI gate plus Story landability"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260906-7964d4, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260906-7964d4)
REVIEW rev1 (story_final): changes_requested -> to-dev. repeat-of: none.

Evidence: TASK-260830-2uowwk_review-verdict-rev1.md (+ gate-defeat probe logs). Reviewed tree 67eea42b, base 157a54c. Worktree verified byte-identical to the candidate tree after every probe.

No gate in the new code admits what it must reject — every G-A/G-B/G-C/G-D/G-E probe held (empty selector, empty fuzz derivation, -fuzz no-match, needs completeness 9/9, all timeouts set, one if: always(), jq aggregator rejects skipped/failure/cancelled, historical v0.4.3 contract alteration reddens, stale catalog caught, planted backticked README claim rejected, origin/main == CR base, both prior leaf commits signed, both frozen leaves byte-frozen). Suites re-run: test/race/cover all exit 0, 22 pkgs, cigate 90.5%.

BLOCKING — five advertised behaviours do not reproduce, on a Story whose AC row is no unsupported capability is advertised. Fixes are localized and touch no gate logic.

F1 ci.yml:293-298 step Record live probe verdicts runs -run TestSkipReportReadsRecordedVerdicts, which records only synthetic verdicts and restores them in Cleanup. No real Require executes, so the grepped line is skipped=0 failed=0 by construction: I got the identical line from the full suite AND from a run that executed zero tests (-run=^$). The step is also non-gating (grep|tail always exits 0). secconftest Report() doc comment already forbids this reading. The claim is carried into README (README scanned against live probe verdicts) and AC row 11 (live secconftest probes / probe-counter line). Neither reproduces: ProbeState.Available is discarded by every consumer — ProbeStates() is called only by CapabilityIDs(), and every CheckAdvertisements call site passes a forced state. The all-unavailable design is STRONGER than live scanning; only the description and the fake evidence step are wrong.

F2 TASK-260830-2uowwk_ci-gates.md names Candidate tree OID 7be8309b, which is exactly HEAD^{tree}: zero of the 10 changed files, no internal/cigate at all (git write-tree omits it because cigate is untracked). cigate 90.5% is unmeasurable there. Record tree is 67eea42b. Third occurrence of this class in this programme. Fix by resource update.

F3 cigate/targets.go:12 says CI iterates this derivation; ci.yml never references FuzzTargets (fuzz-smoke uses go test -list). No non-test caller exists. doc.go:17-18 repeats it.

UNSTATED BOUNDS — F4 CheckAdvertisements only matches backticked IDs: with every probe forced unavailable, both FIFO creation is available and supported on Windows and The fifo capability is available on Windows are ADMITTED. Stated only in the Finding struct comment; doc.go Stated bounds omits it and README says mentions. F5 fuzz-smoke covers only internal/secconftest; 5 real targets in canonicaljson+scalar are never smoked, and doc.go:17 asserts fuzz entry points stay in internal/secconftest, false repo-wide.

OBSERVATION (non-blocking) — verb-drop battery dies by bookkeeping: claims_test.go:57 Fatalfs on the table-size check, aborting the parent before any subtest. Censused all 5 verbs: 0 of 5 behavioural subtests fire under any drop; only available also trips TestFindingCarriesLine. All mutants still killed and all 6 subtests drive their verb in the green state; only the kill mechanism is thin.

NOT charged to this CR: the catalog-freshness step is defeated by //go:generate -> // go:generate (token preserved) — pre-existing at HEAD, and this leafs new VerifyContractPreservation strengthens the contract-version dimension against it.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260906-7964d4, pid=56584, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:08c5d671a0c9e2350b1b8c97c08841f378392d0f4ac8edf05ecf155d29ae97bb rationale="implementation class rank 1; round 2, gates hold but five advertised behaviours do not reproduce"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260906-3ee6d8, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260906-3ee6d8)
rev2 rework handed off: F1 fake probe step deleted (forced-unavailable posture documented), F2 outcome doc carries verified tree 8d8231ac (7 cigate entries, 10 files), F3 CI wired to FuzzTargets via TestListedTargetsMatchDerivation, F4 backtick bound stated+pinned, F5 fuzz-smoke covers 13 targets across 3 packages; battery 30 applied/28 killed/2 census survivors; full gates green (22 ok, race 22 ok, cover cigate 90.5%, vet both GOOS, gofmt, catalog, tracecheck); artifacts updated: ci-gates.md, mutants-rev2.log, verify-rev2.log
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260906-3ee6d8, pid=49440, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/xhigh pair_source=explicit match=not_recommended snapshot=sha256:9f20871155c37dee69e09bc65874033bce16ac39424aba056c24ea8521516f40 rationale="review class rank 1; story_final round 2 — claims corrected, judge the Story for landing"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260906-c68b0a, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260906-c68b0a)
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260906-c68b0a, pid=15675, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:08c5d671a0c9e2350b1b8c97c08841f378392d0f4ac8edf05ecf155d29ae97bb rationale="implementation class; integration run bound to the accepted rev2 producer binding"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260906-ce22ea, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260906-ce22ea)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260830-2uowwk_spawn-log_-implementer--developer--muse-_RUN-260906-fad351.log](file://TASK-260830-2uowwk/TASK-260830-2uowwk_spawn-log_-implementer--developer--muse-_RUN-260906-fad351.log) — System spawn log captured by task-board
- [TASK-260830-2uowwk_ci-gates.md](file://TASK-260830-2uowwk/TASK-260830-2uowwk_ci-gates.md) — M0 CI and release gates rev2: 11/11 AC rows, 30-mutant battery, per-job red proofs, verified candidate tree
- [TASK-260830-2uowwk_mutants.log](file://TASK-260830-2uowwk/TASK-260830-2uowwk_mutants.log) — Mutant battery results: 13 applied, 11 killed, 2 census survivors with bounds
- [TASK-260830-2uowwk_verify.log](file://TASK-260830-2uowwk/TASK-260830-2uowwk_verify.log) — Final verification: 22 pkgs ok, vet/vet-windows/build-windows/gofmt/catalog/tracecheck/pin all green
- [TASK-260830-2uowwk_change-request_rev1.patch](file://TASK-260830-2uowwk/TASK-260830-2uowwk_change-request_rev1.patch) — Change Request CR-TASK-260830-2uowwk-1 revision 1 candidate patch (repository_delta=present, 60 changed paths)
- [TASK-260830-2uowwk_change-request_rev1-validation.log](file://TASK-260830-2uowwk/TASK-260830-2uowwk_change-request_rev1-validation.log) — Change Request CR-TASK-260830-2uowwk-1 revision 1 bounded validation log
- [TASK-260830-2uowwk_spawn-log_-reviewer--reviewer--claude-_RUN-260906-7964d4.log](file://TASK-260830-2uowwk/TASK-260830-2uowwk_spawn-log_-reviewer--reviewer--claude-_RUN-260906-7964d4.log) — System spawn log captured by task-board
- [TASK-260830-2uowwk_review-verdict-rev1.md](file://TASK-260830-2uowwk/TASK-260830-2uowwk_review-verdict-rev1.md) — Reviewer verdict for CR rev1 (story_final): changes_requested. 3 blocking (live-probe evidence that records none; outcome doc names HEAD tree; FuzzTargets claims absent CI wiring), 2 unstated bounds, 1 observation. All G-A/G-B/G-C/G-D/G-E probes reproduced.
- [TASK-260830-2uowwk_review-rev1-probe-logs.log](file://TASK-260830-2uowwk/TASK-260830-2uowwk_review-rev1-probe-logs.log) — Reviewer probe log: cigate suite baseline on the candidate tree (20 PASS / 0 FAIL), the run all verb-census and gate-defeat probes were measured against.
- [TASK-260830-2uowwk_review-rev1-gate-defeat-probes.tgz](file://TASK-260830-2uowwk/TASK-260830-2uowwk_review-rev1-gate-defeat-probes.tgz) — Reviewer gate-defeat probe logs: planted README claim (backticked + unbackticked), historical-contract alteration, stale catalog, generator disarm, empty selectors, fuzz no-match, and the 5-verb narrowing census.
- [TASK-260830-2uowwk_spawn-log_-implementer--developer--muse-_RUN-260906-3ee6d8.log](file://TASK-260830-2uowwk/TASK-260830-2uowwk_spawn-log_-implementer--developer--muse-_RUN-260906-3ee6d8.log) — System spawn log captured by task-board
- [TASK-260830-2uowwk_mutants-rev2.log](file://TASK-260830-2uowwk/TASK-260830-2uowwk_mutants-rev2.log) — Rev2 mutant battery: 30 applied, 28 killed, 2 census survivors, all restored identical
- [TASK-260830-2uowwk_verify-rev2.log](file://TASK-260830-2uowwk/TASK-260830-2uowwk_verify-rev2.log) — Rev2 gate evidence: full suite, race, coverage, tracecheck tails
- [TASK-260830-2uowwk_change-request_rev2.patch](file://TASK-260830-2uowwk/TASK-260830-2uowwk_change-request_rev2.patch) — Change Request CR-TASK-260830-2uowwk-2 revision 2 candidate patch (repository_delta=present, 60 changed paths)
- [TASK-260830-2uowwk_change-request_rev2-validation.log](file://TASK-260830-2uowwk/TASK-260830-2uowwk_change-request_rev2-validation.log) — Change Request CR-TASK-260830-2uowwk-2 revision 2 bounded validation log
- [TASK-260830-2uowwk_spawn-log_-reviewer--reviewer--claude-_RUN-260906-c68b0a.log](file://TASK-260830-2uowwk/TASK-260830-2uowwk_spawn-log_-reviewer--reviewer--claude-_RUN-260906-c68b0a.log) — System spawn log captured by task-board
- [TASK-260830-2uowwk_review-verdict-rev2.md](file://TASK-260830-2uowwk/TASK-260830-2uowwk_review-verdict-rev2.md) — Reviewer verdict for CR rev2 (story_final): accepted. G-A..G-F all resolved; 29/30 producer mutants + 7 reviewer mutants re-run; 6 non-blocking findings (R1-R6) itemised
- [TASK-260830-2uowwk_review-rev2-probe-logs.tgz](file://TASK-260830-2uowwk/TASK-260830-2uowwk_review-rev2-probe-logs.tgz) — Reviewer rev2 probe logs: full suite/race/cover, lint+build+tracecheck, the three name-guarded CI gate steps verbatim, both darwin fixture-matrix legs, and the 13-target fuzz derive+smoke run
- [TASK-260830-2uowwk_spawn-log_-implementer--developer--muse-_RUN-260906-ce22ea.log](file://TASK-260830-2uowwk/TASK-260830-2uowwk_spawn-log_-implementer--developer--muse-_RUN-260906-ce22ea.log) — System spawn log captured by task-board

## Created
2026-08-29T22:00:09Z

## Last Update
2026-09-06T20:04:53Z

## Assigned To
[implementer] developer (muse)
