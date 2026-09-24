## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(8))

## Blocked By
- TASK-260830-nxqqaw

## Blocks
- TASK-260830-2h5uv9

## Checklist
- [x] Production entry points implement the scoped deliverable: Find missing objects, fetch by digest, preserve concurrent branches, and rebuild projections without timestamp authority
- [x] Relevant positive, negative, compatibility, and recovery tests pass with logs attached
- [x] README/doctor/capability evidence and specification traceability are updated without unsupported claims
- [x] Carry-over from TASK-260830-nxqqaw rev3 review notes (reachable arms unpinned by the committed suite): pin FetchObjects client duplicate-ID refusal (fetch-dup-ids), the base64url round-trip refusal of newline-bearing data (b64-roundtrip), the recursive walk quarantine refusal integrity_failure (walk-quarantine, reachable per the reviewer probe), and the walk cross-namespace arm (pin or prove unreachable with reason); each with a narrowing killed by its named test run alone
- [x] Durable union per pinned SPEC v0.7.0 §11.4 rules 1-4 (7925-7933): validated add persists, identical add is idempotent across crash/restart, same digest with different bytes quarantines and aborts the sync, tombstones and acks are unioned not executed; crash/idempotency evidence for every durable mutation
- [x] Concurrent branches and lease heads (§11.4 rules 5-6, §5.3 1987-2061 incl. 2047): losing-lease events are preserved in divergent branches and never affect authoritative state; lease heads are derived after union; an exhaustive arrival-order oracle over every permutation of a small event set (N<=6, including competing leases and perturbed timestamps) yields one identical projection; no timestamp selects a winner
- [x] Projection rebuild is a pure function of the unioned object set: rebuild from any arrival order and from any timestamp perturbation equals the reference; gate x entry census with axes (arrival order, timestamp perturbation, branch shape, namespace), isolated kill attribution, importer outcome grid with moved classes named; task_delta scope, no registry edit
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
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=codex/gpt-6-luna/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:d542feefb585033beb33bb354c9628daa57c01da8e690d98c41826bb8d8155aa rationale="Second leaf of M2 STORY-260830-ub60id (durable union, arrival-order independence, projection rebuild) after nxqqaw checkpoint b81258e; chain 2 gpt-6-luna max."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (codex) (run=RUN-260923-1e9c40, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260923-1e9c40)
Run RUN-260923-1e9c40; checkpoint b81258e3c5bc0f6321ae8f6bea81ec24cc298d70; candidate uncommitted. Implemented durable validated union, missing-object fetch by digest, Tombstone/Acknowledgement union closure, lease tuple derivation and projection rebuild without timestamp authority. Measured acceptance 6/6; six additions across 720 permutations and two timestamp profiles (1,440 builds); final mutation harness 70 narrowing kills, token-preserving bypass killed by behavior, neutral control survived. Package/importer/full suite, native and Windows vet, delta golangci lint, gofmt and diff check exit 0; full default golangci lint exit 1 on existing repository findings with no project config. Bounds and out-of-contract rows are in results, conformance and logbook resources. No §11.5–§11.6 staging/commit, arbitrary event-DAG rebuild, physical power-loss/native Windows crash, public ax sync/transport or capability claim. Four task-scoped attachments were read back and hash-verified. No registry edit; candidate remains uncommitted at the Story checkpoint.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260923-1e9c40, pid=15820, exit=0)
spawn autonomous recovery: run RUN-260923-1e9c40 queued successor RUN-260924-51f38b (attempt 1/3, model=gpt-6-luna): Change Request construction for TASK-260830-147hsj failed: Change Request CR-TASK-260830-147hsj-1 revision 1 validation failed at command 5/30 (1-based) with exit code 1; log resource TASK-260830-147hsj_change-request_rev1-validation.log; retry: fix the failure and complete the producer again; the configured suite will rerun automatically
spawn run started: [implementer] developer (codex) (run=RUN-260924-51f38b)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260924-51f38b, pid=84642, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5-5/low pair_source=explicit match=recommended_rank_1 snapshot=sha256:6ba039c6604a651eb8bc6bcb8eb8bd97cc537695632059c62a97ed7defa7aede rationale="Independent review of the first-leaf CR2 of TASK-260830-147hsj; operator routing: independent reviews on claude-opus-5-5 low; producers on codex gpt-6-luna max (PR #58)"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [reviewer] reviewer (claude) (run=RUN-260924-19d34f, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260924-19d34f)
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260924-19d34f, pid=7071, exit=0)
spawn workload selection: class=operations source=explicit policy=spawn.workload_classes pair=codex/gpt-6-luna/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:0ce9c0cd7a693f55247a7e64b3ead89fb43b5707737a2c69a43ff57cb492b9e0 rationale="Producer-bound checkpoint run for accepted CR rev2 of TASK-260830-147hsj (runner performs worktree checkpoint)."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (codex) (run=RUN-260924-e43ebf, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260924-e43ebf)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260924-e43ebf, pid=98248, exit=0)

