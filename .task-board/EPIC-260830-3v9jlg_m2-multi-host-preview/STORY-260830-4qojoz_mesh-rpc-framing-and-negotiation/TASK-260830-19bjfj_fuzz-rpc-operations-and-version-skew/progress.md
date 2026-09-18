## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(8))

## Blocked By
- TASK-260830-219okr

## Blocks
- TASK-260830-nxqqaw
- TASK-260830-cmzdsq
- TASK-260830-27abiw
- TASK-260830-2wflxd
- TASK-260830-2ewl6a

## Checklist
- [x] Production entry points implement the scoped deliverable: Fuzz every closed operation body, namespace mismatch, unknown field, replay, lost response, and mixed-version pairing
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
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:009eecf230ad676c71fa02f210ad171dc6936cae601b3bc028fa753d219ff3c1 rationale="Final leaf of STORY-260830-4qojoz: closed operation bodies across the 24-operation RPC table, bounded fuzz targets with controls, Section 17 version-skew driven not asserted, replay/lost-response through materialize.status, 8 MiB framing edges, plus the Story-close registry bindings and pin re-derivation; implementation workload, muse-spark max producer with an independent claude-opus-5 max review."
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (muse) (run=RUN-260917-cea647, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260917-cea647)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260917-cea647, pid=10739, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:dd9abb977b5b89fdf2151c43c20e944d7ed0ca7e20f61d52e87424a10564fe3c rationale="Independent review of the story_final CR1 of TASK-260830-19bjfj; operator routing: independent reviews on claude-opus-5 max (PR48) while producers stay on muse-spark max; codex Astra remains the fallback ceiling."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [reviewer] reviewer (claude) (run=RUN-260917-81d7ae, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260917-81d7ae)
REVIEW CR rev1 (RUN-260917-81d7ae, claude-opus-5 max): CHANGES REQUESTED -> to-dev. P1-1 story_final candidate built on stale base 888ae3d while trunk 2fc6d50 adopted v0.7.0 (PR #52): bindings/digest/README/LOGBOOK target ownership.v0.6.0.json, git apply --check fails on 5 trunk-moved paths; refresh-candidate was required by the brief and never run. P2-1 registry clause 11.3#18 (digest-identity arrays) discharged by namespace lanes that drive no digest array. P3-1..P3-8: opaque-body object gate unpinned (R9 survived x2), changed-body retry after commit unpinned (J1 survived x2), body-fuzzer oracle is member-name-only and unknown-field fuzzer lacks a hello-body seed, encode 8388609 lane not at the edge, error-schema refusal class unpinned, production-derived Local expectations, fixture-aimed alphabet mutant, missing raw logs/citation. HOLDS: N1/N2/N3 closed (24/24 non-rpc keys, S12, S14 killed x2); producer harnesses reproduce (40+37 killed, controls survive); 3 fuzz targets execute and fail on controls; 38 reviewer plants killed x2; full suite 38 ok; 23 of 29 AC rows driven confirmed. Verdict: TASK-260830-19bjfj_review-verdict-rev1.md; evidence: TASK-260830-19bjfj_review-evidence-rev1.tar.gz.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260917-81d7ae, pid=5037, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:009eecf230ad676c71fa02f210ad171dc6936cae601b3bc028fa753d219ff3c1 rationale="Rework of the story_final CR1 after the claude-opus-5 max review: the technical body is strong and stays, but the candidate was built on the stale base 888ae3d and bound the no-longer-adopted v0.6.0 ownership registry, so the whole-Story patch does not apply on trunk and the registry work is invisible; refresh onto 2fc6d50, move the bindings, re-measure the clause coordinates and re-derive the pin."
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (muse) (run=RUN-260917-a36df8, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260917-a36df8)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260917-a36df8, pid=88987, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:dd9abb977b5b89fdf2151c43c20e944d7ed0ca7e20f61d52e87424a10564fe3c rationale="Independent review of the story_final CR2 of TASK-260830-19bjfj; operator routing: independent reviews on claude-opus-5 max (PR48) while producers stay on muse-spark max; codex Astra remains the fallback ceiling."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [reviewer] reviewer (claude) (run=RUN-260917-e5ea66, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260917-e5ea66)
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260917-e5ea66, pid=91936, exit=0)
spawn workload selection: class=operations source=explicit policy=spawn.workload_classes pair=muse/muse-spark/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:58b0a9b9fa840607602e2f3176d3ba574704cd38bb765f43b6dbbd06f7d2a566 rationale="Producer-bound integration run for the accepted story_final CR2 of TASK-260830-19bjfj (worktree integrate, exact-head suite, delivery branch + PR; orchestrator lands); muse-spark max while codex is out of quota."
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (muse) (run=RUN-260918-d105c9, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260918-d105c9)
agent completed: [implementer] developer (muse) (exit=143)
spawn run RUN-260918-d105c9 cancelled by operator; operator action required; reason: no operator reason supplied
spawn run completed: muse (run=RUN-260918-d105c9, pid=37890, exit=143)
spawn workload selection: class=operations source=explicit policy=spawn.workload_classes pair=muse/muse-spark/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:58b0a9b9fa840607602e2f3176d3ba574704cd38bb765f43b6dbbd06f7d2a566 rationale="Producer-bound integration run for the accepted story_final CR2 of TASK-260830-19bjfj (worktree integrate, exact-head suite, delivery branch + PR; orchestrator lands); muse-spark max while codex is out of quota."
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (muse) (run=RUN-260918-d5c944, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260918-d5c944)

