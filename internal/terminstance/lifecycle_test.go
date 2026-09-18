package terminstance

import (
	"errors"
	"reflect"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// TestExecuteWaitSuccess drives the wait-safe-boundary row through the
// production entry: quiescing → quiescing with exactly
// safe_boundary_observed and disposition replay_same.
func TestExecuteWaitSuccess(t *testing.T) {
	backend := newMockBackend()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	context, params, source, admitted := waitFixture(t, ProofAXCheckpointBoundary)
	result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationWaitSafeBoundary, context, params, source, admitted)
	if err != nil {
		t.Fatalf("ExecuteMutating() error = %v", err)
	}
	requireLiteral(t, "before", string(result.Before), "quiescing")
	requireLiteral(t, "after", string(result.After), "quiescing")
	requireLiteral(t, "disposition", string(result.Disposition), "replay_same")
	if len(result.Effects) != 1 || result.Effects[0] != terminalbackend.EffectSafeBoundaryObserved {
		t.Errorf("effects = %v, want [safe_boundary_observed]", result.Effects)
	}
	if err := CheckResult(context, source, result); err != nil {
		t.Errorf("CheckResult(engine result) error = %v", err)
	}
}

// TestExecuteWaitCapabilityConditionals proves the wait capability rule:
// safe_boundary_observation is always required, and
// provider_process_observation is additionally required exactly for the
// provider proof kinds.
func TestExecuteWaitCapabilityConditionals(t *testing.T) {
	t.Run("provider kind without provider capability refuses", func(t *testing.T) {
		backend := newMockBackend()
		engine := testEngine(t, backend, testLease(), fixtureGen)
		context, params, source, admitted := waitFixture(t, ProofProviderProcessExit)
		// Re-key the context for the provider kind under test.
		context.IdempotencyKey = fixtureInstance + "/boundary/" + fixtureQuiescence + "/provider_process_exit"
		_, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationWaitSafeBoundary, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_capability_unproven", "operation capability conditional")
		if len(backend.performed()) != 0 {
			t.Errorf("performed = %v, want no effects", backend.performed())
		}
	})
	t.Run("provider kind with both capabilities proceeds", func(t *testing.T) {
		backend := newMockBackend()
		engine := testEngine(t, backend, testLease(), fixtureGen)
		context, params, source, _ := waitFixture(t, ProofProviderProcessExit)
		context.IdempotencyKey = fixtureInstance + "/boundary/" + fixtureQuiescence + "/provider_process_exit"
		_, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationWaitSafeBoundary, context, params, source, testAdmitted("safe_boundary_observation", "provider_process_observation"))
		if err != nil {
			t.Errorf("ExecuteMutating() error = %v, want success", err)
		}
	})
	t.Run("ax boundary without provider capability proceeds", func(t *testing.T) {
		backend := newMockBackend()
		engine := testEngine(t, backend, testLease(), fixtureGen)
		context, params, source, admitted := waitFixture(t, ProofAXCheckpointBoundary)
		if _, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationWaitSafeBoundary, context, params, source, admitted); err != nil {
			t.Errorf("ExecuteMutating() error = %v, want success", err)
		}
	})
	t.Run("provider-only admitted still needs the boundary", func(t *testing.T) {
		backend := newMockBackend()
		engine := testEngine(t, backend, testLease(), fixtureGen)
		context, params, source, _ := waitFixture(t, ProofProviderQuiescence)
		_, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationWaitSafeBoundary, context, params, source, testAdmitted("provider_process_observation"))
		requireRefusal(t, err, "terminal_backend_capability_unproven", "operation capability conditional")
	})
}

