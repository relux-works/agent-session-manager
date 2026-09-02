## Status
to-review

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
- [x] The invented name blacklist is removed, or narrowed to exactly what the pinned spec declares with the declaring line quoted verbatim
- [x] Legitimate reverse-DNS keys containing env, auth, environment or endpoint labels load through the production entry
- [x] The tests that pinned the invented rule are removed with the rule, not left asserting it
- [x] Any retained secret rule cites SPEC.md:2596-2597 and inspects values rather than key names
- [x] Repository tests, vet, build, tracecheck and catalog check exit 0 with no coverage regression
- [x] Code written per task description and AC
- [x] Relevant tests written for new or changed behavior and passing
- [x] Gating, refusing, validating, authorizing, or attesting behavior covered by negative tests that fail when the gate admits what it must reject, with the production call site named
- [x] Lint clean
- [x] Relevant build/validation commands run after changes and build not broken
- [x] New outcome artifact attached on the board with a task-scoped name when the work produces notes, logs, screenshots, or other deliverables
- [x] Important findings, decisions, anomalies, or regressions recorded in logbook when relevant

## Notes
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 recommendation: remove an invented constraint that refuses legitimate reverse-DNS keys and was hardened rather than caught by an earlier review round."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260902-d4dc25, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260902-d4dc25)
Removed hasForbiddenConfigName and both call sites (validation.go:745 extension keys, :790 nested object keys). Not narrowed — the pinned v0.5.0 spec reserves no extension-key label (SPEC.md:345-347) and imposes no naming rule inside an ExtensionValue object (SPEC.md:347-349), so there is nothing to narrow to. No value-level secret rule was added: SPEC.md:2596-2597 is scoped to a terminal backend-config settings object, already enforced by the closed registered BackendSettingsValidator schema, and that enforcement is now pinned by a test. Removed the two refusal_test.go cases that pinned the invented rule. Added internal/config/extension_key_admission_test.go driving the production Load entry: 20 legitimate reverse-DNS keys carrying every previously blacklisted label are admitted and round-trip; nested endpoint/token/password/credential keys preserved as data; grammar, 253-byte key bound and depth-4 bound stay negative-tested. Three mutants each exited 1 and were restored, including the narrowing one (re-added name blacklist -> 17/20 admission subtests RED). Gates all exit 0: build, vet, gofmt, go test ./... -count=1, go test ./... -cover, global tracecheck, assigned-scope tracecheck, catalog -check. internal/config coverage 93.9% before and after (measured by restoring HEAD files). Commits 18560e6 (fix+tests+README) and 02b942c (logbook), both signed and verified. Artifact: BUG-260902-3ru6vw_results.md.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-d4dc25, pid=28080, exit=0)
spawn autonomous recovery: run RUN-260902-d4dc25 queued successor RUN-260902-0ac6eb (attempt 1/3, model=claude-opus-5): Change Request construction for BUG-260902-3ru6vw failed: change_request_base_authority_mismatch: the STORY-260902-1hrtzp committed candidate 02b942cb04baa900fcefcb507cf31589b0fad266 is not exactly one direct single-parent commit past checkpoint 6f7771954ffb4484ff51395c290a82971d2f961e: <nil>
spawn run started: [implementer] developer (claude) (run=RUN-260902-0ac6eb)
Recovery run RUN-260902-0ac6eb. Predecessor CR failed on delivery shape, not on the fix. Two independent causes.

CAUSE A (this leaf, FIXED): RUN-260902-d4dc25 left two commits for one leaf (18560e6 fix+tests+README, 02b942c LOGBOOK). The CR contract admits exactly one direct single-parent commit past the checkpoint. Squashed into one signed commit 2409b11, parent 150040f, git verify-commit exit 0. Same tree plus one added LOGBOOK STATUS line (which names no commit hash on purpose - it ships in the commit it would describe, the exact trap that left beqfwr LOGBOOK.md:17 naming unreachable 7eac813).

CAUSE B (NOT this leaf, ORCHESTRATOR ACTION NEEDED): workspace WS-b12b4e5018e3 reports checkpoint_oid=6f77719 = origin/main, while 150040f (BUG-260902-beqfwr) sits on the Story branch. That leaf was ACCEPTED, its CR-BUG-260902-beqfwr-1 is state=checkpointed with base_oid=150040f, and the bug is status=done - but the workspace checkpoint was never advanced to it and still names the fork point. Its reviewer recorded the mechanism: the beqfwr producer committed before the CR snapshot, so the base OID was captured as the branch head itself and repository_delta came out empty, leaving the checkpoint unmoved. Consequence: the branch reads as two commits past checkpoint no matter what this leaf does. This leaf must NOT squash 150040f in - that would attribute another accepted, already-done leaf ssh_args change to this CR. The repair is advancing the recorded workspace checkpoint to 150040f, a task-board workspace-state action, not a producer action.

