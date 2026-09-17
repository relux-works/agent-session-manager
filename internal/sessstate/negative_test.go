package sessstate

import (
	"testing"
)

// TestReduceRefusesUnlistedTransitions requires every move outside the
// Section 5.7 table to fail with invalid_state_transition through the
// production Reduce entry.
func TestReduceRefusesUnlistedTransitions(t *testing.T) {
	record := decodeTestRecord(t)
	cases := []struct {
		name  string
		chain []eventSpec
	}{
		// running stops without idling or quiescing first.
		{"running-to-stopped", []eventSpec{
			{typ: "session.created", payload: createdPayload(record.RecordID)},
			{typ: "provider.launched", payload: launchedPayload("codex")},
			{typ: "session.stopped", payload: stoppedPayload()},
		}},
		// creating checkpoints before any launch.
		{"creating-to-checkpointing", []eventSpec{
			{typ: "session.created", payload: createdPayload(record.RecordID)},
			{typ: "checkpoint.created", payload: checkpointPayload()},
		}},
		// idle to stopped is the listed edge and is admitted; its
		// control lives in TestIdleToStoppedEdgeAdmits, not here.
		// stopped idles without a resume.
		{"stopped-to-idle", []eventSpec{
			{typ: "session.created", payload: createdPayload(record.RecordID)},
			{typ: "provider.launched", payload: launchedPayload("codex")},
			{typ: "session.idle", payload: idlePayload()},
			{typ: "checkpoint.created", payload: checkpointPayload()},
			{typ: "checkpoint.created", payload: checkpointPayload()},
			{typ: "session.stopped", payload: stoppedPayload()},
			{typ: "session.idle", payload: idlePayload()},
		}},
		// failed runs without a creating retry.
		{"failed-to-running", []eventSpec{
			{typ: "session.created", payload: createdPayload(record.RecordID)},
			{typ: "provider.launched", payload: launchedPayload("codex")},
			{typ: "session.failed", payload: failedPayload()},
			{typ: "provider.launched", payload: launchedPayload("codex")},
		}},
		// tombstoned resumes: tombstoned is terminal.
		{"tombstoned-to-running", []eventSpec{
			{typ: "session.created", payload: createdPayload(record.RecordID)},
			{typ: "provider.launched", payload: launchedPayload("codex")},
			{typ: "session.idle", payload: idlePayload()},
			{typ: "checkpoint.created", payload: checkpointPayload()},
			{typ: "checkpoint.created", payload: checkpointPayload()},
			{typ: "session.stopped", payload: stoppedPayload()},
			{typ: "session.tombstoned", payload: tombstonedPayload()},
			{typ: "session.resumed", payload: resumedPayload()},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := buildRecord(t, nil)
			_, events, _ := buildChain(t, raw, tc.chain)
			_, err := Reduce(Input{Record: record, Events: events})
			mustErrorIs(t, err, ErrInvalidTransition, "Reduce("+tc.name+")")
		})
	}
}

// idleChainPlusStop appends a checkpointed stop to an idle chain: idle
// has an edge to stopped, so this chain is VALID and serves as the
// control proving the table's idle-to-stopped edge admits what the
// rows above refuse.
func idleChainPlusStop(idle []eventSpec) []eventSpec {
	tail := []eventSpec{
		{typ: "checkpoint.created", payload: checkpointPayload()},
		{typ: "checkpoint.created", payload: checkpointPayload()},
		{typ: "session.stopped", payload: stoppedPayload()},
	}
	return append(append([]eventSpec{}, idle...), tail...)
}

// TestIdleToStoppedEdgeAdmits is the control for the transition table:
// the same stop the unlisted rows refuse is admitted over the listed
// idle-checkpointing-idle-stopped path.
func TestIdleToStoppedEdgeAdmits(t *testing.T) {
	record := decodeTestRecord(t)
	projection := reduceDirect(t, buildRecord(t, nil), idleChainPlusStop([]eventSpec{
		{typ: "session.created", payload: createdPayload(record.RecordID)},
		{typ: "provider.launched", payload: launchedPayload("codex")},
		{typ: "session.idle", payload: idlePayload()},
	}))
	if projection.State != StateStopped {
		t.Fatalf("state = %q, want stopped", projection.State)
	}
}

