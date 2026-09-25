# TASK-260830-2ya5le Conformance Matrix — implement-fidelity-report-and-reason-registry

Authority: `internal/specdoc/SPEC.v0.7.0.md`, Section 13.14.2
"Fidelity, projection, and lineage" (line 10548+). Production
entries: `BuildFidelityReport`, `DecodeFidelityReport`,
`ValidDisposition`, `ValidProfile`, `ValidStrategy`,
`ValidReasonCode`, `IsCoreReasonCode`
(`internal/clonefidelity/`). Every proof drives a production entry;
no row calls an unexported helper directly except
`TestCheckRowCountEdges`, which pairs with entry reachability rows
(see row 27).

## A. AC coverage: 39 of 40 rows driven + 1 stated bound (rev3)

Rev2 answers verdict rev1 (`unbound-byte-aggregate`) per the
orchestrator decision: row 24 (byte-aggregate reconciliation) is
an explicit stated bound, not a driven row — `byte_counts` is
shape-validated only, because the spec defines no byte basis for
canonical-event or synthesized rows (SPEC v0.7.0 §13.14.2 line
10636 vs §13.14.1 line 10471). Owner: spec clarification
(agent-session-manager-spec) + the §13.14.1 capture-manifest
leaf. Nothing in this matrix claims `byte_counts` is
row-reconciled.

Rev3 answers verdict rev2 (`unmeasured-build-utf8-sites`): row 33
is now driven over the full derived class, not the 4 of 7 sites
the reviewer measured. `TestUTF8RefusalReflectionGrid` derives 33
string-carrying members from the input types by reflection and
drives member × 5 invalid-UTF-8 classes × both entries (330
cells); build cells pin member-specific literals, decode cells pin
the delegated frame literal. Zero production bytes changed vs the
rev2 candidate tree `7bade702` (measured by `cmp` per file; see
the results). The rev2 reviewer held every other row.

"Narrowing" names the harness row(s) in
`internal/clonefidelity/testdata/mutant_harness.py` that redden the
row's killer(s); every narrowing admits exactly one member of the
class its gate must reject. "(killer)" marks the plant whose
committed killer is the row's test. Battery: 70 narrowings KILLED +
control SURVIVED, executed twice, exit 0, one raw log per plant per
run under `mutants/run1/` + `mutants/run2/` in the producer
evidence archive.

