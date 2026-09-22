## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(5))

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] An append under a superseded lease is REFUSED through the production entry even when the chain tail still sits on that lease, driven by a committed test and killed by a narrowing mutant on the admission arm
- [x] A profile.changed event appended under a losing lease can never become the effective profile source: probes 9 and 15 from TASK-260830-1geqhj_review-evidence-rev1.tar.gz are reproduced on trunk before the fix and refused after, with the yolo/--dangerously-bypass-approvals-and-sandbox outcome named in the before state
- [x] A gate x entry census is committed as a table in the conformance matrix and mirrored in TRACEABILITY: every entry that reaches the append-admission gate gets a named test plus an admitting narrowing row, or unreachable with a reason, or a stated bound with an owner; any gate implemented on two or more sides reports a row-count symmetry line
- [x] Composing writers keep working unchanged, proven by an OUTCOME-keyed comparison over the complete importer set of internal/sessrepo (not a name-keyed per-test diff of one package), with every input that moved across an admission or refusal arm named in the results
- [x] Story-final registry work: internal/traceability ownership.v0.7.0.json bindings added for this Story, reviewedOwnershipCanonicalSHA256 RE-DERIVED from the registry as it stands in the candidate, tracecheck green on the exact tree with the ratios quoted verbatim, and the README measured-coverage subsection equal to what tracecheck prints
- [x] The four P3 notes carried forward from the BUG-260917-2fwf8e rev3 verdict are each dispositioned in the results: the axpane doc-comment qualifier, the observeRemoteWinner structural redundancy (record only), the no-grant step-4 product question (record as unresolved, do not decide), and the evidence-hygiene items (record the candidate TREE OID, and state the test mask beside any row count)
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
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=codex/gpt-5.6-luna/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:c4605ab954a19c115501892eed18a53285befd74d0d65bb51e10eff2f33cc4f9 rationale="Story-final leaf of STORY-260917-kf9g1h: sessrepo admits an append under a superseded lease while the tail still sits on it, which lets a losing-lease profile.changed drive the effective profile to a sandbox-bypassing one against SPEC 2.4. Carries the registry bindings and the digest re-pin. Same luna max producer that closed the sibling leaf in three revisions and holds the Story context."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (codex) (run=RUN-260921-d970d9, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260921-d970d9)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260921-d970d9, pid=80126, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:dd9abb977b5b89fdf2151c43c20e944d7ed0ca7e20f61d52e87424a10564fe3c rationale="Independent review of the story_final CR1 of BUG-260917-3lddu0; operator routing: independent reviews on claude-opus-5 max (PR48) while producers stay on muse-spark max; codex Astra remains the fallback ceiling."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [reviewer] reviewer (claude) (run=RUN-260921-f25edb, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260921-f25edb)
Review verdict CR rev1 (RUN-260921-f25edb, claude-opus-5 max): CHANGES REQUESTED -> to-dev. P1-1 the SPEC 2.4 losing-lease profile-source property has no owner independent of the append gate (registry binds sessprofile Derive; TestLosingLeaseProfileEventIgnored fails at its AppendEvent assertion under a weakened gate; no mutant runs it). P1-2 four landed axpane tests (TestRunPostWindowSupersedes, TestRunSupersededPairIdenticalRetry, TestRunCreateFromStoppedPostWindow re-sequenced; TestLosingLeaseProfileEventIgnored rewritten) unargued while results/matrix claim 0 changed / no moved inputs; checkpoint test files vs candidate production fail x4 with ErrStaleLease. P2-1 shipped narrowings are `&& leaseID == never-a-real-lease` arm-deletes; P2-2 same-epoch preservation unmeasured (reviewer plant R6 SURVIVED x2); P2-3 producer-evidence tarball never attached to the board. P3 x5 (one-direction same-epoch pins per entry, never-minted higher-epoch class admitted pre-existing/bound without owner, Run->EmitParked census cell, empty-lease-store bound without owner, carry-forward 4/4 recorded). Verified: probes 9/15 reproduced on trunk (yolo / --dangerously-bypass-approvals-and-sandbox at the session-head derivation) and refused on the candidate at all four entries; composing writers work under the winner; registry digest re-derived equal (ed6f016f...); tracecheck verbatim green; README pin reddens on plants; shipped harness x2 KILLED; 9 reviewer plants x2; 30/30 configured suite + Windows vet exit 0 on tree 8042bcc. Evidence: BUG-260917-3lddu0_review-verdict-rev1.md, BUG-260917-3lddu0_review-evidence-rev1.tar.gz.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260921-f25edb, pid=83608, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=codex/gpt-5.6-luna/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:98722a691d38a25da073ea2a70dcf2a8e0f60a1e644f210baa284673524fc25a rationale="Rework of CR rev1 of BUG-260917-3lddu0 (story_final): production is correct and reproduced by the reviewer; two P1 are evidence-record defects (four landed tests rewritten unargued while the results claim 0 changed; the 2.4 property has no owner independent of the append gate yet the registry binds sessprofile.Derive for it), three P2, five P3. Same luna max producer that owns the rev1 candidate."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (codex) (run=RUN-260921-9c082b, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260921-9c082b)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260921-9c082b, pid=85013, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=codex/gpt-6-astra/low pair_source=explicit match=recommended_rank_1 snapshot=sha256:35218104f1d47040bffdc70eeebc875fb7e6f581444aaf03c2c07a638b793604 rationale="Independent review of the story_final CR2 of BUG-260917-3lddu0; operator routing of 2026-09-22 moved independent reviews to codex gpt-6-astra low while producers stay on their leaf binding."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [reviewer] reviewer (codex) (run=RUN-260922-5edb17, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260922-5edb17)
Review rev2 RUN-260922-5edb17: CHANGES REQUESTED. Exact candidate tree afb776578e26979c7c2da474116dfc1c95f8eafa. Production AC 2/2 driven and gate-entry census 8/10 measured (2 bounded); all 30 configured checks plus Windows vet passed. P2-1: new profile-source case has zero ownership/clause references despite README claiming 2.4#2. P2-2: permitted route (b) still lacks the required tracked owner element for independent derivation-side work (only a function path is named). P3: producer PASS grid is name-keyed; reviewer actual runtime outcome comparison is attached. Verdict, gzip archive and runtime grid were read back from the board and verified byte-for-byte. Story registry checklist item 5 reopened. Rework is limited to registry/docs/evidence and the named bound owner; no append algorithm rewrite or Story commit requested. See BUG-260917-3lddu0_review-verdict-rev2.md and BUG-260917-3lddu0_review-evidence-rev2.tar.gz.
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260922-5edb17, pid=57144, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=codex/gpt-5.6-luna/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:98722a691d38a25da073ea2a70dcf2a8e0f60a1e644f210baa284673524fc25a rationale="Rework of CR rev2 of BUG-260917-3lddu0 (story_final): no P1, production accepted, 2 of 2 AC rows driven. Two P2 are the registry clause edge and a named owner element (created as STORY-260922-cpkajd / TASK-260922-31qyyi), plus one grid relabel. Same luna max producer that owns the candidate."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (codex) (run=RUN-260922-128d8a, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260922-128d8a)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260922-128d8a, pid=80290, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=codex/gpt-6-astra/low pair_source=explicit match=recommended_rank_1 snapshot=sha256:35218104f1d47040bffdc70eeebc875fb7e6f581444aaf03c2c07a638b793604 rationale="Independent review of the story_final CR3 of BUG-260917-3lddu0; operator routing of 2026-09-22 puts independent reviews on codex gpt-6-astra low."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [reviewer] reviewer (codex) (run=RUN-260922-573ca5, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260922-573ca5)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260922-573ca5, pid=22112, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=codex/gpt-5.6-luna/max pair_source=explicit match=recommended_rank_1 snapshot=sha256:98722a691d38a25da073ea2a70dcf2a8e0f60a1e644f210baa284673524fc25a rationale="Bound integration run for the accepted story_final CR rev3 of BUG-260917-3lddu0 (final leaf of STORY-260917-kf9g1h): integrate, full suite at the new head, delivery branch and PR. Same producer binding as the accepted revision."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [implementer] developer (codex) (run=RUN-260922-90cb74, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260922-90cb74)

