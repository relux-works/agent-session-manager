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
- [x] An endpoint that could be read as an option is refused at the production Load entry, covering every shape from the hostile-endpoint probe
- [x] Dangling value flags with no operand are refused
- [x] The short-flag tables carry the same both-directions key-set pin as the option registry, so a mutant permitting an extra letter reddens
- [x] Each case reddens when only its own clause is weakened, proven by mutant
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
REAPPLY the attached precondition patch; do not reimplement. It was produced and reviewed to acceptance and is byte-for-byte what the previous element carried.

Keep the whole leaf to ONE signed commit, and let the Change Request snapshot open at the checkpoint rather than after your commit - the previous attempt produced an empty repository delta because the commit landed before the snapshot, which is a known board defect that wedges multi-leaf Stories.

Reapply onto the current trunk, rerun the gates, and report any diff beyond trivial rebase context.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Reapplying accepted work on a single-leaf Story created together with its bug and never reparented, which is the only shape that has integrated cleanly today."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260902-c50f0b, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260902-c50f0b)
Accepted patch of BUG-260902-3c7ovg reapplied verbatim; git apply --check exit 0, no conflict. Only the stale element/branch identifiers in the LOGBOOK entry were retargeted, plus two LOGBOOK lines recording the evidence re-measured here. Commit b8c2396 on task-board/story/STORY-260902-cq7ivu, one commit past checkpoint 10aaa16, signed and git verify-commit clean; 7 files, +573/-2.

Re-verified myself, not carried over: 34-shape hostile-endpoint probe through production loadConfigDocument, 32 of 34 admitted against an archive of 10aaa16 and 0 of 34 admitted after, with -ivan@peer.example proving the leading-hyphen clause is not redundant with the grammar. 21 single-clause mutants, 19 killed each reddening only its own cases, 2 proven equivalent (deleting the empty-host guard leaves the empty label refused by admitEndpointHostLabel; weakening the empty-port length bound leaves the port refused by number >= 1) - both subsuming clauses are themselves killed by M10 and M15. Dangling value flags pinned per declared letter in three spellings, 12 refusals.

Gates all exit 0: go build, go vet, gofmt -l, go test ./... -count=1, go test ./... -cover, tracecheck global and sections 6.1-6.5, cataloggen -check. internal/config coverage 94.4 percent at the base measured here from a git archive scratch tree, 94.6 percent at this commit.

Integration into trunk left to the orchestrator per the Story workspace contract.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-c50f0b, pid=38480, exit=0)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Verify the reapplied tree independently rather than inheriting the earlier acceptance, and re-attack the endpoint admission with shapes the producer did not enumerate."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260902-8d07cc, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260902-8d07cc)
REVIEW rev1 = CHANGES REQUESTED. Production code is correct: 68 independent hostile endpoint shapes all refused at the production Load entry on all three schema versions (v1/v2/v3 all route through validateConfiguration -> validateMesh), and every dangling value-flag spelling for -i -l -p -o is refused with the flag-without-its-value clause. All gates rerun and exit 0 (build, vet, gofmt, go test ./..., cover with internal/config at 94.6%, tracecheck global and 6.1-6.5, cataloggen -check).

BLOCKING: the leading-hyphen clause is not pinned against narrowing. TestLoadRefusesOptionLikeMeshEndpoints has 8 cases and none contains an @, so all 8 are also refused by the downstream grammar as a host clause once the hyphen clause is gone. Mutant confirmed and reproduced: narrow endpoint.go:44 to strings.HasPrefix(endpoint, "-") && !strings.Contains(endpoint, "@") and the entire internal/config suite stays green (ok 1.341s) while endpoint = -ivan@peer.example and -4@peer.example load with err=nil. ssh(1) parses -ivan@peer.example as -i with identity file van@peer.example. This is exactly the class the new LOGBOOK entry calls decisive, and its sentence the M01/M02 mutants below redden on exactly that case is false as shipped: the proof lives only in the throwaway .temp probe, not in the 7-file delta.

FIX (one line): add "-ivan@peer.example" and "-4@peer.example" to TestLoadRefusesOptionLikeMeshEndpoints, then the LOGBOOK sentence becomes true. Nothing else needs to change; do not reimplement the gate.

