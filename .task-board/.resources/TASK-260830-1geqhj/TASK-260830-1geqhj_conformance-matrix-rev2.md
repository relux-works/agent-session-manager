# TASK-260830-1geqhj conformance matrix rev2 — ax pane enforcement wrapper

Normative authority: relux-works/agent-session-manager-spec v0.5.0
(commit `28bf96d`), worked against `internal/specdoc/SPEC.v0.6.0.md`.
Every clause below is driven through the named production entry by the
named committed test, or is declared a stated bound. AC ratio: 41 of 41 (rev2 rework).

## §4.B Registry, identity, discovery, and trust

| Clause | Production entry | Named test(s) | Verdict |
| --- | --- | --- | --- |
| Manifest/Probe/Evidence closed identity + digest rule | `Decide` → `terminalbackend.ParseManifest/ParseProbe/ParseEvidence` | `TestDecideLaunchPositive` | Driven |
| Manifest-to-Probe reconciliation + keyed relation | `Decide` → `terminalbackend.Reconcile` | `TestDecideLaunchPositive`, `TestDecideRestoreRequiresRestoreCapability` | Driven |
| Generation binding | `Decide` → `Reconcile` | `TestDecideTable/refused_backend_generation_mismatch` (→ the more specific `terminal_backend_stale_generation`) | Driven |
| Protocol/platform membership | `Decide` → `Reconcile` | `TestDecideTable/refused_backend_protocol_membership` | Driven |
| Evidence liveness + signature before admission | `Decide` → `Reconcile` | `TestDecideTable/refused_backend_evidence_expired`, `TestDecideTable/refused_backend_signature_untrusted` | Driven |
| Mismatch fails before activation | `Decide` → `admitBackend` | above four rows | Driven |

## §4.C Lifecycle and operations + §4.1/§4.2 wrapper rule

| Clause | Production entry | Named test(s) | Verdict |
| --- | --- | --- | --- |
| Validate configuration | `Run` → `ValidateConfig` → `config.Load`; `Decide` config arm | `TestDecideTable/refused_config_invalid`, `TestRunInvalidConfig` | Driven |
| Load the logical session | `Run` → `LoadSession`; `Decide` session arm | `TestDecideTable/refused_unknown_session`, `TestRunUnknownSession` | Driven |
| Synchronize lease records when possible | `Run` → `ObserveOwnership` → `fencing.Observe` | `TestObserveLoadsWinningLease`, `TestRunLaunchBinds` | Driven |
| Compare the local fencing token before launching | `Decide` → `fencing.AuthorizeActivation/AuthorizeRestore` | Full fencing table (lower/losing/future/foreign/expired/malformed rows) | Driven |
| Park when remote, ambiguous, or unverified | `Decide` → `decideParked` (step 5: non-interactive remote parks) | `TestDecideTable/parked_*`, `TestDecideTable/attach_remote_interactive`, `TestDecideTable/takeover_offer_*`, `TestDecideRemoteOwnerPrecedesMaterialization` | Driven |
| Idempotent on `(session_id, bootstrap_operation_id)`; window closes on winner checkpoint; post-window supersedes | `Store.Bind`/`Supersede`, `Decide` windowed arm | `TestBindIdenticalRetryReattaches`, `TestDecideTable/reattach_same_pair`, `TestRunReattaches`, `TestDecidePostWindowNewOperationLaunches`, `TestRunPostWindowSupersedes` | Driven |
| Changed operation in-window → `idempotency_mismatch` (both modes) | `Store.Bind`, `Decide` | `TestBindChangedOperationRefusesMismatch`, `TestDecideTable/refused_idempotency_mismatch`, `TestDecideRestoreChangedOperationRefuses`, `TestRunIdempotencyMismatch` | Driven |
| Status + binding prove absence or the one child | `Store.Status` | `TestBindInstallsAndStatusProves`, `TestStatusUnknownIsNotAbsent`, `TestStatusSessionMismatchRefuses` | Driven |
| Commit race is linearizable | `Store.Bind` (EEXIST re-read) | `TestBindConcurrentSameOperationReattaches`, `TestBindConcurrentChangedOperationRefuses`, `TestRunConcurrentLaunchReattaches` | Driven |
| Crash at the binding seam | `Store.Bind` + `Hooks` | `TestBindCrashChildSelfTerminates` (real SIGKILL), `TestBindHookAfterStageAbortsWithoutInstall`, `TestBindHookAfterInstallKeepsTheCommit` | Driven |
| Materialization bound to the lease checkpoint | `Decide` → `checkMaterialization` | `TestDecideMaterializationStaleParks` | Driven |
| Checkpoint is the winning lease's and names this session | `Decide` → `admitCheckpoint` | `TestDecideCheckpointNotLeaseParks`, `TestDecideForeignCheckpointParks` | Driven |
| Never persist or replicate a PID | `Store.Bind` (closed receipt) | `TestBindingPersistsNoPID` | Driven |
| Background caller: admitted realm row bound to binding/build/generation; else `capability_unavailable` | `Decide` → `checkRealm` → `Reconcile` + `axerror.NewRealmEvidenceUnavailable` | `TestDecideTable/refused_realm_background_no_server`, `TestDecideTypedRefusals/capability_unavailable_details`, `TestDecideTable/launch_background_attested_server`, `TestDecideRealmBindingMismatchRefuses` | Driven |
| Cached sentinel / `managername` / boolean never authorize resume | `Decide` → `checkRealm` + `checkSmoke` (behavioral) | `TestRealmCarriesNoCachedEvidence` (cached smoke without realm refuses; stale refuses; admitted launches) | Driven |
| Stable entrypoint `ax pane SESSION_ID` | `Decide` → `terminalbackend.CheckEntrypoint` | `TestDecideTable/refused_entrypoint_raw_provider` | Driven |
| After-restore sequence 1-5 | `Decide` in restore mode | `TestDecideAfterRestoreSequence` (6 scenarios) | Driven |
| Mode selects activation/restore entry | `Decide` → `authorize` | `TestDecideRestoreRequiresRestoreCapability`, `TestDecideTable/launch_restore_committed_checkpoint` | Driven |
| Bounded mesh lease refresh (step 2) | Caller-reported `Verified` fact | — | Stated bound: no network in this leaf |
| Backend operation errors (`terminal_backend_unavailable` at the call site) | — | — | Stated bound: no backend operation exists in this leaf |

