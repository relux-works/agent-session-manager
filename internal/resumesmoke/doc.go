// Package resumesmoke runs the host-side bounded native-resume smoke
// the pinned specification names as a target-write precondition
// (Section 7.8): a provider-specific discover, read, identify, and
// resume-plan sequence against one provider adapter, with any
// unsupported tuple claim refused.
//
// The smoke drives the landed provider host: every adapter call goes
// through provhost.Host.Call, and every response is validated by the
// landed Section 7 decoders — probe (7.4), identify-session (5.5),
// discovery binding over the Section 8.2 declared native roots, and
// the resume SpawnPlan (7.5). Quiescence (7.6) and the discovery
// proof enter as read-only precondition inputs the smoke validates
// and binds but never produces: proving quiescence and observing
// discovery belong to their owners, not to this framework.
//
// Each run records a closed Native Resume Smoke record
// (urn:ax:schema:native-resume-smoke 1.0.0): the claimed tuple, the
// Section 8.4 native-resume cell, the verdict, digests of the probe,
// discovery proof, identity record, and spawn plan, the store root,
// every check with its pass, fail, or skipped outcome, and
// timestamps. The record carries no secrets and no raw native
// references — only digests, the documented store root, and check
// facts. It is the detailed evidence a Section 13.14
// ResumeSmokeEvidence registry entry points at through its
// evidence_digest; the registry object itself stays owned by
// internal/sessadapter.
//
// Refusal is the point, not an edge. A tuple whose Section 8.4
// native-resume cell is conditional, unsupported, or unknown, or
// that Appendix B leaves unsettled, can never pass: unsupported and
// unknown tuples are refused before any adapter call, conditional
// tuples gather read-only probe, identify, and binding evidence and
// then gate the resume plan, and no smoke API promotes a gated or
// refused cell — promotion belongs to Section 19.3 evidence the
// smoke never produces. A passing smoke never advertises a
// capability: the Section 8.3 capability matrix stays the only
// capability authority, so this package carries no capability map
// and no doctor surface.
//
// Stated bounds: request-body member ranges (terminal geometry,
// lease shape, workspace absoluteness) are validated for the tuple
// platform at the params boundary, but deeper adapter-body semantics
// stay the adapter's to refuse under Section 7.5; the quiescence
// precondition is consumed as valid-and-safe only, and binding the
// proof's own provider and version to the tuple belongs to the
// quiescence owner. Record integrity detects tampering and naive
// promotion, not a forgery that recomputes both the digest and a
// consistent check list: smoke records are local evidence with no
// adversary, and the no-promotion property is that no production
// path emits pass for a non-available cell.
package resumesmoke
