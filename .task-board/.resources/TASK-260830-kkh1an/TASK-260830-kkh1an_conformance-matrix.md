# TASK-260830-kkh1an — conformance matrix (rev4)

Authority: spec v0.5.0 (`28bf96d`), worked against
`internal/specdoc/SPEC.v0.7.0.md` (§4.B-4.D, §5.2, §7.A, §13.1, §13.7;
§§4.B/4.C/4.D/5.2/7.A byte-identical to v0.6.0, §13.1 carries the
v0.7.0 launch-plan pure-addition insertion with the terminal-entry
rule text unchanged, §13.7 is the cold force-takeover rule behind
the grant-less fencing verdict).
Every row names the production entry and the committed test that
drives it; 109 named top-level tests verified present via
`go test -list` (107 in-package + the fencing agreement test + the
`terminalbackend` column test) and every cited subtest verified
`--- PASS` (zero missing; see `ratio-check.log` in the evidence
archive). Full clause-to-test evidence with stated bounds:
`internal/terminstance/TRACEABILITY.md` (93/93 AC rows, counted;
rows 1–84 keep rev3 numbers, rows 85–93 are the rev4 rework rows).

## AXAuthorization (§4.C)

| Clause | Production entry | Committed test |
| --- | --- | --- |
| Closed 7-member object | `ParseAXAuthorization` | `TestParseAXAuthorizationAdmitsClosedFixture` |
| lease_id UUIDv4 | `ParseAXAuthorization` | `TestParseAXAuthorizationMemberTypes` |
| lease_epoch bound at 1, max, 0, max+1 | `ParseAXAuthorization` | `TestParseAXAuthorizationEpochBound` |
| Epoch AX number model | `ParseAXAuthorization` | `TestParseAXAuthorizationEpochNumberModel` |
| holder UUIDv7, kind closed 4, timestamps, digest | `ParseAXAuthorization` | member/kind/expiry tests |
| Kind negatives attach + none (direct + nested) | `ParseAuthorizationKind`, `ParseAXAuthorization` | `TestParseAuthorizationKindClosed`, `TestParseAXAuthorizationKindClosed` |
| Unknown/missing member, frame violations | `ParseAXAuthorization` | `TestParseAXAuthorizationClosedShape` |
| Kind, expiry, lease binding; holder unbound (stated) | `CheckAuthorization` | `TestCheckAuthorizationAdmitsWinningLease`, `TestCheckAuthorizationRefusals` |

## Closed enums and mapping (§4.C)

| Clause | Production entry | Committed test |
| --- | --- | --- |
| RetryDisposition closed 4 + census (+ sibling corpus) | `ParseRetryDisposition` | `TestParseRetryDispositionClosed`, `TestClosedEnumsAreExactlyPinned` |
| Per-code mapping (26 rows; unavailable+new_authorization stated) | `DispositionFor` | `TestDispositionForMapping` |
| ProviderProofKind closed 3 + classification (+ sibling corpus) | `ParseProviderProofKind`, `RequiresProviderObservation` | proof tests, `TestRV3W_M4_ProofKindSiblingNamesRefused` |
| AuthorizationKind closed 4 + census (+ sibling corpus) | `ParseAuthorizationKind` | `TestParseAuthorizationKindClosed` |
| Kind mapping delegates to landed column | `kindFor` → `terminalbackend.TransitionAuthorization` | `TestKindForAgreesWithLandedTable`, `TestTransitionAuthorizationReportsEveryTableColumn` |

## Request/result identity (§4.C)

| Clause | Production entry | Committed test |
| --- | --- | --- |
| MutationContext closed 10 + members, doc + Go twins | `ParseMutationContext` | `TestParseMutationContextAdmitsClosedFixture`, `TestParseMutationContextMemberRefusals`, `TestCheckMutationContextVersionArms` |
| Generation 0/256/257 + multibyte | `ParseMutationContext`, `CheckMutationContext` | bound tests |
| Key material exact per row | `KeySegments` | `TestKeySegmentsDerivesRowMaterial` |
| Carried-key consistency | `CheckMutationContext` | `TestCheckMutationContextKeyMaterial` |
| Per-op params | `CheckMutationContext` | `TestCheckMutationContextParams` |
| Operation bodies closed + scope | `ParseOperationBody` | `TestParseOperationBodyAdmitsRows`, `TestParseOperationBodyRefusals`, `TestParseOperationBodyScopeGate` |
| Status body + null-pair rule | `ParseStatusBody` | `TestParseStatusBodyAdmitsBothScopes`, `TestParseStatusBodyNullPairRule`, `TestParseStatusBodyMemberRefusals` |
| Result repetition + shape + sorting (+ evidence edge 256/257) | `CheckResult` | `TestCheckResultAdmitsEngineResult`, `TestCheckResultRepetitionTamper`, `TestCheckResultShapeArms`, `TestBuildResultSortsEffectsAndEvidence`, `TestRV3W_M2_EvidenceBoundEdge` |