## Precondition Resources
- [TASK-260830-19bjfj_rework-rev2.md](file://TASK-260830-19bjfj/TASK-260830-19bjfj_rework-rev2.md)
- [TASK-260830-19bjfj_reviewer-cr2.md](file://TASK-260830-19bjfj/TASK-260830-19bjfj_reviewer-cr2.md)
- [TASK-260830-19bjfj_integration-rev2.md](file://TASK-260830-19bjfj/TASK-260830-19bjfj_integration-rev2.md)

## Outcome Resources
- [TASK-260830-19bjfj_spawn-log_-implementer--developer--muse-_RUN-260917-cea647.log](file://TASK-260830-19bjfj/TASK-260830-19bjfj_spawn-log_-implementer--developer--muse-_RUN-260917-cea647.log) — System spawn log captured by task-board
- [TASK-260830-19bjfj_results.md](file://TASK-260830-19bjfj/TASK-260830-19bjfj_results.md) — Handoff evidence rev2
- [TASK-260830-19bjfj_conformance-matrix.md](file://TASK-260830-19bjfj/TASK-260830-19bjfj_conformance-matrix.md) — Conformance matrix rev2
- [TASK-260830-19bjfj_producer-evidence.tar.gz](file://TASK-260830-19bjfj/TASK-260830-19bjfj_producer-evidence.tar.gz) — Producer evidence rev2
- [TASK-260830-19bjfj_change-request_rev1.patch](file://TASK-260830-19bjfj/TASK-260830-19bjfj_change-request_rev1.patch) — Change Request CR-TASK-260830-19bjfj-1 revision 1 candidate patch (repository_delta=present, 18 changed paths)
- [TASK-260830-19bjfj_change-request_rev1-validation.log](file://TASK-260830-19bjfj/TASK-260830-19bjfj_change-request_rev1-validation.log) — Change Request CR-TASK-260830-19bjfj-1 revision 1 bounded validation log
- [TASK-260830-19bjfj_spawn-log_-reviewer--reviewer--claude-_RUN-260917-81d7ae.log](file://TASK-260830-19bjfj/TASK-260830-19bjfj_spawn-log_-reviewer--reviewer--claude-_RUN-260917-81d7ae.log) — System spawn log captured by task-board
- [TASK-260830-19bjfj_review-verdict-rev1.md](file://TASK-260830-19bjfj/TASK-260830-19bjfj_review-verdict-rev1.md) — Independent review verdict for CR rev1: CHANGES REQUESTED (P1 stale base / v0.6.0 registry bound while trunk adopted v0.7.0; P2 invented 11.3#18 binding; 8 P3 test/evidence gaps); N1-N3 closed
- [TASK-260830-19bjfj_review-evidence-rev1.tar.gz](file://TASK-260830-19bjfj/TASK-260830-19bjfj_review-evidence-rev1.tar.gz) — Reviewer evidence for CR rev1: reviewer mutant harness (45 rows x2 runs, per-plant raw logs), fuzz controls, producer harness reruns, validation logs, trunk apply-check, digest derivation
- [TASK-260830-19bjfj_spawn-log_-implementer--developer--muse-_RUN-260917-a36df8.log](file://TASK-260830-19bjfj/TASK-260830-19bjfj_spawn-log_-implementer--developer--muse-_RUN-260917-a36df8.log) — System spawn log captured by task-board
- [TASK-260830-19bjfj_change-request_rev2.patch](file://TASK-260830-19bjfj/TASK-260830-19bjfj_change-request_rev2.patch) — Change Request CR-TASK-260830-19bjfj-2 revision 2 candidate patch (repository_delta=present, 20 changed paths)
- [TASK-260830-19bjfj_change-request_rev2-validation.log](file://TASK-260830-19bjfj/TASK-260830-19bjfj_change-request_rev2-validation.log) — Change Request CR-TASK-260830-19bjfj-2 revision 2 bounded validation log
- [TASK-260830-19bjfj_spawn-log_-reviewer--reviewer--claude-_RUN-260917-e5ea66.log](file://TASK-260830-19bjfj/TASK-260830-19bjfj_spawn-log_-reviewer--reviewer--claude-_RUN-260917-e5ea66.log) — System spawn log captured by task-board
- [TASK-260830-19bjfj_review-verdict-rev2.md](file://TASK-260830-19bjfj/TASK-260830-19bjfj_review-verdict-rev2.md) — Reviewer verdict for CR rev2 (RUN-260917-e5ea66): ACCEPTED; rev1 findings graded on executed evidence; advisories A1-A3; trunk resumesmoke time bomb escalated
- [TASK-260830-19bjfj_review-evidence-rev2.tar.gz](file://TASK-260830-19bjfj/TASK-260830-19bjfj_review-evidence-rev2.tar.gz) — Reviewer evidence for CR rev2: mutation batteries (own + producer reruns), fuzz controls, digest derivation, registry/README perturbations, raw logs with exits
- [TASK-260830-19bjfj_spawn-log_-implementer--developer--muse-_RUN-260918-d105c9.log](file://TASK-260830-19bjfj/TASK-260830-19bjfj_spawn-log_-implementer--developer--muse-_RUN-260918-d105c9.log) — System spawn log captured by task-board
- [TASK-260830-19bjfj_spawn-log_-implementer--developer--muse-_RUN-260918-d5c944.log](file://TASK-260830-19bjfj/TASK-260830-19bjfj_spawn-log_-implementer--developer--muse-_RUN-260918-d5c944.log) — System spawn log captured by task-board

## Created
2026-08-29T22:00:53Z

## Last Update
2026-09-18T00:23:00Z

## Assigned To
[implementer] developer (muse)
