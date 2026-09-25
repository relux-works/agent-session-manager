# TASK-260830-1jmmqn conformance matrix: gate x entry census (rev4)

Legend: B = `BuildProjectionPlan`, D = `DecodeProjectionPlan`,
MB = `BuildProjectedObjectManifest`, MD =
`DecodeProjectedObjectManifest`, P = predicate entry. Every cell
names the killer test(s) run ALONE plus the killed narrowing
mutant(s). Shared = one implementation, all call sites. Axes:
member census reflection-derived from the Go input types (13
shapes, 97 members; 4 documented irregular names; synthesized
members by sealed set difference) with a no-literal-list guard;
identities (independent JCS recompute); DAG (every graph through 4
ops + every edge class at every 5-op position + structural
locality); nullability (full kind x 2^3 presence x mode grids,
both entries); bounds (edge/edge+1, 1M mappings sealed at both
entries); constants (reflection-derived gate list; full
edit-distance-1 + case-class + random + empty/whitespace sweep per
literal at both entries; non-string/non-boolean JSON grids at
decode; single-literal AST guard).

Rev4 answers the rev3 finding
`plan-constant-second-value-uncounted`: the hand-picked constant
probes are replaced by construction (generated invalid values per
gate plus a structural single-literal guard), with a named
`copy` regression at every string gate and a null regression at
every boolean gate. No production line changed semantics.

## Vocabularies

| Gate | B | D | MB | MD | P | Killer | Narrowing |
|---|---|---|---|---|---|---|---|
| 4 plan strategies; archive_only refused | x | x | | | x | TestPlanStrategyOracle | N-strategy-vocab, N-strategy-route |
| 4 plan profiles; archive_only refused | x | x | | | x | TestPlanProfileOracle | N-profile-vocab |
| 4 operation actions | x | x | | | x | TestOperationActionOracle | N-action-vocab, N-action-route |
| 3 synthesized purposes | x | x | | | x | TestSynthesizedPurposeOracle | N-purpose-vocab |
| 2 resource kinds, both shapes | x | x | x | x | x | TestResourceKindOracle | N-kind-vocab, N-kind-route |
| 7 expected dispositions | x | x | | | (sib) | TestExpectedDispositionOracle | (sibling-owned;698 cells via entries) |
| core + reverse-DNS mapping reasons | x | x | | | (sib) | TestMappingReasonVocabulary | (sibling-owned;40 cells via entries) |

## Shape, framing, identity

| Gate | B | D | MB | MD | Killer | Narrowing |
|---|---|---|---|---|---|---|
| Strict frame (dup/trailing/non-object/surrogate/UTF-8) | | x | | x | TestFraming, TestMemberCensusDuplicated | N-frame-dup (shared) |
| Unknown member refused, all 13 shapes | | x | | x | TestMemberCensusUnknownMissing | N-unknown-shared |
| Missing member refused, all 13 shapes | | x | | x | TestMemberCensusUnknownMissing | N-missing-shared |
| Miscased member refused, all 97 members x 6 variant classes | | x | | x | TestMemberCensusMiscased | N-case-title-strategy, N-case-upper-strategy |
| Wrong JSON type refused, all 97 members x 5-6 types | | x | | x | TestMemberCensusWrongTypes | (covered per member by value narrowings) |
| No string-literal member list in tests | | | | | TestMemberCensusNoLiteralList | N-guard-literal-list |
| Build zero-value census, all input members incl. nested plans | x | | x | | TestBuildZeroValueCensus/Rows/NestedPlans/Manifest | (covered per member by build narrowings) |
| Plan schema/version literals | | x | | | TestSchemaLiterals, TestPlanConstantCopyRegression, TestPlanConstantDerivedValues, TestPlanConstantGeneratedSweep, TestClosedConstantGateShape | N-schema-plan, N-version-plan, N-copy-schema-plan, N-copy-version-plan |
| Manifest schema/version literals | | | | x | TestSchemaLiterals, TestPlanConstantCopyRegression, TestPlanConstantDerivedValues, TestPlanConstantGeneratedSweep, TestClosedConstantGateShape | N-schema-manifest, N-version-manifest, N-copy-schema-manifest, N-copy-version-manifest |
| Plan omit-self identity (independent recompute) | x | x | | | TestPlanIdentityIndependent | N-digest-plan |
| Manifest omit-self identity (independent recompute) | | | x | x | TestManifestIdentityIndependent | N-digest-manifest |
| Build UTF-8 pre-marshal, 72 reflected members x 5 classes | x | | x | | TestUTF8RefusalReflectionGrid | N-utf8-* (13: one per shape + 2 shared) |
| Decode UTF-8 at frame, same 72 members x 5 classes | | x | | x | TestUTF8RefusalReflectionGrid | N-frame-dup (frame path) |
| Registry predicates refuse invalid UTF-8 | | | | | TestRegistryPredicatesRefuseInvalidUTF8 | (boolean predicates; no error channel) |
| Closed extensions (keys, values, nested dup) | x | x | x | x | TestExtensionsClosed | (owner-gated; entry reachability both sides) |
| Determinism / purity | x | x | x | x | TestDeterminism, TestEntriesArePure, TestPlanRoundTrip | C-doc-comment (control SURVIVED) |

