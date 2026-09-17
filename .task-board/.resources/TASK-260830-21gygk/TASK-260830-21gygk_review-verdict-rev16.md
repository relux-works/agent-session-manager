# CR16 review — accepted

Task TASK-260830-21gygk; reviewer RUN-260917-b69146. Immutable candidate `04c4902bb3d3070214812303920ade9da6f3e824`, base `8626fb361c1d6b62d5f95c7be4cb33c85d4aadf3`, patch SHA-256 `625029a813c7a34eb9cc2a3c52e9c628d4917668d9ffe43b24111fac9d342eae`. Authority remains v0.6.0 / `0cbdf100dbf84df50c64f792b1f940e3a67859a6`. Reviewed per governing reviewer-cr16 brief. All work was local in archive-extracted scratch; live product/index/branch/HEAD preserved. No commits, integration, or hosted CI.

## Findings closed

CR15 P2-A is closed. The census normalizes aliases via types.Unalias before semantic identity comparison and recursive value traversal. Inspection confirms pointer, slice, array, map, channel, struct, signature and generic-argument descent. Direct census subprocesses reject both original alias-seal and alias-token plants with exit 1 and actual TestRev11RecordConsumptionCensus failures. Four additional independent plants also fail semantically: alias-to-pointer, alias generic argument, build-tagged alias constructor, and array/channel alias. No type-check failure or empty selection is counted. The same instrument rejects retained new(T), generic constructor, any-token, build-tag producer, unknown callback and struct-field callback plants. Total direct plants: **12 of 12 refused**, plus **1 applied harmless SURVIVED/0** control.

CR15 P3 is closed: exactly 55 changed paths, no __pycache__, .pyc, task-board.config.json or stray plant files.

Independent original temporal controls retain 12 distinct positive controls and 8 negatives (both kinds; winner and ancestor) through BuildPlan, old-plan Revalidate and both authoritative summaries. Independent sealed-capability matrix passes **36 of 36 negatives** across nine derivation entries, with real constructor control. The bounded retained suite covers Review10/11 ports, Rev4–15 suites, referenced closure, fork, historical/branching/lagging semantics, missing/read failures, selectors, summaries, plan construction and revalidation. There is no separately named Rev16 test: rev16 ports are TestRev15AliasAwareRecordConsumptionCensus. Closed CR7–15 product findings remain closed; no new contrary evidence.

## Evidence and limits

Producer mutation logs have **66 of 66 applied N/B KILLED/1** records, with raw named failures matching every classified row. This is 40 semantic-admission, 6 functional/selection/order, 17 refusal-class/reason precision, and 3 census precision kills; not 66 independent runtime admission violations. Two CONTROL/0 rows, one applied harmless SURVIVED/0, one NOT_APPLIED and one COMPILE_OR_HARNESS_FAILURE/1 remain separate. Alias normalization narrowing is killed specifically by TestRev15AliasAwareRecordConsumptionCensus/container_of_alias. Other alias subtests continue refusing via independent identity checks; do not infer that every alias subtest kills this narrowing. Unknown-callback narrowing is also a real named failure.

The producer mutation copy matches all 476 non-document files under internal, including all Go source/tests and the mutation instrument; internal/sessquery/TRACEABILITY.md differs, as do root README.md and LOGBOOK.md. These are post-run documentation updates, not changed behavioral inputs to the focused mutation tests. Reused producer full test/coverage/race/vet/build logs report exit 0; no independent full repository/coverage/race or whole-battery replay is claimed. Handoff validation reports **26 of 26 command shards green**, test-case coverage unknown, with 5,251,048 bytes omitted. Its historical board validation reports 366 issues and exit 0; this is not an issue-free board claim. Tracecheck is a bounded partial-product claim, not complete specification compliance.

Independent go vet ./internal/sessquery exits 0. gofmt over extracted internal lists only the four copied reviewer probes, no candidate files. Replay commits 46d4a00 and e119aae have verified human signatures. sessrepo at 46d4a00 equals predecessor 7189c41; sessstate at e119aae equals predecessor 1f34c0c. CR15-to-16 changes no predecessor or production Go implementation. Live status and HEAD remain unchanged.

## AC accounting

**8 of 8 shared API AC rows driven and established within stated bounds. 0 of 8 public CLI rows, accepted caller bound.** SelectionPlan and negative/refusal proof remain explicit; no denominator reduction.

| Row | Production call site | Named evidence and assessment |
| --- | --- | --- |
| UUID/name | Reader.Resolve → resolveBare; Repository.Resolve | TestResolvePinnedPrecedence, TestResolveBareIdentityUnion, TestResolveLocalNameAndUUID; established |
| Qualified | Reader.Resolve → resolveExplicit → locateSource | TestResolveQualifiedSourcesSelectOneIndex, TestResolveQualifiedSourceNeverFallsBack, TestResolveExplicitSourceReadFailures; established |
| Ambiguity | matchName, selectIdentity, checkRecordAgreement | TestResolveExactNamesAndASCIICollisions, TestResolveBareIdentityUnion; established |
| List | Reader.List, Reader.AuthoritativeList → authorize/winningLeaseFor | TestSummaryBootstrapRefusals, TestAuthoritativeAdmitsFullCapabilityRegistry, original temporal controls; established |
| Status | Reader.Status, Reader.AuthoritativeStatus → authorize | TestAuthoritativeStatusHealthy, TestAuthoritativeMissingProcessRefusesStatus, TestRev11ReferencedCheckpointAdmissionRefusalClass; established |
| Sorting | Reader.List/read, Reader.Resolve/peerCandidates | TestListStatusDerivedFactsAndStableOrder, TestResolvePeerOrderAndReplicatedIdentity; established |
| SelectionPlan | Reader.BuildPlan/bindPlan, Reader.Revalidate/validateAuthorityUnion, ParsePlan | TestBuildPlanBindsCurrentFacts, TestRevalidateDetectsEachFactChange, TestRev11ReferencedCheckpointAdmissionOldPlan; established for exercised scope |
| Negative/refusal proof | Same shared entries, admittedCheckpoint.authority, record-consumption census | TestRev15AliasAwareRecordConsumptionCensus and independent alias attacks refuse; 36/36 sealed-capability negatives pass; established |


Shared read/plan/summary operations add no durable mutations; crash/idempotency remains the closed repository predecessor evidence. CLI/lifecycle boundary invocation, observation acquisition and publication/storage/transport remain accepted caller responsibilities. No unsupported capability is advertised. The record-consumption census is a test-time inventory gate, not an exported runtime entry; runtime seal checks are separately exercised by the 36-case matrix.

Verdict: accepted, route via accept_cr revision 16 to integrating, never done from this reviewer. This verdict and its task-scoped raw archive serve as the review logbook.

Independent `independent` and `retained` commands both completed with exit 0; exact masks and exits are recorded in exits.json.
