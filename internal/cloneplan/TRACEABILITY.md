# Projection Clause Coverage (TASK-260830-1jmmqn)

Owner: `internal/cloneplan`. Authority:
`internal/specdoc/SPEC.v0.7.0.md`, Section 13.14.2 "Fidelity,
projection, and lineage" (line 10548+): the Projection Plan 1.0.0
table, the component schemas that follow it
(ProjectionItemMapping, ProjectionTargetOperation,
ExpectedTargetResource, SynthesizedProjectionEvent,
TransactionPlan, ReadBackPlan, ResumeProjectionPlan,
RollbackPlan, ContractRequirement), and the Clone Projected Object
Manifest 1.0.0 paragraph. Schema URNs come from the Section 1
registry table (lines 168-169). Section 13.14.4 carries no
constraint on these shapes. This is the SECOND leaf's clause record
for the fidelity-projection-and-validation-contracts story; record
clause-to-test bindings here. The `internal/traceability` registry
edit belongs to the Story's final leaf, not this one. Planning
semantics (strategy/profile choice, checkpoint message authority,
inactive tools/instructions, token metadata) belong to the sibling
planning leaf (TASK-260924-3n78rv), not this one.

## Why this package exists

Section 13.14.2 states the Projection Plan 1.0.0 closed table, the
nine component records, and the Clone Projected Object Manifest
1.0.0 closed shape with its branch-exact entries. No landed package
owns them: the `canonicaljson` closed-shape registry recognizes
`urn:ax:schema:projection-plan` and
`urn:ax:schema:clone-projected-object-manifest` only to refuse them
(`rejectUnsupportedImmutableObjectShape`, pinned by that table's
census), so this package is the validating owner. The sibling
fidelity leaf owns dispositions, reasons, and the Fidelity Report;
where the plan references sibling contracts (Migration Checkpoint,
Read-Back, Validation Report, Lineage Receipt, Bundle Manifest)
only the identity or digest field is carried.

## Landed-owner reuse

| Scope | Owner | Use |
|---|---|---|
| Strict JSON framing | `internal/environ` | `DecodeStrictObject` at `decodeStrictObject`: the plan, the manifest, and every nested shape refuse lone-surrogate escapes, duplicate members, non-objects, and trailing data before any member is read |
| Character string measure | `internal/environ` | `StringLength` at every bound, both entries |
| String/uint53/digest/UUID bounds | `internal/environ` | `CheckStringBounds`, `CheckUint53Bounds`, `CheckDigest`, `CheckUUIDv7` at every decode site |
| Sorted-unique arrays | `internal/environ` | `CheckSortedUniqueStrings`, `CheckSortedUniqueDigests` at every decode site |
| SemVer grammar | `internal/environ` | `CheckSemver` behind the contract version gate, both entries |
| Environment Tuples | `internal/sessadapter` | `DecodeTuple` at all three tuple members, both entries |
| Resource Limits | `internal/sessadapter` | `DecodeResourceLimits` at `resource_limits`, both entries |
| Canonical bytes, digests, UUIDs | `internal/canonicaljson`, `internal/scalar` | `Canonicalize`, `SHA256Digest`, `ParseDigest`, `ParseUUIDv7` |
| Closed extension values | `internal/clonebundle` | `CheckExtensionsClosed` at every extension member on decode; `EncodeExtensions` on build |
| Omit-self digest | `internal/clonebundle` | `OmitSelfDigest` behind `verifySelfDigest` |
| Workspace Binding | `internal/clonebundle` | `DecodeWorkspaceBinding` at `target_workspace`, both entries |
| Capture classes (9) | `internal/clonebundle` | `ValidCaptureClass` for `security_exclusions` |
| Dispositions (7), reasons (19+ext) | `internal/clonefidelity` | `ValidDisposition` for `expected_disposition` and policy values; `ValidReasonCode` for mapping reasons |

