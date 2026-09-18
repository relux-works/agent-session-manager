# TRACEABILITY — internal/termbind (AX v0.7.0)

Normative scope: AX v0.7.0 Section 4.B (Terminal Instance Binding 1.0.0
closed object and the identity rule), Section 5.2 Session Event 4.0.0
Terminal Instance events (terminal.created and session.resumed v4
payloads), Section 4.D (evidence binding), Section 7.A (Provider
Protocol 3 descriptor identity, cited landed), and Section 13.1
(bootstrap idempotency and lost-result recovery). Story:
ax-pane-and-terminal-instance-binding (final leaf).

Rule: every row names its production entry point. A row is driven only
when the named committed test executes that entry. Prose in place of
the ratio is not evidence. 65 of 66 AC rows driven; row 57 is a stated
bound (payload census — no v1-v3-only reader exists on trunk to drive).

## Identity and the Binding object (Section 4.B, Section 7.A)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 1 | UUIDv7 admitted as terminal instance identity | CheckInstanceIdentity (identity.go) | TestCheckInstanceIdentityAdmitsUUIDv7 |
| 2 | Eight forbidden forms (PID, handle, socket, path, pipe, URL, token, endpoint) refuse terminal_backend_protocol_error | CheckInstanceIdentity (identity.go) | TestCheckInstanceIdentityRefusesForbiddenForms (8 subtests) |
| 3 | Closed Binding fixture admitted with literal members | ParseTerminalBinding (identity.go) | TestParseTerminalBindingAdmitsClosedFixture |
| 4 | Eight forbidden forms in terminal_instance_id refuse the pinned code | ParseTerminalBinding (identity.go) | TestParseTerminalBindingRefusesForbiddenIdentity (8 subtests) |
| 5 | Binding closed shape (missing/extra/duplicate member, wrong schema) refuses | ParseTerminalBinding (identity.go) | TestParseTerminalBindingClosedShape (4 subtests) |
| 61 | Every Binding member arm refuses with its code and detail (schema/schema_version/binding_id/session/host/incarnation/instance/backend-ID/impl/protocol/generation/native/created/supersedes/extensions value arms, non-object frame, trailing data) | ParseTerminalBinding (identity.go) | TestParseTerminalBindingMemberArms (37 subtests) |
| 62 | Hostile extension literals (JSON numbers, nested duplicate) refuse at the extensions arm; lone surrogate refuses at the frame | ParseTerminalBinding (identity.go) | TestParseTerminalBindingRawExtensions (6 subtests) |
| 63 | Identity pre-transform walk refuses numbers, nested duplicates, trailing data, non-object, depth and size caps; clean digest equals | bindingIdentity (identity.go) | TestBindingIdentityRefusesHostileBytes (12 subtests + clean) |
| 64 | Identity verdicts and the protocol-major verdict agree with the landed rules | bindingIdentity (identity.go), isProtocolMajorOne (identity.go) | TestBindingIdentityVerdictAgreesWithLandedAdmission (2 subtests), TestBindingProtocolMajorAgreesWithLandedTuple (14 subtests) |
| 6 | Binding backend_generation 256 admits, 0 and 257 refuse | ParseTerminalBinding (identity.go) | TestParseTerminalBindingClosedShape/generation_bounds |
| 7 | Binding native_reference 512 admits, 0 and 513 refuse | ParseTerminalBinding (identity.go) | TestParseTerminalBindingClosedShape/native_reference_bounds |
| 8 | Tampered member and wrong self ID refuse terminal_backend_manifest_probe_mismatch | ParseTerminalBinding (identity.go) | TestParseTerminalBindingClosedShape (tampered, wrong_self_id) |
| 9 | Binding identity construction agrees with the landed manifest admission | bindingIdentity via ParseManifest (identity.go) | TestBindingIdentityAgreesWithLandedAdmission |
| 10 | Eight forbidden forms in the Provider Protocol 3 descriptor refuse the pinned code | AdmitProviderDescriptor, adopted landed (terminalbackend/descriptor.go) | TestAdmitDescriptorRefusesForbiddenIdentity (8 subtests) |
| 11 | Eight forbidden forms in the mutation context refuse the pinned code (documented redundancy: the landed parser refuses the same code; the wrapper adds the Go error type) | AdmitMutationContext (identity.go) | TestAdmitMutationContextRefusesForbiddenIdentity (8 subtests) |
| 12 | Eight forbidden forms in the status body refuse the pinned code (same redundancy); null instance admitted | AdmitStatusBody (identity.go) | TestAdmitStatusBodyRefusesForbiddenIdentity (8 subtests + null) |
| 13 | Non-string identity members delegate to the landed shape authority | AdmitMutationContext, AdmitStatusBody (identity.go) | TestAdmitSurfacesDelegateShapeFaults (2 subtests) |
| 14 | Eight forbidden forms in the bootstrap binding refuse and persist nothing | Store.Bind, adopted (axpane/binding.go) | TestBootstrapBindingRefusesForbiddenIdentity (8 subtests) |
| 15 | CLI Result 4.0.0 identity surface is a stated bound pinned by a tripwire | VersionForCommand, cited (cliresult) | TestCLIResult4IdentityIsUnimplementedBound |

