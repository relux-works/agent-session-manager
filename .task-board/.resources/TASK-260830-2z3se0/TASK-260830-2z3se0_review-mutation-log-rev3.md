# TASK-260830-2z3se0 — reviewer mutation log, revision 3

Method: exact string or exact line replacement in production source; grep-confirmed
present; `go vet ./internal/sessadapter/` clean before the verdict; production
restored immediately after each mutant; working-tree OID re-verified per batch
(`9b1c084a1b0a970bc2eed187d5155f4191528c51` throughout).
Command unless noted: `go test ./internal/sessadapter/ -count=1`.

## G-A — B3 additive-arm probes (floor held at 323)

| ID | Mutant | Arms derived | Verdict | Killer |
|---|---|---:|---|---|
| M-A1 | 8th package-level ctor `failQuota` + additive arm calling it | 323 | KILLED | `TestRefusalConstructorsMatchProduction` (only failure) |
| M-A2 | inline `axerror.New` refusal + additive arm, no constructor | 323 | KILLED | `TestRefusalConstructorsMatchProduction` (only failure) |
| M-A3 | second aliased import `axe ".../axerror"` in protocol.go + `axe.New` additive arm | 323 | **SURVIVED** | — |
| M-A4 | `axe ".../axerror"` in manifest.go + `axe.New` additive arm in `DecodeManifest` | 323 | **SURVIVED** (`go test ./...` exit 0) | — |

M-A1 message: `axerror.New inside unregistered "failQuota" at protocol.go:160:9`.
`TestDerivedRefusalArmsAreAllWitnessed` / `TestWitnessedArmsAreAllDerived` both
logged `323 derived arms across 9 production files` and PASSED in M-A1 and M-A2.

## G-B — N4 (`CheckCallBinding` zero-fact exclusions)

| Mutant | Verdict | Killer subtest |
|---|---|---|
| `role != binding.Role && binding.Role != ""` | KILLED | `TestCheckCallBindingZeroFactsRefuse/zero_binding_role` |
| `… && binding.ProviderID != ""` | KILLED | `/zero_binding_provider` |
| `… && binding.AdapterManifestDigest != ""` | KILLED | `/zero_binding_manifest_digest` |
| `… && binding.ExecutableSHA256 != ""` | KILLED | `/zero_binding_executable_digest` |
| `… && admitted != (Tuple{})` | KILLED | `/zero_admitted_tuple` |

## G-B — N5 and the 11-shape census denominator

| Shape planted in `validCapabilityStatus` | Verdict | Killer |
|---|---|---|
| package-level slice composite (S1) | KILLED | `TestClosedVocabularyTablesAreRegistered` |
| named slice type composite (S2) | KILLED | same |
| inline `\|\|` chain (S3) | KILLED | `TestNoUnregisteredInlineVocabularies` |
| package-level `map[string]struct{}` (S4) | KILLED | `TestClosedVocabularyTablesAreRegistered` |
| tagged switch on a non-`operation` ident (S5) | KILLED | `TestAllProductionSwitchesAreClassified` |
| paired `var a, b = []string{…}, []string{…}` (N5a) | KILLED | `TestClosedVocabularyTablesAreRegistered` |
| tagless switch outside `readUTF16Escape` (N5b) | KILLED | `TestAllProductionSwitchesAreClassified` |
| function-local slice literal (N5c) | SURVIVED — declared bound | — |
| `const` string + `strings.Contains` (N5d) | SURVIVED — declared bound | — |
| `make()` + `init()` population (S8) | SURVIVED — declared bound | — |
| `regexp.MustCompile` alternation (S10) | SURVIVED — declared bound | — |

7 of 11 shapes seen; 4 blind, all 4 named in the census stated bound.

## G-C.1 — every registered vocabulary table widened by one member: 63 of 63 KILLED

Denominator derived from `vocabularyRegistrations` (63 rows), each table located in
production by name and widened with one planted member. Rev2's four survivors are
the first four rows.

