# TASK-260830-2z3se0 — reviewer mutation log, CR revision 5

Method: every row is a byte-offset edit into a production file, asserted
PRESENT after writing, driven through the **whole package suite**
(`go test ./internal/sessadapter/ -count=1`), then restored with a sha256
comparison against the pre-mutation file. The worktree tree OID was
`c0174e8761518432e55ed5bdcf915959da9bd9f4` before and after the run.

`census-only` means the only failing tests were the source-text censuses
(`TestIdentityGatesAreCensused`, `TestBoundGuardsAreCensused`,
`TestNoUnregisteredInlineVocabularies`, ...). Those are never counted as
behavioural kills: an identity mutant that adds a `!= ""` clause reddens
the identity census by construction.

`compile-fail` rows are generator artefacts (a string zero compared against
an int or a struct); the four that mattered were re-run typed and appear in
the last section.

## require* both edges (all 19 call sites) — 35 mutants

| mutant | status | behavioural killers |
| --- | --- | --- |
| `operations.go:311\|requireUint53Bounds\|min\|1->0` | killed | TestEveryArmWitnessRefusesAtTheProductionEntry, TestRequestScalarBoundEdges |
| `operations.go:311\|requireUint53Bounds\|max\|65536->65537` | killed | TestRequestScalarBoundEdges |
| `operations.go:315\|requireStringBounds\|min\|1->0` | killed | TestRequestScalarBoundEdges |
| `operations.go:315\|requireStringBounds\|max\|1024->1025` | killed | TestRequestScalarBoundEdges |
| `operations.go:333\|requireStringBounds\|min\|1->0` | killed | TestEveryArmWitnessRefusesAtTheProductionEntry, TestRequestScalarBoundEdges |
| `operations.go:333\|requireStringBounds\|max\|512->513` | killed | TestRequestScalarBoundEdges |
| `operations.go:352\|requireUint53Bounds\|min\|1->0` | killed | TestRequestScalarBoundEdges |
| `operations.go:352\|requireUint53Bounds\|max\|65536->65537` | killed | TestRequestScalarBoundEdges |
| `operations.go:355\|requireUint53Bounds\|min\|1->0` | killed | TestRequestScalarBoundEdges |
| `operations.go:407\|requireStringBounds\|min\|1->0` | killed | TestRequestScalarBoundEdges |
| `operations.go:407\|requireStringBounds\|max\|512->513` | killed | TestRequestScalarBoundEdges |
| `operations.go:450\|requireStringBounds\|min\|1->0` | killed | TestRequestScalarBoundEdges |
| `operations.go:450\|requireStringBounds\|max\|512->513` | killed | TestRequestScalarBoundEdges |
| `operations.go:486\|requireStringBounds\|min\|1->0` | killed | TestRequestScalarBoundEdges |
| `operations.go:486\|requireStringBounds\|max\|512->513` | killed | TestRequestScalarBoundEdges |
| `operations.go:497\|requireStringBounds\|min\|1->0` | killed | TestRequestScalarBoundEdges |
| `operations.go:497\|requireStringBounds\|max\|512->513` | killed | TestRequestScalarBoundEdges |
| `operations.go:746\|requireStringBounds\|min\|1->0` | killed | TestSuccessScalarBoundEdges |
| `operations.go:746\|requireStringBounds\|max\|1024->1025` | killed | TestSuccessScalarBoundEdges |
| `operations.go:766\|requireStringBounds\|min\|1->0` | killed | TestSuccessScalarBoundEdges |
| `operations.go:766\|requireStringBounds\|max\|512->513` | killed | TestSuccessScalarBoundEdges |
| `operations.go:789\|requireStringBounds\|min\|1->0` | killed | TestSuccessScalarBoundEdges |
| `operations.go:789\|requireStringBounds\|max\|512->513` | killed | TestSuccessScalarBoundEdges |
| `operations.go:796\|requireUint53Bounds\|max\|65536->65537` | killed | TestSuccessScalarBoundEdges |
| `operations.go:821\|requireStringBounds\|min\|1->0` | killed | TestSuccessScalarBoundEdges |
| `operations.go:821\|requireStringBounds\|max\|512->513` | killed | TestSuccessScalarBoundEdges |
| `operations.go:830\|requireUint53Bounds\|max\|65536->65537` | killed | TestSuccessScalarBoundEdges |
| `operations.go:895\|requireStringBounds\|min\|1->0` | killed | TestSuccessScalarBoundEdges |
| `operations.go:895\|requireStringBounds\|max\|512->513` | killed | TestSuccessScalarBoundEdges |
| `operations.go:926\|requireStringBounds\|min\|1->0` | killed | TestSuccessScalarBoundEdges |
| `operations.go:926\|requireStringBounds\|max\|512->513` | killed | TestSuccessScalarBoundEdges |
| `operations.go:963\|requireStringBounds\|min\|1->0` | killed | TestSuccessScalarBoundEdges |
| `operations.go:963\|requireStringBounds\|max\|4096->4097` | killed | TestSuccessScalarBoundEdges |
| `operations.go:355\|requireUint53Bounds\|maxCeiling\|max_total_bytes` | survived | — |
| `operations.go:904\|requireUint53Bounds\|maxCeiling\|parsed_event_count` | survived | — |

