# TASK-260830-21gygk — CR11 changes requested

Reviewer RUN-260916-5a0c3f. Immutable tree `0cb391bd71143364e40be6c4cfab90195c1d36cd`, base `9ff7d2c1d4d391dbc58d40b812757052d324a775`, patch SHA-256 `c7269ed32232a533ab7af539fe8f8e0ed2c18c605c08ef0be2479e7f2dbc9a9f`. Authority: pinned AX v0.6.0 / `0cbdf100dbf84df50c64f792b1f940e3a67859a6`. All 3,076 archive blobs match Git; all five producer manifest hashes match this candidate. CR10→CR11 changes only lease.go, rev11 tests, mutation harness, TRACEABILITY and LOGBOOK. Live Story status and HEAD are unchanged. No product/index/branch mutation, commit, integration or hosted CI.

## P1 — current ancestry membership admits a future checkpoint owner

repeat-of: CR10 P1 / CR7–CR10 cross-record semantic-authority family. The three specific CR10 defects are fixed; this is a remaining temporal relationship in the same referenced-record admission owner.

`checkReferencedCheckpointBinding` (`internal/sessquery/lease.go:890`) locates the referenced checkpoint's lease tuple anywhere in the CURRENT winning ancestry. `referencedCheckpointProfile` (932) has the consuming resume's EventSummary but does not bind that owner to the resume's lease. Therefore a resume under epoch 1 can consume a checkpoint claiming epoch 2, provided epoch 2 appears later in the supplied ancestry. Creator, persistence variant, checkpoint heads, canonical identity and profile pair all still pass.

Independent `TestReview11ReferencedTemporalAuthority` changes the otherwise valid referenced checkpoint to `(lease_epoch=2, lease_id=leaseB)`, recomputes its canonical ID, then places its reference in the real epoch-1 resume. The epoch-2 lease is genuinely present in the winning ancestry; its own checkpoint contains that earlier resume. This is not an absent-record fixture and no product mutant is needed. It admits in all **4 of 4** direct/task_board × winner/necessary-ancestor variants:

- `Reader.BuildPlan` succeeds;
- freshly built `Reader.Revalidate` returns nil;
- `Reader.AuthoritativeStatus` and `Reader.AuthoritativeList` succeed.

This reverses the authority dependency: SPEC 5.4's checkpoint belongs to the winning lease at its boundary, with event heads immediately before that object; 5.2 binds resume to that referenced checkpoint; 13.10 derives and validates the checkpoint before emitting resume. A future lease's checkpoint cannot authorize an earlier lease's resume. Checking membership in today's full ancestry does not establish authority at the consuming event. This is an inference from those combined ordering requirements, not a new storage/publication feature. The shared owner already has the consuming event, record tuple and ancestry inputs.

Two independently varied relationships were exercised: (a) admitted but future owning lease, above; (b) mismatched epoch/fencing tuple (`epoch=2` with the epoch-1 token), which correctly refuses in **4 of 4** variants. **12 of 12** valid controls pass: ordinary, historical pre-change, and preserved divergent-event references, each across kind and winner/ancestor. The test exits 1 solely on the four future-owner admission cases; no fixture/compile failure.

Old-plan bound: every negative first builds and revalidates a valid plan over a narrower required closure ending at `session.stopped`, then restores the full closure. Future-owner cases already refuse `selector_plan_stale: winning lease changed`; epoch/token mismatches refuse `selector_observation_unavailable`. Preserve those refusals. This review does not claim old plans admit, and introduces no error-order requirement. Fresh plans and summaries independently demonstrate P1.

Required rework: bind referenced checkpoint authority to the consuming resume's historical lease position through the shared admission owner, preserving valid same/earlier-owner references, historical closures, branching and lagging copies. Port the attached controls and negatives, add a qualifying narrowing mutant, and audit this relationship as part of the whole K4 admission path rather than adding a fixture-token exception. No caller-owned storage, transport, publication or lifecycle implementation is requested.

## P2 — record-consumption census does not detect an alternate consumption path

repeat-of: CR10 requested class-closing consumption audit; same canonical-map-versus-admitted-authority family. This is not a reopening of the closed CR10 mutation-accounting finding.

`TestRev11RecordConsumptionCensus` (`rev11_regression_test.go:293`) parses only `lease.go`, checks seven named call edges and compares first lexical line positions in one function. It neither enumerates package consumption sites nor establishes control-flow dominance. The result is a static precision check, not proof that every K1–K7 consumption traverses its owner. The relation census statement that it fails if *any* K4 derivation bypasses admission is unsupported.

An isolated alternate-path plant adds `review11_bypass.go`: a new helper copies a checkpoint map entry before admission, selects exactly hostB-created references, and derives their profile through a second helper without invoking the admission owner. The original function keeps every searched-for call and its original order; it merely dispatches this selected branch to the new file. Thus the plant changes reachable behavior, not dead code.

- Static census: **SURVIVED, exit 0**.
- Behavioral `TestReview10ReferencedCheckpointAdmission|TestRev11`: **KILLED, exit 1**, with wrong_creator admission failures through BuildPlan, fresh Revalidate and both summaries; the census also passes inside this run.
- Original lease.go restored byte-for-byte; temporary production helper removed. Reproduction script, mutated source, helper and raw exits are attached.

