package sessstate

import (
	"encoding/json"
	"testing"
)

// mutateRecordBytes rebuilds the fixture record with one member
// rewritten at the raw-JSON level, before any canonical check. Arms
// before Verify fire on these vectors; arms after Verify stay silent
// by construction.
func mutateRecordBytes(t *testing.T, mutate func(object map[string]any)) []byte {
	t.Helper()
	var object map[string]any
	if err := json.Unmarshal([]byte(specRecordExample), &object); err != nil {
		t.Fatalf("unmarshal SPEC record example: %v", err)
	}
	mutate(object)
	out, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal mutated record: %v", err)
	}
	return out
}

// mutateEventBytes rebuilds one canonical-exact event with one member
// rewritten at the raw-JSON level.
func mutateEventBytes(t *testing.T, options eventOptions, mutate func(object map[string]any)) []byte {
	t.Helper()
	raw := buildEvent(t, options)
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatalf("unmarshal built event: %v", err)
	}
	mutate(object)
	out, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal mutated event: %v", err)
	}
	return out
}

func deleteMember(name string) func(map[string]any) {
	return func(object map[string]any) { delete(object, name) }
}

func setMember(name string, value any) func(map[string]any) {
	return func(object map[string]any) { object[name] = value }
}

// TestDecodeRecordRefusesMemberVectors drives every reachable
// DecodeRecord arm with a vector that clears the earlier layers and
// fails exactly there.
func TestDecodeRecordRefusesMemberVectors(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"frame", nil},
		{"schema", setMember("schema", "urn:ax:schema:session-event")},
		{"session_id", setMember("session_id", "not-a-uuid")},
		{"record_id_absent", deleteMember("record_id")},
		{"record_id_grammar", setMember("record_id", "not-a-digest")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var raw []byte
			if tc.mutate == nil {
				raw = []byte("{not json")
			} else {
				raw = mutateRecordBytes(t, tc.mutate)
			}
			_, err := DecodeRecord(raw)
			mustErrorIs(t, err, ErrInvalidRecord, "DecodeRecord("+tc.name+")")
		})
	}
}

// TestDecodeEventRefusesMemberVectors drives every reachable
// DecodeEvent arm the same way.
func TestDecodeEventRefusesMemberVectors(t *testing.T) {
	base := func() eventOptions {
		return eventOptions{
			sessionID: testSessionID, predecessors: []string{zeroDigest},
			epoch: 1, leaseID: testLeaseID, sequence: 1,
			eventType: "session.idle", payload: idlePayload(),
		}
	}
	cases := []struct {
		name   string
		mutate func(map[string]any)
		raw    []byte
	}{
		{"frame", nil, []byte("{not json")},
		{"schema", setMember("schema", "urn:ax:schema:session-record"), nil},
		{"schema_version", deleteMember("schema_version"), nil},
		{"session_id", setMember("session_id", "not-a-uuid"), nil},
		{"event_type", deleteMember("event_type"), nil},
		{"lease_epoch", setMember("lease_epoch", 0), nil},
		{"lease_id_absent", deleteMember("lease_id"), nil},
		{"lease_id_grammar", setMember("lease_id", "not-a-uuid"), nil},
		{"lease_sequence", setMember("lease_sequence", 0), nil},
		{"created_by_host_id", setMember("created_by_host_id", "not-a-uuid"), nil},
		{"predecessors_absent", deleteMember("predecessors"), nil},
		{"predecessors_empty", setMember("predecessors", []any{}), nil},
		{"predecessors_malformed", setMember("predecessors", []any{"not-a-digest"}), nil},
		{"event_id_absent", deleteMember("event_id"), nil},
		{"event_id_grammar", setMember("event_id", "not-a-digest"), nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := tc.raw
			if raw == nil {
				raw = mutateEventBytes(t, base(), tc.mutate)
			}
			_, err := DecodeEvent(raw)
			mustErrorIs(t, err, ErrInvalidEvent, "DecodeEvent("+tc.name+")")
		})
	}
}

// handEvent decodes one event with explicit envelope members for
// continuity vectors the chained builders cannot produce. The version
// selects the canonical payload shape at build time; derivation reads
// whatever version the decoded event carries.
func handEvent(t *testing.T, record Record, predecessors []string, epoch uint64, leaseID string, sequence uint64, eventType string, payload map[string]any, version string) Event {
	t.Helper()
	event, _ := mustDecodeEvent(t, eventOptions{
		sessionID: record.SessionID, predecessors: predecessors,
		epoch: epoch, leaseID: leaseID, sequence: sequence,
		eventType: eventType, payload: payload, schemaVersion: version,
	})
	return event
}