| # | AC behavior | Driving test(s) → call site | Narrowing(s) |
|---|---|---|---|
| 1 | Seven dispositions admitted exactly; case/whitespace/near-miss/empty/non-string refused with a literal code | `TestDispositionVocabularyOracle` → `ValidDisposition` + `BuildFidelityReport` + `DecodeFidelityReport` | `N-disposition-vocab` (killer: admits exactly `EXACT`) |
| 2 | Five profiles admitted exactly; others refused with a literal code | `TestProfileVocabularyOracle` → `ValidProfile` + both report entries | `N-profile-vocab` (killer: admits exactly `Maximal_Safe`) |
| 3 | Five strategies admitted exactly; others refused | `TestStrategyVocabularyOracle` → `ValidStrategy` (registry entry; no report field carries a strategy) | `N-strategy-vocab` (killer: admits exactly `archive-only`) |
| 4 | Nineteen core reasons + reverse-DNS extensions admitted; others refused; extensions can never redefine/shadow a core code | `TestReasonVocabularyOracle` + `TestReasonShadowImpossible` → `ValidReasonCode`/`IsCoreReasonCode` + both entries (disjointness asserted per core code) | `N-reason-core` (killer), `N-reason-extension` (killer), `N-reason-core-route` (killer; token-preserving) |
| 5 | Exact requires an empty reason set | `TestRecordGridOracle` (140 cells through both entries) → `checkRecordReasonRule` | `N-record-exact-reasons` (killer: admits exactly exact+1) |
| 6 | Every non-exact disposition requires ≥1 reason | same grid | `N-record-nonexact-empty` (killer: admits exactly semantic+0) |
| 7 | Synthesized requires no source canonical object | same grid | `N-record-synth-canonical` (killer; single-member class) |
| 8 | Non-synthesized rows carry source evidence (presence half; resolution is a stated bound) | same grid | `N-decode-evidence-empty` (decode), `N-build-evidence-empty` (build) (killers) |
| 9 | Reason-set cardinality 128 admits / 129 refuses on every disposition | same grid (sizes 0,1,2,128,129) | `N-decode-reasons-count` (decode), `N-build-reasons-count` (build) (killers) |
| 10 | Record string bounds at edge and edge+1 (key 512, class 128, locator 1024, explanation 4096, reason item 128) | `TestRecordFieldBounds` (9 refusal pairs + edge admission + null locator, both entries) | `N-decode-key-bound`, `N-build-key-bound`, `N-decode-reason-item-bound` (killers) |
| 11 | String measure in characters, not bytes | `TestRecordMultibyteMeasure` (512/513 `é`, both entries) → `environ.StringLength` | gate-level: rows 10 narrowings pin the bound sites sharing this gate |
| 12 | Sorted-unique arrays refuse unsorted and duplicate | `TestRecordSortedUnique` (reasons/evidence/staged/live × 2 shapes × 2 entries) | `N-decode-sorted-reasons`, `N-build-sorted-evidence` (killers; both helpers × both entries narrowed — §B) |
| 13 | Disposition row exact 11-member shape | `TestRecordMemberSet` (unknown + 11 missing) | `N-decode-record-unknown`, `N-decode-record-missing` (killers) |
| 14 | Report exact 29-member set; schema/version literals | `TestReportMemberSet` (unknown + 29 missing + 9 literal cases) + `TestReportRoundTrip` (wire literals) | `N-decode-member-unknown`, `N-decode-member-missing`, `N-decode-schema`, `N-decode-scope`, `N-build-scope` (killers) |
| 15 | FidelityCounts exact 7-member shape | `TestCountsMemberSet` (unknown + 7 missing) | `N-decode-counts-unknown`, `N-decode-counts-missing` (killers) |
| 16 | `fidelity_report_id` is the JCS digest omitting only itself | `TestReportIdentityIndependent` (test-side JCS recompute agrees; omitting scope instead disagrees; zero/flipped/non-digest claims refuse) → `BuildFidelityReport` + `verifySelfDigest` | `N-decode-selfdigest` (killer: admits exactly the zero claim) |
| 17 | Archive/target branch-exact nullability (plan, target env, staged/live IDs, target booleans) | `TestArchiveTargetBranches` (10 build + 10 decode flips) | `N-decode-archive-plan`, `N-decode-target-plan`, `N-decode-archive-targetbools`, `N-build-archive-plan` (killers) |
| 18 | Profile/scope coupling (`archive_only` exactly for archive) | same matrix (5 profiles × 2 scopes × 2 entries) | `N-decode-scope-profile`, `N-build-scope-profile` (killers) |
| 19 | Rows ordered: source sorted by key, then synthesized sorted by key, keys unique | `TestRowOrdering` (5 build + 5 decode violations) → `checkRowOrder` | `N-roworder-synth-after`, `N-roworder-sorted`, `N-roworder-dupkey` (killers) |
| 20 | `counts` derived from rows; any ±1 cell refuses | `TestCountsDerived` (literal totals + 14-cell sweep + zero-cell boundary) → `deriveCounts` | `N-disposition-route` (killer; token-preserving), `N-decode-counts-cell` (killer) |
| 21 | `reason_counts` derived from rows; any ±1/extra/missing/zero refuses | `TestReasonCountsDerived` (literal occurrences + 6-cell sweep + key cases) → `deriveReasonCounts` | `N-decode-reasoncounts-cell`, `N-decode-reasoncounts-zero` (killers) |
| 22 | `event_kind_counts` closed 26-key map reconciled per disposition | `TestEventKindBreakdown` (182-cell +1 sweep + 7 non-zero −1 + closed/malformed keys + build rows) → `decodeBreakdown`/`buildBreakdown` + `checkBreakdownSums` | `N-eventkind-sum` (killer), `N-decode-eventkind-keys` (killer), `N-decode-breakdown-missing` (killer) |
| 23 | `content_block_counts` closed 8-key map reconciled per disposition | `TestContentBlockBreakdown` (56-cell +1 sweep + 7 non-zero −1 + closed keys + build row) | `N-contentblock-sum` (killer), `N-decode-contentblock-keys` (killer) |
| 24 | `byte_counts` value reconciliation — EXPLICIT STATED BOUND, not driven (shape-validated only: exact 7 keys + uint53 values; no reconciliation gate exists to narrow — §D; owner: spec clarification (agent-session-manager-spec) + the §13.14.1 capture-manifest leaf) | `TestByteCountsReconciliationIsAStatedBoundUntilSpecDefinesByteBasis` (bound witness: exact keys, max value, 6 bad-value shapes, differing-bytes admission, build rows) | `N-decode-bytekeys`, `N-build-uint53-bytecounts` (killers of the shape gate, not of reconciliation) |
| 25 | `required_dispositions` closed capture-class map to non-empty disposition sets | `TestRequiredDispositions` (9 classes + empty map admit; 6 build + 8 decode refusals) | `N-decode-required-keys`, `N-decode-required-values`, `N-build-required-empty` (killers) |
| 26 | `forbid_reasons` sorted unique literal strings (128/129 edges) | `TestForbidReasons` (non-reason strings kept; 4 build + 6 decode refusals) | `N-build-sorted-forbid` (killer) |
| 27 | Row cardinality 1..1000000 (0/1/1000001 at the entries; 999999/1000000/1000001 at the factored gate) | `TestRowCountEdges` + `TestCheckRowCountEdges` → `checkRowCount` | `N-rowcount-max`, `N-rowcount-min` (killers) |
| 28 | Evidence/attestation bounds at edge/edge+1 (65536/65537, 64/65, both entries) | `TestEvidenceBounds` | `N-decode-sorted-attestations` (killer; order gate); count gates share the representative narrowings |
| 29 | Scalar gates: digests, UUIDv7, tuples refuse with member-naming literals | `TestScalarGates` (15 build + 19 decode rows incl. attestation order) → landed owners | `N-decode-digest`, `N-decode-uuid`, `N-decode-tuple` (killers); tuple-shape internals owned by `sessadapter` |
| 30 | AX number model at every numeric gate (fraction/exponent/string/≥2^53/negative refuse; 2^53−1 admits) | `TestNumberModel` (4 cells × 7 bad values + max admit) → `environ.CheckUint53Bounds` | `N-decode-uint53-counts` (killer) |
| 31 | Closed extensions at both layers (reverse-DNS keys, AX values, no nested dup) | `TestExtensionsClosed` (admission + 6 build + 4 decode rows) → `clonebundle` owner | `N-decode-extensions` (killer); value internals owned by `clonebundle` |
| 32 | Strict framing at both object layers | `TestReportFraming` (dup/trailing/non-object/surrogate/UTF-8 + row dup) → `decodeStrictObject` | `N-frame-report`, `N-frame-row` (killers) |
| 33 | Invalid UTF-8 refused at every string-valued member, both entries (no U+FFFD rewrite-and-continue; decode refuses at the delegated frame before dispatch) | `TestUTF8RefusalReflectionGrid` (33 derived members × 5 classes × 2 entries = 330 cells) + `TestRegistryPredicatesRefuseInvalidUTF8` (5 predicates × 5 classes) + `TestBuildUTF8ExtensionsNestedDepth` + `TestBuildUTF8` (kept) → `BuildFidelityReport` + `DecodeFidelityReport` + 5 registry predicates | `N-build-utf8-source-class`, `N-build-utf8-target-locator`, `N-build-utf8-forbid-reasons` (the three reviewer sites), `N-build-utf8-source-item-key`, `N-build-utf8-explanation`, `N-build-utf8-reason-codes`, `N-build-utf8-row-extensions`, `N-build-utf8-report-extensions` (all killers, each killed by the grid run alone) |
| 34 | Determinism: identical inputs seal byte-identical outputs | `TestReportDeterminism` (10 builds compared) | structural: no gate (admitted property — disclosed) |
| 35 | Purity: no durable write (decode never mutates input; builds decode clean) | `TestEntriesArePure` | structural: no gate (disclosed) |
| 36 | Owner vocabularies equal the spec lists (26/8/9 + own registry counts) | `TestOwnerVocabulariesMatchSpec` | structural: cross-check, not a gate (disclosed) |
| 37 | Breakdown reconciliation is a sum, not a shape (spread totals admit) | `TestSpreadBreakdownAdmits` | gate-level: rows 22–23 narrowings pin the sum gate |
| 38 | No unsupported capability advertised (no README/doctor/capability surface; library only) | verified by diff (4 additive landed files + new package; no command/doctor/config touch) | n/a (absence claim over the diff — disclosed) |
| 39 | Crash/idempotency: operation mutates no durable state, so crash semantics are vacuous; determinism is the proof | rows 34–35 | n/a (vacuous — disclosed) |
| 40 | Every refusal carries the stable code plus a literal detail | `requireRefusal`/`requireRefusalTokens` in every refusal assertion (`errors.Is(ErrInvalid)` + literal substring) | every narrowing above kills through a literal assertion |

