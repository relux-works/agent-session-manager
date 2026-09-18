## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(13))

## Blocked By
- TASK-260830-24z2b3

## Blocks
- TASK-260830-32ypr2

## Checklist
- [x] Production entry points implement the scoped deliverable: Capture stable provider store bytes, item identities, workspace checkpoint, source head, and detect mutation before projection
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
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:009eecf230ad676c71fa02f210ad171dc6936cae601b3bc028fa753d219ff3c1 rationale="Second leaf of STORY-260830-1cyj0q: the capture operation itself (real store walk with dirnode containment, item classification, Stable Snapshot Proof, the source-race check ordered before projection, workspace binding, Section 10.2 blob discipline) over the landed clonebundle types; implementation workload, muse-spark max producer with an independent claude-opus-5 max review."
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (muse) (run=RUN-260918-65928a, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260918-65928a)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260918-65928a, pid=50720, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:dd9abb977b5b89fdf2151c43c20e944d7ed0ca7e20f61d52e87424a10564fe3c rationale="Independent review of the first-leaf CR1 of TASK-260830-2g5be6; operator routing: independent reviews on claude-opus-5 max (PR48) while producers stay on muse-spark max; codex Astra remains the fallback ceiling."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [reviewer] reviewer (claude) (run=RUN-260918-cee501, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260918-cee501)
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260918-cee501, pid=54794, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:009eecf230ad676c71fa02f210ad171dc6936cae601b3bc028fa753d219ff3c1 rationale="Rework of CR1 after the claude-opus-5 max review: production is correct on every probe the reviewer drove; the block is the evidence contract — five AC rows whose cited tests do not reach the named member (a narrowing survives 2/2 while the rows stay green), one of them a Section 10.2 MUST whose tests never look at a payload blob, plus the three inherited advisories neither closed nor deferred in the results."
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (muse) (run=RUN-260918-5a9957, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260918-5a9957)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260918-5a9957, pid=84856, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:dd9abb977b5b89fdf2151c43c20e944d7ed0ca7e20f61d52e87424a10564fe3c rationale="Independent review of the first-leaf CR2 of TASK-260830-2g5be6; operator routing: independent reviews on claude-opus-5 max (PR48) while producers stay on muse-spark max; codex Astra remains the fallback ceiling."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [reviewer] reviewer (claude) (run=RUN-260918-fdce92, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260918-fdce92)
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260918-fdce92, pid=37025, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:009eecf230ad676c71fa02f210ad171dc6936cae601b3bc028fa753d219ff3c1 rationale="Rework of CR2: everything rev1 asked for is closed and production is correct on all 36 probes the reviewer drove; two AC rows still name a member their test never reaches (the unstable_archive pre/post digests, and 'before any seal' unpinned at CaptureAndProject), so the published 17-of-17 is really 15 of 17 — two assertions, four harness rows and an honest ratio."
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (muse) (run=RUN-260918-0cd4a1, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260918-0cd4a1)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260918-0cd4a1, pid=79313, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:dd9abb977b5b89fdf2151c43c20e944d7ed0ca7e20f61d52e87424a10564fe3c rationale="Independent review of the first-leaf CR3 of TASK-260830-2g5be6; operator routing: independent reviews on claude-opus-5 max (PR48) while producers stay on muse-spark max; codex Astra remains the fallback ceiling."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [reviewer] reviewer (claude) (run=RUN-260918-4b4c38, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260918-4b4c38)
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260918-4b4c38, pid=67079, exit=0)
spawn workload selection: class=operations source=explicit policy=spawn.workload_classes pair=muse/muse-spark/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:58b0a9b9fa840607602e2f3176d3ba574704cd38bb765f43b6dbbd06f7d2a566 rationale="Producer-bound checkpoint-only run for the accepted non-final CR3 of TASK-260830-2g5be6; muse-spark max producer-role run."
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (muse) (run=RUN-260918-4af317, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260918-4af317)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260918-4af317, pid=57624, exit=0)

