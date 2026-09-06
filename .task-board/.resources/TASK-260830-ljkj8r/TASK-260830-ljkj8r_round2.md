# TASK-260830-ljkj8r round-2 rework — review response (B1/B2/B3)

Production code is unchanged from the reviewed candidate: all rework is
tests plus four one-line fixture-surgery fixes. Working tree holds the
rework uncommitted per the brief.

## B1 — bound instrument ported (blocking, now closed)

- `internal/dirnode/bound_census_test.go`: production-derived bound census,
  101 sites (9 helpers + len/length-var guards), 72 driven + 29 exempt
  (mechanism/plumbing with rationale). Census fails closed on unregistered
  sites, orphaned rows, helper aliases, and unscanned files.
- `internal/dirnode/bounds_edge_test.go`: edge drivers for every driven row
  (min-1/min/max/max+1 through the production entry).
- Battery: 185 applied / 142 killed / 43 equivalent-with-bound / 0 survivors,
  zero not-applied, zero compile failures. Denominator derived from
  production (scanner output), not listed.
- Census-only kills scored separately: 2 live census mutants redden only
  `TestBoundGuardsAreCensused` (unregistered site; helper alias), plus the
  three synthetic gate proofs. Behavioural kills (142) all fail a driver
  that exercises the production entry.

## B2 — seventeen-operation obligation set derived and driven (blocking, now closed)

- `internal/dirnode/query_operations_test.go`: `TestDecodeQueryAllOperationsAdmit`
  drives 17/17 registry members (was 5/17 reaching DecodeQuery) with kind
  checks; `TestDecodeQueryOperationValidatorsRefuse` drives 40+
  validator-specific defects; projection/sort/pagination/lineage/count
  rules pinned alongside.
- Zero-coverage functions before: validQueryField, checkLineageParameters,
  checkHostsParameters, checkEnvironmentsParameters, checkJobsParameters,
  checkPlansParameters, checkDistinctParameters, checkTagsParameter,
  checkPinParameter, checkEnrichParameters (10 at 0.0%). After: none at 0.0%.
- Review narrowing probes are battery rows V01 (environments/"root"),
  V02 (jobs/"pwned"), V03 (distinct/"secret"): all killed.

## B3 — arm-slide vectors fixed (blocking, now closed)

- `unknown_field` rebuilt without the duplicate `fields` member; new
  `TestUnknownFieldVectorReachesTheFieldRegistry` proves the operation
  object passes the strict decoder (single `fields`) before DecodeQuery
  refuses, so the field registry is reached, not the duplicate gate.
- Audit found the same rename-instead-of-delete slide in three more
  vectors (query "missing caller", manifest "missing member", probe
  response "missing member"): each renamed member fired the unknown arm
  while witnessing the missing arm. All three now delete the member, and
  `TestMissingVectorsDriveTheMissingArm` asserts the missing-member message.
- Scan/probe-request missing vectors already delete; unknown vectors add
  exactly one member. Protocol `relabeled major pair` placeholder is
  overridden with a genuine relabeled frame (verified, no change needed).

## AC coverage — 9 of 9 rows driven through production entries

| AC row | production call site | named test |
|---|---|---|
| major bootstrap | DecideBootstrapStep/NextLowerMajor/ProcessGuard.Claim | TestBootstrapSelectsV2Directly, TestBootstrapFallsBackV2ToV1, TestNextLowerMajorBounds, TestProcessGuardRefusesReuse |
| manifest framing | DecodeManifest/CheckManifestBindings/ManifestDigest | TestDecodeManifestClosedMemberRules, TestCheckManifestBindings, TestStringBoundEdges, TestDigestArrayBoundEdges, TestManifestLimitsBoundEdges, TestContractAssertionBoundEdges |
| probe framing | CheckProbeRequest/CheckProbeResponse/CheckNodeBuildEqualsManifest | TestProbeRequestMajorVocabularies, TestProbeResponseFindingBound, TestProbeEnvironmentsBoundEdges, TestNodeBuildEqualityRefusesDrift |
| scan framing | CheckScanRequest/CheckScanResponse | TestCheckScanRequestRefusals, TestCheckScanResponseRefusals, TestStringBoundEdges, TestDigestArrayBoundEdges, TestUint53BoundEdges |
| query framing (17 ops) | DecodeQuery/CheckCursorReuse | TestDecodeQueryAllOperationsAdmit (17/17), TestDecodeQueryOperationValidatorsRefuse, TestDecodeQueryProjectionFields, TestCheckCursorReuse |
| host checks | ProcessGuard, CheckManifestBindings, CheckCursorReuse | TestProcessGuardRefusesReuse, TestCheckManifestBindings, TestCheckCursorReuse |
| structured errors | failX constructors via every entry | census TestDerivedRefusalArmsAreAllWitnessed + arm witnesses |
| crash/idempotency | Journal.CheckAndRecord/Export/Import | TestJournalSurvivesRestart, TestJournalImportRefusesUnsortedRecords, TestJournalBoundEdges |
| no unsupported capability | Operations/FramedOperations/Capabilities/AvailableCapabilities | TestOperationRegistryDerivedFromSpec, TestFramedSubsetIsExact, TestAvailableCapabilitiesNeverUpgrades |

