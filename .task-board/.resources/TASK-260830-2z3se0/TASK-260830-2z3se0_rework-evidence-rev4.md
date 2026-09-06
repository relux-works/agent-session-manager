# TASK-260830-2z3se0 — round-4 rework evidence (answers review-verdict-rev3)

Tree: worktree `task-board/story/STORY-260830-3drr2m`. Test-only
additions in `internal/sessadapter/` (`zero_absent_test.go`,
`bounds_edge_test.go`, `import_census_test.go`), a two-arm fix to
the constructor census in `inventory_test.go` (N6), and one LOGBOOK
entry. No production behavior change: the 323-arm inventory floor
is untouched, `tracecheck` acceptance_cases stays 88, and no new AC
rows are claimed. No commit per the CR-shape instruction.

## Fixes per finding

### B4 (blocking) — zero-value / absent-fact class closed by derivation
- `TestEnvelopeEchoEmptyStringRefuses`: the 10 echo members come
  from the production Required slices (`requestRequired[:2]`,
  `successRequired[:4]`, `failureRequired[:4]`, with head guards),
  wants from `ProtocolID`/`ProtocolVersion`/`wantRequest()`; each
  driven as `""` through `DecodeRequestFrame`,
  `CheckSuccessEnvelope`, `CheckFailureEnvelope`.
- `TestCheckTargetWriteGatesAbsentAdapterRefuses`: the adapter-side
  twin of the provider-side missing-surface case. Required names
  derived from `DoctorRequiredCapabilities` (both writer variants);
  each always-required name deleted alone refuses naming it; each
  writer deleted alone still passes; both writers deleted refuses
  naming the pair.
- `TestCapabilityUsableAbsentIsNotUsable`: every `capabilityOrder`
  name deleted through `CapabilityUsable` and `capabilityMapUsable`
  is unusable; `""` and unknown names are unusable.
- `TestCheckDoctorHealthyRefusesEmptyRequiredName`: `required=[""]`
  refuses `invalid_config` through `CheckDoctorHealthy`.

### B5 (blocking) — every survivor bound pinned at the edge
- `TestStringBoundEdges` (17 rows: all 16 survivor
  `checkStringBounds` maxima + probe detail 0..2048): min-1 refuses,
  min admits, max admits, max+1 refuses, all through the production
  entry. Sort-sensitive fixtures use same-width names
  (`c01..c64`, `1.0.0-a01..a33`).
- `TestArrayBoundEdges`: 0/1/max/max+1 for resume argv (128),
  required dispositions (7; the 8-entry refusal lands on the count
  arm, which fires before the element gates), contract rows (64),
  contract versions (32).

### B6 (blocking) — import census added, no site patch
- `TestAxerrorImportsAreUnaliased`: every production import of the
  error path must bind `axerror`; renamed, blank, and dot bindings
  fail; zero scanned files and zero importers both fail closed.
  Synthetic vectors prove the gate shape
  (`TestAxerrorImportCensusSeesAliases`). The constructor census is
  sound as written; no alias special-casing was added.

### N6 — switch fixed, not the comment
- `constructorSitesInFile` gains `case !topLevel` reporting the
  function-local registered-name case (was a silent fall-through);
  pinned by synthetic
  `TestConstructorSitesReportFunctionLocalRegisteredName`.

## Mutant battery (narrowing; 36 killed / 36)

Method: each mutant applied by exact-string replacement
(occurrence-targeted where the pattern repeats), grep-confirmed
PRESENT, `go vet`-clean before its verdict, production restored
after every mutant; tree sha-verified identical after the run
(script `/tmp/mutant-battery-r4.py`, log
`TASK-260830-2z3se0_mutant-battery-rev4.log`). Every named killer
passes on the clean tree (8 controls green). Survivors: none.