IMPLEMENTATION re-verified in this run, not accepted on the predecessor word. hasForbiddenConfigName and both call sites removed; not narrowed because there is nothing to narrow to. Spec quotes re-read at their exact lines in agent-session-manager-spec@28bf96d (v0.5.0) and they match the code comments verbatim: SPEC.md:345-347 declares the reverse-DNS grammar and reserves no label; SPEC.md:347-349 constrains an ExtensionValue object only as string-keyed within depth 4. No value-level secret rule added: SPEC.md:2596-2597 is scoped two sentences earlier to a terminal backend-config settings object, enforced by the closed registered BackendSettingsValidator schema, now pinned by a test.

MUTANTS run in this run, each applied then restored from a byte-identical backup, control green and tree clean after: (1) re-add the key-name blacklist at the extension-key gate -> TestExtensionKeyAdmissionIsDecidedByTheReverseDNSGrammarAlone exit 1; (2) re-add it at the nested-object-key arm -> TestExtensionValueObjectKeysAreAdmittedAsData 11 subtests red, exit 1; (3) drop reverseDNSPattern from the key gate -> TestExtensionKeyRefusalStillEnforcesTheReverseDNSGrammar exit 1; (4) raise the object nesting bound from depth>=4 to depth>=9 -> exit 1. Mutants 1 and 2 are narrowing, not delete-only, so the tests bound the class of the removed rule rather than recording its deletion; 3 and 4 prove the two real gates left in the same function were not weakened with it. Production call site named: loadConfigDocument -> validateConfiguration -> validateExtensions/validateExtensionValue, driven by every case in extension_key_admission_test.go.

GATES on the delivered tree 2409b11, each standalone with its real exit code: go build ./... 0, go vet ./... 0, gofmt -l . 0 with no output, go test ./... -count=1 0 (9/9 ok), tracecheck 0, tracecheck with the eight assigned section 6/17 scopes 0, cataloggen -check 0. internal/config coverage 93.9 percent, unchanged.

NOT TOUCHED deliberately: LOGBOOK.md:17 names 7eac813 for BUG-260902-beqfwr, unreachable from HEAD, delivered commit is 150040f. Already raised by that leaf reviewer as non-blocking. Correcting it here would pull a foreign line into this CR diff. Left for the orchestrator.

Artifact: BUG-260902-3ru6vw_recovery-run-evidence.md.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-0ac6eb, pid=38836, exit=0)
spawn autonomous recovery: run RUN-260902-0ac6eb queued successor RUN-260902-5ec182 (attempt 2/3, model=claude-opus-5): Change Request construction for BUG-260902-3ru6vw failed: change_request_base_authority_mismatch: the STORY-260902-1hrtzp committed candidate 2409b115ad2814c898051aa007de40c2d5b2e8e0 is not exactly one direct single-parent commit past checkpoint 6f7771954ffb4484ff51395c290a82971d2f961e: <nil>
spawn run started: [implementer] developer (claude) (run=RUN-260902-5ec182)
spawn run RUN-260902-5ec182 cancelled by operator; operator action required; reason: no operator reason supplied
agent completed: [implementer] developer (claude) (exit=143)
spawn run completed: claude (run=RUN-260902-5ec182, pid=47819, exit=143)
RE-PROVISIONED under a single-leaf Story. Work already produced on the previous shared branch is attached as the precondition resource BUG-260902-3ru6vw_produced-work.patch. Reapply it rather than reimplementing; it was never reviewed, so expect the reviewer to judge it on its merits, but do not redo the analysis.

WHY IT MOVED: the previous Story recorded an empty checkpoint OID after its first leaf, so it could not admit a second one. Nothing was wrong with this work.

THE FINDING, restated so the reapply keeps its shape. hasForbiddenConfigName refuses any name containing secret, token, password, credential, auth, environment, env or endpoint, applied to reverse-DNS extension keys and to nested map keys at any depth. The pinned spec defines extension keys as reverse-DNS grammar only at SPEC.md:344-352. The secret rule at SPEC.md:2596-2597 governs VALUES, not key names, and the nested-key arm is wholly invented.

It refuses legitimate keys this organisation would use - works.relux.env-tools, works.relux.environment, com.example.auth-manager, io.example.endpoint-list - while inspecting zero values, so a key holding an actual credential passes. It was pinned by refusal_test.go:390-395, which means an earlier review round hardened the invented rule instead of catching it.

Remove the rule and the tests that pinned it. If any value-level secret handling is kept, cite SPEC.md:2596-2597 and inspect values. Where the pinned spec is silent, be permissive and say so.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 recommendation: reapply the invented-constraint removal on a single-leaf Story, avoiding the empty-checkpoint defect by construction."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260902-8e7b33, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260902-8e7b33)
Reapplied on task-board/story/STORY-260902-suxwmp as one signed commit 94da5a0 (verify-commit good, oparin@me.com). The attached patch did NOT apply: git apply --check failed on README.md:271 and internal/config/validation.go:843 from prior-branch drift; LOGBOOK/refusal_test/new test file applied cleanly, the validation.go and README hunks were reapplied by content and verified to reach the same post-state.