// TestReduceRefusesContinuityVectors drives every chain-continuity arm
// with hand-built envelopes.
func TestReduceRefusesContinuityVectors(t *testing.T) {
	record := decodeTestRecord(t)
	raw := buildRecord(t, nil)
	chain := []eventSpec{
		{typ: "session.created", payload: createdPayload(record.RecordID)},
		{typ: "provider.launched", payload: launchedPayload("codex")},
	}
	_, events, _ := buildChain(t, raw, chain)
	tail := events[len(events)-1].ID

	t.Run("zero sequence", func(t *testing.T) {
		event := Event{ID: "hand", Type: "session.idle", SchemaVersion: "1.0.0",
			SessionID: record.SessionID, LeaseEpoch: 1, LeaseID: testLeaseID,
			Sequence: 0, Predecessors: []string{record.RecordID}, Payload: map[string]any{}}
		_, err := Reduce(Input{Record: record, Events: []Event{event}})
		mustErrorIs(t, err, ErrInvalidTransition, "Reduce(zero sequence)")
	})
	t.Run("no predecessors", func(t *testing.T) {
		event := Event{ID: "hand", Type: "session.idle", SchemaVersion: "1.0.0",
			SessionID: record.SessionID, LeaseEpoch: 1, LeaseID: testLeaseID,
			Sequence: 1, Payload: map[string]any{}}
		_, err := Reduce(Input{Record: record, Events: []Event{event}})
		mustErrorIs(t, err, ErrInvalidTransition, "Reduce(no predecessors)")
	})
	t.Run("first links extras", func(t *testing.T) {
		event := handEvent(t, record, []string{record.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(record.RecordID), "")
		event.Predecessors = []string{record.RecordID, zeroDigest}
		_, err := Reduce(Input{Record: record, Events: []Event{event}})
		mustErrorIs(t, err, ErrInvalidTransition, "Reduce(first links extras)")
	})
	t.Run("first opens past one", func(t *testing.T) {
		event := handEvent(t, record, []string{record.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(record.RecordID), "")
		event.Sequence = 2
		_, err := Reduce(Input{Record: record, Events: []Event{event}})
		mustErrorIs(t, err, ErrInvalidTransition, "Reduce(first opens past one)")
	})
	t.Run("gap", func(t *testing.T) {
		event := handEvent(t, record, []string{tail}, 1, testLeaseID, 4, "session.idle", idlePayload(), "")
		_, err := Reduce(Input{Record: record, Events: append(append([]Event{}, events...), event)})
		mustErrorIs(t, err, ErrInvalidTransition, "Reduce(gap)")
	})
	t.Run("successor restarts past one", func(t *testing.T) {
		event := handEvent(t, record, []string{tail}, 2, testLeaseIDB, 2, "session.idle", idlePayload(), "")
		_, err := Reduce(Input{Record: record, Events: append(append([]Event{}, events...), event)})
		mustErrorIs(t, err, ErrInvalidTransition, "Reduce(successor restarts past one)")
	})
	t.Run("omits prior head", func(t *testing.T) {
		event := handEvent(t, record, []string{record.RecordID}, 1, testLeaseID, 3, "session.idle", idlePayload(), "")
		_, err := Reduce(Input{Record: record, Events: append(append([]Event{}, events...), event)})
		mustErrorIs(t, err, ErrInvalidTransition, "Reduce(omits prior head)")
	})
	t.Run("unregistered version", func(t *testing.T) {
		event := handEvent(t, record, []string{record.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(record.RecordID), "")
		event.SchemaVersion = "9.9.9"
		_, err := Reduce(Input{Record: record, Events: []Event{event}})
		mustErrorIs(t, err, ErrInvalidEvent, "Reduce(unregistered version)")
	})
	t.Run("empty record identity", func(t *testing.T) {
		_, err := Reduce(Input{Record: Record{}, Events: events})
		mustErrorIs(t, err, ErrInvalidRecord, "Reduce(empty record)")
	})
	t.Run("stopped opens chain", func(t *testing.T) {
		event := handEvent(t, record, []string{record.RecordID}, 1, testLeaseID, 1, "session.stopped", stoppedPayload(), "")
		_, err := Reduce(Input{Record: record, Events: []Event{event}})
		mustErrorIs(t, err, ErrInvalidTransition, "Reduce(stopped opens chain)")
	})
	t.Run("checkpoint opens chain", func(t *testing.T) {
		event := handEvent(t, record, []string{record.RecordID}, 1, testLeaseID, 1, "checkpoint.created", checkpointPayload(), "")
		_, err := Reduce(Input{Record: record, Events: []Event{event}})
		mustErrorIs(t, err, ErrInvalidTransition, "Reduce(checkpoint opens chain)")
	})
}

// TestReduceRefusesDerivationVectors drives every payload-derivation
// arm by mutating an admitted payload after decode: the envelope is
// canonical-exact, the member no longer decodes.
func TestReduceRefusesDerivationVectors(t *testing.T) {
	record := decodeTestRecord(t)
	raw := buildRecord(t, nil)
	head := []eventSpec{
		{typ: "session.created", payload: createdPayload(record.RecordID)},
		{typ: "provider.launched", payload: launchedPayload("codex")},
	}
	_, headEvents, _ := buildChain(t, raw, head)
	tail := headEvents[len(headEvents)-1].ID
	next := uint64(len(headEvents) + 1)

	reduceWith := func(t *testing.T, eventType string, payload map[string]any, version string, mutate func(map[string]any)) error {
		t.Helper()
		event := handEvent(t, record, []string{tail}, 1, testLeaseID, next, eventType, payload, version)
		if mutate != nil {
			mutate(event.Payload)
		}
		_, err := Reduce(Input{Record: record, Events: append(append([]Event{}, headEvents...), event)})
		return err
	}
	cases := []struct {
		name      string
		eventType string
		payload   map[string]any
		version   string
		mutate    func(map[string]any)
	}{
		{"string absent", "provider.launched", launchedPayload("codex"), "", deletePayloadMember("provider_id")},
		{"string not string", "provider.launched", launchedPayload("codex"), "", setPayloadMember("provider_version", 7)},
		{"bool absent", "session.stopped", stoppedPayload(), "", deletePayloadMember("resumable")},
		{"bool not bool", "session.stopped", stoppedPayload(), "", setPayloadMember("resumable", "yes")},
		{"uint absent", "task_board.launched", taskBoardLaunchedPayload("running", 1, testLeaseID), "", deletePayloadMember("lease_epoch")},
		{"uint not uint", "task_board.launched", taskBoardLaunchedPayload("running", 1, testLeaseID), "", setPayloadMember("lease_epoch", 0)},
		{"nullable absent", "session.stopped", stoppedPayload(), "", deletePayloadMember("checkpoint_id")},
		{"nullable not string", "session.stopped", stoppedPayload(), "", setPayloadMember("checkpoint_id", 5)},
		{"nullable malformed", "session.stopped", stoppedPayload(), "", setPayloadMember("checkpoint_id", "zzz")},
		{"identity digest malformed", "provider.identified", identifiedPayload(), "", setPayloadMember("provider_identity_record_id", "zzz")},
		{"checkpoint digest malformed", "checkpoint.created", checkpointPayload(), "", setPayloadMember("checkpoint_id", "zzz")},
		{"v1 backend unknown", "terminal.created", terminalPayload(), "", setPayloadMember("backend", "tmuxx")},
		{"v4 binding malformed", "terminal.created", terminalV4Payload(), "4.0.0", setPayloadMember("terminal_binding_id", "zzz")},
		{"v4 backend refused", "terminal.created", terminalV4Payload(), "4.0.0", setPayloadMember("terminal_backend_id", "ax.unregistered-backend")},
		{"resume checkpoint malformed", "session.resumed", resumedPayload(), "", setPayloadMember("checkpoint_id", "zzz")},
		{"resume v1 backend unknown", "session.resumed", resumedPayload(), "", setPayloadMember("terminal_backend", "tmuxx")},
		{"resume v4 binding malformed", "session.resumed", resumedV4Payload(), "4.0.0", setPayloadMember("terminal_binding_id", "zzz")},
		{"resume v4 backend refused", "session.resumed", resumedV4Payload(), "4.0.0", setPayloadMember("terminal_backend_id", "ax.unregistered-backend")},
		{"abort closure with checkpoint", "session.stopped", stoppedAbortPayload(), "", setPayloadMember("checkpoint_id", checkpointOne)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// task_board vectors need the creating bootstrap only: the
			// repeat rule fires before any lifecycle move matters.
			if tc.eventType == "task_board.launched" {
				created := headEvents[:1]
				createdTail := created[0].ID
				event := handEvent(t, record, []string{createdTail}, 1, testLeaseID, 2, tc.eventType, tc.payload, tc.version)
				tc.mutate(event.Payload)
				_, err := Reduce(Input{Record: record, Events: append(append([]Event{}, created...), event)})
				mustErrorIs(t, err, ErrDerivation, "Reduce("+tc.name+")")
				return
			}
			err := reduceWith(t, tc.eventType, tc.payload, tc.version, tc.mutate)
			// The abort-closure vector refuses integrity (a proved
			// shape violated after admission); every other vector
			// refuses derivation (an unreadable member).
			if tc.name == "abort closure with checkpoint" {
				mustErrorIs(t, err, ErrIntegrity, "Reduce("+tc.name+")")
				return
			}
			mustErrorIs(t, err, ErrDerivation, "Reduce("+tc.name+")")
		})
	}
}

// TestReduceRefusesUnionVectors drives the union-input arms: an empty
// lease head and a malformed union lease ID.
func TestReduceRefusesUnionVectors(t *testing.T) {
	record := decodeTestRecord(t)
	specs := []eventSpec{
		{typ: "session.created", payload: createdPayload(record.RecordID)},
		{typ: "provider.launched", payload: launchedPayload("codex")},
	}
	t.Run("empty lease head", func(t *testing.T) {
		_, err := reduceSpecs(t, record, specs, Input{Union: []LeaseHead{{}}})
		mustErrorIs(t, err, ErrInvalidEvent, "Reduce(union empty lease)")
	})
	t.Run("malformed lease id", func(t *testing.T) {
		_, err := reduceSpecs(t, record, specs, Input{Union: []LeaseHead{{Epoch: 1, LeaseID: "nope"}}})
		mustErrorIs(t, err, ErrInvalidEvent, "Reduce(union malformed lease)")
	})
}

// TestReduceEmptyChainValidatesUnion requires the empty-chain path to
// validate the union through the same resolveWinner arms: a malformed
// union refuses exactly as on a non-empty chain, while a well-formed
// union moves only the reported winner and leaves state creating.
func TestReduceEmptyChainValidatesUnion(t *testing.T) {
	record := decodeTestRecord(t)
	t.Run("empty lease head", func(t *testing.T) {
		_, err := Reduce(Input{Record: record, Union: []LeaseHead{{Epoch: 4, LeaseID: "not-a-uuid"}, {}}})
		mustErrorIs(t, err, ErrInvalidEvent, "Reduce(empty chain malformed union)")
	})
	t.Run("malformed lease id", func(t *testing.T) {
		_, err := Reduce(Input{Record: record, Union: []LeaseHead{{Epoch: 1, LeaseID: "nope"}}})
		mustErrorIs(t, err, ErrInvalidEvent, "Reduce(empty chain malformed union)")
	})
	t.Run("well-formed union keeps creating", func(t *testing.T) {
		projection, err := Reduce(Input{Record: record, Union: []LeaseHead{{Epoch: 3, LeaseID: testLeaseIDC}}})
		if err != nil {
			t.Fatalf("Reduce error = %v", err)
		}
		if projection.State != StateCreating {
			t.Fatalf("state = %q, want creating: the union never rewrites authoritative state", projection.State)
		}
		if projection.Winner != (LeaseHead{Epoch: 3, LeaseID: testLeaseIDC}) {
			t.Fatalf("winner = %+v, want the reported union winner", projection.Winner)
		}
		if kinds := conflictKinds(projection); len(kinds) != 1 || kinds[0] != ConflictUnionSupersedesChain {
			t.Fatalf("conflicts = %v, want exactly [union_supersedes_chain]", kinds)
		}
	})
	t.Run("no union stays bare", func(t *testing.T) {
		projection, err := Reduce(Input{Record: record})
		if err != nil {
			t.Fatalf("Reduce error = %v", err)
		}
		if projection.State != StateCreating {
			t.Fatalf("state = %q, want creating", projection.State)
		}
		if !projection.Winner.IsZero() {
			t.Fatalf("winner = %+v, want zero: no chain and no union name no lease", projection.Winner)
		}
		if len(projection.Conflicts) != 0 {
			t.Fatalf("conflicts = %+v, want none", projection.Conflicts)
		}
	})
}

func deletePayloadMember(name string) func(map[string]any) {
	return func(payload map[string]any) { delete(payload, name) }
}

func setPayloadMember(name string, value any) func(map[string]any) {
	return func(payload map[string]any) { payload[name] = value }
}
