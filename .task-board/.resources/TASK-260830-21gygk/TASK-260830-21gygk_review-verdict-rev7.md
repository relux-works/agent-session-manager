# TASK-260830-21gygk — CR7: changes requested

Reviewed immutable candidate tree `0715659e5ac515b387a9a3e5926cba4138d9cbea`, base `8cf4aaaa190e6a11dff2661aa6806ce476128653`, Story checkpoint `83640d191f78f0cd685e5f709027819799efe1cf`. Authority: AX v0.6.0, `0cbdf100dbf84df50c64f792b1f940e3a67859a6`. All 49 changed paths match both the exact archive and active candidate by bytes. No active product, index, branch or commit was modified. All CI is local.

## P1 — Checkpoint event-head authority is never admitted

repeat-of: CR5/CR6 canonical-identity-versus-related-record-authority family. This is the C6 relationship explicitly included in the CR6 rework audit, not reopening the repaired ancestry, creator or persistence gates.

`internal/sessquery/lease.go:191-245` attests the canonical Checkpoint Record but never extracts or resolves `event_heads`. `checkWinnerCheckpoint` (`lease.go:392`) and `checkCheckpointBinding` (`lease.go:481`) admit a required checkpoint after checking session, lease, creator and persistence variant. Neither establishes the checkpoint's event authority. `Revalidate` uses those same gates at `revalidate.go:101`; `collectAuthorityHeads` (`revalidate.go:257`) binds the winning Lease Record and present source tails, not the required checkpoint's closure.

Pinned Section 5.3 requires a validated checkpoint for every necessary successor link. Section 5.4 defines checkpoint event_heads as the authoritative DAG immediately before that object and specifies closure/profile-reference integrity. Section 14.7.2 requires a validated winning Lease Record and current complete authority. A canonical checkpoint naming an absent event or an event under a later lease cannot establish that authority.

Independent `TestRev7CheckpointHeadAuthority` proves the omission through production calls:

- Both controls PASS. They create actual local Session Events, reach idle under the owning lease, bind the checkpoint to that actual head, and use canonical Lease/Checkpoint Records with correct session, epoch, holder and direct persistence variant.
- Both winner negatives FAIL: replacing only checkpoint event_heads with a missing digest or an existing later-lease event is admitted by `Reader.BuildPlan`, freshly built plan `Reader.Revalidate`, `Reader.AuthoritativeStatus` and `Reader.AuthoritativeList`.
- Both necessary-ancestor negatives FAIL through those same entries. Additionally, `Reader.Revalidate(oldPlan)` returns nil after changing only the ancestor checkpoint's heads and its canonical referring ancestor Lease Record. The winning Lease Record, session and event bytes remain unchanged.
- Old-plan winner changes correctly refuse because the winner's digest changes. This is retained behavior, not evidence that checkpoint closure was checked.

Final `probes.log`: exit 1, four invalid-head subtests fail, both head controls pass. The same invocation also passes all eight independent direct/task_board persistence controls/negatives. Earlier invalid test setup/control attempts are preserved and excluded explicitly.

Reproduce from an archive of CR7: copy attached `reviewer_rev7_test.go` into `internal/sessquery/`, then run:

```sh
go test ./internal/sessquery -run '^TestRev7' -count=1 -v
```

Fix the shared admission of event authority for every checkpoint required by the winning ancestry. Preserve valid historical checkpoint heads rather than requiring them to equal today's tail; preserve valid branching/lagging copies. Distinguish a proved missing reference from an unavailable read and propagate required-source failures. Add focused production negatives and a narrowing mutant that admits one invalid reference, with an applied harmless survivor through the same instrument.

This finding does not demand manifest publication, storage, transport, CLI effect execution or a new recovery protocol. The fixture retains those accepted caller boundaries. It concerns the event/lease relationship of a checkpoint that this shared API already consumes to declare a lease validated.

## The required conformance matrix is not yet a trustworthy closure claim

The attached matrix's C5 delegation is substantiated: `canonicaljson.validateSafeBoundaryEvidence` reads the supplied safe-boundary members and rejects false booleans/nonzero counters, while the checkpoint validator enforces status and shape. C4 now has the necessary Session.kind input and its own shared gate.

C6 instead names a generic checkpoint consumer/publication owner, and the final paragraph claims delegated APIs carry the needed inputs. There is no named implementation or test for that claim. `sessrepo.AttestCheckpointRecord(raw)` (`internal/sessrepo/checkpoint.go:22`) calls canonical identity validation; `canonicaljson.validateCheckpointRecord` (`core_records.go:147-153`) checks only the head array's cardinality, sorted uniqueness and digest syntax, without repositories or event closure. `provhost.ProfileMapping(providerID, profile)` only maps strings to adapter flags. The sessstate documentation explicitly excludes persisted-profile derivation and treats profile events as inert facts. None establishes C6 at the selector admission path.

