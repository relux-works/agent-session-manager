package sessstate

import (
	"encoding/json"
	"fmt"

	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const (
	sessionRecordSchema = "urn:ax:schema:session-record"
	sessionEventSchema  = "urn:ax:schema:session-event"
)

// DecodeRecord projects the reducer's record members from recordJSON:
// strict frame (environ) then member grammars (scalar, environ). It
// deliberately does not attest canonical identity: attestation is
// owned by internal/sessrepo at the durable boundary (and pinned
// there by the provhost no-attestation-outside-the-leaf gate), so a
// second Verify here would fork the attestation owner. Production
// bytes always arrive attested through the repository entries; the
// digest below is the grammar-checked self member of those bytes,
// never a recomputation.
func DecodeRecord(recordJSON []byte) (Record, error) {
	members, fault := environ.DecodeStrictObject(recordJSON)
	if fault != nil {
		return Record{}, refuse(ErrInvalidRecord, "decode session record frame: %v", fault)
	}
	schema, err := stringMember(members, "schema")
	if err != nil || schema != sessionRecordSchema {
		return Record{}, refuse(ErrInvalidRecord, "session record schema %q, want %q", schema, sessionRecordSchema)
	}
	sessionID, ok := environ.CheckUUIDv7(members["session_id"])
	if !ok {
		return Record{}, refuse(ErrInvalidRecord, "session record carries no valid session_id member")
	}
	recordID, err := stringMember(members, "record_id")
	if err != nil {
		return Record{}, refuse(ErrInvalidRecord, "session record carries no record_id member")
	}
	if _, err := scalar.ParseDigest(recordID); err != nil {
		return Record{}, refuse(ErrInvalidRecord, "session record record_id: %v", err)
	}
	// Identity members below were admitted by the canonical closed
	// shape above through the same owners, so these reads cannot fail
	// on bytes Verify accepted. The plain errors keep the refusal
	// funnel for reachable arms only.
	name, err := stringMember(members, "name")
	if err != nil {
		return Record{}, fmt.Errorf("session record carries no name member: %v", err)
	}
	kind, err := stringMember(members, "kind")
	if err != nil {
		return Record{}, fmt.Errorf("session record carries no kind member: %v", err)
	}
	provider, err := stringMember(members, "provider_id")
	if err != nil {
		return Record{}, fmt.Errorf("session record carries no provider_id member: %v", err)
	}
	if _, err := scalar.ParseProviderID(provider); err != nil {
		return Record{}, fmt.Errorf("session record provider_id: %v", err)
	}
	return Record{
		SessionID:  sessionID.String(),
		RecordID:   recordID,
		Name:       name,
		Kind:       kind,
		ProviderID: provider,
	}, nil
}

// DecodeEvent projects the routing members plus the payload object the
// reducer reads, under the same no-attestation division as
// DecodeRecord: the closed payload shape was admitted by the canonical
// owner at the sessrepo boundary, and a member that no longer decodes
// refuses the derivation instead of reading as absent.
func DecodeEvent(eventJSON []byte) (Event, error) {
	members, fault := environ.DecodeStrictObject(eventJSON)
	if fault != nil {
		return Event{}, refuse(ErrInvalidEvent, "decode session event frame: %v", fault)
	}
	schema, err := stringMember(members, "schema")
	if err != nil || schema != sessionEventSchema {
		return Event{}, refuse(ErrInvalidEvent, "session event schema %q, want %q", schema, sessionEventSchema)
	}
	version, err := stringMember(members, "schema_version")
	if err != nil {
		return Event{}, refuse(ErrInvalidEvent, "session event carries no schema_version member")
	}
	sessionID, ok := environ.CheckUUIDv7(members["session_id"])
	if !ok {
		return Event{}, refuse(ErrInvalidEvent, "session event carries no valid session_id member")
	}
	eventType, err := stringMember(members, "event_type")
	if err != nil {
		return Event{}, refuse(ErrInvalidEvent, "session event carries no event_type member")
	}
	epoch, ok := environ.CheckUint53Bounds(members["lease_epoch"], 1, uint53Max)
	if !ok {
		return Event{}, refuse(ErrInvalidEvent, "session event carries no lease_epoch at or above 1")
	}
	leaseID, err := stringMember(members, "lease_id")
	if err != nil {
		return Event{}, refuse(ErrInvalidEvent, "session event carries no lease_id member")
	}
	if _, err := scalar.ParseUUIDv4(leaseID); err != nil {
		return Event{}, refuse(ErrInvalidEvent, "session event lease_id: %v", err)
	}
	sequence, ok := environ.CheckUint53Bounds(members["lease_sequence"], 1, uint53Max)
	if !ok {
		return Event{}, refuse(ErrInvalidEvent, "session event carries no lease_sequence at or above 1")
	}
	owner, ok := environ.CheckUUIDv7(members["created_by_host_id"])
	if !ok {
		return Event{}, refuse(ErrInvalidEvent, "session event carries no valid created_by_host_id member")
	}
	predecessors, err := digestMembers(members, "predecessors")
	if err != nil {
		return Event{}, refuse(ErrInvalidEvent, "session event predecessors: %v", err)
	}
	if len(predecessors) == 0 {
		return Event{}, refuse(ErrInvalidEvent, "session event names no predecessor")
	}
	eventID, err := stringMember(members, "event_id")
	if err != nil {
		return Event{}, refuse(ErrInvalidEvent, "session event carries no event_id member")
	}
	if _, err := scalar.ParseDigest(eventID); err != nil {
		return Event{}, refuse(ErrInvalidEvent, "session event event_id: %v", err)
	}
	// The payload object was admitted by the canonical per-type union
	// at the sessrepo boundary; a non-object here is unreachable on
	// attested bytes. Plain for the same reason as the record
	// identity members.
	payload, err := objectMember(members, "payload")
	if err != nil {
		return Event{}, fmt.Errorf("session event carries no payload object: %v", err)
	}
	return Event{
		ID:              eventID,
		Type:            eventType,
		SchemaVersion:   version,
		SessionID:       sessionID.String(),
		CreatedByHostID: owner.String(),
		LeaseEpoch:      epoch,
		LeaseID:         leaseID,
		Sequence:        sequence,
		Predecessors:    predecessors,
		Payload:         payload,
	}, nil
}

// uint53Max is the AX safe-integer ceiling from SPEC.md Section 1.6.
// It is passed to the environ bound checker, never re-enforced here.
const uint53Max = uint64(1<<53 - 1)

// stringMember reads one string member with the language decoder.
// Grammar rules for the value stay with the shape owner. Failures are
// plain errors; the caller attributes them through its own site, so
// the refusal census sees one site per boundary arm, not per helper.
func stringMember(members map[string]json.RawMessage, name string) (string, error) {
	raw, ok := members[name]
	if !ok {
		return "", fmt.Errorf("member %q is absent", name)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("member %q is not a string: %v", name, err)
	}
	return value, nil
}

// digestMembers reads one array of digest strings. Digest grammar stays
// with the scalar owner. Failures are plain errors for the caller to
// attribute.
func digestMembers(members map[string]json.RawMessage, name string) ([]string, error) {
	raw, ok := members[name]
	if !ok {
		return nil, fmt.Errorf("member %q is absent", name)
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("member %q is not a string array: %v", name, err)
	}
	for _, value := range values {
		if _, err := scalar.ParseDigest(value); err != nil {
			return nil, fmt.Errorf("member %q holds malformed digest %q: %v", name, value, err)
		}
	}
	return values, nil
}

// objectMember reads one object member into a decoded map. Failures
// are plain errors for the caller to attribute.
func objectMember(members map[string]json.RawMessage, name string) (map[string]any, error) {
	raw, ok := members[name]
	if !ok {
		return nil, fmt.Errorf("member %q is absent", name)
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, fmt.Errorf("member %q is not an object", name)
	}
	return object, nil
}

// payloadString reads one required string from an admitted payload.
func payloadString(payload map[string]any, name string) (string, error) {
	value, ok := payload[name]
	if !ok {
		return "", refuse(ErrDerivation, "payload member %q is absent", name)
	}
	text, ok := value.(string)
	if !ok {
		return "", refuse(ErrDerivation, "payload member %q is not a string", name)
	}
	return text, nil
}

// payloadBool reads one required boolean from an admitted payload.
func payloadBool(payload map[string]any, name string) (bool, error) {
	value, ok := payload[name]
	if !ok {
		return false, refuse(ErrDerivation, "payload member %q is absent", name)
	}
	flag, ok := value.(bool)
	if !ok {
		return false, refuse(ErrDerivation, "payload member %q is not a boolean", name)
	}
	return flag, nil
}

// payloadUint reads one required uint53-range number from an admitted
// payload. JSON numbers decode as float64; range and integrality stay
// with this check, which refuses what the grammar cannot carry.
func payloadUint(payload map[string]any, name string) (uint64, error) {
	value, ok := payload[name]
	if !ok {
		return 0, refuse(ErrDerivation, "payload member %q is absent", name)
	}
	number, ok := value.(float64)
	if !ok || number != float64(uint64(number)) || uint64(number) == 0 || uint64(number) > uint53Max {
		return 0, refuse(ErrDerivation, "payload member %q is not a uint53 at or above 1", name)
	}
	return uint64(number), nil
}

// payloadNullableDigest reads one required nullable digest from an
// admitted payload: null reports absent without failing, a string must
// parse through the scalar owner.
func payloadNullableDigest(payload map[string]any, name string) (string, bool, error) {
	value, ok := payload[name]
	if !ok {
		return "", false, refuse(ErrDerivation, "payload member %q is absent", name)
	}
	if value == nil {
		return "", false, nil
	}
	text, ok := value.(string)
	if !ok {
		return "", false, refuse(ErrDerivation, "payload member %q is neither a digest nor null", name)
	}
	if _, err := scalar.ParseDigest(text); err != nil {
		return "", false, refuse(ErrDerivation, "payload member %q holds malformed digest %q: %v", name, text, err)
	}
	return text, true, nil
}
