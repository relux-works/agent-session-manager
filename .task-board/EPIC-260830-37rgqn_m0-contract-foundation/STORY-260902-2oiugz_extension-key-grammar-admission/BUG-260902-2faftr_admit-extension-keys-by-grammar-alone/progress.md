## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(2))

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] The invented rule and both call sites are removed; admission decides by reverse-DNS grammar and the depth bound alone
- [x] Legitimate reverse-DNS keys carrying env, auth, environment, endpoint, token, secret, password and credential labels are admitted and round-trip through the production Load entry
- [x] The tests that pinned the invented rule are removed with it, not left asserting it
- [x] Re-adding the blacklist as a narrowing mutant reddens the suite
- [x] Repository tests, vet, build, tracecheck and catalog check exit 0 with no coverage regression
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
REAPPLY the attached precondition patch; do not reimplement. It was produced AND reviewed to acceptance on a previous element that had to be discarded for board-bookkeeping reasons unrelated to the work.

What the accepted review established, worth preserving through the reapply: the removed rule does not merely lack a declaration, it CONTRADICTS the pinned contract, because SPEC.md:2344-2345 permits a configuration field to name a credential profile. The reviewer also read SPEC.md:350-354, which no narrative mentioned, and confirmed removing the rule loses nothing else. And it verified the removal is PINNED rather than merely done: re-adding the blacklist as a narrowing mutant reddened 17 of 20 admission subtests. Keep that mutant-provable property.

Reapply onto the current trunk, rerun the gates, and report any diff beyond trivial rebase context. If reapplying changes behaviour anywhere, stop and say so rather than absorbing it.

One thing to fix in passing, since the earlier producer diagnosed it: keep the whole leaf to ONE signed commit. The previous attempt made two and the Change Request contract admits exactly one direct single-parent commit past the checkpoint.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Reapplying accepted and reviewed work on a clean single-leaf Story with no move history, after a deleted parent broke the previous element's provenance chain."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260902-0ca406, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260902-0ca406)
Reapplied the accepted BUG-260902-3ru6vw patch; NOT a clean apply. BUG-260902-beqfwr had landed since and removed parseSSHConfigOption, so `git apply --check` on the precondition patch exits 1: the helper-removal hunk in internal/config/validation.go and the README insertion point were relocated by hand. Resulting code/test deltas are identical to the accepted patch.

hasForbiddenConfigName and both call sites are gone. validateExtensions decides by `len(key) > 253 || !reverseDNSPattern.MatchString(key)` alone; validateExtensionValue map arm enforces only depth >= 4. The two refusal_test.go cases that pinned the invented rule were removed with it. New extension_key_admission_test.go: 45 subtests through loadConfigDocument -> exported Load (schema_test.go:622), the production call site.

Gates on this tree, real exit codes, all 0: go build ./..., go vet ./..., gofmt -l ./internal, go test ./... -count=1, go test ./... -cover -count=1, full tracecheck, assigned-scope tracecheck (6.1 6.2 6.3 6.4 6.5 17.1 17.2 17.4), cataloggen -check. internal/config coverage 93.9% before and after (baseline from a git archive HEAD copy) — no regression.

Seven single-clause mutants each exited 1 and the tree was restored green after each. The two required narrowing mutants: re-added key blacklist reddens 18 cases, re-added nested-key blacklist reddens 10. Widening mutants: dropped grammar gate 14, widened 253-byte bound 4, widened depth bound 4, dropped registered backend-settings schema call 4, dropped DisallowUnknownFields 8.

One signed commit 4719092 on task-board/story/STORY-260902-2oiugz (forked from 67aed0b, 0 behind main). Not run: the scalar/canonicaljson fuzz targets — outside this delta. Evidence: BUG-260902-2faftr_results.md, BUG-260902-2faftr_gate-logs.tar.gz.
CORRECTION to my earlier note: the mutant counts I first gave counted parent-test lines too. Subtest-only counts: re-added key blacklist reddens 17 of the 20 admission subtests — exactly reproducing the accepted review's 17-of-20 measurement; re-added nested-key blacklist 10; dropped grammar gate 11; widened 253-byte bound 2; widened depth bound 2; dropped backend-settings schema call 2; dropped DisallowUnknownFields 5. The three surviving admission subtests are the grammar edges a.b, a1.b-c and z{63}.b, which carry no blacklisted label. Corrected in LOGBOOK.md and BUG-260902-2faftr_results.md.