Non-blocking, fix only if cheap: (1) endpoint peer.example:022 is admitted, and the padded port grammar case peer.example:002222 reddens only via the len>5 bound, i.e. the same mechanism as port over bound, so it pins no leading-zero rule; not a bypass since ssh reads 022 as 22. (2) Inserting net.ParseIP(host) != nil -> true into admitEndpointHost survives the suite, widening the grammar to admit ::1:22 and 2001:db8::1:2222, because the only bare IPv6 case has no port; grammar widening only, not option injection.

26 distinct mutants run against the committed suite, 24 killed, 2 survived (the @-carve-out above, and the ParseIP widening). Deletions and narrowings on every clause. Full verdict, probe table and mutant list: BUG-260902-4ajzyz_review-verdict.md.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-8d07cc, pid=48643, exit=0)
Review: production code is correct - 68 hostile endpoint shapes refused at the production entry on all three schema versions, every dangling value-flag spelling refused. One blocking finding, and it is worth understanding rather than just patching.

THE TESTS PASS FOR THE WRONG REASON. All 8 cases in TestLoadRefusesOptionLikeMeshEndpoints lack an @, so once the leading-hyphen clause is removed the downstream host grammar refuses them anyway. The clause the test names is never the thing deciding. Confirmed by mutant: narrow endpoint.go:44 to strings.HasPrefix(endpoint, "-") && !strings.Contains(endpoint, "@") and the whole internal/config suite stays green while endpoint = -ivan@peer.example and -4@peer.example load with err=nil.

That is not academic. ssh(1) parses -ivan@peer.example as -i with identity file van@peer.example, so the narrowed form is a live injection.

AND THE ARTIFACT OVERCLAIMS IT. The LOGBOOK entry says the M01/M02 mutants redden on exactly that case. As shipped that sentence is false: the proof lives only in a throwaway .temp probe and never reached the committed tests. The previous acceptance rested on evidence that does not exist in the tree - which is precisely why a re-review does not inherit a verdict.

FIX: add committed cases that ONLY the leading-hyphen clause can refuse - an option-shaped endpoint containing @, at minimum -ivan@peer.example and -4@peer.example - and confirm the narrowing mutant reddens on them. Then correct the LOGBOOK sentence to describe what the committed suite proves, not what a probe once showed.

Apply the isolation rule generally while you are here: a negative case must be refused BY THE CLAUSE IT NAMES. If another clause would refuse the same input, the case pins nothing. Check the other endpoint cases for the same shape rather than fixing only the two the reviewer named.

Keep the leaf to one signed commit.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 recommendation: pin the leading-hyphen clause with cases only it can refuse, and correct a LOGBOOK sentence that describes a probe rather than the committed suite."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260902-120fd0, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260902-120fd0)
Review blocking finding closed. Added TestLoadRefusesOptionShapedEndpointsTheGrammarWouldAdmit: seven @-bearing option-shaped endpoints, each asserted refused AND admitted once its leading hyphen is removed. All seven redden under the reviewers exact narrowing mutant HasPrefix(-) && !Contains(@), which was green before.

Isolation rule applied across the whole grammar, not only the two shapes named: every refusal case in endpointGrammarCases now carries a neighbour - the same endpoint with only the named violation removed - asserted admitted, and TestEveryRefusedMeshEndpointCaseDeclaresAnIsolatingNeighbour reddens if a later case is added without one. 31 refusal cases carry neighbours (7 user, 17 host, 7 port). The whitespace clause is the one clause that cannot be pinned by admission - no whitespace endpoint is spellable by a login name or DNS label either way - so it is documented as pinned by its named reason and ordering rather than overclaimed.

Both non-blocking observations closed. peer.example:022 is now pinned as ADMITTED reviewed behaviour rather than narrowing product behaviour to serve a test name; the overclaiming padded port case is replaced by port over length bound (:0000022, decimal 22 inside the numeric bound, so only len>5 refuses it, killed by M18 alone). Bare-IPv6-with-port closed with 2001:db8::1:2222 and ::1:22, which kill the net.ParseIP host widening that survived the previous suite (M17).

LOGBOOK sentence corrected: the claim that M01/M02 redden on -ivan@peer.example was false as shipped and is replaced by a REGRESSION entry describing what the committed suite proves.

