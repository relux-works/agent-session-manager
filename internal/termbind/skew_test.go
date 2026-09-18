package termbind

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axpane"
	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/sessquery"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/sessstate"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

func successorObservation(now time.Time) fencing.Observation {
	grant := sessrepo.FencingGrant{
		SessionID:   fixtureSession,
		RecordID:    seedDigest(0xA2),
		Token:       sessrepo.FencingToken{Epoch: 2, LeaseID: fixtureLeaseB, HolderHostID: fixtureLocalHost},
		ValidatedAt: now.Add(-time.Minute),
	}
	policy := sessrepo.FencingPolicy{RefreshInterval: time.Hour}
	return fencing.Observation{
		SessionID:   fixtureSession,
		Winner:      sessrepo.LeaseSummary{SessionID: fixtureSession, LeaseID: fixtureLeaseB, Epoch: 2, HolderHostID: fixtureLocalHost, Reason: "graceful_takeover"},
		HasWinner:   true,
		LocalHostID: fixtureLocalHost,
		Verified:    true,
		Grant:       grant,
		HasGrant:    true,
		Policy:      policy,
		Now:         now,
	}
}

func takeoverLease(t *testing.T, repository *sessrepo.Repository) {
	t.Helper()
	leases, err := repository.ListLeases(fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.CompareAndSwapLease(fixtureSession, sessrepo.LeaseExpectation{RecordID: leases[len(leases)-1].RecordID}, sessrepo.SuccessorLeaseInput{
		CreateLeaseInput: sessrepo.CreateLeaseInput{
			LeaseID:        fixtureLeaseB,
			HolderHostID:   fixtureLocalHost,
			IssuedByHostID: fixtureLocalHost,
			CreatedAt:      fixtureCreatedAt,
		},
		Reason:       "graceful_takeover",
		CheckpointID: seedDigest(0xC0),
	}); err != nil {
		t.Fatalf("CompareAndSwapLease() error = %v", err)
	}
}

func TestTakeoverSelectsNewBackendNewEvent(t *testing.T) {
	t.Parallel()
	repository, _ := chainFixture(t)
	tmux := buildTmuxUniverse(t)
	first, firstRaw, _, err := EmitTerminalCreated(repository, emitParams(t, repository), emitFacts(tmux), emitAdmission(t, tmux))
	if err != nil {
		t.Fatal(err)
	}
	storedBefore, err := repository.GetEvent(fixtureSession, first.EventID)
	if err != nil {
		t.Fatal(err)
	}
	takeoverLease(t, repository)
	conpty := buildConptyUniverse(t)
	tail := chainTail(t, repository)
	params := axpane.EmitParams{
		SessionID:     fixtureSession,
		CreatedByHost: fixtureLocalHost,
		LeaseEpoch:    2,
		LeaseID:       fixtureLeaseB,
		LeaseSequence: 1,
		Predecessors:  []string{tail.EventID},
		CreatedAt:     fixtureCreatedAt,
		Presented:     fencing.PresentedToken{SessionID: fixtureSession, Epoch: 2, LeaseID: fixtureLeaseB},
		Observation:   successorObservation(fixtureNow()),
	}
	conptyFacts := emitFacts(conpty)
	conptyFacts.BindingDigest = seedDigest(0xB2)
	second, secondRaw, resolved, err := EmitResumed(repository, params, seedDigest(0xC0), "standard", "", false, conptyFacts, emitAdmission(t, conpty))
	if err != nil {
		t.Fatalf("EmitResumed(takeover) error = %v", err)
	}
	if !resolved.Admitted.Has("durable_disconnect") {
		t.Fatalf("takeover admission = %+v, want an admitted destination backend", resolved.Admitted)
	}
	events := mustListEvents(t, repository)
	if len(events) != 3 {
		t.Fatalf("chain holds %d events, want 3 (created plus two v4)", len(events))
	}
	storedAfter, err := repository.GetEvent(fixtureSession, first.EventID)
	if err != nil {
		t.Fatal(err)
	}
	if string(storedAfter) != string(storedBefore) || string(storedBefore) != string(firstRaw) {
		t.Fatal("first v4 event bytes changed across the takeover, want append-only history")
	}
	for _, raw := range [][]byte{firstRaw, secondRaw} {
		var decoded struct {
			SessionID string `json:"session_id"`
			EventType string `json:"event_type"`
		}
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.SessionID != fixtureSession {
			t.Fatalf("event session = %q, want the one Logical Session", decoded.SessionID)
		}
		if decoded.EventType == "fork.created" {
			t.Fatal("takeover forked the Logical Session, want a new event only")
		}
	}
	var firstPayload, secondPayload struct {
		Payload map[string]any `json:"payload"`
	}
	if err := json.Unmarshal(firstRaw, &firstPayload); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(secondRaw, &secondPayload); err != nil {
		t.Fatal(err)
	}
	if firstPayload.Payload["terminal_binding_id"] == secondPayload.Payload["terminal_binding_id"] {
		t.Fatal("takeover reused the binding digest, want a new one")
	}
	if secondPayload.Payload["terminal_backend_id"] != terminalbackend.BuiltinConpty {
		t.Fatalf("takeover backend = %v, want %q", secondPayload.Payload["terminal_backend_id"], terminalbackend.BuiltinConpty)
	}
	if firstPayload.Payload["terminal_backend_id"] != terminalbackend.BuiltinTmux {
		t.Fatalf("first backend = %v, want %q", firstPayload.Payload["terminal_backend_id"], terminalbackend.BuiltinTmux)
	}
	_ = second
}

func foldEvents(record sessstate.Record, events []sessstate.Event) (sessstate.Projection, error) {
	return sessstate.Reduce(sessstate.Input{Record: record, Events: events})
}

func createdEvent(predecessor string) sessstate.Event {
	return sessstate.Event{
		ID: "sha256:1111111111111111111111111111111111111111111111111111111111111111", Type: "session.created",
		SchemaVersion: "1.0.0", SessionID: fixtureSession, CreatedByHostID: fixtureLocalHost,
		LeaseEpoch: 1, LeaseID: fixtureLeaseA, Sequence: 1, Predecessors: []string{predecessor},
		Payload: map[string]any{
			"session_record_id": predecessor, "bootstrap_operation_id": fixtureBootstrap, "first_checkpoint_operation_id": fixtureOtherOp,
		},
	}
}

func TestVersionSelectionBothDirections(t *testing.T) {
	t.Parallel()
	repository, recordID := chainFixture(t)
	recordJSON, err := repository.GetRecord(fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	record, err := sessstate.DecodeRecord(recordJSON)
	if err != nil {
		t.Fatal(err)
	}
	_ = recordID
	created := createdEvent(record.RecordID)
	t.Run("v1 derives the v1 surface", func(t *testing.T) {
		t.Parallel()
		v1 := sessstate.Event{
			ID: "sha256:2222222222222222222222222222222222222222222222222222222222222222", Type: "terminal.created",
			SchemaVersion: "1.0.0", SessionID: fixtureSession, CreatedByHostID: fixtureLocalHost,
			LeaseEpoch: 1, LeaseID: fixtureLeaseA, Sequence: 2, Predecessors: []string{created.ID},
			Payload: map[string]any{"backend": "tmux", "terminal_id": "tmux-session-1"},
		}
		projection, err := foldEvents(record, []sessstate.Event{created, v1})
		if err != nil {
			t.Fatalf("Reduce(v1) error = %v", err)
		}
		if !projection.HasTerminal || projection.Terminal.TerminalID != "tmux-session-1" {
			t.Fatalf("v1 terminal = %+v, want the v1 handle", projection.Terminal)
		}
		if projection.Terminal.BindingID != "" {
			t.Fatalf("v1 binding = %q, want empty", projection.Terminal.BindingID)
		}
	})
	t.Run("v4 derives the v4 surface", func(t *testing.T) {
		t.Parallel()
		v4 := sessstate.Event{
			ID: "sha256:3333333333333333333333333333333333333333333333333333333333333333", Type: "terminal.created",
			SchemaVersion: "4.0.0", SessionID: fixtureSession, CreatedByHostID: fixtureLocalHost,
			LeaseEpoch: 1, LeaseID: fixtureLeaseA, Sequence: 2, Predecessors: []string{created.ID},
			Payload: map[string]any{
				"terminal_binding_id": seedDigest(0xB1), "terminal_backend_id": terminalbackend.BuiltinTmux,
				"implementation_version": fixtureImpl, "protocol_version": fixtureProto,
				"evidence_ids": []any{seedDigest(0xE0)},
			},
		}
		projection, err := foldEvents(record, []sessstate.Event{created, v4})
		if err != nil {
			t.Fatalf("Reduce(v4) error = %v", err)
		}
		if !projection.HasTerminal || projection.Terminal.BindingID != seedDigest(0xB1) {
			t.Fatalf("v4 terminal = %+v, want the binding digest", projection.Terminal)
		}
		if projection.Terminal.TerminalID != "" {
			t.Fatalf("v4 terminal id = %q, want empty", projection.Terminal.TerminalID)
		}
	})
	t.Run("v4 resumed records binding and newest", func(t *testing.T) {
		t.Parallel()
		resumed := sessstate.Event{
			ID: "sha256:4444444444444444444444444444444444444444444444444444444444444444", Type: "session.resumed",
			SchemaVersion: "4.0.0", SessionID: fixtureSession, CreatedByHostID: fixtureLocalHost,
			LeaseEpoch: 1, LeaseID: fixtureLeaseA, Sequence: 2, Predecessors: []string{created.ID},
			Payload: map[string]any{
				"checkpoint_id": seedDigest(0xC9), "execution_profile": "standard", "profile_source_event_id": nil,
				"terminal_binding_id": seedDigest(0xB1), "terminal_backend_id": terminalbackend.BuiltinTmux,
				"implementation_version": fixtureImpl, "protocol_version": fixtureProto,
				"evidence_ids": []any{seedDigest(0xE0)},
			},
		}
		projection, err := foldEvents(record, []sessstate.Event{created, resumed})
		if err != nil {
			t.Fatalf("Reduce(resumed v4) error = %v", err)
		}
		if projection.Terminal.BindingID != seedDigest(0xB1) {
			t.Fatalf("resumed terminal = %+v, want the binding digest", projection.Terminal)
		}
		if !projection.HasCheckpoint || projection.Newest.ID != seedDigest(0xC9) {
			t.Fatalf("resumed newest = %+v, want the resumed checkpoint", projection.Newest)
		}
	})
}

func TestV4RetainedAsImmutableHistory(t *testing.T) {
	t.Parallel()
	repository, _ := chainFixture(t)
	world := buildTmuxUniverse(t)
	reference, raw, _, err := EmitTerminalCreated(repository, emitParams(t, repository), emitFacts(world), emitAdmission(t, world))
	if err != nil {
		t.Fatal(err)
	}
	stored, err := repository.GetEvent(fixtureSession, reference.EventID)
	if err != nil {
		t.Fatal(err)
	}
	if string(stored) != string(raw) {
		t.Fatal("retained bytes differ from the emitted bytes")
	}
	events := mustListEvents(t, repository)
	versions := map[string]string{}
	for _, summary := range events {
		eventRaw, err := repository.GetEvent(fixtureSession, summary.EventID)
		if err != nil {
			t.Fatal(err)
		}
		var decoded struct {
			SchemaVersion string `json:"schema_version"`
			EventType     string `json:"event_type"`
		}
		if err := json.Unmarshal(eventRaw, &decoded); err != nil {
			t.Fatal(err)
		}
		versions[decoded.EventType] = decoded.SchemaVersion
	}
	if len(events) != 2 || versions["session.created"] != "1.0.0" || versions["terminal.created"] != "4.0.0" {
		t.Fatalf("chain versions = %v, want exactly the appended events (no lower-version replacement)", versions)
	}
	reader := &sessquery.Reader{Local: repository}
	summaries, err := reader.List()
	if err != nil {
		t.Fatalf("Reader.List() error = %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("Reader.List() rows = %d, want 1", len(summaries))
	}
	if !summaries[0].Projection.HasTerminal || summaries[0].Projection.Terminal.BindingID != seedDigest(0xB1) {
		t.Fatalf("query terminal = %+v, want the v4 binding digest", summaries[0].Projection.Terminal)
	}
	if after := mustListEvents(t, repository); len(after) != 2 {
		t.Fatalf("chain holds %d events after the query read, want 2 (reads write nothing)", len(after))
	}
}

func TestV1ReaderSeesV4AsInert(t *testing.T) {
	t.Parallel()
	repository, _ := chainFixture(t)
	world := buildTmuxUniverse(t)
	reference, _, _, err := EmitTerminalCreated(repository, emitParams(t, repository), emitFacts(world), emitAdmission(t, world))
	if err != nil {
		t.Fatal(err)
	}
	stored, err := repository.GetEvent(fixtureSession, reference.EventID)
	if err != nil {
		t.Fatal(err)
	}
	// The v1 derivation input for terminal.created is the backend plus
	// terminal_id members; the v4 payload carries neither, so a v1 reader
	// retains the bytes but derives no terminal state from them.
	event, err := sessstate.DecodeEvent(stored)
	if err != nil {
		t.Fatalf("DecodeEvent(v4) error = %v", err)
	}
	if event.SchemaVersion != "4.0.0" {
		t.Fatalf("schema_version = %q, want 4.0.0", event.SchemaVersion)
	}
	for _, member := range []string{"backend", "terminal_id", "terminal_backend", "native_session_id"} {
		if _, present := event.Payload[member]; present {
			t.Fatalf("v4 payload carries v1 member %q: a v1 reader could derive from it", member)
		}
	}
}