Correct C6 and the audit conclusion using real call sites/input contracts and behavioral evidence. Effective-profile consumer behavior itself was not newly exercised here; no claim is made that every profile scenario fails. The concrete missing/later-lease checkpoint-head admission above is sufficient to withhold acceptance. Do not turn the unsupported delegation into a scope exemption.

## CR6 findings closed; retained conclusions

- CR6 P1 is closed. Independent `TestRev7IndependentPersistence` covers direct AND task_board, winner AND necessary ancestry, valid controls AND wrong variants, BuildPlan, old-plan Revalidate, AuthoritativeStatus and AuthoritativeList. All eight subtests pass. The shipped TestRev6 suite also passes in the full exact-candidate run.
- CR6 P2 is closed. The repaired `N-chain-self` compiles and is behaviorally KILLED by `TestSelfPredecessorWithCheckpointMustRefuse` on exact final source. It is no longer a compilation failure.
- Closed ancestry/creator, canonical identity, lagging-copy, parked-source, seven-capability, changed-record, predecessor/reducer/enum and creation-crash findings remain closed. The full candidate suite was rerun, but unrelated historical mutation audits were not repeated wholesale.
- Caller-owned CLI invocation, wire output/exit mapping, serialized effects and durable recovery remain accepted bounds. No public CLI behavior is credited to this library leaf.

## AC coverage

**8 of 8 AC rows driven through shared production entries; 4 of 8 established and 4 of 8 partial/failing.** Public CLI integration remains a stated bound: 0 of 8 CLI rows delivered here.

| Row | Production call site and named candidate evidence | Review result |
| --- | --- | --- |
| UUID/name | Reader.Resolve; TestResolvePinnedPrecedence, TestResolveBareIdentityUnion | Established |
| Qualified selector | Reader.Resolve / resolveExplicit; TestResolveQualifiedSourcesSelectOneIndex, TestResolveQualifiedSourceNeverFallsBack, TestResolveExplicitSourceReadFailures | Established |
| Ambiguity | Reader.Resolve / matchName / checkRecordAgreement; TestResolveExactNamesAndASCIICollisions, TestResolveBareIdentityUnion | Established |
| List summaries | Reader.AuthoritativeList / authorize / winningLeaseFor; TestSummaryBootstrapRefusals, TestSummaryMixedListRefusesWhole, TestAuthoritativeAdmitsFullCapabilityRegistry | Partial: P1 |
| Status summaries | Reader.AuthoritativeStatus / authorize / winningLeaseFor; TestAuthoritativeStatusHealthy, TestAuthoritativeMissingProcessRefusesStatus, TestAuthoritativeHostNameBound64 | Partial: P1 |
| Stable sorting | Reader.List / Resolve; TestListStatusDerivedFactsAndStableOrder, TestResolvePeerOrderAndReplicatedIdentity | Established |
| SelectionPlan | Reader.BuildPlan / Revalidate / ParsePlan; TestBuildPlanBindsCurrentFacts, TestRevalidateDetectsEachFactChange, TestValidSuccessorBuildsAndRevalidates, TestRevalidateUnionParkedCopyRefuses, TestRev5*, TestRev6* | Partial: P1 |
| Negative/refusal | Resolve / List / Status / BuildPlan / Revalidate; shipped refusal suites and mutation instrument, independent TestRev7CheckpointHeadAuthority | Partial: P1 |

Reviewer TestRev7 is attached evidence, not part of the immutable candidate. Port it during rework; do not claim it is already shipped.

## Validation and provenance

Go library, darwin/arm64, Go 1.25.5. Required skills read, including main checkout's Curator-managed Go skill (managed links absent from this worktree). No installation, hosted CI or integration.

Independent exact-source checks: full `go test ./... -count=1 -v`, coverage, `go build ./...`, full `go vet ./...`, `gofmt -l internal` and exact-tree `git diff --check`. Actual final exits are recorded in `actual-exits.json`; gofmt used an existing path and listed no files. The reviewer probe is intentionally failing and remains exit 1.

Independent focused mutation driver retains the shipped run() implementation and plants, filtering only the selection and setting the exact archive root: N-chain-self and N-checkpoint-persistence KILLED; applied C-harmless-comment SURVIVED; controls before/after pass. Each log records named failures and real subprocess exits. The five other affected lease plants are accepted from producer batch B: all five KILLED, controls pass. All internal bytes in producer batches A/B match this final candidate, verified in source-comparison.json; raw logs/manifests are copied into this review bundle. NOT_APPLIED and COMPILE_OR_HARNESS_FAILURE controls are classified separately and never counted as kills. No full historical battery rerun is claimed.

The evidence bundle contains candidate hashes, exact CR6-to-CR7 path list, original CR6 verdict/probe, producer outcome/matrix, publication log, new executable regressions, raw tests/checks/mutants and operational corrections. The task note records this review finding as the review logbook entry while product LOGBOOK stays untouched.

Verdict: changes requested. Route to-dev; not blocked, accepted, integrating or done.
