# TRACEABILITY — internal/clonereadback

Fourth leaf of STORY-260830-21bxa3 (M4): Clone Read-Back Evidence
Manifest 1.0.0 and Clone Validation Report 1.0.0 as closed schemas
with JCS identities, pinned to SPEC v0.7.0 §13.14.2. The read-back
mode derives from the trusted read authority threaded from the
caller chain, and the report binds its references to the two
sealed sibling reads: each read-back entry mints a
ValidatedReadBack no outside caller can forge, and the report
entries refuse any sibling that is not sealed. Tuple
cross-match, plan/projected/fidelity pairing, and the Story
registry belong to the final leaf (TASK-260830-1esv6u).

Authority quotes below come from
`internal/specdoc/SPEC.v0.7.0.md` (pinned v0.7.0). Line numbers are
1-based.

## Axis inventory (written first)

Every gate axis the conformance matrix walks, with its domain:

| Axis | Domain | Oracle |
|---|---|---|
| A1 shape members | every member of readback (16), evidence (5), report (23): reflection over the input types plus the sealed-minus-input envelope derivation | missing/extra/duplicated/miscased/wrong-type refused with the member literal at every entry, envelope included |
| A2 schema/version | the two literal pairs from the registry (lines 170-171) | sibling URNs, 2.0.0/1.0 versions refused |
| A3 read-back mode | staged\|live (line 10739) | 2 closed modes seal under matching authorities + neighbors incl. invalid UTF-8 refused at Build, Decode, predicate |
| A4 evidence kind | native_sample\|parser_trace\|marker_observation (lines 10748-10751) | 3 closed + 14 neighbors refused at Build, Decode, predicate |
| A5 native-ID equality | equal vs unequal vs empty pairs | equal seals; unequal refuses (read-back) / decides valid=false (report) |
| A6 valid rule | 2^7 check outcomes × ID eq/neq × 6 finding classes = 1536 combos | decided bit seals, flipped bit refuses, at Build and Decode |
| A7 string bounds | [1..512] native IDs/heads, [1..128] media, chars not bytes | 0/1/512/513 and 127/128/129 ASCII + multibyte at edge |
| A8 heads order | sorted unique [0..1024] | unsorted/dup/overlong refuse; 1024 admits, 1025 refuses, at Build and Decode |
| A9 evidence order | sorted unique by (kind, blob) | 6 permutations admit 1; same-kind blob order; dup refuses |
| A10 counts | evidence [0..65536], findings [0..4096] | 0/edge admit, edge+1 refuses at both entries |
| A11 uint53/number | canonical integers, 2^53-1 max | 2^53, float, exponent, string, bool, null refuse |
| A12 scalars | UUIDv7 operation, sha256 digests | v4/malformed refuse at Build; every sealed scalar member probed malformed at Decode |
| A13 owner nesting | tuple/binding/findings documents | broken nested docs refuse with owner detail at every entry |
| A14 extensions | reverse-DNS closed | open keys refuse at every entry |
| A15 identity | JCS digest omitting only the self member | independent recompute agrees; other-member omission disagrees; tamper refuses; stale-id flip refuses under the flipped authority |
| A16 UTF-8 | every Build-side string (7+4+9 derived fields) + heads + extensions | lone/trailing invalid bytes refuse with the member literal |
| A17 ownership | 8 exports, 0 carrying fidelity/plan rows | row census empty; reachability to landed owners; no fork literals/selectors/imports |
| A18 mode authority | purpose (3) × claimed mode (2) × entry (2), derived from ReadAuthority purpose (lines 3796-3797) | admit iff purpose maps to the claim; source_native and relabels refuse; resealed opposite mode refuses with a recomputing digest |
| A19 read-ref binding | claims over {staged-id, live-id, unrelated}² × entry + missing + sibling relations | admit exactly (staged, live); equal/swapped/missing/twice-named/swapped-sibling refuse at both entries |
| A20 sibling seal | seal state {sealed, zero}² × entry (2), plus sealed cross-plan siblings | any zero sibling refuses with the seal literal at both entries; sealed cross-plan siblings admit (reconciliation bound) |

## Clause map (spec sentence → gate → test)

