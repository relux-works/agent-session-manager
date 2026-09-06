# TASK-260830-ljkj8r — reviewer mutant log (RUN-260906-ccfe0d)

Every mutant applied by copy-aside / edit / `go test ./internal/dirnode/ -count=1` /
copy-back, with a byte comparison after each restore. Working tree verified equal to
candidate tree `4d8a5998ba35ce511b3db27288f9ef2454d8c5a6` after the last row.

## Sweep 1 — bound edges (one step per row, gate preserved)

Applied 94, measured 94, killed 19, survived 75, NOT_APPLIED 0, COMPILE_FAIL 0.

| # | status | mutant |
|---:|---|---|
| 0 | KILLED | `manifest.go:296 checkStringBounds(members["node_id"]) min 1->0` |
| 1 | **SURVIVED** | `manifest.go:296 checkStringBounds(members["node_id"]) max 128->129` |
| 2 | **SURVIVED** | `manifest.go:585 checkStringBounds(reason) min 1->0` |
| 3 | **SURVIVED** | `manifest.go:585 checkStringBounds(reason) max 128->129` |
| 4 | KILLED | `manifest.go:635 checkUint53Bounds(members["max_frame_bytes"]) min 1->0` |
| 5 | KILLED | `manifest.go:639 checkUint53Bounds(members["max_scan_instances"]) min 1->0` |
| 6 | **SURVIVED** | `manifest.go:639 checkUint53Bounds(members["max_scan_instances"]) max 65536->65537` |
| 7 | **SURVIVED** | `manifest.go:643 checkUint53Bounds(members["max_inventory_take"]) min 1->0` |
| 8 | KILLED | `manifest.go:643 checkUint53Bounds(members["max_inventory_take"]) max 1000->1001` |
| 9 | KILLED | `manifest.go:647 checkUint53Bounds(members["max_excerpt_count"]) max 20->21` |
| 10 | KILLED | `manifest.go:651 checkUint53Bounds(members["max_excerpt_bytes"]) max 4096->4097` |
| 11 | **SURVIVED** | `manifest.go:655 checkUint53Bounds(members["max_enrichment_events"]) min 1->0` |
| 12 | KILLED | `manifest.go:655 checkUint53Bounds(members["max_enrichment_events"]) max 5000->5001` |
| 13 | **SURVIVED** | `manifest.go:659 checkUint53Bounds(members["max_enrichment_bytes"]) min 1->0` |
| 14 | KILLED | `manifest.go:659 checkUint53Bounds(members["max_enrichment_bytes"]) max 4194304->4194305` |
| 15 | KILLED | `manifest.go:384 checkSortedUniqueDigests(members["redaction_policy_ids"]) minc 1->0` |
| 16 | **SURVIVED** | `manifest.go:384 checkSortedUniqueDigests(members["redaction_policy_ids"]) maxc 64->65` |
| 17 | **SURVIVED** | `manifest.go:392 checkSortedUniqueDigests(members["enrichment_profile_ids"]) maxc 256->257` |
| 18 | **SURVIVED** | `manifest.go:598 checkSortedUniqueDigests(members["evidence_ids"]) maxc 64->65` |
| 19 | **SURVIVED** | `manifest.go:447 checkSortedUniqueStrings(raw) minlen 5->4` |
| 20 | **SURVIVED** | `manifest.go:447 checkSortedUniqueStrings(raw) maxlen 64->65` |
| 21 | KILLED | `manifest.go:447 checkSortedUniqueStrings(raw) minc 1->0` |
| 22 | **SURVIVED** | `manifest.go:447 checkSortedUniqueStrings(raw) maxc 16->17` |
| 23 | **SURVIVED** | `probe.go:422 checkStringBounds(members["node_id"]) min 1->0` |
| 24 | **SURVIVED** | `probe.go:422 checkStringBounds(members["node_id"]) max 128->129` |
| 25 | KILLED | `probe.go:512 checkStringBounds(members["code"]) min 1->0` |
| 26 | **SURVIVED** | `probe.go:512 checkStringBounds(members["code"]) max 128->129` |
| 27 | KILLED | `probe.go:516 checkStringBounds(members["message"]) min 1->0` |
| 28 | **SURVIVED** | `probe.go:516 checkStringBounds(members["message"]) max 4096->4097` |
| 29 | **SURVIVED** | `probe.go:520 checkOptionalString(members["remediation"]) min 1->0` |
| 30 | **SURVIVED** | `probe.go:520 checkOptionalString(members["remediation"]) max 4096->4097` |
| 31 | **SURVIVED** | `probe.go:164 checkSortedUniqueStrings(members["requested_environment_ids"]) minlen 1->0` |
| 32 | **SURVIVED** | `probe.go:164 checkSortedUniqueStrings(members["requested_environment_ids"]) maxlen 64->65` |
| 33 | **SURVIVED** | `probe.go:164 checkSortedUniqueStrings(members["requested_environment_ids"]) maxc 64->65` |
| 34 | **SURVIVED** | `probe.go:181 checkSortedUniqueStrings(members["requested_capabilities"]) minlen 1->0` |
| 35 | **SURVIVED** | `probe.go:181 checkSortedUniqueStrings(members["requested_capabilities"]) maxlen 64->65` |
| 36 | **SURVIVED** | `probe.go:181 checkSortedUniqueStrings(members["requested_capabilities"]) maxc 8->9` |
| 37 | **SURVIVED** | `scan.go:477 checkStringBounds(members["key"]) min 1->0` |
| 38 | **SURVIVED** | `scan.go:477 checkStringBounds(members["key"]) max 512->513` |
| 39 | KILLED | `scan.go:117 checkUint53Bounds(members["max_instances"]) min 1->0` |
| 40 | KILLED | `scan.go:117 checkUint53Bounds(members["max_instances"]) max 65536->65537` |
| 41 | KILLED | `scan.go:109 checkOptionalString(members["cursor"]) min 1->0` |
| 42 | **SURVIVED** | `scan.go:109 checkOptionalString(members["cursor"]) max 4096->4097` |
| 43 | KILLED | `scan.go:223 checkOptionalString(members["next_cursor"]) min 1->0` |
| 44 | **SURVIVED** | `scan.go:223 checkOptionalString(members["next_cursor"]) max 4096->4097` |
| 45 | KILLED | `scan.go:93 checkSortedUniqueDigests(members["installation_ids"]) minc 1->0` |
| 46 | **SURVIVED** | `scan.go:93 checkSortedUniqueDigests(members["installation_ids"]) maxc 256->257` |
| 47 | KILLED | `scan.go:207 checkSortedUniqueDigests(members["environment_observation_ids"]) minc 1->0` |
| 48 | **SURVIVED** | `scan.go:207 checkSortedUniqueDigests(members["environment_observation_ids"]) maxc 256->257` |
| 49 | **SURVIVED** | `scan.go:215 checkSortedUniqueDigests(members["native_observation_ids"]) maxc 65536->65537` |
| 50 | **SURVIVED** | `query.go:503 checkStringBounds(members["caller_id"]) min 1->0` |
| 51 | **SURVIVED** | `query.go:503 checkStringBounds(members["caller_id"]) max 256->257` |
| 52 | **SURVIVED** | `query.go:507 checkStringBounds(members["authentication_subject"]) min 1->0` |
| 53 | **SURVIVED** | `query.go:507 checkStringBounds(members["authentication_subject"]) max 512->513` |
| 54 | **SURVIVED** | `query.go:807 checkStringBounds(members["title"]) min 1->0` |
| 55 | **SURVIVED** | `query.go:807 checkStringBounds(members["title"]) max 512->513` |
| 56 | **SURVIVED** | `query.go:761 checkSortedUniqueDigests(members["profile_ids"]) maxc 256->257` |
| 57 | **SURVIVED** | `query.go:780 checkSortedUniqueDigests(members["plan_ids"]) maxc 256->257` |
| 58 | **SURVIVED** | `query.go:841 checkSortedUniqueDigests(raw) maxc 1024->1025` |
| 59 | **SURVIVED** | `query.go:717 checkSortedUniqueUUIDv7(members["host_ids"]) maxc 256->257` |
| 60 | **SURVIVED** | `query.go:730 checkSortedUniqueUUIDv7(members["host_ids"]) maxc 256->257` |
| 61 | **SURVIVED** | `query.go:758 checkSortedUniqueUUIDv7(members["job_ids"]) maxc 256->257` |
| 62 | **SURVIVED** | `query.go:783 checkSortedUniqueUUIDv7(members["operation_ids"]) maxc 256->257` |
| 63 | **SURVIVED** | `query.go:965 checkSortedUniqueUUIDv7(members["host_ids"]) maxc 256->257` |
| 64 | **SURVIVED** | `query.go:968 checkSortedUniqueUUIDv7(members["workspace_ids"]) maxc 256->257` |
| 65 | **SURVIVED** | `query.go:519 checkSortedUniqueStrings(members["scopes"]) minlen 1->0` |
| 66 | **SURVIVED** | `query.go:519 checkSortedUniqueStrings(members["scopes"]) maxlen 64->65` |
| 67 | KILLED | `query.go:519 checkSortedUniqueStrings(members["scopes"]) minc 1->0` |
| 68 | **SURVIVED** | `query.go:519 checkSortedUniqueStrings(members["scopes"]) maxc 5->6` |
| 69 | **SURVIVED** | `query.go:733 checkSortedUniqueStrings(members["environment_ids"]) minlen 1->0` |
| 70 | **SURVIVED** | `query.go:733 checkSortedUniqueStrings(members["environment_ids"]) maxlen 64->65` |
| 71 | **SURVIVED** | `query.go:733 checkSortedUniqueStrings(members["environment_ids"]) maxc 64->65` |
| 72 | **SURVIVED** | `query.go:742 checkSortedUniqueStrings(members["authentication_status"]) minlen 1->0` |
| 73 | **SURVIVED** | `query.go:742 checkSortedUniqueStrings(members["authentication_status"]) maxlen 32->33` |
| 74 | **SURVIVED** | `query.go:742 checkSortedUniqueStrings(members["authentication_status"]) maxc 4->5` |
| 75 | **SURVIVED** | `query.go:764 checkSortedUniqueStrings(members["states"]) minlen 1->0` |
| 76 | **SURVIVED** | `query.go:764 checkSortedUniqueStrings(members["states"]) maxlen 32->33` |
| 77 | **SURVIVED** | `query.go:764 checkSortedUniqueStrings(members["states"]) maxc 7->8` |
| 78 | **SURVIVED** | `query.go:851 checkSortedUniqueStrings(members["kinds"]) minlen 1->0` |
| 79 | **SURVIVED** | `query.go:851 checkSortedUniqueStrings(members["kinds"]) maxlen 64->65` |
| 80 | **SURVIVED** | `query.go:851 checkSortedUniqueStrings(members["kinds"]) minc 1->0` |
| 81 | **SURVIVED** | `query.go:851 checkSortedUniqueStrings(members["kinds"]) maxc 3->4` |
| 82 | **SURVIVED** | `query.go:956 checkSortedUniqueStrings(members["provider_ids"]) minlen 1->0` |
| 83 | **SURVIVED** | `query.go:956 checkSortedUniqueStrings(members["provider_ids"]) maxlen 32->33` |
| 84 | **SURVIVED** | `query.go:956 checkSortedUniqueStrings(members["provider_ids"]) maxc 64->65` |
| 85 | **SURVIVED** | `query.go:971 checkSortedUniqueStrings(members["states"]) minlen 1->0` |
| 86 | **SURVIVED** | `query.go:971 checkSortedUniqueStrings(members["states"]) maxlen 128->129` |
| 87 | **SURVIVED** | `query.go:971 checkSortedUniqueStrings(members["states"]) maxc 64->65` |
| 88 | **SURVIVED** | `query.go:983 checkSortedUniqueStrings(members["warnings"]) minlen 1->0` |
| 89 | **SURVIVED** | `query.go:983 checkSortedUniqueStrings(members["warnings"]) maxlen 256->257` |
| 90 | **SURVIVED** | `query.go:983 checkSortedUniqueStrings(members["warnings"]) maxc 128->129` |
| 91 | **SURVIVED** | `query.go:1064 checkSortedUniqueStrings(fieldsRaw) minlen 1->0` |
| 92 | **SURVIVED** | `query.go:1064 checkSortedUniqueStrings(fieldsRaw) maxlen 64->65` |
| 93 | **SURVIVED** | `query.go:1064 checkSortedUniqueStrings(fieldsRaw) maxc 128->129` |

