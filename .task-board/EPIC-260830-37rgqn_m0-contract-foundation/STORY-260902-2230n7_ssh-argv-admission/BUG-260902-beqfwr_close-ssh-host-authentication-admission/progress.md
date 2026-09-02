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
- [x] Admission derives the permitted option set from a declared source and refuses the complement; an unknown option is refused, not admitted
- [x] Grouped short flags such as -vo and -4o are parsed the way OpenSSH parses them, with a case proving each
- [x] StrictHostKeyChecking=false is refused, with the OpenSSH alias behaviour cited in the case
- [x] -F, ProxyCommand, KnownHostsCommand, Include and PermitLocalCommand+LocalCommand are each refused at the production entry
- [x] Each negative case reddens when only its own clause is weakened, proven by mutant
- [x] The completeness claim in TestLoadRefusesEveryOpenSSHHostAuthenticationBypassSpelling is either earned by derivation or the name is corrected
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
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 recommendation: close a confirmed host-authentication bypass in landed code, reproduced against a live OpenSSH binary."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260902-83ed8b, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260902-83ed8b)
Ready for review. Commit 150040f on task-board/story/STORY-260902-1hrtzp (signed, verified), one commit past origin/main.

Fix: internal/config/sshargs.go replaces the three-name blacklist with derived admission. Declared sources: the ssh(1) short-option arity transcribed from the OpenSSH 10.2p1 usage text, sshPermittedFlags (4 6 C T a q v), sshPermittedValueFlags (i l p o) and sshOptionRegistry with a value rule per option. admitSSHArguments walks argv with getopt semantics and refuses the complement, so an option name the parser has never heard of is refused, not admitted. Single refusal call site is still validation.go:465; the reason is a closed vocabulary appended to the mesh.peers[i].ssh_args clause.

StrictHostKeyChecking is declared with only the enforcing spelling yes, so the live OpenSSH aliases no/off/false/NO/FALSE/accept-new fall outside the permitted set without any alias being listed. Host-authentication names stay declared only so their refusal reports the 6.3 clause rather than an unknown-name clause; they widen nothing.

Every cited bypass reproduced against OpenSSH 10.2p1 and now refused at loadConfigDocument: -vo StrictHostKeyChecking=no, -4o UserKnownHostsFile=/dev/null, -o StrictHostKeyChecking=false, -F in both spellings, ProxyCommand, KnownHostsCommand, Include, PermitLocalCommand+LocalCommand.

Tests: new internal/config/ssh_admission_test.go, all driving loadConfigDocument. Grammar-arity derivation, registry-derived host-authentication refusals, both-directions closure over permitted options with a key-set equality assertion, whole-grammar letter walk, grouped-flag cases, undeclared-name cases, non-option argument cases, and positive cases so admission is not vacuously strict. TestLoadRefusesEveryOpenSSHHostAuthenticationBypassSpelling is renamed to TestLoadRefusesHostAuthenticationBypassAcrossTheOptionSpellingGrammar (what it actually covers) with six added cases; its doc comment names where name-completeness is now earned.

Two pre-existing schema_test.go bound cases used bare xxx filler as ssh_args, which is an operand and is now refused; rewritten to -i<path> argv of identical byte lengths, still proving the 4096/65536/64 bounds at and past the limit.

Mutants: ten single-clause weakenings (no deletions), each reddens its own cases, control green. Note M1 (permit -F) does not redden the derived per-letter subtest, because permitting F removes that subtest from the loop; it reddens the explicitly named -F assertions in the parent body. Matrix in BUG-260902-beqfwr_mutant-evidence.md, raw output in BUG-260902-beqfwr_mutant-evidence-raw.txt.

Gates, each run standalone with its real exit code: go build ./... 0, go vet ./... 0, gofmt -l . 0 with no output, go test ./... -count=1 0, go test ./... -cover -count=1 0, tracecheck 0, tracecheck with the nine config sections 0, cataloggen -check 0. internal/config coverage 93.7 -> 93.9 percent, no package regressed. golangci-lint run ./internal/config/... exits 1 on one pre-existing staticcheck QF1003 at validation.go:211, untouched by this change; the repository declares no golangci-lint gate.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-83ed8b, pid=98935, exit=0)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Rank-1 recommendation: attack the new admission decision with bypass spellings it was not shown, and verify each against a live ssh binary rather than against the implementation's own idea of OpenSSH."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260902-23ff4c, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260902-23ff4c)
REVIEW VERDICT: ACCEPTED (CR-BUG-260902-beqfwr-1 rev 1, reviewer RUN-260902-23ff4c). Evidence: BUG-260902-beqfwr_review-verdict.md.

