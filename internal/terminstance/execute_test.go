package terminstance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

func quiesceFixture(t *testing.T) (MutationContext, Params, terminalbackend.InstanceState, terminalbackend.Admitted) {
	t.Helper()
	return testContext(t), testParams(), mustParseState(t, "active"), testAdmitted("input_quiescence")
}

// TestExecuteQuiesceSuccess drives the quiesce-input row through the
// production entry from both sources: active|parked → quiescing with
// exactly input_closed, disposition replay_same, and the receipt bound.
func TestExecuteQuiesceSuccess(t *testing.T) {
	for _, source := range []string{"active", "parked"} {
		t.Run(source, func(t *testing.T) {
			backend := newMockBackend()
			backend.evidence[terminalbackend.EffectInputClosed] = fixtureDigestB
			engine := testEngine(t, backend, testLease(), fixtureGen)
			context, params, _, admitted := quiesceFixture(t)
			result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, mustParseState(t, source), admitted)
			if err != nil {
				t.Fatalf("ExecuteMutating() error = %v", err)
			}
			requireLiteral(t, "before", string(result.Before), source)
			requireLiteral(t, "after", string(result.After), "quiescing")
			requireLiteral(t, "disposition", string(result.Disposition), "replay_same")
			if len(result.Effects) != 1 || result.Effects[0] != terminalbackend.EffectInputClosed {
				t.Errorf("effects = %v, want [input_closed]", result.Effects)
			}
			requireLiteral(t, "evidence", result.EvidenceIDs[0], "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
			if err := CheckResult(context, mustParseState(t, source), result); err != nil {
				t.Errorf("CheckResult(engine result) error = %v", err)
			}
			performed := backend.performed()
			if len(performed) != 1 || performed[0] != terminalbackend.EffectInputClosed {
				t.Errorf("performed = %v, want [input_closed]", performed)
			}
			if _, found, err := engine.Store.Lookup(context.IdempotencyKey); err != nil || !found {
				t.Errorf("Lookup(key) = (%v, %v), want the bound receipt", found, err)
			}
		})
	}
}

func contextGo() context.Context { return context.Background() }

// TestExecuteReceiptPrecedesFirstEffect proves creating is entered only
// after its durable receipt: the backend's first effect observes the
// committed receipt through the production entry.
func TestExecuteReceiptPrecedesFirstEffect(t *testing.T) {
	backend := newMockBackend()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	context, params, source, admitted := quiesceFixture(t)
	backend.effectHandler = func(effect terminalbackend.SideEffect) (string, error) {
		if _, found, err := engine.Store.Lookup(context.IdempotencyKey); err != nil || !found {
			t.Errorf("Lookup(key) inside first effect = (%v, %v), want the committed receipt", found, err)
		}
		if _, found, err := engine.Store.Completed(context.IdempotencyKey); err != nil || found {
			t.Errorf("Completed(key) inside first effect = (%v, %v), want no completion yet", found, err)
		}
		return fixtureDigestA, nil
	}
	if _, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted); err != nil {
		t.Fatalf("ExecuteMutating() error = %v", err)
	}
}

// TestExecuteInterimState proves creating is entered only by create:
// the AfterReceipt hook observes creating for create and the source
// for every other engine operation.
func TestExecuteInterimState(t *testing.T) {
	requireLiteral(t, "create interim", string(InterimState(terminalbackend.OperationCreate, mustParseState(t, "absent"))), "creating")
	for _, operation := range []terminalbackend.Operation{
		terminalbackend.OperationQuiesceInput, terminalbackend.OperationWaitSafeBoundary, terminalbackend.OperationRequestStop,
	} {
		if got := InterimState(operation, mustParseState(t, "active")); got != terminalbackend.StateActive {
			t.Errorf("InterimState(%s) = %q, want the source", operation, got)
		}
	}

	backend := newMockBackend()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	var interims []terminalbackend.InstanceState
	engine.Hooks = &EngineHooks{AfterReceipt: func(_ terminalbackend.Operation, _ string, interim terminalbackend.InstanceState) {
		interims = append(interims, interim)
	}}
	context, params, source, admitted := quiesceFixture(t)
	if _, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted); err != nil {
		t.Fatalf("ExecuteMutating() error = %v", err)
	}
	if len(interims) != 1 || interims[0] != terminalbackend.StateActive {
		t.Errorf("interims = %v, want [active]", interims)
	}

	create := testCreateContext(t)
	backendCreate := newMockBackend()
	engineCreate := testEngine(t, backendCreate, testLease(), fixtureGen)
	var createInterims []terminalbackend.InstanceState
	engineCreate.Hooks = &EngineHooks{AfterReceipt: func(_ terminalbackend.Operation, _ string, interim terminalbackend.InstanceState) {
		createInterims = append(createInterims, interim)
	}}
	if _, err := engineCreate.ExecuteMutating(contextGo(), terminalbackend.OperationCreate, create.context, create.params, mustParseState(t, "absent"), testAdmitted("terminal_state_retention")); err != nil {
		t.Fatalf("ExecuteMutating(create) error = %v", err)
	}
	if len(createInterims) != 1 || createInterims[0] != terminalbackend.StateCreating {
		t.Errorf("create interims = %v, want [creating]", createInterims)
	}
}

