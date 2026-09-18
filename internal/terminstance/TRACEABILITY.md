# internal/terminstance — TRACEABILITY

Authority: `internal/specdoc/SPEC.v0.7.0.md` §4.B (generation bound),
§4.C (lifecycle and operations), §4.D (capability conditionals), §5.2
(status/unknown discipline), §7.A (descriptor generation binding, cited
landed), §13.1 (durable terminal entry before provider launch), §13.7
(cold force takeover: the prior owner is stale when it learns the
winner). Every
row names the production entry and the committed test that drives it;
every expectation asserts the literal specification token at the entry.
Coverage: **99 of 99 AC rows driven** (counted below; verified by
`go test ./internal/terminstance/ -count=1 -v`). Rows 73–84 are the
rev3 rework rows (recovery resumption, remote-winner fencing, create
gates, recheck positions, session axes, deadline instants, kind
negatives, landed kind delegation, store pre/post-link); rows 85–93
are the rev4 rework rows (grant-less fencing verdict, expiry recheck,
evidence edge, conditional disposition, proof siblings, last-operation
grammar). Rows 1–84 keep their numbers; rows 46–48's grant-less and
expiry gaps (rev3 review) are closed by rows 85–89. Rows 94–99 are the
story-close rows (final leaf TASK-260830-2056mm): the third-position
expiry recheck, the engine replay-site generation binding and corrupt
image, the in-loop deadline instant, and the mixed-case enum siblings
from the rev4 review's residual P3-1 through P3-4.

## Landed extensions

Every addition extends a landed type or gate; nothing is forked:

| Addition | Extends |
| --- | --- |
| `ParseAXAuthorization`, `CheckAuthorization` | `terminalbackend.ParseAttachAuthorization` twin (shape discipline, strict expiry, binding checks); frame and member grammars delegated to `environ`/`scalar` |
| `ParseRetryDisposition`, `DispositionFor` | New leaf contract over the landed row-code vocabulary (`terminalbackend` code consts aliased, never retyped) |
| `ParseProviderProofKind`, `RequiresProviderObservation` | New leaf contract over the §4.C proof vocabulary |
| `ParseMutationContext`, `CheckMutationContext`, `KeySegments` | `terminalbackend.IdempotencyKey` shapes, `ParseID`, `CheckVersionTuple`, `GenerationDigest` |
| `ParseOperationBody`, `ParseStatusBody` | `environ.DecodeStrictObject` closed-shape discipline |
| `Result`, `CheckResult` | `terminalbackend` state/effect vocabularies, `scalar` digests |
| `Engine.ExecuteMutating`, `ExecuteStatus` | `terminalbackend.CheckTransition`, `CheckErrorAllowed`, `CheckOperation`, `CheckStatusResult`; uncertain retries reconcile through `ExecuteStatus` itself (no second status path) |
| `kindFor` | `terminalbackend.TransitionAuthorization` (landed authorization column; the engine honors create/control, no restated table) |
| `ReceiptStore` | `terminalbackend.Receipt`/`Ledger`/`ImportLedger` triple and line encoding (byte-agreed, crash-durable) |
| `ObserveFencing` | `fencing.Authorize` + `ParkDetails` asked twice (present token, then relative-to-winner from the winner's host), plus the landed `fencing.StaleRelativeToWinner` verdict for grant-gated refusals; stale verdict only; no local tuple comparison; live sources only |
| Engine error codes | `terminalbackend` code consts aliased; the four literals the landed core spells only inside its allowed-error table (`terminal_backend_timeout`, `terminal_backend_process_failed`, `quiesce_timeout`, `stop_timeout`) pinned admitted by `TestRowCodesAreAdmittedByCheckErrorAllowed` |

## AC rows

### AXAuthorization (item 1) — `ParseAXAuthorization`, `CheckAuthorization`

| # | Clause | Committed test |
| --- | --- | --- |
| 1 | Closed 7-member object admitted, every member literal | `TestParseAXAuthorizationAdmitsClosedFixture` |
| 2 | lease_id is UUIDv4 (v7, garbage, number refuse) | `TestParseAXAuthorizationMemberTypes` |
| 3 | lease_epoch 1 admitted | `TestParseAXAuthorizationEpochBound` |
| 4 | lease_epoch 9007199254740991 admitted | `TestParseAXAuthorizationEpochBound` |
| 5 | lease_epoch 0 refused | `TestParseAXAuthorizationEpochBound` |
| 6 | lease_epoch 9007199254740992 refused | `TestParseAXAuthorizationEpochBound` |
| 7 | Epoch AX number model (fraction, exponent, sign, string, bool, null, huge refuse) | `TestParseAXAuthorizationEpochNumberModel` |
| 8 | holder_host_id is UUIDv7 | `TestParseAXAuthorizationMemberTypes` |
| 9 | authorization_kind closed create/control/force_stale/restore | `TestParseAXAuthorizationKindClosed` |
| 10 | Strict expiry (equal and inverted refuse unauthorized) | `TestParseAXAuthorizationStrictExpiry` |
| 11 | authorization_evidence_id is a digest | `TestParseAXAuthorizationMemberTypes` |
| 12 | Unknown member refused | `TestParseAXAuthorizationClosedShape` |
| 13 | Missing member refused | `TestParseAXAuthorizationClosedShape` |
| 14 | Frame violations (non-object, trailing, duplicate, surrogate, non-UTF8) refused | `TestParseAXAuthorizationClosedShape` |
| 15 | Kind binding (control presented as create refuses) | `TestCheckAuthorizationRefusals/kind_mismatch` |
| 16 | Expiry at and past the edge refuses | `TestCheckAuthorizationRefusals/expired`, `/expiry_edge_refuses` |
| 17 | Lease identity drift refuses | `TestCheckAuthorizationRefusals/lease_identity_drift` |
| 18 | Lease epoch drift refuses | `TestCheckAuthorizationRefusals/lease_epoch_drift` |

### Closed enums (item 2) — `ParseRetryDisposition`, `ParseProviderProofKind`, `ParseAuthorizationKind`, `DispositionFor`

| # | Clause | Committed test |
| --- | --- | --- |
| 19 | RetryDisposition closed replay_same/status_first/new_authorization/required_operator_action | `TestParseRetryDispositionClosed`, `TestClosedEnumsAreExactlyPinned` |
| 20 | Per-code disposition mapping (26 literal rows) | `TestDispositionForMapping` |
| 21 | ProviderProofKind closed provider_quiescence/provider_process_exit/ax_checkpoint_boundary | `TestParseProviderProofKindClosed`, `TestClosedEnumsAreExactlyPinned` |
| 22 | Provider classification (2 provider kinds, AX boundary not) | `TestRequiresProviderObservation` |
| 23 | AuthorizationKind closed create/control/force_stale/restore | `TestParseAuthorizationKindClosed`, `TestClosedEnumsAreExactlyPinned` |

### Request identity (item 2) — `ParseMutationContext`, `CheckMutationContext`, `KeySegments`

| # | Clause | Committed test |
| --- | --- | --- |
| 24 | Closed 10-member context admitted, every member literal | `TestParseMutationContextAdmitsClosedFixture` |
| 25 | Context member grammars (UUIDs, backend ID, versions, key, deadline, nested auth), document and Go-level twins | `TestParseMutationContextMemberRefusals`, `TestCheckMutationContextVersionArms` |
| 26 | Generation 0/257 refuse stale class, 256 and multibyte-256 admit | `TestParseMutationContextMemberRefusals`, `TestParseMutationContextGenerationBound256Multibyte` |
| 27 | Key material exact per engine row (4 literal derivations) | `TestKeySegmentsDerivesRowMaterial` |
| 28 | Carried-key inconsistency refuses protocol, never mismatch | `TestCheckMutationContextKeyMaterial` |
| 29 | Per-op params (bootstrap, quiescence, proof, timeouts, evidence) | `TestCheckMutationContextParams` |
| 30 | Go-level generation bound (hand-built 257 refuses stale) | `TestCheckMutationContextGenerationBound` |

### Operation and status bodies (items 2, 5) — `ParseOperationBody`, `ParseStatusBody`

| # | Clause | Committed test |
| --- | --- | --- |
| 31 | Three closed bodies admitted with parsed members | `TestParseOperationBodyAdmitsRows` |
| 32 | Body member arms (member sets, generations, proof, timeouts, evidence, key consistency) | `TestParseOperationBodyRefusals` |
| 33 | Body scope gate (7 refused operations) | `TestParseOperationBodyScopeGate` |
| 34 | Status exact-instance and session-scoped lookups admitted | `TestParseStatusBodyAdmitsBothScopes` |
| 35 | Status null-pair rule (exactly one null refuses) | `TestParseStatusBodyNullPairRule` |
| 36 | Status member arms incl. generation 0/256/257 | `TestParseStatusBodyMemberRefusals` |

### Result identity (item 2) — `CheckResult`, `buildResult`

| # | Clause | Committed test |
| --- | --- | --- |
| 37 | Engine result repeats every identity, before equals source | `TestCheckResultAdmitsEngineResult` |
| 38 | Every repeated identity tamper refuses (generation → stale class) | `TestCheckResultRepetitionTamper` (8 subtests) |
| 39 | Generation 256 admitted in results | `TestCheckResultGenerationBound256` |
| 40 | Shape arms (states, effects, evidence, disposition, order/unique halves) | `TestCheckResultShapeArms` |
| 41 | Execution order vs sorted wire order split | `TestBuildResultSortsEffectsAndEvidence` |

### State-entry rules (item 3) — `Engine.ExecuteMutating`, `InterimState`, `ObserveFencing`

| # | Clause | Committed test |
| --- | --- | --- |
| 42 | creating entered only by create, after receipt, before first effect | `TestExecuteInterimState`, `TestExecuteReceiptPrecedesFirstEffect` |
| 43 | Pre-first-effect errors restore source (auth, generation, deadline, coded refusal, illegal source, conferral) | `TestExecutePreEffectErrorsRestoreSource` (5 subtests) |
| 44 | Post-commit uncertainty moves to unavailable with status_first, never absent | `TestExecutePostEffectErrorsMoveUnavailable` (4 subtests) |
| 45 | Only quiesce-input enters quiescing; no path yields creating or stale_fenced | `TestExecuteOnlyQuiesceEntersQuiescing` |
| 46 | stale_fenced only via fencing observation (epoch + lease-mismatch arms) | `TestObserveFencingStaleEpochFences`, `TestObserveFencingLeaseMismatchFences` |
| 47 | Non-stale observations leave state (authorized, remote, unverified, winnerless) | `TestObserveFencingNonStaleLeavesState`, `TestObserveFencingStaleParkVocabulary` |
| 48 | Auth rechecked before each effect (mid-op rotation caught post-commit) | `TestExecuteRechecksBeforeEachEffect/lease_rotates_mid-operation` |
| 49 | Generation rechecked before each effect (mid-op rotation caught post-commit) | `TestExecuteRechecksBeforeEachEffect/generation_rotates_mid-operation` |
| 50 | Pre-commit rotations restore source (lease + generation) | `TestExecuteRechecksBeforeEachEffect/rotation_before_first_effect_restores_source`, `/generation_rotation_before_first_effect_restores_source` |
| 51 | Deadline cancels waiting, never a committed effect (effect stands) | `TestExecuteDeadlineNeverCancelsCommittedEffect` |
| 52 | Per-row timeout vocabulary (create/quiesce terminal_backend_timeout, wait quiesce_timeout, stop stop_timeout) | `TestExecuteEntryDeadlineCodes` (4 subtests) |

### Generation-safe evidence (item 4)

| # | Clause | Committed test |
| --- | --- | --- |
| 53 | Bound in every owned surface (context, status, result bodies; 0/256/257; chars not bytes) | Rows 26, 36, 39 |
| 54 | Entry + per-effect staleness refuse the landed stale class | Rows 43 (generation subtests), 49, 50 |
| 55 | Bound verdict agrees with landed GenerationDigest on the boundary corpus | `TestGenerationBoundAgreesWithLandedDigest` |
| 56 | Landed row codes admitted by landed CheckErrorAllowed (in-set + outside-set) | `TestRowCodesAreAdmittedByCheckErrorAllowed` |

### Operation rows (item 5) — `Engine.ExecuteMutating`, `Engine.ExecuteStatus`

| # | Clause | Committed test |
| --- | --- | --- |
| 57 | status same-state from all 8 states + unknown source; zero effects and receipts | `TestExecuteStatusAllSources` |
| 58 | status adopts drifted state verbatim (no transition bookkeeping) | `TestExecuteStatusAdoptsReportedState` |
| 59 | status non-match adopts proven absent | `TestExecuteStatusNonMatchAdoptsAbsent` |
| 60 | status unknown never absent (coded, uncoded, outside-set code) | `TestExecuteStatusUnknownIsNeverAbsent` |
| 61 | status report validation (lying match, non-canonical form, unsorted evidence, matched-tuple false form, malformed reported members) | `TestExecuteStatusReportValidation`, `TestExecuteStatusMatchedTupleFalseFormRefuses`, `TestExecuteStatusReportMemberRefusals` |
| 62 | status provider conditional exactly when requested | `TestExecuteStatusProviderConditional` |
| 63 | status session-scoped lookup adopts the reported instance | `TestExecuteStatusSessionScoped` |
| 64 | quiesce from active and parked to quiescing with input_closed | `TestExecuteQuiesceSuccess` (2 subtests) |
| 65 | Row refusal sets exact (quiesce timeout in/stop_timeout out; wait quiesce_timeout in/backend timeout out; stop stop_timeout in/quiesce_timeout out, post-commit and first-effect directions) | `TestExecuteBackendRowErrorSets` |
| 66 | wait quiescing to quiescing with safe_boundary_observed | `TestExecuteWaitSuccess` |
| 67 | wait capabilities (safe always; provider iff provider kind; provider-only still needs safe) | `TestExecuteWaitCapabilityConditionals` (4 subtests) |
| 68 | stop quiescing to stopped, 3 effects in transition order, reported sorted | `TestExecuteStopSuccess` |
| 69 | create receipt rule (interactive to active, headless to parked, headless conditional) | `TestExecuteCreateReceiptRule` |
| 70 | Idempotent replay, changed-operation mismatch, uncertain retry refuses while the observation is unknown (resumption once status proves the source: rows 73–76), age-gated staging sweep | `TestExecuteIdenticalRetryReplays`, `TestExecuteChangedOperationInWindowMismatches`, `TestExecuteUncertainRetryRequiresStatus`, `TestStoreSweepKeepsLiveStaging`, `TestStoreSweepRemovesAgedStaging` |
| 71 | Real-SIGKILL crash seams (quiesce + create): pending survives, no completion, uncertain retry refuses before any status proof, status proves the source, same-key retry completes with one result and one receipt | `TestExecuteCrashChildSelfTerminates`, `TestExecuteCrashCreateResumesAfterStatus` |
| 72 | Landed conferral wiring (empty admitted, all 4 rows) and engine scope gate (6 refused ops) | `TestExecuteEmptyAdmittedRefusesAtLandedGate` (4 subtests), `TestExecuteScopeGate` |

### Rework (rev3) — recovery, remote fencing, row gates

| # | Clause | Committed test |
| --- | --- | --- |
| 73 | Same-key retry resumes when status proves the source (quiesce, stop-not-closed, create-absent): one effect set, one completion, one receipt, no second binding | `TestExecuteUncertainRetryResumesWhenStatusProvesSource`, `TestExecuteRequestStopResumesWhenNotClosed`, `TestExecuteCreateResumesWhenStatusProvesAbsent` |
| 74 | Resumption refuses when status proves the target, closed, a contradiction, or nothing (coded, uncoded, lying report) | `TestExecuteUncertainRetryTargetProvenRefuses` (3 subtests), `TestExecuteUncertainRetryUnknownObservationRefuses` (3 subtests) |
| 75 | Hook-error commit reconciles: standing receipt is uncertain, retry resumes on proven source, refuses on unknown | `TestExecuteHookErrorRetryReconcilesThroughStatus` (3 subtests) |
| 76 | Pre-link store failure restores the source; post-link and unreadable-store failures fence uncertain | `TestExecutePreLinkStoreFailureRestoresSource`, `TestExecuteStoreLookupFailureIsUncertain` |
| 77 | Remote-winner fencing: older epoch and losing lease fence from active/parked/quiescing; winning token and future epoch leave state with the remote park | `TestObserveFencingRemoteWinnerStaleEpochFences`, `TestObserveFencingRemoteWinnerLosingLeaseFences`, `TestObserveFencingRemoteWinnerWinningTokenLeavesState`, `TestObserveFencingRemoteWinnerFutureEpochLeavesState` |
| 78 | Fencing sources: creating fences; absent/stopped/stale_fenced/unavailable leave state (local and remote winners) | `TestObserveFencingNonLiveSourcesLeaveState` |
| 79 | Create entry gates: control/expired/foreign-lease auth, stale generation, headless-from-absent (each binds no receipt), committed second-effect timeout | `TestExecuteCreateEntryAuthorizationRefuses` (3 subtests), `TestExecuteCreateEntryGenerationRefuses`, `TestExecuteCreateHeadlessFromAbsentNeedsCapability`, `TestExecuteCreateCommittedTimeoutReportsStatusFirst` |
| 80 | Auth and generation rechecked before the third stop effect (rotations caught after two committed effects) | `TestExecuteRechecksBeforeThirdEffect` (2 subtests) |
| 81 | Reported session compared on both status paths; every other axis both ways (non-match admits, lying match refuses) | `TestExecuteStatusSessionDriftRefuses` (4 subtests), `TestExecuteStatusIdentityMatchEachAxis` (10 subtests) |
| 82 | Deadline instant refuses on both entries (all four mutating rows bind no receipt; status runs no observation) | `TestExecuteEntryDeadlineInstantRefuses` (4 subtests), `TestExecuteStatusDeadlineInstantRefuses` |
| 83 | Kind negatives attach and none refused (direct and nested entries) | `TestParseAuthorizationKindClosed`, `TestParseAXAuthorizationKindClosed` |
| 84 | Kind mapping delegates to the landed authorization column (all ten ops plus unknown, both sides) | `TestKindForAgreesWithLandedTable`, `TestTransitionAuthorizationReportsEveryTableColumn` (`terminalbackend`) |

### Rework (rev4) — grant-less fencing, expiry recheck, P3 arms

| # | Clause | Committed test |
| --- | --- | --- |
| 85 | Grant-less stale fences: every grant-precondition member (no grant, lapsed grant, no clock reading, unusable policy) fences from active/parked/quiescing under local and remote winners, for the older-epoch and lease-mismatch tokens (48 transitions; the direct gate refuses grant-gated, never a park) | `TestRV3F3_GrantLessStaleFences` |
| 86 | Decided-not-stale leaves state grant-less: winning token and future epoch surface the grant refusal with no transition, local and remote | `TestRV3F3_GrantLessDecidedNotStaleLeavesState` (4 subtests) |
| 87 | Undecidable observations leave state: unverified, ambiguous, no winner, malformed winner, session mismatch, failed handoff (park preserved), malformed session/epoch/lease — each with a second stale-shaped member — surface the gate's own refusal or park | `TestRV3F3_GrantLessUndecidedLeavesState` (9 subtests) |
| 88 | Live-source gate under the verdict path: creating fences grant-less; absent/stopped/stale_fenced/unavailable surface the grant refusal (local and remote) | `TestRV3F3_GrantLessNonLiveSourcesLeaveState`, `TestRV3_FencingNoLocalTupleComparison` (composition pin) |
| 89 | Auth expiry rechecked before each effect: mid-operation expiry caught after one committed effect (`ax authorization expiry`, after unavailable, new_authorization); the E1 expired members kill on the receipt via the late-deadline instant | `TestRV3W_M1_AuthExpiryRecheckedBeforeEachEffect`, `TestExecuteCreateEntryAuthorizationRefuses/expired`, `TestExecutePreEffectErrorsRestoreSource/entry_auth_expired` |
| 90 | Evidence bound edge 256 admitted / 257 refused on the result and status surfaces | `TestRV3W_M2_EvidenceBoundEdge` |
| 91 | Capability-conditional refusal carries required_operator_action at the engine on every conditional | `TestRV3W_M3_ConditionalCapabilityDisposition` (3 subtests) |
| 92 | Proof-kind sibling names refused at the direct entry and through the wait body; sibling vocabularies in all three enum corpora | `TestRV3W_M4_ProofKindSiblingNamesRefused`, `TestParseProviderProofKindClosed`, `TestParseRetryDispositionClosed`, `TestParseAuthorizationKindClosed`, `TestParseAXAuthorizationKindClosed` |
| 93 | Status last_operation_id grammar: empty, garbage and upper-case UUID refuse, never adopted | `TestRV3W_M8_StatusLastOperationGrammar` |
| 94 | Authorization expiry rechecked before the THIRD effect (tuple unchanged, deadline ahead) | `TestRV4W_ExpiryRecheckedBeforeThirdEffect` |
| 95 | Replayed image generation binding refused at the ENGINE (no stale image handed back) | `TestRV4W_ReplayedImageGenerationBindingAtEngine` |
| 96 | Corrupt completion image is an integrity failure at the engine replay site | `TestRV4_CorruptCompletionImageIsIntegrityFailure` |
| 97 | In-loop deadline instant cancels waiting with exactly one committed effect | `TestRV4W_LoopDeadlineInstantCancelsWaiting` |
| 98 | Mixed-case and whitespace enum siblings refused on all three enums (direct + document) | `TestRV4W_AuthorizationKindCaseSiblingRefused` |
| 99 | Mixed-case siblings in the three enum corpora | `TestParseAuthorizationKindClosed`, `TestParseRetryDispositionClosed`, `TestParseProviderProofKindClosed` |

## Mutant table

`mutant_harness.py`: 122 rows, two full passes at 116 plus a full
pass at 122, 120 narrowing KILLED + 1 tightening KILLED + the
SURVIVED control, per-plant raw logs with subprocess exits all
passes, production blobs byte-identical before/after. Every row is narrowing except `N-auth-epoch-high`
(tightening: the weakening direction is equivalent because `environ`
refuses 2^53 at the number layer, so no weakening past `MaxLeaseEpoch`
is observable; the row proves the edge is exactly 9007199254740991 by
refusing it) and `C-control` (harmless SURVIVED control). The eleven
`N-fencing-verdict-*` rows mutate the landed verdict and are killed
through `ObserveFencing`, so the composition — not the helper — is
measured; `N-fencing-verdict-composition` measures the `stale` half of
the `decided && stale` conjunction (dropping `decided` would be
equivalent, since the verdict never reports stale undecided). Retired
in rev4: `N-fencing-epoch-only` — a stale_owner park implies a
decided-stale verdict over the same observation (the verdict's
preconditions are a subset of the gate's pre-grant arms; pinned by
`TestStalenessAgreesWithGateOnHotPath`), and both arms call `fenceLive`
with the same refusal, so weakening the park arm is behaviorally
equivalent and unkillable; the mismatch fact stays measured by
`N-fencing-verdict-winner` and the lease-mismatch members of
`TestRV3F3_GrantLessStaleFences`. `N-fencing-operation` is re-keyed to
the non-live killer (the launch-class requirement now shows on the
surfaced park vocabulary, since live sources fence through the
fallback either way).

| Row | Gate | Killer |
| --- | --- | --- |
| N-auth-epoch-low | Epoch bound min edge (admits 0) | `TestParseAXAuthorizationEpochBound` |
| N-auth-epoch-high | Epoch bound max edge (refuses Max; tightening, see above) | `TestParseAXAuthorizationEpochBound` |
| N-auth-kind | Kind vocabulary (admits admin) | `TestParseAXAuthorizationKindClosed` |
| N-auth-kind-binding | Kind binding (control as create) | `TestCheckAuthorizationRefusals/kind_mismatch` |
| N-auth-expiry | Expiry edge (admits the instant) | `TestCheckAuthorizationRefusals/expiry_edge_refuses` |
| N-auth-lease-id | Lease identity comparison | `TestCheckAuthorizationRefusals/lease_identity_drift` |
| N-auth-lease-epoch | Lease epoch comparison | `TestCheckAuthorizationRefusals/lease_epoch_drift` |
| N-auth-extra-member | Closed member set | `TestParseAXAuthorizationClosedShape` |
| N-auth-expiry-order | Strict expiry order | `TestParseAXAuthorizationStrictExpiry` |
| N-disp-parse | Disposition vocabulary (admits retry) | `TestParseRetryDispositionClosed` |
| N-disp-unauthorized | Mapping: committed unauthorized | `TestDispositionForMapping` |
| N-disp-timeout | Mapping: uncommitted quiesce_timeout | `TestDispositionForMapping` |
| N-disp-stop | Mapping: stop_timeout group membership | `TestDispositionForMapping` |
| N-disp-default | Mapping: uncommitted capability default | `TestDispositionForMapping` |
| N-proof-vocab | Proof vocabulary (admits provider_restart) | `TestParseProviderProofKindClosed` |
| N-proof-provider | Provider classification (AX boundary) | `TestRequiresProviderObservation` |
| N-context-key | Key material equality (same-length wrong keys) | `TestCheckMutationContextKeyMaterial` |
| N-context-generation | Go-level generation bound (admits 257) | `TestCheckMutationContextGenerationBound` |
| N-wait-timeout-bound | Timeout bound (admits 3600001) | `TestCheckMutationContextParams` |
| N-body-key | Body key consistency (quiesce only) | `TestParseOperationBodyRefusals` |
| N-result-operation | Repetition: operation (admits tampered ID) | `TestCheckResultRepetitionTamper/operation` |
| N-result-session | Repetition: session | `TestCheckResultRepetitionTamper/session` |
| N-result-instance | Repetition: instance | `TestCheckResultRepetitionTamper/instance` |
| N-result-backend | Repetition: backend | `TestCheckResultRepetitionTamper/backend` |
| N-result-implementation | Repetition: implementation | `TestCheckResultRepetitionTamper/implementation` |
| N-result-protocol | Repetition: protocol | `TestCheckResultRepetitionTamper/protocol` |
| N-result-generation | Repetition: generation | `TestCheckResultRepetitionTamper/generation` |
| N-result-before | Repetition: before state | `TestCheckResultRepetitionTamper/before` |
| N-result-effects-order | Effects order half | `TestCheckResultShapeArms` |
| N-result-evidence-unique | Evidence uniqueness half | `TestCheckResultShapeArms` |
| N-store-mismatch-op | Mismatch: operation half | `TestStoreBindReplayMismatch` |
| N-store-mismatch-id | Mismatch: operation-ID half | `TestStoreBindReplayMismatch` |
| N-store-replay | Replay signal (fixture key) | `TestExecuteIdenticalRetryReplays` |
| N-store-torn | Torn receipt (4-byte file as absence) | `TestStoreLookupAbsenceIsNotFailure` |
| N-complete-unbound | Completion without receipt | `TestStoreCompleteRequiresReceipt` |
| N-fencing-remote | Fencing: relative question skipped, remote parks fenced | `TestObserveFencingRemoteWinnerWinningTokenLeavesState` |
| N-fencing-operation-remote | Fencing: relative question through mutation refuses stale | `TestObserveFencingRemoteWinnerStaleEpochFences` |
| N-fencing-remote-stale-only | Fencing: relative verdict fences any park | `TestObserveFencingRemoteWinnerFutureEpochLeavesState` |
| N-fencing-source-absent | Fencing: absent fenced | `TestObserveFencingNonLiveSourcesLeaveState` |
| N-fencing-source-stopped | Fencing: stopped fenced | `TestObserveFencingNonLiveSourcesLeaveState` |
| N-fencing-source-creating | Fencing: creating no longer fenced | `TestObserveFencingNonLiveSourcesLeaveState` |
| N-engine-checkoperation | CheckOperation wiring (wait only) | `TestExecuteEmptyAdmittedRefusesAtLandedGate/wait-safe-boundary` |
| N-engine-transition | CheckTransition wiring (stopped sources) | `TestExecutePreEffectErrorsRestoreSource/illegal_source_refuses` |
| N-engine-deadline-pre | Entry deadline (wait only; receipt bound) | `TestExecuteEntryDeadlineCodes/wait-safe-boundary` |
| N-engine-timeout-code | Row timeout code (wait reports stop code) | `TestExecuteEntryDeadlineCodes/wait-safe-boundary` |
| N-engine-auth-entry | Pre-bind auth gate (create exempt) | `TestExecuteCreateEntryAuthorizationRefuses` |
| N-engine-auth-entry-quiesce | Pre-bind auth gate (fixture op) | `TestExecutePreEffectErrorsRestoreSource/entry_auth_expired` |
| N-engine-generation-entry | Pre-bind generation gate (create exempt) | `TestExecuteCreateEntryGenerationRefuses` |
| N-engine-generation-entry-quiesce | Pre-bind generation gate (fixture op) | `TestExecutePreEffectErrorsRestoreSource/entry_generation_stale` |
| N-engine-headless-absent | Headless conditional skipped from absent | `TestExecuteCreateHeadlessFromAbsentNeedsCapability` |
| N-engine-committed-timeout-disposition | Committed timeout maps replay_same | `TestExecuteCreateCommittedTimeoutReportsStatusFirst` |
| N-engine-recheck-auth-third | Per-effect auth recheck (pre-third skipped) | `TestExecuteRechecksBeforeThirdEffect/lease_rotates_before_third_effect` |
| N-engine-recheck-generation-third | Per-effect generation recheck (pre-third skipped) | `TestExecuteRechecksBeforeThirdEffect/generation_rotates_before_third_effect` |
| N-engine-bind-prelink | Pre-link restore skipped for quiesce | `TestExecutePreLinkStoreFailureRestoresSource` |
| N-engine-bind-postlink | Standing receipt restores source | `TestExecuteHookErrorRetryReconcilesThroughStatus/standing_receipt_is_uncertain` |
| N-resume-status-error | Resumption on timed-out reconciliation read | `TestExecuteUncertainRetryUnknownObservationRefuses/coded_read_failure` |
| N-resume-contradiction | Resumption on parked contradiction | `TestExecuteUncertainRetryTargetProvenRefuses/contradictory_observation` |
| N-resume-target | Resumption on proven target | `TestExecuteUncertainRetryTargetProvenRefuses/quiesce_target_proven` |
| N-resume-scope-instance | Reconciliation through session-scoped lookup | `TestExecuteUncertainRetryUnknownObservationRefuses/lying_match_on_drifted_instance` |
| N-resume-complete-key | Completion under a second key | `TestExecuteIdenticalRetryReplays` |
| N-resume-generation-value | Reconciliation with empty generation | `TestExecuteUncertainRetryResumesWhenStatusProvesSource` |
| N-status-session-scoped | Session comparison skipped (session-scoped) | `TestExecuteStatusSessionDriftRefuses/session_scoped_lying_match_refuses` |
| N-status-session-exact | Session comparison skipped (exact) | `TestExecuteStatusSessionDriftRefuses/exact_lying_match_refuses` |
| N-status-match-instance | Match conjunction loses instance | `TestExecuteStatusIdentityMatchEachAxis/instance/lying_match_refuses` |
| N-status-match-backend | Match conjunction loses backend | `TestExecuteStatusIdentityMatchEachAxis/backend/lying_match_refuses` |
| N-status-match-implementation | Match conjunction loses implementation | `TestExecuteStatusIdentityMatchEachAxis/implementation/lying_match_refuses` |
| N-status-match-protocol | Match conjunction loses protocol | `TestExecuteStatusIdentityMatchEachAxis/protocol/lying_match_refuses` |
| N-status-match-generation | Match conjunction loses generation | `TestExecuteStatusIdentityMatchEachAxis/generation/lying_match_refuses` |
| N-engine-deadline-instant | Entry deadline admits the instant | `TestExecuteEntryDeadlineInstantRefuses` |
| N-status-deadline-instant | Status deadline admits the instant | `TestExecuteStatusDeadlineInstantRefuses` |
| N-auth-kind-attach | Kind vocabulary admits attach | `TestParseAuthorizationKindClosed` |
| N-auth-kind-none | Kind vocabulary admits none | `TestParseAuthorizationKindClosed` |
| N-engine-recheck-auth | Per-effect auth recheck (pre-first skipped) | `TestExecuteRechecksBeforeEachEffect/rotation_before_first_effect_restores_source` |
| N-engine-recheck-generation | Per-effect generation recheck (pre-first skipped) | `TestExecuteRechecksBeforeEachEffect/generation_rotation_before_first_effect_restores_source` |
| N-engine-precommit-after | Pre-commit after-state (timeouts to unavailable) | `TestExecutePreEffectErrorsRestoreSource/coded_backend_refusal_on_first_effect` |
| N-engine-postcommit-after | Post-commit after-state (process failures to source) | `TestExecutePostEffectErrorsMoveUnavailable/coded_refusal_on_second_effect` |
| N-engine-status-match | Status match equality (absent-state lies) | `TestExecuteStatusMatchedTupleFalseFormRefuses` |
| N-engine-status-unknown | Status unknown (uncoded reported absent) | `TestExecuteStatusUnknownIsNeverAbsent` |
| N-engine-status-conditional | Status provider conditional (fixture backend) | `TestExecuteStatusProviderConditional` |
| N-status-nullpair | Status null-pair rule (one half) | `TestParseStatusBodyNullPairRule` |
| N-interim | Interim creating (quiesce too) | `TestExecuteInterimState` |
| N-engine-kind-map | Kind mapping (landed control honored as create) | `TestExecuteQuiesceSuccess` |
| N-fencing-operation | Fencing op must be launch-class (non-live park vocabulary) | `TestObserveFencingNonLiveSourcesLeaveState` |
| N-engine-timeout-code-stop | Row timeout code (stop reports quiesce code) | `TestExecuteEntryDeadlineCodes/request-stop` |
| N-engine-timeout-code-default | Row timeout code (default reports stop code) | `TestExecuteEntryDeadlineCodes/quiesce-input` |
| N-engine-status-source | Status source matrix (garbage admitted) | `TestExecuteStatusAllSources` |
| N-params-bootstrap | Params bootstrap (admits nope) | `TestCheckMutationContextParams` |
| N-params-quiescence-quiesce | Params quiescence quiesce-arm (admits nope) | `TestCheckMutationContextParams` |
| N-params-quiescence-wait | Params quiescence wait-arm (admits nope) | `TestCheckMutationContextParams` |
| N-params-proof | Params proof (admits provider_restart) | `TestCheckMutationContextParams` |
| N-params-evidence | Params stop evidence (admits not-a-digest) | `TestCheckMutationContextParams` |
| N-params-graceful | Params graceful bound (admits 3600001) | `TestCheckMutationContextParams` |
| N-version-delegation | Version-tuple delegation (fixture backend skips) | `TestParseMutationContextMemberRefusals` |
| N-body-scope | Body scope map (admits create) | `TestParseOperationBodyScopeGate` |
| N-body-members | Body member set (admits one extra) | `TestParseOperationBodyRefusals` |
| N-store-sweep-age | Sweep age gate (sweeps live-suffixed young staging) | `TestStoreSweepKeepsLiveStaging` |
| N-result-effects-sort | Effects wire order (inverted) | `TestBuildResultSortsEffectsAndEvidence` |
| N-result-evidence-sort | Evidence wire order (inverted) | `TestBuildResultSortsEffectsAndEvidence` |
| N-fencing-verdict-future | Verdict epoch arm (future admitted as stale) | `TestRV3F3_GrantLessDecidedNotStaleLeavesState/future_epoch` |
| N-fencing-verdict-winner | Verdict lease arm (winning lease admitted as stale) | `TestRV3F3_GrantLessDecidedNotStaleLeavesState/winning_token` |
| N-fencing-verdict-unverified | Verdict verified arm (epoch-1 unverified decided) | `TestRV3F3_GrantLessUndecidedLeavesState/unverified` |
| N-fencing-verdict-ambiguous | Verdict ambiguity arm (epoch-1 ambiguous decided) | `TestRV3F3_GrantLessUndecidedLeavesState/ambiguous` |
| N-fencing-verdict-nowinner | Verdict winner arm (epoch-1 winnerless decided) | `TestRV3F3_GrantLessUndecidedLeavesState/no_winner` |
| N-fencing-verdict-winner-shape | Verdict winner-shape arm (epoch-1 malformed winner decided) | `TestRV3F3_GrantLessUndecidedLeavesState/malformed_winner` |
| N-fencing-verdict-session | Verdict session arm (epoch-1 mismatch decided) | `TestRV3F3_GrantLessUndecidedLeavesState/session_mismatch` |
| N-fencing-verdict-handoff | Verdict handoff arm (epoch-1 failed handoff decided) | `TestRV3F3_GrantLessUndecidedLeavesState/failed_handoff` |
| N-fencing-verdict-presented-session | Verdict session-grammar arm (agreed-malformed decided) | `TestRV3F3_GrantLessUndecidedLeavesState/malformed_session` |
| N-fencing-verdict-presented-epoch | Verdict epoch-zero arm (losing-lease zero decided) | `TestRV3F3_GrantLessUndecidedLeavesState/epoch_zero` |
| N-fencing-verdict-presented-lease | Verdict lease-grammar arm (epoch-1 malformed lease decided) | `TestRV3F3_GrantLessUndecidedLeavesState/malformed_lease` |
| N-fencing-verdict-composition | Composition fences on decided alone | `TestRV3F3_GrantLessDecidedNotStaleLeavesState` |
| N-engine-recheck-auth-expiry | Per-effect expiry recheck dropped (tuple stays) | `TestRV3W_M1_AuthExpiryRecheckedBeforeEachEffect` |
| N-result-evidence-bound | Evidence bound admits 257 (both surfaces) | `TestRV3W_M2_EvidenceBoundEdge` |
| N-engine-conditional-disposition | Conditional arm reports status_first | `TestRV3W_M3_ConditionalCapabilityDisposition` |
| N-proof-sibling | Proof vocabulary admits the capability name | `TestRV3W_M4_ProofKindSiblingNamesRefused` |
| N-status-last-operation-empty | Status admits empty last_operation_id | `TestRV3W_M8_StatusLastOperationGrammar` |
| N-engine-recheck-auth-expiry-third | Expiry frozen at entry for the third effect only (reviewer RV4-M1 mirror) | `TestRV4W_ExpiryRecheckedBeforeThirdEffect` |
| N-resume-checkresult-stale | Engine replay tolerates the stale generation (reviewer RV4-M6 mirror) | `TestRV4W_ReplayedImageGenerationBindingAtEngine` |
| N-resume-checkresult-disposition | Refused replay image reports replay_same (reviewer RV4-M6b mirror) | `TestRV4W_ReplayedImageGenerationBindingAtEngine` (disposition assertion) |
| N-resume-corrupt-image | Integrity arm admits brace-led corrupt bytes | `TestRV4_CorruptCompletionImageIsIntegrityFailure` |
| N-engine-loop-deadline-instant | In-loop check admits the deadline instant (reviewer RV4-M7 mirror) | `TestRV4W_LoopDeadlineInstantCancelsWaiting` |
| N-auth-kind-case | Kind vocabulary admits Control (reviewer RV4-M5 mirror) | `TestRV4W_AuthorizationKindCaseSiblingRefused`, `TestParseAuthorizationKindClosed` |
| C-control | Comment-only control (SURVIVED) | `TestRequiresProviderObservation` |

## Stated bounds

- No `ax` command, no doctor result, no runtime capability claim: the
  `Backend` interface is modeled; tmux/ConPTY/process control and the
  provider process do not exist here.
- Terminal Instance Binding 1.0.0 closed-object parsing: no parser
  exists on trunk (`canonicaljson` explicitly rejects
  `urn:ax:schema:terminal-instance-binding`); the generation bound is
  enforced on the surfaces this leaf owns and cited landed for the
  Provider Protocol 3 descriptor (`terminalbackend` checkGeneration
  arms). Full Binding parsing belongs with the provisioning surface.
- CLI Result 4 summary: no landed Result 4 body carries a raw
  generation member (`internal/cliresult` carries none by grep), so
  there is no surface to witness the bound on; consistent with §4.E
  (generation is host-local, non-replicable).
- Attach, restore, terminate-stale, manifest, probe: refused by the
  engine and body scope gates. Restore/terminate-stale recovery and
  terminal binding events belong to the story's final leaf.
- Create is receipt-scoped: receipt, transition, target and headless
  rules execute; the full create body (Binding, Entrypoint,
  presentation transport) belongs to provisioning.
- Credential-realm conditional: credential need is not modeled on the
  engine path.
- Protocol_version membership in the admitted probe list is enforced at
  registry admission (landed), not re-derived per operation; the
  operation gate enforces grammar plus major 1 through the landed
  `CheckVersionTuple` arms.
- The per-code disposition mapping is the leaf's contract where the
  spec pins the vocabulary but not the mapping; every arm is pinned
  literally and mutated. In particular a lease rotation after a
  committed effect reports `after=unavailable` with
  `new_authorization` (the lease names its own remedy even though the
  caller likely needs a status read too); §4.C's only sentence about
  `unavailable` pairs it with `status_first`, so this pairing is the
  leaf's stated contract, recorded as a literal tension, not a defect.
- `holder_host_id` is parsed (UUIDv7) and never bound:
  `CheckAuthorization` compares the §5.3 winning (lease_id, epoch)
  tuple only, so a foreign holder under the winning tuple is
  admitted. No path mints authority from the holder member.
- Resumption re-issues effects under the SAME idempotency key and
  relies on the backend's own key binding for safety (§4.C: "AX and
  backend durably bind the pair before the first child side effect";
  §13.13 safe_retry): the engine never issues one logical operation
  under a second key, so no second process is allocated. For
  wait-safe-boundary the source is the target, so a proven quiescing
  resumes and the resumed observation supersedes any lost one; at
  most one completion exists per key either way (no-replace).