Ratio: **39 of 40 AC rows driven** through production entries by
named committed tests **+ 1 stated bound**. Rows 34–36 are
admitted properties / cross-checks with no gate to narrow
(disclosed, not waived); row 24 is the explicit stated bound
(byte-aggregate reconciliation, §D) and is not counted as driven.
Row 33 is fully driven: the 33-member derived set below, each
member × 5 invalid-UTF-8 classes (lone continuation `80`,
`ff fe`, overlong `c0 af`, encoded surrogate `ed a0 80`,
truncated sequence `e2 82`) × both entries, plus the 5 registry
predicates × 5 classes.

Derived UTF-8 member set (reflection over
`FidelityReportInput`/`DispositionRecordInput`; asserted non-empty
in the test; a new string field fails the test until probed):

- `report.AdapterAttestations[]`, `report.BundleID`,
  `report.ByteCounts<key>`, `report.CanonicalSessionID`,
  `report.CaptureManifestID`, `report.ContentBlockCounts<key>`,
  `report.EventKindCounts<key>`, `report.Extensions<key>`,
  `report.Extensions<value>`, `report.ForbidReasons[]`,
  `report.LiveReadBackEvidenceManifestID`, `report.OperationID`,
  `report.Profile`, `report.ProjectionPlanID`,
  `report.RequiredDispositions<key>`,
  `report.RequiredDispositions[]`
- `report.Rows[].CanonicalObjectID`, `report.Rows[].Disposition`,
  `report.Rows[].Explanation`, `report.Rows[].Extensions<key>`,
  `report.Rows[].Extensions<value>`,
  `report.Rows[].LiveEvidenceObjectIDs[]`,
  `report.Rows[].ReasonCodes[]`, `report.Rows[].SourceClass`,
  `report.Rows[].SourceEvidenceIDs[]`,
  `report.Rows[].SourceItemKey`,
  `report.Rows[].StagedEvidenceObjectIDs[]`,
  `report.Rows[].TargetLocator`
- `report.Scope`, `report.SourceEnvironment`,
  `report.SourceSnapshotDigest`,
  `report.StagedReadBackEvidenceManifestID`,
  `report.TargetEnvironment`

Decode-side granularity, stated plainly: every decode cell refuses
with the frame literal (`fidelity report not valid UTF-8 (frame)`),
because the landed `environ.DecodeStrictObject` gate trips before
any member dispatch. Member-level attribution on decode would
require re-implementing the landed frame parser in this leaf, which
the epic's composition rule forbids; the per-member plants prove
no member's bytes bypass the frame gate.

## B. Gate × entry census

Each row is one gate; "Build"/"Decode" name the implementing code
(shared = one implementation, two call sites; split = implemented
twice). Domain axes: disposition(7) × reason-size(0,1,2,128,129) ×
canonical(2) × evidence(2); profile(5) × scope(2); branch
fields(5) × scope(2); counts cells(7) × ±1; reason cells(R) × ±1;
event cells(182) + content cells(56) × +1; byte keys(7); string
bounds(5 sites) × edge/edge+1; sorted arrays(6 sites) ×
unsorted/duplicate; utf8-members(33) × classes(5) × entries(2);
registry predicates(5) × classes(5).

