# Conformance matrix — TASK-260830-1c28dz (implement-tmux-lifecycle-operations)

DoD verdict: all acceptance criteria driven (rows 59-92 of
`internal/tmuxserver/TRACEABILITY.md`, 34 of 34 new rows green,
plus termbind row 67 for the reboot-successor mint), zero
real-tmux witnesses by the determinism design (bound B1,
carried), 218 second-leaf narrowing mutants KILLED net plus the
shared harmless control SURVIVED (298/298 tmuxserver rows and
43/43 termbind rows as expected), gate x entry census 267
measured of 5536 with 191 stated bounds and 160 named undriven
gaps, after-restore outcome grid (Table E, 22 outcomes) green,
validation suite green firsthand (see the evidence tarball).
No capability or CLI claims: this leaf ships library behavior
only, exercised through `go test` as below.

Scope: §4.2 production adapters (exec, dial, probe, spawn), the
B18/B19/B20 socket bind step, and the §4.C lifecycle operations
create, attach, status, quiesce-input, wait-safe-boundary,
request-stop, terminate-stale, restore with receipt discipline,
tamper evidence, the AX-side state memory, the termbind
reboot-successor mint, and the after-restore wrapper
composition (refresh-before-decide with lapsed-grant remote
offer). Landed code is composed, never forked: terminstance
engine/receipts/results, terminalbackend
vocabulary/transitions/conformance, termbind and axpane stores,
fencing gates, sessrepo summaries, scalar and environ grammars,
secprim argv guard, and the first-leaf custody and readiness
gates are called as-is; the only touched landed files are the
error-code registry (new codes), the ownership seam (test
hook), and the mutant harnesses (new rows).

## AC rows (34 of 34; mirrored from TRACEABILITY rows 59-92)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 59 | The exec adapter runs -S vectors with AX-owned argv only: exits and both streams report as data, TMUX/TMUX_TMPDIR scrub from the child, malformed argv refuses, transport failures and signal deaths pass through as errors and prove nothing, and NUL-containing arguments refuse before any process starts. | OSRunner (exec.go) | TestOSRunnerReportsExitAsData, TestOSRunnerCapturesStderr, TestOSRunnerSignalDeathProvesNothing, TestOSRunnerScrubsAmbientInChild, TestOSRunnerRefusesBadArgv, TestOSRunnerRefusesNulArgument, TestOSRunnerTransportErrorProvesNothing, TestScrubTmuxEnv |
| 60 | The primitive vocabulary is exactly the ten tmux directives and the per-operation map covers exactly the eight lifecycle operations; unknown primitives and non-lifecycle operations refuse. | ParseDirective, DirectivesFor | TestParseDirectiveAdmitsTenPrimitives, TestParseDirectiveRefusesUnknown, TestDirectivesForMapsEightOperations, TestDirectivesForRefusesNonLifecycle |
| 61 | Every built vector is the fixed -S shape for its directive: the socket is pinned to the runtime leaf (ambient sockets refuse), targets parse as UUIDv7, the quiescence channel parses as UUIDv7, the attach vector states its input authorization (read-only emits -r, unstated refuses), the boundary vector is the bare blocking wait, and detach-client addresses the session. | BuildCommand | TestBuildCommandVectors, TestBuildCommandAttachReadOnlyVector, TestBuildCommandAttachRequiresInputFlag, TestBuildCommandRefusesAmbientSocket, TestBuildCommandRefusesBadIdentity |
| 62 | Dialing the socket reports live, stale, or unknown: refused and missing sockets are stale, a missing directory and every other failure is unknown, never absent. | UnixDialer.Dial | TestUnixDialerLiveStaleUnknown, TestUnixDialerMissingSocketIsStale, TestUnixDialerMissingDirectoryIsUnknown |
| 63 | The prober enforces dependencies, length, and custody before dialing: stale reports not-running, unknown errors, live attests through the admission, and admission failures pass through. A writable root refuses at Probe like at the spawner. | ServerProber.Probe | TestServerProberRefusesWithoutDependencies, TestServerProberMapsDialOutcomes, TestServerProberAdmissionFailurePassesThrough, TestServerProberEnforcesLengthAndCustody, TestServerProberRefusesWritableRoot |
| 64 | The broker prober reports the miss, admission reconciliation fails closed without a verifier, and the Production constructor wires the prober, spawner, dialer, and admission into the Dependencies. | probe.go | TestBrokerProberReportsMiss, TestReconcileAdmissionFailsClosedWithoutVerifier, TestProductionWiresDependencies |
| 65 | The spawner enforces length, custody, and the exec-site socket pin before spawning: a stale socket unlinks (hook-observed), a non-socket refuses, and exit/transport outcomes map to the spawn verdict, and a world-writable root refuses before any spawn vector runs. | ServerSpawner.Spawn | TestServerSpawnerRefusesWithoutRunner, TestServerSpawnerPinAndCustody, TestServerSpawnerUnlinksStaleSocket, TestServerSpawnerRefusesNonSocketFile, TestServerSpawnerMapsRunnerOutcome, TestServerSpawnerRefusesAbsentLeaf, TestServerSpawnerRefusesWritableRoot |
| 66 | B18: the socket path refuses at or beyond the platform sun_path limit (104/108), counted in bytes. | CheckSocketLength | TestCheckSocketLengthBounds, TestCheckSocketLengthCountsBytes |
| 67 | B19/B20: the socket verifies at exactly the runtime leaf, and the leaf custody delegates to the landed Verify with its refusal propagated. | CheckSocketCustody | TestCheckSocketCustodyAdmitsCompliant, TestCheckSocketCustodyRefusesMisplacedSocket, TestCheckSocketCustodyPropagatesLeafRefusal |
| 68 | B20: the runtime root verifies as a caller-owned directory unwritable by others through the landed no-follow open, with the open handle bound back to its path. | checkCustodyRoot | TestCheckSocketCustodyRefusesUnsafeRoot, TestCheckSocketCustodyRefusesForeignRoot, TestCheckSocketCustodyIdentityMismatchRefuses, TestCheckSocketCustodyNoCacheAfterChange |
| 69 | B20: every ancestor verifies as a non-symlink directory unwritable by others up the lexical chain; sticky directories admit, symlinks and files refuse through the no-follow opens. | checkCustodyAncestors | TestCheckSocketCustodyRefusesWritableAncestor, TestCheckSocketCustodyAdmitsStickyAncestor, TestCheckSocketAncestorKindArm, TestCheckSocketCustodyRefusesIntermediateSymlink, TestSocketCustodySymlinkRefusesAtProbeAndSpawn |
| 70 | B19: the socket path itself is bindable when absent and unlinkable when a socket; any other kind refuses so the stale-unlink step can never remove a non-socket. | checkCustodySocket | TestCheckSocketCustodyAdmitsCompliant, TestCheckSocketCustodyRefusesNonSocketFile |
| 71 | Create bodies are the closed six-member shape with the operation context, binding, bootstrap, entrypoint, transport, and interactive members. | parseCreateBody | TestParseCreateBodyValid, TestParseCreateBodyMemberSet, TestParseCreateBodyMembers |
| 72 | Attach bodies are the closed eleven-member shape: identity (session, instance, backend, versions, generation), client, transport, input flag, deadline, and the ownership-neutral authorization, which carries no lease member. | parseAttachBody | TestParseAttachBodyValid, TestParseAttachBodyMemberSet, TestParseAttachBodyMembers |
| 73 | Terminate bodies are the closed four-member shape with the stale lease, stale epoch, and evidence members. | parseTerminateBody | TestParseTerminateBodyValid, TestParseTerminateBodyViolations |
| 74 | Restore bodies are the closed four-member shape with the prior binding, checkpoint, and bootstrap members. | parseRestoreBody | TestParseRestoreBodyValid, TestParseRestoreBodyViolations |
| 75 | Execute admits all eight lifecycle operations through the dependency, vocabulary, scope, length, and custody pre-gates; each pre-gate refuses before dispatch on every operation, with per-operation custody twins for root, ancestor, socket-kind, and leaf-mode arms. | Lifecycle.Execute | TestExecuteRefusesMissingDependencies, TestExecuteRefusesNonLifecycleScope, TestExecuteRefusesLongSocket, TestExecuteRefusesUnsafeSocket, TestExecuteCustodyEntryRoots, TestExecuteCreateRefusesWritableRoot, TestExecuteAttachRefusesWritableRootValidBody, TestExecuteEachOperationRefusesWritableRoot, TestExecuteEachOperationRefusesWritableAncestor, TestExecuteEachOperationRefusesNonSocketKind, TestExecuteEachOperationRefusesBadLeafMode, TestExecuteCustodyNarrownessTwins |
| 76 | Every operation verifies the ax.tmux backend identity before dispatch; foreign backends refuse on all eight operations without exec. | checkLifecycleBackend | TestExecuteRefusesForeignBackend |
| 77 | Out-of-row error codes normalize to the protocol error; in-row codes propagate with their report, on all eight rows. | rowError | TestRowErrorNormalizesOutsideSet, TestRowErrorCoversEightRows, TestLifecycleEmittedCodesAreRowAllowed |
| 78 | Create runs the closed-shape admission, binding agreement, and relay refusal through the engine: interactive returns the bound descriptor, headless returns none, and identical requests replay one spawn. A fresh create rotates the incarnation, superseding prior closures; replays never rotate, and a failed rotation fails the operation closed. The wrapper is the path's only command: a rotation during its round trip changes nothing further (single-command probe). | executeCreate | TestExecuteCreateInteractive, TestExecuteCreateHeadless, TestExecuteCreateIdempotent, TestExecuteCreateRefusals, TestExecuteCreateBindingAgreement, TestExecuteCreateReplayPreservesBarrier, TestExecuteCreateIncarnationWriteFailureFailsClosed, TestEffectBoundaryAuditCreateIssuesSingleCommand |
| 79 | Attach returns the caller-executed vector with its descriptor and never execs an interactive attach: closed-shape admission, deadline, matching-transport capability, the quiescing input barrier, live attestation, the authoritative barrier recheck under the ordering lock, the post-lock generation and deadline rechecks, and the durable client receipt, with identical requests replaying; read-only authorization emits the read-only vector, a read-only attach observes without reopening quiesced input, and a lost state-record write fails closed, while a lost outcome-report write after committed closure is bound B43 (input may reopen before the retry heals the report). The barrier verdict is the incarnation-scoped closure proof alone: stale memory, caller-carried source state, and failure results are never consulted for their value, so none of them can reopen a closed input; only a fresh create or restore reopens, by rotating the incarnation. A corrupt scan, an empty time, a keyless report, or an unreadable state, incarnation, outcomes-directory, or outcome record errors. A second client that may overlap a recorded peer needs multi_attach, and new input over a recorded input peer needs multiple_input_clients, with receipt-validated same-client retries exempt (identical bodies replay with a peer present; conflicting retries still refuse idempotency_mismatch) and unreadable peers failing closed; a directory at a receipt-shaped name is corruption and refuses, a valid receipt misfiled under another client's durable key refuses through the keyed binding before any exclusion applies, and a malformed receipt name refuses, while staging and non-receipt entries still skip; client liveness is bound B44 (owned by TASK-260922-vcx6yo). The admission and barrier waits honor the operation deadline and cancellation with no late commit. | executeAttach | TestExecuteAttach, TestExecuteAttachRefusals, TestExecuteAttachIdempotent, TestAttachDescriptorNeverPersisted, TestExecuteAttachRefusesMeshWithoutCapability, TestExecuteAttachRefusesDeadline, TestExecuteAttachRefusesInputWhenQuiescing, TestExecuteReadOnlyAttachIsReadOnly, TestExecuteReadOnlyAttachPreservesQuiescedInput, TestExecuteQuiesceStateWriteFailureRefusesWiredAttach, TestExecuteAttachErrorsOnCorruptBarrierScan, TestExecuteAttachErrorsOnEmptyBarrierTime, TestReviewExistingActiveMemoryCannotReopenQuiescedInput, TestReviewFailedStopCannotEraseQuiescence, TestExecuteFailedStopPreservesQuiescingMemory, TestExecuteAdvisoryMemoryNeverDecidesAdmission, TestExecuteAttachErrorsOnEmptyBarrierKey, TestExecuteAttachErrorsOnStateReadFailure, TestAttachRefusesAfterQuiescenceWhenAdmittedLate, TestAttachQuiesceOrderingOverlapQuiesce, TestAttachQuiesceOrderingOverlapBoundary, TestExecuteAttachErrorsOnIncarnationReadFailure, TestExecuteAttachErrorsOnOutcomesDirReadFailure, TestExecuteAttachErrorsOnOutcomeFileReadFailure, TestAttachRefusesGenerationRotatedDuringLockWait, TestAttachRefusesDeadlineExpiredDuringAdmission, TestAttachRefusesDeadlineOnTheInstantDuringAdmission, TestAttachAuthorizationMembersRefuseWithoutReceipt, TestAttachOverlapRequiresMultiAttach, TestAttachOverlapInputRequiresMultipleInputClients, TestAttachSameClientRetryIgnoresOverlap, TestAttachOverlapPeerReadFailureFailsClosed, TestAttachPeerDirectoryFailsClosed, TestAttachSameClientRetryWithPeerPresent, TestAttachPeerFilenameMismatchFailsClosed, TestAttachWaitBoundedByOperationDeadline, TestAttachWaitCancelledReturnsPromptly |
| 80 | Status classifies present, absent, and unknown observations against the AX-side memory: absence needs its stderr marker (unmarked, refused, and killed probes read unavailable), contradictions read unavailable, malformations refuse, and the provider triple evidences only when requested, capable, and observed. Status observes the recorded binding tuple: drifted queries non-match member by member, unbound sessions refuse unknown, tampered documents refuse integrity, negative backend counts refuse, and nonzero attached exits, foreign attached rows, unmarked panes exits, and malformed counts with plausible output read unavailable. The live probes run under the query deadline in-flight: a probe still blocked at the bound concludes unknown with no report, never absent. | executeStatus, ObserveStatus | TestExecuteStatusPresent, TestExecuteStatusAbsent, TestExecuteStatusPresentIncludesIdentity, TestExecuteStatusUnknownProbeIsUnavailable, TestExecuteStatusProviderConditional, TestExecuteStatusContradictions, TestClassifyStatusAbsentProbe, TestClassifyStatusPresentProbe, TestClassifyAbsenceMarkers, TestObserveStatusPresentParked, TestObserveStatusAbsentProbe, TestObserveStatusMemoryReconciles, TestObserveStatusUnknowns, TestProviderTripleRule, TestCutRowShapes, TestExecuteStatusNonMatchEachMember, TestExecuteStatusExactUnboundSessionRefusesUnknown, TestExecuteStatusBindingTamperRefusesIntegrity, TestObserveStatusSessionScopedObservesRecordedTuple, TestObserveStatusExactBackendDriftNonMatches, TestObserveStatusExactProtocolDriftNonMatches, TestExecuteStatusRefusesNegativeAttachedCount, TestExecuteStatusAttachedExitIsUnknown, TestExecuteStatusForeignAttachedRowIsUnknown, TestExecuteStatusUnknownPanesExitIsUnknown, TestExecuteStatusMalformedAttachedCountIsUnknown, TestExecuteStatusProbeHonorsOperationDeadline |
| 81 | Quiesce, boundary, and stop run through the engine with their outcome members: quiesce locks then detaches with no provider input, the boundary waits blocking with the proof kind bound into its evidence and replays its recorded time, and stop proves closure only from marked absence; identical requests replay. Corrupt, forged-key, and empty-time outcome reports refuse integrity instead of becoming fresh evidence; a failed barrier-state write fails the operation instead of reporting closure, and the retry heals the barrier with the original time. The stop escalation revalidates deadline, authorization, and generation after its poll before the kill issues — a graceful timeout escalates, any other stale fact refuses — the quiesce detach revalidates between the commands, the boundary tail is read-only, and the barrier waits honor the deadline on all three paths. The stop poll itself honors the operation deadline in-flight: a probe still blocked at the bound concludes unknown with no kill and no closure, never absence. The post-escalation re-confirmation observes under the operation deadline, so a cancellation-honouring executor still observes the closure after the graceful wait expires. | executeEngineOp | TestExecuteQuiesce, TestExecuteQuiesceIdempotent, TestExecuteQuiesceBarrierSequence, TestExecuteQuiesceEmitsNoProviderInput, TestExecuteQuiesceRefusesFailedClose, TestExecuteQuiesceReplayKeepsTimestamps, TestExecuteBoundary, TestExecuteBoundaryIdempotent, TestExecuteBoundaryWaitsForSignal, TestExecuteBoundaryBindsProofKind, TestExecuteBoundaryWaitTimeout, TestExecuteBoundaryPastDeadline, TestExecuteBoundaryRefusesFailedWait, TestExecuteBoundaryReplayKeepsProofAndTime, TestExecuteStop, TestExecuteStopIdempotent, TestExecuteStopUnknownProbeIsNotClosure, TestExecuteStopUnmarkedExitIsNotClosure, TestExecuteStopPollHonorsOperationDeadline, TestBoundWaitTakesLesser, TestLesserDeadlineTakesLesser, TestLifecycleIgnoresAmbientEnvironment, TestObserveBoundaryBindsProviderRows, TestOutcomeRecordCrashConverges, TestExecuteQuiesceCorruptOutcomeRefusesIntegrity, TestExecuteBoundaryCorruptOutcomeRefusesIntegrity, TestExecuteQuiesceForgedOutcomeKeyRefusesIntegrity, TestExecuteQuiesceEmptyRecordedTimeRefusesIntegrity, TestExecuteBoundaryEmptyRecordedTimeRefusesIntegrity, TestExecuteBoundaryRecordCrashRefusesIntegrity, TestExecuteQuiesceStateWriteFailureRefusesWiredAttach, TestExecuteQuiesceRetryAfterStateWriteFailureHealsBarrier, TestExecuteBoundaryBarrierSurvivesStateWriteFailure, TestExecuteStopRefusesStaleFactsAfterPoll, TestExecuteStopEscalatesAfterGracefulTimeoutAlone, TestExecuteStopEscalationSucceedsWithContextHonoringRunner, TestExecuteQuiesceRefusesStaleFactsBetweenCommands, TestBackendPostWaitBoundaryRefusesWithoutRecheck, TestEffectBoundaryAuditBoundaryTailIsReadOnly, TestQuiesceBarrierWaitBoundedByOperationDeadline, TestBoundaryCommitWaitBoundedByOperationDeadline, TestQuiesceMalformedDeadlineRefusesBeforeBarrierWait |
| 82 | Terminate-stale runs the key, capability, transition, authorization, generation, fencing, target, winner, and deadline gates before the receipt-bound kill with its live-session confirm; an unproven confirm is unknown, never termination — including a confirm still blocked when the operation deadline fires, which concludes unknown in-flight with no termination evidence; the memory adopts stopped, identical requests replay, and effect failures resume through status. The re-confirm tail issues no command after its poll however the facts moved (no-post-wait-command probe). | executeTerminate | TestExecuteTerminate, TestExecuteTerminateRefusals, TestExecuteTerminateIdempotent, TestExecuteTerminateResumesAfterEffectFailure, TestExecuteTerminateUnknownConfirmIsNotTerminated, TestExecuteTerminateConfirmHonorsOperationDeadline, TestExecuteTerminateWinnerTracksCurrentLease, TestExecuteTerminateTargetLeaseIdentity, TestExecuteTerminateTargetSessionBinds, TestExecuteTerminateRefusesProviderOnlyCapability, TestExecuteTerminateRefusesOnDeadline, TestExecuteTerminateRedriveMismatch, TestExecuteTerminateRefusesAuthorizationExpiredAfterReceipt, TestExecuteEntryRefusesWrongAuthorizationKind, TestEffectBoundaryAuditTerminateIssuesNoPostWaitCommand |
| 83 | Restore runs the capability, prior-binding, and receipt-bound persist-plus-wrapper sequence, then returns the required binding and the parked wire result; the after-restore branch decision (local resume, remote offer, or parked under the enforced refresh bound) is the wrapper composition (row 92, Table E). A substituted prior refuses the mismatch, a swapped document refuses the integrity failure, the memory adopts parked, identical requests replay, and rechecks resume through status. A fresh restore rotates the incarnation, superseding prior closures; replays never rotate, and a failed rotation fails the operation closed. A reboot restore validates its prior before the effects commit, then mints a successor generation persisted after the effects, with successor retries converging. The wrapper is the path's only command (single-command probe). | executeRestore | TestExecuteRestore, TestExecuteRestoreMismatch, TestExecuteRestoreIdempotent, TestExecuteRestoreReturnsBinding, TestExecuteRestoreRefusesSwappedBindingDoc, TestExecuteCreatePersistsBindingDocForRestore, TestExecuteCreateRefusesBindingDocFailure, TestExecuteResumeRefusesWhenStatusProvesOtherwise, TestExecuteRestoreResumesAfterRecheckFailure, TestExecuteRestoreRefusesWithoutRebootCapability, TestExecuteRestoreRefusesOnDeadline, TestExecuteRestoreRefusesOnDeadlineEffect, TestExecuteRestoreRefusesGenerationDrift, TestExecuteRestoreAcrossServerGeneration, TestExecuteRestoreSuccessorRetryConverges, TestExecuteRestorePriorValidationPrecedesEffects, TestExecuteRestoreRefusesSubstitutedInstance, TestExecuteRestoreRefusesSubstitutedSession, TestExecuteRestoreRefusesSubstitutedBackend, TestExecuteQuiesceStopRestoreReopensBarrier, TestExecuteRestoreReplayPreservesBarrier, TestExecuteRestoreIncarnationWriteFailureFailsClosed, TestExecuteRestoreRefusesAuthorizationExpiredAfterReceipt, TestExecuteEntryRefusesWrongAuthorizationKind, TestEffectBoundaryAuditRestoreIssuesSingleCommand |
| 84 | The receipt discipline binds, replays, resumes, and completes under the idempotency key: tampered completions refuse member by member, corrupt images refuse, hook failures report uncertainty, and forged effect evidence fails the run pre-commit. | executeWithReceipt, resumeUncertain, runLifecycleEffects | TestExecuteTerminateReplayRefusesTamperedImage, TestExecuteTerminateReplayRefusesCorruptImage, TestExecuteTerminateBindHookFailure, TestRunLifecycleEffectsRefusesForgedEvidence |
| 85 | Deadlines are strict bounds at entry and before every effect: the on-deadline instant refuses, and the entry arm leaves the key unbound where the loop would bind it. The attach lock wait counts against the deadline: an operation that expires mid-wait refuses even while its authorization stays valid. The attach, quiesce, and boundary waits refuse at the deadline with no late commit, and cancellation returns promptly; the stop escalation answers to the operation deadline, distinct from the graceful wait. The read-only probes honor the same bound in-flight: the stop poll, the terminate confirm, and the status probes conclude unknown at the deadline, never a verdict. | ops.go, executeAttach | TestExecuteTerminateRefusesOnDeadline, TestExecuteRestoreRefusesOnDeadline, TestExecuteRestoreRefusesOnDeadlineEffect, TestExecuteAttachRefusesDeadline, TestAttachRefusesDeadlineExpiredDuringAdmission, TestAttachRefusesDeadlineOnTheInstantDuringAdmission, TestAttachWaitBoundedByOperationDeadline, TestAttachWaitCancelledReturnsPromptly, TestQuiesceBarrierWaitBoundedByOperationDeadline, TestBoundaryCommitWaitBoundedByOperationDeadline, TestExecuteStopRefusesStaleFactsAfterPoll, TestExecuteStopPollHonorsOperationDeadline, TestExecuteTerminateConfirmHonorsOperationDeadline, TestExecuteStatusProbeHonorsOperationDeadline |
| 86 | Generations validate at entry and recheck before every effect: drift between entry and effect refuses under the stale binding with nothing committed. The stop escalation and the quiesce detach revalidate generation after their waits before the next destructive command. | ops.go | TestExecuteEntryRefusesStaleGeneration, TestExecuteRestoreRefusesGenerationDrift, TestAttachRefusesGenerationRotatedDuringLockWait, TestExecuteStopRefusesStaleFactsAfterPoll, TestExecuteQuiesceRefusesStaleFactsBetweenCommands |
| 87 | The presented idempotency key re-derives from the request inputs on both receipt operations; an underived key refuses before capability, receipt, or exec. | executeTerminate, executeRestore | TestExecuteRefusesWrongIdempotencyKey |
| 88 | The AX-side state memory records and looks up per-instance states with identity and value grammars: creating never records, foreign documents refuse, failures install a first echo or the honest unknown but never overwrite an authoritative record, and the memory is advisory for admission. Operation outcome reports and binding documents persist beside the states with key, identity, and shape grammars; closure reports bind the current incarnation, a create or restore success rotates it, and the attach barrier proves exactly from current-incarnation reports. | InstanceStates | TestInstanceStatesRoundTrip, TestOpenInstanceStatesRefusesEmptyRoot, TestInstanceStatesRefusesCreating, TestInstanceStatesRefusesBadIdentity, TestInstanceStatesRefusesUnknownState, TestInstanceStatesRefusesForeignDocument, TestRecordFailureLeavesPresenceReconciliation, TestInstanceStatesOutcomeRoundTrip, TestInstanceStatesBindingDocRoundTrip, TestInstanceStatesLookupOutcomeRefusesForgedKey, TestExecuteQuiesceStateWriteFailureRefusesWiredAttach, TestExecuteQuiesceRetryAfterStateWriteFailureHealsBarrier, TestExecuteBoundaryBarrierSurvivesStateWriteFailure, TestExecuteAttachErrorsOnCorruptBarrierScan, TestExecuteAttachErrorsOnEmptyBarrierTime, TestInstanceIncarnationRoundTrip, TestQuiesceBarrierIgnoresSupersededIncarnation |
| 89 | Every mutating operation is idempotent under retry and resumes uncertainty through status: second identical requests replay without new execs, uncertain receipts reconcile before resuming, and tampered replays refuse. | Lifecycle.Execute | TestExecuteCreateIdempotent, TestExecuteAttachIdempotent, TestExecuteQuiesceIdempotent, TestExecuteBoundaryIdempotent, TestExecuteStopIdempotent, TestExecuteTerminateIdempotent, TestExecuteRestoreIdempotent, TestExecuteTerminateResumesAfterEffectFailure, TestExecuteRestoreResumesAfterRecheckFailure, TestExecuteResumeRefusesWhenStatusProvesOtherwise |
| 90 | B16: lifecycle entries take no ambient input, so a nested invocation cannot collide through them; the child environment scrubs the ambient tmux variables, and the Acquire-level over-strict nested refusal stands unchanged as first-leaf-owned. | Lifecycle.Execute, scrubTmuxEnv | TestLifecycleIgnoresAmbientEnvironment, TestScrubTmuxEnv, TestOSRunnerScrubsAmbientInChild |
| 91 | Terminate-stale decides over the landed fencing authorization with local target and winner bindings: the fencing decision authorizes exactly the keyed stale lease, and only a winner that still wins authorizes. | executeTerminate, mapFencingError | TestExecuteTerminateRefusals, TestExecuteTerminateTargetLeaseIdentity, TestExecuteTerminateTargetSessionBinds, TestExecuteTerminateWinnerTracksCurrentLease |
| 92 | The after-restore composition refreshes the mesh lease under the measured bound before deciding through the landed wrapper owner (Table E): local wins launch or reattach through backend restore, remote winners offer attach or takeover with no backend effects even under a lapsed local grant, and every other case parks without effects; malformed mode, operation, or dependencies refuse before any decision; the executed restore binds to the decided session, bootstrap, instance, backend, generation, and winner as one composed authorization, expired answers never verify, and the bound holds against non-cooperative adapters. | ExecuteWrapperRestore | TestWrapperRestoreLocalWinResumes, TestWrapperRestoreReattachReplaysBackend, TestWrapperRestoreLapsedGrantRemoteOffer, TestWrapperRestoreParksWithoutEffects, TestWrapperRestoreRefusesMalformed, TestWrapperRestoreRefreshBoundMeasured, TestWrapperRestoreRefusesDivergentWinner, TestWrapperRestoreRefusesDivergentWinnerLease, TestWrapperRestoreRefusesDivergentWinnerEpoch, TestReviewWrapperBootstrapMustBindExecutedRestore, TestReviewWrapperInstanceMustBindExecutedRestore, TestWrapperRestoreRefusesDivergentSession, TestWrapperRestoreRefusesDivergentGeneration, TestWrapperRestoreRefusesDivergentBackend, TestWrapperRestoreExpiredRefreshParksUnverified, TestWrapperRestoreRefreshBoundEnforced, TestWrapperRestoreRefreshRaceRejectsLateAnswer |