- A same-key retry that reconciles to the proven TARGET (or closed)
  refuses uncertain rather than replaying: the effects may have
  committed but their evidence cannot be proven, and re-issuing from
  a non-source observation would violate the transition matrix. The
  row recovery columns ("then same-key retry only if not closed")
  are implemented as proceed-iff-source-proven. The interim-proven
  member of the create recovery (P3-5): a receipt without completion
  after `binding_persisted` committed, with status reporting the
  interim `creating` and identity match, refuses uncertain and the
  key stays parked — the partial progress is reachable only through
  the `unavailable → terminate-stale` path of the story's final
  leaf. The absent-proven member instead resumes: the retry
  re-performs both effects under the same receipt with one
  completion and one receipt (no second binding).
- AX-side seam emissions are outside the §4.C row sets: the row sets
  govern backend reports (gated through the landed
  `CheckErrorAllowed`), while an uncoded effect outcome classifies
  `terminal_backend_process_failed` and an uncertain retry refuses
  `terminal_backend_unavailable`, both with after `unavailable` and
  `status_first` per the row recovery columns. Engine deadline
  cancels before any attempt report `replay_same` on every row
  (nothing was attempted); a backend-reported `stop_timeout` maps
  through `DispositionFor` (`status_first` even uncommitted) because
  the graceful wait ran. The two stop directions are pinned together
  (`TestExecuteEntryDeadlineCodes`,
  `TestExecuteBackendRowErrorSets`).
