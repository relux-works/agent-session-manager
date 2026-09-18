package clonebundle

// This package implements the Section 13.14.1 clone capture contracts:
// Clone Raw Object Manifest 1.0.0, Clone Capture Manifest 1.0.0 (with
// Capture Items, Capture Source Basis, and Capture Boundary), Canonical
// Session 1.0.0, Canonical Event 1.0.0 (with Source Evidence and raw
// references), the NativeIdentity and WorkspaceBinding identities, raw
// evidence descriptor agreement over Section 10.2 Blob Descriptors,
// reverse-DNS extension keys, and immutable source generations.
//
// Authority: relux-works/agent-session-manager-spec@v0.7.0, Sections
// 7.8, 10.2, and 13.14.1 (internal/specdoc/SPEC.v0.7.0.md; the
// pinned sections are byte-identical to v0.6.0).
//
// Every Decode entry validates a closed shape: exact members, bounded
// scalars, closed vocabularies, sorted-unique order, and self-digest
// agreement. Every Build entry constructs the same bytes
// deterministically: identical inputs produce byte-identical outputs,
// and every Build output decodes clean. Refusals name the offending
// member and never admit a partial object.
//
// Reuse, not fork: Environment Tuples decode through
// sessadapter.DecodeTuple, Blob Descriptors verify through
// canonicaljson.CalculateObjectIdentity plus a local claim comparison
// (never the attesting entry, per the provhost no-attestation bound),
// extension keys admit through
// environ.CheckExtensions, and blob bytes install through
// localstore.ObjectStore.PutBlob. The canonicaljson closed-shape
// registry keeps its rejectUnsupportedImmutableObjectShape rows for
// these schemas (that table's census pins every row, so replacing
// them would fork ownership); this package is the validating owner
// and says so here and in TRACEABILITY.md.
//
// Stated bounds (sibling scope, not waived): per-kind Canonical Event
// payload fact registries beyond the Section 13.14.1 envelope rules
// enforced here (which facts a kind may carry; the Section 1.6 value
// model on every payload member is enforced here);
// maximal_safe projection gating past the raw_complete=false
// derivation; target-branch (G2) admission beyond
// RefuseUnstableForTarget; capture-plan/capture/normalize adapter
// operation wiring (Section 7.8 bodies stay owned by sessadapter).
// This package performs no durable write itself: blob installation
// delegates to the landed localstore no-replace discipline, and the
// crash/idempotency evidence for that path is cited in
// TRACEABILITY.md.