type createFixture struct {
	context MutationContext
	params  Params
}

func testCreateContext(t *testing.T) createFixture {
	t.Helper()
	return testCreateContextWithDeadline(t, fixtureDeadline)
}

// testCreateContextWithDeadline builds the parsed create fixture with
// an explicit deadline through the production ParseMutationContext
// entry.
func testCreateContextWithDeadline(t *testing.T, deadline string) createFixture {
	t.Helper()
	auth, err := ParseAXAuthorization([]byte(authDoc("1", map[string]*string{"authorization_kind": strptr(`"create"`)})))
	if err != nil {
		t.Fatalf("ParseAXAuthorization(create) error = %v", err)
	}
	key := fixtureSessionA + "/" + fixtureBootstrap
	raw := contextDoc(fixtureOperation, fixtureSessionA, fixtureInstance, fixtureBackend, fixtureImpl, fixtureProto, fixtureGen, key, deadline, authDoc("1", map[string]*string{"authorization_kind": strptr(`"create"`)}))
	context, err := ParseMutationContext([]byte(raw))
	if err != nil {
		t.Fatalf("ParseMutationContext(create) error = %v", err)
	}
	_ = auth
	return createFixture{context: context, params: Params{BootstrapOperationID: fixtureBootstrap, Interactive: true}}
}

// TestExecuteCreateReceiptRule drives the create receipt rule: absent →
// active when interactive with binding_persisted and wrapper_started,
// stopped → parked when headless, and headless without
// headless_creation refuses with the capability class.
func TestExecuteCreateReceiptRule(t *testing.T) {
	create := testCreateContext(t)
	backend := newMockBackend()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationCreate, create.context, create.params, mustParseState(t, "absent"), testAdmitted("terminal_state_retention"))
	if err != nil {
		t.Fatalf("ExecuteMutating(create interactive) error = %v", err)
	}
	requireLiteral(t, "after", string(result.After), "active")
	requireLiteral(t, "disposition", string(result.Disposition), "replay_same")
	performed := backend.performed()
	if len(performed) != 2 || performed[0] != terminalbackend.EffectBindingPersisted || performed[1] != terminalbackend.EffectWrapperStarted {
		t.Errorf("performed = %v, want [binding_persisted wrapper_started]", performed)
	}

	headless := testCreateContext(t)
	headless.params.Interactive = false
	backendHeadless := newMockBackend()
	engineHeadless := testEngine(t, backendHeadless, testLease(), fixtureGen)
	result, err = engineHeadless.ExecuteMutating(contextGo(), terminalbackend.OperationCreate, headless.context, headless.params, mustParseState(t, "stopped"), testAdmitted("headless_creation"))
	if err != nil {
		t.Fatalf("ExecuteMutating(create headless) error = %v", err)
	}
	requireLiteral(t, "after", string(result.After), "parked")

	backendBare := newMockBackend()
	engineBare := testEngine(t, backendBare, testLease(), fixtureGen)
	fresh := testCreateContext(t)
	fresh.context.OperationID = fixtureOperationB
	fresh.params.Interactive = false
	_, err = engineBare.ExecuteMutating(contextGo(), terminalbackend.OperationCreate, fresh.context, fresh.params, mustParseState(t, "stopped"), testAdmitted("terminal_state_retention"))
	requireRefusal(t, err, "terminal_backend_capability_unproven", "operation capability conditional")
	if len(backendBare.performed()) != 0 {
		t.Errorf("performed = %v, want no effects on the refused create", backendBare.performed())
	}
}