Behaviour is unchanged from the accepted patch — nothing was absorbed. Single leaf commit as required: 1a94718, signed and verified, exactly one commit past checkpoint 67aed0b.

Reapply was NOT clean: BUG-260902-beqfwr had landed and removed parseSSHConfigOption, so git apply --check on the precondition patch exits 1. The helper-removal hunk in internal/config/validation.go and the README insertion point were relocated by hand; code and test deltas are otherwise identical to the accepted patch.

Final gates re-run on the committed tree, all exit 0: go build ./..., go vet ./..., gofmt -l ./internal, go test ./... -count=1, full tracecheck, assigned-scope tracecheck, cataloggen -check. internal/config coverage 93.9% before and after (baseline from git archive HEAD) — no regression. Not run: scalar/canonicaljson fuzz targets, outside this delta.

Evidence: BUG-260902-2faftr_results.md, BUG-260902-2faftr_gate-logs.tar.gz.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-0ca406, pid=10925, exit=0)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Verify the reapplied tree independently rather than inheriting the earlier acceptance: confirm from the pinned spec that the removed rule was invented, and that re-adding it still reddens."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260902-cdeacc, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260902-cdeacc)
Review accepted (RUN-260902-cdeacc). Verified against the pinned SPEC.md digest 562546d2... at spec commit 28bf96d: all five citations resolve verbatim, and the three neighbouring clauses that could have declared a naming rule (SPEC.md:348-352 shadow/weaken, 313-316 reverse-DNS retention, 12644-12646 fixture gate) do not — the grammar is the whole rule, and SPEC.md:2344-2345 expressly permits naming a credential profile. Both call sites gone, no third caller existed, count/byte/depth bounds untouched. Nine mutants run by me on a scratch archive copy, all exit 1: full blacklist (18 fails), single-label credential-only narrowing (3), nested blacklist (11), dropped grammar (14), widened 253 bound (4), widened depth (6), dropped backend-settings schema (4), widened 64-entry bound (2), dropped DisallowUnknownFields (8). build/vet/gofmt/test/cover/tracecheck full+scoped/cataloggen -check all exit 0; internal/config coverage 93.9% base -> 93.9% candidate, measured from independent git archive copies. Test file byte-identical to the accepted patch. One non-blocking finding recorded in the verdict: nested-key data preservation is claimed in README and the commit message but only admission is asserted — a synthetic mutant that silently deletes nested token/endpoint/password keys passes the whole repo suite and tracecheck. Outside this AC and not a gate; suggested four-line follow-up. No commit_ack supplied.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-cdeacc, pid=30641, exit=0)
The review ACCEPTED this, and its verification was thorough: absence of the rule was proven by checking the three neighbouring clauses that could have declared one rather than by failing to find it, nine mutants all reddened, and coverage was measured from independent archive copies.

I am asking for one more pass anyway, on the non-blocking finding, because it is the exact class this board has rejected all session.

README and the commit message claim that nested token, endpoint, password and credential keys are PRESERVED AS DATA. The tests assert only that they are ADMITTED. The reviewer built a synthetic mutant that silently deletes those nested keys and it passes. So the artifact claims a property nothing pins, which is how a claim survives that the code does not have.

Close it the cheap way: assert round-trip preservation, not just admission - load a document carrying nested keys with each previously blacklisted label, and assert the values come back byte-identical through the production entry. Then confirm the delete-nested-keys mutant reddens. If for some reason preservation is not actually guaranteed, correct README and the commit message instead; either resolution is fine, claiming it without pinning it is not.

Keep the leaf to one signed commit. Change nothing else - the rest of this revision is accepted and independently verified.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Pin the nested-key preservation the artifacts already claim, or correct the claim; an unpinned assertion is the defect class this board rejects."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260902-11e798, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260902-11e798)
Revision 2 — closed the review's non-blocking finding only; nothing else touched.

REPRODUCED THE FINDING FIRST. On a git archive HEAD copy of the previous revision, a mutant that admits the document and then deletes every nested secret/token/password/credential/auth/env/environment/endpoint key from the live decoded map in validateExtensionValue passes `go test ./... -count=1` at exit 0. So README and the commit message did claim nested preservation with nothing pinning it. Confirmed, not taken on trust.

