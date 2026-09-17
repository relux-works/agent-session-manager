package sessprofile

import (
	"bytes"
	"encoding/json"
	"sort"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file authors profile.changed Session Events (Section 5.2,
// urn:ax:schema:session-event 1.0.0) from host-side inputs: the
// session and author, the acting lease with its sequence and
// predecessors, the from/to ends, the operator confirmation, and
// the creation instant. SetProfile derives those inputs from the
// repository; MintChangeEvent validates and frames them.
//
// Minting is a pure function of its params: no clock, no
// randomness, no filesystem, no durable state. Identical params
// marshal through encoding/json's bytewise-sorted map keys with
// predecessors sorted below and the single textual event_id
// substitution, so identical inputs emit byte-identical events
// across calls and processes, which is the idempotency the
// set-profile retry promises. A time-varying created_at is the
// caller's input, not this function's decision: two mints at
// different instants name different instants and MUST differ,
// exactly as their params do.
//
// Every refusal names the violated rule. The event bytes a mint
// emits pass the canonical closed shape by construction, and the
// verdict is conjoined with the owner's production identity entry
// the same way CreateIdentity conjoins it: a mint this dialect
// admits but the owner refuses is refused at the backstop rather
// than emitted through a drifted copy. The digest return is the
// true omit-self identity the final event claims, and the sessrepo
// append path attests it; this package never calls
// VerifyObjectIdentity, so the no-attestation-outside-the-leaf
// bound holds.

// ChangeParams carries the host-side facts one profile.changed
// event is minted from. SessionID becomes both subject_id and
// session_id, which Section 5.2 requires equal. Predecessors sort
// in the emitted bytes because Section 5.2 declares the array
// sorted. CreatedAt is the caller's instant, not a clock read.
type ChangeParams struct {
	SessionID       string
	CreatedByHostID string
	LeaseEpoch      uint64
	LeaseID         string
	Sequence        uint64
	Predecessors    []string
	From            string
	To              string
	Confirmed       bool
	CreatedAt       string
}

// placeholderEventID stands in for event_id while the omit-self
// digest is computed. The owner requires the member present, so the
// staged bytes carry this well-formed digest and the final bytes
// carry the computed one; the substitution below matches the full
// `"event_id":"<placeholder>"` frame, never the bare digest, so a
// param value that repeats the digest text cannot misdirect it:
// values marshal with their quotes escaped while the frame's
// quotes are bare.
const placeholderEventID = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

// MintChangeEvent mints one canonical profile.changed event 1.0.0
// from host-side params. It returns the event bytes on success and
// a refusal naming the violated rule otherwise: unknown session or
// author identity, a lease outside the envelope grammar, a
// sequence below 1, no predecessor, a profile end outside the
// vocabulary, from equal to to, or a change to yolo without
// operator confirmation.
func MintChangeEvent(params ChangeParams) ([]byte, error) {
	if _, err := scalar.ParseUUIDv7(params.SessionID); err != nil {
		return nil, refuse(ErrInvalidEvent, "change session_id is not a UUIDv7: %v", err)
	}
	if _, err := scalar.ParseUUIDv7(params.CreatedByHostID); err != nil {
		return nil, refuse(ErrInvalidEvent, "change created_by_host_id is not a UUIDv7: %v", err)
	}
	if params.LeaseEpoch < 1 || params.LeaseEpoch > uint53Max {
		return nil, refuse(ErrInvalidEvent, "change lease_epoch %d is outside 1..2^53-1", params.LeaseEpoch)
	}
	if _, err := scalar.ParseUUIDv4(params.LeaseID); err != nil {
		return nil, refuse(ErrInvalidEvent, "change lease_id is not a UUIDv4: %v", err)
	}
	if params.Sequence < 1 || params.Sequence > uint53Max {
		return nil, refuse(ErrInvalidEvent, "change lease_sequence %d is outside 1..2^53-1", params.Sequence)
	}
	if len(params.Predecessors) == 0 {
		return nil, refuse(ErrInvalidEvent, "change names no predecessor")
	}
	predecessors := make([]string, 0, len(params.Predecessors))
	for _, predecessor := range params.Predecessors {
		if _, err := scalar.ParseDigest(predecessor); err != nil {
			return nil, refuse(ErrInvalidEvent, "change predecessor %q is not a digest: %v", predecessor, err)
		}
		predecessors = append(predecessors, predecessor)
	}
	sort.Strings(predecessors)
	if params.From != ProfileStandard && params.From != ProfileYOLO {
		return nil, refuse(ErrInvalidProfile, "change from %q is not standard|yolo", params.From)
	}
	if params.To != ProfileStandard && params.To != ProfileYOLO {
		return nil, refuse(ErrInvalidProfile, "change to %q is not standard|yolo", params.To)
	}
	if params.From == params.To {
		return nil, refuse(ErrProfileUnchanged, "change from %q to %q changes nothing", params.From, params.To)
	}
	if params.To == ProfileYOLO && !params.Confirmed {
		return nil, refuse(ErrUnconfirmedYOLO, "change to yolo names no operator confirmation")
	}
	if _, err := scalar.ParseTimestamp(params.CreatedAt); err != nil {
		return nil, refuse(ErrInvalidEvent, "change created_at is not a timestamp: %v", err)
	}
	object := map[string]any{
		"schema":             sessionEventSchema,
		"schema_version":     "1.0.0",
		"event_id":           placeholderEventID,
		"subject_id":         params.SessionID,
		"session_id":         params.SessionID,
		"event_type":         "profile.changed",
		"created_by_host_id": params.CreatedByHostID,
		"lease_epoch":        params.LeaseEpoch,
		"lease_id":           params.LeaseID,
		"lease_sequence":     params.Sequence,
		"predecessors":       predecessors,
		"created_at":         params.CreatedAt,
		"payload": map[string]any{
			"from":      params.From,
			"to":        params.To,
			"confirmed": params.Confirmed,
		},
		"extensions": map[string]any{},
	}
	staged, err := json.Marshal(object)
	if err != nil {
		return nil, refuse(ErrInvalidEvent, "change params do not marshal: %v", err)
	}
	// The backstop conjoins the owner's production identity entry:
	// any owner rule this dialect never reads is refused here
	// rather than emitted. The digest return is the true omit-self
	// identity the final event claims.
	digest, field, err := canonicaljson.CalculateObjectIdentity(staged)
	if err != nil {
		return nil, refuse(ErrInvalidEvent, "change params are not a valid session event: %v", err)
	}
	if field != canonicaljson.SelfEventID {
		return nil, refuse(ErrInvalidEvent, "change identity field %q, want event_id", string(field))
	}
	framed := []byte(`"event_id":"` + placeholderEventID + `"`)
	claimed := []byte(`"event_id":"` + digest.String() + `"`)
	return bytes.Replace(staged, framed, claimed, 1), nil
}