No behavior in any reused package changed: this leaf adds a new
package only and edits no landed file (importer grid below).

## Interpretation record (decided, pinned)

These readings are taken where the pinned text underdetermines the
gate. Each is pinned by a named test, not prose.

| # | Reading | Test |
|---|---|---|
| I1 | `item_mappings` are sorted unique by source item key: the plan prose binds "ordered item mappings", and the key is the only candidate order | `TestRowOrderSweep` (unsorted + duplicate refuse; ordered admits) |
| I2 | `expected_resources` sort key is the (sequence, key) pair, strictly increasing | `TestRowOrderSweep` (sequence regression, key regression, and full duplicate refuse; both admit shapes admit) |
| I3 | `required_dispositions` is the exact request policy: a non-empty map of classes to sorted unique 1..7 dispositions; key closure is the request owner's stated bound, mirrored | `TestRequiredDispositions` |
| I4 | `forbid_reasons` admits literal bounded strings: the table row types them as strings, not reason codes | `TestForbidReasonsAdmitLiteralStrings` (`frobnicate` seals and decodes) |
| I5 | Operation dependencies name existing strictly-lower sequences: an edge to an absent sequence is not a DAG over the plan operations | `TestDAGRefusalLiterals` (dangling), `TestDAGWholeDomainCensus` (oracle) |
| I6 | Expected-resource branch rule is the blob-ID arm only: blob rows carry non-null `expected_blob_id`, directory rows null; mode keeps its nullable type on both branches | `TestResourceNullabilityGrid` (12 cells, 6 admit) |
| I7 | Manifest directory `mode` keeps the nullable uint32[0..4095] type; the branch difference is the member set (7 vs 4) | `TestEntryNullabilityGrid` (48 cells), `TestEntryGridNeverPanics` |
| I8 | Manifest entries reuse the plan scalar types (sequence uint53>0, key string[1..512]) because entries partition plan resources | `TestEntryStringEdges`, `TestNumberModel` |
| I9 | `ContractRequirement` carries no extensions member: the pinned text states exactly contract_id and version | `TestContractNoExtensions` |
| I10 | Mapping rows carry no disposition/reason/canonical coupling: the pinned mapping text states none (unlike the fidelity row) | `TestNoDispositionCouplingOnMappings` (12 admit cells) |
| I11 | `total_bytes` admits any uint53: the pinned text states no derivation rule | `TestTotalBytesIsAStatedBoundUntilSpecDefinesDerivation` |
| I12 | Synthesized array position IS the insertion sequence: no order gate is expressible without a sequence member | stated bound below |
| I13 | `canonical_event_ids` are ordered unique (session order), not sorted: descending digests admit, duplicates refuse | `TestSortedUniqueSweep` (canonical-ordered-not-sorted) |

## Clause map

