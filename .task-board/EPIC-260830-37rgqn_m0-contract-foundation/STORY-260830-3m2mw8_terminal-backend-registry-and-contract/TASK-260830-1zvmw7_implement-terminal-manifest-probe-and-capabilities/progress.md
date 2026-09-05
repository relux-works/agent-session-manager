## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(8))

## Blocked By
- TASK-260830-2xdt8t

## Blocks
- TASK-260830-1snnef

## Checklist
- [x] Production entry points implement the scoped deliverable: Implement closed manifest/probe schemas, generation-bound capability evidence, dependencies, expiry, and fail-closed admission
- [x] Relevant positive, negative, compatibility, and recovery tests pass with logs attached
- [x] README/doctor/capability evidence and specification traceability are updated without unsupported claims
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
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:cd50227265ca35d490f29a93c82a485be7dd82a62d95840afdbaf40590e6093e rationale="implementation class rank 1; leaf 2 implementation on a checkpointed leaf 1"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260904-e24452, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260904-e24452)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260904-e24452, pid=54538, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:ea5498eaa14293ca9063a299009604c3a4d14fb3b285e8a8d0685edcf4bf1cf7 rationale="review class rank 1; large new parsing/admission surface needs the strongest reviewer"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260904-4cecb2, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260904-4cecb2)
ROUND 1 REVIEW: CHANGES REQUESTED. Verdict artifact TASK-260830-1zvmw7_review-verdict.md.

Ratios: 64/96 mutants killed; 71/121 production refusal arms executed by the shipped suite; 0/29 leaf-2 refusal assertions check which arm fired; G-A aliasing mutant survives (1 of 1).

BLOCKING:
F1 correctness/spec: CheckOperation("manifest"|"probe") refuses unconditionally with terminal_backend_capability_unproven even with all 16 capabilities admitted. Spec 4.D gives both operations Capability dependencies: none, and neither row allows that error code. The sibling CapabilitiesForOperation documents the opposite model. Untested.
F2 evidence: four named TestReconcileRefusals cases (omitted manifest claim, static echo drift, protocol not member, evidence for false claim) land on the right arm at HEAD but shift to a downstream arm when their gate is deleted; suite stays green. Root enabler: every assertion is IsMismatch-only, and ~110 of 121 arms return the same code.
F3 evidence: checkManifestRecordBinding executable-digest half (the 6.5 untrusted-substitution gate on Registry.AdmitProbe) deletes clean with no test reaching it.
F4 evidence: expiry upper boundary one-sided; now == expires_at admits under a narrowing mutant.
F5 evidence: ParseProbe and ParseEvidence schema/schema_version/backend-identity/kind/digest gates all delete clean (M01-M09); only ParseManifest has those cases.
F9 DoD: README:1977 still claims this repository builds no TerminalBackend capability set - falsified by this leaf. No ownership.v0.5.0.json binding registered and no README update, unlike leaf 1.

NON-BLOCKING: F6 six unevidenced gate classes (duplicate halves of both sorted-unique checks, JSON depth cap, UTF-8 validity, signature scheme prefix, realm required-members). F7 four declared bounds never measured. F8 two unreachable arms (checkProbeIdentity L1550, parsePlatformList L950) and one inaccurate doc on checkEvidenceSet.

STRONG AREAS, do not regress: UnsignedEvidenceBytes 19/20 narrowing mutants killed (every dropped member plus domain separator and NUL); GenerationDigest 4/4; signature verification 2/2; evidence tuple/coverage/ID-set 11/12.

STATED BOUNDS: JCS canonicalization is unprovable with these fixtures (M71/M72 survive because the closed schemas have ASCII-only keys and no numbers, so Go map marshalling and RFC 8785 agree) - reported unknown, not covered. M92 and M95 are near-equivalent mutants. Coverage arm mapping uses max block hit count, so 71/121 is an upper bound.

Working tree verified byte-identical to the CR candidate before and after every probe; manifest.go sha256 re-checked after each battery. Repo suite green, vet/gofmt clean, package coverage 87.1% - accepted from the run, re-run here.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260904-4cecb2, pid=37117, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:cd50227265ca35d490f29a93c82a485be7dd82a62d95840afdbaf40590e6093e rationale="implementation class rank 1; rework of one production spec violation plus four gate classes"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260905-e03cb0, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260905-e03cb0)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260905-e03cb0, pid=11529, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:ea5498eaa14293ca9063a299009604c3a4d14fb3b285e8a8d0685edcf4bf1cf7 rationale="review class rank 1; round 2 must retest an arm-shift premise and check resurrections"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260905-e34b30, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260905-e34b30)
Round 2 review (CR rev 2): CHANGES REQUESTED. Evidence: TASK-260830-1zvmw7_review-verdict-round2.md + _review-round2-probe-log.txt + two probe scripts.