## check* literal edges, part A — 34 mutants

| mutant | status | behavioural killers |
| --- | --- | --- |
| `context.go:303\|checkSortedUniqueStrings\|minLen\|1->0` | killed | TestDecodeReadAuthorityRules, TestSortedUniqueBoundEdges |
| `context.go:303\|checkSortedUniqueStrings\|maxLen\|128->129` | killed | TestSortedUniqueBoundEdges |
| `context.go:303\|checkSortedUniqueStrings\|minCount\|1->0` | killed | TestDecodeReadAuthorityRules, TestEveryArmWitnessRefusesAtTheProductionEntry |
| `context.go:303\|checkSortedUniqueStrings\|maxCount\|128->129` | killed | TestSortedUniqueBoundEdges |
| `context.go:613\|checkStringBounds\|min\|1->0` | killed | TestDecodeSourceSelectorComplement, TestEveryArmWitnessRefusesAtTheProductionEntry, TestStringBoundEdges |
| `context.go:613\|checkStringBounds\|max\|512->513` | killed | TestStringBoundEdges |
| `context.go:636\|checkStringBounds\|min\|1->0` | killed | TestEveryArmWitnessRefusesAtTheProductionEntry, TestStringBoundEdges |
| `context.go:636\|checkStringBounds\|max\|512->513` | killed | TestStringBoundEdges |
| `context.go:722\|checkUint53Bounds\|min\|1->0` | killed | TestDecodeResourceLimitsBounds, TestEveryArmWitnessRefusesAtTheProductionEntry, TestResourceLimitsMinimumEdges |
| `context.go:730\|checkUint53Bounds\|min\|1->0` | killed | TestEveryArmWitnessRefusesAtTheProductionEntry, TestResourceLimitsMinimumEdges |
| `context.go:738\|checkUint53Bounds\|min\|1->0` | killed | TestDecodeResourceLimitsBounds, TestEveryArmWitnessRefusesAtTheProductionEntry, TestResourceLimitsMinimumEdges |
| `context.go:746\|checkUint53Bounds\|max\|65536->65537` | killed | TestDecodeResourceLimitsBounds, TestEveryArmWitnessRefusesAtTheProductionEntry |
| `context.go:754\|checkUint53Bounds\|max\|65536->65537` | killed | TestDecodeResourceLimitsBounds, TestEveryArmWitnessRefusesAtTheProductionEntry |
| `context.go:847\|checkStringBounds\|min\|1->0` | killed | TestDecodeFindingRules, TestEveryArmWitnessRefusesAtTheProductionEntry, TestStringBoundEdges |
| `context.go:847\|checkStringBounds\|max\|128->129` | killed | TestStringBoundEdges |
| `context.go:855\|checkStringBounds\|min\|1->0` | killed | TestDecodeFindingRules, TestEveryArmWitnessRefusesAtTheProductionEntry, TestStringBoundEdges |
| `context.go:855\|checkStringBounds\|max\|4096->4097` | killed | TestStringBoundEdges |
| `context.go:865\|checkStringBounds\|min\|1->0` | killed | TestDecodeFindingRules, TestEveryArmWitnessRefusesAtTheProductionEntry, TestStringBoundEdges |
| `context.go:865\|checkStringBounds\|max\|4096->4097` | killed | TestStringBoundEdges |
| `context.go:971\|checkStringBounds\|min\|1->0` | killed | TestEveryArmWitnessRefusesAtTheProductionEntry, TestStringBoundEdges |
| `context.go:971\|checkStringBounds\|max\|512->513` | killed | TestStringBoundEdges |
| `manifest.go:163\|checkStringBounds\|min\|1->0` | killed | TestDecodeManifestClosedMemberRules, TestDecodeManifestValueRules, TestEveryArmWitnessRefusesAtTheProductionEntry |
| `manifest.go:163\|checkStringBounds\|max\|128->129` | killed | TestDecodeManifestValueRules |
| `manifest.go:178\|checkStringBounds\|min\|1->0` | killed | TestDecodeManifestClosedMemberRules, TestDecodeManifestValueRules, TestEveryArmWitnessRefusesAtTheProductionEntry, TestStringBoundEdges |
| `manifest.go:178\|checkStringBounds\|max\|256->257` | killed | TestStringBoundEdges |
| `manifest.go:185\|checkSortedUniqueStrings\|minLen\|1->0` | survived | — |
| `manifest.go:185\|checkSortedUniqueStrings\|maxLen\|16->17` | survived | — |
| `manifest.go:185\|checkSortedUniqueStrings\|minCount\|1->0` | killed | TestDecodeManifestValueRules, TestEveryArmWitnessRefusesAtTheProductionEntry |
| `manifest.go:185\|checkSortedUniqueStrings\|maxCount\|4->5` | survived | — |
| `operations.go:391\|checkSortedUniqueDigests\|maxCount\|65536->65537` | survived | — |
| `operations.go:420\|checkSortedUniqueStrings\|minLen\|1->0` | killed | TestSortedUniqueBoundEdges |
| `operations.go:420\|checkSortedUniqueStrings\|maxLen\|128->129` | killed | TestSortedUniqueBoundEdges |
| `operations.go:420\|checkSortedUniqueStrings\|maxCount\|128->129` | killed | TestSortedUniqueBoundEdges |
| `operations.go:846\|checkSortedUniqueDigests\|maxCount\|65536->65537` | survived | — |