## State-entry rules (§4.C, §13.1, §13.7)

| Clause | Production entry | Committed test |
| --- | --- | --- |
| creating only via create after receipt | `ExecuteMutating`, `InterimState` | `TestExecuteInterimState`, `TestExecuteReceiptPrecedesFirstEffect` |
| Pre-first-effect errors restore source | `ExecuteMutating` | `TestExecutePreEffectErrorsRestoreSource` (5 subtests; expired kills on the receipt via the late-deadline instant) |
| Post-commit uncertainty to unavailable | `ExecuteMutating` | `TestExecutePostEffectErrorsMoveUnavailable` (4 subtests) |
| Only quiesce enters quiescing | `ExecuteMutating` | `TestExecuteOnlyQuiesceEntersQuiescing` |
| stale_fenced only via fencing (launch-class op; local + remote winners; live sources) | `ObserveFencing` | `TestObserveFencingStaleEpochFences`, `TestObserveFencingLeaseMismatchFences`, `TestObserveFencingRemoteWinnerStaleEpochFences`, `TestObserveFencingRemoteWinnerLosingLeaseFences`, `TestObserveFencingNonLiveSourcesLeaveState` |
| Non-stale observations leave state | `ObserveFencing` | `TestObserveFencingNonStaleLeavesState`, `TestObserveFencingStaleParkVocabulary`, `TestObserveFencingRemoteWinnerWinningTokenLeavesState`, `TestObserveFencingRemoteWinnerFutureEpochLeavesState` |
| Grant-less stale fences (4 grant members × local/remote × 2 tokens × 3 states); decided-not-stale and undecidable leave state; live-source gate on the verdict path | `ObserveFencing` → `fencing.StaleRelativeToWinner` | `TestRV3F3_GrantLessStaleFences`, `TestRV3F3_GrantLessDecidedNotStaleLeavesState` (4 subtests), `TestRV3F3_GrantLessUndecidedLeavesState` (9 subtests), `TestRV3F3_GrantLessNonLiveSourcesLeaveState`, `TestRV3_FencingNoLocalTupleComparison` |
| Verdict contract (tuple arms, grant-independence, direction, undecided members, gate agreement) | `fencing.StaleRelativeToWinner` | `TestStalenessTupleArms`, `TestStalenessIgnoresGrantPrecondition`, `TestStalenessRemoteWinnerIsStillStale`, `TestStalenessUndecidedMembers`, `TestStalenessAgreesWithGateOnHotPath` (`internal/fencing`) |
| Per-effect auth+generation rechecks (every position; tuple + expiry axes) | `ExecuteMutating` | `TestExecuteRechecksBeforeEachEffect` (4 subtests), `TestExecuteRechecksBeforeThirdEffect` (2 subtests), `TestRV3W_M1_AuthExpiryRecheckedBeforeEachEffect` |
| Deadline cancels waiting only (+ instant on both entries) | `ExecuteMutating`, `ExecuteStatus` | `TestExecuteDeadlineNeverCancelsCommittedEffect`, `TestExecuteEntryDeadlineCodes` (4 subtests), `TestExecuteEntryDeadlineInstantRefuses` (4 subtests), `TestExecuteStatusDeadlineInstantRefuses` |
| Pre-link store failure restores source | `ExecuteMutating` | `TestExecutePreLinkStoreFailureRestoresSource`, `TestExecuteStoreLookupFailureIsUncertain` |

## Generation-safe evidence (§4.B, §7.A)

| Clause | Production entry | Committed test |
| --- | --- | --- |
| Bound in every owned surface | parse/check entries | rows 26, 36, 39 tests |
| Stale refuses landed stale class | `ExecuteMutating` | entry + per-effect staleness tests |
| Verdict agrees with landed digest | `checkGenerationBound` | `TestGenerationBoundAgreesWithLandedDigest` |
| Row codes admitted by landed table | `CheckErrorAllowed` | `TestRowCodesAreAdmittedByCheckErrorAllowed` |
| Descriptor bound | landed `terminalbackend` | cited landed (checkGeneration arms) |

## Operation rows (§4.C, §5.2)

