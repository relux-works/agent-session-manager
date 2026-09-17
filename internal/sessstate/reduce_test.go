package sessstate

import (
	"testing"
)

// eventSpec is one chain step: envelope position plus type and payload.
type eventSpec struct {
	epoch   uint64
	leaseID string
	typ     string
	payload map[string]any
	version string
}

// buildChain folds specs into canonical-exact decoded events,
// assigning per-lease sequences (a successor lease restarts at 1) and
// chaining predecessors. It returns the raw bytes alongside, so the
// same chain drives both the pure Reduce entry and the
// repository-backed Project entry.
func buildChain(t *testing.T, record []byte, specs []eventSpec) (Record, []Event, [][]byte) {
	t.Helper()
	decoded, err := DecodeRecord(record)
	if err != nil {
		t.Fatalf("DecodeRecord error = %v", err)
	}
	var events []Event
	var raws [][]byte
	predecessors := []string{decoded.RecordID}
	var lastLease string
	var sequence uint64
	for _, spec := range specs {
		epoch := spec.epoch
		if epoch == 0 {
			epoch = 1
		}
		leaseID := spec.leaseIDOr(testLeaseID)
		if leaseID != lastLease {
			sequence = 1
			lastLease = leaseID
		} else {
			sequence++
		}
		event, raw := mustDecodeEvent(t, eventOptions{
			sessionID:     decoded.SessionID,
			predecessors:  predecessors,
			epoch:         epoch,
			leaseID:       leaseID,
			sequence:      sequence,
			eventType:     spec.typ,
			payload:       spec.payload,
			schemaVersion: spec.version,
		})
		events = append(events, event)
		raws = append(raws, raw)
		predecessors = []string{event.ID}
	}
	return decoded, events, raws
}

// reduceDirect folds specs through Decode into Reduce without a
// repository, for the pure-core table.
func reduceDirect(t *testing.T, record []byte, specs []eventSpec) Projection {
	t.Helper()
	decoded, events, _ := buildChain(t, record, specs)
	projection, err := Reduce(Input{Record: decoded, Events: events})
	if err != nil {
		t.Fatalf("Reduce error = %v", err)
	}
	return projection
}

// TestReduceDerivesEverySessionState drives each of the eleven
// Section 5.7 values through the production Reduce entry and, for the
// same chain, through the repository-backed Project entry, requiring
// both to agree. A value with no producing path fails here.
func TestReduceDerivesEverySessionState(t *testing.T) {
	record := buildRecord(t, nil)
	decoded, err := DecodeRecord(record)
	if err != nil {
		t.Fatalf("DecodeRecord error = %v", err)
	}
	created := eventSpec{typ: "session.created", payload: createdPayload(decoded.RecordID)}
	launched := eventSpec{typ: "provider.launched", payload: launchedPayload("codex")}
	idle := eventSpec{typ: "session.idle", payload: idlePayload()}
	checkpoint := eventSpec{typ: "checkpoint.created", payload: checkpointPayload()}
	transferred := eventSpec{epoch: 2, leaseID: testLeaseIDB, typ: "lease.transferred", payload: transferredPayload(testLeaseID, testLeaseIDB)}

	cases := []struct {
		name  string
		chain []eventSpec
		want  State
	}{
		{"creating", []eventSpec{created}, StateCreating},
		{"running", []eventSpec{created, launched}, StateRunning},
		{"idle", []eventSpec{created, launched, idle}, StateIdle},
		{"quiescing", []eventSpec{created, launched, idle,
			{typ: "session.quiescing", payload: quiescingPayload()}}, StateQuiescing},
		{"checkpointing", []eventSpec{created, launched, idle, checkpoint}, StateCheckpointing},
		{"stopped", []eventSpec{created, launched, idle, checkpoint, checkpoint,
			{typ: "session.stopped", payload: stoppedPayload()}}, StateStopped},
		{"materializing", []eventSpec{created, launched, idle, checkpoint, checkpoint,
			{typ: "session.stopped", payload: stoppedPayload()}, transferred}, StateMaterializing},
		{"parked", []eventSpec{created, launched, idle, checkpoint, checkpoint,
			{typ: "session.stopped", payload: stoppedPayload()}, transferred,
			{epoch: 2, leaseID: testLeaseIDB, typ: "session.parked", payload: parkedPayload(testLeaseIDB)}}, StateParked},
		{"failed", []eventSpec{created, launched,
			{typ: "session.failed", payload: failedPayload()}}, StateFailed},
		{"stale", []eventSpec{created, launched, idle,
			{epoch: 2, leaseID: testLeaseIDB, typ: "lease.forced", payload: forcedPayload(testLeaseIDB, 1)}}, StateStale},
		{"tombstoned", []eventSpec{created, launched, idle, checkpoint, checkpoint,
			{typ: "session.stopped", payload: stoppedPayload()},
			{typ: "session.tombstoned", payload: tombstonedPayload()}}, StateTombstoned},
	}
	for _, tc := range cases {
		t.Run(string(tc.want), func(t *testing.T) {
			projection := reduceDirect(t, record, tc.chain)
			if projection.State != tc.want {
				t.Fatalf("Reduce state = %q, want %q", projection.State, tc.want)
			}
			// The same bytes through the repository-backed entry must
			// agree: append them and Project.
			_, _, raws := buildChain(t, record, tc.chain)
			repository, err := openRepo(t)
			if err != nil {
				t.Fatalf("open repo error = %v", err)
			}
			reference := createSession(t, repository, record)
			for _, raw := range raws {
				if _, err := repository.AppendEvent(reference.SessionID, raw); err != nil {
					t.Fatalf("AppendEvent error = %v", err)
				}
			}
			projector := &Projector{Repo: repository}
			projected, err := projector.Project(reference.SessionID)
			if err != nil {
				t.Fatalf("Project error = %v", err)
			}
			if projected.State != tc.want {
				t.Fatalf("Project state = %q, want %q", projected.State, tc.want)
			}
		})
	}
}