// TestExecutePreEffectErrorsRestoreSource proves an error before the
// first side effect restores the source state: entry auth failure,
// entry generation staleness, entry deadline expiry and a coded
// backend refusal on the single quiesce effect each return after equal
// to the source with the class-mapped disposition.
func TestExecutePreEffectErrorsRestoreSource(t *testing.T) {
	t.Run("entry auth expired", func(t *testing.T) {
		// The deadline lies past the expiry and Now sits between
		// them: under an entry-exempting mutant the deadline passes,
		// the receipt binds, and only the receipt assertion below
		// distinguishes the mutant. This member kills on the receipt.
		backend := newMockBackend()
		engine := testEngine(t, backend, testLease(), fixtureGen)
		engine.Now = fixturePastExpiry
		context := testContextWithDeadline(t, fixtureLateDeadline)
		params, source, admitted := testParams(), mustParseState(t, "active"), testAdmitted("input_quiescence")
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_unauthorized", "ax authorization expiry")
		requireLiteral(t, "after", string(result.After), "active")
		requireLiteral(t, "disposition", string(result.Disposition), "new_authorization")
		if _, found, _ := engine.Store.Lookup(context.IdempotencyKey); found {
			t.Error("Lookup(key) found a receipt for the unauthorized request, want none bound")
		}
	})

	t.Run("entry generation stale", func(t *testing.T) {
		backend := newMockBackend()
		engine := testEngine(t, backend, testLease(), "generation-two")
		context, params, source, admitted := quiesceFixture(t)
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_stale_generation", "backend_generation stale")
		requireLiteral(t, "after", string(result.After), "active")
		requireLiteral(t, "disposition", string(result.Disposition), "status_first")
		if _, found, _ := engine.Store.Lookup(context.IdempotencyKey); found {
			t.Error("Lookup(key) found a receipt for the stale request, want none bound")
		}
	})

	t.Run("coded backend refusal on first effect", func(t *testing.T) {
		backend := newMockBackend()
		backend.effectErrors[terminalbackend.EffectInputClosed] = refuse("terminal_backend_timeout", "test timeout")
		engine := testEngine(t, backend, testLease(), fixtureGen)
		context, params, source, admitted := quiesceFixture(t)
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_timeout", "backend effect refused")
		requireLiteral(t, "after", string(result.After), "active")
		requireLiteral(t, "disposition", string(result.Disposition), "replay_same")
	})

	t.Run("illegal source refuses", func(t *testing.T) {
		backend := newMockBackend()
		engine := testEngine(t, backend, testLease(), fixtureGen)
		context, params, _, admitted := quiesceFixture(t)
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, mustParseState(t, "stopped"), admitted)
		requireRefusal(t, err, "local_precondition_failed", "lifecycle transition")
		requireLiteral(t, "after", string(result.After), "stopped")
		requireLiteral(t, "disposition", string(result.Disposition), "status_first")
		if len(backend.performed()) != 0 {
			t.Errorf("performed = %v, want no effects on the illegal source", backend.performed())
		}
	})

	t.Run("missing conferral refuses at the landed gate", func(t *testing.T) {
		backend := newMockBackend()
		engine := testEngine(t, backend, testLease(), fixtureGen)
		context, params, source, _ := quiesceFixture(t)
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, testAdmitted("durable_disconnect"))
		requireRefusal(t, err, "terminal_backend_capability_unproven", "operation capability dependency")
		requireLiteral(t, "after", string(result.After), "active")
		requireLiteral(t, "disposition", string(result.Disposition), "required_operator_action")
	})
}

// TestExecuteEntryDeadlineCodes pins the per-row timeout vocabulary for
// the entry deadline cancel: create and quiesce cancel
// terminal_backend_timeout, wait cancels quiesce_timeout and stop
// cancels stop_timeout, each restoring the source with replay_same.
// The stop replay_same here is the cancel-before-any-attempt direction;
// a backend-REPORTED stop_timeout maps through DispositionFor instead
// (pinned by TestExecuteBackendRowErrorSets).
func TestExecuteEntryDeadlineCodes(t *testing.T) {
	past := fixturePastDeadline()
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
			engine.Now = func() time.Time { return past }
			context, params, source, admitted := tc.build(t)
			result, err := engine.ExecuteMutating(contextGo(), tc.operation, context, params, source, admitted)
			requireRefusal(t, err, tc.code, "operation deadline")
			requireLiteral(t, "after", string(result.After), tc.source)
			requireLiteral(t, "disposition", string(result.Disposition), "replay_same")
			if len(backend.performed()) != 0 {
				t.Errorf("performed = %v, want no effects on the deadline cancel", backend.performed())
			}
			if _, found, _ := engine.Store.Lookup(context.IdempotencyKey); found {
				t.Errorf("Lookup(key) found a receipt for the expired request, want none bound")
			}
		})
	}
}

