## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(8))

## Blocked By
- TASK-260830-1geqhj

## Blocks
- TASK-260830-2056mm

## Checklist
- [x] Production entry points implement the scoped deliverable: Implement absent/creating/parked/active/quiescing/stopped/stale/unavailable transitions and generation-safe evidence
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
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:009eecf230ad676c71fa02f210ad171dc6936cae601b3bc028fa753d219ff3c1 rationale="Second leaf of STORY-260830-ptxkqe: the Section 4.C Terminal Instance lifecycle contract (AXAuthorization closed object, the RetryDisposition enum, the state-entry rules, generation-safe evidence) over the landed internal/terminalbackend; implementation workload, muse-spark max producer with an independent claude-opus-5 max review."
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (muse) (run=RUN-260918-331b74, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260918-331b74)
agent completed: [implementer] developer (muse) (exit=1)
spawn run completed: muse (run=RUN-260918-331b74, pid=73240, exit=1)
spawn autonomous recovery: run RUN-260918-331b74 queued successor RUN-260918-6da9d0 (attempt 1/3, model=muse-spark): spawned agent exited with code 1
spawn run started: [implementer] developer (muse) (run=RUN-260918-6da9d0)
agent completed: [implementer] developer (muse) (exit=1)
spawn run completed: muse (run=RUN-260918-6da9d0, pid=83652, exit=1)
spawn autonomous recovery: run RUN-260918-6da9d0 queued successor RUN-260918-1e1204 (attempt 2/3, model=muse-spark): spawned agent exited with code 1
spawn run started: [implementer] developer (muse) (run=RUN-260918-1e1204)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260918-1e1204, pid=23978, exit=0)
spawn autonomous recovery: run RUN-260918-1e1204 queued successor RUN-260918-ca7917 (attempt 3/3, model=muse-spark): Change Request construction for TASK-260830-kkh1an failed: Change Request CR-TASK-260830-kkh1an-1 revision 1 validation failed at command 4/30 (1-based) with exit code 1; log resource TASK-260830-kkh1an_change-request_rev1-validation.log; retry: fix the failure and complete the producer again; the configured suite will rerun automatically
spawn run started: [implementer] developer (muse) (run=RUN-260918-ca7917)
agent completed: [implementer] developer (muse) (exit=143)
spawn run RUN-260918-ca7917 cancelled by operator; operator action required; reason: no operator reason supplied
spawn run completed: muse (run=RUN-260918-ca7917, pid=82848, exit=143)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:009eecf230ad676c71fa02f210ad171dc6936cae601b3bc028fa753d219ff3c1 rationale="Respawn after the rev1 construction failed at command 4/30 on the trunk resumesmoke time bomb (fixed as a12d1bd), not on the candidate: the brief now tells it to refresh onto c3aae73 first and to keep the work already in the worktree; Section 4.C lifecycle scope unchanged."
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (muse) (run=RUN-260918-2b5d1b, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260918-2b5d1b)
refresh_advanced onto c3aae73 with LOGBOOK replay; stale trunk-changed files restored to HEAD; candidate paths rebased additive; config byte-identical. Inherited rev1 delta verified against SPEC.v0.7.0 + landed owners; fix list open.
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260918-2b5d1b, pid=91328, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:dd9abb977b5b89fdf2151c43c20e944d7ed0ca7e20f61d52e87424a10564fe3c rationale="Independent review of the first-leaf CR2 of TASK-260830-kkh1an; operator routing: independent reviews on claude-opus-5 max (PR48) while producers stay on muse-spark max; codex Astra remains the fallback ceiling."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [reviewer] reviewer (claude) (run=RUN-260918-4f5cc1, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260918-4f5cc1)
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260918-4f5cc1, pid=26869, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:009eecf230ad676c71fa02f210ad171dc6936cae601b3bc028fa753d219ff3c1 rationale="Rework of CR2 after the claude-opus-5 max review: no P1; two P2 production findings (a receipt keeps its idempotency key forever so the table's status-then-same-key-retry recovery is unimplementable, and ObserveFencing never fences a stale incarnation whose winner is on another host) plus two P2 evidence gaps and seven P3."
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (muse) (run=RUN-260918-cf3532, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260918-cf3532)
agent completed: [implementer] developer (muse) (exit=1)
spawn run completed: muse (run=RUN-260918-cf3532, pid=63502, exit=1)
spawn autonomous recovery: run RUN-260918-cf3532 queued successor RUN-260918-cf03db (attempt 1/3, model=muse-spark): spawned agent exited with code 1
spawn run started: [implementer] developer (muse) (run=RUN-260918-cf03db)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260918-cf03db, pid=72262, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:dd9abb977b5b89fdf2151c43c20e944d7ed0ca7e20f61d52e87424a10564fe3c rationale="Independent review of the first-leaf CR3 of TASK-260830-kkh1an; operator routing: independent reviews on claude-opus-5 max (PR48) while producers stay on muse-spark max; codex Astra remains the fallback ceiling."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [reviewer] reviewer (claude) (run=RUN-260918-f6b44c, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260918-f6b44c)
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260918-f6b44c, pid=32695, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:009eecf230ad676c71fa02f210ad171dc6936cae601b3bc028fa753d219ff3c1 rationale="Rework of CR3: no P1 and the reviewer confirms F1, F2, E1, E2, the 100/100 harness, the 84-of-84 ratio and a green suite; two findings remain and BOTH are marked repeat-of the previous round's classes — the grant-less members of the stale-incarnation class (needs an exported staleness verdict in internal/fencing, sanctioned by the reviewer) and the expiry axis of the per-effect recheck."
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (muse) (run=RUN-260918-9edf2c, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260918-9edf2c)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260918-9edf2c, pid=64487, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:dd9abb977b5b89fdf2151c43c20e944d7ed0ca7e20f61d52e87424a10564fe3c rationale="Independent review of the first-leaf CR4 of TASK-260830-kkh1an; operator routing: independent reviews on claude-opus-5 max (PR48) while producers stay on muse-spark max; codex Astra remains the fallback ceiling."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [reviewer] reviewer (claude) (run=RUN-260918-889432, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260918-889432)
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260918-889432, pid=62675, exit=0)
spawn workload selection: class=operations source=explicit policy=spawn.workload_classes pair=muse/muse-spark/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:58b0a9b9fa840607602e2f3176d3ba574704cd38bb765f43b6dbbd06f7d2a566 rationale="Producer-bound checkpoint-only run for the accepted non-final CR4 of TASK-260830-kkh1an; muse-spark max producer-role run."
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (muse) (run=RUN-260918-56e104, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260918-56e104)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260918-56e104, pid=48564, exit=0)