## check* literal edges, part B — 34 mutants

| mutant | status | behavioural killers |
| --- | --- | --- |
| `operations.go:853\|checkSortedUniqueDigests\|maxCount\|65536->65537` | survived | — |
| `operations.go:864\|checkSortedUniqueDigests\|maxCount\|65536->65537` | survived | — |
| `operations.go:884\|checkSortedUniqueStrings\|minLen\|1->0` | killed | TestSortedUniqueBoundEdges |
| `operations.go:884\|checkSortedUniqueStrings\|maxLen\|512->513` | killed | TestSortedUniqueBoundEdges |
| `operations.go:884\|checkSortedUniqueStrings\|maxCount\|65536->65537` | survived | — |
| `operations.go:907\|checkSortedUniqueStrings\|minLen\|1->0` | killed | TestSortedUniqueBoundEdges |
| `operations.go:907\|checkSortedUniqueStrings\|maxLen\|512->513` | killed | TestSortedUniqueBoundEdges |
| `operations.go:907\|checkSortedUniqueStrings\|maxCount\|1024->1025` | killed | TestSortedUniqueBoundEdges |
| `operations.go:942\|checkStringBounds\|min\|1->0` | killed | TestEveryArmWitnessRefusesAtTheProductionEntry, TestStringBoundEdges |
| `operations.go:942\|checkStringBounds\|max\|4096->4097` | killed | TestStringBoundEdges |
| `operations.go:966\|checkSortedUniqueStrings\|minLen\|1->0` | killed | TestSortedUniqueBoundEdges |
| `operations.go:966\|checkSortedUniqueStrings\|maxLen\|256->257` | killed | TestSortedUniqueBoundEdges |
| `operations.go:966\|checkSortedUniqueStrings\|maxCount\|128->129` | killed | TestSortedUniqueBoundEdges |
| `probe.go:144\|checkStringBounds\|max\|2048->2049` | killed | TestDecodeProbeClosedRules, TestEveryArmWitnessRefusesAtTheProductionEntry, TestStringBoundEdges |
| `probe.go:303\|checkSortedUniqueStrings\|maxLen\|2048->2049` | killed | TestDecodeProbeClosedRules |
| `probe.go:303\|checkSortedUniqueStrings\|maxCount\|1024->1025` | killed | TestSortedUniqueBoundEdges |
| `tuple.go:96\|checkStringBounds\|min\|1->0` | killed | TestDecodeTupleRefusals, TestEveryArmWitnessRefusesAtTheProductionEntry, TestStringBoundEdges |
| `tuple.go:96\|checkStringBounds\|max\|128->129` | killed | TestStringBoundEdges |
| `tuple.go:594\|checkStringBounds\|min\|1->0` | killed | TestEveryArmWitnessRefusesAtTheProductionEntry, TestStringBoundEdges |
| `tuple.go:594\|checkStringBounds\|max\|128->129` | killed | TestStringBoundEdges |
| `tuple.go:641\|checkUint53Bounds\|min\|1->0` | killed | TestDecodeTupleEntryCoherence, TestEveryArmWitnessRefusesAtTheProductionEntry |
| `tuple.go:737\|checkStringBounds\|min\|1->0` | killed | TestEveryArmWitnessRefusesAtTheProductionEntry, TestStringBoundEdges |
| `tuple.go:737\|checkStringBounds\|max\|128->129` | killed | TestStringBoundEdges |
| `tuple.go:822\|checkStringBounds\|min\|1->0` | killed | TestEveryArmWitnessRefusesAtTheProductionEntry, TestStringBoundEdges |
| `tuple.go:822\|checkStringBounds\|max\|128->129` | killed | TestStringBoundEdges |
| `tuple.go:830\|checkStringBounds\|min\|1->0` | killed | TestEveryArmWitnessRefusesAtTheProductionEntry, TestStringBoundEdges |
| `tuple.go:830\|checkStringBounds\|max\|128->129` | killed | TestStringBoundEdges |
| `tuple.go:846\|checkStringBounds\|min\|1->0` | killed | TestEveryArmWitnessRefusesAtTheProductionEntry, TestStringBoundEdges |
| `tuple.go:846\|checkStringBounds\|max\|4096->4097` | killed | TestStringBoundEdges |
| `tuple.go:981\|checkStringBounds\|min\|1->0` | killed | TestEveryArmWitnessRefusesAtTheProductionEntry, TestStringBoundEdges |
| `tuple.go:981\|checkStringBounds\|max\|256->257` | killed | TestStringBoundEdges |
| `tuple.go:1079\|checkUint53Bounds\|min\|1->0` | killed | TestEveryArmWitnessRefusesAtTheProductionEntry |
| `tuple.go:1155\|checkStringBounds\|min\|1->0` | killed | TestEveryArmWitnessRefusesAtTheProductionEntry, TestStringBoundEdges |
| `tuple.go:1155\|checkStringBounds\|max\|4096->4097` | killed | TestStringBoundEdges |

