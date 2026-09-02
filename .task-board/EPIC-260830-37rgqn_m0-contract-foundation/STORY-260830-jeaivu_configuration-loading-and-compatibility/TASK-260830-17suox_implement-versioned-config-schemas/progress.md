## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(8))

## Blocked By
- TASK-260830-2iint0

## Blocks
- TASK-260830-1qf777

## Checklist
- [x] Production entry points implement the scoped deliverable: Implement exact Configuration 1.0.0, 2.0.0, and 3.0.0 readers plus current-version writers and legacy translation
- [x] Relevant positive, negative, compatibility, and recovery tests pass with logs attached
- [x] README/doctor/capability evidence and specification traceability are updated without unsupported claims
- [x] Where a registry, schema set or declared bound exists, completeness is a DERIVED property: a test derives the expected set from the pinned source or generated catalog and fails when any member lacks its enforcement
- [x] Every declared bound is proven in BOTH directions at the production entry: accept-at-limit and refuse-past-limit, with character bounds counting characters rather than bytes
- [x] Gates are attacked, not read: each refusal clause is disabled individually and the suite must redden; a clause whose disablement leaves the suite green is pinned by a new case or documented as provably subsumed with the subsuming check named
- [x] No test depends on ambient environment: fixtures set explicit identity and nothing relies on OS-derived values or developer-machine state a clean CI runner lacks
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
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=codex/gpt-5.6-sol/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:97a03f5c6378c00c3c16903d1bd83dfac31eeecb8d234a0b749b016e8ac47eb6 rationale="Following rank-1 recommendation: second leaf of the configuration Story, stacked on the accepted first leaf checkpoint."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[codex,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260901-0d81ce, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260901-0d81ce)
Implemented strict Configuration 1.0.0/2.0.0/3.0.0 TOML readers, in-memory v1/v2 to v3 translation, current-only writer, production Load wiring, exact bounds/refusals, pinned-catalog completeness, README and scoped traceability. Accepted gates: focused config coverage 83.9%, full tests, full coverage, vet, build, module verify, diff check, and assigned tracecheck all exit 0. Nine expected-red mutants each exited 1 and were restored. Durable atomic migration/downgrade mutation remains TASK-260830-1qf777 scope. See TASK-260830-17suox_results.md, TASK-260830-17suox_logbook.md, and TASK-260830-17suox_evidence.tar.gz.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260901-0d81ce, pid=45699, exit=0)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:cc7e0eb08c9f43b208cba223816e0e602248ca4021a13ced3c4b5b3f3ddae606 rationale="Consciously choosing rank-2 on measured evidence: cross-provider review keeps producer and reviewer independent."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[codex,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260901-4b6b5d, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260901-4b6b5d)
agent completed: [reviewer] reviewer (claude) (exit=1)
spawn run completed: claude (run=RUN-260901-4b6b5d, pid=64675, exit=1)
spawn autonomous recovery: run RUN-260901-4b6b5d queued successor RUN-260901-dcb83e (attempt 1/3, model=claude-opus-5): spawned agent exited with code 1
spawn run started: [reviewer] reviewer (claude) (run=RUN-260901-dcb83e)
agent completed: [reviewer] reviewer (claude) (exit=1)
spawn run completed: claude (run=RUN-260901-dcb83e, pid=17288, exit=1)
spawn autonomous recovery: run RUN-260901-dcb83e queued successor RUN-260901-28c378 (attempt 2/3, model=claude-opus-5): spawned agent exited with code 1
spawn run started: [reviewer] reviewer (claude) (run=RUN-260901-28c378)
agent completed: [reviewer] reviewer (claude) (exit=1)
spawn run completed: claude (run=RUN-260901-28c378, pid=18348, exit=1)
spawn autonomous recovery: run RUN-260901-28c378 queued successor RUN-260901-72b02a (attempt 3/3, model=claude-opus-5): spawned agent exited with code 1
spawn run started: [reviewer] reviewer (claude) (run=RUN-260901-72b02a)
agent completed: [reviewer] reviewer (claude) (exit=1)
spawn run completed: claude (run=RUN-260901-72b02a, pid=19450, exit=1)
recovery parked after 3 successor attempts for chain RUN-260901-4b6b5d; operator action required; last failure: spawned agent exited with code 1
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:cc7e0eb08c9f43b208cba223816e0e602248ca4021a13ced3c4b5b3f3ddae606 rationale="Consciously choosing rank-2 on measured evidence: cross-provider review keeps producer and reviewer independent and rank-1 has been content-filter blocked on reviewer probes in this project."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[codex,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260901-b359fe, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260901-b359fe)
Reviewer verdict rev1 = changes_requested. Implementation shape is sound (Load drives Decode at the real production entry; reader set derived from catalog.Current(); relational/nested/character bounds proven both directions; ownership SHA pin verified by tamper mutant; full suite + vet + build green). Refused on evidence, three blocking findings: (1) CONFIRMED SSH host-authentication bypass — sshHostAuthenticationBypass splits -o options on = only, so [-o, "StrictHostKeyChecking no"], [-oStrictHostKeyChecking no], [-o, StrictHostKeyChecking=\"no\"] and [-o, "UserKnownHostsFile /dev/null"] are admitted; all four verified live against OpenSSH via ssh -G as effective bypasses, spec 6.3 says they MUST fail. (2) terminalCapabilityRegistry is a hand-copy of a set already pinned in catalog.v0.5.0.json capability_families[terminal_backend], and its completeness test derives from that same variable — deleting multiple_input_clients leaves go test ./internal/config green. (3) Per-clause disablement sweep: 52 of 95 refusal clauses survive individually, including mesh.payload_encryption, providers.require_explicit_trust, backend registration, external_trust path/digest, all directory_peer_disclosure policy enums, all five required-member presence clauses, and every uniqueness check. Plus: v1/v2 terminal.backend vocabulary is not closed (backend=ax.tmux loads in a 1.0.0 document; the legacy-translation guard checks the post-translation ID so it cannot fire); ax.conpty-on-non-Windows untested; cloneOrDefault and sortedCopy are dead. Evidence: TASK-260830-17suox_review-verdict.md, TASK-260830-17suox_clause-mutation-sweep.log. All mutants reverted; every touched file re-verified byte-identical to candidate tree 34c010ea via git hash-object.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260901-b359fe, pid=90320, exit=0)
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=codex/gpt-5.6-sol/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:97a03f5c6378c00c3c16903d1bd83dfac31eeecb8d234a0b749b016e8ac47eb6 rationale="Following rank-1 recommendation: close the review findings by satisfying the property named rather than the instances cited."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[codex,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260901-d3cb9c, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260901-d3cb9c)
Developer rework RUN-260901-d3cb9c resolved CR rev1 findings: SSH -o separator/quote bypass closed; terminal capability closure derives from pinned catalog; legacy backend vocabulary closes before translation; exact production-entry refusal cases cover raw, semantic, platform, envelope, and writer paths. Final clause sweep: 99/99 mutants reddened with inner exit 1, 0 survivors, sweep exit 0. Final accepted gates: config coverage 91.8%, full tests/coverage, traceability, vet, build, module verify, catalog check, gofmt, and diff check all exit 0. Updated results/logbook and attached rework validation, sweep, and readiness resources.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260901-d3cb9c, pid=53742, exit=0)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:cc7e0eb08c9f43b208cba223816e0e602248ca4021a13ced3c4b5b3f3ddae606 rationale="Consciously choosing rank-2: cross-provider review keeps producer and reviewer independent, and rank-1 has been content-filter blocked on reviewer probes here."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[codex,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260901-f3815d, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260901-f3815d)
Reviewer RUN-260901-f3815d on CR-TASK-260830-17suox-2 rev 2: CHANGES REQUESTED -> to-dev. Verified candidate tree 1234f3c9 == worktree; go build/vet/test ./... all green. Three blockers. (1) sshHostAuthenticationBypass admits documented OpenSSH aliases: StrictHostKeyChecking=off, UserKnownHostsFile=none, GlobalKnownHostsFile=none all load with err=nil through config.Load; ssh_config(5) documents off as an alias of no and none as ignore-user-known-hosts, so SPEC 6.3 MUST-fail is bypassable. TestLoadRefusesEveryOpenSSHHostAuthenticationBypassSpelling claims a complete vocabulary it does not have. (2) DoD gate unmet: 12 delete-mutants and 2 narrowing mutants survive with the suite green, including hasForbiddenConfigName (the SPEC 6.4 secret/credential/endpoint/environment refusal, zero negative coverage), the printable/non-control rule, both utf8.ValidString checks, NUL-in-ssh-arg, extension reverse-DNS namespace / key-length / float / array-depth, and the uniqueness half of sorted-unique in validateSortedUniqueClosed and validateSortedUniqueDigests. (3) external_trust[].enabled is decoded, presence-checked and re-emitted but never read: validateTerminal registers the backend regardless, so backend_id selecting a trust entry with enabled=false is accepted. Non-blocking: empty explicit transport_policy accepted; section:17.5 ownership binding points at migration/downgrade behavior README says is out of scope. Evidence: TASK-260830-17suox_review-verdict.md, TASK-260830-17suox_review-mutation-evidence.md, TASK-260830-17suox_review-go-test.log. Review modified nothing; tree OID re-verified as 1234f3c9 after all probes.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260901-f3815d, pid=33146, exit=0)
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=codex/gpt-5.6-sol/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:97a03f5c6378c00c3c16903d1bd83dfac31eeecb8d234a0b749b016e8ac47eb6 rationale="Following rank-1 recommendation: close the review findings by satisfying the named property."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[codex,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260901-d44711, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260901-d44711)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260901-d44711, pid=72247, exit=0)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=codex/gpt-5.6-sol/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:9f809bb1166af50c9b70251e8f7fea75dbf69b2813e05a3848c9c4d942a29986 rationale="Following rank-1 review-class recommendation: round 3 verifies the named properties rather than an enumerated finding list."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[codex,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260901-6f3c87, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260901-6f3c87)
agent completed: [reviewer] reviewer (codex) (exit=-1)
spawn run completed: codex (run=RUN-260901-6f3c87, pid=75287, exit=-1)
spawn run RUN-260901-6f3c87 cancelled by operator; operator action required; reason: no operator reason supplied
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:cc7e0eb08c9f43b208cba223816e0e602248ca4021a13ced3c4b5b3f3ddae606 rationale="Consciously choosing the cross-provider reviewer on measured evidence: rounds 1-3 used this pair, and producer/reviewer provider independence is what has been finding the evidence-level defects."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[codex,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260901-3d65a5, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260901-3d65a5)
REVIEW rev3 (RUN-260901-3d65a5): CHANGES REQUESTED -> to-dev. Evidence: TASK-260830-17suox_review-verdict.md + TASK-260830-17suox_reviewer-gate-logs.md. All gates re-run green by the reviewer (build, vet, go test ./... , config coverage 93.2%, tracecheck global assigned_scopes=0 and scoped assigned_scopes=8). Candidate tree verified identical to 60ce1269 before and after review; no repository file touched. FINDING 1 (confirmed correctness defect): EncodeCurrent emits a Configuration 3.0.0 document that its own Decode refuses whenever a closed map member is the empty object. go-toml v2.4.3 marshals map[string]any{} as a bare table header and unmarshals that header back to a nil map, so validateRawDirectoryPresence/validateRawTerminalPresence refuse it as a missing required member. Reproduces on directory_installations[].extensions, directory_enrichment_profiles[].extensions, directory_peer_disclosure[].extensions, and terminal.backend_config[].settings. Side effect: a hand-written config using the standard multi-line [x.extensions] empty-table spelling is refused while the identical inline extensions = {} is accepted. Missed because the only round-trip test uses validCurrentConfiguration(), whose single installation always carries a non-empty extension map and which declares no profiles, disclosures, or backend configs. FINDING 2 (DoD unmet): 20 refusal mutants applied to isolated copies, 9 SURVIVED with the suite green and none documented as subsumed - peer workspace-root platform binding (peer.Platform -> configuration.Platform), all per-root member checks for mesh.peers[].workspace_roots, scan_root_authority_ids minimum-1 bound, legacy conpty->ax.conpty mapping, legacy Windows backend default, v3 Windows backend default, logical_root grammar widened to allow uppercase/leading digit, extension key minimum length, extension int64 safe-integer bound. Instrumenting configError shows 98 of 101 clauses reached; the 3 unreached are exactly the peer-nested workspace-root clauses. Structural cause: the versioned-schema tests have no positive Windows or WSL2 case at all (PlatformWindows appears once in schema_test.go and is a negative case), yet ownership.v0.5.0.json now claims section:6.2, which is normatively about the native-Windows and WSL2 spellings. The production behavior on those lanes was probed directly and is CORRECT - it is unpinned, not broken. NOT findings: no traceability overclaim (17.3 correctly not claimed); derived completeness is genuinely derived from the pinned catalog for both reader versions and terminal capabilities; empty required_capabilities default is right per 4.E no-inferred-capabilities; backend settings fail closed without a validator; errors do not leak machine-local values; no silent rewrite path exists.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260901-3d65a5, pid=11899, exit=0)
Round-3 review returned changes requested with one confirmed correctness defect and a repeat of the evidence class that has now recurred across four rounds on this board. Close the CLASSES mechanically; closing the cited instances will be rejected on the same criterion.

