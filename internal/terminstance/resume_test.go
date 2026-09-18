package terminstance

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// countExportedReceipts counts the committed receipts in the store
// export: one key/operation/operation-ID triple per receipt.
func countExportedReceipts(t *testing.T, store *ReceiptStore) int {
	t.Helper()
	image, err := store.Export()
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	lines := 0
	for _, line := range strings.Split(string(image), "\n") {
		if line != "" {
			lines++
		}
	}
	if lines%3 != 0 {
		t.Fatalf("Export() = %d non-empty lines, want a multiple of 3", lines)
	}
	return lines / 3
}

// TestExecuteUncertainRetryResumesWhenStatusProvesSource proves the
// quiesce-input recovery column through the production entry: a bound
// key without its completion refuses uncertain while the observation
// is unknown, and the same-key retry continues under the same receipt
// once a successful status read proves the source state — exactly one
// effect, one completion and one receipt, never a second binding.
func TestExecuteUncertainRetryResumesWhenStatusProvesSource(t *testing.T) {
	backend := newMockBackend()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	context, params, source, admitted := quiesceFixture(t)
	if _, _, err := engine.Store.Bind(context.IdempotencyKey, terminalbackend.OperationQuiesceInput, context.OperationID); err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	// Without any status proof the identical retry refuses uncertain
	// instead of replaying or inventing absence.
	result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
	requireRefusal(t, err, "terminal_backend_unavailable", "idempotency result uncertain")
	requireLiteral(t, "after", string(result.After), "unavailable")
	requireLiteral(t, "disposition", string(result.Disposition), "status_first")
	if len(backend.performed()) != 0 {
		t.Fatalf("performed = %v, want no effects before the status proof", backend.performed())
	}
	// The caller reads status: the input was never closed.
	backend.observation = matchingObservation()
	report, err := engine.ExecuteStatus(contextGo(), exactStatusBody(t), source, testAdmitted("durable_disconnect"))
	if err != nil {
		t.Fatalf("ExecuteStatus() error = %v", err)
	}
	requireLiteral(t, "recovered state", string(report.State), "active")
	// The same-key retry after the proof continues the same operation.
	var interims []terminalbackend.InstanceState
	engine.Hooks = &EngineHooks{AfterReceipt: func(_ terminalbackend.Operation, _ string, interim terminalbackend.InstanceState) {
		interims = append(interims, interim)
	}}
	resumed, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
	if err != nil {
		t.Fatalf("ExecuteMutating() same-key retry after status error = %v", err)
	}
	requireLiteral(t, "before", string(resumed.Before), "active")
	requireLiteral(t, "after", string(resumed.After), "quiescing")
	requireLiteral(t, "disposition", string(resumed.Disposition), "replay_same")
	if len(resumed.Effects) != 1 || resumed.Effects[0] != terminalbackend.EffectInputClosed {
		t.Errorf("effects = %v, want [input_closed]", resumed.Effects)
	}
	performed := backend.performed()
	if len(performed) != 1 || performed[0] != terminalbackend.EffectInputClosed {
		t.Errorf("performed = %v, want exactly the one resumed effect", performed)
	}
	if len(interims) != 1 || interims[0] != terminalbackend.StateActive {
		t.Errorf("interims = %v, want [active] on the resumption", interims)
	}
	if _, found, err := engine.Store.Completed(context.IdempotencyKey); err != nil || !found {
		t.Errorf("Completed(key) = (%v, %v), want the one recorded result", found, err)
	}
	if got := countExportedReceipts(t, engine.Store); got != 1 {
		t.Errorf("exported receipts = %d, want exactly one (no second receipt)", got)
	}
}

