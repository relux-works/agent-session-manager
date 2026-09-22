# TASK-260830-1geqhj: implementation evidence

Authority: relux-works/agent-session-manager-spec v0.5.0, commit
`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, worked against the
`internal/specdoc/SPEC.v0.7.0.md` headings named below (byte-identical
to v0.6.0 in these sections). Primary scope is Sections 4.B-4.D,
4.1-4.2, 5.2 Terminal Events, 5.7 (newest checkpoint), 7.A, and
13.1, plus the 2.4 profile authority the wrapper composes. This
file records implementation evidence; per the Story contract the
`internal/traceability` ownership registry is untouched in this
first leaf — the Story's final leaf carries the registry bindings
and re-pin. This package claims no public CLI delivery: no `ax`
command exists in this tree.

The wrapper validation core (`internal/axpane`) composes every
check from a landed gate and re-implements none: owner epoch and
fencing from `fencing` (`AuthorizeActivation`, `AuthorizeRestore`,
the sealed `LeaseToken`, the parked vocabulary); the authoritative
newest checkpoint from `sessstate` (`Reduce` over the record with
its authoritative chain: `Projection.HasCheckpoint`/`Newest.ID`);
materialization validity from `matjournal` (`Store.Get` with its
grammar and cross-reference validation, plus the stated
committed-phase and fold-newest-source rules) and checkpoint
admission from `sessckpt` (`Get`, which re-attests through
`sessrepo.AttestCheckpointRecord` on every read) with the identity,
newest, and session bindings in `Decide`, plus the one
successor-lease implication (a checkpoint-carrying winner means the
fold published a newest) asserted once directly after fencing; the
effective persisted profile from `sessprofile` (`DecodeRecord`,
`DecodeEvent`, `Derive`, `DeriveForHeads`) with Section 7.7 mapping
from `provhost` (`ResolveMapping`,
`profile_mapping_unavailable`); provider identity from `provhost`
(`CheckResumeTuple`, `VerifyIdentityBuild`,
`VerifyIdentityDiscovery`, `CreateIdentity` in fixtures) with
`resumesmoke` evidence (`VerifyRecord`) as a precondition input;
backend identity from `terminalbackend` (`ParseManifest`,
`ParseProbe`, `ParseEvidence`, `GenerationDigest`, `Reconcile`,
`CapabilitiesForOperation` via `CheckOperation`, `CheckEntrypoint`,
`AdmitProviderDescriptor`); configuration validity from `config`
(`Load`); session, lease, and event durability from `sessrepo`
(`GetRecord`, `ListEvents`, `ListSessions`, `fencing.Observe` over
`ListLeases`, `AppendEvent`); refusal shapes from `axerror`
(`NewRealmEvidenceUnavailable`, `NewTargetAuthMissing`).

## Acceptance rows

43 of 43 AC rows are driven at the production entries named below;
the denominator is re-derived by counting this table (43 data
rows), not restated: rev2 claimed "41 of 41" while the table has
always held 43 rows, and the independent reviewer measured 39 of
43 faithfully driven. The four unfaithful rev2 rows —
"Materialization validity gates local resume", "Checkpoint
admission gates resume", "After-restore sequence 1-5" (all three
shared the lease-checkpoint keying), and "Bootstrap idempotency"
(the superseded receipt) — were re-pinned faithfully in rev3, and
the four rows the rev3 reviewer measured unfaithful (the same
first three, now sharing the strict lease/fold equality, plus
"Effective profile derived with source", pinned only for the
epoch-1 owner) are re-pinned faithfully in rev4: the post-takeover
lifecycle launches from the fold's newest with the newest
checkpoint's closure profile in both modes and both profile
directions, the successor null fold parks, and the stale base
parks. 0 of 43 are delivered as a public CLI surface,
which is an explicit caller-integration bound (no `ax` command
exists in this tree), not a gap in the assigned validation
behavior. Process control, mesh transport, provider supervision,
and backend operations belong to modeled callers and later leaves,
which invoke these entries at every required boundary.

| AC row | Production call site | Named test(s) | Evidence and bound |
| --- | --- | --- | --- |
| Lower epoch parks or refuses | `Decide` → `fencing.AuthorizeActivation` | `TestDecideTable/parked_lower_epoch` | Driven: presented epoch 1 under a winning epoch 2 parks `stale_owner` with the winning lease. |
| Same-epoch loser parks | `Decide` → `fencing.AuthorizeActivation` | `TestDecideTable/parked_same_epoch_loser` | Driven: the losing lease at the winning epoch parks `stale_owner`. |
| Future epoch parks | `Decide` → `fencing.AuthorizeActivation` | `TestDecideTable/parked_future_epoch` | Driven: an epoch beyond the winner parks `restore_policy`. |
| Foreign session refuses | `Decide` → `fencing.AuthorizeActivation` | `TestDecideTable/refused_fencing_foreign_session` | Driven: a token for another session refuses `lease_conflict`. |
| Expired grant refuses | `Decide` → `fencing.AuthorizeActivation` → `sessrepo.CheckFencingExpiry` | `TestDecideTable/refused_fencing_expired_grant` | Driven: a lapsed grant refuses `lease_conflict` on the launch entry. |
| Malformed token refuses | `Decide` → `fencing.AuthorizeActivation` | `TestDecideTable/refused_fencing_malformed_token` | Driven: epoch 0 refuses `invalid_arguments`. |
| Absent winner parks without authority | `Decide` → `fencing.AuthorizeActivation`; `Run` skips emission | `TestDecideTable/parked_absent_winner`, `TestRunParkedWithoutWinnerSkipsEmission` | Driven: no winner parks `restore_policy` with an empty lease ID and authors no event, since `session.parked` requires a winning lease. |
| Ambiguous ownership parks | `Decide` → `fencing.AuthorizeActivation` | `TestDecideTable/parked_ambiguous` | Driven: conflicting union observations park `restore_policy`. |
| Unverified ownership parks | `Decide` → `fencing.AuthorizeActivation`; `Emit` → `fencing.AuthorizeMutation` refuses | `TestDecideTable/parked_unverified`, `TestEmitParkedVocabularyLoop/refuses_unverified` | Driven: without a proving lease refresh the wrapper parks `restore_policy` with no chain event — the mutation gate refuses unverified emission. |
| Failed handoff parks | `Decide` → `fencing.AuthorizeActivation`; `Emit` → `fencing.AuthorizeMutation` refuses | `TestDecideTable/parked_failed_handoff`, `TestEmitParkedVocabularyLoop/refuses_failed_handoff` | Driven: a losing force lease parks `failed_handoff`, never activates and never emits. |
| Remote interactive owner offers attach | `Decide` → `decideParked` + `terminalbackend.CheckOperation("attach")`; `Run` skips remote emission | `TestDecideTable/attach_remote_interactive` | Driven: a remote winning holder with an interactive terminal and an admitted attach capability offers `attach_remote` with no lease mutation and no chain event under the remote lease. |
| Remote owner otherwise parks or offers takeover | `Decide` → `decideParked`; `Run` skips remote emission | `TestDecideTable/takeover_offer_noninteractive`, `TestDecideTable/takeover_offer_attach_unproven`, `TestRunRemoteNonInteractiveParksWithoutEmission` | Driven: a non-interactive remote owner parks `remote_owner` (§4.2 step 5); an interactive attach-unproven remote owner offers `takeover_offer`. Both start no runtime and author no chain event under the remote lease. |
| Mode selects entry and capability row | `Decide` → `authorize` + `admitBackend` | `TestDecideRestoreRequiresRestoreCapability`, `TestDecideTable/launch_restore_committed_checkpoint` | Driven: restore mode gates through `AuthorizeRestore` and the `restore` row (a restore-admitting, create-denying backend launches); launch mode gates through `AuthorizeActivation` and `create`. |
| Materialization validity gates local resume | `Decide` → `checkMaterialization`; `Run` → `LoadMaterialization` + `LoadNewestCheckpoint` | `TestDecideTable/parked_materialization_not_committed`, `TestDecideTable/parked_materialization_absent`, `TestDecideMaterializationStaleParks`, `TestDecideJournalNullFoldParks`, `TestRunRestoreCommitted`, `TestRunRestoreMaterializingParks`, `TestRunEpochOneOwnerAfterCheckpoint`, `TestRunNullFoldRestoreParks`, `TestDecidePostTakeoverDivergedBaseLaunches`, `TestRunPostTakeoverRestoreLaunchesWithNewestClosure`, `TestRunPostTakeoverStaleBaseParks`, `TestDecideSuccessorNullFoldParks`, `TestRunSuccessorNullFoldParks` | Driven: a required journal must be admitted by the `matjournal` owner, committed, and sourced from the fold's newest published checkpoint — the winning lease's handoff base is never compared; staging, absence, a null fold, or a superseded source parks `restore_policy`. The epoch-1 owner (null lease checkpoint) parks on an unnamed-digest journal and launches on a newest-sourced one; the post-takeover owner (successor lease base C0, fold newest C1) launches on a C1-sourced journal and parks on a C0-sourced one. A checkpoint-carrying winner over a null fold parks at the one implication arm. Fresh launch leaves the requirement unset (§13.1 runs before the journal commits). |
| Checkpoint admission gates resume | `Decide` → `admitCheckpoint` → `sessrepo.AttestCheckpointRecord`; `Run` → `LoadCheckpoint` + `LoadNewestCheckpoint` | `TestDecideTable/parked_checkpoint_inadmissible`, `TestDecideTable/parked_checkpoint_absent`, `TestDecideTable/parked_checkpoint_wrong_identity`, `TestDecideTable/parked_checkpoint_flipped_digest`, `TestDecideCheckpointNotLeaseParks`, `TestDecideForeignCheckpointParks`, `TestDecideCheckpointNullFoldParks`, `TestDecideCheckpointUnpublishedParks`, `TestRunEpochOneOwnerAfterCheckpoint`, `TestRunNullFoldRestoreParks`, `TestDecidePostTakeoverDivergedBaseLaunches`, `TestRunPostTakeoverRestoreLaunchesWithNewestClosure`, `TestRunPostTakeoverStaleBaseParks` | Driven: a required checkpoint must attest through the `sessrepo` owner, match the required digest, equal the fold's newest published checkpoint, and name this session — the winning lease's handoff base is never compared; garbage, absence, identity drift, a null fold, an unpublished checkpoint, a non-newest checkpoint, or a foreign session parks `restore_policy`. The post-takeover owner (successor lease base C0, fold newest C1) launches on C1 and parks on the stale base C0. |
| Effective profile derived with source | `Decide` → `deriveProfile` → `sessprofile.Derive`/`DeriveForHeads`; `Run` → `LoadProfile` + resumed-checkpoint closure | `TestDecideProfileSourcePinsEvent`, `TestDecideYoloMapping`, `TestDecideResumeProfileUsesClosure`, `TestLosingLeaseProfileEventIgnored`, `TestRunNoCkptStorePublishedNewestFails`, `TestRunPostTakeoverLaunchCarriesNewestClosureStandardToYolo`, `TestRunPostTakeoverLaunchCarriesNewestClosureYoloToStandard`, `TestRunPostTakeoverRestoreLaunchesWithNewestClosure`, `TestRunNewestAbsentFromStoreFails` | Driven: launch carries the session head (newest authoritative `profile.changed` with that event's digest, or the record value with null source) only on the checkpoint-free path; resume carries the checkpoint actually resumed — the required checkpoint when one is required, else the fold's newest — through `DeriveForHeads`, never the winning lease's handoff base, so a head-only or losing-lease change never becomes the launch profile (PROFILE-DIRECT-RESUME direction). Both post-takeover profile directions launch with the newest checkpoint's closure. A published newest with no checkpoint store bound fails the run, as does a newest absent from the store — an absent store is unknown, never "no closure". Corrupt derivation refuses `integrity_failure` (`TestDecideTable/refused_profile_derivation_corrupt`). |
| Profile mapping resolved or refused | `Decide` → `provhost.ResolveMapping` | `TestDecideYoloMapping`, `TestDecideTable/refused_profile_mapping_unavailable` | Driven: the yolo launch carries the exact codex adapter flag; a Pi build off the pinned version refuses `profile_mapping_unavailable` per §2.4. The tuple gate precedes mapping so a tuple refusal keeps its own class. |
| Resume tuple gate | `Decide` → `checkProviderIdentity` → `provhost.CheckResumeTuple` | `TestDecideTable/refused_provider_tuple_qwen` | Driven: a tuple outside the §8.4 direction refuses `invalid_config` with the landed detail. |
| Identity bind to exact build | `Decide` → `provhost.VerifyIdentityBuild` + `checkIdentitySession` | `TestDecideTable/refused_provider_identity_other_provider`, `TestDecideIdentityForeignSessionRefuses` | Driven: an identity minted for another provider (`CreateIdentity` fixture) refuses `invalid_config`; the probed version must equal the record version exactly; and an identity minted for another session with the same provider and version refuses `invalid_config` — the wrapper binds the record's own `session_id` to the session under decision (`VerifyIdentityBuild` binds provider and version only). |
| Discovery bind | `Decide` → `provhost.VerifyIdentityDiscovery` | `TestDecideTable/launch_discovery_bound`, `TestDecideTable/refused_discovery_native_mismatch`, `TestDecideTable/refused_discovery_root_outside`, `TestDecideTable/refused_discovery_empty_home` | Driven: a proof binding the identity's native ID under the §8.2 store root launches; native drift, an outside root, or an empty home refuse `invalid_config`. |
| Provider-auth smoke precondition | `Decide` → `checkSmoke` → `resumesmoke.VerifyRecord` + `axerror.NewTargetAuthMissing` | `TestDecideTable/refused_smoke_failing_verdict`, `TestDecideTable/refused_smoke_absent`, `TestDecideTable/refused_smoke_wrong_build`, `TestDecideTypedRefusals/target_auth_missing_details`, `TestRunSmokeRequiredPasses`, `TestRealmCarriesNoCachedEvidence/stale_generation_with_realm_refuses` | Driven: a required smoke must verify, pass, bind the exact probed build, and bind the current generation with the probed OS version in the typed target; failure, absence, build drift, or a stale generation/version refuses `target_auth_missing` with the typed §15.3 details. |
| Backend identity admission | `Decide` → `admitBackend` → `terminalbackend.ParseManifest/ParseProbe/ParseEvidence/Reconcile` | `TestDecideLaunchPositive` | Driven: a real manifest/probe/RSA-signed-evidence universe reconciles and admits `headless_creation` for `create` (and `reboot_restoration` for `restore`, `local_attach` for `attach`). |
| Mismatch fails before activation | `Decide` → `admitBackend` | `TestDecideTable/refused_backend_protocol_membership`, `TestDecideTable/refused_backend_generation_mismatch`, `TestDecideTable/refused_backend_evidence_expired`, `TestDecideTable/refused_backend_signature_untrusted` | Driven: protocol drift refuses `terminal_backend_manifest_probe_mismatch`; generation drift refuses the more specific `terminal_backend_stale_generation`; expired evidence refuses; an untrusted signature refuses `terminal_backend_integrity_failure`. |
| Only admitted rows authorize | `Decide` → `terminalbackend.CheckOperation` + headless conditional | `TestDecideTable/refused_backend_capability_unproven`, `TestDecideNonInteractiveRequiresHeadless` | Driven: with no admitted capability conferring `create`, the wrapper refuses `terminal_backend_capability_unproven`; a non-interactive create without `headless_creation` refuses even when another row confers `create`. No capability map is invented: the `Reconcile` set is the only authority. |
| Background caller without attested server | `Decide` → `checkRealm` → `terminalbackend.Reconcile` + `axerror.NewRealmEvidenceUnavailable` | `TestDecideTable/refused_realm_background_no_server`, `TestDecideTypedRefusals/capability_unavailable_details`, `TestDecideTable/launch_background_attested_server`, `TestDecideRealmBindingMismatchRefuses`, `TestDecideForegroundCredentialRequiresRealm`, `TestDecideRealmEvidenceOrderIndependent`, `TestDecideExpiredRealmRowRefusesUnavailable`, `TestDecidePreRebootRealmRowRefusesUnavailable`, `TestDecideLiveRealmRowBackendFaultStands` | Driven: a background caller, and any caller on a credential-requiring path (§4.C conditional rows), authorizes only through the admitted `credential_capable_execution_realm` row bound to the host binding, probed build, and current generation; without it (or with a mismatched binding/build or stale generation) it refuses `capability_unavailable` with typed details and never creates a server. An expired or pre-reboot realm row refuses the same typed refusal (the reconcile failure is the cause); an unrelated backend fault with a usable realm row keeps its own class; the cross-bind accepts when any realm object binds, in either evidence order. Incomplete typed details refuse `invalid_arguments` (`TestDecideTable/refused_realm_background_incomplete_details`). |
| Cached evidence never authorizes | `Decide` → `checkRealm` + `checkSmoke` (behavioral) | `TestRealmCarriesNoCachedEvidence`, `TestDecideForegroundCredentialRequiresRealm` | Driven: a background caller with a passing (cached) smoke record but no admitted realm row refuses, as does a foreground caller on a credential-requiring path; with a stale generation refuses; only the admitted row bound to the current generation, host binding, and probed build launches — no cached sentinel, `managername`, or boolean authorizes. |
| Bootstrap idempotency | `Store.Bind`/`Supersede`/`Lookup`/`SessionBound`, `Decide` (windowed idempotency arm), `Run` | `TestDecideTable/refused_idempotency_mismatch`, `TestDecideRestoreChangedOperationRefuses`, `TestDecidePostWindowNewOperationLaunches`, `TestDecideWindowEpochOneOwnerCloses`, `TestDecideTable/reattach_same_pair`, `TestBindIdenticalRetryReattaches`, `TestBindChangedOperationRefusesMismatch`, `TestBindSweepsStaleStaging`, `TestBindConcurrentSameOperationReattaches`, `TestBindConcurrentChangedOperationRefuses`, `TestSupersedeKeepsReceipts`, `TestSupersedeConcurrentDifferentOps`, `TestSupersedeConcurrentSamePairReplays`, `TestLookupSessionMismatchRefuses`, `TestRunReattaches`, `TestRunConcurrentLaunchReattaches`, `TestRunIdempotencyMismatch`, `TestRunPostWindowSupersedes`, `TestRunSupersededPairIdenticalRetry`, `TestRunCreateFromStoppedPostWindow`, `TestRunPostWindowSamePairRaceReattaches`, `TestRunWindowLeaseCheckpointNullFoldRefuses` | Driven: a changed operation inside the bootstrap window (the fold names no newest checkpoint — the lease record's checkpoint never closes it) refuses `idempotency_mismatch` in both modes; an identical retry of any recorded pair replays its own receipt (the stored receipt is authoritative, superseded pairs keep theirs); a concurrent same-operation commit reattaches while a changed-operation commit refuses, in-window and post-window alike (`Supersede` reports the replay and `Run` flips to `reattach`); after the window closes a new operation records its own pair receipt no-replace and launches, in both modes. Stale pre-commit staging is swept. |
| Status proves absence or the one child | `Store.Status` + `Store.Lookup` | `TestBindInstallsAndStatusProves`, `TestStatusSessionMismatchRefuses`, `TestLookupSessionMismatchRefuses`, `TestSupersedeKeepsReceipts` | Driven: absence proves only when no binding exists; `Status` identifies the first-binding anchor while `Lookup` replays each pair's own receipt; a receipt read under another session or operation refuses even with a shared prefix. |
| Unknown is not absent | `Store.Status`/`Lookup`/`SessionBound`, `Run` → `ObserveOwnership` + `LoadNewestCheckpoint` | `TestStatusUnknownIsNotAbsent`, `TestRunTornBindingPropagates`, `TestRunTornLeasePropagates`, `TestRunUnfoldableChainFails` | Driven: a torn anchor fails `Status` and `SessionBound`, a torn pair receipt fails `Lookup` and the identical retry, a torn lease fails `ObserveOwnership`, and an unfoldable chain fails `LoadNewestCheckpoint`, each propagating through `Run` with no decision — never absence, never a parked cover, never a launch. |
| Crash at the binding seam | `Store.Bind`/`Supersede` + `Hooks` | `TestBindCrashChildSelfTerminates`, `TestBindHookAfterStageAbortsWithoutInstall`, `TestBindHookAfterInstallKeepsTheCommit`, `TestSupersedeCrashSeams` | Driven: a real SIGKILL after the no-replace commit leaves the complete binding and the identical retry reattaches; an aborted stage leaves absence the retry converges from; a post-commit hook error keeps the commit, on both the anchor and the pair-receipt paths. |
| No PID persisted | `Store.Bind` (closed receipt) | `TestBindingPersistsNoPID` | Driven: the stored bytes carry exactly the seven closed members — UUIDv7 instance plus opaque digest — and no PID/handle/socket/endpoint/token member. |
| Stable entrypoint | `Decide` → `terminalbackend.CheckEntrypoint` | `TestDecideTable/refused_entrypoint_raw_provider` | Driven: a raw provider argv refuses `local_precondition_failed`; only `ax pane SESSION_ID` passes. |
| §7.A descriptor supplied and admitted | `BuildDescriptor`, `Decide` → `terminalbackend.AdmitProviderDescriptor` | `TestDescriptorBuildAdmitRoundTrip`, `TestDescriptorMismatchRefuses`, `TestDescriptorGenerationBounds`, `TestDecideTable/refused_descriptor_generation_drift` | Driven: the wrapper supplies the closed descriptor (fresh UUIDv7 instance, never a PID) and admits it against the validated host-local binding before any launch effect — parse, match, then launch, with no caching across the check. Binding, version, and generation drift refuse; generation lengths 0 and 257 refuse while 256 admits. |
| Fencing token passed separately | `Decide` (token projection) → `fencing.LeaseToken.Bind` | `TestDecideLaunchPositive`, `TestDecideForgedTokensRefuse`, `TestObserveLoadsWinningLease` | Driven: launch and reattach carry the sealed token minted by the passed gate, binding `quiesce`/`capture`/`materialize` to the winning triple; parked, refused, and offer decisions carry the zero token, which refuses every provider operation. |
| `session.parked` authored under lease | `EmitParked` → `Emit` → `fencing.AuthorizeMutation` + `sessstate.Reduce` + `sessrepo.AppendEvent`; `Run` pre-checks | `TestEmitParkedRoundTrip`, `TestEmitParkedVocabularyLoop`, `TestEmitParkedUnfoldableRefuses`, `TestEmitUnderLosingLeaseRefuses`, `TestRunParkedEmits`, `TestRunParkedUnfoldableSkipsEmission`, `TestRunRemoteNonInteractiveParksWithoutEmission`, `TestEmitParkedRefusesNonParked` | Driven: verified, locally held, foldable parks author the exact closed payload under the winning lease with current chain linkage (every `Emit` gates through `AuthorizeMutation`; `session.parked` folds only from materializing/stopped/failed/parked); a losing presented tuple refuses at the mutation gate itself (`not_owner` under a remote winner, `stale_owner` under a local successor) on a foldable chain; remote, unverified, and unfolderable parks author nothing rather than manufacturing a collision; launch and refused decisions emit nothing, and emission for a non-parked decision refuses. |
| v4 terminal payloads | `TerminalCreatedPayload`, `ResumedPayload`, `Emit` | `TestTerminalCreatedPayload`, `TestResumedPayload`, `TestEmitTerminalV4RoundTrip` | Driven: both v4 payloads carry exactly the §5.2 members (opaque binding digest, backend identity, evidence IDs; checkpoint plus effective profile pair for resumed), append under schema 4.0.0, and read back; malformed members refuse. |
| §13.1 step 3 durable entry before launch | `Run` → `Store.Bind` | `TestRunLaunchBinds`, `TestRunRestoreCommitted` | Driven: the bootstrap pair binds durably before the launch outcome returns; the binding names the launched instance. Provider launch itself is a later leaf (stated bound). |
| §13.1 step 4 validated plan | `Decide` (mapping + descriptor + token) | `TestLaunchCarriesValidatedPlan` | Driven: every launch carries the resolved adapter mapping, the admitted descriptor, and the sealed token together — no partial plan. Provider argv/env supervision is a later leaf (stated bound). |
| Configuration validated | `Run` → `ValidateConfig` → `config.Load`; `Decide` (config arm) | `TestDecideTable/refused_config_invalid`, `TestRunInvalidConfig` | Driven: a failing configuration load refuses `invalid_config` through the real loader. |
| Session loaded | `Run` → `LoadSession`; `Decide` (session arm) | `TestDecideTable/refused_unknown_session`, `TestRunUnknownSession` | Driven: an unknown SESSION_ID refuses `invalid_arguments`; observation is skipped without a session. |
| After-restore sequence 1-5 | `Decide` in restore mode; `Run` → `LoadNewestCheckpoint` | `TestDecideAfterRestoreSequence`, `TestDecideRemoteOwnerPrecedesMaterialization`, `TestDecideWindowEpochOneOwnerCloses`, `TestRunEpochOneOwnerAfterCheckpoint`, `TestDecidePostTakeoverDivergedBaseLaunches`, `TestRunPostTakeoverRestoreLaunchesWithNewestClosure`, `TestRunPostTakeoverStaleBaseParks`, `TestDecideSuccessorNullFoldParks`, `TestRunSuccessorNullFoldParks` | Driven: winning lease plus valid required materialization resumes — on the epoch-1 owner and on the post-takeover owner from the fold's newest — with the window, admission, and journal all keyed on the fold's newest published checkpoint; materializing, unverified, lease-absent, and successor-null-fold inputs park per steps 3 and 5; interactive remote owners offer per step 4 and non-interactive remote owners park per step 5; authorize precedes materialization. The bounded mesh refresh of step 2 is the caller-reported `Verified` fact (stated bound: no network here). |
| Arm precedence | `Decide` | `TestDecideArmPrecedence` | Driven: an all-faulted input refuses at the configuration arm first; peeling faults walks the documented order. |
| Orchestrated effects | `Run` | `TestRunLaunchBinds`, `TestRunReattaches`, `TestRunParkedEmits`, `TestRunParkedUnfoldableSkipsEmission`, `TestRunRemoteNonInteractiveParksWithoutEmission`, `TestRunPostWindowSupersedes`, `TestRunSupersededPairIdenticalRetry`, `TestRunCreateFromStoppedPostWindow`, `TestRunEpochOneOwnerAfterCheckpoint`, `TestRunNullFoldRestoreParks`, `TestRunNoCkptStorePublishedNewestFails`, `TestRunTornLeasePropagates`, `TestRunRefusedWritesNothing`, `TestObserveLoadsWinningLease`, `TestRunPostWindowSamePairRaceReattaches`, `TestRunSuccessorNullFoldParks`, `TestRunUnfoldableChainFails`, `TestRunNewestAbsentFromStoreFails` | Driven: exactly one effect per outcome — launch binds (recording its own pair receipt post-window, reattaching on a reported replay), reattach returns the pair receipt, locally held foldable parks emit while remote/unfolderable parks emit nothing, refused writes nothing; ownership observes the real winning lease, the fold supplies the newest checkpoint, the run fails when a published newest has no checkpoint store (or the newest is absent from it), and torn leases and unfoldable chains propagate with no decision. |

## Refusal and recovery coverage

Every `N-` row is a genuine narrowing mutant: the gate stays
present and is weakened to admit exactly one member of the class it
must reject, and the named behavioral test fails through the
delivered harness (`internal/axpane/mutant_harness.py`, run as
`PYTHONDONTWRITEBYTECODE=1 python3
internal/axpane/mutant_harness.py`). Whole-arm deletes are not
accepted as narrowing. The `C-` row is the applied harmless
SURVIVED control. Verdicts below are from the evidence run
(`mutants.log` in the producer evidence archive): 53 of 53 rows
match, 52 KILLED plus the SURVIVED control. The story-close rows
(N-parked-winner, N-journal-closure, N-ckpt-corrupt) bring the
harness to 56 of 56 rows match, 55 KILLED plus the SURVIVED
control, verified by two full post-close passes.

| Mutant | Weakening (one member admitted) | Killer test | Verdict |
| --- | --- | --- | --- |
| N-config | Invalid config admitted on launch mode only | `TestDecideTable/refused_config_invalid` | KILLED |
| N-mode | Empty mode admitted; other non-modes refuse | `TestDecideTable/refused_empty_mode` | KILLED |
| N-session | Unknown session admitted on launch mode only | `TestDecideTable/refused_unknown_session` | KILLED |
| N-idempotency | Changed operation admitted on launch mode only (in-window) | `TestDecideTable/refused_idempotency_mismatch` | KILLED |
| N-idempotency-restore | Changed operation admitted on restore mode only (in-window) | `TestDecideRestoreChangedOperationRefuses` | KILLED |
| N-idempotency-bound | Unrecorded pairs on bound sessions admitted, requiring a mismatched receipt | `TestDecideRestoreChangedOperationRefuses` | KILLED |
| N-window-fold | Window closed on launch mode without a published checkpoint | `TestDecideTable/refused_idempotency_mismatch` | KILLED |
| N-caller | `batch` caller admitted; other non-callers refuse | `TestDecideTable/refused_unknown_caller` | KILLED |
| N-backend-op | Restore-mode callers admitted to the create row | `TestDecideRestoreRequiresRestoreCapability` | KILLED |
| N-materialization | Staging journals admitted; absent and other phases park | `TestDecideTable/parked_materialization_not_committed` | KILLED |
| N-materialization-source | Stale source generation admitted; other stale sources park | `TestDecideMaterializationStaleParks` | KILLED |
| N-materialization-nullfold | Restore-mode journals past the null-fold arm (wrong cause surfaces) | `TestDecideJournalNullFoldParks` | KILLED |
| N-successor-nullfold | Restore-mode successor null folds past the implication arm (per-arm null cause surfaces) | `TestDecideSuccessorNullFoldParks/restore_with_requirements` | KILLED |
| N-checkpoint-nullfold | Restore-mode checkpoints past the null-fold arm (wrong cause surfaces) | `TestDecideCheckpointNullFoldParks` | KILLED |
| N-closure-source | Profile closure read from the lease handoff base again | `TestRunPostTakeoverLaunchCarriesNewestClosureYoloToStandard` | KILLED |
| N-checkpoint-absent | Absent checkpoints admitted; bad bytes still park | `TestDecideTable/parked_checkpoint_absent` | KILLED |
| N-checkpoint-prefix | Same-prefix digests admitted; full compare otherwise | `TestDecideTable/parked_checkpoint_flipped_digest` | KILLED |
| N-checkpoint-newest | Restore-mode non-newest checkpoints admitted; launch still parks | `TestDecideCheckpointNotLeaseParks` | KILLED |
| N-checkpoint-session | Foreign fixture session admitted; other foreign still park | `TestDecideForeignCheckpointParks` | KILLED |
| N-identity-version | 0.147.0 identity mismatches admitted; other versions refuse | `TestDecideTable/refused_provider_identity_other_provider` | KILLED |
| N-discovery | Empty-home discovery admitted; other proofs still verify | `TestDecideTable/refused_discovery_empty_home` | KILLED |
| N-smoke-verdict | Failing verdicts admitted; gated/unsupported/unknown refuse | `TestDecideTable/refused_smoke_failing_verdict` | KILLED |
| N-smoke-version | Version drift admitted; provider/platform/arch still bind | `TestDecideTable/refused_smoke_wrong_build` | KILLED |
| N-smoke-mode | Required smoke skipped on launch only | `TestDecideTable/refused_smoke_absent` | KILLED |
| N-profile-derive | Corrupt derivation admitted for codex (wrong class surfaces) | `TestDecideTable/refused_profile_derivation_corrupt` | KILLED |
| N-profile-closure | Session head derived instead of closure on restore only | `TestDecideResumeProfileUsesClosure` | KILLED |
| N-mapping | Standard mapping gaps admitted; yolo gaps refuse | `TestDecideTable/refused_profile_mapping_unavailable` | KILLED |
| N-realm | Broker-running background callers admitted; other states refuse | `TestDecideTable/refused_realm_background_no_server` | KILLED |
| N-realm-foreground | Foreground credential-requiring callers admitted without the realm row | `TestDecideForegroundCredentialRequiresRealm` | KILLED |
| N-realm-deadmap | Launch-mode dead-realm evidence admitted to the backend class | `TestDecideExpiredRealmRowRefusesUnavailable` | KILLED |
| N-realm-binding | Mismatched test binding admitted; other mismatches refuse | `TestDecideRealmBindingMismatchRefuses/binding` | KILLED |
| N-realm-generation | Stale test generation admitted; other stale refuse | `TestDecideRealmBindingMismatchRefuses/generation` | KILLED |
| N-headless | Durable-holding non-interactive without headless admitted | `TestDecideNonInteractiveRequiresHeadless` | KILLED |
| N-entrypoint | Two-element argv admitted; other arities refuse | `TestDecideTable/refused_entrypoint_raw_provider` | KILLED |
| N-descriptor | Launch-mode descriptor mismatch admitted; restore refuses | `TestDecideTable/refused_descriptor_generation_drift` | KILLED |
| N-attach | Remote attach without the capability admitted | `TestDecideTable/takeover_offer_attach_unproven` | KILLED |
| N-reattach | Launch selected over reattach on launch mode only | `TestDecideTable/reattach_same_pair` | KILLED |
| N-binding-prefix | Same-prefix changed operations admitted at the race | `TestBindConcurrentChangedOperationRefuses` | KILLED |
| N-binding-session | Same-prefix foreign sessions admitted at status | `TestStatusSessionMismatchRefuses` | KILLED |
| N-bind-first | Changed test operation admitted past the anchor pre-check (wrong error surfaces) | `TestBindChangedOperationRefusesMismatch` | KILLED |
| N-supersede-no-replace | Test pair receipt replaced on the commit race | `TestSupersedeConcurrentSamePairReplays` | KILLED |
| N-parked-reason | Five-letter reasons admitted; vocabulary otherwise closed | `TestParkedPayloadVocabulary` | KILLED |
| N-emit-mutation | Unverified emissions admitted; verified refusals refuse | `TestEmitParkedVocabularyLoop/refuses_unverified` | KILLED |
| N-emit-losing-epoch | Losing-epoch emissions admitted; same-or-winning-epoch refusals refuse | `TestEmitUnderLosingLeaseRefuses` | KILLED |
| N-parked-fold | Unfolderable restore_policy admitted; other unfolderable refuse | `TestEmitParkedUnfoldableRefuses` | KILLED |
| N-run-fold | Run admits unfolderable restore_policy; other still skip | `TestRunParkedUnfoldableSkipsEmission` | KILLED |
| N-run-ckpt-store | Launch-mode runs with a published newest and no store admitted (head pair launches) | `TestRunNoCkptStorePublishedNewestFails` | KILLED |
| N-window-lease-checkpoint | Window closed on the lease record's checkpoint with a null fold | `TestRunWindowLeaseCheckpointNullFoldRefuses` | KILLED |
| N-run-fold-error | Unfoldable chain laundered as a null fold | `TestRunUnfoldableChainFails` | KILLED |
| N-supersede-replay | Post-window replay keeps launch in restore mode | `TestRunPostWindowSamePairRaceReattaches` | KILLED |
| N-identity-session | Foreign fixture session's identity record admitted | `TestDecideIdentityForeignSessionRefuses` | KILLED |
| N-run-newest-absent | Restore-mode runs with the newest absent from the store admitted (head pair launches) | `TestRunNewestAbsentFromStoreFails/restore` | KILLED |
| N-parked-winner | Stale_owner winner mismatches admitted (story-close P3-4) | `TestEmitParkedWinningLeaseMismatchRefuses` | KILLED |
| N-journal-closure | Handoff base read on journal-only restore with a checkpoint winner (story-close P3-1) | `TestRunJournalOnlyRestoreClosureFromNewest` | KILLED |
| N-ckpt-corrupt | Invalid-closure class laundered as absence (story-close P3-5) | `TestRunCorruptCheckpointBlobIsNotAbsence` | KILLED |
| C-doc-comment | Comment-only control edit | `TestRealmCarriesNoCachedEvidence` | SURVIVED |

No token-preserving source-text mutant applies: this package
contains no gate that inspects source text, so the additional
behavioral-suite mutant the evidence contract names for such gates
has no target here. The standalone `CheckResumeTuple` call in
`checkProviderIdentity` is fail-fast ordering over the same gate
`VerifyIdentityBuild` enforces internally, so weakening it alone is
unobservable by construction; likewise the `admitted.Has` realm
pre-check is fail-fast before the evidence cross-bind, so weakening
it alone is unobservable and the binding mutants cover the gate;
the N-identity-version mutant covers the provider-identity gate as
a whole. The fencing entry selection (`AuthorizeRestore` vs
`AuthorizeActivation`) shares one launch-class core in the
`fencing` owner, so no behavioral delta exists to mutate; the
mode-selected backend capability row carries the mode mutant
(N-backend-op).

## Crash and idempotency evidence

The bootstrap binding is the only durable write. Every receipt
installs under the landed discipline: same-directory staging,
0600, fsync, atomic link as the no-replace commit, staging
cleanup, directory fsync (skipped, never faked, on Windows).
Readers see absence or the complete receipt, never a torn prefix.
The session's first binding anchors the in-window contention and
every recorded pair keeps its own receipt, so post-window pairs
record alongside — never over — their superseded receipts; a crash
between the anchor and pair installs converges the pair from the
anchor bytes on retry. Crash hooks arm both boundaries
(`AfterStage` before the commit, `AfterInstall` after it) on both
install paths; the real-kill test
(`TestBindCrashChildSelfTerminates`) SIGKILLs the child between
commit and directory sync and proves the complete binding plus a
converging identical retry. Parked events append through the
`sessrepo` owner, whose own crash evidence stays with that
package; this leaf proves linkage and round-trip only.

## Stated bounds

- No `ax` CLI surface, no tmux/ConPTY process control, no provider
  process supervision, no mesh transport, and no GUI/Keychain
  broker live here. Those are modeled callers (their facts arrive
  as `Decide` inputs) and later leaves.
- The bounded mesh lease refresh (after-restore step 2) is the
  caller-reported `Verified` fact, not a network call.
- Signature verification is caller-supplied through
  `terminalbackend.SignatureVerifier`; tests authenticate through
  a generated RSA key.
- Backend liveness (`terminal_backend_unavailable` at the
  operation call site) belongs to the later backend-operation
  leaf: no backend operation exists here, so no availability rule
  is invented.
- The host-local `InstanceBinding` the descriptor must match
  arrives from the modeled backend caller; the wrapper proves
  document-to-binding consistency and never mints backend
  bindings.
- The committed-phase rule for a required materialization is the
  wrapper's stated reading of after-restore step 3 ("the required
  materialization is valid").
- Invalid materialization or checkpoint on an otherwise winning
  local owner parks `restore_policy` per the literal step-5
  reading ("enter parked without launching the provider in all
  other cases"), with the blocking cause in the decision detail.
- Unknown SESSION_ID refuses `invalid_arguments`: the call names
  no logical session. No `unknown_session` wire code is minted.
- The dead-realm refusal mapping applies only when the caller
  needs the realm row and submitted at least one realm evidence
  object that is unusable (expired or bound to a superseded
  generation). With no submitted realm row, or any usable one, a
  reconcile failure keeps its backend class.
- The realm cross-bind accepts when any submitted realm object
  binds this host and build: order-independent by scan, pinned in
  both evidence orders plus the all-foreign refusal.
- Binding the successor lease's handoff base to a checkpoint the
  chain published is the takeover leaf's obligation: this leaf
  asserts only the implication "a checkpoint-carrying winner means
  the fold published a newest" (one site, directly after fencing)
  and compares the base against nothing — a diverged base is the
  ordinary post-takeover state, not a refusal.
- The `realmRowUsable` liveness and generation predicate restates
  the reconcile rule as a stated bound: the aggregate `Reconcile`
  error carries no per-object identity (liveness shares the
  generic mismatch code), so no per-row usability verdict can be
  read off it. The copy is classification-only and fail-closed in
  both directions — it selects the refusal class, never admission.

## Story close (final leaf TASK-260830-2056mm)

The rev4 review's residual P3-1, P3-4, P3-5 are closed here; the AC
row count stays 43 of 43 (the closures strengthen existing rows,
they add no new contract):

- P3-4 production fix: `EmitParked` requires
  `decision.WinningLeaseID == params.LeaseID` (`events.go`) and
  pins it with `TestEmitParkedWinningLeaseMismatchRefuses` and
  the `N-parked-winner` narrowing row. Through `Run` both leases
  come from `observation.Winner`, so the fix changes no `Run`
  behavior. `TestEmitUnderLosingLeaseRefuses` keeps its
  assertions; its two decision fixtures now name the authoring
  lease (the previously lying winner pair is exactly the shape
  P3-4 forbids, covered by the new test).
- P3-1: `TestRunJournalOnlyRestoreClosureFromNewest` (the
  reviewer's witness, committed) pins the journal-only restore
  closure through `Run`; `N-journal-closure` mirrors the
  reviewer's plant.
- P3-5: `TestRunCorruptCheckpointBlobIsNotAbsence` (the
  reviewer's witness, committed) pins the corrupt-blob load
  error through `Run` in both modes; `N-ckpt-corrupt` launders
  exactly the invalid-closure class as absence.
- P3-2 bound (owner-stated): `Decide` is a pure function over
  caller-loaded facts and performs no closure loading itself;
  the missing-closure input shape is unreachable from any
  `Run` path, which loads the closure before deciding.
- P3-3 bound (owner-stated): closure loading runs before the
  fencing decision on every path, so a remote-owned session whose
  newest checkpoint is absent from the local store (replica lag)
  or with no store bound fails the run with the closure/store
  error instead of reaching the attach_remote/takeover offer.
  The launch-class paths for a remote winner are unreachable
  until the local store holds the newest; the bound is
  fail-closed (no decision launches without its closure), never
  a silent proceed. Deferring the load to launch-class
  decisions is future work, not this leaf.

## Story-final BUG-260917-3lddu0: losing-lease profile source

The production append gate is owned by `sessrepo`; this package supplies
the wrapper entries that must not bypass it. `TestLosingLeaseProfileEventIgnored`
drives `world.repo.AppendEvent` after a successor lease wins while the
tail still belongs to lease A, asserts the literal `ErrStaleLease`, then
loads the profile through `LoadProfile`/`sessprofile.Derive` and runs the
production `Run` path. The effective result stays standard with no
losing-event source, so the archived probe-15 yolo/
`--dangerously-bypass-approvals-and-sandbox` outcome cannot be reproduced.

The four wrapper entry cells are independently driven by
`TestEmitReachesAppendAdmissionGateAfterStaleObservation`,
`TestEmitParkedReachesAppendAdmissionGateAfterStaleObservation`,
`TestEmitReachesSameEpochAppendAdmissionGate`, and
`TestEmitParkedReachesSameEpochAppendAdmissionGate`; their corresponding
lower-epoch and same-epoch narrowing mutants are recorded in the
sessrepo matrix. The same-epoch cells cover both loser-ID directions and
read the preserved event blob back from the repository.

The four landed tests below changed fixture meaning under stable names, so
the importer test-status grid cannot expose the moves. The complete status
comparison includes 2,018 / 2,030 keys (9 package rows included), or 2,009 /
2,021 actual test rows; the independent runtime outcome grid is recorded in
the task evidence and keys actual `AppendEvent` outcomes.

| Test | Fixture/meaning correction | Moved input class | Spec |
| --- | --- | --- | --- |
| `TestRunPostWindowSupersedes` | `publishCheckpoint` now precedes successor B. | Old A-owned `session.idle` + `session.stopped` after B → `ErrStaleLease`; now pre-takeover admitted. | §5.2/§5.3 |
| `TestRunSupersededPairIdenticalRetry` | Same checkpoint-before-successor correction; retry assertions stay unchanged. | Same lower-epoch A-after-B stop-event class → `ErrStaleLease`; now pre-takeover history. | §5.2/§5.3 |
| `TestRunCreateFromStoppedPostWindow` | Stopped/checkpointed state is made while A owns it, then B follows. | Same lower-epoch A-after-B stop-event class → `ErrStaleLease`; now pre-takeover history. | §5.2/§5.3 |
| `TestLosingLeaseProfileEventIgnored` | The raw production append is now asserted to refuse before the profile/Run checks. | Losing A-owned `profile.changed` after B: admitted/yolo/bypass before → refused/preserved now. | §2.4 + §5.3 |

The first three are graceful-takeover fixture corrections; the old ordering
would hit the new owner gate. The last is the probe-15 regression and names
the old yolo/`--dangerously-bypass-approvals-and-sandbox` outcome.
No production axpane implementation was changed for this leaf. This is route
(b): the append gate closes the profile-source property; independent
lease-aware derivation-side authority is owned by
`STORY-260922-cpkajd` / `TASK-260922-31qyyi` and is not implemented here.
The
`decide.go` doc-comment qualifier that “lapsed grants refuse” remains
untouched and scoped to the local-owner path. The
`observeRemoteWinner` redundancy and grant-less remote-owner product
question remain record-only dispositions, not simplifications or decisions.