## Order, DAG, branches, counts

| Gate | B | D | MB | MD | Killer | Narrowing |
|---|---|---|---|---|---|---|
| Mappings sorted-unique by key | x | x | | | TestRowOrderSweep | N-mapping-order (shared) |
| Operations strictly increasing sequence | x | x | | | TestDAGRefusalLiterals | N-operation-order (shared) |
| Dependencies strictly lower (self/higher refuse) | x | x | | | TestDAGRefusalLiterals, TestDAGWholeDomainCensus | N-dag-lower (shared), N-dagfour-self, N-dagfive-self |
| Dependencies exist (dangling refuses) | x | x | | | TestDAGRefusalLiterals, TestDAGWholeDomainCensus | N-dangling-pair (shared) |
| Locality: per-edge rule, no global-graph state | x | x | | | TestDAGLocalityStructure (static) + TestDAGSingleEdgeLocality (35 toggles) | N-dag-locality-evasion (token-preserving arm-delete; killer is the behavioral census) |
| Every graph thru 4 ops + every 5-op valid DAG admits | x | x | | | TestDAGWholeDomainCensus, TestDAGValidEnumerationFive, TestDAGNonContiguousValid | (admission direction; refusals above) |
| Resource branch (blob/non-null, dir/null) | x | x | | | TestResourceNullabilityGrid, TestResourceBranchLiterals | N-resource-branch-blob/directory (shared) |
| Entry branch build (6 arms + combo) | | | x | | TestEntryBranchLiterals, TestEntryNullabilityGrid, TestEntryGridNeverPanics | N-entry-branch-* (6, single-member) + N-entry-branch-combo |
| Entry branch decode (kind dispatch) | | | | x | TestEntryNullabilityGrid | N-entry-decode-dirbyte |
| Resources sorted by (sequence,key) | x | x | | | TestRowOrderSweep | N-resource-order (shared) |
| Entries sorted by (sequence,key) | | | x | x | TestRowOrderSweep | N-entry-order (shared) |
| Contracts sorted by ID | x | x | | | TestRowOrderSweep | N-contract-order (shared) |
| Mapping count [1..1000000] at both entries | x | x | | | TestCountEdges, TestMappingCountMillionEntryLevel, TestFactoredCountGates | N-count-mapping-min/max (shared) |
| Operation count [1..65536] | x | x | | | TestCountEdges, TestFactoredCountGates | N-count-operation-min/max (shared) |
| Resource count [0..65536] | x | x | | | TestFactoredCountGates | N-count-resource-max (shared) |
| Event count [0..65536] | x | x | | | TestFactoredCountGates | N-count-event-max (shared) |
| Contract count [1..64] | x | x | | | TestCountEdges, TestFactoredCountGates | N-count-contract-min/max (shared) |
| Entry count [0..65536] | | | x | x | TestFactoredCountGates | N-count-entry-max (shared) |
| canonical_event_ids ordered-unique (unsorted admits) | x | x | | | TestSortedUniqueSweep | N-canonical-dup-build/decode |

## Scalars, arrays, policy, constants

