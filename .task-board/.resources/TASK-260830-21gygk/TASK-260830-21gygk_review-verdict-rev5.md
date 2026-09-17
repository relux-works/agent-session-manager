# TASK-260830-21gygk — revision 5: changes requested

Reviewed CR-TASK-260830-21gygk-5, immutable tree `ab3a8894386128681aa4e9dce451cb43c261db3e`, base `8cf4aaaa190e6a11dff2661aa6806ce476128653`. Story checkpoint remains `83640d191f78f0cd685e5f709027819799efe1cf`. Pinned authority is AX v0.6.0, source commit `0cbdf100dbf84df50c64f792b1f940e3a67859a6`, available in `internal/specdoc/SPEC.v0.6.0.md`.

Scope remains all assigned shared selector/plan/authoritative-summary behavior. CLI invocation, effect authorization and durable recovery remain their accepted caller boundaries. No product files were edited, staged or committed by this reviewer.

## P1 — Intermediate leases bypass required checkpoint validation

`internal/sessquery/lease.go:268-275` validates the ancestry's epoch links, but calls `checkWinnerCheckpoint` only for the final winner. `checkLeaseChain` at lines 290-321 walks earlier leases without validating their checkpoint references. SPEC 5.3 requires a valid epoch greater than 1 to reference a validated checkpoint for its session and predecessor lease; this applies to the epoch-2 ancestor as well as the epoch-3 winner. Section 14.7.2 requires validated winning authority and current-fact validation.

`TestRev5AncestorCheckpointAuthority` constructs epoch-1/2/3 canonical Lease Records with legal predecessor IDs and epochs, and an event chain observing the epoch-3 force takeover through Repository.AppendEvent. Both referenced checkpoints are canonically identified. The complete control passes BuildPlan, Revalidate, AuthoritativeStatus and AuthoritativeList. Removing only the checkpoint required by the epoch-2 ancestor leaves the epoch-3 checkpoint available:

- BuildPlan still succeeds; Revalidate on that plan returns nil.
- Revalidate on the originally valid plan after removing that checkpoint also returns nil.
- AuthoritativeStatus and AuthoritativeList still succeed.

The negative subtest FAILS with actual exit 1. The positive control passes. This is not a missing root or malformed-record fixture, and the selected epoch-3 winner's checkpoint tuple is correct. It demonstrates missing authority admitted by production, not a changed refusal reason.

Validate the required checkpoint relationship for each lease whose validity establishes the winning ancestry, using the owned related-record boundary. Preserve legal branching/tie semantics and genuine absence versus unavailable authority; do not use another digest or a comment declaring inputs validated as a substitute. Add positive multi-generation ancestry and missing/wrong ancestor-checkpoint negatives through build, old-plan revalidation, and both summary entries, with qualifying narrowing evidence. README/TRACEABILITY's "fully validated succession" claim is currently unsupported.

## P1 — Checkpoint attestation omits the holder relationship

`internal/sessquery/lease.go:183-230` extracts only checkpoint digest, session and lease tuple; `checkWinnerCheckpoint` at lines 363-377 checks those members and returns success. Neither it nor `sessrepo.AttestCheckpointRecord` checks `created_by_host_id` against the referenced lease holder. The wrapper at `internal/sessrepo/checkpoint.go:22` verifies canonical shape/identity only. SPEC 5.4 explicitly requires checkpoint `created_by_host_id` to be the current lease holder. The required holder is already available from the same admitted lease used for the tuple comparison.

`TestRev5CheckpointCreatorMustBeHolder` uses an otherwise identical, canonically re-identified checkpoint under epoch 1/lease A. Both the epoch-1 and epoch-2 holders are host A. The control's checkpoint creator is host A; all four entries succeed. Changing only the checkpoint creator to host B, and binding its new canonical digest in the successor, also succeeds through BuildPlan, Revalidate, AuthoritativeStatus and AuthoritativeList. The negative subtest FAILS with exit 1. No identity corruption, unavailable checkpoint or wrong lease tuple masks the missing check.

Enforce the checkpoint's semantic relationship to its owning lease through the shared admission path and cover this concrete wrong-holder case with a narrowing mutant. Reconcile all applicable checkpoint/session/lease relationships against the pinned source rather than equating canonical identity with semantic validation. Preserve the correct canonical Lease Record digest and the existing caller-owned transport/observation boundaries.

## Closed findings and retained evidence

- All four carried CR4 negative probes now pass in the candidate suite. Self-predecessor and missing winner-checkpoint gates are reached; the agreeing-record parked-source arms isolate actual refusal. The seven literal Section 7.3 capability names are accepted and an invented name is rejected through both summaries.
- Exact Lease Record identity, different canonical attestation refusal, valid lagging copies, tombstones, unreadable peers, and live changed-record refusal remain green. Earlier reviewed record-head/index subsumption and predecessor crash/reducer/enum conclusions are retained, not reopened.
- The new checkpoint wrapper's fifth attestation call site is explicitly justified in the existing provhost bound and its test passes. That proves identity-owner placement, not the missing semantics above.
- No unrelated board or stray --help/mutation-source path enters the 47-path immutable CR delta. Every one of the 47 live changed paths matches the immutable candidate by bytes. Ordinary git diff reports untracked candidate additions as deletions, so that output was not used as a content mismatch conclusion. Candidate evidence is from a Git archive, excluding the stale board copy. HEAD..main is 0; this is recorded only as local ancestry context.

## AC coverage

**8 of 8 AC rows driven at shared production entries; 4 of 8 established, 4 of 8 partial/failing.** This retains the full eight-row denominator rather than merging away the plan and refusal obligations. Public CLI execution remains a stated caller-integration bound (0 of 8 CLI rows implemented here), not an extra finding.

