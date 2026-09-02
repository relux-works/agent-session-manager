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
- [x] Entry-local overlap is refused at both public identity entries: an entry whose ancestor is a file, symlink or hardlink is rejected
- [x] Negative cases cover file-over-file, symlink-over-file and file-over-directory, each driving the production identity entries
- [x] Each case reddens when only the overlap clause is weakened, proven by mutant
- [x] The sorted order the validator already computes is reused rather than re-derived
- [x] The constraint-enumeration row states the whole rule from SPEC.md:4768-4769 rather than the external child-partition half
- [x] Repository tests, vet, build, tracecheck, catalog check and the seeded fuzz targets exit 0 with no coverage regression
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
FRONT-LOADED DoD, distilled from twenty-plus review rounds across the Stories that landed today. These are the criteria the reviewer will apply.

1. THIS IS AN ABSENT CLAUSE, not a weak one. No mutation sweep can find it, because there is nothing to mutate - that is precisely why it survived on main while the enumeration artifact read as covered. Write the clause, then pin it.

2. NEVER INVENT WHAT YOU ADD. SPEC.md:4768-4769 requires entries to contain no duplicate, OVERLAPPING, or destination-case-colliding path. Implement exactly that. Do not extend it to cases the spec does not name, and quote the declaring line verbatim in the enumeration row - a quote that is not literally in SPEC.md at 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c is itself a defect.

3. FIX THE ARTIFACT TOO. testdata/constraint-enumeration.md currently narrows overlapping to the external child-partition case, so the row reads as enforced while the entry-local half is absent. Correct the row to the whole rule.

4. A NEGATIVE CASE MUST BE REFUSED BY THE CLAUSE IT NAMES. Build fixtures where nothing else could refuse them: the sort and case-fold checks already present must PASS on your overlap cases, so the only thing that can reject them is the new clause. A case that also trips an earlier check pins nothing - that mistake shipped twice today.

5. BOUNDS AND ADJACENCY. If you introduce any bound, prove it at its limit and one step past, never two.

6. DRIVE THE REAL PRODUCTION ENTRY POINTS, CalculateObjectIdentity and VerifyObjectIdentity, not the validator directly.

The probe that found this, for your fixtures: file a plus file a/b accepted; symlink s plus file s/x accepted; file a plus directory a/b accepted. Only a case-fold duplicate was refused. A symlink parent over a real child is the reason this matters.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 recommendation: write an absent clause no mutation sweep could have found, closing a materialization-escape primitive on landed main."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260902-c0cf4e, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260902-c0cf4e)
Entry-local Transfer Manifest path overlap is now refused in validateManifestEntries and reaches both public identity entries (CalculateObjectIdentity, VerifyObjectIdentity). Commit b16266a on task-board/story/STORY-260902-1evi33, one signed commit past checkpoint a031128.

Fix: the loop already proves entries strictly bytewise sorted, so every ancestor is a proper prefix ending at / and the open non-directory owners form a stack. An entry whose ancestor is a declared file, symlink or hardlink is refused; the sorted order is reused rather than re-deriving each path ancestor set. The non-obvious half is the closing rule: an owner is closed only once the scan passes strictly above owner + "/", because . (0x2E) sorts below / (0x2F) and a.txt sits between a and a/b. Closing on the first non-descendant loses the a owner before its escaping child arrives; mutant M5 is that implementation and dies to a dedicated case.

Negative evidence: TestManifestEntryOverlapRefusalReachesBothIdentityEntries pins six cases (file-over-file, symlink-over-file, file-over-directory, hardlink-over-file, intervening .-sorted sibling, nested owner), each with its own exact refusal message so a neighbouring refusal cannot stand in. TestManifestEntryOverlapAdmitsDirectoryParents keeps directory parents and shared byte prefixes admitted so the clause cannot be satisfied by refusing every nested manifest.

Mutants: 7 generated, 7 killed, 0 survivors, acceptance cases green throughout. M2 (only file owners refuse) and M7 (only non-file owners refuse) together prove no tag arm is redundant; M3/M4 prove the descendant side is unconstrained; M5 proves the closing rule; M1/M6 are delete and narrow-to-duplicate.

Artifact: constraint-enumeration.md:415 now quotes the whole SPEC.md:4768-4769 sentence, states that all three properties are entry-local, names production enforcement and both pinning tests, and leaves only the child-partition closure external. README states the same enforcement claim.

Declined over-refusal: parent directories are NOT required to be declared. The spec states no such rule and the shipped fixture already declares p/00000 with no p entry. Overlap is compared bytewise, not case-folded, since folding does not preserve byte order and the stack cannot run over folded paths; that narrowing is disclosed in the row.

Gates, each a standalone process with its real exit code recorded in BUG-260902-2luo5h_gates.log: go build ./... 0, go vet ./... 0, gofmt -l . 0, go test ./... -count=1 0, go test ./... -cover 0, five seeded fuzz targets 100x 0, tracecheck 0, cataloggen -check 0. gofmt/test/tracecheck/cataloggen were re-run after the README and LOGBOOK edits and exit 0. internal/canonicaljson coverage 97.2% at checkpoint a031128 (measured in a throwaway worktree, not inherited) -> 97.2%. No fuzz corpus file was written.