func (spec eventSpec) leaseIDOr(fallback string) string {
	if spec.leaseID == "" {
		return fallback
	}
	return spec.leaseID
}

// TestProjectDerivesFullLifecycle folds one realistic bootstrap-to-run
// chain through the production repository entries and requires the
// full derived surface: state, winner, newest checkpoint, provider
// identity, and terminal binding.
func TestProjectDerivesFullLifecycle(t *testing.T) {
	record := buildRecord(t, nil)
	builder := openChain(t, record)
	decoded, err := DecodeRecord(record)
	if err != nil {
		t.Fatalf("DecodeRecord error = %v", err)
	}
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 1, eventType: "session.created", payload: createdPayload(decoded.RecordID)})
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 2, eventType: "terminal.created", payload: terminalPayload()})
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 3, eventType: "provider.launched", payload: launchedPayload("codex")})
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 4, eventType: "provider.identified", payload: identifiedPayload()})
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 5, eventType: "session.idle", payload: idlePayload()})
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 6, eventType: "session.quiescing", payload: quiescingPayload()})
	firstCheckpoint := builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 7, eventType: "checkpoint.created", payload: checkpointPayload()})
	_ = firstCheckpoint
	secondPayload := map[string]any{"checkpoint_id": checkpointTwo, "kind": "pre_stop"}
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 8, eventType: "checkpoint.created", payload: secondPayload})
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 9, eventType: "session.stopped", payload: stoppedPayloadWith(checkpointTwo)})
	resumed := builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 10, eventType: "session.resumed", payload: resumedPayloadWith(checkpointTwo)})

	projector := &Projector{Repo: builder.repo, LocalHostID: testHostID}
	projection, err := projector.Project(builder.sessionID)
	if err != nil {
		t.Fatalf("Project error = %v", err)
	}
	if projection.State != StateRunning {
		t.Fatalf("state = %q, want running", projection.State)
	}
	if projection.Winner != (LeaseHead{Epoch: 1, LeaseID: testLeaseID}) {
		t.Fatalf("winner = %+v, want epoch 1 lease A", projection.Winner)
	}
	if len(projection.Conflicts) != 0 {
		t.Fatalf("conflicts = %+v, want none", projection.Conflicts)
	}
	if !projection.HasCheckpoint || projection.Newest.ID != checkpointTwo {
		t.Fatalf("newest = %+v, want checkpoint two", projection.Newest)
	}
	if projection.Newest.EventID != resumed.ID {
		t.Fatalf("newest event = %s, want resumed event %s", projection.Newest.EventID, resumed.ID)
	}
	if projection.Provider.ID != "codex" || projection.Provider.Version != "0.147.0" {
		t.Fatalf("provider = %+v, want codex 0.147.0", projection.Provider)
	}
	if projection.Provider.IdentityRecordID != identityRecord || projection.Provider.Confidence != "exact" {
		t.Fatalf("provider identity = %+v, want identity record exact", projection.Provider)
	}
	if !projection.HasTerminal || projection.Terminal.Backend != "tmux" || projection.Terminal.NativeSessionID != "native-1" {
		t.Fatalf("terminal = %+v, want tmux native-1", projection.Terminal)
	}
	if projection.Terminal.EventID != resumed.ID {
		t.Fatalf("terminal event = %s, want resumed event %s", projection.Terminal.EventID, resumed.ID)
	}
	if projection.OwnerHostID != testHostID {
		t.Fatalf("owner host = %q, want %q", projection.OwnerHostID, testHostID)
	}
	if projection.LocalRole != "owner" {
		t.Fatalf("local role = %q, want owner", projection.LocalRole)
	}
	if projection.Parked != nil {
		t.Fatalf("parked = %+v, want nil", projection.Parked)
	}
}