WHAT PASSED (re-measured, not accepted from the run). Working tree recomputed to tree-oid d99bf886 before and after every probe = the CR candidate; manifest.go sha256 re-verified after each battery; full suite green 14/14, coverage 88.6%, tracecheck exit 0 acceptance_cases=75.
- G-A: 17 gates deleted/narrowed (4 x §4.B, both executable-substitution sites, expiry boundary, 10 x closed-schema class). ALL 17 killed and ALL 17 fail with error=<nil> - no fixture shifts to a lower arm. The rework premise holds for the named gate set.
- Arm identity: 0/29 -> 85/85 executed arm SITES identity-asserted, 76/76 executed identities named. 85/119 sites executed (was 71/121). Bound: 19 executed sites share a detail string with another site, so the assertion names the rule not the site; 66 are file-unique.
- G-B: both round-1 batteries re-run plus 2 re-anchored mutants = 91/95 killed, ZERO resurrections. The 4 survivors are exactly M71/M72 (JCS still unprovable - unchanged by the rework, reported unknown) and M92/M95 (near-equivalents).
- G-C: round-1 positive control (parseClaim returns registry row slices) is now KILLED by TestParsedClaimSlicesShareNoRegistryBacking, with REVIEW-POISON shown inside capabilityRegistry rows. Round 1 it survived all 52 tests.
- F1-F9 all closed. F9 gate attacked 5 ways (bogus test decl, bogus production decl, gap prose edit without re-pin, case removal, digest revert) - all 5 killed.

BLOCKING H1. checkClaimRelation has 5 arms; the rework gave 4 of them narrowed fixtures. The fifth, case proved.Origin == OriginStatic && !exists -> mismatchf("probe static claim without manifest"), has 0 coverage hits and deletes cleanly with the suite green. I built the bypass through the real entry points (ParseManifest/ParseProbe/Reconcile): a probe declaring graceful_stop with origin=static while the manifest declares only headless_creation. At HEAD it refuses at the right arm with admitted=[]. With the arm neutered: err=<nil>, admitted=[graceful_stop headless_creation] - a capability the Manifest never declared is admitted. The omission loop cannot catch it; it checks the other direction. Self-minted-claim shape on the object this leaf exists to gate.

NON-BLOCKING H2: ParseEvidence platform vocabulary (evidence platform) narrows cleanly, 0 hits - the F5 clause-present-in-one-parser shape, in the one parser not extended. H3: parseClaim non-object guard (claim shape) narrows cleanly, 0 hits; refuses downstream as document members so still fail-closed, but arm-blind.

Fix is three table rows. Re-run both batteries plus the 17-gate probe after: target 91/95 held with the same four declared survivors and 17/17 still error=<nil>.

Observations, not blocking: section:4.B/4.C/4.D/6.5/7.A bindings still name internal/catalog ForRelease at coverage=unevidenced, and unevidenced=41 / clauses_discharged=17/403 are unchanged from leaf 1 - that under-claims rather than over-claims, which is the right direction. The ownership gap prose asserting the capability registry carries no Structured Error code is true (I verified the 16 names are disjoint from internal/axerror) but held by no test.

