package axpane

import (
	"encoding/json"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

func bindingFacts() TerminalBindingFacts {
	return TerminalBindingFacts{
		BindingDigest:         seedDigest(0xB1),
		BackendID:             "ax.tmux",
		ImplementationVersion: "2.1.0",
		ProtocolVersion:       "1.1.0",
		ProtocolVersions:      []string{"1.0.0", "1.1.0"},
		EvidenceIDs:           []string{seedDigest(0xE0), seedDigest(0xE1)},
	}
}

func TestParkedPayloadVocabulary(t *testing.T) {
	t.Parallel()
	for _, reason := range []fencing.ParkReason{
		fencing.ParkRemoteOwner, fencing.ParkStaleOwner, fencing.ParkRestorePolicy, fencing.ParkFailedHandoff,
	} {
		payload, err := ParkedPayload(reason, fixtureLeaseA)
		if err != nil {
			t.Fatalf("ParkedPayload(%q) error = %v", string(reason), err)
		}
		if payload["reason"] != string(reason) || payload["winning_lease_id"] != fixtureLeaseA {
			t.Fatalf("ParkedPayload(%q) = %v", string(reason), payload)
		}
		if len(payload) != 2 {
			t.Fatalf("ParkedPayload(%q) carries %d members, want exactly 2", string(reason), len(payload))
		}
	}
	if _, err := ParkedPayload(fencing.ParkReason("bogus"), fixtureLeaseA); err == nil {
		t.Fatal("ParkedPayload(bogus) succeeded, want refusal")
	}
	if _, err := ParkedPayload(fencing.ParkRemoteOwner, "not-a-lease"); err == nil {
		t.Fatal("ParkedPayload(bad lease) succeeded, want refusal")
	}
}

func TestTerminalCreatedPayload(t *testing.T) {
	t.Parallel()
	payload, err := TerminalCreatedPayload(bindingFacts())
	if err != nil {
		t.Fatalf("TerminalCreatedPayload() error = %v", err)
	}
	want := map[string]any{
		"terminal_binding_id":    seedDigest(0xB1),
		"terminal_backend_id":    "ax.tmux",
		"implementation_version": "2.1.0",
		"protocol_version":       "1.1.0",
		"evidence_ids":           []string{seedDigest(0xE0), seedDigest(0xE1)},
	}
	raw, _ := json.Marshal(payload)
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != len(want) {
		t.Fatalf("TerminalCreatedPayload() members = %v, want exactly %v", decoded, want)
	}
	for name, value := range want {
		round, _ := json.Marshal(decoded[name])
		expect, _ := json.Marshal(value)
		if string(round) != string(expect) {
			t.Fatalf("TerminalCreatedPayload()[%q] = %s, want %s", name, round, expect)
		}
	}
	cases := []struct {
		name   string
		mutate func(*TerminalBindingFacts)
	}{
		{"bad digest", func(facts *TerminalBindingFacts) { facts.BindingDigest = "nope" }},
		{"bad backend", func(facts *TerminalBindingFacts) { facts.BackendID = "Bogus!" }},
		{"bad versions", func(facts *TerminalBindingFacts) { facts.ProtocolVersion = "2.0.0" }},
		{"empty evidence", func(facts *TerminalBindingFacts) { facts.EvidenceIDs = nil }},
		{"unsorted evidence", func(facts *TerminalBindingFacts) {
			facts.EvidenceIDs = []string{seedDigest(0xE1), seedDigest(0xE0)}
		}},
		{"bad evidence digest", func(facts *TerminalBindingFacts) { facts.EvidenceIDs = []string{"nope"} }},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			facts := bindingFacts()
			testCase.mutate(&facts)
			if _, err := TerminalCreatedPayload(facts); err == nil {
				t.Fatalf("TerminalCreatedPayload(%s) succeeded, want refusal", testCase.name)
			}
		})
	}
}

func TestResumedPayload(t *testing.T) {
	t.Parallel()
	payload, err := ResumedPayload(seedDigest(0xC9), "yolo", seedDigest(0xCA), true, bindingFacts())
	if err != nil {
		t.Fatalf("ResumedPayload() error = %v", err)
	}
	if payload["checkpoint_id"] != seedDigest(0xC9) || payload["execution_profile"] != "yolo" || payload["profile_source_event_id"] != seedDigest(0xCA) {
		t.Fatalf("ResumedPayload() = %v", payload)
	}
	if len(payload) != 8 {
		t.Fatalf("ResumedPayload() carries %d members, want exactly 8", len(payload))
	}
	nullSource, err := ResumedPayload(seedDigest(0xC9), "standard", "", false, bindingFacts())
	if err != nil {
		t.Fatalf("ResumedPayload(null source) error = %v", err)
	}
	if nullSource["profile_source_event_id"] != nil {
		t.Fatalf("ResumedPayload(null source) source = %v, want null", nullSource["profile_source_event_id"])
	}
	if _, err := ResumedPayload("nope", "standard", "", false, bindingFacts()); err == nil {
		t.Fatal("ResumedPayload(bad checkpoint) succeeded, want refusal")
	}
	if _, err := ResumedPayload(seedDigest(0xC9), "bogus", "", false, bindingFacts()); err == nil {
		t.Fatal("ResumedPayload(bad profile) succeeded, want refusal")
	}
	if _, err := ResumedPayload(seedDigest(0xC9), "yolo", "nope", true, bindingFacts()); err == nil {
		t.Fatal("ResumedPayload(bad source) succeeded, want refusal")
	}
}

