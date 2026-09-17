# TASK-260830-21gygk — CR10 changes requested

Immutable candidate `106f5e99000827f20cfae6780c6f93eb4321c505`, base `9ff7d2c1d4d391dbc58d40b812757052d324a775`; patch SHA-256 `b3eec7b47c54d1096ee52dcec9a325bb3a7ecbfd28cdbe6d7732a20ada1d1a14`. Authority: pinned AX v0.6.0, commit `0cbdf100dbf84df50c64f792b1f940e3a67859a6`. Reviewer RUN-260916-64d466. All 3,075 tracked archive blobs match their Git IDs. The seven producer manifest files match the candidate SHA-256 values. Only an additional reviewer test file was written in the isolated archive; no live product/index/branch changes, commits, integration or hosted CI. Before/after live status is identical and HEAD remains `1f34c0c1a2da282d4ee7353a4abb444b0e5b778b`.

## P1 — referenced resume checkpoint bypasses semantic admission

repeat-of: CR9 P1-B / CR7–CR9 C6 canonical-shape-versus-cross-record-authority family. This is a newly introduced referenced-checkpoint path, not a reopening of the fixed winner/ancestor checks.

`referencedCheckpointProfile` (`internal/sessquery/lease.go:892`, checks at 910–918) treats an entry in `map[string]validatedCheckpoint` as admitted authority after only session equality and event-head membership. The map is canonically parsed, not semantically admitted. This path does not establish the referenced checkpoint's owning lease/fencing token, creator-holder relationship, or Session.kind persistence variant. It does not call the existing semantic checkpoint admission owner. By contrast, the outer required checkpoint uses those checks through `checkWinnerCheckpoint` / `checkCheckpointBinding`.

SPEC 5.4 binds `lease_id` to the owning lease fencing token, `created_by_host_id` to its holder, and persistence variant to the Session Record kind (pinned SPEC lines 1980–2010). SPEC 5.2's resume pair rule and 5.4's closure rule require the referenced checkpoint's authority, not merely a canonical record with known event IDs. These inputs are already supplied to the shared reader; this is not a request for storage, transport, publication, or materialization implementation.

Independent `TestReview10ReferencedCheckpointAdmission` copies the producer's resume fixture into an isolated reviewer file and changes only its separately referenced checkpoint before recomputing its canonical identity and wiring real event references. The profile pair and closure remain correct. It tests:

- `wrong_creator`: creator host B while its owning lease holder is A;
- `wrong_variant`: task-board persistence under a direct Session Record, and the reverse;
- `wrong_lease`: checkpoint fencing token `leaseCycleB` without an admitted owning lease at that epoch.

Each is exercised for direct/task_board and outer winner/necessary-ancestor closure: **12 of 12 invalid cases admitted** by `Reader.BuildPlan`, freshly built `Reader.Revalidate` (nil), `Reader.AuthoritativeStatus`, and `Reader.AuthoritativeList`. **4 of 4 valid controls pass**. The negative suite exits 1 because its refusal assertions fail, not because fixtures fail construction or compilation. Canonical digest recomputation does not manufacture semantic authority.

Old-plan behavior is explicitly bounded: every negative first binds a valid old plan over a narrower closure ending at session.stopped, then restores the full required checkpoint/lease reference. All 12 old-plan calls refuse `selector_plan_stale: winning lease changed`. That refusal is valid and preserved; no claim that these old plans were admitted, and no new error-order requirement. The missing gate is independently demonstrated by fresh plans and summaries.

Required rework: extend the shared semantic admission path to checkpoints consumed by resume derivation, with access to the necessary winning ancestry and Session.kind, rather than trusting the canonical map. Preserve historical/branching/lagging references and missing-versus-read-failed distinctions. Port the attached failing controls/negatives, add focused narrowing evidence for the newly covered path, and update C6b-R5/census claims. Audit its available 5.4 relationships as one owner; do not add only three fixture-specific exceptions. No expansion of the accepted public-caller boundary is requested.

## Prior findings and mutation accounting

CR9 P1-A and the particular outer-walk P1-B failure are closed: independent `TestReview9*`, `TestRev10*`, historical closure and two-generation controls pass (exit 0). The resume expectation is now derived from the referenced closure, and the fork comparison uses the new Session Record. The remaining issue is admission of that referenced record.