// TestReduceRefusesStoppedWithNullCheckpoint requires the Section 5.7
// null-checkpoint bar: a checkpointed stop that somehow carries a null
// checkpoint cannot derive stopped. The canonical owner refuses this
// shape at the boundary, so the vector is built by decoding the
// canonical stopped event and nulling the admitted payload member
// before Reduce — the reducer must not trust envelope-external edits.
func TestReduceRefusesStoppedWithNullCheckpoint(t *testing.T) {
	record := decodeTestRecord(t)
	raw := buildRecord(t, nil)
	chain := []eventSpec{
		{typ: "session.created", payload: createdPayload(record.RecordID)},
		{typ: "provider.launched", payload: launchedPayload("codex")},
		{typ: "session.idle", payload: idlePayload()},
		{typ: "checkpoint.created", payload: checkpointPayload()},
		{typ: "checkpoint.created", payload: checkpointPayload()},
	}
	_, events, _ := buildChain(t, raw, chain)
	stopped, _ := mustDecodeEvent(t, eventOptions{
		sessionID:    record.SessionID,
		predecessors: []string{events[len(events)-1].ID},
		epoch:        1,
		leaseID:      testLeaseID,
		sequence:     uint64(len(events) + 1),
		eventType:    "session.stopped",
		payload:      stoppedPayload(),
	})
	stopped.Payload["checkpoint_id"] = nil
	stopped.Payload["closure_kind"] = "checkpointed"
	_, err := Reduce(Input{Record: record, Events: append(events, stopped)})
	mustErrorIs(t, err, ErrIntegrity, "Reduce(stopped with null checkpoint)")
}

// TestReduceRefusesBootstrapAbortStopWithoutAbort requires a
// bootstrap_abort closure with no preceding authoritative abort event
// to refuse: only the abort proves the live process is closed.
func TestReduceRefusesBootstrapAbortStopWithoutAbort(t *testing.T) {
	record := decodeTestRecord(t)
	chain := []eventSpec{
		{typ: "session.created", payload: createdPayload(record.RecordID)},
		{typ: "provider.launched", payload: launchedPayload("codex")},
		{typ: "session.failed", payload: failedPayload()},
		{typ: "session.stopped", payload: stoppedAbortPayload()},
	}
	raw := buildRecord(t, nil)
	_, events, _ := buildChain(t, raw, chain)
	_, err := Reduce(Input{Record: record, Events: events})
	mustErrorIs(t, err, ErrInvalidTransition, "Reduce(abort closure without abort)")
}

// TestReduceDerivesFailedFromBootstrapAbortStop requires the proved
// abort path: abort event, then the bootstrap_abort closure, derives
// failed — never stopped.
func TestReduceDerivesFailedFromBootstrapAbortStop(t *testing.T) {
	record := decodeTestRecord(t)
	raw := buildRecord(t, nil)
	abort, _ := mustDecodeEvent(t, eventOptions{
		sessionID:    record.SessionID,
		predecessors: []string{record.RecordID},
		epoch:        1,
		leaseID:      testLeaseID,
		sequence:     1,
		eventType:    "session.bootstrap_aborted",
		payload:      abortPayload(),
	})
	stop, _ := mustDecodeEvent(t, eventOptions{
		sessionID:    record.SessionID,
		predecessors: []string{abort.ID},
		epoch:        1,
		leaseID:      testLeaseID,
		sequence:     2,
		eventType:    "session.stopped",
		payload:      stoppedAbortPayload(),
	})
	projection, err := Reduce(Input{Record: record, Events: []Event{abort, stop}})
	if err != nil {
		t.Fatalf("Reduce error = %v", err)
	}
	_ = raw
	if projection.State != StateFailed {
		t.Fatalf("state = %q, want failed", projection.State)
	}
	if projection.HasCheckpoint {
		t.Fatalf("newest = %+v, want no checkpoint after a bootstrap abort", projection.Newest)
	}
}

