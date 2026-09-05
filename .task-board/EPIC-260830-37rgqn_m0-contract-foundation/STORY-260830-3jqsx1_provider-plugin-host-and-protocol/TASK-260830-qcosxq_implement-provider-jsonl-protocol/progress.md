## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(8))

## Blocked By
- TASK-260830-2890sd

## Blocks
- TASK-260830-32jeti

## Checklist
- [x] Production entry points implement the scoped deliverable: Implement one-frame JSONL transport, deadlines, limits, stdout/stderr separation, operation dispatch, and status recovery
- [x] Relevant positive, negative, compatibility, and recovery tests pass with logs attached
- [x] README/doctor/capability evidence and specification traceability are updated without unsupported claims
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
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:cd50227265ca35d490f29a93c82a485be7dd82a62d95840afdbaf40590e6093e rationale="implementation class rank 1; leaf 2 protocol work on a checkpointed leaf 1"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260904-f92ab5, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260904-f92ab5)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260904-f92ab5, pid=65613, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:ea5498eaa14293ca9063a299009604c3a4d14fb3b285e8a8d0685edcf4bf1cf7 rationale="review class rank 1; new untrusted-input parsing surface plus three inherited closers"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260905-464cea, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260905-464cea)
Round 1 review: CHANGES REQUESTED. Evidence: TASK-260830-qcosxq_review-verdict.md.

Battery: 44/66 mutants killed over a traversal derived from production source (not from the test files). 22 survived, 4 equivalent, so 18 gates admit what they must reject with the suite green. Leaf 1 ended at 80/83. Tree verified byte-identical to candidate 2c7022b before and after.

Inherited findings all genuinely closed, not just present. F18: a planted correctly-typed production init() reassigning ownerAttester turns TestOwnerAttesterIsUnreassigned red, so the pin closes round 4s exact mutant. F19: planted return out,err at ALL FIVE refusal returns in Discover (brief named two) - 5/5 red. F20: a planted Windows-only type error passes go build, go vet and the whole suite, and is caught only by the new GOOS=windows go vet step.

Zero resurrections: 13 leaf-1 mutants re-run, 13 killed.

BLOCKING:
F21 parseMajor - all four rejection arms deletable green, three of them flip a malformed protocol_version from provider_protocol_error (exit 13) to incompatible_protocol (exit 6). Every existing fixture has major-part 2 or a non-numeric first char, so the misclassification collapses to the same code. Needs foreign-major malformed fixtures.
F22 rollback token - entropy floor weakenable from 256 bits to 64 bits with the suite green. Provable only in (4,38] bytes: short fixture decodes to 3, valid to 38, nothing pins the 32-byte floor. Security-relevant.
F23 readCapped - removing the LimitReader entirely, and removing the truncation, both survive. The only readCapped test drives the error path. runner.go:145 and doc.go claim memory stays bounded; the claim does not reproduce.

ALSO THIS ROUND:
F24 request-side 8 MiB bound halvable to 4 MiB green, while the response side kills an exact +1 mutant. Same limit, one direction exact, the other not.
F26 discarding exit.ExitCode() in ExecRunner survives; driven through a real exit-3 plugin it flips provider_process_failed to provider_protocol_error. Crash classification is proven only through the scripted fake.
F28 README: the new section was spliced above a paragraph belonging to the provider section, so README now says six constructors/six codes and four constructors/three codes about provhost in adjacent paragraphs.
F25 structural ask: 0 of 23 failureCode assertions name the refusal arm. 18 provider_protocol_error sites, 18 integrity_failure arms. Deleting the whole required-member list in checkResponseMembers AND in DecodeStatusOutcome both survive by sliding to a lower arm with the same code. The inventory audit is strong (killed M24/M28/R04 alone) but audits constructor call sites, so it is blind to arm-slide by construction.

NON-BLOCKING: F27 outbound line terminator never asserted (impact unproven - POSIX read tolerates it). F29 README claim each gate is proven by narrowing mutants does not reproduce.

Producer evidence re-verified and accurate: go test ./... 15/15, -race clean, gofmt clean, both vets, tracecheck exit 0 at 17/403 and 74 cases. Coverage 86.3% vs 86.1% claimed - trivial.

