# TASK-260830-21gygk — CR6: changes requested

Reviewed immutable tree `e8783471325b552f7057eda36c768102b34807d9`, base `8cf4aaaa190e6a11dff2661aa6806ce476128653`, Story checkpoint `83640d191f78f0cd685e5f709027819799efe1cf`. Authority remains AX v0.6.0 at `0cbdf100dbf84df50c64f792b1f940e3a67859a6`. All 48 changed paths match the live candidate and the review archive by bytes; no board or stray mutation tree is in the immutable delta. No active product file, index, branch or commit was changed by this reviewer.

## P1 — Checkpoint persistence variant is not bound to the Session Record

repeat-of: CR5 semantic authority admission family (canonical identity mistaken for complete related-record validation); this is a new relationship, not reopening the repaired ancestor/holder findings.

Pinned `internal/specdoc/SPEC.v0.6.0.md` Section 5.4 requires a direct Session Record's checkpoint to have non-null `provider_manifest_id` and null `task_board_bundle_id`; task_board requires the reverse. Section 5.3 requires validated checkpoints throughout necessary lease ancestry, and 14.7.2 requires validated current authority.

`internal/sessquery/lease.go:195-229` attests canonical checkpoint shape/identity and extracts session/lease/creator, but does not bind the persistence variant to the Session Record. `checkWinnerCheckpoint` and `checkCheckpointBinding` (`lease.go:454-468`) never inspect that relationship. The canonical owner at `internal/canonicaljson/core_records.go:163-176` proves only exactly one persistence ID exists. Its no-Session-Record API cannot establish which variant is valid.

Independent `TestRev6CheckpointPersistenceMustMatchSession` uses the existing valid direct-session fixture, canonical checkpoint identity, correct session, lease, epoch and creator. It swaps only provider/non-provider persistence fields, re-identifies the checkpoint and updates the referencing lease digest. Both positive direct controls PASS; both invalid variant subtests FAIL with exit 1:

- Winning-successor case: BuildPlan succeeds, Revalidate of that plan returns nil, and AuthoritativeStatus and AuthoritativeList succeed.
- Necessary-ancestor case: all the same entries succeed. Additionally, an originally valid plan remains accepted by Revalidate after changing only the ancestor's checkpoint variant and canonical referencing lease. The winner record and event bytes remain unchanged.

Raw evidence: `probes.log`, executable `reviewer_rev6_executed_test.go`. Reproduce by copying this reviewer test into the exact candidate's `internal/sessquery/` in scratch and running `go test ./internal/sessquery -run '^TestRev6CheckpointPersistenceMustMatchSession$' -count=1 -v`.

Fix the shared semantic admission path for every checkpoint required by the winning ancestry, using the actual Session Record kind. Add positive direct/task_board controls and wrong-variant negatives, including old-plan revalidation, through all four shared entries. Include a qualifying narrowing mutant. Do not substitute canonical digest or caller-trusted prose for this concrete relationship. Other object publication/transport boundaries remain outside this finding; no demand is made here to implement all manifest storage or checkpoint publication.

The recurring family now spans multiple reviews. Route a class-level conformance gate/audit for the owned Session/Lease/Checkpoint relationships before another fixture-only rework. Preserve the complete denominator and the existing closed findings.

## P2 — Previously qualifying self-predecessor mutant now fails compilation

`internal/sessquery/testdata/mutate.py:116` still inserts `return parent, nil`, but CR6 changes checkLeaseChain to return `([]validatedLease, validatedLease, error)`. The anchor still appears once; that proves application only. Running the shipped run() implementation against exact CR6 yields COMPILE_OR_HARNESS_FAILURE, exit 1, with `lease.go:328:19: not enough return values`. No named behavioral test fails for this plant. The focused battery therefore exits 1.

Repair this mutation for the new signature while preserving the intended narrow self-link admission and execute its behavioral test. This does not reopen the existing product self-link guard: its current negative tests pass. It repairs newly broken final-source gate evidence. Prior CR5 KILLED evidence cannot establish a compiling plant against the changed signature.

## Closed CR5 findings and retained bounds

Independently reran all TestRev5 controls/negatives. Missing required ancestor checkpoint, wrong ancestor session/lease/creator, and wrong winner checkpoint creator now refuse; valid multi-generation and holder controls pass. The original old-plan loss-of-ancestor checkpoint regression passes. These findings are closed.

Full candidate tests pass, including prior canonical Lease Record identity, lagging replicas, required parked sources, live changed-record refusal, all seven capabilities, predecessor/reducer/enum/crash cases. Earlier architecture and caller integration conclusions remain accepted: CLI invocation, wire Result 5, effects and durable recovery are caller-owned. This read-only API adds no durable mutation protocol. No hosted CI was used.