// TestExecuteRequestStopResumesWhenNotClosed proves the request-stop
// recovery column ("lost result requires status, then same-key retry
// only if not closed"): status proving quiescing resumes the same stop
// under the same receipt with all three effects in transition order.
func TestExecuteRequestStopResumesWhenNotClosed(t *testing.T) {
	backend := newMockBackend()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	context, params, source, admitted := stopFixture(t)
	if _, _, err := engine.Store.Bind(context.IdempotencyKey, terminalbackend.OperationRequestStop, context.OperationID); err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	if _, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationRequestStop, context, params, source, admitted); err == nil {
		t.Fatal("ExecuteMutating() uncertain retry = nil, want the uncertain refusal")
	}
	observed := matchingObservation()
	observed.State = terminalbackend.StateQuiescing
	backend.observation = observed
	report, err := engine.ExecuteStatus(contextGo(), exactStatusBody(t), source, testAdmitted("durable_disconnect"))
	if err != nil {
		t.Fatalf("ExecuteStatus() error = %v", err)
	}
	requireLiteral(t, "recovered state", string(report.State), "quiescing")
	resumed, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationRequestStop, context, params, source, admitted)
	if err != nil {
		t.Fatalf("ExecuteMutating() same-key retry after status error = %v", err)
	}
	requireLiteral(t, "after", string(resumed.After), "stopped")
	requireLiteral(t, "disposition", string(resumed.Disposition), "replay_same")
	performed := backend.performed()
	want := []terminalbackend.SideEffect{terminalbackend.EffectGracefulStopRequested, terminalbackend.EffectProcessClosed, terminalbackend.EffectBackendStoreClosed}
	if len(performed) != len(want) {
		t.Fatalf("performed = %v, want the three stop effects once", performed)
	}
	for index := range want {
		if performed[index] != want[index] {
			t.Fatalf("performed = %v, want the transition order", performed)
		}
	}
	if _, found, err := engine.Store.Completed(context.IdempotencyKey); err != nil || !found {
		t.Errorf("Completed(key) = (%v, %v), want the one recorded result", found, err)
	}
	if got := countExportedReceipts(t, engine.Store); got != 1 {
		t.Errorf("exported receipts = %d, want exactly one (no second receipt)", got)
	}
}

// TestExecuteCreateResumesWhenStatusProvesAbsent proves create is
// idempotent on (session_id, bootstrap_operation_id) across controller
// crash and lost result: only a successful status read proves absence,
// and the identical retry then performs the effects under the same
// receipt and completes it — exactly one result and one instance.
func TestExecuteCreateResumesWhenStatusProvesAbsent(t *testing.T) {
	create := testCreateContext(t)
	backend := newMockBackend()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	source := mustParseState(t, "absent")
	admitted := testAdmitted("terminal_state_retention")
	if _, _, err := engine.Store.Bind(create.context.IdempotencyKey, terminalbackend.OperationCreate, create.context.OperationID); err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	if _, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationCreate, create.context, create.params, source, admitted); err == nil {
		t.Fatal("ExecuteMutating() uncertain retry = nil, want the uncertain refusal")
	}
	// Only a successful status read proves absence: the canonical
	// absent form with a drifted generation is a proven non-match.
	backend.observation = StatusObservation{
		State: terminalbackend.StateAbsent, IdentityMatch: false,
		SessionID: fixtureSessionA, InstanceID: fixtureInstance,
		BackendID: fixtureBackend, ImplVersion: fixtureImpl,
		ProtoVersion: fixtureProto, Generation: "generation-two",
	}
	report, err := engine.ExecuteStatus(contextGo(), exactStatusBody(t), source, testAdmitted("durable_disconnect"))
	if err != nil {
		t.Fatalf("ExecuteStatus() error = %v", err)
	}
	requireLiteral(t, "recovered state", string(report.State), "absent")
	var interims []terminalbackend.InstanceState
	engine.Hooks = &EngineHooks{AfterReceipt: func(_ terminalbackend.Operation, _ string, interim terminalbackend.InstanceState) {
		interims = append(interims, interim)
	}}
	resumed, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationCreate, create.context, create.params, source, admitted)
	if err != nil {
		t.Fatalf("ExecuteMutating() identical create retry after status error = %v", err)
	}
	requireLiteral(t, "after", string(resumed.After), "active")
	requireLiteral(t, "disposition", string(resumed.Disposition), "replay_same")
	performed := backend.performed()
	if len(performed) != 2 || performed[0] != terminalbackend.EffectBindingPersisted || performed[1] != terminalbackend.EffectWrapperStarted {
		t.Errorf("performed = %v, want exactly one instance's [binding_persisted wrapper_started]", performed)
	}
	if len(interims) != 1 || interims[0] != terminalbackend.StateCreating {
		t.Errorf("interims = %v, want [creating] on the create resumption", interims)
	}
	if _, found, err := engine.Store.Completed(create.context.IdempotencyKey); err != nil || !found {
		t.Errorf("Completed(key) = (%v, %v), want the one recorded result", found, err)
	}
	if got := countExportedReceipts(t, engine.Store); got != 1 {
		t.Errorf("exported receipts = %d, want exactly one (no second receipt)", got)
	}
}

