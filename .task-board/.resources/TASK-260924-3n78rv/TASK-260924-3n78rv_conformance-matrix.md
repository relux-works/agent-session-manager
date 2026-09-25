# TASK-260924-3n78rv conformance matrix: gate x entry census

Legend: SS = `SelectStrategy`, SP = `SelectProfile`, CI =
`ClassifyItem`, PI = `PlanItem`, PS = `PlanSession`, PV =
`ProjectVisibleText`, EV = `EscapeVisibleText`, TE =
`PlanTargetEffects`. Every cell names the killer test(s) run ALONE
plus the killed narrowing mutant(s). Shared = one implementation,
all call sites (checkItem serves PI and CI; the both-entries
killer drives both). Axes: strategy/profile/item whole-domain
oracle (975 literal cells); classification all-kinds sweep
(8432 sealed + 1054 hand-built + 1581 generated/long, rev2
regression for classification-text-axis-untested); visible
adversarial sweep (6986 cells); session edges (empty/1M/1M+1 at
the entry); effects sweep (556 mappings); AST authority-shape
guard and AST classify-fact guard, each with control plants;
rev4 UTF-8 refusal table (7 entries x 5 classes) with an AST
entry census and a Marshal-after-validator guard, each with
control plants; rev5 record-owner rule x entry table (16 rows)
with a disposition-row census, reachability pin, and no-fork
guards, each with control plants.

## Selection

| Gate | SS | SP | Killer | Narrowing |
|---|---|---|---|---|
| 16 signal cells, archive > continuation > pair/capability | x | | TestSelectStrategyOracle | N-select-archive-priority, N-select-official |
| Archive routes archive; 4 plan profiles route target; else refuses | | x | TestSelectProfileOracle | N-select-profile-archive, N-select-profile-unknown |
| Every selected strategy/profile inside owner vocabs; target == plan profiles | x | x | TestSelectionAgreement | (membership supplement; literals above) |
| Invalid UTF-8 refuses with the literal marker (rev4 R-UTF8) | (no text; census-excluded) | x | TestSelectProfileRefusesInvalidUTF8 | N-select-profile-utf8 |

## Item classification

| Gate | CI | PI | Killer | Narrowing |
|---|---|---|---|---|
| Exactly one of kind/block set | x | x | TestPlanItemInvalidArms, TestSharedItemVocabBothEntries | (shape; both/neither probes) |
| Kind vocabulary (shared) | x | x | TestSharedItemVocabBothEntries | N-kind-vocab |
| Block vocabulary (shared) | | x | TestSharedItemVocabBothEntries | N-block-vocab |
| Resolution vocabulary completed/aborted (shared) | x | x | TestSharedItemVocabBothEntries | N-resolution-vocab |
| Protection vocabulary encrypted/signed (shared) | x | x | TestSharedItemVocabBothEntries | N-protection-vocab |
| Resolution required for tool kinds | x | x | TestClassifyItemRefusals | N-classify-resolution-required |
| Protection required for opaque_reasoning | x | x | TestClassifyItemRefusals | N-classify-protection-required |
| Facts must be strings; null maps to absent | x | | TestClassifyItemRefusals | N-classify-fact-type |
| 31 valid event items from sealed canonical events | x | | TestClassifyItemValidGrid | (admission direction; refusals above) |
| Adversarial payload text never changes the item (every kind x block x text, sealed + hand-built + downstream chain) | x | | TestClassifyItemIgnoresPayloadTextAllKinds (TestClassifyItemIgnoresPayloadText remains as a 5-carrier smoke test) | N-classify-text-subagent (reviewer exact plant), N-classify-text-usage; N-classify-fact-type (value arm) |
| Carrier set equals the spec closed lists | x | | TestClassifyItemVocabMatchesSpec | (assertion; SPEC.v0.7.0.md:10385/:10387 pin, no hand list) |
| ClassifyItem reads closed facts only, no direct payload index | x | | TestClassifyItemReadsClosedFactsOnly | control plants (direct index reddens, comment passes) |
| Fact names/spellings follow cloneproject registry | x | | TestClassifyItemFactNamePin | (composition pin; statuses by reference) |
| Consumed text (kind, closed facts) refuses invalid UTF-8; ignored invalid bytes cannot change the item (rev4 R-UTF8) | x | | TestClassifyItemRefusesInvalidUTF8, TestClassifyItemIgnoresInvalidPayloadBytes | (shared validText gate; measured by N-escape-utf8-fffe, N-select-profile-utf8, N-visible-utf8) |