| Clause | Production entry | Committed test |
| --- | --- | --- |
| status same-state all 8 + unknown source, zero effects | `ExecuteStatus` | `TestExecuteStatusAllSources` |
| status adopts drifted state verbatim | `ExecuteStatus` | `TestExecuteStatusAdoptsReportedState` |
| status non-match adopts proven absent | `ExecuteStatus` | `TestExecuteStatusNonMatchAdoptsAbsent` |
| status unknown never absent | `ExecuteStatus` | `TestExecuteStatusUnknownIsNeverAbsent` |
| status report validation + member arms | `ExecuteStatus` | `TestExecuteStatusReportValidation`, `TestExecuteStatusMatchedTupleFalseFormRefuses`, `TestExecuteStatusReportMemberRefusals` |
| status conditionals + session scope + session/axis drift | `ExecuteStatus` | `TestExecuteStatusProviderConditional`, `TestExecuteStatusSessionScoped`, `TestExecuteStatusSessionDriftRefuses` (4 subtests), `TestExecuteStatusIdentityMatchEachAxis` (10 subtests) |
| status last_operation_id grammar (empty/garbage/upper refuse) | `ExecuteStatus` | `TestRV3W_M8_StatusLastOperationGrammar` |
| quiesce rows + refusal set | `ExecuteMutating` | `TestExecuteQuiesceSuccess`, `TestExecuteBackendRowErrorSets` |
| wait rows + capabilities | `ExecuteMutating` | `TestExecuteWaitSuccess`, `TestExecuteWaitCapabilityConditionals` |
| stop rows + order + both timeout directions | `ExecuteMutating` | `TestExecuteStopSuccess`, `TestExecuteBackendRowErrorSets` |
| create receipt rule + entry gates (expired kills on receipt) + committed timeout | `ExecuteMutating` | `TestExecuteCreateReceiptRule`, `TestExecuteCreateEntryAuthorizationRefuses` (3 subtests), `TestExecuteCreateEntryGenerationRefuses`, `TestExecuteCreateHeadlessFromAbsentNeedsCapability`, `TestExecuteCreateCommittedTimeoutReportsStatusFirst` |
| Capability-conditional disposition at the engine | `ExecuteMutating` | `TestRV3W_M3_ConditionalCapabilityDisposition` (3 subtests) |
| Replay, mismatch, uncertain retry, sweep age-gate | `ExecuteMutating`, `ReceiptStore` | `TestExecuteIdenticalRetryReplays`, `TestExecuteChangedOperationInWindowMismatches`, `TestExecuteUncertainRetryRequiresStatus`, `TestStoreSweepKeepsLiveStaging`, `TestStoreSweepRemovesAgedStaging` |
| Same-key resumption after status proof | `ExecuteMutating` (`resumeUncertain`) | `TestExecuteUncertainRetryResumesWhenStatusProvesSource`, `TestExecuteRequestStopResumesWhenNotClosed`, `TestExecuteCreateResumesWhenStatusProvesAbsent`, `TestExecuteUncertainRetryTargetProvenRefuses` (3 subtests), `TestExecuteUncertainRetryUnknownObservationRefuses` (3 subtests), `TestExecuteHookErrorRetryReconcilesThroughStatus` (3 subtests) |
| Crash seams + status-gated resumption | `ReceiptStore`, `ExecuteMutating` | `TestExecuteCrashChildSelfTerminates`, `TestExecuteCrashCreateResumesAfterStatus` |
| Conferral wiring + scope gate | `ExecuteMutating` | `TestExecuteEmptyAdmittedRefusesAtLandedGate` (4 subtests), `TestExecuteScopeGate` |

## Negative-evidence index (narrowing mutants)

116/116 harness rows on two full passes with identical verdict lists:
114 narrowing KILLED + 1 tightening edge KILLED + control
(`mutants/pass1/`, `mutants/pass2/`, 116 per-plant raw logs each with
subprocess exits). One narrowing mutant per auth arm
(`N-auth-*`, incl. `N-auth-kind-attach`, `N-auth-kind-none`), per
mapping arm (`N-disp-*`), per enum (`N-proof-vocab`,
`N-proof-provider`, `N-proof-sibling`, `N-disp-parse`), per
key/generation/timeout/version arm (`N-context-*`, `N-params-*`,
`N-version-delegation`, `N-body-key`, `N-wait-timeout-bound`), per
repetition member (8× `N-result-*`), per order/unique/sort/bound half
(`N-result-effects-order`, `N-result-evidence-unique`,
`N-result-effects-sort`, `N-result-evidence-sort`,
`N-result-evidence-bound`), per body gate (`N-body-scope`,
`N-body-members`), per receipt gate (`N-store-*`,
`N-complete-unbound`), per fencing arm (`N-fencing-remote`,
`N-fencing-operation`, `N-fencing-operation-remote`,
`N-fencing-remote-stale-only`, `N-fencing-source-*`, 11×
`N-fencing-verdict-*` mutating the landed verdict and killed through
`ObserveFencing`, `N-fencing-verdict-composition`;
`N-fencing-epoch-only` retired in rev4 with its equivalence proof),
per engine gate (`N-engine-*` incl. create-exempt entry,
quiesce-side entry, third-effect recheck, expiry recheck,
headless, committed-timeout, conditional-disposition,
deadline-instant, and bind pre/post-link rows;
`N-status-session-*`, `N-status-match-*`,
`N-status-last-operation-empty`, `N-status-deadline-instant`,
`N-status-nullpair`, `N-interim`), per resumption arm
(`N-resume-status-error`, `N-resume-contradiction`, `N-resume-target`,
`N-resume-scope-instance`, `N-resume-complete-key`,
`N-resume-generation-value`), and the `C-control` SURVIVED control
(see TRACEABILITY.md mutant table).