## Lifecycle adapters (Section 4.2, second leaf)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 59 | The exec adapter runs -S vectors with AX-owned argv only: exits and both streams report as data, TMUX/TMUX_TMPDIR scrub from the child, malformed argv refuses, transport failures and signal deaths pass through as errors and prove nothing, and NUL-containing arguments refuse before any process starts. | OSRunner (exec.go) | TestOSRunnerReportsExitAsData, TestOSRunnerCapturesStderr, TestOSRunnerSignalDeathProvesNothing, TestOSRunnerScrubsAmbientInChild, TestOSRunnerRefusesBadArgv, TestOSRunnerRefusesNulArgument, TestOSRunnerTransportErrorProvesNothing, TestScrubTmuxEnv |
| 60 | The primitive vocabulary is exactly the ten tmux directives and the per-operation map covers exactly the eight lifecycle operations; unknown primitives and non-lifecycle operations refuse. | ParseDirective, DirectivesFor | TestParseDirectiveAdmitsTenPrimitives, TestParseDirectiveRefusesUnknown, TestDirectivesForMapsEightOperations, TestDirectivesForRefusesNonLifecycle |
| 61 | Every built vector is the fixed -S shape for its directive: the socket is pinned to the runtime leaf (ambient sockets refuse), targets parse as UUIDv7, the quiescence channel parses as UUIDv7, the attach vector states its input authorization (read-only emits -r, unstated refuses), the boundary vector is the bare blocking wait, and detach-client addresses the session. | BuildCommand | TestBuildCommandVectors, TestBuildCommandAttachReadOnlyVector, TestBuildCommandAttachRequiresInputFlag, TestBuildCommandRefusesAmbientSocket, TestBuildCommandRefusesBadIdentity |
| 62 | Dialing the socket reports live, stale, or unknown: refused and missing sockets are stale, a missing directory and every other failure is unknown, never absent. | UnixDialer.Dial | TestUnixDialerLiveStaleUnknown, TestUnixDialerMissingSocketIsStale, TestUnixDialerMissingDirectoryIsUnknown |
| 63 | The prober enforces dependencies, length, and custody before dialing: stale reports not-running, unknown errors, live attests through the admission, and admission failures pass through. A writable root refuses at Probe like at the spawner. | ServerProber.Probe | TestServerProberRefusesWithoutDependencies, TestServerProberMapsDialOutcomes, TestServerProberAdmissionFailurePassesThrough, TestServerProberEnforcesLengthAndCustody, TestServerProberRefusesWritableRoot |
| 64 | The broker prober reports the miss, admission reconciliation fails closed without a verifier, and the Production constructor wires the prober, spawner, dialer, and admission into the Dependencies. | probe.go | TestBrokerProberReportsMiss, TestReconcileAdmissionFailsClosedWithoutVerifier, TestProductionWiresDependencies |
| 65 | The spawner enforces length, custody, and the exec-site socket pin before spawning: a stale socket unlinks (hook-observed), a non-socket refuses, and exit/transport outcomes map to the spawn verdict, and a world-writable root refuses before any spawn vector runs. | ServerSpawner.Spawn | TestServerSpawnerRefusesWithoutRunner, TestServerSpawnerPinAndCustody, TestServerSpawnerUnlinksStaleSocket, TestServerSpawnerRefusesNonSocketFile, TestServerSpawnerMapsRunnerOutcome, TestServerSpawnerRefusesAbsentLeaf, TestServerSpawnerRefusesWritableRoot |

## Socket bind step (Sections 3.2, 4.2; B18, B19, B20)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 66 | B18: the socket path refuses at or beyond the platform sun_path limit (104/108), counted in bytes. | CheckSocketLength | TestCheckSocketLengthBounds, TestCheckSocketLengthCountsBytes |
| 67 | B19/B20: the socket verifies at exactly the runtime leaf, and the leaf custody delegates to the landed Verify with its refusal propagated. | CheckSocketCustody | TestCheckSocketCustodyAdmitsCompliant, TestCheckSocketCustodyRefusesMisplacedSocket, TestCheckSocketCustodyPropagatesLeafRefusal |
| 68 | B20: the runtime root verifies as a caller-owned directory unwritable by others through the landed no-follow open, with the open handle bound back to its path. | checkCustodyRoot | TestCheckSocketCustodyRefusesUnsafeRoot, TestCheckSocketCustodyRefusesForeignRoot, TestCheckSocketCustodyIdentityMismatchRefuses, TestCheckSocketCustodyNoCacheAfterChange |
| 69 | B20: every ancestor verifies as a non-symlink directory unwritable by others up the lexical chain; sticky directories admit, symlinks and files refuse through the no-follow opens. | checkCustodyAncestors | TestCheckSocketCustodyRefusesWritableAncestor, TestCheckSocketCustodyAdmitsStickyAncestor, TestCheckSocketAncestorKindArm, TestCheckSocketCustodyRefusesIntermediateSymlink, TestSocketCustodySymlinkRefusesAtProbeAndSpawn |
| 70 | B19: the socket path itself is bindable when absent and unlinkable when a socket; any other kind refuses so the stale-unlink step can never remove a non-socket. | checkCustodySocket | TestCheckSocketCustodyAdmitsCompliant, TestCheckSocketCustodyRefusesNonSocketFile |

## Lifecycle bodies (Section 4.C)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 71 | Create bodies are the closed six-member shape with the operation context, binding, bootstrap, entrypoint, transport, and interactive members. | parseCreateBody | TestParseCreateBodyValid, TestParseCreateBodyMemberSet, TestParseCreateBodyMembers |
| 72 | Attach bodies are the closed eleven-member shape: identity (session, instance, backend, versions, generation), client, transport, input flag, deadline, and the ownership-neutral authorization, which carries no lease member. | parseAttachBody | TestParseAttachBodyValid, TestParseAttachBodyMemberSet, TestParseAttachBodyMembers |
| 73 | Terminate bodies are the closed four-member shape with the stale lease, stale epoch, and evidence members. | parseTerminateBody | TestParseTerminateBodyValid, TestParseTerminateBodyViolations |
| 74 | Restore bodies are the closed four-member shape with the prior binding, checkpoint, and bootstrap members. | parseRestoreBody | TestParseRestoreBodyValid, TestParseRestoreBodyViolations |