// TestExecuteStopSuccess drives the request-stop row through the
// production entry: quiescing → stopped, the three effects committed in
// transition order and reported sorted, disposition replay_same.
func TestExecuteStopSuccess(t *testing.T) {
	backend := newMockBackend()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	context, params, source, admitted := stopFixture(t)
	result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationRequestStop, context, params, source, admitted)
	if err != nil {
		t.Fatalf("ExecuteMutating() error = %v", err)
	}
	requireLiteral(t, "after", string(result.After), "stopped")
	requireLiteral(t, "disposition", string(result.Disposition), "replay_same")
	performed := backend.performed()
	wantOrder := []terminalbackend.SideEffect{
		terminalbackend.EffectGracefulStopRequested,
		terminalbackend.EffectProcessClosed,
		terminalbackend.EffectBackendStoreClosed,
	}
	if len(performed) != 3 {
		t.Fatalf("performed = %v, want the three stop effects", performed)
	}
	for i := range wantOrder {
		requireLiteral(t, "performed", string(performed[i]), string(wantOrder[i]))
	}
	wantSorted := []terminalbackend.SideEffect{
		terminalbackend.EffectBackendStoreClosed,
		terminalbackend.EffectGracefulStopRequested,
		terminalbackend.EffectProcessClosed,
	}
	for i := range wantSorted {
		requireLiteral(t, "reported", string(result.Effects[i]), string(wantSorted[i]))
	}
	if err := CheckResult(context, source, result); err != nil {
		t.Errorf("CheckResult(engine result) error = %v", err)
	}

	backendBare := newMockBackend()
	engineBare := testEngine(t, backendBare, testLease(), fixtureGen)
	fresh, freshParams, freshSource, _ := stopFixture(t)
	fresh.OperationID = fixtureOperationB
	_, err = engineBare.ExecuteMutating(contextGo(), terminalbackend.OperationRequestStop, fresh, freshParams, freshSource, testAdmitted("durable_disconnect"))
	requireRefusal(t, err, "terminal_backend_capability_unproven", "operation capability dependency")
}

// TestExecuteIdenticalRetryReplays proves the idempotency window through
// the production entry: the identical retry returns the recorded result
// and performs no new effect.
func TestExecuteIdenticalRetryReplays(t *testing.T) {
	backend := newMockBackend()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	context, params, source, admitted := quiesceFixture(t)
	first, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
	if err != nil {
		t.Fatalf("ExecuteMutating() error = %v", err)
	}
	second, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
	if err != nil {
		t.Fatalf("ExecuteMutating() retry error = %v", err)
	}
	if len(backend.performed()) != 1 {
		t.Errorf("performed = %v, want exactly the first-run effect", backend.performed())
	}
	requireLiteral(t, "replay after", string(second.After), string(first.After))
	requireLiteral(t, "replay disposition", string(second.Disposition), string(first.Disposition))
	if len(second.Effects) != len(first.Effects) || len(second.EvidenceIDs) != len(first.EvidenceIDs) {
		t.Errorf("replay = %+v, want %+v", second, first)
	}
}

// TestExecuteChangedOperationInWindowMismatches proves a changed
// operation reusing an established key is idempotency_mismatch through
// the production entry: same key material, different operation ID.
func TestExecuteChangedOperationInWindowMismatches(t *testing.T) {
	backend := newMockBackend()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	context, params, source, admitted := quiesceFixture(t)
	if _, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted); err != nil {
		t.Fatalf("ExecuteMutating() error = %v", err)
	}
	changed := context
	changed.OperationID = fixtureOperationB
	result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, changed, params, source, admitted)
	requireRefusal(t, err, "idempotency_mismatch", "idempotency key conflict")
	requireLiteral(t, "after", string(result.After), "active")
	requireLiteral(t, "disposition", string(result.Disposition), "status_first")
	if len(backend.performed()) != 1 {
		t.Errorf("performed = %v, want no new effect on the mismatch", backend.performed())
	}
}

