package terminstance

import (
	"errors"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

func exactStatusBody(t *testing.T) StatusBody {
	t.Helper()
	body, err := ParseStatusBody([]byte(statusDoc(`"`+fixtureInstance+`"`, `"generation-one"`)))
	if err != nil {
		t.Fatalf("ParseStatusBody() error = %v", err)
	}
	return body
}

func matchingObservation() StatusObservation {
	effect := terminalbackend.EffectInputClosed
	return StatusObservation{
		State:           terminalbackend.StateActive,
		IdentityMatch:   true,
		WrapperPresent:  true,
		Attachable:      false,
		LastEffect:      &effect,
		EvidenceIDs:     []string{fixtureDigestA},
		SessionID:       fixtureSessionA,
		InstanceID:      fixtureInstance,
		BackendID:       fixtureBackend,
		ImplVersion:     fixtureImpl,
		ProtoVersion:    fixtureProto,
		Generation:      fixtureGen,
		AttachEvidenced: false,
	}
}

// TestExecuteStatusAdoptsReportedState drives the status row through the
// production entry: the reported state is adopted verbatim — including
// drift from the caller's source — with no receipt, no side effects
// and no authorization.
func TestExecuteStatusAdoptsReportedState(t *testing.T) {
	backend := newMockBackend()
	backend.observation = matchingObservation()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	report, err := engine.ExecuteStatus(contextGo(), exactStatusBody(t), mustParseState(t, "parked"), testAdmitted("durable_disconnect"))
	if err != nil {
		t.Fatalf("ExecuteStatus() error = %v", err)
	}
	requireLiteral(t, "state", string(report.State), "active")
	if !report.IdentityMatch {
		t.Error("identity_match = false, want true")
	}
	if len(backend.performed()) != 0 {
		t.Errorf("performed = %v, want no effects on status", backend.performed())
	}
	image, err := engine.Store.Export()
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if len(image) != 0 {
		t.Errorf("store export = %q, want no receipts on status", string(image))
	}
}

// TestExecuteStatusAllSources drives status from every one of the eight
// states through the production entry: the same-state row admits them
// all and transitions none.
func TestExecuteStatusAllSources(t *testing.T) {
	for _, source := range []string{"absent", "creating", "parked", "active", "quiescing", "stopped", "stale_fenced", "unavailable"} {
		t.Run(source, func(t *testing.T) {
			backend := newMockBackend()
			observed := matchingObservation()
			observed.State = mustParseState(t, source)
			backend.observation = observed
			engine := testEngine(t, backend, testLease(), fixtureGen)
			report, err := engine.ExecuteStatus(contextGo(), exactStatusBody(t), mustParseState(t, source), testAdmitted("durable_disconnect"))
			if err != nil {
				t.Fatalf("ExecuteStatus(%s) error = %v", source, err)
			}
			requireLiteral(t, "state", string(report.State), source)
		})
	}
	// An unknown (empty) source is admitted too: the report is adopted
	// as the first observation.
	backend := newMockBackend()
	backend.observation = matchingObservation()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	report, err := engine.ExecuteStatus(contextGo(), exactStatusBody(t), "", testAdmitted("durable_disconnect"))
	if err != nil {
		t.Fatalf("ExecuteStatus(unknown source) error = %v", err)
	}
	requireLiteral(t, "state", string(report.State), "active")

	// A malformed source refuses before any observation runs, even
	// with a valid report scripted behind it.
	backendGarbage := newMockBackend()
	backendGarbage.observation = matchingObservation()
	engineGarbage := testEngine(t, backendGarbage, testLease(), fixtureGen)
	_, err = engineGarbage.ExecuteStatus(contextGo(), exactStatusBody(t), "bogus", testAdmitted("durable_disconnect"))
	requireRefusal(t, err, "terminal_backend_protocol_error", "lifecycle state vocabulary")
	if backendGarbage.observeCalls != 0 {
		t.Errorf("observe calls = %d, want none on the malformed source", backendGarbage.observeCalls)
	}
}

// TestExecuteStatusReportMemberRefusals proves the reported identity
// tuple is shape-validated before comparison: a malformed reported
// session, backend, version, instance or generation refuses through
// the production entry instead of comparing.
func TestExecuteStatusReportMemberRefusals(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*StatusObservation)
		code   string
		detail string
	}{
		{"session", func(observed *StatusObservation) { observed.SessionID = "nope" }, "terminal_backend_protocol_error", "status report session"},
		{"backend", func(observed *StatusObservation) { observed.BackendID = "Has-Upper" }, "terminal_backend_not_found", "terminal_backend_id grammar"},
		{"implementation", func(observed *StatusObservation) { observed.ImplVersion = "1.2" }, "terminal_backend_implementation_drift", "implementation_version semver"},
		{"protocol", func(observed *StatusObservation) { observed.ProtoVersion = "2.0.0" }, "terminal_backend_implementation_drift", "protocol_version major 1"},
		{"instance", func(observed *StatusObservation) { observed.InstanceID = "nope" }, "terminal_backend_protocol_error", "status report instance"},
		{"generation", func(observed *StatusObservation) { observed.Generation = strings.Repeat("g", 257) }, "terminal_backend_protocol_error", "status report generation"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			backend := newMockBackend()
			observed := matchingObservation()
			tc.mutate(&observed)
			backend.observation = observed
			engine := testEngine(t, backend, testLease(), fixtureGen)
			_, err := engine.ExecuteStatus(contextGo(), exactStatusBody(t), mustParseState(t, "active"), testAdmitted("durable_disconnect"))
			requireRefusal(t, err, tc.code, tc.detail)
		})
	}
}

