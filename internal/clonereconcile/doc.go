// Package clonereconcile proves reconciliation completeness for the
// fidelity-projection-and-validation story (STORY-260830-21bxa3).
//
// Authority: internal/specdoc/SPEC.v0.7.0.md Section 13.14.2 lines
// 10554-10556: "Every captured candidate reconciles once to raw
// evidence or exclusion; every raw item to a canonical item or
// normalization disposition; every canonical item to staged/live
// target evidence or a target disposition."
//
// The package owns only the cross-tier link census and the
// target-evidence pairing. Every shape, vocabulary, digest, tuple,
// and report rule delegates to its landed owner:
//
//   - read-back history: internal/clonereadback (ValidatedReadBack
//     only; this package never mints, copies fields out of, or
//     re-validates a read)
//   - fidelity rows and report: internal/clonefidelity
//     (ValidateDispositionRow, BuildFidelityReport,
//     DecodeFidelityReport)
//   - plan and projected manifest pairing:
//     internal/cloneplan (DecodeProjectionPlan,
//     DecodeProjectedObjectManifest)
//   - validation report and its valid bit: internal/clonereadback
//     (BuildValidationReport; the valid bit is decided there)
//   - capture classes: internal/clonebundle (ValidCaptureClass)
//   - digests: internal/scalar (ParseDigest)
//   - tuples: internal/sessadapter (DecodeTuple)
//   - character string measure: internal/environ (StringLength)
//
// The two production entries are ReadBackHistory (decode staged and
// live target history through the read-back owner) and Reconcile
// (census all three tiers, pair target evidence against the sealed
// reads, and derive both reports through their owners). Both are
// pure: no durable write, no clock, no network. Every refusal
// carries a literal clonereconcile: code naming the tier and the
// broken exactly-once shape.
package clonereconcile