## len guards in if-conditions + environment identity rows — 22 mutants

| mutant | status | behavioural killers |
| --- | --- | --- |
| `decode.go:441\|widen-lower\|<\|3->2` | survived | — |
| `decode.go:441\|widen-upper\|>\|253->254` | killed | TestExtensionKeyBoundEdges |
| `operations.go:584\|never-empty\|==\|0->-1` | killed | TestEveryArmWitnessRefusesAtTheProductionEntry, TestProjectionPlanDispositionRules, TestRequiredDispositionsEmptyMapRefuses |
| `operations.go:593\|never-empty\|==\|0->-1` | killed | TestArrayBoundEdges |
| `operations.go:593\|widen-upper\|>\|7->8` | killed | TestArrayBoundEdges, TestEveryArmWitnessRefusesAtTheProductionEntry |
| `operations.go:933\|never-empty\|==\|0->-1` | killed | TestArrayBoundEdges, TestEveryArmWitnessRefusesAtTheProductionEntry |
| `operations.go:933\|widen-upper\|>\|128->129` | killed | TestArrayBoundEdges |
| `operations.go:1016\|widen-upper\|>\|65536->65537` | killed | TestEveryArmWitnessRefusesAtTheProductionEntry |
| `operations.go:1046\|widen-upper\|>\|9->10` | killed | TestEveryArmWitnessRefusesAtTheProductionEntry |
| `protocol.go:242\|never-empty\|==\|0->-1` | killed | TestDecodeRequestFrameRefusals, TestEmptyFramesRefuse, TestEveryArmWitnessRefusesAtTheProductionEntry |
| `protocol.go:348\|never-empty\|==\|0->-1` | killed | TestEmptyFramesRefuse, TestEveryArmWitnessRefusesAtTheProductionEntry |
| `protocol.go:438\|never-empty\|==\|0->-1` | killed | TestEmptyFramesRefuse, TestEveryArmWitnessRefusesAtTheProductionEntry |
| `protocol.go:527\|never-empty\|==\|0->-1` | census-only | TestBoundGuardsAreCensused |
| `tuple.go:790\|widen-upper\|>\|1024->1025` | killed | TestEveryArmWitnessRefusesAtTheProductionEntry |
| `tuple.go:902\|never-empty\|==\|0->-1` | killed | TestDecodeTupleEntryCoherence, TestEveryArmWitnessRefusesAtTheProductionEntry, TestStrategiesEmptyRefuses |
| `tuple.go:948\|never-empty\|==\|0->-1` | killed | TestArrayBoundEdges, TestEveryArmWitnessRefusesAtTheProductionEntry |
| `tuple.go:948\|widen-upper\|>\|64->65` | killed | TestArrayBoundEdges |
| `tuple.go:998\|never-empty\|==\|0->-1` | killed | TestArrayBoundEdges, TestEveryArmWitnessRefusesAtTheProductionEntry |
| `tuple.go:998\|widen-upper\|>\|32->33` | killed | TestArrayBoundEdges |
| `ID3-callbinding-env` | killed | TestCheckCallBindingZeroFactsRefuse |
| `ID3-tuple-env-entry` | killed | TestTupleAdmissionZeroFactsRefuse |
| `ID3-tuple-env-caller` | census-only | TestIdentityGatesAreCensused |

## identity NEQ, both sides, part A — 60 mutants

