## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(8))

## Blocked By
- TASK-260830-3uzfyn

## Blocks
- TASK-260830-1geqhj
- TASK-260830-2zmg1x
- TASK-260830-3l8ny3
- TASK-260830-2yefhm
- TASK-260830-24z2b3
- TASK-260830-2wflxd
- TASK-260916-1ec708

## Checklist
- [x] Production entry points implement the scoped deliverable: Run provider-specific discover/read/resume/identity checks and refuse unsupported tuple claims
- [x] Relevant positive, negative, compatibility, and recovery tests pass with logs attached
- [x] README/doctor/capability evidence and specification traceability are updated without unsupported claims
- [x] Code written per task description and AC
- [x] Relevant tests written for new or changed behavior and passing
- [x] In a managed Story worktree the candidate is left UNCOMMITTED in the worktree for the handoff to snapshot — never commit on the Story branch. A producer commit moves the branch tip off the recorded checkpoint and the handoff refuses with change_request_candidate_committed_past_checkpoint; repair with `git reset --soft <checkpoint_oid>` before completing again.
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
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:6a48f712eb7f7a5d67774693da5c6178cdf2e81494bfdf694bf079dd907a4c40 rationale="Final leaf of the provider-identity Story after the 3uzfyn checkpoint (resume smoke framework over provhost/sessprofile, Story-close registry re-pin, story_final); muse-spark max while codex is out of quota."
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (muse) (run=RUN-260917-ee1362, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260917-ee1362)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260917-ee1362, pid=11107, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:1fce99bebc94ff339ceae6f5f5862c930c3b4330c7fea22014bcf6622dc1e20b rationale="Independent review of the story_final CR1 of TASK-260830-2zvo8m; codex gpt-6-astra reviewers are out of quota until 2026-09-19, muse-spark xhigh is the cheapest admitted muse pair for the reviewer role."
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex], config: spawn.preferred_agentic_system)
spawn queued: [reviewer] reviewer (muse) (run=RUN-260917-3ff440, max_parallel=20)
spawn run started: [reviewer] reviewer (muse) (run=RUN-260917-3ff440)
Reviewer rev1 verdict: CHANGES REQUESTED (to-dev). Product code fully verified green (suites, 10 shipped + 4 own narrowing mutants killed, derivation/doc/self-mint probes, crash replay, tracecheck 29/463). P1-1: restore trunk LOGBOOK entry TASK-260917-3lt2xv deleted from LOGBOOK.md. P1-2: restore README -adopted docs (regenerate block + toolchain row) deleted from README.md. Both hunks restorable from HEAD. See TASK-260830-2zvo8m_review-verdict-rev1.md.
agent completed: [reviewer] reviewer (muse) (exit=0)
spawn run completed: muse (run=RUN-260917-3ff440, pid=88124, exit=0)
spawn workload selection: class=mechanical source=explicit policy=spawn.workload_classes pair=muse/muse-spark/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:0216b9ada1fe64f88fc2fa2ca1482b7b56c798c3f01aba145ef4a461398991b2 rationale="Mechanical rework after CR1 changes requested: restore two trunk hunks the base refresh overwrote (LOGBOOK 3lt2xv entry, README -adopted docs), stale-copy audit, re-handoff; muse-spark max while codex is out of quota."
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (muse) (run=RUN-260917-ba1ec9, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260917-ba1ec9)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260917-ba1ec9, pid=43065, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=muse/muse-spark/xhigh pair_source=explicit match=not_recommended snapshot=sha256:1fce99bebc94ff339ceae6f5f5862c930c3b4330c7fea22014bcf6622dc1e20b rationale="Independent review of the story_final CR2 of TASK-260830-2zvo8m; codex gpt-6-astra reviewers are out of quota until 2026-09-19, muse-spark xhigh is the cheapest admitted muse pair for the reviewer role."
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex], config: spawn.preferred_agentic_system)
spawn queued: [reviewer] reviewer (muse) (run=RUN-260917-8f1e0e, max_parallel=20)
spawn run started: [reviewer] reviewer (muse) (run=RUN-260917-8f1e0e)
agent completed: [reviewer] reviewer (muse) (exit=0)
spawn run completed: muse (run=RUN-260917-8f1e0e, pid=16631, exit=0)
spawn workload selection: class=operations source=explicit policy=spawn.workload_classes pair=muse/muse-spark/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:e9e482910af5615a9e25e2116664cd6c852a64233514fde7ab2a0c16816eae50 rationale="Producer-bound integration run for the accepted story_final CR2 of TASK-260830-2zvo8m (worktree integrate, exact-head suite, delivery branch + PR; orchestrator lands); muse-spark max while codex is out of quota."
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (muse) (run=RUN-260917-373f86, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260917-373f86)

