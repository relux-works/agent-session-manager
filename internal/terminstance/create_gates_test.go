package terminstance

import (
	"testing"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// TestExecuteCreateEntryAuthorizationRefuses proves the create row's
// entry authorization gate through the production entry: a control-kind,
// an expired, and a foreign-lease authorization each refuse before any
// receipt is bound — binding one would poison the
// (session_id, bootstrap_operation_id) key — with no side effect.
func TestExecuteCreateEntryAuthorizationRefuses(t *testing.T) {
	t.Run("control kind presented as create", func(t *testing.T) {
		create := testCreateContext(t)
		create.context.Authorization = testAuth(t)
		backend := newMockBackend()
		engine := testEngine(t, backend, testLease(), fixtureGen)
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationCreate, create.context, create.params, mustParseState(t, "absent"), testAdmitted("terminal_state_retention"))
		requireRefusal(t, err, "terminal_backend_unauthorized", "ax authorization kind binding")
		requireLiteral(t, "after", string(result.After), "absent")
		requireLiteral(t, "disposition", string(result.Disposition), "new_authorization")
		if len(backend.performed()) != 0 {
			t.Errorf("performed = %v, want no effects on the unauthorized create", backend.performed())
		}
		if _, found, _ := engine.Store.Lookup(create.context.IdempotencyKey); found {
			t.Error("Lookup(key) found a receipt for the unauthorized create, want none bound")
		}
	})
	t.Run("expired", func(t *testing.T) {
		// The deadline lies past the expiry and Now sits between
		// them, so the expiry arm fires while the deadline passes:
		// under a create-exempting mutant the deadline passes too,
		// the receipt binds, and only the receipt assertion below
		// distinguishes the mutant (the recheck reports the same
		// literal). This member kills on the receipt, not the code.
		create := testCreateContextWithDeadline(t, fixtureLateDeadline)
		backend := newMockBackend()
		engine := testEngine(t, backend, testLease(), fixtureGen)
		engine.Now = fixturePastExpiry
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationCreate, create.context, create.params, mustParseState(t, "absent"), testAdmitted("terminal_state_retention"))
		requireRefusal(t, err, "terminal_backend_unauthorized", "ax authorization expiry")
		requireLiteral(t, "after", string(result.After), "absent")
		if len(backend.performed()) != 0 {
			t.Errorf("performed = %v, want no effects on the expired create", backend.performed())
		}
		if _, found, _ := engine.Store.Lookup(create.context.IdempotencyKey); found {
			t.Error("Lookup(key) found a receipt for the expired create, want none bound")
		}
	})
	t.Run("foreign lease", func(t *testing.T) {
		create := testCreateContext(t)
		backend := newMockBackend()
		engine := testEngine(t, backend, LeaseView{LeaseID: fixtureLeaseB, Epoch: 1}, fixtureGen)
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationCreate, create.context, create.params, mustParseState(t, "absent"), testAdmitted("terminal_state_retention"))
		requireRefusal(t, err, "terminal_backend_unauthorized", "ax authorization lease identity")
		requireLiteral(t, "after", string(result.After), "absent")
		if len(backend.performed()) != 0 {
			t.Errorf("performed = %v, want no effects under the foreign lease", backend.performed())
		}
		if _, found, _ := engine.Store.Lookup(create.context.IdempotencyKey); found {
			t.Error("Lookup(key) found a receipt for the foreign-lease create, want none bound")
		}
	})
}

// TestExecuteCreateEntryGenerationRefuses proves the create row's entry
// generation gate: a stale generation refuses before any receipt is
// bound, with the landed stale class.
func TestExecuteCreateEntryGenerationRefuses(t *testing.T) {
	create := testCreateContext(t)
	backend := newMockBackend()
	engine := testEngine(t, backend, testLease(), "generation-two")
	result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationCreate, create.context, create.params, mustParseState(t, "absent"), testAdmitted("terminal_state_retention"))
	requireRefusal(t, err, "terminal_backend_stale_generation", "backend_generation stale")
	requireLiteral(t, "after", string(result.After), "absent")
	requireLiteral(t, "disposition", string(result.Disposition), "status_first")
	if len(backend.performed()) != 0 {
		t.Errorf("performed = %v, want no effects under the stale generation", backend.performed())
	}
	if _, found, _ := engine.Store.Lookup(create.context.IdempotencyKey); found {
		t.Error("Lookup(key) found a receipt for the stale create, want none bound")
	}
}

// TestExecuteCreateHeadlessFromAbsentNeedsCapability proves the
// headless_creation conditional exactly at the absent source: a
// headless create from absent without the capability refuses with no
// effect and no receipt, while the same create with the capability
// admitted parks.
func TestExecuteCreateHeadlessFromAbsentNeedsCapability(t *testing.T) {
	headless := testCreateContext(t)
	headless.params.Interactive = false
	backend := newMockBackend()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationCreate, headless.context, headless.params, mustParseState(t, "absent"), testAdmitted("terminal_state_retention"))
	requireRefusal(t, err, "terminal_backend_capability_unproven", "operation capability conditional")
	requireLiteral(t, "after", string(result.After), "absent")
	if len(backend.performed()) != 0 {
		t.Errorf("performed = %v, want no effects without headless_creation", backend.performed())
	}
	if _, found, _ := engine.Store.Lookup(headless.context.IdempotencyKey); found {
		t.Error("Lookup(key) found a receipt for the unproven headless create, want none bound")
	}

	admitted := testCreateContext(t)
	admitted.params.Interactive = false
	backendAdmitted := newMockBackend()
	engineAdmitted := testEngine(t, backendAdmitted, testLease(), fixtureGen)
	parked, err := engineAdmitted.ExecuteMutating(contextGo(), terminalbackend.OperationCreate, admitted.context, admitted.params, mustParseState(t, "absent"), testAdmitted("headless_creation"))
	if err != nil {
		t.Fatalf("ExecuteMutating(headless with headless_creation) error = %v", err)
	}
	requireLiteral(t, "after", string(parked.After), "parked")
}

// TestExecuteCreateCommittedTimeoutReportsStatusFirst proves create's
// second-effect committed timeout: binding_persisted committed, then
// wrapper_started reports terminal_backend_timeout, so the local
// observation moves to unavailable with status_first.
func TestExecuteCreateCommittedTimeoutReportsStatusFirst(t *testing.T) {
	create := testCreateContext(t)
	backend := newMockBackend()
	backend.effectErrors[terminalbackend.EffectWrapperStarted] = refuse("terminal_backend_timeout", "test wrapper timeout")
	engine := testEngine(t, backend, testLease(), fixtureGen)
	result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationCreate, create.context, create.params, mustParseState(t, "absent"), testAdmitted("terminal_state_retention"))
	requireRefusal(t, err, "terminal_backend_timeout", "backend effect refused")
	requireLiteral(t, "after", string(result.After), "unavailable")
	requireLiteral(t, "disposition", string(result.Disposition), "status_first")
	performed := backend.performed()
	if len(performed) != 2 || performed[0] != terminalbackend.EffectBindingPersisted || performed[1] != terminalbackend.EffectWrapperStarted {
		t.Errorf("performed = %v, want [binding_persisted wrapper_started] (first committed, second attempted)", performed)
	}
	if len(result.Effects) != 1 || result.Effects[0] != terminalbackend.EffectBindingPersisted {
		t.Errorf("result effects = %v, want exactly the committed [binding_persisted] prefix", result.Effects)
	}
}
