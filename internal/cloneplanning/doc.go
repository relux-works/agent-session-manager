package cloneplanning

// This package plans the pair-neutral target projection: strategy
// and profile selection, per-item expected dispositions with reason
// sets, checkpoint visible-text authority, historical-tool
// inertness, and source/target usage separation. It is the third
// leaf of the fidelity-projection-and-validation-contracts story:
// planning semantics only. Closed shapes stay owned by the schema
// siblings (internal/clonefidelity dispositions/reasons,
// internal/cloneplan Projection Plan and Projected Object Manifest,
// internal/clonebundle capture/canonical contracts,
// internal/cloneproject G1 normalization); this package decides,
// never re-validates, their shapes.
//
// Authority: relux-works/agent-session-manager-spec@v0.7.0,
// Section 13.14.1 (internal/specdoc/SPEC.v0.7.0.md, lines
// 10385-10393: event kinds, content blocks, "Historical tools are
// inert; incomplete calls become aborted history", "Foreign
// instructions are low-authority history, foreign encrypted/signed
// reasoning is opaque-preserved, and source usage is not target
// accounting") and Section 13.14.2 (lines 10548-10570:
// dispositions, reasons, profiles, strategies, "Continuation
// context is explicitly non-native historical fidelity"), the
// Projection Plan 1.0.0 table (strategy/profile four-lists), the
// projection-plan adapter contract ("archive_only is forbidden",
// line 3841), and Migration Checkpoint VisibleMigrationProjection
// (lines 10808-10812: "authority=user_context", "never an
// assistant reply, system instruction, or authorization",
// "Visible text comes from typed escaped fields and is user
// context, never an assistant reply or control instruction").
//
// Production entries: SelectStrategy, SelectProfile, ClassifyItem,
// PlanItem, PlanSession, ProjectVisibleText, EscapeVisibleText, and
// PlanTargetEffects. Every rule is driven through these entries by
// a whole-domain oracle written in the tests from the derivation in
// TRACEABILITY.md; every gate ships a narrowing mutant killed by
// its named test run alone (see
// testdata/mutant_harness.py). Authority is decided by type, never
// by text: the visible authority is one constant, the item
// classifier admits closed vocabularies only, and the structural
// test pins that shape with control plants.
//
// Reuse, not fork: strategy/profile/disposition/reason/event-kind/
// block vocabularies delegate to internal/clonefidelity,
// internal/cloneplan, and internal/clonebundle; the Section
// 13.14.2 record rules (reason-set coupling, bounds,
// sorted-unique) delegate to clonefidelity.ValidateDispositionRow
// for every caller-supplied and every emitted mapping;
// target-branch
// admission (unstable refusal, maximal-safe completeness) stays
// with clonesnap.AdmitForTarget and the clonebundle G2 gates;
// G1 tool/instruction/usage folds stay with internal/cloneproject
// (this package consumes the completed/aborted and
// encrypted/signed facts it lands); character measures and uint53
// come from internal/environ and internal/scalar. The fact names
// ClassifyItem reads (resolution, protection) follow cloneproject's
// landed registry; value vocabularies are gated here.
//
// Stated bounds (sibling scope or spec-silent, not waived): target
// operation synthesis from dispositions; source-condition reasons
// (source_not_persisted, source_truncated, source_corrupt) and
// target-condition reasons (target_schema_constraint,
// target_size_limit, target_version_gate, credential_excluded,
// secret_policy, derived_index_rebuilt) need inputs the item axis
// does not carry; capability-gated degradation beyond the tool and
// subagent importer classes; per-kind text-field extraction (the
// visible entry takes extracted typed texts); the full Migration
// Checkpoint Build/Decode shape (a future leaf seals it from the
// authority value pinned here); cross-package payload fact-name
// renames past the grep pin. See TRACEABILITY.md.