| # | Pinned clause | Gate | Production site | Test |
|---|---|---|---|---|
| C1 | "Clone Read-Back Evidence Manifest 1.0.0 contains exactly its schema/version" + 15 members (10737-10747) | closed shape, 16 members | `DecodeReadBackEvidenceManifest`, `buildReadBackManifest` renders | `TestMemberCensus*` (readback/evidence) |
| C2 | "`read_back_evidence_manifest_id` under the omission rule" (10738) | JCS identity | `Build…` seals, `verifySelfDigest` | `TestReadBackIdentityIndependent` |
| C3 | "`mode:staged\|live`" (10739) | closed mode | `ValidReadBackMode`, both entries | `TestModeVocabularyOracle` |
| C4 | "equal expected and observed native Session IDs" (10741-10742) | equality | `checkNativeIdentityMatch`, both entries | `TestNativeIdentityEqualityOracle` |
| C5 | "`observed_environment:EnvironmentTuple`" (10742) | owner tuple | `sessadapter.DecodeTuple` at both entries | `TestOwnerDelegation` |
| C6 | "`parsed_event_count:uint53`" (10743) | uint53, no derivation (bound) | `checkUint53Bounds` / `> maxUint53` | `TestScalarGates`, `TestParsedCountAndStructuralDigestAreStatedBounds` |
| C7 | "`parsed_head_ids:sorted unique string[1..512][0..1024]`" (10744) | sorted-unique | `checkSortedUniqueBoundedStrings` / environ | `TestParsedHeadIDsSweep` |
| C8 | "`workspace_binding:WorkspaceBinding`" (10745) | owner binding | `clonebundle.DecodeWorkspaceBinding` | `TestOwnerDelegation` |
| C9 | "`structural_digest:digest`" (10746) | digest, no derivation (bound) | `scalar.ParseDigest` / `checkDigest` | `TestScalarGates`, stated-bound test |
| C10 | "`evidence_objects:EvidenceObject[0..65536]`" (10747) | count + row shape | `checkEvidenceCount`, row build/decode | `TestEvidenceCountEdges`, census |
| C11 | "`EvidenceObject` contains exactly…" (10748-10751) | 5-member row, closed kind | `buildEvidenceObject`, `decodeEvidenceObject` | `TestMemberCensus*` (evidence), kind oracle |
| C12 | "Evidence rows are sorted unique by evidence kind/blob ID." (10752) | row order | `checkEvidenceOrder`, both entries | `TestEvidenceOrderSweep` |
| C13 | "Staged and live manifests are distinct and cannot be relabeled." (10753-10754) + "Modes cannot be relabeled." (10580) + ReadAuthority purpose (3796-3797) | mode derived from trusted authority, claim must agree | `modeForAuthorityPurpose` + `checkModeAuthority`, both entries | `TestModeAuthorityGrid`, 4 relabel regressions, `TestModeRelabelRefused`, `TestModeBindsIdentity` |
| C14 | "`schema=urn:ax:schema:clone-validation-report`, `schema_version=1.0.0`" (10756-10758) + 20 members (10758-10776) | closed shape, 23 members | `DecodeValidationReport`, `buildValidationReport` renders | `TestMemberCensus*` (report) |
| C15 | "`validation_report_id:digest` under the omission rule" (10759) | JCS identity | `Build…` seals, `verifySelfDigest` | `TestReportIdentityIndependent` |
| C16 | six manifest/fidelity digest refs + operation UUID (10760-10765) + "aggregates both reads" (10755, 10581) | siblings must be sealed reads from the read-back validator; staged/live refs bound to them; plan/projected/provider/fidelity stay opaque (reconciliation owns their pairing) | `checkSiblingsSealed` first + `checkReadRefs`, both entries + scalar gates | `TestBuild/DecodeReportRefusesUnsealedSiblings`, `TestReportEntriesTakeOnlySealedSiblings`, `TestValidatedReadBackSealedConstruction`, `TestReportReadRefRelationOracle`, equal/swap regressions, `TestScalarGates`, `TestDecodeScalarStrings` |
| C17 | expected/observed IDs `string[1..512]` (10766-10767) | bounds; equality decides valid | bound gates + `decideValid` | string edges, valid oracle |
| C18 | "`target_environment:EnvironmentTuple`" (10768) | owner tuple (cross-match is reconciliation) | `sessadapter.DecodeTuple` | `TestOwnerDelegation`, `TestReportTupleMatchIsReconciliationBound` |
| C19 | seven check booleans (10769-10774) | required non-null booleans | `decodeValidationChecks` / Go bools | wrong-type census (null probe), oracle |
| C20 | "`findings:AdapterFinding[0..4096]`" (10775) | owner findings + count | `sessadapter.DecodeFindings(·, 4096)` | `TestFindingsCountEdges`, delegation |
| C21 | "`valid=true` requires every boolean true, matching native IDs/tuple, and no error finding." (10776-10777) + "every applicable check must pass" (10582) | valid iff all-true ∧ IDs equal ∧ no error (tuple cross-match → reconciliation) | `decideValid` + `checkValidAgreement`, both entries | `TestValidWholeDomainOracle` (1536 combos) |
| C22 | "`extensions`" on both shapes + registry URNs at 1.0.0 (170-171) | closed extensions, literal envelope | `clonebundle` extensions owner | `TestExtensionsClosed`, `TestEnvelopeLiterals` |

