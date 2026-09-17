package sessprofile

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
// owned by internal/sessrepo at the durable boundary, so a second
// Verify here would fork the attestation owner. Production bytes
// always arrive attested through the repository entries; the digest
// below is the grammar-checked self member of those bytes, never a
// recomputation.
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
	creation, err := stringMember(members, "execution_profile")
	if err != nil {
		return Record{}, refuse(ErrInvalidRecord, "session record carries no execution_profile member")
	}
	if creation != ProfileStandard && creation != ProfileYOLO {
		return Record{}, refuse(ErrInvalidRecord, "session record carries creation profile %q, want standard|yolo", creation)
	}
	return Record{
		SessionID: sessionID.String(),
		RecordID:  recordID,
		Creation:  creation,
	}, nil
}

// DecodeEvent projects the routing members plus the payload object
// the reducer reads, under the same no-attestation division as
// DecodeRecord: the closed payload shape was admitted by the
// canonical owner at the sessrepo boundary, and a member that no
// longer decodes refuses the derivation instead of reading as
// absent.
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
	author, ok := environ.CheckUUIDv7(members["created_by_host_id"])
	if !ok {
		return Event{}, refuse(ErrInvalidEvent, "session event carries no valid created_by_host_id member")
	}
	createdAt, err := stringMember(members, "created_at")
	if err != nil {
		return Event{}, refuse(ErrInvalidEvent, "session event carries no created_at member")
	}
	if _, err := scalar.ParseTimestamp(createdAt); err != nil {
		return Event{}, refuse(ErrInvalidEvent, "session event created_at: %v", err)
	}
	// The payload object was admitted by the canonical per-type union
	// at the sessrepo boundary; a non-object here is unreachable on
	// attested bytes. Plain for the same reason as the sessstate
	// identity members: the refusal funnel is for reachable arms.
	payload, err := objectMember(members, "payload")
	if err != nil {
		return Event{}, fmt.Errorf("session event carries no payload object: %v", err)
	}
	return Event{
		ID:              eventID,
		Type:            eventType,
		SchemaVersion:   version,
		SessionID:       sessionID.String(),
		CreatedByHostID: author.String(),
		CreatedAt:       createdAt,
		LeaseEpoch:      epoch,
		LeaseID:         leaseID,
		Sequence:        sequence,
		Predecessors:    predecessors,
		Payload:         payload,
	}, nil
}

// uint53Max is the AX safe-integer ceiling from the pinned Section
// 1.6. It is passed to the environ bound checker, never re-enforced
// here.
const uint53Max = uint64(1<<53 - 1)

// stringMember reads one string member with the language decoder.
// Grammar rules for the value stay with the shape owner. Failures
// are plain errors; the caller attributes them through its own
// site, so the refusal census sees one site per boundary arm, not
// per helper.
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

// digestMembers reads one array of digest strings. Digest grammar
// stays with the scalar owner. Failures are plain errors for the
// caller to attribute.
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