CLOSED IT BY PINNING, NOT BY CORRECTING. Preservation is genuinely guaranteed, so the claim stands and is now backed. TestExtensionValueObjectKeysArePreservedAsData drives the same production loadConfigDocument entry (reached from exported Load, schema_test.go:622) and asserts BYTE IDENTITY of the re-encoded extensions map rather than a nil error. encoding/json sorts object keys, so a dropped, added, renamed, RETYPED or rewritten nested key all fail. 11 subtests: one per previously blacklisted label with a value distinct per label so one drop cannot be masked by another; a blacklisted label at every admitted depth; non-string values (int, bool, array, nested object) so retyping fails too; and a root key that itself carries a blacklisted label with blacklisted nested keys, so that single case fails if EITHER arm of the removed rule returns.

MUTANTS, all run by me on scratch copies under .temp, tree restored green after each, real exit codes: delete-nested-keys on the PREVIOUS revision exit 0 / 0 failures (the finding); delete-nested-keys on THIS revision exit 1 / 11 preservation subtests red; re-added key blacklist exit 1 / 18; re-added nested-key blacklist exit 1 / 21. The two AC-required narrowing mutants redden MORE than before (18 vs 17, 21 vs 10) — the preservation tests add pinning and remove none.

GATES on the exact committed tree 29d8cee, all exit 0: go build ./..., go vet ./..., gofmt -l ./internal (no output), go test ./... -count=1, go test ./... -cover -count=1, full tracecheck, assigned-scope tracecheck (6.1 6.2 6.3 6.4 6.5 17.1 17.2 17.4), cataloggen -check. internal/config coverage 93.9% -> 94.4%, baseline from an independent git archive HEAD copy — no regression, the new tests raise it. Not run: scalar/canonicaljson fuzz targets, outside this delta.

COMMIT CONTRACT HONOURED: 1a94718 was AMENDED, not followed by a second commit. One signed commit 29d8cee, git verify-commit exits 0, exactly one past checkpoint 67aed0b. Working tree clean. Only two files differ from the accepted revision: internal/config/extension_key_admission_test.go (the new test + helper + encoding/json import) and LOGBOOK.md (the finding recorded, plus the entry's own stale coverage/mutant status line refreshed since it ships in this commit).

Evidence: BUG-260902-2faftr_preservation-evidence.md, BUG-260902-2faftr_gate-logs-rev2.tar.gz.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-11e798, pid=46033, exit=0)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Confirm the delete-nested-keys mutant now reddens and that nothing outside the non-blocking finding moved between revisions."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260902-006eb7, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260902-006eb7)
Review of CR revision 2: ACCEPTED (accept_cr recorded, element parked at to-review for the orchestrator; no commit_ack from this reviewer). Verified independently, not read: SPEC.md fetched at tag v0.5.0 and hashed to the pinned 562546d2...ac484a, all five citations resolve verbatim at the stated lines, and no line in the pinned document reserves an extension-key label - the removed rule really was invented. hasForbiddenConfigName and both call sites are gone with no third caller, and the three extensions surfaces share one validator so no profile keeps the clause. Eight single-clause mutants re-run on a git archive copy, each exiting 1: re-added key blacklist 18 subtests, re-added nested-key blacklist 21, dropped grammar gate 11, widened 253-byte bound 2, widened depth bound 2, silent nested-key deletion 5, dropped DisallowUnknownFields 5, dropped registered backend-settings schema 2. Every mutant count claimed in the shipped LOGBOOK reproduced exactly. Gates in this workspace: build, vet, gofmt, go test ./... -count=1, -cover, full and scoped tracecheck, cataloggen -check all exit 0; internal/config coverage 94.4% against a 93.9% baseline I measured from a git archive of base 67aed0b. Non-blocking finding recorded in the verdict: the LOGBOOK PROVENANCE bullet still says the code and test deltas are byte-identical to the accepted patch, which was true at revision 1 but is false at revision 2 (the test file differs by 82 diff lines - the preservation test added this revision). Documentation-only, correct account adjacent in the same entry, one-sentence correction suggested in the verdict artifact. Evidence: BUG-260902-2faftr_review-verdict-rev2.md.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-006eb7, pid=60064, exit=0)