// TestProjectDerivesV4TerminalBinding requires the v4 terminal binding
// digest path through the repository: a v4 terminal.created event
// derives the binding digest with its backend.
func TestProjectDerivesV4TerminalBinding(t *testing.T) {
	record := buildRecord(t, nil)
	builder := openChain(t, record)
	decoded, err := DecodeRecord(record)
	if err != nil {
		t.Fatalf("DecodeRecord error = %v", err)
	}
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 1, eventType: "session.created", payload: createdPayload(decoded.RecordID)})
	terminal := builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 2, eventType: "terminal.created", payload: terminalV4Payload(), schemaVersion: "4.0.0"})
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 3, eventType: "provider.launched", payload: launchedPayload("codex")})

	projection, err := (&Projector{Repo: builder.repo}).Project(builder.sessionID)
	if err != nil {
		t.Fatalf("Project error = %v", err)
	}
	if projection.State != StateRunning {
		t.Fatalf("state = %q, want running", projection.State)
	}
	if !projection.HasTerminal || projection.Terminal.BindingID != bindingID || projection.Terminal.Backend != "ax.tmux" {
		t.Fatalf("terminal = %+v, want binding digest on ax.tmux", projection.Terminal)
	}
	if projection.Terminal.EventID != terminal.ID {
		t.Fatalf("terminal event = %s, want %s", projection.Terminal.EventID, terminal.ID)
	}
}

// TestReduceKeepsFactEventsStateless requires every fact-class event to
// leave a running session running: facts record data, never lifecycle.
func TestReduceKeepsFactEventsStateless(t *testing.T) {
	record := buildRecord(t, nil)
	decoded, err := DecodeRecord(record)
	if err != nil {
		t.Fatalf("DecodeRecord error = %v", err)
	}
	facts := []eventSpec{
		{typ: "provider.identified", payload: identifiedPayload()},
		{typ: "sync.completed", payload: syncPayload()},
		{typ: "fork.created", payload: forkPayload()},
		{typ: "profile.changed", payload: profilePayload()},
		{typ: "takeover.force_confirmed", payload: forceConfirmedPayload()},
		{typ: "replica.replace_confirmed", payload: replaceConfirmedPayload()},
		{typ: "task_board.adopted", payload: taskBoardAdoptedPayload()},
		{typ: "tombstone.issued", payload: tombstoneIssuedPayload()},
		{typ: "tombstone.resolved", payload: tombstoneResolvedPayload()},
	}
	base := []eventSpec{
		{typ: "session.created", payload: createdPayload(decoded.RecordID)},
		{typ: "provider.launched", payload: launchedPayload("codex")},
	}
	for _, fact := range facts {
		chain := append(append([]eventSpec{}, base...), fact)
		projection := reduceDirect(t, record, chain)
		if projection.State != StateRunning {
			t.Fatalf("fact %s state = %q, want running", fact.typ, projection.State)
		}
	}
}

// TestReduceAdmitsSameStateRestatement requires a confirming event
// while already in its target state to restate rather than refuse: a
// restatement performs no transition, so the table needs no edge for
// it. Deleting the restatement rule breaks both rows below.
func TestReduceAdmitsSameStateRestatement(t *testing.T) {
	record := buildRecord(t, nil)
	decoded, err := DecodeRecord(record)
	if err != nil {
		t.Fatalf("DecodeRecord error = %v", err)
	}
	idle := reduceDirect(t, record, []eventSpec{
		{typ: "session.created", payload: createdPayload(decoded.RecordID)},
		{typ: "provider.launched", payload: launchedPayload("codex")},
		{typ: "session.idle", payload: idlePayload()},
		{typ: "session.idle", payload: idlePayload()},
	})
	if idle.State != StateIdle {
		t.Fatalf("state = %q, want idle after restatement", idle.State)
	}
	failed := reduceDirect(t, record, []eventSpec{
		{typ: "session.created", payload: createdPayload(decoded.RecordID)},
		{typ: "provider.launched", payload: launchedPayload("codex")},
		{typ: "session.failed", payload: failedPayload()},
		{typ: "session.failed", payload: failedPayload()},
	})
	if failed.State != StateFailed {
		t.Fatalf("state = %q, want failed after restatement", failed.State)
	}
}

// TestReduceTreatsUnknownV1TypeAsInert requires an unknown v1 event
// type to be retained without changing derived state, per Section 5.2.
func TestReduceTreatsUnknownV1TypeAsInert(t *testing.T) {
	record := buildRecord(t, nil)
	decoded, err := DecodeRecord(record)
	if err != nil {
		t.Fatalf("DecodeRecord error = %v", err)
	}
	// An unknown v1 type passes the canonical owner with an object
	// payload and must not move the derivation.
	chain := []eventSpec{
		{typ: "session.created", payload: createdPayload(decoded.RecordID)},
		{typ: "provider.launched", payload: launchedPayload("codex")},
		{typ: "session.future_probe", payload: map[string]any{"note": "tomorrow"}},
	}
	projection := reduceDirect(t, record, chain)
	if projection.State != StateRunning {
		t.Fatalf("state = %q, want running past an inert unknown type", projection.State)
	}
}