## Lifecycle dispatch (Sections 4, 4.D)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 75 | Execute admits all eight lifecycle operations through the dependency, vocabulary, scope, length, and custody pre-gates; each pre-gate refuses before dispatch on every operation, with per-operation custody twins for root, ancestor, socket-kind, and leaf-mode arms. | Lifecycle.Execute | TestExecuteRefusesMissingDependencies, TestExecuteRefusesNonLifecycleScope, TestExecuteRefusesLongSocket, TestExecuteRefusesUnsafeSocket, TestExecuteCustodyEntryRoots, TestExecuteCreateRefusesWritableRoot, TestExecuteAttachRefusesWritableRootValidBody, TestExecuteEachOperationRefusesWritableRoot, TestExecuteEachOperationRefusesWritableAncestor, TestExecuteEachOperationRefusesNonSocketKind, TestExecuteEachOperationRefusesBadLeafMode, TestExecuteCustodyNarrownessTwins |
| 76 | Every operation verifies the ax.tmux backend identity before dispatch; foreign backends refuse on all eight operations without exec. | checkLifecycleBackend | TestExecuteRefusesForeignBackend |
| 77 | Out-of-row error codes normalize to the protocol error; in-row codes propagate with their report, on all eight rows. | rowError | TestRowErrorNormalizesOutsideSet, TestRowErrorCoversEightRows, TestLifecycleEmittedCodesAreRowAllowed |

## Lifecycle operations (Section 4.C)

8 of 8 operations driven through the production entry `Execute`
(lifecycle.go:192), one dispatch arm each, plus the after-restore
composition entry `ExecuteWrapperRestore` (wrapper.go:100):

| Operation | Dispatch arm | Test |
|-----------|--------------|------|
| create | executeCreate (lifecycle.go:392) | TestExecuteCreateHeadless, TestExecuteCreateInteractive, TestExecuteCreateIdempotent, TestExecuteCreateRefusals, TestExecuteCreateBindingAgreement, TestExecuteCreateRefusesWritableRoot, TestExecuteCreateReplayPreservesBarrier, TestExecuteCreateIncarnationWriteFailureFailsClosed, TestEffectBoundaryAuditCreateIssuesSingleCommand |
| attach | executeAttach (ops.go:68) | TestExecuteAttach, TestExecuteAttachRefusals, TestExecuteAttachIdempotent, TestAttachDescriptorNeverPersisted, TestExecuteAttachRefusesMeshWithoutCapability, TestExecuteAttachRefusesDeadline, TestExecuteAttachRefusesInputWhenQuiescing, TestExecuteReadOnlyAttachIsReadOnly, TestExecuteReadOnlyAttachPreservesQuiescedInput, TestExecuteQuiesceStateWriteFailureRefusesWiredAttach, TestExecuteAttachErrorsOnCorruptBarrierScan, TestExecuteAttachErrorsOnEmptyBarrierTime, TestReviewExistingActiveMemoryCannotReopenQuiescedInput, TestReviewFailedStopCannotEraseQuiescence, TestExecuteFailedStopPreservesQuiescingMemory, TestExecuteAdvisoryMemoryNeverDecidesAdmission, TestExecuteAttachErrorsOnEmptyBarrierKey, TestExecuteAttachErrorsOnStateReadFailure, TestAttachRefusesAfterQuiescenceWhenAdmittedLate, TestAttachQuiesceOrderingOverlapQuiesce, TestAttachQuiesceOrderingOverlapBoundary, TestExecuteAttachErrorsOnIncarnationReadFailure, TestExecuteAttachErrorsOnOutcomesDirReadFailure, TestExecuteAttachErrorsOnOutcomeFileReadFailure, TestAttachRefusesGenerationRotatedDuringLockWait, TestAttachRefusesDeadlineExpiredDuringAdmission, TestAttachRefusesDeadlineOnTheInstantDuringAdmission, TestAttachAuthorizationMembersRefuseWithoutReceipt, TestAttachOverlapRequiresMultiAttach, TestAttachOverlapInputRequiresMultipleInputClients, TestAttachSameClientRetryIgnoresOverlap, TestAttachOverlapPeerReadFailureFailsClosed, TestAttachPeerDirectoryFailsClosed, TestAttachSameClientRetryWithPeerPresent, TestAttachWaitBoundedByOperationDeadline, TestAttachWaitCancelledReturnsPromptly |
| status | executeStatus (lifecycle.go:454) | TestExecuteStatusAbsent, TestExecuteStatusPresent, TestExecuteStatusProviderConditional, TestExecuteStatusContradictions, TestExecuteStatusNonMatchEachMember, TestExecuteStatusExactUnboundSessionRefusesUnknown, TestExecuteStatusBindingTamperRefusesIntegrity, TestExecuteStatusRefusesNegativeAttachedCount, TestObserveStatusSessionScopedObservesRecordedTuple, TestObserveStatusExactBackendDriftNonMatches, TestObserveStatusExactProtocolDriftNonMatches, TestExecuteStatusAttachedExitIsUnknown, TestExecuteStatusForeignAttachedRowIsUnknown, TestExecuteStatusUnknownPanesExitIsUnknown, TestExecuteStatusMalformedAttachedCountIsUnknown, TestExecuteStatusProbeHonorsOperationDeadline |
| quiesce | executeEngineOp (lifecycle.go:480) | TestExecuteQuiesce, TestExecuteQuiesceIdempotent, TestExecuteQuiesceBarrierSequence, TestExecuteQuiesceEmitsNoProviderInput, TestExecuteQuiesceRefusesFailedClose, TestExecuteQuiesceReplayKeepsTimestamps, TestExecuteQuiesceCorruptOutcomeRefusesIntegrity, TestExecuteQuiesceForgedOutcomeKeyRefusesIntegrity, TestExecuteQuiesceEmptyRecordedTimeRefusesIntegrity, TestOutcomeRecordCrashConverges, TestExecuteQuiesceStateWriteFailureRefusesWiredAttach, TestExecuteQuiesceRetryAfterStateWriteFailureHealsBarrier, TestExecuteQuiesceRefusesStaleFactsBetweenCommands, TestQuiesceBarrierWaitBoundedByOperationDeadline, TestQuiesceMalformedDeadlineRefusesBeforeBarrierWait |
| safe-boundary | executeEngineOp (lifecycle.go:480) | TestExecuteBoundary, TestExecuteBoundaryIdempotent, TestExecuteBoundaryWaitsForSignal, TestExecuteBoundaryBindsProofKind, TestExecuteBoundaryWaitTimeout, TestExecuteBoundaryPastDeadline, TestExecuteBoundaryRefusesFailedWait, TestExecuteBoundaryReplayKeepsProofAndTime, TestExecuteBoundaryCorruptOutcomeRefusesIntegrity, TestExecuteBoundaryEmptyRecordedTimeRefusesIntegrity, TestExecuteBoundaryRecordCrashRefusesIntegrity, TestObserveBoundaryBindsProviderRows, TestExecuteBoundaryBarrierSurvivesStateWriteFailure, TestEffectBoundaryAuditBoundaryTailIsReadOnly, TestBoundaryCommitWaitBoundedByOperationDeadline |
| stop | executeEngineOp (lifecycle.go:480) | TestExecuteStop, TestExecuteStopIdempotent, TestExecuteStopUnknownProbeIsNotClosure, TestExecuteStopUnmarkedExitIsNotClosure, TestExecuteStopRefusesStaleFactsAfterPoll, TestExecuteStopEscalatesAfterGracefulTimeoutAlone, TestExecuteStopPollHonorsOperationDeadline |
| terminate-stale | executeTerminate (ops.go:356) | TestExecuteTerminate, TestExecuteTerminateIdempotent, TestExecuteTerminateRefusals, TestExecuteTerminateResumesAfterEffectFailure, TestExecuteTerminateWinnerTracksCurrentLease, TestExecuteTerminateTargetLeaseIdentity, TestExecuteTerminateTargetSessionBinds, TestExecuteTerminateRefusesProviderOnlyCapability, TestExecuteTerminateRefusesOnDeadline, TestExecuteTerminateRedriveMismatch, TestExecuteTerminateRefusesAuthorizationExpiredAfterReceipt, TestExecuteEntryRefusesWrongAuthorizationKind, TestEffectBoundaryAuditTerminateIssuesNoPostWaitCommand, TestExecuteTerminateConfirmHonorsOperationDeadline |
| restore | executeRestore (ops.go:506) | TestExecuteRestore, TestExecuteRestoreMismatch, TestExecuteRestoreIdempotent, TestExecuteRestoreReturnsBinding, TestExecuteRestoreRefusesSwappedBindingDoc, TestExecuteRestoreResumesAfterRecheckFailure, TestExecuteResumeRefusesWhenStatusProvesOtherwise, TestExecuteRestoreRefusesWithoutRebootCapability, TestExecuteRestoreRefusesOnDeadline, TestExecuteRestoreRefusesOnDeadlineEffect, TestExecuteRestoreRefusesGenerationDrift, TestExecuteRestoreAcrossServerGeneration, TestExecuteRestoreSuccessorRetryConverges, TestExecuteRestorePriorValidationPrecedesEffects, TestExecuteRestoreRefusesSubstitutedInstance, TestExecuteRestoreRefusesSubstitutedSession, TestExecuteRestoreRefusesSubstitutedBackend, TestExecuteQuiesceStopRestoreReopensBarrier, TestExecuteRestoreReplayPreservesBarrier, TestExecuteRestoreIncarnationWriteFailureFailsClosed, TestExecuteRestoreRefusesAuthorizationExpiredAfterReceipt, TestExecuteEntryRefusesWrongAuthorizationKind, TestEffectBoundaryAuditRestoreIssuesSingleCommand |
| after-restore | ExecuteWrapperRestore (wrapper.go:100) | TestWrapperRestoreLocalWinResumes, TestWrapperRestoreReattachReplaysBackend, TestWrapperRestoreLapsedGrantRemoteOffer, TestWrapperRestoreParksWithoutEffects, TestWrapperRestoreRefusesMalformed, TestWrapperRestoreRefreshBoundMeasured, TestWrapperRestoreRefusesDivergentWinner, TestWrapperRestoreRefusesDivergentWinnerLease, TestWrapperRestoreRefusesDivergentWinnerEpoch, TestWrapperRestoreExpiredRefreshParksUnverified, TestWrapperRestoreRefreshBoundEnforced, TestWrapperRestoreRefreshRaceRejectsLateAnswer, TestReviewWrapperBootstrapMustBindExecutedRestore, TestReviewWrapperInstanceMustBindExecutedRestore, TestWrapperRestoreRefusesDivergentSession, TestWrapperRestoreRefusesDivergentGeneration, TestWrapperRestoreRefusesDivergentBackend |

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 78 | Create runs the closed-shape admission, binding agreement, and relay refusal through the engine: interactive returns the bound descriptor, headless returns none, and identical requests replay one spawn. A fresh create rotates the incarnation, superseding prior closures; replays never rotate, and a failed rotation fails the operation closed. The wrapper is the path's only command: a rotation during its round trip changes nothing further (single-command probe). | executeCreate | TestExecuteCreateInteractive, TestExecuteCreateHeadless, TestExecuteCreateIdempotent, TestExecuteCreateRefusals, TestExecuteCreateBindingAgreement, TestExecuteCreateReplayPreservesBarrier, TestExecuteCreateIncarnationWriteFailureFailsClosed, TestEffectBoundaryAuditCreateIssuesSingleCommand |
| 79 | Attach returns the caller-executed vector with its descriptor and never execs an interactive attach: closed-shape admission, deadline, matching-transport capability, the quiescing input barrier, live attestation, the authoritative barrier recheck under the ordering lock, the post-lock generation and deadline rechecks, and the durable client receipt, with identical requests replaying; read-only authorization emits the read-only vector, a read-only attach observes without reopening quiesced input, and a lost state-record write fails closed, while a lost outcome-report write after committed closure is bound B43 (input may reopen before the retry heals the report). The barrier verdict is the incarnation-scoped closure proof alone: stale memory, caller-carried source state, and failure results are never consulted for their value, so none of them can reopen a closed input; only a fresh create or restore reopens, by rotating the incarnation. A corrupt scan, an empty time, a keyless report, or an unreadable state, incarnation, outcomes-directory, or outcome record errors. A second client that may overlap a recorded peer needs multi_attach, and new input over a recorded input peer needs multiple_input_clients, with receipt-validated same-client retries exempt (identical bodies replay with a peer present; conflicting retries still refuse idempotency_mismatch) and unreadable peers failing closed; a directory at a receipt-shaped name is corruption and refuses, a valid receipt misfiled under another client's durable key refuses through the keyed binding before any exclusion applies, and a malformed receipt name refuses, while staging and non-receipt entries still skip; client liveness is bound B44 (owned by TASK-260922-vcx6yo). The admission and barrier waits honor the operation deadline and cancellation with no late commit. | executeAttach | TestExecuteAttach, TestExecuteAttachRefusals, TestExecuteAttachIdempotent, TestAttachDescriptorNeverPersisted, TestExecuteAttachRefusesMeshWithoutCapability, TestExecuteAttachRefusesDeadline, TestExecuteAttachRefusesInputWhenQuiescing, TestExecuteReadOnlyAttachIsReadOnly, TestExecuteReadOnlyAttachPreservesQuiescedInput, TestExecuteQuiesceStateWriteFailureRefusesWiredAttach, TestExecuteAttachErrorsOnCorruptBarrierScan, TestExecuteAttachErrorsOnEmptyBarrierTime, TestReviewExistingActiveMemoryCannotReopenQuiescedInput, TestReviewFailedStopCannotEraseQuiescence, TestExecuteFailedStopPreservesQuiescingMemory, TestExecuteAdvisoryMemoryNeverDecidesAdmission, TestExecuteAttachErrorsOnEmptyBarrierKey, TestExecuteAttachErrorsOnStateReadFailure, TestAttachRefusesAfterQuiescenceWhenAdmittedLate, TestAttachQuiesceOrderingOverlapQuiesce, TestAttachQuiesceOrderingOverlapBoundary, TestExecuteAttachErrorsOnIncarnationReadFailure, TestExecuteAttachErrorsOnOutcomesDirReadFailure, TestExecuteAttachErrorsOnOutcomeFileReadFailure, TestAttachRefusesGenerationRotatedDuringLockWait, TestAttachRefusesDeadlineExpiredDuringAdmission, TestAttachRefusesDeadlineOnTheInstantDuringAdmission, TestAttachAuthorizationMembersRefuseWithoutReceipt, TestAttachOverlapRequiresMultiAttach, TestAttachOverlapInputRequiresMultipleInputClients, TestAttachSameClientRetryIgnoresOverlap, TestAttachOverlapPeerReadFailureFailsClosed, TestAttachPeerDirectoryFailsClosed, TestAttachSameClientRetryWithPeerPresent, TestAttachPeerFilenameMismatchFailsClosed, TestAttachWaitBoundedByOperationDeadline, TestAttachWaitCancelledReturnsPromptly |
| 80 | Status classifies present, absent, and unknown observations against the AX-side memory: absence needs its stderr marker (unmarked, refused, and killed probes read unavailable), contradictions read unavailable, malformations refuse, and the provider triple evidences only when requested, capable, and observed. Status observes the recorded binding tuple: drifted queries non-match member by member, unbound sessions refuse unknown, tampered documents refuse integrity, negative backend counts refuse, and nonzero attached exits, foreign attached rows, unmarked panes exits, and malformed counts with plausible output read unavailable. The live probes run under the query deadline in-flight: a probe still blocked at the bound concludes unknown with no report, never absent. | executeStatus, ObserveStatus | TestExecuteStatusPresent, TestExecuteStatusAbsent, TestExecuteStatusPresentIncludesIdentity, TestExecuteStatusUnknownProbeIsUnavailable, TestExecuteStatusProviderConditional, TestExecuteStatusContradictions, TestClassifyStatusAbsentProbe, TestClassifyStatusPresentProbe, TestClassifyAbsenceMarkers, TestObserveStatusPresentParked, TestObserveStatusAbsentProbe, TestObserveStatusMemoryReconciles, TestObserveStatusUnknowns, TestProviderTripleRule, TestCutRowShapes, TestExecuteStatusNonMatchEachMember, TestExecuteStatusExactUnboundSessionRefusesUnknown, TestExecuteStatusBindingTamperRefusesIntegrity, TestObserveStatusSessionScopedObservesRecordedTuple, TestObserveStatusExactBackendDriftNonMatches, TestObserveStatusExactProtocolDriftNonMatches, TestExecuteStatusRefusesNegativeAttachedCount, TestExecuteStatusAttachedExitIsUnknown, TestExecuteStatusForeignAttachedRowIsUnknown, TestExecuteStatusUnknownPanesExitIsUnknown, TestExecuteStatusMalformedAttachedCountIsUnknown, TestExecuteStatusProbeHonorsOperationDeadline |
| 81 | Quiesce, boundary, and stop run through the engine with their outcome members: quiesce locks then detaches with no provider input, the boundary waits blocking with the proof kind bound into its evidence and replays its recorded time, and stop proves closure only from marked absence; identical requests replay. Corrupt, forged-key, and empty-time outcome reports refuse integrity instead of becoming fresh evidence; a failed barrier-state write fails the operation instead of reporting closure, and the retry heals the barrier with the original time. The stop escalation revalidates deadline, authorization, and generation after its poll before the kill issues — a graceful timeout escalates, any other stale fact refuses — the quiesce detach revalidates between the commands, the boundary tail is read-only, and the barrier waits honor the deadline on all three paths. The stop poll itself honors the operation deadline in-flight: a probe still blocked at the bound concludes unknown with no kill and no closure, never absence. The post-escalation re-confirmation observes under the operation deadline, so a cancellation-honouring executor still observes the closure after the graceful wait expires. | executeEngineOp | TestExecuteQuiesce, TestExecuteQuiesceIdempotent, TestExecuteQuiesceBarrierSequence, TestExecuteQuiesceEmitsNoProviderInput, TestExecuteQuiesceRefusesFailedClose, TestExecuteQuiesceReplayKeepsTimestamps, TestExecuteBoundary, TestExecuteBoundaryIdempotent, TestExecuteBoundaryWaitsForSignal, TestExecuteBoundaryBindsProofKind, TestExecuteBoundaryWaitTimeout, TestExecuteBoundaryPastDeadline, TestExecuteBoundaryRefusesFailedWait, TestExecuteBoundaryReplayKeepsProofAndTime, TestExecuteStop, TestExecuteStopIdempotent, TestExecuteStopUnknownProbeIsNotClosure, TestExecuteStopUnmarkedExitIsNotClosure, TestExecuteStopPollHonorsOperationDeadline, TestBoundWaitTakesLesser, TestLesserDeadlineTakesLesser, TestLifecycleIgnoresAmbientEnvironment, TestObserveBoundaryBindsProviderRows, TestOutcomeRecordCrashConverges, TestExecuteQuiesceCorruptOutcomeRefusesIntegrity, TestExecuteBoundaryCorruptOutcomeRefusesIntegrity, TestExecuteQuiesceForgedOutcomeKeyRefusesIntegrity, TestExecuteQuiesceEmptyRecordedTimeRefusesIntegrity, TestExecuteBoundaryEmptyRecordedTimeRefusesIntegrity, TestExecuteBoundaryRecordCrashRefusesIntegrity, TestExecuteQuiesceStateWriteFailureRefusesWiredAttach, TestExecuteQuiesceRetryAfterStateWriteFailureHealsBarrier, TestExecuteBoundaryBarrierSurvivesStateWriteFailure, TestExecuteStopRefusesStaleFactsAfterPoll, TestExecuteStopEscalatesAfterGracefulTimeoutAlone, TestExecuteStopEscalationSucceedsWithContextHonoringRunner, TestExecuteQuiesceRefusesStaleFactsBetweenCommands, TestBackendPostWaitBoundaryRefusesWithoutRecheck, TestEffectBoundaryAuditBoundaryTailIsReadOnly, TestQuiesceBarrierWaitBoundedByOperationDeadline, TestBoundaryCommitWaitBoundedByOperationDeadline, TestQuiesceMalformedDeadlineRefusesBeforeBarrierWait |
| 82 | Terminate-stale runs the key, capability, transition, authorization, generation, fencing, target, winner, and deadline gates before the receipt-bound kill with its live-session confirm; an unproven confirm is unknown, never termination — including a confirm still blocked when the operation deadline fires, which concludes unknown in-flight with no termination evidence; the memory adopts stopped, identical requests replay, and effect failures resume through status. The re-confirm tail issues no command after its poll however the facts moved (no-post-wait-command probe). | executeTerminate | TestExecuteTerminate, TestExecuteTerminateRefusals, TestExecuteTerminateIdempotent, TestExecuteTerminateResumesAfterEffectFailure, TestExecuteTerminateUnknownConfirmIsNotTerminated, TestExecuteTerminateConfirmHonorsOperationDeadline, TestExecuteTerminateWinnerTracksCurrentLease, TestExecuteTerminateTargetLeaseIdentity, TestExecuteTerminateTargetSessionBinds, TestExecuteTerminateRefusesProviderOnlyCapability, TestExecuteTerminateRefusesOnDeadline, TestExecuteTerminateRedriveMismatch, TestExecuteTerminateRefusesAuthorizationExpiredAfterReceipt, TestExecuteEntryRefusesWrongAuthorizationKind, TestEffectBoundaryAuditTerminateIssuesNoPostWaitCommand |
| 83 | Restore runs the capability, prior-binding, and receipt-bound persist-plus-wrapper sequence, then returns the required binding and the parked wire result; the after-restore branch decision (local resume, remote offer, or parked under the enforced refresh bound) is the wrapper composition (row 92, Table E). A substituted prior refuses the mismatch, a swapped document refuses the integrity failure, the memory adopts parked, identical requests replay, and rechecks resume through status. A fresh restore rotates the incarnation, superseding prior closures; replays never rotate, and a failed rotation fails the operation closed. A reboot restore validates its prior before the effects commit, then mints a successor generation persisted after the effects, with successor retries converging. The wrapper is the path's only command (single-command probe). | executeRestore | TestExecuteRestore, TestExecuteRestoreMismatch, TestExecuteRestoreIdempotent, TestExecuteRestoreReturnsBinding, TestExecuteRestoreRefusesSwappedBindingDoc, TestExecuteCreatePersistsBindingDocForRestore, TestExecuteCreateRefusesBindingDocFailure, TestExecuteResumeRefusesWhenStatusProvesOtherwise, TestExecuteRestoreResumesAfterRecheckFailure, TestExecuteRestoreRefusesWithoutRebootCapability, TestExecuteRestoreRefusesOnDeadline, TestExecuteRestoreRefusesOnDeadlineEffect, TestExecuteRestoreRefusesGenerationDrift, TestExecuteRestoreAcrossServerGeneration, TestExecuteRestoreSuccessorRetryConverges, TestExecuteRestorePriorValidationPrecedesEffects, TestExecuteRestoreRefusesSubstitutedInstance, TestExecuteRestoreRefusesSubstitutedSession, TestExecuteRestoreRefusesSubstitutedBackend, TestExecuteQuiesceStopRestoreReopensBarrier, TestExecuteRestoreReplayPreservesBarrier, TestExecuteRestoreIncarnationWriteFailureFailsClosed, TestExecuteRestoreRefusesAuthorizationExpiredAfterReceipt, TestExecuteEntryRefusesWrongAuthorizationKind, TestEffectBoundaryAuditRestoreIssuesSingleCommand |
| 84 | The receipt discipline binds, replays, resumes, and completes under the idempotency key: tampered completions refuse member by member, corrupt images refuse, hook failures report uncertainty, and forged effect evidence fails the run pre-commit. | executeWithReceipt, resumeUncertain, runLifecycleEffects | TestExecuteTerminateReplayRefusesTamperedImage, TestExecuteTerminateReplayRefusesCorruptImage, TestExecuteTerminateBindHookFailure, TestRunLifecycleEffectsRefusesForgedEvidence |
| 85 | Deadlines are strict bounds at entry and before every effect: the on-deadline instant refuses, and the entry arm leaves the key unbound where the loop would bind it. The attach lock wait counts against the deadline: an operation that expires mid-wait refuses even while its authorization stays valid. The attach, quiesce, and boundary waits refuse at the deadline with no late commit, and cancellation returns promptly; the stop escalation answers to the operation deadline, distinct from the graceful wait. The read-only probes honor the same bound in-flight: the stop poll, the terminate confirm, and the status probes conclude unknown at the deadline, never a verdict. | ops.go, executeAttach | TestExecuteTerminateRefusesOnDeadline, TestExecuteRestoreRefusesOnDeadline, TestExecuteRestoreRefusesOnDeadlineEffect, TestExecuteAttachRefusesDeadline, TestAttachRefusesDeadlineExpiredDuringAdmission, TestAttachRefusesDeadlineOnTheInstantDuringAdmission, TestAttachWaitBoundedByOperationDeadline, TestAttachWaitCancelledReturnsPromptly, TestQuiesceBarrierWaitBoundedByOperationDeadline, TestBoundaryCommitWaitBoundedByOperationDeadline, TestExecuteStopRefusesStaleFactsAfterPoll, TestExecuteStopPollHonorsOperationDeadline, TestExecuteTerminateConfirmHonorsOperationDeadline, TestExecuteStatusProbeHonorsOperationDeadline |
| 86 | Generations validate at entry and recheck before every effect: drift between entry and effect refuses under the stale binding with nothing committed. The stop escalation and the quiesce detach revalidate generation after their waits before the next destructive command. | ops.go | TestExecuteEntryRefusesStaleGeneration, TestExecuteRestoreRefusesGenerationDrift, TestAttachRefusesGenerationRotatedDuringLockWait, TestExecuteStopRefusesStaleFactsAfterPoll, TestExecuteQuiesceRefusesStaleFactsBetweenCommands |
| 87 | The presented idempotency key re-derives from the request inputs on both receipt operations; an underived key refuses before capability, receipt, or exec. | executeTerminate, executeRestore | TestExecuteRefusesWrongIdempotencyKey |
| 88 | The AX-side state memory records and looks up per-instance states with identity and value grammars: creating never records, foreign documents refuse, failures install a first echo or the honest unknown but never overwrite an authoritative record, and the memory is advisory for admission. Operation outcome reports and binding documents persist beside the states with key, identity, and shape grammars; closure reports bind the current incarnation, a create or restore success rotates it, and the attach barrier proves exactly from current-incarnation reports. | InstanceStates | TestInstanceStatesRoundTrip, TestOpenInstanceStatesRefusesEmptyRoot, TestInstanceStatesRefusesCreating, TestInstanceStatesRefusesBadIdentity, TestInstanceStatesRefusesUnknownState, TestInstanceStatesRefusesForeignDocument, TestRecordFailureLeavesPresenceReconciliation, TestInstanceStatesOutcomeRoundTrip, TestInstanceStatesBindingDocRoundTrip, TestInstanceStatesLookupOutcomeRefusesForgedKey, TestExecuteQuiesceStateWriteFailureRefusesWiredAttach, TestExecuteQuiesceRetryAfterStateWriteFailureHealsBarrier, TestExecuteBoundaryBarrierSurvivesStateWriteFailure, TestExecuteAttachErrorsOnCorruptBarrierScan, TestExecuteAttachErrorsOnEmptyBarrierTime, TestInstanceIncarnationRoundTrip, TestQuiesceBarrierIgnoresSupersededIncarnation |
| 92 | The after-restore composition refreshes the mesh lease under the measured bound before deciding through the landed wrapper owner (Table E): local wins launch or reattach through backend restore, remote winners offer attach or takeover with no backend effects even under a lapsed local grant, and every other case parks without effects; malformed mode, operation, or dependencies refuse before any decision; the executed restore binds to the decided session, bootstrap, instance, backend, generation, and winner as one composed authorization, expired answers never verify, and the bound holds against non-cooperative adapters. | ExecuteWrapperRestore | TestWrapperRestoreLocalWinResumes, TestWrapperRestoreReattachReplaysBackend, TestWrapperRestoreLapsedGrantRemoteOffer, TestWrapperRestoreParksWithoutEffects, TestWrapperRestoreRefusesMalformed, TestWrapperRestoreRefreshBoundMeasured, TestWrapperRestoreRefusesDivergentWinner, TestWrapperRestoreRefusesDivergentWinnerLease, TestWrapperRestoreRefusesDivergentWinnerEpoch, TestReviewWrapperBootstrapMustBindExecutedRestore, TestReviewWrapperInstanceMustBindExecutedRestore, TestWrapperRestoreRefusesDivergentSession, TestWrapperRestoreRefusesDivergentGeneration, TestWrapperRestoreRefusesDivergentBackend, TestWrapperRestoreExpiredRefreshParksUnverified, TestWrapperRestoreRefreshBoundEnforced, TestWrapperRestoreRefreshRaceRejectsLateAnswer |

