# TASK-260830-21gygk — CR9 changes requested

Reviewed immutable tree `6eb65f537f4ebffe6e6044f5b64ffc1b89d4487e`, base `8cf4aaaa190e6a11dff2661aa6806ce476128653`, patch SHA-256 `d49f4cfa4ea076f6f428de36ebad950ac59a862f866e1c4519e7105eb4652b74`. Authority: AX v0.6.0, `0cbdf100dbf84df50c64f792b1f940e3a67859a6`. Exact archive: 475 tracked internal/module files verified against Git blob IDs, zero mismatches. All reviewer probes and mutation copies are isolated under ignored scratch. No active product/index/branch mutation, commit, checkpoint, integration, or hosted CI.

## P1-A — fork local profile authority is skipped

repeat-of: CR8 P1, C6 incomplete cross-record profile authority (canonical shape versus semantic admission family).

`checkCheckpointProfileAuthority` loads the new Session Record's creation profile, but `checkClosureProfilePairs` in `internal/sessquery/lease.go:793` has no `fork.created` case. The canonical owner pins the nullable source to null but does not compare execution_profile against that record. The reducer treats fork.created as a non-state-changing event. Thus no invoked owner establishes this required local relationship.

SPEC 2.4 and 5.2 (lines 649-653, 1822-1831) explicitly distinguish new-session authority from source-session provenance. The fork pair must equal the NEW record profile/null. Unlike the source checkpoint provenance, that comparison requires no cross-session input: this function already has both necessary inputs.

Independent `TestReview9ForkLocalProfile` drives direct and task_board persistence through BuildPlan, fresh Revalidate, AuthoritativeStatus and AuthoritativeList. Both matching-profile controls pass. Both negatives carry a canonical fork event with standard/null under a yolo Session Record; BuildPlan and both summaries succeed and fresh Revalidate returns nil. Old-plan checkpoint substitution refuses only `selector_plan_stale: winning lease changed`, not the missing integrity gate; do not report that old plan as accepted. That is a valid old-plan refusal; it does not demonstrate the absent profile gate. No additional refusal-order requirement is imposed. Source-session provenance is held fixed as an explicitly untested caller bound; this probe proves the local pair relationship, not an end-to-end fork.

Required: admit the locally derivable fork authority in the shared closure owner, preserve the separate provenance boundary, add winner/necessary-ancestor and old-plan regression coverage with valid controls and a qualifying narrowing mutant. Do not treat the unavailable cross-session provenance input as a reason to skip the available new-session comparison.

## P1-B — resume pair is derived from the outer walk instead of its referenced checkpoint

repeat-of: CR8 P1, C6 profile-source authority.

In the session.resumed arm of `checkClosureProfilePairs`, the expected pair is simply the latest change encountered in the enclosing closure. The event's checkpoint_id is never read. The function receives no checkpoint-record map, so it cannot establish the pair at that referenced checkpoint. Canonical event shape and a valid outer checkpoint do not prove this relationship.

SPEC 5.2 lines 1822-1827 binds the event pair to the referenced checkpoint; 5.4 lines 2034-2043 forbids consulting a later local-only profile event for that checkpoint. Independent `TestReview9ResumeReferencedCheckpoint` supplies both real canonical Checkpoint Records. The positive referenced checkpoint includes E1 and standard/E1 is admitted. The negative referenced checkpoint ends at a real idle event BEFORE E1, fixing yolo/null, but the resume repeats standard/E1 from later history. BuildPlan, fresh Revalidate and both authoritative summaries all admit it. The outer checkpoint closes over the resumed event; both records exist and have valid identities, session/lease/creator/persistence bindings. This is not an existence-only fixture or a request to build storage/materialization.

Required: bind resume profile/source to the closure of its referenced admitted checkpoint, distinguish missing/read-failed/contradictory authority, and retain historical/branching/lagging semantics. Add old-plan and ancestor variants to the regression, with narrowing evidence. The current independent resume probe covers direct persistence and fresh plans; task_board/ancestor/old-plan completeness is not claimed for this new probe.

## P2 — correct mutation accounting and evidence scope

repeat-of: CR9 brief's explicit accounting caveats; prior evidence-provenance findings, not a reopened product gate.

The four supplied slice manifests enumerate **56 unique N/B plants (53 N, 3 B), not 59 kills**. Three classifier controls cannot fill the missing denominator: harmless-comment SURVIVED, missing-token NOT_APPLIED, and compile-failure COMPILE_OR_HARNESS_FAILURE. Each of the actual 56 is classified in attached `mutation-classification.md/json`: 39 semantic-admission plants and 17 label/class/output-precision plants. Three deterministic B plants test output ordering, not narrowing. Precision includes revocation/binding/record/lease/heads/absence backstops and the two plan shape backstops. No manifest reports a fixture/compile failure among those 56; most old raw per-plant logs are absent, so this is an inspected manifest/test/plant classification, not an independent 56-plant replay.