## Precondition Resources
- [TASK-260830-2g5be6_producer.md](file://TASK-260830-2g5be6/TASK-260830-2g5be6_producer.md)
- [TASK-260830-2g5be6_rework-rev3.md](file://TASK-260830-2g5be6/TASK-260830-2g5be6_rework-rev3.md)
- [TASK-260830-2g5be6_reviewer-cr3.md](file://TASK-260830-2g5be6/TASK-260830-2g5be6_reviewer-cr3.md)
- [TASK-260830-2g5be6_checkpoint-brief-rev3.md](file://TASK-260830-2g5be6/TASK-260830-2g5be6_checkpoint-brief-rev3.md)

## Outcome Resources
- [TASK-260830-2g5be6_spawn-log_-implementer--developer--muse-_RUN-260918-65928a.log](file://TASK-260830-2g5be6/TASK-260830-2g5be6_spawn-log_-implementer--developer--muse-_RUN-260918-65928a.log) — System spawn log captured by task-board
- [TASK-260830-2g5be6_results.md](file://TASK-260830-2g5be6/TASK-260830-2g5be6_results.md) — Handoff evidence (rev3)
- [TASK-260830-2g5be6_conformance-matrix.md](file://TASK-260830-2g5be6/TASK-260830-2g5be6_conformance-matrix.md) — Conformance matrix (rev3)
- [TASK-260830-2g5be6_producer-evidence.tar.gz](file://TASK-260830-2g5be6/TASK-260830-2g5be6_producer-evidence.tar.gz) — Producer evidence archive (rev3)
- [TASK-260830-2g5be6_change-request_rev1.patch](file://TASK-260830-2g5be6/TASK-260830-2g5be6_change-request_rev1.patch) — Change Request CR-TASK-260830-2g5be6-1 revision 1 candidate patch (repository_delta=present, 22 changed paths)
- [TASK-260830-2g5be6_change-request_rev1-validation.log](file://TASK-260830-2g5be6/TASK-260830-2g5be6_change-request_rev1-validation.log) — Change Request CR-TASK-260830-2g5be6-1 revision 1 bounded validation log
- [TASK-260830-2g5be6_spawn-log_-reviewer--reviewer--claude-_RUN-260918-cee501.log](file://TASK-260830-2g5be6/TASK-260830-2g5be6_spawn-log_-reviewer--reviewer--claude-_RUN-260918-cee501.log) — System spawn log captured by task-board
- [TASK-260830-2g5be6_review-verdict-rev1.md](file://TASK-260830-2g5be6/TASK-260830-2g5be6_review-verdict-rev1.md) — Independent review verdict for CR rev1: CHANGES REQUESTED (P2 evidence contract: 5 of 17 AC rows carry surviving narrowings; inherited advisories silently dropped)
- [TASK-260830-2g5be6_review-evidence-rev1.tar.gz](file://TASK-260830-2g5be6/TASK-260830-2g5be6_review-evidence-rev1.tar.gz) — Reviewer rev1 evidence: probes (117 lines), reviewer mutant battery 15 rows x 2 runs with per-plant raw logs, producer harness reproduced 2x, hygiene logs, verdict copy
- [TASK-260830-2g5be6_spawn-log_-implementer--developer--muse-_RUN-260918-5a9957.log](file://TASK-260830-2g5be6/TASK-260830-2g5be6_spawn-log_-implementer--developer--muse-_RUN-260918-5a9957.log) — System spawn log captured by task-board
- [TASK-260830-2g5be6_change-request_rev2.patch](file://TASK-260830-2g5be6/TASK-260830-2g5be6_change-request_rev2.patch) — Change Request CR-TASK-260830-2g5be6-2 revision 2 candidate patch (repository_delta=present, 22 changed paths)
- [TASK-260830-2g5be6_change-request_rev2-validation.log](file://TASK-260830-2g5be6/TASK-260830-2g5be6_change-request_rev2-validation.log) — Change Request CR-TASK-260830-2g5be6-2 revision 2 bounded validation log
- [TASK-260830-2g5be6_spawn-log_-reviewer--reviewer--claude-_RUN-260918-fdce92.log](file://TASK-260830-2g5be6/TASK-260830-2g5be6_spawn-log_-reviewer--reviewer--claude-_RUN-260918-fdce92.log) — System spawn log captured by task-board
- [TASK-260830-2g5be6_review-verdict-rev2.md](file://TASK-260830-2g5be6/TASK-260830-2g5be6_review-verdict-rev2.md) — Independent review verdict for CR rev2: CHANGES REQUESTED (P2 evidence contract: claimed 17/17 member-level ratio measured 15/17 — row 8 sealed unstable pre/post digests unpinned, row 10 seal-before-gate at CaptureAndProject unpinned; P3: unplanned special member, symlinked store root, excluded-member size claim, branch-bound attribution). All five rev1 survivors KILLED 2/2; production correct on every probe.
- [TASK-260830-2g5be6_review-evidence-rev2.tar.gz](file://TASK-260830-2g5be6/TASK-260830-2g5be6_review-evidence-rev2.tar.gz) — Reviewer rev2 evidence: 36 probe lines, reviewer battery 20 rows x 2 runs with per-plant raw logs and subprocess exits (5 rev1 re-plants KILLED, 5 new SURVIVED executed under mutant), producer harness reproduced 35/35 x2, crash injection at AfterWalk x2, hygiene/neighbour/whole-repo logs, MANIFEST.sha256 (137 rows).
- [TASK-260830-2g5be6_spawn-log_-implementer--developer--muse-_RUN-260918-0cd4a1.log](file://TASK-260830-2g5be6/TASK-260830-2g5be6_spawn-log_-implementer--developer--muse-_RUN-260918-0cd4a1.log) — System spawn log captured by task-board
- [TASK-260830-2g5be6_change-request_rev3.patch](file://TASK-260830-2g5be6/TASK-260830-2g5be6_change-request_rev3.patch) — Change Request CR-TASK-260830-2g5be6-3 revision 3 candidate patch (repository_delta=present, 22 changed paths)
- [TASK-260830-2g5be6_change-request_rev3-validation.log](file://TASK-260830-2g5be6/TASK-260830-2g5be6_change-request_rev3-validation.log) — Change Request CR-TASK-260830-2g5be6-3 revision 3 bounded validation log
- [TASK-260830-2g5be6_spawn-log_-reviewer--reviewer--claude-_RUN-260918-4b4c38.log](file://TASK-260830-2g5be6/TASK-260830-2g5be6_spawn-log_-reviewer--reviewer--claude-_RUN-260918-4b4c38.log) — System spawn log captured by task-board
- [TASK-260830-2g5be6_review-verdict-rev3.md](file://TASK-260830-2g5be6/TASK-260830-2g5be6_review-verdict-rev3.md) — Reviewer verdict for CR rev3 (RUN-260918-4b4c38): ACCEPTED, no P1/P2, seven P3 evidence advisories carried forward
- [TASK-260830-2g5be6_review-evidence-rev3.tar.gz](file://TASK-260830-2g5be6/TASK-260830-2g5be6_review-evidence-rev3.tar.gz) — Reviewer evidence for CR rev3: reviewer battery (31 rows x2), producer harness reproduction (44 x2), under-mutant witnesses, probes, crash logs, refusal-site census, hygiene logs, MANIFEST.sha256
- [TASK-260830-2g5be6_spawn-log_-implementer--developer--muse-_RUN-260918-4af317.log](file://TASK-260830-2g5be6/TASK-260830-2g5be6_spawn-log_-implementer--developer--muse-_RUN-260918-4af317.log) — System spawn log captured by task-board
- [TASK-260830-2g5be6_checkpoint-rev3.md](file://TASK-260830-2g5be6/TASK-260830-2g5be6_checkpoint-rev3.md) — Checkpoint-only integration evidence for accepted CR rev3

## Created
2026-08-29T22:02:03Z

## Last Update
2026-09-18T16:09:52Z

## Assigned To
[implementer] developer (muse)