| Pinned clause | Production entry | Test |
|---|---|---|
| 13.14.2 four plan strategies admitted exactly; archive_only refused | `ValidPlanStrategy`, `BuildProjectionPlan`, `DecodeProjectionPlan` | `TestPlanStrategyOracle` (4 admit + predicate/build/decode; 10 refusals incl. archive_only) |
| 13.14.2 four plan profiles admitted exactly; archive_only refused | `ValidPlanProfile`, both plan entries | `TestPlanProfileOracle` (4 admit; 9 refusals incl. archive_only) |
| 13.14.2 four operation actions admitted exactly | `ValidOperationAction`, both plan entries | `TestOperationActionOracle` (4 admit; 9 refusals) |
| 13.14.2 three synthesized purposes admitted exactly | `ValidSynthesizedPurpose`, both plan entries | `TestSynthesizedPurposeOracle` (3 admit; 8 refusals) |
| 13.14.2 two resource kinds admitted exactly, both shapes | `ValidResourceKind`, all four entries | `TestResourceKindOracle` (2 admit; 7 refusals x 2 shapes x 2 entries) |
| 13.14.2 seven expected dispositions admitted exactly | `clonefidelity.ValidDisposition` via both plan entries | `TestExpectedDispositionOracle` (7 admit; 7 refusals x 2 entries) |
| 13.14.2 mapping reasons are core + reverse-DNS | `clonefidelity.ValidReasonCode` via both plan entries | `TestMappingReasonVocabulary` (20 admit; 7 refusals x 2 entries) |
| 13.14.2 Projection Plan exact 36-member set | `DecodeProjectionPlan` | `TestMemberCensusExactSets` (reflection-derived census vs sealed, both directions), `TestMemberCensusUnknownMissing`, `TestMemberCensusMiscased` (6 case-variant classes), `TestMemberCensusWrongTypes`, `TestMemberCensusDuplicated` (every member), `TestCensusShapeInventory`, `TestMemberCensusNoLiteralList` (derivation guard) |
| 13.14.2 manifest exact 10-member set; blob 7 / directory 4 branch sets | `DecodeProjectedObjectManifest` | same census tests (13 shapes: 36/10/6/5/6/4/4/5/4/4/2/7/4; census reflected from the input types, synthesized members by sealed set difference) |
| 13.14.2 schema/version literals | both decode entries | `TestSchemaLiterals` (plan 4+4, manifest 3+1 refusal literals) + `TestPlanConstantCopyRegression` (copy refuses, 4 subtests) + `TestPlanConstantDerivedValues` + `TestPlanConstantGeneratedSweep` (full edit-distance-1 + case + random + non-string JSON per literal) + `TestClosedConstantGateShape` (single-literal AST pin) |
| 13.14.2 `projection_plan_id` is the JCS digest omitting only itself | `BuildProjectionPlan`, `verifySelfDigest` | `TestPlanIdentityIndependent` (test-side JCS recompute agrees; omitting strategy disagrees; tampered claim and non-digest refuse) |
| 13.14.2 `projected_object_manifest_id` under the omission rule | `BuildProjectedObjectManifest`, `verifySelfDigest` | `TestManifestIdentityIndependent` (same shape) |
| 13.14.2 dependencies are lower sequences and form a DAG | `checkOperationOrder`, `checkOperationDAG` (both entries) | `TestDAGWholeDomainCensus` (every graph through 4 ops: 2+16+512+65536 vs oracle, both entries; every edge class at every 5-op position), `TestDAGValidEnumerationFive` (1024 valid DAGs admit), `TestDAGSingleEdgeLocality` (35 toggles), `TestDAGLocalityStructure` (per-edge-local AST pin), `TestDAGRefusalLiterals`, `TestDAGNonContiguousValid` |
| 13.14.2 blob/directory nullability branch-exact (resources) | `checkResourceBranch` (both entries) | `TestResourceNullabilityGrid` (12 cells vs oracle, both entries), `TestResourceBranchLiterals` |
| 13.14.2 blob/directory branch-exact entries | `checkEntryBranch` (build), kind dispatch (decode) | `TestEntryNullabilityGrid` (48 cells: kind x 2^3 presence x mode, structural literals, both entries), `TestEntryGridNeverPanics` (same grid under recovery), `TestEntryBranchLiterals` |
| 13.14.2 every sorted-unique array refuses unsorted/duplicated | both entries | `TestSortedUniqueSweep` (8 arrays x 2 shapes x 2 entries + ordered-not-sorted admission) |
| 13.14.2 cross-row order (mappings/resources/entries/contracts) | shared order gates (both entries) | `TestRowOrderSweep` (build violations + decode documents per gate) |
| 13.14.2 string bounds at edge/edge+1, character measure | both entries via `environ.StringLength` | `TestStringBoundsEdges` (5 members x edges x 2 entries), `TestMultibyteMeasure` (512/513 `é`), `TestManifestStringEdges`, `TestEntryStringEdges` |
| 13.14.2 cardinality bounds at edge/edge+1 | factored count gates + both entries | `TestCountEdges` (entry reachability + 64/65 + 129 + 65536/65537) + `TestFactoredCountGates` (6 gates x min/max/min-1/max+1) + `TestMappingCountMillionEntryLevel` (1M admit + 1M+1 refuse, both entries) |
| 13.14.2 Transaction/ReadBack/Resume/Rollback constants refused on any other value | build + decode per component | `TestPlanConstants` (every member incl. near-miss X2 values, every modes shape, every flag, bounded both booleans + 5 non-boolean refusals) + `TestPlanConstantCopyRegression` (copy refuses at every string gate x both entries, the rev3 plant value) + `TestPlanConstantDerivedValues` (fast derived set per gate) + `TestPlanConstantGeneratedSweep` (every edit-distance-1 neighbor + every case class + fixed-seed sample + empty/whitespace + non-string JSON per gate) + `TestPlanConstantBooleans` (other boolean refuses; bounded admits both) + `TestPlanConstantNonBooleanJSON` (5 non-boolean types x every boolean member) + `TestClosedConstantGateShape` (AST pin: gate list reflection-derived, each gate one spec literal, no second value/switch/fold) |
| 13.14.2 ContractRequirement (contract_id, SemVer; no extensions) | build + decode | `TestContractSemver` (5 admit + 7 refusals x 2 entries), `TestContractNoExtensions` |
| 13.14.2 required_dispositions exact policy from request | build + decode policy gates | `TestRequiredDispositions` (empty + 5 bad shapes x 2 entries + full 7-set admission) |
| 13.14.2 security_exclusions sorted unique capture-class[0..9] | build + decode | `TestSecurityExclusions` (9 admit, 10-count, vocab refusal x 2 entries) + `TestSortedUniqueSweep` |
| 13.14.2 scalar gates (digests, UUIDv7, tuples, workspace, limits) | both entries via landed owners | `TestScalarGates` (14 decode + 11 build rows), `TestManifestScalarGates` (3+3 rows) |
| 13.14.2 AX number model (no fractions/exponents/strings/≥2^53) | decode via `environ.CheckUint53Bounds`; build overflow checks | `TestNumberModel` (6 members x 6 bad values + max admit + 6 build overflows) |
| 13.14.2 closed extensions (reverse-DNS keys, AX values, no nested dup) | both entries via `clonebundle` owner | `TestExtensionsClosed` (admission + build/decode rows + nested dup) |
| 13.14.2 strict framing (dup/trailing/non-object/surrogate/UTF-8) | `decodeStrictObject` | `TestFraming` (9 plan + 3 manifest + 1 surrogate rows) |
| 13.14.2 build UTF-8 pre-marshal gate (every string-valued member refuses invalid UTF-8; decode refuses at the delegated strict frame before dispatch) | build string admissions; decode via `decodeStrictObject` | `TestUTF8RefusalReflectionGrid` (72 reflection-derived members x 5 invalid-UTF-8 classes x both entries) + `TestRegistryPredicatesRefuseInvalidUTF8` (5 predicates x 5 classes) |
| 13.14.2 build-side member census (every input member zeroed) | both build entries | `TestBuildZeroValueCensus` (33 plan cells, 6 admit), `TestBuildZeroValueRows` (mapping/operation/resource/event/contract), `TestBuildZeroValueNestedPlans` (transaction/read-back/resume/rollback members), `TestBuildZeroValueManifest` (top/entry); every table asserts its cell count against the reflected field count |
| 13.14.2 determinism (identical inputs, identical bytes) | both build entries | `TestDeterminism` (10 builds byte-identical per entry) |
| 13.14.2 purity (no durable write) | all four entries | `TestEntriesArePure` (decode never mutates input; every build output decodes clean) |

