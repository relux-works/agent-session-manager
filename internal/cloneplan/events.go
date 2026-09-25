package cloneplan

import (
	"encoding/json"
	"fmt"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file validates and constructs SynthesizedProjectionEvent:
// the closed synthesized-event row Section 13.14.2 states. Array
// position IS the insertion sequence the table row orders by, so no
// order gate runs over the event set; insertion anchors against
// canonical IDs need the Canonical Session as an input and are a
// stated bound.

var synthesizedEventMembers = map[string]bool{
	"canonical_event_id":       true,
	"insertion_after_event_id": true,
	"purpose":                  true,
	"extensions":               true,
}

var synthesizedEventRequired = []string{
	"canonical_event_id",
	"insertion_after_event_id",
	"purpose",
	"extensions",
}

// SynthesizedEvent is one validated synthesized projection event.
type SynthesizedEvent struct {
	CanonicalEventID    scalar.Digest
	InsertionAfterEvent *scalar.Digest
	Purpose             string
}

// SynthesizedEventInput is the caller-supplied synthesized event
// candidate for Build. A nil InsertionAfterEvent seals as null.
type SynthesizedEventInput struct {
	CanonicalEventID    string
	InsertionAfterEvent *string
	Purpose             string
	Extensions          map[string]any
}

// buildSynthesizedEvent validates one caller-supplied event and
// renders its closed object.
func buildSynthesizedEvent(input SynthesizedEventInput, index int) (SynthesizedEvent, map[string]any, error) {
	owner := fmt.Sprintf("synthesized projection event[%d]", index)
	eventID, err := scalar.ParseDigest(input.CanonicalEventID)
	if err != nil {
		return SynthesizedEvent{}, nil, invalid("%s canonical_event_id is not a digest: %v", owner, err)
	}
	var after *scalar.Digest
	var afterValue any
	if input.InsertionAfterEvent != nil {
		id, err := scalar.ParseDigest(*input.InsertionAfterEvent)
		if err != nil {
			return SynthesizedEvent{}, nil, invalid("%s insertion_after_event_id is not a digest: %v", owner, err)
		}
		after = &id
		afterValue = id.String()
	}
	if !validText(input.Purpose) {
		return SynthesizedEvent{}, nil, invalid("%s purpose is not valid UTF-8", owner)
	}
	if !ValidSynthesizedPurpose(input.Purpose) {
		return SynthesizedEvent{}, nil, invalid("%s purpose is outside migration_checkpoint|summary|delimiter", owner)
	}
	if _, err := clonebundle.EncodeExtensions(input.Extensions); err != nil {
		return SynthesizedEvent{}, nil, invalid("%s extensions invalid: %v", owner, err)
	}
	extensions := input.Extensions
	if extensions == nil {
		extensions = map[string]any{}
	}
	return SynthesizedEvent{
			CanonicalEventID:    eventID,
			InsertionAfterEvent: after,
			Purpose:             input.Purpose,
		}, map[string]any{
			"canonical_event_id":       eventID.String(),
			"insertion_after_event_id": afterValue,
			"purpose":                  input.Purpose,
			"extensions":               extensions,
		}, nil
}

// decodeSynthesizedEvent validates one closed
// SynthesizedProjectionEvent.
func decodeSynthesizedEvent(raw json.RawMessage, index int) (SynthesizedEvent, error) {
	owner := "synthesized projection event"
	members, fault := decodeStrictObject(raw)
	if fault != nil {
		return SynthesizedEvent{}, invalid("%s[%d] %s (%s)", owner, index, fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, synthesizedEventMembers); unknown {
		return SynthesizedEvent{}, invalid("%s[%d] carries unknown member %q", owner, index, name)
	}
	if name, missing := missingMember(members, synthesizedEventRequired); missing {
		return SynthesizedEvent{}, invalid("%s[%d] misses a required member %q", owner, index, name)
	}
	eventID, ok := checkDigest(members["canonical_event_id"])
	if !ok {
		return SynthesizedEvent{}, invalid("%s[%d] canonical_event_id is not a digest", owner, index)
	}
	var after *scalar.Digest
	if !isNull(members["insertion_after_event_id"]) {
		id, ok := checkDigest(members["insertion_after_event_id"])
		if !ok {
			return SynthesizedEvent{}, invalid("%s[%d] insertion_after_event_id is not a digest or null", owner, index)
		}
		after = &id
	}
	purpose, ok := rawString(members["purpose"])
	if !ok || !ValidSynthesizedPurpose(purpose) {
		return SynthesizedEvent{}, invalid("%s[%d] purpose is outside migration_checkpoint|summary|delimiter", owner, index)
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return SynthesizedEvent{}, invalid("%s[%d] extensions %s", owner, index, extensionsFault)
	}
	return SynthesizedEvent{
		CanonicalEventID:    eventID,
		InsertionAfterEvent: after,
		Purpose:             purpose,
	}, nil
}

// checkSynthesizedEventCount enforces the synthesized_events
// cardinality bound SynthesizedProjectionEvent[0..65536]. The gate
// is factored so the exact upper edge pins at the gate while entry
// reachability pins at both entries.
func checkSynthesizedEventCount(count int) error {
	if count < 0 || count > 65536 {
		return invalid("projection plan carries %d synthesized_events, want [0..65536]", count)
	}
	return nil
}