## Precondition Resources
- [BUG-260902-2faftr_accepted-work.patch](file://BUG-260902-2faftr/BUG-260902-2faftr_accepted-work.patch) — Accepted and reviewed work from the discarded element: blacklist and helper removed, 20 legitimate reverse-DNS keys admitted and round-tripping, re-added-blacklist mutant reddens. Reapply, do not reimplement.

## Outcome Resources
- [BUG-260902-2faftr_spawn-log_-implementer--developer--claude-_RUN-260902-0ca406.log](file://BUG-260902-2faftr/BUG-260902-2faftr_spawn-log_-implementer--developer--claude-_RUN-260902-0ca406.log) — System spawn log captured by task-board
- [BUG-260902-2faftr_results.md](file://BUG-260902-2faftr/BUG-260902-2faftr_results.md) — Reapplied accepted patch: change summary, gate exit codes, coverage delta, seven mutants with subtest-level counts
- [BUG-260902-2faftr_gate-logs.tar.gz](file://BUG-260902-2faftr/BUG-260902-2faftr_gate-logs.tar.gz) — Raw stdout/stderr for every gate and mutant run, including the post-amend final full-suite log
- [BUG-260902-2faftr_change-request_rev1.patch](file://BUG-260902-2faftr/BUG-260902-2faftr_change-request_rev1.patch) — Change Request CR-BUG-260902-2faftr-1 revision 1 candidate patch (repository_delta=present, 5 changed paths)
- [BUG-260902-2faftr_change-request_rev1-validation.log](file://BUG-260902-2faftr/BUG-260902-2faftr_change-request_rev1-validation.log) — Change Request CR-BUG-260902-2faftr-1 revision 1 bounded validation log
- [BUG-260902-2faftr_spawn-log_-reviewer--reviewer--claude-_RUN-260902-cdeacc.log](file://BUG-260902-2faftr/BUG-260902-2faftr_spawn-log_-reviewer--reviewer--claude-_RUN-260902-cdeacc.log) — System spawn log captured by task-board
- [BUG-260902-2faftr_review-verdict.md](file://BUG-260902-2faftr/BUG-260902-2faftr_review-verdict.md) — Reviewer verdict for CR revision 2: ACCEPTED — spec digest re-verified, eight single-clause mutants re-run, all gates green, one non-blocking LOGBOOK provenance inaccuracy recorded
- [BUG-260902-2faftr_review-logs.tar.gz](file://BUG-260902-2faftr/BUG-260902-2faftr_review-logs.tar.gz) — Reviewer-side raw logs: repo-wide test run and coverage run on the reviewed tree.
- [BUG-260902-2faftr_spawn-log_-implementer--developer--claude-_RUN-260902-11e798.log](file://BUG-260902-2faftr/BUG-260902-2faftr_spawn-log_-implementer--developer--claude-_RUN-260902-11e798.log) — System spawn log captured by task-board
- [BUG-260902-2faftr_preservation-evidence.md](file://BUG-260902-2faftr/BUG-260902-2faftr_preservation-evidence.md) — Revision 2: reproduction of the review's non-blocking finding (delete-nested-keys mutant passes on the previous revision, exit 0), the byte-identity preservation test that closes it, mutant table, and gate exit codes on committed tree 29d8cee
- [BUG-260902-2faftr_gate-logs-rev2.tar.gz](file://BUG-260902-2faftr/BUG-260902-2faftr_gate-logs-rev2.tar.gz) — Revision 2 raw logs: build/vet/gofmt/test/cover/tracecheck full+scoped/cataloggen on committed tree 29d8cee, plus the four mutant runs including the previous-revision delete-nested-keys run that exits 0
- [BUG-260902-2faftr_change-request_rev2.patch](file://BUG-260902-2faftr/BUG-260902-2faftr_change-request_rev2.patch) — Change Request CR-BUG-260902-2faftr-2 revision 2 candidate patch (repository_delta=present, 5 changed paths)
- [BUG-260902-2faftr_change-request_rev2-validation.log](file://BUG-260902-2faftr/BUG-260902-2faftr_change-request_rev2-validation.log) — Change Request CR-BUG-260902-2faftr-2 revision 2 bounded validation log
- [BUG-260902-2faftr_spawn-log_-reviewer--reviewer--claude-_RUN-260902-006eb7.log](file://BUG-260902-2faftr/BUG-260902-2faftr_spawn-log_-reviewer--reviewer--claude-_RUN-260902-006eb7.log) — System spawn log captured by task-board
- [BUG-260902-2faftr_review-verdict-rev2.md](file://BUG-260902-2faftr/BUG-260902-2faftr_review-verdict-rev2.md) — Reviewer verdict for CR-BUG-260902-2faftr-2 revision 2: ACCEPTED — spec digest re-verified, eight single-clause mutants re-run, all gates green, one non-blocking LOGBOOK provenance inaccuracy recorded

## Created
2026-09-02T11:13:56Z

## Last Update
2026-09-02T11:54:36Z

## Assigned To
[reviewer] reviewer (claude)