## Planning oracle

| Gate | PI | PS | Killer | Narrowing |
|---|---|---|---|---|
| 975-cell whole-domain oracle vs literals | x | | TestPlanItemWholeDomainOracle | (battery; rule tests below per gate) |
| Group census: 26 kinds, 8 blocks, 5+5, 39 items | x | | TestPlanItemGroupCensus | (partition pin; additions break loudly) |
| Strategy vocabulary | x | | TestPlanItemInvalidArms | N-strategy-vocab |
| Profile overlay default refuses unknown | x | | TestPlanItemInvalidArms | N-profile-vocab |
| Archive strategy refuses (exact cell) | x | | TestPlanItemArchiveRule | N-archive-strategy |
| Archive profile refuses with archive-branch message | x | | TestPlanItemArchiveRule | N-archive-profile |
| Invalid items refuse at all 25 contexts, archive first | x | | TestPlanItemInvalidArms | N-kind/block/resolution/protection-vocab |
| Opaque reasoning reason by protection | x | | TestPlanItemOpaqueRule | N-opaque-reason |
| Opaque event/block unknown-native reason | x | | TestPlanItemOpaqueRule | N-opaque-event-reason |
| Aborted tools are summarized history | x | | TestPlanItemAbortRule | N-abort-drop |
| Aborted history carries unsafe_pending_action | x | | TestPlanItemAbortRule | N-abort-reason |
| Continuation never exact | x | | TestPlanItemContinuationRule | N-continuation-exact |
| Continuation reason target_no_equivalent | x | | TestPlanItemContinuationRule | N-continuation-reason |
| Importer tool loss | x | | TestPlanItemImporterRule | N-importer-tool |
| Importer graph flattening | x | | TestPlanItemImporterRule | N-importer-graph |
| Strict refuses non-exact | x | | TestPlanItemStrictRule | N-strict-admit |
| Compact clamps semantic | x | | TestPlanItemCompactRule | N-compact-skip |
| Compact appends target_context_limit | x | | TestPlanItemCompactRule | N-compact-reason |
| Messages_only survivor set | x | | TestPlanItemMessagesRule | N-messages-admit |
| Omitted carries operator_policy | x | | TestPlanItemMessagesRule | N-messages-reason |
| Media names unsupported_media_type | x | | TestPlanItemMessagesRule | N-media-reason |
| Reasons emit sorted | x | | TestPlanItemMessagesRule | N-reason-order |
| Never synthesized/unrecoverable; exact iff empty reasons | x | | TestPlanItemWholeDomainOracle | (global asserts; N-*-reason rows) |
| Emitted values inside owner vocabs | x | | TestPlanItemAgreement | (membership supplement; literals above) |
| Session counts derive and reconcile | | x | TestPlanSessionCounts | N-count-swap |
| Empty and 1M+1 sessions refuse | | x | TestPlanSessionRefusals | N-session-empty, N-session-bound |
| 1M items plan at the entry | | x | TestPlanSessionMillionEdge | (upper edge admission) |
| Invalid UTF-8 refuses with the literal marker at strategy/profile/item (rev4 R-UTF8) | x | | TestPlanItemRefusesInvalidUTF8 | (shared validText gate; measured by N-escape-utf8-fffe, N-select-profile-utf8, N-visible-utf8) |
| Invalid UTF-8 refuses with the literal marker at strategy/profile/items (rev4 R-UTF8) | | x | TestPlanSessionRefusesInvalidUTF8 | (outer checks are defense in depth; inner PlanItem gate measured above) |

## Authority and visible text