// TestExecuteUncertainRetryTargetProvenRefuses proves a same-key retry
// never re-issues when status proves anything but the presented source:
// the quiesce target (the input may already be closed), the closed stop
// ("only if not closed"), and a contradictory observation all refuse
// uncertain with no new effect, because the lost evidence cannot be
// proven and re-issuing from a non-source state would violate the
// transition matrix.
func TestExecuteUncertainRetryTargetProvenRefuses(t *testing.T) {
	t.Run("quiesce target proven", func(t *testing.T) {
		backend := newMockBackend()
		engine := testEngine(t, backend, testLease(), fixtureGen)
		context, params, source, admitted := quiesceFixture(t)
		if _, _, err := engine.Store.Bind(context.IdempotencyKey, terminalbackend.OperationQuiesceInput, context.OperationID); err != nil {
			t.Fatalf("Bind() error = %v", err)
		}
		observed := matchingObservation()
		observed.State = terminalbackend.StateQuiescing
		backend.observation = observed
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_unavailable", "idempotency result uncertain")
		requireLiteral(t, "after", string(result.After), "unavailable")
		requireLiteral(t, "disposition", string(result.Disposition), "status_first")
		if len(backend.performed()) != 0 {
			t.Errorf("performed = %v, want no re-issue onto the proven target", backend.performed())
		}
	})
	t.Run("stop closed", func(t *testing.T) {
		backend := newMockBackend()
		engine := testEngine(t, backend, testLease(), fixtureGen)
		context, params, source, admitted := stopFixture(t)
		if _, _, err := engine.Store.Bind(context.IdempotencyKey, terminalbackend.OperationRequestStop, context.OperationID); err != nil {
			t.Fatalf("Bind() error = %v", err)
		}
		observed := matchingObservation()
		observed.State = terminalbackend.StateStopped
		backend.observation = observed
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationRequestStop, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_unavailable", "idempotency result uncertain")
		requireLiteral(t, "after", string(result.After), "unavailable")
		if len(backend.performed()) != 0 {
			t.Errorf("performed = %v, want no same-key retry once closed", backend.performed())
		}
	})
	t.Run("contradictory observation", func(t *testing.T) {
		backend := newMockBackend()
		engine := testEngine(t, backend, testLease(), fixtureGen)
		context, params, source, admitted := quiesceFixture(t)
		if _, _, err := engine.Store.Bind(context.IdempotencyKey, terminalbackend.OperationQuiesceInput, context.OperationID); err != nil {
			t.Fatalf("Bind() error = %v", err)
		}
		observed := matchingObservation()
		observed.State = terminalbackend.StateParked
		backend.observation = observed
		_, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_unavailable", "idempotency result uncertain")
		if len(backend.performed()) != 0 {
			t.Errorf("performed = %v, want no re-issue onto a contradiction", backend.performed())
		}
	})
}