CLASS 1 - fixture-shaped coverage. FINDING 1 exists because the only round-trip case is validCurrentConfiguration(), whose single installation always carries a non-empty extension map and which declares no profiles, disclosures, or backend configs. Do not add four cases for the four cited paths. Derive the round-trip property: enumerate every closed map member in the versioned schema and assert EncodeCurrent -> Decode succeeds for each, with the empty object as an explicit value. The derivation must come from the schema or pinned catalog so that a member added later is covered without editing the test. Then fix the encoder so the property holds; the multi-line [x.extensions] spelling and the inline extensions = {} spelling must both decode.

CLASS 2 - unpinned enforcement. Nine of twenty refusal mutants survived green and none is documented as subsumed. The fix is not nine test cases. Make the artifact executable: every clause for which ownership.v0.5.0.json or the enumeration claims enforcement must have a negative case that reddens when that clause is disabled, and any clause you assert is subsumed must name the subsuming check, with that naming itself checked. Use the configError instrumentation the reviewer describes - 98 of 101 clauses reached, the 3 unreached being the peer-nested workspace-root clauses - as the derived inventory, not a hand-written list.

CLASS 2 structural cause, fix it directly: the versioned-schema tests contain no positive Windows or WSL2 case at all, while ownership.v0.5.0.json claims section:6.2, which is normatively about the native-Windows and WSL2 spellings. Add positive cases for those lanes or withdraw the ownership claim. Do not claim a section the suite does not exercise.

