package terminstance

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// Story-close witnesses for the independent review's residual P4
// evidence items (rev4 verdict P3-1 through P3-4): the expiry axis at
// the third effect position, the replayed-image generation binding and
// corrupt-image integrity failure at the engine replay site, the
// in-loop deadline instant, and the mixed-case enum siblings. The
// bodies are the reviewer's committable witnesses verbatim; the mutant
// harness mirrors the reviewer's plants as committed rows.

// TestRV4W_ExpiryRecheckedBeforeThirdEffect (WITNESS for RV4-M1): the
// authorization expires between the second and third stop effect while
// the tuple is unchanged and the deadline is still ahead. The recheck
// must refuse before the third effect with the literal expiry arm,
// after=unavailable, new_authorization and exactly two committed effects.
func TestRV4W_ExpiryRecheckedBeforeThirdEffect(t *testing.T) {
	backend := newMockBackend()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	context, params := rv3StopContextWithDeadline(t, "2026-09-03T00:00:00.000Z")
	afterExpiry, err := time.Parse(time.RFC3339Nano, "2026-09-02T01:00:00.000Z")
	if err != nil {
		t.Fatal(err)
	}
	effects := 0
	engine.Hooks = &EngineHooks{AfterEffect: func(terminalbackend.SideEffect) { effects++ }}
	engine.Now = func() time.Time {
		if effects >= 2 {
			return afterExpiry
		}
		return fixtureNow()
	}
	result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationRequestStop, context, params, mustParseState(t, "quiescing"), testAdmitted("graceful_stop"))
	requireRefusal(t, err, "terminal_backend_unauthorized", "ax authorization expiry")
	requireLiteral(t, "after", string(result.After), "unavailable")
	requireLiteral(t, "disposition", string(result.Disposition), "new_authorization")
	performed := backend.performed()
	if len(performed) != 2 || performed[0] != terminalbackend.EffectGracefulStopRequested || performed[1] != terminalbackend.EffectProcessClosed {
		t.Errorf("performed = %v, want exactly [graceful_stop_requested process_closed] before the expired authorization is caught", performed)
	}
	if _, found, err := engine.Store.Completed(context.IdempotencyKey); err != nil || found {
		t.Errorf("Completed(key) = (%v, %v), want no completion after the mid-operation expiry", found, err)
	}
}

// TestRV4W_ReplayedImageGenerationBindingAtEngine (WITNESS for RV4-M6 and
// RV4-M6b): a completion recorded under generation-one replayed to an
// identical request whose validated binding generation is now
// generation-two must refuse with the landed stale-generation class
// (`terminal_backend_stale_generation` / `result generation binding`),
// after=unavailable and status_first, never hand back the stale image
// as a successful replay.
func TestRV4W_ReplayedImageGenerationBindingAtEngine(t *testing.T) {
	backend := newMockBackend()
	backend.evidence[terminalbackend.EffectInputClosed] = fixtureDigestB
	engine := testEngine(t, backend, testLease(), fixtureGen)
	context, params, source, admitted := quiesceFixture(t)
	if _, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted); err != nil {
		t.Fatalf("first quiesce error = %v", err)
	}
	// Same key, same operation ID, but the binding generation moved.
	key := fixtureInstance + "/quiesce/" + fixtureQuiescence
	raw := contextDoc(fixtureOperation, fixtureSessionA, fixtureInstance, fixtureBackend, fixtureImpl, fixtureProto, "generation-two", key, fixtureDeadline, authDoc("1", nil))
	moved, err := ParseMutationContext([]byte(raw))
	if err != nil {
		t.Fatalf("ParseMutationContext(generation-two) error = %v", err)
	}
	engine.CurrentGeneration = func() string { return "generation-two" }
	result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, moved, params, source, admitted)
	requireRefusal(t, err, "terminal_backend_stale_generation", "result generation binding")
	requireLiteral(t, "after", string(result.After), "unavailable")
	requireLiteral(t, "disposition", string(result.Disposition), "status_first")
	if performed := backend.performed(); len(performed) != 1 {
		t.Errorf("performed = %v, want no second effect on the refused replay", performed)
	}
}