func waitFixture(t *testing.T, kind ProviderProofKind) (MutationContext, Params, terminalbackend.InstanceState, terminalbackend.Admitted) {
	t.Helper()
	key := fixtureInstance + "/boundary/" + fixtureQuiescence + "/" + string(kind)
	raw := contextDoc(fixtureOperation, fixtureSessionA, fixtureInstance, fixtureBackend, fixtureImpl, fixtureProto, fixtureGen, key, fixtureDeadline, authDoc("1", nil))
	context, err := ParseMutationContext([]byte(raw))
	if err != nil {
		t.Fatalf("ParseMutationContext(wait) error = %v", err)
	}
	params := Params{QuiescenceGeneration: fixtureQuiescence, ProviderProofKind: kind, TimeoutMs: 1000}
	return context, params, mustParseState(t, "quiescing"), testAdmitted("safe_boundary_observation")
}

func stopFixture(t *testing.T) (MutationContext, Params, terminalbackend.InstanceState, terminalbackend.Admitted) {
	t.Helper()
	key := fixtureInstance + "/stop/" + fixtureDigestB
	raw := contextDoc(fixtureOperation, fixtureSessionA, fixtureInstance, fixtureBackend, fixtureImpl, fixtureProto, fixtureGen, key, fixtureDeadline, authDoc("1", nil))
	context, err := ParseMutationContext([]byte(raw))
	if err != nil {
		t.Fatalf("ParseMutationContext(stop) error = %v", err)
	}
	params := Params{SafeBoundaryEvidenceID: fixtureDigestB, GracefulTimeoutMs: 1000}
	return context, params, mustParseState(t, "quiescing"), testAdmitted("graceful_stop")
}

// TestExecutePostEffectErrorsMoveUnavailable proves an error after a
// committed effect whose result cannot be proven moves the local
// observation to unavailable with status_first and never claims
// absent: a coded refusal on the second stop effect, an uncoded
// backend failure on the first effect, malformed success evidence,
// and a refusal code outside the row set.
func TestExecutePostEffectErrorsMoveUnavailable(t *testing.T) {
	t.Run("coded refusal on second effect", func(t *testing.T) {
		backend := newMockBackend()
		backend.effectErrors[terminalbackend.EffectProcessClosed] = refuse("terminal_backend_process_failed", "test crash")
		engine := testEngine(t, backend, testLease(), fixtureGen)
		context, params, source, admitted := stopFixture(t)
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationRequestStop, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_process_failed", "backend effect refused")
		requireLiteral(t, "after", string(result.After), "unavailable")
		requireLiteral(t, "disposition", string(result.Disposition), "status_first")
		performed := backend.performed()
		if len(performed) != 2 {
			t.Errorf("performed = %v, want the first effect committed and the second attempted", performed)
		}
	})

	t.Run("uncoded failure on first effect", func(t *testing.T) {
		backend := newMockBackend()
		backend.effectErrors[terminalbackend.EffectInputClosed] = errors.New("backend transport died")
		engine := testEngine(t, backend, testLease(), fixtureGen)
		context, params, source, admitted := quiesceFixture(t)
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_process_failed", "backend effect uncertain")
		requireLiteral(t, "after", string(result.After), "unavailable")
		requireLiteral(t, "disposition", string(result.Disposition), "status_first")
	})

	t.Run("malformed success evidence", func(t *testing.T) {
		backend := newMockBackend()
		backend.evidence[terminalbackend.EffectInputClosed] = "not-a-digest"
		engine := testEngine(t, backend, testLease(), fixtureGen)
		context, params, source, admitted := quiesceFixture(t)
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_integrity_failure", "backend effect evidence")
		requireLiteral(t, "after", string(result.After), "unavailable")
		requireLiteral(t, "disposition", string(result.Disposition), "status_first")
	})

	t.Run("code outside the row set", func(t *testing.T) {
		backend := newMockBackend()
		backend.effectErrors[terminalbackend.EffectInputClosed] = refuse("stop_timeout", "test wrong row")
		engine := testEngine(t, backend, testLease(), fixtureGen)
		context, params, source, admitted := quiesceFixture(t)
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_protocol_error", "operation error vocabulary")
		requireLiteral(t, "after", string(result.After), "unavailable")
		requireLiteral(t, "disposition", string(result.Disposition), "status_first")
	})
}

