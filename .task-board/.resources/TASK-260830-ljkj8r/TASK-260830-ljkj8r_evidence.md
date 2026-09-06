# TASK-260830-ljkj8r — Directory Node protocol host: evidence

Leaf 2 of STORY-260830-3drr2m. New package `internal/dirnode`
(7 production files, 9 test files), 6 traceability acceptance
cases (88 → 94), README paragraph. No commit (board creates the
signed commit); work is in the working tree for CR publication.

## AC coverage: 10 of 10 rows driven through production entries

| # | AC row | Production call site | Named committed test(s) |
|---|--------|----------------------|-------------------------|
| 1 | Major bootstrap: descending enumeration, fresh process per attempt, one exact downgrade tuple, success/exit rules, own incompatible_protocol when no major is common | `DecideBootstrapStep`, `NextLowerMajor`, `ProcessGuard.Claim`, `VersionForMajor` (`internal/dirnode/bootstrap.go`, `protocol.go`) | `TestBootstrapSelectsV2Directly`, `TestBootstrapFallsBackV2ToV1`, `TestBootstrapServesV1OnlyCaller`, `TestBootstrapEndsWithoutCommonMajor`, `TestBootstrapRefusesNonExactDowngradeTuple`, `TestBootstrapTerminalFailuresNeverDowngrade`, `TestBootstrapRefusesNonManifestAttempt`, `TestProcessGuardRefusesReuse`, `TestNextLowerMajorBounds` |
| 2 | Manifest framing (closed 17-member 1.0.0, registries, limits, digest) | `DecodeManifest`, `ManifestDigest`, `Manifest.Supports` (`manifest.go`) | `TestDecodeManifestAcceptsFixture`, `TestDecodeManifestClosedMemberRules`, `TestDecodeManifestRegistryRules`, `TestManifestLimitsBounds` |
| 3 | Probe framing (per-major platform vocabulary, response, node-build equality) | `CheckProbeRequest`, `CheckProbeResponse`, `CheckNodeBuildEqualsManifest` (`probe.go`) | `TestProbeRequestMajorVocabularies`, `TestProbeRequestRefusals`, `TestProbeResponseAcceptsFixture`, `TestProbeResponseAdmitsOpaqueEnvironments`, `TestProbeResponseRefusals`, `TestProbeResponseFindingBound`, `TestNodeBuildEqualityRefusesDrift` |
| 4 | Scan framing | `CheckScanRequest`, `CheckScanResponse` (`scan.go`) | `TestCheckScanRequestAcceptsFixture`, `TestCheckScanRequestRefusals`, `TestCheckScanRequestNullableOptions`, `TestCheckScanResponseAcceptsFixture`, `TestCheckScanResponseRefusals` |
| 5 | Query framing (§10.8.5: envelope, caller, 17-op union, projection, pagination, flags, cursor) | `DecodeQuery`, `CheckCursorReuse` (`query.go`) | `TestDecodeQueryAcceptsFixture`, `TestDecodeQueryIndexRules`, `TestDecodeQueryEnvelopeRules`, `TestDecodeQueryRegistryRules`, `TestDecodeQueryMutationFlagRules`, `TestDecodeQueryPaginationRules`, `TestCheckCursorReuse` |
| 6 | Host checks (façade bindings, build equality, advertisement, process freshness) | `CheckManifestBindings`, `CheckNodeBuildEqualsManifest`, `AvailableCapabilities`, `ProcessGuard` | `TestCheckManifestBindings`, `TestNodeBuildEqualityRefusesDrift`, `TestAvailableCapabilitiesNeverUpgrades`, `TestProcessGuardRefusesReuse` |
| 7 | Structured errors (static 1.2.0 binding, 8 codes, exit classes) | `ErrorVersion`, all `failX` constructors (`protocol.go`) | `TestStructuredErrorBinding` + every refusal test via `requireCode` |
| 8 | Exact contract fixtures + negative/refusal cases | all entries | fixture suite (`fixtures_test.go`, independent literals) + per-gate refusal tables |
| 9 | Crash/idempotency evidence for mutating ops (scan, enrichment-run share the keyed journal) | `Journal.CheckAndRecord`, `Journal.Export`, `Journal.Import` (`scan.go`) | `TestJournalReplaysIdenticalBody`, `TestJournalRefusesChangedBody`, `TestJournalScopesKeysByOperation`, `TestJournalSurvivesRestart`, `TestJournalImportRefusesUnsortedRecords`, `TestJournalExportIsDeterministic` |
| 10 | No unsupported capability advertised (§8.1 labels are not aliases) | `AvailableCapabilities`, `checkCapabilities` (`manifest.go`) | `TestAvailableCapabilitiesNeverUpgrades`, `TestManifestCapabilityReasonCoherence`, `TestDecodeManifestRegistryRules` |

