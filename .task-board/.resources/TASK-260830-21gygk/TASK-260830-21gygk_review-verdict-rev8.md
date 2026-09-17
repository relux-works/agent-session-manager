# TASK-260830-21gygk — CR8 changes requested

Candidate tree: `5102e49fe89c9bb6d085f45b7b4ea3a392422a25`; base: `8cf4aaaa190e6a11dff2661aa6806ce476128653`. Authority: AX v0.6.0, `0cbdf100dbf84df50c64f792b1f940e3a67859a6`. Review used an exact git archive and isolated probes. All 472 internal files match candidate Git blobs; all 50 changed active paths match the archive. Active product, index and HEAD were preserved. No integration, manual commits or hosted CI.

## P1 — C6 closure admission still accepts missing profile authority

repeat-of: CR7 C6 incomplete event/profile authority; CR5/CR6 canonical-shape-versus-semantic-admission family.

`internal/sessquery/lease.go:572-604` calls `Repository.ListEvents`, indexes summaries by digest, and checks only each named head's existence and lease tuple. `EventSummary` (`internal/sessrepo/sessrepo.go:62`) contains no profile payload or profile-source reference. Neither this gate nor another invoked owner establishes the profile/source semantics of the reachable events. The gate runs from BuildPlan (`plan.go:274`), Revalidate (`revalidate.go:101`), and both authoritative summaries (`summary.go:365`).

Pinned 5.4 at lines 2034-2043 says the transitive closure fixes effective profile/source and that a missing, losing-lease or non-newest profile.changed reference is integrity_failure. Sections 2.4 (626-680) and 5.2 (1822-1831) require the first launch to carry the Session Record profile/null and subsequent profile sources to identify their authoritative change. A canonical event with a well-shaped but nonexistent digest does not satisfy that relationship.

Independent `TestRev8CheckpointProfileAuthority` creates real, canonically admitted repository events and Lease/Checkpoint Records. Its four controls (direct/task_board × winner/necessary ancestor) pass with the correct creation profile and null first-launch source. Each negative changes that launch's source to a validly shaped all-zero digest for which no profile.changed event exists. The checkpoint points to a real later idle event: all current head, session, lease, creator and persistence checks remain satisfied. All four negatives FAIL because BuildPlan succeeds, fresh Revalidate returns nil, AuthoritativeStatus succeeds, and AuthoritativeList succeeds. The required refusal is not produced. Raw probe exit is 1, not a passing diagnostic.

This is consumption/admission of already supplied authority, not a request to implement publication, bundle storage, transport, provider execution or CLI integration. C6's matrix states profile derivation is absent from sessquery/sessstate and future materialization APIs do not exist. Those observations establish an absent semantic owner; they do not delegate the required admission to an API that actually checks it. Correct the shared admission or compose an actual owner with the necessary record/event closure inputs. Port the attached regression, add real profile.changed positive controls and missing/non-newest/losing references under the pinned rules, and prove the gate with qualifying narrowing evidence. Do not substitute current-tail equality or later local events for checkpoint closure authority.

Evidence limit: the new probe establishes the missing-source class and freshly built plan revalidation. It does not claim independently to prove every profile scenario, old-plan profile-only substitution, or end-to-end materialization behavior. Existing CR7 old-plan checkpoint-head regressions were independently rerun and pass.

## Closed findings and retained bounds

CR7 missing/later checkpoint heads are now refused: exact-source TestRev7CheckpointHeadAuthority passes, including winner/ancestor controls and old-plan substitution. Direct/task_board persistence controls also pass. Retain the closed ancestry/creator, canonical Lease identity, lagging copies, required parked-source refusal, seven capabilities, changed-record, predecessor/reducer/enum and creation-crash findings. No unrelated closed audit is reopened.

Read-only shared APIs remain the accepted caller boundary. CLI invocation, wire exits, publication/storage/transport and lifecycle effects are not credited as delivered here and are not newly requested.

## AC ratio

**8 of 8 AC rows driven through shared production entries; 4 of 8 established, 4 of 8 partial/failing.** Public CLI remains a stated bound (0 of 8 CLI rows delivered by this leaf).

