package terminstance

import (
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// TestExecuteRechecksBeforeThirdEffect proves authorization and
// generation are rechecked immediately before EACH side effect, not
// just the first two: a lease and a generation rotation between the
// second and third stop effects are both caught at the third recheck,
// after two committed effects.
func TestExecuteRechecksBeforeThirdEffect(t *testing.T) {
	t.Run("lease rotates before third effect", func(t *testing.T) {
		backend := newMockBackend()
		engine := testEngine(t, backend, testLease(), fixtureGen)
		calls := 0
		engine.CurrentLease = func() LeaseView {
			calls++
			if calls <= 3 {
				return testLease()
			}
			return LeaseView{LeaseID: fixtureLease, Epoch: 2}
		}
		context, params, source, admitted := stopFixture(t)
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationRequestStop, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_unauthorized", "ax authorization lease epoch")
		requireLiteral(t, "after", string(result.After), "unavailable")
		requireLiteral(t, "disposition", string(result.Disposition), "new_authorization")
		if len(backend.performed()) != 2 {
			t.Errorf("performed = %v, want exactly two effects before the rotated lease is caught", backend.performed())
		}
	})
	t.Run("generation rotates before third effect", func(t *testing.T) {
		backend := newMockBackend()
		engine := testEngine(t, backend, testLease(), fixtureGen)
		calls := 0
		engine.CurrentGeneration = func() string {
			calls++
			if calls <= 3 {
				return fixtureGen
			}
			return "generation-two"
		}
		context, params, source, admitted := stopFixture(t)
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationRequestStop, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_stale_generation", "backend_generation stale")
		requireLiteral(t, "after", string(result.After), "unavailable")
		requireLiteral(t, "disposition", string(result.Disposition), "status_first")
		if len(backend.performed()) != 2 {
			t.Errorf("performed = %v, want exactly two effects before the rotated generation is caught", backend.performed())
		}
	})
}

// TestExecuteStatusSessionDriftRefuses proves the reported session is
// compared on both status lookup paths: a backend reporting ANOTHER
// session's instance with identity_match true is a lying match and
// refuses, never adopted — on the exact-instance path and on the
// Session-scoped path alike. A foreign session with the canonical
// non-match form is a proven non-match and adopts absent: the report
// carries no instance facts, only the mismatch the query asked about.
func TestExecuteStatusSessionDriftRefuses(t *testing.T) {
	t.Run("exact lying match refuses", func(t *testing.T) {
		backend := newMockBackend()
		observed := matchingObservation()
		observed.SessionID = fixtureSessionB
		backend.observation = observed
		engine := testEngine(t, backend, testLease(), fixtureGen)
		_, err := engine.ExecuteStatus(contextGo(), exactStatusBody(t), mustParseState(t, "active"), testAdmitted("durable_disconnect"))
		requireRefusal(t, err, "terminal_backend_protocol_error", "status identity binding")
	})
	t.Run("exact non-match adopts absent", func(t *testing.T) {
		backend := newMockBackend()
		backend.observation = StatusObservation{
			State: terminalbackend.StateAbsent, IdentityMatch: false,
			SessionID: fixtureSessionB, InstanceID: fixtureInstance,
			BackendID: fixtureBackend, ImplVersion: fixtureImpl,
			ProtoVersion: fixtureProto, Generation: fixtureGen,
		}
		engine := testEngine(t, backend, testLease(), fixtureGen)
		report, err := engine.ExecuteStatus(contextGo(), exactStatusBody(t), mustParseState(t, "active"), testAdmitted("durable_disconnect"))
		if err != nil {
			t.Fatalf("ExecuteStatus() error = %v, want the proven non-match", err)
		}
		requireLiteral(t, "state", string(report.State), "absent")
		if report.IdentityMatch {
			t.Error("identity_match = true, want false")
		}
	})
	t.Run("session_scoped lying match refuses", func(t *testing.T) {
		body, err := ParseStatusBody([]byte(statusDoc(`null`, `null`)))
		if err != nil {
			t.Fatal(err)
		}
		backend := newMockBackend()
		observed := matchingObservation()
		observed.SessionID = fixtureSessionB
		backend.observation = observed
		engine := testEngine(t, backend, testLease(), fixtureGen)
		_, err = engine.ExecuteStatus(contextGo(), body, mustParseState(t, "absent"), testAdmitted("durable_disconnect"))
		requireRefusal(t, err, "terminal_backend_protocol_error", "status identity binding")
	})
	t.Run("session_scoped non-match adopts absent", func(t *testing.T) {
		body, err := ParseStatusBody([]byte(statusDoc(`null`, `null`)))
		if err != nil {
			t.Fatal(err)
		}
		backend := newMockBackend()
		backend.observation = StatusObservation{
			State: terminalbackend.StateAbsent, IdentityMatch: false,
			SessionID: fixtureSessionB, InstanceID: fixtureInstance,
			BackendID: fixtureBackend, ImplVersion: fixtureImpl,
			ProtoVersion: fixtureProto, Generation: fixtureGen,
		}
		engine := testEngine(t, backend, testLease(), fixtureGen)
		report, err := engine.ExecuteStatus(contextGo(), body, mustParseState(t, "absent"), testAdmitted("durable_disconnect"))
		if err != nil {
			t.Fatalf("ExecuteStatus() error = %v, want the proven non-match", err)
		}
		requireLiteral(t, "state", string(report.State), "absent")
	})
}

// TestExecuteStatusIdentityMatchEachAxis proves identity_match is the
// six-way conjunction on the exact-instance path: drift on EACH axis
// alone with the canonical absent form reads as a proven non-match,
// and a lying match on each axis refuses.
func TestExecuteStatusIdentityMatchEachAxis(t *testing.T) {
	axes := map[string]func(*StatusObservation){
		"instance":       func(observed *StatusObservation) { observed.InstanceID = "0198f4c8-8e50-7f66-8f70-bbbbbbbbbbb2" },
		"backend":        func(observed *StatusObservation) { observed.BackendID = "other.backend" },
		"implementation": func(observed *StatusObservation) { observed.ImplVersion = "1.2.4" },
		"protocol":       func(observed *StatusObservation) { observed.ProtoVersion = "1.0.1" },
		"generation":     func(observed *StatusObservation) { observed.Generation = "generation-two" },
	}
	for name, drift := range axes {
		t.Run(name+"/canonical_non_match_admits", func(t *testing.T) {
			backend := newMockBackend()
			observed := StatusObservation{
				State: terminalbackend.StateAbsent, IdentityMatch: false,
				SessionID: fixtureSessionA, InstanceID: fixtureInstance,
				BackendID: fixtureBackend, ImplVersion: fixtureImpl,
				ProtoVersion: fixtureProto, Generation: fixtureGen,
			}
			drift(&observed)
			backend.observation = observed
			engine := testEngine(t, backend, testLease(), fixtureGen)
			report, err := engine.ExecuteStatus(contextGo(), exactStatusBody(t), mustParseState(t, "active"), testAdmitted("durable_disconnect"))
			if err != nil {
				t.Fatalf("ExecuteStatus() drift on %s with the canonical absent form error = %v", name, err)
			}
			requireLiteral(t, "state", string(report.State), "absent")
		})
		t.Run(name+"/lying_match_refuses", func(t *testing.T) {
			backend := newMockBackend()
			observed := matchingObservation()
			drift(&observed)
			backend.observation = observed
			engine := testEngine(t, backend, testLease(), fixtureGen)
			_, err := engine.ExecuteStatus(contextGo(), exactStatusBody(t), mustParseState(t, "active"), testAdmitted("durable_disconnect"))
			requireRefusal(t, err, "terminal_backend_protocol_error", "status identity binding")
		})
	}
}

// TestExecuteEntryDeadlineInstantRefuses proves the deadline instant
// itself refuses at the mutating entry on every engine row: now equal
// to deadline_at cancels before any receipt or effect with the row's
// timeout code and replay_same.
func TestExecuteEntryDeadlineInstantRefuses(t *testing.T) {
	deadline, err := time.Parse(time.RFC3339Nano, fixtureDeadline)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		operation terminalbackend.Operation
		build     func(t *testing.T) (MutationContext, Params, terminalbackend.InstanceState, terminalbackend.Admitted)
		code      string
		source    string
	}{
		{terminalbackend.OperationQuiesceInput, func(t *testing.T) (MutationContext, Params, terminalbackend.InstanceState, terminalbackend.Admitted) {
			return quiesceFixture(t)
		}, "terminal_backend_timeout", "active"},
		{terminalbackend.OperationWaitSafeBoundary, func(t *testing.T) (MutationContext, Params, terminalbackend.InstanceState, terminalbackend.Admitted) {
			return waitFixture(t, ProofAXCheckpointBoundary)
		}, "quiesce_timeout", "quiescing"},
		{terminalbackend.OperationRequestStop, func(t *testing.T) (MutationContext, Params, terminalbackend.InstanceState, terminalbackend.Admitted) {
			return stopFixture(t)
		}, "stop_timeout", "quiescing"},
		{terminalbackend.OperationCreate, func(t *testing.T) (MutationContext, Params, terminalbackend.InstanceState, terminalbackend.Admitted) {
			created := testCreateContext(t)
			return created.context, created.params, mustParseState(t, "absent"), testAdmitted("terminal_state_retention")
		}, "terminal_backend_timeout", "absent"},
	}
	for _, tc := range cases {
		t.Run(string(tc.operation), func(t *testing.T) {
			backend := newMockBackend()
			engine := testEngine(t, backend, testLease(), fixtureGen)
			engine.Now = func() time.Time { return deadline }
			context, params, source, admitted := tc.build(t)
			result, err := engine.ExecuteMutating(contextGo(), tc.operation, context, params, source, admitted)
			requireRefusal(t, err, tc.code, "operation deadline")
			requireLiteral(t, "after", string(result.After), tc.source)
			requireLiteral(t, "disposition", string(result.Disposition), "replay_same")
			if len(backend.performed()) != 0 {
				t.Errorf("performed = %v, want no effects at the deadline instant", backend.performed())
			}
			if _, found, _ := engine.Store.Lookup(context.IdempotencyKey); found {
				t.Error("Lookup(key) found a receipt at the deadline instant, want none bound")
			}
		})
	}
}

// TestExecuteStatusDeadlineInstantRefuses proves the deadline instant
// refuses at the status entry too: now equal to deadline_at reports
// terminal_backend_timeout with no observation run.
func TestExecuteStatusDeadlineInstantRefuses(t *testing.T) {
	deadline, err := time.Parse(time.RFC3339Nano, fixtureDeadline)
	if err != nil {
		t.Fatal(err)
	}
	backend := newMockBackend()
	backend.observation = matchingObservation()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	engine.Now = func() time.Time { return deadline }
	_, err = engine.ExecuteStatus(contextGo(), exactStatusBody(t), mustParseState(t, "active"), testAdmitted("durable_disconnect"))
	requireRefusal(t, err, "terminal_backend_timeout", "operation deadline")
	if backend.observeCalls != 0 {
		t.Errorf("observe calls = %d, want none at the deadline instant", backend.observeCalls)
	}
}