28 single-clause mutants, 26 killed each reddening only its own cases, 2 proven equivalent by measurement not argument: a differential dump of admitMeshEndpoint over a generated 9360-endpoint corpus is byte-identical between baseline and each survivor (0 of 9360 rows differ). The previous harness counted two compile failures as whitespace-clause kills; this one reports them as INVALID-BUILD and mutates the unicode import too.

No production change this cycle - review confirmed endpoint.go correct against 68 hostile shapes on all three schema versions, the defect was in the evidence. Scope: endpoint_admission_test.go, LOGBOOK.md, README.md.

Kept to ONE signed commit: amended to 1d6ce05 on task-board/story/STORY-260902-cq7ivu, one commit past checkpoint 10aaa16, git verify-commit clean, worktree clean, 7 files +711/-2.

Gates all exit 0 on the committed tree: go build, go vet, gofmt -l, go test ./... -count=1, go test ./... -cover, tracecheck global and sections 6.1-6.5, cataloggen -check, no generated-file drift. internal/config coverage 94.4 base -> 94.6 pre-review -> 94.7 here.

Integration into trunk left to the orchestrator per the Story workspace contract.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-120fd0, pid=60627, exit=0)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Verify the new cases are refused BY the leading-hyphen clause and not by a downstream one, and that the corrected LOGBOOK sentence now describes the committed suite."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260902-7ef84f, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260902-7ef84f)
REVIEW rev2 = CHANGES REQUESTED. Production code confirmed correct and unbypassable: gate wired at validation.go:452 inside validateMesh, reached only through validateConfiguration (all three schema translators + writer); 171 hostile loads (57 shapes x 3 schema versions) all refused with the clause named; 28 independent single-clause mutants with 24 killed and 2 proven equivalent by a 4704-endpoint differential corpus; all repo gates green at the candidate tree, internal/config coverage 94.7%. ONE LIVE SURVIVOR: admitEndpointPort len(port) > 5 -> > 6 keeps the whole suite green while admitting peer.example:000022 (168 differing corpus entries). The owning case "port over length bound" uses a 7-character port, so it pins the bound to [5,6] not to 5; the reviewed baseline patch had "padded port" at 6 characters, which killed this mutant, and rev2 replaced it. No security exposure (only 6-char zero-padded ports already inside 1..65535), but AC requires each clause to redden when weakened. REWORK IS TEST-ONLY, one case: add "port at length bound": {endpoint: "peer.example:000022", reason: endpointRefusalPort, neighbour: "peer.example:22"} to endpointGrammarCases, re-run the mutant, and correct the LOGBOOK line claiming both survivors are equivalent. Leave every production file exactly as is. Full detail: BUG-260902-4ajzyz_review-verdict.md
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-7ef84f, pid=81511, exit=0)
Review rev2: production code confirmed correct and unbypassable - 171 hostile loads across 57 shapes and three schema versions all refused with the clause named, 28 mutants with 24 killed and 2 proven equivalent by a 4704-endpoint differential corpus, all gates green, coverage 94.7 percent.

ONE LIVE SURVIVOR, and it is a regression this revision introduced. admitEndpointPort narrowed from len(port) > 5 to > 6 keeps the whole suite green and admits peer.example:000022, with 168 differing corpus entries. Cause: the owning case port over length bound now uses a SEVEN-character port, so it pins the bound to the range 5 to 6 rather than to 5. The reviewed baseline patch had a six-character padded port that killed this mutant exactly, and rev2 replaced it.

This is the boundary-adjacency rule again, the same shape as the epoch coupling earlier in this Story: a bound proven at a distant value leaves the adjacent one unpinned, and the narrowed form is then indistinguishable from the original for every input the suite actually tries.

REWORK IS TEST-ONLY, one case. Add to endpointGrammarCases:
  port at length bound: endpoint peer.example:000022, reason endpointRefusalPort, neighbour peer.example:22
Re-run the mutant and confirm it now reddens. Leave every production file exactly as it is - the reviewer states the production code is correct.

Also correct the LOGBOOK line claiming both surviving mutants are equivalent. One of them was not, and a logbook that records a closure the suite does not have is what the next round trusts.