In particular N-checkpoint-heads remains refused by the profile backstop; its failure establishes error class/order, not semantic admission. The independent exact-source rerun confirms this. The four new profile plants are killed by behavioral tests with exit 1; change-direction also kills positive controls, but its stale_value negative demonstrates semantic admission. The applied harmless comment survives with exit 0 through the SAME instrument, both before/after controls exit 0, and the direct driver exit is 0. No pipeline-tail status was used.

Producer manifest exit fields are generated directly from subprocess.returncode by the inspected harness; they are distinct from an unprovable parent `python | tail; echo $?` status. Seven producer source-manifest files match the immutable CR9 bytes exactly. Preserve these limits and correct the 59/narrowing claims in outcome/traceability. Do not rerun unrelated closed audits merely for ceremony or count an unapplied/compile/empty selection as a behavioral kill.

## Conformance boundaries

- Cross-session fork.created source provenance remains caller-owned under 2.4's separate source_profile_event_id rule; NEW-session profile/null is locally owned (P1-A).
- Resume checkpoint existence as a generic storage service remains outside this leaf. Access to its already supplied checkpoint closure is necessary for the explicitly owned profile comparison; declaring all checkpoint-ID consumption out of scope narrows C6b (P1-B).
- task_board.launched lease/provider/launch-mode repetition is the 5.2 non-profile relationship. The actual invoked reducer `effectTaskBoardLaunched` checks provider and creation lease. It does not compare launch_mode and its input lacks that field; this is not evidence that canonical shape enforces mode equality. No newly invented consumer delegation is credited. The closed reducer audit and accepted profile-only review focus are retained; this review adds no separate launch-mode product finding.
- profile.changed `from` continuity is not used in the 2.4 newest-event/to derivation. Operator confirmation is required by the set-profile publication flow in 2.4; this read-only leaf does not issue the change or perform confirmation. Canonical shape checks from != to. Those boundaries are retained, not represented as end-to-end confirmation coverage.
- CLI invocation, wire Result/exit mapping, lifecycle effects, publication/storage/transport and materialization remain the accepted caller-owned boundary. No expansion into those implementations is requested.

## Closed findings and validation

Independently reran `TestRev8*`, `TestRev7*`, valid-successor and parked-union tests on exact candidate source: exit 0. This includes all four CR8 missing-source controls/negatives, real profile changes, wrong-type/non-newest/losing/cross-session sources, historical closure, two generations and old-plan profile substitution. The CR8 missing-source finding itself is closed; the broader contract is incomplete for P1-A/B.

Independent new probes: exit 1, three positive controls pass, three negative cases fail by admitting the invalid pair. Raw logs and runnable source are attached. Reused exact CR9 handoff validation: formatting, build, vet, repository-wide verbose tests and diff-check each record exit 0. Reused producer coverage log: sessquery 86.7%, sessrepo 87.2%; it is package coverage, not a claimed new full-repository coverage run. Independent mutation before/after package suites also pass. Closed ancestry/creator, canonical Lease identity, persistence variants, parked required source, seven capabilities, changed-record, predecessor/reducer/enums and creation-crash findings remain retained. No repeat of unrelated closed audits.

## AC accounting

**8 of 8 AC rows driven through shared production entries; 4 of 8 established, 4 of 8 partial/failing. 0 of 8 CLI rows delivered (accepted caller bound).** Candidate tests are immutable snapshot bytes, uncommitted per managed Story contract. Reviewer probes are attached evidence, not part of CR9.

| Row | Production call and named evidence | Result |
|---|---|---|
| UUID/name | Reader.Resolve; TestResolvePinnedPrecedence, TestResolveBareIdentityUnion | Established |
| Qualified | Reader.Resolve/resolveExplicit; TestResolveQualifiedSourcesSelectOneIndex, TestResolveQualifiedSourceNeverFallsBack, TestResolveExplicitSourceReadFailures | Established |
| Ambiguity | Reader.Resolve/matchName/checkRecordAgreement; TestResolveExactNamesAndASCIICollisions, TestResolveBareIdentityUnion | Established |
| List summaries | Reader.AuthoritativeList/authorize/winningLeaseFor; TestSummaryBootstrapRefusals, TestAuthoritativeAdmitsFullCapabilityRegistry, TestReview9* | Partial: P1-A/B |
| Status summaries | Reader.AuthoritativeStatus/authorize/winningLeaseFor; TestAuthoritativeStatusHealthy, TestAuthoritativeMissingProcessRefusesStatus, TestReview9* | Partial: P1-A/B |
| Sorting | Reader.List/Resolve; TestListStatusDerivedFactsAndStableOrder, TestResolvePeerOrderAndReplicatedIdentity | Established |
| SelectionPlan | Reader.BuildPlan/Revalidate/ParsePlan; TestBuildPlanBindsCurrentFacts, TestRevalidateDetectsEachFactChange, TestRev8ProfileOldPlanSubstitution, TestReview9* | Partial: P1-A/B |
| Negative/refusal | Shared entries above; TestRev8ProfileSourceRefusals, shipped instrument, TestReview9* | Partial: P1-A/B; P2 accounting |

Verdict: changes requested. Attach this verdict and reproducible evidence before routing to-dev. No accept_cr, done, commit acknowledgement or integration. Task notes are the review logbook; product LOGBOOK.md remains unchanged.
