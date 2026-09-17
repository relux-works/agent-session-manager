# CR15 review — changes requested

Task TASK-260830-21gygk; reviewer RUN-260917-d0b0b9. Reviewed immutable tree `53fc631adb15546892cc8f32fbf619eda5a6a957`, base `8626fb361c1d6b62d5f95c7be4cb33c85d4aadf3`, patch SHA-256 `8293d908c233f8fcc29dab55ac80c8722433a9db00ee038b779cafe3792a05e8`. Authority remains v0.6.0 / `0cbdf100dbf84df50c64f792b1f940e3a67859a6`. All probes were local, in extracted immutable copies; no live product/index/branch/HEAD edits, commits, integration, or hosted CI.

## P2-A — type aliases bypass the seal/token inventory gate

Repeat-of: CR14 P2-A (class-wide seal construction inventory), previously CR13 P2-A / CR11 P2. The two specific CR14 plants are fixed, but the required class-wide enforcement remains incomplete.

`TestReview15AdditionalShapes/alias_seal` and `/alias_token` introduce these compiling production shapes independently:

```go
type review15Alias = checkpointSeal // second case: checkpointSealToken
func review15Mint() any { return new(review15Alias) }
```

The actual `rev12CheckRecordConsumptionSource` returns nil. Both negative assertions fail because an alternate seal/token producer is admitted, not because compilation failed. The defect is in `internal/sessquery/rev11_regression_test.go`: `rev14SealedTypeExpr` (around 1261) compares the alias's own TypeName object; `rev14SealedValueContainsSeen` (around 1340) handles Named/Pointer/container types but not `*types.Alias`. Thus canonical Go type identity is lost at the inventory boundary. This directly contradicts the assigned requirement and TRACEABILITY claim that only the exact admission constructor creates seals. It is an inventory enforcement defect, not proof of a current externally supplied Reader exploit.

Required repair: normalize Go aliases semantically throughout the seal/token identity checks, including chained/container aliases, rather than adding alias spellings to a lexical list. Port the independent failing controls, retain the real baseline and non-alias negatives, and provide focused qualifying narrowing evidence through this same gate. Preserve closed product behavior and callback repair; no new runtime cryptography or caller-owned functionality is requested.

## P3 — generated Python cache in candidate

`internal/sessquery/testdata/__pycache__/mutate.cpython-314.pyc` is newly included in CR15. Remove this interpreter-generated artifact from the next candidate; it is not source or task evidence. The candidate otherwise has no task-board.config.json or stray probe source. This cleanup does not require a separate product audit.

## Closed findings and independently exercised controls

- CR14's non-composite `new(checkpointSeal)` producer and unknown package callback now fail the census with a semantic violation. Unknown callback and struct-field callback are refused; no compile failure is counted as a census kill.
- Independently added generic constructor, any-typed token extraction, and build-tagged seal producer are rejected. The alias pair above remains admitted. Thus the additional five-shape inventory check is **3 of 5 refused**, not complete.
- CR14 P2-B is closed: `review11TemporalFixture` now uses a pre-change head and yolo/null pair for good_prechange; good_divergent preserves a losing-lease event while retaining the winning pair. The relation census accurately describes admission. Independently reran the original fixture: **12 distinct positive controls and 8 temporal negatives** (4 future-owner, 4 epoch-mismatch), both kinds and winner/ancestor, through BuildPlan, old-plan Revalidate and both authoritative summaries.
- Independent admittedCheckpoint boundary matrix passes **36 of 36 negative cases**: zero, unsealed field-assembled, copied/cleared seal and copied/cleared token across all nine derivation entries, with the actual admitCheckpoint constructor positive control.
- Closed canonical lease identity, ancestry/creator/persistence, referenced closure, fork, temporal product binding, live changed-record refusal, lagging copies, branching, missing/read-failed distinctions, parked sources, seven capabilities, predecessor/reducer/enums and crash findings remain closed; no new contrary evidence was found. CLI/lifecycle invocation, publication/storage/transport and observation acquisition remain accepted caller bounds.

## Evidence audit and provenance

