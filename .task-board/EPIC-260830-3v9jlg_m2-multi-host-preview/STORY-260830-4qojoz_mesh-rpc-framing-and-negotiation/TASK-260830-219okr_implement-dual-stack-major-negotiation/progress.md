## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(8))

## Blocked By
- TASK-260830-z1yxg9
- TASK-260830-2x16gz
- STORY-260908-18woqo

## Blocks
- TASK-260830-19bjfj

## Checklist
- [x] Production entry points implement the scoped deliverable: Negotiate supported majors, preserve core-only peers, expose unsupported directory/backend activation, and refuse no-common-major
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
TASK-260908-2tkufa assigns the approved v0.6.0 delta owner. Original AC and partial work remain preserved; pending runtime support is not claimed by catalog adoption. Own subsequent major negotiation under sections 11.2-11.3/17 with adopted RPC 5 and explicit historical RPC 2/3/4 bounds. Config-4 Host Channel endpoints select RPC 5 only; legacy installations keep explicit historical behavior. Negotiation cannot downgrade a required authenticated endpoint or substitute for TASK-260830-z1yxg9 admission.
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:009eecf230ad676c71fa02f210ad171dc6936cae601b3bc028fa753d219ff3c1 rationale="Next fan-out (M2 first leaf: RPC major negotiation over rpcwire/hostchannel) unblocked by STORY-260830-1kiyj6; muse-spark max is the primary configured producer."
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (muse) (run=RUN-260917-b15a5b, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260917-b15a5b)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260917-b15a5b, pid=76784, exit=0)
spawn autonomous recovery: run RUN-260917-b15a5b queued successor RUN-260917-244f06 (attempt 1/3, model=muse-spark): Change Request construction for TASK-260830-219okr failed: Change Request CR-TASK-260830-219okr-1 revision 1 validation failed at command 5/27 (1-based) with exit code 1; log resource TASK-260830-219okr_change-request_rev1-validation.log; retry: fix the failure and complete the producer again; the configured suite will rerun automatically
spawn run started: [implementer] developer (muse) (run=RUN-260917-244f06)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260917-244f06, pid=9322, exit=0)
spawn autonomous recovery: run RUN-260917-244f06 queued successor RUN-260917-27bea1 (attempt 2/3, model=muse-spark): Change Request construction for TASK-260830-219okr failed: Change Request CR-TASK-260830-219okr-2 revision 2 validation failed at command 5/27 (1-based) with exit code 1; log resource TASK-260830-219okr_change-request_rev2-validation.log; retry: fix the failure and complete the producer again; the configured suite will rerun automatically
spawn run started: [implementer] developer (muse) (run=RUN-260917-27bea1)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260917-27bea1, pid=57043, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:dd9abb977b5b89fdf2151c43c20e944d7ed0ca7e20f61d52e87424a10564fe3c rationale="Independent review of the first-leaf CR3 of TASK-260830-219okr; operator routing: independent reviews on claude-opus-5 max (PR48) while producers stay on muse-spark max; codex Astra remains the fallback ceiling."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [reviewer] reviewer (claude) (run=RUN-260917-92b548, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260917-92b548)
review rev3 (RUN-260917-92b548): CHANGES REQUESTED -> to-dev, repeat-of: none. F1 P2 AC exposure tokens pinned circularly (same-exit code swaps R19/R20 survive); F2 P2 Section 6.6 unknown-generation refusal pinned only at LocalMajors, Negotiate legacy-fallback plant R12 survives; F3 P2 doc names config.Configuration.SchemaVersion, which config normalizes to 3.0.0 for legacy sources (use LoadedConfiguration.SourceVersion); F4-F6 P3 (PeerOffer trusts non-rpc arrays from decode; frame-vocabulary 4.x/5.x edge unmeasured; TRACEABILITY 17.1 wording). Production behavior matched all 32 pair rows and 32 hostile offers; producer harness reran twice identically. Evidence: TASK-260830-219okr_review-verdict-rev3.md, TASK-260830-219okr_review-evidence-rev3.tar.gz
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260917-92b548, pid=6590, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:009eecf230ad676c71fa02f210ad171dc6936cae601b3bc028fa753d219ff3c1 rationale="Rework of CR3 after the claude-opus-5 max review requested changes (no P1: the selection logic is right; circularly pinned exposure tokens, the 6.6 unknown-generation refusal pinned only at the helper, and a documented generation field the config owner normalizes away); evidence-and-documentation workload on the same muse-spark max producer ceiling."
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (muse) (run=RUN-260917-8fb798, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260917-8fb798)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260917-8fb798, pid=23238, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:dd9abb977b5b89fdf2151c43c20e944d7ed0ca7e20f61d52e87424a10564fe3c rationale="Independent review of the first-leaf CR4 of TASK-260830-219okr; operator routing: independent reviews on claude-opus-5 max (PR48) while producers stay on muse-spark max; codex Astra remains the fallback ceiling."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [reviewer] reviewer (claude) (run=RUN-260917-b1ebcb, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260917-b1ebcb)
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260917-b1ebcb, pid=81217, exit=0)
spawn workload selection: class=operations source=explicit policy=spawn.workload_classes pair=muse/muse-spark/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:58b0a9b9fa840607602e2f3176d3ba574704cd38bb765f43b6dbbd06f7d2a566 rationale="Producer-bound checkpoint-only run for the accepted non-final CR4 of TASK-260830-219okr; muse-spark max producer-role run."
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (muse) (run=RUN-260917-9db189, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260917-9db189)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260917-9db189, pid=86691, exit=0)

