## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(3))

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] A remote interactive owner with a lapsed local grant yields the attach/takeover offer path of SPEC v0.7.0 4.2 step 4, driven through fencing.Authorize itself and not through a helper, and returns something other than lease_conflict
- [x] The arm ordering is pinned by a committed test that FAILS when the ownership-direction and grant-expiry arms are swapped, and that swap is shipped as a labelled narrowing row in the harness
- [x] The reviewer probe 10 vector from TASK-260830-1geqhj_review-evidence-rev1.tar.gz is reproduced on trunk before the fix and passes after it, with the before/after output recorded
- [x] At least three neighbouring vectors a step away from the reported one are driven and their results stated: a live local grant, a remote NON-interactive owner, and a local owner with a lapsed grant
- [x] Per-test outcomes of the full internal/fencing suite are diffed before and after the change, so any input that silently moved across refusal arms is named in the results rather than hidden by an unchanged pass total
- [x] Scope held: no change to the lease chain and no change to the park vocabulary; if the fix appears to need either, the run stops and says so instead of widening
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
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=codex/gpt-5.6-luna/max pair_source=explicit match=not_recommended snapshot=sha256:62741185359aadc77f1ac9e43857a1ae81255bb7cb14de288b07471d63d2a998 rationale="Operator-directed pair for a parallelism stability test: second concurrent Story chain on codex/gpt-5.6-luna max (admitted by the developer ceiling entries) while the tmux chain stays on muse-spark max. Narrow landed-gate fix (fencing Authorize arm ordering) with a reviewer-supplied reproduction probe, so the work is well-bounded for a first luna run."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (codex) (run=RUN-260921-2649f6, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260921-2649f6)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260921-2649f6, pid=85260, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:dd9abb977b5b89fdf2151c43c20e944d7ed0ca7e20f61d52e87424a10564fe3c rationale="Independent review of the first-leaf CR1 of BUG-260917-2fwf8e; operator routing: independent reviews on claude-opus-5 max (PR48) while producers stay on muse-spark max; codex Astra remains the fallback ceiling."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [reviewer] reviewer (claude) (run=RUN-260921-766ad0, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260921-766ad0)
REVIEW CR rev1 (RUN-260921-766ad0, claude-opus-5 max): CHANGES REQUESTED -> to-dev. Verdict: BUG-260917-2fwf8e_review-verdict-rev1.md; evidence: BUG-260917-2fwf8e_review-evidence-rev1.tar.gz (sha256 d6ebfec8bab846ad9adfa46f067bb65718aa9a56bb472524e91ed6697bda0404). P1-1: the reorder composes into terminstance.ObserveFencing so a STALE incarnation under a REMOTE winner with a LAPSED grant no longer reaches stale_fenced (6 of 60 grid rows regress; SPEC 13.7/13.4 loser MUST stop when it learns the winner); the landed regression test TestRV3F3_GrantLessStaleFences was rewritten to expect the no-transition outcome. Reviewer property test PASS on trunk, FAIL 6/6 on the candidate; trunk test file vs candidate production: 8 red. P2-1: direction-before-expiry pinned on 1 of 5 entries (restore); reviewer narrowings RM-A (restore-only) and RM-H (non-launch reverts) SURVIVED 2/2 across every importer package. P2-2: moved inputs unnamed (remote+lapsed with stale/losing/future/empty-host tokens; non-launch lease_conflict->not_owner); per-test diff fencing-only and name-keyed. P3: foreign TASK-260830-1geqhj archive inside the producer evidence; misnamed remote_noninteractive_owner fencing row; LOGBOOK/TRACEABILITY wording; coverage figure 99.3% measured vs 97.9% stated. Holds: probe 10 reproduced through Authorize, Decide and Run (trunk refused/lease_conflict; candidate attach_remote/takeover_offer/parked remote_owner, no event authored); swap row KILLED 2/2; shipped harness 32/32 KILLED with controls; full 30-command suite green on the exact tree (race 498s under load 9-15). Measured AC coverage 4 of 6 rows. Rework scope in the verdict: fall through to the landed StaleRelativeToWinner fallback in terminstance.observeRemoteWinner when the relative question is not a stale park; restore the stale_fenced assertions; pin all five entries + ModeLaunch; name every moved class with a cross-run diff; hygiene.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260921-766ad0, pid=63216, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=codex/gpt-5.6-luna/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:c4605ab954a19c115501892eed18a53285befd74d0d65bb51e10eff2f33cc4f9 rationale="Rework of CR rev1 of BUG-260917-2fwf8e: one P1 (the reorder stopped stale fencing for the lapsed-grant remote-winner shape in terminstance.ObserveFencing and the landed regression test was rewritten to bless it), plus a 1-of-5-entries pin gap and an unnamed moved-vector set. Operator-directed luna max pair for the parallelism stability test; same producer that owns the rev1 candidate."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (codex) (run=RUN-260921-fcea9e, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260921-fcea9e)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260921-fcea9e, pid=88465, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:dd9abb977b5b89fdf2151c43c20e944d7ed0ca7e20f61d52e87424a10564fe3c rationale="Independent review of the first-leaf CR2 of BUG-260917-2fwf8e; operator routing: independent reviews on claude-opus-5 max (PR48) while producers stay on muse-spark max; codex Astra remains the fallback ceiling."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [reviewer] reviewer (claude) (run=RUN-260921-442ca9, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260921-442ca9)
REVIEW CR rev2 (RUN-260921-442ca9, claude-opus-5 max): CHANGES REQUESTED -> to-dev. Verdict: BUG-260917-2fwf8e_review-verdict-rev2.md; evidence: BUG-260917-2fwf8e_review-evidence-rev2.tar.gz (sha256 5f9b2c6d2c216149e76b75a1dc6daca405d29c849abde0c90e8eaaea55cecf4b). The code is accepted on its merits: every rev1 finding is closed in the tree (probe 10 fixed at Authorize/Decide/Run; 60-row ObserveFencing stale grid identical to trunk, 13.7 property 6/6; stale_fenced assertion restored; order pinned on 5/5 entries + axpane ModeLaunch; rev1 survivors RM-A/RM-H KILLED, RM-C killed inside internal/fencing; swap row APPLIED+KILLED x2; fencing harness 32/32 x2, terminstance harness 122/122 x2; full 30-command suite green on d1761767, race 432s at load 15). P2-1: the attached outcome resources BUG-260917-2fwf8e_results.md and _conformance-matrix.md (top-level AND inside the evidence archive) are the byte-identical rev1 files (md5 215e4f60 / e5b38b9d) that assert the rev1 regression rewrite and "no pre-existing per-test outcome changed"; the rev2 results/matrix were written to the control checkout .temp and never attached (resource update used the Story-worktree path holding 20:52 copies); the attached terminstance battery evidence is a 2-row targeted stdout, not the 122-row pass. P3-1 the relative-question arms in observeRemoteWinner are dead under the new fallback (RM-O5/RM-O6 SURVIVED 2/2 over 1,103 rows) while doc comment/TRACEABILITY still call them the mechanism; P3-2 outer fallback lost its harness row (still test-pinned: RM-O1 KILLED); P3-3 swap row mask does not execute the axpane ModeLaunch cell; P3-4 TRACEABILITY names nonexistent subtests, LOGBOOK misattributes the 180 MISSING_ACTIVITY lines to cataloggen; P3-5 pre-existing unpinned bound: empty LocalHostID (RM-W1 SURVIVED on trunk too). Rework = re-attach the rev2 results/matrix + full battery evidence and add the missing sections; no production change required.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260921-442ca9, pid=45154, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=codex/gpt-5.6-luna/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:c4605ab954a19c115501892eed18a53285befd74d0d65bb51e10eff2f33cc4f9 rationale="Rework of CR rev2 of BUG-260917-2fwf8e: no P1, the code is verified correct and complete; the P2 is that the attached board record is the rejected revision-1 file because resource update resolved .temp against the Story worktree instead of the control root. Mostly re-attachment plus three missing sections. Same luna max producer that owns the candidate."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (codex) (run=RUN-260921-21d6d2, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260921-21d6d2)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260921-21d6d2, pid=34135, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:dd9abb977b5b89fdf2151c43c20e944d7ed0ca7e20f61d52e87424a10564fe3c rationale="Independent review of the first-leaf CR3 of BUG-260917-2fwf8e; operator routing: independent reviews on claude-opus-5 max (PR48) while producers stay on muse-spark max; codex Astra remains the fallback ceiling."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [reviewer] reviewer (claude) (run=RUN-260921-a9f83c, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260921-a9f83c)
REVIEW CR rev3 (RUN-260921-a9f83c, claude-opus-5 max): ACCEPTED -> accept_cr. Verdict: BUG-260917-2fwf8e_review-verdict-rev3.md; evidence: BUG-260917-2fwf8e_review-evidence-rev3.tar.gz (sha256 21897ac2bdfdbaf523a3484e381740a06ef284085b0ace54207170259f075a45). Live worktree tree OID == candidate 6b7603cf before and after; origin/main == base 799c338 (no drift). Probe 10 reproduced verbatim at Decide: trunk refused/lease_conflict -> candidate attach_remote/remote_owner/not_owner; through Run with a durable remote takeover the offer/park is under the remote lease B with no event authored (1->1). 1,008-row Authorize grid: exactly 40 rows moved = {remote, empty LocalHostID} x lapsed x 4 token classes x 5 entries (lease_conflict -> remote_owner park / not_owner), all named by the results; badpolicy+lapsed still invalid_arguments; local lapsed lease_conflict x5; remote no-grant invalid_arguments (stated bound). 320-row ObserveFencing grid: 24 rows change only the surfaced error, 80/80 stale_fenced transitions identical, 13.7 property 8/8 on both trees. Per-test diff 1,073->1,103, 0 changed; cross-runs: trunk tests fail only at the rewritten expiry-first preconditions, stale_fenced assertion retained. Shipped harnesses x2: fencing 32/32 KILLED + controls; terminstance 122/122 + C-control SURVIVED, 123 raw blocks, blobs identical; swap row and new outer-fallback row APPLIED+KILLED x2. 19 reviewer plants x2 over the importer set: 13 KILLED + swap; survivors = RM3-I empty LocalHostID (pre-existing, trunk too), RM3-L relative arm (documented bound), RM3-N whole observeRemoteWinner dispatch (structural redundancy, P3 note). Rev2 P2-1 closed: board results/matrix are the rev3 files, byte-identical inside the archive, MANIFEST 1821/1821 verified. Full 30-command suite green on the exact tree (race 412s at load 11-17). P3 notes for the final leaf: decide.go doc comments still say lapsed grants refuse; observeRemoteWinner redundant as a whole; no-grant after-restore reachability is a product question for the grant-minting leaf; tree-identity.txt records the commit not the tree OID.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260921-a9f83c, pid=4050, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=codex/gpt-5.6-luna/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:c4605ab954a19c115501892eed18a53285befd74d0d65bb51e10eff2f33cc4f9 rationale="Producer-bound checkpoint run for the accepted CR rev3 of BUG-260917-2fwf8e: worktree checkpoint requires the same binding as the accepted revision, which this luna max producer holds. No product work."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (codex) (run=RUN-260921-5e5409, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260921-5e5409)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260921-5e5409, pid=68469, exit=0)