## Sweep 2 — closed vocabulary tables (sentinel member appended)

Applied 54, measured 54, killed 20, survived 34.
Weak discriminator where the production entry already carries an unknown-token
witness; see N1 in the verdict for which survivors that caveat applies to.

| status | table |
|---|---|
| KILLED | `TABLE manifest.go:23 var capabilityOrder` |
| **SURVIVED** | `TABLE manifest.go:58 var capabilityStatuses` |
| KILLED | `TABLE manifest.go:103 var manifestRequired` |
| KILLED | `TABLE manifest.go:133 var capabilityRequired` |
| KILLED | `TABLE manifest.go:154 var limitsRequired` |
| KILLED | `TABLE manifest.go:173 var assertionRequired` |
| **SURVIVED** | `TABLE probe.go:22 var platformV1` |
| **SURVIVED** | `TABLE probe.go:29 var platformV2` |
| **SURVIVED** | `TABLE probe.go:38 var architectures` |
| KILLED | `TABLE probe.go:92 var probeRequestRequired` |
| KILLED | `TABLE probe.go:225 var probeResponseRequired` |
| KILLED | `TABLE probe.go:245 var nodeBuildRequired` |
| KILLED | `TABLE probe.go:264 var findingRequired` |
| **SURVIVED** | `TABLE probe.go:274 var findingSeverities` |
| KILLED | `TABLE scan.go:36 var scanRequestRequired` |
| KILLED | `TABLE scan.go:153 var scanResponseRequired` |
| **SURVIVED** | `VOCAB query.go:631 "sessions", "count"` |
| **SURVIVED** | `VOCAB query.go:633 "session"` |
| **SURVIVED** | `VOCAB query.go:637 "hosts"` |
| **SURVIVED** | `VOCAB query.go:639 "environments"` |
| **SURVIVED** | `VOCAB query.go:641 "jobs"` |
| **SURVIVED** | `VOCAB query.go:643 "plans"` |
| **SURVIVED** | `VOCAB query.go:645 "distinct"` |
| **SURVIVED** | `VOCAB query.go:647 "set_title"` |
| **SURVIVED** | `VOCAB query.go:651 "set_tags"` |
| **SURVIVED** | `VOCAB query.go:655 "set_pin"` |
| **SURVIVED** | `VOCAB query.go:659 "enrich"` |
| **SURVIVED** | `VOCAB query.go:662 "plan_continue"` |
| **SURVIVED** | `VOCAB query.go:665 "execute_plan"` |
| **SURVIVED** | `VOCAB query.go:680 "ax_session", "native_instance"` |
| **SURVIVED** | `VOCAB query.go:748 "available", "missing", "expired", "unknown"` |
| **SURVIVED** | `VOCAB query.go:770 "queued", "claimed", "running", "succeeded", "superseded", "failed", "` |
| **SURVIVED** | `VOCAB query.go:798 "kind", "lineage_anchor", "provider", "host", "workspace", "state", "m` |
| **SURVIVED** | `VOCAB query.go:857 "generated_title", "summary", "recent_activity"` |
| **SURVIVED** | `VOCAB query.go:883 "attach", "resume", "takeover", "fork", "adopt", "clone", "move", "ope` |
| **SURVIVED** | `VOCAB query.go:892 "refuse", "exact_checkpoint", "materialize_copy"` |
| **SURVIVED** | `VOCAB query.go:901 "retain", "stop_and_release"` |
| KILLED | `TABLE query.go:44 var queryRequired` |
| KILLED | `TABLE query.go:65 var callerRequired` |
| KILLED | `TABLE query.go:93 var operationRequired` |
| KILLED | `TABLE query.go:127 var filterRequired` |
| KILLED | `TABLE query.go:151 var sortRequired` |
| KILLED | `TABLE query.go:159 var readOperations` |
| KILLED | `TABLE query.go:175 var mutationOperations` |
| **SURVIVED** | `TABLE query.go:220 var listOperations` |
| **SURVIVED** | `TABLE query.go:241 var annotationMutations` |
| **SURVIVED** | `TABLE query.go:286 var queryFields` |
| **SURVIVED** | `TABLE query.go:327 var queryPresets` |
| **SURVIVED** | `TABLE query.go:346 var querySortFields` |
| **SURVIVED** | `TABLE query.go:360 var queryScopes` |
| **SURVIVED** | `VOCAB protocol.go:89 "1.0.0"` |
| **SURVIVED** | `VOCAB protocol.go:91 "2.0.0"` |
| KILLED | `TABLE protocol.go:275 var requestRequired` |
| KILLED | `TABLE protocol.go:303 var responseRequired` |

