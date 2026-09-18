package termbind

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axpane"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

func emitFacts(world *universe) axpane.TerminalBindingFacts {
	return axpane.TerminalBindingFacts{
		BindingDigest:         seedDigest(0xB1),
		BackendID:             world.backendID,
		ImplementationVersion: fixtureImpl,
		ProtocolVersion:       fixtureProto,
		ProtocolVersions:      []string{"1.0.0", "1.1.0"},
		EvidenceIDs:           append([]string(nil), world.ids...),
	}
}

func emitAdmission(t *testing.T, world *universe) EvidenceAdmission {
	t.Helper()
	return EvidenceAdmission{
		Registry:      world.registry,
		Universe:      universeMap(t, world),
		RawGeneration: world.rawGeneration,
		Now:           world.now,
		Verify:        world.verify,
	}
}

func emitParams(t *testing.T, repository *sessrepo.Repository) axpane.EmitParams {
	t.Helper()
	tail := chainTail(t, repository)
	return axpane.EmitParams{
		SessionID:     fixtureSession,
		CreatedByHost: fixtureLocalHost,
		LeaseEpoch:    1,
		LeaseID:       fixtureLeaseA,
		LeaseSequence: tail.LeaseSequence + 1,
		Predecessors:  []string{tail.EventID},
		CreatedAt:     fixtureCreatedAt,
		Presented:     fixturePresented(),
		Observation:   fixtureObservation(fixtureNow()),
	}
}