| Row | Production entry and named candidate tests | Result |
| --- | --- | --- |
| UUID/name | Reader.Resolve; TestResolvePinnedPrecedence, TestResolveBareIdentityUnion | Established within supplied configuration/repository boundary |
| Qualified selector | Reader.Resolve -> resolveExplicit; TestResolveQualifiedSourcesSelectOneIndex, TestResolveQualifiedSourceNeverFallsBack, TestResolveExplicitSourceReadFailures | Established |
| Ambiguity | Reader.Resolve -> matchName/checkRecordAgreement; TestResolveExactNamesAndASCIICollisions, TestResolveBareIdentityUnion | Established within reached tiers |
| List summaries | Reader.List/AuthoritativeList -> authorize -> winningLeaseFor; TestSummaryBootstrapRefusals, TestSummaryMixedListRefusesWhole, TestAuthoritativeAdmitsFullCapabilityRegistry | Partial: both checkpoint-authority findings |
| Status summaries | Reader.AuthoritativeStatus -> authorize -> winningLeaseFor; TestAuthoritativeStatusHealthy, TestAuthoritativeMissingProcessRefusesStatus, TestAuthoritativeHostNameBound64 | Partial: same findings |
| Stable sorting | Reader.List/Resolve; TestListStatusDerivedFactsAndStableOrder, TestResolvePeerOrderAndReplicatedIdentity | Established |
| SelectionPlan | Reader.BuildPlan/Revalidate/ParsePlan -> winningLeaseFor; TestBuildPlanBindsCurrentFacts, TestRevalidateDetectsEachFactChange, TestValidSuccessorBuildsAndRevalidates, TestRevalidateUnionParkedCopyRefuses | Partial: both findings, including invalidation after loss of ancestor checkpoint |
| Negative/refusal | Resolve/List/Status/BuildPlan/Revalidate; candidate refusal suites, mutation logs, reviewer_rev5_test.go | Partial: two demonstrated unguarded classes |

Reviewer regressions are attached evidence, not candidate tests. Carry them with corrected production behavior into the next immutable candidate; do not claim them committed now.

## Mutation audit

Reused complete exact-source evidence instead of rerunning the 47-plant battery. Attached mutants-rev4-final.json is byte-identical to scratch mutants-shipped-final/mutants.json. Every file in that retained mutation-source/internal matches the immutable candidate/internal. Every named failure in all 47 raw kill logs matches the manifest. The exact candidate's delivered run() performs replacement-count checks, executes behavioral suites, classifies real exits, and restores bytes.

Observed classifications: **44 N + 3 B = 47 KILLED; 2 CONTROL exit 0; 1 SURVIVED exit 0; 1 NOT_APPLIED; 1 COMPILE_OR_HARNESS_FAILURE**. The harmless applied-comment SURVIVED has a real passing log through the same run() instrument. This closes the previous survivor-evidence concern. The focused four-plant manifest binds the final lease.go/revalidate.go/summary.go hashes and reports 4 KILLED plus the harmless survivor. Exact scripts, manifests and raw logs are preserved in this review bundle.

The new self-link/checkpoint-placeholder/agreeing-parked/capability plants have strong admission failures. Older N-rev-record/lease/heads and similar plants may only change the refusal reason; preserve their stated subsumption bounds rather than counting classifier KILLED as proof of unauthorized success. None covers the two new unguarded semantic classes, so "every gate/class covered" is not established by the count.

## Independently run validation and limits

Go library target, darwin/arm64, Go 1.25.5. No iOS target applies. Read the required project-management/architecture skills and the existing main checkout's Curator-managed Go skill; managed skill links are absent in this Story checkout. No installation or infrastructure change.

| Command | Exit / result |
| --- | --- |
| go test ./internal/sessquery ./internal/sessrepo ./internal/sessstate ./internal/provhost -count=1 -v, exact archive before probe injection | 0; full package log attached |
| go test ./internal/sessquery -run '^TestRev5' -count=1 -v, exact source plus reviewer tests | 1; two negative subtests fail, two positive controls pass |
| gofmt -l internal | 0, no listed files |
| go vet ./internal/sessquery ./internal/sessrepo ./internal/provhost | 0 |
| go build ./... | 0 |
| git diff --check BASE TREE | 0 |

An initial ancestor probe used an invalid stale-to-materializing event transition, so its positive control failed. That attempt is retained as probes-initial-invalid-premise.log and excluded from product-defect evidence. The final force-takeover fixture has a passing control, supported production event APIs, and identical event bytes in its positive and negative arms.

Reused the CR5 publication log's exit-0 markers for full tests/build/vet/format/diff, not a new full repository/race/coverage or 26-command board run. The log is bounded and contains 3080 board-copy issues; it cannot establish a clean full board validation. Producer coverage/race values remain reported prior evidence. validation-final.log actually has `gofmt -l internal/ cmd/` exit 2 because cmd/ is absent, contrary to the outcome's clean claim; the fresh valid formatting command above establishes current formatting without rewriting that old failure. Correct the evidence description in the handoff.

Shared query/plan/summary reads introduce no durable writes. Retained creation crash tests establish composition with repository recovery, not a new durability protocol.

## Task logbook and disposition

**Changes requested; route TASK-260830-21gygk to to-dev.** These are reproducible implementation gaps, not a human-only decision or external blocker. Preserve the full assigned objective and fix the shared semantic admission path. Task-scoped verdict, raw test source/logs, immutable/source hashes, mutation audit and durable producer logs are attached before routing. Incomplete conformance and negative-coverage checklist claims are unchecked; completed validation retains the bounds above. No accept_cr, commit_ack, branch mutation, checkpoint, integration or landing.