| Gate | Build site | Decode site | Killer(s) | Narrowing(s) |
|---|---|---|---|---|
| Disposition vocabulary | shared `ValidDisposition` | shared | `TestDispositionVocabularyOracle` | `N-disposition-vocab` |
| Profile vocabulary | shared `ValidProfile` | shared | `TestProfileVocabularyOracle` | `N-profile-vocab` |
| Strategy vocabulary | `ValidStrategy` (registry only) | — | `TestStrategyVocabularyOracle` | `N-strategy-vocab` |
| Core reason vocabulary | shared `IsCoreReasonCode` | shared | `TestReasonVocabularyOracle` | `N-reason-core`, `N-reason-core-route` |
| Extension reason arm | shared `isExtensionReason` | shared | `TestReasonVocabularyOracle` | `N-reason-extension` |
| Exact/non-exact reason rule | shared `checkRecordReasonRule` | shared | `TestRecordGridOracle` | `N-record-exact-reasons`, `N-record-nonexact-empty` |
| Synthesized canonical rule | shared `checkRecordCanonicalRule` | shared | `TestRecordGridOracle` | `N-record-synth-canonical` |
| Counts derivation | shared `deriveCounts`/`addRow` | shared | `TestCountsDerived` | `N-disposition-route` |
| Row order | shared `checkRowOrder` | shared | `TestRowOrdering` | `N-roworder-synth-after`, `N-roworder-sorted`, `N-roworder-dupkey` |
| Row count bound | shared `checkRowCount` | shared | `TestRowCountEdges` (+ helper) | `N-rowcount-max`, `N-rowcount-min` |
| Breakdown sums | shared `checkBreakdownSums` | shared | `TestEventKindBreakdown`, `TestContentBlockBreakdown` | `N-eventkind-sum`, `N-contentblock-sum` |
| Strict frame | — (Go inputs) | shared `decodeStrictObject` | `TestReportFraming` | `N-frame-report`, `N-frame-row` |
| UTF-8 refusal, build per-member (explicit pre-marshal arms + implicit vocab/digest/tuple refusals) | 8 explicit sites (`record.go` key/class/locator/explanation, `decode.go` reason/forbid arms, 2 extension call sites) + member-naming refusals at every other derived member | — (decode subsumed by the strict frame) | `TestUTF8RefusalReflectionGrid` | `N-build-utf8-source-class`, `N-build-utf8-target-locator`, `N-build-utf8-forbid-reasons`, `N-build-utf8-source-item-key`, `N-build-utf8-explanation`, `N-build-utf8-reason-codes`, `N-build-utf8-row-extensions`, `N-build-utf8-report-extensions` |
| UTF-8 refusal, decode frame | — | shared `decodeStrictObject` (invalid bytes trip before dispatch) | `TestUTF8RefusalReflectionGrid` (165 decode cells, one plant per member × class) | frame gate pinned by `N-frame-report`/`N-frame-row`; the UTF-8 arm itself is owned by `environ` (disclosed — §D) |
| UTF-8 refusal, registry predicates | 5 predicates return false | n/a (boolean entries) | `TestRegistryPredicatesRefuseInvalidUTF8` | predicate arms pinned by the vocabulary narrowings above (disclosed — §D) |
| Self digest | seal (no gate) | `verifySelfDigest` | `TestReportIdentityIndependent` | `N-decode-selfdigest` |
| Counts reconciliation | derived (no gate) | comparison loop | `TestCountsDerived` | `N-decode-counts-cell` |
| Reason reconciliation | derived (no gate) | comparison loops | `TestReasonCountsDerived` | `N-decode-reasoncounts-cell`, `N-decode-reasoncounts-zero` |
| String bounds (key) | `buildDispositionRecord` | `decodeDispositionRecord` | `TestRecordFieldBounds` | `N-build-key-bound`, `N-decode-key-bound` |
| Reason item/count bounds | `checkSortedUniqueReasonStrings` | `checkSortedUniqueStrings` call | `TestRecordFieldBounds`, `TestRecordGridOracle` | `N-build-reasons-count`, `N-decode-reason-item-bound`, `N-decode-reasons-count` |
| Evidence presence | `parseSortedUniqueDigestStrings` call | `checkSortedUniqueDigests` call | `TestRecordGridOracle` | `N-build-evidence-empty`, `N-decode-evidence-empty` |
| Sorted order (strings) | `checkSortedUniqueBoundedStrings` | `checkSortedUniqueStrings` calls | `TestForbidReasons`, `TestRecordSortedUnique` | `N-build-sorted-forbid`, `N-decode-sorted-reasons` |
| Sorted order (digests) | `parseSortedUniqueDigestStrings` | `checkSortedUniqueDigests` calls | `TestRecordSortedUnique`, `TestScalarGates` | `N-build-sorted-evidence`, `N-decode-sorted-attestations` |
| Scope literal | build switch | decode switch | `TestReportMemberSet` | `N-build-scope`, `N-decode-scope` |
| Scope/profile coupling | build branch | decode branch | `TestArchiveTargetBranches` | `N-build-scope-profile`, `N-decode-scope-profile` |
| Archive plan nullability | build branch | decode branch | `TestArchiveTargetBranches` | `N-build-archive-plan`, `N-decode-archive-plan` |
| Target plan nullability | build branch | decode branch | `TestArchiveTargetBranches` | (build: message-level, covered by behavior rows; decode: `N-decode-target-plan`) |
| Archive target booleans | build branch | decode branch | `TestArchiveTargetBranches` | `N-decode-archive-targetbools` (decode; build arm disclosed in §D) |
| Member sets (report/row/counts/cell) | — (constructed) | 8 member gates | member-set tests | `N-decode-member-unknown/missing`, `N-decode-record-unknown/missing`, `N-decode-counts-unknown/missing`, `N-decode-breakdown-missing` |
| Closed breakdown keys | `buildBreakdown` | `decodeBreakdown` | breakdown tests | `N-decode-eventkind-keys`, `N-decode-contentblock-keys` (decode; build arms covered by behavior rows — §D) |
| Byte keys/values (shape gate only; reconciliation is the row-24 stated bound) | `buildByteCounts` | `decodeByteCounts` | `TestByteCountsReconciliationIsAStatedBoundUntilSpecDefinesByteBasis` | `N-decode-bytekeys`, `N-build-uint53-bytecounts` |
| Required keys/values | `buildRequiredDispositions` | `decodeRequiredDispositions` | `TestRequiredDispositions` | `N-decode-required-keys/values`, `N-build-required-empty` |
| Scalar delegation (digest/UUID/tuple) | `scalar.*`/`sessadapter` calls | `environ.*`/`sessadapter` calls | `TestScalarGates` | `N-decode-digest`, `N-decode-uuid`, `N-decode-tuple` (decode reachability; grammars owned downstream) |
| uint53 delegation | Go uint64 checks | `checkUint53Bounds` calls | `TestNumberModel`, `TestByteCountsReconciliationIsAStatedBoundUntilSpecDefinesByteBasis` | `N-decode-uint53-counts`, `N-build-uint53-bytecounts` |
| Extensions delegation | `EncodeExtensions` call | `checkExtensionsClosed` calls | `TestExtensionsClosed` | `N-decode-extensions` (decode reachability; rule owned by `clonebundle`) |
| Extensions UTF-8 call sites | `EncodeExtensions` calls at both layers | — (subsumed by frame) | `TestUTF8RefusalReflectionGrid`, `TestBuildUTF8ExtensionsNestedDepth` | `N-build-utf8-row-extensions`, `N-build-utf8-report-extensions` |
| Schema literal | constructed (no gate) | comparison | `TestReportMemberSet` | `N-decode-schema` |