Member-name derivation note: the pinned text states the
read-back native-ID RULE ("equal expected and observed native
Session IDs") without member names; the member names
`expected/observed_target_native_session_id` follow the
validation-report table (C17) and the read-back adapter success
body (line 3843: `observed_target_native_session_id`), which spell
both. The census exact-set test fails if production drifts from
this spelling.

## Gate × entry table

Every production entry that reaches a gate is its own cell. B =
Build refuses/admits, D = Decode refuses/admits, P = predicate.

| Gate | BuildReadBack | DecodeReadBack | BuildReport | DecodeReport | Predicates |
|---|---|---|---|---|---|
| closed members (A1) | renders exact¹ | refuses | renders exact¹ | refuses | — |
| schema/version literals (A2) | renders | refuses | renders | refuses | — |
| mode vocab (A3) | refuses | refuses | — | — | `ValidReadBackMode` |
| evidence kind vocab (A4) | refuses | refuses | — | — | `ValidEvidenceKind` |
| native-ID equality (A5) | refuses | refuses | decides | decides | — |
| valid rule (A5/A6) | — | — | refuses flipped | refuses flipped | — |
| string bounds chars (A7) | refuses | refuses | refuses | refuses | — |
| heads order (A8) | refuses | refuses (owner) | — | — | — |
| evidence order (A9) | refuses | refuses | — | — | — |
| evidence count (A10) | refuses | refuses | — | — | — |
| findings count (A10) | — | — | refuses (owner max) | refuses (owner max) | — |
| uint53/number (A11) | refuses | refuses (owner type ceiling) | — | — | — |
| scalars (A12) | refuses | refuses | refuses | refuses | — |
| owner nesting (A13) | refuses w/ owner detail | refuses w/ owner detail | refuses w/ owner detail | refuses w/ owner detail | — |
| extensions (A14) | refuses | refuses | refuses | refuses | — |
| identity (A15) | computes | verifies | computes | verifies | — |
| UTF-8 (A16) | refuses | n/a (JSON) | refuses | n/a (JSON) | predicates return false |
| mode authority (A18) | refuses relabel/source | refuses relabel/source | — | — | — |
| read-ref binding (A19) | — | — | refuses equal/swap/missing | refuses equal/swap/missing | — |
| sibling seal (A20) | mints seal | mints seal | refuses unsealed | refuses unsealed | — |

¹ "renders exact" is pinned by `TestMemberCensusExactSets`
(derived set vs sealed bytes), not by a refusal.

## Mutant table (narrowing per gate, killed alone)

Harness: `testdata/mutant_harness.py`. One raw log per plant per
run carries the subprocess exit; the battery runs twice. All rows
below are NARROWING (the gate stays present and admits exactly one
forbidden member) except the marked control. No arm-deletes.

| Mutant | Gate | Admits exactly | Killer (run alone) | Run1 | Run2 |
|---|---|---|---|---|---|
| N-mode-archived | A3 | `archived` mode | `TestModeVocabularyOracle` | KILLED | KILLED |
| N-kind-native | A4 | `native` kind | `TestEvidenceKindVocabularyOracle` | KILLED | KILLED |
| N-identity-len | A5 read-back | same-length-differing IDs | `TestNativeIdentityEqualityOracle` | KILLED | KILLED |
| N-valid-drop-staged | A6 | staged-false + valid=true | `TestValidWholeDomainOracle` | KILLED | KILLED |
| N-valid-drop-live | A6 | live-false + valid=true | `TestValidWholeDomainOracle` | KILLED | KILLED |
| N-valid-drop-marker | A6 | marker-false + valid=true | `TestValidWholeDomainOracle` | KILLED | KILLED |
| N-valid-drop-identity | A6 | identity-false + valid=true | `TestValidWholeDomainOracle` | KILLED | KILLED |
| N-valid-drop-binding | A6 | binding-false + valid=true | `TestValidWholeDomainOracle` | KILLED | KILLED |
| N-valid-drop-resume | A6 | resume-false + valid=true | `TestValidWholeDomainOracle` | KILLED | KILLED |
| N-valid-drop-generation | A6 | generation-false + valid=true | `TestValidWholeDomainOracle` | KILLED | KILLED |
| N-valid-drop-ids | A6 | ID-mismatch + valid=true | `TestValidWholeDomainOracle` | KILLED | KILLED |
| N-valid-drop-error | A6 | error finding + valid=true | `TestValidWholeDomainOracle` | KILLED | KILLED |
| N-heads-dup | A8 Build | duplicated heads | `TestParsedHeadIDsSweep` | KILLED | KILLED |
| N-evidence-dup | A9 | duplicated (kind, blob) | `TestEvidenceOrderSweep` | KILLED | KILLED |
| N-evidence-count | A10 | 65537 rows, both entries | `TestEvidenceCountEdges` | KILLED | KILLED |
| N-findings-count-build | A10 | 4097 findings at Build | `TestFindingsCountEdges` | KILLED | KILLED |
| N-findings-count-decode | A10 | 4097 findings at Decode | `TestFindingsCountEdges` | KILLED | KILLED |
| N-closed-readback | A1 | `zz_extra_member` in shape | `TestMemberCensusUnknownMissing` | KILLED | KILLED |
| N-closed-evidence | A1 | `zz_extra_member` in row | `TestMemberCensusUnknownMissing` | KILLED | KILLED |
| N-closed-report | A1 | `zz_extra_member` in shape | `TestMemberCensusUnknownMissing` | KILLED | KILLED |
| N-schema-neighbor | A2 | sibling URN as schema | `TestEnvelopeLiterals` | KILLED | KILLED |
| N-version-neighbor | A2 | `2.0.0` as version | `TestEnvelopeLiterals` | KILLED | KILLED |
| N-relabel-admit | A15/C13 | docs valid under the other mode (now via the flipped-authority cell) | `TestModeRelabelRefused` | KILLED | KILLED |
| N-parse-count-edge | A11 Build | 2^53 count | `TestScalarGates` | KILLED | KILLED |
| N-mode-auth-build | A18 Build | cross-mode claims at Build | `TestBuildStagedAuthorityRefusesLiveClaim` | KILLED | KILLED |
| N-mode-auth-decode | A18 Decode | resealed relabels at Decode | `TestDecodeStagedAuthorityRefusesResealedLiveClaim` | KILLED | KILLED |
| N-purpose-source | A18 | source_native admitted as staged | `TestModeAuthorityGrid` | KILLED | KILLED |
| N-report-refs-build | A19 Build | equal/swapped refs at Build | `TestBuildReportRefusesEqualReadRefs` | KILLED | KILLED |
| N-report-refs-decode | A19 Decode | equal/swapped refs at Decode | `TestDecodeReportRefusesEqualReadRefs` | KILLED | KILLED |
| N-seal-build | A20 Build | unsealed siblings past the seal gate at Build | `TestBuildReportRefusesUnsealedSiblings` | KILLED | KILLED |
| N-seal-decode | A20 Decode | unsealed siblings past the seal gate at Decode | `TestDecodeReportRefusesUnsealedSiblings` | KILLED | KILLED |
| N-envelope-alias | A1 | `Schema` normalized before closure | `TestMemberCensusMiscased` | KILLED | KILLED |
| N-utf8-lone | A16 | lone-invalid bytes at Build strings | `TestBuildUTF8Reflection` | KILLED | KILLED |
| N-media-bound-build | A7 Build | 129-char media at Build | `TestMediaTypeStringEdges` | KILLED | KILLED |
| N-media-bound-decode | A7 Decode | 129-char media at Decode | `TestMediaTypeStringEdges` | KILLED | KILLED |
| N-heads-item-build | A8 Build | 513-char heads at Build | `TestParsedHeadIDsSweep` | KILLED | KILLED |
| N-heads-count-build | A8 Build | 1025 heads at Build | `TestParsedHeadIDsSweep` | KILLED | KILLED |
| N-heads-item-decode | A8 Decode | 513-char heads at Decode | `TestParsedHeadIDsSweep` | KILLED | KILLED |
| N-heads-count-decode | A8 Decode | 1025 heads at Decode | `TestParsedHeadIDsSweep` | KILLED | KILLED |
| N-scalar-digest-build | A12 Build | `not-a-digest` at Build | `TestScalarGates` | KILLED | KILLED |
| N-scalar-digest-decode | A12 Decode | `not-a-digest` at Decode | `TestDecodeScalarStrings` | KILLED | KILLED |
| N-uuid-build | A12 Build | `not-a-uuid` at Build | `TestScalarGates` | KILLED | KILLED |
| N-uuid-decode | A12 Decode | `not-a-uuid` at Decode | `TestDecodeScalarStrings` | KILLED | KILLED |
| N-control-neutral | control | nothing (comment-only) | full suite | SURVIVED | SURVIVED |
| P-census-entry | A17 plant | ninth export carrying a row | `TestExportedAPIInventory` | RED | RED |
| P-owner-call-removed | A17 plant | (removed owner call) | `TestOwnerReachability` | RED¹ | RED¹ |
| P-forked-literal | A17 plant | re-added owner literal | `TestNoForkedLiterals` | RED | RED |
| P-severity-second | A17 plant | second Severity matcher | `TestErrorSeverityDecidedOnDecodedField` | RED | RED |
| P-guard-literal-list | A1 plant | 10-name literal list | `TestMemberCensusNoLiteralList` | RED | RED |
| P-seal-param-struct | A20 plant | exported struct sibling param | `TestReportEntriesTakeOnlySealedSiblings` | RED | RED |
| P-seal-param-digest | A20 plant | exported digest sibling param | `TestReportEntriesTakeOnlySealedSiblings` | RED | RED |
| P-seal-ctor | A20 plant | third seal constructor | `TestValidatedReadBackSealedConstruction` | RED | RED |
| P-seal-exported-field | A20 plant | exported seal field | `TestValidatedReadBackSealedConstruction` | RED | RED |

¹ red via build failure (the removed call leaves the variable
undefined): the pin depends on the call's presence.

Kill-mechanism honesty: N-closed-*, N-schema-neighbor, and
N-version-neighbor admit past the weakened gate and are then
refused downstream by the identity gate; the killer fails on the
changed refusal literal, which proves the weakened gate's presence
and literal for the admitted member. N-seal-* are the same shape:
the skipped seal gate admits unsealed siblings past it and the
downstream distinctness/slot gate refuses them, so the killer
fails on the changed refusal literal — this proves the seal
gate's presence, position (first), and literal, since without the
gate the probe refuses elsewhere. N-valid-drop-* disagree with
the spec-derived oracle (never with production), which is the kill.
N-mode-auth-* and N-report-refs-* are conjunct drops at one call
site (the N-valid-drop shape): the killed admission is direct, and
the same mutants also redden the twin regression, the driving
oracle, and the `TestOwnerReachability` pin, which is the expected
same-gate collateral. N-utf8-lone is killed by the literal shift
at `ExpectedTargetNativeSession` (the downstream equality refusal
lacks the member literal) and would admit at `media_type`; the
vocab oracles redden under it for the same literal shift.
N-session-bound was WITHDRAWN after it survived the first battery:
a single-member bound widening is masked by the sibling bound gate
under the equality constraint; the edge is proven by
`TestNativeSessionStringEdges` and the conjoining gate by
N-identity-len. N-uint53-decode was WITHDRAWN after it survived
this revision's first battery: the environ owner runs its uint53
type check before the call-site max, so no local widening can
admit 2^53 at Decode; behavior is pinned by `TestScalarGates`
and the Build-side ceiling by N-parse-count-edge. Both
withdrawn logs ride the evidence tar under `withdrawn/`.

## Importer outcome grid

This leaf adds `internal/clonereadback` (new files only) and edits
no landed file: the tracked tree equals the story tip (`5e40826`
= trunk `6d3bff9` + the three signed story checkpoints), so the
five reused owner packages are byte-identical between base and
candidate and their suites are green on the candidate tree — no
input class moved. The grid is keyed (package, entry, input).
Base is the story tip; candidate is the uncommitted tree. The
read-back entries consume `sessadapter.ReadAuthority` values and
mint sealed `ValidatedReadBack` values; the report entries consume
those sealed siblings only — never the exported struct, never
bare digests. No new owner call is added —
`DecodeReadAuthority` runs in the tests to mint genuinely
validated fixtures, mirroring the caller chain.

| Package | Entry | Input classes through this leaf | Base | Candidate | Moved |
|---|---|---|---|---|---|
| `internal/environ` | `DecodeStrictObject` | valid objects; dup-member / trailing-data / non-object docs | green | green | none |
| `internal/environ` | `StringLength` | ASCII + multibyte at every bound | green | green | none |
| `internal/environ` | `CheckStringBounds` | in-bound; over-max / under-min / non-string | green | green | none |
| `internal/environ` | `CheckUint53Bounds` | in-bound; fraction / exponent / string / 2^53 | green | green | none |
| `internal/environ` | `CheckDigest` | valid sha256; malformed | green | green | none |
| `internal/environ` | `CheckUUIDv7` | valid v7; v4 / malformed | green | green | none |
| `internal/environ` | `CheckSortedUniqueStrings` | sorted-unique; unsorted / dup / over-count | green | green | none |
| `internal/scalar` | `ParseDigest` / `ParseUUIDv7` | valid; malformed | green | green | none |
| `internal/scalar` | `SHA256Digest` | canonical omit-self bytes | green | green | none |
| `internal/canonicaljson` | `Canonicalize` | manifest / report / row objects | green | green | none |
| `internal/clonebundle` | `EncodeExtensions` | closed maps; bad keys / values | green | green | none |
| `internal/clonebundle` | `CheckExtensionsClosed` | closed; open / nested-dup | green | green | none |
| `internal/clonebundle` | `OmitSelfDigest` | member maps + self field; tampered claims | green | green | none |
| `internal/clonebundle` | `DecodeWorkspaceBinding` | valid bindings; broken bindings | green | green | none |
| `internal/sessadapter` | `DecodeTuple` | valid tuples; broken tuples | green | green | none |
| `internal/sessadapter` | `DecodeFindings` | valid arrays; broken rows / over-count | green | green | none |
| `internal/clonereadback` | `BuildReadBackEvidenceManifest` | (input, authority): valid manifest; every invalid class in the clause map incl. relabel/source | n/a (new) | green | none (new surface, no importer yet) |
| `internal/clonereadback` | `DecodeReadBackEvidenceManifest` | (bytes, authority): sealed manifest; mutated / hand-built / resealed docs | n/a (new) | green | none |
| `internal/clonereadback` | `BuildValidationReport` | (input, sealed staged, sealed live): valid report; every invalid class incl. flipped valid, ref relations, unsealed siblings | n/a (new) | green | none |
| `internal/clonereadback` | `DecodeValidationReport` | (bytes, sealed staged, sealed live): sealed report; mutated / hand-built docs; unsealed siblings | n/a (new) | green | none |
| `internal/clonereadback` | 4 vocabulary helpers | in-vocabulary; neighbors incl. invalid UTF-8 | n/a (new) | green | none |

No `internal/clonefidelity` or `internal/cloneplan` row: no entry
carries their row shapes (references are digests), pinned by
`TestOwnerRowCensusIsEmpty` + `TestNoOwnerImports`.

## Stated bounds (sibling scope or spec-silent, not waived)

- Report "matching tuple" needs the read manifests' observed
  tuples as cross-inputs; only the single-tuple shape is gated
  here. Owned by the final reconciliation leaf.
- Plan/projected/provider/fidelity pairing across shapes needs
  the wider reconciliation inputs; those digests stay opaque here
  (the staged/live read binding against the trusted siblings IS
  gated here). Final leaf.
- Report/operation coherence across the three shapes (the reads'
  operation vs the report's) is a cross-input pairing; ungated
  here. Final leaf.
- `parsed_event_count` carries no stated derivation from heads or
  rows; any uint53 admits. Spec-silent.
- `structural_digest` carries no stated preimage; any digest
  admits. Spec-silent.
- Blob descriptor agreement (descriptor vs blob ID + byte count)
  needs an object store; these entries take no fetch. Unverified
  here by construction.
- `parsed_event_count` vs `parsed_head_ids` cardinality: no stated
  relation; independent bounds only.
- Authority freshness/expiry: the entries consume the Purpose of
  an already-validated authority the caller chain mints; expiry
  enforcement lives where the chain mints it. Composition bound.
- Seal forgery assumes Go memory safety: outside callers cannot
  mint `ValidatedReadBack` (unexported fields, no exported
  constructor, zero refused — all pinned); `unsafe`/linkname
  forgery is out of scope by construction.
- Owner-inner rules (tuple/binding/finding inner members,
  extension value model, decode-side uint53 type ceiling) are
  delegated to the sessadapter/clonebundle/environ suites; the
  wiring is pinned by the delegation table and the reachability
  census, and a narrowing inside those owners belongs to their
  packages, not to a fork here. Excluded from the measured
  narrowing ratio.
- Single-member native-ID bound narrowings are masked by the
  sibling bound under the equality constraint (see mutant table);
  edges proven directly. Excluded from the measured ratio.
