| mutant | file | narrowing | driver | verdict |
| --- | --- | --- | --- | --- |
| Bstr-display-max | manifest.go | `checkStringBounds(members["display_name"], 1, 129)` | TestDecodeManifestValueRules | killed |
| Bstr-envrange-min | manifest.go | `checkStringBounds(members["environment_version_range"], 0, 256)` | TestStringBoundEdges | killed |
| Bstr-findcode-max | context.go | `checkStringBounds(members["code"], 1, 129)` | TestStringBoundEdges | killed |
| Bstr-tuplever-min | tuple.go | `checkStringBounds(members["environment_version"], 0, 128)` | TestStringBoundEdges | killed |
| Bstr-contract-max | tuple.go | `checkStringBounds(identifier, 1, 257)` | TestStringBoundEdges | killed |
| Bstr-probedetail-max | probe.go | `checkStringBounds(members["detail"], 0, 2049)` | TestStringBoundEdges | killed |
| Breq-cursor-max | operations.go | `requireStringBounds(members, "cursor", 1, 1025)` | TestRequestScalarBoundEdges/discover/cursor | killed |
| Breq-cursor-min | operations.go | `requireStringBounds(members, "cursor", 0, 1024)` | TestRequestScalarBoundEdges/discover/cursor | killed |
| Breq-gen-max | operations.go | `requireStringBounds(members, "expected_source_store_generation", 1, 513)` | TestRequestScalarBoundEdges/snapshot-proof/expected_source_store_generation | killed |
| Breq-target0-max | operations.go | `requireStringBounds(members, "expected_target_native_session_id", 1, 513)` | TestRequestScalarBoundEdges/projection-plan/expected_target_native_session_id | killed |
| Breq-target1-max | operations.go | `requireStringBounds(members, "expected_target_native_session_id", 1, 513)` | TestRequestScalarBoundEdges/read-back/expected_target_native_session_id | killed |
| Breq-target2-max | operations.go | `requireStringBounds(members, "expected_target_native_session_id", 1, 513)` | TestRequestScalarBoundEdges/validate/expected_target_native_session_id | killed |
| Breq-target3-max | operations.go | `requireStringBounds(members, "expected_target_native_session_id", 1, 513)` | TestRequestScalarBoundEdges/resume-plan/expected_target_native_session_id | killed |
| Breq-target2-min | operations.go | `requireStringBounds(members, "expected_target_native_session_id", 0, 512)` | TestRequestScalarBoundEdges/validate/expected_target_native_session_id | killed |
| Breq-limit-max | operations.go | `requireUint53Bounds(members, "limit", 1, 65537)` | TestRequestScalarBoundEdges/discover/limit | killed |
| Breq-maxitems-min | operations.go | `requireUint53Bounds(members, "max_items", 0, 65536)` | TestRequestScalarBoundEdges/capture-plan/max_items | killed |
| Breq-maxbytes-min | operations.go | `requireUint53Bounds(members, "max_total_bytes", 0, maxUint53)` | TestRequestScalarBoundEdges/capture-plan/max_total_bytes | killed |
| Bsucc-nextcursor-max | operations.go | `requireStringBounds(members, "next_cursor", 1, 1025)` | TestSuccessScalarBoundEdges/discover/next_cursor | killed |
| Bsucc-nextcursor-min | operations.go | `requireStringBounds(members, "next_cursor", 0, 1024)` | TestSuccessScalarBoundEdges/discover/next_cursor | killed |
| Bsucc-gen0-max | operations.go | `requireStringBounds(members, "source_store_generation", 1, 513)` | TestSuccessScalarBoundEdges/inspect/source_store_generation | killed |
| Bsucc-gen1-max | operations.go | `requireStringBounds(members, "source_store_generation", 1, 513)` | TestSuccessScalarBoundEdges/capture-plan/source_store_generation | killed |
| Bsucc-gen2-max | operations.go | `requireStringBounds(members, "source_store_generation", 1, 513)` | TestSuccessScalarBoundEdges/capture/source_store_generation | killed |
| Bsucc-observed-max | operations.go | `requireStringBounds(members, "observed_target_native_session_id", 1, 513)` | TestSuccessScalarBoundEdges/read-back/observed_target_native_session_id | killed |
| Bsucc-resumetarget-min | operations.go | `requireStringBounds(members, "target_native_session_id", 0, 512)` | TestSuccessScalarBoundEdges/resume-plan/target_native_session_id | killed |
| Bsucc-cwd-max | operations.go | `requireStringBounds(members, "cwd_relative", 1, 4097)` | TestSuccessScalarBoundEdges/resume-plan/cwd_relative | killed |
| Bsucc-cwd-min | operations.go | `requireStringBounds(members, "cwd_relative", 0, 4096)` | TestSuccessScalarBoundEdges/resume-plan/cwd_relative | killed |
| Bsucc-candcount-max | operations.go | `requireUint53Bounds(members, "candidate_count", 0, 65537)` | TestSuccessScalarBoundEdges/capture-plan/candidate_count | killed |
| Bsucc-itemcount-min | operations.go | `requireUint53Bounds(members, "item_count", 1, 65536)` | TestSuccessScalarBoundEdges/capture/item_count | killed |
| Bsucc-parsedevent-min | operations.go | `requireUint53Bounds(members, "parsed_event_count", 1, maxUint53)` | TestSuccessScalarBoundEdges/read-back/parsed_event_count | killed |
| Bsus-forbid-element | operations.go | `checkSortedUniqueStrings(members["forbid_reasons"], 1, 129, 0, 128)` | TestSortedUniqueBoundEdges/forbid_reasons | killed |
| Bsus-forbid-count | operations.go | `checkSortedUniqueStrings(members["forbid_reasons"], 1, 128, 0, 129)` | TestSortedUniqueBoundEdges/forbid_reasons | killed |
| Bsus-keys-element | operations.go | `checkSortedUniqueStrings(members["created_resource_keys"], 1, 513, 0, 65536)` | TestSortedUniqueBoundEdges/created_resource_keys | killed |
| Bsus-heads-element | operations.go | `checkSortedUniqueStrings(members["parsed_head_ids"], 1, 513, 0, 1024)` | TestSortedUniqueBoundEdges/parsed_head_ids | killed |
| Bsus-heads-count | operations.go | `checkSortedUniqueStrings(members["parsed_head_ids"], 1, 512, 0, 1025)` | TestSortedUniqueBoundEdges/parsed_head_ids | killed |
| Bsus-envnames-element | operations.go | `checkSortedUniqueStrings(members["environment_names"], 1, 257, 0, 128)` | TestSortedUniqueBoundEdges/environment_names | killed |
| Bsus-envnames-count | operations.go | `checkSortedUniqueStrings(members["environment_names"], 1, 256, 0, 129)` | TestSortedUniqueBoundEdges/environment_names | killed |
| Bsus-warnings-count | probe.go | `checkSortedUniqueStrings(members["warnings"], 0, 2048, 0, 1025)` | TestSortedUniqueBoundEdges/probe_warnings_count | killed |
| Bsus-roothandles-element | context.go | `checkSortedUniqueStrings(members["root_handle_names"], 1, 129, 1, 128)` | TestSortedUniqueBoundEdges/root_handle_names | killed |
| Bsus-roothandles-count | context.go | `checkSortedUniqueStrings(members["root_handle_names"], 1, 128, 1, 129)` | TestSortedUniqueBoundEdges/root_handle_names | killed |
| Bfind-ambig | operations.go | `DecodeFindings(members["ambiguities"], 1025)` | TestFindingsCapEdges/inspect_ambiguities_1024 | killed |
| Bfind-proj | operations.go | `DecodeFindings(members["findings"], 4097)` | TestFindingsCapEdges/projection-plan_findings_4096 | killed |
| Bfind-validate | operations.go | `DecodeFindings(members["findings"], 4097)` | TestFindingsCapEdges/validate_findings_4096 | killed |
| Bfind-doctor | probe.go | `DecodeFindings(members["findings"], 4097)` | TestFindingsCapEdges/doctor_findings_4096 | killed |
| Eequiv-ext-min | decode.go | `len(name) < 2` | full package suite | equivalent (full suite green) |
| Bext-max | decode.go | `len(name) > 254` | TestExtensionKeyBoundEdges | killed |
| Bres-maxobjects | context.go | `checkUint53Bounds(members["max_objects"], 0, maxUint53)` | TestResourceLimitsMinimumEdges | killed |
| Bres-maxtotal | context.go | `checkUint53Bounds(members["max_total_bytes"], 0, maxUint53)` | TestResourceLimitsMinimumEdges | killed |
| Bres-maxsingle | context.go | `checkUint53Bounds(members["max_single_object_bytes"], 0, maxUint53)` | TestResourceLimitsMinimumEdges | killed |
| Bseq-min | probe.go | `checkUint53Bounds(members["registry_sequence"], 1, maxUint53)` | TestDoctorSequenceMinimum | killed |
| Bframe-req-max | protocol.go | `if len(frame) > MaxFrameBytes+1 {` | TestFrameBoundEdges | killed |
| Bframe-succ-max | protocol.go | `if len(frame) > MaxFrameBytes+1 {` | TestFrameBoundEdges | killed |
| Bframe-fail-max | protocol.go | `if len(frame) > MaxFrameBytes+1 {` | TestFrameBoundEdges | killed |
| Bframe-req-empty | protocol.go | `if len(frame) < 0 {` | TestEmptyFramesRefuse | killed |
| Bframe-succ-empty | protocol.go | `if len(frame) < 0 {` | TestEmptyFramesRefuse | killed |
| Bframe-fail-empty | protocol.go | `if len(frame) < 0 {` | TestEmptyFramesRefuse | killed |
| Bstrategies-empty | tuple.go | `if len(elements) < 0 {` | TestStrategiesEmptyRefuses | killed |
| Blen-dispositions | operations.go | `if !ok || len(elements) == 0 || len(elements) > 8 {` | TestArrayBoundEdges | killed |
| Blen-argv | operations.go | `if !ok || len(argv) == 0 || len(argv) > 129 {` | TestArrayBoundEdges | killed |
| Blen-contracts | tuple.go | `if len(elements) == 0 || len(elements) > 65 {` | TestArrayBoundEdges | killed |
| Blen-versions | tuple.go | `if !ok || len(versions) == 0 || len(versions) > 33 {` | TestArrayBoundEdges | killed |
| Blen-sources | operations.go | `if uint64(len(elements)) > 65537 {` | TestEveryArmWitnessRefusesAtTheProductionEntry | killed |
| Blen-fidelity | tuple.go | `if uint64(len(elements)) > 1025 {` | TestEveryArmWitnessRefusesAtTheProductionEntry | killed |
| Blen-entryseq | tuple.go | `checkUint53Bounds(members["entry_sequence"], 0, maxUint53)` | TestEveryArmWitnessRefusesAtTheProductionEntry | killed |
| Blen-fixturecount | tuple.go | `checkUint53Bounds(members["fixture_count"], 0, maxUint53)` | TestEveryArmWitnessRefusesAtTheProductionEntry | killed |
| Blen-contractshape | tuple.go | `if len(members) != 2 && len(members) != 3 {` | TestEveryArmWitnessRefusesAtTheProductionEntry | killed |
| Blen-regops | manifest.go | `if !ok || len(elements) != len(operationOrder) && len(elements) != 0 {` | TestRegistryLengthEmptyRefuses | killed |
| Blen-regcaps | manifest.go | `if !ok || len(elements) != len(capabilityOrder) && len(elements) != 0 {` | TestRegistryLengthEmptyRefuses | killed |
| Blen-probecaps | probe.go | `if len(members) != len(capabilityOrder) && len(members) != 0 {` | TestRegistryLengthEmptyRefuses | killed |
| Blen-dispmap | operations.go | `if len(members) < 0 {` | TestRequiredDispositionsEmptyMapRefuses | killed |
| Bstr-nativekey-max | context.go | `checkStringBounds(members["native_item_key"], 1, 513)` | TestStringBoundEdges | killed |
| Bstr-message-max | context.go | `checkStringBounds(members["message"], 1, 4097)` | TestStringBoundEdges | killed |
| Bstr-remediation-max | context.go | `checkStringBounds(members["remediation"], 1, 4097)` | TestStringBoundEdges | killed |
| Bstr-nativesess-max | context.go | `checkStringBounds(members["native_session_id"], 1, 513)` | TestStringBoundEdges | killed |
| Bstr-opaqueref-max | context.go | `checkStringBounds(members["opaque_source_ref"], 1, 513)` | TestStringBoundEdges | killed |
| Bstr-argvword-max | operations.go | `checkStringBounds(element, 1, 4097)` | TestStringBoundEdges | killed |
| Bstr-tuplesuite-max | tuple.go | `checkStringBounds(members["suite_revision"], 1, 129)` | TestStringBoundEdges | killed |
| Bstr-clifamily-max | tuple.go | `checkStringBounds(members["native_cli_family"], 1, 129)` | TestStringBoundEdges | killed |
| Bstr-fidelitycode-max | tuple.go | `checkStringBounds(members["code"], 1, 129)` | TestStringBoundEdges | killed |
| Bstr-fidelityclass-max | tuple.go | `checkStringBounds(members["affected_class"], 1, 129)` | TestStringBoundEdges | killed |
| Bstr-fidelitydetail-max | tuple.go | `if _, ok := checkStringBounds(members["detail"], 1, 4097); !ok {` | TestStringBoundEdges | killed |
| Bstr-revocation-max | tuple.go | `checkStringBounds(members["revocation_reason"], 1, 4097)` | TestStringBoundEdges | killed |
| Bstr-display-min | manifest.go | `checkStringBounds(members["display_name"], 0, 128)` | TestDecodeManifestValueRules | killed |
| Bsus-warnings-element | probe.go | `checkSortedUniqueStrings(members["warnings"], 0, 2049, 0, 1024)` | TestDecodeProbeClosedRules | killed |
| Blen-argv-min | operations.go | `if !ok || len(argv) < 0 || len(argv) > 128 {` | TestArrayBoundEdges | killed |
| Blen-contracts-min | tuple.go | `if len(elements) < 0 || len(elements) > 64 {` | TestArrayBoundEdges | killed |
| Blen-versions-min | tuple.go | `if !ok || len(versions) < 0 || len(versions) > 32 {` | TestArrayBoundEdges | killed |
| Blen-dispelements-min | operations.go | `if !ok || len(elements) < 0 || len(elements) > 7 {` | TestArrayBoundEdges | killed |
| Blen-strategies-len | tuple.go | `if len(entry.Strategies) != 1 && len(entry.Strategies) != 2 || entry.Strategies[0] != "...` | TestCheckTupleAdmissionRefusals | killed |
| Ibind-manifest | discovery.go | `if context.ManifestDigest != binding.AdapterManifestDigest && binding.AdapterManifestDi...` | TestCheckCallBindingZeroFactsRefuse | killed |
| Ibind-exe | discovery.go | `if context.ExecutableSHA256 != binding.ExecutableSHA256 && binding.ExecutableSHA256 != ...` | TestCheckCallBindingZeroFactsRefuse | killed |
| Ibind-tuple | discovery.go | `if context.Environment != admitted && admitted != (Tuple{}) {` | TestCheckCallBindingZeroFactsRefuse | killed |
| Ituple-bind-kind | tuple.go | `entry.Key.CandidateKind != binding.CandidateKind && binding.CandidateKind != "" ||` | TestTupleAdmissionZeroFactsRefuse | killed |
| Ituple-bind-provdig | tuple.go | `entry.Key.ProviderManifestDigest != binding.ProviderManifestDigest && binding.ProviderM...` | TestTupleAdmissionZeroFactsRefuse | killed |
| Ituple-bind-adaptdig | tuple.go | `entry.Key.AdapterManifestDigest != binding.AdapterManifestDigest && binding.AdapterMani...` | TestTupleAdmissionZeroFactsRefuse | killed |
| Iseal-role-sealed | discovery.go | `if sealed.Role != fresh.Role && sealed.Role != "" ||` | TestBindingEqualityZeroFactsRefuse/sealed_role | killed |
| Iseal-provider-fresh | discovery.go | `sealed.ProviderID != fresh.ProviderID && fresh.ProviderID != "" ||` | TestBindingEqualityZeroFactsRefuse/fresh_provider | killed |
| Iseal-kind-sealed | discovery.go | `sealed.CandidateKind != fresh.CandidateKind && sealed.CandidateKind != "" ||` | TestBindingEqualityZeroFactsRefuse/sealed_kind | killed |
| Iseal-path-fresh | discovery.go | `sealed.ExecutablePath != fresh.ExecutablePath && fresh.ExecutablePath != "" ||` | TestBindingEqualityZeroFactsRefuse/fresh_path | killed |
| Iseal-owner-sealed | discovery.go | `sealed.OwnerIdentity != fresh.OwnerIdentity && sealed.OwnerIdentity != "" ||` | TestBindingEqualityZeroFactsRefuse/sealed_owner | killed |
| Iseal-exe-fresh | discovery.go | `sealed.ExecutableSHA256 != fresh.ExecutableSHA256 && fresh.ExecutableSHA256 != "" ||` | TestBindingEqualityZeroFactsRefuse/fresh_executable | killed |
| Iseal-provdig-sealed | discovery.go | `sealed.ProviderManifestDigest != fresh.ProviderManifestDigest && sealed.ProviderManifes...` | TestBindingEqualityZeroFactsRefuse/sealed_provider_digest | killed |
| Iseal-adaptdig-fresh | discovery.go | `sealed.AdapterManifestDigest != fresh.AdapterManifestDigest && fresh.AdapterManifestDig...` | TestBindingEqualityZeroFactsRefuse/fresh_adapter_digest | killed |
| Ibind-role | discovery.go | `if role != binding.Role && binding.Role != "" {` | TestCheckCallBindingZeroFactsRefuse | killed |
| Ibind-provider | discovery.go | `if context.ProviderID != binding.ProviderID && binding.ProviderID != "" {` | TestCheckCallBindingZeroFactsRefuse | killed |
| Ibind-caller-provider | discovery.go | `if context.ProviderID != binding.ProviderID && context.ProviderID != "" {` | TestIdentityZeroRows/call_binding_caller_zeros | killed |
| Ibind-caller-manifest | discovery.go | `if context.ManifestDigest != binding.AdapterManifestDigest && context.ManifestDigest !=...` | TestIdentityZeroRows/call_binding_caller_zeros | killed |
| Ibind-caller-exe | discovery.go | `if context.ExecutableSHA256 != binding.ExecutableSHA256 && context.ExecutableSHA256 != ...` | TestIdentityZeroRows/call_binding_caller_zeros | killed |
| Ibind-caller-role | discovery.go | `if role != binding.Role && role != "" {` | TestIdentityZeroRows/call_binding_caller_zeros | killed |
| Idisc-manifest | discovery.go | `if manifest.ProviderID != candidate.ProviderID && manifest.ProviderID != "" {` | TestIdentityZeroRows/discover_provider_zeros | killed |
| Idisc-candidate | discovery.go | `if manifest.ProviderID != candidate.ProviderID && candidate.ProviderID != "" {` | TestIdentityZeroRows/discover_provider_zeros | killed |
| Iman-schema | manifest.go | `if schema, ok := rawString(members["schema"]); !ok || schema != manifestSchema && schem...` | TestDecodeManifestClosedMemberRules | killed |
| Iman-version | manifest.go | `if version, ok := rawString(members["schema_version"]); !ok || version != manifestSchem...` | TestDecodeManifestClosedMemberRules | killed |
| Iman-ops | manifest.go | `if !ok || name != string(operationOrder[index]) && name != "" {` | TestRegistryElementEmptyRefuses | killed |
| Iman-caps | manifest.go | `if !ok || name != capabilityOrder[index] && name != "" {` | TestRegistryElementEmptyRefuses | killed |
| Eequiv-candidate-digest | operations.go | `if candidate != digest && candidate != "" {` | full package suite | equivalent (behavioral green; roster fails closed on the added clause) |
| Iops-occurrences | operations.go | `if occurrences != 1 && occurrences != 0 {` | TestResumePlanIdentityComplement | killed |
| Iops-mode | operations.go | `if mode != facts.ValidateMode && facts.ValidateMode != "" {` | TestValidateResultRules | killed |
| Iprobe-schema | probe.go | `if schema, ok := rawString(members["schema"]); !ok || schema != probeSchema && schema !...` | TestIdentityZeroRows/probe_schema_empty | killed |
| Iprobe-version | probe.go | `if version, ok := rawString(members["schema_version"]); !ok || version != probeSchemaVe...` | TestIdentityZeroRows/probe_schema_empty | killed |
| Icheckprobe-392 | probe.go | `if probe.ProviderID != facts.ExpectedProviderID && probe.ProviderID != "" {` | TestIdentityZeroRows/probe_provider_zeros | killed |
| Icheckprobe-399 | probe.go | `if probe.ProviderID != facts.Manifest.ProviderID && facts.Manifest.ProviderID != "" {` | TestIdentityZeroRows/probe_provider_zeros | killed |
| Icheckprobe-406a | probe.go | `if probe.ManifestDigest != facts.ManifestDigest && probe.ManifestDigest != "" {` | TestIdentityZeroRows/probe_digest_zeros | killed |
| Icheckprobe-406b | probe.go | `if probe.ManifestDigest != facts.ManifestDigest && facts.ManifestDigest != "" {` | TestIdentityZeroRows/probe_digest_zeros | killed |
| Icheckprobe-413 | probe.go | `if probe.AdapterVersion != facts.Manifest.AdapterVersion && facts.Manifest.AdapterVersi...` | TestIdentityZeroRows/probe_version_zeros | killed |
| Icheckprobe-420 | probe.go | `if probe.Environment.EnvironmentID != facts.Manifest.EnvironmentID && probe.Environment...` | TestIdentityZeroRows/probe_tuple_environment_zero | killed |
| Icheckprobe-427 | probe.go | `if probe.Environment.AdapterVersion != probe.AdapterVersion && probe.Environment.Adapte...` | TestIdentityZeroRows/probe_version_zeros | killed |
| Ireq-provider | probe.go | `if expectedProviderID != candidate.ProviderID && expectedProviderID != "" {` | TestIdentityZeroRows/probe_request_zeros | killed |
| Ireq-candidate | probe.go | `if expectedProviderID != candidate.ProviderID && candidate.ProviderID != "" {` | TestIdentityZeroRows/probe_request_zeros | killed |
| Ireq-kind | probe.go | `if expectedKind != candidate.Kind && expectedKind != "" {` | TestZeroValueHostFactsRefuse | killed |
| Iloop-status | probe.go | `if !present || value.Status != "available" && value.Status != "" || !value.Enabled {` | TestIdentityZeroRows/provider_status_zero | killed |
| Idoctor-direction | probe.go | `if Direction(direction) != sentDirection && sentDirection != "" {` | TestIdentityZeroRows/doctor_direction_zero | killed |
| Idoctor-entry | probe.go | `if result.EntryStatus != "accepted" && result.EntryStatus != "" {` | TestIdentityZeroRows/doctor_entry_status_zero | killed |
| Ienv-req-protocol | protocol.go | `if protocol, ok := rawString(members["protocol"]); !ok || protocol != ProtocolID && pro...` | TestEnvelopeEchoEmptyStringRefuses | killed |
| Ienv-req-version | protocol.go | `if version, ok := rawString(members["protocol_version"]); !ok || version != ProtocolVer...` | TestEnvelopeEchoEmptyStringRefuses | killed |
| Ienv-succ-requestid | protocol.go | `if requestID, ok := rawString(members["request_id"]); !ok || requestID != want.RequestI...` | TestEnvelopeEchoEmptyStringRefuses | killed |
| Ienv-fail-operation | protocol.go | `if operation, ok := rawString(members["operation"]); !ok || operation != string(want.Op...` | TestEnvelopeEchoEmptyStringRefuses | killed |
| Ituple-direction | tuple.go | `if entry.Key.Direction != direction && entry.Key.Direction != "" {` | TestTupleAdmissionZeroFactsRefuse/empty_direction | killed |
| Ituple-environment | tuple.go | `if entry.Key.Environment != environment && entry.Key.Environment.EnvironmentID != "" {` | TestTupleAdmissionZeroFactsRefuse/zero_environment | killed |
| Ituple-key-provider | tuple.go | `if entry.Key.ProviderID != binding.ProviderID && entry.Key.ProviderID != "" ||` | TestTupleAdmissionZeroFactsRefuse | killed |
| Ituple-key-kind | tuple.go | `entry.Key.CandidateKind != binding.CandidateKind && entry.Key.CandidateKind != "" ||` | TestTupleAdmissionZeroFactsRefuse | killed |
| Ituple-key-exe | tuple.go | `entry.Key.ExecutableSHA256 != binding.ExecutableSHA256 && entry.Key.ExecutableSHA256 !=...` | TestTupleAdmissionZeroFactsRefuse | killed |
| Ituple-key-provdig | tuple.go | `entry.Key.ProviderManifestDigest != binding.ProviderManifestDigest && entry.Key.Provide...` | TestTupleAdmissionZeroFactsRefuse | killed |
| Ituple-key-adaptdig | tuple.go | `entry.Key.AdapterManifestDigest != binding.AdapterManifestDigest && entry.Key.AdapterMa...` | TestTupleAdmissionZeroFactsRefuse | killed |
| Ituple-bind-provider | tuple.go | `if entry.Key.ProviderID != binding.ProviderID && binding.ProviderID != "" ||` | TestTupleAdmissionZeroFactsRefuse | killed |
| Ituple-bind-exe | tuple.go | `entry.Key.ExecutableSHA256 != binding.ExecutableSHA256 && binding.ExecutableSHA256 != "...` | TestTupleAdmissionZeroFactsRefuse | killed |
| Ituple-status | tuple.go | `if entry.Status != "accepted" && entry.Status != "" {` | TestTupleAdmissionZeroFactsRefuse/empty_status | killed |
| Ituple-strategies | tuple.go | `if len(entry.Strategies) != 1 || entry.Strategies[0] != "archive_only" && entry.Strateg...` | TestTupleAdmissionZeroFactsRefuse/empty_strategy_word | killed |
| Icount-selector | context.go | `if count != 1 && count != 0 {` | TestDecodeSourceSelectorComplement | killed |
| Idigest-zero | context.go | `if digest.String() != context.RequestDigest && context.RequestDigest != "" {` | TestIdentityZeroRows/request_digest_zero | killed |
| Eequiv-platforms-element | manifest.go | `checkSortedUniqueStrings(members["platforms"], 1, 17, 1, 4)` | full package suite | equivalent (full suite green) |
| Eequiv-platforms-count | manifest.go | `checkSortedUniqueStrings(members["platforms"], 1, 16, 1, 5)` | full package suite | equivalent (full suite green) |
| Eequiv-validate-mode | operations.go | `if mode != facts.ValidateMode && mode != "" {` | full package suite | equivalent (behavioral green; roster fails closed on the added clause) |
| Eequiv-capability-status | probe.go | `if enabled && status != "available" && status != "" {` | full package suite | equivalent (behavioral green; roster fails closed on the added clause) |