## Precondition Resources
- [TASK-260830-219okr_producer.md](file://TASK-260830-219okr/TASK-260830-219okr_producer.md)
- [TASK-260830-219okr_rework-rev4.md](file://TASK-260830-219okr/TASK-260830-219okr_rework-rev4.md)
- [TASK-260830-219okr_reviewer-cr4.md](file://TASK-260830-219okr/TASK-260830-219okr_reviewer-cr4.md)
- [TASK-260830-219okr_checkpoint-brief-rev4.md](file://TASK-260830-219okr/TASK-260830-219okr_checkpoint-brief-rev4.md)

## Outcome Resources
- [TASK-260830-219okr_spawn-log_-implementer--developer--muse-_RUN-260917-b15a5b.log](file://TASK-260830-219okr/TASK-260830-219okr_spawn-log_-implementer--developer--muse-_RUN-260917-b15a5b.log) — System spawn log captured by task-board
- [TASK-260830-219okr_results.md](file://TASK-260830-219okr/TASK-260830-219okr_results.md) — Rework-rev4 handoff evidence with finding-by-finding table
- [TASK-260830-219okr_conformance-matrix.md](file://TASK-260830-219okr/TASK-260830-219okr_conformance-matrix.md) — Conformance matrix rev3: literal pins, Negotiate-level unknown rows, 31 mutants
- [TASK-260830-219okr_producer-evidence.tar.gz](file://TASK-260830-219okr/TASK-260830-219okr_producer-evidence.tar.gz) — Rework-rev4 evidence: suite logs, 33 per-plant logs, leaf logs
- [TASK-260830-219okr_change-request_rev1.patch](file://TASK-260830-219okr/TASK-260830-219okr_change-request_rev1.patch) — Change Request CR-TASK-260830-219okr-1 revision 1 candidate patch (repository_delta=present, 8 changed paths)
- [TASK-260830-219okr_change-request_rev1-validation.log](file://TASK-260830-219okr/TASK-260830-219okr_change-request_rev1-validation.log) — Change Request CR-TASK-260830-219okr-1 revision 1 bounded validation log
- [TASK-260830-219okr_spawn-log_-implementer--developer--muse-_RUN-260917-244f06.log](file://TASK-260830-219okr/TASK-260830-219okr_spawn-log_-implementer--developer--muse-_RUN-260917-244f06.log) — System spawn log captured by task-board
- [TASK-260830-219okr_change-request_rev2.patch](file://TASK-260830-219okr/TASK-260830-219okr_change-request_rev2.patch) — Change Request CR-TASK-260830-219okr-2 revision 2 candidate patch (repository_delta=present, 8 changed paths)
- [TASK-260830-219okr_change-request_rev2-validation.log](file://TASK-260830-219okr/TASK-260830-219okr_change-request_rev2-validation.log) — Change Request CR-TASK-260830-219okr-2 revision 2 bounded validation log
- [TASK-260830-219okr_spawn-log_-implementer--developer--muse-_RUN-260917-27bea1.log](file://TASK-260830-219okr/TASK-260830-219okr_spawn-log_-implementer--developer--muse-_RUN-260917-27bea1.log) — System spawn log captured by task-board
- [TASK-260830-219okr_change-request_rev3.patch](file://TASK-260830-219okr/TASK-260830-219okr_change-request_rev3.patch) — Change Request CR-TASK-260830-219okr-3 revision 3 candidate patch (repository_delta=present, 8 changed paths)
- [TASK-260830-219okr_change-request_rev3-validation.log](file://TASK-260830-219okr/TASK-260830-219okr_change-request_rev3-validation.log) — Change Request CR-TASK-260830-219okr-3 revision 3 bounded validation log
- [TASK-260830-219okr_spawn-log_-reviewer--reviewer--claude-_RUN-260917-92b548.log](file://TASK-260830-219okr/TASK-260830-219okr_spawn-log_-reviewer--reviewer--claude-_RUN-260917-92b548.log) — System spawn log captured by task-board
- [TASK-260830-219okr_review-verdict-rev3.md](file://TASK-260830-219okr/TASK-260830-219okr_review-verdict-rev3.md) — Review verdict rev3: changes requested (F1-F3 P2, F4-F6 P3), reviewer instruments and verified items
- [TASK-260830-219okr_review-evidence-rev3.tar.gz](file://TASK-260830-219okr/TASK-260830-219okr_review-evidence-rev3.tar.gz) — Review evidence rev3: suite logs, producer harness reruns x2, reviewer plants (12 killed / 4 survived / control), survivor reruns, probe sources, config probe
- [TASK-260830-219okr_spawn-log_-implementer--developer--muse-_RUN-260917-8fb798.log](file://TASK-260830-219okr/TASK-260830-219okr_spawn-log_-implementer--developer--muse-_RUN-260917-8fb798.log) — System spawn log captured by task-board
- [TASK-260830-219okr_change-request_rev4.patch](file://TASK-260830-219okr/TASK-260830-219okr_change-request_rev4.patch) — Change Request CR-TASK-260830-219okr-4 revision 4 candidate patch (repository_delta=present, 8 changed paths)
- [TASK-260830-219okr_change-request_rev4-validation.log](file://TASK-260830-219okr/TASK-260830-219okr_change-request_rev4-validation.log) — Change Request CR-TASK-260830-219okr-4 revision 4 bounded validation log
- [TASK-260830-219okr_spawn-log_-reviewer--reviewer--claude-_RUN-260917-b1ebcb.log](file://TASK-260830-219okr/TASK-260830-219okr_spawn-log_-reviewer--reviewer--claude-_RUN-260917-b1ebcb.log) — System spawn log captured by task-board
- [TASK-260830-219okr_review-verdict-rev4.md](file://TASK-260830-219okr/TASK-260830-219okr_review-verdict-rev4.md) — Review verdict rev4: ACCEPTED (F1-F6 fixed, rev3 survivors die x2, 0xP1/0xP2, 3xP3 advisories), reviewer instruments and verified items
- [TASK-260830-219okr_review-evidence-rev4.tar.gz](file://TASK-260830-219okr/TASK-260830-219okr_review-evidence-rev4.tar.gz) — Review evidence rev4: suite logs (full go test ./... 38 ok), producer harness reruns x2, reviewer plants x2 (36 rows), survivor kill-control, per-key sweep, probe sources, notes
- [TASK-260830-219okr_spawn-log_-implementer--developer--muse-_RUN-260917-9db189.log](file://TASK-260830-219okr/TASK-260830-219okr_spawn-log_-implementer--developer--muse-_RUN-260917-9db189.log) — System spawn log captured by task-board
- [TASK-260830-219okr_checkpoint-rev4.md](file://TASK-260830-219okr/TASK-260830-219okr_checkpoint-rev4.md) — Checkpoint rev4: signed internal checkpoint e390d5b on Story branch, task integrating

## Created
2026-08-29T22:00:53Z

## Last Update
2026-09-18T00:23:00Z

## Assigned To
[implementer] developer (muse)