## Precondition Resources
- [TASK-260830-kkh1an_producer.md](file://TASK-260830-kkh1an/TASK-260830-kkh1an_producer.md)
- [TASK-260830-kkh1an_rework-rev4.md](file://TASK-260830-kkh1an/TASK-260830-kkh1an_rework-rev4.md)
- [TASK-260830-kkh1an_reviewer-cr4.md](file://TASK-260830-kkh1an/TASK-260830-kkh1an_reviewer-cr4.md)
- [TASK-260830-kkh1an_checkpoint-brief-rev4.md](file://TASK-260830-kkh1an/TASK-260830-kkh1an_checkpoint-brief-rev4.md)

## Outcome Resources
- [TASK-260830-kkh1an_spawn-log_-implementer--developer--muse-_RUN-260918-331b74.log](file://TASK-260830-kkh1an/TASK-260830-kkh1an_spawn-log_-implementer--developer--muse-_RUN-260918-331b74.log) — System spawn log captured by task-board
- [TASK-260830-kkh1an_spawn-log_-implementer--developer--muse-_RUN-260918-6da9d0.log](file://TASK-260830-kkh1an/TASK-260830-kkh1an_spawn-log_-implementer--developer--muse-_RUN-260918-6da9d0.log) — System spawn log captured by task-board
- [TASK-260830-kkh1an_spawn-log_-implementer--developer--muse-_RUN-260918-1e1204.log](file://TASK-260830-kkh1an/TASK-260830-kkh1an_spawn-log_-implementer--developer--muse-_RUN-260918-1e1204.log) — System spawn log captured by task-board
- [TASK-260830-kkh1an_results.md](file://TASK-260830-kkh1an/TASK-260830-kkh1an_results.md) — Handoff evidence (rev4)
- [TASK-260830-kkh1an_conformance-matrix.md](file://TASK-260830-kkh1an/TASK-260830-kkh1an_conformance-matrix.md) — Clause-to-test conformance matrix (rev4)
- [TASK-260830-kkh1an_producer-evidence.tar.gz](file://TASK-260830-kkh1an/TASK-260830-kkh1an_producer-evidence.tar.gz) — Validation logs and mutation artifacts (rev4)
- [TASK-260830-kkh1an_change-request_rev1.patch](file://TASK-260830-kkh1an/TASK-260830-kkh1an_change-request_rev1.patch) — Change Request CR-TASK-260830-kkh1an-1 revision 1 candidate patch (repository_delta=present, 31 changed paths)
- [TASK-260830-kkh1an_change-request_rev1-validation.log](file://TASK-260830-kkh1an/TASK-260830-kkh1an_change-request_rev1-validation.log) — Change Request CR-TASK-260830-kkh1an-1 revision 1 bounded validation log
- [TASK-260830-kkh1an_spawn-log_-implementer--developer--muse-_RUN-260918-ca7917.log](file://TASK-260830-kkh1an/TASK-260830-kkh1an_spawn-log_-implementer--developer--muse-_RUN-260918-ca7917.log) — System spawn log captured by task-board
- [TASK-260830-kkh1an_spawn-log_-implementer--developer--muse-_RUN-260918-2b5d1b.log](file://TASK-260830-kkh1an/TASK-260830-kkh1an_spawn-log_-implementer--developer--muse-_RUN-260918-2b5d1b.log) — System spawn log captured by task-board
- [TASK-260830-kkh1an_change-request_rev2.patch](file://TASK-260830-kkh1an/TASK-260830-kkh1an_change-request_rev2.patch) — Change Request CR-TASK-260830-kkh1an-2 revision 2 candidate patch (repository_delta=present, 31 changed paths)
- [TASK-260830-kkh1an_change-request_rev2-validation.log](file://TASK-260830-kkh1an/TASK-260830-kkh1an_change-request_rev2-validation.log) — Change Request CR-TASK-260830-kkh1an-2 revision 2 bounded validation log
- [TASK-260830-kkh1an_spawn-log_-reviewer--reviewer--claude-_RUN-260918-4f5cc1.log](file://TASK-260830-kkh1an/TASK-260830-kkh1an_spawn-log_-reviewer--reviewer--claude-_RUN-260918-4f5cc1.log) — System spawn log captured by task-board
- [TASK-260830-kkh1an_review-verdict-rev2.md](file://TASK-260830-kkh1an/TASK-260830-kkh1an_review-verdict-rev2.md) — Independent review verdict for CR revision 2: CHANGES REQUESTED (no P1; P2 F1 poisoned idempotency key / F2 remote-winner fencing; P2 evidence E1/E2; 7 P3)
- [TASK-260830-kkh1an_review-evidence-rev2.tar.gz](file://TASK-260830-kkh1an/TASK-260830-kkh1an_review-evidence-rev2.tar.gz) — Review evidence rev2: probes, witnesses, reviewer mutants x2 with raw logs, shipped harness rerun x2, real-kill, tree-exact checks, manifest
- [TASK-260830-kkh1an_spawn-log_-implementer--developer--muse-_RUN-260918-cf3532.log](file://TASK-260830-kkh1an/TASK-260830-kkh1an_spawn-log_-implementer--developer--muse-_RUN-260918-cf3532.log) — System spawn log captured by task-board
- [TASK-260830-kkh1an_spawn-log_-implementer--developer--muse-_RUN-260918-cf03db.log](file://TASK-260830-kkh1an/TASK-260830-kkh1an_spawn-log_-implementer--developer--muse-_RUN-260918-cf03db.log) — System spawn log captured by task-board
- [TASK-260830-kkh1an_change-request_rev3.patch](file://TASK-260830-kkh1an/TASK-260830-kkh1an_change-request_rev3.patch) — Change Request CR-TASK-260830-kkh1an-3 revision 3 candidate patch (repository_delta=present, 39 changed paths)
- [TASK-260830-kkh1an_change-request_rev3-validation.log](file://TASK-260830-kkh1an/TASK-260830-kkh1an_change-request_rev3-validation.log) — Change Request CR-TASK-260830-kkh1an-3 revision 3 bounded validation log
- [TASK-260830-kkh1an_spawn-log_-reviewer--reviewer--claude-_RUN-260918-f6b44c.log](file://TASK-260830-kkh1an/TASK-260830-kkh1an_spawn-log_-reviewer--reviewer--claude-_RUN-260918-f6b44c.log) — System spawn log captured by task-board
- [TASK-260830-kkh1an_review-verdict-rev3.md](file://TASK-260830-kkh1an/TASK-260830-kkh1an_review-verdict-rev3.md) — Independent review verdict for CR revision 3: CHANGES REQUESTED (no P1; P2 F3 grant-less stale members never fenced [repeat-of rev2 F2 class]; P2 E3 per-effect auth expiry axis unmeasured [repeat-of rev2 E2 class]; 5 P3; F1/F2/E1/E2 rework confirmed closed; harness 100/100 x2; 84/84 ratio; full suite green)
- [TASK-260830-kkh1an_review-evidence-rev3.tar.gz](file://TASK-260830-kkh1an/TASK-260830-kkh1an_review-evidence-rev3.tar.gz) — Review evidence rev3: 16 probes + 2 real-kill probes, 7 reviewer mutants + 3 controls x4 with raw logs, shipped harness rerun x2 with 100 raw logs per pass, go test ./... 40/40, race/cover/tracecheck/cataloggen, hygiene, ratio; sha256 manifest
- [TASK-260830-kkh1an_spawn-log_-implementer--developer--muse-_RUN-260918-9edf2c.log](file://TASK-260830-kkh1an/TASK-260830-kkh1an_spawn-log_-implementer--developer--muse-_RUN-260918-9edf2c.log) — System spawn log captured by task-board
- [TASK-260830-kkh1an_change-request_rev4.patch](file://TASK-260830-kkh1an/TASK-260830-kkh1an_change-request_rev4.patch) — Change Request CR-TASK-260830-kkh1an-4 revision 4 candidate patch (repository_delta=present, 44 changed paths)
- [TASK-260830-kkh1an_change-request_rev4-validation.log](file://TASK-260830-kkh1an/TASK-260830-kkh1an_change-request_rev4-validation.log) — Change Request CR-TASK-260830-kkh1an-4 revision 4 bounded validation log
- [TASK-260830-kkh1an_spawn-log_-reviewer--reviewer--claude-_RUN-260918-889432.log](file://TASK-260830-kkh1an/TASK-260830-kkh1an_spawn-log_-reviewer--reviewer--claude-_RUN-260918-889432.log) — System spawn log captured by task-board
- [TASK-260830-kkh1an_review-verdict-rev4.md](file://TASK-260830-kkh1an/TASK-260830-kkh1an_review-verdict-rev4.md) — Independent review verdict for CR revision 4: ACCEPTED (no P1, no P2; four P3 evidence residuals with committable witnesses: third-position expiry recheck, replay-seam CheckResult composition, in-loop deadline instant, mixed-case enum siblings). F3 and E3 verified as classes with a 31,104-cell oracle census, a 6,912-cell gate-vs-verdict differential, 7 reviewer narrowings x2, shipped harness 116/116 x2, real kills x2 incl. a new lost-result seam, ratio 93/93, full suite green.
- [TASK-260830-kkh1an_review-evidence-rev4.tar.gz](file://TASK-260830-kkh1an/TASK-260830-kkh1an_review-evidence-rev4.tar.gz) — Review evidence rev4: 11 probes + 1 real-kill probe, 7 reviewer narrowings + 3 controls x2 with raw per-plant logs and diffs, shipped harness rerun x2 (116 raw logs each), producer/rev3 real kills x2, race/cover/tracecheck/cataloggen/hygiene/ratio logs, REVIEW-MANIFEST.txt (PYTHONDONTWRITEBYTECODE=1, no __pycache__).
- [TASK-260830-kkh1an_spawn-log_-implementer--developer--muse-_RUN-260918-56e104.log](file://TASK-260830-kkh1an/TASK-260830-kkh1an_spawn-log_-implementer--developer--muse-_RUN-260918-56e104.log) — System spawn log captured by task-board
- [TASK-260830-kkh1an_checkpoint-rev4.md](file://TASK-260830-kkh1an/TASK-260830-kkh1an_checkpoint-rev4.md) — Checkpoint outcome: accepted CR rev4 checkpointed as ca1c1d9, task integrating

## Created
2026-08-29T22:00:23Z

## Last Update
2026-09-18T11:49:05Z

## Assigned To
[implementer] developer (muse)