## Durability

This package holds no local state and performs no durable write:
sealing is pure bytes (`json.Marshal` + JCS + SHA-256 over memory),
so crash semantics are vacuous. The durability proof is
determinism plus purity: `TestDeterminism` (ten builds
byte-identical per entry) and `TestEntriesArePure` (decode never
mutates its input; every build output decodes clean).

## Mutants

`testdata/mutant_harness.py` ships 143 narrowing rows, one
token-preserving arm-delete row (`N-dag-locality-evasion`, marked
NOT a narrowing in its note), and the harmless `C-doc-comment`
SURVIVED control: 145 rows in total. Every narrowing weakens one
gate to admit exactly one member of the class it must reject, and
every killer drives a production entry alone. Shared gates (one
implementation, two call sites: the vocabulary predicates, the
strict frame, the member census helpers, the order helpers, the
DAG gate, the branch gates, the factored count gates, the modes
constant, the build UTF-8 helpers) carry one plant each with a
killer that drives the production entries; rules implemented twice
(build + decode) carry one plant per side. `N-strategy-route` and
`N-action-route` are the token-preserving attacks: the searched-for
token still matches while the decision changes, and the killers
execute the behavioral oracle suites through both entries.
Single-member-class rows label gates whose reject class holds
exactly one member. The rev4 rows are 13 copy-admitting narrowings
(`N-copy-*`: every closed string gate plus the four
schema/version literals) killed by the named copy-regression
subtests, and 7 null-admitting decode narrowings (`N-null-*`:
every closed boolean gate) killed by the named non-boolean
subtests; each killer carries sibling probes (the literal admits,
a derived neighbor refuses) so the kill lands on the admitted
value only. The battery runs twice with one raw log per plant per
run carrying the subprocess exit.