## Evidence resolution (Section 4.D, Section 5.2)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 16 | Coherent universe resolves to manifest, probe, evidence, and admission | ResolveEvidence (resolve.go) | TestResolveEvidenceAdmitsUniverse |
| 17 | Empty and 257-element evidence sets refuse the 1..256 bound | ResolveEvidence (resolve.go) | TestResolveEvidenceShapeBounds (empty, 257) |
| 18 | Unsorted, duplicate, and non-digest evidence sets refuse | ResolveEvidence (resolve.go) | TestResolveEvidenceShapeBounds (unsorted, duplicate, non-digest) |
| 19 | One digest passes the shape and refuses at the manifest/probe partition | ResolveEvidence (resolve.go) | TestResolveEvidenceShapeBounds/one_passes_shape_to_partition |
| 20 | 256 digests pass the shape and refuse at lookup | ResolveEvidence (resolve.go) | TestResolveEvidenceShapeBounds/256_passes_shape_to_lookup |
| 21 | Eleven foreign resolution targets refuse the mismatch class | ResolveEvidence (resolve.go) | TestResolveEvidenceRefusesForeignKinds (11 subtests) |
| 22 | Missing or doubled manifest/probe refuses the stated cardinality | ResolveEvidence (resolve.go) | TestResolveEvidenceRequiresManifestAndProbe (4 subtests) |
| 23 | Foreign-backend and protocol-drift evidence refuse at this gate | ResolveEvidence (resolve.go) | TestResolveEvidenceBindsEventTuple (2 subtests) |
| 24 | Document substitution under another ID refuses | ResolveEvidence (resolve.go) | TestResolveEvidenceRejectsSubstitution (2 subtests) |
| 25 | Admission faults (unevidenced claim, forged signature, expiry, registry drift) surface landed | ResolveEvidence via Registry.AdmitProbe (resolve.go) | TestResolveEvidenceDrivesLandedAdmission (4 subtests) |
| 26 | Resolved objects carry no live-process fact | ResolveEvidence (resolve.go) | TestResolvedObjectsCarryNoLiveFacts |

## Emission (Section 5.2)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 27 | terminal.created v4 round-trips through the append path | EmitTerminalCreated (emit.go) | TestEmitTerminalCreatedRoundTrip |
| 28 | session.resumed v4 round-trips (null and carried source); bad profile refuses | EmitResumed (emit.go) | TestEmitResumedRoundTrip (3 subtests) |
| 66 | Caller-bound checkpoint/pair boundary pinned: a fabricated checkpoint appends and the fold reports it newest (the wrapper supplies both from the fold; CheckResumedPair uncomposed — stated bound) | EmitResumed (emit.go) | TestEmitResumedCheckpointBindingIsCallerBound |
| 29 | Unsorted and empty evidence refuse at the emission entries | EmitTerminalCreated (emit.go) | TestEmitClosedShapeRefusals (unsorted, empty) |
| 30 | Missing/extra payload members refuse at the canonical boundary | Emit, adopted (axpane/events.go) | TestEmitClosedShapeRefusals (3 append subtests) |
| 31 | Unresolvable and foreign evidence refuse inside both emission entries | EmitTerminalCreated, EmitResumed (emit.go) | TestEmitResolvesEvidence (3 subtests) |
| 32 | Losing lease refuses and appends nothing | EmitTerminalCreated via Emit (emit.go) | TestEmitUnderLosingLeaseRefuses |
| 33 | Byte-identical retry replays the one event | EmitTerminalCreated (emit.go) | TestEmitAppendsIdempotently |
| 34 | Emitted bytes carry the opaque digest and no binding secret | EmitTerminalCreated (emit.go) | TestEmittedEventCarriesNoBindingObject |

