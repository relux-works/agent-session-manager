# TASK-260830-1geqhj rev3 — conformance matrix

Authority: spec v0.5.0 (`28bf96d`), worked against
`internal/specdoc/SPEC.v0.7.0.md`. Every row names the production
entry and the committed test that drives it. Full clause-to-test
evidence with stated bounds:
`internal/axpane/TRACEABILITY.md` (43/43 AC rows).

## Wrapper rule (§4.C, §4.1, §4.2)

| Clause | Production entry | Committed test |
| --- | --- | --- |
| Config validity gates every path | `Decide` → `ConfigErr` | `TestDecideTable/refused_config_invalid` |
| Unknown session refuses | `Decide`; `Run` → `LoadSession` | `TestDecideTable/refused_unknown_session`, `TestRunUnknownSession` |
| Mode selects activation/restore entry | `Decide` → `authorize` | `TestDecideTable/launch_restore_committed_checkpoint` |
| In-window changed op refuses (both modes) | `Decide` idempotency arm | `TestDecideTable/refused_idempotency_mismatch`, `TestDecideRestoreChangedOperationRefuses` |
| Window closes on fold newest (incl. epoch-1 owner) | `Decide` → `bootstrapWindowClosed`; `Run` → `LoadNewestCheckpoint` | `TestDecideWindowEpochOneOwnerCloses`, `TestRunEpochOneOwnerAfterCheckpoint` |
| Post-window new op records + launches (both modes) | `Run` → `Store.Supersede` | `TestRunPostWindowSupersedes`, `TestRunCreateFromStoppedPostWindow` |
| Identical retry replays any recorded pair | `Store.Lookup`; `Run` reattach | `TestRunSupersededPairIdenticalRetry`, `TestSupersedeKeepsReceipts` |
| Supersede is no-replace per pair | `Store.Supersede` → `installPair` | `TestSupersedeConcurrentDifferentOps`, `TestSupersedeConcurrentSamePairReplays` |
| Crash seams converge (both install paths) | `Store` + `Hooks` | `TestBindCrashChildSelfTerminates`, `TestSupersedeCrashSeams` |
| Unknown is not absent (anchor/pair/lease) | `Store.Status`/`Lookup`/`SessionBound`; `Run` | `TestStatusUnknownIsNotAbsent`, `TestRunTornBindingPropagates`, `TestRunTornLeasePropagates` |
| After-restore 1-5 (fencing precedes materialization) | `Decide` restore mode | `TestDecideAfterRestoreSequence`, `TestDecideRemoteOwnerPrecedesMaterialization` |
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
| Only admitted rows authorize | `terminalbackend` via `Decide` | `TestDecideTable`, `TestTerminalBackendRefusesUnproven` |
| Operation, entrypoint, descriptor admission | `Decide` | `TestDecideRestoreRequiresRestoreCapability`, entrypoint/descriptor rows |

## Terminal events (§5.2)

| Clause | Production entry | Committed test |
| --- | --- | --- |
| session.parked vocabulary + authorship under lease | `EmitParked` → `Emit` → `AuthorizeMutation` | `TestEmitParkedRoundTrip`, `TestEmitParkedVocabularyLoop`, `TestRunParkedEmits` |
| Losing-lease emission refused at mutation gate | `Emit` → `AuthorizeMutation` | `TestEmitUnderLosingLeaseRefuses` |
| Remote/unverifiable/unfolderable parks emit nothing | `Run` pre-checks; `Emit` fold gate | `TestRunParkedUnfoldableSkipsEmission`, `TestEmitParkedUnfoldableRefuses` |
| v4 terminal binding payloads | `BuildDescriptor` fixtures | `TestV4TerminalPayloadsParse` |

## Newest checkpoint (§5.7, §13.11, §14.4)

| Clause | Production entry | Committed test |
| --- | --- | --- |
| Newest derived from chain fold | `LoadNewestCheckpoint` → `sessstate.Reduce` | `TestRunEpochOneOwnerAfterCheckpoint` |
| Null fold + requirement parks (journal/checkpoint) | `checkMaterialization` / `admitCheckpoint` | `TestDecideJournalNullFoldParks`, `TestDecideCheckpointNullFoldParks`, `TestRunNullFoldRestoreParks` |
| Unpublished checkpoint parks | `admitCheckpoint` newest arm | `TestDecideCheckpointUnpublishedParks` |
| Lease/fold disagreement parks | `leaseCheckpointAgrees` | `TestDecideCheckpointLeaseFoldDisagreeParks`, `TestDecideJournalLeaseFoldDisagreeParks` |
| Committed newest-sourced journal resumes | `checkMaterialization` | `TestRunRestoreCommitted`, step3 |

## Profile (§2.4) and descriptor (§7.A)

| Clause | Production entry | Committed test |
| --- | --- | --- |
| Effective profile + source (head/closure) | `deriveProfile`; `Run` → `LoadProfile` | `TestDecideProfileSourcePinsEvent`, `TestDecideResumeProfileUsesClosure` |
| Unknown checkpoint store fails the run | `Run` guard | `TestRunNoCkptStoreWinnerCheckpointFails` |
| Mapping refusal keeps tuple class | `Decide` arm order | `TestDecideYoloMapping` |
| Binding descriptor admits + round-trips | `AdmitProviderDescriptor`, `BuildDescriptor` | descriptor tests |

## Direct-session launch (§13.1 steps 3–4)

| Clause | Production entry | Committed test |
| --- | --- | --- |
| Durable terminal entry with op ID, then provider launch | `Run` launch path | `TestRunLaunchBinds`, `TestRunRestoreCommitted` |
| One effect per outcome; refused writes nothing | `Run` | `TestRunRefusedWritesNothing`, `TestRunLaunchBinds` |

## Negative-evidence index (narrowing mutants)

48/48 harness rows: 47 KILLED + control (`mutants.log`). One
narrowing mutant per P1-1 conditional (`N-window-fold`,
`N-checkpoint-nullfold`, `N-materialization-nullfold`,
`N-checkpoint-newest`, `N-checkpoint-agreement`,
`N-materialization-agreement`), per receipt gate
(`N-bind-first`, `N-supersede-no-replace`, `N-binding-prefix`),
per realm gate (`N-realm`, `N-realm-foreground`,
`N-realm-deadmap`, `N-realm-binding`, `N-realm-generation`),
per emission gate (`N-emit-mutation`, `N-emit-losing-epoch`,
`N-parked-fold`, `N-run-fold`), the store guard
(`N-run-ckpt-store`), and one per remaining decision arm (see
TRACEABILITY.md mutant table).