## Precondition Resources
- [BUG-260917-2fwf8e_producer.md](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_producer.md)
- [BUG-260917-2fwf8e_reviewer-cr1.md](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_reviewer-cr1.md)
- [BUG-260917-2fwf8e_rework-rev2.md](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_rework-rev2.md)
- [BUG-260917-2fwf8e_reviewer-cr2.md](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_reviewer-cr2.md)
- [BUG-260917-2fwf8e_rework-rev3.md](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_rework-rev3.md)
- [BUG-260917-2fwf8e_reviewer-cr3.md](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_reviewer-cr3.md)

## Outcome Resources
- [BUG-260917-2fwf8e_spawn-log_-implementer--developer--codex-_RUN-260921-2649f6.log](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_spawn-log_-implementer--developer--codex-_RUN-260921-2649f6.log) — System spawn log captured by task-board
- [BUG-260917-2fwf8e_results.md](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_results.md)
- [BUG-260917-2fwf8e_conformance-matrix.md](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_conformance-matrix.md)
- [BUG-260917-2fwf8e_producer-evidence.tar.gz](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_producer-evidence.tar.gz)
- [BUG-260917-2fwf8e_change-request_rev1.patch](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_change-request_rev1.patch) — Change Request CR-BUG-260917-2fwf8e-1 revision 1 candidate patch (repository_delta=present, 9 changed paths)
- [BUG-260917-2fwf8e_change-request_rev1-validation.log](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_change-request_rev1-validation.log) — Change Request CR-BUG-260917-2fwf8e-1 revision 1 bounded validation log
- [BUG-260917-2fwf8e_spawn-log_-reviewer--reviewer--claude-_RUN-260921-766ad0.log](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_spawn-log_-reviewer--reviewer--claude-_RUN-260921-766ad0.log) — System spawn log captured by task-board
- [BUG-260917-2fwf8e_review-verdict-rev1.md](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_review-verdict-rev1.md) — Reviewer verdict CR rev1: CHANGES REQUESTED (P1 stale-fencing regression under remote winner + lapsed grant; P2 one-of-five entries pinned; P2 unnamed moved inputs)
- [BUG-260917-2fwf8e_review-evidence-rev1.tar.gz](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_review-evidence-rev1.tar.gz) — Reviewer evidence rev1: probes, grids, cross-run, property test, harness x2, reviewer mutants x2, full suite logs, manifest
- [BUG-260917-2fwf8e_spawn-log_-implementer--developer--codex-_RUN-260921-fcea9e.log](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_spawn-log_-implementer--developer--codex-_RUN-260921-fcea9e.log) — System spawn log captured by task-board
- [BUG-260917-2fwf8e_change-request_rev2.patch](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_change-request_rev2.patch) — Change Request CR-BUG-260917-2fwf8e-2 revision 2 candidate patch (repository_delta=present, 11 changed paths)
- [BUG-260917-2fwf8e_change-request_rev2-validation.log](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_change-request_rev2-validation.log) — Change Request CR-BUG-260917-2fwf8e-2 revision 2 bounded validation log
- [BUG-260917-2fwf8e_spawn-log_-reviewer--reviewer--claude-_RUN-260921-442ca9.log](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_spawn-log_-reviewer--reviewer--claude-_RUN-260921-442ca9.log) — System spawn log captured by task-board
- [BUG-260917-2fwf8e_review-verdict-rev2.md](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_review-verdict-rev2.md) — Review verdict for CR revision 2 (RUN-260921-442ca9): CHANGES REQUESTED, P2 evidence record + 5 P3
- [BUG-260917-2fwf8e_review-evidence-rev2.tar.gz](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_review-evidence-rev2.tar.gz) — Reviewer evidence for CR revision 2: grids, cross-runs, two harness passes each, reviewer plants, full configured suite
- [BUG-260917-2fwf8e_spawn-log_-implementer--developer--codex-_RUN-260921-21d6d2.log](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_spawn-log_-implementer--developer--codex-_RUN-260921-21d6d2.log) — System spawn log captured by task-board
- [BUG-260917-2fwf8e_rev3-evidence.tar.gz](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_rev3-evidence.tar.gz) — Revision 3 producer evidence
- [BUG-260917-2fwf8e_change-request_rev3.patch](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_change-request_rev3.patch) — Change Request CR-BUG-260917-2fwf8e-3 revision 3 candidate patch (repository_delta=present, 11 changed paths)
- [BUG-260917-2fwf8e_change-request_rev3-validation.log](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_change-request_rev3-validation.log) — Change Request CR-BUG-260917-2fwf8e-3 revision 3 bounded validation log
- [BUG-260917-2fwf8e_spawn-log_-reviewer--reviewer--claude-_RUN-260921-a9f83c.log](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_spawn-log_-reviewer--reviewer--claude-_RUN-260921-a9f83c.log) — System spawn log captured by task-board
- [BUG-260917-2fwf8e_review-verdict-rev3.md](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_review-verdict-rev3.md) — Review verdict for CR revision 3 (RUN-260921-a9f83c): ACCEPTED; probe 10 + neighbours reproduced on both trees at Authorize/Decide/Run, 1,008-row Authorize grid, 320-row ObserveFencing grid, per-test + cross-run diffs, both shipped harnesses x2, 19 reviewer plants x2, 30/30 configured suite; 4 P3 notes for the final leaf
- [BUG-260917-2fwf8e_review-evidence-rev3.tar.gz](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_review-evidence-rev3.tar.gz) — Reviewer evidence for CR revision 3 (RUN-260921-a9f83c): probes, grids, per-test and cross-run diffs, shipped harness passes x2, reviewer plants x2 with per-plant logs and diffs, full configured suite logs, manifest
- [BUG-260917-2fwf8e_spawn-log_-implementer--developer--codex-_RUN-260921-5e5409.log](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_spawn-log_-implementer--developer--codex-_RUN-260921-5e5409.log) — System spawn log captured by task-board
- [BUG-260917-2fwf8e_checkpoint-rev3.md](file://BUG-260917-2fwf8e/BUG-260917-2fwf8e_checkpoint-rev3.md) — Checkpoint evidence for accepted CR revision 3

## Created
2026-09-17T18:42:11Z

## Last Update
2026-09-22T03:01:20Z

## Assigned To
[implementer] developer (codex)