// TestExecuteUncertainRetryRequiresStatus proves a bound key without its
// completion refuses the identical retry with unavailable and
// status_first: the interrupted attempt proves nothing, so only status
// recovers the observation.
func TestExecuteUncertainRetryRequiresStatus(t *testing.T) {
	backend := newMockBackend()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	context, params, source, admitted := quiesceFixture(t)
	if _, _, err := engine.Store.Bind(context.IdempotencyKey, terminalbackend.OperationQuiesceInput, context.OperationID); err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
	requireRefusal(t, err, "terminal_backend_unavailable", "idempotency result uncertain")
	requireLiteral(t, "after", string(result.After), "unavailable")
	requireLiteral(t, "disposition", string(result.Disposition), "status_first")
	if len(backend.performed()) != 0 {
		t.Errorf("performed = %v, want no effects on the uncertain retry", backend.performed())
	}
}

// TestExecuteEmptyAdmittedRefusesAtLandedGate proves the engine wires
// the landed capability conferral: every mutating row with no admitted
// capability refuses through CheckOperation before any conditional,
// receipt or effect.
func TestExecuteEmptyAdmittedRefusesAtLandedGate(t *testing.T) {
	quiesceContext, quiesceParams, quiesceSource, _ := quiesceFixture(t)
	waitContext, waitParams, waitSource, _ := waitFixture(t, ProofAXCheckpointBoundary)
	stopContext, stopParams, stopSource, _ := stopFixture(t)
	create := testCreateContext(t)
	rows := []struct {
		operation terminalbackend.Operation
		context   MutationContext
		params    Params
		source    terminalbackend.InstanceState
	}{
		{terminalbackend.OperationQuiesceInput, quiesceContext, quiesceParams, quiesceSource},
		{terminalbackend.OperationWaitSafeBoundary, waitContext, waitParams, waitSource},
		{terminalbackend.OperationRequestStop, stopContext, stopParams, stopSource},
		{terminalbackend.OperationCreate, create.context, create.params, mustParseState(t, "absent")},
	}
	for _, row := range rows {
		t.Run(string(row.operation), func(t *testing.T) {
			backend := newMockBackend()
			engine := testEngine(t, backend, testLease(), fixtureGen)
			result, err := engine.ExecuteMutating(contextGo(), row.operation, row.context, row.params, row.source, testAdmitted())
			requireRefusal(t, err, "terminal_backend_capability_unproven", "operation capability dependency")
			requireLiteral(t, "after", string(result.After), string(row.source))
			if len(backend.performed()) != 0 {
				t.Errorf("performed = %v, want no effects", backend.performed())
			}
		})
	}
}

