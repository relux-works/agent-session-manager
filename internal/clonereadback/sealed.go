package clonereadback

import (
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file seals validated read-back manifests (SPEC v0.7.0
// Section 13.14.2 lines 10753-10763: the validation report
// aggregates TWO DISTINCT validated staged and live read-back
// manifests). A report entry cannot vouch for a sibling from a
// caller-minted struct that merely repeats an ID and a mode
// label: every other field could be zero without any validated
// read-back bytes behind it. Parse, don't validate: the report
// entries accept ONLY ValidatedReadBack, a value the read-back
// validator mints and no outside caller can forge — its fields
// are unexported, the only constructors are the two read-back
// entries, and the zero value is refused at both report entries.
// The sealed value carries the validated manifest (with the
// omit-self digest set and the ReadAuthority-derived mode) behind
// read-only accessors; the accessors return scalars and a struct
// copy, never a new seal.

// ValidatedReadBack is one sealed read-back manifest: proof that
// the read passed the read-back validator. The zero value seals
// nothing and every report entry refuses it.
type ValidatedReadBack struct {
	read   ReadBackManifest
	sealed bool
}

// sealReadBack mints one sealed read from a validated manifest.
// Only the read-back entries call it, after every gate passes.
func sealReadBack(read ReadBackManifest) ValidatedReadBack {
	return ValidatedReadBack{read: read, sealed: true}
}

// Manifest returns the validated manifest behind the seal: a
// struct copy for field reads. The copy cannot mint a seal —
// only the read-back entries seal — so handing it out changes no
// gate.
func (v ValidatedReadBack) Manifest() ReadBackManifest {
	return v.read
}

// ManifestID returns the sealed omit-self digest.
func (v ValidatedReadBack) ManifestID() scalar.Digest {
	return v.read.ManifestID
}

// Mode returns the sealed ReadAuthority-derived mode.
func (v ValidatedReadBack) Mode() string {
	return v.read.Mode
}
