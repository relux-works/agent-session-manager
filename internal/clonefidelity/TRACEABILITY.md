# Fidelity Clause Coverage (TASK-260830-2ya5le)

Owner: `internal/clonefidelity`. Authority:
`internal/specdoc/SPEC.v0.7.0.md`, Section 13.14.2 "Fidelity,
projection, and lineage" (line 10548+). This is the FIRST leaf's
clause record for the fidelity-projection-and-validation-contracts
story; record clause-to-test bindings here. The
`internal/traceability` registry edit belongs to the Story's final
leaf, not this one.

## Why this package exists

Section 13.14.2 states the disposition, reason, profile, and
strategy vocabularies, `FidelityCounts`,
`FidelityDispositionRecord`, the Fidelity Report 1.0.0 closed table,
and the reconciliation rules that follow it. No landed package owns
them: the `canonicaljson` closed-shape registry recognizes
`urn:ax:schema:fidelity-report` only to refuse it
(`rejectUnsupportedImmutableObjectShape`, pinned by that table's
census), so this package is the validating owner. Sibling leaves
own Projection Plan, Read-Back, Validation Report, Migration
Checkpoint, and Lineage Receipt; where the report references them,
only the identity or digest field is carried.

## Landed-owner reuse

| Scope | Owner | Use |
|---|---|---|
| Strict JSON framing | `internal/environ` | `DecodeStrictObject` at `decodeStrictObject`: the report, every row, every counts record, and every map member refuse lone-surrogate escapes, duplicate members, non-objects, and trailing data before any member is read |
| Character string measure | `internal/environ` | `StringLength` at every bound, both entries |
| String/uint53/digest/UUID bounds | `internal/environ` | `CheckStringBounds`, `CheckUint53Bounds`, `CheckDigest`, `CheckUUIDv7` at every decode site |
| Sorted-unique arrays | `internal/environ` | `CheckSortedUniqueStrings`, `CheckSortedUniqueDigests` at every decode site |
| Reverse-DNS grammar | `internal/environ` | `CheckReverseDNS` (new export; `CheckExtensions` refactored onto it with identical behavior) behind extension reasons |
| Environment Tuples | `internal/sessadapter` | `DecodeTuple` at both tuple members, both entries |
| Canonical bytes, digests, UUIDs | `internal/canonicaljson`, `internal/scalar` | `Canonicalize`, `SHA256Digest`, `ParseDigest`, `ParseUUIDv7` |
| Closed extension values | `internal/clonebundle` | `CheckExtensionsClosed` (new export) at both extension members on decode; `EncodeExtensions` (new export) on build |
| Omit-self digest | `internal/clonebundle` | `OmitSelfDigest` (new export) behind `verifySelfDigest` |
| Event kinds (26) | `internal/clonebundle` | `EventKinds()` iterated for the closed key set; `ValidEventKind` in tests |
| Content-block types (8) | `internal/clonebundle` | `ContentBlockTypes()` (new export) iterated for the closed key set; `ValidContentBlockType` (new export) in tests |
| Capture classes (9) | `internal/clonebundle` | `ValidCaptureClass` for `required_dispositions` keys |

The six `clonebundle`/`environ` exports are additive one-line
delegates over tested internals; no behavior in those packages
changed (importer grid in the conformance matrix,
`clonebundle`/`environ` suites green before and after).

## Interpretation record (decided, pinned)

These readings are taken where the pinned text underdetermines the
gate. Each is pinned by a named test, not prose.