Why repository_delta=empty is not a no-op: the CR base OID 150040f IS the branch head, and base tree == candidate tree == HEAD tree == 7fce177. The producer committed before the snapshot was taken, so the base landed post-commit. main..HEAD = 1 commit = 150040f (signed, verified, 7 files, +609/-55). That commit is the reviewed delta; the commit-owning mover should treat main..HEAD as this leaf scope.

Reviewer attack, not reading:
- Production reachability grepped: admitSSHArguments has one caller (validateMesh:465), reached from validateConfiguration:391 via all four entries that can carry ssh_args (validation.go:53,77,101 decode paths and writer.go:16). No path materializes Peer.SSHArgs without crossing the gate.
- Arity of all 11 admitted letters re-derived from the live OpenSSH 10.2p1 binary, not from the comment. Tables match.
- 63 adversarial argv driven through loadConfigDocument. All dangerous spellings refused, including 23 the producer never enumerated: ProxyJump, -J, CertificateFile, PKCS11Provider, SetEnv, SendEnv, RemoteCommand, ControlMaster/ControlPath, PubkeyAuthentication=no, PasswordAuthentication, HostName, Host=*, CanonicalizeHostname, ForwardAgent, -A, Tunnel, FingerprintHash=md5, RequiredRSASize=1024, PubkeyAcceptedAlgorithms=+ssh-rsa, CASignatureAlgorithms=+ssh-rsa, RevokedHostKeys=/dev/null, SecurityKeyProvider, ObscureKeystrokeTiming.
- Name-parsing divergence attacked specifically (quoted keyword, newline injection, second directive in one -o, unicode separators): AX refuses or ssh refuses in every case; no spelling makes AX read a permitted name where ssh reads a dangerous one.
- Admitted-side differential verified against the binary: ssh -G -F/tmp/cfg h sets proxycommand, ssh -G -i -F/tmp/cfg h does not. -i consuming an option-looking optarg is faithful getopt, not a hole.
- Five single-clause narrowing mutants run by the reviewer (permit -F; stricthostkeychecking permits no; declare proxycommand permitted; group walk ignores non-leading letters, which reintroduces the exact -vo bypass; admit bare operands). Each reddens cases naming its own clause. Control green, tree clean before and after. No delete-only mutant.

Gates re-run by the reviewer, real exit codes: go build 0, go vet 0, gofmt -l no output, go test ./... -count=1 0 (9/9 ok), go test ./... -cover 0, tracecheck 0, tracecheck -section 6.3 0, cataloggen -check 0. internal/config coverage 93.9 percent.

Non-blocking findings for the orchestrator:
1. LOGBOOK.md STATUS line names commit 7eac813, which is not reachable from HEAD and carries a different tree (964d44e vs delivered 7fce177) - the producer re-committed as 150040f. Correct to 150040f during checkpoint/integration.
2. parseSSHConfigOption splits the option name on any unicode.IsSpace rune while OpenSSH splits on ASCII whitespace/quote/=, so -o "BatchMode<U+00A0>yes" is admitted by AX and rejected by ssh with Bad configuration option. One-directional: the name is truncated then required to be in the registry, so no dangerous keyword reaches the permitted set. Startup error, not a relaxed host check.
3. UserKnownHostsFile and GlobalKnownHostsFile are refused for every value, stricter than SPEC 6.3 which only requires refusing an empty one. Spec-conformant (6.3 example argv still loads); flagged so a future legitimate custom known-hosts request is a deliberate registry widening with both-direction samples.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-23ff4c, pid=15731, exit=0)
RE-PROVISIONED. The accepted work is attached as the precondition resource BUG-260902-beqfwr_accepted-work.patch. REAPPLY it; do not reimplement and do not redesign.

It was already produced AND reviewed to acceptance: the reviewer re-derived the arity of all eleven admitted option letters from the live OpenSSH 10.2p1 binary rather than from the code comment, drove 63 adversarial argv through loadConfigDocument, and confirmed 23 dangerous options the producer never enumerated are refused by construction - ProxyJump, -J, CertificateFile, PKCS11Provider, SetEnv, SendEnv, RemoteCommand, ControlMaster, PubkeyAuthentication=no, RevokedHostKeys=/dev/null and others. That is the whole point of deriving the permitted set instead of listing forbidden names, and it must survive the reapply.

WHY THIS MOVED: the previous Story recorded an empty checkpoint OID, so it could no longer admit a leaf or produce a story-final Change Request. Nothing is wrong with the work.