Kill attribution is isolated: every harness row runs its named
killer alone (`-run <Test> -count=1`) with one raw log per plant
per run.

## C. Importer outcome grid (package, entry, input): base vs candidate

Base = trunk `0ca3e4c` before this leaf; candidate = uncommitted
tree. Moved classes: **none** — the landed diff is six additive
one-line delegates plus one behavior-identical refactor line
(`CheckExtensions` body now calls `CheckReverseDNS`).

Rev3 note: the rev3 delta vs the rev2 candidate changes zero
production bytes (all 10 production files byte-identical to tree
`7bade702` by `cmp`; only `mutant_harness.py` differs and
`utf8_reflection_test.go` is new, plus TRACEABILITY wording), so
no outcome can have moved; the importer set below was still rerun
on the rev3 tree and is green.

| Package | Entry | Input class | Base | Candidate |
|---|---|---|---|---|
| clonebundle | full suite (`go test ./internal/clonebundle/`) | committed suite incl. extension/digest/vocabulary rows | ok 1.3s | ok 1.3s |
| clonebundle | `CheckExtensionsClosed` (new) | fidelity extensions (report + row) | n/a (absent) | exercised by `TestExtensionsClosed` (both entries) |
| clonebundle | `EncodeExtensions` (new) | fidelity build extensions | n/a (absent) | exercised by build rows |
| clonebundle | `OmitSelfDigest` (new) | fidelity omit-self members | n/a (absent) | exercised by `TestReportIdentityIndependent` |
| clonebundle | `EventKinds`/`ContentBlockTypes`/`ValidContentBlockType` (new) | fidelity breakdown key sets | n/a (absent) | exercised by breakdown tests + `TestOwnerVocabulariesMatchSpec` |
| environ | full suite | committed suite incl. extensions/frame rows | ok 0.4s | ok 0.4s |
| environ | `CheckReverseDNS` (new) | extension reason strings | n/a (absent) | exercised by `TestReasonVocabularyOracle` |
| environ | `CheckExtensions` (refactored body) | extension objects | ok | ok (identical verdicts; suite green) |
| sessadapter | `DecodeTuple` via fidelity tuples | valid/invalid tuples | ok 0.7s | ok 0.7s (package untouched) |
| scalar | digest/UUID gates via fidelity scalars | valid/invalid scalars | ok 0.3s | ok 0.3s (package untouched) |
| canonicaljson | `Canonicalize` via fidelity sealing | fidelity objects | ok 21s | ok (package untouched; fidelity-report registry row still refuses by design) |
| clonefidelity | `BuildFidelityReport` | 38-test suite (grid + predicates + nested + kept rows) | n/a (new) | ok, race clean |
| clonefidelity | `DecodeFidelityReport` | same suite | n/a (new) | same run |
| clonefidelity | registry predicates | oracle suites + UTF-8 predicate sweep | n/a (new) | same run |