- The wait/stop "lesser of request deadline and timeout" bound is
  enforced by the backend wait (modeled here), which reports a coded
  `quiesce_timeout`/`stop_timeout` proving non-commit; the engine
  validates the timeout members (`uint53[1..3600000]`, carried in
  `Params`) and enforces the `MutationContext` deadline.
- Document member arms delegate their grammars to the landed
  `environ`/`scalar` owners and are pinned literally through the
  parse entries; where a Go-level twin re-validates with the same
  detail, the Go twin carries the narrowing row (a document-side
  weakening alone is masked by the twin). The body scope gate and
  the body member-set gate have no twin and carry their own rows
  (`N-body-scope`, `N-body-members`). The engine scope gate is
  fail-fast ordering: admitting an operation past it is masked by
  the `checkParams`/`KeySegments` scope defaults firing the same
  detail (defense in depth, stated); `kindFor` is not redundant and
  carries `N-engine-kind-map`.
- The fencing operation must be launch-class (`restore`, equivalent
  to `activation` on the stale arms and chosen for the terminal
  domain); the non-launch entries refuse stale tokens instead of
  parking them, measured by `N-fencing-operation` on the direct
  question and `N-fencing-operation-remote` on the relative question.
  The relative question re-authorizes from the winner's own host; it
  composes the landed gate (no local tuple comparison) and fences
  only on the relative stale verdict from a live source. Grant-gated
  refusals (no grant, no clock reading, lapsed grant, unusable
  policy — the complete grant-precondition class, since these are all
  four refusals `Authorize` can surface after the ownership arms
  pass) consult the landed `StaleRelativeToWinner` verdict instead:
  decided-stale fences from a live source, decided-not-stale and
  undecidable surface the refusal. The relative question needs no
  fallback (it carries the direct grant facts unchanged and the grant
  arms do not read `LocalHostID`), and the failed-handoff park is
  preserved (ownership history, not a grant precondition).