BOUNDS: provhost does NOT import internal/provider (only a comment at runner.go:44), so leaf 1s no-production-caller bound is UNCHANGED, contrary to the briefs expectation. provhost has no production caller either and there is no cmd/ax, so the bound widens to two orphaned packages. Windows still compile-verified only, now actually enforced in CI. Inspect->ReadFile TOCTOU still unprobed. New bound: doc.go non-retryability claim has zero assertions.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260905-464cea, pid=32058, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:cd50227265ca35d490f29a93c82a485be7dd82a62d95840afdbaf40590e6093e rationale="implementation class rank 1; rework of one structural arm-blindness plus three blocking gates"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260905-1d9be4, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260905-1d9be4)
Round-2 rework handed off uncommitted: F21/F22/F23/F24/F25/F26/F27/F28/F29 closed; 18 killed + 2 declared-equivalent (M09b, single-site R17) with proofs; all gates exit 0; evidence attached.
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260905-1d9be4, pid=95280, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:ea5498eaa14293ca9063a299009604c3a4d14fb3b285e8a8d0685edcf4bf1cf7 rationale="review class rank 1; round 2 must judge whether branch enumeration generalises"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260905-55bda6, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260905-55bda6)
Round 2 review: CHANGES REQUESTED. Evidence: TASK-260830-qcosxq_review-verdict.md (batteries: _round2-review-battery-b1..b4.json, logs: _round2-review-mutant-logs.tgz).

Battery: 89/106 killed on a traversal re-derived from production this round, 6 confirmed equivalent, 11 real survivors. Tree verified byte-identical to candidate 81f7c566 before and after every mutant; restores by cp, never git checkout.

Round 1 fully closed: all 18 named real survivors disposed of - 16 killed (M19-M22, token <8/<16/<48 plus my <31/<33, R09, R07, M06c plus widen-2x and exact >=, M09/M09a/S02/S02a/M27, R17 both-sites, R14), 2 producer-declared equivalent and independently confirmed (M09b, R17 single-site). F28 README paragraph moved back; F29 restated to what reproduces. Zero resurrections: 23/23 round-1 provhost killer tests still present, and F18/F19/F20 re-probed from scratch (planted production init reassigning ownerAttester -> red; return out,err at all five Discover refusal returns -> red; Windows-only type error passes build/vet/test and is caught only by GOOS=windows go vet).

G-A ANSWER: the arm enumeration is STILL MANUAL. inventory_test.go is byte-identical between rev1 and rev2; the round added named rows, not a mechanism. Measured: a new failProtocol/failInvalid CALL SITE is killed by the audit (G2, G4), a new frameFault or integrity() ARM through an existing site SURVIVES (G1, G3). Arm coverage 51/52 by assertion.

BLOCKING F30: parseMajor has FIVE rejection branches; round 1 named four and the producer closed exactly those four. The fifth - the non-digit MAJOR check at protocol.go:346-352 - is still deletable with the suite green and flips a.0.0 / 2a.0.0 / -1.0.0 / +3.0.0 from provider_protocol_error exit 13 to incompatible_protocol exit 6. Same class, same consequence as F21.

BLOCKING F31: stdoutCap (runner.go:20, used only at runner.go:85) is unpinned in both directions. Narrowing MaxFrameBytes+2 to +1 makes production ExecRunner+Host.Call ACCEPT a maximal frame followed by a terminator and one junk byte (err=nil, body 8388472 bytes) where the baseline refuses stdout carries more than one frame. The +2 probe byte is load-bearing exactly as the comment claims, and no test reaches the constant - both extra-line tests drive scriptRunner, which bypasses readCapped.

Same round, small: X19 (integrity status operation_id is not a string is the one unasserted arm), AS6 (status_state asserted on exactly one row; fold it into requireLocalRefusal), and correct the runner_test.go:484-487 comment - removing the process-group Cancel still returns in 300ms, so the timing does not prove group termination; whether the group kill buys anything beyond Setpgid/Kill(-pid) consistency is unknown on macOS from my probes. S11/S12/S13/S4b acceptable as stated bounds if written down.

Gates re-run by me: go test ./... exit 0 (15/15), -race provhost exit 0, go vet and GOOS=windows go vet exit 0, go build host/linux/windows exit 0, gofmt clean, tracecheck exit 0 unchanged 17/403 with 74 cases, provhost coverage 88.1% exact.