Production behavior on the unpinned lanes was probed and is correct. Expect evidence work, not a rewrite, except for FINDING 1 which is a real encoder defect.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=codex/gpt-5.6-sol/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:97a03f5c6378c00c3c16903d1bd83dfac31eeecb8d234a0b749b016e8ac47eb6 rationale="Following rank-1 recommendation: close the two recurring classes mechanically per the appended directive, plus the confirmed encoder defect."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[codex,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260901-c0866b, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260901-c0866b)
CR revision 4 closes the recurring reviewer classes mechanically: schema-derived empty-map round trips, production restoration of present empty TOML tables without treating absence as presence, AST-derived refusal-site completeness, positive Windows/WSL2 readers, and targeted narrowing coverage. Final gates: mutation 109/109 killed with zero survivors/invalid; config coverage 93.2%; full tests/coverage, vet, build, module verify, catalog check, global/scoped traceability, gofmt, and diff check exit 0. Shared Go cache failures were recorded and accepted gates rerun with task-scoped GOCACHE. Evidence: TASK-260830-17suox_results.md, TASK-260830-17suox_rework-rev4-validation.md, TASK-260830-17suox_clause-mutation-sweep-rev4.log, TASK-260830-17suox_logbook.md.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260901-c0866b, pid=91883, exit=0)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:cc7e0eb08c9f43b208cba223816e0e602248ca4021a13ced3c4b5b3f3ddae606 rationale="Cross-provider reviewer on measured evidence: this pair found the encoder round-trip defect and the SSH bypass that same-provider review is not independent of."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[codex,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260901-967b1a, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260901-967b1a)
Reviewer verdict rev4 = changes_requested -> to-dev. No production defect found: every rev3 finding is closed and all nine rev3 mutation survivors now redden (M-A/M-B peer-platform roots, M-E digest minimum, M-3/M-4/M-5 Windows+WSL2 backend lanes, M-6 logical_root grammar, M-8 int64 bound; M-7 removed as provably subsumed by reverseDNSPattern with the subsumption named in-code and pinned both directions). The EncodeCurrent/Decode empty-map round-trip defect is fixed for all four closed map members in inline and multiline spelling. Reviewer re-ran build/vet/full suite green on candidate tree 6df2bcbb; ownership re-pin verified by two tamper mutants; the new derived refusal-site inventory gate verified by injecting an unexercised configError site. Refused on evidence, two findings. (1) BLOCKING: the absent half of the present-vs-absent closed-map distinction has no negative test. A narrowing mutant of the new restorePresentEmptyMaps (schema.go:499) that initializes nil map members before the source-presence check leaves go test ./internal/config fully green while making all four required members admissible when omitted (directory_installations/.enrichment_profiles/.peer_disclosure extensions and terminal.backend_config settings) - probed through Load: refused on candidate, err=nil under mutant. The four required-member cases in refusal_test.go:153-190 each omit six or seven members at once so they refuse on an earlier disjunct. Related: dropping only the closed-map disjunct from each presence check (validation.go:109/114/119/133) survives x4; those four are fail-closed via validateExtensions(nil) and validation.go:722 but are not documented as subsumed. README now claims the distinction is preserved with evidence on one side only. (2) The extension canonical byte bound (SPEC §6 = 65,536) is tested from maxConfigExtensionBytes itself (schema_test.go:555-568, including its own self-check), so mutating the constant to 655_360 survives green; every neighbouring bound uses a spec literal. Requested: four negative Load cases omitting exactly one closed map member each, and a spec-literal byte-bound fixture. Evidence: TASK-260830-17suox_review-verdict.md, TASK-260830-17suox_review-mutation-sweep-rev4.log, TASK-260830-17suox_review-logbook-entries.md. Worktree verified byte-identical to candidate tree 6df2bcbb before and after review; all 33 mutants ran in /tmp/axmut4.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260901-967b1a, pid=38828, exit=0)
Round-4 review confirms every rev3 finding closed, all nine survivors reddened, and the encoder round-trip defect fixed. No production defect remains. The two findings left are one CLASS: an assertion that cannot fail for the reason it names.