// TestExecuteRechecksBeforeEachEffect proves authorization and
// generation are rechecked immediately before each side effect, not
// once at entry: a lease rotation and a generation rotation between
// the first and second stop effects are both caught at the second
// recheck, after one committed effect.
func TestExecuteRechecksBeforeEachEffect(t *testing.T) {
	t.Run("lease rotates mid-operation", func(t *testing.T) {
		backend := newMockBackend()
		engine := testEngine(t, backend, testLease(), fixtureGen)
		calls := 0
		engine.CurrentLease = func() LeaseView {
			calls++
			if calls <= 2 {
				return testLease()
			}
			return LeaseView{LeaseID: fixtureLease, Epoch: 2}
		}
		context, params, source, admitted := stopFixture(t)
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationRequestStop, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_unauthorized", "ax authorization lease epoch")
		requireLiteral(t, "after", string(result.After), "unavailable")
		requireLiteral(t, "disposition", string(result.Disposition), "new_authorization")
		if len(backend.performed()) != 1 {
			t.Errorf("performed = %v, want exactly the first effect committed", backend.performed())
		}
	})

	t.Run("generation rotates mid-operation", func(t *testing.T) {
		backend := newMockBackend()
		engine := testEngine(t, backend, testLease(), fixtureGen)
		calls := 0
		engine.CurrentGeneration = func() string {
			calls++
			if calls <= 2 {
				return fixtureGen
			}
			return "generation-two"
		}
		context, params, source, admitted := stopFixture(t)
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationRequestStop, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_stale_generation", "backend_generation stale")
		requireLiteral(t, "after", string(result.After), "unavailable")
		requireLiteral(t, "disposition", string(result.Disposition), "status_first")
		if len(backend.performed()) != 1 {
			t.Errorf("performed = %v, want exactly the first effect committed", backend.performed())
		}
	})

	t.Run("rotation before first effect restores source", func(t *testing.T) {
		backend := newMockBackend()
		engine := testEngine(t, backend, testLease(), fixtureGen)
		calls := 0
		engine.CurrentLease = func() LeaseView {
			calls++
			if calls == 1 {
				return testLease()
			}
			return LeaseView{LeaseID: fixtureLeaseB, Epoch: 1}
		}
		context, params, source, admitted := quiesceFixture(t)
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_unauthorized", "ax authorization lease identity")
		requireLiteral(t, "after", string(result.After), "active")
		requireLiteral(t, "disposition", string(result.Disposition), "new_authorization")
		if len(backend.performed()) != 0 {
			t.Errorf("performed = %v, want no effects", backend.performed())
		}
	})

	t.Run("generation rotation before first effect restores source", func(t *testing.T) {
		backend := newMockBackend()
		engine := testEngine(t, backend, testLease(), fixtureGen)
		calls := 0
		engine.CurrentGeneration = func() string {
			calls++
			if calls == 1 {
				return fixtureGen
			}
			return "generation-two"
		}
		context, params, source, admitted := quiesceFixture(t)
		result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationQuiesceInput, context, params, source, admitted)
		requireRefusal(t, err, "terminal_backend_stale_generation", "backend_generation stale")
		requireLiteral(t, "after", string(result.After), "active")
		requireLiteral(t, "disposition", string(result.Disposition), "status_first")
		if len(backend.performed()) != 0 {
			t.Errorf("performed = %v, want no effects", backend.performed())
		}
	})
}

// TestExecuteDeadlineNeverCancelsCommittedEffect proves a deadline
// breached after the first effect leaves the committed effect standing
// and moves to unavailable: the performed prefix is reported, not
// rolled back.
func TestExecuteDeadlineNeverCancelsCommittedEffect(t *testing.T) {
	backend := newMockBackend()
	engine := testEngine(t, backend, testLease(), fixtureGen)
	breached := false
	engine.Hooks = &EngineHooks{AfterEffect: func(terminalbackend.SideEffect) { breached = true }}
	engine.Now = func() time.Time {
		if breached {
			return fixturePastDeadline()
		}
		return fixtureNow()
	}
	context, params, source, admitted := stopFixture(t)
	result, err := engine.ExecuteMutating(contextGo(), terminalbackend.OperationRequestStop, context, params, source, admitted)
	requireRefusal(t, err, "stop_timeout", "operation deadline")
	requireLiteral(t, "after", string(result.After), "unavailable")
	requireLiteral(t, "disposition", string(result.Disposition), "status_first")
	performed := backend.performed()
	if len(performed) != 1 || performed[0] != terminalbackend.EffectGracefulStopRequested {
		t.Errorf("performed = %v, want [graceful_stop_requested] standing", performed)
	}
	if len(result.Effects) != 1 || result.Effects[0] != terminalbackend.EffectGracefulStopRequested {
		t.Errorf("result effects = %v, want the committed prefix", result.Effects)
	}
}
