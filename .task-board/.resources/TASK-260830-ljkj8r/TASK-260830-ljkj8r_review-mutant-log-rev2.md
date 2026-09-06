# TASK-260830-ljkj8r — reviewer mutant log rev2 (RUN-260906-8a5258)

Harness: copy-aside / one-step edit / `go test ./internal/dirnode/ -count=1` /
copy-back, byte-restore after every row. Production code is **byte-identical**
to rev1 (`git diff 4d8a599 bd3cb8c` touches only test files, `LOGBOOK.md`), so
every rev1 mutant coordinate still applies without re-anchoring: 0 NOT_APPLIED
across all five batteries.

Working tree verified equal to candidate tree
`bd3cb8c05f421365e1e20f9e18ad53cb13d33d18` after the last row.

Baseline: `go test ./... -count=1` 18 packages ok; `go vet ./...` exit 0;
`gofmt -l internal/` clean; `go test ./internal/dirnode -cover` 83.3%.

| battery | denominator derived from | applied | killed | survived | not applied | compile fail |
|---|---|---:|---:|---:|---:|---:|
| 1. rev1 bound battery, re-run verbatim | production helper call sites | 94 | **68** | 26 | 0 | 0 |
| 2. sites rev1 never touched (len guards, wrapper internals, shape-7/8/10 stated bounds) | production comparison sites | 54 | 42 | 12 | 0 | 0 |
| 3. targeted follow-ups | — | 9 | 3 | 6 | 0 | 0 |
| 4. refusal-arm reachability (every `if` whose body builds a `failX`) | production AST | 144 | 115 | 29 | 0 | 0 |
| 5. query operation obligations (every `return false` in query.go) | production AST | 89 | 58 | 31 | 0 | 0 |

Census-only kills: **0**. Every kill I recorded names at least one behavioural
driver; the four rows that redden `TestBoundGuardsAreCensused` (B12, B13, B21,
B24) each also redden a behavioural test in the same run.

## Battery 1 — rev1 battery re-run (68/94, was 19/94)

Zero resurrections: all 19 rev1 kills are still killed. The 26 survivors map
1:1 onto declared `equiv-ok` rows in `TASK-260830-ljkj8r_round2.md`, and each
masking gate was verified in production source rather than accepted from the
rationale:

| survivor class | rows | masking gate verified |
|---|---:|---|
| `requested_environment_ids` / `environment_ids` element floor+ceiling | 4 | `environmentIDPattern` `^[a-z][a-z0-9.-]{0,63}$` — 0 and 65 both unmatchable |
| `provider_ids` element floor+ceiling | 2 | `providerIDPattern` `^[a-z][a-z0-9-]{0,31}$` |
| closed-vocabulary element floor/ceiling + Nth-member count (`requested_capabilities`, `scopes`, `authentication_status`, `job states`, `enrich kinds`) | 13 | `validCapability` over the 8-name `capabilityOrder`; the 5/4/7/3-member tables; Nth valid member unbuildable under sorted-unique |
| `supported_versions` element floor 5→4 | 1 | `checkSemver` — shortest SemVer is 5 characters |
| projection `fields` element floor/ceiling + count | 3 | 27-member `queryFields` registry |
| batch `operations` ceiling 64→65 and `index` ceiling 63→64 | 2 | joint gate: the pair coincide at 65; both tightenings die (B28, B30) |
| `native_observation_ids` count 65536→65537 | 1 | stated bound (65537-element body not built) |

No false equivalence found.

## Battery 2/3 — sites rev1 never touched

Killed (42): `checkURI` ceiling both directions, extension key 253 both
directions and floor tightening, contract assertion 15/64 all four directions,
`checkNestedObjects` caller literals both directions, findings 4096 both
directions, `MaxFrameBytes`, `MaxDeadlineMS`, `MinDeadlineMS`, both frame
`>` → `>=` tightenings, batch floor/ceiling tightening, `index` tightening,
tags 256 both directions, confirmations 64 both directions, `skip` 1000000
both directions, `take` 1/1000 all three, sort 8 both directions, cursor
1/1024 all three, journal version gate both spellings, all three bootstrap
exit-status gates, `checkSortedUniqueStrings` uniqueness.