## Lifecycle crash and ambient discipline (Sections 4.2, 4.C; B16)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 89 | Every mutating operation is idempotent under retry and resumes uncertainty through status: second identical requests replay without new execs, uncertain receipts reconcile before resuming, and tampered replays refuse. | Lifecycle.Execute | TestExecuteCreateIdempotent, TestExecuteAttachIdempotent, TestExecuteQuiesceIdempotent, TestExecuteBoundaryIdempotent, TestExecuteStopIdempotent, TestExecuteTerminateIdempotent, TestExecuteRestoreIdempotent, TestExecuteTerminateResumesAfterEffectFailure, TestExecuteRestoreResumesAfterRecheckFailure, TestExecuteResumeRefusesWhenStatusProvesOtherwise |
| 90 | B16: lifecycle entries take no ambient input, so a nested invocation cannot collide through them; the child environment scrubs the ambient tmux variables, and the Acquire-level over-strict nested refusal stands unchanged as first-leaf-owned. | Lifecycle.Execute, scrubTmuxEnv | TestLifecycleIgnoresAmbientEnvironment, TestScrubTmuxEnv, TestOSRunnerScrubsAmbientInChild |
| 91 | Terminate-stale decides over the landed fencing authorization with local target and winner bindings: the fencing decision authorizes exactly the keyed stale lease, and only a winner that still wins authorizes. | executeTerminate, mapFencingError | TestExecuteTerminateRefusals, TestExecuteTerminateTargetLeaseIdentity, TestExecuteTerminateTargetSessionBinds, TestExecuteTerminateWinnerTracksCurrentLease |

## Gate × entry census (mirrored from TRACEABILITY)

Table A (first leaf, 24 × 11) is frozen and unchanged; only
row N-verify-absence-detail widened its killer mask to the new
entries (old cells unchanged). Tables B/C/D/E below are copied
verbatim from the package TRACEABILITY, whose counts were
verified by script over the tables.
## Second-leaf census

New columns: EXC/EXA/EXS/EXE/EXT/EXR = Execute create, attach,
status, engine ops (quiesce/boundary/stop), terminate-stale, restore;
PRB = ServerProber.Probe, SPW = ServerSpawner.Spawn, DIAL =
UnixDialer.Dial, CMD = BuildCommand, DIRV = DirectivesFor, PDIR =
ParseDirective, LEN = CheckSocketLength, CUS = CheckSocketCustody,
REC/LOOK = InstanceStates Record/Lookup (including the RecordOutcome/RecordBindingDoc/RecordIncarnation writes and the LookupOutcome/LookupBindingDoc/LookupIncarnation reads), CLS = ClassifyStatus, OBSP =
ObserveStatus called directly, PFX = PerformEffect called directly,
OSR = OSRunner called directly, UNT = unexported internals called
directly in unit tests (body parsers, rowError, providerTriple,
cutRow, ReconcileAdmission). Cell codes M/U1/U2/U3/B as above, with
U2 reasons tagged: U2a = entry never calls this code (the retired
U2b, reachable-but-undriven, is bound B39 now: those cells state
what the committed tests drive, with the narrowed direction
measured at the listed entry); U2c = reachable only through
caller-injected
Production Dependencies with no committed composition test (bound
B27).

Table B: old gates × new entries (24 × 21 = 504 cells; 10 M, 494
U2a). Only the verify side (via custody delegation) and the server
attestation (via attach) are reachable from new entries; every other
old gate reads U2a throughout because no new code calls it (new code
never ensures, never lexically gates, never reads ambient, never
remaps, never consults creation, and never calls BuildArgv,
CheckBrokerContact, or the Acquire call sites).

| Gate (site) | EXC | EXA | EXS | EXE | EXT | EXR | PRB | SPW | DIAL | CMD | DIRV | PDIR | LEN | CUS | REC | LOOK | CLS | OBSP | PFX | OSR | UNT |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| caller (acquire.go:149) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| details (acquire.go:152) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| render (acquire.go:289) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| dispatch (acquire.go:163) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| plat (runtime.go:76) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| root (runtime.go:87) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| name (runtime.go:110) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| guard (runtime.go:93) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| wiring (acquire.go:155,183,228) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| override (socket.go:85) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| collision (socket.go:106) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| deps (acquire.go:180,222) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| session (acquire.go:225, argv.go:20) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| commit (runtime_unix.go:35) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| verify (runtime_unix.go:106) | M UnsafeSocket/create / N-verify-absence-detail | M UnsafeSocket/attach / same row | M UnsafeSocket/status / same row | M UnsafeSocket/quiesce-input,boundary,stop / same row | M UnsafeSocket/terminate-stale / same row | M UnsafeSocket/restore / same row | M EnforcesLengthAndCustody / same row | M RefusesAbsentLeaf / same row | U2a | U2a | U2a | U2a | U2a | M PropagatesLeafRefusal / same row | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| absence (acquire.go:184,258) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| creation (acquire.go:277) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| broker (readiness.go:80) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| broker-server (readiness.go:86) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| server (readiness.go:40) | U2a | M AttachRefusals fresh-decoy,stale-admission / N-attach-attestation-decoy | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| running (acquire.go:236) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| probe (acquire.go:190,232) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| spawn (acquire.go:246) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| argv (argv.go:17,24) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |

Table C: new gates × old entries (149 × 11 = 1639 cells; 1621 U2a,
18 U2c). Table C derives its gate set from Table D: the eight
revision-5 gates (barrierincarnation, barrierkey, attachhealth,
failurerecord, createincarnation, restoreincarnation,
incarnationrecord, incarnationlookup), the three revision-6
gates (recheck, ordering, barrierread), the four revision-8
gates (effectrecheck, overlapmulti, overlapinput, waitbound), and
the four revision-9 gates (peershape, overlapreplay,
confirmbound, statusbound) and the one revision-10 gate
(escalationbound) add 220 U2a cells, because old code
calls none of them. Old code
predates new code and calls
none of it, except
through caller-injected Dependencies: AFG reaches the adapter and
bind gates below only when the caller injects the Production
implementations, and no committed test composes Production into
Acquire (bound B27), so those 18 cells read U2c. ABG reaches no new
gate (background never probes, spawns, or binds). The U2c cells are
AFG × {dialstale, probedeps, probeunknown, spawnpin, spawnshape,
spawnexit, spawnnil, unlinkkind, reconcilenil, sunpath, placement,
leafdelegate, leafwiring, rootwrite, rootowner, ancestorwrite,
ancestorkind, socketkind}; every other Table C cell is U2a.