## Recovery (Section 13.1)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 35 | No binding plus proven absence yields absence with replay_same | RecoverCreate (recover.go) | TestRecoverCreateProvesAbsence |
| 36 | Binding plus matching live status yields the ONE child | RecoverCreate (recover.go) | TestRecoverCreateIdentifiesOneChild (2 subtests) |
| 37 | Duplicate retry yields the identical verdict | RecoverCreate (recover.go) | TestRecoverCreateDuplicateRetrySameVerdict |
| 38 | Changed operation in the fold-derived open window refuses idempotency_mismatch | RecoverCreate (recover.go) | TestRecoverCreateChangedOperationInWindowRefuses |
| 39 | Changed operation in the fold-derived closed window proceeds | RecoverCreate (recover.go) | TestRecoverCreateChangedOperationPostWindowProceeds |
| 40 | Lost create result recovers the ONE recorded child | RecoverCreate (recover.go) | TestRecoverCreateLostResult |
| 41 | Five unprovable shapes yield unavailable with status_first, never absence | RecoverCreate (recover.go) | TestRecoverCreateNeverClaimsAbsent (5 subtests) |
| 42 | Recorded child with a backend-absent observation replays the same pair | RecoverCreate (recover.go) | TestRecoverCreateRecordedChildBackendAbsentReplays |
| 43 | Status read failure is an error, never absence | RecoverCreate (recover.go) | TestRecoverCreateUnknownIsNotAbsent |
| 65 | 257-character generation refuses the landed bound before the status read runs | RecoverCreate (recover.go) | TestRecoverGenerationBoundBeforeStatusRead |

## Attach receipts (Section 4.C attach, Section 4.E)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 44 | Attach records and replays one receipt per client pair | AttachStore.Attach/Lookup (attach.go) | TestAttachRoundTrip |
| 45 | Identical retry replays the recorded receipt | AttachStore.Attach (attach.go) | TestAttachIdenticalRetryReplays |
| 46 | Second client records alongside, never over | AttachStore.Attach (attach.go) | TestAttachSecondClientRecordsAlongside |
| 47 | Transport/input conflict refuses idempotency_mismatch | AttachStore.Attach (attach.go) | TestAttachConflictRefusesMismatch |
| 48 | Sixteen forbidden identity values refuse the pinned code | AttachStore.Attach (attach.go) | TestAttachRefusesForbiddenIdentity (16 subtests) |
| 49 | Unknown transport refuses; relay refuses by landed policy | AttachStore.Attach (attach.go) | TestAttachTransportVocabulary (2 subtests) |
| 50 | Expired, mismatched, and malformed policy refuses on every call | AttachStore.Attach (attach.go) | TestAttachRequiresLiveAuth (4 subtests) |
| 51 | Attach emits neither event nor lease/fencing change | AttachStore.Attach (attach.go) | TestAttachEmitsNeitherEventNorStateChange |
| 52 | Torn receipt is an error, never absence | AttachStore.Attach/Lookup (attach.go) | TestAttachUnknownIsNotAbsent |
| 53 | Foreign-session lookup refuses | AttachStore.Lookup (attach.go) | TestAttachSessionMismatchRefuses |

## Version skew and authority (Section 5.2)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 54 | Takeover selects an admitted backend with a new digest and event, append-only, no fork | EmitTerminalCreated, EmitResumed (emit.go) | TestTakeoverSelectsNewBackendNewEvent |
| 55 | v1 and v4 terminal events derive exactly their own surface | Reduce, adopted (sessstate) | TestVersionSelectionBothDirections (3 subtests) |
| 56 | v4 retained verbatim, no lower-version replacement, query reads derive nothing new | GetEvent/ListEvents, Reader.List, adopted (sessrepo, sessquery) | TestV4RetainedAsImmutableHistory |
| 57 | STATED BOUND: v4 carries no v1 derivation member (payload census); a v1-only reader staying inert cannot be driven — no v1-v3-only reader exists on trunk (the retained-verbatim half is row 56) | DecodeEvent, adopted (sessstate) | TestV1ReaderSeesV4AsInert (census, not a reader drive) |