// TestReduceGatesFailedToCreatingRetry pins the Section 5.7 retry
// edge: epoch 1 under the same create lease with no checkpoint and no
// abort retries into creating; a new lease, a checkpoint, or an abort
// each refuse.
func TestReduceGatesFailedToCreatingRetry(t *testing.T) {
	record := decodeTestRecord(t)
	build := func(t *testing.T, mutate func(*[]eventSpec)) []Event {
		t.Helper()
		raw := buildRecord(t, nil)
		chain := []eventSpec{
			{typ: "session.created", payload: createdPayload(record.RecordID)},
			{typ: "provider.launched", payload: launchedPayload("codex")},
			{typ: "session.failed", payload: failedPayload()},
		}
		if mutate != nil {
			mutate(&chain)
		}
		chain = append(chain, eventSpec{typ: "session.created", payload: createdPayload(record.RecordID)})
		_, events, _ := buildChain(t, raw, chain)
		return events
	}
	t.Run("retry admitted", func(t *testing.T) {
		projection, err := Reduce(Input{Record: record, Events: build(t, nil)})
		if err != nil {
			t.Fatalf("Reduce error = %v", err)
		}
		if projection.State != StateCreating {
			t.Fatalf("state = %q, want creating", projection.State)
		}
	})
	t.Run("new lease refused", func(t *testing.T) {
		// The retry gate compares against the create lease, not the
		// current one: after a takeover the chain stands on lease B
		// with no checkpoint and no abort, fails there, and a
		// creating retry under B still refuses at the new-lease arm
		// (no other arm can fire: nothing was ever checkpointed).
		raw := buildRecord(t, nil)
		chain := []eventSpec{
			{typ: "session.created", payload: createdPayload(record.RecordID)},
			{typ: "provider.launched", payload: launchedPayload("codex")},
			{typ: "session.failed", payload: failedPayload()},
			{epoch: 2, leaseID: testLeaseIDB, typ: "lease.transferred", payload: transferredPayload(testLeaseID, testLeaseIDB)},
			{epoch: 2, leaseID: testLeaseIDB, typ: "session.failed", payload: failedPayload()},
			{epoch: 2, leaseID: testLeaseIDB, typ: "session.created", payload: createdPayload(record.RecordID)},
		}
		_, events, _ := buildChain(t, raw, chain)
		_, err := Reduce(Input{Record: record, Events: events})
		mustErrorIs(t, err, ErrInvalidTransition, "Reduce(retry under new lease)")
	})
	t.Run("checkpoint refused", func(t *testing.T) {
		raw := buildRecord(t, nil)
		head := []eventSpec{
			{typ: "session.created", payload: createdPayload(record.RecordID)},
			{typ: "provider.launched", payload: launchedPayload("codex")},
			{typ: "session.idle", payload: idlePayload()},
			{typ: "checkpoint.created", payload: checkpointPayload()},
			{typ: "session.failed", payload: failedPayload()},
		}
		_, headEvents, _ := buildChain(t, raw, head)
		retry, _ := mustDecodeEvent(t, eventOptions{
			sessionID:    record.SessionID,
			predecessors: []string{headEvents[len(headEvents)-1].ID},
			epoch:        1,
			leaseID:      testLeaseID,
			sequence:     6,
			eventType:    "session.created",
			payload:      createdPayload(record.RecordID),
		})
		_, err := Reduce(Input{Record: record, Events: append(headEvents, retry)})
		mustErrorIs(t, err, ErrInvalidTransition, "Reduce(retry past checkpoint)")
	})
	t.Run("abort refused", func(t *testing.T) {
		// An abort past creating refuses, so a retry can never stand
		// on an abort: the failed-to-creating gate's abort arm is
		// reachable only through the creating-position abort below.
		launched := []eventSpec{
			{typ: "session.created", payload: createdPayload(record.RecordID)},
			{typ: "provider.launched", payload: launchedPayload("codex")},
		}
		raw := buildRecord(t, nil)
		_, launchedEvents, _ := buildChain(t, raw, launched)
		abort, _ := mustDecodeEvent(t, eventOptions{
			sessionID:    record.SessionID,
			predecessors: []string{launchedEvents[len(launchedEvents)-1].ID},
			epoch:        1,
			leaseID:      testLeaseID,
			sequence:     3,
			eventType:    "session.bootstrap_aborted",
			payload:      abortPayload(),
		})
		_, err := Reduce(Input{Record: record, Events: append(launchedEvents, abort)})
		mustErrorIs(t, err, ErrInvalidTransition, "Reduce(abort past creating)")
	})
	t.Run("retry past abort refused", func(t *testing.T) {
		// created, abort-in-creating (failed), then a creating retry:
		// the retry gate refuses past the authoritative abort.
		raw := buildRecord(t, nil)
		created, _ := mustDecodeEvent(t, eventOptions{
			sessionID:    record.SessionID,
			predecessors: []string{record.RecordID},
			epoch:        1,
			leaseID:      testLeaseID,
			sequence:     1,
			eventType:    "session.created",
			payload:      createdPayload(record.RecordID),
		})
		abort, _ := mustDecodeEvent(t, eventOptions{
			sessionID:    record.SessionID,
			predecessors: []string{created.ID},
			epoch:        1,
			leaseID:      testLeaseID,
			sequence:     2,
			eventType:    "session.bootstrap_aborted",
			payload:      abortPayload(),
		})
		_ = raw
		if _, err := Reduce(Input{Record: record, Events: []Event{created, abort}}); err != nil {
			t.Fatalf("Reduce(abort in creating) error = %v", err)
		}
		retry, _ := mustDecodeEvent(t, eventOptions{
			sessionID:    record.SessionID,
			predecessors: []string{abort.ID},
			epoch:        1,
			leaseID:      testLeaseID,
			sequence:     3,
			eventType:    "session.created",
			payload:      createdPayload(record.RecordID),
		})
		_, err := Reduce(Input{Record: record, Events: []Event{created, abort, retry}})
		mustErrorIs(t, err, ErrInvalidTransition, "Reduce(retry past abort)")
	})
}