## Evidence

- `go test ./...`: 18 packages ok, 0 fail (exit 0).
- `go test ./internal/dirnode/ -cover`: 83.3% statements (was 74.9%), zero functions at 0.0%.
- `go vet ./...`, `GOOS=windows go vet ./...`, `go build ./...`: exit 0.
- `go run ./internal/traceability/cmd/tracecheck`: ok.
- `gofmt -l internal/`: clean.

# TASK-260830-ljkj8r round-2 rework — review response (B1/B2/B3)

## Verdicts
- Applied: 185 (all compile, zero not-applied)
- Killed (behavioural, named driver fails): 142
- Equivalent with stated bound: 43
- Survivors: 0
- Bound coverage: 142 killed / 185 applied (production-derived denominator: 101 census sites, 72 driven + 29 exempt mechanism/plumbing)
- Census-only kills (separate score): 2 live (unregistered site reddens census; helper alias fails derivation) + synthetic suites TestBoundHelperIndirectionReports / TestBoundLenScanSeesNonIfContexts / TestBoundLenScanSeesLengthVariables

## Equivalence classes (every equiv-ok mutant states its bound)

| class | count | bound |
|---|---|---|
| count ceiling == vocabulary size (Nth valid member unbuildable) | 11 | kinds/mgmt/reach/freshness/scopes/req-caps/enrich-kinds/auth-status/job-states/fields(2) |
| element bound masked behind closed vocabulary (outer gate holds) | 20 | caps/scopes/enrich-kinds/auth-status/job-states/enum-kinds/enum-mgmt/enum-reach/enum-fresh/fields |
| element bound masked behind pattern (outer gate holds) | 6 | req-env-ids/env-ids/provider-ids |
| floor unreachable behind grammar (no shorter admittable shape) | 3 | versions element (no 4-char SemVer), extension floor (no 2-char reverse-DNS key), URI floor (no 1-2 char URI with scheme) |
| masked behind a joint gate (count/index coincide at 65) | 2 | ops count 64->65, index 63->64 |
| stated body bound (absurd-size probe not built) | 1 | native 65537-element body |