// TestExecuteUncertainRetryUnknownObservationRefuses proves the retry
// refuses while the status reconciliation itself proves nothing: a
// coded read failure, an uncoded read failure, and a lying backend
// report (a claimed match on a drifted instance) each refuse uncertain
// with no new effect. Only a validated proof of the source resumes.
func TestExecuteUncertainRetryUnknownObservationRefuses(t *testing.T) {
	t.Run("coded read failure", func(t *testing.T) {
		backend := newMockBackend()
		backend.observeErr = refuse("terminal_backend_timeout", "test timeout")
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
			t.Errorf("performed = %v, want no effects on the failed reconciliation", backend.performed())
		}
	})
	t.Run("uncoded read failure", func(t *testing.T) {
		backend := newMockBackend()
		backend.observeErr = errors.New("read failed")
		engine := testEngine(t, backend, testLease(), fixtureGen)
		context, params, source, admitted := quiesceFixture(t)
		if _, _, err := engine.Store.Bind(context.IdempotencyKey, terminalbackend.OperationQuiesceInput, context.OperationID); err != nil {
			t.Fatalf("Bind() error = %v", err)
		}
		_, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_unavailable", "idempotency result uncertain")
		if len(backend.performed()) != 0 {
			t.Errorf("performed = %v, want no effects on the failed reconciliation", backend.performed())
		}
	})
	t.Run("lying match on drifted instance", func(t *testing.T) {
		backend := newMockBackend()
		engine := testEngine(t, backend, testLease(), fixtureGen)
		context, params, source, admitted := quiesceFixture(t)
		if _, _, err := engine.Store.Bind(context.IdempotencyKey, terminalbackend.OperationQuiesceInput, context.OperationID); err != nil {
			t.Fatalf("Bind() error = %v", err)
		}
		observed := matchingObservation()
		observed.InstanceID = "0198f4c8-8e50-7f66-8f70-bbbbbbbbbbb2"
		backend.observation = observed
		_, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_unavailable", "idempotency result uncertain")
		if len(backend.performed()) != 0 {
			t.Errorf("performed = %v, want no resumption on a lying report", backend.performed())
		}
	})
}

// TestExecuteHookErrorRetryReconcilesThroughStatus proves the reworded
// AfterCommit contract at the engine: the hook error propagates and the
// receipt stands with no completion and no effects, and the identical
// retry then reconciles — it resumes when status proves the source and
// refuses uncertain while the observation is unknown.
func TestExecuteHookErrorRetryReconcilesThroughStatus(t *testing.T) {
	newHooked := func(t *testing.T) (*Engine, *mockBackend, MutationContext, Params, terminalbackend.InstanceState, terminalbackend.Admitted) {
		backend := newMockBackend()
		engine := testEngine(t, backend, testLease(), fixtureGen)
		engine.Store.WithHooks(&StoreHooks{AfterCommit: func(string) error { return errTestHook }})
		context, params, source, admitted := quiesceFixture(t)
		return engine, backend, context, params, source, admitted
	}
	t.Run("standing receipt is uncertain", func(t *testing.T) {
		engine, backend, context, params, source, admitted := newHooked(t)
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_process_failed", "idempotency store failure")
		requireLiteral(t, "after", string(result.After), "unavailable")
		requireLiteral(t, "disposition", string(result.Disposition), "status_first")
		if len(backend.performed()) != 0 {
			t.Errorf("performed = %v, want no effects on the hook failure", backend.performed())
		}
		if _, found, err := engine.Store.Lookup(context.IdempotencyKey); err != nil || !found {
			t.Errorf("Lookup(key) = (%v, %v), want the standing receipt", found, err)
		}
		if _, found, err := engine.Store.Completed(context.IdempotencyKey); err != nil || found {
			t.Errorf("Completed(key) = (%v, %v), want no completion", found, err)
		}
	})
	t.Run("retry resumes on proven source", func(t *testing.T) {
		engine, backend, context, params, source, admitted := newHooked(t)
		if _, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted); err == nil {
			t.Fatal("ExecuteMutating() hook failure = nil, want the hook error")
		}
		engine.Store.WithHooks(nil)
		backend.observation = matchingObservation()
		resumed, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
		if err != nil {
			t.Fatalf("ExecuteMutating() retry after hook failure error = %v", err)
		}
		requireLiteral(t, "after", string(resumed.After), "quiescing")
		if len(backend.performed()) != 1 {
			t.Errorf("performed = %v, want exactly the one resumed effect", backend.performed())
		}
		if _, found, err := engine.Store.Completed(context.IdempotencyKey); err != nil || !found {
			t.Errorf("Completed(key) = (%v, %v), want the one recorded result", found, err)
		}
		if got := countExportedReceipts(t, engine.Store); got != 1 {
			t.Errorf("exported receipts = %d, want exactly one (no second receipt)", got)
		}
	})
	t.Run("retry refuses on unknown observation", func(t *testing.T) {
		engine, backend, context, params, source, admitted := newHooked(t)
		if _, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted); err == nil {
			t.Fatal("ExecuteMutating() hook failure = nil, want the hook error")
		}
		engine.Store.WithHooks(nil)
		_, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_unavailable", "idempotency result uncertain")
		if len(backend.performed()) != 0 {
			t.Errorf("performed = %v, want no effects without a status proof", backend.performed())
		}
	})
}