Reapply onto the current trunk, rerun the gates, and report any diff outside a trivial rebase context. If reapplying changes behaviour anywhere, stop and say so rather than absorbing it - the acceptance rests on that exact tree.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Reapplying accepted and reviewed work on a single-leaf Story, so no checkpoint pointer is involved and the empty-OID defect cannot recur."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260902-00f3d4, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260902-00f3d4)
Reapply run in Story workspace STORY-260902-2230n7. The accepted patch was applied with git am --3way (clean, no conflict) and not reimplemented; the leaf is exactly one signed single-parent commit past checkpoint 6f77719, author Ivan Oparin, signature verified G.

One correction on top of the accepted patch: its LOGBOOK.md STATUS line named commit 7eac813 on the retired branch STORY-260902-1hrtzp, an object unreachable from this leaf. The line now records the checkpoint it sits on and names no shipping hash. That is the only delta from the accepted content.

All evidence was rerun here, none inherited. go build, go vet, gofmt -l, go test ./... -count=1 -cover, tracecheck, tracecheck -section 6.3 (assigned_scopes=1), and the cataloggen -check gate each exit 0 as standalone processes. internal/config coverage 93.7% measured in a detached worktree at the checkpoint, 93.9% on this leaf. The five cited OpenSSH behaviours were reproduced again against the live OpenSSH_10.2p1 binary on this host, including -vo StrictHostKeyChecking=no printing stricthostkeychecking false and -o StrictHostKeyChecking=false printing the same.

Ten single-clause mutants of internal/config/sshargs.go were run: permit -F, declare ProxyCommand / Include / KnownHostsCommand / PermitLocalCommand+LocalCommand permitted, widen StrictHostKeyChecking to its disabling aliases, stop walking a group past its first letter, admit bare operands, admit undeclared option names, and let UserKnownHostsFile permit every value. All ten reddened the suite; none survived. Production call site under attack is validation.go:465, validateMesh calling admitSSHArguments, driven through loadConfigDocument with real TOML.

Evidence: BUG-260902-beqfwr_reapply-reverification.md, BUG-260902-beqfwr_mutants-01.log, BUG-260902-beqfwr_mutants.py.
Tree equivalence proof for the reapply: git diff 150040f HEAD reports exactly one changed file, LOGBOOK.md, +2/-1 — the STATUS-line correction for the reviewer non-blocking finding #1. Every code, test, and README byte is identical to the accepted tree, and the accepted commit 150040f has the same parent 6f77719 as this leaf, so there was no rebase context to absorb and no behaviour changed anywhere. The acceptance rests on the same tree it was given for.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-00f3d4, pid=50657, exit=0)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Verify the reapplied tree independently rather than inheriting the earlier acceptance, and re-attack the admission with spellings the producer did not enumerate."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260902-80664c, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260902-80664c)
Review rev2 ACCEPTED (RUN-260902-80664c). Reapply verified byte-identical to BUG-260902-beqfwr_accepted-work.patch for all code+README; only LOGBOOK.md differs, carrying the stale-SHA correction the rev1 review requested. Gates rerun in this workspace: gofmt/build/vet/test/cover/tracecheck/tracecheck -section 6.3/cataloggen -check all exit 0; internal/config 93.7% (checkpoint 6f77719, measured) -> 93.9%. Twelve reviewer single-clause narrowing mutants each redden cases naming their own clause. Differential attack: 6710 generated argv through loadConfigDocument, 260 admitted, all 260 replayed against live OpenSSH 10.2p1 -- the admitted set moves only declared settings, stricthostkeychecking only ask->true, and 0 divergences across 16 dangerous resolved keys (proxycommand, localcommand, knownhostscommand, userknownhostsfile, forwardagent, ...). Non-blocking finding for a follow-up leaf: the short-flag tables lack the both-directions key-set pin that permittedSSHOptionSamples gives the -o registry, so mutants permitting -E, -L or -A pass the suite green; those are capability widenings outside section 6.3, and the two letters that can reach the normative class (-F, -J) do carry explicit named assertions. No commit_ack supplied; scope is the single signed commit bc3ed70 past checkpoint 6f77719.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-80664c, pid=61987, exit=0)