Bounds unchanged: provhost does not import internal/provider (comment-only at runner.go:44); provhost has no importer anywhere and there is no cmd/. Two orphaned packages, every Host.Call assertion made from a test.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260905-55bda6, pid=57891, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:cd50227265ca35d490f29a93c82a485be7dd82a62d95840afdbaf40590e6093e rationale="implementation class rank 1; round 3 must build a derived arm inventory, not add another named row"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260905-1226b1, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260905-1226b1)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260905-1226b1, pid=45040, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:ea5498eaa14293ca9063a299009604c3a4d14fb3b285e8a8d0685edcf4bf1cf7 rationale="review class rank 1; round 3 must attack a now load-bearing derived inventory"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260905-f6e0ac, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260905-f6e0ac)
Round-3 review ACCEPTED (RUN-260905-f6e0ac). Arm ratio 60/60, denominator counted independently by grep (10 frame + 17 integrity literals + 3 wrapper expansions + 5 parse branches + 25 ctor literals = 60). G-A: 6 of 8 novel attack shapes RED (helper-built frameFault, switch-shared variable, unvisited file, new-file arm, aliased existing site, hoisted literal); 2 blind spots measured and stated as bounds - GA11 (a one-line wrapper fronting a refusal constructor hides a whole new arm family from both the arm walk and the site audit) and GA9 (alias-minted arm at an unexercised site). Census floor settled: lowering it alone survives, but with the floor forced to 1 every truncation shape still reddens on orphaned witnesses, so the ratio is not compared against itself. G-B: 112 re-runs, 102 killed, 10 survivors (6 previously-confirmed equivalents + S11/S12/S13/U2, all round-2 stated bounds), zero resurrections, zero new survivors; F30, F31, S4b, X19 and AS6 all closed. Leaf 1 re-attacked: F18/F19 and the trust gates still red, and this CR strengthens both provider tests. G-C: the macOS group-kill unknown is preserved honestly - U2 still survives, the comment now states what the timing measures, and no test implies detached-descendant knowledge. Bounds unchanged: provhost does not import internal/provider, has no importer, and there is no top-level cmd/. Gates re-run: build/vet/GOOS=windows vet/gofmt/15-package test/race all exit 0, coverage 88.4%, tracecheck 17/403 with 74 acceptance cases. Non-blocking follow-ups for the next toucher: the arm-inventory header comment claims non-literal shapes are never silent, which is false for refusal-constructor calls (its own inline comment says so); and round 3 added no LOGBOOK entry.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260905-f6e0ac, pid=69659, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260830-qcosxq_spawn-log_-implementer--developer--muse-_RUN-260904-f92ab5.log](file://TASK-260830-qcosxq/TASK-260830-qcosxq_spawn-log_-implementer--developer--muse-_RUN-260904-f92ab5.log) — System spawn log captured by task-board
- [TASK-260830-qcosxq_outcome.md](file://TASK-260830-qcosxq/TASK-260830-qcosxq_outcome.md) — Implementation outcome, evidence, bounds, and file list
- [TASK-260830-qcosxq_tests.log](file://TASK-260830-qcosxq/TASK-260830-qcosxq_tests.log) — Verbose provhost+provider tests: 232 PASS, exit 0
- [TASK-260830-qcosxq_mutants.log](file://TASK-260830-qcosxq/TASK-260830-qcosxq_mutants.log) — Narrowing mutant sweep: 25 killed, 0 survivors
- [TASK-260830-qcosxq_change-request_rev1.patch](file://TASK-260830-qcosxq/TASK-260830-qcosxq_change-request_rev1.patch) — Change Request CR-TASK-260830-qcosxq-1 revision 1 candidate patch (repository_delta=present, 15 changed paths)
- [TASK-260830-qcosxq_change-request_rev1-validation.log](file://TASK-260830-qcosxq/TASK-260830-qcosxq_change-request_rev1-validation.log) — Change Request CR-TASK-260830-qcosxq-1 revision 1 bounded validation log
- [TASK-260830-qcosxq_spawn-log_-reviewer--reviewer--claude-_RUN-260905-464cea.log](file://TASK-260830-qcosxq/TASK-260830-qcosxq_spawn-log_-reviewer--reviewer--claude-_RUN-260905-464cea.log) — System spawn log captured by task-board
- [TASK-260830-qcosxq_review-verdict.md](file://TASK-260830-qcosxq/TASK-260830-qcosxq_review-verdict.md) — Round-2 review verdict: changes requested. 89/106 mutants killed, 6 confirmed equivalent, 11 real survivors; all 18 round-1 survivors closed, zero resurrections. Blocking F30 (fifth parseMajor branch, exit 13->6) and F31 (stdoutCap probe byte unpinned, accepts frame+junk). G-A: arm enumeration still manual.
- [TASK-260830-qcosxq_spawn-log_-implementer--developer--muse-_RUN-260905-1d9be4.log](file://TASK-260830-qcosxq/TASK-260830-qcosxq_spawn-log_-implementer--developer--muse-_RUN-260905-1d9be4.log) — System spawn log captured by task-board
- [TASK-260830-qcosxq_round2-rework-evidence.md](file://TASK-260830-qcosxq/TASK-260830-qcosxq_round2-rework-evidence.md) — Round-2 rework evidence: finding closures, mutant sweep, gates
- [TASK-260830-qcosxq_round2-verify-sweep.sh](file://TASK-260830-qcosxq/TASK-260830-qcosxq_round2-verify-sweep.sh) — Re-runnable 19-mutant verification sweep (cp-aside restore, presence checks)
- [TASK-260830-qcosxq_change-request_rev2.patch](file://TASK-260830-qcosxq/TASK-260830-qcosxq_change-request_rev2.patch) — Change Request CR-TASK-260830-qcosxq-2 revision 2 candidate patch (repository_delta=present, 15 changed paths)
- [TASK-260830-qcosxq_change-request_rev2-validation.log](file://TASK-260830-qcosxq/TASK-260830-qcosxq_change-request_rev2-validation.log) — Change Request CR-TASK-260830-qcosxq-2 revision 2 bounded validation log
- [TASK-260830-qcosxq_spawn-log_-reviewer--reviewer--claude-_RUN-260905-55bda6.log](file://TASK-260830-qcosxq/TASK-260830-qcosxq_spawn-log_-reviewer--reviewer--claude-_RUN-260905-55bda6.log) — System spawn log captured by task-board
- [TASK-260830-qcosxq_round2-review-battery-b1.json](file://TASK-260830-qcosxq/TASK-260830-qcosxq_round2-review-battery-b1.json) — Round-2 review mutation battery, rows 1-26 (round-1 survivor re-runs and bounds)
- [TASK-260830-qcosxq_round2-review-battery-b2.json](file://TASK-260830-qcosxq/TASK-260830-qcosxq_round2-review-battery-b2.json) — Round-2 review mutation battery, batch 2
- [TASK-260830-qcosxq_round2-review-battery-b3.json](file://TASK-260830-qcosxq/TASK-260830-qcosxq_round2-review-battery-b3.json) — Round-2 review mutation battery, batch 3
- [TASK-260830-qcosxq_round2-review-battery-b4.json](file://TASK-260830-qcosxq/TASK-260830-qcosxq_round2-review-battery-b4.json) — Round-2 review mutation battery, batch 4
- [TASK-260830-qcosxq_round2-review-mutant-logs.tgz](file://TASK-260830-qcosxq/TASK-260830-qcosxq_round2-review-mutant-logs.tgz) — Round-2 review: one full-package go test log per mutant (106 measured rows)
- [TASK-260830-qcosxq_spawn-log_-implementer--developer--muse-_RUN-260905-1226b1.log](file://TASK-260830-qcosxq/TASK-260830-qcosxq_spawn-log_-implementer--developer--muse-_RUN-260905-1226b1.log) — System spawn log captured by task-board
- [TASK-260830-qcosxq_round3-evidence.md](file://TASK-260830-qcosxq/TASK-260830-qcosxq_round3-evidence.md) — Round-3 rework evidence: derived arm inventory, F30/F31/X19/AS6 fixes, 9/9 mutant reds, gate log
- [TASK-260830-qcosxq_change-request_rev3.patch](file://TASK-260830-qcosxq/TASK-260830-qcosxq_change-request_rev3.patch) — Change Request CR-TASK-260830-qcosxq-3 revision 3 candidate patch (repository_delta=present, 16 changed paths)
- [TASK-260830-qcosxq_change-request_rev3-validation.log](file://TASK-260830-qcosxq/TASK-260830-qcosxq_change-request_rev3-validation.log) — Change Request CR-TASK-260830-qcosxq-3 revision 3 bounded validation log
- [TASK-260830-qcosxq_spawn-log_-reviewer--reviewer--claude-_RUN-260905-f6e0ac.log](file://TASK-260830-qcosxq/TASK-260830-qcosxq_spawn-log_-reviewer--reviewer--claude-_RUN-260905-f6e0ac.log) — System spawn log captured by task-board
- [TASK-260830-qcosxq_review-verdict-round3.md](file://TASK-260830-qcosxq/TASK-260830-qcosxq_review-verdict-round3.md) — Round-3 review verdict: ACCEPTED. 112 mutant re-runs (102 killed, 10 survivors, 0 resurrections, 0 new), 13 novel arm-inventory attack probes (6 RED, 2 blind spots measured), 4 floor-forced combinations, 8 leaf-1 probes. Arm ratio 60/60 with the denominator independently counted.
- [TASK-260830-qcosxq_round3-review-battery.json](file://TASK-260830-qcosxq/TASK-260830-qcosxq_round3-review-battery.json) — Round-3 review mutation battery: 112 rerun rows + 13 novel G-A probes + 4 floor-forced combinations + 8 leaf-1 probes + 4 S1 repeats, each with exit code, verdict, byte-restore check and failure tail
- [TASK-260830-qcosxq_round3-review-probes.tgz](file://TASK-260830-qcosxq/TASK-260830-qcosxq_round3-review-probes.tgz) — Round-3 review probe harness: exact-once plant / presence check / targeted run / cp restore / sha256 verify, plus every probe definition as literal old-new source pairs

## Created
2026-08-29T22:00:00Z

## Last Update
2026-09-05T03:02:41Z

## Assigned To
[reviewer] reviewer (claude)