The behavioral tests successfully protect the known creator defect; credit that evidence. They do not make the census a package-wide admission proof. Required rework: make the claimed consumption inventory enforce the actual scoped call/consumption boundary, including new-file/helper paths, with an alternate-path mutant that the inventory gate itself detects; label source-order/token checks as precision and keep behavioral narrowing evidence separately. Do not substitute another hand-picked list for complete record-path coverage or claim source order proves execution order.

## Closed findings, evidence reuse and accounting

Independently reran ported `TestReview10ReferencedCheckpointAdmission` and all `TestRev11*` on the immutable source: exit 0. The **12 CR10 negatives refuse**, **4 controls pass**, 12 class pins and 12 old-plan cases pass. Creator, variant and absent owning-lease repairs are closed for their exercised paths.

Inspected 35 mutation manifests plus matching raw logs: **63 unique N/B plants**, all recorded KILLED with exit 1 and named failures; zero manifest/raw failure-list discrepancies. Inventory remains **46 semantic-admission + 17 precision**, not 63 admission kills. New creator/variant/lease plants each show actual admitted-invalid-case lines; no fixture/compile failures count. Dedicated applied harmless control SURVIVED exit 0 through the same instrument; slice2's cache-corrupted harmless attempt is not counted. NOT_APPLIED and COMPILE_OR_HARNESS_FAILURE are excluded. The runner uses subprocess return codes and restores exact bytes. All 63 current anchors occur once; not-applied is zero. See mutation-audit.md/json.

Bounds: no independent full 63-plant replay. Three new plants reuse verified final-source producer logs; three affected old plants have focused producer rev11 reruns; the remaining 57 reuse CR10 reviewed evidence with unchanged anchors/bodies and the documented signature-threading delta. Closed canonical Lease identity, required parked sources, seven capabilities, predecessor/reducer/enums, ancestry creator/variant, lagging copies and creation-crash findings remain closed. This review does not demand their unrelated replay.

Reused immutable handoff validation: **5 of 5 commands exit 0** (formatting on existing tracked Go paths, build, vet, `go test ./... -count=1 -v`, diff check). Reused producer full-suite and coverage evidence: sessquery 86.8%, sessrepo 87.2%. No fresh repository-wide coverage/race run claimed. Independent targeted results and exact commands are in the evidence archive. Full replay is unnecessary after a decisive new production failure. All CI local.

Missing supplied checkpoint and failed repository read remain distinct branches: lookup absence refuses unavailable without pretending a read succeeded; repository errors retain their cause. Mid-derivation I/O races remain the stated defensive bound, not independently simulated or counted as a new behavioral proof. Targeted retained missing-source/historical/profile suites independently pass (exit 0, 30.984s), recorded in retained-controls.log.

## AC accounting and caller boundary

**8 of 8 AC rows driven through shared production APIs; 4 of 8 established, 4 of 8 partial/failing. 0 of 8 public CLI rows delivered (accepted stated bound).** Named tests are frozen in the managed candidate; the new reviewer regression is attached for rework, not silently added to it.

| AC row | Production call and named evidence | Result |
|---|---|---|
| UUID/name | Reader.Resolve/resolveBare; TestResolvePinnedPrecedence, TestResolveBareIdentityUnion | Established |
| Qualified | Reader.Resolve/resolveExplicit/locateSource; TestResolveQualifiedSourcesSelectOneIndex, TestResolveQualifiedSourceNeverFallsBack, TestResolveExplicitSourceReadFailures | Established |
| Ambiguity | Reader.Resolve/matchName/checkRecordAgreement; TestResolveExactNamesAndASCIICollisions, TestResolveBareIdentityUnion | Established |
| List summaries | Reader.AuthoritativeList/authorize/winningLeaseFor; TestSummaryBootstrapRefusals, TestAuthoritativeAdmitsFullCapabilityRegistry, TestReview10ReferencedCheckpointAdmission | Partial: P1 |
| Status summaries | Reader.AuthoritativeStatus/authorize/winningLeaseFor; TestAuthoritativeStatusHealthy, TestAuthoritativeMissingProcessRefusesStatus, TestRev11ReferencedCheckpointAdmissionRefusalClass | Partial: P1 |
| Sorting | Reader.List/read/Resolve; TestListStatusDerivedFactsAndStableOrder, TestResolvePeerOrderAndReplicatedIdentity | Established |
| SelectionPlan | Reader.BuildPlan/bindPlan (plan.go:274), Reader.Revalidate (revalidate.go:101), ParsePlan; TestBuildPlanBindsCurrentFacts, TestRevalidateDetectsEachFactChange, TestRev11ReferencedCheckpointAdmissionOldPlan | Partial: P1 |
| Negative/refusal | Same shared entries; TestRev8ProfileSourceRefusals, TestRev10*, TestRev11*, narrowing battery; independent temporal regression and alternate-path attack | Partial: P1/P2 |

All affected entries converge on winningLeaseFor → required checkpoint profile closure → referencedCheckpointProfile. K1–K7 and R1–R6 have named exercised rows, but row reachability does not establish complete semantic admission; K4 remains partial. Source search finds no production CLI importer of sessquery. CLI invocation, wire Result/exit mapping, lifecycle effects and publication/storage/transport/materialization remain accepted caller responsibilities. README makes no new CLI/doctor availability claim; correct the stronger conformance/census claims rather than inventing a public surface.

Verdict: **changes requested**, route `to-dev`, no accept_cr. This task-scoped verdict is the review logbook. Attach verdict and reproducible evidence before routing; leave failing checklist gates unchecked. No commit_ack, checkpoint or integration.