## Precondition Resources
- [BUG-260902-beqfwr_accepted-work.patch](file://BUG-260902-beqfwr/BUG-260902-beqfwr_accepted-work.patch) — Accepted and reviewed work (63 adversarial argv, 23 unenumerated options refused); preserved before re-provisioning past an empty-OID checkpoint. Reapply, do not reimplement.

## Outcome Resources
- [BUG-260902-beqfwr_spawn-log_-implementer--developer--claude-_RUN-260902-83ed8b.log](file://BUG-260902-beqfwr/BUG-260902-beqfwr_spawn-log_-implementer--developer--claude-_RUN-260902-83ed8b.log) — System spawn log captured by task-board
- [BUG-260902-beqfwr_results.md](file://BUG-260902-beqfwr/BUG-260902-beqfwr_results.md) — Derived ssh_args admission: design, live OpenSSH evidence, tests, gate results
- [BUG-260902-beqfwr_mutant-evidence.md](file://BUG-260902-beqfwr/BUG-260902-beqfwr_mutant-evidence.md) — Ten single-clause mutants with per-case redden matrix
- [BUG-260902-beqfwr_mutant-evidence-raw.txt](file://BUG-260902-beqfwr/BUG-260902-beqfwr_mutant-evidence-raw.txt) — Raw go test output for the mutant matrix
- [BUG-260902-beqfwr_change-request_rev1.patch](file://BUG-260902-beqfwr/BUG-260902-beqfwr_change-request_rev1.patch) — Change Request CR-BUG-260902-beqfwr-1 revision 1 candidate patch (repository_delta=empty, 0 changed paths)
- [BUG-260902-beqfwr_change-request_rev1-validation.log](file://BUG-260902-beqfwr/BUG-260902-beqfwr_change-request_rev1-validation.log) — Change Request CR-BUG-260902-beqfwr-1 revision 1 bounded validation log
- [BUG-260902-beqfwr_spawn-log_-reviewer--reviewer--claude-_RUN-260902-23ff4c.log](file://BUG-260902-beqfwr/BUG-260902-beqfwr_spawn-log_-reviewer--reviewer--claude-_RUN-260902-23ff4c.log) — System spawn log captured by task-board
- [BUG-260902-beqfwr_review-verdict.md](file://BUG-260902-beqfwr/BUG-260902-beqfwr_review-verdict.md) — Reviewer verdict for CR revision 1: accepted, with reviewer-run differential attack against live OpenSSH 10.2p1, five single-clause mutants, re-run gates, and three non-blocking findings
- [BUG-260902-beqfwr_spawn-log_-implementer--developer--claude-_RUN-260902-00f3d4.log](file://BUG-260902-beqfwr/BUG-260902-beqfwr_spawn-log_-implementer--developer--claude-_RUN-260902-00f3d4.log) — System spawn log captured by task-board
- [BUG-260902-beqfwr_reapply-reverification.md](file://BUG-260902-beqfwr/BUG-260902-beqfwr_reapply-reverification.md) — Reapply run: gates, coverage delta against the checkpoint, live OpenSSH reproductions, and ten single-clause mutant results, all rerun in the new Story workspace
- [BUG-260902-beqfwr_mutants-01.log](file://BUG-260902-beqfwr/BUG-260902-beqfwr_mutants-01.log) — Raw output of the ten single-clause mutant runs against internal/config/sshargs.go
- [BUG-260902-beqfwr_mutants.py](file://BUG-260902-beqfwr/BUG-260902-beqfwr_mutants.py) — Mutant runner: applies one clause weakening at a time, runs the config suite, restores the tree
- [BUG-260902-beqfwr_change-request_rev2.patch](file://BUG-260902-beqfwr/BUG-260902-beqfwr_change-request_rev2.patch) — Change Request CR-BUG-260902-beqfwr-2 revision 2 candidate patch (repository_delta=present, 7 changed paths)
- [BUG-260902-beqfwr_change-request_rev2-validation.log](file://BUG-260902-beqfwr/BUG-260902-beqfwr_change-request_rev2-validation.log) — Change Request CR-BUG-260902-beqfwr-2 revision 2 bounded validation log
- [BUG-260902-beqfwr_spawn-log_-reviewer--reviewer--claude-_RUN-260902-80664c.log](file://BUG-260902-beqfwr/BUG-260902-beqfwr_spawn-log_-reviewer--reviewer--claude-_RUN-260902-80664c.log) — System spawn log captured by task-board
- [BUG-260902-beqfwr_review-verdict-rev2.md](file://BUG-260902-beqfwr/BUG-260902-beqfwr_review-verdict-rev2.md) — Reviewer verdict for CR revision 2: ACCEPTED. Reapply verified byte-identical to accepted work; 8 gates rerun; 12 single-clause mutants; 260 admitted argv replayed against live OpenSSH 10.2p1 with 0 dangerous divergences.

## Created
2026-09-02T09:38:18Z

## Last Update
2026-09-02T10:49:07Z

## Assigned To
[reviewer] reviewer (claude)