// TestRV4_CorruptCompletionImageIsIntegrityFailure characterizes the
// unmarshal arm of the replay seam: a completion whose stored bytes are
// not a Result image refuses terminal_backend_integrity_failure /
// idempotency result image with after=unavailable and status_first.
func TestRV4_CorruptCompletionImageIsIntegrityFailure(t *testing.T) {
	backend := newMockBackend()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	context, params, source, admitted := quiesceFixture(t)
	if _, _, err := engine.Store.Bind(context.IdempotencyKey, terminalbackend.OperationQuiesceInput, context.OperationID); err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	if err := engine.Store.Complete(context.IdempotencyKey, []byte("{not json")); err != nil {
		t.Fatalf("Complete(corrupt) error = %v", err)
	}
	result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
	requireRefusal(t, err, "terminal_backend_integrity_failure", "idempotency result image")
	requireLiteral(t, "after", string(result.After), "unavailable")
	requireLiteral(t, "disposition", string(result.Disposition), "status_first")
	if len(backend.performed()) != 0 {
		t.Errorf("performed = %v, want no effect on a corrupt image", backend.performed())
	}
}

// TestRV4W_LoopDeadlineInstantCancelsWaiting (WITNESS for RV4-M7): the
// deadline instant itself, reached exactly between the first and second
// stop effect, cancels the remaining waits (the spec's "a deadline
// cancels waiting") with stop_timeout, after=unavailable (one effect is
// committed), status_first and exactly one committed effect.
func TestRV4W_LoopDeadlineInstantCancelsWaiting(t *testing.T) {
	backend := newMockBackend()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	context, params, source, admitted := stopFixture(t)
	deadline, err := context.Deadline.Time()
	if err != nil {
		t.Fatal(err)
	}
	effects := 0
	engine.Hooks = &EngineHooks{AfterEffect: func(terminalbackend.SideEffect) { effects++ }}
	engine.Now = func() time.Time {
		if effects >= 1 {
			return deadline
		}
		return fixtureNow()
	}
	result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationRequestStop, context, params, source, admitted)
	requireRefusal(t, err, "stop_timeout", "operation deadline")
	requireLiteral(t, "after", string(result.After), "unavailable")
	requireLiteral(t, "disposition", string(result.Disposition), "status_first")
	if performed := backend.performed(); len(performed) != 1 {
		t.Errorf("performed = %v, want exactly one committed effect at the deadline instant", performed)
	}
}

// TestRV4W_AuthorizationKindCaseSiblingRefused (WITNESS for RV4-M5): the
// closed authorization_kind enum refuses case-variant and whitespace
// spellings of its members at the direct entry and through the document
// entry (nested AXAuthorization).
func TestRV4W_AuthorizationKindCaseSiblingRefused(t *testing.T) {
	for _, bad := range []string{"Control", "Create", "CONTROL", " control", "control ", "Force_Stale", "Restore"} {
		_, err := ParseAuthorizationKind(bad)
		requireRefusal(t, err, "terminal_backend_protocol_error", "ax authorization kind")
		encoded, _ := json.Marshal(bad)
		raw := authDoc("1", map[string]*string{"authorization_kind": strptr(string(encoded))})
		_, err = ParseAXAuthorization([]byte(raw))
		requireRefusal(t, err, "terminal_backend_protocol_error", "ax authorization kind")
	}
	// The same class on the two sibling enums.
	for _, bad := range []string{"Replay_Same", "Status_First", "NEW_AUTHORIZATION", " required_operator_action"} {
		_, err := ParseRetryDisposition(bad)
		requireRefusal(t, err, "terminal_backend_protocol_error", "retry disposition vocabulary")
	}
	for _, bad := range []string{"Provider_Quiescence", "Ax_Checkpoint_Boundary", "provider_process_exit "} {
		_, err := ParseProviderProofKind(bad)
		requireRefusal(t, err, "terminal_backend_protocol_error", "provider proof vocabulary")
	}
}