Stated bounds (not driven, declared): the 8 unframed registry
operations (`TestFramedSubsetIsExact` pins the set);
EnvironmentObservation / InventoryBatch / enrichment / runtime
content (`TestProbeResponseAdmitsOpaqueEnvironments` pins the
presence-only boundary); transport process lifecycle and
CallerContext authentication (caller-owned, documented in
`doc.go`); 2 defensive arms in `defensiveArms` (encoder-produced
values that cannot fail encoding).

## Census (leaf-1 lesson applied first, not last)

- Shapes enumerated up front in `census_test.go`: 10 of 10
  AST shapes derived-or-refused, each with a synthetic proof
  (`TestConstructorAliasSpellingsFailDerivation`) or a named
  residue (`frameFault` alias, assembly/reflect, unscanned
  files, second-argument identity).
- Result: 147 derived arms (145 witnessed + 2 defensive),
  8 constructors, 5 frame faults. Both directions green.
- The census caught a real omission during authoring (5 Import
  arms missing from the witness table) and 3 vacuous vectors
  (no-op replacements now fail via `replaceOnce`).
- Registries derived from `internal/specdoc` text: 11
  operations, 8 capabilities, 11+6 query operations.

## Narrowing-mutant battery (each restored byte-for-byte via copy-back + `cmp`)

| Mutant | Narrows the gate to | Named test that fails (exit 1) | Survivors / bound stated |
|--------|---------------------|--------------------------------|--------------------------|
| M1: downgrade admits `retryable=true` | exact-tuple check minus one field | `TestBootstrapRefusesNonExactDowngradeTuple/retryable_claim` | none |
| M2: v1 probe admits `macos` | one cross-major token | `TestProbeRequestMajorVocabularies/v1_refuses_macos` | none |
| M3: advertisement includes `conditional` | one upgraded status | `TestAvailableCapabilitiesNeverUpgrades` | none |
| M4: scan changed bodies replay prior result | one operation class exempted | `TestJournalRefusesChangedBody` | none |
| M5: query index contiguity dropped | sparse/duplicate/reordered indexes | `TestDecodeQueryIndexRules` (4 subtests; empty batch still refuses — narrowing precision) | none |
| M6: response admits `schema_version` 2.0.0 | one relabeled version | `TestCheckSuccessEnvelopeRefusals` + `TestCheckFailureEnvelopeRefusals` (relabeled rows) | none |
| M7: non-available capability without reason admitted | one incoherent member | `TestManifestCapabilityReasonCoherence` (2 subtests) | none |
| M8: aliased error import (`axe`, path token preserved) in `protocol.go`; compiles, behavior identical | source-text gate attack | 4 census tests fail (`TestAxerrorImportsAreUnaliased`, `TestRefusalConstructorsMatchProduction`, both witness-direction tests fail closed on zero derived constructors); FULL behavioral suite stays green (no entry test fails) | none; census-only kills scored separately per evidence rules |

Census-only kills: M8's 4. No other mutant reddened a
census test (expected — they touch no construction site).
Surviving mutants: 0. Every mutant was confirmed PRESENT
(its named test failed) and reverted without `git checkout`
(copy aside, copy back, `cmp` clean after every battery).

## Gates (real exit codes, standalone processes, no pipes)

- `go test ./internal/dirnode/ -count=1` → exit 0
- `go test ./internal/dirnode/ -count=3` → exit 0 (map-iteration shake)
- `go test ./internal/dirnode/ -race -count=1` → exit 0
- `go test ./internal/traceability/ ./internal/traceability/cmd/tracecheck/ -race -count=1` → exit 0
- `go test ./... -count=1` → exit 0 (18 packages ok, 0 fail)
- `go test ./... -cover` → exit 0 (dirnode 75.0% of statements)
- `go vet ./...` → exit 0; `GOOS=windows go vet ./...` → exit 0
- `gofmt -l internal/` → clean
- `go run ./internal/traceability/cmd/tracecheck -root .` → `traceability ok: ... acceptance_cases=94 ...` exit 0
- Full-repo `-race` NOT run: exceeds one bounded shell call; race
  was run on every touched package instead (stated, not implied).

## Specification findings (recorded in logbook)

1. §10.8.5 prose names `query_cursor_mismatch`; the pinned
   error registry (catalog + §15.3 table) does not register it.
   Cursor reuse refuses as `query_invalid` with a
   cursor-mismatch detail (`CheckCursorReuse`); divergence
   recorded on the call site.
2. Response envelope carries its own `schema` +
   `schema_version` (= 1.0.0 both majors): the downgrade tuple
   names a response `schema_version` distinct from every echo,
   so a schemaless reading cannot carry it. Pinned in
   envelope tests both branches.
3. `framedOperations` = manifest/probe/scan + Directory
   Query; the other 8 registry operations are a named bound
   for the sibling leaves, not a silent absence.
4. `capability_unavailable` constructor removed: no entry
   point in this deliverable raises it (no dispatcher, no
   invented availability claim); the taxonomy test pins the
   8 emitted codes.
5. `node manifest is not canonical JSON` is reachable, not
   defensive: member checks cross extensions opaquely while
   the canonicalizer enforces depth 256 — proven by the
   300-deep extensions vector.
