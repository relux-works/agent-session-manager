package sessstate

import (
	"reflect"
	"testing"
)

// lifecycleSpecs returns a running-session chain reused by the purity
// tests.
func lifecycleSpecs(record Record) []eventSpec {
	return []eventSpec{
		{typ: "session.created", payload: createdPayload(record.RecordID)},
		{typ: "terminal.created", payload: terminalPayload()},
		{typ: "provider.launched", payload: launchedPayload("codex")},
		{typ: "provider.identified", payload: identifiedPayload()},
		{typ: "session.idle", payload: idlePayload()},
	}
}

// TestReduceIsPure requires the same event sequence to yield the same
// projection on every fold: state, winner, checkpoint, provider,
// terminal, and warnings must all compare equal.
func TestReduceIsPure(t *testing.T) {
	record := decodeTestRecord(t)
	raw := buildRecord(t, nil)
	_, events, _ := buildChain(t, raw, lifecycleSpecs(record))
	first, err := Reduce(Input{Record: record, Events: events})
	if err != nil {
		t.Fatalf("Reduce(first) error = %v", err)
	}
	second, err := Reduce(Input{Record: record, Events: events})
	if err != nil {
		t.Fatalf("Reduce(second) error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("Reduce is not pure:\nfirst = %+v\nsecond = %+v", first, second)
	}
}

// TestReduceRefusesChainForbiddenReordering requires a reordering the
// chain forbids — two authoritative events swapped — to refuse with
// invalid_state_transition instead of silently deriving a different
// state.
func TestReduceRefusesChainForbiddenReordering(t *testing.T) {
	record := decodeTestRecord(t)
	raw := buildRecord(t, nil)
	_, events, _ := buildChain(t, raw, lifecycleSpecs(record))
	reordered := append([]Event(nil), events...)
	reordered[2], reordered[4] = reordered[4], reordered[2]
	_, err := Reduce(Input{Record: record, Events: reordered})
	mustErrorIs(t, err, ErrInvalidTransition, "Reduce(reordered)")
}

// TestReduceRefusesReversedChain requires the fully reversed chain to
// refuse: the first event no longer links the session record.
func TestReduceRefusesReversedChain(t *testing.T) {
	record := decodeTestRecord(t)
	raw := buildRecord(t, nil)
	_, events, _ := buildChain(t, raw, lifecycleSpecs(record))
	reversed := make([]Event, 0, len(events))
	for index := len(events) - 1; index >= 0; index-- {
		reversed = append(reversed, events[index])
	}
	_, err := Reduce(Input{Record: record, Events: reversed})
	mustErrorIs(t, err, ErrInvalidTransition, "Reduce(reversed)")
}

// TestReduceRefusesDuplicatedEvent requires a replayed authoritative
// event to refuse as a sequence repeat rather than double-apply.
func TestReduceRefusesDuplicatedEvent(t *testing.T) {
	record := decodeTestRecord(t)
	raw := buildRecord(t, nil)
	_, events, _ := buildChain(t, raw, lifecycleSpecs(record))
	duplicated := append(append([]Event(nil), events...), events[len(events)-1])
	_, err := Reduce(Input{Record: record, Events: duplicated})
	mustErrorIs(t, err, ErrInvalidTransition, "Reduce(duplicated)")
}

// TestReduceRefusesCrossSessionEvent requires an event bound to
// another session to refuse instead of merging foreign history.
func TestReduceRefusesCrossSessionEvent(t *testing.T) {
	record := decodeTestRecord(t)
	raw := buildRecord(t, nil)
	_, events, _ := buildChain(t, raw, lifecycleSpecs(record))
	foreign, _ := mustDecodeEvent(t, eventOptions{
		sessionID:    testSessionIDB,
		predecessors: []string{events[len(events)-1].ID},
		epoch:        1,
		leaseID:      testLeaseID,
		sequence:     uint64(len(events) + 1),
		eventType:    "session.idle",
		payload:      idlePayload(),
	})
	_, err := Reduce(Input{Record: record, Events: append(events, foreign)})
	mustErrorIs(t, err, ErrInvalidEvent, "Reduce(cross-session)")
}

// TestReduceRefusesStaleEnvelope requires a lower-epoch envelope in
// direct input to refuse: losing-lease history cannot be authoritative.
func TestReduceRefusesStaleEnvelope(t *testing.T) {
	record := decodeTestRecord(t)
	raw := buildRecord(t, nil)
	specs := append(lifecycleSpecs(record),
		eventSpec{epoch: 2, leaseID: testLeaseIDB, typ: "lease.transferred", payload: transferredPayload(testLeaseID, testLeaseIDB)},
	)
	_, events, _ := buildChain(t, raw, specs)
	stale, _ := mustDecodeEvent(t, eventOptions{
		sessionID:    record.SessionID,
		predecessors: []string{events[len(events)-1].ID},
		epoch:        1,
		leaseID:      testLeaseID,
		sequence:     1,
		eventType:    "session.idle",
		payload:      idlePayload(),
	})
	_, err := Reduce(Input{Record: record, Events: append(events, stale)})
	mustErrorIs(t, err, ErrStaleLease, "Reduce(stale envelope)")
}

// TestReduceRefusesDivergentEnvelope requires a same-epoch second
// lease in direct input to refuse: the chain cannot hold a tie.
func TestReduceRefusesDivergentEnvelope(t *testing.T) {
	record := decodeTestRecord(t)
	raw := buildRecord(t, nil)
	_, events, _ := buildChain(t, raw, lifecycleSpecs(record))
	divergent, _ := mustDecodeEvent(t, eventOptions{
		sessionID:    record.SessionID,
		predecessors: []string{events[len(events)-1].ID},
		epoch:        1,
		leaseID:      testLeaseIDB,
		sequence:     1,
		eventType:    "session.idle",
		payload:      idlePayload(),
	})
	_, err := Reduce(Input{Record: record, Events: append(events, divergent)})
	mustErrorIs(t, err, ErrDivergentLease, "Reduce(divergent envelope)")
}