// TestExecuteStatusNonMatchAdoptsAbsent proves a successful non-match
// read adopts the canonical absent form: drifted generation with the
// false form is a proven absence, not an error.
func TestExecuteStatusNonMatchAdoptsAbsent(t *testing.T) {
	backend := newMockBackend()
	backend.observation = StatusObservation{
		State:         terminalbackend.StateAbsent,
		IdentityMatch: false,
		SessionID:     fixtureSessionA,
		InstanceID:    fixtureInstance,
		BackendID:     fixtureBackend,
		ImplVersion:   fixtureImpl,
		ProtoVersion:  fixtureProto,
		Generation:    "generation-two",
	}
	engine := testEngine(t, backend, testLease(), fixtureGen)
	report, err := engine.ExecuteStatus(contextGo(), exactStatusBody(t), mustParseState(t, "active"), testAdmitted("durable_disconnect"))
	if err != nil {
		t.Fatalf("ExecuteStatus() error = %v", err)
	}
	requireLiteral(t, "state", string(report.State), "absent")
	if report.IdentityMatch {
		t.Error("identity_match = true, want false")
	}
}

// TestExecuteStatusUnknownIsNeverAbsent proves a timeout or read failure
// is unknown, never absent: coded failures pass their row code through
// with no report, uncoded failures refuse unavailable with no report,
// and a code outside the status set is rejected.
func TestExecuteStatusUnknownIsNeverAbsent(t *testing.T) {
	backend := newMockBackend()
	backend.observeErr = refuse("terminal_backend_timeout", "test timeout")
	engine := testEngine(t, backend, testLease(), fixtureGen)
	_, err := engine.ExecuteStatus(contextGo(), exactStatusBody(t), mustParseState(t, "active"), testAdmitted("durable_disconnect"))
	requireRefusal(t, err, "terminal_backend_timeout", "status observation unknown")

	backendUncoded := newMockBackend()
	backendUncoded.observeErr = errors.New("read failed")
	engineUncoded := testEngine(t, backendUncoded, testLease(), fixtureGen)
	_, err = engineUncoded.ExecuteStatus(contextGo(), exactStatusBody(t), mustParseState(t, "active"), testAdmitted("durable_disconnect"))
	requireRefusal(t, err, "terminal_backend_unavailable", "status observation unknown")

	backendOutside := newMockBackend()
	backendOutside.observeErr = refuse("quiesce_timeout", "test outside set")
	engineOutside := testEngine(t, backendOutside, testLease(), fixtureGen)
	_, err = engineOutside.ExecuteStatus(contextGo(), exactStatusBody(t), mustParseState(t, "active"), testAdmitted("durable_disconnect"))
	requireRefusal(t, err, "terminal_backend_protocol_error", "operation error vocabulary")
}