- `internal/traceability` untouched per the Story contract; the final
  leaf carries the registry bindings and the re-pin.

## Inherited advisories (predecessor P3, for the record)

- P3-6 closed by this leaf: the `Decision.Descriptor` doc in
  `internal/axpane/decide.go` now states the recorded Binding
  terminal instance ID is authoritative on reattach; this package's
  `doc.go` states the consumer side.
- P3-2 bound stated: this leaf adds no I/O-free `Decide` caller (the
  engine never calls `Decide`), so the missing-closure input shape is
  unreachable from this package; the pure-core hardening stays with
  the owner.
- P3-3 bound stated: this leaf never loads checkpoints (no
  `LoadCheckpoint` caller), so the absent-store shape is unreachable
  from this package.
- P3-1, P3-4, P3-5 left to the final leaf (TASK-260830-2056mm):
  journal-only restore closure (recovery domain), `EmitParked`
  winner binding (events domain) and the corrupt-blob negative
  (recovery domain) belong to events-and-recovery work.

## Story close (final leaf TASK-260830-2056mm)

- The predecessor P3-1, P3-4, P3-5 above are closed in
  `internal/axpane/leafclose_test.go` with the production
  `EmitParked` winner binding and the `N-parked-winner`,
  `N-journal-closure`, `N-ckpt-corrupt` harness rows (see the
  axpane story-close section).
- This package's rev4 residual P3-1 through P3-4 are closed by
  rows 94–99: the five reviewer witnesses committed verbatim in
  `leafclose_test.go`, the mixed-case siblings added to the three
  enum corpora, and the six mirrored narrowing rows
  (`N-engine-recheck-auth-expiry-third`,
  `N-resume-checkresult-stale`,
  `N-resume-checkresult-disposition`, `N-resume-corrupt-image`,
  `N-engine-loop-deadline-instant`, `N-auth-kind-case`).