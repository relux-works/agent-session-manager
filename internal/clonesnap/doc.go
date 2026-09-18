package clonesnap

// This package implements the Section 13.14.1 native capture operation
// over a real on-disk provider store: a contained capture walk that
// produces the Clone Raw Object Manifest and Clone Capture Manifest
// from actual bytes, classifies every Capture Item, builds the
// Capture Boundary from measured evidence, detects source mutation
// before any projection output, checkpoints the workspace binding
// and source head, and admits stable captures to a target branch.
//
// Authority: relux-works/agent-session-manager-spec@v0.7.0, Sections
// 7.8, 10.2, and 13.14.1 (internal/specdoc/SPEC.v0.7.0.md; the
// pinned sections are byte-identical to v0.6.0).
//
// Every artifact this package emits is produced through the landed
// clonebundle builders and validated by the landed decoders:
// BuildRawObjectManifest and BuildCaptureManifest seal the
// manifests, SanitizeNativeKey admits native keys,
// CheckDigestsEqual decides the source-race gate,
// RefuseUnstableForTarget and RefuseMaximalSafeUnlessComplete guard
// target admission, and InstallRawBlob installs payload bytes. No
// second manifest, boundary, blob, or identity model exists here.
//
// Reuse, not fork: store containment is the landed secprim Guard
// discipline (every payload open descends from a verified root
// handle, never by re-resolving a path string); durable writes go
// only through localstore.ObjectStore.PutBlob (no-replace install,
// fsync, size/digest verification); Blob Descriptors validate
// through canonicaljson.CalculateObjectIdentity plus local claim
// comparisons (never the attesting entry, per the provhost
// no-attestation bound); capture classes admit through
// clonebundle.ValidCaptureClass; the source digest and receipt bytes
// seal through scalar digests and canonicaljson.Canonicalize.
//
// Stated bounds (sibling scope, not waived): no ax CLI, no provider
// process, no network — the store is a fixture directory; Section
// 7.8 capture-plan/capture/normalize operation wiring stays owned by
// sessadapter (this package consumes plan keys and classes, it does
// not speak the adapter protocol); projection planning past target
// admission is the final leaf's (this package emits only the
// admission receipt that pins G2 admission and race-before-projection
// ordering); per-kind event payload registries are the normalization
// sibling's. internal/traceability bindings and the re-pin ride with
// the Story FINAL leaf; the clause map lives in TRACEABILITY.md.