| Gate | B | D | MB | MD | Killer | Narrowing |
|---|---|---|---|---|---|---|
| String bounds edge/edge+1 (5 plan + manifest + entry) | x | x | x | x | TestStringBoundsEdges, TestManifestStringEdges, TestEntryStringEdges | N-decode-native-min, N-decode-contractid-min, N-build-native-min, N-build-contractid-min |
| Character measure (512/513 multibyte) | x | x | | | TestMultibyteMeasure | (owner measure; entry reachability) |
| depends_on sorted-unique uint53 | x | x | | | TestSortedUniqueSweep | N-depends-dup-build/decode |
| Sorted-unique string arrays (keys/reasons/forbid/caps) | x | x | | | TestSortedUniqueSweep | N-sorted-order-build (shared) |
| Reason order (build) | x | | | | TestSortedUniqueSweep | N-reason-order-build |
| Exclusion order (build) | x | | | | TestSortedUniqueSweep | N-exclusion-order-build |
| Mode uint32[0..4095]/null | x | x | x | x | TestNumberModel | N-decode-mode-max, N-build-mode-max (shared) |
| Sequence uint53>0 (ops/resources/entries) | x | x | x | x | TestNumberModel, TestBuildZeroValueManifest | N-decode-sequence-min, N-build-seq-min, N-build-opseq-min, N-build-entryseq-min |
| total_bytes/byte_count uint53 | | | x | x | TestNumberModel | N-build-total-max, N-build-bytecount-max |
| AX number model (fraction/exponent/string/2^53) | | x | | x | TestNumberModel | (owner gate; 36 refusal cells via entries) |
| Digests/UUIDv7/tuples/workspace/limits | x | x | x | x | TestScalarGates, TestManifestScalarGates | N-uuid-plan-build, N-digest-plan-build, N-uuid-manifest-build, N-digest-manifest-build |
| required_dispositions non-empty | x | x | | | TestRequiredDispositions | N-policy-empty-build/decode |
| Policy values 1..7 sorted-unique dispositions | x | x | | | TestRequiredDispositions | N-policy-width-build/decode, N-policy-order-build/decode |
| forbid_reasons [0..128] | x | x | | | TestCountEdges | N-forbid-129-build/decode |
| Mapping reasons [0..128] | x | x | | | TestCountEdges | N-reason-129-build/decode |
| Exclusion count [0..9] | x | x | | | TestSecurityExclusions | N-exclusion-count-build/decode |
| Exclusion 9-class vocabulary | x | x | | | TestSecurityExclusions | (owner vocabulary; entry cells) |
| Capabilities [1..64] | x | x | | | TestCountEdges | N-decode-cap-min, N-build-cap-min |
| materialization_intent=clone | x | x | | | TestPlanConstants, TestPlanConstantCopyRegression, TestPlanConstantDerivedValues, TestPlanConstantGeneratedSweep, TestClosedConstantGateShape | N-intent-build/decode, N-copy-intent-build/decode |
| target_collision_policy=must_be_absent | x | x | | | TestPlanConstants, TestPlanConstantCopyRegression, TestPlanConstantDerivedValues, TestPlanConstantGeneratedSweep, TestClosedConstantGateShape | N-policy-build/decode, N-copy-policy-build/decode |
| activation=dormant_validated | x | x | | | TestPlanConstants, TestPlanConstantCopyRegression, TestPlanConstantDerivedValues, TestPlanConstantGeneratedSweep, TestClosedConstantGateShape | N-activation-build/decode, N-copy-activation-build/decode |
| retain_through=live_validated | x | x | | | TestPlanConstants, TestPlanConstantCopyRegression, TestPlanConstantDerivedValues, TestPlanConstantGeneratedSweep, TestClosedConstantGateShape | N-retain-build/decode, N-copy-retain-build/decode |
| modes=[staged,live] | x | x | | | TestPlanConstants, TestPlanConstantCopyRegression, TestPlanConstantDerivedValues, TestPlanConstantGeneratedSweep, TestClosedConstantGateShape | N-modes-shared, N-copy-modes-shared |
| 7 boolean constants (=true/=false) | x | x | | | TestPlanConstants, TestPlanConstantBooleans, TestPlanConstantNonBooleanJSON, TestClosedConstantGateShape | N-reqident/reqws/reqsem/opens/blank/required/forbidden-build/decode (14, single-member), N-null-reqident/reqws/reqsem/opens/blank/required/forbidden-decode (7, null-only) |
| bounded flag is boolean (both admit) | | x | | | TestPlanConstants, TestPlanConstantBooleans, TestPlanConstantNonBooleanJSON, TestClosedConstantGateShape | N-bounded-decode |
| Contract SemVer | x | x | | | TestContractSemver | N-semver-build/decode |
| Contract has no extensions | | x | | | TestContractNoExtensions | (covered by N-unknown-shared) |
| forbid admits literal strings (I4) | x | x | | | TestForbidReasonsAdmitLiteralStrings | (admission bound, not a gate) |
| No mapping disposition coupling (I10) | x | x | | | TestNoDispositionCouplingOnMappings | (admission bound, not a gate) |
| total_bytes any uint53 (I11 bound) | | | x | x | TestTotalBytesIsAStatedBoundUntilSpecDefinesDerivation | (admission bound, not a gate) |