| mutant | status | behavioural killers |
| --- | --- | --- |
| `context.go\|VerifyRequestDigest\|digest.String() != context.RequestDigest\|0\|Y` | killed | TestIdentityZeroRows |
| `context.go\|CheckFreshSink\|authority.Mode != "fresh_sink"\|0\|X` | census-only | TestIdentityGatesAreCensused |
| `context.go\|CheckFreshSink\|authority.Mode != "fresh_sink"\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `context.go\|DecodeSourceSelector\|count != 1\|0\|X` | compile-fail | — |
| `context.go\|DecodeSourceSelector\|count != 1\|0\|Y` | compile-fail | — |
| `discovery.go\|Discover\|manifest.ProviderID != candidate.ProviderID\|0\|X` | killed | TestIdentityZeroRows |
| `discovery.go\|Discover\|manifest.ProviderID != candidate.ProviderID\|0\|Y` | killed | TestIdentityZeroRows |
| `discovery.go\|CheckBindingEquality\|sealed.Role != fresh.Role\|0\|X` | killed | TestBindingEqualityZeroFactsRefuse |
| `discovery.go\|CheckBindingEquality\|sealed.Role != fresh.Role\|0\|Y` | killed | TestBindingEqualityZeroFactsRefuse |
| `discovery.go\|CheckBindingEquality\|sealed.ProviderID != fresh.ProviderID\|0\|X` | killed | TestBindingEqualityZeroFactsRefuse |
| `discovery.go\|CheckBindingEquality\|sealed.ProviderID != fresh.ProviderID\|0\|Y` | killed | TestBindingEqualityZeroFactsRefuse |
| `discovery.go\|CheckBindingEquality\|sealed.CandidateKind != fresh.CandidateKind\|0\|X` | killed | TestBindingEqualityZeroFactsRefuse |
| `discovery.go\|CheckBindingEquality\|sealed.CandidateKind != fresh.CandidateKind\|0\|Y` | killed | TestBindingEqualityZeroFactsRefuse |
| `discovery.go\|CheckBindingEquality\|sealed.ExecutablePath != fresh.ExecutablePath\|0\|X` | killed | TestBindingEqualityZeroFactsRefuse |
| `discovery.go\|CheckBindingEquality\|sealed.ExecutablePath != fresh.ExecutablePath\|0\|Y` | killed | TestBindingEqualityZeroFactsRefuse |
| `discovery.go\|CheckBindingEquality\|sealed.OwnerIdentity != fresh.OwnerIdentity\|0\|X` | killed | TestBindingEqualityZeroFactsRefuse |
| `discovery.go\|CheckBindingEquality\|sealed.OwnerIdentity != fresh.OwnerIdentity\|0\|Y` | killed | TestBindingEqualityZeroFactsRefuse |
| `discovery.go\|CheckBindingEquality\|sealed.ExecutableSHA256 != fresh.ExecutableSHA256\|0\|X` | killed | TestBindingEqualityZeroFactsRefuse |
| `discovery.go\|CheckBindingEquality\|sealed.ExecutableSHA256 != fresh.ExecutableSHA256\|0\|Y` | killed | TestBindingEqualityZeroFactsRefuse |
| `discovery.go\|CheckBindingEquality\|sealed.ProviderManifestDigest != fresh.ProviderManifestDigest\|0\|X` | killed | TestBindingEqualityZeroFactsRefuse |
| `discovery.go\|CheckBindingEquality\|sealed.ProviderManifestDigest != fresh.ProviderManifestDigest\|0\|Y` | killed | TestBindingEqualityZeroFactsRefuse |
| `discovery.go\|CheckBindingEquality\|sealed.AdapterManifestDigest != fresh.AdapterManifestDigest\|0\|X` | killed | TestBindingEqualityZeroFactsRefuse |
| `discovery.go\|CheckBindingEquality\|sealed.AdapterManifestDigest != fresh.AdapterManifestDigest\|0\|Y` | killed | TestBindingEqualityZeroFactsRefuse |
| `discovery.go\|CheckCallBinding\|role != binding.Role\|0\|X` | killed | TestIdentityZeroRows |
| `discovery.go\|CheckCallBinding\|role != binding.Role\|0\|Y` | killed | TestCheckCallBindingZeroFactsRefuse |
| `discovery.go\|CheckCallBinding\|context.ProviderID != binding.ProviderID\|0\|X` | killed | TestIdentityZeroRows |
| `discovery.go\|CheckCallBinding\|context.ProviderID != binding.ProviderID\|0\|Y` | killed | TestCheckCallBindingZeroFactsRefuse |
| `discovery.go\|CheckCallBinding\|context.ManifestDigest != binding.AdapterManifestDigest\|0\|X` | killed | TestIdentityZeroRows |
| `discovery.go\|CheckCallBinding\|context.ManifestDigest != binding.AdapterManifestDigest\|0\|Y` | killed | TestCheckCallBindingZeroFactsRefuse |
| `discovery.go\|CheckCallBinding\|context.ExecutableSHA256 != binding.ExecutableSHA256\|0\|X` | killed | TestIdentityZeroRows |
| `discovery.go\|CheckCallBinding\|context.ExecutableSHA256 != binding.ExecutableSHA256\|0\|Y` | killed | TestCheckCallBindingZeroFactsRefuse |
| `discovery.go\|CheckCallBinding\|context.Environment != admitted\|0\|X` | compile-fail | — |
| `discovery.go\|CheckCallBinding\|context.Environment != admitted\|0\|Y` | compile-fail | — |
| `manifest.go\|DecodeManifest\|schema != manifestSchema\|0\|X` | killed | TestDecodeManifestClosedMemberRules |
| `manifest.go\|DecodeManifest\|schema != manifestSchema\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `manifest.go\|DecodeManifest\|version != manifestSchemaVersion\|0\|X` | killed | TestDecodeManifestClosedMemberRules |
| `manifest.go\|DecodeManifest\|version != manifestSchemaVersion\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `manifest.go\|checkOperationRegistry\|name != string(operationOrder[index])\|0\|X` | killed | TestRegistryElementEmptyRefuses |
| `manifest.go\|checkOperationRegistry\|name != string(operationOrder[index])\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `manifest.go\|checkCapabilityRegistry\|name != capabilityOrder[index]\|0\|X` | killed | TestRegistryElementEmptyRefuses |
| `manifest.go\|checkCapabilityRegistry\|name != capabilityOrder[index]\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `operations.go\|checkRequiredDispositions\|previous != ""\|0\|X` | compile-fail | — |
| `operations.go\|checkSuccessScalars\|candidate != digest\|0\|X` | census-only | TestIdentityGatesAreCensused |
| `operations.go\|checkSuccessScalars\|candidate != digest\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `operations.go\|checkSuccessScalars\|occurrences != 1\|0\|X` | compile-fail | — |
| `operations.go\|checkSuccessScalars\|occurrences != 1\|0\|Y` | compile-fail | — |
| `operations.go\|checkExcludedClasses\|previous != ""\|0\|X` | compile-fail | — |
| `operations.go\|checkValidateResult\|mode != facts.ValidateMode\|0\|X` | census-only | TestIdentityGatesAreCensused |
| `operations.go\|checkValidateResult\|mode != facts.ValidateMode\|0\|Y` | killed | TestValidateResultRules |
| `probe.go\|decodeCapabilityValue\|status != "available"\|0\|X` | census-only | TestIdentityGatesAreCensused |
| `probe.go\|decodeCapabilityValue\|status != "available"\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `probe.go\|DecodeProbe\|schema != probeSchema\|0\|X` | killed | TestIdentityZeroRows |
| `probe.go\|DecodeProbe\|schema != probeSchema\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `probe.go\|DecodeProbe\|version != probeSchemaVersion\|0\|X` | killed | TestIdentityZeroRows |
| `probe.go\|DecodeProbe\|version != probeSchemaVersion\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `probe.go\|CheckProbe\|probe.ProviderID != facts.ExpectedProviderID\|0\|X` | killed | TestIdentityZeroRows |
| `probe.go\|CheckProbe\|probe.ProviderID != facts.ExpectedProviderID\|0\|Y` | killed | TestZeroValueHostFactsRefuse |
| `probe.go\|CheckProbe\|probe.ProviderID != facts.Manifest.ProviderID\|0\|X` | census-only | TestIdentityGatesAreCensused |
| `probe.go\|CheckProbe\|probe.ProviderID != facts.Manifest.ProviderID\|0\|Y` | killed | TestIdentityZeroRows |
| `probe.go\|CheckProbe\|probe.ManifestDigest != facts.ManifestDigest\|0\|X` | killed | TestIdentityZeroRows |

## identity NEQ, both sides, part B — 59 mutants

| mutant | status | behavioural killers |
| --- | --- | --- |
| `probe.go\|CheckProbe\|probe.ManifestDigest != facts.ManifestDigest\|0\|Y` | killed | TestIdentityZeroRows |
| `probe.go\|CheckProbe\|probe.AdapterVersion != facts.Manifest.AdapterVersion\|0\|X` | killed | TestIdentityZeroRows |
| `probe.go\|CheckProbe\|probe.AdapterVersion != facts.Manifest.AdapterVersion\|0\|Y` | killed | TestIdentityZeroRows |
| `probe.go\|CheckProbe\|probe.Environment.EnvironmentID != facts.Manifest.EnvironmentID\|0\|X` | killed | TestIdentityZeroRows |
| `probe.go\|CheckProbe\|probe.Environment.EnvironmentID != facts.Manifest.EnvironmentID\|0\|Y` | killed | TestZeroValueHostFactsRefuse |
| `probe.go\|CheckProbe\|probe.Environment.AdapterVersion != probe.AdapterVersion\|0\|X` | killed | TestIdentityZeroRows |
| `probe.go\|CheckProbe\|probe.Environment.AdapterVersion != probe.AdapterVersion\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `probe.go\|CheckProbeRequest\|expectedProviderID != candidate.ProviderID\|0\|X` | killed | TestIdentityZeroRows |
| `probe.go\|CheckProbeRequest\|expectedProviderID != candidate.ProviderID\|0\|Y` | killed | TestIdentityZeroRows |
| `probe.go\|CheckProbeRequest\|expectedKind != candidate.Kind\|0\|X` | killed | TestZeroValueHostFactsRefuse |
| `probe.go\|CheckProbeRequest\|expectedKind != candidate.Kind\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `probe.go\|CheckTargetWriteGates\|value.Status != "available"\|0\|X` | killed | TestIdentityZeroRows |
| `probe.go\|CheckTargetWriteGates\|value.Status != "available"\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `probe.go\|DecodeDoctorResult\|Direction(direction) != sentDirection\|0\|X` | census-only | TestIdentityGatesAreCensused |
| `probe.go\|DecodeDoctorResult\|Direction(direction) != sentDirection\|0\|Y` | killed | TestIdentityZeroRows |
| `probe.go\|CheckDoctorHealthy\|result.EntryStatus != "accepted"\|0\|X` | killed | TestIdentityZeroRows |
| `probe.go\|CheckDoctorHealthy\|result.EntryStatus != "accepted"\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `probe.go\|DoctorRequiredCapabilities\|direction != DirectionTargetWrite\|0\|X` | census-only | TestIdentityGatesAreCensused |
| `probe.go\|DoctorRequiredCapabilities\|direction != DirectionTargetWrite\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `protocol.go\|DecodeRequestFrame\|protocol != ProtocolID\|0\|X` | killed | TestEnvelopeEchoEmptyStringRefuses |
| `protocol.go\|DecodeRequestFrame\|protocol != ProtocolID\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `protocol.go\|DecodeRequestFrame\|version != ProtocolVersion\|0\|X` | killed | TestEnvelopeEchoEmptyStringRefuses |
| `protocol.go\|DecodeRequestFrame\|version != ProtocolVersion\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `protocol.go\|CheckSuccessEnvelope\|protocol != ProtocolID\|0\|X` | killed | TestEnvelopeEchoEmptyStringRefuses |
| `protocol.go\|CheckSuccessEnvelope\|protocol != ProtocolID\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `protocol.go\|CheckSuccessEnvelope\|version != ProtocolVersion\|0\|X` | killed | TestEnvelopeEchoEmptyStringRefuses |
| `protocol.go\|CheckSuccessEnvelope\|version != ProtocolVersion\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `protocol.go\|CheckSuccessEnvelope\|requestID != want.RequestID\|0\|X` | killed | TestEnvelopeEchoEmptyStringRefuses |
| `protocol.go\|CheckSuccessEnvelope\|requestID != want.RequestID\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `protocol.go\|CheckSuccessEnvelope\|operation != string(want.Operation)\|0\|X` | killed | TestEnvelopeEchoEmptyStringRefuses |
| `protocol.go\|CheckSuccessEnvelope\|operation != string(want.Operation)\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `protocol.go\|CheckFailureEnvelope\|protocol != ProtocolID\|0\|X` | killed | TestEnvelopeEchoEmptyStringRefuses |
| `protocol.go\|CheckFailureEnvelope\|protocol != ProtocolID\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `protocol.go\|CheckFailureEnvelope\|version != ProtocolVersion\|0\|X` | killed | TestEnvelopeEchoEmptyStringRefuses |
| `protocol.go\|CheckFailureEnvelope\|version != ProtocolVersion\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `protocol.go\|CheckFailureEnvelope\|requestID != want.RequestID\|0\|X` | killed | TestEnvelopeEchoEmptyStringRefuses |
| `protocol.go\|CheckFailureEnvelope\|requestID != want.RequestID\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `protocol.go\|CheckFailureEnvelope\|operation != string(want.Operation)\|0\|X` | killed | TestEnvelopeEchoEmptyStringRefuses |
| `protocol.go\|CheckFailureEnvelope\|operation != string(want.Operation)\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `tuple.go\|checkContractsShape\|previousVersion != ""\|0\|X` | compile-fail | — |
| `tuple.go\|checkContractsShape\|previous != ""\|0\|X` | compile-fail | — |
| `tuple.go\|CheckTupleAdmission\|entry.Key.Direction != direction\|0\|X` | killed | TestTupleAdmissionZeroFactsRefuse |
| `tuple.go\|CheckTupleAdmission\|entry.Key.Direction != direction\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `tuple.go\|CheckTupleAdmission\|entry.Key.Environment != environment\|0\|X` | compile-fail | — |
| `tuple.go\|CheckTupleAdmission\|entry.Key.Environment != environment\|0\|Y` | compile-fail | — |
| `tuple.go\|CheckTupleAdmission\|entry.Key.ProviderID != binding.ProviderID\|0\|X` | killed | TestTupleAdmissionZeroFactsRefuse |
| `tuple.go\|CheckTupleAdmission\|entry.Key.ProviderID != binding.ProviderID\|0\|Y` | killed | TestTupleAdmissionZeroFactsRefuse |
| `tuple.go\|CheckTupleAdmission\|entry.Key.CandidateKind != binding.CandidateKind\|0\|X` | killed | TestTupleAdmissionZeroFactsRefuse |
| `tuple.go\|CheckTupleAdmission\|entry.Key.CandidateKind != binding.CandidateKind\|0\|Y` | killed | TestTupleAdmissionZeroFactsRefuse |
| `tuple.go\|CheckTupleAdmission\|entry.Key.ExecutableSHA256 != binding.ExecutableSHA256\|0\|X` | killed | TestTupleAdmissionZeroFactsRefuse |
| `tuple.go\|CheckTupleAdmission\|entry.Key.ExecutableSHA256 != binding.ExecutableSHA256\|0\|Y` | killed | TestTupleAdmissionZeroFactsRefuse |
| `tuple.go\|CheckTupleAdmission\|entry.Key.ProviderManifestDigest != binding.ProviderManifestDigest\|0\|X` | killed | TestTupleAdmissionZeroFactsRefuse |
| `tuple.go\|CheckTupleAdmission\|entry.Key.ProviderManifestDigest != binding.ProviderManifestDigest\|0\|Y` | killed | TestTupleAdmissionZeroFactsRefuse |
| `tuple.go\|CheckTupleAdmission\|entry.Key.AdapterManifestDigest != binding.AdapterManifestDigest\|0\|X` | killed | TestTupleAdmissionZeroFactsRefuse |
| `tuple.go\|CheckTupleAdmission\|entry.Key.AdapterManifestDigest != binding.AdapterManifestDigest\|0\|Y` | killed | TestTupleAdmissionZeroFactsRefuse |
| `tuple.go\|CheckTupleAdmission\|entry.Status != "accepted"\|0\|X` | killed | TestTupleAdmissionZeroFactsRefuse |
| `tuple.go\|CheckTupleAdmission\|entry.Status != "accepted"\|0\|Y` | census-only | TestIdentityGatesAreCensused |
| `tuple.go\|CheckTupleAdmission\|entry.Strategies[0] != "archive_only"\|0\|X` | killed | TestTupleAdmissionZeroFactsRefuse |
| `tuple.go\|CheckTupleAdmission\|entry.Strategies[0] != "archive_only"\|0\|Y` | census-only | TestIdentityGatesAreCensused |

## == -spelled refusal gates (outside both censuses) — 13 mutants

| mutant | status | behavioural killers |
| --- | --- | --- |
| `EQ-disc-exepath` | killed | TestDiscoverRefusals, TestEveryArmWitnessRefusesAtTheProductionEntry |
| `EQ-disc-owner` | killed | TestDiscoverRefusals, TestEveryArmWitnessRefusesAtTheProductionEntry |
| `EQ-freshsink-mode` | killed | TestDecodeObjectAuthorityRules, TestEveryArmWitnessRefusesAtTheProductionEntry |
| `EQ-freshsink-objects` | killed | TestDecodeObjectAuthorityRules, TestEveryArmWitnessRefusesAtTheProductionEntry |
| `EQ-freshsink-bytes` | killed | TestDecodeObjectAuthorityRules |
| `EQ-partial-cursor` | killed | TestDiscoverPartialCursorComplement |
| `EQ-validate-errfinding` | killed | TestEveryArmWitnessRefusesAtTheProductionEntry, TestValidateResultRules |
| `EQ-capusable-status` | census-only | TestIdentityGatesAreCensused |
| `EQ-capmapusable-status` | census-only | TestIdentityGatesAreCensused |
| `EQ-tuple-accepted` | killed | TestArrayBoundEdges, TestCheckTupleAdmissionRefusals, TestDecodeTupleEntryAcceptsBothDirections, TestDecodeTupleEntryCoherence, TestEveryArmWitnessRefusesAtTheProductionEntry, TestStringBoundEdges, TestTupleAdmissionValidityBoundaries, TestTupleAdmissionZeroFactsRefuse |
| `EQ-tuple-direction` | killed | TestCheckTupleAdmissionRefusals, TestEveryArmWitnessRefusesAtTheProductionEntry, TestTupleAdmissionZeroFactsRefuse |
| `EQ-tuple-archiveonly` | killed | TestCheckTupleAdmissionRefusals, TestEveryArmWitnessRefusesAtTheProductionEntry |
| `EQ-validcapname` | census-only | TestNoUnregisteredInlineVocabularies |

## typed re-runs of the rows above that needed real types — 5 mutants

| mutant | status | behavioural killers |
| --- | --- | --- |
| `EQ2-capusable` | survived | — |
| `EQ2-capmapusable` | survived | — |
| `EQ2-validcapname` | killed | TestCheckDoctorHealthyRefusesEmptyRequiredName |
| `ID2-count-selector` | killed | TestDecodeSourceSelectorComplement, TestEveryArmWitnessRefusesAtTheProductionEntry |
| `ID2-occurrences` | killed | TestEveryArmWitnessRefusesAtTheProductionEntry, TestResumePlanIdentityComplement |

## Totals

| status | count |
| --- | ---: |
| killed | 198 |
| survived | 13 |
| census-only | 39 |
| compile-fail | 12 |
| **applied** | **262** |
