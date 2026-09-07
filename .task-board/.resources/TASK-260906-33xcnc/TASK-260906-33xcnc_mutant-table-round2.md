## Mutant battery (round 2)

Denominator (production-derived at runtime): 31 refuse sites + 59 boolean-false exits + 11 frame-fault exits = 101.
Counting the 4 bool-`return <expr>` arms (CheckEnvironmentID, CheckSemver, isNull, rawUint53 forward) the exit inventory is 105; see the outcome for the kind-by-kind account.

Killed over applied: **70/71**. Classes: narrowing (gate stays, admits exactly one
rejected member), arm-deletion, census-only (behavioral green, census red), audit-only
(behavioral green, audit red). NOT_APPLIED and COMPILE_FAIL are distinct rows; the
harness-validation mutants prove both rows exist.

| Mutant | Narrows the gate to | Named failing test | Status | Killed by |
| --- | --- | --- | --- | --- |
| `N1_lowsurrogate` (narrowing) | admits lone low escapes only; highs still refused | `TestFrameAgreementAcrossFacades/bare_low_escape` | KILLED | behavioral |
| `N2_bytelength` (narrowing) | StringLength counts bytes; 128 wide chars refused | `TestStringMeasureCountsRunes/environ_helper` | KILLED | behavioral,census |
| `N3_tuplextensions` (narrowing) | tuple admits exactly the extensions member | `TestTupleAgreementAcrossFacades/extensions_refused` | KILLED | behavioral |
| `N4_caplength` (narrowing) | capabilities admit exactly nine keys | `TestDecodeEnvironmentObservation/ninth_capability_refused` | KILLED | behavioral |
| `N6_envunderscore` (narrowing) | env-id admits underscore (literal drift caught by one-language) | `TestSharedGrammarsAreOneLanguage` | KILLED | census |
| `T1_rawscan` (narrowing) | raw entry scan: quoted text misread as escape (both directions) | `TestFrameAgreementAcrossFacades/escaped_backslash_high_run2` | KILLED | behavioral |
| `M1_uint53floor` (narrowing) | admits below-minimum values; ceiling still holds | `TestCheckUint53BoundsEdges` | KILLED | behavioral |
| `M2_uint53magnitude` (narrowing) | admits exactly 2^53; 2^53+1 still refused | `TestCheckUint53BoundsEdges` | KILLED | behavioral |
| `M4_uint53trailing` (arm-deletion) | trailing-data arm removed; `12a` reads as 12 | `TestCheckUint53BoundsEdges` | KILLED | behavioral |
| `M5_stringsdup` (narrowing) | admits duplicates; unsorted still refused | `TestCheckSortedUniqueStringsHalves/sorted_duplicated_refuses` | KILLED | behavioral |
| `M6_stringsorder` (narrowing) | admits unsorted-but-unique; duplicates still refused | `TestCheckSortedUniqueStringsHalves/unsorted_unique_refuses` | KILLED | behavioral |
| `M7_digestbridge` (arm-deletion) | digest bridge removed; non-digests admitted | `TestTupleAgreementAcrossFacades/bad_fingerprint` | KILLED | behavioral |
| `M8_timestampbridge` (arm-deletion) | timestamp bridge removed; non-timestamps admitted | `TestDecodeEnvironmentObservation/bad_timestamp` | KILLED | behavioral |
| `M9_tuplearch` (narrowing) | admits exactly x86 alongside the closed pair | `TestTupleAgreementAcrossFacades/x86_refused` | KILLED | behavioral |
| `M10_tuplesemver` (narrowing) | admits short versions; also drifts the grammar literal (census must fire too) | `TestTupleAgreementAcrossFacades/short_version` | KILLED | behavioral,census |
| `M11_reasonavailable` (arm-deletion) | available-with-reason arm removed | `TestDecodeEnvironmentObservation/available_with_reason_refused` | KILLED | behavioral |
| `M12_reasonconditional` (arm-deletion) | conditional-without-reason arm removed | `TestDecodeEnvironmentObservation/conditional_without_reason_refused` | KILLED | behavioral |
| `M13_capabsent` (arm-deletion) | capability presence arm removed; swapped key admitted | `TestDecodeEnvironmentObservation/unknown_capability_key` | KILLED | behavioral |
| `M14a_evidencedup` (narrowing) | evidence admits duplicates; unsorted still refused | `TestDecodeEnvironmentObservation/duplicated_evidence_refused` | KILLED | behavioral |
| `M14b_evidenceorder` (narrowing) | evidence admits unsorted; duplicates still refused | `TestDecodeEnvironmentObservation/unsorted_evidence_refused` | KILLED | behavioral |
| `M15_framedup` (arm-deletion) | duplicate-member arm removed | `TestFrameAgreementAcrossFacades/duplicate_member` | KILLED | behavioral,census |
| `M16_frametrailing` (arm-deletion) | trailing-data arm removed | `TestFrameAgreementAcrossFacades/trailing_data` | KILLED | behavioral |
| `M17_frameutf8` (arm-deletion) | UTF-8 arm removed | `TestFrameAgreementAcrossFacades/non-utf8_bytes` | KILLED | behavioral |
| `M18_highmispair` (narrowing) | admits high-followed-by-non-low; lone lows still refused | `TestFrameAgreementAcrossFacades/high_followed_by_non-low` | KILLED | behavioral |
| `M19_extensions` (narrowing) | admits exactly the nodots key; other keys still refused | `TestDecodeEnvironmentObservation/bad_extensions_key` | KILLED | behavioral |
| `M20_tupleunknown` (arm-deletion) | unknown-member arm removed | `TestTupleAgreementAcrossFacades/unknown_member` | KILLED | behavioral |
| `M21_obsschema` (narrowing) | admits exactly the session-adapter-manifest schema | `TestDecodeEnvironmentObservation/wrong_schema` | KILLED | behavioral |
| `R_missingfirst` (narrowing) | missingMember skips the first required name; missing environment_id admitted | `TestTupleAgreementAcrossFacades/missing_environment_id` | KILLED | behavioral |
| `R_laddernondigit` (narrowing) | ladder admits exactly the letter a; other non-digits still refused | `TestParseUint53LiteralRefusesNonDigits` | KILLED | behavioral |
| `R_strfloor` (narrowing) | CheckStringBounds drops the floor; empty strings admitted | `TestDecodeEnvironmentObservation/empty_version` | KILLED | behavioral |
| `R_rawstringerr` (arm-deletion) | rawString error arm removed; the direct-rawString arms (platform/arch/version/provider) admit non-strings as empty and refuse at the wrong arm | `TestEveryRefusalSiteIsExercised/tuple_platform_string` | KILLED | behavioral |
| `R_decodearraytrailing` (arm-deletion) | decodeArray trailing arm removed | `TestDecodeArrayRefusesTrailingData` | KILLED | behavioral |
| `R_digestnonstr` (arm-deletion) | CheckDigest type arm removed; EXPECTED SURVIVOR: ParseDigest refuses the zero value downstream | - | SURVIVED | - |
| `S1_sesstrailingdel` (arm-deletion) | sessadapter trailing arm removed; `12a` reads as 12 | `TestRawUint53RefusesTrailingData` | KILLED | behavioral |
| `S2_sesstrailingnarrow` (narrowing) | refuses only error trailing; `12 13` admitted as 12 | `TestRawUint53RefusesTrailingData` | KILLED | behavioral |
| `S3_dirtrailingdel` (arm-deletion) | dirnode trailing arm removed; `12a` reads as 12 | `TestRawUint53RefusesTrailingData` | KILLED | behavioral |
| `S4_dirtrailingnarrow` (narrowing) | refuses only error trailing; `12 13` admitted as 12 | `TestRawUint53RefusesTrailingData` | KILLED | behavioral |
| `S5_provtrailingdel` (arm-deletion) | provhost trailing arm neutered (`&& false`: deleting the block would orphan the file's only io use and fail closed at compile time); `12a` reads as 12 | `TestRawUint53RefusesTrailingData` | KILLED | behavioral |
| `S6_provtrailingnarrow` (narrowing) | refuses only error trailing; `12 13` admitted as 12 | `TestRawUint53RefusesTrailingData` | KILLED | behavioral |
| `S7_sessboundforward` (narrowing) | delegation forwards maximum+1; 129-char values admitted through sessadapter entries | `TestStringBoundEdges` | KILLED | behavioral |
| `S8_dirboundforward` (narrowing) | delegation forwards maximum+1; overlong values admitted through dirnode entries | `TestStringBoundEdges` | KILLED | behavioral |
| `S9_sesssemversever` (narrowing) | tuple semver check admits exactly 1.2 (seam contract runs in the environ package) | `TestTupleAgreementAcrossFacades/short_version` | KILLED | behavioral,census |
| `S10_dirextsever` (arm-deletion) | dirnode extensions delegation severed to admit-all | `TestExtensionKeyBoundEdges` | KILLED | behavioral |
| `C1_scalarlows` (narrowing) | scalar admits lone lows; highs still refused | `TestScalarSurrogateAgreementAcrossUnits/bare_low_refused` | KILLED | behavioral |
| `C2_scalarmispair` (narrowing) | scalar admits high-followed-by-non-low | `TestScalarSurrogateAgreementAcrossUnits/high_followed_by_non-low_refused` | KILLED | behavioral |
| `C3_scalarevenrun` (narrowing) | token-preserving: pair-skip dropped, quoted text misreads as escape and the composed lone low is admitted (the Story hole reborn) | `TestScalarSurrogateAgreementAcrossUnits/escaped_backslash_plus_real_lone_low_refused` | KILLED | behavioral |
| `G1_revivedgrammar` (census-only) | revived semver copy with the identical literal; no entry changes verdict | `TestCheckHelpersDelegateToEnviron` | KILLED | census |
| `G2_freshmeasure` (census-only) | fresh-name rune counter; no entry changes verdict | `TestSharedShapesAreLedgered` | KILLED | census |
| `G3_freshdecoder` (census-only) | fresh-name decoder with duplicate texture; no entry changes verdict | `TestSharedShapesAreLedgered` | KILLED | census |
| `A1_refusealias` (audit-only) | rebound refuse var; every entry keeps its verdict, the use site is unattributed | `constructor "refuse" referenced outside direct-call position` | KILLED | audit |
| `M7n_digestnarrow` (narrowing) | digest bridge admits exactly sha256:zzzz; other non-digests still refused | `TestEveryRefusalSiteIsExercised/tuple_fingerprint` | KILLED | behavioral |
| `M8n_timestampnarrow` (narrowing) | timestamp bridge admits exactly yesterday; other non-timestamps still refused | `TestEveryRefusalSiteIsExercised/observation_timestamp` | KILLED | behavioral |
| `M13n_capabsentnarrow` (narrowing) | presence admits exactly a missing directory_discovery (skipping its decode, which would refuse the nil member as fall-through); other absences still refused | `TestDecodeEnvironmentObservation/unknown_capability_key` | KILLED | behavioral |
| `M15n_framedupnarrow` (narrowing) | duplicate arm admits exactly a duplicated v; other duplicates still refused | `TestFrameAgreementAcrossFacades/duplicate_member` | KILLED | behavioral |
| `M17n_frameutf8narrow` (narrowing) | UTF-8 arm admits exactly bodies containing 0xff; other non-UTF-8 still refused | `TestFrameAgreementAcrossFacades/non-utf8_bytes` | KILLED | behavioral |
| `D1_decoycheckdigest` (narrowing) | token-preserving: admits exactly uppercase-hex digests; lowercase still refused | `TestCheckHelpersDelegateToEnviron/sessadapter` | KILLED | behavioral,census |
| `D2_decoywrapperfault` (arm-deletion) | token-preserving: fault check removed; malformed frames slide to later arms | `TestDelegatingWrappersCallEnviron/sessadapter` | KILLED | behavioral,census |
| `E_dupnarrow_body` (narrowing) | duplicate arm admits exactly a duplicated body; every other key still refused | `TestFrameAgreementRefusesDuplicateOfEveryDerivedMember/body` | KILLED | behavioral |
| `E_dupnarrow_ok` (narrowing) | duplicate arm admits exactly a duplicated ok; every other key still refused | `TestFrameAgreementRefusesDuplicateOfEveryDerivedMember/ok` | KILLED | behavioral |
| `E_dupnarrow_protocol_version` (narrowing) | duplicate arm admits exactly a duplicated protocol_version; every other key still refused | `TestFrameAgreementRefusesDuplicateOfEveryDerivedMember/protocol_version` | KILLED | behavioral |
| `E_dupnarrow_request_id` (narrowing) | duplicate arm admits exactly a duplicated request_id; every other key still refused | `TestFrameAgreementRefusesDuplicateOfEveryDerivedMember/request_id` | KILLED | behavioral |
| `E_dupnarrow_error` (narrowing) | duplicate arm admits exactly a duplicated error; every other key still refused | `TestFrameAgreementRefusesDuplicateOfEveryDerivedMember/error` | KILLED | behavioral |
| `E_dupnarrow_capabilities` (narrowing) | duplicate arm admits exactly a duplicated capabilities; every other key still refused | `TestFrameAgreementRefusesDuplicateOfEveryDerivedMember/capabilities` | KILLED | behavioral |
| `E_dupnarrow_provider_id` (narrowing) | duplicate arm admits exactly a duplicated provider_id; every other key still refused | `TestFrameAgreementRefusesDuplicateOfEveryDerivedMember/provider_id` | KILLED | behavioral |
| `E_dupnarrow_cursor` (narrowing) | duplicate arm admits exactly a duplicated cursor; every other key still refused | `TestFrameAgreementRefusesDuplicateOfEveryDerivedMember/cursor` | KILLED | behavioral |
| `P_dupnarrow_body` (narrowing) | provhost duplicate arm admits exactly a duplicated body; every other key still refused | `TestDecodeResponseRefusesDuplicateOfEveryMember/body` | KILLED | behavioral |
| `P_dupnarrow_ok` (narrowing) | provhost duplicate arm admits exactly a duplicated ok; every other key still refused | `TestDecodeResponseRefusesDuplicateOfEveryMember/ok` | KILLED | behavioral |
| `P_dupnarrow_protocol_version` (narrowing) | provhost duplicate arm admits exactly a duplicated protocol_version; every other key still refused | `TestDecodeResponseRefusesDuplicateOfEveryMember/protocol_version` | KILLED | behavioral |
| `P_dupnarrow_error` (narrowing) | provhost duplicate arm admits exactly a duplicated error; every other key still refused | `TestDecodeResponseRefusesDuplicateOfEveryMember/error` | KILLED | behavioral |
| `P_dupnarrow_capabilities` (narrowing) | provhost duplicate arm admits exactly a duplicated capabilities; every other key still refused | `TestDecodeResponseRefusesDuplicateOfEveryMember/capabilities` | KILLED | behavioral |
| `P_dupnarrow_provider_id` (narrowing) | provhost duplicate arm admits exactly a duplicated provider_id; every other key still refused | `TestDecodeResponseRefusesDuplicateOfEveryMember/provider_id` | KILLED | behavioral |

| Harness mutant | Purpose | Status |
| --- | --- | --- |
| `H1_ambiguous` | ambiguous pattern must NOT_APPLY, never half-apply | NOT_APPLIED |
| `H2_arity` | arity break must COMPILE_FAIL, never pass as survivor | COMPILE_FAIL |

### Survivor bounds

`R_digestnonstr` (expected survivor, arm-deletion): removing CheckDigest's type arm changes no entry
verdict because scalar.ParseDigest refuses the empty string downstream for every non-string
input. The property is now pinned by `TestDigestBridgeRefusesEmptyDownstream` (round 2 closes
the N2 rationale gap): if `ParseDigest("")` ever admitted, that test reddens first. The bridge
itself is narrowed by `M7`/`M7n` and killed. No unpredicted survivor, no KILLED_OTHER, no billed
NOT_APPLIED and no COMPILE_FAIL.

### Honest survivor in-session

The first `P_dupnarrow_provider_id` run SURVIVED with a green suite and exposed a real test gap:
the new envelope sweeper did not include the foreign `provider_id` key, so no subtest existed to
fail. The key was added to `TestDecodeResponseRefusesDuplicateOfEveryMember`, the mutant was
re-run, and it is KILLED above. A survivor with no named failing test is reported as a survivor;
this one is reported as found-fixed-and-killed, not as first-time green.