Table D: new gates × new entries (149 × 21 = 3129 cells; 207 M, 187
B, 2735 U; counts verified by script over the table). Test names
drop the common Test prefix; "same N rows" means the row list in the
gate's first M cell.

| Gate (site) | EXC | EXA | EXS | EXE | EXT | EXR | PRB | SPW | DIAL | CMD | DIRV | PDIR | LEN | CUS | REC | LOOK | CLS | OBSP | PFX | OSR | UNT |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| directive (exec.go:104) | B39 post-gate vectors; M at PDIR,CMD | B39 post-gate; M at PDIR,CMD | B39 post-gate; M at PDIR,CMD | B39 post-gate; M at PDIR,CMD | B39 post-gate; M at PDIR,CMD | B39 post-gate; M at PDIR,CMD | U1 | U1 | U1 | M RefusesUnknownDirective / N-exec-directive-unknown | U1 | M ParseDirectiveRefusesUnknown / same row | U1 | U1 | U1 | U1 | U1 | U1 | B39 engine effects valid; M at PDIR,CMD | U1 | U1 |
| cmdsocket (exec.go:158) | B39 derived sockets; M at CMD | B39 derived; M at CMD | B39 derived; M at CMD | B39 derived; M at CMD | B39 derived; M at CMD | B39 derived; M at CMD | U1 | U1 | U1 | M RefusesAmbientSocket / N-exec-ambient-socket | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 derived; M at CMD | U1 | U1 |
| cmdident (exec.go:200,213,226) | B39 parsed identities; M at CMD | B39 parsed; M at CMD | B39 parsed; M at CMD | B39 parsed; M at CMD | B39 parsed; M at CMD | B39 parsed; M at CMD | U1 | U1 | U1 | M RefusesBadIdentity / N-exec-instance-identity, N-exec-session-identity, N-exec-quiescence-identity | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 parsed; M at CMD | U1 | U1 |
| scrub (exec.go:69) | U2a | U2a | U2a | U2a | U2a | U2a | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M ScrubsAmbientInChild / N-exec-scrub-tmux, N-exec-scrub-tmpdir | M ScrubTmuxEnv / same 2 rows |
| cmdvector (exec.go:158) | M CreateInteractive / N-exec-vector-detached | M Attach / N-cmdvector-attach | U2a status builds no vectors | M BoundaryWaitsForSignal / N-boundary-blocking-vector; quiesce/stop subcommand-only (M at CMD) | B39 kill/has subcommands; M at CMD | M Restore / N-exec-vector-detached | U1 | U1 | U1 | M Vectors / N-exec-vector-detached, N-cmdvector-attach, N-boundary-blocking-vector, N-cmdvector-detach | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 subcommand-only; M at CMD | U1 | U1 |
| osrun (exec.go:40) | U2a | U2a | U2a | U2a | U2a | U2a | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M SignalDeathProvesNothing / N-osrunner-signal-death; other mappings B30 total | U1 |
| directives (backend.go:30) | M RefusesNonLifecycleScope / N-exec-directives-manifest | M same test+row; pre-gate op-independent | M same test+row; pre-gate | M same test+row; pre-gate | M same test+row; pre-gate | M same test+row; pre-gate | U1 | U1 | U1 | U1 | M DirectivesForRefusesNonLifecycle / same row | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| effectvocab (backend.go:100) | B39 landed effects; M at PFX | B39 landed; M at PFX | U1 | B39 landed; M at PFX | B39 landed; M at PFX | B39 landed; M at PFX | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M RefusesUnknownVocabulary / N-backend-effect-vocabulary | U1 | U1 |
| exitrefused (backend.go:261) | B39 execs succeed; M at PFX | U1 attach never execs | U1 | B39 execs succeed; M at PFX | B39 kill confirms inline; M at PFX | B39 execs succeed; M at PFX | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M RefusesFailedExec / N-backend-exit-refused | U1 | U1 |
| transport (backend.go:261) | B39 runners scripted; M at PFX | U1 | U1 | B39 valid ctx; M at PFX | B39 scripted; M at PFX | B39 scripted; M at PFX | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M TransportProvesNothing / N-backend-transport-unknown | U1 | U1 |
| reconfirm (backend.go:224) | U1 | U1 | U1 | U1 | B39 single-effect failures; M at PFX | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M TerminateLiveRefuses, ReconfirmRefusesFailedKill / N-backend-reconfirm-failed-kill | U1 | U1 |
| boundarywait (backend.go:286) | U1 | U1 | U1 | M BoundaryWaitTimeout / N-backend-boundary-timeout | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M BoundaryTimeout / same row | U1 | U1 |
| escalation (backend.go:224) | U1 | U1 | U1 | B39 clean closes; M at PFX | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M EscalationKillFailure / N-stop-escalation-kill | U1 | U1 |
| persistmap (backend.go:132) | B39 bindings succeed; M at PFX | U1 | U1 | U1 | U1 | B39 priors mismatch first; M at PFX | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M BindingConflict / N-persist-conflict-map | U1 | U1 |
| attachmap (backend.go:158) | U1 | B39 receipts replay; M at PFX | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M AttachClient / N-attach-conflict-map | U1 | U1 |
| selfconfirm (backend.go:195) | U1 | U1 | U1 | U1 | B22 ResumesAfterEffectFailure; singleton | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B22 TerminateLiveRefuses; singleton | U1 | U1 |
| stoptimeout (backend.go:224) | U1 | U1 | U1 | B39 closes clean; M nowhere (B22 at PFX) | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B22 StopTimeout; singleton | U1 | U1 |
| cmdmap (backend.go:107) | B39 parsed identities; B28 at PFX | B39 parsed; B28 at PFX | B39 parsed; B28 at PFX | B39 parsed; B28 at PFX | B39 parsed; B28 at PFX | B39 parsed; B28 at PFX | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B28 untested mapping; builder rows at CMD | U1 | U1 |
| storenil (backend.go:132,158) | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B22 NilStoresRefuse; singleton | U1 | U1 |
| staterecord (state.go:74) | B39 adopted states; M at REC | B39 adopted; M at REC | B39 adopted; M at REC | B39 adopted; M at REC | B39 adopted; M at REC | B39 adopted; M at REC | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M RoundTrip, RefusesCreating, RefusesUnknownState / N-state-creating, N-state-unknown | U1 | U1 | U1 | U1 | U1 | U1 |
| statelookup (state.go:104) | U2a | M AttachErrorsOnStateReadFailure / N-state-read-absent (value ignored; error propagates) | B39 parsed instance; M at LOOK | U2a | U2a | U2a | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M RefusesBadIdentity, RefusesForeignDocument / N-state-lookup-identity, N-state-foreign | U1 | U1 | U1 | U1 | U1 |
| stateopen (state.go:60) | U2a | U2a | U2a | U2a | U2a | U2a | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B22 RefusesEmptyRoot; singleton | U1 | U1 | U1 | U1 | U1 | U1 |
| createcount (bodies.go:64) | B39 valid bodies; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M MemberSet / N-bodies-create-count |
| attachcount (bodies.go:152) | U1 | B39 valid bodies; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M MemberSet / N-bodies-attach-count |
| termcount (bodies.go:241) | U1 | U1 | U1 | U1 | B39 valid bodies; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M Violations / N-bodies-terminate-count |
| restorecount (bodies.go:304) | U1 | U1 | U1 | U1 | U1 | B39 valid bodies; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M Violations / N-bodies-restore-count |
| transportvocab (bodies.go:340) | B39 valid transports; M at UNT | B39 valid; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M Create/AttachMembers / N-bodies-transport |
| epochzero (bodies.go:265) | U1 | U1 | U1 | U1 | B39 valid epochs; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M Violations/epoch-zero / N-bodies-epoch-zero |
| entrypoint (bodies.go:110) | B39 valid entrypoints; M at UNT | B39 valid; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M StringRefuses, shapes pinned / N-bodies-entrypoint-string |
| absentmemory (status.go:296) | U1 | U1 | M Contradictions/quiescing-memory / N-status-absent-quiescing | U1 | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M AbsentProbe/memory-quiescing / same row | B39 live shapes; M at CLS,EXS | U1 | U1 | U1 |
| presentmemory (status.go:314) | U1 | U1 | M Contradictions/stopped-memory / N-status-present-stopped | U1 | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M PresentProbe/memory-stopped / same row | M MemoryReconciles / same row | U1 | U1 | U1 |
| attachable (status.go:334) | U1 | U1 | M Contradictions/incapable-attach / N-status-attachable | U1 | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M PresentProbe/no-attach-cap / same row | B39 quiescing never attachable; M at CLS,EXS | U1 | U1 | U1 |
| cutrow (status.go:160) | U1 | U1 | B23 count-less-row pins code; engine normalizes detail | U1 | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U2a classify takes parsed rows | M Unknowns / N-status-cutrow-count | U1 | U1 | M CutRowShapes / same row |
| emptypanes (status.go:96) | U1 | U1 | B23 empty-rows pins code; engine maps uncoded | U1 | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M Unknowns/empty-rows / N-status-empty-panes | U1 | U1 | U1 |
| triple (status.go:344) | U1 | U1 | M Contradictions/unrequested-present / N-provider-triple-unrequested | U1 | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 requested shapes; M at UNT,EXS | U1 | U1 | M TripleRule / same row |
| dialstale (probe.go:60) | U2a | U2a | U2a | U2a | U2a | U2a | B39 injected outcomes; M at DIAL | U1 | M LiveStaleUnknown, MissingSocketIsStale / N-probe-dial-refused, N-probe-dial-enoent | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| probedeps (probe.go:100) | U2a | U2a | U2a | U2a | U2a | U2a | M RefusesWithoutDependencies / N-probe-deps-dialer, N-probe-deps-admit | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| probeunknown (probe.go:112) | U2a | U2a | U2a | U2a | U2a | U2a | M MapsDialOutcomes/unknown / N-probe-unknown-passthrough | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| spawnpin (probe.go:182) | U2a | U2a | U2a | U2a | U2a | U2a | U1 | M PinAndCustody / N-spawn-pin | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| spawnshape (probe.go:182) | U2a | U2a | U2a | U2a | U2a | U2a | U1 | M PinAndCustody / N-spawn-shape | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| spawnexit (probe.go:196) | U2a | U2a | U2a | U2a | U2a | U2a | U1 | M MapsRunnerOutcome / N-spawn-exit | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| spawnnil (probe.go:177) | U2a | U2a | U2a | U2a | U2a | U2a | U1 | B22 RefusesWithoutRunner; singleton | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| unlinkkind (probe.go:206) | U2a | U2a | U2a | U2a | U2a | U2a | U1 | U3 custody refuses non-sockets first | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| reconcilenil (probe.go:238) | U2a | U2a | U2a | U2a | U2a | U2a | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B22 FailsClosedWithoutVerifier; singleton |
| sunpath (bind.go:28) | M AtLimitSocket/create / N-bind-length-boundary | M AtLimitSocket/attach / same row | M AtLimitSocket/status / same row | M AtLimitSocket/3 engine ops / same row | M AtLimitSocket/terminate-stale / same row | M AtLimitSocket/restore / same row | M EnforcesLengthAndCustody / same row | B39 socket arg short; M at LEN,EXS,PRB | U1 | U1 | U1 | U1 | M Bounds, CountsBytes / same row | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| placement (bind.go:44) | U2a socket derived from RuntimeDir; entry input never reaches placement | U2a socket derived from RuntimeDir; entry input never reaches placement | U2a socket derived from RuntimeDir; entry input never reaches placement | U2a socket derived from RuntimeDir; entry input never reaches placement | U2a socket derived from RuntimeDir; entry input never reaches placement | U2a socket derived from RuntimeDir; entry input never reaches placement | M EnforcesLengthAndCustody / N-bind-placement | B39 pinned vectors; M at CUS,PRB | U1 | U1 | U1 | U1 | U1 | M RefusesMisplacedSocket / same row | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| leafdelegate (bind.go:52) | B32 UnsafeSocket/create; propagation pinned | B32 UnsafeSocket/attach; pinned | B32 UnsafeSocket/status; pinned | B32 UnsafeSocket/3 ops; pinned | B32 UnsafeSocket/terminate; pinned | B32 UnsafeSocket/restore; pinned | B32 EnforcesLengthAndCustody; pinned | B32 RefusesAbsentLeaf; pinned | U1 | U1 | U1 | U1 | U1 | M PropagatesLeafRefusal / N-bind-wiring-verify | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| leafwiring (bind.go:44,52) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U1 | U1 | U1 | U1 | U1 | M AdmitsCompliant / N-bind-wiring-placement, N-bind-wiring-verify | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| rootwrite (bind.go:96) | M CreateRefusesWritableRoot / N-custody-entry-root-create; EachOperationRefusesWritableRoot/create / N-bind-root-writable | M AttachRefusesWritableRootValidBody / N-attach-entry-root-custody; EachOperationRefusesWritableRoot/attach / N-bind-root-writable | M CustodyEntryRoots/status/root / N-custody-entry-root-status | M EachOperationRefusesWritableRoot/quiesce-input,boundary,stop / N-bind-root-writable | M EachOperationRefusesWritableRoot/terminate-stale / N-bind-root-writable | M CustodyEntryRoots/restore/root / N-custody-entry-root-restore | M ServerProberRefusesWritableRoot / N-prober-root-custody | M ServerSpawnerRefusesWritableRoot / N-spawn-root-custody | U1 | U1 | U1 | U1 | U1 | M RefusesUnsafeRoot / N-bind-root-writable | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| rootowner (bind.go:96) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U1 | U1 | U1 | U1 | U1 | B31 ForeignRoot stages leaf-first; arm untested+singleton | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| ancestorwrite (bind.go:116) | M EachOperationRefusesWritableAncestor/create / N-bind-ancestor-writable | M EachOperationRefusesWritableAncestor/attach / N-bind-ancestor-writable | M CustodyEntryRoots/status/ancestor / N-custody-entry-ancestor-status | M EachOperationRefusesWritableAncestor/quiesce-input,boundary,stop / N-bind-ancestor-writable | M EachOperationRefusesWritableAncestor/terminate-stale / N-bind-ancestor-writable | M CustodyEntryRoots/restore/ancestor / N-custody-entry-ancestor-restore | B39 staged; M at CUS + 6 EX | B39 staged; M at CUS + 6 EX | U1 | U1 | U1 | U1 | U1 | M RefusesWritableAncestor / N-bind-ancestor-writable | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| ancestorkind (bind.go:116) | B39 staged compliant; M at CUS,PRB,SPW | B39 staged; M at CUS,PRB,SPW | B39 staged; M at CUS,PRB,SPW | B39 staged; M at CUS,PRB,SPW | B39 staged; M at CUS,PRB,SPW | B39 staged; M at CUS,PRB,SPW | M SymlinkRefusesAtProbeAndSpawn/probe, RefusesIntermediateSymlink / N-custody-alias-skip | M SymlinkRefusesAtProbeAndSpawn/spawn / same row | U1 | U1 | U1 | U1 | U1 | M RefusesIntermediateSymlink, AncestorKindArm / same row | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| socketkind (bind.go:164) | M EachOperationRefusesNonSocketKind/create / N-bind-socket-kind | M EachOperationRefusesNonSocketKind/attach / N-bind-socket-kind | M EachOperationRefusesNonSocketKind/status / N-bind-socket-kind | M EachOperationRefusesNonSocketKind/quiesce-input,boundary,stop / N-bind-socket-kind | M EachOperationRefusesNonSocketKind/terminate-stale / N-bind-socket-kind | M EachOperationRefusesNonSocketKind/restore / N-bind-socket-kind | U2a staged; M at CUS + 6 EX | B39 absent sockets; M at CUS + 6 EX | U1 | U1 | U1 | U1 | U1 | M RefusesNonSocketFile / N-bind-socket-kind | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| backendid (lifecycle.go:200) | M RefusesForeignBackend/create / N-lifecycle-backend-identity | M RefusesForeignBackend/attach / same row | M RefusesForeignBackend/status / same row | M RefusesForeignBackend/3 engine ops / same row | M RefusesForeignBackend/terminate-stale / same row | M RefusesForeignBackend/restore / same row | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| agreement (lifecycle.go:255) | M BindingAgreement / N-lifecycle-agreement-session, -instance, -backend, -generation | U2a create-only gate | U2a | U2a | U2a | U2a | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| rowvocab (lifecycle.go:270) | B39 in-row codes; M at UNT | B39 in-row; M at UNT | B39 in-row; M at UNT | B39 in-row; M at UNT | B39 in-row; M at UNT | B39 in-row; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M NormalizesOutsideSet, CoversEightRows / N-rowvocab-pass |
| enginemembers (lifecycle.go:380) | U1 | U1 | U1 | M Quiesce, Boundary, Stop / N-engineop-quiesce-generation, N-engineop-boundary-evidence, N-engineop-stop-process, N-engineop-stop-store | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| waitlesser (lifecycle.go:410) | U1 | U1 | U1 | B39 scripted waits; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M BoundWaitTakesLesser / N-wait-bound-lesser |
| stoplesser (lifecycle.go:421) | U1 | U1 | U1 | B39 staged deadlines; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 staged; M at UNT | U1 | M LesserDeadlineTakesLesser / N-stop-deadline-lesser |
| relay (lifecycle.go:300) | B22 CreateRefusals/relay; singleton | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| descpresence (lifecycle.go:330) | B22 Interactive+Headless; boolean fork | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| attachcap (ops.go:100) | U1 | M wrong-transport-capability, MeshWithoutCapability / N-attach-capability-local, N-attach-capability-mesh | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| attachdeadline (ops.go:72) | U1 | M RefusesDeadline / D-attach-entry-deadline-shadowed survived-supplementary (entry fail-fast; enforced boundary at post-lock + wait); AttachRefusesDeadlineExpiredDuringAdmission, AttachRefusesDeadlineOnTheInstantDuringAdmission / N-attach-postlock-deadline; AttachWaitBoundedByOperationDeadline, AttachWaitCancelledReturnsPromptly / N-attach-wait-deadline shared | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| attestation (ops.go:113) | U1 | M fresh-decoy / N-attach-attestation-decoy; stale member pinned | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| attachrecord (ops.go:140) | U1 | M Attach / N-attach-record | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| attachdesc (ops.go:46) | U1 | M Attach / N-attach-descriptor-order | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| createdesc (ops.go:56) | M CreateInteractive / N-create-descriptor-order | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| serveadmitnil (ops.go:108) | U1 | B22 no-admission; singleton | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| attacheffect (ops.go:119) | U1 | B26 unstaged failure mapping | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| attachbuild (ops.go:128) | U1 | U3 body parse precedes; parsed identities never refuse | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| authkind (ops.go:196,352,520) | U1 | U1 | U1 | U1 | M ExecuteEntryRefusesWrongAuthorizationKind/terminate / N-entry-authkind-terminate; ExecuteTerminateRefusesAuthorizationExpiredAfterReceipt / N-lifecycle-loop-authorization shared; D-lifecycle-terminate-authkind supplementary wiring | M ExecuteEntryRefusesWrongAuthorizationKind/restore / N-entry-authkind-restore; ExecuteRestoreRefusesAuthorizationExpiredAfterReceipt / N-lifecycle-loop-authorization shared; D-lifecycle-restore-authkind supplementary wiring | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| generation (ops.go:205,361,524) | U1 | M AttachRefusesGenerationRotatedDuringLockWait / N-attach-postlock-generation | U1 | U1 | M EntryRefusesStaleGeneration/terminate / N-entry-generation | M EntryRefusesStaleGeneration/restore, RefusesGenerationDrift / N-entry-generation, N-lifecycle-loop-generation | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| deadline (ops.go:72,226,374,510) | U1 | M RefusesDeadline/boundary / D-attach-entry-deadline-shadowed survived-supplementary (entry fail-fast; enforced boundary at post-lock + wait); AttachRefusesDeadlineExpiredDuringAdmission, AttachRefusesDeadlineOnTheInstantDuringAdmission / N-attach-postlock-deadline; AttachWaitBoundedByOperationDeadline, AttachWaitCancelledReturnsPromptly / N-attach-wait-deadline shared | U1 | U1 | M RefusesOnDeadline / N-lifecycle-entry-deadline-terminate | M RefusesOnDeadline, RefusesOnDeadlineEffect / N-lifecycle-entry-deadline-restore, N-lifecycle-loop-deadline | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| keymaterial (ops.go:176,338) | U1 | U1 | U1 | U1 | M WrongIdempotencyKey/terminate / N-key-material-terminate | M WrongIdempotencyKey/restore / N-key-material-restore | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| capdual (ops.go:240) | U1 | U1 | U1 | U1 | M missing-capability, RefusesProviderOnlyCapability / N-lifecycle-capability-provider, N-lifecycle-capability-stale | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| restorecap (ops.go:345) | U1 | U1 | U1 | U1 | U1 | M RefusesWithoutRebootCapability / N-lifecycle-restore-capability | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| fencingmap (ops.go:262) | U1 | U1 | U1 | U1 | M no-force, malformed-presented / N-lifecycle-fencing-force, N-fencing-invalid-reroute; other members pinned | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| target (ops.go:270) | U1 | U1 | U1 | U1 | M target-mismatch, TargetLeaseIdentity, TargetSessionBinds / N-lifecycle-target-epoch, N-lifecycle-target-lease, N-lifecycle-target-session | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| winner (ops.go:285) | U1 | U1 | U1 | U1 | M winner-rotated, winner-lease-identity / N-lifecycle-winner-epoch, N-lifecycle-winner-lease; session arm U3 fenced | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| mismatch (ops.go:368) | U1 | U1 | U1 | U1 | U1 | M RestoreMismatch/wrong-digest / N-lifecycle-restore-mismatch | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| bindmap (ops.go:410) | U1 | U1 | U1 | U1 | M RedriveMismatch / N-bind-mismatch-map | B39 shared code; unstaged redrive on restore; M at EXT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| bindfound (ops.go:424) | U1 | U1 | U1 | U1 | M BindHookFailure / N-bind-hook-found | B39 shared code; unstaged hook failure on restore; M at EXT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| bindmiss (ops.go:424) | U1 | U1 | U1 | U1 | B25 clean-miss unstageable; hooks fire post-commit | B25 clean-miss unstageable | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| completedread (ops.go:440) | U1 | U1 | U1 | U1 | B24 read failure unstaged; no seam | B24 read failure unstaged | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| evidence (ops.go:548) | U1 | U1 | U1 | U1 | B39 real digests; M at UNT | B39 real digests; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M RefusesForgedEvidence / N-lifecycle-evidence-digest |
| record (ops.go:505,592) | U1 | U2a attach records at its own site | U1 | U1 | M Terminate / N-lifecycle-record-success; failure arm untested here | M Restore / same row; failure arm untested here | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M RefusesForgedEvidence / N-lifecycle-record-failure |
| resume (ops.go:463) | U1 | U1 | U1 | U1 | B39 source-proven resumes; M at EXR | M RefusesWhenStatusProvesOtherwise / N-lifecycle-resume-state; error arm subsumed | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| replay (ops.go:407) | U1 | U1 | U1 | U1 | M TamperedImage, CorruptImage / N-lifecycle-replay-session, N-replay-corrupt-image; other members pinned | B39 clean replays; M at EXT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| completearms (ops.go:580) | U1 | U1 | U1 | U1 | U2a Complete runs post-Bind once; never fails here | U2a Complete runs post-Bind once | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| readonly (exec.go:attach) | U1 | M ReadOnlyAttachIsReadOnly / N-attach-readonly-vector | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M AttachReadOnlyVector / same row | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| inputflag (exec.go:attach) | U1 | B39 stated authorizations; M at CMD | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M AttachRequiresInputFlag / N-attach-input-flag | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| detachvec (exec.go:detach) | U2a quiesce-only vector | U2a quiesce-only | U2a quiesce-only | B39 subcommand-only; M at CMD | U2a quiesce-only | U2a quiesce-only | U1 | U1 | U1 | M Vectors / N-cmdvector-detach | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 subcommand-only; M at CMD | U1 | U1 |
| boundarykind (backend.go:observeBoundary) | U2a boundary-only gate | U2a boundary-only | U2a boundary-only | M BoundaryBindsProofKind / N-boundary-proof-kind | U2a boundary-only | U2a boundary-only | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 kind-echoed; M at EXE | U1 | U1 |
| boundexit (backend.go:observeBoundary) | U2a boundary-only gate | U2a boundary-only | U2a boundary-only | M BoundaryRefusesFailedWait / N-boundary-exit-refused | U2a boundary-only | U2a boundary-only | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M BoundaryRefusedExit / same row | U1 | U1 |
| quiescedetach (backend.go:closeInput) | U2a quiesce-only gate | U2a quiesce-only | U2a quiesce-only | M BarrierSequence / N-quiesce-detach-step | U2a quiesce-only | U2a quiesce-only | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M InputClosed / same row | U1 | U1 |
| nosendkeys (backend.go:closeInput) | U2a quiesce-only gate | U2a quiesce-only | U2a quiesce-only | M EmitsNoProviderInput / N-quiesce-sendkeys-absent | U2a quiesce-only | U2a quiesce-only | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 shared sequence; M at EXE | U1 | U1 |
| closeexit (backend.go:closeInput) | U2a quiesce-only gate | U2a quiesce-only | U2a quiesce-only | M RefusesFailedClose / N-quiesce-close-exit | U2a quiesce-only | U2a quiesce-only | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 clean closes; M at EXE | U1 | U1 |
| quiescegate (ops.go:checkAttachQuiesced) | U1 | M RefusesInputWhenQuiescing / N-quiesce-attach-gate (scoping, key, decode, and empty arms are separate rows below) | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| probemarker (backend.go:classifyAbsence) | U2a no has/list probes | U2a no probes | M StatusUnknownProbe / N-probe-unmarked-exit | M StopUnmarkedExit / same row | M TerminateUnknownConfirm / same row | U2a no probes | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 shared code; M at EXS | B39 shared code; M at EXE,EXT | U1 | U1 |
| probeneg (backend.go:classifyAbsence) | U2a no has/list probes | U2a no probes | B39 non-negative scripts; M at EXE | M StopUnknownProbe / N-probe-negative-exit | B39 non-negative scripts; M at EXE | U2a no probes | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 non-negative scripts; M at EXE | U1 | U1 |
| identity (bind.go:checkCustodyRoot) | B39 matching identity; M at CUS | B39 matching; M at CUS | B39 matching; M at CUS | B39 matching; M at CUS | B39 matching; M at CUS | B39 matching; M at CUS | B39 matching; M at CUS | B39 matching; M at CUS | U1 | U1 | U1 | U1 | U1 | M IdentityMismatchRefuses / N-custody-identity-bind | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| bindcontract (ops.go:restoreBinding) | U1 | U1 | U1 | U1 | U1 | M ReturnsBinding, Idempotent / N-restore-binding-omit | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| restoreagree (ops.go:restoreBinding) | U1 | U1 | U1 | U1 | U1 | M RefusesSwappedBindingDoc / N-restore-agreement-session | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| replaytime (lifecycle.go:report) | U1 | U1 | U1 | M QuiesceReplayKeepsTimestamps, BoundaryReplayKeepsProofAndTime (+Idempotent) / N-replay-timestamp-quiesce, N-replay-timestamp-boundary | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| outcomekey (state.go:RecordOutcome) | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M OutcomeRoundTrip / N-outcome-key-record | M LookupOutcomeRefusesForgedKey / N-outcome-key-forged | U1 | U1 | U1 | U1 | U1 |
| binddocid (state.go:RecordBindingDoc) | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M BindingDocRoundTrip / N-binddoc-session-identity | M BindingDocRoundTrip / N-binddoc-lookup-identity | U1 | U1 | U1 | U1 | U1 |
| binddocshape (state.go:RecordBindingDoc) | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M BindingDocRoundTrip / N-binddoc-document-shape | U1 | U1 | U1 | U1 | U1 | U1 |
| docpersist (backend.go:persistBinding) | B39 doc-kept; M at EXR | U1 | U1 | U1 | U1 | M CreatePersistsBindingDocForRestore / N-create-doc-persist | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 unstaged bindingDoc; M at EXR | U1 | U1 |
| pollbound (backend.go:460) | U1 | U1 | U1 | M ExecuteStopPollHonorsOperationDeadline / N-probe-deadline shared | B39 terminate tail reaches the shared poll; M at EXE | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 direct staging unmeasured here; M at EXE | U1 | U1 |
| attachreadonly (ops.go:134) | U2a attach-only gate | M ReadOnlyAttachPreservesQuiescedInput / N-attach-readonly-record | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| quiescereport (lifecycle.go:reportInputClosedAt) | U2a quiesce-only gate | U2a quiesce-only gate | U2a quiesce-only gate | M CorruptOutcomeRefusesIntegrity, EmptyRecordedTimeRefusesIntegrity, OutcomeRecordCrashConverges / N-quiesce-report-corrupt, N-quiesce-report-empty, N-quiesce-report-record (quiesce leg) | U2a quiesce-only gate | U2a quiesce-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| boundaryreport (lifecycle.go:reportBoundaryObservedAt) | U2a boundary-only gate | U2a boundary-only gate | U2a boundary-only gate | M CorruptOutcomeRefusesIntegrity, EmptyRecordedTimeRefusesIntegrity, RecordCrashRefusesIntegrity / N-boundary-report-corrupt, N-boundary-report-empty, N-boundary-report-record (boundary leg) | U2a boundary-only gate | U2a boundary-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| statusunbound (status.go:ObserveStatus) | U2a status-only gate | U2a status-only gate | M ExactUnboundSessionRefusesUnknown / N-status-exact-unbound | U2a status-only gate | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| statusdoc (status.go:statusBindingDoc) | U2a status-only gate | U2a status-only gate | M BindingTamperRefusesIntegrity (missing, corrupt, foreign-session, foreign-instance, digest-mismatch) / N-status-doc-missing, N-status-parse-admit, N-status-agree-session, N-status-agree-instance | U2a status-only gate | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| statusmatch (status.go:bindingMatchesQuery) | U2a status-only gate | U2a status-only gate | M NonMatchEachMember (instance, implementation, generation) / N-status-match-instance, N-status-match-impl, N-status-match-generation | U2a status-only gate | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M ExactBackendDriftNonMatches, ExactProtocolDriftNonMatches / N-status-match-backend, N-status-match-proto | U1 | U1 | U1 | U1 | U1 |
| statusnegative (status.go:probeAttached) | U2a status-only gate | U2a status-only gate | M RefusesNegativeAttachedCount / N-status-negative-attached; MalformedAttachedCountIsUnknown / N-status-attached-parse | U2a status-only gate | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| prioragree (ops.go:readRestorePrior) | U2a restore-only gate | U2a restore-only gate | U2a restore-only gate | U2a restore-only gate | U2a restore-only gate | M RefusesSubstitutedInstance, RefusesSubstitutedSession, RefusesSubstitutedBackend, RefusesSwappedBindingDoc / N-restore-prior-instance, N-restore-prior-backend, N-restore-agreement-session | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| succession (ops.go:restoreBinding) | U2a restore-only gate | U2a restore-only gate | U2a restore-only gate | U2a restore-only gate | U2a restore-only gate | M AcrossServerGeneration, SuccessorRetryConverges, PriorValidationPrecedesEffects / N-restore-succession-gen, N-restore-mint-generation, N-restore-successor-persist | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| execdeps (lifecycle.go:Execute) | B36 nil-lifecycle pinned; singleton arms | B36 nil-lifecycle pinned; singleton arms | B36 nil-lifecycle pinned; singleton arms | B36 nil-lifecycle pinned; singleton arms | B36 nil-lifecycle pinned; singleton arms | B36 nil-lifecycle pinned; singleton arms | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| quiescerecord (lifecycle.go:executeEngineOp) | U2a quiesce-only gate | U2a quiesce-only gate | U2a quiesce-only gate | M QuiesceStateWriteFailureRefusesWiredAttach / N-quiesce-state-record | U2a quiesce-only gate | U2a quiesce-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| boundaryrecord (lifecycle.go:executeEngineOp) | U2a boundary-only gate | U2a boundary-only gate | U2a boundary-only gate | M BoundaryBarrierSurvivesStateWriteFailure / N-boundary-state-record | U2a boundary-only gate | U2a boundary-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| barrierquiesce (state.go:QuiesceBarrierProven) | U2a attach-only gate | M QuiesceStateWriteFailureRefusesWiredAttach / N-attach-barrier-quiesce-recovery | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| barrierboundary (state.go:QuiesceBarrierProven) | U2a attach-only gate | M BoundaryBarrierSurvivesStateWriteFailure / N-attach-barrier-boundary-recovery | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| barrierdecode (state.go:QuiesceBarrierProven) | U2a attach-only gate | M AttachErrorsOnCorruptBarrierScan / N-attach-barrier-corrupt-scan | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| barrierempty (state.go:QuiesceBarrierProven) | U2a attach-only gate | M AttachErrorsOnEmptyBarrierTime/quiesce,boundary / N-attach-barrier-empty-quiesce, N-attach-barrier-empty-boundary | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| barrierincarnation (state.go:QuiesceBarrierProven) | U2a attach-only gate | M QuiesceStopRestoreReopensBarrier / N-attach-barrier-incarnation | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| barrierkey (state.go:QuiesceBarrierProven) | U2a attach-only gate | M AttachErrorsOnEmptyBarrierKey / N-attach-barrier-empty-key | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| attachhealth (ops.go:checkAttachQuiesced) | U1 | M AttachErrorsOnStateReadFailure / N-attach-health-check | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| failurerecord (lifecycle.go:recordFailure) | B39 first-echo only; M at EXE,UNT | U1 | U1 | M FailedStopCannotEraseQuiescence / N-failure-preserves-record | B39 echo-or-unknown; M at UNT | B39 echo-or-unknown; M at EXE,UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M RunLifecycleEffectsRefusesForgedEvidence / N-lifecycle-record-failure |
| createincarnation (lifecycle.go:executeCreate) | M CreateIncarnationWriteFailureFailsClosed / N-create-incarnation-record, N-create-replay-skip | U2a create-only gate | U2a create-only gate | U2a create-only gate | U2a create-only gate | U2a create-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| restoreincarnation (ops.go:runLifecycleEffects) | U2a restore-only gate | U2a restore-only gate | U2a restore-only gate | U2a restore-only gate | M ExecuteTerminate / N-terminate-no-rotation | M RestoreIncarnationWriteFailureFailsClosed / N-restore-incarnation-record; replay skip B41 structural with D-restore-replay-rotation | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| incarnationrecord (state.go:RecordIncarnation) | B39 adopted rotations; M at REC | U2a | U2a | U2a | U2a | B39 adopted rotations; M at REC | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M IncarnationRoundTrip / N-incarnation-record-identity, N-incarnation-record-key | U1 | U1 | U1 | U1 | U1 | U1 |
| incarnationlookup (state.go:LookupIncarnation) | U2a | M AttachErrorsOnIncarnationReadFailure / N-incarnation-read-absent | U2a | B39 closure binds current; M at LOOK | U2a | U2a | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M IncarnationRoundTrip / N-incarnation-lookup-identity, N-incarnation-foreign, N-incarnation-empty-key | U1 | U1 | U1 | U1 | U1 |
| attachedexit (status.go:probeAttached) | U2a status-only gate | U2a status-only gate | M AttachedExitIsUnknown / N-status-attached-exit | U2a status-only gate | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| foreignrow (status.go:probeAttached) | U2a status-only gate | U2a status-only gate | M ForeignAttachedRowIsUnknown / N-status-foreign-row | U2a status-only gate | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| panesexit (status.go:probePanes) | U2a status-only gate | U2a status-only gate | M UnknownPanesExitIsUnknown / N-status-panes-unknown-exit | U2a status-only gate | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| argvnul (exec.go:OSRunner.Run) | U2a | U2a | U2a | U2a | U2a | U2a | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M RefusesNulArgument / N-osrunner-argv-nul | U1 |
| recheck (ops.go:executeAttach) | U2a attach-only gate | M AttachRefusesAfterQuiescenceWhenAdmittedLate, AttachQuiesceOrderingOverlapQuiesce / N-attach-barrier-recheck | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| ordering (lifecycle.go:barrierMu, ops.go:executeAttach) | U2a no barrier commit on this path | M AttachQuiesceOrderingOverlapQuiesce / N-ordering-lock-attach | U2a no barrier commit on this path | M AttachQuiesceOrderingOverlapQuiesce, AttachQuiesceOrderingOverlapBoundary / N-ordering-lock-quiesce, N-ordering-lock-boundary | U2a no barrier commit on this path | U2a no barrier commit on this path | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| barrierread (state.go:QuiesceBarrierProven) | U2a attach-only gate | M AttachErrorsOnOutcomesDirReadFailure, AttachErrorsOnOutcomeFileReadFailure / N-outcomes-dir-read-absent, N-outcome-file-read-skip | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| effectrecheck (lifecycle.go:727; backend.go:closeInput,confirmClosed) | U2a create effects are single commands with no post-wait boundary; recheck never configured | U2a attach commits via the store with no post-wait destructive command; recheck never configured | U2a status has no side effects | M ExecuteStopRefusesStaleFactsAfterPoll, ExecuteStopEscalatesAfterGracefulTimeoutAlone, ExecuteQuiesceRefusesStaleFactsBetweenCommands / N-stop-escalation-generation, N-stop-escalation-expiry, N-stop-escalation-deadline shared 3 arms + N-stop-escalation-skip, N-quiesce-midsequence-skip per-site | U2a terminate tail issues no command after its poll; recheck never configured; the no-post-wait-command probe pins it | U2a restore effects are single commands with no post-wait boundary; recheck never configured | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B22 BackendPostWaitBoundaryRefusesWithoutRecheck; singleton | U1 | U1 |
| overlapmulti (ops.go:299; termbind/overlap.go:36) | U2a attach-only gate | M AttachOverlapRequiresMultiAttach / N-attach-overlap-multi | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| overlapinput (ops.go:299; termbind/overlap.go:36) | U2a attach-only gate | M AttachOverlapInputRequiresMultipleInputClients / N-attach-overlap-input | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| waitbound (lifecycle.go:750,776; ops.go:259) | U2a create takes no barrier and runs no admission wait | M AttachWaitBoundedByOperationDeadline/admission,barrier / N-attach-wait-deadline shared; AttachWaitCancelledReturnsPromptly stays green (cancellation companion) | U2a status takes no barrier and runs no admission wait | M QuiesceBarrierWaitBoundedByOperationDeadline, BoundaryCommitWaitBoundedByOperationDeadline / N-attach-wait-deadline shared | U2a terminate takes no barrier and runs no admission wait | U2a restore takes no barrier and runs no admission wait | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| peershape (termbind/overlap.go:36) | U2a attach-only gate | M AttachPeerDirectoryFailsClosed / N-attach-peers-dir-skip; AttachPeerFilenameMismatchFailsClosed / N-attach-peers-filename-skip | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| overlapreplay (ops.go:299) | U2a attach-only gate | M AttachSameClientRetryWithPeerPresent / N-attach-overlap-replay-unproven | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| confirmbound (backend.go:540) | U2a terminate-only gate | U2a terminate-only gate | U2a terminate-only gate | U2a terminate-only gate | M ExecuteTerminateConfirmHonorsOperationDeadline / N-probe-deadline shared | U2a terminate-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 direct staging unmeasured here; M at EXT | U1 | U1 |
| statusbound (status.go:28) | U2a status-only gate | U2a status-only gate | M ExecuteStatusProbeHonorsOperationDeadline / N-probe-deadline shared | U2a status-only gate | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| escalationbound (backend.go:431) | U2a stop-only gate | U2a stop-only gate | U2a stop-only gate | M ExecuteStopEscalationSucceedsWithContextHonoringRunner / N-stop-escalation-budget | U2a stop-only gate | U2a stop-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 direct staging unmeasured here; M at EXE | U1 | U1 |