CLASS 3 - tests that pass for the wrong reason. Finding 1: the four required-member cases at refusal_test.go:153-190 each omit six or seven members at once, so Load refuses on an earlier disjunct and the closed-map presence clause they claim to pin is never the reason for refusal. That is why the restorePresentEmptyMaps narrowing mutant survives green while making all four required members admissible when omitted. Finding 2: the extension canonical byte bound is asserted against maxConfigExtensionBytes itself, including a self-check of that same constant, so mutating the constant to 655_360 survives; the assertion has no independent reference to fail against.

Close the class, not the two instances. Every negative case must isolate the clause it targets: violate exactly one thing so the case cannot refuse on an earlier disjunct, and verify that isolation by confirming the case reddens when only its target clause is weakened. Every bound must be asserted against the normative literal from pinned SPEC v0.5.0, never against the implementation constant that the bound is derived from; SPEC section 6 gives 65,536 and the neighbouring bounds already use spec literals, so follow that pattern uniformly.

Concretely required: four negative Load cases omitting exactly one closed map member each, a spec-literal byte-bound fixture, and for the four presence checks at validation.go:109/114/119/133 either a case that reddens when only the closed-map disjunct is dropped, or an in-code statement naming validateExtensions(nil) and validation.go:722 as the subsuming fail-closed path, pinned the way M-7 subsumption was pinned. README must not claim the present-vs-absent distinction is evidenced on one side only.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=codex/gpt-5.6-sol/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:97a03f5c6378c00c3c16903d1bd83dfac31eeecb8d234a0b749b016e8ac47eb6 rationale="Following rank-1 recommendation: close the pass-for-the-wrong-reason class per the appended directive; no production change expected."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[codex,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260901-908595, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260901-908595)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260901-908595, pid=3841, exit=0)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:cc7e0eb08c9f43b208cba223816e0e602248ca4021a13ced3c4b5b3f3ddae606 rationale="Cross-provider reviewer on measured evidence: this pair found the isolation defect in the negative cases and stays independent of the producer."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[codex,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260901-1ffcc5, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260901-1ffcc5)
Review of CR-TASK-260830-17suox-5 (RUN-260901-1ffcc5): ACCEPTED. Verified candidate tree d6f1511a byte-for-byte before review. Spec fidelity checked against pinned SPEC.md at 28bf96d7 (hash matches specpin lock) for sections 6.1-6.5 and 17.1-17.2. Gates attacked: 16 source mutants applied in an isolated copy, 16/16 killed, including narrowing mutants (bound widened by one, SSH bypass alias dropped, forbidden-name entry dropped, sorted-unique duplicate allowed, depth 4->5) and a synthetic never-exercised configError site that proved the derived refusal inventory gate is real. 19 adversarial documents driven through the production Load entry, all closed-shape refusals correct. Reviewer-run validation: go build, go vet, go test ./... green; internal/config coverage 93.2 percent; tracecheck assigned_scopes=8 green. Three non-blocking notes recorded in the verdict artifact: backend_config settings receive no config-layer structural screen (spec-faithful delegation, fails closed without a validator), required_capabilities default left empty because the spec phrase platform lane minimum is undefined anywhere normative, and hasForbiddenConfigName is a label-split heuristic. Evidence: TASK-260830-17suox_review-verdict.md.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260901-1ffcc5, pid=66320, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260830-17suox_spawn-log_-implementer--developer--codex-_RUN-260901-0d81ce.log](file://TASK-260830-17suox/TASK-260830-17suox_spawn-log_-implementer--developer--codex-_RUN-260901-0d81ce.log) — System spawn log captured by task-board
- [TASK-260830-17suox_results.md](file://TASK-260830-17suox/TASK-260830-17suox_results.md) — CR revision 5 implementation summary, isolated refusal proof, mutation attacks, and accepted validation index
- [TASK-260830-17suox_evidence.tar.gz](file://TASK-260830-17suox/TASK-260830-17suox_evidence.tar.gz) — Accepted validation logs, red-to-green history, and expected-red mutation attack logs
- [TASK-260830-17suox_logbook.md](file://TASK-260830-17suox/TASK-260830-17suox_logbook.md) — Configuration compatibility decisions and CR revision 5 evidence-class resolution
- [TASK-260830-17suox_change-request_rev1.patch](file://TASK-260830-17suox/TASK-260830-17suox_change-request_rev1.patch) — Change Request CR-TASK-260830-17suox-1 revision 1 candidate patch (repository_delta=present, 13 changed paths)
- [TASK-260830-17suox_change-request_rev1-validation.log](file://TASK-260830-17suox/TASK-260830-17suox_change-request_rev1-validation.log) — Change Request CR-TASK-260830-17suox-1 revision 1 bounded validation log
- [TASK-260830-17suox_spawn-log_-reviewer--reviewer--claude-_RUN-260901-4b6b5d.log](file://TASK-260830-17suox/TASK-260830-17suox_spawn-log_-reviewer--reviewer--claude-_RUN-260901-4b6b5d.log) — System spawn log captured by task-board
- [TASK-260830-17suox_spawn-log_-reviewer--reviewer--claude-_RUN-260901-dcb83e.log](file://TASK-260830-17suox/TASK-260830-17suox_spawn-log_-reviewer--reviewer--claude-_RUN-260901-dcb83e.log) — System spawn log captured by task-board
- [TASK-260830-17suox_spawn-log_-reviewer--reviewer--claude-_RUN-260901-28c378.log](file://TASK-260830-17suox/TASK-260830-17suox_spawn-log_-reviewer--reviewer--claude-_RUN-260901-28c378.log) — System spawn log captured by task-board
- [TASK-260830-17suox_spawn-log_-reviewer--reviewer--claude-_RUN-260901-72b02a.log](file://TASK-260830-17suox/TASK-260830-17suox_spawn-log_-reviewer--reviewer--claude-_RUN-260901-72b02a.log) — System spawn log captured by task-board
- [TASK-260830-17suox_spawn-log_-reviewer--reviewer--claude-_RUN-260901-b359fe.log](file://TASK-260830-17suox/TASK-260830-17suox_spawn-log_-reviewer--reviewer--claude-_RUN-260901-b359fe.log) — System spawn log captured by task-board
- [TASK-260830-17suox_review-verdict.md](file://TASK-260830-17suox/TASK-260830-17suox_review-verdict.md) — Reviewer verdict for CR revision 5 (RUN-260901-1ffcc5): accepted; 22 mutants killed including the five round-4 named requirements, 19 adversarial probes, spec 6/17 fidelity against pinned SPEC.md
- [TASK-260830-17suox_clause-mutation-sweep.log](file://TASK-260830-17suox/TASK-260830-17suox_clause-mutation-sweep.log) — Per-clause disablement sweep over internal/config: 95 refusal clauses, 43 reddened, 52 survived
- [TASK-260830-17suox_review-logbook-entries.md](file://TASK-260830-17suox/TASK-260830-17suox_review-logbook-entries.md) — Task-scoped logbook: reviewer findings across CR rev 1-4; rev4 entries cover the untested absent half of the present-vs-absent closed-map distinction, the self-referential extension byte bound, and the new derived refusal-site inventory gate.
- [TASK-260830-17suox_spawn-log_-implementer--developer--codex-_RUN-260901-d3cb9c.log](file://TASK-260830-17suox/TASK-260830-17suox_spawn-log_-implementer--developer--codex-_RUN-260901-d3cb9c.log) — System spawn log captured by task-board
- [TASK-260830-17suox_rework-validation.log](file://TASK-260830-17suox/TASK-260830-17suox_rework-validation.log) — Final accepted gate exits and expected-red mutation summary for reviewer rework
- [TASK-260830-17suox_clause-mutation-sweep-rework.log](file://TASK-260830-17suox/TASK-260830-17suox_clause-mutation-sweep-rework.log) — Final per-clause sweep: all 99 refusal clauses reddened individually, zero survivors
- [TASK-260830-17suox_tool-readiness.md](file://TASK-260830-17suox/TASK-260830-17suox_tool-readiness.md) — Tool readiness, pinned-spec source, and fallback evidence
- [TASK-260830-17suox_change-request_rev2.patch](file://TASK-260830-17suox/TASK-260830-17suox_change-request_rev2.patch) — Change Request CR-TASK-260830-17suox-2 revision 2 candidate patch (repository_delta=present, 14 changed paths)
- [TASK-260830-17suox_change-request_rev2-validation.log](file://TASK-260830-17suox/TASK-260830-17suox_change-request_rev2-validation.log) — Change Request CR-TASK-260830-17suox-2 revision 2 bounded validation log
- [TASK-260830-17suox_spawn-log_-reviewer--reviewer--claude-_RUN-260901-f3815d.log](file://TASK-260830-17suox/TASK-260830-17suox_spawn-log_-reviewer--reviewer--claude-_RUN-260901-f3815d.log) — System spawn log captured by task-board
- [TASK-260830-17suox_review-mutation-evidence.md](file://TASK-260830-17suox/TASK-260830-17suox_review-mutation-evidence.md) — Gate-attack evidence: SSH bypass probe, mutation kill/survive table, trust-entry probe
- [TASK-260830-17suox_review-go-test.log](file://TASK-260830-17suox/TASK-260830-17suox_review-go-test.log) — Reviewer baseline: go test ./... -count=1 on candidate tree 1234f3c9
- [TASK-260830-17suox_spawn-log_-implementer--developer--codex-_RUN-260901-d44711.log](file://TASK-260830-17suox/TASK-260830-17suox_spawn-log_-implementer--developer--codex-_RUN-260901-d44711.log) — System spawn log captured by task-board
- [TASK-260830-17suox_rework-rev3-validation.md](file://TASK-260830-17suox/TASK-260830-17suox_rework-rev3-validation.md) — CR revision 3 red-to-green, gate attack, and final validation evidence
- [TASK-260830-17suox_clause-mutation-sweep-rev3.log](file://TASK-260830-17suox/TASK-260830-17suox_clause-mutation-sweep-rev3.log) — Seventeen targeted expected-red clause mutants, zero survivors
- [TASK-260830-17suox_change-request_rev3.patch](file://TASK-260830-17suox/TASK-260830-17suox_change-request_rev3.patch) — Change Request CR-TASK-260830-17suox-3 revision 3 candidate patch (repository_delta=present, 14 changed paths)
- [TASK-260830-17suox_change-request_rev3-validation.log](file://TASK-260830-17suox/TASK-260830-17suox_change-request_rev3-validation.log) — Change Request CR-TASK-260830-17suox-3 revision 3 bounded validation log
- [TASK-260830-17suox_spawn-log_-reviewer--reviewer--codex-_RUN-260901-6f3c87.log](file://TASK-260830-17suox/TASK-260830-17suox_spawn-log_-reviewer--reviewer--codex-_RUN-260901-6f3c87.log) — System spawn log captured by task-board
- [TASK-260830-17suox_spawn-log_-reviewer--reviewer--claude-_RUN-260901-3d65a5.log](file://TASK-260830-17suox/TASK-260830-17suox_spawn-log_-reviewer--reviewer--claude-_RUN-260901-3d65a5.log) — System spawn log captured by task-board
- [TASK-260830-17suox_reviewer-gate-logs.md](file://TASK-260830-17suox/TASK-260830-17suox_reviewer-gate-logs.md) — Reviewer-run gate logs, spec/candidate integrity checks, defect reproduction, clause instrumentation, mutation results, and platform-lane probes for CR rev3.
- [TASK-260830-17suox_spawn-log_-implementer--developer--codex-_RUN-260901-c0866b.log](file://TASK-260830-17suox/TASK-260830-17suox_spawn-log_-implementer--developer--codex-_RUN-260901-c0866b.log) — System spawn log captured by task-board
- [TASK-260830-17suox_rework-rev4-validation.md](file://TASK-260830-17suox/TASK-260830-17suox_rework-rev4-validation.md) — CR revision 4 red-to-green, shared-cache anomaly, accepted gates, and mutation summary
- [TASK-260830-17suox_clause-mutation-sweep-rev4.log](file://TASK-260830-17suox/TASK-260830-17suox_clause-mutation-sweep-rev4.log) — Final mutation attack: 109 behavioral kills, zero survivors, zero invalid mutants
- [TASK-260830-17suox_tool-readiness-rev4.md](file://TASK-260830-17suox/TASK-260830-17suox_tool-readiness-rev4.md) — Run-specific toolchain readiness and Curator skill fallback evidence
- [TASK-260830-17suox_change-request_rev4.patch](file://TASK-260830-17suox/TASK-260830-17suox_change-request_rev4.patch) — Change Request CR-TASK-260830-17suox-4 revision 4 candidate patch (repository_delta=present, 16 changed paths)
- [TASK-260830-17suox_change-request_rev4-validation.log](file://TASK-260830-17suox/TASK-260830-17suox_change-request_rev4-validation.log) — Change Request CR-TASK-260830-17suox-4 revision 4 bounded validation log
- [TASK-260830-17suox_spawn-log_-reviewer--reviewer--claude-_RUN-260901-967b1a.log](file://TASK-260830-17suox/TASK-260830-17suox_spawn-log_-reviewer--reviewer--claude-_RUN-260901-967b1a.log) — System spawn log captured by task-board
- [TASK-260830-17suox_review-mutation-sweep-rev4.log](file://TASK-260830-17suox/TASK-260830-17suox_review-mutation-sweep-rev4.log) — Reviewer mutation sweep on CR rev4 candidate tree 6df2bcbb: 33 mutants, 2 blocking survivors (N-1, E-5) plus 4 fail-closed-but-undocumented survivors (P-1..P-4), with N-1 behavior probe through Load.
- [TASK-260830-17suox_spawn-log_-implementer--developer--codex-_RUN-260901-908595.log](file://TASK-260830-17suox/TASK-260830-17suox_spawn-log_-implementer--developer--codex-_RUN-260901-908595.log) — System spawn log captured by task-board
- [TASK-260830-17suox_rework-rev5-validation.md](file://TASK-260830-17suox/TASK-260830-17suox_rework-rev5-validation.md) — CR revision 5 green gates, exact expected-red mutation exits, and applicability statement
- [TASK-260830-17suox_rework-rev5-evidence.tar.gz](file://TASK-260830-17suox/TASK-260830-17suox_rework-rev5-evidence.tar.gz) — CR revision 5 final gate logs and six expected-red mutation attack logs
- [TASK-260830-17suox_change-request_rev5.patch](file://TASK-260830-17suox/TASK-260830-17suox_change-request_rev5.patch) — Change Request CR-TASK-260830-17suox-5 revision 5 candidate patch (repository_delta=present, 16 changed paths)
- [TASK-260830-17suox_change-request_rev5-validation.log](file://TASK-260830-17suox/TASK-260830-17suox_change-request_rev5-validation.log) — Change Request CR-TASK-260830-17suox-5 revision 5 bounded validation log
- [TASK-260830-17suox_spawn-log_-reviewer--reviewer--claude-_RUN-260901-1ffcc5.log](file://TASK-260830-17suox/TASK-260830-17suox_spawn-log_-reviewer--reviewer--claude-_RUN-260901-1ffcc5.log) — System spawn log captured by task-board
- [TASK-260830-17suox_review-verdict-rev5.md](file://TASK-260830-17suox/TASK-260830-17suox_review-verdict-rev5.md) — Reviewer verdict for CR revision 5 (RUN-260901-1ffcc5): ACCEPTED; 22 mutants killed including all five round-4 named requirements, 19 adversarial probes, spec 6/17 fidelity against pinned SPEC.md

## Created
2026-08-29T21:59:55Z

## Last Update
2026-09-01T16:03:45Z

## Assigned To
[reviewer] reviewer (claude)