No repository file was modified by this review; the tree is byte-identical to the CR candidate.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260905-e34b30, pid=19946, exit=0)
spawn workload selection: class=implementation source=explicit policy=spawn.workload_classes pair=muse/muse-spark/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:cd50227265ca35d490f29a93c82a485be7dd82a62d95840afdbaf40590e6093e rationale="implementation class rank 1; three-row gate coverage fix on an otherwise accepted leaf"
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260905-1dbca2, max_parallel=20)
spawn run started: [implementer] developer (muse) (run=RUN-260905-1dbca2)
Round-3 rework in working tree, NOT committed per CR instructions. Three rows added to internal/terminalbackend/manifest_test.go (H1 static claim without manifest, H2 evidence platform, H3 claim shape); no production file touched. All three mutants KILLED (H1/H2 error=<nil>, H3 arm-blind at document members). Enumeration done independently via mismatchf/Detail extraction, not from the verdict list. Full suite green (14/14, exit 0), coverage 88.9%. Evidence: TASK-260830-1zvmw7_round3-three-rows.md. Logbook entry appended.
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260905-1dbca2, pid=90582, exit=0)
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:ea5498eaa14293ca9063a299009604c3a4d14fb3b285e8a8d0685edcf4bf1cf7 rationale="review class rank 1; round 3 must judge whether arm enumeration is mechanised or still manual"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[muse,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260905-68568a, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260905-68568a)
Round-3 review (CR rev 3): ACCEPTED. Delta vs rev 2 is three test table rows plus a logbook entry; manifest.go byte-identical to round 2 (sha256 97edf447...). Tree OID recomputed 9e3fb59 before and after every probe. Re-ran everything: batteries 1+2 => 91/95 killed, ZERO resurrections, same four declared survivors (M71/M72 JCS reported unknown, M92/M95 near-equivalent). Arm-shift 20/20 killed; all 17 gate deletions and H1/H2 fail at error = <nil>; H3 fails arm-blind at document members as disclosed. Ratios: 88/119 arm sites executed, 88/88 executed sites identity-asserted, 79/79 distinct executed identities asserted, 69/88 file-unique; all 19 detail-string collisions remain (unchanged bound). Re-anchored S5 divergence probe KILLED. NON-BLOCKING G-A: the refusal-arm enumeration is manual - positive control inserting a new unasserted production arm keeps gofmt/vet/full 14-package suite/tracecheck all green. Zero file-unique arms are unasserted today, so there is no sixth row to add; recommend a follow-up leaf for an AST-derived refusal_arm_inventory_test.go modelled on internal/canonicaljson/grammar_inventory_test.go. Producer note undercounts the zero-hit set by one (omits evidence canonical bytes, already inside round-2 bounds). Traceability/README/terminalbackend.go evidence carried forward from round 2 on byte-identity, not re-measured.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260905-68568a, pid=19559, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260830-1zvmw7_spawn-log_-implementer--developer--muse-_RUN-260904-e24452.log](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_spawn-log_-implementer--developer--muse-_RUN-260904-e24452.log) — System spawn log captured by task-board
- [TASK-260830-1zvmw7_manifest-probe-evidence.md](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_manifest-probe-evidence.md) — Leaf 2 implementation evidence: closed manifest/probe/evidence schemas, generation-bound admission, mutant protocol, gates
- [TASK-260830-1zvmw7_gate-logs.txt](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_gate-logs.txt) — Gate logs: full suite, package cover, tracecheck (all exit 0)
- [TASK-260830-1zvmw7_change-request_rev1.patch](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_change-request_rev1.patch) — Change Request CR-TASK-260830-1zvmw7-1 revision 1 candidate patch (repository_delta=present, 6 changed paths)
- [TASK-260830-1zvmw7_change-request_rev1-validation.log](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_change-request_rev1-validation.log) — Change Request CR-TASK-260830-1zvmw7-1 revision 1 bounded validation log
- [TASK-260830-1zvmw7_spawn-log_-reviewer--reviewer--claude-_RUN-260904-4cecb2.log](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_spawn-log_-reviewer--reviewer--claude-_RUN-260904-4cecb2.log) — System spawn log captured by task-board
- [TASK-260830-1zvmw7_review-verdict.md](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_review-verdict.md) — Round-1 reviewer verdict (CR rev 1): CHANGES REQUESTED (restored)
- [TASK-260830-1zvmw7_review-mutant-battery-1.py](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_review-mutant-battery-1.py) — Reviewer mutant battery 1 (60 mutants): apply, grep-confirm, vet, run, revert, hash-verify
- [TASK-260830-1zvmw7_review-mutant-battery-2.py](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_review-mutant-battery-2.py) — Reviewer mutant battery 2 (47 mutants incl. compile-fix retries and canonicalization narrowing)
- [TASK-260830-1zvmw7_review-arm-shift-probe.py](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_review-arm-shift-probe.py) — F2 evidence: shows which downstream arm each named refusal case shifts to when its gate is removed
- [TASK-260830-1zvmw7_review-aliasing-probe.go.txt](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_review-aliasing-probe.go.txt) — G-A evidence: reflective interior-aliasing probe over every leaf-2 exported entry point; passes at HEAD, kills the injected parseClaim row-aliasing bug the shipped suite misses
- [TASK-260830-1zvmw7_spawn-log_-implementer--developer--muse-_RUN-260905-e03cb0.log](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_spawn-log_-implementer--developer--muse-_RUN-260905-e03cb0.log) — System spawn log captured by task-board
- [TASK-260830-1zvmw7_rework-evidence.md](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_rework-evidence.md) — Round-1 rework evidence: findings closed, production/test changes, mutant and gate verification
- [TASK-260830-1zvmw7_change-request_rev2.patch](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_change-request_rev2.patch) — Change Request CR-TASK-260830-1zvmw7-2 revision 2 candidate patch (repository_delta=present, 11 changed paths)
- [TASK-260830-1zvmw7_gate-logs-rev2.txt](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_gate-logs-rev2.txt) — Gate logs: full repo suite exit 0, 14 packages ok, package coverage 88.6%
- [TASK-260830-1zvmw7_change-request_rev2-validation.log](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_change-request_rev2-validation.log) — Change Request CR-TASK-260830-1zvmw7-2 revision 2 bounded validation log
- [TASK-260830-1zvmw7_spawn-log_-reviewer--reviewer--claude-_RUN-260905-e34b30.log](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_spawn-log_-reviewer--reviewer--claude-_RUN-260905-e34b30.log) — System spawn log captured by task-board
- [TASK-260830-1zvmw7_review-verdict-round2.md](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_review-verdict-round2.md) — Round 2 review verdict: CHANGES REQUESTED. 91/95 mutants killed, 0 resurrections, 17/17 gate deletions fail with error=<nil>, 85/85 executed arm sites identity-asserted, G-C positive control killed. Blocking H1: probe static claim without manifest deletes cleanly and admits an undeclared capability.
- [TASK-260830-1zvmw7_review-round2-probe-log.txt](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_review-round2-probe-log.txt) — Round 2 reviewer machine-readable probe results: 17-gate arm-shift battery, supplementary battery (incl. G-C positive control), 5 traceability gate attacks, both round-1 mutant batteries re-run
- [TASK-260830-1zvmw7_review-round2-armshift.py](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_review-round2-armshift.py) — Round 2 arm-shift probe: deletes 17 gates and reports whether the suite fails by naming the right arm
- [TASK-260830-1zvmw7_review-round2-supplementary.py](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_review-round2-supplementary.py) — Round 2 supplementary battery: G-C aliasing positive control, F1 bypass widening, re-anchored M37, and the three uncovered arms H1-H3
- [TASK-260830-1zvmw7_spawn-log_-implementer--developer--muse-_RUN-260905-1dbca2.log](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_spawn-log_-implementer--developer--muse-_RUN-260905-1dbca2.log) — System spawn log captured by task-board
- [TASK-260830-1zvmw7_round3-three-rows.md](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_round3-three-rows.md) — Round-3 three-row rework evidence and mutant log
- [TASK-260830-1zvmw7_change-request_rev3.patch](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_change-request_rev3.patch) — Change Request CR-TASK-260830-1zvmw7-3 revision 3 candidate patch (repository_delta=present, 11 changed paths)
- [TASK-260830-1zvmw7_change-request_rev3-validation.log](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_change-request_rev3-validation.log) — Change Request CR-TASK-260830-1zvmw7-3 revision 3 bounded validation log
- [TASK-260830-1zvmw7_spawn-log_-reviewer--reviewer--claude-_RUN-260905-68568a.log](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_spawn-log_-reviewer--reviewer--claude-_RUN-260905-68568a.log) — System spawn log captured by task-board
- [TASK-260830-1zvmw7_review-round3-probe-log.txt](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_review-round3-probe-log.txt) — Round-3 probe log: batteries 1/2, 20-row arm-shift, supplementary, unasserted-arm positive control, arm-site ratios
- [TASK-260830-1zvmw7_review-round3-armratio.py](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_review-round3-armratio.py) — Round-3 arm-site enumeration + coverage-mapped identity-assertion ratio script
- [TASK-260830-1zvmw7_review-round3-newarm-control.py](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_review-round3-newarm-control.py) — G-A positive control: inserts an unasserted production refusal arm and runs gofmt/vet/full suite/tracecheck
- [TASK-260830-1zvmw7_review-round3-armshift.py](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_review-round3-armshift.py) — Round-3 arm-shift battery: 17 gate deletions plus H1/H2/H3 arm narrowings
- [TASK-260830-1zvmw7_review-round3-divergence.py](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_review-round3-divergence.py) — Re-anchored S5: registry row confers a bypassed operation (CheckOperation vs CapabilitiesForOperation divergence)
- [TASK-260830-1zvmw7_review-verdict-round3.md](file://TASK-260830-1zvmw7/TASK-260830-1zvmw7_review-verdict-round3.md) — Round-3 reviewer verdict (CR rev 3): ACCEPTED; G-A non-blocking (refusal-arm enumeration is manual, positive control attached)

## Created
2026-08-29T22:00:03Z

## Last Update
2026-09-05T01:26:13Z

## Assigned To
[reviewer] reviewer (claude)