## Crash and idempotency seams

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 58 | SIGKILL after the receipt commit leaves the complete receipt; retry replays | AttachStore.Attach (attach.go) | TestAttachCrashChildSelfTerminates (unix) |
| 59 | SIGKILL between effect and result recovers the ONE child | RecoverCreate (recover.go) | TestRecoverCreateAfterKillRecoversChild (unix) |
| 60 | Before-write abort leaves no event; after-commit keeps and replays | EmitTerminalCreated (emit.go) | TestEmitAppendHookSeams (2 subtests) |

## Mutant table (36 rows: 34 narrowing + 1 supplementary arm-delete + 1 control)

Harness: `PYTHONDONTWRITEBYTECODE=1 python3
internal/termbind/mutant_harness.py`. Verdict rule (M/R/V): R fails
with a test failure line means KILLED; R passes means SURVIVED; an
unapplied patch, build break, or timeout is ERROR. All rows pass 1,
mutants/pass2.

| ID | Kind | Plant | Killer |
|----|------|-------|--------|
| N-identity-pid | narrowing | CheckInstanceIdentity admits "12345" | TestCheckInstanceIdentityRefusesForbiddenForms, TestParseTerminalBindingRefusesForbiddenIdentity, TestAdmitMutationContextRefusesForbiddenIdentity, TestAdmitStatusBodyRefusesForbiddenIdentity, TestAttachRefusesForbiddenIdentity |
| N-identity-endpoint | narrowing | CheckInstanceIdentity admits "10.0.0.9:2222" | same five surfaces |
| N-binding-members | narrowing | ParseTerminalBinding admits one extra member | TestParseTerminalBindingClosedShape/extra_member |
| N-binding-identity | narrowing | ParseTerminalBinding admits the wrong self-ID fixture | TestParseTerminalBindingClosedShape/wrong_self_id |
| N-binding-generation | narrowing | ParseTerminalBinding admits the 257-character generation | TestParseTerminalBindingClosedShape/generation_bounds |
| N-binding-protocol-major | narrowing | ParseTerminalBinding admits protocol 2.0.0 past the major-1 arm | TestParseTerminalBindingMemberArms/protocol_version_major_2 |
| N-binding-extensions | narrowing | ParseTerminalBinding admits the one-key note extensions object | TestParseTerminalBindingMemberArms/extensions_non-empty |
| N-binding-identity-number | narrowing | Identity walk admits the 1.0 number | TestBindingIdentityRefusesHostileBytes/float |
| N-binding-nested-dup | narrowing | Identity decode admits nested duplicates | TestBindingIdentityRefusesHostileBytes/nested_duplicate |
| N-binding-supersedes | narrowing | ParseTerminalBinding admits the not-a-digest supersedes value | TestParseTerminalBindingMemberArms/supersedes_non-digest |
| N-prescan-context | narrowing | AdmitMutationContext skips the gate for "12345" | TestAdmitMutationContextRefusesForbiddenIdentity/pid |
| N-status-prescan | narrowing | AdmitStatusBody skips the gate for "12345" | TestAdmitStatusBodyRefusesForbiddenIdentity/pid |
| N-evidence-257 | narrowing | Shape admits 257 IDs | TestResolveEvidenceShapeBounds/257_refuses |
| N-evidence-0 | narrowing | Shape admits the empty set | TestResolveEvidenceShapeBounds/empty_refuses |
| N-evidence-dup | narrowing | Order admits duplicates | TestResolveEvidenceShapeBounds/duplicate_refuses |
| N-evidence-unsorted | narrowing | Order skips the first-pair comparison | TestResolveEvidenceShapeBounds/unsorted_refuses |
| N-resolve-kind | narrowing | Resolution admits the endpoint document | TestResolveEvidenceRefusesForeignKinds/endpoint, TestEmitResolvesEvidence/foreign_kind |
| N-resolve-manifest-count | narrowing | Partition admits the four-ID double manifest | TestResolveEvidenceRequiresManifestAndProbe/second_manifest |
| N-resolve-binding | narrowing | Binding drops the evidence protocol arm | TestResolveEvidenceBindsEventTuple/protocol_drift_evidence |
| N-resolve-identity | narrowing | Substitution check drops the probe half | TestResolveEvidenceRejectsSubstitution/probe_alias |
| N-resolve-admission | narrowing | Resolution swallows the landed drift refusal | TestResolveEvidenceDrivesLandedAdmission/registry_drift |
| N-emit-created-v3 | narrowing | Created event emitted at v3 | TestEmitTerminalCreatedRoundTrip |
| N-emit-resumed-v3 | narrowing | Resumed event emitted at v3 | TestEmitResumedRoundTrip |
| D-emit-noresolve | arm-delete, supplementary | Resolution dropped at both emission entries | TestEmitResolvesEvidence |
| N-recover-mismatch | narrowing | Recovery admits the fixture changed operation | TestRecoverCreateChangedOperationInWindowRefuses |
| N-recover-second-child | narrowing | Unbound live instance adopted as a child | TestRecoverCreateNeverClaimsAbsent/unbound_live_instance |
| N-recover-contradiction-absent | narrowing | Bound contradiction claimed as absence | TestRecoverCreateNeverClaimsAbsent/bound_contradiction |
| N-recover-creating-child | narrowing | Creating interim admitted as a child | TestRecoverCreateNeverClaimsAbsent/creating_interim |
| N-recover-status-error | narrowing | Status read failure laundered as absence | TestRecoverCreateUnknownIsNotAbsent/status_read_failure |
| N-recover-window | narrowing | Window bit inverted | TestRecoverCreateChangedOperationInWindowRefuses |
| N-recover-generation | narrowing | Generation bound skipped for the 257-character generation before the status read | TestRecoverGenerationBoundBeforeStatusRead |
| N-attach-conflict | narrowing | Conflict drops the input half at both comparisons | TestAttachConflictRefusesMismatch |
| N-attach-transport | narrowing | Enum admits smoke_signal | TestAttachTransportVocabulary/unknown_transport |
| N-attach-auth-replay | narrowing | Replay skips authorization | TestAttachRequiresLiveAuth/replay_with_expired_auth |
| N-attach-key | narrowing | Idempotency key drops the client axis | TestAttachSecondClientRecordsAlongside |
| C-control | control | Comment-only doc edit | TestCheckInstanceIdentityAdmitsUUIDv7 (SURVIVED) |

