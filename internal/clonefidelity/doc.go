package clonefidelity

// This package implements the Section 13.14.2 fidelity contracts:
// the disposition, reason, profile, and strategy vocabularies,
// FidelityCounts, FidelityDispositionRecord, and Fidelity Report
// 1.0.0 with its reconciliation rules. It is the first leaf of the
// fidelity-projection-and-validation-contracts story.
//
// Authority: relux-works/agent-session-manager-spec@v0.7.0, Section
// 13.14.2 "Fidelity, projection, and lineage"
// (internal/specdoc/SPEC.v0.7.0.md, line 10548+). Section 13.14.4
// carries no constraint on the report shape; where the report
// references sibling contracts (Projection Plan, Read-Back,
// Validation Report, Migration Checkpoint, Lineage Receipt) only
// the identity or digest field is carried, and those contracts
// belong to sibling leaves.
//
// Every Decode entry validates a closed shape: exact members,
// bounded scalars, closed vocabularies, sorted-unique order, the
// archive/target nullability branches, row ordering, aggregate
// reconciliation, and self-digest agreement. BuildFidelityReport
// constructs the same bytes deterministically: identical inputs
// produce byte-identical outputs, counts and reason counts are
// derived from the rows (never accepted), and every Build output
// decodes clean. Refusals name the offending member and never admit
// a partial object.
//
// Reuse, not fork: strict JSON framing, character string measures,
// uint53/digest/UUID bounds, and sorted-unique array gates delegate
// to internal/environ; the reverse-DNS grammar delegates to
// environ.CheckReverseDNS; Environment Tuples decode through
// sessadapter.DecodeTuple; canonical bytes and the SHA-256 digest
// come from internal/canonicaljson and internal/scalar; closed
// extension values, the omit-self digest construction, event kinds,
// content-block types, and capture classes come from the
// internal/clonebundle owner. The canonicaljson closed-shape
// registry keeps its rejectUnsupportedImmutableObjectShape row for
// this schema (that table's census pins every row, so replacing it
// would fork ownership); this package is the validating owner and
// says so here and in TRACEABILITY.md.
//
// Stated bounds (sibling scope, not waived): cross-artifact row
// coverage ("every Capture Manifest item and Canonical Event occurs
// in exactly one non-synthesized row") needs the manifests as
// inputs and no entry in this leaf takes them; source-evidence
// resolution (rows trace to captured bytes) likewise; byte_counts
// values carry no row reconciliation because rows carry no byte
// measure (only the exact seven-disposition key set and uint53
// values are gated); the core-derived booleans admit any value
// except the pinned archive rule (both target booleans false);
// per-row target evidence in archive scope is admitted as shaped;
// forbid_reasons admits literal bounded strings (the table row
// types them as strings, not reason codes).
// This package performs no durable write: sealing is pure bytes,
// so crash semantics are vacuous and determinism is the durability
// proof (see TRACEABILITY.md).