| # | Reading | Test |
|---|---|---|
| I1 | `required_dispositions` keys are the 9 capture classes: the spec word for a closed class vocabulary is "capture-class" (kinds are "kinds", block types are "types") | `TestRequiredDispositions` (all 9 admit; `user_message` refused as a key) |
| I2 | `event_kind_counts`/`content_block_counts` carry exactly all 26/8 keys; per-disposition cross-sums equal the row-derived counts (the reconciliation expressible without per-row kind/block fields) | `TestEventKindBreakdown`, `TestContentBlockBreakdown` (closed keys + 182/56-cell +1 sweeps + non-zero -1 cells) |
| I3 | `forbid_reasons` admits literal bounded strings: the table row types them as strings, not reason codes | `TestForbidReasons` (`frobnicate` seals and decodes) |
| I4 | `required_dispositions` values are 1..7 by set semantics (sorted-unique over 7 dispositions); no separate upper check | `TestRequiredDispositions` |
| I5 | Non-synthesized rows admit a digest or null `canonical_object_id`: capture items that never normalized carry null | `TestRecordGridOracle` (non-synth/null-canonical cells admit) |
| I6 | Every row carries `source_evidence_ids[1..65536]` (shape bound for all rows, including synthesized) | `TestRecordGridOracle` (absent-evidence cells refuse on every disposition) |
| I7 | "A target report does not name Clone Validation Report, Lineage Receipt, G4, or a future event" holds structurally: the closed 29-member set has no such member | `TestReportMemberSet` (unknown member refused) |
| I8 | Row order is source rows sorted by key, then synthesized rows sorted by key, keys unique across all rows | `TestRowOrdering` |

## Clause map