## Importer outcome grid

This leaf adds `internal/cloneplan` (new files only) and edits no
landed file: the six reused owner packages are byte-identical
between base and candidate (171 files, hash-compared), and their
suites are green on both trees, so no input class moved. The grid
below is keyed (package, entry, input): one row per reused owner
entry with the input classes flowing through this leaf's call
sites, plus this leaf's own new entries. Base is trunk HEAD
(`547ea28`); candidate is the uncommitted tree.

| Package | Entry | Input classes through this leaf | Base | Candidate | Moved |
|---|---|---|---|---|---|
| `internal/environ` | `DecodeStrictObject` | valid object docs; dup-member / trailing-data / non-object / surrogate-escape docs | green | green | none |
| `internal/environ` | `StringLength` | ASCII + multibyte strings at every bound | green | green | none |
| `internal/environ` | `CheckStringBounds` | in-bound; over-max / under-min; non-string JSON | green | green | none |
| `internal/environ` | `CheckUint53Bounds` | in-bound uint53; fraction / exponent / string / 2^53 | green | green | none |
| `internal/environ` | `CheckDigest` | valid sha256; malformed digests | green | green | none |
| `internal/environ` | `CheckUUIDv7` | valid v7; v4 / malformed ids | green | green | none |
| `internal/environ` | `CheckSortedUniqueStrings` | sorted-unique; unsorted / duplicated / over-count | green | green | none |
| `internal/environ` | `CheckSortedUniqueDigests` | sorted-unique; unsorted / duplicated | green | green | none |
| `internal/environ` | `CheckSemver` | valid SemVer; non-SemVer versions | green | green | none |
| `internal/scalar` | `ParseDigest` | valid sha256; malformed digests | green | green | none |
| `internal/scalar` | `ParseUUIDv7` | valid v7; malformed ids | green | green | none |
| `internal/scalar` | `SHA256Digest` | canonical omit-self bytes | green | green | none |
| `internal/canonicaljson` | `Canonicalize` | plan / manifest / row objects | green | green | none |
| `internal/clonebundle` | `EncodeExtensions` | closed maps; bad keys / values | green | green | none |
| `internal/clonebundle` | `CheckExtensionsClosed` | closed; open / nested-dup | green | green | none |
| `internal/clonebundle` | `OmitSelfDigest` | member maps + self field; tampered claims | green | green | none |
| `internal/clonebundle` | `DecodeWorkspaceBinding` | valid bindings; broken bindings | green | green | none |
| `internal/clonebundle` | `ValidCaptureClass` | 9 classes; unknown / 10th | green | green | none |
| `internal/sessadapter` | `DecodeTuple` | valid tuples; broken tuples | green | green | none |
| `internal/sessadapter` | `DecodeResourceLimits` | valid limits; broken limits | green | green | none |
| `internal/clonefidelity` | `ValidDisposition` | 7 dispositions; unknown | green | green | none |
| `internal/clonefidelity` | `ValidReasonCode` | core + reverse-DNS; bad codes | green | green | none |
| `internal/cloneplan` | `BuildProjectionPlan` | valid plan; every invalid class in the clause map | n/a (new) | green | none (new surface, no importer yet) |
| `internal/cloneplan` | `DecodeProjectionPlan` | sealed plan; mutated / hand-built docs | n/a (new) | green | none |
| `internal/cloneplan` | `BuildProjectedObjectManifest` | valid manifest; every invalid class | n/a (new) | green | none |
| `internal/cloneplan` | `DecodeProjectedObjectManifest` | sealed manifest; mutated docs | n/a (new) | green | none |
| `internal/cloneplan` | 5 vocabulary predicates | in-vocabulary; out-of-vocabulary incl. invalid UTF-8 | n/a (new) | green | none |