// TestExecuteScopeGate proves operations outside the engine rows are
// refused at the production entry with an empty result.
func TestExecuteScopeGate(t *testing.T) {
	backend := newMockBackend()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	context, params, source, admitted := quiesceFixture(t)
	for _, operation := range []terminalbackend.Operation{
		terminalbackend.OperationStatus, terminalbackend.OperationAttach,
		terminalbackend.OperationRestore, terminalbackend.OperationTerminateStale,
		terminalbackend.OperationManifest, terminalbackend.OperationProbe,
	} {
		result, err := engine.ExecuteMutating(contextGo(), operation, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_protocol_error", "operation lifecycle scope")
		if !reflect.DeepEqual(result, Result{}) {
			t.Errorf("ExecuteMutating(%s) result = %+v, want empty on the scope refusal", operation, result)
		}
	}
}

// TestExecuteOnlyQuiesceEntersQuiescing proves no operation other than
// quiesce-input enters quiescing: across every engine operation, every
// legal source and the error injections, after equals quiescing from a
// non-quiescing before only for quiesce-input.
func TestExecuteOnlyQuiesceEntersQuiescing(t *testing.T) {
	quiesce := func(t *testing.T) (MutationContext, Params, terminalbackend.Admitted) {
		context, params, _, admitted := quiesceFixture(t)
		return context, params, admitted
	}
	wait := func(t *testing.T) (MutationContext, Params, terminalbackend.Admitted) {
		context, params, _, admitted := waitFixture(t, ProofAXCheckpointBoundary)
		return context, params, admitted
	}
	stop := func(t *testing.T) (MutationContext, Params, terminalbackend.Admitted) {
		context, params, _, admitted := stopFixture(t)
		return context, params, admitted
	}
	create := func(t *testing.T) (MutationContext, Params, terminalbackend.Admitted) {
		fixture := testCreateContext(t)
		return fixture.context, fixture.params, testAdmitted("terminal_state_retention")
	}
	rows := []struct {
		operation terminalbackend.Operation
		sources   []string
		build     func(*testing.T) (MutationContext, Params, terminalbackend.Admitted)
	}{
		{terminalbackend.OperationQuiesceInput, []string{"active", "parked"}, quiesce},
		{terminalbackend.OperationWaitSafeBoundary, []string{"quiescing"}, wait},
		{terminalbackend.OperationRequestStop, []string{"quiescing"}, stop},
		{terminalbackend.OperationCreate, []string{"absent", "stopped"}, create},
	}
	for _, row := range rows {
		for _, source := range row.sources {
			t.Run(string(row.operation)+"/"+source, func(t *testing.T) {
				backend := newMockBackend()
				engine := testEngine(t, backend, testLease(), fixtureGen)
				context, params, admitted := row.build(t)
				result, err := engine.ExecuteMutating(contextGo(), row.operation, context, params, mustParseState(t, source), admitted)
				if err != nil {
					t.Fatalf("ExecuteMutating() error = %v", err)
				}
				if result.After == terminalbackend.StateQuiescing && result.Before != terminalbackend.StateQuiescing && row.operation != terminalbackend.OperationQuiesceInput {
					t.Errorf("ExecuteMutating(%s) enters quiescing from %q", row.operation, source)
				}
				if result.After == terminalbackend.StateCreating || result.After == terminalbackend.StateStaleFenced {
					t.Errorf("ExecuteMutating(%s) after = %q, want never creating or stale_fenced", row.operation, result.After)
				}
			})
		}
	}
	// The error paths never synthesize quiescing, creating or
	// stale_fenced either: after is the source or unavailable.
	backend := newMockBackend()
	backend.effectErrors[terminalbackend.EffectGracefulStopRequested] = errors.New("transport died")
	engine := testEngine(t, backend, testLease(), fixtureGen)
	context, params, source, admitted := stopFixture(t)
	result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationRequestStop, context, params, source, admitted)
	if err == nil {
		t.Fatal("ExecuteMutating() = nil, want the uncoded failure")
	}
	requireLiteral(t, "after", string(result.After), "unavailable")
}