## Precondition Resources
- [BUG-260917-3lddu0_producer.md](file://BUG-260917-3lddu0/BUG-260917-3lddu0_producer.md)
- [BUG-260917-3lddu0_reviewer-cr1.md](file://BUG-260917-3lddu0/BUG-260917-3lddu0_reviewer-cr1.md)
- [BUG-260917-3lddu0_rework-rev2.md](file://BUG-260917-3lddu0/BUG-260917-3lddu0_rework-rev2.md)
- [BUG-260917-3lddu0_reviewer-cr2.md](file://BUG-260917-3lddu0/BUG-260917-3lddu0_reviewer-cr2.md)
- [BUG-260917-3lddu0_rework-rev3.md](file://BUG-260917-3lddu0/BUG-260917-3lddu0_rework-rev3.md)
- [BUG-260917-3lddu0_reviewer-cr3.md](file://BUG-260917-3lddu0/BUG-260917-3lddu0_reviewer-cr3.md)
- [BUG-260917-3lddu0_integration-rev3.md](file://BUG-260917-3lddu0/BUG-260917-3lddu0_integration-rev3.md)

## Outcome Resources
- [BUG-260917-3lddu0_spawn-log_-implementer--developer--codex-_RUN-260921-d970d9.log](file://BUG-260917-3lddu0/BUG-260917-3lddu0_spawn-log_-implementer--developer--codex-_RUN-260921-d970d9.log) — System spawn log captured by task-board
- [BUG-260917-3lddu0_results.md](file://BUG-260917-3lddu0/BUG-260917-3lddu0_results.md) — Rev3 handoff evidence: production stale-lease gate, exact 30-command validation, two-pass narrowing mutants, importer runtime outcomes, registry binding and traceability.
- [BUG-260917-3lddu0_conformance-matrix.md](file://BUG-260917-3lddu0/BUG-260917-3lddu0_conformance-matrix.md) — Rev3 gate-by-entry census, narrowing mutants, semantic moved-input table, complete importer mask, runtime outcome grid and exact-tree traceability.
- [BUG-260917-3lddu0_change-request_rev1.patch](file://BUG-260917-3lddu0/BUG-260917-3lddu0_change-request_rev1.patch) — Change Request CR-BUG-260917-3lddu0-1 revision 1 candidate patch (repository_delta=present, 26 changed paths)
- [BUG-260917-3lddu0_change-request_rev1-validation.log](file://BUG-260917-3lddu0/BUG-260917-3lddu0_change-request_rev1-validation.log) — Change Request CR-BUG-260917-3lddu0-1 revision 1 bounded validation log
- [BUG-260917-3lddu0_spawn-log_-reviewer--reviewer--claude-_RUN-260921-f25edb.log](file://BUG-260917-3lddu0/BUG-260917-3lddu0_spawn-log_-reviewer--reviewer--claude-_RUN-260921-f25edb.log) — System spawn log captured by task-board
- [BUG-260917-3lddu0_review-verdict-rev1.md](file://BUG-260917-3lddu0/BUG-260917-3lddu0_review-verdict-rev1.md) — Independent review verdict for CR revision 1 (RUN-260921-f25edb): CHANGES REQUESTED — P1-1 §2.4 profile-source property has no owner independent of the append gate (registry binds Derive; one test set covers both; no mutant for AC row 2), P1-2 four landed tests re-sequenced/rewritten unargued while results claim 0 moved inputs; P2 delete-shaped shipped mutants, same-epoch preservation unmeasured (R6 SURVIVED x2), producer evidence archive not attached; 5 P3; probes 9/15 reproduced both trees, 30/30 suite green on the exact tree
- [BUG-260917-3lddu0_review-evidence-rev1.tar.gz](file://BUG-260917-3lddu0/BUG-260917-3lddu0_review-evidence-rev1.tar.gz) — Reviewer evidence for CR revision 1 (RUN-260921-f25edb): adapted probes 9/15 on trunk and candidate, property-2 independence instrument, composing-writer and idempotency probes, old-tests-vs-candidate-production run, outcome grids (trunk/checkpoint/candidate), shipped harness x2, 9 reviewer plants x2 + control with raw per-plant logs and exits, registry digest re-derivation, README pin plants, tracecheck, hygiene, full 30-command suite logs, sha256 manifest
- [BUG-260917-3lddu0_spawn-log_-implementer--developer--codex-_RUN-260921-9c082b.log](file://BUG-260917-3lddu0/BUG-260917-3lddu0_spawn-log_-implementer--developer--codex-_RUN-260921-9c082b.log) — System spawn log captured by task-board
- [BUG-260917-3lddu0_producer-evidence.tar.gz](file://BUG-260917-3lddu0/BUG-260917-3lddu0_producer-evidence.tar.gz) — Producer evidence archive required by review; attached and read back for byte comparison.
- [BUG-260917-3lddu0_rev2-evidence.tar.gz](file://BUG-260917-3lddu0/BUG-260917-3lddu0_rev2-evidence.tar.gz) — Rev2 validation, importer OUTCOME comparison, two-pass narrowing mutants, baseline probes, results and conformance matrix.
- [BUG-260917-3lddu0_change-request_rev2.patch](file://BUG-260917-3lddu0/BUG-260917-3lddu0_change-request_rev2.patch) — Change Request CR-BUG-260917-3lddu0-2 revision 2 candidate patch (repository_delta=present, 28 changed paths)
- [BUG-260917-3lddu0_change-request_rev2-validation.log](file://BUG-260917-3lddu0/BUG-260917-3lddu0_change-request_rev2-validation.log) — Change Request CR-BUG-260917-3lddu0-2 revision 2 bounded validation log
- [BUG-260917-3lddu0_spawn-log_-reviewer--reviewer--codex-_RUN-260922-5edb17.log](file://BUG-260917-3lddu0/BUG-260917-3lddu0_spawn-log_-reviewer--reviewer--codex-_RUN-260922-5edb17.log) — System spawn log captured by task-board
- [BUG-260917-3lddu0_review-runtime-outcomes-rev2.md](file://BUG-260917-3lddu0/BUG-260917-3lddu0_review-runtime-outcomes-rev2.md) — Independent runtime AppendEvent outcome comparison over all nine sessrepo importer packages on trunk and exact candidate.
- [BUG-260917-3lddu0_review-verdict-rev2.md](file://BUG-260917-3lddu0/BUG-260917-3lddu0_review-verdict-rev2.md) — Independent rev2 verdict: CHANGES REQUESTED; production AC 2/2 and census 8/10 pass, two P2 evidence/ownership issues, 30/30 configured checks green.
- [BUG-260917-3lddu0_review-evidence-rev2.tar.gz](file://BUG-260917-3lddu0/BUG-260917-3lddu0_review-evidence-rev2.tar.gz) — Independent rev2 evidence: probes 9/15, actual importer outcome grids, shipped mutants x2, reviewer narrowings x2, registry/readme attacks, complete 30-command validation, verified SHA256 manifest.
- [BUG-260917-3lddu0_spawn-log_-implementer--developer--codex-_RUN-260922-128d8a.log](file://BUG-260917-3lddu0/BUG-260917-3lddu0_spawn-log_-implementer--developer--codex-_RUN-260922-128d8a.log) — System spawn log captured by task-board
- [BUG-260917-3lddu0_rev3-evidence.tar.gz](file://BUG-260917-3lddu0/BUG-260917-3lddu0_rev3-evidence.tar.gz) — Rev3 evidence archive: exact configured validation logs, runtime importer comparison, baseline probes, and two-pass narrowing-mutant raw logs.
- [BUG-260917-3lddu0_change-request_rev3.patch](file://BUG-260917-3lddu0/BUG-260917-3lddu0_change-request_rev3.patch) — Change Request CR-BUG-260917-3lddu0-3 revision 3 candidate patch (repository_delta=present, 28 changed paths)
- [BUG-260917-3lddu0_change-request_rev3-validation.log](file://BUG-260917-3lddu0/BUG-260917-3lddu0_change-request_rev3-validation.log) — Change Request CR-BUG-260917-3lddu0-3 revision 3 bounded validation log
- [BUG-260917-3lddu0_spawn-log_-reviewer--reviewer--codex-_RUN-260922-573ca5.log](file://BUG-260917-3lddu0/BUG-260917-3lddu0_spawn-log_-reviewer--reviewer--codex-_RUN-260922-573ca5.log) — System spawn log captured by task-board
- [BUG-260917-3lddu0_review-verdict-rev3.md](file://BUG-260917-3lddu0/BUG-260917-3lddu0_review-verdict-rev3.md) — Independent rev3 review: accepted scoped route (b), 2/2 AC, 8/10 gate cells, all 30 checks green; explicit derivation follow-up bound.
- [BUG-260917-3lddu0_review-evidence-rev3.tar.gz](file://BUG-260917-3lddu0/BUG-260917-3lddu0_review-evidence-rev3.tar.gz) — Independent exact-tree evidence: original probes reproduced, runtime importer grids, shipped and reviewer mutants twice, registry re-derivation, full validation, 176-entry SHA256 manifest.
- [BUG-260917-3lddu0_spawn-log_-implementer--developer--codex-_RUN-260922-90cb74.log](file://BUG-260917-3lddu0/BUG-260917-3lddu0_spawn-log_-implementer--developer--codex-_RUN-260922-90cb74.log) — System spawn log captured by task-board

## Created
2026-09-17T18:42:12Z

## Last Update
2026-09-22T03:01:20Z

## Assigned To
[implementer] developer (codex)
