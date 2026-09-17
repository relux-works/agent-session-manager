# CR13 review — changes requested

Reviewer RUN-260916-9cefe4. Exact candidate tree `f6f92e676df1e10adac55ef340ecfe5e759a67e7`, base `8626fb361c1d6b62d5f95c7be4cb33c85d4aadf3`, patch SHA-256 `a532c2f23875c8f4574b1b4305c7cc2e567018b4191fcee79e88f6cb8a80dac3`. Authority: pinned v0.6.0 / `0cbdf100dbf84df50c64f792b1f940e3a67859a6`. No product/index/branch/HEAD edits, commits, integration, or hosted CI. All probes ran in an extracted immutable copy on darwin/arm64, Go 1.25.5. All 479 extracted source/module/README blobs match the candidate after restoration.

CR12 to CR13 differs only in task-board.config.json; that file is absent from the base-to-candidate changed paths. Signed replay checkpoints 46d4a00 and e119aae verify with Ivan Oparin's configured key. Their sessrepo and sessstate content is respectively unchanged from predecessor checkpoints 7189c41 and 1f34c0c. Patch digest matches the assignment. Live worktree status before/after is identical.

## P2-A: the claimed package-wide admission boundary remains bypassable

Repeat-of CR11 P2, class-closing semantic consumption census. This is not a new current-input creator vulnerability: existing behavioral tests correctly kill the bypass. It is an unmet explicitly assigned enforcement/evidence requirement.

`rev11_regression_test.go:865` records only `function.Name.Name`; lines 783/791 grant raw-record privileges by that string. A method on an unrelated new receiver named `admitCheckpoint` inherits the free function's privilege. Lines 769–773 inspect only nonempty composite literals; a zero value populated by field assignments is invisible. `admittedCheckpoint` is a mutable package struct with no admission seal; unexported visibility alone does not make it unforgeable inside this package.

Independent `TestReview13GateShapes` exercises five plants through the real census:

- interface assertion consuming a raw checkpoint: rejected;
- package-level closure consuming raw checkpoint: rejected;
- undefined/unresolvable callee: rejected at type check (not a behavioral kill);
- method on a new receiver named admitCheckpoint: incorrectly admitted;
- field-by-field construction of admittedCheckpoint outside admission: incorrectly admitted.

The last two are real census failures (test exit 1), not compile failures. `TestReview13ZeroCapability` also demonstrates that `sessionCreationProfile(admittedCheckpoint{}, validRecord)` successfully derives `yolo` with no admission (exit 1 of the negative assertion). This helper-level demonstration is not claimed as external Reader exploitability.

A reachable narrowing plant keeps every original admission call/token and only redirects hostB-created references in `buildCheckpointAdmission` to a new receiver method named admitCheckpoint. That method copies the raw fields into a zero-value capability without admission. Results:

| Instrument | Actual exit | Classification |
|---|---:|---|
| TestRev11RecordConsumptionCensus | 0 | SURVIVED incorrectly |
| TestReview10ReferencedCheckpointAdmission + TestRev11ReferencedCheckpointAdmissionRefusalClass + TestRev11ReferencedCheckpointAdmissionOldPlan | 1 | Behavioral KILLED; wrong_creator cases admit through BuildPlan, fresh Revalidate and both summaries; old-plan refusal class changes |
| Applied harmless comment, same behavioral mask and subprocess runner | 0 | SURVIVED correctly |

`attack.py`, exact planted source, logs and subprocess return codes are attached. Product bytes are restored. The original CR11 review11_bypass.go was also tested through the new census; its old raw-argument signatures now fail type checking. Credit that as rejection, not a compiling behavioral kill. The new reachable plant above supplies the missing compiling attack.

Required repair: bind admission privileges to the actual declaration/object and receiver identity; enforce every way of producing or using the authority capability, including zero-value field assignment and indirect construction. Keep the scope to this package boundary, with negative alternate-path tests and a reachable narrowing plant that the inventory gate itself detects. A new name allowlist or another lexical token check is insufficient. Do not reopen the closed production creator/variant/owner findings.

## P2-B: final-source mutation evidence is overstated and four shipped plants are unapplied

Repeat-of the final-source/accounting obligation in CR11/CR13 briefs; new regression caused by this refactor, not invalidation of the previously reviewed CR11 logs.

Executed the actual shipped `run` function and four actual AST call expressions on the final source. All four return NOT_APPLIED before executing tests:

| Shipped plant | Anchor count |
|---|---:|
| N-checkpoint-holder | 0 |
| N-ancestor-holder | 0 |
| N-checkpoint-heads | 2 |
| N-profile-resume-missing | 0 |

The current file still has 63 N/B entries, but its inventory is not the old 63: N-referenced-lease is replaced by N-referenced-temporal, and shared admission changes several anchors/bodies. Therefore the producer statement that the prior 63 are reused as prior exact-source evidence is not supported. The 59 unique current anchors are application evidence only, not 59 kills.

The old CR11 audit remains valid for its source: 46 semantic plus 17 precision plants. Rev12's focused raw logs report a temporal behavioral kill, a census precision kill, and an applied harmless survivor; inspected those logs and the subprocess exit wrapper. The boundary kill must stay classified as census/precision, not semantic product admission. No full 63-plant replay was performed in this review; no NOT_APPLIED, compile failure, or empty selection is counted as a behavioral kill.

Required repair: update/retire redundant moved plants with an explicit gate mapping, refresh affected final-source evidence, preserve a qualifying narrowing attack for each guard (including missing reference and event heads), and report precise semantic/precision/control denominators. Full unrelated historical batteries need not be repeated.