// TestExecuteBackendRowErrorSets proves the exact admitted refusal sets
// per engine row through the production entry: every listed code passes
// through with its mapped disposition, and one code outside each row's
// set is rejected with the protocol class.
func TestExecuteBackendRowErrorSets(t *testing.T) {
	// quiesce-input admits terminal_backend_timeout and refuses
	// stop_timeout; wait-safe-boundary admits quiesce_timeout and
	// refuses terminal_backend_timeout; request-stop admits
	// stop_timeout and refuses quiesce_timeout. The admitted stop
	// timeout is driven twice: post-commit (unavailable) and on the
	// first effect (source restored, still status_first).
	backend := newMockBackend()
	backend.effectErrors[terminalbackend.EffectInputClosed] = refuse("terminal_backend_timeout", "test timeout")
	engine := testEngine(t, backend, testLease(), fixtureGen)
	context, params, source, admitted := quiesceFixture(t)
	result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
	requireRefusal(t, err, "terminal_backend_timeout", "backend effect refused")
	requireLiteral(t, "disposition", string(result.Disposition), "replay_same")

	backendWait := newMockBackend()
	backendWait.effectErrors[terminalbackend.EffectSafeBoundaryObserved] = refuse("quiesce_timeout", "test quiesce timeout")
	engineWait := testEngine(t, backendWait, testLease(), fixtureGen)
	waitContext, waitParams, waitSource, waitAdmitted := waitFixture(t, ProofAXCheckpointBoundary)
	waitResult, err := engineWait.ExecuteMutating(contextGo(), terminalbackend.OperationWaitSafeBoundary, waitContext, waitParams, waitSource, waitAdmitted)
	requireRefusal(t, err, "quiesce_timeout", "backend effect refused")
	requireLiteral(t, "wait after", string(waitResult.After), "quiescing")
	requireLiteral(t, "wait disposition", string(waitResult.Disposition), "replay_same")

	backendWaitOutside := newMockBackend()
	backendWaitOutside.effectErrors[terminalbackend.EffectSafeBoundaryObserved] = refuse("terminal_backend_timeout", "test outside set")
	engineWaitOutside := testEngine(t, backendWaitOutside, testLease(), fixtureGen)
	waitContext2, waitParams2, waitSource2, waitAdmitted2 := waitFixture(t, ProofAXCheckpointBoundary)
	waitContext2.OperationID = fixtureOperationB
	_, err = engineWaitOutside.ExecuteMutating(contextGo(), terminalbackend.OperationWaitSafeBoundary, waitContext2, waitParams2, waitSource2, waitAdmitted2)
	requireRefusal(t, err, "terminal_backend_protocol_error", "operation error vocabulary")

	backendStop := newMockBackend()
	backendStop.effectErrors[terminalbackend.EffectBackendStoreClosed] = refuse("stop_timeout", "test stop timeout")
	engineStop := testEngine(t, backendStop, testLease(), fixtureGen)
	stopContext, stopParams, stopSource, stopAdmitted := stopFixture(t)
	stopResult, err := engineStop.ExecuteMutating(contextGo(), terminalbackend.OperationRequestStop, stopContext, stopParams, stopSource, stopAdmitted)
	requireRefusal(t, err, "stop_timeout", "backend effect refused")
	requireLiteral(t, "stop after", string(stopResult.After), "unavailable")
	requireLiteral(t, "stop disposition", string(stopResult.Disposition), "status_first")

	backendStopOutside := newMockBackend()
	backendStopOutside.effectErrors[terminalbackend.EffectGracefulStopRequested] = refuse("quiesce_timeout", "test outside set")
	engineStopOutside := testEngine(t, backendStopOutside, testLease(), fixtureGen)
	stopContext2, stopParams2, stopSource2, stopAdmitted2 := stopFixture(t)
	stopContext2.OperationID = fixtureOperationB
	_, err = engineStopOutside.ExecuteMutating(contextGo(), terminalbackend.OperationRequestStop, stopContext2, stopParams2, stopSource2, stopAdmitted2)
	requireRefusal(t, err, "terminal_backend_protocol_error", "operation error vocabulary")

	// A backend-REPORTED stop_timeout on the first effect restores the
	// source but still routes through status: the graceful wait ran
	// and timed out, unlike the entry cancel that never attempted it.
	backendStopFirst := newMockBackend()
	backendStopFirst.effectErrors[terminalbackend.EffectGracefulStopRequested] = refuse("stop_timeout", "test graceful wait timed out")
	engineStopFirst := testEngine(t, backendStopFirst, testLease(), fixtureGen)
	stopContext3, stopParams3, stopSource3, stopAdmitted3 := stopFixture(t)
	stopFirst, err := engineStopFirst.ExecuteMutating(contextGo(), terminalbackend.OperationRequestStop, stopContext3, stopParams3, stopSource3, stopAdmitted3)
	requireRefusal(t, err, "stop_timeout", "backend effect refused")
	requireLiteral(t, "stop first-effect after", string(stopFirst.After), "quiescing")
	requireLiteral(t, "stop first-effect disposition", string(stopFirst.Disposition), "status_first")
}