P2 accounting is corrected. Inspected 32 manifests and their raw logs: 60 unique N/B plants, 60 KILLED with subprocess exit 1 and named failures; 64 green before/after controls; one applied harmless SURVIVED control with exit 0; NOT_APPLIED and COMPILE_OR_HARNESS_FAILURE separate. Missing-token NOT_APPLIED has no behavioral log by design. Inventory is 43 semantic-admission and 17 precision plants (including three output-order B plants), not 60 narrowing admission kills. The four new plants each have actual admitted-invalid-case failure lines in raw logs: fork-value, resume-missing, resume-newest, first-value. The resume-missing complementary stale case still refuses with a changed class; that line is not counted as admission. No fixture or compile failure is counted among these four. `N-checkpoint-heads` remains precision. Inspected harness uses subprocess.returncode and restores exact source bytes. The producer's sliced harness copies were not retained, so their filtering is reported as producer provenance; no claim of an independent 60-plant replay. Existing logs and final-source hashes were reused, not rerun.

Retain closed winner/ancestor creator and variant checks, canonical Lease identity, CR7 heads, CR8 missing-source and real-profile changes, parked required sources, seven capabilities, predecessor/reducer/enums and creation crashes. The new finding does not invalidate those path-specific results.

## Validation and evidence limits

Reran independently:

1. `go test ./internal/sessquery -count=1 -run 'TestReview9|TestRev10|TestRev8ProfileHistorical|TestRev8ProfileTwoGeneration' -v` — exit 0, 70.157s. Began on the exact archive before adding the separately named reviewer test; no tracked test or product file changed. The new test name is outside that selection.
2. `go test ./internal/sessquery -count=1 -run '^TestReview10ReferencedCheckpointAdmission$' -v` — exit 1, 21.325s; 4 controls pass, 12 admission assertions fail; all old-plan refusals recorded.
3. Git blob and seven-file SHA-256 verification, patch hash, live status comparison — pass.

Reused the immutable CR10 handoff validation log: 26/26 command shards green, including build, vet, formatting, repository tests and diff check. It also prints 366 board diagnostics while returning exit 0; this is a command result, not proof of board health. Reused producer full-suite and coverage logs (sessquery 86.7%, sessrepo 87.2%); no new full-repository coverage or race replay claimed. No reason to replay unrelated closed suites after a decisive independent refusal failure.

## AC accounting

**8 of 8 AC rows driven through shared production APIs; 4 of 8 established, 4 of 8 partial/failing. 0 of 8 CLI rows delivered (accepted caller bound).** Snapshot tests remain uncommitted per managed Story policy; the new reviewer regression is attached, not part of CR10.

| Row | Production call and named candidate evidence | Result |
|---|---|---|
| UUID/name | Reader.Resolve; TestResolvePinnedPrecedence, TestResolveBareIdentityUnion | Established |
| Qualified | Reader.Resolve/resolveExplicit; TestResolveQualifiedSourcesSelectOneIndex, TestResolveQualifiedSourceNeverFallsBack, TestResolveExplicitSourceReadFailures | Established |
| Ambiguity | Reader.Resolve/matchName/checkRecordAgreement; TestResolveExactNamesAndASCIICollisions, TestResolveBareIdentityUnion | Established |
| List summaries | Reader.AuthoritativeList/authorize/winningLeaseFor; TestSummaryBootstrapRefusals, TestAuthoritativeAdmitsFullCapabilityRegistry, TestRev10ResumeReferencedCheckpoint | Partial: P1 |
| Status summaries | Reader.AuthoritativeStatus/authorize/winningLeaseFor; TestAuthoritativeStatusHealthy, TestAuthoritativeMissingProcessRefusesStatus, TestRev10ResumeReferencedCheckpoint | Partial: P1 |
| Sorting | Reader.List/Resolve; TestListStatusDerivedFactsAndStableOrder, TestResolvePeerOrderAndReplicatedIdentity | Established |
| SelectionPlan | Reader.BuildPlan/Revalidate/ParsePlan; TestBuildPlanBindsCurrentFacts, TestRevalidateDetectsEachFactChange, TestRev10ResumeOldPlanSubstitution/ReferencedWithdrawal | Partial: P1 |
| Negative/refusal | Shared entries; TestRev8ProfileSourceRefusals, TestRev10*, shipped mutants; independent TestReview10ReferencedCheckpointAdmission | Partial: P1; accounting P2 closed |

All four affected rows reach the same missing owner: BuildPlan (`plan.go:274`), Revalidate (`revalidate.go:101`), summaries via authorize (`summary.go:366`) → winningLeaseFor → checkpoint profile closure → referencedCheckpointProfile. CLI invocation, wire Result/exit mapping, lifecycle effects, publication/storage/transport/materialization, fork source provenance and confirmation remain accepted caller responsibilities. No unsupported end-to-end claim is accepted.

Verdict: changes requested, route `to-dev`. This task-scoped verdict is the review logbook; product LOGBOOK.md remains untouched. Attach verdict and reproducible raw evidence before routing. No accept_cr, commit_ack, checkpoint or integration.
