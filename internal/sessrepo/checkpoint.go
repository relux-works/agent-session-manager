// Checkpoint Record attestation for the session-persistence leaf.
//
// Checkpoint Records (urn:ax:schema:checkpoint) are session-persistence
// objects alongside Session Records, Session Events, and Lease Records.
// This helper owns their canonical identity attestation through the
// canonicaljson owner, so query-layer admission (internal/sessquery)
// never attests the binding itself: it calls AttestCheckpointRecord
// and then extracts the already validated members. The provhost
// identity bound allowlists this site with the session record/event
// sites and the lease site.
package sessrepo

import (
	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// AttestCheckpointRecord validates one canonical Checkpoint Record
// frame through the canonicaljson owner and returns its canonical
// digest with the self-identity field. Closed-shape and self-identity
// failures are returned for the caller to map to its refusal class.
func AttestCheckpointRecord(raw []byte) (scalar.Digest, canonicaljson.SelfField, error) {
	return canonicaljson.VerifyObjectIdentity(raw)
}