| Table | Killer |
|---|---|
| successMembers | TestClosedMemberSetsAreDerivedFromSpec |
| failureMembers | TestClosedMemberSetsAreDerivedFromSpec |
| manifestMembers | TestClosedMemberSetsAreDerivedFromSpec |
| doctorResultMembers | TestClosedMemberSetsAreDerivedFromSpec |
| candidateKindNames, capabilityEvidences, capabilityStatuses, captureClasses, directionNames, evidenceResults, fidelityDispositions, fidelityProfiles, findingSeverities, objectModes, objectPurposes, projectionStrategies, readPurposes, registryEntryStatuses, roleNames, tupleArchitectures, tupleEntryStatuses, validateModes | TestValueVocabulariesMatchSpec |
| capabilityOrder | TestValueVocabulariesMatchSpec/capabilities_ordered |
| capabilityValueMembers, capturePlanItemMembers, contextMembers, doctorResultRequired, fidelityLimitMembers, fidelityLimitRequired, findingMembers, findingRequired, fixtureEvidenceMembers, objectAuthorityMembers, probeMembers, readAuthorityMembers, requestMembers, requestRequired, resourceLimitsMembers, resourceLimitsRequired, smokeEvidenceMembers, sourceSelectorMembers, tupleEntryMembers, tupleKeyMembers, tupleMembers, capabilityValueRequired, capturePlanItemRequired | TestClosedMemberSetsAreDerivedFromSpec (+ entry-point tests) |
| contextRequired, failureRequired, successRequired, manifestRequired, probeRequired, tupleRequired, objectAuthorityRequired, readAuthorityRequired, sourceSelectorRequired, fixtureEvidenceRequired, smokeEvidenceRequired, tupleEntryRequired, tupleKeyRequired | entry-point tests (CheckSuccessEnvelope, CheckFailureEnvelope, CheckTupleAdmission, CapturePlanDigest, …) |
| operationOrder | TestValueVocabulariesMatchSpec/operations_ordered |
| contextFreeOperations | TestValueVocabulariesMatchSpec/context-free_operations |
| requestBodyMembers | TestOperationBodiesAreDerivedFromSpec |
| successBodyMembers | TestDiscoverPartialCursorComplement + derived-body tests |

## G-C.2 — obligation loops and capability gates: 10 KILLED / 3 SURVIVED

| ID | Mutant | Verdict | Killer |
|---|---|---|---|
| A2 | `DoctorRequiredCapabilities` drops `workspace_binding` | KILLED | TestDoctorRequiredCapabilities |
| A3 | `CheckDoctorHealthy` usable loop exempts `workspace_binding` | KILLED | …DrivesEveryRequiredCapability/workspace_binding |
| A5 | `checkValidateNullability` targets drops one | KILLED | TestCheckValidateNullabilityDrivesEveryTarget |
| A6 | archive arm exempts `expected_target_native_session_id` | KILLED | …/archive_carries_each_target |
| A7 | `checkValidateResult` applicable triple → pair | KILLED | …DrivesEveryApplicableCheck/resume_surface_valid |
| A8 | nullable-bool triple → pair | KILLED | …/non_boolean_check_member |
| A9 | target-write adapter list drops `workspace_binding` | KILLED | TestCheckTargetWriteGates |
| A10 | writer pair narrowed by `&& len(adapter) != 15` | KILLED | TestCheckTargetWriteGates |
| A11 | target-write provider list drops `native_resume` | KILLED | TestCheckTargetWriteGates |
| A17 | doctor entry-status gate admits `revoked` | KILLED | TestDoctorResultRules |
| A1 | `CheckDoctorHealthy` name check exempts `""` | **SURVIVED** | — |
| A18 | `capabilityMapUsable` absent ⇒ usable | **SURVIVED** | — |
| A19 | `CapabilityUsable` absent ⇒ usable | **SURVIVED** | — |

A18 behavioural proof at the production entry (scratch test, deleted after the run):

```
baseline  CheckTargetWriteGates(minus native_read_back) -> capability_unavailable:
              target write misses a usable adapter capability
baseline  CheckTargetWriteGates(minus canonical_write and official_import) ->
              capability_unavailable: neither usable canonical-write nor official-import
mutant    both -> err=<nil>
```

## G-C.3 — bound guards, both edges: 12 KILLED / 20 SURVIVED

Upper edges moved +1; lower `== 0` edges moved to `== -1`.