Symmetry (every gate on two or more sides, with admitting-row
counts per side; "shared" marks one row killing through several
entries, attribution verified per entry from the raw logs):
directive CMD 1 / PDIR 1 shared; scrub OSR 2 / UNT 2 shared; cmdident
CMD 3 members; cmdvector EXC 1 / EXA 1 / EXE 1 / EXR 1 / CMD 4;
directives 6 EX + DIRV shared 1; effectvocab/exitrefused/transport
single-sided at PFX; reconfirm/escalation/persistmap/attachmap
single-sided at PFX; boundarywait EXE 1 / PFX 1 shared; staterecord
REC 2 members; statelookup LOOK 2 members; entrypoint/transportvocab
single-sided at UNT; absentmemory EXS 1 / CLS 1 shared; presentmemory
EXS 1 / CLS 1 / OBSP 1 shared; attachable EXS 1 / CLS 1 shared; cutrow
OBSP 1 / UNT 1 shared (EXS B23); triple EXS 1 / UNT 1 shared; dialstale
DIAL 2 members; probedeps PRB 2 members; sunpath LEN 1 / 6 EX 1 /
PRB 1 shared; placement CUS 1 / PRB 1 shared; leafdelegate CUS 1 (B32
propagation elsewhere); leafwiring CUS 2 sites; rootwrite CUS 1 /
EXC 1 / EXA 1 / EXS 1 / EXR 1 / SPW 1 / shared EachOperation row
(EXA, EXE, EXT);
ancestorwrite CUS 1 / EXS 1 / EXR 1 / shared EachOperation row
(EXC, EXA, EXE, EXT); socketkind CUS 1 / shared EachOperation row
(6 EX); ancestorkind CUS 1 /
PRB 1 / SPW 1 shared; backendid 6 EX shared 1; agreement EXC 4
members; enginemembers EXE 4 members; attachcap EXA 2 members;
attestation EXA 1 + pinned member; attachdeadline EXA 2 arms;
authkind EXT 2 / EXR 2 (entry 1 / shared loop 1 each; D-loop wiring
supplementary, uncounted); generation EXA 1 / EXT 1 / EXR 2; deadline
EXA 2 / EXT 1 / EXR 2; keymaterial EXT 1 / EXR
1; capdual EXT 2 members; fencingmap EXT 2 + 3 pinned; target EXT 3
members; winner EXT 2 members (session arm U3: fencing+target
precede); record EXT 1 / EXR 1 shared + UNT 1; replay EXT 2 + 6
pinned; bindmap/bindfound single-sided at EXT; readonly EXA 1 / CMD 1
shared; detachvec CMD 1; boundarykind/boundexit EXE 1 each + PFX
0/1; quiescedetach EXE 1 / PFX 1 shared; nosendkeys/closeexit EXE 1
each; probemarker EXS 1 / EXE 1 / EXT 1 shared; probeneg EXE 1;
identity CUS 1; bindcontract/restoreagree EXR 1 each; replaytime EXE
2; after-restore outcomes in Table E (launch/reattach/offer/park/
refuse + refresh audit); outcomekey REC 1 / LOOK 1; binddocid REC
1 / LOOK 1; binddocshape REC 1; docpersist EXR 1; attachreadonly
EXA 1; quiescereport/boundaryreport EXE 3 rows each
(corrupt/empty/crash legs); statusunbound/statusdoc/statusmatch/
statusnegative EXS 1 each + statusmatch LOOK 1; prioragree EXR 4
tests, 3 rows; succession EXR 3 members; execdeps 6 EX pinned
(B36); Table B verify 9 new-entry sides shared 1, server EXA 1 (its
other sides are first-leaf symmetry). Revision-4 gates:
quiescerecord/boundaryrecord single-sided at EXE;
barrierquiesce/barrierboundary/barrierdecode/barrierempty
single-sided at EXA (same entry as the quiescegate memory arm, whose
row points at them); attachedexit/foreignrow/panesexit single
measured side at EXS (EXT/EXR B39 via the resume reconcile);
argvnul single-sided at OSR (mirrors osrun: no committed Execute
composition injects OSRunner). Revision-5 gates:
barrierincarnation/barrierkey/attachhealth single-sided at EXA;
failurerecord EXE 1 / UNT 1 (different arms: never-overwrite vs
call value); createincarnation EXC 2 rows; restoreincarnation EXR
1 + structural D row / EXT 1; incarnationrecord REC 2 members;
incarnationlookup LOOK 3 members. Rotation symmetry:
createincarnation EXC 1 narrowing row / restoreincarnation EXR 1
narrowing row (symmetric); rootwrite gains PRB 1 (CUS 1 / EXC 1 /
EXS 1 / EXR 1 / PRB 1 / SPW 1 / shared EachOperation row);
statelookup gains EXA 1 alongside LOOK 2 (different arms: absent
vs grammar); statusnegative EXS 2 rows (negative + parse legs).
Revision-6 gates: recheck single-sided at EXA; ordering EXA 1 /
EXE 2 with the three-sided symmetry attach 1 / quiesce 1 /
boundary 1; barrierread EXA 2 arms with the symmetry
directory-read 1 / file-read 1; incarnationlookup gains EXA 1
alongside LOOK 3 (different arms: read-error vs grammar);
rootwrite EXA gains the entry row alongside the shared row (the
valid-body attach is served when the gate is weakened for attach
alone). Revision-8 gates: effectrecheck single-column at EXE with
the two-site symmetry stop 4 / quiesce 4 (3 shared arms + 1
per-site skip row each); overlapmulti/overlapinput single-sided at
EXA; waitbound EXA 1 / EXE 1 shared (4 killers across
attach-admission, attach-barrier, quiesce-span, and
boundary-commit; the cancellation companion stays green);
attachdeadline/deadline EXA gain the shared wait row beside the
survived-supplementary entry row. Revision-9 gates:
peershape/overlapreplay single-sided at EXA; pollbound EXE 1
(B39 at PFX and EXT: the terminate tail reaches the shared
poll); confirmbound single measured side at EXT (B39 at PFX);
statusbound single measured side at EXS (EXT/EXR B39 via the
resume reconcile); the shared probe row kills through the stop,
terminate-confirm, and status deadline tests with isolated
per-test attribution. Revision-10 gate: escalationbound single
measured side at EXE (B39 at PFX); the peershape EXA cell gains
the filename arm beside the directory arm, each narrowing killing
through its own entry test.