hasForbiddenConfigName and both call sites (validation.go:745 extension keys, :790 nested keys at any depth) are gone; the two refusal_test.go cases that pinned it were deleted with the rule. Located the pinned SPEC by digest 562546d2... matching ownership.v0.5.0.json and verified every quoted line at its stated number: 345-347 is the only extension-key rule and reserves no label; 347-349 constrains a nested key only as string-keyed within depth 4.

Went past the patch on one point. Board history shows the original demand for this coverage came from TASK-260830-17suox review evidence citing the SEC 6.4 secret/credential/endpoint refusal, and the patch answered only SPEC.md:2596-2597. Checked SEC 6.4: SPEC.md:2562-2563 is a field-declaration rule enforced by decodeStrict DisallowUnknownFields, and SPEC.md:2344-2345 explicitly permits a config field to NAME a machine-local environment variable or credential profile - the exact thing the removed rule refused. Added a negative test pinning 2562-2563 and recorded both clauses in README/LOGBOOK. Section 16 swept: no supporting rule, and SPEC.md:11264-11268 forbids describing a best-effort scanner as a guarantee.

Seven single-clause mutants, all red; five are narrowing/widening mutants against gates that survive this change (grammar dropped, 253 bound widened to 254, depth 4 widened to 5, ValidateBackendSettings call dropped, DisallowUnknownFields dropped), not delete-mutants against the removed rule. ANOMALY: mutant 7 initially SURVIVED - the first draft of the SEC 6.4 test omitted the required extensions member, so the document was refused for a missing member and dropping DisallowUnknownFields only changed ErrConfigDecode to ErrConfigValidation. Fixed by adding extensions = {}, asserting the same document without api_token loads, and asserting the refusal is specifically ErrConfigDecode; only then did the mutant redden. Corrected the patch LOGBOOK status line, which named the wrong branch and 93.9% coverage.

All gates run in this session as standalone processes, real exit codes: gofmt -l ./internal 0, go build ./... 0, go vet ./... 0, go test ./... -count=1 0, go test ./... -cover -count=1 0, full tracecheck 0 (assigned_scopes=0), scoped tracecheck 0 (assigned_scopes=9), cataloggen -check 0. Coverage baseline MEASURED not assumed: git archive HEAD of the pre-change tree gives internal/config 93.7%, post-change 93.7%, no regression. Branch not pushed: Story integration is the orchestrator step. Evidence: BUG-260902-3ru6vw_reapply-evidence.md.

## Precondition Resources
- [BUG-260902-3ru6vw_produced-work.patch](file://BUG-260902-3ru6vw/BUG-260902-3ru6vw_produced-work.patch) — Work produced on the shared Story branch before the checkpoint pointer defect blocked its Change Request; reapply rather than reimplement

## Outcome Resources
- [BUG-260902-3ru6vw_spawn-log_-implementer--developer--claude-_RUN-260902-d4dc25.log](file://BUG-260902-3ru6vw/BUG-260902-3ru6vw_spawn-log_-implementer--developer--claude-_RUN-260902-d4dc25.log) — System spawn log captured by task-board
- [BUG-260902-3ru6vw_results.md](file://BUG-260902-3ru6vw/BUG-260902-3ru6vw_results.md) — Spec quotes, removal rationale, new admission/negative suite, mutant table, gate exit codes, coverage delta
- [BUG-260902-3ru6vw_spawn-log_-implementer--developer--claude-_RUN-260902-0ac6eb.log](file://BUG-260902-3ru6vw/BUG-260902-3ru6vw_spawn-log_-implementer--developer--claude-_RUN-260902-0ac6eb.log) — System spawn log captured by task-board
- [BUG-260902-3ru6vw_recovery-run-evidence.md](file://BUG-260902-3ru6vw/BUG-260902-3ru6vw_recovery-run-evidence.md) — Recovery run RUN-260902-0ac6eb: CR failure diagnosis (leaf squashed to one signed commit; sibling checkpoint gap is orchestrator-owned), independent re-verification, four mutants, gate exit codes
- [BUG-260902-3ru6vw_spawn-log_-implementer--developer--claude-_RUN-260902-5ec182.log](file://BUG-260902-3ru6vw/BUG-260902-3ru6vw_spawn-log_-implementer--developer--claude-_RUN-260902-5ec182.log) — System spawn log captured by task-board
- [BUG-260902-3ru6vw_spawn-log_-implementer--developer--claude-_RUN-260902-8e7b33.log](file://BUG-260902-3ru6vw/BUG-260902-3ru6vw_spawn-log_-implementer--developer--claude-_RUN-260902-8e7b33.log) — System spawn log captured by task-board
- [BUG-260902-3ru6vw_reapply-evidence.md](file://BUG-260902-3ru6vw/BUG-260902-3ru6vw_reapply-evidence.md) — Reapply on STORY-260902-suxwmp: patch-drift resolution, independent spec verification of all five Section 6 clauses, seven single-clause mutants (incl. one initially-surviving mutant and its fix), gate exit codes, measured coverage baseline

## Created
2026-09-02T09:38:37Z

## Last Update
2026-09-02T10:55:06Z

## Assigned To
[implementer] developer (claude)