// TestEmitParkedRoundTrip authors the parked event for a Decide
// parked outcome through the sessrepo owner under the locally held
// winning lease, then reads it back: the envelope lease repeats the
// winner and the payload carries the exact closed members. The
// parked decision is a verified materialization park from a failed
// chain, so both the mutation gate and the §5.7 fold gate pass.
func TestEmitParkedRoundTrip(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	appendChainEvent(t, deps.repo, "session.failed", 1, fixtureLeaseA, 2, deps.createdID, map[string]any{
		"error_code": "bootstrap_probe", "retryable": false, "operation_id": nil,
	})
	input := validInput(t, deps)
	input.MaterializationRequired = true
	input.JournalOK = true
	decision := Decide(input)
	if decision.Action != ActionParked {
		t.Fatalf("Decide() action = %q, want parked", decision.Action)
	}
	recordID, tail, err := ChainTail(deps.repo, fixtureSession)
	if err != nil {
		t.Fatalf("ChainTail() error = %v", err)
	}
	_ = recordID
	observation := fixtureObservation(fixtureNow())
	reference, raw, err := EmitParked(deps.repo, decision, EmitParams{
		SessionID:     fixtureSession,
		CreatedByHost: fixtureLocalHost,
		LeaseEpoch:    input.Observation.Winner.Epoch,
		LeaseID:       input.Observation.Winner.LeaseID,
		LeaseSequence: tail.LeaseSequence + 1,
		Predecessors:  []string{tail.EventID},
		CreatedAt:     fixtureCreatedAt,
		Presented:     fixturePresented(),
		Observation:   observation,
	})
	if err != nil {
		t.Fatalf("EmitParked() error = %v", err)
	}
	stored, err := deps.repo.GetEvent(fixtureSession, reference.EventID)
	if err != nil {
		t.Fatalf("GetEvent() error = %v", err)
	}
	if string(stored) != string(raw) {
		t.Fatal("GetEvent() bytes differ from the emitted bytes")
	}
	var decoded struct {
		SchemaVersion string `json:"schema_version"`
		EventType     string `json:"event_type"`
		LeaseEpoch    uint64 `json:"lease_epoch"`
		LeaseID       string `json:"lease_id"`
		Payload       struct {
			Reason       string `json:"reason"`
			WinningLease string `json:"winning_lease_id"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(stored, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.SchemaVersion != "1.0.0" || decoded.EventType != "session.parked" {
		t.Fatalf("emitted event = (%q, %q), want (1.0.0, session.parked)", decoded.SchemaVersion, decoded.EventType)
	}
	if decoded.LeaseEpoch != 1 || decoded.LeaseID != fixtureLeaseA {
		t.Fatalf("emitted lease = (%d, %q), want the winner", decoded.LeaseEpoch, decoded.LeaseID)
	}
	if decoded.Payload.Reason != "restore_policy" || decoded.Payload.WinningLease != fixtureLeaseA {
		t.Fatalf("emitted payload = %+v, want the parked vocabulary", decoded.Payload)
	}
}

func TestEmitTerminalV4RoundTrip(t *testing.T) {
	t.Parallel()
	repository, _ := chainFixture(t)
	events, err := repository.ListEvents(fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	tail := events[len(events)-1]
	created, err := TerminalCreatedPayload(bindingFacts())
	if err != nil {
		t.Fatal(err)
	}
	observation := fixtureObservation(fixtureNow())
	presented := fixturePresented()
	reference, _, err := Emit(repository, EmitParams{
		SessionID: fixtureSession, CreatedByHost: fixtureLocalHost,
		LeaseEpoch: 1, LeaseID: fixtureLeaseA, LeaseSequence: tail.LeaseSequence + 1,
		Predecessors: []string{tail.EventID}, CreatedAt: fixtureCreatedAt,
		EventType: "terminal.created", SchemaVersion: "4.0.0", Payload: created,
		Presented: presented, Observation: observation,
	})
	if err != nil {
		t.Fatalf("Emit(terminal.created v4) error = %v", err)
	}
	resumed, err := ResumedPayload(seedDigest(0xC9), "standard", "", false, bindingFacts())
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := Emit(repository, EmitParams{
		SessionID: fixtureSession, CreatedByHost: fixtureLocalHost,
		LeaseEpoch: 1, LeaseID: fixtureLeaseA, LeaseSequence: tail.LeaseSequence + 2,
		Predecessors: []string{reference.EventID}, CreatedAt: fixtureCreatedAt,
		EventType: "session.resumed", SchemaVersion: "4.0.0", Payload: resumed,
		Presented: presented, Observation: observation,
	})
	if err != nil {
		t.Fatalf("Emit(session.resumed v4) error = %v", err)
	}
	for _, eventID := range []string{reference.EventID, second.EventID} {
		stored, err := repository.GetEvent(fixtureSession, eventID)
		if err != nil {
			t.Fatalf("GetEvent() error = %v", err)
		}
		var decoded struct {
			SchemaVersion string `json:"schema_version"`
		}
		if err := json.Unmarshal(stored, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.SchemaVersion != "4.0.0" {
			t.Fatalf("emitted schema_version = %q, want 4.0.0", decoded.SchemaVersion)
		}
	}
}

func TestEmitParkedRefusesNonParked(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	decision := Decide(validInput(t, deps))
	if decision.Action != ActionLaunch {
		t.Fatalf("Decide() action = %q, want launch", decision.Action)
	}
	if _, _, err := EmitParked(deps.repo, decision, EmitParams{}); err == nil {
		t.Fatal("EmitParked(launch) succeeded, want refusal")
	}
	var _ = sessrepo.EventRef{}
}