## Precondition Resources
- [TASK-260830-147hsj_producer.md](file://TASK-260830-147hsj/TASK-260830-147hsj_producer.md)
- [TASK-260830-147hsj_reviewer-cr2.md](file://TASK-260830-147hsj/TASK-260830-147hsj_reviewer-cr2.md)
- [TASK-260830-147hsj_checkpoint-rev2.md](file://TASK-260830-147hsj/TASK-260830-147hsj_checkpoint-rev2.md)

## Outcome Resources
- [TASK-260830-147hsj_spawn-log_-implementer--developer--codex-_RUN-260923-1e9c40.log](file://TASK-260830-147hsj/TASK-260830-147hsj_spawn-log_-implementer--developer--codex-_RUN-260923-1e9c40.log) — System spawn log captured by task-board
- [TASK-260830-147hsj_results.md](file://TASK-260830-147hsj/TASK-260830-147hsj_results.md) — Developer results including accepted handoff status and validation exits
- [TASK-260830-147hsj_conformance-matrix.md](file://TASK-260830-147hsj/TASK-260830-147hsj_conformance-matrix.md) — Measured conformance matrix with production call sites and bounds
- [TASK-260830-147hsj_logbook.md](file://TASK-260830-147hsj/TASK-260830-147hsj_logbook.md) — Task logbook including validation recovery and handoff
- [TASK-260830-147hsj_producer-evidence.tar.gz](file://TASK-260830-147hsj/TASK-260830-147hsj_producer-evidence.tar.gz) — Updated producer evidence and validation recovery logs; 1021257 bytes
- [TASK-260830-147hsj_change-request_rev1.patch](file://TASK-260830-147hsj/TASK-260830-147hsj_change-request_rev1.patch) — Change Request CR-TASK-260830-147hsj-1 revision 1 candidate patch (repository_delta=present, 14 changed paths)
- [TASK-260830-147hsj_change-request_rev1-validation.log](file://TASK-260830-147hsj/TASK-260830-147hsj_change-request_rev1-validation.log) — Change Request CR-TASK-260830-147hsj-1 revision 1 bounded validation log
- [TASK-260830-147hsj_spawn-log_-implementer--developer--codex-_RUN-260924-51f38b.log](file://TASK-260830-147hsj/TASK-260830-147hsj_spawn-log_-implementer--developer--codex-_RUN-260924-51f38b.log) — System spawn log captured by task-board
- [TASK-260830-147hsj_change-request_rev2.patch](file://TASK-260830-147hsj/TASK-260830-147hsj_change-request_rev2.patch) — Change Request CR-TASK-260830-147hsj-2 revision 2 candidate patch (repository_delta=present, 14 changed paths)
- [TASK-260830-147hsj_change-request_rev2-validation.log](file://TASK-260830-147hsj/TASK-260830-147hsj_change-request_rev2-validation.log) — Change Request CR-TASK-260830-147hsj-2 revision 2 bounded validation log
- [TASK-260830-147hsj_spawn-log_-reviewer--reviewer--claude-_RUN-260924-19d34f.log](file://TASK-260830-147hsj/TASK-260830-147hsj_spawn-log_-reviewer--reviewer--claude-_RUN-260924-19d34f.log) — System spawn log captured by task-board
- [TASK-260830-147hsj_review-verdict-rev2.md](file://TASK-260830-147hsj/TASK-260830-147hsj_review-verdict-rev2.md) — Reviewer verdict CR rev2: accepted
- [TASK-260830-147hsj_review-evidence-rev2.tar.gz](file://TASK-260830-147hsj/TASK-260830-147hsj_review-evidence-rev2.tar.gz) — Review evidence rev2: plants, harness reruns, importer grid, JCS check
- [TASK-260830-147hsj_spawn-log_-implementer--developer--codex-_RUN-260924-e43ebf.log](file://TASK-260830-147hsj/TASK-260830-147hsj_spawn-log_-implementer--developer--codex-_RUN-260924-e43ebf.log) — System spawn log captured by task-board
- [TASK-260830-147hsj_checkpoint-preconditions-rev2.md](file://TASK-260830-147hsj/TASK-260830-147hsj_checkpoint-preconditions-rev2.md) — Fresh integration checkpoint precondition record for accepted CR revision 2

## Created
2026-08-29T22:00:56Z

## Last Update
2026-09-24T22:25:57Z

## Assigned To
[implementer] developer (codex)