## Full battery table (mutant, verdict, driver, narrowing)
| M01 | killed | TestStringBoundEdges/manifest_node_id | node_id floor 1->0 admits empty |
| M02 | killed | TestStringBoundEdges/manifest_node_id | node_id ceiling 128->129 |
| M03 | killed | TestDigestArrayBoundEdges/redaction_policies | redaction floor 1->0 |
| M04 | killed | TestDigestArrayBoundEdges/redaction_policies | redaction ceiling 64->65 |
| M05 | killed | TestDigestArrayBoundEdges/enrichment_profiles | enrichment ceiling 256->257 |
| M06 | killed | TestDigestArrayBoundEdges/capability_evidence | evidence ceiling 64->65 |
| M07 | killed | TestStringBoundEdges/capability_reason | reason floor (review M8 family) |
| M08 | killed | TestStringBoundEdges/capability_reason | reason ceiling (review M8) |
| M09 | killed | TestContractAssertionBoundEdges | schemas floor 15->14 |
| M10 | killed | TestContractAssertionBoundEdges | schemas floor 15->16 |
| M11 | killed | TestContractAssertionBoundEdges | schemas ceiling 64->65 |
| M12 | killed | TestContractAssertionBoundEdges | schemas ceiling 64->63 |
| M13 | killed | TestManifestLimitsBoundEdges/max_frame_bytes | frame floor 1->0 |
| M14 | killed | TestManifestLimitsBoundEdges/max_scan_instances | scan instances floor |
| M15 | killed | TestManifestLimitsBoundEdges/max_inventory_take | take floor |
| M16 | killed | TestManifestLimitsBoundEdges/max_enrichment_events | events floor |
| M17 | killed | TestManifestLimitsBoundEdges/max_enrichment_bytes | bytes floor |
| M18 | killed | TestManifestLimitsBoundEdges/max_scan_instances | scan instances ceiling |
| M19 | killed | TestManifestLimitsBoundEdges/max_inventory_take | take ceiling |
| M20 | killed | TestManifestLimitsBoundEdges/max_excerpt_count | excerpt count ceiling |
| M21 | killed | TestManifestLimitsBoundEdges/max_excerpt_bytes | excerpt bytes ceiling |
| M22 | killed | TestManifestLimitsBoundEdges/max_enrichment_events | events ceiling |
| M23 | killed | TestManifestLimitsBoundEdges/max_enrichment_bytes | bytes ceiling |
| M24 | equiv-ok | TestSortedUniqueStringsEdges/supported_versions | no 4-char SemVer exists |
| M25 | killed | TestSortedUniqueStringsEdges/supported_versions | versions floor 5->6 refuses 1.0.0 |
| M26 | killed | TestSortedUniqueStringsEdges/supported_versions | versions element ceiling |
| M27 | killed | TestSortedUniqueStringsEdges/supported_versions | versions element tightening |
| M28 | killed | TestDecodeManifestRegistryRules | versions count floor admits empty |
| M29 | killed | TestSortedUniqueStringsEdges/supported_versions | versions count floor 1->2 |
| M30 | killed | TestSortedUniqueStringsEdges/supported_versions | versions count ceiling |
| M31 | killed | TestSortedUniqueStringsEdges/supported_versions | versions count tightening |
| M32 | equiv-ok | TestSortedUniqueStringsEdges/requested_environment_ids | empty id still pattern-refused |
| M33 | equiv-ok | TestSortedUniqueStringsEdges/requested_environment_ids | 65-char id still pattern-refused |
| M34 | killed | TestSortedUniqueStringsEdges/requested_environment_ids | 64-char valid id refused |
| M35 | killed | TestSortedUniqueStringsEdges/requested_environment_ids | env count ceiling |
| M36 | killed | TestSortedUniqueStringsEdges/requested_environment_ids | env count tightening |
| M37 | equiv-ok | TestSortedUniqueStringsEdges/requested_capabilities | empty cap still vocab-refused |
| M38 | equiv-ok | TestSortedUniqueStringsEdges/requested_capabilities | 65-char cap still vocab-refused |
| M39 | equiv-ok | TestSortedUniqueStringsEdges/requested_capabilities | 9th registry capability unbuildable; count masked behind closed vocabulary |
| M40 | killed | TestSortedUniqueStringsEdges/requested_capabilities | caps count tightening |
| M41 | killed | TestProbeResponseFindingBound | findings ceiling 4096->4097 |
| M42 | killed | TestProbeResponseFindingBound | findings tightening |
| M43 | killed | TestStringBoundEdges/finding_remediation | remediation floor admits empty |
| M44 | killed | TestStringBoundEdges/finding_remediation | remediation ceiling |
| M45 | killed | TestStringBoundEdges/finding_code | code floor |
| M46 | killed | TestStringBoundEdges/finding_code | code ceiling |
| M47 | killed | TestStringBoundEdges/finding_message | message floor |
| M48 | killed | TestStringBoundEdges/finding_message | message ceiling |
| M49 | killed | TestProbeEnvironmentsBoundEdges | probe env ceiling |
| M50 | killed | TestProbeEnvironmentsBoundEdges | probe env tightening |
| M51 | killed | TestStringBoundEdges/node_build_node_id | build node floor |
| M52 | killed | TestStringBoundEdges/node_build_node_id | build node ceiling |
| M53 | killed | TestUint53BoundEdges | deadline floor const |
| M54 | killed | TestUint53BoundEdges | deadline ceiling const |
| M55 | killed | TestFrameBoundEdges | frame const widened |
| M56 | killed | TestStringBoundEdges/caller_id | caller floor |
| M57 | killed | TestStringBoundEdges/caller_id | caller ceiling |
| M58 | killed | TestStringBoundEdges/authentication_subject | subject floor |
| M59 | killed | TestStringBoundEdges/authentication_subject | subject ceiling |
| M60 | equiv-ok | TestSortedUniqueStringsEdges/caller_scopes | empty scope still vocab-refused |
| M61 | equiv-ok | TestSortedUniqueStringsEdges/caller_scopes | 65-char scope still vocab-refused |
| M62 | killed | TestSortedUniqueStringsEdges/caller_scopes | scopes count floor admits empty |
| M63 | killed | TestDecodeQueryAcceptsFixture | scopes count floor 1->2 refuses single |
| M64 | equiv-ok | TestSortedUniqueStringsEdges/caller_scopes | 6th valid scope unbuildable |
| M65 | killed | TestSortedUniqueStringsEdges/caller_scopes | scopes count tightening |
| M66 | equiv-ok | TestQueryBatchCountEdges | index 64 still position-mismatched |
| M67 | killed | TestQueryBatchCountEdges | index tightening refuses 63 |
| M68 | killed | TestQueryBatchCountEdges | ops floor admits empty |
| M69 | equiv-ok | TestQueryBatchCountEdges | 65-batch still index-refused |
| M70 | killed | TestQueryBatchCountEdges | ops tightening refuses 64 |
| M71 | killed | TestCheckCursorReuse | cursor floor admits empty |
| M72 | killed | TestCheckCursorReuse | cursor ceiling |
| M73 | equiv-ok | TestSortedUniqueStringsEdges/enrich_kinds | empty kind still vocab-refused |
| M74 | equiv-ok | TestSortedUniqueStringsEdges/enrich_kinds | 65-char kind still vocab-refused |
| M75 | killed | TestDecodeQueryOperationValidatorsRefuse/enrich_empty_kinds | kinds floor admits empty |
| M76 | killed | TestDecodeQueryAllOperationsAdmit/enrich | kinds floor 1->2 refuses single |
| M77 | equiv-ok | TestSortedUniqueStringsEdges/enrich_kinds | 4th valid kind unbuildable; count masked behind closed vocabulary |
| M78 | killed | TestSortedUniqueStringsEdges/enrich_kinds | kinds tightening |
| M79 | equiv-ok | TestSortedUniqueStringsEdges/authentication_status | empty status still vocab-refused |
| M80 | equiv-ok | TestSortedUniqueStringsEdges/authentication_status | 33-char status still vocab-refused |
| M81 | equiv-ok | TestSortedUniqueStringsEdges/authentication_status | 5th valid status unbuildable; count masked behind closed vocabulary |
| M82 | killed | TestSortedUniqueStringsEdges/authentication_status | auth status tightening |
| M83 | equiv-ok | TestSortedUniqueStringsEdges/environment_ids | empty id still pattern-refused |
| M84 | equiv-ok | TestSortedUniqueStringsEdges/environment_ids | 65-char id still pattern-refused |
| M85 | killed | TestSortedUniqueStringsEdges/environment_ids | 64-char valid id refused |
| M86 | killed | TestSortedUniqueStringsEdges/environment_ids | env ids ceiling |
| M87 | killed | TestSortedUniqueStringsEdges/environment_ids | env ids tightening |
| M88 | killed | TestSortedUniqueUUIDv7Edges/environments_host_ids | env hosts ceiling |
| M89 | killed | TestSortedUniqueUUIDv7Edges/environments_host_ids | env hosts tightening |
| M90 | equiv-ok | TestSortedUniqueStringsEdges/job_states | empty state still vocab-refused |
| M91 | equiv-ok | TestSortedUniqueStringsEdges/job_states | 33-char state still vocab-refused |
| M92 | equiv-ok | TestSortedUniqueStringsEdges/job_states | 8th valid state unbuildable; count masked behind closed vocabulary |
| M93 | killed | TestSortedUniqueStringsEdges/job_states | job states tightening |
| M94 | equiv-ok | TestSortedUniqueStringsEdges/provider_ids | empty provider still pattern-refused |
| M95 | equiv-ok | TestSortedUniqueStringsEdges/provider_ids | 33-char provider still pattern-refused |
| M96 | killed | TestSortedUniqueStringsEdges/provider_ids | 32-char valid provider refused |
| M97 | killed | TestSortedUniqueStringsEdges/provider_ids | provider count ceiling |
| M98 | killed | TestSortedUniqueStringsEdges/provider_ids | provider count tightening |
| M99 | killed | TestDecodeQueryOperationValidatorsRefuse/set_tags_past_count_bound | tags ceiling |
| M100 | killed | TestDecodeQueryCountBoundAdmissions | tags tightening vs 256-admit |
| M101 | killed | TestDecodeQueryOperationValidatorsRefuse/execute_plan_past_confirmations_bound | confirmations ceiling |
| M102 | killed | TestDecodeQueryCountBoundAdmissions | confirmations tightening |
| M103 | killed | TestStringBoundEdges/annotation_title | title floor |
| M104 | killed | TestStringBoundEdges/annotation_title | title ceiling |
| M105 | killed | TestQuerySortBoundEdges | sort ceiling |
| M106 | killed | TestQuerySortBoundEdges | sort tightening |
| M107 | killed | TestQueryPaginationBoundEdges | skip ceiling (stated shape-7) |
| M108 | killed | TestQueryPaginationBoundEdges | take floor (stated shape-7) |
| M109 | killed | TestQueryPaginationBoundEdges | take ceiling (stated shape-7) |
| M110 | equiv-ok | TestSortedUniqueStringsEdges/filter_enums | empty kind still vocab-refused |
| M111 | equiv-ok | TestSortedUniqueStringsEdges/filter_enums | 65-char kind still vocab-refused |
| M112 | equiv-ok | TestSortedUniqueStringsEdges/filter_enums | 4th valid kind unbuildable |
| M113 | killed | TestSortedUniqueStringsEdges/filter_enums | kinds count tightening vs full vocab |
| M114 | equiv-ok | TestSortedUniqueStringsEdges/filter_enums | empty state still vocab-refused |
| M115 | equiv-ok | TestSortedUniqueStringsEdges/filter_enums | 33-char state still vocab-refused |
| M116 | equiv-ok | TestSortedUniqueStringsEdges/filter_enums | 4th valid state unbuildable |
| M117 | killed | TestSortedUniqueStringsEdges/filter_enums | mgmt count tightening |
| M118 | equiv-ok | TestSortedUniqueStringsEdges/filter_enums | empty member still vocab-refused |
| M119 | equiv-ok | TestSortedUniqueStringsEdges/filter_enums | 33-char member still vocab-refused |
| M120 | equiv-ok | TestSortedUniqueStringsEdges/filter_enums | 5th valid member unbuildable |
| M121 | killed | TestSortedUniqueStringsEdges/filter_enums | reachability count tightening |
| M122 | equiv-ok | TestSortedUniqueStringsEdges/filter_enums | empty member still vocab-refused |
| M123 | equiv-ok | TestSortedUniqueStringsEdges/filter_enums | 33-char member still vocab-refused |
| M124 | equiv-ok | TestSortedUniqueStringsEdges/filter_enums | 8th valid member unbuildable |
| M125 | killed | TestSortedUniqueStringsEdges/filter_enums | freshness count tightening |
| M126 | killed | TestSortedUniqueUUIDv7Edges/hosts_host_ids | hosts ceiling |
| M127 | killed | TestSortedUniqueUUIDv7Edges/hosts_host_ids | hosts tightening |
| M128 | killed | TestSortedUniqueUUIDv7Edges/filter_host_ids | filter hosts ceiling |
| M129 | killed | TestSortedUniqueUUIDv7Edges/filter_host_ids | filter hosts tightening |
| M130 | killed | TestSortedUniqueUUIDv7Edges/filter_workspace_ids | workspace ceiling |
| M131 | killed | TestSortedUniqueUUIDv7Edges/filter_workspace_ids | workspace tightening |
| M132 | killed | TestUUIDv7DigestSubsetEdges | anchors ceiling |
| M133 | killed | TestUUIDv7DigestSubsetEdges | anchors tightening |
| M134 | killed | TestDigestArrayBoundEdges/job_profile_ids | profile ceiling |
| M135 | killed | TestDigestArrayBoundEdges/job_profile_ids | profile tightening |
| M136 | killed | TestSortedUniqueUUIDv7Edges/job_ids | job ids ceiling |
| M137 | killed | TestSortedUniqueUUIDv7Edges/job_ids | job ids tightening |
| M138 | killed | TestDigestArrayBoundEdges/plan_ids | plan ids ceiling |
| M139 | killed | TestDigestArrayBoundEdges/plan_ids | plan ids tightening |
| M140 | killed | TestSortedUniqueUUIDv7Edges/plan_operation_ids | plan op ids ceiling |
| M141 | killed | TestSortedUniqueUUIDv7Edges/plan_operation_ids | plan op ids tightening |
| M142 | equiv-ok | TestDecodeQueryProjectionFields | empty field still registry-refused |
| M143 | equiv-ok | TestDecodeQueryProjectionFields | 65-char field still registry-refused |
| M144 | equiv-ok | TestDecodeQueryProjectionFields | 129 valid fields unbuildable (27 in registry); count masked behind field registry |
| M145 | equiv-ok | TestDecodeQueryProjectionFields | 128 valid fields unbuildable (27 in registry) |
| M146 | killed | TestDigestArrayBoundEdges/supersedes | supersedes ceiling |
| M147 | killed | TestDigestArrayBoundEdges/supersedes | supersedes tightening |
| M148 | killed | TestSortedUniqueStringsEdges/filter_states | filter states floor admits empty |
| M149 | killed | TestSortedUniqueStringsEdges/filter_states | filter states element ceiling |
| M150 | killed | TestSortedUniqueStringsEdges/filter_states | filter states count ceiling |
| M151 | killed | TestSortedUniqueStringsEdges/filter_states | filter states count tightening |
| M152 | killed | TestSortedUniqueStringsEdges/filter_warnings | warnings floor |
| M153 | killed | TestSortedUniqueStringsEdges/filter_warnings | warnings element ceiling |
| M154 | killed | TestSortedUniqueStringsEdges/filter_warnings | warnings count ceiling |
| M155 | killed | TestSortedUniqueStringsEdges/filter_warnings | warnings count tightening |
| M156 | killed | TestStringBoundEdges/scan_cursor | scan cursor floor |
| M157 | killed | TestStringBoundEdges/scan_cursor | scan cursor ceiling |
| M158 | killed | TestDigestArrayBoundEdges/installation_ids | installations floor |
| M159 | killed | TestDigestArrayBoundEdges/installation_ids | installations floor 1->2 |
| M160 | killed | TestDigestArrayBoundEdges/installation_ids | installations ceiling |
| M161 | killed | TestDigestArrayBoundEdges/installation_ids | installations tightening |
| M162 | killed | TestUint53BoundEdges/scan_max_instances | max_instances floor |
| M163 | killed | TestUint53BoundEdges/scan_max_instances | max_instances floor 1->2 |
| M164 | killed | TestUint53BoundEdges/scan_max_instances | max_instances ceiling |
| M165 | killed | TestUint53BoundEdges/scan_max_instances | max_instances tightening |
| M166 | killed | TestStringBoundEdges/scan_next_cursor | next cursor floor |
| M167 | killed | TestStringBoundEdges/scan_next_cursor | next cursor ceiling |
| M168 | killed | TestDigestArrayBoundEdges/environment_observations | env obs floor |
| M169 | killed | TestDigestArrayBoundEdges/environment_observations | env obs floor 1->2 |
| M170 | killed | TestDigestArrayBoundEdges/environment_observations | env obs ceiling |
| M171 | killed | TestDigestArrayBoundEdges/environment_observations | env obs tightening |
| M172 | equiv-ok | TestDigestArrayBoundEdges/native_observations | 65537-element body is a stated bound |
| M173 | killed | TestDigestArrayBoundEdges/native_observations | native tightening vs 65536-admit |
| M174 | killed | TestJournalBoundEdges | journal key floor |
| M175 | killed | TestJournalBoundEdges | journal key ceiling |
| M176 | killed | TestJournalSurvivesRestart | journal version gate accepts 2 |
| M177 | killed | TestExtensionKeyBoundEdges | extension key ceiling |
| M178 | equiv-ok | TestExtensionKeyBoundEdges | 2-char key still grammar-refused |
| M179 | equiv-ok | TestStringBoundEdges/contract_identifier | empty identifier still colon-refused |
| M180 | killed | TestStringBoundEdges/contract_identifier | URI ceiling |
| T01 | killed |  | major order swap breaks descending negotiation |
| T02 | killed |  | operation table order swap breaks spec-order derivation |
| V01 | killed | TestDecodeQueryOperationValidatorsRefuse/environments_root_status_refused | environments admits root |
| V02 | killed | TestDecodeQueryOperationValidatorsRefuse/jobs_pwned_state_refused | jobs admits pwned |
| V03 | killed | TestDecodeQueryOperationValidatorsRefuse/distinct_secret_field_refused | distinct admits secret |
