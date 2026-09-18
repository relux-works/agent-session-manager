# TASK-260830-1geqhj rev4 — conformance matrix

Authority: spec v0.5.0 (`28bf96d`), worked against
`internal/specdoc/SPEC.v0.7.0.md`. Every row names the production
entry and the committed test that drives it; every named test was
verified present via `go test -list` (see `ratio-check.log` in
the evidence archive — the two phantom names from the rev3 matrix
are replaced with the real drivers below). Full clause-to-test
evidence with stated bounds:
`internal/axpane/TRACEABILITY.md` (43/43 AC rows, counted).

## Wrapper rule (§4.C, §4.1, §4.2)

| Clause | Production entry | Committed test |
| --- | --- | --- |
| Config validity gates every path | `Decide` → `ConfigErr` | `TestDecideTable/refused_config_invalid` |
| Unknown session refuses | `Decide`; `Run` → `LoadSession` | `TestDecideTable/refused_unknown_session`, `TestRunUnknownSession` |
| Mode selects activation/restore entry | `Decide` → `authorize` | `TestDecideTable/launch_restore_committed_checkpoint` |
| In-window changed op refuses (both modes) | `Decide` idempotency arm | `TestDecideTable/refused_idempotency_mismatch`, `TestDecideRestoreChangedOperationRefuses` |
| Window closes on fold newest (incl. epoch-1 owner); never on the lease | `Decide` → `bootstrapWindowClosed`; `Run` → `LoadNewestCheckpoint` | `TestDecideWindowEpochOneOwnerCloses`, `TestRunEpochOneOwnerAfterCheckpoint`, `TestRunWindowLeaseCheckpointNullFoldRefuses` |
| Post-window new op records + launches (both modes) | `Run` → `Store.Supersede` | `TestRunPostWindowSupersedes`, `TestRunCreateFromStoppedPostWindow` |
| Identical retry replays any recorded pair | `Store.Lookup`; `Run` reattach | `TestRunSupersededPairIdenticalRetry`, `TestSupersedeKeepsReceipts` |
| Supersede is no-replace per pair + reports replay | `Store.Supersede` → `installPair`; `Run` flip | `TestSupersedeConcurrentDifferentOps`, `TestSupersedeConcurrentSamePairReplays`, `TestRunPostWindowSamePairRaceReattaches` |
| Crash seams converge (both install paths) | `Store` + `Hooks` | `TestBindCrashChildSelfTerminates`, `TestSupersedeCrashSeams` |
| Unknown is not absent (anchor/pair/lease/fold) | `Store.Status`/`Lookup`/`SessionBound`; `Run` | `TestStatusUnknownIsNotAbsent`, `TestRunTornBindingPropagates`, `TestRunTornLeasePropagates`, `TestRunUnfoldableChainFails` |
| After-restore 1-5 (fencing precedes materialization; both owners) | `Decide` restore mode | `TestDecideAfterRestoreSequence`, `TestDecideRemoteOwnerPrecedesMaterialization`, `TestDecidePostTakeoverDivergedBaseLaunches`, `TestRunPostTakeoverRestoreLaunchesWithNewestClosure` |
| Remote parks / offers, no remote emission | `Decide` → `decideParked`; `Run` | `TestRunRemoteNonInteractiveParksWithoutEmission` |
| Credential conditional, every caller | `Decide` → `checkRealm` | `TestDecideForegroundCredentialRequiresRealm`, `TestRunSmokeRequiredPasses` |
| Realm row binds host/build/generation; any-match | `Decide` → `checkRealmBinding` | `TestDecideRealmEvidenceOrderIndependent`, `TestDecideRealmBindingMismatchRefuses` |
| Dead realm row → typed capability_unavailable | `Decide` → `deadRealmRefusal` | `TestDecideExpiredRealmRowRefusesUnavailable`, `TestDecidePreRebootRealmRowRefusesUnavailable` |
| Live row + backend fault keeps backend class | `Decide` → `admitBackend` | `TestDecideLiveRealmRowBackendFaultStands` |
| Cached evidence never authorizes | `Decide` → `checkRealm` + `checkSmoke` | `TestRealmCarriesNoCachedEvidence` |
| Non-interactive create requires headless_creation | `Decide` → backend conditionals | `TestDecideNonInteractiveRequiresHeadless` |

## Backend / capabilities (§4.B, §4.D)

| Clause | Production entry | Committed test |
| --- | --- | --- |
| Manifest/probe/evidence identity + reconcile | `Decide` → `admitBackend` | `TestDecideTable` backend rows |
| Only admitted rows authorize | `terminalbackend` via `Decide` | `TestDecideTable`, `TestDecideTable/refused_backend_capability_unproven` |
| Operation, entrypoint, descriptor admission | `Decide` | `TestDecideRestoreRequiresRestoreCapability`, entrypoint/descriptor rows |