While you are there: check the other bound cases in the same table for the same shape. A case that sits two or more steps past a limit pins a range, not the limit.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 recommendation: restore the adjacent-boundary case this revision replaced, and correct a LOGBOOK equivalence claim that does not hold."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260902-327c41, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260902-327c41)
Review rev2 blocking finding closed, test-only. Added "port at length bound": peer.example:000022 (six bytes, decimal 22, inside 1..65535, all digits) with neighbour peer.example:22, so len(port) > 5 is the only clause that can refuse it. The reviewers exact mutant (> 5 narrowed to > 6) is now KILLED and reddens ONLY the two cases that own the clause. Kept "port over length bound" and "leading zero port" as instructed. Zero production changes: endpoint.go, validation.go, sshargs.go, schema_test.go, ssh_admission_test.go and README.md are byte-identical to the reviewed candidate tree.

Generalised as the orchestrator asked, because this is the third time in this Story a bound proved at a distant value left its edge unpinned. endpointBoundEdges + TestEveryDeclaredMeshEndpointBoundIsPinnedAtItsEdge name every numeric bound the grammar declares with the last admitted value and the first refused one - user length 64/65 and 1/0, host label length 63/64 and 1/0, port length 5/6 and 1/0, port value 65535/65536 and 1/0 - each driven through the production Load entry. Audit of every other bound in the table: no other gap; each was already adjacent. One bound provably cannot be pinned by adjacency and is recorded rather than papered over - the bracketed len(host) < 3 guard excludes exactly "[]", which net.ParseIP refuses anyway, and the shortest admitted bracketed host is [::] at 4 bytes.

18 single-clause mutants run at this tree from a harness written for this cycle, 17 killed and 1 equivalent, each reported with the exact leaf subtests it reddens. M04 (whitespace clause deleted) previously reported INVALID-BUILD and is now a real kill because the harness drops the unused unicode import with it; M11 kills by making production panic on an empty label and is reported as such rather than as a clean assertion failure. The one survivor - len(port) < 1 dropped, left to number >= 1 - is equivalent by MEASUREMENT: 0 of 9,120 corpus rows differ, while the same corpus separates the > 5 -> > 6 narrowing on 178 rows, so it is discriminating rather than blind.

LOGBOOK corrected: the claim that both surviving mutants were equivalent was true of the 28 mutants the previous cycle ran, but that set contained only the DELETION of this bound, never its one-step narrowing. The entry now records the regression, its cause, the structural adjacency rule, and the unreachable bracket guard.

Gates all exit 0 at the committed tree, each run standalone: go build, go vet, gofmt -l (empty), go test ./... -count=1, go test ./... -cover, tracecheck global and sections 6.1-6.5, cataloggen -check. No generated-file drift. internal/config coverage 94.4 base -> 94.7 percent, unchanged by this test-only rework.

Kept to ONE signed commit: amended to 20f2586 on task-board/story/STORY-260902-cq7ivu, one commit past checkpoint 10aaa16, git verify-commit clean, worktree clean. Integration into trunk left to the orchestrator per the Story workspace contract.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-327c41, pid=97084, exit=0)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Confirm the port bound now pins to 5 rather than a range, that no production file moved, and that the corrected LOGBOOK equivalence claim holds."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260902-1be582, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260902-1be582)
Review rev3 ACCEPTED. Attacked, not read: 29 reviewer-run single-clause mutants in a disposable archive copy (27 killed, 2 proven equivalent - the empty-port length guard and the unreachable bracket-suffix guard, neither changes an admit/refuse decision or a refusal reason). Isolation held: no bound mutant reddened a clause other than its own. The carried-forward short-flag finding is discharged by a letter no named assertion covers: permitting -X reddens on the both-directions key-set pin alone (ssh_admission_test.go:364); -A/-E/-L would have reddened on their named assertions regardless. Dangling value flags: admitting one as a bare flag reddens all four value-taking letters. Independent 76-shape hostile probe (Unicode hyphen lookalikes, ZWSP/BOM prefixes, shell metacharacters, URI shapes, bracket and zone shapes, non-ASCII digits) admitted 12, every one an ordinary LDH host or user@host - no option shape, no whitespace. Reachability proven not argued: the same six hostile endpoints load with err=nil 18/18 on the pre-fix base across all three pinned readers and are refused 0/18 at the candidate; the packages own refusal-inventory audit confirms the new configError site is exercised from production. Gates re-run at the candidate tree: build, vet, gofmt, go test ./... , go test ./... -cover, tracecheck global and 6.1-6.5, cataloggen -check all exit 0, no generated drift. internal/config coverage measured by review at both ends: 94.4 percent at base 10aaa16 -> 94.7 percent. Production code is byte-identical to the accepted BUG-260902-3c7ovg patch; rev2/rev3 rework is test and documentation only and strictly strengthens the evidence. Recorded not blocking: the grammar deliberately narrows a previously unbounded field (underscore hosts, trailing-dot FQDNs, bare IPv6, %zone) as a declared reviewed fail-closed decision, and admitEndpointHostLabel indexes label[0] behind its own length guard so removing that guard panics rather than mis-admitting. No commit_ack supplied - the orchestrator owns the checkpoint and the done transition.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-1be582, pid=11613, exit=0)
Rev 3 was ACCEPTED and nothing is wrong with it. Trunk then advanced when the record-schema Story landed, and both that change and this one edit LOGBOOK.md, so the board refuses to integrate a combination nobody has looked at. That refusal is correct.

