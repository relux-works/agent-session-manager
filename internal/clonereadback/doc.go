package clonereadback

// This package seals Clone Read-Back Evidence Manifest 1.0.0 and
// Clone Validation Report 1.0.0: the two closed validation
// contracts of Section 13.14.2. It is the fourth leaf of the
// fidelity-projection-and-validation-contracts story: read-back and
// validation shapes only. Reconciliation completeness (cross-input
// coverage, tuple cross-match, partitioning) and the Story
// registry belong to the final leaf; this package gates each shape
// closed and internally coherent, never across inputs.
//
// Authority: relux-works/agent-session-manager-spec@v0.7.0,
// internal/specdoc/SPEC.v0.7.0.md Section 13.14.2 (lines
// 10737-10754: "Clone Read-Back Evidence Manifest 1.0.0 contains
// exactly its schema/version,
// <code>read_back_evidence_manifest_id</code> under the omission
// rule, <code>operation_id:UUIDv7</code>,
// <code>mode:staged|live</code>, ... equal expected and observed
// native Session IDs, ... Evidence rows are sorted unique by
// evidence kind/blob ID. Staged and live manifests are distinct
// and cannot be relabeled."; lines 10755-10777: "Clone Validation
// Report 1.0.0 contains exactly
// <code>schema=urn:ax:schema:clone-validation-report</code>, ...,
// <code>valid:boolean</code>, and <code>extensions</code>.
// <code>valid=true</code> requires every boolean true, matching
// native IDs/tuple, and no error finding."), the prose at lines
// 10580-10582 ("Modes cannot be relabeled. ... every applicable
// check must pass."), the schema registry (lines 170-171: the two
// URNs at 1.0.0), and the nested closed shapes the contracts reuse:
// EnvironmentTuple (lines 3877-3883), NativeIdentity-adjacent
// WorkspaceBinding (lines 10407-10415), and AdapterFinding (lines
// 3813-3817).
//
// Production entries: BuildReadBackEvidenceManifest,
// DecodeReadBackEvidenceManifest, BuildValidationReport,
// DecodeValidationReport, and the two vocabulary predicates
// ValidReadBackMode and ValidEvidenceKind. The read-back entries
// take the trusted read authority the caller threads from its
// authority chain and derive the expected mode from it; each
// mints a sealed ValidatedReadBack the report entries aggregate,
// and the report entries refuse any sibling that is not sealed.
// The carried references bind to the sealed reads in their own
// slots. Every rule is driven through these entries by the
// whole-domain oracles in the tests; every gate ships a narrowing
// mutant killed by its named test run alone (see
// testdata/mutant_harness.py).
//
// Reuse, not fork: Environment Tuples decode through
// internal/sessadapter, Workspace Bindings through
// internal/clonebundle, Adapter Findings through
// internal/sessadapter, strict objects / string measures / uint53 /
// sorted-unique arrays through internal/environ, digests and UUIDs
// through internal/scalar, JCS and the omit-self identity through
// internal/canonicaljson and internal/clonebundle. No Fidelity
// Report, disposition/reason, or Projection Plan data crosses these
// entries — the shapes carry opaque digest references only — so no
// clonefidelity or cloneplan validator is reachable here by
// construction; the owner census pins that empty set and the
// structural guard pins that no second local implementation of
// their rules exists.
//
// Stated bounds (sibling scope or spec-silent, not waived): the
// report tuple cross-match against the read manifests' observed
// tuples and the plan/projected/fidelity pairing need the wider
// reconciliation inputs and belong to the final reconciliation
// leaf (the read-reference identity/slot binding against the two
// trusted sibling reads is gated here); parsed count and
// structural digest carry no stated derivation, so any uint53 /
// digest admits; blob descriptor agreement needs an object store
// this package takes no fetch for; member-name choice for the two
// native Session IDs in the read-back manifest (the pinned text
// names the rule — "equal expected and observed native Session
// IDs" — without member names) follows the validation-report and
// adapter-body spelling expected/observed_target_native_session_id.
// See TRACEABILITY.md.
