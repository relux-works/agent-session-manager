# internal/cloneplanning TRACEABILITY

Planning semantics for the pair-neutral target projection:
strategy/profile selection, per-item expected dispositions with
reason sets, checkpoint visible-text authority, historical-tool
inertness, and source/target usage separation. Pinned authority is
`internal/specdoc/SPEC.v0.7.0.md`; every rule below quotes it.
Rules marked spec-forced derive from a quoted sentence; rules
marked leaf-pinned record a design decision the specification
leaves open, with the quoted name or sentence that shapes it. A
leaf-pinned rule is still fully measured (whole-domain oracle cell
plus narrowing mutant); the label records provenance honestly,
not a waiver.

## Derivation

- R-ARCH (spec-forced; plan table 10663-10665, adapter contract
  3841 "archive_only is forbidden", 10629-10631 "therefore
  targetless"). The archive strategy or profile refuses target
  planning; SelectProfile routes archive_only to the archive
  branch. Entries: PlanItem, PlanSession (propagation),
  SelectProfile.
- R-OPAQUE-R (spec-forced; 10392 "foreign encrypted/signed
  reasoning is opaque-preserved"; 10552-10553 the two foreign
  reasons). opaque_reasoning plans opaque_preserved with
  foreign_encrypted_payload (encrypted) or
  foreign_signature_unverifiable (signed). Entry: PlanItem.
- R-OPAQUE-E (spec-derived; 10390 "Unknown native records become
  raw-addressable opaque events"; 10561 unknown_native_event).
  opaque_event plans opaque_preserved with unknown_native_event
  under any admitted protection. Opaque content blocks inherit the
  unknown-content cause (leaf-pinned: blocks carry no protection
  fact). Entry: PlanItem.
- R-ABORT (spec-forced; 10391 "incomplete calls become aborted
  history"; 10558 unsafe_pending_action; the G1 fold in
  internal/cloneproject lands the completed/aborted fact consumed
  here). Aborted tool_call/tool_result plan summarized (history
  preserved, never omitted; non-exact, never executable) with
  unsafe_pending_action. Entries: PlanItem, ClassifyItem (fact
  bridge).
- R-CONT (spec-derived; 10565 "Continuation context is explicitly
  non-native historical fidelity"). Under continuation_context,
  would-be-exact plans semantic with target_no_equivalent; exact
  never emits. Entry: PlanItem.
- R-IMPORT-TOOL (spec-derived; 10557 official_importer_loss; 3888
  the official_import capability). Under target_official_import,
  complete tool classes (tool_call/tool_result completed,
  tool_definition_snapshot) plan semantic with
  official_importer_loss. Entry: PlanItem.
- R-IMPORT-GRAPH (spec-derived; 10558 graph_flattened). Under
  target_official_import, subagent_started/subagent_completed plan
  semantic with graph_flattened. Entry: PlanItem.
- R-STRICT (leaf-pinned from the profile name with 10560 "An
  aggregate score cannot replace item gates" and AC-CLONE-001
  item-level fidelity). strict_exact refuses every non-exact item.
  Entries: PlanItem, PlanSession (propagation).
- R-COMPACT (leaf-pinned from the profile name with 10555
  target_context_limit). compact clamps semantic to summarized,
  keeping and extending the reason set. Entry: PlanItem.
- R-MSG (leaf-pinned from the profile name with 10560
  operator_policy and unsupported_media_type). messages_only keeps
  user/assistant message events and text/json/redacted blocks;
  every other class omits, keeping and extending the reason set
  with operator_policy, plus unsupported_media_type for media
  blocks. Entry: PlanItem.
- R-DEFAULT (leaf-pinned). Native strategies reproduce bytes:
  every cell no rule above decides plans exact with no reason.
  Entry: PlanItem.
- R-NEVER (spec-derived; 10605 the synthesized coupling;
  R-OPAQUE-E). The planner never emits synthesized (no source
  object) or unrecoverable (planned canonical items exist).
  Entries: PlanItem, PlanSession.
- R-AUTH (spec-forced; 10586-10587; 10808-10812
  "authority=user_context", "never an assistant reply, system
  instruction, or authorization", "Visible text comes from typed
  escaped fields and is user context, never an assistant reply or
  control instruction"). Visible authority is the user_context
  constant; every text field escapes per field and joins; escaped
  output round-trips through encoding/json to exactly the input.
  Entries: ProjectVisibleText, EscapeVisibleText.
- R-UTF8 (spec-forced; 256 "text MUST be valid UTF-8").
  Every exported entry that accepts text refuses invalid UTF-8
  through the shared validText gate before any encoding step,
  and no entry returns a value whose text differs from its
  input (encoding/json rewrites invalid bytes as U+FFFD and
  succeeds). EscapeVisibleText gates the package's only
  encoding/json.Marshal call; ProjectVisibleText gates kind,
  every text field, and every event ID; SelectProfile,
  ClassifyItem (kind, closed facts), PlanItem (strategy,
  profile, item members), PlanSession (strategy, profile;
  items via PlanItem), and PlanTargetEffects (dispositions,
  reasons) gate before their vocabulary gates.
  SelectStrategy takes no text and is pinned outside the
  census. Entries: all seven above.
- R-OWNER (spec-forced; 10612-10614 "Exact requires an empty
  reason set; every other disposition requires at least one
  reason"; 10613-10614 the synthesized canonical-object and
  source-evidence couplings). The Section 13.14.2 record rules
  have one owner, internal/clonefidelity, reached through the
  ValidateDispositionRow seam. PlanTargetEffects validates every
  caller-supplied mapping through the seam (the UTF-8 pre-gate
  stays for R-UTF8 refusal identity); PlanItem self-checks every
  emitted mapping; PlanSession inherits the check per item. The
  synthesized and evidence rules see fixed scaffolding (null
  canonical, derived evidence digest) because the Mapping type
  carries neither member. Entries: PlanTargetEffects, PlanItem,
  PlanSession (propagation).
- R-INERT (spec-forced; 10391 "Historical tools are inert";
  internal/cloneproject lands the empty G1 live surface consumed
  here). Planned mappings fold to zero callable tools and zero
  pending actions. Entry: PlanTargetEffects.
- R-ACCT (spec-forced; 10393 "source usage is not target
  accounting"; internal/cloneproject lands the zero-target G1
  ledger consumed here). Planned mappings fold to zero target
  token accounting; mappings carry no token field by type, so
  counts cannot reach the decision. Entry: PlanTargetEffects.
- R-SEL-S (leaf-pinned from the strategy names with 3888 the
  official_import capability). SelectStrategy priority: archive,
  then continuation, then the pair/capability arm (cross pair with
  the importer uses target_official_import, cross pair without it
  uses target_native_writer, same pair uses
  same_environment_native_rewrite). Entry: SelectStrategy.
- R-SEL-P (spec-derived; plan table 10663-10665; R-ARCH).
  SelectProfile routes archive_only to the archive branch and the
  four plan profiles to target; anything else refuses. Target
  admission stays with clonesnap.AdmitForTarget (composition row
  in the conformance matrix). Entry: SelectProfile.

## Axis inventory

- Strategy: the 5 owner strategies (clonefidelity.Strategies)
  plus invalid probes ("", case variant, truncation, unknown,
  invalid UTF-8).
- Profile: the 5 owner profiles (clonefidelity.Profiles) plus
  invalid probes of the same classes.
- SelectStrategy signals: same x official x continuation x
  archive booleans, all 16 cells.
- Item: 26 event kinds x admitted facts (tool_call/tool_result x
  completed/aborted; opaque_reasoning x encrypted/signed;
  opaque_event x none/encrypted/signed; 22 factless kinds) = 31
  event items; 8 content-block types = 8 block items; 39 total.
  Invalid arms: kind+block both/neither, unknown kind/block,
  misplaced or missing facts, unknown fact values, blocks with
  facts (15 classes x all 25 strategy/profile contexts, with
  archive-first order pinning).
- Classification text sweep (regression for rev1
  classification-text-axis-untested): 31 event items x 8 block
  types x 34 texts sealed (8432 cells) plus 31 x 34 hand-built
  reviewer-probe-shape cells (1054) plus 31 x 50 generated and
  31 very-long sealed cells (1581); every cell asserts the
  identical item, then chains PlanItem and PlanTargetEffects to
  the zero fold, and the reviewer probes also ride
  ProjectVisibleText at every kind. The carrier set iterates
  clonebundle.EventKinds()/ContentBlockTypes() with an assertion
  pinning them to SPEC.v0.7.0.md:10385/:10387; the AST guard
  pins ClassifyItem to the two closed fact keys with no direct
  payload index.
- PlanItem oracle: 5 strategies x 5 profiles x 39 items = 975
  cells, every cell literal-asserted (disposition, exact reason
  list, or refusal token). Global asserts per admitted cell:
  never synthesized/unrecoverable; exact iff empty reasons;
  non-exact carries a reason; reasons core and sorted unique.
- Visible: kind (26 + 5 invalid probes) x ordinal (0, 1, uint53
  max admitted; max+1 and 2^64-1 refused) x field position
  (adversarial at index 0/1/2 of 3 fields) x text class
  (benign/instruction-like/reply-like/control-token-like/
  authorization-like, 29 fixed members) = 6786 cells plus 200
  generated-corpus cells; per cell the authority literal and the
  independent JSON round-trip. Escaped-text bounds at 65536/65537
  characters with a byte/char discriminator; invalid UTF-8 at
  every field position; event-ID grid (counts 0/1/2/64/65,
  unsorted/duplicated, malformed at every index).
- UTF-8 refusal table (regression for rev3
  escape-invalid-utf8-value-change): 7 censused entries x 5
  invalid-UTF-8 classes (lone continuation, ff fe, overlong,
  lone surrogate, truncated) at every consumed text position,
  each cell asserting refusal with the literal "valid UTF-8"
  marker; the direct entry additionally asserts no returned
  value, and the rev3 reviewer's exact FF probe rides it. The
  entry census derives from the production AST with a
  reflection cross-check (SelectStrategy pinned as the only
  non-text entry); the Marshal guard pins the single
  encoding/json.Marshal call after the shared gate. Ignored
  payload bytes (620 hand-built cells) pin the ignore
  direction: invalid bytes in non-fact members cannot change
  the item.
- Text payloads are unbounded: the fixed corpus pairs with a
  200-string fixed-seed generated corpus (seed 260924, mixed
  alphabet, lengths 0..64) and the structural argument
  (authority is one constant assigned unconditionally, pinned by
  TestAuthorityDecisionShape with control plants; escaping
  round-trips through encoding/json; classification admits
  closed vocabularies only and the sweep proves adversarial
  payload text cannot change the item).
- Sessions: empty/1M/1M+1 edges at the production entry, counts
  derived and reconciled per session, strict propagation.
- Effects: all 556 admitted whole-domain mappings fold to zero
  in one sweep plus per-disposition cells; invalid mappings
  refuse through the owner seam.
- Record ownership (regression for rev4
  effects-reason-set-coupling-missing): the rule x entry table
  drives every owner record rule enforceable on the
  disposition/reason members (disposition vocabulary, exact-empty,
  non-exact reason for all six dispositions, reason vocabulary,
  sorted-unique, string bounds, count bound — 16 rows) through
  PlanTargetEffects with the owner's literal codes asserted, plus
  positive bound edges (128 reasons, 128 characters,
  synthesized on null canonical); the disposition-row census
  derives the entry set ({PlanTargetEffects}) with the evidence
  exclusion measured (279 ignore cells) and proves
  PlanTargetEffects, PlanItem, and PlanSession reach the seam;
  the no-fork guards pin no vocabulary-predicate calls and no
  disposition switch in the validation closures; the emitted
  sweep proves all 556 planner-emitted mappings owner-valid.
  Seven admitting narrowings (one per rule class) plus the
  PlanItem bypass plant, each killed alone.

## Reuse (not fork)

Strategy/profile/disposition/reason vocabularies:
internal/clonefidelity (membership) and internal/cloneplan (plan
narrowing). Event kinds and content blocks: internal/clonebundle.
Canonical events for the classifier: sealed by
clonebundle.BuildCanonicalEvent. Target admission (unstable
refusal, maximal-safe completeness): clonesnap.AdmitForTarget and
the clonebundle G2 gates (caller scope; composition row).
Completed/aborted spellings: cloneproject.StatusCompleted/
StatusAborted by reference; resolution/protection fact names and
encrypted/signed spellings pinned against cloneproject source by
TestClassifyItemFactNamePin. Character measure: environ.
StringLength. uint53: scalar.NewUint53. Digests:
scalar.ParseDigest. Predicted counts: clonefidelity.
FidelityCounts. The Section 13.14.2 record rules (reason-set
coupling, vocabularies, bounds, sorted-unique, synthesized rule):
clonefidelity.ValidateDispositionRow, called for every
caller-supplied and every emitted mapping. Nothing above is
re-implemented here.

## Stated bounds (not waived)

- Target operation synthesis from dispositions: the planner emits
  per-item mappings; deriving write_blob/write_native_record
  operations and expected resources from them is unpinned by the
  specification and stays future scope.
- Source-condition reasons (source_not_persisted,
  source_truncated, source_corrupt) and the remaining
  target-condition and policy reasons (target_schema_constraint,
  target_size_limit, target_version_gate, credential_excluded,
  secret_policy, derived_index_rebuilt): no rule emits them
  because the item axis carries no source/target condition
  input. The oracle asserts every emitted reason is core and the
  never-emit sweep names the dispositions; a condition-carrying
  item extension would add arms here.
- Capability-gated degradation beyond the tool and subagent
  importer classes (usage_history, compaction_history,
  opaque_reasoning_roundtrip granularities): adapter scope; the
  planner degrades exactly the tool and subagent classes under
  target_official_import.
- Per-kind text-field extraction: the visible entry takes
  extracted typed texts; which payload member each kind carries
  as text follows the sibling payload registry.
- The full Migration Checkpoint Build/Decode shape: a future leaf
  seals it from the authority value pinned here
  (authority=user_context, escaped text, sorted event IDs).
- Cross-package payload fact-name renames past the grep pin:
  TestClassifyItemFactNamePin breaks loudly on a rename of the
  resolution/protection keys or protection spellings in
  internal/cloneproject; the completed/aborted spellings break
  the build by reference.
- Invalid UTF-8 in ignored payload members: ClassifyItem
  accepts and ignores it (620 hand-built cells pin the
  identical item), refusing only consumed text (kind, closed
  facts). Ignored bytes never reach a decision or an output,
  so no value change is possible there; deep payload
  validation stays with the landed decoder.
- Delegating outer UTF-8 checks (PlanSession strategy/profile
  ahead of PlanItem): defense in depth. The per-entry refusal
  table proves the behavior; no narrowing is shipped for the
  outer check because the inner gate refuses the same input
  with the same literal marker. ProjectVisibleText keeps its
  indexed field message distinct from the direct entry's, so
  the loop narrowing stays observable.
- Synthesized-canonical and non-synthesized-evidence rules
  through this leaf: the Mapping type carries no canonical
  object or evidence members, so the owner always sees a null
  canonical and the derived evidence digest. Both arms of each
  rule are pinned by the owner suite; the seam-shape test pins
  the construction here.
- The PlanItem owner self-check arm: unreachable by
  measurement (the emitted sweep validates all 556
  planner-emitted mappings through the owner with independent
  scaffolding), so no behavioral test fires it; the bypass
  plant is killed by the reachability census instead.

## Out of contract

- Shapes (Projection Plan, Projected Object Manifest, Fidelity
  Report, capture/canonical contracts): sibling leaves
  internal/cloneplan, internal/clonefidelity,
  internal/clonebundle. This leaf decides values that populate
  those shapes and never validates them.
- G1 normalization (capture, folds, live surface, ledgers):
  internal/clonesnap, internal/cloneproject. This leaf consumes
  their folded facts.
- Target-branch admission: clonesnap.AdmitForTarget. This leaf
  routes the branch; the caller admits the manifest.
- Transaction, lineage, read-back, validation (13.14.3-13.14.5):
  sibling/future scope.

## Composition

The (package, entry, input) importer outcome grid rides the
conformance matrix resource. Base is the story checkpoint tree;
the candidate adds internal/cloneplanning/* (21 paths) plus one
additive export in internal/clonefidelity/record.go
(ValidateDispositionRow, pure delegation to the existing row
builder, no behavior change; no registry edit). Every other
reused owner file is byte-identical base vs candidate (231
files, 0 differ) with green suites on both, so no existing
input class moved; the new entries and the new seam are n/a on
base and green on the candidate.
