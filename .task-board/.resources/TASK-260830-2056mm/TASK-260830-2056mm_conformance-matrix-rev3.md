# TASK-260830-2056mm conformance matrix — terminal binding events and recovery (rev3)

Normative source: SPEC.v0.7.0 (§4.1, §4.B, §4.D, §5.2, §7.A).
AC ratio: **65 of 66 rows driven** through the named production entry by the
named committed test. Row 57 is a stated bound (payload census — no
v1-v3-only reader exists on trunk to drive). Rows 61–66 are the rev3
rework rows. Owner `internal/termbind` unless noted.

## AC rows (from internal/termbind/TRACEABILITY.md)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 1 | UUIDv7 admitted as terminal instance identity | CheckInstanceIdentity | TestCheckInstanceIdentityAdmitsUUIDv7 |
| 2 | Eight forbidden forms refuse terminal_backend_protocol_error | CheckInstanceIdentity | TestCheckInstanceIdentityRefusesForbiddenForms (8) |
| 3 | Closed Binding fixture admitted with literal members | ParseTerminalBinding | TestParseTerminalBindingAdmitsClosedFixture |
| 4 | Eight forbidden forms in terminal_instance_id refuse pinned code | ParseTerminalBinding | TestParseTerminalBindingRefusesForbiddenIdentity (8) |
| 5 | Binding closed shape refuses (missing/extra/dup member, wrong schema) | ParseTerminalBinding | TestParseTerminalBindingClosedShape (4) |
| 6 | backend_generation 256 admits, 0 and 257 refuse | ParseTerminalBinding | .../generation_bounds |
| 7 | native_reference 512 admits, 0 and 513 refuse | ParseTerminalBinding | .../native_reference_bounds |
| 8 | Tampered member / wrong self ID refuse manifest_probe_mismatch | ParseTerminalBinding | ...(tampered, wrong_self_id) |
| 9 | Binding identity agrees with landed manifest admission | bindingIdentity via ParseManifest | TestBindingIdentityAgreesWithLandedAdmission |
| 10 | Eight forbidden forms in Provider Protocol 3 descriptor refuse | AdmitProviderDescriptor (adopted landed) | TestAdmitDescriptorRefusesForbiddenIdentity (8) |
| 11 | Eight forbidden forms in mutation context refuse pinned code (documented redundancy: landed refuses same code; wrapper adds Go type) | AdmitMutationContext | TestAdmitMutationContextRefusesForbiddenIdentity (8) |
| 12 | Eight forbidden forms in status body refuse; null admitted (same redundancy) | AdmitStatusBody | TestAdmitStatusBodyRefusesForbiddenIdentity (8+null) |
| 13 | Non-string identity members delegate to landed shape authority | AdmitMutationContext, AdmitStatusBody | TestAdmitSurfacesDelegateShapeFaults (2) |
| 14 | Eight forbidden forms in bootstrap binding refuse, persist nothing | Store.Bind (adopted axpane) | TestBootstrapBindingRefusesForbiddenIdentity (8) |
| 15 | CLI Result 4.0.0 identity surface: stated bound, tripwire-pinned | VersionForCommand (cited cliresult) | TestCLIResult4IdentityIsUnimplementedBound |
| 16 | Coherent universe resolves to manifest/probe/evidence/admission | ResolveEvidence | TestResolveEvidenceAdmitsUniverse |
| 17 | Empty and 257-element sets refuse the 1..256 bound | ResolveEvidence | TestResolveEvidenceShapeBounds (empty, 257) |
| 18 | Unsorted/duplicate/non-digest sets refuse | ResolveEvidence | TestResolveEvidenceShapeBounds (3) |
| 19 | One digest passes shape, refuses at partition | ResolveEvidence | .../one_passes_shape_to_partition |
| 20 | 256 digests pass shape, refuse at lookup | ResolveEvidence | .../256_passes_shape_to_lookup |
| 21 | Eleven foreign resolution targets refuse mismatch class | ResolveEvidence | TestResolveEvidenceRefusesForeignKinds (11) |
| 22 | Missing/doubled manifest/probe refuses cardinality | ResolveEvidence | TestResolveEvidenceRequiresManifestAndProbe (4) |
| 23 | Foreign-backend / protocol-drift evidence refuses at gate | ResolveEvidence | TestResolveEvidenceBindsEventTuple (2) |
| 24 | Document substitution under another ID refuses | ResolveEvidence | TestResolveEvidenceRejectsSubstitution (2) |
| 25 | Admission faults surface landed (claim/forgery/expiry/drift) | ResolveEvidence via Registry.AdmitProbe | TestResolveEvidenceDrivesLandedAdmission (4) |
| 26 | Resolved objects carry no live-process fact | ResolveEvidence | TestResolvedObjectsCarryNoLiveFacts |
| 27 | terminal.created v4 round-trips through append path | EmitTerminalCreated | TestEmitTerminalCreatedRoundTrip |
| 28 | session.resumed v4 round-trips (null+carried source); bad profile refuses | EmitResumed | TestEmitResumedRoundTrip (3) |
| 29 | Unsorted/empty evidence refuse at emission entries | EmitTerminalCreated | TestEmitClosedShapeRefusals (2) |
| 30 | Missing/extra payload members refuse at canonical boundary | Emit (adopted axpane) | TestEmitClosedShapeRefusals (3 append) |
| 31 | Unresolvable/foreign evidence refuses inside both entries | EmitTerminalCreated, EmitResumed | TestEmitResolvesEvidence (3) |
| 32 | Losing lease refuses, appends nothing | EmitTerminalCreated via Emit | TestEmitUnderLosingLeaseRefuses |
| 33 | Byte-identical retry replays the one event | EmitTerminalCreated | TestEmitAppendsIdempotently |
| 34 | Emitted bytes carry opaque digest, no binding secret | EmitTerminalCreated | TestEmittedEventCarriesNoBindingObject |
| 35 | No binding + proven absence → absence with replay_same | RecoverCreate | TestRecoverCreateProvesAbsence |
| 36 | Binding + matching live status → the ONE child | RecoverCreate | TestRecoverCreateIdentifiesOneChild (2) |
| 37 | Duplicate retry yields identical verdict | RecoverCreate | TestRecoverCreateDuplicateRetrySameVerdict |
| 38 | Changed operation in open window refuses idempotency_mismatch | RecoverCreate | TestRecoverCreateChangedOperationInWindowRefuses |
| 39 | Changed operation in closed window proceeds | RecoverCreate | TestRecoverCreateChangedOperationPostWindowProceeds |
| 40 | Lost create result recovers the ONE recorded child | RecoverCreate | TestRecoverCreateLostResult |
| 41 | Five unprovable shapes → unavailable+status_first, never absence | RecoverCreate | TestRecoverCreateNeverClaimsAbsent (5) |
| 42 | Recorded child + backend-absent observation replays same pair | RecoverCreate | TestRecoverCreateRecordedChildBackendAbsentReplays |
| 43 | Status read failure is error, never absence | RecoverCreate | TestRecoverCreateUnknownIsNotAbsent |
| 44 | Attach records/replays one receipt per client pair | AttachStore.Attach/Lookup | TestAttachRoundTrip |
| 45 | Identical retry replays recorded receipt | AttachStore.Attach | TestAttachIdenticalRetryReplays |
| 46 | Second client records alongside, never over | AttachStore.Attach | TestAttachSecondClientRecordsAlongside |
| 47 | Transport/input conflict refuses idempotency_mismatch | AttachStore.Attach | TestAttachConflictRefusesMismatch |
| 48 | Sixteen forbidden identity values refuse pinned code | AttachStore.Attach | TestAttachRefusesForbiddenIdentity (16) |
| 49 | Unknown transport refuses; relay refuses by landed policy | AttachStore.Attach | TestAttachTransportVocabulary (2) |
| 50 | Expired/mismatched/malformed policy refuses on every call | AttachStore.Attach | TestAttachRequiresLiveAuth (4) |
| 51 | Attach emits neither event nor lease/fencing change | AttachStore.Attach | TestAttachEmitsNeitherEventNorStateChange |
| 52 | Torn receipt is error, never absence | AttachStore.Attach/Lookup | TestAttachUnknownIsNotAbsent |
| 53 | Foreign-session lookup refuses | AttachStore.Lookup | TestAttachSessionMismatchRefuses |
| 54 | Takeover: admitted backend, new digest+event, append-only, no fork | EmitTerminalCreated, EmitResumed | TestTakeoverSelectsNewBackendNewEvent |
| 55 | v1 and v4 terminal events derive exactly their own surface | Reduce (adopted sessstate) | TestVersionSelectionBothDirections (3) |
| 56 | v4 retained verbatim, no lower-version replacement | GetEvent/ListEvents, Reader.List (adopted sessrepo/sessquery) | TestV4RetainedAsImmutableHistory |
| 57 | STATED BOUND: v4 carries no v1 derivation member (census); v1-only-reader inertness undrivable, no v1-v3-only reader on trunk | DecodeEvent (adopted sessstate) | TestV1ReaderSeesV4AsInert (census) |
| 58 | SIGKILL after receipt commit leaves complete receipt; retry replays | AttachStore.Attach | TestAttachCrashChildSelfTerminates (unix) |
| 59 | SIGKILL between effect and result recovers the ONE child | RecoverCreate | TestRecoverCreateAfterKillRecoversChild (unix) |
| 60 | Before-write abort leaves no event; after-commit keeps+replays | EmitTerminalCreated | TestEmitAppendHookSeams (2) |
| 61 | Every Binding member arm refuses with code+detail (all 15 members, frame, trailing) | ParseTerminalBinding | TestParseTerminalBindingMemberArms (37) |
| 62 | Hostile extension literals refuse at extensions arm; lone surrogate at frame | ParseTerminalBinding | TestParseTerminalBindingRawExtensions (6) |
| 63 | Identity pre-transform walk refuses hostile bytes; clean digest equals | bindingIdentity | TestBindingIdentityRefusesHostileBytes (12+clean) |
| 64 | Identity/major verdicts agree with landed rules | bindingIdentity, isProtocolMajorOne | TestBindingIdentityVerdictAgreesWithLandedAdmission (2), TestBindingProtocolMajorAgreesWithLandedTuple (14) |
| 65 | 257-char generation refuses before the status read runs | RecoverCreate | TestRecoverGenerationBoundBeforeStatusRead |
| 66 | Caller-bound checkpoint/pair boundary pinned (fabricated appends, fold newest) | EmitResumed | TestEmitResumedCheckpointBindingIsCallerBound |