## Closed temporal finding and evidence limits

CR11 P1 is closed for its exact independent regression. I reran the original attached review11 fixture/test with symbols renamed only, not merely the producer's replacement fixture: **4 future-owner negatives refuse, 4 epoch-mismatch negatives refuse, 12 real ordinary/pre-change/divergent controls pass**, including old-plan refusal and BuildPlan/fresh Revalidate/AuthoritativeStatus/AuthoritativeList paths. `original-and-retained.log` exits 0 and also covers retained historical/profile-source negatives. The shipped temporal suite plus later-lease control and new census tests independently exit 0 in baseline.log.

The producer's new temporal fixture labels good_prechange/good_divergent as controls but does not actually vary those modes (line 345 groups them with good). Its relation census additionally describes good_divergent as refusing, although the test expects success. Preserve the actual historical/divergent distinctions when porting the original test and correct the census wording. This is evidence precision, not a demonstrated product regression: the original controls pass here.

Own additional temporal probes: eight ordinary/later-lease controls (both session kinds and winner/necessary-ancestor) pass. Attempts to construct same-lease out-of-predecessor and earlier-owner out-of-predecessor references through the real repository are rejected while appending events: omitted prior authoritative predecessor and decreasing lease epoch. Their eight failed fixture cases in temporal-own.log are **not** temporal-admission test failures and **not** passing Reader negative evidence. A same-chain future head can also induce a content-digest cycle (resume -> checkpoint -> later head -> resume); this review does not fabricate such immutable objects. These requested extra Reader relationships remain an explicit verification bound; no production temporal defect is inferred from the failed fixture attempts.

Interpretation checked against pinned 5.2/5.3/5.4/13.10: checkpoint heads describe authority before the checkpoint, resume consumes the checkpoint, and future-owner membership in the current ancestry does not suffice. The new owner-index and resume predecessor-closure checks implement that relationship on the exercised paths. The four summary/plan entries converge on shared winningLeaseFor/admitCheckpoint; fork/profile derivation signatures use admittedCheckpoint. The enforcement weakness above concerns alternate construction, not a current duplicate resolver.

Closed canonical lease identity, missing/read-failed distinction, parked sources, seven capabilities, ancestry creator/variant, lagging copies, branching, predecessor/reducer/enums and creation-crash findings remain closed; no contradictory production evidence found and no unrelated replay demanded. Shared API is read-only; no new durable crash surface. Caller-owned publication/storage/transport, observations, CLI wire mapping and lifecycle boundary invocation remain accepted bounds.

## Validation and AC accounting

Reused exact immutable CR13 handoff validation: **26 of 26 configured command shards green**, including local build, vet, tests and diff checks; the log's board validation reports 366 historical issues with exit 0, not an issue-free board. Reused producer repository-wide test/coverage logs (sessquery 86.5%, sessrepo 87.2%, sessstate 92.2%). No independent full-suite/coverage/race or full mutation replay claimed. Fresh bounded probes and raw exits are attached; formatting evidence uses existing internal paths. No new CLI/doctor capability is advertised. Remove the generated `internal/sessquery/testdata/__pycache__/mutate.cpython-314.pyc` from the candidate as repository hygiene.

**8 of 8 AC rows driven at shared production APIs; 7 of 8 established within stated bounds, 1 of 8 partial (gate-proof completeness). 0 of 8 public CLI rows delivered, accepted caller bound.**

| AC row | Production call / named candidate test | Assessment |
|---|---|---|
| UUID/name | Reader.Resolve/resolveBare; TestResolvePinnedPrecedence, TestResolveBareIdentityUnion | Established, retained evidence |
| Qualified | Reader.Resolve/resolveExplicit/locateSource; TestResolveQualifiedSourcesSelectOneIndex, TestResolveQualifiedSourceNeverFallsBack, TestResolveExplicitSourceReadFailures | Established, retained evidence |
| Ambiguity | matchName/checkRecordAgreement; TestResolveExactNamesAndASCIICollisions, TestResolveBareIdentityUnion | Established, retained evidence |
| List | Reader.AuthoritativeList/authorize/winningLeaseFor; TestSummaryBootstrapRefusals, TestAuthoritativeAdmitsFullCapabilityRegistry, TestReview11ReferencedTemporalAuthority | Established for exercised scope |
| Status | Reader.AuthoritativeStatus/authorize; TestAuthoritativeStatusHealthy, TestAuthoritativeMissingProcessRefusesStatus, TestRev11ReferencedCheckpointAdmissionRefusalClass | Established for exercised scope |
| Sorting | Reader.List/read/Resolve; TestListStatusDerivedFactsAndStableOrder, TestResolvePeerOrderAndReplicatedIdentity | Established, retained evidence |
| SelectionPlan | Reader.BuildPlan/bindPlan, Reader.Revalidate/validateAuthorityUnion, ParsePlan; TestBuildPlanBindsCurrentFacts, TestRevalidateDetectsEachFactChange, TestRev11ReferencedCheckpointAdmissionOldPlan | Established for exercised scope; new temporal fixture limits stated above |
| Negative/refusal proof | Same shared APIs plus TestRev11RecordConsumptionCensus and testdata/mutate.py | Partial: P2-A/P2-B |

Verdict: **changes requested**, route `to-dev`; do not accept_cr. This task-scoped verdict records the review findings/logbook. Preserve all closed findings and the 8-row denominator. No commit acknowledgement or integration by reviewer.