func TestEmitTerminalCreatedRoundTrip(t *testing.T) {
	t.Parallel()
	repository, _ := chainFixture(t)
	world := buildTmuxUniverse(t)
	reference, raw, resolved, err := EmitTerminalCreated(repository, emitParams(t, repository), emitFacts(world), emitAdmission(t, world))
	if err != nil {
		t.Fatalf("EmitTerminalCreated() error = %v", err)
	}
	var decoded struct {
		SchemaVersion string         `json:"schema_version"`
		EventType     string         `json:"event_type"`
		EventID       string         `json:"event_id"`
		Payload       map[string]any `json:"payload"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.SchemaVersion != "4.0.0" || decoded.EventType != "terminal.created" {
		t.Fatalf("emitted event = %s %s, want 4.0.0 terminal.created", decoded.SchemaVersion, decoded.EventType)
	}
	if decoded.EventID != reference.EventID {
		t.Fatalf("event id = %q, want ref %q", decoded.EventID, reference.EventID)
	}
	if len(decoded.Payload) != 5 {
		t.Fatalf("payload members = %v, want exactly 5", decoded.Payload)
	}
	if decoded.Payload["terminal_binding_id"] != seedDigest(0xB1) || decoded.Payload["terminal_backend_id"] != terminalbackend.BuiltinTmux {
		t.Fatalf("payload identity = %v", decoded.Payload)
	}
	stored, err := repository.GetEvent(fixtureSession, reference.EventID)
	if err != nil {
		t.Fatal(err)
	}
	if string(stored) != string(raw) {
		t.Fatal("stored bytes differ from the emitted bytes")
	}
	if !resolved.Admitted.Has("durable_disconnect") {
		t.Fatalf("resolved admission = %+v, want durable_disconnect", resolved.Admitted)
	}
}

func TestEmitResumedRoundTrip(t *testing.T) {
	t.Parallel()
	t.Run("null source", func(t *testing.T) {
		t.Parallel()
		repository, _ := chainFixture(t)
		world := buildTmuxUniverse(t)
		_, raw, _, err := EmitResumed(repository, emitParams(t, repository), seedDigest(0xC9), "standard", "", false, emitFacts(world), emitAdmission(t, world))
		if err != nil {
			t.Fatalf("EmitResumed() error = %v", err)
		}
		var decoded struct {
			SchemaVersion string         `json:"schema_version"`
			EventType     string         `json:"event_type"`
			Payload       map[string]any `json:"payload"`
		}
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.SchemaVersion != "4.0.0" || decoded.EventType != "session.resumed" {
			t.Fatalf("emitted event = %s %s, want 4.0.0 session.resumed", decoded.SchemaVersion, decoded.EventType)
		}
		if len(decoded.Payload) != 8 {
			t.Fatalf("payload members = %v, want exactly 8", decoded.Payload)
		}
		if decoded.Payload["checkpoint_id"] != seedDigest(0xC9) || decoded.Payload["execution_profile"] != "standard" {
			t.Fatalf("payload pair = %v", decoded.Payload)
		}
		if decoded.Payload["profile_source_event_id"] != nil {
			t.Fatalf("source = %v, want null", decoded.Payload["profile_source_event_id"])
		}
	})
	t.Run("profile source", func(t *testing.T) {
		t.Parallel()
		repository, _ := chainFixture(t)
		world := buildTmuxUniverse(t)
		_, raw, _, err := EmitResumed(repository, emitParams(t, repository), seedDigest(0xC9), "yolo", seedDigest(0xCA), true, emitFacts(world), emitAdmission(t, world))
		if err != nil {
			t.Fatalf("EmitResumed() error = %v", err)
		}
		var decoded struct {
			Payload map[string]any `json:"payload"`
		}
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.Payload["profile_source_event_id"] != seedDigest(0xCA) || decoded.Payload["execution_profile"] != "yolo" {
			t.Fatalf("payload pair = %v", decoded.Payload)
		}
	})
	t.Run("bad profile refuses", func(t *testing.T) {
		t.Parallel()
		repository, _ := chainFixture(t)
		world := buildTmuxUniverse(t)
		if _, _, _, err := EmitResumed(repository, emitParams(t, repository), seedDigest(0xC9), "bogus", "", false, emitFacts(world), emitAdmission(t, world)); err == nil {
			t.Fatal("EmitResumed(bogus profile) succeeded, want refusal")
		}
	})
}

func TestEmitResumedCheckpointBindingIsCallerBound(t *testing.T) {
	t.Parallel()
	// Stated-bound pin for the EmitResumed doc comment: a checkpoint the
	// chain never published, carried with a yolo profile and a null
	// source, appends through the direct entry, and the landed fold then
	// reports that digest as the newest checkpoint. The wrapper decision
	// supplies both values from the fold, so this shape is unreachable on
	// the composed path; the test pins the boundary a direct caller sees.
	repository, _ := chainFixture(t)
	world := buildTmuxUniverse(t)
	hadNewest, _ := foldNewest(t, repository)
	if hadNewest {
		t.Fatal("fixture chain already folds a newest checkpoint")
	}
	fabricated := seedDigest(0xFA)
	if _, _, _, err := EmitResumed(repository, emitParams(t, repository), fabricated, "yolo", "", false, emitFacts(world), emitAdmission(t, world)); err != nil {
		t.Fatalf("EmitResumed(fabricated checkpoint) refused: %v", err)
	}
	hasNewest, newest := foldNewest(t, repository)
	if !hasNewest || newest != fabricated {
		t.Fatalf("fold newest = %v/%s, want the appended fabricated digest", hasNewest, newest)
	}
}

func TestEmitClosedShapeRefusals(t *testing.T) {
	t.Parallel()
	t.Run("unsorted evidence", func(t *testing.T) {
		t.Parallel()
		repository, _ := chainFixture(t)
		world := buildTmuxUniverse(t)
		facts := emitFacts(world)
		facts.EvidenceIDs[0], facts.EvidenceIDs[len(facts.EvidenceIDs)-1] = facts.EvidenceIDs[len(facts.EvidenceIDs)-1], facts.EvidenceIDs[0]
		if _, _, _, err := EmitTerminalCreated(repository, emitParams(t, repository), facts, emitAdmission(t, world)); err == nil {
			t.Fatal("EmitTerminalCreated(unsorted) succeeded, want refusal")
		}
	})
	t.Run("empty evidence", func(t *testing.T) {
		t.Parallel()
		repository, _ := chainFixture(t)
		world := buildTmuxUniverse(t)
		facts := emitFacts(world)
		facts.EvidenceIDs = nil
		if _, _, _, err := EmitTerminalCreated(repository, emitParams(t, repository), facts, emitAdmission(t, world)); err == nil {
			t.Fatal("EmitTerminalCreated(empty) succeeded, want refusal")
		}
	})
	t.Run("missing member at append", func(t *testing.T) {
		t.Parallel()
		repository, _ := chainFixture(t)
		world := buildTmuxUniverse(t)
		payload, err := axpane.TerminalCreatedPayload(emitFacts(world))
		if err != nil {
			t.Fatal(err)
		}
		delete(payload, "evidence_ids")
		params := emitParams(t, repository)
		params.EventType = "terminal.created"
		params.SchemaVersion = "4.0.0"
		params.Payload = payload
		_, _, err = axpane.Emit(repository, params)
		if err == nil || !strings.Contains(err.Error(), `"evidence_ids"`) {
			t.Fatalf("Emit(missing member) error = %v, want the evidence_ids member refusal", err)
		}
	})
	t.Run("extra member at append", func(t *testing.T) {
		t.Parallel()
		repository, _ := chainFixture(t)
		world := buildTmuxUniverse(t)
		payload, err := axpane.TerminalCreatedPayload(emitFacts(world))
		if err != nil {
			t.Fatal(err)
		}
		payload["native_reference"] = "tmux-0"
		params := emitParams(t, repository)
		params.EventType = "terminal.created"
		params.SchemaVersion = "4.0.0"
		params.Payload = payload
		_, _, err = axpane.Emit(repository, params)
		if err == nil || !strings.Contains(err.Error(), `"native_reference"`) {
			t.Fatalf("Emit(extra member) error = %v, want the native_reference member refusal", err)
		}
	})
	t.Run("resumed extra member at append", func(t *testing.T) {
		t.Parallel()
		repository, _ := chainFixture(t)
		world := buildTmuxUniverse(t)
		payload, err := axpane.ResumedPayload(seedDigest(0xC9), "standard", "", false, emitFacts(world))
		if err != nil {
			t.Fatal(err)
		}
		payload["terminal_instance_id"] = fixtureInstance
		params := emitParams(t, repository)
		params.EventType = "session.resumed"
		params.SchemaVersion = "4.0.0"
		params.Payload = payload
		_, _, err = axpane.Emit(repository, params)
		if err == nil || !strings.Contains(err.Error(), `"terminal_instance_id"`) {
			t.Fatalf("Emit(extra member) error = %v, want the terminal_instance_id member refusal", err)
		}
	})
}

func TestEmitResolvesEvidence(t *testing.T) {
	t.Parallel()
	t.Run("unresolvable", func(t *testing.T) {
		t.Parallel()
		repository, _ := chainFixture(t)
		world := buildTmuxUniverse(t)
		facts := emitFacts(world)
		facts.EvidenceIDs = []string{seedDigest(0xEE), world.manifestID, world.probeID}
		sortStrings(facts.EvidenceIDs)
		_, _, _, err := EmitTerminalCreated(repository, emitParams(t, repository), facts, emitAdmission(t, world))
		requireCode(t, err, "terminal_backend_manifest_probe_mismatch")
	})
	t.Run("foreign kind", func(t *testing.T) {
		t.Parallel()
		repository, _ := chainFixture(t)
		world := buildTmuxUniverse(t)
		hostileID := seedDigest(0xF0)
		admission := emitAdmission(t, world)
		admission.Universe.(mapUniverse)[hostileID] = []byte(`{"schema":"ax-terminal-endpoint","address":"10.0.0.9:2222"}`)
		facts := emitFacts(world)
		facts.EvidenceIDs = []string{world.manifestID, world.probeID, hostileID}
		sortStrings(facts.EvidenceIDs)
		_, _, _, err := EmitTerminalCreated(repository, emitParams(t, repository), facts, admission)
		requireCode(t, err, "terminal_backend_manifest_probe_mismatch")
	})
	t.Run("resumed unresolvable", func(t *testing.T) {
		t.Parallel()
		repository, _ := chainFixture(t)
		world := buildTmuxUniverse(t)
		facts := emitFacts(world)
		facts.EvidenceIDs = []string{seedDigest(0xEE), world.manifestID, world.probeID}
		sortStrings(facts.EvidenceIDs)
		_, _, _, err := EmitResumed(repository, emitParams(t, repository), seedDigest(0xC9), "standard", "", false, facts, emitAdmission(t, world))
		requireCode(t, err, "terminal_backend_manifest_probe_mismatch")
	})
}

func TestEmitUnderLosingLeaseRefuses(t *testing.T) {
	t.Parallel()
	repository, _ := chainFixture(t)
	world := buildTmuxUniverse(t)
	before := len(mustListEvents(t, repository))
	params := emitParams(t, repository)
	params.Presented.Epoch = 2
	params.LeaseEpoch = 2
	if _, _, _, err := EmitTerminalCreated(repository, params, emitFacts(world), emitAdmission(t, world)); err == nil {
		t.Fatal("EmitTerminalCreated(losing lease) succeeded, want refusal")
	}
	if after := len(mustListEvents(t, repository)); after != before {
		t.Fatalf("chain holds %d events after refused emission, want %d", after, before)
	}
}

func TestEmitAppendsIdempotently(t *testing.T) {
	t.Parallel()
	repository, _ := chainFixture(t)
	world := buildTmuxUniverse(t)
	params := emitParams(t, repository)
	first, _, _, err := EmitTerminalCreated(repository, params, emitFacts(world), emitAdmission(t, world))
	if err != nil {
		t.Fatal(err)
	}
	second, _, _, err := EmitTerminalCreated(repository, params, emitFacts(world), emitAdmission(t, world))
	if err != nil {
		t.Fatalf("identical retry error = %v", err)
	}
	if second.EventID != first.EventID {
		t.Fatalf("retry event = %q, want %q", second.EventID, first.EventID)
	}
	events := mustListEvents(t, repository)
	if len(events) != 2 {
		t.Fatalf("chain holds %d events, want 2 (created plus one)", len(events))
	}
}

func TestEmittedEventCarriesNoBindingObject(t *testing.T) {
	t.Parallel()
	repository, _ := chainFixture(t)
	world := buildTmuxUniverse(t)
	parsed, err := ParseTerminalBinding(bindingDoc(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	facts := emitFacts(world)
	facts.BindingDigest = parsed.BindingID
	_, raw, _, err := EmitTerminalCreated(repository, emitParams(t, repository), facts, emitAdmission(t, world))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{parsed.TerminalInstanceID, parsed.NativeReference, parsed.BackendGeneration} {
		if strings.Contains(string(raw), secret) {
			t.Fatalf("emitted event reaches the binding secret %q", secret)
		}
	}
	for _, member := range []string{`"native_reference"`, `"backend_generation"`, `"terminal_instance_id"`, `"binding_id":`} {
		if strings.Contains(string(raw), member) {
			t.Fatalf("emitted event carries binding member %q", member)
		}
	}
	if !strings.Contains(string(raw), parsed.BindingID) {
		t.Fatal("emitted event lacks the opaque binding digest")
	}
}

func mustListEvents(t *testing.T, repository *sessrepo.Repository) []sessrepo.EventSummary {
	t.Helper()
	events, err := repository.ListEvents(fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	return events
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