## AC coverage

**8 of 8 AC rows driven at shared production entries; 4 of 8 established and 4 of 8 partial/failing.** Public CLI execution remains the accepted stated bound, 0 of 8 CLI rows delivered by this library leaf.

| Row | Production entry / named candidate evidence | Result |
| --- | --- | --- |
| UUID/name | Reader.Resolve; TestResolvePinnedPrecedence, TestResolveBareIdentityUnion | Established |
| Qualified selector | Reader.Resolve -> resolveExplicit; TestResolveQualifiedSourcesSelectOneIndex, TestResolveQualifiedSourceNeverFallsBack, TestResolveExplicitSourceReadFailures | Established |
| Ambiguity | Reader.Resolve -> matchName/checkRecordAgreement; TestResolveExactNamesAndASCIICollisions, TestResolveBareIdentityUnion | Established |
| List summaries | Reader.List/AuthoritativeList -> authorize -> winningLeaseFor; TestSummaryBootstrapRefusals, TestSummaryMixedListRefusesWhole, TestAuthoritativeAdmitsFullCapabilityRegistry | Partial: P1 |
| Status summaries | Reader.AuthoritativeStatus -> authorize -> winningLeaseFor; TestAuthoritativeStatusHealthy, TestAuthoritativeMissingProcessRefusesStatus, TestAuthoritativeHostNameBound64 | Partial: P1 |
| Stable sorting | Reader.List/Resolve; TestListStatusDerivedFactsAndStableOrder, TestResolvePeerOrderAndReplicatedIdentity | Established |
| SelectionPlan | Reader.BuildPlan/Revalidate/ParsePlan -> winningLeaseFor; TestBuildPlanBindsCurrentFacts, TestRevalidateDetectsEachFactChange, TestValidSuccessorBuildsAndRevalidates, TestRevalidateUnionParkedCopyRefuses, TestRev5AncestorCheckpointAuthority | Partial: P1 |
| Negative/refusal | Resolve/List/Status/BuildPlan/Revalidate; candidate refusal suites, mutation instrument, reviewer Rev6 test | Partial: P1 and P2 |

Reviewer Rev6 regressions are attached evidence, not committed candidate tests. Carry them into the producer's next uncommitted candidate.

## Validation and evidence limits

Go library, darwin/arm64, Go 1.25.5. Required project-management and architecture-diagrams skills read; main checkout's Curator-managed Go testing skill used because managed links are absent in this Story checkout. No installation.

- Exact archive `go test ./... -count=1 -v`: exit 0 (`tests.log`); no reviewer TestRev6 in this run.
- Exact archive `go test ./... -cover`: exit 0 (`coverage.log`); sessquery 87.9%, sessrepo 87.2%.
- Exact product plus reviewer test `go test ./internal/sessquery -run '^TestRev[56]' -count=1 -v`: exit 1 (`probes.log`); all CR5 cases pass, both new invalid variant cases fail, positive controls pass.
- `go build ./...`: exit 0; targeted `go vet ./internal/sessquery ./internal/sessrepo ./internal/provhost`: exit 0. These ran with reviewer test present; product bytes unchanged.
- Exact candidate `gofmt -l internal`: exit 0 with no listed paths (`format-exact.log`). Earlier format.log listed only the unformatted scratch reviewer test, not product code. Producer correctly records old nonexistent cmd/ format command as exit 2, not clean.
- `git diff --check BASE TREE`: exit 0. Local HEAD..main count is 0, recorded only as local ancestry context.
- CR6 publication log independently records exit-0 format/build/full-vet/test/diff commands; this is reused publication evidence, not a new board/race run.

Operational correction: an initial probe invocation used the active cwd and ran only existing TestRev5 tests, exit 0; its log is retained and excluded from new-probe proof. The first focused mutant copy raced with scratch probe injection, so its control failed on the deliberately failing reviewer test; that run is retained as invalid setup, excluded from mutation classifications. The corrected focused run uses exact candidate source without reviewer injection. No failure is relabeled passing.

The attached evidence bundle includes exact-tree path hashes, reviewer tests, actual logs, focused script retaining the shipped run() body/plants, final mutation classifications, and producer focused logs. Prior CR5 review evidence was materialized and read; unchanged closed audits are reused with the above bounds rather than repeated wholesale. Acceptance remains withheld; route to-dev, not blocked, integrating or done.

Final independent focused mutation result: **3 KILLED (new narrowing plants), 1 SURVIVED (applied harmless comment), 1 COMPILE_OR_HARNESS_FAILURE (N-chain-self), 2 CONTROL exit 0; overall exit 1**. Raw logs and mutants.json are in mutants-exact/. This is a focused selection of shipped plants with the identical run() implementation, not a claimed rerun of the entire historical battery.