Artifacts: BUG-260902-2luo5h_results.md, BUG-260902-2luo5h_overlap-mutants.log, BUG-260902-2luo5h_gates.log. Logbook entry 2026-09-02 1900.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-c0cf4e, pid=44702, exit=0)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Verify the new clause is what refuses the overlap cases rather than the sort or case-fold checks, and that the enumeration quote is literally present in the pinned SPEC."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260902-729ab0, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260902-729ab0)
REVIEW ACCEPTED (RUN-260902-729ab0, CR rev1, tree 716683f). Verified independently, not read: validateManifestEntries is reached only from validateTransferManifest (closed_shapes.go:948, registered validator at :65) and runs for all five kind arms; the six new refusal subtests FAIL against a pristine base tree a031128 while the acceptance test already passed there. 8 reviewer-authored mutants, 8 killed, no survivors - six of them narrowings, not deletions: owners restricted to file kills symlink+hardlink cases, owners restricted to non-file kills the file cases (no entry-type arm is redundant); close-on-first-non-descendant and single-pop both die on the intervening a.txt sibling; ancestor-match-without-separator and directories-also-own-subtrees die on the acceptance cases, so the clause cannot be passed by a validator that refuses every nested manifest. Order-reuse algorithm proven correct from the strict bytewise sort plus ParseRelativePath segment rules, amortized one push/pop per entry. Nine extra shapes probed through CalculateObjectIdentity all refuse with the correct owner named. Gates rerun by the reviewer: build, vet, gofmt, go test ./... -count=1 -cover (9/9 ok), tracecheck (acceptance_cases=38), cataloggen -check, all five seeded fuzz targets, each rc=0; canonicaljson coverage 97.2% at candidate and 97.2% measured independently at base, no regression. Declined declared-parent-directory requirement is correct: SPEC 13.14/10.4 impose no such rule. RESIDUAL, disclosed by the producer and not grounds to reject: overlap is compared bytewise, so case-fold overlaps are still ACCEPTED (symlink S over file s/x, file A over file a/b, file s over file S/x, file k over U+212A/x) - on a case-insensitive destination that is the same escape primitive; recommend a follow-up Bug. Note also that the corrected constraint-enumeration narrative row is not machine-checked - constraint_inventory_test.go parses only the requireExactMembers member rows - which is structurally why a half-quoted row survived review. Evidence: BUG-260902-2luo5h_review-verdict.md, BUG-260902-2luo5h_review-attack.log.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-729ab0, pid=60544, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [BUG-260902-2luo5h_spawn-log_-implementer--developer--claude-_RUN-260902-c0cf4e.log](file://BUG-260902-2luo5h/BUG-260902-2luo5h_spawn-log_-implementer--developer--claude-_RUN-260902-c0cf4e.log) — System spawn log captured by task-board
- [BUG-260902-2luo5h_results.md](file://BUG-260902-2luo5h/BUG-260902-2luo5h_results.md)
- [BUG-260902-2luo5h_overlap-mutants.log](file://BUG-260902-2luo5h/BUG-260902-2luo5h_overlap-mutants.log) — Seven overlap-clause mutants with per-subtest verdicts; 7 killed, 0 survivors
- [BUG-260902-2luo5h_gates.log](file://BUG-260902-2luo5h/BUG-260902-2luo5h_gates.log)
- [BUG-260902-2luo5h_change-request_rev1.patch](file://BUG-260902-2luo5h/BUG-260902-2luo5h_change-request_rev1.patch) — Change Request CR-BUG-260902-2luo5h-1 revision 1 candidate patch (repository_delta=present, 6 changed paths)
- [BUG-260902-2luo5h_change-request_rev1-validation.log](file://BUG-260902-2luo5h/BUG-260902-2luo5h_change-request_rev1-validation.log) — Change Request CR-BUG-260902-2luo5h-1 revision 1 bounded validation log
- [BUG-260902-2luo5h_spawn-log_-reviewer--reviewer--claude-_RUN-260902-729ab0.log](file://BUG-260902-2luo5h/BUG-260902-2luo5h_spawn-log_-reviewer--reviewer--claude-_RUN-260902-729ab0.log) — System spawn log captured by task-board
- [BUG-260902-2luo5h_review-verdict.md](file://BUG-260902-2luo5h/BUG-260902-2luo5h_review-verdict.md) — Reviewer verdict: ACCEPTED, with 8/8 reviewer mutants killed, base-tree reproduction, literal SPEC quote check, production-entry probes and all gates rerun
- [BUG-260902-2luo5h_review-attack.log](file://BUG-260902-2luo5h/BUG-260902-2luo5h_review-attack.log) — Reviewer independent attack log: mutants, base-tree reproduction, residual case-fold overlap probes, rerun gate output

## Created
2026-09-02T11:59:01Z

## Last Update
2026-09-02T15:17:21Z

## Assigned To
[reviewer] reviewer (claude)