| Gate | PV | EV | Killer | Narrowing |
|---|---|---|---|---|
| Authority always user_context over 6986 adversarial cells | x | | TestProjectVisibleWholeDomain | N-visible-authority |
| Escaped round-trips to exactly the input via encoding/json | x | x | TestProjectVisibleWholeDomain, TestProjectVisibleEscapingOracle | N-visible-escape-suffix |
| Encoder choice pinned (HTML escapes) | | x | TestProjectVisibleEscapingOracle | N-visible-escape-suffix |
| No raw control byte crosses | | x | TestProjectVisibleEscapingOracle | (behavioral; suffix row) |
| Escaped_text 65536/65537 chars; empty refuses | x | | TestProjectVisibleBounds | N-visible-max, N-visible-empty |
| Character measure, not bytes | x | | TestProjectVisibleBounds | (discriminator cell) |
| Ordinal uint53 edges | x | | TestProjectVisibleOrdinalEdges | N-visible-ordinal |
| Kind vocabulary | x | | TestProjectVisibleKindRefusals | N-visible-kind |
| UTF-8 per field with index named (rev4: shared gate; indexed message kept distinct from the direct entry) | x | | TestProjectVisibleTextRefusals | N-visible-utf8 |
| Direct escaping entry refuses invalid UTF-8 and returns no value (rev4 regression for escape-invalid-utf8-value-change) | | x | TestEscapeVisibleTextRefusesInvalidUTF8 | N-escape-utf8-fffe |
| Kind and event IDs refuse invalid UTF-8 (rev4 R-UTF8) | x | | TestProjectVisibleTextRefusesInvalidUTF8 | (shared validText gate; measured by the rows above) |
| Event IDs sorted unique digests[1..64] | x | | TestProjectVisibleEventIDGrid | N-visible-ids-count0, N-visible-ids-count65, N-visible-ids-order, N-visible-ids-digest |
| Escaping/authority position-independent | x | | TestProjectVisiblePositionIndependence | (structural sweep) |
| Authority assigned once, unconditionally, from the constant | x | | TestAuthorityDecisionShape | control plants (conditional, foreign literal) |

## Target effects