// TestExecuteStatusReportValidation proves the reported match must equal
// the compared tuple and the landed status rules admit the report: a
// lying match, a non-canonical false form and unsorted evidence all
// refuse through the production entry.
func TestExecuteStatusReportValidation(t *testing.T) {
	backend := newMockBackend()
	lying := matchingObservation()
	lying.IdentityMatch = false
	backend.observation = lying
	engine := testEngine(t, backend, testLease(), fixtureGen)
	_, err := engine.ExecuteStatus(contextGo(), exactStatusBody(t), mustParseState(t, "active"), testAdmitted("durable_disconnect"))
	requireRefusal(t, err, "terminal_backend_protocol_error", "status identity binding")

	backendForm := newMockBackend()
	noncanonical := matchingObservation()
	noncanonical.IdentityMatch = false
	noncanonical.WrapperPresent = true
	noncanonical.Generation = "generation-two"
	backendForm.observation = noncanonical
	engineForm := testEngine(t, backendForm, testLease(), fixtureGen)
	_, err = engineForm.ExecuteStatus(contextGo(), exactStatusBody(t), mustParseState(t, "active"), testAdmitted("durable_disconnect"))
	requireRefusal(t, err, "terminal_backend_protocol_error", "status identity binding")

	backendEvidence := newMockBackend()
	unsorted := matchingObservation()
	unsorted.EvidenceIDs = []string{fixtureDigestB, fixtureDigestA}
	backendEvidence.observation = unsorted
	engineEvidence := testEngine(t, backendEvidence, testLease(), fixtureGen)
	_, err = engineEvidence.ExecuteStatus(contextGo(), exactStatusBody(t), mustParseState(t, "active"), testAdmitted("durable_disconnect"))
	requireRefusal(t, err, "terminal_backend_protocol_error", "status evidence order")
}

// TestExecuteStatusMatchedTupleFalseFormRefuses proves the engine's own
// match-equality arm, past which the landed gate alone would admit: a
// report whose tuple matches but which claims non-match with the
// canonical absent form would adopt a lying absence, so the engine
// refuses it before the landed check runs.
func TestExecuteStatusMatchedTupleFalseFormRefuses(t *testing.T) {
	backend := newMockBackend()
	lying := StatusObservation{
		State:         terminalbackend.StateAbsent,
		IdentityMatch: false,
		SessionID:     fixtureSessionA,
		InstanceID:    fixtureInstance,
		BackendID:     fixtureBackend,
		ImplVersion:   fixtureImpl,
		ProtoVersion:  fixtureProto,
		Generation:    fixtureGen,
	}
	backend.observation = lying
	engine := testEngine(t, backend, testLease(), fixtureGen)
	_, err := engine.ExecuteStatus(contextGo(), exactStatusBody(t), mustParseState(t, "active"), testAdmitted("durable_disconnect"))
	requireRefusal(t, err, "terminal_backend_protocol_error", "status identity binding")
}

// TestExecuteStatusProviderConditional proves provider_process_observation
// is required exactly when provider observation is requested.
func TestExecuteStatusProviderConditional(t *testing.T) {
	raw := statusDoc(`"`+fixtureInstance+`"`, `"generation-one"`)
	withProvider := func(t *testing.T) StatusBody {
		t.Helper()
		body, err := ParseStatusBody([]byte(withObservation(raw)))
		if err != nil {
			t.Fatalf("ParseStatusBody() error = %v", err)
		}
		return body
	}
	backend := newMockBackend()
	backend.observation = matchingObservation()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	_, err := engine.ExecuteStatus(contextGo(), withProvider(t), mustParseState(t, "active"), testAdmitted("durable_disconnect"))
	requireRefusal(t, err, "terminal_backend_capability_unproven", "operation capability conditional")

	backendProven := newMockBackend()
	observed := matchingObservation()
	present := true
	observed.ProviderPresent = &present
	observed.ProviderRequested = true
	observed.ProviderEvidenced = true
	backendProven.observation = observed
	engineProven := testEngine(t, backendProven, testLease(), fixtureGen)
	report, err := engineProven.ExecuteStatus(contextGo(), withProvider(t), mustParseState(t, "active"), testAdmitted("provider_process_observation"))
	if err != nil {
		t.Fatalf("ExecuteStatus() error = %v", err)
	}
	if report.ProviderPresent == nil || !*report.ProviderPresent {
		t.Error("provider_present is not true, want the evidenced observation")
	}
}

func withObservation(raw string) string {
	before := `"include_provider_observation":false`
	after := `"include_provider_observation":true`
	for i := 0; i+len(before) <= len(raw); i++ {
		if raw[i:i+len(before)] == before {
			return raw[:i] + after + raw[i+len(before):]
		}
	}
	return raw
}

// TestExecuteStatusSessionScoped proves the Session-scoped lookup adopts
// the reported instance: both-null members query, the reported session
// must match, and the reported state is adopted.
func TestExecuteStatusSessionScoped(t *testing.T) {
	body, err := ParseStatusBody([]byte(statusDoc(`null`, `null`)))
	if err != nil {
		t.Fatalf("ParseStatusBody() error = %v", err)
	}
	backend := newMockBackend()
	backend.observation = matchingObservation()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	report, err := engine.ExecuteStatus(contextGo(), body, mustParseState(t, "absent"), testAdmitted("durable_disconnect"))
	if err != nil {
		t.Fatalf("ExecuteStatus() error = %v", err)
	}
	requireLiteral(t, "state", string(report.State), "active")
}