The producer mutation-source contains **477 files matching the exact candidate**, with zero byte mismatches. All **65 of 65 N/B plants** apply and have named failures with exit 1 in their raw logs. This retains the prior precise classification: 40 semantic-admission kills, 6 functional/selection/order kills, 17 refusal-class/reason precision kills, and now 2 census precision kills. It is not 65 independent product-admission kills. Separate controls: two CONTROL/0, one applied harmless SURVIVED/0, one intentional NOT_APPLIED without execution/log, one COMPILE_OR_HARNESS_FAILURE/1. The callback narrowing mutant's log really fails the unknown_callback.go assertion. These exact-source results are reused; no independent full mutation-battery replay is claimed.

The handoff validation log records **26 of 26 command shards green**, test-case coverage unknown; 5,243,890 bytes are explicitly omitted. It also reports 366 historical board issues with exit 0, not an issue-free board. Producer full test/coverage/race/build/vet evidence is retained with its recorded bounds; independent checks are separately listed in the archive. Formatting uses existing directories. No full independent coverage/race rerun is claimed.

Replay signatures for 46d4a00 and e119aae verify under the configured human key. Replayed sessrepo bytes at 46d4a00 match 7189c41; replayed sessstate bytes at e119aae match 1f34c0c. Final sessrepo additionally includes the already reviewed query leaf's exact-name/attestation changes; it is not byte-identical to the old predecessor as a whole. CR14-to-CR15 sessrepo/sessstate delta is empty. The 55 changed paths and patch digest match the assignment.

## AC accounting — unchanged denominator

**8 of 8 shared API AC rows driven; 7 of 8 established within stated bounds; 1 of 8 partial (negative/gate-proof completeness). 0 of 8 public CLI rows, accepted caller bound.** The producer outcome relabels rows by splitting UUID/name; this review preserves the established denominator and SelectionPlan/refusal rows explicitly.

| Row | Production call site | Named evidence and assessment |
| --- | --- | --- |
| UUID/name | Reader.Resolve → resolveBare; Repository.Resolve | TestResolvePinnedPrecedence, TestResolveBareIdentityUnion, TestResolveLocalNameAndUUID; established |
| Qualified | Reader.Resolve → resolveExplicit → locateSource | TestResolveQualifiedSourcesSelectOneIndex, TestResolveQualifiedSourceNeverFallsBack, TestResolveExplicitSourceReadFailures; established |
| Ambiguity | matchName, selectIdentity, checkRecordAgreement | TestResolveExactNamesAndASCIICollisions, TestResolveBareIdentityUnion; established |
| List | Reader.List, Reader.AuthoritativeList → authorize/winningLeaseFor | TestSummaryBootstrapRefusals, TestAuthoritativeAdmitsFullCapabilityRegistry, original temporal controls; established |
| Status | Reader.Status, Reader.AuthoritativeStatus → authorize | TestAuthoritativeStatusHealthy, TestAuthoritativeMissingProcessRefusesStatus, TestRev11ReferencedCheckpointAdmissionRefusalClass; established |
| Sorting | Reader.List/read, Reader.Resolve/peerCandidates | TestListStatusDerivedFactsAndStableOrder, TestResolvePeerOrderAndReplicatedIdentity; established |
| SelectionPlan | Reader.BuildPlan/bindPlan, Reader.Revalidate/validateAuthorityUnion, ParsePlan | TestBuildPlanBindsCurrentFacts, TestRevalidateDetectsEachFactChange, TestRev11ReferencedCheckpointAdmissionOldPlan; established for exercised scope |
| Negative/refusal proof | Same shared entries, admittedCheckpoint.authority, record-consumption census | Independent alias attacks fail; P2-A remains partial |

Read/plan/summary operations introduce no durable mutation, so new crash/idempotency tests do not apply. No unsupported CLI/doctor/provider capability is advertised.

Verdict: **changes requested**, route to `to-dev`. No external blocker or human decision. This task-scoped verdict and raw evidence are the review logbook. Rework only the alias-aware inventory gap and generated cache cleanup; preserve the closed temporal repair and exact-source evidence.

Direct gate subprocess results are in direct-plants.json: six expected census refusals exit 1; two alias plants survive exit 0; same-instrument applied harmless control survives exit 0. No compile failure or empty selection is counted. Live status and HEAD before/after are identical.

Independent retained suite completed with exit 0 (`retained.log`, exact mask in exits.json): TestReview10/TestReview11 ports, Rev4–8/10–12/14 suites, resolver, plan/revalidation and summaries. Original temporal and copied-capability controls are in independent.log; that command exits 1 solely for the two new alias negative assertions. Reviewer did not rerun full repository/coverage/race; producer evidence above is reused.