| Mutant | Narrows the gate to | Named test that fails |
|---|---|---|
| E1 request `protocol` `&& != ""` | admit a protocol-less request frame | TestEnvelopeEchoEmptyStringRefuses/request/protocol |
| E2 request `protocol_version` `&& != ""` | admit a version-less request frame | …/request/protocol_version |
| E3 success `protocol` `&& != ""` | admit a protocol-less success | …/success/protocol |
| E4 success `protocol_version` `&& != ""` | admit a version-less success | …/success/protocol_version |
| E5 success `request_id` `&& != ""` | admit an id-less success | …/success/request_id |
| E6 success `operation` `&& != ""` | admit an operation-less success | …/success/operation |
| E7 failure `protocol` `&& != ""` | admit a protocol-less failure | …/failure/protocol |
| E8 failure `protocol_version` `&& != ""` | admit a version-less failure | …/failure/protocol_version |
| E9 failure `request_id` `&& != ""` | admit an id-less failure | …/failure/request_id |
| E10 failure `operation` `&& != ""` | admit an operation-less failure | …/failure/operation |
| C1 `capabilityMapUsable` absent ⇒ true | target-write on a map missing adapter surfaces | TestCheckTargetWriteGatesAbsentAdapterRefuses/absent_native_read_back (+ unit) |
| C2 `CapabilityUsable` absent ⇒ true | usable unseen capability | TestCapabilityUsableAbsentIsNotUsable |
| C3 doctor name check `&& name != ""` | empty required name passes | TestCheckDoctorHealthyRefusesEmptyRequiredName |
| M1 env range 256 → 257 | admit a 257-char version range | TestStringBoundEdges/manifest_environment_version_range |
| M2 native_session_id 512 → 513 | admit a 513-char session id | …/source_native_session_id |
| M3 opaque_source_ref 512 → 513 | admit a 513-char opaque ref | …/source_opaque_source_ref |
| M4 finding code 128 → 129 | admit a 129-char finding code | …/finding_code |
| M5 finding message 4096 → 4097 | admit a 4097-char message | …/finding_message |
| M6 remediation 4096 → 4097 | admit a 4097-char remediation | …/finding_remediation |
| M7 native_item_key 512 → 513 | admit a 513-char item key | …/capture_native_item_key |
| M8 argv word 4096 → 4097 | admit a 4097-char argv word | …/resume_argv_word |
| M9 tuple env_version 128 → 129 | admit a 129-char tuple version | …/tuple_environment_version |
| M10 suite_revision 128 → 129 | admit a 129-char suite revision | …/tuple_suite_revision |
| M11 native_cli_family 128 → 129 | admit a 129-char CLI family | …/tuple_native_cli_family |
| M12 fidelity code 128 → 129 | admit a 129-char limit code | …/fidelity_code |
| M13 affected_class 128 → 129 | admit a 129-char limit class | …/fidelity_affected_class |
| M14 fidelity detail 4096 → 4097 | admit a 4097-char limit detail | …/fidelity_detail |
| M15 contract_id 256 → 257 | admit a 257-char contract id | …/contract_identifier |
| M16 revocation_reason 4096 → 4097 | admit a 4097-char reason | …/revocation_reason |
| M17 argv count 128 → 129 | admit a 129-word argv | TestArrayBoundEdges/resume_argv_1..128 |
| M18 dispositions lower `==0` → `==-1` | admit an empty disposition list | …/required_dispositions_1..7 |
| M19 contracts 64 → 65 | admit a 65-row contract array | …/contracts_1..64 |
| M20 versions 32 → 33 | admit a 33-version row | …/versions_1..32 |
| A3 protocol.go `axe` import + `axe.New` arm | aliased eighth constructor ships | TestAxerrorImportsAreUnaliased (constructor census stays green) |
| A4 manifest.go `axe` import + `axe.New` arm | aliased inline refusal ships | TestAxerrorImportsAreUnaliased (constructor census stays green) |
| N1 switch fall-through restored | function-local registered name hides | TestConstructorSitesReportFunctionLocalRegisteredName/function-local_registered_name_reports |

Anti-inflation: A3/A4 hold `TestRefusalConstructorsMatchProduction`
green while the import census fails, so the kill comes from the new
census, not the 323 floor. Method bound: battery traverses the
reviewer's 35 survivors + the N6 arm; delegated packages
(`canonicaljson`, `scalar`, `axerror`) not attacked; no in-repo
importers, gates measured at the exported API.

## AC coverage delta (this round; prior rows stand)

14 of 14 new gate rows driven through the production entry point:

| AC row | Production call site | Named committed test |
|---|---|---|
| envelope echo, request protocol/version | DecodeRequestFrame (protocol.go) | TestEnvelopeEchoEmptyStringRefuses (2 rows) |
| envelope echo, success identity | CheckSuccessEnvelope | TestEnvelopeEchoEmptyStringRefuses (4 rows) |
| envelope echo, failure identity | CheckFailureEnvelope | TestEnvelopeEchoEmptyStringRefuses (4 rows) |
| target-write adapter absence | CheckTargetWriteGates (probe.go) | TestCheckTargetWriteGatesAbsentAdapterRefuses |
| capability presence default | CapabilityUsable / capabilityMapUsable | TestCapabilityUsableAbsentIsNotUsable |
| doctor required-name gate vs "" | CheckDoctorHealthy | TestCheckDoctorHealthyRefusesEmptyRequiredName |
| manifest/probe/context/tuple string maxima | DecodeManifest / DecodeSourceSelector / DecodeFinding / DecodeCapturePlanItem / CheckSuccessBody / DecodeTuple / DecodeTupleEntry / DecodeProbe | TestStringBoundEdges (17 rows) |
| argv / disposition / contract / version caps | CheckSuccessBody / CheckRequestBody / DecodeTupleEntry | TestArrayBoundEdges (4 rows) |
| error-import spelling | production imports (all files) | TestAxerrorImportsAreUnaliased |
| constructor function-local case | constructorSitesInFile (inventory_test.go) | TestConstructorSitesReportFunctionLocalRegisteredName |

## Gates (exit codes observed directly, no pipes)

- `go build ./...`: exit 0
- `go vet ./...`: exit 0; `GOOS=windows go vet ./...`: exit 0
- `gofmt -l internal/`: empty
- `go test ./... -count=1`: 17/17 packages ok, exit 0
- `go test ./internal/sessadapter/ -count=1 -v`: 976 PASS, 0 FAIL, exit 0
- `go test ./internal/sessadapter/ -cover`: 81.5% of statements, exit 0
- `go test -race ./internal/sessadapter/ -count=1`: exit 0
- `go run ./internal/traceability/cmd/tracecheck`: ok (acceptance_cases=88), exit 0