Ratio: 267 measured of 5536 (50+10+207 M across Tables A/B/D;
Table C all U); 191 bound; 5078 unreachable with reasons. Every
driven refusal direction is measured except the bound cells: 27
driven but rowless (B22 singletons/forks, B23 normalized details,
B32 propagation, B36 singleton deps) and
160 named undriven gaps (B24/B25/B26/B28/B31 cells
and the 153 B39 reachable-undriven cells), each with its
owner below; B29/B40/B41/B42/B43/B44/B45 are non-cell bounds stated below.
(Table A's 4 B15 guard-name arms are bound outside this split.)

Table E: after-restore outcome grid (ExecuteWrapperRestore,
wrapper.go:97). Each row is one decided outcome with its evidence
condition, its backend effect, and the committed test plus the
narrowing row that kills a routing/order mutant through the
wrapper entry (test names drop the Test prefix):

| Outcome | Evidence condition | Backend effect | Test / row |
|---|---|---|---|
| launch | verified local win + valid materialization | executes backend restore exactly once | WrapperRestoreLocalWinResumes; reorder covered by N-wrapper-reorder-refresh |
| reattach | verified local win + recorded bootstrap pair | executes backend restore exactly once | WrapperRestoreReattachReplaysBackend; reorder covered by N-wrapper-reorder-refresh |
| attach_remote | refreshed remote winner + lapsed local grant + interactive | none (no backend op, no exec, no provider) | WrapperRestoreLapsedGrantRemoteOffer/attach; N-wrapper-reorder-refresh |
| takeover_offer | refreshed remote winner + lapsed local grant + non-capable attach | none (no backend op, no exec, no provider) | WrapperRestoreLapsedGrantRemoteOffer/takeover; N-wrapper-reorder-refresh |
| parked (remote-noninteractive) | refreshed remote winner + non-interactive terminal | none | WrapperRestoreParksWithoutEffects/remote-noninteractive; N-wrapper-reorder-refresh |
| parked (refresh-failed) | refresh error → unverified local knowledge | none | WrapperRestoreParksWithoutEffects/refresh-failed; N-wrapper-reorder-refresh |
| parked (no-adapter) | missing refresh adapter → unverified local knowledge | none | WrapperRestoreParksWithoutEffects/no-adapter; N-wrapper-reorder-refresh |
| parked (no-known-lease) | no lease known at all → zero winner | none | WrapperRestoreParksWithoutEffects/no-known-lease; N-wrapper-reorder-refresh |
| parked (invalid-material) | required materialization not admitted | none (routing mutant would execute) | WrapperRestoreParksWithoutEffects/invalid-material; N-wrapper-route-material |
| parked (refresh-timeout) | refresh blocks → cancelled at the bound, unverified | none | WrapperRestoreRefreshBoundMeasured/configured-50ms; N-refresh-bound-default (default leg) |
| parked (expired refresh) | refresh answers after the deadline with a nil error → rejected, unverified | none | WrapperRestoreExpiredRefreshParksUnverified; expiry covered by N-wrapper-refresh-expired |
| parked (late-answer race) | both-ready selection → the late answer never verifies | none | WrapperRestoreRefreshRaceRejectsLateAnswer; N-wrapper-refresh-expired |
| parked (non-cooperative adapter) | adapter ignores cancellation → the entry returns at the bound | none | WrapperRestoreRefreshBoundEnforced; N-wrapper-refresh-deadline |
| refuse (divergent winner) | refreshed winner ≠ carried restore authorization | none, failed local precondition | WrapperRestoreRefusesDivergentWinner(+Lease,+Epoch); N-wrapper-winner-lease, N-wrapper-winner-epoch |
| refuse (divergent session) | decision session ≠ restore session | none, failed local precondition | WrapperRestoreRefusesDivergentSession; N-wrapper-restore-session |
| refuse (divergent bootstrap) | decision bootstrap ≠ restore bootstrap | none, failed local precondition | ReviewWrapperBootstrapMustBindExecutedRestore; N-wrapper-restore-bootstrap |
| refuse (divergent instance) | admitted descriptor instance ≠ restore instance | none, failed local precondition | ReviewWrapperInstanceMustBindExecutedRestore; N-wrapper-restore-instance |
| refuse (divergent backend) | admitted descriptor backend ≠ restore backend | none, failed local precondition | WrapperRestoreRefusesDivergentBackend; N-wrapper-restore-backend |
| refuse (divergent generation) | admitted descriptor generation ≠ restore generation | none, failed local precondition | WrapperRestoreRefusesDivergentGeneration; N-wrapper-restore-generation |
| refuse (mode) | Decide.Mode != restore (incl. garbage) | none, invalid-arguments | WrapperRestoreRefusesMalformed; N-wrapper-mode |
| refuse (operation) | Restore.Operation != restore (incl. garbage) | none, invalid-arguments | WrapperRestoreRefusesMalformed; N-wrapper-operation |
| refuse (deps) | nil lifecycle or nil dependency | none, protocol-error | WrapperRestoreRefusesMalformed; B36 singleton (no narrowing row) |

The refresh runs before the decision on every row (N-wrapper-
reorder-refresh admits exactly the fixture session past the
refresh-success arm into stale local knowledge and is killed by
the offer/resume/park tests together), so the offer path never
presents the lapsed grant to backend restore authorization. The
resume rows execute only bound to the decided session, bootstrap,
instance, backend, generation, and winner as one composed
authorization (N-wrapper-restore-session, N-wrapper-restore-
bootstrap, N-wrapper-restore-instance, N-wrapper-restore-backend,
N-wrapper-restore-generation, N-wrapper-winner-lease,
N-wrapper-winner-epoch); late answers never verify whether they
arrive after the deadline (N-wrapper-refresh-expired) or the
adapter ignores cancellation (N-wrapper-refresh-deadline). No
production mesh transport backs the adapter (B37). Union-rival
ambiguity is unmodeled (B38). The parked wire result
of the backend restore entry itself (RestoredParked) stays a
backend-row concern, not a Table E row.

## Bounds (mirrored B22-B45 plus carried dispositions)

- B22 Singleton arms ship no narrowing mutant: admitting the
  singleton refused member is deleting the gate. Covered sites, each
  test-pinned in both directions where a fork exists: create relay
  transport (relay), descriptor presence fork (descpresence),
  attach admission nil (serveadmitnil), spawner runner nil
  (spawnnil), reconcile verifier nil (reconcilenil), states root
  empty (stateopen), persist/attach stores nil (storenil),
  failed-kill self-confirm (selfconfirm), live-after-escalation
  (stoptimeout), and post-wait boundaries without a configured
  revalidation (recheckPostWait nil). The outcome-lookup empty-key
  arm left this bound: it narrows now (LookupOutcomeRefusesForgedKey /
  N-outcome-key-forged). Owner: this leaf; retest if any site
  grows a second refused member.
- B23 The landed status engine normalizes backend error details to
  "status observation unknown" (coded or not), so the cutrow and
  empty-panes narrowings measure at the helper entries while the
  dispatch legs pin the code. Owner: this leaf; retest if the
  engine stops normalizing.
- B24 The idempotency-store Completed read-failure arm is unstaged:
  no seam fails receipt reads (hooks fire on write paths only).
  Owner: hardening; stage with a read-failure seam to measure.
- B25 The Bind clean-miss sub-arm is unstageable: every Bind error
  the hooks stage leaves the committed file, so Lookup always finds
  it. The clean-found sub-arm is measured (N-bind-hook-found).
  Owner: hardening.
- B26 The attach effect-failure dispatch mapping is unstaged: no
  committed failure reaches it (the store replays identical
  receipts; conflicting receipts are untested at dispatch).
  Owner: hardening.
- B27 No committed test composes the Production Dependencies into
  Acquire: AFG reaches the adapter and bind gates only through
  caller-injected implementations (the 18 Table C U2c cells). The
  injection seam is first-leaf-owned; Production wiring is
  unit-pinned (TestProductionWiresDependencies). Owner: composition
  test, sibling scope.
- B28 The backend command-error wrap is unmeasured at PFX (no
  backend test stages a refused build; the builder rows pin the
  refused shapes at CMD). Owner: this leaf; retest with a
  refused-build backend test.
- B29 The close-confirm poll's exact-instant member is unmeasured
  at every entry: the real-time in-flight bound has no stageable
  equal instant, and no committed test stages the clock arm's
  instant member (past-deadline concludes, pinned at EXE and at
  PFX behaviorally). Dispatch-level staging is measured (row
  pollbound). Owner: hardening.
- B30 The OSRunner exit/transport/argv mappings are total functions
  over uncoded inputs with no refused class to admit a member of;
  tests pin them including real-exec scrub, stderr-capture, and
  signal-death integrations. The signal-death arm (a killed child
  errors instead of reporting -1 data) is a refused class and
  carries its own narrowing row; the remaining mappings stay total.
  Owner: this leaf.
- B31 The custody root-ownership arm is untested and singleton: the
  ownership seam stages leaf-first (the leaf refuses before the
  root arm runs), and a truly foreign root needs privilege. Owner:
  hardening with a privileged stage or a root-level seam.
- B32 The leaf-delegation wiring is measured at CUS (wrong-leaf
  mutants are observable only on compliant roots); the absence
  propagation through the six dispatch paths, the prober, and the
  spawner is pinned by absence tests without a row. Owner: this
  leaf.
- B33 The wrapper safe-boundary signal step is unimplemented in the
  landed wrapper: the boundary entry waits blocking on
  `ax-boundary-<generation>` for the wrapper's `wait-for -S`, and
  the wrapper never signals today, so boundary waits time out with
  the coded quiesce_timeout (fail closed, never a faked proof)
  until the signal step lands. The wait vector, the proof-kind
  binding, the provider-row context, and the timeout are this
  leaf's and fully driven; the signal emission is the wrapper's.
  Owner: the wrapper (`ax pane`) task; retest the full
  quiesce-boundary-stop flow when the signal lands.
- B34 A crash between the mutation-receipt commit and the operation
  outcome record re-observes the report time on retry: the effect
  never re-runs and the evidence never changes, but the reported
  closure/boundary time renders from the retry clock instead of
  the original one. Sequential retries without a crash replay
  identical times. Before the retry heals the report, input can
  reopen: attach consults the outcome proof, so the missing report
  reads as no closure (bound B43 states the owner and the scoped
  contract for the structural fix). Owner: this leaf for the
  timestamp behavior; the receipt owner for the authority window.
- B35 The absence stderr markers are verified against tmux 3.6a
  (`can't find session`, `can't find window`, and the
  no-such-socket connect error); any other version's
  absence-shaped stderr fails closed to unknown. Owner: this leaf;
  re-verify the marker set when the tmux floor version moves.
- B36 The Execute dispatch and the ExecuteWrapperRestore entry pin
  their dependency singletons without per-arm narrowing rows: a
  nil lifecycle or a nil Runner, Receipts, Bindings, Attach,
  States, CurrentLease, CurrentGeneration, or Now refuses
  protocol-error before any decision or effect. Admitting one nil
  arm is deleting the check. Owner: this leaf; retest if any site
  grows a second refused member.
- B37 No production mesh lease-refresh transport exists:
  LeaseRefresh is an injected adapter (function field), and the
  after-restore composition enforces the configured bound on any
  adapter — cooperative or not — and rejects late answers even
  with a nil error. The mesh RPC transport that would back a
  production adapter is sibling scope (the mesh story); until it
  lands, callers that need verified refreshes inject their own.
  Owner: the mesh-refresh composition; retest the bound when a
  production adapter lands.
- B38 Union-rival ambiguity is unmodeled at the after-restore
  composition: the entry decides over the single refreshed or
  local winner, and a refresh adapter that observes rivals must
  surface them as refresh failure so the entry parks unverified.
  Owner: the mesh-refresh composition; retest when rivals are
  modeled.
- B39 Reachable-undriven: the entry reaches the gate but the
  committed tests carry post-gate values only, so the refused
  direction is undriven at that entry; the narrowed direction is
  measured at the listed entry ("M at ..."). Formerly claimed
  unreachable as U2b; stated as a bound now. Owner: this leaf;
  drive a direction to move its cell to M.
- B40 Lost incarnation rotation with a prior closure stays
  closed with no production-entry recovery: the failed rotation
  fails the operation closed, identical retries replay success
  without rotating (rotating on replay would erase a genuine
  newer closure), and a fresh key is unreachable because the
  landed bootstrap store binds one bootstrap per session and the
  receipt store refuses a second operation under a replayed key.
  The stuck closure is over-strict in the safe direction; the
  committed tests pin the failure, the replay skip, and the
  bootstrap-store mechanism. A lost rotation with no prior
  closure is harmless (the zero incarnation stays
  self-consistent). Owner: axpane receipt/bootstrap contracts
  for a repair entry; operator surgery clears the superseded
  reports until one lands.
- B41 The restore replay rotation skip is structural: the replay
  branch contains no rotation code, so no narrowing mutant can
  weaken it — the supplementary additive row
  D-restore-replay-rotation inserts the forbidden write to prove
  the committed replay test observes it. The create-side skip
  has a narrowing row (N-create-replay-skip) because its
  freshness condition is explicit code. Owner: this leaf; the
  asymmetry is stated, not hidden.
- B42 The input-closure ordering contract is in-process: the
  barrier mutex serializes quiesce/boundary report commits
  against attach receipt commits within one Lifecycle, but
  concurrent ax processes racing attach against quiesce observe
  only the report files, so a receipt committed in another
  process's report-write window is not excluded. No committed
  test stages two processes. Owner: this leaf; closing the
  window needs cross-process exclusion (for example a lock file
  under the state root). The vector handoff the contract covers
  ends at vector construction: a caller that execs a
  pre-quiescence vector after quiescence commits holds a
  pre-quiescence authorization, which quiesce does not revoke.
  The same in-process scope covers the attach post-lock
  generation and deadline rechecks: a rotation landing between
  the recheck and the commit — inside one process or across
  processes — is not excluded; the window holds no waits.
- B43 Closure authority has no single durable source. The engine
  commits input closure and its receipt before this leaf records
  the outcome report, and attach admission consults the outcome
  proof alone, so a lost outcome-report write after committed
  closure admits a fresh writable attach until the identical
  retry heals the report: the failure to create the proof reads
  as permission, and pending-versus-committed closure is not
  representable in the outcome store. The same window covers a
  crash between the receipt commit and the report, and a
  simultaneous loss of both the report and the state record.
  Retrying the identical quiesce heals the report and re-closes
  input; no other production entry heals it. Closing the window
  structurally needs the landed receipt owner
  (internal/terminstance ReceiptStore) to expose
  completed-closure enumeration — the completed quiesce/boundary
  receipts for one instance, distinguishing
  pending-without-completion from completed — so that attach
  admission can consult receipt completion as the authoritative
  proof. Owner: internal/terminstance for the enumeration
  contract; this leaf for the admission side once it lands.
  Until then the construction (incarnation-scoped proof,
  advisory-memory split, in-process ordering) stands, and row 79
  claims only the lost state-record write fails closed.
- B44 Attach overlap is inferred from stored receipts, not
  liveness: a recorded peer reads as possibly live, so the overlap
  gate requires multi_attach (and multiple_input_clients for
  concurrent input) even when the peer already detached, with
  receipt-validated same-client retries exempt and unreadable
  peers failing closed. The receipt-validated distinction —
  a Lookup-proven recorded receipt replays, a client with none
  still gates — is this leaf's; the active-client census that
  refines the possibility further — overlap admission over live
  clients — is owned by
  TASK-260922-vcx6yo (active-client-census-and-overlap-admission).
  Until it lands this leaf fails closed (over-strict) rather than
  admitting overlap without the capabilities. Owner:
  TASK-260922-vcx6yo for the census; this leaf for the fail-closed
  gate.

- B45 Single-shot commit commands (lock-session, detach-client,
  send-keys, new-session, kill-session) and the bind-step spawn
  are bounded by caller cancellation only — not by the operation
  deadline. A hung commit therefore hangs until the caller
  cancels; it never manufactures a verdict (a cancelled commit
  surfaces the honest uncertain: uncoded failure, status_first).
  Mid-commit cancellation cannot un-commit, so bounding commits
  would trade hangs for uncertainty without changing any verdict;
  the landed engine passes its context through to commits and
  checks the deadline between effects, and this leaf mirrors
  that contract rather than diverging from the landed owner.
  Every read-only wait and probe, by contrast, carries a
  deadline-derived context (rows waitbound, pollbound,
  confirmbound, statusbound, boundarywait). Owner: this leaf
  for the audit; internal/terminstance for any engine-wide
  commit-bound contract change.

Carried bounds (first-leaf statements dispositioned by this leaf).
B9 adapter split is discharged: the production exec, dial, probe,
and spawn adapters ship (exec.go, probe.go, backend.go) and
Production wires them into Dependencies; the residual is the
uncomposed wiring (B27). B16 nested invocation is decided: lifecycle
entries take no Ambient input, so nesting cannot collide through
them (row 90); the Acquire-level over-strict nested refusal stands
unchanged as first-leaf-owned. B18 sun_path is decided: strict-below
per platform, counted in bytes (row 66). B19/B20 bind step is built:
CheckSocketCustody enforces placement, leaf delegation, root,
ancestors, and socket kind (rows 67-70); the bind step refuses
intermediate symlinks through no-follow opens, which is stricter
than the B11 Acquire rule it stands beside. The rev7 P3-A fresh-decoy
foreground attach wiring is closed: TestAcquireForegroundRefusesEveryCatalogDecoyRunning
refuses every catalog-derived decoy alone and together on the
foreground running path with a spawn spy silent on every member, and
the attach attestation cells carry N-attach-attestation-decoy. New
bounds this revision: B33 states the unimplemented wrapper boundary
signal (fail-closed timeout until it lands), B34 the receipt/outcome
crash window, B35 the tmux-version marker set, B36 the dependency
singletons, B38 the unmodeled union rivals, and B39 the
reachable-undriven reclass. Revision 4 fills B37 (no production
mesh refresh transport; the composition enforces the bound on any
injected adapter), extends Table E with the binding, expiry, race,
and enforcement outcomes, and moves the SPW rootwrite cell from
B39 to measured. Revision 5 makes the incarnation-scoped closure
proof the sole barrier verdict (new Table D rows
barrierincarnation, barrierkey, attachhealth, failurerecord,
createincarnation, restoreincarnation, incarnationrecord,
incarnationlookup), extends Table E with the five composed-binding
refusals, moves the PRB rootwrite cell from B39 to measured, and
states B40 (stuck rotation) and B41 (structural replay skip).
Revision 6 adds the post-admission barrier recheck with the
in-process ordering lock (new Table D rows recheck and ordering;
attach 1 / quiesce 1 / boundary 1), the barrier read-error arms
(new Table D row barrierread; directory-read 1 / file-read 1),
moves the incarnationlookup EXA cell from B39 to measured, extends
the rootwrite EXA cell with the attach entry row, recounts Table C
from the Table D gate set (140 gates, 1540 cells), narrows row 79
to the lost state-record write with the outcome-report window
stated as B43 (receipt-owner enumeration contract), and states B42
(in-process ordering). Revision 7 closes the attach
admission-to-effect interval for every remaining live fact (the
post-lock generation and deadline rechecks with the full fact
enumeration; the authorization facts revalidate inside the
commit), moves the generation EXA cell from U1 to measured,
extends the deadline and authkind cells with the new rows,
reclassifies the two effect-loop authkind wirings as
supplementary D rows beside the genuine shared loop narrowing
N-lifecycle-loop-authorization, converts the entry authkind rows
to genuine no-bind narrowings, and extends B42 to the new
rechecks. Revision 8 closes the stop admission-to-effect interval
at the escalation boundary and the quiesce interval between its
commands (new Table D row effectrecheck; stop 4 / quiesce 4),
bounds every barrier and admission wait by the operation deadline
and cancellation (new Table D row waitbound; EXA 1 / EXE 1
shared), enforces the attach overlap capabilities fail-closed
over the stored peer census (new Table D rows overlapmulti and
overlapinput; liveness owned by TASK-260922-vcx6yo as B44),
reclassifies the entry-deadline instant row as
shadowed-supplementary (the wait bound refuses the same member),
extends B22 with the nil-recheck singleton, and recounts Table C
from the Table D gate set (144 gates, 1584 cells). Revision 9
refuses receipt-namespace directories in the peer census (new
Table D row peershape), exempts receipt-validated same-client
replays with a peer present (new Table D row overlapreplay),
bounds every read-only probe in-flight by the operation deadline
(new Table D rows confirmbound and statusbound; pollbound moves
from B29 to measured with the terminate tail corrected from U1
to B39), repairs the three false per-entry kill attributions
with isolated single-test evidence (stop 4 / quiesce 4 and 4
wait killers now measured, not claimed), revises B29 to the
exact-instant remainder, clarifies B44 (receipt-validated replay
here, liveness with TASK-260922-vcx6yo), states B45 (commits
bounded by caller cancellation), and recounts Table C from the
Table D gate set (148 gates, 1628 cells).

## Mutant battery

`PYTHONDONTWRITEBYTECODE=1 python3
internal/tmuxserver/mutant_harness.py`: 298/298 rows as
expected (291 narrowing `N-*` KILLED plus 5 supplementary
`D-*` KILLED (3 additive + 2 positive-path wiring), 1
supplementary `D-*` SURVIVED-shadowed, and the shared
harmless control SURVIVED), run in
two full passes with one raw log per
plant per pass in the evidence tarball. The second leaf owns 218
narrowing rows net (221 added, 3 reclassified). Multi-entry rows list every entry their
killers cover, attribution verified per test from the raw
logs and, for the shared arms, from isolated single-test runs
(see the row notes and the isolated attribution logs).
`PYTHONDONTWRITEBYTECODE=1 python3
internal/termbind/mutant_harness.py`: 43/43 rows as expected
(41 narrowing plus 1 supplementary plus the control),
including the 4 reboot-successor rows and the 3 peer-census rows.
Singleton refusal arms
ship no row by bound B22 (the outcome-lookup empty-key arm
left that set by narrowing); engine-normalized status details
measure at the helper by bound B23; reachable-undriven cells
are bound B39, not claimed unreachable.

## Crash, idempotency, and determinism

Every mutating operation replays the identical request without
new execs (idempotency tests, one per mutating op); uncertain
receipts reconcile through status before resuming (resume
tests for effect failure, recheck failure, and
otherwise-proven status); tampered and corrupt replays refuse
(per-member tamper table, corrupt-image, hook-failure tests);
reboot restore mints a deterministic successor generation and
converges on retry after a crash between effects and persist.
No committed test execs a real tmux or dials a real tmux
server socket: the scripted runner and staged sockets are the
only tmux the suite sees (one dialer test dials a synthetic
local listener it creates itself, not a tmux server), and the
suite passes with no tmux installed
(bound B1). The three wrapper refresh tests plus the eight wait/probe
tests measure wall-clock time against release-controlled
instants or generous guards (enforcement is timing by
definition); every other test runs on the manual clock.

## Validation and evidence

Configured suite green firsthand on the tree (gofmt, build,
vet, full suite, race on both touched packages, cover,
tracecheck, cataloggen, specdoc, linux/windows builds);
341/341 mutant rows as expected across both harnesses with one
raw log per plant. Raw logs:
`TASK-260830-1c28dz_producer-evidence.tar.gz`.