## Spec-clause mapping (measured section ratios quoted verbatim)

| Section | Measured | Discharged here | Rest |
|---------|----------|-----------------|------|
| 4.1 | sliver 1/5 | 4.1#5 via RecoverCreate (rows 35–43, 59, 65) | backend semantic ops, wrapper validation/parking, durable pair bind — operation matrix / axpane wrapper / backend receipt owners |
| 4.B | sliver 1/12 | 4.B#11 via CheckInstanceIdentity (rows 1–15) + Binding 1.0.0 closed object via ParseTerminalBinding (rows 3–9, 61–64) | Manifest/Probe/evidence reconciliation, reader ID recompute, generation boundaries — internal/terminalbackend Reconcile/AdmitProbe |
| 4.D | sliver 1/3 | 4.D#2 via ResolveEvidence→Registry.AdmitProbe signature verification (rows 16–26) | 4.D#1 registry-row equality, 4.D#3 no-silent-fallback — landed Reconcile / explicit selection |
| 5.2 | sliver 3/18 | 5.2#16, #17 via ResolveEvidence (rows 16–26); #18 via EmitTerminalCreated + sessrepo/sessstate/sessquery retention (rows 27–34, 54–56; v1-half bound row 57) | envelope/table/authorship/ordering/epoch — landed validateSessionEvent + sessrepo ordering |
| 7.A | partial 1/2 | 7.A#1 via AdmitProviderDescriptor mismatch pin (row 10) | 7.A#2 LeaseToken v2 / 3.x envelope — provhost v2, dual-stack follow-up |
| 13.1 (as cited by producer) | NOT APPLICABLE to this story | The recovery rule lives at §4.1 L1395/1399 in SPEC.v0.7.0; §13.1's eight measured clauses are launch-plan/boot-direct/argv rules owned by another story | owner: that story; section:13.1 untouched |

## Stated bounds (with owners)

- Attach emits neither event by construction (Attach takes no repository) —
  proven by row 51; no text plant can add a write. Owner: termbind.
- Lease-authority narrowing belongs to adopted axpane rows N-emit-mutation /
  N-emit-losing-epoch; composition pinned by row 32. Owner: axpane.
- CLI Result 4 identity surface unimplemented — tripwire bound, row 15.
  Owner: cliresult (follow-up).
- v1-only-reader inertness undrivable — payload census bound, row 57.
  Owner: termbind (no v1-v3-only reader exists on trunk).
- EmitResumed checkpoint/pair binding caller-supplied — boundary pinned,
  row 66. Owner: termbind (wrapper decision supplies both from the fold).
- Restore/terminate-stale execution deferred to a Story follow-up task
  (no board task exists yet — no brief assigned it). Owner: follow-up.
- kkh1an `HandoffFailed` reporter note NOT APPLICABLE — this leaf wires
  no reporter. Owner: n/a.
- Operation-body pre-scans are a documented redundancy (landed parsers
  refuse the same code) — rows 11, 12. Owner: termbind.
- No source-text gate exists — token-preserving mutant N/A. Owner: termbind.