## §4.D Closed capabilities and evidence

| Clause | Production entry | Named test(s) | Verdict |
| --- | --- | --- | --- |
| Only exact admitted rows authorize an action; non-interactive create requires `headless_creation` | `Decide` → `terminalbackend.CheckOperation` + headless conditional | `TestDecideTable/refused_backend_capability_unproven`, `TestDecideNonInteractiveRequiresHeadless` | Driven |
| Attach row gates `attach_remote` | `Decide` → `decideParked` + `CheckOperation("attach")` | `TestDecideTable/takeover_offer_attach_unproven` | Driven |
| No capability map invented | `Reconcile` set is the only authority | Structural (every admission flows through `admitBackend`) + unproven row | Driven |
| Mode selects create/restore row | `Decide` → `admitBackend` | `TestDecideRestoreRequiresRestoreCapability` | Driven |

## §5.2 Terminal Events

| Clause | Production entry | Named test(s) | Verdict |
| --- | --- | --- | --- |
| `session.parked` reason vocabulary | `ParkedPayload` | `TestParkedPayloadVocabulary` | Driven |
| Parked authored under the locally held winner iff foldable | `EmitParked` → `Emit` → `AuthorizeMutation` + `Reduce` + `AppendEvent`; `Run` pre-checks | `TestEmitParkedRoundTrip`, `TestEmitParkedVocabularyLoop`, `TestEmitParkedUnfoldableRefuses`, `TestEmitUnderLosingLeaseRefuses`, `TestRunParkedEmits`, `TestRunParkedUnfoldableSkipsEmission` | Driven |
| Parked without local authority authors nothing | `Run` (emission skip) | `TestRunParkedWithoutWinnerSkipsEmission`, `TestRunTakeoverOfferEmitsParked` (remote parks emit nothing) | Driven |
| Non-parked emission refuses | `EmitParked` | `TestEmitParkedRefusesNonParked` | Driven |
| v4 `terminal.created` closed payload + append | `TerminalCreatedPayload`, `Emit` | `TestTerminalCreatedPayload`, `TestEmitTerminalV4RoundTrip` | Driven |
| v4 `session.resumed` closed payload + append | `ResumedPayload`, `Emit` | `TestResumedPayload`, `TestEmitTerminalV4RoundTrip` | Driven |

## §7.A Provider Protocol 3.0.0 Terminal Instance binding

| Clause | Production entry | Named test(s) | Verdict |
| --- | --- | --- | --- |
| Wrapper supplies the closed descriptor (UUIDv7 instance, never a PID) | `BuildDescriptor`, `MintInstanceID` | `TestDescriptorBuildAdmitRoundTrip`, `TestMintInstanceID`, `TestDescriptorRefusesNonUUIDInstance`, `TestDescriptorRefusesBadGeometry` | Driven |
| Provider rejects a mismatched binding before launch/observe | `Decide` → `terminalbackend.AdmitProviderDescriptor` | `TestDescriptorMismatchRefuses`, `TestDecideTable/refused_descriptor_generation_drift` | Driven |
| Generation bound 1..256 | `AdmitProviderDescriptor` | `TestDescriptorGenerationBounds` (0 and 257 refuse, 256 admits) | Driven |
| LeaseToken passed separately; zero token refuses all operations | `Decide` token projection → `fencing.LeaseToken.Bind` | `TestDecideLaunchPositive`, `TestDecideForgedTokensRefuse` | Driven |
| Host-local binding arrives from the backend caller | `Request.HostBinding` input | — | Stated bound: the wrapper proves document-to-binding consistency, never mints backend bindings |