Not rows, by construction: attach cannot emit an event because Attach
takes no repository (proven by row 51 — no text plant can add a write);
the lease-authority narrowing belongs to the adopted axpane rows
N-emit-mutation and N-emit-losing-epoch (composition pinned by row 32);
the descriptor identity gate is landed and narrowed by the
terminalbackend owner (this leaf drives its eight forms through the
landed entry in row 10).

## Stated contracts and bounds

- Resolution cardinality: exactly one Manifest and exactly one Probe per
  event (the specification pins the resolution requirement but not the
  cardinality; ambiguity fails closed). Rows 19, 22.
- The bootstrap window bit is caller-derived from the sessstate fold's
  newest checkpoint; recovery tests close the loop by folding a real
  chain (rows 38, 39) instead of forking the loader.
- CLI Result 4.0.0 is registered but unimplemented in internal/cliresult
  (owner: cliresult); the Result 4 identity surface is NOT APPLICABLE
  with a tripwire that fails if Result 4 ever lands (row 15).
- Restore/terminate-stale engine execution remains refused by the
  terminstance scope gates; recovery in this leaf covers the Section 4.1
  bootstrap (session_id, bootstrap_operation_id) create row. Executing
  restore/terminate-stale is deferred to a Story follow-up task (no board
  task exists yet — no brief in this Story assigned it).
- The kkh1an `HandoffFailed` reporter note is NOT APPLICABLE: this leaf
  wires no `HandoffFailed` reporter (only tests set the member).
- EmitResumed binds neither checkpoint_id to the chain nor the
  (profile, source) pair to the referenced checkpoint's closure via
  sessprofile.CheckResumedPair; the wrapper decision supplies both from
  the fold, and the direct-caller boundary is pinned (row 66).
- The v1-only-reader half of 5.2#18 is a stated bound: no v1-v3-only
  reader exists on trunk, so row 57 is a payload census and the
  retained-verbatim half (row 56) carries the drive.
- The operation-body pre-scans (rows 11, 12) are a documented
  redundancy: the landed parsers refuse the same pinned code, and the
  wrappers add only the Go error type their narrowing rows kill on.
- No `ax` CLI surface, no tmux/ConPTY process control, no provider
  process, and no mesh transport live in this package.