| Pinned clause | Production entry | Test |
|---|---|---|
| 13.14.2 seven dispositions admitted exactly | `ValidDisposition`, `BuildFidelityReport`, `DecodeFidelityReport` | `TestDispositionVocabularyOracle` (7 admit + predicate/build/decode; 24 refusal tokens + 5 non-strings refuse with the literal code) |
| 13.14.2 five profiles admitted exactly | `ValidProfile`, `BuildFidelityReport`, `DecodeFidelityReport` | `TestProfileVocabularyOracle` (5 admit; 17 refusals + 5 non-strings) |
| 13.14.2 five strategies admitted exactly | `ValidStrategy` (registry entry; no report field carries a strategy) | `TestStrategyVocabularyOracle` (5 admit; 15 refuse) |
| 13.14.2 nineteen core reasons + reverse-DNS extensions; extensions can never redefine a core code | `ValidReasonCode`, `IsCoreReasonCode`, both report entries | `TestReasonVocabularyOracle` (19 core + 5 extensions admit; 27 refusals + empty + 5 non-strings refuse; disjointness asserted per core code), `TestReasonShadowImpossible` |
| 13.14.2 `FidelityDispositionRecord` exact 11-member shape | `DecodeFidelityReport` → `decodeDispositionRecord` | `TestRecordMemberSet` (unknown + 11 missing) |
| 13.14.2 exact requires empty reasons; non-exact requires ≥1 | `checkRecordReasonRule` (both entries) | `TestRecordGridOracle` (140 cells through both entries; 35 admit, 105 refuse with first-fire literals) |
| 13.14.2 synthesized requires no source canonical object | `checkRecordCanonicalRule` (both entries) | `TestRecordGridOracle` |
| 13.14.2 non-synthesized rows trace to source evidence | shape bound + stated resolution bound | `TestRecordGridOracle` (presence half); resolution bound below |
| 13.14.2 record string bounds at edge/edge+1, character measure | both entries via `environ.StringLength` | `TestRecordFieldBounds` (9 refusal pairs + edge admission + null locator), `TestRecordMultibyteMeasure` (512/513 `é`) |
| 13.14.2 sorted-unique arrays (unsorted + duplicate) | both entries | `TestRecordSortedUnique` (4 arrays × 2 shapes × 2 entries) |
| 13.14.2 Fidelity Report exact 29-member set, schema/version literals | `DecodeFidelityReport` | `TestReportMemberSet` (unknown + 29 missing + 9 literal cases), `TestReportRoundTrip` (wire literals) |
| 13.14.2 `fidelity_report_id` is the JCS digest omitting only itself | `BuildFidelityReport`, `verifySelfDigest` | `TestReportIdentityIndependent` (test-side JCS recompute agrees; omitting scope instead disagrees; tampered claim and non-digest refuse) |
| 13.14.2 archive/target branch-exact nullability | both entries | `TestArchiveTargetBranches` (10 build flips + 10 decode flips + profile/scope coupling 5×2×2) |
| 13.14.2 rows ordered, synthesized after source | `checkRowOrder` (both entries) | `TestRowOrdering` (5 build + 5 decode violations) |
| 13.14.2 `counts` derived from rows | `deriveCounts`, both entries | `TestCountsDerived` (literal totals + 14-cell ±1 sweep + zero-cell uint53 boundary) |
| 13.14.2 `reason_counts` derived from rows | `deriveReasonCounts`, both entries | `TestReasonCountsDerived` (literal occurrences incl. shared reason + 6-cell sweep + extra/missing/zero keys) |
| 13.14.2 `event_kind_counts` closed map reconciled | `decodeBreakdown`/`buildBreakdown` + `checkBreakdownSums` | `TestEventKindBreakdown` (182-cell +1 sweep + 7 non-zero -1 + closed keys + build-side rows) |
| 13.14.2 `content_block_counts` closed map reconciled | same | `TestContentBlockBreakdown` (56-cell +1 sweep + 7 non-zero -1 + closed keys + build row) |
| 13.14.2 `byte_counts` closed seven-disposition map | `decodeByteCounts`/`buildByteCounts` | `TestByteCountsReconciliationIsAStatedBoundUntilSpecDefinesByteBasis` (exact keys, uint53 values, 6 bad-value shapes; value reconciliation is the explicit stated bound below, witnessed by admission) |
| 13.14.2 `required_dispositions` closed policy map | `decodeRequiredDispositions`/`buildRequiredDispositions` | `TestRequiredDispositions` (9 classes admit, empty map admits, 6 build + 8 decode refusals) |
| 13.14.2 `forbid_reasons` sorted unique strings | both entries | `TestForbidReasons` (literal strings kept, 128/129 edges, 4 build + 6 decode refusals) |
| 13.14.2 row cardinality [1..1000000] | `checkRowCount` (both entries) | `TestRowCountEdges` (0/1/1000001 at the entries) + `TestCheckRowCountEdges` (999999/1000000/1000001 at the factored gate) |
| 13.14.2 evidence/attestation bounds at edge/edge+1 | both entries | `TestEvidenceBounds` (65536 admit ×3 + 64 admit; 65537/65 refuse both entries) |
| 13.14.2 scalar gates (digests, UUIDv7, tuples) | both entries via landed owners | `TestScalarGates` (15 build + 18 decode rows) |
| 13.14.2 AX number model (no fractions/exponents/strings/≥2^53) | decode via `environ.CheckUint53Bounds` | `TestNumberModel` (4 cells × 7 bad values + max admit) |
| 13.14.2 closed extensions (reverse-DNS keys, AX values, no nested dup) | both entries via `clonebundle` owner | `TestExtensionsClosed` (admission + 6 build + 4 decode rows) |
| 13.14.2 strict framing (dup/trailing/non-object/surrogate/UTF-8) | `decodeStrictObject` | `TestReportFraming` (5 report + 1 row rows) |
| 13.14.2 build UTF-8 pre-marshal gate (every string-valued member refuses invalid UTF-8; decode refuses at the delegated strict frame before dispatch) | `BuildFidelityReport` string admissions; `DecodeFidelityReport` via `decodeStrictObject` | `TestUTF8RefusalReflectionGrid` (33 reflection-derived members × 5 invalid-UTF-8 classes × both entries: 10 report scalars + required key/values + forbid + 12 row members + 3 breakdown/byte keys + 2 manifests + attestations + 2 report extension key/value; build pins member-specific literals, decode pins the frame literal) + `TestRegistryPredicatesRefuseInvalidUTF8` (5 predicates × 5 classes) + `TestBuildUTF8ExtensionsNestedDepth` + `TestBuildUTF8` (kept: original 4 sites) |
| 13.14.2 determinism (identical inputs, identical bytes) | `BuildFidelityReport` | `TestReportDeterminism` (10 builds byte-identical) |
| 13.14.2 purity (no durable write) | both entries | `TestEntriesArePure` (input bytes untouched) |
| 13.14.1 owner vocabularies equal the spec lists | `clonebundle` tables iterated by production | `TestOwnerVocabulariesMatchSpec` (26/8/9 cross-checks + own registry counts) |