| Row | Production call and named evidence | Result |
| --- | --- | --- |
| UUID/name | Reader.Resolve; TestResolvePinnedPrecedence, TestResolveBareIdentityUnion | Established |
| Qualified selectors | Reader.Resolve/resolveExplicit; TestResolveQualifiedSourcesSelectOneIndex, TestResolveQualifiedSourceNeverFallsBack, TestResolveExplicitSourceReadFailures | Established |
| Ambiguity | Reader.Resolve/matchName/checkRecordAgreement; TestResolveExactNamesAndASCIICollisions, TestResolveBareIdentityUnion | Established |
| List summaries | Reader.AuthoritativeList/authorize/winningLeaseFor; TestSummaryBootstrapRefusals, TestSummaryMixedListRefusesWhole, TestAuthoritativeAdmitsFullCapabilityRegistry; new TestRev8CheckpointProfileAuthority | Partial: P1 |
| Status summaries | Reader.AuthoritativeStatus/authorize/winningLeaseFor; TestAuthoritativeStatusHealthy, TestAuthoritativeMissingProcessRefusesStatus, TestAuthoritativeHostNameBound64; new TestRev8CheckpointProfileAuthority | Partial: P1 |
| Stable sorting | Reader.List/Resolve; TestListStatusDerivedFactsAndStableOrder, TestResolvePeerOrderAndReplicatedIdentity | Established |
| SelectionPlan | Reader.BuildPlan/Revalidate/ParsePlan; TestBuildPlanBindsCurrentFacts, TestRevalidateDetectsEachFactChange, TestValidSuccessorBuildsAndRevalidates, TestRevalidateUnionParkedCopyRefuses, TestRev5/6/7 suites; new TestRev8CheckpointProfileAuthority | Partial: P1 |
| Negative/refusal | Shared entries above; shipped refusal tests and mutation instrument; new TestRev8CheckpointProfileAuthority | Partial: P1 |

The new reviewer test is attached evidence, not part of the immutable candidate. Candidate tests are snapshotted but uncommitted, as the managed Story contract requires.

## Validation and mutation evidence

Independent local checks on exact candidate bytes (darwin/arm64, Go 1.25.5): full `go test ./... -count=1`, full `go test ./... -count=1 -v`, full `go test ./... -cover`, `go build ./...`, `go vet ./...`, `gofmt -l internal`, and exact-tree `git diff --check` all exit 0. Formatting listed no files. Coverage: sessquery 87.4%, sessrepo 87.2%. The separate new reviewer regression exits 1 with four positive controls passing and four negative cases failing. No failed evidence is represented as clean.

Independently reran the eight affected shipped lease plants with the delivered behavioral instrument on the exact candidate archive: N-lease-missing, N-chain-self, N-checkpoint-placeholder, N-ancestor-checkpoint, N-checkpoint-holder, N-ancestor-holder, N-checkpoint-persistence and N-checkpoint-heads all KILLED with real exit 1 and named failing tests. Applied C-harmless-comment SURVIVED with exit 0; before/after controls exit 0; driver exit 0. N-checkpoint-heads fails precisely the winner/ancestor missing-head cases while later-lease cases retain refusal. No compile failure, unapplied plant or empty selection is counted. Raw logs and mutants.json are bundled, with byte restoration asserted by the shipped instrument. Full historical unrelated mutation batteries were not rerun; their closed conclusions remain the stated prior-evidence bound.

Retained producer evidence differs from current source only in TRACEABILITY.md and the comment header of rev7_regression_test.go (source-comparison.json); it is supplementary. Current review conclusions about affected mutants rely on the independent exact-source run, not a claim that those producer copies are byte-identical. CR8 handoff validation log was inspected; its board diagnostics are preserved, including warning counts and reported exit 0, not rewritten as a warning-free validation.

The bundle contains runnable probe, reproduction instructions, source comparisons, exact revision delta, baseline and mutation raw logs, producer outcome/matrix, and publication evidence. The task note serves as the review logbook entry; product LOGBOOK.md remains untouched.

Verdict: changes requested. Route to-dev. No accept_cr, integration or done transition.