## D. Coverage map, brief gaps, and out-of-contract rows

The brief carries **no surface table** (no
`references/attack-surface-catalog.md` rows), so per the Handoff
Preconditions this row reports that brief gap instead of a
per-surface map: the gate census (§B) plus the clause map in
`internal/clonefidelity/TRACEABILITY.md` is the coverage map — every
gate points to its tests and a killed narrowing (§A, §E).

Out-of-contract rows, each with its acceptance clause:

| Row | Treated as | Acceptance clause |
|---|---|---|
| Projection Plan, Read-Back Evidence Manifest, Validation Report, Migration Checkpoint, Lineage Receipt | sibling scope (identity/digest fields only) | Producer brief: "Projection Plan, Read-Back, Validation Report, Migration Checkpoint and Lineage Receipt belong to sibling leaves. Do not implement them." |
| Cross-artifact row coverage ("every Capture Manifest item and Canonical Event occurs in exactly one non-synthesized row") | stated bound (no entry takes the manifests) | Task Scope + brief oracle (enumerates the four record rules only) |
| Source-evidence resolution against captured bytes | stated bound (presence/shape gated) | Same |
| `byte_counts` value reconciliation | EXPLICIT STATED BOUND — row 24 not driven (shape-validated only; rows carry no byte measure and the spec defines no byte basis for canonical-event/synthesized rows) | SPEC §13.14.2 table row (line 10636) + reconciliation rule (line 10645) vs record shape; owner: spec clarification (agent-session-manager-spec) + the §13.14.1 capture-manifest leaf |
| Core-derived boolean semantics beyond the archive rule | stated bound (booleans admitted; archive rule pinned) | SPEC §13.14.2 ("Core-derived booleans", derivation undefined here) |
| Tuple-sub-member UTF-8 enumeration (this leaf pins per-tuple refusal × 5 classes at both tuple members and both entries; member-level tuple enumeration belongs to the tuple owner) | landed-owner scope (`sessadapter`) | Composition rule: never re-implement what a landed owner gates; the leaf decodes tuples opaquely through `DecodeTuple` |
| Decode-side member-level UTF-8 attribution (decode refuses every member plant with the frame literal, pinned per member × class) | delegated frame granularity (driven, not waived — see row 33) | Same: attribution would re-implement `environ.DecodeStrictObject`; the plants prove no member bypasses it |
| Registry edit (`ownership.v0.7.0.json`) | final-leaf scope (untouched) | Producer brief: "Do NOT edit ... the Story's final leaf carries the registry." |
| Admission at exactly 1,000,000 rows via a sealed report | factored proof (exact gate + entry reachability, not a 1M-row drive) | Method bound, disclosed in row 27 + TRACEABILITY |

Single-side narrowings disclosed (rule implemented twice; the other
side covered by behavior rows + shared-gate plants): target-plan
build arm, archive-booleans build arm, breakdown-keys build arms.
The rev2 "UTF-8 single-predicate arm" disclosure is closed: all
eight build pre-marshal sites carry their own narrowing. No row is
prose-only.

## E. Mutant table

Generated from `testdata/mutant_harness.py` (name, note, killer,
expected verdict); every row ran twice with exit 0 and one raw log
per plant per run.