## §13.1 Direct-session launch steps 3-4

| Clause | Production entry | Named test(s) | Verdict |
| --- | --- | --- | --- |
| Step 3: durable terminal entry with the bootstrap operation ID | `Run` → `Store.Bind` | `TestRunLaunchBinds`, `TestRunRestoreCommitted` | Driven |
| Step 4: validated plan (mapping + descriptor + token) on launch | `Decide` | `TestLaunchCarriesValidatedPlan` | Driven |
| Provider launch / argv-env supervision | — | — | Stated bound: later leaf |

## §2.4 Execution profiles (composed authority)

| Clause | Production entry | Named test(s) | Verdict |
| --- | --- | --- | --- |
| Effective persisted profile + source (PROFILE-DIRECT-RESUME: head on launch, closure on resume) | `Decide` → `deriveProfile` → `Derive`/`DeriveForHeads`; `Run` loads winner heads | `TestDecideProfileSourcePinsEvent`, `TestDecideYoloMapping`, `TestDecideResumeProfileUsesClosure`, `TestLosingLeaseProfileEventIgnored` | Driven |
| Corrupt derivation refuses | `Decide` | `TestDecideTable/refused_profile_derivation_corrupt` | Driven |
| Resume fails `profile_mapping_unavailable` when unmappable | `Decide` → `provhost.ResolveMapping` | `TestDecideTable/refused_profile_mapping_unavailable` | Driven |

## Fencing composition detail (via the `fencing` owner)

| Clause | Production entry | Named test(s) | Verdict |
| --- | --- | --- | --- |
| Lower / losing / future epoch | `Decide` → `AuthorizeActivation` | `TestDecideTable/parked_lower_epoch`, `/parked_same_epoch_loser`, `/parked_future_epoch` | Driven |
| Foreign session / expired grant / malformed token | `Decide` → `AuthorizeActivation` | `TestDecideTable/refused_fencing_foreign_session`, `/refused_fencing_expired_grant`, `/refused_fencing_malformed_token` | Driven |
| Absent / ambiguous / unverified / failed handoff | `Decide` → `AuthorizeActivation` | `TestDecideTable/parked_absent_winner`, `/parked_ambiguous`, `/parked_unverified`, `/parked_failed_handoff` | Driven |

## Provider identity composition detail (via the `provhost` owner)

| Clause | Production entry | Named test(s) | Verdict |
| --- | --- | --- | --- |
| Resume tuple gate | `Decide` → `provhost.CheckResumeTuple` | `TestDecideTable/refused_provider_tuple_qwen` | Driven |
| Identity bind to exact build | `Decide` → `provhost.VerifyIdentityBuild` | `TestDecideTable/refused_provider_identity_other_provider` | Driven |
| Discovery bind | `Decide` → `provhost.VerifyIdentityDiscovery` | `TestDecideTable/launch_discovery_bound`, `/refused_discovery_native_mismatch`, `/refused_discovery_root_outside`, `/refused_discovery_empty_home` | Driven |
| Smoke precondition + typed `target_auth_missing` (build/generation/OS bound) | `Decide` → `resumesmoke.VerifyRecord` + `axerror.NewTargetAuthMissing` | `TestDecideTable/refused_smoke_failing_verdict`, `/refused_smoke_absent`, `/refused_smoke_wrong_build`, `TestDecideSmokeTargetBindingRefuses`, `TestRunSmokeRequiredPasses` | Driven |

## Wrapper-rule structure

| Clause | Production entry | Named test(s) | Verdict |
| --- | --- | --- | --- |
| Arm precedence (first fault wins) | `Decide` | `TestDecideArmPrecedence` | Driven |
| Orchestrated effects (bind/supersede / return / conditional emit / nothing) | `Run` | `TestRunLaunchBinds`, `TestRunReattaches`, `TestRunParkedEmits`, `TestRunParkedUnfoldableSkipsEmission`, `TestRunPostWindowSupersedes`, `TestRunRefusedWritesNothing` | Driven |
| Store failure propagates without decision | `Run` | `TestRunTornBindingPropagates`, `TestRunTornLeasePropagates` | Driven |
| Matrix completeness (named tests exist) | `go test -list` | `TestConformanceMatrixCompleteness` | Driven |