| Gate | TE | Killer | Narrowing |
|---|---|---|---|
| 556 planned mappings fold to zero; per-disposition cells | x | TestPlanTargetEffectsZeroSweep | N-effects-nonzero |
| Exact/non-exact reason-set coupling (rev5 regression for effects-reason-set-coupling-missing; reviewer's exact cells) | x | TestPlanTargetEffectsRefusesMalformedReasonSets | N-effects-exact-reasons, N-effects-nonexact-bare |
| Record rules x entry: disposition/reason vocabulary, sorted-unique, string bounds, count bound, owner literals + positive edges | x | TestPlanTargetEffectsOwnerRecordRules, TestOwnerRuleTableProbeHygiene | N-effects-disposition, N-effects-reason, N-effects-order-skip, N-effects-reason-bounds, N-effects-reason-count |
| Emitted mappings owner-valid over the whole domain (independent scaffolding) | x | TestPlanItemEmittedMappingsOwnerValid | (agreement sweep; N-*-reason rows) |
| Seam construction: verbatim pass-through, null canonical/locator, derived evidence | x | TestOwnerRowScaffoldingShape | (construction pin) |
| Disposition/reason/order admission direction (probes kept) | x | TestPlanTargetEffectsRefusals | (killers shared with the rows above) |
| PlanSession output folds to zero through the real chain | x | TestPlanTargetEffectsChained | (composition) |
| Invalid UTF-8 refuses with the literal marker at disposition/reason (rev4 R-UTF8) | x | TestPlanTargetEffectsRefusesInvalidUTF8 | (shared validText gate; measured by N-escape-utf8-fffe, N-select-profile-utf8, N-visible-utf8) |

## UTF-8 entry census (rev4)

| Gate | Killer | Narrowing / control |
|---|---|---|
| Census from the production AST equals the refusal-table key set; reflection cross-check proves the classification; SelectStrategy pinned as the only non-text entry | TestExportedTextEntryCensus | control plants (new entry reddens at the reflection linkage, then at the table linkage; revert passes byte-identical) |
| The single encoding/json.Marshal call runs after the shared validText gate | TestJSONMarshalFollowsValidator | control plants (second site reddens; removed gate reddens); pre-fix shape reddens the escape killer |

## Record-owner delegation (rev5)

| Gate | Killer | Narrowing / control |
|---|---|---|
| Disposition-row census derives exactly {PlanTargetEffects}; evidence exclusion measured (SourceEvidence pins + 279 ignore cells) | TestDispositionEntryCensus, TestClassifyItemIgnoresEvidenceReasons | control plant (new row entry reddens at the list linkage; revert passes byte-identical) |
| PlanTargetEffects, PlanItem, PlanSession reach the owner seam on the static call graph | TestDelegationReachesOwner | N-planitem-owner-bypass; control plant (removed self-check reddens; revert passes byte-identical) |
| No disposition/reason vocabulary predicate calls package-wide | TestNoDispositionVocabularyFork | control plant (re-added call reddens; revert passes byte-identical) |
| No disposition switch in the PlanTargetEffects/PlanItem validation closures (PlanSession counting is derivation, oracle-pinned) | TestNoDispositionSwitchInValidationClosure | control plant (closure switch reddens; revert passes byte-identical) |

## Importer outcome grid (package, entry, input): base vs candidate

Base is the story checkpoint tree (`618d78d`, on trunk
`6d3bff9`); candidate is the uncommitted tree adding
`internal/cloneplanning/*` (21 paths) plus one additive export
in `internal/clonefidelity/record.go`
(`ValidateDispositionRow`, +15/-0, pure delegation). Every other
reused owner file is byte-identical base vs candidate (231
files, 0 differ; see `importer-grid.log`) with green suites on
both (see `owners.log`), so no existing input class moved. New
entries and the new seam are n/a on base and green on the
candidate.

| Package | Entry | Input classes through this leaf | Base | Candidate | Moved |
|---|---|---|---|---|---|
| `internal/clonefidelity` | `ValidStrategy` | 5 strategies; unknown/empty/case/invalid-UTF-8 | green | green | none |
| `internal/clonefidelity` | `ValidProfile` | 5 profiles (census/oracle domains) | green | green | none |
| `internal/clonefidelity` | `ValidateDispositionRow` (new seam, rev5) | 16 rule-table rows + 7 positive edges + 556 emitted mappings + refusal probes | n/a (new) | green | none (new surface, additive) |
| `internal/clonefidelity` | `ValidDisposition` | reached only owner-internally via the seam (no direct leaf call; guard-pinned) | green | green | none |
| `internal/clonefidelity` | `ValidReasonCode` | reached only owner-internally via the seam (no direct leaf call; guard-pinned) | green | green | none |
| `internal/clonefidelity` | `IsCoreReasonCode` | emitted reasons (agreement) | green | green | none |
| `internal/clonefidelity` | `FidelityCounts` | count folds incl. 1M edge | green | green | none |
| `internal/cloneplan` | `ValidPlanProfile` | 5 profiles + invalids (SelectProfile gate) | green | green | none |
| `internal/clonebundle` | `ValidEventKind` | 26 kinds + invalids (item + visible gates) | green | green | none |
| `internal/clonebundle` | `EventKinds` | oracle/classifier/sweep domains | green | green | none |
| `internal/clonebundle` | `ValidContentBlockType` | 8 types + invalids (item gate) | green | green | none |
| `internal/clonebundle` | `ContentBlockTypes` | oracle domain | green | green | none |
| `internal/clonebundle` | `BuildCanonicalEvent` | 31+ classifier fixtures incl. adversarial payloads | green | green | none |
| `internal/clonebundle` | `DecodeCanonicalEvent` | sealed fixtures | green | green | none |
| `internal/cloneproject` | `StatusCompleted`/`StatusAborted` | resolution spellings by reference | green | green | none |
| `internal/cloneproject` | fact registry (source pin) | resolution/protection names + encrypted/signed | green | green | none |
| `internal/clonesnap` | `AdmitForTarget` | branch composition (caller scope; suite-cited) | green | green | none |
| `internal/environ` | `StringLength` | ASCII + multibyte at the 65536 bound | green | green | none |
| `internal/scalar` | `NewUint53` | ordinals 0/1/max/max+1/2^64-1 | green | green | none |
| `internal/scalar` | `ParseDigest` | valid digests; malformed at every ID index | green | green | none |
| `internal/cloneplanning` | `SelectStrategy` | 16 signal cells | n/a (new) | green | none (new surface) |
| `internal/cloneplanning` | `SelectProfile` | 5 profiles + 6 invalids + 5-class UTF-8 refusal | n/a (new) | green | none (new surface) |
| `internal/cloneplanning` | `ClassifyItem` | 31 sealed events + refusals + all-kinds adversarial-text sweep (8432 sealed + 1054 hand-built + 1581 generated/long) + 5-class UTF-8 refusal at kind/facts + 620 ignored-bytes cells | n/a (new) | green | none (new surface) |
| `internal/cloneplanning` | `PlanItem` | 975 oracle cells + invalid arms + 5-class UTF-8 refusal at strategy/profile/item + emitted owner self-check (556 sweep) | n/a (new) | green | none (new surface) |
| `internal/cloneplanning` | `PlanSession` | counts/refusals/1M edge + 5-class UTF-8 refusal at strategy/profile/items | n/a (new) | green | none (new surface) |
| `internal/cloneplanning` | `ProjectVisibleText` | 6986 adversarial cells + bounds/ID grid + 5-class UTF-8 refusal at kind/texts/IDs | n/a (new) | green | none (new surface) |
| `internal/cloneplanning` | `EscapeVisibleText` | edges + round-trip oracle + 5-class UTF-8 refusal + reviewer-exact FF probe | n/a (new) | green | none (new surface) |
| `internal/cloneplanning` | `PlanTargetEffects` | 556 mappings + refusals + chain + 5-class UTF-8 refusal at disposition/reason + 16-row owner rule table (rev5) | n/a (new) | green | none (new surface) |

## Axis inventory

- Selection: 16 signal cells; 5 profiles + 6 invalid probes.
- Item: 39 valid items over owner vocabularies; 15 invalid
  classes x 25 contexts with archive-first order pinning.
- Classification text: 31 event items x 8 block types x 34
  texts sealed (8432) + hand-built (1054) + generated/long
  (1581); identical-item assert plus PlanItem/PlanTargetEffects
  zero chain per cell; domain derived from production
  vocabularies pinned to SPEC.v0.7.0.md:10385/:10387; AST
  closed-fact guard with control plants.
- Planning: 975 literal oracle cells; never-emit, exact-empty,
  reason-sorted asserts per admitted cell.
- Visible: 6986 adversarial cells (26 kinds x 3 ordinals x 3
  field positions x 29 texts + 200 generated); authority
  literal + independent JSON round-trip per cell; bounds,
  ordinal, kind, UTF-8, and event-ID refusal grids; AST
  authority-shape guard with control plants.
- Sessions/effects: counts reconcile; empty/1M/1M+1 at the
  entry; 556 mappings fold to zero; invalid mappings refuse
  through the owner seam with owner literals.
- UTF-8 (rev4): 7 censused entries x 5 invalid classes at every
  consumed position (96 cells + reviewer-exact FF probe), each
  refusing with the literal marker; AST entry census with a
  reflection cross-check; Marshal-after-validator guard; 620
  ignored-bytes cells.
- Record ownership (rev5): 16-row rule x entry table with
  probe hygiene + 7 positive edges; disposition-row census
  ({PlanTargetEffects}) with measured evidence exclusion (279
  ignore cells); reachability pin for 3 entries; 2 no-fork
  guards; 556-mapping emitted-valid sweep; 4 control plants
  with byte-identical reverts.
- Battery: 57 narrowing rows + 1 neutral control SURVIVED,
  three full runs with one raw log per plant per run.
