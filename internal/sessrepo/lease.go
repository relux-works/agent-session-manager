// Lease Record attestation for the session-persistence leaf.
//
// Lease Records (urn:ax:schema:lease) are session-persistence objects
// alongside Session Records and Session Events. This helper owns their
// canonical identity attestation through the canonicaljson owner, so
// query-layer admission (internal/sessquery) never attests the binding
// itself: it calls AttestLeaseRecord and then extracts the already
// validated members. The lease store's mint path confirms through the
// same funnel, keeping this file the leaf's only direct Verify caller
// for lease objects. The provhost identity bound allowlists this site
// with the three session record/event sites.
package sessrepo

import (
	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// AttestLeaseRecord validates one canonical Lease Record frame through
// the canonicaljson owner and returns its canonical digest with the
// self-identity field. Closed-shape and self-identity failures are
// returned for the caller to map to its refusal class.
func AttestLeaseRecord(raw []byte) (scalar.Digest, canonicaljson.SelfField, error) {
	return canonicaljson.VerifyObjectIdentity(raw)
}