## Precondition Resources
- [TASK-260830-2zvo8m_producer.md](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_producer.md)
- [TASK-260830-2zvo8m_reviewer-cr1.md](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_reviewer-cr1.md)
- [TASK-260830-2zvo8m_rework-rev1.md](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_rework-rev1.md)
- [TASK-260830-2zvo8m_reviewer-cr2.md](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_reviewer-cr2.md)
- [TASK-260830-2zvo8m_integration-rev2.md](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_integration-rev2.md)

## Outcome Resources
- [TASK-260830-2zvo8m_spawn-log_-implementer--developer--muse-_RUN-260917-ee1362.log](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_spawn-log_-implementer--developer--muse-_RUN-260917-ee1362.log) — System spawn log captured by task-board
- [TASK-260830-2zvo8m_results.md](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_results.md) — Handoff evidence: AC coverage 16/16, mutants, validation
- [TASK-260830-2zvo8m_conformance-matrix.md](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_conformance-matrix.md) — Clause-to-test conformance matrix with stated bounds
- [TASK-260830-2zvo8m_producer-evidence.tar.gz](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_producer-evidence.tar.gz) — Producer evidence bundle: logs, sample record, mutant run
- [TASK-260830-2zvo8m_change-request_rev1.patch](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_change-request_rev1.patch) — Change Request CR-TASK-260830-2zvo8m-1 revision 1 candidate patch (repository_delta=present, 55 changed paths)
- [TASK-260830-2zvo8m_change-request_rev1-validation.log](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_change-request_rev1-validation.log) — Change Request CR-TASK-260830-2zvo8m-1 revision 1 bounded validation log
- [TASK-260830-2zvo8m_spawn-log_-reviewer--reviewer--muse-_RUN-260917-3ff440.log](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_spawn-log_-reviewer--reviewer--muse-_RUN-260917-3ff440.log) — System spawn log captured by task-board
- [TASK-260830-2zvo8m_review-verdict-rev1.md](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_review-verdict-rev1.md) — Reviewer verdict rev1: changes requested (P1 LOGBOOK/README regressions)
- [TASK-260830-2zvo8m_review-evidence-rev1.tar.gz](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_review-evidence-rev1.tar.gz) — Reviewer raw evidence rev1: suites, mutants, tracecheck, status
- [TASK-260830-2zvo8m_spawn-log_-implementer--developer--muse-_RUN-260917-ba1ec9.log](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_spawn-log_-implementer--developer--muse-_RUN-260917-ba1ec9.log) — System spawn log captured by task-board
- [TASK-260830-2zvo8m_results-rev2.md](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_results-rev2.md) — rev2 rework results: P1 restorations, audit, validation
- [TASK-260830-2zvo8m_conformance-matrix-rev2.md](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_conformance-matrix-rev2.md) — rev2 conformance matrix (unchanged content)
- [TASK-260830-2zvo8m_producer-evidence-rev2.tar.gz](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_producer-evidence-rev2.tar.gz) — rev2 producer evidence tarball
- [TASK-260830-2zvo8m_change-request_rev2.patch](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_change-request_rev2.patch) — Change Request CR-TASK-260830-2zvo8m-2 revision 2 candidate patch (repository_delta=present, 55 changed paths)
- [TASK-260830-2zvo8m_change-request_rev2-validation.log](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_change-request_rev2-validation.log) — Change Request CR-TASK-260830-2zvo8m-2 revision 2 bounded validation log
- [TASK-260830-2zvo8m_spawn-log_-reviewer--reviewer--muse-_RUN-260917-8f1e0e.log](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_spawn-log_-reviewer--reviewer--muse-_RUN-260917-8f1e0e.log) — System spawn log captured by task-board
- [TASK-260830-2zvo8m_review-verdict-rev2.md](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_review-verdict-rev2.md) — Reviewer acceptance verdict for CR rev2
- [TASK-260830-2zvo8m_review-evidence-rev2.tar.gz](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_review-evidence-rev2.tar.gz) — Reviewer independent verification evidence for CR rev2
- [TASK-260830-2zvo8m_spawn-log_-implementer--developer--muse-_RUN-260917-373f86.log](file://TASK-260830-2zvo8m/TASK-260830-2zvo8m_spawn-log_-implementer--developer--muse-_RUN-260917-373f86.log) — System spawn log captured by task-board

## Created
2026-08-29T22:00:21Z

## Last Update
2026-09-17T09:26:15Z

## Assigned To
[implementer] developer (muse)