## Sweep 3 — targeted narrowing mutants inside 0%-coverage validators

| mutant | result |
|---|---|
| `checkEnvironmentsParameters` authentication_status admits `"root"` | **SURVIVED** |
| `checkJobsParameters` states admits `"pwned"` | **SURVIVED** |
| `checkDistinctParameters` field admits `"secret"` | **SURVIVED** |

## Controls

| control | result |
|---|---|
| `max_inventory_take` upper 1000->1001 (edge is probed) | KILLED by `TestManifestLimitsBounds/take_past_bound` |
| sessadapter `checkStringBounds(display_name,1,128)` -> `1,129` | KILLED by `TestDecodeManifestValueRules` |
| producer M8 aliased error import (`axe`), path token preserved | 4 census tests fail, behavioural suite green — reproduces as reported |
| ownership registry: one acceptance-case id renamed | digest `d7a837dc…` differs from reviewed `fbdbb5b6…` in both traceability and tracecheck |

## Reachability probes (production entry, unmutated)

Confirms the survivors above are reachable and correctly refused today, so the
gap is in the evidence and not in the bounds.

| input through production entry | current verdict |
|---|---|
| `DecodeManifest` `max_scan_instances=65537` | refused |
| `DecodeManifest` `max_inventory_take=0` | refused |
| `DecodeQuery` `caller_id=""` | refused |
| `DecodeQuery` `caller_id` at 257 characters | refused |
| `DecodeQuery` `fields:["nope"]` (no duplicate member) | refused |
| `DecodeQuery` `fields:["id"]`, `fields:["host","id"]` | admitted |
| `DecodeQuery` `fields:["id","host"]` (unsorted) | refused |
| committed `TestDecodeQueryRegistryRules/unknown_field` vector | refused as duplicate member, not by the field registry |