| Site | Bound | Verdict | Killer |
|---|---|---|---|
| protocol.go:249 / :355 / :445 | `> MaxFrameBytes` | KILLED ×3 | TestFrameBoundEdges |
| manifest.go:163 | display_name 1..128 | KILLED | TestDecodeManifestValueRules |
| probe.go:144 | detail 0..2048 | KILLED | TestDecodeProbeClosedRules |
| operations.go:593 | `> 7` | KILLED | TestEveryArmWitnessRefusesAtTheProductionEntry |
| operations.go:1016 | `> 65536` | KILLED | same |
| operations.go:1046 | `> 9` | KILLED | same |
| tuple.go:790 | `> 1024` | KILLED | same |
| operations.go:933 | `len(argv) == 0` | KILLED | same |
| tuple.go:948 | `len(elements) == 0` | KILLED | same |
| tuple.go:998 | `len(versions) == 0` | KILLED | same |
| manifest.go:178 | environment_version_range 1..256 | **SURVIVED** | — |
| context.go:613 | native_session_id 1..512 | **SURVIVED** | — |
| context.go:636 | opaque_source_ref 1..512 | **SURVIVED** | — |
| context.go:847 | code 1..128 | **SURVIVED** | — |
| context.go:855 | message 1..4096 | **SURVIVED** | — |
| context.go:865 | remediation 1..4096 | **SURVIVED** | — |
| context.go:971 | native_item_key 1..512 | **SURVIVED** | — |
| operations.go:933 | `len(argv) > 128` | **SURVIVED** | — |
| operations.go:942 | argv word 1..4096 | **SURVIVED** | — |
| operations.go:593 | `len(elements) == 0` (lower) | **SURVIVED** | — |
| tuple.go:96 | environment_version 1..128 | **SURVIVED** | — |
| tuple.go:594 | suite_revision 1..128 | **SURVIVED** | — |
| tuple.go:737 | native_cli_family 1..128 | **SURVIVED** | — |
| tuple.go:822 | code 1..128 | **SURVIVED** | — |
| tuple.go:830 | affected_class 1..128 | **SURVIVED** | — |
| tuple.go:846 | detail 1..4096 | **SURVIVED** | — |
| tuple.go:948 | `len(elements) > 64` | **SURVIVED** | — |
| tuple.go:981 | contract identifier 1..256 | **SURVIVED** | — |
| tuple.go:998 | `len(versions) > 32` | **SURVIVED** | — |
| tuple.go:1155 | revocation_reason 1..4096 | **SURVIVED** | — |

Behavioural proof (scratch test, deleted after the run):

```
baseline  DecodeManifest, environment_version_range = 257 chars ->
              session_adapter_protocol_error: … not a non-empty string[1..256]
mutant    checkStringBounds(…, 1, 257) -> err=<nil>
```

## G-C.4 — envelope echo gates, zero-value exemption: 0 KILLED / 10 SURVIVED

Every gate narrowed to `(x != want && x != "")`.

| Site | Member | Verdict |
|---|---|---|
| protocol.go:278 | request `protocol` | **SURVIVED** |
| protocol.go:285 | request `protocol_version` | **SURVIVED** |
| protocol.go:384 | success `protocol` | **SURVIVED** |
| protocol.go:391 | success `protocol_version` | **SURVIVED** |
| protocol.go:398 | success `request_id` | **SURVIVED** |
| protocol.go:405 | success `operation` | **SURVIVED** |
| protocol.go:474 | failure `protocol` | **SURVIVED** |
| protocol.go:481 | failure `protocol_version` | **SURVIVED** |
| protocol.go:488 | failure `request_id` | **SURVIVED** |
| protocol.go:495 | failure `operation` | **SURVIVED** |

Behavioural proof at protocol.go:278 (scratch test, deleted after the run) —
the package's own valid doctor frame with `"protocol"` set to `""`:

```
baseline  DecodeRequestFrame -> session_adapter_protocol_error:
              request protocol is not the session adapter
mutant    DecodeRequestFrame -> err=<nil>,
              Request{Operation:doctor, RequestID:0198f4c8-…-1234567890ab, Body:{…}}
```

## G-C.5 — verification of the accepted rev2 findings

| Mutant | Verdict | Killer |
|---|---|---|
| B1 defect restored (`checkValidateNullability` called on the success body) | KILLED | `TestValidateSuccessAdmitsEveryMode/archive` |
| B2 `successMembers` widened by one name | KILLED | `TestClosedMemberSetsAreDerivedFromSpec` |

## Totals

- Gate-narrowing mutants applied: **131**. Killed **96**, survived **35** (73%).
- Census shape probes (behaviour-neutral, reported outside the ratio): 11 —
  7 seen, 4 blind by declared bound.
- Round-2 survivors closed: **11 of 11**. Resurrections among re-measured
  round-2 kills: **0**.
- Tree OID after the final batch: `9b1c084a1b0a970bc2eed187d5155f4191528c51`
  (equal to the candidate). `go test ./... -count=1` on the restored tree: 17/17 ok.