Refresh the base onto current trunk and resolve the LOGBOOK overlap deliberately - both entries belong, neither replaces the other. Say in your handoff what the combined section reads like; that sentence is the only genuinely new work here.

Everything else must come back byte-identical to the accepted rev-3 tree. If reapplying changes anything outside LOGBOOK.md, stop and report it rather than absorbing it: the acceptance rests on that exact tree, including the port-at-length-bound case that pins the bound to 5 rather than to a range, and the -X letter that discharges the short-flag pin.

Keep the leaf to one signed commit.

## Precondition Resources
- [BUG-260902-4ajzyz_accepted-work.patch](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_accepted-work.patch) — Accepted and reviewed work: 7 files, +571/-2. Reapply, do not reimplement.

## Outcome Resources
- [BUG-260902-4ajzyz_spawn-log_-implementer--developer--claude-_RUN-260902-c50f0b.log](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_spawn-log_-implementer--developer--claude-_RUN-260902-c50f0b.log) — System spawn log captured by task-board
- [BUG-260902-4ajzyz_evidence.md](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_evidence.md) — Reapply evidence: hostile-endpoint probe before/after, 21 mutants, gates, coverage
- [BUG-260902-4ajzyz_audit-probe-before-01.log](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_audit-probe-before-01.log) — 34-shape hostile-endpoint probe against the pre-fix base 10aaa16: 32 admitted
- [BUG-260902-4ajzyz_audit-probe-after-01.log](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_audit-probe-after-01.log) — Same probe on the fixed tree: all 34 refused, clause named
- [BUG-260902-4ajzyz_mutants-01.log](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_mutants-01.log) — 21 single-clause mutants: 19 killed, 2 proven equivalent
- [BUG-260902-4ajzyz_mutants.py](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_mutants.py) — Mutant harness used to produce mutants-01.log
- [BUG-260902-4ajzyz_go-test-cover-01.log](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_go-test-cover-01.log) — Repository coverage run, internal/config 94.6%
- [BUG-260902-4ajzyz_change-request_rev1.patch](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_change-request_rev1.patch) — Change Request CR-BUG-260902-4ajzyz-1 revision 1 candidate patch (repository_delta=present, 7 changed paths)
- [BUG-260902-4ajzyz_change-request_rev1-validation.log](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_change-request_rev1-validation.log) — Change Request CR-BUG-260902-4ajzyz-1 revision 1 bounded validation log
- [BUG-260902-4ajzyz_spawn-log_-reviewer--reviewer--claude-_RUN-260902-8d07cc.log](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_spawn-log_-reviewer--reviewer--claude-_RUN-260902-8d07cc.log) — System spawn log captured by task-board
- [BUG-260902-4ajzyz_review-verdict.md](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_review-verdict.md) — Reviewer verdict rev3: ACCEPTED. 29 reviewer-run mutants (27 killed, 2 equivalent), independent 76-shape hostile probe, cross-reader bypass check 18/18 admitted at base vs 0/18 after fix, all gates exit 0, internal/config coverage 94.4%->94.7%.
- [BUG-260902-4ajzyz_review-mutants-02.log](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_review-mutants-02.log) — Reviewer round-2 mutants: natural-refactor and narrowing shapes; n06 (@-carve-out) survived
- [BUG-260902-4ajzyz_review-mutate2.sh](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_review-mutate2.sh) — Reviewer mutant harness round 2, reproduces the surviving @-carve-out mutant
- [BUG-260902-4ajzyz_spawn-log_-implementer--developer--claude-_RUN-260902-120fd0.log](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_spawn-log_-implementer--developer--claude-_RUN-260902-120fd0.log) — System spawn log captured by task-board
- [BUG-260902-4ajzyz_rework-evidence.md](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_rework-evidence.md) — Review-cycle rework: blocking finding closed, isolation rule applied across the endpoint grammar, 28 mutants with 2 differentially-proven equivalents, gate results
- [BUG-260902-4ajzyz_mutants-04.log](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_mutants-04.log) — 28-mutant run against the committed suite: 26 killed with the reddened tests named, 2 equivalent
- [BUG-260902-4ajzyz_mutants2.py](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_mutants2.py) — Mutation harness: single-clause mutants over endpoint.go, sshargs.go and the validateMesh wiring; reports compile-failure mutants as INVALID-BUILD
- [BUG-260902-4ajzyz_change-request_rev2.patch](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_change-request_rev2.patch) — Change Request CR-BUG-260902-4ajzyz-2 revision 2 candidate patch (repository_delta=present, 7 changed paths)
- [BUG-260902-4ajzyz_change-request_rev2-validation.log](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_change-request_rev2-validation.log) — Change Request CR-BUG-260902-4ajzyz-2 revision 2 bounded validation log
- [BUG-260902-4ajzyz_spawn-log_-reviewer--reviewer--claude-_RUN-260902-7ef84f.log](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_spawn-log_-reviewer--reviewer--claude-_RUN-260902-7ef84f.log) — System spawn log captured by task-board
- [BUG-260902-4ajzyz_review-mutants.log](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_review-mutants.log) — Review evidence: review-mutants.log
- [BUG-260902-4ajzyz_review-hostile-probe.log](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_review-hostile-probe.log) — Review evidence: review-hostile-probe.log
- [BUG-260902-4ajzyz_review-gates.log](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_review-gates.log) — Review evidence: review-gates.log
- [BUG-260902-4ajzyz_review-equivalence.txt](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_review-equivalence.txt) — Review evidence: review-equivalence.txt
- [BUG-260902-4ajzyz_spawn-log_-implementer--developer--claude-_RUN-260902-327c41.log](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_spawn-log_-implementer--developer--claude-_RUN-260902-327c41.log) — System spawn log captured by task-board
- [BUG-260902-4ajzyz_rework-rev2-evidence.md](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_rework-rev2-evidence.md) — rev2 rework: port length bound pinned at its edge, boundary adjacency made structural, 18 mutants with the reviewer's survivor killed, all gates exit 0
- [BUG-260902-4ajzyz_rework-rev2-mutants.log](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_rework-rev2-mutants.log) — 15-mutant harness run at the reworked tree with the reddened leaf subtests per mutant
- [BUG-260902-4ajzyz_rework-rev2-gates.log](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_rework-rev2-gates.log) — build, vet, gofmt, test, cover, tracecheck global and 6.1-6.5, cataloggen -check at the committed tree; all rc=0
- [BUG-260902-4ajzyz_change-request_rev3.patch](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_change-request_rev3.patch) — Change Request CR-BUG-260902-4ajzyz-3 revision 3 candidate patch (repository_delta=present, 7 changed paths)
- [BUG-260902-4ajzyz_change-request_rev3-validation.log](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_change-request_rev3-validation.log) — Change Request CR-BUG-260902-4ajzyz-3 revision 3 bounded validation log
- [BUG-260902-4ajzyz_spawn-log_-reviewer--reviewer--claude-_RUN-260902-1be582.log](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_spawn-log_-reviewer--reviewer--claude-_RUN-260902-1be582.log) — System spawn log captured by task-board
- [BUG-260902-4ajzyz_review-verdict-rev3.md](file://BUG-260902-4ajzyz/BUG-260902-4ajzyz_review-verdict-rev3.md) — Reviewer verdict for CR rev3: ACCEPTED. 29 reviewer-run mutants (27 killed, 2 equivalent), independent 76-shape hostile probe, cross-reader bypass check 18/18 admitted at base vs 0/18 at candidate, all gates exit 0, internal/config coverage 94.4->94.7 percent.

## Created
2026-09-02T13:03:25Z

## Last Update
2026-09-02T14:17:08Z

## Assigned To
[reviewer] reviewer (claude)
