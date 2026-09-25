package cloneplan

// This package implements the Section 13.14.2 projection closed
// schemas: Projection Plan 1.0.0 with its component records
// (ProjectionItemMapping, ProjectionTargetOperation,
// ExpectedTargetResource, SynthesizedProjectionEvent,
// TransactionPlan, ReadBackPlan, ResumeProjectionPlan,
// RollbackPlan, ContractRequirement) and Clone Projected Object
// Manifest 1.0.0 with ProjectedObjectEntry. It is the second leaf
// of the fidelity-projection-and-validation-contracts story: closed
// shapes only. Planning semantics (strategy/profile choice,
// checkpoint message authority, inactive tools/instructions, token
// metadata) belong to the sibling planning leaf.
//
// Authority: relux-works/agent-session-manager-spec@v0.7.0, Section
// 13.14.2 "Fidelity, projection, and lineage"
// (internal/specdoc/SPEC.v0.7.0.md, line 10548+), the Projection
// Plan 1.0.0 table, the component schemas that follow it, and the
// Clone Projected Object Manifest 1.0.0 paragraph. Schema URNs come
// from the Section 1 registry table (lines 168-169). Section
// 13.14.4 carries no constraint on these shapes; where the plan or
// manifest references sibling contracts (Migration Checkpoint,
// Read-Back, Validation Report, Lineage Receipt, Bundle Manifest)
// only the identity or digest field is carried.
//
// Every Decode entry validates a closed shape: exact members,
// schema/version literals, bounded scalars, closed vocabularies,
// sorted-unique order, the blob/directory nullability branches,
// operation ordering with the dependency DAG, the closed plan
// constants, and self-digest agreement. Every Build entry seals the
// same bytes deterministically: identical inputs produce
// byte-identical outputs, cross-row order is checked on the sealed
// row set (never repaired), and every Build output decodes clean.
// Refusals name the offending member and never admit a partial
// object.
//
// Reuse, not fork: strict JSON framing, character string measures,
// uint53/digest/UUID bounds, sorted-unique array gates, the
// reverse-DNS grammar, and the SemVer grammar delegate to
// internal/environ; closed extension values and the omit-self
// digest construction delegate to the internal/clonebundle owner,
// which also owns WorkspaceBinding validation and the nine capture
// classes behind security_exclusions; Environment Tuples and
// ResourceLimits decode through internal/sessadapter; the
// disposition and reason vocabularies delegate to the
// internal/clonefidelity sibling; canonical bytes and the SHA-256
// digest come from internal/canonicaljson and internal/scalar. The
// canonicaljson closed-shape registry keeps its
// rejectUnsupportedImmutableObjectShape rows for these schemas
// (that table's census pins every row, so replacing them would fork
// ownership); this package is the validating owner and says so here
// and in TRACEABILITY.md.
//
// Stated bounds (sibling scope or spec-silent, not waived):
// canonical_event_ids equality with Canonical Session order,
// item-mapping coverage ("one for every captured/canonical item"),
// expected-resource and manifest-entry partitioning against plan
// operations, synthesized insertion anchors against canonical IDs,
// manifest total_bytes reconciliation (no derivation rule is
// pinned), synthesized array order (array position IS the insertion
// sequence), required_dispositions key closure (the request owner's
// bound, mirrored exactly), and forbid_reasons admitting literal
// bounded strings. This package performs no durable write: sealing
// is pure bytes, so crash semantics are vacuous and determinism is
// the durability proof (see TRACEABILITY.md).