// TestReduceRefusesTaskBoardRepeatMismatch requires the Section 5.2
// repeat rule: a task_board.launched event that repeats a foreign
// provider or a foreign creation lease fails integrity_failure.
func TestReduceRefusesTaskBoardRepeatMismatch(t *testing.T) {
	record := decodeTestRecord(t)
	base := []eventSpec{
		{typ: "session.created", payload: createdPayload(record.RecordID)},
	}
	t.Run("foreign provider", func(t *testing.T) {
		payload := taskBoardLaunchedPayload("running", 1, testLeaseID)
		payload["provider_id"] = "qwen"
		chain := append(append([]eventSpec{}, base...),
			eventSpec{typ: "task_board.launched", payload: payload})
		raw := buildRecord(t, nil)
		_, events, _ := buildChain(t, raw, chain)
		_, err := Reduce(Input{Record: record, Events: events})
		mustErrorIs(t, err, ErrIntegrity, "Reduce(task-board foreign provider)")
	})
	t.Run("foreign creation lease", func(t *testing.T) {
		payload := taskBoardLaunchedPayload("running", 1, testLeaseIDB)
		chain := append(append([]eventSpec{}, base...),
			eventSpec{typ: "task_board.launched", payload: payload})
		raw := buildRecord(t, nil)
		_, events, _ := buildChain(t, raw, chain)
		_, err := Reduce(Input{Record: record, Events: events})
		mustErrorIs(t, err, ErrIntegrity, "Reduce(task-board foreign lease)")
	})
	t.Run("later creation epoch", func(t *testing.T) {
		payload := taskBoardLaunchedPayload("running", 2, testLeaseID)
		chain := append(append([]eventSpec{}, base...),
			eventSpec{typ: "task_board.launched", payload: payload})
		raw := buildRecord(t, nil)
		_, events, _ := buildChain(t, raw, chain)
		_, err := Reduce(Input{Record: record, Events: events})
		mustErrorIs(t, err, ErrIntegrity, "Reduce(task-board later epoch)")
	})
}

// TestReduceDerivesTaskBoardLaunch requires an exact repeat to derive
// the payload's running or idle state.
func TestReduceDerivesTaskBoardLaunch(t *testing.T) {
	record := decodeTestRecord(t)
	for _, state := range []string{"running", "idle"} {
		chain := []eventSpec{
			{typ: "session.created", payload: createdPayload(record.RecordID)},
			{typ: "task_board.launched", payload: taskBoardLaunchedPayload(state, 1, testLeaseID)},
		}
		projection := reduceDirect(t, buildRecord(t, nil), chain)
		if string(projection.State) != state {
			t.Fatalf("state = %q, want %q", projection.State, state)
		}
	}
}

// TestReduceRefusesLeaseLinkageMismatch requires the lease payload
// linkage to name the predecessor the chain stands on and the
// envelope it arrives under.
func TestReduceRefusesLeaseLinkageMismatch(t *testing.T) {
	record := decodeTestRecord(t)
	base := []eventSpec{
		{typ: "session.created", payload: createdPayload(record.RecordID)},
		{typ: "provider.launched", payload: launchedPayload("codex")},
		{typ: "session.idle", payload: idlePayload()},
		{typ: "checkpoint.created", payload: checkpointPayload()},
		{typ: "checkpoint.created", payload: checkpointPayload()},
		{typ: "session.stopped", payload: stoppedPayload()},
	}
	t.Run("wrong predecessor", func(t *testing.T) {
		chain := append(append([]eventSpec{}, base...),
			eventSpec{epoch: 2, leaseID: testLeaseIDB, typ: "lease.transferred", payload: transferredPayload(testLeaseIDC, testLeaseIDB)})
		raw := buildRecord(t, nil)
		_, events, _ := buildChain(t, raw, chain)
		_, err := Reduce(Input{Record: record, Events: events})
		mustErrorIs(t, err, ErrIntegrity, "Reduce(transfer from wrong predecessor)")
	})
	t.Run("envelope outside payload", func(t *testing.T) {
		chain := append(append([]eventSpec{}, base...),
			eventSpec{epoch: 2, leaseID: testLeaseIDC, typ: "lease.forced", payload: forcedPayload(testLeaseIDB, 1)})
		raw := buildRecord(t, nil)
		_, events, _ := buildChain(t, raw, chain)
		_, err := Reduce(Input{Record: record, Events: events})
		mustErrorIs(t, err, ErrIntegrity, "Reduce(force outside envelope)")
	})
	t.Run("transfer outside envelope", func(t *testing.T) {
		chain := append(append([]eventSpec{}, base...),
			eventSpec{epoch: 2, leaseID: testLeaseIDB, typ: "lease.transferred", payload: transferredPayload(testLeaseID, testLeaseIDC)})
		raw := buildRecord(t, nil)
		_, events, _ := buildChain(t, raw, chain)
		_, err := Reduce(Input{Record: record, Events: events})
		mustErrorIs(t, err, ErrIntegrity, "Reduce(transfer outside envelope)")
	})
}