## Importer outcome grid (package, entry, input): base vs candidate

Base is trunk HEAD (`547ea28`); candidate is the uncommitted
tree. The six reused owner packages are byte-identical between the
two (171 files, hash-compared) and their suites are green on both,
so no input class moved. Re-verified in rev4 (production files
untouched since rev3: plans.go still
`sha256:c2f7eb93506291635218beb49a589d8920bf32459d2906d73c9c3c74e311f02f`).

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

## Axis inventory

- Member census: 13 shapes, 97 members, reflection-derived from
  the Go input types (nested structs, slices of structs, maps;
  4 documented irregular names; synthesized members by sealed set
  difference); per member: unknown/missing (decode), miscased in 6
  variant classes (decode), 5-6 wrong JSON types (decode),
  duplicated (decode, byte-path splicer), zero-value (build),
  5 invalid-UTF-8 classes (both). Bounded: all axes enumerated
  exhaustively. `TestMemberCensusNoLiteralList` guards the
  derivation (trips at 10+ member names, 40%+ purity).
- Identities: 2 ids; independent JCS recompute + wrong-omission
  disagreement + tampered/non-digest refusals.
- DAG: sequences unbounded above (uint53) — structural argument
  (strict decrease forbids cycles; existence is a set lookup;
  locality pinned by AST + 35 single-edge toggles) + every graph
  through 4 ops (2+16+512+65536) vs the oracle through both
  entries + every edge class at every 5-op position (30 probes) +
  all 1024 valid 5-DAGs admit + targeted invalids
  (higher/self/dangling/dup/unordered/non-contiguous).
- Nullability: resources kind(2) x mode(3) x blob(2) = 12 cells
  (full), entries kind(2) x 2^3 presence x mode(3) = 48 cells
  (full), both entries, vs the oracle; every refuse cell asserts a
  structural literal; the full entry grid additionally runs under
  panic recovery.
- Bounds: every length/count bound at min/max/min-1/max+1 with the
  1M mapping edge sealed and decoded at both production entries;
  mode/sequence/uint53 edges; multibyte character spot check.
- Constants: gate list reflection-derived from the nested plan
  input types (4 string, 7 closed boolean, 1 open boolean, modes)
  plus the 4 decode-only schema/version literals, cross-checked
  both directions against the spec oracle. Per string literal at
  both entries: every edit-distance-1 neighbor over
  lowercase+digit+underscore, every case variant class, a
  fixed-seed 200-string sample, the `copy` second-value probe,
  empty, and whitespace paddings; per boolean: the other boolean
  both entries plus all 5 non-boolean JSON types at decode;
  modes: per-element neighbor/case sweeps, order swap, lengths
  0..4, non-string elements, non-array JSON. The single-literal
  AST guard (`TestClosedConstantGateShape`) pins each gate to one
  equality against the oracle literal: a second admitted value, a
  switch, or a prefix/fold/contains call fails it (control-planted
  with the reviewer's copy widening and a two-case switch, both
  redden). The unbounded remainder past the generated set is
  closed by the guard, not by sampling: any second value changes
  the gate shape or the comparison-literal set.