Survivors requiring a finding are listed in the verdict. The remainder are
verified-equivalent: `checkURI` floor `<1`→`<2` (2-character `a:` admits under
both; `<1`→`<4` dies, C09), extension key floor `<3`→`<2` (reverse-DNS minimum
is `a.b`), and the joint batch/index pair above.

## Battery 4 — refusal-arm reachability, 144 arms, 29 survive

`if <cond> { … failX … }` → `if false && (<cond>) { … }`, i.e. the arm is
removed while every other arm stays. A survivor means no committed test would
notice the arm's removal.

| survivor group | arms | site |
|---|---:|---|
| envelope `unknownMember` | 3 | `manifest.go:268`, `probe.go:338`, `query.go:415` |
| envelope `missingMember` | 5 | `probe.go:141`, `protocol.go:428`, `protocol.go:559`, `scan.go:78`, `scan.go:193` |
| `checkExtensions` | 2 | `probe.go:392` (probe response), `query.go:467` (query envelope) |
| empty-frame guard | 3 | `bootstrap.go:144`, `protocol.go:399`, `protocol.go:530` |
| strict-decode `fault != nil` masked by a downstream member arm | 8 | `manifest.go:261`, `probe.go:127`, `probe.go:331`, `protocol.go:545`, `query.go:408`, `scan.go:64`, `scan.go:179`, `scan.go:395` |
| envelope member-read `!ok` / `!present` masked downstream | 6 | `manifest.go:329`, `manifest.go:337`, `protocol.go:443`, `protocol.go:474`, `protocol.go:639`, `protocol.go:684` |
| marshal/parse `err != nil` | 2 | `protocol.go:377`, `scan.go:365` |

## Battery 5 — query operation obligations, 89 sites, 31 survive

`return false` → `return true`, one site per row: the obligation is removed and
every sibling stays.

| validator | obligations | driven | undriven |
|---|---:|---:|---:|
| `checkPlanContinueParameters` | 9 | 3 | 6 |
| `checkQuerySort` | 8 | 3 | 5 |
| `checkFilters` | 16 | 13 | 3 |
| `checkQueryParameters` | 5 | 2 | 3 |
| `checkExecutePlanParameters` | 5 | 2 | 3 |
| `checkSubject` | 4 | 1 | 3 |
| `checkUUIDv7DigestSubset` | 5 | 3 | 2 |
| `checkQueryFlags` | 3 | 1 | 2 |
| `checkTagsParameter` | 3 | 2 | 1 |
| `checkLineageParameters` | 2 | 1 | 1 |
| `checkDistinctParameters` | 2 | 1 | 1 |
| `isAnnotationMutation` | 1 | 0 | 1 |
| 11 validators at full coverage (`checkEnvironmentsParameters`, `checkJobsParameters`, `checkQueryProjection`, `checkEnrichParameters`, `checkPlansParameters`, `checkEnumSubset`, `checkQueryPagination`, `isListOperation`, `validQueryField`, `validQueryPreset`, `checkHostsParameters`) | 26 | 26 | 0 |
| **total** | **89** | **58** | **31** |

## Direct production probes (unmutated, through the production entry)

| probe | observed refusal |
|---|---|
| manifest `unknown member` vector (`manifest_test.go:62`) | `node manifest trailing data after the object` — the strict-decode arm, not `unknownMember` |
| query `unknown member` vector (`query_test.go:75`) | `directory query operation is not a closed QueryOperation` — the operation arm; the needle matched `operations[0].parameters` |
| probe-response `unknown member` vector (`probe_test.go:142`) | `probe response node build is not the closed DirectoryNodeBuild` — the node-build arm; the needle matched `node_build` |
| probe request / scan request / scan response / request-frame `missing member` vectors | all four report `misses a required member` — the vectors are built correctly; the assertions do not discriminate the arm |
| probe response `extensions:{"NOTDNS":{}}` | `probe response extensions are not reverse-DNS keyed` — the gate works, nothing drives it |
| `EncodeRequest(MajorV2, manifest, id, 3600001, {})` | refused today; `TestUint53BoundEdges` never calls it with that value |