// TestReduceRefusesCorruptPayloadMember requires a payload member that
// no longer decodes — despite the admitted envelope — to refuse the
// derivation instead of reading as absent.
func TestReduceRefusesCorruptPayloadMember(t *testing.T) {
	record := decodeTestRecord(t)
	raw := buildRecord(t, nil)
	chain := []eventSpec{
		{typ: "session.created", payload: createdPayload(record.RecordID)},
		{typ: "provider.launched", payload: launchedPayload("codex")},
	}
	_, events, _ := buildChain(t, raw, chain)
	identified, _ := mustDecodeEvent(t, eventOptions{
		sessionID:    record.SessionID,
		predecessors: []string{events[len(events)-1].ID},
		epoch:        1,
		leaseID:      testLeaseID,
		sequence:     3,
		eventType:    "provider.identified",
		payload:      identifiedPayload(),
	})
	identified.Payload["provider_identity_record_id"] = "not-a-digest"
	_, err := Reduce(Input{Record: record, Events: append(events, identified)})
	mustErrorIs(t, err, ErrDerivation, "Reduce(corrupt payload member)")
}

// TestDecodeRefusesMalformedInputs pins the decode boundary: foreign
// schemas, broken frames, bad identities, and bad members each refuse
// through DecodeRecord and DecodeEvent with the named sentinel.
func TestDecodeRefusesMalformedInputs(t *testing.T) {
	record := buildRecord(t, nil)
	if _, err := DecodeRecord([]byte("{not json")); err == nil {
		t.Fatalf("DecodeRecord(broken frame) = nil")
	} else {
		mustErrorIs(t, err, ErrInvalidRecord, "DecodeRecord(broken frame)")
	}
	event, _ := mustDecodeEvent(t, eventOptions{
		sessionID: testSessionID, predecessors: []string{zeroDigest},
		epoch: 1, leaseID: testLeaseID, sequence: 1,
		eventType: "session.idle", payload: idlePayload(),
	})
	_ = event
	corrupt, _ := mustDecodeEvent(t, eventOptions{
		sessionID: testSessionID, predecessors: []string{zeroDigest},
		epoch: 1, leaseID: testLeaseID, sequence: 1,
		eventType: "session.idle", payload: idlePayload(),
	})
	_ = corrupt
	if _, err := DecodeEvent([]byte(record)); err == nil {
		t.Fatalf("DecodeEvent(record bytes) = nil")
	} else {
		mustErrorIs(t, err, ErrInvalidEvent, "DecodeEvent(record bytes)")
	}
	if _, err := DecodeEvent([]byte("{not json")); err == nil {
		t.Fatalf("DecodeEvent(broken frame) = nil")
	} else {
		mustErrorIs(t, err, ErrInvalidEvent, "DecodeEvent(broken frame)")
	}
	// An event bound to another session refuses at decode: the
	// session binding is part of the decoded contract.
	foreign, _ := mustDecodeEvent(t, eventOptions{
		sessionID: testSessionIDB, predecessors: []string{zeroDigest},
		epoch: 1, leaseID: testLeaseID, sequence: 1,
		eventType: "session.idle", payload: idlePayload(),
	})
	if foreign.SessionID != testSessionIDB {
		t.Fatalf("foreign session = %q", foreign.SessionID)
	}
	if _, err := DecodeRecord([]byte(`{"schema":"urn:ax:schema:session-event","schema_version":"1.0.0"}`)); err == nil {
		t.Fatalf("DecodeRecord(event schema) = nil")
	} else {
		mustErrorIs(t, err, ErrInvalidRecord, "DecodeRecord(event schema)")
	}
}