## Stated bounds (sibling scope or spec-silent, not waived)

- `canonical_event_ids` equality with Canonical Session order needs
  the session as an input; only ordered-unique shape is gated here.
- Item-mapping coverage ("one for every captured/canonical item")
  needs the manifests as inputs; only the row shape, count, and key
  order are gated here.
- Expected-resource and manifest-entry partitioning against plan
  operations needs the plan as a cross-input; only intra-shape
  rules are gated here.
- Synthesized insertion anchors against canonical IDs need the
  Canonical Session; only digest-or-null shape is gated here.
- `total_bytes` reconciliation is an EXPLICIT STATED BOUND: the
  pinned text states no derivation rule, so any uint53 admits
  (witnessed by
  `TestTotalBytesIsAStatedBoundUntilSpecDefinesDerivation`).
  Owner: spec clarification (agent-session-manager-spec).
- Synthesized array order: array position IS the insertion
  sequence; no order gate is expressible (I12).
- `required_dispositions` key closure is the request owner's bound
  (`sessadapter` checkRequiredDispositions), mirrored exactly (I3).
- `forbid_reasons` admits literal bounded strings (I4), not
  reason-vocabulary members.
- Five-operation DAG coverage is structural, not exhaustive: every
  graph through four operations is enumerated against the oracle,
  and at five operations every edge class is driven at every
  position over the per-edge-local rule pinned by
  `TestDAGLocalityStructure` and `TestDAGSingleEdgeLocality`. The
  full 33M-graph five-operation enumeration is infeasible and is
  not run.
- The no-literal-list guard trips on census-scale composites (at
  least 10 distinct member names at 40%+ member purity); smaller
  literal fragments are below its line by design.
- JSON-level member probes (missing/extra/duplicated/wrong-type/
  case-variant) run at the decode entries; Go inputs cannot express
  an extra, duplicated, or wrong-typed member, so the build-side
  member census is the per-field zero-value sweep
  (`TestBuildZeroValue*`, reflection-count-checked).
- Planning semantics (strategy/profile choice, checkpoint message
  authority, inactive tools/instructions, token metadata) belong to
  the sibling planning leaf (TASK-260924-3n78rv): this leaf gates
  shapes only.
- Sections 13.14.3-13.14.5 and the Read-Back, Validation Report,
  Migration Checkpoint, and Lineage Receipt contracts are sibling
  scope.