## Terminal events (§5.2)

| Clause | Production entry | Committed test |
| --- | --- | --- |
| session.parked vocabulary + authorship under lease | `EmitParked` → `Emit` → `AuthorizeMutation` | `TestEmitParkedRoundTrip`, `TestEmitParkedVocabularyLoop`, `TestRunParkedEmits` |
| Losing-lease emission refused at mutation gate | `Emit` → `AuthorizeMutation` | `TestEmitUnderLosingLeaseRefuses` |
| Remote/unverifiable/unfolderable parks emit nothing | `Run` pre-checks; `Emit` fold gate | `TestRunParkedUnfoldableSkipsEmission`, `TestEmitParkedUnfoldableRefuses` |
| v4 terminal binding payloads | `TerminalCreatedPayload`, `ResumedPayload`, `Emit` | `TestTerminalCreatedPayload`, `TestResumedPayload`, `TestEmitTerminalV4RoundTrip` |

## Newest checkpoint (§5.7, §13.11, §14.4)

| Clause | Production entry | Committed test |
| --- | --- | --- |
| Newest derived from chain fold | `LoadNewestCheckpoint` → `sessstate.Reduce` | `TestRunEpochOneOwnerAfterCheckpoint` |
| Null fold + requirement parks (journal/checkpoint) | `checkMaterialization` / `admitCheckpoint` | `TestDecideJournalNullFoldParks`, `TestDecideCheckpointNullFoldParks`, `TestRunNullFoldRestoreParks` |
| Unpublished checkpoint parks | `admitCheckpoint` newest arm | `TestDecideCheckpointUnpublishedParks` |
| Successor null fold parks at the one implication arm | `Decide` post-fencing arm | `TestDecideSuccessorNullFoldParks`, `TestRunSuccessorNullFoldParks` |
| Diverged handoff base launches from the newest; stale base parks | `checkMaterialization` / `admitCheckpoint` (fold-only) | `TestDecidePostTakeoverDivergedBaseLaunches`, `TestRunPostTakeoverRestoreLaunchesWithNewestClosure`, `TestRunPostTakeoverStaleBaseParks` |
| Committed newest-sourced journal resumes | `checkMaterialization` | `TestRunRestoreCommitted`, step3 |

## Profile (§2.4) and descriptor (§7.A)

| Clause | Production entry | Committed test |
| --- | --- | --- |
| Effective profile + source (required/newest/head) | `deriveProfile`; `Run` → `LoadProfile` | `TestDecideProfileSourcePinsEvent`, `TestDecideResumeProfileUsesClosure`, `TestRunPostTakeoverLaunchCarriesNewestClosureStandardToYolo`, `TestRunPostTakeoverLaunchCarriesNewestClosureYoloToStandard` |
| Unknown checkpoint store fails the run | `Run` guard | `TestRunNoCkptStorePublishedNewestFails`, `TestRunNewestAbsentFromStoreFails` |
| Mapping refusal keeps tuple class | `Decide` arm order | `TestDecideYoloMapping` |
| Identity record binds the session | `Decide` → `checkIdentitySession` | `TestDecideIdentityForeignSessionRefuses` |
| Binding descriptor admits + round-trips | `AdmitProviderDescriptor`, `BuildDescriptor` | descriptor tests |

## Direct-session launch (§13.1 steps 3–4)

| Clause | Production entry | Committed test |
| --- | --- | --- |
| Durable terminal entry with op ID, then provider launch | `Run` launch path | `TestRunLaunchBinds`, `TestRunRestoreCommitted` |
| One effect per outcome; refused writes nothing | `Run` | `TestRunRefusedWritesNothing`, `TestRunLaunchBinds` |

## Negative-evidence index (narrowing mutants)

53/53 harness rows: 52 KILLED + control (`mutants/pass1/`,
`mutants/pass2-summary.log`). One narrowing mutant per P1-1
conditional (`N-window-fold`, `N-checkpoint-nullfold`,
`N-materialization-nullfold`, `N-checkpoint-newest`,
`N-successor-nullfold`, `N-closure-source`, `N-run-ckpt-store`,
`N-run-newest-absent`), per receipt gate (`N-bind-first`,
`N-supersede-no-replace`, `N-supersede-replay`,
`N-binding-prefix`, `N-window-lease-checkpoint`), per realm gate
(`N-realm`, `N-realm-foreground`, `N-realm-deadmap`,
`N-realm-binding`, `N-realm-generation`), per emission gate
(`N-emit-mutation`, `N-emit-losing-epoch`, `N-parked-fold`,
`N-run-fold`), the fold-error gate (`N-run-fold-error`), the
identity gate (`N-identity-session`), and one per remaining
decision arm (see TRACEABILITY.md mutant table).