## Durability

This package holds no local state and performs no durable write:
sealing is pure bytes (`json.Marshal` + JCS + SHA-256 over memory),
so crash semantics are vacuous. The durability proof is
determinism plus purity: `TestReportDeterminism` (ten builds
byte-identical) and `TestEntriesArePure` (decode never mutates its
input; every build output decodes clean).

## Mutants

`testdata/mutant_harness.py` ships 70 narrowing rows plus the
harmless `C-doc-comment` SURVIVED control. Every narrowing weakens
one gate to admit exactly one member of the class it must reject,
and every killer drives a production entry. Shared gates (one
implementation, two call sites: the vocabulary predicates, the
reason/canonical/order/count helpers, the breakdown sums, the
row-count gate, the strict frame) carry one plant each with a
dual-entry killer, exactly like the prior leaf's `N-strict-*` rows;
rules implemented twice (build + decode) carry one plant per side.
`N-disposition-route` and `N-reason-core-route` are the
token-preserving attacks: the searched-for token still matches
while the decision changes, and the killers execute the behavioral
oracle suites through both entries. `N-rowcount-max` and
`N-rowcount-min` are the factored-bound narrowings (the exact edges
are pinned at the factored gate; entry reachability by
`TestRowCountEdges`). `N-record-synth-canonical` is labelled
single-member class. The eight `N-build-utf8-*` rows narrow the
build pre-marshal UTF-8 sites (the three rev2 reviewer sites plus
one per shape and the remaining explicit sites: key, explanation,
reason arm, both extension call sites), each killed by
`TestUTF8RefusalReflectionGrid` run alone; the decode side needs no
plant because invalid bytes trip the shared strict frame before any
member gate runs. The battery runs twice with one raw log per
plant per run carrying the subprocess exit.

## Stated bounds (sibling scope, not waived)

- Cross-artifact row coverage ("every Capture Manifest item and
  Canonical Event occurs in exactly one non-synthesized row") needs
  the manifests as inputs; no entry in this leaf takes them.
- Source-evidence resolution (rows trace to captured bytes) needs
  the Capture Manifest; only ID presence/shape is gated here.
- `byte_counts` value reconciliation is an EXPLICIT STATED BOUND
  (AC 39 of 40 driven + 1 bound): rows carry no byte measure and
  the spec defines no byte basis for canonical-event or synthesized
  rows, so only the exact seven-disposition key set and uint53
  values are gated (witnessed by
  `TestByteCountsReconciliationIsAStatedBoundUntilSpecDefinesByteBasis`
  admission of differing bytes over identical rows). Owner: spec
  clarification (agent-session-manager-spec) + the §13.14.1
  capture-manifest leaf.
- The core-derived booleans admit any value except the pinned
  archive rule (both target booleans false for archive).
- Per-row target evidence (`target_locator`, staged/live arrays) in
  archive scope is admitted as shaped; no scope-based row refusal is
  pinned in the spec text.
- `forbid_reasons` admits literal bounded strings (I3), not
  reason-vocabulary members.
- Admission at exactly 1,000,000 rows is pinned at the factored
  `checkRowCount` gate plus entry reachability rows, not by sealing
  a million-row report (see `TestRowCountEdges` and
  `TestCheckRowCountEdges`).
- Sections 13.14.3-13.14.5 and the Projection Plan, Read-Back,
  Validation Report, Migration Checkpoint, and Lineage Receipt
  contracts are sibling scope: this leaf plans no projection,
  stages no target, and carries only identity/digest references.