Mutant | Narrows the gate to | Named test that fails | Verdict x2
--- | --- | --- | ---
`N-disposition-vocab` | Shared gate: admits exactly EXACT past the disposition vocabulary at both entries; every other non-member still refuses. | `TestDispositionVocabularyOracle` | KILLED |
`N-profile-vocab` | Shared gate: admits exactly Maximal_Safe past the profile vocabulary at both entries. | `TestProfileVocabularyOracle` | KILLED |
`N-strategy-vocab` | Admits exactly archive-only past the strategy registry; the predicate is the production entry. | `TestStrategyVocabularyOracle` | KILLED |
`N-reason-core` | Shared gate: admits exactly the truncated near-miss unknown_native_even as a core reason at both entries. | `TestReasonVocabularyOracle` | KILLED |
`N-reason-extension` | Shared gate: admits exactly nodots past the extension arm at both entries; every other malformed name still refuses. | `TestReasonVocabularyOracle` | KILLED |
`N-reason-core-route` | Token-preserving: unknown_native_event still matches the table but falls through to the extension check and refuses; the behavioral oracle pins its admission through both entries. | `TestReasonVocabularyOracle` | KILLED |
`N-record-exact-reasons` | Shared gate: admits exactly exact rows with one reason at both entries; exact with 2+ still refuses. | `TestRecordGridOracle` | KILLED |
`N-record-nonexact-empty` | Shared gate: admits exactly semantic rows with no reasons at both entries; every other non-exact disposition still refuses. | `TestRecordGridOracle` | KILLED |
`N-record-synth-canonical` | Labelled single-member class: the reject class holds exactly (synthesized, has-canonical), so admitting it admits one member at both entries. | `TestRecordGridOracle` | KILLED |
`N-disposition-route` | Token-preserving: the exact label still matches but folds into Semantic; the behavioral killer pins literal totals through both entries. | `TestCountsDerived` | KILLED |
`N-roworder-synth-after` | Shared gate: admits exactly the k2 source row after a synthesized row at both entries. | `TestRowOrdering` | KILLED |
`N-roworder-sorted` | Shared gate: admits exactly k1 after k2 in the source group at both entries; every other unsorted pair still refuses. | `TestRowOrdering` | KILLED |
`N-roworder-dupkey` | Shared gate: admits exactly the duplicate k1 key at both entries; every other duplicate still refuses. | `TestRowOrdering` | KILLED |
`N-rowcount-max` | Shared gate: admits exactly 1000001 rows; the entry killer refuses them with the count detail (the factored helper test reddens too). | `TestRowCountEdges` | KILLED |
`N-rowcount-min` | Shared gate: admits exactly zero rows; the entry killer refuses the empty dispositions with the count detail. | `TestRowCountEdges` | KILLED |
`N-eventkind-sum` | Shared gate: admits exactly a +1 event-kind sum at both entries; content sums and larger drifts still refuse. | `TestEventKindBreakdown` | KILLED |
`N-contentblock-sum` | Shared gate: admits exactly a +1 content-block sum at both entries; event sums and larger drifts still refuse. | `TestContentBlockBreakdown` | KILLED |
`N-frame-report` | Shared gate: admits exactly a duplicate scope member past the strict frame into a lenient decode; every other frame fault still refuses. | `TestReportFraming` | KILLED |
`N-frame-row` | Shared gate: admits exactly a duplicate source_class member past the row frame into a lenient decode; the report duplicate still refuses. | `TestReportFraming` | KILLED |
`N-decode-key-bound` | Decode side: admits exactly 513-character item keys; 514+ still refuse and the build side is untouched. | `TestRecordFieldBounds` | KILLED |
`N-decode-reason-item-bound` | Decode side: admits exactly 129-character reason items past the item bound; the killer pins the bound message (the plant moves the refusal to the vocabulary gate). | `TestRecordFieldBounds` | KILLED |
`N-decode-reasons-count` | Decode side: admits exactly 129 reasons past the count bound; the grid killer pins the bound message (downstream gates refuse with their own details). | `TestRecordGridOracle` | KILLED |
`N-decode-evidence-empty` | Decode side: admits exactly empty source evidence past the shape gate; the grid killer pins the shape message. | `TestRecordGridOracle` | KILLED |
`N-decode-counts-cell` | Decode side: admits exactly counts.exact one above derived; every other cell and drift still refuses. | `TestCountsDerived` | KILLED |
`N-decode-reasoncounts-cell` | Decode side: admits exactly credential_excluded one above derived; every other reason cell still refuses. | `TestReasonCountsDerived` | KILLED |
`N-decode-reasoncounts-zero` | Decode side: admits exactly a zero credential_excluded value past the above-zero gate; the killer pins the shape message (reconciliation refuses downstream). | `TestReasonCountsDerived` | KILLED |
`N-decode-eventkind-keys` | Decode side: admits exactly the frobnicate event-kind key (ignored downstream, so the digest kills); content keys still refuse. | `TestEventKindBreakdown` | KILLED |
`N-decode-contentblock-keys` | Decode side: admits exactly the frobnicate content-block key (ignored downstream, so the digest kills); event keys still refuse. | `TestContentBlockBreakdown` | KILLED |
`N-decode-breakdown-missing` | Decode side: admits exactly the usage cell missing exact; the killer pins the missing-member message (the nil cell refuses at the uint53 gate). | `TestEventKindBreakdown` | KILLED |
`N-decode-bytekeys` | Decode side: admits exactly the frobnicate byte key (ignored downstream, so the digest kills). | `TestByteCountsReconciliationIsAStatedBoundUntilSpecDefinesByteBasis` | KILLED |
`N-decode-required-keys` | Decode side: admits exactly the frobnicate policy class (sealed downstream, so the digest kills); the build side is untouched. | `TestRequiredDispositions` | KILLED |
`N-decode-required-values` | Decode side: admits exactly the empty unknown set; every other empty set still refuses. | `TestRequiredDispositions` | KILLED |
`N-decode-scope` | Decode side: admits exactly the staging scope (read as target downstream, so the digest kills); the build side is untouched. | `TestReportMemberSet` | KILLED |
`N-decode-scope-profile` | Decode side: admits exactly archive with maximal_safe; every other archive profile still refuses. | `TestArchiveTargetBranches` | KILLED |
`N-decode-archive-plan` | Decode side: admits exactly the fixture plan digest in archive scope; every other non-null plan still refuses. | `TestArchiveTargetBranches` | KILLED |
`N-decode-target-plan` | Decode side: admits exactly a null plan in target scope; malformed non-null plans still refuse. | `TestArchiveTargetBranches` | KILLED |
`N-decode-archive-targetbools` | Decode side: admits exactly continuable-without-resumable in archive scope; every other true combination still refuses. | `TestArchiveTargetBranches` | KILLED |
`N-decode-selfdigest` | Admits exactly the all-zero identity claim; every other mismatch still refuses. | `TestReportIdentityIndependent` | KILLED |
`N-decode-schema` | Decode side: admits exactly the truncated schema urn (sealed downstream, so the digest kills). | `TestReportMemberSet` | KILLED |
`N-decode-member-unknown` | Decode side: admits exactly the frobnicate top-level member (ignored downstream, so the digest kills). | `TestReportMemberSet` | KILLED |
`N-decode-member-missing` | Decode side: admits exactly a missing counts member; the killer pins the missing-member message (the nil counts refuse at the shape gate). | `TestReportMemberSet` | KILLED |
`N-decode-record-unknown` | Decode side: admits exactly the frobnicate row member (ignored downstream, so the digest kills). | `TestRecordMemberSet` | KILLED |
`N-decode-record-missing` | Decode side: admits exactly a missing explanation; the killer pins the missing-member message (the nil member refuses at its bound gate). | `TestRecordMemberSet` | KILLED |
`N-decode-counts-unknown` | Decode side: admits exactly the frobnicate counts member (ignored downstream, so the digest kills). | `TestCountsMemberSet` | KILLED |
`N-decode-counts-missing` | Decode side: admits exactly counts missing exact; the killer pins the missing-member message (the nil cell refuses at the uint53 gate). | `TestCountsMemberSet` | KILLED |
`N-decode-sorted-reasons` | Decode side: admits exactly the unsorted fixture pair past the strings order gate (valid downstream, so the digest kills); duplicates still refuse. | `TestRecordSortedUnique` | KILLED |
`N-decode-sorted-attestations` | Decode side: admits exactly the unsorted hand-digest pair past the digests order gate (valid downstream, so the digest kills). | `TestScalarGates` | KILLED |
`N-decode-uint53-counts` | Decode side: admits exactly 1.5 as 1 in counts.exact (reconciles by coincidence, so the digest kills); every other non-uint53 still refuses. | `TestNumberModel` | KILLED |
`N-decode-digest` | Decode side: admits exactly sha256:xyz as the snapshot digest (substituted downstream, so the digest kills). | `TestScalarGates` | KILLED |
`N-decode-uuid` | Decode side: admits exactly not-a-uuid as the operation ID (substituted downstream, so the digest kills). | `TestScalarGates` | KILLED |
`N-decode-tuple` | Decode side: admits exactly the platform-only tuple document (zero tuple downstream, so the digest kills). | `TestScalarGates` | KILLED |
`N-decode-extensions` | Decode side: admits exactly the frobnicate-keyed extensions (ignored downstream, so the digest kills); the row layer is untouched. | `TestExtensionsClosed` | KILLED |
`N-build-key-bound` | Build side: admits exactly 513-character item keys and seals them; 514+ still refuse and the decode side is untouched. | `TestRecordFieldBounds` | KILLED |
`N-build-reasons-count` | Build side: admits exactly 129 build reasons and seals them; 130+ still refuse. | `TestRecordGridOracle` | KILLED |
`N-build-evidence-empty` | Build side: admits exactly empty source evidence and seals it; the decode side is untouched. | `TestRecordGridOracle` | KILLED |
`N-build-scope` | Build side: admits exactly the staging scope and seals it as target; the decode side is untouched. | `TestReportMemberSet` | KILLED |
`N-build-scope-profile` | Build side: admits exactly archive with strict_exact and seals it; every other archive profile still refuses. | `TestArchiveTargetBranches` | KILLED |
`N-build-archive-plan` | Build side: admits exactly the fixture plan digest in archive scope and seals it; every other non-null plan still refuses. | `TestArchiveTargetBranches` | KILLED |
`N-build-uint53-bytecounts` | Build side: admits exactly 2^53 in a byte cell and seals it; larger values still refuse. | `TestByteCountsReconciliationIsAStatedBoundUntilSpecDefinesByteBasis` | KILLED |
`N-build-sorted-forbid` | Build side: admits exactly the unsorted fixture pair past the strings order gate and seals it; duplicates still refuse. | `TestForbidReasons` | KILLED |
`N-build-sorted-evidence` | Build side: admits exactly the unsorted hand-digest pair past the digests order gate and seals it; duplicates still refuse. | `TestRecordSortedUnique` | KILLED |
`N-build-required-empty` | Build side: admits exactly the empty unknown set and seals it; every other empty set still refuses. | `TestRequiredDispositions` | KILLED |
`N-build-utf8-source-class` | Reviewer site 1/3: admits exactly the ff-fe source_class past the pre-marshal gate and seals it. | `TestUTF8RefusalReflectionGrid` | KILLED |
`N-build-utf8-target-locator` | Reviewer site 2/3: admits exactly the ff-fe target_locator past the pre-marshal gate and seals it. | `TestUTF8RefusalReflectionGrid` | KILLED |
`N-build-utf8-forbid-reasons` | Reviewer site 3/3: admits exactly the ff-fe forbid_reasons item (sole caller) past the pre-marshal gate. | `TestUTF8RefusalReflectionGrid` | KILLED |
`N-build-utf8-source-item-key` | Row shape: admits exactly the ff-fe source_item_key past the pre-marshal gate and seals it. | `TestUTF8RefusalReflectionGrid` | KILLED |
`N-build-utf8-explanation` | Row shape: admits exactly the ff-fe explanation past the pre-marshal gate and seals it. | `TestUTF8RefusalReflectionGrid` | KILLED |
`N-build-utf8-reason-codes` | Row shape: admits exactly the ff-fe reason past the UTF-8 arm; it then refuses downstream with the vocabulary literal, so the killer UTF-8 literal assertion fails. | `TestUTF8RefusalReflectionGrid` | KILLED |
`N-build-utf8-row-extensions` | Row shape: admits exactly the ff-fe row extension probe value past the leaf call site of the delegated gate. | `TestUTF8RefusalReflectionGrid` | KILLED |
`N-build-utf8-report-extensions` | Report shape: admits exactly the ff-fe report extension probe value past the leaf call site of the delegated gate. | `TestUTF8RefusalReflectionGrid` | KILLED |
`C-doc-comment` | Harmless control: a comment-only edit must survive. | `TestEntriesArePure` | SURVIVED |