// TestExecutePreLinkStoreFailureRestoresSource proves an idempotency
// store failure before any link restores the source state: nothing is
// durable and no side effect ran, so the identical retry is safe and
// reports the class-mapped disposition.
func TestExecutePreLinkStoreFailureRestoresSource(t *testing.T) {
	backend := newMockBackend()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	pending := filepath.Join(engine.Store.base, "pending")
	if err := os.Chmod(pending, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(pending, 0o755) })
	context, params, source, admitted := quiesceFixture(t)
	result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
	requireRefusal(t, err, "terminal_backend_process_failed", "idempotency store failure")
	requireLiteral(t, "after", string(result.After), "active")
	requireLiteral(t, "disposition", string(result.Disposition), "required_operator_action")
	if len(backend.performed()) != 0 {
		t.Errorf("performed = %v, want no effects on the pre-link failure", backend.performed())
	}
	_ = os.Chmod(pending, 0o755)
	if _, found, _ := engine.Store.Lookup(context.IdempotencyKey); found {
		t.Error("Lookup(key) found a receipt despite the read-only directory, want none bound")
	}
}

// TestExecuteStoreLookupFailureIsUncertain proves an unreadable receipt
// store is genuinely uncertain: when the engine cannot decide what is
// durable it moves to unavailable with status_first instead of
// restoring a source it cannot prove.
func TestExecuteStoreLookupFailureIsUncertain(t *testing.T) {
	backend := newMockBackend()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	context, params, source, admitted := quiesceFixture(t)
	// A directory at the receipt path makes every read of the key fail:
	// Bind cannot install or replay, and the engine Lookup cannot
	// decide durability.
	if err := os.Mkdir(filepath.Join(engine.Store.base, "pending", receiptName(context.IdempotencyKey)), 0o755); err != nil {
		t.Fatal(err)
	}
	result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
	requireRefusal(t, err, "terminal_backend_process_failed", "idempotency store failure")
	requireLiteral(t, "after", string(result.After), "unavailable")
	requireLiteral(t, "disposition", string(result.Disposition), "status_first")
	if len(backend.performed()) != 0 {
		t.Errorf("performed = %v, want no effects on the unreadable store", backend.performed())
	}
}

// TestKindForAgreesWithLandedTable proves kindFor delegates to the
// landed transition-table authorization column instead of restating it:
// every engine operation honors the kind the table names, and every
// other operation (plus unknown names) honors none. A drift in either
// direction fails this test.
func TestKindForAgreesWithLandedTable(t *testing.T) {
	for _, row := range []struct {
		operation terminalbackend.Operation
		kind      string
		ok        bool
	}{
		{terminalbackend.OperationCreate, "create", true},
		{terminalbackend.OperationQuiesceInput, "control", true},
		{terminalbackend.OperationWaitSafeBoundary, "control", true},
		{terminalbackend.OperationRequestStop, "control", true},
		{terminalbackend.OperationAttach, "", false},
		{terminalbackend.OperationStatus, "", false},
		{terminalbackend.OperationManifest, "", false},
		{terminalbackend.OperationProbe, "", false},
		{terminalbackend.OperationTerminateStale, "", false},
		{terminalbackend.OperationRestore, "", false},
	} {
		kind, ok := kindFor(row.operation)
		if ok != row.ok || string(kind) != row.kind {
			t.Errorf("kindFor(%s) = (%q, %v), want (%q, %v)", row.operation, kind, ok, row.kind, row.ok)
		}
		landed, err := terminalbackend.TransitionAuthorization(string(row.operation))
		if err != nil {
			t.Errorf("TransitionAuthorization(%s) error = %v", row.operation, err)
			continue
		}
		if row.ok && landed != row.kind {
			t.Errorf("TransitionAuthorization(%s) = %q, want the honored %q", row.operation, landed, row.kind)
		}
		if !row.ok && (landed == "create" || landed == "control") {
			t.Errorf("kindFor(%s) refuses but the table names the honored kind %q", row.operation, landed)
		}
	}
	if _, ok := kindFor("launch"); ok {
		t.Error("kindFor(launch) = true, want refusal for the unknown operation")
	}
}
